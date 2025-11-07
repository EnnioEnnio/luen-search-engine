# Profiling Guide

This document explains how to collect and inspect CPU, memory, and trace profiles for `luen-search-engine`, both from the interactive CLI and from the deterministic benchmark harness.

> **Prerequisites**
>
> - `go` 1.25+
> - `data/msmarco-docs.tsv` as the primary corpus
> - Benchmark assets:
>   - `data/msmarco-docs-bench-100000.tsv` – first 100k rows of the dataset (`head -n 100000 data/msmarco-docs.tsv > data/msmarco-docs-bench-100000.tsv`)
>   - `data/msmarco-queries-bench-1000.txt` – generated via `scripts/generate-bench-queries.sh`

---

## 1. Runtime profiling via CLI

The CLI exposes two flags in `main.go` that wire `runtime/pprof` into the interactive session:

| Flag | Argument | Purpose |
| ---- | -------- | ------- |
| `-cpuprofile <path>` | Path to write sampled CPU stacks (e.g., `cpu.pprof`) | Captures where time is spent during the _entire_ process lifetime (from startup until exit). |
| `-memprofile <path>` | Path to write a heap snapshot (e.g., `mem.pprof`) | Captures allocation information at the moment the program exits (after a forced GC). |

Example run that loads 100k documents, executes a few queries, and emits both profiles:

```bash
go run . -limit 100000 \
  -cpuprofile cpu.pprof \
  -memprofile mem.pprof
```

After the session, inspect the files with `go tool pprof`:

```bash
# CLI-driven CPU profile
go tool pprof cpu.pprof
pprof> top           # hottest functions
pprof> list Search   # annotated source
pprof> web           # opens flame graph in browser
```

```bash
# Heap allocations
go tool pprof -alloc_space mem.pprof
```

> **Tip:** `go tool pprof -http=:8080 cpu.pprof` serves an interactive flame graph UI that behaves similarly to Perfetto’s call charts.

### Collecting traces (Perfetto-style timelines)

Though not exposed via a dedicated flag, you can wrap any CLI run with Go’s execution tracer for a timeline view:

```bash
GODEBUG=tracebackancestors=1 go test -run=^$ -bench=^$ -trace trace.out ./cmd/benchqueries   # placeholder command
```

However, traces are usually more informative when captured from benchmarks (see §2). Once you have a `trace.out`, open it with:

```bash
go tool trace trace.out
```

or upload the file to https://ui.perfetto.dev/ for a Perfetto UI experience.

---

## 2. Deterministic benchmark profiling

Use the provided Make target to exercise both index building and query execution under a controlled workload:

```bash
make bench
```

This invokes:

- `BenchmarkBuildMSMarco100k` – builds the inverted index for the 100k-row subset, reporting CPU time and heap usage.
- `BenchmarkSearchWorkload` – executes 1,000 pre-generated queries (stored in `data/msmarco-queries-bench-1000.txt`) against the index.

Both benchmarks emit `ns/op`, `B/op`, and `allocs/op`. To capture profiles during these runs, append the usual Go test flags:

```bash
go test ./internal/index ./internal/search \
  -run=^$ -bench=. -benchmem \
  -cpuprofile bench_cpu.pprof \
  -memprofile bench_mem.pprof
```

For targeted runs:

```bash
go test ./internal/search \
  -run=^$ -bench=BenchmarkSearchWorkload \
  -cpuprofile search_cpu.pprof \
  -memprofile search_mem.pprof \
  -trace search_trace.out
```

Now:

- `go tool pprof bench_cpu.pprof` compares benchmark regressions to runtime profiles.
- `go tool trace search_trace.out` opens a Chrome/Perfetto-style timeline showing goroutines, syscalls, GC pauses, etc.

---

## 3. Workflow summary

1. **Interactive diagnosis** – run the CLI with `-cpuprofile` / `-memprofile` to capture real usage, then inspect with `go tool pprof`.
2. **Regression tracking** – run `make bench` (or the nightly GitHub Action) to benchmark the canonical workloads; add `-cpuprofile` / `-memprofile` / `-trace` when deeper analysis is needed.
3. **Perfetto-like visualization** – use `go tool trace trace.out` locally or upload the trace to https://ui.perfetto.dev for a rich timeline.

Keep the generated `.pprof` / `.out` artifacts alongside your experiments so you can compare builds, share findings, or attach them to issues during performance investigations.
