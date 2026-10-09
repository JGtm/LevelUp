# LOT 2 — télémétrie de couverture inventaire + garde SchemaVersion côté web (2026-08-25)

> Branche `wt/inventaire-fiches`, aucun commit (relecture superviseur).
> Suite directe du lot 1 (`.ai/V7.5/replay2d/LOT1_LECTURE_VIDE_2026-08-24.md`) sur les manques
> restants de `.ai/V7.5/replay2d/AUDIT_AVAL_INVENTAIRE_2026-08-24.md` : point 3 (erreurs de
> chunk avalées sans télémétrie) et point 5 (inventaire sans entrée de couverture) côté Go ;
> point 1 (absence de garde SchemaVersion, le plus grave de l'audit) côté web.
>
> HORS PÉRIMÈTRE, par consigne explicite : la correction du rejet des lectures antérieures à
> l'origine (point 2 — sa MESURE, elle, entre dans la couverture ajoutée) et le fixture
> `SelectedGrenadeRank` (découverte du lot 1).

---

## 0. Ce que le lot ferme, en deux phrases

1. **Le décodeur d'inventaire raconte désormais sa propre lecture** : chunks lus/illisibles,
   images-clés parcourues, records de biped vus — plus une couverture `Coverage.Inventory`
   publiée dans le document, symétrique de `Shots`/`Grenades`/`Equipment`/`Grapple`.
2. **Un artefact d'une version de schéma différente de celle que ce client sait exploiter
   l'affiche désormais** : une note discrète, jamais un blocage.

---

## 1. Volet A — télémétrie Go

### 1.1 `KeyframeInventoryStats` (inventory_decode.go / inventory.go)

Avant ce lot, `ScanFilmKeyframeInventory` avalait toute erreur de lecture de chunk par un
`continue` nu (`inventory_decode.go:195-200` avant le lot), sans compteur ni log — contrairement
aux scanners frères (`ScanFilmAbilityRanks`, `ScanFilmCamoStates`, `ScanFilmGrappleReads`) qui
remontent chacun une `Stats` détaillée, loguée en détail par `build.go`.

`KeyframeInventoryStats` comble l'écart, même vocabulaire :

```go
type KeyframeInventoryStats struct {
    Chunks       int // total (CountFilmChunks)
    ChunksUnread int // lectures disque échouées
    Keyframes    int // paquets d'image-clé parcourus
    Records      int // records de biped vus == lectures rendues (aucun filtrage ici)
}
```

`Records` porte deux noms pour une seule grandeur : `keyframeInventories` ne filtre AUCUN
record de biped (chaque span produit une lecture, lue ou non) — l'écrire en double aurait été
une redondance, pas une mesure de plus.

**Où vit le type, et pourquoi ce n'est pas dans `inventory_decode.go`** : ce fichier touchait
exactement 500 lignes avec la struct en plus (seuil du dépôt, CLAUDE.md n°5). `Stats` a migré
dans `inventory.go`, à côté d'`InventoryCoverage` qu'elle alimente — `inventory_decode.go`
reste le fichier du décodage bit à bit, `inventory.go` celui de la projection et de sa
télémétrie.

**Test du cas dégradé** (`inventory_decode_test.go`, neuf) : un chunk illisible est simulé par
un **répertoire** nommé `chunk_NN.bin` — `os.Stat` (que `CountFilmChunks` utilise) le voit,
`os.ReadFile` (que `ReadFilmChunk` utilise) échoue dessus sur toutes les plateformes. C'est le
geste le plus portable pour ce test, sans dépendre de permissions fichier (fragiles sous
Windows). Deux cas couverts : un chunk illisible parmi d'autres lisibles (le balayage réussit,
`ChunksUnread` le dit) et la totalité illisible (l'erreur remonte, comme avant).

### 1.2 `Coverage.Inventory` (coverage.go / inventory.go / build.go)

`InventoryCoverage` (nouveau) publie, symétrique de `LayerCoverage` pour les autres calques :

```go
type InventoryCoverage struct {
    Decoded             int // lectures produites par le décodeur, avant tout filtrage
    DroppedBeforeOrigin int // écartées : horodatage antérieur à l'origine du rejeu
    Unpublished         int // écartées : slot sans trajectoire publiée
    Published           int // publiées dans le document
}
```

`buildInventory` (inventory.go) rend désormais aussi le compte des lectures écartées avant
l'origine — **seule la MESURE est ajoutée, le comportement du filtre reste inchangé** (le point
2 de l'audit, sa correction, est explicitement hors périmètre de ce lot). `buildInventoryCoverage`
assemble la structure depuis les trois étapes (décodé -> construit -> publié), et vit dans
`inventory.go` plutôt que `build.go` (empreinte minimale dans un fichier déjà à la limite, cf.
§3).

**Aucun bump de SchemaVersion.** Vérifié précédent par précédent dans la chronique de
`document.go` (18 montées, chacune motivée) avant de trancher : toutes les montées passées
avaient une raison de RE-CUISSON — un champ que le client CONSOMME et qu'un vieil artefact ne
porte pas (v3 à v18, systématiquement « clé de reprise du backfill »). `Coverage.Inventory` est
une télémétrie PURE : aucun rendu client n'en dépend, un artefact qui ne la porte pas n'affirme
rien de faux. C'est exactement la même situation que `Structure`/`StructureBounds`
(`TestStructureIsOptionalInDocument`, qui existe précisément pour verrouiller « un champ
optionnel de plus n'incrémente pas la version »).

### 1.3 Mesure sur le film de référence

```
INFO rejeu : inventaire : lectures de keyframe chunks=<N> chunksIllisibles=0 imagesCles=<N> records=184
INFO rejeu : couverture inventaire decodees=184 ecarteesAvantOrigine=0 ecarteesSansPiste=0 publiees=184
```

**0 perte sur ce film précis**, aux trois filtres mesurés ici. La perte que l'audit chiffre
(17,4 % de lectures publiées vides) est ailleurs dans la chaîne — fermée par le lot 1
(marquage `dead`/`unknown`), pas par ce lot.

---

## 2. Volet B — garde SchemaVersion côté web

### 2.1 Le constat de l'audit (point 1, le plus grave)

`replayNormalize.ts` ne lit jamais `schemaVersion`. Un artefact construit avant un bump de
schéma sert une fiche amputée, indiscernable de « rien à afficher » — la reprise du backfill
est une commande manuelle d'opérateur (`cmd_backfill_replay.go`), et rien à l'écran ne signale
qu'elle n'a pas encore tourné pour un match donné.

### 2.2 La source de vérité côté client

Le contrat généré (`lib/api/generated.ts`) type `schemaVersion` en `number` — il ne porte
**aucune valeur littérale**, la valeur variant précisément d'un artefact à l'autre (c'est le
point du champ). La source de vérité reste donc la constante Go `replay.SchemaVersion`
(document.go, valeur 19 depuis le lot 1).

**Choix retenu** : une copie locale documentée, `EXPECTED_REPLAY_SCHEMA_VERSION`
(`replaySchemaLogic.ts`), gardée en PARITÉ par un test dédié
(`replaySchemaLogic.guard.test.ts`) qui lit `document.go` par `readFileSync` et compare — même
patron que `placementFamily.guard.test.ts`, déjà en place dans la feature. Une génération
dédiée pour un seul entier aurait été disproportionnée ; une constante non gardée aurait dérivé
en silence au prochain bump de schéma.

### 2.3 Le comportement

`replaySchemaState(schemaVersion)` rend `'current' | 'stale' | 'ahead'` :
- `stale` : artefact ANTÉRIEUR (cas de l'audit — backfill non rejoué) ;
- `ahead` : artefact POSTÉRIEUR (ce déploiement du client est en retard sur le format servi) ;
- `current` : rien à dire.

Dans la route (`replay.tsx`), une note DISCRÈTE (`text-xs text-muted-foreground`, même registre
que les notes de bas de canvas) s'affiche sous l'en-tête quand l'état n'est pas `current` —
**jamais de blocage** : la fiche continue d'afficher tout ce que l'artefact porte. Deux
chaînes, FR et EN, typées dans `i18nContract.ts` :

- FR (`stale`) : « Données de rejeu d'une version antérieure — certains éléments peuvent
  manquer. »
- FR (`ahead`) : « Cette page est plus ancienne que le format de ces données de rejeu —
  certains éléments peuvent manquer. »

---

## 3. Fichiers modifiés

### Go — `apps/go-api/`

| fichier | nature |
|---|---|
| `internal/analysis/replay/inventory_decode.go` | `ScanFilmKeyframeInventory` compte chunks/keyframes/records ; simplifié pour tenir sous 500 L |
| `internal/analysis/replay/inventory.go` | `KeyframeInventoryStats`, `InventoryCoverage`, `buildInventoryCoverage` ; `buildInventory` rend aussi le compte pré-origine |
| `internal/analysis/replay/coverage.go` | `Coverage.Inventory *InventoryCoverage` |
| `internal/analysis/replay/build.go` | branchement + log de couverture inventaire |
| `internal/analysis/replay/inventory_decode_test.go` | **neuf** — chunk illisible simulé (répertoire), cas partiel et total |
| `internal/analysis/replay/inventory_test.go` | mis à jour (buildInventory rend 2 valeurs) |
| `internal/analysis/replay/inventory_dead_readings_test.go` | mis à jour (idem) |
| `internal/analysis/replay/minifilm_test.go` | assertions sur `KeyframeInventoryStats` |
| `internal/analysis/replay/golden_inputs_test.go`, `inventory_mort_recouvrement_test.go`, `inventory_rules_test.go` | signature à 3 retours |
| `contracttest/replay_contract_test.go` | `InventoryCoverage` ajoutée au registre `replaySchemas` |
| `api/openapi.yaml` | régénéré (`go run ./cmd/openapi-gen`) — +25 lignes additives |

### Web — `apps/web/src/`

| fichier | nature |
|---|---|
| `features/match-replay/replaySchemaLogic.ts` | **neuf** — `EXPECTED_REPLAY_SCHEMA_VERSION`, `replaySchemaState` |
| `features/match-replay/replaySchemaLogic.test.ts` | **neuf** — les trois états |
| `features/match-replay/replaySchemaLogic.guard.test.ts` | **neuf** — parité avec `replay.SchemaVersion` (lecture de document.go) |
| `features/match-replay/i18n.ts` | 2 chaînes, FR et EN |
| `features/match-replay/i18nContract.ts` | les 2 clés au contrat typé |
| `routes/{-$lang}/.../replay.tsx` | note discrète sous l'en-tête, jamais de blocage |
| `lib/api/generated.ts` | régénéré (`npm run generate-types`) — +12 lignes |

---

## 4. État des tests — sorties réelles

```
cd apps/go-api && go test -count=1 ./internal/analysis/replay/ ./internal/replaybuild/... ./contracttest/...
ok      levelup/go-api/internal/analysis/replay        16.901s
ok      levelup/go-api/internal/analysis/replay/mapvar  0.264s
ok      levelup/go-api/internal/replaybuild             0.514s
ok      levelup/go-api/contracttest                     0.419s

make go-api-lint
golangci-lint : 0 issues

make check-types
cd apps/web && npm run typecheck / tsc -b        exit 0, aucune sortie

make test-web
Test Files  472 passed (472)        (+2 vs lot 1 : 470 -> 472)
     Tests  4500 passed | 14 skipped (4514)   (+4 vs lot 1 : 4496 -> 4500)

npm run lint
20 problems (0 errors, 20 warnings)             baseline inchangee (identique au lot 1)
```

Journal de production sur le film de référence, après le lot :

```
INFO rejeu : couverture inventaire decodees=184 ecarteesAvantOrigine=0 ecarteesSansPiste=0 publiees=184
```

---

## 5. Découvertes hors périmètre — NON traitées

1. **`build.go` dépasse le seuil de 500 lignes** (620 L après ce lot, contre 607 avant — dette
   déjà gelée que l'ajout d'un cinquième calque de couverture aggrave de 13 L nets malgré
   l'extraction de `buildInventoryCoverage` vers `inventory.go`). Une extraction structurelle de
   `BuildFromPositions` en sous-fonctions par calque réglerait la cause ; c'est un chantier à
   part entière, hors périmètre.
2. Les découvertes du lot 1 restent ouvertes, inchangées par ce lot : la tension de doctrine sur
   le mot « slot » (rosterLogic.ts/lives.go affirment une réattribution à chaque réapparition,
   la mesure du lot 1 trouve des collisions dans la fenêtre de mort) et les 2 lectures sans pont
   slot->joueur du film de référence.

---

## 6. Ce qui reste ouvert

Le point 2 de l'audit (rejet des lectures antérieures à l'origine) N'EST PAS corrigé — seule sa
MESURE est désormais publiée (`Coverage.Inventory.DroppedBeforeOrigin`, 0 sur le film de
référence). Le point 4 (abandon en bloc, probablement inatteignable) et le point 6 (ligne
d'inventaire invisible sans lecture dans la fenêtre de vie, comportement voulu mais muet) de
l'audit ne sont pas non plus traités par ce lot.

---

## 7. Corrections post-revue (2026-08-25)

Le diff complet du chantier (lots 1 + 2) a été relu par une revue adversariale en contexte frais :
**8 constats recevables, 6 corrigés ici, 2 consignés** (ce sont les deux découvertes hors
périmètre du §5, inchangées). Chaque correction porte son test de non-régression.

### 1. `Coverage.Inventory` sans témoin (P1)

Muter `Published` ou `DroppedBeforeOrigin` ne faisait tomber aucun test, et le golden
d'assemblage ne rendait pas le bloc. Deux témoins posés, sur le modèle de `Placements` et
`GroundWeapons` :

- `apps/go-api/internal/analysis/replay/golden_assembly_test.go:433-443` (`renderInventory`) —
  ligne de couverture chiffrée, plus une branche « aucune couverture d inventaire (rien n a ete
  fourni a lire) » ;
- `apps/go-api/internal/analysis/replay/testdata/assembly_000d5950.golden:53` — figé :
  `couverture : 184 lecture(s) decodee(s) -> 0 ecartee(s) avant l origine du rejeu · 0
  ecartee(s) faute de piste publiee -> 184 publiee(s)` ;
- `apps/go-api/internal/analysis/replay/inventory_test.go:211` —
  **`TestInventoryCoverageBalances`** : l'invariant `Decoded == DroppedBeforeOrigin +
  Unpublished + Published` sur des comptes NON TRIVIAUX (6 = 2 + 1 + 3), le film de référence
  n'ayant aucun rejet.

Vérification par mutation manuelle (temporaire, remise en état) : `DroppedBeforeOrigin: 0` et
`Published: len(built)` font tomber `TestInventoryCoverageBalances` ; `Decoded: len(decoded)-1`
fait tomber `TestGoldenAssembly` (premier écart ligne 53).

### 2. Affectation non gardée (P1)

`apps/go-api/internal/analysis/replay/build.go:478` — l'affectation inconditionnelle est
remplacée par `attachInventoryCoverage(&doc, ...)`, défini avec le type qu'il publie
(`apps/go-api/internal/analysis/replay/inventory.go:296`) et gardé sur `decoded == nil ||
doc.Coverage == nil`, comme `attachFlagLayer` (`build_objectives_live.go`) et `attachZoneStates`
(`build_zones.go`). Un inventaire illisible ne publie plus `{0,0,0,0}` — il ne publie RIEN, ce
que la doctrine de `coverage.go` réserve à « l'appelant n'a rien fourni à lire ».

Le type était DÉJÀ un pointeur (`Coverage.Inventory *InventoryCoverage`, `json:",omitempty"`) et
`inventory` n'est pas dans le `required` du schéma `Coverage` : **aucune régénération
`openapi.yaml` / `generated.ts` nécessaire**, `contracttest` inchangé et vert.

Test : `apps/go-api/internal/analysis/replay/inventory_test.go:242` —
**`TestInventoryCoverageAbsentWhenNothingToRead`** (nil -> ABSENTE ; tranche vide mais NON NULLE
-> présente à quatre zéros). Mutation de contrôle : retirer `decoded == nil` de la garde le fait
tomber.

### 3. Arme « en main » pour un joueur mort (P1)

`apps/web/src/features/match-replay/equippedLogic.ts:66-73` — quand la lecture qui couvre l'image
est vide, `equippedWeapons` reprenait le sélecteur `d` de la lecture PLEINE substituée par
`inventoryAt` et rendait `drawn: 0 / inHand: true` : la rangée d'armes affirmait une arme
dégainée pour un joueur que l'artefact déclare mort. Le sélecteur est désormais tenu pour NON LU
(`drawn: null`, `drawnUnread: true`, `holstered: false` — une lecture vide n'est pas la MESURE
`D=2`). La ligne d'INVENTAIRE, elle, garde la dernière lecture pleine : le contrat du lot 1 n'est
pas touché.

Tests : `apps/web/src/features/match-replay/equippedLogic.test.ts:100` et `:126` (aucun cas
`empty` n'y existait).

### 4. Infobulle mensongère (P1)

`apps/web/src/features/match-replay/ReplayInventoryRow.tsx:120` — quand
AUCUNE lecture pleine antérieure n'existe, la fiche affirmait « l'équipement affiché est la
dernière lecture pleine, lue il y a X » avec l'âge de la lecture VIDE. L'infobulle a maintenant
deux moitiés : le POURQUOI suivi de l'âge de la lecture vide, puis SOIT le report réel
(`inventoryFallbackHint` + âge de l'équipement) SOIT `inventoryNoPriorHint`. Le discriminant est
le nouveau `InventoryReading.substituted`
(`apps/web/src/features/match-replay/inventoryReading.ts:63`, `:98-102`).

La composition du texte vit dans `apps/web/src/features/match-replay/inventoryReading.ts:135`
(`inventoryEmptyHint`) et non dans le composant : c'est une composition PURE, donc testable sans
rendu — et la garder dans `ReplayInventoryRow.tsx` faisait passer ce fichier à 503 L, au-dessus
du seuil de 500 (CLAUDE.md n°5). Le composant l'appelle en
`apps/web/src/features/match-replay/ReplayInventoryRow.tsx:120`.

Textes FR/EN : `apps/web/src/features/match-replay/i18n.ts:238-239` et `:462-463` ; contrat de
parité `apps/web/src/features/match-replay/i18nContract.ts:347-348` (doc du bloc réécrite,
`:329-346`).

Tests : `apps/web/src/features/match-replay/ReplayTeams.test.tsx:933` (report réel — l'infobulle
porte DEUX durées différentes) et `:954` (aucun report — une seule durée, aucune promesse de
lecture pleine) ; plus `inventoryReading.test.ts` sur `substituted`.

### 5. Champ mort documenté comme indispensable (P1)

`InventoryEmptyState.age` (`apps/web/src/features/match-replay/inventoryReading.ts:35-43`) était
documenté sur quatre lignes et lu par personne.

**Option retenue : l'IMPLÉMENTER, pas le supprimer.** La distinction sert directement
l'utilisateur — « mort il y a 2 s, équipement lu il y a 20 s » et « mort il y a 20 s » ne
décrivent pas la même situation, et c'est exactement ce que la correction 4 avait besoin de
dire. Le badge d'état vide (Mort / Inventaire indisponible) emploie donc `empty.age` ; la partie
équipement garde `read.age`. Supprimer le champ aurait obligé à réintroduire la même grandeur
pour rendre l'infobulle honnête.

Tests : `ReplayTeams.test.tsx:933` verrouille que les deux âges sont DIFFÉRENTS dans la même
infobulle, `:954` qu'il n'y en a qu'un quand il n'y a rien à dater.

### 6. Mesure fondatrice dégradable en silence (P1)

`apps/go-api/internal/analysis/replay/inventory_mort_recouvrement_test.go:236` — la branche
d'erreur de `ScanFilmFireEvents` était la seule muette des cinq (`fire = nil` sans `t.Logf`) :
sans tirs, `buildOwners` rend un pont slot->joueur plus pauvre, `attribues` baisse, et le
DÉNOMINATEUR du taux change sans qu'aucune ligne du rapport ne l'explique. `t.Logf` ajouté, comme
ses voisines.

`:170-200` — `TestInventaireRecordVideCorpus` ne portait AUCUNE assertion, que des `t.Log`. Il
conclut désormais : `signal >= 75 %` et `temoin <= 10 %` (constantes `corpusSignalMin` /
`corpusTemoinMax`, `:200-203`), plus deux `t.Fatal` de sujet (corpus sans lecture vide, corpus
sans lecture vide attribuée). Choix des seuils documenté en commentaire : le corpus du lot 1
(8 films, 1 419 records) donne 88,3 % / 1,1 %, le fixture 93,8 % / 0,7 % ; un corpus mêle cartes
et modes, donc il varie plus qu'un film — le signal est desserré d'un cran sous le test de
vérité terrain (qui reste à 80), le témoin garde la même borne. Ce sont les deux bornes qui
protègent le RAPPORT (82x mesuré), pas un centième.

### Gates après corrections

```
go test ./internal/analysis/replay/ ./internal/replaybuild/... ./contracttest/...
ok  levelup/go-api/internal/analysis/replay   17.718s
ok  levelup/go-api/internal/replaybuild        0.481s
ok  levelup/go-api/contracttest                0.395s

make go-api-lint    golangci-lint run --new-from-merge-base=origin/main : 0 issues
make check-types    tsc -b : exit 0, aucune sortie
make test-web       472 fichiers passes · 4504 tests passes | 14 skipped   (+4 vs avant revue)
```

## 8. Ronde 2 de la revue — un P1 corrige, un P2 consigne (2026-08-25)

La ronde 2 (contexte frais, perimetre strict : les 6 corrections du §7) a rendu 1 P1, 1 P2,
18 conditions verifiees qui tiennent.

### P1 corrige — « Mort » affiche avant que la mort survienne

Quand la PREMIERE lecture d'un slot est vide, `nearestReading` retombe sur la lecture A VENIR
(age negatif) et le badge « Mort » s'affichait avec « lue il y a X » derriere `Math.abs` —
pendant que l'infobulle de la ligne disait « dans X s ». Mesure sur le film de reference :
8 vies sur 90 (~9 %) portaient « Mort » de 7,5 a 19,1 s AVANT la lecture, jusqu'a ~11 s sur un
joueur vivant.

Correctif : `inventoryAt` (`apps/web/src/features/match-replay/inventoryReading.ts`) — une
lecture vide d'age NEGATIF est rendue comme une lecture ordinaire « a venir » : ni `empty`, ni
substitution (le report `lastFullBefore` n'aurait cherche que parmi des lectures elles aussi
futures). La fiche se tait sur l'etat, comme avant le lot.

Temoins : `inventoryReading.test.ts` (« une lecture vide A VENIR n'affirme rien » + « ne
substitue pas un equipement pas encore ramasse ») et `ReplayTeams.test.tsx` (« une lecture vide
A VENIR n'affiche pas Mort »). Gates re-passes : vitest feature 1 020/1 020, `make test-web`
4 507 passes + 14 skipped, `tsc -b` propre. Aucun fichier Go touche.

### P2 consigne (dette, non corrige)

`attachInventoryCoverage` (`inventory.go`) : la garde `decoded == nil` ne distingue pas
« scan OK, zero lecture » de « scan en panne » — `ScanFilmKeyframeInventory` rend `nil` dans
les deux cas (var non allouee, sortie anticipee `len(known)==0`). Le cas `{0,0,0,0}` que
`TestInventoryCoverageAbsentWhenNothingToRead` verrouille n'est atteignable que par un appelant
passant explicitement une slice vide, jamais par la production. Telemetrie pure, aucun rendu
n'en depend. Piste si on veut la distinction : faire rendre au scanner une slice allouee vide
en cas de succes sans record.
