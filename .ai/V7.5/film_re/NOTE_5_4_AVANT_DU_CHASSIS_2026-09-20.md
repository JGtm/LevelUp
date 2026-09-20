# NOTE 5.4 — L AVANT DU CHASSIS : IL N EST PAS ECRIT, IL EST RECONSTRUIT

> Lot 5.4, branche `feat/decfilm-54`, base `6e86db356`. Ecrite le 2026-09-20.
> Source : decompile Ghidra en LECTURE SEULE (`127.0.0.1:8089`) + lecture memoire des constantes
> (`/read_memory`). Aucun film du cache n a ete lu pour etablir ce qui suit.

## 0. LA REPONSE, EN UNE PHRASE

Le composant `object-forward-and-up` ne porte pas deux vecteurs : il porte **une direction
unitaire** — le vecteur **HAUT** — **et un angle de roulis**, et le moteur fabrique l **AVANT**
en tournant une perpendiculaire autour du haut. Le depot lisait la direction et **jetait
l angle**. C est pour cela que le lot 5.2b.2 n a rien trouve : il mesurait l azimut du HAUT, qui
sur un chassis a plat est indetermine, et il jetait la seule valeur qui porte le cap.

## 1. CE QUE LE BRIEF VISAIT, ET POURQUOI C ETAIT LA MAUVAISE PORTE

Le brief pointait « la feuille 4 de l etat par defaut de `ti=40` — un quaternion derriere une
porte de flux ». **Les deux moities de cette phrase sont fausses**, et la mesure le dit :

- **Ce n est pas un quaternion.** La feuille 4 est `consumeVehicleMediaFrame` =
  `consumeSimStateHandleTail` + `consume140c1e79c`. La premiere porte `FUN_14076e494`, qui lit
  une **POSITION absolue** (porte d index, index de region, trois axes aux largeurs de la carte)
  — pas une rotation. La seconde porte l orientation, et sous la forme decrite au §2.
- **Ce n est pas la bonne cadence.** La feuille 4 n existe qu aux **images-cles**, et seulement
  quand la porte `bVar14` est posee (41,2 % des records, mesure de 5.1.7-b). L orientation vive du
  chassis est dans **`i2`, a chaque delta**.

La piste utile etait donc a cote : le meme encodage, dans le composant que 5.2b.2 avait deja
ouvert — mais dont il n avait lu que la moitie.

## 2. LA GRAMMAIRE, RELUE AU DECOMPILE

### 2.1 La chaine

```
FUN_140c5f7ec  (deser i2, ti=38/39/40/43)
    A = R(1) ; si A : mode = 2      sinon B = R(1)
    si level >= 2 : C = R(1) ; si C : mode = 1
    base = *(comp+0x10)
    B == 0 -> FUN_140c5f938(flux, base+0x18, base+0x24, mode)
    B == 1 -> FUN_140c5f8a8(flux, precedent, base+0x18, base+0x24, mode)
    puis FUN_140501798(base+0x18, base+0x24)      predicat d orthonormalite

FUN_140c5f938 / FUN_140c5f8a8  (aiguillage par mode)
    mode 1 : FUN_142e29bac                        R(1)[+R(30)] + R(30)
    mode 2 : 2 x FUN_1406d676c(0x60)              192 bits — JAMAIS emprunte
    sinon  : FUN_140c5fa84 (ou FUN_14076e744)     puis FUN_140c5f9c8

FUN_140c5fa84  (charge utile nominale)
    gate = R(1) -> [+0x2]
    gate == 0 : v = R(19) ; face = v / DAT_144708568 ; rem = v % ... ;
                iu = rem / DAT_14470856c ; iv = rem % ...   -> [+0x3], [+0x4], [+0x6]
    gate == 1 : face/iu/iv = DAT_144e0b240 / DAT_144e0b248
    puis angle = R(8)                                        -> [+0x0]

FUN_140c5f9c8  (0 bit — LE PONT)
    gate == 0 : dir = FUN_1406d8b98(face, iu, iv, 0x13)
    gate == 1 : dir = DAT_143b8f860 = (0, 0, 1)
    theta = raw * DAT_143cd891c - DAT_143cd8918 + DAT_143cd97a0
    FUN_1406d8678(&dir, theta, out = base+0x18)    la PERPENDICULAIRE
    *(base+0x24) = dir                              la DIRECTION LUE
```

### 2.2 Les constantes, relues dans le binaire

| Symbole | Valeur relue | Role |
|---|---|---|
| `DAT_143cd891c` | `0x3cc90fdb` = **pi/128** | pas de l angle sur 8 bits (un tour sur 256) |
| `DAT_143cd8918` | `0x40490fdb` = **+pi** | borne haute |
| `DAT_143cd8920` | `0xc0490fdb` = **-pi** | borne basse |
| `DAT_143cd97a0` | `0x3c490fdb` = **pi/256** | demi-pas (convention du milieu d intervalle) |
| `DAT_143cd84ec` | `-1,0` | cosinus court-circuite a `theta == +-pi` |
| `DAT_143cd837c` | `1,0e-4` | seuil de normalisation |
| `DAT_143b8f860` | **(0, 0, 1)** | direction PAR DEFAUT, porte posee |
| `PTR_DAT_14474c2f8` | -> `0x14472a63c` = **(1, 0, 0)** | premier axe de base |
| `PTR_DAT_14474c2e0` | -> `0x14472a648` = **(0, 1, 0)** | second axe de base |

