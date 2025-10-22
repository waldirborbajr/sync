<?php
// Configurações de conexão
$firebird_config = [
    'dbname'   => 'C:\caminho\para\seu\database.fdb', // ou servidor: 'localhost/3050:C:\path\database.fdb'
    'username' => 'sysdba',
    'password' => 'masterkey',
    'charset'  => 'UTF8'
];

$mysql_config = [
    'host'     => 'localhost',
    'dbname'   => 'seu_banco_mysql',
    'username' => 'seu_usuario',
    'password' => 'sua_senha',
    'charset'  => 'utf8mb4'
];

// Função para conectar Firebird
function conectarFirebird($config) {
    $dsn = "firebird:dbname={$config['dbname']},charset={$config['charset']}";
    try {
        $pdo = new PDO($dsn, $config['username'], $config['password'], [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC
        ]);
        return $pdo;
    } catch (PDOException $e) {
        die("Erro ao conectar no Firebird: " . $e->getMessage());
    }
}

// Função para conectar MySQL
function conectarMySQL($config) {
    $dsn = "mysql:host={$config['host']};dbname={$config['dbname']};charset={$config['charset']}";
    try {
        $pdo = new PDO($dsn, $config['username'], $config['password'], [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::MYSQL_ATTR_INIT_COMMAND => "SET NAMES utf8mb4"
        ]);
        return $pdo;
    } catch (PDOException $e) {
        die("Erro ao conectar no MySQL: " . $e->getMessage());
    }
}

// Função principal de sincronização
function sincronizarEstoque($pdo_firebird, $pdo_mysql) {
    $sql_firebird = "
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
    ";
    
    try {
        // Buscar dados do Firebird
        $stmt = $pdo_firebird->query($sql_firebird);
        $dados = $stmt->fetchAll();
        
        if (empty($dados)) {
            echo "Nenhum dado encontrado no Firebird.\n";
            return;
        }
        
        echo "Total de registros encontrados: " . count($dados) . "\n";
        
        // Preparar INSERT/UPDATE para MySQL
        $sql_insert = "INSERT INTO tb_estoque_sinc (ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR) 
                       VALUES (:id_estoque, :descricao, :qtd_atual, :prc_custo, :prc_dolar)
                       ON DUPLICATE KEY UPDATE 
                       DESCRICAO = VALUES(DESCRICAO),
                       QTD_ATUAL = VALUES(QTD_ATUAL),
                       PRC_CUSTO = VALUES(PRC_CUSTO),
                       PRC_DOLAR = VALUES(PRC_DOLAR)";
        
        $stmt_mysql = $pdo_mysql->prepare($sql_insert);
        
        $inseridos = 0;
        $atualizados = 0;
        
        // Processar em lotes para melhor performance
        $pdo_mysql->beginTransaction();
        
        foreach ($dados as $linha) {
            $stmt_mysql->execute([
                ':id_estoque' => $linha['ID_ESTOQUE'],
                ':descricao'  => $linha['DESCRICAO'] ?? '',
                ':qtd_atual'  => $linha['QTD_ATUAL'] ?? 0,
                ':prc_custo'  => $linha['PRC_CUSTO'] ?? 0,
                ':prc_dolar'  => $linha['PRC_DOLAR'] ?? 0
            ]);
            
            $rowCount = $stmt_mysql->rowCount();
            if ($rowCount > 0) {
                if ($stmt_mysql->rowCount() == 2) { // MySQL retorna 2 quando atualiza
                    $atualizados++;
                } else {
                    $inseridos++;
                }
            }
        }
        
        $pdo_mysql->commit();
        
        echo "Sincronização concluída!\n";
        echo "Registros inseridos: $inseridos\n";
        echo "Registros atualizados: $atualizados\n";
        echo "Total processado: " . ($inseridos + $atualizados) . "\n";
        
    } catch (Exception $e) {
        $pdo_mysql->rollBack();
        die("Erro na sincronização: " . $e->getMessage());
    }
}

// Função para criar tabela no MySQL (se não existir)
function criarTabelaMySQL($pdo) {
    $sql_create = "
    CREATE TABLE IF NOT EXISTS `tb_estoque_sinc` (
        `ID_ESTOQUE` int(11) NOT NULL,
        `DESCRICAO` varchar(255) DEFAULT NULL,
        `QTD_ATUAL` decimal(15,4) DEFAULT 0,
        `PRC_CUSTO` decimal(15,4) DEFAULT 0,
        `PRC_DOLAR` decimal(15,4) DEFAULT 0,
        PRIMARY KEY (`ID_ESTOQUE`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    ";
    
    try {
        $pdo->exec($sql_create);
        echo "Tabela tb_estoque_sinc criada/verificada com sucesso!\n";
    } catch (PDOException $e) {
        die("Erro ao criar tabela: " . $e->getMessage());
    }
}

// EXECUÇÃO PRINCIPAL
echo "=== INICIANDO SINCRONIZAÇÃO FIREBIRD -> MySQL ===\n";

try {
    // Conectar aos bancos
    echo "Conectando aos bancos de dados...\n";
    $pdo_firebird = conectarFirebird($firebird_config);
    $pdo_mysql = conectarMySQL($mysql_config);
    
    echo "Conexões estabelecidas com sucesso!\n";
    
    // Criar/verificar tabela no MySQL
    criarTabelaMySQL($pdo_mysql);
    
    // Executar sincronização
    sincronizarEstoque($pdo_firebird, $pdo_mysql);
    
} catch (Exception $e) {
    echo "Erro geral: " . $e->getMessage() . "\n";
}

// Fechar conexões
$pdo_firebird = null;
$pdo_mysql = null;

echo "=== SINCRONIZAÇÃO FINALIZADA ===\n";
?>
