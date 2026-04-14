"""Factory pour créer un client API Halo Infinite.

Centralise l'instanciation du client API. Permet de switcher de backend
(SPNKr, Grunt, etc.) sans modifier les consommateurs.

Usage:
    async with create_api_client() as client:
        history = await client.get_match_history("SpartanB")

    # Backend explicite
    async with create_api_client(backend="spnkr", tokens=tokens) as client:
        ...

"""

from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from src.data.sync._tokens import Tokens
    from src.ports.api import HaloAPIPort


def create_api_client(
    backend: str = "spnkr",
    *,
    tokens: Tokens | None = None,
    requests_per_second: int = 5,
    lang: str | None = None,
) -> HaloAPIPort:
    """Crée un client API Halo Infinite pour le backend demandé.

    Args:
        backend: Nom du backend (``spnkr``).
        tokens: Tokens pré-fournis (sinon récupérés depuis env).
        requests_per_second: Rate limiting.
        lang: Code BCP-47 pour Accept-Language (ex: 'fr-FR').
              Permet de récupérer les PublicName localisés depuis Discovery UGC.

    Returns:
        Instance de client implémentant ``HaloAPIPort``.

    Raises:
        ValueError: Si le backend est inconnu.
    """
    if backend == "spnkr":
        from src.data.sync.api_client import SPNKrAPIClient

        return SPNKrAPIClient(tokens=tokens, requests_per_second=requests_per_second, lang=lang)

    raise ValueError(f"Backend API inconnu : {backend!r}. Backends disponibles : 'spnkr'.")
