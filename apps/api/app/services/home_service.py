"""Service Accueil — Hero card, signaux, sessions, matchs récents, médias récents.

Endpoints :
  GET /players/{slug}/pages/home    → HomePageResponse
  GET /players/{slug}/battlepass    → BattlePassResponse
  GET /players/{slug}/challenges    → ChallengesResponse
"""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime
from pathlib import Path

from apps.api.app._db_helpers import (
    OUTCOME_LABELS,
    OUTCOME_TONES,
    Outcome,
    build_match_source_sql,
    has_mv_player_matches,
    resolve_xuid,
)
from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.home import (
    BattlePassResponse,
    ChallengesResponse,
    HeroKPIs,
    HeroTrend,
    HighlightItem,
    HomeHeroCard,
    HomePageResponse,
    RecentMatchItem,
    RecentMediaItem,
    SessionSummaryItem,
)

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def get_home_page(player: PlayerContext) -> HomePageResponse:
    """Retourne la page d'accueil Mission Control agrégée."""
    matches_df = _load_matches_home(player)
    sessions_df = _load_sessions(player)
    media_entries = _load_recent_media(player, limit=4)

    hero = _build_hero(player, matches_df)
    highlights = _build_highlights(matches_df, sessions_df)
    recent_matches = _build_recent_matches(matches_df, limit=6)
    solo_session = _build_session_summary(matches_df, sessions_df, squad_mode=False)
    squad_session = _build_session_summary(matches_df, sessions_df, squad_mode=True)

    return HomePageResponse(
        hero=hero,
        highlights=highlights,
        recent_matches=recent_matches,
        recent_media=media_entries,
        solo_session=solo_session,
        squad_session=squad_session,
    )


def get_battlepass(player: PlayerContext) -> BattlePassResponse:
    """Tente de récupérer les infos Battle Pass via l'API Halo (best-effort)."""
    try:
        return _fetch_battlepass_live(player)
    except Exception as exc:
        logger.debug("get_battlepass(%s): erreur %s", player.player_slug, exc)
        return BattlePassResponse(available=False, error_hint="live_unavailable")


def get_challenges(player: PlayerContext) -> ChallengesResponse:
    """Tente de récupérer les défis actifs via l'API Halo (best-effort)."""
    try:
        return _fetch_challenges_live(player)
    except Exception as exc:
        logger.debug("get_challenges(%s): erreur %s", player.player_slug, exc)
        return ChallengesResponse(available=False, error_hint="live_unavailable")


# ---------------------------------------------------------------------------
# Chargement DuckDB
# ---------------------------------------------------------------------------


def _load_matches_home(player: PlayerContext):  # type: ignore[return]
    """Charge les matchs récents avec toutes les colonnes KPI nécessaires."""
    try:
        import polars as pl

        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        shared_path = Path(player.shared_db_path)

        if not db_path.exists():
            return pl.DataFrame()

        with duckdb_read_only(str(db_path)) as conn:
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")

            xuid = resolve_xuid(conn)
            if not xuid:
                return pl.DataFrame()

            has_mv = has_mv_player_matches(conn)
            source_sql = build_match_source_sql(has_mv)

            result = conn.execute(
                f"""
                SELECT
                    ms.match_id,
                    ms.start_time,
                    ms.map_name,
                    COALESCE(ms.map_name_fr, ms.map_name)           AS map_name_fr,
                    ms.pair_name,
                    COALESCE(ms.pair_name_fr, ms.pair_name)         AS pair_name_fr,
                    ms.playlist_name,
                    COALESCE(ms.is_firefight, FALSE)                AS is_firefight,
                    COALESCE(ms.is_ranked, FALSE)                   AS is_ranked,
                    pme.session_label,
                    COALESCE(pme.is_with_friends, FALSE)            AS is_with_friends,
                    COALESCE(p.outcome, 0)                          AS outcome,
                    p.kills,
                    p.deaths,
                    p.assists,
                    p.kda,
                    CASE WHEN p.deaths > 0 THEN CAST(p.kills AS DOUBLE) / p.deaths
                         ELSE CAST(COALESCE(p.kills, 0) AS DOUBLE) END AS ratio,
                    p.accuracy,
                    p.average_life_seconds,
                    p.time_played_seconds,
                    p.my_team_score,
                    p.enemy_team_score
                FROM {source_sql} ms
                LEFT JOIN shared.match_participants p
                    ON ms.match_id = p.match_id AND p.xuid = ?
                LEFT JOIN player_match_enrichment pme
                    ON ms.match_id = pme.match_id
                ORDER BY ms.start_time DESC
                LIMIT 200
                """,
                [xuid, xuid] if "?" in source_sql else [xuid],
            )
            columns = [d[0] for d in result.description]
            rows = result.fetchall()

        if not rows:
            return pl.DataFrame()
        return pl.DataFrame(rows, schema=columns, orient="row")

    except Exception:
        logger.exception("_load_matches_home(%s)", player.player_slug)
        return pl.DataFrame()


