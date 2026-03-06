"""Chargement des médias pour l'affichage UI (Streamlit).

Fonctions extraites de ``MediaIndexer`` (``load_media_for_ui`` et
``load_media_for_match``), converties en fonctions libres.
"""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl

from src.data.media_helpers import get_all_player_dbs, get_gamertag_from_db_path
from src.utils.db import duckdb_flexible, has_table
from src.utils.paths import get_shared_matches_path_from_player

logger = logging.getLogger(__name__)


def load_media_for_ui(  # noqa: C901, PLR0912, PLR0915
    db_path: Path | str,
    current_xuid: str | None,
) -> pl.DataFrame:
    """Charge les médias actifs avec associations pour l'onglet Médias.

    Cross-DB : « Mes captures » depuis la DB courante ; « Captures de XXX »
    depuis les autres DB dont le match_id existe dans la DB courante.
    Une seule ligne par média : priorité mine > teammate > unassigned.

    Returns:
        Polars DataFrame avec colonnes: file_path, file_name, kind, thumbnail_path,
        capture_end_utc, map_name, match_id, match_start_time, xuid, gamertag, section.
        section ∈ {'mine', 'teammate', 'unassigned'}.
    """
    db_path = Path(db_path)
    if not db_path.exists():
        return pl.DataFrame()

    if not _check_media_tables(db_path):
        return pl.DataFrame()

    cu = str(current_xuid or "")
    current_resolved = db_path.resolve()
    player_dbs = get_all_player_dbs()

    match_ids_current = _load_current_match_ids(db_path, cu)
    df_current = _load_current_media(db_path)
    dfs_teammate = _load_teammate_media(player_dbs, current_resolved, match_ids_current)

    if df_current.is_empty() and not dfs_teammate:
        return pl.DataFrame()
    dfs = [df_current] + dfs_teammate
    df = pl.concat(dfs) if len(dfs) > 1 else dfs[0]

    df = _annotate_sections(df, db_path, player_dbs)
    return df.sort("capture_end_utc", descending=True, nulls_last=True)


def load_media_for_match(  # noqa: C901
    db_path: Path | str,
    match_id: str,
    current_xuid: str | None = None,  # noqa: ARG001
) -> pl.DataFrame:
    """Charge tous les médias associés à un match spécifique (toutes les DBs joueurs).

    Returns:
        Polars DataFrame avec colonnes: file_path, file_name, kind, thumbnail_path,
        capture_end_utc, map_name, match_id, xuid, gamertag, section.
        section ∈ {'mine', 'teammate'}.
    """
    db_path = Path(db_path)
    if not db_path.exists() or not match_id:
        return pl.DataFrame()

    player_dbs = get_all_player_dbs()
    if not player_dbs:
        return pl.DataFrame()

    current_gamertag = get_gamertag_from_db_path(db_path) or ""
    current_gamertag_lower = current_gamertag.lower()
    known_gamertags: set[str] = {Path(pdb).parent.name.lower() for pdb, _ in player_dbs}

    dfs: list[pl.DataFrame] = []
    for pdb, _xuid in player_dbs:
        try:
            with duckdb_flexible(pdb) as c:
                if not has_table(c, "media_files") or not has_table(c, "media_match_associations"):
                    continue
                df_p = c.execute(
                    """
                    SELECT mf.file_path, mf.file_name, mf.kind, mf.thumbnail_path,
                           mf.capture_end_utc, mma.map_name, mma.match_id, mma.xuid
                    FROM media_files mf
                    JOIN media_match_associations mma ON mf.file_path = mma.media_path
                    WHERE mf.status = 'active' AND mma.match_id = ?
                    """,
                    [match_id],
                ).pl()
                if not df_p.is_empty():
                    dfs.append(df_p)
        except Exception as e:
            logger.debug("load_media_for_match db %s: %s", pdb, e)

    if not dfs:
        return pl.DataFrame()

    df = pl.concat(dfs) if len(dfs) > 1 else dfs[0]
    df = df.unique(subset=["file_path"], keep="first")

    owners = [_extract_owner(p, known_gamertags, player_dbs) for p in df["file_path"].to_list()]
    df = df.with_columns(pl.Series("owner_gamertag", owners))

    df = df.with_columns(
        pl.when(pl.col("owner_gamertag").str.to_lowercase() == current_gamertag_lower)
        .then(pl.lit("mine"))
        .otherwise(pl.lit("teammate"))
        .alias("section")
    )
    df = df.with_columns(
        pl.when(pl.col("owner_gamertag") != "")
        .then(pl.col("owner_gamertag"))
        .otherwise(pl.col("xuid").cast(pl.Utf8))
        .alias("gamertag")
    )
    df = df.drop("owner_gamertag")
    return df.sort("capture_end_utc", descending=True, nulls_last=True)


# ---------------------------------------------------------------------------
#  Sous-fonctions internes
# ---------------------------------------------------------------------------


def _check_media_tables(db_path: Path) -> bool:
    """Vérifie que la table ``media_files`` existe (lecture flexible)."""
    for ro in (True, False):
        try:
            with duckdb_flexible(db_path) as c:
                return has_table(c, "media_files")
        except Exception as e:
            if ro:
                continue
            logger.warning("load_media_for_ui schema check: %s", e)
            return False
    return False


