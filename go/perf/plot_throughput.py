#!/usr/bin/env python3
"""Plot successful command lifecycle throughput from k6 summary exports."""

import argparse
import json
from pathlib import Path
from xml.sax.saxutils import escape

PALETTE = ["#25c2a0", "#f4b942", "#68a7ff", "#f47d73", "#d18bff", "#89c66d"]


def parse_run(value):
    try:
        label_and_size, path_text = value.split("=", 1)
        label, size_text = label_and_size.split(":", 1)
        size = int(size_text)
    except ValueError as error:
        raise argparse.ArgumentTypeError(
            "runs must use LABEL:SIZE=SUMMARY.json, for example array:100=perf/summary-100.json"
        ) from error
    if not label or size < 0:
        raise argparse.ArgumentTypeError("run label must be non-empty and size must be non-negative")
    return label, size, Path(path_text)


def read_rate(path):
    try:
        summary = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ValueError(f"cannot read k6 summary {path}: {error}") from error
    try:
        metric = summary["metrics"]["commands_completed"]
        rate = float(metric["rate"])
        count = int(metric["count"])
    except (KeyError, TypeError, ValueError) as error:
        raise ValueError(f"{path} has no commands_completed counter summary") from error
    if rate < 0 or count < 0:
        raise ValueError(f"{path} contains a negative throughput value")
    return rate, count


def chart_svg(runs):
    width, height = 1000, 620
    left, right, top, bottom = 105, 55, 115, 95
    plot_width, plot_height = width - left - right, height - top - bottom
    labels = sorted({label for label, _, _ in runs})
    sizes = sorted({size for _, size, _ in runs})
    values = {(label, size): read_rate(path) for label, size, path in runs}
    max_rate = max(rate for rate, _ in values.values())
    axis_max = max(1, max_rate * 1.15)
    colors = {label: PALETTE[index % len(PALETTE)] for index, label in enumerate(labels)}

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#101820"/>',
        '<style>text{font-family:Arial,Helvetica,sans-serif}.title{font-size:28px;font-weight:700;fill:#f4f7f8}.sub{font-size:14px;fill:#aebbc3}.axis{font-size:12px;fill:#9cabb4}.legend{font-size:13px;fill:#d9e1e5}.value{font-size:11px;fill:#e7edf0}</style>',
        '<text class="title" x="55" y="52">Command queue throughput</text>',
        '<text class="sub" x="55" y="80">Successfully completed POST, PATCH, GET, PATCH, DELETE cycles per second</text>',
    ]

    for tick in range(5):
        value = axis_max * tick / 4
        y = top + plot_height - value / axis_max * plot_height
        parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{width-right}" y2="{y:.1f}" stroke="#2a3943"/>')
        parts.append(f'<text class="axis" text-anchor="end" x="{left-12}" y="{y+4:.1f}">{value:.1f}</text>')

    group_width = plot_width / len(sizes)
    bar_width = min(54, group_width * 0.72 / max(1, len(labels)))
    group_bar_width = bar_width * len(labels)
    for size_index, size in enumerate(sizes):
        center = left + group_width * (size_index + 0.5)
        parts.append(f'<text class="axis" text-anchor="middle" x="{center:.1f}" y="{height-bottom+24}">{size}</text>')
        start_x = center - group_bar_width / 2
        for label_index, label in enumerate(labels):
            if (label, size) not in values:
                continue
            rate, count = values[(label, size)]
            bar_height = rate / axis_max * plot_height
            x = start_x + label_index * bar_width
            y = top + plot_height - bar_height
            parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_width-4:.1f}" height="{bar_height:.1f}" fill="{colors[label]}"><title>{escape(label)}, queue {size}: {rate:.2f} cycles/s, {count} completed</title></rect>')
            parts.append(f'<text class="value" text-anchor="middle" x="{x+bar_width/2-2:.1f}" y="{y-8:.1f}">{rate:.1f}</text>')

    for index, label in enumerate(labels):
        x = left + index * 190
        parts.append(f'<rect x="{x}" y="{height-43}" width="16" height="12" fill="{colors[label]}"/>')
        parts.append(f'<text class="legend" x="{x+24}" y="{height-33}">{escape(label)}</text>')

    parts.append(f'<text class="axis" text-anchor="middle" x="{left+plot_width/2}" y="{height-12}">Baseline queue size (commands)</text>')
    parts.append(f'<text class="axis" transform="translate(24 {top+plot_height/2}) rotate(-90)" text-anchor="middle">Successful cycles per second</text>')
    parts.append('</svg>')
    return '\n'.join(parts) + '\n'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", action="append", type=parse_run, required=True, metavar="LABEL:SIZE=SUMMARY.json")
    parser.add_argument("--output", type=Path, default=Path("perf/throughput.svg"))
    args = parser.parse_args()

    try:
        keys = [(label, size) for label, size, _ in args.run]
        if len(set(keys)) != len(keys):
            raise ValueError("each label and queue-size pair must be unique")
        args.output.write_text(chart_svg(args.run), encoding="utf-8")
    except (OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Wrote throughput chart to {args.output}")


if __name__ == "__main__":
    main()
