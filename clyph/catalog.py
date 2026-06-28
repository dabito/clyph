from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_CATALOG_PATH = ROOT / "data" / "catalog.json"
CATALOG_ENV = "CLYPH_CATALOG_PATH"


@dataclass(frozen=True)
class Record:
    name: str
    codepoint: str
    unicode: str
    glyph: str
    label: str = ""
    aliases: tuple[str, ...] = ()

    @classmethod
    def from_dict(cls, data: dict) -> "Record":
        aliases = tuple(sorted({str(alias).strip() for alias in data.get("aliases", []) if str(alias).strip()}))
        return cls(
            name=str(data["name"]),
            codepoint=str(data["codepoint"]),
            unicode=str(data["unicode"]),
            glyph=str(data["glyph"]),
            label=str(data.get("label", "")),
            aliases=aliases,
        )

    def to_dict(self) -> dict:
        return {
            "name": self.name,
            "codepoint": self.codepoint,
            "unicode": self.unicode,
            "glyph": self.glyph,
            "label": self.label,
            "aliases": list(self.aliases),
        }


class CatalogError(RuntimeError):
    pass


def catalog_path() -> Path:
    override = os.environ.get(CATALOG_ENV)
    if override:
        return Path(override)
    return DEFAULT_CATALOG_PATH


def unicode_escape(codepoint: int) -> str:
    if codepoint <= 0xFFFF:
        return f"\\u{codepoint:04x}"
    return f"\\U{codepoint:08x}"


def codepoint_to_glyph(codepoint_hex: str) -> str:
    return chr(int(codepoint_hex, 16))


def record_from_parts(name: str, codepoint_hex: str, *, label: str = "", aliases: Iterable[str] = ()) -> Record:
    normalized = codepoint_hex.lower().removeprefix("0x")
    return Record(
        name=name,
        codepoint=normalized,
        unicode=unicode_escape(int(normalized, 16)),
        glyph=codepoint_to_glyph(normalized),
        label=label,
        aliases=tuple(sorted({alias.strip() for alias in aliases if alias.strip()})),
    )


def load_records(path: Path | None = None) -> list[Record]:
    path = path or catalog_path()
    if not path.exists():
        raise CatalogError(f"catalog not found: {path}")
    data = json.loads(path.read_text(encoding="utf-8"))
    records = [Record.from_dict(item) for item in data.get("records", [])]
    return sorted(records, key=lambda record: record.name)


def save_records(records: Iterable[Record], path: Path | None = None) -> None:
    path = path or catalog_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    ordered = sorted(records, key=lambda record: record.name)
    payload = {"records": [record.to_dict() for record in ordered]}
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    os.replace(tmp, path)


def build_index(records: Iterable[Record]) -> dict[str, Record]:
    index: dict[str, Record] = {}
    for record in records:
        index[record.name] = record
    return index


def search_records(records: Iterable[Record], query: str, limit: int | None = None) -> list[Record]:
    needle = query.strip().lower()
    matches: list[Record] = []
    for record in sorted(records, key=lambda item: item.name):
        haystacks = [record.name.lower(), record.label.lower(), *[alias.lower() for alias in record.aliases]]
        if not needle or any(needle in hay for hay in haystacks):
            matches.append(record)
            if limit is not None and len(matches) >= limit:
                break
    return matches


def merge_catalog(existing: Iterable[Record], fresh: Iterable[Record]) -> list[Record]:
    metadata = {record.name: record for record in existing}
    merged: list[Record] = []
    for record in fresh:
        old = metadata.get(record.name)
        if old:
            merged.append(
                Record(
                    name=record.name,
                    codepoint=record.codepoint,
                    unicode=record.unicode,
                    glyph=record.glyph,
                    label=old.label,
                    aliases=old.aliases,
                )
            )
        else:
            merged.append(record)
    return sorted(merged, key=lambda record: record.name)
