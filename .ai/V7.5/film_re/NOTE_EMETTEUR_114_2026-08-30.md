# L'emetteur des evenements de film : enveloppe cote ecriture, sieges, sortie de vehicule

Date : 2026-08-30. Lot B2 du chantier « visee lunette ». Retro-ingenierie **pure, lecture seule**
sur `HaloInfinite.exe` (greffon Ghidra HTTP 127.0.0.1:8089). Aucun code du depot n'a ete modifie.

Note precedente : `NOTE_ENVELOPPE_EVENTS_2026-08-30.md` (lot B1). Cette note **corrige un point
de la grammaire etablie en B1** (voir B2b) et **confirme la refutation** de l'hypothese
« lunette = siege de l'arme » par une preuve nouvelle et independante (voir B2c).

Toutes les adresses sont des adresses virtuelles du binaire retail analyse.

---

## Resume executif

1. **L'emetteur generique est confirme.** `FUN_1424d80bc` appelle bien l'ecrivain `+0x60` du
   descripteur, resolu par la meme table runtime indexee par type que le recepteur.
2. **Il n'existe aucun site qui emet le type 114 avec un immediat.** Ce n'est pas un echec de
   recherche : la chaine d'emission a ete cartographiee de bout en bout et **le numero de type
   n'y apparait jamais comme constante** — il est lu dans un champ (`slot+0x04`) de la structure
   d'evenement. La question « quel code de gameplay ecrit 0x72 » n'a donc pas de reponse : la
   formulation suppose un mecanisme qui n'existe pas dans ce moteur.
3. **B2b est resolu, et l'hypothese du sous-en-tete de 16 bits est REFUTEE.** Entre le `R(7)` du
   type et la premiere reference il n'y a **rien**. En revanche il y a **un bit AVANT le type** :
   un drapeau de continuation (1 = un evenement suit, 0 = fin de la liste). Prouve des deux cotes.
   Aucun compteur de sequence, aucun drapeau de priorite dans le paquet.
4. **Consequence directe pour le lot de mesure** : un decodeur qui ignore ce bit de continuation
   decale tout d'un bit et lit `0x40 | (T >> 1)` au lieu de `T`. Les « 125 evenements 114 » du
   film s'expliquent alors comme des **type 100 `weapon_effect`** (voir la prediction testable).
5. **La table des sieges est entierement decodee** (pointeur `+0xab8`, cardinal `+0xac8`, entree
   de `0x278` octets, drapeau `0x400`, marqueur `+0x96`), et l'applicateur du 114 **refuse toute
   reference qui n'est pas du type d'objet 1** par une garde de masque a l'execution. Une arme ne
   peut structurellement pas etre la cible d'un `biped_board_vehicle`.
6. **L'applicateur de `unit_zoom` (126) est identifie** : `FUN_14110ec20`, il pose un octet
   d'etat de lunette a `unit+0x462`. C'est l'evenement de visee, sans ambiguite.

---

## B2a. L'emetteur : confirme, mais sans immediat de type

### B2a.1 `FUN_1424d80bc` appelle bien l'ecrivain (PROUVE)

```c
void FUN_1424d80bc(longlong ctx, int type, undefined4 payload_size, undefined8 payload, longlong writer)
{
  plVar1 = *(longlong **)(*(longlong *)(*(longlong *)(ctx + 8) + 0x18) + 0x210 + (longlong)type * 8);
  (**(code **)(*plVar1 + 0x60))(plVar1, payload_size, payload, writer, 1);
  cVar3 = FUN_14076cea8();
  if (cVar3 != '\0') {
    if (DAT_1450e2520 == '\0') {
      FUN_1406d49c4(writer);
      ...
      *(int *)(writer + 0x2c) = *(int *)(writer + 0x2c) + 0x20;   // 32 bits
      *(undefined8 *)(writer + 0x30) = 0xbcddcba;                 // sentinelle
      ...
```

