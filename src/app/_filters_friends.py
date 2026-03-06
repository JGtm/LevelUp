"""Helpers de gestion des amis pour les filtres.

Sélection d'amis, defaults locaux, trio detection.
Extraits de filters.py pour respecter la limite de 500L.
"""

from __future__ import annotations

import json
from pathlib import Path

import polars as pl
import streamlit as st

from src.analysis import build_xuid_option_map
from src.ui import display_name_from_xuid
from src.ui.cache import (
    cached_list_other_xuids,
    cached_list_top_teammates,
    cached_same_team_match_ids_with_friend,
)

# =============================================================================
# Friends helpers
# =============================================================================


def _load_local_friends_defaults() -> dict[str, list[str]]:
    """Charge un mapping local {self_xuid: [friend1, friend2, ...]}.

    Fichier local (ignoré git): .streamlit/friends_defaults.json
    Valeurs: gamertags OU XUIDs.

    Convention: préfixer un nom avec ``~`` pour le garder dans le fichier
    mais l'exclure de la sélection par défaut (ami inactif/occasionnel).
    Exemple: ``"~SpartanD"``
    """
    try:
        p = Path(__file__).resolve().parents[2] / ".streamlit" / "friends_defaults.json"
        if not p.exists():
            return {}
        with open(p, encoding="utf-8") as f:
            obj = json.load(f) or {}
    except Exception:
        return {}

    if not isinstance(obj, dict):
        return {}

    out: dict[str, list[str]] = {}
    for k, v in obj.items():
        if not isinstance(k, str) or not k.strip():
            continue
        if not isinstance(v, list):
            continue
        vals: list[str] = []
        for it in v:
            if isinstance(it, str) and it.strip():
                raw = it.strip()
                # ~prefix = ami inactif, on l'ignore pour les defaults
                if raw.startswith("~"):
                    continue
                vals.append(raw)
        if vals:
            out[k.strip()] = vals
    return out


@st.cache_data(show_spinner=False)
def build_friends_opts_map(
    db_path: str,
    self_xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
) -> tuple[dict[str, str], list[str]]:
    """Construit le mapping d'options pour la sélection d'amis.

    Args:
        db_path: Chemin vers la base de données.
        self_xuid: XUID du joueur principal.
        db_key: Clé de cache DB.
        aliases_key: Clé de cache des alias.

    Returns:
        Tuple (opts_map {label: xuid}, default_labels [labels par défaut]).
    """
    top = cached_list_top_teammates(db_path, self_xuid, db_key=db_key, limit=20)
    default_two = [t[0] for t in top[:2]]
    all_other = cached_list_other_xuids(db_path, self_xuid, db_key=db_key, limit=500)

    ordered: list[str] = []
    seen: set[str] = set()
    for xx, _cnt in top:
        if xx not in seen:
            ordered.append(xx)
            seen.add(xx)
    for xx in all_other:
        if xx not in seen:
            ordered.append(xx)
            seen.add(xx)

    opts_map = build_xuid_option_map(
        ordered, display_name_fn=lambda x: display_name_from_xuid(x, db_path=db_path)
    )

    # Defaults: top 2, ou override local.
    default_xuids = list(default_two)
    try:
        overrides = _load_local_friends_defaults().get(str(self_xuid).strip())
        if overrides:
            name_to_xuid = {
                str(display_name_from_xuid(xu, db_path=db_path) or "").strip().casefold(): str(
                    xu
                ).strip()
                for xu in ordered
            }
            ordered_xuids = [str(xu).strip() for xu in ordered]

            chosen: list[str] = []
            for ident in overrides:
                s = str(ident or "").strip()
                if not s:
                    continue
                if s.isdigit() and s in ordered_xuids:
                    chosen.append(s)
                    continue
                xu = name_to_xuid.get(s.casefold())
                if xu:
                    chosen.append(xu)
            chosen = [x for x in chosen if x]
            if len(chosen) >= 2:
                default_xuids = chosen[:2]
    except Exception:
        pass

    label_by_xuid = {v: k for k, v in opts_map.items()}
    default_labels = [label_by_xuid[x] for x in default_xuids if x in label_by_xuid]
    return opts_map, default_labels


def get_friends_xuids_for_sessions(
    db_path: str,
    self_xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
) -> tuple[str, ...]:
    """Retourne les XUIDs des amis du joueur pour le calcul des sessions.

    Utilise .streamlit/friends_defaults.json ou top 2 coéquipiers par défaut.
    """
    opts_map, default_labels = build_friends_opts_map(
        db_path, str(self_xuid).strip(), db_key, aliases_key
    )
    xuids = [opts_map[lbl] for lbl in default_labels if lbl in opts_map]
    return tuple(sorted(set(xuids)))


def compute_trio_label(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    base_for_filters: pl.DataFrame,
    base_s_ui: pl.DataFrame,
) -> str | None:
    """Calcule le label de la dernière session en trio."""
    try:
        friends_opts_map, friends_default_labels = build_friends_opts_map(
            db_path, xuid.strip(), db_key, aliases_key
        )
        picked_friend_labels = st.session_state.get("friends_picked_labels")
        if not isinstance(picked_friend_labels, list) or not picked_friend_labels:
            picked_friend_labels = friends_default_labels
        picked_xuids = [
            friends_opts_map[lbl] for lbl in picked_friend_labels if lbl in friends_opts_map
        ]
        if len(picked_xuids) >= 2:
            f1_xuid, f2_xuid = picked_xuids[0], picked_xuids[1]
            ids_m = set(
                cached_same_team_match_ids_with_friend(
                    db_path, xuid.strip(), f1_xuid, db_key=db_key
                )
            )
            ids_c = set(
                cached_same_team_match_ids_with_friend(
                    db_path, xuid.strip(), f2_xuid, db_key=db_key
                )
            )
            trio_ids = ids_m & ids_c
            base_match_ids = set(base_for_filters["match_id"].cast(pl.Utf8).to_list())
            trio_ids = trio_ids & base_match_ids
            if trio_ids:
                trio_rows = base_s_ui.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(trio_ids)))
                if not trio_rows.is_empty() and "start_time" in trio_rows.columns:
                    agg = (
                        trio_rows.group_by(["session_id", "session_label"])
                        .agg(pl.col("start_time").max())
                        .sort("start_time", descending=True)
                    )
                    if not agg.is_empty():
                        return agg["session_label"][0]
    except Exception:
        pass
    return None