def _load_sessions(player: PlayerContext):  # type: ignore[return]
    """Charge les sessions depuis player_match_enrichment."""
    try:
        import polars as pl

        from src.utils.db import duckdb_read_only

        db_path = Path(player.db_path)
        if not db_path.exists():
            return pl.DataFrame()

        with duckdb_read_only(str(db_path)) as conn:
            shared_path = Path(player.shared_db_path)
            if shared_path.exists():
                with contextlib.suppress(Exception):
                    conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
            try:
                result = conn.execute(
                    """
                    SELECT pme.match_id, pme.session_id, pme.session_label,
                           COALESCE(pme.is_with_friends, FALSE) AS is_with_friends,
                           r.start_time
                    FROM player_match_enrichment pme
                    LEFT JOIN shared.match_registry r ON r.match_id = pme.match_id
                    WHERE pme.session_label IS NOT NULL
                    ORDER BY r.start_time DESC
                    """
                )
                columns = [d[0] for d in result.description]
                rows = result.fetchall()
                if not rows:
                    return pl.DataFrame()
                return pl.DataFrame(rows, schema=columns, orient="row")
            except Exception:
                return pl.DataFrame()
    except Exception:
        logger.debug("_load_sessions(%s): erreur", player.player_slug, exc_info=True)
        return pl.DataFrame()


def _load_recent_media(player: PlayerContext, limit: int = 4) -> list[RecentMediaItem]:
    """Charge les médias récents depuis MediaIndexer."""
    try:
        from pathlib import Path as _Path

        from src.data.media_indexer import MediaIndexer

        db_path = _Path(player.db_path)
        media_df = MediaIndexer.load_media_for_ui(db_path, player.xuid)
        if hasattr(media_df, "is_empty") and media_df.is_empty():
            return []

        items: list[RecentMediaItem] = []
        for row in media_df.head(limit).iter_rows(named=True):
            basename = str(row.get("file_name") or row.get("basename") or "")
            if not basename:
                continue
            match_id = str(row.get("match_id") or "").strip() or None
            ms_time = row.get("match_start_time")
            if isinstance(ms_time, str):
                try:
                    ms_time = datetime.fromisoformat(ms_time)
                except Exception:
                    ms_time = None
            items.append(
                RecentMediaItem(basename=basename, match_id=match_id, match_start_time=ms_time)
            )
        return items
    except Exception:
        logger.debug("_load_recent_media: erreur", exc_info=True)
        return []


# ---------------------------------------------------------------------------
# Construction des blocs Hero
# ---------------------------------------------------------------------------


def _build_hero(player: PlayerContext, matches_df) -> HomeHeroCard:  # type: ignore[return]
    """Construit le bloc Hero card depuis les matchs chargés."""
    try:
        player_name = _resolve_gamertag(player)
        if hasattr(matches_df, "is_empty") and matches_df.is_empty():
            return HomeHeroCard(
                player_name=player_name,
                kpis=HeroKPIs(),
                trend=None,
            )

        kpis = _compute_kpis(matches_df)
        trend = _compute_trend(matches_df)
        return HomeHeroCard(player_name=player_name, kpis=kpis, trend=trend)
    except Exception:
        logger.debug("_build_hero: erreur", exc_info=True)
        return HomeHeroCard(player_name=player.gamertag or player.player_slug, kpis=HeroKPIs())


