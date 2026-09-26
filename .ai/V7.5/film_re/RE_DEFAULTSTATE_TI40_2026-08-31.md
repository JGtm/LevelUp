# RE — LE DEFAULT-STATE DE L'ARCHETYPE VEHICULE (ti=40), FUN_1410A5A74

> Etabli le 2026-08-31 dans le worktree `LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`.
> Lot V1b (retro-ingenierie + portage). Chaque largeur de bit cite sa piece : une adresse de
> decompilation/desassemblage Ghidra, jamais une supposition. Aucune ecriture Ghidra (lecture
> seule stricte). Aucun `go build`/`go test` lance (cache Go partage avec le lot V1a).

## 0. LE RESULTAT EN CINQ LIGNES

1. **La grammaire du default-state de `ti=40` est etablie feuille par feuille**, et six de ses
   sept feuilles ont une largeur figee STATIQUEMENT (§ 2).
2. **UNE feuille n'est PAS etablie statiquement** : le bloc quaternion `FUN_14076e494`, derriere
   la porte de flux `bVar14`, lit des largeurs per-axe de configuration runtime (`DAT_1445cc9e0`)
   et un index runtime (`DAT_144632be0`) — exactement le meme bloc que le media-frame du BIPEDE,
   modelise ABSENT dans le depot (§ 3, FINDING A).
3. **En consequence, et par la regle de `default_state_arch.go:30-32`, `ti=40` n'est PAS inscrit
   dans `defaultStateDeserByTI`** : un archetype dont UNE largeur de feuille n'est pas etablie
   statiquement n'entre pas dans la table. C'est un resultat, pas un echec (§ 5).
4. **Le mot d'identite `MPPWord32` est LISIBLE sans decoder la position** : le bloc MPP
   (`FUN_14080cfe8`) est la 2e feuille, lue AVANT toute position. Le port le publie par le hook
   `mppHook`, deja existant (§ 4). MESURE 2026-08-31 : **c'est l'identite du chassis** — 7 valeurs
   sur 0d76e8f1, 5 sur fccc61cd, CONSTANTES par vie, dont **5 partagees entre les deux builds**
   (§ 8).
5. **Le gate de selectivite est i0 en PRECISION-DYNAMIQUE** (bipede, 5 bits), PAS world-object
   (3 bits) : c'est la cause racine mesuree par le superviseur (le gate world-object rendait un
   temoin fantome de PLUS de records que la vraie bande). Corrige (§ 6, § 8) : `equipCreationWalk`
   est desormais parametre par un decodeur de position, `ti=40` passe `decodeBipedI0Pos`, et
   l'oracle durcit le gate par le nuage des positions reelles (methode ti=42 transposee). Le
   temoin fantome tombe alors a 0-3 %.

---

## 1. CHAINE DE RESOLUTION

`KEYFRAME_ARCHETYPE_DEFAULTSTATE_TABLE.md:30` donne `ti=40 -> vtable[0x60] = 0x1410A5A74`,
classe **REAL** (pas le stub a zero bit `FUN_1408d8220`). C'est le deserialiseur appele par le
lecteur de record NEW `(**(code **)(*plVar2 + 0x60))(...)` (cf. `default_state_arch.go:14`).

La decompilation de `FUN_1410a5a74` est sauvee en scratchpad (`decomp_FUN_1410a5a74.c`) et
confrontee ci-dessous a son desassemblage (adresses `0x1410a5a74`..`0x1410a5b6b` pour le corps
principal, `0x1424a3a02` et `0x1424a3a39` pour les deux blocs FROIDS hors-ligne).

