# LOT 1 — une lecture d'inventaire VIDE n'efface plus la fiche (2026-08-25)

> Branche `wt/inventaire-fiches`, aucun commit (relecture superviseur).
> Périmètre : la piste 2 de `MESURE_TROUS_INVENTAIRE_2026-08-24.md` — le symptôme utilisateur
> « la fiche de certains joueurs ne donne rien par moments ». Les pistes 1, 3, 4, 5 et 6 du
> rapport de mesure ne sont PAS traitées ici.
>
> Le lot est daté du 25 ; le nom du fichier garde le 24, celui du rapport de mesure dont il est
> la suite directe.

---

## 0. Ce que le lot ferme, en trois phrases

1. **L'interprétation « record sans arme = bipède mort » est désormais PROUVÉE**, et elle l'a été
   AVANT d'être encodée : 88,3 % des lectures vides tombent dans les 8 s qui suivent une mort de
   leur porteur, contre 1,1 % des lectures pleines soumises à la MÊME fenêtre — un rapport de 82x.
2. **Une lecture vide est publiée MARQUÉE** (`inventory[].empty`, deux valeurs : `dead` corroboré
   par le fil des morts, `unknown` sinon). Aucune information n'est retirée du document.
3. **La fiche ne disparaît plus.** Elle affiche la dernière lecture PLEINE du slot, et dit à côté
   pourquoi la lecture courante est vide.

---

## 1. Étape 1 — la preuve, et son témoin

### 1.1 Ce qui est mesuré

Pour chaque record d'image-clé, l'instant de la lecture est-il dans les `W` millisecondes qui
suivent une mort **du porteur du slot** ? L'identité vient du pont `slot -> XUID`
(`buildOwners`) ; le décalage entre l'horloge du match (fil des morts) et celle du film est celui
que `bestDeathOffset` résout déjà, pas une valeur recopiée.

**Le témoin est la pièce maîtresse** : la même fenêtre est appliquée aux lectures **PLEINES**.
Sans lui, un recouvrement élevé ne dirait rien — si un joueur meurt toutes les 20 s et que la
fenêtre en dure 10, la moitié de TOUS les records y tomberait par construction.

Outil : `apps/go-api/internal/analysis/replay/inventory_mort_recouvrement_test.go`.
Le test `TestInventaireRecordVideRecouvrementMorts` tourne sur le **fixture d'entrées déjà
versionné** (`golden_inputs_test.go` — zéro octet de film, donc jouable en CI) ;
`TestInventaireRecordVideCorpus` étend au cache de films et est **gaté par `INV_MORT_FILMS`**.

### 1.2 Film de vérité terrain `000d5950` — 184 records, 34 lectures vides

| fenêtre | vides | % | pleines (témoin) | % | rapport |
|---:|---:|---:|---:|---:|---:|
| 2 000 ms | 6 | 18,8 % | 1 | 0,7 % | 27,4x |
| 4 000 ms | 16 | 50,0 % | 1 | 0,7 % | 73,0x |
| 6 000 ms | 22 | 68,8 % | 1 | 0,7 % | 100,4x |
| **8 000 ms** | **30** | **93,8 %** | **1** | **0,7 %** | **136,9x** |
| 10 000 ms | 32 | 100,0 % | 4 | 2,7 % | 36,5x |
| 12 000 ms | 32 | 100,0 % | 17 | 11,6 % | 8,6x |
| 15 000 ms | 32 | 100,0 % | 26 | 17,8 % | 5,6x |
| 20 000 ms | 32 | 100,0 % | 45 | 30,8 % | 3,2x |

Dénominateur : 32 lectures vides **attribuées** (2 des 34 ont un slot sans pont) et 146 lectures
pleines attribuées sur 150.

### 1.3 Corpus de 8 films — 1 419 records, 247 lectures vides

`000d5950, 121be2d6, 34dac77d, 66aa2908, dd3095de, f1e41f31, 0873c469, 4013dc34` (les six premiers
« avec ancre », les deux derniers du régime « sans ancre » du rapport de mesure — le défaut aval
ne dépend pas de ce régime).

