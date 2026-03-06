"""PersonalScoreNameId et données associées (points, groupements, fonctions).

Extrait de ``refdata.py`` pour respecter le seuil de 500 lignes.
"""

from enum import IntEnum
from typing import Final


class PersonalScoreNameId(IntEnum):
    """Types de scores personnels avec leurs valeurs de points associées.

    Les noms sont ceux de l'API. Utilisez `display_name()` pour le nom affiché.
    Les points associés sont documentés en commentaire.
    """

    # Combat
    KILLED_PLAYER = 1024030246  # 100 pts
    BETRAYED_PLAYER = 911992497  # -100 pts
    SELF_DESTRUCTION = 249491819  # -100 pts
    ELIMINATED_PLAYER = 2408971842  # 200 pts
    REVIVED_PLAYER = 3428202435  # 100 pts
    REVIVE_DENIED = 2130209372  # 25 pts

    # Assistances
    KILL_ASSIST = 638246808  # 50 pts
    MARK_ASSIST = 152718958  # 10 pts
    SENSOR_ASSIST = 1267013266  # 10 pts
    EMP_ASSIST = 221060588  # 50 pts
    DRIVER_ASSIST = 963594075  # 50 pts

    # CTF (Capture de drapeau)
    FLAG_CAPTURED = 601966503  # 300 pts
    FLAG_STOLEN = 3002710045  # 25 pts
    FLAG_RETURNED = 22113181  # 25 pts
    FLAG_TAKEN = 2387185397  # 10 pts
    FLAG_CAPTURE_ASSIST = 555570945  # 100 pts
    RUNNER_STOPPED = 316828380  # 25 pts

    # Oddball
    BALL_CONTROL = 454168309  # 50 pts
    BALL_TAKEN = 204144695  # 10 pts
    CARRIER_STOPPED = 746397417  # 25 pts

    # King of the Hill
    HILL_CONTROL = 340198991  # 25 pts
    HILL_SCORED = 1032565232  # 100 pts

    # Strongholds / Zones
    ZONE_CAPTURED_50 = 3507884073  # 50 pts
    ZONE_CAPTURED_75 = 4026987576  # 75 pts
    ZONE_CAPTURED_100 = 757037588  # 100 pts
    ZONE_SECURED = 709346128  # 25 pts

    # Stockpile
    POWER_SEED_SECURED = 2188620691  # 100 pts
    POWER_SEED_STOLEN = 3996338664  # 50 pts
    CARRIER_KILLED = 4128329646  # 10 pts
    STOCKPILE_SCORED = 2801241965  # 150 pts

    # Extraction
    EXTRACTION_INITIATED = 1825517751  # 50 pts
    EXTRACTION_CONVERTED = 1117301492  # 50 pts
    EXTRACTION_COMPLETED = 4130011565  # 200 pts
    EXTRACTION_DENIED = 1552628741  # 25 pts

    # Infection
    CONVERSION_DENIED = 4247243561  # 25 pts

    # Last Spartan Standing
    COLLECTED_BONUS_XP = 522435689  # 300 pts

    # Hacking (Halo Infinite modes)
    HACKED_TERMINAL = 665081740  # 100 pts

    # Destruction de véhicules
    DESTROYED_BANSHEE = 597066859  # 50 pts
    DESTROYED_CHOPPER = 3472794399  # 50 pts
    DESTROYED_FALCON = 395875864  # 75 pts
    DESTROYED_GHOST = 4254982885  # 50 pts
    DESTROYED_GUNGOOSE = 2107631925  # 25 pts
    DESTROYED_MONGOOSE = 1416267372  # 25 pts
    DESTROYED_PHANTOM = 2742351765  # 100 pts
    DESTROYED_RAZORBACK = 1661163286  # 50 pts
    DESTROYED_ROCKET_WARTHOG = 2008690931  # 50 pts
    DESTROYED_SCORPION = 3454330054  # 100 pts
    DESTROYED_WARTHOG = 3107879375  # 50 pts
    DESTROYED_WASP = 2106274556  # 50 pts
    DESTROYED_WRAITH = 3243589708  # 100 pts

    # Hijacks (détournement de véhicules)
    HIJACKED_BANSHEE = 3150095814  # 25 pts
    HIJACKED_CHOPPER = 1059880024  # 25 pts
    HIJACKED_FALCON = 586857799  # 25 pts
    HIJACKED_GHOST = 1614285349  # 25 pts
    HIJACKED_GUNGOOSE = 4186766732  # 25 pts
    HIJACKED_MONGOOSE = 2191528998  # 25 pts
    HIJACKED_RAZORBACK = 2848565291  # 25 pts
    HIJACKED_ROCKET_WARTHOG = 4294405210  # 25 pts
    HIJACKED_WARTHOG = 1834653062  # 25 pts
    HIJACKED_WASP = 674964649  # 25 pts

    # Custom (scripts Forge)
    CUSTOM = 4294967295  # Variable


