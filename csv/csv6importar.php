<?php
// Habilita exibição de erros para depuração
ini_set('display_errors', 1);
ini_set('display_startup_errors', 1);
error_reporting(E_ALL);

// Configuração da conexão com o banco de dados MySQL
$host = '192.168.0.4';
$dbname = 'Controle_KBB'; // Nome do banco especificado
$username = 'root'; // Substitua pelo seu usuário
$password = '@senha';   // Substitua pela sua senha

$pdo = new PDO("mysql:host=192.168.0.4;dbname=Controle_KBB", "root", "@senha");
echo "Conexão OK";

try {
    $pdo = new PDO("mysql:host=$host;dbname=$dbname", $username, $password);
    $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
    echo "<div class='message success'>Conexão ao MySQL bem-sucedida</div>";
} catch (PDOException $e) {
    die("<div class='message error'>Erro ao conectar ao MySQL: " . $e->getMessage() . "</div>");
}

// Função para criar a tabela Lista_Apple, se não existir
function criarTabela($pdo) {
    try {
        $sql = "
            CREATE TABLE IF NOT EXISTS Lista_Apple (
                id INT AUTO_INCREMENT PRIMARY KEY,
                Produto VARCHAR(100),
                PartNumber VARCHAR(50) UNIQUE,
                Descricao TEXT,
                Tipo_de_peca VARCHAR(50),
                Mao_de_obra DECIMAL(10,2),
                Moeda VARCHAR(10),
                Opcao_de_valor VARCHAR(20),
                Preco_ST DECIMAL(10,2),
                Preco_Estoque DECIMAL(10,2),
                Codigo_EEE VARCHAR(50),
                Peca_substituta VARCHAR(50)
            )
        ";
        $pdo->exec($sql);
        echo "<div class='message success'>Tabela Lista_Apple criada ou já existe</div>";
    } catch (PDOException $e) {
        echo "<div class='message error'>Erro ao criar tabela: " . $e->getMessage() . "</div>";
    }
}

// Função para verificar se o PartNumber já existe
function partNumberExiste($pdo, $partNumber) {
    try {
        $sql = "SELECT COUNT(*) FROM Lista_Apple WHERE PartNumber = ?";
        $stmt = $pdo->prepare($sql);
        $stmt->execute([$partNumber]);
        $exists = $stmt->fetchColumn() > 0;
        echo "<pre>Verificando PartNumber '$partNumber': " . ($exists ? "Existe" : "Não existe") . "</pre>";
        return $exists;
    } catch (PDOException $e) {
        echo "<div class='message error'>Erro ao verificar PartNumber: " . $e->getMessage() . "</div>";
        return false;
    }
}

