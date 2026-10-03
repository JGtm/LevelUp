# Ou le film porte l'equipe de chaque joueur

Date : 2026-09-12, phase 3. Suite de `NOTE_SECTION3_CHUNK00_2026-09-12.md` (phase 1) et
`NOTE_SECTION3_SLOTS_2026-09-12.md` (phase 2), dont elle ferme la question ouverte n°1
(« ou est l'equipe ? »). Travail **hors ligne, lecture seule** : desassemblage statique de
`HaloInfinite.exe` (Ghidra, instance partagee, aucun renommage ni sauvegarde) + mesures sur les
`chunk_00` et les paquets de type 2 du cache local. Aucun code de production modifie, aucun
commit.

Instruments (tous sous garde d'environnement, sautes en CI) :

| fichier | ce qu'il mesure |
|---|---|
| `apps/go-api/internal/analysis/filmdec/equipe_film_recon_research_test.go` | le recensement : l'archetype ti=9 relu dans le registre du FILM, le nombre d'entites ti=9 par image-cle, leur emprise, la regle des paquets modaux |
| `apps/go-api/internal/analysis/filmdec/equipe_film_designateur_research_test.go` | la mesure du decalage du champ, par balayage, avec ses six controles ecrits d'avance |
| `apps/go-api/internal/analysis/filmdec/equipe_film_oracle_research_test.go` | la confrontation a l'oracle externe `match_participants.team_id`, le balayage complet, le plancher de bruit voisin |

---

## Resume execute

1. **L'EQUIPE EST DANS LE FILM, ET ELLE N'EST PAS DANS `chunk_00`.** Elle est dans la
   **trame d'etat** (paquets de type 2), portee par le composant
   **`managed-player-team-designator-component`** de l'archetype **ti=9** (« managed-player »),
   **composant i0**, sur **4 bits**, a **186 bits du debut du record d'image-cle**. La valeur
   ecrite vaut le **designateur d'equipe PLUS UN** : brut `0` = aucune equipe, brut `1` =
   equipe 0, brut `2` = equipe 1, et ainsi de suite.
2. **La preuve tient par deux chaines sans etape commune.** Chaine 1, l'executable : le
   descripteur du composant (`0x143d08ad0`) donne le nom en `+0x08`, l'ECRIVAIN en `+0x18`
   (`0x142edbd3c`) et le LECTEUR en `+0x30` (`0x140f581e8`) ; ce dernier n'appelle que
   `FUN_1407ef804`, dont le desassemblage donne la largeur (`ADD dword ptr [RCX+0x2c],0x4`) et
   la convention de valeur (`SHR R9,0x3c ; DEC R9B` = brut moins un). Chaine 2, les octets :
   **16 films sur 18 en accord EXACT terme a terme** avec `match_participants.team_id`,
   **160 slots sur 176**, dont **deux Grandes batailles a 24/24**.
3. **LE DECALAGE EST LE SEUL DE TOUT LE RECORD A SATISFAIRE L'ORACLE.** Balayage des 456 ou
   457 decalages possibles d'un record de 459/460 bits, sur les 18 films apparies :
   **exactement UN decalage rend l'accord exact, et c'est 186, sur les 16 films a equipes.**
   Plancher de bruit voisin : **0 touche sur 576** decalages voisins (32 par film).
4. **TEMOIN NEGATIF NATUREL, ET IL EST PROPRE.** Les deux films de FFA du cache
   (`1950c59b` Arena FFA Slayer, `610363ee` Shotty Snipe FFA, 10 joueurs, 10 equipes a la base)
   lisent **`0` sur les huit entites** : le moteur ne donne AUCUN designateur d'equipe en FFA.
   Ce sont les deux seuls films qui ne s'accordent pas a la base — et le desaccord vient de la
   BASE, qui fabrique un `team_id` par joueur la ou le film dit « aucune equipe ».
5. **LES ENTITES ti=9 SONT EXACTEMENT LES JOUEURS, ET LEUR ORDRE EST CELUI DE `chunk_00`.**
   8 entites par image-cle en arene, **24 en Grande bataille**, a slots consecutifs de pas 2 et
   de longueur constante. L'ordre des records concorde avec l'ordre de la table des slots de
   `chunk_00`, dont la phase 2 a etabli qu'il EST le `player_index` de production.
6. **Un designateur ne change jamais pour une entite donnee ; ce qui change, c'est quelle
   entite.** Controle interne, **22 films sur 22** : le vecteur des designateurs bouge d'une
   image-cle a l'autre **si et seulement si** la suite des slots des entites ti=9 bouge aussi
   (slot reattribue, arrivee, depart). Sur les 16 films a suite de slots constante :
   **0 entite change sur 19 a 55 images-cles**.
7. **Question ouverte de la phase 2 FERMEE, et une affirmation de la PRODUCTION REFUTEE.**
   `replay/document.go:577` declare « `Team` vaut -1 : L'EQUIPE N'EST PAS DANS LE FILM » et
   `replay/build.go:577` l'ecrit en dur. C'est faux : le film la porte, et cette note dit ou.
8. **La branche « chunk de type 8 PLAYER_METADATA » est REFUTEE pour ce corpus, par la
   mesure.** Les 1 351 manifestes du cache ne declarent que **trois types de chunk** :
   1 (x1 351), 2 (x37 661), 3 (x1 351). **Aucun chunk de type 8, aucun de type 12.** Il n'y a
   pas de roster de type 8 a lire dans ces films.
9. **Le consommateur nomme les equipes, et la table est lue.** L'enumeration de script
   `mp_team_designator` (« Enum for MP team designators », descripteur `0x1445c0c00`, **9
   entrees** a `0x144723da0`) vaut, dans l'ordre : `First`, `Second`, `Third`, `Fourth`,
   `Fifth`, `Sixth`, `Seventh`, `Eighth`, `Neutral`. Neuf valeurs, ce qui ferme avec le test de
   validite du lecteur de la composante globale (`valeur + 1 < 10`, donc -1..8).
10. **Le composant `game-engine-team-mapping-component` n'est PAS l'equipe d'un joueur** — et
    le depot le portait deja sans le savoir. Son etat fait 20 octets, six champs puis un
    tableau de **huit** octets signes gates par un masque a `+0x06` : c'est une table **par
    equipe**, pas par joueur. Son vidangeur de debug (`FUN_142f1b44c`) nomme ses six champs :
    `team-mapping`, `shared-team-lives`, `current-state`, `game-finished`, `current-round`,
    `round-timer`. Le depot le consomme deja dans `components_team_mapping.go`.

---

## A. La chaine de l'executable, pas a pas

Elle part du CONSOMMATEUR (methode, regle 1) et ne touche aucun film.

```
chaine "managed-player-team-designator-component"  @0x143c953c0
  -> thunk de nom  0x141177eb0    (LEA RAX,[chaine] ; RET)
  -> descripteur   0x143d08ad0    le thunk de nom est en +0x08
       +0x00  0x14117b4a0
       +0x08  0x141177eb0   NOM
       +0x10  0x1404ab600
       +0x18  0x142edbd3c   ECRIVAIN (serialiseur)
       +0x20  0x1411c8f80
       +0x28  0x14076ce9c
       +0x30  0x140f581e8   LECTEUR (deserialiseur)
       +0x38  0x1404ab600
       +0x40  0x141191ab0
       +0x48  0x14076ced0
  -> lecteur 0x140f581e8 : un seul appel, FUN_1407ef804(flux, flux, *(param_3+0x10))
  -> FUN_1407ef804 : ADD dword ptr [RCX+0x2c],0x4   -> 4 BITS
                     SHR R9,0x3c                    -> les 4 bits de tete du mot
                     DEC R9B                        -> STOCKE = BRUT - 1
  -> enumeration mp_team_designator : 9 noms, First..Eighth, Neutral
```

**La forme du descripteur est verifiee sur un second exemplaire**, ce qui la rend lisible et pas
devinee : celui de `game-engine-team-mapping-component` (`0x143d0f7a0`) a la meme disposition
(nom en `+0x08` = `0x141173050`, ecrivain en `+0x18` = `0x142f068bc`, lecteur en `+0x30` =
`0x140f58200`), et le depot avait deja porte CE lecteur-la en notant « deser thunk
(vtable+0x28) @140f58200 » — meme adresse, meme emplacement, compte a partir du champ de nom.
Le pas du tableau de descripteurs est `0x50` (dix pointeurs), mesure sur trois entrees
consecutives a `0x143d0f700`, `0x143d0f750`, `0x143d0f7a0`.

### A.1 La convention de valeur, et pourquoi elle est une PREDICTION

`DEC R9B` n'est pas un detail d'implementation : il fixe le codage. Le champ de 4 bits porte
`designateur + 1`, donc `0` est representable et vaut **-1 = aucune equipe**. Le lecteur de la
composante globale `game-engine-team-mapping-component` le confirme par son test de validite :
il refuse un octet dont `valeur + 1` depasse `9` (`INC AL ; CMP AL,0x9 ; JA`), soit un domaine
`-1..8` — exactement les neuf designateurs plus « aucun ».

De la sort une prediction ecrite AVANT toute mesure sur les films : **`brut = team_id + 1`**.
Pas « correle a », pas « a une permutation pres » : egal, terme a terme. C'est elle qui est
testee en section C.

---

## B. Ce que la trame porte, mesure sans verdict

Recensement sur les paquets de type 2, `KeyframeRecordSpans` (le marcheur de la table
d'image-cle du depot, valide 249/250 par une autre voie).

| mesure | valeur |
|---|---|
| archetype ti=9, composant i0, relu dans le registre du FILM | `managed-player-team-designator-component` sur **22/22 films** |
| entites ti=9 par image-cle, arene et FFA | **8** |
| entites ti=9 par image-cle, Grande bataille | **24** |
| slots des entites ti=9 | consecutifs, **pas de 2** (ex. 1297, 1299, ..., 1311) |
| longueur d'un record ti=9 | **460 bits** en `HI_1_13_0`, **459** sur les autres builds du lot |
| images-cles portant ti=9, par film | 9 a 55 |

Le compte de 8 et celui de 24 ne sont pas cherches : le marcheur ignore tout des modes de jeu.
Sur `000d5950`, la premiere image-cle du film (`chunk_01`) ne porte AUCUN record ti=9 — les
entites `managed-player` n'y apparaissent qu'a partir de `chunk_02`. C'est la raison pour
laquelle un recensement limite au premier paquet de type 2 rendait zero et concluait a tort a
l'absence du composant.

---

## C. La mesure du decalage, et ses six controles

### C.1 Les controles, ecrits avant la mesure

`C1` stabilite du vecteur sur toutes les images-cles retenues du film · `C2` partage du roster
en deux moities egales (oracle interne) · `C3` domaine `1..9` · `C4` plancher de bruit MESURE
(le nombre de decalages qui passent C1+C2+C3) · `C5` le decalage retenu est le meme d'un film a
l'autre · `C6` temoin negatif naturel : un film FFA ne doit pas produire de partage equilibre.

Regle de selection des paquets, elle aussi ecrite d'avance : on ne garde que les paquets dont
le NOMBRE d'entites ti=9 vaut le mode du film (`equipePaquetsModaux`). Un joueur qui arrive ou
part change ce nombre, et comparer un vecteur de 8 a un vecteur de 7 n'a pas de sens. Les
paquets ecartes sont comptes et publies.

### C.2 Le balayage interne : deux decalages sortent, et l'un est l'echo de l'autre

Corpus : 22 films (14 d'arene a 8 joueurs, 6 de Grande bataille a 24-30 participants, 2 de FFA).

| decalage | films passant C1+C2+C3 | ce qu'on y lit |
|---|---|---|
| **186** | **14 / 22** | des valeurs dans `{1, 2}`, 4-4 en arene, **12-12** en Grande bataille |
| 187 | 14 / 22 | `{3, 5}` — c'est **186 relu un bit plus loin** : `1 -> 3`, `2 -> 5`, le bit suivant valant 1 |
| 95 | 12 / 22 | `[1 9 1 9 1 9 1 9]` sur TOUS les films : une alternance structurelle, jamais un roster |
| 25 | 4 / 22 | `[2 2 2 2 3 3 3 3]` — bloc de quatre, insensible au roster reel |
| 23, 24 | 1 / 22 | idem, un seul film |

Les huit films qui ne passent pas C1+C2+C3 se repartissent en deux causes MESUREES, pas
supposees : les deux FFA (C6, section C.5) et six films ou la suite des slots des entites ti=9
change d'une image-cle a l'autre, ce qui fait echouer C1 pour une raison qui n'a rien a voir
avec l'equipe (section C.6).

### C.3 L'oracle externe : 16/18 films en accord EXACT

Trois producteurs, **aucune etape commune** :

- **l'ORDRE** vient du film : la table des slots de `chunk_00` lue par la grammaire de la
  phase 2, dont le rang EST le `player_index` de production (`filmIndex - rang` constant sur
  76/76 films) ;
- **l'EQUIPE** vient de la base : `match_participants.team_id`, passe a l'instrument comme un
  **ENSEMBLE** `xuid:equipe` — pas une suite. L'instrument n'en recoit aucun ordre : il apparie
  par XUID. Une permutation de la garde ne peut donc pas fabriquer d'accord ;
- **la VALEUR** vient de la trame : les 4 bits a `record + 186`.

| film | mode | entites ti=9 | accord | voisins |
|---|---|---|---|---|
| `000d5950` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `0014603f` | Tactical Slayer | 8 | **8/8** | 0/32 |
| `00502e52` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `00761d27` | Arena VIP | 8 | **8/8** | 0/32 |
| `0076ebdc` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `008e1bba` | CTF Arena | 8 | **8/8** | 0/32 |
| `010d1f7b` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `01db4132` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `01e1f945` | KOTH Arena | 8 | **8/8** | 0/32 |
| `02205ed3` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `022d2e0e` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| `02784ce1` | Slayer Arena | 8 | **8/8** | 0/32 |
| `02b172ab` | Team Slayer Arena | 8 | **8/8** | 0/32 |
| `02d39fa0` | Slayer Arena Super Fiesta | 8 | **8/8** | 0/32 |
| **`03af54c3`** | **BTB Slayer** | **24** | **24/24** | 0/32 |
| **`213a87dc`** | **BTB Heavies CTF** | **24** | **24/24** | 0/32 |
| `1950c59b` | **Arena FFA Slayer** | 8 | 0/8 (cf. C.5) | 0/32 |
| `610363ee` | **Shotty Snipe FFA** | 8 | 0/8 (cf. C.5) | 0/32 |
| **total** | | | **16/18 films, 160/176 slots** | **0/576** |

Les deux Grandes batailles repondent a l'objection d'echelle : un candidat qui « colle sur
8 joueurs » colle par hasard une fois sur 35 a deux equipes de 4 ; coller **24 fois sur 24**,
deux fois, sur une partition 12-12 dictee par une source externe, ne se produit pas par
hasard sur un champ de 4 bits.

Quatre films de Grande bataille de plus (`00ba2e1c` 26 joueurs, `036c102a` 25, `06a883f7` 30,
`1c4c63c2` 24) montrent bien un partage **12-12** a `d = 186`, mais ne sont pas comptes dans le
tableau : le nombre de rangs que la table de `chunk_00` apparie a l'oracle (22, 23, 23, 11) ne
vaut pas le nombre d'entites ti=9 (24), et un appariement a cardinaux differents ne peut pas
etre exact terme a terme. **C'est un defaut du lecteur de `chunk_00` sur les gros rosters, deja
consigne (phase 2, section G.3), pas un desaccord de l'equipe.** Ils sont comptes a part, et
publies comme tels par l'instrument.

### C.4 Le balayage COMPLET contre l'oracle : un seul decalage, sur tout le record

Le controle le plus severe du lot, parce qu'il n'a aucun a priori de position : pour chaque
film, on confronte l'oracle a **tous** les decalages du record.

| film | decalages rendant l'accord EXACT | sur |
|---|---|---|
| les 14 films d'arene | **1** — `[186]` | 457 |
| `03af54c3`, `213a87dc` (BTB) | **1** — `[186]` | 456 |
| `1950c59b`, `610363ee` (FFA) | **0** | 456 |

**16 films sur 16 rendent exactement un decalage, et c'est le meme.** Le plancher de bruit
n'est pas estime : il est de **0 faux positif sur 455 ou 456 decalages par film**, soit
**0 sur 7 292 positions** confrontees au total.

### C.5 C6 : le temoin negatif naturel, avec son temoin positif

Les deux films de FFA du cache lisent `[0 0 0 0 0 0 0 0]` a `d = 186` : **une seule valeur, et
c'est celle qui code « aucune equipe »** (brut 0 = designateur -1). Le partage equilibre
echoue, le domaine `1..9` echoue, et c'est le comportement attendu d'un mode sans equipes.

La base, elle, attribue a ces matchs `team_id` 0 a 7 — un camp synthetique par joueur. **Le
desaccord n'est donc pas une erreur de lecture : c'est une divergence de SENS entre le film
(aucune equipe) et l'API (une equipe par joueur).** Le temoin positif du meme instrument est
dans le meme journal : les 16 autres films rendent le partage exact.

### C.6 Un controle interne gratuit : ce qui bouge, c'est l'entite, pas l'equipe

Sur les six films ou le vecteur de designateurs change d'une image-cle a l'autre, la suite des
**slots** des entites ti=9 change aussi. Sur les seize ou elle ne change pas, **aucune** entite
ne change de designateur.

| film | entites dont le designateur change | suite de slots identique partout |
|---|---|---|
| `000d5950`, `0014603f`, `00502e52`, `00761d27`, `0076ebdc`, `010d1f7b`, `01db4132`, `02205ed3`, `022d2e0e`, `02784ce1`, `02b172ab`, `02d39fa0`, `1c4c63c2`, `213a87dc`, `1950c59b`, `610363ee` | **0** | oui |
| `008e1bba` | 6 / 8 | non |
| `01e1f945` | 4 / 8 | non |
| `00ba2e1c` | 13 / 24 | non |
| `036c102a` | 12 / 24 | non |
| `03af54c3` | 22 / 24 | non |
| `06a883f7` | 14 / 24 | non |

**22 films sur 22 : le designateur bouge si et seulement si la suite des slots bouge.** La
lecture la plus economique, et la seule que la mesure laisse debout : le designateur est stable
par ENTITE, et c'est l'appariement « i-eme entite ti=9 -> joueur » qui se deplace quand un slot
est reattribue. Consequence pratique pour un decodeur : lire le designateur par SLOT, pas par
rang, des que le roster n'est pas stable. `03af54c3` le montre a l'extreme : 22 des 24 entites
changent d'une image-cle a l'autre, et pourtant la PREMIERE image-cle retenue s'accorde a
**24/24** avec l'oracle.

---

## D. Le composant global `game-engine-team-mapping-component` — ce qu'il est, et ce qu'il n'est pas

La branche « l'equipe est dans le composant qui porte `team` dans son nom » a ete ouverte et
fermee, par le desassemblage.

`0x140f58200` (lecteur) lit, sur l'etat du composant :

| source | largeur | contenu |
|---|---|---|
| `+0x00` | 8 bits (`FUN_140f582d0`) | champ A |
| `+0x02` | 9 bits (`FUN_140f58324`) | champ B |
| `+0x04` | 9 bits | champ C |
| `+0x06` | 8 bits | **MASQUE**, teste ensuite bit a bit (`LEA R14,[RBX+0x6] ; TEST word ptr [R14],AX`) |
| `+0x08` | 8 bits | champ D |
| `+0x0a` | 8 bits | champ E |
| `+0x0c` .. `+0x13` | **8 octets signes** | pour `i` de 0 a 7 : si le bit `i` du masque est mis, `FUN_1407ef804` (4 bits, valeur-1) ; sinon `-1` |

**Huit entrees, pas trente-deux** : ce n'est pas une table par joueur. Le vidangeur de debug de
son masque de champs sales (`FUN_142f1b44c`) nomme les six champs dans l'ordre des bits :
`team-mapping` (bit 0), `shared-team-lives` (1), `current-state` (2), `game-finished` (3),
`current-round` (4), `round-timer` (5) — le vocabulaire d'un composant GLOBAL du moteur de jeu,
pas d'un joueur. L'ecrivain symetrique est `0x142f068bc`, meme ordre de champs, meme boucle de
huit gatee par `*(ushort*)(etat + 6)`.

Le depot porte deja ce lecteur (`filmdec/components_team_mapping.go`) avec les bonnes largeurs.
Rien a y changer.

---

## E. Les deux autres branches du cadrage, tranchees

**(b) Le chunk de type 8 « PLAYER_METADATA ».** `HANDOFF_FILM_RE_STATE.md` et
`GITHUB_RE_FINDINGS_EN.md` le decrivent comme « ~25 ko, roster ». **Mesure sur les
1 351 manifestes du cache** : les seuls types declares sont **1 (x1 351), 2 (x37 661) et
3 (x1 351)**. Aucun type 8, aucun type 12. **La branche est REFUTEE pour ce corpus** : il n'y a
pas de chunk de type 8 a lire dans ces films, et la production n'en lit aucun (`grep
PLAYER_METADATA` sur `internal/` ne rend qu'un manifeste de fixture, qui ne declare lui non plus
que 1/2/3).

**(c) Le pied (type 3), octet 55 du bloc de 60 octets.** `objectiveevents/film.go:182` le lit
sous le nom `teamRaw` avec le commentaire « NON fiable sur certains matchs -> a confirmer via
roster », et `objectiveevents/extract.go:63` tranche : « le champ team du film etant non fiable,
l'equipe d'un event vient TOUJOURS du roster via le xuid de l'acteur ».
**Statut : NON MESURE par ce lot, et la justification est ecrite.** Deux raisons. D'abord le
resultat de cette note le rend sans objet pour l'usage vise : la trame porte l'equipe par
joueur, exactement, sur tous les modes a equipes, et non seulement sur les matchs qui ont des
evenements d'objectif. Ensuite l'archive porte deja une contradiction datee qu'il faudrait
trancher AVANT de mesurer, et qui est hors du perimetre de ce lot :
`.ai/archive/V7/RESEARCH_THEATER_RE.md` ligne 534 identifie l'equipe a **`b37`/`b38`**
(« CONFIRME : le split 4/4 des events colle exactement au roster DB »), ligne 624 la dit fiable
en `b55`, et ligne 645 la dit **fausse** (« le champ team d'evdump est faux sur 24db »). La
production lit `b55`. **Decouverte hors perimetre, notee, non traitee** (regle 7) : voir
section G.

---

## F. Ce que la production fait aujourd'hui, et ce qu'elle perd

### F.1 Le chemin actuel

```
match_participants.team_id                       (la base)
  -> port.MatchFacts.Players[].TeamID
  -> replaybuild.equipesParXUID  (matchfacts.go:264)   map[xuid]equipe, -1 exclu
  -> replay.FlagInput.TeamOf     (build_objectives_live.go:110)
  -> l'invariant « on ne porte jamais son propre drapeau » du calque des drapeaux
  -> objectiveevents.Roster.TeamOf (extract.go:66)     l'equipe d'un evenement d'objectif
  -> sessionusage.TeamOf                               le contexte d'escouade
```

Et dans l'artefact de rejeu lui-meme :

- `replay/build.go:577` ecrit **`Team: -1`** en dur sur chaque piste ;
- `replay/document.go:577` documente ce -1 par « **L'EQUIPE N'EST PAS DANS LE FILM.** Elle vit
  dans la base, avec le gamertag, et le client la joint par XUID » ;
- `replay/document.go:365` repete la meme chose pour la zone de retour de drapeau : « l'equipe
  d'un joueur n'est PAS dans le film (cf. `Track.Team`) ».

**Ces trois affirmations sont refutees par cette note.** Ce n'est pas un bug : c'etait l'etat du
savoir. Elles se corrigeront dans le commit qui branchera la lecture, pas avant (aucune
correction n'est faite ici).

### F.2 Les pertes, telles que le code les compte lui-meme

1. **Le rejeu hors ligne n'a AUCUNE equipe.** Le constructeur d'artefact est hors ligne par
   construction et n'ouvre aucune base (`document_objectives_live.go:140`). Sans lignes de
   match, `equipesParXUID` rend `nil`, `TeamOf` est vide, et l'invariant du drapeau « se tait ».
   Le compteur qui mesure exactement cette perte existe : **`CarrierTeamUnknown`** — « compte
   les portages dont l'EQUIPE du porteur ne nomme aucun drapeau : table `TeamOf` vide (CLI hors
   ligne, ouvrier sans faits) ou carte a plus de deux socles »
   (`document_objectives_live.go:318`).
2. **Sans equipe, un portage de drapeau peut etre pose sur le mauvais drapeau.** La revue
   DRAPEAUX-R1 (constat C1, 2026-09-07) le documente sur piece : « son repli sur le socle le
   plus proche a pose une prise, puis la CAPTURE qu'elle porte, sur le drapeau du camp de son
   auteur (`64e8adfa`, prise a 527 555 ms a 2,4 m de son propre socle, drapeau adverse au sol a
   11,2 m) ».
3. **Les evenements d'objectif d'un match absent de la base sortent sans equipe.**
   `extract.go:184` n'ecrit `ev.TeamID` que si `roster.TeamOf(xuid)` repond ; sinon le champ
   reste nul.
4. **L'artefact publie `team: -1` pour toutes les pistes**, donc tout consommateur qui ne
   rejoint pas la base par XUID voit un rejeu sans camps. C'est la source des « sans equipe »
   que l'utilisateur constate sur un film dont le match n'est pas (ou plus) en base.
5. **Les modes a manches** heritent du meme trou : `build_objectives_live.go:71` prevoit
   explicitement le repli « tous dans UN drapeau d'equipe `[TeamNeutral]` » quand les socles
   manquent.

Ce que le resultat de cette note rend possible — et qui n'est PAS fait ici : le film est
auto-suffisant pour l'equipe. Un artefact hors ligne pourrait porter l'equipe reelle sans
ouvrir de base, et l'invariant du drapeau pourrait tenir sur un film seul.

---

## G. Ce qui est ETABLI, ce qui est HYPOTHESE, ce qui est REFUTE

### Etabli (deux chaines independantes)

1. L'equipe d'un joueur est dans la trame d'etat (type 2), composant
   `managed-player-team-designator-component`, archetype ti=9 composant i0, **4 bits**.
   Ecrivain `0x142edbd3c`, lecteur `0x140f581e8`, primitive `FUN_1407ef804`, descripteur
   `0x143d08ad0`.
2. La valeur ecrite vaut **`designateur + 1`** ; `0` = aucune equipe. Prediction
   `brut = team_id + 1` verifiee **160/176 slots, 16/18 films**, dont **24/24 deux fois** en
   Grande bataille.
3. Le champ est a **186 bits du debut du record d'image-cle**, et c'est le **seul** decalage du
   record qui satisfasse l'oracle : 1 sur 456/457 par film, 0 faux positif sur 7 292 positions,
   0 touche sur 576 decalages voisins.
4. Les entites ti=9 d'une image-cle sont **exactement les joueurs** : 8 en arene, **24 en
   Grande bataille**, slots consecutifs de pas 2, longueur de record constante (459/460 bits
   selon le build).
5. Leur ordre s'accorde a l'ordre de la table des slots de `chunk_00`, dont la phase 2 a etabli
   qu'il est le `player_index` de production.
6. Le designateur est stable par entite : il change **si et seulement si** la suite des slots
   change (22/22 films).
7. En FFA, le film ne porte **aucune** equipe : `0` sur les huit entites, sur les deux films FFA
   du cache.
8. `game-engine-team-mapping-component` est une table **par equipe** (8 entrees, masque a
   `+0x06`), pas par joueur. Six champs nommes par `FUN_142f1b44c`.
9. Le cache ne contient que des chunks de type 1, 2 et 3 (1 351 manifestes) : pas de type 8
   « PLAYER_METADATA » a lire.
10. L'enumeration `mp_team_designator` compte **9 valeurs** : `First` .. `Eighth`, `Neutral`.

### Hypothese (une seule chaine, ou non tranche)

1. **Pourquoi 186.** Le decalage est MESURE, pas explique : il vaut l'en-tete par entite plus le
   bloc d'etat par defaut de ti=9, et aucune de ces deux largeurs n'est tranchee par le dossier
   (64 / 108 / 47 selon la source, cf. `keyframe_fullstate_loop.go`). `186 - 108 = 78` et
   `186 - 64 = 122` ne tombent sur rien de connu. **A ne pas ecrire comme une grammaire** : un
   decodeur qui s'appuierait sur 186 sans calibrer devra le re-mesurer par build.
2. **La correspondance designateur -> nom de camp.** `brut - 1 = team_id` est mesure ; que
   `team_id 0` soit `First` (et non un autre nom de l'enumeration) est l'ordre de la table lue,
   donc plausible, pas prouve — aucune mesure ne relie un nom affiche a un rang ici.
3. **Les builds anterieurs.** Le lot a mesure 459 et 460 bits de longueur de record ; `d = 186`
   tient sur les deux. Les builds `HI_1_8_0` a `HI_1_11_0` n'ont pas ete essayes.

### Refute

1. **« L'equipe n'est pas dans le film. »** (`replay/document.go:577`, `document.go:365`,
   `build.go:577`) REFUTE : elle y est, par joueur, sur 4 bits.
2. **« L'equipe est dans l'enregistrement de slot de `chunk_00`. »** REFUTE en phase 2 par la
   negative (huit champs courts constants), et maintenant par le positif : elle est ailleurs, et
   on sait ou.
3. **« Le roster et les equipes sont dans un chunk de type 8 PLAYER_METADATA. »** REFUTE pour ce
   corpus : aucun chunk de type 8 dans les 1 351 manifestes.
4. **« `game-engine-team-mapping-component` porte l'equipe des joueurs. »** REFUTE : 8 entrees,
   composant global, six champs nommes qui sont tous des etats de partie.
5. **« Les 8 entites ti=9 sont huit equipes sans identite. »** (fork chasewoodhams, constat n°3)
   REFUTE : ce sont les huit JOUEURS, dans l'ordre du `player_index`, et l'identite se lit par le
   rang ou par le slot.

---

## H. Commandes pour rejouer

Les instruments sont sous garde d'environnement ; sans la variable ils sont sautes, donc la CI
reste verte. Les chemins sont **en style Windows** (`C:/...`) : un chemin de style Git Bash
(`/c/...`) fait echouer l'ouverture.

```bash
cd apps/go-api
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"

# Le corpus du lot : 14 films d'arene, 6 de Grande bataille, 2 de FFA.
F="000d5950 0014603f 00502e52 00761d27 0076ebdc 008e1bba 010d1f7b 01db4132 01e1f945 02205ed3"
F="$F 022d2e0e 02784ce1 02b172ab 02d39fa0 00ba2e1c 036c102a 03af54c3 06a883f7 1c4c63c2 213a87dc"
F="$F 1950c59b 610363ee"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"

# B : le recensement (archetype relu dans le film, entites ti=9 par image-cle, emprises)
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run 'TestEquipeFilmArchetype|TestEquipeFilmRecon|TestEquipeFilmCensusTousPaquets' -v -timeout 30m

# C.2 : le balayage interne (C1+C2+C3), le plancher C4 et la convergence C5
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run TestEquipeFilmDecalage -v -timeout 30m

# C.5 / C.6 : le releve brut au decalage mesure — temoin negatif FFA et controle de slots
CHUNK00_FILMS="$L" go test ./internal/analysis/filmdec/ \
  -run TestEquipeFilmBrutAuDecalage -v -timeout 30m

# C.3 / C.4 : l'oracle externe et le balayage complet (voir la garde ci-dessous)
CHUNK00_FILMS="$L" CHUNK00_XUID_EQUIPES="$ORACLE" go test ./internal/analysis/filmdec/ \
  -run 'TestEquipeFilmOracleXuid|TestEquipeFilmOracleBalayage' -v -timeout 60m
```

`CHUNK00_XUID_EQUIPES` est de la forme `<prefixe>=<xuid>:<equipe>,<xuid>:<equipe>,...`, blocs
separes par `;`. **L'ordre n'y est pas lu** : l'instrument apparie par XUID. Il se construit en
**lecture seule** depuis la shared (le fichier de production peut etre tenu par le serveur —
dans ce cas ne pas forcer et lire une sauvegarde, par exemple
`data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb`) :

```sql
SELECT substr(p.match_id,1,8) AS pref,
       count(*) AS n,
       count(DISTINCT p.team_id) AS nt,
       COALESCE(r.playlist_name,'') AS pl,
       COALESCE(r.game_variant_name,'') AS gv,
       string_agg(p.xuid || ':' || COALESCE(CAST(p.team_id AS VARCHAR),'NULL'), ',') AS roster
  FROM match_participants p
  LEFT JOIN match_registry r ON r.match_id = p.match_id
 GROUP BY 1, 4, 5;
```

`nt >= 8` designe les films de FFA, `n >= 24` ceux de Grande bataille. Le recensement des types
de chunk (section E) se rejoue sans Go :

```bash
cd data/cache/film_manifests
for f in *.json; do tr ',' '\n' < "$f" | grep -o '"chunk_type":[0-9]*'; done \
  | sed 's/.*://' | sort -n | uniq -c
```

Gates passes : `gofmt -l` net, `go vet ./internal/analysis/filmdec/` net,
`go test ./internal/analysis/filmdec/` sans garde **ok**, `go test ./internal/archlint/` **ok**
— le ratchet des variables de paquet de `filmdec` n'est pas touche (les instruments
n'introduisent que des `const`, des types et des locales).

---

## I. Decouvertes hors perimetre — notees, NON TRAITEES (regle 7)

1. **La production lit `b55` la ou l'archive a confirme `b37`/`b38`.** L'equipe d'un evenement
   du pied de film (type 3) : `objectiveevents/film.go:251` lit l'octet 55 du bloc de 60 et le
   nomme `teamRaw` « NON fiable » ; `.ai/archive/V7/RESEARCH_THEATER_RE.md:534` identifie
   l'equipe a `b37`/`b38` avec une validation 4/4 contre la base, et sa ligne 645 dit le champ
   d'`evdump` faux sur un film. Trois affirmations, deux offsets, aucune tranchee. **Non
   traitee** : le champ n'est plus le chemin de l'equipe apres cette note, et le trancher demande
   une mesure sur le pied, hors perimetre du lot.
2. **Le decalage de 8 octets de `parseRegistry`** : connu depuis la phase 1, non tranche, pas
   touche.
3. **Le marcheur de la table d'image-cle ne trouve aucune entite ti=9 dans le PREMIER paquet de
   type 2** (`chunk_01`), sur les deux films verifies. Les `managed-player` n'apparaissent qu'a
   partir de `chunk_02`. Cause non etablie : arrivee tardive des entites, ou un record que le
   filtre fort du marcheur ecarte. **Non traitee** : le lot n'en a pas besoin, tous les autres
   paquets suffisent.
4. **Le lecteur de la table des slots de `chunk_00` casse sur les gros rosters.** Consigne en
   phase 2 (G.3), re-constate ici : 11 a 23 enregistrements rendus la ou le match en compte 24 a
   30. L'instrument de ce lot contourne par un repli documente (balayage brut filtre, trie par
   position de bit) ; **la cause n'est pas traitee**.
5. **Six films sur 22 voient la suite des slots des entites ti=9 changer d'une image-cle a
   l'autre.** C'est la reattribution de slot, mesuree ici comme correlee a 22/22 aux changements
   de designateur. Un decodeur devra indexer par SLOT. **Non traite** : travail de decodeur.

---

## Ce qui reste ouvert

1. **Expliquer 186** au lieu de le mesurer : il faut la largeur de l'en-tete par entite ET celle
   du bloc d'etat par defaut de ti=9. C'est la meme question que le plan R7-e
   (`keyframe_fullstate_loop.go`) et que la valeur 47 du fork chasewoodhams. Tant qu'elle n'est
   pas tranchee, un decodeur doit **calibrer** le decalage sur le film (par exemple par le
   controle C2 sur une image-cle) plutot que le cabler.
2. **Les builds anciens** (`HI_1_8_0` a `HI_1_11_0`, 76 films du cache) : `d = 186` n'y a pas ete
   essaye. Le profil par build de la phase 2 (section D) montre que les largeurs FIXES bougent
   d'une constante par build ; il faut s'attendre a la meme chose ici.
3. **Le nom d'un designateur** : `brut - 1 = team_id` est mesure ; relier `team_id 0` a `First`
   (ou a `Eagle` cote interface) demande une mesure de plus, cote affichage.
4. **Brancher la lecture en production** : le pont `rang -> xuid` de `chunk_00` (phase 2) plus le
   designateur par slot (cette note) donnent un artefact de rejeu qui porte l'equipe sans ouvrir
   de base. Il faut d'abord rendre le lecteur de `chunk_00` robuste aux gros rosters (ouvert n°4
   de la phase 2) et indexer le designateur par slot. **Aucun code de production n'a ete touche
   par ce lot.**
5. **Le pied (type 3), `b37`/`b38` contre `b55`** : voir section I.1.
