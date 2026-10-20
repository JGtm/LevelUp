# NOTE 3.4 — LE REMPLISSEUR DES LARGEURS D'AXE, OUVERT (Q7), ET CE QU'IL DECIDE DE Q6

Date : 2026-09-16. Lot 3.4, volet PREPARATION (Ghidra en lecture seule, aucun decodage de film,
aucune ligne de production touchee). Branche `feat/decfilm-34g`, base `e7b9bd48e`.

Cible : `HaloInfinite.exe`, base d'image `0x140000000`, build `hi_1_13_0`, projet Ghidra du
2026-06-04, lu par HTTP direct `127.0.0.1:8089` (le pont MCP nomme est refuse sur ce projet).
Rien n'a ete renomme ni commente dans le projet Ghidra.

Reponse courte, dans l'ordre du brief :

1. **Les valeurs ne sont NI une constante de code NI une donnee de carte recopiee : elles sont
   CALCULEES, par une loi fermee, a partir des seules bornes.** La loi vit dans le build ; les
   bornes viennent de la carte pour la table « par index de plage » et du build (`+/-20000`,
   `.rdata`) pour la table « defaut ».
2. **Le niveau est un IMMEDIAT du site d'appel**, `0x10` = 16 pour le composant de position, sur
   les neuf sites de position releves. Ni champ de record, ni niveau lu au registre.
3. **Les `axisWidths` du catalogue `map_quant_bounds.json` sont EXACTEMENT ce que la table Ghidra
   donnerait : 79 cartes sur 79.** Et le releve memoire historique de la table par index est
   celui de Bazaar, a la ligne pres. 3.4.1 n'a donc AUCUNE valeur a saisir.
4. **Verdict Q6 : option (b) — `map_quant_bounds.json` tel quel, sans dimension `build`** ; les
   deux grandeurs (`axisW`, `indexW`) y sont deja, sous les noms `axisWidths` et
   `regionIndexBits`. Ce qui appartient au build, c'est la LOI (cinq constantes), et elle a sa
   place dans le code, pas dans un catalogue de valeurs.

---

## 1. Q7 — `FUN_140be9a14`, ouvert

### 1.1 Le remplisseur

Decompilation collee (extraits significatifs ; le corps entier tient en une page) :

```c
void FUN_140be9a14(void)
{
  lVar3  = DAT_144976b60;                          // le bloc de globales du moteur
  iVar11 = *(int *)(DAT_144976b60 + 0x7bc);        // NOMBRE de plages declarees
  if (0 < iVar11) {
    lVar6 = 0; puVar9 = &DAT_14462cbe0; puVar8 = &DAT_1445cc9b0;
    do {
      lVar10 = *(longlong *)(lVar3 + 0x7ac) + lVar6;   // tableau des plages, pas 0xdc
      cVar4  = FUN_140be9d1c(lVar10 + 0x44);           // AABB valide ?
      if (cVar4 != '\0') {
        *(uint *)(puVar8 + (ulonglong)(uVar7 >> 5) * 4 + 0x1b0) |= 1 << ((byte)uVar7 & 0x1f);
        uVar2 = *(undefined8 *)(lVar10 + 0x4c);
        uVar1 = *(undefined8 *)(lVar10 + 0x54);
        *puVar9 = *(undefined8 *)(lVar10 + 0x44);      // 24 octets de bornes recopies
        puVar9[1] = uVar2; puVar9[2] = uVar1;
        do { FUN_140be9b88(iVar11, lVar10 + 0x44); iVar11 = iVar11 + 1; } while (iVar11 < 0x20);
        FUN_140be9cc0(&DAT_1445cc9b0, &DAT_1445cc9b0 + ((longlong)(int)uVar7 * 3 + 0xc046) * 8);
      }
      iVar11 = *(int *)(lVar3 + 0x7bc);
      uVar7  = uVar7 + 1; lVar6 = lVar6 + 0xdc; puVar9 = puVar9 + 3;
    } while ((int)uVar7 < iVar11);
  }
  if (iVar11 == 1) { DAT_144632be0 = 1; }
  else            { DAT_144632be0 = FUN_1406d310c(); }   // ECX = nombre de plages
  DAT_1445cc9c8 = _DAT_143b8c6b8;  DAT_1445cc9cc = _UNK_143b8c6bc;   // +/-20000, 24 octets
  DAT_1445cc9d0 = _UNK_143b8c6c0;  DAT_1445cc9d4 = _UNK_143b8c6c4;
  _DAT_1445cc9d8 = DAT_143b8c6c8;
  do { FUN_140be9b88(iVar5, &DAT_1445cc9c8); iVar5 = iVar5 + 1; } while (iVar5 < 0x20);
}
```

