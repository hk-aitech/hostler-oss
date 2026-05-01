"""Unit tests for claude-md-audit audit.py.

Uses Python standard library unittest (no external dependencies).
Run: cd skills/claude-md-audit && python3 -m unittest tests.test_audit -v
Or:  python3 -m unittest discover skills/claude-md-audit/tests -v
"""

from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

# Add the scripts directory to sys.path so audit.py can be imported
_SCRIPT_DIR = Path(__file__).resolve().parent.parent / "scripts"
sys.path.insert(0, str(_SCRIPT_DIR))

import audit  # noqa: E402 — must follow sys.path manipulation


# ──────────────────────────────────────────────────────────────────────
# parse_sections
# ──────────────────────────────────────────────────────────────────────


class TestParseSections(unittest.TestCase):
    def test_basic_l2_sections(self) -> None:
        text = "# Title\n\n## Section A\nContent A\n\n## Section B\nContent B\n"
        sections = audit.parse_sections(text)
        self.assertEqual(len(sections), 2)
        self.assertEqual(sections[0].title, "Section A")
        self.assertEqual(sections[1].title, "Section B")

    def test_ignores_fenced_code_blocks(self) -> None:
        text = (
            "## Real Section\n"
            "text\n"
            "```\n"
            "## Fake Section inside code\n"
            "```\n"
            "## Another Real\n"
        )
        sections = audit.parse_sections(text)
        self.assertEqual(len(sections), 2)
        self.assertEqual(sections[0].title, "Real Section")
        self.assertEqual(sections[1].title, "Another Real")

    def test_ignores_l1_and_l3(self) -> None:
        text = "# Title (H1)\n## Valid L2\n### Sub L3\nContent\n"
        sections = audit.parse_sections(text)
        self.assertEqual(len(sections), 1)
        self.assertEqual(sections[0].title, "Valid L2")


# ──────────────────────────────────────────────────────────────────────
# count_imports
# ──────────────────────────────────────────────────────────────────────


class TestCountImports(unittest.TestCase):
    def test_basic_imports(self) -> None:
        text = "@import/one.md and @another/path.md"
        self.assertEqual(audit.count_imports(text), 2)

    def test_ignores_code_block_imports(self) -> None:
        text = (
            "@real-import.md\n"
            "```\n"
            "@fake-import.md\n"
            "@another-fake.md\n"
            "```\n"
            "@another-real.md\n"
        )
        self.assertEqual(audit.count_imports(text), 2)

    def test_ignores_inline_backtick(self) -> None:
        # @token inside backtick should not count (preceded by `)
        text = "`@backtick.md` and @normal.md"
        self.assertEqual(audit.count_imports(text), 1)


# ──────────────────────────────────────────────────────────────────────
# classify_section
# ──────────────────────────────────────────────────────────────────────


class TestClassifySection(unittest.TestCase):
    def test_skill_development(self) -> None:
        # Matches the runtime regex for "skill development" headings
        principle, target, _paths, severity = audit.classify_section("Skill Development Standards")
        self.assertEqual(principle, "P4")
        self.assertIn("skill-development", target)
        self.assertEqual(severity, "high")

    def test_mcp_tool(self) -> None:
        # Matches the runtime regex for "mcp tool development" headings
        principle, _target, _paths, severity = audit.classify_section("MCP Tool Development")
        self.assertEqual(principle, "P4")
        self.assertEqual(severity, "high")

    def test_environment_variables(self) -> None:
        # Matches the runtime regex for "environment variables" headings
        principle, _target, _paths, severity = audit.classify_section("Environment Variables")
        self.assertEqual(principle, "P3")
        self.assertEqual(severity, "medium")

    def test_unknown_title_returns_empty(self) -> None:
        # An arbitrary heading that does not match any classifier
        principle, target, paths, severity = audit.classify_section("Completely Arbitrary Section")
        self.assertEqual(principle, "")
        self.assertEqual(target, "")
        self.assertEqual(paths, "")
        self.assertEqual(severity, "")


# ──────────────────────────────────────────────────────────────────────
# _is_excluded
# ──────────────────────────────────────────────────────────────────────


