"""Détection des matchs avec données manquantes.

Supporte deux modes de détection :
- OR (défaut) : sélectionne les matchs manquant AU MOINS UNE donnée demandée
- AND (strict) : sélectionne les matchs manquant TOUTES les données demandées

V5 : La détection se fait via shared_matches.duckdb (match_registry + match_participants).
Le bitmask backfill_completed est stocké dans match_registry.

Note architecture (v5.2+) :
    ``find_matches_missing_data`` accepte ``scope: SyncScope`` qui remplace
    les 30+ kwargs individuels.  Les kwargs sont marqués ``LEGACY``.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING, Any

from src.data.sync.migrations import BACKFILL_FLAGS, compute_backfill_mask  # noqa: F401

if TYPE_CHECKING:
    from src.data.sync.scope import SyncScope

logger = logging.getLogger(__name__)


def _get_match_source(conn: Any) -> str:
    """Retourne 'v_match_full' si la vue est disponible, sinon 'match_registry'."""
    try:
        conn.execute("SELECT 1 FROM v_match_full LIMIT 1")
        return "v_match_full"
    except Exception:
        return "match_registry"


# ─────────────────────────────────────────────────────────────────────────────
# Re-exports pour rétro-compatibilité (BACKFILL_FLAGS, compute_backfill_mask)
# Définition canonique : src.data.sync.migrations
# ─────────────────────────────────────────────────────────────────────────────


# ─────────────────────────────────────────────────────────────────────────────
# Point d'entrée principal
# ─────────────────────────────────────────────────────────────────────────────


def find_matches_missing_data(
    conn: Any,
    xuid: str,
    *,
    shared_conn: Any,
    scope: SyncScope | None = None,
    # ── LEGACY kwargs (v5.1) ──────────────────────────────────────────────
    # TODO(cleanup): supprimer ces kwargs quand tous les appelants
    #   utilisent SyncScope.
    # ─────────────────────────────────────────────────────────────────────
    detection_mode: str = "or",
    medals: bool = False,
    events: bool = False,
    skill: bool = False,
    personal_scores: bool = False,
    performance_scores: bool = False,
    accuracy: bool = False,
    enemy_mmr: bool = False,
    assets: bool = False,
    participants: bool = False,
    participants_scores: bool = False,
    participants_kda: bool = False,
    participants_shots: bool = False,
    participants_damage: bool = False,
    participants_avg_life: bool = False,
    force_participants_shots: bool = False,
    force_participants_damage: bool = False,
    force_participants_avg_life: bool = False,
    force_medals: bool = False,
    force_accuracy: bool = False,
    shots: bool = False,
    force_shots: bool = False,
    force_skill: bool = False,
    force_performance_scores: bool = False,
    force_enemy_mmr: bool = False,
    force_aliases: bool = False,
    force_assets: bool = False,
    force_participants: bool = False,
    max_matches: int | None = None,
    all_data: bool = False,
) -> list[str]:
    """Trouve les matchs avec des données manquantes via shared DB (V5).

    .. deprecated:: v5.2
        Passer ``scope=SyncScope(...)`` au lieu des kwargs individuels.

    Args:
        conn: Connexion DuckDB (player DB).
        xuid: XUID du joueur.
        shared_conn: Connexion à shared_matches.duckdb (obligatoire).
        scope: Périmètre de données — **méthode recommandée** (SyncScope).
        detection_mode: (legacy) "or" (défaut) ou "and" (strict).
        max_matches: (legacy) Limite de résultats.
        all_data: (legacy) True si --all-data est actif.
        **kwargs: (legacy) 30+ flags booléens — voir SyncScope.

    Returns:
        Liste des match_id manquants.
    """
    # ── Extraire les valeurs depuis le scope si fourni ──
    if scope is not None:
        medals = scope.medals
        events = scope.events
        skill = scope.skill
        personal_scores = scope.personal_scores
        performance_scores = scope.performance_scores
        accuracy = scope.accuracy
        enemy_mmr = scope.enemy_mmr
        assets = scope.assets
        participants = scope.participants
        participants_scores = scope.participants_scores
        participants_kda = scope.participants_kda
        participants_shots = scope.participants_shots
        participants_damage = scope.participants_damage
        participants_avg_life = scope.participants_avg_life
        force_participants_shots = scope.force_participants_shots
        force_participants_damage = scope.force_participants_damage
        force_participants_avg_life = scope.force_participants_avg_life
        force_medals = scope.force_medals
        force_accuracy = scope.force_accuracy
        shots = scope.shots
        force_shots = scope.force_shots
        force_skill = scope.force_skill
        force_performance_scores = scope.force_performance_scores
        force_enemy_mmr = scope.force_enemy_mmr
        force_aliases = scope.force_aliases
        force_assets = scope.force_assets
        force_participants = scope.force_participants
        max_matches = scope.max_matches
        all_data = scope.all_data
    else:
        _ = all_data  # compat signature

    pve_stats = getattr(scope, "pve_stats", False) if scope is not None else False
    force_pve_stats = getattr(scope, "force_pve_stats", False) if scope is not None else False
    weapons = getattr(scope, "weapons", False) if scope is not None else False
    force_weapons = getattr(scope, "force_weapons", False) if scope is not None else False
    force_no_film = getattr(scope, "force_no_film", False) if scope is not None else False
    # --force-no-film implique --weapons et traite les matchs NO_FILM comme force_weapons
    if force_no_film:
        weapons = True
        force_weapons = True

    # Détecter le type de flags demandés
    local_data_requested = any(
        [
            medals,
            events,
            skill,
            personal_scores,
            performance_scores,
            accuracy,
            shots,
            enemy_mmr,
            assets,
            participants,
            pve_stats,
            weapons,
        ]
    )
    participants_data_requested = any(
        [
            participants_scores,
            participants_kda,
            participants_shots,
            participants_damage,
            participants_avg_life,
        ]
    )

    # ── Détection participants via shared DB ──
    shared_match_ids: list[str] = []
    if participants_data_requested:
        shared_match_ids = _find_matches_in_shared_db(
            shared_conn=shared_conn,
            xuid=xuid,
            max_matches=max_matches,
            participants_scores=participants_scores,
            participants_kda=participants_kda,
            participants_shots=participants_shots,
            participants_damage=participants_damage,
            participants_avg_life=participants_avg_life,
            force_participants_shots=force_participants_shots,
            force_participants_damage=force_participants_damage,
            force_participants_avg_life=force_participants_avg_life,
        )
        # Si aucun flag local → on retourne directement les résultats shared
        if not local_data_requested:
            return shared_match_ids

    # ── Détection données via shared DB (V5) ──
    local_match_ids = _find_matches_in_shared_all(
        shared_conn=shared_conn,
        conn=conn,
        xuid=xuid,
        max_matches=max_matches,
        medals=medals,
        events=events,
        skill=skill,
        personal_scores=personal_scores,
        performance_scores=performance_scores,
        accuracy=accuracy,
        shots=shots,
        enemy_mmr=enemy_mmr,
        assets=assets,
        participants=participants,
        pve_stats=pve_stats,
        force_medals=force_medals,
        force_accuracy=force_accuracy,
        force_shots=force_shots,
        force_performance_scores=force_performance_scores,
        force_enemy_mmr=force_enemy_mmr,
        force_assets=force_assets,
        force_aliases=force_aliases,
        force_participants=force_participants,
        force_skill=force_skill,
        force_pve_stats=force_pve_stats,
        weapons=weapons,
        force_weapons=force_weapons,
        playable_duration=getattr(scope, "playable_duration", False) if scope is not None else False,
        force_playable_duration=getattr(scope, "force_playable_duration", False) if scope is not None else False,
    )

    # Fusionner résultats locaux + shared (dédoublonner, garder l'ordre)
    if shared_match_ids:
        seen = set(local_match_ids)
        merged = list(local_match_ids)
        for mid in shared_match_ids:
            if mid not in seen:
                merged.append(mid)
                seen.add(mid)
        return merged

    return local_match_ids


# ─────────────────────────────────────────────────────────────────────────────
# Helpers internes
# ─────────────────────────────────────────────────────────────────────────────


def _has_backfill_completed_column(shared_conn: Any) -> bool:
    """Vérifie si match_registry possède la colonne backfill_completed."""
    try:
        result = shared_conn.execute(
            "SELECT COUNT(*) FROM information_schema.columns "
            "WHERE table_name = 'match_registry' AND column_name = 'backfill_completed'"
        ).fetchone()
        return bool(result and result[0] > 0)
    except Exception:
        return False


def _done_guard(flag_name: str, has_column: bool) -> str:
    """Clause SQL excluant les matchs déjà traités dans match_registry.

    Utilise mr.backfill_completed.

    .. warning::
        Ne convient que pour les données **globales au match** (assets,
        participants, aliases…).  Pour les données **per-player** (performance_scores,
        personal_scores, medals, accuracy, shots, enemy_mmr), utiliser
        ``_player_done_guard()`` qui vérifie les données réelles du joueur courant.
    """
    if not has_column:
        return ""
    bit = BACKFILL_FLAGS.get(flag_name, 0)
    if bit == 0:
        return ""
    return f" AND (COALESCE(mr.backfill_completed, 0) & {bit} = 0)"


def _player_done_guard(conn: Any, table: str, column: str | None = None) -> str:
    """Clause SQL excluant les matchs déjà traités dans la player DB.

    Fix multi-joueur (v5.5) : vérifie les données réelles du joueur courant
    au lieu du bitmask global ``backfill_completed`` qui est posé par le
    premier joueur syncé et masque les matchs pour les autres joueurs.

    Args:
        conn: Connexion à la player DB (stats.duckdb).
        table: Nom de la table à vérifier.
        column: Si fourni, filtre sur ``column IS NOT NULL``.

    Returns:
        Clause SQL ``mp.match_id NOT IN (...)`` ou ``1=1`` si table vide.
    """
    try:
        if column:
            query = f"SELECT match_id FROM {table} WHERE {column} IS NOT NULL"
        else:
            query = f"SELECT DISTINCT match_id FROM {table}"
        done_ids = {r[0] for r in conn.execute(query).fetchall()}
    except Exception:
        return "1=1"
    if not done_ids:
        return "1=1"
    items = ",".join(f"'{mid}'" for mid in done_ids)
    return f"mp.match_id NOT IN ({items})"


def _find_matches_in_shared_all(
    shared_conn: Any,
    conn: Any,
    xuid: str,
    *,
    max_matches: int | None,
    medals: bool = False,
    events: bool = False,
    skill: bool = False,
    personal_scores: bool = False,
    performance_scores: bool = False,
    accuracy: bool = False,
    shots: bool = False,
    enemy_mmr: bool = False,
    assets: bool = False,
    participants: bool = False,
    pve_stats: bool = False,
    force_medals: bool = False,
    force_accuracy: bool = False,
    force_shots: bool = False,
    force_performance_scores: bool = False,
    force_enemy_mmr: bool = False,
    force_assets: bool = False,
    force_aliases: bool = False,
    force_participants: bool = False,
    force_skill: bool = False,  # Ajouté v5.1 : force le rescan skill pour tous les matchs
    force_pve_stats: bool = False,  # v5.2 : force le rescan PVE pour tous les Firefight
    weapons: bool = False,  # v5.5 : kills par arme → shared.weapon_kills
    force_weapons: bool = False,  # v5.5 : ignorer MatchBits.WEAPON_KILLS
    playable_duration: bool = False,  # v6.3 : playable_duration_seconds + real_start_time
    force_playable_duration: bool = False,  # v6.3 : re-traiter même si déjà rempli
) -> list[str]:
    """Détection V5 FINALE : tous les flags via shared DB.

    Remplace la détection via match_stats quand celle-ci n'existe plus.
    Base : shared.match_participants + match_registry.

    force_skill (v5.1) : quand True, utilise "1=1" au lieu de
    "(mp.team_mmr IS NULL)" pour re-télécharger les données skill
    de TOUS les matchs, y compris ceux déjà peuplés.
    """
    conditions: list[str] = []
    has_bf_col = _has_backfill_completed_column(shared_conn)

    # Médailles — per-player : NOT IN ... WHERE xuid = ? suffit
    if medals:
        conditions.append(
            "mp.match_id NOT IN (SELECT DISTINCT match_id FROM medals_earned WHERE xuid = ?)"
        )

    # Events
    # Fix v5.4: utiliser mr.events_loaded (source de vérité 100% fiable) plutôt que
    # le bitmask backfill_completed. Le bit "events" (=2) était posé dans
    # backfill_completed même quand l'API retournait highlight_events=[] → les matchs
    # sans events étaient exclus silencieusement de la détection.
    if events:
        conditions.append("mr.events_loaded = false")

    # Skill
    # NOTE V5.2 : Condition composite pour éviter la re-détection infinie des modes
    # non-classés (Firefight, Rumble Pit, social…).
    #
    # Guard per-player (backfill_bits) + guard global (backfill_completed).
    # Le guard global empêche la boucle infinie pour les modes non-ranked
    # (team_mmr NULL par design), mais a le même faux positif multi-joueur
    # que les autres flags per-player.
    # TODO(v5.6): migrer vers un ParticipantBits.SKILL_ATTEMPTED posé même
    # quand les données sont NULL, pour remplacer le guard global.
    if skill:
        if force_skill:
            conditions.append("1=1")
        else:
            # bit TEAM_MMR = 1 dans ParticipantBits ; skill = 4 dans BACKFILL_FLAGS
            conditions.append(
                "(COALESCE(mp.backfill_bits, 0) & 1) = 0"
                " AND (COALESCE(mr.backfill_completed, 0) & 4) = 0"
            )

    # Personal scores — per-player : vérifier données réelles dans player DB
    if personal_scores:
        conditions.append(_player_done_guard(conn, "personal_score_awards"))

    # Performance scores — per-player : vérifier player_match_enrichment du joueur
    if performance_scores:
        if force_performance_scores:
            conditions.append("1=1")
        else:
            conditions.append(
                _player_done_guard(conn, "player_match_enrichment", "performance_score")
            )

    # Accuracy — per-player : mp.accuracy IS NULL est déjà filtré par xuid
    if accuracy:
        if force_accuracy:
            conditions.append("1=1")
        else:
            conditions.append("(mp.accuracy IS NULL)")

    # Shots — per-player : mp.shots_* IS NULL est déjà filtré par xuid
    if shots:
        if force_shots:
            conditions.append("1=1")
        else:
            conditions.append("(mp.shots_fired IS NULL OR mp.shots_hit IS NULL)")

    # Enemy MMR — per-player : mp.team_mmr IS NULL est déjà filtré par xuid
    if enemy_mmr:
        conditions.append("(mp.team_mmr IS NULL)")

    # Assets
    if assets:
        asset_cond = (
            "(mr.playlist_name IS NULL OR mr.map_name IS NULL "
            "OR mr.pair_name IS NULL OR mr.game_variant_name IS NULL)"
        )
        if force_assets:
            conditions.append(asset_cond)
        else:
            conditions.append(asset_cond + _done_guard("assets", has_bf_col))

    # Aliases (force uniquement)
    if force_aliases:
        conditions.append("1=1")

    # Participants
    if participants:
        if force_participants:
            conditions.append("1=1")
        else:
            conditions.append("1=1" + _done_guard("participants", has_bf_col))

    # PVE stats (Firefight uniquement) — v5.2
    # Double guard : is_firefight = TRUE ET PVE_STATS bit non posé
    if pve_stats:
        from src.data.sync.constants import MatchBits

        pve_bit = MatchBits.PVE_STATS
        if force_pve_stats:
            conditions.append("mr.is_firefight = TRUE")
        else:
            conditions.append(
                f"mr.is_firefight = TRUE AND (COALESCE(mr.backfill_completed, 0) & {pve_bit}) = 0"
            )

    # Weapon kills — v5.5 / v5.6
    # Guard : WEAPON_KILLS bit non posé dans backfill_completed.
    # WEAPON_KILLS_NO_FILM : film 404/expiré → exclus par défaut, inclus avec force_weapons.
    if weapons:
        from src.data.sync.constants import MatchBits

        wk_bit = MatchBits.WEAPON_KILLS
        no_film_bit = MatchBits.WEAPON_KILLS_NO_FILM
        if force_weapons:
            # --force-weapons : re-tenter tout, y compris les matchs NO_FILM
            conditions.append("1=1")
        else:
            # Exclure WEAPON_KILLS (traité) ET WEAPON_KILLS_NO_FILM (film expiré)
            conditions.append(
                f"(COALESCE(mr.backfill_completed, 0) & {wk_bit}) = 0"
                f" AND (COALESCE(mr.backfill_completed, 0) & {no_film_bit}) = 0"
            )

    # Playable duration — global match : colonne NULL dans match_registry
    # Note : utiliser match_registry directement (pas v_match_full qui peut ne pas exposer la colonne)
    if playable_duration:
        if force_playable_duration:
            conditions.append("1=1")
        else:
            conditions.append(
                "mp.match_id NOT IN ("
                "  SELECT match_id FROM match_registry WHERE playable_duration_seconds IS NOT NULL"
                ")"
            )

    if not conditions:
        return []

    where_clause = " OR ".join(conditions)

    # Paramètres pour les sous-requêtes de médailles
    params: list[Any] = [xuid]
    if medals:
        params.append(xuid)

    _mr_src = _get_match_source(shared_conn)
    query = f"""
        SELECT DISTINCT mp.match_id
        FROM match_participants mp
        JOIN {_mr_src} mr ON mr.match_id = mp.match_id
        WHERE mp.xuid = ? AND ({where_clause})
        ORDER BY mr.start_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        result = shared_conn.execute(query, params).fetchall()
        return [row[0] for row in result]
    except Exception as e:
        logger.error(f"Erreur détection V5 dans shared DB: {e}")
        return []


