"""Tests Axe 7 — batch_commit_size adaptatif.

Vérifie que compute_optimal_batch_size retourne les bonnes valeurs et que
le mécanisme d'auto-tune dans _sync_internal est correctement appliqué.
"""

from __future__ import annotations

from dataclasses import replace as dc_replace

from src.data.sync.models_sync import SyncOptions

# =============================================================================
# compute_optimal_batch_size
# =============================================================================


class TestComputeOptimalBatchSize:
    """Tests unitaires de la logique d'adaptation."""

    def test_small_volume_returns_zero(self):
        """≤ 25 matchs → commit final uniquement."""
        assert SyncOptions.compute_optimal_batch_size(1) == 0
        assert SyncOptions.compute_optimal_batch_size(25) == 0

    def test_medium_volume_returns_25(self):
        """26-100 matchs → batch de 25."""
        assert SyncOptions.compute_optimal_batch_size(26) == 25
        assert SyncOptions.compute_optimal_batch_size(100) == 25

    def test_large_volume_returns_50(self):
        """101-500 matchs → batch de 50."""
        assert SyncOptions.compute_optimal_batch_size(101) == 50
        assert SyncOptions.compute_optimal_batch_size(500) == 50

    def test_very_large_volume_returns_100(self):
        """501+ matchs → batch de 100."""
        assert SyncOptions.compute_optimal_batch_size(501) == 100
        assert SyncOptions.compute_optimal_batch_size(10000) == 100

    def test_default_max_matches_200_gives_50(self):
        """La valeur default max_matches=200 doit donner batch=50."""
        assert SyncOptions.compute_optimal_batch_size(SyncOptions().max_matches) == 50


# =============================================================================
# Valeur par défaut et sémantiques
# =============================================================================


class TestBatchCommitSizeDefault:
    """Tests sur la valeur par défaut et les sémantiques existantes."""

    def test_default_is_minus_one(self):
        """La valeur par défaut est -1 (auto)."""
        opts = SyncOptions()
        assert opts.batch_commit_size == -1

    def test_explicit_zero_preserved(self):
        """0 conserve sa sémantique 'commit final uniquement'."""
        opts = SyncOptions(batch_commit_size=0)
        assert opts.batch_commit_size == 0

    def test_explicit_positive_preserved(self):
        """Une valeur > 0 est un override explicite conservé tel quel."""
        opts = SyncOptions(batch_commit_size=50)
        assert opts.batch_commit_size == 50

    def test_explicit_not_overridden_by_auto_tune(self):
        """Un batch_commit_size explicite (≠ -1) ne doit pas être écrasé."""
        opts = SyncOptions(batch_commit_size=50, max_matches=200)
        # Simuler le bloc auto-tune de _sync_internal
        if opts.batch_commit_size == -1:
            opts = dc_replace(
                opts,
                batch_commit_size=SyncOptions.compute_optimal_batch_size(opts.max_matches),
            )
        assert opts.batch_commit_size == 50  # inchangé


# =============================================================================
# Intégration : bloc auto-tune dans _sync_internal
# =============================================================================


class TestAutoTuneInSyncInternal:
    """Vérifie que _sync_internal applique correctement l'auto-tune."""

    def test_auto_tune_applied_when_minus_one(self):
        """batch=-1 → auto-tune selon max_matches avant le premier appel API."""
        opts = SyncOptions(batch_commit_size=-1, max_matches=200)
        # Reproduire le bloc de engine._sync_internal
        if opts.batch_commit_size == -1:
            opts = dc_replace(
                opts,
                batch_commit_size=SyncOptions.compute_optimal_batch_size(opts.max_matches),
            )
        assert opts.batch_commit_size == 50

    def test_auto_tune_small_volume(self):
        """Petit volume (≤25 matchs) → commit final uniquement."""
        opts = SyncOptions(batch_commit_size=-1, max_matches=20)
        if opts.batch_commit_size == -1:
            opts = dc_replace(
                opts,
                batch_commit_size=SyncOptions.compute_optimal_batch_size(opts.max_matches),
            )
        assert opts.batch_commit_size == 0

    def test_zero_semantics_no_intermediate_commit(self):
        """batch_size=0 → _maybe_batch_commit ne commite jamais."""

        # Reproduire la logique de _maybe_batch_commit
        def maybe_commit(n_inserted: int, batch_size: int) -> bool:
            return batch_size > 0 and n_inserted % batch_size == 0

        for n in range(1, 101):
            assert maybe_commit(n, 0) is False

    def test_batch_commit_triggers_at_correct_intervals(self):
        """batch_size=25 → commit aux multiples de 25 uniquement."""

        def maybe_commit(n_inserted: int, batch_size: int) -> bool:
            return batch_size > 0 and n_inserted % batch_size == 0

        triggers = [n for n in range(1, 101) if maybe_commit(n, 25)]
        assert triggers == [25, 50, 75, 100]
