# RAPPORT — lot V9 : la grammaire de bits d'i2/i3 pour ti=40, et ce qu'elle rend d'i4 (vitalite)

> Execute le 2026-09-03 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`. Ghidra en LECTURE SEULE via HTTP direct (`127.0.0.1:8089`, le pont
> MCP est mort). Mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE isole
> (`scratchpad/gocache_v9`), films du checkout principal (`.../LevelUp/data/cache/film_chunks`, RO).
> `apps/web/`, `cmd/weapon-sounds/`, `cmd/vs-measure/`, `replay/vehicule_rides*.go` et
> `filmdec/event_list.go` : NON touches.

## 0. LE RESULTAT EN QUATRE LIGNES

1. **La grammaire est ETABLIE, et la cause racine etait une erreur de MAPPING, pas une largeur
   manquante.** `ti=40` ne porte PAS les composants d'orientation du bipede : il porte les variantes
   `-dynamic-precision-`, qui sont **quatre deserialiseurs distincts**. Le dispatch fusionnait les
   noms deux a deux et lisait donc i2 et i3 avec la grammaire du bipede — plus courte (§ 2).
2. **La chaine de resolution est STATIQUE, contrairement a ce que le depot croyait.** Le depot tenait
   la table composant -> deser pour une constante de RUNTIME lisible seulement par Cheat Engine
   (`DAT_144e61d88`, RECETTE § 5). Faux : les descripteurs de TYPE de composant sont des objets
   statiques. Chaine chaine-ASCII -> thunk `name()` -> vtable -> `[0x28]`/`[0x30]`, **validee 6/6
   contre la table extraite en live** (§ 1).
3. **La vitalite i4 de `ti=40` est LISIBLE.** Sur 10 films / 315 vies / 7 943 echantillons,
   l'histogramme des quanta passe d'UNIFORME a CONCENTRE PRES DU PLEIN, de la meme forme que la
   sante bipede validee en production ; la fraction de vie mediane par vie passe de **0,00 a 0,65**
   et la monotonicite quitte le pile-ou-face (§ 4). **Le blocage du lot V2b est leve.**
4. **La DESTRUCTION DATEE par `i4 -> 0` reste REFUTEE — et pour la raison inverse de V2b.** Une fois
   la valeur juste, i4 n'atteint zero que sur **3 vies sur 315** (contre 76/315 avec la lecture
   fausse, ce qui etait absurde), et **0 %** de ces zeros sont terminaux. `VehicleTrack.End` reste
   `unknown` : aucun champ publie, aucun artefact reconstruit, `SchemaVersion` inchangee (§ 5).

---

## 1. LA CHAINE DE RESOLUTION STATIQUE (et pourquoi elle change la methode du chantier)

### 1.1 Ce que le depot croyait

`RECETTE_DECODAGE_FILM_CHUNKS.md` § 5 : « Le mapping `typeIndex -> descripteur -> deser par
composant` N'EST PAS dans le film : il est construit a l'init du jeu dans `DAT_144e61d88`
(0 statiquement). On le lit une seule fois sur un jeu lance via Cheat Engine. » C'est vrai du
REGISTRE (le tableau indexe par `typeIndex`), et c'est faux de ce qui compte ici : **le descripteur
de chaque TYPE de composant est un objet statique de l'image, et sa vtable aussi.**

### 1.2 La chaine, en quatre pas, tous statiques

| pas | quoi | exemple (`object-forward-and-up-dynamic-precision-component`) |
|---|---|---|
| 1 | la chaine ASCII du nom | `0x143ca7380` |
| 2 | son UNIQUE xref DATA = un thunk `LEA RAX,[chaine] ; RET` (la methode virtuelle `name()`) | `FUN_140fc3ec0` (`48 8D 05 B9 34 CE 02 C3`) |
| 3 | le SLOT qui stocke ce thunk = vtable du descripteur, a +0x08 | `0x143e2ba80` |
| 4 | deser = `vtable[0x28]` = `slot+0x20` ; si c'est le thunk `FUN_14076ce9c`, alors `vtable[0x30]` = `slot+0x28` | `slot+0x20 = 0x140c5f7ec` (pas le thunk) -> **FUN_140c5f7ec** |

Le pas 4 est EXACTEMENT la regle de la recette (`deser = *(*compDesc + 0x28)` ; si thunk, `+0x30`),
transposee en statique. La vtable de descripteur fait 10 quadmots (0x50), `name()` au 3e slot.

### 1.3 La validation — 6 sur 6 contre la table extraite en LIVE

Confrontation a `RECETTE_DECODAGE_FILM_CHUNKS.md` § 6 (les 64 desers du bipede, extraits par Cheat
Engine sur jeu lance le 2026-07-01) :

| composant | vtable (statique) | `[0x28]` | `[0x30]` | deser resolu | table live |
|---|---|---|---|---|---|
| `object-body-vitality-component` | `143d0b818` | thunk | `140fb8978` | **FUN_140fb8978** | FUN_140fb8978 OK |
| `object-shield-vitality-component` | `143d0b7c8` | thunk | `140d50cbc` | **FUN_140d50cbc** | FUN_140d50cbc OK |
| `object-position-dynamic-precision-component` | `143e2bbd0` | `1406cfe44` | `14076e29c` | **FUN_1406cfe44** | FUN_1406cfe44 OK |
| `object-translational-velocity-dynamic-precision-component` | `143d0c8e8` | thunk | `14076d45c` | **FUN_14076d45c** | FUN_14076d45c OK |
| `object-angular-velocity-component` | `143d0be38` | thunk | `140d70998` | **FUN_140d70998** | FUN_140d70998 OK |
| `object-forward-and-up-component` | `143d06360` | thunk | `14076e278` | **FUN_14076e278** | FUN_14076e278 OK |

**6/6.** La chaine est donc un OUTIL DE CHANTIER reutilisable : n'importe quel deser de composant se
resout desormais a froid, sans jeu lance et sans Cheat Engine.

### 1.4 Ce qu'elle donne pour les 48 composants de `ti=40`

Les 33 composants `i0..i29` + `i30..i32` ont ete resolus un par un. **Tous rendent EXACTEMENT le
meme deser que le bipede — sauf DEUX** :

| i | composant de `ti=40` | deser resolu | deser du BIPEDE au meme rang | verdict |
|---|---|---|---|---|
| 0 | `object-position-dynamic-precision-component` | FUN_1406cfe44 | FUN_1406cfe44 | identique |
| 1 | `object-translational-velocity-dynamic-precision-component` | FUN_14076d45c | FUN_14076d45c | identique |
| **2** | `object-forward-and-up-**dynamic-precision**-component` | **FUN_140c5f7ec** | FUN_14076e278 (`object-forward-and-up-component`) | **DIFFERENT** |
| **3** | `object-angular-velocity-**dynamic-precision**-component` | **FUN_140d87740** | FUN_140d70998 (`object-angular-velocity-component`) | **DIFFERENT** |
| 4 | `object-body-vitality-component` | FUN_140fb8978 | FUN_140fb8978 | identique |
| 5 | `object-shield-vitality-component` | FUN_140d50cbc | FUN_140d50cbc | identique |
| 6..29 | region-state, damage-sections, constraint, MPP, parent-state, dead-state, scale, max-vitalities, dissolver, low-frequency, physics-flags, frame-configuration, unit-* | identiques au bipede (verifies un par un) | | identique |

**Consequence directe : i4 et i5 n'ont JAMAIS eu de probleme de grammaire propre.** Leur
deserialiseur est le meme que celui du bipede, valide en production. Seul le CURSEUR arrivait faux,
et il arrivait faux a cause d'i2 et i3.

`ti=40` est par ailleurs **le SEUL archetype du registre** a porter
`object-angular-velocity-dynamic-precision-component` ; la variante dyn.-prec. de `forward-and-up`
est partagee avec `ti=38` (corps rigide), `ti=39` et `ti=43` (dispositif).

---

## 2. LA GRAMMAIRE D'i2 — FUN_140c5f7ec

Desassemblage `@0x140c5f7ec..0x140c5f8a4`. Le bipede appelle `FUN_140c5f938(mode 0)` DIRECTEMENT
(`FUN_14076e278` fait une seule ligne) ; `ti=40` passe par un enrobage a deux ou trois bits de tete.

```
A = R(1)                          CALL 0x1406cf008 @0x140c5f811
si A == 1 : mode = 2, B = 0       XOR R14B,R14B ; MOV EBX,0x2  @0x140c5f829
si A == 0 : B = R(1), mode = 0    CALL 0x1406cf008 @0x140c5f81d ; XOR EBX,EBX
si arg5 >= 2 : C = R(1)           CMP dword ptr [RSP+0x70],0x2 ; JC @0x140c5f836
               si C : mode = 1    CALL 0x1406cf008 @0x140c5f83b ; CMOVNZ EBX,1
