# Plan — Armes au sol, socles et power-ups : deuxieme lecture (positions, spawns, cycle, ramassage)

> Ecrit le 2026-08-18. Sujet utilisateur (item 6 de la file) : « emplacements de spawn de
> power-ups et d'armes speciales sur les cartes, avec leur compteur de reapparition ; et quand
> l'arme est recuperee par un joueur (evenement pick-up, ou disparition de l'emplacement) ».
> **Ce plan est a VALIDER par l'utilisateur avant tout lancement** (decision du 18/08). Il
> couvre les items 11 (« objets au sol : n'afficher que ceux apparus par le mode/la carte,
> discriminant = recurrence spatiale, temoin = nombre de grappes petit et stable ») et 12
> (dispositifs de carte) du cahier des charges Notion. Branche `feat/v75`, principal (films),
> contrat `plan-execution`.

## Ce qui est FAIT (sur pieces, ne pas refaire)

| acquis | ou | portee |
|---|---|---|
| Objectifs de MODE (drapeaux, zones, socles de livraison, crane…) : placement statique depuis `.mvar` | `mapObjectives`, `map_objectives.json` | fait, publie |
| Armes au sol NOMMEES aux images-cles (`ti=42`, familles high-32, 22 armes, 0 fuite d'alias) | `keyframe_ground_weapons.go` (12/08) | fait — un ETAT toutes les ~20 s, pas de position |
| Objets `ti=37` : identite par tag `eqip` (20/21 nommes), positions valides (97,2 % dans l'emprise), ORIGINE (`deployed`/`dropped`), 2 power-ups nommes (`powerup_overshield`, `powerup_camo`) | lots du 17-18/08 | fait — mais **aucun power-up de socle observe** (n = 1 chacun, tous deux portes puis laches) |
| Positions des objets du monde aux LARGEURS DE LA CARTE (`MapQuantEntry`) | `4e2084d8e` (16/08) | fait — **posterieur** a la refutation des positions `ti=42` du 12/08 |
| Calibration d'une largeur de default-state par oracle de position (`CalibrateMPPWidths`), balayage des records de CREATION, vies slot/gen | `equipment_creation*.go` (17-18/08) | fait pour `ti=37` — **transposable a `ti=42`** |
| Le film ne porte AUCUN evenement type pick-up / drop (mesure 12/08) | `keyframe_ground_weapons.go` en-tete | acquis negatif : le ramassage se lira par la DISPARITION |
| Les objets `ti=37` naissent la ou leur porteur MEURT (88,6 % a <= 2 frames / 0,57 m de la fin de vie) | phase G, 18/08 | le miroir attendu du ramassage : un objet DISPARAIT la ou un joueur PASSE |

## Ce qui est A FAIRE — et pourquoi la porte s'est rouverte

La position des armes au sol (`ti=42`) a ete REFUTEE le 12/08 : 62,4 % des slots s'etalaient
au-dela de 20 u, 3,3 % seulement tenaient dans 0,5 u (une arme posee ne bouge pas). Deux
causes etaient invoquees : (1) default-state `ti=42` non resolu au keyframe (offset d'i0
inconnu) ; (2) en delta, bande de slots contaminee par les archetypes voisins. **Depuis** :
le correctif des largeurs (16/08) a transforme les positions d'objets du monde de 0,09 % a
99 % de justesse sur Bazaar — la refutation du 12/08 a ete faite AVEC les largeurs de
Cliffhanger sur les autres cartes ; et la cause (1) est exactement le probleme que la
calibration par oracle a resolu pour `ti=37`. La refutation est donc a REJOUER, pas a croire.


## Apport du worktree concurrent `wt/fusion-lots-go` (lots R3-R6, 17/08 — verifie sur pieces le 18/08)

Quatre lots de recherche menes en parallele sur la base `085cda41b` (docs + instruments, aucun
changement de comportement en production) apportent a CE plan quatre faits qui le precisent :

1. **L'etat par defaut de `ti=42` est DECOMPILE, bit-exact, et ecrit** (`WALK_PORT_NOTES.md`
   § IMAGE-CLE §4, `FUN_1407f0c68`) : `V` ; bloc `multiplayer-properties` (**le MEME
   `consumeMultiplayerPropertiesBlock` que `ti=37`** — donc le meme mot 32 bits = GlobalID de la
   DEFINITION, ici vraisemblablement le tag `weap` de l'arme, et la MEME largeur double 9/5 QP
   vs 8/3 BTB deja calibree par `CalibrateMPPWidths`) ; `R(12)` ; `R(7)` ; bloc de liste
   `FUN_1407f2494` (porte ; si 0 : R(1)[+R(32)] ; sinon n=R(4) x R(1)[+R(32)]) ;
   `ECS_ReadEntityRefIndex5`. Le port (`default_state_ti42.go`) a ete ecrit puis RETIRE a la
   fusion, sur decision superviseur : aucun oracle ne le validait. **La condition de reprise ecrite
   au registre est exactement la phase 0 de ce plan** : « un oracle de position d'arme au sol, ou
   une calibration a la maniere des poses `ti=37` ». => La phase 0.2 ne CALIBRE plus a l'aveugle :
   elle REBRANCHE la grammaire decompilee et la VALIDE par l'oracle de position (le corps du
   record de creation retombe sur le premier point de la vie delta — `equipment_creation_offset_test.go`
   transpose), avec la calibration MPP existante. Bonus attendu : l'IDENTITE de l'arme au sol
   par le mot 32 bits (a croiser avec la famille high-32 des images-cles — deux chaines).
