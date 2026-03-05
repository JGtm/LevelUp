"""Fonctions de synchronisation et gestion des bases SPNKr.

Ce module contient les fonctions pour :
- Détecter et sélectionner les bases SPNKr
- Afficher l'indicateur de synchronisation
- Rafraîchir les bases via l'API
- Nettoyer les fichiers temporaires orphelins

Les fonctions utilitaires, l'indicateur de sync et les opérations DuckDB
sont réparties dans des sous-modules pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

import logging
import subprocess
import sys
from pathlib import Path

logger = logging.getLogger(__name__)

# ── Re-exports depuis les sous-modules ──────────────────────────────────
from src.ui._sync_duckdb_ops import (  # noqa: E402
    _sync_duckdb_player,
    sync_player_duckdb,
    sync_player_duckdb_async,
)
from src.ui._sync_indicator import render_sync_indicator  # noqa: E402
from src.ui._sync_utils import (  # noqa: E402
    _get_sync_metadata_smart,
    _shared_path,
    _summarize_sync_results,
    cleanup_orphan_tmp_dbs,
    get_player_duckdb_path,
    is_duckdb_player,
    is_spnkr_db_path,
    pick_latest_spnkr_db_if_any,
)
from src.utils.paths import REPO_ROOT

__all__ = [
    "_get_sync_metadata_smart",
    "_shared_path",
    "_sync_duckdb_player",
    "cleanup_orphan_tmp_dbs",
    "get_player_duckdb_path",
    "is_duckdb_player",
    "is_spnkr_db_path",
    "pick_latest_spnkr_db_if_any",
    "refresh_spnkr_db_via_api",
    "render_sync_indicator",
    "sync_all_players",
    "sync_all_players_duckdb",
    "sync_player_auto",
    "sync_player_duckdb",
    "sync_player_duckdb_async",
]


def refresh_spnkr_db_via_api(  # noqa: PLR0913
    *,
    db_path: str,
    player: str,
    match_type: str,
    max_matches: int,
    rps: int,
    with_highlight_events: bool = True,
    with_aliases: bool = True,
    delta: bool = False,
    timeout_seconds: int = 180,
    repo_root: Path | None = None,
) -> tuple[bool, str]:
    """Rafraîchit une DB SPNKr en appelant scripts/spnkr_import_db.py.

    Écrit directement dans la DB cible avec --resume (pas de copie temporaire).

    IMPORTANT: Toutes les données sont toujours récupérées (highlights, skill, aliases).
    Les paramètres with_highlight_events et with_aliases sont forcés à True.

    Args:
        db_path: Chemin vers la DB cible.
        player: Gamertag ou XUID du joueur.
        match_type: Type de matchs (all, matchmaking, custom, local).
        max_matches: Nombre maximum de matchs à récupérer.
        rps: Requêtes par seconde.
        with_highlight_events: Ignoré (toujours True).
        with_aliases: Ignoré (toujours True).
        delta: Mode delta (défaut: False).
        timeout_seconds: Timeout en secondes (défaut: 180).
        repo_root: Racine du repo (déduit automatiquement si None).

    Returns:
        Tuple (succès, message).
    """
    if repo_root is None:
        repo_root = REPO_ROOT
    importer = repo_root / "scripts" / "spnkr_import_db.py"
    if not importer.exists():
        return False, f"Script introuvable: {importer}"

    p = (player or "").strip()
    if not p:
        return False, "Aucun joueur pour SPNKr (gamertag ou XUID)."

    mt = (match_type or "matchmaking").strip().lower()
    if mt not in {"all", "matchmaking", "custom", "local"}:
        mt = "matchmaking"

    cmd = [
        sys.executable,
        str(importer),
        "--out-db",
        str(db_path),
        "--player",
        p,
        "--match-type",
        mt,
        "--max-matches",
        str(int(max_matches)),
        "--requests-per-second",
        str(int(rps)),
        "--resume",
    ]
    if delta:
        cmd.append("--delta")

    try:
        proc = subprocess.run(
            cmd,
            cwd=str(repo_root),
            capture_output=True,
            text=True,
            timeout=int(timeout_seconds),
        )
    except subprocess.TimeoutExpired:
        return False, f"Timeout après {timeout_seconds}s (import SPNKr trop long)."
    except Exception as e:
        return False, f"Erreur au lancement de l'import SPNKr: {e}"

    if int(proc.returncode) != 0:
        tail = (proc.stderr or proc.stdout or "").strip()
        if len(tail) > 1200:
            tail = tail[-1200:]
        return False, f"Import SPNKr en échec (code={proc.returncode}).\n{tail}".strip()

    return True, f"Sync OK pour {p}"


def sync_all_players(  # noqa: C901, PLR0912, PLR0913
    *,
    db_path: str,
    match_type: str = "matchmaking",
    max_matches: int = 200,
    rps: int = 5,
    with_highlight_events: bool = True,
    with_aliases: bool = True,
    delta: bool = True,
    timeout_seconds: int = 120,
) -> tuple[bool, str]:
    """Synchronise tous les joueurs d'une DB fusionnée (table Players).

    IMPORTANT: Toutes les données sont toujours récupérées.

    Returns:
        Tuple (succès_global, message_résumé).
    """
    from src.utils import (
        guess_xuid_from_db_path,
        infer_spnkr_player_from_db_path,
    )

    players = _resolve_players_from_db(db_path)

    if not players:
        # Fallback: DB mono-joueur, on déduit depuis le nom du fichier
        single_player = infer_spnkr_player_from_db_path(db_path) or ""
        if not single_player:
            xuid_from_path = guess_xuid_from_db_path(db_path)
            if xuid_from_path:
                single_player = xuid_from_path
                players = [
                    {"xuid": xuid_from_path, "gamertag": xuid_from_path, "label": xuid_from_path}
                ]
        if not players:
            if not single_player:
                return (
                    False,
                    "Aucun joueur trouvé dans la DB. Utilisez --player <gamertag ou XUID> "
                    "pour une DB sans table Players.",
                )
            players = [{"xuid": "", "gamertag": single_player, "label": single_player}]

    is_duckdb = db_path.endswith(".duckdb")
    results: list[tuple[str, bool, str]] = []

    for p in players:
        player_id = str(p.get("xuid") or p.get("gamertag") or "").strip()
        player_label = str(p.get("label") or p.get("gamertag") or player_id).strip()
        gamertag = str(p.get("gamertag") or "").strip()

        if not player_id:
            continue

        if is_duckdb and gamertag:
            ok, msg = _sync_duckdb_player(
                db_path=db_path,
                gamertag=gamertag,
                max_matches=max_matches,
                delta=delta,
                timeout_seconds=timeout_seconds,
            )
        else:
            ok, msg = refresh_spnkr_db_via_api(
                db_path=db_path,
                player=player_id,
                match_type=match_type,
                max_matches=max_matches,
                rps=rps,
                with_highlight_events=True,
                with_aliases=True,
                delta=delta,
                timeout_seconds=timeout_seconds,
            )
        results.append((player_label, ok, msg))

    return _summarize_sync_results(results)


def _resolve_players_from_db(db_path: str) -> list[dict]:
    """Résout la liste des joueurs depuis le chemin de la DB.

    Pour DuckDB v4, extrait le gamertag depuis le chemin conventionnel
    ``data/players/{gamertag}/stats.duckdb``.
    """
    players: list[dict] = []
    if not db_path.endswith(".duckdb"):
        return players

    try:
        p = Path(db_path)
        if p.name == "stats.duckdb" and p.parent.parent.name == "players":
            gamertag = p.parent.name
            from src.ui.cache_loaders import _resolve_player_xuid

            xuid = _resolve_player_xuid(db_path)
            players = [{"xuid": xuid, "gamertag": gamertag, "label": gamertag}]
    except Exception:
        pass

    return players


def sync_player_auto(  # noqa: PLR0913
    gamertag: str,
    xuid: str,
    *,
    db_path: str | None = None,
    delta: bool = True,
    match_type: str = "matchmaking",
    max_matches: int = 200,
    with_highlight_events: bool = True,
    with_aliases: bool = True,
    timeout_seconds: int = 300,
    repo_root: Path | None = None,
) -> tuple[bool, str]:
    """Synchronise un joueur en détectant automatiquement le mode.

    Utilise DuckDB si le joueur a une DB v4, sinon fallback sur SPNKr legacy.

    IMPORTANT: Toutes les données sont toujours récupérées.

    Args:
        gamertag: Gamertag du joueur.
        xuid: XUID du joueur.
        db_path: Chemin DB legacy (pour fallback).
        delta: Mode delta (True) ou full (False).
        match_type: Type de matchs.
        max_matches: Nombre max de matchs.
        with_highlight_events: Ignoré (toujours True).
        with_aliases: Ignoré (toujours True).
        timeout_seconds: Timeout pour le mode legacy.
        repo_root: Racine du repo.

    Returns:
        Tuple (success, message).
    """
    # Forcer la récupération de toutes les données
    with_highlight_events = True
    with_aliases = True
    # Priorité 1: DuckDB v4
    if is_duckdb_player(gamertag, repo_root):
        return sync_player_duckdb(
            gamertag=gamertag,
            xuid=xuid,
            delta=delta,
            match_type=match_type,
            max_matches=max_matches,
            with_highlight_events=with_highlight_events,
            with_skill=True,
            with_aliases=with_aliases,
            repo_root=repo_root,
        )

    # Fallback: SPNKr legacy
    if db_path:
        return refresh_spnkr_db_via_api(
            db_path=db_path,
            player=xuid or gamertag,
            match_type=match_type,
            max_matches=max_matches,
            rps=5,
            with_highlight_events=with_highlight_events,
            with_aliases=with_aliases,
            delta=delta,
            timeout_seconds=timeout_seconds,
            repo_root=repo_root,
        )

    return False, f"Aucune DB trouvée pour {gamertag}"


def sync_all_players_duckdb(  # noqa: PLR0913
    *,
    delta: bool = True,
    match_type: str = "matchmaking",
    max_matches: int = 200,
    with_highlight_events: bool = True,
    with_aliases: bool = True,
    repo_root: Path | None = None,
) -> tuple[bool, str]:
    """Synchronise tous les joueurs DuckDB v4 via db_profiles.json.

    Args:
        delta: Mode delta (True) ou full (False).
        match_type: Type de matchs.
        max_matches: Nombre max de matchs.
        with_highlight_events: Récupérer les highlight events.
        with_aliases: Mettre à jour les aliases.
        repo_root: Racine du repo.

    Returns:
        Tuple (success_global, message_résumé).
    """
    import json

    if repo_root is None:
        repo_root = REPO_ROOT

    db_profiles_path = repo_root / "db_profiles.json"
    if not db_profiles_path.exists():
        return False, "Fichier db_profiles.json introuvable."

    try:
        with open(db_profiles_path, encoding="utf-8") as f:
            profiles_data = json.load(f)
    except (json.JSONDecodeError, OSError) as e:
        return False, f"Erreur lecture db_profiles.json: {e}"

    profiles = profiles_data.get("profiles", {})
    if not profiles:
        return False, "Aucun profil dans db_profiles.json."

    results: list[tuple[str, bool, str]] = []

    for gamertag, profile in profiles.items():
        xuid = profile.get("xuid", "")
        player_db_path = repo_root / profile.get("db_path", "")

        if not player_db_path.exists():
            results.append((gamertag, False, f"DB introuvable: {player_db_path}"))
            continue

        ok, msg = sync_player_duckdb(
            gamertag=gamertag,
            xuid=xuid,
            delta=delta,
            match_type=match_type,
            max_matches=max_matches,
            with_highlight_events=with_highlight_events,
            with_skill=True,
            with_aliases=with_aliases,
            repo_root=repo_root,
        )
        results.append((gamertag, ok, msg))

    return _summarize_sync_results(results)
