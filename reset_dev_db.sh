#!/bin/bash
# ============================================================================
# Reset Development Databases
# ============================================================================
# This script resets the SQLite mock databases for development
# Usage: ./reset_dev_db.sh
# ============================================================================

set -e

echo "🔄 Resetting development databases..."

# Remove existing databases
if [ -f "dev_firebird.db" ]; then
    rm dev_firebird.db
    echo "✓ Removed old dev_firebird.db"
fi

if [ -f "dev_mysql.db" ]; then
    rm dev_mysql.db
    echo "✓ Removed old dev_mysql.db"
fi

# Create Firebird mock with sample data
echo ""
echo "📦 Creating Firebird mock database..."
sqlite3 dev_firebird.db < dev_firebird_data.sql
echo "✓ Firebird mock created with 110 products"

# Create MySQL mock (empty table)
echo ""
echo "📦 Creating MySQL mock database..."
sqlite3 dev_mysql.db <<EOF
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
EOF
echo "✓ MySQL mock created (empty table)"

echo ""
echo "✅ Development databases reset successfully!"
echo ""
echo "📊 Database Statistics:"
sqlite3 dev_firebird.db "SELECT 
    COUNT(*) as total,
    SUM(CASE WHEN STATUS='A' THEN 1 ELSE 0 END) as active,
    SUM(CASE WHEN STATUS='I' THEN 1 ELSE 0 END) as inactive
FROM TB_ESTOQUE;" -header -column

echo ""
echo "🚀 You can now run: go run ."