def _find_matches_in_shared_db(
    shared_conn: Any,
    xuid: str,
    *,
    max_matches: int | None,
    participants_scores: bool,
    participants_kda: bool,
    participants_shots: bool,
    participants_damage: bool,
    participants_avg_life: bool,
    force_participants_shots: bool,
    force_participants_damage: bool,
    force_participants_avg_life: bool,
) -> list[str]:
    """Détection des matchs manquants dans shared_matches.duckdb (v5).

    Utilisé pour les backfills participants-only qui travaillent directement
    sur la base partagée sans dépendre de la DB joueur locale.

    La requête filtre d'abord par xuid dans match_participants, puis
    applique les conditions de NULL sur les colonnes demandées.

    Args:
        shared_conn: Connexion à shared_matches.duckdb.
        xuid: XUID du joueur.
        max_matches: Limite de résultats.
        participants_*: Flags de backfill.
        force_participants_*: Flags de forçage.

    Returns:
        Liste des match_id nécessitant un backfill.
    """
    conditions: list[str] = []

    # Vérifier si backfill_completed existe dans match_registry
    has_bf_col = _has_backfill_completed_column(shared_conn)

    # ── Participants scores/rank ──
    if participants_scores:
        conditions.append(
            "(mp.score IS NULL OR mp.rank IS NULL)" + _done_guard("participants_scores", has_bf_col)
        )

    # ── Participants K/D/A ──
    if participants_kda:
        conditions.append(
            "(mp.kills IS NULL OR mp.deaths IS NULL OR mp.assists IS NULL)"
            + _done_guard("participants_kda", has_bf_col)
        )

    # ── Participants shots ──
    if participants_shots:
        if force_participants_shots:
            conditions.append("1=1")
        else:
            conditions.append(
                "(mp.shots_fired IS NULL OR mp.shots_hit IS NULL)"
                + _done_guard("participants_shots", has_bf_col)
            )

    # ── Participants damage ──
    if participants_damage:
        if force_participants_damage:
            conditions.append("1=1")
        else:
            conditions.append(
                "(mp.damage_dealt IS NULL OR mp.damage_taken IS NULL)"
                + _done_guard("participants_damage", has_bf_col)
            )

    # ── Participants avg_life_seconds ──
    if participants_avg_life:
        if force_participants_avg_life:
            conditions.append("1=1")
        else:
            conditions.append(
                "mp.avg_life_seconds IS NULL" + _done_guard("participants_avg_life", has_bf_col)
            )

    if not conditions:
        return []

    where_clause = " OR ".join(conditions)

    query = f"""
        SELECT DISTINCT mp.match_id
        FROM match_participants mp
        JOIN match_registry mr ON mr.match_id = mp.match_id
        WHERE mp.xuid = ? AND ({where_clause})
        ORDER BY mr.end_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        result = shared_conn.execute(query, [xuid]).fetchall()
        return [row[0] for row in result]
    except Exception as e:
        logger.error(f"Erreur lors de la détection dans shared DB: {e}")
        return []


# ─────────────────────────────────────────────────────────────────────────────
# Détection v5.2 — Bitmask granulaire par participant
# ─────────────────────────────────────────────────────────────────────────────


def find_matches_missing_participant_bits(
    shared_conn: Any,
    xuid: str,
    *,
    bits_required: int,
    force: bool = False,
    max_matches: int | None = None,
) -> list[str]:
    """Trouve les matchs où ce joueur a des bits ``backfill_bits`` manquants.

    Utilise le bitmask granulaire ``match_participants.backfill_bits``
    pour identifier précisément quelles données sont absentes.

    Args:
        shared_conn: Connexion à shared_matches.duckdb.
        xuid: XUID du joueur.
        bits_required: Masque des bits requis (ex: ``ParticipantBits.SKILL``).
        force: Si True, retourne tous les matchs (ignore le bitmask).
        max_matches: Limite de résultats.

    Returns:
        Liste des match_id nécessitant un backfill.

    Note:
        ``COALESCE(backfill_bits, 0)`` est obligatoire — les enregistrements
        antérieurs à la migration ont ``backfill_bits IS NULL``.
        Sans COALESCE, NULL & bits_required = NULL != bits_required, ce qui
        forcerait un backfill pour tous les anciens matchs.
    """
    if force:
        condition = "1=1"
    else:
        condition = f"(COALESCE(mp.backfill_bits, 0) & {bits_required}) != {bits_required}"

    _mr_src = _get_match_source(shared_conn)
    query = f"""
        SELECT mp.match_id
        FROM match_participants mp
        JOIN {_mr_src} mr ON mr.match_id = mp.match_id
        WHERE mp.xuid = ? AND {condition}
        ORDER BY mr.start_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        return [row[0] for row in shared_conn.execute(query, [xuid]).fetchall()]
    except Exception as e:
        logger.error(f"Erreur détection bits participants (xuid={xuid}): {e}")
        return []


