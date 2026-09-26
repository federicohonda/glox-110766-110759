#!/usr/bin/env python3
"""Arma los gráficos (SVG) y las tablas (RESULTS.md) a partir de results/results.json.

Uso:
  python3 benchmarks/plot.py

Solo usa la biblioteca estándar: los SVG se escriben a mano para no depender
de matplotlib. Cada SVG trae su versión clara y oscura (prefers-color-scheme),
así se ven bien en GitHub con cualquiera de los dos temas.

Codificación que comparten todos los gráficos:
  - glox en azul, el resto en gris: la pregunta siempre es "dónde queda glox";
  - las implementaciones que compilan a bytecode (rlox, loxx) en naranja y con
    rombo en vez de círculo, porque es la diferencia de diseño que más explica
    los tiempos (y la que glox va a tener en el Bloque 2).
"""

import json
import math
import statistics
from pathlib import Path

BENCH_DIR = Path(__file__).resolve().parent
RESULTS = BENCH_DIR / "results"

# Scripts con carga suficiente para que el arranque no domine (> ~100 ms en glox).
PESADOS = ["fib_grande", "loops_grande", "calls", "counter", "strings"]
# Los de Fran, con la carga original.
CHICOS = ["fib", "loops", "closures"]

TITULOS = {
    "startup": "Arranque (print \"ok\")",
    "fib": "fib(25)",
    "loops": "for 100k",
    "closures": "10k closures",
    "fib_grande": "fib(28)",
    "loops_grande": "for 2M",
    "calls": "1M llamadas",
    "counter": "closure con estado 1M",
    "strings": "1M concatenaciones",
}

STYLE = """
<style>
  svg { --surface:#fcfcfb; --ink:#0b0b0b; --ink2:#52514e; --muted:#8a8984; --grid:#e6e5e1;
        --glox:#2a78d6; --bytecode:#eb6834; --other:#86857f; }
  @media (prefers-color-scheme: dark) {
    svg { --surface:#1a1a19; --ink:#ffffff; --ink2:#c3c2b7; --muted:#8f8e86; --grid:#383835;
          --glox:#3987e5; --bytecode:#d95926; --other:#7d7c76; }
  }
  text { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif; fill: var(--ink2); font-size: 12px; }
  .title { fill: var(--ink); font-size: 15px; font-weight: 600; }
  .subtitle { fill: var(--ink2); font-size: 12px; }
  .panel { fill: var(--ink); font-size: 13px; font-weight: 600; }
  .label { fill: var(--ink); }
  .label.glox { font-weight: 700; }
  .value { fill: var(--ink2); font-size: 11px; }
  .tick { fill: var(--muted); font-size: 11px; }
  .grid { stroke: var(--grid); stroke-width: 1; }
  .stem { stroke: var(--grid); stroke-width: 2; }
  .glox { fill: var(--glox); stroke: var(--glox); }
  .bytecode { fill: var(--bytecode); stroke: var(--bytecode); }
  .other { fill: var(--other); stroke: var(--other); }
  .line { fill: none; stroke-width: 2; }
  .line.glox { stroke-width: 3; }
  .ring { stroke: var(--surface); stroke-width: 2; }
  .bg { fill: var(--surface); }
</style>
"""


def esc(s):
    return str(s).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def svg(width, height, body, title):
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}" role="img" aria-label="{esc(title)}">'
        f"<title>{esc(title)}</title>{STYLE}"
        f'<rect class="bg" width="{width}" height="{height}" rx="8"/>'
        + "".join(body)
        + "</svg>\n"
    )


def klass(name, impls):
    if name == "glox":
        return "glox"
    return "bytecode" if impls[name]["kind"] == "bytecode" else "other"


