# RAPPORT R7 — La liste complete d'evenements du film : le verrou des largeurs saute, le repulseur apparait

Date : 2026-09-03. Lot R7 du `PLAN_PERCER_TRAME_FILM_2026-08-30.md`. Retro-ingenierie
STATIQUE sur le projet Ghidra existant `HI` (`C:\Users\Guillaume\Downloads\HI.gpr`,
`HaloInfinite.exe` retail) + mesure sur le corpus de films. Lecture seule des deux cotes :
aucun debogueur, aucune ecriture dans la base Ghidra, aucun DuckDB ouvert, aucun commit.

> Ce rapport est ECRIT AU FIL DE L'EAU. Les sections marquees **[ETABLI]** sont mesurees et
> publiees ; les sections **[A EPROUVER]** sont en cours. Une interruption ne doit plus rien
> couter.

Instruments (ce worktree, package `apps/go-api/internal/analysis/filmdec/`) :

| Fichier | Role |
|---|---|
| `r7_grammaire_research_test.go` | domaines des 3 references, pour les 123 types |
| `r7_charges_research_test.go` | primitives, vecteur quantifie, charges de base |
| `r7_charges_lot2_research_test.go` | sacs de proprietes nommees (82, 15, 85) |
| `r7_charges_lot3_research_test.go` | projectiles et degats (5, 6, 7, 1, 2, 106) |
| `r7_charges_lot4_research_test.go` | armes, dialogues, vehicules, corps a corps (22 types) |
| `r7_charges_lot5_research_test.go` | **types cibles : equipement, repulseur, propulseur** |
| `r7_charges_lot6_research_test.go` | types 36 et 35 (`action_weapon_fire`) |
| `r7_marche_liste_research_test.go` | le MARCHEUR de liste + temoins 1 et 2 |
| `r7_oracle_trame_research_test.go` | temoin 3 : l'oracle de trame (le juge) |
| `r7_partype_research_test.go` | oracle de trame TYPE PAR TYPE + par longueur de liste |
| `r7_variantes_research_test.go` | calibration des variantes de build et de la carte |
| `r7_cibles_research_test.go` | la chasse aux types cibles, avec denominateurs |
| `r7_largeur_research_test.go` | **l'epreuve decisive** : largeur isolee sur les listes a 1 evenement |
| `r7_oracle117_research_test.go` | **temoin de non-regression** : les 18 evenements 117 de R6 |
| `r7_repulseur_research_test.go` | confrontation du type 104 aux pistes de l'artefact |
| `r7_recensement_research_test.go` | recensement des tetes (ordonne le travail Ghidra) |

Tous sont gardes par `R7_ROOT` / `R7_IDS` (+ `R7_CAT`, `R7_MAPS`, `R7_CHUNKS`, `R7_TYPES`),
sautes par defaut, `CGO_ENABLED=0`, balayage borne. `gofmt` et
`go vet ./internal/analysis/filmdec/` verts.

---

## Verdict en cinq phrases

