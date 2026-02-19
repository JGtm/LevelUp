"""Constantes de bitmask pour le tracking granulaire des données backfillées.

Deux niveaux de bitmask :
- ``ParticipantBits`` : stocké dans ``match_participants.backfill_bits``
  Granularité par joueur × match. Indique quelles colonnes sont remplies.
- ``MatchBits`` : stocké dans ``match_registry.backfill_completed`` (bits ≥ 16)
  Granularité par match uniquement. Données globales au match.

Architecture :
    Les anciens BACKFILL_FLAGS (bits 0-15 de match_registry.backfill_completed)
    sont conservés dans ``migrations.py`` pour rétrocompatibilité.
    ``MatchBits`` utilise les bits ≥ 16 pour éviter toute collision.
"""

from __future__ import annotations


class ParticipantBits:
    """Bitmask pour ``match_participants.backfill_bits`` (données par joueur × match).

    Chaque bit indique qu'une ou plusieurs colonnes associées sont remplies
    (non-NULL) pour ce participant dans ce match.

    Usage :
        bits = ParticipantBits.TEAM_MMR | ParticipantBits.ENEMY_MMR
        conn.execute(
            "UPDATE match_participants SET backfill_bits = COALESCE(backfill_bits, 0) | ?"
            " WHERE match_id = ? AND xuid = ?",
            (bits, match_id, xuid),
        )
    """

    # ── Stats skill / MMR ────────────────────────────────────────────────────
    TEAM_MMR = 1 << 0  # 1       — team_mmr
    ENEMY_MMR = 1 << 1  # 2       — enemy_mmr
    KILLS_EXP = 1 << 2  # 4       — kills_expected, kills_stddev
    DEATHS_EXP = 1 << 3  # 8       — deaths_expected, deaths_stddev
    ASSISTS_EXP = 1 << 4  # 16      — assists_expected, assists_stddev

    # ── Stats de combat (get_match_stats) ───────────────────────────────────
    ACCURACY = 1 << 5  # 32      — accuracy
    SHOTS = 1 << 6  # 64      — shots_fired, shots_hit
    DAMAGE = 1 << 7  # 128     — damage_dealt, damage_taken
    AVG_LIFE = 1 << 8  # 256     — avg_life_seconds
    MEDALS = 1 << 9  # 512     — médailles dans medals_earned pour ce xuid

    # ── Kills détaillés ──────────────────────────────────────────────────────
    GRENADE_KILLS = 1 << 10  # 1024  — grenade_kills
    MELEE_KILLS = 1 << 11  # 2048  — melee_kills
    POWER_WEAPON = 1 << 12  # 4096  — power_weapon_kills
    PERSONAL_SCORE = 1 << 13  # 8192  — personal_score
    HEADSHOT_KILLS = 1 << 14  # 16384 — headshot_kills
    MAX_SPREE = 1 << 15  # 32768 — max_killing_spree

    # ── Stats calculées ──────────────────────────────────────────────────────
    KDA = 1 << 16  # 65536  — kda (calculé)
    TIME_PLAYED = 1 << 17  # 131072 — time_played_seconds

    # ── Killer/victim ────────────────────────────────────────────────────────
    KILLER_VICTIM = 1 << 18  # 262144 — ce joueur est présent dans killer_victim_pairs

    # ── Groupes (combinaisons logiques) ──────────────────────────────────────
    MMR = TEAM_MMR | ENEMY_MMR
    EXPECTED = KILLS_EXP | DEATHS_EXP | ASSISTS_EXP
    SKILL = MMR | EXPECTED
    COMBAT = ACCURACY | SHOTS | DAMAGE
    KILLS_DETAIL = GRENADE_KILLS | MELEE_KILLS | POWER_WEAPON | HEADSHOT_KILLS
    CORE_STATS = (
        ACCURACY
        | SHOTS
        | DAMAGE
        | AVG_LIFE
        | GRENADE_KILLS
        | MELEE_KILLS
        | POWER_WEAPON
        | PERSONAL_SCORE
        | HEADSHOT_KILLS
        | MAX_SPREE
        | KDA
        | TIME_PLAYED
    )
    ALL_STATS = SKILL | CORE_STATS | MEDALS | KILLER_VICTIM


class MatchBits:
    """Bitmask pour ``match_registry.backfill_completed`` (données par match).

    ⚠️ Compatibilité : Les bits 0-15 sont occupés par les anciens BACKFILL_FLAGS
    définis dans ``migrations.py``. MatchBits utilise uniquement les bits ≥ 16
    pour éviter toute collision pendant la période de transition.

    Anciens bits occupés (BACKFILL_FLAGS) :
        medals=1, events=2, skill=4, personal_scores=8, performance=16,
        accuracy=32, shots=64, enemy_mmr=128, assets=256, participants=512,
        participants_scores=1024, participants_kda=2048, participants_shots=4096,
        participants_damage=8192, aliases=16384, participants_avg_life=32768
    """

    EVENTS = 1 << 16  # 65536   — highlight_events chargés
    ASSETS = 1 << 17  # 131072  — map_name, playlist_name résolus
    ALIASES = 1 << 18  # 262144  — xuid_aliases extraits
    KILLER_VICTIM_LOADED = 1 << 19  # 524288 — killer_victim_pairs chargés (global)


def compute_participant_bits_from_data(data: dict) -> int:
    """Calcule le bitmask ``ParticipantBits`` depuis un dict de données participant.

    Vérifie chaque colonne clé : si non-NULL → le bit correspondant est activé.

    Args:
        data: Dict colonne → valeur (peut être None).

    Returns:
        Entier bitmask avec les bits des données présentes.
    """
    bits = 0
    if data.get("team_mmr") is not None:
        bits |= ParticipantBits.TEAM_MMR
    if data.get("enemy_mmr") is not None:
        bits |= ParticipantBits.ENEMY_MMR
    if data.get("kills_expected") is not None:
        bits |= ParticipantBits.KILLS_EXP
    if data.get("deaths_expected") is not None:
        bits |= ParticipantBits.DEATHS_EXP
    if data.get("assists_expected") is not None:
        bits |= ParticipantBits.ASSISTS_EXP
    if data.get("accuracy") is not None:
        bits |= ParticipantBits.ACCURACY
    if data.get("shots_fired") is not None:
        bits |= ParticipantBits.SHOTS
    if data.get("damage_dealt") is not None:
        bits |= ParticipantBits.DAMAGE
    if data.get("avg_life_seconds") is not None:
        bits |= ParticipantBits.AVG_LIFE
    if data.get("grenade_kills") is not None:
        bits |= ParticipantBits.GRENADE_KILLS
    if data.get("melee_kills") is not None:
        bits |= ParticipantBits.MELEE_KILLS
    if data.get("power_weapon_kills") is not None:
        bits |= ParticipantBits.POWER_WEAPON
    if data.get("personal_score") is not None:
        bits |= ParticipantBits.PERSONAL_SCORE
    if data.get("headshot_kills") is not None:
        bits |= ParticipantBits.HEADSHOT_KILLS
    if data.get("max_killing_spree") is not None:
        bits |= ParticipantBits.MAX_SPREE
    if data.get("kda") is not None:
        bits |= ParticipantBits.KDA
    if data.get("time_played_seconds") is not None:
        bits |= ParticipantBits.TIME_PLAYED
    return bits
