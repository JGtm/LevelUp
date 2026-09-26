# RAPPORT — lot V3 item B : L'OCCUPANT DE L'EMBARQUEMENT (`biped_board_vehicle`), par Ghidra

> Execute le 2026-09-02 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB. Ghidra en HTTP direct sur
> `127.0.0.1:8089`, **LECTURE SEULE** (aucun `create_function`, aucun `disassemble_bytes`, aucune
> ecriture dans le projet : uniquement `read_memory`, `decompile_function`, `get_xrefs_to`,
> `search_strings`). Mesures en avant-plan, `CGO_ENABLED=0`, GOCACHE isole.
> `internal/himap/`, `cmd/vs-measure/`, `cmd/vehicle-sprite/`, `sprites_v4/` : NON touches.

## 0. LE RESULTAT EN CINQ LIGNES

1. **GHIDRA A TRANCHE, et le portage du 2026-09-01 avait TROIS domaines faux sur trois.**
   L'embarquement lit ses references en **domaines 2, 3, 7** — pas 1, 7, 7. La « sonde variable »
   qu'on lui pretait n'existe pas : `FUN_1406d3140` ne lit la sonde **que** pour le domaine 1.
2. **L'OCCUPANT DE L'EMBARQUEMENT EST RESOLU** : reference 0, domaine 2, `slot = base + R(8)`.
   Sur 22 embarquements (8 films), il tombe dans la bande bipede **22/22 = 100 %** (il n'y tombait
   JAMAIS avant : le dump de reference rendait `occ_slot=3160 inBand=false`, il rend maintenant
   `occ_slot=561 inBand=true`), et il **ouvre un trou du flux de position a l'instant exact dans
   77,3 % des cas, temoin decale 0,0 %**.
3. **LA LARGEUR DU DOMAINE 3 EST 7 BITS, ET C'EST LA MESURE QUI L'A DIT** (la table de largeurs
   est en BSS, illisible statiquement). Oracle independant : le SIEGE, lu apres les trois refs.
   A 7 bits, le siege de l'embarquement egale celui de la sortie appariee **5/6 = 83,3 %** et vaut
   **0 sur 21/22** — le meme « siege dominant 0 » que la sortie. A 8 bits : **0/6**. A 13 bits : 4/6.
4. **LE GATE DE RECOUPEMENT A 90 % N'EST PAS ATTEINT (77,3 %), MAIS LA REFERENCE NON PLUS.** Le
   controle de non-regression — la SORTIE, deja validee — rend **90,7 %** sur le meme corpus avec
   la meme primitive. L'embarquement est donc a **85 % du taux de la reference**, avec un temoin
   NUL : ce qui manque tient a la primitive du trou, pas au decodage.
