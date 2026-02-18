"""Service d'estimation de timeline approximative pour les événements d'objectifs.

Ce module permet de créer une timeline grossière des captures de drapeaux/bases
en croisant les données PersonalScores (sans timestamp) avec highlight_events
(kills/deaths horodatés).

Principe :
1. Extraire tous les highlight_events d'un joueur (avec time_ms)
2. Si le joueur a capturé N fois, distribuer les N captures entre ses événements
3. Utiliser une interpolation simple pour estimer les moments de capture

Précision attendue : ±30 secondes

Exemples:
    >>> from src.data.services.objective_timeline_service import estimate_capture_timeline
    >>> timeline = estimate_capture_timeline(repo, "match_abc123", "CTF")
    >>> print(timeline)
    
    team_id  gamertag     award_name      count  estimated_time_ms  confidence
    0        JohnSpartan  Flag Captured   3      45000             high
    0        JohnSpartan  Flag Captured   3      120000            medium
    0        JohnSpartan  Flag Captured   3      180000            high
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Literal

import polars as pl

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository


def get_player_highlight_events(
    repo: DuckDBRepository,
    match_id: str,
    xuid: str,
) -> pl.DataFrame:
    """Extrait tous les highlight_events d'un joueur avec timestamps.
    
    Récupère les kills et deaths d'un joueur spécifique depuis la table
    highlight_events de shared_matches.duckdb.
    
    Args:
        repo: Repository DuckDB (player-specific ou shared).
        match_id: ID du match à analyser.
        xuid: XUID du joueur.
        
    Returns:
        DataFrame Polars avec colonnes:
        - time_ms (int): Timestamp en millisecondes
        - event_type (str): "kill" ou "death"
        - killer_xuid (str): XUID du tueur (pour kills)
        - victim_xuid (str): XUID de la victime (pour deaths)
        
    Example:
        >>> events = get_player_highlight_events(repo, "match123", "xuid456")
        >>> print(events.sort("time_ms"))
        time_ms  event_type
        15000    kill
        32000    death
        48000    kill
    """
    # Note: highlight_events est dans shared_matches.duckdb
    # Si repo pointe vers player DB, il faut attacher shared DB
    query = """
    SELECT 
        time_ms,
        event_type,
        killer_xuid,
        victim_xuid
    FROM highlight_events
    WHERE match_id = ?
      AND (killer_xuid = ? OR victim_xuid = ?)
    ORDER BY time_ms ASC
    """
    return repo.query_df(query, [match_id, xuid, xuid])


def estimate_objective_captures_timeline(
    repo: DuckDBRepository,
    match_id: str,
    mode: Literal["CTF", "Strongholds", "Oddball"] = "CTF",
) -> pl.DataFrame:
    """Estime la timeline des captures d'objectifs en croisant les données.
    
    Algorithme :
    1. Récupérer tous les joueurs ayant capturé (personal_score_awards)
    2. Pour chaque joueur, récupérer ses highlight_events (avec time_ms)
    3. Distribuer ses captures entre ses événements horodatés
    4. Calculer un score de confiance basé sur la densité d'événements
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match à analyser.
        mode: Mode de jeu ("CTF", "Strongholds", "Oddball").
        
    Returns:
        DataFrame Polars avec colonnes:
        - team_id (int): Équipe du joueur
        - xuid (str): XUID du joueur
        - gamertag (str): Nom du joueur
        - award_name (str): Type de capture
        - capture_index (int): Numéro de la capture (1, 2, 3, ...)
        - estimated_time_ms (int): Timestamp estimé
        - confidence (str): "high", "medium", "low"
        - nearby_events_count (int): Nombre d'événements proches
        
    Example:
        >>> timeline = estimate_objective_captures_timeline(repo, "match123", "CTF")
        >>> print(timeline.head(3))
        team_id  gamertag     award_name      capture_index  estimated_time_ms  confidence
        0        JohnSpartan  Flag Captured   1              45000             high
        0        JohnSpartan  Flag Captured   2              120000            medium
        0        JohnSpartan  Flag Captured   3              180000            high
        
    Note:
        La précision dépend de l'activité du joueur (nombre de kills/deaths).
        Un joueur avec beaucoup d'activité aura des estimations plus précises.
    """
    # Déterminer les awards à chercher selon le mode
    award_filters = _get_award_filters_for_mode(mode)
    
    # Étape 1 : Récupérer les joueurs ayant capturé
    capturers_query = f"""
    SELECT 
        mp.team_id,
        mp.xuid,
        mp.gamertag,
        psa.award_name,
        psa.award_count
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name IN ({', '.join('?' * len(award_filters))})
    ORDER BY mp.team_id, mp.gamertag
    """
    capturers = repo.query_df(
        capturers_query, 
        [match_id] + award_filters
    )
    
    if capturers.is_empty():
        return pl.DataFrame()
    
    # Étape 2 : Pour chaque joueur, estimer les moments de capture
    all_estimates = []
    
    for row in capturers.iter_rows(named=True):
        xuid = row["xuid"]
        capture_count = row["award_count"]
        
        # Récupérer les highlight_events du joueur
        events = get_player_highlight_events(repo, match_id, xuid)
        
        if events.is_empty():
            # Pas d'événements horodatés, impossible d'estimer
            continue
        
        # Estimer les timestamps des captures
        estimates = _distribute_captures_across_events(
            events=events,
            capture_count=capture_count,
            team_id=row["team_id"],
            xuid=xuid,
            gamertag=row["gamertag"],
            award_name=row["award_name"],
        )
        
        all_estimates.append(estimates)
    
    if not all_estimates:
        return pl.DataFrame()
    
    # Combiner toutes les estimations
    timeline = pl.concat(all_estimates)
    
    return timeline.sort(["estimated_time_ms", "team_id", "gamertag"])


def _get_award_filters_for_mode(mode: str) -> list[str]:
    """Retourne les awards à filtrer selon le mode de jeu.
    
    Args:
        mode: "CTF", "Strongholds", ou "Oddball".
        
    Returns:
        Liste des noms d'awards à chercher.
    """
    if mode == "CTF":
        return ["Flag Captured"]
    elif mode == "Strongholds":
        return ["Zone Captured 100%", "Zone Captured 75%", "Zone Captured 50%"]
    elif mode == "Oddball":
        return ["Ball Control"]
    else:
        return ["Flag Captured"]  # Défaut


def _distribute_captures_across_events(
    events: pl.DataFrame,
    capture_count: int,
    team_id: int,
    xuid: str,
    gamertag: str,
    award_name: str,
) -> pl.DataFrame:
    """Distribue les captures entre les événements horodatés d'un joueur.
    
    Algorithme :
    1. Diviser la plage temporelle des événements en N segments (N = capture_count)
    2. Pour chaque segment, prendre le milieu comme estimation
    3. Calculer un score de confiance basé sur la densité d'événements autour
    
    Args:
        events: DataFrame des highlight_events du joueur (avec time_ms).
        capture_count: Nombre de captures à distribuer.
        team_id: ID de l'équipe.
        xuid: XUID du joueur.
        gamertag: Nom du joueur.
        award_name: Type de capture ("Flag Captured", etc.).
        
    Returns:
        DataFrame avec les estimations de timeline.
    """
    if events.is_empty() or capture_count == 0:
        return pl.DataFrame()
    
    # Extraire les timestamps triés
    timestamps = events["time_ms"].sort().to_list()
    
    if len(timestamps) == 0:
        return pl.DataFrame()
    
    min_time = timestamps[0]
    max_time = timestamps[-1]
    total_duration = max_time - min_time
    
    if total_duration == 0:
        # Tous les événements au même instant (rare), utiliser ce timestamp
        estimated_times = [min_time] * capture_count
        confidences = ["low"] * capture_count
        nearby_counts = [len(timestamps)] * capture_count
    else:
        # Diviser en segments égaux
        segment_size = total_duration / capture_count
        estimated_times = []
        confidences = []
        nearby_counts = []
        
        for i in range(capture_count):
            # Centre du segment
            segment_center = min_time + int((i + 0.5) * segment_size)
            estimated_times.append(segment_center)
            
            # Compter les événements proches (±30s)
            window = 30000  # 30 secondes en ms
            nearby = sum(
                1 for t in timestamps 
                if abs(t - segment_center) <= window
            )
            nearby_counts.append(nearby)
            
            # Score de confiance basé sur la densité
            if nearby >= 5:
                confidences.append("high")
            elif nearby >= 2:
                confidences.append("medium")
            else:
                confidences.append("low")
    
    # Créer le DataFrame de résultat
    return pl.DataFrame({
        "team_id": [team_id] * capture_count,
        "xuid": [xuid] * capture_count,
        "gamertag": [gamertag] * capture_count,
        "award_name": [award_name] * capture_count,
        "capture_index": list(range(1, capture_count + 1)),
        "estimated_time_ms": estimated_times,
        "confidence": confidences,
        "nearby_events_count": nearby_counts,
    })


def format_timeline_for_display(
    timeline: pl.DataFrame,
    match_duration_ms: int | None = None,
) -> pl.DataFrame:
    """Formate la timeline pour affichage UI avec timestamps lisibles.
    
    Convertit les timestamps en format MM:SS et ajoute des informations
    contextuelles pour l'affichage.
    
    Args:
        timeline: DataFrame retourné par estimate_objective_captures_timeline().
        match_duration_ms: Durée totale du match en ms (optionnel).
        
    Returns:
        DataFrame enrichi avec colonnes supplémentaires:
        - time_formatted (str): "MM:SS"
        - time_percent (float): Position dans le match (0.0 - 1.0)
        
    Example:
        >>> timeline = estimate_objective_captures_timeline(repo, "match123")
        >>> display = format_timeline_for_display(timeline, 600000)
        >>> print(display[["gamertag", "time_formatted", "confidence"]])
        gamertag     time_formatted  confidence
        JohnSpartan  01:30          high
        JohnSpartan  03:45          medium
        Arbiter      05:12          high
    """
    if timeline.is_empty():
        return timeline
    
    # Convertir time_ms en MM:SS
    timeline = timeline.with_columns([
        pl.col("estimated_time_ms")
        .map_elements(lambda ms: _ms_to_mmss(ms), return_dtype=pl.Utf8)
        .alias("time_formatted")
    ])
    
    # Ajouter pourcentage si durée fournie
    if match_duration_ms:
        timeline = timeline.with_columns([
            (pl.col("estimated_time_ms") / match_duration_ms)
            .alias("time_percent")
        ])
    
    return timeline


def _ms_to_mmss(ms: int) -> str:
    """Convertit des millisecondes en format MM:SS.
    
    Args:
        ms: Millisecondes.
        
    Returns:
        String au format "MM:SS".
        
    Example:
        >>> _ms_to_mmss(90000)
        '01:30'
        >>> _ms_to_mmss(195000)
        '03:15'
    """
    if ms is None:
        return "00:00"
    
    total_seconds = ms // 1000
    minutes = total_seconds // 60
    seconds = total_seconds % 60
    
    return f"{minutes:02d}:{seconds:02d}"


def get_timeline_summary(
    timeline: pl.DataFrame,
) -> dict[str, any]:
    """Calcule des statistiques résumées sur la timeline.
    
    Args:
        timeline: DataFrame retourné par estimate_objective_captures_timeline().
        
    Returns:
        Dictionnaire avec:
        - total_captures (int): Nombre total de captures estimées
        - high_confidence_count (int): Captures avec confiance "high"
        - medium_confidence_count (int): Captures avec confiance "medium"
        - low_confidence_count (int): Captures avec confiance "low"
        - average_nearby_events (float): Moyenne des événements proches
        - first_capture_time_ms (int): Timestamp de la première capture
        - last_capture_time_ms (int): Timestamp de la dernière capture
        
    Example:
        >>> summary = get_timeline_summary(timeline)
        >>> print(f"Précision globale: {summary['high_confidence_count']} / {summary['total_captures']}")
        Précision globale: 8 / 12
    """
    if timeline.is_empty():
        return {
            "total_captures": 0,
            "high_confidence_count": 0,
            "medium_confidence_count": 0,
            "low_confidence_count": 0,
            "average_nearby_events": 0.0,
            "first_capture_time_ms": None,
            "last_capture_time_ms": None,
        }
    
    return {
        "total_captures": timeline.height,
        "high_confidence_count": timeline.filter(pl.col("confidence") == "high").height,
        "medium_confidence_count": timeline.filter(pl.col("confidence") == "medium").height,
        "low_confidence_count": timeline.filter(pl.col("confidence") == "low").height,
        "average_nearby_events": timeline["nearby_events_count"].mean(),
        "first_capture_time_ms": timeline["estimated_time_ms"].min(),
        "last_capture_time_ms": timeline["estimated_time_ms"].max(),
    }


# =============================================================================
# Agrégation par équipe : Estimation du contrôle de bases
# =============================================================================


def estimate_team_base_control(
    timeline: pl.DataFrame,
    mode: Literal["CTF", "Strongholds", "Oddball"] = "Strongholds",
    window_ms: int = 15000,
) -> pl.DataFrame:
    """Estime le nombre de bases contrôlées par équipe (agrégation).
    
    Problème : L'API ne fournit pas l'ID de la base capturée (A, B ou C).
    On doit donc estimer combien de bases UNIQUES ont été capturées.
    
    Approche : Clustering temporel - Les captures proches temporellement
    sont probablement la même base.
    
    Args:
        timeline: DataFrame retourné par estimate_objective_captures_timeline().
        mode: Mode de jeu pour limites ("CTF", "Strongholds", "Oddball").
        window_ms: Fenêtre temporelle en ms pour grouper les captures (défaut: 15s).
        
    Returns:
        DataFrame avec colonnes:
        - team_id (int): ID de l'équipe
        - total_captures (int): Nombre total de captures individuelles
        - estimated_unique_bases (int): Estimation du nombre de bases uniques
        - confidence (str): "high", "medium", "low"
        - method (str): "temporal_clustering"
        
    Example:
        >>> timeline = estimate_objective_captures_timeline(repo, match_id, "Strongholds")
        >>> control = estimate_team_base_control(timeline, "Strongholds")
        >>> print(control)
        team_id  total_captures  estimated_unique_bases  confidence
        0        8               3                       high
        1        6               2                       medium
        
    Note:
        Pour Strongholds, max 3 bases. Si estimated > 3, on plafonne à 3.
    """
    if timeline.is_empty():
        return pl.DataFrame()
    
    # Limites par mode
    max_bases = {
        "Strongholds": 3,
        "CTF": 1,  # 1 drapeau à la fois
        "Oddball": 1,  # 1 balle
    }
    
    mode_limit = max_bases.get(mode, 10)  # Défaut si mode inconnu
    
    # Grouper par équipe
    results = []
    
    for team_id in timeline["team_id"].unique().sort():
        team_data = timeline.filter(pl.col("team_id") == team_id).sort("estimated_time_ms")
        
        if team_data.is_empty():
            continue
        
        # Détecter les clusters temporels
        timestamps = team_data["estimated_time_ms"].to_list()
        clusters = _detect_temporal_clusters(timestamps, window_ms)
        
        # Nombre de bases uniques = nombre de clusters
        estimated_bases = min(len(clusters), mode_limit)
        
        # Calculer confiance basée sur la cohérence des clusters
        confidence = _calculate_cluster_confidence(clusters, team_data.height)
        
        results.append({
            "team_id": team_id,
            "total_captures": team_data.height,
            "estimated_unique_bases": estimated_bases,
            "confidence": confidence,
            "method": "temporal_clustering",
        })
    
    return pl.DataFrame(results)


def _detect_temporal_clusters(
    timestamps: list[int],
    window_ms: int,
) -> list[list[int]]:
    """Détecte les clusters temporels dans une liste de timestamps.
    
    Un cluster = groupe de timestamps espacés de moins de window_ms.
    
    Args:
        timestamps: Liste de timestamps en ms (doit être triée).
        window_ms: Fenêtre temporelle en ms.
        
    Returns:
        Liste de clusters, chaque cluster est une liste de timestamps.
        
    Example:
        >>> timestamps = [10000, 12000, 50000, 52000, 90000]
        >>> clusters = _detect_temporal_clusters(timestamps, 15000)
        >>> len(clusters)
        3
        >>> # Cluster 1: [10000, 12000]
        >>> # Cluster 2: [50000, 52000]
        >>> # Cluster 3: [90000]
    """
    if not timestamps:
        return []
    
    clusters = []
    current_cluster = [timestamps[0]]
    
    for i in range(1, len(timestamps)):
        # Si le timestamp est proche du dernier du cluster actuel
        if timestamps[i] - current_cluster[-1] <= window_ms:
            current_cluster.append(timestamps[i])
        else:
            # Nouveau cluster
            clusters.append(current_cluster)
            current_cluster = [timestamps[i]]
    
    # Ajouter le dernier cluster
    if current_cluster:
        clusters.append(current_cluster)
    
    return clusters


def _calculate_cluster_confidence(
    clusters: list[list[int]],
    total_captures: int,
) -> str:
    """Calcule la confiance de l'estimation basée sur les clusters.
    
    Args:
        clusters: Liste de clusters temporels.
        total_captures: Nombre total de captures individuelles.
        
    Returns:
        "high", "medium", ou "low".
        
    Logic:
        - high : Clusters bien définis (avg size >= 2)
        - medium : Clusters moyens (avg size < 2 mais > 1)
        - low : Pas de multi-captures (avg size = 1)
    """
    if not clusters:
        return "low"
    
    # Taille moyenne des clusters
    avg_cluster_size = total_captures / len(clusters)
    
    # Nombre de clusters avec multi-captures
    multi_capture_clusters = sum(1 for c in clusters if len(c) >= 2)
    multi_capture_ratio = multi_capture_clusters / len(clusters)
    
    # Calcul de confiance
    if avg_cluster_size >= 2.0 and multi_capture_ratio >= 0.5:
        return "high"
    elif avg_cluster_size >= 1.5 or multi_capture_ratio >= 0.3:
        return "medium"
    else:
        return "low"


def get_base_control_summary(
    timeline: pl.DataFrame,
    mode: Literal["CTF", "Strongholds", "Oddball"] = "Strongholds",
) -> dict[str, any]:
    """Résumé du contrôle de bases par les deux équipes.
    
    Args:
        timeline: DataFrame retourné par estimate_objective_captures_timeline().
        mode: Mode de jeu.
        
    Returns:
        Dictionnaire avec:
        - team_0_bases (int): Bases contrôlées par équipe 0
        - team_1_bases (int): Bases contrôlées par équipe 1
        - team_0_confidence (str): Confiance estimation équipe 0
        - team_1_confidence (str): Confiance estimation équipe 1
        - dominant_team (int | None): Équipe dominante (ou None si égal)
        - total_unique_bases (int): Total bases capturées (union des 2 équipes)
        
    Example:
        >>> summary = get_base_control_summary(timeline, "Strongholds")
        >>> print(f"Équipe {summary['dominant_team']} domine avec {summary['team_0_bases']} bases")
        Équipe 0 domine avec 3 bases
    """
    if timeline.is_empty():
        return {
            "team_0_bases": 0,
            "team_1_bases": 0,
            "team_0_confidence": "low",
            "team_1_confidence": "low",
            "dominant_team": None,
            "total_unique_bases": 0,
        }
    
    # Calculer le contrôle par équipe
    control = estimate_team_base_control(timeline, mode)
    
    # Extraire les valeurs par équipe
    team_0 = control.filter(pl.col("team_id") == 0)
    team_1 = control.filter(pl.col("team_id") == 1)
    
    team_0_bases = team_0["estimated_unique_bases"][0] if not team_0.is_empty() else 0
    team_1_bases = team_1["estimated_unique_bases"][0] if not team_1.is_empty() else 0
    
    team_0_conf = team_0["confidence"][0] if not team_0.is_empty() else "low"
    team_1_conf = team_1["confidence"][0] if not team_1.is_empty() else "low"
    
    # Équipe dominante
    if team_0_bases > team_1_bases:
        dominant = 0
    elif team_1_bases > team_0_bases:
        dominant = 1
    else:
        dominant = None
    
    return {
        "team_0_bases": team_0_bases,
        "team_1_bases": team_1_bases,
        "team_0_confidence": team_0_conf,
        "team_1_confidence": team_1_conf,
        "dominant_team": dominant,
        "total_unique_bases": team_0_bases + team_1_bases,  # Approximation
    }
