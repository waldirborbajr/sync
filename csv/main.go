package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type PecaInfo struct {
	Produto        string
	CodigoPeca     string
	Descricao      string
	TipoPeca       string
	MaoObra        string
	Moeda          string
	CodigoEEE      string
	PecaSubstituta string
	PrecoST        float64
	PrecoEstoqueDB float64
}

// Query base
const insertBase = `
INSERT INTO Lista_Apple
(Produto, PartNumber, Descricao, Tipo_de_peca, Mao_de_obra, Moeda, Opcao_de_valor, Preco_ST, Preco_Estoque, Codigo_EEE, Peca_substituta)
VALUES
`

// Configuração inicial
const (
	mysqlDSN        = "sync:MasterKey**@tcp(192.168.0.46:3306)/omni_db?charset=utf8mb4&parseTime=True&loc=Local"
	csvFilePath     = "parts_27_08_25 11_27_data.csv"
	showProgressMod = 10000
)

func main() {
	startAll := time.Now()

	db, err := sql.Open("mysql", mysqlDSN)
	must(err, "Erro ao abrir conexão MySQL")
	defer db.Close()

	// 1) Pega parâmetros do MySQL
	maxConns, maxPacket := getMySQLParams(db)

	// Define workers e batch dinamicamente
	numWorkers := maxConns / 4
	if numWorkers > 32 {
		numWorkers = 32
	}
	if numWorkers < 2 {
		numWorkers = 2
	}

	// batchSize ≈ max_allowed_packet / 2048 bytes médios por linha
	batchSize := int(maxPacket / 2048)
	if batchSize > 1000 {
		batchSize = 1000
	}
	if batchSize < 50 {
		batchSize = 50
	}

	fmt.Printf("MySQL max_connections=%d → usando %d workers\n", maxConns, numWorkers)
	fmt.Printf("MySQL max_allowed_packet=%d → usando batchSize=%d\n", maxPacket, batchSize)

	// 2) TRUNCATE
	_, err = db.Exec("TRUNCATE TABLE Lista_Apple")
	must(err, "Erro ao truncar a tabela Lista_Apple")
	fmt.Println("Tabela Lista_Apple truncada com sucesso.")

	// 3) Lê e consolida CSV
	startCSV := time.Now()
	data, infoMap, stats, err := loadAndAggregateCSV(csvFilePath)
	must(err, "Erro ao processar CSV")
	csvDuration := time.Since(startCSV).Seconds()

	// 4) Canal + workers
	pecaChan := make(chan PecaInfo, 2000)
	var wg sync.WaitGroup

	var mu sync.Mutex
	insertedCount := 0
	var errorLog []string

	// Semaphore baseado em max_connections
	sem := make(chan struct{}, numWorkers)

	startInsert := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			batch := make([]PecaInfo, 0, batchSize)
			flush := func() {
				if len(batch) == 0 {
					return
				}

				sem <- struct{}{} // acquire
				if err := execBatchInsert(db, batch); err != nil {
					mu.Lock()
					errorLog = append(errorLog, fmt.Sprintf("Worker %d - erro batch (%d registros): %v", workerID, len(batch), err))
					mu.Unlock()
				} else {
					mu.Lock()
					insertedCount += len(batch)
					mu.Unlock()
				}
				<-sem // release
				batch = batch[:0]
			}

			for p := range pecaChan {
				batch = append(batch, p)
				if len(batch) >= batchSize {
					flush()
				}
			}
			flush()
		}(i)
	}

	// Envia registros ao canal
	for codigo, precos := range data {
		info := infoMap[codigo]
		precoTroca := precos["Preço de troca"]
		precoEstoque := precos["Preço de estoque"]

		precoST := precoTroca
		if precoEstoque < precoST || precoST == 0 {
			precoST = precoEstoque
		}
		precoEstoqueDB := precoTroca
		if precoEstoque > precoEstoqueDB {
			precoEstoqueDB = precoEstoque
		}

		info.PrecoST = precoST
		info.PrecoEstoqueDB = precoEstoqueDB
		pecaChan <- info
	}
	close(pecaChan)

	// Espera workers
	wg.Wait()
	insertDuration := time.Since(startInsert).Seconds()

	// Logs de erro
	if len(errorLog) > 0 {
		fmt.Println("Erros encontrados:")
		for _, e := range errorLog {
			fmt.Println(e)
		}
	}

	// 5) Resumo
	totalDuration := time.Since(startAll).Seconds()
	fmt.Println("Resumo:")
	fmt.Printf("Total de linhas do CSV (ignorando cabeçalho): %d\n", stats.totalLines)
	fmt.Printf("Total de linhas únicas inseridas: %d\n", insertedCount)
	fmt.Printf("Total de intangíveis encontrados: %d\n", stats.intangible)
	fmt.Printf("Tempo CSV parse: %.2f segundos (%.0f linhas/s)\n", csvDuration, float64(stats.totalLines)/csvDuration)
	fmt.Printf("Tempo Inserts: %.2f segundos (%.0f registros/s)\n", insertDuration, float64(insertedCount)/insertDuration)
	fmt.Printf("Tempo total: %.2f segundos (%.0f registros/s end-to-end)\n",
		totalDuration, float64(insertedCount)/totalDuration)
}