| fenêtre | vides | % | pleines (témoin) | % | rapport |
|---:|---:|---:|---:|---:|---:|
| 2 000 ms | 55 | 23,0 % | 7 | 0,6 % | 36,7x |
| 4 000 ms | 109 | 45,6 % | 8 | 0,7 % | 63,7x |
| 6 000 ms | 158 | 66,1 % | 9 | 0,8 % | 82,0x |
| **8 000 ms** | **211** | **88,3 %** | **12** | **1,1 %** | **82,2x** |
| 10 000 ms | 229 | 95,8 % | 81 | 7,3 % | 13,2x |
| 12 000 ms | 231 | 96,7 % | 146 | 13,1 % | 7,4x |
| 15 000 ms | 235 | 98,3 % | 247 | 22,1 % | 4,4x |
| 20 000 ms | 235 | 98,3 % | 379 | 33,9 % | 2,9x |

Dénominateurs : 239 lectures vides attribuées sur 247, 1 117 pleines sur 1 172.

### 1.4 Le verdict, et le choix de la fenêtre

**Recouvrement retenu : 88,3 % à 8 s**, très au-dessus du seuil de 80 % fixé au mandat.
**L'étiquette « mort » est justifiée.**

La fenêtre de 8 s n'est pas choisie, elle est mesurée **deux fois** :
- c'est le **point de séparation maximale** du balayage — au-delà, le témoin s'envole (0,8 % à
  6 s, 1,1 % à 8 s, puis **7,3 % à 10 s** et 13,1 % à 12 s) : la fenêtre se met à attraper des
  joueurs **réapparus**, donc vivants ;
- c'est la **médiane de réapparition** déjà relevée par `lives.go` (commentaire de `lifeGapUS`).

Les 11,7 % restantes gardent `unknown` : le décodeur n'a rien lu et personne ne sait pourquoi.
Les étiqueter « mort » affirmerait à l'écran ce qu'aucune pièce n'établit.

### 1.5 Ce que la mesure a corrigé au passage

Le prédicat « lecture vide » porte sur **deux** drapeaux (`GrenadesRead`, `AmmoRead`), pas quatre.
Une première écriture y ajoutait `DrawnSlot` et `SelectedGrenadeRank` et trouvait **0** lecture
vide sur 184 : ces deux champs ne portent aucun contenu par eux-mêmes (le sélecteur désigne une
arme parmi les munitions, le rang sélectionné un type parmi les compteurs), et le fixture ne
sérialise même pas le second (cf. §5, découverte 1). Distribution vérifiée sur le film de
référence : 120 records avec grenades, 150 avec munitions, 150 avec sélecteur (les mêmes), **34
avec ni grenade ni munition** — exactement les 34 « fiches vides » du rapport de mesure.

---

## 2. Étape 2 — le contrat retenu

### 2.1 Le champ

```go
// Inventory (inventory.go)
Empty string `json:"empty,omitempty"`

const (
    InventoryEmptyDead    = "dead"
    InventoryEmptyUnknown = "unknown"
)
```

**Un champ additif à deux valeurs, pas un booléen.** Un `Empty bool` aurait dit « c'est vide » —
ce que le client déduit déjà de l'absence de `g` et de `am`. Ce qui manquait, c'est le **pourquoi**.
La forme « chaîne à valeurs closes » est celle que le paquet emploie déjà pour
`EquipmentPlacement.origin` (`deployed` / `dropped` / `unknown`) ; aucune balise `enum:` n'est
posée, aucun autre schéma du paquet n'en porte.

**Absent = la lecture porte quelque chose.** `omitempty` est ici sans piège : la chaîne vide n'est
pas une valeur du domaine.

### 2.2 SchemaVersion 18 -> 19

**Oui, un champ optionnel additif justifie la montée** — et c'est exactement la doctrine des
montées v13 (`Point.p`) et v18 (`zoneStates[].gauge`), toutes deux des sous-champs optionnels :
la version monte quand un artefact ancien **AFFIRME quelque chose de faux**.

Ici, un artefact v18 publie `{"t":N,"slot":S}` nu, le client retient cette lecture comme la plus
récente `<= T`, et **efface la fiche pendant ~20 s**. Le marqueur ne peut pas y être ajouté après
coup : il faut le fil des morts, donc le film. Un v18 se lit donc « à re-cuire », pas « à jour »,
et la reprise du backfill se fait par SchemaVersion (`cmd_backfill_replay.go`).