Le decompilateur cache les arguments passes en `R8` / `R9` ; le desassemblage les donne, et c'est
la qu'on lit l'adressage EXACT des deux tables :

```
140be9a30: MOV RSI,qword ptr [0x144976b60]
140be9a3b: MOV ECX,dword ptr [RSI + 0x7bc]          ; nombre de plages
140be9a4b: LEA R12,[0x14462cbe0]                    ; bornes par index, pas 0x18
140be9a52: LEA R10,[0x1445cc9b0]
140be9a59: MOV R13,qword ptr [RSI + 0x7ac]          ; tableau des plages
140be9a63: LEA RCX,[R13 + 0x44]
140be9a67: CALL 0x140be9d1c                         ; AABB valide ?
140be9a8d: LEA R15,[RAX + RAX*0x2]                  ; index*3
140be9a91: SHL R15,0x7                              ; index*0x180 = index*32*12
140be9a95: LEA RAX,[0x1445ccbe0]
140be9a9c: OR   dword ptr [R10 + R8*0x4 + 0x1b0],EDX ; bitset des plages valides @1445ccb60
140be9aa4: MOVUPS XMM1,xmmword ptr [R13 + 0x44]     ; 24 octets de bornes ...
140be9ab2: MOVUPS xmmword ptr [R12],XMM1            ; ... vers DAT_14462cbe0 + index*0x18
140be9abe: MOV R9,R15                               ; SORTIE = 1445ccbe0 + (idx*0x20+L)*0xc
140be9ac1: LEA RDX,[R13 + 0x44]                     ; ENTREE = les bornes de la plage
140be9ac5: MOV ECX,R14D                             ; NIVEAU
140be9ac8: CALL 0x140be9b88
140be9ad0: ADD R15,0xc
140be9ad4: CMP R14D,0x20                            ; 32 niveaux
140be9b03: ADD RBP,0xdc                             ; pas d'une plage
140be9b16: CMP ECX,0x1
140be9b19: JZ  0x1423c6b40                          ; 1 plage -> DAT_144632be0 = 1
140be9b1f: CALL 0x1406d310c                         ; sinon ceilLog2(nombre de plages)
140be9b24: MOV dword ptr [0x144632be0],EAX
```

Ce qui se lit, sans interpretation :

| Sortie | Adresse | Pas | Remplie depuis |
|---|---|---|---|
| Bornes par index de plage | `DAT_14462cbe0` | `0x18` (24 octets) | `plage[i] + 0x44`, recopie brute |
| Largeurs par index de plage | `DAT_1445ccbe0` | `(index*0x20 + niveau)*0xc` | `FUN_140be9b88(niveau, plage[i]+0x44)` |
| Bitset des plages valides | `DAT_1445ccb60` | 1 bit par index | `FUN_140be9d1c` |
| Bornes par defaut | `DAT_1445cc9c8` | 24 octets | `DAT_143b8c6b8` en `.rdata` |
| Largeurs par defaut | `DAT_1445cc9e0` | `niveau*0xc` | `FUN_140be9b88(niveau, &DAT_1445cc9c8)` |
| Largeur d'index de plage | `DAT_144632be0` | u32 | `1` si une plage, sinon `ceilLog2(compte)` |

`FUN_140be9d1c` donne la FORME des bornes, six `float32` :

```c
undefined8 FUN_140be9d1c(float *p) {          // minX maxX minY maxY minZ maxZ
  return p != 0 && p[0] <= p[1] && p[1] != p[0]
                && p[2] <= p[3] && p[3] != p[2]
                && p[4] <= p[5] && p[5] != p[4];
}
```

### 1.2 La loi — `FUN_140be9b88` et `FUN_140be9c78`

```c
void FUN_140be9b88(int niveau, float *bornes, ..., int *sortie)
{
  etendue[0] = bornes[1] - bornes[0];
  etendue[1] = bornes[3] - bornes[2];
  etendue[2] = bornes[5] - bornes[4];
  pas = FUN_140be9c78(niveau);
  if (pas < DAT_143cd837c) { sortie[0] = sortie[1] = sortie[2] = 0x1a; return; }
  seuil = (pas + pas) * DAT_143cd975c;
  for (axe = 0; axe < 3; axe++) {
    casiers = 0x400000;
    if (etendue[axe] < seuil) casiers = (int)ceilf(etendue[axe] / (pas + pas));
    w = FUN_1406d310c(casiers);
    if (0x1a < w) w = 0x1a;
    sortie[axe] = w;
  }
}
```

