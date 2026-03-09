"""Logique métier du wizard de configuration initiale.

Sépare la logique testable de l'UI Streamlit.
Gère la validation, la création de profil joueur et la génération de config.

Usage :
    from src.ui.pages.setup_wizard_logic import (
        validate_dc_credentials,
        validate_gamertag,
        create_player_profile,
        get_setup_status,
        SetupStatus,
    )
"""

from __future__ import annotations

import json
import logging
import os
import re
from dataclasses import dataclass
from pathlib import Path

from src.utils.auth import AuthStatus, get_auth_status, write_env_local
from src.utils.paths import PLAYERS_DIR, REPO_ROOT

logger = logging.getLogger(__name__)

_DB_PROFILES_PATH = REPO_ROOT / "db_profiles.json"

# Regex pour un gamertag Xbox valide (1-15 chars, alphanum + espaces)
_GAMERTAG_PATTERN = re.compile(r"^[\w\s\-]{1,50}$")


@dataclass(frozen=True)
class SetupStatus:
    """État global de la configuration initiale."""

    auth: AuthStatus
    has_players: bool
    player_count: int

    @property
    def needs_setup(self) -> bool:
        """Retourne True si le wizard doit s'afficher."""
        return not self.auth.has_credentials or not self.has_players

    @property
    def current_step(self) -> int:
        """Détermine l'étape courante du wizard (1-3).

        1 = Credentials Azure manquantes
        2 = Token OAuth manquant
        3 = Aucun joueur configuré
        """
        if not self.auth.has_credentials:
            return 1
        if not self.auth.has_refresh_token:
            return 2
        if not self.has_players:
            return 3
        return 0  # Tout est configuré


def get_setup_status() -> SetupStatus:
    """Évalue l'état complet de la configuration.

    Returns:
        SetupStatus avec auth et état des joueurs.
    """
    auth = get_auth_status()
    player_count = _count_players()
    return SetupStatus(
        auth=auth,
        has_players=player_count > 0,
        player_count=player_count,
    )


def validate_dc_credentials(client_id: str) -> list[str]:
    """Valide le client_id Azure pour un public client (sans secret).

    Args:
        client_id: Azure Application (client) ID.

    Returns:
        Liste d'erreurs (vide si tout est valide).
    """
    errors: list[str] = []
    client_id = client_id.strip()
    if not client_id:
        errors.append("Le Client ID est requis.")
    elif not re.match(
        r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
        client_id,
        re.IGNORECASE,
    ):
        errors.append("Le Client ID doit être un UUID (ex: 12345678-1234-1234-1234-123456789abc).")
    return errors


def save_dc_credentials(client_id: str) -> None:
    """Sauvegarde uniquement le client_id Azure (public client, sans secret).

    Args:
        client_id: Azure Application (client) ID.
    """
    clean = client_id.strip()
    write_env_local({"SPNKR_AZURE_CLIENT_ID": clean})
    os.environ["SPNKR_AZURE_CLIENT_ID"] = clean
    logger.info("Client ID Azure (public client) sauvegardé dans .env.local")


def validate_gamertag(gamertag: str) -> list[str]:
    """Valide le format d'un gamertag Xbox.

    Args:
        gamertag: Gamertag ou XUID à valider.

    Returns:
        Liste d'erreurs (vide si valide).
    """
    errors: list[str] = []
    gamertag = gamertag.strip()

    if not gamertag:
        errors.append("Le gamertag est requis.")
    elif not _GAMERTAG_PATTERN.match(gamertag):
        errors.append(
            "Le gamertag contient des caractères invalides. "
            "Seuls les lettres, chiffres, espaces et tirets sont autorisés."
        )

    return errors


def create_player_profile(gamertag: str, xuid: str = "") -> str:
    """Crée un profil joueur dans db_profiles.json.

    Crée aussi le dossier data/players/<gamertag>/.

    Args:
        gamertag: Gamertag Xbox du joueur.
        xuid: XUID optionnel (sera résolu par la sync si absent).

    Returns:
        Clé du profil dans db_profiles.json.
    """
    gamertag = gamertag.strip()
    xuid = xuid.strip()

    # Charger ou créer db_profiles.json
    data = _load_db_profiles()
    profiles = data.setdefault("profiles", {})

    # Chercher une clé existante (case-insensitive)
    existing_key = _find_key_ci(profiles, gamertag)
    final_key = existing_key or gamertag

    db_path = f"data/players/{final_key}/stats.duckdb"
    profile = {
        "db_path": db_path,
        "waypoint_player": gamertag,
    }
    if xuid:
        profile["xuid"] = xuid

    # Merge avec l'existant si présent
    if final_key in profiles and isinstance(profiles[final_key], dict):
        merged = {**profiles[final_key], **profile}
        profiles[final_key] = merged
    else:
        profiles[final_key] = profile

    _save_db_profiles(data)

    # Créer le dossier joueur
    player_dir = PLAYERS_DIR / final_key
    player_dir.mkdir(parents=True, exist_ok=True)

    logger.info("Profil joueur créé : %s (db_path=%s)", final_key, db_path)
    return final_key


def get_token_script_path() -> Path:
    """Retourne le chemin vers le script d'acquisition de token.

    Ce chemin sera utilisé par le wizard pour lancer le script
    dans un terminal séparé.
    """
    return REPO_ROOT / "scripts" / "spnkr_get_refresh_token.py"


def get_sync_command(gamertag: str, max_matches: int = 200) -> list[str]:
    """Construit la commande de première sync pour un joueur.

    Args:
        gamertag: Gamertag du joueur.
        max_matches: Nombre max de matchs à récupérer.

    Returns:
        Liste d'arguments pour subprocess.
    """
    return [
        "python",
        "scripts/sync.py",
        "--add-player",
        gamertag.strip(),
        "--full",
        "--max-matches",
        str(max_matches),
    ]


# ── Fonctions privées ──────────────────────────────────────────────────────


def _count_players() -> int:
    """Compte le nombre de joueurs avec une DB existante."""
    if not PLAYERS_DIR.exists():
        return 0
    count = 0
    for player_dir in PLAYERS_DIR.iterdir():
        if player_dir.is_dir() and (player_dir / "stats.duckdb").exists():
            count += 1
    return count


def _load_db_profiles() -> dict:
    """Charge db_profiles.json ou retourne un squelette par défaut."""
    if not _DB_PROFILES_PATH.exists():
        return {
            "version": "2.1",
            "warehouse_path": "data/warehouse",
            "metadata_db": "data/warehouse/metadata.duckdb",
            "profiles": {},
        }
    with open(_DB_PROFILES_PATH, encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        return {"version": "2.1", "profiles": {}}
    if not isinstance(data.get("profiles"), dict):
        data["profiles"] = {}
    return data


def _save_db_profiles(data: dict) -> None:
    """Écrit db_profiles.json (UTF-8, indent)."""
    with open(_DB_PROFILES_PATH, "w", encoding="utf-8", newline="\n") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
        f.write("\n")


def _find_key_ci(profiles: dict, key: str) -> str | None:
    """Cherche une clé existante case-insensitive dans les profils."""
    target = key.strip().lower()
    for k in profiles:
        if str(k).strip().lower() == target:
            return str(k)
    return None
