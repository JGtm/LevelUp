# Lot B — Document servi (Go) — journal d'exécution

> Plan : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`, lot B. Worktree
> `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-v2-docservi`, branche `feat/v2-docservi`,
> base `a21fd77f4`. Contrat d'exécution : skill `plan-execution`.
> Décision utilisateur ferme n°2 : séparer MAINTENANT le document stocké du document servi,
> forme de fil strictement inchangée dans ce lot.

## Items

- [x] **B.1** — `internal/domain/replaydoc` créé : 99 types struct + 3 types scalaires nommés
  (`WeaponChangeKind`, `PickupKind`, `EquipmentChangeKind`), miroir exact de la forme de fil.
  Mêmes noms de types (Huma en dérive les noms de schémas OpenAPI), mêmes tags JSON, mêmes
  `omitempty`, MÊME ORDRE de champs (l'ordre gouverne la liste `required` du schéma généré).
  Aucun import d'`internal/analysis/replay`. Doc de type = premier paragraphe du type source,
  reporté ; les chroniques de schéma restent côté stocké. `ContractVersion = 39`, documentée
  (« ne bouge que si la forme servie change ; la version stockée gouverne la re-cuisson »).
  10 fichiers, 1 319 L, tous < 500 L.
  Commit `40667b2c2`.
- [x] **B.2** — `internal/service/replayview` : 99 fonctions de projection pures (zéro E/S),
  trois entrées publiques (`FromArtifact`, `MapBackgroundOf`, `MapCalloutsOf`), trois
  génériques (`sliceOf`, `ptrOf`, `mapOf`) qui PRÉSERVENT la nullité. `parity_test.go` :
  trois volets (décision par champ stocké avec liste datée `champsNonServis`, source par champ
  servi, projection exhaustive prouvée sur un document rempli de valeurs distinctes) + deux
  tests de contrat (nullité, `ContractVersion` vs `schemaVersion`).
  `TestReplayDocumentFieldCountIsFrozen` et `replaySchemas` réconciliés : ils portent
  désormais sur le contrat SERVI (`domain/replaydoc`) ; chronique et en-tête mis à jour.
  Commit `1d32a6356`.
- [x] **B.3** — `handlers/replay.go` sert `replaydoc` sur ses trois routes JSON
  (`replayOutput`, `backgroundOutput`, `calloutsOutput`) ; `port.ReplayService` annonce les
  types servis ; le service convertit après lecture de l'artefact et après ses trois
  résolutions de titre. `analysis/replay.ReplayDocument` n'est plus un schéma OpenAPI.
  Recherche exhaustive faite (`grep -rn "analysis/replay" internal/api/` : un seul fichier de
  code, `replay.go`, plus une mention en commentaire dans `replay_local_gate.go`).
  Commit `fac6d2c07`.
- [x] **B.4** — `internal/archlint/no_analysis_type_in_http_body_test.go` : scan AST de tous
  les `.go` d'`internal/api/` (tests compris) ; sont jugés les champs `Body`/`RawBody` de
  toute struct ET tous les champs des structs `*Input`/`*Output` ; détection à toute
  profondeur d'expression de type. Plancher de 80 corps scannés (104 mesurés) pour qu'un
  ratchet devenu aveugle échoue au lieu de passer. Allowlist datée à UNE entrée
  (`patternsOutput.Body`, découverte du scan, hors périmètre — voir Découvertes).
  Commit `28b3e4ce1`.

## Gates

Toutes les commandes jouées en avant-plan, en série, depuis
`C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-v2-docservi\apps\go-api` sauf mention,
avec `GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-docservi CGO_ENABLED=1`.

| Gate | Commande | Dernière ligne |
|---|---|---|
| Contrat régénéré | `go run ./cmd/openapi-gen` | `openapi-gen: api/openapi.yaml écrit (676077 octets)` |
| Diff contrat | `git diff --exit-code apps/go-api/api/openapi.yaml` | `OPENAPI_SANS_DIFF` (sortie vide, exit 0) |
| Types web | `make generate-types` (racine du worktree) | `../../apps/go-api/api/openapi.yaml → src/lib/api/generated.ts [278.4ms]` |
| Diff types web | `git diff --exit-code apps/web/src/lib/api/generated.ts` | `GENERATED_TS_SANS_DIFF` (sortie vide, exit 0) |
| Suites ciblées | `go test ./internal/api/... ./internal/service/... ./internal/domain/... ./internal/archlint/... ./contracttest/...` | `ok levelup/go-api/contracttest 0.361s` |
| `make go-api-test` | `CGO_ENABLED=0 LEVELUP_DEMO_MODE=true go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -timeout 60s -count=1` | aucune ligne non-`ok` |
| Compilation | `go build ./...` | `BUILD_OK` |
| Lint | `golangci-lint run ./internal/domain/replaydoc/... ./internal/service/replayview/... ./internal/archlint/... ./internal/api/handlers/... ./internal/port/... ./contracttest/...` | `19 issues` — **aucune dans un fichier du lot** (toutes dans `internal/api/handlers/{admin*,settings,sync_handler,field_mappings,feature_matrix,capabilities,setup,home,helpers}.go`, dette antérieure gelée par `only-new-issues`). `golangci-lint run` sur les seuls paquets créés (`replaydoc`, `replayview`, `archlint`) : `0 issues.` |
| Typecheck web | `npm --prefix apps/web run typecheck` | `> tsc -b` puis `EXIT=0` |
| Arbre propre | `git status --short` après gates | sortie vide (aucun `routeTree.gen.ts` à restaurer) |

Aucun test skippé. Aucune allowlist agrandie sans justification datée (une seule entrée
posée, avec date d'inscription, date cible de retrait et critère mesurable).

## Preuves

**1. Les 99 schémas Huma sont IDENTIQUES des deux côtés.** Avant d'écrire quoi que ce soit,
la question « Huma renomme-t-il les schémas en cas de collision ? » a été tranchée sur pièces
dans la source du module (`huma@v2.39.1/registry.go:126-149`) : `DefaultSchemaNamer` prend le
seul nom du type (le paquet est ignoré), et le suffixe numérique ne s'applique QU'aux types
anonymes — deux types NOMMÉS de même nom font `panic("duplicate name")`. Un miroir aux mêmes
noms de types rend donc exactement les mêmes noms de schémas, à condition que l'ancien type
sorte du registre (ce que fait B.3). Contrôle direct par un programme jetable : deux
`huma.NewMapRegistry` séparés, l'un chargé des trois racines `analysis/replay`, l'autre des
trois racines `replaydoc` — **99 schémas de chaque côté, JSON identique octet pour octet**.
Le programme n'est pas versionné (outil de génération à usage unique, supprimé avant les
gates) ; le gate permanent est le double `git diff --exit-code` ci-dessus.

**2. Le convertisseur est prouvé par mutation, pas seulement par une suite verte.**
- Retirer `StartFrame: v.StartFrame` du seul `toTrack` :
  `TestProjectionCopieChaqueChamp` échoue en nommant le chemin —
  `.tracks[0].startFrame : present cote stocke, ABSENT cote servi`.
- Le retirer AUSSI de `replaydoc.Track` (le scénario « champ silencieusement perdu ») :
  `TestChaqueChampStockeAUneDecision` échoue en plus —
  `Track.StartFrame est publie par l'artefact et ABSENT du document servi, sans entree dans champsNonServis`.