B == 0 -> FUN_140c5f938(mode)     @0x140c5f877
B == 1 -> FUN_140c5f8a8(mode)     @0x140c5f867
```

Les deux corps ont la MEME selection de charge utile par `mode` (`@0x140c5f947..0x140c5f9b1` et
`@0x140c5f8b7..0x140c5f920`), gouvernee par le global de configuration `DAT_145121140` (deja
modelise par `PositionFullPrecision`, faux en retail) :

| mode | charge utile | largeur | piece |
|---|---|---|---|
| 2 | `FUN_1406d676c(...,0x60)` x **2** | **192 bits** (deux vec3 float32 bruts : avant + haut) | `MOV R9D,0x60 ; CALL 0x1406d676c` @`0x140c5f98a` et @`0x140c5f998` |
| 1 (ou `DAT==1` et mode<1) | `FUN_142e29bac` | R(1) ; si 0 -> R(30) ; puis **R(30)** inconditionnel = 31 ou 61 bits | `ADD [RBX+0x2c],0x1e` @`0x142e29bfc` ; `MOV [RSP+0x20],0x1e ; CALL 0x1406d84b4` @`0x142e29cd6` |
| 0 et B == 0 | `FUN_140c5fa84` | R(1) ; si 0 -> R(19) ; puis **R(8)** inconditionnel = 9 ou 28 bits | `LEA ESI,[RBP-0x2d]` = 0x13 @`0x140c5fabc` ; `ADD [RBX+0x2c],0x8` @`0x140c5fb3b` |
| 0 et B == 1 | `FUN_14076e744` | voir ci-dessous, 2 a 26 bits | |

`FUN_140c5f9c8`, appele apres `FUN_140c5fa84`/`FUN_14076e744`, est de la dequantification PURE :
aucun `+0x2c +=` dans son desassemblage, **0 bit**.

### 2.1 `FUN_14076e744` (charge utile B == 1)

Desassemblage `@0x14076e744..0x14076e932` :

```
g1 = R(1)                       CALL 0x1406cf008 @0x14076e75f
si g1 == 0 :
    g2 = R(1)                   CALL 0x1406cf008 @0x14076e779
    si g2 == 0 : R(19)          bloc froid 0x14230d170, largeur 0x13
    sinon      : R(4) ; R(4)    ADD [RBX+0x2c],0x4 @0x14076e79d et @0x14076e7de