2. **Le corps d'un record d'IMAGE-CLE (type-2) n'est PAS un record NEW et le jeu ne le relit
   JAMAIS** (R5 : <= 1,8 % de marches exactes sur 128 decalages x 16 lectures x 3 films ; R6 : le
   lecteur de film n'a aucun handler pour le type 2, le handler du type 1 saute le bloc). => Les
   POSITIONS d'armes au sol ne se liront PAS a l'image-cle : le chemin est le DELTA (record de
   creation + `ScanFilmWorldObjects(dir, wr, 42)`), comme pour `ti=37`. Le nommage par balayage
   de familles aux images-cles (`keyframe_ground_weapons.go`) reste valable — c'est un ancrage de
   bits, pas une marche.
3. **Corpus** : la contre-liste type-2 par chunk (R6) donne `ti=42` x4 a x21 par chunk sur
   `000d5950`, mais **x0 sur `00502e52` et `07aa428d`** ; `ti=11` x5 sur les deux films CTF
   (`64e8adfa`, `530820e5`). => Choisir des films ou `ti=42` est PRESENT (Arena classique,
   BTB, CTF), pas les Super Fiesta de Bazaar/Illusion.
4. **Un oracle de LARGEUR jamais consomme** : `.ai/V7.5/dumps/kf_capture_sample.txt` (400
   frontieres de records EXACTES — 266 NEW + 134 DELTA — avec leur bit de depart) et son tampon
   `kf_slot0_live.bin` (= le PREMIER paquet delta d'une session, 7 286 o, forme universelle sur
   949 films). => Si des records `ti=42` y figurent, ils valident la grammaire decompilee AVANT
   tout film ; sinon l'oracle de position reste seul.

Prealable d'orchestration : fusionner `wt/fusion-lots-go` dans `feat/v75` (documents + instruments
sous gardes `TI11_FILM` / `KF_GRAM_FILM` / `KFQ_FILM` ; conflits attendus limites a
`REGISTRE_REPORTS.md` et `thought_log.md`, resolus par UNION) — pour que ce lot lise
`WALK_PORT_NOTES.md` §4 dans son propre arbre.

## Decisions tranchees avant execution

