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

### Phase 1 — L'oracle des cinq ramassages `[ ]`

- `[ ]` 1.1 Lire `equipmentEpisodes` + `tracks[].points[]` de `01e1f945` ; rendre la position
  du bipede de chaque episode a son `t0` (et l'encadrement le plus proche si `t0` n'est pas
  un point).
- `[ ]` 1.2 Dispersion des 5 points : centroide, rayon max, ecart au centre de la carte.
- `[ ]` 1.3 TEMOIN : 5 instants tires au hasard sur les MEMES bipedes, par le MEME code.
- `[ ]` 1.4 Verdict d'etape : la position du socle est-elle MESUREE ? Si oui, elle devient la
  cible des phases 2 et 3 (boite 3 m autour d'ELLE, en plus du centre geometrique).
- `[ ]` 1.5 Gate : `go vet` + `go test ./internal/analysis/...` + instrument joue sur
  `01e1f945`.

### Phase 2 — Les archetypes dans la boite (H2 / H3) `[ ]`

- `[ ]` 2.1 `ScanFilmWorldObjects` pour ti = 36..42 sur les 4 films ; ne garder que les vies
  dont un point tombe dans la boite large. Par archetype : nombre de vies, t du premier et du
  dernier point, etendue spatiale, z.
- `[ ]` 2.2 Presence a t=0 : vies dont le premier point est a t <= 5 s.
- `[ ]` 2.3 Disparitions liees a un joueur : dernier point d'une vie de la boite avec un
  bipede a < 1,5 m dans les 2 images. Sur `01e1f945`, croiser avec les 5 instants de l'oracle.
- `[ ]` 2.4 Reapparitions : vies successives au meme point (<= 1 m) ; ecarts, stabilite.
- `[ ]` 2.5 Gate : `go vet` + `go test ./internal/analysis/...` + instrument joue sur 4 films.

### Phase 3 — Images-cles, creations, identite (H1 / H4) et verdict `[ ]`

- `[ ]` 3.1 Recensement `WalkKeyframeWorld` : par image-cle et par ti, nombre de slots ; les
  slots presents DES LA PREMIERE image-cle et a TOUTES ; ceux qui n'ont AUCUNE position delta
  (population « invisible », la forme meme de H1) — par archetype, avec denominateurs.
- `[ ]` 3.2 Creations dans la boite : `ScanFilmEquipmentCreationsForBand` (ti=37) et
  `...GroundWeaponCreationsForBand` (ti=42), largeurs MPP CALIBREES par film (jamais les
  defauts), sans l'oracle de vie delta. Identite `MPPVal[MPPWord32]` -> famille du manifeste
  (`replay_labels.toml`, `powerup_overshield` `0xb781197a`, `powerup_camo` `0xe7be9f5c`).
- `[ ]` 3.3 Temoin fantome sur 3.2, meme code, meme cardinalite.
- `[ ]` 3.4 Croisement des 4 films : un identifiant present sur les 4 ? un identifiant qui
  CHANGE entre CTF et KOTH (signal du sous-mode) ?
- `[ ]` 3.5 Verdict : hypothese retenue (ou negatif ecrit avec ses chiffres) ; ce que cela
  implique pour la chaine des socles — UNE phase de production A PROPOSER, pas a faire.
- `[ ]` 3.6 Gate : `go vet` + `go test ./internal/analysis/...` + golangci 0.

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

## 7. Decouvertes (a ne PAS traiter dans ce lot)

1. **`64e8adfa` : `bounds` de l'artefact hors carte.** Le milieu des bornes publiees vaut
   (-106,0 ; -93,7) alors que les deux autres films Catalyst tombent a moins de 0,21 m du
   centre. Au moins une position aberrante entre dans `bounds` — le cadrage du rejeu 2D de ce
   match est donc dilate. Constate, NON traite (hors perimetre du lot).
