package processor

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/waldirborbajr/sync/config"
	"github.com/waldirborbajr/sync/logger"
)

// mysqlRecord define a estrutura dos registros do MySQL
type mysqlRecord struct {
	Descricao  sql.NullString
	Quantidade sql.NullFloat64
	ValorCusto sql.NullFloat64
	ValorUsd   sql.NullFloat64
	PrcVenda   sql.NullFloat64
	Prc3x      sql.NullFloat64
	Prc6x      sql.NullFloat64
	Prc10x     sql.NullFloat64
}

// ProcessingStats para métricas de performance
type ProcessingStats struct {
	LoadTime       time.Duration
	QueryTime      time.Duration
	ProcessingTime time.Duration
	ProcedureTime  time.Duration
	TotalRows      int
}

// Operation types
type OperationType int

const (
	OpInsert OperationType = iota
	OpUpdate
	OpIgnore
)

// RowOperation represents a single database operation
type RowOperation struct {
	Type      OperationType
	IDEstoque int
	Descricao string
	QtdAtual  float64
	PrcCusto  float64
	PrcDolar  float64
	PrcVenda  float64
	Prc3x     float64
	Prc6x     float64
	Prc10x    float64
}

// ProcessRows - High-performance version using worker pool pattern
func ProcessRows(ctx context.Context, firebirdDB, mysqlDB *sql.DB, numWorkers int, cfg config.Config) (inserted, updated, ignored int, batchSize int, stats *ProcessingStats, err error) {
	log := logger.GetLogger()
	stats = &ProcessingStats{}

	// Load MySQL records into memory
	startLoad := time.Now()
	existingRecords, err := loadMySQLRecords(mysqlDB)
	if err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("error loading MySQL records: %w", err)
	}
	stats.LoadTime = time.Since(startLoad)
	log.Info().Int("records", len(existingRecords)).Msg("MySQL records loaded")

	// Query Firebird
	query := `
        SELECT 
            e.ID_ESTOQUE, 
            e.DESCRICAO, 
            p.QTD_ATUAL, 
            e.PRC_CUSTO, 
            i.VALOR AS PRC_DOLAR
        FROM TB_ESTOQUE e
        JOIN TB_EST_PRODUTO p 
            ON e.ID_ESTOQUE = p.ID_IDENTIFICADOR
        LEFT JOIN TB_EST_INDEXADOR i 
            ON i.ID_ESTOQUE = e.ID_ESTOQUE
        WHERE e.STATUS = 'A'
    `

	startQuery := time.Now()
	rows, err := firebirdDB.QueryContext(ctx, query)
	if err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("error querying Firebird: %w", err)
	}
	defer rows.Close()
	stats.QueryTime = time.Since(startQuery)
	log.Info().Msg("Firebird query executed")

	// Calculate batch size
	batchSize = 500 // Optimal batch size for bulk operations

	// Channel for work distribution
	workChan := make(chan RowOperation, batchSize*2) // Buffered channel

	// Atomic counters for thread-safe counting
	var insertedCount, updatedCount, ignoredCount atomic.Int64

	// Worker pool
	var wg sync.WaitGroup
	processingStart := time.Now()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, i, workChan, mysqlDB, &insertedCount, &updatedCount, &ignoredCount, &wg, cfg)
	}

	// Feed workers from Firebird query
	rowCount := 0
	for rows.Next() {
		var idEstoque int
		var descricao string
		var qtdAtual float64
		var prcCusto, prcDolar sql.NullFloat64

		if err := rows.Scan(&idEstoque, &descricao, &qtdAtual, &prcCusto, &prcDolar); err != nil {
			log.Error().Err(err).Int("id_estoque", idEstoque).Msg("Error scanning Firebird row")
			continue
		}

		// Process row
		op := processRowOptimized(existingRecords, idEstoque, descricao, qtdAtual, prcCusto, prcDolar, cfg)

		select {
		case workChan <- op:
			rowCount++
		case <-ctx.Done():
			close(workChan)
			return 0, 0, 0, 0, nil, ctx.Err()
		}
	}

	// Close work channel and wait for workers
	close(workChan)
	wg.Wait()

	if err = rows.Err(); err != nil {
		return 0, 0, 0, 0, nil, err
	}

	stats.ProcessingTime = time.Since(processingStart)
	stats.TotalRows = rowCount

	// Run post-processing procedures
	if err := runPostProcessing(mysqlDB, stats); err != nil {
		return 0, 0, 0, 0, nil, err
	}

	return int(insertedCount.Load()), int(updatedCount.Load()), int(ignoredCount.Load()), batchSize, stats, nil
}

