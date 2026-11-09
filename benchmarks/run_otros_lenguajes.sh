#!/usr/bin/env bash
# Las mismas dos ideas que fib_grande.lox y loops_grande.lox, escritas
# directamente en Python, Ruby, JavaScript (Node) y Perl, contra glox (VM),
# glox-parcial (tree-walk) y plox. Es la comparación "misma idea, otros
# lenguajes" de la consigna. Usa hyperfine y deja la tabla en
# results/otros_lenguajes.md.
#
# Uso (desde la raíz del repo, después de setup_impls.sh):
#   ./benchmarks/run_otros_lenguajes.sh
#
# Los fuentes están en benchmarks/otros_lenguajes/. Los while de Python y Ruby
# son a propósito: el `for` de Lox es un while desazucarado, así que se compara
# el mismo bucle y no un range() optimizado.

set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
B="$ROOT/benchmarks"
BUILD="$B/.build"
OUT="$B/results/otros_lenguajes.md"
PLOX="$BUILD/plox-venv/bin/plox"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

{
    echo "# Misma idea, otros lenguajes"
    echo
    echo "Generado por \`benchmarks/run_otros_lenguajes.sh\` ($(date +%F), hyperfine, 1 corrida de calentamiento)."
    echo "Versiones: $(python3 --version 2>&1), $(ruby --version | cut -d' ' -f1-2), Node $(node --version), Perl $(perl -e 'print $^V')."
} >"$OUT"

for idea in fib loops; do
    lox="$B/${idea}_grande.lox"
    echo >>"$OUT"
    echo "## ${idea}" >>"$OUT"
    echo >>"$OUT"
    hyperfine --warmup 1 --min-runs 5 -N --style none --export-markdown "$TMP" \
        -n "glox (VM)" "$BUILD/glox $lox" \
        -n "glox-parcial (tree-walk)" "$BUILD/glox-parcial $lox" \
        -n "plox" "$PLOX $lox" \
        -n "Python" "python3 $B/otros_lenguajes/$idea.py" \
        -n "Ruby" "ruby $B/otros_lenguajes/$idea.rb" \
        -n "JavaScript (Node)" "node $B/otros_lenguajes/$idea.js" \
        -n "Perl" "perl $B/otros_lenguajes/$idea.pl"
    cat "$TMP" >>"$OUT"
done
echo "tabla en $OUT" >&2
