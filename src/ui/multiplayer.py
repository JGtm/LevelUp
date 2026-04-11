"""Gestion du sélecteur multi-joueurs.

Ce module fournit les fonctions pour :
- Détecter si une DB est multi-joueurs (table Players) - Legacy SQLite
- Lister les joueurs disponibles (Legacy et DuckDB v4)
- Afficher un sélecteur dans la sidebar

Architecture v4 (DuckDB):
Chaque joueur a sa propre DB dans data/players/{gamertag}/stats.duckdb.
Le sélecteur liste les dossiers joueurs disponibles.

Legacy SQLite:
Une seule DB avec table Players pour les DBs fusionnées.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING

import streamlit as st

from src.ui.i18n import t
from src.utils.paths import PLAYERS_DIR

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


def _get_players_dir() -> Path:
    """Retourne le chemin vers data/players/."""
    return PLAYERS_DIR


def _is_duckdb_file(db_path: str) -> bool:
    """Détecte si le fichier est une base DuckDB."""
    return db_path.endswith(".duckdb")


@dataclass
class PlayerInfo:
    """Informations sur un joueur dans une DB multi-joueurs."""

    xuid: str
    gamertag: str | None
    label: str | None
    total_matches: int
    first_match_date: str | None
    last_match_date: str | None

    @property
    def display_name(self) -> str:
        """Nom d'affichage pour le sélecteur."""
        if self.label:
            return self.label
        if self.gamertag:
            return self.gamertag
        return self.xuid[:15] + "…"

    @property
    def display_with_stats(self) -> str:
        """Nom d'affichage avec statistiques."""
        name = self.display_name
        if self.total_matches:
            return f"{name} ({self.total_matches} matchs)"
        return name


def is_multi_player_db(db_path: str) -> bool:
    """Vérifie si la DB est multi-joueurs. Toujours False en v5 (chaque joueur a sa propre DB)."""
    return False


def render_player_selector(
    db_path: str,
    current_xuid: str,
    key: str = "player_selector",
) -> str | None:
    """Sélecteur legacy multi-joueurs. Toujours None en v5 (single-player par DB)."""
    return None


# =============================================================================
# Architecture DuckDB v4 - Sélecteur multi-joueurs
# =============================================================================


@dataclass
class DuckDBPlayerInfo:
    """Informations sur un joueur DuckDB v4."""

    gamertag: str
    db_path: Path
    total_matches: int
    xuid: str | None = None

    @property
    def display_with_stats(self) -> str:
        """Nom d'affichage avec statistiques."""
        if self.total_matches:
            return f"{self.gamertag} ({self.total_matches} matchs)"
        return f"{self.gamertag} (0 matchs)"


def _count_matches_from_player_db(con) -> int:
    """Compte les matchs depuis la player DB (fallback v5 → v4 → v3)."""
    for table in ("player_match_enrichment",):
        try:
            result = con.execute(f"SELECT COUNT(*) FROM {table}").fetchone()  # noqa: S608
            if result and result[0] > 0:
                return result[0]
        except Exception:
            pass
    return 0


def _resolve_xuid_from_player_db(con) -> str | None:
    """Résout le XUID depuis la player DB (sync_meta)."""
    try:
        result = con.execute("SELECT value FROM sync_meta WHERE key = 'xuid'").fetchone()
        if result and result[0] and str(result[0]).strip():
            return str(result[0]).strip()
    except Exception:
        pass
    return None


