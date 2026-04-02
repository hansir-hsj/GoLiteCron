#!/bin/bash
# GoLiteCron Benchmark Suite Runner
# Usage: ./scripts/run_benchmark.sh [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
RESULT_DIR="$PROJECT_DIR/.benchmark_results"
BENCH_DIR="$PROJECT_DIR/benchmark"
COUNT=6  # Number of benchmark runs for statistical significance

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    echo "GoLiteCron Benchmark Suite"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -a, --all          Run all benchmarks"
    echo "  -p, --parser       Run CronParser benchmarks only"
    echo "  -s, --storage      Run Storage benchmarks only"
    echo "  -c, --comparison   Run comparison benchmarks (vs robfig/cron)"
    echo "  -m, --memory       Run memory benchmarks only"
    echo "  -b, --baseline     Save current results as baseline"
    echo "  -d, --diff         Compare current results with baseline"
    echo "  -n, --count NUM    Number of benchmark runs (default: 6)"
    echo "  --cpu-profile      Generate CPU profile"
    echo "  --mem-profile      Generate memory profile"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 -a                    # Run all benchmarks"
    echo "  $0 -c -n 10              # Run comparison benchmarks 10 times"
    echo "  $0 -a -b                 # Run all and save as baseline"
    echo "  $0 -a -d                 # Run all and compare with baseline"
    echo "  $0 --cpu-profile -p      # Run parser benchmarks with CPU profiling"
}

run_parser_benchmarks() {
    echo -e "${BLUE}=== Running CronParser Benchmarks ===${NC}"
    go test -bench="Benchmark(Parse|Next)" -benchmem -count=$COUNT "$BENCH_DIR/..." 2>/dev/null | \
        grep -v "^goos\|^goarch\|^pkg\|^cpu\|^PASS\|^ok" | tee "$RESULT_DIR/parser_current.txt"
}

run_storage_benchmarks() {
    echo -e "${BLUE}=== Running Storage Benchmarks ===${NC}"
    go test -bench="Benchmark(Heap|TimeWheel|Comparison_(Add|Tick))" -benchmem -count=$COUNT "$BENCH_DIR/..." 2>/dev/null | \
        grep -v "^goos\|^goarch\|^pkg\|^cpu\|^PASS\|^ok" | tee "$RESULT_DIR/storage_current.txt"
}

run_comparison_benchmarks() {
    echo -e "${BLUE}=== Running Comparison Benchmarks (GoLiteCron vs robfig/cron) ===${NC}"
    go test -bench="BenchmarkComparison_" -benchmem -count=$COUNT "$BENCH_DIR/..." 2>/dev/null | \
        grep -v "^goos\|^goarch\|^pkg\|^cpu\|^PASS\|^ok" | tee "$RESULT_DIR/comparison_current.txt"
}

run_memory_benchmarks() {
    echo -e "${BLUE}=== Running Memory Benchmarks ===${NC}"
    go test -bench="BenchmarkMemory" -benchmem -count=$COUNT "$BENCH_DIR/..." 2>/dev/null | \
        grep -v "^goos\|^goarch\|^pkg\|^cpu\|^PASS\|^ok" | tee "$RESULT_DIR/memory_current.txt"
}

run_all_benchmarks() {
    echo -e "${GREEN}=== GoLiteCron Full Benchmark Suite ===${NC}"
    echo "Running with count=$COUNT"
    echo ""
    go test -bench=. -benchmem -count=$COUNT "$BENCH_DIR/..." 2>/dev/null | \
        tee "$RESULT_DIR/all_current.txt"
}