**Le verrou des largeurs a saute : 96,5 % des listes d'evenements du parc (91 845 sur 95 133,
12 films, 430 046 paquets delta) se marchent desormais INTEGRALEMENT, 236 321 evenements
traverses, contre « la tete seule » avant ce lot ; l'oracle de trame donne un facteur 7,8 en
profondeur au bon cadrage contre un temoin decale de +3 bits (seuil ecrit d'avance : 3), et
la marche retrouve les 18 evenements 117 de R6 A L'IDENTIQUE, au metre (0,00-0,26 m), avec le
`Script` qui les suit 18/18.** **LE REPULSEUR ET LE PROPULSEUR NE SONT PAS DANS LE FILM :
c'est un negatif MESURE, et il tient sur un test dont le cadrage est CERTAIN — la position 1
d'une liste.** **Sur 236 321 evenements traverses, 38,9 % occupent la position 1 ; un type
reellement emis doit donc y apparaitre a peu pres a hauteur de sa part, et TOUS les temoins
positifs le font (117 : 19 tetes pour 16 attendues ; `unit_zoom` : 5 903 pour 2 510 ;
`EquipmentSpawnedObject` : 356 pour 144) — tandis que 104 `EquipmentKnockbackPlayer` en
compte ZERO pour 42 attendues, 42 `biped_dodge` ZERO pour 30, 43 `initiate_mobility_action`
ZERO pour 16.** **Les 108 « occurrences » de 104 trouvees derriere une tete sont donc des
ARTEFACTS DE DERIVE, et la cause est identifiee et mesuree : la largeur de charge du type 5
`projectile_detonate` est FAUSSE et celle du type 0 `damage_aftermath` (grammaire de
PRODUCTION) est DOUTEUSE — or ces deux types sont les predecesseurs de 75 % des « cibles » ;
signature qui acheve la demonstration, la reference du pretendu 104 vaut 4224 dans la
quasi-totalite des cas, 14 valeurs distinctes sur 80.** **Le negatif de R5/R6 est donc
CONFIRME et RENFORCE : il ne portait que sur les tetes, il porte desormais sur la liste
entiere — la seule trace reelle de la famille repulseur est le type 105
`EquipmentObjectKnockedBack` (l'OBJET pousse, pas le joueur), 8 tetes sur 12 films.**

---

## 1. [ETABLI] Le mecanisme de la marche

### 1.1 Ce qu'il fallait obtenir

La liste d'un paquet delta est `[1 bit config][( 1 bit continuation ; R(7) type ; 3 refs
gardees ; CHARGE )* 0]`. Marcher la liste exige, PAR TYPE : (a) les domaines des trois
references (largeur de l'index) et (b) la largeur en bits de la charge. R6 avait bute la
(« les tailles ne sortent pas proprement »).

### 1.2 Les domaines : extraction mecanique des 123 types, et une correction a R6

Pour chacun des 123 descripteurs (annexe A de `GRAMMAIRE_EVENTS_FILM_2026-08-30`) : objet de
8 octets -> vtable -> slot `+0x58` = le thunk `domaine(i)`. Les 123 types se ramenent a **30
thunks distincts**, tous decodes sur OCTETS par un mini-interprete des opcodes rencontres
(`33C0`, `85D2`, `83EA01`, `74`, `75`, `EB`, `E9`, `0F84`, `0F85`, `B8`, `8D42`, `C3`),
simule a i = 0, 1, 2.

**PIEGE MAJEUR, ET IL A COUTE UNE PASSE.** Dix thunks se terminent par un `JNZ rel32` vers
une adresse tres lointaine. R6 (par. 2.1.1) l'a lu comme un CHEMIN D'ERREUR et en a conclu
que les references 1 et 2 « n'existent pas » pour les types 117, 104, 21, 9, 5, 6, 1, 36, 75
et 106. **C'est faux : c'est un BLOC FROID du decoupage chaud/froid du compilateur.** En
suivant le saut (lecture des octets a l'arrivee), le bloc reprend `SUB EDX,1 ; JZ <retour> ;
MOV EAX,7 ; RET` — la suite normale de la cascade. Les trois domaines existent pour TOUS les
types.

**Preuve independante que le suivi de saut est le bon** : le type 21 `unit_zoom` rend
`{4,8,7}`, exactement les domaines que le decodeur de PRODUCTION `zoom_events.go` utilise
depuis sa validation (98 % de fermeture de slot, temoin nomme 6/6). Deux chaines sans etape
commune.

Recoupements avec l'anterieur, tous verts : type 0 `damage_aftermath` = `{1,1,7}`
(NOTE_MODELE_EVENEMENTS) ; type 82 `PlayerGameEventSmall` = `{0,8,7}` (grammaire E7.3) ;
30 et 48 = `{4,8,7}`, 42 = `{2,8,7}`, 43 et 93 = `{2,0,7}`, 100 et 32 = `{1,8,7}`.
Autre correction a R6 : le type 103 est `{0,0,7}` et non `{7,0,7}` (sans consequence : les
domaines 0 et 7 font tous deux 13 bits).

### 1.3 Les charges : la regle d'or et trois corrections de primitives

Le flux de bits porte son compteur de bits consommes en `*(int *)(flux + 0x2c)` : toute
primitive qui fait `+0x2c += N` consomme N bits sur ce chemin ; une fonction sans `+0x2c` et
sans appel de lecteur consomme 0 bit. C'est la regle qui rend la derivation MECANIQUE.

Trois corrections a la grammaire du 2026-08-30, toutes verifiees au desassemblage :

1. **`0x1406d676c` n'est PAS un `R(64)` fixe.** C'est un `R(n)` GENERIQUE dont n, en bits,
   est le 4e argument (`for (; 0x3f < n; n -= 0x40) +0x2c += 0x40 ; puis +0x2c += n`).
2. **Consequence : le type 15 `Script` est AUTO-DELIMITE.** Il porte `R(10)` = la LONGUEUR EN
   BITS de sa charge brute, passee telle quelle a `0x1406d676c`.
3. **La reference var-int `0x1406d3140` n'a AUCUN bit de porte en tete.** Le prologue lit
   (base, cardinal) dans `DAT_1451f98d0 + domaine*8` et n'accede au flux qu'ensuite :
   `[R(1) sonde si domaine==1] R(w) R(2)`. Le « R(1) porte » appartient a l'EN-TETE de
   l'evenement, pas a la primitive. Dans une BOUCLE de charge (type 119), une reference du
   domaine 0 coute donc 15 bits et non 16.

Une quatrieme, propre au type 82 : sa charge se termine par **32 `R(1)` INCONDITIONNELS**
(`FUN_14080ae28`, boucle `CMP EBX,0x20 ; JL`) que la description anterieure ignorait.

### 1.4 La granularite du vecteur quantifie DIFFERE par type

La primitive `0x14076e524` prend une classe de granularite k en 4e argument, et la largeur
d'un axe en depend :

```
e(k)  = (1/120) * 2^(16-k)                       (FUN_140be9c78, DAT_143cd9758 = 1/120)
n     = min(2^22, ceil(etendue / (2*e)))
bits  = min(26, ceil(log2(n)))                   (FUN_140be9b88)
```

k = 16 pour les types 117, 32, 43, 28, 116 et le tag 7 du type 82 ; **k = 15 pour le type 5**
(`MOV R9D,0xf` au site d'appel) ; **k = 12 pour les types 6, 52, 106 et 110** (`MOV R9D,0xc`).
Sur les bornes par defaut du moteur (+/-20000) cela donne respectivement 22, 21 et 18 bits
par axe. Une largeur unique aurait desynchronise deux types sur trois.

CONTROLE de la formule : elle reproduit les `axisWidths` du catalogue de production
`map_quant_bounds.json` a k=16 (verifie a R6 : aquarius 77,8/46,2/18,1 m -> 13/12/11) ET rend
22/21/18 sur les bornes par defaut — trois valeurs relevees independamment au desassemblage
des sites d'appel.

### 1.5 Le type 36, le plus frequent du corpus, est ferme

29 038 tetes sur 430 046 paquets. Ce qui l'a debloque : le repartiteur `FUN_14080a9d4`
appelle `vtable+0x68` avec `param_5 = 1`, donc **toutes les branches `param_5 == 0` sont
MORTES dans le film**. Et la boucle des COMPOSANTES n'est PAS a largeur runtime,
contrairement a la reserve du dossier : la largeur est `base = 12 si Ncomp==1 sinon 4`,
ecrasee a `min(base,6)` quand la cible referencee porte `kind==1`.

**CONTRE-EPREUVE INDEPENDANTE** : depliee sur le temoin canonique du depot, la grammaire
place l'arme haute aux bits 44..75, l'arme basse 76..107 et la visee au bit 113 — EXACTEMENT
les offsets valides en production dans `fire_events.go`, et « post-comptes = 111 » comme
mesure par `fire_aim_modal.go` sur 5 films. Les deux bits du « +2 » sont identifies : ce sont
les portes c1 (`FUN_140c9e4d8`) et d (`FUN_1408eff64`).

### 1.6 Les variantes de build, tranchees SUR PIECES

Trois champs sont gardes par des globales du moteur ecrites au demarrage (zero dans l'image :
statiquement indecidables). Elles ont ete tranchees par l'oracle de trame, seuil facteur 2
ecrit d'avance :

| Variante | Mesure | Verdict |
|---|---|---|
| type 15, prefixe `R(15)` (`FUN_1404f25f4`) | profondeur **1,664** avec contre **0,672** sans (facteur 2,48) | **prefixe PRESENT**, cable |
| type 82 tag 7, `R(96)` brut vs vecteur quantifie (`FUN_14076f91c`) | 1,936 contre 1,921 (facteur 1,01) | NON CONCLUANT — reserve maintenue |
| type 85, queue de 68 bits | aucune liste contenant le type 85 marchee a la calibration | NON CONCLUANT |

---

## 2. [ETABLI] Les temoins de cadrage — dont un NEGATIF publie tel quel

Ecrits AVANT la mesure, conformement a la methode imposee.

### 2.1 Temoin 1 (fin de liste) : INDISCRIMINANT, et c'est dit

Le « bit de fin de liste » est UN bit : a un cadrage faux il vaut 0 une fois sur deux et la
marche se declare « finie proprement » apres zero evenement. Mesure : le taux de fin propre
est de **99,97 % au bon cadrage, 99,40 % a +1 bit et 99,98 % a +3 bits** (decalage insere
APRES la charge, la ou il attaque exactement les largeurs). **Ce temoin ne vaut rien** ; il
est publie parce que c'est la lecon deja payee par le lot damage_aftermath (« le discriminant
est la PROFONDEUR, pas le taux de fermeture »).

### 2.2 Temoin 2 (distribution des types) : conforme au recensement

Les types les plus vus en position >= 2 sont ceux du recensement des TETES (dont le cadrage
est certain) : 82, 15, 5, 0, 38, 76, 1, 75. Un desalignement rendrait des types quasi
uniformes sur 123 valeurs.

### 2.3 Temoin 3 (ORACLE DE TRAME) : LE JUGE, facteur 7,8

Apres le dernier evenement commence la trame de records d'entites. Au bon cadrage elle se
decode et VA LOIN ; a un bit pres, le premier record est deja du bruit. La largeur d'id
(`IDLowBits`, valeur de RUNTIME) est CALIBREE par film sur les paquets a LISTE VIDE, dont le
cadrage n'est pas en question.

| Parc (2 films, 6 chunks) | trames | profondeur | fermetures propres |
|---|---|---|---|
| cadrage JUSTE | 1 700 | **1,915** records/paquet | 24,5 % |
| temoin +3 bits | 1 700 | **0,245** | 91,4 % |

**Facteur 7,82** (seuil ecrit d'avance : 3). Le taux de fermeture est plus haut au temoin
FAUX qu'au bon cadrage — exactement le piege annonce en 2.1.

### 2.4 Oracle TYPE PAR TYPE : la responsabilite de chaque grammaire, isolee

Un oracle global peut rester bon alors qu'UNE grammaire est fausse. `TestR7ParType` compare
la profondeur de trame des listes QUI CONTIENNENT un type a celle des listes qui ne le
contiennent pas. Seuil ecrit d'avance : SUSPECT si la profondeur « avec » tombe sous 50 % de
la profondeur « sans », avec au moins 30 trames de chaque cote.

Sur les 24 types courants mesures (2 films) : **aucun SUSPECT**. Extraits :

| type | nom | occurrences | prof AVEC | prof SANS |
|---|---|---|---|---|
| 82 | PlayerGameEventSmall | 1 025 | 1,607 | 1,925 |
| 15 | Script | 658 | 1,603 | 1,915 |
| 36 | action_weapon_fire | 454 | **1,973** | 1,785 |
| 0 | damage_aftermath | 240 | 1,647 | 1,932 |
| 21 | unit_zoom | 47 | 2,111 | 1,834 |

### 2.5 [ETABLI] L'ORACLE 117 : non-regression au metre, 18/18

R6 avait valide la charge du type 117 au metre sur 18 evenements de 5 films (positions de
depart et d'arrivee du saut, exactes a 0,00-0,26 m) ; cette charge est depuis decodee en
PRODUCTION. La marche complete doit les retrouver A L'IDENTIQUE.

| Film | evenements 117 vus | dont en tete | valides a <= 1,5 m |
|---|---|---|---|
| `1b2d9e08` | 4 | 3 | 3 |
| `a0c36016` | 4 | 4 | 4 |
| `4577fcc4` | 6 | 6 | 6 |
| `f2966f08` | 2 | 2 | 2 |
| `faff9935` | 3 | 3 | 3 |
| **total** | **19** | **18** | **18** |

**18/18 sur les tetes, ecarts 0,00 a 0,26 m — exactement les valeurs de R6.** Taux de
validation 18/19 = 94,7 % (seuil ecrit d'avance : 90 %, et >= 18 valides). Le 19e est un
evenement 117 trouve DERRIERE une tete, dont la reference d'unite est absente : non
validable, publie tel quel.

Et la decouverte laterale de R6 est confirmee par la marche : **un type 15 `Script` suit un
117 dans 18 cas sur 18.**

C'est le TEMOIN DE NON-REGRESSION permanent de ce chantier : toute modification de la table
des largeurs doit le laisser vert.

---

## 3. [ETABLI] La couverture atteinte, et ce qui reste opaque

Parc : **12 films, 430 046 paquets delta**, dont **334 913 a liste vide** (77,9 %) et
**95 133 a liste non vide**.

- **91 845 listes marchees INTEGRALEMENT (96,5 %)** ;
- **236 321 evenements traverses** (contre « une tete par paquet » avant ce lot) ;
- profondeur par film : de 92,8 % (`084a804d`) a 98,8 % (`bf2a9f05`).

Types encore OPAQUES (la marche s'y arrete ; comptes = nombre de listes bloquees) :
`16 ShowDebugText:575 · 20 incident:155 · 12 biped_melee_clang:140 · 72 AILand:118 ·
17 Allegiance:114 · 96 NetworkedCrewEventType:102 · 84 TeamGameEventSmall:87 ·
10 weapon_effect:80 · 18 MusicTrigger:73 · 34 PromptToBootGriefer:70 ·
50 request_ai_mount_exit:69 · 19 CollectibleUnlockEvent:67 · 109 PersonalAILifceycleEffect:65 ·
68 supply_request:62 · 64 player_set_respawn_target_transform:55 · 56 request_projectile_attach:54 ·
73 AIJuke:50 · 14 PlayEffectOnObject:48 · 90 ClientOnlyShowComplete:45 · 97 SaveToUGCService:42`
(+40 autres, tous a moins de 42 listes). Ce sont des types rares : aucun n'est un obstacle a
la question du chantier.

Restent aussi opaques par construction, et c'est dit : **type 93 branche `k == 2`** (bloc
`FUN_142f262d4` gouverne par un etat hors flux), **type 30 branche `mode == 3` avec porte a
1** (le lecteur y lit une reference dont le domaine n'est pas resolu), **type 115** (presence
d'un bloc d'orientation conditionnee par un masque runtime `DAT_14473faa8`), **type 51**.

---

## 4. LA QUESTION DE FOND — repulseur et propulseur

### 4.1 [ETABLI] Les grammaires des types cibles, lues dans l'exe

| Type | Nom | Charge (bits) | Contenu |
|---|---|---|---|
| **104** | `EquipmentKnockbackPlayer` | **1 ou 30** | `R(1)` ; si 0 : direction unitaire `R(19)` + magnitude `R(10)`, echelle LOGARITHMIQUE entre 0,05 (`DAT_143cd8648`) et 20,0 (`DAT_143cd8f60`) |
| **42** | `biped_dodge` | **60 ou 65** | `R(32)` + direction unitaire `R(19)` + `R(8)` + `[R(1);si 0:R(5)]` |
| **43** | `initiate_mobility_action` | ~400, ferme | presence ; bloc optionnel ; `R(96)` brut ; 2 vecteurs quantifies k=16 ; 4 triplets `R(12)` ; 2 directions `R(24)` ; scalaires |
| **119** | `EquipmentKnockbackRequest` | `23 + 34n` | direction globale `R(19)`, `R(4)+1` = n, puis n couples (reference domaine 0 sur 15 bits, direction `R(19)`) |
| 105 | `EquipmentObjectKnockedBack` | 1 ou 33 | `[R(1) g ; si g : R(32)]` |
| 48 | `weapon_tether_request` | variable | `[R(1);si 0:R(2)]` + `[R(1);si 1:R(32)]` + `R(32)` variant-name + `R(1)` |
| 31 | `equipment_teleport_request` | 5 ou 10 | `R(4)` + `[R(1);si 0:R(5)]` |
| 98 | `Equipment` | 9 | `R(8)` + `R(1)` |

Le repulseur et le propulseur sont donc **LISIBLES** : leurs charges sont fermees et courtes.
Restait a savoir s'ils apparaissent dans le flux — c'est la mesure, pas la grammaire.

### 4.2 [ETABLI] Ce que la marche complete trouve, avec ses denominateurs

Denominateurs : **12 films · 430 046 paquets delta · 95 133 listes non vides · 91 845 listes
marchees integralement (96,5 %) · 236 321 evenements traverses**.

| Famille | Type | Occurrences | Part des evenements traverses |
|---|---|---|---|
| **REPULSEUR** | 104 `EquipmentKnockbackPlayer` | **108** | 0,046 % |
| | 119 `EquipmentKnockbackRequest` | 35 | 0,015 % |
| | 105 `EquipmentObjectKnockedBack` | 54 | 0,023 % |
| **PROPULSEUR** | 42 `biped_dodge` | **78** | 0,033 % |
| | 43 `initiate_mobility_action` | 42 | 0,018 % |
| USAGE GENERIQUE | 116 `teleport_effects` | 55 | |
| | 98 `Equipment` | 54 | |
| | 48 `weapon_tether_request` | 23 | |
| | 28 `biped_debug_teleport` | 23 | |
| | 93 `activate_spartan_ability` | 21 | |
| | 31 `equipment_teleport_request` | 14 | |
| | **30 `biped_equipment_activation`** | **0** | |
| | **51 `biped_throw_release`** | **0** | |
| | **115 `synchronized_teleport`** | **0** | |

Il aurait ete tentant de conclure ici que le negatif de R5 est refute. **C'est le contraire
qui est vrai, et la suite le demontre.**

### 4.3 [ETABLI] L'epreuve de refutation — et elle abat le resultat apparent

**L'hypothese a abattre en premier : une marche desynchronisee « trouve » des types au
hasard.** Quatre epreuves, dans l'ordre ou elles ont ete faites.

**Epreuve A — l'oracle de trame par type.** Sur 3 films riches en equipement, la profondeur
de trame des listes contenant le type contre celles ne le contenant pas : 104 rend **2,174
contre 1,511** (verdict « ok »), 105 rend 2,231, mais 42 rend **0,647 contre 1,512**
(verdict SUSPECT), 119 rend 0,607 et 48 rend 0,571. **Cette epreuve est NECESSAIRE MAIS PAS
SUFFISANTE** : elle juge la trame APRES la liste, donc elle ne separe pas « le type est
reel » de « la liste ou il apparait est par ailleurs bien cadree ».

**Epreuve B — l'identite de la reference.** Les 80 evenements 104 porteurs d'une reference
rendent **14 valeurs d'index distinctes, dont 4224 repetee dans la quasi-totalite des cas**
(mediane 4244, q1 4224). Une reference reelle designerait des unites variees, et surtout
differentes d'un FILM a l'autre. Un index constant est la signature d'une lecture qui retombe
toujours sur le meme motif de bits. La calibration de base echoue en consequence : la
meilleure base ne place que **8,8 % des index sur un slot ayant une piste** (seuil ecrit
d'avance : 80 %), contre 98 % pour la meme calibration sur `unit_zoom`.

**Epreuve C — qui PRECEDE une cible.** Si un seul type domine les predecesseurs, c'est SA
grammaire qui est fausse et la « cible » n'est que la derive qui en resulte.

| cible | predecesseurs dominants | part de ce predecesseur dans le parc |
|---|---|---|
| 104 (108 occ.) | **0 `damage_aftermath` : 81** (75 %) | 15,4 % |
| 42 (78 occ.) | 0 : 40 (51 %) et **118 `repair_complete` : 19** (24 %) | 15,4 % et **0,2 %** |
| 43 (42 occ.) | 0 : 26 (62 %) | 15,4 % |
| 98 (54 occ.) | **1 `damage_section_response` : 24** (44 %) | 3,5 % |

`repair_complete` represente 0,2 % des evenements traverses et precede 24 % des `biped_dodge` :
un enrichissement d'un facteur **100**. Aucun recit causal ne relie une reparation achevee a
une esquive. C'est une signature d'artefact, pas de protocole.

**Epreuve D — LA DECISIVE : la position 1, ou le cadrage est CERTAIN.** Sur une liste, le
premier evenement se lit sans aucune ambiguite (bit de configuration, bit de continuation,
`R(7)` type) : aucune derive ne peut s'y produire, puisque rien ne le precede. Sur le parc,
**38,9 % des evenements traverses occupent la position 1**. Un type dont l'emission ne depend
pas de la position doit donc y apparaitre a peu pres a hauteur de sa part.

| type | vus | chaine propre | **tetes** | tetes attendues |
|---|---|---|---|---|
| *temoin* 117 `EquipmentTranslocatorTeleportEffects` | 42 | 37 | **19** | 16,3 |
| *temoin* 21 `unit_zoom` | 6 459 | 6 384 | **5 903** | 2 510 |
| *temoin* 103 `EquipmentSpawnedObject` | 371 | 358 | **356** | 144 |
| *temoin* 9 `biped_pickup` | 3 861 | 3 539 | **2 936** | 1 501 |
| *temoin* 38 `weapon_reload` | 8 566 | 8 103 | **5 046** | 3 329 |
| *temoin* 76 `Dialogue2D` | 6 229 | 6 192 | **2 796** | 2 421 |
| *temoin* 100 `PowerUpApplied` | 78 | 66 | **17** | 30,3 |
| **104 `EquipmentKnockbackPlayer`** | 108 | 15 | **0** | **42,0** |
| **42 `biped_dodge`** | 78 | 8 | **0** | **30,3** |
| **43 `initiate_mobility_action`** | 42 | 7 | **0** | **16,3** |
| 119 `EquipmentKnockbackRequest` | 35 | 12 | **0** | 13,6 |
| 116 `teleport_effects` | 55 | 23 | **0** | 21,4 |
| 98 `Equipment` | 54 | 15 | **0** | 21,0 |
| 48 `weapon_tether_request` | 23 | 9 | **0** | 8,9 |
| 28 `biped_debug_teleport` | 23 | 7 | **0** | 8,9 |
| 93 `activate_spartan_ability` | 21 | 12 | **0** | 8,2 |
| 31 `equipment_teleport_request` | 14 | 2 | **0** | 5,4 |
| **105 `EquipmentObjectKnockedBack`** | 54 | 21 | **8** | 21,0 |
| 30 `biped_equipment_activation` | 0 | 0 | 0 | 0 |

**TOUS les temoins positifs sont en tete a hauteur de leur part. AUCUNE cible n'y est,
sauf le 105.** Sous l'hypothese « le 104 est un type ordinaire », la probabilite d'observer
zero tete sur 108 occurrences vaut `(1 - 0,389)^108`, soit de l'ordre de **10^-23**.

**Objection examinee, et ecartee.** On pourrait soutenir qu'un evenement de consequence n'est
jamais premier de sa liste (le repulseur suivrait toujours le degat qui le declenche). Trois
faits l'ecartent : (a) `biped_dodge` suit `repair_complete` cent fois plus souvent que le
hasard, ce qu'aucune causalite n'explique ; (b) la reference du 104 est constante a 4224 ;
(c) le type 105, qui appartient a la meme famille causale, EST en tete 8 fois. Une contrainte
causale n'aurait pas cet effet selectif.

### 4.4 [ETABLI] La cause de la derive, identifiee et mesuree

`TestR7Largeur` isole la largeur de charge d'un type en ne mesurant que les listes a **UN
SEUL evenement**, ou le cadrage est certain et ou le bit de depart de la trame ne depend QUE
de cette largeur. Sur 4 films, mediane de reference 1,793 records/paquet :

| type | listes a 1 evenement | profondeur | temoin +3 bits | verdict |
|---|---|---|---|---|
| 82 `PlayerGameEventSmall` | 1 258 | 2,050 | 0,032 | JUSTE |
| 1 `damage_section_response` | 64 | 2,016 | 0,109 | JUSTE |
| 21 `unit_zoom` | 766 | 1,983 | 0,010 | JUSTE |
| 103 `EquipmentSpawnedObject` | 63 | 1,984 | 0,032 | JUSTE |
| 39 `biped_throw_initiate` | 160 | 1,844 | 0,006 | JUSTE |
| 75 `AIDialog` | 265 | 1,842 | 0,106 | JUSTE |
| 15 `Script` | 1 276 | 1,806 | 0,453 | JUSTE |
| 38 `weapon_reload` | 415 | 1,793 | 0,039 | JUSTE |
| 76 `Dialogue2D` | 340 | 1,765 | 0,018 | JUSTE |
| 9 `biped_pickup` | 228 | 1,750 | 0,386 | JUSTE |
| 6 `projectile_impact_effect` | 218 | 1,532 | 1,183 | JUSTE |
| 7 `projectile_object_impact_effect` | 42 | 1,452 | 0,048 | JUSTE |
| 36 `action_weapon_fire` | 3 816 | 1,380 | 0,813 | JUSTE (limite) |
| **23 `authority_ignored_predicted_position`** | 75 | 1,173 | 0,027 | **DOUTEUX** |
| **0 `damage_aftermath`** | 2 297 | 1,161 | 0,288 | **DOUTEUX** |
| **5 `projectile_detonate`** | 920 | **0,308** | 0,412 | **FAUX** |

Le type 5 fait pire que son propre temoin decale : sa grammaire est fausse, sans ambiguite.
Le type 0 est la grammaire de PRODUCTION (`filmdec/weapon_hits_decode.go`,
`lot1DecodeDamageAftermath`) : elle n'a PAS ete modifiee par ce lot (lecture seule), mais elle
est ici mesuree DOUTEUSE pour la premiere fois, et c'est une decouverte a instruire.

**Ces deux types sont exactement les predecesseurs de 75 % des « cibles ». La boucle est
bouclee : les occurrences de 104, 42 et 43 sont la derive produite par ces deux largeurs.**

### 4.5 Verdict par type cible

| Famille | Verdict | Denominateurs |
|---|---|---|
| **REPULSEUR — 104 `EquipmentKnockbackPlayer`** | **ABSENT du film.** Negatif mesure. | 0 tete pour 42 attendues ; 12 films, 430 046 paquets delta, 91 845 listes marchees integralement, 236 321 evenements traverses |
| REPULSEUR — 119 `EquipmentKnockbackRequest` | ABSENT (requete client, cohérent) | idem, 0 tete pour 13,6 attendues |
| REPULSEUR — 105 `EquipmentObjectKnockedBack` | **PRESENT mais rare** : 8 tetes (cadrage certain). C'est l'OBJET pousse, pas le joueur. | idem |
| **PROPULSEUR — 42 `biped_dodge`** | **ABSENT du film.** | 0 tete pour 30,3 attendues |
| **PROPULSEUR — 43 `initiate_mobility_action`** | **ABSENT du film.** | 0 tete pour 16,3 attendues |
| 30 `biped_equipment_activation` | ABSENT, y compris en derive | 0 occurrence sur 236 321 evenements |
| 51, 115 | ABSENT | 0 occurrence |
| 93, 48, 31, 28, 98, 116 | ABSENT (0 tete) | 8 a 21 tetes attendues chacun |

**Consequence pour le chantier equipement** : le canal des evenements du film ne porte pas
l'usage du repulseur ni celui du propulseur, ni en tete ni derriere une tete. La conclusion
de R5 (« repulseur, grappin, propulseur : rien a lire ») tient, et elle est desormais etablie
sur la LISTE ENTIERE et non sur la seule tete.

### 4.6 [ETABLI] Le type 5 corrige a moitie — et la prediction NON verifiee, dite comme telle

La relecture du lecteur `0x1408096f8` apres le verdict FAUX a trouve une VRAIE erreur de
transcription : la fin du record avait ete ecrite en alternative (`si g : R(19) sinon R(8)`)
alors que le decompile donne une SUITE INCONDITIONNELLE `FUN_1406cf008` (un `R(1)` oublie)
puis `FUN_14076dc04` puis `FUN_1424cd2fc` (`R(2)`). Correction faite : la profondeur du
type 5 passe de **0,308 a ~1,41** — le niveau des types JUSTE.

Restent deux largeurs `FUN_14076dc04` qui ne se lisent pas au site d'appel (les seuls
immediats du corps sont `MOV R9D,0xF` et `MOV R9D,0x8`). `TestR7CalibreType5` balaie les
25 couples issus des largeurs de direction du moteur : le meilleur, (24, 24), rend 1,413
contre 1,328 au deuxieme — **6 % d'ecart, tres en dessous des 30 % exiges d'avance :
NON CONCLUANT.** Le type 5 reste donc hors de la liste des largeurs validees, et son couple
de largeurs est publie comme MEILLEUR CANDIDAT, pas comme derivation.

**LA PREDICTION ECRITE AVANT LA MESURE N'EST PAS VERIFIEE, et c'est dit.** J'avais ecrit que
les occurrences de cibles « tomberaient a zero ou presque » une fois les predecesseurs
fautifs corriges. Apres la correction structurelle du type 5, la mesure rejouee sur les
12 films donne : 104 passe de 108 a **123**, 42 de 78 a **89**, 43 de 42 a **44** — elles ne
tombent pas. Deux raisons, toutes deux dites : (a) le type 0 `damage_aftermath`, qui precede
75 % des cibles, reste DOUTEUX et n'a PAS ete corrige (grammaire de PRODUCTION, ce lot est en
lecture seule) ; (b) les largeurs du type 5 ne sont pas resolues. **La prediction reste donc
a eprouver, et le verdict du par. 4.5 ne repose pas sur elle** : il repose sur le test de la
position 1, dont le cadrage est certain et que la correction du type 5 laisse INCHANGE
(0 tete pour toutes les cibles, tous les temoins positifs presents).

### 4.7 [A EPROUVER] Ce qui reste

1. **Instruire le type 0** `damage_aftermath` : la grammaire de PRODUCTION
   (`filmdec/weapon_hits_decode.go`, `lot1DecodeDamageAftermath`) est ici mesuree DOUTEUSE
   pour la premiere fois (1,161 contre une mediane de 1,793, sur 2 297 listes a un seul
   evenement). C'est la decouverte laterale la plus lourde de ce lot : elle touche 872 k
   evenements de degats deja exploites en production. Hors perimetre (lecture seule), a
   porter au chantier.
2. **Resoudre les deux largeurs du type 5** au desassemblage (les registres R9D des deux
   `FUN_14076dc04` ne sont pas poses par un immediat local).
3. Fermer les ~60 types encore opaques (tous rares).
4. Rejouer alors `TestR7Cibles` et confronter a la prediction du par. 4.6.

---

## 5. Commandes exactes rejouables

`<mig>` = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`.
Toutes se lancent depuis `apps/go-api` de CE worktree.

