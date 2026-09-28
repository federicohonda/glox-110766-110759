#!/usr/bin/env bash
# Clona (si hace falta) y compila las implementaciones de Lox contra las que se
# compara glox. Deja en $BUILD_DIR/impls.tsv una línea por implementación con
# el comando exacto para correr un script: la lee run_benchmarks.py.
#
# Uso:
#   ./benchmarks/setup_impls.sh
#
# Variables opcionales:
#   IMPLS_DIR  dónde están (o se clonan) los repos   (default: benchmarks/.impls)
#   BUILD_DIR  dónde quedan los binarios compilados  (default: benchmarks/.build)
#   ONLY       lista separada por comas para compilar solo algunas (ej. ONLY=glox,rlox)
#
# Toolchains necesarios: go, cargo, swift, javac/java (JDK 17+), lein, cmake,
# php, dart, uv. Si falta alguno, esa implementación se saltea con un aviso y
# el resto se compila igual.

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMPLS_DIR="${IMPLS_DIR:-$ROOT/benchmarks/.impls}"
BUILD_DIR="${BUILD_DIR:-$ROOT/benchmarks/.build}"
ONLY="${ONLY:-}"
mkdir -p "$IMPLS_DIR" "$BUILD_DIR"
IMPLS_DIR="$(cd "$IMPLS_DIR" && pwd)"
BUILD_DIR="$(cd "$BUILD_DIR" && pwd)"
TSV="$BUILD_DIR/impls.tsv"

# El java de macOS en /usr/bin es un stub si no hay JDK instalado: se prefiere
# el de JAVA_HOME o el de Homebrew.
if [ -n "${JAVA_HOME:-}" ]; then
    JAVA="$JAVA_HOME/bin/java"; JAVAC="$JAVA_HOME/bin/javac"
elif command -v brew >/dev/null && [ -x "$(brew --prefix openjdk 2>/dev/null)/bin/java" ]; then
    JAVA_HOME="$(brew --prefix openjdk)"; export JAVA_HOME
    JAVA="$JAVA_HOME/bin/java"; JAVAC="$JAVA_HOME/bin/javac"
else
    JAVA="java"; JAVAC="javac"
fi
export PATH="$(dirname "$JAVA"):$PATH"

# nombre | repo | lenguaje | estrategia
IMPLS=(
    "rlox|https://github.com/Darksecond/lox|Rust|bytecode"
    "slox|https://github.com/alexito4/slox|Swift|tree-walk"
    "jlox|https://github.com/ryanq/jlox|Java|tree-walk"
    "cloxure|https://github.com/ceronman/cloxure|Clojure|tree-walk"
    "loxx|https://github.com/mspraggs/loxx|C++|bytecode"
    "plox-php|https://github.com/minirop/plox|PHP|tree-walk"
    "dlox|https://github.com/sma/lox|Dart|tree-walk"
    "plox|https://github.com/FdelMazo/plox|Python|tree-walk"
)

wanted() { [ -z "$ONLY" ] || [[ ",$ONLY," == *",$1,"* ]]; }
log() { echo "==> $*" >&2; }
warn() { echo "!!  $*" >&2; }

# Con ONLY se recompilan algunas y se conservan las demás entradas.
if [ -n "$ONLY" ] && [ -f "$TSV" ]; then
    awk -F'\t' -v only=",$ONLY," 'index(only, "," $1 ",") == 0' "$TSV" >"$TSV.tmp" && mv "$TSV.tmp" "$TSV"
else
    : >"$TSV"
fi
emit() { # nombre lenguaje estrategia comando...
    local name="$1" lang="$2" kind="$3"; shift 3
    printf '%s\t%s\t%s\t%s\n' "$name" "$lang" "$kind" "$*" >>"$TSV"
    log "$name listo: $*"
}

clone() { # nombre url
    local dir="$IMPLS_DIR/$1"
    if [ ! -d "$dir" ]; then
        log "clonando $2 en $dir"
        git clone --depth 1 "$2" "$dir" >&2 || return 1
    fi
}

# glox siempre se compila desde este mismo repo.
if wanted glox; then
    log "compilando glox"
    (cd "$ROOT" && go build -o "$BUILD_DIR/glox" ./cmd/glox) \
        && emit glox Go tree-walk "$BUILD_DIR/glox" \
        || warn "glox no compiló"
fi

