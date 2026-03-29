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
    """Arena:Slayer → préfixe Arena redondant (catég. Assassin) → mode seul."""
    _skip_if_no_db()
    assert translate_pair_name("Arena:Slayer on Aquarius", "fr") == "Assassin"


def test_btb_ctf_fr() -> None:
    """BTB:CTF → préfixe BTB redondant (catég. BTB) → mode seul."""
    _skip_if_no_db()
    assert translate_pair_name("BTB:CTF on Fragmentation", "fr") == "Capture du drapeau"


def test_arena_slayer_en() -> None:
    """Arena:Slayer EN → préfixe Arena redondant (catég. Assassin) → mode seul."""
    _skip_if_no_db()
    assert translate_pair_name("Arena:Slayer on Aquarius", "en") == "Slayer"


# ---------------------------------------------------------------------------
# 2. Override _pairs
# ---------------------------------------------------------------------------


def test_assault_neutral_bomb_override() -> None:
    """Override Assault:Neutral Bomb → label direct depuis mode_pair_overrides."""
    _skip_if_no_db()
    assert translate_pair_name("Assault:Neutral Bomb on Curfew", "fr") == "Bombe neutre"


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
    """Strip ' on Recharge' puis préfixe Arena redondant → mode seul."""
    _skip_if_no_db()
    assert translate_pair_name("Arena:Oddball on Recharge", "fr") == "Oddball"


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
    """Langue absente en DB → tables vides + préfixe Arena redondant → mode_EN seul."""
    _skip_if_no_db()
    result = translate_pair_name("Arena:Slayer on Aquarius", "de")
    assert result == "Slayer"  # Arena redondant (catég. Assassin), tables vides → mode_EN


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


# ---------------------------------------------------------------------------
# 10. Mode d\u00e9grad\u00e9 : DB absente
# ---------------------------------------------------------------------------


def test_degraded_mode_db_absent(
    monkeypatch: pytest.MonkeyPatch, caplog: pytest.LogCaptureFixture
) -> None:
    """Sans metadata.duckdb : warning + passthrough du nom brut."""
    from pathlib import Path

    monkeypatch.setattr(
        "src.utils.paths.get_metadata_db_path",
        lambda: Path("/tmp/absent_metadata_xyzzy.duckdb"),
    )
    translations._load_mode_tables.cache_clear()
    with caplog.at_level(logging.WARNING, logger="src.ui.translations"):
        result = translate_pair_name("Arena:Slayer on Aquarius", "fr")
    assert "introuvable" in caplog.text
    # Passthrough : s\u00e9parateur fallback + termes anglais
    assert result is not None
    assert "Arena" in result or "Slayer" in result


def test_degraded_mode_returns_empty_dict(monkeypatch: pytest.MonkeyPatch) -> None:
    """_load_mode_tables retourne un dict vide (stable) quand la DB est absente."""
    from pathlib import Path

    monkeypatch.setattr(
        "src.utils.paths.get_metadata_db_path",
        lambda: Path("/tmp/absent_metadata_xyzzy.duckdb"),
    )
    translations._load_mode_tables.cache_clear()
    result = translations._load_mode_tables("fr")
    assert result["mode_prefix_names"] == {}
    assert result["mode_name_tr"] == {}
    assert result["mode_pair_overrides"] == {}


def test_degraded_mode_translate_playlist_uuid(caplog: pytest.LogCaptureFixture) -> None:
    """translate_playlist_name : UUID → 'Inconnue' avec warning."""
    from src.ui.translations import translate_playlist_name

    with caplog.at_level(logging.WARNING, logger="src.ui.translations"):
        result_fr = translate_playlist_name("a446725e-b281-414c-a21e-abcdef123456", "fr")
        result_en = translate_playlist_name("a446725e-b281-414c-a21e-abcdef123456", "en")
    assert result_fr == "Inconnue"
    assert result_en == "Unknown"
    assert "UUID brut" in caplog.text
