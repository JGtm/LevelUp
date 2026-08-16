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

### Phase 1 — LIRE : la grammaire du corps de record d'image-cle, dans Ghidra

- [ ] 1.1 Decompiler et ECRIRE la grammaire des deux deserialiseurs de record NEW
      (`FUN_141f86704`, `FUN_1408f1aa4`), de la boucle de composants (`FUN_14076cb60`), du
      masque (`FUN_1406d7610`) et du dispatch (`FUN_1406cbaa0`) : ordre des lectures, slots
      de vtable appeles, lesquels touchent le bitreader et lesquels non.
- [ ] 1.2 Etablir CE QUI DIFFERE entre le chemin IMAGE-CLE et le chemin DELTA : le prologue
      de mode film (`R(1)[+R(8)]`), le `R(32)` de tete de record, le tail terminal, le
      routage de l'etat par defaut. Chaque difference citee par son adresse.
- [ ] 1.3 Trancher l'hypothese n°1 du brief PAR LECTURE : le corps NEW d'image-cle
      appelle-t-il, pour chaque composant present, le deser feuille `+0x28` (etat complet) la
      ou le delta appelle un wrapper ? Reponse ECRITE, avec l'adresse du site d'appel.
- [ ] 1.4 Resoudre l'etat par defaut de **`ti=42`** (chaine `default_state_arch.go:5-18`) :
      descripteur, vtable, `*(vtable+0x60)`, grammaire bit-exacte ou verdict « non resolue
      statiquement » ecrit.
- [ ] 1.5 Consigner TOUT le decompile porte dans une section neuve « image-cle » de
      `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` (adresses + grammaire), c'est le journal RE du
      depot. Aucune adresse en dur dans le code Go.

**Gate 1** :

```
grep -c "image-cle" .ai/V7.5/killweapon/WALK_PORT_NOTES.md        # section presente
grep -c "FUN_14076cb60\|FUN_1406d7610\|FUN_1408f1aa4" .ai/V7.5/killweapon/WALK_PORT_NOTES.md
```

Critere : la section existe, cite au moins 6 adresses, et repond en toutes lettres a 1.2,
1.3 et 1.4 (y compris par un « non resolu » motive).

### Phase 2 — PORTER et PROUVER : le walker DETERMINISTE d'image-cle

- [ ] 2.1 `filmdec/keyframe_record_walk.go` (fichier NEUF, a moi) : `WalkKeyframeRecords(pay,
      reg)` qui PARSE la table — en-tete 64 bits puis corps par le lecteur de record NEW de
      PRODUCTION — et rend, par record, `Slot / Gen / TI / Field26 / BitStart / BitEnd /
      Desync`. Il REUTILISE `TraverseEntity` ; il ne recopie aucun deser.
- [ ] 2.2 **TEST D'H1, et c'est la mesure qui commande tout le lot** : sur les records
      `ti=37` bornes de l'oracle R3 (3 films, denominateur 1 226), publier (a) le taux
      d'atterrissage sur une ancre VALIDE-RELACHEE, (b) le taux d'atterrissage exact sur
      `recs[i+1].Bit` APRES chainage, (c) le nombre de records intercales sautes par le
      filtre fort, (d) la distribution des `field26` non nuls.
- [ ] 2.3 C1 statue avec ses chiffres (seuil 95 %). Si < 95 %, ouvrir H2 : distribution des
      ecarts modulo 8/32/64 et balayage du tail 0..64 bits, publie.
- [ ] 2.4 C2 : meme controle sur le second archetype le plus frequent des memes payloads,
      denominateur publie.
- [ ] 2.5 C3 : marche globale d'un payload complet — taux de payloads traverses de bout en
      bout, et verification que TOUS les records de `WalkKeyframeWorld` sont retrouves.
- [ ] 2.6 Instrument versionne `filmdec/keyframe_record_walk_test.go`, garde d'environnement
      `KF_GRAM_FILM` (saute en CI, patron `equipment_identity_test.go:35-50`), lecture seule,
      aucune ecriture disque.

**Gate 2** :

```
export GOCACHE=C:/Users/Guillaume/Projects/LevelUp-wt-kfgram/.gocache
cd apps/go-api && CGO_ENABLED=0 go build ./internal/analysis/...
cd apps/go-api && CGO_ENABLED=0 go vet ./internal/analysis/filmdec/
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/filmdec/ -run KFGram -v   # SKIP sans la garde
cd apps/go-api && CGO_ENABLED=0 KF_GRAM_FILM=<film> go test ./internal/analysis/filmdec/ -run KFGram -v -timeout 30m
gofmt -l internal/analysis/filmdec/                                                    # vide
```

Critere : build/vet/test verts, la garde saute sans la variable, et les quatre chiffres de
2.2 sont publies avec leurs denominateurs. **C1 passe ou echoue, les deux sont un livrable.**

### Phase 3 — `ti=42` : la position des armes au sol a l'image-cle

Ouverte SEULEMENT si C1 passe (sinon `[!]` + registre : la cause est ecrite en phase 2).