# slox declara swift-tools 4.0 y depende de antitypical/Result (tools 3.1),
# que los Swift actuales ya no aceptan. Swift 5 trae `Result` en la stdlib, así
# que se compila una copia con un Package.swift moderno, sin esa dependencia y
# con un shim para `.value`/`.error` (lo único que la stdlib no tiene). El repo
# original no se toca.
build_slox() { # dir
    local src="$BUILD_DIR/slox-src"
    rm -rf "$src" && mkdir -p "$src" || return 1
    cp -R "$1/Sources" "$src/" || return 1
    rm -rf "$src/Sources/GenerateAst"
    find "$src/Sources/LoxCore" -name '*.swift' | while read -r f; do
        grep -v '^import Result$' "$f" >"$f.tmp" && mv "$f.tmp" "$f"
    done
    # Lo único de antitypical/Result que no tiene el Result de la stdlib.
    cat >"$src/Sources/LoxCore/ResultCompat.swift" <<'SWIFT'
extension Result {
    var value: Success? { if case let .success(v) = self { return v }; return nil }
    var error: Failure? { if case let .failure(e) = self { return e }; return nil }
}
SWIFT
    cat >"$src/Package.swift" <<'SWIFT'
// swift-tools-version:5.9
import PackageDescription
let package = Package(
    name: "slox",
    targets: [
        .executableTarget(name: "slox", dependencies: ["LoxCore"]),
        .target(name: "LoxCore"),
    ]
)
SWIFT
    swift build -c release --package-path "$src" --scratch-path "$BUILD_DIR/slox-build" >&2
}

# loxx inicializa un iterador de std::vector con `ip_(0)`, algo que el libc++
# actual ya no permite (el constructor desde puntero es privado), y declara
# una versión mínima de CMake que CMake 4 rechaza. Se compila una copia con
# `ip_()` y la política de CMake forzada. El repo original no se toca.
build_loxx() { # dir
    local src="$BUILD_DIR/loxx-src"
    rm -rf "$src" "$BUILD_DIR/loxx" && mkdir -p "$src" || return 1
    cp -R "$1/CMakeLists.txt" "$1/src" "$1/deps" "$src/" || return 1
    sed 's/ip_(0)/ip_()/' "$src/src/VirtualMachine.cpp" >"$src/src/VirtualMachine.cpp.tmp" \
        && mv "$src/src/VirtualMachine.cpp.tmp" "$src/src/VirtualMachine.cpp"
    cmake -S "$src" -B "$BUILD_DIR/loxx" -DCMAKE_BUILD_TYPE=Release \
        -DCMAKE_POLICY_VERSION_MINIMUM=3.5 -Wno-deprecated >&2 \
        && cmake --build "$BUILD_DIR/loxx" --target loxx -j >&2
}

build() { # nombre lenguaje estrategia dir
    local name="$1" lang="$2" kind="$3" dir="$4"
    case "$name" in
    rlox)
        CARGO_TARGET_DIR="$BUILD_DIR/rlox-target" cargo build --release -p lox \
            --manifest-path "$dir/Cargo.toml" >&2 \
            && emit "$name" "$lang" "$kind" "$BUILD_DIR/rlox-target/release/lox" ;;
    slox)
        build_slox "$dir" \
            && emit "$name" "$lang" "$kind" "$BUILD_DIR/slox-build/release/slox" ;;
    jlox)
        mkdir -p "$BUILD_DIR/jlox" \
            && "$JAVAC" -nowarn -d "$BUILD_DIR/jlox" "$dir"/src/com/craftinginterpreters/lox/*.java >&2 \
            && emit "$name" "$lang" "$kind" "$JAVA -cp $BUILD_DIR/jlox com.craftinginterpreters.lox.Lox" ;;
    cloxure)
        # El proyecto no hace AOT del namespace principal, así que el jar no
        # es ejecutable con -jar: se arranca con clojure.main.
        (cd "$dir" && lein uberjar >&2) \
            && cp "$(find "$dir/target" -name 'cloxure-*-standalone.jar' | head -1)" "$BUILD_DIR/cloxure.jar" \
            && emit "$name" "$lang" "$kind" "$JAVA -cp $BUILD_DIR/cloxure.jar clojure.main -m cloxure.core" ;;
    loxx)
        build_loxx "$dir" \
            && emit "$name" "$lang" "$kind" "$BUILD_DIR/loxx/loxx" ;;
    plox-php)
        # error_reporting sin E_DEPRECATED: PHP 8.5 avisa por cada cast (double).
        php --version >/dev/null \
            && emit "$name" "$lang" "$kind" "php -d error_reporting=24575 $dir/plox.php" ;;
    dlox)
        (cd "$dir" && dart pub get >&2 && dart compile exe bin/lox.dart -o "$BUILD_DIR/dlox" >&2) \
            && emit "$name" "$lang" "$kind" "$BUILD_DIR/dlox" ;;
    plox)
        UV_PROJECT_ENVIRONMENT="$BUILD_DIR/plox-venv" uv sync --project "$dir" --python 3.12 >&2 \
            && emit "$name" "$lang" "$kind" "$BUILD_DIR/plox-venv/bin/plox" ;;
    esac
}

for entry in "${IMPLS[@]}"; do
    IFS='|' read -r name url lang kind <<<"$entry"
    wanted "$name" || continue
    log "preparando $name ($lang)"
    clone "$name" "$url" || { warn "no se pudo clonar $name"; continue; }
    build "$name" "$lang" "$kind" "$IMPLS_DIR/$name" || warn "$name no compiló, se saltea"
done

log "implementaciones disponibles en $TSV:"
cut -f1 "$TSV" >&2
