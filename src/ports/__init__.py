"""
Couche ports — interfaces structurelles (typing.Protocol).

Architecture Ports & Adapters (hexagonale) :
- Les ports définissent les contrats indépendamment des implémentations.
- Les couches analysis/, app/ et ui/ importent d'ici pour ne pas dépendre
  de src.data (couche adaptateur).

Exports :
    DataRepository  — contrat lecture données joueur
    HaloAPIPort     — contrat client API Halo Infinite
"""

from src.ports.api import HaloAPIPort
from src.ports.repository import DataRepository

__all__ = ["DataRepository", "HaloAPIPort"]
