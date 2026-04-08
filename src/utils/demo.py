"""Détection du mode démo public.

Ce module expose une seule fonction utilitaire pour savoir si l'application
tourne en mode démo (accès public, sync désactivé, données restreintes).
"""

from __future__ import annotations

import os


def is_demo_mode() -> bool:
    """Retourne True si l'app tourne en mode démo public.

    Activé via la variable d'environnement ``LEVELUP_DEMO_MODE=true`` (ou ``1``, ``yes``).
    C'est le cas du conteneur Docker ``levelup-demo`` exposé sur le sous-domaine public.
    """
    return os.environ.get("LEVELUP_DEMO_MODE", "").lower() in ("1", "true", "yes")