- Les deux mutations ont été annulées et la suite re-jouée verte.

**3. Le ratchet B.4 est prouvé par mutation.** Remettre `Body replay.ReplayDocument` dans
`replayOutput` : `TestAucunTypeAnalysisEnCorpsHuma` échoue en nommant
`internal/api/handlers/replay.go:replayOutput.Body -> levelup/go-api/internal/analysis/replay.ReplayDocument`.
Mutation annulée, test re-joué vert.

## Décisions

1. **Le convertisseur vit dans `internal/service/replayview/`, sous-paquet de
   `internal/service/`.** Le plan dit « dans `internal/service/` » ; 1 419 L de projection
   dans la racine du paquet auraient violé la règle 5 (fichier ≤ 500 L) sans apporter de
   lisibilité, et le dépôt a déjà le précédent `internal/service/teammates/`. Les fichiers
   sont nommés en miroir de ceux de `replaydoc` (`convert_<bucket>.go`), pour qu'un ajout de
   type ait un seul endroit évident de chaque côté.

2. **Projection explicite, pas un aller-retour JSON ni une copie par réflexion.** Les deux
   raccourcis auraient rendu la suite verte à moindre coût, mais ils retirent précisément ce
   que la séparation doit apporter : une DÉCISION par champ. Une fonction par type rend un
   champ non servi visible dans le diff, là où un `json.Marshal`/`Unmarshal` le rendrait
   invisible (et coûterait deux sérialisations de 2 Mo par requête).

