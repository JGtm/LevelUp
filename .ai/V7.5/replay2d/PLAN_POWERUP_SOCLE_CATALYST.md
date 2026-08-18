# PLAN — LE POWER-UP DE SOCLE AU CENTRE DE CATALYST (lot de MESURE)

> Lot COURT, MESURE SEULE. Aucune publication, aucun changement de production, aucune base.
> Instrument sous garde `OBJ_FILM`, lecture seule, `CGO_ENABLED=0`, un seul `go` a la fois.
> Ni journal ni registre : les textes partent au CR.
> Worktree : `C:\Users\Guillaume\Projects\LevelUp-wt-powerup` (branche `wt/powerup`).
> Contrat d'execution : `.claude/skills/plan-execution/SKILL.md`.

## 0. La question

Temoignage utilisateur du 2026-08-18 : « sur Catalyst on a un power-up sur un socle au centre
de la carte, sur un pont ; selon les sous-modes c'est un camouflage ou un surbouclier. »

La chaine de production ne le voit pas. Ce lot mesure POURQUOI, et si le film le porte.

## 1. Acquis — verifie sur pieces le 2026-08-18, ne pas remesurer

### 1.1 Le negatif, et sa cause mecanique probable

| fait | piece |
|---|---|
| Aucun power-up de socle dans les artefacts cuits de Catalyst | `data/cache/replays/halo_infinite/{64e8adfa,530820e5,01e1f945}.json`, champs `weaponPads` (10 socles d'ARME, aucun power-up) et `equipmentPlacements` |
| Un SEUL `powerup_overshield`, sur `01e1f945`, `origin: dropped`, avec porteur | `{"t0":4104,"t1":4125,"x":0.239,"y":6.802,"z":26.894,"family":"powerup_overshield","id":"0xb781197a","owner":589,"origin":"dropped"}` |
| **La chaine ti=37 EXIGE une vie delta** : `confirmPlacements` ne retient un record de creation que si sa position retombe sur le premier point de la vie que son en-tete annonce (`MatchEquipmentLife`) | `filmdec/equipment_placements.go:158-190` |
| Un objet POSE cesse d'emettre sa position (meme acquis pour `ti=37` et `ti=42`) | en-tete de `filmdec/equipment_placements.go` ; `keyframe_ground_weapons.go` (GroundWeaponPositions) |

**Consequence a poser avant toute mesure** : un objet de socle QUI NE BOUGE JAMAIS n'a aucune
vie delta, donc AUCUN de ses records de creation ne peut etre confirme — il est invisible a
`equipmentPlacements` PAR CONSTRUCTION. L'absence dans l'artefact n'est donc PAS une preuve
d'absence dans le film. La chaine `ti=42` (armes au sol), elle, retient les creations SANS vie
delta (`at_rest`) et filtre par IDENTITE : c'est l'asymetrie a exploiter.

### 1.2 L'ORACLE : cinq ramassages dates sur `01e1f945`

`coverage.equipment` des trois films cuits, et il DISCRIMINE :

| film | mode | `overshieldLives` / `overshieldEpisodes` | `camoLives` |
|---|---|---|---|
| `01e1f945` | KOTH | **5 / 5** | 0 |
| `64e8adfa` | CTF | 0 / 0 | 0 |
| `530820e5` | CTF | 0 / 0 | 0 |

`equipmentEpisodes` de `01e1f945` (schema 7, episode date par vie de bipede) :

```
{slot:512,t0:347}  {slot:537,t0:1444}  {slot:564,t0:2690}  {slot:604,t0:4363}  {slot:619,t0:5219}
```

Cinq surbouclier ACTIFS dans un film KOTH hors Fiesta. Hors Fiesta, aucune autre source de
surbouclier : chaque `t0` date un RAMASSAGE, et la position du bipede a cet instant EST la
position du socle. C'est un oracle a cinq points, disponible SANS toucher au film (les pistes
de bipede sont dans `tracks[].points[]` de l'artefact cuit).

Reserve ecrite avant mesure : sur `000d5950` (Fiesta) i28 s'allume au DASH du mode, pas a un
ramassage (registre des reports, ligne 163). Le present film n'est pas Fiesta, mais la mesure
de dispersion de la phase 1 est ce qui tranche — pas cette remarque.

### 1.3 Le cote fichiers de jeu : la question est deja tranchee, en negatif

- `map_objectives.json` ne connait que 8 roles (`flag_spawn`, `extraction_zone`,
  `flag_delivery`, `strongholds_zone`, `oddball_spawn`, `stockpile_socket`,
  `stockpile_navpoint`, `assault_bomb`) : **aucun role de power-up n'existe dans le
  vocabulaire d'extraction**, pour aucune carte. Catalyst (`e859cf75-...`, 337 objets dans le
  `.mvar`, 14 objectifs retenus) n'y porte donc rien de tel — et n'aurait pas pu.
