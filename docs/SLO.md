# TradeMind Service Level Objectives (SLO)

This document captures the conservative performance thresholds used by the P7-V2 harness.
These values are fixed for Phase P7 closure and must not be lowered to pass verification.

## API SLO

| Metric | Target |
| --- | --- |
| Overall `http_req_failed` | < 1% |
| Overall 5xx | < 0.2% |
| Overall timeout | < 0.2% |

## Core Read List Endpoints

Applies to product, order, inventory, task, webhook event, and operation log list APIs.

| Metric | Target |
| --- | --- |
| p95 latency | <= 800 ms |
| p99 latency | <= 1500 ms |

## Internal Lightweight Endpoints

| Metric | Target |
| --- | --- |
| p95 latency | <= 500 ms |
| p99 latency | <= 1000 ms |

## Controlled Write Endpoints

| Metric | Target |
| --- | --- |
| p95 latency | <= 1200 ms |
| p99 latency | <= 2500 ms |

## Performance Regression

When comparing Initial Controlled Baseline vs Independent Current Load Run:

| Metric | Allowed Degradation |
| --- | --- |
| p95 | <= 10% |
| p99 | <= 15% |
| throughput | <= 10% |
| error rate increase | <= 0.2 percentage points |
| timeout increase | <= 0.1 percentage points |
| peak RSS increase | <= 15% |
| heap growth increase | <= 15% |
| DB pool wait duration increase | <= 20% |
| queue peak depth increase | <= 20% |

Absolute SLO thresholds above still apply during regression comparison.

## Regression Materiality

The P7-V2 regression policy preserves the relative thresholds above. For latency
metrics, a relative degradation fails the regression gate only when it also
exceeds the applicable absolute materiality floor; an absolute SLO failure
always fails.

| Metric | Minimum absolute degradation |
| --- | ---: |
| p50 latency | 1 ms |
| p90 latency | 2 ms |
| p95 latency | 2 ms |
| p99 latency | 3 ms |

`max` is recorded for investigation and is not a primary materiality-gated
regression metric. These thresholds are retained as engineering guidance only;
the phase-specific comparison policy and harness were removed during production
maintenance. Any future CI regression gate must select a versioned policy before
measurement and must not change it after results are known.

## Soak Test

| Metric | Target |
| --- | --- |
| steady window | >= 30 continuous minutes |
| goroutine end | <= goroutine start + 10% |
| heap end | <= steady median heap + 15% |
| connection growth | no sustained growth |
| queue growth | no sustained growth |
| cache entries | bounded |
| limiter registry entries | bounded |

## Source

- Runtime metrics and alert configuration are the current operational source.
- Historical load harnesses and phase regression gates are no longer kept in the production-maintenance working tree.
- Capacity changes require an externally planned benchmark and human approval; they must not create persistent local evidence directories.