1. **On rejoue d'abord la refutation aux bonnes largeurs** (gate 0). Si les positions `ti=42`
   restent du bruit, le plan s'arrete sur la partie « armes » et l'ecrit ; les socles ne se
   deduisent alors QUE des images-cles (etat toutes les ~20 s : quelles armes gisent) — un
   substitut degrade, publie comme tel.
2. **Un socle est une RECURRENCE SPATIALE**, pas une lecture de fichier de carte (Notion 11) :
   une grappe de positions ou des objets de MEME famille apparaissent a plusieurs reprises,
   sans poseur (pas de mort a proximite au moment de l'apparition — c'est le NEGATIF de la
   regle `dropped`). Temoin : le nombre de grappes est petit et stable (6 a 12 sur une arene) ;
   les positions de mort des joueurs (les lachers) NE forment PAS de grappes recurrentes de
   meme famille au meme endroit — sinon le critere ne separe rien.
3. **Le cycle de reapparition se MESURE** : ecart entre deux apparitions successives de la
   meme famille sur le meme socle (mediane, p10-p90) ; un socle dont le cycle n'est pas
   stable (ecart-type > 20 % de la mediane) publie « cycle non etabli », pas un chiffre.
4. **Le ramassage = disparition + proximite** : la fin de vie d'un objet au sol (`t1` du slot,
   ou dernier keyframe ou il est vu) coincide avec un joueur a < 1,5 m dans les 2 frames — le
   miroir de la regle `dropped` (memes seuils, ecrits avant mesure). Temoin : la disparition
   d'un objet SANS joueur a proximite (respawn de socle qui « recycle » l'objet ? destruction ?)
   doit etre rare et se lister. Le joueur ramasseur = le plus proche a < 1,5 m ; si deux joueurs
   a < 1,5 m, `unknown`.
5. **Power-ups de socle** : memes regles, sur les familles `powerup_*` de `ti=37` (les 2 deja
   nommees + celles que la chaine `sofa` nommera si de nouveaux identifiants apparaissent) ; le
   corpus des 12 films n'en porte aucun de socle — il faut CHOISIR des films qui en ont (modes
   Arena avec power-ups : registre par snapshot parquet, playlists ranked/arena, cartes connues
   pour porter surbouclier/camo — l'utilisateur peut en designer).
6. **Rendu (lot ulterieur, apres validation des donnees)** : socle = icone de l'arme/du
   power-up a la position, etat « present / vide », compte a rebours du cycle si etabli ;
   ramassage = l'icone s'eteint et le nom de l'arme apparait sur la fiche du ramasseur (les
   loadouts d'images-cles le confirmeront a la prochaine image-cle). Aucun rendu dans ce plan.
7. Regles inchangees : seuils avant mesure, un seul decodage filmdec par process, aucune base
   en ecriture, JAMAIS `git add -A`, jamais d'attente passive, decouvertes au registre.

## Phases

### Phase 0 — REJOUER la refutation des positions `ti=42` aux largeurs de la carte

- [ ] 0.1 Reprendre l'instrument du 12/08 (`replay/ground_weapon_research_test.go`, garde
      `GW_FILM`) avec `MapQuantEntry` de la carte installee (patron `installWorldObjectPrecision`)
      : temoin fantome, part des slots reels tenant dans 0,5 u a >= 3 echantillons, part
      au-dela de 20 u. Sur >= 4 films de >= 3 cartes (dont Cliffhanger, ou le defaut etait
      juste — temoin de non-regression).
- [ ] 0.2 REBRANCHER la grammaire DECOMPILEE de l'etat par defaut `ti=42` (WALK_PORT_NOTES §4,
      `FUN_1407f0c68`, retiree a la fusion R5 faute d'oracle) et la VALIDER par l'oracle de
      position (le corps du record de creation retombe sur le premier point de la vie delta —
      transposition de `equipment_creation_offset_test.go`), avec la calibration MPP existante
      (9/5 QP, 8/3 BTB). Publier avant/apres et l'IDENTITE (mot 32 bits -> tag `weap`) croisee
      avec la famille high-32 des images-cles. Si l'oracle ne valide pas : la grammaire reste
      ecrite et NON branchee, comme la fusion R5 l'a decide — c'est la cause (1) du 12/08.