def find_matches_missing_match_bits(
    shared_conn: Any,
    *,
    bits_required: int,
    force: bool = False,
    max_matches: int | None = None,
) -> list[str]:
    """Trouve les matchs où des données au niveau match sont manquantes.

    Utilise ``match_registry.backfill_completed`` avec les bits ``MatchBits``
    (bits ≥ 16). Pas de filtre xuid — les données match sont globales.

    Args:
        shared_conn: Connexion à shared_matches.duckdb.
        bits_required: Masque des bits requis (ex: ``MatchBits.EVENTS``).
        force: Si True, retourne tous les matchs.
        max_matches: Limite de résultats.

    Returns:
        Liste des match_id nécessitant un backfill.
    """
    if force:
        condition = "1=1"
    else:
        condition = f"(COALESCE(mr.backfill_completed, 0) & {bits_required}) != {bits_required}"

    _mr_src = _get_match_source(shared_conn)
    query = f"""
        SELECT mr.match_id
        FROM {_mr_src} mr
        WHERE {condition}
        ORDER BY mr.start_time DESC
    """
    if max_matches:
        query += f" LIMIT {max_matches}"

    try:
        return [row[0] for row in shared_conn.execute(query).fetchall()]
    except Exception as e:
        logger.error(f"Erreur détection bits match: {e}")
        return []


