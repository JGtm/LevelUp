"""Enums de référence pour Halo Infinite basés sur SPNKr refdata.py.

Ce module fournit les enums officiels de l'API Halo Infinite :
- GameVariantCategory : Catégories de modes de jeu
- PersonalScoreNameId : Décomposition du score personnel
- Outcome : Résultats de match

Source : https://github.com/acurtis166/SPNKr/blob/master/spnkr/models/refdata.py

PersonalScoreNameId et données associées sont dans ``_refdata_personal_scores.py``
et ré-exportés ici pour compatibilité.
"""

from enum import IntEnum
from typing import Final

# Re-export PersonalScoreNameId et données associées
from src.data.domain._refdata_personal_scores import (  # noqa: F401
    PERSONAL_SCORE_DISPLAY_NAMES,
    PERSONAL_SCORE_POINTS,
    PERSONAL_SCORE_TECHNICAL_IDS,
    PersonalScoreNameId,
    get_personal_score_display_name,
    get_personal_score_points,
    get_personal_score_technical_id,
)


class GameVariantCategory(IntEnum):
    """Catégories officielles de modes de jeu Halo Infinite."""

    UNKNOWN = -1
    NONE = 0
    CAMPAIGN = 1
    FORGE = 2
    ACADEMY = 3
    ACADEMY_TUTORIAL = 4
    ACADEMY_PRACTICE = 5
    MULTIPLAYER_SLAYER = 6
    MULTIPLAYER_ATTRITION = 7
    MULTIPLAYER_ELIMINATION = 8
    MULTIPLAYER_FIESTA = 9
    MULTIPLAYER_SWAT = 10
    MULTIPLAYER_STRONGHOLDS = 11
    MULTIPLAYER_BASTION = 12
    MULTIPLAYER_KING_OF_THE_HILL = 13
    MULTIPLAYER_TOTAL_CONTROL = 14
    MULTIPLAYER_CTF = 15
    MULTIPLAYER_ASSAULT = 16
    MULTIPLAYER_EXTRACTION = 17
    MULTIPLAYER_ODDBALL = 18
    MULTIPLAYER_STOCKPILE = 19
    MULTIPLAYER_JUGGERNAUT = 20
    MULTIPLAYER_REGICIDE = 21
    MULTIPLAYER_INFECTION = 22
    MULTIPLAYER_ESCORT = 23
    MULTIPLAYER_GUN_GAME = 24
    MULTIPLAYER_GRIFBALL = 25
    MULTIPLAYER_RACE = 26
    MULTIPLAYER_PROTOTYPE = 27
    TEST = 28
    TEST_ACADEMY = 29
    TEST_AUDIO = 30
    TEST_CAMPAIGN = 31
    TEST_ENGINE = 32
    TEST_FORGE = 33
    TEST_GRAPHICS = 34
    TEST_MULTIPLAYER = 35
    TEST_SANDBOX = 36
    ACADEMY_TRAINING = 37
    ACADEMY_WEAPON_DRILL = 38
    MULTIPLAYER_LAND_GRAB = 39
    MULTIPLAYER_MINIGAME = 41
    MULTIPLAYER_FIREFIGHT = 42


class Outcome(IntEnum):
    """Résultats possibles d'un match."""

    UNKNOWN = -1
    NONE = 0
    TIE = 1
    WIN = 2
    LOSS = 3
    DID_NOT_FINISH = 4
    DID_NOT_START = 5


# =============================================================================
# Mappings vers le français
# =============================================================================

CATEGORY_TO_FR: Final[dict[int, str]] = {
    GameVariantCategory.UNKNOWN: "Inconnu",
    GameVariantCategory.NONE: "Aucun",
    GameVariantCategory.CAMPAIGN: "Campagne",
    GameVariantCategory.FORGE: "Forge",
    GameVariantCategory.ACADEMY: "Académie",
    GameVariantCategory.ACADEMY_TUTORIAL: "Tutoriel",
    GameVariantCategory.ACADEMY_PRACTICE: "Entraînement",
    GameVariantCategory.MULTIPLAYER_SLAYER: "Assassin",
    GameVariantCategory.MULTIPLAYER_ATTRITION: "Attrition",
    GameVariantCategory.MULTIPLAYER_ELIMINATION: "Élimination",
    GameVariantCategory.MULTIPLAYER_FIESTA: "Fiesta",
    GameVariantCategory.MULTIPLAYER_SWAT: "SWAT",
    GameVariantCategory.MULTIPLAYER_STRONGHOLDS: "Bastions",
    GameVariantCategory.MULTIPLAYER_BASTION: "Bastion",
    GameVariantCategory.MULTIPLAYER_KING_OF_THE_HILL: "Roi de la colline",
    GameVariantCategory.MULTIPLAYER_TOTAL_CONTROL: "Contrôle total",
    GameVariantCategory.MULTIPLAYER_CTF: "Capture de drapeau",
    GameVariantCategory.MULTIPLAYER_ASSAULT: "Assaut",
    GameVariantCategory.MULTIPLAYER_EXTRACTION: "Extraction",
    GameVariantCategory.MULTIPLAYER_ODDBALL: "Balle",
    GameVariantCategory.MULTIPLAYER_STOCKPILE: "Stockpile",
    GameVariantCategory.MULTIPLAYER_JUGGERNAUT: "Juggernaut",
    GameVariantCategory.MULTIPLAYER_REGICIDE: "Régicide",
    GameVariantCategory.MULTIPLAYER_INFECTION: "Infection",
    GameVariantCategory.MULTIPLAYER_ESCORT: "Escorte",
    GameVariantCategory.MULTIPLAYER_GUN_GAME: "Gun Game",
    GameVariantCategory.MULTIPLAYER_GRIFBALL: "Grifball",
    GameVariantCategory.MULTIPLAYER_RACE: "Course",
    GameVariantCategory.MULTIPLAYER_PROTOTYPE: "Prototype",
    GameVariantCategory.MULTIPLAYER_LAND_GRAB: "Land Grab",
    GameVariantCategory.MULTIPLAYER_MINIGAME: "Mini-jeu",
    GameVariantCategory.MULTIPLAYER_FIREFIGHT: "Firefight",
}

