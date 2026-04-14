"""UI: affichage des commendations (Halo 5 : Guardians).

Les données de référence sont chargées depuis la table ``citation_mappings``
de ``metadata.duckdb`` (peuplée par ``scripts/populate_citation_mappings.py``).

Fichiers d'images attendus :
- static/commendations/h5g/*.png
"""

from __future__ import annotations

import base64
import html
import json as _json
import logging
import os
import unicodedata
from functools import lru_cache
from typing import Any

from src.ui.i18n import get_lang, t
from src.utils.paths import REPO_ROOT

logger = logging.getLogger(__name__)


# ── Utilitaires internes ────────────────────────────────────────────────────


def _abs_from_repo(path: str) -> str:
    """Transforme un chemin relatif au repo en chemin absolu."""
    if not path:
        return path
    if os.path.isabs(path):
        return path
    return os.path.join(str(REPO_ROOT), path)


def _normalize_name(s: str) -> str:
    """Normalise un nom de citation (minuscules, sans accents, espaces uniques)."""
    base = " ".join(str(s or "").strip().lower().split())
    return "".join(
        ch for ch in unicodedata.normalize("NFKD", base) if not unicodedata.combining(ch)
    )


# ── Chargement depuis DuckDB ────────────────────────────────────────────────


def _parse_tier_targets(csv_targets: str | None) -> list[dict[str, Any]]:
    """Convertit une chaîne CSV de cibles en liste de tiers dict.

    Input : ``"10,20,30,50,100"``
    Output : ``[{"tier": 1, "target_count": 10}, ...]``
    """
    if not csv_targets:
        return []
    result: list[dict[str, Any]] = []
    for i, part in enumerate(str(csv_targets).split(","), 1):
        part = part.strip()
        if not part:
            continue
        try:
            result.append({"tier": i, "target_count": int(part)})
        except ValueError:
            continue
    return result


@lru_cache(maxsize=1)
def _load_citations_from_db() -> list[dict[str, Any]]:
    """Charge toutes les citations activées depuis ``citation_mappings``.

    Délègue à ``src.data.citation_definitions.load_citation_definitions()``.
    """
    from src.data.citation_definitions import load_citation_definitions

    return load_citation_definitions()


# ── Mastery / progression ───────────────────────────────────────────────────


def _compute_mastery_display(  # noqa: PLR0912
    current_count: int,
    tiers: list[dict[str, Any]],
) -> tuple[str, str, bool, float]:
    """Retourne (label_niveau, label_compteur, is_master, progress_ratio).

    ``progress_ratio`` représente l'avancement *dans le niveau actuel* (0..1).
    """

    targets: list[int] = []
    for tier_item in tiers or []:
        v = tier_item.get("target_count")
        if v is None:
            continue
        try:
            targets.append(int(v))
        except Exception:
            continue
    targets = sorted({x for x in targets if x > 0})

    try:
        cur = int(current_count)
    except Exception:
        cur = 0
    if cur < 0:
        cur = 0

    if not targets:
        return "—", "", False, 0.0

    master_target = targets[-1]
    if cur >= master_target:
        return t("cit_mastery_master"), f"{cur}", True, 1.0

    completed = 0
    for target in targets:
        if cur >= target:
            completed += 1
        else:
            break

    next_target = targets[min(completed, len(targets) - 1)]
    prev_target = 0 if completed <= 0 else targets[completed - 1]
    denom = max(1, int(next_target - prev_target))
    ratio = float(max(0, cur - prev_target)) / float(denom)
    if ratio < 0.0:
        ratio = 0.0
    if ratio > 1.0:
        ratio = 1.0
    level = completed + 1
    return t("cit_mastery_level", level=level), f"{cur}/{next_target}", False, ratio


# ── Image helpers ────────────────────────────────────────────────────────────


def _img_src(image_path: str | None) -> str | None:
    """Retourne le chemin absolu vers l'image ou ``None``."""
    if not image_path:
        return None
    abs_p = _abs_from_repo(image_path)
    if os.path.exists(abs_p):
        return abs_p
    return None


@lru_cache(maxsize=512)
def _img_data_uri(abs_path: str, mtime: float | None = None) -> str | None:
    """Génère un data-URI base64 pour une image locale."""
    _ = mtime
    if not abs_path or not os.path.exists(abs_path):
        return None
    ext = os.path.splitext(abs_path)[1].lower()
    mime = (
        "image/png"
        if ext == ".png"
        else "image/jpeg"
        if ext in {".jpg", ".jpeg"}
        else "application/octet-stream"
    )
    try:
        with open(abs_path, "rb") as f:
            raw = f.read()
    except Exception:
        return None
    if not raw:
        return None
    b64 = base64.b64encode(raw).decode("ascii")
    return f"data:{mime};base64,{b64}"


# ── Rendu d'une rangée de citations ────────────────────────────────────────