// worker processes operations from the work channel in batches
func worker(ctx context.Context, id int, workChan <-chan RowOperation, db *sql.DB, insertedCount, updatedCount, ignoredCount *atomic.Int64, wg *sync.WaitGroup, cfg config.Config) {
	defer wg.Done()
	log := logger.GetLogger()

	const batchSize = 500
	insertBatch := make([]RowOperation, 0, batchSize)
	updateBatch := make([]RowOperation, 0, batchSize)

	flushBatches := func() error {
		if len(insertBatch) > 0 {
			if err := executeBulkInsert(db, insertBatch); err != nil {
				log.Error().Err(err).Int("worker", id).Msg("Error executing bulk insert")
				return err
			}
			insertedCount.Add(int64(len(insertBatch)))
			insertBatch = insertBatch[:0]
		}

		if len(updateBatch) > 0 {
			if err := executeBulkUpdate(db, updateBatch); err != nil {
				log.Error().Err(err).Int("worker", id).Msg("Error executing bulk update")
				return err
			}
			updatedCount.Add(int64(len(updateBatch)))
			updateBatch = updateBatch[:0]
		}
		return nil
	}

	// Process work items
	for op := range workChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		switch op.Type {
		case OpInsert:
			insertBatch = append(insertBatch, op)
			if len(insertBatch) >= batchSize {
				if err := flushBatches(); err != nil {
					log.Error().Err(err).Msg("Error flushing insert batch")
				}
			}

		case OpUpdate:
			updateBatch = append(updateBatch, op)
			if len(updateBatch) >= batchSize {
				if err := flushBatches(); err != nil {
					log.Error().Err(err).Msg("Error flushing update batch")
				}
			}

		case OpIgnore:
			ignoredCount.Add(1)
		}
	}

	// Flush remaining batches
	if err := flushBatches(); err != nil {
		log.Error().Err(err).Msg("Error flushing final batches")
	}
}

// processRowOptimized determines what operation to perform on a row
func processRowOptimized(existingRecords map[int]mysqlRecord, idEstoque int, descricao string, qtdAtual float64, prcCusto, prcDolar sql.NullFloat64, cfg config.Config) RowOperation {
	// Calculate prices
	prcVenda, prc3x, prc6x, prc10x := calculatePrices(prcCusto, cfg)
	custo := roundFloat(prcCusto)
	dolar := roundFloat(prcDolar)

	rec, exists := existingRecords[idEstoque]

	// New record
	if !exists {
		return RowOperation{
			Type:      OpInsert,
			IDEstoque: idEstoque,
			Descricao: descricao,
			QtdAtual:  qtdAtual,
			PrcCusto:  custo,
			PrcDolar:  dolar,
			PrcVenda:  prcVenda,
			Prc3x:     prc3x,
			Prc6x:     prc6x,
			Prc10x:    prc10x,
		}
	}

	// Check if update needed
	existingCusto := roundFloat(rec.ValorCusto)
	existingDolar := roundFloat(rec.ValorUsd)
	existingPrcVenda := roundFloat(rec.PrcVenda)
	existingPrc3x := roundFloat(rec.Prc3x)
	existingPrc6x := roundFloat(rec.Prc6x)
	existingPrc10x := roundFloat(rec.Prc10x)

	if rec.Descricao.Valid && rec.Quantidade.Valid &&
		rec.Descricao.String == descricao &&
		rec.Quantidade.Float64 == qtdAtual &&
		existingCusto == custo &&
		existingDolar == dolar &&
		existingPrcVenda == prcVenda &&
		existingPrc3x == prc3x &&
		existingPrc6x == prc6x &&
		existingPrc10x == prc10x {
		return RowOperation{Type: OpIgnore}
	}

	// Update needed
	return RowOperation{
		Type:      OpUpdate,
		IDEstoque: idEstoque,
		Descricao: descricao,
		QtdAtual:  qtdAtual,
		PrcCusto:  custo,
		PrcDolar:  dolar,
		PrcVenda:  prcVenda,
		Prc3x:     prc3x,
		Prc6x:     prc6x,
		Prc10x:    prc10x,
	}
}

