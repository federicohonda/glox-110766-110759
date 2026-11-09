# Resultados de benchmarks

Generado por `benchmarks/plot.py` a partir de `results/results.json`. No editar a mano.

- **Máquina:** Apple M5, 16 GB de RAM, macOS-26.6.2-arm64-arm-64bit
- **Fecha:** 2026-09-26
- **Métrica:** mediana del tiempo total del proceso (arranque + scan + parse + resolve + ejecución).

## Tiempo (mediana)

| Implementación | Lenguaje | Estrategia | `startup` | `fib` | `loops` | `closures` | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| rlox | Rust | bytecode | 5.5 ms | 8.0 ms | 5.9 ms | 5.0 ms | 20 ms | 31 ms | 41 ms | 25 ms | — |
| loxx | C++ | bytecode | 1.8 ms | 6.6 ms | 3.3 ms | 2.9 ms | 23 ms | 30 ms | 48 ms | 26 ms | 49 ms |
| jlox | Java | tree-walk | 22 ms | 56 ms | 44 ms | 42 ms | 95 ms | 159 ms | 382 ms | 222 ms | 127 ms |
| **glox** | Go | tree-walk | 2.2 ms | 36 ms | 17 ms | 6.7 ms | 147 ms | 273 ms | 416 ms | 194 ms | 185 ms |
| slox | Swift | tree-walk | 2.8 ms | 180 ms | 67 ms | 25 ms | 762 ms | 1.30 s | 2.49 s | 1.12 s | 1.02 s |
| dlox | Dart | tree-walk | 10 ms | 2.04 s | 24 ms | 53 ms | 9.73 s | 304 ms | 4.78 s | 2.21 s | 237 ms |
| plox-php | PHP | tree-walk | 30 ms | 2.34 s | 201 ms | 101 ms | 10.8 s | 3.36 s | 7.66 s | 4.05 s | 2.31 s |
| cloxure | Clojure | tree-walk | 199 ms | 10.6 s | 354 ms | 484 ms | 48.9 s | 1.96 s | 24.5 s | 11.4 s | 1.28 s |
| plox | Python | tree-walk | 53 ms | 2.06 s | 1.10 s | 266 ms | 8.56 s | 20.5 s | 22.9 s | 15.0 s | 14.3 s |

## Tiempo relativo a glox (> 1 = más lenta que glox)

| Implementación | `startup` | `fib` | `loops` | `closures` | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` | Media geom. (pesados) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| rlox | 2.5× | 0.22× | 0.34× | 0.75× | 0.14× | 0.11× | 0.10× | 0.13× | — | 0.12× |
| loxx | 0.80× | 0.18× | 0.19× | 0.44× | 0.16× | 0.11× | 0.11× | 0.13× | 0.27× | 0.15× |
| jlox | 10.0× | 1.6× | 2.6× | 6.3× | 0.65× | 0.58× | 0.92× | 1.1× | 0.69× | 0.77× |
| **glox** | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× |
| slox | 1.2× | 5.0× | 4.0× | 3.8× | 5.2× | 4.8× | 6.0× | 5.8× | 5.5× | 5.4× |
| dlox | 4.6× | 56× | 1.4× | 7.9× | 66× | 1.1× | 11× | 11× | 1.3× | 6.6× |
| plox-php | 14× | 65× | 12× | 15× | 73× | 12× | 18× | 21× | 13× | 21× |
| cloxure | 89× | 292× | 21× | 72× | 333× | 7.2× | 59× | 59× | 6.9× | 36× |
| plox | 23× | 57× | 64× | 40× | 58× | 75× | 55× | 77× | 78× | 68× |

## Memoria residente pico (MB)

| Implementación | `startup` | `fib` | `loops` | `closures` | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| rlox | 49.6 | 49.7 | 49.7 | 50.7 | 49.7 | 49.7 | 49.7 | 49.7 | — |
| loxx | 1.8 | 1.8 | 1.8 | 3.1 | 1.8 | 1.8 | 1.8 | 1.8 | 1.8 |
| jlox | 42.0 | 75.5 | 65.3 | 55.9 | 128.5 | 124.5 | 128.3 | 123.0 | 124.4 |
| **glox** | 4.3 | 12.7 | 10.0 | 10.2 | 12.8 | 11.4 | 11.1 | 11.0 | 11.1 |
| slox | 5.8 | 5.9 | 5.9 | 9.5 | 5.9 | 6.0 | 6.1 | 5.9 | 6.0 |
| dlox | 13.9 | 18.3 | 17.5 | 17.6 | 18.8 | 17.6 | 17.6 | 17.6 | 17.6 |
| plox-php | 25.8 | 26.1 | 25.9 | 28.6 | 26.0 | 25.8 | 25.9 | 25.9 | 25.8 |
| cloxure | 101.7 | 342.2 | 258.6 | 244.0 | 333.2 | 828.3 | 356.3 | 340.3 | 828.6 |
| plox | 32.1 | 32.0 | 31.9 | 32.2 | 32.2 | 31.9 | 31.9 | 32.1 | 32.1 |

## CPU (user + sys, mediana) sobre tiempo de reloj

Un valor mayor a 1 indica que el proceso usó más de un núcleo (JIT, GC o compilación en paralelo).

| Implementación | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` |
|---|---:|---:|---:|---:|---:|
| rlox | 0.96 | 0.97 | 0.98 | 0.97 | — |
| loxx | 0.96 | 0.97 | 0.98 | 0.97 | 0.98 |
| jlox | 1.83 | 1.43 | 1.28 | 1.35 | 1.56 |
| glox | 1.11 | 1.03 | 1.09 | 1.04 | 1.04 |
| slox | 1.00 | 1.00 | 1.00 | 1.00 | 1.00 |
| dlox | 1.00 | 0.99 | 1.00 | 1.00 | 0.99 |
| plox-php | 1.00 | 0.99 | 1.00 | 0.99 | 0.99 |
| cloxure | 1.02 | 1.35 | 1.03 | 1.07 | 1.58 |
| plox | 0.99 | 1.00 | 1.00 | 1.00 | 1.00 |

## Escalado: fib(n)

| Implementación | n=15 | n=18 | n=21 | n=24 | n=27 |
|---|---:|---:|---:|---:|---:|
| rlox | 4.2 ms | 4.3 ms | 4.8 ms | 7.0 ms | 14 ms |
| loxx | 1.8 ms | 2.0 ms | 2.6 ms | 4.9 ms | 15 ms |
| jlox | 29 ms | 34 ms | 45 ms | 55 ms | 78 ms |
| glox | 2.5 ms | 3.9 ms | 7.9 ms | 25 ms | 92 ms |
| slox | 4.5 ms | 9.3 ms | 30 ms | 115 ms | 463 ms |
| dlox | 20 ms | 63 ms | 273 ms | 1.26 s | 5.80 s |
| plox-php | 44 ms | 93 ms | 322 ms | 1.42 s | 6.41 s |
| cloxure | 290 ms | 518 ms | 1.54 s | 6.38 s | 29.0 s |
| plox | 75 ms | 149 ms | 382 ms | 1.31 s | 5.30 s |

## Corridas inválidas (no aparecen en tablas ni gráficos)

- `strings` con rlox: código de salida 101

## Notas de validación

- `loops_grande` con loxx: imprime con 6 cifras significativas (2e+12)
- `calls` con loxx: imprime con 6 cifras significativas (5.00000e+11)

Corridas medidas por combinación: entre 3 y 5 (ver `run_benchmarks.py --help`).
