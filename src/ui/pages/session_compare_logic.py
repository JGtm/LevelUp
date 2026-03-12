"""Logique métier pour la comparaison de sessions.

Fonctions extraites de ``session_compare.py`` qui n'utilisent pas Streamlit.
"""

from __future__ import annotations

import polars as pl

from src.analysis.mode_categories import infer_custom_category_from_pair_name
from src.data.domain.refdata import Outcome
from src.ui.i18n import get_weekdays, t
from src.ui.vectorize_helpers import build_mapping
from src.visualization._compat import DataFrameLike, ensure_polars

# ------------------------------------------------------------------
# Catégorisation
# ------------------------------------------------------------------


def _get_category_fr() -> dict[str, str]:
    """Labels traduits des catégories (Assassin/Fiesta absents : fallback .get(cat, cat))."""
    return {
        "BTB": t("cat_btb"),
        "Ranked": t("cat_ranked"),
        "Firefight": t("cat_firefight"),
        "Other": t("cat_other"),
    }


def infer_session_dominant_category(df_session: DataFrameLike) -> str:
    """Infère la catégorie dominante d'une session.

    Applique la catégorisation custom à chaque match via ``pair_name``,
    puis prend la catégorie la plus fréquente.
    """
    df_session = ensure_polars(df_session)
    if df_session.is_empty() or "pair_name" not in df_session.columns:
        return "Other"

    cats = (
        df_session.get_column("pair_name")
        .cast(pl.Utf8)
        .replace_strict(
            build_mapping(df_session.get_column("pair_name"), infer_custom_category_from_pair_name),
            default="Other",
            return_dtype=pl.Utf8,
        )
        .fill_null("Other")
        .alias("category")
    )
    if cats.is_empty():
        return "Other"

    vc = cats.value_counts().sort("count", descending=True)
    return str(vc[0, "category"]) if not vc.is_empty() else "Other"


# ------------------------------------------------------------------
# Amis / solo
# ------------------------------------------------------------------


def is_session_with_friends(df_session: DataFrameLike) -> bool:
    """Détermine si une session est avec amis (majorité des matchs)."""
    df_session = ensure_polars(df_session)
    if df_session.is_empty():
        return False
    if "is_with_friends" not in df_session.columns:
        return False
    return df_session.get_column("is_with_friends").sum() > len(df_session) / 2


def get_session_friends_signature(df_session: DataFrameLike) -> set[str]:
    """Extrait l'ensemble des XUIDs d'amis présents dans une session."""
    df_session = ensure_polars(df_session)
    if df_session.is_empty() or "friends_xuids" not in df_session.columns:
        return set()

    all_friends: set[str] = set()
    for friends_str in df_session.get_column("friends_xuids").drop_nulls().to_list():
        if friends_str:
            all_friends.update(friends_str.split(","))

    all_friends.discard("")
    return all_friends


# ------------------------------------------------------------------
# Filtrage et agrégation des sessions
# ------------------------------------------------------------------


