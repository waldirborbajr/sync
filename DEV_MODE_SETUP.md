# Development Mode Setup

This document explains how to set up and use the development mode feature, which allows you to run the sync application using SQLite mock databases instead of requiring real Firebird and MySQL servers.

## Overview

Development mode (`DEV_MODE=true`) creates two SQLite database files that simulate Firebird and MySQL:
- `dev_firebird.db` - Mocks the Firebird source database (with product data)
- `dev_mysql.db` - Mocks the MySQL target database (with optional pre-existing data)

**Automatic Data Loading**: 
- **Firebird Mock**: The application automatically looks for `dev_firebird_data.sql`:
  - If found: Loads 110 products across 8 categories
  - If not found: Uses minimal hardcoded data (7 products)

- **MySQL Mock**: The application automatically looks for `dev_mysql_data.sql`:
  - If found: Loads pre-existing records to simulate a target database with data
  - If not found: Creates an empty table (all records will be inserted during sync)

To get the full testing experience, simply ensure both SQL files are in your project root.

## Setup Instructions

### 1. Install SQLite Driver

Run the following command to download the SQLite driver dependency:

```bash
go get modernc.org/sqlite
```

Then tidy up the module dependencies:

```bash
go mod tidy
```

**Note**: This project uses `modernc.org/sqlite`, a pure Go SQLite implementation that doesn't require CGO. This makes it easier to cross-compile and work in containerized environments.

### 2. Configure Environment

Copy `.env.example` to `.env` and set `DEV_MODE=true`:

```bash
cp .env.example .env
```

Edit `.env`:

```env
# Development Mode - uses SQLite mocks instead of real databases
DEV_MODE=true

# When DEV_MODE=true, these credentials are not required
# FIREBIRD_USER=
# FIREBIRD_PASSWORD=
# FIREBIRD_HOST=
# FIREBIRD_PATH=

# MYSQL_USER=
# MYSQL_PASSWORD=
# MYSQL_HOST=
# MYSQL_PORT=
# MYSQL_DATABASE=

# Pricing Configuration (still required)
LUCRO=15.00
PARC3X=5.00
PARC6X=10.00
PARC10X=15.00

# Debug log
DEBUG_MODE=true
```

### 3. Run the Application

```bash
go run .
```

On first run, the application will:
1. Create `dev_firebird.db` with sample product data (from `dev_firebird_data.sql` if present)
2. Create `dev_mysql.db` with target table structure (from `dev_mysql_data.sql` if present)
3. Perform synchronization from Firebird mock to MySQL mock

## Mock Data

### Automatic Data Loading

When the application creates `dev_firebird.db`, it automatically checks for `dev_firebird_data.sql`:

**With `dev_firebird_data.sql` present:**
- 110 products across 8 categories
- Electronics, Smartphones, Home Appliances, Furniture, Sports, Fashion, Toys, Test products
- Realistic product names, prices, quantities, and USD values
- Mix of in-stock, low-stock, and out-of-stock items
- Inactive products to test STATUS filtering

**Without `dev_firebird_data.sql`:**
- Minimal 7 products for basic testing
- Includes special test product ID 17973
- One inactive product to verify STATUS='A' filter

### Firebird Mock (dev_firebird.db) - Full Dataset

When using `dev_firebird_data.sql`, you get 110 products organized as:

| Category | ID Range | Count | Examples |
|----------|----------|-------|----------|
| Electronics & Computers | 1000-1099 | 15 | Notebooks, Monitors, SSDs, GPUs |
| Smartphones & Accessories | 2000-2099 | 15 | Samsung, iPhone, Xiaomi, Cases |
| Home Appliances | 3000-3099 | 15 | Smart TVs, Refrigerators, Microwaves |
| Furniture & Decoration | 4000-4099 | 15 | Sofas, Tables, Beds, Lamps |
| Sports & Fitness | 5000-5099 | 15 | Treadmills, Bikes, Weights, Shoes |
| Fashion & Clothing | 6000-6099 | 15 | Jeans, Dresses, Sneakers, Watches |
| Toys & Games | 7000-7099 | 15 | Lego, PS5, Xbox, Nintendo Switch |
| Special Test Products | 17970-17999 | 5 | Including ID 17973 for testing |
| Inactive Products | 9000-9099 | 5 | Discontinued items (STATUS='I') |

**Sample Active Products:**