// Verifica se o arquivo foi enviado
if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_FILES['arquivo'])) {
    $arquivo = $_FILES['arquivo'];

    // Verifica se o upload foi bem-sucedido
    if ($arquivo['error'] !== UPLOAD_ERR_OK) {
        die("<div class='message error'>Erro no upload do arquivo: Código " . $arquivo['error'] . "</div>");
    }

    // Verifica se é um arquivo CSV
    if ($arquivo['type'] !== 'text/csv' && pathinfo($arquivo['name'], PATHINFO_EXTENSION) !== 'csv') {
        die("<div class='message error'>Por favor, envie um arquivo CSV válido.</div>");
    }

    // Cria a tabela
    criarTabela($pdo);

    // Processa o arquivo CSV
    try {
        $file = fopen($arquivo['tmp_name'], 'r');
        if ($file === false) {
            die("<div class='message error'>Erro ao abrir o arquivo CSV.</div>");
        }

        // Pula o cabeçalho
        $header = fgetcsv($file);
        echo "<pre>Cabeçalho do CSV:\n";
        var_dump($header);
        echo "</pre>";

        $dados_para_processar = [];
        $item_atual = null;
        $preco_estoque = null;
        $preco_troca = null;
        $inseridos = 0;
        $atualizados = 0;
        $linha_num = 1; // Contador de linhas para depuração

        echo "<h3>Dados lidos do CSV:</h3>";
        while (($linha = fgetcsv($file)) !== false) {
            $linha_num++;
            // Verifica se a linha tem o número esperado de colunas
            if (count($linha) < 10) {
                echo "<div class='message error'>Linha $linha_num incompleta: " . implode(',', $linha) . "</div>";
                continue;
            }

            // Mapeia as colunas do CSV
            $produto = $linha[0] ?? '';
            $part_number = $linha[1] ?? '';
            $descricao = $linha[2] ?? '';
            $tipo_de_peca = $linha[3] ?? '';
            $mao_de_obra = !empty($linha[4]) ? floatval($linha[4]) : 0.0;
            $moeda = $linha[5] ?? '';
            $opcao_de_valor = $linha[6] ?? '';
            $preco = !empty($linha[7]) ? floatval($linha[7]) : 0.0;
            $codigo_eee = $linha[8] ?? '';
            $peca_substituta = $linha[9] ?? '';

            // var_dump da linha lida
            echo "<pre>Linha $linha_num lida do CSV:\n";
            var_dump([
                'Produto' => $produto,
                'PartNumber' => $part_number,
                'Descricao' => $descricao,
                'Tipo_de_peca' => $tipo_de_peca,
                'Mao_de_obra' => $mao_de_obra,
                'Moeda' => $moeda,
                'Opcao_de_valor' => $opcao_de_valor,
                'Preco' => $preco,
                'Codigo_EEE' => $codigo_eee,
                'Peca_substituta' => $peca_substituta
            ]);
            echo "</pre>";

            if ($item_atual != [$produto, $part_number]) {
                // Novo item encontrado
                if ($item_atual !== null) {
                    // Consolida o item anterior
                    if ($preco_estoque === null || $preco_troca === null) {
                        echo "<div class='message error'>Erro na linha $linha_num: Item incompleto para PartNumber {$item_atual[1]} (Estoque ou Troca ausente)</div>";
                    } else {
                        $dado = [
                            $item_atual[0], $item_atual[1], $descricao, $tipo_de_peca,
                            $mao_de_obra, $moeda, 'Estoque', $preco_troca,
                            $preco_estoque, $codigo_eee, $peca_substituta
                        ];
                        // var_dump do item consolidado
                        echo "<pre>Item consolidado para PartNumber {$item_atual[1]}:\n";
                        var_dump([
                            'Produto' => $dado[0],
                            'PartNumber' => $dado[1],
                            'Descricao' => $dado[2],
                            'Tipo_de_peca' => $dado[3],
                            'Mao_de_obra' => $dado[4],
                            'Moeda' => $dado[5],
                            'Opcao_de_valor' => $dado[6],
                            'Preco_ST' => $dado[7],
                            'Preco_Estoque' => $dado[8],
                            'Codigo_EEE' => $dado[9],
                            'Peca_substituta' => $dado[10]
                        ]);
                        echo "</pre>";
                        $dados_para_processar[] = $dado;
                    }
                }
                $item_atual = [$produto, $part_number];
                $preco_estoque = null;
                $preco_troca = null;
            }

            // Atribui o preço conforme a opção de valor
            if (strtolower($opcao_de_valor) === 'estoque') {
                $preco_estoque = $preco;
            } elseif (strtolower($opcao_de_valor) === 'troca') {
                $preco_troca = $preco;
            } else {
                echo "<div class='message error'>Linha $linha_num: Opcao_de_valor inválida: $opcao_de_valor</div>";
            }
        }

        // Processa o último item
        if ($item_atual !== null) {
            if ($preco_estoque === null || $preco_troca === null) {
                echo "<div class='message error'>Último item incompleto para PartNumber {$item_atual[1]} (Estoque ou Troca ausente)</div>";
            } else {
                $dado = [
                    $item_atual[0], $item_atual[1], $descricao, $tipo_de_peca,
                    $mao_de_obra, $moeda, 'Estoque', $preco_troca,
                    $preco_estoque, $codigo_eee, $peca_substituta
                ];
                // var_dump do último item consolidado
                echo "<pre>Último item consolidado para PartNumber {$item_atual[1]}:\n";
                var_dump([
                    'Produto' => $dado[0],
                    'PartNumber' => $dado[1],
                    'Descricao' => $dado[2],
                    'Tipo_de_peca' => $dado[3],
                    'Mao_de_obra' => $dado[4],
                    'Moeda' => $dado[5],
                    'Opcao_de_valor' => $dado[6],
                    'Preco_ST' => $dado[7],
                    'Preco_Estoque' => $dado[8],
                    'Codigo_EEE' => $dado[9],
                    'Peca_substituta' => $dado[10]
                ]);
                echo "</pre>";
                $dados_para_processar[] = $dado;
            }
        }

        fclose($file);

        // var_dump da lista final de dados a processar
        echo "<h3>Lista final de dados a processar (antes de INSERT/UPDATE):</h3>";
        echo "<pre>";
        var_dump($dados_para_processar);
        echo "</pre>";

        // Prepara as consultas de INSERT e UPDATE
        $sql_insert = "
            INSERT INTO Lista_Apple (
                Produto, PartNumber, Descricao, Tipo_de_peca, 
                Mao_de_obra, Moeda, Opcao_de_valor, Preco_ST, 
                Preco_Estoque, Codigo_EEE, Peca_substituta
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ";
        $sql_update = "
            UPDATE Lista_Apple SET
                Produto = ?, Descricao = ?, Tipo_de_peca = ?, 
                Mao_de_obra = ?, Moeda = ?, Opcao_de_valor = ?, 
                Preco_ST = ?, Preco_Estoque = ?, Codigo_EEE = ?, 
                Peca_substituta = ?
            WHERE PartNumber = ?
        ";
        $stmt_insert = $pdo->prepare($sql_insert);
        $stmt_update = $pdo->prepare($sql_update);

        // Processa os dados (inserir ou atualizar)
        foreach ($dados_para_processar as $dado) {
            try {
                if (partNumberExiste($pdo, $dado[1])) {
                    // Atualiza registro existente
                    echo "<pre>Atualizando PartNumber: {$dado[1]}</pre>";
                    $stmt_update->execute([
                        $dado[0], $dado[2], $dado[3], $dado[4], $dado[5], 
                        $dado[6], $dado[7], $dado[8], $dado[9], $dado[10], 
                        $dado[1]
                    ]);
                    $atualizados++;
                } else {
                    // Insere novo registro
                    echo "<pre>Inserindo PartNumber: {$dado[1]}</pre>";
                    $stmt_insert->execute($dado);
                    $inseridos++;
                }
            } catch (PDOException $e) {
                echo "<div class='message error'>Erro ao processar PartNumber {$dado[1]}: " . $e->getMessage() . "</div>";
            }
        }

        // Exibe resultado
        if ($inseridos > 0 || $atualizados > 0) {
            echo "<div class='message success'>$inseridos registros inseridos e $atualizados registros atualizados com sucesso.</div>";
        } else {
            echo "<div class='message error'>Nenhum registro foi inserido ou atualizado.</div>";
        }

    } catch (Exception $e) {
        echo "<div class='message error'>Erro ao processar CSV: " . $e->getMessage() . "</div>";
    }
} else {
    echo "<div class='message error'>Nenhum arquivo enviado.</div>";
}
?>

<a href="index.html">Voltar</a>