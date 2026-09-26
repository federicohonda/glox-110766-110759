# Benchmarks de Rendimiento — glox

## Índice

1. [Benchmarks Unitarios en Go](#1-benchmarks-unitarios-en-go)
2. [Scripts de Benchmark en Lox](#2-scripts-de-benchmark-en-lox)
3. [Comparativa glox vs plox](#3-comparativa-glox-go-vs-plox-python)
4. [Conclusiones](#4-conclusiones)
5. [Cómo Reproducir](#5-cómo-reproducir)

---

## 1. Benchmarks Unitarios en Go

Medición interna del runtime de Go con `go test -bench=. -benchmem`, aislando cada etapa del pipeline.

**Entorno:** Linux/WSL2, AMD Ryzen 3 3200U, Go 1.26.1.

### Scanner (`internal/scanner`)

| Benchmark | ns/op | B/op | allocs/op | Descripción |
|---|---|---|---|---|
| `BenchmarkScanArithmetic` | ~110.559 | 126.193 | 511 | Escaneo de 100 líneas de expresiones aritméticas (~500 tokens) |
| `BenchmarkScanKeywords` | ~367.288 | 476.850 | 214 | Escaneo denso de keywords, identificadores, strings y números |

### Parser (`internal/parser`)

| Benchmark | ns/op | B/op | allocs/op | Descripción |
|---|---|---|---|---|
| `BenchmarkParseExpressions` | ~194.032 | 71.616 | 1.506 | Construcción del AST de expresiones con todos los niveles de precedencia |
| `BenchmarkParseFunctions` | ~148.767 | 49.712 | 905 | Parseo de declaraciones de funciones con bloques y condicionales |

### Intérprete (`internal/interpreter`)

| Benchmark | ns/op | B/op | allocs/op | Descripción |
|---|---|---|---|---|
| `BenchmarkFibRecursive` | ~1.609.118 | 778.934 | 12.453 | Ejecución pura de `fib(15)` recursivo (sin scan/parse) |
| `BenchmarkLoopIntensive` | ~8.001.265 | 1.440.988 | 60.008 | Bucle `for` de 10.000 iteraciones con acumulación |
| `BenchmarkClosureCounter` | ~1.287.364 | 225.385 | 9.015 | 1.000 invocaciones a un closure con variable capturada |
| `BenchmarkPipelineEndToEnd` | ~168.326 | 65.568 | 1.167 | Pipeline completo scan → parse → resolve → interpret |

**Observaciones:**
- El scanner es la etapa más rápida del pipeline (~110-370 µs para 100 líneas).
- El parser genera significativamente más alocaciones que el scanner (nodos del AST en el heap), pero a un costo absoluto similar.
- El intérprete domina el costo total en programas computacionalmente intensivos: `fib(15)` cuesta ~1.6 ms, pero el pipeline end-to-end completo sobre un programa breve solo tarda ~168 µs.
- Los bucles son proporcionalmente más costosos en alocaciones por iteración (6 allocs/iteración en `BenchmarkLoopIntensive`) debido a la creación de entornos (`Environment`) por cada vuelta del `for` desazucarado.

---

## 2. Scripts de Benchmark en Lox

Scripts autocontenidos en `benchmarks/` que estresan distintas áreas del intérprete:

| Script | Qué mide | Carga | Resultado esperado |
|---|---|---|---|
| `fib.lox` | Llamadas recursivas a función, creación de entornos/frames | `fib(25)` → 242.785 llamadas | `75025` |
| `loops.lox` | Iteración, aritmética, variables locales | 100.000 iteraciones de `for` | `4999950000` |
| `closures.lox` | Creación de closures, captura léxica, retención de entornos | 10.000 closures instanciados | `50005000` |

---

## 3. Comparativa glox (Go) vs plox (Python)

Medición de proceso completo (arranque, scan, parse, resolve, interpret) con `/usr/bin/time -v` sobre Linux/WSL2.

`plox` es el intérprete de referencia de la cátedra, implementado en Python (rama `main` del repo oficial).

| Script de Benchmark | Intérprete | Tiempo Real (s) | CPU Usuario (s) | Memoria RAM Pico (KB) | Ratio de Velocidad |
|---|---|---|---|---|---|
| `fib.lox` | **glox (Go)** | `0.180` | `0.180` | `10.948` | **81x más rápido** |
| `fib.lox` | plox (Python) | `14.630` | `13.980` | `29.056` | 1.0x (base) |
| `loops.lox` | **glox (Go)** | `0.070` | `0.080` | `10.904` | **109x más rápido** |
| `loops.lox` | plox (Python) | `7.640` | `7.370` | `28.960` | 1.0x (base) |
| `closures.lox` | **glox (Go)** | `0.030` | `0.030` | `8.668` | **55x más rápido** |
| `closures.lox` | plox (Python) | `1.660` | `1.570` | `29.012` | 1.0x (base) |

---

## 4. Conclusiones

### Velocidad de Ejecución

`glox` es entre **55x y 109x más rápido** que `plox` en todos los escenarios evaluados:

- **Recursión profunda (`fib.lox`, 81x):** El costo dominante es la creación y destrucción de entornos (`Environment`) por cada llamada a función. En Go, los structs se alocan con un layout de memoria compacto y contiguo, y el Garbage Collector generacional libera eficientemente los frames de corta vida. En Python, cada frame conlleva objetos `PyObject` con punteros, contadores de referencia y metadatos que multiplican el overhead.

- **Bucles intensivos (`loops.lox`, 109x):** Es el caso con mayor aceleración. El `for` de Lox se desazucara en el parser a un `while` con bloque envolvente, y la ejecución de cada iteración en Go se resuelve como un `type switch` estático compilado a código de máquina x86_64 nativo, mientras que CPython debe interpretar bytecode instrucción a instrucción en un loop `PREDICT`/`DISPATCH` con overhead de evaluación dinámica en cada paso.

- **Closures (`closures.lox`, 55x):** Es el caso de menor ventaja relativa porque el costo se concentra en la alocación de entornos retenidos (no efímeros), que tanto Go como Python delegan al heap y a sus respectivos GC. Aun así, la compactitud de los structs de Go sigue ganando.

### Consumo de Memoria

`glox` utiliza **~2.7 veces menos memoria RAM** en pico (~10 MB vs ~29 MB):

- Los nodos del AST y los registros de entornos en Go son structs fuertemente tipados con un footprint mínimo en el heap.
- El runtime de CPython arranca con un baseline de ~20 MB solo por cargar el intérprete, sus módulos internos y las tablas de objetos.
- El Garbage Collector de Go (tri-color, concurrent, mark-and-sweep) mantiene baja la presión de memoria con pausas sub-milisegundas, mientras que el ciclo de referencia-conteo + GC cíclico de CPython es menos agresivo con la liberación de objetos de vida corta.

### Nota sobre la Comparación

Esta comparativa mide el **intérprete tree-walk** de ambas implementaciones. El enunciado pide documentar cómo mejora el rendimiento a lo largo del desarrollo; en la entrega final (Bloque 2, con compilación a bytecode y VM), estos números servirán como **baseline** contra el cual medir la aceleración de la nueva arquitectura.

---

## 5. Cómo Reproducir

### Benchmarks unitarios en Go

```bash
cd glox-110766-110759
go test -bench=. -benchmem ./internal/scanner ./internal/parser ./internal/interpreter
```

### Scripts `.lox` individuales

```bash
go build -o glox ./cmd/glox
./glox benchmarks/fib.lox       # → 75025
./glox benchmarks/loops.lox     # → 4999950000
./glox benchmarks/closures.lox  # → 50005000
```

### Comparativa automatizada glox vs plox

```bash
# Requiere plox instalado (ver sección de requisitos)
./benchmarks/run_comparison.sh [glox_bin] [plox_bin] [repeticiones]

# Ejemplo con valores por defecto (3 repeticiones):
./benchmarks/run_comparison.sh

# Guardar resultado en archivo Markdown:
./benchmarks/run_comparison.sh > benchmarks/RESULTS.md
```

El script:
1. Compila `glox` automáticamente si no existe el binario.
2. Detecta `plox` en el virtualenv local o en el PATH.
3. Ejecuta N repeticiones de cada script con `/usr/bin/time -v`.
4. Calcula promedios de tiempo real, CPU usuario y memoria RAM pico.
5. Imprime la tabla Markdown con el ratio de aceleración.
