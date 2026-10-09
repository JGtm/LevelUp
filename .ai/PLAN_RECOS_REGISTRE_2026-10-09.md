# Plan — recommandations issues du tri du registre (2026-10-09)

Origine : tri sur pièces du registre des reports et du backlog (journal du 2026-10-09, commit
`eb0c374ef`). Accord du user le 2026-10-09 (« ok avec ce coût, tu peux y aller ! Parallélise au
besoin »). Contrat : skill `plan-execution`. Superviseur : session principale ; exécutants Opus,
un par lot, chacun dans son worktree et sa branche `feat/recos-<lot>`.

## Décisions (fermes)

- **D-1** Démo : les noms des joueurs se masquent dans l'INTERFACE seulement ; les artefacts de
  rejeu ne sont PAS modifiés (décision user du 2026-10-09).
- **D-2** Démo : le rejeu est servi en mode démo, limité aux matchs figés de la démo (exception au
  garde « rejeu local seulement », qui reste inchangé hors démo).
- **D-3** Démo : au moins un rejeu par mode de jeu présent dans le cache de films ; jeu de matchs
  figé ; recuisson automatique quand le schéma d'artefact monte.
- **D-4** Un jeton marqué « réauthentification requise » (AADSTS70000 et classes mortes) n'est plus
  réessayé à chaque cycle par les chemins secondaires (succès Xbox) ; jamais de re-capture.
- **D-5** Fusion dans `feat/v75` : geste réservé au user (le superviseur prépare, le user dit oui).

## Règles d'environnement (tous les lots)

- Worktree dédié `../LevelUp-wt-recos-<lot>`, branche `feat/recos-<lot>` depuis `feat/v75`.
  Jamais le worktree principal. Jamais `git stash`, jamais `git add -A`.
- `GOCACHE=%LOCALAPPDATA%\go-build-recos-<lot>` et `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-recos-<lot>`.
  Commandes `go` en série dans le lot. Jamais `go build ./...` ; tester les paquets touchés,
  puis `go test ./...` une fois en fin de lot. `-tags=integration` si persist/sync/ops touchés.
- Web : `npm ci` dans `apps/web` du worktree (pas de jonction vers le principal).
- Données : le worktree n'a pas `data/` ; pour lire les bases ou le cache de films, poser
  `LEVELUP_REPO_ROOT` sur le dépôt principal en LECTURE ; toute écriture de données locales
  (cuisson, base) se fait sur copie ou avec l'accord écrit au plan. Un film par processus.
- Les lots tournent en parallèle : l'exécutant NE modifie NI ce plan NI `.ai/thought_log.md`,
  ni le registre ni le backlog (surfaces partagées, conflits garantis). Il rend dans son rapport
  final le statut de chaque case, le texte de l'entrée de journal et ses découvertes ; le
  superviseur les reporte sur `feat/v75`.
- Fin de lot : gate vert, commit(s) préfixé(s) `recos-<lot>`, push de la branche (CI), rapport, puis
  `powershell -File scripts/disk-hygiene.ps1 -Lot recos-<lot> -Apply`.
- Messages au user : sans identifiant technique (voir mémoire « parler clair »).

## Lot A — Rejeu : cartes manquantes et plan Tactique (parallèle à B)

- [x] A1 (catalogue ; cuisson des 24 films [!] reportée après fusion, base tenue par un autre processus) Trois cartes hors catalogue de bornes : « Serenity - Ranked », « Interference »,
  « Vacancy - Ranked » (absentes de `data/titles/halo_infinite/reference/map_quant_bounds.json`
  et `map_objectives.json` ; avertissement `internal/sync/replayartifacts/cuisson.go:169`).
  Ajouter bornes, objets et fond par la chaîne existante (installation de Halo requise), puis
  cuire les seuls matchs de ces cartes présents au cache (un film par processus).
- [~] A2 (déjà servi par héritage variante -> base depuis le 2026-09-03 ; garde-rail ajouté) « Argyle - Ranked » sans fond sous son propre map_id : rattacher le fond d'Argyle.
- [x] A3 (non reproductible depuis la Tactique v2 ; test sur le calage réel) Tactique, plan d'Illusion : canvas de 1 070 x 13 375 px qui ne peint rien (constat du
  2026-09-09) ; reproduire d'abord, corriger si toujours présent.