// executeBulkInsert performs a true bulk INSERT with multi-value syntax
func executeBulkInsert(db *sql.DB, ops []RowOperation) error {
	if len(ops) == 0 {
		return nil
	}

	log := logger.GetLogger()

	// Build multi-value INSERT statement
	var sb strings.Builder
	sb.WriteString("INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, PRC_VENDA, PRC_3X, PRC_6X, PRC_10X) VALUES ")

	values := make([]interface{}, 0, len(ops)*9)
	for i, op := range ops {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		values = append(values, op.IDEstoque, op.Descricao, op.QtdAtual, op.PrcCusto, op.PrcDolar, op.PrcVenda, op.Prc3x, op.Prc6x, op.Prc10x)
	}

	_, err := db.Exec(sb.String(), values...)
	if err != nil {
		log.Error().Err(err).Int("count", len(ops)).Msg("Bulk insert failed")
		return fmt.Errorf("bulk insert failed: %w", err)
	}

	log.Debug().Int("count", len(ops)).Msg("Bulk insert successful")
	return nil
}

// executeBulkUpdate performs batch updates (MySQL doesn't support multi-row UPDATE well, so we use transaction)
func executeBulkUpdate(db *sql.DB, ops []RowOperation) error {
	if len(ops) == 0 {
		return nil
	}

	log := logger.GetLogger()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		UPDATE TB_ESTOQUE 
		SET DESCRICAO = ?, QTD_ATUAL = ?, PRC_CUSTO = ?, PRC_DOLAR = ?, 
			PRC_VENDA = ?, PRC_3X = ?, PRC_6X = ?, PRC_10X = ?
		WHERE ID_ESTOQUE = ?
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error preparing update statement: %w", err)
	}
	defer stmt.Close()

	for _, op := range ops {
		_, err := stmt.Exec(op.Descricao, op.QtdAtual, op.PrcCusto, op.PrcDolar, op.PrcVenda, op.Prc3x, op.Prc6x, op.Prc10x, op.IDEstoque)
		if err != nil {
			tx.Rollback()
			log.Error().Err(err).Int("id_estoque", op.IDEstoque).Msg("Update failed")
			return fmt.Errorf("update failed for ID %d: %w", op.IDEstoque, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Int("count", len(ops)).Msg("Bulk update commit failed")
		return fmt.Errorf("bulk update commit failed: %w", err)
	}

	log.Debug().Int("count", len(ops)).Msg("Bulk update successful")
	return nil
}

// loadMySQLRecords loads existing MySQL records into a map
func loadMySQLRecords(db *sql.DB) (map[int]mysqlRecord, error) {
	log := logger.GetLogger()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM TB_ESTOQUE WHERE ID_ESTOQUE IS NOT NULL").Scan(&count)
	if err != nil {
		return nil, err
	}

	records := make(map[int]mysqlRecord, count)

	rows, err := db.Query("SELECT ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, PRC_VENDA, PRC_3X, PRC_6X, PRC_10X FROM TB_ESTOQUE WHERE ID_ESTOQUE IS NOT NULL")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing MySQL rows")
		}
	}()

	for rows.Next() {
		var idClipp int
		var rec mysqlRecord
		if err := rows.Scan(&idClipp, &rec.Descricao, &rec.Quantidade, &rec.ValorCusto, &rec.ValorUsd, &rec.PrcVenda, &rec.Prc3x, &rec.Prc6x, &rec.Prc10x); err != nil {
			return nil, err
		}
		records[idClipp] = rec
	}
	return records, rows.Err()
}

