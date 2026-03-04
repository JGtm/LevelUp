"""Logique pure pour la section Historique des rencontres.

Module testable sans Streamlit : modèles Pydantic, calcul de badges,
filtrage des XUIDs cibles et utilitaires de formatage.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from pydantic import BaseModel, Field

from src.ui.i18n import t

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Modèles
# ---------------------------------------------------------------------------

_FRIENDS_JSON = Path(__file__).resolve().parents[3] / ".streamlit" / "friends_defaults.json"


@dataclass(frozen=True)
class Badge:
    """Badge d'information affiché en ligne sur le gamertag."""

    label_key: str  # clé i18n résolue dans la couche rendu
    css_class: str  # classe CSS os-sb-td--best | os-sb-td--worst | encounter-badge--amber
    tooltip: str  # texte français libre (dynamique, non traduit)


class EncounterStats(BaseModel):
    """Statistiques d'historique des rencontres pour un joueur."""

    xuid: str
    gamertag: str
    total_encounters: int = 0
    ally_count: int = 0
    enemy_count: int = 0
    winrate_as_ally: float | None = None
    winrate_vs_enemy: float | None = None
    kills_dealt: int = Field(default=0, ge=0)
    deaths_suffered: int = Field(default=0, ge=0)
    last_seen: datetime | None = None
    # Side dans le match courant (renseigné après construction)
    current_side: str = "ennemi"  # "allié" | "ennemi"

    model_config = {"arbitrary_types_allowed": True}


# ---------------------------------------------------------------------------
# Utilitaires
# ---------------------------------------------------------------------------


def ordinal_fr(n: int) -> str:
    """Retourne l'ordinal français pour un entier positif.

    Examples:
        ordinal_fr(1) == "1ère"
        ordinal_fr(2) == "2e"
        ordinal_fr(12) == "12e"
    """
    if n <= 0:
        return str(n)
    return "1ère" if n == 1 else f"{n}e"


def _relative_date_fr(dt: datetime) -> str:
    """Formate une datetime en texte relatif français approximatif."""
    now = datetime.now(tz=timezone.utc)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    delta = now - dt
    days = delta.days
    if days < 0:
        return "à venir"
    if days == 0:
        return "aujourd'hui"
    if days == 1:
        return "hier"
    if days < 7:
        return f"il y a {days} j"
    if days < 30:
        weeks = days // 7
        return f"il y a {weeks} sem."
    if days < 365:
        months = days // 30
        return f"il y a {months} mois"
    years = days // 365
    return f"il y a {years} an{'s' if years > 1 else ''}"


# ---------------------------------------------------------------------------
# Chargement des amis
# ---------------------------------------------------------------------------


def _load_friends_from_json(self_xuid: str) -> set[str]:
    """Charge les XUIDs des amis depuis friends_defaults.json.

    Args:
        self_xuid: XUID du joueur principal (clé dans le JSON).

    Returns:
        Set de gamertags/XUIDs déclarés comme amis. Vide si absent.
    """
    if not _FRIENDS_JSON.exists():
        return set()
    try:
        with open(_FRIENDS_JSON, encoding="utf-8") as fh:
            data = json.load(fh)
    except Exception:
        logger.debug("Lecture friends_defaults.json échouée", exc_info=True)
        return set()

    raw = data.get(str(self_xuid).strip())
    if not isinstance(raw, list):
        return set()

    result: set[str] = set()
    for entry in raw:
        if isinstance(entry, str) and entry.strip() and not entry.startswith("~"):
            result.add(entry.strip())
    return result