# =============================================================================
# Sets pour regroupement par type
# =============================================================================

# Scores liés aux objectifs (modes à objectifs)
OBJECTIVE_SCORES: Final[frozenset[int]] = frozenset(
    {
        # CTF
        PersonalScoreNameId.FLAG_CAPTURED,
        PersonalScoreNameId.FLAG_STOLEN,
        PersonalScoreNameId.FLAG_RETURNED,
        PersonalScoreNameId.FLAG_TAKEN,
        PersonalScoreNameId.FLAG_CAPTURE_ASSIST,
        PersonalScoreNameId.RUNNER_STOPPED,
        # Oddball
        PersonalScoreNameId.BALL_CONTROL,
        PersonalScoreNameId.BALL_TAKEN,
        PersonalScoreNameId.CARRIER_STOPPED,
        # KOTH
        PersonalScoreNameId.HILL_CONTROL,
        PersonalScoreNameId.HILL_SCORED,
        # Zones
        PersonalScoreNameId.ZONE_CAPTURED_50,
        PersonalScoreNameId.ZONE_CAPTURED_75,
        PersonalScoreNameId.ZONE_CAPTURED_100,
        PersonalScoreNameId.ZONE_SECURED,
        # Stockpile
        PersonalScoreNameId.POWER_SEED_SECURED,
        PersonalScoreNameId.POWER_SEED_STOLEN,
        PersonalScoreNameId.STOCKPILE_SCORED,
        # Extraction
        PersonalScoreNameId.EXTRACTION_INITIATED,
        PersonalScoreNameId.EXTRACTION_CONVERTED,
        PersonalScoreNameId.EXTRACTION_COMPLETED,
        PersonalScoreNameId.EXTRACTION_DENIED,
        # Hacking
        PersonalScoreNameId.HACKED_TERMINAL,
    }
)

# Scores liés aux assistances
ASSIST_SCORES: Final[frozenset[int]] = frozenset(
    {
        PersonalScoreNameId.KILL_ASSIST,
        PersonalScoreNameId.MARK_ASSIST,
        PersonalScoreNameId.SENSOR_ASSIST,
        PersonalScoreNameId.EMP_ASSIST,
        PersonalScoreNameId.DRIVER_ASSIST,
        PersonalScoreNameId.FLAG_CAPTURE_ASSIST,
    }
)

# Scores liés aux kills directs
KILL_SCORES: Final[frozenset[int]] = frozenset(
    {
        PersonalScoreNameId.KILLED_PLAYER,
        PersonalScoreNameId.ELIMINATED_PLAYER,
        PersonalScoreNameId.CARRIER_KILLED,
    }
)

# Scores négatifs (trahisons, suicides)
NEGATIVE_SCORES: Final[frozenset[int]] = frozenset(
    {
        PersonalScoreNameId.BETRAYED_PLAYER,
        PersonalScoreNameId.SELF_DESTRUCTION,
    }
)

# Alias pour compatibilité
PENALTY_SCORES = NEGATIVE_SCORES

# Scores liés aux véhicules
VEHICLE_DESTRUCTION_SCORES: Final[frozenset[int]] = frozenset(
    {
        PersonalScoreNameId.DESTROYED_BANSHEE,
        PersonalScoreNameId.DESTROYED_CHOPPER,
        PersonalScoreNameId.DESTROYED_FALCON,
        PersonalScoreNameId.DESTROYED_GHOST,
        PersonalScoreNameId.DESTROYED_GUNGOOSE,
        PersonalScoreNameId.DESTROYED_MONGOOSE,
        PersonalScoreNameId.DESTROYED_PHANTOM,
        PersonalScoreNameId.DESTROYED_RAZORBACK,
        PersonalScoreNameId.DESTROYED_ROCKET_WARTHOG,
        PersonalScoreNameId.DESTROYED_SCORPION,
        PersonalScoreNameId.DESTROYED_WARTHOG,
        PersonalScoreNameId.DESTROYED_WASP,
        PersonalScoreNameId.DESTROYED_WRAITH,
    }
)

