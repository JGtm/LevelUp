"""Score de performance Halo Infinite — Module façade.

Ce module re-exporte les fonctions des sous-modules :
- _performance_relative : Score relatif par match (vs historique personnel)
- _performance_session  : Score agrégé par session (composantes pondérées)
"""

from __future__ import annotations

# ── Re-exports : score relatif (par match) ────────────────────────────────────
# Alias interne exporté pour les tests
from src.analysis._performance_relative import (  # noqa: F401  # noqa: F401
    _compute_rank_performance,
    _normalize_df,
    _percentile_rank,
    _percentile_rank_inverse,
    _prepare_history_metrics,
    _safe_col,
    compute_performance_series,
    compute_relative_performance_score,
)

# ── Re-exports : score session ────────────────────────────────────────────────
from src.analysis._performance_session import (  # noqa: F401
    ScoreComponent,
    compute_session_performance_score_v1,
    compute_session_performance_score_v2,
)
