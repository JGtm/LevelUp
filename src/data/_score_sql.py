"""Constantes SQL pour la normalisation des scores d'équipe BTB objectifs.

Contexte : L'API Halo Infinite retourne parfois dans CoreStats.Score (par équipe)
la somme des scores personnels (somme PS) au lieu du score objectif réel
(captures CTF, rounds Total Control, skulls Stockpile).
Ce comportement touche ~55 % des matchs CTF/TC en BTB.

Exemple d'incohérence observée :
  team_0_score = 1       (score objectif correct)
  team_1_score = 22 270  (somme PS polluée par l'API)

Solution : quand GREATEST(team_0_score, team_1_score) > 500 pour un mode objectif,
l'API a manifestement stocké la somme PS — on utilise team_0_ps_score / team_1_ps_score
qui sont recalculés depuis match_participants (toujours cohérents entre les deux équipes).

Modes couverts : CTF, Total Control, Stockpile, One Flag.
Modes NON touchés : Slayer (score 100 = nombre de kills, toujours correct).

Alias attendus dans la requête : r = match_registry, p = match_participants.
"""

from __future__ import annotations

# Condition de détection des modes objectifs susceptibles d'être pollués
_OBJECTIVE_MODES = (
    "(r.game_variant_name LIKE '%CTF%' "
    "OR r.game_variant_name LIKE '%Total Control%' "
    "OR r.game_variant_name LIKE '%Stockpile%' "
    "OR r.game_variant_name LIKE '%One Flag%')"
)

# Seuil au-delà duquel le score est considéré comme une somme PS polluée.
# Les vrais scores objectifs sont ≤ ~50 (captures CTF, rounds TC, skulls Stockpile).
_SCORE_POLLUTION_THRESHOLD = 500

# ---------------------------------------------------------------------------
# Fragments SQL réutilisables (r = match_registry, p = match_participants)
# ---------------------------------------------------------------------------

NORM_MY_TEAM_SCORE_SQL = (
    f"CASE\n"
    f"            WHEN {_OBJECTIVE_MODES}\n"
    f"                 AND GREATEST(COALESCE(r.team_0_score, 0), COALESCE(r.team_1_score, 0))"
    f" > {_SCORE_POLLUTION_THRESHOLD}\n"
    f"            THEN CASE WHEN p.team_id = 0 THEN r.team_0_ps_score ELSE r.team_1_ps_score END\n"
    f"            ELSE CASE WHEN p.team_id = 0 THEN r.team_0_score ELSE r.team_1_score END\n"
    f"        END"
)

NORM_ENEMY_TEAM_SCORE_SQL = (
    f"CASE\n"
    f"            WHEN {_OBJECTIVE_MODES}\n"
    f"                 AND GREATEST(COALESCE(r.team_0_score, 0), COALESCE(r.team_1_score, 0))"
    f" > {_SCORE_POLLUTION_THRESHOLD}\n"
    f"            THEN CASE WHEN p.team_id = 0 THEN r.team_1_ps_score ELSE r.team_0_ps_score END\n"
    f"            ELSE CASE WHEN p.team_id = 0 THEN r.team_1_score ELSE r.team_0_score END\n"
    f"        END"
)
