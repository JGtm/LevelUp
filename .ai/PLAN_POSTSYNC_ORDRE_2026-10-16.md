# Plan — Post-sync : un match d'escouade ne s'affiche plus « Solo » sans LUSR pendant des minutes (2026-10-16)

> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, aucun report,
> statut de chaque item `[x]` / `[~]` réf / `[!]` justifié, aucune case vide à la clôture,
> zéro correctif hors périmètre : les trouvailles vont en section « Découvertes »).
> Branche : `feat/postsync-ordre` depuis `feat/v75`, worktree dédié `../LevelUp-wt-postsync`.
> Exécutant : Sonnet ; relecture adversariale : Opus (lot sync = à risque).
> Reprise de session : lire ce fichier (cases cochées), `git log --oneline -15` de la branche,
> dernière entrée du `.ai/thought_log.md`.

## Objectif et critère de succès

Symptôme (user, 16/10) : un match d'escouade fraîchement synchronisé s'affiche « Solo », sans
LUSR, pendant quelques minutes, puis se complète seul.

Causes établies sur pièces (16/10) :
- C1. `is_with_friends` est écrit FAUX par défaut au premier écrit (`sync/collect.go:216-222`,
  colonne `DEFAULT FALSE`, vue `_latest` en `COALESCE(..., FALSE)`) : « pas encore calculé » et
  « solo » sont indiscernables. La vraie valeur n'est posée qu'à l'étape 3.5 du post-sync.
- C2. Le post-sync est séquentiel (`sync/engine_postsync.go:267-477`) et les étapes film lentes
  (1.57 source du kill : 8 films / 5 min ; 1.58 rejeu : 5 min) passent AVANT LUSR (2),
  CSR (3.1), escouade (3.5) et agrégats (4).
- C3. Le cache de lecture joueur n'est vidé qu'en fin de post-sync (`engine_postsync.go:170`) ;
  `HomeMatchesCache` (45 s) n'est jamais vidé par le cycle automatique V2.

Critère de succès : sur un vrai match d'escouade synchronisé, la tuile affiche « Escouade » dès
la première lecture, le LUSR est présent avant la lecture des films ; aucun résultat final ne
change (citations, domination, préparation du match, source du kill, rejeu).

## Décisions tranchées (pas de décision en cours de route)

- D1. Ordre cible du post-sync (contraintes de dépendance vérifiées le 16/10) :
  `-2 lignes d'enrichissement` → `scoring` → `1.54 événements` → `1.56 PSA` → `3.5 escouade` →
  `1.7 domination` → `2 LUSR` → `3.1 CSR` → `4 agrégats` → **vidage des caches** →
  `1.57 source du kill` → `1.58 rejeu` → `1.6 citations` → `4.5 médias` → `5 succès` →
  `6 préparation` → (vidage final existant, `defer`).
  Interdits : 1.54 après la domination ou les citations (domination figée à 0) ; citations avant
  1.57 (citations par arme perdues, calcul unique) ; préparation avant 1.57 ou les citations ;
  PSA après les citations ou l'escouade ; 1.58 avant 1.57.
- D2. `is_with_friends` est calculé dès le premier écrit, avec le MÊME prédicat que le recalcul
  (même équipe, xuid dans les amis, xuid différent du joueur). Le recalcul de l'étape 3.5 reste
  en place, il reste l'autorité convergente (ami ajouté/retiré). Liste d'amis inconnue ou vide →
  champ non posé (comportement actuel).
- D3. Aucun changement de schéma, de vue `_latest`, ni de sémantique de colonne. L'écriture passe
  par le champ pointeur existant `EnrichmentRow.IsWithFriends` (chemin INSERT-only documenté,
  `persist/player_persister.go:218`).
- D4. Hors périmètre : indicateur « analyse en cours » côté web (film, rejeu), CSR (aucun défaut
  constaté, voir « Constat CSR »), Halo 5 (`livesync` n'a pas d'étape film).

## Étape 0 — Préparation (rapide)

- [x] 0.1 `git worktree list` ; worktree `../LevelUp-wt-postsync`, branche `feat/postsync-ordre`
  depuis `feat/v75` ; noter le commit de base.
- [x] 0.2 Base verte : `go test ./internal/sync/... ./internal/persist/...` et
  `go test -tags=integration ./internal/sync/... ./internal/persist/...` (`-p 1`) ; noter les
  rouges préexistants éventuels (ils restent à réparer, règle CI).
