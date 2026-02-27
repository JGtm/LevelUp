"""Tests pour le refactoring i18n (déduplication, fallback viz_t → common).

Vérifie :
- viz_t() résout les clés common.STRINGS en fallback
- Aucun doublon interne dans chaque module STRINGS
- Pas de collision silencieuse avec valeurs différentes entre modules
- viz_t() supporte **kwargs (format strings)
"""

from __future__ import annotations

import pytest


class TestVizTFallbackToCommon:
    """viz_t() doit résoudre les clés de common.STRINGS quand elles ne sont pas dans viz."""

    def test_viz_t_resolves_common_key_fr(self) -> None:
        """viz_t() résout col_kills depuis common.STRINGS."""
        from src.ui.i18n.viz import viz_t

        result = viz_t("col_kills", "fr")
        assert result == "Frags"

    def test_viz_t_resolves_common_key_en(self) -> None:
        from src.ui.i18n.viz import viz_t

        result = viz_t("col_kills", "en")
        assert result == "Kills"

    def test_viz_t_returns_bracket_for_unknown(self) -> None:
        """Clé totalement inconnue → [key]."""
        from src.ui.i18n.viz import viz_t

        result = viz_t("__nonexistent_key_xyz__", "fr")
        assert result == "[__nonexistent_key_xyz__]"

    def test_viz_t_prefers_viz_over_common(self) -> None:
        """Si la clé existe dans viz ET common, viz gagne."""
        from src.ui.i18n.common import STRINGS as COMMON
        from src.ui.i18n.viz import STRINGS as VIZ
        from src.ui.i18n.viz import viz_t

        # Trouver une clé présente dans les deux
        shared_keys = set(VIZ.keys()) & set(COMMON.keys())
        if not shared_keys:
            pytest.skip("Aucune clé partagée entre viz et common")

        key = next(iter(shared_keys))
        result = viz_t(key, "fr")
        assert result == VIZ[key]["fr"]

    def test_viz_t_kwargs_support(self) -> None:
        """viz_t() doit supporter str.format(**kwargs)."""
        from src.ui.i18n.viz import viz_t

        # label_target contient {value}
        result = viz_t("label_target", "fr", value="1.5")
        assert "1.5" in result

    def test_viz_t_common_outcomes(self) -> None:
        """Les clés outcome_* de common sont accessibles via viz_t."""
        from src.ui.i18n.viz import viz_t

        assert viz_t("outcome_win", "fr") == "Victoire"
        assert viz_t("outcome_loss", "en") == "Loss"
        assert viz_t("outcome_draw", "fr") == "Égalité"

    def test_viz_t_common_no_data(self) -> None:
        """Les messages d'erreur de common sont accessibles via viz_t."""
        from src.ui.i18n.viz import viz_t

        result = viz_t("no_data", "fr")
        assert result == "Aucune donnée disponible."


class TestNoInternalDuplicates:
    """Chaque module STRINGS ne doit pas contenir de clé en double."""

    @pytest.mark.parametrize(
        "module_name",
        ["common", "pages", "widgets", "viz", "cli"],
    )
    def test_no_duplicate_keys_in_module(self, module_name: str) -> None:
        """Détecte les clés définies 2+ fois dans un même fichier source.

        Note : Python dict garde la dernière valeur, donc on parse le source
        pour détecter les doublons.
        """
        import ast
        from pathlib import Path

        # Résoudre le chemin depuis la racine du projet
        test_dir = Path(__file__).resolve().parent
        project_root = test_dir.parent
        module_path = project_root / "src" / "ui" / "i18n" / f"{module_name}.py"
        if not module_path.exists():
            pytest.skip(f"Module {module_name}.py introuvable à {module_path}")

        source = module_path.read_text(encoding="utf-8")
        tree = ast.parse(source)

        # Trouver l'assignation STRINGS = {...} ou STRINGS: ... = {...}
        for node in ast.walk(tree):
            target_name: str | None = None
            value_node = None

            if isinstance(node, ast.Assign):
                for tgt in node.targets:
                    if isinstance(tgt, ast.Name) and tgt.id == "STRINGS":
                        target_name = tgt.id
                        value_node = node.value
                        break
            elif (
                isinstance(node, ast.AnnAssign)
                and isinstance(node.target, ast.Name)
                and node.target.id == "STRINGS"
            ):
                target_name = node.target.id
                value_node = node.value

            if target_name and isinstance(value_node, ast.Dict):
                keys: list[str] = []
                for k in value_node.keys:
                    if isinstance(k, ast.Constant) and isinstance(k.value, str):
                        keys.append(k.value)
                seen: dict[str, int] = {}
                duplicates: list[str] = []
                for k in keys:
                    if k in seen:
                        duplicates.append(k)
                    seen[k] = seen.get(k, 0) + 1
                assert not duplicates, f"Clés en double dans {module_name}.py: {duplicates}"
                return

        pytest.skip(f"STRINGS non trouvé dans {module_name}.py")


class TestNoSilentCollisions:
    """Pas de collision entre modules avec des valeurs DIFFÉRENTES (sans warning)."""

    def test_no_divergent_collisions(self) -> None:
        """Détecte les clés avec le même nom mais des valeurs différentes entre modules."""
        from src.ui.i18n import cli, common, pages, viz, widgets

        all_modules = [
            ("common", common.STRINGS),
            ("pages", pages.STRINGS),
            ("widgets", widgets.STRINGS),
            ("viz", viz.STRINGS),
            ("cli", cli.STRINGS),
        ]

        collisions: list[str] = []
        seen: dict[str, tuple[str, dict[str, str]]] = {}

        for mod_name, strings in all_modules:
            for key, translations in strings.items():
                if key in seen:
                    prev_mod, prev_val = seen[key]
                    if prev_val != translations:
                        collisions.append(
                            f"  '{key}': {prev_mod}={prev_val} vs {mod_name}={translations}"
                        )
                else:
                    seen[key] = (mod_name, translations)

        assert not collisions, "Collisions de clés avec valeurs différentes :\n" + "\n".join(
            collisions
        )


class TestRegistryIntegrity:
    """Le registre global t() fonctionne correctement après refactoring."""

    def test_t_resolves_common_key(self) -> None:
        from src.ui.i18n import reset_registry, t

        reset_registry()
        assert t("col_kills", lang="fr") == "Frags"
        assert t("col_kills", lang="en") == "Kills"

    def test_t_resolves_pages_key(self) -> None:
        from src.ui.i18n import reset_registry, t

        reset_registry()
        result = t("radar_desc_combat", lang="fr")
        assert "Combat" in result

    def test_t_resolves_widgets_key(self) -> None:
        from src.ui.i18n import reset_registry, t

        reset_registry()
        assert t("btn_sync", lang="fr") == "Actualiser"

    def test_t_resolves_viz_key(self) -> None:
        from src.ui.i18n import reset_registry, t

        reset_registry()
        result = t("trace_kills", lang="fr")
        assert result == "Frags"

    def test_t_bracket_for_unknown(self) -> None:
        from src.ui.i18n import reset_registry, t

        reset_registry()
        assert t("__totally_fake__") == "[__totally_fake__]"