def build_friends_set(
    self_xuid: str,
    friends_xuids_csv: str | None,
) -> set[str]:
    """Construit l'ensemble des XUIDs / gamertags amis pour ce match.

    Stratégie double source :
    1. Colonne `friends_xuids` de player_match_enrichment (CSV, si disponible).
    2. Fallback : friends_defaults.json.

    Args:
        self_xuid: XUID du joueur principal.
        friends_xuids_csv: Valeur CSV depuis player_match_enrichment.friends_xuids.

    Returns:
        Set de chaînes (XUIDs ou gamertags) à exclure du tableau.
    """
    if friends_xuids_csv and friends_xuids_csv.strip():
        parts = {p.strip() for p in friends_xuids_csv.split(",") if p.strip()}
        if parts:
            return parts

    return _load_friends_from_json(self_xuid)


def filter_encounter_xuids(
    players: list[dict[str, Any]],
    self_xuid: str,
    friends_set: set[str],
) -> list[str]:
    """Retourne les XUIDs à analyser dans le tableau des rencontres.

    Exclut le joueur principal et les amis connus du match.

    Args:
        players: Liste de dicts (champs xuid, gamertag) du scoreboard.
        self_xuid: XUID du joueur principal (toujours exclu).
        friends_set: Ensemble d'identifiants amis (XUIDs ou gamertags).

    Returns:
        Liste de XUIDs uniques à analyser. Peut être vide.
    """
    seen: set[str] = set()
    result: list[str] = []
    self_norm = str(self_xuid).strip()

    for p in players:
        xuid = str(p.get("xuid") or "").strip()
        gamertag = str(p.get("gamertag") or "").strip().casefold()

        if not xuid or xuid == self_norm:
            continue
        if xuid in seen:
            continue
        # Exclusion ami : via XUID direct ou gamertag (caseless)
        if xuid in friends_set or gamertag in {f.casefold() for f in friends_set}:
            continue

        seen.add(xuid)
        result.append(xuid)

    return result


# ---------------------------------------------------------------------------
# Calcul des badges
# ---------------------------------------------------------------------------


def compute_encounter_badges(stats: EncounterStats) -> list[Badge]:
    """Calcule les badges automatiques pour un joueur rencontré.

    Conditions :
    - Dur à cuire : ratio deaths/kills > 2.0 ET deaths_suffered >= 3
    - Allié+     : winrate_as_ally >= 0.65 ET ally_count >= 2
    - Coriace    : winrate_vs_enemy <= 0.35 ET enemy_count >= 3

    Args:
        stats: Données d'historique pour ce joueur.

    Returns:
        Liste de badges applicables (peut être vide).
    """
    badges: list[Badge] = []

    # Dur à cuire
    if (
        stats.deaths_suffered >= 3
        and stats.kills_dealt > 0
        and (stats.deaths_suffered / stats.kills_dealt) > 2.0
    ):
        badges.append(
            Badge(
                label_key="badge_tough_nut",
                css_class="os-sb-td--worst",
                tooltip=t(
                    "tooltip_tough_nut_kd", killed=stats.deaths_suffered, kills=stats.kills_dealt
                ),
            )
        )
    elif stats.deaths_suffered >= 3 and stats.kills_dealt == 0:
        badges.append(
            Badge(
                label_key="badge_tough_nut",
                css_class="os-sb-td--worst",
                tooltip=t("tooltip_tough_nut_zero", killed=stats.deaths_suffered),
            )
        )

    # Allié+
    if (
        stats.winrate_as_ally is not None
        and stats.winrate_as_ally >= 0.65
        and stats.ally_count >= 2
    ):
        pct = round(stats.winrate_as_ally * 100)
        badges.append(
            Badge(
                label_key="badge_ally_plus",
                css_class="os-sb-td--best",
                tooltip=t("tooltip_ally_plus", pct=pct, count=stats.ally_count),
            )
        )

    # Adversaire coriace
    if (
        stats.winrate_vs_enemy is not None
        and stats.winrate_vs_enemy <= 0.35
        and stats.enemy_count >= 3
    ):
        pct = round(stats.winrate_vs_enemy * 100)
        badges.append(
            Badge(
                label_key="badge_coriace",
                css_class="encounter-badge--amber",
                tooltip=t("tooltip_coriace", pct=pct, count=stats.enemy_count),
            )
        )

    return badges
