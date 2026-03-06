"""Helpers DRY pour les requêtes de matchs.

Centralise les patterns répétés dans MatchQueriesMixin :
- Résolution du contexte SQL (source, jointures, expressions)
- Construction de la clause SELECT standard (26 colonnes)
- Conversion DuckDB result → list[MatchRow]
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from src.data.domain.models.stats import MatchRow


@dataclass(frozen=True)
class QueryContext:
    """Contexte SQL résolu pour les requêtes de matchs."""

    source_sql: str
    source_params: list[Any] = field(default_factory=list)
    uses_mv: bool = False
    is_shared: bool = False
    metadata_joins: str = ""
    map_name_expr: str = "match_stats.map_name"
    playlist_name_expr: str = "match_stats.playlist_name"
    pair_name_expr: str = "match_stats.pair_name"
    pms_join: str = ""
    team_mmr_expr: str = "match_stats.team_mmr"
    enemy_mmr_expr: str = "match_stats.enemy_mmr"
    personal_score_select: str = "match_stats.personal_score"


def resolve_query_context(repo: Any, conn: Any) -> QueryContext:
    """Résout le contexte SQL complet pour requêtes de matchs.

    Remplace le bloc de 20 lignes dupliqué 5× dans MatchQueriesMixin.

    Args:
        repo: Instance DuckDBRepository (fournit _get_match_source, etc.).
        conn: Connexion DuckDB active.

    Returns:
        QueryContext avec toutes les expressions SQL résolues.
    """
    source_sql, source_params, uses_mv = repo._get_match_source(conn)
    is_shared = bool(source_params)

    if uses_mv:
        meta_kwargs: dict[str, str] = {
            "metadata_joins": "",
            "map_name_expr": "match_stats.map_name",
            "playlist_name_expr": "match_stats.playlist_name",
            "pair_name_expr": "match_stats.pair_name",
            "pms_join": "",
            "team_mmr_expr": "match_stats.team_mmr",
            "enemy_mmr_expr": "match_stats.enemy_mmr",
        }
    else:
        mj, mn, pn, ppn = repo._build_metadata_resolution(conn)
        pj, tm, em = repo._build_mmr_fallback(conn)
        meta_kwargs = {
            "metadata_joins": mj,
            "map_name_expr": mn,
            "playlist_name_expr": pn,
            "pair_name_expr": ppn,
            "pms_join": pj,
            "team_mmr_expr": tm,
            "enemy_mmr_expr": em,
        }

    if is_shared:
        ps = "match_stats.personal_score"
    else:
        ps = repo._select_optional_column(
            conn,
            table_name="match_stats",
            table_alias="match_stats",
            column_name="personal_score",
        )

    return QueryContext(
        source_sql=source_sql,
        source_params=source_params,
        uses_mv=uses_mv,
        is_shared=is_shared,
        personal_score_select=ps,
        **meta_kwargs,
    )


def build_match_select(
    ctx: QueryContext,
    *,
    coalesce_stats: bool = False,
    direct_names: bool = False,
) -> str:
    """Construit la clause SELECT standard pour les matchs (26 colonnes).

    Args:
        ctx: Contexte SQL résolu.
        coalesce_stats: Si True, COALESCE(kills/deaths/assists, 0).
        direct_names: Si True, utilise les noms directs (fallback sans joins).

    Returns:
        Expression SQL SELECT (sans le mot-clé SELECT).
    """
    mn = "match_stats.map_name" if direct_names else ctx.map_name_expr
    pn = "match_stats.playlist_name" if direct_names else ctx.playlist_name_expr
    ppn = "match_stats.pair_name" if direct_names else ctx.pair_name_expr
    tmr = "match_stats.team_mmr" if direct_names else ctx.team_mmr_expr
    emr = "match_stats.enemy_mmr" if direct_names else ctx.enemy_mmr_expr

    if coalesce_stats:
        k_expr = "COALESCE(match_stats.kills, 0) as kills"
        d_expr = "COALESCE(match_stats.deaths, 0) as deaths"
        a_expr = "COALESCE(match_stats.assists, 0) as assists"
    else:
        k_expr = "match_stats.kills"
        d_expr = "match_stats.deaths"
        a_expr = "match_stats.assists"

    return f"""\
                match_stats.match_id,
                match_stats.start_time,
                match_stats.map_id,
                {mn} as map_name,
                match_stats.playlist_id,
                {pn} as playlist_name,
                match_stats.pair_id,
                {ppn} as pair_name,
                match_stats.game_variant_id,
                match_stats.game_variant_name,
                match_stats.outcome,
                match_stats.team_id,
                match_stats.kda,
                match_stats.max_killing_spree,
                match_stats.headshot_kills,
                match_stats.avg_life_seconds,
                match_stats.time_played_seconds,
                {k_expr},
                {d_expr},
                {a_expr},
                match_stats.accuracy,
                match_stats.my_team_score,
                match_stats.enemy_team_score,
                {tmr} as team_mmr,
                {emr} as enemy_mmr,
                {ctx.personal_score_select}"""


def result_to_match_rows(result: Any) -> list[MatchRow]:
    """Convertit un résultat DuckDB en liste de MatchRow.

    Remplace le bloc de 30 lignes dupliqué 4× dans MatchQueriesMixin.
    """
    rows = result.fetchall()
    if not rows:
        return []
    columns = [desc[0] for desc in result.description]
    col_idx = {col: i for i, col in enumerate(columns)}
    return [_row_to_matchrow(row, col_idx) for row in rows]


def _row_to_matchrow(row: tuple, col_idx: dict[str, int]) -> MatchRow:
    """Convertit un tuple DuckDB en MatchRow via index pré-calculé."""
    return MatchRow(
        match_id=row[col_idx["match_id"]],
        start_time=row[col_idx["start_time"]],
        map_id=row[col_idx["map_id"]],
        map_name=row[col_idx["map_name"]],
        playlist_id=row[col_idx["playlist_id"]],
        playlist_name=row[col_idx["playlist_name"]],
        map_mode_pair_id=row[col_idx["pair_id"]],
        map_mode_pair_name=row[col_idx["pair_name"]],
        game_variant_id=row[col_idx["game_variant_id"]],
        game_variant_name=row[col_idx["game_variant_name"]],
        outcome=row[col_idx["outcome"]],
        last_team_id=row[col_idx["team_id"]],
        kda=row[col_idx["kda"]],
        max_killing_spree=row[col_idx["max_killing_spree"]],
        headshot_kills=row[col_idx["headshot_kills"]],
        average_life_seconds=row[col_idx["avg_life_seconds"]],
        time_played_seconds=row[col_idx["time_played_seconds"]],
        kills=row[col_idx["kills"]] or 0,
        deaths=row[col_idx["deaths"]] or 0,
        assists=row[col_idx["assists"]] or 0,
        accuracy=row[col_idx["accuracy"]],
        my_team_score=row[col_idx["my_team_score"]],
        enemy_team_score=row[col_idx["enemy_team_score"]],
        team_mmr=row[col_idx["team_mmr"]],
        enemy_mmr=row[col_idx["enemy_mmr"]],
        personal_score=row[col_idx["personal_score"]] if "personal_score" in col_idx else None,
    )