save_baseline() {
    echo -e "${YELLOW}Saving current results as baseline...${NC}"
    for f in "$RESULT_DIR"/*_current.txt; do
        if [ -f "$f" ]; then
            baseline="${f/_current.txt/_baseline.txt}"
            cp "$f" "$baseline"
            echo "  Saved: $(basename "$baseline")"
        fi
    done
    echo -e "${GREEN}Baseline saved!${NC}"
}

compare_with_baseline() {
    echo -e "${BLUE}=== Comparing with Baseline ===${NC}"
    
    # Check if benchstat is installed
    if ! command -v benchstat &> /dev/null; then
        echo -e "${YELLOW}Installing benchstat...${NC}"
        go install golang.org/x/perf/cmd/benchstat@latest
    fi
    
    for current in "$RESULT_DIR"/*_current.txt; do
        if [ -f "$current" ]; then
            baseline="${current/_current.txt/_baseline.txt}"
            if [ -f "$baseline" ]; then
                name=$(basename "$current" _current.txt)
                echo ""
                echo -e "${GREEN}=== $name comparison ===${NC}"
                benchstat "$baseline" "$current" 2>/dev/null || echo "  (no comparable data)"
            fi
        fi
    done
}

run_with_cpu_profile() {
    echo -e "${BLUE}=== Running with CPU Profile ===${NC}"
    go test -bench=. -cpuprofile="$RESULT_DIR/cpu.prof" "$BENCH_DIR/..." 2>/dev/null
    echo -e "${GREEN}CPU profile saved to: $RESULT_DIR/cpu.prof${NC}"
    echo "Analyze with: go tool pprof $RESULT_DIR/cpu.prof"
}

run_with_mem_profile() {
    echo -e "${BLUE}=== Running with Memory Profile ===${NC}"
    go test -bench=. -memprofile="$RESULT_DIR/mem.prof" "$BENCH_DIR/..." 2>/dev/null
    echo -e "${GREEN}Memory profile saved to: $RESULT_DIR/mem.prof${NC}"
    echo "Analyze with: go tool pprof $RESULT_DIR/mem.prof"
}

# Create result directory
mkdir -p "$RESULT_DIR"

# Parse arguments
RUN_ALL=false
RUN_PARSER=false
RUN_STORAGE=false
RUN_COMPARISON=false
RUN_MEMORY=false
SAVE_BASELINE=false
COMPARE_BASELINE=false
CPU_PROFILE=false
MEM_PROFILE=false

if [ $# -eq 0 ]; then
    usage
    exit 0
fi

while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--all)
            RUN_ALL=true
            shift
            ;;
        -p|--parser)
            RUN_PARSER=true
            shift
            ;;
        -s|--storage)
            RUN_STORAGE=true
            shift
            ;;
        -c|--comparison)
            RUN_COMPARISON=true
            shift
            ;;
        -m|--memory)
            RUN_MEMORY=true
            shift
            ;;
        -b|--baseline)
            SAVE_BASELINE=true
            shift
            ;;
        -d|--diff)
            COMPARE_BASELINE=true
            shift
            ;;
        -n|--count)
            COUNT="$2"
            shift 2
            ;;
        --cpu-profile)
            CPU_PROFILE=true
            shift
            ;;
        --mem-profile)
            MEM_PROFILE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Change to project directory
cd "$PROJECT_DIR"

# Run benchmarks
if [ "$CPU_PROFILE" = true ]; then
    run_with_cpu_profile
elif [ "$MEM_PROFILE" = true ]; then
    run_with_mem_profile
elif [ "$RUN_ALL" = true ]; then
    run_all_benchmarks
else
    if [ "$RUN_PARSER" = true ]; then
        run_parser_benchmarks
    fi
    if [ "$RUN_STORAGE" = true ]; then
        run_storage_benchmarks
    fi
    if [ "$RUN_COMPARISON" = true ]; then
        run_comparison_benchmarks
    fi
    if [ "$RUN_MEMORY" = true ]; then
        run_memory_benchmarks
    fi
fi

# Save baseline if requested
if [ "$SAVE_BASELINE" = true ]; then
    save_baseline
fi

# Compare with baseline if requested
if [ "$COMPARE_BASELINE" = true ]; then
    compare_with_baseline
fi

echo ""
echo -e "${GREEN}Done!${NC}"
echo "Results saved to: $RESULT_DIR/"
