package processor

import (
	"database/sql"
	"log"
	"sync"
)

// ProcessRows queries Firebird and processes rows, updating or inserting into MySQL
func ProcessRows(firebirdDB, mysqlDB *sql.DB, updateStmt, insertStmt *sql.Stmt, semaphoreSize int, insertedCount, updatedCount, ignoredCount *int) error {
	query := `
		SELECT 
			e.ID_ESTOQUE, 
			e.DESCRICAO, 
			p.QTD_ATUAL, 
			e.PRC_CUSTO, 
			i.VALOR AS prc_dolar
		FROM TB_ESTOQUE e
		JOIN TB_EST_PRODUTO p 
			ON e.ID_ESTOQUE = p.ID_IDENTIFICADOR
		LEFT JOIN TB_EST_INDEXADOR i 
			ON i.ID_ESTOQUE = e.ID_ESTOQUE
		WHERE e.status = 'A'
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

	for rows.Next() {
		var idEstoque int
		var descricao string
		var qtdAtual float64
		if err := rows.Scan(&idEstoque, &descricao, &qtdAtual); err != nil {
			log.Printf("error scanning Firebird row: %v", err)
			continue
		}

		semaphore <- struct{}{}
		wg.Add(1)

		go processRow(mysqlDB, updateStmt, insertStmt, idEstoque, descricao, qtdAtual, &mu, &wg, semaphore, insertedCount, updatedCount, ignoredCount)
	}

	if err = rows.Err(); err != nil {
		return err
	}

	wg.Wait()
	return nil
}

// processRow processes a single row, updating or inserting into MySQL
func processRow(mysqlDB *sql.DB, updateStmt, insertStmt *sql.Stmt, idEstoque int, descricao string, qtdAtual float64, mu *sync.Mutex, wg *sync.WaitGroup, semaphore chan struct{}, insertedCount, updatedCount, ignoredCount *int) {
	defer wg.Done()
	defer func() { <-semaphore }()

	var existingDescricao sql.NullString
	var existingQuantidade sql.NullFloat64
	err := mysqlDB.QueryRow(`
        SELECT descricao, quantidade 
        FROM estoque_produtos 
        WHERE id_clipp = ?`, idEstoque).Scan(&existingDescricao, &existingQuantidade)
	if err != nil {
		if err == sql.ErrNoRows {
			// No record exists, insert new record
			_, err = insertStmt.Exec(idEstoque, descricao, qtdAtual)
			if err != nil {
				log.Printf("error inserting MySQL record for id_clipp %d: %v", idEstoque, err)
				return
			}
			mu.Lock()
			*insertedCount++
			mu.Unlock()
			return
		}
		log.Printf("error checking MySQL record for id_clipp %d: %v", idEstoque, err)
		return
	}

	// Record exists, check if update is needed
	if existingDescricao.Valid && existingQuantidade.Valid &&
		existingDescricao.String == descricao && existingQuantidade.Float64 == qtdAtual {
		mu.Lock()
		*ignoredCount++
		mu.Unlock()
		return
	}

	// Update existing record
	_, err = updateStmt.Exec(descricao, qtdAtual, idEstoque)
	if err != nil {
		log.Printf("error updating MySQL record for id_clipp %d: %v", idEstoque, err)
		return
	}
	mu.Lock()
	*updatedCount++
	mu.Unlock()
}
