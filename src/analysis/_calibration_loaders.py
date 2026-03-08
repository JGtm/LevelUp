"""Chargement des données pour la calibration LUSR.

Fonctions utilitaires d'accès base de données et de résolution de chemin.
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl

logger = logging.getLogger(__name__)


def _detect_shared_db(db_path: Path, explicit_path: str | Path | None) -> Path:
    """Résout le chemin shared_matches.duckdb (auto-détecté ou explicite)."""
    if explicit_path is not None:
        p = Path(explicit_path)
        if not p.exists():
            raise FileNotFoundError(
                "shared_matches.duckdb introuvable. Vérifiez le chemin ou utilisez --shared-db."
            )
        return p
    candidates_paths = [
        db_path.parent.parent.parent / "warehouse" / "shared_matches.duckdb",
        Path(__file__).resolve().parents[2] / "data" / "warehouse" / "shared_matches.duckdb",
    ]
    found = next((p for p in candidates_paths if p.exists()), None)
    if found is None:
        raise FileNotFoundError(
            "shared_matches.duckdb introuvable. Vérifiez le chemin ou utilisez --shared-db."
        )
    return found


def _resolve_xuid_from_gamertag(gamertag: str, shared_db: Path) -> str | None:
    """Résout le XUID depuis le gamertag via xuid_aliases."""
    from src.utils.db import duckdb_read_only

    if not shared_db.exists():
        return None
    with duckdb_read_only(shared_db) as conn:
        try:
            row = conn.execute(
                "SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
                [gamertag],
            ).fetchone()
            return str(row[0]) if row and row[0] else None
        except Exception:
            return None


def _compute_individual_mmr_map(
    df_matches: pl.DataFrame,
    df_participants: pl.DataFrame,
) -> dict[str, float]:
    """Calcule individual_mmr = team_mmr × (ke_joueur / ke_moyen_match)."""
    ke_avg_by_match: dict[str, float] = {}
    for row in (
        df_participants.filter(
            pl.col("kills_expected").is_not_null() & (pl.col("kills_expected") > 0)
        )
        .group_by("match_id")
        .agg(pl.col("kills_expected").mean().alias("ke_avg"))
        .iter_rows(named=True)
    ):
        ke_avg_by_match[row["match_id"]] = row["ke_avg"]

    individual_mmr_map: dict[str, float] = {}
    for row in df_matches.iter_rows(named=True):
        mid = row["match_id"]
        team_mmr = row["team_mmr"]
        ke_me = row["kills_expected"]
        ke_avg = ke_avg_by_match.get(mid)
        if team_mmr is None or ke_me is None or ke_avg is None or ke_avg <= 0:
            continue
        individual_mmr_map[mid] = team_mmr * (ke_me / ke_avg)
    return individual_mmr_map


def _load_matches_for_calibration(
    shared_db: Path,
    xuid: str,
    *,
    min_matches_with_mmr: int = 30,
) -> tuple[pl.DataFrame, pl.DataFrame, dict[str, float]]:
    """Charge les matchs, participants et MMR individuel décorrélé.

    Raises:
        ValueError: si moins de min_matches_with_mmr matchs ont team_mmr + kills_expected.
    """
    from src.utils.db import duckdb_read_only

    with duckdb_read_only(shared_db) as conn:
        df_matches = conn.execute(
            """
            SELECT
                mp.match_id,
                mr.start_time,
                COALESCE(mr.playlist_name, '') AS playlist_name,
                COALESCE(mr.pair_name, '')     AS pair_name,
                COALESCE(mp.outcome, 3)        AS outcome,
                COALESCE(mp.kills, 0)          AS kills,
                COALESCE(mp.deaths, 1)         AS deaths,
                COALESCE(mp.assists, 0)        AS assists,
                mp.kills_expected,
                mp.deaths_expected,
                mp.damage_dealt,
                mp.damage_taken,
                mp.accuracy,
                mp.team_id,
                mp.team_mmr,
                mp.enemy_mmr,
                COALESCE(mr.is_ranked, FALSE)  AS is_ranked,
                COALESCE(mr.is_firefight, FALSE) AS is_firefight
            FROM match_participants mp
            JOIN match_registry mr ON mr.match_id = mp.match_id
            WHERE mp.xuid = ?
              AND COALESCE(mr.is_firefight, FALSE) = FALSE
            ORDER BY mr.start_time ASC
            """,
            [xuid],
        ).pl()

        if df_matches.is_empty():
            raise ValueError(f"Aucun match trouvé pour XUID {xuid}.")

        match_ids = df_matches["match_id"].to_list()
        placeholders = ", ".join(["?"] * len(match_ids))
        df_participants = conn.execute(
            f"""
            SELECT
                match_id,
                xuid::TEXT AS xuid,
                team_id,
                kills_expected,
                deaths_expected
            FROM match_participants
            WHERE match_id IN ({placeholders})
            """,
            match_ids,
        ).pl()

        individual_mmr_map = _compute_individual_mmr_map(df_matches, df_participants)

        n_with_mmr = len(individual_mmr_map)
        if n_with_mmr < min_matches_with_mmr:
            raise ValueError(
                f"Seulement {n_with_mmr} matchs avec individual_mmr calculable "
                f"(minimum {min_matches_with_mmr}). "
                "Effectuez un backfill --skill pour enrichir les données."
            )

        return df_matches, df_participants, individual_mmr_map