# =============================================================================
# Noms d'affichage FR (legacy)
# =============================================================================

PERSONAL_SCORE_DISPLAY_NAMES: Final[dict[int, str]] = {
    # Combat
    PersonalScoreNameId.KILLED_PLAYER: "Joueur tué",
    PersonalScoreNameId.BETRAYED_PLAYER: "Trahison",
    PersonalScoreNameId.SELF_DESTRUCTION: "Auto-destruction",
    PersonalScoreNameId.ELIMINATED_PLAYER: "Joueur éliminé",
    PersonalScoreNameId.REVIVED_PLAYER: "Joueur réanimé",
    PersonalScoreNameId.REVIVE_DENIED: "Réanimation empêchée",
    # Assistances
    PersonalScoreNameId.KILL_ASSIST: "Assistance kill",
    PersonalScoreNameId.MARK_ASSIST: "Assistance marquage",
    PersonalScoreNameId.SENSOR_ASSIST: "Assistance capteur",
    PersonalScoreNameId.EMP_ASSIST: "Assistance EMP",
    PersonalScoreNameId.DRIVER_ASSIST: "Assistance conducteur",
    # CTF
    PersonalScoreNameId.FLAG_CAPTURED: "Drapeau capturé",
    PersonalScoreNameId.FLAG_STOLEN: "Drapeau volé",
    PersonalScoreNameId.FLAG_RETURNED: "Drapeau ramené",
    PersonalScoreNameId.FLAG_TAKEN: "Drapeau pris",
    PersonalScoreNameId.FLAG_CAPTURE_ASSIST: "Assistance capture",
    PersonalScoreNameId.RUNNER_STOPPED: "Porteur arrêté",
    # Oddball
    PersonalScoreNameId.BALL_CONTROL: "Contrôle balle",
    PersonalScoreNameId.BALL_TAKEN: "Balle prise",
    PersonalScoreNameId.CARRIER_STOPPED: "Porteur arrêté",
    # KOTH
    PersonalScoreNameId.HILL_CONTROL: "Contrôle colline",
    PersonalScoreNameId.HILL_SCORED: "Points colline",
    # Zones
    PersonalScoreNameId.ZONE_CAPTURED_50: "Zone capturée",
    PersonalScoreNameId.ZONE_CAPTURED_75: "Zone capturée",
    PersonalScoreNameId.ZONE_CAPTURED_100: "Zone capturée",
    PersonalScoreNameId.ZONE_SECURED: "Zone sécurisée",
    # Stockpile
    PersonalScoreNameId.POWER_SEED_SECURED: "Graine sécurisée",
    PersonalScoreNameId.POWER_SEED_STOLEN: "Graine volée",
    PersonalScoreNameId.CARRIER_KILLED: "Porteur tué",
    PersonalScoreNameId.STOCKPILE_SCORED: "Stockpile marqué",
    # Extraction
    PersonalScoreNameId.EXTRACTION_INITIATED: "Extraction initiée",
    PersonalScoreNameId.EXTRACTION_CONVERTED: "Extraction convertie",
    PersonalScoreNameId.EXTRACTION_COMPLETED: "Extraction complète",
    PersonalScoreNameId.EXTRACTION_DENIED: "Extraction refusée",
    # Infection
    PersonalScoreNameId.CONVERSION_DENIED: "Conversion empêchée",
    # LSS
    PersonalScoreNameId.COLLECTED_BONUS_XP: "XP bonus collecté",
    # Hacking
    PersonalScoreNameId.HACKED_TERMINAL: "Terminal piraté",
    # Custom
    PersonalScoreNameId.CUSTOM: "Personnalisé",
}

