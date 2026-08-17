# Plan — Armes au sol, socles et power-ups : deuxieme lecture (positions, spawns, cycle, ramassage)

> Ecrit le 2026-08-18. Sujet utilisateur (item 6 de la file) : « emplacements de spawn de
> power-ups et d'armes speciales sur les cartes, avec leur compteur de reapparition ; et quand
> l'arme est recuperee par un joueur (evenement pick-up, ou disparition de l'emplacement) ».
> **VALIDE par l'utilisateur le 2026-08-17** (« go pour l'implem pilotee » sur le handoff
> superviseur qui le portait). Il couvre les items 11 (« objets au sol : n'afficher que ceux
> apparus par le mode/la carte, discriminant = recurrence spatiale, temoin = nombre de grappes
> petit et stable ») et 12 (dispositifs de carte) du cahier des charges Notion. Branche
> `feat/v75`, worktree PRINCIPAL (films), contrat `plan-execution`. Chemins verifies le
> 2026-08-17 : instruments sous `apps/go-api/internal/analysis/filmdec/` (`keyframe_ground_weapons.go`,
> `keyframe_loadout.go`, `equipment_creation*.go`, `default_state*.go`, `projectiles.go` pour
> `ScanFilmWorldObjects`) et `apps/go-api/internal/analysis/replay/` (`ground_weapon_research_test.go`
> garde `GW_FILM`, `world_object_precision.go` pour `installWorldObjectPrecision`) ; grammaire
> decompilee dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` § IMAGE-CLE (present dans `feat/v75`
> apres la fusion de `wt/fusion-lots-go`).
>
> **Selection des films — tranchee** : l'agent choisit lui-meme, sur preuve, des films ou
> `ti=42` est PRESENT (contre-liste type-2 par chunk de R6, ou un balayage `ScanFilmWorldObjects(dir,
> wr, 42)` sur les films du corpus) et, pour les power-ups, des films de modes Arena/ranked
> (registre des matchs : mode, playlist, carte). Les deux questions posees a l'utilisateur
> (films a power-ups a designer, source officielle des cycles) sont FACULTATIVES : sans reponse,
> les power-ups de socle se cherchent dans le corpus et, s'il n'en porte aucun, l'item 1.1
> l'ecrit comme negatif de corpus (pas comme echec) ; les cycles se MESURENT sans reference
> officielle (decision 3).

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
la cause (1) est exactement le probleme que la calibration par oracle a resolu pour `ti=37`, et
la grammaire `ti=42` est decompilee (R5). La refutation est donc a REJOUER, pas a croire.

> **Correction du 2026-08-17 (verifiee sur pieces, `CLE_USB_REJEU_2D.md` l. 70)** : `000d5950` est
> **Cliffhanger** (Super Fiesta Slayer), pas Bazaar (Bazaar Super Fiesta = `00502e52`). La
> refutation du 12/08 (`GW_MAP=Cliffhanger` sur `000d5950`) a donc ete faite avec les BONNES
> largeurs pour ce film : le correctif des largeurs du 16/08 (0,09 % -> 99 % sur Bazaar) ne la
> change PAS sur `000d5950`. Ce qui rouvre la porte, c'est la cause (1) — le default-state
> `ti=42`, aujourd'hui decompile et calibrable par oracle — et le fait que le verdict n'existe
> que sur UN film. La phase 0.1 est donc une LIGNE DE BASE multi-cartes (le « AVANT »), pas un
> levier ; le levier est la phase 0.2 ; le gate 0 se juge sur l'APRES.


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

- [x] 0.1 Reprendre l'instrument du 12/08 (`replay/ground_weapon_research_test.go`, garde
      `GW_FILM`) avec `MapQuantEntry` de la carte installee (patron `installWorldObjectPrecision`)
      : temoin fantome, part des slots reels tenant dans 0,5 u a >= 3 echantillons, part
      au-dela de 20 u. Sur >= 4 films de >= 3 cartes (dont Cliffhanger, ou le defaut etait
      juste — temoin de non-regression).
      **FAIT** : 6 films / 5 cartes, tableau AVANT au journal ci-dessous. Le verdict du 12/08 se
      reproduit AU CHIFFRE PRES sur `000d5950` (458 slots, 3,3 %, 62,4 %) et tient partout
      ailleurs (1,7 % a 4,8 %). Films choisis SUR PREUVE (recensement `ti=42` de 18 films,
      `replay/ground_weapon_corpus_test.go`, garde `GW_CORPUS`).
- [x] 0.2 REBRANCHER la grammaire DECOMPILEE de l'etat par defaut `ti=42` (WALK_PORT_NOTES §4,
      `FUN_1407f0c68`, retiree a la fusion R5 faute d'oracle) et la VALIDER par l'oracle de
      position (le corps du record de creation retombe sur le premier point de la vie delta —
      transposition de `equipment_creation_offset_test.go`), avec la calibration MPP existante
      (9/5 QP, 8/3 BTB). Publier avant/apres et l'IDENTITE (mot 32 bits -> tag `weap`) croisee
      avec la famille high-32 des images-cles. Si l'oracle ne valide pas : la grammaire reste
      ecrite et NON branchee, comme la fusion R5 l'a decide — c'est la cause (1) du 12/08.
      **FAIT, ET L'ORACLE VALIDE** : `filmdec/default_state_ti42.go` + entree `42:` de
      `defaultStateDeserByTI`. 282 atterrissages exacts sur 289 (97,6 %) contre 0/289 pour trois
      deserialiseurs faux ; identite 937/947 (98,9 %). Detail au journal.
- [x] 0.3 Publier AVANT/APRES avec denominateurs. **FAIT** — journal du plan, `.ai/thought_log.md`,
      `REGISTRE_REPORTS.md` (3 lignes amendees), `WALK_PORT_NOTES.md` §4.

**Gate 0** : si la part des slots stables (0,5 u) passe d'un ordre de grandeur et que le
temoin fantome reste ce qu'il est, la position est REHABILITEE ; sinon le negatif s'ecrit et
la partie « armes » se rabat sur les images-cles (decision 1).

> **GATE 0 — NON ATTEINT SUR SON CRITERE, ET LE VERROU EST POURTANT LEVE.** Les deux enonces
> sont vrais en meme temps, et le seuil n'a pas ete rebaisse :
>
> - le critere ecrit (part des vies stables a 0,5 u : 3,3 % -> >= 33 %) **N'EST PAS ATTEINT** :
>   3,2 % -> 6,2 % (62/1 965 -> 71/1 140). Il double, il ne change pas d'ordre de grandeur ;
> - la cause (1) du 12/08 — le default-state `ti=42` — **EST LEVEE**, mesuree par deux oracles
>   independants (atterrissage 97,6 %, identite 98,9 %) ;
> - et la mesure explique POURQUOI le critere ne pouvait pas etre atteint : un objet pose cesse
>   d'emettre sa position (acquis `ti=37`, `EquipmentLifeSpan`), donc les echantillons delta
>   d'une arme au sol sont sa phase MOBILE. Mesurer leur immobilite est une contradiction dans
>   les termes. La position publiable est celle du record de CREATION, pas la dispersion des
>   deltas — et ce record est desormais lisible.
>
> **Arbitrage utilisateur requis avant la phase 1** (l'agent s'arrete ici, conformement au
> brief) : appliquer la decision 1 telle quelle (repli sur les images-cles), ou amender la
> phase 1 pour qu'elle parte des records de CREATION (`ScanFilmGroundWeaponCreations`) plutot
> que de la dispersion des deltas. Le second chemin est celui que la mesure designe.

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
      donnee de REFERENCE derivee des films. **Critere tranche** : si, sur une carte vue dans
      >= 2 films, >= 80 % des grappes d'un film retrouvent une grappe de MEME famille a < 1 m
      dans l'autre, les socles sont une propriete de la carte -> catalogue versionne
      `map_weapon_pads.json` (au meme endroit et au meme format d'ecriture que
      `map_objectives.json`), alimente par les films ; sinon ils restent PAR MATCH (publies
      dans le document de rejeu seulement) et le registre le dit.

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
de push. Lancement valide le 2026-08-17.

## Contrat d'execution (rappel, `plan-execution` fait foi)

- Une phase a la fois, dans l'ordre ; une phase est CLOSE quand son gate a tourne dans la
  session (sortie collee au journal du plan ci-dessous), tous ses items sont statues
  (`[x]` / `[~]` ref / `[!]` justifie), le fichier plan est mis a jour et commite avec le lot,
  et l'entree `.ai/thought_log.md` existe.
- Les gates d'arret sont REELS : un gate 0 negatif ARRETE la partie « armes » (decision 1) et
  la phase 1 ne s'ouvre que sur les images-cles ; un gate 2 negatif publie `[!]` avec la mesure.
- Seuils ecrits ci-dessus, jamais rebaisses apres mesure. Denominateurs toujours publies.
- Aucun `git add -A` (fichiers d'autres sessions dans l'arbre) ; aucune attente passive ;
  aucun fix hors perimetre — les decouvertes vont dans la section ci-dessous.
- Un seul decodage filmdec par process ; `installWorldObjectPrecision` restaure toujours ;
  aucune base en ecriture.

## Decouvertes (hors perimetre — notees, NON traitees)

1. **La contre-liste type-2 de R6 (« `ti=42` x0 sur `00502e52` et `07aa428d` ») ne dit PAS ce
   qu'on lui faisait dire.** Recensement direct des images-cles (`GW_CORPUS`, 18 films) :
   `00502e52` porte **157** records `ti=42` (150 slots) et `07aa428d` **181** (150 slots). Le
   « x0 » de R6 portait sur le paquet RECONCILIE de son propre instrument, pas sur le walker
   d'images-cles. **`ti=42` est present sur les 18 films testes, sans exception** — la selection
   de films n'a donc pas a se soucier du mode de jeu. Aucune correction faite au document R6.
2. **Le temoin FANTOME du balayage de creations n'est pas discriminant a lui seul**, et il faut
   le dire avant que quelqu'un s'appuie dessus : sur `00162144`, la bande fantome rend **398**
   creations acceptees contre **366** pour la bande reelle (elle porte 27 563 ancres contre
   6 485). Sur les cinq autres films le reel devance le fantome d'un facteur 1,9 a 3,7. Ce qui
   rend les creations credibles n'est donc PAS le taux d'acceptation, c'est le croisement
   d'identite avec les images-cles (98,9 %). **La phase 1 doit filtrer les creations par ce
   croisement (ou par la bande d'images-cles), jamais par l'acceptation seule.**
3. **`gwWorldRange` (instrument du 12/08) ecrivait `filmdec.WorldObjectPrecision` sans verrou ni
   restauration** — un global de paquet. Corrige DANS le perimetre (le meme instrument est
   rejoue ici) : `LockProcessDecode` + `t.Cleanup`. A verifier sur les autres instruments sous
   garde qui installent des largeurs, non audites ici.
4. **`.ai/V7.5/dumps/kf_capture_sample.txt` : 195 frontieres sur 400 ne se reconcilient pas**
   avec un en-tete de 18 ou 16 bits portant l'identifiant releve, et des `ti` >= 50 apparaissent
   (53, 61, 63) alors que le jeu les cappe a 50. Le releve est donc partiellement desaligne ou
   partiel. Non traite : l'oracle etait de toute facon epuise par le manque de records `ti=42`.

## Branches utilisateur fusionnees AVANT le lancement (`66e867b80`, 2026-08-17)

- `wt/fusion-finale` (lignee R7 kf35/kfd/kfe/kfc, polarite i9 corrigee dans `traverse.go`/
  `components_batch7.go`, `keyframe_fullstate_loop.go`) et `wt/poses-revue-fix` (correctif de la
  revue des poses, `PLAN_CORRECTIF_REVUE_POSES.md`) sont DANS `feat/v75` : la phase 0 mesure sur
  le decodeur definitif. La RE de l'image-cle est ARRETEE (decision utilisateur apres R7-e).
- **Acquis du correctif de revue, decisif pour la phase 2** : `t1` d'une piste d'objet = fin du
  flux de POSITION (mise au repos), PAS la disparition ; **le record DEL n'est PAS isolable**
  (78 090 / 158 098 candidats pour 477 / 993 vies) ; le **recensement des images-cles** (walker
  durci, 249/250) prouve la SURVIE d'un objet mais est espace de 20,0 s. => La decision 4 se lit
  ainsi : la disparition d'un objet au sol se BORNE par les images-cles (derniere ou il est
  recense, premiere ou il ne l'est plus) et se DATE, dans cet intervalle, par le passage d'un
  joueur a < 1,5 m (le ramasseur = ce joueur ; aucun passage = `unknown`, date = borne haute) ;
  l'oracle du loadout a l'image-cle suivante reste le controle independant. Seuils inchanges.

## Journal du plan (avancement, source de verite pour la reprise)

- 2026-08-17 — plan valide par l'utilisateur ; fusion de `wt/fusion-lots-go` (`d2e46eb5d`) puis
  fusion utilisateur de `wt/fusion-finale` + `wt/poses-revue-fix` (`66e867b80`) ; **phase 0
  LANCEE** sur le worktree principal (agent Opus, brief du superviseur).
- 2026-08-17 — **phase 0 CLOSE. Gate 0 non atteint sur son critere ; le verrou du 12/08 est leve.**
  Tout ci-dessous a tourne dans la session, aux largeurs de chaque carte, `CGO_ENABLED=0`,
  un film par process.

### Selection des films — SUR PREUVE (recensement `ti=42`, 18 films)

Instrument : `replay/ground_weapon_corpus_test.go` (garde `GW_CORPUS`). `ti=42` est PRESENT sur
les 18 films testes (62 a 650 records d'image-cle). Cartes lues sans ouvrir aucune base : instantane
parquet `data/backups/staging/halo_infinite/shared_matches_v2/match_registry_20260711_090652.parquet`
(un fichier, pas une base — le serveur local tenait les DuckDB en ecriture).

Six films retenus, cinq cartes, cinq decoupages d'axe DIFFERENTS :

| film | carte | mode | largeurs | records `ti=42` |
|---|---|---|---|---:|
| `000d5950` | Cliffhanger | Slayer Super Fiesta | `[13 13 14]` (= defaut, TEMOIN) | 269 |
| `bcb6d393` | Cliffhanger | CTF:Arena | `[13 13 14]` | 441 |
| `01e1f945` | Catalyst | KOTH:Arena | `[15 15 15]` | 420 |
| `7f1bbf06` | Streets | KOTH:Arena | `[12 12 12]` | 275 |
| `b8d1fe0c` | Recharge | CTF Neutral Flag | `[18 18 15]` | 163 |
| `00162144` | Smallhalla | Slayer:Arena | `[15 15 17]` | 493 |

### 0.1 AVANT — la refutation du 12/08, rejouee aux largeurs de la carte

`GW_FILM` / `GW_BOUNDS` / `GW_MAP`, `TestGroundWeaponCoverage`. Dispersion par SLOT, slots a
>= 3 echantillons.

| film | eligibles | <= 0,5 u | > 20 u | fantome eligibles | fantome <= 0,5 u | fantome > 20 u |
|---|---:|---:|---:|---:|---:|---:|
| `000d5950` | 458 | 15 (3,3 %) | 286 (62,4 %) | 188 | 1 (0,5 %) | 155 (82,4 %) |
| `bcb6d393` | 255 | 9 (3,5 %) | 118 (46,3 %) | 37 | 1 (2,7 %) | 36 (97,3 %) |
| `01e1f945` | 467 | 10 (2,1 %) | 341 (73,0 %) | 240 | 6 (2,5 %) | 197 (82,1 %) |
| `7f1bbf06` | 225 | 5 (2,2 %) | 127 (56,4 %) | 45 | 2 (4,4 %) | 43 (95,6 %) |
| `b8d1fe0c` | 119 | 2 (1,7 %) | 48 (40,3 %) | 24 | 0 (0,0 %) | 23 (95,8 %) |
| `00162144` | 441 | 21 (4,8 %) | 241 (54,6 %) | 210 | 0 (0,0 %) | 173 (82,4 %) |
| **TOTAL** | **1 965** | **62 (3,2 %)** | **1 161 (59,1 %)** | 744 | 10 (1,3 %) | 627 (84,3 %) |

`000d5950` reproduit le verdict du 12/08 AU CHIFFRE PRES (458 / 15 / 3,3 % / 286 / 62,4 % ;
fantome 188 / 1 / 155). **La refutation n'etait pas un artefact de largeurs** — elle tient sur
cinq cartes dont quatre ont un decoupage different du defaut.

### 0.2 — la grammaire `ti=42`, branchee parce que l'oracle la valide

**Ecrit** : `filmdec/default_state_ti42.go` (`consumeDefaultStateTI42` = `V` ; `consumeDefault
StateTI36` ; `R(12)` ; `R(7)` ; `consumeWeaponMagazineList` ; `consumeGate0R(5)`), entree `42:`
dans `defaultStateDeserByTI`. **Doublon `FUN_1407f2494` tranche par REUTILISATION** : la
grammaire decompilee est celle de `consumeWeaponMagazineList` feuille pour feuille (le fichier a
bouge depuis le 26/07 : `components_object.go`, plus `unit_weaponstate.go`).

**Oracle de LARGEUR live : EPUISE (negatif publie).** `default_state_ti42_oracle_test.go`
(`TI42_CAPTURE` / `TI42_BUFFER`) : 400 frontieres, 205 en-tetes reconcilies, **un seul** record
NEW `ti=42` (corps 971 bits, porte OUVERTE) et **aucun** `ti=37`. Cet oracle ne pouvait pas
valider quoi que ce soit ; le negatif est garde rejouable.

**Oracle de POSITION : VALIDE.** `ground_weapon_creation_offset_test.go` (`GW_CREATION_FILM`),
transposition de `equipment_creation_offset_test.go` avec `ti=37` en CONTROLE POSITIF par le
MEME code. On localise le corps par la position, puis on DEROULE le deserialiseur depuis
l'en-tete NEW et on exige l'atterrissage AU BIT PRES sur le masque.

| film | records `ti=42` ancres | ti42 porte | temoin ti37 | temoin ti36 | temoin ti38 |
|---|---:|---:|---:|---:|---:|
| `000d5950` | 119 | **118 (99,2 %)** | 0 | 0 | 0 |
| `01e1f945` | 99 | **97 (98,0 %)** | 0 | 0 | 0 |
| `00162144` | 71 | **67 (94,4 %)** | 0 | 0 | 0 |
| **TOTAL** | **289** | **282 (97,6 %)** | **0** | **0** | **0** |

Controle croise sur les records `ti=37` des memes films : le deser `ti=42` atterrit **0 / 265**,
le deser `ti=37` **176 / 265**. **PIEGE DE LECTURE ecarte** : la distance en-tete -> masque ne
vaut PAS le chemin minimal (105 bits) ; les pics mesures sont a 118, 150 ou 182 selon le film et
se decomposent exactement en portes de la grammaire (+5 reference d'entite, +8 prefixe de
version, +13 champ optionnel MPP, +32 identifiant). Seul l'atterrissage tranche.

**IDENTITE — deuxieme oracle, independant du premier.** Le mot de 32 bits du bloc MPP du record
de creation, croise avec la famille high-32 lue aux IMAGES-CLES pour la meme vie (slot, gen) :

| film | paires croisees | accord | mots distincts | dont au catalogue d'armes |
|---|---:|---:|---:|---:|
| `000d5950` | 173 | 169 (97,7 %) | 105 | 220 / 317 |
| `bcb6d393` | 118 | 118 (100 %) | 45 | 159 / 219 |
| `01e1f945` | 215 | 213 (99,1 %) | 97 | 291 / 405 |
| `7f1bbf06` | 142 | 140 (98,6 %) | 55 | 166 / 210 |
| `b8d1fe0c` | 72 | 70 (97,2 %) | 21 | 86 / 118 |
| `00162144` | 227 | 227 (100 %) | 82 | 278 / 366 |
| **TOTAL** | **947** | **937 (98,9 %)** | | |

Les mots resolvent en armes reelles (Gravity Hammer, Sentinel Beam, Energy Sword, M41 SPNKr,
MA40 AR, Mk51 Sidekick, S7 Sniper...). **Le mot de 32 bits du bloc MPP EST l'identite de l'arme
au sol**, comme il est le GlobalID du tag `eqip` pour `ti=37`.

### 0.3 APRES — ce que la bande CONFIRMEE change, et ce qu'elle ne change pas

`ground_weapon_after_test.go` (`GW_FILM` / `GW_BOUNDS` / `GW_MAP`). Un slot n'est plus PRESUME
`ti=42` par la bande d'images-cles : il est PROUVE par un record de creation qui porte son
`R(6) typeIndex`. Dispersion a la granularite d'une VIE (slot, generation) — la cle de tout le
decodeur d'objets du monde, que l'AVANT ne pouvait pas prendre faute de savoir quelles
generations etaient des armes.

| film | creations acceptees (fantome) | vies eligibles | <= 0,5 u | <= 2 u cumule | > 20 u |
|---|---|---:|---:|---:|---:|
| `000d5950` | 317 (164) | 237 | 15 (6,3 %) | 119 (50,2 %) | 41 (17,3 %) |
| `bcb6d393` | 219 (125) | 150 | 9 (6,0 %) | 97 (64,7 %) | 19 (12,7 %) |
| `01e1f945` | 405 (147) | 273 | 16 (5,9 %) | 108 (39,6 %) | 86 (31,5 %) |
| `7f1bbf06` | 210 (57) | 142 | 6 (4,2 %) | 84 (59,2 %) | 13 (9,2 %) |
| `b8d1fe0c` | 118 (35) | 90 | 3 (3,3 %) | 58 (64,4 %) | 9 (10,0 %) |
| `00162144` | 366 (398) | 248 | 22 (8,9 %) | 142 (57,3 %) | 49 (19,8 %) |
| **TOTAL** | **1 635 (926)** | **1 140** | **71 (6,2 %)** | **608 (53,3 %)** | **217 (19,0 %)** |

**AVANT -> APRES, memes films, memes largeurs, meme chaine de decodage** :

| grandeur | AVANT (par slot presume) | APRES (par vie confirmee) |
|---|---:|---:|
| eligibles (>= 3 echantillons) | 1 965 | 1 140 |
| **stables <= 0,5 u** | **62 (3,2 %)** | **71 (6,2 %)** |
| sous 2 u (cumule) | 501 (25,5 %) | 608 (53,3 %) |
| **etales > 20 u** | **1 161 (59,1 %)** | **217 (19,0 %)** |

Le bruit s'effondre (59,1 % -> 19,0 %) et la concentration double (25,5 % -> 53,3 %), mais la
part STRICTEMENT stable ne fait que doubler : **3,2 % -> 6,2 %**, contre >= 33 % exige. Le seuil
n'est pas rebaisse : **le gate 0 n'est pas atteint.**

**LE TEMOIN DIT POURQUOI, et il faut le lire.** Les vies NON confirmees des memes slots (memes
films, meme code) : 139 eligibles, **9 sous 0,5 u (6,5 %)** et 105 au-dela de 20 u (75,5 %). Sur
le critere > 20 u la confirmation separe nettement (19,0 % contre 75,5 %) ; **sur le critere
0,5 u elle ne separe RIEN** (6,2 % contre 6,5 %). Le discriminant « une arme posee ne bouge pas »
ne discrimine pas parce qu'il ne peut pas : un objet qui se pose CESSE d'emettre sa position
(acquis `ti=37`, cf. `EquipmentLifeSpan` et `splitLives`), donc tout echantillon delta d'une arme
au sol appartient a sa phase MOBILE. Le critere du 12/08 mesurait l'immobilite sur le seul
sous-ensemble de records qui n'existe que parce qu'il y a eu mouvement.
