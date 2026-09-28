# glox — Lox en Go

Implementación del lenguaje [Lox](https://craftinginterpreters.com/the-lox-language.html)
en Go, para el TP de **Lenguajes y Compiladores I (FIUBA)**, Opción I.

| | |
|---|---|
| **Integrantes** | Federico Honda (110766) · Franco Bustos (110759) |
| **Entrega** | Parcial — Bloque 1: intérprete *tree-walk* (28/09/2026) |
| **Estado** | Scanner, parser, resolver e intérprete completos · pasa las 5 pruebas de la cátedra (57 aserciones) |
| **Próximo** | Bloque 2: compilación a bytecode + VM, clases y herencia, una funcionalidad extra (16/11/2026) |

## Índice

1. [Cómo compilar y ejecutar](#cómo-compilar-y-ejecutar)
2. [Arquitectura](#arquitectura)
3. [Diferencias con la implementación de la cátedra](#diferencias-con-la-implementación-de-la-cátedra)
4. [Pruebas](#pruebas)
5. [Benchmarks](#benchmarks)
6. [Cómo trabajamos](#cómo-trabajamos)
7. [Implementaciones usadas en los benchmarks](#implementaciones-usadas-en-los-benchmarks)

---

## Cómo compilar y ejecutar

Requiere **Go 1.26** o posterior. No tiene dependencias externas.

```bash
git clone https://github.com/federicohonda/glox-110766-110759.git
cd glox-110766-110759
go build -o glox ./cmd/glox
```

### Ejecutar un programa

```bash
./glox examples/hello.lox        # recorrido por la sintaxis básica
```

Sin compilar el binario también se puede usar `go run ./cmd/glox examples/hello.lox`.

### REPL

Sin argumentos, glox abre un REPL. Las variables y funciones se conservan entre
líneas, y un error no cierra la sesión:

```text
$ ./glox
> fun cuadrado(x) { return x * x; }
> var n = cuadrado(12);
> print n;
144
> print m;
[línea 1] Error en tiempo de ejecución en 'm': variable no definida 'm'.
> print n + 1;
145
```

Se sale con `Ctrl+D` (fin de la entrada).

### Modos de inspección

```bash
./glox --scanning examples/hello.lox   # lista de tokens: TIPO 'lexema' literal
./glox --parsing  examples/hello.lox   # AST en notación prefija, una sentencia por línea
```

```text
$ echo 'print 1 + 2 * 3;' > /tmp/t.lox
$ ./glox --parsing /tmp/t.lox
(print (+ 1 (* 2 3)))
```

Los dos modos también funcionan en el REPL (`./glox --parsing`).

### Códigos de salida

| Código | Significado |
|---|---|
| `0` | El programa terminó bien |
| `64` | Uso incorrecto del CLI (más de un archivo) |
| `65` | Error estático: léxico, de sintaxis o semántico. **No se ejecuta ninguna línea** |
| `70` | Error en tiempo de ejecución. Lo anterior al error sí se ejecutó |

Los errores van a `stderr` con la línea donde ocurrieron:

```text
[línea 3] Error de sintaxis en ';': se esperaba una expresión.
[línea 5] Error semántico en 'return': no se puede usar 'return' fuera de una función.
[línea 7] Error en tiempo de ejecución en '+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: a y 1
```

### Tests y benchmarks

```bash
go test ./...                               # unitarios + end-to-end
go test -cover ./internal/...               # cobertura por paquete
python3 ./real-tests/script.py ./glox       # pruebas de la cátedra
go test -run '^$' -bench . -benchmem ./internal/...   # benchmarks de Go
```

La comparación contra otras implementaciones tiene su propia guía en
[`benchmarks/README.md`](benchmarks/README.md).

---

## Arquitectura

```
Código fuente (.lox o REPL)
       │
       ▼
 [1. Scanner]  ──────────► errores léxicos (65)
       │ []token.Token
       ▼
 [2. Parser]   ──────────► errores de sintaxis (65)
       │ []ast.Stmt
       ▼
 [3. Resolver] ──────────► errores semánticos (65)
       │ map[ast.Expr]int   (a cuántos entornos está cada variable local)
       ▼
 [4. Interpreter] ───────► errores de runtime (70)
```

| Paquete | Qué hace |
|---|---|
| `cmd/glox` | CLI: REPL, archivo, `--scanning`, `--parsing` y códigos de salida. Arma el pipeline y corta en la primera etapa con errores. |
| `internal/token` | `TokenType` (enum con `iota`) y `Token{Type, Lexeme, Literal, Line}`. |
| `internal/scanner` | Análisis léxico *maximal munch*. Acumula todos los errores léxicos del archivo en vez de cortar en el primero. |
| `internal/ast` | Nodos del AST como structs de datos puros, separados por archivo (`ast.go`, `stmt.go`, `expr_variable.go`, `expr_logical.go`, `expr_call.go`). Cada nodo se imprime en notación prefija (`String()`). |
| `internal/parser` | Parser recursivo descendente: una función por regla de la gramática. Desazucara `for` a `while`. Se recupera de los errores sincronizando hasta la próxima sentencia. |
| `internal/resolver` | Análisis semántico estático: calcula la distancia de scope de cada variable local y detecta `return` fuera de funciones, lecturas de una variable en su propio inicializador y redeclaraciones locales. |
| `internal/interpreter` | Evaluación con `type switch`, `Environment` encadenado, funciones con closures. |
| `internal/bytecode` | Vacío por ahora: es el lugar del compilador y la VM del Bloque 2. |

### Gramática implementada

```
program     → declaration* EOF ;
declaration → funDecl | varDecl | statement ;
funDecl     → "fun" IDENTIFIER "(" parameters? ")" block ;
varDecl     → "var" IDENTIFIER ( "=" expression )? ";" ;
statement   → exprStmt | forStmt | ifStmt | printStmt | returnStmt | whileStmt | block ;
expression  → assignment ;
assignment  → IDENTIFIER "=" assignment | logic_or ;
logic_or    → logic_and ( "or" logic_and )* ;
logic_and   → equality ( "and" equality )* ;
equality    → comparison ( ( "!=" | "==" ) comparison )* ;
comparison  → term ( ( ">" | ">=" | "<" | "<=" ) term )* ;
term        → factor ( ( "-" | "+" ) factor )* ;
factor      → unary ( ( "/" | "*" | "%" ) unary )* ;
unary       → ( "!" | "-" ) unary | call ;
call        → primary ( "(" arguments? ")" )* ;
primary     → NUMBER | STRING | "true" | "false" | "nil" | IDENTIFIER | "(" expression ")" ;
```

Las clases (`class`, `this`, `super`, `.`) quedan para el Bloque 2: sus tokens
todavía no existen en el scanner, a propósito, para no tener código a medias.

---

## Diferencias con la implementación de la cátedra

La referencia es **plox**, el intérprete tree-walk en Python que se construye
en clase. Todo lo de esta sección se verificó corriendo los mismos scripts en
plox y en glox.

### Qué cambia en el lenguaje

| | plox | glox |
|---|---|---|
| Operador `%` | Sí | Sí (`math.Mod`, misma precedencia que `*` y `/`) |
| `print 1 / 0;` y `print 1 % 0;` | Error de runtime | Error de runtime |
| Varios errores léxicos o de sintaxis en un archivo | Reporta el primero y corta | **Reporta todos**: el scanner los acumula y el parser se sincroniza en la próxima sentencia |
| Varios errores semánticos | Reporta el primero y corta | **Reporta todos**, y si hay alguno no se ejecuta nada |
| Código de salida ante un error | `0` en todos los casos | **`65`** si es estático, **`70`** si es de runtime |
| `print 4999950000;` | `4999950000.0` | `4.99995e+09` (formato `%v` de Go) |
| `print 3;` | `3.0` | `3` |
| Mensajes de error | Inglés, sin número de línea en runtime | Español, con línea y lexema: `[línea 7] Error en tiempo de ejecución en '+': ...` |
| Clases, `this`, `super` | Sí | Bloque 2 |

### Qué nos obligó (o nos permitió) hacer distinto Go

**1. `type switch` para recorrer el AST.** plox despacha por tipo de nodo con
`@singledispatchmethod`: registra una función por clase y Python elige en
runtime. En Go, la interfaz `ast.Expr` es un conjunto cerrado de structs y un
`switch e := expr.(type)` despacha directo, sin registros ni un patrón Visitor
con métodos `accept` en cada nodo. El costo: el compilador no avisa si falta un
caso. Lo cubrimos con un `default` que devuelve un error explícito y con tests
por tipo de nodo.

**2. `return` como un valor de error, no como excepción.** plox corta la
ejecución de una función con `raise ReturnValue(...)`, que atrapa la llamada.
Go no tiene excepciones, y usar `panic` para control de flujo normal va contra
el estilo del lenguaje. Como `Execute` ya devolvía `error` en todas las
sentencias, `return` devuelve un `*returnSignal` (un tipo que implementa
`error` y lleva el valor): sube solo a través de bloques, `if` y `while` sin
que ninguno lo conozca, hasta que `Function.Call` lo reconoce. Esto tiene un
efecto medible: ver el análisis de [Benchmarks](#benchmarks).

**3. `panic`/`recover` sí, pero solo dentro del parser.** plox lanza
`SyntaxError` y el primer error termina el parseo. En glox un error de sintaxis
también puede detectarse diez niveles abajo en la recursión, pero queríamos
volver hasta la sentencia, sincronizar y seguir buscando errores. Enhebrar
`(ast.Expr, error)` en cada regla habría duplicado el tamaño del parser, así
que se usa `panic(*ParseError)` en el punto exacto del error y un único
`recover` por declaración. El `panic` nunca sale del paquete. Los errores que
no desorientan al parser (más de 255 argumentos) se anotan sin `panic`, para
no generar errores falsos en cascada.

**4. El resolver no conoce al intérprete.** En plox, el `Resolver` recibe el
intérprete y le va avisando la distancia de cada variable
(`interpreter.resolve_depth`). En Go eso sería un import circular prohibido
entre paquetes, y además mezcla análisis estático con ejecución.
`resolver.Resolve(stmts)` es una función pura que devuelve un `map[ast.Expr]int`;
la clave es el puntero al nodo, así que dos `print a;` iguales en lugares
distintos tienen distancias independientes. El paquete no tiene ningún `Value`:
queda a la vista que el análisis semántico no ejecuta nada.

**5. Tipos estáticos para valores dinámicos.** En Python cualquier variable
puede tener un número, un string o una función. En Go, un valor de Lox es
`any` y cada operación pregunta explícitamente qué tiene (`v.(float64)`), lo
que obliga a escribir cada chequeo de tipos de Lox en el intérprete: no hay
ninguna conversión implícita "prestada" del lenguaje anfitrión.

**6. El GC resuelve las closures, igual que en Python.** Una `Function` guarda
un puntero al `Environment` donde se declaró; ese entorno sigue vivo mientras
alguna función lo referencie, aunque el bloque que lo creó ya haya terminado.
Esto va a cambiar en el Bloque 2: con una VM de pila, las variables locales
viven en el stack y hay que sacarlas de ahí cuando una closure las captura.

**En común con plox:** `for` se desazucara en el parser a
`{ init; while (cond) { cuerpo; inc; } }`, sin nodo propio en el AST.

---

## Pruebas

| Nivel | Dónde | Qué cubre |
|---|---|---|
| Pruebas de la cátedra | `real-tests/` | 5 archivos, 57 aserciones: aritmética, strings, control de flujo, funciones, el bug de closures, una máquina de Minsky y FizzBuzz. **5/5 OK.** |
| Unitarias | `internal/*/*_test.go` | Cada paquete por separado. Cobertura: parser 96,4%, scanner 94,8%, interpreter 93,9%, resolver 91,1%. |
| End-to-end | `tests/` | El binario real sobre 38 scripts `.lox`, más el REPL, los modos de inspección, los códigos de salida, la salida de `examples/hello.lox` y el resultado de cada benchmark. |

Los scripts de `tests/lox/` declaran lo que esperan en comentarios, y
`tests/e2e_test.go` compila glox y los verifica:

```lox
// exit: 70
// stderr: [línea 5] Error en tiempo de ejecución en '+'
// Todo lo anterior al error se ejecuta.
print "antes"; // expect: antes
print "a" + 1;
print "después";
```

Están agrupados por tema: `expresiones/`, `variables/`, `control/`,
`funciones/`, `errores_estaticos/` (código 65: no se ejecuta nada) y
`errores_runtime/` (código 70: se ejecuta hasta el error). Algunos casos que
vale la pena mirar:

- `funciones/closure_bug.lox`: el caso clásico que arregla el resolver.
- `errores_runtime/regla_de_oro.lox`: `123(1 / 0)` da *división por cero*, no
  *solo se pueden llamar funciones*, porque primero se evalúan los argumentos.
- `errores_estaticos/return_global.lox`: el `print` anterior al `return` no
  llega a ejecutarse, porque el error es estático.
- `control/logicos_cortocircuito.lox`: `and`/`or` devuelven el operando que
  decidió y no evalúan el otro.

```bash
go test ./tests -v                 # solo end-to-end, con el detalle de cada script
go test ./tests -update            # regenerar tests/golden/ si el ejemplo cambia a propósito
```

---

## Benchmarks

Comparamos glox contra **otras 8 implementaciones de Lox** escritas en Rust,
C++, Java, Swift, Dart, PHP, Clojure y Python (la de la cátedra), corriendo los
mismos 9 scripts en la misma máquina. Cómo se mide y cómo reproducirlo está en
[`benchmarks/README.md`](benchmarks/README.md); las tablas completas, en
[`benchmarks/results/RESULTS.md`](benchmarks/results/RESULTS.md).

- **Máquina:** Apple M5, 16 GB de RAM, macOS 26 · Go 1.27 · 26/09/2026.
- **Métrica:** mediana del tiempo total del proceso (arranque + scan + parse +
  resolve + ejecución) sobre 3 a 5 corridas, validando que cada corrida
  imprima el resultado correcto.
- **Estrategia de cada implementación:** todas son *tree-walk* como glox,
  salvo rlox y loxx, que compilan a bytecode y ejecutan en una VM. En los
  gráficos aparecen en naranja y con rombo; glox en azul.

### Resultados

**Resumen: cuántas veces más lenta que glox es cada una**, promediando los
cinco benchmarks con carga (fib(28), un `for` de 2 millones de vueltas, 2
millones de llamadas, una closure con estado llamada un millón de veces y un
millón de concatenaciones):

![Tiempo relativo a glox](benchmarks/results/relativo.svg)

| | rlox | loxx | jlox | **glox** | slox | dlox | plox-php | cloxure | plox |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Relativo a glox | 0,12× | 0,15× | 0,77× | **1×** | 5,4× | 6,6× | 21× | 36× | 68× |

**Tiempo por benchmark:**

![Tiempo total por benchmark](benchmarks/results/tiempos.svg)

| Script | glox | plox (cátedra) | glox vs plox |
|---|---:|---:|---:|
| `fib(28)` | 147 ms | 8,56 s | 58× más rápido |
| `for` 2M | 273 ms | 20,5 s | 75× |
| 2M llamadas | 416 ms | 22,9 s | 55× |
| closure con estado 1M | 194 ms | 15,0 s | 77× |
| 1M concatenaciones | 185 ms | 14,3 s | 78× |

**Arranque y memoria:**

![Costo de arranque](benchmarks/results/arranque.svg)

![Memoria residente pico](benchmarks/results/memoria.svg)

**Escalado:** fib(n) para n = 15, 18, 21, 24 y 27:

![Escalado de fib(n)](benchmarks/results/escalado.svg)

**Costo por etapa de glox** (benchmarks de Go, sin el arranque del proceso;
mediana de 5 corridas):

| Qué | Tiempo | Memoria | Alocaciones |
|---|---:|---:|---:|
| Escanear 100 líneas de aritmética (~500 tokens) | 17 µs | 123 KB | 511 |
| Parsear 100 líneas de expresiones | 22 µs | 70 KB | 1.506 |
| Ejecutar `fib(15)` (1.973 llamadas) | 277 µs | 761 KB | 12.453 |
| Ejecutar un `for` de 10.000 vueltas | 1,35 ms | 1,4 MB | 60.008 |
| Pipeline completo de un programa chico | 29 µs | 64 KB | 1.167 |

### Análisis

**1. Frente a la cátedra: entre 55× y 78× más rápido, y entre 2,5 y 3 veces
menos memoria.** plox es un intérprete escrito en un lenguaje interpretado: cada
nodo del AST de Lox que visita se traduce en decenas de instrucciones de
bytecode de CPython, que a su vez se interpretan. glox es código de máquina
nativo recorriendo el mismo árbol. Hay dos niveles de interpretación contra
uno. La diferencia es pareja en todos los benchmarks (55×–78×) justamente
porque es un costo multiplicativo que se paga en cada nodo, sin importar qué
haga el programa. En memoria, glox necesita 11–13 MB contra 32 MB de plox,
casi todo por el runtime: plox ejecutando `print "ok";` ya ocupa 32 MB, más
que glox ejecutando fib(28).

**2. Las implementaciones a bytecode son 7–8× más rápidas que glox, y los
benchmarks de Go muestran por qué.** Cada llamada a función de glox hace unas
**6 alocaciones en el heap** (12.453 para las 1.973 llamadas de fib(15)).
El perfil de memoria dice de dónde salen:

| Origen | Alocaciones |
|---|---:|
| Crear el `Environment` de la llamada (struct + mapa) | 35% |
| Guardar el parámetro en ese mapa (`Define`) | 18% |
| Slice de argumentos y números que pasan a `any` | 29% |
| El `returnSignal` del `return` | 18% |

Además, cada lectura de una variable local es una búsqueda por string en un
mapa, después de saltar `depth` entornos. El `for` paga lo mismo en cada
vuelta, porque el cuerpo es un bloque y cada bloque crea su `Environment`
(60.008 alocaciones para 10.000 vueltas). Una VM de pila como la de rlox o
loxx no crea nada de eso: los parámetros y las locales son posiciones fijas del
stack, calculadas al compilar, y leer una variable es indexar un arreglo. Es
exactamente lo que vamos a construir en el Bloque 2, y estos números son la
línea de base contra la que lo vamos a medir.

**3. jlox le gana a glox en programas largos y pierde en los cortos: es el
JIT.** jlox también es tree-walk, pero la JVM compila a código de máquina los
métodos del intérprete que más se usan mientras el programa corre. La curva de
escalado lo muestra: con fib(15), jlox tarda 29 ms contra 2,5 ms de glox (casi
todo es arranque de la JVM, 22 ms); con fib(27) ya lo alcanza (78 ms contra
92 ms). Lo mismo se ve en el uso de CPU: jlox usa entre 1,3 y 1,8 núcleos de
CPU por segundo de reloj, porque el JIT y el GC corren en paralelo al programa;
glox usa ~1,05. glox no tiene JIT: el código que ejecuta es el que compiló
`go build`, así que su rendimiento es el mismo desde la primera línea. Esa es
también la razón de que arranque en 2 ms y sea de los más rápidos en scripts
cortos. El costo de la JVM aparece en memoria: jlox necesita 129 MB para fib(28)
y cloxure (Clojure sobre la JVM) 333 MB, contra 13 MB de glox.

**4. Cómo se implementa `return` importa, y mucho.** dlox es 66× más lento que
glox en fib(28), pero apenas 1,1× en el `for` de 2 millones. La diferencia es
que fib hace llamadas y el `for` no. dlox, cloxure, jlox y plox-php
implementan `return` lanzando una excepción que atrapa la llamada, como plox.
Para aislar ese costo corrimos el mismo bucle de 300.000 llamadas en dos
versiones, una que devuelve el resultado con `return` y otra que lo acumula
en una variable global sin `return`:

| | glox | slox | plox | jlox | plox-php | cloxure | dlox |
|---|---:|---:|---:|---:|---:|---:|---:|
| Con `return` / sin `return` | **1,02×** | 1,01× | 1,05× | 1,35× | 1,57× | 5,1× | 9,4× |

En glox y en slox, `return` es un valor que se devuelve (el `returnSignal`
de glox; un caso de `Result` en slox) y cuesta lo mismo que cualquier otra
sentencia. En dlox, cada `throw` de Dart captura el stack de la llamada, y
hacerlo una vez por cada llamada de Lox multiplica el costo por 9. En cloxure,
la excepción es un `ex-info` de la JVM, con stack trace completo. jlox, en
cambio, construye su excepción sin stack trace y por eso paga poco. En plox el
efecto casi no se ve (1,05×) porque en CPython lanzar una excepción es barato
comparado con todo lo demás que hace el intérprete. Esto confirma con números
la decisión de diseño 2: usar `panic` o una excepción para el `return` no solo
iba contra el estilo de Go, también habría sido medible en cada llamada.

**5. Arranque: glox tarda 2,2 ms en ejecutar `print "ok";`**, contra 53 ms de
plox, 22 ms de jlox y 199 ms de cloxure. Un binario de Go es un ejecutable
nativo que trae su runtime (GC, scheduler) adentro y lo inicializa en
microsegundos: no hay que cargar una máquina virtual ni módulos. Por eso los tres scripts originales de la primera comparativa
(`fib`, `loops`, `closures`, de 7 a 36 ms en glox) miden en buena parte
arranque, y por eso sumamos las versiones con más carga: con `loops.lox`
(100.000 vueltas), dlox parece tan rápido como glox (24 ms contra 17 ms), y
recién con 2 millones de vueltas se ve la diferencia real de ejecución.

**6. Lo que no se pudo medir.** rlox no termina `strings.lox`: entre 2.000 y
4.000 iteraciones corta con un error interno (`UnexpectedValue`) de su VM, un
bug de rlox que aparece cuando crece la cantidad de strings creados. loxx
imprime los números con 6 cifras significativas (`2e+12` en vez de
`1999999000000`); verificamos aparte que el valor calculado es exacto, así que
sus corridas se cuentan como válidas. rlox reserva de entrada 4 GB de memoria
virtual para su heap, y eso explica sus ~50 MB residentes aunque el programa no
haga nada.

**En una frase:** entre los intérpretes tree-walk, glox es el más rápido en
todos los scripts cortos y en `counter`, y queda segundo detrás de jlox en los
otros cuatro benchmarks largos, cuando el JIT de la JVM tiene tiempo de
calentar. Contra la cátedra es 55–78× más rápido y usa entre 2,5 y 3 veces
menos memoria. La brecha de 7–8× contra las implementaciones a bytecode se
explica por las ~6 alocaciones y las búsquedas en mapas de cada llamada, que es
justo lo que el Bloque 2 viene a sacar.

---

## Cómo trabajamos

- **Un plan por fases alineado a las clases.** Dividimos el TP en fases, una
  por semana de cursada (Fase 1 escaneo, Fase 2 parseo, Fase 3 ejecución y
  control de flujo, Fase 4 funciones y resolver, Fase 5 cierre), y cada fase en
  partes chicas con un criterio de "terminado" verificable. La regla fue no
  implementar nada de una clase que todavía no se había dado: si sobraba
  tiempo, iba a tests y documentación de lo ya visto.
- **Una rama por tema y Pull Requests a `main`.** `scanning`, `interpreter`,
  `state`, `control-flow`, `functions`, `resolver`, `benchmarks` y
  `demo-and-docs`, cada una integrada por PR (#1 a #8 al día de hoy). Nos
  repartimos las fases de a bloques: Federico hizo el setup, el parser, las
  funciones y el resolver; Franco el scanner, el intérprete con estado y
  control de flujo, y los benchmarks; el cierre (pruebas end-to-end, comparación
  ampliada y este README) lo hicimos entre los dos.
- **Commits chicos y descriptivos,** uno por parte del plan, en español.
- **Cada parte cierra en verde:** `go build`, `go vet`, `gofmt -l .`,
  `go test ./...` y, desde que hubo funciones, las pruebas de la cátedra.
- **Las decisiones se escriben cuando se toman.** Cada vez que había dos
  caminos razonables (`panic` o error explícito en el parser, Visitor o
  `type switch`, cómo implementar `return`), dejamos anotado cuál elegimos y
  por qué. De ahí sale la sección de diferencias de este README.
- **Lo que no salió a la primera también quedó registrado.** Por ejemplo, al
  incorporar las pruebas de la cátedra notamos que `script.py` decide "Todo OK"
  mirando solo stdout: con errores de parseo (que van a stderr) daba un falso
  verde. Desde entonces revisamos la salida real de cada aserción, no solo el
  veredicto del script. Y un test del resolver que suponíamos válido
  (`{ var a = 1; { var a = a; } }`) resultó ser un error de verdad en Lox: se
  corrigió el test, no el resolver.

---

## Implementaciones usadas en los benchmarks

| Implementación | Lenguaje | Estrategia | Repositorio |
|---|---|---|---|
| Rlox | Rust | AST → bytecode + VM | https://github.com/Darksecond/lox |
| Slox | Swift | tree-walk | https://github.com/alexito4/slox |
| Jlox | Java | tree-walk | https://github.com/ryanq/jlox |
| Cloxure | Clojure | tree-walk | https://github.com/ceronman/cloxure |
| Loxx | C++ | bytecode + VM | https://github.com/mspraggs/loxx |
| Plox-master | PHP | tree-walk | https://github.com/minirop/plox |
| Dlox | Dart | tree-walk | https://github.com/sma/lox |
| Plox | Python | tree-walk (implementación de la cátedra) | https://github.com/FdelMazo/plox |
