"""Turns results.csv into Amdahl's law tables and speedup graphs.

Usage: python3 analyze.py results.csv output_dir
"""
import csv
import sys
from collections import defaultdict

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt

THREAD_COUNTS = [2, 4, 6, 8, 12]
DIRS = ["small", "mixture", "big"]
MODES = ["parfiles", "parslices"]


def speedup_amdahl(parallel_fraction, n):
    """Amdahl's law: S(n) = 1 / ((1 - p) + p / n)."""
    return 1.0 / ((1.0 - parallel_fraction) + parallel_fraction / n)


def parallel_seconds(mode, row):
    """Time of the part of the program that the version parallelizes.

    parfiles: every image is processed (load + effects + save) by one worker,
              so all per-image work runs in parallel.
    parslices: only the effects are split across workers; decoding and encoding
              the PNG files stay sequential.
    """
    if mode == "parfiles":
        return row["load"] + row["effects"] + row["save"]
    return row["effects"]


def read_best_runs(path):
    """Returns {(mode, dir, threads): row of the fastest run}."""
    best = {}
    with open(path) as f:
        for r in csv.DictReader(f):
            row = {k: float(r[k]) for k in ("total", "load", "effects", "save")}
            key = (r["mode"], r["dir"], int(r["threads"]))
            if key not in best or row["total"] < best[key]["total"]:
                best[key] = row
    return best


def amdahl_tables(best):
    lines = []
    for mode in MODES:
        lines.append(f"## {mode}\n")
        header = ["data", "T1 total (s)", "parallel part (s)", "p", "limit 1/(1-p)"]
        header += [f"S({n}) expected" for n in THREAD_COUNTS]
        header += [f"S({n}) measured" for n in THREAD_COUNTS]
        lines.append("| " + " | ".join(header) + " |")
        lines.append("|" + "---|" * len(header))
        for d in DIRS:
            base = best.get((mode, d, 1))
            if base is None:
                continue
            t1 = base["total"]
            par = parallel_seconds(mode, base)
            p = min(par / t1, 1.0)  # timer rounding can push the ratio past 1
            limit = float("inf") if p >= 1 else 1.0 / (1.0 - p)
            expected = [speedup_amdahl(p, n) for n in THREAD_COUNTS]
            measured = [t1 / best[(mode, d, n)]["total"] if (mode, d, n) in best else float("nan")
                        for n in THREAD_COUNTS]
            cells = [d, f"{t1:.3f}", f"{par:.3f}", f"{p:.4f}", f"{limit:.2f}"]
            cells += [f"{s:.2f}" for s in expected] + [f"{s:.2f}" for s in measured]
            lines.append("| " + " | ".join(cells) + " |")
        lines.append("")
    return "\n".join(lines)


def plot_speedups(best, mode, out_path):
    plt.figure(figsize=(7, 5))
    top = 1.0
    for d in DIRS:
        base = best.get((mode, d, 1))
        if base is None:
            continue
        ys = [base["total"] / best[(mode, d, n)]["total"] for n in THREAD_COUNTS
              if (mode, d, n) in best]
        xs = [n for n in THREAD_COUNTS if (mode, d, n) in best]
        top = max(top, max(ys))
        plt.plot(xs, ys, marker="o", label=d)
    plt.title(f"Speedup of the '{mode}' version")
    plt.xlabel("Number of threads")
    plt.ylabel("Speedup (time with 1 thread / time with N threads)")
    plt.xticks(THREAD_COUNTS)
    plt.ylim(0.8, top * 1.1)
    plt.grid(True, alpha=0.3)
    plt.legend(title="data directory")
    plt.tight_layout()
    plt.savefig(out_path, dpi=150)
    plt.close()


def main():
    results, out_dir = sys.argv[1], sys.argv[2]
    best = read_best_runs(results)
    with open(f"{out_dir}/amdahl.md", "w") as f:
        f.write(amdahl_tables(best))
    for mode in MODES:
        plot_speedups(best, mode, f"{out_dir}/speedup-{mode}.png")


if __name__ == "__main__":
    main()