- [ ] 3.1 Etat par defaut de `ti=42` inscrit dans `defaultStateDeserByTI`
      (`default_state_arch.go`, **une seule ligne d'ajout** au fichier partage) a partir de
      la decompile de 1.4 — ou `[!]` motive si 1.4 a rendu « non resolue ».
- [ ] 3.2 `filmdec/keyframe_ti42_pos.go` (fichier NEUF, a moi) : positions `ti=42` lues au
      record d'image-cle par le walker deterministe. AUCUNE reecriture de
      `keyframe_ground_weapons.go`.
- [ ] 3.3 C4 : emprise (>= 95 %) contre le nuage de positions biped du meme film, ET bande
      FANTOME de meme cardinalite passee par le MEME code (reel/fantome >= 3x). Chiffres et
      denominateurs publies, par film.
- [ ] 3.4 Verdict ecrit : les 55 % de « positions fantomes » du 2026-08-12 tombent-elles ?
      Oui avec les chiffres, ou non avec les chiffres.

**Gate 3** :

```
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/
cd apps/go-api && CGO_ENABLED=0 KF_GRAM_FILM=<film> go test ./internal/analysis/filmdec/ -run KFTI42 -v -timeout 30m
```

### Phase 4 — `ti=11` : l'objectif vivant (type, etat, progression)

Ouverte SEULEMENT si C1 passe.

- [ ] 4.1 Decompiler et porter les composants `ti=11` necessaires : `i5 type`,
      `i12 progress`, `i13 required-progress`, `i14 state` au minimum ; `i3
      object-reference` si le mur tombe. Chaque composant porte cite sa fonction.
- [ ] 4.2 `filmdec/keyframe_ti11_objectifs.go` (fichier NEUF, a moi) : lecture par record
      d'image-cle, sur `64e8adfa` et `530820e5`, avec `0014603f` en temoin negatif.
- [ ] 4.3 C5 : `i12 <= i13` (>= 95 %), `i5` stable par slot (>= 95 %), et confrontation aux
      evenements `flag_grabs` / `flag_captures` d'`objectiveevents` (TimeMS BRUT — le
      decalage `originMs` est un defaut connu, contourne, NON corrige ici).
- [ ] 4.4 Verdict ecrit, avec le temoin negatif (le film sans objectif doit rendre ZERO).

**Gate 4** :

```
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/objectiveevents/
cd apps/go-api && CGO_ENABLED=0 KF_GRAM_FILM=<film> go test ./internal/analysis/filmdec/ -run KFTI11 -v -timeout 30m
```

### Phase 5 — PUBLIER dans l'artefact, SANS bosse de `SchemaVersion`

Ouverte SEULEMENT si C4 ou C5 passe. Sinon `[!]` + registre (regle de non-publication §1.1).

- [ ] 5.1 Assemblage dans `replay/` (fichier NEUF, a moi) : par entite, le TYPE (identifiant
      stable, jamais un libelle) et la donnee mesuree, bornes a la fenetre du document.
      Test unitaire PUR (entrees synthetiques, aucune I/O).
- [ ] 5.2 Champ optionnel (`omitempty`) sur `ReplayDocument` + `Coverage`. **`SchemaVersion`
      NON TOUCHE** — le champ et la raison de la bosse notes en 5.4.
- [ ] 5.3 Contrat OpenAPI regenere + `generated.ts` + normalisation web si le champ traverse.
      Aucun composant de rendu touche, aucune string i18n ajoutee.
- [ ] 5.4 Note ECRITE au plan : quel champ exigera une bosse `SchemaVersion` 8 -> 9 et
      pourquoi (la reprise du backfill se fait PAR la version). **La bosse unique est faite
      par le superviseur a la fusion.**

**Gate 5** :

```
cd apps/go-api && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./internal/analysis/...
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/... ./contracttest/
make check-types && make test-web        (si 5.3 touche le web)
```

### Phase 6 — CLORE

- [ ] 6.1 Toutes les cases de ce plan statuees `[x]` / `[~]` / `[!]`.
- [ ] 6.2 Lignes proposees pour `.ai/V7.5/REGISTRE_REPORTS.md` redigees ICI (§11), en une
      seule fois, y compris les lignes a AMENDER : report `ti=42` du 2026-08-12, ligne
      `ti=11` de R4, decouverte 2 de R3.
- [ ] 6.3 Entree `.ai/thought_log.md` REDIGEE et remise au superviseur — ce lot n'ecrit PAS
      dans le journal, il appartient au superviseur.
- [ ] 6.4 Rapport final : par phase, ce que Ghidra a montre, les mesures REELLES avec leurs
      denominateurs, les gates (commandes + sorties), les commits et les push.

---

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

_(rempli en cours d'execution)_

## 10. Journal d'execution

_(rempli a la cloture de chaque phase : date, gate execute, sorties, commit)_

## 11. Lignes de registre proposees

_(redigees en une seule fois a la phase 6)_

## 12. Entree `thought_log.md` proposee

_(redigee a la phase 6 ; ce lot n'ecrit pas dans le journal)_

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
