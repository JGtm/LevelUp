"""Composant de filtre par checkboxes dans un expander.

Remplace les selectbox par des listes de checkboxes groupées,
plus pratiques quand il y a beaucoup de valeurs à filtrer.
"""

from __future__ import annotations

import streamlit as st

# Mapping préfixe -> catégorie pour inférer la catégorie des modes non traduits
# Catégories simplifiées: Assassin, Fiesta, BTB, Ranked, Firefight, Other
PREFIX_TO_CATEGORY: dict[str, str] = {
    # Assassin (Arena, Tactical, Community, etc.)
    "Arena": "Assassin",
    "Arène": "Assassin",
    "Tactical": "Assassin",
    "Tactique": "Assassin",
    "Community": "Assassin",
    "Communauté": "Assassin",
    "Assault": "Assassin",
    # Fiesta (Super Fiesta, Husky Raid, Castle Wars, etc.)
    "Fiesta": "Fiesta",
    "Super Fiesta": "Fiesta",
    "Husky Raid": "Fiesta",
    "Super Husky Raid": "Fiesta",
    "Castle Wars": "Fiesta",
    # BTB
    "BTB": "BTB",
    "BTB Heavies": "BTB",
    # Ranked
    "Ranked": "Ranked",
    "Classé": "Ranked",
    # Firefight
    "Firefight": "Firefight",
    "Gruntpocalypse": "Firefight",
    # Autre
    "Event": "Other",
}


def _infer_category(mode_name: str) -> str:
    """Infère la catégorie d'un mode à partir de son préfixe ou contenu.

    Catégories: Assassin, Fiesta, BTB, Ranked, Firefight, Other

    Exemples:
        "Arène : Assassin" -> "Assassin"
        "BTB : CTF" -> "BTB"
        "Super Fiesta : Assassin" -> "Fiesta"
        "Communauté : Fiesta Assassin" -> "Fiesta" (contient Fiesta)
    """
    # Détecter les modes Fiesta par leur contenu (pas seulement le préfixe)
    mode_lower = mode_name.lower()
    if "fiesta" in mode_lower or "husky raid" in mode_lower or "castle wars" in mode_lower:
        return "Fiesta"

    # Extraire le préfixe (avant ":" ou " : ")
    prefix = None
    if " : " in mode_name:
        prefix = mode_name.split(" : ", 1)[0].strip()
    elif ":" in mode_name:
        prefix = mode_name.split(":", 1)[0].strip()

    if prefix:
        # Vérifier si le préfixe correspond à une catégorie connue
        if prefix in PREFIX_TO_CATEGORY:
            return PREFIX_TO_CATEGORY[prefix]
        # Essayer en ignorant la casse
        for p, cat in PREFIX_TO_CATEGORY.items():
            if prefix.lower() == p.lower():
                return cat

    return "Other"


# Traduction des catégories en français
CATEGORY_FR: dict[str, str] = {
    "Assassin": "Assassin",
    "Fiesta": "Fiesta",
    "BTB": "Grande bataille en équipe",
    "Ranked": "Classé",
    "Firefight": "Baptême du feu",
    "Other": "Autre",
}


def _translate_category(cat: str) -> str:
    """Traduit une catégorie en français."""
    return CATEGORY_FR.get(cat, cat)


