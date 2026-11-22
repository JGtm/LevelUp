# Plan : « Impact des coéquipiers » par session (Escouade, 8.3.0)

Ouvert le 2026-11-22. Contrat d'exécution : skill `plan-execution` + complément `levelup-plan`.
Branche : `feat/impact-sessions` (depuis `feat/medailles-bouclier`, avant sa fusion dans `release/v8.3.0` ; se fusionne après elle),
worktree `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-impact-sessions`. Intégration : `release/v8.3.0`.

## Objectif

Sur Escouade › Contributions, un seul tableau remplace la matrice « Impact des coéquipiers » et le graphe
« Points d'impact par soirée et par rôle ». Critère observable : la maquette validée par l'utilisateur
(https://claude.ai/artifact/9nY1aZsdqAKrbw17YaLiNU, version 5) rendue avec les vraies données : une ligne
par session, la session sélectionnée ouverte d'office et teintée, chaque ligne dépliable sur la matrice
de sa session, une ligne « Toutes les sessions » dépliable sur le décompte par rôle.

## Décisions (validées par l'utilisateur le 2026-11-22 sur la maquette)

- **D-1 Lignes** : une par session, de la plus récente à la plus ancienne ; les sessions sont celles de
  l'historique actuel (`squadimpact.SelectEvenings` : sessions de la population, puis jusqu'à 10 sessions
  antérieures de la composition, 11 au plus). Le mot « soirée » disparaît de l'interface : « session ».
- **D-2 Colonnes d'une ligne** : « Session » (date, nombre de matchs, chevron), « Bilan » (barre composite
  victoires / défaites au format de la colonne « Rencontres » de l'Historique des rencontres : nombre
  vert, barre, nombre rouge ; jetons `outcome-win` / `outcome-loss`), puis une colonne par joueur : mini-barre
  centrée sur zéro + net par match signé (une décimale), vert / rouge classiques (`outcome-win` /
  `outcome-loss`), jamais les rampes `impact-gain/loss`.
