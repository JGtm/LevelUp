"""Tests unitaires pour src/analysis/mode_display.py.

Tous les tests sont purs (zéro DB, zéro Streamlit).
Les tables de traduction sont injectées comme fixtures.
"""

from __future__ import annotations

from src.analysis.mode_display import resolve_display_mode

# ── Fixtures tables ───────────────────────────────────────────────────────────

TABLES_FR: dict = {
    "mode_pair_overrides": {
        "Fiesta:Slayer": "Fiesta",
        "Community:Shotty Snipe Slayer Ffa": "Fusils snipers à grenaille",
        "Arena:Shotty Snipes Slayer": "Fusils snipers à grenaille",
    },
    "mode_name_tr": {
        "Slayer": "Assassin",
        "CTF": "Capture du drapeau",
        "King Of The Hill": "Roi de la colline",
        "Oddball": "Oddball",
        "Strongholds": "Bases",
        "Total Control": "Contrôle total",
        "Stockpile": "Stockage",
        "Fiesta Slayer": "Fiesta",
        "Fiesta Ctf": "Fiesta CDD",
        "Fiesta Total Control": "Fiesta Contrôle total",
        "Team Slayer": "Assassin en équipe",
    },
    "mode_prefix_names": {
        "Arena": "Arène",
        "BTB": "Grande bataille en équipe",
        "BTB Heavies": "Grande bataille en équipe Heavies",
        "Ranked": "Classé",
        "Fiesta": "Fiesta",
        "Super Fiesta": "Super Fiesta",
        "Husky Raid": "Husky Raid",
        "Super Husky Raid": "Super Husky Raid",
        "Firefight": "Baptême du feu",
        "Gruntpocalypse": "Gruntpocalypse",
        "Tactical": "Tactique",
        "Community": "Communauté",
        "Event": "Événement",
        "Assault": "Assaut",
    },
    "separator": " : ",
}

TABLES_EN: dict = {
    "mode_pair_overrides": {
        "Fiesta:Slayer": "Fiesta",
    },
    "mode_name_tr": {
        "Slayer": "Slayer",
        "CTF": "CTF",
        "King Of The Hill": "King of the Hill",
        "Oddball": "Oddball",
        "Strongholds": "Strongholds",
        "Total Control": "Total Control",
        "Stockpile": "Stockpile",
    },
    "mode_prefix_names": {
        "Arena": "Arena",
        "BTB": "BTB",
        "BTB Heavies": "BTB Heavies",
        "Ranked": "Ranked",
        "Fiesta": "Fiesta",
        "Super Fiesta": "Super Fiesta",
        "Husky Raid": "Husky Raid",
        "Super Husky Raid": "Super Husky Raid",
        "Firefight": "Firefight",
        "Gruntpocalypse": "Gruntpocalypse",
        "Tactical": "Tactical",
        "Community": "Community",
        "Event": "Event",
        "Assault": "Assault",
    },
    "separator": ": ",
}


# ── Format standard — préfixe redondant supprimé ──────────────────────────────


class TestStandardFormatRedundant:
    def test_btb_slayer_in_btb(self):
        assert resolve_display_mode("BTB:Slayer", "BTB", "fr", TABLES_FR) == "Assassin"

    def test_btb_ctf_in_btb(self):
        assert resolve_display_mode("BTB:CTF", "BTB", "fr", TABLES_FR) == "Capture du drapeau"

    def test_btb_total_control_in_btb(self):
        assert resolve_display_mode("BTB:Total Control", "BTB", "fr", TABLES_FR) == "Contrôle total"

    def test_arena_slayer_in_assassin(self):
        assert resolve_display_mode("Arena:Slayer", "Assassin", "fr", TABLES_FR) == "Assassin"

    def test_arena_ctf_in_assassin(self):
        assert (
            resolve_display_mode("Arena:CTF", "Assassin", "fr", TABLES_FR) == "Capture du drapeau"
        )

    def test_ranked_slayer_in_ranked(self):
        assert resolve_display_mode("Ranked:Slayer", "Ranked", "fr", TABLES_FR) == "Assassin"

    def test_ranked_ctf_in_ranked(self):
        assert resolve_display_mode("Ranked:CTF", "Ranked", "fr", TABLES_FR) == "Capture du drapeau"

    def test_firefight_koth_in_firefight(self):
        assert (
            resolve_display_mode("Firefight:King Of The Hill", "Firefight", "fr", TABLES_FR)
            == "Roi de la colline"
        )


# ── Format standard — préfixe NON redondant conservé ─────────────────────────


