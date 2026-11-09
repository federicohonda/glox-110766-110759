# Resultados de benchmarks

Generado por `benchmarks/plot.py` a partir de `results/results.json`. No editar a mano.

- **Máquina:** Apple M5, 16 GB de RAM, macOS-26.6.2-arm64-arm-64bit-Mach-O
- **Fecha:** 2026-11-08
- **Métrica:** mediana del tiempo total del proceso (arranque + scan + parse + resolve + ejecución).

## Tiempo (mediana)

| Implementación | Lenguaje | Estrategia | `startup` | `fib` | `loops` | `closures` | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` | `metodos` |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| rlox | Rust | bytecode | 4.2 ms | 7.9 ms | 5.7 ms | 4.8 ms | 20 ms | 30 ms | 41 ms | 25 ms | — | 34 ms |
| loxx | C++ | bytecode | 1.8 ms | 6.6 ms | 3.3 ms | 2.9 ms | 23 ms | 31 ms | 48 ms | 26 ms | 50 ms | 39 ms |
| **glox** | Go | bytecode | 2.1 ms | 12 ms | 5.9 ms | 3.4 ms | 41 ms | 70 ms | 84 ms | 50 ms | 66 ms | 91 ms |
| glox-vm-sin-opt | Go | bytecode | 2.0 ms | 12 ms | 6.9 ms | 3.3 ms | 41 ms | 93 ms | 98 ms | 64 ms | 95 ms | — |
| jlox | Java | tree-walk | 23 ms | 57 ms | 44 ms | 43 ms | 96 ms | 164 ms | 382 ms | 222 ms | 125 ms | 295 ms |
| glox-parcial | Go | tree-walk | 2.0 ms | 37 ms | 16 ms | 6.6 ms | 146 ms | 271 ms | 418 ms | 194 ms | 182 ms | — |
| slox | Swift | tree-walk | 2.8 ms | 178 ms | 69 ms | 25 ms | 746 ms | 1.29 s | 2.46 s | 1.10 s | 1.04 s | 1.81 s |
| dlox | Dart | tree-walk | 8.4 ms | 2.01 s | 24 ms | 53 ms | 9.77 s | 305 ms | 4.80 s | 2.22 s | 232 ms | 2.44 s |
| plox-php | PHP | tree-walk | 30 ms | 2.34 s | 199 ms | 100 ms | 10.8 s | 3.37 s | 7.69 s | 4.03 s | 2.31 s | — |
| cloxure | Clojure | tree-walk | 200 ms | 10.5 s | 355 ms | 490 ms | 48.7 s | 1.91 s | 24.8 s | 11.2 s | 1.26 s | 12.4 s |
| plox | Python | tree-walk | 53 ms | 2.05 s | 1.10 s | 265 ms | 8.53 s | 20.6 s | 22.9 s | 14.8 s | 14.3 s | — |

## Tiempo relativo a glox (> 1 = más lenta que glox)

| Implementación | `startup` | `fib` | `loops` | `closures` | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` | `metodos` | Media geom. (pesados) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| rlox | 2.0× | 0.67× | 0.96× | 1.4× | 0.49× | 0.43× | 0.48× | 0.50× | — | 0.37× | 0.47× |
| loxx | 0.86× | 0.56× | 0.55× | 0.87× | 0.56× | 0.44× | 0.57× | 0.51× | 0.75× | 0.43× | 0.56× |
| **glox** | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× | 1.0× |
| glox-vm-sin-opt | 0.95× | 1.0× | 1.2× | 1.00× | 1.0× | 1.3× | 1.2× | 1.3× | 1.4× | — | 1.2× |
| jlox | 11× | 4.9× | 7.5× | 13× | 2.4× | 2.3× | 4.5× | 4.4× | 1.9× | 3.2× | 2.9× |
| glox-parcial | 0.96× | 3.1× | 2.8× | 2.0× | 3.6× | 3.8× | 5.0× | 3.9× | 2.8× | — | 3.7× |
| slox | 1.3× | 15× | 12× | 7.4× | 18× | 18× | 29× | 22× | 16× | 20× | 20× |
| dlox | 3.9× | 171× | 4.0× | 16× | 240× | 4.3× | 57× | 44× | 3.5× | 27× | 25× |
| plox-php | 14× | 199× | 34× | 30× | 267× | 48× | 91× | 80× | 35× | — | 80× |
| cloxure | 94× | 897× | 60× | 146× | 1199× | 27× | 295× | 223× | 19× | 136× | 133× |
| plox | 25× | 174× | 186× | 79× | 210× | 293× | 271× | 296× | 216× | — | 254× |

## Memoria residente pico (MB)