def render_checkbox_filter(
    *,
    label: str,
    options: list[str],
    session_key: str,
    default_unchecked: set[str] | None = None,
    show_select_buttons: bool = True,
    expanded: bool = False,
) -> set[str]:
    """Affiche un expander avec checkboxes pour filtrer une liste de valeurs.

    Args:
        label: Titre de l'expander (ex: "Playlists").
        options: Liste des valeurs disponibles à cocher.
        session_key: Clé session_state pour persister la sélection.
        default_unchecked: Valeurs décochées par défaut (ex: Firefight).
            Si None, tout est coché par défaut.
        show_select_buttons: Afficher les boutons Tout/Aucun.
        expanded: Si l'expander est ouvert par défaut.

    Returns:
        Ensemble des valeurs sélectionnées (cochées).
    """
    if not options:
        return set()

    # Initialisation session_state si nécessaire
    if session_key not in st.session_state:
        if default_unchecked:
            st.session_state[session_key] = set(options) - default_unchecked
        else:
            st.session_state[session_key] = set(options)

    # Nettoyer les valeurs obsolètes (plus dans options)
    current_selection: set[str] = st.session_state[session_key]
    current_selection = current_selection & set(options)
    st.session_state[session_key] = current_selection

    selected_count = len(current_selection)
    total_count = len(options)

    # Titre avec compteur
    if selected_count == total_count:
        title = f"{label} (tous)"
    elif selected_count == 0:
        title = f"{label} (aucun)"
    else:
        title = f"{label} ({selected_count}/{total_count})"

    with st.expander(title, expanded=expanded):
        # Boutons Tout / Aucun
        if show_select_buttons and len(options) > 1:
            cols = st.columns(2)

            def _select_all(sk: str = session_key, opts: list[str] = options) -> None:
                st.session_state[sk] = set(opts)
                # Synchroniser les widget keys individuels pour éviter
                # que les checkboxes précédemment décochées n'annulent le "Tout"
                for o in opts:
                    st.session_state[f"{sk}_cb_{o}"] = True

            def _select_none(sk: str = session_key, opts: list[str] = options) -> None:
                if st.session_state.get(f"{sk}_confirm_clear"):
                    st.session_state[sk] = set()
                    st.session_state[f"{sk}_confirm_clear"] = False
                    # Synchroniser les widget keys individuels
                    for o in opts:
                        st.session_state[f"{sk}_cb_{o}"] = False
                else:
                    st.session_state[f"{sk}_confirm_clear"] = True

            cols[0].button(
                "✓ Tout", key=f"{session_key}_all", width="stretch", on_click=_select_all
            )
            cols[1].button(
                "✗ Aucun", key=f"{session_key}_none", width="stretch", on_click=_select_none
            )

        # Confirmation message
        if st.session_state.get(f"{session_key}_confirm_clear"):
            st.warning("⚠️ Confirmer : vider toutes les sélections ?")

            def _confirm_clear(sk: str = session_key, opts: list[str] = options) -> None:
                st.session_state[sk] = set()
                st.session_state[f"{sk}_confirm_clear"] = False
                # Synchroniser les widget keys individuels
                for o in opts:
                    st.session_state[f"{sk}_cb_{o}"] = False

            def _cancel_clear(sk: str = session_key) -> None:
                st.session_state[f"{sk}_confirm_clear"] = False

            st.button("Confirmer", key=f"{session_key}_confirm_btn", on_click=_confirm_clear)
            st.button("Annuler", key=f"{session_key}_cancel_btn", on_click=_cancel_clear)

        # Checkboxes
        for opt in options:
            checked = opt in st.session_state[session_key]
            new_val = st.checkbox(
                opt,
                value=checked,
                key=f"{session_key}_cb_{opt}",
            )
            if new_val and opt not in st.session_state[session_key]:
                st.session_state[session_key] = st.session_state[session_key] | {opt}
            elif not new_val and opt in st.session_state[session_key]:
                st.session_state[session_key] = st.session_state[session_key] - {opt}

    return st.session_state[session_key]


def _extract_mode_name(full_mode: str) -> str:
    """Extrait le nom du mode sans le préfixe de catégorie.

    Exemples:
        "Arène : Assassin" -> "Assassin"
        "BTB : Capture du drapeau" -> "Capture du drapeau"
        "Super Husky Raid : CDD" -> "CDD"
    """
    if " : " in full_mode:
        return full_mode.split(" : ", 1)[1].strip()
    return full_mode