```
140be9c78: MOV EAX,ECX ; MOV ECX,0x10 ; CMP EAX,ECX ; JG 0x140be9c9f
140be9c83: SUB ECX,EAX ; MOV EAX,0x1 ; SHL EAX,CL ; CVTDQ2PS ; MULSS XMM1,[0x143cd9758]
140be9c9f: MOVSS XMM1,[0x143cd9758] ; LEA ECX,[RAX + -0x10] ; SHL EAX,CL ; CVTDQ2PS ; DIVSS
```

Soit, en clair :

```
pas(L)  = 2^(16-L) * C          si L <= 16
pas(L)  = C / 2^(L-16)          si L >  16
W[axe]  = min(26, ceilLog2( min( ceil(etendue / (2*pas(L))), 2^22 ) ))
W[axe]  = 26 pour tout axe si pas(L) < 1e-4  (donc pour L >= 23)
```

Les cinq constantes, lues en `.rdata` (octets colles) :

| Symbole | Adresse | Octets | Valeur | Role |
|---|---|---|---|---|
| `C` | `143cd9758` | `89 88 08 3c` | `0x3c088889` = 1/120 | le pas au niveau 16 |
| plafond de casiers | `143cd975c` | `00 00 80 4a` | `0x4a800000` = 2^22 | garde de debordement |
| epsilon | `143cd837c` | `17 b7 d1 38` | `0x38d1b717` = 1e-4 | en dessous, 26/26/26 sans regarder les bornes |
| plafond de largeur | (immediat) | `LEA EBP,[RSI+0x17]`, RSI=3 | `0x1a` = 26 | `CMOVG EAX,EBP` en `140be9c34` |
| bornes par defaut | `143b8c6b8` | `00 40 9c c6` / `00 40 9c 46` x3 | `-20000` / `+20000` | l'AABB du chemin « defaut » |

`FUN_1406d310c` est bien `ceilLog2`, et il rend **0** pour 0 comme pour 1 :

```c
int FUN_1406d310c(uint v) {
  h = 0x1f; if (v != 0) { for (; v >> h == 0; h--) {} }
  if (v == 0) h = -1;
  return h == -1 ? 0 : ((v & ((1 << h) - 1)) != 0) + h;
}
```

### 1.3 Le lecteur — `FUN_14076e524`, qui referme la boucle

```c
longlong FUN_14076e524(longlong sortie, longlong lecteur, uint *indexLu, int NIVEAU)
{
  cVar6 = FUN_1406cf008(lecteur);          // R(1) : le bit de porte
  uVar7 = 0xffffffff;
  if (cVar6 == '\0') { ... uVar7 = R(DAT_144632be0) ... }   // l'index, sur indexW bits
  if (uVar7 != 0xffffffff) {
    pfVar13  = (float *)(&DAT_14462cbe0 + (int)uVar7 * 3);            // bornes de la plage
    lVar12   = (longlong)(int)uVar7 * 0x20 + NIVEAU;
    local_48 = *(undefined8 *)(&DAT_1445ccbe0 + lVar12 * 0xc);        // W[0], W[1]
    local_40 = *(undefined4 *)(&DAT_1445ccbe8 + lVar12 * 0xc);        // W[2]
  } else {
    pfVar13  = (float *)&DAT_1445cc9c8;                               // +/-20000
    local_48 = *(undefined8 *)(&DAT_1445cc9e0 + NIVEAU * 0xc);
    local_40 = *(undefined4 *)(&DAT_1445cc9e8 + NIVEAU * 0xc);
  }
  FUN_140cc5128(lecteur);                  // lit les 3 mots aux largeurs ci-dessus
  do {
    pas = (pfVar13[1] - pfVar13[0]) / (float)(1 << W[axe]);
    sortie[axe] = brut[axe] * pas + pfVar13[0] + pas * DAT_143cd84b0;  // 143cd84b0 = 0,5
  } while (...);
  *indexLu = uVar7;
}
```

Deux consequences qui comptent pour le lot :

- **Les bornes et les largeurs voyagent ENSEMBLE, et les largeurs sont derivees des bornes.** Il
  n'y a donc qu'UNE donnee : l'AABB de la plage. Le reste est de l'arithmetique.
