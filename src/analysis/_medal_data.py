"""Résolution medal_name_id → nom / description FR/EN.

Ce module est conservé pour compatibilité d'import.
Délègue à ``src.data.medal_definitions`` (couche données centralisée).
"""

from src.data.medal_definitions import resolve_medal_description, resolve_medal_name

__all__ = ["resolve_medal_description", "resolve_medal_name"]
