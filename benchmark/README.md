# GoLiteCron Benchmark Suite

This directory contains comprehensive benchmarks for GoLiteCron, including comparisons with [robfig/cron](https://github.com/robfig/cron).

## Quick Start

```bash
# Run all benchmarks
./scripts/run_benchmark.sh -a

# Run specific benchmark categories
./scripts/run_benchmark.sh -p    # Parser benchmarks
./scripts/run_benchmark.sh -s    # Storage benchmarks  
./scripts/run_benchmark.sh -c    # Comparison with robfig/cron
./scripts/run_benchmark.sh -m    # Memory benchmarks

# Compare with baseline
./scripts/run_benchmark.sh -a -b  # Run and save as baseline
./scripts/run_benchmark.sh -a -d  # Run and compare with baseline
```

## Benchmark Categories

### 1. CronParser Benchmarks (`cron_parser_bench_test.go`)

Tests the cron expression parsing and `Next()` calculation performance.

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkParse_*` | Expression parsing with varying complexity |
| `BenchmarkNext_*` | Next time calculation performance |
| `BenchmarkNext_Sequential_*` | Sequential Next() calls (simulates real scheduling) |
| `BenchmarkNext_Parallel` | Thread-safe concurrent Next() calls |

### 2. Storage Benchmarks (`storage_bench_test.go`)

Compares Heap vs TimeWheel storage backends.

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkHeap_*` | Heap-based priority queue operations |
| `BenchmarkTimeWheel_*` | Multi-level time wheel operations |
| `BenchmarkComparison_*` | Direct Heap vs TimeWheel comparison |

### 3. Comparison Benchmarks (`comparison_bench_test.go`)

Head-to-head comparison with `robfig/cron`.

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkComparison_Parse_*` | Expression parsing comparison |
| `BenchmarkComparison_Next_*` | Next() calculation comparison |
| `BenchmarkComparison_AddTasks*` | Bulk task addition comparison |
| `BenchmarkComparison_StartStop_*` | Scheduler lifecycle comparison |

### 4. Memory Benchmarks (`memory_bench_test.go`)

Memory allocation and leak detection tests.

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkMemory_*` | Memory allocation patterns |
| `BenchmarkMemoryOverhead_*` | Per-task memory overhead |
| `TestMemory_LeakDetection_*` | Memory leak detection tests |

## Running Benchmarks

### Basic Usage

```bash
# Run all benchmarks with default settings (6 iterations)
go test -bench=. -benchmem ./benchmark/...

# Run specific benchmark pattern
go test -bench=BenchmarkNext -benchmem ./benchmark/...

# Run with more iterations for statistical significance
go test -bench=. -benchmem -count=10 ./benchmark/...
```

### Using benchstat for Comparison

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks and save results
go test -bench=. -benchmem -count=6 ./benchmark/... > old.txt

# Make changes, then run again
go test -bench=. -benchmem -count=6 ./benchmark/... > new.txt

# Compare results
benchstat old.txt new.txt
```

### Profiling

```bash
# CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./benchmark/...
go tool pprof cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof ./benchmark/...
go tool pprof mem.prof

# Or use the script
./scripts/run_benchmark.sh --cpu-profile -a
./scripts/run_benchmark.sh --mem-profile -a
```

## Interpreting Results

### Benchmark Output Format

```
BenchmarkNext_Simple-8    1000000    1050 ns/op    0 B/op    0 allocs/op
```

- `BenchmarkNext_Simple-8`: Benchmark name, `-8` indicates GOMAXPROCS
- `1000000`: Number of iterations
- `1050 ns/op`: Nanoseconds per operation
- `0 B/op`: Bytes allocated per operation
- `0 allocs/op`: Allocations per operation

### benchstat Output

```
name                old time/op    new time/op    delta
Next_Simple-8       1.05µs ± 2%    0.98µs ± 1%   -6.67%  (p=0.002 n=6+6)
```

- `old time/op`: Previous benchmark result
- `new time/op`: Current benchmark result  
- `delta`: Percentage change (negative = improvement)
- `p=0.002`: Statistical significance (p < 0.05 is significant)
- `n=6+6`: Number of samples used

## Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| Parse (simple) | < 5µs | 5-field standard cron |
| Parse (complex) | < 10µs | With ranges and steps |
| Next() | < 500ns | Per calculation |
| AddTask | < 2µs | Single task addition |
| Tick (1000 tasks) | < 100µs | 0 tasks ready |

## Adding New Benchmarks

1. Create benchmark function with `Benchmark` prefix
2. Use `b.ReportAllocs()` for memory tracking
3. Use `b.ResetTimer()` after setup code
4. For parallel benchmarks, use `b.RunParallel()`

Example:
```go
func BenchmarkMyFeature(b *testing.B) {
    // Setup (not measured)
    data := setupData()
    
    b.ReportAllocs()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        // Code to benchmark
        _ = myFeature(data)
    }
}
```

## CI Integration

Add to `.github/workflows/benchmark.yml`:

```yaml
name: Benchmark

on:
  push:
    branches: [master]
  pull_request:
    branches: [master]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run Benchmarks
        run: go test -bench=. -benchmem -count=6 ./benchmark/...
```

## Results Directory

Benchmark results are saved to `.benchmark_results/`:

```
.benchmark_results/
├── all_current.txt        # Latest full benchmark run
├── all_baseline.txt       # Saved baseline for comparison
├── parser_current.txt     # Latest parser benchmarks
├── storage_current.txt    # Latest storage benchmarks
├── comparison_current.txt # Latest comparison benchmarks
├── memory_current.txt     # Latest memory benchmarks
├── cpu.prof               # CPU profile (if generated)
└── mem.prof               # Memory profile (if generated)
```