# IDs techniques snake_case pour stockage / i18n (clés = PersonalScoreNameId).
# Couvre TOUS les IDs, y compris DESTROYED_*/HIJACKED_* absents de l'ancien dict.
PERSONAL_SCORE_TECHNICAL_IDS: Final[dict[int, str]] = {
    # Combat
    PersonalScoreNameId.KILLED_PLAYER: "killed_player",
    PersonalScoreNameId.BETRAYED_PLAYER: "betrayed_player",
    PersonalScoreNameId.SELF_DESTRUCTION: "self_destruction",
    PersonalScoreNameId.ELIMINATED_PLAYER: "eliminated_player",
    PersonalScoreNameId.REVIVED_PLAYER: "revived_player",
    PersonalScoreNameId.REVIVE_DENIED: "revive_denied",
    # Assistances
    PersonalScoreNameId.KILL_ASSIST: "kill_assist",
    PersonalScoreNameId.MARK_ASSIST: "mark_assist",
    PersonalScoreNameId.SENSOR_ASSIST: "sensor_assist",
    PersonalScoreNameId.EMP_ASSIST: "emp_assist",
    PersonalScoreNameId.DRIVER_ASSIST: "driver_assist",
    # CTF
    PersonalScoreNameId.FLAG_CAPTURED: "flag_captured",
    PersonalScoreNameId.FLAG_STOLEN: "flag_stolen",
    PersonalScoreNameId.FLAG_RETURNED: "flag_returned",
    PersonalScoreNameId.FLAG_TAKEN: "flag_taken",
    PersonalScoreNameId.FLAG_CAPTURE_ASSIST: "flag_capture_assist",
    PersonalScoreNameId.RUNNER_STOPPED: "runner_stopped",
    # Oddball
    PersonalScoreNameId.BALL_CONTROL: "ball_control",
    PersonalScoreNameId.BALL_TAKEN: "ball_taken",
    PersonalScoreNameId.CARRIER_STOPPED: "carrier_stopped",
    # KOTH
    PersonalScoreNameId.HILL_CONTROL: "hill_control",
    PersonalScoreNameId.HILL_SCORED: "hill_scored",
    # Zones (3 variantes → une seule clé technique)
    PersonalScoreNameId.ZONE_CAPTURED_50: "zone_captured",
    PersonalScoreNameId.ZONE_CAPTURED_75: "zone_captured",
    PersonalScoreNameId.ZONE_CAPTURED_100: "zone_captured",
    PersonalScoreNameId.ZONE_SECURED: "zone_secured",
    # Stockpile
    PersonalScoreNameId.POWER_SEED_SECURED: "power_seed_secured",
    PersonalScoreNameId.POWER_SEED_STOLEN: "power_seed_stolen",
    PersonalScoreNameId.CARRIER_KILLED: "carrier_killed",
    PersonalScoreNameId.STOCKPILE_SCORED: "stockpile_scored",
    # Extraction
    PersonalScoreNameId.EXTRACTION_INITIATED: "extraction_initiated",
    PersonalScoreNameId.EXTRACTION_CONVERTED: "extraction_converted",
    PersonalScoreNameId.EXTRACTION_COMPLETED: "extraction_completed",
    PersonalScoreNameId.EXTRACTION_DENIED: "extraction_denied",
    # Infection
    PersonalScoreNameId.CONVERSION_DENIED: "conversion_denied",
    # LSS
    PersonalScoreNameId.COLLECTED_BONUS_XP: "collected_bonus_xp",
    # Hacking
    PersonalScoreNameId.HACKED_TERMINAL: "hacked_terminal",
    # Destruction de véhicules
    PersonalScoreNameId.DESTROYED_BANSHEE: "destroyed_banshee",
    PersonalScoreNameId.DESTROYED_CHOPPER: "destroyed_chopper",
    PersonalScoreNameId.DESTROYED_FALCON: "destroyed_falcon",
    PersonalScoreNameId.DESTROYED_GHOST: "destroyed_ghost",
    PersonalScoreNameId.DESTROYED_GUNGOOSE: "destroyed_gungoose",
    PersonalScoreNameId.DESTROYED_MONGOOSE: "destroyed_mongoose",
    PersonalScoreNameId.DESTROYED_PHANTOM: "destroyed_phantom",
    PersonalScoreNameId.DESTROYED_RAZORBACK: "destroyed_razorback",
    PersonalScoreNameId.DESTROYED_ROCKET_WARTHOG: "destroyed_rocket_warthog",
    PersonalScoreNameId.DESTROYED_SCORPION: "destroyed_scorpion",
    PersonalScoreNameId.DESTROYED_WARTHOG: "destroyed_warthog",
    PersonalScoreNameId.DESTROYED_WASP: "destroyed_wasp",
    PersonalScoreNameId.DESTROYED_WRAITH: "destroyed_wraith",
    # Hijacks
    PersonalScoreNameId.HIJACKED_BANSHEE: "hijacked_banshee",
    PersonalScoreNameId.HIJACKED_CHOPPER: "hijacked_chopper",
    PersonalScoreNameId.HIJACKED_FALCON: "hijacked_falcon",
    PersonalScoreNameId.HIJACKED_GHOST: "hijacked_ghost",
    PersonalScoreNameId.HIJACKED_GUNGOOSE: "hijacked_gungoose",
    PersonalScoreNameId.HIJACKED_MONGOOSE: "hijacked_mongoose",
    PersonalScoreNameId.HIJACKED_RAZORBACK: "hijacked_razorback",
    PersonalScoreNameId.HIJACKED_ROCKET_WARTHOG: "hijacked_rocket_warthog",
    PersonalScoreNameId.HIJACKED_WARTHOG: "hijacked_warthog",
    PersonalScoreNameId.HIJACKED_WASP: "hijacked_wasp",
    # Custom
    PersonalScoreNameId.CUSTOM: "custom",
}

