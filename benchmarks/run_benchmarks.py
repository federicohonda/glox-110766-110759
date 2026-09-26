#!/usr/bin/env python3
"""Corre los scripts de benchmarks/ con glox y con las demás implementaciones de Lox.

Lee las implementaciones de .build/impls.tsv (lo genera setup_impls.sh), corre
cada script varias veces por implementación y guarda todas las muestras en
results/results.json. Después, plot.py arma los gráficos y las tablas.

Por cada corrida se mide:
  - tiempo de reloj (wall) con time.perf_counter
  - tiempo de CPU (user + sys) y memoria residente pico (RSS) del proceso
    hijo, con os.wait4, que funciona igual en macOS y en Linux
y se valida que la última línea impresa sea el `// resultado:` que declara el
script (comparando por valor: 4.99995e+09, 4.99995E9 y 4999950000.0 son lo mismo).

Uso:
  python3 benchmarks/run_benchmarks.py                    # todo
  python3 benchmarks/run_benchmarks.py --only glox,plox   # algunas implementaciones
  python3 benchmarks/run_benchmarks.py --scripts fib,loops --runs 10
  python3 benchmarks/run_benchmarks.py --sin-escalado     # saltea la curva de fib(n)

Solo usa la biblioteca estándar de Python 3.9+.
"""

import argparse
import datetime
import json
import os
import platform
import re
import shlex
import statistics
import subprocess
import sys
import tempfile
import threading
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
BENCH_DIR = ROOT / "benchmarks"

# Orden en el que se presentan: primero los de Fran (carga chica, dominados por
# el arranque), después los de carga grande.
SCRIPTS = [
    "startup",
    "fib",
    "loops",
    "closures",
    "fib_grande",
    "loops_grande",
    "calls",
    "counter",
    "strings",
]

# Curva de escalado: fib(n) para estos n, generados al vuelo.
ESCALADO_N = [15, 18, 21, 24, 27]

RESULT_RE = re.compile(r"^// resultado: (.+)$", re.M)


def log(msg):
    print(msg, file=sys.stderr, flush=True)


def load_impls(tsv):
    impls = []
    for line in tsv.read_text().splitlines():
        if not line.strip():
            continue
        name, lang, kind, cmd = line.split("\t")
        impls.append({"name": name, "lang": lang, "kind": kind, "cmd": shlex.split(cmd)})
    return impls


def same_value(got, want):
    """Devuelve (coincide, nota). Compara por valor numérico: 4.99995e+09,
    4.99995E9 y 4999950000.0 son el mismo número. Además tolera la impresión
    con 6 cifras significativas (el %g por defecto de C/C++, que usa loxx:
    imprime 2e+12 para 1999999000000), pero solo si el esperado redondeado a 6
    cifras da exactamente lo impreso."""
    try:
        g, w = float(got), float(want)
    except ValueError:
        return got == want, ""
    if g == w:
        return True, ""
    if float(f"{w:.6g}") == g:
        return True, f"imprime con 6 cifras significativas ({got})"
    return False, ""


def maxrss_bytes(rusage):
    # ru_maxrss viene en bytes en macOS y en kilobytes en Linux.
    return rusage.ru_maxrss if sys.platform == "darwin" else rusage.ru_maxrss * 1024


def run_once(cmd, script, timeout):
    """Corre el script una vez y devuelve (muestra, stdout)."""
    with tempfile.TemporaryFile() as out, tempfile.TemporaryFile() as err:
        timed_out = threading.Event()
        start = time.perf_counter()
        proc = subprocess.Popen(cmd + [str(script)], stdout=out, stderr=err, stdin=subprocess.DEVNULL)

        def kill():
            timed_out.set()
            proc.kill()

        timer = threading.Timer(timeout, kill)
        timer.start()
        _, status, rusage = os.wait4(proc.pid, 0)
        timer.cancel()
        proc.returncode = os.waitstatus_to_exitcode(status)  # ya lo recogió wait4
        if timed_out.is_set():
            return None, "TIMEOUT"
        wall = time.perf_counter() - start
        out.seek(0)
        stdout = out.read().decode(errors="replace")
        code = proc.returncode
    sample = {
        "wall": wall,
        "cpu": rusage.ru_utime + rusage.ru_stime,
        "rss": maxrss_bytes(rusage),
        "exit": code,
    }
    return sample, stdout


