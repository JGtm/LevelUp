# ti=35 (bipede) — grammaires des quatre composants `partiel` (2026-09-17)

> Preparation du lot 3.6 (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, « porter les composants
> manquants archetype par archetype »), groupe **ti35** : les quatre `partiel` de la table
> (`i57`, `i59`, `i60`, `i63`) et les deux largeurs laissees ouvertes par
> `NOTE_3_6_TI35_2026-09-16.md` (`FUN_140c1e79c`, `FUN_14076e494`). Sans production : aucun
> film decode, aucun test, aucun fichier Go touche. Source : `HaloInfinite.exe` par Ghidra en
> LECTURE SEULE (HTTP direct `127.0.0.1:8089`, decompile / disassemble / read_memory /
> xrefs / search_strings). Methode : `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` (nom a
> `descripteur+0x18`, ecrivain a `descripteur+0x40`).
>
> Instrument / Ghidra lecture seule. Archetype `ti=35` (bipede). Groupe ti35.
>
> Toutes les adresses, largeurs et constantes de cette note sont COLLEES depuis Ghidra. Les
> lignes marquees « depot » citent le code Go existant pour dire ce que le port changera ;
> elles ne sont pas des mesures.

---

## 0. Resume — six faits qui changent la table

1. **`FUN_140c1e79c`** (champ 6 de `i60`) : `R(1)` ; si le bit vaut **0** : `R(19)` (code
   cubemap de direction) ; puis **toujours** `R(8)` (angle dans `[-pi, +pi]`). **9 ou 28
   bits.** Il ecrit DEUX vecteurs (`+0x38` la direction, `+0x2c` une perpendiculaire) — ce
   sont eux que le predicat teste, et non les blocs `4 x R(16)` (correction du §2.7 de la
   note du 16).
2. **`FUN_14076e494(flux, dst, 0x10, 0, 0, 0)`** : `0x10` est une **classe de precision**
   (ligne 16 d'une table de 32 lignes), pas une largeur. La queue vaut `R(1)` porte ; porte
   a **1** : `3 x R(22)` **constants** (bornes par defaut +-20000, loi de `FUN_140be9b88`) ;
   porte a **0** : `R(ceil_log2(nb_regions))` index de region puis `3 x R(W_axe)` aux
   largeurs de la region de la CARTE ; ou `R(96)` brut si la garde runtime `FUN_14076f91c`
   est levee (0 bit du flux ; deux globals statiquement a 0 ; le depot la modelise deja par
   `fullPrecisionGate`).
3. **`i57`, branche etiquette 3** : les « octets d'etat RUNTIME » de la table sont un
   **`R(6)` DU FLUX** (`FUN_14297ea84` ecrit `param_1+2` ; disasm `142f26308..142f26312`).
   La branche est **entierement decidable hors ligne**.
4. **`i63`** : le « masque RAM de 73 bits absent du flux » est le **bloc `3 x R(32)` lu en
   tete du composant** (`FUN_142f21b10` ecrit `etat+0..+0xc` ; `FUN_1409fe718(etat, 0x49)`
   le compte). **Compte de la boucle 2 = popcount(m0) + popcount(m1) + popcount(m2 & 0x1ff).**
5. **`i63`** : les etiquettes `>= 6` ne consomment PAS zero bit. Le dispatch est une chaine :
   `FUN_141fd4814` (0..5) -> `FUN_142ef01c4` (6..11) -> `FUN_142ef0538` (12..17) ->
   `FUN_142ef0388` (18..23) -> `FUN_142ef0494` (>= 24, non ouverte). Les corps 6..16 sont
   releves ici.
6. **`i59`** : la grammaire MESUREE du depot (`components_biped_anchor.go`) est un **cas
   particulier** de celle de l'ecrivain : reference d'entite absente + carte a une seule
   region + references de domaine 5/0 absentes. Les 8 records « drapeaux != 000 » sont
   exactement les autres branches de l'ecrivain.

Bilan : `i57`, `i59`, `i63` sont **decidables hors ligne** ; `i60` l'etait deja. Le seul
« non elucide » restant est la queue de la chaine de dispatch d'`i63` (etiquettes 17..31),
nommee mais pas ouverte jusqu'au bout.

---

## 1. Les descripteurs (chaine des quatre pas, verifiee)

| i | Composant | Chaine | Accesseur de nom | Slot du nom (`+0x18`) | Descripteur | `+0x40` lu | Ecrivain effectif |
|---|---|---|---|---|---|---|---|
| 57 | `biped-spartan-ability-component` | `0x143c98d98` (depot) | `0x141177530` (depot) | `0x143d0ccb0` | `0x143d0cc98` | `0x142f02810` | `FUN_142f02810` -> `FUN_142f268c4(ctx+0x12e4)` |
| 59 | `biped-spartan-ability-non-predicted-state` | `0x143c99048` (depot) | `0x141177500` (depot) | `0x143d0cd08` | `0x143d0ccf0` | `0x142f02994` | `FUN_142f02994` -> `FUN_142f2679c(ctx+0x1324)` |
| 60 | `simulation-state-component` | `0x143c993e8` | `0x14119e480` | `0x143d0b358` **et** `0x143d0c938` | `0x143d0b340` **et** `0x143d0c920` | `0x142f02444` / `0x142f02434` | deux thunks `MOV RCX,[R8+0x10]; ADD RCX,0x850 / 0xa48; JMP FUN_142ed6d88` (octets `49 8b 48 10 48 81 c1 50 08 00 00 e9 34 49 fd ff` / `... 48 0a 00 00 e9 44 49 fd ff`) |
| 63 | `biped-action-component` | `0x143c98cf0` | `0x141177560` | `0x143d0ce08` | `0x143d0cdf0` | `0x142f027f4` | `FUN_142f027f4` -> `FUN_142f26a20(ctx+0xaa8)` |

Les « vtables » `0x143d0ccb0` / `0x143d0cd08` des commentaires Go sont les SLOTS DU NOM ; les
descripteurs sont 0x18 plus bas et leur `+0x40` rend bien l'ecrivain de la table (lu en
memoire : `10 28 f0 42 01` et `94 29 f0 42 01`). `i60` a DEUX descripteurs qui menent au meme
ecrivain avec deux offsets d'etat (`0x850` et `0xa48`) : meme grammaire.

