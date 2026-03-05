"""Helper pour gérer les Career Ranks Halo Infinite.

Ce module charge les métadonnées des rangs depuis metadata.duckdb
et fournit des fonctions pour afficher le rang d'un joueur.

Données requises:
- data/warehouse/metadata.duckdb (table career_ranks, 272 rangs)
- data/cache/career_ranks/ (icônes PNG optionnelles)

Usage:
    from src.ui.career_ranks import get_rank_info, get_rank_icon_path

    # Si on connaît le numéro de rang du joueur (1-272):
    info = get_rank_info(150)
    print(f"{info.full_label_fr}: {info.xp_required} XP")

    # Récupérer le chemin local de l'icône:
    icon_path = get_rank_icon_path(150)
    if icon_path:
        st.image(str(icon_path), width=64)
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from functools import lru_cache
from pathlib import Path

from src.ui.i18n.data_labels import load_domain
from src.utils.paths import REPO_ROOT

# Legacy dicts conservés comme fallback — source de vérité = JSON
_CAREER_RANK_TIER_FR: dict[str, str] = {
    "Bronze": "Bronze",
    "Silver": "Argent",
    "Gold": "Or",
    "Platinum": "Platine",
    "Diamond": "Diamant",
    "Onyx": "Onyx",
}


_CAREER_RANK_TITLE_FR: dict[str, str] = {
    "Recruit": "Recrue",
    "Cadet": "Cadet",
    "Private": "Soldat",
    "Lance Corporal": "Caporal suppléant",
    "Corporal": "Caporal",
    "Sergeant": "Sergent",
    "Staff Sergeant": "Sergent-chef",
    "Gunnery Sergeant": "Sergent d'artillerie",
    "Master Sergeant": "Adjudant",
    "Lieutenant": "Lieutenant",
    "Captain": "Capitaine",
    "Major": "Lieutenant-major",
    "Lt Colonel": "Lieutenant-colonel",
    "Colonel": "Colonel",
    "Brigadier General": "Général de brigade",
    "General": "Général",
    "Hero": "Héros",
}


_GRADE_TO_ROMAN: dict[str, str] = {
    "1": "I",
    "2": "II",
    "3": "III",
}


def _get_rank_translations(lang: str) -> tuple[dict[str, str], dict[str, str], dict[str, str]]:
    """Charge les traductions de rangs depuis les JSON i18n.

    Returns:
        (titles, tiers, grades) — dicts de traduction.
    """
    data = load_domain("ranks", lang)
    titles = data.get("_titles", {})
    tiers = data.get("_tiers", {})
    grades = data.get("_grades", {})
    if not titles:
        # Fallback sur les dicts legacy FR
        titles = _CAREER_RANK_TITLE_FR
        tiers = _CAREER_RANK_TIER_FR
        grades = _GRADE_TO_ROMAN
    return titles, tiers, grades


def format_career_rank_label(
    *, tier: str | None, title: str | None, grade: str | None, lang: str = "fr"
) -> str:
    """Formate un libellé de rang Career dans la langue demandée.

    Args:
        tier: Tier/type de rang (ex: "Silver", "Gold")
        title: Titre du rang (ex: "Private", "Lt Colonel")
        grade: Sous-grade ("1"/"2"/"3") ou None
        lang: Code langue ("fr", "en").

    Returns:
        Libellé localisé (ex: "Soldat - Argent II", "Private - Silver II").
    """
    titles, tiers, grades = _get_rank_translations(lang)

    raw_title = (title or "").strip()
    raw_tier = (tier or "").strip()
    raw_grade = (grade or "").strip()

    title_loc = titles.get(raw_title, raw_title)
    tier_loc = tiers.get(raw_tier, raw_tier)

    # Cas spéciaux: grade initial et grade final
    # Recruit/Recrue et Hero/Héros n'ont pas de tier
    if raw_title in ("Recruit", "Hero"):
        return title_loc

    # Format: "Titre - Tier Grade" (ex: "Général de brigade - Or III")
    if title_loc and tier_loc and raw_grade:
        grade_roman = grades.get(raw_grade, raw_grade)
        return f"{title_loc} - {tier_loc} {grade_roman}"
    elif title_loc and tier_loc:
        return f"{title_loc} - {tier_loc}"
    elif title_loc:
        return title_loc
    return ""


def format_career_rank_label_fr(*, tier: str | None, title: str | None, grade: str | None) -> str:
    """Formate un libellé de rang Career en français (compatibilité).

    Wrapper autour de ``format_career_rank_label(lang="fr")``.
    """
    return format_career_rank_label(tier=tier, title=title, grade=grade, lang="fr")


@dataclass(frozen=True)
class CareerRankInfo:
    """Informations sur un rang Career."""

    rank_number: int
    title: str
    subtitle: str | None
    tier: str | None
    xp_required: int
    icon_path_remote: str  # Chemin relatif pour l'API CMS

    @property
    def full_label(self) -> str:
        """Retourne le label complet du rang EN (ex: 'Gold Lance Corporal 3')."""
        parts = []
        if self.subtitle:
            parts.append(self.subtitle)
        parts.append(self.title)
        if self.tier:
            parts.append(self.tier)
        return " ".join(parts)

    @property
    def full_label_fr(self) -> str:
        """Retourne le label complet du rang en français (compatibilité)."""
        return format_career_rank_label(
            tier=self.subtitle, title=self.title, grade=self.tier, lang="fr"
        )

    def full_label_localized(self, lang: str = "fr") -> str:
        """Retourne le label complet du rang dans la langue demandée."""
        return format_career_rank_label(
            tier=self.subtitle, title=self.title, grade=self.tier, lang=lang
        )

    @property
    def display_label(self) -> str:
        """Retourne un label compact EN (ex: 'Lance Corporal Gold 3')."""
        if self.subtitle and self.tier:
            return f"{self.title} {self.subtitle} {self.tier}"
        elif self.subtitle:
            return f"{self.title} {self.subtitle}"
        return self.title

    @property
    def display_label_fr(self) -> str:
        """Retourne un label compact en français (compatibilité)."""
        return format_career_rank_label(
            tier=self.subtitle, title=self.title, grade=self.tier, lang="fr"
        )

    def display_label_localized(self, lang: str = "fr") -> str:
        """Retourne un label compact dans la langue demandée."""
        return format_career_rank_label(
            tier=self.subtitle, title=self.title, grade=self.tier, lang=lang
        )


def _get_icons_dir() -> Path:
    return REPO_ROOT / "data" / "cache" / "career_ranks"


logger = logging.getLogger(__name__)


@lru_cache(maxsize=1)
def _build_ranks_lookup() -> dict[int, CareerRankInfo]:
    """Construit le lookup depuis metadata.duckdb (table career_ranks)."""
    from src.utils.db import duckdb_read_only

    db_path = REPO_ROOT / "data" / "warehouse" / "metadata.duckdb"
    if not db_path.exists():
        logger.warning("metadata.duckdb introuvable : %s", db_path)
        return {}

    with duckdb_read_only(str(db_path)) as conn:
        # Vérifier que la table existe
        from src.utils.db import has_table

        if not has_table(conn, "career_ranks"):
            logger.warning("Table career_ranks absente de metadata.duckdb")
            return {}

        rows = conn.execute(
            "SELECT rank_id, title_en, subtitle_en, tier, xp_required, large_icon_path "
            "FROM career_ranks ORDER BY rank_id"
        ).fetchall()

    lookup: dict[int, CareerRankInfo] = {}
    for rank_id, title, subtitle, tier, xp_req, large_icon in rows:
        lookup[rank_id] = CareerRankInfo(
            rank_number=rank_id,
            title=title,
            subtitle=subtitle or None,
            tier=tier or None,
            xp_required=xp_req,
            icon_path_remote=large_icon or "",
        )
    return lookup


def get_rank_info(rank_number: int) -> CareerRankInfo | None:
    """Retourne les informations d'un rang par son numéro (1-272).

    Args:
        rank_number: Numéro du rang (1 = Recruit, 272 = Hero)

    Returns:
        CareerRankInfo ou None si le rang n'existe pas
    """
    lookup = _build_ranks_lookup()
    return lookup.get(rank_number)


def get_all_ranks() -> list[CareerRankInfo]:
    """Retourne la liste de tous les rangs triés par numéro."""
    lookup = _build_ranks_lookup()
    return sorted(lookup.values(), key=lambda r: r.rank_number)


def get_rank_icon_path(rank_number: int) -> Path | None:
    """Retourne le chemin local de l'icône d'un rang si elle existe.

    Args:
        rank_number: Numéro du rang (1-272)

    Returns:
        Path vers le fichier PNG ou None si non téléchargé
    """
    icons_dir = _get_icons_dir()
    icon_path = icons_dir / f"rank_{rank_number:03d}_large.png"

    if icon_path.exists():
        return icon_path
    return None


def get_rank_icon_url(rank_number: int) -> str | None:
    """Retourne l'URL CMS pour télécharger l'icône d'un rang.

    Args:
        rank_number: Numéro du rang (1-272)

    Returns:
        URL complète ou None si le rang n'existe pas
    """
    info = get_rank_info(rank_number)
    if not info or not info.icon_path_remote:
        return None

    return f"https://gamecms-hacs.svc.halowaypoint.com/hi/images/file/{info.icon_path_remote}"


def get_rank_for_xp(total_xp: int) -> CareerRankInfo | None:
    """Détermine le rang correspondant à un total d'XP.

    Args:
        total_xp: Total d'XP Career du joueur

    Returns:
        Le rang le plus élevé atteint avec cet XP
    """
    ranks = get_all_ranks()

    current_rank = None
    for rank in ranks:
        if total_xp >= rank.xp_required:
            current_rank = rank
        else:
            break

    return current_rank


def count_cached_icons() -> int:
    """Compte le nombre d'icônes téléchargées en local."""
    icons_dir = _get_icons_dir()
    if not icons_dir.exists():
        return 0
    return len(list(icons_dir.glob("rank_*_large.png")))


@lru_cache(maxsize=1)
def is_metadata_available() -> bool:
    """Vérifie si les métadonnées des rangs sont disponibles dans metadata.duckdb."""
    from src.utils.db import duckdb_read_only

    db_path = REPO_ROOT / "data" / "warehouse" / "metadata.duckdb"
    if not db_path.exists():
        return False
    try:
        with duckdb_read_only(str(db_path)) as conn:
            from src.utils.db import has_table

            return has_table(conn, "career_ranks")
    except Exception:
        return False
