# Revue `plan-review` du plan d'intégration — corrections à fondre dans le plan AVANT l'ouverture de l'étape B

> Relecture à contexte frais du 2026-09-05 (agent Opus, grille `plan-review`, affirmations du §1
> re-vérifiées sur pièces). Verdict : **exécutable après corrections** ; l'étape A n'est touchée par
> aucun constat et continue. Ce fichier est un addendum temporaire : ses textes remplacent ou
> complètent les sections du plan à la clôture de A (le plan est en cours d'écriture par l'agent A,
> on n'y écrit pas à deux). Il est supprimé une fois fondu.

## BLOQUANT

### BL-1 — `openapi.yaml` et `generated.ts` sont GÉNÉRÉS ; B et C les modifient tous deux
`apps/go-api/api/openapi.yaml` porte l'en-tête « FICHIER GÉNÉRÉ — NE PAS ÉDITER À LA MAIN —
`make openapi-gen` — verrouillé par `TestOpenAPIYAMLIsUpToDate` (golden byte-à-byte) ». Les deux
branches le touchent (véhicules +282, session-usage +307) ainsi que `apps/web/src/lib/api/generated.ts`
(+132, +140) : conflit garanti sur des fichiers générés, qu'une résolution manuelle rend faux en
silence. `make generate-types` ne régénère que le TS. Aucun job CI ne vérifie la fraîcheur de
`generated.ts` (`tools/check-generated-types-fresh.mjs` n'est appelé que par `make openapi-check`).

Texte à insérer (B.3bis et C.1bis) :
```
CONTRAT OPENAPI : `openapi.yaml` et `generated.ts` sont GÉNÉRÉS. Un conflit de merge sur l'un des
deux ne se résout JAMAIS à la main : on prend `--ours` (indifférent), puis on REGÉNÈRE et on
committe le résultat de la machine :
    make openapi-gen && make generate-types && make openapi-check
Gate de l'étape (B et C) :
    cd apps/go-api && go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1
    cd apps/go-api && CGO_ENABLED=0 go test ./contracttest/... -run TestContract -count=1
    node tools/check-generated-types-fresh.mjs
Si `make openapi-gen` produit un diff APRÈS résolution, la résolution Go était incomplète :
corriger le handler/DTO, jamais le YAML.
```

### BL-2 — Le 39 n'est réservé par rien
`feat/v75` a bumpé sept fois en trois jours et a déjà subi une renumérotation par collision
(`6ce0fcc2a`, 29 -> 30, « collision avec la lunette »). F.1 ne dit rien du numéro.

Complément à D3 et F.1 :
```
LE NUMÉRO N'EST PAS ACQUIS. (a) à l'ouverture de B, annoncer à l'utilisateur que le 39 est pris
par l'intégration (seul mécanisme de réservation du dépôt) ; (b) F.1 vérifie
`git show origin/feat/v75:apps/go-api/internal/analysis/replay/document.go | grep 'const SchemaVersion'`
AVANT le merge. Si l'amont a atteint 39 ou plus : notre bump devient N+1, constante et
commentaire corrigés, les 13 TSV RE-FIGÉS (-update) puis re-vérifiés 13/13 — le numéro entre dans
le digest `artifact`.
```

### BL-3 — Liste des jobs CI incomplète
`shared-social-gate.yml` se déclenche sur `push` vers `feat/**` dès que le diff touche
`internal/platform/duckdb/**` ou `internal/persist/**` — C touche les deux. `E2E React` ne tourne
que sur `pull_request`.

Remplace la parenthèse de jobs de D8 :
```
CI verte au niveau JOB sur la liste MESURÉE au push (`gh run view --json jobs`) : `Frontend`,
`Go Build + Test` (matrice), `Go Lint`, `Go Lease Enforcement`, `Go Coverage + Baseline`,
`OpenAPI Lint`, `Go Contract Test`, et `ADR 0021 Gate — shared_social invariants` (workflow
séparé, déclenché par paths). `E2E React (Playwright)` ne tourne PAS sur un push : son absence
n'est pas un rouge.
```

## IMPORTANT

### IMP-1 — B.5 attend des écarts sur des étapes inexistantes (`shots`, `coverage`)
Les 45 étapes n'ont ni `shots` ni `coverage` ; `observe_test.go` ne lit que le corps de
`BuildFromFilm`, or `buildShots`/`attachVehicles`/`attachVehicleShots` vivent dans
`BuildFromPositions` (même régime que `bots`/`successions` le 03/09 : couverts par `artifact`).

Remplace B.5 :
```
B.5 Harnais. ÉCARTS ATTENDUS, SUR LA LISTE RÉELLE DES ÉTAPES : une étape NEUVE `vehicles` (seule
greffe DANS BuildFromFilm) ; `artifact` bouge sur 13/13 (numéro de schéma) et davantage sur les
films à véhicule ; AUCUNE autre étape ne bouge. `shots` et `coverage` NE SONT PAS des étapes
observées — ne pas leur en créer, le garde la refuserait et `artifact` les couvre. Témoin
obligatoire : un BTB du corpus.
```

### IMP-2 — B.2 : cinq sites de production, un manquant, périmètre à fermer
`build_vehicles.go` appelle : `:84 ScanFilmWorldObjectKeyframes(dir, 40)` — la forme neuve
EXISTE (`filmdec/world_object_census.go:79 ScanWorldObjectKeyframes(film, ti)`) : remplacement
d'appel ; `:90 ScanFilmVehicleCreationsForBand` ; `:96 ScanFilmBipedPositionsForBand` (n'existe
pas sur cuisson-perf) ; `:125 ScanFilmBipedAimOnly` ; `:139 ScanFilmVehicleEvents`. Formes neuves
+ enveloppes `(dir)` hors production pour `ScanKeyframeRecordSpans`, `ScanVehicleCreations`,
`ScanVehicleOccupancy`. `decodeFilmVehicleScan(filmDir, ...)` devient `decodeVehicleScan(film,
ctx, ...)` : UN appel dans `BuildFromFilm`, UNE étape `vehicles`. ENVELOPPES OBLIGATOIRES
(appelants mesurés dans les tests v13 de `wt/vehicule-deadstate`) : `ReadFilmChunk`,
`CountFilmChunks`, `ScanFilmBipedPositions`, `ScanFilmBipedPositionsForBand`. Une enveloppe sans
appelant après merge se supprime.

### IMP-3 — Les TSV gagnent des LIGNES, c'est attendu
45 -> 48 étapes à l'étape A (`translocations`, `abilityImpulses`, `abilityCharges`) -> 49 à B
(`vehicles`). Une ligne AJOUTÉE n'est pas un écart ; une ligne MANQUANTE l'est (et
`observe_test.go` rougit avant). Gate explicite : compter les lignes des 13 TSV après `-update`
(48 en A, 49 en B, partout).

### IMP-4 — Gate web = ce que fait le job `Frontend`, pas moins
```
cd apps/web && npm run typecheck && npm run lint && npm run lint:fields
cd apps/web && npm run test        # vitest COMPLET
cd apps/web && npm run build
node tools/knip-ratchet.mjs        # ratchet code mort — casse typiquement à la réconciliation
```
Un export neuf non consommé (C : session-detail ; D : components/ui) fait sauter le ratchet :
câbler ou supprimer, JAMAIS relever le plafond.

### IMP-5 — L'amont ne gèle pas tout seul (D8bis)
`feat/v75` prend 15 à 121 commits/jour. (a) DEMANDER à l'utilisateur un gel de `feat/v75` pour la
durée de B, ou à défaut qu'il signale tout bump de schéma / tout balayage `Scan*` neuf ; (b) au
DÉBUT de chaque étape B, C, D, E : `git fetch` puis re-mesurer commits depuis A, `const
SchemaVersion`, et `git diff <base_A>..origin/feat/v75 | grep -oE 'ScanFilm[A-Za-z]+\('`. Delta
non nul = mini-réconciliation IMMÉDIATE (protocole A). Mesure écrite au §6 même nulle.

### IMP-6 — D.1 : le conflit de D est avec C, pas avec B
Mesuré : D ∩ B = `match-replay/i18n.ts`, `i18nContract.ts` (union des clés, parité FR/EN par le
typage) ; D ∩ C = `match-replay/MatchEquipmentUsageSection.tsx` (les deux le modifient :
résoudre dans le sens des DEUX intentions). Ordre A->B->C->D confirmé.

### IMP-7 — `make gate-push` n'est pas un filet complet
Il ne lance ni `go test`, ni vitest, ni build, ni `openapi-check` ; et son ratchet lint compare à
`origin/main` (tout le delta v75 s'affiche : ne pas le lire comme une régression).

Remplace E.1 :
```
E.1 FILET COMPLET, pièce par pièce, un code de sortie consigné par ligne (EXIT_*=0 dans un log
persistant, jamais un pipe) : `make gate-push` · `gofmt -l .` vide · `go vet ./...` ·
`go build ./...` · `go test ./... -count=1` · `go test -tags=integration -p 1 ./internal/sync/...
-count=1` · `make openapi-check` · `npm run lint:fields && npm run test && npm run build` ·
`node tools/knip-ratchet.mjs` · harnais 13/13 identiques SANS -update.
```

## MINEUR (à fondre dans le même passage de plume)
- MIN-1 D5 confirmé (aucun appelant, imports `sort`/`strconv`/`objectiveevents` seulement) ;
  nuance N-1 : la branche modifie aussi `bombe_portage_gate_test.go` (12 l.) — son merge ultérieur
  ne sera pas strictement additif.
- MIN-2 D6 confirmé ; corriger « deux lecteurs web » en « quatre sites, dont un lecteur métier
  `replayLogic.ts:274` » (+ `replayNormalize.ts:131/250/407`).
- MIN-3 D9 : les deux têtes véhicules sont DISJOINTES hors `thought_log.md` (base commune
  `f4c3ed417`) ; ordre indifférent pour le contenu ; `vehicules-sons` d'abord parce que ses 4
  commits réparent le lint `unused` et deux gardes archlint (état intermédiaire vert).
- MIN-4 §1 : `wt/vehicule-deadstate` est 25/339 (4 DERRIÈRE `sons` + 1 devant), pas « 28 + 1 ».
- MIN-5 D8ter : le 39 périme tous les artefacts (`cmd_backfill_replay.go:79`) MAIS aucune
  re-cuisson ne part seule — le rattrapage sélectionne par `os.Stat` (présence du fichier), pas
  par schéma (`replayartifacts/backlog.go`). État servi après le push : corpus MIXTE 38/39, calque
  véhicules absent sur l'historique ; le front lit `schemaVersion` comme un nombre
  (`replaySchemaLogic.ts`) — dégradation gracieuse. ASSUMÉ jusqu'à l'accord sur
  `levelup backfill-replay --only-existing`.
- MIN-6 C.4 : le diff `document_pickups.go`/`pad_pickup_dating.go` est un renommage pur
  (`padFamilyKey` -> `PadWeaponFamilyKey`, corps identique) : à l'étape C, TOUT écart au harnais
  est une anomalie, pas un « écart à nommer ». Gate ferme.
- MIN-7 C.2bis : statuer si `match_usage_players`/`match_usage_films` doivent entrer dans
  `internal/migration/append_only_rebuild.go` (la branche ne le fait pas) — l'ajouter, ou écrire
  pourquoi non au §4.
- MIN-8 seuils : `usageLogic.ts` 486 L, `SessionUsageSection.tsx` 417 L — passent ; une résolution
  de conflit qui ajoute des lignes franchit 500 : noter au §4, ne pas « réparer » hors périmètre.
- MIN-9 §7 contrat d'exécution opposable : statuts `[x]`/`[~]`/`[!]`, aucune case vide à la
  clôture, ordre strict, zéro fix hors périmètre ; REPRISE DE SESSION : lire §5 puis §6, la
  première case non cochée de la première étape non close est le point de reprise.
- MIN-10 F.4 : statuer le sort des branches absorbées (`origin/feat/v75-vehicules-sons` est
  publiée) et de `wt/cuisson-perf` (poussée comme trace ou non) — une ligne.

## Vérifié BON (ne pas re-litiger)
Schémas 38/31/37 ; avances 65/22/15/34 ; les 3 balayages amont neufs ; 8 balayages véhicules
dont 5 en production ; D5 ; D6 ; C conforme ART (persister INSERT-only, `no_art_patterns_test`
durci avec justification datée, migration append-only + vues `_latest` dans le bon ordre) ;
corpus 13 films / 45 étapes / grammaire 2 ; aucune collision de nom ni de tag JSON entre les
champs véhicules (`vehicles`, `vehicleLabels`, `v`) et les champs 38 amont — la branche véhicules
porte trois blocs de doc de version (29, 30, 31) à fondre en un seul bloc 39.

## Journal de l addendum
- 2026-09-05 15 h — IMP-5(a) SATISFAIT : l utilisateur GELE `feat/v75` pour la duree de l integration (« C est gele, je touche plus a rien »). La re-mesure de l amont au debut de chaque etape (IMP-5 b) reste en vigueur comme controle, elle devrait rendre zero. Pilotage confirme : orchestrateur-verificateur ; agents Opus (conflits, balayages) ou Sonnet (taches bornees) selon la complexite.