def measure(impl, script, expected, args):
    """Corre un script varias veces con una implementación y devuelve el resumen."""
    samples = []
    valid = True
    detail = ""
    total = 0.0
    # Una corrida de calentamiento (disco, cachés del SO) que no se cuenta,
    # salvo que ya sea lenta: ahí el calentamiento no cambia nada y es tiempo perdido.
    warm, stdout = run_once(impl["cmd"], script, args.timeout)
    if warm is None:
        return {"valid": False, "detail": "timeout", "samples": []}
    if warm["wall"] > 2:
        samples.append(warm)
        total += warm["wall"]
    while len(samples) < args.runs:
        if len(samples) >= args.min_runs and total > args.budget:
            break
        sample, stdout = run_once(impl["cmd"], script, args.timeout)
        if sample is None:
            return {"valid": False, "detail": "timeout", "samples": samples}
        samples.append(sample)
        total += sample["wall"]
    lines = stdout.strip().splitlines()
    last = lines[-1].strip() if lines else ""
    if samples[-1]["exit"] != 0:
        valid, detail = False, f"código de salida {samples[-1]['exit']}"
    elif expected is not None:
        ok, note = same_value(last, expected)
        if not ok:
            valid, detail = False, f"imprimió {last!r}, se esperaba {expected}"
        else:
            detail = note
    walls = [s["wall"] for s in samples]
    return {
        "valid": valid,
        "detail": detail,
        "output": last,
        "samples": samples,
        "wall_median": statistics.median(walls),
        "wall_min": min(walls),
        "wall_stdev": statistics.stdev(walls) if len(walls) > 1 else 0.0,
        "cpu_median": statistics.median(s["cpu"] for s in samples),
        "rss_max": max(s["rss"] for s in samples),
    }


def machine_info():
    info = {
        "platform": platform.platform(),
        "machine": platform.machine(),
        "python": platform.python_version(),
        "date": datetime.date.today().isoformat(),
    }
    try:
        if sys.platform == "darwin":
            info["cpu"] = subprocess.check_output(["sysctl", "-n", "machdep.cpu.brand_string"], text=True).strip()
            mem = int(subprocess.check_output(["sysctl", "-n", "hw.memsize"], text=True))
            info["memory_gb"] = round(mem / 2**30)
        else:
            for line in Path("/proc/cpuinfo").read_text().splitlines():
                if line.startswith("model name"):
                    info["cpu"] = line.split(":", 1)[1].strip()
                    break
    except Exception:  # la info de la máquina es decorativa, no debe cortar la corrida
        pass
    return info


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--impls", default=str(BENCH_DIR / ".build" / "impls.tsv"), help="archivo generado por setup_impls.sh")
    parser.add_argument("--only", help="implementaciones a correr, separadas por coma")
    parser.add_argument("--scripts", help="scripts a correr (sin .lox), separados por coma")
    parser.add_argument("--runs", type=int, default=5, help="corridas medidas por script (default 5)")
    parser.add_argument("--min-runs", type=int, default=3, help="mínimo de corridas aunque se pase el presupuesto")
    parser.add_argument("--budget", type=float, default=60, help="segundos por script e implementación antes de cortar en --min-runs")
    parser.add_argument("--timeout", type=float, default=300, help="segundos máximos por corrida")
    parser.add_argument("--sin-escalado", action="store_true", help="no correr la curva de fib(n)")
    parser.add_argument("--out", default=str(BENCH_DIR / "results" / "results.json"))
    args = parser.parse_args()

    impls = load_impls(Path(args.impls))
    if args.only:
        wanted = args.only.split(",")
        impls = [i for i in impls if i["name"] in wanted]
    scripts = args.scripts.split(",") if args.scripts else SCRIPTS

    out_path = Path(args.out)
    # Si ya hay resultados, se actualizan solo las combinaciones que se vuelven
    # a correr: así se puede re-medir una implementación sin repetir todo.
    data = json.loads(out_path.read_text()) if out_path.exists() else {}
    data["machine"] = machine_info()
    data.setdefault("impls", {})
    data.setdefault("results", {})
    data.setdefault("escalado", {})
    data["scripts"] = SCRIPTS
    data["escalado_n"] = ESCALADO_N

    for impl in impls:
        data["impls"][impl["name"]] = {"lang": impl["lang"], "kind": impl["kind"]}

    for script_name in scripts:
        script = BENCH_DIR / f"{script_name}.lox"
        m = RESULT_RE.search(script.read_text())
        expected = m.group(1).strip() if m else None
        for impl in impls:
            log(f"[{script_name}] {impl['name']}...")
            res = measure(impl, script, expected, args)
            data["results"].setdefault(script_name, {})[impl["name"]] = res
            if res["valid"]:
                log(f"    mediana {res['wall_median']:.3f}s  ({len(res['samples'])} corridas)  RSS {res['rss_max'] / 2**20:.1f} MB")
            else:
                log(f"    INVÁLIDO: {res['detail']}")
            save(out_path, data)

    if not args.sin_escalado:
        with tempfile.TemporaryDirectory() as tmp:
            for n in ESCALADO_N:
                script = Path(tmp) / f"fib_{n}.lox"
                script.write_text(
                    "fun fib(n) {\n  if (n < 2) return n;\n  return fib(n - 1) + fib(n - 2);\n}\n"
                    f"print fib({n});\n"
                )
                a, b = 0, 1
                for _ in range(n):
                    a, b = b, a + b
                for impl in impls:
                    log(f"[escalado fib({n})] {impl['name']}...")
                    res = measure(impl, script, str(a), args)
                    data["escalado"].setdefault(str(n), {})[impl["name"]] = res
                    save(out_path, data)

    log(f"resultados en {out_path}")


def save(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=1, ensure_ascii=False))


if __name__ == "__main__":
    main()
