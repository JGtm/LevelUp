# Plan R5 — La grammaire du CORPS d'un record d'IMAGE-CLE

> Ecrit le 2026-08-17. Lot R5, ouvert par la conclusion CONVERGENTE de deux lots mesures le
> meme jour : R3 (`PLAN_R3_IDENTITE_TI37.md`, phase 1.3) et R4
> (`PLAN_R4_OBJECTIFS_VIVANTS_TI11.md`, phase 4). Les deux nomment le MEME manque :
> « le deserialiseur du corps d'un record d'image-cle (la fonction qui suit l'en-tete
> 64 bits) » — resolu NULLE PART dans le depot.
> Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-kfgram`, branche `wt/kf-grammaire`,
> base `wt/ti37-identite` = `339d2956e` (= `feat/v75` `3058afbba` + le lot R3).
> Execution sous le contrat du skill `plan-execution` (ordre strict, un item = un statut,
> zero fix hors perimetre, zero report d'une etape executable).

> **AMENDEMENT A LA FUSION (2026-08-17, `wt/fusion-lots-go`)** — l'item 3.1 de ce plan
> (« Etat par defaut de `ti=42` PORTE ET INSCRIT ») a ete DEFAIT au moment de la fusion, sur
> decision superviseur : `filmdec/default_state_ti42.go` et la ligne `42:` de
> `defaultStateDeserByTI` ne sont PAS dans la branche d'integration. Motif : aucun oracle ne
> valide ce port, et une entree fausse decale le decodage de TOUS les records `ti=42` sans
> qu'aucune mesure ne le dise. La grammaire, elle, est conservee — elle fait foi dans
> `../killweapon/WALK_PORT_NOTES.md` § IMAGE-CLE §4, et sa condition de rebranchement est au
> registre (`../REGISTRE_REPORTS.md`). Tout ce que ce plan dit du port en Go (items 3.1, §6bis,
> §10, ligne de registre §11) est donc PERIME sur ce point ; le reste du plan tient.

---

## 1. Objectif et critere de succes

**Objectif.** Etablir, PAR LECTURE DU CODE DU JEU puis par mesure, la grammaire exacte du
corps d'un record de la table d'image-cle (payload de frame type-2), la porter en Go, et
s'en servir pour lever DEUX reports datés que ce seul deblocage commande :

1. **`ti=42`** — position des armes au sol a l'image-cle (report du 2026-08-12,
   `REGISTRE_REPORTS.md` : « 55 % de positions fantomes », condition de reprise ecrite
   « default-state de `ti=42` resolu »).
2. **`ti=11`** — l'objectif vivant : `i5 type`, `i12 progress` / `i13 required-progress`,
   `i14 state` — « ou en est une capture » (report du 2026-08-17, lot R4).

**Ce que ce lot ne cherche PAS** (perimetre FERME) : aucun rendu, aucun son, aucune string
UI ; aucune bosse de `SchemaVersion` ; aucune re-cuisson d'artefact ; aucune ecriture
DuckDB ; aucun nommage d'un `eqip` (c'est le report de R3, pas celui-ci) ; aucune
correction du decalage `originMs` du calque d'objectifs (deja au registre).

### 1.1 Criteres mesurables, avec leurs denominateurs

| C | controle | seuil | temoin independant |
|---|---|---|---|
| **C1** | **MARCHE BIT-EXACTE `ti=37`** — la marche du corps d'un record d'image-cle atterrit sur le premier bit du record suivant | **>= 95 %** des **1 226** records `ti=37` bornes de l'oracle R3 (3 films) | l'oracle est `WalkKeyframeWorld`, chaine INDEPENDANTE (il ne decode aucun corps : il balaie des en-tetes) |
| **C2** | **SECOND ARCHETYPE** — meme controle sur au moins un archetype autre que `ti=37`, sur le meme corpus | >= 95 %, denominateur publie | idem C1 ; l'archetype est choisi par sa FREQUENCE, pas par sa commodite |
| **C3** | **MARCHE GLOBALE** — un walker DETERMINISTE (qui PARSE au lieu de BALAYER) reproduit la liste de records de `WalkKeyframeWorld` comme un SUR-ENSEMBLE, sans desync | >= 95 % des images-cles d'un film traversees de bout en bout ; les records de l'oracle tous retrouves | `WalkKeyframeWorld` (249/250 entites contre un oracle Cheat Engine, 2026-07-11) |
| **C4** | **`ti=42` POSITIONS** — les positions d'armes au sol lues a l'image-cle tombent dans l'emprise de la carte ET battent une bande FANTOME de meme cardinalite passee par le MEME code | dans l'emprise >= 95 % ; reel/fantome >= 3x | emprise = nuage de positions biped du meme film (autre chaine) ; fantome = patron du lot arme-au-sol |
| **C5** | **`ti=11` PROGRESSION** — `i5` / `i12` / `i13` / `i14` lus, valeurs STABLES par vie d'entite et coherentes avec les evenements de capture deja decodes | i12 <= i13 sur >= 95 % des lectures ; i5 stable par slot sur >= 95 % | `objectiveevents.IdentifiedEvent` (statborg), chaine TOTALEMENT disjointe |

**Un negatif MESURE est un livrable.** Toute phase qui echoue publie ses denominateurs, son
temoin de controle et sa cause nommee, et statue `[!]` avec la justification ecrite.
**Regle de non-publication** : aucune donnee n'entre dans l'artefact tant que le controle de
sa phase n'est pas passe.

---

## 2. Etat des lieux — VERIFIE SUR PIECES le 2026-08-17

### 2.1 Le mur, mesure deux fois le meme jour

| fait | piece |
|---|---|
| 8 combinaisons de grammaire probees, **2 marches bit-exactes sur 1 226 records bornes** (0,16 %), 3 films | `PLAN_R3_IDENTITE_TI37.md` §Phase 1 gate ; instrument `filmdec/equipment_identity.go:216` (`ScanFilmEquipmentIdentity`) et `:283` (`probeLayouts`) |
| **ZERO desynchronisation** sur ces 1 226 marches — donc tous les composants de `ti=37` sont portes, et la marche s'arrete TOUJOURS **TROP TOT**, de 557 a 1 104 bits | idem ; conclusion 2 de la phase 1 de R3 |
| L'ecart residuel prend un petit jeu de valeurs RECURRENTES, LES MEMES d'un film a l'autre (557 / 707 / 773 / 881 / 950 / 1 104) | R3 §9 « Decouvertes » |
| Le temoin `ti=37` (30/31 composants portes) echoue AUTANT que la cible `ti=11` : 33,2 % de masques hors grammaire, 34,3 % de traversees abouties | `PLAN_R4_OBJECTIFS_VIVANTS_TI11.md` §Phase 4.1, instrument `LevelUp-wt-ti11/.../filmdec/sonde_ti11_mur_test.go` (LECTURE SEULE depuis ici) |
| Les 6 appelants de `TraverseEntity` sont TOUS sur le chemin delta ; les deux lecteurs d'image-cle existants EVITENT la grammaire (ils balaient des motifs de 32 bits) | R4 §Phase 4.2 ; `filmdec/keyframe_loadout.go`, `filmdec/keyframe_ground_weapons.go` |

### 2.2 Ce que le depot sait de l'image-cle (relu ce jour, fichier:ligne)

| fait | piece |
|---|---|
| En-tete d'un record de la table type-2 : `[id:32][field:26][ti:6]` = 64 bits ; `ti` lu a `q+58` ; `gen = id>>30 ∈ {1,2,3}` ; `slot = id & 0x3FFFFFFF` | `filmdec/keyframe_world.go:17-23` |
| `WalkKeyframeWorld` **BALAIE, il ne PARSE pas** : il cherche l'ancre suivante (`kfScanNext`, `:114`) et apprend les largeurs EMPIRIQUEMENT (`width[ti]`, `:180-200`) | `filmdec/keyframe_world.go:153-207` |
| **LE FILTRE FORT du balayeur** : `kfReadBits(buf, q+32, 32) < 50` — il n'accepte une ancre QUE si le mot de 32 bits a `q+32` est `< 50`, c'est-a-dire **si les 26 bits de `field` sont TOUS NULS** | `filmdec/keyframe_world.go:70` |
| Le lecteur de record NEW porte : `R(6) ti` -> etat par defaut -> `R(1)` porte -> masque -> composants -> tail | `filmdec/traverse.go:1008-1049` (`TraverseEntity`) |
| Le masque : `R(1)` ; si 0 -> `R(3)` compte + compte x `R(6)` index ; si 1 -> `R(64)` | `filmdec/traverse.go` (`consumeMask`) — porte de `FUN_1406d7610` |
| Corruption-check per-composant du mode film : `R(1)` ; si 1 `R(32)` sentinelle `0xbcddcba` | `filmdec/traverse.go:1104-1109` (`consumeCorruptionCheck`) |
| Etat par defaut par archetype (vtable[0x60]) : table `defaultStateDeserByTI` — **20 archetypes portes**, `ti=11` = `consumeVersionPrefix`, **`ti=42` ABSENT** | `filmdec/default_state_arch.go:44-64` |
| Positions d'armes au sol a l'image-cle : rendues NON PUBLIABLES par leur propre en-tete | `filmdec/keyframe_ground_weapons.go` |
| Oracle bit-exact du record-loop (buffer runtime, 11 485 o, deterministe entre deux lancements) | `.ai/V7.5/dumps/keyframe_buffer_live.bin` + `kf_capture_sample.txt` |
| `SchemaVersion = 8` | `replay/document.go:75` |

### 2.3 Ce que GHIDRA a montre AVANT d'ecrire ce plan (reconnaissance, adresses relues ce jour)

Instance PID 10104, `HaloInfinite.exe`, GhidraMCP sur `127.0.0.1:8089`. **LECTURE SEULE**
(decompile / xref / read_memory ; aucun rename, aucun script, aucune analyse relancee).

| adresse | ce que la decompile montre |
|---|---|
| `FUN_1406cd128` | boucle de records. `cVar3 = *(param_1+0x12)` (porte image-cle) -> `uVar14 = 2` -> la boucle **BREAK immediatement** : elle NE decode PAS l'image-cle. Deux variantes selon `DAT_14474cd78` : bufferisee (dispatch `FUN_141f86704` pour NEW, `FUN_141f86b58` pour DELTA) ou directe (`FUN_1406cbaa0`) |
| `FUN_141f86704` | deser NEW, variante bufferisee. `R(6) ti` -> `desc = *(*(p1+0x18)+8+ti*8)` -> `vtable[0x20]`/`vtable[0x10]` (tailles, 0 bit) -> **`vtable[0x60](desc, n, dst, reader, 1)` = etat par defaut** -> `vtable[0x88](...)` (0 bit, remplit le MASQUE PAR DEFAUT) -> `vtable[0x30]()` (0 bit) -> mode film : `R(1)` -> si (masqueParDefaut != 0 OU porte) : `R(1)` si pas deja lu, puis **`FUN_14076cb60`** |
| `FUN_1408f1aa4` | deser NEW, variante directe (celle qu'appelle `FUN_1406cbaa0`) : **MEME grammaire**, memes slots de vtable, meme ordre. Les deux lecteurs de record NEW du jeu sont donc d'accord |
| `FUN_14076cb60` | boucle de composants : `FUN_1406d7610` (masque) puis, pour chaque composant PRESENT, `vtable[0x28](desc, reader, ctx, &pred, n)` + le corruption-check du mode film. **L'index de bit de masque est `i - skipped`** — un compteur de composants filtres decale le masque (filtre actif seulement hors mode film, `DAT_144c232e1`) |
| `FUN_1406d7610` | masque : `R(1)` ; si 0 -> `R(3)` compte + compte x `R(6)` ; si 1 -> `R(64)`. Conforme au port |
| `FUN_1406cbaa0` | dispatch de record. Prologue MODE FILM avant le deser NEW : `R(1)` ; si 1 -> `R(8)` (`:146-149`). Meme prologue dans `FUN_1406cd128` avant `FUN_141f86704` |

**Ce que cette reconnaissance etablit deja, et qui est un resultat en soi :** les deux
deserialiseurs de record NEW du jeu portent la MEME grammaire que `TraverseEntity`. La
grammaire portee n'est donc pas fausse « en general ». **Le suspect n'est pas la grammaire,
c'est l'ORACLE contre lequel R3 l'a mesuree** — voir H1 ci-dessous.

---

## 3. Hypotheses, ORDONNEES par cout croissant

**H1 — L'ORACLE DE R3 EST CONTAMINE PAR LE FILTRE FORT DU BALAYEUR (cout : TRES FAIBLE).**
`WalkKeyframeWorld` n'accepte une ancre que si les 26 bits de `field` sont NULS
(`keyframe_world.go:70`). Si un seul record d'un film porte un `field` non nul, le balayeur
le SAUTE, et le « record suivant » qu'il rend a R3 n'est pas le voisin : c'est le voisin du
voisin. La marche de R3 atterrirait alors TROP TOT d'exactement UNE LONGUEUR DE RECORD —
ce qui explique a la fois (a) le sens de l'ecart (toujours trop tot, jamais trop loin),
(b) sa taille (557 a 1 104 bits = l'ordre de grandeur d'un record d'image-cle), et (c) sa
RECURRENCE d'un film a l'autre (les memes archetypes intercales rendent les memes longueurs).
**Test, et il est gratuit** : la marche atterrit-elle sur une ancre VALIDE-RELACHEE (gen
∈ {1,2,3}, slot croissant, `ti < 50` a `+58`, SANS exiger `field == 0`) ? Puis : en
CHAINANT la marche depuis ce point, retombe-t-on sur le `want` de R3 ?

**H2 — UN BLOC MANQUANT DE TAILLE STRUCTUREE (cout : moyen).** Si H1 tombe, l'ecart est
interne au record : quelque chose se lit ENTRE la fin des composants et le record suivant
(tail plus long que 1 bit, alignement, bloc de queue), ou AVANT le masque (le prologue mode
film `R(1)[+R(8)]` vu dans `FUN_1406cbaa0` / `FUN_1406cd128`, jamais porte). Test : la
distribution des ecarts modulo 8 / 32 / 64, et un balayage du tail de 0 a 64 bits.

**H3 — LE MASQUE OU LA PORTE DIFFERENT EN IMAGE-CLE (cout : moyen).** Le masque par defaut
rendu par `vtable[0x88]` (0 bit, RAM) commande la lecture de la porte dans les deux
deserialiseurs. Un archetype dont le masque par defaut est NON NUL lit la porte meme quand
le flux ne la porte pas. Test : porter `vtable[0x88]` par archetype, ou mesurer l'effet
d'une porte forcee.

**H4 — L'ETAT PAR DEFAUT DE `ti=42` (cout : moyen, INDEPENDANT des precedentes).**
`defaultStateDeserByTI` n'a pas d'entree 42. La chaine de resolution est ECRITE et
REPRODUCTIBLE (`default_state_arch.go:5-18`) : registrar `FUN_140e453b4` ->
`FUN_140e45fc4(world, ti, &desc)` -> xref WRITE sur le descripteur -> `LEA` precedent ->
vtable -> `*(vtable+0x60)`. C'est du travail de decompile, pas une hypothese.

---

## 4. Corpus — FERME

Racine, LECTURE SEULE : `C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/`.
Presence des 6 films verifiee sur disque le 2026-08-17 (nombre de chunks entre parentheses).

| film | (chunks) | role |
|---|---|---|
| `000d5950` | 49 | principal — golden du rejeu, oracle R3 (415 records `ti=37` bornes) |
| `00502e52` | 30 | principal — oracle R3 (408 bornes) |
| `07aa428d` | 28 | principal — oracle R3 (403 bornes) |
| `64e8adfa` | 45 | CTF Catalyst — cible `ti=11` (201 records, 10 slots, mesure R4) |
| `530820e5` | 27 | CTF Catalyst — second temoin `ti=11` (115 records, 5 slots) |
| `0014603f` | 22 | TEMOIN NEGATIF (`ti=11` absent, `i48` jamais au masque) |

## 4bis. Blockers connus et leur contournement

| blocker | nature | contournement |
|---|---|---|
| Les films vivent dans le depot PRINCIPAL, absent du worktree | chemin | lecture seule par chemin absolu ; aucune ecriture, aucun lien |
| Un seul decodage `filmdec` par process (hooks + bascules GLOBALES) | technique | un seul test a la fois, bascules restaurees en `defer` (patron `ScanFilmEquipmentIdentity`) |
| Cache Go partage corrompu par deux `go` concurrents (autres lots en parallele) | technique | `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfgram/.gocache` (ajoute a l'`info/exclude` du worktree, jamais au `.gitignore` versionne) + UNE commande `go` a la fois |
| Un paquet exigeant CGO (DuckDB, `himap`) sur le chemin de build | technique | `CGO_ENABLED=0` sur `filmdec`/`replay` ; si un instrument exige CGO il porte l'etiquette `gamefiles` (patron impose par R3, sans quoi trois jobs de CI cassent) |
| Ghidra est le programme de l'utilisateur, ouvert et en cours d'usage | ressource partagee | LECTURE SEULE stricte : `decompile_function`, `get_xrefs_*`, `read_memory`, `disassemble_bytes`. Aucun rename, aucun script, aucune analyse relancee, aucune sauvegarde |
| `ti=11` a **0/34 composants portes** | perimetre | la phase 4 en porte le MINIMUM (i5, i12, i13, i14) et statue les autres |

---

## 5. Phases

Une phase n'ouvre pas tant que la precedente n'est pas CLOSE : gate passe (commandes exactes,
sorties collees au journal §10), tous les items statues, plan mis a jour, commit sur
`wt/kf-grammaire`, `git push -u origin wt/kf-grammaire`, point d'etape.

| phase | effort | livrable seule ? |
|---|---|---|
| 1 LIRE la grammaire dans le code du jeu | moyen | oui (le journal RE est un livrable) |
| 2 PORTER et PROUVER sur l'oracle R3 | lourd, risque eleve | oui (un negatif mesure est un livrable) |
| 3 `ti=42` positions | moyen | oui |
| 4 `ti=11` progression | moyen | oui |
| 5 PUBLIER | rapide | non (depend de 3 et 4) |
| 6 CLORE | rapide | non |

### Phase 1 — LIRE : la grammaire du corps de record d'image-cle, dans Ghidra — CLOSE le 2026-08-17

- [x] 1.1 Grammaire des deux deserialiseurs de record NEW ecrite ligne a ligne : ils sont
      IDENTIQUES (`FUN_141f86704` bufferise, `FUN_1408f1aa4` direct), meme ordre, memes slots
      de vtable, un SEUL appel qui lit le flux avant la boucle (`vtable[0x60]`). Boucle de
      composants (`FUN_14076cb60`) et masque (`FUN_1406d7610`) portes au journal RE avec la
      liste des appels a 0 bit (`vtable[0x10]/0x20/0x30/0x48/0x88`).
- [x] 1.2 Les differences sont ECRITES et elles appartiennent toutes au CADRE, pas au CORPS :
      `R(32)` de tete d'iteration en mode film (`FUN_1406cd128`), prologue `R(1)[+R(8)]`
      avant NEW et DEL (`FUN_1406cd128` et `FUN_1406cbaa0:146-149`), et la porte d'image-cle
      `*(param_1+0x12)` qui DESACTIVE la boucle de records. Le corps, lui, est le meme.
- [x] 1.3 **Hypothese n°1 du brief REFUTEE PAR LECTURE.** Le chemin DELTA
      (`FUN_141f86b58`) appelle **la meme `FUN_14076cb60`** que les deux lecteurs NEW, avec un
      contexte de meme forme, et il n'existe **aucun second site d'appel de composant** :
      chaque composant present passe par `vtable[0x28]` dans les trois chemins. Il n'y a pas
      de « deser feuille etat-complet » distinct d'un « wrapper delta ».
- [x] 1.4 **Etat par defaut de `ti=42` RESOLU, bit-exact.** Chaine rejouee : `FUN_140e453b4`
      -> `FUN_140e45fc4(world, 0x2a, &PTR_PTR_144701780)` @`0x140e4578f` ; xref [WRITE]
      `0x144701780` -> `FUN_1403721d0` : vtable = `0x1436fd790` ; `*(vtable+0x60)` =
      `FUN_1407f0c68` = `V ; FUN_1407f2224 (== consumeDefaultStateTI36) ; R(12) ; R(7) ;
      FUN_1407f2494 ; ECS_ReadEntityRefIndex5`. Largeur 7 figee par
      `MOV dword [RSP+0x20],7` @`0x1407f0d30`. Sous-bloc `FUN_1407f2494` ecrit au journal.
- [x] 1.5 Section « IMAGE-CLE » ecrite dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`
      (5 sous-sections, dont une §5 « ce qui reste NON resolu »). Aucune adresse en dur cote Go.

