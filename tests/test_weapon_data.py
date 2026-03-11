"""Tests unitaires — _weapon_data (resolve_weapon_display, UBIGINT, fusion, migration).

Couvre les lacunes critiques identifiées lors de l'audit du refactoring weapon_id v5.7 :
- resolve_weapon_display() : 9 cas (None, sentinelles, connu, inconnu, fusion, lang)
- UBIGINT overflow : INSERT/SELECT round-trip avec IDs > INT64 max
- Migration weapon_kills : BIGINT → UBIGINT, legacy weapon_name, idempotence
- WEAPON_FUSION_MAP_ID : fusion end-to-end (insert + read + resolve)
"""

from __future__ import annotations

import duckdb
import pytest

from src.analysis._weapon_data import (
    EXCLUDED_WEAPON_IDS,
    GRENADE_WEAPON_ID,
    MELEE_WEAPON_ID,
    VEHICLE_WEAPON_ID,
    WEAPON_FUSION_MAP,
    WEAPON_FUSION_MAP_ID,
    WEAPON_INT_TO_NAME,
    WEAPON_NAME_FR,
    WEAPON_NAME_TO_INT,
    resolve_weapon_display,
)

# ═════════════════════════════════════════════════════════════════════════════
# resolve_weapon_display
# ═════════════════════════════════════════════════════════════════════════════


class TestResolveWeaponDisplay:
    """Tests de la fonction resolve_weapon_display()."""

    def test_none_returns_none(self):
        assert resolve_weapon_display(None) is None

    def test_melee_fr(self):
        assert resolve_weapon_display(MELEE_WEAPON_ID, lang="fr") == "Corps à corps"

    def test_melee_en(self):
        assert resolve_weapon_display(MELEE_WEAPON_ID, lang="en") == "Melee"

    def test_grenade_fr(self):
        assert resolve_weapon_display(GRENADE_WEAPON_ID, lang="fr") == "Grenade"

    def test_grenade_en(self):
        assert resolve_weapon_display(GRENADE_WEAPON_ID, lang="en") == "Grenade"

    def test_vehicle_fr(self):
        assert resolve_weapon_display(VEHICLE_WEAPON_ID, lang="fr") == "Véhicule"

    def test_vehicle_en(self):
        assert resolve_weapon_display(VEHICLE_WEAPON_ID, lang="en") == "Vehicle"

    def test_known_weapon_fr(self):
        br75_id = WEAPON_NAME_TO_INT["BR75"]
        result = resolve_weapon_display(br75_id, lang="fr")
        assert result == "BR75"

    def test_known_weapon_en(self):
        br75_id = WEAPON_NAME_TO_INT["BR75"]
        result = resolve_weapon_display(br75_id, lang="en")
        assert result == "BR75"

    def test_cindershot_fr_translated(self):
        """Cindershot → Crémateur en FR."""
        cid = WEAPON_NAME_TO_INT["Cindershot"]
        assert resolve_weapon_display(cid, lang="fr") == "Crémateur"

    def test_cindershot_en_canonical(self):
        """Cindershot → Cindershot en EN (pas de traduction)."""
        cid = WEAPON_NAME_TO_INT["Cindershot"]
        assert resolve_weapon_display(cid, lang="en") == "Cindershot"

    def test_unknown_weapon_fallback(self):
        """Un weapon_id inconnu retourne 'weapon_<id>'."""
        result = resolve_weapon_display(9999999999)
        assert result == "weapon_9999999999"

    def test_fusion_m392_bandit_to_bandit_evo_fr(self):
        """M392 Bandit est fusionné en Bandit EVO (FR)."""
        m392_id = WEAPON_NAME_TO_INT["M392 Bandit"]
        result = resolve_weapon_display(m392_id, lang="fr")
        assert result == WEAPON_NAME_FR["Bandit Evo"]

    def test_fusion_fuel_rod_to_spnkr_fr(self):
        """Fuel Rod SPNKr est fusionné en M41 SPNKr (FR)."""
        fr_id = WEAPON_NAME_TO_INT["Fuel Rod SPNKr"]
        result = resolve_weapon_display(fr_id, lang="fr")
        assert result == WEAPON_NAME_FR["M41 SPNKr"]


