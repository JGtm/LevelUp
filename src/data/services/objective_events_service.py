"""Service pour extraire et analyser les événements d'objectifs (CTF, Strongholds, Oddball).

Ce module fournit des fonctions pour :
- Extraire les événements d'objectifs par équipe
- Identifier les joueurs ayant contribué aux objectifs
- Calculer des statistiques d'équipe
- Préparer les données pour les visualisations

Exemples:
    >>> from src.data.services.objective_events_service import get_flag_captures_by_team
    >>> df = get_flag_captures_by_team(repo, "match_abc123")
    >>> print(df)
    
    team_id  award_name       total_count  total_score
    0        Flag Captured    3            900
    1        Flag Captured    2            600
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import polars as pl

if TYPE_CHECKING:
    from src.data.repositories.duckdb_repo import DuckDBRepository


def get_objective_events_by_team(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne tous les événements d'objectifs agrégés par équipe.
    
    Agrège tous les événements catégorisés comme 'objective' depuis
    personal_score_awards, en les groupant par équipe.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - team_id (int): ID de l'équipe (0 ou 1)
        - award_name (str): Nom de l'événement ("Flag Captured", etc.)
        - total_count (int): Nombre total d'occurrences
        - total_score (int): Points totaux gagnés
        
    Example:
        >>> df = get_objective_events_by_team(repo, "match123")
        >>> print(df)
        team_id  award_name         total_count  total_score
        0        Flag Captured      3            900
        0        Flag Stolen        1            25
        1        Flag Captured      2            600
    """
    query = """
    SELECT 
        mp.team_id,
        psa.award_name,
        SUM(psa.award_count) AS total_count,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_category = 'objective'
    GROUP BY mp.team_id, psa.award_name
    ORDER BY mp.team_id, total_score DESC
    """
    return repo.query_df(query, [match_id])


