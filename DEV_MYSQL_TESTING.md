# MySQL Mock Testing Guide

This guide explains how to test different synchronization scenarios using the MySQL mock database in DEV_MODE.

## Quick Start

The MySQL mock database (`dev_mysql.db`) can be initialized with pre-existing data from `dev_mysql_data.sql`. This allows you to test all three sync operations:

- **INSERT**: New records from Firebird that don't exist in MySQL
- **UPDATE**: Existing records in MySQL that have changed in Firebird
- **IGNORED**: Records that are identical in both databases

## Testing Scenarios

### Scenario 1: Test INSERTS (Empty MySQL)

Test a clean initial sync where all Firebird records are inserted into an empty MySQL database.

**Setup:**
```bash
# Remove both databases
rm -f dev_firebird.db dev_mysql.db

# Run the app (or use reset script without dev_mysql_data.sql)
go run .
```

**Expected Result:**
- All ~110 active Firebird products will be INSERTED
- 0 updates
- 0 ignored

**Files needed:**
- `dev_firebird_data.sql` (for source data)
- NO `dev_mysql_data.sql` (empty target)

---

### Scenario 2: Test UPDATES (Pre-existing Data)

Test updating existing records when Firebird data has changed.

**Setup:**
```bash
# Make sure dev_mysql_data.sql exists with sample records
cat dev_mysql_data.sql  # Should have 3 pre-existing records

# Reset databases
./reset_dev_db.sh

# Run sync
go run .
```

**Expected Result:**
- 3 records will be UPDATED (IDs: 1001, 1003, 1005)
- ~107 records will be INSERTED (new from Firebird)
- 0 ignored (on first run)

**Files needed:**
- `dev_firebird_data.sql` (source data)
- `dev_mysql_data.sql` (with 3 sample records)

---

### Scenario 3: Test IGNORED (No Changes)

Test the optimization that skips records when data hasn't changed.

**Setup:**
```bash
# Run sync twice
go run .   # First run: updates/inserts
go run .   # Second run: should ignore all (no changes)
```

**Expected Result (second run):**
- 0 inserts
- 0 updates
- ~110 records IGNORED (all records match exactly)

---

### Scenario 4: Test MIXED Operations

Test a realistic scenario with a mix of inserts, updates, and ignored records.

**Setup:**
```bash
# Edit dev_mysql_data.sql to have:
# - Some old data (will UPDATE)
# - Some exact matches (will IGNORE)
# - Missing records (will INSERT)

# Reset and run
./reset_dev_db.sh
go run .
```

**Example:** Edit `dev_mysql_data.sql`:
```sql
-- Old data (will UPDATE)
INSERT INTO TB_ESTOQUE (...) VALUES (1001, 'OLD DATA', ...);

-- Exact match (will IGNORE) - uncomment in dev_mysql_data.sql:
INSERT INTO TB_ESTOQUE (...) VALUES 
    (1002, 'Mouse Logitech MX Master 3 Wireless', 45, 425.50, 78.80, ...);

-- Missing records (will INSERT): Everything else not in MySQL
```

---

## Customizing Test Data

### Add More Pre-existing Records

Edit `dev_mysql_data.sql` and add more records:

```sql
INSERT INTO TB_ESTOQUE (ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO, PRC_DOLAR, 
                        PRC_VENDA, PRC_3X, PRC_6X, PRC_10X) VALUES
    (2001, 'Samsung Galaxy - OLD DATA', 5, 1500.00, 280.00, 2100.00, 2205.00, 2310.00, 2415.00),
    (3001, 'Smart TV 50" - OLD DATA', 3, 2000.00, 370.00, 2800.00, 2940.00, 3080.00, 3220.00);
```

Then reset and run:
```bash
./reset_dev_db.sh
go run .
```

### Create Exact Matches for IGNORED Testing

To test the IGNORED path, add records that match Firebird exactly. Find the exact values in `dev_firebird_data.sql`:

```sql
-- From dev_firebird_data.sql, find exact values:
-- (1002, 'Mouse Logitech MX Master 3 Wireless', 425.50, 'A')
-- With QTD_ATUAL: 45, PRC_DOLAR: 78.80

-- Calculate expected prices using your LUCRO config
-- Then add to dev_mysql_data.sql with those exact values
```

Or simply run sync twice - the second run will ignore all records.

---

## Verifying Results

### Check Record Counts

```bash
# Before sync - MySQL count
sqlite3 dev_mysql.db "SELECT COUNT(*) as mysql_records FROM TB_ESTOQUE;"

# After sync
sqlite3 dev_mysql.db "SELECT COUNT(*) as mysql_records FROM TB_ESTOQUE;"

# Compare with Firebird active records
sqlite3 dev_firebird.db "SELECT COUNT(*) as active_records FROM TB_ESTOQUE WHERE STATUS='A';"
```

### Inspect Specific Records