def _compute_kpis(matches_df) -> HeroKPIs:  # type: ignore[return]
    """Calcule les KPIs globaux."""
    try:
        import polars as pl

        total = len(matches_df)
        wins = 0
        if "outcome" in matches_df.columns:
            wins = int(matches_df["outcome"].cast(pl.Int64, strict=False).eq(Outcome.WIN).sum())
        win_rate = wins / total if total else 0.0

        global_ratio = None
        if "ratio" in matches_df.columns:
            vals = matches_df["ratio"].cast(pl.Float64, strict=False).drop_nulls()
            if not vals.is_empty():
                global_ratio = float(vals.mean())

        avg_accuracy = None
        if "accuracy" in matches_df.columns:
            vals = matches_df["accuracy"].cast(pl.Float64, strict=False).drop_nulls()
            if not vals.is_empty():
                avg_accuracy = float(vals.mean())

        losses = 0
        if "outcome" in matches_df.columns:
            losses = int(matches_df["outcome"].cast(pl.Int64, strict=False).eq(3).sum())

        return HeroKPIs(
            win_rate=round(win_rate, 4),
            global_ratio=round(global_ratio, 2) if global_ratio is not None else None,
            avg_accuracy=round(avg_accuracy, 1) if avg_accuracy is not None else None,
            total_matches=total,
            wins=wins,
            losses=losses,
        )
    except Exception:
        return HeroKPIs()


def _compute_trend(matches_df, window: int = 5) -> HeroTrend | None:
    """Calcule la tendance par fenêtre glissante."""
    try:
        import polars as pl

        if hasattr(matches_df, "is_empty") and matches_df.is_empty():
            return None
        if "start_time" in matches_df.columns:
            mdf = matches_df.sort("start_time", descending=True)
        else:
            mdf = matches_df
        current = mdf.head(window)
        previous = mdf.slice(window, window)
        if previous.is_empty():
            return None

        def _mean(df, col: str) -> float | None:
            if col not in df.columns:
                return None
            vals = df[col].cast(pl.Float64, strict=False).drop_nulls()
            return float(vals.mean()) if not vals.is_empty() else None

        def _wr(df) -> float:
            if "outcome" not in df.columns or df.is_empty():
                return 0.0
            return float(df["outcome"].cast(pl.Int64, strict=False).eq(Outcome.WIN).sum()) / len(df)

        ratio_delta = None
        cr, pr = _mean(current, "ratio"), _mean(previous, "ratio")
        if cr is not None and pr is not None:
            ratio_delta = round(cr - pr, 3)

        acc_delta = None
        ca, pa = _mean(current, "accuracy"), _mean(previous, "accuracy")
        if ca is not None and pa is not None:
            acc_delta = round(ca - pa, 2)

        wr_delta = round(_wr(current) - _wr(previous), 4)

        return HeroTrend(ratio_delta=ratio_delta, accuracy_delta=acc_delta, win_rate_delta=wr_delta)
    except Exception:
        return None


# ---------------------------------------------------------------------------
# Highlights
# ---------------------------------------------------------------------------