- **`index == -1` (bit de porte pose) est le SEUL chemin `+/-20000`.** Un `index >= 0` designe
  toujours une plage reelle de la carte — y compris `index == 1` quand la carte en declare
  plusieurs (Live Fire : quatre plages, la jouee est la 1).

---

## 2. Reponse (1) — d'ou viennent les valeurs ecrites dans les deux tables

**Ni carte recopiee, ni constante de code : un CALCUL.** La reponse est la troisieme branche de
la question du brief — « un calcul a partir des bornes, alors la formule est la donnee, et les
bornes suffisent ».

| Table | Bornes d'entree | Nature de ces bornes |
|---|---|---|
| `DAT_1445cc9e0` (defaut) | `DAT_143b8c6b8` = `+/-20000` | **donnee de BUILD** : des octets de `.rdata` |
| `DAT_1445ccbe0` (par index) | `*(globales+0x7ac)[i] + 0x44` | **donnee de CARTE** : le bloc de plages, rempli au chargement |

Le tableau de plages vit a `*(DAT_144976b60 + 0x7ac)`, compte a `+0x7bc`, pas `0xdc`, AABB a
`+0x44`. `DAT_144976b60` est le bloc de globales du moteur (593 references) et aucune fonction du
binaire n'ecrit `+0x7ac` champ par champ — le comportement attendu d'un bloc de tag charge en
masse, pas d'une structure batie par du code.

**La preuve que ce bloc porte les AABB de la carte est NUMERIQUE, et elle est double** (§4) : les
79 entrees du catalogue de cartes reproduisent la loi au niveau 16, et le releve memoire
historique de `DAT_1445ccbe0` est, ligne a ligne, celui de Bazaar.

Chaine d'appel : `FUN_140be9a14` a un seul appelant, `FUN_140be9890` (fin de la sequence de
chargement : `FUN_140beae80`, `FUN_140bea190`, `FUN_140be9d48`, puis `FUN_140be9a14`), lui-meme
appele par `FUN_142e2f520` et `FUN_142e34d10`. C'est bien un remplissage « au chargement de la
carte », comme le supposait `killsource/calibrate.go`.

---

## 3. Reponse (2) — le NIVEAU : un immediat de site d'appel, pas une donnee du film

`FUN_14076e524` prend le niveau en quatrieme argument (`R9D`). Les 23 sites d'appel ont ete
releves ; voici les immediats, colles :

| Site | Fonction | Instruction | Niveau |
|---|---|---|---|
| `1406d009d` | `FUN_1406cfe44` (le chemin i0 de la position d'objet) | `MOV R9D,0x10` en `1406d008a` | **16** |
| `140f04de0` | `FUN_140f04d88` | `MOV R9D,0x10` en `140f04dd5` | **16** |
| `140f04f3d` | `FUN_140f04f18` | `MOV R9D,0x10` en `140f04f32` | **16** |
| `140f04f8b` | `FUN_140f04f68` | `MOV R9D,0x10` en `140f04f80` | **16** |
| `140f04ff0` | `FUN_140f04fb8` | `MOV R9D,0x10` en `140f04fe5` | **16** |
| `140f05023` | `FUN_140f04fb8` | `MOV R9D,0x10` en `140f05018` | **16** |
| `140fb8b3e` | `FUN_140fb8af0` | `MOV R9D,0x10` en `140fb8b33` | **16** |
| `140ee7293` | `FUN_140ee7270` | `MOV R9D,0x10` en `140ee7288` | **16** |
| `14226a6c7` | `FUN_14076f3ec` | `MOV R9D,0x10` en `14226a6b8` | **16** |
| `1410f045b` | `FUN_1410f03b4` | `MOV R9D,0xc` en `1410f044d` | 12 |
| `141121387` | `FUN_14112134c` | `LEA R9D,[RDI + 0xc]`, RDI=0, en `14112137e` | 12 |
| `140809783` | `FUN_1408096f8` | `MOV R9D,0xf` en `140809775` | 15 |
| `14076e4c0` / `14076e516` | `FUN_14076e494` / `FUN_14076e4ec` | `MOV R9D,R8D` — enveloppes generiques | herite |

**Conclusion.** Le niveau n'est jamais lu : ni dans le flux, ni dans le record, ni au registre du
build. C'est un immediat fige a la compilation, `0x10` partout ou il s'agit d'une position
d'objet. Les autres valeurs (12, 15) appartiennent a d'AUTRES composants, pas a un choix
dynamique.

**Ce que cela decide pour la cle de l'entree de profil** : `axisW` est **par CARTE** (a build
fige), et non « par carte x niveau ». La dimension « niveau » existe dans le moteur mais elle est
constante pour le chemin qui nous interesse ; elle n'a donc pas a exister dans le catalogue. Le
BUILD n'entre que par cinq constantes (le `0x10` du site d'appel, `C = 1/120`, le plafond 26, le
garde `2^22`, l'epsilon `1e-4`) — c'est-a-dire par la LOI, pas par une valeur.