- Le releve Forge du 2026-08-02 nomme pourtant les types d'emplacement du jeu :
  `PowerUpPad`, `PowerWeaponPad`, `WeaponRack`, `EquipmentPad`... (`.ai/V7.5/film_re/HANDOFF_PALETTE_FORGE_2026-08-02.md`).
  Le releve des objets a delai de reapparition
  (`.ai/V7.5/dumps/forge_zones/emplacements_a_observer.txt`, 285 socles sur 50 cartes)
  **ne contient AUCUNE ligne Catalyst** : les racks des cartes NATIVES ne sont pas dans le
  `.mvar` (piege connu : `.mvar` = canevas + rack, mais le canevas d'une carte native est
  vide de son armement).

Ce lot ne rouvre PAS la voie fichiers de jeu (prealable (a) du registre, entier).

### 1.4 Les outils disponibles, et ce qu'ils rendent

| outil | rend |
|---|---|
| `filmdec.ScanFilmWorldObjects(dir, wr, ti)` | vies (slot, gen) + positions ABSOLUES des paquets delta, pour un archetype |
| `filmdec.WalkKeyframeWorld(pay)` | slot -> ti a chaque image-cle (aucune position) |
| `filmdec.ScanFilmEquipmentCreationsForBand` / `...GroundWeaponCreationsForBand` | records NEW : position i0 + identite `MPPVal[MPPWord32]` (`eqip` pour 37, `weap` pour 42) |
| `filmdec.CalibrateMPPWidths` | largeur du bloc MPP, MESUREE film par film (defaut 9/5 : un film calibre autrement rend zero identite en silence) |
| `mapQuantEntryFromEnv` / `installWorldObjectPrecision` | bornes de dequantification (`map_quant_bounds.json`, Catalyst `[15 15 15]`) |
| archetypes 36..42 | 36 inconnu · 37 equipement · 38 corps rigide · 39 inconnu · 40 vehicule · 41 projectile · 42 arme au sol (`filmdec/testdata/ecs_table.tsv`) |

A noter dans la table : `ti=37` porte i30 `equipment-has-infinite-uses-component`, documente
« socle de carte contre equipement de joueur ». Le composant existe ; personne ne l'a lu.

## 2. Hypotheses — ECRITES AVANT LA MESURE

- **H1** — le power-up de socle est un objet `ti=37` present dans l'ETAT INITIAL du film
  (des t=0, sans record NEW delta) : invisible a `ScanFilmEquipmentCreations`, mais recense
  aux images-cles.
- **H2** — c'est un AUTRE archetype (36, 38, 39, ou un ti hors 36..42).
- **H3** — il n'est jamais replique : rien dans le film.
- **H4** — il n'est cree qu'a son PREMIER RESPAWN (apres ramassage) et manque a t=0.

Ces quatre ne sont pas exclusives deux a deux (H1 et H4 peuvent cohabiter : etat initial +
records NEW aux respawns). Le verdict le dira.

## 3. Seuils — ECRITS AVANT LA MESURE

| seuil | valeur | pourquoi |
|---|---|---|
| centre de Catalyst | origine des symetries des socles publies, calculee en phase 0 | y : paires (25,298/-25,204), (12,411/-12,404), (6,939/-6,945) ; x : mediane des socles de l'axe y=0 |
| boite large | \|dx\| <= 6 m ET \|dy\| <= 6 m, z LIBRE (rapporte) | le pont est en hauteur : contraindre z aveuglerait la mesure |
| boite candidat | <= 3 m du centre en XY | seuil de l'enonce |
| dispersion de l'oracle (phase 1) | les 5 points tiennent dans un rayon <= 2,0 m, contre un temoin (5 instants tires au hasard sur les memes bipedes) >= 3x plus disperse | un ramassage se fait EN MARCHANT SUR l'objet |
| disparition liee a un joueur | bipede a < 1,5 m dans les 2 images (200 ms) qui encadrent le dernier point de l'objet | constantes de production `originDropWindowUS` / `originDropMaxDist` |
| « meme point » a la reapparition | <= 1,0 m | `gwPadRadiusM` (regle de grappe en production) |
| delai stable | >= 2 ecarts, coefficient de variation <= `gwPadCycleMaxCV` (regle de production) | ne pas inventer une seconde regle de stabilite |
| temoin fantome (creations) | bande de MEME cardinalite faite de slots jamais vus porter l'archetype ; un positif exige un facteur >= 10 sur les creations RETENUES PAR IDENTITE | decouverte 2 du plan armes au sol : l'acceptation seule n'est pas discriminante (398 fantomes contre 366 reelles) |
| **verdict positif** | un candidat present a t=0 a < 3 m du centre, qui disparait avec un joueur a < 1,5 m dans les 2 images, reapparait au meme point apres un delai stable, ET dont l'identite resout en `powerup_*` — OU qui est le seul objet du centre commun aux 4 films | enonce du lot |
| **verdict negatif** | tout le reste : ecrire ce qui a ete vu, avec ses chiffres et ses denominateurs | un negatif sans denominateur n'est pas un negatif |

## 4. Phases

### Phase 0 — Ancrage et instrument `[x]`

