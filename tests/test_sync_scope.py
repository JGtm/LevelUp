"""Tests unitaires pour SyncScope (src/data/sync/scope.py).

Vérifie :
- Construction par défaut et surcharges
- ``resolve()`` : all_data et force → data
- ``from_cli_args()``
- ``make_all()``
- ``has_any_option()``, ``needs_api``, ``needs_local_only``
- ``requested_types``
- Rétro-compatibilité : ajout d'un flag ne casse pas les constructeurs existants
"""

from __future__ import annotations

import argparse
from types import SimpleNamespace

import pytest

from src.data.sync.scope import (
    _ALL_DATA_FIELDS,
    _FORCE_MAP,
    _REQUESTED_TYPE_MAP,
    SyncScope,
)

# ─────────────────────────────────────────────────────────────────────────────
# Construction par défaut
# ─────────────────────────────────────────────────────────────────────────────


class TestSyncScopeDefaults:
    """Tous les flags sont False/None par défaut."""

    def test_all_data_fields_false(self) -> None:
        scope = SyncScope()
        for field_name in _ALL_DATA_FIELDS:
            assert getattr(scope, field_name) is False, f"{field_name} devrait être False"

    def test_all_force_fields_false(self) -> None:
        scope = SyncScope()
        for force_field in _FORCE_MAP:
            assert getattr(scope, force_field) is False, f"{force_field} devrait être False"

    def test_general_defaults(self) -> None:
        scope = SyncScope()
        assert scope.dry_run is False
        assert scope.max_matches is None
        assert scope.requests_per_second == 5
        assert scope.detection_mode == "or"
        assert scope.all_data is False

    def test_has_any_option_false_by_default(self) -> None:
        assert SyncScope().has_any_option() is False

    def test_needs_api_false_by_default(self) -> None:
        assert SyncScope().needs_api is False

    def test_needs_local_only_false_by_default(self) -> None:
        assert SyncScope().needs_local_only is False

    def test_requested_types_empty_by_default(self) -> None:
        assert SyncScope().requested_types == []


# ─────────────────────────────────────────────────────────────────────────────
# resolve() — all_data
# ─────────────────────────────────────────────────────────────────────────────


class TestResolveAllData:
    """``all_data=True`` active tous les champs listés dans _ALL_DATA_FIELDS."""

    def test_all_data_activates_all_fields(self) -> None:
        scope = SyncScope(all_data=True)
        scope.resolve()
        for field_name in _ALL_DATA_FIELDS:
            assert getattr(scope, field_name) is True, f"{field_name} devrait être True"

    def test_all_data_does_not_set_force_flags(self) -> None:
        scope = SyncScope(all_data=True)
        scope.resolve()
        for force_field in _FORCE_MAP:
            assert getattr(scope, force_field) is False, f"{force_field} ne devrait pas être activé"

    def test_all_data_has_any_option(self) -> None:
        scope = SyncScope(all_data=True)
        scope.resolve()
        assert scope.has_any_option() is True

    def test_all_data_needs_api(self) -> None:
        scope = SyncScope(all_data=True)
        scope.resolve()
        assert scope.needs_api is True

    def test_all_data_needs_local(self) -> None:
        scope = SyncScope(all_data=True)
        scope.resolve()
        assert scope.needs_local_only is True


# ─────────────────────────────────────────────────────────────────────────────
# resolve() — force_X → X
# ─────────────────────────────────────────────────────────────────────────────


class TestResolveForce:
    """Chaque ``force_X`` implique le champ ``X`` correspondant."""

    @pytest.mark.parametrize(
        "force_field,data_field",
        list(_FORCE_MAP.items()),
        ids=list(_FORCE_MAP.keys()),
    )
    def test_force_implies_data(self, force_field: str, data_field: str) -> None:
        scope = SyncScope(**{force_field: True})
        scope.resolve()
        assert (
            getattr(scope, data_field) is True
        ), f"{force_field}=True devrait impliquer {data_field}=True"

    def test_force_does_not_override_existing_true(self) -> None:
        """Si le champ data est déjà True, force ne le met pas False."""
        scope = SyncScope(medals=True, force_medals=True)
        scope.resolve()
        assert scope.medals is True

    def test_multiple_forces(self) -> None:
        scope = SyncScope(force_medals=True, force_shots=True, force_sessions=True)
        scope.resolve()
        assert scope.medals is True
        assert scope.shots is True
        assert scope.sessions is True
        # Non activés
        assert scope.events is False
        assert scope.skill is False


# ─────────────────────────────────────────────────────────────────────────────
# from_cli_args()
# ─────────────────────────────────────────────────────────────────────────────


