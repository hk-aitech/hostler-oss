"""skills/_shared/config_loader — Shared utility for loading skill config files.

Provides a "script directory first → project_root second" dual-search pattern
for finding a skill's config file even when the skill operates against an
external path (e.g. /tmp/foo.md). The pattern was first introduced as an
emergency fix in claude-md-audit and has been generalized here.

Design rationale:
    - KB `docs/07-knowledge/architecture/config-loading.md` A01
    - A skill's config ships with the skill itself, so a `__file__`-based path
      is the most stable. The project_root path is for legacy compatibility
      and user overrides.

Example:

    from pathlib import Path
    from skills._shared.config_loader import load_json_config

    cfg = load_json_config(
        config_name="my-config.json",
        script_file=__file__,  # __file__ of the calling script
        project_root=Path.cwd(),
        legacy_relative_path="skills/my-skill/config/my-config.json",
    )
    if cfg is None:
        # File not found or parse failure
        ...
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any


def _resolve_script_config_dir(script_file: str) -> Path:
    """Compute the ../config directory path relative to script_file.

    Example: skills/foo/scripts/foo.py → skills/foo/config/
    """
    return Path(script_file).resolve().parent.parent / "config"


def load_json_config(
    config_name: str,
    script_file: str,
    project_root: Path,
    legacy_relative_path: str | None = None,
) -> dict[str, Any] | None:
    """Load a JSON config file via dual-search.

    Search order:
        1. Script directory first: `<script_dir>/../config/<config_name>`
        2. project_root next (when legacy_relative_path is given): `project_root / legacy_relative_path`

    Args:
        config_name: config file basename (e.g. "my-config.json")
        script_file: __file__ path of the calling script
        project_root: project root (legacy fallback)
        legacy_relative_path: relative path under project_root (omit to skip fallback)

    Returns:
        Parsed dict, or None when the file is missing or fails to parse.
    """
    script_config_dir = _resolve_script_config_dir(script_file)
    candidates: list[Path] = [script_config_dir / config_name]
    if legacy_relative_path:
        candidates.append(project_root / legacy_relative_path)

    for config_path in candidates:
        if not config_path.is_file():
            continue
        try:
            return json.loads(config_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
    return None


def list_config_candidates(
    config_name: str,
    script_file: str,
    project_root: Path,
    legacy_relative_path: str | None = None,
) -> list[Path]:
    """Return the list of candidate paths considered (for debugging).

    Returns the candidate paths regardless of whether the files actually exist,
    which is useful when surfacing "where I looked" to the user.
    """
    script_config_dir = _resolve_script_config_dir(script_file)
    candidates = [script_config_dir / config_name]
    if legacy_relative_path:
        candidates.append(project_root / legacy_relative_path)
    return candidates


def load_first_existing_json(candidates: list[Path]) -> dict[str, Any] | None:
    """Iterate through the candidate paths and return the first valid JSON found.

    Use when the caller wants to construct the candidate list directly
    (e.g. keeping a monkey-patchable constant at the module level).
    Lower-level API than load_json_config.

    Args:
        candidates: ordered list of paths to try.

    Returns:
        Parsed dict, or None if all candidates fail.
    """
    for config_path in candidates:
        if not config_path.is_file():
            continue
        try:
            return json.loads(config_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
    return None
