#!/usr/bin/env python3
"""Plot k6 delete latency samples against queue depth from delete_staircase.js."""

import argparse
import json
from pathlib import Path
from xml.sax.saxutils import escape

COLORS = ["#25c2a0", "#f4b942", "#68a7ff", "#f47d73", "#d18bff", "#89c66d"]


def parse_run(value):
    if "=" not in value:
        raise argparse.ArgumentTypeError("runs must use LABEL=PATH, for example main=perf/results/main/delete-staircase-20000.ndjson")
    label, path = value.split("=", 1)
    if not label:
        raise argparse.ArgumentTypeError("run label must not be empty")
    return label, Path(path)


def read_samples(path):
    samples = {}
    with path.open(encoding="utf-8") as source:
        for line_number, line in enumerate(source, start=1):
            if not line.strip():
                continue
            try:
                point = json.loads(line)
            except json.JSONDecodeError as error:
                raise ValueError(f"Invalid JSON on line {line_number} in {path}: {error}") from error
            if point.get("type") != "Point" or point.get("metric") != "commands_delete_staircase_latency":
                continue
            data = point.get("data", {})
            tags = data.get("tags", {})
            depth_band = tags.get("queue_depth_band")
            value = data.get("value")
            if depth_band is None or not isinstance(value, (int, float)):
                continue
            try:
                band_start, band_end = (int(part) for part in depth_band.split("-", 1))
            except ValueError as error:
                raise ValueError(f"Invalid queue_depth_band {depth_band!r} in {path}") from error
            samples.setdefault((band_start, band_end), []).append(float(value))

    if not samples:
        raise ValueError(f"No delete staircase samples found in {path}")
    return samples


def percentile(values, percentage):
    ordered = sorted(values)
    position = (len(ordered) - 1) * percentage / 100
    lower = int(position)
    upper = min(lower + 1, len(ordered) - 1)
    fraction = position - lower
    return ordered[lower] * (1 - fraction) + ordered[upper] * fraction


def make_svg(runs, title, y_percentile):
    width, height = 1200, 620
    left, right, top, bottom = 90, 45, 115, 75
    plot_width = width - left - right
    plot_height = height - top - bottom
    all_samples = [(label, read_samples(path)) for label, path in runs]
    max_depth = max(max(start for start, _ in samples) for _, samples in all_samples)
    band_means = [
        sum(values) / len(values)
        for _, samples in all_samples
        for values in samples.values()
    ]
    y_limit = percentile(band_means, y_percentile)
    axis_max = max(0.001, y_limit * 1.1)
    clipped_count = sum(value > axis_max for value in band_means)
    colors = {label: COLORS[index % len(COLORS)] for index, (label, _) in enumerate(all_samples)}

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" fill="#101820"/>',
        '<style>text{font-family:Arial,Helvetica,sans-serif}.title{font-size:27px;font-weight:700;fill:#f4f7f8}.sub{font-size:14px;fill:#aebbc3}.axis{font-size:12px;fill:#9cabb4}.legend{font-size:13px;fill:#d9e1e5}.line{fill:none;stroke-width:2.5}.point{stroke:#101820;stroke-width:1.5}.clip{stroke:#101820;stroke-width:1}</style>',
        f'<text class="title" x="55" y="52">{escape(title)}</text>',
        f'<text class="sub" x="55" y="78">Sequential DELETE requests · y-axis capped at p{y_percentile} of band means; triangles mark clipped outliers ({clipped_count})</text>',
    ]

    for index, (label, _) in enumerate(all_samples):
        x = 70 + index * 190
        color = colors[label]
        parts.append(f'<line x1="{x}" y1="98" x2="{x+22}" y2="98" stroke="{color}" stroke-width="3"/>')
        parts.append(f'<text class="legend" x="{x+30}" y="102">{escape(label)}</text>')

    for tick in range(5):
        latency = axis_max * tick / 4
        y = top + plot_height - latency / axis_max * plot_height
        parts.append(f'<line x1="{left}" y1="{y:.1f}" x2="{width-right}" y2="{y:.1f}" stroke="#2a3943"/>')
        tick_format = ".3f" if axis_max < 1 else ".2f"
        parts.append(f'<text class="axis" text-anchor="end" x="{left-12}" y="{y+4:.1f}">{format(latency, tick_format)}</text>')

    for label, samples in all_samples:
        bands = sorted(samples, reverse=True)
        points = []
        for band_start, band_end in bands:
            values = samples[(band_start, band_end)]
            latency = sum(values) / len(values)
            depth_midpoint = (band_start + band_end) / 2
            x = left + (max_depth - depth_midpoint) / max(1, max_depth - 1) * plot_width
            clipped = latency > axis_max
            y = top if clipped else top + plot_height - latency / axis_max * plot_height
            points.append((x, y, band_start, band_end, latency, len(values), clipped))

        coordinates = " ".join(f"{x:.1f},{y:.1f}" for x, y, *_ in points)
        color = colors[label]
        parts.append(f'<polyline class="line" stroke="{color}" points="{coordinates}"/>')
        for x, y, band_start, band_end, latency, count, clipped in points:
            if clipped:
                marker = f'<polygon class="clip" points="{x:.1f},{y-7:.1f} {x-6:.1f},{y+4:.1f} {x+6:.1f},{y+4:.1f}" fill="{color}"><title>{escape(label)} depths {band_start}-{band_end}: mean {latency:.3f} ms over {count} DELETE requests (above displayed y-axis)</title></polygon>'
            else:
                marker = f'<circle class="point" cx="{x:.1f}" cy="{y:.1f}" r="4" fill="{color}"><title>{escape(label)} depths {band_start}-{band_end}: mean {latency:.3f} ms over {count} DELETE requests</title></circle>'
            parts.append(marker)

    for depth in sorted({max_depth, max(1, max_depth // 2), 1}):
        x = left + (max_depth - depth) / max(1, max_depth - 1) * plot_width
        parts.append(f'<text class="axis" text-anchor="middle" x="{x:.1f}" y="{height-bottom+22}">{depth}</text>')
    parts.append(f'<text class="axis" text-anchor="middle" x="{left+plot_width/2}" y="{height-18}">Commands remaining before DELETE</text>')
    parts.append(f'<text class="axis" transform="translate(22 {top+plot_height/2}) rotate(-90)" text-anchor="middle">DELETE latency (ms)</text>')
    parts.append('</svg>')
    return "\n".join(parts) + "\n"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", action="append", type=parse_run, required=True, metavar="LABEL=PATH", help="k6 JSON result; repeat to compare implementations")
    parser.add_argument("--output", type=Path, default=Path("perf/delete-staircase.svg"))
    parser.add_argument("--title", default="DELETE latency while draining queue")
    parser.add_argument("--y-percentile", type=float, default=99.0, help="percentile of band means used as y-axis cap (default: 99); clipped points are marked")
    args = parser.parse_args()

    try:
        if not 50 <= args.y_percentile <= 100:
            raise ValueError("--y-percentile must be between 50 and 100")
        labels = [label for label, _ in args.run]
        if len(set(labels)) != len(labels):
            raise ValueError("run labels must be unique")
        args.output.write_text(make_svg(args.run, args.title, args.y_percentile), encoding="utf-8")
    except (OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Wrote delete staircase chart to {args.output}")


if __name__ == "__main__":
    main()