3. **Les tranches de scalaires sont partagées, pas recopiées** (documenté dans l'en-tête de
   `replayview`) : l'appelant lit l'artefact, projette, et jette la valeur stockée. Recopier
   des mégaoctets de coordonnées pour une valeur qui meurt à la ligne suivante serait payer
   le rejeu deux fois. La nullité, elle, est préservée partout — sur les champs sans
   `omitempty` le client voit la différence entre `null` et `[]`.

4. **Aucun test d'égalité permanente entre schéma stocké et schéma servi.** Le contrôle
   « 99 schémas identiques » est une preuve DU JOUR de la séparation, pas un invariant : le
   figer en test interdirait la divergence que le lot existe pour rendre possible. Les deux
   invariants permanents sont ailleurs — le golden `openapi.yaml` tient le contrat, le test de
   parité tient « aucun champ stocké perdu sans décision écrite ».

5. **Les enrichissements du service (objectifs statiques, socles de carte, libellés d'arme et
   de véhicule) restent posés sur le document STOCKÉ, avant projection.** Minimal-diff
   volontaire : ce lot ne doit changer aucun octet de la réponse. Leur remontée éventuelle
   côté servi est un choix de conception qui appartient au lot qui touchera ces catalogues.

6. **Les enrichissements du contrat côté `contracttest` changent de sujet, pas de forme.** Le
   fichier confronte désormais `domain/replaydoc` à `api/openapi.yaml` ; la garantie « la
   cuisson écrit bien tout ce que le contrat promet », qu'il portait implicitement en étant
   adossé au format de fichier, est reprise par `replayview/parity_test.go`.

## Découvertes (hors périmètre, NON traitées)

1. **`handlers/patterns.go:77` sert `analysis/patterns.PatternReport` en corps de route.**
   Découverte par le ratchet B.4 lui-même (le grep préparatoire l'avait manquée : il ne
   cherchait que les paquets du rejeu). Quatre types d'`internal/analysis/patterns`
   (`PatternReport`, `BehavioralPattern`, `ContextualPattern`, `Lever`) sont des schémas du
   contrat public — le même défaut que le rejeu, mais antérieur au lot B et hors de son
   périmètre. Inscrit à l'allowlist avec date (2026-09-05), cible de retrait (2026-11-01) et
   critère mesurable (intersection des types exportés d'`analysis/patterns` avec
   `components.schemas` = 0 ; elle vaut 4 aujourd'hui). Traitement : un DTO `domain/` +
   projection, sur le patron de ce lot.

2. **`Vec3` est un faux positif de la reproduction du constat C7.** Le registre annonce
   « 100 schémas sur 165 types » par un `comm -12` entre les noms de types exportés
   d'`analysis/replay` et les clés de `components.schemas`. Le schéma `Vec3` du contrat vient
   en réalité d'`internal/games/canonical/events.go` (il est référencé par `MatchEvent`), pas
   de `analysis/replay/killpos.go:99` qui porte un homonyme non atteignable depuis les trois
   racines de route. Le chiffre juste est **99**, pas 100 — sans conséquence sur le constat ni
   sur le traitement.

3. **`port/services.go` publie encore `analysis/positions` et `analysis/temporal`** (constat
   C8), et `port/session_usage.go` `analysis/sessionusage`. Le lot B n'a retiré que `replay`.
   Le ratchet B.4 ne les couvre pas : il porte sur les corps de route Huma, pas sur les
   signatures de port — c'est le ratchet de couches (C1/C2/C3, non planifié) qui les prendra.

4. **`Track.name` n'est jamais écrit par le Go** (inventaire déjà consigné en tête de
   `contracttest/replay_contract_test.go`, « le seul des douze dont la suppression est
   acquise »). Il est désormais retirable SANS toucher au format d'artefact : une entrée dans
   `champsNonServis` suffirait. Non fait — c'est un arbitrage produit, pas un item du lot.

## Questions ouvertes

- Aucune bloquante. La seule décision à confirmer par le superviseur est le choix n°1
  (sous-paquet `internal/service/replayview` plutôt que la racine de `internal/service`).