// loadMySQLRecordsForID
func loadMySQLRecordsForID(idEstoque int, existingRecords map[int]mysqlRecord, rec *mysqlRecord) error {
	if r, exists := existingRecords[idEstoque]; exists {
		*rec = r
		return nil
	}
	return fmt.Errorf("record not found in existingRecords for ID_ESTOQUE=%d", idEstoque)
}

// calculatePrices calcula os novos preços baseado nas regras
func calculatePrices(prcCusto sql.NullFloat64, cfg config.Config) (prcVenda, prc3x, prc6x, prc10x float64) {
	if !prcCusto.Valid || prcCusto.Float64 == 0 {
		return 0, 0, 0, 0
	}

	custo := prcCusto.Float64

	// PRC_VENDA = PRC_CUSTO * (1 + LUCRO/100)
	prcVenda = custo * (1 + cfg.Lucro/100)
	prcVenda = math.Round(prcVenda*100) / 100

	// PRC_3X = (PRC_CUSTO * (1 + LUCRO/100) * (1 + PARC3X/100)) / 3
	prc3x = (custo * (1 + cfg.Lucro/100) * (1 + cfg.Parc3x/100)) / 3
	prc3x = math.Round(prc3x*100) / 100

	// PRC_6X = (PRC_CUSTO * (1 + LUCRO/100) * (1 + PARC6X/100)) / 6
	prc6x = (custo * (1 + cfg.Lucro/100) * (1 + cfg.Parc6x/100)) / 6
	prc6x = math.Round(prc6x*100) / 100

	// PRC_10X = (PRC_CUSTO * (1 + LUCRO/100) * (1 + PARC10X/100)) / 10
	prc10x = (custo * (1 + cfg.Lucro/100) * (1 + cfg.Parc10x/100)) / 10
	prc10x = math.Round(prc10x*100) / 100

	return prcVenda, prc3x, prc6x, prc10x
}