class TestFromCliArgs:
    """Construction depuis un namespace argparse."""

    def test_basic_args(self) -> None:
        args = SimpleNamespace(
            medals=True,
            events=False,
            dry_run=True,
            max_matches=50,
            requests_per_second=10,
            detection_mode="and",
            all_data=False,
        )
        scope = SyncScope.from_cli_args(args)
        assert scope.medals is True
        assert scope.events is False
        assert scope.dry_run is True
        assert scope.max_matches == 50
        assert scope.requests_per_second == 10
        assert scope.detection_mode == "and"

    def test_all_data_resolved(self) -> None:
        args = SimpleNamespace(all_data=True)
        scope = SyncScope.from_cli_args(args)
        assert scope.medals is True
        assert scope.killer_victim is True

    def test_force_resolved(self) -> None:
        args = SimpleNamespace(force_medals=True)
        scope = SyncScope.from_cli_args(args)
        assert scope.force_medals is True
        assert scope.medals is True

    def test_unknown_attrs_ignored(self) -> None:
        args = SimpleNamespace(medals=True, unknown_flag=True)
        scope = SyncScope.from_cli_args(args)
        assert scope.medals is True
        assert not hasattr(scope, "unknown_flag")

    def test_missing_attrs_use_defaults(self) -> None:
        args = SimpleNamespace(medals=True)
        scope = SyncScope.from_cli_args(args)
        assert scope.events is False
        assert scope.max_matches is None

    def test_real_argparse_namespace(self) -> None:
        """Fonctionne avec un vrai argparse.Namespace."""
        ns = argparse.Namespace(
            medals=True,
            events=True,
            skill=False,
            personal_scores=False,
            performance_scores=False,
            aliases=False,
            accuracy=False,
            enemy_mmr=False,
            assets=False,
            participants=False,
            participants_scores=False,
            participants_kda=False,
            participants_shots=False,
            participants_damage=False,
            participants_avg_life=False,
            killer_victim=False,
            end_time=False,
            sessions=False,
            shots=False,
            citations=False,
            participants_enrich=False,
            dry_run=False,
            max_matches=None,
            requests_per_second=5,
            detection_mode="or",
            all_data=False,
            force_medals=False,
            force_accuracy=False,
            force_shots=False,
            force_participants_shots=False,
            force_participants_damage=False,
            force_participants_avg_life=False,
            force_enemy_mmr=False,
            force_aliases=False,
            force_assets=False,
            force_participants=False,
            force_end_time=False,
            force_sessions=False,
            force_citations=False,
            force_participants_enrich=False,
        )
        scope = SyncScope.from_cli_args(ns)
        assert scope.medals is True
        assert scope.events is True
        assert scope.skill is False


# ─────────────────────────────────────────────────────────────────────────────
# make_all()
# ─────────────────────────────────────────────────────────────────────────────


class TestMakeAll:
    """``make_all()`` crée un scope tout activé."""

    def test_all_fields_true(self) -> None:
        scope = SyncScope.make_all()
        for field_name in _ALL_DATA_FIELDS:
            assert getattr(scope, field_name) is True

    def test_overrides(self) -> None:
        scope = SyncScope.make_all(max_matches=100, requests_per_second=3)
        assert scope.max_matches == 100
        assert scope.requests_per_second == 3
        assert scope.medals is True

    def test_dry_run_override(self) -> None:
        scope = SyncScope.make_all(dry_run=True)
        assert scope.dry_run is True
        assert scope.medals is True


# ─────────────────────────────────────────────────────────────────────────────
# has_any_option()
# ─────────────────────────────────────────────────────────────────────────────


class TestHasAnyOption:
    """``has_any_option()`` détecte si au moins un type est activé."""

    @pytest.mark.parametrize("field_name", _ALL_DATA_FIELDS)
    def test_single_field(self, field_name: str) -> None:
        scope = SyncScope(**{field_name: True})
        assert scope.has_any_option() is True

    def test_no_option(self) -> None:
        assert SyncScope().has_any_option() is False

    def test_only_general_options(self) -> None:
        scope = SyncScope(dry_run=True, max_matches=10, detection_mode="and")
        assert scope.has_any_option() is False


# ─────────────────────────────────────────────────────────────────────────────
# needs_api / needs_local_only
# ─────────────────────────────────────────────────────────────────────────────