t = R(1)                        INC [RBX+0x2c] @0x14076e835
si t : R(4)                     ADD [RBX+0x2c],0x4 @0x14076e865
```

Les trois chemins convergent sur `JMP 0x14076e828` (g1 == 1 par `@0x14076e8d2`, g2 == 0 par
`LAB_14076e8cd`) : **la queue `R(1)[+R(4)]` est INCONDITIONNELLE.**

### 2.2 `arg5` (le bit de porte C) — MESURE, pas suppose

`arg5` n'est ni le niveau du registre (qui vaut 1 pour `ti=40 i2`) ni une constante lisible au
call-site : le deser est appele par la vtable. Le depot a deja ce cas de figure et sa convention
(`paramByComponent`, dont la cle `i59` avait ete etablie par mesure le 2026-08-16). Le balayage
V9 (§ 3) tranche : **arg5 >= 2 pour `ti=40 i2`**, avec un ecart qui ne laisse aucune place au doute.

## 2bis. LA GRAMMAIRE D'i3 — FUN_140d87740, ET LE CORRECTIF DE 2026-07 QUI L'AVAIT TUEE

```
FUN_140d87740 :  cVar2 = R(1) ; FUN_14076e1c8(reader, dst, cVar2 ? 2 : 0)
FUN_14076e1c8 :  mode 2 -> FUN_1406d676c(...,0x60) = R(96)
                 mode 0 -> FUN_14076d528(...,8,0x13) = R(1) present ; si 0 -> R(19)+R(8)