def filter_candidate_sessions(  # noqa: PLR0912
    all_sessions_df: DataFrameLike,
    is_with_friends: bool,
    exclude_session_ids: list[int] | None = None,
    same_friends_xuids: set[str] | None = None,
    mode_category: str | None = None,
) -> list:
    """Filtre les sessions candidates pour la comparaison historique."""
    all_sessions_df = ensure_polars(all_sessions_df)
    if all_sessions_df.is_empty() or "session_id" not in all_sessions_df.columns:
        return []

    exclude_ids = list(set(exclude_session_ids or []))

    df = all_sessions_df.filter(~pl.col("session_id").is_in(exclude_ids))
    if df.is_empty():
        return []

    if "is_with_friends" not in df.columns:
        matching_session_ids = df.get_column("session_id").drop_nulls().unique().to_list()
    elif same_friends_xuids and len(same_friends_xuids) > 0:
        matching_session_ids = []
        for group_df in df.partition_by("session_id", maintain_order=True):
            session_id = group_df[0, "session_id"]
            session_friends = get_session_friends_signature(group_df)
            if same_friends_xuids <= session_friends:
                matching_session_ids.append(session_id)
    else:
        session_agg = df.group_by("session_id").agg(
            pl.col("is_with_friends").mean().alias("friend_ratio")
        )
        if is_with_friends:
            matching_session_ids = (
                session_agg.filter(pl.col("friend_ratio") > 0.5).get_column("session_id").to_list()
            )
        else:
            matching_session_ids = (
                session_agg.filter(pl.col("friend_ratio") <= 0.5).get_column("session_id").to_list()
            )

    if len(matching_session_ids) == 0:
        return []

    if mode_category and "pair_name" in df.columns:
        df_candidates = df.filter(pl.col("session_id").is_in(matching_session_ids))
        if df_candidates.is_empty():
            return []

        df_candidates = df_candidates.with_columns(
            pl.col("pair_name")
            .cast(pl.Utf8)
            .replace_strict(
                build_mapping(df_candidates["pair_name"], infer_custom_category_from_pair_name),
                default="Other",
                return_dtype=pl.Utf8,
            )
            .alias("_cat")
        )
        dom_by_session: dict = {}
        for group_df in df_candidates.partition_by("session_id", maintain_order=True):
            sid = group_df[0, "session_id"]
            vc = group_df.get_column("_cat").value_counts().sort("count", descending=True)
            dom_by_session[sid] = str(vc[0, "_cat"]) if not vc.is_empty() else "Other"

        matching_session_ids = [
            sid for sid in matching_session_ids if dom_by_session.get(sid) == mode_category
        ]

    return matching_session_ids


def aggregate_session_stats(
    df_matching: DataFrameLike,
    matching_session_ids: list,
) -> dict:
    """Agrège les statistiques des sessions filtrées."""
    df_matching = ensure_polars(df_matching)
    if df_matching.is_empty():
        return {}

    session_count = len(matching_session_ids)

    total_kills = df_matching.get_column("kills").sum()
    total_deaths = df_matching.get_column("deaths").sum()
    total_assists = df_matching.get_column("assists").sum()
    total_matches = len(df_matching)

    agg_exprs: list = [
        pl.col("kills").sum(),
        pl.col("deaths").sum(),
        pl.col("assists").sum(),
        (
            (pl.col("outcome") == Outcome.WIN).sum().cast(pl.Float64)
            / pl.len().cast(pl.Float64)
            * 100
        ).alias("win_rate"),
        pl.col("average_life_seconds").mean(),
    ]
    acc_col: str | None = None
    if "accuracy" in df_matching.columns:
        acc_col = "accuracy"
    elif "shots_accuracy" in df_matching.columns:
        acc_col = "shots_accuracy"
    if acc_col:
        agg_exprs.append(pl.col(acc_col).mean())

    session_stats = df_matching.group_by("session_id").agg(agg_exprs)

    session_stats = session_stats.with_columns(
        pl.when(pl.col("deaths") > 0)
        .then(pl.col("kills").cast(pl.Float64) / pl.col("deaths").cast(pl.Float64))
        .otherwise(pl.col("kills").cast(pl.Float64))
        .alias("kd_ratio")
    )

    avg_kd = session_stats.get_column("kd_ratio").mean()
    avg_win_rate = session_stats.get_column("win_rate").mean()
    avg_life = session_stats.get_column("average_life_seconds").mean()
    avg_accuracy = (
        session_stats.get_column(acc_col).mean()
        if acc_col and acc_col in session_stats.columns
        else None
    )

    return {
        "kd_ratio": avg_kd,
        "win_rate": avg_win_rate,
        "accuracy": avg_accuracy,
        "avg_life_seconds": avg_life,
        "kills_per_match": total_kills / total_matches if total_matches > 0 else 0,
        "deaths_per_match": total_deaths / total_matches if total_matches > 0 else 0,
        "assists_per_match": total_assists / total_matches if total_matches > 0 else 0,
        "session_count": session_count,
    }


def compute_similar_sessions_average(
    all_sessions_df: DataFrameLike,
    is_with_friends: bool,
    exclude_session_ids: list[int] | None = None,
    same_friends_xuids: set[str] | None = None,
    mode_category: str | None = None,
) -> dict:
    """Calcule la moyenne des sessions similaires (orchestre filtrage + agrégation)."""
    all_sessions_df = ensure_polars(all_sessions_df)
    matching_session_ids = filter_candidate_sessions(
        all_sessions_df,
        is_with_friends,
        exclude_session_ids,
        same_friends_xuids,
        mode_category,
    )
    if not matching_session_ids:
        return {}

    exclude_ids = list(set(exclude_session_ids or []))
    df = all_sessions_df.filter(~pl.col("session_id").is_in(exclude_ids))
    df_matching = df.filter(pl.col("session_id").is_in(matching_session_ids))

    return aggregate_session_stats(df_matching, matching_session_ids)


