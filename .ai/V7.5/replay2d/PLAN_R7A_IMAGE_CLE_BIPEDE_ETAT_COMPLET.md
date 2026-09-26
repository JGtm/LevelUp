# Plan R7-a — Le corps d'un record d'IMAGE-CLE est-il l'ETAT COMPLET du BIPEDE ?

> Ecrit le 2026-08-17. EXPERIENCE BORNEE (une demi-journee au plus), ouverte par le negatif
> de R5 (le corps d'image-cle n'est pas un record NEW) et par la decouverte 3 de R3
> (« l'image-cle porterait un etat complet »). R5 a probe cette lecture sur `ti=37` et
> `ti=38` ; R7-a la probe sur le SEUL archetype dont le portage est quasi complet : le
> BIPEDE, `ti=35`.
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfbiped`, branche
> `wt/kf-biped-etat-complet`, base `wt/kf-file-entite` = `7faeb25e1` (= R3 + R5 + R6).
> Execution sous le contrat du skill `plan-execution`.

---

## 1. La question — UNE seule

**Le corps d'un record NEW d'image-cle (paquet type-2) est-il l'ETAT COMPLET de l'objet,
serialise composant par composant dans l'ordre du registre (chunk_00), SANS masque ni porte
— c'est-a-dire la concatenation des deserialiseurs « leaf » de chaque composant de
l'archetype ?**

Si oui, la marche des 64 deserialiseurs sur un record `ti=35` d'image-cle doit ATTERRIR
BIT-EXACT sur le debut du record suivant, dont la position est connue par une chaine
DISJOINTE (balayage des en-tetes 64 bits `[id:32][field:26][ti:6]`, `WalkKeyframeWorld`).

### 1.1 Critere, avec son denominateur

| C | controle | seuil |
|---|---|---|
| **C1** | **>= 95 %** des records `ti=35` BORNES des 3 films atterrissent bit-exact sur la frontiere suivante (direct, ou par chainage sous la MEME variante, <= 16 intercales) | 95 % |
| **C2** | si C1 echoue : **histogramme des points de decrochage** publie (composant pendant lequel la frontiere est franchie) + histogramme des ecarts en bits, par variante et par film | publie, pas de seuil |

Denominateurs MESURES le 2026-08-17 (`TestKF35Inventory`) : **184** (`000d5950`) + **209**
(`00502e52`) + **198** (`07aa428d`) = **591** records `ti=35` bornes, sur 80 tables
d'image-cle. **Un negatif net est un livrable au meme titre qu'un positif.**

## 2. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17

### 2.1 Ce qui est etabli et NE SE REMESURE PAS

| fait | piece |
|---|---|
| Le corps d'image-cle n'est PAS un record NEW « [etat par defaut][porte][masque][composants] » (0 marche exacte, 128 decalages x 16 lectures, `ti=37`/`ti=38`) | `PLAN_R5_GRAMMAIRE_IMAGE_CLE.md` phases 2-3 |
| Le jeu ne relit JAMAIS le payload type-2 (aucun handler : `JZ 0x1428e2412`) | `PLAN_R6_FILE_PAR_ENTITE.md` §10 |
| En-tete 64 bits `[id:32][field:26][ti:6]` confirme ; `ti` a `+58` (415/415, 2008/2008) | R5, `keyframe_record_walk.go:29` |
| Longueur reelle des records fortement quantifiee (`ti=38` : 39 valeurs / 2 008) | R5 |
| L'ecrivain de l'image-cle peut differer du deser delta MEME sur un champ deja porte (lot « poses », `feat/v75` `160e4ea7b` : double champ, decoupage 9/5 vs 8/3 sur `ti=37`) | brief du superviseur |
| Le balayeur d'oracle SAUTE les records a `field26 != 0` -> le « suivant » peut etre le voisin du voisin ; le chainage le rattrape | `keyframe_world.go:75`, `ChainKeyframeRecords` |

### 2.2 Le portage du bipede — MESURE, pas suppose (`TestKF35Inventory`, 2026-08-17)

L'archetype `ti=35` (`bipedDefaultStateTypeIndex`, `traverse.go:958`) porte **64 composants**
(`i0`..`i63`). Sonde a flux nul sur le dispatch de production (`consumeByName`,
`traverse.go:195`) : **63 portes, 1 NON PORTE**.

- `[!]` **`i60 simulation-state-component` — NON PORTE** (`traverse.go:1154`
  `consumeSimulationState`, gardee par `simStateComplete` / `simStateExtra`, defaut : rend
  `ported=false`). C'est le seul trou statique, et la variante **v4** le neutralise.
- **Reserve honnete** : la sonde est a flux NUL. Trois desers a branchement sur VALEUR
  peuvent rendre `ported=false` sur donnee reelle (`components_biped_ability.go:493`
  « tag >= 6 ... treat as unported », `:590` « value-gated dispatch unported »). La mesure
  publie donc AUSSI l'histogramme reel des composants de desync — c'est lui qui fait foi.
- **Reserve n°2** : « porte » ne veut pas dire « bit-exact en image-cle ». Plusieurs largeurs
  sont gouvernees par la precision runtime de la carte (`i0`, `i21` — `calibratedWidth`,
  `traverse.go:1240`) et lues 0 statiquement. Un decrochage sur `i0` n'est donc PAS une
  refutation de l'hypothese : c'est un trou de largeur connu. La mesure le nomme.

### 2.3 Instruments REUTILISES (aucun deserialiseur recopie)

`WalkKeyframeBody` + `KeyframeBodyVariant` (R5, `keyframe_record_walk.go:305`),
`readKeyframeHeader`, `WalkKeyframeWorld`, `TraverseEntity` / `traverseComponentLoop`,
`SetUnportedStubWidth` (hook de calibration deja present, `traverse.go:1261`),
`SetFilmComponentCorruptionCheck`, `WalkPackets` / `ReadFilmChunk` / `ParseRegistryChunk`,
`LockProcessDecode`.

## 3. Corpus — FERME

Racine LECTURE SEULE : `C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/`.
Films : `000d5950`, `00502e52`, `07aa428d` (les 3 oracles de R3/R5/R6), **toutes** leurs
tables d'image-cle (80 au total : 26 / 28 / 26 — une par chunk, cf. decouverte R6).

## 4. Variantes probees — ORDONNEES, peu nombreuses

| v | lecture du corps |
|---|---|
| **v1** | les 64 leaf dans l'ordre du registre, **sans etat par defaut, sans porte, sans masque** |
| **v2** | **etat par defaut du bipede** (`consumeBipedDefaultState` + tail) puis les 64 leaf |
| **v3** | etat par defaut, **porte `R(1)`**, masque PLEIN (non lu), puis les 64 leaf |
| **v4** | v1 **en sautant (0 bit) tout composant non porte**, par convergence, et mesure de l'ecart residuel |

**AJOUT en cours d'execution — le TEMOIN DE CONTROLE.** Une longueur consommee ne se juge
pas dans le vide : les variantes sont precedees de **v0** (la lecture « record NEW » que R5
a refutee : etat par defaut + porte + masque) et **v0b** (la meme, trous neutralises). Sans
ce temoin, dire « l'etat complet consomme 2 880 bits » n'apprend rien.

Dimension croisee : le **corruption-check du mode film** (`R(1)` par composant present,
`filmComponentCorruptionCheck`) ETEINT puis ALLUME — il vaut 64 bits d'ecart sur un
archetype plein, il ne se suppose pas. **8 passes par film, 24 au total.**

## 5. Phases

### Phase 0 — Plan — CLOSE le 2026-08-17

- [x] 0.1 Question, critere, denominateurs, corpus ferme, variantes ordonnees, gates, contrat.
- [x] 0.2 Etat des lieux SUR PIECES du portage bipede (§2.2), mesure et non suppose.

### Phase 1 — MESURER — CLOSE le 2026-08-17

- [x] 1.1 Instrument `filmdec/keyframe_biped_fullstate_test.go` (452 L, fichier NEUF, garde
      `KF35_ROOT`, **2 SKIP verifies sans la garde**), `TestKF35Inventory` +
      `TestKF35FullState`. Aucun deserialiseur recopie : il rejoue `WalkKeyframeBody` (R5),
      `TraverseEntity` / `traverseComponentLoop`, `SetUnportedStubWidth`, `WalkKeyframeWorld`.
- [x] 1.2 **36 passes** executees (6 variantes x 3 films x corruption-check OFF/ON — 24
      prevues, 36 apres l'ajout du temoin), denominateurs publies : **184 / 209 / 198 = 591**
      records `ti=35` bornes.
- [x] 1.3 Publie par passe : atterrissages exacts + chaines, atterrissages sur en-tete valide,
      desync, longueur consommee MEDIANE, ecart absolu MEDIAN, histogramme des ecarts,
      **histogramme des points de decrochage**, histogramme des composants de desync.
- [x] 1.4 **Aucune variante n'approche au bit** — mais l'ECHELLE, elle, tombe juste, et c'est
      le resultat neuf du lot (§8, tableau). L'ecart n'est ni petit ni stable (167 a 202
      valeurs distinctes pour 184 a 209 records : quasiment tous differents), donc il n'y a
      **aucun bloc constant manquant** a trouver — la derive est per-composant.

### 1.5 LES CHIFFRES — longueur du record, mediane, par film

| grandeur (mediane, bits) | `000d5950` | `00502e52` | `07aa428d` |
|---|---|---|---|
| **longueur REELLE du record `ti=35`** (oracle `want - Bit`) | **2 765** | **2 777** | **2 781** |
| v0b lecture « record NEW » (temoin R5), corr. OFF | 1 074 (39 %) | 325 (12 %) | 537 (19 %) |
| **v4 etat complet (64 leaf, trous a 0), corr. OFF** | **2 877 (104 %)** | **2 890 (104 %)** | **2 830 (102 %)** |
| v4 etat complet, corr. ON | 2 944 (106 %) | 3 075 (111 %) | 3 070 (110 %) |
| ecart ABSOLU median, v4 corr. OFF | 920 | 757 | 927 |
| **ecart ABSOLU median, v4 corr. ON** | **370** | **474** | **499** |

Longueur REELLE : **132 / 154 / 151 valeurs distinctes** pour 184 / 209 / 198 records — la
quantification forte que R5 avait vue sur `ti=38` **ne se retrouve PAS sur `ti=35`** (les
longueurs de bipede sont presque toutes differentes).

**Gate 1** (commandes exactes, sorties collees au journal §8) :

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                       (doit etre vide)
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ -run '^TestKF35' -v   (SKIP sans la garde)
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ -run '^TestKF35' -timeout 30m -v
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ ./internal/analysis/replay/
```

### Phase 2 — VERDICT — CLOSE le 2026-08-17

- [x] 2.1 **VERDICT : NEGATIF AU BIT, POSITIF A L'ECHELLE.**

  **C1 ECHOUE, et largement.** Seuil 95 % ; **plafond mesure 0,51 %** (1 record sur 198,
  `07aa428d`, v4, corruption OFF). Total sur les 36 passes : **2 atterrissages exacts sur
  ~7 000 tentatives**, aucun par chainage. La concatenation des 64 deserialiseurs « leaf »
  n'atterrit PAS bit-exact sur le record suivant.

  **CE QUE LA MESURE EXCLUT.**
  1. Les variantes v1, v2, v3 **ne finissent jamais** : 591/591 records desynchronisent, et
     toujours sur les trois memes composants a branchement sur valeur — `i60`
     `simulation-state-component` (~83 %), `i57` `biped-spartan-ability-component` (~9 %),
     `i59` `biped-spartan-ability-non-predicted-state` (~6 %). Tant que ces trois-la ne sont
     pas portes, AUCUNE lecture « etat complet » du bipede ne peut etre validee au bit. C'est
     le verrou nomme, et il est court : trois desers, pas soixante-quatre.
  2. L'ecart residuel de v4 n'est **ni petit ni stable** (167 a 202 valeurs distinctes pour
     ~200 records). Il n'y a donc **pas de bloc constant manquant** (pas de tail oublie, pas
     d'en-tete supplementaire) : la derive s'accumule composant par composant.
  3. Le point de decrochage change de dominante avec le corruption-check : OFF, c'est la
     **sous-lecture** (46 a 50 %) suivie de `i9 object-multiplayer-properties-component`
     (25 %) ; ON, c'est `i63 biped-action-component` (35 a 36 %). Les deux suspects sont
     nommes ; `i9` est precisement le bloc dont R3 avait deja mesure une divergence de
     decoupage (« double champ », lot poses `160e4ea7b`).

  **CE QUE LA MESURE OUVRE, ET C'EST NEUF.** A l'ECHELLE, l'hypothese tient : la
  concatenation des 64 leaf consomme **102 a 104 %** de la longueur reelle du record
  (2 830-2 890 bits contre 2 765-2 781), quand la lecture « record NEW » n'en consomme que
  **12 a 39 %**. Un facteur 3 a 8 separe les deux lectures, et c'est l'etat complet qui tombe
  juste. **Le corps d'un record d'image-cle EST de la taille d'un etat complet, pas d'un
  record NEW** — la decouverte 3 de R3 est confirmee quantitativement pour la premiere fois,
  sur 591 records et 3 films. Ce qui manque n'est plus la FORME de la lecture, mais la
  bit-exactitude de quelques desers. Le corruption-check du mode film ALLUME divise l'erreur
  absolue mediane par ~2 (920/757/927 -> 370/474/499) : il est probablement present dans le
  payload type-2, ce qu'aucun lot n'avait mesure.
- [x] 2.2 Lignes de registre redigees au §10 (ce lot n'ecrit PAS dans `REGISTRE_REPORTS.md`).
- [x] 2.3 Entree `thought_log.md` redigee au §9 et remise au superviseur.
- [x] 2.4 Commit `mesure(v7.5-rejeu-kf35):` sur `wt/kf-biped-etat-complet`, hooks ACTIFS,
      `git push -u origin wt/kf-biped-etat-complet`.

## 6. Contrat — NON NEGOCIABLE

1. **Je ne CREE que mes fichiers** : `filmdec/keyframe_biped_fullstate_test.go` et ce plan.
   **AUCUN fichier partage du decodeur modifie** (`traverse.go`, `keyframe_*.go`,
   `default_state*.go`, `frame_records.go`, `replay/*`).
2. **Aucune bosse de `SchemaVersion`** (reste a 8). Aucune ecriture DuckDB, aucun rendu,
   aucune string UI, aucune re-cuisson d'artefact, aucun run de masse.
3. Lecture seule stricte hors du worktree ; `LevelUp` et les autres `LevelUp-wt-*` ne sont
   pas touches.
4. `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfbiped/.gocache`, **une seule commande
   `go` a la fois**, `CGO_ENABLED=0`.
5. Bascules globales restaurees en `defer` ; `LockProcessDecode` tenu pour tout le test.
6. Pas de Python, pas d'emoji, seuils 500 L / 80 L / 5 parametres respectes.
7. Zero fix opportuniste hors perimetre : toute decouverte va au §7, non traitee.
8. Jamais `--no-verify`, jamais `git stash`, jamais `main`, jamais de merge.

## 7. Decouvertes — consignees, NON traitees

1. **Trois desers, pas soixante-quatre.** `i57` / `i59` / `i60` sont les SEULS composants du
   bipede qui bloquent la lecture « etat complet » (591/591 desyncs y aboutissent). `i60`
   `simulation-state-component` a deja ses bascules (`simStateExtra`, `simStateComplete`,
   `traverse.go:1115-1131`) et n'attend qu'une largeur. C'est le chantier suivant le moins
   cher du dossier. NE PAS traiter ici.
2. **Le corruption-check du mode film semble PRESENT dans le payload type-2** : l'allumer
   divise l'erreur absolue mediane par ~2 sur v4 (920/757/927 -> 370/474/499 bits). Aucun lot
   ne l'avait mesure sur l'image-cle. NE PAS traiter ici.
3. **La quantification des longueurs n'est PAS universelle** : R5 mesurait 39 valeurs
   distinctes sur 2 008 records `ti=38` ; `ti=35` en donne 132 / 154 / 151 pour 184 / 209 /
   198. Un archetype riche a des longueurs quasi toutes differentes. Toute inference basee sur
   « les longueurs sont quantifiees » doit nommer son archetype. NE PAS traiter ici.
4. **`i9 object-multiplayer-properties-component` est le suspect n°1 en corruption OFF**
   (25 % des decrochages, les 3 films) — le meme bloc que le lot « poses » (`160e4ea7b`)
   accusait deja d'un mauvais decoupage sur `ti=37`. Deux chaines independantes le designent.
   NE PAS traiter ici.
5. **Piege d'instrument (generique).** Une marche « etat complet » peut lire TRES au-dela de
   la fin du tampon : le `BitReader` rend des zeros et certains desers a boucle explosent
   (ecarts observes a -123 168 et -8 387 965 bits). Les MEDIANES publiees ici sont robustes a
   ces valeurs ; une moyenne ne le serait pas. Meme famille de piege que le `KFQWalk.Overrun`
   de R6. NE PAS traiter ici.
6. **Piege d'outillage.** `go test ... | tee log | head -N` a rendu un `FAIL` **sans aucun
   message** (run du 2026-08-17, 165 s) alors que la meme commande sans tube passe. Le verdict
   d'un gate ne se lit jamais a travers un tube — memoire `reference_pipeline_exit_code_masks_gate_verdict`.
   NE PAS traiter ici.

## 8. Journal d'execution

**2026-08-17 — Phase 0 CLOSE.** Reconnaissance de pre-plan : `TestKF35Inventory` publie les
64 composants du bipede et leur portage (63/64 sur flux nul, `i60` non porte), et les 3
denominateurs (184 / 209 / 198).

**2026-08-17 — Phase 1 CLOSE.** 36 passes, 591 records bornes, 80 tables d'image-cle.
Commandes de gate executees dans cette session, toutes vertes :

```
CGO_ENABLED=0 go build ./internal/analysis/...                          -> BUILD_OK
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/                     -> (vide)
gofmt -l internal/analysis/filmdec/                                     -> (vide)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run '^TestKF35' -v  -> 2 SKIP
CGO_ENABLED=0 KF35_ROOT=<...> go test ./internal/analysis/filmdec/ \
      -run '^TestKF35' -timeout 60m -v                                  -> PASS (~9 min)
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
      -> ok filmdec 31,7 s · ok replay 33,0 s
```

Resultats au §1.5 (longueurs) et §2.1 (verdict). Aucun fichier partage du decodeur touche :
`git status` ne montre que ce plan et `keyframe_biped_fullstate_test.go`.

**2026-08-17 — Phase 2 CLOSE.** Verdict ecrit, decouvertes consignees, lignes de registre et
entree de journal redigees, commit et push.

## 9. Entree thought_log (redigee, NON ecrite par ce lot)

```
### [2026-08-17] Lot R7-a — L'image-cle a la TAILLE d'un etat complet, pas les BITS

Statut : Complete (experience bornee, negatif mesure + resultat quantitatif neuf)

Decision technique. Probe sur le BIPEDE (ti=35, 64 composants, 63 portes) l'hypothese
« le corps d'un record d'image-cle est l'etat complet, 64 deserialiseurs leaf concatenes
dans l'ordre du registre, sans masque ni porte ». Instrument neuf
`filmdec/keyframe_biped_fullstate_test.go` (garde KF35_ROOT), qui REJOUE les lecteurs de
production (WalkKeyframeBody de R5, TraverseEntity, SetUnportedStubWidth) — aucun deser
recopie, aucun fichier partage du decodeur modifie. 6 variantes (dont un temoin « record
NEW »), corruption-check du mode film OFF/ON, 3 films oracles, 591 records ti=35 bornes
par WalkKeyframeWorld (chaine disjointe).

Resultats observes. C1 (>= 95 % d'atterrissage bit-exact) ECHOUE : plafond 0,51 %,
2 atterrissages sur ~7 000 tentatives. Les variantes sans neutralisation des trous ne
finissent JAMAIS : 591/591 desyncs, toujours sur i60 simulation-state (~83 %), i57 et i59
biped-spartan-ability (~15 %). MAIS l'echelle tombe juste : la concatenation des 64 leaf
consomme 2 830-2 890 bits pour une longueur reelle mediane de 2 765-2 781 (102-104 %),
quand la lecture « record NEW » n'en consomme que 12-39 %. L'ecart residuel n'est ni petit
ni stable (167-202 valeurs distinctes pour ~200 records) : pas de bloc constant manquant,
la derive est per-composant. Allumer le corruption-check du mode film divise l'erreur
absolue mediane par ~2 (920/757/927 -> 370/474/499 bits).

Conclusion / prochaine etape. La FORME de la lecture d'image-cle est tranchee : c'est un
etat complet, pas un record NEW — la decouverte 3 de R3 est confirmee quantitativement
pour la premiere fois. Ce qui manque n'est plus la grammaire mais la bit-exactitude de
TROIS desers (i57, i59, i60) plus le decoupage d'i9 object-multiplayer-properties (deja
suspecte par le lot poses) et d'i63 biped-action. Verrou court et nomme, a arbitrer.
```

## 10. Lignes de registre (redigees, NON ecrites par ce lot)

```
| 2026-08-17 | R7-a image-cle = etat complet (ti=35) | MESURE, negatif au bit / positif a
l'echelle : 0,51 % d'atterrissage bit-exact (seuil 95 %) sur 591 records, MAIS la
concatenation des 64 leaf consomme 102-104 % de la longueur reelle du record contre 12-39 %
pour la lecture « record NEW ». Condition de reprise : porter bit-exact i60
simulation-state-component, i57 et i59 biped-spartan-ability-* (les 3 seuls blocages, 591/591
desyncs), puis re-mesurer avec le corruption-check du mode film ALLUME (il divise deja
l'erreur mediane par 2). |
| 2026-08-17 | AMENDEMENT au report R5/R6 « grammaire du corps d'image-cle » | la question
n'est plus « quelle grammaire » (la forme est tranchee : etat complet) mais « quels desers
sont faux ». Suspects nommes et ordonnes : i60 / i57 / i59 (bloquants), i9
object-multiplayer-properties (25 % des decrochages, corruption OFF ; deja suspecte par le lot
poses 160e4ea7b), i63 biped-action (35 % des decrochages, corruption ON). |
```