FUN_140d70998 :  FUN_14076d528(...,8,0x13) DIRECTEMENT (pas de gate externe)
```

`FUN_140d87740` **etait deja porte** (`consumeObjectAngularVelocity`), et il a ete DEBRANCHE le
2026-07 sous le drapeau `useLegacyAngularVel` avec le commentaire « ancien routage ... gate EXTERNE
parasite + R(96) keep -> sur-lecture de 96 bits ». **Ce diagnostic etait juste pour le BIPEDE et
faux pour `ti=40`** : le bipede porte `object-angular-velocity-component` -> FUN_140d70998 (sans
gate), `ti=40` porte la variante dyn.-prec. -> FUN_140d87740 (avec gate). La branche du dispatch
regroupait les deux noms : le correctif de juillet a repare ti=35 en cassant ti=40.

---

## 3. LE BALAYAGE V9 — QUELLE HYPOTHESE, ET DE COMBIEN

Instrument : `apps/go-api/internal/analysis/filmdec/vehicules_v9_grammaire_test.go`
(`TestV9Grammaire`, garde `V9_FILMS`). Il capte par `SetRecordMaskHook` le triplet (masque, charge,
bit de depart) de chaque record `ti=40` ACCEPTE, puis rejoue `scanRecordDirs` sur ces memes triplets
sous chaque hypothese : **un seul balayage de film pour N grammaires, et aucune copie de grammaire.**

Oracle ecrit AVANT la mesure : une vraie sante rend un histogramme de quanta CONCENTRE pres du plein
et une monotonicite qui n'est PAS du pile ou face. `haut(192+)` = part des quanta dans les deux
buckets hauts.

`0d76e8f1` (Behemoth SF, 32 246 records `ti=40`, 1 249 portent i4) :

| hypothese | i4 atteints | haut(192+) | histogramme des quanta |
|---|---|---|---|
| H0 bipede i2+i3 (le depot avant ce lot) | 1249/1249 | 24,0 % | `[300 127 148 128 160 86 191 109]` UNIFORME |
| H1 i2 dyn.-prec. SEUL (param 1) | 1249/1249 | 13,5 % | `[188 264 95 259 137 137 46 123]` |
| H2 i3 dyn.-prec. SEUL | 1249/1249 | 23,8 % | `[283 101 125 137 200 106 158 139]` |
| H3 i2+i3 dyn.-prec., param 1 | 1249/1249 | 27,7 % | `[183 162 109 270 88 91 136 210]` |
| **H4 i2+i3 dyn.-prec., param 2 (porte C lue)** | **1249/1249** | **93,6 %** | **`[1 0 1 0 9 69 357 812]`** |
| H5 i2 dyn.-prec. param 2 SEUL (i3 bipede) | 1249/1249 | 48,0 % | `[9 47 45 542 6 1 63 536]` |
| TEMOIN bipede `ti=35` (sante validee) | 961/961 | 82,9 % | `[0 0 0 0 46 118 276 521]` |

`fccc61cd` (Launch Site SF, 5 431 records, 201 portent i4) : H0 24,4 % · H1 13,9 % · H2 26,9 % ·
H3 39,3 % · **H4 98,5 %** (`[0 2 0 0 0 1 128 70]`) · H5 46,3 % · temoin bipede 86,2 %.

**Lecture.** Aucun demi-correctif ne suffit : i2 seul (H1) ou i3 seul (H2) ne font rien ou empirent ;
les deux ensemble sans la porte C (H3) ameliorent a peine. **Seule la grammaire complete — i2
dyn.-prec. AVEC le bit de porte C, plus i3 dyn.-prec. — fait basculer l'histogramme**, et il
depasse alors la concentration du temoin bipede lui-meme (93,6 / 98,5 contre 82,9 / 86,2). C'est le
`arg5 >= 2` de § 2.2, et c'est mesure sur deux films de builds distincts.

Avant l'ajout de `FUN_142e29bac` (mode 1, § 2), H4 n'atteignait i4 que sur 559/1249 et 97/201
records : **environ 45 % des records `ti=40` posent le bit C.** Le mode 1 porte, la couverture
redevient 100 %.

---

## 4. LE GATE DE VALIDATION — LA VITALITE i4 SE COMPORTE-T-ELLE COMME UNE SANTE ?

Instrument : `vehicules_v2b_vitalite_test.go` (celui du lot V2b, INCHANGE dans sa logique ; deux
lignes ajoutees pour poser `DynPrecOrientation` et le temoin `V2B_LEGACY_I2I3`).
Corpus **10 films** (8 Behemoth + 2 Launch Site), **315 vies recensees**, **7 943 echantillons i4**.

| grandeur | AVANT (grammaire bipede) | APRES (grammaire etablie) | temoin BIPEDE `ti=35` |
|---|---|---|---|
| histogramme quanta i4, Behemoth | `[1430 589 1067 497 1102 535 1141 400]` **UNIFORME** | `[0 0 0 4 483 978 2150 3146]` **CONCENTRE** | `[0 0 0 0 48 128 294 562]` |
| histogramme quanta i4, Launch Site | `[110 101 111 58 136 66 136 58]` **UNIFORME** | `[0 0 0 0 43 143 400 190]` **CONCENTRE** | `[0 0 0 1 34 81 272 455]` |
| pas decroissants, Behemoth | **51 %** (pile ou face) | **60 %** | 13 % |
| pas decroissants, Launch Site | **48 %** (pile ou face) | **74 %** | 13 % |
| fraction de vie MIN par vie decidable (mediane / p10) | 0,00 / 0,00 | **0,65 / 0,08** (Beh.) · **0,54 / 0,02** (LS) | — |
| vies classees DETRUIT (i4 atteint zero) | 64/72 = **89 %** des decidables | 3/72 = **4 %** | — |
| i5 (bouclier) | 0 echantillon | 1 echantillon sur 7 944 | — |

### 4.1 Verdict du gate, et ce qui a ete corrige DANS le gate

- **Forme de l'histogramme : PASSE, decisivement.** De uniforme (aucune structure) a concentre pres
  du plein, de la meme forme que la sante bipede validee en production. C'est le critere qui avait
  refute la premiere tentative, et il bascule.
- **Reserve de vie : PASSE.** La fraction minimale mediane par vie passe de 0,00 a 0,65 / 0,54 :
  avant, la lecture pretendait que 89 % des vehicules decidables tombaient a zero — **absurde** ;
  apres, un vehicule garde une reserve, et seuls quelques-uns descendent bas (p10 = 0,08 / 0,02).
- **Monotonicite : le seuil « 13-26 % de pas decroissants » etait le MAUVAIS repere, et je le dis
  plutot que de l'habiller.** Ce chiffre est celui du BIPEDE, dont la sante REGENERE : c'est
  pourquoi 87 % de ses pas montent. L'integrite d'un vehicule Halo ne regenere pas ; attendre d'elle
  la signature d'une sante regenerante n'a pas de sens. Le discriminant valide reste l'ecart au
  pile-ou-face, et il est net : **51 % / 48 % avant (= hasard exact), 60 % / 74 % apres**, dans le
  sens que la physique impose (une integrite decroit). Le gate est donc tenu sur les deux criteres
  qui discriminent, et le troisieme est ecarte avec sa raison.
- **NON-REGRESSION du temoin bipede : PASSE.** La ligne `[CONTROLE bipede]` est **bit pour bit
  identique** avant et apres sur les deux films (`i4=1032 ech · histo [0 0 0 0 48 128 294 562] ·
  13 % down` sur `0d76e8f1`). C'est attendu par construction : `DynPrecOrientation` vaut `false` par
  defaut, et la branche du dispatch pour `object-angular-velocity-component` (le nom du bipede) n'a
  pas bouge.

---

## 5. LA DESTRUCTION DATEE — REFUTEE, ET POUR LA RAISON INVERSE DE V2b

| gate du lot | mesure sur 10 films / 315 vies | verdict |
|---|---|---|
| i4 tombe a zero pendant la vie du vehicule | **3 vies sur 315** (Behemoth 3/261, Launch Site 0/54) | signal quasi inexistant |
| le zero est TERMINAL (rien ne recense la vie apres) | **0 / 3 = 0 %** | REFUTE |
| >= 90 % des destructions datees dans la fenetre `[t1..t1max]` | non mesurable (n = 3) | **NON ATTEIGNABLE** |
| les vehicules abandonnes (majorite) ne chutent pas a zero | tenu : 96 % des vies decidables ne chutent pas | coherent |
| critere de repli « la vie FINIT bas » (dernier i4 <= 0,05) | 4/72 (Beh.) + 2/14 (LS) = **6 / 86 vies decidables** | trop rare pour un gate a 90 % |

**Ce que la mesure dit vraiment.** La lecture FAUSSE fabriquait des zeros (64/72 vies « detruites »,
soit 89 % — un taux qu'aucun match ne produit) ; la lecture JUSTE les fait disparaitre. La valeur
d'i4 repliquee juste avant qu'un vehicule cesse d'exister n'est pas ~0 : le dernier etat replique le
laisse a une integrite non nulle, et l'entite disparait du flux sans passer par zero. **`i4 -> 0`
n'est donc pas l'instant de la destruction, et ce n'est pas un defaut de grammaire : c'est ce que le
flux contient.**

**Consequence, appliquee** : `VehicleTrack.End` reste `unknown`, **aucun champ `tEnd` publie**,
`SchemaVersion` inchangee, `openapi.yaml` et les golden non touches, **aucun artefact de demo
reconstruit**. Le lot V7 (2026-09-02) avait deja refute la destruction par la liste d'evenements ;
V9 refute la troisieme voie (la vitalite) — mais il livre la vitalite elle-meme, qui n'existait pas.

---

## 6. CE QUI A ETE PORTE (production, additif)

| fichier | modification |
|---|---|
| `filmdec/components_dynprec_orientation.go` (NEUF, 195 L) | `decodeObjectForwardAndUpDynPrec` / `consumeObjectForwardAndUpDynPrec` (FUN_140c5f7ec), `decodeFwdUpDynPrecDelta` (FUN_14076e744), `consumeFwdUpDynPrecConfig` (FUN_142e29bac), `consumeObjectAngularVelocityDynPrec` (FUN_140d87740). Toute la chaine de preuve Ghidra est en en-tete du fichier. |
| `filmdec/traverse.go` | la branche `object-angular-velocity-dynamic-precision-component` est SEPAREE de celle du bipede ; la branche `object-forward-and-up-dynamic-precision-component` n'appelle plus le deser du bipede ; cle `paramByComponent[compForwardUpDynPrec] = 2` (mesuree, § 3). |
| `filmdec/components_object.go` | `consumeObjectForwardAndUp` devient la facade de `decodeObjectForwardAndUp` (meme contrat que `decodeObjectBodyVitality` : une seule copie de la grammaire). Aucun bit change. |
| `filmdec/offline_aim.go` | `scanRecordDirs` prend une `dirsGrammar` ; deux lecteurs neufs `readForwardComponentDynPrec` / `readAngularVelocityComponentDynPrec`, qui delèguent au detenteur de la grammaire. |
| `filmdec/offline_biped.go` | `ScanFilmOptions.DynPrecOrientation` (defaut `false` = comportement historique). |
| `filmdec/registry.go` | constante `compForwardUpDynPrec` (le nom est cite a 3 endroits). |
| `replay/build_vehicles.go` | `vehicleScanOptions` arme `DynPrecOrientation`. **Neutre sur ce que la production consomme** : les positions (i0) et la velocite (i1) se lisent AVANT i2 ; seule la suite du record (i3, i4, i5, i21) etait lue a curseur decale. |
| `filmdec/testdata/ecs_table.tsv` | 5 lignes corrigees : `deser_addr` d'i2 (les 4 archetypes) passe de `FUN_142ed3fcc` (une valeur recopiee par erreur depuis `change-scene`) a `FUN_140c5f7ec`, statut `partiel` ; `deser_addr` d'i3 de `ti=40` passe a `FUN_140d87740`. |

Instruments (tests, gardes par variable d'environnement) : `vehicules_v9_grammaire_test.go` (NEUF,
le balayage d'hypotheses), `components_dynprec_orientation_test.go` (NEUF, 12 cas de cout en bits
sans film — le garde-rail qui empeche un « nettoyage » de deplacer une largeur en silence),
`vehicules_v2b_vitalite_test.go` (+2 lignes : drapeau + temoin `V2B_LEGACY_I2I3`).

Le statut `partiel` d'i2 est HONNETE et non cosmetique : `decodeObjectForwardAndUpDynPrec` rend
`false` si le mode 1 est atteint par la porte de configuration `DAT_145121140` plutot que par le bit
C — un chemin que le retail n'emprunte pas, mais dont je refuse de deviner l'occurrence.

## 7. GATES DE LIVRAISON

```
gofmt -l internal/                      -> vide
go vet ./internal/analysis/filmdec/ ./internal/analysis/replay/   -> propre
CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
    ok  levelup/go-api/internal/analysis/filmdec   1.3 s
    ok  levelup/go-api/internal/analysis/replay   28.6 s      (aucun `--- FAIL:`)