---

## 4. Reponse (3) — la confrontation au catalogue commis : 79 sur 79

Instrument : `apps/go-api/tools/film_re/loi_largeurs_axe.go` (`//go:build research`), la loi
recopiee du desassemblage, et son test `loi_largeurs_axe_research_test.go`. Aucune ligne de
production, aucun film ouvert, aucune installation du jeu requise.

```
go test -tags=research ./tools/film_re/ -count=1 -v
  accord loi / catalogue : 79 cartes sur 79
  --- PASS: TestLoiDuRemplisseurEgaleLeCatalogue
  --- PASS: TestTableDefautEgaleLeReleveMemoire
  --- PASS: TestTableParIndexEgaleBazaar
  --- PASS: TestLargeurIndexDePlageSuitLeCatalogue
```

**Trois accords, trois sources independantes.**

1. **Le catalogue.** Les 79 entrees de `data/titles/halo_infinite/reference/map_quant_bounds.json`
   — bornes lues hors ligne dans les `.module` par `cmd/mapquant-build` — rendent, par la loi au
   niveau 16, exactement leur champ `axisWidths`. Aucun desaccord.
2. **La table DEFAUT.** Calculee depuis les seules bornes `+/-20000` de `.rdata`, elle rend
   `6/6/6`, `7/7/7`, `8/8/8` aux niveaux 0, 1, 2 — **mot pour mot le releve Cheat Engine de
   `DAT_1445cc9e0`** consigne au journal le 2026-06-11 (`ce_prec_widths_1445cc9e0.bin`).
3. **La table PAR INDEX.** Le releve historique de `DAT_1445ccbe0` (`1/1/0`, `2/2/1`, `3/3/2` aux
   trois premiers niveaux) est **celui de BAZAAR** : la loi appliquee aux bornes `bazaar` du
   catalogue rend ces trois lignes, puis `17/17/16` au niveau 16, ce qui est le champ
   `axisWidths` de la carte. Le releve n'avait jamais ete rattache a une carte ; il l'est.

Deux morsures jouees (fichiers restaures, arbre propre) : `C = 1/120 -> 1/121` fait rougir le
catalogue sur `streets` (`[12 12 12]` attendu, `[12 12 13]` rendu) ; `niveau 16 -> 15` fait rougir
les deux releves memoire sur quatre lignes.

### 4.1 Corollaire : 3.4.1 n'a AUCUNE valeur a saisir

Les deux grandeurs que l'item 3.4.1 veut « lire dans le profil » sont **deja au catalogue de
cartes**, et deja justes :

| Grandeur du decodeur | Champ du catalogue | Loi moteur correspondante |
|---|---|---|
| `axisW` (les 3 largeurs du chemin absolu i0) | `MapQuantEntry.AxisWidths` | `FUN_140be9b88` au niveau 16 |
| `indexW` (`DAT_144632be0`) | `MapQuantEntry.RegionIndexBits` (defaut 1) | `1` si une plage, sinon `ceilLog2(compte)` |
| les bornes de dequantification | `MapQuantEntry.Min` / `.Max` | `DAT_14462cbe0 + index*0x18` |

Il ne manque donc que le **branchement** : `MovementProfile.AbsoluteAxisW` (uniforme 14) et
`MovementProfile.Traversal` doivent cesser d'etre des constantes et devenir la projection de
`Profile.Map`.

### 4.2 Le lien carte -> film : une cle EXTERNE, et elle existe deja

Le film **ne nomme pas sa carte** (lot H, mesure a zero occurrence). La cuisson la connait par la
BASE, et il n'y a qu'un site pour le dire : `internal/sync/killcollector/map_identity.go`.
`nomsDeCarteDuMatch` interroge `port.ReplayMapNameRepo.MapKeysForMatch`, puis
`entreeDeCatalogueParNom` resout au catalogue ; deux echecs distincts et COMPTES
(`ErrSansNomDeCarte`, `profile.ErrUnknownMapBounds`), jamais un « au plus proche ».

