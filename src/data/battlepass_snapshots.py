"""Façade publique des snapshots joueur battle pass."""

from __future__ import annotations

from src.data import _battlepass_snapshots as _snapshots

BattlepassProgressSnapshot = _snapshots.BattlepassProgressSnapshot
persist_battlepass_snapshots = _snapshots.persist_battlepass_snapshots

__all__ = ["BattlepassProgressSnapshot", "persist_battlepass_snapshots"]
