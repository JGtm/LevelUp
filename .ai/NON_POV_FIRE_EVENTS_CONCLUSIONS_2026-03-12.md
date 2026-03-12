# Conclusions - Fire Events non-POV (2026-03-12)

## Contexte

Un retour externe indiquait que les fire events du film sont exploitables au-dela du seul POV, alors que le pipeline LevelUp etait configure comme "POV Section 2 uniquement".

## Cause racine

Le parseur bas niveau etait deja capable de scanner un `player_index` arbitraire, mais l'orchestrateur de service imposait une bifurcation stricte :

- POV (`xuid == pov_xuid`) : Section 2 (`scan_fire_events`, `pi=1`)
- non-POV : Formula A (snapshot par chunk) uniquement

Conclusion : la limitation etait dans la logique de routage du service, pas dans la capacite technique du parseur.

## Correctif applique

Fichier principal : `src/data/services/weapon_extraction_service.py`

1. Conservation du comportement POV existant (`pi=1`) pour eviter les regressions.
2. Ajout d'une tentative Section 2 pour les non-POV avec le `player_index` detecte.
3. Fusion conservative des attributions non-POV :
   - remplace T1 seulement si T1 est faible/non resolu (`none`/`low` ou `weapon_id` non resolu)
   - et si la ligne fire event est exploitable (`medium`/`high`, `weapon_id` resolu)
   - sinon, T1 est conservee.

Helpers ajoutes :

- `_is_resolved_weapon_row`
- `_fire_row_beats_t1_row`
- `_merge_non_pov_attributions`

## Tests ajoutes

Fichier : `tests/test_weapon_service.py`

- `test_prefers_fire_event_when_t1_unresolved`
- `test_keeps_t1_when_t1_is_strong`

Ces tests verrouillent la politique de fusion non-POV.

## Validation

- Diagnostics editeur sur les fichiers modifies : aucune erreur.
- Execution `pytest` impossible dans cette session faute d'environnement Python projet configure (`.venv` absent, `pytest` absent du Python global).

## Impact attendu

- Le pipeline ne considere plus par principe que les fire events sont exclusifs au POV.
- Les non-POV peuvent desormais beneficier des fire events quand ils sont detectables.
- La fusion conservative limite les regressions en preservant les attributions T1 solides.

## Suivi recommande

1. Rejouer un backfill weapon sur un echantillon de matchs avec ground truth externe.
2. Mesurer le taux de remplacement T1 -> Section 2 par joueur non-POV.
3. Ajuster les regles de fusion si necessaire (ex: prise en compte de `delta_ms` et `swap_detected`).