def compute_bits_needed_from_scope(scope: SyncScope) -> tuple[int, int]:
    """Calcule les masques de bits requis depuis un SyncScope.

    Args:
        scope: Périmètre de synchronisation.

    Returns:
        Tuple (participant_bits, match_bits) pour la détection.
    """
    from src.data.sync.constants import MatchBits, ParticipantBits

    p_bits = 0  # bits participant (match_participants.backfill_bits)
    m_bits = 0  # bits match (match_registry.backfill_completed, bits ≥ 16)

    # ── Skill / MMR ──
    if getattr(scope, "team_mmr", False) or getattr(scope, "mmr", False):
        p_bits |= ParticipantBits.TEAM_MMR
    if scope.enemy_mmr or getattr(scope, "mmr", False):
        p_bits |= ParticipantBits.ENEMY_MMR
    if getattr(scope, "kills_expected", False) or getattr(scope, "expected", False):
        p_bits |= ParticipantBits.KILLS_EXP
    if getattr(scope, "deaths_expected", False) or getattr(scope, "expected", False):
        p_bits |= ParticipantBits.DEATHS_EXP
    if getattr(scope, "assists_expected", False) or getattr(scope, "expected", False):
        p_bits |= ParticipantBits.ASSISTS_EXP
    if scope.skill:
        p_bits |= ParticipantBits.SKILL

    # ── Combat ──
    if scope.accuracy or getattr(scope, "combat", False) or getattr(scope, "core_stats", False):
        p_bits |= ParticipantBits.ACCURACY
    if scope.shots or getattr(scope, "combat", False) or getattr(scope, "core_stats", False):
        p_bits |= ParticipantBits.SHOTS
    if (
        getattr(scope, "damage", False)
        or getattr(scope, "combat", False)
        or getattr(scope, "core_stats", False)
    ):
        p_bits |= ParticipantBits.DAMAGE
    if getattr(scope, "avg_life", False) or getattr(scope, "core_stats", False):
        p_bits |= ParticipantBits.AVG_LIFE

    # ── Kills détaillés ──
    if (
        getattr(scope, "grenade_kills", False)
        or getattr(scope, "kills_detail", False)
        or getattr(scope, "core_stats", False)
    ):
        p_bits |= ParticipantBits.GRENADE_KILLS
    if (
        getattr(scope, "melee_kills", False)
        or getattr(scope, "kills_detail", False)
        or getattr(scope, "core_stats", False)
    ):
        p_bits |= ParticipantBits.MELEE_KILLS
    if (
        getattr(scope, "power_weapon_kills", False)
        or getattr(scope, "kills_detail", False)
        or getattr(scope, "core_stats", False)
    ):
        p_bits |= ParticipantBits.POWER_WEAPON
    if (
        getattr(scope, "headshot_kills", False)
        or getattr(scope, "kills_detail", False)
        or getattr(scope, "core_stats", False)
    ):
        p_bits |= ParticipantBits.HEADSHOT_KILLS
    if getattr(scope, "max_spree", False) or getattr(scope, "core_stats", False):
        p_bits |= ParticipantBits.MAX_SPREE
    if getattr(scope, "kda_recalc", False) or getattr(scope, "core_stats", False):
        p_bits |= ParticipantBits.KDA
    if getattr(scope, "time_played", False) or getattr(scope, "core_stats", False):
        p_bits |= ParticipantBits.TIME_PLAYED

    # ── Médailles ──
    if scope.medals:
        p_bits |= ParticipantBits.MEDALS

    # ── Match-level ──
    if scope.events:
        m_bits |= MatchBits.EVENTS
    if scope.assets:
        m_bits |= MatchBits.ASSETS
    if scope.aliases:
        m_bits |= MatchBits.ALIASES
    if getattr(scope, "pve_stats", False):
        m_bits |= MatchBits.PVE_STATS

    return p_bits, m_bits