def _resolve_citation_progress(
    item: dict,
    current_full: int,
) -> tuple[str, str, bool, float]:
    """Retourne (level_label, counter_label, is_master, progress_ratio) pour un item citation.

    Pour les composites, la progression est directe (N/total).
    Pour les citations normales, délègue à ``_compute_mastery_display``.
    """
    composite_total = item.get("composite_total")
    if composite_total:
        n_total = composite_total
        is_master = current_full >= n_total
        progress_ratio = 1.0 if is_master else current_full / n_total
        level_label = t("cit_mastery_master") if is_master else ""
        counter_label = str(current_full) if is_master else f"{current_full}/{n_total}"
        return level_label, counter_label, is_master, progress_ratio
    return _compute_mastery_display(current_full, item["tiers"])


def _render_citation_row(
    items: list[dict],
    cols_per_row: int,
    citations_full: dict[str, int],
    citations_filtered: dict[str, int],
    is_filtered: bool,
) -> None:
    """Affiche une grille de citations (cols_per_row colonnes par rangée)."""
    import streamlit as st  # lazy — rendu Streamlit uniquement

    cols = st.columns(cols_per_row)
    for i, item in enumerate(items):
        col = cols[i % cols_per_row]
        name = item["name"]
        desc = item["description"]
        norm_name = item["norm"]

        current_full = citations_full.get(norm_name, 0)
        current_filtered = citations_filtered.get(norm_name, 0) if is_filtered else current_full
        delta_citation = current_filtered if (is_filtered and current_filtered > 0) else 0

        level_label, counter_label, is_master, progress_ratio = _resolve_citation_progress(
            item, current_full
        )

        with col:
            st.markdown("<div class='os-citation-top-gap'></div>", unsafe_allow_html=True)
            img = _img_src(item["image_path"])
            data_uri = None
            if img:
                try:
                    mtime = os.path.getmtime(img)
                except OSError:
                    mtime = None
                data_uri = _img_data_uri(img, mtime)
            tip = html.escape(desc) if desc else html.escape(name)
            if data_uri:
                ring_class = "os-citation-ring" + (" os-citation-ring--master" if is_master else "")
                ring_color = "#d6b35a" if is_master else "#41d6ff"
                st.markdown(
                    "<div class='"
                    + ring_class
                    + "' title='"
                    + tip
                    + "' "
                    + 'style="--p:'
                    + str(float(progress_ratio))
                    + ";--ring-color:"
                    + ring_color
                    + ";--img:url('"
                    + data_uri
                    + "')\"></div>",
                    unsafe_allow_html=True,
                )
            else:
                st.markdown(
                    "<div class='os-medal-missing' title='" + tip + "'>?</div>",
                    unsafe_allow_html=True,
                )
            st.markdown(
                "<div class='os-citation-name' title='" + tip + "'>" + html.escape(name) + "</div>",
                unsafe_allow_html=True,
            )
            level_class = "os-citation-level" + (" os-citation-level--master" if is_master else "")
            st.markdown(
                f"<div class='{level_class}'>{html.escape(level_label)}</div>",
                unsafe_allow_html=True,
            )
            delta_html = ""
            if is_filtered and delta_citation > 0:
                delta_html = (
                    f" <span style='color: #4CAF50; font-weight: bold;'>+{delta_citation}</span>"
                )
            st.markdown(
                "<div class='os-citation-counter'>"
                + html.escape(counter_label)
                + delta_html
                + "</div>",
                unsafe_allow_html=True,
            )


# ── Section principale ──────────────────────────────────────────────────────