// processRowForBatch otimizado com cálculo dos novos preços
func processRowForBatch(existingRecords map[int]mysqlRecord, idEstoque int, descricao string, qtdAtual float64, prcCusto, prcDolar sql.NullFloat64, mu *sync.Mutex, insertedCount, updatedCount, ignoredCount *int, cfg config.Config) (string, []interface{}) {
	log := logger.GetLogger()

	// Log source (Firebird) row when DEBUG_MODE=true
	if cfg.DebugMode {
		log.Debug().
			Str("source", "Firebird").
			Int("id_estoque", idEstoque).
			Str("descricao", descricao).
			Float64("qtd_atual", qtdAtual).
			Float64("prc_custo", roundFloat(prcCusto)).
			Float64("prc_dolar", roundFloat(prcDolar)).
			Msg("Processing Firebird row")
	}

	rec, exists := existingRecords[idEstoque]

	// Log target (MySQL) row when DEBUG_MODE=true
	if cfg.DebugMode {
		if exists {
			log.Debug().
				Str("target", "MySQL").
				Int("id_estoque", idEstoque).
				Str("descricao", nullString(rec.Descricao)).
				Float64("qtd_atual", nullFloat(rec.Quantidade)).
				Float64("prc_custo", roundFloat(rec.ValorCusto)).
				Float64("prc_dolar", roundFloat(rec.ValorUsd)).
				Float64("prc_venda", roundFloat(rec.PrcVenda)).
				Float64("prc_3x", roundFloat(rec.Prc3x)).
				Float64("prc_6x", roundFloat(rec.Prc6x)).
				Float64("prc_10x", roundFloat(rec.Prc10x)).
				Msg("Comparing against MySQL record")
		} else {
			log.Debug().
				Str("target", "MySQL").
				Int("id_estoque", idEstoque).
				Msg("No matching MySQL record found")
		}
	}

	// Calcular os novos preços
	prcVenda, prc3x, prc6x, prc10x := calculatePrices(prcCusto, cfg)

	custo := roundFloat(prcCusto)
	dolar := roundFloat(prcDolar)

	if !exists {
		mu.Lock()
		*insertedCount++
		mu.Unlock()

		log.Debug().
			Int("id_estoque", idEstoque).
			Msg("Inserting new record")

		return "insert", []any{idEstoque, descricao, qtdAtual, custo, dolar, prcVenda, prc3x, prc6x, prc10x}
	}

	existingCusto := roundFloat(rec.ValorCusto)
	existingDolar := roundFloat(rec.ValorUsd)
	existingPrcVenda := roundFloat(rec.PrcVenda)
	existingPrc3x := roundFloat(rec.Prc3x)
	existingPrc6x := roundFloat(rec.Prc6x)
	existingPrc10x := roundFloat(rec.Prc10x)

	if rec.Descricao.Valid && rec.Quantidade.Valid &&
		rec.Descricao.String == descricao &&
		rec.Quantidade.Float64 == qtdAtual &&
		existingCusto == custo &&
		existingDolar == dolar &&
		existingPrcVenda == prcVenda &&
		existingPrc3x == prc3x &&
		existingPrc6x == prc6x &&
		existingPrc10x == prc10x {
		mu.Lock()
		*ignoredCount++
		mu.Unlock()
		return "", nil
	}

	log.Info().
		Int("id_estoque", idEstoque).
		Str("descricao", descricao).
		Float64("qtd_atual", qtdAtual).
		Float64("prc_custo", custo).
		Float64("prc_dolar", dolar).
		Float64("prc_venda", prcVenda).
		Float64("prc_3x", prc3x).
		Float64("prc_6x", prc6x).
		Float64("prc_10x", prc10x).
		Msg("Updating record")

	mu.Lock()
	*updatedCount++
	mu.Unlock()
	return "update", []any{descricao, qtdAtual, custo, dolar, prcVenda, prc3x, prc6x, prc10x, idEstoque}
}

func roundFloat(value sql.NullFloat64) float64 {
	if value.Valid {
		return math.Round(value.Float64*100) / 100
	}
	return 0.0
}

// executeBatch otimizado com bulk operations
func executeBatch(tx *sql.Tx, insertStmt, updateStmt *sql.Stmt, batchInsert, batchUpdate []any) error {
	log := logger.GetLogger()

	if len(batchInsert) > 0 {
		log.Debug().Int("count", len(batchInsert)/9).Msg("Processing insert operations")
		stmt := tx.Stmt(insertStmt)
		for i := 0; i < len(batchInsert); i += 9 {
			log.Debug().Interface("id_estoque", batchInsert[i]).Msg("Executing insert")
			if _, err := stmt.Exec(batchInsert[i], batchInsert[i+1], batchInsert[i+2], batchInsert[i+3],
				batchInsert[i+4], batchInsert[i+5], batchInsert[i+6], batchInsert[i+7], batchInsert[i+8]); err != nil {
				log.Error().Err(err).Interface("id_estoque", batchInsert[i]).Msg("Error executing insert")
				return fmt.Errorf("error executing batch insert: %w", err)
			}
		}
	}

	if len(batchUpdate) > 0 {
		log.Debug().Int("count", len(batchUpdate)/9).Msg("Processing update operations")
		stmt := tx.Stmt(updateStmt)
		for i := 0; i < len(batchUpdate); i += 9 {
			idEstoque := batchUpdate[i+8]
			log.Debug().Interface("id_estoque", idEstoque).Msg("Executing update")
			if _, err := stmt.Exec(batchUpdate[i], batchUpdate[i+1], batchUpdate[i+2], batchUpdate[i+3],
				batchUpdate[i+4], batchUpdate[i+5], batchUpdate[i+6], batchUpdate[i+7], batchUpdate[i+8]); err != nil {
				log.Error().Err(err).Interface("id_estoque", idEstoque).Msg("Error executing update")
				return fmt.Errorf("error executing batch update: %w", err)
			}
		}
	}
	return nil
}

