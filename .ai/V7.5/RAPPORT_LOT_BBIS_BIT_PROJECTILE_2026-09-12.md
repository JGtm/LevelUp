# Lot B-bis — le bit de trop peu de la porte d'i0 (2026-09-12)

> Branche `wt/bit-projectile`, base `wt/decodeur-fork` `b3f92f86d` (lot B). Question ouverte par
> le lot B : 947 trajectoires de projectile sur 15 735 portent un pas impossible, et le saut vaut
> l'étendue de la carte sur un axe **divisée par une puissance de deux**. Le lot B l'avait
> caractérisé sans l'expliquer. Ce lot le remonte jusqu'aux bits, le corrige, et le mesure.

## Verdict en une ligne

**Hypothèse (b) — largeur de champ fausse.** La porte d'`object-position-component` était écrite
**en dur à 3 bits** ; l'index de région qu'elle porte fait **DEUX bits sur Live Fire**. Les trois
axes y étaient lus **un bit trop tôt**, si bien que le bit de poids faible de chaque champ
devenait le bit de poids **fort** du suivant.

Ce n'est ni (a) un bit de signe, ni (c) des bornes de carte fausses, ni (d) un delta lu comme
absolu : les bornes et les largeurs d'axe du catalogue sont JUSTES, et c'est avec elles, à la
bonne porte, que les trajectoires redeviennent continues.

---

## BB.1 — De la trajectoire publiée aux bits

### L'instrument

`internal/analysis/filmdec/bit_projectile_research_test.go`, gardé par `BITPROJ_PARC` +
`BITPROJ_FILMS` (sans films, il se saute). Il rejoue le balayage des records d'objet du monde en
gardant, pour chaque échantillon, **la position de bit du début d'i0, les bits de porte et le
quantum brut de chaque axe**, et balaye le même film sous **deux largeurs de porte** : celle de
production (3 bits) et celle que le catalogue impose (`2 + regionIndexBits`).

### Le catalogue, d'abord

`map_quant_bounds.json` porte 79 cartes. **Une seule** déclare `regionIndexBits = 2` :

| Carte | Module | `regionIndexBits` | Région jouée | Largeurs d'axe | Étendues (m) |
|---|---|---|---|---|---|
| **Live Fire** | `sgh_interlock` | **2** | **1** | 12 / 12 / 11 | 63,23 / **63,78** / 22,90 |
| Banished Narrows, The Pit, Isolation | `fo05/08/09_*` | 1 | 0 | 15 / 15 / 17 | 462,64 / 453,43 / 1188,54 |
| Cliffhanger | `ridgeline` | 1 | 0 | 13 / 13 / 14 | — |

**63,775 / 2 = 31,89 m**, la médiane exacte du |Δy| mesuré au lot B sur les quatre films Live
Fire. Et les quatre films qui concentrent 3 907 des 4 901 pas impossibles sont **tous** Live Fire
(`0797ce72`, `21ece4d8`, `30724141`, `c88ec007` — carte confirmée en base, lecture seule).

### Le découpage vrai, et celui que le code lisait

Vrai (porte 4 bits) : `[precHigh][index-sel][index de région : 2 bits][X:12][Y:12][Z:11][queue:2]`
Lu (porte 3 bits)   : `[precHigh][index-sel][ 1 bit ][X':12][Y':12][Z':11]`

Décalés d'un bit, les champs se chevauchent :

```
X' = (bit bas de l'index de région) << 11 | X[11:1]     (l'index vaut 1 : X' >= 2048, toujours)
Y' = X[0]                          << 11 | Y[11:1]      <- le MSB de Y est le LSB de X
Z' = Y[0]                          << 10 | Z[10:1]      <- le MSB de Z est le LSB de Y
```

Un bit de poids faible bascule d'une image à l'autre. Le MSB d'un champ de 12 bits pèse la
**moitié** de l'étendue de l'axe : 31,89 m sur Y, 11,45 m sur Z.

### Le tableau, film par film (20 pas chacun, tous du même régime)

