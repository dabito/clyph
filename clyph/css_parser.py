from __future__ import annotations

import re
from pathlib import Path
from urllib.request import urlopen

from .catalog import Record, codepoint_to_glyph, record_from_parts

COMMENT_RE = re.compile(r"/\*.*?\*/", re.S)
BLOCK_RE = re.compile(r"([^{}]+)\{([^{}]*)\}", re.S)
CONTENT_RE = re.compile(r"content\s*:\s*([\"'])(.*?)(?<!\\)\1\s*;?", re.S)
CLASS_RE = re.compile(r"\.([A-Za-z0-9_-]+)")


class ParseError(RuntimeError):
    pass


def load_source(source: str) -> str:
    if source.startswith(("http://", "https://")):
        with urlopen(source) as response:  # noqa: S310 - intentional network fetch for update
            return response.read().decode("utf-8")
    path = Path(source)
    return path.read_text(encoding="utf-8")


def decode_css_escape(value: str) -> str:
    chars: list[str] = []
    index = 0
    while index < len(value):
        char = value[index]
        if char != "\\":
            chars.append(char)
            index += 1
            continue
        index += 1
        if index >= len(value):
            break
        hex_digits: list[str] = []
        while index < len(value) and len(hex_digits) < 6 and value[index].lower() in "0123456789abcdef":
            hex_digits.append(value[index])
            index += 1
        if hex_digits:
            chars.append(chr(int("".join(hex_digits), 16)))
            if index < len(value) and value[index].isspace():
                index += 1
            continue
        chars.append(value[index])
        index += 1
    return "".join(chars)


def parse_css_catalog(text: str) -> list[Record]:
    text = COMMENT_RE.sub("", text)
    records: dict[str, Record] = {}
    for selector, body in BLOCK_RE.findall(text):
        content_match = CONTENT_RE.search(body)
        if not content_match:
            continue
        raw_content = content_match.group(2)
        decoded = decode_css_escape(raw_content)
        codepoint_hex = f"{ord(decoded[0]):x}" if decoded else ""
        if not codepoint_hex:
            continue
        aliases = [match.group(1) for match in CLASS_RE.finditer(selector)]
        if not aliases:
            continue
        for name in aliases:
            records[name] = record_from_parts(name, codepoint_hex, label="", aliases=())
    return sorted(records.values(), key=lambda record: record.name)