// commitTx commits the transaction and logs errors
func commitTx(tx *sql.Tx) error {
	log := logger.GetLogger()
	if tx == nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Error committing transaction")
		return fmt.Errorf("error committing transaction: %w", err)
	}
	return nil
}

// verifyUpdatedRowIfNeeded logs a verification of a specific test row when updates occurred
func verifyUpdatedRowIfNeeded(db *sql.DB, updatedCount int) {
	if updatedCount <= 0 {
		return
	}
	log := logger.GetLogger()
	var mysqlDescricao string
	var mysqlQtdAtual float64
	var mysqlPrcCusto, mysqlPrcDolar, mysqlPrcVenda, mysqlPrc3x, mysqlPrc6x, mysqlPrc10x sql.NullFloat64
	err := db.QueryRow("SELECT DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, PRC_VENDA, PRC_3X, PRC_6X, PRC_10X FROM TB_ESTOQUE WHERE ID_ESTOQUE = 17973").Scan(
		&mysqlDescricao, &mysqlQtdAtual, &mysqlPrcCusto, &mysqlPrcDolar, &mysqlPrcVenda, &mysqlPrc3x, &mysqlPrc6x, &mysqlPrc10x)
	if err != nil {
		log.Error().Err(err).Msg("Error verifying updated row in MySQL")
		return
	}
	log.Info().
		Int("id_estoque", 17973).
		Str("descricao", mysqlDescricao).
		Float64("qtd_atual", mysqlQtdAtual).
		Float64("prc_custo", mysqlPrcCusto.Float64).
		Float64("prc_dolar", mysqlPrcDolar.Float64).
		Float64("prc_venda", mysqlPrcVenda.Float64).
		Float64("prc_3x", mysqlPrc3x.Float64).
		Float64("prc_6x", mysqlPrc6x.Float64).
		Float64("prc_10x", mysqlPrc10x.Float64).
		Msg("Verified MySQL row after update")
}

// runPostProcessing executes DB procedures and updates stats
func runPostProcessing(db *sql.DB, stats *ProcessingStats) error {
	log := logger.GetLogger()
	startProc := time.Now()
	_, err := db.Exec("CALL UpdateQtdVirtual()")
	if err != nil {
		log.Error().Err(err).Msg("Error calling UpdateQtdVirtual procedure")
		return fmt.Errorf("error calling UpdateQtdVirtual procedure: %w", err)
	}
	stats.ProcedureTime += time.Since(startProc)
	log.Debug().Msg("UpdateQtdVirtual procedure executed successfully")

	startProc = time.Now()
	_, err = db.Exec("CALL SP_ATUALIZAR_PART_NUMBER()")
	if err != nil {
		log.Error().Err(err).Msg("Error calling SP_ATUALIZAR_PART_NUMBER procedure")
		return fmt.Errorf("error calling SP_ATUALIZAR_PART_NUMBER procedure: %w", err)
	}
	stats.ProcedureTime += time.Since(startProc)
	log.Debug().Msg("SP_ATUALIZAR_PART_NUMBER procedure executed successfully")
	return nil
}

// Helper functions to handle null values in logging
func nullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullFloat(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0.0
}
