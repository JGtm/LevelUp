"""Chargement dynamique des modules de migration."""

from __future__ import annotations

import importlib
import pkgutil


def load_all_migration_steps() -> None:
    """Importe tous les modules de ``src.data.migration.steps`` une seule fois."""
    if load_all_migration_steps._loaded:
        return

    import src.data.migration.steps as steps_pkg

    for module_info in pkgutil.iter_modules(steps_pkg.__path__):
        if module_info.name.startswith("_"):
            continue
        importlib.import_module(f"{steps_pkg.__name__}.{module_info.name}")
    load_all_migration_steps._loaded = True


load_all_migration_steps._loaded = False  # type: ignore[attr-defined]