- Gate : tests des paquets touchés, un rejeu par carte ouvert au navigateur local ;
  gate visuel du user à la fusion.

## Lot B — Corrections visibles (parallèle à A)

- [!] B1 (RETIRÉ le 2026-10-09 par le superviseur : aucun défaut constaté ; passes non publiables = 53 matchs à la révision killsource-2026-09-27, dont 63 lignes de bots sur 13 matchs ; les bots comptent partout) Vue match : `Q20KVPairs` (`platform/duckdb/queries_match.go`) lu sans filtre
  `publishable` (dominance, cumul FDA, victime du fil, antagonistes, « morts vengées »).
- [x] B2 (403 `halo_tokens_missing`, garde 2 rechargements/min en sessionStorage, relecture de `/bootstrap`) Connexion : la sync initiale rend 401 `auth_required` sans tokens Halo
  (`handlers/sync_handler.go:445`) → statut non éjectant ; la coquille (`apps/web/src/routes/__root.tsx`)
  gagne un garde anti-boucle qui survit au rechargement, et un 401 de route secondaire n'éjecte
  pas quand `/bootstrap` dit connecté.
- [x] B3 (`migratePlayerDBs` par titre de profil) Halo 5 : la boucle « migrations player » du boot (`cmd/server/main.go`) n'emploie que le
  titre par défaut → parcourir les titres de chaque profil (PathResolver, sans comparaison de slug).
- [x] B4 (porte `sync/deadtoken`, empreinte du refresh token) Succès Xbox : ne plus réessayer un jeton marqué mort à chaque cycle (D-4) ; une trace par
  changement d'état, pas par cycle.
- Gate : tests unitaires et d'intégration des chemins touchés, tsc + vitest du web touché.

## Lot C — Fiabilité des données (après A et B ; relecture adversariale)

- [ ] C1 `cmd/levelup/cmd_restore_csr.go:101` : DELETE sur `match_skill_rank` (append-only, ADR 0026).
- [ ] C2 Index secondaires `idx_lch_*`, `idx_pme_match_lookup`, `idx_pcs_lookup` : réexamen par
  `EXPLAIN ANALYZE` (recette MSR) ; retrait par migration + soin convergent s'ils ne servent pas ;
  sept fixtures qui créent encore des `idx_msr_*`.
- [ ] C3 Écrivains LUSR en ligne de commande sans alignement des séquences
  (`cmd/recompute_perfnote`, `cmd/h5-lusr-backfill`, `cmd/lusr_v2_canonical…`) → porte `platform/duckdb`.
- [ ] C4 Doublons d'ids dans des PK déclarées et ids NULL des bases joueur : mesure sur copies,
  remède proposé ; réparation seulement si elle est sûre (sinon `[!]` argumenté).
- [ ] C5 Quatre lignes parasites de `match_vehicle_takes` (match_id concaténé) : trouver et
  corriger l'écrivain, puis retirer les lignes (sauvegarde avant).
- [ ] C6 `match_registry.mode_category` possiblement faux : mesurer l'ampleur (copie), décider.
- [ ] C7 Clé héritée `oauth_refresh_token` dans `sync_meta` de 4 bases joueur : purge par migration
  player (aucun code ne la lit, ADR 0023).
- Gate : `go test -tags=integration` des paquets persist/migration/ops touchés, garde-rails ART verts.

## Lot D — Démo (après A et B, parallèle à C ; relecture adversariale)

- [ ] D1 Garde « démo en lecture seule » générale avec liste blanche des POST de lecture ;
  un seul contrat de refus (403 `demo_mode_forbidden`) ; `PATCH /profiles/{p}/titles/{t}/sync`,
  `POST /watcher/auth/start` et les actions admin couvertes.
- [ ] D2 `internal/ops/seed_demo_corpus.go` : `UPDATE kill_positions` sur table append-only →
  anonymiser à la copie ; commentaire faux de `seed_demo.go` sur `kill_positions` ; garde-rail
  d'écriture étendu à `internal/ops`.