PERSONAL_SCORE_POINTS: Final[dict[int, int]] = {
    # Combat
    PersonalScoreNameId.KILLED_PLAYER: 100,
    PersonalScoreNameId.BETRAYED_PLAYER: -100,
    PersonalScoreNameId.SELF_DESTRUCTION: -100,
    PersonalScoreNameId.ELIMINATED_PLAYER: 200,
    PersonalScoreNameId.REVIVED_PLAYER: 100,
    PersonalScoreNameId.REVIVE_DENIED: 25,
    # Assistances
    PersonalScoreNameId.KILL_ASSIST: 50,
    PersonalScoreNameId.MARK_ASSIST: 10,
    PersonalScoreNameId.SENSOR_ASSIST: 10,
    PersonalScoreNameId.EMP_ASSIST: 50,
    PersonalScoreNameId.DRIVER_ASSIST: 50,
    # CTF
    PersonalScoreNameId.FLAG_CAPTURED: 300,
    PersonalScoreNameId.FLAG_STOLEN: 25,
    PersonalScoreNameId.FLAG_RETURNED: 25,
    PersonalScoreNameId.FLAG_TAKEN: 10,
    PersonalScoreNameId.FLAG_CAPTURE_ASSIST: 100,
    PersonalScoreNameId.RUNNER_STOPPED: 25,
    # Oddball
    PersonalScoreNameId.BALL_CONTROL: 50,
    PersonalScoreNameId.BALL_TAKEN: 10,
    PersonalScoreNameId.CARRIER_STOPPED: 25,
    # KOTH
    PersonalScoreNameId.HILL_CONTROL: 25,
    PersonalScoreNameId.HILL_SCORED: 100,
    # Zones
    PersonalScoreNameId.ZONE_CAPTURED_50: 50,
    PersonalScoreNameId.ZONE_CAPTURED_75: 75,
    PersonalScoreNameId.ZONE_CAPTURED_100: 100,
    PersonalScoreNameId.ZONE_SECURED: 25,
    # Stockpile
    PersonalScoreNameId.POWER_SEED_SECURED: 100,
    PersonalScoreNameId.POWER_SEED_STOLEN: 50,
    PersonalScoreNameId.CARRIER_KILLED: 10,
    PersonalScoreNameId.STOCKPILE_SCORED: 150,
    # Extraction
    PersonalScoreNameId.EXTRACTION_INITIATED: 50,
    PersonalScoreNameId.EXTRACTION_CONVERTED: 50,
    PersonalScoreNameId.EXTRACTION_COMPLETED: 200,
    PersonalScoreNameId.EXTRACTION_DENIED: 25,
    # Infection
    PersonalScoreNameId.CONVERSION_DENIED: 25,
    # LSS
    PersonalScoreNameId.COLLECTED_BONUS_XP: 300,
    # Hacking
    PersonalScoreNameId.HACKED_TERMINAL: 100,
    # Véhicules (25-100 pts selon le véhicule)
    PersonalScoreNameId.DESTROYED_BANSHEE: 50,
    PersonalScoreNameId.DESTROYED_CHOPPER: 50,
    PersonalScoreNameId.DESTROYED_FALCON: 75,
    PersonalScoreNameId.DESTROYED_GHOST: 50,
    PersonalScoreNameId.DESTROYED_GUNGOOSE: 25,
    PersonalScoreNameId.DESTROYED_MONGOOSE: 25,
    PersonalScoreNameId.DESTROYED_PHANTOM: 100,
    PersonalScoreNameId.DESTROYED_RAZORBACK: 50,
    PersonalScoreNameId.DESTROYED_ROCKET_WARTHOG: 50,
    PersonalScoreNameId.DESTROYED_SCORPION: 100,
    PersonalScoreNameId.DESTROYED_WARTHOG: 50,
    PersonalScoreNameId.DESTROYED_WASP: 50,
    PersonalScoreNameId.DESTROYED_WRAITH: 100,
    # Hijacks (tous 25 pts)
    PersonalScoreNameId.HIJACKED_BANSHEE: 25,
    PersonalScoreNameId.HIJACKED_CHOPPER: 25,
    PersonalScoreNameId.HIJACKED_FALCON: 25,
    PersonalScoreNameId.HIJACKED_GHOST: 25,
    PersonalScoreNameId.HIJACKED_GUNGOOSE: 25,
    PersonalScoreNameId.HIJACKED_MONGOOSE: 25,
    PersonalScoreNameId.HIJACKED_RAZORBACK: 25,
    PersonalScoreNameId.HIJACKED_ROCKET_WARTHOG: 25,
    PersonalScoreNameId.HIJACKED_WARTHOG: 25,
    PersonalScoreNameId.HIJACKED_WASP: 25,
    # Custom (variable, on met 0 par défaut)
    PersonalScoreNameId.CUSTOM: 0,
}


