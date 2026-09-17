# Plan — correctifs de la revue adversariale des 8 lots feat/v75 (2026-09-17)

Source : entrée thought_log « Revue adversariale des 8 lots livrés sur feat/v75
(ea9ba1b4e..016703f8e) ». Décisions utilisateur du 2026-09-17 : corriger P1-1, P1-2, P1-3
(garde par comptage des équipes dans match_participants), P1-4, P1-5, P1-6 (tests aux bornes),
P1-7 = exception datée (commentaire seul), P2-1 = laisser + commentaire, P2-2 = doc + ordre de
livraison, Cividis = recherche « deux surfaces » et retrait de l'exemption.

Branches : `fix/revue-lots-v75` (web + plan + journal, worktree LevelUp-wt-revue-fix) et
`fix/revue-lots-v75-go` (Go, worktree LevelUp-wt-revue-fix-go), fusionnée dans la première à la
clôture. Exécutants Opus, pilote = session principale. Ronde 2 de relecture sur les seules
corrections (sync/ touché). Merge final dans feat/v75 après CI verte.

## Étape 1 — Go (branche fix/revue-lots-v75-go)

- [ ] 1.1 P1-1 Voleur : Q32e écarte les victimes bots sur la branche « candidat au vol »
      (`victim_xuid IS NOT NULL`), + test Q32e qui rougit sans le filtre.
- [ ] 1.2 P1-2 Dominance : durée absente (NULL/0) → WARN structuré + `ComputeFragContrastDominance`
      reçoit 0 explicitement ; test sync « durée NULL » qui vérifie le repli ET la trace.
- [ ] 1.3 P1-3 Dominance : garde « exactement 2 équipes » par `COUNT(DISTINCT team_id)` sur
      `match_participants` ; retrait du filtre `IN (0,1)` comme seule garde ; test à 3 équipes.
- [ ] 1.4 P1-6 : tests aux bornes des trois seuils (10/9 frags, 1,15/1,149, 75 %/74 %) des deux
      côtés.
- [ ] 1.5 P2-1 : commentaire « exclusion volontaire des flags 6/7 » dans
      `career_repo_top_matches.go`.
- [ ] 1.6 P2-2 : doc de `steps_player_reset_dominance_none.go` corrigée (le recalcul rejoue toute
      la chaîne ; les 0 dont les données ont évolué peuvent recevoir 3/4/5 ; livrer avec 1.2/1.3).
- Gate : `go test -count=1 -tags=integration ./internal/analysis/... ./internal/sync/...
  ./internal/platform/duckdb/... ./internal/migration/... ./internal/service/...` vert ;
  `golangci-lint run ./...` 0 issue sur le diff ; baseline JSONL des tests à jour si un test est
  renommé.

## Étape 2 — Web (branche fix/revue-lots-v75)

- [ ] 2.1 P1-4 : colonne « Assistances » conditionnelle dans `MatchEncountersTable` (absente sur
      Carrière > Joueurs les plus croisés), test de rendu.
- [ ] 2.2 P1-5 : composant `CopyButton` partagé (4 sites migrés : ShareLinkButton,
      IdentitiesSection, MatchHeader.card, CopyCodeButton), durée 2 s unique, timer nettoyé,
      échec presse-papier journalisé (log.error) sans faux « copié », garde-rail grep interdisant
      `setCopied(false)` hors du composant, tests (échec presse-papier, retour à l'état initial).
- [ ] 2.3 P1-7 : commentaire daté sur `squadPerformanceLineCharts.ts` (exception lisibilité,
      décision utilisateur 2026-09-17).
- [ ] 2.4 Cividis : `assist-received` / `assist-given` re-choisis pour passer le contraste 3:1 sur
      les DEUX surfaces + dE >= 15 avec les autres jetons de la famille ; `contrast: 'both'` pour
      cividis dans `combatStatTokens.test.ts` ; snapshot de palette mis à jour.
- Gate : `npm run typecheck`, `npm run lint` (0 erreur), `npm run lint:colors`, `npm run
  lint:fields`, vitest sur `src/features/match-view src/features/career src/features/auth
  src/components src/features/admin src/features/squad src/lib/accessibility src/lib/i18n` vert.

## Étape 3 — Clôture (pilote)

- [ ] 3.1 Fusion `fix/revue-lots-v75-go` → `fix/revue-lots-v75`, gates complets rejoués.
- [ ] 3.2 Ronde 2 de relecture adversariale sur les seules corrections (1 relecteur Go, 1 web).
- [ ] 3.3 Push, CI verte au niveau job.
- [ ] 3.4 Journal thought_log (entrée revue + entrée correctifs), plan statué.
- [ ] 3.5 Merge dans feat/v75 (demander avant).

## Découvertes (non traitées)

- (P2 de la revue restant ouverts, cf. thought_log : ratchets OKLab/anti-emprunt aveugles,
  ReplayCountersBadge jetons non typés, tests 6/7 non étendus côté front, « killstreak »
  passe le garde-rail, waitForExportLayout non discriminé, véhicule non dessiné pendant le
  chargement de la bordure, stat-deaths = outcome-loss dans 3 palettes.)

## Journal

- 2026-09-17 : plan créé, exécution lancée.