C'est licite au regard de la decision « le film est autoportant » : la carte n'est pas une cle du
FILM, c'est une donnee du MATCH — exactement ce que l'en-tete de `profile_table.go` a deja
tranche le 2026-09-16 (« une quatrieme entree n'est pas une cle du film mais une donnee du MATCH
— l'entree de catalogue de la CARTE (`MapQuantEntry`) — et elle entre au profil par le
constructeur »). Le lot 3.4 n'a donc rien a rouvrir de cette decision.

Et la voie d'a cote est fermee par la mesure, pas par un gout : la signature de largeurs a ete
SUPPRIMEE (lot 1.9.4, D13) parce qu'elle n'identifie qu'11 cartes sur 79, et qu'elle designait
`aquarius` sur les deux films Live Fire. La lecture ci-dessus explique POURQUOI elle ne peut pas
marcher : `W = min(26, ceilLog2(ceil(60*etendue)))` ecrase toute etendue d'un facteur 2 dans la
meme classe.

---

## 5. Reponse (4) — VERDICT pour le pilote

### 5.1 Q6 : option (b), `map_quant_bounds.json` tel quel — sans dimension `build`

- **(a) etendre la grammaire des cles du catalogue de profils avec une cle composite : NON.**
  Rien ne le justifie. La carte entre deja au profil par le constructeur, comme une donnee de
  match ; la grammaire fermee `format` / `build` / `majeure` / `toutes` reste intacte.
- **(c) une entree `film_profiles.json` par build : NON.** Les largeurs ne dependent pas du build
  *en valeur*. Ce qui depend du build, ce sont les cinq constantes de la loi — et une loi n'est
  pas une valeur de catalogue : elle est du code, aujourd'hui `internal/himap/sbsp.go`
  (`Bounds.AxisWidths`, `PositionLevel`, `quantDivisor`, `maxAxisWidth`).
- **(b) `map_quant_bounds.json` tel quel : OUI**, et sans y ajouter une colonne. Les deux champs
  necessaires existent (`axisWidths`, `regionIndexBits`), ils sont prouves justes 79 fois sur 79,
  et ils sont deja produits par un outil hors ligne dont le gate `gamefiles` verifie qu'il rend le
  fichier commis a l'octet.

**Reserve honnete, a ecrire dans le lot** : la mesure porte sur UN build, `hi_1_13_0` — c'est le
seul ouvert dans le projet Ghidra. Le build n'entre dans la valeur que par les cinq constantes
ci-dessus. La bonne facon de couvrir ce risque n'est pas une dimension `build` dans un catalogue
de cartes (il faudrait la remplir pour 79 x N builds sans avoir les N executables), c'est une
**entree de profil sur la LOI** : une ligne de `film_profiles.json` de sorte `toutes`, champ
`Movement.LoiLargeursAxe`, valeur `"L=16 C=1/120 cap=26 garde=2^22 eps=1e-4"`, provenance
`relue`, preuve = les adresses de §1.2. Le jour ou un build change une de ces cinq constantes, la
ligne se dedouble par `build` et rien d'autre ne bouge.

### 5.2 Ce que 3.4.1 devra coder, une fois M2 clos

Perimetre, dans l'ordre. Chemins relatifs a `apps/go-api/`.

1. **`internal/games/halo_infinite/film/profile/`** — la derivation, cote donnees :
   - `map_bounds.go` : `MapQuantEntry.Layout()` rend deja `I0Layout{GateBits, AxisW, Region}`.
     Ajouter la projection vers le descripteur du chemin absolu : un
     `MapQuantEntry.PrecisionAbsolue() PrecisionDescriptor` = `{IndexW: EffectiveRegionIndexBits(),
     AxisW: AxisWidths, Region: Region}`. Zero valeur nouvelle, une methode.
   - `profil.go` : `MovementProfile.AbsoluteAxisW` (uniforme `14`, pose en dur ligne 337)
     **disparait** au profit du descripteur par carte. `Traversal` idem.
   - `profile_table.go` : les deux lignes `Movement.Traversal` et `Movement.AbsoluteAxisW`
     passent de `cleToutes` / `ProvenancePresumee` a une ligne `relue` dont la preuve est
     « derivee de `MapQuantEntry` par la loi de `FUN_140be9b88` — cette note ».