def render_hierarchical_checkbox_filter(
    *,
    label: str,
    options: list[str],
    session_key: str,
    default_unchecked: set[str] | None = None,
    expanded: bool = False,
) -> set[str]:
    """Affiche un expander avec checkboxes groupées par catégorie.

    Les modes sont fusionnés par nom (ex: "Arène : Assassin" et "Communauté : Assassin"
    deviennent une seule checkbox "Assassin" dans la catégorie correspondante).

    Args:
        label: Titre de l'expander principal (ex: "Modes").
        options: Liste des valeurs disponibles (modes traduits avec préfixe).
        session_key: Clé session_state pour persister la sélection.
        default_unchecked: Valeurs décochées par défaut.
        expanded: Si l'expander principal est ouvert par défaut.

    Returns:
        Ensemble des valeurs sélectionnées (cochées) - valeurs originales avec préfixe.
    """
    if not options:
        return set()

    # Grouper les options par catégorie, puis par nom de mode (sans préfixe)
    # Structure: {category: {mode_name: [full_mode1, full_mode2, ...]}}
    categories: dict[str, dict[str, list[str]]] = {}
    for opt in options:
        cat = _infer_category(opt)
        mode_name = _extract_mode_name(opt)
        if cat not in categories:
            categories[cat] = {}
        if mode_name not in categories[cat]:
            categories[cat][mode_name] = []
        categories[cat][mode_name].append(opt)

    # Trier les catégories selon l'ordre de priorité
    priority_order = ["Assassin", "Fiesta", "BTB", "Ranked", "Firefight", "Other"]
    sorted_cats = []
    for cat in priority_order:
        if cat in categories:
            sorted_cats.append(cat)
    for cat in sorted(categories.keys()):
        if cat not in sorted_cats:
            sorted_cats.append(cat)

    # Initialisation session_state
    if session_key not in st.session_state:
        if default_unchecked:
            st.session_state[session_key] = set(options) - default_unchecked
        else:
            st.session_state[session_key] = set(options)

    # Nettoyer les valeurs obsolètes
    current_selection: set[str] = st.session_state[session_key]
    current_selection = current_selection & set(options)
    st.session_state[session_key] = current_selection

    selected_count = len(current_selection)
    total_count = len(options)

    # Titre principal avec compteur
    if selected_count == total_count:
        title = f"{label} (tous)"
    elif selected_count == 0:
        title = f"{label} (aucun)"
    else:
        title = f"{label} ({selected_count}/{total_count})"

    with st.expander(title, expanded=expanded):
        # Boutons globaux Tout / Aucun
        cols = st.columns(2)

        def _select_all_g(sk: str = session_key, opts: list[str] = options) -> None:
            st.session_state[sk] = set(opts)
            # Synchroniser les widget keys pour que les checkboxes reflètent le "Tout"
            # Les clés de catégorie et de mode individuel seront recréées au prochain render
            # mais on force les widgets existants à True pour éviter qu'ils n'annulent le set
            for wk in list(st.session_state.keys()):
                if wk.startswith(f"{sk}_cat_") or wk.startswith(f"{sk}_mode_"):
                    st.session_state[wk] = True

        def _select_none_g(sk: str = session_key, opts: list[str] = options) -> None:
            if st.session_state.get(f"{sk}_confirm_clear"):
                st.session_state[sk] = set()
                st.session_state[f"{sk}_confirm_clear"] = False
                # Synchroniser les widget keys
                for wk in list(st.session_state.keys()):
                    if wk.startswith(f"{sk}_cat_") or wk.startswith(f"{sk}_mode_"):
                        st.session_state[wk] = False
            else:
                st.session_state[f"{sk}_confirm_clear"] = True

        cols[0].button("✓ Tout", key=f"{session_key}_all", width="stretch", on_click=_select_all_g)
        cols[1].button(
            "✗ Aucun", key=f"{session_key}_none", width="stretch", on_click=_select_none_g
        )

        # Confirmation message
        if st.session_state.get(f"{session_key}_confirm_clear"):
            st.warning("⚠️ Confirmer : vider toutes les sélections ?")
            cols_confirm = st.columns(2)

            def _confirm_clear_g(sk: str = session_key) -> None:
                st.session_state[sk] = set()
                st.session_state[f"{sk}_confirm_clear"] = False
                # Synchroniser les widget keys
                for wk in list(st.session_state.keys()):
                    if wk.startswith(f"{sk}_cat_") or wk.startswith(f"{sk}_mode_"):
                        st.session_state[wk] = False

            def _cancel_clear_g(sk: str = session_key) -> None:
                st.session_state[f"{sk}_confirm_clear"] = False

            cols_confirm[0].button(
                "Confirmer", key=f"{session_key}_confirm_btn", on_click=_confirm_clear_g
            )
            cols_confirm[1].button(
                "Annuler", key=f"{session_key}_cancel_btn", on_click=_cancel_clear_g
            )

        st.markdown("---")

        # Afficher chaque catégorie
        for cat in sorted_cats:
            cat_modes = categories[cat]  # dict {mode_name: [full_modes]}
            cat_fr = _translate_category(cat)

            # Récupérer tous les full_modes de cette catégorie
            all_cat_options = [fm for modes in cat_modes.values() for fm in modes]

            # Compter les sélections dans cette catégorie
            cat_selected = [m for m in all_cat_options if m in st.session_state[session_key]]
            all_selected = len(cat_selected) == len(all_cat_options)

            # Nombre de modes uniques (après fusion)
            unique_modes_count = len(cat_modes)

            if unique_modes_count == 1:
                # Une seule catégorie/mode : checkbox simple
                mode_name = list(cat_modes.keys())[0]
                full_modes = cat_modes[mode_name]

                # Le mode est coché si TOUS les full_modes sont cochés
                checked = all(fm in st.session_state[session_key] for fm in full_modes)

                cb_key = f"{session_key}_cat_{cat}"

                def _on_cat_single_change(
                    sk: str = session_key, fms: list[str] = full_modes, k: str = cb_key
                ) -> None:
                    if st.session_state[k]:
                        st.session_state[sk] = st.session_state[sk] | set(fms)
                    else:
                        st.session_state[sk] = st.session_state[sk] - set(fms)

                st.checkbox(
                    f"{cat_fr}",
                    value=checked,
                    key=cb_key,
                    on_change=_on_cat_single_change,
                )
            else:
                # Plusieurs modes dans la catégorie
                # Compter les modes uniques sélectionnés
                modes_selected_count = sum(
                    1
                    for mode_name, full_modes in cat_modes.items()
                    if all(fm in st.session_state[session_key] for fm in full_modes)
                )

                cat_label = f"{cat_fr} ({modes_selected_count}/{unique_modes_count})"

                cat_cb_key = f"{session_key}_cat_{cat}"

                def _on_cat_change(
                    sk: str = session_key,
                    opts: list[str] = all_cat_options,
                    k: str = cat_cb_key,
                ) -> None:
                    if st.session_state[k]:
                        st.session_state[sk] = st.session_state[sk] | set(opts)
                    else:
                        st.session_state[sk] = st.session_state[sk] - set(opts)

                st.checkbox(
                    cat_label,
                    value=all_selected,
                    key=cat_cb_key,
                    on_change=_on_cat_change,
                )

                # Modes individuels affichés directement (sans expander)
                for mode_name in sorted(cat_modes.keys()):
                    full_modes = cat_modes[mode_name]
                    # Le mode est coché si TOUS les full_modes sont cochés
                    checked = all(fm in st.session_state[session_key] for fm in full_modes)

                    mode_cb_key = f"{session_key}_mode_{cat}_{mode_name}"

                    def _on_mode_change(
                        sk: str = session_key,
                        fms: list[str] = full_modes,
                        k: str = mode_cb_key,
                    ) -> None:
                        if st.session_state[k]:
                            st.session_state[sk] = st.session_state[sk] | set(fms)
                        else:
                            st.session_state[sk] = st.session_state[sk] - set(fms)

                    with st.container():
                        _, cb_col = st.columns([0.05, 0.95])
                        with cb_col:
                            st.checkbox(
                                mode_name,
                                value=checked,
                                key=mode_cb_key,
                                on_change=_on_mode_change,
                            )

    return st.session_state[session_key]


def get_firefight_playlists(playlist_values: list[str]) -> set[str]:
    """Identifie les playlists Firefight dans une liste.

    Args:
        playlist_values: Liste des noms de playlists.

    Returns:
        Ensemble des playlists contenant "Firefight".
    """
    return {p for p in playlist_values if "firefight" in p.lower()}
