"""Tests — ensure_weapon_labels (migration metadata.duckdb) et ui/i18n/weapons.py.

Couvre :
- ensure_weapon_labels : création, contenu, idempotence
- Step auto-enregistré dans le registre de migrations
- get_weapon_label (ui/i18n/weapons.py) : délègue à resolve_weapon_display
- get_weapon_faction (ui/i18n/weapons.py) : lecture JSON faction
"""

from __future__ import annotations

from unittest.mock import patch

import duckdb
import pytest

# ═════════════════════════════════════════════════════════════════════════════
# ensure_weapon_labels
# ═════════════════════════════════════════════════════════════════════════════


class TestEnsureWeaponLabels:
    """Tests de la migration ensure_weapon_labels sur metadata.duckdb."""

    @pytest.fixture()
    def conn(self):
        c = duckdb.connect(":memory:")
        yield c
        c.close()

    def test_creates_table(self, conn) -> None:
        """La table weapon_labels est créée si elle n'existe pas."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        exists = conn.execute(
            "SELECT 1 FROM information_schema.tables WHERE table_name='weapon_labels'"
        ).fetchone()
        assert exists is not None

    def test_primary_key_is_ubigint(self, conn) -> None:
        """weapon_id est de type UBIGINT."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name='weapon_labels' AND column_name='weapon_id'"
        ).fetchone()
        assert col_type is not None
        assert col_type[0].upper() == "UBIGINT"

    def test_sentinels_present(self, conn) -> None:
        """Les 3 sentinelles (0, 1, 2) sont présentes avec les bons labels."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        rows = conn.execute(
            "SELECT weapon_id, name_en, name_fr FROM weapon_labels WHERE weapon_id < 3 ORDER BY weapon_id"
        ).fetchall()
        assert len(rows) == 3
        assert rows[0] == (0, "Grenade", "Grenade")
        assert rows[1] == (1, "Melee", "Corps à corps")
        assert rows[2] == (2, "Vehicle", "Véhicule")

    def test_all_known_weapons_loaded(self, conn) -> None:
        """Toutes les armes de WEAPON_INT_TO_NAME sont présentes + 3 sentinelles."""
        from src.analysis._weapon_data import WEAPON_INT_TO_NAME
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        count = conn.execute("SELECT COUNT(*) FROM weapon_labels").fetchone()[0]
        expected = len(WEAPON_INT_TO_NAME) + 3  # +3 sentinelles
        assert count == expected

    def test_cindershot_translated_to_fr(self, conn) -> None:
        """Cindershot → Crémateur en FR."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        row = conn.execute(
            "SELECT name_fr FROM weapon_labels WHERE name_en = 'Cindershot'"
        ).fetchone()
        assert row is not None
        assert row[0] == "Crémateur"

    def test_fusion_variant_has_canonical_fr(self, conn) -> None:
        """M392 Bandit → label FR = 'Bandit EVO' (canonique via WEAPON_FUSION_MAP)."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        row = conn.execute(
            "SELECT name_fr FROM weapon_labels WHERE name_en = 'M392 Bandit'"
        ).fetchone()
        assert row is not None
        assert row[0] == "Bandit EVO"

    def test_idempotent_same_count(self, conn) -> None:
        """Appeler 2× ne duplique pas les entrées."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)
        count1 = conn.execute("SELECT COUNT(*) FROM weapon_labels").fetchone()[0]
        ensure_weapon_labels(conn)
        count2 = conn.execute("SELECT COUNT(*) FROM weapon_labels").fetchone()[0]
        assert count1 == count2

    def test_idempotent_existing_row_not_overwritten(self, conn) -> None:
        """Une ligne customisée existante n'est pas écrasée par INSERT OR IGNORE."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)
        # Modifier manuellement une traduction
        conn.execute("UPDATE weapon_labels SET name_fr = 'OverrideFR' WHERE weapon_id = 0")
        # Ré-appeler : le override doit être préservé
        ensure_weapon_labels(conn)
        row = conn.execute("SELECT name_fr FROM weapon_labels WHERE weapon_id = 0").fetchone()
        assert row[0] == "OverrideFR"

    def test_all_name_en_not_null(self, conn) -> None:
        """Toutes les entrées ont name_en non nul."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        null_count = conn.execute(
            "SELECT COUNT(*) FROM weapon_labels WHERE name_en IS NULL OR name_en = ''"
        ).fetchone()[0]
        assert null_count == 0

    def test_all_name_fr_not_null(self, conn) -> None:
        """Toutes les entrées ont name_fr non nul."""
        from src.data.sync.migrations import ensure_weapon_labels

        ensure_weapon_labels(conn)

        null_count = conn.execute(
            "SELECT COUNT(*) FROM weapon_labels WHERE name_fr IS NULL OR name_fr = ''"
        ).fetchone()[0]
        assert null_count == 0


# ═════════════════════════════════════════════════════════════════════════════
# Migration step enregistré dans le registre
# ═════════════════════════════════════════════════════════════════════════════