- [x] 0.3 Inventaire écrit dans ce fichier (section « Inventaire ») : colonnes de
  `mv_player_matches` (`sync/aggregates.go`) et étape productrice de chacune. Si une colonne
  dépend de 1.57, 1.58 ou des citations, un second rafraîchissement des agrégats est ajouté après
  1.6 (item 1.4) ; sinon 1.4 est statué `[~]`.

Gate 0 : tests de base exécutés, inventaire rempli.

## Étape 1 — Ordre du post-sync (moyen)

- [ ] 1.1 Réordonner `runPostSyncPipeline` selon D1 (déplacement de blocs, aucun changement de
  logique d'étape). Construire `postSyncFilmSteps` une seule fois et le réutiliser.
- [ ] 1.2 Vidage du cache de lecture joueur après les agrégats
  (`duckdbpkg.InvalidatePlayerReadCaches`), le `defer` final reste (ratchet
  `archlint/player_read_cache_invalidation_test.go`).
- [ ] 1.3 Vidage de `HomeMatchesCache` au même point, pour V1 ET V2, via un rappel du moteur
  (modèle `mediaHook` / `prestigeHook` ; `sync` n'importe pas `service`). Brancher le rappel dans
  le câblage du serveur pour les deux chemins.
- [~] 1.4 Second rafraîchissement des agrégats après 1.6 — seulement si l'inventaire 0.3 le
  justifie.
- [ ] 1.5 `sync/postsync_clock.go` : la liste des étapes affichées suit l'ordre D1 et contient
  `kill_source` et `replay_artifacts` (aujourd'hui absentes, `weapon_kills` périmé) — nécessaire
  pour lire l'ordre réel au tableau de bord en étape 4.
- [ ] 1.6 Test qui épingle l'ordre : `StepTimings` d'un post-sync contient `friends`,
  `skill_rating`, `aggregates` AVANT `kill_source`, et `citations` APRÈS `replay_artifacts`,
  `snapshot_readiness` en dernier. Mutation : remettre l'ancien ordre → rouge.
- [ ] 1.7 Test d'intégration : match inséré par un coéquipier (`events_loaded=false`) → après un
  post-sync, domination NON figée à 0 et citations avec événements (preuve que 1.54 reste avant).
- [ ] 1.8 Seuils : `runPostSyncPipeline` ≤ 80 L ou exemption existante inchangée ; aucun nouveau
  fichier à la racine de `internal/sync` (ratchet `sync_root_freeze_test.go`).

Gate 1 : `go test ./internal/sync/...`, `go test -tags=integration -p 1 ./internal/sync/...
./internal/persist/...`, `go test ./internal/archlint/...`, `golangci-lint` 0 nouveau.

## Étape 2 — « Escouade » dès le premier écrit (moyen)

- [ ] 2.1 Prédicat pur unique (dans un fichier existant de `internal/sync`) :
  `withFriends(participants, playerXUID, friendXUIDs) bool` = même équipe, xuid ami, xuid ≠
  joueur, bots exclus comme dans le SQL.
- [ ] 2.2 `fetchedMatch` porte les xuids amis résolus ; `buildBatchFromFetchedMatchCtx` pose
  `EnrichmentRow.IsWithFriends` quand la liste est connue (signature ≤ 5 paramètres).
- [ ] 2.3 Résolution des amis UNE fois par passe : V1 (`engine_batch_path.go`, via
  `friendsLoader` + `LookupFriendXUIDs`) et V2 (`engine_v2bridge.go`, même chose côté cycle).
  Échec de résolution → log WARN, champ non posé, pas d'échec du sync.
- [ ] 2.4 Test d'équivalence : sur un même jeu de participants (même équipe, équipe adverse, joueur
  présent dans sa propre liste d'amis, bot, liste vide), le prédicat pur et
  `loadMatchesWithFriends` (SQL) rendent le même verdict.
- [ ] 2.5 Test d'intégration persist : lot d'un match d'escouade → `player_match_enrichment_latest`
  porte `is_with_friends = TRUE` AVANT tout post-sync ; le recalcul 3.5 qui suit promeut 0 et
  démote 0.
- [ ] 2.6 Garde-rail ART : `go test ./internal/sync/ -run NoArt` vert sans allowlist ajoutée.

Gate 2 : mêmes commandes que le gate 1 + `go test ./internal/persist/...`.

## Étape 2bis — Simulation de bout en bout sur le faux serveur Halo (moyen)

Harnais existant (plan de stratégie des tests de sync, 24/05) : `internal/sync/halotest/fake_server.go`
sert le match réel figé `internal/sync/testdata/jgtm_full_match/` (stats, skill, film, historique) ;
`internal/sync/engine_e2e_test.go` l exerce déjà.

- [ ] 2b.1 Test de bout en bout (dans un fichier de test existant ou un nouveau `*_test.go` sous
  `internal/sync/`) : faux serveur + base vierge + liste d amis contenant un coéquipier réel du
  match figé → après le PREMIER écrit (avant le post-sync), la ligne d enrichissement lue par
  `player_match_enrichment_latest` vaut `is_with_friends = TRUE` ; après le post-sync complet :
  LUSR présent, domination non NULL, citations présentes, ordre des `StepTimings` conforme à D1.
- [ ] 2b.2 Même test sans liste d amis → `is_with_friends = FALSE`, post-sync identique par
  ailleurs.
- [ ] 2b.3 Comparaison avant/après : le même test exécuté sur le commit de base (ancien ordre) et
  sur la branche donne les MÊMES valeurs finales (citations, domination, LUSR, préparation,
  escouade) ; seule la chronologie change. Écart = arrêt et analyse.

Gate 2bis : 2b.1-2b.3 cochés, tableau avant/après dans le journal.

## Étape 3 — Vérification complète (lourd)

- [ ] 3.1 (après le gate 2bis) `cd apps/go-api && go test ./...` et `go test -tags=integration ./...` verts.
- [ ] 3.2 `make go-api-lint` ; `make gate-push`.
- [ ] 3.3 Relecture adversariale (skill `adversarial-review`, Opus, contexte frais) du diff
  complet ; chaque constat P1/P2 corrigé ou statué par écrit.
- [ ] 3.4 Push de la branche, CI verte (tous jobs).

Gate 3 : 3.1-3.4 cochés, sorties consignées dans le journal.

## Étape 4 — Validation sur données réelles (moyen, nécessite un vrai match)

- [ ] 4.1 Mesure AVANT sur l'historique local : pour les matchs déjà présents, délai entre
  `match_registry.first_sync_at` et le `written_at` de la ligne d'enrichissement « friends » et de
  la ligne LUSR (requête sur une COPIE de la base, `cmd/diag_q`).
- [ ] 4.2 Serveur local relancé sur le code de la branche (après accord des sessions pairs,
  `ListAgents`), un cycle de synchro avec au moins un nouveau match d'escouade. S'il n'y en a
  aucun, demander au user d'en jouer un (blocage légitime).
- [ ] 4.3 Sur ce match : tuile « Escouade » à la première lecture (navigateur, capture), LUSR
  présent avant la fin de 1.57, ordre des étapes lu au tableau de bord admin, citations et
  domination présentes, préparation atteinte, source du kill traitée au cycle normal.
- [ ] 4.4 Mesure APRÈS (même requête qu'en 4.1) et comparaison écrite dans le journal.
- [ ] 4.5 Recalcul de l'escouade relancé à blanc sur les 4 joueurs : 0 promotion, 0 démotion.

Gate 4 : 4.1-4.5 cochés, capture et chiffres dans le journal.

## Étape 5 — Clôture (rapide)

- [ ] 5.1 Documents : ordre du post-sync dans `docs/SYNC_GUIDE.md` ET `docs/FR/` (bilingue) s'il
  y est décrit ; commentaires d'en-tête de `engine_postsync.go` alignés.
- [ ] 5.2 `.ai/thought_log.md`, `.ai/REGISTRE_REPORTS.md` (découvertes non traitées).
- [ ] 5.3 Skill `delivery-checklist`.
- [ ] 5.4 Fusion dans `feat/v75` : SUR ACCORD du user.

## Constat CSR (16/10, sur pièces, aucune action)

- Sur 691 matchs en matchmaking synchronisés moins de 5 min après leur fin (base locale + base de
  prod copiée), 688 ont leurs données de classement ; les 3 autres sont des matchs avortés de
  moins de 2 min. Le retard de publication du classement par Halo n'apparaît pas dans les données.
- Les 58 matchs « Ranked » locaux sans classement durent tous 32 à 76 s : matchs avortés, Halo ne
  publie pas de classement pour eux. Pas un défaut.
- La correction de la semaine passée (le classement décidé par la réponse de Halo) est dans
  `feat/v75` mais PAS dans `main` : elle arrive en prod avec la version 7.5. Dernière écriture de
  CSR par match en prod : 26/05 ; aucun match classé de plus de 2 min n'y a été synchronisé depuis
  le 17/09, donc rien à observer là-bas d'ici le déploiement.

## Inventaire (étape 0.3)

`mv_player_matches` (`sync/aggregates.go`, `playerMaterializedViews`) = 5 colonnes lues dans
`player_match_enrichment_latest` (filtre `performance_score IS NOT NULL`) :

| Colonne | Étape productrice | Dépend de 1.57 / 1.58 / citations ? |
|---|---|---|
| `match_id` | ligne d'enrichissement (`-2` ou collecte) | non |
| `performance_score` | `scoring` (`batchComputePerformanceScores`, lit `match_participants` seul) | non |
| `session_id`, `session_label` | `scoring` (`recalculateSessionsInline`) | non |
| `is_with_friends` | `3.5 friends` (et, dès l'étape 2 de ce plan, le premier écrit) | non |

Conclusion : aucune colonne ne dépend de 1.57, 1.58 ni des citations ; un second rafraîchissement
des agrégats après 1.6 n'apporterait rien. Item 1.4 : `[~]` couvert par cet inventaire.

## Découvertes

- (16/10, exécutant, étape 0 -> 1) CONTREDIT D1 : le LUSR v2 lit la source du kill. La pénalité de
  départ du LUSR v2 (`sync/skill/skill_v2_quit_penalty.go`, `loadQuitTimeline`, appelée par
  `skill_v2_shadow.go:270` quand le match compte un joueur parti) lit `match_kill_events_latest`,
  table dont 1.57 est le producteur au fil de l'eau. Sans ces lignes : `available=false`, repli sur
  l'issue finale (`quitDeltaForTeam`), donc un delta de départ différent (« lié » / « sans lien »
  décidé par la situation au moment du départ, vs par l'issue). Le filigrane du groupe avance
  (`lusrWatermarkCovers`) : le match n'est pas recalculé au cycle suivant. Ordre actuel (1.57 avant
  LUSR) : la chronologie est lue ; D1 (LUSR avant 1.57) la perdrait pour tout nouveau match avec
  un parti, ce qui contredit « aucun résultat final ne change ». Le plan disait les dépendances
  « vérifiées le 16/10 » : celle-ci a été manquée. Arrêt de l'exécutant à l'entrée de l'étape 1
  (consigne : décision tranchée contredite par un fait sur pièces).
- (16/10, superviseur) Le worktree `feat/citations-film` (plan `.ai/PLAN_CITATIONS_FILM_2026-10-16.md`,
  non implémenté à ce jour) touche aussi l étape des citations. Ce plan place les citations APRÈS
  les films, ce qui va dans son sens (son étape E4 suppose déjà les citations après le Deriver).
  À signaler à la fusion, pas à traiter ici.

- (16/10, superviseur) Le worktree `feat/citations-film` (plan `.ai/PLAN_CITATIONS_FILM_2026-10-16.md`,
  non implémenté à ce jour) touche aussi l étape des citations. Ce plan place les citations APRÈS
  les films, ce qui va dans son sens (son étape E4 suppose déjà les citations après le Deriver).
  À signaler à la fusion, pas à traiter ici.

## Journal d'exécution

- 16/10 Étape 0 CLOSE. 0.1 : worktree `LevelUp-wt-postsync`, branche `feat/postsync-ordre`, base `27947da10`.
  0.2 : `go test ./internal/sync/... ./internal/persist/...` vert (sync 138 s, persist 4 s) ;
  `go test -tags=integration -p 1` des mêmes paquets vert (sync 223 s, persist 59 s) ; aucun rouge
  préexistant. 0.3 : inventaire rempli, 1.4 statué `[~]` (aucune colonne de `mv_player_matches` ne dépend
  de 1.57/1.58/citations).
- 16/10 ARRÊT avant l'étape 1 : D1 contredit par un fait sur pièces (voir « Découvertes », premier
  point : le LUSR v2 lit `match_kill_events_latest`, produit par 1.57). Rien du code n'a été modifié.
  Options à trancher par le superviseur : (a) LUSR APRÈS 1.57 mais avant 1.58 (perd une partie du
  gain : 1.57 = 8 films / 5 min avant le LUSR) ; (b) LUSR avant 1.57, et le shadow saute les matchs
  avec un joueur parti tant qu'aucune ligne `match_kill_events_latest` n'existe, avec une 2e passe LUSR
  après 1.57 (le filigrane ne doit pas avancer sur le match sauté) ; (c) accepter le repli sur l'issue
  finale pour ces matchs (change des résultats : exclu par le critère de succès).
