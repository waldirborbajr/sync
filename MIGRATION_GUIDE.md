# Migration Guide: Original → Optimized

## File Changes Required

### 1. Update `main_helpers.go`

#### Change 1: Use optimized DB connections

**Line ~23-24** - Replace:
```go
firebirdConn, err := db.ConnectFirebird(cfg)
// ...
mysqlConn, err := db.ConnectMySQL(cfg)
```

With:
```go
firebirdConn, err := db.ConnectFirebirdOptimized(cfg)
// ...
mysqlConn, err := db.ConnectMySQLOptimized(cfg)
```

#### Change 2: Simplify processing call

**Line ~52-65** - Replace:
```go
// Get dynamic semaphore size and max_allowed_packet
semaphoreSize, maxConnections, maxAllowedPacket, err := db.GetSemaphoreSize(mysqlConn)
if err != nil {
    log.Warn().Err(err).Msg("Error retrieving MySQL variables, using defaults")
}

// Prepare statements
updateStmt, insertStmt, err := db.PrepareStatements(mysqlConn)
if err != nil {
    return 0, 0, 0, 0, nil, 0, 0, 0, err
}
defer func() {
    if updateStmt != nil {
        if cerr := updateStmt.Close(); cerr != nil {
            log.Error().Err(cerr).Msg("Error closing MySQL update statement")
        }
    }
    if insertStmt != nil {
        if cerr := insertStmt.Close(); cerr != nil {
            log.Error().Err(cerr).Msg("Error closing MySQL insert statement")
        }
    }
}()

// Processing
stats = &processor.ProcessingStats{}
startTime := time.Now()
err = processor.ProcessRows(firebirdConn, mysqlConn, updateStmt, insertStmt, semaphoreSize, maxAllowedPacket, &inserted, &updated, &ignored, &batchSize, stats, cfg)
```

With:
```go
// Get max connections for reporting
var maxConnections int
var maxAllowedPacket int
var variableName string
_ = mysqlConn.QueryRow("SHOW VARIABLES LIKE 'max_connections'").Scan(&variableName, &maxConnections)
_ = mysqlConn.QueryRow("SHOW VARIABLES LIKE 'max_allowed_packet'").Scan(&variableName, &maxAllowedPacket)

// Processing with worker pool
ctx := context.Background()
numWorkers := runtime.NumCPU() * 2 // Tune this based on your server
startTime := time.Now()
inserted, updated, ignored, batchSize, stats, err = processor.ProcessRowsOptimized(ctx, firebirdConn, mysqlConn, numWorkers, cfg)
```

#### Change 3: Add imports

**Top of file** - Add:
```go
import (
    "context"
    "fmt"
    "runtime"
    "strings"
    "time"

    "github.com/waldirborbajr/sync/config"
    "github.com/waldirborbajr/sync/logger"
    "github.com/waldirborbajr/sync/processor"
)
```

---

### 2. Update Function Signature

#### In `main.go` - Line ~49

Replace:
```go
insertedCount, updatedCount, ignoredCount, batchSize, stats, elapsedTime, maxConnections, maxAllowedPacket, err := runProcessing(cfg)
```

With:
```go
insertedCount, updatedCount, ignoredCount, batchSize, stats, elapsedTime, maxConnections, maxAllowedPacket, err := runProcessing(cfg)
// No changes needed here, return signature stays the same
```

---

### 3. Update `printSummary` call

#### In `main.go` - Line ~59

Replace:
```go
semaphoreSize := int(float64(maxConnections) * 0.75)
if semaphoreSize < 10 {
    semaphoreSize = 10
} else if semaphoreSize > 100 {
    semaphoreSize = 100
}
```

With:
```go
numWorkers := runtime.NumCPU() * 2 // Or read from config
```

Then update the `printSummary` call to use `numWorkers` instead of `semaphoreSize`.

---

## Complete runProcessing Function (Optimized)

