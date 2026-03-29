"""Tests unitaires pour src.ui.translations.

Valide la nouvelle architecture JSON-first de translate_pair_name() :
- Overrides explicites (_pairs dans modes_{lang}.json)
- Combinateur _prefixes + mode (chemin générique)
- Strip " on <carte>" avant lookup
- Normalisation de casse
- Fallback UUID → "Mode inconnu"
"""

from __future__ import annotations

from src.ui.translations import translate_pair_name, translate_playlist_name

# ---------------------------------------------------------------------------
# translate_pair_name — cas de base
# ---------------------------------------------------------------------------


class TestTranslatePairNameFR:
    """Tests en français (lang="fr")."""

    def test_none_returns_none(self) -> None:
        assert translate_pair_name(None) is None

    def test_empty_returns_none(self) -> None:
        assert translate_pair_name("") is None
        assert translate_pair_name("   ") is None

    def test_generic_combinator_arena_ctf(self) -> None:
        """Arena:CTF → préfixe Arena redondant (catég. Assassin) → juste le mode."""
        assert translate_pair_name("Arena:CTF") == "Capture du drapeau"

    def test_generic_combinator_btb_slayer(self) -> None:
        """BTB:Slayer → préfixe BTB redondant (catég. BTB) → juste le mode."""
        assert translate_pair_name("BTB:Slayer") == "Assassin"

    def test_generic_combinator_ranked_strongholds(self) -> None:
        """Ranked:Strongholds → préfixe Ranked redondant → juste le mode."""
        assert translate_pair_name("Ranked:Strongholds") == "Bases"

    def test_generic_map_stripped_then_combined(self) -> None:
        """Strip ' on <carte>' puis résolution (préfixe Arena redondant)."""
        assert translate_pair_name("Arena:CTF on Aquarius") == "Capture du drapeau"

    def test_case_normalization(self) -> None:
        """Normalisation douce de la casse : préfixe Arena → Assassin → mode seul."""
        assert translate_pair_name("arena:Slayer") == "Assassin"

    # -- _pairs overrides --

    def test_pairs_assault_neutral_bomb(self) -> None:
        """Override _pairs : Assault:Neutral Bomb → label simplifié (sans préfixe Arène)."""
        assert translate_pair_name("Assault:Neutral Bomb") == "Bombe neutre"

    def test_pairs_assault_neutral_bomb_squad(self) -> None:
        assert translate_pair_name("Assault:Neutral Bomb Squad") == "Bombe neutre"

    def test_pairs_assault_one_bomb_on_curfew(self) -> None:
        """Override _pairs exact sur nom avec carte."""
        assert translate_pair_name("Assault:One Bomb on Curfew") == "Bombe neutre"

    def test_pairs_arena_shotty_snipes(self) -> None:
        """Override _pairs : pas de préfixe dans la traduction."""
        assert translate_pair_name("Arena:Shotty Snipes Slayer") == "Fusils snipers à grenaille"

    def test_pairs_community_fiesta_slayer_high_ground(self) -> None:
        assert translate_pair_name("Community:Fiesta Slayer on High Ground") == "Fiesta"

    def test_pairs_community_fiesta_slayer_snowbound(self) -> None:
        assert translate_pair_name("Community:Fiesta Slayer on Snowbound") == "Fiesta"

    def test_pairs_community_shotty_dynasty(self) -> None:
        assert (
            translate_pair_name("Community:Shotty Snipe Slayer FFA on Dynasty")
            == "Fusils snipers à grenaille"
        )

    def test_pairs_fiesta_slayer_behemoth(self) -> None:
        assert translate_pair_name("Fiesta:Slayer on Behemoth - Forge") == "Fiesta"

    def test_pairs_fiesta_slayer_catalyst(self) -> None:
        assert translate_pair_name("Fiesta:Slayer on Catalyst - Forge") == "Fiesta"

    def test_pairs_husky_raid_assault(self) -> None:
        assert translate_pair_name("Husky Raid:Assault on Urban Raid") == "Husky Raid"

    def test_pairs_survive_undead(self) -> None:
        assert (
            translate_pair_name("Survive The Undead 3.0 on TFF | Night Of The Undead")
            == "Survivre aux morts-vivants 3.0"
        )

    # -- UUID --

    def test_uuid_returns_unknown(self) -> None:
        result = translate_pair_name("a446725e-b281-414c-a21e")
        assert result == "Mode inconnu"


class TestTranslatePairNameEN:
    """Tests en anglais (lang="en")."""

    def test_generic_combinator_en(self) -> None:
        """Arena:CTF EN → préfixe Arena redondant (catég. Assassin) → juste le mode."""
        assert translate_pair_name("Arena:CTF", lang="en") == "CTF"

    def test_pairs_en_shotty_snipes(self) -> None:
        """Override _pairs EN : Shotty Snipers (sans préfixe)."""
        assert translate_pair_name("Arena:Shotty Snipes Slayer", lang="en") == "Shotty Snipers"

    def test_pairs_en_community_fiesta(self) -> None:
        assert translate_pair_name("Community:Fiesta Slayer on High Ground", lang="en") == "Fiesta"

    def test_pairs_en_survive_undead(self) -> None:
        assert (
            translate_pair_name("Survive The Undead 3.0 on TFF | Night Of The Undead", lang="en")
            == "Survive the Undead"
        )

    def test_uuid_en_returns_unknown(self) -> None:
        result = translate_pair_name("a446725e-b281-414c-a21e", lang="en")
        assert result == "Unknown mode"

    def test_none_returns_none_en(self) -> None:
        assert translate_pair_name(None, lang="en") is None


# ---------------------------------------------------------------------------
# translate_playlist_name
# ---------------------------------------------------------------------------


class TestTranslatePlaylistNameFR:
    def test_known_playlist_fr(self) -> None:
        """Playlists connues traduites en FR via static/i18n/playlists_fr.json."""
        assert translate_playlist_name("Quick Play") == "Partie rapide"
        assert translate_playlist_name("Ranked Arena") == "Arène classée"

    def test_uuid_playlist_returns_inconnue(self) -> None:
        """UUID brut → 'Inconnue' (metadata.duckdb incomplet)."""
        result = translate_playlist_name("a446725e-b281-414c-a21e-1234567890ab")
        assert result == "Inconnue"

    def test_none_returns_none(self) -> None:
        assert translate_playlist_name(None) is None

    def test_unknown_playlist_passthrough(self) -> None:
        """Playlist inconnue : retourne telle quelle."""
        assert translate_playlist_name("Unknown Playlist XYZ123") == "Unknown Playlist XYZ123"