- L'indexation `[[ctx+8]+0x18] + 0x210 + type*8` est **la meme table runtime** que celle du
  recepteur `FUN_14080a9d4` (`[[param_1+0x18]] + 0x210 + type*8`). Meme mecanisme, meme table.
- **Le trailer est identifie** : le « `R(1)` queue ; si 1 : `R(32)` » du lot B1 est une
  **sentinelle de debogage de valeur constante `0x0BCDDCBA`**, ecrite seulement si
  `FUN_14076cea8()` est vrai et `DAT_1450e2520 == 0`. Ce n'est pas une donnee de jeu.

### B2a.2 La chaine d'emission complete (cartographiee)

| Etage | Fonction | Role |
|---|---|---|
| 1. file | liste chainee `net+0x128` | les evenements en attente, chainee par `slot+0x40` |
| 2. selection | `FUN_140bbd2b4` | filtre par pertinence (`FUN_1408f0074`), produit des descripteurs 32 bits `{canal:2, priorite, index:13}` |
| 3. enveloppe | `FUN_140bbd474` | ecrit en-tete + les 3 references dans un buffer de cache |
| 4. charge utile | `FUN_1424d80bc` | appelle l'ecrivain `+0x60` puis la sentinelle |
| 5. cache | `DAT_14521d920` (2048 entrees de 16 o) | `{type, offset octet, longueur en bits, ...}` par index d'evenement |
| 6. concatenation | `FUN_14076ba54` / `FUN_142f2c050` | recopie les paquets caches dans le flux du client (`FUN_1406d60f4`) |
| 7. purge | `FUN_14076be40` | remet le cache et le curseur a zero a chaque tick |

`FUN_1406d60f4` est une **copie de bits pure** : elle n'ajoute aucun prefixe. Le paquet insere
dans le flux commence donc exactement par les bits ecrits a l'etage 3.

### B2a.3 Structure du slot d'evenement sortant (relevee sur pieces)

Lue dans `FUN_140bbd474`, `FUN_140bbd1a4` et `FUN_140bbd2b4` :

| Offset | Contenu |
|---|---|
| `+0x04` | **numero de type** (indexe la table des descripteurs) |
| `+0x08` | identifiant de groupe / chaine d'evenements lies (`-1` = isole) |
| `+0x0c` | estampille consommee par `FUN_1405f5008` (age) |
| `+0x10` | seuil / duree de vie (`-1` = pas de limite) |
| `+0x14`, `+0x18`, `+0x1c` | **les 3 references** (handles 32 bits, `-1` = nulle) |
| `+0x20` | pointeur vers la charge utile |
| `+0x28` | taille de la charge utile (0 = pas de charge utile) |
| `+0x30` | masque 64 bits des clients **cibles** |
| `+0x38` | masque 64 bits des clients **deja servis** |
| `+0x40` | pointeur vers l'evenement suivant |
| `+0x48` | index de l'evenement (13 bits utiles) |
| `+0x50` | horodatage |

La porte d'1 bit de B1 est confirmee cote ecriture par la source meme du bit :
`local_f8 = local_f8 * 2 | (ulonglong)(iVar6 != -1)` ou `iVar6` est la reference lue a
`slot+0x14+4*i`. **« Reference non nulle » = handle different de `-1`.** Conforme a B1.

### B2a.4 Pourquoi il n'y a pas d'immediat 0x72 (resultat NEGATIF, argumente)

- `FUN_140bbd474` lit le type par `*(int *)(slot + 4)` — jamais une constante.
- Recherche d'octets `C7 4? 04 72 00 00 00` (`MOV dword ptr [reg+0x4], 0x72`) sur tout le
  binaire : **0 resultat**.
- Recherche d'octets `BA 72 00 00 00 E8` (`MOV EDX,0x72 ; CALL`) : 2 resultats, tous deux dans du
  code sans rapport (bloc froid de `FUN_140795d7c`).