La raison est écrite aux deux endroits que le dépôt impose : la chronique de `document.go` et le
ratchet `TestStructureIsOptionalInDocument` (`structure_test.go`), qui refuse toute montée sans
justification écrite.

### 2.3 Où le croisement se fait

- `buildInventory` (`inventory.go`) — le **décodeur** — marque `unknown` : c'est tout ce qu'il
  sait. Il ne reçoit PAS le fil des morts.
- `markInventoryDeadReadings` (`inventory_dead_readings.go`) — appelé depuis `build.go`, où
  `opt.Deaths`, `own.SlotXUID` et `own.DeathOffsetMS` existent déjà — requalifie en `dead`.

L'instant se reconstruit depuis la grille (`origin + T*step`) : l'arrondi vaut un pas de 100 ms
contre une fenêtre de 8 000, sans portée.

**Sans fil des morts, rien ne bouge** : tout garde `unknown`. On ne requalifie jamais par défaut.

---

## 3. Le comportement côté web

`inventoryAt` (déplacé dans `inventoryReading.ts`) rend désormais :

```ts
{ state,          // la dernière lecture PLEINE du slot quand la lecture courante est vide
  age,            // l'âge de CETTE lecture-là
  empty?: { kind: 'dead' | 'unknown', age } }   // l'âge de la lecture VIDE, distinct
```

- La ligne **n'est plus supprimée** : le `return null` de `ReplayInventoryRow` ne s'applique plus
  quand un état vide existe.
- L'équipement affiché est la dernière lecture pleine ; l'état vide s'affiche **à côté**, avec sa
  propre infobulle qui porte l'âge de l'équipement montré (les deux lectures ne datent pas du même
  instant).
- `dead` -> « Mort » / « Dead », encré au token `destructive`. `unknown` -> « Inventaire
  indisponible » / « Inventory unavailable », en soulignement pointillé neutre — la même grammaire
  que `sél. ?` et `dégainée ?`.
- Une étiquette **inconnue** d'un artefact futur retombe sur `unknown`, jamais sur « Mort ».
- Le repli ne franchit jamais une vie : la recherche de la lecture pleine porte sur le SLOT et
  s'arrête strictement avant la lecture vide.

---

## 4. Fichiers modifiés

### Go — `apps/go-api/`