# ------------------------------------------------------------------
# Utilitaires de formatage
# ------------------------------------------------------------------


def format_seconds_to_mmss(seconds) -> str:
    """Formate des secondes en mm:ss ou retourne '—'."""
    if seconds is None:
        return "—"
    try:
        total = int(round(float(seconds)))
        if total < 0:
            return "—"
        m, s = divmod(total, 60)
        return f"{m}:{s:02d}"
    except Exception:
        return "—"


def format_date_with_weekday(dt) -> str:
    """Formate une date avec jour de la semaine abrégé : lun. 12/01/26 14:30."""
    if dt is None:
        return "-"
    try:
        weekdays = get_weekdays()
        wd = weekdays[dt.weekday()]
        return f"{wd} {dt.strftime('%d/%m/%y %H:%M')}"
    except Exception:
        return "-"


def outcome_class(label: str) -> str:
    """Retourne la classe CSS pour un résultat."""
    v = str(label or "").strip().casefold()
    if v.startswith("victoire") or v.startswith("victory") or v == "win":
        return "text-win"
    if v.startswith("défaite") or v.startswith("defaite") or v.startswith("defeat") or v == "loss":
        return "text-loss"
    if (
        v.startswith("égalité")
        or v.startswith("egalite")
        or v.startswith("draw")
        or v.startswith("tie")
    ):
        return "text-tie"
    if v.startswith("non") or v.startswith("did not") or v.startswith("dnf"):
        return "text-nf"
    return ""


# ------------------------------------------------------------------
# Contexte historique
# ------------------------------------------------------------------


def compute_historical_context(
    all_sessions_df: DataFrameLike,
    df_session_b: DataFrameLike,
    exclude_ids: list,
    session_b_category: str,
) -> tuple[dict, str]:
    """Calcule la moyenne historique des sessions similaires.

    Returns:
        Tuple (hist_avg dict, compare_mode string).
    """
    has_with_friends_col = "is_with_friends" in all_sessions_df.columns
    session_b_with_friends = (
        is_session_with_friends(df_session_b) if has_with_friends_col else False
    )
    session_b_friends = (
        get_session_friends_signature(df_session_b) if has_with_friends_col else set()
    )

    if not has_with_friends_col:
        hist_avg = compute_similar_sessions_average(
            all_sessions_df,
            is_with_friends=False,
            exclude_session_ids=exclude_ids,
            mode_category=session_b_category,
        )
        return hist_avg, "all"

    if session_b_with_friends and len(session_b_friends) > 0:
        hist_avg = compute_similar_sessions_average(
            all_sessions_df,
            is_with_friends=True,
            exclude_session_ids=exclude_ids,
            same_friends_xuids=session_b_friends,
            mode_category=session_b_category,
        )
        compare_mode = "same_friends"

        if hist_avg.get("session_count", 0) < 3:
            hist_avg = compute_similar_sessions_average(
                all_sessions_df,
                is_with_friends=True,
                exclude_session_ids=exclude_ids,
                same_friends_xuids=None,
                mode_category=session_b_category,
            )
            compare_mode = "any_friends"
        return hist_avg, compare_mode

    hist_avg = compute_similar_sessions_average(
        all_sessions_df,
        is_with_friends=False,
        exclude_session_ids=exclude_ids,
        mode_category=session_b_category,
    )
    return hist_avg, "solo"