class TestStandardFormatNonRedundant:
    def test_fiesta_slayer_in_assassin_playlist(self):
        """Fiesta dans une playlist Assassin → Fiesta n'est pas redondant → conserver."""
        result = resolve_display_mode("Fiesta:Slayer", "Assassin", "fr", TABLES_FR)
        # Override exact attendu depuis mode_pair_overrides
        assert result == "Fiesta"

    def test_btb_slayer_in_assassin_context(self):
        """BTB:Slayer dans contexte Assassin → préfixe BTB ≠ Assassin → label complet."""
        result = resolve_display_mode("BTB:Slayer", "Assassin", "fr", TABLES_FR)
        assert result == "Grande bataille en équipe : Assassin"

    def test_arena_slayer_in_btb_context(self):
        """Arena:Slayer dans contexte BTB → Arena ≠ BTB → label complet."""
        result = resolve_display_mode("Arena:Slayer", "BTB", "fr", TABLES_FR)
        assert result == "Arène : Assassin"


# ── Qualificatif Heavies conservé ────────────────────────────────────────────


class TestHeaviesQualifier:
    def test_btb_heavies_ctf_in_btb(self):
        result = resolve_display_mode("BTB Heavies:CTF", "BTB", "fr", TABLES_FR)
        assert result == "Heavies : Capture du drapeau"

    def test_btb_heavies_slayer_in_btb(self):
        result = resolve_display_mode("BTB Heavies:Slayer", "BTB", "fr", TABLES_FR)
        assert result == "Heavies : Assassin"

    def test_btb_heavies_total_control_in_btb(self):
        result = resolve_display_mode("BTB Heavies:Total Control", "BTB", "fr", TABLES_FR)
        assert result == "Heavies : Contrôle total"


# ── Format inversé ────────────────────────────────────────────────────────────


class TestInvertedFormat:
    def test_ctf_arena_in_assassin(self):
        """CTF:Arena → format inversé → prefix=Arena, mode=CTF."""
        result = resolve_display_mode("CTF:Arena", "Assassin", "fr", TABLES_FR)
        assert result == "Capture du drapeau"

    def test_koth_arena_in_assassin(self):
        result = resolve_display_mode("KOTH:Arena", "Assassin", "fr", TABLES_FR)
        # KOTH pas dans mode_name_tr → fallback brut
        assert result == "KOTH"

    def test_slayer_btb_heavies_in_btb(self):
        """Slayer:BTB Heavies → format inversé → prefix=BTB Heavies, mode=Slayer."""
        result = resolve_display_mode("Slayer:BTB Heavies", "BTB", "fr", TABLES_FR)
        assert result == "Heavies : Assassin"

    def test_ctf_btb_in_btb(self):
        """CTF:BTB → right=BTB est un préfixe connu → inversé → prefix=BTB, mode=CTF."""
        result = resolve_display_mode("CTF:BTB", "BTB", "fr", TABLES_FR)
        assert result == "Capture du drapeau"


# ── Variants sans séparateur ──────────────────────────────────────────────────


class TestNoSeparator:
    def test_castle_wars(self):
        result = resolve_display_mode("CASTLE WARS", "Fiesta", "fr", TABLES_FR)
        assert result == "CASTLE WARS"

    def test_survive_the_undead(self):
        result = resolve_display_mode("TFF | Survive The Undead", "Other", "fr", TABLES_FR)
        assert result == "TFF | Survive The Undead"


# ── Overrides exact ───────────────────────────────────────────────────────────


class TestPairOverrides:
    def test_fiesta_slayer_override(self):
        """mode_pair_overrides a la priorité absolue."""
        result = resolve_display_mode("Fiesta:Slayer", "Fiesta", "fr", TABLES_FR)
        assert result == "Fiesta"

    def test_shotty_snipes_override(self):
        result = resolve_display_mode("Arena:Shotty Snipes Slayer", "Assassin", "fr", TABLES_FR)
        assert result == "Fusils snipers à grenaille"


# ── Guards input ──────────────────────────────────────────────────────────────


class TestInputGuards:
    def test_none_input(self):
        result = resolve_display_mode(None, "Assassin", "fr", TABLES_FR)
        assert result == "Mode inconnu"

    def test_empty_string(self):
        result = resolve_display_mode("", "Assassin", "fr", TABLES_FR)
        assert result == "Mode inconnu"

    def test_none_input_en(self):
        result = resolve_display_mode(None, "Assassin", "en", TABLES_EN)
        assert result == "Unknown mode"


# ── Langue EN ────────────────────────────────────────────────────────────────


class TestEnglishLang:
    def test_btb_ctf_en(self):
        assert resolve_display_mode("BTB:CTF", "BTB", "en", TABLES_EN) == "CTF"

    def test_arena_slayer_en(self):
        assert resolve_display_mode("Arena:Slayer", "Assassin", "en", TABLES_EN) == "Slayer"

    def test_btb_heavies_en(self):
        result = resolve_display_mode("BTB Heavies:CTF", "BTB", "en", TABLES_EN)
        assert result == "Heavies: CTF"
