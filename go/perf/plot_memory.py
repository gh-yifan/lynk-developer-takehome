#!/usr/bin/env python3
"""Plot retained heap and cumulative allocations from pprof summaries."""

import argparse
import csv
from pathlib import Path
from xml.sax.saxutils import escape

PALETTE = ["#25c2a0", "#f4b942", "#68a7ff", "#f47d73", "#d18bff", "#89c66d"]
METRICS = [
    ("inuse_bytes", "Live heap after forced GC", "retained MiB"),
    ("alloc_bytes", "Cumulative allocation", "allocated MiB"),
]
MIB = 1024 * 1024


def parse_run(value):
    try:
        label_and_size, path_text = value.split("=", 1)
        label, size_text = label_and_size.split(":", 1)
        size = int(size_text)
    except ValueError as error:
        raise argparse.ArgumentTypeError(
            "runs must use LABEL:SIZE=PROFILE_DIR, for example array:20000=perf/profiles/array-20000-TIMESTAMP"
        ) from error
    if not label or size < 0:
        raise argparse.ArgumentTypeError("run label must be non-empty and size must be non-negative")
    return label, size, Path(path_text)


def read_summary(profile_dir):
    path = profile_dir / "memory-summary.tsv"
    try:
        with path.open(encoding="utf-8", newline="") as source:
            rows = list(csv.DictReader(source, delimiter="\t"))
        if len(rows) != 1:
            raise ValueError(f"expected one summary row in {path}")
        row = rows[0]
        return {key: int(row[key]) for key, _, _ in METRICS}
    except (OSError, KeyError, ValueError) as error:
        raise ValueError(f"cannot read memory summary in {profile_dir}: {error}") from error


def chart_svg(runs):
    width, height = 1100, 690
    left, right, top = 115, 55, 125
    plot_width = width - left - right
    panel_height, panel_gap = 205, 55
    labels = sorted({label for label, _, _ in runs})
    sizes = sorted({size for _, size, _ in runs})
    values = {(label, size): read_summary(path) for label, size, path in runs}
    colors = {label: PALETTE[index % len(PALETTE)] for index, label in enumerate(labels)}

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#101820"/>',
        '<style>text{font-family:Arial,Helvetica,sans-serif}.title{font-size:28px;font-weight:700;fill:#f4f7f8}.sub{font-size:14px;fill:#aebbc3}.panel{font-size:17px;font-weight:700;fill:#e7edf0}.axis{font-size:12px;fill:#9cabb4}.legend{font-size:13px;fill:#d9e1e5}.value{font-size:11px;fill:#e7edf0}</style>',
        '<text class="title" x="55" y="52">Command queue memory profile</text>',
        '<text class="sub" x="55" y="80">Same Go version, workload, and queue size per comparison; values from pprof</text>',
    ]

    for index, label in enumerate(labels):
        x = 55 + index * 190
        parts.append(f'<rect x="{x}" y="99" width="16" height="12" fill="{colors[label]}"/>')
        parts.append(f'<text class="legend" x="{x+24}" y="109">{escape(label)}</text>')

    for panel_index, (key, title, axis_label) in enumerate(METRICS):
        panel_top = top + panel_index * (panel_height + panel_gap)
        plot_top = panel_top + 28
        plot_height = panel_height - 40
        max_value = max(summary[key] / MIB for summary in values.values())
        axis_max = max(0.01, max_value * 1.15)
        parts.append(f'<text class="panel" x="55" y="{panel_top+14}">{title}</text>')
        for tick in range(5):
            value = axis_max * tick / 4
            y = plot_top + plot_height - value / axis_max * plot_height
            parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{width-right}" y2="{y:.1f}" stroke="#2a3943"/>')
            parts.append(f'<text class="axis" text-anchor="end" x="{left-12}" y="{y+4:.1f}">{value:.2f}</text>')

        group_width = plot_width / len(sizes)
        bar_width = min(54, group_width * 0.72 / max(1, len(labels)))
        group_bar_width = bar_width * len(labels)
        for size_index, size in enumerate(sizes):
            center = left + group_width * (size_index + 0.5)
            parts.append(f'<text class="axis" text-anchor="middle" x="{center:.1f}" y="{plot_top+plot_height+20}">{size}</text>')
            start_x = center - group_bar_width / 2
            for label_index, label in enumerate(labels):
                if (label, size) not in values:
                    continue
                value = values[(label, size)][key] / MIB
                bar_height = value / axis_max * plot_height
                x = start_x + label_index * bar_width
                y = plot_top + plot_height - bar_height
                parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_width-4:.1f}" height="{bar_height:.1f}" fill="{colors[label]}"><title>{escape(label)}, queue {size}: {value:.3f} MiB</title></rect>')
                parts.append(f'<text class="value" text-anchor="middle" x="{x+bar_width/2-2:.1f}" y="{y-8:.1f}">{value:.2f}</text>')

        if panel_index == len(METRICS) - 1:
            parts.append(f'<text class="axis" text-anchor="middle" x="{left+plot_width/2}" y="{height-18}">Baseline queue size (commands)</text>')
        parts.append(f'<text class="axis" transform="translate(25 {plot_top+plot_height/2}) rotate(-90)" text-anchor="middle">{axis_label} (MiB)</text>')

    parts.append('</svg>')
    return '\n'.join(parts) + '\n'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", action="append", type=parse_run, required=True, metavar="LABEL:SIZE=PROFILE_DIR")
    parser.add_argument("--output", type=Path, default=Path("perf/memory-comparison.svg"))
    args = parser.parse_args()

    try:
        keys = [(label, size) for label, size, _ in args.run]
        if len(set(keys)) != len(keys):
            raise ValueError("each label and queue-size pair must be unique")
        args.output.write_text(chart_svg(args.run), encoding="utf-8")
    except (OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Wrote memory chart to {args.output}")


if __name__ == "__main__":
    main()