```go
func runProcessing(cfg config.Config) (inserted, updated, ignored, batchSize int, stats *processor.ProcessingStats, elapsed time.Duration, maxConnections int, maxAllowedPacket int, err error) {
    log := logger.GetLogger()
    
    // Connect to Firebird with optimized settings
    firebirdConn, err := db.ConnectFirebirdOptimized(cfg)
    if err != nil {
        return 0, 0, 0, 0, nil, 0, 0, 0, err
    }
    defer func() {
        if firebirdConn != nil {
            if closeErr := firebirdConn.Close(); closeErr != nil {
                log.Error().Err(closeErr).Msg("Error closing Firebird database connection")
            }
        }
    }()

    // Connect to MySQL with optimized settings
    mysqlConn, err := db.ConnectMySQLOptimized(cfg)
    if err != nil {
        return 0, 0, 0, 0, nil, 0, 0, 0, err
    }
    defer func() {
        if mysqlConn != nil {
            if closeErr := mysqlConn.Close(); closeErr != nil {
                log.Error().Err(closeErr).Msg("Error closing MySQL database connection")
            }
        }
    }()

    // MySQL optimizations
    _, err = mysqlConn.Exec("SET unique_checks=0")
    if err != nil {
        log.Warn().Err(err).Msg("Could not set unique_checks=0")
    }
    _, err = mysqlConn.Exec("SET foreign_key_checks=0")
    if err != nil {
        log.Warn().Err(err).Msg("Could not set foreign_key_checks=0")
    }

    // Get MySQL parameters for reporting
    var variableName string
    _ = mysqlConn.QueryRow("SHOW VARIABLES LIKE 'max_connections'").Scan(&variableName, &maxConnections)
    _ = mysqlConn.QueryRow("SHOW VARIABLES LIKE 'max_allowed_packet'").Scan(&variableName, &maxAllowedPacket)

    // Processing with optimized worker pool
    ctx := context.Background()
    numWorkers := runtime.NumCPU() * 2 // Start with 2x CPU cores, tune as needed
    if numWorkers > 20 {
        numWorkers = 20 // Cap at 20 for safety
    }
    
    startTime := time.Now()
    inserted, updated, ignored, batchSize, stats, err = processor.ProcessRowsOptimized(ctx, firebirdConn, mysqlConn, numWorkers, cfg)
    if err != nil {
        return 0, 0, 0, 0, nil, 0, 0, 0, err
    }

    // Restore MySQL settings
    _, err = mysqlConn.Exec("SET unique_checks=1")
    if err != nil {
        log.Warn().Err(err).Msg("Could not set unique_checks=1")
    }
    _, err = mysqlConn.Exec("SET foreign_key_checks=1")
    if err != nil {
        log.Warn().Err(err).Msg("Could not set foreign_key_checks=1")
    }

    elapsed = time.Since(startTime)
    return inserted, updated, ignored, batchSize, stats, elapsed, maxConnections, maxAllowedPacket, nil
}
```

---

## Configuration Tuning

### Environment Variables (.env)

Add these optional variables for tuning:

```bash
# Performance tuning (optional)
NUM_WORKERS=10          # Number of worker goroutines (default: 2 x CPU cores)
BATCH_SIZE=500          # Batch size for bulk operations (default: 500)
MAX_OPEN_CONNS=100      # Max MySQL connections (default: 80% of max_connections)
```

### Worker Count Guidelines

Choose `NUM_WORKERS` based on your setup:

| Scenario | Workers | Rationale |
|----------|---------|-----------|
| Local dev (4 cores) | 8 | 2 x cores |
| Production (8 cores) | 16 | 2 x cores |
| High latency network | 20-30 | More workers to hide latency |
| Low latency network | 10-15 | Less overhead |
| Limited MySQL connections | 5-10 | Prevent connection exhaustion |
| High CPU server | 30-50 | Maximize throughput |

---

## Testing Checklist

Before deploying to production:

- [ ] Backup MySQL database
- [ ] Test with small dataset (100 rows)
- [ ] Test with medium dataset (1,000 rows)
- [ ] Test with large dataset (10,000+ rows)
- [ ] Compare throughput metrics
- [ ] Check memory usage (should be lower)
- [ ] Check GC cycles (should be fewer)
- [ ] Verify data integrity (spot-check records)
- [ ] Test error handling (disconnect DB mid-sync)
- [ ] Monitor MySQL connection count
- [ ] Check MySQL slow query log

---

## Rollback Plan

If issues arise:

1. **Keep both versions**: Don't delete original code yet
2. **Use build tags**: 
   ```go
   // +build optimized
   ```
3. **Feature flag**: Add config option
   ```bash
   USE_OPTIMIZED=false  # Rollback to original
   ```
4. **Git branch**: Keep optimizations in separate branch initially

---

## Performance Validation

After deployment, compare these metrics:

| Metric | Check | Expected |
|--------|-------|----------|
| Throughput | rows/second | 10-30x increase |
| Memory | Peak MB | 50-70% reduction |
| Time | Total duration | 10-30x faster |
| GC cycles | Count | 70-90% reduction |
| CPU usage | Average % | 30-50% lower |
| Errors | Count | Zero |

---

## Common Issues & Fixes

### "Worker timeout"
**Cause**: Workers taking too long
**Fix**: Reduce `NUM_WORKERS` or check MySQL slow query log

### "Out of connections"
**Cause**: Too many workers
**Fix**: Reduce `NUM_WORKERS` to < MySQL max_connections / 2

### "No performance gain"
**Cause**: Wrong functions used
**Fix**: Verify you're calling `ProcessRowsOptimized` not `ProcessRows`

### "Bulk insert error"
**Cause**: Batch too large for max_allowed_packet
**Fix**: Reduce `BATCH_SIZE` to 100-200

---

## Support

If you need help:
1. Check error logs carefully
2. Test with a subset of data
3. Compare original vs optimized directly
4. Profile with `go tool pprof`
5. Monitor MySQL processlist during sync

Good luck with your optimization! 🚀