def _build_highlights(matches_df, sessions_df) -> list[HighlightItem]:
    """Construit les faits saillants récents."""
    try:
        if hasattr(matches_df, "is_empty") and matches_df.is_empty():
            return []

        highlights: list[HighlightItem] = []
        if "start_time" in matches_df.columns:
            mdf = matches_df.sort("start_time", descending=True)
        else:
            mdf = matches_df

        # Highlight 1 : pic KD récent
        recent_window = mdf.head(8)
        if "ratio" in recent_window.columns and not recent_window.is_empty():
            best = recent_window.sort("ratio", descending=True, nulls_last=True).row(0, named=True)
            best_ratio = f"{float(best['ratio']):.2f}" if best.get("ratio") is not None else "-"
            best_map = str(best.get("map_name_fr") or best.get("map_name") or "-")
            best_mode = str(best.get("pair_name_fr") or best.get("pair_name") or "-")
            highlights.append(
                HighlightItem(
                    title="Pic KD récent",
                    value=f"KD {best_ratio}",
                    detail=f"{best_map} · {best_mode}",
                )
            )

        # Highlight 2 : tendance
        trend = _compute_trend(mdf)
        if trend and trend.ratio_delta is not None:
            sign = "+" if trend.ratio_delta > 0 else ""
            highlights.append(
                HighlightItem(
                    title="Tendance",
                    value=f"KD {sign}{trend.ratio_delta:.2f}",
                    detail=f"WR {'+' if (trend.win_rate_delta or 0) > 0 else ''}{(trend.win_rate_delta or 0) * 100:.0f}%",
                )
            )

        # Highlight 3 : volume
        if len(highlights) < 3:
            kpis = _compute_kpis(mdf.head(10))
            highlights.append(
                HighlightItem(
                    title="Volume récent",
                    value=f"{min(10, len(mdf))} parties",
                    detail=f"KD {kpis.global_ratio or '-'} · WR {kpis.win_rate * 100:.0f}%",
                )
            )

        return highlights[:3]
    except Exception:
        logger.debug("_build_highlights: erreur", exc_info=True)
        return []


# ---------------------------------------------------------------------------
# Matchs récents
# ---------------------------------------------------------------------------


def _build_recent_matches(matches_df, limit: int = 6) -> list[RecentMatchItem]:
    """Construit la timeline des derniers matchs."""
    try:
        if hasattr(matches_df, "is_empty") and matches_df.is_empty():
            return []
        if "start_time" in matches_df.columns:
            mdf = matches_df.sort("start_time", descending=True)
        else:
            mdf = matches_df

        items: list[RecentMatchItem] = []
        for row in mdf.head(limit).iter_rows(named=True):
            match_id = str(row.get("match_id") or "").strip()
            if not match_id:
                continue
            outcome_code = int(row.get("outcome") or 0)
            outcome_label = OUTCOME_LABELS.get(outcome_code, "DNF")
            outcome_tone = OUTCOME_TONES.get(outcome_code, "dnf")
            map_l = str(row.get("map_name_fr") or row.get("map_name") or "-")
            mode_l = str(row.get("pair_name_fr") or row.get("pair_name") or "-")
            ratio = row.get("ratio")
            ratio_txt = f"{float(ratio):.2f}" if ratio is not None else "-"
            accuracy = row.get("accuracy")
            acc_txt = f"{float(accuracy):.0f}%" if accuracy is not None else "-"
            started_at = row.get("start_time")
            if isinstance(started_at, str):
                try:
                    started_at = datetime.fromisoformat(started_at)
                except Exception:
                    started_at = None
            items.append(
                RecentMatchItem(
                    match_id=match_id,
                    title=f"{outcome_label} · {map_l}",
                    detail=f"{mode_l} · KD {ratio_txt} · {acc_txt}",
                    started_at=started_at,
                    outcome_label=outcome_label,
                    outcome_tone=outcome_tone,
                )
            )
        return items
    except Exception:
        logger.debug("_build_recent_matches: erreur", exc_info=True)
        return []


# ---------------------------------------------------------------------------
# Résumé de session
# ---------------------------------------------------------------------------


