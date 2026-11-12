# glox — Lox en Go

Implementación del lenguaje [Lox](https://craftinginterpreters.com/the-lox-language.html)
en Go, para el TP de **Lenguajes y Compiladores I (FIUBA)**, Opción I.

> **Este es el informe de la entrega final.** El de la entrega parcial quedó,
> tal como se entregó el 28/09, en [`entrega-parcial/README.md`](entrega-parcial/README.md),
> junto con los resultados de benchmarks de ese momento. Este README cubre
> todo el TP, con el foco en lo que se agregó para la entrega final.

| | |
|---|---|
| **Integrantes** | Federico Honda (110766) · Franco Bustos (110759) |
| **Entrega** | Final (16/11/2026) |

## Índice

1. [Cómo compilar y ejecutar](#cómo-compilar-y-ejecutar)
2. [Arquitectura](#arquitectura)
3. [Del tree-walk al bytecode](#del-tree-walk-al-bytecode)
4. [Clases y herencia](#clases-y-herencia)
5. [Funcionalidad extra: listas](#funcionalidad-extra-listas)
6. [Diferencias con la implementación de la cátedra](#diferencias-con-la-implementación-de-la-cátedra)
7. [Pruebas](#pruebas)
8. [Benchmarks](#benchmarks)
9. [Programas de uso real](#programas-de-uso-real)
10. [Limitaciones conocidas](#limitaciones-conocidas)
11. [Cómo trabajamos](#cómo-trabajamos)

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
./glox examples/gastos.lox       # programa de uso real: resumen de gastos del mes
./glox examples/banco.lox        # programa de uso real: cuentas con herencia
```

Sin compilar el binario también se puede usar `go run ./cmd/glox examples/hello.lox`.

### REPL

Sin argumentos, glox abre un REPL. Las variables, funciones y clases se
conservan entre líneas, y un error no cierra la sesión:

```text
$ ./glox
> class Punto { init(x, y) { this.x = x; this.y = y; } }
> var ps = [Punto(1, 2), Punto(3, 4)];
> print ps[1].x + ps[0].y;
5
> print ps[2];
[línea 1] Error en tiempo de ejecución en '[': índice fuera de rango: 2 (la lista tiene 2 elementos).
> push(ps, Punto(5, 6));
> print len(ps);
3
```

Se sale con `Ctrl+D` (fin de la entrada).

### Modos de inspección

```bash
./glox --scanning    prog.lox   # lista de tokens: TIPO 'lexema' literal
./glox --parsing     prog.lox   # AST en notación prefija, una sentencia por línea
./glox --disassemble prog.lox   # bytecode del programa (y de cada función) y después su ejecución
./glox --tree-walk   prog.lox   # ejecuta con el intérprete tree-walk de la entrega parcial
```

Sin flags, `glox` compila el programa a bytecode y lo ejecuta en una máquina
virtual de pila. El intérprete tree-walk de la entrega parcial sigue
disponible con `--tree-walk`, para comparar los dos (no tiene clases ni
listas: las rechaza con un error). Los cuatro modos también funcionan en el
REPL.

```text
$ echo 'var x = 2 + 3 * 4; print x;' > /tmp/t.lox
$ ./glox --parsing /tmp/t.lox
(var x = (+ 2 (* 3 4)))
(print x)
$ ./glox --disassemble /tmp/t.lox
== COMPILETIME ==
0000    1 OP_CONSTANT         0 '2'
0002    | OP_CONSTANT         1 '3'
0004    | OP_CONSTANT         2 '4'
0006    | OP_MULTIPLY
0007    | OP_ADD
0008    | OP_DEFINE_GLOBAL    3 'x'
0010    | OP_GET_GLOBAL       3 'x'
0012    | OP_PRINT
0013    | OP_RETURN
== RUNTIME ==
14
```

Las columnas del listado son: posición del opcode, línea del fuente (`|` si no
cambió), instrucción y, si lleva operando, el operando y el valor al que
refiere. Es el mismo árbol que muestra `--parsing`, aplanado en notación
postfija: los operandos antes que el operador. Cada función, método e
inicializador compila a su propio chunk y se lista después del programa.

### Códigos de salida

| Código | Significado |
|---|---|
| `0` | El programa terminó bien |
| `64` | Uso incorrecto del CLI (más de un archivo) |
| `65` | Error estático: léxico, de sintaxis, semántico o de compilación a bytecode. **No se ejecuta ninguna línea** |
| `70` | Error en tiempo de ejecución. Lo anterior al error sí se ejecutó |

Los errores van a `stderr` con la línea donde ocurrieron:

```text
[línea 3] Error de sintaxis en ';': se esperaba una expresión.
[línea 5] Error semántico en 'super': no se puede usar 'super' en una clase sin superclase.
[línea 7] Error en tiempo de ejecución en '+': los operandos de '+' deben ser dos números o dos cadenas, se obtuvo: a y 1
```

### Tests y benchmarks

```bash
go test ./...                               # unitarios + end-to-end
go test -cover ./internal/...               # cobertura por paquete
python3 ./real-tests/script.py ./glox       # pruebas de la cátedra (con la VM)
go test -run '^$' -bench . -benchmem ./internal/...   # benchmarks de Go
```

La comparación contra otras implementaciones y otros lenguajes tiene su
propia guía en [`benchmarks/README.md`](benchmarks/README.md).

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
       │
       ├────────────────────────────────┐ con --tree-walk
       ▼                                ▼
 [4. Compiler] ──► errores de     [Interpreter] ──► errores de runtime (70)
       │           compilación (65)  (entrega parcial)
       │ *bytecode.Chunk  (instrucciones + constantes + líneas)
       ▼
 [5. VM]       ──────────► errores de runtime (70)
```

Scanner, parser y resolver son los mismos para los dos caminos: lo que cambia
es qué se hace con el árbol.

| Paquete | Qué hace |
|---|---|
| `cmd/glox` | CLI: REPL, archivo, `--scanning`, `--parsing`, `--disassemble`, `--tree-walk` y códigos de salida. Arma el pipeline y corta en la primera etapa con errores. |
| `internal/token` | `TokenType` (enum con `iota`) y `Token{Type, Lexeme, Literal, Line}`. |
| `internal/scanner` | Análisis léxico *maximal munch*. Acumula todos los errores léxicos del archivo en vez de cortar en el primero. |
| `internal/ast` | Nodos del AST como structs de datos puros, separados por tema (`stmt.go`, `expr_variable.go`, `expr_call.go`, `expr_class.go`, `expr_list.go`, ...). Cada nodo se imprime en notación prefija (`String()`). |
| `internal/parser` | Parser recursivo descendente: una función por regla de la gramática. Desazucara `for` a `while`. Se recupera de los errores sincronizando hasta la próxima sentencia. |
| `internal/resolver` | Análisis semántico estático: distancia de scope de cada variable local, y los errores que se pueden detectar sin ejecutar (`return` fuera de una función, `this` fuera de una clase, `super` sin superclase, una clase que hereda de sí misma, ...). |
| `internal/interpreter` | El intérprete tree-walk de la entrega parcial: `type switch`, `Environment` encadenado, closures. Queda como estaba, para comparar y como oráculo de los tests. |
| `internal/bytecode` | El camino por defecto. `Chunk` (bytecode + constantes + línea de cada byte), `OpCode`, disassembler, el compilador (`compiler.go`, más un pase previo en `captures.go` que detecta qué variables captura una closure), la VM de pila (`vm.go`), los objetos de runtime (`object.go`: funciones, closures, celdas, clases, instancias, métodos ligados, listas y nativas) y las funciones nativas (`natives.go`). |

### Gramática implementada

```
program     → declaration* EOF ;
declaration → classDecl | funDecl | varDecl | statement ;
classDecl   → "class" IDENTIFIER ( "<" IDENTIFIER )? "{" function* "}" ;
funDecl     → "fun" function ;
function    → IDENTIFIER "(" parameters? ")" block ;
varDecl     → "var" IDENTIFIER ( "=" expression )? ";" ;
statement   → exprStmt | forStmt | ifStmt | printStmt | returnStmt | whileStmt | block ;
expression  → assignment ;
assignment  → ( call "." )? IDENTIFIER "=" assignment
            | call "[" expression "]" "=" assignment
            | logic_or ;
logic_or    → logic_and ( "or" logic_and )* ;
logic_and   → equality ( "and" equality )* ;
equality    → comparison ( ( "!=" | "==" ) comparison )* ;
comparison  → term ( ( ">" | ">=" | "<" | "<=" ) term )* ;
term        → factor ( ( "-" | "+" ) factor )* ;
factor      → unary ( ( "/" | "*" | "%" ) unary )* ;
unary       → ( "!" | "-" ) unary | call ;
call        → primary ( "(" arguments? ")" | "." IDENTIFIER | "[" expression "]" )* ;
primary     → NUMBER | STRING | "true" | "false" | "nil" | IDENTIFIER | "(" expression ")"
            | "this" | "super" "." IDENTIFIER | "[" arguments? "]" ;
```

Respecto de Lox estándar, lo nuevo es `%` (desde la entrega parcial) y todo lo
que tiene `[` y `]`.

---

## Del tree-walk al bytecode

La consigna pedía **reconvertir** el TP, no reescribirlo. Lo tomamos
literal: el compilador arranca del mismo AST que ya armaba el parser, después
del mismo resolver, y el intérprete de la entrega parcial quedó intacto al
lado.

**Compilar desde el AST, no en un solo paso.** `clox`, el compilador del
libro, parsea y emite bytecode al mismo tiempo, sin árbol. Nosotros ya
teníamos el árbol y el resolver, así que el compilador es un recorrido más:
la precedencia ya viene resuelta en la forma del árbol, los errores
estáticos ya los detectó el resolver, y antes de emitir se puede mirar el
programa entero. Eso último fue clave para las closures (ver abajo). El
costo es tener el AST completo en memoria, que para programas de este tamaño
no se nota.

**El orden en que se hizo** fue incremental: expresiones aritméticas y la
VM de pila; variables globales, locales en slots y saltos; funciones,
frames, recursión y closures. Con eso la VM corrió todo el lenguaje de la
entrega parcial, pasó las pruebas de la cátedra y pasó a ser el camino por
defecto. Después vinieron la optimización y las clases, ya solo en la VM.

**Qué cambió de representación:**

| | Tree-walk (entrega parcial) | VM (entrega final) |
|---|---|---|
| Variable local | Búsqueda por nombre en un `map` del `Environment`, `depth` entornos más arriba | Índice de slot en la pila, calculado al compilar (`OpGetLocal 3`) |
| Variable global | `map[string]Value` | Un arreglo de slots, con el índice cacheado por constante en cada chunk (la optimización, ver [Benchmarks](#benchmarks)) |
| Llamada | Un `Environment` nuevo con su mapa, más un slice de argumentos | Los argumentos ya están en la pila: pasan a ser los slots del frame nuevo, sin copiarse |
| `return` | Un `*returnSignal` que sube por el canal de `error` | `OpReturn`: corta la pila en la base del frame de una vez |
| Closure | La función guarda un puntero a su `Environment` | La función guarda las celdas de las variables que captura |

**Las variables capturadas viven en celdas del heap desde que se declaran.**
Es la decisión más grande del bloque y donde más nos separamos de `clox`. En
`clox` una variable vive en la pila mientras su función corre, y si una
closure la captura, se "muda" al heap cuando la función termina (los
*upvalues*): hay que llevar la lista de upvalues abiertos y cerrarlos al
salir de cada scope y en cada `return`. Nosotros, como compilamos desde el
AST, sabemos *antes de emitir* qué variables captura alguna función anidada
(un pase previo, `captures.go`), y esas se guardan en una celda desde su
declaración (`OpBox`). No hay nada que cerrar: el GC de Go mantiene viva la
celda mientras alguna closure la use. Y en Go la otra opción tenía un
problema concreto: nuestra pila es un slice que crece con `append`, así que un
upvalue abierto con un puntero a la pila quedaría apuntando a la copia vieja
después de una realocación. Las variables que nadie captura (las de `fib`,
las de cualquier loop) siguen siendo slots comunes y no pagan nada.

**El tree-walk quedó como oráculo.** Las reglas de cada operador están
escritas dos veces, una en cada backend. Lo que garantiza que no se desvíen
es compararlos: la misma expresión tiene que dar el mismo valor o el mismo
error (mensaje y línea incluidos) en los dos, sobre 60 casos a mano y 3000
expresiones generadas al azar, y todos los scripts de `tests/lox/` que no
usan clases ni listas se corren por los dos backends.

**Clases y listas, solo en la VM.** Para cuando llegaron, la VM ya era el
camino por defecto y la entrega final se evalúa sobre ella. Implementarlas
también en el tree-walk era duplicar la parte más grande del bloque en un
backend que ya no se usa. El front-end sí las conoce (es compartido), así
que el tree-walk las rechaza con un error claro (`las clases solo están
disponibles en la VM`) en vez de uno genérico.

---

## Clases y herencia

```lox
class Animal {
  init(nombre) {
    this.nombre = nombre;
  }
  sonido() { return "..."; }
  hablar() { return this.nombre + " dice " + this.sonido(); }
}

class Perro < Animal {
  sonido() { return "guau"; }
}

class Cachorro < Perro {
  init(nombre) { super.init(nombre + " (cachorro)"); }
  sonido() { return super.sonido() + " guau"; }
}

print Perro("Firulais").hablar(); // Firulais dice guau
print Cachorro("Toby").hablar();  // Toby (cachorro) dice guau guau
```

Es la semántica de Lox: campos dinámicos (aparecen al asignarlos), llamar a
la clase crea la instancia y corre `init`, `init` siempre devuelve `this`, un
campo tapa a un método del mismo nombre, `obj.metodo` sin llamar devuelve el
método ligado a `obj`, y `super.m()` llama al `m` de la superclase.

**Cómo está hecho:**

- **Una clase y una instancia son objetos de Go**, siempre por puntero:
  `Class{Name, Methods map[string]*Closure}` e `Instance{Class, Fields map[string]Value}`.
  Por puntero para que el `==` de Lox compare identidad (y porque comparar con
  `==` dos `any` que guardan un struct con un mapa adentro hace `panic` en Go).
- **`this` es la variable local del slot 0.** En cualquier llamada, el slot 0
  del frame es el lugar de la pila donde estaba el valor llamado. En un método,
  la VM pone ahí al receptor, así que `this` compila a `OpGetLocal 0`, sin
  instrucción propia. Si una función anidada en el método usa `this`, se
  captura como cualquier otra variable.
- **`obj.m(args)` es una sola instrucción, `OpInvoke`**, que busca el método y
  lo llama directo. Compilarlo como "leer la propiedad y después llamar" crea
  un método ligado que se descarta en la instrucción siguiente.
- **Al heredar se copian los métodos.** `OpInherit` copia el mapa de métodos
  de la superclase al de la subclase, y después se agregan los propios, que
  pisan a los heredados (eso es el override). Buscar un método heredado de
  tres niveles arriba cuesta una sola consulta al mapa. Es correcto porque en
  Lox una clase no cambia después de declarada.
- **`super` es una variable local escondida que los métodos capturan.** Al
  compilar una clase con superclase, la superclase queda en la pila como una
  local llamada `super`, en un scope que envuelve a los métodos. Un
  `super.m()` es leer una variable de afuera, con el mismo mecanismo de
  closures de las funciones:

  ```
  == COMPILETIME ==                   == <fn hola> ==   (de B)
  ...                                 OP_CONSTANT       'B y '
  OP_GET_GLOBAL      'A'              OP_GET_LOCAL      0     ; this
  OP_GET_GLOBAL      'B'              OP_GET_CAPTURED   0     ; super
  OP_INHERIT    ; A queda en la pila: es `super`
  OP_BOX        0      ; en una celda, porque hola la captura
  OP_GET_GLOBAL      'B'              OP_SUPER_INVOKE   (0 args) 'hola'
  OP_CLOSURE    <fn hola>             OP_ADD
                local 0               OP_RETURN
  OP_METHOD     'hola'
  OP_POP        ; saca B
  OP_POP        ; cierra el scope de `super`
  ```

  Importa por el caso de tres niveles: en `C < B < A`, un `super.m()` escrito
  en `B` y ejecutado sobre una `C` tiene que ir a `A`. Si se resolviera
  mirando la clase de la instancia, iría a `B` y se llamaría a sí mismo para
  siempre. Con la variable capturada, `super` es la superclase de la clase
  donde está *escrito* el método. Y un `super` dentro de una función anidada
  en un método anduvo de entrada, porque es el mismo caso que una closure
  dentro de otra.

**Errores:**

| Error | Cuándo | Código |
|---|---|---|
| `this` fuera de una clase | estático | 65 |
| `return valor;` adentro de `init` | estático | 65 |
| `super` fuera de una clase, o en una clase sin superclase | estático | 65 |
| Una clase que hereda de sí misma | estático | 65 |
| La superclase no es una clase (`var A = 3; class B < A {}`) | runtime | 70 |
| Leer o asignar una propiedad de algo que no es una instancia | runtime | 70 |
| Una propiedad o un método que no existe (también con `super`) | runtime | 70 |
| Cantidad de argumentos distinta a la de `init` | runtime | 70 |

Los estáticos los detecta el resolver, que ya corría antes de ejecutar: es
una ventaja de tener el análisis separado del compilador.

---

## Funcionalidad extra: listas

```lox
var nums = [3, 1, 2];
push(nums, 10);
nums[0] = nums[0] * 2;
print nums;          // [6, 1, 2, 10]
print len(nums);     // 4

var matriz = [[1, 2], [3, 4]];
print matriz[1][0];  // 3
print len("año");    // 3
```

Elegimos listas porque era lo que más extrañamos al escribir los programas de
uso real de la entrega parcial: Lox no tiene ninguna estructura de datos, y
los dos programas tuvieron que armar listas enlazadas con closures
(`cons(cabeza, cola)` devolviendo una función). Además, a diferencia de un
ternario o un `break`, toca todas las etapas: sintaxis nueva, una forma de
asignación nueva, un objeto de runtime, instrucciones y funciones nativas, que
la VM no tenía.

- **Sintaxis:** literal `[a, b, c]`, índice `xs[i]` y asignación `xs[i] = v`
  (que es una expresión, como cualquier asignación). El `[` aparece en dos
  lugares de la gramática: en `primary` abre un literal y después de una
  expresión es un índice, así que `[1, 2][0]`, `f()[0]`, `this.items[i]` y
  `m[1][0]` salen solos.
- **`len(x)` y `push(lista, x)` son funciones nativas**, escritas en Go y
  definidas como globales al arrancar la VM. Una nativa no abre un frame:
  recibe los argumentos que ya están en la pila y deja el resultado en el
  lugar del callee. Preferimos nativas a métodos (`xs.push(1)`) porque así
  la lista es un valor más y el modelo de objetos no cambia: con métodos,
  `OpInvoke` tendría que aceptar receptores que no son instancias.
- **Índices estrictos:** un entero entre `0` y `len - 1`; cualquier otra
  cosa es un error de runtime (`índice fuera de rango: 3 (la lista tiene 3
  elementos).`). Nada de índices negativos como Python ni de `nil` al pasarse
  como JavaScript: por la regla de oro de Lox, si `xs[-1]` no tiene un
  significado obvio, tiene que ser un error.
- **Las listas son referencias:** `var b = a; push(b, 1);` modifica `a`, y
  `==` compara identidad (`[1] == [1]` es `false`), igual que con las
  instancias.
- **Regla de oro en la asignación:** `a[5] = 1 / 0` sobre una lista vacía da
  *división por cero*, no *índice fuera de rango*: se evalúan la lista, el
  índice y el valor, y recién ahí se chequea.
- **`print` muestra los elementos como los mostraría `print`**
  (`[1, dos, nil, [3]]`), y una lista que se contiene a sí misma imprime
  `[...]` en esa posición en vez de recursar hasta el stack overflow.

Las tres instrucciones nuevas son `OpList n` (arma la lista con los `n`
valores del tope de la pila), `OpGetIndex` y `OpSetIndex`.

---

## Diferencias con la implementación de la cátedra

La referencia para el lenguaje es **plox**, el intérprete tree-walk en Python
de la cátedra, y para la parte de bytecode, **clox**, el compilador en C del
libro, que es el que se sigue en las clases de compilación.

### Qué cambia en el lenguaje

| | plox | glox |
|---|---|---|
| Clases y herencia | No tiene (no las reconoce el scanner) | Sí, solo en la VM |
| Listas | No | Sí: `[a, b]`, `xs[i]`, `xs[i] = v`, `len`, `push` |
| Funciones nativas | Ninguna | `len` y `push` |
| Operador `%` | Sí | Sí (`math.Mod`, misma precedencia que `*` y `/`) |
| `print 1 / 0;` y `print 1 % 0;` | Error de runtime | Error de runtime |
| Varios errores estáticos en un archivo | Reporta el primero y corta | **Reporta todos** (léxicos, de sintaxis y semánticos) y no ejecuta nada |
| Código de salida ante un error | `0` en todos los casos | **`65`** si es estático, **`70`** si es de runtime |
| Recursión infinita | `RecursionError` de Python | Error de runtime: `desbordamiento de pila (más de 100000 llamadas anidadas).` |
| `print 3;` y `print 4999950000;` | `3.0` y `4999950000.0` | `3` y `4.99995e+09` (formato `%v` de Go) |
| Mensajes de error | Inglés, sin número de línea en runtime | Español, con línea y lexema |

### Qué nos obligó (o nos permitió) hacer distinto Go

De la entrega parcial (el detalle está en el [informe parcial](entrega-parcial/README.md#qué-nos-obligó-o-nos-permitió-hacer-distinto-go)):

- **`type switch` en vez de Visitor** para recorrer el AST, tanto en el
  intérprete como en el compilador.
- **`return` como un valor de error y no como excepción** en el tree-walk:
  medimos que en las implementaciones que usan una excepción, `return` cuesta
  hasta 9 veces más.
- **`panic`/`recover` solo adentro del parser**, para volver a la sentencia,
  sincronizar y seguir reportando errores.
- **El resolver no conoce al intérprete:** es una función pura que devuelve un
  mapa, así que el compilador lo usa igual que el intérprete.
- **Valores como `any`**: cada operación pregunta explícitamente el tipo, sin
  conversiones implícitas prestadas del lenguaje anfitrión.

Nuevas de la entrega final:

**1. Celdas en vez de upvalues.** Explicado arriba en
[Del tree-walk al bytecode](#del-tree-walk-al-bytecode): el AST nos deja
saber de antemano qué se captura, el GC nos deja no decidir cuándo liberar, y
la pila como slice de Go hacía frágil apuntar adentro de ella. Es la misma
idea que usa el compilador de Go con sus propias closures: una variable
capturada que puede sobrevivir a su función se aloca en el heap.

**2. El frame en curso vive en la VM, no en un puntero.** `clox` guarda un
puntero al `CallFrame` actual. En Go, un puntero a un elemento del slice de
frames quedaría colgado cuando el slice crece, así que el `chunk`, el `ip`, la
`base` y la closure del frame actual se copian a campos de la VM, y el `ip` se
guarda en el frame recién al hacer una llamada. De paso, las instrucciones que
no llaman ni retornan no se enteraron de que existen los frames.

**3. Pila y frames que crecen, con un tope.** `clox` usa arreglos de tamaño
fijo y da *stack overflow* al llenarlos. Nosotros usamos slices, que crecen
solos; para que una recursión infinita no termine agotando la memoria, hay un
tope de 100.000 llamadas anidadas que da un error de runtime normal (en el
tree-walk, en cambio, termina con el *stack overflow* fatal del runtime de Go).

**4. Los objetos los recolecta el GC de Go.** En `clox`, cada tipo de objeto
nuevo (clase, instancia, método ligado) obliga a enseñarle al recolector cómo
recorrerlo, y las tablas de métodos y campos son tablas de hash propias con
strings internados. Acá los métodos y los campos son `map` de Go, los ciclos
(una instancia que se guarda a sí misma en un campo, una lista que se contiene)
los resuelve el GC, y sumar las listas no tocó nada de eso.

**5. Errores estáticos en el resolver, no en el compilador.** `clox` detecta
`this` fuera de una clase o `super` sin superclase en medio del compilador de
un solo paso. Acá ya estaba el resolver, que corre antes y es compartido: los
errores nuevos de clases se agregaron una sola vez y valen para los dos
backends.

---

## Pruebas

| Nivel | Dónde | Qué cubre |
|---|---|---|
| Pruebas de la cátedra | `real-tests/` | 5 archivos, 57 aserciones: aritmética, strings, control de flujo, funciones, el bug de closures, una máquina de Minsky y FizzBuzz. **5/5 OK con la VM y 5/5 con `--tree-walk`.** |
| Unitarias | `internal/*/*_test.go` | Cada paquete por separado: parser ~98,5%, resolver ~92,7%, scanner ~93,3%, interpreter ~92,6%, bytecode ~89,5% de cobertura. En `bytecode`, archivos por tema (`compiler`, `vm`, `functions`, `classes`, `inheritance`, `lists`), con listados exactos del bytecode esperado en los casos delicados. |
| Tree-walk vs. bytecode | `internal/bytecode/treewalk_test.go` | El intérprete como oráculo de la VM: 60 casos borde y 3000 expresiones al azar con semilla fija, comparando valor o mensaje de error completo. |
| End-to-end | `tests/` | El binario real sobre 53 scripts `.lox` (corridos por los dos backends) y 31 programas para `--disassemble`, más el REPL, los modos de inspección, los códigos de salida, la salida completa de los tres programas de `examples/` y el resultado de cada benchmark. |

Los scripts de `tests/lox/` declaran lo que esperan en comentarios, y
`tests/e2e_test.go` compila glox y los verifica:

```lox
// exit: 70
// stderr: [línea 6] Error en tiempo de ejecución en '[': índice fuera de rango: 3 (la lista tiene 3 elementos).
var a = [1, 2, 3];
print a[2]; // expect: 3

print a[3];
print "no llega";
```

Están agrupados por tema: `expresiones/`, `variables/`, `control/`,
`funciones/`, `clases/`, `listas/`, `errores_estaticos/` (código 65: no se
ejecuta nada) y `errores_runtime/` (código 70: se ejecuta hasta el error). Los
de `clases/` y `listas/` se corren por la VM, y por el tree-walk se verifica
que los rechace con el error explicativo. Algunos que vale la pena mirar:

- `clases/super_tres_niveles.lox`: el caso de `C < B < A` donde un `super`
  mal resuelto se llamaría a sí mismo para siempre.
- `errores_estaticos/super_sin_superclase.lox`: dos errores semánticos en el
  mismo archivo, reportados los dos.
- `funciones/closure_bug.lox`: el caso clásico que arregla el resolver, que
  en la VM se resuelve en compilación.
- `errores_runtime/regla_de_oro.lox`: `123(1 / 0)` da *división por cero*,
  no *solo se pueden llamar funciones*, porque primero se evalúan los argumentos.

Los de `tests/bytecode/` se corren con `--disassemble` y declaran además las
instrucciones que tiene que mostrar el listado, en orden:

```lox
// ops: OP_CONSTANT OP_CONSTANT OP_LIST OP_DEFINE_GLOBAL OP_GET_GLOBAL OP_CONSTANT OP_GET_GLOBAL OP_CONSTANT OP_GET_INDEX OP_SET_INDEX OP_POP OP_GET_GLOBAL OP_PRINT OP_RETURN
// expect: [2, 2]
var a = [1, 2];
a[0] = a[1];
print a;
```

```bash
go test ./tests -v                 # solo end-to-end, con el detalle de cada script
go test ./tests -update            # regenerar tests/golden/ si un ejemplo cambia a propósito
```

---

## Benchmarks

La consigna pide tres comparaciones: contra la entrega parcial (cómo mejoró
la implementación con el desarrollo), contra otras implementaciones de Lox, y
las mismas ideas en otros lenguajes. Las tres están acá. Cómo se mide y cómo
reproducirlo está en [`benchmarks/README.md`](benchmarks/README.md); las
tablas completas, en [`benchmarks/results/RESULTS.md`](benchmarks/results/RESULTS.md).

- **Máquina:** Apple M5, 16 GB de RAM, macOS 26 · Go 1.27 (`go.mod` pide 1.26+) · 08/11/2026. La
  misma de la entrega parcial.
- **Métrica:** mediana del tiempo total del proceso (arranque + scan + parse +
  resolve + compilación + ejecución) sobre 3 a 5 corridas, validando que cada
  corrida imprima el resultado correcto.
- **Scripts:** los mismos 9 de la entrega parcial, sin tocar, más
  `metodos.lox` (clases) que se sumó ahora.
- **Tres versiones de glox, medidas en la misma corrida:** `glox-parcial` (el
  tree-walk entregado el 28/09, compilado desde el commit `6f06574`),
  `glox-vm-sin-opt` (la primera versión de la VM, antes de la optimización) y `glox`
  (la versión final).

### glox a lo largo del desarrollo

![glox a lo largo del desarrollo](benchmarks/results/evolucion.svg)

| Script | Entrega parcial (tree-walk) | VM sin optimizar | VM final | Parcial → final |
|---|---:|---:|---:|---:|
| `fib(28)` | 146 ms | 41 ms | 41 ms | **3,6×** |
| `for` 2M | 271 ms | 93 ms | 70 ms | **3,9×** |
| 2M llamadas | 418 ms | 98 ms | 84 ms | **5,0×** |
| Closure con estado 1M | 194 ms | 64 ms | 50 ms | **3,9×** |
| 1M concatenaciones | 182 ms | 95 ms | 66 ms | **2,8×** |

**La VM es entre 2,8 y 5 veces más rápida que el tree-walk** (3,7× en
promedio geométrico), con el mismo lenguaje, el mismo front-end y el mismo
GC. La diferencia está en qué cuesta cada operación, y los benchmarks de Go
lo muestran por etapa, sin el arranque del proceso:

| Programa | Tree-walk | VM | Alocaciones tree-walk | Alocaciones VM |
|---|---:|---:|---:|---:|
| `fib(15)` (1.973 llamadas) | 282 µs | 73 µs | 12.453 | 2.598 |
| `for` de 10.000 vueltas | 1,35 ms | 340 µs | 60.008 | 20.010 |
| Closure, 1.000 llamadas | 176 µs | 46 µs | 9.015 | 2.016 |

En la entrega parcial habíamos medido de dónde salían las ~6 alocaciones por
llamada del tree-walk: el `Environment` de la llamada y su mapa (35%), guardar
el parámetro en ese mapa (18%), el slice de argumentos (29%) y el
`returnSignal` del `return` (18%). La VM no hace ninguna de esas cosas: los
argumentos ya están en la pila y pasan a ser los slots del frame nuevo, las
locales son un índice calculado al compilar y `return` corta la pila en la
base del frame. Por eso `fib(15)` baja de 12.453 alocaciones a 2.598, y lo que
queda (~1,3 por llamada) son números: en Go, un `float64` guardado en un `any`
se aloca en el heap. Lo mismo en el `for`: el tree-walk crea un `Environment`
por vuelta (el cuerpo es un bloque) y la VM no crea ninguno.

La mayor mejora es en `calls` (5×) porque es el benchmark que más llamadas
hace por unidad de trabajo, y la menor en `strings` (2,8×) porque ahí buena
parte del tiempo es crear strings nuevos, que cuesta lo mismo en los dos.

### La optimización: globales en slots con caché por instrucción

La elegimos midiendo: un perfil de CPU de la VM mostró que **~13% del tiempo**
se iba en buscar variables globales en un `map[string]Value` (hashear el
nombre en cada acceso). Las locales ya eran un índice; las globales (todas las
variables de un script y todas las funciones de nivel superior) se seguían
buscando por nombre. La optimización le da a cada global un slot fijo en un
arreglo de la VM, y cada chunk guarda, por cada constante de nombre, el slot
que le corresponde: el primer acceso lo resuelve por nombre y los siguientes
son una indexación directa (*inline caching*).

| Script | Antes | Después | Mejora | Accesos a globales |
|---|---:|---:|---:|---|
| `fib(28)` | 40,8 ms | 39,6 ms | 3% | 2 lecturas de `fib` por llamada |
| 2M llamadas | 98 ms | 84 ms | 14% | 4 por vuelta (`total` y dos veces `sumar`) |
| Closure con estado 1M | 64 ms | 50 ms | 22% | 2 por vuelta (`contador` y `ultimo`) |
| `for` 2M | 93 ms | 70 ms | 25% | 2 por vuelta (`sum`), casi lo único que hace |
| 1M concatenaciones | 95 ms | 66 ms | 31% | 3 por vuelta (`s`), 5 cada cinco vueltas (`completos`) |

(`fib(28)` medido con `hyperfine`, 20 corridas, porque la diferencia es chica.)

**La mejora es proporcional a qué fracción del trabajo son accesos a
globales**, que es lo que el perfil predecía. En el `for`, cada vuelta lee y
escribe `sum` y no hace mucho más: el mapa era una parte grande del costo. En
`fib`, las dos lecturas de `fib` quedan enterradas en el costo de la llamada
en sí (armar el frame, la comparación, la resta, el retorno), así que casi no
se nota. En promedio, la VM final es **1,2× más rápida** que antes de
optimizar, sin cambiar ni un opcode ni la semántica.

### Contra otras implementaciones de Lox

![Tiempo relativo a glox](benchmarks/results/relativo.svg)

| | rlox | loxx | **glox** | jlox | glox-parcial | slox | dlox | plox-php | cloxure | plox |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Estrategia | bytecode | bytecode | **bytecode** | tree-walk | tree-walk | tree-walk | tree-walk | tree-walk | tree-walk | tree-walk |
| Relativo a glox | 0,47× | 0,56× | **1×** | 2,9× | 3,7× | 20× | 25× | 80× | 133× | 254× |

![Tiempo total por benchmark](benchmarks/results/tiempos.svg)

**En la entrega parcial glox era el tree-walk más rápido salvo jlox, y las
dos implementaciones a bytecode eran 7–8× más rápidas. Ahora glox es más
rápida que todos los tree-walk en todos los benchmarks con carga, y la distancia con
rlox (Rust) y loxx (C++) bajó a ~2×.** jlox, que con el JIT de la JVM nos
ganaba en los programas largos, ahora queda 2,9× atrás, y en la curva de
escalado glox es más rápido para todo n (con fib(27): 26 ms contra 78 ms).

Contra plox, la de la cátedra, la VM es entre 210× y 296× más rápida (era
55–78× con el tree-walk).

**Por qué rlox y loxx siguen siendo 2× más rápidas:** las tres compilan a
bytecode y ejecutan en una VM de pila parecida, así que la diferencia ya no es
de diseño sino de cómo se representa un valor. En C y en Rust un valor de Lox
es un struct chico (o un `double` con los otros tipos escondidos adentro) que
se copia sin alocar. En glox es un `any` de Go, y guardar un número en un
`any` lo manda al heap: son las ~2 alocaciones por vuelta del `for` y las ~1,3
por llamada de `fib` que quedan. Cada una es trabajo para el GC. Sacarlas
implicaría reemplazar `any` por un struct propio con un tag, y tocar cada
instrucción de la VM: es la próxima optimización si hubiera que seguir.

**Clases (`metodos.lox`):** glox tarda 91 ms, contra 34 ms de rlox y 39 ms de
loxx, una distancia un poco mayor que en el resto (2,3–2,7×). Un perfil de CPU
muestra por qué: ~20% del tiempo se va en buscar y asignar en los mapas de
campos y métodos, hasheando el nombre en cada acceso. Es exactamente el costo
que la optimización le sacó a las globales, ahora en las propiedades; `clox`
lo evita con strings internados, que hacen que comparar dos nombres sea
comparar dos punteros. Contra los tree-walk, glox sigue adelante (jlox 295 ms,
slox 1,8 s). **Solo 7 de las 11 pudieron correrlo**: plox no tiene clases, en
plox-php falla `this` adentro de los métodos, y las dos versiones viejas de
glox son anteriores a las clases. Además, rlox corta con `UndefinedProperty`
cuando se llama a un método heredado, por lo que el benchmark quedó sin
herencia.

**Memoria y arranque:** glox usa 10–11 MB en los benchmarks pesados (el
tree-walk, 11–13 MB) y arranca en 2,1 ms. Esos números no cambiaron con la VM
porque los domina el runtime de Go, no el programa.

### Las mismas ideas en otros lenguajes

`fib(28)` y el `for` de 2 millones de vueltas, escritos directamente en
cada lenguaje (`benchmarks/otros_lenguajes/`) y medidos con `hyperfine`:

| | Node 24 | Python 3.15 | **glox** | Ruby 2.6 | Perl 5.34 | glox-parcial | plox |
|---|---:|---:|---:|---:|---:|---:|---:|
| `fib(28)` | 22,7 ms | 28,2 ms | **39,8 ms** | 52,3 ms | 74,4 ms | 146 ms | 8,45 s |

| | Node 24 | Perl 5.34 | Ruby 2.6 | **glox** | Python 3.15 | glox-parcial | plox |
|---|---:|---:|---:|---:|---:|---:|---:|
| `for` 2M | 23,1 ms | 40,7 ms | 60,3 ms | **69,7 ms** | 135 ms | 271 ms | 20,6 s |

- **glox queda en el mismo rango que intérpretes de producción**: entre
  Python y Ruby en `fib`, entre Ruby y Python en el loop. Son lenguajes con
  décadas de optimización, pero también son, como glox, intérpretes de
  bytecode. Node es otra categoría: V8 compila a código de máquina mientras
  corre, y sus ~23 ms son casi todo arranque.
- **Python es más rápido que glox en `fib` y 2 veces más lento en el `for`.**
  En `fib.py`, `n` es una local, que CPython guarda en un arreglo. En
  `loops.py`, `total` e `i` son variables de módulo, que CPython guarda en un
  diccionario. Es la misma diferencia entre locales y globales que vimos en
  glox, y la que vino a atacar la optimización de globales.
- **plox es entre 150 y 300 veces más lento que Python ejecutando el mismo
  algoritmo**, aunque está escrito en Python. Es el costo de interpretar un lenguaje arriba
  de otro intérprete: cada operación de Lox se convierte en decenas de
  operaciones de Python. glox paga un solo nivel de interpretación.

### Resumen de los resultados

La entrega final es 3,7 veces más rápida que la parcial, y casi todo sale de
pasar del tree-walk a la VM: dejar de crear un entorno con un mapa por cada
llamada y por cada bloque, y acceder a las variables por posición. La
optimización suma otro 1,2× atacando el último acceso por nombre que quedaba
en el camino caliente, las globales. Lo que nos separa ahora de las VMs en
Rust y C++ ya no es el diseño sino la representación de los valores en Go, y
lo que más pesa en los programas con objetos son las búsquedas por nombre en
los mapas de campos y métodos.

---

## Programas de uso real

### `examples/gastos.lox` — Resumen de gastos del mes

Toma los gastos de un mes y produce un reporte: listado, total y promedio,
subtotal y porcentaje por categoría, los tres gastos más grandes, los "gastos
hormiga", alertas cuando una categoría supera su presupuesto y una proyección
con inflación mensual. Lo elegimos porque es el tipo de cosa que uno termina
haciendo en una planilla, y porque obliga a usar de todo: estructuras de
datos, ordenamiento, funciones de orden superior y estado.

La primera versión es de la entrega parcial y está armada solo con funciones:
las listas eran cadenas de closures (`cons(cabeza, cola)`), un gasto era una
closure que respondía por sus campos (`g("monto")`). Para la entrega final lo
reescribimos con una clase `Gasto`, una clase `Presupuesto` (que lleva su
propio acumulado y avisa cuando se pasa) y listas, y **la salida es
exactamente la misma**, byte por byte: el golden de `tests/golden/gastos.out`
no cambió. Fue la mejor prueba de clases y listas juntas. Lo que no se
reemplazó son las closures que tenían sentido como closures: la proyección
aplica `repetir(ajusteMensual, n)`, que compone la función consigo misma `n`
veces.

```text
$ ./glox examples/gastos.lox
=== Gastos de septiembre ===
  día 1  Alquiler (vivienda): $450.000,00
  día 2  Nafta (transporte): $35.000,00
  ...
Total: $719.811,74 en 15 gastos
Promedio por gasto: $47.987,45

=== Por categoría ===
  vivienda: $450.000,00 (63%)
  comida: $120.421,45 (17%)
  ...
=== Control de presupuesto ===
  ALERTA día 27: comida se pasó del presupuesto de $100.000,00 (lleva $120.421,45)
  ALERTA día 28: salidas se pasó del presupuesto de $40.000,00 (lleva $48.300,00)

=== Proyección con 3% de inflación mensual ===
  En 1 mes(es): $741.406,09
  En 2 mes(es): $763.648,27
  En 3 mes(es): $786.557,72
```

Como Lox no puede pasar un número a texto, los montos se formatean con
aritmética (`pesos(12345.5)` devuelve `"$12.345,50"`), ahora con una lista
de dígitos en vez de diez `if`.

### `examples/banco.lox` — Cuentas bancarias con herencia

En la entrega parcial, cada cuenta era una closure que respondía mensajes
(`cuenta("extraer", 200)`): la idea de objeto armada con funciones. Ahora es
una jerarquía de clases. `Cuenta` tiene el saldo, el historial (una lista) y
las operaciones; las dos subclases cambian solo lo que las distingue:

- `CuentaCorriente` pisa `puedeQuedarEn(saldo)` para permitir el descubierto,
  y en el cierre de mes cobra mantenimiento.
- `CajaDeAhorro` no admite descubierto y en el cierre de mes acredita
  interés, reusando el `depositar` heredado.

Las dos llaman a `super.init` y el programa recorre una lista de cuentas
llamando a `cierreDeMes()`, que hace algo distinto en cada una:

```text
$ ./glox examples/banco.lox
=== Operaciones del día ===
  OK       Ana deposita 500
  OK       Beto extrae 600 (usa descubierto)
  RECHAZO  Beto extrae 200 (supera descubierto): fondos insuficientes
  OK       Ana transfiere 300 a Beto
  RECHAZO  Caja transfiere 50 a Ana: fondos insuficientes
  RECHAZO  Ana extrae 1300 (caja de ahorro, sin descubierto): fondos insuficientes
  RECHAZO  Ana deposita -10: el monto tiene que ser positivo

=== Cierre de mes ===
  OK       cierre de Ana
  OK       cierre de Beto
  RECHAZO  cierre de Caja chica: sin saldo positivo, no genera interés

=== Saldos ===
  Ana (caja de ahorro 12%): 1212
  Beto (cuenta corriente): -130
  Caja chica (caja de ahorro 12%): 0
...
```

La salida completa de los tres programas de `examples/` está en
`tests/golden/`, y los tests end-to-end verifican que no cambie.

---

## Limitaciones conocidas

- **Hasta 256 constantes por función** (incluidos los nombres de variables
  globales y de propiedades), y 256 locales y 256 variables capturadas,
  porque los operandos son de un byte. Pasarse da un error de compilación
  claro, no bytecode inválido. `clox` tiene el mismo límite.
- **Las listas no tienen más operaciones que `len` y `push`**: no hay `pop`,
  concatenación ni recorte, y los strings no se pueden indexar.
- **No hay forma de pasar un número a texto** (no es parte de Lox), por eso
  los programas de ejemplo lo hacen con aritmética.
- El tree-walk no tiene clases ni listas (a propósito, ver
  [Del tree-walk al bytecode](#del-tree-walk-al-bytecode)), ni tope propio de
  recursión (una recursión infinita termina con el *stack overflow* fatal de
  Go en vez de un error de runtime 70 como en la VM).

---

## Cómo trabajamos

- **Diseñar antes de necesitarlo.** Antes de escribir código de clases,
  escribimos el diseño completo de clases y herencia en bytecode (opcodes,
  cómo se resuelven `this` y `super`, qué errores hay). Ahí se decidió también
  cómo iban a ser las closures, porque `super` dependía de eso. Cuando llegó el
  momento, el bytecode que salió fue el que habíamos escrito a mano, y quedó
  como test.
- **Una rama por tema y Pull Requests a `main`.** `scanning`, `interpreter`,
  `state`, `control-flow`, `functions`, `resolver`, `benchmarks` y
  `demo-and-docs` para la entrega parcial (PRs #1 a #9); `bytecode`,
  `bytecode-tests`, `bytecode-variables`, `bytecode-funciones`,
  `bytecode-clases` y `entrega-final` para la final (PRs #10 a #15).
- **Commits chicos y descriptivos,** en español.
- **Cada cambio cierra en verde:** `go build`, `go vet`, `gofmt -l .`,
  `go test ./...` y las pruebas de la cátedra, sobre la VM y sobre el
  tree-walk.
- **Las decisiones se escriben cuando se toman.** Cada vez que había dos
  caminos razonables (upvalues o celdas, recorrer la cadena de superclases o
  copiar los métodos, métodos o nativas para las listas), dejamos anotado cuál
  elegimos y por qué. De ahí salen las secciones de diferencias de este README.
- **Medir antes de optimizar.** La optimización de las globales salió de un
  perfil de CPU, no de una intuición, y se midió antes y después con los
  mismos scripts.