class TestNeedsApi:
    """``needs_api`` est True pour les types nécessitant l'API."""

    @pytest.mark.parametrize(
        "field_name",
        [
            "medals",
            "events",
            "skill",
            "personal_scores",
            "performance_scores",
            "aliases",
            "accuracy",
            "enemy_mmr",
            "assets",
            "participants",
            "shots",
            "participants_scores",
            "participants_kda",
            "participants_shots",
            "participants_damage",
            "participants_avg_life",
            "participants_enrich",
        ],
    )
    def test_api_fields(self, field_name: str) -> None:
        scope = SyncScope(**{field_name: True})
        assert scope.needs_api is True, f"{field_name} devrait nécessiter l'API"

    @pytest.mark.parametrize(
        "field_name",
        ["killer_victim", "end_time", "sessions", "citations"],
    )
    def test_local_fields_not_api(self, field_name: str) -> None:
        scope = SyncScope(**{field_name: True})
        assert scope.needs_api is False, f"{field_name} ne devrait pas nécessiter l'API"


class TestNeedsLocalOnly:
    """``needs_local_only`` est True pour les types locaux."""

    @pytest.mark.parametrize(
        "field_name",
        ["killer_victim", "end_time", "sessions", "citations"],
    )
    def test_local_fields(self, field_name: str) -> None:
        scope = SyncScope(**{field_name: True})
        assert scope.needs_local_only is True

    def test_api_field_not_local(self) -> None:
        scope = SyncScope(medals=True)
        assert scope.needs_local_only is False


# ─────────────────────────────────────────────────────────────────────────────
# requested_types
# ─────────────────────────────────────────────────────────────────────────────


class TestRequestedTypes:
    """``requested_types`` retourne les clés pour le bitmask."""

    def test_empty(self) -> None:
        assert SyncScope().requested_types == []

    def test_single(self) -> None:
        scope = SyncScope(medals=True)
        assert scope.requested_types == ["medals"]

    def test_multiple(self) -> None:
        scope = SyncScope(medals=True, events=True, skill=True)
        types = scope.requested_types
        assert "medals" in types
        assert "events" in types
        assert "skill" in types
        assert len(types) == 3

    def test_local_fields_not_in_bitmask(self) -> None:
        """Les champs locaux (killer_victim, sessions, etc.) ne sont pas dans le bitmask."""
        scope = SyncScope(killer_victim=True, sessions=True, citations=True, end_time=True)
        assert scope.requested_types == []

    def test_all_data_requested_types(self) -> None:
        scope = SyncScope.make_all()
        types = scope.requested_types
        assert len(types) == len(_REQUESTED_TYPE_MAP)

    def test_matches_requested_type_map(self) -> None:
        """Chaque clé de _REQUESTED_TYPE_MAP est un champ valide de SyncScope."""
        scope = SyncScope()
        for field_name in _REQUESTED_TYPE_MAP:
            assert hasattr(scope, field_name), f"{field_name} absent de SyncScope"


# ─────────────────────────────────────────────────────────────────────────────
# Registres cohérents
# ─────────────────────────────────────────────────────────────────────────────


class TestRegistryConsistency:
    """Vérifie la cohérence entre les registres et les champs SyncScope."""

    def test_all_data_fields_exist_in_scope(self) -> None:
        scope = SyncScope()
        for field_name in _ALL_DATA_FIELDS:
            assert hasattr(scope, field_name), f"{field_name} absent de SyncScope"

    def test_force_map_targets_exist(self) -> None:
        scope = SyncScope()
        for force_field, data_field in _FORCE_MAP.items():
            assert hasattr(scope, force_field), f"{force_field} absent de SyncScope"
            assert hasattr(scope, data_field), f"{data_field} absent de SyncScope"

    def test_requested_type_map_fields_exist(self) -> None:
        scope = SyncScope()
        for field_name in _REQUESTED_TYPE_MAP:
            assert hasattr(scope, field_name), f"{field_name} absent de SyncScope"

    def test_force_map_values_in_all_data_fields(self) -> None:
        """Chaque cible de _FORCE_MAP est dans _ALL_DATA_FIELDS."""
        for data_field in _FORCE_MAP.values():
            assert (
                data_field in _ALL_DATA_FIELDS
            ), f"{data_field} dans _FORCE_MAP mais absent de _ALL_DATA_FIELDS"


# ─────────────────────────────────────────────────────────────────────────────
# Idempotence
# ─────────────────────────────────────────────────────────────────────────────


class TestIdempotence:
    """``resolve()`` est idempotent."""

    def test_double_resolve(self) -> None:
        scope = SyncScope(all_data=True, force_medals=True)
        scope.resolve()
        scope.resolve()
        assert scope.medals is True
        assert scope.events is True

    def test_resolve_without_flags(self) -> None:
        scope = SyncScope()
        scope.resolve()
        assert scope.has_any_option() is False