| fichier | nature |
|---|---|
| `internal/analysis/replay/inventory.go` | champ `Empty`, constantes, `invDeadWindowMS`, prédicat `invReadingIsEmpty`, marquage dans `buildInventory` |
| `internal/analysis/replay/inventory_dead_readings.go` | **neuf** — croisement avec le fil des morts + journalisation de couverture |
| `internal/analysis/replay/build.go` | appel du croisement à l'assemblage |
| `internal/analysis/replay/document.go` | `SchemaVersion` 18 -> 19 + chronique v19 |
| `internal/analysis/replay/inventory_mort_recouvrement_test.go` | **neuf** — la mesure de l'étape 1 (fixture + corpus gaté `INV_MORT_FILMS`) |
| `internal/analysis/replay/inventory_dead_readings_test.go` | **neuf** — la règle de marquage (fenêtre, ses deux bords, ce qu'elle ne touche pas) |
| `internal/analysis/replay/structure_test.go` | ratchet de version : justification v18 -> v19 |
| `internal/analysis/replay/golden_assembly_test.go` | le golden rend le décompte des lectures vides |
| `internal/analysis/replay/testdata/assembly_000d5950.golden` | régénéré (`-run GoldenAssembly -update`) |
| `api/openapi.yaml` | régénéré (`go run ./cmd/openapi-gen`) — 2 lignes |

### Web — `apps/web/src/`

| fichier | nature |
|---|---|
| `features/match-replay/inventoryReading.ts` | **neuf** — `inventoryAt` avec l'état vide, `grenadesCarried`, `selectedGrenade` (extraits de `rosterLogic.ts`, seuil de taille) |
| `features/match-replay/inventoryReading.test.ts` | **neuf** — cas déplacés + la nouvelle règle de sélection |
| `features/match-replay/rosterLogic.ts` | bloc inventaire retiré (474 -> 415 L), imports nettoyés |
| `features/match-replay/rosterLogic.test.ts` | cas déplacés, imports mis à jour |
| `features/match-replay/ReplayInventoryRow.tsx` | plus de `return null` sur lecture vide, `InventoryEmptyMark` |
| `features/match-replay/ReplayTeams.test.tsx` | 3 cas de rendu de la lecture vide |
| `features/match-replay/equippedLogic.ts` | import de `inventoryAt` déplacé |
| `features/match-replay/i18n.ts` | 4 chaînes, FR **et** EN |
| `features/match-replay/i18nContract.ts` | les 4 clés au contrat typé |
| `lib/api/generated.ts` | régénéré (`npm run generate-types`) — 1 ligne |

---

## 5. Découvertes hors périmètre — NON traitées

1. **Le fixture d'entrées ne sérialise pas `KeyframeInventory.SelectedGrenadeRank`.**
   `golden_inputs_test.go` encode `GrenadesRead`, `Grenades`, `AbilityRank`, `DrawnSlot`,
   `AmmoCandidates`, `AmmoRead` et `Ammo` — jamais le rang sélectionné. Il se relit donc à `0`
   pour les 184 records, et le golden verrouille un document dont `Inventory.Gs` vaut 0 partout,
   ce qui n'est pas ce que la production sert. La production n'est pas affectée (le fixture ne
   nourrit que le golden), mais le golden ment sur ce champ, et le format devrait monter d'une
   version (`REPLAYINPUTS9` -> `10`).

2. **Deux affirmations sur le mot « slot » qui ne peuvent pas être vraies au même sens.**
   `rosterLogic.ts` et `lives.go` écrivent « le slot est réattribué à chaque réapparition » (c'est
   ce qui rend le report de lecture sûr) ; or la mesure trouve des lectures vides d'un slot DANS
   la fenêtre de mort du porteur de CE slot, et `ownersFromLives` mesure 0 collision de porteur sur
   90 vies. À instruire — la doctrine du report de lecture s'appuie dessus.

3. **Deux lectures vides sur 34 (film de référence) portent un slot sans pont `slot -> XUID`.**
   Elles ne pourront jamais être qualifiées, quelle que soit la fenêtre. C'est le plancher de
   `unknown`.

---

## 6. État des tests — sorties réelles

```
go test ./internal/analysis/replay/
ok      levelup/go-api/internal/analysis/replay 17.097s

go test ./internal/analysis/... ./contracttest/... ./internal/replaybuild/...
18 paquets « ok », 0 FAIL

make check-types
cd apps/web && npm run typecheck / tsc -b        exit 0, aucune sortie

make test-web
Test Files  470 passed (470)
     Tests  4496 passed | 14 skipped (4510)

npm run lint
20 problems (0 errors, 20 warnings)             baseline inchangee
```

Journal de production sur le film de référence, après le lot :

```
INFO rejeu : lectures d'inventaire vides lectures=184 vides=34 morts=31 inexpliquees=3
```

31/34 requalifiées (91,2 %). L'écart avec les 93,8 % de la mesure vient de la reconstruction de
l'instant depuis la grille de 100 ms et des 2 slots sans pont.

---

## 7. Reproduire la mesure

```bash
# le fixture versionne — aucun octet de film, joue en CI
cd apps/go-api && CGO_ENABLED=0 \
  go test ./internal/analysis/replay/ -run RecouvrementMorts -v

# le corpus, gate par variable d'environnement (I/O disque, ~6 min pour 8 films)
cd apps/go-api && CGO_ENABLED=0 \
  INV_MORT_FILMS=<repo>/data/cache/film_chunks/000d5950,<repo>/data/cache/film_chunks/121be2d6 \
  go test ./internal/analysis/replay/ -run RecordVideCorpus -v -timeout 120m
```

`INV_MORT_FILMS` accepte une liste séparée par des virgules. Le balayage lit les positions en
`QuantaOnly` : la mesure ne consomme d'une position que son slot et son horodatage, exiger les
bornes de carte bornerait le corpus aux seules cartes du catalogue sans rien ajouter.

---

## 8. Ce qui reste ouvert

Le symptôme utilisateur est fermé. Le plus gros volume du rapport de mesure ne l'est pas :
**piste 1** (découpler R2 de R1 — 4 278 records, 63,7 %, retrouveraient leur ligne de grenades) et
**piste 3** (R2 : accepter la somme nulle — 104 records, la seule cause entièrement close du
rapport). Un **re-backfill des artefacts est nécessaire** (SchemaVersion 19).