def _resolve_from_shared(
    db_path: Path,
    gamertag: str,
    xuid: str | None,
    total_matches: int,
) -> tuple[str | None, int]:
    """Résout xuid et/ou match count depuis shared_matches_v2.duckdb.

    Indépendant de la player DB — fonctionne même si celle-ci est verrouillée.
    """
    try:
        from src.utils.db import duckdb_read_only
        from src.utils.paths import get_shared_matches_path_from_player

        shared_path = get_shared_matches_path_from_player(db_path)
        if not shared_path or not shared_path.exists():
            return xuid, total_matches

        with duckdb_read_only(shared_path) as shared_con:
            from src.utils.xuid import lookup_xuid_for_gamertag

            # Résoudre xuid si manquant
            if not xuid:
                xuid = lookup_xuid_for_gamertag(shared_con, gamertag)

            # Compter les matchs si toujours 0
            if total_matches == 0:
                if xuid:
                    result = shared_con.execute(
                        "SELECT COUNT(*) FROM match_participants WHERE xuid = ?",
                        [xuid],
                    ).fetchone()
                else:
                    # Dernier recours via v_gamertag_lookup : count sans xuid connu
                    result = shared_con.execute(
                        "SELECT COUNT(*) FROM match_participants mp "
                        "JOIN v_gamertag_lookup vg ON mp.xuid = vg.xuid "
                        "WHERE LOWER(vg.gamertag) = LOWER(?)",
                        [gamertag],
                    ).fetchone()
                total_matches = result[0] if result else 0
    except Exception as e:
        logger.debug("_resolve_from_shared: résolution shared échouée pour %s: %s", db_path, e)
    return xuid, total_matches


@st.cache_data(ttl=1800, show_spinner=False)
def list_duckdb_v4_players() -> list[DuckDBPlayerInfo]:
    """Liste les joueurs depuis data/players/*/stats.duckdb.

    Résultat mis en cache 30 min. La logique est séparée en deux phases
    indépendantes (player DB puis shared DB) pour être robuste même si
    la player DB est temporairement verrouillée par le MediaIndexer.

    Returns:
        Liste triée par nombre de matchs (décroissant).
    """
    players_dir = _get_players_dir()
    players: list[DuckDBPlayerInfo] = []

    if not players_dir.exists():
        return players

    for player_dir in sorted(players_dir.iterdir()):
        if not player_dir.is_dir():
            continue

        db_path = player_dir / "stats.duckdb"
        if not db_path.exists():
            continue

        gamertag = player_dir.name
        total_matches = 0
        xuid = None

        # ── Phase 1 : player DB (peut échouer si verrouillée par MediaIndexer) ──
        try:
            from src.utils.db import duckdb_read_only

            with duckdb_read_only(str(db_path)) as con:
                total_matches = _count_matches_from_player_db(con)
                xuid = _resolve_xuid_from_player_db(con)
        except Exception as e:
            logger.debug("list_duckdb_v4_players: player DB verrouillée pour %s: %s", db_path, e)

        # ── Phase 2 : shared DB (indépendante du player DB) ──
        if not xuid or total_matches == 0:
            xuid, total_matches = _resolve_from_shared(
                db_path,
                gamertag,
                xuid,
                total_matches,
            )

        players.append(
            DuckDBPlayerInfo(
                gamertag=gamertag,
                db_path=db_path,
                total_matches=total_matches,
                xuid=xuid,
            )
        )

    # Trier par nombre de matchs décroissant
    players.sort(key=lambda p: p.total_matches, reverse=True)
    return players


def is_duckdb_v4_path(db_path: str) -> bool:
    """Vérifie si le chemin est une DB joueur DuckDB v4.

    Détecte si le chemin correspond au pattern data/players/{gamertag}/stats.duckdb.
    """
    if not db_path:
        return False

    try:
        p = Path(db_path).resolve()
        # Vérifier le pattern: .../data/players/{gamertag}/stats.duckdb
        if p.name == "stats.duckdb" and p.parent.parent.name == "players":
            return True
    except Exception:
        pass

    return False


def get_gamertag_from_duckdb_v4_path(db_path: str) -> str | None:
    """Extrait le gamertag depuis un chemin DuckDB v4.

    Args:
        db_path: Chemin vers stats.duckdb

    Returns:
        Gamertag ou None si le chemin n'est pas valide.
    """
    if not db_path:
        return None

    try:
        p = Path(db_path).resolve()
        if p.name == "stats.duckdb":
            return p.parent.name
    except Exception:
        pass

    return None


