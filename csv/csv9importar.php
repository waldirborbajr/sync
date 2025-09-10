<?php
// Configuração da conexão com o banco de dados (ajuste os valores conforme necessário)
$servername = "192.168.0.4";
$username = "root"; // Substitua pelo seu usuário do banco
$password = "@senha"; // Substitua pela sua senha do banco
$dbname = "Controle_KBB"; // Substitua pelo nome do seu banco de dados

// Cria a conexão
$conn = new mysqli($servername, $username, $password, $dbname);

// Verifica a conexão
if ($conn->connect_error) {
    die("Falha na conexão: " . $conn->connect_error);
}

// Nome do arquivo CSV
$csvFile = "parts_27_08_25 11_27_data.csv";

// Remove limite de tempo de execução (use com cautela)
set_time_limit(0);

// Array para rastrear dados únicos por Código de peça
$data = [];

// Inicia o tempo de execução
$startTime = microtime(true);

// Abre o arquivo CSV
if (($handle = fopen($csvFile, "r")) !== FALSE) {
    // Ignora o cabeçalho
    fgetcsv($handle, 1000, ",");

    // Lê as linhas do CSV linha por linha
    $lineCount = 0;
    $intangibleCount = 0;
    $ignoredCount = 0;
    $errorLog = [];
    while (($row = fgetcsv($handle, 1000, ",")) !== FALSE) {
        $lineCount++;
        if (count($row) < 11) { // Verifica se a linha tem os campos esperados
            $errorLog[] = "Linha $lineCount incompleta: " . implode(",", $row);
            continue;
        }

        $produto = $row[0];
        $codigo_peca = $row[1];
        $descricao = $row[3];
        $tipo_peca = $row[4];
        $nivel_mao_obra = $row[5];
        $moeda = $row[6];
        $opcao_valor = trim($row[7]);
        $preco = (float) $row[8];
        $codigo_eee = $row[9];
        $peca_substituta = $row[10];

        // Verifica se é "Serviço intangível" e ignora ambas as linhas se já existe
        if ($tipo_peca === "Serviço intangível") {
            if (isset($data[$codigo_peca])) {
                unset($data[$codigo_peca]); // Remove a entrada existente
                $intangibleCount += 2; // Conta as duas linhas ignoradas
                continue;
            }
            $intangibleCount++;
            continue; // Ignora a linha atual
        }

        // Inicializa o array para o código se não existir
        if (!isset($data[$codigo_peca])) {
            $data[$codigo_peca] = [
                'produto' => $produto,
                'descricao' => $descricao,
                'tipo_peca' => $tipo_peca,
                'mao_obra' => $nivel_mao_obra,
                'moeda' => $moeda,
                'codigo_eee' => $codigo_eee,
                'peca_substituta' => $peca_substituta,
                'precos' => []
            ];
        }

        // Armazena o preço pela opção de valor
        $data[$codigo_peca]['precos'][$opcao_valor] = $preco;

        // Limpa memória periodicamente
        if ($lineCount % 10000 == 0) {
            gc_collect_cycles();
            echo "Processadas $lineCount linhas...\n";
        }
    }
    fclose($handle);

    // Processa e insere cada grupo único
    $insertedCount = 0;

    foreach ($data as $codigo => $info) {
        // Obtém os preços (usa 0 se não existir)
        $preco_troca = isset($info['precos']['Preço de troca']) ? $info['precos']['Preço de troca'] : 0;
        $preco_estoque = isset($info['precos']['Preço de estoque']) ? $info['precos']['Preço de estoque'] : 0;

        // Determina o menor e o maior preço
        $preco_st = min($preco_troca, $preco_estoque);
        $preco_estoque_db = max($preco_troca, $preco_estoque);

        // Opcao_de_valor combinada (deixamos vazia, pois é um registro único)
        $opcao = '';

        // SQL para inserção (assume que Id é auto-incremento)
        $sql = "INSERT INTO Lista_Apple (Produto, PartNumber, Descricao, Tipo_de_peca, Mao_de_obra, Moeda, Opcao_de_valor, Preco_ST, Preco_Estoque, Codigo_EEE, Peca_substituta)
                VALUES (
                    '" . $conn->real_escape_string($info['produto']) . "',
                    '" . $conn->real_escape_string($codigo) . "',
                    '" . $conn->real_escape_string($info['descricao']) . "',
                    '" . $conn->real_escape_string($info['tipo_peca']) . "',
                    '" . $conn->real_escape_string($info['mao_obra']) . "',
                    '" . $conn->real_escape_string($info['moeda']) . "',
                    '" . $conn->real_escape_string($opcao) . "',
                    " . $preco_st . ",
                    " . $preco_estoque_db . ",
                    '" . $conn->real_escape_string($info['codigo_eee']) . "',
                    '" . $conn->real_escape_string($info['peca_substituta']) . "'
                )";

        // Executa a query
        if ($conn->query($sql) === TRUE) {
            $insertedCount++;
            // echo "Registro inserido com sucesso para o código: $codigo\n";
        } else {
            $errorLog[] = "Erro ao inserir código $codigo: " . $conn->error;
        }
    }

    // Exibe erros, se houver
    if (!empty($errorLog)) {
        echo "Erros encontrados:\n";
        foreach ($errorLog as $error) {
            echo "$error\n";
        }
    }
} else {
    echo "Erro ao abrir o arquivo CSV.\n";
}

// Fecha a conexão
$conn->close();

// Calcula o tempo decorrido
$endTime = microtime(true);
$executionTime = $endTime - $startTime;

// Exibe o resumo
echo "Resumo:\n";
echo "Total de linhas do .csv (ignorando cabeçalho): $lineCount\n";
echo "Total de linhas únicas inseridas: $insertedCount\n";
echo "Total de intangíveis encontrados: $intangibleCount\n";
echo "Total de linhas ignoradas: $ignoredCount\n";
echo "Tempo decorrido: " . number_format($executionTime, 2) . " segundos\n";
?>