Les vingt premiers pas impossibles de `0797ce72` et de `21ece4d8`. Colonne « XOR » = le bit de
poids le plus fort qui a basculé entre les deux points, compté depuis le MSB du champ.

`0797ce72` (Live Fire), extrait représentatif des 20 :

| slot | bit i0 | quanta A (x/y/z) | quanta B (x/y/z) | XOR x/y/z | pas |
|---|---|---|---|---|---|
| 1029 | 1968 | 2786 / 1161 / 447 | 2791 / **3212** / 448 | msb-9 / **msb** / msb-4 | 31,93 m (dx 0,08) |
| 1029 | 1965 | 2791 / 3212 / 448 | 2797 / **1167** / 1473 | msb-8 / **msb** / msb | 31,84 m (dx 0,09) |
| 1029 | 2046 | 2803 / 1170 / 1473 | 2808 / **3222** / 450 | msb-8 / **msb** / msb | 31,95 m (dx 0,08) |
| 1029 | 1778 | 2808 / 3222 / 450 | 2814 / **1177** / 450 | msb-9 / **msb** / — | 31,84 m (dx 0,09) |
| 1029 | 1769 | 2814 / 1177 / 450 | 2819 / **3228** / 451 | msb-3 / **msb** / msb-10 | 31,93 m (dx 0,08) |

`21ece4d8` (Live Fire), extrait représentatif des 20 :

| slot | bit i0 | quanta A (x/y/z) | quanta B (x/y/z) | XOR x/y/z | pas |
|---|---|---|---|---|---|
| 1024 | 1192 | 2715 / 1017 / 1461 | 2721 / **3066** / 1462 | msb-6 / **msb** / msb-9 | 31,90 m (dx 0,09) |
| 1024 | 1738 | 2721 / 3066 / 1462 | 2728 / **1019** / 1463 | msb-8 / **msb** / msb-10 | 31,87 m (dx 0,11) |
| 1024 | 1263 | 2728 / 1019 / 1463 | 2734 / **3069** / 440 | msb-9 / **msb** / msb | 31,92 m (dx 0,09) |
| 1024 | 1112 | 2741 / 3070 / 441 | 2748 / **1023** / 442 | msb-8 / **msb** / msb-9 | 31,87 m (dx 0,11) |
| 1024 | 1069 | 2748 / 1023 / 442 | 2754 / **3072** / 443 | msb-5 / **msb** / msb-10 | 31,90 m (dx 0,09) |

**40 pas sur 40, sur deux films : le MSB de Y bascule, et lui seul explique le saut.** Le quantum
de Y alterne entre ~1 020 et ~3 070 — un écart de 2 048 = 2^11, le MSB exact d'un champ de
12 bits — pendant que ses bits bas avancent régulièrement (1017, 1019, 1023, 1026, 1029…). Ce
n'est pas une valeur qui saute : c'est un bit étranger posé en tête.

### La contre-épreuve arithmétique, sur le point publié

Premier point publié de `0797ce72` (artefact du parc, schéma 51) : `y = 7,98` puis `y = 39,92`.

- Y' = 1161 → `-10,103 + (1161+0,5) x 63,775/4096` = **7,98** ✓ (l'artefact, au centième)
- Y' = 3212 → `-10,103 + (3212+0,5) x 63,775/4096` = **39,92** ✓

Et à la porte juste, les deux se rejoignent : Y vrai ≈ 2 322 puis 2 328, soit 26,06 puis 26,25 m —
**19 cm de pas**, une trajectoire.

### Pourquoi les bipèdes ne sautent PAS (hypothèse (c), réfutée)

Le lecteur canonique du même champ, `traverse.go` (`object-position-component`) et le jumeau
bipède `decodeBipedI0Pos` (`vehicle_creation.go`), lisent tous deux l'index de région **à la
largeur de la carte** (`WorldObjectPrecision.IndexW`, installée depuis le catalogue) et, pour le
second, **comparent** l'index lu à la région attendue. Seuls `decodeWorldObjectPos` et
`projPosBits` écrivaient 3 en dur. Les bornes de carte ne sont donc pas en cause : elles sont
partagées, et elles sont justes. C'est une **factorisation abandonnée** — deux écritures du même
champ, dont une seule a suivi le catalogue quand Live Fire est arrivée (lot C catalogues,
2026-08-27).

