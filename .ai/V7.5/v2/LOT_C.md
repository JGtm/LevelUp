# Lot C — Capabilities et vocabulaire (Go + web)

> Plan : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`, section « Lot C ».
> Worktree `LevelUp-wt-v2-capabilities`, branche `feat/v2-capabilities`, base `a21fd77f4`.
> Décisions utilisateur appliquées : n°3 (`film.replay_artifact` gouverne la PRODUCTION,
> l'affichage suit ; capability title-level `replay` pour la route web) et n°6 (« heatmap »
> banni au profit de « carte de chaleur » ; « lobby » gardé comme mot assimilé, documenté).

## Items

### [x] C.1 — La clé fine et la capability title-level

Commit `b2f536c14`.

- `games.CapFilmReplayArtifact` (`"film.replay_artifact"`) déclarée dans
  `internal/games/adapter.go` avec le commentaire de doctrine des cinq clés `film.*`
  (clé FINE, gouverne la PRODUCTION, ne pas l'élargir) ; ajoutée à
  `games.AllCapabilityKeys()` ; compteur du garde-fou porté de 25 à 26.
- Manifestes : `supported` pour halo_infinite, `not_exposed` pour halo_5, avec dans les
  deux cas le commentaire qui dit pourquoi. Parité TOML ↔ `CapabilityMap` hardcodée
  maintenue dans les deux adapters (`halo_infinite/adapter_data.go`,
  `halo_5/adapter_data.go`) — sinon `capabilities_parity_test.go` rougit.
- `title.CapReplay` (`"replay"`) déclarée dans `internal/domain/title/registry.go`,
  ajoutée à `knownCapabilities` (`config_loader.go`) et au descripteur built-in de
  halo_infinite ; côté web, `'replay'` entre dans `TITLE_CAPABILITIES` et son libellé
  dans `FEATURE_LABEL` (`Record<TitleCapability, …>` : le typage l'exigeait).
- `synthetic_title_b` déclare les SIX clés `film.*` : `film.replay_artifact` en
  `supported`, les cinq dérivés en `not_exposed` — la fixture qui prouve que ces clés sont
  fines et que chaque porte se ferme séparément.

Ratchets couverts sans aucune allowlist : parité Go ↔ TS (`capabilities_ts_mirror_test.go`),
`knownCapabilities` ↔ constantes, « accordée par un titre public », « lue par un
consommateur », parité TOML ↔ CapabilityMap des deux adapters.

**Écart au plan, vérifié sur pièces.** Le plan demandait de déclarer `CapReplay` dans
`config/titles/halo_infinite/title.toml`. Ce fichier N'EXISTE PAS, et un tel fichier serait
IGNORÉ : `config_loader.go` documente que « halo_infinite reste le descripteur BUILT-IN
câblé en dur dans NewRegistry() […] un title.toml halo_infinite est donc ignoré ici », et
le loader émet `title_builtin_toml_ignored` s'il en trouve un. La capability est donc
déclarée dans `NewRegistry()`, seul endroit où elle prend effet. Rien n'est perdu : le
bootstrap projette `descripteur.Capabilities` tel quel dans `availableTitles[].capabilities`.

**Chevauchement assumé avec C.2.** Le ratchet `TestCapabilitiesReferencedByAConsumer`
refuse une capability title-level que personne ne lit (« pas de feature livrée OFF »,
CLAUDE.md n°11). C.1 ne pouvait donc pas être vert seul : le middleware
`RequireCapability(titleRegistry, CapReplay)` sur le groupe des routes `/replay*`
(`server_apiv1.go`) part avec C.1, et C.2 s'appuie dessus pour son test 503.

### [x] C.2 — Les portes Go (décision 3)

Commit `ee58a81c5`.

1. **Production** — `replayartifacts.Run` refuse EN TÊTE (3 lignes dans `artifacts.go`,
   la décision dans `capability.go`) : avant `selectionnerLeTravail`, avant
   `rattraperCartesAbsentes`, avant `enqueueAll`. Deux « non » distincts, comme les gates
   jumeaux du paquet : TOML illisible = incident (WARN), clé absente = configuration de
   titre. Ce dernier est journalisé en **INFO une fois par cycle** (consigne du lot) et
   non en DEBUG comme `usage.go`/`bombstats.go` : cette porte éteint l'étape ENTIÈRE, pas
   un dérivé — la différence de niveau est délibérée et écrite dans le fichier.
2. **Lecture** — `withFilmArtifactRepos` (`internal/api/wire/registry_pages_film.go`) :
   les deux loaders du film ne sont câblés que si le titre déclare la clé. La doc inversée
   de `registry_pages.go:113-117` est réécrite et devient vraie par la seule voie qui
   pouvait la rendre vraie (le service ne dégrade que sur repo `nil`).
3. **Routes** — les quatre `/replay*` sous `RequireCapability(CapReplay)` (posé en C.1) ;
   le commentaire « Hors sous-groupe capability : la disponibilité EST la présence
   d'artefact » est remplacé par la règle des deux portes (503 « ce TITRE n'a pas de
   rejeu » / 404 « CE MATCH n'en a pas »).

Tests, tous sur les fichiers LIVRÉS (`config/titles/**`, registre réel) :
- `replayartifacts/capability_test.go` : porte vraie pour halo_infinite, fausse pour
  halo_5 ; capabilities illisibles → WARN ; `Run` sur halo_5 ne lit PAS la base et
  n'enfile RIEN (la preuve la plus économique que rien n'a commencé) ;
- `api/wire/registry_pages_film_test.go` : `/objective-events` et `/positions` → 503
  `capability_not_supported` sur halo_5 sans qu'aucun loader soit appelé, 200 sur
  halo_infinite (volet indispensable : une porte qui refuserait tout passerait l'autre) ;
- `api/handlers/replay_capability_gate_test.go` : les QUATRE routes `/replay*` → 503
  `capability_unavailable` sur halo_5 sans résolution de service, et passage sur
  halo_infinite.

Deux tests préexistants réparés (`journal_test.go`, `placement_test.go`) : ils appelaient
`Run` sans `RepoRoot`/`TitleSlug` et parlent du cas où l'étape TOURNE — ils déclarent
désormais halo_infinite. `racineDepot` descend de `usage_integration_test.go` (tag
`integration`) vers `capability_test.go` (sans tag) : la porte s'exerce aussi hors
intégration, deux copies auraient divergé.

### [x] C.3 — Le web : client, hook, FeatureGate, trois surfaces

Commit `fc4307c0c`.

- `lib/capabilities/dataCapabilities.ts` : client de `GET /titles/{slug}/capabilities`
  (query key `titleDataCapabilities`, une requête par titre partagée par tous les
  appelants), `useDataCapability(key)`, `hasDataCapabilityIn` (prédicat pur).
  `supported`/`degraded` = oui, `not_exposed`/absent = non — même sémantique que
  `CapabilityMap.Has` côté Go. **Fail-open** tant que la réponse n'est pas là (pas de
  clignotement au montage ; une panne de cet endpoint n'ampute pas l'application), comme
  `useCapability`.
- `FeatureGate` accepte l'une OU l'autre porte, cumulables. Découpé en deux
  sous-composants : la porte data-level ouvre une requête, donc exige un
  `QueryClientProvider` — les dizaines de gates title-level ne doivent ni le payer ni le
  faire exiger par leurs tests (deux tests existants cassaient sur la première version).
  La branche est prise sur la PRÉSENCE de la prop, jamais sur sa valeur.
- `MatchKillDistanceSection` (L3) : porte 1 `film.kill_positions` → la carte n'est plus
  rendue du tout ; porte 2 inchangée (l'état vide explicite reste pour un match sans frag
  mesuré). Doc inversée de `MatchViewPage.tsx:352` réécrite.
- `SessionUsageSection` / `usageLogic.ts` (L4) : `unsupported` → `hidden` ; `load_failed`
  garde son état vide (transitoire, il faut le dire). Le libellé devenu mort
  (`unavailableUnsupported`, FR + EN + champ du type) est supprimé.
- Explorer (L5) : filtre « Avec rejeu / Sans rejeu » et colonne « Rejeu » des DEUX
  tableaux gatés par `replay` ; la porte de LIGNE (`has_replay`) descend dans
  `MatchReplayLink`, point unique de l'icône — elle couvre du même coup la carte de match.
- Tests : `dataCapabilities.test.tsx` (prédicat, hook, les deux portes du gate),
  `MatchKillDistanceSection.test.tsx` (titre sans clé → RIEN, pas même l'état vide),
  `SessionUsageSection.gate.test.tsx`, `ExplorerPage.test.tsx` (filtre présent/absent),
  `ExplorerMatchesTable.test.tsx` et `SquadSynergyHistoryTable.test.tsx` (colonne).

**Ajout au périmètre, assumé** : `internal/games/capabilities_front_parity_test.go` —
jumeau data-level du garde title-level existant. Tout littéral passé à
`useDataCapability` / `dataCapability=` doit être une `CapabilityKey` réelle, et toute
entrée de `DATA_CAPABILITIES` doit être réellement consommée. Sans lui, une typo fermerait
une porte pour toujours, sans erreur ni test rouge — c'est le piège n°1 de cet item, et le
dépôt le ferme déjà pour l'autre système.

**Décision de conception** : `DATA_CAPABILITIES` ne contient QUE `film.kill_positions`.
`film.usage_summary` n'y est pas parce que sa porte de titre arrive déjà dans le payload
(`usage.unavailable_reason === 'unsupported'` = « ce titre ne déclare pas
film.usage_summary », `domain/session_usage.go`) : la lire une seconde fois ferait deux
sources de vérité et une requête inutile. `film.replay_artifact` n'y est pas non plus : la
porte d'affichage du rejeu est la capability title-level `replay`. Une clé listée « pour
plus tard » serait du vocabulaire mort (le nouveau garde le refuse).

### [x] C.4 — Vocabulaire (décision 6)

Commit `2ddca8291`.

- `heatmap` ajouté à `FORBIDDEN_PATTERNS` du garde anti-anglicismes. **Preuve de morsure
  jouée** : la chaîne remise en « Heatmap d'activité commune » fait rougir le garde
  nommément (`explorer.toml:explorer.player.activity_heatmap_title — "Heatmap d'activité
  commune" (pattern: heatmap)`), puis la correction est remise.
- Trois chaînes FR corrigées — les seules VALEURS françaises qui portaient le mot :
  `manifests/explorer.toml` (« Carte de chaleur d'activité commune »),
  `manifests/timeseries.toml` (« Carte de chaleur d'intensité (jour × heure) »),
  `MatchPositionsHeatmap.tsx` (titre seul). Modules i18n générés régénérés
  (`build_i18n_manifests.mjs`), assertion du test du composant alignée.
- Le reste des occurrences de « heatmap » dans `apps/web/src` sont des identifiants
  techniques, hors décision de vocabulaire : type de série ECharts (`type: 'heatmap'`),
  tokens de couleur (`heatmap-cold`, `heatmap-freq-low`…), noms de fichiers et de clés de
  manifeste. `config/titles/**` n'en contient aucune.
- `lobby` inscrit dans la doctrine du garde comme mot ASSIMILÉ, à côté de badge et
  playlist, avec sa justification : c'est le DÉNOMINATEUR d'une part (« les 8 ou 12
  joueurs du match »), « partie » désigne le match lui-même, « salon » n'a pas ce sens en
  français de jeu vidéo, et `engagement.toml` sert déjà le mot en prose FR.

**Écart de chemin, vérifié sur pièces.** Le plan citait
`config/titles/halo_infinite/mappings/explorer.toml:54` et `timeseries.toml:194` ; les
fichiers réels sont `apps/web/src/lib/i18n/manifests/{explorer,timeseries}.toml` aux mêmes
lignes (ce sont eux que scanne le garde). Aucun manifeste de `config/titles/` ne porte le
mot.

## Gates

Tous joués en avant-plan, depuis `apps/go-api` pour le Go, avec
`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-capabilities CGO_ENABLED=1`.

| Gate | Commande | Dernière ligne |
|---|---|---|
| Tests Go du lot | `go test ./internal/domain/title/... ./internal/games/... ./internal/api/... ./internal/sync/replayartifacts/... ./contracttest/... ./internal/archlint/...` | `ok levelup/go-api/internal/archlint (cached)` — aucun FAIL |
| Suite Go complète | `go test ./internal/... ./contracttest/...` | aucune ligne hors `ok` / `no test files` |
| Build | `go build ./...` | `build exit=0` |
| Lint Go (ratchet CI) | `golangci-lint run --timeout 5m --new-from-merge-base=origin/main` | `0 issues.` |
| Types OpenAPI | `make generate-types` | `../../apps/go-api/api/openapi.yaml → src/lib/api/generated.ts [266.2ms]` — **aucun diff** |
| Typecheck web | `npm --prefix apps/web run typecheck` | `typecheck exit=0` |
| Lint web | `npm --prefix apps/web run lint` | `28 problems (0 errors, 28 warnings)` → `lint exit=0` (warnings préexistants : `react-hooks/incompatible-library` sur `useReactTable`) |
| Vitest ciblé | `node_modules/.bin/vitest run --pool=forks src/lib/capabilities src/features/match-view src/features/session-detail src/features/explorer src/lib/i18n/no-anglicisms.guard.test.ts` | `Test Files 84 passed (84)` / `Tests 716 passed (716)` |
| Vitest complet | `node_modules/.bin/vitest run --pool=forks` | `Test Files 591 passed (591)` / `Tests 6249 passed | 14 skipped (6263)` |

**Correction de commande** : le plan écrivait `./internal/contracttest/...` ; le paquet est
à `apps/go-api/contracttest/` (le chemin du plan échoue en `[setup failed]`).

## Découvertes (hors périmètre, non traitées)

1. **`MatchPositionsHeatmap` et les hooks du film de la Match View.** Le verdict V-GO-B2
   relève que `features/match-view/queries.ts:22-26` et `:47-51` promettent un 503 que le
   serveur ne rendait pas ; C.2 rend cette doc VRAIE (rien à corriger). Reste que
   `useMatchObjectiveEvents` / `useMatchPositions` n'ont pas de garde de capability dans
   leur `enabled` : sur un titre sans film, les deux requêtes partent et prennent
   désormais un 503 au lieu d'un 200 `[]`. La carte de chaleur des positions
   (`MatchPositionsHeatmap`) affichera alors son état vide « Aucune position décodée pour
   ce match » — la règle des deux portes voudrait qu'elle ne soit pas rendue. Trois
   surfaces étaient nommées au lot C ; celle-ci ne l'était pas.
2. **Dictionnaires i18n inline hors du garde anti-anglicismes.** `MatchPositionsHeatmap.tsx`
   porte son propre dictionnaire `TEXT = { fr, en }` : le garde ne scanne que les cinq
   dicts de features listés et les neuf manifestes. Sa chaîne est corrigée à la main ici,
   mais rien n'empêche une régression. Élargir le périmètre du garde (ou déplacer ce
   dictionnaire dans `features/match-view/i18n.ts`) est un lot en soi.
3. **`film.replay_artifact` déclarée `not_exposed` chez halo_5 alors que ses cinq sœurs
   sont ABSENTES du même fichier.** Asymétrie voulue (un refus explicite se lit ; celle-ci
   gouverne deux routes servies) et documentée dans le TOML comme dans la CapabilityMap,
   mais c'est une divergence de style dans un fichier par ailleurs homogène.

4. **`MatchHeader.replayLink.tsx` (le bouton « Rejeu » de la fiche de match) ne porte que
   la porte de DONNEE** (`header.replay_available`, un `os.Stat` cote serveur), pas la
   porte de TITRE. Aucun bloc mort en pratique — sur un titre sans film l'artefact n'existe
   jamais, donc le bouton ne s'affiche jamais — mais c'est la seule des quatre entrees du
   rejeu a ne pas porter les deux portes. La route qu'il ouvre est gatee par le lot D.

## Questions ouvertes

Aucune bloquante. Deux points d'arbitrage pour la revue :

- le niveau de log INFO du refus de production (consigne du lot) contre le DEBUG des deux
  gates jumeaux du même paquet — justifié dans `capability.go`, à confirmer ;
- la découverte n°1 (carte de chaleur des positions) : à verser au lot D, ou à traiter en
  ronde de correction du lot C ?
