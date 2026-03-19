"""Résolution medal_name_id → nom / description FR/EN.

Sources par ordre de priorité :
1. Table ``medals`` dans ``metadata.duckdb`` (quand elle existe)
2. Fichiers JSON statiques ``static/medals/medals_{lang}.json``
3. Fallback : ``str(medal_name_id)``
"""

from __future__ import annotations

import json
import logging
from functools import cache
from pathlib import Path
from typing import Final

from src.utils.paths import get_metadata_db_path

logger = logging.getLogger(__name__)

_STATIC_DIR: Final[Path] = Path(__file__).resolve().parents[2] / "static" / "medals"


@cache
def _load_json_map(lang: str) -> dict[int, str]:
    """Charge le mapping medal_name_id (int) → nom depuis le fichier JSON statique."""
    path = _STATIC_DIR / f"medals_{lang}.json"
    if not path.exists():
        return {}
    try:
        raw: dict = json.loads(path.read_text(encoding="utf-8"))
        return {int(k): str(v).strip() for k, v in raw.items() if str(v).strip()}
    except Exception:
        logger.debug("_load_json_map(%s): erreur lecture", lang)
        return {}


@cache
def _load_description_map(lang: str) -> dict[int, str]:
    """Charge le mapping medal_name_id (int) → description depuis le fichier JSON statique."""
    path = _STATIC_DIR / f"medals_descriptions_{lang}.json"
    if not path.exists():
        return {}
    try:
        raw: dict = json.loads(path.read_text(encoding="utf-8"))
        return {int(k): str(v).strip() for k, v in raw.items() if str(v).strip()}
    except Exception:
        logger.debug("_load_description_map(%s): erreur lecture", lang)
        return {}


def _resolve_text_from_db(medal_name_id: int, lang: str, columns: list[str]) -> str | None:
    """Résolution d'un texte de médaille depuis metadata.duckdb.

    Retourne None si la table n'existe pas ou si aucune colonne candidate n'est disponible.
    """
    try:
        import duckdb

        db_path = get_metadata_db_path()
        if not db_path.exists():
            return None

        with duckdb.connect(str(db_path), read_only=True) as conn:
            has_table = conn.execute(
                "SELECT 1 FROM information_schema.tables WHERE table_name = 'medals' LIMIT 1"
            ).fetchone()
            if not has_table:
                return None
            available_columns = {
                str(row[0])
                for row in conn.execute(
                    "SELECT column_name FROM information_schema.columns WHERE table_name = 'medals'"
                ).fetchall()
            }
            selected_col = next((col for col in columns if col in available_columns), None)
            if not selected_col:
                return None
            row = conn.execute(
                f"SELECT {selected_col} FROM medals WHERE medal_name_id = ?",
                [medal_name_id],
            ).fetchone()
            if row and row[0]:
                return str(row[0]).strip() or None
    except Exception:
        pass
    return None


def _resolve_from_db(medal_name_id: int, lang: str) -> str | None:
    """Résolution du nom depuis metadata.duckdb."""
    columns = ["name_fr", "name_en"] if lang == "fr" else ["name_en", "name_fr"]
    return _resolve_text_from_db(medal_name_id, lang, columns)


def resolve_medal_description(medal_name_id: int, lang: str = "fr") -> str | None:
    """Résout une description de médaille si disponible.

    Cherche d'abord dans metadata.duckdb puis dans les fichiers JSON statiques.
    """
    primary = (
        ["description_fr", "desc_fr", "blurb_fr", "description_en", "desc_en", "blurb_en"]
        if lang == "fr"
        else ["description_en", "desc_en", "blurb_en", "description_fr", "desc_fr", "blurb_fr"]
    )
    result = _resolve_text_from_db(medal_name_id, lang, primary)
    if result:
        return result
    # Fallback JSON
    fallback_lang = lang if lang in ("fr", "en") else "en"
    desc_map = _load_description_map(fallback_lang)
    desc = desc_map.get(medal_name_id)
    if desc:
        return desc
    # Dernier fallback : langue opposée
    other = "en" if fallback_lang == "fr" else "fr"
    return _load_description_map(other).get(medal_name_id)


def resolve_medal_name(medal_name_id: int, lang: str = "fr") -> str:
    """Résout un medal_name_id en nom lisible.

    Args:
        medal_name_id: Identifiant numérique de la médaille.
        lang: Langue cible (``"fr"`` ou ``"en"``).

    Returns:
        Label lisible ou ``str(medal_name_id)`` si non résolu.
    """
    name = _resolve_from_db(medal_name_id, lang)
    if name:
        return name

    name = _load_json_map(lang).get(medal_name_id)
    if name:
        return name

    if lang != "en":
        name = _load_json_map("en").get(medal_name_id)
        if name:
            return name

    return str(medal_name_id)


__all__ = ["resolve_medal_description", "resolve_medal_name"]
