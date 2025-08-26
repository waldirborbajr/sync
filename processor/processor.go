package processor

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sync"
)

// ProcessRows com pré-carregamento de dados do MySQL
func ProcessRows(firebirdDB, mysqlDB *sql.DB, updateStmt, insertStmt *sql.Stmt, semaphoreSize int, insertedCount, updatedCount, ignoredCount *int) error {
	// Pré-carregar dados do MySQL em um mapa
	existingRecords, err := loadMySQLRecords(mysqlDB)
	if err != nil {
		return fmt.Errorf("error loading MySQL records: %w", err)
	}

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
	rows, err := firebirdDB.Query(query)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing Firebird rows: %v", err)
		}
	}()

	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, semaphoreSize)
	batchSize := 1000
	var batchInsert []interface{}
	var batchUpdate []interface{}
	tx, err := mysqlDB.Begin()
	if err != nil {
		return fmt.Errorf("error starting MySQL transaction: %w", err)
	}
	rowCount := 0

	for rows.Next() {
		var idEstoque int
		var descricao string
		var qtdAtual float64
		var prcCusto sql.NullFloat64
		var prcDolar sql.NullFloat64
		if err := rows.Scan(&idEstoque, &descricao, &qtdAtual, &prcCusto, &prcDolar); err != nil {
			log.Printf("error scanning Firebird row: %v", err)
			continue
		}

		semaphore <- struct{}{}
		wg.Add(1)

		go func(idEstoque int, descricao string, qtdAtual float64, prcCusto, prcDolar sql.NullFloat64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			action, params := processRowForBatch(existingRecords, idEstoque, descricao, qtdAtual, prcCusto, prcDolar, &mu, insertedCount, updatedCount, ignoredCount)
			mu.Lock()
			switch action {
			case "insert":
				batchInsert = append(batchInsert, params...)
			case "update":
				batchUpdate = append(batchUpdate, params...)
			}
			rowCount++
			if rowCount%batchSize == 0 {
				if err := executeBatch(tx, insertStmt, updateStmt, batchInsert, batchUpdate); err != nil {
					log.Printf("error executing batch: %v", err)
				}
				batchInsert = nil
				batchUpdate = nil
				if err := tx.Commit(); err != nil {
					log.Printf("error committing transaction: %v", err)
				}
				tx, err = mysqlDB.Begin()
				if err != nil {
					log.Printf("error starting new transaction: %v", err)
				}
			}
			mu.Unlock()
		}(idEstoque, descricao, qtdAtual, prcCusto, prcDolar)
	}

	if err = rows.Err(); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("error rolling back transaction: %v", rollbackErr)
		}
		return err
	}

	if len(batchInsert) > 0 || len(batchUpdate) > 0 {
		if err := executeBatch(tx, insertStmt, updateStmt, batchInsert, batchUpdate); err != nil {
			log.Printf("error executing final batch: %v", err)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("error rolling back transaction: %v", rollbackErr)
			}
			return err
		}
		if err := tx.Commit(); err != nil {
			log.Printf("error committing final transaction: %v", err)
			return err
		}
	} else {
		if err := tx.Commit(); err != nil {
			log.Printf("error committing transaction: %v", err)
			return err
		}
	}

	// Call stored procedure to update virtual stock
	// log.Println("Updating virtual stock")
	_, err = mysqlDB.Exec("CALL UpdateQtdVirtual()")
	if err != nil {
		log.Printf("error calling UpdateQtdVirtual procedure: %v", err)
		return fmt.Errorf("error calling UpdateQtdVirtual procedure: %w", err)
	}

	wg.Wait()
	return nil
}

// loadMySQLRecords carrega todos os registros existentes do MySQL em um mapa
type mysqlRecord struct {
	Descricao  sql.NullString
	Quantidade sql.NullFloat64
	ValorCusto sql.NullFloat64
	ValorUsd   sql.NullFloat64
}

func loadMySQLRecords(db *sql.DB) (map[int]mysqlRecord, error) {
	records := make(map[int]mysqlRecord)
	rows, err := db.Query("SELECT ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR FROM TB_ESTOQUE WHERE ID_ESTOQUE IS NOT NULL")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("error closing MySQL rows: %v", err)
		}
	}()

	for rows.Next() {
		var idClipp int
		var rec mysqlRecord
		if err := rows.Scan(&idClipp, &rec.Descricao, &rec.Quantidade, &rec.ValorCusto, &rec.ValorUsd); err != nil {
			return nil, err
		}
		records[idClipp] = rec
	}
	return records, rows.Err()
}

// processRowForBatch ajustado para usar o mapa
func processRowForBatch(existingRecords map[int]mysqlRecord, idEstoque int, descricao string, qtdAtual float64, prcCusto, prcDolar sql.NullFloat64, mu *sync.Mutex, insertedCount, updatedCount, ignoredCount *int) (string, []interface{}) {
	rec, exists := existingRecords[idEstoque]
	if !exists {
		custo := 0.0
		if prcCusto.Valid {
			custo = math.Round(prcCusto.Float64*100) / 100
		}
		dolar := 0.0
		if prcDolar.Valid {
			dolar = math.Round(prcDolar.Float64*100) / 100
		}
		mu.Lock()
		*insertedCount++
		mu.Unlock()
		return "insert", []any{idEstoque, descricao, qtdAtual, custo, dolar}
	}

	custo := 0.0
	if prcCusto.Valid {
		custo = math.Round(prcCusto.Float64*100) / 100
	}
	dolar := 0.0
	if prcDolar.Valid {
		dolar = math.Round(prcDolar.Float64*100) / 100
	}
	existingCusto := 0.0
	if rec.ValorCusto.Valid {
		existingCusto = math.Round(rec.ValorCusto.Float64*100) / 100
	}
	existingDolar := 0.0
	if rec.ValorUsd.Valid {
		existingDolar = math.Round(rec.ValorUsd.Float64*100) / 100
	}

	if rec.Descricao.Valid && rec.Quantidade.Valid &&
		rec.Descricao.String == descricao &&
		rec.Quantidade.Float64 == qtdAtual &&
		existingCusto == custo &&
		existingDolar == dolar {
		mu.Lock()
		*ignoredCount++
		mu.Unlock()
		return "", nil
	}

	mu.Lock()
	*updatedCount++
	mu.Unlock()
	return "update", []any{descricao, qtdAtual, custo, dolar, idEstoque}
}

// executeBatch executa as operações de INSERT e UPDATE acumuladas
func executeBatch(_ *sql.Tx, insertStmt, updateStmt *sql.Stmt, batchInsert, batchUpdate []any) error {
	if len(batchInsert) > 0 {
		for i := 0; i < len(batchInsert); i += 5 {
			params := batchInsert[i : i+5]
			if _, err := insertStmt.Exec(params...); err != nil {
				return fmt.Errorf("error executing batch insert: %w", err)
			}
		}
	}
	if len(batchUpdate) > 0 {
		for i := 0; i < len(batchUpdate); i += 5 {
			params := batchUpdate[i : i+5]
			if _, err := updateStmt.Exec(params...); err != nil {
				return fmt.Errorf("error executing batch update: %w", err)
			}
		}
	}
	return nil
}