def build_skill_series(
    df: pl.DataFrame,
    db_path: str | None,
) -> tuple[list[float | None] | None, str]:
    """Charge les ratings LUSR/CSR depuis match_skill_rank pour les matchs de df.

    Retourne (valeurs_par_match, label) — label vaut 'LUSR' ou 'CSR'.
    Si aucune donnée disponible, retourne (None, 'CSR').
    """
    if not db_path or "match_id" not in df.columns:
        return None, "CSR"
    match_ids = df.get_column("match_id").drop_nulls().to_list()
    if not match_ids:
        return None, "CSR"
    try:
        from src.utils.db import duckdb_read_only

        ph = ", ".join(["?"] * len(match_ids))
        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(
                f"SELECT match_id, rating_value, rating_type FROM match_skill_rank "
                f"WHERE match_id IN ({ph})",
                match_ids,
            ).fetchall()
    except Exception:
        return None, "CSR"
    if not rows:
        return None, "CSR"
    rating_map = {r[0]: (float(r[1]), str(r[2])) for r in rows}
    types = [v[1] for v in rating_map.values()]
    label = "LUSR" if types.count("LUSR") >= types.count("CSR") else "CSR"
    series = [rating_map[mid][0] if mid in rating_map else None for mid in match_ids]
    return (None, label) if all(v is None for v in series) else (series, label)


# ------------------------------------------------------------------
# Sélection de la session de comparaison
# ------------------------------------------------------------------


def _first_matching_label(
    ordered_labels: list[str],
    session_chars: dict[str, tuple],
    predicate,
) -> str | None:
    """Retourne le premier label dont les caractéristiques satisfont ``predicate``."""
    for lbl in ordered_labels:
        ch = session_chars.get(lbl)
        if ch and predicate(ch):
            return lbl
    return None


def _build_session_chars(
    df_cands: pl.DataFrame,
    older_labels: list[str],
    has_friends_col: bool,
) -> dict[str, tuple]:
    """Précalcule (catégorie, statut_amis, amis_xuids) pour chaque session candidate."""
    chars: dict[str, tuple] = {}
    for grp in df_cands.partition_by("session_label", maintain_order=True):
        lbl = str(grp[0, "session_label"])
        cat = infer_session_dominant_category(grp)
        wf = is_session_with_friends(grp) if has_friends_col else None
        fx = get_session_friends_signature(grp) if (has_friends_col and wf) else set()
        chars[lbl] = (cat, wf, fx)
    return chars


def find_best_matching_previous_session(
    all_sessions_df: DataFrameLike,
    picked_label: str,
    session_labels: list[str],
) -> str:
    """Trouve la session précédente la plus similaire à ``picked_label``.

    ``session_labels`` est trié du plus récent au plus ancien (index croissant = plus vieux).
    La session retournée est parmi ``session_labels[idx_b + 1:]`` (antérieures à B).

    Cascade de correspondance (du plus exigeant au moins exigeant) :
    1. Même catégorie + mêmes amis (sous-ensemble exact)
    2. Même catégorie + même statut ami/solo
    3. Même catégorie seulement
    4. Fallback chronologique (session immédiatement précédente)
    """
    all_sessions_df = ensure_polars(all_sessions_df)

    if picked_label not in session_labels or "session_label" not in all_sessions_df.columns:
        return session_labels[0] if session_labels else picked_label

    idx_b = session_labels.index(picked_label)
    older_labels = session_labels[idx_b + 1 :]

    if not older_labels:
        return session_labels[idx_b - 1] if idx_b > 0 else picked_label

    df_b = all_sessions_df.filter(pl.col("session_label") == picked_label)
    if df_b.is_empty():
        return older_labels[0]

    category_b = infer_session_dominant_category(df_b)
    has_friends_col = "is_with_friends" in df_b.columns
    with_friends_b = is_session_with_friends(df_b) if has_friends_col else None
    friends_b = (
        get_session_friends_signature(df_b) if (has_friends_col and with_friends_b) else set()
    )

    df_cands = all_sessions_df.filter(pl.col("session_label").is_in(older_labels))
    chars = _build_session_chars(df_cands, older_labels, has_friends_col)

    # Niveau 1 : même catégorie + mêmes amis (subset)
    if friends_b and has_friends_col:
        match = _first_matching_label(
            older_labels, chars, lambda ch: ch[0] == category_b and ch[2] and friends_b <= ch[2]
        )
        if match:
            return match

    # Niveau 2 : même catégorie + même statut ami/solo
    if has_friends_col and with_friends_b is not None:
        match = _first_matching_label(
            older_labels, chars, lambda ch: ch[0] == category_b and ch[1] == with_friends_b
        )
        if match:
            return match

    # Niveau 3 : même catégorie seulement
    return (
        _first_matching_label(older_labels, chars, lambda ch: ch[0] == category_b)
        or older_labels[0]
    )