def render_h5g_commendations_section(  # noqa: C901, PLR0912, PLR0915
    *,
    db_path: str | None = None,
    xuid: str | None = None,
    filtered_match_ids: list[str] | None = None,
    all_match_ids: list[str] | None = None,
) -> None:
    """Affiche la section des commendations Halo 5.

    Utilise ``CitationEngine`` pour agréger les valeurs depuis ``match_citations``.

    Args:
        db_path: Chemin vers la base DuckDB du joueur.
        xuid: XUID du joueur.
        filtered_match_ids: IDs des matchs filtrés (pour delta). ``None`` = pas de filtre.
        all_match_ids: IDs de tous les matchs. ``None`` = agrège tout sans filtre.
    """
    import streamlit as st  # lazy — rendu Streamlit uniquement

    from src.analysis.citations.engine import CitationEngine

    # ── 1. Charger les citations depuis metadata.duckdb ─────────────────
    citations_db = _load_citations_from_db()

    if not citations_db:
        st.info(
            "Référentiel Citations indisponible. "
            "Vérifiez que `citation_mappings` est peuplé dans metadata.duckdb "
            "(`python scripts/populate_citation_mappings.py`)."
        )
        return

    # ── 2. Agréger les valeurs via CitationEngine ───────────────────────
    engine: CitationEngine | None = None
    citations_full: dict[str, int] = {}
    citations_filtered: dict[str, int] = {}

    if db_path and xuid:
        engine = CitationEngine(db_path, xuid)
        citations_full = engine.aggregate_for_display(match_ids=None)
        if filtered_match_ids is not None:
            citations_filtered = engine.aggregate_for_display(match_ids=filtered_match_ids)

    is_filtered = filtered_match_ids is not None and all_match_ids is not None

    # ── 3. Construire les items d'affichage ─────────────────────────────
    enabled_norms: set[str] = {c["citation_name_norm"] for c in citations_db}
    items: list[dict[str, Any]] = []

    lang = get_lang()
    for cit in citations_db:
        norm = cit["citation_name_norm"]
        # Données i18n directement depuis la DB (citation_mappings)
        name = cit["citation_name_display"]
        desc = cit.get("description") or ""
        category = cit.get("category") or ""
        image_path = cit.get("image_path")
        tier_targets = cit.get("tier_targets")
        composite_children = cit.get("composite_children")
        mapping_type = cit.get("mapping_type", "")

        # Tiers : pour les composites, 1 seul tier = total enfants activés
        # → progression directe N/total, pas "Niveau N+1"
        if mapping_type == "composite" and composite_children:
            try:
                children = _json.loads(composite_children)
            except Exception:
                children = []
            n_enabled = sum(1 for c in children if _normalize_name(c) in enabled_norms)
            tiers = [{"tier": 1, "target_count": n_enabled}] if n_enabled > 0 else []
        else:
            tiers = _parse_tier_targets(tier_targets)

        composite_total = (
            n_enabled if (mapping_type == "composite" and composite_children) else None
        )
        items.append(
            {
                "name": name,
                "norm": norm,
                "description": desc,
                "category": category,
                "subcategory": cit.get("subcategory"),
                "image_path": image_path,
                "tiers": tiers,
                "composite_total": composite_total,
            }
        )

    # ── 4. Filtre texte ───────────────────────────────────────────────────
    q = st.text_input(t("cit_search"), value="", placeholder=t("cit_search_placeholder"))
    if q.strip():
        qn = q.strip().lower()
        items = [
            x
            for x in items
            if qn in x["name"].lower()
            or qn in x["description"].lower()
            or qn in (x["category"] or "").lower()
        ]

    # ── 5. Grille 8 colonnes ────────────────────────────────────────────
    # ── 5. Affichage groupé par catégorie / sous-catégorie ─────────────
    _CATEGORY_ORDER_BY_LANG: dict[str, list[str]] = {
        "fr": ["Mode de jeu", "Multijoueur", "Arme", "Spartan Companies", "Véhicule", "Ennemi"],
        "en": ["Game mode", "Multiplayer", "Weapon", "Spartan Companies", "Vehicle", "Enemy"],
    }
    _CATEGORY_ORDER = _CATEGORY_ORDER_BY_LANG.get(lang, _CATEGORY_ORDER_BY_LANG["en"])
    _SUBCAT_ORDER: dict[str, list[str]] = {
        "Arme": ["Général", "UNSC", "Paria", "Forerunner", "Grenade"],
        "Weapon": ["General", "UNSC", "Paria", "Forerunner", "Grenade"],
        "Véhicule": ["Général", "UNSC", "Covenant"],
        "Vehicle": ["Général", "UNSC", "Covenant"],
        "Ennemi": ["Covenant", "Banished"],
        "Enemy": ["Covenant", "Banished"],
    }
    cols_per_row = 8

    from collections import defaultdict

    by_cat: dict[str, dict[str | None, list[dict]]] = defaultdict(lambda: defaultdict(list))
    for it in items:
        by_cat[it["category"] or ""][it.get("subcategory")].append(it)

    present_cats = {it["category"] for it in items}
    cat_order = [c for c in _CATEGORY_ORDER if c in present_cats]
    cat_order += sorted(c for c in present_cats if c not in _CATEGORY_ORDER)

    for cat in cat_order:
        subcats_dict = by_cat[cat]
        all_in_cat = [it for sub_items in subcats_dict.values() for it in sub_items]
        if not all_in_cat:
            continue

        st.markdown(
            f"<h2 style='margin-top:1.2rem; margin-bottom:0.25rem'>{html.escape(cat)}</h2>"
            "<hr style='margin:0.2rem 0 0.6rem 0;opacity:0.4'>",
            unsafe_allow_html=True,
        )

        if cat in _SUBCAT_ORDER:
            _SUBCAT_DISPLAY: dict[str, dict[str, str]] = {
                "en": {"Général": "General", "Paria": "Banished"},
                "fr": {"Paria": "Parias"},
            }
            ordered_keys = _SUBCAT_ORDER[cat]
            extra_keys = sorted(k for k in subcats_dict if k not in ordered_keys and k is not None)
            none_keys: list[str | None] = [None] if None in subcats_dict else []
            for subcat in ordered_keys + extra_keys + none_keys:  # type: ignore[operator]
                sub_items = subcats_dict.get(subcat, [])
                if not sub_items:
                    continue
                if subcat is not None:
                    subcat_label = _SUBCAT_DISPLAY.get(lang, {}).get(subcat, subcat)
                    st.markdown(
                        f"<h4 style='margin-top:0.5rem'>{html.escape(subcat_label)}</h4>"
                        "<hr style='margin:0.15rem 0 0.5rem 0;opacity:0.3'>",
                        unsafe_allow_html=True,
                    )
                _render_citation_row(
                    sub_items, cols_per_row, citations_full, citations_filtered, is_filtered
                )
        else:
            _render_citation_row(
                all_in_cat, cols_per_row, citations_full, citations_filtered, is_filtered
            )
