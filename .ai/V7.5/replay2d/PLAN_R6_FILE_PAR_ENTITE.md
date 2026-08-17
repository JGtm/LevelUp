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

**Gate 1** (a executer et coller au journal) :

```
grep -ci "file par entite" .ai/V7.5/killweapon/WALK_PORT_NOTES.md                       -> >= 1
grep -o "FUN_1[0-9a-f]*\|0x14[0-9a-f]*" .ai/V7.5/killweapon/WALK_PORT_NOTES.md | sort -u | wc -l  -> > 62
grep -c "FUN_142f29538\|FUN_142f2913c\|FUN_1432fe23c\|FUN_142f25334" .ai/V7.5/killweapon/WALK_PORT_NOTES.md -> >= 4
```

Critere : les 4 adresses du couple producteur/consommateur citees, la question 1.3 tranchee
par ecrit, le compte d'adresses distinctes en hausse.

### Phase 2 — PORTER et PROUVER : reconciliation (C0) puis marche (C1, C2)

- [ ] 2.1 `filmdec/keyframe_entity_queue.go` ECRIT : le modele d'item de la file (struct
      documentee, portee depuis la decompile) et **`FindPacketByPayload`** — la
      reconciliation d'un tampon capture avec un paquet de film, par `WalkPackets`. Il
      REUTILISE `WalkPackets` / `ReadFilmChunk` ; aucun deser recopie ; aucune adresse en
      dur.
- [ ] 2.2 **C0 mesure** : `keyframe_buffer_live.bin` et `kf_slot0_live.bin` reconcilies —
      type de paquet, rang, taille, et film si le cache le contient. Le negatif « film
      absent du cache » est un resultat acceptable ET publiable, a condition de publier la
      preuve structurelle (prefixe commun + rang du paquet) et le denominateur du balayage.
- [ ] 2.3 **C1 mesure** : `DecodeFrameRecords` sur le PREMIER paquet type-0 de chaque film
      oracle. Publier PAR FILM : nombre de records, NEW / DEL / DELTA, bits consommes /
      bits du paquet, cause d'arret nommee.
- [ ] 2.4 **C2 mesure** : liste des `typeIndex` lies par des NEW PROPRES, avec le compte
      par archetype et la presence ou l'absence de `ti=37`, `ti=42`, `ti=11`. Contre-liste :
      les `ti` que `WalkKeyframeWorld` rend sur le payload type-2 du meme film.
- [ ] 2.5 **H2 mesure (gratuite, tant qu'on y est)** : distribution des mots de 32 bits a
      `+32` sur les ancres du payload type-2 et distribution des ecarts entre ancres
      modulo 8. Conclusion ecrite : bitstream ou table d'octets.
- [ ] 2.6 Instrument versionne `filmdec/keyframe_entity_queue_test.go`, garde
      `KFQ_FILM` (SKIP verifie sans la variable), lecture seule, aucune ecriture disque.
- [ ] 2.7 Les deux outils de reconnaissance jetables du §2.2 sont SUPPRIMES ou remplaces par
      l'instrument versionne (regle 7 : zero code mort, zero sonde jetable qui survit).

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

### Phase 3 — `ti=37` : le corps complet de l'equipement

Condition d'ouverture : C1 passe ET `ti=37` present dans C2.

- [ ] 3.1 Les records NEW `ti=37` du premier paquet type-0 releves : compte par film, slots,
      et longueur de traversee.
- [ ] 3.2 Les 4 `R(32)` publies par R3 relus sur ces records, avec leur stabilite par slot.
- [ ] 3.3 Preuve independante : les slots `ti=37` trouves ici recoupent ceux que
      `WalkKeyframeWorld` rend sur le payload type-2 du meme film. Denominateur publie.
- [ ] 3.4 Verdict ecrit (positif ou negatif mesure).

**Gate 3** : les 4 commandes de build/vet/gofmt/test du gate 2, plus la commande d'instrument
`KFQ_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFQTi37' -v`.

### Phase 4 — `ti=42` : la position des armes au sol

Condition d'ouverture : C1 passe ET `ti=42` present dans C2.

- [ ] 4.1 `filmdec/keyframe_entity_queue_ti42.go` — lecture des positions sur les records NEW
      `ti=42` propres du premier paquet type-0. Reutilise `consumeDefaultStateTI42` (R5).
- [ ] 4.2 C3 mesure : emprise + bande fantome de meme cardinalite par le MEME code.
- [ ] 4.3 Verdict ecrit sur les 55 % de positions fantomes du 2026-08-12.

**Gate 4** : gate 2 + `KFQ_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFQTi42' -v`.

### Phase 5 — `ti=11` : l'objectif vivant

Condition d'ouverture : C1 passe ET `ti=11` present dans C2 (films `64e8adfa`, `530820e5`).

- [ ] 5.1 Portage MINIMAL des composants `ti=11` necessaires a `i5`, `i12`, `i13`, `i14` ;
      les autres composants statues explicitement.
- [ ] 5.2 `filmdec/keyframe_entity_queue_ti11.go` — lecture des 4 champs.
- [ ] 5.3 C4 mesure contre `objectiveevents.IdentifiedEvent` + le temoin negatif `0014603f`.
- [ ] 5.4 Verdict ecrit.

**Gate 5** : gate 2 + `KFQ_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFQTi11' -v`.

### Phase 6 — PUBLIER dans l'artefact, et noter la bosse

- [ ] 6.1 Assemblage `replay/` — SEULEMENT si au moins un controle C3/C4 est passe.
- [ ] 6.2 Champ optionnel sur `ReplayDocument` — **AUCUNE bosse de `SchemaVersion` dans ce
      lot**. Le champ ajoute (le cas echeant) et la raison pour laquelle il EXIGERA une
      bosse a la fusion sont ECRITS ici, pour le superviseur.
- [ ] 6.3 Contrat OpenAPI / `generated.ts` / normalisation web : sans objet si aucun champ ne
      traverse ; a verifier explicitement et a statuer.

**Gate 6** : gate 2 + `grep -n "SchemaVersion" internal/analysis/replay/document.go` (doit
toujours rendre 8) + `git diff --stat` sur `apps/web/` (doit etre vide).

### Phase 7 — CLORE

- [ ] 7.1 Toutes les cases des phases 1 a 6 statuees, aucune vide.
- [ ] 7.2 Lignes de registre redigees en une seule fois (§11), neuves ET a amender.
- [ ] 7.3 Entree `thought_log.md` redigee (§12) et remise au superviseur — ce lot n'ecrit PAS
      dans le journal.
- [ ] 7.4 Rapport final rendu, avec denominateurs, gates executes et ce qui reste.

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
question 1.3 tranchee par ecrit) : REMPLI — 90 adresses distinctes, 62 avant ce lot.

## 11. Lignes de registre proposees

(redigees en phase 7)

## 12. Entree `thought_log.md` proposee

(redigee en phase 7)

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