2. **`internal/games/halo_infinite/film/grammar/position_capture.go`** — la lecture :
   - `absAxisWFor(br, idx, i)` cesse de jeter `idx`. Il lit la largeur de la plage `idx` ;
     `idx == -1` (porte posee) prend la table DEFAUT, c'est-a-dire les bornes `+/-20000` et les
     largeurs `22/22/22` sur ce build, et surtout PAS les largeurs de la carte.
   - `absAxisW` / `absoluteAxisW` : le repli uniforme part avec `AbsoluteAxisW`.
   - `dequantWorldAxis` : la range doit suivre l'index, comme chez `FUN_14076e524` — bornes de la
     plage pour `idx >= 0`, `+/-20000` pour `idx == -1`.
3. **`internal/games/halo_infinite/film/facts/killsource/calibrate.go`** — l'inference devient
   ORACLE (D2 de l'item) : le balayage reste, il ne DECIDE plus. Le test s'ecrit sur le modele
   deja en place pour `Movement.WorldObject` : « accord catalogue / film, N temoins sur N ».
   Les deux replis `repli_calibration_paquet_exclu` et `repli_calibration_paquet_non_localise`
   sortent du registre.
4. **`internal/himap/sbsp.go`** — deux completions de la loi, sans effet mesurable aujourd'hui
   mais qui la rendent exacte (§6, C2 et C3).
5. **Gate** : le test de recherche de ce lot (`tools/film_re/`) est deja la preuve « catalogue =
   moteur ». Ce qui reste a mesurer avec decodage : `TestBTB2025Abstention` sur les temoins,
   AVANT et APRES, consigne dans le depot (Q9 de la note M3).

**Dependances inchangees** : 3.4.1 reste apres 2.3 (fusionne `88f1a1115`) et apres la fusion de
M2 ; rien dans cette note ne la leve.

---

## 6. Ce que la lecture CONTREDIT dans le depot (corrections a porter par 3.4.1)

Aucune n'a ete corrigee ici (regle 7 : hors perimetre d'un lot de preparation).

- **C1 — `grammar/components_position_i0.go`, `consumeAbsolutePayload` : « index 1 / no-index =
  +/-20000 » est FAUX, et le filtre `if idx != 0 { return }` avec lui.** Le desassemblage de
  `FUN_14076e524` ne donne `+/-20000` que pour `index == -1`. Un `index >= 0` designe toujours une
  plage reelle. Le catalogue le dit deja par ailleurs : `live fire` porte `region = 1`,
  `regionIndexBits = 2` — sa plage JOUEE est la 1. Sur cette carte, le code jette donc les
  positions valides et garde les autres. Portee : les deux films Live Fire du corpus.
- **C2 — le garde de debordement n'est pas modelise.** `himap.Bounds.AxisWidths` ne borne pas le
  compte de casiers a `2^22`. Au niveau 16 le seuil est une etendue de **69 905,1** unites monde ;
  la plus grande etendue du catalogue est 2 707,4 (`recharge`), donc **aucune carte actuelle n'est
  concernee** — c'est pourquoi l'accord est de 79 sur 79. Un canevas Forge plus grand divergerait.
- **C3 — le garde d'epsilon n'est pas modelise** (`pas < 1e-4` -> `26/26/26`, soit niveau >= 23).
  Sans effet au niveau 16 ; a ecrire si la loi devient generale sur les 32 niveaux.
- **C4 — `profile/i0_layout.go` dit « `DAT_144632be0` = ceilLog2(nb de BSP VALIDES) ».** Le
  desassemblage relit le compte BRUT (`MOV ECX,dword ptr [RSI + 0x7bc]` en `140be9afb`) ; le
  bitset des plages valides (`DAT_1445ccb60`) n'alimente pas cette largeur. Sans consequence tant
  que toutes les plages declarees ont une AABB valide, mais l'enonce doit etre exact.
- **C5 — `grammar/position_capture.go` : « 14 est une entree REELLE de la table du .exe ».**
  Exact, mais au NIVEAU 8 de la table DEFAUT — or le composant de position lit au niveau 16, ou
  la table defaut vaut `22/22/22` et la table par index vaut les largeurs de la carte. L'uniforme
  14 n'est donc l'entree d'aucune des deux cases reellement lues. La phrase reste vraie et
  trompeuse : a reecrire au lot.
