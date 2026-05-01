#!/usr/bin/env python3
"""
CLAUDE.md audit — measure + compare (Anthropic targets) + suggest migration.

Read-only tool. Produces an ASCII report with:
1. measurements (total lines, sections, max section, @imports, .claude/rules/)
2. comparison against Anthropic targets (≤60 / ≤200 / ≤300)
3. migration suggestions based on design principles P1~P7
4. task templates for proposed relocations

Never modifies files. Exit code reflects severity so this can be wired
into pre-commit hooks if desired.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable

# Import shared utility from skills/_shared/config_loader.
# Adds the grandparent directory (skills/) to sys.path so that this script
# works reliably even when auditing CLAUDE.md files from external paths,
# then imports _shared directly as a module.
_SKILLS_DIR = Path(__file__).resolve().parent.parent.parent
if str(_SKILLS_DIR) not in sys.path:
    sys.path.insert(0, str(_SKILLS_DIR))
from _shared.config_loader import load_first_existing_json  # noqa: E402

# ──────────────────────────────────────────────────────────────────────
# Constants (no magic numbers — all extracted as named constants)
# ──────────────────────────────────────────────────────────────────────

# Anthropic-aligned thresholds (research §3 + official memory documentation)
THRESHOLD_EXCELLENT = 60    # Anthropic "minimal CLAUDE.md" level
THRESHOLD_GOOD = 200         # Official target ("target under 200 lines per CLAUDE.md")
THRESHOLD_ACCEPTABLE = 300   # Soft ceiling — exceeding it warns of reduced adherence

# Section-level heuristics
SECTION_LARGE_LINES = 40     # At or above this, consider as a relocation candidate
SECTION_HUGE_LINES = 80      # At or above this, mark as a top-priority relocation

# Exit codes
EXIT_OK = 0
EXIT_WARN = 1           # Acceptable range (200 < lines <= 300)
EXIT_BLOATED = 2        # > 300
EXIT_FILE_NOT_FOUND = 3

# Section classification — mapping based on principles P1~P7
SECTION_PATTERNS = [
    # (regex, principle, suggested_target, paths_pattern, severity)
    (
        re.compile(r"skill\s+development", re.IGNORECASE),
        "P4",
        ".claude/rules/skill-development.md + docs/04-guides/skill-development-standard.md",
        "skills/**/*, **/SKILL.md",
        "high",
    ),
    (
        re.compile(r"mcp\s+tool\s+development", re.IGNORECASE),
        "P4",
        ".claude/rules/mcp-tool-development.md + docs/04-guides/mcp-tool-development-standard.md",
        "mcp-server/**/*.go",
        "high",
    ),
    (
        re.compile(r"command\s+development", re.IGNORECASE),
        "P4",
        ".claude/rules/command-development.md + docs/04-guides/command-development-standard.md",
        "commands/**/*.md",
        "high",
    ),
    (
        re.compile(r"environment\s+variables?", re.IGNORECASE),
        "P3",
        "docs/04-guides/configuration.md §4~§5 (keep pointer only)",
        "—",
        "medium",
    ),
    (
        re.compile(r"skill\s+trigger", re.IGNORECASE),
        "P5",
        "docs/03-design/skill-trigger-index.md (auto-generated)",
        "—",
        "medium",
    ),
    (
        re.compile(r"roadmap|completion\s*history", re.IGNORECASE),
        "P6",
        "docs/00-project/roadmap.md",
        "—",
        "low",
    ),
]

# Config paths are searched in two ways:
# 1. Relative to the script directory (../config/*.json) — works for external file audits
# 2. Relative to project_root (skills/claude-md-audit/config/*.json) — legacy compatibility
_SCRIPT_CONFIG_DIR = Path(__file__).resolve().parent.parent / "config"
DEPRECATED_TOKENS_CONFIG = "skills/claude-md-audit/config/deprecated-tokens.json"
CONFLICT_PATTERNS_CONFIG = "skills/claude-md-audit/config/conflict-patterns.json"

SECTION_HEADER_RE = re.compile(r"^(#{1,6})\s+(.*)$")
IMPORT_RE = re.compile(r"(?:^|[^\w`])@([A-Za-z0-9_./~-]+)")


@dataclass
class DeprecatedHit:
    """A single deprecated-token detection hit."""
    pattern: str
    file: str
    line_no: int
    line_text: str
    removed_in: str
    replacement: str
    severity: str


@dataclass
class ConflictHit:
    """A single semantic-conflict detection hit.

    Represents the case where contradictory verbs (e.g. forbid vs allow)
    target the same token within the same or adjacent sections of one document.
    """
    token: str
    forbid_section: str
    forbid_line_no: int
    forbid_snippet: str
    allow_section: str
    allow_line_no: int
    allow_snippet: str
    pair_id: str


@dataclass
class Section:
    title: str
    level: int
    start_line: int
    end_line: int = 0
    principle: str = ""
    suggested_target: str = ""
    paths_pattern: str = ""
    severity: str = ""

    @property
    def line_count(self) -> int:
        return self.end_line - self.start_line


@dataclass
class AuditResult:
    path: Path
    total_lines: int
    sections: list[Section] = field(default_factory=list)
    import_count: int = 0
    claude_rules_files: list[Path] = field(default_factory=list)
    deprecated_hits: list[DeprecatedHit] = field(default_factory=list)
    conflict_hits: list[ConflictHit] = field(default_factory=list)

    @property
    def top_section(self) -> Section | None:
        l2_sections = [s for s in self.sections if s.level == 2]
        if not l2_sections:
            return None
        return max(l2_sections, key=lambda s: s.line_count)

    @property
    def verdict(self) -> tuple[str, str, int]:
        lines = self.total_lines
        if lines <= THRESHOLD_EXCELLENT:
            return ("Excellent", f"≤{THRESHOLD_EXCELLENT} lines, Anthropic minimal level", EXIT_OK)
        if lines <= THRESHOLD_GOOD:
            pct = int(lines * 100 / THRESHOLD_GOOD)
            return ("Good", f"{pct}% of the ≤{THRESHOLD_GOOD}-line target", EXIT_OK)
        if lines <= THRESHOLD_ACCEPTABLE:
            pct = int(lines * 100 / THRESHOLD_GOOD)
            return ("Acceptable", f"{pct}% of {THRESHOLD_GOOD} lines — action recommended", EXIT_WARN)
        pct = int(lines * 100 / THRESHOLD_GOOD)
        return ("Bloated", f"{pct}% of {THRESHOLD_GOOD} lines — immediate action required", EXIT_BLOATED)


# ──────────────────────────────────────────────────────────────────────
# Parsing
# ──────────────────────────────────────────────────────────────────────


def classify_section(title: str) -> tuple[str, str, str, str]:
    for pattern, principle, target, paths, severity in SECTION_PATTERNS:
        if pattern.search(title):
            return principle, target, paths, severity
    return "", "", "", ""


def parse_sections(text: str) -> list[Section]:
    """Extract level-2 sections from markdown, ignoring fenced code blocks."""
    sections: list[Section] = []
    in_code_block = False
    lines = text.splitlines()

    for i, line in enumerate(lines, start=1):
        if line.lstrip().startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block:
            continue

        match = SECTION_HEADER_RE.match(line)
        if not match or len(match.group(1)) != 2:
            continue

        title = match.group(2).strip()
        if sections:
            sections[-1].end_line = i
        principle, target, paths, severity = classify_section(title)
        sections.append(
            Section(
                title=title,
                level=2,
                start_line=i,
                principle=principle,
                suggested_target=target,
                paths_pattern=paths,
                severity=severity,
            )
        )

    if sections:
        sections[-1].end_line = len(lines) + 1

    return sections


def count_imports(text: str) -> int:
    in_code_block = False
    count = 0
    for line in text.splitlines():
        if line.lstrip().startswith("```"):
            in_code_block = not in_code_block
            continue
        if in_code_block:
            continue
        count += len(IMPORT_RE.findall(line))
    return count


def discover_claude_rules(project_root: Path) -> list[Path]:
    rules_dir = project_root / ".claude" / "rules"
    if not rules_dir.is_dir():
        return []
    return sorted(rules_dir.glob("*.md"))


# ──────────────────────────────────────────────────────────────────────
# Deprecated token detection
# ──────────────────────────────────────────────────────────────────────


def _is_excluded(rel_path: str, exclude_paths: list[str]) -> bool:
    """Check whether a path matches the exclude list. Supports both directories (/) and filenames.

    Note: a directory pattern like "works/" must match only sub-paths shaped exactly
    as "works/...". It must not produce false positives on same-prefix files such as
    "worksheet.md" (this fix originated from a bug discovered during a Task).
    """
    for excl in exclude_paths:
        excl_clean = excl.rstrip("/")
        if excl.endswith("/"):
            # Directory-boundary match — exact equality with excl_clean or starts with excl_clean + "/"
            if rel_path == excl_clean or rel_path.startswith(excl_clean + "/"):
                return True
        else:
            # Exact filename match, or subpath match when treated as a directory
            if rel_path == excl_clean or rel_path.startswith(excl_clean + "/"):
                return True
    return False


def _load_deprecated_config(project_root: Path) -> dict | None:
    """Load deprecated-tokens.json, trying script directory first, then project_root.

    This function is responsible for building the candidate paths (it preserves the
    monkey-patchable `_SCRIPT_CONFIG_DIR`); the actual file discovery and parsing is
    delegated to skills/_shared/config_loader.load_first_existing_json.
    """
    candidates = [
        _SCRIPT_CONFIG_DIR / "deprecated-tokens.json",
        project_root / DEPRECATED_TOKENS_CONFIG,
    ]
    return load_first_existing_json(candidates)


def _detect_deprecated_tokens(project_root: Path) -> list[DeprecatedHit]:
    """Grep project documents for deprecated tokens.

    Automatically excludes paths under archive, completed, knowledge, and reports.
    """
    config = _load_deprecated_config(project_root)
    if not config or "tokens" not in config:
        return []

    exclude_paths = config.get("exclude_paths", [])
    scan_globs = ["*.md", "*.yaml", "*.yml", "CLAUDE.md"]
    hits: list[DeprecatedHit] = []

    for token_entry in config["tokens"]:
        pattern = token_entry["pattern"]
        removed_in = token_entry.get("removed_in", "")
        replacement = token_entry.get("replacement", "")
        severity = token_entry.get("severity", "medium")

        # Detect with rg (ripgrep), falling back to grep
        rg_args = [
            "rg", "--no-heading", "--line-number", "--color=never",
            "--glob=*.md", "--glob=*.yaml", "--glob=*.yml",
            pattern, str(project_root),
        ]
        for excl in exclude_paths:
            rg_args.insert(-2, f"--glob=!{excl}*")

        try:
            result = subprocess.run(
                rg_args, capture_output=True, text=True, timeout=10,
            )
        except (FileNotFoundError, subprocess.TimeoutExpired):
            # Fall back to grep if rg is unavailable
            try:
                grep_args = [
                    "grep", "-rn", "--include=*.md", "--include=*.yaml",
                    pattern, str(project_root),
                ]
                result = subprocess.run(
                    grep_args, capture_output=True, text=True, timeout=10,
                )
            except (FileNotFoundError, subprocess.TimeoutExpired):
                continue

        if result.returncode != 0:
            continue

        for line in result.stdout.strip().splitlines():
            # format: /path/to/file:123:line content
            parts = line.split(":", 2)
            if len(parts) < 3:
                continue
            file_path = parts[0]
            try:
                line_no = int(parts[1])
            except ValueError:
                continue
            line_text = parts[2].strip()

            # Re-check exclude paths (double-check for the grep fallback path)
            rel_path = str(Path(file_path).relative_to(project_root))
            if _is_excluded(rel_path, exclude_paths):
                continue

            hits.append(DeprecatedHit(
                pattern=pattern,
                file=rel_path,
                line_no=line_no,
                line_text=line_text[:80],
                removed_in=removed_in,
                replacement=replacement,
                severity=severity,
            ))

    return hits


# ──────────────────────────────────────────────────────────────────────
# Semantic conflict detection
# ──────────────────────────────────────────────────────────────────────


def _load_conflict_config(project_root: Path) -> dict | None:
    """Load conflict-patterns.json, trying script directory first, then project_root.

    Delegates discovery and parsing to config_loader.load_first_existing_json.
    """
    candidates = [
        _SCRIPT_CONFIG_DIR / "conflict-patterns.json",
        project_root / CONFLICT_PATTERNS_CONFIG,
    ]
    return load_first_existing_json(candidates)


def _section_is_excluded(title: str, patterns: list[str]) -> bool:
    """Check whether the section title matches any of the exclude patterns."""
    for pattern in patterns:
        if re.search(pattern, title, re.IGNORECASE):
            return True
    return False


def _has_time_context(line: str, markers: list[str]) -> bool:
    """Check whether the line contains a temporal-context marker (e.g. "previously", "in the past")."""
    for marker in markers:
        if re.search(marker, line, re.IGNORECASE):
            return True
    return False


def _detect_semantic_conflicts(
    text: str, sections: list[Section], project_root: Path,
) -> list[ConflictHit]:
    """Detect contradictory instructions targeting the same token within a document.

    Algorithm:
    1. Load conflict-patterns.json (return empty list if missing).
    2. Skip a section whose title matches any exclude pattern.
    3. For each line in a section, if a forbid- or allow-verb pattern matches:
       - Extract backtick-quoted tokens from the line.
       - Record (section_idx, line_no, snippet).
    4. Per token, collect the forbid and allow sets.
    5. When the same token appears on both sides, emit a conflict.
    6. Report only pairs within `min_section_distance`.
    """
    config = _load_conflict_config(project_root)
    if not config or "verb_pairs" not in config:
        return []

    token_regex = re.compile(config.get("token_regex", r"`([A-Za-z0-9_./-]+)`"))
    time_markers = config.get("time_context_markers", [])
    exclude_sections = config.get("exclude_section_patterns", [])
    min_distance = int(config.get("min_section_distance", 0))

    lines = text.splitlines()
    hits: list[ConflictHit] = []

    for pair in config["verb_pairs"]:
        pair_id = pair.get("id", "unknown")
        forbid_patterns = [re.compile(p, re.IGNORECASE) for p in pair.get("forbid", [])]
        allow_patterns = [re.compile(p, re.IGNORECASE) for p in pair.get("allow", [])]

        # Per-token collection: {token: [(section_idx, line_no, snippet, verb_type)]}
        token_mentions: dict[str, list[tuple[int, int, str, str]]] = {}

        for section_idx, section in enumerate(sections):
            if _section_is_excluded(section.title, exclude_sections):
                continue

            start = section.start_line
            end = section.end_line if section.end_line > 0 else len(lines) + 1
            # start/end are 1-based; lines is 0-based
            for line_no in range(start, min(end, len(lines) + 1)):
                line_text = lines[line_no - 1] if line_no - 1 < len(lines) else ""
                if not line_text.strip():
                    continue
                if _has_time_context(line_text, time_markers):
                    continue

                # Extract backtick tokens from the line
                tokens_in_line = token_regex.findall(line_text)
                if not tokens_in_line:
                    continue

                # Check forbid/allow verb matches
                is_forbid = any(p.search(line_text) for p in forbid_patterns)
                is_allow = any(p.search(line_text) for p in allow_patterns)

                if not (is_forbid or is_allow):
                    continue

                verb_type = "forbid" if is_forbid else "allow"
                snippet = line_text.strip()[:80]

                for token in tokens_in_line:
                    token_mentions.setdefault(token, []).append(
                        (section_idx, line_no, snippet, verb_type)
                    )

        # Per token, check whether both forbid and allow sides have entries
        for token, mentions in token_mentions.items():
            forbid_list = [m for m in mentions if m[3] == "forbid"]
            allow_list = [m for m in mentions if m[3] == "allow"]
            if not forbid_list or not allow_list:
                continue

            # Report only the closest pair (avoid duplicate flooding)
            f = forbid_list[0]
            a = allow_list[0]
            section_distance = abs(f[0] - a[0])
            # min_distance < 0 means unlimited distance (always detect within the same document)
            if min_distance >= 0 and section_distance > min_distance:
                continue

            f_section = sections[f[0]].title if f[0] < len(sections) else "?"
            a_section = sections[a[0]].title if a[0] < len(sections) else "?"

            hits.append(ConflictHit(
                token=token,
                forbid_section=f_section,
                forbid_line_no=f[1],
                forbid_snippet=f[2],
                allow_section=a_section,
                allow_line_no=a[1],
                allow_snippet=a[2],
                pair_id=pair_id,
            ))

    return hits


# ──────────────────────────────────────────────────────────────────────
# Audit orchestration
# ──────────────────────────────────────────────────────────────────────


def audit(path: Path) -> AuditResult:
    text = path.read_text(encoding="utf-8")
    total_lines = len(text.splitlines())
    sections = parse_sections(text)
    import_count = count_imports(text)
    project_root = path.parent
    claude_rules = discover_claude_rules(project_root)
    deprecated_hits = _detect_deprecated_tokens(project_root)
    conflict_hits = _detect_semantic_conflicts(text, sections, project_root)

    return AuditResult(
        path=path,
        total_lines=total_lines,
        sections=sections,
        import_count=import_count,
        claude_rules_files=claude_rules,
        deprecated_hits=deprecated_hits,
        conflict_hits=conflict_hits,
    )


# ──────────────────────────────────────────────────────────────────────
# Rendering
# ──────────────────────────────────────────────────────────────────────


def _hbar() -> str:
    return "━" * 61


def _iter_migration_candidates(result: AuditResult) -> Iterable[Section]:
    for s in result.sections:
        if not s.principle:
            continue
        if s.line_count < SECTION_LARGE_LINES and s.severity != "high":
            continue
        yield s


def render_report(result: AuditResult) -> tuple[str, int]:
    lines: list[str] = []
    verdict_name, verdict_reason, exit_code = result.verdict

    rel = result.path.as_posix()
    lines.append(_hbar())
    lines.append(f"  CLAUDE.md Audit — {rel}")
    lines.append(_hbar())
    lines.append("")

    # Step 1 — Measure
    lines.append("  Measure")
    lines.append("  " + "─" * 7)
    lines.append(f"  Total:         {result.total_lines} lines")
    lines.append(f"  Sections (L2): {sum(1 for s in result.sections if s.level == 2)}")
    top = result.top_section
    if top:
        pct = int(top.line_count * 100 / max(result.total_lines, 1))
        lines.append(f'  Largest:       "{top.title}" {top.line_count} lines ({pct}%)')
    lines.append(f"  @import:       {result.import_count}")
    lines.append(f"  .claude/rules: {len(result.claude_rules_files)} files")
    for rf in result.claude_rules_files:
        lines.append(f"                 - {rf.name}")
    lines.append("")

    # Step 2 — Compare
    lines.append("  Compare")
    lines.append("  " + "─" * 7)
    symbol = "✅" if exit_code == EXIT_OK else ("⚠️" if exit_code == EXIT_WARN else "❌")
    lines.append(f"  {symbol} {verdict_name} — {verdict_reason}")
    lines.append(
        f"  Thresholds: Excellent ≤{THRESHOLD_EXCELLENT} / Good ≤{THRESHOLD_GOOD} / "
        f"Acceptable ≤{THRESHOLD_ACCEPTABLE}"
    )
    lines.append("")

    # Step 3 & 4 — Suggest + Migration matrix
    candidates = list(_iter_migration_candidates(result))
    lines.append("  Migration matrix")
    lines.append("  " + "─" * 16)
    if not candidates:
        lines.append("  No additional relocation candidates (current state is healthy)")
    else:
        lines.append(
            "  | Section | Lines | Principle | Target | Paths | Est. reduction |"
        )
        lines.append(
            "  |---------|-------|-----------|--------|-------|----------------|"
        )
        for s in candidates:
            expected_reduction = max(s.line_count - 4, 0)  # Assumes a 4-line pointer remains
            lines.append(
                f"  | {s.title[:20]} | {s.line_count} | {s.principle} | "
                f"{s.suggested_target[:40]}… | {s.paths_pattern[:20]} | "
                f"-{expected_reduction} |"
            )
    lines.append("")

    # Step 5 — Deprecated tokens
    lines.append("  Deprecated token detection")
    lines.append("  " + "─" * 26)
    if not result.deprecated_hits:
        lines.append("  ✅ 0 deprecated tokens — clean")
    else:
        lines.append(f"  ❌ {len(result.deprecated_hits)} deprecated tokens detected")
        lines.append("")
        # Group by pattern
        by_pattern: dict[str, list[DeprecatedHit]] = {}
        for hit in result.deprecated_hits:
            by_pattern.setdefault(hit.pattern, []).append(hit)
        for pattern, group in by_pattern.items():
            first = group[0]
            lines.append(f"  [{first.severity.upper()}] `{pattern}`")
            lines.append(f"    Removed in:  {first.removed_in}")
            lines.append(f"    Replacement: {first.replacement}")
            lines.append(f"    {len(group)} occurrences:")
            for hit in group:
                lines.append(f"      {hit.file}:{hit.line_no}")
            lines.append("")
        if exit_code == EXIT_OK:
            exit_code = EXIT_WARN  # Warn when deprecated tokens are present
    lines.append("")

    # Step 5b — Semantic conflicts
    lines.append("  Contradictory-instruction detection")
    lines.append("  " + "─" * 35)
    if not result.conflict_hits:
        lines.append("  ✅ 0 semantic conflicts — clean")
    else:
        lines.append(f"  ❌ {len(result.conflict_hits)} semantic conflicts detected")
        lines.append("")
        for hit in result.conflict_hits:
            lines.append(f"  [{hit.pair_id}] token: `{hit.token}`")
            lines.append(f"    Forbid — '{hit.forbid_section}' (L{hit.forbid_line_no}):")
            lines.append(f"      {hit.forbid_snippet}")
            lines.append(f"    Allow  — '{hit.allow_section}' (L{hit.allow_line_no}):")
            lines.append(f"      {hit.allow_snippet}")
            lines.append("")
        if exit_code == EXIT_OK:
            exit_code = EXIT_WARN
    lines.append("")

    # Step 6 — Execution guide
    lines.append("  Execution guide")
    lines.append("  " + "─" * 15)
    if candidates:
        lines.append("  Create a separate Task for each relocation candidate via hstl-oss task create:")
        for i, s in enumerate(candidates, start=1):
            title_short = s.title.strip()
            lines.append(
                f"  {i}. hstl-oss task create --title {title_short!r} --type refactor --estimate S"
            )
        lines.append("")
        lines.append("  Verification:")
        lines.append("  - Check load state with /memory")
        lines.append("  - Debug load timing with the InstructionsLoaded hook")
    else:
        lines.append("  No action required. Maintain drift monitoring:")
        lines.append("  - python3 scripts/generate-skill-triggers.py")
        lines.append("  - Verify .claude/rules/ ↔ docs/04-guides/ stay in sync")
    lines.append("")
    lines.append(_hbar())

    return "\n".join(lines), exit_code


# ──────────────────────────────────────────────────────────────────────
# CLI
# ──────────────────────────────────────────────────────────────────────


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Audit a CLAUDE.md file against Anthropic memory targets."
    )
    parser.add_argument(
        "path",
        nargs="?",
        default="CLAUDE.md",
        help="Path to CLAUDE.md (default: CLAUDE.md in cwd)",
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Exit non-zero even for 'Acceptable' verdicts (CI mode)",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv if argv is not None else sys.argv[1:])
    path = Path(args.path).resolve()

    if not path.is_file():
        print(f"ERROR: file not found: {path}", file=sys.stderr)
        return EXIT_FILE_NOT_FOUND

    result = audit(path)
    report, exit_code = render_report(result)
    print(report)

    # In strict mode, even EXIT_OK with > 140 lines would not trigger a fail
    # (we never fail on Good verdict) — strict only upgrades WARN to fail.
    if args.strict and exit_code == EXIT_WARN:
        return EXIT_BLOATED
    return exit_code


if __name__ == "__main__":
    sys.exit(main())
