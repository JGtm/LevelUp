# Plan — backlog du 2026-10-15 (items 4, 5, 6 de `.ai/BACKLOG.md`)

Origine : demande du user du 2026-10-15 (« ok pour 4, 5 et 6 »). Items 1 et 2 (courbe de score
Lattice, carte Harvest) : en dernier, hors de ce plan (séquence de pré-release). Item 3 (restes
de la démo) : conversation dédiée, avec l'audit de sécurité.

Contrat d'exécution : skill `plan-execution` (fait foi). Branche `feat/backlog-2026-10-15`,
worktree `LevelUp-wt-backlog-1015`, partie de `origin/feat/v75` `a9a9ac78a`. Superviseur : session
principale. Exécutants : Sonnet. Fusion dans `feat/v75` : geste du user. Push `main` : jamais.

Ordre strict : E1 → E2 → E3 → E4. Une étape est close quand son gate est vert, chaque item
statué (`[x]` / `[~]` réf / `[!]` justifié), ce fichier à jour, une entrée au thought_log, un
commit préfixé par l'étape. Découvertes hors périmètre : section 3, non traitées.

Environnement : `GOCACHE=%LOCALAPPDATA%\go-build-backlog1015`,
`GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-backlog1015`. Jamais `go build ./...`. Un seul
`go` à la fois. Clôture : `scripts/disk-hygiene.ps1 -Lot backlog1015 -Apply`.

## 1. Décisions (fermes)

- **D-1** Le garde `archlint/campaign_exclusion_guard_test.go` est ÉTENDU, pas doublé : critère
  élargi aux tables `killer_victim_pairs`, `match_kill_events(_latest)` et `medals_earned`,
  racines élargies à `internal/sync`, `internal/ops`, `cmd`.
