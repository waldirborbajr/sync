-- ============================================================================
-- Firebird Mock Database - Sample Data
-- ============================================================================
-- This file contains sample data to populate the SQLite Firebird mock database
-- Usage: sqlite3 dev_firebird.db < dev_firebird_data.sql
-- 
-- To reset the database:
--   rm dev_firebird.db && sqlite3 dev_firebird.db < dev_firebird_data.sql
-- ============================================================================

-- Drop existing tables if they exist
DROP TABLE IF EXISTS TB_EST_INDEXADOR;
DROP TABLE IF EXISTS TB_EST_PRODUTO;
DROP TABLE IF EXISTS TB_ESTOQUE;

-- Create tables
CREATE TABLE TB_ESTOQUE (
    ID_ESTOQUE INTEGER PRIMARY KEY,
    DESCRICAO TEXT NOT NULL,
    PRC_CUSTO REAL,
    STATUS TEXT DEFAULT 'A'
);

CREATE TABLE TB_EST_PRODUTO (
    ID_IDENTIFICADOR INTEGER PRIMARY KEY,
    QTD_ATUAL REAL DEFAULT 0,
    FOREIGN KEY (ID_IDENTIFICADOR) REFERENCES TB_ESTOQUE(ID_ESTOQUE)
);

CREATE TABLE TB_EST_INDEXADOR (
    ID_ESTOQUE INTEGER PRIMARY KEY,
    VALOR REAL DEFAULT 0,
    FOREIGN KEY (ID_ESTOQUE) REFERENCES TB_ESTOQUE(ID_ESTOQUE)
);