### Verdict par hypothèse

| # | Hypothèse | Verdict |
|---|---|---|
| (a) | Bit de signe / complément à deux | **Réfutée** — les quanta sont non signés et leurs bits bas avancent régulièrement ; aucun basculement de plage |
| **(b)** | **Largeur de champ fausse** | **RETENUE** — porte 3 bits en dur contre 4 imposés par la carte ; 40/40 pas expliqués, correctif vérifié |
| (c) | Bornes de carte fausses | **Réfutée** — mêmes bornes et mêmes largeurs d'axe avant/après ; ce sont elles qui rendent la trajectoire continue une fois la porte juste. Les bipèdes ne sautent pas parce que leur lecteur lit déjà la bonne largeur d'index |
| (d) | Delta lu comme absolu | **Réfutée** — le saut est un bit EN TÊTE du champ, pas une accumulation ; les bits bas ne se réinitialisent jamais |
| (e) | Autre | sans objet |

---

## BB.2 — Le correctif

`decodeWorldObjectPos` exige désormais precHigh et index-sel nuls (2 bits), puis lit l'index de
région à `WorldObjectPrecision.IndexW` bits et le **compare** à la région jouée de la carte. Un
record d'une autre région est refusé : ses quanta sont exprimés dans une autre AABB, les
déquantifier ici rendrait une position fausse silencieuse. `projGateBits()` remplace le littéral,
et `projPosBits()` en dérive.

La région attendue voyage dans `PrecisionDescriptor.Region`, installée par
`SetWorldObjectPrecisionFromLayout` depuis `MapQuantEntry.Layout()` — **le même appel que les
largeurs d'axe**, donc sauvée et restaurée par valeur par `installWorldObjectPrecision`. Aucun
second global à armer à part.

**Garde-rail** : `world_object_gate_region_test.go` — la porte suit l'index de région (4 bits à
`IndexW=2`), une autre région est refusée, et le cas historique (`IndexW=1`, région 0) ne bouge
pas d'un bit.

### Mesure au niveau record (instrument BB.1)

| Film | Porte 3 bits (production) | Porte 4 bits (catalogue) |
|---|---|---|
| `0797ce72` | 8 091 / 15 971 pas impossibles (**50,66 %**) | **7 / 15 930 (0,04 %)** |
| `21ece4d8` | 6 958 / 13 173 (**52,82 %**) | **3 / 13 109 (0,02 %)** |
| `c88ec007` | 2 758 / 5 693 (**48,45 %**) | **1 / 5 691 (0,02 %)** |

Le nombre de records acceptés ne s'effondre pas (16 880 → 16 722) : la porte plus longue ne
sélectionne pas moins, elle lit juste.

---

## BB.3 — Mesure avant/après sur documents cuits

Dix films cuits **deux fois** (code `wt/decodeur-fork` et code de ce lot), dans une racine de
travail jetable — **aucune écriture dans le parc**, serveur de dev non touché.

| Film | Carte | Pistes | Points | **Vols tronqués** |
|---|---|---|---|---|
| `0797ce72` | Live Fire | 152 → **299** | 471 → **2 949** | 239 → **4** |
| `21ece4d8` | Live Fire | 70 → **143** | 209 → **2 367** | 144 → **1** |
| `30724141` | Live Fire | 109 → **195** | 291 → **2 012** | 162 → **0** |
| `c88ec007` | Live Fire | 27 → **70** | 96 → **1 050** | 69 → **0** |
| `fb1a1a72` | Banished Narrows | 258 = 258 | 2 692 = 2 692 | 32 = 32 |
| `51ebbc0f` | Banished Narrows | 228 = 228 | 1 862 = 1 862 | 19 = 19 |
| `e60aaf06` | Banished Narrows | 239 = 239 | 2 013 = 2 013 | 13 = 13 |
| `a4083bd2` | The Pit | 172 = 172 | 2 854 = 2 854 | 18 = 18 |
| `daaa17d6` | Isolation | 214 = 214 | 2 112 = 2 112 | 10 = 10 |
| `000d5950` | Cliffhanger (témoin) | 436 = 436 | 2 725 = 2 725 | 3 = 3 |

