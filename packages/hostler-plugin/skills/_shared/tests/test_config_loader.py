"""Unit tests for skills/_shared/config_loader."""

import json
import sys
import tempfile
import unittest
from pathlib import Path

# Add the parent directory (_shared) to the import path
_THIS_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(_THIS_DIR.parent))

from config_loader import (  # noqa: E402
    list_config_candidates,
    load_json_config,
)


class TestConfigLoader(unittest.TestCase):
    def setUp(self):
        # Temporary directory layout: tmp/scripts/script.py + tmp/config/foo.json
        self.tmp = tempfile.TemporaryDirectory()
        self.tmp_path = Path(self.tmp.name)
        self.scripts_dir = self.tmp_path / "scripts"
        self.config_dir = self.tmp_path / "config"
        self.scripts_dir.mkdir()
        self.config_dir.mkdir()
        self.fake_script = self.scripts_dir / "script.py"
        self.fake_script.write_text("# placeholder", encoding="utf-8")

    def tearDown(self):
        self.tmp.cleanup()

    def test_script_directory_takes_priority(self):
        """script_file-based ../config/foo.json wins as priority 1."""
        cfg_file = self.config_dir / "foo.json"
        cfg_file.write_text(json.dumps({"source": "script_dir"}), encoding="utf-8")

        # Even when project_root has a same-named file, script_dir wins
        legacy_dir = self.tmp_path / "legacy"
        legacy_dir.mkdir()
        (legacy_dir / "foo.json").write_text(
            json.dumps({"source": "project_root"}), encoding="utf-8"
        )

        result = load_json_config(
            config_name="foo.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
            legacy_relative_path="legacy/foo.json",
        )
        self.assertIsNotNone(result)
        self.assertEqual(result["source"], "script_dir")

    def test_project_root_fallback(self):
        """When script_dir has no file, fall back to the project_root legacy path."""
        legacy_dir = self.tmp_path / "legacy"
        legacy_dir.mkdir()
        (legacy_dir / "bar.json").write_text(
            json.dumps({"source": "project_root"}), encoding="utf-8"
        )

        result = load_json_config(
            config_name="bar.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
            legacy_relative_path="legacy/bar.json",
        )
        self.assertIsNotNone(result)
        self.assertEqual(result["source"], "project_root")

    def test_file_missing(self):
        """Returns None when neither path has the file."""
        result = load_json_config(
            config_name="missing.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
            legacy_relative_path="legacy/missing.json",
        )
        self.assertIsNone(result)

    def test_legacy_path_omitted(self):
        """When legacy_relative_path is omitted, only script_dir is searched."""
        cfg_file = self.config_dir / "only.json"
        cfg_file.write_text(json.dumps({"k": "v"}), encoding="utf-8")

        result = load_json_config(
            config_name="only.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
        )
        self.assertEqual(result, {"k": "v"})

        # Returns None when not in script_dir either
        missing = load_json_config(
            config_name="absent.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
        )
        self.assertIsNone(missing)

    def test_parse_failure_returns_none(self):
        """Invalid JSON returns None (the exception is not propagated)."""
        cfg_file = self.config_dir / "broken.json"
        cfg_file.write_text("{ invalid json", encoding="utf-8")

        result = load_json_config(
            config_name="broken.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
        )
        self.assertIsNone(result)

    def test_list_candidates(self):
        """list_config_candidates returns candidate paths (for debugging)."""
        candidates = list_config_candidates(
            config_name="x.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
            legacy_relative_path="rel/x.json",
        )
        self.assertEqual(len(candidates), 2)
        self.assertEqual(candidates[0], self.config_dir / "x.json")
        self.assertEqual(candidates[1], self.tmp_path / "rel" / "x.json")

        # Returns 1 entry when legacy is omitted
        candidates_solo = list_config_candidates(
            config_name="x.json",
            script_file=str(self.fake_script),
            project_root=self.tmp_path,
        )
        self.assertEqual(len(candidates_solo), 1)


if __name__ == "__main__":
    unittest.main()