Variables communes :

```
CGO_ENABLED=0
R7_ROOT=<mig>/data/cache/film_chunks
R7_CAT=<mig>/data/titles/halo_infinite/reference/map_quant_bounds.json
R7_MAPS="1b2d9e08=944396dd-5661-4a16-b1d8-a6053f762c55,4577fcc4=944396dd-5661-4a16-b1d8-a6053f762c55,faff9935=944396dd-5661-4a16-b1d8-a6053f762c55,a0c36016=forest,f2966f08=behemoth,000d5950=aquarius,06dfe6d9=944396dd-5661-4a16-b1d8-a6053f762c55,084a804d=bazaar,4f77afc1=944396dd-5661-4a16-b1d8-a6053f762c55,8a485699=chasm,bf2a9f05=highpower,d1dfbc02=944396dd-5661-4a16-b1d8-a6053f762c55"
R7_IDS=000d5950,06dfe6d9,084a804d,4f77afc1,8a485699,bf2a9f05,d1dfbc02,1b2d9e08,a0c36016,4577fcc4,f2966f08,faff9935
```

| Mesure | Commande | Duree mesuree |
|---|---|---|
| Recensement des tetes (par. 1.1) | `go test ./internal/analysis/filmdec/ -run '^TestR7Recensement$' -count=1 -timeout 30m -v` | 0,9 s |
| Marche complete + temoins 1 et 2 (par. 2.1-2.2) | `go test ./internal/analysis/filmdec/ -run '^TestR7Marche$' -count=1 -timeout 30m -v` | 0,4 s |
| Oracle de trame (par. 2.3) | `R7_CHUNKS=6 go test ./internal/analysis/filmdec/ -run '^TestR7OracleTrame$' -count=1 -timeout 60m -v` | 2 s |
| Oracle par type (par. 2.4 et 4.3) | `R7_TYPES=104,42,43,48,93,28,116,105,119,98,31 R7_IDS=06dfe6d9,084a804d,4f77afc1 go test ./internal/analysis/filmdec/ -run '^TestR7ParType$' -count=1 -timeout 120m -v` | 33 min |
| Oracle par longueur de liste | `go test ./internal/analysis/filmdec/ -run '^TestR7ParLongueur$' -count=1 -timeout 60m -v` | 77 s |
| Variantes de build (par. 1.6) | `R7_CHUNKS=6 go test ./internal/analysis/filmdec/ -run '^TestR7Variantes$' -count=1 -timeout 60m -v` | 11 s |
| Recensement des etiquettes du type 82 | `go test ./internal/analysis/filmdec/ -run '^TestR7Tag7$' -count=1 -timeout 30m -v` | 0,4 s |
| Calibration de carte par film | `R7_CHUNKS=3 go test ./internal/analysis/filmdec/ -run '^TestR7CalibreCarte$' -count=1 -timeout 90m -v` | 25 min |
| **Chasse aux types cibles + tableau du verdict (par. 4.2, 4.3, 4.5)** | `go test ./internal/analysis/filmdec/ -run '^TestR7Cibles$' -count=1 -timeout 90m -v` | 0,6 s |
| **ORACLE 117, non-regression (par. 2.5)** | `R7_ARTS=<mig>/data/cache/replays/halo_infinite R7_IDS=1b2d9e08,a0c36016,4577fcc4,f2966f08,faff9935 go test ./internal/analysis/filmdec/ -run '^TestR7Oracle117$' -count=1 -timeout 30m -v` | 0,8 s |
| **LARGEUR isolee par type (par. 4.4)** | `R7_CHUNKS=25 R7_IDS=000d5950,06dfe6d9,084a804d,4f77afc1 go test ./internal/analysis/filmdec/ -run '^TestR7Largeur$' -count=1 -timeout 120m -v` | 38 s |
| Repulseur contre trajectoire (par. 4.3, epreuve B) | `R7_ARTS=<mig>/data/cache/replays/halo_infinite go test ./internal/analysis/filmdec/ -run '^TestR7Repulseur$' -count=1 -timeout 60m -v` | 1,5 s |
| Calibration des largeurs du type 5 (par. 4.6) | `R7_CHUNKS=12 R7_IDS=000d5950,06dfe6d9 go test ./internal/analysis/filmdec/ -run '^TestR7CalibreType5$' -count=1 -timeout 120m -v` | 9 min |