**Gate 1 : PASSE le 2026-08-17.** Commandes et sorties exactes :

```
grep -ci "image-cle" .ai/V7.5/killweapon/WALK_PORT_NOTES.md                              -> 7
grep -c "FUN_14076cb60\|FUN_1406d7610\|FUN_1408f1aa4" .ai/V7.5/killweapon/WALK_PORT_NOTES.md -> 7
grep -o "FUN_1[0-9a-f]*\|0x14[0-9a-f]*" .ai/V7.5/killweapon/WALK_PORT_NOTES.md | sort -u | wc -l -> 62
```

Critere (section presente, >= 6 adresses, reponses ecrites a 1.2/1.3/1.4) : REMPLI —
62 adresses distinctes citees.

**CE QUE LA PHASE 1 ETABLIT, et ce n'est pas ce que les deux lots precedents supposaient.**

1. **La grammaire portee par `TraverseEntity` est CELLE DU JEU.** Les deux lecteurs de record
   NEW et le lecteur DELTA sont d'accord ; il n'y a pas de variante « image-cle » du corps.
   Le verdict de R3 (« la grammaire du record d'image-cle n'est celle d'aucun archetype »)
   ne peut donc pas s'expliquer par une grammaire de corps differente.
2. **Le suspect designe devient l'ORACLE.** `WalkKeyframeWorld` n'accepte une ancre que si
   les 26 bits de `field` sont nuls (`keyframe_world.go:70`). Tout record dont le `field`
   n'est pas nul est SAUTE, et le « record suivant » rendu a R3 n'est alors pas le voisin.
   C'est l'hypothese H1 du §3, et la phase 2 la mesure.
3. **`ti=42` n'est plus un blocage de reverse** : son etat par defaut est resolu et se porte
   avec des primitives DEJA presentes dans le depot.

### Phase 2 — PORTER et PROUVER : le walker DETERMINISTE d'image-cle — CLOSE le 2026-08-17, **NEGATIF MESURE**

- [x] 2.1 `filmdec/keyframe_record_walk.go` ECRIT : `readKeyframeHeader` (l'en-tete de 64 bits
      SANS le filtre fort), `WalkKeyframeRecords` (parse deterministe avec cause d'arret
      nommee), `ChainKeyframeRecords` (chainage vers une frontiere connue) et
      `WalkKeyframeBody` (les huit lectures possibles du corps). Il REUTILISE
      `TraverseEntity`, `traverseComponentLoop`, `consumeMask` et les deserialiseurs d'etat
      par defaut de production — aucun deser recopie.
- [x] 2.2 **H1 TESTEE ET REFUTEE.** Sur les 1 226 records `ti=37` bornes (415 + 408 + 403, le
      denominateur EXACT de R3, retrouve a l'unite pres) : **0 atterrissage direct, 0 par
      chainage**. La cause est publiee et elle tue l'hypothese : sur `000d5950`, **382 des
      415 marches n'atterrissent meme pas sur un en-tete valide** (`arret en-tete-invalide`),
      24 atterrissent sur un en-tete mais a slot non croissant, 9 en fin de payload.
      **ZERO record intercale traverse, donc zero `field26` non nul rencontre** : le balayeur
      ne saute rien, ce n'est pas lui le coupable.
- [x] 2.3 **C1 ECHOUE, et H2/H3 sont refutees avec lui.** Trois balayages independants :
      (a) **decalage du corps** — pour chaque record, le lecteur de record NEW pose a chacun
      des 128 decalages possibles depuis le debut du record : **0 marche exacte** sur les
      415 records `ti=37` et les 2 008 records `ti=38` de `000d5950`. Le decalage 58 est le
      SEUL ou le `typeIndex` relu est correct (415/415 et 2008/2008) — l'en-tete de 64 bits
      est donc CONFIRME par une chaine independante, et le corps n'est nulle part.
      (b) **lecture du corps** — les huit combinaisons (etat par defaut x porte x masque) x
      corruption-check : le meilleur resultat sur `ti=37` est **7 marches exactes sur 403**
      (1,7 %), sur `ti=38` **95 sur 5 311** (1,8 %). Aucune ne s'approche du seuil.
      (c) **grammaire de record NEW** (matrice de R3, 8 combinaisons) : au mieux **1 marche
      sur 1 226**, cote a cote avec les 2 / 1 226 de R3 — reproduction independante du meme
      negatif.
- [x] 2.4 **C2 ECHOUE** sur le second archetype, choisi par sa FREQUENCE : `ti=38` est le plus
      frequent des tables d'image-cle (2 013 / 7 825 records sur `000d5950`). Denominateur
      cumule **9 460 records bornes** (2 008 + 2 141 + 5 311). Meilleur taux 1,8 %.
- [x] 2.5 **C3 ECHOUE, et c'est la mesure la plus nette du lot** : le walker deterministe
      parse **exactement UN record par image-cle** puis s'arrete, sur les 26 images-cles de
      `000d5950` et pour les huit combinaisons — **26 records de l'oracle retrouves sur
      7 825 (0,3 %)**, cause d'arret `en-tete-invalide` dans 26 payloads sur 26. La marche du
      TOUT PREMIER corps ne retombe deja pas sur une frontiere.
- [x] 2.6 Instrument versionne `filmdec/keyframe_record_walk_test.go` (4 tests), garde
      `KF_GRAM_FILM`, **4 SKIP verifies sans la variable**, lecture seule, aucune ecriture
      disque.

**GATE 2 : PASSE PAR LE NEGATIF** (le plan prevoyait les deux issues). Commandes et sorties :

```
CGO_ENABLED=0 go build ./internal/analysis/...                       BUILD_EXIT=0
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/                  VET_EXIT=0
gofmt -l internal/analysis/filmdec/                                  (vide)
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ ./internal/analysis/replay/
    ok levelup/go-api/internal/analysis/filmdec  34.486s
    ok levelup/go-api/internal/analysis/replay   23.990s             TEST_EXIT=0
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ -run '^TestKFGram' -v
    4 SKIP (garde OK)
CGO_ENABLED=0 KF_GRAM_FILM=<film> go test ./internal/analysis/filmdec/ \
    -run '^TestKFGram(Chain|Offset|Variant|Global)$' -timeout 30m -v     PASS sur 3 films
```

**LE RESULTAT DE FOND — LE CORPS D'UN RECORD D'IMAGE-CLE N'EST PAS UN RECORD NEW.**

C'est un fait NEUF, et il contredit une croyance ecrite du depot (« le keyframe utilise la
MEME grammaire de records que les deltas », `HANDOFF_KEYFRAME_LIVE_CAPTURE.md`). Ce que la
phase 2 etablit, avec ses denominateurs :

1. **L'EN-TETE est confirme, par une chaine independante** : 64 bits, `typeIndex` a `+58`,
   relu correct sur 415/415 (`ti=37`) et 2 008/2 008 (`ti=38`), et aucun autre des
   128 decalages ne fait mieux que le hasard. Ce n'est donc pas l'ancrage qui est faux.
2. **LE CORPS N'EST PAS ATTEIGNABLE par le lecteur de record NEW** : 128 decalages x 16
   lectures de corps x 3 films, **jamais plus de 1,8 %** de marches exactes, et le walker
   global s'arrete au premier record dans 26 payloads sur 26.
3. **L'ORACLE DE R3 EST DISCULPE.** L'ecart residuel n'est pas un record saute : les marches
   n'atterrissent pas sur des en-tetes du tout (382/415), et zero record intercale n'a pu
   etre traverse. Le filtre `field26 == 0` de `keyframe_world.go:70` n'est pas en cause.
4. **La longueur reelle des records est FORTEMENT quantifiee**, ce qui reste le meilleur
   indice pour la suite : `ti=38` ne prend que **39 valeurs distinctes sur 2 008 records**
   (dominantes 827 x 396, 870 x 243, 851 x 231, 846 x 204, 859 x 174) et `ti=37` 106 valeurs
   sur 415 (dominante 888 x 100). Le lecteur de record NEW n'en consomme qu'environ 40 %.
   Une serialisation d'ETAT COMPLET par composant reste la piste la plus economique — mais
   elle ne se lit pas par la boucle `FUN_14076cb60`, qui a ete essayee et mesuree.
5. **Ce qui manque, nomme** : le CONSOMMATEUR du payload type-2 dans le jeu. La phase 1 a
   etabli que la boucle de records (`FUN_1406cd128`) est DESACTIVEE quand la porte
   d'image-cle `*(param_1+0x12)` est mise, et que le chemin d'image-cle passe par
   `FUN_142f2913c` (baseline-emit) qui draine une file par-entite — donc un bitstream
   RECONSTRUIT, pas le payload du film. La transformation « payload type-2 -> file
   par-entite » n'est identifiee nulle part, et c'est ELLE qu'il faut decompiler.

### Phase 3 — `ti=42` : la position des armes au sol a l'image-cle — PARTIELLE, le reste `[!]`

La condition d'ouverture ecrite (« C1 passe ») n'est PAS remplie. L'item 3.1 est neanmoins
EXECUTE, parce qu'il ne depend pas de C1 : l'etat par defaut d'un archetype sert au lecteur
de record NEW du chemin DELTA, ou `TraverseEntity` est reellement appele. Les items 3.2 a 3.4
sont statues `[!]` : ils lisent le corps d'un record d'image-cle, et la phase 2 vient de
montrer qu'on ne sait pas le lire.

- [x] 3.1 **Etat par defaut de `ti=42` PORTE ET INSCRIT.** `filmdec/default_state_ti42.go`
      (fichier NEUF, a moi) porte `FUN_1407f0c68` feuille par feuille, avec la chaine de
      resolution et les adresses en commentaire ; **UNE SEULE ligne ajoutee** au fichier
      partage `default_state_arch.go` (`42: consumeDefaultStateTI42`). Build, vet, gofmt et
      les suites `filmdec` + `replay` restent verts. **Caveat ECRIT dans le fichier** : aucun
      oracle ne le valide, le seul disponible ayant ete refute en phase 2 ; il entre dans la
      table pour la meme raison que les vingt autres — toutes ses largeurs sont etablies
      statiquement.
- [!] 3.2 `filmdec/keyframe_ti42_pos.go` — **non cree**. Il lirait la position dans le corps
      d'un record d'image-cle ; ce corps n'est pas decodable aujourd'hui (phase 2). Un fichier
      qui rendrait des positions non controlables serait exactement l'erreur que le lot des
      armes au sol a payee le 2026-08-12.
- [!] 3.3 C4 (emprise + bande fantome) — sans objet : aucune position a encadrer.
- [!] 3.4 Verdict sur les 55 % de positions fantomes du 2026-08-12 — **inchange**. La
      condition de reprise du registre (« default-state de `ti=42` resolu ») est desormais
      REMPLIE, mais elle n'etait pas suffisante : la condition reelle est la grammaire du
      corps d'image-cle. Le registre est amende en ce sens (§11).

**Gate 3 : PASSE SUR SON PERIMETRE REEL.**

```
CGO_ENABLED=0 go build ./internal/analysis/...                       BUILD_EXIT=0
CGO_ENABLED=0 go vet   ./internal/analysis/filmdec/                  VET_EXIT=0
gofmt -l internal/analysis/filmdec/                                  (vide)
CGO_ENABLED=0 go test  ./internal/analysis/filmdec/ ./internal/analysis/replay/   ok / ok
```

### Phase 4 — `ti=11` : l'objectif vivant — `[!]` NON OUVERTE, DEPENDANCE MORTE

La condition d'ouverture (« C1 passe ») n'est pas remplie, et la dependance est TOTALE :
`ti=11` n'est atteignable qu'au record d'image-cle (la voie DELTA a ete refutee par R4 —
`matchWorldObjectRecord` ne reconnait pas ces records, rapport reel/fantome 0,73x et 0,37x).
Porter `i5` / `i12` / `i13` / `i14` sans savoir ou ils commencent produirait des valeurs
inverifiables.

