#!/usr/bin/env python3
"""Plot k6 endpoint latency percentiles as grouped SVG bars."""

import argparse
import json
import math
from pathlib import Path
from xml.sax.saxutils import escape

METRICS = [
    ("commands_get_latency", "GET /commands", "#25c2a0"),
    ("commands_post_latency", "POST /commands", "#f4b942"),
    ("commands_patch_latency", "PATCH /commands/{id}/status", "#68a7ff"),
    ("commands_delete_latency", "DELETE /commands/{id}", "#f47d73"),
]
PERCENTILES = [(50, "p50"), (95, "p95"), (99, "p99")]


def percentile(values, percent):
    ordered = sorted(values)
    position = (len(ordered) - 1) * percent / 100
    lower = math.floor(position)
    upper = math.ceil(position)
    if lower == upper:
        return ordered[lower]
    fraction = position - lower
    return ordered[lower] * (1 - fraction) + ordered[upper] * fraction


def read_samples(path):
    samples = {name: [] for name, _, _ in METRICS}
    with path.open(encoding="utf-8") as source:
        for line_number, line in enumerate(source, start=1):
            if not line.strip():
                continue
            try:
                record = json.loads(line)
            except json.JSONDecodeError as error:
                raise ValueError(f"Invalid JSON on line {line_number} in {path}: {error}") from error
            if record.get("type") != "Point":
                continue
            metric = record.get("metric")
            point = record.get("data", {})
            if metric in samples and isinstance(point.get("value"), (int, float)):
                samples[metric].append(float(point["value"]))

    empty = [label for name, label, _ in METRICS if not samples[name]]
    if empty:
        raise ValueError(f"No custom latency samples in {path} for: " + ", ".join(empty))
    return samples


def percentile_summary(samples):
    return {
        metric: {label: percentile(values, value) for value, label in PERCENTILES}
        for metric, values in samples.items()
    }


def chart_svg(runs):
    width, height = 1450, 930
    left, right, top = 110, 65, 190
    panel_height, panel_gap, plot_height = 205, 35, 145
    plot_width = width - left - right
    categories = sorted(runs, key=lambda run: (run[1], run[0]))
    summaries = {(label, size): percentile_summary(samples) for label, size, samples in categories}
    metric_labels = {name: label for name, label, _ in METRICS}
    metric_colors = {name: color for name, _, color in METRICS}

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#101820"/>',
        '<style>text{font-family:Arial,Helvetica,sans-serif}.title{font-size:29px;font-weight:700;fill:#f4f7f8}.sub{font-size:15px;fill:#aebbc3}.panel{font-size:18px;font-weight:700;fill:#e7edf0}.axis{font-size:12px;fill:#9cabb4}.legend{font-size:13px;fill:#d9e1e5}</style>',
        '<text class="title" x="55" y="55">Command API latency</text>',
        '<text class="sub" x="55" y="84">Grouped endpoint bars · k6 response time in milliseconds</text>',
    ]

    legend_x = 55
    for metric, label, color in METRICS:
        parts.append(f'<rect x="{legend_x}" y="119" width="16" height="12" fill="{color}"/>')
        parts.append(f'<text class="legend" x="{legend_x + 24}" y="130">{escape(label)}</text>')
        legend_x += 335

    for panel_index, (_, percentile_label) in enumerate(PERCENTILES):
        panel_top = top + panel_index * (panel_height + panel_gap)
        plot_top = panel_top + 30
        plot_bottom = plot_top + plot_height
        panel_max = max(
            summaries[(label, size)][metric][percentile_label]
            for label, size, _ in categories
            for metric, _, _ in METRICS
        )
        axis_max = max(0.1, panel_max * 1.15)
        parts.append(f'<text class="panel" x="55" y="{panel_top + 14}">{percentile_label} response time</text>')
        for tick_index in range(5):
            value = axis_max * tick_index / 4
            y = plot_bottom - value / axis_max * plot_height
            parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{width-right}" y2="{y:.1f}" stroke="#2a3943"/>')
            parts.append(f'<text class="axis" text-anchor="end" x="{left-12}" y="{y+4:.1f}">{value:.2f}</text>')

        group_width = plot_width / len(categories)
        bar_width = min(24, group_width * 0.78 / len(METRICS))
        group_bar_width = bar_width * len(METRICS)
        for index, (implementation, size, _) in enumerate(categories):
            center = left + group_width * (index + 0.5)
            parts.append(f'<text class="axis" text-anchor="middle" x="{center:.1f}" y="{plot_bottom+17}">{size}</text>')
            parts.append(f'<text class="axis" text-anchor="middle" x="{center:.1f}" y="{plot_bottom+32}">{escape(implementation)}</text>')
            start_x = center - group_bar_width / 2
            for metric_index, (metric, label, _) in enumerate(METRICS):
                value = summaries[(implementation, size)][metric][percentile_label]
                bar_height = value / axis_max * plot_height
                x = start_x + metric_index * bar_width
                y = plot_bottom - bar_height
                color = metric_colors[metric]
                parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bar_width-2:.1f}" height="{bar_height:.1f}" fill="{color}"><title>{escape(implementation)} queue {size}, {escape(label)}, {percentile_label}: {value:.3f} ms</title></rect>')

        if panel_index == len(PERCENTILES) - 1:
            parts.append(f'<text class="axis" text-anchor="middle" x="{left+plot_width/2}" y="{plot_bottom+53}">Baseline queue size (commands), implementation</text>')

    parts.append('</svg>')
    return '\n'.join(parts) + '\n'


def parse_run(value):
    if "=" not in value:
        raise argparse.ArgumentTypeError("runs must use SIZE=PATH or LABEL:SIZE=PATH")
    run_text, path_text = value.split("=", 1)
    if ":" in run_text:
        label, size_text = run_text.split(":", 1)
        if not label:
            raise argparse.ArgumentTypeError("implementation label must not be empty")
    else:
        label, size_text = "run", run_text
    try:
        size = int(size_text)
    except ValueError as error:
        raise argparse.ArgumentTypeError(f"queue size is not an integer: {size_text}") from error
    if size < 0:
        raise argparse.ArgumentTypeError("queue size must not be negative")
    return label, size, Path(path_text)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", action="append", type=parse_run, required=True, metavar="LABEL:SIZE=PATH")
    parser.add_argument("--output", type=Path, default=Path("perf/queue-size-latency.svg"))
    args = parser.parse_args()

    try:
        keys = [(label, size) for label, size, _ in args.run]
        if len(set(keys)) != len(keys):
            raise ValueError("implementation and queue-size pairs must be unique")
        runs = [
            (label, size, read_samples(path))
            for label, size, path in sorted(args.run, key=lambda run: (run[1], run[0]))
        ]
        args.output.write_text(chart_svg(runs), encoding="utf-8")
    except (OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Wrote queue-size latency chart to {args.output}")


if __name__ == "__main__":
    main()