Convention d'appel des ecrivains (disasm `FUN_142f02810`, `FUN_142f02994`, `FUN_142f027f4`) :
`RCX` = descripteur, `RDX` = flux, `R8` = contexte (`[R8+0x10]` = base d'etat de l'entite,
`byte [R8+0x38]` = drapeau de contexte), `R9D` = `param_4` (le « niveau » du composant, que
le depot mesure a 2 pour `i57`/`i59` : `component_param4.go`). `i57` transmet `R9D` sans le
toucher ; `i63` ne le lit pas.

---

## 2. Primitives communes (toutes lues dans le desassemblage)

Le flux est `param_2` ; `+0x2c` est le compteur de bits (chaque `ADD [..+0x2c], N` = champ de
N bits), `+0x30` l'accumulateur, `+0x38` la reserve, `+0x40` le curseur d'octets.

| Primitive | Grammaire | Valeur / notes |
|---|---|---|
| `FUN_1406cf008(flux)` | `R(1)` | — |
| `FUN_1406d6c7c(flux, n)` | `R(n)` chemin lent | — |
| `FUN_1407f2058(flux)` | `R(1)` ; si **0** -> `R(5)` ; si 1 -> `0xffffffff` | porte INVERSEE (sondage E2) |
| `FUN_1407f08bc(flux, out)` | `R(1)` ; si **1** -> `FUN_1407f08f8` = `R(8)` ; sinon `0xffff` | « gate8 » |
| `FUN_14297ea84(flux, ?, out)` | `R(6)` -> `*out` | drapeaux |
| `FUN_142f21c0c(flux, flux, out)` | `R(3)` ; `*out = v + 1` | sous-type 1..8 |
| `FUN_142f21cf0(flux, flux, out)` | `R(2)` ; `*out = v - 1` | etiquette |
| `FUN_142af27f8(flux, ?, out)` | `R(2)` | — |
| `FUN_1424d9a30(flux, ?, out)` | `R(3)` | — |
| `FUN_140fc147c(flux, ?, out)` | `R(3)` | queue d'`i59` |
| `FUN_14076e304(flux, out)` | `R(2)` ; `*out = v & 3` | boucle 2 d'`i63` |
| `FUN_14080dec4(flux, flux, out)` / `FUN_141d0f344` / `FUN_14318d788` (1er champ) | `R(32)` | — |
| `FUN_142f21b10(flux, flux, out)` | `3 x R(32)` -> `out[0..2]` | boucle `for (p=out; p != out+3; p++)` |
| `FUN_1406d676c(flux, flux, out, n)` | `R(n)` brut par paquets de 64 puis reste | `n = 0x60` -> 96 bits (3 x float32 bruts) |
| `FUN_1406d310c(n)` | 0 bit | `ceil(log2(n))` ; `FUN_1406d310c(4) = 2` |
| `FUN_1406d3140(?, flux, dom, out)` | `[R(1) si dom == 1]` puis `R(w_dom)` puis `R(2)` ; `*out = (gen << 30) \| (base + v)` | `w_dom = ceil_log2(cardinal)` : domaines 0/1/7/8 = 13 au defaut, 2/3/5 = 8, 4/6 = 9 (`NOTE_BANDE_SLOTS_BIPEDE_2026-09-16.md` §2.3) ; la sonde `R(1)` du domaine 1, si 1, bascule sur l'entree 4 (`DAT_1451f98f0/f4`, 9 bits) ; garde `DAT_144706104` = bit du film (`varwidth.go`) |
| `FUN_1408f0ac4(out, flux, dom)` | `R(1)` ; si **1** -> `FUN_1406d3140(dom)` ; puis `FUN_1406cb0cc` (globals `DAT_1445a78a0`, 0 bit) et `FUN_1405d5dbc` (table, 0 bit) | reference d'entite optionnelle |
| `FUN_1406d84b4(flux, ?, min=XMM2, max=XMM3, n=[RSP+0x20], f6=[RSP+0x28], f7=[RSP+0x30])` (offsets vus de l'appelant) | `R(n)` | `D = 2^n` (`D-1` si `f6`). `f7 == 0` : `x = v*(max-min)/D + min + 0,5*(max-min)/D`. `f7 != 0` : `v == 0 -> min` ; `v == D-1 -> max` ; sinon `pas = (max-min)/(D-2)`, `x = (v-1)*pas + min + 0,5*pas`. Si `f6` et `2v == D-1` : `x = (min+max)*0,5` (double `DAT_143cd8910`). `DAT_143cd84b0 = 0x3f000000 = 0,5` |
| `FUN_142ee2194(flux, ?, out)` | `FUN_1406d84b4(n=0x10, f6=0, f7=1, min=DAT_143cd8f84=-100,0, max=DAT_143cd84a8=+100,0)` = **`R(16)`** | composante de vecteur |
| `FUN_1406d8288(code, out, n)` | 0 bit | cubemap : `F = DAT_1447084d0[8n]`, `N = DAT_1447084d4[8n]` ; `face = code / F`, `rem = code % F`, `iu = rem / N`, `iv = rem % N`, `pas = 2/(N-1)` (`DAT_143cd8934 = 2,0`), `a = iu*pas - 1 + pas/2` (0 si `2iu == N-2`), `b` idem ; faces 0..5 = `(1,a,b) (a,1,b) (a,b,1) (-1,a,b) (a,-1,b) (a,b,-1)`, sinon `DAT_143cf7758 = (0,0,1)` ; normalisation si norme `>= DAT_143cd837c = 1,0e-4`. Statique pour n=19 : `F = 0x15555 = 87381 = floor(2^19/6)`, `N = 1` (peuple au runtime — le depot le porte deja en dur, `aim_vector.go`) |
| `FUN_14076dc04(flux, ?, out, n)` | `R(n)` puis `FUN_1406d8288(v, out, n)` | direction unitaire sur n bits |
| `FUN_14076d6dc(flux, ?, min, max, n)` | `R(n)` | magnitude log : `v == 0 -> min` ; `v == 2^n-1 -> max` ; sinon `a = 1 - min`, `L = ln(a + max)/2^n`, `x = exp(v*L + 0,5*L) - a` |
| `FUN_14076d528(flux, flux, out, min=XMM3, max=[RSP+0x20], nmag=[RSP+0x28], ndir=[RSP+0x30])` | `R(1)` ; si **0** : `R(ndir)` direction (`FUN_1406d8288`) puis `R(nmag)` magnitude (`FUN_14076d6dc`) ; si 1 : `out = *PTR_DAT_14474c2f0` = `0x143b61708` = `(0,0,0)` | ordre direction PUIS magnitude (disasm `14076d5c2` -> `14076d5e1`) |
| `FUN_142f26e9c(out, flux, max=XMM3)` | `FUN_14076d528(min=DAT_143cd839c=0,1, max, nmag=0xc, ndir=0x18)` = `R(1)[0 -> R(24) + R(12)]` | disasm `142f26ea4..142f26ec9` : `[RAX-0x28]=0x18`, `[RAX-0x30]=0xc`, `[RAX-0x38]=XMM3` |
| `FUN_140c1e924(flux, out, k, n)` | `FUN_140c1e9d4` = `3 x R(n)` puis `FUN_140c1e978` dequantifie chaque axe dans la ligne `k` de `0x143b8c6f0` (`0x18` octets par ligne, 3 paires min/max) | lignes lues : 0 = `+-3,0`, 1 = `+-0,7`, 2 = `+-100,0` ; ligne 3 = octets non flottants (`00 00 00 04 ...`) donc `k` attendu dans 0..2. `x = q*(max-min)/2^n + min + 0,5*(max-min)/2^n` |
| `FUN_1409fe718(p, nbits)` | 0 bit | popcount des mots 32 bits de `p` : mots pleins `0..(nbits+31)/32-2`, dernier mot masque par `0xffffffff >> (32 - nbits%32)` ; pour `0x49 = 73` : `pop(p[0]) + pop(p[1]) + pop(p[2] & 0x1ff)` |
| `FUN_142ef4db8(entree)` | 0 bit | destructeur virtuel selon l'octet `entree+0x28` |
| `FUN_140f03dfc`, `FUN_140f03e58`, `FUN_14058c250`, `FUN_1404d5f44` | 0 bit | remises a l'etat par defaut (`DAT_143b8c970 = (0,0,0)`) |

### 2.1 La position absolue : `FUN_14076e494` (la « queue handle »)

```
FUN_14076e494(flux, dst, classe, 0, ?, p6) :
  si FUN_14076f91c() != 0                         ; DAT_144e61ea0 != 0 || DAT_145121140 == 1
     -> FUN_1411b259c = FUN_1406d676c(flux, flux, dst, 0x60)     = R(96) brut
  sinon si p6 != 0 -> FUN_141f85880 (non emprunte : p6 = 0 a tous les sites de ti=35)
  sinon -> FUN_14076e524(dst, flux, &idx, classe)
```

`FUN_14076e524(dst, flux, &idx, classe)` (disasm `14076e524..14076e742`) :

| champ | largeur | condition | dependance de config |
|---|---|---|---|
| porte `r` | `R(1)` | toujours | — |
| index de region `idx` | `R(DAT_144632be0)` | `r == 0` | **`DAT_144632be0`** = `1` si une seule region, sinon `FUN_1406d310c(nb_regions)` (`FUN_140be9a14`, `140be9b4c..b60`) ; statique `0` |
| `q_x, q_y, q_z` | `R(W_x)`, `R(W_y)`, `R(W_z)` (`FUN_140cc5128`) | toujours | `r == 0` : `W = int[3]` a `0x1445ccbe0 + 12*(32*idx + classe)` ; `r == 1` : `W` a `0x1445cc9e0 + 12*classe` (= `0x1445ccaa0` pour la classe 16) |

Dequantification (`14076e61b..14076e66c`) : `x_i = q_i * (max_i - min_i) / 2^W_i + min_i + 0,5 * (max_i - min_i) / 2^W_i`
avec les bornes de la region (`0x14462cbe0 + 24*idx`, 3 paires min/max) ou par defaut
(`0x1445cc9c8` = `-20000,0 / +20000,0` x3, copie de `0x143b8c6b8`).

**La loi des largeurs** (`FUN_140be9b88(classe, bornes, ?, out_w)` + `FUN_140be9c78(classe)`) :

- `prec = FUN_140be9c78(classe)` : `classe <= 16` -> `2^(16-classe) * DAT_143cd9758` ;
  sinon `DAT_143cd9758 / 2^(classe-16)` ; **`DAT_143cd9758 = 0x3c088889 = 1/120`**. Pour la
  classe 16 : `prec = 1/120`, `2*prec = 1/60`.
- si `prec < 1,0e-4` (`DAT_143cd837c`) : `W = 26` sur les trois axes ;
- sinon, par axe : `etendue = max - min` ; si `etendue < 2*prec * DAT_143cd975c`
  (`DAT_143cd975c = 0x4a800000 = 4194304 = 2^22`) alors `bins = ceil(etendue / (2*prec))`
  sinon `bins = 0x400000` ; **`W = min(26, ceil_log2(bins))`**.
- Classe 16 : `W = min(26, ceil_log2(ceil(60 * etendue)))` — c'est mot pour mot la loi que
  `MapQuantEntry.AxisWidths` porte deja (`map_bounds.go`).
- **Bornes par defaut +-20000** : `etendue = 40000 < 69905,07` -> `bins = 2 400 000` ->
  `2^21 < 2 400 000 <= 2^22` -> **`W = 22`** sur les trois axes. La branche `r == 1` vaut donc
  **`1 + 66 = 67` bits, constante**, sans dependance de carte.
- Les bornes de region viennent de la structure de niveau chargee : `FUN_140be9a14` lit
  `DAT_144976b60 + 0x7bc` (nombre de regions), `+0x7ac` (tableau, pas `0xdc`), AABB a `+0x44`
  (6 floats) — le « world bounds » du BSP que le depot fige dans son catalogue par carte.

La garde `FUN_14076f91c` est un etat runtime (0 bit du flux, statique `0`/`0`) ; le depot la
modelise deja (`fullPrecisionGate`). Le `p5` (5e argument) n'est jamais lu par
`FUN_14076e494` (aucun acces `[RSP+0x60]` dans son disasm) : le drapeau de contexte
`ctx[0x38]` qu'`i57`/`i59` y poussent est **sans effet**.

---

## 3. `i57 biped-spartan-ability-component`

- Index `57`, descripteur `0x143d0cc98`, ecrivain `FUN_142f02810` -> `FUN_142f268c4(etat = ctx+0x12e4, flux, drapeau = ctx[0x38], param_4 = R9D)`.

| champ | largeur (bits) | condition | dependance de config |
|---|---|---|---|
| etiquette brute `e` ; stockee `e - 1` en `etat+3` | `R(2)` — `FUN_1406d310c(4) = 2` si `param_4 < 2`, `FUN_142f21cf0` = `R(2)` sinon | toujours | **aucune** : les deux branches lisent 2 bits |
| `e == 1` (`etat+3 == 0`) : `FUN_142f25d78(etat+0xc)` — sous-etiquette | `R(2)` -> ushort `+0xc` | `e == 1` | — |
| `e == 1` : direction | `R(24)` (`FUN_14076dc04(..., R9D = 0x18)` -> `FUN_1406d8288`) -> `+0x0` (12 octets) | `e == 1`, **inconditionnel** (le `!= 0xffff` porte sur 2 bits) | table cubemap n=24 (valeur seulement) |
| `e == 3` (`etat+3 == 2`) **et `param_4 > 1`** : `FUN_142f262d4(etat+0x1c, flux, drapeau)` | voir ci-dessous | `e == 3` | `param_4` (depot : 2) |
| `e == 0` ou `e == 2` : rien | 0 | | |

`FUN_142f262d4(p, flux, drapeau)` (decompile + disasm `142f262d4..142f263a8`) :

| champ | largeur | condition | dependance |
|---|---|---|---|
| `FUN_140f03dfc(p)` | 0 | | remise a zero |
| `a` -> `p[0]` | `R(1)` | toujours | |
| `f` -> `p[2]` | **`R(6)`** (`FUN_14297ea84`, `R8 = p+2`) | `a == 1` | **lu dans le flux** (et non en RAM) |
| `c` | `R(1)` | `a == 1 && (f & 1)` | |
| reference d'entite -> `p+0x14` | `FUN_1406d3140(?, flux, dom = 0, p+0x14)` = `R(w0) + R(2)` (`w0` = 13 au defaut) | `c == 1` | cardinal du domaine 0 (`DAT_1451f98d4`, garde `DAT_144706104`) |
| fin de sous-branche (pas de `FUN_142f04664`) | 0 | `c == 1 && (f & 0x10) == 0` | |
| `FUN_142f04664(p+4, flux, c, drapeau)` | voir §3.1 | `c == 0`, ou `c == 1 && (f & 0x10)` | |
| `t` -> `p[1]` | `R(1)` | toujours (apres la sous-branche, `a` quel qu'il soit) | |
| position absolue -> `p+0x18` | `FUN_14076e494(flux, p+0x18, 0x10, 0, drapeau, 0)` (§2.1) | `t == 1` | largeurs d'axe de la carte |

### 3.1 `FUN_142f04664(dst, flux, c, drapeau)` — position absolue OU relative (partage avec `i59`)

| champ | largeur | condition | dependance |
|---|---|---|---|
| position absolue -> `dst+0` | `FUN_14076e494(flux, dst, 0x10, 0, drapeau, 0)` (§2.1) | `c == 0` | largeurs d'axe de la carte |
| `k` -> `dst+0xe` | `R(2)` | `c == 1` | |
| position relative -> `dst+0` | **`3 x R(13)`** (`FUN_140c1e924(flux, dst, R8 = k, R9D = 0xd)` ; disasm `142f04745..142f04755` : `R8B` est la valeur `k` qui vient d'etre lue, pas le drapeau de contexte) | `c == 1` | plage = ligne `k` de `0x143b8c6f0` (valeur seulement) |
| `g` | `R(1)` | `c == 1` | |
| `-> dst+0xc` | `R(16)` (sinon `0xffff`) | `c == 1 && g == 1` | |

Largeurs d'`i57` : `e ∈ {0,2}` -> **2** ; `e == 1` -> **28** ; `e == 3` -> `2 + 1 + [6 + 1 + ...] + 1 + [position]`.
Le port du depot (`consumeSpartanAbilityTag3`) rend `false` des que `a == 1` en invoquant un
« octet d'etat runtime » : **cet octet est le `R(6)` du flux**. Statut : **releve** (complet).

---

## 4. `i59 biped-spartan-ability-non-predicted-state`

- Index `59`, descripteur `0x143d0ccf0`, ecrivain `FUN_142f02994` -> `FUN_142f2679c(etat = ctx+0x1324, flux, drapeau = ctx[0x38])` puis, si `param_4 > 1`, `FUN_140fc147c(flux, ?, ctx+0x1328)` = `R(3)`.

| champ | largeur | condition | dependance |
|---|---|---|---|
| etiquette brute `e` ; stockee `e - 1` en `etat+0` | `R(2)` (`FUN_1406d310c(4)`) | toujours | — |
| corps du grappin `FUN_142f25e90(etat+0x14, flux, drapeau)` | voir ci-dessous | `e == 3` (`e - 1 == 2`) | |
| queue | `R(3)` (`FUN_140fc147c`) -> `etat+4` | `param_4 > 1` | `param_4` (depot : 2) |

`FUN_142f25e90(p, flux, drapeau)` (decompile + disasm `142f25e90..142f262d3`) :

| ordre | champ | largeur | condition | dependance |
|---|---|---|---|---|
| 1 | sous-type `s` -> `p+0x4e` | **`R(3) + 1`** (`FUN_142f21c0c`) ; `s ∈ 1..8` — la branche « `s == 0` -> `FUN_140f03e58` » est morte | toujours | |
| 2a | `r` : reference presente ? -> `p+0x54` | `R(1)` (`FUN_1408f0ac4(p+0x50, flux, dom = 1)`) | toujours | |
| 2b | reference -> `p+0x54` | `FUN_1406d3140(dom 1)` : sonde `R(1)` ; si 1 -> `R(9)` (entree 4), sinon `R(13)` au defaut ; puis `R(2)` | `r == 1` | table de domaines |
| 2c | `FUN_142f04664(p+0x58, flux, c = (p+0x54 != -1) = r, drapeau)` | `c == 0` : position ABSOLUE (§2.1) ; `c == 1` : `R(2) k + 3 x R(13) + R(1) g [+ R(16)]` (§3.1) | toujours | largeurs d'axe de la carte si `r == 0` |
| 3 | drapeaux `f` -> `p+0x7a` | **`R(6)`** (`FUN_14297ea84`, `R8 = p+0x7a`) | toujours | |
| 4 | corps selon `s` | voir ci-dessous | | |

Corps selon `s` (adresses de branche du disasm) :

| `s` | branche | champs dans l'ordre |
|---|---|---|
| 1 | `142f26293` | `p[0..3] = -1` ; `gate8` -> `p+0x76` (`R(1) [+ R(8)]`) |
| 2 | `142f26278` | `FUN_1408f0ac4(p+8, dom 5)` = `R(1) [+ R(8) + R(2)]` ; `gate8` -> `p+0x76` |
| 3 | `142f260d5` | `FUN_1408f0ac4(p+0, dom 0)` = `R(1) [+ R(13) + R(2)]` ; `FUN_1408f0ac4(p+8, dom 5)` = `R(1) [+ R(8) + R(2)]` ; `FUN_142f26e9c(max = DAT_143cd8394 = 30,0)` -> `p+0x10` ; `FUN_142f26e9c(max = DAT_143cd84a8 = 100,0)` -> `p+0x1c` ; `FUN_142f26e9c(max = DAT_143cd8374 = 1,0)` -> `p+0x28` (chacun `R(1) [0 -> R(24) + R(12)]`, min `0,1`) ; `R(24)` (`FUN_14076dc04(p+0x68, 0x18)`) ; `R(9)` -> `v - 1` -> `FUN_140809d94(p+0x74, DAT_144976b50, v-1)` (0 bit, table runtime de validation) |
| 4, 5 | `142f25ff8` | `p[0..1] = -1` ; `FUN_1408f0ac4(p+8, dom 5)` ; `FUN_142f26e9c(max = 1,0)` -> `p+0x28` ; **position absolue** `FUN_14076e494(flux, p+0x34, 0x10, 0, 0, 0)` ; `R(24)` -> `p+0x40` ; `R(9)` -> `p+0x4c` |
| 6 | `142f25f2d` | `gate8` -> `p+0x78` ; si `p+0x78 == 0xffff` (porte fermee) -> `FUN_1408f0ac4(p+8, dom 5)`, sinon `p[2..3] = -1` ; `FUN_1408f0ac4(p+0, dom 0)` ; `FUN_142f26e9c(30,0)` -> `p+0x10` ; `FUN_142f26e9c(100,0)` -> `p+0x1c` ; `R(1)` -> `p+0x4f` ; `R(24)` -> `p+0x68` ; `p+0x74 = 0xffff` |
| 7, 8 | `142f262b9` | rien |

**Lecture croisee avec la grammaire MESUREE du depot** (`components_biped_anchor.go`) :
`Inner = R(3)` brut -> `s = Inner + 1` (1 -> `s = 2` « leger », 2 -> `s = 3` « lourd ») ;
`Zero3 = 000` = `r = 0` + porte de region `0` + index de region sur `IndexW = 1` bit ;
`R(Wx) R(Wy) R(Wz)` = la position absolue de 2c ; `Mid7` = `f` (6 bits) + le `R(1)` de
`FUN_1408f0ac4(p+8, dom 5)` a 0 (`s = 2`) ou de `FUN_1408f0ac4(p+0, dom 0)` a 0 (`s = 3`) ;
`gate8` = pour `s = 2` la porte `FUN_1407f08bc`, pour `s = 3` la porte de
`FUN_1408f0ac4(p+8, dom 5)` (qui, ouverte, lirait `R(8) + R(2)` et non `R(8)`). Les 8 records
« `001` / `100` / `110` » sont : `001` = index de region 1 (carte a deux regions) ; `100` =
`r = 1`, reference sur 13 bits + position RELATIVE `R(2) + 3 x R(13) + R(1)[+R(16)]`
(les « deux tailles independantes de la carte ») ; `110` = `r = 1`, sonde a 1, reference sur
9 bits. La magnitude des vecteurs, « plage inconnue » au depot, est `[0,1 ; 30]`,
`[0,1 ; 100]`, `[0,1 ; 1]` (loi log de `FUN_14076d6dc`).

Statut : **releve** (complet, 8 sous-types). Les sous-types 4..8 et la branche `r == 1` ne
sont pas dans le corpus mesure du depot ; ils sont dans l'ecrivain.

---

## 5. `i60 simulation-state-component`

- Index `60`, descripteurs `0x143d0b340` et `0x143d0c920`, ecrivain `FUN_142ed6d88(etat, flux)` (deux thunks, offsets d'etat `0x850` / `0xa48`).

| ordre | champ | largeur | condition | dependance de config |
|---|---|---|---|---|
| 1 | drapeau -> `etat+0x28` | `R(1)` | toujours | — |
| — | `FUN_14058c250(etat)` (defaut) et retour 1 | 0 | drapeau `== 0` | — |
| 2 | deux index d'entite -> `etat+0`, `+4` | `2 x FUN_1407f2058` = `2 x (R(1) [0 -> R(5)])` | drapeau `== 1` | — |
| 3 | quatre composantes -> `etat+0x08..+0x14` | `4 x R(16)` (`FUN_142ee2194`, `[-100, +100]`) | | — |
| 4 | deux octets -> `etat+0x29`, `+0x2a` | `R(2)`, `R(2)` (inline, `142ed6e1d` / `142ed6ed2`) | | — |
| 5 | quatre composantes -> `etat+0x18..+0x24` | `4 x R(16)` | | — |
| 6 | `FUN_140c1e79c(flux, R8 = etat+0x2c, R9 = etat+0x38)` | `R(1)` ; si **0** -> `R(19)` ; puis `R(8)` — **9 ou 28 bits** | | table cubemap n=19 (valeur) |
| 7 | `FUN_140501798(etat+0x2c, etat+0x38)` | 0 | | — |
| 8 | position absolue -> `etat+0x44` | `FUN_14076e494(flux, etat+0x44, 0x10, 0, 0, 0)` = `R(1)` ; porte 1 -> `3 x R(22)` ; porte 0 -> `R(IndexW) + R(W_x) + R(W_y) + R(W_z)` | predicat vrai | **largeurs d'axe de la region de la carte** (§2.1) |
| 9 | `FUN_140492128(etat+0x44)` : trois floats finis | 0 | | — |

Detail du champ 6, `FUN_140c1e79c` (disasm `140c1e79c..140c1e921`) :

- `R(1)` ; si le bit vaut **1** : `etat+0x38 = *PTR_DAT_14474c2e8` = `0x14472a654` =
  `(0, 0, 1)` ; si **0** : `R(19)` -> `FUN_1406d8288(v, etat+0x38, 0x13)` = direction unitaire.
- puis, toujours, `FUN_1406d84b4(n = 8 [RSP+0x20], f6 = 0, f7 = 0, min = XMM2 = DAT_143cd8920 = 0xc0490fdb = -pi, max = XMM3 = DAT_143cd8918 = 0x40490fdb = +pi)` = **`R(8)`**, angle `theta`.
- `FUN_1406d8678(dir = etat+0x38, theta, out = etat+0x2c)` (0 bit) : choisit entre
  `(0,1,0)` (`0x14472a648`) et `(1,0,0)` (`0x14472a63c`) l'axe le moins aligne avec `dir`,
  produit vectoriel, normalisation (seuil `1,0e-4`), rotation de Rodrigues d'angle `theta`
  autour de `dir` (`theta == +-pi` -> `cos = DAT_143cd84ec = -1,0`, `sin = 0`), puis
  `FUN_1404fec88` (normalisation).
- Le predicat `FUN_140501798` teste `| |v1|^2 - 1 | < 1,0e-3` (`DAT_143cd84bc`),
  `| |v2|^2 - 1 | < 1,0e-3`, `| v1.v2 | < 1,0e-3`, exposants non `0x7f800000`. `v1` est la
  perpendiculaire construite, `v2` la direction : **vrai par construction** des que la
  direction decodee a une norme `>= 1,0e-4` (une direction cubemap l'a toujours ; le defaut
  `(0,0,1)` aussi). Le seul echec possible est un `NaN` d'entree, que le lecteur hors ligne
  peut reproduire a l'identique puisque le predicat ne lit que les bits deja decodes et cinq
  constantes de `.rdata`.

Largeur totale : drapeau 0 -> **1** ; drapeau 1 -> `1 + (2..12) + 64 + 4 + 64 + (9..28) + (1 + 66 | 1 + IndexW + W_x + W_y + W_z)` soit **211..240** sur la branche « porte a 1 » et `145 + IndexW + W_x + W_y + W_z` .. `174 + ...` sur la branche « region ».

Statut : **releve** (complet). Le depot porte deja la grammaire (`consumeSimulationState`,
`consume140c1e79c`, `consumeSimStateHandleTail`) ; le kill-switch `simStateComplete = false`
tient a la SOURCE des largeurs d'axe, pas a la grammaire (§7).

---

## 6. `i63 biped-action-component`

- Index `63`, descripteur `0x143d0cdf0`, ecrivain `FUN_142f027f4` -> `FUN_142f26a20(etat = ctx+0xaa8, flux)`.

| ordre | champ | largeur | condition | dependance |
|---|---|---|---|---|
| 0 | `etat[0..0xc] -> etat[0xc..0x18]` (sauvegarde du masque precedent) | 0 | | |
| 1 | **masque `m0, m1, m2`** -> `etat+0, +4, +8` | **`3 x R(32)`** (`FUN_142f21b10(flux, flux, R8 = etat)`, disasm `142f26a50..142f26a56`) | toujours | — |
| 2 | `n1` | `R(4)` (inline, `142f26a77`) | toujours | — |
| 3 | `n1` fois : `R(7)` -> `entree+0x30` ; `FUN_142ef4c98(&entree, flux)` -> `FUN_142ef1734` : **`t = R(5)`** puis dispatch `FUN_141fd4814(t, {flux, &entree})` | voir §6.1 | `n1 > 0` | — |
| 4 | `n2 = FUN_1409fe718(etat, 0x49)` = `pop(m0) + pop(m1) + pop(m2 & 0x1ff)` | 0 | | **calcule sur le masque du pas 1**, pas sur de la RAM externe |
| 5 | `n2` fois : `R(1)` ; si 1 -> `R(2)` (`FUN_14076e304`) sinon `0xff` -> `etat+0x740+k` | `n2 x (1 [+ 2])` | `n2 > 0` | — |
| 6 | second bloc -> `etat+0x728` | `3 x R(32)` (`FUN_142f21b10`, appel de queue `142f26cd7`) | toujours | — |

Le decompile est sans ambiguite : `FUN_142f21b10(param_2, param_2, param_1)` ecrit
`param_1[0..2]` puis `FUN_1409fe718(param_1, 0x49)` lit `param_1[0..2]`. Le compte de la
boucle 2 est une **fonction pure des 96 premiers bits du composant**. La table disait
« masque RAM absent du flux » et le depot fige `bipedActionLoop2Count = 0` : c'est le trou
qui explique que `i63` « franchisse la frontiere du record » quand `n2 > 0`.

### 6.1 Le dispatch des etiquettes `t` (chaine de quatre fonctions)

Chaque corps commence par `FUN_142ef4db8(entree)` (0 bit) et une remise a zero de l'entree.

| `t` | fonction | grammaire (dans l'ordre) | statut |
|---|---|---|---|
| 0 | inline `FUN_141fd4814` | `FUN_1408f0ac4(dom 0)` = `R(1) [+ R(13) + R(2)]` ; `gate8` | releve |
| 1 | `FUN_143193fe0` | `gate8` ; `R(8)` ; `R(8)` ; `R(32)` ; `FUN_1431a3a50` = `R(15)` (`FUN_1406d84b4(n = 0xf, f6 = 0, f7 = 1, [-pi, +pi])`) | releve |
| 2 | `FUN_1431bc8a8` | `R(8)` (`FUN_1406d84b4(n = 8, [0, DAT_143cd87e4])`) ; `gate8` ; `R(16)` | releve |
| 3 | inline | `FUN_142af27f8` = `R(2)` ; `R(15)` | releve |
| 4 | `FUN_14319572c` | `R(32)` (`FUN_14080dec4`) ; `R(8)` ; `R(15)` (`[RSP+0x20] = 0xf`, `143195828..14319582d`) ; `FUN_14076d528(nmag = 0xa [RSP+0x28], ndir = 0x13 [RSP+0x30])` = `R(1) [0 -> R(19) + R(10)]` ; `gate8` ; `R(16)` | releve |
| 5 | `FUN_1431a2f10` | `R(32)` ; `gate8` ; `R(16)` ; `R(8)` | releve |
| 6 | `FUN_1431a4130` (via `FUN_142ef01c4`) | `gate8` ; `R(32)` (`FUN_14080dec4`) ; `R(16)` | releve |
| 7 | inline `FUN_142ef01c4` | `gate8` ; `FUN_1424d9a30` = `R(3)` ; `FUN_1431a3a50` = `R(15)` | releve |
| 8 | `FUN_143200ee8` | corps de `FUN_1432026f4` (ci-dessous) ; `R(4)` ; `R(1)` | releve (arguments de l'appel interne non relus ligne a ligne) |
| 9, 10 | `FUN_1432026f4` (avec vtable `PTR_FUN_143e0c970`) | `R(1)` -> `entree+0x19` ; si 1 -> `R(19)` (`FUN_14076dc04(entree+8, R9D = RDI + 0x13`, `XOR EDI,EDI` en `143202711`)) ; `FUN_141d0f344` = `R(32)` -> `entree+0x14` ; `R(4)` (borne a 11 : `CMP R9B, 0xb`) | releve |
| 11 | inline `FUN_142ef01c4` | `gate8` ; `R(19)` (`FUN_14076dc04(entree+4, R9D = 0x13)`, `142ef0249`) ; `FUN_142af27f8` = `R(2)` | releve |
| 12 | `FUN_14318d788` (via `FUN_142ef0538`) | `R(32)` ; `R(1)` | releve |
| 13 | inline `FUN_142ef0538` | `FUN_1406d676c(n = 0x60)` = `R(96)` (`142ef061b`) ; `R(19)` (`FUN_14076dc04(entree+0xc, 0x13)`, `142ef0643`) ; `gate8` (`142ef0658`, partage avec `t = 15`) | releve |
| 14 | inline | `FUN_1406d84b4(n = 8, max = DAT_143cd8374 = 1,0, f6 = f7 = AL)` = `R(8)` | releve (min non lu) |
| 15 | inline | `gate8` | releve |
| 16 | `FUN_142eefc54` | `FUN_14076d528(min = 0, max = DAT_143cd98f4, nmag = 10, ndir = 0x13)` = `R(1) [0 -> R(19) + R(10)]` | releve |
| 17 | `FUN_142eefd08` | `2 x FUN_1406d84b4` (largeurs non relevees) ; `R(1)` ; `gate8` | **partiel** |
| 18 | `FUN_14319c6f8` (via `FUN_142ef0388`) | non ouvert | non elucide |
| 19 | `FUN_142eefe1c` | non ouvert | non elucide |
| 20 | inline `FUN_142ef0388` | `FUN_1406d84b4` (largeur non relevee) | partiel |
| 21 | `FUN_142eeff18` | non ouvert | non elucide |
| 22 | inline `FUN_142ef0388` | `gate8` | releve |
| 23 | `FUN_142ef004c` | non ouvert | non elucide |
| >= 24 | `FUN_142ef0494` | non ouverte | non elucide |

Le depot (`consumeBipedActionTag`, `default`) affirme « tag >= 6 -> `FUN_142ef01c4` = 0 bit,
verite EXE 2026-06-13 » : le decompile de `FUN_142ef01c4` (`if (param_1 < 0xc) { 6..11 }
else FUN_142ef0538()`) et son disasm (`CMP RCX, 0xb ; JA 142ef0373`) montrent une
**continuation de dispatch**, pas une voie d'erreur ; chaque corps 6..11 lit des bits.

Statut d'`i63` : **partiel** — squelette et boucle 2 **releves et decidables** ; etiquettes
0..16 relevees (17 et 20 partielles) ; 18, 19, 21, 23 et `>= 24` nommees mais non ouvertes.
Le depot mesure `n1 = 0` dans le cas commun, ce qui rend la chaine rarement empruntee, mais
le port ne peut pas pretendre « 0 bit » pour ce qu'il n'a pas ouvert.

---

## 7. Ce que le port demandera

1. **`i60`** — rien de grammatical : la branche `porte == 1` vaut `3 x R(22)` constants
   (a comparer a `absAxisWFor`, aujourd'hui `AbsoluteAxisW = 14` uniforme, `profile.go:312`) ;
   la branche `porte == 0` doit tirer `IndexW = EffectiveRegionIndexBits()` et
   `W_axe = MapQuantEntry.AxisWidths` du match (loi identique a `FUN_140be9b88`, §2.1). C'est
   la condition ecrite du kill-switch `simStateComplete` ; une fois cablee, le drapeau se
   retire (regle 11 du `CLAUDE.md`). Le predicat peut etre porte a l'identique (cinq
   constantes) ou admis vrai par construction — les deux sont exacts.
2. **`i57`** — porter la branche `a == 1` de `FUN_142f262d4` : `R(6)`, `R(1)`, reference
   `FUN_1406d3140(dom 0)` (reutiliser `readVarWidthInt(br, 0)`), puis `FUN_142f04664`
   (position absolue = `consumeSimStateHandleTail` ; position RELATIVE = nouveau lecteur
   `R(2) + 3 x R(13) + R(1)[+R(16)]`, a partager avec `i59`). Retirer la mention « octet
   d'etat runtime » du commentaire et de la table.
3. **`i59`** — remplacer la grammaire mesuree par celle de l'ecrivain : `s = R(3) + 1`,
   `r = R(1)` (+ reference dom 1 avec sonde), `FUN_142f04664` (absolue ou relative), `R(6)`,
   les huit corps du §4. `Zero3` disparait (c'etait `r` + porte de region + index). Les
   vecteurs deviennent dequantifiables (min `0,1`, max `30 / 100 / 1`).
4. **`i63`** — (a) remplacer `bipedActionLoop2Count = 0` par
   `pop(m0) + pop(m1) + pop(m2 & 0x1ff)` calcule sur les trois mots du prologue ; (b) porter
   les etiquettes 6..16 relevees ; (c) supprimer le `default` « 0 bit » et rendre `ported =
   false` sur 17..31 tant qu'elles ne sont pas ouvertes ; (d) ouvrir `FUN_142eefd08`,
   `FUN_14319c6f8`, `FUN_142eefe1c`, `FUN_142eeff18`, `FUN_142ef004c`, `FUN_142ef0494` si
   un corpus les fait apparaitre (le depot mesure `n1 = 0` dans le cas commun).
5. **Entrees de profil / configuration** (jamais des constantes) : (i) bornes AABB par
   region de la carte et nombre de regions (`MapQuantEntry`, deja) — d'ou `IndexW` et
   `W_axe` ; (ii) cardinal des domaines d'entite (`FUN_140d10bb0`) et le bit de garde
   `DAT_144706104` du film ; (iii) `param_4` des composants (`i57`/`i59` : 2) ;
   (iv) la garde `FUN_14076f91c` (`fullPrecisionGate`).
6. **Constantes de valeur** (ne changent aucune largeur) : `+-100` (`R(16)`), `+-pi`
   (`R(8)`, `R(15)`), `0,5`, `1,0e-3`, `1,0e-4`, lignes `+-3 / +-0,7 / +-100` de
   `0x143b8c6f0`, `0,1 / 30 / 100 / 1` des vecteurs du grappin, `+-20000`, `1/120`,
   `(0,0,1)` / `(0,1,0)` / `(1,0,0)` / `(0,0,0)` de `0x14472a654 / 648 / 63c / 0x143b61708`.
7. **`ecs_table.tsv`** : les notes des quatre lignes sont a reecrire (« RUNTIME » x2,
   « masque RAM », « queue conditionnelle ») dans le commit qui porte chaque composant, jamais
   avant (`ecs_table_guard_test.go`).

## 8. Journal des appels Ghidra

Une cinquantaine d'appels HTTP (decompile, disassemble, read_memory, get_xrefs_to,
search_strings, lecture du catalogue). **8 appels en echec**, tous d'outillage, aucun sur une
fonction : 4 `search_strings` avec le nom de parametre `pattern` au lieu de `search_term`,
2 `disassemble_function` sur les thunks `0x142f02434` / `0x142f02444` (non definis comme
fonctions — decodes a la main depuis `read_memory`), 2 `disassemble_bytes` avec `address` au
lieu de `start_address`. Aucune ecriture dans Ghidra.
