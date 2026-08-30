# RE LECTEUR de l'image-cle — passe de scouting BORNEE, verdict go/no-go

> Ecrit le 2026-08-27. Passe de reconnaissance sur le chemin LECTEUR de la grammaire
> d'etat complet (image-cle), jamais attaque de front : la serie R7-a..e attaquait
> l'ECRIVAIN (vtable +0x18) et le CADRE, plafonnant a 0,51-0,85 % de bit-exactitude sur
> le bipede (ti=35). L'utilisateur a rouvert la RE le 27/08. **Ce document est un livrable
> de FINDINGS + un verdict chiffre, PAS un port.** Aucun code Go ecrit, aucun `go build`.
>
> Canal : serveur **MCP Ghidra OPERATIONNEL** (le pont `mcp__ghidra__*` refuse par R6/R7
> repond de nouveau), `HaloInfinite.exe` build du 2026-06-04, image base `0x140000000`,
> 311 103 fonctions. Toutes les adresses ci-dessous sont verifiees ce jour par
> `decompile_function` / `read_memory` / `get_xrefs_to`. LECTURE SEULE stricte : aucun
> rename, aucun script, aucune analyse relancee.

---

## 0. Ce que la passe apporte de NEUF (resume executif)

1. **La chaine lecteur est confirmee au decompile pres** (Q1) : `FUN_1428e2a04` ->
   `FUN_1428e2a9c` -> `FUN_142e2bfd0` -> `FUN_1428e2b68` -> `FUN_142e2c690`. C'est bien le
   lecteur d'ETAT COMPLET, ARCHETYPE-AGNOSTIQUE, qui centralise le dispatch par composant.
2. **Fait neuf vs R6** : le lecteur d'etat complet est appele dans une branche explicitement
   nommee **"Film View"** (`FUN_142e2e104`), sous `FUN_142e2aab4`. R6 avait conclu « le
   film ne relit JAMAIS le type-2 ». Le demultiplexeur STREAMING (`FUN_1428e22c0`) le saute
   toujours, mais la mise en place d'une VUE DE FILM, elle, invoque le lecteur d'etat complet.
   Cela ne renverse pas R6 (la mesure R7-c/e tient, cf. §5), mais cela change le statut du
   lecteur : il a un site d'appel cote FILM.