- **C6 — la table `DAT_143b8c6f0` a TROIS entrees, pas cinq.** Octets relus, stride `0x18` :
  `143b8c6f0` = `+/-3`, `143b8c708` = `+/-0,7`, `143b8c720` = `+/-100`, puis `143b8c738` n'est
  plus une plage. L'ordre de `profile/plages_quant.go` (`QuantRangeUnit3`, `QuantRangeNorm`,
  `QuantRangeWorld100`) est donc JUSTE ; les deux autres variables du fichier sont des captures
  de CARTE. C'est le plan (item 2.5.b, « les cinq plages de `DAT_143b8c6f0` ») qui parle mal —
  la correction est une ligne de prose, pas une ligne de code.

---

## 7. Ce qui reste non elucide

- **Le remplissage du bloc de plages lui-meme.** On sait d'ou `FUN_140be9a14` le LIT
  (`*(DAT_144976b60 + 0x7ac)`, compte `+0x7bc`, pas `0xdc`, AABB `+0x44`) et que son contenu EST
  celui des `sbsp` de la carte (preuve numerique, §4). On n'a pas trouve le site qui l'ECRIT :
  aucune des 170 instructions qui adressent `+0x7ac` n'est un stockage de pointeur sur ce bloc
  (la seule candidate, `FUN_141582a60` en `141582b8a`, appartient a une autre structure, offsets
  non alignes). Hypothese non verifiee : le bloc arrive par chargement de tag en masse. **Sans
  consequence pour 3.4** — la preuve numerique suffit — mais la ligne reste ouverte.
- **`FUN_140be9cc0(&DAT_1445cc9b0, bornesDeLaPlage)`**, appele une fois par plage valide apres le
  remplissage des 32 niveaux : non ouvert. Probablement une notification ou un enregistrement ;
  il n'ecrit aucune des six sorties du tableau de §1.1.
- **Les autres builds.** Un seul executable est disponible (`hi_1_13_0`). Les cinq constantes de
  la loi n'ont donc pas ete comparees d'un build a l'autre (§5.1, reserve).
- **Le releve Cheat Engine `QuantRangeCEBiped`** (`profile/plages_quant.go`) et les bornes
  `cliffhanger` du catalogue designent la meme boite a ~3e-4 unite pres sur des etendues de 113 a
  137 — soit un ecart relatif de ~2e-6, au-dela de l'epsilon de `float32`. Les largeurs sont
  identiques (13/13/14) dans les deux cas, donc sans consequence ; l'ecart lui-meme n'est pas
  explique (precision de la capture, ou une AABB legerement differente).

---

## 8. Provenance — toutes les adresses citees

`HaloInfinite.exe`, base `0x140000000`, build `hi_1_13_0`,
`D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe`, projet Ghidra du 2026-06-04,
311 103 fonctions. Lecture seule par `curl` sur `http://127.0.0.1:8089` (catalogue
`/mcp/schema`), le 2026-09-16.

Fonctions : `FUN_140be9a14` (remplisseur) · `FUN_140be9890` (son seul appelant) · `FUN_142e2f520`,
`FUN_142e34d10` (appelants de celui-ci) · `FUN_140be9b88` (la loi) · `FUN_140be9c78` (le pas) ·
`FUN_140be9d1c` (validite d'une AABB) · `FUN_140be9cc0` (non ouvert) · `FUN_1406d310c` (ceilLog2)
· `FUN_14076e524` (le lecteur) · `FUN_1406cf008` (R(1)) · `FUN_140cc5128` (les 3 mots d'axe) ·
`FUN_1406cfe44` (le chemin i0 de la position d'objet).

Donnees : `DAT_144976b60` (globales, `+0x7ac` tableau des plages, `+0x7bc` compte, pas `0xdc`,
AABB `+0x44`) · `DAT_14462cbe0` (bornes par index, pas `0x18`) · `DAT_1445ccbe0` (largeurs par
index, `(idx*0x20 + L)*0xc`) · `DAT_1445ccb60` (bitset des plages valides) · `DAT_1445cc9c8`
(bornes par defaut) · `DAT_1445cc9e0` (largeurs par defaut, `L*0xc`) · `DAT_144632be0` (largeur
d'index de plage) · `DAT_143b8c6b8` (`+/-20000`) · `DAT_143cd9758` (1/120) · `DAT_143cd975c`
(2^22) · `DAT_143cd837c` (1e-4) · `DAT_143cd84b0` (0,5, le centre de casier).

Instrument : `apps/go-api/tools/film_re/loi_largeurs_axe.go` et son test, tag `research`.