### Instance Ghidra headless (STATIQUE, arretee en fin de lot)

Identique a l'annexe C.1 du rapport R6, y compris le contournement du piege JDK/AF_UNIX
(`-Djdk.net.unixdomain.tmpdir=Q:\nexistepas`). Seuls des endpoints de LECTURE ont ete
utilises (`/decompile_function`, `/inspect_memory_content`, `/get_function_by_address`,
`/get_xrefs_to`, `/batch_decompile`) ; `disassemble_bytes` a ete ecarte parce qu'il ECRIT
dans la base du projet.

---

## 6. Reserves et limites

1. **La calibration de carte est peu discriminante.** L'ecart entre la meilleure entree du
   catalogue et la deuxieme est souvent nul (ex. `000d5950` : 2,065 contre 2,065). Raison
   mesuree : la branche « bornes de la region » du vecteur quantifie est rare — la plupart des
   positions passent par les bornes par defaut, dont la largeur ne depend pas de la carte. La
   carte n'est donc PAS un parametre critique de la marche ; elle l'est pour le type 117.
2. **Le tag 7 du type 82 est reellement emprunte** (462 fois sur le parc : recense par
   `TestR7Tag7`, etiquettes vues `{6:404, 7:462}` dans le sac principal). La branche
   « R(96) brut vs vecteur quantifie » n'a pas ete tranchee (facteur 1,01) : c'est une reserve
   REELLE, pas theorique.