- **D-2** Nouvelle catégorie de dispense « outil opérateur » réservée à `cmd/diag_*`,
  `cmd/audit_*` et aux backfills qui réécrivent le match lu lui-même (la Campagne n'y change
  aucun agrégat). Tout lecteur de `internal/sync` ou `internal/ops` qui produit une donnée
  servie à l'UI (dérivé, agrégat, snapshot) est CORRIGÉ, pas dispensé, sauf catégorie existante
  prouvée (mono-match, ensemble fourni par l'appelant, prédicat classé, sémantique).
- **D-3** Exclusion = helpers existants (`analysis.SQLExcludeCampaign*`, `excludeCampaign*` de
  `platform/duckdb/campaign_exclusion.go`). Aucun nouveau mécanisme, aucune comparaison de slug.
- **D-4** E1 est un diagnostic : si la cause est une donnée du poste (sync partielle locale),
  on clôt avec le constat ; si c'est un défaut de code de taille S, il est corrigé en E1 ;
  au-delà, STOP et retour au user.
- **D-5** Lecture prod pour E1 : requête en LECTURE seule sur le VPS, le superviseur prévient le
  user avant (pas d'écriture, pas de redémarrage).
- **D-6** Item 6 : `migration.SplitSQL` (limite documentée) et la CLI de migration d'une player
  DB désignée sont RETIRÉS du backlog sans traitement (aucun besoin, proposé le 15/10, validé).
- **D-7** `title_slug` de `GET /squads` : supprimé du contrat (OpenAPI + `make generate-types`
  + appelants web), pas de rétrocompatibilité (paramètre sans effet).

## 2. Étapes

### E1 — Matchs sans tableau des scores (item 5, diagnostic) — S

Pièces : `.ai/V7.5/PLAN_BACKLOG_2026-09-26.md` DA-5 ; témoins `000d5950`, `28c9b538`,
`4f77afc1`, `94a28b8b` (vue match `scoreboard_empty`).

- [ ] E1.1 Sur une COPIE de `shared_matches_v2.duckdb` du poste : pour chaque témoin, présence
  dans `match_registry`, nombre de lignes de `match_participants` (vue `_latest` si append-only),
  date d'insertion ; résultat en tableau au journal.
- [ ] E1.2 Recenser localement tous les matchs du registre sans participant (compte, période,
  titre, joueurs suivis concernés).
- [ ] E1.3 Lire le chemin d'écriture des participants (sync + rattrapage) et nommer la cause
  (sync interrompue, rattrapage manquant, filtre) avec `fichier:ligne`.
- [ ] E1.4 Même compte qu'E1.2 en prod (D-5).
- [ ] E1.5 Conclusion selon D-4 : constat clos, ou correctif S avec test rouge puis vert, ou STOP.
- [ ] E1.6 Mettre à jour l'entrée du backlog (retirée si close, réécrite sinon). La sous-partie
  « ERROR au boot de la démo sur `halo_5/warehouse/metadata.duckdb` » passe dans l'audit de
  sécurité/démo (`[~]`).

Gate : requêtes rejouables collées au journal ; si correctif, `go test` des paquets touchés.

### E2 — Lecteurs sans exclusion de la Campagne (item 4) — M

Pièces : `archlint/campaign_exclusion_guard_test.go` + `_dispenses_test.go` ;
`platform/duckdb/kill_events_source.go:71-85` (`QKillsBetweenPlayers`, `…Borne`) ;
`platform/duckdb/queries_citations.go:38` (`Q36aMedalTotals`, lu par `citations_repo.go:113` et
`medals_repo.go:92`) ; `api/wire/post_sync_deltas_snapshot.go:322` (`loadEarnedMedalIDs`) ;
liste des 56 lecteurs au journal B4 de `.ai/V7.5/PLAN_BACKLOG_2026-09-26.md` (l. 2351-2380).

- [ ] E2.1 Test de comportement rouge puis vert (tag `integration`, joué `halo_5` ET
  `halo_infinite`, comme `campaign_exclusion_behavior_test.go`) puis correction de
  `QKillsBetweenPlayers` et `QKillsBetweenPlayersBorne` (duel A/B sans frag de Campagne).
- [ ] E2.2 Idem pour `Q36aMedalTotals` (totaux de médailles, citations).
- [ ] E2.3 Idem pour `loadEarnedMedalIDs` (médaille de Campagne non « déjà obtenue »).
- [ ] E2.4 Étendre le garde selon D-1 ; mutation vérifiée (retirer une exclusion d'E2.1 → rouge).
- [ ] E2.5 Statuer les 17 lecteurs de `internal/sync` (corriger ou dispense D-2, avec raison
  et `fichier:ligne`).
- [ ] E2.6 Statuer les 10 lecteurs de `internal/ops`.
- [ ] E2.7 Statuer les 29 lecteurs de `cmd`.
- [ ] E2.8 Tout autre entrant révélé par l'extension (nouvelles tables) statué de même ; un
  entrant hors de toute catégorie = STOP, retour au superviseur.
- [ ] E2.9 Ratchet `no_slug_comparison_test.go` vert ; aucune comparaison de slug ajoutée.

Gate : `go test ./internal/archlint/ ./internal/platform/duckdb/... ./internal/api/wire/` + les
paquets corrigés, puis `go test -tags=integration -p 1` sur les mêmes paquets.

### E3 — Petites dettes Go (item 6) — S

- [ ] E3.1 `campaign_repo.go` `LoadAxisSamples` (l. ~278-286) et la requête sur `match_registry`
  (l. ~364-373) : temps canonique (fragment partagé, règle n°8) pour filtre et tri ; test.
- [ ] E3.2 `engagement_score_repo_queries.go` `ListRecentPvPMatchIDs` (l. ~309-312) : temps
  canonique ; erreur de `Scan` (l. ~85 du bloc, `if err == nil`) et `continue` sur erreur sans
  log (l. ~34-38) → remontée ou `slog.WarnContext` + compteur ; test.
- [ ] E3.3 `GET /squads` : retrait de `title_slug` (D-7), `make generate-types`,
  `make check-types`, appelants web mis à jour.
- [ ] E3.4 D-6 appliqué dans `.ai/BACKLOG.md`.

Gate : `go test` des paquets touchés, `go test ./internal/api/...` (contrat OpenAPI golden),
`make check-types`, `make test-web` si un fichier web change.

### E4 — Clôture

- [ ] E4.1 `make go-api-lint` (ratchet), `make check-types`.
- [ ] E4.2 Revue adversariale du diff (skill `adversarial-review`, contexte frais).
- [ ] E4.3 `.ai/BACKLOG.md` : items 4, 5, 6 retirés ou réécrits ; `.ai/REGISTRE_REPORTS.md`
  alimenté ; thought_log.
- [ ] E4.4 Push de la branche, CI verte (tout rouge se répare).
- [ ] E4.5 Fusion dans `feat/v75` : demandée au user.

## 3. Découvertes

(aucune)

## 4. Journal

- 2026-10-15 : plan écrit, worktree créé sur `a9a9ac78a`.