VEHICLE_HIJACK_SCORES: Final[frozenset[int]] = frozenset(
    {
        PersonalScoreNameId.HIJACKED_BANSHEE,
        PersonalScoreNameId.HIJACKED_CHOPPER,
        PersonalScoreNameId.HIJACKED_FALCON,
        PersonalScoreNameId.HIJACKED_GHOST,
        PersonalScoreNameId.HIJACKED_GUNGOOSE,
        PersonalScoreNameId.HIJACKED_MONGOOSE,
        PersonalScoreNameId.HIJACKED_RAZORBACK,
        PersonalScoreNameId.HIJACKED_ROCKET_WARTHOG,
        PersonalScoreNameId.HIJACKED_WARTHOG,
        PersonalScoreNameId.HIJACKED_WASP,
    }
)

# =============================================================================
# Catégories de modes (groupements)
# =============================================================================

# Modes à objectifs (non-slayer)
OBJECTIVE_MODE_CATEGORIES: Final[frozenset[int]] = frozenset(
    {
        GameVariantCategory.MULTIPLAYER_CTF,
        GameVariantCategory.MULTIPLAYER_ODDBALL,
        GameVariantCategory.MULTIPLAYER_STRONGHOLDS,
        GameVariantCategory.MULTIPLAYER_KING_OF_THE_HILL,
        GameVariantCategory.MULTIPLAYER_TOTAL_CONTROL,
        GameVariantCategory.MULTIPLAYER_STOCKPILE,
        GameVariantCategory.MULTIPLAYER_ASSAULT,
        GameVariantCategory.MULTIPLAYER_EXTRACTION,
        GameVariantCategory.MULTIPLAYER_LAND_GRAB,
        GameVariantCategory.MULTIPLAYER_GRIFBALL,
    }
)

# Modes "Slayer" et variantes
SLAYER_MODE_CATEGORIES: Final[frozenset[int]] = frozenset(
    {
        GameVariantCategory.MULTIPLAYER_SLAYER,
        GameVariantCategory.MULTIPLAYER_ATTRITION,
        GameVariantCategory.MULTIPLAYER_ELIMINATION,
        GameVariantCategory.MULTIPLAYER_FIESTA,
        GameVariantCategory.MULTIPLAYER_SWAT,
        GameVariantCategory.MULTIPLAYER_GUN_GAME,
    }
)

# Modes spéciaux
SPECIAL_MODE_CATEGORIES: Final[frozenset[int]] = frozenset(
    {
        GameVariantCategory.MULTIPLAYER_INFECTION,
        GameVariantCategory.MULTIPLAYER_JUGGERNAUT,
        GameVariantCategory.MULTIPLAYER_REGICIDE,
        GameVariantCategory.MULTIPLAYER_FIREFIGHT,
        GameVariantCategory.MULTIPLAYER_RACE,
        GameVariantCategory.MULTIPLAYER_MINIGAME,
    }
)


# =============================================================================
# Fonctions utilitaires
# =============================================================================


def get_category_name_fr(category: int | GameVariantCategory) -> str:
    """Retourne le nom français d'une catégorie de mode.

    Args:
        category: Valeur de GameVariantCategory (int ou enum).

    Returns:
        Nom français de la catégorie, ou "Autre" si non trouvé.
    """
    cat_value = int(category) if isinstance(category, GameVariantCategory) else category
    return CATEGORY_TO_FR.get(cat_value, "Autre")


def is_objective_score(name_id: int | PersonalScoreNameId) -> bool:
    """Vérifie si un score est lié aux objectifs.

    Args:
        name_id: Valeur de PersonalScoreNameId.

    Returns:
        True si le score est lié aux objectifs.
    """
    id_value = int(name_id) if isinstance(name_id, PersonalScoreNameId) else name_id
    return id_value in OBJECTIVE_SCORES


def is_assist_score(name_id: int | PersonalScoreNameId) -> bool:
    """Vérifie si un score est une assistance.

    Args:
        name_id: Valeur de PersonalScoreNameId.

    Returns:
        True si le score est une assistance.
    """
    id_value = int(name_id) if isinstance(name_id, PersonalScoreNameId) else name_id
    return id_value in ASSIST_SCORES


def is_objective_mode(category: int | GameVariantCategory) -> bool:
    """Vérifie si une catégorie est un mode à objectifs.

    Args:
        category: Valeur de GameVariantCategory.

    Returns:
        True si c'est un mode à objectifs.
    """
    cat_value = int(category) if isinstance(category, GameVariantCategory) else category
    return cat_value in OBJECTIVE_MODE_CATEGORIES


def is_slayer_mode(category: int | GameVariantCategory) -> bool:
    """Vérifie si une catégorie est un mode Slayer.

    Args:
        category: Valeur de GameVariantCategory.

    Returns:
        True si c'est un mode Slayer ou variante.
    """
    cat_value = int(category) if isinstance(category, GameVariantCategory) else category
    return cat_value in SLAYER_MODE_CATEGORIES
