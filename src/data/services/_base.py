"""Protocole de base pour les services stateless.

Les services de ce package suivent le pattern stateless :
- Pas d'état interne (pas de ``__init__`` avec ``self.db_path``).
- Toutes les méthodes sont des ``@staticmethod`` ou ``@classmethod``.
- Chaque méthode reçoit les données dont elle a besoin (DataFrame, chemin DB…).

Ce module documente le contrat via un ``Protocol`` structurel (duck typing).
Aucun service n'a besoin d'hériter de ``StatelessServiceProtocol`` explicitement.

Exemple de service conforme::

    class WinLossService:
        @staticmethod
        def compute_win_loss(df: pl.DataFrame, *, lang: str = "fr") -> dict: ...

Vérification de conformité (optionnelle, pour les tests)::

    from src.data.services._base import StatelessServiceProtocol
    assert isinstance(WinLossService, type)  # stateless, pas d'instance
"""

from __future__ import annotations

from typing import Protocol, runtime_checkable


@runtime_checkable
class StatelessServiceProtocol(Protocol):
    """Protocole structurel pour les services stateless.

    Un service est conforme s'il n'a pas de ``__init__`` avec état
    (pas de ``self.db_path``, etc.). Toutes ses méthodes sont des
    ``@staticmethod`` ou ``@classmethod``.

    Services conformes actuels :
        - ``WinLossService``   (win_loss_service.py)
        - ``TeammatesService`` (teammates_service.py)
        - ``TimeseriesService`` (timeseries_service.py)
    """