| ID_ESTOQUE | DESCRICAO | PRC_CUSTO | QTD_ATUAL | PRC_DOLAR | STATUS |
|------------|-----------|-----------|-----------|-----------|--------|
| 1001 | Notebook Dell Inspiron 15 i5 8GB 256GB SSD | 2850.00 | 12 | 525.50 | A |
| 2001 | Samsung Galaxy A54 5G 128GB Preto | 1685.00 | 8 | 310.80 | A |
| 7004 | PlayStation 5 Console 825GB | 3850.00 | 2 | 710.20 | A |
| 17973 | Special Test Product - DO NOT DELETE | 1000.00 | 5 | 184.80 | A |

### Firebird Mock - Minimal Dataset

When `dev_firebird_data.sql` is not found, you get 7 basic products:

| ID_ESTOQUE | DESCRICAO | PRC_CUSTO | QTD_ATUAL | PRC_DOLAR (USD) | STATUS |
|------------|-----------|-----------|-----------|-----------------|--------|
| 1 | Product A - Sample Item | 100.00 | 50 | 18.50 | A |
| 2 | Product B - Test Widget | 250.50 | 25 | 46.20 | A |
| 3 | Product C - Development Kit | 500.00 | 10 | 92.40 | A |
| 4 | Product D - Mock Component | 75.25 | 100 | 13.90 | A |
| 5 | Product E - Testing Tool | 150.00 | 35 | 27.70 | A |
| 17973 | Special Test Product | 1000.00 | 5 | 184.80 | A |
| 100 | Inactive Product | 200.00 | 0 | 36.90 | I |

Only products with `STATUS='A'` will be synced to MySQL.

### MySQL Mock (dev_mysql.db)

The MySQL target database simulation supports two modes:

**With `dev_mysql_data.sql` present (recommended for testing):**
- Simulates a target database with pre-existing records
- Contains 3 sample products with old/outdated data
- Allows testing UPDATE operations (when Firebird data differs)
- Allows testing INSERT operations (for new products not in MySQL)
- Allows testing IGNORED operations (when data matches exactly)

**Without `dev_mysql_data.sql`:**
- Starts with an empty `TB_ESTOQUE` table
- All Firebird records will be INSERTED during sync
- Good for testing clean initial sync

**Testing Scenarios** (by editing `dev_mysql_data.sql`):
1. **Test UPDATES**: Keep pre-existing records → They get updated with new Firebird data
2. **Test INSERTS**: Remove all INSERT statements → All Firebird records get inserted
3. **Test MIXED**: Keep some records → Mix of updates and inserts
4. **Test IGNORED**: Add exact matches → Records with no changes are ignored

After running the sync, the table will contain all active products from Firebird with calculated prices (PRC_VENDA, PRC_3X, PRC_6X, PRC_10X).

## Database Inspection

You can inspect the mock databases using the SQLite command-line tool:

```bash
# View Firebird mock data
sqlite3 dev_firebird.db "SELECT * FROM TB_ESTOQUE;"

# View MySQL mock data after sync
sqlite3 dev_mysql.db "SELECT * FROM TB_ESTOQUE;"
```

## Performance Characteristics

### SQLite Concurrency Handling

SQLite is optimized for single-writer scenarios. To prevent "database is locked" errors, the dev mode implementation:

- **Serializes writes**: `MaxOpenConns=1` ensures only one connection can write at a time
- **WAL mode enabled**: Write-Ahead Logging improves concurrent read performance
- **Busy timeout**: 5-second wait if database is locked
- **Sync mode**: Set to NORMAL for faster writes (safe for development)

**Expected behavior:**
- Multiple worker goroutines will queue writes sequentially
- Sync performance will be slower than production Firebird/MySQL
- Read operations remain fast and non-blocking

**Performance tip:** Development mode is designed for testing correctness, not performance benchmarking. Use production databases for performance testing.

## Resetting Mock Data

### Option 1: Using the Reset Script (Recommended)

Use the provided reset script to quickly recreate databases with sample data:

```bash
chmod +x reset_dev_db.sh  # First time only
./reset_dev_db.sh
```

This script will:
- Remove existing `dev_firebird.db` and `dev_mysql.db`
- Create fresh Firebird mock with 110 products from `dev_firebird_data.sql` (if present)
- Create MySQL mock with pre-existing data from `dev_mysql_data.sql` (if present)
- Show database statistics

### Option 2: Manual Reset

Manually recreate the databases:

```bash
# Remove old databases
rm -f dev_firebird.db dev_mysql.db

# Recreate Firebird mock from SQL file (if you have it)
sqlite3 dev_firebird.db < dev_firebird_data.sql

# Recreate MySQL mock from SQL file (if you have it)
sqlite3 dev_mysql.db < dev_mysql_data.sql

# OR create empty MySQL mock manually
sqlite3 dev_mysql.db "CREATE TABLE TB_ESTOQUE (
    ID_ESTOQUE INTEGER PRIMARY KEY,
    DESCRICAO TEXT NOT NULL,
    QTD_ATUAL REAL DEFAULT 0,
    PRC_CUSTO REAL DEFAULT 0,
    PRC_DOLAR REAL DEFAULT 0,
    PRC_VENDA REAL DEFAULT 0,
    PRC_3X REAL DEFAULT 0,
    PRC_6X REAL DEFAULT 0,
    PRC_10X REAL DEFAULT 0
);"
```

### Option 3: Let the Application Auto-Create Them

Simply delete the databases and run the application:

```bash
rm dev_firebird.db dev_mysql.db
go run .
```

**The application will automatically:**
- Look for `dev_firebird_data.sql` and load it if present (110 products)
- Otherwise, create minimal Firebird sample data (7 products)
- Look for `dev_mysql_data.sql` and load it if present (pre-existing data for testing)
- Otherwise, create empty MySQL table (all records will be inserted)

**Tip**: Keep both `dev_firebird_data.sql` and `dev_mysql_data.sql` in your project root for automatic full data loading and realistic testing scenarios.

## Switching to Production

When you're ready to use real Firebird and MySQL databases:

1. Set `DEV_MODE=false` in your `.env` file
2. Provide all required database credentials:
   - FIREBIRD_USER, FIREBIRD_PASSWORD, FIREBIRD_HOST, FIREBIRD_PATH
   - MYSQL_USER, MYSQL_PASSWORD, MYSQL_HOST, MYSQL_PORT, MYSQL_DATABASE

## Architecture

Development mode works by:
1. `config/config.go` - Reads `DEV_MODE` environment variable and skips credential validation
2. `db/db.go` - Routes `ConnectFirebird()` and `ConnectMySQL()` calls to dev functions when `cfg.DevMode=true`
3. `db/db_dev.go` - Contains SQLite connection functions and schema initialization
4. `processor/processor.go` - Skips MySQL stored procedures (`UpdateQtdVirtual`, `SP_ATUALIZAR_PART_NUMBER`) in dev mode since SQLite doesn't support them
5. The rest of the application code remains unchanged - it uses the same `*sql.DB` interfaces

**SQLite Limitations in Dev Mode:**
- No stored procedures support (auto-skipped)
- Serialized writes only (MaxOpenConns=1)
- Different SQL syntax for some advanced features

## Benefits

- **No External Dependencies**: Test without Firebird or MySQL servers
- **Fast Iteration**: Quickly test changes without network latency
- **Reproducible**: Same sample data on every reset
- **Portable**: Works on any system with Go and SQLite
- **Safe**: Can't accidentally modify production databases during development

## Troubleshooting

### "database is locked" errors

**Cause:** Multiple workers writing to SQLite simultaneously (older configurations)

**Solution:** Already fixed! The connection pool is limited to 1 connection with WAL mode enabled. If you still see this error:

```bash
# Delete databases and let them be recreated with proper settings
rm dev_firebird.db dev_mysql.db
go run .
```

### "unknown driver sqlite3" error

**Cause:** Wrong driver name for `modernc.org/sqlite`

**Solution:** Use driver name `"sqlite"` (not `"sqlite3"`). This is already configured correctly in `db/db_dev.go`.

### Slow sync performance in dev mode

**Expected behavior:** SQLite dev mode is slower than production because:
- Writes are serialized (MaxOpenConns=1)
- SQLite has different performance characteristics than Firebird/MySQL

**Not an issue:** Dev mode is for testing logic, not performance. Use production databases for benchmarking.

### SQL file not loading automatically

**Symptoms:** Only 7 products created instead of 110

**Solution:** Ensure `dev_firebird_data.sql` is in the project root directory (same location as `main.go`)

```bash
ls -la dev_firebird_data.sql  # Should exist
```

### "SQL logic error: near CALL" or stored procedure errors

**Cause:** SQLite doesn't support stored procedures

**Solution:** Already fixed! The code automatically skips MySQL stored procedures (`UpdateQtdVirtual`, `SP_ATUALIZAR_PART_NUMBER`) when `DEV_MODE=true`. You should see an info log message:
```
DEV_MODE: Skipping MySQL stored procedures - not supported in SQLite
```

If you still see this error, ensure you're using the latest version of the code.
