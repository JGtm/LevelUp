# Handoff chantier Tactique — 2026-09-07 (arret sur limite de quota)

Branche `feat/tactique`, worktree `LevelUp-wt-tactique`, HEAD pousse `c894e0282` (gates locaux verts :
vet, unit, integration `-p 1` sync/persist/migration/duckdb/service/scheduler, typecheck, vitest complet).

## Etat du plan (`.ai/PLAN_TACTIQUE_2026-09-06.md`)
- Closes, CI verte : 1, 2, 3 (graphes de l'echange, page Escouade), 4, 4 bis, 6, 7A.
- 7C : code livre (2 tables append-only au sync, lecture `isole`), revue rondes 1 et 2 faites ;
  **UN FAIT FAUX POSSIBLE reste ouvert** (horloges participation/film) — voir
  `DECOUVERTES_TACTIQUE_2026-09-07.md`, premiere entree. A regler avant merge, de preference en
  RETIRANT l'usage de la participation (V1).
- 5 (vue d'analyse tactique = les cartes de chaleur) : NON FAITE. Le gel « attendre le lot D » est
  leve par l'utilisateur : a faire sans attendre D (peintre `heatmapLayer.ts` copie UNE fois dans
  `features/tactical/`, fusion avec D consignee). Spec = maquette artefact
  https://claude.ai/code/artifact/034b1915-ea1b-49d9-a4f5-06aa9fbd1ccd, composants existants
  (`SectionCard`, `KPIStrip`, `EmptyState`), contrat deja livre (`POST .../tactical/{map_id}/raster`,
  questions `morts|kills|gagne|temps|routes|isole`, `grappes`, `spawn`, `matchs_en_attente` /
  `matchs_non_cuisables` -> message « traitement en cours » vs « donnees non disponibles »).
- 7B (nuage isolement x couverture, Escouade) : NON FAITE. Spec = maquette
  https://claude.ai/code/artifact/4c520da6-775b-4fb8-9d6e-dd6aa8d629b9 ; source = `match_death_context_latest`
  + journal ; par session, taille = morts, plancher 30, session < 5 morts exclue.
- 8 : cloture (thought_log, `make gate-push`, revue du diff integral) : NON FAITE.

## Regles a la reprise (decision utilisateur 2026-09-07)
- Ne plus traiter les decouvertes : les consigner dans `DECOUVERTES_TACTIQUE_2026-09-07.md`.
- Aucune extension de perimetre. Une ronde de revue par VUE (rendu + contrat), pas de revue de substrat.
- Sobriete : executeurs Sonnet pour le web, Opus seulement si necessaire ; pas de double relecteur.