**Live Fire : 614 vols tronqués → 5 ; 1 067 points publiés → 8 378.** Les cinq cartes à index de
région d'un bit sont identiques **au point près** — le correctif ne déplace rien là où il n'y
avait rien à corriger.

Le compte de pas > 10 m vaut 0 des deux côtés : depuis v52, le garde-fou B.3 coupe le vol au
premier pas impossible. **C'est le nombre de coupures qui mesure le défaut**, plus le nombre de
pas. Traduit sur le parc : les 947 trajectoires aberrantes du lot B comptaient **614 Live Fire**
(65 %) et 3 907 pas sur 4 901 (**80 %**) — c'est cette part qui disparaît.

### Les autres calques

Le même décodeur sert l'équipement (`ti=37`) et les armes au sol (`ti=42`) :

| Calque | `0797ce72` | `21ece4d8` | `000d5950` (témoin) |
|---|---|---|---|
| Poses d'équipement | 49 → **227** | 27 → **114** | 295 = 295 |
| Lancers liés à leur projectile | 2 → **85** | 0 → **137** | 65 = 65 |
| Armes au sol / tirs / ramassages | inchangés | inchangés | inchangés |

Un lancer se perd sur `0797ce72` (103 → 102) : le garde d'auteur de v52 refuse une fenêtre
ambiguë, comme il le faisait déjà sur Cliffhanger.

### Contrôle croisé indépendant du seuil de 10 m

Le banc `grenade_ecart_research_test.go` (lot B) mesure la distance entre la position publiée
d'un lancer et le biped de son auteur — **il ne regarde jamais la continuité**, seulement la
position. Au lot B, sur Live Fire, la branche projectile s'effondrait à 1 et 0 lancers (toutes
les naissances refusées, à 25-27 m de leur lanceur). Après correctif :

| Film | Lancers par projectile | Médiane | Pire cas | > 4 m |
|---|---|---|---|---|
| `0797ce72` | **83** (lot B : 1) | **0,44 m** | 0,50 m | 0 |
| `21ece4d8` | **137** (lot B : 0) | **0,44 m** | 0,66 m | 0 |

C'est le régime exact de Cliffhanger (0,44 m). Deux instruments sans rapport, même verdict.

### Santé de la trajectoire corrigée

`0797ce72`, pas entre points consécutifs : médiane **0,85 m**, p95 2,48 m, max 7,70 m. Nuage des
projectiles X[-16,7 ; 24,4] Y[3,7 ; 47,9] contre bipèdes X[-10,5 ; 27,3] Y[15,3 ; 47,8] — les
projectiles débordent le nuage des corps, comme il se doit. Avant, le nuage était X[-0,7 ; 34,7] :
comprimé de moitié et décalé d'une demi-étendue, la signature du bit d'index resté en tête de X.

### Schéma et gate

`SchemaVersion` **52 → 53**, chronique datée dans `document_chronicle.go`, raison dans le garde de
`structure_test.go`. Golden `assembly_000d5950` régénéré : **il ne diffère que par la ligne de
schéma** — preuve que Cliffhanger est intact.

`replay-corpus-gate --reference=base --base=wt/decodeur-fork` : **sortie 0**, 7 témoins sur 7,
`0 perte`, `1 gain` chacun (le champ `schemaVersion`). Verdict ligne par ligne :