class TestIsExcluded(unittest.TestCase):
    def setUp(self) -> None:
        self.excludes = [
            "archive/",
            "works/",
            "CHANGELOG.md",
            "docs/07-knowledge/",
        ]

    def test_directory_prefix_match(self) -> None:
        self.assertTrue(audit._is_excluded("archive/foo/bar.md", self.excludes))
        self.assertTrue(audit._is_excluded("works/tasks/Task.md", self.excludes))
        self.assertTrue(audit._is_excluded("docs/07-knowledge/mistakes.md", self.excludes))

    def test_exact_file_match(self) -> None:
        self.assertTrue(audit._is_excluded("CHANGELOG.md", self.excludes))

    def test_not_excluded(self) -> None:
        self.assertFalse(audit._is_excluded("README.md", self.excludes))
        self.assertFalse(audit._is_excluded("skills/foo/SKILL.md", self.excludes))
        self.assertFalse(audit._is_excluded("docs/04-guides/guide.md", self.excludes))

    def test_not_false_prefix(self) -> None:
        # "worksheet.md" must not be treated as starting with "works/"
        self.assertFalse(audit._is_excluded("worksheet.md", self.excludes))
        # "CHANGELOG.md.backup" must not be an exact match
        self.assertFalse(audit._is_excluded("CHANGELOG.md.backup", self.excludes))


# ──────────────────────────────────────────────────────────────────────
# _load_deprecated_config
# ──────────────────────────────────────────────────────────────────────


