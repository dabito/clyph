from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from .catalog import (
    CatalogError,
    Record,
    build_index,
    catalog_path,
    load_records,
    merge_catalog,
    save_records,
    search_records,
)
from .css_parser import load_source, parse_css_catalog

DEFAULT_LIMIT = 10
DEFAULT_SOURCE = "https://www.nerdfonts.com/assets/css/webfont.css"


def record_payload(record: Record) -> dict:
    return {
        "name": record.name,
        "codepoint": record.codepoint,
        "unicode": record.unicode,
        "glyph": record.glyph,
        "label": record.label,
        "aliases": list(record.aliases),
    }


def format_row(record: Record) -> str:
    label = record.label or "/".join(record.aliases) or "-"
    return f"{record.name}\t{record.codepoint}\t{record.glyph}\t{label}"


def cmd_search(args: argparse.Namespace) -> int:
    records = load_records()
    matches = search_records(records, args.query, args.limit)
    if args.json:
        print(json.dumps({"query": args.query, "matches": [record_payload(record) for record in matches]}, indent=2, ensure_ascii=False))
        return 0
    for record in matches:
        print(format_row(record))
    return 0


def cmd_get(args: argparse.Namespace) -> int:
    records = build_index(load_records())
    record = records.get(args.name)
    if record is None:
        print(f"not found: {args.name}", file=sys.stderr)
        return 1
    if args.json:
        print(json.dumps(record_payload(record), indent=2, ensure_ascii=False))
        return 0
    print(format_row(record))
    return 0


def cmd_glyph(args: argparse.Namespace) -> int:
    records = build_index(load_records())
    record = records.get(args.name)
    if record is None:
        print(f"not found: {args.name}", file=sys.stderr)
        return 1
    if args.json:
        print(json.dumps({"name": record.name, "glyph": record.glyph}, indent=2, ensure_ascii=False))
        return 0
    print(record.glyph)
    return 0


def cmd_codepoint(args: argparse.Namespace) -> int:
    records = build_index(load_records())
    record = records.get(args.name)
    if record is None:
        print(f"not found: {args.name}", file=sys.stderr)
        return 1
    if args.json:
        print(json.dumps({"name": record.name, "codepoint": record.codepoint}, indent=2, ensure_ascii=False))
        return 0
    print(record.codepoint)
    return 0


def cmd_update(args: argparse.Namespace) -> int:
    source = args.source or DEFAULT_SOURCE
    try:
        fresh = parse_css_catalog(load_source(source))
        if not fresh:
            raise ValueError("no glyph records parsed from source")
        try:
            existing = load_records()
        except CatalogError:
            existing = []
        merged = merge_catalog(existing, fresh)
        save_records(merged, catalog_path())
    except (OSError, ValueError, RuntimeError) as exc:
        print(f"update failed: {exc}", file=sys.stderr)
        return 1
    if args.json:
        print(json.dumps({"status": "updated", "records": len(merged), "catalog": str(catalog_path())}, indent=2, ensure_ascii=False))
        return 0
    print(f"updated {len(merged)} records")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="clyph", description="Deterministic Nerd Font glyph lookup")
    subparsers = parser.add_subparsers(dest="command", required=True)

    search = subparsers.add_parser("search", help="find Nerd Font glyphs by name or keyword")
    search.add_argument("query")
    search.add_argument("--limit", type=int, default=DEFAULT_LIMIT)
    search.add_argument("--json", action="store_true")
    search.set_defaults(func=cmd_search)

    get = subparsers.add_parser("get", help="return one glyph by exact name")
    get.add_argument("name")
    get.add_argument("--json", action="store_true")
    get.set_defaults(func=cmd_get)

    glyph = subparsers.add_parser("glyph", help="print only the glyph")
    glyph.add_argument("name")
    glyph.add_argument("--json", action="store_true")
    glyph.set_defaults(func=cmd_glyph)

    codepoint = subparsers.add_parser("codepoint", help="print only the codepoint")
    codepoint.add_argument("name")
    codepoint.add_argument("--json", action="store_true")
    codepoint.set_defaults(func=cmd_codepoint)

    update = subparsers.add_parser("update", help="refresh the local catalog")
    update.add_argument("--source", required=False, help="file path or URL to Nerd Fonts webfont.css")
    update.add_argument("--json", action="store_true")
    update.set_defaults(func=cmd_update)

    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    return args.func(args)