Convention du bitreader (`param_4` dans l'exe) : toute feuille qui lit fait `*(param_4+0x2c) += N`.
C'est le compteur de bits consommes ; il tranche chaque largeur.

## 2. GRAMMAIRE FEUILLE PAR FEUILLE — CE QUI EST STATIQUE

Ordre d'execution de `FUN_1410a5a74`. Chaque ligne : nom, largeur, adresse-source, porte.

| # | feuille | largeur | source (piece) | gardee par |
|---|---|---|---|---|
| 1 | prefixe « version » | R(1) ; si 1 -> R(8) | `CALL 0x1406cf008 @0x1410a5a96` (gate) ; `R10D=R14+7=8 @0x1410a5aa5`, `ADD [RSI+0x2c],R10D @0x1410a5abf` (corps) | inconditionnel (gate), puis bit 1 |
| 2 | bloc `object-multiplayer-properties` | variable, largeurs closes | `CALL 0x14080cfe8 @0x1410a5ad4` (RDX=RSI = bitreader, RCX=RBP = DST) | inconditionnel |
| 3 | porte `bVar14` -> DST+0x60 | R(1) | inline `ADD [RSI+0x2c],R14D=1 @0x1410a5aed` ; `MOV [RBP+0x60],DL @0x1410a5b04` | inconditionnel |
| 4 | quaternion + `FUN_140c1e79c` | **NON ETABLIE (config runtime)** | `JNZ 0x1424a3a02 @0x1410a5b09` -> bloc froid ; cf. § 3 | bit 3 == 1 |
| 5 | `FUN_14076dc04` -> DST+0x88 | R(19) | `MOV R9D,0x13 @0x1410a5b16` (largeur = 4e arg), `CALL 0x14076dc04 @0x1410a5b1f` ; `FUN_14076dc04` fait `+= param_4` @`0x14076dc04` | inconditionnel |
| 6 | porte `cVar3` -> DST+0xac | R(1) | `CALL 0x1406cf008 @0x1410a5b46` ; `MOV [RBP+0xac],AL @0x1410a5b4b` | inconditionnel |
| 7a | `FUN_14080d69c` -> DST+0xa8 | R(1) ; si 1 -> R(32) | `CALL 0x14080d69c @0x1410a5b63` (RDX=RSI = bitreader) ; interne `MOV RDX,RBP ; CALL 0x14080d6f0 @0x14080d6ce` | atteinte si `cVar3 == 0` |
| 7b | liste de refs -> DST+0x9c[] | R(2) count ; count x [R(1) ; si 1 -> R(32)] | `ADD [RSI+0x2c],2 @0x1424a3a4c` (count, `SHR R9,0x3e` = 2 bits de tete) ; boucle : `MOV RCX,RSI ; CALL 0x1406cf008 @0x1424a3afb` (R(1)), `MOV RDX,RSI @0x1424a3b09 ; CALL 0x14080d6f0 @0x1424a3b0c` (R(32)) | atteinte si `cVar3 != 0` |

### 2.1 Sous-fonctions — une phrase de synthese chacune

- **`FUN_1406cf008` (`@0x1406cf008`)** : lecteur d'UN bit ; `+0x2c += 1` puis renvoie le bit de
  tete. C'est la porte R(1) partout dans ce deser.
- **`FUN_1406d6c7c` (`@0x1406d6c7c`)** : lecteur GENERIQUE de N bits (`param_2` = N) ; `+0x2c += N`.
  C'est le chemin de recharge (refill) qu'inlinent les feuilles 1/3/7b.
- **`FUN_14080cfe8` (bloc MPP, deja porte `consumeMultiplayerPropertiesBlock`, `default_state.go:331`)** :
  le plus gros sous-lecteur ; il publie `MPPWord9` (R(9) `FUN_141fd72c0`) puis **`MPPWord32`
  (R(32) `FUN_14080d6f0`)**, le mot inconditionnel d'identite. Les six appelants directs de
  `FUN_14080cfe8` incluent `FUN_1410a5a74` (`KILLFEED_STATE.md:166-168`) : le record de creation
  d'un vehicule CONTIENT un bloc MPP, comme l'equipement et l'arme au sol. **On le REUTILISE, on
  ne le re-porte pas.** Les deux largeurs variables du bloc (lead/index, defaut 9/5) sont une
  config de replication du film, deja gouvernee par `CalibrateMPPWidths` (§ 4.2).
- **`FUN_14076dc04` (`@0x14076dc04`)** : lecteur de `param_4` bits ; ici `param_4 = R9D = 0x13`
  (fige au call-site `@0x1410a5b16`), donc **R(19)** inconditionnel. Meme feuille et meme largeur
  que dans le default-state BIPEDE (`default_state.go:187`, « R(19) width R9D=0x13 »).
- **`FUN_14080d69c` (`@0x14080d69c`, deja porte `consumeOpt32`, `unit_weaponstate.go:49`)** :
  R(1) ; si 1 -> `FUN_14080d6f0` = R(32). Le desassemblage `@0x14080d6cb (MOV RDX,RBP)` confirme
  que le R(32) frappe le MEME reader que la porte (ici le flux film) — donc `consumeOpt32` exact.
- **`FUN_14080d6f0` (`@0x14080d6f0`)** : lecteur de **R(32)** (`+0x2c += 0x20`). Sert et le
  MPPWord32 et la feuille 7.
- **`FUN_1405838f0` (`@0x1405838f0`)** : validation PURE sur un entier local (`*param_1 + 1U`),
  **0 bit de flux**. Dans la boucle 7b, elle juge la valeur lue avant de la stocker.
- **`FUN_14076e494` / `FUN_14076e524` / `FUN_140cc5128`** : le bloc quaternion — voir § 3, c'est
  la feuille non-statique.
- **`FUN_140c1e79c` (`@0x140c1e79c`)** : R(1) ; si 0 -> R(19) ; PUIS `FUN_1406d84b4`
  inconditionnel dont la largeur est un arg de pile (`in_stack_00000028`) non figee a ce
  call-site. Cette feuille est ELLE AUSSI derriere `bVar14` (§ 3) — donc absente quand `bVar14==0`.

## 3. FINDING A — LA FEUILLE NON ETABLIE STATIQUEMENT (le quaternion, porte bVar14)

La feuille 4 (bloc froid `@0x1424a3a02`, atteint quand `bVar14==1`) est, dans la decompilation :

```
if (bVar14 != false) {
  FUN_14076e494(param_4, param_3 + 100, 0x10, 0, param_5, 0);   // quaternion -> DST+0x64
  FUN_140c1e79c(param_4);
}
```

Sa largeur depend de trois globaux de configuration RUNTIME, illisibles statiquement :

1. **`FUN_14076f91c` (`@0x14076f91c`)** — porte de branche : `uVar1 = 1 ; if (DAT_144e61ea0 == 0
   && DAT_145121140 != 1) uVar1 = 0`. Deux globaux de config (les memes que la queue config-gatee
   du BIPEDE, `default_state.go:51`).
2. **`FUN_14076e524` (`@0x14076e524`)** — le corps du quaternion : R(1) gate (`FUN_1406cf008`) ;
   si gate==0 -> R(**`DAT_144632be0`**) index (`ADD [param_2+0x2c], DAT_144632be0`) ; puis
   `FUN_140cc5128`.
3. **`FUN_140cc5128` (`@0x140cc5128`)** — bloc per-axe : boucle `lVar8 = 3 ; do { lit
   `*(int*)param_4` bits ; param_4 += 4 } while(--lVar8)`. Les trois largeurs viennent d'un
   tableau fourni par l'appelant = **`DAT_1445cc9e0`** (les axis widths, poses au chargement de la
   carte). `default_state.go:330` les declare deja « non sourcable statiquement ».