def render_duckdb_v4_player_selector(
    current_db_path: str,
    key: str = "duckdb_v4_player_selector",
    *,
    show_heading: bool = True,
    show_sync_indicator: bool = True,
) -> str | None:
    """Affiche un sélecteur de joueur pour l'architecture DuckDB v4.

    Args:
        current_db_path: Chemin vers la DB actuelle.
        key: Clé Streamlit pour le widget.

    Returns:
        Nouveau db_path si changement, None sinon.
    """
    players = list_duckdb_v4_players()

    if len(players) <= 1:
        # Un seul joueur ou aucun, pas besoin de sélecteur
        return None

    # Trouver le joueur actuel
    current_gamertag = get_gamertag_from_duckdb_v4_path(current_db_path)

    # Construire les options
    options = {str(p.db_path): p.display_with_stats for p in players}
    db_paths = list(options.keys())
    labels = list(options.values())

    # Index actuel
    try:
        current_idx = next(i for i, p in enumerate(players) if p.gamertag == current_gamertag)
    except StopIteration:
        current_idx = 0

    # Initialiser session_state avant le rendu pour éviter le conflit
    # "default value + Session State API" (warning Streamlit)
    if key not in st.session_state:
        st.session_state[key] = labels[current_idx] if labels else None

    if show_heading and show_sync_indicator:
        _col_heading, _col_sync = st.columns([1, 1])
        with _col_heading:
            st.markdown(f"#### {t('sidebar_player_heading')}")
        with _col_sync:
            from src.ui._sync_indicator import render_sync_indicator as _render_sync

            _render_sync(current_db_path)
    elif show_heading:
        st.markdown(f"#### {t('sidebar_player_heading')}")
    elif show_sync_indicator:
        from src.ui._sync_indicator import render_sync_indicator as _render_sync

        _render_sync(current_db_path)

    selected_label = st.selectbox(
        t("sidebar_player_label"),
        options=labels,
        key=key,
        label_visibility="collapsed",
    )

    # Retrouver le db_path sélectionné
    try:
        selected_idx = labels.index(selected_label)
        selected_db_path = db_paths[selected_idx]
    except (ValueError, IndexError):
        return None

    # Vérifier si changement
    try:
        if Path(selected_db_path).resolve() != Path(current_db_path).resolve():
            gamertag = Path(selected_db_path).parent.name if selected_db_path else selected_db_path
            logger.info("Joueur sélectionné: %s", gamertag or selected_db_path)
            return selected_db_path
    except Exception:
        pass

    return None


def render_player_selector_unified(
    db_path: str,
    current_xuid: str,
    key: str = "player_selector",
    *,
    show_heading: bool = True,
    show_sync_indicator: bool = True,
) -> tuple[str | None, str | None]:
    """Sélecteur de joueur unifié (Legacy SQLite + DuckDB v4).

    Cette fonction détecte automatiquement l'architecture et affiche
    le sélecteur approprié.

    Args:
        db_path: Chemin vers la DB actuelle.
        current_xuid: XUID actuellement sélectionné.
        key: Clé Streamlit pour le widget.

    Returns:
        Tuple (new_db_path, new_xuid):
        - new_db_path: Nouveau chemin DB si changement (DuckDB v4), None sinon
        - new_xuid: Nouveau XUID si changement (Legacy), None sinon
    """
    if not db_path:
        return None, None

    # Cas 1: Architecture DuckDB v4
    if is_duckdb_v4_path(db_path):
        new_db_path = render_duckdb_v4_player_selector(
            db_path,
            key=f"{key}_v4",
            show_heading=show_heading,
            show_sync_indicator=show_sync_indicator,
        )
        if new_db_path:
            # Récupérer le XUID du nouveau joueur avec fallback intelligent
            new_xuid = None
            try:
                from src.ui.cache_loaders import _resolve_player_xuid

                resolved = _resolve_player_xuid(new_db_path)
                if resolved:
                    new_xuid = resolved
            except Exception:
                pass
            return new_db_path, new_xuid
        return None, None

    # Cas 2: Legacy SQLite multi-joueurs
    new_xuid = render_player_selector(db_path, current_xuid, key=key)
    if new_xuid:
        return None, new_xuid

    return None, None