- `[x]` 0.1 Calculer le centre de Catalyst depuis les `weaponPads` publies des 3 films cuits
  (symetries en y ; mediane en x sur l'axe y=0). Ecrire le chiffre au plan.
- `[x]` 0.2 Creer `apps/go-api/internal/analysis/replay/powerup_socle_research_test.go` :
  garde `OBJ_FILM` (repertoire des chunks), `OBJ_FILM_MAP` (carte), `OBJ_FILM_ART`
  (repertoire des artefacts cuits, defaut `data/cache/replays/halo_infinite`). Sans garde,
  tout se saute. `LockProcessDecode` + `installWorldObjectPrecision` + restauration des
  largeurs MPP.
- `[x]` 0.3 Gate : `go vet ./internal/analysis/...` ; `go test ./internal/analysis/...`
  (l'instrument se saute, la suite reste verte) ; `golangci-lint` 0 nouveau.

### Phase 1 — L'oracle des cinq ramassages `[x]`

- `[x]` 1.1 Lire `equipmentEpisodes` + `tracks[].points[]` de `01e1f945` ; rendre la position
  du bipede de chaque episode a son `t0` (et l'encadrement le plus proche si `t0` n'est pas
  un point).
- `[x]` 1.2 Dispersion des 5 points : centroide, rayon max, ecart au centre de la carte.
- `[x]` 1.3 TEMOIN : 5 instants tires au hasard sur les MEMES bipedes, par le MEME code.
- `[x]` 1.4 Verdict d'etape : la position du socle est-elle MESUREE ? Si oui, elle devient la
  cible des phases 2 et 3 (boite 3 m autour d'ELLE, en plus du centre geometrique).
- `[x]` 1.5 Gate : `go vet` + `go test ./internal/analysis/...` + instrument joue sur
  `01e1f945`.

### Phase 2 — Les archetypes dans la boite (H2 / H3) `[x]`

- `[x]` 2.1 `ScanFilmWorldObjects` pour ti = 36..42 sur les 4 films ; ne garder que les vies
  dont un point tombe dans la boite large. Par archetype : nombre de vies, t du premier et du
  dernier point, etendue spatiale, z.
- `[x]` 2.2 Presence a t=0 : vies dont le premier point est a t <= 5 s.
- `[~]` 2.3 Disparitions liees a un joueur : dernier point d'une vie de la boite avec un
  bipede a < 1,5 m dans les 2 images. Sur `01e1f945`, croiser avec les 5 instants de l'oracle.
- `[~]` 2.4 Reapparitions : vies successives au meme point (<= 1 m) ; ecarts, stabilite.
- `[x]` 2.5 Gate : `go vet` + `go test ./internal/analysis/...` + instrument joue sur 4 films.

### Phase 3 — Images-cles, creations, identite (H1 / H4) et verdict `[x]`

- `[x]` 3.0 CONTROLE de la phase 1 (ajoute en cours d'execution, item declare) : rejouer la
  remontee sur d'AUTRES vies aux MEMES instants. Le centre d'une arene est un goulot ; sans ce
  temoin, un croisement au centre pourrait n'etre qu'un artefact de frequentation.
- `[x]` 3.1 Recensement `WalkKeyframeWorld` : par image-cle et par ti, nombre de slots ; les
  slots presents DES LA PREMIERE image-cle et a TOUTES ; ceux qui n'ont AUCUNE position delta
  (population « invisible », la forme meme de H1) — par archetype, avec denominateurs.
- `[x]` 3.2 Creations dans la boite : `ScanFilmEquipmentCreations` (ti=37), largeurs MPP
  CALIBREES par film (jamais les defauts), sans l'oracle de vie delta. Identite
  `MPPVal[MPPWord32]` -> famille du manifeste (`replay_labels.toml`, `powerup_overshield`
  `0xb781197a`, `powerup_camo` `0xe7be9f5c`).
  **Le volet `ti=42` est `[~]`** : ce chemin est DEJA en production
  (`ScanFilmGroundWeaponCreations` -> `weaponPads`, 10 socles publies sur Catalyst, aucun
  power-up parmi eux), et son mot d'identite se resout dans le catalogue d'ARMES (`weap`),
  pas dans le catalogue `eqip` ou vivent les familles `powerup_*`. Le rejouer ici n'aurait
  rien pu rendre de plus.
- `[x]` 3.3 Temoin fantome sur 3.2, meme code, meme cardinalite.
- `[x]` 3.4 Croisement des 4 films : un identifiant present sur les 4 ? un identifiant qui
  CHANGE entre CTF et KOTH (signal du sous-mode) ?
- `[x]` 3.5 Verdict : hypothese retenue (ou negatif ecrit avec ses chiffres) ; ce que cela
  implique pour la chaine des socles — UNE phase de production A PROPOSER, pas a faire.
- `[x]` 3.6 Gate : `go vet` + `go test ./internal/analysis/...` + golangci 0.

## 5. Ce que ce lot ne fait PAS

- Aucune publication au document de rejeu, aucun schema, aucun champ.
- Aucune modification du web, aucune re-cuisson d'artefact.
- Aucune ecriture au journal ni au registre (les textes partent au CR du lot).
- Aucune reouverture de la voie « racks des fichiers de jeu » (prealable (a) du registre).
- Aucun fix opportuniste : toute decouverte va en section 7.

## 6. Journal d'execution

### Phase 0 — CLOSE le 2026-08-18

Instrument : `apps/go-api/internal/analysis/replay/powerup_socle_research_test.go`. Il
REUTILISE la garde `OBJ_FILM` du paquet (racine du cache film, `objRequireRoot`) et n'ajoute
que `OBJ_FILM_ART` (repertoire des artefacts cuits, defaut = chemin du depot).

**Corpus (0.1)** — les quatre films, dont l'artefact du quatrieme n'a jamais ete cuit :

| film | mode | images | socles d'arme | poses | episodes | surbouclier | camo |
|---|---|---|---|---|---|---|---|
| `64e8adfa` | CTF | 8 337 | 10 | 229 | 0 | 0 vies / 0 ep. | 0 |
| `530820e5` | CTF | 4 751 | 5 | 156 | 0 | 0 / 0 | 0 |
| `01e1f945` | KOTH | 5 343 | 10 | 151 | **5** | **5 vies / 5 ep.** | 0 |
| `75f1188f` | KOTH | — | artefact ABSENT | | | | |

**Centre de Catalyst (0.1)** — 10 socles uniques (dedupliques a 0,5 m), **3 paires miroir**
-> axe `y = 0,0159` ; **4 socles sur l'axe**, x de -11,046 a 11,597 -> milieu `0,2755`.

> **CENTRE RETENU = (0,276 ; 0,016)**

Controle independant (milieu des bornes des positions JOUEES de chaque film) : `530820e5`
a 0,042 m du centre retenu, `01e1f945` a 0,208 m. `64e8adfa` rend (-106,0 ; -93,7), a
141,7 m — voir Decouvertes n°1.

**Gate (0.3)** : `go vet ./internal/analysis/...` 0 · `go test ./internal/analysis/...` 0 ·
`golangci-lint run ./internal/analysis/replay/...` **0 issues**.

### Phase 1 — CLOSE le 2026-08-18. **LE SOCLE EST MESURE.**

Instrument : `apps/go-api/internal/analysis/replay/powerup_socle_oracle_test.go`.

**1.1 — les cinq ramassages, a leur `T0` d'episode** (centre de la carte (0,276 ; 0,016)) :

| slot | `t0` | position a `T0` | z | d(centre) |
|---|---|---|---|---|
| 512 | 347 (34,7 s) | (-3,520 ; -0,850) | 22,08 | 3,89 m |
| 537 | 1444 (144,4 s) | (4,230 ; -0,090) | 21,36 | 3,96 m |
| 564 | 2690 (269,0 s) | (3,770 ; 0,210) | 22,02 | 3,50 m |
| 604 | 4363 (436,3 s) | (0,200 ; 5,920) | **26,73** | 5,90 m |
| 619 | 5219 (521,9 s) | (-3,460 ; -0,410) | 21,36 | 3,76 m |

Rayon du nuage a `T0` : **3,98 m** (temoin, memes vies a instants decorreles : 15,78 m,
facteur 3,18). Le seuil de 2 m n'est PAS atteint a `T0` — et c'est le resultat qui a ouvert
l'etape 1.5 : `T0` ne date pas le ramassage, il date l'instant ou la LECTURE du bouclier
depasse le plein. Entre les deux, le porteur a couru.

**1.6 — un ramassage sur cinq n'est PAS celui d'un socle, et la regle l'ecarte.** Le slot 604
ramasse a 0,88 m d'une pose `powerup_overshield` `0xb781197a` d'origine `dropped` posee a
`t0=4104` : c'est un surbouclier LACHE A UNE MORT, pas un socle. Regle appliquee (ecrite avant
la mesure, symetrique de la production) : pose `powerup_*` `dropped`, anterieure, a moins
d'`equipOwnerMaxDist` (3 m). **4 episodes retenus sur 5.**

**1.5 — la remontee : les quatre trajectoires SE CROISENT.** Dispersion du nuage `k` images
avant `T0`, k de 0 a 40 :

| k | 0 | 5 | 10 | 13 | **15** | 16 | 20 | 30 | 40 |
|---|---|---|---|---|---|---|---|---|---|
| rayon (m) | 3,980 | 2,745 | 1,463 | 0,650 | **0,256** | 0,312 | 0,781 | 2,359 | 4,545 |
| d(centre) | 0,30 | 0,19 | 0,11 | 0,06 | **0,12** | 0,28 | 0,96 | 3,04 | 5,12 |

Un V net, monotone des deux cotes. Minimum a **k = 15 images (1,5 s avant `T0`)** :

> **SOCLE MESURE en (0,393 ; -0,012), rayon 0,256 m, a 0,12 m du centre de la carte.**
> Altitude des quatre porteurs a cet instant : **z de 21,36 a 21,90** — l'etage BAS du
> milieu, 0,5 a 5,6 m sous les socles d'arme (22,40 a 27,02).

Les deux seuils ecrits avant la mesure sont ATTEINTS (rayon <= 2 m ; centroide a <= 3 m du
centre), et de deux ordres de grandeur pour le premier. Le decalage de 1,5 s est coherent avec
la MONTEE du surbouclier, qui n'est pas instantanee.

**Ce que la phase 1 etablit, et qui ne se remesure pas** : le power-up du centre de Catalyst
EXISTE, il est a **(0,39 ; -0,01)** au niveau **z ~ 21,4-21,9**, il est ramasse **4 fois** en
534 s sur `01e1f945`, et ses instants de ramassage sont `T0 - 15` images, soit **19,7 s ;
142,9 s ; 267,5 s ; 520,4 s**.

**Gate (1.5)** : `go vet` 0 · `go test ./internal/analysis/...` 0 ·
`golangci-lint run ./internal/analysis/replay/...` **0 issues**.

### Phase 2 — CLOSE le 2026-08-18 : les archetypes autour du socle. **NEGATIF, ET IL SE CHIFFRE.**

Instrument : `apps/go-api/internal/analysis/replay/powerup_socle_archetypes_test.go`. Cible :
le socle mesure en phase 1, **(0,393 ; -0,012) z 21,50** — calcule par appel a
`psSocleParRemontee`, jamais recopie. Boite XY 6 m (large) / 3 m (serree), z LIBRE.

**2.1 — vies des paquets DELTA, par archetype et par film :**

| film | ti=36 | ti=37 equip. | ti=38 rigide | ti=39 | ti=40 | ti=41 proj. (temoin) | ti=42 arme |
|---|---|---|---|---|---|---|---|
| `64e8adfa` CTF | aucun slot | 610 vies · 35/20/2 · 3D 1,13 | 468 · 27/19/1 · 1,13 | aucun | aucun | 303 · 34/20/2 · 2,16 | 810 · 52/25/4 · 1,13 |
| `530820e5` CTF | aucun slot | 282 · 32/21/4 · **0,91** | 210 · 27/18/4 · 0,91 | aucun | aucun | 199 · 39/28/**0** · 3,51 | 461 · 56/27/4 · 0,91 |
| `01e1f945` KOTH | aucun slot | 374 · 32/13/3 · **0,98** | 216 · 28/11/1 · 0,98 | aucun | aucun | 251 · 57/44/**0** · 2,26 | 646 · 79/27/5 · 0,98 |
| `75f1188f` KOTH | aucun slot | 225 · 50/21/1 · 1,16 | 129 · 15/9/1 · 1,16 | aucun | aucun | 271 · 85/57/7 · 1,95 | 399 · 89/43/8 · **0,70** |

Duree du balayage : 536 s pour les quatre films (204 / 112 / 130 / 90 s).

(lecture : `vies · boite/serree/immobiles · plus courte distance 3D au socle, en metres`)

**2.2 — « present des le depart » (premier point dans les 5 premieres secondes) : ZERO, pour
TOUS les archetypes et TOUS les films.** Aucun objet du monde n'emet de position au centre au
debut du match. C'est le resultat le plus net de la phase, et il ferme H4 dans sa forme naive
(« cree au premier respawn seulement ») autant que la lecture delta de H1.

**2.3 — aucune vie immobile a l'ALTITUDE du socle.** Les vies immobiles de la boite serree
sont a `z = 27,3 a 29,4` — l'etage HAUT du milieu, 6 a 8 m au-dessus du socle. La seule
exception est `530820e5` : deux vies a `z = 21,90` (slots 1931/1933, a 1,7-1,9 m du socle), de
1,2 et 3,8 s de duree, a 224,3 s — trop breves et trop tardives pour un socle. Il n'y a donc
**aucune disparition a correler avec les quatre ramassages** : `[~]` couvert par 2.1/2.2, il
n'existe pas d'objet a faire disparaitre.

**2.4 — reapparitions** : `[~]` meme raison. Sans vie delta au socle, aucune grappe.

**Ce que la phase 2 etablit** : le chemin DELTA ne porte PAS le power-up du socle, sur aucun
des quatre films, ni a `ti=37` ni ailleurs. Le temoin `ti=41` se comporte comme prevu (44 a
57 vies dans la boite serree, 0 immobile sur trois films sur quatre). Et il borne sa propre
portee : voir Decouverte n°3 — les bandes de slots delta se recouvrent, donc ce balayage
mesure une PRESENCE, jamais une attribution d'archetype.

### Phase 3

**3.0 — CONTROLE de la phase 1 : le croisement n'est PAS un artefact de goulot.**
Instrument : `powerup_socle_temoin_test.go`. Memes instants `T0`, meme remontee, meme code,
mais D'AUTRES vies (les slots 513, 518, 557, 612, choisis comme le plus petit slot vivant a
chaque instant et different des porteurs) :

| | rayon minimal | k | centroide |
|---|---|---|---|
| REEL (les 4 porteurs) | **0,256 m** | 15 | (0,393 ; -0,012) |
| TEMOIN (4 autres vies) | **14,242 m** | 4 | (0,208 ; -4,655) |

**Facteur temoin/reel = 55,6**, pour un seuil ecrit avant mesure de 3. Quatre joueurs
quelconques aux memes instants ne se croisent nulle part ; ces quatre-la si. La position du
socle tient.

**3.2 / 3.3 — LE BALAYAGE DES CREATIONS SANS L'ORACLE DE VIE DELTA. LE SOCLE EST TROUVE.**
Instrument : `powerup_socle_creations_test.go`. Les quatre films calibrent `9/5`.

| film | vies | ancres | acceptees | **confirmees** (production) | **retenues par identite** (brut) | ecartees | **power-ups** | temoin fantome |
|---|---|---|---|---|---|---|---|---|
| `64e8adfa` CTF | 483 | 13 162 | 495 | 229 | 326 | 169 | **0** | 350 acceptees / 55 retenues -> **5,9** |
| `530820e5` CTF | 265 | 4 832 | 253 | 156 | 197 | 56 | **0** | 150 / **0** -> infini |
| `01e1f945` KOTH | 300 | 5 492 | 283 | 151 | 206 | 77 | **10** | 182 / **0** -> infini |
| `75f1188f` KOTH | 191 | 3 713 | 202 | 101 | 153 | 49 | **8** | 117 / **0** -> infini |

> **`01e1f945` : NEUF creations de `powerup_overshield` (`0xb781197a`) a la position EXACTE
> (0,257 ; -0,003 ; 21,36)** — soit **0,19 m en 3D** du socle que la phase 1 avait localise par
> le croisement des trajectoires des porteurs, sans lire un bit de record de creation.
> La dixieme est a (0,239 ; 6,802 ; 26,89) : c'est le surbouclier LACHE, deja publie par
> l'artefact.
>
> **`75f1188f` : SEPT creations au MEME point, au centimetre pres** (la huitieme a
> (3,698 ; 0,122 ; 21,98), un lacher). Le socle appartient a la CARTE, exactement comme les dix
> socles d'arme de Catalyst.

Instants (horloge du film, secondes) et intervalles entre creations du SOCLE :

```
01e1f945  11,6 · 123,7 · 154,0 · 243,8 · 274,2 · 364,0 · 394,3 · 484,1 · 514,4
          ecarts : 112,1 · 30,3 · 89,8 · 30,4 · 89,8 · 30,3 · 89,8 · 30,3
75f1188f   4,4 · 116,6 · 146,9 · 236,7 · 267,0 · 356,8 · 387,1
          ecarts : 112,2 · 30,3 · 89,8 · 30,3 · 89,8 · 30,3
```

Le MEME motif sur les deux films : un premier ecart de ~112,1 s, puis l'alternance
**30,3 s / 89,8 s**, de periode **120,1 s**. Le 30,3 s tombe exactement sur le pic de 30,5 s
deja mesure sur les socles d'ARME (registre, phase 2 du plan des armes au sol). Ce que le
motif SIGNIFIE (deux entites par cycle au meme point) demande la phase de ramassage : hors
perimetre de ce lot, note en Decouvertes.

**Les deux films CTF ne portent AUCUN power-up**, ni au socle ni ailleurs. Le sous-mode decide
donc de l'armement du socle — ce que disait le temoignage utilisateur.

**Reserve honnete sur le temoin fantome** : le seuil ecrit avant mesure (facteur >= 10) est
ATTEINT sur trois films (la bande fantome ne retient RIEN) et **NON atteint sur `64e8adfa`
(5,9)**. Cela interdirait d'affirmer un POSITIF sur ce film-la ; le resultat y est un NEGATIF,
qu'un filtre trop permissif ne peut que rendre plus difficile a obtenir, jamais plus facile.
Les deux positifs (`01e1f945`, `75f1188f`) viennent de films ou le fantome retient zero.

**3.1 — LE RECENSEMENT DES IMAGES-CLES : H1 REFUTEE.** Instrument :
`powerup_socle_keyframes_test.go`. Lecture : `slots recenses | des la 1re image-cle | a TOUTES
| SANS aucune position delta | a trou interieur`.

| film | images-cles | ti=37 | ti=38 | ti=41 (temoin) | ti=42 |
|---|---|---|---|---|---|
| `64e8adfa` | 43 (0 -> 840,3 s) | 311 \| **0** \| **0** \| 223 \| 0 | 86 \| 0 \| 0 \| 79 \| 0 | 9 \| 0 \| 0 \| **0** \| 0 | 309 \| 0 \| 0 \| 66 \| 1 |
| `530820e5` | 25 (0 -> 480,1 s) | 187 \| **0** \| **0** \| 135 \| 0 | 41 \| 0 \| 0 \| 41 \| 0 | 23 \| 0 \| 0 \| **0** \| 0 | 203 \| 0 \| 0 \| 39 \| 0 |
| `01e1f945` | 28 (0 -> 540,1 s) | 225 \| **0** \| **0** \| 164 \| 0 | 36 \| 0 \| 0 \| 36 \| 0 | 13 \| 0 \| 0 \| **0** \| 0 | 227 \| 0 \| 0 \| 35 \| 0 |
| `75f1188f` | 22 (0 -> 420,1 s) | 170 \| **0** \| **0** \| 132 \| 0 | 36 \| 0 \| 0 \| 36 \| 0 | 9 \| 0 \| 0 \| **0** \| 0 | 170 \| 0 \| 0 \| 32 \| 0 |

**ZERO slot recense a la premiere image-cle, pour TOUS les archetypes et TOUS les films.**
Aucun objet du monde n'est dans l'etat initial : **H1 est REFUTEE**, et H2 avec elle (les
archetypes 36, 39 et 40 n'ont AUCUN slot sur les quatre films).

La population « invisible » est en revanche massive et attendue : 132 a 223 slots `ti=37` sur
170 a 311 n'emettent JAMAIS de position delta (72 % en moyenne), et 100 % des `ti=38`. Le
temoin `ti=41` en compte **zero** — un projectile bouge toujours, et c'est ce qui valide le
compteur.

**Item 3.1 « trou interieur » : `[~]` couvert par 3.2.** Le critere cherchait un slot qui
disparait et revient. La mesure 3.2 montre que le socle prend un **slot NEUF a chaque
reapparition** (1452, 1625, 1701, 1955, 2011, 2274, 2332, 2511, 2563 sur `01e1f945`) : la
notion de trou ne s'applique pas. Un seul trou interieur a ete vu sur tout le corpus
(`64e8adfa`, `ti=42`, slot 1611, retour a 720 s) — sans rapport.

**3.5 — VERDICT**

> **H4 CONFIRMEE, H1 / H2 / H3 REFUTEES.** Le power-up du socle central de Catalyst EXISTE
> dans le film, il est de l'archetype `ti=37`, son identite `eqip` est **`0xb781197a` =
> `powerup_overshield`**, il est cree par un record NEW **A CHAQUE REAPPARITION** (jamais dans
> l'etat initial), a la position **(0,257 ; -0,003 ; 21,36)**, identique au centimetre entre
> les deux films KOTH.
>
> **Pourquoi la production ne le voit pas** : `confirmPlacements` exige que la position de
> creation retombe sur le premier point d'une vie DELTA. Le socle ne bouge jamais, n'a aucune
> vie delta, et se fait ecarter par cet oracle — sur 9 creations, ZERO publiee.

Les deux mesures qui se croisent sont INDEPENDANTES et ne partagent aucun code :
la phase 1 lit des positions de BIPEDE dans l'artefact cuit et n'ouvre aucun film ; la
phase 3.2 lit des records de CREATION d'objet dans les paquets delta du film et ignore tout
des joueurs. **Elles tombent a 0,19 m l'une de l'autre.**

**Gate (3.6)** : `go vet` 0 · `go test ./internal/analysis/...` 0 ·
`golangci-lint run ./internal/analysis/replay/...` 0 issues.

## 8. PHASE DE PRODUCTION — la chaine des socles gagne une entree `ti=37`

> Lot de PRODUCTION du 2026-08-19, execute sous le meme contrat
> (`.claude/skills/plan-execution/SKILL.md`). Worktree `LevelUp-wt-powerup-prod`,
> branche `wt/powerup-prod`.

La correction est petite et son perimetre est net : dans `filmdec/equipment_placements.go`,
l'oracle de vie delta (`MatchEquipmentLife`) est le SEUL filtre de selectivite de la chaine
`ti=37`. Il faut lui ADJOINDRE — jamais le remplacer — la voie que la chaine `ti=42` emploie
deja : **retenir aussi les creations SANS vie delta dont l'identite se resout dans le
manifeste**, et les publier comme des SOCLES, aux memes regles que les armes.

Trois garde-fous que la mesure impose a cette phase :

1. **Le filtre d'identite ne suffit pas partout.** Le temoin fantome rend 5,9 sur `64e8adfa` :
   la publication doit exiger la GRAPPE (>= 2 creations au meme point, `gwPadRadiusM`), qui
   est ce qui a fait la preuve ici (9 creations au meme centimetre).
2. **`t1` n'existe pas pour un objet sans vie delta.** La pose se publie avec `t0` seul, et la
   presence se borne par le recensement des images-cles — exactement ce que `weaponPads` fait
   deja (`PadPresence`).
3. **La ligne R2-P du registre se resout d'elle-meme** : `powerup_overshield` et `powerup_camo`
   figurent dans `POWER_PAD_KEYS` sans membre mesure. Ce lot fournit le membre.

ARBITRAGE DE PUBLICATION (pris avant l'ecriture) : les socles de power-up entrent dans
`weaponPads`, PAS dans un calque neuf. Le socle est le meme objet de jeu, la regle de grappe
groupe deja par (nature, famille) — `gwPadKindPowerup` existait sans membre —, le calque web
`weaponPadsLayer` les dessine sans une ligne de plus, et un second tableau aurait duplique
trois types publies (`PadPresence`, `PadCycle`, `PadPickup`) pour la meme grandeur.

### Items

- `[x]` 8.1 `filmdec` — le RECENSEMENT d'images-cles se lit pour tout archetype d'objet du
  monde (`ScanFilmWorldObjectKeyframes(dir, ti)` / `WorldObjectKeyframes`). Sans lui, la voie
  `ti=37` n'a pas de quoi borner une presence.
- `[ ]` 8.2 `replay` — la REGLE D'IDENTITE d'une chaine de socles devient une regle NOMMEE
  (`padRule`) que la chaine `ti=42` prend telle quelle : meme sortie, ancrage golden inchange.
- `[ ]` 8.3 La voie `ti=37` : trois lectures du film, apparitions dont la famille du manifeste
  commence par `powerup_`, exclusion des creations a vie delta (les lachers), grappe 1 m,
  socle >= 2, bornage par le recensement, cycle depuis le ramassage. Publication dans
  `weaponPads` avec la FAMILLE (`powerup_overshield`) pour identifiant.
- `[ ]` 8.4 COUVERTURE etendue : trois compteurs de la voie `ti=37` (creations acceptees,
  retenues par identite, socles publies) + journal.
- `[ ]` 8.5 `SchemaVersion` 17 + chronique ; `GroundWeaponCoverage` +3 champs ; OpenAPI
  regenere ; `generated.ts` regenere.
- `[ ]` 8.6 TESTS UNITAIRES PURS : grappe, seuil >= 2, exclusion des creations a vie delta,
  famille inconnue = rien. Instrument de mesure du lot REBRANCHE sur la production (pas de
  seconde copie de la regle).
- `[ ]` 8.7 GOLDENS : fixture d'entrees v9 (il porte la lecture `ti=37`) et sortie figee,
  regeneres depuis le film de reference.
- `[ ]` 8.8 TEMOINS re-cuits et ANCRAGE : `01e1f945` / `75f1188f` un socle de power-up au
  centre ; `64e8adfa` / `530820e5` zero ; `000d5950` inchange ; socles d'ARME inchanges.
- `[ ]` 8.9 Gates : `go build ./...`, `go vet ./...`, `go test` (analysis, replaybuild,
  contracttest, archlint), golangci 0, et la chaine web (typecheck, lint, vitest).

### Journal de la phase 8

**8.1 — CLOSE le 2026-08-19.** `filmdec/world_object_census.go` (ex-`ground_weapon_census.go`) :
`ScanFilmWorldObjectKeyframes(dir, ti)` rend `WorldObjectKeyframes` pour NIMPORTE QUEL archetype.
Aucun second walker : la marche d images-cles reste unique, l archetype descend au rang de
parametre. Gate : `go vet ./internal/analysis/...` 0 · `go test ./internal/analysis/{replay,filmdec}/...` 0.

## 7. Decouvertes (a ne PAS traiter dans ce lot)

1. **`64e8adfa` : `bounds` de l'artefact hors carte.** Le milieu des bornes publiees vaut
   (-106,0 ; -93,7) alors que les deux autres films Catalyst tombent a moins de 0,21 m du
   centre. Au moins une position aberrante entre dans `bounds` — le cadrage du rejeu 2D de ce
   match est donc dilate. Constate, NON traite (hors perimetre du lot).
2. **Doc inversee dans `filmdec/equipment_placements.go`.** Le commentaire de
   `EquipmentPlacementStats.Calibration` dit « Bits == 0 : le film n'a pas tranche » ;
   `MPPCalibration` ne porte AUCUN champ `Bits` (le test juste est `Widths.Valid()`).
   Constate, NON traite.
3. **Deux creations par cycle au MEME point, et personne ne sait pourquoi.** Le socle de
   Catalyst rend l'alternance 30,3 s / 89,8 s (periode 120,1 s) sur les deux films KOTH, avec
   des SLOTS DIFFERENTS a chaque fois — donc deux entites distinctes par cycle, pas une
   retransmission. Lecture possible (non mesuree) : le power-up est ramasse peu apres son
   apparition et le socle le remet 30,3 s plus tard. Trancher demande la phase de ramassage
   (`gwPickupHit`), hors perimetre.
4. **`ti=37` porte i30 `equipment-has-infinite-uses-component`**, que le registre documente
   « socle de carte contre equipement de joueur ». Aucun lecteur ne le lit. C'est le canal
   qui distinguerait nativement un objet de SOCLE d'un objet POSE par un joueur, sans passer
   par la proximite d'un poseur. Non instrumente dans ce lot.
5. **Le chemin DELTA n'attribue aucun archetype, et la mesure le rejoue.** Sur `530820e5`,
   `ti=37`, `ti=38` et `ti=42` rendent la MEME plus courte distance au socle (0,91 m) et les
   MEMES slots (1931, 1933, 2288, 2300) : les bandes de slots se recouvrent, et un record
   delta ne DIT pas son archetype. Ce n'est pas une decouverte nouvelle (c'est la cause (2)
   documentee dans `keyframe_ground_weapons.go`), mais elle borne la phase 2 : seule la
   CREATION (qui porte `ti`) et l'IMAGE-CLE attribuent.