- [ ] 0.3 Publier AVANT/APRES avec denominateurs.

**Gate 0** : si la part des slots stables (0,5 u) passe d'un ordre de grandeur et que le
temoin fantome reste ce qu'il est, la position est REHABILITEE ; sinon le negatif s'ecrit et
la partie « armes » se rabat sur les images-cles (decision 1).

### Phase 1 — SOCLES par recurrence spatiale, et CYCLE

- [ ] 1.1 Sur les vies d'objets au sol (`ti=42` avec famille high-32 lue ; `ti=37` power-ups) :
      classer chaque apparition `dropped` (mort a proximite, regle du 18/08 en miroir) ou
      `spawned` (aucune mort a < 1,5 m dans les 2 frames) — distribution publiee.
- [ ] 1.2 Grapper les apparitions `spawned` par famille et position (rayon 1 m) ; compter les
      grappes par carte ; temoin de Notion 11 (petit et stable, 6-12 sur une arene) ; temoin
      negatif : les positions de mort ne forment pas de grappes de meme famille.
- [ ] 1.3 Cycle : ecart entre apparitions successives par grappe ; mediane / p10 / p90 ;
      « cycle non etabli » si instable. Comparer aux cycles connus du jeu quand ils existent
      (armes de puissance ~2-3 min en Arena — source Waypoint si l'utilisateur en a une).
- [ ] 1.4 Publier au registre les socles par carte (position, famille, cycle) comme
      donnee de REFERENCE derivee des films (versionnee ? — a decider apres mesure : si les
      socles sont stables entre films d'une meme carte, un catalogue `map_weapon_pads.json`
      se justifie ; sinon ils restent par match).

**Gate 1** : grappes petites et stables sur >= 3 cartes, temoin negatif tenu, cycles publies.

### Phase 2 — RAMASSAGE par disparition + proximite

- [ ] 2.1 Pour chaque vie d'objet au sol : instant de disparition, joueur le plus proche a
      < 1,5 m dans les 2 frames -> ramasseur ; sinon `unknown`. Distribution des distances
      (mediane, p90) et temoin (distance au joueur le plus proche a un instant tire au
      hasard dans la vie de l'objet).
- [ ] 2.2 CONTROLE INDEPENDANT : le loadout d'images-cles du ramasseur doit porter la famille
      ramassee a l'image-cle suivante (`keyframe_loadout.go`, 98,3 % de temoin croise) — c'est
      l'oracle du ramassage. Publier le taux d'accord et un temoin (joueur au hasard).
- [ ] 2.3 Publier : `weaponPickups` {t, x, y, family, slot ramasseur | -1} et l'etat des
      socles dans le temps.

**Gate 2** : accord avec l'oracle des loadouts >= 90 % sur les ramassages a ramasseur
identifie ; sinon `[!]` avec la mesure.

### Phase 3 — PUBLICATION (schema 11) et note UI

- [ ] 3.1 Document : `weaponPads` (socles : position, famille, cycle, etats presence dans le
      temps), `weaponPickups` ; `SchemaVersion` chronique ; contrat, OpenAPI, `generated.ts`,
      golden, couverture ; temoins re-cuits.
- [ ] 3.2 Note UI (decision 6) pour l'utilisateur ; aucun rendu ici.

## Regles dures

Refutation rejouee AVANT tout socle ; socle = recurrence mesuree ; cycle publie seulement
s'il est stable ; ramassage seulement avec oracle ; aucun rendu ; commits sur `feat/v75`, pas
de push. **Lancement : sur validation utilisateur de ce plan.**