- `get_xrefs_to` sur l'ecrivain `0x142f19afc`, sur l'applicateur `0x142f10aac` et sur la vtable
  `0x143d0d330` : **aucun appelant nomme**, uniquement les entrees de table. Idem pour la table
  statique `0x144724A90`, referencee par la seule fonction d'initialisation `FUN_14025fda0`.

**Conclusion (PROUVE pour la partie negative, SUPPOSE pour la cause)** : l'allocation d'un slot
sortant est generique et parametree par le type ; le code de gameplay ne manipule pas de numero
de type litteral au site d'emission. **La fonction d'allocation elle-meme n'a pas ete
identifiee** — c'est la seule question de B2a qui reste ouverte, et je la marque comme telle.
Trois pistes ont ete epuisees sans resultat : les xrefs du contexte reseau `DAT_144e61d78`
(une centaine de lecteurs, piste noyee), les xrefs du cache `DAT_14521d920` (uniquement les
etages 3, 6 et 7 ci-dessus), une recherche d'instructions sur l'offset `0x130` (le serveur a
expire ; non rejouee, conformement a la consigne de prudence).

### B2a.5 Ce qui part dans le `R(6)` et dans les references (etabli par les applicateurs)

Faute de site d'emission, la semantique est etablie par le cote reception, ou elle est explicite.
`FUN_140a10204(handle, masque)` **prouve** la nature des references :

```c
int FUN_140a10204(undefined8 handle, uint masque) {
  if ((int)handle != -1) {
    iVar2 = FUN_140809d20(handle, 0);                 // handle -> index d'objet
    if (iVar2 != -1) {
      bVar1 = FUN_1406b68a8(iVar2);                   // type d'objet (enum)
      if ((masque & 1 << (bVar1 & 0x1f)) != 0) return iVar2;   // garde de type
    }
  }
  return -1;
}
```

Le second argument est donc un **masque de bits sur l'enumeration des types d'objets**.

| Type | ref0 (masque) | ref1 (masque) | charge utile |
|---|---|---|---|
| 114 `biped_board_vehicle` | `1` = type 0 seul | `2` = **type 1 seul** | `R(6)` = index de siege |
| 103 `unit_exit_vehicle` | `0xFFFFFFFF` = tout | `0x1003` = types 0, 1, 12 | siege + 2 octets (voir B2d) |
| 126 `unit_zoom` | `0x1003` = types 0, 1, 12 | non lue par l'applicateur | 1 octet = niveau |

Preuve annexe et independante du nom du type 114 : le slot `+0x48` de sa vtable est
`FUN_142f1e2cc`, qui formate `"unit board vehicle: relevance=%5.3f"` (chaine `0x143e0d1e0`).

---

## B2b. Le sous-en-tete : il n'existe pas, mais il y a un bit avant le type

### B2b.1 Cote ecriture (PROUVE)

Initialisation du bit-writer dans `FUN_140bbd474`, avant toute autre ecriture :

```c
local_f0 = 8;                                              // writer+0x38 : bits utiles
local_fc = 8;                                              // writer+0x2c : compteur total
local_f8 = (ulonglong)*(uint *)(lVar3 + 4) | 0x80;         // writer+0x30 : accumulateur
...
do {                                                        // immediatement : les 3 references
    iVar6 = *piVar16;
    plVar8 = <descripteur du type>;
    (**(code **)(*plVar8 + 0x58))(plVar8, piVar15);          // domaine de la reference i
    local_f8 = local_f8 * 2 | (ulonglong)(iVar6 != -1);      // la porte d'1 bit
    if (iVar6 != -1) { FUN_14080ada0(...); FUN_1406d5110(); }
} while (...)
```