-- ============================================================================
-- ELECTRONICS & COMPUTERS (IDs: 1000-1099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (1001, 'Notebook Dell Inspiron 15 i5 8GB 256GB SSD', 2850.00, 'A'),
    (1002, 'Mouse Logitech MX Master 3 Wireless', 425.50, 'A'),
    (1003, 'Teclado Mecânico Razer BlackWidow V3', 689.90, 'A'),
    (1004, 'Monitor LG UltraWide 29" Full HD', 1245.00, 'A'),
    (1005, 'Headset Gamer HyperX Cloud II 7.1', 385.00, 'A'),
    (1006, 'Webcam Logitech C920 Full HD 1080p', 425.00, 'A'),
    (1007, 'SSD Kingston 480GB SATA III', 285.00, 'A'),
    (1008, 'Memória RAM DDR4 16GB 3200MHz', 320.00, 'A'),
    (1009, 'HD Externo Seagate 2TB USB 3.0', 425.00, 'A'),
    (1010, 'Placa de Vídeo GTX 1650 4GB GDDR6', 1850.00, 'A'),
    (1011, 'Processador Intel Core i5 11400F', 965.00, 'A'),
    (1012, 'Fonte Corsair 650W 80 Plus Bronze', 385.00, 'A'),
    (1013, 'Gabinete Gamer RGB Mid Tower', 285.00, 'A'),
    (1014, 'Placa Mãe ASUS Prime B560M-A', 645.00, 'A'),
    (1015, 'Cooler CPU DeepCool Gammaxx 400', 125.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (1001, 12), (1002, 45), (1003, 28), (1004, 8), (1005, 34),
    (1006, 22), (1007, 56), (1008, 48), (1009, 18), (1010, 5),
    (1011, 15), (1012, 25), (1013, 19), (1014, 11), (1015, 38);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (1001, 525.50), (1002, 78.50), (1003, 127.20), (1004, 229.50),
    (1005, 71.00), (1006, 78.50), (1007, 52.50), (1008, 59.00),
    (1009, 78.50), (1010, 341.20), (1011, 178.00), (1012, 71.00),
    (1013, 52.50), (1014, 119.00), (1015, 23.00);

-- ============================================================================
-- SMARTPHONES & ACCESSORIES (IDs: 2000-2099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (2001, 'Samsung Galaxy A54 5G 128GB Preto', 1685.00, 'A'),
    (2002, 'iPhone 13 128GB Azul', 3850.00, 'A'),
    (2003, 'Xiaomi Redmi Note 12 Pro 256GB', 1425.00, 'A'),
    (2004, 'Motorola Edge 40 5G 256GB', 1985.00, 'A'),
    (2005, 'Carregador Turbo USB-C 33W Original', 85.50, 'A'),
    (2006, 'Capinha Anti-Impacto Transparente', 35.00, 'A'),
    (2007, 'Película de Vidro 3D Curva', 45.00, 'A'),
    (2008, 'Fone Bluetooth TWS AirDots 3', 125.00, 'A'),
    (2009, 'Power Bank 20000mAh Fast Charge', 145.00, 'A'),
    (2010, 'Suporte Veicular Magnético', 55.00, 'A'),
    (2011, 'Cabo USB-C Reforçado 2 Metros', 38.50, 'A'),
    (2012, 'Ring Light LED 10" com Tripé', 125.00, 'A'),
    (2013, 'Smartwatch Xiaomi Mi Band 7', 245.00, 'A'),
    (2014, 'Cartão MicroSD 128GB Classe 10', 85.00, 'A'),
    (2015, 'Hub USB-C 7 em 1 HDMI', 165.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (2001, 8), (2002, 3), (2003, 15), (2004, 6), (2005, 125),
    (2006, 245), (2007, 180), (2008, 67), (2009, 42), (2010, 98),
    (2011, 156), (2012, 34), (2013, 28), (2014, 76), (2015, 45);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (2001, 310.80), (2002, 710.20), (2003, 263.00), (2004, 366.20),
    (2005, 15.80), (2006, 6.45), (2007, 8.30), (2008, 23.00),
    (2009, 26.75), (2010, 10.15), (2011, 7.10), (2012, 23.00),
    (2013, 45.20), (2014, 15.70), (2015, 30.45);

-- ============================================================================
-- HOME APPLIANCES (IDs: 3000-3099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (3001, 'Smart TV LED 50" 4K Samsung', 2145.00, 'A'),
    (3002, 'Geladeira Frost Free 450L Brastemp', 3285.00, 'A'),
    (3003, 'Micro-ondas 30L Electrolux Branco', 485.00, 'A'),
    (3004, 'Ar Condicionado Split 12000 BTUs', 1685.00, 'A'),
    (3005, 'Lavadora de Roupas 12kg Automática', 1845.00, 'A'),
    (3006, 'Fogão 4 Bocas Atlas Mônaco Bivolt', 685.00, 'A'),
    (3007, 'Liquidificador Turbo Philips Walita', 185.00, 'A'),
    (3008, 'Cafeteira Expresso Nespresso', 485.00, 'A'),
    (3009, 'Aspirador de Pó Robô Multilaser', 925.00, 'A'),
    (3010, 'Ventilador de Torre Mallory', 285.00, 'A'),
    (3011, 'Ferro de Passar a Vapor Philco', 125.00, 'A'),
    (3012, 'Panela Elétrica de Arroz 10 Xícaras', 145.00, 'A'),
    (3013, 'Mixer Philco 600W com Acessórios', 95.00, 'A'),
    (3014, 'Torradeira Elétrica 2 Fatias', 85.00, 'A'),
    (3015, 'Chaleira Elétrica Inox 1.7L', 105.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (3001, 6), (3002, 3), (3003, 12), (3004, 8), (3005, 5),
    (3006, 9), (3007, 28), (3008, 14), (3009, 7), (3010, 22),
    (3011, 35), (3012, 18), (3013, 42), (3014, 38), (3015, 31);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (3001, 395.60), (3002, 606.00), (3003, 89.50), (3004, 311.00),
    (3005, 340.50), (3006, 126.40), (3007, 34.15), (3008, 89.50),
    (3009, 170.70), (3010, 52.60), (3011, 23.00), (3012, 26.75),
    (3013, 17.50), (3014, 15.70), (3015, 19.35);

-- ============================================================================
-- FURNITURE & DECORATION (IDs: 4000-4099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (4001, 'Sofá Retrátil 3 Lugares Cinza', 1685.00, 'A'),
    (4002, 'Mesa de Jantar 6 Cadeiras MDF', 985.00, 'A'),
    (4003, 'Rack para TV até 65" com Painel', 545.00, 'A'),
    (4004, 'Guarda Roupa Casal 6 Portas Branco', 1285.00, 'A'),
    (4005, 'Cama Box Queen Size Ortobom', 1485.00, 'A'),
    (4006, 'Escrivaninha com Gavetas 120cm', 385.00, 'A'),
    (4007, 'Cadeira Gamer Presidente Reclinável', 685.00, 'A'),
    (4008, 'Estante Livros 5 Prateleiras Rustic', 285.00, 'A'),
    (4009, 'Tapete Sala 200x150cm Cinza', 245.00, 'A'),
    (4010, 'Lustre Pendente Cristal 5 Lâmpadas', 385.00, 'A'),
    (4011, 'Espelho Decorativo Bisotado 90x60cm', 285.00, 'A'),
    (4012, 'Poltrona Decorativa Veludo Azul', 545.00, 'A'),
    (4013, 'Painel Parede 3D Auto Adesivo 10un', 125.00, 'A'),
    (4014, 'Kit 3 Quadros Decorativos Canvas', 145.00, 'A'),
    (4015, 'Prateleira Flutuante MDF 90cm Kit 3', 95.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (4001, 4), (4002, 6), (4003, 11), (4004, 3), (4005, 7),
    (4006, 15), (4007, 9), (4008, 18), (4009, 24), (4010, 8),
    (4011, 13), (4012, 5), (4013, 45), (4014, 32), (4015, 28);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (4001, 311.00), (4002, 181.80), (4003, 100.60), (4004, 237.10),
    (4005, 274.00), (4006, 71.00), (4007, 126.40), (4008, 52.60),
    (4009, 45.20), (4010, 71.00), (4011, 52.60), (4012, 100.60),
    (4013, 23.00), (4014, 26.75), (4015, 17.50);

-- ============================================================================
-- SPORTS & FITNESS (IDs: 5000-5099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (5001, 'Esteira Elétrica 12 Velocidades', 1685.00, 'A'),
    (5002, 'Bicicleta Ergométrica Horizontal', 925.00, 'A'),
    (5003, 'Kit Halteres 2kg a 10kg Emborrachado', 385.00, 'A'),
    (5004, 'Colchonete Yoga Premium 8mm', 85.00, 'A'),
    (5005, 'Corda de Pular Profissional Speed', 45.00, 'A'),
    (5006, 'Faixa Elástica Exercícios Kit 5un', 65.00, 'A'),
    (5007, 'Bola Suíça Pilates 65cm Anti-Burst', 75.00, 'A'),
    (5008, 'Tênis Running Nike Revolution 6', 285.00, 'A'),
    (5009, 'Bicicleta MTB Aro 29 Alumínio 21v', 1285.00, 'A'),
    (5010, 'Luva de Boxe Everlast 12oz', 185.00, 'A'),
    (5011, 'Kimono Jiu Jitsu A3 Competição', 285.00, 'A'),
    (5012, 'Bola de Futebol Society Penalty', 95.00, 'A'),
    (5013, 'Raquete Tênis Babolat Pure Drive', 985.00, 'A'),
    (5014, 'Patins Inline Ajustável Rollers', 285.00, 'A'),
    (5015, 'Squeeze Academia 1L com Alça', 35.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (5001, 3), (5002, 5), (5003, 12), (5004, 48), (5005, 85),
    (5006, 67), (5007, 34), (5008, 24), (5009, 8), (5010, 16),
    (5011, 22), (5012, 38), (5013, 6), (5014, 14), (5015, 125);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (5001, 311.00), (5002, 170.70), (5003, 71.00), (5004, 15.70),
    (5005, 8.30), (5006, 12.00), (5007, 13.85), (5008, 52.60),
    (5009, 237.10), (5010, 34.15), (5011, 52.60), (5012, 17.50),
    (5013, 181.80), (5014, 52.60), (5015, 6.45);

-- ============================================================================
-- FASHION & CLOTHING (IDs: 6000-6099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (6001, 'Jaqueta Jeans Masculina Destroyed', 145.00, 'A'),
    (6002, 'Vestido Longo Floral Feminino', 125.00, 'A'),
    (6003, 'Calça Jeans Skinny Feminina', 95.00, 'A'),
    (6004, 'Camisa Social Slim Fit Masculina', 85.00, 'A'),
    (6005, 'Tênis Casual Adidas SuperStar', 385.00, 'A'),
    (6006, 'Bolsa Transversal Feminina Couro', 185.00, 'A'),
    (6007, 'Relógio Masculino Orient Automático', 685.00, 'A'),
    (6008, 'Óculos de Sol Rayban Aviador', 485.00, 'A'),
    (6009, 'Mochila Executiva Notebook 15.6"', 145.00, 'A'),
    (6010, 'Cinto Couro Legítimo Masculino', 65.00, 'A'),
    (6011, 'Carteira Couro Pequena Feminina', 75.00, 'A'),
    (6012, 'Perfume Importado 100ml Masculino', 245.00, 'A'),
    (6013, 'Conjunto Moletom Masculino', 125.00, 'A'),
    (6014, 'Jaqueta Bomber Feminina Nylon', 165.00, 'A'),
    (6015, 'Boné Aba Reta Trucker Destroyed', 55.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (6001, 18), (6002, 22), (6003, 35), (6004, 28), (6005, 15),
    (6006, 12), (6007, 8), (6008, 11), (6009, 24), (6010, 45),
    (6011, 38), (6012, 16), (6013, 32), (6014, 19), (6015, 56);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (6001, 26.75), (6002, 23.00), (6003, 17.50), (6004, 15.70),
    (6005, 71.00), (6006, 34.15), (6007, 126.40), (6008, 89.50),
    (6009, 26.75), (6010, 12.00), (6011, 13.85), (6012, 45.20),
    (6013, 23.00), (6014, 30.45), (6015, 10.15);

-- ============================================================================
-- TOYS & GAMES (IDs: 7000-7099)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (7001, 'Lego Star Wars Millennium Falcon', 485.00, 'A'),
    (7002, 'Boneca Baby Alive Primeiros Sons', 185.00, 'A'),
    (7003, 'Carrinho Controle Remoto 4x4', 145.00, 'A'),
    (7004, 'PlayStation 5 Console 825GB', 3850.00, 'A'),
    (7005, 'Nintendo Switch OLED 64GB', 2485.00, 'A'),
    (7006, 'Xbox Series S 512GB', 2185.00, 'A'),
    (7007, 'Controle Joystick PS5 DualSense', 385.00, 'A'),
    (7008, 'Jogo FIFA 24 PS5', 285.00, 'A'),
    (7009, 'Quebra-Cabeça 2000 Peças Paisagem', 65.00, 'A'),
    (7010, 'Baralho Profissional Bicycle', 35.00, 'A'),
    (7011, 'Jogo Banco Imobiliário Clássico', 125.00, 'A'),
    (7012, 'Nerf Elite 2.0 Commander', 185.00, 'A'),
    (7013, 'Patinete Infantil 3 Rodas', 145.00, 'A'),
    (7014, 'Bola de Basquete Spalding Oficial', 185.00, 'A'),
    (7015, 'Slime Kit Faça Você Mesmo 12 Cores', 45.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (7001, 5), (7002, 14), (7003, 18), (7004, 2), (7005, 4),
    (7006, 6), (7007, 22), (7008, 28), (7009, 34), (7010, 95),
    (7011, 16), (7012, 12), (7013, 9), (7014, 25), (7015, 58);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (7001, 89.50), (7002, 34.15), (7003, 26.75), (7004, 710.20),
    (7005, 458.50), (7006, 403.30), (7007, 71.00), (7008, 52.60),
    (7009, 12.00), (7010, 6.45), (7011, 23.00), (7012, 34.15),
    (7013, 26.75), (7014, 34.15), (7015, 8.30);

-- ============================================================================
-- SPECIAL TEST PRODUCTS (IDs: 17970-17999)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (17973, 'Special Test Product - DO NOT DELETE', 1000.00, 'A'),
    (17974, 'High Value Item - Gold Edition', 5500.00, 'A'),
    (17975, 'Budget Product - Economy Line', 25.50, 'A'),
    (17976, 'Out of Stock Test Item', 850.00, 'A'),
    (17977, 'High Quantity Item - Bulk', 15.00, 'A');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (17973, 5),
    (17974, 1),
    (17975, 250),
    (17976, 0),
    (17977, 5000);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (17973, 184.80),
    (17974, 1015.00),
    (17975, 4.70),
    (17976, 156.90),
    (17977, 2.77);

-- ============================================================================
-- INACTIVE PRODUCTS (for testing STATUS filter)
-- ============================================================================
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, PRC_CUSTO, STATUS) VALUES
    (9001, 'DISCONTINUED - Old Model Keyboard', 150.00, 'I'),
    (9002, 'DISCONTINUED - Obsolete RAM Module', 200.00, 'I'),
    (9003, 'DISCONTINUED - Legacy Monitor', 800.00, 'I'),
    (9004, 'DISCONTINUED - Vintage Phone Case', 35.00, 'I'),
    (9005, 'DISCONTINUED - Classic T-Shirt', 45.00, 'I');

INSERT INTO TB_EST_PRODUTO (ID_IDENTIFICADOR, QTD_ATUAL) VALUES
    (9001, 0), (9002, 0), (9003, 0), (9004, 2), (9005, 3);

INSERT INTO TB_EST_INDEXADOR (ID_ESTOQUE, VALOR) VALUES
    (9001, 27.70), (9002, 36.90), (9003, 147.70),
    (9004, 6.45), (9005, 8.30);

-- ============================================================================
-- SUMMARY
-- ============================================================================
-- Total Products: 110
-- Active Products (STATUS='A'): 105
-- Inactive Products (STATUS='I'): 5
-- Product Categories: 8 (Electronics, Smartphones, Home, Furniture, Sports, Fashion, Toys, Test)
-- 
-- ID Ranges:
--   1000-1099: Electronics & Computers (15 products)
--   2000-2099: Smartphones & Accessories (15 products)
--   3000-3099: Home Appliances (15 products)
--   4000-4099: Furniture & Decoration (15 products)
--   5000-5099: Sports & Fitness (15 products)
--   6000-6099: Fashion & Clothing (15 products)
--   7000-7099: Toys & Games (15 products)
--   17970-17999: Special Test Products (5 products)
--   9000-9099: Inactive/Discontinued (5 products)
-- ============================================================================