# ═════════════════════════════════════════════════════════════════════════════
# EXCLUDED_WEAPON_IDS
# ═════════════════════════════════════════════════════════════════════════════


class TestExcludedWeaponIds:
    """Vérifie que les sentinelles sont bien exclues."""

    def test_contains_all_sentinels(self):
        assert MELEE_WEAPON_ID in EXCLUDED_WEAPON_IDS
        assert GRENADE_WEAPON_ID in EXCLUDED_WEAPON_IDS
        assert VEHICLE_WEAPON_ID in EXCLUDED_WEAPON_IDS

    def test_no_real_weapon_excluded(self):
        """Aucune arme réelle ne doit être dans les exclusions."""
        for wid in WEAPON_INT_TO_NAME:
            assert wid not in EXCLUDED_WEAPON_IDS, f"{WEAPON_INT_TO_NAME[wid]} exclu à tort"


# ═════════════════════════════════════════════════════════════════════════════
# WEAPON_FUSION_MAP_ID consistency
# ═════════════════════════════════════════════════════════════════════════════


class TestWeaponFusionMapId:
    """Vérifie la cohérence de WEAPON_FUSION_MAP_ID."""

    def test_all_fusion_entries_mapped(self):
        """Chaque entrée de WEAPON_FUSION_MAP (str) doit avoir un équivalent dans _ID."""
        for source, target in WEAPON_FUSION_MAP.items():
            if source in WEAPON_NAME_TO_INT and target in WEAPON_NAME_TO_INT:
                src_id = WEAPON_NAME_TO_INT[source]
                tgt_id = WEAPON_NAME_TO_INT[target]
                assert src_id in WEAPON_FUSION_MAP_ID
                assert WEAPON_FUSION_MAP_ID[src_id] == tgt_id

    def test_fusion_target_not_in_source(self):
        """La cible de fusion ne doit pas elle-même être fusionnée (pas de chaîne)."""
        for tgt_id in WEAPON_FUSION_MAP_ID.values():
            assert tgt_id not in WEAPON_FUSION_MAP_ID, "Chaîne de fusion détectée"


# ═════════════════════════════════════════════════════════════════════════════
# UBIGINT overflow — round-trip INSERT/SELECT
# ═════════════════════════════════════════════════════════════════════════════