- [!] 4.1 Portage des composants `ti=11` — non fait (aucun point d'entree fiable).
- [!] 4.2 `filmdec/keyframe_ti11_objectifs.go` — non cree (code mort, anti-pattern n°1).
- [!] 4.3 C5 — sans objet. **L'oracle reste intact et disponible**
      (`objectiveevents.IdentifiedEvent`) : c'est l'objet a confronter qui manque, pas le
      temoin. La reprise sera donc peu couteuse une fois le corps decode.
- [!] 4.4 Verdict — celui de la phase 2, avec ses chiffres.

**Gate 4 : SANS OBJET.** La phase n'a pas ete ouverte.

### Phase 5 — PUBLIER dans l'artefact — `[!]` RIEN A PUBLIER

La regle de non-publication du §1.1 est appliquee telle qu'elle a ete ecrite : aucune donnee
n'entre dans l'artefact tant que le controle de sa phase n'est pas passe. Aucun ne l'est.

- [!] 5.1 Assemblage `replay/` — non cree.
- [!] 5.2 Champ optionnel sur `ReplayDocument` — non ajoute. `SchemaVersion` reste a **8**.
- [!] 5.3 Contrat OpenAPI / `generated.ts` / normalisation web — sans objet, aucun champ ne
      traverse. Verifie : **aucun fichier de `internal/analysis/replay/` ni de `apps/web/`
      n'est touche par ce lot.**
- [x] 5.4 **Note de bosse : ce lot ne demande AUCUNE bosse de `SchemaVersion`.** Il n'ajoute
      aucun champ au document. (Information utile au superviseur a la fusion : R3 peut en
      demander une pour son compte, R4 n'en demande pas, R5 non plus.)

**Gate 5 : SANS OBJET.** Aucune modification de l'artefact ni du document.

### Phase 6 — CLORE — CLOSE le 2026-08-17

- [x] 6.1 Toutes les cases des phases 1 a 5 sont statuees, aucune vide.
- [x] 6.2 Lignes de registre redigees en une seule fois (§11), y compris les DEUX lignes a
      amender (report `ti=42` du 2026-08-12, ligne `ti=11` de R4) et la decouverte 2 de R3.
- [x] 6.3 Entree `thought_log.md` redigee (§12) et remise au superviseur — ce lot n'ecrit PAS
      dans le journal.
- [x] 6.4 Rapport final rendu, avec denominateurs, gates executes et ce qui reste.
## 6. Decisions produit — TRANCHEES avant execution

1. **L'UI n'est pas dans ce lot.** On publie la donnee, ou on ne publie rien. Aucun fichier
   `apps/web/` touche sauf la normalisation d'un champ qui traverse (item 5.3).
2. **Aucun nom, aucun libelle en dur cote Go.** Un type non nomme garde son identifiant. Si
   des noms sont poses, ils entrent dans le TOML du titre (`replay_labels.toml` /
   `objective_roles.toml`), jamais dans le decodeur — ADR 0011.
3. **Aucune comparaison `slug == "..."`** (ratchet `no_slug_comparison_test.go`).
   Reserve honnete heritee de R4 : `filmdec` est aujourd'hui du RE Halo Infinite
   (`keyframe_world.go:19-23` le dit) ; ce lot n'aggrave pas cette dette et ne la traite pas.
4. **C'est le DESERIALISEUR qui publie**, jamais un second lecteur pose a cote de lui.
5. **Rien n'est publie sans son temoin de controle** (fantome / temoin negatif obligatoire).
6. **Offline-pur** : aucune capture runtime, aucun Cheat Engine. Ghidra en LECTURE SEULE.
7. **Pas de CGO** sur `filmdec`/`replay` ; un instrument qui exige CGO porte `gamefiles`.
8. **Un negatif mesure est un livrable**, publie avec ses denominateurs et ses temoins.

## 6bis. Conformite architecture (grille `plan-review`, passee le 2026-08-17)

| couche | ce que ce lot y met |
|---|---|
| `internal/analysis/filmdec/` | le walker deterministe et les instruments de mesure — algorithme pur, aucun acces DB/HTTP |
| `internal/analysis/replay/` | l'assemblage du calque (phase 5) — pur, sans I/O |
| `internal/service/`, `internal/port/`, `internal/api/handlers/` | **RIEN** : le champ voyage sur `ReplayDocument`, deja construit et servi |
| `platform/duckdb/`, `persist/` | **RIEN** : aucune lecture ni ecriture DuckDB |
| `apps/web/` | rien, sauf la normalisation d'un champ qui traverse (5.3) |

- Seuils : fichier <= 500 L, fonction <= 80 L, <= 5 parametres. Les grandeurs d'une marche
  sont regroupees en struct plutot qu'ajoutees en parametres (patron `equipIDWalk`).
- Logging : les instruments sont des TESTS, ils rendent par `t.Logf` (patron du depot).
  Le code d'assemblage de `replay/` est PUR : il rend sa couverture (`LayerCoverage`), il ne
  journalise pas. **Aucun `fmt.Println` / `log.Printf`.** Aucune erreur avalee en silence.
- Tests : instruments gardes par variable d'environnement (sautent en CI) + un test unitaire
  PUR sans I/O pour tout assemblage de `replay/`.
- i18n / couleurs / query keys / routes : sans objet (aucun rendu dans ce lot).

## 7. Contrat d'interface avec les lots paralleles

R1 est fusionne ; R3 bis tourne dans le worktree principal ; l'habillage est en fusion.

1. **Je ne CREE que mes fichiers** : `filmdec/keyframe_record_walk*.go`,
   `filmdec/keyframe_ti42_pos*.go`, `filmdec/keyframe_ti11_objectifs*.go`, le fichier
   d'assemblage `replay/` de la phase 5, ce plan, et la section « image-cle » de
   `WALK_PORT_NOTES.md`.
   **AMENDEMENT du 2026-08-17, ecrit au moment ou il a ete pris** : `filmdec/
   default_state_ti42.go` s'ajoute a cette liste. L'item 3.1 porte l'etat par defaut de
   `ti=42` ; le loger dans `keyframe_ti42_pos.go` aurait ete trompeur (ce n'est pas de la
   position) et dans `keyframe_record_walk.go` hors sujet. Le fichier partage
   `default_state_arch.go` ne recoit qu'UNE ligne d'enregistrement, conformement au point 2.
   **FICHIERS PARTAGES REELLEMENT TOUCHES PAR CE LOT, liste exhaustive** :
   `filmdec/default_state_arch.go`, une ligne (`42: consumeDefaultStateTI42`). Rien d'autre.
2. **Fichiers PARTAGES du decodeur** (`traverse.go`, `default_state*.go`, `keyframe_world.go`,
   `replay/build.go`, `replay/document.go`, `replay/coverage.go`) : **une ligne
   d'enregistrement chacun au maximum**, jamais de reecriture, jamais de reindentation,
   jamais de reordonnancement.
3. **AUCUNE bosse de `SchemaVersion`.** La bosse unique est faite par le superviseur a la
   fusion (item 5.4 dit laquelle et pourquoi).
4. **Interdits** : aucun run de masse, aucune re-cuisson d'artefact publie sous `data/`,
   aucune ecriture DuckDB, aucune ecriture hors du worktree `LevelUp-wt-kfgram`. Les films et
   les autres worktrees sont lus en LECTURE SEULE.
5. **Cache Go isole** : `GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfgram/.gocache` sur
   TOUTE commande `go`, **une seule a la fois**.
6. Commits par phase sur `wt/kf-grammaire` (`feat(v7.5-rejeu-kf):` / `mesure(...)` /
   `docs(...)`, terminaison `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`),
   hooks ACTIFS (jamais `--no-verify`), `git push -u origin wt/kf-grammaire` apres chaque
   phase close. Jamais `git stash`, jamais `main`, jamais de merge.
7. Ce lot n'ecrit PAS dans `.ai/thought_log.md` (superviseur) ; l'entree est redigee au §12.

## 8. Statuts d'item et cloture

`[x]` fait et verifie · `[~]` couvert ailleurs (avec la reference) · `[!]` non traite (avec
la justification ecrite). **Aucune case vide a la cloture d'une phase.** Clore une phase =
gate passe + items statues + plan a jour + commit + push + point d'etape.

## 9. Decouvertes — a consigner, NE PAS traiter dans ce lot

- **(phase 2) La longueur reelle d'un record d'image-cle est FORTEMENT quantifiee, et c'est
  un signal gratuit.** `ti=38` : **39 valeurs distinctes sur 2 008 records** (827 bits
  x 396, 870 x 243, 851 x 231, 846 x 204, 859 x 174, 886 x 125, 948 x 100, **1 536 x 100**
  — cette derniere vaut exactement 192 octets). `ti=37` : 106 valeurs sur 415, dominante
  888 x 100. Une entite dont la longueur de record ne prend qu'une poignee de valeurs est
  une entite dont l'etat est ecrit de facon quasi FIXE. NE PAS traiter ici, mais c'est le
  meilleur point d'entree pour la prochaine passe : mesurer, pour un archetype donne, la
  correspondance longueur <-> contenu.
- **(phase 2) La marche du record NEW ne consomme qu'environ 40 % de la longueur reelle.**
  Sur les 100 records `ti=37` de longueur 888 bits, la variante « etat par defaut + porte +
  masque » s'arrete 557 bits trop tot (soit 331 bits consommes). Un facteur 2,5, stable.
- **(phase 1) `vtable[0x88]` (le MASQUE PAR DEFAUT) n'est porte pour aucun archetype.** Il
  ne lit aucun bit, mais il commande la lecture de la porte `R(1)` dans les deux lecteurs de
  record NEW : un archetype dont le masque par defaut est non nul lit la porte meme quand le
  flux ne la porte pas. Sans effet mesure ici (la porte a ete probee dans les deux sens),
  mais c'est une inconnue restante du chemin DELTA. NE PAS traiter ici.
- **(phase 1) Un prologue de mode film jamais porte** : `FUN_1406cd128` lit `R(32)` en tete
  de chaque iteration de record quand le mode film est actif, et `R(1)[+R(8)]` avant les
  branches NEW et DEL. Le meme `R(1)[+R(8)]` figure dans `FUN_1406cbaa0` avant le deser NEW.
  Le decodeur du depot n'en porte aucun. Sans effet sur l'image-cle (qui ne passe pas par
  cette boucle), mais a verifier sur le chemin DELTA. NE PAS traiter ici.
- **(phase 1) `FUN_14076cb60` decale l'index de bit du masque d'un compteur de composants
  SAUTES** (`i - sautes`). Le saut n'a lieu que hors mode film (`DAT_144c232e1`), donc sans
  effet ici — mais le port Go (`traverseComponentLoopFrom`) utilise `i` nu, ce qui serait
  faux si le filtre s'activait. NE PAS traiter ici.

## 10. Journal d'execution

**2026-08-17 — Phase 1 CLOSE.** Lecture Ghidra read-only (7 fonctions decompilees, une vtable
lue octet a octet, un immediat verifie au desassemblage, un site d'appel verifie par
`get_assembly_context`). Trois resultats : (a) les deux lecteurs de record NEW du jeu portent
la MEME grammaire, celle que `TraverseEntity` porte deja ; (b) l'hypothese « deser feuille
`+0x28` en image-cle contre wrapper en delta » est REFUTEE par lecture — le chemin delta
appelle la meme `FUN_14076cb60` et il n'existe pas de second site d'appel de composant ;
(c) l'etat par defaut de `ti=42` est RESOLU bit-exact (`FUN_1407f0c68`). Gate 1 passe
(3 greps, sorties ci-dessus). Journal RE : section « IMAGE-CLE » de `WALK_PORT_NOTES.md`.
Commit `docs(v7.5-rejeu-kf)` sur `wt/kf-grammaire`.

**2026-08-17 — Phase 2 CLOSE PAR LE NEGATIF.** Walker deterministe ecrit
(`filmdec/keyframe_record_walk.go`) + instrument a 4 tests garde par `KF_GRAM_FILM`. Trois
balayages independants sur 3 films : decalage du corps (128 positions), lecture du corps
(8 variantes x corruption-check), grammaire de record NEW (matrice R3). **Aucun ne depasse
1,8 % de marches bit-exactes** ; le walker global s'arrete au PREMIER record dans 26 payloads
sur 26. H1 (oracle contamine par le filtre du balayeur) REFUTEE : zero record intercale
traverse, 382 marches sur 415 n'atterrissent meme pas sur un en-tete. L'en-tete de 64 bits,
lui, est CONFIRME par une chaine independante (decalage 58 seul valide, 415/415 et
2 008/2 008). Gate 2 passe par le negatif. Commit `mesure(v7.5-rejeu-kf)`.

**2026-08-17 — Phase 3 PARTIELLE, phases 4/5 `[!]`, phase 6 CLOSE.** L'etat par defaut de
`ti=42` est porte et inscrit (`filmdec/default_state_ti42.go` + une ligne dans
`default_state_arch.go`) : build, vet, gofmt et les suites `filmdec` + `replay` verts. Le
reste de la phase 3 et toute la phase 4 dependent du corps d'image-cle, non decodable :
statues `[!]` avec leur justification. Rien n'est publie, `SchemaVersion` reste a 8, aucune
bosse demandee au superviseur.

## 11. Lignes de registre proposees

A verser en une seule fois dans `.ai/V7.5/REGISTRE_REPORTS.md`. **Deux lignes existantes sont
a AMENDER** (elles portent une condition de reprise que ce lot a mesuree comme FAUSSE ou
INSUFFISANTE), et deux lignes sont NEUVES.

| sujet | lot / date | ce qui a ete mesure | condition de reprise |
|---|---|---|---|
| **[NEUF — c'est le verrou central] La grammaire du CORPS d'un record d'IMAGE-CLE (payload type-2)** | lot R5, phases 1-2, 2026-08-17 | **Le corps d'un record d'image-cle N'EST PAS un record NEW**, et c'est mesure trois fois independamment. (1) Par LECTURE du jeu : les deux lecteurs de record NEW (`FUN_141f86704` bufferise, `FUN_1408f1aa4` direct) portent la MEME grammaire, le chemin DELTA (`FUN_141f86b58`) appelle la MEME boucle de composants `FUN_14076cb60`, et il n'existe AUCUN second site d'appel de composant — l'hypothese « deser feuille `+0x28` en image-cle contre wrapper en delta » est donc refutee sans mesure. (2) Par BALAYAGE DU DECALAGE : le lecteur de record NEW pose a chacun des 128 decalages possibles rend **0 marche bit-exacte** sur 415 records `ti=37` et 2 008 records `ti=38` ; le decalage 58 est le SEUL ou le `typeIndex` relu est correct (415/415, 2008/2008) — **l'en-tete `[id:32][field:26][ti:6]` est donc CONFIRME**, ce n'est pas l'ancrage qui est faux. (3) Par BALAYAGE DES LECTURES DE CORPS : 8 combinaisons (etat par defaut x porte x masque) x corruption-check, 3 films, **jamais plus de 1,8 %** (meilleur : 7/403 sur `ti=37`, 95/5 311 sur `ti=38`). Le walker DETERMINISTE parse **1 seul record par image-cle** puis s'arrete : **26 records de l'oracle retrouves sur 7 825 (0,3 %)**, arret `en-tete-invalide` dans 26 payloads sur 26. **L'oracle de R3 est DISCULPE** : zero record intercale traverse, donc le filtre `field26 == 0` de `keyframe_world.go:70` n'explique rien. Denominateurs `ti=37` : 1 226 records bornes (415+408+403), identiques a ceux de R3 | condition de reprise : **decompiler le CONSOMMATEUR du payload type-2**. Ce que la phase 1 a etabli et qui borne la recherche : la boucle de records `FUN_1406cd128` est DESACTIVEE quand la porte d'image-cle `*(param_1+0x12)` est mise ; le chemin d'image-cle passe par `FUN_142f2913c` (baseline-emit) qui draine une file par-entite alimentee par `FUN_142f29538` — donc un bitstream RECONSTRUIT, pas le payload du film. **La transformation « payload type-2 -> file par-entite » n'est identifiee nulle part.** Indice a exploiter en priorite : la longueur reelle des records est fortement quantifiee (`ti=38` : 39 valeurs distinctes sur 2 008 records ; `ti=37` : dominante 888 bits, 100 fois), et le lecteur de record NEW n'en consomme qu'environ 40 %. Reproductible : `KF_GRAM_FILM=<film> go test ./internal/analysis/filmdec/ -run '^TestKFGram' -v` |
| **[NEUF] Etat par defaut de `ti=42` : RESOLU et porte, sans oracle** | lot R5, phase 1.4 + 3.1, 2026-08-17 | Chaine rejouee : `FUN_140e453b4` -> `FUN_140e45fc4(world, 0x2a, &PTR_PTR_144701780)` @`0x140e4578f` ; xref [WRITE] -> `FUN_1403721d0` : vtable `0x1436fd790` ; `*(vtable+0x60)` = `FUN_1407f0c68` = `V ; FUN_1407f2224 (== consumeDefaultStateTI36) ; R(12) ; R(7) ; FUN_1407f2494 ; ECS_ReadEntityRefIndex5`. Largeur 7 figee par `MOV dword [RSP+0x20],7` @`0x1407f0d30`. Porte dans `filmdec/default_state_ti42.go`, inscrit par UNE ligne dans `default_state_arch.go`. Toutes les suites restent vertes | **AUCUN oracle ne le valide** : le seul disponible (marche bit-exacte au record d'image-cle) est refute par la ligne ci-dessus. Le valider exigera soit la grammaire du corps d'image-cle, soit un record NEW `ti=42` observe dans un paquet DELTA avec une frontiere connue |
| **[A AMENDER — ligne « armes au sol `ti=42` », 2026-08-12]** | amendement lot R5, 2026-08-17 | La condition de reprise ecrite le 12/08 (« default-state de `ti=42` resolu ») est **REMPLIE depuis le 2026-08-17** — et elle etait **INSUFFISANTE**. Le vrai verrou est la grammaire du corps d'un record d'image-cle (ligne ci-dessus), pas l'etat par defaut. Les 55 % de positions fantomes restent inchanges ; aucune position n'a ete relue | remplacer la condition par : « grammaire du CORPS d'un record d'image-cle resolue » |
| **[A AMENDER — ligne « objectifs vivants `ti=11` », lot R4, 2026-08-17]** | amendement lot R5, 2026-08-17 | La condition de reprise de R4 (« la grammaire du CORPS d'un record d'IMAGE-CLE ») est CONFIRMEE comme le bon verrou, et **PRECISEE** : ce corps n'est pas un record NEW (128 decalages x 16 lectures x 3 films, jamais plus de 1,8 %), et l'en-tete de 64 bits est confirme. Ce qu'il faut decompiler est nomme : le consommateur du payload type-2 / la transformation vers la file par-entite de `FUN_142f2913c` | inchangee dans son principe, precisee dans sa cible |
| **[A AMENDER — decouverte 2 de R3, « l'ecart residuel est structure »]** | amendement lot R5, 2026-08-17 | La lecture proposee par R3 — « la longueur du record depend du TYPE de l'objet, donc c'est un signal de partition gratuit » — est **MESUREE ET CONFIRMEE, mais elle ne se lit pas sur l'ECART** : c'est la LONGUEUR REELLE du record qui est fortement quantifiee (`ti=38` : 39 valeurs distinctes sur 2 008 ; `ti=37` : 106 sur 415, dominante 888 bits x 100). L'ecart de R3 (557 / 707 / ...) n'est qu'un artefact de la marche trop courte | exploiter la LONGUEUR, pas l'ecart ; et le faire APRES la grammaire du corps, sinon on partitionne du bruit |

## 12. Entree `thought_log.md` proposee

```
## [2026-08-17] Lot R5 — la grammaire du corps d'un record d'image-cle : negatif mesure,
et le verrou change de nom

Statut : Complete (negatif mesure, livrable).
Branche : wt/kf-grammaire (worktree LevelUp-wt-kfgram), 4 commits, pousses.

DECISION TECHNIQUE. Deux lots du meme jour (R3 ti=37, R4 ti=11) avaient conclu que « la
grammaire du corps d'un record d'image-cle n'est resolue nulle part ». R5 l'a attaquee
d'abord par LECTURE du jeu (Ghidra, read-only), ensuite seulement par mesure — l'ordre
inverse de ce qui avait ete fait jusque-la, et c'est ce qui a paye.

RESULTATS OBSERVES.
1. Lecture : les deux lecteurs de record NEW du jeu (FUN_141f86704, FUN_1408f1aa4) portent
   la MEME grammaire, et le chemin DELTA (FUN_141f86b58) appelle la MEME boucle de
   composants FUN_14076cb60. Il n'existe aucun second site d'appel de composant :
   l'hypothese « deser feuille +0x28 en image-cle contre wrapper en delta » est refutee
   sans qu'une seule mesure soit necessaire.
2. Mesure : sur 1 226 records ti=37 bornes (le denominateur exact de R3, retrouve a l'unite
   pres) et 9 460 records ti=38, 128 decalages de corps x 16 lectures de corps x 3 films ne
   rendent JAMAIS plus de 1,8 % de marches bit-exactes. Le walker deterministe parse un
   seul record par image-cle puis s'arrete : 26 records de l'oracle sur 7 825 (0,3 %).
3. L'en-tete de 64 bits [id:32][field:26][ti:6] est CONFIRME par une chaine independante :
   le decalage 58 est le seul ou le typeIndex relu est correct, 415/415 et 2 008/2 008.
4. L'oracle de R3 est disculpe : zero record intercale traverse, 382 marches sur 415
   n'atterrissent meme pas sur un en-tete. Le filtre field26==0 du balayeur n'y est pour
   rien.
5. Acquis positif : l'etat par defaut de ti=42 est RESOLU bit-exact (vtable 0x1436fd790,
   *(vtable+0x60) = FUN_1407f0c68) et porte. Il remplit la condition de reprise ecrite le
   12/08 pour les armes au sol — qui s'avere insuffisante.

CONCLUSION / PROCHAINE ETAPE. Le verrou n'est plus « la grammaire du corps », il est
NOMME et BORNE : le consommateur du payload type-2 du film. La boucle de records est
desactivee quand la porte d'image-cle est mise ; le chemin d'image-cle draine une file
par-entite (FUN_142f2913c), donc un bitstream RECONSTRUIT. La transformation
payload -> file n'est identifiee nulle part : c'est elle qu'il faut decompiler. Indice
gratuit pour la suite : la longueur reelle des records est fortement quantifiee (ti=38 :
39 valeurs distinctes sur 2 008 records), et le lecteur de record NEW n'en consomme
qu'environ 40 %. Rien n'est publie, SchemaVersion reste a 8.
```

## 13. Protocole de reprise de session

1. Relire le skill `plan-execution`, puis ce fichier de haut en bas : les cases disent ou en
   est le lot, le journal §10 dit ce qui a ete mesure.
2. Verifier la branche : `git -C C:/Users/Guillaume/Projects/LevelUp-wt-kfgram branch
   --show-current` doit rendre `wt/kf-grammaire`.
3. Relire §7 (contrat d'interface) AVANT de toucher un fichier partage du decodeur.
4. **Verifier sur pieces** avant de coder ET avant de cocher : les references de §2 sont
   datees du 2026-08-17, et trois autres lots travaillent en parallele.
5. Reprendre a la premiere case non statuee de la phase courante. Ne pas re-decider ce qui
   est deja tranche (§6 : decisions fermes).