def _build_session_summary(  # noqa: C901
    matches_df, sessions_df, *, squad_mode: bool
) -> SessionSummaryItem | None:
    """Construit le résumé de la dernière session solo ou escouade."""
    try:
        import polars as pl

        if hasattr(sessions_df, "is_empty") and sessions_df.is_empty():
            return None
        if hasattr(matches_df, "is_empty") and matches_df.is_empty():
            return None

        scope = sessions_df.filter(pl.col("is_with_friends").cast(pl.Boolean) == squad_mode)
        if scope.is_empty() or "session_label" not in scope.columns:
            return None

        latest_session = (
            scope.group_by("session_label")
            .agg(pl.col("start_time").max().alias("last_start"))
            .sort("last_start", descending=True)
        )
        if latest_session.is_empty():
            return None

        latest_label = latest_session.row(0, named=True).get("session_label")
        if latest_label is None:
            return None

        session_match_ids = (
            scope.filter(pl.col("session_label") == latest_label)
            .get_column("match_id")
            .drop_nulls()
            .unique()
            .to_list()
        )
        if not session_match_ids or "match_id" not in matches_df.columns:
            return None

        session_matches = matches_df.filter(pl.col("match_id").is_in(session_match_ids))
        if session_matches.is_empty():
            return None

        kpis = _compute_kpis(session_matches)
        started_at = None
        if "start_time" in session_matches.columns:
            ts = session_matches["start_time"].drop_nulls()
            if not ts.is_empty():
                started_at = ts.min()
                if isinstance(started_at, str):
                    try:
                        started_at = datetime.fromisoformat(started_at)
                    except Exception:
                        started_at = None

        return SessionSummaryItem(
            session_label=str(latest_label),
            match_count=len(session_matches),
            win_rate=kpis.win_rate,
            global_ratio=kpis.global_ratio,
            started_at=started_at,
        )
    except Exception:
        logger.debug("_build_session_summary(squad=%s): erreur", squad_mode, exc_info=True)
        return None


# ---------------------------------------------------------------------------
# Helpers live (battlepass / challenges) — best-effort via SPNKr
# ---------------------------------------------------------------------------


def _fetch_battlepass_live(player: PlayerContext) -> BattlePassResponse:
    """Tente un appel SPNKr pour récupérer les infos Battle Pass."""
    import asyncio

    from apps.api.app._pure_bridge import fetch_battlepass_info

    db_path = Path(player.db_path)

    async def _inner() -> BattlePassResponse:
        from src.auth.provider import get_halo_tokens_or_raise

        tokens = await get_halo_tokens_or_raise(db_path)
        bp = await fetch_battlepass_info(tokens)
        return BattlePassResponse(
            available=True,
            rank=getattr(bp, "rank", None),
            reward_track=getattr(bp, "reward_track", None),
            progress=getattr(bp, "progress", None),
        )

    loop = asyncio.new_event_loop()
    try:
        return loop.run_until_complete(_inner())
    finally:
        loop.close()


def _fetch_challenges_live(player: PlayerContext) -> ChallengesResponse:
    """Tente un appel SPNKr pour récupérer les défis actifs."""
    import asyncio

    from apps.api.app._pure_bridge import fetch_home_progressions

    db_path = Path(player.db_path)

    async def _inner() -> ChallengesResponse:
        from src.auth.provider import get_halo_tokens_or_raise

        tokens = await get_halo_tokens_or_raise(db_path)
        result = await fetch_home_progressions(db_path, tokens, player.xuid or "")
        ch = getattr(result, "challenge", None) if result else None
        if ch is None:
            return ChallengesResponse(available=False)
        return ChallengesResponse(
            available=True,
            total=getattr(ch, "total", None),
            completed=getattr(ch, "completed", None),
            xp_available=getattr(ch, "xp_available", None),
            next_expiry=getattr(ch, "next_expiry", None),
        )

    loop = asyncio.new_event_loop()
    try:
        return loop.run_until_complete(_inner())
    finally:
        loop.close()


# ---------------------------------------------------------------------------
# Helpers SQL bas niveau
# ---------------------------------------------------------------------------


def _resolve_gamertag(player: PlayerContext) -> str:
    """Retourne le gamertag affichable du joueur."""
    if player.gamertag:
        return player.gamertag
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(str(player.db_path)) as conn:
            row = conn.execute("SELECT value FROM sync_meta WHERE key = 'gamertag'").fetchone()
            if row:
                return str(row[0]).strip()
    except Exception:
        pass
    return player.player_slug
