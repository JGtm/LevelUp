"""Tests pour translate_pair_name() via metadata.duckdb.

Valide l'architecture DB-backed (v6) :
- Overrides explicites (mode_pair_overrides)
- Combinatoire préfixe + séparateur + mode  (mode_prefix_names + mode_name_tr)
- Mode seul (sans catégorie)
- Fallback langue inconnue
- Cache LRU process-level
"""

from __future__ import annotations

import logging

import pytest

from src.ui import translations
from src.ui.translations import translate_pair_name

pytestmark = pytest.mark.usefixtures("clear_mode_cache")


@pytest.fixture(autouse=True)
def clear_mode_cache():
    """Vide le cache LRU avant chaque test pour isoler les appels DB."""
    translations._load_mode_tables.cache_clear()
    yield
    translations._load_mode_tables.cache_clear()


def _skip_if_no_db() -> None:
    """Passe le test si metadata.duckdb n'est pas migré."""
    try:
        from src.utils.paths import get_metadata_db_path

        db_path = get_metadata_db_path()
        if not db_path.exists():
            pytest.skip("metadata.duckdb introuvable")
        import duckdb

        with duckdb.connect(str(db_path), read_only=True) as conn:
            found = conn.execute(
                "SELECT 1 FROM information_schema.tables WHERE table_name='mode_prefix_names' LIMIT 1"
            ).fetchone()
            if not found:
                pytest.skip("tables modes absentes de metadata.duckdb (migration requise)")
    except Exception as exc:
        pytest.skip(f"DB inaccessible: {exc}")


# ---------------------------------------------------------------------------
# 1. Combinatoire générique
# ---------------------------------------------------------------------------


def test_arena_slayer_fr() -> None:
    """Préfixe + mode en FR via combinatoire."""
    _skip_if_no_db()
    assert translate_pair_name("Arena:Slayer on Aquarius", "fr") == "Arène : Assassin"


def test_btb_ctf_fr() -> None:
    """Préfixe BTB long + mode CTF en FR."""
    _skip_if_no_db()
    assert (
        translate_pair_name("BTB:CTF on Fragmentation", "fr")
        == "Grande bataille en équipe : Capture du drapeau"
    )


def test_arena_slayer_en() -> None:
    """Combinatoire EN — séparateur ': ' (sans espace avant)."""
    _skip_if_no_db()
    assert translate_pair_name("Arena:Slayer on Aquarius", "en") == "Arena: Slayer"


# ---------------------------------------------------------------------------
# 2. Override _pairs
# ---------------------------------------------------------------------------


def test_assault_neutral_bomb_override() -> None:
    """Override Assault:Neutral Bomb (préfixe trompeur — traduction non standard)."""
    _skip_if_no_db()
    assert translate_pair_name("Assault:Neutral Bomb on Curfew", "fr") == "Arène : Bombe neutre"


def test_survive_undead_override() -> None:
    """Override sans ':' — cas non combinatoire."""
    _skip_if_no_db()
    assert (
        translate_pair_name("Survive The Undead 3.0 on TFF | Night Of The Undead", "fr")
        == "Survivre aux morts-vivants 3.0"
    )


# ---------------------------------------------------------------------------
# 3. Mode seul (sans catégorie)
# ---------------------------------------------------------------------------


def test_mode_alone_fr() -> None:
    """Mode seul sans préfixe — lookup dans mode_name_tr."""
    _skip_if_no_db()
    assert translate_pair_name("Slayer", "fr") == "Assassin"


# ---------------------------------------------------------------------------
# 4. Strip " on <carte>" avant lookup
# ---------------------------------------------------------------------------


def test_strip_map_suffix() -> None:
    """Strip ' on Recharge' avant combinatoire."""
    _skip_if_no_db()
    assert translate_pair_name("Arena:Oddball on Recharge", "fr") == "Arène : Oddball"


# ---------------------------------------------------------------------------
# 5. UUID → warning + label inconnu
# ---------------------------------------------------------------------------


def test_uuid_logs_warning(caplog: pytest.LogCaptureFixture) -> None:
    """UUID brut déclenche un warning et retourne un label non-None."""
    with caplog.at_level(logging.WARNING, logger="src.ui.translations"):
        result = translate_pair_name("a446725e-b281-414c-a21e-12345678abcd", "fr")
    assert "UUID" in caplog.text
    assert result is not None


# ---------------------------------------------------------------------------
# 6. None / vide → None
# ---------------------------------------------------------------------------


def test_none_input() -> None:
    assert translate_pair_name(None, "fr") is None
    assert translate_pair_name("", "fr") is None


# ---------------------------------------------------------------------------
# 7. Cache LRU process-level
# ---------------------------------------------------------------------------


def test_lru_cache_hit() -> None:
    """_load_mode_tables n'est chargé depuis la DB qu'une seule fois (un seul miss)."""
    _skip_if_no_db()
    translations._load_mode_tables.cache_clear()
    translate_pair_name("Arena:Slayer on Aquarius", "fr")
    translate_pair_name("Arena:CTF on Recharge", "fr")
    info = translations._load_mode_tables.cache_info()
    assert info.misses == 1  # DB ouverte 1 seule fois
    assert info.hits >= 1  # cache utilisé au moins une fois


# ---------------------------------------------------------------------------
# 8. Langue inconnue → fallback gracieux
# ---------------------------------------------------------------------------


def test_unknown_lang_fallback() -> None:
    """Langue absente en DB → retourne préfixe_EN + séparateur_fallback + mode_EN."""
    _skip_if_no_db()
    result = translate_pair_name("Arena:Slayer on Aquarius", "de")
    assert result == "Arena : Slayer"  # séparateur fallback ' : '


# ---------------------------------------------------------------------------
# 9. Régression — overrides existants
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    "raw,expected",
    [
        ("Community:Fiesta Slayer on High Ground", "Fiesta"),
        ("Husky Raid:Assault on Urban Raid", "Husky Raid"),
        ("Arena:Shotty Snipes Slayer", "Fusils snipers à grenaille"),
    ],
)
def test_pairs_overrides_regression(raw: str, expected: str) -> None:
    """Régression : overrides _pairs FR traduits correctement."""
    _skip_if_no_db()
    assert translate_pair_name(raw, "fr") == expected
