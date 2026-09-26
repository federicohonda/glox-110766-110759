# Benchmarks de glox

Esta carpeta tiene todo lo necesario para medir glox y compararlo contra otras
implementaciones de Lox: los scripts, las herramientas para compilar las otras
implementaciones, correr las mediciones y generar gráficos, y los resultados
versionados. **El análisis de los resultados está en el [README principal](../README.md#benchmarks)**;
acá está el detalle de cómo se mide y cómo reproducirlo.

## Índice

1. [Contenido de la carpeta](#1-contenido-de-la-carpeta)
2. [Scripts de benchmark](#2-scripts-de-benchmark)
3. [Implementaciones comparadas](#3-implementaciones-comparadas)
4. [Cómo reproducir todo](#4-cómo-reproducir-todo)
5. [Metodología](#5-metodología)
6. [Benchmarks unitarios en Go](#6-benchmarks-unitarios-en-go)
7. [Experimentos del análisis](#7-experimentos-del-análisis)
8. [Corrida histórica: glox vs plox en WSL2](#8-corrida-histórica-glox-vs-plox-en-wsl2)

---

## 1. Contenido de la carpeta

| Archivo | Qué es |
|---|---|
| `*.lox` | Los scripts que se miden. Cada uno declara su resultado con `// resultado: N`. |
| `setup_impls.sh` | Clona (si hace falta) y compila las otras 8 implementaciones de Lox. Genera `.build/impls.tsv` con el comando para correr cada una. |
| `run_benchmarks.py` | Corre cada script con cada implementación, valida la salida y guarda todas las muestras en `results/results.json`. |
| `plot.py` | Lee `results/results.json` y genera los gráficos SVG y `results/RESULTS.md` con todas las tablas. |
| `run_comparison.sh` | La comparativa original glox vs plox (solo Linux, usa GNU `time -v`). Se conserva por la corrida histórica de la sección 7. |
| `results/` | Resultados de la última corrida: `results.json` (muestras crudas), `RESULTS.md` (tablas), `*.svg` (gráficos), `go-bench.txt` (benchmarks de Go). |
| `.impls/`, `.build/` | Repos clonados y binarios compilados. No se versionan (`.gitignore`). |

## 2. Scripts de benchmark

Todos usan solo Lox estándar (sin `%`, sin clases, sin funciones nativas como
`clock()`), para que corran igual en todas las implementaciones.

| Script | Qué estresa | Carga | Resultado |
|---|---|---|---|
| `startup.lox` | Nada: mide el costo fijo de levantar el intérprete | `print "ok";` | `ok` |
| `fib.lox` | Llamadas recursivas, creación de entornos, `return` | `fib(25)`: 242.785 llamadas | `75025` |
| `loops.lox` | Bucle `for`, aritmética, asignación | 100.000 iteraciones | `4999950000` |
| `closures.lox` | Crear closures que capturan su entorno | 10.000 closures | `50005000` |
| `fib_grande.lox` | Lo mismo que `fib.lox`, con más carga | `fib(28)`: 832.039 llamadas | `317811` |
| `loops_grande.lox` | Lo mismo que `loops.lox`, con más carga | 2.000.000 iteraciones | `1999999000000` |
| `calls.lox` | Llamadas no recursivas: costo fijo de cada llamada | 2.000.000 llamadas a una función hoja | `500000500000` |
| `counter.lox` | Leer y escribir una variable capturada por una closure | 1.000.000 llamadas | `1000000` |
| `strings.lox` | Crear strings nuevos por concatenación y compararlos | 1.000.000 concatenaciones | `200000` |

`fib`, `loops` y `closures` son los scripts originales de la primera
comparativa y se dejaron con su carga original. En una máquina rápida, glox
los termina en menos de 40 ms, así que ahí gran parte de lo que se mide es el
arranque del proceso: por eso se agregaron las versiones `_grande` y los
demás scripts, con carga suficiente para que el arranque no domine.

Además, `run_benchmarks.py` genera al vuelo `fib(n)` para n = 15, 18, 21, 24 y 27
(la **curva de escalado**), que muestra a partir de qué tamaño deja de pesar el
arranque en cada implementación.

## 3. Implementaciones comparadas

| Nombre | Lenguaje | Estrategia | Repositorio | Cómo se compila |
|---|---|---|---|---|
| **glox** | Go | tree-walk | este repo | `go build` |
| rlox | Rust | AST → bytecode + VM | [Darksecond/lox](https://github.com/Darksecond/lox) | `cargo build --release -p lox` |
| slox | Swift | tree-walk | [alexito4/slox](https://github.com/alexito4/slox) | `swift build -c release` (con parche, ver abajo) |
| jlox | Java | tree-walk | [ryanq/jlox](https://github.com/ryanq/jlox) | `javac` |
| cloxure | Clojure | tree-walk | [ceronman/cloxure](https://github.com/ceronman/cloxure) | `lein uberjar` |
| loxx | C++ | bytecode + VM | [mspraggs/loxx](https://github.com/mspraggs/loxx) | CMake, `-O3` (con parche, ver abajo) |
| plox-php | PHP | tree-walk | [minirop/plox](https://github.com/minirop/plox) | interpretado por `php` |
| dlox | Dart | tree-walk | [sma/lox](https://github.com/sma/lox) | `dart compile exe` (AOT) |
| plox | Python | tree-walk | [FdelMazo/plox](https://github.com/FdelMazo/plox) | `uv sync` con Python 3.12 |

**Parches necesarios para compilar con toolchains actuales.** `setup_impls.sh`
nunca modifica los repos: los parches se aplican sobre copias en `.build/`.

- **slox** declara `swift-tools-version:4.0` y depende de `antitypical/Result`
  (tools 3.1), versiones que Swift 6 ya no acepta. Como Swift 5 trae `Result`
  en la biblioteca estándar, se compila con un `Package.swift` moderno, sin esa
  dependencia y con un shim de 4 líneas para `.value`/`.error`.
- **loxx** inicializa un iterador de `std::vector` con `ip_(0)`, algo que el
  libc++ actual no permite, y pide una versión de CMake que CMake 4 rechaza. Se
  compila con `ip_()` y `-DCMAKE_POLICY_VERSION_MINIMUM=3.5`.
- **cloxure** no hace AOT de su namespace principal, así que el jar se arranca
  con `clojure.main -m cloxure.core` en vez de `java -jar`.
- **plox-php** se corre con `-d error_reporting=24575` para que PHP 8.5 no
  llene stderr de avisos de deprecación (no cambia el comportamiento).

## 4. Cómo reproducir todo

### Requisitos

- Go (para glox) y Python 3.9+ (para los scripts de medición; solo usan la biblioteca estándar).
- Para las otras implementaciones: `cargo`, `swift`, un JDK 17+, `lein`, `cmake`
  y un compilador C++, `php`, `dart` y `uv`. Si falta alguno, esa
  implementación se saltea con un aviso y el resto se mide igual.

En macOS con Homebrew:

```bash
brew install go rust php openjdk leiningen cmake uv dart-sdk
# swift y clang vienen con las Command Line Tools de Xcode: xcode-select --install
```

### Paso a paso

Todos los comandos se corren desde la raíz del repo (`glox-110766-110759/`).

```bash
# 1. Clonar y compilar las 9 implementaciones (glox incluida).
#    Deja los repos en benchmarks/.impls y los binarios en benchmarks/.build.
./benchmarks/setup_impls.sh

#    Si ya tenés los repos clonados en otro lado:
IMPLS_DIR=~/lox-impls ./benchmarks/setup_impls.sh
#    Para recompilar solo algunas (el resto de impls.tsv se conserva):
ONLY=glox,rlox ./benchmarks/setup_impls.sh

# 2. Medir. Tarda alrededor de 30 minutos con las 9 implementaciones,
#    sobre todo por plox y cloxure.
python3 benchmarks/run_benchmarks.py

#    Variantes útiles:
python3 benchmarks/run_benchmarks.py --only glox,plox            # solo algunas
python3 benchmarks/run_benchmarks.py --scripts fib,calls --runs 10
python3 benchmarks/run_benchmarks.py --sin-escalado              # sin la curva fib(n)
python3 benchmarks/run_benchmarks.py --help

# 3. Generar gráficos y tablas en benchmarks/results/.
python3 benchmarks/plot.py
```

`run_benchmarks.py` **actualiza** `results/results.json` en vez de pisarlo: si
se vuelve a correr con `--only glox`, solo se reemplazan las mediciones de glox.
Esto es lo que se va a usar en el Bloque 2 para sumar la versión a bytecode
sin tener que re-medir todo. Ojo: mezclar mediciones de máquinas distintas en
el mismo JSON invalida la comparación.

### Solo glox, sin instalar nada más

```bash
go build -o glox ./cmd/glox
./glox benchmarks/fib_grande.lox       # 317811
time ./glox benchmarks/calls.lox        # 5.000005e+11
```

## 5. Metodología

- **Qué se mide:** el proceso completo, de punta a punta (arranque del runtime,
  scan, parse, resolución y ejecución), que es lo que experimenta alguien que
  corre `./glox script.lox`. Los costos por etapa se miden aparte con los
  benchmarks de Go (sección 6).
- **Tiempo:** `time.perf_counter` alrededor del proceso hijo. Se reporta la
  **mediana** de las corridas: es menos sensible que el promedio a una corrida
  aislada con interferencia del sistema.
- **CPU y memoria:** `os.wait4` devuelve el `rusage` del hijo: tiempo de CPU
  (user + sys) y memoria residente pico (RSS). Funciona igual en macOS y en
  Linux, a diferencia de `/usr/bin/time -v`, que es solo de GNU.
- **Cantidad de corridas:** una de calentamiento que se descarta (salvo que
  tarde más de 2 s, donde el calentamiento no cambia nada) y hasta 5 medidas.
  Si una combinación ya lleva más de 60 s acumulados, se corta en 3 corridas.
- **Validación:** cada corrida tiene que terminar con código 0 y su última
  línea tiene que coincidir con el `// resultado:` del script, comparando por
  valor numérico (glox imprime `4.99995e+09`, jlox `4.99995E9`, plox
  `4999950000.0`: son el mismo número). Una implementación que da mal el
  resultado se marca como inválida y no aparece en los gráficos.
- **Misma máquina para todo:** las 9 implementaciones se miden en la misma
  corrida, en la misma máquina y sin otras cargas pesadas en paralelo. Los
  datos de la máquina quedan guardados en `results/results.json`.

**Limitaciones conocidas.** El tiempo total incluye el arranque, que en la JVM
(jlox, cloxure) es de decenas a cientos de milisegundos; por eso se mide aparte
con `startup.lox`. Las implementaciones con JIT (jlox, cloxure, dlox en menor
medida) mejoran a medida que el programa corre, así que su posición relativa
depende del tamaño del programa: la curva de escalado lo muestra.

## 6. Benchmarks unitarios en Go

Miden cada etapa del pipeline por separado, con `testing.B`, sin el costo de
arranque del proceso:

```bash
go test -run '^$' -bench . -benchmem ./internal/...
```

| Paquete | Benchmark | Qué mide |
|---|---|---|
| `scanner` | `BenchmarkScanArithmetic` | Escanear 100 líneas de expresiones aritméticas (~500 tokens) |
| `scanner` | `BenchmarkScanKeywords` | Escaneo denso de keywords, identificadores, strings y números |
| `parser` | `BenchmarkParseExpressions` | Armar el AST de expresiones con todos los niveles de precedencia |
| `parser` | `BenchmarkParseFunctions` | Parsear declaraciones de funciones con bloques y condicionales |
| `interpreter` | `BenchmarkFibRecursive` | Ejecutar `fib(15)` (el scan/parse/resolve queda afuera del timer) |
| `interpreter` | `BenchmarkLoopIntensive` | Bucle `for` de 10.000 iteraciones |
| `interpreter` | `BenchmarkClosureCounter` | 1.000 llamadas a una closure con variable capturada |
| `interpreter` | `BenchmarkPipelineEndToEnd` | scan → parse → resolve → interpret de un programa chico |

Resultados de la última corrida en `results/go-bench.txt`.

Mediana de 5 corridas (`-count 5`) en Apple M5:

| Benchmark | Tiempo/op | Memoria/op | Alocaciones/op |
|---|---:|---:|---:|
| `BenchmarkScanArithmetic` | 17.0 µs | 123.2 KB | 511 |
| `BenchmarkScanKeywords` | 49.0 µs | 465.7 KB | 214 |
| `BenchmarkParseExpressions` | 21.6 µs | 69.9 KB | 1.506 |
| `BenchmarkParseFunctions` | 17.2 µs | 48.5 KB | 905 |
| `BenchmarkFibRecursive` | 276.6 µs | 760.7 KB | 12.453 |
| `BenchmarkLoopIntensive` | 1.35 ms | 1407.2 KB | 60.008 |
| `BenchmarkClosureCounter` | 175.7 µs | 220.1 KB | 9.015 |
| `BenchmarkPipelineEndToEnd` | 28.6 µs | 64.0 KB | 1.167 |

Lectura rápida: `fib(15)` son 1.973 llamadas, así que cada llamada cuesta ~140 ns y ~6 alocaciones; el `for` de 10.000 vueltas cuesta ~135 ns y 6 alocaciones por vuelta (cada vuelta ejecuta un bloque, y cada bloque crea su `Environment`). El análisis está en el README principal.


## 7. Experimentos del análisis

El README principal usa dos experimentos más, que no forman parte de la suite
porque responden una pregunta puntual cada uno.

**Costo del `return`.** El mismo bucle de 300.000 llamadas, en una versión que
devuelve el valor con `return` y otra que lo acumula en una global sin
`return`. El cociente entre las dos aísla lo que cuesta el mecanismo de
`return` de cada implementación:

```lox
// con_return.lox
var total = 0;
fun f(a) { return a; }
for (var i = 0; i < 300000; i = i + 1) { total = total + f(i); }
print total;

// sin_return.lox
var total = 0;
fun f(a) { total = total + a; }
for (var i = 0; i < 300000; i = i + 1) { f(i); }
print total;
```

```bash
for f in sin_return con_return; do time ./glox $f.lox; done
```

**De dónde salen las alocaciones de glox.** Perfil de memoria de
`BenchmarkFibRecursive` (el perfil de CPU en macOS muestra sobre todo hilos
ociosos del runtime y no sirve para esto; el de memoria es exacto):

```bash
cd internal/interpreter
go test -run '^$' -bench BenchmarkFibRecursive -memprofile mem.prof -memprofilerate 1 -o interp.test
go tool pprof -sample_index=alloc_objects -top interp.test mem.prof
```

## 8. Corrida histórica: glox vs plox en WSL2

La primera comparativa se hizo con `run_comparison.sh` (promedio de 3
corridas, `/usr/bin/time -v`) sobre Linux/WSL2 con un AMD Ryzen 3 3200U. Se
conserva como registro; **no** se mezcla con los resultados actuales porque es
otra máquina.

| Script | Intérprete | Tiempo real | CPU usuario | RAM pico | Relación |
|---|---|---|---|---|---|
| `fib.lox` | glox | 0,180 s | 0,180 s | 10.948 KB | 81× más rápido |
| `fib.lox` | plox | 14,630 s | 13,980 s | 29.056 KB | base |
| `loops.lox` | glox | 0,070 s | 0,080 s | 10.904 KB | 109× más rápido |
| `loops.lox` | plox | 7,640 s | 7,370 s | 28.960 KB | base |
| `closures.lox` | glox | 0,030 s | 0,030 s | 8.668 KB | 55× más rápido |
| `closures.lox` | plox | 1,660 s | 1,570 s | 29.012 KB | base |

Para correrla (Linux):

```bash
./benchmarks/run_comparison.sh [glox_bin] [plox_bin] [repeticiones]
```

En la corrida actual (Apple M5, Python 3.12) la ventaja de glox sobre plox en
esos mismos tres scripts es 57×, 64× y 40×: menor que en WSL2 porque en una
máquina más rápida glox los termina en 7–36 ms, y a esa escala el arranque de
cada proceso pesa más. Por eso se agregaron los scripts con más carga.

Nota: esa tabla reportaba la salida de `loops.lox` como `4999950000`; glox en
realidad imprime `4.99995e+09` (mismo valor, formato `%g` como clox).
