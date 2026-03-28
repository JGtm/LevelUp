"""Constantes SQL pour la détection des scores d'équipe corrompus.

Contexte : L'API Halo Infinite retourne parfois dans CoreStats.Score (par équipe)
la somme des scores personnels au lieu du score objectif réel
(captures CTF, ticks Total Control, crânes Stockpile).
Ce comportement touche ~47 % des matchs BTB CTF/TC.

Le transformer est corrigé pour lire les stats mode-spécifiques
(CaptureTheFlagStats.FlagCaptures, ZonesStats.StrongholdScoringTicks).
Pour les données déjà en base avec un score > 100 pour un mode objectif,
on retourne NULL (données indisponibles) plutôt qu'une valeur aberrante.

Seuil 100 : Slayer max = 100 kills, donc tout score > 100 dans un mode
objectif est manifestement corrompu.

Alias attendus dans la requête : r = match_registry, p = match_participants.
"""

from __future__ import annotations

# Condition de détection des modes objectifs dont le score peut être corrompu
_OBJECTIVE_MODES = (
    "(r.game_variant_name LIKE '%CTF%' "
    "OR r.game_variant_name LIKE '%Total Control%' "
    "OR r.game_variant_name LIKE '%Stockpile%' "
    "OR r.game_variant_name LIKE '%One Flag%')"
)

# Seuil au-delà duquel le score d'un mode objectif est considéré corrompu.
# Slayer max = 100 kills → scores objectifs légitimes sont tous ≤ 100.
_SCORE_CORRUPTION_THRESHOLD = 100

# ---------------------------------------------------------------------------
# Fragments SQL réutilisables (r = match_registry, p = match_participants)
# ---------------------------------------------------------------------------

NORM_MY_TEAM_SCORE_SQL = (
    f"CASE\n"
    f"            WHEN {_OBJECTIVE_MODES}\n"
    f"                 AND GREATEST(COALESCE(r.team_0_score, 0), COALESCE(r.team_1_score, 0))"
    f" > {_SCORE_CORRUPTION_THRESHOLD}\n"
    f"            THEN NULL\n"
    f"            ELSE CASE WHEN p.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END\n"
    f"        END"
)

NORM_ENEMY_TEAM_SCORE_SQL = (
    f"CASE\n"
    f"            WHEN {_OBJECTIVE_MODES}\n"
    f"                 AND GREATEST(COALESCE(r.team_0_score, 0), COALESCE(r.team_1_score, 0))"
    f" > {_SCORE_CORRUPTION_THRESHOLD}\n"
    f"            THEN NULL\n"
    f"            ELSE CASE WHEN p.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END\n"
    f"        END"
)