| Implementación | `startup` | `fib` | `loops` | `closures` | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` | `metodos` |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| rlox | 49.6 | 49.7 | 49.7 | 50.7 | 49.7 | 49.7 | 49.7 | 49.7 | — | 50.7 |
| loxx | 1.8 | 1.8 | 1.8 | 3.1 | 1.8 | 1.8 | 1.8 | 1.8 | 1.8 | 16.2 |
| **glox** | 4.3 | 7.0 | 6.0 | 5.3 | 10.0 | 10.6 | 10.5 | 10.5 | 10.6 | 11.0 |
| glox-vm-sin-opt | 4.3 | 7.0 | 5.9 | 5.2 | 9.5 | 10.5 | 10.7 | 10.3 | 11.1 | — |
| jlox | 41.9 | 74.2 | 65.6 | 56.2 | 128.4 | 124.4 | 127.8 | 122.9 | 124.1 | 128.4 |
| glox-parcial | 4.4 | 10.9 | 9.8 | 9.9 | 13.2 | 10.8 | 10.9 | 11.1 | 10.6 | — |
| slox | 5.8 | 6.0 | 5.9 | 9.5 | 6.1 | 6.0 | 6.1 | 6.0 | 6.0 | 6.0 |
| dlox | 14.0 | 18.4 | 17.7 | 17.7 | 18.9 | 17.6 | 17.7 | 17.7 | 17.7 | 17.7 |
| plox-php | 25.8 | 26.0 | 25.9 | 28.6 | 26.1 | 25.9 | 25.8 | 25.8 | 25.9 | — |
| cloxure | 102.5 | 334.3 | 260.9 | 252.2 | 328.0 | 828.3 | 337.4 | 339.5 | 830.5 | 335.0 |
| plox | 31.9 | 32.0 | 31.9 | 31.9 | 32.0 | 32.0 | 31.9 | 31.9 | 31.8 | — |

## CPU (user + sys, mediana) sobre tiempo de reloj

Un valor mayor a 1 indica que el proceso usó más de un núcleo (JIT, GC o compilación en paralelo).

| Implementación | `fib_grande` | `loops_grande` | `calls` | `counter` | `strings` |
|---|---:|---:|---:|---:|---:|
| rlox | 0.96 | 0.98 | 0.98 | 0.97 | — |
| loxx | 0.96 | 0.98 | 0.98 | 0.97 | 0.98 |
| glox | 1.00 | 1.01 | 1.01 | 1.00 | 1.02 |
| glox-vm-sin-opt | 0.99 | 1.01 | 1.01 | 1.00 | 1.01 |
| jlox | 1.81 | 1.40 | 1.29 | 1.35 | 1.57 |
| glox-parcial | 1.11 | 1.04 | 1.08 | 1.04 | 1.03 |
| slox | 0.99 | 1.00 | 1.00 | 1.00 | 1.00 |
| dlox | 1.00 | 0.99 | 1.00 | 1.00 | 0.99 |
| plox-php | 1.00 | 1.00 | 1.00 | 1.00 | 0.98 |
| cloxure | 1.02 | 1.38 | 1.04 | 1.08 | 1.63 |
| plox | 1.00 | 1.00 | 1.00 | 1.00 | 1.00 |

## Escalado: fib(n)

| Implementación | n=15 | n=18 | n=21 | n=24 | n=27 |
|---|---:|---:|---:|---:|---:|
| rlox | 4.4 ms | 4.2 ms | 4.8 ms | 6.4 ms | 14 ms |
| loxx | 1.8 ms | 1.9 ms | 2.5 ms | 4.8 ms | 14 ms |
| glox | 2.3 ms | 2.7 ms | 3.6 ms | 7.9 ms | 26 ms |
| glox-vm-sin-opt | 2.1 ms | 2.5 ms | 3.6 ms | 8.2 ms | 27 ms |
| jlox | 30 ms | 34 ms | 44 ms | 53 ms | 78 ms |
| glox-parcial | 2.4 ms | 3.7 ms | 7.6 ms | 24 ms | 91 ms |
| slox | 4.4 ms | 9.4 ms | 30 ms | 117 ms | 483 ms |
| dlox | 20 ms | 63 ms | 271 ms | 1.25 s | 5.88 s |
| plox-php | 44 ms | 92 ms | 322 ms | 1.39 s | 6.45 s |
| cloxure | 280 ms | 515 ms | 1.53 s | 6.42 s | 29.1 s |
| plox | 74 ms | 147 ms | 381 ms | 1.32 s | 5.27 s |

## Corridas inválidas (no aparecen en tablas ni gráficos)

- `strings` con rlox: código de salida 101
- `metodos` con plox: imprimió 'Scanning Error: Unexpected character: `.`', se esperaba 5000950000
- `metodos` con plox-php: imprimió '[line 11] Only instances have fields.', se esperaba 5000950000
- `metodos` con glox-parcial: código de salida 65
- `metodos` con glox-vm-sin-opt: código de salida 65

## Notas de validación

- `loops_grande` con loxx: imprime con 6 cifras significativas (2e+12)
- `calls` con loxx: imprime con 6 cifras significativas (5.00000e+11)

Corridas medidas por combinación: entre 3 y 5 (ver `run_benchmarks.py --help`).
