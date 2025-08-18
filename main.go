package main

import (
    "database/sql"
    "fmt"
    "log"
    "sync"
    "time"
    _ "github.com/go-sql-driver/mysql"    // MariaDB/MySQL driver
    _ "github.com/nakagami/firebirdsql"  // Firebird driver
)

func main() {
    // Track counts and start time
    var insertedCount, updatedCount, ignoredCount int
    var mu sync.Mutex // Mutex for thread-safe counter updates
    var wg sync.WaitGroup // WaitGroup to wait for all goroutines
    startTime := time.Now()

    // Semaphore to limit concurrent goroutines to 20
    semaphore := make(chan struct{}, 20)

    // Connect to Firebird database
    firebirdConn, err := sql.Open("firebirdsql", "SYSDBA:masterkey@192.168.0.15/C:\\Program Files (x86)\\CompuFour\\Clipp\\Base\\CLIPP.FDB")
    if err != nil {
        log.Fatal("Error connecting to Firebird:", err)
    }
    defer firebirdConn.Close()

    // Test Firebird connection
    if err = firebirdConn.Ping(); err != nil {
        log.Fatal("Failed to ping Firebird database:", err)
    }
    fmt.Println("Connected to Firebird database")

    // Connect to MySQL database
    dsn := "sync:MasterKey**@tcp(192.168.0.46:3306)/omni_db?charset=utf8&parseTime=True&loc=Local"
    mysqlConn, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("Error connecting to MySQL:", err)
    }
    defer mysqlConn.Close()

    // Test MySQL connection
    if err = mysqlConn.Ping(); err != nil {
        log.Fatal("Failed to ping MySQL database:", err)
    }
    fmt.Println("Connected to MySQL database")

    // Query Firebird database
    query := `
        SELECT e.ID_ESTOQUE, e.DESCRICAO, p.QTD_ATUAL
        FROM TB_ESTOQUE e, TB_EST_PRODUTO p
        WHERE e.ID_ESTOQUE = p.ID_IDENTIFICADOR
        AND e.status = 'A'
    `
    rows, err := firebirdConn.Query(query)
    if err != nil {
        log.Fatal("Error querying Firebird:", err)
    }
    defer rows.Close()

    // Prepare MySQL statements
    updateStmt, err := mysqlConn.Prepare(`
        UPDATE estoque_produtos
        SET descricao = ?, quantidade = ?
        WHERE id_clipp = ?
    `)
    if err != nil {
        log.Fatal("Error preparing MySQL update statement:", err)
    }
    defer updateStmt.Close()

    insertStmt, err := mysqlConn.Prepare(`
        INSERT INTO estoque_produtos (id_clipp, descricao, quantidade)
        VALUES (?, ?, ?)
    `)
    if err != nil {
        log.Fatal("Error preparing MySQL insert statement:", err)
    }
    defer insertStmt.Close()

    // Process each row from Firebird
    for rows.Next() {
        var idEstoque int
        var descricao string
        var qtdAtual float64
        if err := rows.Scan(&idEstoque, &descricao, &qtdAtual); err != nil {
            log.Printf("Error scanning Firebird row: %v", err)
            continue
        }

        // Acquire semaphore (limit to 20 concurrent goroutines)
        semaphore <- struct{}{}
        wg.Add(1)

        // Process each row in a goroutine
        go func(idEstoque int, descricao string, qtdAtual float64) {
            defer wg.Done()
            defer func() { <-semaphore }() // Release semaphore

            // Check if record exists in MySQL
            var count int
            var existingDescricao string
            var existingQuantidade float64
            err = mysqlConn.QueryRow(`
                SELECT COUNT(*), descricao, quantidade 
                FROM estoque_produtos 
                WHERE id_clipp = ?`, idEstoque).Scan(&count, &existingDescricao, &existingQuantidade)
            if err != nil && err != sql.ErrNoRows {
                log.Printf("Error checking MySQL record for id_clipp %d: %v", idEstoque, err)
                return
            }

            if count > 0 {
                // Check if update is needed
                if existingDescricao == descricao && existingQuantidade == qtdAtual {
                    // No changes needed, increment ignored count
                    mu.Lock()
                    ignoredCount++
                    mu.Unlock()
                    return
                }

                // Update existing record
                _, err = updateStmt.Exec(descricao, qtdAtual, idEstoque)
                if err != nil {
                    log.Printf("Error updating MySQL record for id_clipp %d: %v", idEstoque, err)
                    return
                }
                mu.Lock()
                updatedCount++
                mu.Unlock()
            } else {
                // Insert new record
                _, err = insertStmt.Exec(idEstoque, descricao, qtdAtual)
                if err != nil {
                    log.Printf("Error inserting MySQL record for id_clipp %d: %v", idEstoque, err)
                    return
                }
                mu.Lock()
                insertedCount++
                mu.Unlock()
            }
        }(idEstoque, descricao, qtdAtual)
    }

    if err = rows.Err(); err != nil {
        log.Fatal("Error iterating Firebird rows:", err)
    }

    // Wait for all goroutines to complete
    wg.Wait()

    // Calculate elapsed time
    elapsedTime := time.Since(startTime)

    // Print summary
    fmt.Printf("Data synchronization completed.\n")
    fmt.Printf("Total de linhas inseridas: %d\n", insertedCount)
    fmt.Printf("Total de linhas alteradas: %d\n", updatedCount)
    fmt.Printf("Total de linhas ignoradas: %d\n", ignoredCount)
    fmt.Printf("Tempo decorrido: %s\n", elapsedTime)
}