# ─────────────────────────────────────────────────────────────────────────────
# Détection de matchs syncés avec une version SPNKr obsolète (v5.4)
# ─────────────────────────────────────────────────────────────────────────────


def find_matches_with_stale_spnkr(
    shared_conn: Any,
    *,
    min_version: str = "0.10.1",
    max_matches: int | None = None,
) -> dict[str, list[str]]:
    """Identifie les matchs potentiellement impactés par un bug SPNKr.

    Retourne deux listes :
    - ``stale_versioned`` : matchs dont ``sync_spnkr_version`` est inférieure
      à *min_version* et dont les events ont été chargés (possiblement corrompus).
    - ``stale_unknown`` : matchs récents sans version trackée ET sans events
      (probablement échoués à cause du bug, avant l'ajout du tracking).

    Args:
        shared_conn: Connexion à shared_matches.duckdb.
        min_version: Version SPNKr minimum acceptable.
        max_matches: Limite de résultats par catégorie.

    Returns:
        Dict avec clés ``stale_versioned`` et ``stale_unknown``.
    """
    from src.data.sync.migrations import column_exists

    result: dict[str, list[str]] = {"stale_versioned": [], "stale_unknown": []}

    has_col = column_exists(shared_conn, "match_registry", "sync_spnkr_version")

    # 1) Matchs avec version connue < min_version ET events chargés
    if has_col:
        q1 = """
            SELECT mr.match_id
            FROM match_registry mr
            WHERE mr.sync_spnkr_version IS NOT NULL
              AND mr.sync_spnkr_version < ?
              AND mr.events_loaded = TRUE
            ORDER BY mr.start_time DESC
        """
        if max_matches:
            q1 += f" LIMIT {max_matches}"
        try:
            result["stale_versioned"] = [
                row[0] for row in shared_conn.execute(q1, (min_version,)).fetchall()
            ]
        except Exception as e:
            logger.error(f"Erreur détection stale (versioned): {e}")

    # 2) Matchs pré-tracking (version NULL) + events absents + récents
    q2 = """
        SELECT mr.match_id
        FROM match_registry mr
        WHERE mr.events_loaded = FALSE
          AND mr.start_time >= '2025-12-01'
    """
    if has_col:
        q2 += " AND mr.sync_spnkr_version IS NULL"
    q2 += " ORDER BY mr.start_time DESC"
    if max_matches:
        q2 += f" LIMIT {max_matches}"
    try:
        result["stale_unknown"] = [row[0] for row in shared_conn.execute(q2).fetchall()]
    except Exception as e:
        logger.error(f"Erreur détection stale (unknown): {e}")

    return result