# =============================================================================
# Fonctions utilitaires — PersonalScore
# =============================================================================


def get_personal_score_display_name(name_id: int | PersonalScoreNameId) -> str:
    """Retourne le nom d'affichage français (legacy) d'un type de score personnel.

    .. deprecated:: Utiliser ``get_personal_score_technical_id()`` pour les nouvelles
       insertions en DB. Ce helper reste utile pour l'affichage FR direct.

    Args:
        name_id: Valeur de PersonalScoreNameId (int ou enum).

    Returns:
        Nom d'affichage français, ou "Score" si non trouvé.
    """
    id_value = int(name_id) if isinstance(name_id, PersonalScoreNameId) else name_id
    return PERSONAL_SCORE_DISPLAY_NAMES.get(id_value, "Score")


def get_personal_score_technical_id(name_id: int | PersonalScoreNameId) -> str:
    """Retourne l'identifiant technique snake_case d'un type de score personnel.

    Couvre TOUS les types y compris DESTROYED_*/HIJACKED_* (corrige le bug
    où ces types étaient stockés comme ``"Score"``).

    Args:
        name_id: Valeur de PersonalScoreNameId (int ou enum).

    Returns:
        Identifiant technique (ex: ``"killed_player"``, ``"destroyed_banshee"``),
        ou ``name_id.name.lower()`` si l'enum est connu, ``"score"`` sinon.
    """
    id_value = int(name_id) if isinstance(name_id, PersonalScoreNameId) else name_id
    tech_id = PERSONAL_SCORE_TECHNICAL_IDS.get(id_value)
    if tech_id is not None:
        return tech_id
    # Fallback : essayer le nom de l'enum
    try:
        return PersonalScoreNameId(id_value).name.lower()
    except ValueError:
        return "score"


def get_personal_score_points(name_id: int | PersonalScoreNameId) -> int:
    """Retourne les points associés à un type de score personnel.

    Args:
        name_id: Valeur de PersonalScoreNameId (int ou enum).

    Returns:
        Nombre de points, ou 0 si non trouvé.
    """
    id_value = int(name_id) if isinstance(name_id, PersonalScoreNameId) else name_id
    return PERSONAL_SCORE_POINTS.get(id_value, 0)
