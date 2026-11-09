# Misma idea, otros lenguajes

Generado por `benchmarks/run_otros_lenguajes.sh` (2026-11-08, hyperfine, 1 corrida de calentamiento).
Versiones: Python 3.15.2, ruby 2.6.10p210, Node v24.21.0, Perl v5.34.1.

## fib

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `glox (VM)` | 39.8 ± 0.3 | 39.0 | 40.7 | 1.75 ± 0.06 |
| `glox-parcial (tree-walk)` | 145.7 ± 2.9 | 142.1 | 155.1 | 6.41 ± 0.26 |
| `plox` | 8453.3 ± 41.5 | 8413.0 | 8514.6 | 372.14 ± 13.09 |
| `Python` | 28.2 ± 0.8 | 27.0 | 33.7 | 1.24 ± 0.06 |
| `Ruby` | 52.3 ± 0.5 | 51.2 | 54.4 | 2.30 ± 0.08 |
| `JavaScript (Node)` | 22.7 ± 0.8 | 21.5 | 26.6 | 1.00 |
| `Perl` | 74.4 ± 0.5 | 73.4 | 75.7 | 3.28 ± 0.12 |

## loops

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `glox (VM)` | 69.7 ± 0.4 | 69.2 | 71.2 | 3.02 ± 0.09 |
| `glox-parcial (tree-walk)` | 271.0 ± 1.9 | 267.9 | 275.1 | 11.71 ± 0.36 |
| `plox` | 20595.7 ± 128.7 | 20476.9 | 20781.4 | 890.33 ± 27.34 |
| `Python` | 134.7 ± 7.0 | 127.2 | 152.6 | 5.82 ± 0.35 |
| `Ruby` | 60.3 ± 0.5 | 59.4 | 61.4 | 2.61 ± 0.08 |
| `JavaScript (Node)` | 23.1 ± 0.7 | 21.8 | 25.1 | 1.00 |
| `Perl` | 40.7 ± 1.0 | 38.6 | 44.7 | 1.76 ± 0.07 |
