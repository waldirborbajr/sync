-- ============================================================================
-- MySQL Mock Database - Sample Data for Testing
-- ============================================================================
-- This file contains sample data to populate the SQLite MySQL mock database
-- Usage: sqlite3 dev_mysql.db < dev_mysql_data.sql
-- 
-- Purpose: Simulates a MySQL target database with some existing records
-- This allows testing both INSERT (new records) and UPDATE (existing records)
-- 
-- To reset the database:
--   rm dev_mysql.db && sqlite3 dev_mysql.db < dev_mysql_data.sql
-- ============================================================================

-- Drop existing table if it exists
DROP TABLE IF EXISTS TB_ESTOQUE;

-- Create MySQL target table structure
CREATE TABLE TB_ESTOQUE (
    ID_ESTOQUE INTEGER PRIMARY KEY,
    DESCRICAO TEXT NOT NULL,
    QTD_ATUAL REAL DEFAULT 0,
    PRC_CUSTO REAL DEFAULT 0,
    PRC_DOLAR REAL DEFAULT 0,
    PRC_VENDA REAL DEFAULT 0,
    PRC_3X REAL DEFAULT 0,
    PRC_6X REAL DEFAULT 0,
    PRC_10X REAL DEFAULT 0
);

-- ============================================================================
-- Pre-existing records in MySQL (simulating existing data in target database)
-- ============================================================================
-- These records simulate a target MySQL database that already has some data.
-- When sync runs, some will be UPDATED (if data changed in Firebird),
-- and new records from Firebird will be INSERTED.
-- 
-- Note: You can add more or remove these records to test different scenarios:
--   - Empty table: Tests all INSERTs
--   - Partial data: Tests mix of INSERTs and UPDATEs
--   - Full data with old prices: Tests UPDATEs with price recalculation
-- ============================================================================

-- Some existing records with older/different values (will be updated)
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, PRC_VENDA, PRC_3X, PRC_6X, PRC_10X) VALUES
    (1001, 'Notebook Dell Inspiron 15 - OLD DATA', 10, 2700.00, 500.00, 3780.00, 3969.00, 4158.00, 4347.00),
    (1003, 'Teclado Mecânico RGB Gamer - OLD DATA', 30, 350.00, 65.00, 490.00, 514.50, 539.00, 563.50),
    (1005, 'Webcam Full HD 1080p - OLD DATA', 20, 250.00, 46.50, 350.00, 367.50, 385.00, 402.50);

-- Some records that match Firebird exactly (will be ignored - no changes)
-- Uncomment these to test the IGNORED path
-- INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, PRC_VENDA, PRC_3X, PRC_6X, PRC_10X) VALUES
--     (1002, 'Mouse Logitech MX Master 3 Wireless', 45, 425.50, 78.80, 595.70, 625.49, 655.27, 685.06);

-- Note: Records NOT in this file but present in Firebird will be INSERTED
-- Examples from Firebird that will be inserted (if not uncommented above):
--   - ID 1002: Mouse Logitech (will be INSERTED)
--   - ID 1004, 1006-1099: All other electronics (will be INSERTED)
--   - ID 2001-2099: All home & kitchen items (will be INSERTED)
--   - ID 3001-3099: All sports & outdoor items (will be INSERTED)
--   - ID 4001-4099: All office supplies (will be INSERTED)
--   - ID 5001-5099: All audio & video items (will be INSERTED)
--   - ID 17973: Special test product (will be INSERTED)
--   - And many more...

-- ============================================================================
-- Testing Scenarios
-- ============================================================================
-- 
-- Scenario 1: Test UPDATES
--   Keep the 3 records above -> They will be updated with new Firebird data
-- 
-- Scenario 2: Test INSERTS
--   Comment out all INSERT statements -> All Firebird records will be inserted
-- 
-- Scenario 3: Test MIXED (INSERT + UPDATE)
--   Keep some records -> Mix of updates and inserts
-- 
-- Scenario 4: Test IGNORED (no changes)
--   Uncomment the exact match for ID 1002 -> Will be ignored (no changes)
--   Run sync twice -> Second run should ignore all records
-- 
-- ============================================================================