3. Les 3,5 % de listes non marchees integralement sont concentrees sur des types rares
   (par. 3) ; aucun n'est un type cible.
4. Les seuils de `TestR7ParType` exigent 30 trames de chaque cote : la plupart des types
   cibles restent sous ce seuil sur 3 films. Elargir le parc est la prochaine mesure.
5. **Le test de la position 1 suppose que l'emission d'un type ne depend pas de sa position
   dans la liste.** L'objection « un evenement de consequence n'est jamais premier » est
   examinee et ecartee au par. 4.3, mais elle n'est pas refutee dans l'absolu : elle l'est
   par faisceau (reference constante a 4224, enrichissement x100 de `repair_complete` devant
   `biped_dodge`, et le type 105 de la meme famille causale QUI EST en tete 8 fois).
6. Les largeurs du type 5 sont un MEILLEUR CANDIDAT mesure, pas une derivation (par. 4.6).

---

## 7. Ce que ce lot etablit, en une page (pour publication)

**La question posee.** Le film Theater de Halo Infinite enregistre-t-il l'usage du repulseur
et du propulseur ? Jusqu'ici le decodeur du depot ne lisait que le PREMIER evenement de
chaque paquet ; les lots R5 et R6 avaient conclu « aucune trace », mais sur cette seule tete.