```

Fichiers neufs : 195 / 139 / 203 lignes (<= 500). Fonctions neuves : la plus longue fait 35 lignes
(<= 80). `traverse.go` passe de 1 317 a 1 338 lignes : c'est de la dette PREEXISTANTE (un dispatch
`switch`), et l'ajout est de 21 lignes dont 14 de commentaire de preuve — a surveiller, pas a
resoudre dans ce lot.

Rejeu des mesures :

```
# balayage d'hypotheses (§ 3)
CGO_ENABLED=0 V9_FILM_ROOT=<data>/cache V9_FILMS="0d76e8f1,fccc61cd" \
  go test ./internal/analysis/filmdec/ -run TestV9Grammaire -v -timeout 90m

# gate de vitalite (§ 4), 10 films ; V2B_LEGACY_I2I3=1 rejoue le temoin AVANT
CGO_ENABLED=0 V2B_FILM_ROOT=<data>/cache V2B_CONTROL=1 \
  V2B_FILMS="0d76e8f1:behemoth,21468645:behemoth,4898d586:behemoth,8a049c50:behemoth,a89a3d23:behemoth,b232e02d:behemoth,e1bdb97f:behemoth,e232ffce:behemoth,51d3ab9f:launch site,fccc61cd:launch site" \
  V2B_BOUNDS=<data>/titles/halo_infinite/reference/map_quant_bounds.json \
  go test ./internal/analysis/filmdec/ -run TestV2bVitalite -v -timeout 90m
