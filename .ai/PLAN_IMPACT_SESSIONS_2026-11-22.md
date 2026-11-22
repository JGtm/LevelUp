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

- [x] Types `domain.SquadImpactSessions` (joueurs ; barème ; sessions : libellé, début, matchs
  {match_id, issue}, par joueur {matchs joués, points, comptes par rôle, rôles par match} ; totaux) ;
  champ sur `TeammatesPageResponse` ; `make openapi-gen` + `make generate-types`. Anciens champs encore
  présents à ce stade.
- Gate : `go build ./...`, `go test ./contracttest/...`, `make generate-types` sans diff résiduel.

### Lot G.2 — calcul (Sonnet) — moyen — après G.1, en parallèle de W

- [x] Fonction pure `squadimpact.BuildSessions` (à partir des rôles par match, des sessions de
  `SelectEvenings`, de l'équipe alliée) ; tests purs.
- [x] Service : remplissage du payload, suppression de `assemblerMatrice`, `BuildHistory` et des deux
  anciens champs (types, schémas, tests) ; journal D-9.
- [x] Témoin avant/après : sur les fixtures de `teammates_squad_impact_test.go`, les comptes par rôle et
  les points par joueur et par session du nouveau payload égalent ceux de l'ancien historique, et les
  totaux de la population ceux de l'ancienne matrice (test écrit AVANT la suppression, gardé ensuite sur
  des valeurs figées).
- Gate : `go test ./...`, `go test -tags=integration -p 1 ./internal/service/... ./internal/platform/duckdb/...`,
  `make go-api-lint`.

### Lot W — tableau web (Sonnet) — lourd — après G.1

- [x] Logique pure (`impactSessions.logic.ts`) : lignes, net par match, état ouvert initial (D-6),
  surlignage (extrait des helpers de `SquadImpactScoreboard` vers un module partagé de la feature).
- [x] Composant `SquadImpactSessionsTable` (TanStack Table, patron des lignes dépliables de
  `MatchScoreboard`), barre composite V/D (exporter un `WinLossSplitBar` depuis
  `features/_shared/EncounterSplitBars.tsx`), textes fr/en, infobulle de titre.
- [x] Montage dans `SquadContributionsPage` à la place des deux blocs ; transmission de la session
  sélectionnée selon `filterBarBoundary.guard.test.ts`.
- [x] Suppression de `SquadImpactScoreboard*`, `impact/*` (historique), fixtures, textes devenus morts ;
  D-10 ; cliquets (`functionSize`, contrat) abaissés ou mis à jour avec justification.
- Gate : `npx vitest run src/features/squad src/lib src/components --maxWorkers=2`, `npm run typecheck`
  (après purge de `node_modules/.tmp`), `npm run lint`, `node tools/lint-cross-feature-imports.mjs`.

### Clôture — pilote

- [x] Z.1 `npx vitest run` complet, `go test -tags=integration -p 1 ./...`, `make gate-push` (avec
  `LEVELUP_REPO_ROOT` et `LEVELUP_IDENTITES_ORIGINALES` du dépôt principal).
- [x] Z.2 Revue adversariale (Fable, une ronde), P0/P1 corrigés, P2 au backlog.
- [x] Z.3 Lignes de vérification dans l'artefact `verifs` ; `thought_log` ; fusion dans `release/v8.3.0` ;
  CI lue.

## Découvertes

