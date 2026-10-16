# Plan — indicateur « analyse du film en cours » (tuile de match + vue match)

Origine : demande du user du 2026-10-16 (maquette validée le 16/10). Quand un match est
synchronisé mais que l'analyse de son film n'est pas finie, l'interface le dit au lieu
d'afficher des trous.

Contrat d'exécution : skill `plan-execution` (fait foi). Branche `feat/indicateur-analyse`,
worktree `LevelUp-wt-indicateur`, partie de `origin/feat/v75` `aae0a6c6f`. Fusion dans
`feat/v75` : geste du user. Push `feat/v75` / `main` : jamais. Pas de relecture adversariale,
pas d'attente de CI (demande du user).

Environnement Go : `GOCACHE=C:\t\go-build-indicateur`, `GOLANGCI_LINT_CACHE=C:\t\golangci-indicateur`,
gcc ucrt64. Jamais `go build ./...`. Un seul `go` à la fois. Intégration : `-tags=integration -p 1`.

## 1. Décisions (fermes, tranchées sur pièces le 2026-10-16)

- **D-1 Surfaces servies.** Le grep des usages web de `MatchCard` ne trouve qu'UN consommateur :
  `features/home/HomePage.tsx` (deux listes, `recent_matches` et `favorite_matches`, toutes deux
  des `RecentMatchItem`). Les autres listes de matchs (Explorer, escouade, historique) sont des
  tableaux, pas des tuiles : hors périmètre. Le champ est donc posé sur `RecentMatchItem`
  (`analysis_pending`) et sur `MatchViewHeader` (`analysis_pending`), tous deux `omitempty`
  comme `has_replay` (absent = faux, aucune fixture existante à réécrire).
- **D-2 Capability (a).** La porte du titre est `film.kill_source` (`games.CapFilmKillSource`) :
  c'est la clé qui gouverne le décodage du film de ce titre et la condition (c1). Titre sans la
  clé : le résolveur n'est PAS câblé (même patron que `withFilmArtifactRepos`), le champ est
  toujours faux. La condition (c3) exige en plus `film.replay_artifact` (`CapFilmReplayArtifact`) :
  un titre qui ne produit pas d'artefact de rejeu ne peut pas être « en attente » de son rejeu.
  Aucune comparaison de slug. `citations.engine` est NON utilisable : `not_exposed` pour Infinite
  alors que les citations tournent en prod par le chemin legacy (capabilities.toml, en-tête).