def marker(x, y, cls, r=5):
    """Círculo para tree-walk, rombo para bytecode; con anillo del color del fondo."""
    if cls == "bytecode":
        d = r + 1.5
        return f'<path class="{cls} ring" d="M{x:.1f},{y - d:.1f} L{x + d:.1f},{y:.1f} L{x:.1f},{y + d:.1f} L{x - d:.1f},{y:.1f} Z"/>'
    return f'<circle class="{cls} ring" cx="{x:.1f}" cy="{y:.1f}" r="{r}"/>'


def fmt_time(s):
    if s < 1:
        return f"{s * 1000:.0f} ms" if s >= 0.0095 else f"{s * 1000:.1f} ms"
    return f"{s:.2f} s" if s < 10 else f"{s:.1f} s"


def fmt_ratio(r):
    if r < 1:
        return f"{r:.2f}×"
    if r < 10:
        return f"{r:.1f}×"
    return f"{r:.0f}×"


def log_ticks(lo, hi):
    ticks = []
    e = math.floor(math.log10(lo))
    while 10**e <= hi * 1.0001:
        if 10**e >= lo * 0.9999:
            ticks.append(10**e)
        e += 1
    return ticks


def tick_label_time(v):
    if v < 1:
        ms = v * 1000
        return f"{ms:g} ms"
    return f"{v:g} s"


def valid(data, script, impl):
    r = data["results"].get(script, {}).get(impl)
    return r if r and r.get("valid") else None


def orden(data, impls):
    """Implementaciones de la más rápida a la más lenta (media geométrica en los pesados)."""
    def gm(name):
        vals = [valid(data, s, name)["wall_median"] for s in PESADOS if valid(data, s, name)]
        return statistics.geometric_mean(vals) if vals else float("inf")
    return sorted(impls, key=gm)


# ---------------------------------------------------------------------------
# 1. Tiempos por benchmark: small multiples con eje log compartido
# ---------------------------------------------------------------------------