class TestWeaponLabelsMigrationStep:
    """Vérifie que le step add_weapon_labels est bien enregistré."""

    def test_step_registered(self) -> None:
        """add_weapon_labels doit figurer dans le registre MIGRATIONS."""
        import src.data.migration.steps  # noqa: F401 — force chargement
        from src.data.migration.registry import MIGRATIONS

        names = [m.name for m in MIGRATIONS]
        assert "add_weapon_labels" in names

    def test_step_targets_metadata_db(self) -> None:
        """Le step doit cibler target_db='metadata'."""
        import src.data.migration.steps  # noqa: F401
        from src.data.migration.registry import MIGRATIONS

        mig = next(m for m in MIGRATIONS if m.name == "add_weapon_labels")
        assert mig.target_db == "metadata"


# ═════════════════════════════════════════════════════════════════════════════
# get_weapon_label (ui/i18n/weapons.py)
# ═════════════════════════════════════════════════════════════════════════════


class TestGetWeaponLabel:
    """Tests de get_weapon_label — délégation à resolve_weapon_display."""

    def setup_method(self) -> None:
        from src.analysis._weapon_data import _resolve_weapon_cached

        _resolve_weapon_cached.cache_clear()

    def teardown_method(self) -> None:
        from src.analysis._weapon_data import _resolve_weapon_cached

        _resolve_weapon_cached.cache_clear()

    def test_delegates_to_resolve_weapon_display(self) -> None:
        """get_weapon_label doit retourner le même résultat que resolve_weapon_display."""
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT, resolve_weapon_display
        from src.ui.i18n.weapons import get_weapon_label

        # Cindershot FR
        cid = WEAPON_NAME_TO_INT["Cindershot"]
        assert get_weapon_label(cid, lang="fr") == resolve_weapon_display(cid, lang="fr")

    def test_sentinel_melee_fr(self) -> None:
        from src.ui.i18n.weapons import get_weapon_label

        assert get_weapon_label(1, lang="fr") == "Corps à corps"

    def test_sentinel_melee_en(self) -> None:
        from src.ui.i18n.weapons import get_weapon_label

        assert get_weapon_label(1, lang="en") == "Melee"

    def test_unknown_id_returns_weapon_id_fallback(self) -> None:
        from src.ui.i18n.weapons import get_weapon_label

        result = get_weapon_label(99999999, lang="fr")
        assert result == "weapon_99999999"

    def test_cindershot_fr(self) -> None:
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT
        from src.ui.i18n.weapons import get_weapon_label

        cid = WEAPON_NAME_TO_INT["Cindershot"]
        assert get_weapon_label(cid, lang="fr") == "Crémateur"

    def test_cindershot_en(self) -> None:
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT
        from src.ui.i18n.weapons import get_weapon_label

        cid = WEAPON_NAME_TO_INT["Cindershot"]
        assert get_weapon_label(cid, lang="en") == "Cindershot"


# ═════════════════════════════════════════════════════════════════════════════
# get_weapon_faction (ui/i18n/weapons.py)
# ═════════════════════════════════════════════════════════════════════════════


class TestGetWeaponFaction:
    """Tests de get_weapon_faction — lecture depuis weapons_{lang}.json."""

    def test_known_id_fr_returns_faction(self) -> None:
        """Un ID présent dans weapons_fr.json retourne la faction."""
        from src.ui.i18n.weapons import get_weapon_faction

        # ID 1 = Melee dans les JSONs i18n (sentinelle)
        result = get_weapon_faction(1, lang="fr")
        # Peut être 'Neutre' ou 'Unknown' si absent — pas d'erreur
        assert isinstance(result, str)

    def test_unknown_id_returns_unknown(self) -> None:
        """Un ID absent des JSONs retourne 'Unknown'."""
        from src.ui.i18n.weapons import get_weapon_faction

        result = get_weapon_faction(99999999, lang="fr")
        assert result == "Unknown"

    def test_works_for_en_lang(self) -> None:
        """get_weapon_faction fonctionne pour lang='en'."""
        from src.ui.i18n.weapons import get_weapon_faction

        result = get_weapon_faction(1, lang="en")
        assert isinstance(result, str)

    def test_json_missing_returns_unknown(self, tmp_path) -> None:
        """Chemin JSON absent → 'Unknown' sans exception."""
        from src.ui.i18n import weapons as weapons_mod
        from src.ui.i18n.weapons import _load_weapons_json, get_weapon_faction

        _load_weapons_json.cache_clear()
        try:
            with patch.object(weapons_mod, "_I18N_DIR", tmp_path):
                result = get_weapon_faction(9999, lang="fr")
            assert result == "Unknown"
        finally:
            _load_weapons_json.cache_clear()

    def test_json_malformed_returns_unknown_and_logs_error(self, tmp_path) -> None:
        """JSON invalide → 'Unknown' + logger.error (ligne 32 weapons.py)."""
        from src.ui.i18n import weapons as weapons_mod
        from src.ui.i18n.weapons import _load_weapons_json, get_weapon_faction

        malformed = tmp_path / "weapons_fr.json"
        malformed.write_text("{invalid json{{", encoding="utf-8")

        _load_weapons_json.cache_clear()
        try:
            with patch.object(weapons_mod, "_I18N_DIR", tmp_path):
                result = get_weapon_faction(9999, lang="fr")
            assert result == "Unknown"
        finally:
            _load_weapons_json.cache_clear()