3. **Le dispatch par composant lit ti=11 (l'objet d'objectif) A L'IDENTIQUE** (Q2) : meme
   thunk `FUN_14076ce9c` en `vtable[0x28]`, meme layout de descripteur que le bipede. Les
   cinq composants semantiques de l'objectif sont RESOLUS et DECOMPILES ce jour — et ils
   sont TRIVIAUX (des `R(32)` et un `R(3)`), sans aucun des constructs qui font deriver le
   bipede (pas de vec3 quantifie, pas de compte garde par un popcount RAM, pas de drapeau
   d'encodage).
4. **Verdict (Q3)** : GO, mais ETROIT et cible **ti=11 D'ABORD** — pas une reprise du
   bipede, pas ti=42 (deja fait). Raison : ti=11 est le test le plus PETIT, le plus PROPRE
   et le plus RENTABLE de la seule question qui reste ouverte (le CADRE reproduit-il sur le
   film ?), et il rend NATIFS le type d'objectif, la progression, l'etat et la reference au
   porteur physique — exactement la semantique VIP / Oddball / porteur d'objectif.

---

## 1. La chaine lecteur, decompilee (reponse Q1)

### 1.1 Carte des fonctions

| adresse | role | ce qu'elle lit / fait |
|---|---|---|
| `FUN_142e2aab4` | **mise en place de la vue** (appelant) | installe les 3 vues de replication (`+0x1b4c8`, `+0x1a0`, `+0x1b908`), puis `cVar2 = FUN_1428e2a04(&DAT_144c23178)` ; si succes, itere les entites lues (`local_68..local_60`, stride 200) via `FUN_142e2de40` |
| `FUN_1428e2a04` | **entree lecteur d'etat complet** | gate `FUN_14298924c(ctx+0x130, ...)` ; `FUN_142988338(ctx+0x130, hdr, 0x10, 0)` (en-tete 16 o) ; puis `FUN_1428e2a9c(ctx)` |
| `FUN_1428e2a9c` | **cadrage du paquet + BitReader** | `FUN_1429883ec(...)` ; relit `0x10` o d'en-tete ; lit `local_f4` octets de payload dans `*(ctx+0x240)` ; **gate `DAT_144db4330 == 0`** ; `FUN_1424c7b4c(br, buf, len)` construit le BitReader ; `cVar4 = FUN_142e2bfd0(br, param_5)` |
| `FUN_142e2bfd0` | **BOUCLE PAR ENTITE** | detail §1.2 |
| `FUN_1428e2b68` | **resolution d'archetype + cadrage de la table** | `plVar1 = *(DAT_144e61d88 + 8 + typeIndex*8)` ; `plVar1->vtable[0x10]()` = taille d'etat ; base des blocs registre = mode film ? `*(ctx+0x120)+0x130` : `*(ctx+0x108)` ; appelle `FUN_142e2c690(plVar1+1, br, ctx, typeIndex*0x4100 + 8 + base)` |
| `FUN_142e2c690` | **BOUCLE PAR COMPOSANT** | detail §1.3 |

Appelants remontes : `FUN_142e2aab4` <- `FUN_142e2e104` <- `FUN_14059ad80` <-
{`FUN_14059aea4`, `FUN_142a52190`}. **`FUN_142e2e104` cree la "Film View"** :

```
FUN_140b87098(2, &local_res8, 0, "Film View");
lVar3 = FUN_141f864ac(*(param_1 + 0x10), &local_res8);
if (lVar3 != 0) { FUN_142e2aab4(lVar3); ... }   // <- lecteur d'etat complet
```

### 1.2 `FUN_142e2bfd0` — la boucle par entite (le CADRE)

Par entite (structure de sortie de stride `0x32` uint = 200 octets), dans l'ordre lu au flux :

```
R(32) id              -> item[0x00]
R(32) typeIndex       -> item[0x04]
R(32)                 -> item[0x0c]
FUN_142e29cf8(br)     = R(4)             (R7-e : un seul quartet, PAS 2x4)
R(8)                  -> item+9 (octet)
si typeIndex != 0xffffffff :
    plVar2 = *(DAT_144e61d88 + 8 + typeIndex*8)          (descripteur d'archetype)
    R(32) n1
    si n1 > 0 :
        DAT_144e61ea0 = 1                                (PORTEE pleine precision)
        plVar2->vtable[0x60](plVar2, id, dst, br, 0)     = ETAT PAR DEFAUT
        si FUN_14076cea8() : R(32)                       (mot de controle mode film)
        DAT_144e61ea0 = 0
    R(32) n2
    si n2 > 0 :
        plVar2->vtable[0x88](...)                        = MASQUE PAR DEFAUT (0 bit)
        FUN_1428e2b68(&DAT_144c23178, br, typeIndex, id) = BOUCLE COMPOSANTS
avance item += 0x32 (200 o)
```

En-tete par entite = `32+32+32+4+8` = **108 bits** (confirme R7-e). La table de destination
est indexee par `typeIndex` dans le registre `DAT_144e61d88` : **le lecteur est
archetype-agnostique par construction** — rien n'est cable en dur sur le bipede.

### 1.3 `FUN_142e2c690` — la boucle par composant (le DISPATCH)

```
DAT_144e61ea0 = 1                                (portee pleine precision, TOUTE la boucle)
cVar6 = FUN_14076cea8()                          (drapeau corruption mode film)
pour k de 0 a 63, table += 0x104 :
    si *table != 0 :                             (entree active)
        plVar10 = comps[k]                       (chemin rapide : meme index)
        si nom(plVar10) != nom(table) :          (nom via vtable[0x08])
            recherche LINEAIRE par nom sur 64    (rattrapage de version)
            si introuvable -> return 0
        level = *(table + 0x100)                 (u32)
        cVar7 = plVar10->vtable[0x28](plVar10, br, ctx, &local, level)   = DESER COMPOSANT
        si cVar7 == 0 -> return 0
        si cVar6 && R(1) : R(32)                 (mot de controle par composant)
```

Trois faits qui tranchent (confirment R7-d/e) : **AUCUN masque de presence** (table FIXE de
64 entrees de `0x104` octets = le bloc d'archetype de `chunk_00`, `0x4100 = 64 x 0x104`) ;
**l'ordre suit la table**, descripteur retrouve par NOM en rattrapage ; le deser est
`vtable[0x28]` (thunk `FUN_14076ce9c` -> `vtable[0x30]`).

**Reponse Q1 : OUI.** `FUN_1428e2a04`/`FUN_1428e2a9c` est bien le lecteur d'etat complet des
enregistrements d'image-cle, et il expose un dispatch par archetype (registre
`DAT_144e61d88`) puis par composant (`vtable[0x28]`), uniforme.

---

## 2. Le dispatch lit-il ti=11 (l'objet d'objectif) ? (reponse Q2)

**OUI — structurellement ET concretement.** ti=11 = archetype **managed-objective** (le
suivi d'objectif du HUD, 34 composants). Le dispatch `FUN_142e2c690` ne connait pas le
bipede : il resout `DAT_144e61d88 + 8 + 11*8`, itere la table de 64 entrees de ti=11, et
appelle `vtable[0x28]` de chaque composant. Verifie ce jour : les descripteurs de composant
d'objectif ont **le meme layout de vtable que le bipede** (R7-d).

### 2.1 Les cinq composants semantiques, RESOLUS et DECOMPILES ce jour

Methode : `search_strings` sur le nom -> getter `vtable[0x08]` (xref DATA) -> vtable (xref
DATA du getter, base = slot-0x08) -> `read_memory` de la vtable -> deser en `+0x30`.

| i | composant | vtable | deser | ecrivain (+0x18) | **grammaire LUE** | dst |
|---|---|---|---|---|---|---|
| i3 | object-reference | `0x143d09118` | `FUN_142ed5550` | `FUN_142edb6a4` | **R(32)** | `+0x40` |
| i5 | type | `0x143d09078` | `FUN_1410fc4a4` -> `FUN_14080dec4` | `FUN_142edbb00` | **R(32)** | `+0x150` |
| i12 | progress | `0x143d08e90` | `FUN_142ed575c` | `FUN_142edb8c0` | **R(32)** | `+0x18c` |
| i13 | required-progress | `0x143d08e40` | `FUN_142ed5844` | `FUN_142edb960` | **R(32)** | `+0x190` |
| i14 | state | `0x143d08df0` | `FUN_142ed5948` -> `FUN_1424d9a30` | `FUN_142edba10` | **R(3)** | `+0x194` |

Toutes les vtables partagent le layout du bipede : dtor `0x14117b4a0`, ret-false
`0x1404ab600`, **thunk `0x14076ce9c`** (le MEME que le bipede), int3 `0x1411c8f80`. Seuls
`+0x08` (getter), `+0x18` (ecrivain) et `+0x30` (deser) different. Les getters sont un bloc
contigu (`0x141177ef0`-`0x141177fa0`), signature d'une meme famille d'archetype.

### 2.2 Pourquoi c'est le deblocage

- **i5 type** (`FUN_14080dec4` = R(32) inline, MSB-first, le meme modele de BitReader que
  `filmdec`) : drapeau/zone/colline/crane/noyau. Le nom "objective-type" est un parametre
  MORT (comme les noms de champ cote ecrivain, R7-d).
- **i12 progress** + **i13 required-progress** : R(32)/R(32) = le NUMERATEUR et le
  DENOMINATEUR de la capture (fraction de prise KOTH, temps de garde Oddball, etc.).
- **i14 state** (`FUN_1424d9a30` = R(3), 8 etats max) : l'etat VIVANT de l'objectif.
- **i3 object-reference** (R(32) -> `+0x40`) : **la reference vers l'objet physique** (le
  drapeau, le crane, le noyau). C'est le pont objectif-HUD -> entite portee, donc la voie
  native vers le PORTEUR.

**Aucun de ces cinq deser ne contient un construct fragile** : pas de vec3 quantifie aux
largeurs de carte, pas de compte garde par `FUN_1409fe718(state,0x49)` (popcount RAM), pas
de drapeau `DAT_144e61ea0`/`DAT_145121140`. Ce sont des lectures a largeur FIXE sur l'etat
du BitReader (`+0x28`/`+0x2c`/`+0x30`/`+0x38`/`+0x40`) que `filmdec.BitReader` implemente
deja au bit pres. **C'est l'oppose exact du bipede.**

---

## 3. Le lecteur est-il PLUS portable que l'ecrivain ? (reponse Q3)

**Reponse nuancee : ils sont EGALEMENT portables en FORME, et le lecteur a un seul avantage
operationnel — il est deja porte.**

- **Uniformite symetrique.** L'ecrivain est `vtable[0x18]`, le lecteur `vtable[0x28]` ->
  `vtable[0x30]` — deux cases uniques et uniformes de la MEME vtable. R7-d a etabli qu'ils
  se MIROITENT pour 4 composants du bipede sur 5. Pour ti=11, les feuilles sont si simples
  que lecteur == ecrivain trivialement (R(32) des deux cotes). Aucun des deux ne « bat »
  l'autre.
- **Le lecteur centralise le CADRE**, et ce cadre est **DEJA PORTE** : `FUN_142e2c690` est
  `keyframe_fullstate_loop.go` (R7-e, `WalkKeyframeFullState`), avec son harnais d'oracle.
  Ajouter ti=11 = porter ~5 feuilles TRIVIALES et REUTILISER la boucle. L'ecrivain n'offre
  pas cet acquis operationnel.
- **Mais le lecteur ne resout PAS le plafond de bit-exactitude.** Ce plafond (0,51 % sur le
  bipede) etait DANS les feuilles du bipede, pas dans le cadre (R7-e) — et le lecteur a
  DEJA ete porte pour le bipede. Reporter « le lecteur » pour ti=35 ne rend rien de neuf.

**Conclusion Q3** : le lecteur n'est pas categoriquement plus portable que l'ecrivain ; il
est le MEILLEUR VEHICULE parce qu'il est deja en place et archetype-agnostique. La question
de portabilite ne se joue pas lecteur-contre-ecrivain, elle se joue **archetype simple
(ti=11) contre archetype complexe (ti=35)** — et ti=11 gagne largement.

---

## 4. VERDICT go/no-go

### GO — etroit, cible ti=11, comme test decisif du cadre

**Porter ti=11 en PREMIER** via la boucle d'etat complet deja en place. Ordre des composants
(plus petit gain d'abord, tous des feuilles a largeur fixe deja mesurees ci-dessus) :

1. **i5 type** (R(32)) — la nature de l'objectif, une seule valeur, le gain le plus direct.
2. **i14 state** (R(3)) — l'etat vivant.
3. **i12 + i13 progress/required-progress** (R(32)/R(32)) — la fraction de capture.
4. **i3 object-reference** (R(32)) — le pont vers le porteur physique (VIP/Oddball/drapeau).

**Pourquoi GO** : (a) les feuilles sont triviales et sans source de derive ; (b) la valeur
produit est haute et directement alignee sur le chantier objectifs (type + progression +
etat + reference porteur = VIP et porteur d'objectif NATIFS) ; (c) c'est le test le plus
PROPRE de la seule question qui reste — le CADRE (en-tete 108 bits, ordre, mots de controle)
reproduit-il sur le payload type-2 du film ? Le bipede ne pouvait pas l'isoler (ses feuilles
derivent aussi). ti=11 l'isole : **si les feuilles triviales atterrissent, le cadre est
valide et l'objectif devient natif ; si elles n'atterrissent PAS malgre leur trivialite, le
mur est definitivement dans le CADRE/format, pas dans les deser** — ce qui contredirait le
« le cadre n'est pas la cause » de R7-e et fermerait la derniere hypothese ouverte. Les deux
issues sont decisives et bon marche (~5 deser + reutilisation de `WalkKeyframeFullState`).

### NO-GO explicites

- **NO-GO** sur une reprise du lecteur pour le bipede (ti=35) : deja porte, 0,51 %, derive
  dans les feuilles, cadre exonere (R7-e). Rien de neuf a en tirer.
- **NO-GO** sur la these « le lecteur bat l'ecrivain » : ce sont deux dispatches uniformes
  miroirs qui s'accordent. Ne pas construire de campagne sur cette premisse.
- **NO-GO** sur ti=42 (arme au sol) : deja resolu et branche (97,6 %,
  `default_state_ti42.go`).

### UNKNOWN BORNE (honnete, non mesure — `go build` interdit cette passe)

Je n'ai PAS mesure l'atterrissage bit-exact de ti=11 sur le film (build interdit). Le risque
residuel est ENTIEREMENT dans deux inconnues que seul le port + mesure trancheront :

1. **Le CADRE reproduit-il ?** en-tete 108 bits, ordre de la table, mots de controle —
   jamais valides sur un archetype SIMPLE. C'est precisement ce que ti=11 testerait.
2. **Provenance du flux `*(ctx+0x130)`** lu par `FUN_1428e2a04` : est-ce le bloc type-2 du
   film, ou un snapshot re-synthetise pour la "Film View" ? La mesure R7-c/e dit que la
   grammaire d'etat complet (portee `DAT_144e61ea0` = vec3 BRUT 96 bits) ne reproduit PAS le
   payload type-2 du film (position QUANTIFIEE, 102/57/62 bits). Ce fil statique unique reste
   a tirer SI la mesure ti=11 est ambigue ; il est hors perimetre de cette passe bornee.

En clair : la portabilite des FEUILLES ti=11 est ETABLIE (necessaire) ; la reproduction du
CADRE sur le film reste l'inconnue (suffisante), et ti=11 est l'instrument le moins cher pour
la lever.

---

## 5. Reconciliation avec R6 (le film « saute » le type-2)

R6 : le demultiplexeur STREAMING `FUN_1428e22c0` route le type-0 vers le decodeur de
replication et SAUTE le type-2 (le handler du type-1 avance le curseur par-dessus). Cela
tient. Cette passe ajoute un fait STATIQUE que R6 n'avait pas : la mise en place d'une **vue
de film** (`FUN_142e2e104`, branche "Film View") appelle `FUN_142e2aab4` -> le lecteur
d'etat complet. Il existe donc un site d'appel du lecteur d'etat complet cote FILM — ce qui
NUANCE « aucun consommateur » sans le renverser, puisque la mesure R7-c/e (grammaire d'etat
complet != payload type-2 au bit pres) reste valide. L'arbitre de la contradiction serait le
port ti=11.

---

## 6. Texte pret pour `.ai/thought_log.md` (NON ecrit par cette passe)

```
### [2026-08-27] RE lecteur image-cle — scouting borne : GO etroit sur ti=11 (objectif)

Statut : Complete (passe de reconnaissance, verdict go/no-go, aucun port, aucun build)

Decision technique. Passe Ghidra LECTURE SEULE (MCP de nouveau operationnel) sur le chemin
LECTEUR de l'etat complet, jamais attaque (R7-a..e attaquaient l'ecrivain/cadre, plafond
0,51-0,85 % sur le bipede). Chaine confirmee au decompile pres : FUN_1428e2a04 ->
FUN_1428e2a9c -> FUN_142e2bfd0 (boucle par entite, en-tete 108 bits, descripteur par
typeIndex dans DAT_144e61d88) -> FUN_1428e2b68 -> FUN_142e2c690 (boucle par composant, sans
masque, dispatch vtable[0x28] thunk FUN_14076ce9c). Fait neuf vs R6 : ce lecteur est appele
dans une branche "Film View" (FUN_142e2e104) ; le demux streaming saute toujours le type-2,
mais la vue de film, elle, invoque le lecteur d'etat complet. Le dispatch est
archetype-agnostique : ti=11 (managed-objective) est lu a l'identique. Ses cinq composants
semantiques sont resolus et decompiles ce jour, tous triviaux : i3 object-reference R(32)
(FUN_142ed5550), i5 type R(32) (FUN_1410fc4a4->FUN_14080dec4), i12 progress R(32)
(FUN_142ed575c), i13 required-progress R(32) (FUN_142ed5844), i14 state R(3)
(FUN_142ed5948->FUN_1424d9a30). Aucun vec3 quantifie, aucun compte garde RAM, aucun drapeau
d'encodage — l'oppose du bipede.

Resultats observes. Q1 : oui, c'est le lecteur d'etat complet, dispatch par archetype puis
par composant. Q2 : oui, le dispatch lit ti=11 a l'identique (meme thunk, meme layout de
vtable, deser resolus). Q3 : lecteur et ecrivain sont deux dispatches uniformes miroirs
(vtable[0x28] vs +0x18) qui s'accordent ; le lecteur n'est pas plus portable en soi, mais il
est DEJA porte (keyframe_fullstate_loop.go, R7-e) et archetype-agnostique, donc ajouter ti=11
= ~5 feuilles triviales reutilisant la boucle. Non mesure (build interdit) : la reproduction
du CADRE sur le film reste l'inconnue.

Conclusion / prochaine etape. GO etroit : porter ti=11 D'ABORD (ordre i5 type, i14 state,
i12/i13 progress, i3 object-reference), pas une reprise du bipede, pas ti=42 (fait). C'est le
test le moins cher et le plus decisif du cadre : feuilles triviales -> un echec localise le
mur dans le cadre/format, un succes rend l'objectif (type/progression/etat/porteur) NATIF.
NO-GO : reprise lecteur ti=35 (0,51 %, derive dans les feuilles), these lecteur>ecrivain.
```

## 7. Ligne pour `.ai/V7.5/REGISTRE_REPORTS.md` (NON ecrite par cette passe)

```
| 2026-08-27 | RE lecteur image-cle (scouting borne, jamais attaque avant) | GO ETROIT
chiffre : la chaine lecteur d'etat complet est confirmee au decompile (FUN_1428e2a04 ->
FUN_1428e2a9c -> FUN_142e2bfd0 -> FUN_1428e2b68 -> FUN_142e2c690), archetype-agnostique,
appelee en branche "Film View" (FUN_142e2e104) — nuance R6 sans le renverser. Le dispatch
par composant lit ti=11 (managed-objective) a l'identique du bipede : cinq deser resolus et
decompiles, TOUS triviaux (i3/i5/i12/i13 = R(32), i14 = R(3)), sans construct fragile.
Condition de reprise = PORTER ti=11 via la boucle d'etat complet deja en place
(keyframe_fullstate_loop.go), ordre i5->i14->i12/i13->i3, et MESURER l'atterrissage : c'est
le test le moins cher du cadre (en-tete 108 bits/ordre/controle), jamais isolable sur le
bipede. Inconnue bornee non mesuree (build interdit cette passe) : reproduction du cadre sur
le payload type-2, et provenance du flux *(ctx+0x130). NO-GO : reprise lecteur bipede ti=35
(deja porte, 0,51 %), these lecteur>ecrivain (dispatches miroirs qui s'accordent). |
```