def chart_tiempos(data, impls, names, scripts, filename, title, subtitle):
    cols = 2
    rows = math.ceil(len(scripts) / cols)
    label_w, panel_w, row_h = 78, 330, 22
    panel_h = 34 + row_h * len(names) + 26
    width = 24 + cols * (label_w + panel_w + 30)
    height = 70 + rows * panel_h + 10
    vals = [valid(data, s, n)["wall_median"] for s in scripts for n in names if valid(data, s, n)]
    lo = 10 ** math.floor(math.log10(min(vals)))
    hi = 10 ** math.ceil(math.log10(max(vals)))
    lx = lambda v: (math.log10(v) - math.log10(lo)) / (math.log10(hi) - math.log10(lo)) * (panel_w - 60)
    body = [
        f'<text class="title" x="24" y="30">{esc(title)}</text>',
        f'<text class="subtitle" x="24" y="50">{esc(subtitle)}</text>',
    ]
    for i, script in enumerate(scripts):
        ox = 24 + (i % cols) * (label_w + panel_w + 30)
        oy = 70 + (i // cols) * panel_h
        px = ox + label_w
        body.append(f'<text class="panel" x="{ox}" y="{oy + 16}">{esc(TITULOS[script])}</text>')
        top = oy + 30
        bottom = top + row_h * len(names)
        for t in log_ticks(lo, hi):
            x = px + lx(t)
            body.append(f'<line class="grid" x1="{x:.1f}" y1="{top - 4}" x2="{x:.1f}" y2="{bottom}"/>')
            body.append(f'<text class="tick" x="{x:.1f}" y="{bottom + 15}" text-anchor="middle">{tick_label_time(t)}</text>')
        for j, name in enumerate(names):
            y = top + row_h * j + row_h / 2
            cls = klass(name, impls)
            lcls = "label glox" if name == "glox" else "label"
            body.append(f'<text class="{lcls}" x="{px - 10}" y="{y + 4:.1f}" text-anchor="end">{esc(name)}</text>')
            r = valid(data, script, name)
            if not r:
                body.append(f'<text class="value" x="{px}" y="{y + 4:.1f}">sin dato</text>')
                continue
            x = px + lx(r["wall_median"])
            body.append(f'<line class="stem" x1="{px}" y1="{y:.1f}" x2="{x:.1f}" y2="{y:.1f}"/>')
            body.append(marker(x, y, cls))
            body.append(f'<text class="value" x="{x + 10:.1f}" y="{y + 4:.1f}">{fmt_time(r["wall_median"])}</text>')
    (RESULTS / filename).write_text(svg(width, height, body, title))


# ---------------------------------------------------------------------------
# 2. Cuántas veces más lento que glox (media geométrica + rango)
# ---------------------------------------------------------------------------

def ratios_vs_glox(data, name):
    out = []
    for s in PESADOS:
        r, g = valid(data, s, name), valid(data, s, "glox")
        if r and g:
            out.append(r["wall_median"] / g["wall_median"])
    return out


def chart_relativo(data, impls, names):
    label_w, plot_w, row_h = 90, 560, 30
    top = 90
    width = 24 + label_w + plot_w + 90
    height = top + row_h * len(names) + 50
    rows = [(n, ratios_vs_glox(data, n)) for n in names]
    rows = [(n, rs) for n, rs in rows if rs]
    allv = [v for _, rs in rows for v in rs]
    lo = 10 ** math.floor(math.log10(min(allv)))
    hi = 10 ** math.ceil(math.log10(max(allv)))
    lx = lambda v: (math.log10(v) - math.log10(lo)) / (math.log10(hi) - math.log10(lo)) * plot_w
    px = 24 + label_w
    title = "Tiempo relativo a glox"
    body = [
        f'<text class="title" x="24" y="30">{title}</text>',
        '<text class="subtitle" x="24" y="50">Media geométrica en los 5 benchmarks pesados (punto) y rango entre el más y el menos favorable (línea).</text>',
        '<text class="subtitle" x="24" y="68">A la izquierda de 1× es más rápida que glox, a la derecha más lenta. Escala logarítmica.</text>',
    ]
    bottom = top + row_h * len(rows)
    for t in log_ticks(lo, hi):
        x = px + lx(t)
        body.append(f'<line class="grid" x1="{x:.1f}" y1="{top - 6}" x2="{x:.1f}" y2="{bottom}"/>')
        body.append(f'<text class="tick" x="{x:.1f}" y="{bottom + 16}" text-anchor="middle">{t:g}×</text>')
    for j, (name, rs) in enumerate(rows):
        y = top + row_h * j + row_h / 2
        cls = klass(name, impls)
        lcls = "label glox" if name == "glox" else "label"
        body.append(f'<text class="{lcls}" x="{px - 12}" y="{y + 4:.1f}" text-anchor="end">{esc(name)}</text>')
        gm = statistics.geometric_mean(rs)
        x1, x2, xg = px + lx(min(rs)), px + lx(max(rs)), px + lx(gm)
        if name != "glox":
            body.append(f'<line class="{cls}" x1="{x1:.1f}" y1="{y:.1f}" x2="{x2:.1f}" y2="{y:.1f}" stroke-width="2" stroke-linecap="round" opacity="0.55"/>')
        body.append(marker(xg, y, cls, r=6))
        body.append(f'<text class="value" x="{max(x2, xg) + 12:.1f}" y="{y + 4:.1f}">{fmt_ratio(gm)}</text>')
    (RESULTS / "relativo.svg").write_text(svg(width, height, body, title))


# ---------------------------------------------------------------------------
# 3 y 4. Barras horizontales lineales (arranque y memoria)
# ---------------------------------------------------------------------------

def chart_barras(impls, rows, filename, title, subtitle, fmt, unit_ticks):
    label_w, plot_w, row_h, bar_h = 90, 520, 26, 14
    top = 76
    width = 24 + label_w + plot_w + 90
    height = top + row_h * len(rows) + 44
    hi = max(v for _, v in rows)
    step = unit_ticks(hi)
    hi_axis = math.ceil(hi / step) * step
    lx = lambda v: v / hi_axis * plot_w
    px = 24 + label_w
    body = [
        f'<text class="title" x="24" y="30">{esc(title)}</text>',
        f'<text class="subtitle" x="24" y="50">{esc(subtitle)}</text>',
    ]
    bottom = top + row_h * len(rows)
    t = 0
    while t <= hi_axis + 1e-9:
        x = px + lx(t)
        body.append(f'<line class="grid" x1="{x:.1f}" y1="{top - 6}" x2="{x:.1f}" y2="{bottom}"/>')
        body.append(f'<text class="tick" x="{x:.1f}" y="{bottom + 16}" text-anchor="middle">{fmt(t, axis=True)}</text>')
        t += step
    for j, (name, v) in enumerate(rows):
        y = top + row_h * j + (row_h - bar_h) / 2
        cls = klass(name, impls)
        lcls = "label glox" if name == "glox" else "label"
        body.append(f'<text class="{lcls}" x="{px - 12}" y="{y + bar_h - 3:.1f}" text-anchor="end">{esc(name)}</text>')
        w = max(lx(v), 2)
        # Extremo de datos redondeado, anclado al eje en cero.
        r = min(4, w / 2)
        body.append(
            f'<path class="{cls}" d="M{px},{y:.1f} h{w - r:.1f} a{r},{r} 0 0 1 {r},{r} v{bar_h - 2 * r:.1f} '
            f'a{r},{r} 0 0 1 -{r},{r} h-{w - r:.1f} Z"/>'
        )
        body.append(f'<text class="value" x="{px + w + 8:.1f}" y="{y + bar_h - 3:.1f}">{fmt(v)}</text>')
    (RESULTS / filename).write_text(svg(width, height, body, title))


def nice_step(hi, candidates):
    for c in candidates:
        if hi / c <= 6:
            return c
    return candidates[-1]


# ---------------------------------------------------------------------------
# 5. Escalado: fib(n) para varios n, eje y logarítmico
# ---------------------------------------------------------------------------

def chart_escalado(data, impls, names):
    ns = [int(n) for n in data["escalado_n"] if str(n) in data.get("escalado", {})]
    if not ns:
        return
    left, plot_w, plot_h, right = 70, 600, 340, 110
    top = 80
    width = left + plot_w + right
    height = top + plot_h + 60
    series = {}
    for name in names:
        pts = []
        for n in ns:
            r = data["escalado"][str(n)].get(name)
            if r and r.get("valid"):
                pts.append((n, r["wall_median"]))
        if pts:
            series[name] = pts
    allv = [v for pts in series.values() for _, v in pts]
    lo = 10 ** math.floor(math.log10(min(allv)))
    hi = 10 ** math.ceil(math.log10(max(allv)))
    sx = lambda n: left + (n - ns[0]) / (ns[-1] - ns[0]) * plot_w
    sy = lambda v: top + plot_h - (math.log10(v) - math.log10(lo)) / (math.log10(hi) - math.log10(lo)) * plot_h
    title = "Escalado: fib(n) recursivo"
    body = [
        f'<text class="title" x="24" y="30">{title}</text>',
        '<text class="subtitle" x="24" y="50">Tiempo total del proceso (mediana) según n. Eje y logarítmico: una recta con pendiente</text>',
        '<text class="subtitle" x="24" y="66">es crecimiento exponencial; el tramo plano de la izquierda es el costo de arranque.</text>',
    ]
    for t in log_ticks(lo, hi):
        y = sy(t)
        body.append(f'<line class="grid" x1="{left}" y1="{y:.1f}" x2="{left + plot_w}" y2="{y:.1f}"/>')
        body.append(f'<text class="tick" x="{left - 8}" y="{y + 4:.1f}" text-anchor="end">{tick_label_time(t)}</text>')
    for n in ns:
        body.append(f'<text class="tick" x="{sx(n):.1f}" y="{top + plot_h + 18}" text-anchor="middle">n = {n}</text>')
    # Primero las grises, al final glox para que quede arriba.
    ordered = sorted(series, key=lambda n: {"other": 0, "bytecode": 1, "glox": 2}[klass(n, impls)])
    for name in ordered:
        cls = klass(name, impls)
        pts = series[name]
        d = " ".join(f"{'M' if k == 0 else 'L'}{sx(n):.1f},{sy(v):.1f}" for k, (n, v) in enumerate(pts))
        body.append(f'<path class="line {cls}" d="{d}" style="fill:none"/>')
        for n, v in pts:
            body.append(marker(sx(n), sy(v), cls, r=4))
    # Etiquetas directas a la derecha, separadas para que no se pisen.
    ends = sorted(((sy(series[n][-1][1]), n) for n in series), key=lambda e: e[0])
    placed = []
    for y, name in ends:
        if placed and y - placed[-1][0] < 14:
            y = placed[-1][0] + 14
        placed.append((y, name))
    overflow = placed[-1][0] - (top + plot_h)
    if overflow > 0:
        placed = [(y - overflow, n) for y, n in placed]
    for y, name in placed:
        lcls = "label glox" if name == "glox" else "label"
        body.append(f'<text class="{lcls}" x="{left + plot_w + 12}" y="{y + 4:.1f}">{esc(name)}</text>')
    (RESULTS / "escalado.svg").write_text(svg(width, height, body, title))


# ---------------------------------------------------------------------------
# Tablas
# ---------------------------------------------------------------------------

def tablas(data, impls, names):
    m = data["machine"]
    out = [
        "# Resultados de benchmarks",
        "",
        "Generado por `benchmarks/plot.py` a partir de `results/results.json`. No editar a mano.",
        "",
        f"- **Máquina:** {m.get('cpu', '?')}, {m.get('memory_gb', '?')} GB de RAM, {m.get('platform', '?')}",
        f"- **Fecha:** {m.get('date', '?')}",
        "- **Métrica:** mediana del tiempo total del proceso (arranque + scan + parse + resolve + ejecución).",
        "",
        "## Tiempo (mediana)",
        "",
        "| Implementación | Lenguaje | Estrategia | " + " | ".join(f"`{s}`" for s in data["scripts"]) + " |",
        "|---|---|---|" + "---:|" * len(data["scripts"]),
    ]
    for n in names:
        cells = []
        for s in data["scripts"]:
            r = valid(data, s, n)
            cells.append(fmt_time(r["wall_median"]) if r else "—")
        nm = f"**{n}**" if n == "glox" else n
        out.append(f"| {nm} | {impls[n]['lang']} | {impls[n]['kind']} | " + " | ".join(cells) + " |")

    out += ["", "## Tiempo relativo a glox (> 1 = más lenta que glox)", "",
            "| Implementación | " + " | ".join(f"`{s}`" for s in data["scripts"]) + " | Media geom. (pesados) |",
            "|---|" + "---:|" * (len(data["scripts"]) + 1)]
    for n in names:
        cells = []
        for s in data["scripts"]:
            r, g = valid(data, s, n), valid(data, s, "glox")
            cells.append(fmt_ratio(r["wall_median"] / g["wall_median"]) if r and g else "—")
        rs = ratios_vs_glox(data, n)
        gm = fmt_ratio(statistics.geometric_mean(rs)) if rs else "—"
        nm = f"**{n}**" if n == "glox" else n
        out.append(f"| {nm} | " + " | ".join(cells) + f" | {gm} |")

    out += ["", "## Memoria residente pico (MB)", "",
            "| Implementación | " + " | ".join(f"`{s}`" for s in data["scripts"]) + " |",
            "|---|" + "---:|" * len(data["scripts"])]
    for n in names:
        cells = []
        for s in data["scripts"]:
            r = valid(data, s, n)
            cells.append(f"{r['rss_max'] / 2**20:.1f}" if r else "—")
        nm = f"**{n}**" if n == "glox" else n
        out.append(f"| {nm} | " + " | ".join(cells) + " |")

    out += ["", "## CPU (user + sys, mediana) sobre tiempo de reloj", "",
            "Un valor mayor a 1 indica que el proceso usó más de un núcleo (JIT, GC o compilación en paralelo).", "",
            "| Implementación | " + " | ".join(f"`{s}`" for s in PESADOS) + " |",
            "|---|" + "---:|" * len(PESADOS)]
    for n in names:
        cells = []
        for s in PESADOS:
            r = valid(data, s, n)
            cells.append(f"{r['cpu_median'] / r['wall_median']:.2f}" if r else "—")
        out.append(f"| {n} | " + " | ".join(cells) + " |")

    if data.get("escalado"):
        ns = [str(n) for n in data["escalado_n"] if str(n) in data["escalado"]]
        out += ["", "## Escalado: fib(n)", "", "| Implementación | " + " | ".join(f"n={n}" for n in ns) + " |",
                "|---|" + "---:|" * len(ns)]
        for n in names:
            cells = []
            for k in ns:
                r = data["escalado"][k].get(n)
                cells.append(fmt_time(r["wall_median"]) if r and r.get("valid") else "—")
            out.append(f"| {n} | " + " | ".join(cells) + " |")

    invalid = [(s, n, r["detail"]) for s, rr in data["results"].items() for n, r in rr.items() if not r.get("valid")]
    if invalid:
        out += ["", "## Corridas inválidas (no aparecen en tablas ni gráficos)", ""] + [f"- `{s}` con {n}: {d}" for s, n, d in invalid]
    notes = [(s, n, r["detail"]) for s, rr in data["results"].items() for n, r in rr.items() if r.get("valid") and r.get("detail")]
    if notes:
        out += ["", "## Notas de validación", ""] + [f"- `{s}` con {n}: {d}" for s, n, d in notes]

    corridas = sorted({len(r["samples"]) for rr in data["results"].values() for r in rr.values() if r.get("valid")})
    out += ["", f"Corridas medidas por combinación: entre {corridas[0]} y {corridas[-1]} (ver `run_benchmarks.py --help`).", ""]
    (RESULTS / "RESULTS.md").write_text("\n".join(out))


def main():
    data = json.loads((RESULTS / "results.json").read_text())
    impls = data["impls"]
    names = orden(data, list(impls))

    chart_tiempos(
        data, impls, names, PESADOS + ["fib"], "tiempos.svg",
        "Tiempo total por benchmark (mediana)",
        "De la más rápida a la más lenta en promedio. Escala logarítmica compartida. Rombo naranja = compilan a bytecode.",
    )
    chart_relativo(data, impls, names)

    arranque = [(n, valid(data, "startup", n)["wall_median"] * 1000) for n in names if valid(data, "startup", n)]
    arranque.sort(key=lambda e: e[1])
    chart_barras(
        impls, arranque, "arranque.svg", "Costo de arranque",
        'Tiempo de un script que solo hace print "ok": levantar el runtime, sin trabajo real.',
        lambda v, axis=False: f"{v:.0f} ms" if axis or v >= 10 else f"{v:.1f} ms",
        lambda hi: nice_step(hi, [1, 2, 5, 10, 20, 25, 50, 100, 200, 500]),
    )
    mem = [(n, valid(data, "fib_grande", n)["rss_max"] / 2**20) for n in names if valid(data, "fib_grande", n)]
    mem.sort(key=lambda e: e[1])
    chart_barras(
        impls, mem, "memoria.svg", "Memoria residente pico en fib(28)",
        "Máximo RSS del proceso: incluye el runtime de cada lenguaje, no solo los datos del programa.",
        lambda v, axis=False: f"{v:.0f} MB",
        lambda hi: nice_step(hi, [5, 10, 20, 25, 50, 100, 200, 250, 500]),
    )
    chart_escalado(data, impls, names)
    tablas(data, impls, names)
    print(f"gráficos y tablas en {RESULTS}")


if __name__ == "__main__":
    main()