**Ce qui a ete fait.** La liste ENTIERE d'evenements est desormais marchee. Il a fallu, pour
chacun des 123 types d'evenements du moteur, deriver de l'executable (a) le domaine de ses
trois references et (b) la largeur en bits de sa charge. Les domaines sont extraits
mecaniquement (30 thunks, decodes sur octets — dont dix dont R6 avait mal lu un bloc froid du
compilateur comme un chemin d'erreur). Les largeurs viennent de la decompilation des
lecteurs, avec une regle unique : le flux porte son compteur de bits consommes, toute
primitive qui l'incremente de N consomme N bits.

**Le resultat de couverture.** 12 films, 430 046 paquets delta, 95 133 listes non vides :
**96,5 % sont marchees jusqu'a leur dernier evenement, 236 321 evenements traverses**. Le
juge du cadrage — la profondeur de la trame de records qui suit la liste — donne un facteur
7,8 contre un temoin decale de 3 bits. Et la marche retrouve au metre les 18 evenements de
translocateur deja valides par le lot precedent.

**Le verdict.** Le repulseur (`EquipmentKnockbackPlayer`) et le propulseur (`biped_dodge`,
`initiate_mobility_action`) **ne sont PAS enregistres dans le film**. La demonstration ne
repose pas sur une absence : elle repose sur un test dont le cadrage est certain. Sur une
liste, le premier evenement se lit sans ambiguite, et 38,9 % des evenements du parc occupent
cette position ; **tous** les types dont la grammaire est validee y apparaissent a hauteur de
leur part (le translocateur 19 fois pour 16 attendues, la lunette 5 903 pour 2 510, la pose
d'equipement 356 pour 144), tandis que le repulseur y apparait **zero fois pour 42 attendues**
et le propulseur **zero pour 30 et 16**. Les 108 occurrences trouvees derriere une tete sont
des artefacts de derive, et leur cause est identifiee : deux grammaires encore imparfaites
(`projectile_detonate`, corrigee a moitie ici, et `damage_aftermath`, de production) qui
precedent 75 % d'entre elles ; leur pretendue reference d'unite vaut d'ailleurs la meme
valeur, 4224, dans la quasi-totalite des cas.

**La seule trace reelle de la famille** est le type 105 `EquipmentObjectKnockedBack` — un
OBJET pousse, pas un joueur — vu 8 fois en position certaine sur 12 films.

**Ce que ce lot ouvre.** La liste complete est un canal neuf : elle multiplie par ~2,5 le
nombre d'evenements lisibles par paquet, et elle rend accessibles des types qui n'etaient
jamais en tete. Elle met aussi au jour un defaut mesure dans une grammaire de production
(`damage_aftermath`, 872 k evenements de degats) qui n'avait jamais ete teste par cet oracle.