Les champs `+0x30` / `+0x38` / `+0x2c` sont ceux de la structure de bit-writer deja connue
(`FUN_1424d80bc` les manipule a l'identique). L'accumulateur est **cale a droite** et emis
**MSB en premier** (`uVar6 << (0x40 - nbits)`), ce que confirme l'ecrivain du 114 lu octet a
octet a `0x142f19afc` (Ghidra n'y declare pas de fonction) :

```
45 8B 59 38     MOV  R11D, dword ptr [R9+0x38]   ; nbits
B9 06 00 00 00  MOV  ECX, 6                      ; largeur = 6 bits
45 8B 10        MOV  R10D, dword ptr [R8]        ; la valeur, depuis la charge utile
41 01 49 2C     ADD  dword ptr [R9+0x2C], ECX    ; compteur += 6
48 C1 E2 06     SHL  RDX, 6                      ; acc <<= 6
41 8D 43 06     LEA  EAX, [R11+6]
49 0B D2        OR   RDX, R10                    ; acc |= valeur
41 89 41 38     MOV  dword ptr [R9+0x38], EAX
49 89 51 30     MOV  qword ptr [R9+0x30], RDX
C3              RET
```

**Le `R(6)` du type 114 est donc confirme au bit pres, cote ecriture.**

Avec `nbits = 8` et un accumulateur valant `0x80 | type`, les huit premiers bits emis sont :
**`1`, puis les 7 bits du type, MSB en premier.**

### B2b.2 Cote lecture : le bit appartient a la boucle, pas au paquet (PROUVE)

Le dispatcher `FUN_14080a9d4` lit **7 bits exactement** pour le type (`+= 7`, puis `>> 0x19` sur
les bits de poids fort) et **rien avant**. Le huitieme bit est lu par son appelant,
`FUN_14076a1c4` :

```c
do {
    lVar3 = *(longlong *)(param_3 + 0x30);
    *(int *)(param_3 + 0x2c) += 1;
    *(longlong *)(param_3 + 0x30) = lVar3 * 2;
    bVar4 = -1 < lVar3;                       // bit de tete == 0 ?
    *(uint *)(param_3 + 0x38) += 1;
    ...
    if (bVar4) break;                          // 0 -> fin de la liste des evenements
    iVar2 = FUN_14080a9d4(...);                // 1 -> un evenement suit
} while (iVar2 == 0);
```

**Reponse a B2b : il n'y a AUCUN champ entre le type et la premiere reference.** Le seul champ
supplementaire est **un bit de continuation place AVANT le type**, valant 1 quand un evenement
suit et 0 pour clore la liste. Cote ecriture il est integre au paquet mis en cache (d'ou le
`| 0x80`), ce qui permet de concatener les paquets sans retouche — le `0` final etant ecrit par
l'etage de concatenation.

**Ni compteur de sequence, ni drapeau de priorite, ni champ de longueur** n'existent dans le
paquet. Le triplet `{canal, priorite, index}` que produit `FUN_140bbd2b4` sert au choix des
evenements a envoyer, pas a leur encodage.

### B2b.3 Grammaire corrigee

```
[1]  bit de continuation : 1 = un evenement suit, 0 = fin de la liste
R(7) type
pour i dans 0,1,2 :
    R(1) porte                                     (0 = reference nulle)
    si porte == 1 :
        si domaine(i) == 1 : R(1) sonde
        R(ceil(log2(cardinal)))                    -> index relatif
        R(2)                                       -> generation
charge utile (ecrivain +0x60 / lecteur +0x68)
si mode debogage : R(1) ; si 1 : R(32) == 0x0BCDDCBA
```

Taille corrigee : **114 = 1 + 51 = 52 bits** hors sentinelle (B1 comptait 52 en incluant a tort
le bit de queue) ; **126 = 1 + 41 = 42 bits** hors sentinelle.

### B2b.4 Prediction testable pour le lot de mesure (SUPPOSE, mais quantifie)

Si le decodeur du lot a aligne son `R(7)` sur le bit de continuation, il lit
`0x40 | (T >> 1)` au lieu de `T`. Consequences arithmetiques :

- un vrai **114** est lu **121** ;
- un vrai **126 `unit_zoom`** est lu **127** ;
- ce qui est lu **114** est en realite un **type 100 `weapon_effect`** (ou 101) ;
- ce qui est lu **126** est en realite un type **124** ou **125**.

**Sur une carte sans vehicule, 125 paquets lus « 114 » qui coincident avec les mises et sorties
de lunette s'expliquent alors parfaitement : ce sont des `weapon_effect`.** Le test discriminant
tient en une ligne : **rejouer le decodage en consommant 1 bit de continuation avant chaque
`R(7)`, et compter les paquets de type 126.** Si le compte de 126 s'aligne sur les transitions
relevees a la main, la question de la lunette est close.

---

## B2c. Les sieges : structure decodee, et l'arme definitivement ecartee

### B2c.1 Indexation de la table des sieges (PROUVE)

Extrait de l'applicateur du 114, `FUN_142f10aac` :

```c
iVar3 = FUN_140a10204(*param_3, 1);            // ref0 : type d'objet 0 exige
lVar6 = FUN_140498800(iVar3, 0x1003);          // composant unite du bipede
iVar4 = FUN_140a10204(param_3[1], 2);          // ref1 : type d'objet 1 exige
lVar7 = FUN_1404777f0(iVar4);                  // l'objet porteur
if (*param_5 == -1) return;                    // le R(6) : -1 = pas de siege
lVar8 = FUN_1405839d0(lVar7 + 0x2c, 0x6f626a65);           // bloc de tag 'obje'
iVar5 = *param_5;
if (*(int *)(lVar8 + 0xac8) <= iVar5) return;              // cardinal de la table
puVar9 = (uint *)((longlong)iVar5 * 0x278 + *(longlong *)(lVar8 + 0xab8));  // entree
if ((*puVar9 & 0x400) == 0) return;                        // drapeau du siege
if (*(short *)((longlong)puVar9 + 0x96) == -1) return;     // marqueur du siege
```

| Element | Valeur |
|---|---|
| Groupe de tag interroge | `'obje'` = `0x6f626a65` (definition d'objet) |
| Pointeur du tableau des sieges | `bloc + 0xab8` |
| Nombre de sieges | `bloc + 0xac8` (int) |
| Taille d'une entree de siege | **`0x278` octets** |
| Drapeaux du siege | offset `+0x00` de l'entree, **bit `0x400`** requis pour embarquer |
| Marqueur / point d'attache | `short` a l'offset **`+0x96`**, doit etre different de `-1` |

Precision par rapport a l'acquis du lot B1 : le champ `+0x96` n'est pas un indicateur de
« disponibilite » mais un **identifiant de marqueur** (un siege sans marqueur valide est
ignore). L'occupation courante, elle, vit sur le composant unite : `unite+0x18` = index du
porteur, `unite+0x146` (short) = index du siege occupe.

Le reste de l'applicateur : eviction de l'occupant precedent (`FUN_1408de1dc` puis
`FUN_14080b824`), puis emission d'un evenement **de gameplay local** (et non reseau) de type
`0x13` via `FUN_14080b870`, avec une structure de `0x4c` octets portant `{0x13, 0xffff, porteur,
(short)siege, 0}`. `FUN_14080b870` est le bus d'evenements local (`PTR_PTR_143cf00a0` indexe par
type) : il ne serialise rien.

### B2c.2 Les armes portent-elles des sieges ? Reponse : NON pour cet evenement (PROUVE)

Deux niveaux de reponse, a ne pas confondre :

- **Stockage** : la table des sieges est lue depuis le bloc `'obje'`, c'est-a-dire la definition
  d'objet, socle commun a toutes les classes d'objets. **Un stockage partage ne prouve pas qu'une
  arme porte des sieges** : le champ peut simplement etre declare a la racine et rester vide pour
  les classes qui ne l'utilisent pas. C'est une preuve indirecte tres faible, et je la donne pour
  ce qu'elle vaut.
- **Recevabilite** : c'est la preuve forte, et elle est negative. L'applicateur du 114 exige
  `masque = 2` sur la reference 1, soit **le type d'objet 1 et lui seul**. Une arme, qui porte un
  type d'objet distinct, fait echouer `FUN_140a10204` qui rend `-1`, et l'evenement est ignore
  sans effet. **Un `biped_board_vehicle` ne peut pas designer une arme.** La coherence du couple
  est totale : ref0 exige le type 0 (bipede), ref1 le type 1 (vehicule), conformement au nom.

Faisceau concordant, deja releve en B1 et reverifie ici :

- recherche exhaustive de `seat` : 80 chaines environ, toutes du registre vehicule
  (`VehicleSeat`, `VEHICLE_SEAT`, `vehicle_seat`, `IsDriverSeat`, `IsGunnerSeat`,
  `IsPassengerSeat`, `unit_seat_mapping`, `Vehicle_HasEmptySeat`, `biped_vehicle_switch_seat`...).
  **Aucune chaine ne relie un siege a une arme** ; `weapon_seat` n'existe pas ;
- recherche exhaustive de `zoom` : une soixantaine de chaines, toutes du registre arme, joueur ou
  reglages (`GetZoomState`, `IsZoomed`, `button_action_scope_zoom`, `player_scope_zoom_on_hold`,
  `hold_to_zoom`, `prop_is_zoom_available`, `legacy_zoom_retention`...). **Les deux vocabulaires
  sont entierement disjoints** : aucun terme ne fait le pont.

### B2c.3 L'evenement de lunette, lui, est trivial (PROUVE)

Applicateur du type 126 `unit_zoom`, atteint par `table[126] -> vtable 0x143d0da50 -> +0x78` :

```c
void FUN_14110ec20(..., undefined4 *param_3, ..., undefined1 *param_5)
{
  local_res18[0] = FUN_140a10204(*param_3, 0x1003);   // une unite (bipede, vehicule, ou type 12)
  if (local_res18[0] != -1) {
    lVar1 = FUN_140471c88(local_res18);
    *(undefined1 *)(lVar1 + 0x462) = *param_5;        // etat de lunette, un octet
  }
}
```

L'evenement de visee ecrit **un seul octet d'etat** a `unite+0x462`, avec la valeur `R(2) - 1`
etablie en B1 (`-1` = pas de lunette, `0..2` = niveaux). Aucun siege, aucune arme referencee : la
lunette est un **etat de l'unite**, pas une place occupee.

---

## B2d. Le type 103 `unit_exit_vehicle` : sortie de siege symetrique

**Oui.** Son applicateur est **`FUN_142f11e04`** (vtable `0x143d0c708`, slot `+0x78`) : il resout
l'unite (ref0, tout type) et le porteur (ref1, masque `0x1003`), **verifie que l'unite est bien
assise sur ce porteur et sur ce siege precis** (`unite+0x18 == porteur` et
`unite+0x146 == charge utile`), puis appelle la methode virtuelle `+0x168` de l'unite (sortie) ou
`FUN_14080b824(unite, 0x10)` selon un drapeau de la charge utile. La symetrie avec le 114 est
exacte : meme identifiant de siege, verification en miroir de la condition que le 114 avait
etablie.

**Ecart signale par rapport a B1** : l'applicateur du 103 lit **plus que le seul `R(6)`** —
`param_5[1]` (dont l'octet 0 sert de booleen « ejection forcee ») et l'octet a
`((char *)param_5)[5]`. La charge utile du 103 fait donc au moins 8 octets et comporte au moins
trois champs. Le lecteur `FUN_142f17b94` merite une relecture avant tout decodage du type 103 :
**la mention « 103 : `R(6)` » du lot B1 est probablement incomplete.**

---

## Niveaux de certitude

| Affirmation | Statut |
|---|---|
| `FUN_1424d80bc` appelle l'ecrivain `+0x60`, table runtime indexee par type | **PROUVE** (decompilation) |
| Trailer = sentinelle constante `0x0BCDDCBA`, mode debogage seulement | **PROUVE** |
| En-tete = 1 bit de continuation + `R(7)` type, rien avant les references | **PROUVE** (les deux cotes) |
| Bit de continuation : 1 = evenement suit, 0 = fin de liste | **PROUVE** (`FUN_14076a1c4`) |
| L'ecrivain du 114 ecrit exactement 6 bits | **PROUVE** (octets desassembles) |
| Porte d'1 bit = handle different de `-1` | **PROUVE** cote ecriture |
| Structure du slot d'evenement (types des champs cites) | **PROUVE** pour `+0x04`, `+0x14..0x1c`, `+0x20`, `+0x28`, `+0x30`, `+0x38`, `+0x40`, `+0x48` ; **SUPPOSE** pour `+0x0c`, `+0x50` |
| Aucun site d'emission n'utilise d'immediat de type | **PROUVE** pour l'absence ; **SUPPOSE** pour la cause (allocation generique parametree) |
| Fonction d'allocation du slot sortant | **NON TROUVEE** — question ouverte |
| Table des sieges : `+0xab8` / `+0xac8` / entree `0x278` / drapeau `0x400` / marqueur `+0x96` | **PROUVE** |
| Le 114 refuse une ref1 qui n'est pas du type d'objet 1 | **PROUVE** (garde de masque) |
| Type d'objet 1 = vehicule, type 0 = bipede | **SUPPOSE** (coherent avec le nom du type et la convention du moteur ; l'enumeration elle-meme n'a pas ete lue) |
| Les armes ne portent pas de sieges | **PROUVE pour cet evenement** ; non prouve dans l'absolu |
| Applicateur du 126 ecrit l'etat de lunette a `unite+0x462` | **PROUVE** |
| Applicateur du 103 = `FUN_142f11e04`, sortie symetrique | **PROUVE** |
| Charge utile du 103 plus large que `R(6)` | **PROUVE** cote applicateur ; largeur exacte **non mesuree** |
| Decalage d'un bit expliquant les « 114 » du film comme des `weapon_effect` | **SUPPOSE** — arithmetiquement exact, mais depend de la convention du decodeur employe par le lot de mesure |

## Ce qui reste ouvert

1. **L'allocation du slot d'evenement sortant.** Trois pistes epuisees (cf. B2a.4). Piste
   suivante recommandee : identifier qui insere en tete de la liste chainee `net+0x128`, en
   passant par une recherche d'instructions **restreinte a une plage d'adresses**, jamais sur
   tout le binaire.
2. **La largeur exacte de la charge utile du type 103** (relire `FUN_142f17b94`).
3. **L'enumeration des types d'objets** (`FUN_1406b68a8`) : la lire fermerait definitivement la
   correspondance « masque 2 = vehicule ».
4. **La borne du dispatcher a 123** signalee en B1 reste inexpliquee et touche toujours le type
   126. Elle doit etre tranchee avant de conclure sur une absence de paquets 126 dans un film.

## Etat de l'outillage a la cloture

Le greffon HTTP `127.0.0.1:8089` a repondu a toutes les requetes de ce lot. Un seul incident :
une recherche d'instructions par motif d'operande a expire cote MCP ; elle n'a pas ete rejouee et
le greffon HTTP est reste sain ensuite (verifie par une requete de controle). La consigne tient :
**ne pas lancer de recherche d'instructions non bornee**, preferer les xrefs et les recherches de
motifs d'octets, qui se sont montrees rapides et fiables.