```bash
# Check if record was updated
sqlite3 dev_mysql.db "SELECT * FROM TB_ESTOQUE WHERE ID_ESTOQUE=1001;" -header -column

# Compare with Firebird source
sqlite3 dev_firebird.db "
    SELECT e.ID_ESTOQUE, e.DESCRICAO, p.QTD_ATUAL, e.PRC_CUSTO, i.VALOR as PRC_DOLAR
    FROM TB_ESTOQUE e
    JOIN TB_EST_PRODUTO p ON e.ID_ESTOQUE = p.ID_IDENTIFICADOR
    LEFT JOIN TB_EST_INDEXADOR i ON i.ID_ESTOQUE = e.ID_ESTOQUE
    WHERE e.ID_ESTOQUE=1001;
" -header -column
```

### Check Price Calculations

Verify that prices were calculated correctly:

```bash
sqlite3 dev_mysql.db "
    SELECT 
        ID_ESTOQUE,
        DESCRICAO,
        PRC_CUSTO,
        PRC_VENDA,
        ROUND((PRC_VENDA - PRC_CUSTO) / PRC_CUSTO * 100, 2) as margin_percent,
        PRC_3X,
        PRC_6X,
        PRC_10X
    FROM TB_ESTOQUE 
    WHERE ID_ESTOQUE IN (1001, 1002, 1003)
    ORDER BY ID_ESTOQUE;
" -header -column
```

---

## Performance Testing

### Measure Sync Speed

```bash
# Time the sync operation
time go run .
```

**Note:** DEV_MODE uses SQLite with serialized writes (MaxOpenConns=1), so it will be slower than production MySQL. This is expected and by design - dev mode is for testing correctness, not performance.

### Test with Different Batch Sizes

Edit your code to test different batch sizes (if applicable) and measure the impact:

```bash
# Run multiple times and compare
for i in {1..3}; do
    echo "Run $i:"
    rm -f dev_mysql.db
    ./reset_dev_db.sh > /dev/null
    time go run . 2>&1 | grep "Total elapsed"
done
```

---

## Common Patterns

### Reset Before Each Test

```bash
#!/bin/bash
# test_sync.sh - Template for testing

# Reset databases
./reset_dev_db.sh

# Optional: Modify dev_mysql_data.sql for your test case
# Edit dev_mysql_data.sql here or use sed/awk

# Run sync
echo "Running sync..."
go run .

# Verify results
echo ""
echo "Results:"
sqlite3 dev_mysql.db "SELECT COUNT(*) FROM TB_ESTOQUE;" 
```

### Compare Before and After

```bash
# Save before state
sqlite3 dev_mysql.db "SELECT * FROM TB_ESTOQUE ORDER BY ID_ESTOQUE;" > before.txt

# Run sync
go run .

# Save after state
sqlite3 dev_mysql.db "SELECT * FROM TB_ESTOQUE ORDER BY ID_ESTOQUE;" > after.txt

# Compare
diff before.txt after.txt | head -20
```

---

## Troubleshooting

### "database is locked" errors

Already fixed! But if you see this:
```bash
rm -f dev_*.db
go run .
```

### Records not updating as expected

1. Check source data in Firebird:
   ```bash
   sqlite3 dev_firebird.db "SELECT * FROM TB_ESTOQUE WHERE ID_ESTOQUE=1001;" -header -column
   ```

2. Check comparison logic in processor code
3. Enable DEBUG_MODE=true in .env for detailed logs

### All records showing as IGNORED unexpectedly

This usually means:
- You ran sync twice without changing source data (expected behavior)
- Or dev_mysql_data.sql has exact matches that you don't expect

To verify:
```bash
# Check if data matches exactly
sqlite3 dev_mysql.db "SELECT ID_ESTOQUE, DESCRICAO, QTD_ATUAL, PRC_CUSTO FROM TB_ESTOQUE WHERE ID_ESTOQUE=1001;"
sqlite3 dev_firebird.db "SELECT e.ID_ESTOQUE, e.DESCRICAO, p.QTD_ATUAL, e.PRC_CUSTO FROM TB_ESTOQUE e JOIN TB_EST_PRODUTO p ON e.ID_ESTOQUE=p.ID_IDENTIFICADOR WHERE e.ID_ESTOQUE=1001;"
```

---

## Summary

The MySQL mock testing system gives you:
- ✅ Complete control over test scenarios
- ✅ Ability to test INSERT, UPDATE, and IGNORED paths
- ✅ No need for actual MySQL server installation
- ✅ Fast iteration and reproducible tests
- ✅ Safe testing environment

**Pro Tip:** Keep multiple versions of `dev_mysql_data.sql` for different test scenarios:
- `dev_mysql_data_empty.sql` - Empty table
- `dev_mysql_data_partial.sql` - Some old records
- `dev_mysql_data_full.sql` - Matches Firebird exactly

Then copy the one you need before testing:
```bash
cp dev_mysql_data_partial.sql dev_mysql_data.sql
./reset_dev_db.sh
go run .
```