**C'est le MEME bloc que le media-frame du BIPEDE** (`consumeBipedDefaultStateMediaFrame`,
`default_state.go:299`), lui-meme modelise ABSENT (`bipedMediaFramePresent = false`) parce que,
sur un decode d'image-cle frais, l'objet DST est memset(0) et la porte tombe. Difference a
NOTER : chez le bipede la porte est un etat DST (0 bit de flux) ; ici `bVar14` est un BIT DE FLUX
(feuille 3, R(1)). Donc pour `ti=40` on doit LIRE le bit (1 bit, statique), mais si sa valeur est
1 on ne peut pas sauter statiquement le corps config-dependant.

**Conclusion FINDING A** : la feuille 4 n'a PAS de largeur etablie statiquement. Le port la
modelise absente (`vehicleMediaFrameBits = 0`, exactement `bipedDefaultStateTailBits`) : bit-exact
tant que `bVar14 == 0` (le cas nominal attendu d'un spawn, comme le bipede), desaligne sinon. La
part de records a `bVar14 == 1` est une MESURE (l'oracle de position, non lance ici) — pas une
supposition.

## 4. L'IDENTITE MPPWord32 — LISIBLE, ET SANS LA POSITION

La feuille 2 (bloc MPP) est lue juste apres le prefixe version, AVANT toute porte optionnelle et
AVANT toute position. `consumeMultiplayerPropertiesBlock` publie `MPPWord32` par `mppHook`
(`default_state.go:333`). Donc l'identite du chassis se lit meme si la position est indecodable :
il suffit de derouler `consumeDefaultStateTI40` depuis la fin d'un en-tete NEW `ti=40` et de
capter le hook. Le test V1.5 (§ 4.2) le fait SANS decoder i0.

### 4.1 Precedent (deux fois etabli dans le depot)

- `ti=37` (equipement) : `MPPWord32` = GlobalID du tag `eqip`, 11 valeurs sur 11 resolues
  (`equipment_creation.go:30-33`).
- `ti=42` (arme au sol) : hypothese tag `weap` (`ground_weapon_creation.go:18`).
- **`ti=40` (vehicule) : hypothese tag `vehi`** (cadrage § 3.1).

### 4.2 Gates de mesure (cadrage § 5.1 V1.5), ecrits AVANT le code du test

1. constance 100 % par vie `(slot, gen)` ;
2. <= 8 valeurs distinctes par film (Behemoth / Launch Site) ;
3. le nombre de valeurs ne suit PAS le nombre de vies ;
4. (plus) resolution en tag `vehi` par `himap.ModuleIndex` — note comme etape suivante si le
   cout depasse le lot.

## 5. CONFRONTATION A `default_state_ti42.go` ET DECISION D'INSCRIPTION

| aspect | `ti=42` (`FUN_1407f0c68`) | `ti=40` (`FUN_1410a5a74`) |
|---|---|---|
| prefixe version | R(1)+opt R(8) | R(1)+opt R(8) — IDENTIQUE |
| bloc MPP | oui (via deser ti36) | oui (appel direct `FUN_14080cfe8`) |
| feuilles suivantes | R(12), R(7), magazine-list, gate0R(5) — TOUTES statiques | R(1) porte, **quat config**, R(19), R(1) porte, opt32 / R(2)+boucle |
| feuille non-statique | AUCUNE | **1 (le quaternion, feuille 4)** |
| chemin minimal | 80 bits (`GroundWeaponDefaultStateMinBits`) | **79 bits** (`VehicleDefaultStateMinBits`, quat absent) |
| inscription table | OUI (apres oracle, 2026-08-17) | **NON** (feuille non-statique) |

Detail du chemin minimal `ti=40` (toutes portes fermees, `bVar14==0`, defaut MPP 9/5) :
`1 (V) + 56 (MPP) + 1 (porte bVar14) + 19 (R19) + 1 (porte cVar3) + 1 (opt32 ferme) = 79`.

**DECISION** — conforme a `default_state_arch.go:30-32` et a la consigne de lot (« si une largeur
reste inconnue, tu le DIS et tu n'inscris pas ti=40 dans la table ») :

- `consumeDefaultStateTI40` est ECRIT (feuilles statiques bit-exactes, quat modelise absent) ;
- **`ti=40` n'est PAS ajoute a `defaultStateDeserByTI`** (le chemin record-NEW des IMAGES-CLES),
  car la feuille 4 n'est pas etablie statiquement ;
- l'inscription est la MEME etape post-oracle que pour `ti=42` : elle attend que l'oracle de
  position confirme que le port atterrit sur i0 (donc que `bVar14` est nominalement 0). Cet oracle
  se lance par le superviseur (contrainte de cache Go).

## 6. FINDING B — i0 DE ti=40 EST EN PRECISION-DYNAMIQUE (divergence de la voie equipCreationWalk)

`equipCreationWalk.readCreation` (`equipment_creation.go:373`) decode i0 par
`decodeWorldObjectPos` (`projectiles.go:362`), dont la PORTE est de **3 bits**
(`PeekBits(pay, at, 3) != 0`). Or i0 de `ti=40` est `object-position-dynamic-precision-component`
(cadrage § 2.1), a porte de **5 bits** (bipede). Consequence :

- la POSITION rendue par `equipCreationWalk` pour `ti=40` est fausse, et les records peuvent etre
  REJETES sur `PosBad` (la porte 3 bits tombe sur des bits dyn.-prec.) ;
- donc l'oracle de position d'`equipCreationWalk` / `equipOffsetProbe` (world-object) ne sert PAS
  `ti=40` tel quel : il faudrait un variant dyn.-prec., qui est le terrain de V1.2
  (`ScanFilmBipedPositionsForBand`), hors de ce lot.

Ce qui n'est PAS affecte : `MPPWord32`, lu dans la feuille 2 AVANT toute position (§ 4). La voie
`equipCreationWalk{ti:40}` est cablee (consigne de lot) comme ECHAFAUDAGE — elle rendra la position
juste le jour ou un decode i0 dyn.-prec. lui est fourni. La mesure d'identite, elle, ne l'attend
pas : le test V1.5 deroule `consumeDefaultStateTI40` directement.

## 7. STATUT DE CHAQUE FEUILLE (recapitulatif)

| feuille | largeur etablie ? | source |
|---|---|---|
| 1 version | OUI R(1)+opt R(8) | `@0x1410a5a96`, `@0x1410a5abf` |
| 2 MPP | OUI (closes, lead/index config connue) | `@0x1410a5ad4` + `default_state.go:331` |
| 3 porte bVar14 | OUI R(1) | `@0x1410a5aed` |
| 4 quaternion | **NON (config runtime `DAT_1445cc9e0`/`DAT_144632be0`)** | `@0x14076e524`, `@0x140cc5128` |
| 5 R(19) | OUI | `@0x1410a5b16`, `@0x14076dc04` |
| 6 porte cVar3 | OUI R(1) | `@0x1410a5b46` |
| 7a opt32 | OUI R(1)+opt R(32) | `@0x1410a5b63`, `@0x14080d6ce` |
| 7b R(2)+boucle | OUI R(2) + count x [R(1)+opt R(32)] | `@0x1424a3a4c`, `@0x1424a3afb`, `@0x1424a3b0c` |

Une seule feuille non etablie (la 4). Par la regle, `ti=40` n'entre pas dans la table. Le reste du
deser est bit-exact et pilote la mesure d'identite `MPPWord32`.

## 8. MESURE — L'ORACLE TRANCHE (2026-08-31, corrige apres retour superviseur)

### 8.1 Ce qui etait faux dans la premiere passe, et pourquoi

La premiere version de l'instrument reconnaissait un en-tete NEW `ti=40` (9 bits fixes) puis
deroulait le deser SANS gate de position selectif : le temoin fantome rendait 2057 records pour
526 dans la vraie bande — la reconnaissance mesurait du bruit, pas des creations. CAUSE RACINE =
FINDING B : `equipCreationWalk` gate i0 par `decodeWorldObjectPos` (porte 3 bits), inadapte a l'i0
dyn.-prec. de `ti=40` (porte 5 bits).

### 8.2 La correction (le walk parametre par le decodeur de position)

- `equipment_creation.go` : `equipCreationWalk` recoit deux champs, `posDecode` (defaut =
  decodeWorldObjectPos) et `posBits` — modif purement additive, `ti=37`/`ti=42` inchanges
  (non-regression : `ti=42` atterrit 118/119 = 99,2 %, temoins 0/119, identique a avant).
- `vehicle_creation.go` : `decodeBipedI0Pos` (porte 5 bits + rejet des quanta satures, bati sur les
  primitives de V1a : DequantBipedAxis, saturatedQuantum, le decoupage `DetectI0Layout` du film).
- L'INSTRUMENT durcit le gate : un i0 n'est accepte que s'il coincide avec une position REELLE de
  vehicule (nuage decode par `ScanFilmBipedPositionsForBand`, livre par V1a) — c'est l'oracle qui a
  valide `ti=42`, transpose en dyn.-prec.

### 8.3 Les chiffres, deux films de builds distincts

| mesure | `0d76e8f1` (Behemoth SF, 8 j) | `fccc61cd` (Launch Site SF) |
|---|---|---|
| decoupage i0 (lu DU film) | gate=5, 17/17/15 | gate=5, 17/17/15 |
| decoupage MPP (calibre via ti=37) | 9/5 | 9/5 |
| nuage positions reelles ti=40 | 32 246 -> 1 587 cellules | 5 431 -> 152 cellules |
| ancres NEW / ACCEPTES (vrai deser) | 526 / 31 | 206 / 11 |
| **TEMOIN FANTOME** | **1 / 31 = 3,2 % (PASS < 5 %)** | **0 / 11 = 0,0 % (PASS)** |
| temoins FAUX (ti37/ti36/ti38) | 3 / 0 / 3 (<= 9,7 %, PASS < 10 %) | 0 / 0 / 0 (PASS) |
| **V1.5 vies / valeurs / inconstantes** | **31 / 7 / 0** | **11 / 5 / 0** |
| V1.5 gate 1 constance 100 %/vie | PASS | PASS |
| V1.5 gate 2 cardinalite <= 8 | PASS (7) | PASS (5) |
| V1.5 gate 3 valeurs decouplees des vies | PASS (7 pour 31) | PASS (5 pour 11) |

### 8.4 Le resultat le plus fort : identifiant STABLE inter-build

**Cinq valeurs `MPPWord32` sur les sept de `0d76e8f1` reapparaissent sur `fccc61cd`** (autre carte,
empreinte de registre differente donc autre build) : `0x00fe32c0f4`, `0x005b80c406`, `0x000000254b`,
`0x00af31ab1a`, `0x00c6e79dcc`. Une valeur qui survit au changement de build et de carte n'est pas un
artefact de film : c'est un **GlobalID de tag**, exactement le comportement etabli pour `ti=37`
(`eqip`). L'hypothese `vehi` (voie A du cadrage) est donc SOUTENUE ; sa resolution en tag `vehi` par
`himap.ModuleIndex` (gate 4 / voie C) est l'etape suivante, non faite ici.

### 8.5 Robustesse au build et le WARN d'empreinte

Le log `WARN empreinte du registre ECS du film INCONNUE` est emis par le fingerprint global
(comparaison a UNE reference figee), PAS par l'instrument : celui-ci lit l'archetype (48 composants)
DANS le registre du film (`reg.Archetype(40)`) et le decoupage i0 DANS le film (`DetectI0Layout`).
Preuve de robustesse : deux films d'empreintes distinctes rendent des mesures coherentes. Le warning
est sans effet sur les mesures ; il n'est pas corrige (hors perimetre, chemin de production tiers).

### 8.6 Statut d'inscription — INCHANGE, et desormais fonde sur la mesure

Le port nominal (`bVar14 == 0`) est VALIDE par l'oracle : le vrai deser accepte des records dont i0
tombe sur une position reelle, les faux ~0. Mais le taux de capture (31/47, 11/21 vies) laisse une
part de creations non prises — coherent avec la feuille 4 config-dependante (`bVar14 == 1` deplace
i0, le record est rejete). **`ti=40` reste donc HORS de `defaultStateDeserByTI`** : le chemin
keyframe record-NEW y decode TOUS les records `ti=40` sans filtre, il ne tolererait pas ce que
l'oracle ecarte. La voie CREATION+oracle (`ScanFilmVehicleCreations`), elle, est validee.
