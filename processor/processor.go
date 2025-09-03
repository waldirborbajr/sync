package processor

import (
	"database/sql"
	"fmt"
	"math"
	"sync"
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

// batchPoolType defines the structure for the sync.Pool wrapper
type batchPoolType struct {
	pool sync.Pool
}

// ProcessRows com foco em ID_ESTOQUE=17973
func ProcessRows(firebirdDB, mysqlDB *sql.DB, updateStmt, insertStmt *sql.Stmt, semaphoreSize int, maxAllowedPacket int, insertedCount, updatedCount, ignoredCount *int, batchSize *int, stats *ProcessingStats, cfg config.Config) error {
	log := logger.GetLogger()

	// Pré-carregar dados do MySQL em um mapa
	startLoad := time.Now()
	existingRecords, err := loadMySQLRecords(mysqlDB)
	if err != nil {
		return fmt.Errorf("error loading MySQL records: %w", err)
	}
	stats.LoadTime = time.Since(startLoad)

	log.Debug().Msg("MySQL records loaded successfully")

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
	rows, err := firebirdDB.Query(query)
	if err != nil {
		return fmt.Errorf("error querying Firebird: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing Firebird rows")
		}
	}()
	stats.QueryTime = time.Since(startQuery)

	log.Debug().Msg("Firebird query executed successfully")

	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, semaphoreSize)

	// Calcular batch size (mantido para consistência, mas só um registro será processado)
	estimatedRowSize := 200
	*batchSize = maxAllowedPacket / estimatedRowSize
	if *batchSize > 5000 {
		*batchSize = 5000
	}
	if *batchSize < 100 {
		*batchSize = 100
	}

	// Usar sync.Pool para reutilizar slices (agora com ponteiros)
	batchPool := &batchPoolType{
		pool: sync.Pool{
			New: func() interface{} {
				slice := make([]interface{}, 0, *batchSize*9) // 9 campos agora
				return &slice
			},
		},
	}

	batchInsertPtr := batchPool.pool.Get().(*[]interface{})
	batchUpdatePtr := batchPool.pool.Get().(*[]interface{})
	batchInsert := *batchInsertPtr
	batchUpdate := *batchUpdatePtr

	defer func() {
		// Reset slices antes de devolver ao pool
		*batchInsertPtr = (*batchInsertPtr)[:0]
		*batchUpdatePtr = (*batchUpdatePtr)[:0]
		batchPool.pool.Put(batchInsertPtr)
		batchPool.pool.Put(batchUpdatePtr)
	}()

	tx, err := mysqlDB.Begin()
	if err != nil {
		return fmt.Errorf("error starting MySQL transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("Error rolling back transaction")
			}
		}
	}()

	rowCount := 0
	processingStart := time.Now()

	// Buffer para o único registro
	const rowBufferSize = 1
	rowBuffer := make([]struct {
		idEstoque int
		descricao string
		qtdAtual  float64
		prcCusto  sql.NullFloat64
		prcDolar  sql.NullFloat64
	}, rowBufferSize)
	bufferIndex := 0

	for rows.Next() {
		var idEstoque int
		var descricao string
		var qtdAtual float64
		var prcCusto sql.NullFloat64
		var prcDolar sql.NullFloat64

		if err := rows.Scan(&idEstoque, &descricao, &qtdAtual, &prcCusto, &prcDolar); err != nil {
			log.Error().Err(err).Int("id_estoque", idEstoque).Msg("Error scanning Firebird row")
			continue
		}

		// Adicionar ao buffer
		rowBuffer[bufferIndex] = struct {
			idEstoque int
			descricao string
			qtdAtual  float64
			prcCusto  sql.NullFloat64
			prcDolar  sql.NullFloat64
		}{idEstoque, descricao, qtdAtual, prcCusto, prcDolar}
		bufferIndex++

		// Processar buffer imediatamente (apenas um registro)
		if bufferIndex == rowBufferSize {
			processRowBuffer(&rowBuffer, bufferIndex, existingRecords, &batchInsert, &batchUpdate, &mu, insertedCount, updatedCount, ignoredCount, &wg, semaphore, cfg)
			bufferIndex = 0
			rowCount++
		}
	}

	// Processar batch final
	if len(batchInsert) > 0 || len(batchUpdate) > 0 {
		if err := processBatch(tx, insertStmt, updateStmt, &batchInsert, &batchUpdate, batchPool); err != nil {
			log.Error().Err(err).Msg("Error processing final batch")
		}
	}

	if err = rows.Err(); err != nil {
		return err
	}

	stats.ProcessingTime = time.Since(processingStart)

	// Commit final com logging detalhado
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Error committing transaction")
		return fmt.Errorf("error committing transaction: %w", err)
	}
	tx = nil

	// Verificação pós-update
	if *updatedCount > 0 {
		var mysqlDescricao string
		var mysqlQtdAtual float64
		var mysqlPrcCusto, mysqlPrcDolar, mysqlPrcVenda, mysqlPrc3x, mysqlPrc6x, mysqlPrc10x sql.NullFloat64
		err = mysqlDB.QueryRow("SELECT DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, PRC_VENDA, PRC_3X, PRC_6X, PRC_10X FROM TB_ESTOQUE WHERE ID_ESTOQUE = 17973").Scan(
			&mysqlDescricao, &mysqlQtdAtual, &mysqlPrcCusto, &mysqlPrcDolar, &mysqlPrcVenda, &mysqlPrc3x, &mysqlPrc6x, &mysqlPrc10x)
		if err != nil {
			log.Error().Err(err).Msg("Error verifying updated row in MySQL")
		} else {
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
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Call stored procedure to update virtual stock
	startProc := time.Now()
	_, err = mysqlDB.Exec("CALL UpdateQtdVirtual()")
	if err != nil {
		log.Error().Err(err).Msg("Error calling UpdateQtdVirtual procedure")
		return fmt.Errorf("error calling UpdateQtdVirtual procedure: %w", err)
	}
	stats.ProcedureTime = time.Since(startProc)

	log.Debug().Msg("UpdateQtdVirtual procedure executed successfully")

	stats.TotalRows = rowCount
	return nil
}

func processRowBuffer(rowBuffer *[]struct {
	idEstoque int
	descricao string
	qtdAtual  float64
	prcCusto  sql.NullFloat64
	prcDolar  sql.NullFloat64
}, count int, existingRecords map[int]mysqlRecord, batchInsert, batchUpdate *[]interface{}, mu *sync.Mutex, insertedCount, updatedCount, ignoredCount *int, wg *sync.WaitGroup, semaphore chan struct{}, cfg config.Config) {
	log := logger.GetLogger()

	for i := 0; i < count; i++ {
		row := (*rowBuffer)[i]
		semaphore <- struct{}{}
		wg.Add(1)

		go func(idEstoque int, descricao string, qtdAtual float64, prcCusto, prcDolar sql.NullFloat64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// Recarregar o registro do MySQL para garantir dados atuais
			var rec mysqlRecord
			err := loadMySQLRecordsForID(idEstoque, existingRecords, &rec)
			if err != nil {
				log.Error().Err(err).Int("id_estoque", idEstoque).Msg("Error reloading MySQL record")
				return
			}

			action, params := processRowForBatch(map[int]mysqlRecord{idEstoque: rec}, idEstoque, descricao, qtdAtual, prcCusto, prcDolar, mu, insertedCount, updatedCount, ignoredCount, cfg)
			mu.Lock()
			switch action {
			case "insert":
				*batchInsert = append(*batchInsert, params...)
			case "update":
				*batchUpdate = append(*batchUpdate, params...)
			}
			mu.Unlock()
		}(row.idEstoque, row.descricao, row.qtdAtual, row.prcCusto, row.prcDolar)
	}
}

func processBatch(tx *sql.Tx, insertStmt, updateStmt *sql.Stmt, batchInsert, batchUpdate *[]interface{}, _ *batchPoolType) error {
	log := logger.GetLogger()

	if len(*batchUpdate) > 0 {
		log.Debug().Int("count", len(*batchUpdate)/9).Msg("Processing update operations")
	}
	if err := executeBatch(tx, insertStmt, updateStmt, *batchInsert, *batchUpdate); err != nil {
		log.Error().Err(err).Msg("Error in executeBatch")
		return err
	}

	// Reset batches usando pool
	*batchInsert = (*batchInsert)[:0]
	*batchUpdate = (*batchUpdate)[:0]

	return nil
}

// loadMySQLRecords otimizado
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

	rec, exists := existingRecords[idEstoque]

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