- **D-3 Condition (b).** `match_registry.first_sync_at` (TIMESTAMP naïf UTC : `persistRegistry`
  l'écrit en `time.Now().UTC()`). La borne est évaluée en Go avec une horloge injectable
  (`AnalysisPendingWindow = 24 h`, âge STRICTEMENT inférieur à 24 h), pas en SQL, pour que la
  frontière soit testable sans dormir. Pas de `start_time` (règle 8 : le fragment canonique ne
  concerne que les filtres/tris sur l'heure de JEU, ici on mesure l'âge de la SYNCHRO).
  Réserve connue et assumée : un joueur fraîchement onboardé (tout son historique a
  `first_sync_at` récent) voit l'indicateur sur ses matchs affichés tant que leur film n'est pas
  analysé, 24 h au plus : c'est exact (l'analyse n'est pas finie).
- **D-4 Condition (c2), ÉCART AU TEXTE DE LA DEMANDE.** La demande dit « pas de sentinelle
  `_processed` dans `match_citations` ». Vérifié sur pièces : `writeCitations`
  (`sync/citations.go`) n'écrit la sentinelle QUE pour un match à ZÉRO citation ; un match avec
  des citations porte des lignes de deltas et AUCUNE sentinelle. Lire « sentinelle absente »
  déclarerait « en cours » tous les matchs traités qui ont des citations. La sélection des
  candidats (`citations_backfill.go`, `LEFT JOIN ... WHERE mc.match_id IS NULL`) définit « pas
  encore calculé » = AUCUNE ligne pour le match. On lit donc : aucune ligne dans
  `match_citations_latest` pour ce `match_id` (vue `_latest`, règle ART n°2).
- **D-5 Condition (c1), AFFINÉE.** « Source du kill non traitée » = ni `MBitWeaponKills` ni
  `MBitFilmAbsent` dans `match_registry.backfill_completed` ET `killsource_sans_killfeed_rev IS
  NULL`. Un film lu sans kill-feed (`OutcomeNoKillFeed`, `registry_flags.go`) ne pose AUCUN bit
  mais pose cette révision : le film EST analysé, l'afficher « en cours » serait faux. Colonne
  vérifiée : migration `shared_registry_killsource_sans_killfeed_rev_v1`.
  Limite connue : `OutcomeUnknownKey` et `OutcomeCarteNonResolue` ne posent rien (le film existe,
  l'analyse ne peut pas aboutir pour l'instant) : l'indicateur reste jusqu'à la borne de 24 h,
  puis s'éteint. Acceptable et borné ; pas d'indicateur éternel.
- **D-6 Condition (c3).** Rejeu absent ET film non déclaré absent : `has_replay` faux
  (home : ensemble de `port.ReplayAvailability`, déjà résolu une fois par requête ; vue match :
  `header.replay_available`) ET pas de `MBitFilmAbsent`. Si l'ensemble des rejeux n'a pas pu
  être résolu (service non câblé ou listing en échec, `replayAvailability` rend nil), (c3) est
  IGNORÉE : on ne déclare pas « rejeu manquant » sur une lecture qui a échoué.
- **D-7 Règle.** `analysis_pending` = (a) ET (b) ET [(c1) OU (c2) OU (c3)] ; au-delà de 24 h :
  faux, quoi qu'il arrive. Fonction pure `analysis.IsAnalysisPending` (zéro DB/HTTP).
- **D-8 Lecture.** UN lot par page, jamais un N+1 : `port.AnalysisProbeRepository.LoadAnalysisProbes
  (ctx, matchIDs)` ; deux requêtes en tout (un `IN (...)` sur `match_registry`, lecteur shared LIVE
  `pdb.SharedReadDB()` ; un `IN (...)` sur `match_citations_latest`, base du joueur), car les deux
  tables vivent dans deux fichiers DuckDB. Pas de `v_gamertag_lookup`, pas de fenêtre `_latest`
  non bornée (ADR 0036). Aucune écriture, aucun schéma. Erreur : journalisée en WARN, champ faux
  (jamais d'indicateur fabriqué sur une lecture ratée).
- **D-9 Rafraîchissement web.** `refetchInterval` 60 s conditionnel dans un `*_logic.ts` pur
  (`analysisPending.logic.ts` de `lib/`) ; accueil (`useHomePage`, additionné au poll existant de
  première synchro) et vue match (`useMatchView`) ; `matchViewStaleTime` rend 0 quand la vue est
  en cours d'analyse. HTTP : la vue match répond `Cache-Control: no-store` tant que le flag est
  vrai (l'accueil a `max-age=30`, inférieur à 60 s : inchangé).
- **D-10 Composants.** `match-card.tsx` a une exemption de taille datée : rien n'y grossit.
  Nouveaux composants sous `components/ui/` : pastille + infobulle, emplacements en attente
  (frags assistés, citations, rejeu), bandeau, emplacement de section.
- **D-11 i18n.** Tuile : manifeste `common.toml` (clés `common.match_card.analysis_*`, générateur
  `build_i18n_manifests.mjs`). Vue match : `features/match-view/i18n.ts` (`Record<Locale, T>`).
  FR exactement ceux de la demande ; JAMAIS « en général » ni « quelques minutes ».
- **D-12 Couleurs.** Teinte info par le jeton sémantique `info` (`text-info` est déjà utilisé
  dans la tuile) ; fond sombre translucide par `bg-black/40` comme le bouton favori (précédent
  de la tuile). Aucun hex.

## 2. Étapes (ordre strict)

### E0 — Préparation — S

- [x] E0.1 Lire `CLAUDE.md` du worktree, invoquer les skills `plan-execution`, `arch-rules`,
  `db-schema`, `frontend-patterns`, `color-tokens`, `foundations-usage`.
- [x] E0.2 Vérifier sur pièces : bits (`matchflags`), colonnes (`first_sync_at`,
  `backfill_completed`, `killsource_sans_killfeed_rev`), vue `match_citations_latest`,
  capabilities (`film.kill_source`, `film.replay_artifact`), consommateurs de `MatchCard`.
- [x] E0.3 Écrire ce plan (décisions D-1 à D-12).
- [x] E0.4 `npm ci` dans `apps/web` (node_modules absent du worktree).
- Gate : plan relu, commit `indicateur(E0)`.

### E1 — Signal Go et tests — M

- [x] E1.1 `domain/analysis_pending.go` : `AnalysisProbe` (FirstSyncAt, RegistryBits,
  KillSourceReadNoFeed, CitationsRows).
- [x] E1.2 `analysis/analysis_pending.go` : `AnalysisPendingWindow`, `AnalysisPendingInput`,
  `IsAnalysisPending` (pure).
- [x] E1.3 `port` : `AnalysisProbeRepository`.
- [x] E1.4 `platform/duckdb/analysis_probe_repo.go` : `LoadAnalysisProbes` (2 requêtes).
- [x] E1.5 `service/analysis_pending.go` : `AnalysisPendingResolver` (horloge injectable,
  `Resolve`), branché sur `HomeService` (`recent_matches` + `favorite_matches`) et
  `MatchViewService` (en-tête).
- [x] E1.6 Wire : porte `film.kill_source` / `film.replay_artifact` (patron `withFilmArtifactRepos`).
- [x] E1.7 Handler vue match : `no-store` tant que pending.
- [x] E1.8 Tests : règle (3 cas de c, borne 24 h juste avant / pile / après, absence de
  capability, rejeu non résolu, FilmAbsent), repo sur DuckDB en mémoire, service (home +
  vue match), wire sur les `capabilities.toml` LIVRÉS (Infinite vrai, Halo 5 toujours faux).
  Bout en bout de l'accueil : `TestGetHomePage_AnalysisPending_DeBoutEnBout` (GetHomePage, repo simulé, c'est le harnais existant ; pas de harnais HTTP+DuckDB de la page).
- Gate : `go test` des paquets touchés ; `go test -tags=integration -p 1` des paquets touchés ;
  vet ; golangci-lint des paquets touchés.

### E2 — Contrat et types — S

- [x] E2.1 `openapi.yaml` mis à jour (RecentMatchItem, MatchViewHeader).
- [x] E2.2 `make generate-types` (jamais `generated.ts` à la main).
- Gate : `make check-types`.

### E3 — Web — L

- [x] E3.1 `lib/` logique de rafraîchissement (`analysisPending.logic.ts`) + tests.
- [x] E3.2 Tuile : pastille + infobulle, emplacements en attente (frags assistés, citations,
  rejeu) ; i18n manifeste ; tests des deux états.
- [x] E3.3 Vue match : bandeau INFO `role="status"`, emplacements de section (arme par frag,
  rejeu 2D), `i18n.ts` FR/EN ; tests des deux états.
- [x] E3.4 `refetchInterval` accueil + vue match, `matchViewStaleTime` ; query keys inchangées
  (déjà dans `lib/query/keys.ts`) ; tests.
- [x] E3.5 Animations CSS avec `prefers-reduced-motion`.
- Gate : vitest ciblé, tsc, eslint, lints maison (i18n, couleurs).

### E4 — Gates finaux — M

- [ ] E4.1 `go test` des paquets touchés + `go test ./...`.
- [ ] E4.2 `go test -tags=integration -p 1` des paquets touchés.
- [ ] E4.3 `make go-api-lint`.
- [ ] E4.4 `make check-types`.
- [ ] E4.5 `make test-web`.
- [ ] E4.6 `make gate-push`.
- Sorties consignées en section 4.

### E5 — Clôture — S

- [ ] E5.1 Entrée en tête de `.ai/thought_log.md`.
- [ ] E5.2 Plan à jour (toutes cases statuées), `disk-hygiene.ps1 -Lot indicateur -Apply`.
- [ ] E5.3 Push de `feat/indicateur-analyse` UNIQUEMENT, puis arrêt.

## 3. Découvertes (non traitées)

- (vide)

## 4. Journal

- 2026-10-16 E0 : plan écrit après lecture des pièces. Écarts à la demande documentés en D-4
  (sentinelle) et D-5 (film lu sans kill-feed).
- 2026-10-16 E1 : signal Go livré. Gate : `go test` vert sur domain/analysis/port/service/platform-duckdb/
  api/wire/api/handlers/archlint ; `go test ./internal/api` vert HORS `TestOpenAPIYAMLIsUpToDate` (rouge
  attendu, le contrat est le livrable d'E2) ; intégration `AnalysisProbeRepo` verte ; golangci-lint
  (nouveau code vs origin/main) 0 issue. Piège rencontré : un identifiant local `replay.X` hors de
  `film/` fait monter le plafond de `film_facade_surface_test` (variable renommée `lookup`).
- 2026-10-16 E2 : `openapi.yaml` regénéré (`MatchViewHeader.analysis_pending`), `generated.ts` par
  `npm run generate-types`. `RecentMatchItem` n'a PAS de schéma OpenAPI (`/pages/home` sert un corps
  JSON marshalé à la main, cf. note de `types.ts`) : le champ est ajouté au miroir manuel
  `lib/api/types.ts`, au même endroit que `has_replay`. Gate : `tsc -b` vert, `openapi-gen -check`
  et `check-generated-types-fresh.mjs` verts, `TestOpenAPIYAMLIsUpToDate` vert.
- 2026-10-16 E3 : web livré. Tuile : `analysis-pending-badge.tsx` (pastille + infobulle en portail,
  `aria-describedby`, ouverte au survol/focus, fermée à Échap), `match-card-analysis-pending.tsx`
  (frags assistés, citations, rejeu), intégrés dans `match-card.tsx` et `match-card-assisted-frags.tsx`
  (props, pas de gros ajout). Vue match : `MatchAnalysisPendingBanner` (role=status, teinte info),
  `analysis-pending-slot.tsx`, `arsenalPendingSlots` (blockPredicates) dans l'onglet Armes et terrain :
  « En cours d'analyse » (distance par arme) et « En préparation » (rejeu 2D, Emprise, positions).
  Rafraîchissement : `lib/analysis-pending/analysisPending.logic.ts` (60 s ; poll de première synchro
  30 s conservé), `useHomePage` / `useMatchView`, `matchViewStaleTime` = 0 en cours. Aucune nouvelle
  query key (`home` et `matchView` existent). Animations sous `motion-safe:` (prefers-reduced-motion).
  Gate : vitest ciblé 93 fichiers / 799 tests verts, `tsc -b` vert, eslint 0 erreur (2 warnings
  préexistants), lint:colors et lint:fields propres.
  Limite assumée : la puce « Rejeu à venir » s'affiche si le match est en cours d'analyse ET sans
  rejeu ; si le film est déclaré absent et que seules les citations manquent, elle peut annoncer un
  rejeu qui ne viendra pas, le temps que les citations soient posées (le signal serveur ne distingue pas).