class TestUbigintRoundTrip:
    """Vérifie que les weapon_id uint64 survivent au round-trip DuckDB UBIGINT."""

    @pytest.fixture()
    def conn(self):
        c = duckdb.connect(":memory:")
        c.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                time_ms INTEGER NOT NULL,
                weapon_id UBIGINT,
                delta_ms INTEGER,
                confidence VARCHAR DEFAULT 'none',
                swap_detected BOOLEAN DEFAULT FALSE,
                delayed_damage BOOLEAN DEFAULT FALSE
            )
        """)
        yield c
        c.close()

    def test_large_weapon_id_above_int64_max(self, conn):
        """Sidekick (0xf408190f42c9679f) dépasse INT64 max."""
        sidekick_id = WEAPON_NAME_TO_INT["Mk51 Sidekick"]
        assert sidekick_id > (2**63 - 1), "Sidekick devrait dépasser INT64 max"
        conn.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            ("m1", "x1", 1000, sidekick_id, 100, "high", False, False),
        )
        row = conn.execute("SELECT weapon_id FROM weapon_kills WHERE match_id='m1'").fetchone()
        assert row[0] == sidekick_id

    def test_max_ubigint_value(self, conn):
        """Test avec la valeur max UBIGINT (2^64-1)."""
        max_val = (2**64) - 1
        conn.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            ("m2", "x1", 2000, max_val, None, "none", False, False),
        )
        row = conn.execute("SELECT weapon_id FROM weapon_kills WHERE match_id='m2'").fetchone()
        assert row[0] == max_val

    def test_all_known_weapons_roundtrip(self, conn):
        """Tous les weapon_id connus survivent au round-trip."""
        for idx, (wid, _name) in enumerate(WEAPON_INT_TO_NAME.items()):
            conn.execute(
                "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
                (f"m_{idx}", "x1", idx * 100, wid, None, "none", False, False),
            )
        rows = conn.execute(
            "SELECT match_id, weapon_id FROM weapon_kills ORDER BY time_ms"
        ).fetchall()
        stored_ids = {r[1] for r in rows}
        expected_ids = set(WEAPON_INT_TO_NAME.keys())
        assert stored_ids == expected_ids

    def test_null_weapon_id_preserved(self, conn):
        """weapon_id NULL est préservé (kill non résolu)."""
        conn.execute(
            "INSERT INTO weapon_kills VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
            ("m3", "x1", 3000, None, None, "none", False, False),
        )
        row = conn.execute("SELECT weapon_id FROM weapon_kills WHERE match_id='m3'").fetchone()
        assert row[0] is None


# ═════════════════════════════════════════════════════════════════════════════
# Migration weapon_kills schema
# ═════════════════════════════════════════════════════════════════════════════


class TestWeaponKillsMigration:
    """Vérifie _migrate_weapon_kills_schema et ensure_weapon_kills_table."""

    @pytest.fixture()
    def conn(self):
        c = duckdb.connect(":memory:")
        yield c
        c.close()

    def test_create_table_from_scratch(self, conn):
        """Table n'existe pas → créée avec UBIGINT."""
        from src.data.sync.migrations import ensure_weapon_kills_table

        ensure_weapon_kills_table(conn)
        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name='weapon_kills' AND column_name='weapon_id'"
        ).fetchone()
        assert col_type is not None
        assert col_type[0].upper() == "UBIGINT"

    def test_idempotent_no_change(self, conn):
        """Appeler 2x ne casse rien."""
        from src.data.sync.migrations import ensure_weapon_kills_table

        ensure_weapon_kills_table(conn)
        # Insérer des données
        conn.execute("INSERT INTO weapon_kills VALUES ('m1','x1',100,42,10,'high',false,false)")
        # Ré-appeler
        ensure_weapon_kills_table(conn)
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 1  # données préservées

    def test_migrate_legacy_weapon_name_schema(self, conn):
        """Ancien schéma per-kill avec weapon_name → conversion vers weapon_id UBIGINT."""
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT
        from src.data.sync.migrations import ensure_weapon_kills_table

        # Créer un ancien schéma per-kill avec weapon_name
        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR, xuid VARCHAR, time_ms INTEGER,
                weapon_name VARCHAR, delta_ms INTEGER,
                confidence VARCHAR, swap_detected BOOLEAN, delayed_damage BOOLEAN
            )
        """)
        conn.execute(
            "INSERT INTO weapon_kills VALUES "
            "('m1','x1',100,'BR75',50,'high',false,false),"
            "('m1','x1',200,'MELEE',30,'low',true,false),"
            "('m1','x1',300,'UNKNOWN',0,'none',false,false),"
            "('m1','x1',400,'?00004a9642c9679f',10,'medium',false,true)"
        )
        ensure_weapon_kills_table(conn)
        cols = {
            r[0]
            for r in conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name='weapon_kills'"
            ).fetchall()
        }
        assert "weapon_id" in cols
        assert "weapon_name" not in cols
        # Données préservées
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 4
        # BR75 correctement converti
        br75_id = conn.execute(
            "SELECT weapon_id FROM weapon_kills WHERE match_id='m1' AND time_ms=100"
        ).fetchone()[0]
        assert br75_id == WEAPON_NAME_TO_INT["BR75"]
        # MELEE → sentinel 1
        melee_id = conn.execute("SELECT weapon_id FROM weapon_kills WHERE time_ms=200").fetchone()[
            0
        ]
        assert melee_id == 1
        # UNKNOWN → NULL
        unk_id = conn.execute("SELECT weapon_id FROM weapon_kills WHERE time_ms=300").fetchone()[0]
        assert unk_id is None
        # ?hex → parsed uint64
        hex_id = conn.execute("SELECT weapon_id FROM weapon_kills WHERE time_ms=400").fetchone()[0]
        assert hex_id == int("00004a9642c9679f", 16)

    def test_migrate_legacy_aggregated_schema(self, conn):
        """Ancien schéma agrégé (kills, sans time_ms) → DROP+CREATE."""
        from src.data.sync.migrations import ensure_weapon_kills_table

        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR, xuid VARCHAR, weapon_name VARCHAR, kills INTEGER
            )
        """)
        conn.execute("INSERT INTO weapon_kills VALUES ('m1','x1','BR75',5)")
        ensure_weapon_kills_table(conn)
        cols = {
            r[0]
            for r in conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name='weapon_kills'"
            ).fetchall()
        }
        assert "weapon_id" in cols
        assert "weapon_name" not in cols
        # Anciennes données perdues (schéma agrégé non convertible)
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 0

    def test_migrate_bigint_to_ubigint(self, conn):
        """Schéma intermédiaire BIGINT → migre vers UBIGINT, données préservées."""
        from src.data.sync.migrations import ensure_weapon_kills_table

        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
                time_ms INTEGER NOT NULL, weapon_id BIGINT,
                delta_ms INTEGER, confidence VARCHAR DEFAULT 'none',
                swap_detected BOOLEAN DEFAULT FALSE,
                delayed_damage BOOLEAN DEFAULT FALSE
            )
        """)
        conn.execute("INSERT INTO weapon_kills VALUES ('m1','x1',100,42,10,'high',false,false)")
        ensure_weapon_kills_table(conn)
        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name='weapon_kills' AND column_name='weapon_id'"
        ).fetchone()
        assert col_type[0].upper() == "UBIGINT"
        # Données préservées
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 1
        wid = conn.execute("SELECT weapon_id FROM weapon_kills").fetchone()[0]
        assert wid == 42


# ═════════════════════════════════════════════════════════════════════════════
# Fusion end-to-end (insert + aggregate + resolve)
# ═════════════════════════════════════════════════════════════════════════════


class TestFusionEndToEnd:
    """Vérifie la fusion de variantes dans le pipeline complet."""

    @pytest.fixture()
    def conn(self):
        c = duckdb.connect(":memory:")
        c.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
                time_ms INTEGER NOT NULL, weapon_id UBIGINT,
                delta_ms INTEGER, confidence VARCHAR DEFAULT 'none',
                swap_detected BOOLEAN DEFAULT FALSE,
                delayed_damage BOOLEAN DEFAULT FALSE
            )
        """)
        yield c
        c.close()

    def test_m392_bandit_fused_with_bandit_evo(self, conn):
        """M392 Bandit et Bandit Evo comptés ensemble après fusion."""
        m392_id = WEAPON_NAME_TO_INT["M392 Bandit"]
        bandit_id = WEAPON_NAME_TO_INT["Bandit Evo"]
        # 2 kills M392, 3 kills Bandit Evo
        for i in range(2):
            conn.execute(
                "INSERT INTO weapon_kills VALUES (?,?,?,?,?,?,?,?)",
                ("m1", "x1", i * 100, m392_id, 50, "high", False, False),
            )
        for i in range(3):
            conn.execute(
                "INSERT INTO weapon_kills VALUES (?,?,?,?,?,?,?,?)",
                ("m1", "x1", (i + 2) * 100, bandit_id, 50, "high", False, False),
            )
        # Lecture + fusion
        rows = conn.execute(
            "SELECT weapon_id, COUNT(*)::INTEGER AS kills "
            "FROM weapon_kills WHERE xuid='x1' AND weapon_id IS NOT NULL "
            "GROUP BY weapon_id"
        ).fetchall()
        # Appliquer la fusion (comme dans le code UI)
        fused: dict[int, int] = {}
        for wid, kills in rows:
            canonical = WEAPON_FUSION_MAP_ID.get(wid, wid)
            fused[canonical] = fused.get(canonical, 0) + kills
        # Résultat : tout sous Bandit Evo (5 kills)
        assert fused[bandit_id] == 5
        assert m392_id not in fused

    def test_fusion_then_resolve_display(self, conn):
        """Après fusion, resolve_weapon_display retourne le nom canonique."""
        fr_spnkr_id = WEAPON_NAME_TO_INT["Fuel Rod SPNKr"]
        canonical_id = WEAPON_FUSION_MAP_ID[fr_spnkr_id]
        name_fr = resolve_weapon_display(canonical_id, lang="fr")
        assert name_fr == "M41 SPNKr"