- `squadimpact.MatchIDs` (evenings.go) n'avait plus aucun appelant après la suppression de l'historique : supprimé par le pilote (règle « 0 code mort », la fonction servait le préchargement des soirées retiré avec l'historique).

## Reprise de session

Lire ce fichier puis `git log --oneline -10` dans le worktree.

## Journal

- 2026-11-22 : plan ouvert ; décisions D-1 à D-10 tirées de la maquette validée par l'utilisateur.
- 2026-11-22 : G.1 close. Go (`internal/domain/squad_impact_sessions.go`) : `SquadImpactSessions` {`players`, `scale`, `role_order`, `sessions`, `totals`} ; `SquadImpactSessionsRoleWeight` {`role`, `points`} ; `SquadImpactSessionsSession` {`session_label`, `start_time`, `matches`, `players`} ; `SquadImpactSessionsMatch` {`match_id`, `outcome`} ; `SquadImpactSessionsPlayer` {`player`, `matches_played`, `points`, `counts`, `per_match`} ; `SquadImpactSessionsRoleCount` {`role`, `count`} ; `SquadImpactSessionsPerMatch` {`match_id`, `roles`} ; `SquadImpactSessionsTotal` {`player`, `matches_played`, `points`, `counts`}. Champ `squad_impact_sessions` (omitempty, pointeur) sur `TeammatesPageResponse`, nil jusqu'à G.2. Barème servi : lecteur = sens du rôle (points < 0) pour le surlignage, qui remplace la table `BADGE_INVERTED` du web (une seule définition, D-8) ; le nom et la description des infobulles restent ceux du texte web. `counts` porte une entrée par rôle de `role_order` (0 compris), pour les sessions comme pour `totals` ; `per_match` ne porte que les matchs où le joueur a un rôle. Contrat régénéré (`openapi.yaml`, `generated.ts`, ré-exports `types.ts`, `contract-surface.snapshot.json`).
- 2026-11-22 : G.2, fonction pure close. `squadimpact.BuildSessions` (`analysis/squadimpact/sessions.go`, tests `sessions_test.go`) : entrées `SessionsInput` {sessions de `SelectEvenings`, joueurs, rôles par match, joueurs présents dans l'équipe alliée par match} ; `EveningMatch` gagne `Outcome` (issue du joueur principal). `matches_played` = matchs où le joueur est dans l'équipe alliée OU y a un rôle (un rôle vaut présence : garde l'invariant « 0 match joué => 0 point »). Matchs de chaque session triés chronologiquement par la fonction (l'ordre d'entrée de `SelectEvenings` ne l'est pas). Nil comme `BuildHistory` (aucune session, aucun joueur, aucun rôle). Ordre d'exécution : le témoin d'équivalence exige le nouveau payload, il est donc écrit après la fonction et le câblage du service, avant toute suppression.
- 2026-11-22 : G.2, témoin d'équivalence écrit avant suppression (`service/teammates/teammates_squad_impact_temoin_test.go`, vert) : sur les fixtures S1 / S2 plus un match de la population sans libellé, par session et par joueur les comptes par rôle, les points, le début et les victoires du nouveau bloc égalent l'ancien historique ; les totaux égalent l'ancienne matrice restreinte aux matchs à libellé ; l'ancienne matrice sur toute la population en diffère (D-9 montré) ; le journal D-9 (`teammates_squad_impact_sessions_unlabelled_matches`, une ligne) est testé. Aides du service : `teammates_squad_impact_sessions.go` (`sessionsDImpact`, `joueursDansLEquipeAlliee`, `journaliserMatchsSansSession`).
- 2026-11-22 : G.2, service rempli et anciens blocs supprimes. `buildSquadImpact` rend `*domain.SquadImpactSessions` (champ `squad_impact_sessions` rempli ; log D-9 `teammates_squad_impact_sessions_unlabelled_matches` une fois par requete ; log `teammates_squad_impact_sessions_absent` avec raison `no_session` / `no_role_attributed`). Supprimes : `assemblerMatrice`, `matrixPlayers`, `squadimpact.BuildHistory` / `HistoryInput` / `buildEvening` / `playerEvening`, champ `Won` de `EveningMatch` (remplace par `Outcome`), types `SquadImpactMatrix`, `SquadImpactCell`, `SquadImpactMatchHeader`, `SquadImpactPlayerSummary`, `SquadImpactBadgeCount`, `SquadImpactHistory*` (fichier `domain/squad_impact_history.go`), champs `impact_matrix` et `squad_impact_history` de `TeammatesPageResponse`, leurs tests (parite matrice / historique, historique absent). Constante `SquadImpactHistoryMaxEvenings` renommee `SquadImpactSessionsMax` (domain/squad_impact_sessions.go). `history.go` renomme `evenings.go`. Temoin reecrit sur valeurs figees (Main 9,5 points par session, 19 au total ; Ally -4 et -8 ; sans le match sans libelle : l'ancienne matrice sur toute la population donnait 23,5 et -9). Contrat web (openapi, generated.ts, types.ts, snapshot) regenere dans un commit separe, le dernier ; le typecheck web echoue a ce stade (composants lisant les anciens champs) : attendu, traite par le lot W.
- 2026-11-22 : G.2, contrat regenere (commit separe, le dernier de la piste). `make openapi-gen`, `make generate-types`, retrait des re-exports morts de `types.ts`, `contract-surface.snapshot.json` mis a jour (`UPDATE_CONTRACT_SURFACE=1`). Disparitions justifiees (D-8, regle 0 code mort) : schemas `SquadImpactBadgeCount`, `SquadImpactCell`, `SquadImpactEvening`, `SquadImpactEveningPlayer`, `SquadImpactHistory`, `SquadImpactMatchHeader`, `SquadImpactMatrix`, `SquadImpactPlayerSummary`, `SquadImpactRoleCount`, `SquadImpactRoleWeight`, remplaces par `SquadImpactSessions*` ; champs `impact_matrix` et `squad_impact_history` de la reponse de `/pages/teammates` supprimes. Le typecheck web echoue tant que le lot W n'a pas retire les composants qui lisent les anciens champs (attendu).
- 2026-11-22 : revue adversariale, deux P1 corriges. (1) La section « Impact des coequipiers » (titre compris) est de nouveau derriere la capacite `match.events.timeline` (`useDataCapability` dans `SquadContributionsPage.tsx`) : absente sans la capacite, presente avec son etat vide sinon ; commentaires Go (`domain/teammates.go`, `buildSquadImpact`) alignes ; test ajoute. (2) Corps de `SquadImpactSessionsTable` extrait dans `impact-sessions/ImpactSessionRow.tsx` (rendu inchange), plafond `PLAFOND_LONGUEUR` 244 -> 243.

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
- 2026-11-22 : lot W fusionné (worktree `LevelUp-wt-impact-sessions-w`) : tableau `features/squad/impact-sessions/`, `WinLossSplitBar`, anciens blocs et jetons `impact-gain/loss` supprimés (avec `paletteForTheme`). Écart validé par la maquette (D-4 précisé) : à égalité, plusieurs couronnes ou bananes ; aucune marque si tous les scores sont égaux.
- 2026-11-22 : clôture. Z.1 : suite web complète verte hors le cliquet `functionSize` abaissé pendant la suite par la correction P1-2 (relancé vert) ; typecheck vert ; intégration Go `-p 1` de G.2 sur service, platform/duckdb et api verte (la passe `./...` complète, sans objet pour ce diff qui ne touche ni persist ni sync ni migration, a été arrêtée) ; `make gate-push` vert. Z.2 : revue Fable, aucun P0, deux P1 corrigés (porte de capacité `match.events.timeline`, sous-composant `ImpactSessionRow`), P2 au backlog. Z.3 : ligne `is-tableau-sessions` dans l'artefact des vérifications, journal, fusion dans `release/v8.3.0`.