def _load_current_match_ids(db_path: Path, current_xuid: str) -> set[str]:
    """Charge les match_ids du joueur courant."""
    match_ids: set[str] = set()
    try:
        with duckdb_flexible(db_path) as c:
            if has_table(c, "player_match_enrichment"):
                rows = c.execute(
                    "SELECT DISTINCT match_id FROM player_match_enrichment "
                    "WHERE match_id IS NOT NULL"
                ).fetchall()
                match_ids = {str(r[0]).strip() for r in rows if r[0]}
    except Exception:
        pass

    if not match_ids:
        shared_path = get_shared_matches_path_from_player(db_path)
        if shared_path and shared_path.exists() and current_xuid:
            try:
                with duckdb_flexible(shared_path) as c:
                    rows = c.execute(
                        "SELECT DISTINCT mp.match_id FROM match_participants mp "
                        "WHERE mp.xuid = ? AND mp.match_id IS NOT NULL",
                        [current_xuid],
                    ).fetchall()
                    match_ids = {str(r[0]).strip() for r in rows if r[0]}
            except Exception:
                pass
    return match_ids


def _load_current_media(db_path: Path) -> pl.DataFrame:
    """Charge les médias de la DB courante (mine + unassigned)."""
    try:
        with duckdb_flexible(db_path) as c:
            if not has_table(c, "media_files"):
                return pl.DataFrame()
            return c.execute("""
                SELECT
                    mf.file_path, mf.file_name, mf.kind, mf.thumbnail_path,
                    mf.capture_end_utc, mma.map_name, mma.match_id, mma.match_start_time,
                    mma.xuid
                FROM media_files mf
                LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path
                WHERE mf.status = 'active'
            """).pl()
    except Exception as e:
        logger.warning("load_media_for_ui current: %s", e)
        return pl.DataFrame()


def _load_teammate_media(
    player_dbs: list[tuple[Path, str]],
    current_resolved: Path,
    match_ids_current: set[str],
) -> list[pl.DataFrame]:
    """Charge les médias des coéquipiers dont le match_id est dans le set courant."""
    dfs: list[pl.DataFrame] = []
    for pdb, _xuid in player_dbs:
        if Path(pdb).resolve() == current_resolved:
            continue
        if not match_ids_current:
            continue
        try:
            with duckdb_flexible(pdb) as c:
                if not has_table(c, "media_match_associations"):
                    continue
                placeholders = ",".join("?" for _ in match_ids_current)
                q = f"""
                    SELECT mf.file_path, mf.file_name, mf.kind, mf.thumbnail_path,
                           mf.capture_end_utc, mma.map_name, mma.match_id, mma.match_start_time,
                           mma.xuid
                    FROM media_files mf
                    JOIN media_match_associations mma ON mf.file_path = mma.media_path
                    WHERE mf.status = 'active' AND mma.match_id IN ({placeholders})
                """
                df_t = c.execute(q, list(match_ids_current)).pl()
                if not df_t.is_empty():
                    dfs.append(df_t)
        except Exception as e:
            logger.debug("load_media_for_ui other db %s: %s", pdb, e)
    return dfs


def _extract_owner(
    file_path: str,
    known_gamertags: set[str],
    player_dbs: list[tuple[Path, str]],
) -> str:
    """Extrait le gamertag propriétaire depuis le chemin du fichier."""
    if not file_path:
        return ""
    path_lower = file_path.lower().replace("\\", "/")
    for gt in known_gamertags:
        if f"/{gt}/" in path_lower or path_lower.endswith(f"/{gt}"):
            for pdb, _ in player_dbs:
                orig_gt = Path(pdb).parent.name
                if orig_gt.lower() == gt:
                    return orig_gt
    return ""


def _annotate_sections(
    df: pl.DataFrame,
    db_path: Path,
    player_dbs: list[tuple[Path, str]],
) -> pl.DataFrame:
    """Ajoute les colonnes *gamertag* et *section* puis déduplique."""
    current_gamertag = get_gamertag_from_db_path(db_path) or ""
    current_gamertag_lower = current_gamertag.lower()
    known_gamertags: set[str] = set()

    xuid_to_gamertag: dict[str, str] = {}
    for pdb, xu in player_dbs:
        gamertag = Path(pdb).parent.name
        xuid_to_gamertag[str(xu)] = gamertag
        known_gamertags.add(gamertag.lower())

    df = df.with_columns(
        pl.col("xuid")
        .cast(pl.Utf8)
        .replace_strict(
            xuid_to_gamertag, default=pl.col("xuid").cast(pl.Utf8), return_dtype=pl.Utf8
        )
        .fill_null("")
        .alias("gamertag")
    )

    owners = [_extract_owner(p, known_gamertags, player_dbs) for p in df["file_path"].to_list()]
    df = df.with_columns(pl.Series("owner_gamertag", owners))

    df = df.with_columns(
        pl.when(
            pl.col("match_id").is_null()
            | (pl.col("match_id").cast(pl.Utf8).str.strip_chars() == "")
        )
        .then(pl.lit("unassigned"))
        .when(pl.col("owner_gamertag").str.to_lowercase() == current_gamertag_lower)
        .then(pl.lit("mine"))
        .otherwise(pl.lit("teammate"))
        .alias("section")
    )

    df = df.with_columns(
        pl.when(pl.col("owner_gamertag") != "")
        .then(pl.col("owner_gamertag"))
        .otherwise(pl.col("gamertag"))
        .alias("gamertag")
    )
    df = df.drop("owner_gamertag")

    df = df.with_columns(
        pl.when(pl.col("section") == "mine")
        .then(0)
        .when(pl.col("section") == "teammate")
        .then(1)
        .otherwise(2)
        .alias("_section_rank")
    )
    df = df.sort(["file_path", "_section_rank", "gamertag"]).unique(
        subset=["file_path"], keep="first"
    )
    return df.drop("_section_rank")