L angle vaut donc exactement `dequantMidpoint(raw, n, -pi, +pi)`. Pour ce champ, la convention du
milieu d intervalle n est plus une convention **retenue** : elle est **mesuree**.

### 2.3 `FUN_1406d8678`, a la lettre

```
base = celui de (1,0,0) / (0,1,0) le MOINS aligne avec dir   (|dir.X| < |dir.Y| -> X)
branche X : v = dir x X          branche Y : v = Y x dir      (l ordre n est PAS symetrique)
si ||v|| >= 1e-4 : v /= ||v||
theta == +-pi : cos = -1, sin = 0      sinon : cos = cos(theta), sin = sin(theta)
out = v*cos + (dir x v)*sin + dir*(dir.v)*(1 - cos)          Rodrigues, sens direct
out /= ||out||
```

Le terme axial est nul en arithmetique exacte (`v` est perpendiculaire a `dir`) ; il est conserve
dans le port parce que l executable le calcule.

## 3. POURQUOI LA DIRECTION ECRITE EST LE HAUT — DEUX PREUVES, DONT UNE STRUCTURELLE

1. **Le defaut.** Porte posee, le moteur prend `(0, 0, 1)`. Un **avant** par defaut vertical
   n aurait aucun sens ; un **haut** par defaut vertical est l objet a plat. Cette preuve ne
   depend d aucun echantillon.
2. **La mesure de 5.2b.2**, qui devient lisible : |z| median **0,960** (19 bits) et **0,981**
   (30 bits) sur `4f77afc1`, 77 % et 94 % des echantillons au-dessus de 0,9. C etait le negatif ;
   c est desormais la confirmation.

Et la consequence, figee par un test sans film : **un chassis a plat a pour cap au sol l angle
lui-meme**. Avec `haut = (0,0,1)`, la base choisie est `(0,1,0) x haut = (1,0,0)`, et Rodrigues
autour de +Z rend `(cos theta, sin theta, 0)`. Le cap **EST** `theta`.

## 4. LES CHEMINS, ET CE QUI RESTE HORS D ATTEINTE

| Chemin | Direction | Angle | Absolu ? | Poids mesure (`4f77afc1`, 5.2b.2) |
|---|---|---|---|---|
| mode 0, B = 0 (`FUN_140c5fa84`) | R(1)[+R(19)] | R(8) | **oui** | 31 143 records |
| mode 1 (`FUN_142e29bac`) | R(1)[+R(30)] | R(30) | **oui** | **113 242** (dominant) |
| mode 0, B = 1 (`FUN_14076e744`) | quartets sur l etat precedent | `R(1)[+R(4)]` increment | **non** | inclus dans le mode 0 |
| mode 2 | 2 vec3 bruts | — | — | **0** (jamais emprunte) |

Le sous-chemin « delta » est le seul angle mort : il exige un registre par entite
(slot -> face/iu/iv/angle). Il est **compte, pas reconstruit** (`HasRoll` faux) — D2 (5.4).

**La cadence est celle des deltas**, donc `vehicles[].samples[].h` garde sa forme : pas de
lectures datees a publier, pas d interpolation, **pas de montee de schema**.

## 5. CE QUE LES MINI-BOBINES PEUVENT, ET CE QU ELLES NE PEUVENT PAS

- **Elles ne peuvent pas porter la preuve.** Leur nuage de positions `ti=40` est **vide sur les
  sept** (bande d images-cles pourtant non vide sur cinq) : ce sont des extraits d image-cle, et
  `i2` vit dans les deltas. L oracle du deplacement y est inaccessible.
- **Elles portent la feuille 4**, lue en valeur sur 63 records : 7 a plat, |z| median **0,577**
  (= 1/racine(3), un coin de face cubemap), 52 caps sur 63 dans un seul secteur de 30 deg.
  Population trop petite et trop biaisee pour conclure, et ce n est pas le chemin de production.
  Consigne en D4 (5.4), non conclu.

La preuve demande donc un film du cache — `4f77afc1` (build recent, mode 1 dominant) et
`a349fea8` (vieux build, mode 0 dominant) — et elle attend la voie libre du pilote.

## 6. CE QUI EST LIVRE

| Fichier | Contenu |
|---|---|
| `film/internal/grammar/orientation_frame.go` | port de `FUN_1406d8678`, `RollAngleFromRaw`, `AimVector`, `ChassisForwardVector` |
| `film/internal/grammar/orientation_frame_test.go` | les 4 invariants, sans film |
| `film/internal/grammar/components_dynprec_orientation.go` | `FwdUpDynPrec` rend l angle ; `Haut()` / `Avant()` ; largeurs par mode |
| `film/internal/grammar/components_object.go` | `decodeObjectForwardAndUp` rend la queue `R(8)` |
| `film/internal/grammar/offline_aim.go` | capture du roulis, du mode et du defaut |
| `film/internal/grammar/avant_chassis_54_research_test.go` | l instrument (tag `research`), oracle + temoin, pret pour la voie libre |

`grammar.Rev` monte a `grammar-2026-09-20.2` (la SORTIE de la couche change : le mode 1 pose
desormais `HasAim`/`AimRaw`). `facts.Rev` ne monte PAS — la couche des faits ne lit `i2` nulle
part. `SchemaVersion` reste 64, et le seul ecart des huit fixtures de contrat est la chaine
`grammarRev`.