def get_flag_captures_by_team(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne les captures de drapeaux (CTF) par équipe.
    
    Filtre uniquement les événements FLAG_CAPTURED pour le mode CTF.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match CTF à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - team_id (int): ID de l'équipe
        - total_captures (int): Nombre de drapeaux capturés
        - total_score (int): Points gagnés
        
    Example:
        >>> df = get_flag_captures_by_team(repo, "match_ctf_123")
        >>> print(df)
        team_id  total_captures  total_score
        0        3               900
        1        2               600
    """
    query = """
    SELECT 
        mp.team_id,
        SUM(psa.award_count) AS total_captures,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name = 'Flag Captured'
    GROUP BY mp.team_id
    ORDER BY total_captures DESC
    """
    return repo.query_df(query, [match_id])


def get_flag_captures_by_player(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne les captures de drapeaux (CTF) par joueur.
    
    Détaille les contributeurs individuels aux captures de drapeaux.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match CTF à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - gamertag (str): Nom du joueur
        - team_id (int): ID de l'équipe
        - flag_captured (int): Nombre de drapeaux capturés
        - flag_stolen (int): Nombre de drapeaux ennemis volés
        - flag_returned (int): Nombre de drapeaux alliés ramenés
        - flag_capture_assist (int): Nombre d'assistances à la capture
        - total_score (int): Points totaux gagnés sur objectifs CTF
        
    Example:
        >>> df = get_flag_captures_by_player(repo, "match_ctf_123")
        >>> print(df.head(3))
        gamertag     team_id  flag_captured  flag_stolen  total_score
        JohnSpartan  0        3              1            1025
        MasterChief  0        1              2            350
        Arbiter      1        2              0            600
    """
    query = """
    SELECT 
        mp.gamertag,
        mp.team_id,
        SUM(CASE WHEN psa.award_name = 'Flag Captured' THEN psa.award_count ELSE 0 END) AS flag_captured,
        SUM(CASE WHEN psa.award_name = 'Flag Stolen' THEN psa.award_count ELSE 0 END) AS flag_stolen,
        SUM(CASE WHEN psa.award_name = 'Flag Returned' THEN psa.award_count ELSE 0 END) AS flag_returned,
        SUM(CASE WHEN psa.award_name = 'Flag Capture Assist' THEN psa.award_count ELSE 0 END) AS flag_capture_assist,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name IN (
          'Flag Captured', 'Flag Stolen', 'Flag Returned', 
          'Flag Taken', 'Flag Capture Assist', 'Runner Stopped'
      )
    GROUP BY mp.xuid, mp.gamertag, mp.team_id
    ORDER BY total_score DESC
    """
    return repo.query_df(query, [match_id])


def get_base_captures_by_team(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne les captures de bases (Strongholds) par équipe.
    
    Agrège les événements Zone Captured (50%, 75%, 100%) par équipe.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match Strongholds à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - team_id (int): ID de l'équipe
        - zone_50_count (int): Captures à 50%
        - zone_75_count (int): Captures à 75%
        - zone_100_count (int): Captures à 100%
        - total_captures (int): Total toutes captures
        - total_score (int): Points totaux
        
    Example:
        >>> df = get_base_captures_by_team(repo, "match_sh_123")
        >>> print(df)
        team_id  zone_50_count  zone_75_count  zone_100_count  total_score
        0        8              6              5               1375
        1        7              5              4               1200
    """
    query = """
    SELECT 
        mp.team_id,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 50%' THEN psa.award_count ELSE 0 END) AS zone_50_count,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 75%' THEN psa.award_count ELSE 0 END) AS zone_75_count,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 100%' THEN psa.award_count ELSE 0 END) AS zone_100_count,
        SUM(psa.award_count) AS total_captures,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name IN (
          'Zone Captured 50%', 'Zone Captured 75%', 
          'Zone Captured 100%', 'Zone Secured'
      )
    GROUP BY mp.team_id
    ORDER BY total_score DESC
    """
    return repo.query_df(query, [match_id])


def get_base_captures_by_player(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne les captures de bases (Strongholds) par joueur.
    
    Détaille les contributions individuelles aux captures de bases.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match Strongholds à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - gamertag (str): Nom du joueur
        - team_id (int): ID de l'équipe
        - zone_50_count (int): Captures à 50%
        - zone_75_count (int): Captures à 75%
        - zone_100_count (int): Captures à 100%
        - zone_secured_count (int): Bases sécurisées
        - total_score (int): Points totaux
        
    Example:
        >>> df = get_base_captures_by_player(repo, "match_sh_123")
        >>> print(df.head(3))
        gamertag     team_id  zone_100_count  total_score
        PlayerA      0        3               300
        PlayerB      0        2               200
        PlayerC      1        2               200
    """
    query = """
    SELECT 
        mp.gamertag,
        mp.team_id,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 50%' THEN psa.award_count ELSE 0 END) AS zone_50_count,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 75%' THEN psa.award_count ELSE 0 END) AS zone_75_count,
        SUM(CASE WHEN psa.award_name = 'Zone Captured 100%' THEN psa.award_count ELSE 0 END) AS zone_100_count,
        SUM(CASE WHEN psa.award_name = 'Zone Secured' THEN psa.award_count ELSE 0 END) AS zone_secured_count,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name IN (
          'Zone Captured 50%', 'Zone Captured 75%', 
          'Zone Captured 100%', 'Zone Secured'
      )
    GROUP BY mp.xuid, mp.gamertag, mp.team_id
    ORDER BY total_score DESC
    """
    return repo.query_df(query, [match_id])


def get_oddball_control_by_team(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne le contrôle de la balle (Oddball) par équipe.
    
    Agrège les événements Ball Control par équipe.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match Oddball à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - team_id (int): ID de l'équipe
        - ball_control_count (int): Nombre d'événements Ball Control
        - ball_taken_count (int): Nombre de fois la balle a été prise
        - carrier_stopped_count (int): Porteurs ennemis éliminés
        - total_score (int): Points totaux
        
    Note:
        ball_control_count n'est PAS une durée exacte, mais un indicateur
        de contrôle relatif (l'API génère ces événements périodiquement).
        
    Example:
        >>> df = get_oddball_control_by_team(repo, "match_ob_123")
        >>> print(df)
        team_id  ball_control_count  total_score
        0        12                  600
        1        8                   400
    """
    query = """
    SELECT 
        mp.team_id,
        SUM(CASE WHEN psa.award_name = 'Ball Control' THEN psa.award_count ELSE 0 END) AS ball_control_count,
        SUM(CASE WHEN psa.award_name = 'Ball Taken' THEN psa.award_count ELSE 0 END) AS ball_taken_count,
        SUM(CASE WHEN psa.award_name = 'Carrier Stopped' THEN psa.award_count ELSE 0 END) AS carrier_stopped_count,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name IN ('Ball Control', 'Ball Taken', 'Carrier Stopped')
    GROUP BY mp.team_id
    ORDER BY total_score DESC
    """
    return repo.query_df(query, [match_id])


def get_oddball_control_by_player(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame:
    """Retourne le contrôle de la balle (Oddball) par joueur.
    
    Détaille les contributions individuelles au contrôle de la balle.
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match Oddball à analyser.
        
    Returns:
        DataFrame Polars avec colonnes:
        - gamertag (str): Nom du joueur
        - team_id (int): ID de l'équipe
        - ball_control_count (int): Événements Ball Control
        - ball_taken_count (int): Balle prise
        - carrier_stopped_count (int): Porteurs ennemis éliminés
        - total_score (int): Points totaux
        
    Example:
        >>> df = get_oddball_control_by_player(repo, "match_ob_123")
        >>> print(df.head(3))
        gamertag     team_id  ball_control_count  total_score
        PlayerX      0        8                   400
        PlayerY      0        4                   200
        PlayerZ      1        6                   300
    """
    query = """
    SELECT 
        mp.gamertag,
        mp.team_id,
        SUM(CASE WHEN psa.award_name = 'Ball Control' THEN psa.award_count ELSE 0 END) AS ball_control_count,
        SUM(CASE WHEN psa.award_name = 'Ball Taken' THEN psa.award_count ELSE 0 END) AS ball_taken_count,
        SUM(CASE WHEN psa.award_name = 'Carrier Stopped' THEN psa.award_count ELSE 0 END) AS carrier_stopped_count,
        SUM(psa.award_score) AS total_score
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_name IN ('Ball Control', 'Ball Taken', 'Carrier Stopped')
    GROUP BY mp.xuid, mp.gamertag, mp.team_id
    ORDER BY total_score DESC
    """
    return repo.query_df(query, [match_id])


def get_objective_mvp(
    repo: DuckDBRepository,
    match_id: str,
) -> pl.DataFrame | None:
    """Identifie le joueur MVP sur les objectifs du match.
    
    Retourne le joueur ayant contribué le plus aux objectifs
    (tous types confondus).
    
    Args:
        repo: Repository DuckDB.
        match_id: ID du match à analyser.
        
    Returns:
        DataFrame Polars avec UNE ligne contenant:
        - gamertag (str): Nom du MVP
        - team_id (int): Équipe du MVP
        - total_objective_score (int): Points objectifs totaux
        - total_actions (int): Nombre d'actions objectifs
        
        None si aucune donnée.
        
    Example:
        >>> mvp = get_objective_mvp(repo, "match123")
        >>> print(mvp)
        gamertag     team_id  total_objective_score  total_actions
        JohnSpartan  0        1025                   15
    """
    query = """
    SELECT 
        mp.gamertag,
        mp.team_id,
        SUM(psa.award_score) AS total_objective_score,
        SUM(psa.award_count) AS total_actions
    FROM personal_score_awards psa
    JOIN match_participants mp 
        ON psa.match_id = mp.match_id 
        AND psa.xuid = mp.xuid
    WHERE psa.match_id = ?
      AND psa.award_category = 'objective'
    GROUP BY mp.xuid, mp.gamertag, mp.team_id
    ORDER BY total_objective_score DESC
    LIMIT 1
    """
    df = repo.query_df(query, [match_id])
    return df if not df.is_empty() else None
