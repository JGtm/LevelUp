# PLAN — TOUS LES SOCLES DEPUIS LES FICHIERS DE CARTE (.mvar)

Lot de MESURE SEULE, ouvert le 2026-08-19. Worktree frere `LevelUp-wt-socles-mvar`
(branche `wt/socles-mvar`, base `feat/v75`). Aucune publication, aucun changement de
production : ce plan mesure et rend un verdict. Une section « production proposee » ne
s'ecrit QUE si la mesure la porte.

## 1. La question

La detection actuelle des socles (`ground_weapon_pads.go`) exige une RECURRENCE : une
position ne devient un socle qu'a partir de DEUX apparitions d'armes de meme famille dans
le film. Consequence mesuree et acceptee jusqu'ici : un socle dont l'arme n'a ete servie
qu'une fois est INVISIBLE, et l'objet deja pose a t=0 n'est pas repliqué. Le film sous-
compte donc les socles par construction (temoin : `530820e5`, Catalyst CTF, rend 5 socles
la ou deux autres films de la meme carte en rendent 10 aux memes coordonnees).

D'ou la demande : **chercher tous les socles DES LE DEPART**, dans les fichiers de carte,
pour ne rien manquer.

## 2. Acquis verifies sur pieces (2026-08-19)

| Piece | Chemin absolu | Ce qu'elle donne |
|---|---|---|
| Decodeur Bond CB2 | `C:\Users\Guillaume\Projects\LevelUp\apps\go-api\internal\analysis\replay\mapvar\cb2.go` | `DecodeRoot` generique ; tout tag inconnu = erreur, jamais de saut silencieux |
| Grammaire `.mvar` | `...\mapvar\mapvar.go` | `root[3]` objets (type_id, pos, up, forward, flags, categorie, team, labels, instance_id, forme) ; `root[10][1]` table de chaines ; **`root[6]` et `root[11]` NON EXPLOITES** |
| Roles d'objectif | `...\mapvar\objectives.go` | labels = murmur3_x86_32(seed 0) du nom snake_case ; 28 noms resolus ; les `*_include/_exclude` sont des FILTRES DE MODE, pas des roles |
| Generateur du catalogue | `...\apps\go-api\cmd\mapobj-build\` | `--from-file` + `--dump-objects` (dump JSON de TOUS les objets), `--refresh-from <dossier>` hors ligne, `--save-mvar` |
| Depot local des `.mvar` | `C:\Users\Guillaume\Projects\LevelUp\.ai\re_dump\mapvar\` | **199 fichiers**, nommes `{carte}_{fichier}.mvar` |
| Catalogue fige | `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_objectives.json` | schema 2, cle = map_id, 72 cartes, champs `names[]`, `objectives[]`, `bounds`, `level_id`, `objects_n` |
| Artefacts de rejeu cuits | `C:\Users\Guillaume\Projects\LevelUp\data\cache\replays\halo_infinite\{film}.json` | `weaponPads[]` = L'ORACLE (x, y, z au centimetre, `weapon`, `spawns`, `presence`) |
| Correspondance film -> carte | `...\internal\analysis\replay\objectifs_phase1_drapeau_test.go` (`objCTFModules`, `objCTFCarteNom`) | Catalyst : `64e8adfa`, `530820e5`, `01e1f945` ; Cliffhanger : `bcb6d393`, `000d5950` |
| Piege connu | memoire du depot | une carte Forge `.mvar` = **canevas + rack** ; Smallhalla est Forge (canevas `fo08_wetland`) |

### 2.1 L'oracle, extrait des artefacts locaux (pas des artefacts publies)

**Catalyst** (`e859cf75-9b8a-429a-91be-2376681c8537`, module `catalyst`) — 10 socles
d'armes aux MEMES coordonnees dans `01e1f945` (KOTH) et `64e8adfa` (CTF), au centimetre :

```
 1  -9.738    -0.003    22.403
 2   9.472    12.411    24.003
 3   9.481   -12.404    24.015
 4 -11.046    -0.003    25.344
 5   5.160    -0.003    26.501
 6   0.003   -25.204    26.501
 7   0.003    25.298    26.501
 8  11.597    -0.003    22.600
 9   6.277     6.939    27.018