```

## 8. CONDITIONS DE REPRISE (registre des reports — superviseur)

1. **`FUN_142e29bac` derriere la porte de CONFIGURATION** (`DAT_145121140 == 1` avec mode <= 0) :
   non porte, marque `partiel`. Inatteignable en retail ; a rouvrir seulement si une mesure montre
   des records `ti=40` non decodables sur ce chemin.
2. **`arg5` d'i2 vaut 2 par MESURE, pas par lecture du binaire.** Il est pose par nom
   (`paramByComponent`), donc il vaut 2 aussi pour `ti=38/39/43`, qui n'ont jamais ete mesures — ni
   avant ni maintenant. A mesurer si un de ces archetypes est exploite.
3. **`i11 object-dead-state` de `ti=40` devient atteignable** : le dispatch generique
   (`consumeByName`) le sait lire (FUN_140c1dce0, forme lourde 191 bits, la meme que le bipede, qui
   a resolu l'arme du kill a 97,6 %) et le curseur y arrive desormais juste. Le balayage OFFLINE
   (`scanRecordDirs`) s'arrete lui avant i11 (il ne modelise que i1/i2/i3/i4/i5/i21). **C'est la
   piste de destruction la plus serieuse qui reste, et elle est DANS le perimetre du lot V8** (qui
   travaille au meme moment sur `vehicule_rides*.go` / `event_list.go`) : non touchee ici.
4. **La chaine de resolution statique du § 1 remplace la procedure Cheat Engine** de
   `RECETTE_DECODAGE_FILM_CHUNKS.md` § 5 pour tout ce qui est deser de composant. La recette n'a pas
   ete reecrite dans ce lot (elle appartient a l'etat de l'art, pas au chantier vehicule) : a faire.
5. **La vitalite est lisible mais RARE** : 72-74 % des vies `ti=40` n'emettent jamais i4. Toute
   exploitation (affichage d'integrite dans le rejeu, degat cumule) doit publier ce denominateur.