- [ ] D3 Rejeux de la démo (D-1 à D-3) : choix d'un match par mode, films et artefacts embarqués
  par `seed-demo`, service en mode démo, recuisson automatique à la montée de schéma, noms masqués
  par l'interface.
- Gate : tests du garde démo (chaque mutation refusée, chaque lecture permise), démo lancée en
  local et un rejeu par mode ouvert.

## Lot E — Hygiène (en dernier)

- [ ] E1 Code mort : `service.FanoutService`, `sync_meta.last_delta_sync`, lecteur de snapshots
  qui exige `weapon_kills`, `TacticalKillEvents.Events` / `KillEvents`, zero-value de
  `ListeBlancheMatchs`, clés `squad.header.*`.
- [ ] E2 `himap.DepotVariantesCarte` pointe sur un dossier absent : échouer bruyamment ou repointer.
- [ ] E3 `MapKeysForMap` : `LIMIT 1` sans tri.
- [ ] E4 Course sur `SyncResult.AddWarning` (goroutines de fetch).
- [ ] E5 `prestige_lazy_service.go` > 500 L ; `outcome = 2` magique ; `cmd/diag_exec` en `sql.Open` direct.
- [ ] E6 Avertissement « pont par morts contredit le lien direct » (~3 400 lignes/jour) → compteur
  et une trace par film.
- [ ] E7 Tests instables : `PalmaresRelationsPage`, assertion de temps de
  `TestRosterDesFilms_AnnuaireContreJointure`, `TestLUSRV2Shadow_RafalesBornees_300Candidats`.
- [ ] E8 Textes et finitions web : « Choisis 1 à 3 coéquipiers… » ; tirets faits main hors
  `EmptyStateNotice` ; abonnements de zoom du rejeu au premier montage ; graduations de « Portée
  par arme » ; pied de page sous le cockpit Tactique.

## Clôture

- [ ] Fusions dans `feat/v75` (accord du user), CI verte.
- [ ] Registre et backlog : lignes traitées retirées.
- [ ] Nettoyage : worktrees et branches locales et distantes `feat/recos-*`, caches dédiés,
  fichiers temporaires.

## Journal

- 2026-10-09 : plan écrit ; jetons de Chocoboflor, Madina97294 et XxDaemonGamerxX importés de la
  prod en local (sauvegarde des anciens sous `data/backups/watcher_tokens_avant_import_prod_2026-10-09/`).

- 2026-10-09 : lot A rendu (`feat/recos-a`, 4 commits, CI verte). Les trois cartes sont des cartes Forge sur canevas connus (preuve level_id) : entrées de bornes et d'objectifs ajoutées. Cuisson des 24 films concernés reportée après fusion. Jetons des trois amis vérifiés en local : succès Xbox synchronisés à 17:38.

- 2026-10-09 : lot B rendu (`feat/recos-b`, 5 commits, CI verte) ; correctif demandé avant fusion : texte « connecte-le » à la 2e personne, « Réessayer » en dur, emoji dans l'étape de sync initiale.

## Découvertes

- (lot B) `SquadRepo.LoadKVPairs` (Q32c) et `sync/engagement.go` lisent les paires sans le prédicat de B1.
- (lot B) `internal/worldenrich/wiring.go` résout un access token sans la porte des jetons morts.
- (lot B) Importer un jeton (SSO, token-import, token-capture) ne lève pas `reauth_required` ni `last_auth_error_*` : bannière affichée jusqu'au prochain refresh réussi.
- (lot B) Import OpenSpartan : 401 `halo_auth_required` pour une session sans compte lié (statut discutable).
- (lot B) Bases Halo 5 : `schema_migrations` mêle des entrées `halo_5` et `halo_infinite` (chemin CLI du 01/09 sous le titre par défaut).
- (lot A) « Argyle - Ranked » sans objectifs au catalogue (identifiant absent de `map_objectives.json`) ; variante à récupérer.
- (lot A) Environ 24 cartes Forge déclarées avec un fond restent absentes du catalogue de bornes (relevé de septembre).
- (lot A) Après fusion, relancer le calcul de la source des kills des 24 matchs (carte non résolue jusqu'ici).