5. **LE DEVENIR D'UN EMBARQUEMENT** : 6/22 (27,3 %) sont suivis d'une SORTIE du meme occupant, et
   ces 6 sont des trajets COMPLETS (le trou ouvert a l'embarquement est referme par CETTE sortie),
   **ecart de numerotation 0 sur 6/6** entre domaine 2 et domaine 1. La branche MORT, mesurable
   seulement cote `replay`, rend **2 morts / 5** contre **1 / 5** au temoin. Ensemble, sur les
   films ou les deux branches sont mesurables : **4 embarquements sur 5 (80 %) finissent par une
   sortie du meme occupant OU par sa mort** — n=5, et c'est dit.

---

## 1. CE QUE GHIDRA A LU, ET COMMENT (chemin reproductible)

Le rapport `PORT_LISTE_EVENEMENTS_2026-09-01.md` § 1.2 s'etait arrete sur : « la contrepartie
statique 0x144724A90 pointe vers des descripteurs dont les vtables emploient des thunks relocates
(l'image statique les rend en 0xcc) ». **Deux constats ont debloque la lecture.**

### 1.1 0x144724A90 n'etait pas la bonne table

Lue en memoire, elle contient des pointeurs vers `0x143d03fc0..0x143d04120`, un bloc de CHAINES —
`"AuntieDot"`, `"DespawnSad"`, `"EmoteGreeting"`, `"HackingFailure"`... : c'est une table de noms
d'annonces/medailles, pas les handlers d'evenements. La piste etait fausse, ce qui explique
l'impasse du 01/09.

### 1.2 Le bon chemin : la chaine de nom -> le thunk -> la vtable

Le dispatcher `FUN_14080AADE` (redecompile, inchange) confirme la grammaire au bit pres :

```c
if ((ulonglong)(longlong)(int)uVar9 < 0x7b) {                     // type < 123
  plVar1 = *(longlong **)(*(longlong *)(param_1+0x18) + 0x210 + (longlong)(int)uVar9 * 8);
  do {                                                            // EXACTEMENT 3 references
    cVar3 = FUN_1406cf008(param_3);                               // bit de garde
    if (cVar3 != '\0') {
      (**(code **)(*plVar1 + 0x58))(plVar1, puVar8);               // -> DOMAINE de la ref n° puVar8
      FUN_14080ada0(uVar9);
      FUN_1406d3140();                                             // id-reader
    }
    uVar14 = (int)puVar8 + 1; puVar8 = (undefined1 *)(ulonglong)uVar14;
  } while ((int)uVar14 < 3);
  ...
  cVar3 = (**(code **)(*plVar1 + 0x68))(plVar1, local_res18, puVar15, param_3, 1); // charge
```

`FUN_14080ada0(type)` donne la cle du parcours : il rend le NOM d'un type via
`(**(code **)(**(longlong **)(DAT_144e61d88 + 0x210 + type*8) + 8))()`. **Donc vtable+0x08 = le
getter de nom.** D'ou la chaine, entierement statique :

| etape | outil | resultat |
|---|---|---|
| 1. la chaine | `search_strings "board_vehicle"` | `"biped_board_vehicle"` @ **0x143c97f80** |
| 2. le thunk de nom | `get_xrefs_to 0x143c97f80` | **0x14119e9b0** (`48 8d 05 …  c3` = `LEA RAX,[rip+off]; RET`) |
| 3. la vtable | `get_xrefs_to 0x14119e9b0` | slot **0x143d0d338** = vtable+0x08 -> **vtable = 0x143d0d330** |
| 4. le lecteur de domaine | `read_memory 0x143d0d330+0x58` | **0x142f1556c** |

Meme chaine pour la sortie : `"unit_exit_vehicle"` @ 0x143c3a058 -> thunk 0x14119eaf0 -> vtable
**0x143d0c708** -> vtable+0x58 = **0x14080a018**. (Les deux vtables sont referencees par une table
de descripteurs en `0x144724c20..0x144724ff0` ; son index n'est PAS le numero de type — verifie,
et sans importance : les vtables sont identifiees par leur nom, pas par leur rang.)

### 1.3 Les deux aiguillages, decodes octet par octet

Ni `0x142f1556c` ni `0x14080a018` n'est defini comme fonction dans le projet Ghidra (d'ou le
`No function found` du decompilateur). Ils ont donc ete lus en **octets bruts** (`read_memory`,
sans rien ecrire dans le projet) et decodes a la main.

**EMBARQUEMENT — `0x142f1556c`** (`edx` = indice de la reference) :

```
85 d2              test edx,edx
74 11              je   +0x11        -> mov eax,2 ; ret      ref 0 -> DOMAINE 2
83 ea 01           sub  edx,1
74 06              je   +0x06        -> mov eax,3 ; ret      ref 1 -> DOMAINE 3
b8 07 00 00 00     mov  eax,7 ; ret                          ref 2 -> DOMAINE 7
```

**SORTIE — `0x14080a018`** (le CONTROLE de la methode) :

```
85 d2              test edx,edx
74 0a              je   +0x0a        -> mov eax,1 ; ret      ref 0 -> DOMAINE 1
b8 07 00 00 00     mov  eax,7
83 ea 01           sub  edx,1
75 05              jne  +0x05        -> ret (eax=7)          ref 2 -> DOMAINE 7
                                      sinon mov eax,1        ref 1 -> DOMAINE 1
```

La sortie rend **1, 1, 7** — exactement la grammaire deja validee par la mesure le 01/09
(occupant en bande 95,5 %, siege 0 a 93,8 %). **La methode de lecture est donc controlee par un
cas dont la reponse etait connue d'avance**, avant d'etre appliquee a l'embarquement.

### 1.4 La sonde n'existe que pour le domaine 1

`FUN_1406d3140(_, reader, domaine, out)`, decompile :

```c
uVar8 = (&DAT_1451f98d0)[domaine*2];  uVar7 = (&DAT_1451f98d4)[domaine*2];   // base, taille
if (((param_3 == 1) && (cVar2 = FUN_1406cf008(param_2), cVar2 != '\0')) && ...) {
  uVar8 = DAT_1451f98f0;  uVar7 = DAT_1451f98f4;                             // variante SONDE
}
iVar3 = FUN_1406d310c(uVar7);            // largeur = ceil(log2(taille))
... lit R(iVar3) puis R(2) generation ; *param_4 = gen<<30 | (base + low);
```

Trois consequences portees dans le code :

1. `param_3 == 1` : **aucune** des trois refs de l'embarquement ne porte de sonde. La « sonde
   variable » observee le 01/09 n'etait que le **premier bit de l'index**.
2. La largeur est **runtime** : `FUN_1406d310c(count) = ceil(log2(count))` sur `DAT_1451f98d0`,
   qui est en **BSS** (lue a 0 dans l'image statique). Elle ne peut donc PAS etre lue par Ghidra —
   d'ou le balayage de largeurs mesure au § 2.
3. Le codec rend `base + low` : la base est per-domaine, runtime. Le § 2.4 mesure qu'entre le
   domaine 2 et le domaine 1 elle est **identique** (ecart 0 sur 6/6).

---

## 2. LA MESURE — le gate ecrit avant, les chiffres apres

Instrument : `apps/go-api/internal/analysis/filmdec/event_list_board_test.go` (+ ses deux fichiers
frere pour la sortie journalisee et le garde-rail), garde `V3_BOARD_FILMS` / `V3_BOARD_ROOT`.
Corpus : les 8 films les plus riches en embarquements du corpus V1a (recensement prealable, mode
`V3_BOARD_COUNT`) — `e232ffce`, `829abef9`, `53ce4390`, `d332c3a9`, `21468645`, `4898d586`,
`a89a3d23`, `0d76e8f1`. **22 embarquements, 75 sorties, 174 trous >= 3 s.**

Note de methode : le nuage bipede est balaye en **quanta seuls** (`QuantaOnly`). Un TROU est une
absence d'ECHANTILLON dans le temps, pas une absence de coordonnee — et la moitie du corpus a
embarquements (CTF / Slayer hors Behemoth SF) n'a pas de bornes de carte au catalogue.

### 2.1 Les gates, ecrits avant la mesure

| gate | enonce | seuil |
|---|---|---|
| **B1** | l'embarquement OUVRE un trou du flux de position de son occupant (+/-2 s), temoin decale de 37 s | `>= 90 %` |
| **B2** | l'occupant (`base + index`) tombe dans la bande bipede | `>= 90 %` |
| **B3** | *controle* : la SORTIE, sur le meme corpus, garde ses chiffres publies | `>= 90 %` |
| **B4** | l'embarquement est suivi d'une SORTIE du MEME occupant | `>= 90 %` |
| **B5** | (cote `replay`) l'occupant MEURT pendant son trou, au-dessus du temoin | `> +10 pts` |

### 2.2 Le balayage de largeurs — le domaine 2 vaut 8, sans ambiguite

| (dom2, dom3, dom7) | occupant en bande | ouvre un trou | TEMOIN |
|---|---|---|---|
| (7, 7, 13) | 22/22 = 100 % | **1 = 4,5 %** | 0 % |
| **(8, 7, 13)** | **22/22 = 100 %** | **17 = 77,3 %** | **0 %** |
| (8, 8, 13) | 22/22 = 100 % | 17 = 77,3 % | 0 % |
| (9, 8, 13) | 16/22 = 72,7 % | 1 = 6,2 % | 0 % |
| (13, 8, 13) | 2/22 = 9,1 % | 0 = 0,0 % | 0 % |

Le maximum est unique et net : **dom2 = 8**. A 7 bits la bande est encore tenue (un index de
7 bits tombe aussi dans une bande de ~100 slots) mais le recoupement s'effondre a 4,5 % — c'est le
TROU, pas la bande, qui departage. A 9 et 13 bits tout tombe.

### 2.3 Le siege departage le domaine 3 : 7 bits

Le siege se lit apres les trois refs ; il depend donc des trois largeurs. Oracle : il doit
s'accorder avec le siege de la SORTIE appariee (le meme joueur, le meme vehicule, le meme siege).

| (dom2, dom3, dom7) | siege = celui de la sortie appariee | histogramme des sieges |
|---|---|---|
| (8, 6, 13) | 0/6 = 0,0 % | `32 x 22` |
| **(8, 7, 12/13/14)** | **5/6 = 83,3 %** | **`0 x 21`, `61 x 1`** |
| (8, 8, 11..14) | 0/6 = 0,0 % | `2 x 10  1 x 7  3 x 5` |
| (8, 9, 13) | 0/6 = 0,0 % | `5 x 10  3 x 7  7 x 4` |
| (8, 13, 13) | 4/6 = 66,7 % | `0 x 14  58 x 7  40 x 1` |

**dom3 = 7** : accord maximal, et l'histogramme rend le **siege 0 sur 21/22**, la meme forme que
la sortie (« le conducteur »). C'est ce qui a ete porte. `dom7 ∈ {12,13,14}` donnent le meme
resultat — la ref 2 est gardee-absente en pratique, comme deja mesure pour la sortie ; 13 est
conserve.

Effet sur la validation deja publiee (`TestEvtVehicleValidate`, 5 films) :

| grandeur | avant (portage 01/09) | apres (Ghidra + mesure 02/09) |
|---|---|---|
| occupant en bande (board + exit) | 95,5 % (et **aucun** board en bande) | **68/68 = 100 %** |
| siege board (n = 16) | `16 x 5`, puis 40, 56, 8, 0, 46 | **`0 x 16`** |
| dump `0d76e8f1` | `BOARD occ_slot=3160 inBand=false seat=16` | **`BOARD occ_slot=561 inBand=true seat=0`** |

### 2.4 Le devenir d'un embarquement, et le controle de non-regression

| gate | mesure (8 films) | verdict |
|---|---|---|
| **B1** recoupement ouverture de trou | **77,3 %** (17/22), temoin **0,0 %** | **ECHOUE** (seuil 90 %) |
| **B2** occupant en bande | **100,0 %** (22/22) | **PASSE** |
| **B3** *controle* SORTIE | en bande **75/75 = 100 %**, ferme un trou **68/75 = 90,7 %**, temoin **0,0 %** | **PASSE** |
| **B4** suivi d'une sortie du meme occupant | **27,3 %** (6/22) ; dont trajet complet **6/6 = 100 %** | **ECHOUE** |
| **B5** *(replay, 4 films Behemoth)* mort de l'occupant | **2/5 = 40 %** contre temoin **1/5 = 20 %** | **PASSE** (+20 pts, n=5) |

**Diagnostic d'ecart de base** (l'hypothese qui restait a eliminer : deux domaines numerotant le
meme bipede depuis deux bases differentes) : sur les 6 trous ouverts a un embarquement et refermes
par une sortie, l'ecart `slot(sortie) - slot(embarquement)` vaut **0 dans 6 cas sur 6**. Les
domaines 1 et 2 partagent la numerotation ; il n'y a **pas** de base a corriger.

**Lecture de B1** : le seuil de 90 % etait fixe sur le souvenir du 10/10 de V2b, obtenu sur UN
film avec le flux monde-filtre. Sur 8 films et le flux quanta, **la reference elle-meme
(la sortie) tombe a 90,7 %** — l'embarquement, a 77,3 % avec un temoin nul, est a **85 % du taux
de la reference**. Ce qui manque tient a la primitive (un trajet de moins de 3 s n'ouvre aucun
trou ; un joueur deja silencieux avant d'embarquer non plus), pas au decodage. Le gate est
neanmoins declare **ECHOUE** : il etait absolu, il le reste.

**Lecture de B4** : seuls 27,3 % des embarquements ont leur sortie dans le film. Ce n'est pas une
faiblesse du decodage — les 6 apparies sont des trajets PARFAITS (100 % de trajet complet, ecart
de base 0/6). C'est la consequence du ratio corpus **board:exit = 1:15** deja mesure : la grande
masse des montees en vehicule n'emet PAS de `biped_board_vehicle`. L'evenement de type 8 est un
sous-ensemble particulier, et la sortie correspondante manque souvent (elle peut aussi etre portee
en 2e position d'une liste d'evenements — borne basse documentee au § 5 du portage).

**Un bug d'instrument corrige en cours de route, note car il fabriquait un faux negatif** : la
premiere version appariait chaque embarquement contre une liste de sorties encore en cours de
remplissage dans la MEME boucle. Un embarquement ne voyait donc que les sorties ANTERIEURES,
c'est-a-dire aucune de celles qui l'interessent : B4 rendait **0/22**, ce qui ressemblait a une
refutation. Deux passes explicites (toutes les sorties d'abord) rendent 6/22. La lecon vaut d'etre
gardee : un appariement « evenement suivant » ne se code jamais dans la boucle qui construit la
liste.

---

## 3. CE QUI EST PORTE EN PRODUCTION

`apps/go-api/internal/analysis/filmdec/event_list.go` — **additif, la sortie n'est pas touchee** :

- `dom2RefWidth = 8`, `dom3RefWidth = 7` (constantes commentees : origine executable pour les
  DOMAINES, mesure pour les LARGEURS, avec les chiffres de chaque oracle) ;
- `decodeExitRefs` : la sortie, extraite telle quelle dans sa propre fonction, avec l'adresse de
  sa vtable en commentaire ;
- `boardRefs` + `decodeBoardRefs` : l'embarquement, refs en domaines 2/3/7, **sans sonde**,
  occupant = `base + R(8)`, avec l'aiguillage desassemble en commentaire ;
- `decodeVehicleEvent` reduit a l'aiguillage entre les deux, sa documentation corrigee ;
- `VehicleEvent.OccupantSonde` : documente comme toujours nul pour un embarquement.

Aucune ligne de `traverse.go` / `frame_records.go` / `fire_events.go` touchee. Le garde-fou de
cadrage tient : `TestEvtHeadHistogram` rejoue `fire_events == head type36` a l'unite, et
`TestEvtCorpusCounts` rend les memes **348 board / 5 144 exit** sur 949 films.

**Garde-rail neuf, SANS environnement ni film** :
`internal/analysis/filmdec/event_list_board_grammar_test.go`, `TestBoardEventGrammar`. Il fabrique
un payload synthetique (bit de config, continuation, `R(7)=8`, refs 2/3/7, `R(6)` siege) et exige
l'occupant, l'absence de sonde et le siege attendus. **Temoin integre** : les MEMES bits de charge
relus par la grammaire de la SORTIE doivent rendre un occupant DIFFERENT — sinon le test ne
prouverait rien. Il tombe si quelqu'un remet une sonde sur la ref 0, reordonne les domaines ou
change une largeur.

---

## 4. STATUT DES ITEMS

| item | statut | justification |
|---|---|---|
| Trouver l'ordre/domaine des refs de l'embarquement (Ghidra) | `[x]` | domaines **2, 3, 7**, lus dans l'executable ; methode controlee sur la sortie (1, 1, 7 = grammaire deja validee). |
| Porter dans `event_list.go`, additif, sans casser les sorties | `[x]` | `event_list_test.go` vert ; occupant en bande passe de 95,5 % a **100 %** ; comptes corpus inchanges. |
| Ajouter un cas board | `[x]` | `TestBoardEventGrammar`, non garde, avec temoin — il tourne dans la suite ordinaire. |
| Gate : embarquement -> sortie du meme occupant (ou mort) + debut de trou, >= 90 % | `[!]` **partiellement atteint** | debut de trou **77,3 %** (temoin 0 %) contre **90,7 %** pour la reference sortie ; sortie du meme occupant **27,3 %** (mais 6/6 trajets parfaits, ecart de base 0/6) ; mort **40 % vs 20 %** au temoin sur n=5. Le seuil absolu de 90 % n'est pas atteint, et la raison est mesuree : la primitive du trou plafonne a 90,7 % sur la reference, et le ratio board:exit = 1:15 prive la plupart des embarquements de leur sortie. |
| Rapport + thought_log | `[x]` | ce fichier ; entree en tete de `.ai/thought_log.md`. |

**Ce qui reste ouvert** (non traite, note sans etre traite — regle du perimetre) :

1. **La largeur runtime par film.** `dom2=8` / `dom3=7` sont les valeurs de la build de reference,
   confirmees par la mesure sur 8 films. Un film a table de domaine differente les decalerait.
   La lecture propre passerait par `DAT_1451f98d0` **a l'execution** (debogueur), pas par l'image.
2. **Le domaine 3 (ref 1) et le domaine 7 (ref 2) ne sont pas interpretes** : leur largeur est
   mesuree, leur SENS non. La ref 2 (domaine 7, celui des objets du monde) est le candidat naturel
   pour le VEHICULE, mais elle est gardee-absente dans la quasi-totalite des cas mesures — donc
   l'evenement d'embarquement ne nomme pas le vehicule de facon exploitable.
3. **Marcher la liste ENTIERE** (evenements non-tete) reste la voie pour retrouver les sorties
   manquantes de B4, et porterait les comptes de la borne basse a l'exact.

---

## 5. FICHIERS

| fichier | etat | lignes |
|---|---|---|
| `apps/go-api/internal/analysis/filmdec/event_list.go` | **MODIFIE** (production, additif) | 317 |
| `apps/go-api/internal/analysis/filmdec/event_list_board_test.go` | neuf (mesure, garde env) | 413 |
| `apps/go-api/internal/analysis/filmdec/event_list_board_report_test.go` | neuf (sortie journalisee) | 135 |
| `apps/go-api/internal/analysis/filmdec/event_list_board_grammar_test.go` | neuf (**garde-rail sans env**) | 88 |
| `apps/go-api/internal/analysis/replay/vehicules_v3_trous_test.go` | etage 4 ajoute (gate B5) | 411 |

Commandes de rejeu (avant-plan, GOCACHE isole, `CGO_ENABLED=0`) :

```
# recensement prealable : quels films portent des embarquements (sans decoder les positions)
V3_BOARD_COUNT=1 V3_BOARD_ROOT=<data>/cache V3_BOARD_FILMS="<short8>,..." \
  go test ./internal/analysis/filmdec/ -run TestV3BoardOccupant -v -timeout 60m

# la mesure : balayage de largeurs, gates B1-B4, diagnostic d'ecart de base
V3_BOARD_ROOT=<data>/cache \
V3_BOARD_FILMS="e232ffce,829abef9,53ce4390,d332c3a9,21468645,4898d586,a89a3d23,0d76e8f1" \
  go test ./internal/analysis/filmdec/ -run TestV3BoardOccupant -v -timeout 60m

# gate B5 (la branche MORT), cote replay : exige les bornes de carte
V3_DESTR_ROOT=<data>/cache \
V3_DESTR_FILMS="0d76e8f1:behemoth,21468645:behemoth,4898d586:behemoth,a89a3d23:behemoth" \
  go test ./internal/analysis/replay/ -run TestV3DestructionDatee -v -timeout 60m

# non-regression du portage du 01/09
EVT_CHUNK_DIR=<data>/cache/film_chunks/0d76e8f1 EVT_CACHE=<data>/cache/film_chunks \
EVT_FILMS="0d76e8f1,e232ffce,829abef9,53ce4390,4898d586" \
  go test ./internal/analysis/filmdec/ -run TestEvt -v -timeout 60m
```

Suite sans environnement (tout ce qui est garde saute ; `TestBoardEventGrammar` tourne) :

```
go test ./internal/analysis/filmdec/ ./internal/analysis/replay/ -count=1
ok  levelup/go-api/internal/analysis/filmdec   1.559s
ok  levelup/go-api/internal/analysis/replay   28.942s
EXIT=0 · grep -c '^--- FAIL:' = 0 · gofmt -l vide · go vet propre
```