// -------- Funções auxiliares --------

type csvStats struct {
	totalLines int
	intangible int
	ignored    int
}

func loadAndAggregateCSV(path string) (map[string]map[string]float64, map[string]PecaInfo, csvStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, csvStats{}, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	_, _ = r.Read() // ignora cabeçalho

	data := make(map[string]map[string]float64)
	infoMap := make(map[string]PecaInfo)
	stats := csvStats{}
	start := time.Now()

	for {
		row, err := r.Read()
		if err != nil {
			break
		}
		stats.totalLines++

		if len(row) < 11 {
			stats.ignored++
			continue
		}

		produto := row[0]
		codigoPeca := row[1]
		descricao := row[3]
		tipoPeca := row[4]
		maoObra := row[5]
		moeda := row[6]
		opcaoValor := strings.TrimSpace(row[7])
		precoStr := strings.ReplaceAll(row[8], ",", ".")
		preco, _ := strconv.ParseFloat(precoStr, 64)
		codigoEEE := row[9]
		pecaSubstituta := row[10]

		if tipoPeca == "Serviço intangível" {
			if _, ok := data[codigoPeca]; ok {
				delete(data, codigoPeca)
				delete(infoMap, codigoPeca)
				stats.intangible += 2
				continue
			}
			stats.intangible++
			continue
		}

		if _, ok := data[codigoPeca]; !ok {
			data[codigoPeca] = make(map[string]float64)
			infoMap[codigoPeca] = PecaInfo{
				Produto:        produto,
				CodigoPeca:     codigoPeca,
				Descricao:      descricao,
				TipoPeca:       tipoPeca,
				MaoObra:        maoObra,
				Moeda:          moeda,
				CodigoEEE:      codigoEEE,
				PecaSubstituta: pecaSubstituta,
			}
		}
		data[codigoPeca][opcaoValor] = preco

		if stats.totalLines%showProgressMod == 0 {
			fmt.Printf("Processadas %d linhas (%.1fs)\n", stats.totalLines, time.Since(start).Seconds())
		}
	}
	return data, infoMap, stats, nil
}

func execBatchInsert(db *sql.DB, batch []PecaInfo) error {
	if len(batch) == 0 {
		return nil
	}

	var b strings.Builder
	b.Grow(len(insertBase) + len(batch)*30)
	b.WriteString(insertBase)

	ph := "(?,?,?,?,?,?,?,?,?,?,?)"
	for i := 0; i < len(batch); i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(ph)
	}

	args := make([]any, 0, len(batch)*11)
	for _, p := range batch {
		args = append(args,
			p.Produto,
			p.CodigoPeca,
			p.Descricao,
			p.TipoPeca,
			p.MaoObra,
			p.Moeda,
			"", // opcao valor em branco
			p.PrecoST,
			p.PrecoEstoqueDB,
			p.CodigoEEE,
			p.PecaSubstituta,
		)
	}

	_, err := db.Exec(b.String(), args...)
	return err
}

func getMySQLParams(db *sql.DB) (int, int) {
	var maxConns int
	var maxPacket int

	db.QueryRow("SHOW VARIABLES LIKE 'max_connections'").Scan(new(string), &maxConns)
	db.QueryRow("SHOW VARIABLES LIKE 'max_allowed_packet'").Scan(new(string), &maxPacket)

	if maxConns == 0 {
		maxConns = 20
	}
	if maxPacket == 0 {
		maxPacket = 4 * 1024 * 1024
	}
	return maxConns, maxPacket
}

func must(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