10   6.286    -6.945    27.018
```

Plus **1 socle de power-up** mesure au lot precedent : `0.257 ; -0.003 ; 21.36`
(`powerup_overshield`, presente en KOTH, absent des deux films CTF de la meme carte).
Total oracle Catalyst : **11 positions**.

**Cliffhanger** (`bcb6d393`, CTF, module `cliffhanger_ridgeline`) : 10 socles.
**Smallhalla** (`00162144`, module `smallhalla_map`) : 11 socles.
**Cliffhanger Super Fiesta** (`000d5950`) : **aucune cle `weaponPads`** — 0 socle mesure.

Fait etabli du lot precedent, qui borne ce qu'on peut esperer : **le socle appartient a la
carte, l'arme qui y apparait appartient au match** (epee et marteau au meme point selon le
film ; trois des dix socles de Catalyst portent une arme differente entre deux films).

## 3. Hypotheses, ecrites AVANT la mesure

- **H-SCENARIO (en tete, precision utilisateur du 2026-08-19)** — ce qui dicte ce que les
  socles font apparaitre, voire ce qui les MET EN PLACE, ce sont les **scenarios** (le
  variant de MODE), pas la carte seule. Une carte peut avoir davantage de socles sur
  certains modes ; en Fiesta il n'y en a en principe aucun. Deux formes a distinguer :
  - **H-SCENARIO-a (activation)** : les spawners sont dans le `.mvar` de carte et le mode
    les allume/eteint (labels `*_include` / `*_exclude`, deja connus du decodeur).
  - **H-SCENARIO-b (pose)** : les spawners vivent dans un fichier de MODE (variant de
    scenario / gametype, ou un `.mvar` par combinaison carte x mode) et le `.mvar` de
    carte n'en porte aucun.
  Le **test discriminant est deja disponible et hors ligne** : Cliffhanger, meme carte,
  deux modes, 10 socles (`bcb6d393`, CTF) contre 0 (`000d5950`, Super Fiesta). Si les deux
  matchs partagent le meme `.mvar` et que ce fichier porte les 10 spawners, H-SCENARIO-a
  est retenue et H-SCENARIO-b refutee pour la POSE (le mode ne fait qu'activer).
- **H1 (portage)** : le `.mvar` d'une carte DEV porte des objets a moins d'un metre de
  chaque socle de l'oracle.
- **H2 (identite)** : l'objet du `.mvar` porte de quoi savoir CE QU'IL FAIT SPAWN
  (reference d'arme `weap` / d'equipement `eqip`), ou bien il est generique et l'arme vient
  d'ailleurs. Les deux champs jamais decodes (`root[6]` regroupements, `root[11]`
  surcharges de proprietes indexees) sont les candidats a inspecter en priorite.
- **H3 (Forge)** : sur une carte Forge (Smallhalla), le rack d'objets porte les socles et
  le canevas n'en porte aucun — le piege « canevas + rack » impose de mesurer les DEUX
  fichiers avant de conclure.

## 4. Seuils, ecrits AVANT la mesure

| Mesure | Seuil de reussite | Refutation |
|---|---|---|
| **Appariement (Q2)** | par carte, **>= 90 %** des positions de l'oracle ont un objet du `.mvar` a **< 1 m** (distance 3D) | < 90 % : le `.mvar` de carte ne porte pas les socles |
| **Temoin negatif** | 100 tirages de N positions aleatoires dans l'emprise XY de la carte (Z conserve de l'oracle, graine fixe) : **<= 20 %** apparies a < 1 m | temoin > 20 % : la carte est si dense que « a moins d'un metre » ne discrimine rien — l'appariement ne prouve alors rien |
| **Rapport signal/temoin** | **>= 3** | < 3 : verdict negatif meme si le taux brut passe |
| **Diagnostic XY** (declare AVANT la mesure, ne remplace aucun seuil) | la distance est aussi mesuree en XY seul | si XY apparie a < 1 m et pas la 3D, l'objet est au bon endroit du plan mais a une autre ALTITUDE (origine au sol contre arme flottante) : a dire, sans requalifier le verdict |
| **Identite (Q3)** | l'arme est declaree LISIBLE si un champ de l'objet apparie prend **>= 2 valeurs distinctes** entre socles portant des armes distinctes ET est **constant** entre les deux films de la meme carte | sinon : « spawner generique, arme non lisible dans le `.mvar` » |
| **Generalisation (Q5)** | les 3 cartes passent le seuil d'appariement | 1 carte sur 3 : resultat partiel, ecrit comme tel |

**Gate d'arret global** : si la phase 2 (Catalyst, la carte la mieux mesuree) echoue au
seuil d'appariement OU au rapport signal/temoin, les phases 3 et 5 ne s'executent pas —
on ecrit le negatif avec ce qui a ete vu (phase 6 directement). La phase 4 s'execute dans
tous les cas : elle repond a H-SCENARIO, qui vaut meme en cas de negatif.

## 5. Contraintes d'execution

Lecture seule ; aucune base de donnees ouverte ; chemins ABSOLUS vers
`C:\Users\Guillaume\Projects\LevelUp\data\...` et `...\.ai\re_dump\mapvar\` ; instruments
sous garde d'environnement (`MVAR_FILE`, `MVAR_DIR`, `MVAR_ORACLE`) — sans la garde, le
test SKIPPE ; `CGO_ENABLED=0` ; `GOCACHE` local au worktree ; **un seul `go` a la fois** ;
jamais de push, de stash, ni de `git add -A`. Gates par phase :
`go vet ./internal/analysis/...`, `go test ./internal/analysis/...`, golangci 0 nouveau.


## 6. Phases

### Phase 0 — l'instrument (commit 1)

- [x] 0.1 Ecrire `internal/analysis/replay/mapvar/socles_research_test.go` : chargement
      d'un `.mvar` sous garde `MVAR_FILE`, inventaire des objets (type_id, categorie,
      labels resolus/non resolus, forme, nom d'instance si atteignable).
- [x] 0.2 Ajouter la lecture BRUTE de `root[6]` et `root[11]` (champs jamais decodes) —
      forme, cardinalite, correlation avec l'index d'objet.
- [x] 0.3 Gate : `go vet` + `go test` du paquet sans garde (skip propre).

### Phase 1 — que contient le `.mvar` de Catalyst ? (Q1) (commit 2)

- [x] 1.1 Inventaire complet de `catalyst_catalyst.mvar` (337 objets) : histogramme des
      type_id, des categories, des labels.
- [x] 1.2 Idem pour `catalyst_map.mvar` (357 objets) — deux fichiers pour la meme carte,
      il faut savoir lequel est le bon avant d'apparier.
- [x] 1.3 Verdict Q1 ecrit : le `.mvar` porte-t-il des objets de type « spawner » ?

### Phase 2 — appariement de l'oracle Catalyst (Q2) (commit 3)

- [ ] 2.1 Coder les 11 positions de l'oracle (section 2.1) dans l'instrument, sous garde.
- [ ] 2.2 Pour chaque position : objet du `.mvar` le plus proche, distance 3D, type_id,
      labels.
- [ ] 2.3 Temoin negatif : 100 tirages, graine fixe, meme emprise, meme N.
- [ ] 2.4 **GATE D'ARRET** : >= 90 % a < 1 m et signal/temoin >= 3, sinon aller en phase 4
      puis 6.

### Phase 3 — que fait spawn le socle ? (Q3) (commit 4)

- [ ] 3.1 Sur les objets apparies : quels champs varient avec l'arme observee ?
- [ ] 3.2 Croiser avec les deux films de Catalyst (armes differentes sur 3 socles).
- [ ] 3.3 Inspecter `root[11]` (surcharges indexees) sur les index des objets apparies.
- [ ] 3.4 Verdict Q3 ecrit : arme fixee dans le fichier, ou spawner generique ?

### Phase 4 — le variant de MODE (Q4 + H-SCENARIO) (commit 5)

- [ ] 4.1 Inventorier les `.mvar` du depot par carte : combien de fichiers, quels noms
      (`ctf_breaker.mvar`, `ridgeline.mvar`, `map.mvar`...), quel `level_id`.
- [ ] 4.2 Cliffhanger : comparer objet par objet les fichiers servant le match CTF
      (`bcb6d393`) et le match Super Fiesta (`000d5950`) — meme fichier ? memes objets ?
- [ ] 4.3 Chercher, dans la chaine `himap` et le catalogue UGC, ce qu'une combinaison
      carte x mode expose comme fichiers (tags de scenario / gametype referencant des
      spawners).
- [ ] 4.4 Verdict : H-SCENARIO-a (activation) ou -b (pose), ou « hors de portee hors
      ligne » ecrit comme tel.

### Phase 5 — generalisation (Q5) (commit 6)

- [ ] 5.1 Cliffhanger (`bcb6d393`, 10 socles) : appariement + temoin.
- [ ] 5.2 Smallhalla (`00162144`, 11 socles) : appariement + temoin, sur le rack ET sur le
      canevas (piege Forge).
- [ ] 5.3 Verdict Q5.

### Phase 6 — cloture (commit 7)

- [ ] 6.1 Section « production proposee » (catalogue statique `map_weapon_pads.json`) OU
      negatif ecrit.
- [ ] 6.2 Gates finaux : `go vet ./internal/analysis/...`,
      `go test ./internal/analysis/...`, golangci 0 nouveau.
- [ ] 6.3 CR : reponses aux 5 questions avec chiffres, textes journal et registre.

## 7. Journal

- **2026-08-19** — Plan ouvert. Acquis verifies sur pieces (chaine `mapvar`, `mapobj-build`,
  depot des 199 `.mvar`, catalogue fige) ; oracle re-extrait des artefacts LOCAUX plutot que
  des artefacts publies (10 socles Catalyst identiques au centimetre entre `01e1f945` et
  `64e8adfa`). Hypothese H-SCENARIO placee en tete a la demande de l'utilisateur, avec son
  test discriminant deja disponible hors ligne (Cliffhanger 10 vs 0). Aucune mesure lancee
  a cette heure.
- **2026-08-19, phase 0 close** — Instrument `socles_research_test.go` ecrit dans le paquet
  `mapvar`, sous gardes `MVAR_FILE` / `MVAR_DIR` / `MVAR_ORACLE` / `MVAR_ORACLE_POINTS`.
  L'oracle n'est PAS code en dur : il se lit dans un artefact de rejeu cuit
  (`weaponPads`), plus des points fournis a la main pour le socle de power-up, qui vient
  d'une autre voie de mesure. Gates : `go vet` propre, `go test -run TestSocles` vert avec
  trois SKIP explicites sans garde. Report assume : la consigne du lot demande « ni journal
  ni registre », donc AUCUNE entree `.ai/thought_log.md` n'est ecrite — les textes partent
  au CR. C'est la seule derogation au contrat `plan-execution`, et elle est ordonnee.
- **2026-08-19, phase 1 close** — Inventaire des deux `.mvar` de Catalyst (meme `level_id`
  `-1044063363`, deux map_id distincts au catalogue).
  `catalyst_catalyst.mvar` : 337 objets, **36 type_id distincts**, 3 categories (-1, 1, 2),
  31 objets avec forme, 124 avec index d'equipe, table de chaines **VIDE**. Labels resolus :
  `infection_include` 109, `ctf_include` 8, `strongholds_include` 6, `extraction_zone` 6,
  `extraction_include` 5, `assault_include` 4, `flag_spawn` 3, `strongholds_zone` 3,
  `flag_delivery` 2, plus 4 unitaires. **9 hashs de label INCONNUS** (le plus frequent
  `-886053664`, 18 fois).
  `catalyst_map.mvar` : 357 objets, 42 type_id, 4 categories, 3 noms, **20 hashs inconnus**.
  `root[6]` DECODE (il n'etait pas seulement « non exploite », il n'avait jamais ete
  regarde) : 11 blocs, chacun `{.0 = offset de depart, .1 = liste d'entrees, .5.0 = un
  int32 de type hash}`. Les offsets s'enchainent (0, 135, 143, 167, 268, 293, 294, 300,
  337, 340, 346) et la somme des entrees vaut 348 = `root[6].2`. Chaque entree vaut
  `{0:1, 1:identifiant sequentiel, 6:0xFFFFFFFF, 8:index, 9:struct vide}` : c'est une
  **table d'allocation d'identifiants**, sans position ni reference d'objet. La coincidence
  « 11 blocs / 11 socles » est fortuite et le disque le montre.
  `root[11]` : present, type BT_MAP, **vide** sur Catalyst.
  **Verdict 1.3** : le fichier porte 36 a 42 types d'objets dont un seul est identifie
  (`-1239931096`, le volume d'objectif que `map_objectives.json` publie deja) ; RIEN ne les
  nomme (table de chaines vide sur les cartes DEV) ; aucune reference d'arme n'est lisible.
  Le seul discriminant disponible est donc la POSITION — c'est l'objet de la phase 2.

## 8. Verdict et suite

A ECRIRE A LA CLOTURE — section « production proposee » si la mesure la porte, negatif
ecrit sinon.

## 9. Decouvertes (notees, NON traitees)

A remplir au fil de l'execution.
