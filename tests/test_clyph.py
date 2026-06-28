from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PROGRESS_GLYPH = chr(0xF1395)
CHECK_GLYPH = chr(0xF012C)

FIXTURE_RECORDS = [
    {
        "name": "nf-fa-check",
        "codepoint": "f00c",
        "unicode": "\\uf00c",
        "glyph": chr(0xF00C),
        "label": "done",
        "aliases": ["done"],
    },
    {
        "name": "nf-fa-circle",
        "codepoint": "f111",
        "unicode": "\\uf111",
        "glyph": chr(0xF111),
        "label": "offline",
        "aliases": ["offline"],
    },
    {
        "name": "nf-fa-circle_o",
        "codepoint": "f10c",
        "unicode": "\\uf10c",
        "glyph": chr(0xF10C),
        "label": "offline outline",
        "aliases": ["offline-outline"],
    },
    {
        "name": "nf-md-check",
        "codepoint": "f012c",
        "unicode": "\\U000f012c",
        "glyph": CHECK_GLYPH,
        "label": "check",
        "aliases": [],
    },
    {
        "name": "nf-md-circle_half",
        "codepoint": "f1395",
        "unicode": "\\U000f1395",
        "glyph": PROGRESS_GLYPH,
        "label": "progress",
        "aliases": ["progress"],
    },
]


class ClyphTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.catalog_path = Path(self.tmp.name) / "catalog.json"
        self.catalog_path.write_text(
            json.dumps({"records": FIXTURE_RECORDS}, indent=2, ensure_ascii=False) + "\n",
            encoding="utf-8",
        )

    def tearDown(self):
        self.tmp.cleanup()

    def run_cli(self, *args: str, env: dict[str, str] | None = None, cwd: Path | None = None):
        merged_env = os.environ.copy()
        merged_env["CLYPH_CATALOG_PATH"] = str(self.catalog_path)
        if env:
            merged_env.update(env)
        return subprocess.run(
            [sys.executable, "-m", "clyph", *args],
            cwd=cwd or ROOT,
            env=merged_env,
            text=True,
            capture_output=True,
            check=False,
        )

    def test_checked_in_catalog_is_full_nerd_font_export(self):
        payload = json.loads((ROOT / "data" / "catalog.json").read_text(encoding="utf-8"))
        records = payload["records"]
        self.assertGreater(len(records), 10_000)
        lookup = {item["name"]: item for item in records}
        self.assertEqual(lookup["nf-md-check"]["codepoint"], "f012c")
        self.assertEqual(lookup["nf-fa-circle"]["codepoint"], "f111")

    def test_search_plain(self):
        proc = self.run_cli("search", "circle")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        lines = [line for line in proc.stdout.splitlines() if line.strip()]
        self.assertEqual(
            lines,
            [
                "nf-fa-circle\tf111\t\toffline",
                "nf-fa-circle_o\tf10c\t\toffline outline",
                f"nf-md-circle_half\tf1395\t{PROGRESS_GLYPH}\tprogress",
            ],
        )

    def test_search_json(self):
        proc = self.run_cli("search", "check", "--json")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        payload = json.loads(proc.stdout)
        self.assertEqual(payload["query"], "check")
        self.assertEqual([item["name"] for item in payload["matches"]], ["nf-fa-check", "nf-md-check"])
        self.assertEqual(payload["matches"][1]["unicode"], "\\U000f012c")

    def test_get_and_scalar_commands(self):
        proc = self.run_cli("get", "nf-md-check")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), f"nf-md-check\tf012c\t{CHECK_GLYPH}\tcheck")

        proc = self.run_cli("glyph", "nf-md-check")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), CHECK_GLYPH)

        proc = self.run_cli("glyph", "nf-md-check", "--json")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(json.loads(proc.stdout), {"name": "nf-md-check", "glyph": CHECK_GLYPH})

        proc = self.run_cli("codepoint", "nf-md-check")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "f012c")

        proc = self.run_cli("codepoint", "nf-md-check", "--json")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(json.loads(proc.stdout), {"name": "nf-md-check", "codepoint": "f012c"})

    def test_css_parser_handles_whitespace_multiple_selectors_and_duplicates(self):
        from clyph.css_parser import parse_css_catalog

        css = """
        /* comment */
        .nf-a:before,
        .nf-b::before {
          content: "\\f111";
        }

        .nf-a:before { content : "\\f222"; }
        .nf-c:before,
        .nf-d:before {
          content: "\\f1395";
        }
        """
        records = parse_css_catalog(css)
        self.assertEqual([record.name for record in records], ["nf-a", "nf-b", "nf-c", "nf-d"])
        lookup = {record.name: record for record in records}
        self.assertEqual(lookup["nf-a"].codepoint, "f222")
        self.assertEqual(lookup["nf-b"].codepoint, "f111")
        self.assertEqual(lookup["nf-c"].codepoint, "f1395")
        self.assertEqual(lookup["nf-c"].glyph, PROGRESS_GLYPH)

    def test_update_is_atomic_and_preserves_existing_catalog_on_failure(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp = Path(tmpdir)
            catalog_path = tmp / "catalog.json"
            catalog_path.write_text(
                '{"records":[{"name":"keep","codepoint":"f000","unicode":"\\uf000","glyph":"","label":"keep","aliases":[]}]}\n',
                encoding="utf-8",
            )
            before = catalog_path.read_text(encoding="utf-8")
            env = {"CLYPH_CATALOG_PATH": str(catalog_path)}
            proc = self.run_cli("update", "--source", str(tmp / "missing.css"), env=env)
            self.assertNotEqual(proc.returncode, 0)
            self.assertEqual(catalog_path.read_text(encoding="utf-8"), before)

    def test_update_from_file(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp = Path(tmpdir)
            catalog_path = tmp / "catalog.json"
            css_path = tmp / "webfont.css"
            css_path.write_text('.nf-x:before { content: "\\f123"; }', encoding="utf-8")
            env = {"CLYPH_CATALOG_PATH": str(catalog_path)}
            proc = self.run_cli("update", "--source", str(css_path), "--json", env=env)
            self.assertEqual(proc.returncode, 0, proc.stderr)
            self.assertEqual(json.loads(proc.stdout)["records"], 1)
            payload = json.loads(catalog_path.read_text(encoding="utf-8"))
            self.assertEqual([item["name"] for item in payload["records"]], ["nf-x"])
            self.assertEqual(payload["records"][0]["codepoint"], "f123")


    def test_update_rejects_sources_with_no_glyph_records(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp = Path(tmpdir)
            catalog_path = tmp / "catalog.json"
            catalog_path.write_text(
                '{"records":[{"name":"keep","codepoint":"f000","unicode":"\\uf000","glyph":"","label":"keep","aliases":[]}]}\n',
                encoding="utf-8",
            )
            before = catalog_path.read_text(encoding="utf-8")
            empty_css = tmp / "empty.css"
            empty_css.write_text("body { color: red; }", encoding="utf-8")
            env = {"CLYPH_CATALOG_PATH": str(catalog_path)}
            proc = self.run_cli("update", "--source", str(empty_css), "--json", env=env)
            self.assertNotEqual(proc.returncode, 0)
            self.assertIn("no glyph records parsed", proc.stderr)
            self.assertEqual(catalog_path.read_text(encoding="utf-8"), before)
if __name__ == "__main__":
    unittest.main()