class TestLoadDeprecatedConfig(unittest.TestCase):
    """Isolate the _SCRIPT_CONFIG_DIR fallback into tmp and verify only the project_root logic.

    audit.py looks both at the script directory and at project_root, so the tests
    temporarily redirect the script-relative path to a nonexistent location.
    """

    def setUp(self) -> None:
        self._original_script_dir = audit._SCRIPT_CONFIG_DIR
        # Redirect the script-relative path to a nonexistent location
        audit._SCRIPT_CONFIG_DIR = Path("/nonexistent-for-test-isolation")

    def tearDown(self) -> None:
        audit._SCRIPT_CONFIG_DIR = self._original_script_dir

    def test_returns_none_when_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            result = audit._load_deprecated_config(Path(tmp))
            self.assertIsNone(result)

    def test_loads_valid_json(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            config_dir = Path(tmp) / "skills" / "claude-md-audit" / "config"
            config_dir.mkdir(parents=True)
            config_path = config_dir / "deprecated-tokens.json"
            config_data = {
                "tokens": [{"pattern": "foo", "removed_in": "X", "replacement": "Y"}],
                "exclude_paths": ["archive/"],
            }
            config_path.write_text(json.dumps(config_data), encoding="utf-8")

            result = audit._load_deprecated_config(Path(tmp))
            self.assertIsNotNone(result)
            assert result is not None  # mypy guard
            self.assertEqual(len(result["tokens"]), 1)
            self.assertEqual(result["tokens"][0]["pattern"], "foo")

    def test_returns_none_on_invalid_json(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            config_dir = Path(tmp) / "skills" / "claude-md-audit" / "config"
            config_dir.mkdir(parents=True)
            (config_dir / "deprecated-tokens.json").write_text(
                "{ this is not valid json",
                encoding="utf-8",
            )
            result = audit._load_deprecated_config(Path(tmp))
            self.assertIsNone(result)


# ──────────────────────────────────────────────────────────────────────
# AuditResult.verdict
# ──────────────────────────────────────────────────────────────────────


class TestAuditResultVerdict(unittest.TestCase):
    def test_excellent(self) -> None:
        result = audit.AuditResult(path=Path("dummy"), total_lines=50)
        name, _reason, exit_code = result.verdict
        self.assertEqual(name, "Excellent")
        self.assertEqual(exit_code, audit.EXIT_OK)

    def test_good(self) -> None:
        result = audit.AuditResult(path=Path("dummy"), total_lines=150)
        name, _reason, exit_code = result.verdict
        self.assertEqual(name, "Good")
        self.assertEqual(exit_code, audit.EXIT_OK)

    def test_acceptable_warn(self) -> None:
        result = audit.AuditResult(path=Path("dummy"), total_lines=250)
        name, _reason, exit_code = result.verdict
        self.assertEqual(name, "Acceptable")
        self.assertEqual(exit_code, audit.EXIT_WARN)

    def test_bloated(self) -> None:
        result = audit.AuditResult(path=Path("dummy"), total_lines=400)
        name, _reason, exit_code = result.verdict
        self.assertEqual(name, "Bloated")
        self.assertEqual(exit_code, audit.EXIT_BLOATED)


# ──────────────────────────────────────────────────────────────────────
# Full audit integration
# ──────────────────────────────────────────────────────────────────────


class TestAuditIntegration(unittest.TestCase):
    def test_minimal_claude_md(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            claude_md = Path(tmp) / "CLAUDE.md"
            claude_md.write_text(
                "# Project\n\n## Overview\n\nShort overview.\n",
                encoding="utf-8",
            )
            result = audit.audit(claude_md)
            # splitlines() yields 5 lines before the trailing newline
            self.assertEqual(result.total_lines, 5)
            self.assertEqual(len(result.sections), 1)
            self.assertEqual(result.sections[0].title, "Overview")
            # No config file → deprecated_hits is an empty list
            self.assertEqual(result.deprecated_hits, [])

    def test_render_report_no_crash(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            claude_md = Path(tmp) / "CLAUDE.md"
            claude_md.write_text("# Project\n\n## S\nx\n", encoding="utf-8")
            result = audit.audit(claude_md)
            report_text, exit_code = audit.render_report(result)
            self.assertIn("CLAUDE.md Audit", report_text)
            self.assertIn("Measure", report_text)
            self.assertEqual(exit_code, audit.EXIT_OK)


# ──────────────────────────────────────────────────────────────────────
# Semantic conflict detection
# ──────────────────────────────────────────────────────────────────────


class TestSemanticConflicts(unittest.TestCase):
    """Unit tests for _detect_semantic_conflicts.

    These tests load the real conflict-patterns.json from the script directory,
    so `_SCRIPT_CONFIG_DIR` is left untouched.
    """

    def test_basic_conflict_detected(self) -> None:
        """Reproduction: forbid/allow co-exist in different sections of one doc → detect."""
        text = (
            "# Project\n\n"
            "## Command Priority Policy\n\n"
            "Direct calls to MCP tools like `task_create` are forbidden. Always go through Command.\n\n"
            "## MCP Tool Priority Rule\n\n"
            "Please use MCP tools such as `task_create`.\n"
        )
        sections = audit.parse_sections(text)
        project_root = Path(".")  # Script-relative config loads first, so root is irrelevant here
        hits = audit._detect_semantic_conflicts(text, sections, project_root)
        self.assertGreaterEqual(len(hits), 1, "Reproduction scenario must yield at least one hit")
        self.assertEqual(hits[0].token, "task_create")

    def test_normal_document_no_false_positive(self) -> None:
        """A normal document yields zero conflicts."""
        text = (
            "# Project\n\n"
            "## How to Use\n\n"
            "Please use the `task_create` MCP tool.\n"
        )
        sections = audit.parse_sections(text)
        hits = audit._detect_semantic_conflicts(text, sections, Path("."))
        self.assertEqual(len(hits), 0)

    def test_time_context_excluded(self) -> None:
        """Sentences with past/temporal markers must not produce false positives."""
        text = (
            "# Project\n\n"
            "## History\n\n"
            "Previously direct calls to `old_tool` were forbidden, but now please use `old_tool`.\n"
        )
        sections = audit.parse_sections(text)
        hits = audit._detect_semantic_conflicts(text, sections, Path("."))
        # The "History" section is in exclude_section_patterns and the line carries the "Previously" time marker
        self.assertEqual(len(hits), 0)

    def test_excluded_section_not_scanned(self) -> None:
        """Sections matching exclude_section_patterns are skipped during scanning."""
        text = (
            "# Project\n\n"
            "## Migration Guide\n\n"
            "Do not use `old_api`. Please use `new_api`.\n"
            "Before: call `old_api`\n"
        )
        sections = audit.parse_sections(text)
        hits = audit._detect_semantic_conflicts(text, sections, Path("."))
        self.assertEqual(len(hits), 0, "Migration sections are excluded from scanning")

    def test_section_is_excluded_helper(self) -> None:
        patterns = ["migration", "retro"]
        self.assertTrue(audit._section_is_excluded("Migration Guide", patterns))
        self.assertTrue(audit._section_is_excluded("Retrospective retro", patterns))
        self.assertFalse(audit._section_is_excluded("General Section", patterns))

    def test_has_time_context_helper(self) -> None:
        markers = ["previously", "in\\s*the\\s*past"]
        self.assertTrue(audit._has_time_context("previously this was so", markers))
        self.assertTrue(audit._has_time_context("previously used", markers))
        self.assertFalse(audit._has_time_context("currently this is so", markers))


if __name__ == "__main__":
    unittest.main()
