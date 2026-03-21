"""Constantes d'authentification LevelUp — App Azure publique unique.

L'application "LevelUp Halo" est enregistrée une seule fois sur portal.azure.com.
Chaque utilisateur se connecte via cette app publique (Device Code Flow).
Aucune configuration Azure côté utilisateur requise.

Prérequis de l'app :
- Supported account types : Personal Microsoft accounts only (Consumers)
- Authentication > Allow public client flows : Yes
- Pas de client_secret, pas de redirect URI
"""

from __future__ import annotations

# Application (client) ID de l'app "LevelUp Halo" (Azure App Registration)
LEVELUP_CLIENT_ID: str = "e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca"  # pragma: allowlist secret
# Fallback SPNKR_AZURE_CLIENT_ID (désactivé — réactiver si l'app Azure doit être surchargée) :
# import os
# LEVELUP_CLIENT_ID = os.environ.get("SPNKR_AZURE_CLIENT_ID", LEVELUP_CLIENT_ID) or LEVELUP_CLIENT_ID

# Authority Microsoft pour les comptes personnels (Xbox Live)
MSAL_AUTHORITY: str = "https://login.microsoftonline.com/consumers"

# Scopes Xbox Live — offline_access requis pour obtenir un refresh_token
XBOX_SCOPES: list[str] = ["Xboxlive.signin", "Xboxlive.offline_access"]

# Clé de stockage dans sync_meta (DuckDB) pour le cache MSAL sérialisé
MSAL_CACHE_DB_KEY: str = "msal_token_cache"

# Durée de vie des tokens Halo en cache process (spartan + clearance valides ~4h)
HALO_TOKEN_CACHE_TTL_S: int = 3600 * 4