| Témoin | Famille | Carte | Verdict |
|---|---|---|---|
| `bcb6d393` | ctf_mono_manche | Cliffhanger | ok — 0 perte, gain = schéma |
| `fb1a1a72` | ctf_multi_manche | Banished Narrows | ok — 0 perte, gain = schéma |
| `d9781168` | oddball | Dredge | ok — 0 perte, gain = schéma |
| `c75f33b8` | assaut_bombe | Curfew | ok — 0 perte, gain = schéma |
| `bf15f7ab` | slayer | Perilous | ok — 0 perte, gain = schéma |
| `51ebbc0f` | deux_manches | Banished Narrows | ok — 0 perte, gain = schéma |
| `084a804d` | vehicules | Fortitude Heavies | ok — 0 perte, gain = schéma |

**Le gate ne couvre pas le correctif** : aucun témoin n'est sur Live Fire, la seule carte à deux
bits d'index. Il prouve la non-régression, pas le gain — c'est une découverte, ci-dessous.

---

## Gates

| Gate | Résultat |
|---|---|
| `go build ./...` (CGO) | OK |
| `go vet ./...` (CGO) | OK |
| `go test ./...` (CGO, suite complète) | OK, zéro échec |
| `make go-api-lint` (golangci-lint, ratchet) | **0 issue** |
| Fichiers > 500 L / fonctions > 80 L introduits | aucun (instrument 318 L, garde-rail 121 L) |
| Tests sans films | saut propre (`t.Skipf` sur `BITPROJ_PARC`/`BITPROJ_FILMS`) |
| Écriture dans le parc | aucune (racine de cuisson jetable) |

---

## Découvertes (non traitées)

1. **Le corpus témoin n'a aucun témoin Live Fire** — donc aucun témoin d'une carte à index de
   région de 2 bits, le seul cas où ce chemin diverge. Le gate est sorti vert sur un correctif
   qu'il ne voyait pas. Ajouter `0797ce72` (ou `21ece4d8`) à `config/replay_corpus.toml` pour la
   famille « carte à plusieurs régions de compression » fermerait ce trou.
2. **La queue de pas impossibles des cartes Forge n'est PAS la porte** (612 pas sur cinq films :
   `fb1a1a72` 241, `51ebbc0f` 139, `e60aaf06` 109, `a4083bd2` 82, `daaa17d6` 41 ; |Δx| médian
   21,69 m). Ces cartes ont un index de région d'un bit, et leurs artefacts sont **identiques**
   avant et après. Instruite ici, cette queue porte la signature d'un **faux positif du balayage
   par position de bit** : sur `a4083bd2`, les quanta X sautent de 8 192 exactement (7 938 →
   16 130 → 24 322 → 32 514, soit 2^13) pendant que Y reste figé **au quantum près** (479) sur
   quatre slots et quatre générations. Un projectile ne traverse pas 115 m en gardant son Y au
   centimètre. Le garde-fou de v52 les couvre et devient rare : c'est exactement son rôle. À
   instruire par la sélectivité du balayage (`matchWorldObjectRecord`), pas par la
   déquantification.
3. **`document_chronicle.go` passe 1 119 → 1 190 L** et `document.go` 601 → 611 L : deux fichiers
   au-delà du seuil de 500 L que la convention de chronique fait grossir à chaque bump. Même
   famille que la découverte du lot B sur `build.go` / `replaybuild.go`.
4. **La table de largeurs par index de région n'existe toujours pas** (`absPerIndexAxisW`,
   supprimée au lot E car nil). Le lecteur world-object déquantifie tous les records aux largeurs
   de LA région cataloguée ; il refuse désormais proprement ceux des autres régions au lieu de les
   lire de travers, mais il ne les lit toujours pas. Sur Live Fire, cela coûte 158 records sur
   16 880 (0,9 %).

## Ce que ce lot n'a pas fait

- **Pas de recuisson du parc** (consigne explicite) : elle reste au pilote, serveur arrêté.
- **Pas de merge, pas de push.**
- Ghidra n'a pas été ouvert : la preuve était dans le catalogue de bornes, le film et le jumeau
  bipède du dépôt — trois sources du dépôt qui concordent, sans désassemblage.
