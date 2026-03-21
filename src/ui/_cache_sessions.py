"""Cache Streamlit — Calcul de sessions.

Fonction cached_compute_sessions_db extraite de cache_filters.py.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.analysis import compute_sessions_with_context_polars
from src.config import SESSION_CONFIG
from src.ui.cache_loaders import _is_duckdb_v4_path

logger = logging.getLogger(__name__)


@st.cache_data(show_spinner=False)
def cached_compute_sessions_db(  # noqa: C901, PLR0912, PLR0913, PLR0915
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    include_firefight: bool,
    gap_minutes: int,
    friends_xuids: tuple[str, ...] | None = None,
) -> pl.DataFrame:
    """Compute sessions sur la base (cache) avec logique avancée (gap + coéquipiers).

    Si friends_xuids est fourni, mode amis : seuls les amis déclenchent une
    nouvelle session (randoms matchmaking ignorés).
    """
    friends_set = frozenset(friends_xuids) if friends_xuids else None

    # DuckDB v4 : lecture hybride (stocké si stable, sinon calcul à la volée)
    if _is_duckdb_v4_path(db_path):
        try:
            from datetime import datetime, timezone

            from src.utils.db import duckdb_read_only
            from src.utils.paths import get_shared_matches_path_from_player

            with duckdb_read_only(db_path) as conn:
                # Attacher shared_matches.duckdb pour accéder aux données partagées
                shared_attached = False
                shared_path = get_shared_matches_path_from_player(db_path)
                if shared_path and shared_path.exists():
                    # Vérifier si déjà attaché
                    try:
                        dbs = conn.execute(
                            "SELECT database_name FROM duckdb_databases()"
                        ).fetchall()
                        existing_dbs = {db[0].lower() for db in dbs if db[0]}
                        if "shared" in existing_dbs:
                            shared_attached = True
                        else:
                            conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
                            shared_attached = True
                    except Exception:
                        pass

                df_pl = None

                if shared_attached:
                    # Production v5 : données dans shared + player_match_enrichment
                    firefight_filter = "" if include_firefight else "AND r.is_firefight = FALSE"

                    player_xuid = xuid.strip()
                    if not player_xuid:
                        try:
                            row = conn.execute(
                                "SELECT value FROM sync_meta WHERE key = 'xuid'"
                            ).fetchone()
                            if row:
                                player_xuid = str(row[0]).strip()
                        except Exception:
                            pass

                    if not player_xuid:
                        logger.warning("Impossible de résoudre XUID pour %s", db_path)
                    else:
                        query = f"""
                            SELECT
                                r.match_id,
                                r.start_time,
                                COALESCE(
                                    e.teammates_signature,
                                    (
                                        SELECT string_agg(sp.xuid, ',' ORDER BY sp.xuid)
                                        FROM shared.match_participants sp
                                        WHERE sp.match_id = r.match_id
                                          AND sp.xuid IS NOT NULL
                                          AND sp.xuid <> p.xuid
                                    )
                                ) AS teammates_signature,
                                e.is_with_friends,
                                e.session_id,
                                e.session_label
                            FROM shared.match_registry r
                            INNER JOIN shared.match_participants p
                                ON r.match_id = p.match_id
                                AND p.xuid = ?
                            LEFT JOIN player_match_enrichment e
                                ON r.match_id = e.match_id
                            WHERE r.start_time IS NOT NULL
                            {firefight_filter}
                            ORDER BY r.start_time ASC
                        """
                        try:
                            df_pl = conn.execute(query, [player_xuid]).pl()
                            logger.debug(
                                "Chargé %d matchs pour xuid=%s",
                                len(df_pl) if df_pl is not None else 0,
                                player_xuid,
                            )
                        except Exception as e:
                            logger.warning("Erreur chargement matchs depuis shared: %s", e)
                            df_pl = None
                else:
                    logger.info("shared_matches.duckdb non attaché pour %s", db_path)

                if df_pl is None:
                    # v5.1 : pas de fallback local (match_stats supprimée des DBs individuelles)
                    # Retourner un DataFrame vide si shared non disponible
                    logger.debug("Aucune donnée chargée, retour DataFrame vide")
                    df_pl = pl.DataFrame(
                        schema={
                            "match_id": pl.Utf8,
                            "start_time": pl.Datetime,
                            "teammates_signature": pl.Utf8,
                            "is_with_friends": pl.Boolean,
                            "session_id": pl.Utf8,
                            "session_label": pl.Utf8,
                        }
                    )

            if df_pl is None or df_pl.is_empty():
                logger.debug("DataFrame vide après chargement")
                return pl.DataFrame(
                    schema={
                        "match_id": pl.Utf8,
                        "start_time": pl.Datetime,
                        "is_with_friends": pl.Boolean,
                        "session_id": pl.Utf8,
                        "session_label": pl.Utf8,
                    }
                )

            # Cas A : tous ont session_id et sont >= 4h → utiliser stocké
            # Cas B : au moins un NULL ou récent → calcul complet à la volée
            stability_hours = SESSION_CONFIG.session_stability_hours
            threshold = datetime.now(timezone.utc).timestamp() - (stability_hours * 3600)
            has_null_session = df_pl.filter(pl.col("session_id").is_null()).height > 0
            df_pl = df_pl.with_columns(pl.col("start_time").dt.epoch(time_unit="s").alias("_ts"))
            has_recent = df_pl.filter(pl.col("_ts") > threshold).height > 0
            df_pl = df_pl.drop("_ts")

            if not has_null_session and not has_recent:
                # Cas A : tout stable, retourner tel quel
                cols_cas_a = ["match_id", "start_time", "session_id", "session_label"]
                if "teammates_signature" in df_pl.columns:
                    cols_cas_a.append("teammates_signature")
                if "is_with_friends" in df_pl.columns:
                    cols_cas_a.append("is_with_friends")
                return df_pl.select(cols_cas_a)

            # Cas B : calcul complet à la volée
            meta_cols = ["match_id"]
            if "is_with_friends" in df_pl.columns:
                meta_cols.append("is_with_friends")
            meta_df = df_pl.select(meta_cols).unique(subset=["match_id"])

            df_pl = compute_sessions_with_context_polars(
                df_pl.select(["match_id", "start_time", "teammates_signature"]),
                gap_minutes=gap_minutes,
                teammates_column="teammates_signature",
                friends_xuids=friends_set,
            )
            if "is_with_friends" in meta_df.columns:
                df_pl = df_pl.join(meta_df, on="match_id", how="left")

            cols_cas_b = ["match_id", "start_time", "session_id", "session_label"]
            if "teammates_signature" in df_pl.columns:
                cols_cas_b.append("teammates_signature")
            if "is_with_friends" in df_pl.columns:
                cols_cas_b.append("is_with_friends")
            return df_pl.select(cols_cas_b)

        except Exception as e:
            logger.warning("Erreur calcul sessions Polars: %s", e)
            # En cas d'erreur, retourner les données sans calcul de session si disponibles
            if df_pl is not None and not df_pl.is_empty():
                # Assurer que les colonnes session existent (même vides)
                if "session_id" not in df_pl.columns:
                    df_pl = df_pl.with_columns(
                        [
                            pl.lit(None).cast(pl.Utf8).alias("session_id"),
                            pl.lit(None).cast(pl.Utf8).alias("session_label"),
                        ]
                    )
                cols_err = ["match_id", "start_time", "session_id", "session_label"]
                if "teammates_signature" in df_pl.columns:
                    cols_err.append("teammates_signature")
                if "is_with_friends" in df_pl.columns:
                    cols_err.append("is_with_friends")
                return df_pl.select(cols_err)

    # Si on arrive ici, c'est qu'on n'a pas de données v5
    # Retourner un DataFrame vide plutôt que de risquer un conflit de connexion
    return pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "start_time": pl.Datetime,
            "is_with_friends": pl.Boolean,
            "session_id": pl.Utf8,
            "session_label": pl.Utf8,
        }
    )