- **D-3 Net par match** d'un joueur dans une session = ses points de la session / ses matchs joués dans la
  session (matchs où il est dans l'équipe alliée), servi par le serveur.
- **D-4 Ligne dépliée** : la matrice de la session pleine largeur, sans cadre dans le cadre, fond en retrait.
  Colonnes : joueur ; un match par colonne (TOUS les matchs de la session, ordre chronologique ; numéro
  centré, trait vert ou rouge dessous selon l'issue ; un point « · » quand le joueur n'a aucun rôle) ; une
  colonne par rôle (icône en en-tête, compte, meilleur et pire de l'escouade surlignés comme la matrice
  actuelle : comptes non nuls, au moins deux valeurs distinctes, rôle qui coûte des points = plus grand
  compte pire) ; « Score » (net de la session, même surlignage, couronne au premier, banane au dernier si
  négatif, passager clandestin au dernier sinon : règle actuelle de `buildRows`). AUCUN titre de groupe
  au-dessus des colonnes.
- **D-5 « Toutes les sessions »** : première ligne, repliée par défaut, comptée sur les sessions affichées
  (somme des lignes : ce qu'on voit s'additionne). Dépliée : décompte par rôle + score, même surlignage.
- **D-6 Session sélectionnée** : celles de `picked_sessions` du filtre de la page (plusieurs possibles)
  s'ouvrent d'office et ont un fond teinté (pas de pastille) ; sans sélection (mode période) : la plus
  récente s'ouvre, sans teinte. Les autres se déplient au clic ou au clavier (Entrée / Espace),
  plusieurs à la fois.
- **D-7 Pleine largeur** quoi qu'il arrive, colonnes fixes (rien ne bouge au dépliage), défilement
  horizontal dans le cadre en dessous d'une largeur minimale. En-têtes légers (petits libellés), pas de
  bandeau gris. Infobulles des rôles = celles d'aujourd'hui (nom + description).
- **D-8 Données** : un payload neuf `squad_impact_sessions` remplace `impact_matrix` et
  `squad_impact_history` sur la réponse de `/pages/teammates` ; les deux anciens champs, leurs types, leurs
  constructeurs et leurs composants web sont SUPPRIMÉS (règle « 0 code mort », contrat mis à jour avec
  justification). Les rôles viennent toujours de `rolesDesMatchs` / `squadimpact.RolesOfMatch` et du même
  barème (une seule définition).
- **D-9 Matchs sans libellé de session** : hors du tableau (comme l'historique actuel) ; journalisé une
  fois par requête s'il y en a.
- **D-10 Jetons `impact-gain-*` / `impact-loss-*`** : supprimés s'ils n'ont plus de lecteur après le lot W
  (palettes, tests de jetons), sinon gardés avec leur lecteur nommé au journal.

## Étapes

Statuts : `[ ]`, `[x]`, `[~]` (référence), `[!]` (justifié). Aucune case vide à la clôture. L'étape 0 est
couverte par la cartographie du 2026-11-22 (agent d'exploration, rapport recopié en annexe A).

### Lot G.1 — contrat (Sonnet) — rapide — chemin critique

- [ ] Types `domain.SquadImpactSessions` (joueurs ; barème ; sessions : libellé, début, matchs
  {match_id, issue}, par joueur {matchs joués, points, comptes par rôle, rôles par match} ; totaux) ;
  champ sur `TeammatesPageResponse` ; `make openapi-gen` + `make generate-types`. Anciens champs encore
  présents à ce stade.
- Gate : `go build ./...`, `go test ./contracttest/...`, `make generate-types` sans diff résiduel.

### Lot G.2 — calcul (Sonnet) — moyen — après G.1, en parallèle de W

- [ ] Fonction pure `squadimpact.BuildSessions` (à partir des rôles par match, des sessions de
  `SelectEvenings`, de l'équipe alliée) ; tests purs.
- [ ] Service : remplissage du payload, suppression de `assemblerMatrice`, `BuildHistory` et des deux
  anciens champs (types, schémas, tests) ; journal D-9.
- [ ] Témoin avant/après : sur les fixtures de `teammates_squad_impact_test.go`, les comptes par rôle et
  les points par joueur et par session du nouveau payload égalent ceux de l'ancien historique, et les
  totaux de la population ceux de l'ancienne matrice (test écrit AVANT la suppression, gardé ensuite sur
  des valeurs figées).
- Gate : `go test ./...`, `go test -tags=integration -p 1 ./internal/service/... ./internal/platform/duckdb/...`,
  `make go-api-lint`.

### Lot W — tableau web (Sonnet) — lourd — après G.1

- [ ] Logique pure (`impactSessions.logic.ts`) : lignes, net par match, état ouvert initial (D-6),
  surlignage (extrait des helpers de `SquadImpactScoreboard` vers un module partagé de la feature).
- [ ] Composant `SquadImpactSessionsTable` (TanStack Table, patron des lignes dépliables de
  `MatchScoreboard`), barre composite V/D (exporter un `WinLossSplitBar` depuis
  `features/_shared/EncounterSplitBars.tsx`), textes fr/en, infobulle de titre.
- [ ] Montage dans `SquadContributionsPage` à la place des deux blocs ; transmission de la session
  sélectionnée selon `filterBarBoundary.guard.test.ts`.
- [ ] Suppression de `SquadImpactScoreboard*`, `impact/*` (historique), fixtures, textes devenus morts ;
  D-10 ; cliquets (`functionSize`, contrat) abaissés ou mis à jour avec justification.
- Gate : `npx vitest run src/features/squad src/lib src/components --maxWorkers=2`, `npm run typecheck`
  (après purge de `node_modules/.tmp`), `npm run lint`, `node tools/lint-cross-feature-imports.mjs`.

### Clôture — pilote

- [ ] Z.1 `npx vitest run` complet, `go test -tags=integration -p 1 ./...`, `make gate-push` (avec
  `LEVELUP_REPO_ROOT` et `LEVELUP_IDENTITES_ORIGINALES` du dépôt principal).
- [ ] Z.2 Revue adversariale (Fable, une ronde), P0/P1 corrigés, P2 au backlog.
- [ ] Z.3 Lignes de vérification dans l'artefact `verifs` ; `thought_log` ; fusion dans `release/v8.3.0` ;
  CI lue.

## Découvertes

## Reprise de session

Lire ce fichier puis `git log --oneline -10` dans le worktree.

## Journal

- 2026-11-22 : plan ouvert ; décisions D-1 à D-10 tirées de la maquette validée par l'utilisateur.

## Annexe A — cartographie (2026-11-22, chemins relatifs à la racine du dépôt)

- Réponse : `domain.TeammatesPageResponse` (`apps/go-api/internal/domain/teammates.go:558` `impact_matrix`,
  `:565` `squad_impact_history`), POST `/pages/teammates` ; schémas `openapi.yaml` (`SquadImpact*`, 11 schémas
  listés dans `apps/web/src/lib/api/contract-surface.snapshot.json:855-865`).
- La matrice ne porte ni session ni date par match ; l'historique ne porte pas les matchs (comptes seulement,
  `matches`/`wins`, pas de défaites). D'où le payload neuf (D-8).
- Sessions : `SessionLabel` de `domain.SquadMatchRow` (`domain/squad.go:77`, format `analysis/sessions.go:310`,
  suffixe volatil « (N) », réconcilié côté web par `reconcileSquadSessionLabels`) ; `SelectEvenings`
  (`analysis/squadimpact/history.go:37-62`) ; population `allSquadRows` (`service/teammates/teammates_service.go:322`,
  `:354`) ; parité matrice/historique `TestBuildSquadImpact_PariteMatriceHistorique`
  (`teammates_squad_impact_test.go:83`).
- Session sélectionnée : `useSquadFilterStore` → `filterContext.sessions.picked_sessions`, exposée par
  `useSquadSessionSelection` (`apps/web/src/features/squad/useSquadSessionSelection.ts:169-176`) ; absente de
  `SquadContext` (`SquadContext.ts:12-21`, construit `SquadLayout.tsx:275-282`) ; garde à relire
  `features/squad/filterBarBoundary.guard.test.ts`.
- Montage : `SquadContributionsPage.tsx` `ImpactMedalsMechanics` (`:197-260`) ; palette des joueurs
  `useSquadPlayerPalette` (garde `playerColors.guard.test.ts`).
- Réutilisables : `SplitBar` non exporté (`features/_shared/EncounterSplitBars.tsx:17-49`) ; `OutcomeBar`
  (`components/ui/outcome-bar.tsx`) ; `BadgeIcon` ; helpers locaux `BADGE_INVERTED`, `computeExtremes`,
  `extremeStyle`, `aggCellState`, `outcomeBg` (`SquadImpactScoreboard.tsx:40-150`) ; lignes dépliables :
  `features/match-view/MatchScoreboard.tsx:148`, `:250`, `:629-690`.
- Gardes : `contract-surface.guard.test.ts` (`UPDATE_CONTRACT_SURFACE=1` + justification),
  `generated-types-fresh`, `response-types`, `functionSize` (cliquet), `lintNoHardcodedColors`,
  `singleCountSource.guard.test.ts` (compte de matchs par session : `squadSessionCounts.ts`),
  `lib/outcome.guard.test.ts` (mapping d'issue), i18n, `keys.guard.test.ts`.
- Lecteurs à supprimer avec les blocs : `SquadImpactScoreboard(.test).tsx`, `impact/*` (dont
  `impactHistory.fixtures.ts`), cas de `SquadContributionsPage.test.tsx:103-106`, `:161-187`,
  `features/timeseries/usages/textesSansPersonne.test.ts:36`, `:124` (lit `IMPACT_HISTORY_TEXT.*.empty`),
  `eveningDate` partagé (`objectif/objectif.logic.ts:463`, garder).
