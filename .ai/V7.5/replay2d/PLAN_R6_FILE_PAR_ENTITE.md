# Plan R6 — La FILE PAR ENTITE et le vrai consommateur du payload type-2

> Ecrit le 2026-08-17. Lot R6, ouvert par la condition de reprise ECRITE du lot R5
> (`PLAN_R5_GRAMMAIRE_IMAGE_CLE.md` §11, ligne « la grammaire du CORPS d'un record
> d'IMAGE-CLE ») : « decompiler le CONSOMMATEUR du payload type-2 ; le chemin d'image-cle
> passe par `FUN_142f2913c` (baseline-emit) qui draine une file par-entite alimentee par
> `FUN_142f29538` — donc un bitstream RECONSTRUIT, pas le payload du film. La
> transformation "payload type-2 -> file par-entite" n'est identifiee nulle part. »
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfqueue`, branche `wt/kf-file-entite`,
> base `wt/kf-grammaire` = `ffc465336` (= R3 + R5).
> Execution sous le contrat du skill `plan-execution`. Grille `plan-review` passee au §6bis.

---

## 1. Objectif et critere de succes

**Objectif.** Etablir, PAR LECTURE DU CODE DU JEU puis par mesure, ce que la file par-entite
fait reellement du flux (transformation ou report), reconcilier l'oracle
`keyframe_buffer_live.bin` avec un objet NOMME du film, et en tirer — si la lecture le
permet — un point d'entree VERIFIABLE vers les trois cibles reportees : `ti=37` (corps
complet), `ti=42` (position des armes au sol), `ti=11` (objectif vivant).

**Ce que ce lot ne cherche PAS** (perimetre FERME) : aucun rendu, aucun son, aucune string
UI ; aucune bosse de `SchemaVersion` ; aucune re-cuisson d'artefact ; aucune ecriture
DuckDB ; aucun nommage d'`eqip` ; aucune correction du decalage `originMs` ; aucun run de
masse.

### 1.1 Criteres mesurables, avec leurs denominateurs

| C | controle | seuil | temoin independant |
|---|---|---|---|
| **C0** | **RECONCILIATION** — `keyframe_buffer_live.bin` (11 485 o, deterministe entre deux lancements) est identifie comme un objet NOMME du film : type de paquet, rang, film si le corpus le contient | identification exacte OU negatif publie avec la liste des candidats ecartes et leur cause | la chaine de decoupe de paquets `filmdec.WalkPackets` (`film_packets.go:70`), independante de toute lecture de bits |
| **C1** | **MARCHE BIT-EXACTE SUR L'OBJET RECONCILIE** — la boucle de records PORTEE (`DecodeFrameRecords`) traverse le paquet identifie en C0 sans desync, sur les 3 films de l'oracle R3 | >= 95 % des bits du paquet consommes sans desync, denominateur = taille du paquet en bits, PAR FILM | la borne de fin de paquet (`FilmPacket.Size`) est donnee par l'en-tete de 16 octets, chaine disjointe du bitstream |
| **C2** | **COUVERTURE D'ARCHETYPES** — la liste des `typeIndex` lies par des records NEW PROPRES du paquet reconcilie contient `ti=37`, `ti=42`, `ti=11` | presence mesuree et publiee archetype par archetype ; un archetype absent est un negatif publie, pas un echec du lot | `WalkKeyframeWorld` (payload type-2, 249/250 entites contre un oracle Cheat Engine) sert de contre-liste : les memes slots doivent apparaitre |
| **C3** | **`ti=42` POSITIONS** — positions d'armes au sol lues sur les records NEW propres du paquet reconcilie, dans l'emprise de la carte, et battant une bande FANTOME de meme cardinalite passee par le MEME code | dans l'emprise >= 95 % ; reel/fantome >= 3x | emprise = nuage de positions biped du meme film (autre chaine) ; fantome = patron du lot arme-au-sol du 2026-08-12 |
| **C4** | **`ti=11` PROGRESSION** — `i5` / `i12` / `i13` / `i14` lus sur les records NEW propres, valeurs stables par slot et coherentes avec les evenements de capture deja decodes | i12 <= i13 sur >= 95 % des lectures ; i5 stable par slot sur >= 95 % | `objectiveevents.IdentifiedEvent` (statborg), chaine TOTALEMENT disjointe |

**Un negatif MESURE est un livrable.** Toute phase qui echoue publie ses denominateurs, son
temoin de controle et sa cause nommee, et statue `[!]` avec justification ecrite.
**Regle de non-publication** : aucune donnee n'entre dans l'artefact tant que le controle de
sa phase n'est pas passe.

---

## 2. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17 (reconnaissance de pre-plan)

### 2.1 Ce que Ghidra montre du couple producteur / consommateur — LU AVANT D'ECRIRE CE PLAN

Instance PID 10104, `HaloInfinite.exe`. **Note d'acces** : le pont `mcp__ghidra__*` refuse de
se connecter (l'instance UDS se declare `unknown`, `connect_instance` refuse tout repli TCP).
L'API HTTP du plugin (`127.0.0.1:8089`) repond et a ete utilisee en LECTURE SEULE
(`/decompile_function`, `/get_xrefs_to`, `/get_xrefs_from`) — memes endpoints, meme
programme, aucun rename, aucun script, aucune analyse relancee.

| adresse | ce que la decompile montre |
|---|---|
| `FUN_142f29538` | **push d'un tampon circulaire**. Element de 0x38 = 56 octets (`param_1 + idx*0xe + 4`, ints). Recopie `param_2[0..3]`, un handle refcompte (`FUN_142a777e8`), puis `param_2[8..0xd]`. **Aucune ecriture de bits, aucun reencodage.** |
| `FUN_142f2913c` | **drain**. Pour chaque item : `FUN_14064c350` (init reader), **`FUN_1411b149c(reader, *(item+0x10), *(item+0x20))`** = pose (data, len), `FUN_1406d5cc0(reader,3)`, **`FUN_1432fe23c(reader, *(item+0x2c))`**, puis `FUN_1406cbaa0(*(item+0x30) type, *(item+0x28) id, ..., reader, 0)` |
| `FUN_1432fe23c` | `*(reader+0x2c) = param_2 ; *(reader+0x28) = param_2 ; *(reader+0x20) = 2`. **`+0x2c` est la POSITION EN BITS du reader** (meme champ que la capture live de juillet). Donc `item+0x2c` = **le bit de depart du record** |
| `FUN_1411b149c` | `*(r+8)=data ; *(r+0x10)=data+len ; *(r+0x18)=len ; FUN_1406d5cc0(r,0)`. `item+0x20` = **longueur du tampon**, pas du record |
| `FUN_142f25334` | **`memcpy` du tampon ENTIER du paquet** (`src = *(reader+8)`, `n = *(reader+0x18)`) dans un bloc alloue refcompte. Chemin source revele par l'allocateur : `...\engine\source\blofeld\networking\replication\replication_entity_manager_view.cpp` |
| `FUN_1406cd128` @`0x1422f44fb` | **UNIQUE xref CODE du push** (l'autre xref est une DATA). Il est dans la branche `DAT_14474cd78 != 0` de la boucle de records : `uVar23 = *(param_3+0x2c)` (**bit courant du reader, capture AVANT decodage**), `FUN_1406cbaa0(..., 1)`, puis construction de l'item — `local_110/uStack_108 = *param_2` (0x00), le handle du memcpy (0x10), `*(param_3+0x18)` = longueur (0x20), horodatage (0x24), **id (0x28)**, **bit de depart (0x2c)**, **type (0x30)**, index de vue (0x34) — et `FUN_142f29538(queue, &local_110)` |
| `FUN_1406cd128` tete | `cVar3 = *(param_1+0x12)` ; `if (cVar3 == 0) { <toute la boucle> }` — quand la porte d'image-cle est mise, **la fonction ne fait rien du tout** et rend 2. Confirme R5 |
| `FUN_142987460` | traite UN paquet : 3 vues, chacune `vtable[0x60]` (= `FUN_142f2913c`, drain) puis `vtable[0x40]` (= `FUN_1406cd128`, lit le paquet), puis applique tous les records par `vtable[0x48]`. **Aucun traitement de type-2 ici** |

**LE RESULTAT DE LECTURE, ET IL REFUTE L'HYPOTHESE N°1 DU BRIEF AVANT TOUTE MESURE :**
**la file par-entite N'EST PAS UNE TRANSFORMATION.** Un item ne contient aucun bitstream
reconstruit : il contient (a) un handle vers une COPIE OCTET POUR OCTET du tampon du paquet,
(b) la longueur de ce tampon, (c) l'id, (d) **la position en bits du debut du record**,
(e) le type. Le drain repose un reader sur la meme copie, se replace au meme bit, et appelle
le MEME `FUN_1406cbaa0` avec la MEME grammaire. La file **DIFFERE** des records ; elle ne les
reecrit pas. Il n'existe donc aucune transformation « payload type-2 -> file par-entite » a
porter : l'objet cherche par la condition de reprise de R5 n'existe pas.

### 2.2 Ce que la mesure offline montre de l'oracle `keyframe_buffer_live.bin`

Mesures faites avant d'ecrire ce plan, avec deux outils de reconnaissance jetables (scan
d'octets, decoupe de paquets) — **a REECRIRE en instrument versionne en phase 2**, la
mesure ci-dessous n'est retenue que comme etat des lieux.

| fait mesure | denominateur / piece |
|---|---|
| Les 16 premiers octets de `keyframe_buffer_live.bin` et de `kf_slot0_live.bin` sont IDENTIQUES : `88 00 15 84 00 2c 54 0c 61 c9 00 0b ff ff ff fc` — puis ils divergent | 2 dumps, `od -t x1` |
| Ces 16 octets ne sont PAS ceux du payload type-2 (`a0 00 00 00 00 00 00 0b 7f ff ff ff 98 ...`) | idem, et `HANDOFF_KEYFRAME_LIVE_CAPTURE.md` le disait deja |
| Ces 16 octets sont **la tete du PREMIER paquet de type 0** de chaque film — mesure sur les 6 films du corpus | decoupe de paquets : `00502e52/chunk_01` paquet #8 type=0 taille=11 312 ; `07aa428d/chunk_01` #8 type=0 taille=11 066 ; `000d5950/chunk_01` #8 type=0 taille=9 297 |
| Le prefixe de 32 octets du dump se retrouve dans **37 des 951 films** du cache, toujours au meme endroit structurel | scan d'octets sur `data/cache/film_chunks/` complet |
| Le dump ne coincide avec AUCUNE region des 6 films du corpus au-dela de 16 octets | `cmp` aux 5 positions candidates : divergence au 17e octet dans les 5 cas |
| Structure d'entree de session, identique sur les 3 films oracles : paquet #0 type=1 (343 019 o, table de precision), **#1 type=2 (138 340 / 140 837 / 142 695 o)**, #2 type=6, #3 type=8, ... , **#8 type=0 (9 297 / 11 312 / 11 066 o)** | decoupe de paquets, 3 films |

**CE QUE CELA ETABLIT** : `keyframe_buffer_live.bin` **n'est pas une sortie de
transformation du payload type-2** ; c'est la **COPIE VERBATIM D'UN PAQUET DE TYPE 0** — ce
que la lecture de `FUN_142f25334` (memcpy du tampon entier) predisait exactement. La capture
live de juillet, etiquetee « keyframe » depuis, a ete prise sur **le premier paquet delta**,
pas sur l'image-cle. Sa pile d'appel le disait deja (`FUN_1406cbaa0` <- `FUN_1406cd128`) :
`FUN_1406cd128` ne s'execute JAMAIS en mode image-cle.

**CONSEQUENCE, ET C'EST LE PIVOT DU LOT** : le premier paquet type-0 d'une session est un
paquet de records NEW (la capture en montre 266 a slots croissants, puis 134 DELTA), il se
lit par la grammaire DEJA PORTEE (`DecodeFrameRecords` / `TraverseEntity`), et c'est LUI que
le jeu utilise comme etat initial. Le payload type-2 n'est traverse par aucun des trois
lecteurs de record connus. **Les trois cibles reportees (`ti=37`, `ti=42`, `ti=11`) doivent
donc etre cherchees dans le premier paquet type-0, pas dans le payload type-2.**

### 2.3 Ce que le depot sait deja, relu ce jour

| fait | piece |
|---|---|
| Decoupe de paquets et types : `PacketTypeDelta = 0`, `PacketTypeKeyframe = 2` | `filmdec/film_packets.go:19-21`, `WalkPackets` `:70` |
| Boucle de records type-0 portee, NEW / DEL / DELTA, liaison du World sur traversee PROPRE uniquement | `filmdec/frame_records.go:710-780` (`DecodeFrameRecords`) |
| Record NEW porte : `R(6) ti` -> etat par defaut -> `R(1)` porte -> masque -> composants -> tail | `filmdec/traverse.go:1008` (`TraverseEntity`) |
| Etat par defaut par archetype : 21 archetypes portes, `ti=42` inscrit par R5 | `filmdec/default_state_arch.go:44-64`, `filmdec/default_state_ti42.go` |
| Walker du payload type-2 (BALAYAGE, pas parse) et son filtre fort `field26 == 0` | `filmdec/keyframe_world.go:70`, `:153-207` |
| Le corps d'un record du payload type-2 n'est PAS un record NEW (128 decalages x 16 lectures x 3 films, jamais > 1,8 %) | `PLAN_R5_GRAMMAIRE_IMAGE_CLE.md` phase 2 |
| `SchemaVersion = 8` | `replay/document.go:75` |

---

## 3. Hypotheses, ORDONNEES par cout croissant

**H1 — LE PREMIER PAQUET TYPE-0 EST L'ETAT INITIAL, ET IL SE LIT AVEC LA GRAMMAIRE DEJA
PORTEE (cout : TRES FAIBLE).** C'est la lecture directe du §2.2. Test : lancer
`DecodeFrameRecords` sur le paquet #8 type-0 des 3 films oracles et mesurer (a) le nombre de
records, (b) la fraction du paquet consommee sans desync, (c) la liste des `typeIndex` lies.
Si H1 tient, `ti=37` / `ti=42` / `ti=11` ont un point d'entree VERIFIABLE et le verrou
« grammaire du corps d'image-cle » devient **hors du chemin critique**.

**H2 — LE PAYLOAD TYPE-2 N'EST PAS UN BITSTREAM DE RECORDS (cout : FAIBLE).** La file ne le
transforme pas (§2.1), les trois lecteurs de record ne le lisent pas (R5), et son en-tete de
64 bits `[id:32][field:26][ti:6]` se lit aussi bien comme **deux mots de 32 bits**
(`[id:u32][typeIndex:u32]`) — ce que confirme le filtre fort du balayeur, qui exige que le
mot a `+32` vaille `< 50`. Test, gratuit : mesurer la distribution des mots de 32 bits a
`+32` sur tous les ancres du payload, et la distribution des ecarts entre ancres modulo 8.
Un ecart TOUJOURS multiple de 8 dirait « table d'octets », pas « bitstream ».

**H3 — LE PAYLOAD TYPE-2 EST CONSOMME PAR UN AUTRE CHEMIN QUE LE DECODEUR DE REPLICATION
(cout : MOYEN).** `FUN_142987460` traite un paquet et ne connait que 3 vues x (drain +
delta-loop). Aucun de ses appels ne prend un type de paquet. Le type-2 doit donc etre
dispatche en amont, par le lecteur de film. Test : remonter les appelants de
`FUN_14298816c` jusqu'au demultiplexeur de paquets et lire son aiguillage sur le type.

**H4 — SI H1 TOMBE : LE PREMIER PAQUET TYPE-0 EXIGE UN ETAT INITIAL VENU DU TYPE-2
(cout : ELEVE).** Le record NEW du premier paquet lie l'archetype ; mais si sa traversee
desynchronise tot, c'est que le paquet suppose un World deja peuple. Test : rejouer avec le
World prealablement peuple par `WorldFromKeyframe` (payload type-2), et comparer les deux
denominateurs.

---

## 4. Corpus — FERME

Racine, LECTURE SEULE : `C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/`.

| film | role |
|---|---|
| `000d5950` | principal — golden du rejeu, oracle R3 (415 records `ti=37` bornes au type-2) |
| `00502e52` | principal — oracle R3 (408 bornes) |
| `07aa428d` | principal — oracle R3 (403 bornes) |
| `64e8adfa` | CTF Catalyst — cible `ti=11` (201 records, 10 slots, mesure R4) |
| `530820e5` | CTF Catalyst — second temoin `ti=11` (115 records, 5 slots) |
| `0014603f` | TEMOIN NEGATIF (`ti=11` absent) |

Le cache complet (951 films) n'est lu QUE pour la reconciliation C0 (recherche du film de la
capture live), en lecture seule, et une seule fois.

## 4bis. Blockers connus et leur contournement

| blocker | nature | contournement |
|---|---|---|
| Le pont `mcp__ghidra__*` refuse de se connecter (instance UDS `unknown`) | outillage | API HTTP du plugin en LECTURE SEULE (`127.0.0.1:8089`, memes endpoints). Consigne dans ce plan et dans `WALK_PORT_NOTES.md` |
| Les films vivent dans le depot PRINCIPAL, absent du worktree | chemin | lecture seule par chemin absolu, aucune ecriture |
| Un seul decodage `filmdec` par process (hooks + bascules GLOBALES) | technique | un seul test a la fois, bascules restaurees en `defer` |
| Cache Go partage corrompu par deux `go` concurrents | technique | `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfqueue/.gocache` (deja dans l'`info/exclude` partage) + UNE commande `go` a la fois |
| CGO sur le chemin de build | technique | `CGO_ENABLED=0` sur `filmdec`/`replay` |
| Ghidra est le programme de l'utilisateur | ressource partagee | LECTURE SEULE stricte, aucun rename, aucun script |
| Le film de la capture live peut ne pas etre dans le cache | donnee | C0 accepte un negatif : la reconciliation STRUCTURELLE (type de paquet + rang) suffit a la conclusion, l'identification du film n'est qu'un bonus |

---

## 5. Phases

Une phase n'ouvre pas tant que la precedente n'est pas CLOSE : gate passe (commandes exactes,
sorties collees au journal §10), tous les items statues, plan mis a jour, commit sur
`wt/kf-file-entite`, `git push -u origin wt/kf-file-entite`, point d'etape.

| phase | effort | livrable seule ? |
|---|---|---|
| 1 LIRE le producteur et le consommateur dans Ghidra | moyen | oui (le journal RE est un livrable) |
| 2 PORTER et PROUVER : reconciliation + marche sur le paquet reconcilie | moyen | oui (un negatif mesure est un livrable) |
| 3 `ti=37` corps complet | moyen | oui |
| 4 `ti=42` positions | moyen | oui |
| 5 `ti=11` progression | moyen | oui |
| 6 PUBLIER / noter la bosse | rapide | non |
| 7 CLORE | rapide | non |

### Phase 1 — LIRE : la file par-entite, dans le code du jeu — CLOSE le 2026-08-17

- [x] 1.1 `FUN_142f29538` (producteur) decompile, structure d'item ecrite champ par champ,
      avec la taille d'element mesuree sur la decompile.
- [x] 1.2 `FUN_142f2913c` (consommateur) decompile, sequence de reconstruction du reader
      ecrite appel par appel, et le role de chaque champ d'item NOMME (data / len / bit de
      depart / id / type / vue).
- [x] 1.3 **Reponse ecrite a la question du brief** : y a-t-il une transformation
      « payload type-2 -> file par-entite » ? Reponse justifiee par les adresses.
- [x] 1.4 Les xrefs du push remontees : QUI alimente la file, et sous quelle garde.
      Verifier sur pieces que la porte d'image-cle desactive bien ce chemin.
- [x] 1.5 Chercher le demultiplexeur de paquets du lecteur de film (H3) : remonter les
      appelants de `FUN_14298816c` jusqu'a l'aiguillage sur le type de paquet, et ECRIRE ce
      qu'on trouve — y compris « pas trouve dans le budget de cette phase », avec les
      adresses parcourues.
- [x] 1.6 Sous-section « IMAGE-CLE — file par entite » ecrite dans
      `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` (>= 10 adresses distinctes citees).
      Aucune adresse en dur cote Go.
- [x] 1.7 **AJOUT du superviseur (2026-08-17, apres la cloture)** : puisque le JEU ne relit
      jamais le payload type-2, la question devient « qui l'ECRIT ». Recherche BORNEE a
      trois sondes chaines/xrefs — toutes negatives, consignees au journal RE §5 : une
      seule chaine `saved_games` dans tout le binaire (le site d'allocation du LECTEUR,
      `0x143ddb470`, xref unique) ; une seule chaine `FilmBlock*` (`FilmBlockReadError`,
      cote lecture) ; l'encodeur de snapshot `FUN_142f2e174` n'a aucun appel direct (case
      de vtable `0x1436a87f0` seulement). **Le chemin d'ECRITURE des chunks n'est pas dans
      ce binaire par la voie chaines/xrefs.** Fil borne et ferme, comme demande.

**Gate 1** (a executer et coller au journal) :

```
grep -ci "file par entite" .ai/V7.5/killweapon/WALK_PORT_NOTES.md                       -> >= 1
grep -o "FUN_1[0-9a-f]*\|0x14[0-9a-f]*" .ai/V7.5/killweapon/WALK_PORT_NOTES.md | sort -u | wc -l  -> > 62
grep -c "FUN_142f29538\|FUN_142f2913c\|FUN_1432fe23c\|FUN_142f25334" .ai/V7.5/killweapon/WALK_PORT_NOTES.md -> >= 4
```

Critere : les 4 adresses du couple producteur/consommateur citees, la question 1.3 tranchee
par ecrit, le compte d'adresses distinctes en hausse.

### Phase 2 — PORTER et PROUVER : reconciliation (C0) puis marche (C1, C2) — CLOSE le 2026-08-17, **C1 NEGATIF MESURE**

- [x] 2.1 `filmdec/keyframe_entity_queue.go` ECRIT (fichier NEUF, a moi) :
      `FindPacketByPayload` / `FindPacketByPrefix` (reconciliation par `WalkPackets`),
      `FirstPacketOfType` / `AllPacketsOfType`, `KFQVariant` + `KFQWalk` +
      `WalkPacketRecords(WithWorld)` + `BestVariant` (marche mesuree), et
      `MeasureKeyframeAnchors` (forme des ancres type-2). Il REUTILISE
      `DecodeFrameRecords`, `WalkKeyframeWorld`, `WorldFromKeyframe`, `WalkPackets`,
      `ReadFilmChunk` — **aucun deserialiseur recopie, aucune adresse de binaire cablee**.
      La structure d'item de la file est documentee dans le journal RE, PAS en Go : une
      struct sans lecteur serait du code mort (anti-patron n°1).
- [x] 2.2 **C0 mesure — negatif sur l'identification, POSITIF sur la structure.**
      Coincidence EXACTE de `keyframe_buffer_live.bin` (11 485 o) sur les **951 films** du
      cache : **0 paquet** — le film de la capture live de juillet n'est pas cache. La
      reconciliation STRUCTURELLE, elle, est nette : coincidence de PREFIXE (16 octets) =
      **949 paquets sur 951 films, `par type map[0:949]`, `par rang map[8:949]`**. Autrement
      dit : le tampon capture est le PREMIER PAQUET DELTA d'une session, et cette forme est
      universelle. Sortie exacte au journal §10.
      **Le SECOND dump mesure aussi**, et il dit la meme chose : `kf_slot0_live.bin`
      (7 286 o) — coincidence EXACTE **0** sur les 951 films, coincidence de PREFIXE
      **949 paquets, `par type map[0:949]`, `par rang map[8:949]`**. Detail qui compte :
      7 286 octets est EXACTEMENT la taille du premier paquet type-0 des deux films CTF
      (`64e8adfa`, `530820e5`), sans qu'aucun ne coincide au contenu — la taille de ce
      paquet est donc dictee par la CARTE et le MODE, pas par le match.
- [x] 2.3 **C1 mesure — ECHOUE, largement.** `DecodeFrameRecords` sur le PREMIER paquet
      type-0, **30 combinaisons de cadre** probees par film (largeur d'id 10..14 x amorce
      0/1/2 x prologue de mode film present/absent), **6 films** :

      | film | paquet type-0 #8 | meilleure couverture | combinaison retenue | NEW propres |
      |---|---|---|---|---|
      | `000d5950` | 9 297 o (74 376 bits) | **0,33 %** (243 bits) | idLow 12, amorce 2, extra=non | 1 (`ti=43`) |
      | `00502e52` | 11 312 o (90 496 bits) | **2,75 %** (2 487 bits) | idLow 12, amorce 2, extra=non | 0 |
      | `07aa428d` | 11 066 o (88 528 bits) | **2,84 %** (2 511 bits) | idLow 12, amorce 2, extra=non | 0 |
      | `64e8adfa` | 7 286 o (58 288 bits) | **3,22 %** (1 877 bits) | idLow 12, amorce 2, extra=non | 0 |
      | `530820e5` | 7 286 o (58 288 bits) | **3,22 %** (1 877 bits) | idLow 12, amorce 2, extra=non | 0 |
      | `0014603f` | 8 514 o (68 112 bits) | **0,34 %** (233 bits) | idLow 11, amorce 2, extra=non | 1 (`ti=21`) |

      Seuil C1 = 95 % ; plafond mesure **3,22 %**. Le prologue de mode film
      (`HasExtraFields` : `R(32)` par iteration + `R(1)[+R(8)]` avant NEW/DEL, lu dans
      `FUN_1406cd128` sous `FUN_14076cea8()`) n'ameliore RIEN : il etait deja porte par
      `frame_records.go` derriere ce drapeau, et il est probe ici dans les deux positions.
      **Correction d'instrument faite en cours de mesure** : une combinaison peut lire
      AU-DELA de la fin du tampon (le `BitReader` rend des zeros passe la fin) et remporter
      le balayage avec une « couverture » absurde — mesure du 2026-08-17 sur `0014603f` :
      **12 317 %**. Le drapeau `KFQWalk.Overrun` disqualifie ces marches ; les six chiffres
      ci-dessus sont post-correction.
- [x] 2.4 **C2 mesure.** Cote marche : 1 seul `ti` lie proprement par film au mieux
      (`ti=43` sur `000d5950`, `ti=21` sur `0014603f`), donc `ti=37` / `ti=42` / `ti=11`
      **ABSENTS** — consequence directe de C1, pas une information sur le contenu du paquet.
      **Contre-liste INDEPENDANTE** (balayeur de la table type-2, chaine disjointe), et elle
      est instructive : **chaque chunk porte SA table type-2** (26 sur `000d5950`, une par
      chunk), et son contenu CROIT avec le match — `000d5950` chunk_01 : 123 records /
      13 archetypes, `ti=37` x0, `ti=42` x0 ; chunk_02 : 294 / 25, `ti=37` x10, `ti=42` x4 ;
      chunk_13 : 323 / 25, `ti=37` x29, `ti=42` x17. `ti=11` : **x0 partout sur `000d5950`**
      (film sans objectif, temoin negatif attendu) ; **x5 sur CHAQUE table des deux films
      CTF** (`64e8adfa`, `530820e5`) — les 5 slots que R4 avait mesures. `ti=42` : x4 a x21
      sur `000d5950`, **x0 sur `00502e52` et `07aa428d`**.
- [x] 2.5 **H2 mesure — REFUTEE dans sa lecture « table d'octets ».** Ecarts entre ancres
      consecutives de la table type-2, modulo 8 : `000d5950` chunk_01 — **1 ecart sur 122**
      est un multiple de 8, et 9 ancres sur 123 sont alignees sur l'octet ; `07aa428d`
      chunk_01 — 28 sur 291, et 45 ancres sur 292. Une table d'octets donnerait 100 %.
      **C'est donc bien un bitstream.** La quantification de R5 est confirmee sur une
      seconde chaine : **12 valeurs d'ecart distinctes sur 122** (`000d5950`), 39 sur 291
      (`07aa428d`).
- [x] 2.6 Instrument versionne `filmdec/keyframe_entity_queue_test.go` (3 tests), gardes
      `KFQ_FILM` / `KFQ_ROOT` + `KFQ_DUMP`, **3 SKIP verifies sans les variables**, lecture
      seule, aucune ecriture disque.
- [x] 2.7 Les trois outils de reconnaissance jetables n'ont JAMAIS ete versionnes : ils ont
      vecu dans le repertoire de brouillon de session (hors depot) et sont remplaces, un
      pour un, par les fonctions de `keyframe_entity_queue.go`. `git status` du worktree ne
      montre aucun fichier de sonde.

**H4 TESTEE ET REFUTEE, gratuitement** : rejouer les 30 combinaisons avec le World
PRE-PEUPLE par `WorldFromKeyframe` (table type-2 du meme film) donne **exactement la meme
couverture** (0,33 % / 2,75 % / 2,84 %). Le premier paquet type-0 ne bute pas sur un World
vide.

**Gate 2** :

```
CGO_ENABLED=0 go build ./internal/analysis/...
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/
gofmt -l internal/analysis/filmdec/                      (doit etre vide)
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ ./internal/analysis/replay/
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ -run '^TestKFQ' -v      (SKIP sans la garde)
CGO_ENABLED=0 KFQ_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFQ' -timeout 30m -v
```

Critere de passage : C0 tranche (identification OU negatif publie) ET C1/C2 mesures avec
leurs denominateurs PAR FILM. **Le gate passe aussi par le negatif** : si C1 echoue, la
phase publie le taux reel et la cause, et les phases 3-5 sont statuees `[!]`.

### Phase 3 — `ti=37` : le corps complet de l'equipement — `[!]` NON OUVERTE, CONDITION NON REMPLIE

Condition d'ouverture ecrite : « C1 passe ET `ti=37` present dans C2 ». C1 plafonne a 2,84 %
pour un seuil de 95 % (phase 2, item 2.3), et aucun `ti=37` n'est lie proprement. La phase
lirait le corps de records dont la marche ne tient pas : elle produirait des valeurs
inverifiables — exactement l'erreur payee le 2026-08-12 sur les armes au sol.

- [!] 3.1 Records NEW `ti=37` du premier paquet type-0 — **non releves** : la marche ne
      depasse pas 3 records. Ce que le lot mesure a la place, et qui est publie : la
      contre-liste type-2 (item 2.4) donne `ti=37` x10 a x29 selon le chunk sur `000d5950`,
      x11 sur `00502e52` chunk_02, x13 sur `07aa428d` chunk_02.
- [!] 3.2 Les 4 `R(32)` de R3 relus — sans objet, aucun record `ti=37` atteint.
- [~] 3.3 Recoupement des slots avec `WalkKeyframeWorld` — **couvert par l'item 2.4**, qui
      publie la contre-liste par chunk. Le recoupement lui-meme est sans objet (rien a
      recouper cote marche).
- [x] 3.4 **Verdict ecrit** : `ti=37` reste lisible UNIQUEMENT par la voie R3 (ancres de la
      table type-2 + marche partielle du corps), inchangee par ce lot. Le premier paquet
      type-0 n'offre PAS d'entree alternative tant que la marche sequentielle bute.

**Gate 3 : SANS OBJET.** La phase n'a pas ete ouverte ; aucun fichier `*_ti37*` cree.

### Phase 4 — `ti=42` : la position des armes au sol — `[!]` NON OUVERTE, CONDITION NON REMPLIE

Meme cause que la phase 3.

- [!] 4.1 `filmdec/keyframe_entity_queue_ti42.go` — **non cree** (code mort, anti-patron n°1).
- [!] 4.2 C3 (emprise + bande fantome) — sans objet : aucune position a encadrer.
- [x] 4.3 **Verdict ecrit sur les 55 % de positions fantomes du 2026-08-12** : inchange, et
      la condition de reprise du registre doit etre amendee une SECONDE fois. R5 l'avait
      deja jugee insuffisante (« default-state de `ti=42` resolu » l'est, depuis le
      2026-08-17, sans rien debloquer). R6 ajoute que la voie de contournement esperee — lire
      `ti=42` dans le premier paquet type-0 — est **fermee par mesure** (2,84 % de
      couverture au mieux). La contre-liste montre pourtant que la matiere existe :
      `ti=42` x4 a x21 par chunk sur `000d5950`, mais **x0 sur `00502e52` et `07aa428d`** —
      l'archetype n'est meme pas present dans la table type-2 de deux des trois films
      oracles, ce qui est une information neuve pour qui reprendra le sujet.

**Gate 4 : SANS OBJET.**

### Phase 5 — `ti=11` : l'objectif vivant — `[!]` NON OUVERTE, avec sa mesure de contexte

Meme cause. La phase a neanmoins produit sa mesure de contexte, qui ne depend pas de C1.

- [!] 5.1 Portage des composants `ti=11` — **non fait** : aucun point d'entree fiable.
- [!] 5.2 `filmdec/keyframe_entity_queue_ti11.go` — **non cree** (code mort).
- [x] 5.3 **Mesure de contexte EXECUTEE** (elle ne depend pas de C1) : contre-liste type-2
      sur les deux films CTF et sur le temoin negatif. `64e8adfa` et `530820e5` portent
      **`ti=11` x5 dans CHACUNE de leurs tables type-2** (les 5 slots de R4), `000d5950` et
      `0014603f` x0 — le temoin negatif se comporte comme prevu. Les deux films CTF ont en
      outre un premier paquet type-0 de **taille identique, 7 286 octets**, exactement la
      taille de `kf_slot0_live.bin`. L'oracle `objectiveevents.IdentifiedEvent` reste intact
      et disponible : c'est l'objet a confronter qui manque, pas le temoin.
- [x] 5.4 **Verdict ecrit** : la condition de reprise de R4 (« la grammaire du corps d'un
      record d'image-cle ») est desormais **REMPLACEE**, pas precisee : le jeu ne relit
      jamais ce corps (phase 1), donc il n'y a pas de grammaire a retrouver par lecture du
      lecteur. Ce qui reste est nomme en §11.

**Gate 5** : execute sur son perimetre reel (mesure de contexte), commandes au journal §10.

### Phase 6 — PUBLIER dans l'artefact, et noter la bosse — `[!]` RIEN A PUBLIER

La regle de non-publication du §1.1 s'applique telle qu'elle a ete ecrite : aucun controle de
donnee (C3, C4) n'est passe.

- [!] 6.1 Assemblage `replay/` — **non cree**.
- [x] 6.2 **Note de bosse : ce lot ne demande AUCUNE bosse de `SchemaVersion`.** Il n'ajoute
      aucun champ au document ; `SchemaVersion` reste a 8. Information utile au superviseur
      a la fusion : R3 peut en demander une pour son compte, R4, R5 et R6 non.
- [x] 6.3 Contrat OpenAPI / `generated.ts` / normalisation web : **sans objet, verifie** —
      aucun fichier de `internal/analysis/replay/` ni de `apps/web/` n'est touche par ce lot
      (`git diff --stat` au journal §10).

**Gate 6** : execute, sorties au journal §10.

### Phase 7 — CLORE — CLOSE le 2026-08-17

- [x] 7.1 Toutes les cases des phases 1 a 6 sont statuees, aucune vide.
- [x] 7.2 Lignes de registre redigees en une seule fois (§11), neuves ET a amender.
- [x] 7.3 Entree `thought_log.md` redigee (§12) et remise au superviseur — ce lot n'ecrit PAS
      dans le journal.
- [x] 7.4 Rapport final rendu, avec denominateurs, gates executes et ce qui reste.

---

## 6. Decisions produit — TRANCHEES avant execution

1. **L'UI n'est pas dans ce lot.** On publie la donnee, ou on ne publie rien.
2. **Aucun nom, aucun libelle en dur cote Go.** Un type non nomme garde son identifiant.
3. **Aucune comparaison `slug == "..."`** (ratchet `no_slug_comparison_test.go`). Reserve
   honnete heritee de R4/R5 : `filmdec` est aujourd'hui du RE Halo Infinite ; ce lot
   n'aggrave pas cette dette et ne la traite pas.
4. **C'est le DESERIALISEUR qui publie**, jamais un second lecteur pose a cote de lui.
5. **Rien n'est publie sans son temoin de controle** (fantome / temoin negatif obligatoire).
6. **Offline-pur** : aucune capture runtime, aucun Cheat Engine. Ghidra en LECTURE SEULE.
7. **Pas de CGO** sur `filmdec`/`replay`.
8. **Un negatif mesure est un livrable**, publie avec ses denominateurs et ses temoins.
9. **Les sondes de reconnaissance jetables du §2.2 ne survivent pas au lot** (item 2.7).

## 6bis. Conformite architecture (grille `plan-review`, passee le 2026-08-17)

| couche | ce que ce lot y met |
|---|---|
| `internal/analysis/filmdec/` | le modele de file, la reconciliation de paquet et les instruments de mesure — algorithme pur, aucun acces DB/HTTP |
| `internal/analysis/replay/` | l'assemblage du calque (phase 6) — pur, sans I/O |
| `internal/service/`, `internal/port/`, `internal/api/handlers/` | **RIEN** : le champ voyagerait sur `ReplayDocument`, deja construit et servi |
| `platform/duckdb/`, `persist/` | **RIEN** : aucune lecture ni ecriture DuckDB |
| `apps/web/` | **RIEN** |

- Seuils : fichier <= 500 L, fonction <= 80 L, <= 5 parametres. Les grandeurs d'une marche
  sont regroupees en struct plutot qu'ajoutees en parametres.
- Logging : les instruments sont des TESTS, ils rendent par `t.Logf`. Le code d'assemblage de
  `replay/` est PUR. **Aucun `fmt.Println` / `log.Printf`.** Aucune erreur avalee en silence.
- Tests : instruments gardes par variable d'environnement (sautent en CI) + un test unitaire
  PUR sans I/O pour tout assemblage de `replay/`.
- Multi-titre : aucun `PathResolver` en jeu (les films sont lus par chemin passe en
  parametre, comme tout `filmdec`) ; aucune capability nouvelle ; aucun champ TOML.
- i18n / couleurs / query keys / routes : sans objet (aucun rendu dans ce lot).

## 7. Contrat d'interface avec les lots paralleles

R3 bis tourne dans le worktree principal (`ti=37`) ; l'habillage est en fusion.

1. **Je ne CREE que mes fichiers** : `filmdec/keyframe_entity_queue*.go`, tout fichier
   `*_kfq_*`, ce plan, et la sous-section « file par entite » de `WALK_PORT_NOTES.md`.
2. **Fichiers PARTAGES du decodeur** (`traverse.go`, `default_state*.go`, `keyframe_world.go`,
   `frame_records.go`, `replay/build.go`, `replay/document.go`, `replay/coverage.go`) :
   **une ligne d'enregistrement chacun au maximum**, jamais de reecriture, jamais de
   reindentation, jamais de reordonnancement.
3. **AUCUNE bosse de `SchemaVersion`.**
4. **Interdits** : aucun run de masse, aucune re-cuisson d'artefact publie sous `data/`,
   aucune ecriture DuckDB, aucune ecriture hors du worktree `LevelUp-wt-kfqueue`. Les films
   et les autres worktrees sont lus en LECTURE SEULE.
5. **Cache Go isole** : `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfqueue/.gocache`,
   **une seule commande `go` a la fois**.
6. Commits par phase sur `wt/kf-file-entite` (`feat(v7.5-rejeu-kfq):` / `mesure(...)` /
   `docs(...)`, terminaison `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`), hooks
   ACTIFS (jamais `--no-verify`), `git push -u origin wt/kf-file-entite` apres chaque phase
   close. Jamais `git stash`, jamais `main`, jamais de merge.
7. Ce lot n'ecrit PAS dans `.ai/thought_log.md` (superviseur) ; l'entree est redigee au §12.

## 8. Statuts d'item et cloture

`[x]` fait et verifie · `[~]` couvert ailleurs (avec la reference) · `[!]` non traite (avec
la justification ecrite). **Aucune case vide a la cloture d'une phase.** Clore une phase =
gate passe + items statues + plan a jour + commit + push + point d'etape.

## 9. Decouvertes — a consigner, NE PAS traiter dans ce lot

- (reconnaissance) Le pont `mcp__ghidra__*` ne se connecte plus : l'instance UDS se declare
  `unknown` et `connect_instance` refuse tout repli TCP. Contournement en place (HTTP direct,
  lecture seule). A traiter hors lot.
- **(phase 2) LE LEVIER NON CONSOMME, et c'est le meilleur du dossier.**
  `.ai/V7.5/dumps/kf_capture_sample.txt` porte **400 frontieres de records EXACTES**
  (266 NEW + 134 DELTA) avec leur BIT DE DEPART, sur un vrai paquet, dont le tampon est
  `kf_slot0_live.bin` (7 286 o). C'est un **oracle de LARGEUR par archetype** : pour chaque
  record on connait le bit de debut ET le bit de fin, donc la longueur exacte que l'etat par
  defaut + les composants doivent consommer. Aucun lot ne l'a jamais utilise pour CALIBRER
  les etats par defaut, alors que c'est precisement ce qui manque depuis juillet. Largeurs
  lisibles directement dans la capture : 275 bits (le mode), 321, 60, 32, 25.
  NE PAS traiter ici — c'est le lot suivant, et il a son oracle.
- **(phase 2) Les deux films CTF ont un premier paquet type-0 de taille IDENTIQUE**
  (7 286 octets, `64e8adfa` et `530820e5`), qui est exactement la taille de
  `kf_slot0_live.bin`. La taille du premier paquet delta semble donc dictee par la CARTE et
  le MODE, pas par le match. NE PAS traiter ici.
- **(phase 2) Chaque chunk d'un film porte SA table type-2**, et son contenu croit avec le
  match (123 records au chunk_01, ~320 en milieu de match). Le depot parlait d'« une
  image-cle par film » ; c'est faux, il y en a une par chunk (26 sur `000d5950`).
  NE PAS traiter ici, mais toute mesure future doit publier DE QUEL chunk elle parle.
- **(phase 2) `ti=42` est absent des tables type-2 de deux des trois films oracles**
  (`00502e52` et `07aa428d` : x0 ; `000d5950` : x4 a x21 selon le chunk). Un lot qui
  reprend les armes au sol doit choisir son corpus en consequence. NE PAS traiter ici.
- **(phase 2) Bug d'instrument generique, a connaitre** : `DecodeFrameRecords` ne s'arrete
  PAS a la fin du tampon — le `BitReader` rend des zeros passe la fin et la boucle continue.
  Tout balayage de combinaisons qui classe par « bits consommes » doit disqualifier les
  marches qui debordent, sinon une combinaison absurde gagne (mesure : 12 317 % de
  couverture sur `0014603f`). Corrige DANS MON fichier (`KFQWalk.Overrun`) ; le fichier
  partage `frame_records.go` n'est PAS touche (hors perimetre).

## 10. Journal d'execution

**2026-08-17 — Phase 1 CLOSE.** Lecture Ghidra read-only (13 fonctions decompilees, une
fonction desassemblee instruction par instruction), via l'API HTTP du plugin, le pont MCP
etant hors service (§4bis). Trois resultats, dont deux NEUFS :

1. **La file par-entite n'est PAS une transformation** : l'item de 56 octets porte un handle
   vers une COPIE OCTET POUR OCTET du tampon du paquet (`FUN_142f25334` = `memcpy` integral),
   plus le BIT DE DEPART du record (`item+0x2c`, pose dans le reader par `FUN_1432fe23c`).
   Le drain rejoue le MEME `FUN_1406cbaa0` avec la MEME grammaire. La file DIFFERE les
   records, elle ne les reecrit pas. L'objet demande par la condition de reprise de R5
   n'existe pas.
2. **Le demultiplexeur de paquets du film est `FUN_1428e22c0`** (`sVar2 = *param_3` = le `u16`
   de tete de l'en-tete de 16 octets). Verifie au DESASSEMBLAGE : type 0 -> decodeur de
   replication, type 1 -> `FUN_142989418`, **type 2 -> AUCUN handler** (`JZ 0x1428e2412` =
   `XOR SIL,SIL` + telemetrie `FilmBlockReadError`), types 6/7/8/9/10/11/12 -> handlers.
3. **Le payload type-2 est SAUTE, jamais decode** : `FUN_142989418` (handler du type 1)
   avance le curseur de la taille du type-1, relit l'en-tete SUIVANT (16 o) et avance encore
   le curseur de SA taille — c'est-a-dire qu'il saute le bloc type-2 avec lui. Le jeu ne lit
   donc jamais ces records. Cela explique a posteriori le negatif de R5.

Chemin complementaire mesure offline : les 16 premiers octets de `keyframe_buffer_live.bin`
sont ceux du PREMIER paquet type-0 de chaque film (`88 00 15 84 00 2c 54 0c 61 c9 00 0b
ff ff ff fc`) — la capture live de juillet portait sur le premier paquet DELTA, pas sur
l'image-cle.

**Gate 1 : PASSE le 2026-08-17.** Commandes et sorties exactes :

```
grep -ci "file par entite" .ai/V7.5/killweapon/WALK_PORT_NOTES.md                          -> 2
grep -o "FUN_1[0-9a-f]*\|0x14[0-9a-f]*" .ai/V7.5/killweapon/WALK_PORT_NOTES.md | sort -u | wc -l -> 90
grep -c "FUN_142f29538\|FUN_142f2913c\|FUN_1432fe23c\|FUN_142f25334" .ai/V7.5/killweapon/WALK_PORT_NOTES.md -> 8
```

Critere (section presente, > 62 adresses distinctes, les 4 adresses du couple citees,
question 1.3 tranchee par ecrit) : REMPLI — 90 adresses distinctes, 62 avant ce lot
(98 apres l'ajout 1.7).

**2026-08-17 — Phase 2 CLOSE PAR LE NEGATIF sur C1 (le plan prevoyait les deux issues).**

C0 — reconciliation, sortie exacte de l'instrument :

```
CGO_ENABLED=0 KFQ_ROOT=<repo>/data/cache/film_chunks \
  KFQ_DUMP=.ai/V7.5/dumps/keyframe_buffer_live.bin \
  go test ./internal/analysis/filmdec/ -run '^TestKFQReconcile$' -timeout 60m -v

  dump keyframe_buffer_live.bin : 11485 octets ; denominateur : 951 entrees
  coincidence EXACTE : 0 paquet(s)
  coincidence de PREFIXE (16 octets) : 949 paquet(s) ; par type map[0:949] ; par rang map[8:949]
  --- PASS: TestKFQReconcile (550.00s)
```

**Confirmation de C0, executee.** La sortie ci-dessus vient de l'instrument dans sa version
a DEUX balayages (un par question). Le fichier a ensuite ete refactore en UN SEUL balayage
(`FindPackets` + predicats `KFQEqual` / `KFQPrefix`). La re-execution sur le MEME corpus
avec la version refactoree rend **exactement les memes deux chiffres** (`0 paquet(s)` /
`949 paquet(s) ; par type map[0:949] ; par rang map[8:949]`) et **le cout tombe de 550 s a
219 s** — le refactor est donc verifie sur pieces, pas seulement plausible.

Second dump, meme instrument, meme corpus :

```
CGO_ENABLED=0 KFQ_ROOT=<repo>/data/cache/film_chunks \
  KFQ_DUMP=.ai/V7.5/dumps/kf_slot0_live.bin \
  go test ./internal/analysis/filmdec/ -run '^TestKFQReconcile$' -timeout 30m -v

  dump kf_slot0_live.bin : 7286 octets ; denominateur : 951 entrees
  coincidence EXACTE : 0 paquet(s)
  coincidence de PREFIXE (16 octets) : 949 paquet(s) ; par type map[0:949] ; par rang map[8:949]
  --- PASS: TestKFQReconcile (1563.77s)
```

C1 / C2 / H4 — six films, 30 combinaisons de cadre chacun (tableau a l'item 2.3).
Plafond de couverture **3,22 %** pour un seuil de 95 %. Le World pre-peuple par
`WorldFromKeyframe` ne change RIEN sur aucun des six films (H4 refutee).

H2 — forme des ancres de la table type-2 :

```
CGO_ENABLED=0 KFQ_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFQAnchorShape$' -v
  000d5950 chunk_01 : 123 ancres, 9 alignees octet, ecarts mod 8 = map[0:1 1:33 2:4 3:32 4:51 5:1],
                      12 valeurs d'ecart distinctes sur 122
  07aa428d chunk_01 : 292 ancres, 45 alignees octet, ecarts mod 8 = map[0:28 1:34 2:3 3:43 4:103 5:16 6:32 7:32],
                      39 valeurs distinctes sur 291
```

**GATE 2 : PASSE PAR LE NEGATIF.** Commandes et sorties :

```
gofmt -l internal/analysis/filmdec/                             (vide)
CGO_ENABLED=0 go build ./internal/analysis/...                  BUILD_EXIT=0
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/             VET_EXIT=0
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ ./internal/analysis/replay/
    ok levelup/go-api/internal/analysis/filmdec  37.704s
    ok levelup/go-api/internal/analysis/replay   34.532s        TEST_EXIT=0
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ -run '^TestKFQ' -v
    --- SKIP: TestKFQReconcile / TestKFQFirstDelta / TestKFQAnchorShape   (3 SKIP, garde OK)
```

**GATE 6 (perimetre) : PASSE.**

```
git diff --stat ffc465336 -- apps/web/                          (vide)
git diff --stat ffc465336 -- apps/go-api/internal/analysis/replay/   (vide)
grep -n "SchemaVersion = " internal/analysis/replay/document.go -> 78: const SchemaVersion = 8
```

## 11. Lignes de registre proposees

A verser en une seule fois dans `.ai/V7.5/REGISTRE_REPORTS.md`. **Deux lignes NEUVES**, et
**trois lignes a AMENDER** (dont une que R5 avait deja proposee d'amender : l'amendement
change de nature).

| sujet | lot / date | ce qui a ete mesure | condition de reprise |
|---|---|---|---|
| **[NEUF — il ferme le verrou de R5] Le jeu ne relit JAMAIS le payload type-2 d'un film : il le SAUTE** | lot R6, phase 1, 2026-08-17 | Etabli par LECTURE, et verifie au DESASSEMBLAGE. (1) `FUN_1428e22c0` est l'aiguillage par type de paquet du lecteur de film (`MOVSX EDX,word ptr [R8]` @`0x1428e22ca`, chaine `CMP 8 / TEST / SUB 1 / SUB 1 / SUB 4 / CMP 1`) : handlers pour les types 0, 1, 6, 7, 8, 9, 10, 11, 12 ; **le type 2 n'en a AUCUN** — il saute a `0x1428e2412` = `XOR SIL,SIL` (retour 0) + telemetrie `FilmBlockReadError`, comme les types 3, 4 et 5. (2) Et pourtant la lecture ne casse pas : `FUN_142989418` (handler du type 1) avance le curseur de la taille du type-1, **relit l'en-tete suivant (16 o) et avance encore le curseur de SA taille** — il saute donc le bloc type-2 avec lui. (3) La FILE PAR ENTITE n'est pas une transformation : l'item de 56 octets porte un handle vers une COPIE octet pour octet du tampon du paquet (`FUN_142f25334` = `memcpy` integral, chemin source `replication_entity_manager_view.cpp`), la longueur du tampon, l'id, **le BIT DE DEPART du record** (`item+0x2c`, pose dans le reader par `FUN_1432fe23c`) et le type ; le drain `FUN_142f2913c` rejoue le MEME `FUN_1406cbaa0`. Unique xref CODE du push : `0x1422f44fb`, dans `FUN_1406cd128`. **Consequence : la condition de reprise que R5 avait ecrite (« decompiler le consommateur du payload type-2 ») est SANS OBJET — il n'y a pas de consommateur** | condition de reprise : la grammaire du bloc type-2 ne s'obtiendra PAS par lecture d'un lecteur. Les deux voies restantes, dans l'ordre de cout : (a) par le CONTENU — la table est deja balayee a 249/250 entites contre un oracle Cheat Engine, et ses ecarts d'ancres sont fortement quantifies (12 valeurs distinctes sur 122 ecarts, `000d5950` chunk_01) ; (b) par l'ECRIVAIN, qui **n'est pas dans `HaloInfinite.exe`** : recherche bornee et negative (une seule chaine `saved_games` dans tout le binaire, celle du LECTEUR ; une seule chaine `FilmBlock*`, `FilmBlockReadError`, cote lecture ; l'encodeur `FUN_142f2e174` sans appel direct). **Et la vraie question a poser avant de reprendre : a quoi sert une table que le jeu n'ouvre jamais ?** |
| **[NEUF] Le premier paquet type-0 ne se traverse pas non plus — et c'est LE MEME mur que partout** | lot R6, phase 2, 2026-08-17 | La capture live de juillet (`.ai/V7.5/dumps/keyframe_buffer_live.bin`, 11 485 o, deterministe entre deux lancements), etiquetee « keyframe » depuis, portait en realite sur le **PREMIER PAQUET DELTA** : ses 16 premiers octets (`88 00 15 84 00 2c 54 0c 61 c9 00 0b ff ff ff fc`) sont ceux du premier paquet de type 0 de chaque film, toujours au rang #8 d'un chunk. Coincidence EXACTE sur les **951 films** du cache : **0** — le film de la capture n'est pas cache. Mesure de la marche : `DecodeFrameRecords` sur ce premier paquet type-0, **30 combinaisons de cadre** (largeur d'id 10..14 x amorce 0/1/2 x prologue de mode film present/absent), **6 films** : 0,33 / 2,75 / 2,84 / 3,22 / 3,22 / 0,34 % des bits consommes (seuil 95 %, plafond mesure **3,22 %**), au mieux 1 seul `ti` lie proprement par film. H4 refutee : le World pre-peuple par `WorldFromKeyframe` donne EXACTEMENT la meme couverture. Le prologue de mode film (`R(32)` par iteration + `R(1)[+R(8)]` avant NEW/DEL, lu dans `FUN_1406cd128` sous `FUN_14076cea8()`) etait deja porte derriere `FrameConfig.HasExtraFields` et n'ameliore rien. H2 refutee dans sa lecture « table d'octets » : 1 ecart d'ancres sur 122 est multiple de 8 (`000d5950`), 28 sur 291 (`07aa428d`) — c'est bien un bitstream | condition de reprise : **le deserialiseur d'etat par defaut par archetype, bit-exact**. C'est le mur documente depuis juillet, et R6 etablit qu'il est LE MEME sur le premier paquet delta que partout ailleurs — il n'y a pas de porte derobee par l'image-cle. Levier disponible et non consomme : `.ai/V7.5/dumps/kf_capture_sample.txt` donne **400 frontieres de records EXACTES** (266 NEW + 134 DELTA, avec leur bit de depart) sur un vrai paquet, avec son tampon `kf_slot0_live.bin` — c'est un oracle de LARGEUR par archetype, jamais exploite pour calibrer les etats par defaut. Reproductible : `KFQ_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFQ' -v` |
| **[A AMENDER — ligne « Calque drops / armes au sol du rejeu 2D », 2026-08-12]** | amendement lot R6, 2026-08-17 | R5 avait deja signale que la condition ecrite (« default-state de `ti=42` resolu ») etait REMPLIE et INSUFFISANTE. R6 ferme la voie de contournement esperee : lire `ti=42` dans le premier paquet type-0 est impossible aujourd'hui (2,84 % de couverture au mieux, mesure sur 30 combinaisons x 3 films). Information neuve utile : `ti=42` est present dans la table type-2 de `000d5950` (x4 a x21 selon le chunk) mais **x0 sur `00502e52` et `07aa428d`** — deux des trois films oracles n'en portent aucun | remplacer la condition par : « etat par defaut par archetype bit-exact (mur commun), OU evenements de cycle de vie d'entite decodes offline ». Retirer toute mention d'une voie « image-cle » : elle est fermee |
| **[A AMENDER — ligne « objectifs vivants `ti=11` », lot R4]** | amendement lot R6, 2026-08-17 | La condition de reprise de R4 puis R5 (« la grammaire du CORPS d'un record d'IMAGE-CLE ») n'est pas « precisee », elle est **SANS OBJET** : le jeu ne relit jamais ce corps, il n'existe aucun lecteur a decompiler. Le verrou reel est le meme que pour tous les autres sujets : l'etat par defaut par archetype, bit-exact | remplacer par : « etat par defaut par archetype bit-exact ». L'oracle `objectiveevents.IdentifiedEvent` reste intact et disponible |
| **[A AMENDER — ligne R5 « la grammaire du CORPS d'un record d'IMAGE-CLE »]** | amendement lot R6, 2026-08-17 | La condition de reprise ecrite par R5 (« decompiler le CONSOMMATEUR du payload type-2 ; la transformation payload -> file par-entite n'est identifiee nulle part ») est **executee et negative sur les deux termes** : il n'y a pas de consommateur (le lecteur saute le bloc), et la file n'est pas une transformation (copie du tampon + bit de depart). Le negatif de R5 (« le corps d'un record d'image-cle n'est pas un record NEW », 128 decalages x 16 lectures x 3 films) recoit ainsi son EXPLICATION : ces records ne sont relus par aucun deserialiseur du jeu | remplacer par la ligne NEUVE ci-dessus (« le jeu ne relit jamais le payload type-2 ») |

## 12. Entree `thought_log.md` proposee

```
## [2026-08-17] Lot R6 — la file par entite n'existe pas comme transformation, et le jeu
ne relit jamais son image-cle

Statut : Complete (negatif mesure sur les deux fronts, livrable).
Branche : wt/kf-file-entite (worktree LevelUp-wt-kfqueue), 3 commits, pousses.

DECISION TECHNIQUE. R5 avait laisse une condition de reprise precise : « decompiler le
consommateur du payload type-2, qui alimenterait une file par-entite ». R6 l'a executee
par LECTURE d'abord (Ghidra read-only, API HTTP du plugin, le pont MCP etant hors
service), mesure ensuite. Les deux termes de la condition sont faux.

RESULTATS OBSERVES.
1. La file par-entite n'est PAS une transformation. Son item de 56 octets porte un handle
   vers une COPIE octet pour octet du tampon du paquet (FUN_142f25334 = memcpy integral)
   et le BIT DE DEPART du record (item+0x2c, pose dans le reader par FUN_1432fe23c). Le
   drain FUN_142f2913c rejoue le MEME FUN_1406cbaa0 avec la MEME grammaire. Elle DIFFERE
   des records, elle ne les reecrit pas.
2. Le jeu ne relit JAMAIS le payload type-2. Verifie au desassemblage : FUN_1428e22c0 est
   l'aiguillage par type de paquet, et le type 2 n'a aucun handler (XOR SIL,SIL +
   telemetrie FilmBlockReadError). Il ne casse rien parce que FUN_142989418, le handler
   du type 1, saute le payload type-1 PUIS relit l'en-tete suivant et saute son payload
   avec — c'est-a-dire le bloc type-2. Cela EXPLIQUE le negatif de R5.
3. La capture live de juillet, etiquetee « keyframe » depuis, portait sur le PREMIER
   PAQUET DELTA : ses 16 premiers octets sont ceux du premier paquet type-0 de chaque
   film (rang #8). Coincidence exacte sur les 951 films du cache : 0 (le film de la
   capture n'est pas cache) ; mais la coincidence de PREFIXE (16 octets) rend 949 paquets
   sur 951 films, TOUS de type 0 et TOUS au rang #8 — la reconciliation structurelle est
   donc universelle.
4. Ce premier paquet type-0 ne se traverse pas davantage : 30 combinaisons de cadre x
   6 films, de 0,33 % a 3,22 % des bits consommes pour un seuil de 95 %. Le World
   pre-peuple par la table type-2 ne change RIEN (H4 refutee). Le mur est donc le meme
   partout : l'etat par defaut par archetype, bit-exact.
5. L'ecrivain du bloc type-2 n'est pas dans HaloInfinite.exe : une seule chaine
   saved_games dans tout le binaire (celle du LECTEUR), une seule chaine FilmBlock*
   (FilmBlockReadError), l'encodeur FUN_142f2e174 sans appel direct. Fil borne, ferme.

CONCLUSION / PROCHAINE ETAPE. Trois lots (R4, R5, R6) ont converge sur le meme verrou et
R6 lui donne enfin son vrai nom : ce n'est pas « la grammaire de l'image-cle », c'est
l'ETAT PAR DEFAUT PAR ARCHETYPE, bit-exact — le mur documente depuis juillet. Le levier
disponible et jamais consomme est kf_capture_sample.txt : 400 frontieres de records
EXACTES (266 NEW + 134 DELTA, avec leur bit de depart) sur un vrai paquet, avec son
tampon kf_slot0_live.bin. C'est un oracle de LARGEUR par archetype. Rien n'est publie,
SchemaVersion reste a 8.
```

## 13. Protocole de reprise de session

1. Relire le skill `plan-execution`, puis ce fichier de haut en bas : les cases disent ou en
   est le lot, le journal §10 dit ce qui a ete mesure.
2. Verifier la branche : `git -C C:/Users/Guillaume/Projects/LevelUp-wt-kfqueue branch
   --show-current` doit rendre `wt/kf-file-entite`.
3. Relire §7 (contrat d'interface) AVANT de toucher un fichier partage du decodeur.
4. **Verifier sur pieces** avant de coder ET avant de cocher : les references de §2 sont
   datees du 2026-08-17, et d'autres lots travaillent en parallele.
5. Reprendre a la premiere case non statuee de la phase courante. Ne pas re-decider ce qui
   est deja tranche (§6 : decisions fermes).
