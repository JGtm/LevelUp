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
>
> **ARBITRAGE SUPERVISEUR (2026-08-17, CR verifie sur pieces : commits `e7aa61494`, `6603eeaf8`,
> `de1de707b`, entree `42:` de `default_state_arch.go:61`, gates `EXIT_*=0`)** : la phase 1 est
> AMENDEE pour partir des records de CREATION. Motif ecrit : le seuil du gate 0 n'est pas rebaisse
> (il reste NON ATTEINT et le plan le dit) ; mais la condition que la decision 1 traitait — « les
> positions `ti=42` restent du bruit » — est CONTREDITE par une validation positionnelle plus
> forte que le critere de dispersion : 282/289 atterrissages exacts du corps de creation sur la
> position de la vie (0/289 sur trois archetypes temoins), identite 937/947. Le critere de
> dispersion mesurait l'immobilite de la seule phase mobile : il etait mal pose, et son propre
> temoin (6,5 %) le montre. Le repli « images-cles seules » n'a donc plus d'objet. Ce choix est
> une decision de PERIMETRE (entree de la phase 1), pas un seuil ; l'utilisateur peut le renverser.
> Garde-fou herite de la decouverte 2 : une creation ne compte QUE si son identite est croisee
> (mot MPP resolu en famille d'arme connue, ou egal a la famille high-32 du meme slot a l'image-cle
> voisine) — jamais par la seule acceptation du balayage (fantome 398 vs reel 366 sur `00162144`).

### Phase 1 — SOCLES par recurrence spatiale, et CYCLE (entree = records de CREATION `ti=42`)

- [x] 1.0 Entree de la phase : les APPARITIONS = records de creation `ti=42`
      (`ScanFilmGroundWeaponCreations`, position i0 aux largeurs de la carte, temps du record,
      identite = mot MPP -> famille d'arme, croisee avec la famille high-32 des images-cles du
      meme slot) ; une creation sans identite croisee est ECARTEE et comptee. Meme chose pour les
      power-ups `ti=37` (`ScanFilmEquipmentCreations`, familles `powerup_*`). Publier par film :
      creations acceptees / croisees / ecartees, et la part des creations SANS vie delta (candidats
      « apparus au repos » = socles attendus).
      **FAIT** : 8 films / 6 cartes, `replay/ground_weapon_pads_research_test.go` (garde `GW_PADS`).
      2 399 creations acceptees, **1 785 croisees**, 614 ecartees ; temoin fantome 1 291 acceptees
      pour **13 croisees** (le filtre d'identite rend le fantome discriminant, ce que l'acceptation
      seule ne faisait pas). 235 retenues sur 1 785 (13,2 %) sont SANS vie delta. Tables au journal.
- [x] 1.1 Classer chaque apparition `dropped` (mort d'un joueur a <= 2 frames et < 1,5 m —
      regle du 18/08 en miroir) ou `spawned` (aucune mort a < 1,5 m dans les 2 frames) —
      distribution publiee par film ; temoin : la part `dropped` doit etre elevee sur un Super
      Fiesta (`000d5950`) et plus faible sur une arene classique.
      **FAIT, TEMOIN TENU** : 1 275 `dropped` / 515 `spawned` sur 1 790 ; part `dropped` maximale
      sur `000d5950` (Super Fiesta, **82,3 %**) et minimale sur les arenes classiques (`bcb6d393`
      CTF 62,3 %, `00162144` 64,9 %). **Decouverte 5** : `dropped` implique une vie delta
      **1 275 / 1 275**, donc « sans vie delta » est un SOUS-ENSEMBLE STRICT de `spawned`.
- [x] 1.2 Grapper les apparitions `spawned` par famille et position (rayon 1 m) ; compter les
      grappes par carte ; temoin de Notion 11 (petit et stable, 6-12 sur une arene) ; temoin
      negatif : les positions de mort ne forment pas de grappes de meme famille.
      **FAIT, SUR LES DEUX JEUX DE CANDIDATS** (le critere ecrit `spawned`, et celui que l'item 1.0
      designe — `at_rest` = `spawned` sans vie delta). Le temoin de Notion 11 tient sur `at_rest`
      (**6 a 10 socles sur 4 cartes**) et PAS sur `spawned` (1 a 21). **Temoin negatif TENU** : les
      morts rendent 60 a 204 grappes dont 82 a 97 % a une seule apparition.
- [x] 1.3 Cycle : ecart entre apparitions successives par grappe ; mediane / p10 / p90 ;
      « cycle non etabli » si instable. Comparer aux cycles connus du jeu quand ils existent
      (armes de puissance ~2-3 min en Arena — source Waypoint si l'utilisateur en a une).
      **FAIT** : cycles publies socle par socle (lignes `PAD`). **4 socles sur 57 ont un cycle
      ETABLI** (jeu `at_rest`) ; les autres publient « non etabli », jamais un chiffre. **Aucune
      comparaison a une source officielle** : le lot n'en avait aucune hors ligne, et le brief
      interdit de comparer sans source.
- [x] 1.4 Publier au registre les socles par carte (position, famille, cycle) comme
      donnee de REFERENCE derivee des films. **Critere tranche** : si, sur une carte vue dans
      >= 2 films, >= 80 % des grappes d'un film retrouvent une grappe de MEME famille a < 1 m
      dans l'autre, les socles sont une propriete de la carte -> catalogue versionne
      `map_weapon_pads.json` (au meme endroit et au meme format d'ecriture que
      `map_objectives.json`), alimente par les films ; sinon ils restent PAR MATCH (publies
      dans le document de rejeu seulement) et le registre le dit.
      **FAIT — LE CRITERE N'EST PAS ATTEINT, ET AUCUN CATALOGUE N'EST CREE.** 0 paire sur 3 le
      tient (meilleure : Catalyst, 70,0 % dans les deux sens). Le registre le dit, avec la raison
      MESUREE : la POSITION du socle est une propriete de la carte (Catalyst 10/10 dans les deux
      sens, au centimetre), la FAMILLE ne l'est pas.

**Gate 1** : grappes petites et stables sur >= 3 cartes, temoin negatif tenu, cycles publies.

> **GATE 1 — NON ATTEINT sur la clause de recouvrement ; les trois autres clauses sont TENUES.**
> Seuil NON rebaisse, et le detail est au journal :
>
> - grappes petites et stables (6-12) sur >= 3 cartes : **TENU** sur le jeu `at_rest` (Cliffhanger
>   10, Catalyst 10 et 10, Streets 6 et 7, Smallhalla 10 — 4 cartes) ; NON tenu sur le jeu
>   `spawned` litteral (1 a 21, une seule carte dans la bande) ;
> - **>= 80 % de recouvrement entre deux films de la meme carte : NON TENU — 0 paire sur 3**,
>   meilleure mesure 7/10 = 70,0 % dans les deux sens sur Catalyst ;
> - temoin negatif (les morts ne grappent pas par famille) : **TENU**, et largement ;
> - cycles publies, etablis ou « non etabli » : **TENU**.
>
> **CE QUE LA MESURE DIT DU DESACCORD, et c'est un resultat POSITIF sur les socles eux-memes** :
> sur les deux films de Catalyst, les **10 socles sont aux 10 MEMES positions, au centimetre**
> (`-9,74 0,00 22,40`, `5,16 0,00 26,50`, `0,00 ±25,2x 26,50`...). Trois d'entre eux portent une
> arme DIFFERENTE d'un film a l'autre : Energy Sword <-> Gravity Hammer, et deux socles
> VK78 Commando <-> BR75. La meme signature sur Streets (Shock Rifle <-> Stalker Rifle,
> Cindershot <-> M41 SPNKr aux memes positions). **Le SOCLE appartient a la carte ; l'ARME qui y
> apparait appartient au MATCH.** Le critere du plan exigeait les deux ensemble : il refuse donc
> a bon droit le catalogue `map_weapon_pads.json` tel qu'il etait specifie (position + famille).
> Un catalogue de POSITIONS seules serait mesurable — ce n'est pas ce que le plan a tranche, et
> ce lot ne le decide pas.
>
> **Consequence contractuelle** : la phase 2 n'est PAS ouverte par ce lot (gate d'arret reel).
>
> **ARBITRAGE SUPERVISEUR (2026-08-17, CR verifie sur pieces : `668b2dc78`, `5161da69c`, items
> 1.0-1.4 `[x]`, gates `EXIT_*=0`)** : la clause « >= 80 % de recouvrement » est le CRITERE DE
> L'ITEM 1.4 (catalogue ou par match), pas une clause du gate de phase tel que ce plan l'ecrit
> (« grappes petites et stables sur >= 3 cartes, temoin negatif tenu, cycles publies » — les trois
> sont TENUES). Le critere 1.4 a rendu son verdict, et le plan avait tranche la suite : « sinon ils
> restent PAR MATCH et le registre le dit ». C'est le cas : **les socles sont publies PAR MATCH**
> (position, famille observee dans le match, presence dans le temps), **aucun catalogue** ; l'idee
> d'un catalogue de POSITIONS seules (le socle appartient a la carte, l'arme au match) va aux
> Decouvertes/registre comme option ulterieure, hors de ce plan. **La phase 2 est OUVERTE.** Deux
> precisions pour la phase 2, tirees de la mesure : (a) l'horloge d'un socle repart au RAMASSAGE —
> le cycle se re-mesure donc du ramassage a la reapparition suivante (item 2.4 ajoute, memes
> regles de stabilite que 1.3) ; (b) la regle `dropped` ne voit pas les ECHANGES d'arme (280/1 790
> `spawned` avec vie delta, presque tous MA40/Sidekick) — la phase 2 les traite comme des
> apparitions ordinaires, sans les nommer autrement (decouverte 7, vocabulaire a trancher en
> phase 3).

### Phase 2 — RAMASSAGE par disparition + proximite

- [x] 2.1 Pour chaque apparition (creation `ti=42` identifiee, phase 1.0) : la disparition se
      BORNE par le recensement des images-cles du slot (derniere image-cle ou l'objet est
      recense, premiere ou il ne l'est plus — acquis du correctif de revue : `t1` = mise au
      repos, DEL non isolable, pas de fin explicite) et se DATE dans cet intervalle par le
      passage d'un joueur a < 1,5 m de la position de l'objet -> ramasseur = ce joueur (si
      plusieurs : le premier ; si aucun : `unknown`, date = borne haute). Distribution des
      distances (mediane, p90), largeur des intervalles, et temoin (distance au joueur le plus
      proche a un instant tire au hasard PENDANT la presence de l'objet).
      **FAIT** : 1 785 apparitions, 1 111 datees / 554 `unknown` / 120 « jamais ramassees ».
      Intervalles : mediane 20,00 s (= l'espacement des images-cles). **La distribution des
      distances est vide de sens PAR CONSTRUCTION** (decouverte 9) ; les temoins tranchent :
      le critere discrimine d'un facteur 4 sur les socles et de 1,4 seulement sur l'ensemble.
- [!] 2.2 CONTROLE INDEPENDANT : le loadout d'images-cles du ramasseur doit porter la famille
      ramassee a l'image-cle suivante (`keyframe_loadout.go`, 98,3 % de temoin croise) — c'est
      l'oracle du ramassage. Publier le taux d'accord et un temoin (joueur au hasard).
      **MESURE FAITE ET PUBLIEE ; LE GATE 2 QU'ELLE DEVAIT PASSER EST NON ATTEINT** — 393 / 1 111
      = **35,4 %** contre >= 90 % exige, seuil NON rebaisse. Le detail est au journal : sur les
      SOCLES l'accord ecrase son temoin (62,7 % contre 3,5 %), sur les armes `dropped` il passe
      SOUS son temoin (32,1 % contre 65,0 %) — et ces dernieres font 84,1 % du denominateur.
- [x] 2.3 Publier (au journal du plan et au registre — le document de rejeu est la phase 3) :
      `weaponPickups` {t, x, y, family, slot ramasseur | -1} et l'etat des socles dans le temps
      (present / vide, par socle de la phase 1).
      **FAIT** : lignes `PICKUP` et `PADSTATE` (format au journal), publiees comme PROPOSITION
      pour la phase 3. Rien n'est ecrit au document de rejeu.
- [x] 2.4 Cycle RE-MESURE depuis le ramassage : ecart entre le ramassage sur un socle et la
      reapparition suivante sur ce socle (mediane, p10, p90 par socle et par famille) ; « non
      etabli » si ecart-type > 20 % de la mediane ; comparer au 1.3 (horloge d'apparition) et
      publier combien de socles passent de « non etabli » a « etabli ».
      **FAIT, ET C'EST LE RESULTAT DU LOT** : **24 socles sur 57 ont un cycle etabli contre 4
      sur 57 a l'item 1.3 — 20 socles gagnes**, aux memes regles de stabilite. Mode mesure a
      **30,5 s** (55 ecarts sur 142 dans 30-35 s, tenant dans 0,34 s), armes de puissance a
      100-195 s. Aucune comparaison a une source officielle (aucune hors ligne).

**Gate 2** : accord avec l'oracle des loadouts >= 90 % sur les ramassages a ramasseur
identifie ; sinon `[!]` avec la mesure.

> **GATE 2 — NON ATTEINT. 35,4 % contre >= 90 % exige (393 / 1 111), seuil NON rebaisse.** Le
> denominateur est celui que le plan ecrit : ramassages a ramasseur identifie ayant une image-cle
> suivante avant la fin du film. Le gate tombe aussi sur le sous-ensemble le plus favorable
> (socles : 62,7 %), donc aucun decoupage ne le sauve, et aucun n'a ete essaye pour le sauver.
>
> **CE QUE LA MESURE ETABLIT MALGRE LE GATE, et il faut les deux enonces** :
>
> - **le RAMASSAGE SUR UN SOCLE est etabli par deux mesures independantes** : l'accord de
>   l'oracle ecrase son temoin d'un facteur 17,8 (62,7 % contre 3,5 %), et le critere de
>   proximite d'un facteur 4 (89,6 % contre 22,4 %). Les armes concernees ne sont pas portees
>   avant (2,3 %) : ce sont des ramassages NOUVEAUX ;
> - **la « disparition » d'une arme `dropped` n'est PAS un ramassage** — c'est une despawn. Son
>   accord passe SOUS son propre temoin (32,1 % contre 65,0 %), ce qui est la signature d'un
>   critere qui ne mesure rien sur cette population. Elle pese 84,1 % du denominateur du gate :
>   le gate a ete ecrit sur une population que la mesure vient de scinder ;
> - **l'oracle a un plafond structurel de 55,3 %** (le slot du bipede migre au respawn, et 20 s
>   suffisent a mourir) : meme un ramasseur toujours juste ne pourrait pas atteindre 90 % sur ce
>   denominateur. Le seuil de 90 % etait donc inatteignable par construction — constat ECRIT
>   APRES coup, qui n'excuse rien et ne rebaisse rien, mais que la phase 3 doit connaitre.
>
> **Consequence contractuelle** : la phase 3 n'est PAS ouverte par ce lot. Ce qu'un arbitrage
> aurait a trancher : publier les ramassages des SEULS socles (`at_rest`), ou refuser la
> publication tant qu'un oracle sans plafond n'existe pas. Ce lot ne le decide pas.
>
> **ARBITRAGE SUPERVISEUR (2026-08-17, CR verifie sur pieces : `bfa67b4de` instrument et seuils
> AVANT mesure, `7d3f58940` mesure ; items 2.1/2.3/2.4 `[x]`, 2.2 `[!]` ; gates `EXIT_*=0`)** :
> le seuil de 90 % reste ; l'ORACLE, lui, etait mal construit : il suivait le SLOT DE VIE du
> ramasseur, qui migre au respawn — le plafond de 55,3 % est un artefact de l'oracle, pas du
> phenomene (decouverte 10). Le pont slot -> joueur existe dans le constructeur du document
> (`replay/owners.go` `SlotXUID`, `lives.go`, `closures.go closeBridge`, `Track.XUID`) : l'oracle
> doit suivre le JOUEUR. D'ou l'item 2.5 ci-dessous, avec le MEME seuil, sur la SEULE population
> que la mesure a qualifiee (les socles `at_rest` — la « disparition » d'une arme `dropped` est
> une despawn, decouverte 11, hors publication du ramassage). Perimetre de la phase 3 TRANCHE des
> maintenant, dans les deux issues de 2.5 : `weaponPads` (socles par match : position, famille,
> apparitions, intervalles de presence bornes par les images-cles, cycle depuis le ramassage s'il
> est etabli sinon `null`) est publie ; `padPickups` (t = intervalle [borne basse, borne haute],
> socle) est publie ; le RAMASSEUR (xuid) n'est publie QUE si 2.5 tient son gate, sinon `null`
> partout et le registre porte la condition de reprise. Aucun ramassage d'arme `dropped` n'est
> publie. L'utilisateur peut renverser.

- [!] 2.5 ORACLE PAR JOUEUR (meme seuil, meme temoin) — sur les ramassages de SOCLES a ramasseur
      identifie : ramasseur = slot de vie -> xuid par le pont du constructeur (`SlotXUID`) ; a la
      premiere image-cle qui suit, prendre le loadout de la VIE COURANTE de ce xuid (son slot a cet
      instant, quel qu'il soit) ; accord = la famille ramassee y figure. Denominateur = ramassages
      de socle dont le xuid a un loadout observable a cette image-cle (les morts sans loadout sont
      COMPTES a part, pas dans le denominateur, et publies). Temoin : autre joueur au hasard a la
      meme image-cle. Publier aussi la part des ramassages sans pont (slot non attribue).
      **Gate 2.5** : accord >= 90 % => le ramasseur (xuid) est publiable en phase 3 ; sinon `[!]`
      et `null`.
      **MESURE FAITE ET PUBLIEE ; LE GATE 2.5 EST NON ATTEINT** — 102 / 128 = **79,7 %** contre
      >= 90 % exige, seuil NON rebaisse. Le ramasseur (xuid) n'est donc PAS publiable : `null`
      partout en phase 3. Le detail est au journal, et il REFUTE l'hypothese de l'arbitrage : le
      pont fonctionne (11 ramassages sans pont sur 177), mais le plafond ne se leve pas
      (71,2 % -> 72,3 %) et l'accord, rapporte aux cas ou l'oracle peut parler, BAISSE
      (88,1 % par slot -> 79,7 % par joueur). **Decouverte 12.**

**Gate 2 (relu apres 2.5)** : la phase 3 s'ouvre dans les deux issues de 2.5, avec le perimetre
tranche ci-dessus.

> **GATE 2.5 — NON ATTEINT. 79,7 % contre >= 90 % exige (102 / 128), seuil NON rebaisse.** Le
> denominateur est celui que l'item ecrit : ramassages de SOCLE (`at_rest`) dates, a pont, ayant
> une image-cle suivante, et dont le xuid a un loadout OBSERVABLE a cette image-cle. Les trois
> ecartements sont publies a part : **11 / 177 sans pont** (6,2 %), **0 sans image-cle suivante**,
> **38 / 177 sans loadout observable** (21,5 %).
>
> **CE QUE LA MESURE ETABLIT, ET C'EST UNE REFUTATION — il faut les trois enonces** :
>
> - **le pont du constructeur n'est pas en cause.** Il nomme 94 % des ramasseurs de socle
>   (166 / 177), avec 0 collision de slot et 0 desaccord d'index sur les 8 films, et 90 a 97 % des
>   vies nommees par le seul fil des morts. Quand il manque, c'est le film qui est pauvre en morts
>   (`b8d1fe0c`, le plus court : 31 morts pour 44 vies, 4 des 6 ramassages sans pont) ;
> - **le PLAFOND NE SE LEVE PAS, et c'etait toute l'hypothese de l'arbitrage.** Par slot,
>   126 / 177 = 71,2 % des ramassages avaient un loadout lisible ; par joueur, 128 / 177 = 72,3 %.
>   Un point de gagne. Suivre le joueur a travers les respawns ne recupere qu'entre 2 et 13 des
>   51 cas muets, et 8 de ces recuperations tiennent a un seul film (`00162144`, 18 muets par slot
>   -> 10 par joueur) ;
> - **rapporte aux cas ou l'oracle PEUT parler — la meme restriction des deux cotes — l'oracle par
>   joueur est MOINS bon que l'oracle par slot : 102 / 128 = 79,7 % contre 111 / 126 = 88,1 %.**
>   C'est le fait central, et il a une explication que la mesure designe : un joueur qui MEURT
>   entre le ramassage et l'image-cle suivante LACHE ce qu'il a ramasse. Son slot d'origine
>   disparait (cas muet par slot) mais sa nouvelle vie, elle, est bien la — avec le loadout de
>   REAPPARITION. Suivre le joueur ne transforme donc pas un silence en accord : il transforme un
>   silence en DESACCORD. **Decouverte 12.**
>
> **AUCUN DECOUPAGE NE SAUVE LE GATE, et aucun n'a ete essaye pour le sauver.** Deux bornes
> suffisent a le dire : si les 5 cas ou un xuid porte un loadout sur PLUSIEURS slots a la meme
> image-cle etaient tous lus a l'envers, les corriger donnerait au mieux 107 / 128 = 83,6 % ; et si
> les 11 ramassages sans pont etaient tous ponts et tous d'accord, 113 / 139 = 81,3 %.
>
> **Consequence contractuelle** : item 2.5 `[!]`. Le RAMASSEUR (xuid) n'est pas publiable — `null`
> partout en phase 3, comme l'arbitrage l'avait tranche pour cette issue. Le reste du perimetre de
> la phase 3 (`weaponPads`, `padPickups` sans ramasseur) est inchange et reste ouvert. Ce que la
> mesure laisse comme condition de reprise n'est PAS un meilleur pont : c'est un oracle plus
> RAPPROCHE que 20 s — l'inventaire lu dans le flux delta, ou toute source qui observe le porteur
> avant qu'il puisse mourir.

### Phase 3 — PUBLICATION (schema 11) et note UI

- [ ] 3.1 Document : `weaponPads` (socles PAR MATCH : position i0, famille (mot MPP -> label
      d'arme via le catalogue existant), apparitions [t], intervalles de presence [t_apparition,
      borne basse, borne haute] bornes par les images-cles, cycle depuis le ramassage {medianeS,
      p10, p90} si etabli sinon `null`), `padPickups` [{pad, tLow, tHigh, xuid|null}] (xuid selon
      2.5) ; couverture (nombre de creations retenues / ecartees, socles, ramassages dates /
      unknown / never) dans le champ de couverture existant ; `SchemaVersion` chronique (10 -> 11
      avec la ligne d'historique) ; contrat (`wantReplayDocumentFields`, chronique), OpenAPI
      regeneree, `generated.ts`, `NULLABLE_ARRAYS`, goldens et fixture v5 re-cuits, temoins
      re-cuits (`000d5950`, `01e1f945`, `00162144`) ; aucun ramassage d'arme `dropped`, aucun
      catalogue de carte.
- [ ] 3.2 Note UI (decision 6, corrigee par la mesure) pour l'utilisateur : icone de l'arme sur le
      socle, etat present / vide (vide des `tLow`, incertain jusqu'a `tHigh`), compte a rebours
      SEULEMENT si le cycle est etabli, ramasseur sur la fiche SEULEMENT si publie ; aucun rendu ici.

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
5. **L'ARME D'UN SOCLE N'EST PAS UNE PROPRIETE DE LA CARTE — la position, si.** Mesure phase 1.4 :
   sur deux films de Catalyst (meme carte, MEME mode KOTH), les 10 socles sont aux 10 memes
   coordonnees au centimetre, mais trois portent une arme differente (Energy Sword <-> Gravity
   Hammer ; deux socles VK78 Commando <-> BR75). Idem Streets (Shock Rifle <-> Stalker Rifle,
   Cindershot <-> M41 SPNKr). Consequence pour QUI VOUDRA UN CATALOGUE : il devra etre keye sur la
   POSITION, la famille restant une donnee de match. Non traite ici — le plan a tranche un critere
   position+famille, et ce lot ne le renverse pas.
6. **Le corpus local de films ne contient AUCUN match classe** : 951 films joints a l'instantane
   parquet du registre, `is_ranked` faux partout. Toute mesure future qui voudrait « un mode Arena
   ranked » doit d'abord constater cette absence au lieu de chercher un film qui n'existe pas.
7. **La regle `dropped` du 18/08 ne voit pas les ECHANGES d'arme.** 280 apparitions sur 1 790 sont
   `spawned` (aucune mort a proximite) tout en ayant une vie delta ; la lecture ligne a ligne les
   montre presque toutes en `MA40 AR` / `Mk51 Sidekick`, aux quatre coins de la carte — l'arme de
   depart qu'un joueur lache en ramassant autre chose. Ce n'est pas un defaut de la regle (elle
   mesure une mort) mais c'est un troisieme mode que le vocabulaire `deployed`/`dropped`/`unknown`
   ne nomme pas. Non traite : le nommer changerait un vocabulaire publie au contrat.
8. **`ScanFilmGroundWeaponCreationsForBand` ne calibre pas la largeur du bloc MPP** — elle utilise
   `CurrentMPPWidths()` telle qu'elle est. Sans consequence ici (les 8 films tranchent tous a
   `9/5`, la valeur par defaut, et l'accord d'identite est de 100 %), mais un film BTB calibre a
   `8/3` lirait l'identite `ti=42` a la mauvaise largeur sans que rien ne le signale. Non traite :
   aucun film BTB dans le jeu mesure.
9. **La distance publiee du ramasseur est un ARTEFACT DE LA REGLE, pas une mesure**, et quelqu'un
   la lira comme une mesure si on ne l'ecrit pas. La regle retient « le PREMIER passage sous
   1,5 m » : la distance rendue est donc celle a laquelle le seuil est franchi, et elle vaut
   mediane 1,46-1,49 m / p90 1,48-1,50 m / max 1,50 m sur les HUIT films et sur TOUS les
   sous-ensembles — indistinctement. Une mesure de distance qui aurait un sens serait le MINIMUM
   sur l'intervalle, ou la distance a l'instant de la derniere image-cle recensante. Non traite :
   le plan demande la distribution des distances du ramasseur, elle est publiee telle quelle avec
   cet avertissement.
10. **L'oracle des loadouts a un PLAFOND STRUCTUREL de 55,3 %, et sa cause est le slot de bipede
    qui MIGRE au respawn** (`offline_biped.go` : « un slot correspond a UNE VIE, pas a un
    joueur »). 497 ramasseurs sur 1 111 n'ont aucun loadout a l'image-cle suivante — sur les
    socles, 51 sur 177. Vingt secondes d'ecart entre images-cles suffisent a mourir. **Ce qui
    leverait le plafond est l'attribution slot -> JOUEUR a travers les respawns**, qui n'existe
    pas dans ce decodeur et qui debloquerait aussi la publication du ramasseur au document
    (aujourd'hui `PICKUP.slot` designe une vie, pas un joueur). Non traite : hors du perimetre de
    la phase 2, et c'est un chantier a soi seul.
11. **La regle `dropped` designe une DESPAWN, pas un ramassage — et le vocabulaire publie ne le
    dit pas.** Prolonge la decouverte 7. L'accord de l'oracle sur les armes `dropped` (32,1 %)
    passe SOUS son propre temoin (65,0 %) : le critere de proximite n'y mesure rien, parce
    qu'une arme lachee a une mort disparait le plus souvent toute seule. Consequence pour QUI
    VOUDRA PUBLIER LES RAMASSAGES : le seul sous-ensemble ou la mesure tient est `at_rest` (les
    socles). Non traite : la phase 3 est fermee par le gate 2.
12. **LE PLAFOND DE L'ORACLE N'EST PAS LE PONT — C'EST L'ESPACEMENT DES IMAGES-CLES, et la
    decouverte 10 se trompait de cause.** L'item 2.5 a rejoue l'oracle par JOUEUR, avec le pont du
    constructeur, sur la meme population de socles : le plafond passe de 71,2 % a **72,3 %** (un
    point), et l'accord rapporte aux cas ou l'oracle peut parler BAISSE de **88,1 % a 79,7 %**. La
    raison est physique : un joueur qui MEURT dans les 20 s qui suivent le ramassage lache ce
    qu'il a pris. Par slot, ce cas etait muet ; par joueur, il devient un DESACCORD, parce que la
    nouvelle vie porte le loadout de reapparition. **Ce qui leverait le plafond n'est donc pas un
    meilleur pont mais un oracle plus RAPPROCHE** : l'inventaire lu dans le flux delta, ou toute
    source qui observe le porteur avant qu'il puisse mourir. Non traite : c'est un chantier a soi
    seul, et la phase 3 est tranchee sans ramasseur.
13. **UN MEME XUID PORTE PARFOIS UN LOADOUT SUR PLUSIEURS SLOTS A LA MEME IMAGE-CLE** — 5 cas sur
    128, dont 4 sur le seul `01e1f945`. Un slot est une vie et un joueur n'en a qu'une a la fois :
    soit deux vies du meme joueur sont repliquees en meme temps a la frontiere d'un respawn, soit
    le pont attribue au meme joueur un slot qui ne lui appartient pas (les fermetures y sont plus
    exposees que la lecture). La regle de l'item 2.5, ecrite avant la mesure, retient le plus petit
    slot et COMPTE le cas. Non traite : 5 cas ne changent pas le verdict du gate (83,6 % au mieux),
    et l'elucider demanderait de re-mesurer le pont lui-meme, hors perimetre de cet item.

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
- 2026-08-17 — **arbitrage superviseur** : phase 1 amendee (entree = records de CREATION `ti=42`
  identifies par croisement, item 1.0 ajoute ; phase 2.1 reecrite sur le bornage par images-cles).
  **Phase 1 LANCEE** (agent Opus, principal). L'utilisateur peut renverser l'arbitrage.
- 2026-08-17 — **phase 1 CLOSE. Gate 1 non atteint sur la clause de recouvrement ; les socles
  eux-memes sont etablis.** Tout ci-dessous a tourne dans la session, aux largeurs de chaque
  carte, `CGO_ENABLED=0`, UN FILM PAR PROCESSUS.

### Instruments et seuils (ecrits AVANT la mesure)

`replay/ground_weapon_pads_research_test.go` (garde `GW_PADS` / `GW_PADS_MAP`) : un film, tout
l'enchainement 1.0 -> 1.3. `replay/ground_weapon_pads_cluster_test.go` : la regle de grappe et la
regle de cycle, isolees et TESTEES sans garde (3 tests dans le gate ordinaire).
`replay/ground_weapon_pads_aggregate_test.go` (garde `GW_PADS_AGG`) : le critere 1.4, sur les
sorties des runs par film — il ne decode rien.

| seuil | valeur | source |
|---|---|---|
| identite retenue | mot MPP 32 bits dans `weaponv3.KnownWeaponHigh32` | arbitrage superviseur, decouverte 2 |
| `dropped` | fin d'une vie de bipede a <= 2 frames ET < 1,5 m | `originDropWindowUS` / `originDropMaxDist`, production du 18/08 |
| rayon de grappe | 1,0 m, meme famille | item 1.2 |
| socle | grappe a >= 2 apparitions | decision 2 (« recurrence ») |
| cycle ETABLI | >= 2 ecarts ET ecart-type <= 20 % de la mediane | decision 3 |
| recouvrement carte | >= 80 %, meme famille, < 1 m, DANS LES DEUX SENS | item 1.4 |

**LA SECONDE FORMULATION DE L'ARBITRAGE EST UN SOUS-ENSEMBLE DE LA PREMIERE, et il faut le dire** :
« egal a la famille high-32 du meme slot a l'image-cle » n'ajoute rien a « mot MPP resolu en famille
d'arme connue », parce que `ScanFilmKeyframeGroundWeapons` ne rend QUE des familles deja au
catalogue. Le croisement d'image-cle n'est donc pas le filtre — il en est le CONTROLE, publie a part.

### Films — les 6 de la phase 0, plus 2 secondes vues lues au parquet du registre

Cartes lues sans ouvrir aucune base : instantane parquet
`data/backups/staging/halo_infinite/shared_matches_v2/match_registry_20260711_090652.parquet`.
**Aucun film classe (`is_ranked`) dans le cache** (951 films, 0 ranked) : les secondes vues sont
donc des modes Arena de Quick Play, choisies NATIVES (pas de variante Forge).

| film | carte | mode | largeurs | duree | role |
|---|---|---|---|---|---|
| `000d5950` | Cliffhanger - Forge | Super Fiesta Slayer | `[13 13 14]` | 496 s | phase 0 (temoin Fiesta) |
| `bcb6d393` | Cliffhanger | Arena:CTF | `[13 13 14]` | 353 s | phase 0 |
| `01e1f945` | Catalyst | Arena:KOTH | `[15 15 15]` | 540 s | phase 0 |
| `75f1188f` | Catalyst | Arena:KOTH | `[15 15 15]` | 427 s | **AJOUT — 2e vue, MEME mode** |
| `7f1bbf06` | Streets | Arena:KOTH | `[12 12 12]` | 318 s | phase 0 |
| `b974a390` | Streets | Arena:Strongholds | `[12 12 12]` | 684 s | **AJOUT — 2e vue** |
| `b8d1fe0c` | Recharge | Arena:Neutral Flag CTF | `[18 18 15]` | 226 s | phase 0 |
| `00162144` | Smallhalla | Community:Slayer | `[15 15 17]` | 446 s | phase 0 |

### 1.0 — les apparitions, et ce que le filtre d'identite change au temoin fantome

| film | ancres | acceptees | CROISEES | ecartees | fantome acc. | fantome crois. | sans vie delta |
|---|---:|---:|---:|---:|---:|---:|---:|
| `000d5950` | 6 487 | 317 | 220 | 97 | 164 | 0 | 0 (0,0 %) |
| `bcb6d393` | 2 377 | 219 | 159 | 60 | 125 | 0 | 34 (21,4 %) |
| `01e1f945` | 7 953 | 405 | 291 | 114 | 147 | 0 | 38 (13,1 %) |
| `75f1188f` | 4 073 | 265 | 213 | 52 | 98 | 0 | 35 (16,4 %) |
| `7f1bbf06` | 3 446 | 210 | 166 | 44 | 57 | 0 | 25 (15,1 %) |
| `b974a390` | 13 928 | 499 | 372 | 127 | 267 | 13 | 42 (11,3 %) |
| `b8d1fe0c` | 691 | 118 | 86 | 32 | 35 | 0 | 12 (14,0 %) |
| `00162144` | 6 485 | 366 | 278 | 88 | 398 | 0 | 49 (17,6 %) |
| **TOTAL** | **45 440** | **2 399** | **1 785** | **614** | **1 291** | **13** | **235 (13,2 %)** |

**LE GARDE-FOU DE LA DECOUVERTE 2 FONCTIONNE, ET IL SE MESURE.** Sur `00162144`, la bande fantome
rend 398 creations acceptees contre 366 pour la bande reelle — l'acceptation ne separe rien. Apres
le filtre d'identite : **0 fantome croisee contre 278 reelles**. Sur les 8 films, 13 fantomes
croisees contre 1 785 (un facteur 137). Le fantome n'est plus un contre-exemple, il est un temoin.

**CONTROLE d'identite par la chaine independante des images-cles** (familles high-32 lues au
keyframe pour la meme vie) : **1 360 paires, 1 360 accords, 100 %** (169/169, 118/118, 213/213,
160/160, 140/140, 263/263, 70/70, 227/227). A ne PAS comparer au 98,9 % de la phase 0 : celui-ci
portait sur TOUTES les creations acceptees, celui-la sur les seules retenues — le filtre est en
amont, donc il ecarte precisement les cas qui desaccordaient.

### 1.1 — `dropped` / `spawned`, et le croisement avec la vie delta

| film | retenues | dropped | spawned | dont SANS vie delta | dont AVEC vie delta |
|---|---:|---:|---:|---:|---:|
| `000d5950` (Fiesta) | 220 | 181 (**82,3 %**) | 39 (17,7 %) | 0 | 39 |
| `bcb6d393` (CTF) | 159 | 99 (**62,3 %**) | 60 (37,7 %) | 34 | 26 |
| `01e1f945` | 292 | 204 (69,9 %) | 88 (30,1 %) | 38 | 50 |
| `75f1188f` | 214 | 153 (71,5 %) | 61 (28,5 %) | 35 | 26 |
| `7f1bbf06` | 166 | 116 (69,9 %) | 50 (30,1 %) | 25 | 25 |
| `b974a390` | 372 | 274 (73,7 %) | 98 (26,3 %) | 42 | 56 |
| `b8d1fe0c` | 88 | 67 (76,1 %) | 21 (23,9 %) | 12 | 9 |
| `00162144` | 279 | 181 (64,9 %) | 98 (35,1 %) | 49 | 49 |
| **TOTAL** | **1 790** | **1 275 (71,2 %)** | **515 (28,8 %)** | **235** | **280** |

**TEMOIN DU PLAN TENU** : la part `dropped` est maximale sur le Super Fiesta (82,3 %) et minimale
sur les arenes classiques (62,3 % et 64,9 %).

**DEUX CRITERES INDEPENDANTS, EMBOITES SANS UNE SEULE EXCEPTION** : `dropped` implique une vie
delta **1 275 fois sur 1 275**. Aucune apparition n'est a la fois lachee a une mort et immobile
des sa naissance. Donc « sans vie delta » (`at_rest`) est un sous-ensemble strict de `spawned`, et
les 280 `spawned` AVEC vie delta sont autre chose : la lecture ligne a ligne les montre presque
toutes en `MA40 AR` et `Mk51 Sidekick` — **l'arme de depart qu'un joueur LACHE en ramassant autre
chose**, qui n'a pas de mort a proximite mais qui TOMBE. La regle du 18/08 ne les voit pas, et
c'est normal : elle mesure une mort, pas un echange.

### 1.2 — grappes, et le temoin negatif

| film | carte | `spawned` grappes / socles | `at_rest` grappes / socles | MORTS grappes / >= 2 / singletons |
|---|---|---:|---:|---:|
| `000d5950` | Cliffhanger-Forge | 38 / 1 | 0 / 0 | 171 / 9 / 162 |
| `bcb6d393` | Cliffhanger | 38 / 14 | 16 / **10** | 96 / 3 / 93 |
| `01e1f945` | Catalyst | 44 / 21 | 10 / **10** | 168 / 24 / 144 |
| `75f1188f` | Catalyst | 30 / 14 | 10 / **10** | 124 / 20 / 104 |
| `7f1bbf06` | Streets | 27 / 10 | 9 / **6** | 101 / 13 / 88 |
| `b974a390` | Streets | 39 / 17 | 8 / **7** | 204 / 42 / 162 |
| `b8d1fe0c` | Recharge | 16 / 5 | 8 / 4 | 60 / 6 / 54 |
| `00162144` | Smallhalla | 42 / 18 | 14 / **10** | 155 / 23 / 132 |

**LE TEMOIN DE NOTION 11 SE LIT SUR `at_rest`, PAS SUR `spawned`** : 6 a 10 socles sur quatre
cartes (Cliffhanger, Catalyst deux fois, Streets deux fois, Smallhalla), contre 1 a 21 pour le
critere litteral. Deux valeurs hors bande, toutes deux expliquees par un denominateur et non par
un ajustement : `b8d1fe0c` (4 socles) est le film le plus COURT du jeu (226 s), et `000d5950`
(0 socle) est un Super Fiesta sur variante FORGE — aucun rack de carte, ce qui est coherent avec
ses 82,3 % de lachers.

**TEMOIN NEGATIF TENU, ET LARGEMENT** : les positions de mort rendent 60 a 204 grappes, dont
**82 a 97 % a une seule apparition** (mediane de taille 1, max 3 a 8). Les socles `at_rest` rendent
8 a 16 grappes avec 0 a 4 singletons (mediane de taille 2 a 4, max 7 a 9). Les deux populations ne
se ressemblent pas : le critere separe.

### 1.3 — cycles

| film | socles `at_rest` | cycle ETABLI | mediane des medianes |
|---|---:|---:|---:|
| `bcb6d393` | 10 | 0 | — |
| `01e1f945` | 10 | 1 (10,0 %) | 235,4 s |
| `75f1188f` | 10 | 1 (10,0 %) | 160,9 s |
| `7f1bbf06` | 6 | 1 (16,7 %) | 69,5 s |
| `b974a390` | 7 | 1 (14,3 %) | 177,9 s |
| `b8d1fe0c` | 4 | 0 | — |
| `00162144` | 10 | 0 | — |
| **TOTAL** | **57** | **4 (7,0 %)** | |

**LE CYCLE NE S'ETABLIT PAS, ET LA MESURE DIT POURQUOI.** Les ecarts d'un meme socle varient du
simple au triple (`7f1bbf06` Cindershot : mediane 30,1 s, ecart-type 36,4 ; `b974a390` Bandit Evo :
226,6 s / 95,3). C'est attendu une fois pose : l'horloge d'un socle repart au RAMASSAGE, pas a
l'apparition, donc l'ecart mesure = temps ou l'arme est restee au sol + delai de reapparition. Un
cycle ne se lira qu'apres la phase 2 (le ramassage), en mesurant l'ecart ramassage -> apparition.
Aucun chiffre instable n'est publie comme cycle : les 53 autres socles disent « non etabli ».
Aucune comparaison a une source officielle (aucune disponible hors ligne).

### 1.4 — le critere de catalogue, et ce que son echec designe

| carte | paire (jeu `at_rest`) | MEME famille A->B / B->A | POSITION SEULE A->B / B->A |
|---|---|---:|---:|
| Catalyst | `01e1f945` <-> `75f1188f` | 7/10 = 70,0 % / 7/10 = 70,0 % | **10/10 = 100 % / 10/10 = 100 %** |
| Streets | `7f1bbf06` <-> `b974a390` | 3/6 = 50,0 % / 3/7 = 42,9 % | 5/6 = 83,3 % / 5/7 = 71,4 % |
| Cliffhanger | `000d5950` <-> `bcb6d393` | — / 0/10 = 0,0 % | — / 0/10 = 0,0 % |

**0 paire sur 3 tient le critere** (idem sur le jeu `spawned` : 47,6/71,4 · 60,0/35,3 · 0/0).

Les DIX socles de Catalyst, aux memes coordonnees dans les deux films :

| position | `01e1f945` | `75f1188f` |
|---|---|---|
| `-9,74  0,00 22,40` | CQS48 Bulldog x3 | CQS48 Bulldog x2 |
| ` 9,47 12,41 24,00` | Disruptor x2 | Disruptor x3 |
| ` 9,48 -12,40 24,02` | Disruptor x2 | Disruptor x3 |
| `-11,05  0,00 25,34` | **Energy Sword** x4 | **Gravity Hammer** x3 |
| ` 5,16  0,00 26,50` | M41 SPNKr x8 | M41 SPNKr x7 |
| ` 0,00 25,30 26,50` | S7 Sniper x5 | S7 Sniper x3 |
| ` 0,00 -25,20 26,50` | S7 Sniper x5 | S7 Sniper x5 |
| `11,60  0,00 22,60` | Sentinel Beam x4 | Sentinel Beam x2 |
| ` 6,28  6,94 27,02` | **VK78 Commando** x2 | **BR75** x2 |
| ` 6,29 -6,95 27,02` | **VK78 Commando** x3 | **BR75** x5 |

Meme signature sur Streets aux cinq positions partagees : `-1,00 -9,00 2,51` porte Shock Rifle puis
**Stalker Rifle**, `0,21 1,44 0,40` porte Cindershot puis **M41 SPNKr**. Les trois socles orphelins
de Streets (Mangler ; Pulse Carbine ; Needler) sont des socles vus une seule fois dans l'autre film
— le film court fait 318 s contre 684 s.

**AUCUN CATALOGUE N'EST CREE** : le critere tranche du plan exige position ET famille, il n'est pas
atteint, et le rabaisser pour publier un fichier serait exactement ce que ce plan interdit. Les
socles restent PAR MATCH. Le registre porte le report avec sa condition de reprise.

### Power-ups de socle — NEGATIF DE CORPUS (etendu, pas repete)

`ScanFilmEquipmentPlacements` (la fonction de production, calibration MPP comprise ; les 8 films
tranchent tous a `9/5`). **5 apparitions de power-up sur 8 films** — `01e1f945` 1, `75f1188f` 1,
`b8d1fe0c` 2, `00162144` 1 — **toutes `powerup_overshield`, toutes `dropped`, toutes avec une vie
delta**. Zero `powerup_camo`. Zero grappe, zero socle. Les 5 cartes NATIVES d'arene du jeu de films
(Catalyst x2, Streets x2, Recharge, Smallhalla, Cliffhanger) n'en portent aucun au repos. Le corpus
des films locaux ne contient AUCUN film classe (0 sur 951) : la voie « Arena ranked » n'a pas pu
etre essayee faute de matiere, et c'est le negatif qu'il faut ecrire — pas un echec de mesure.
- 2026-08-17 — **arbitrage superviseur apres la phase 1** : le recouvrement etait le critere de
  l'item 1.4 (verdict : PAR MATCH, aucun catalogue), pas une clause du gate de phase ; les trois
  clauses du gate 1 sont tenues => **phase 2 OUVERTE** (item 2.4 « cycle depuis le ramassage »
  ajoute). Agent Opus lance sur le principal.
- 2026-08-17 — **phase 2 LANCEE.** Instruments et seuils ci-dessous, ECRITS ET COMMITES AVANT LA
  MESURE (le commit d'instrument precede le commit de mesure — l'ordre est verifiable dans
  l'historique).

### Phase 2 — instruments et seuils (ecrits AVANT la mesure)

`replay/ground_weapon_pickup_research_test.go` (garde `GW_PICKUP` / `GW_PICKUP_MAP`) : un film,
toute la chaine 2.1 -> 2.4, cinq balayages du film et pas un de plus.
`replay/ground_weapon_pickup_report_test.go` : les quatre publications, qui ne decodent rien.
`replay/ground_weapon_pickup_rule_test.go` : la regle de bornage, la regle de datation et le
temoin d'instant, ISOLES et TESTES sans garde (4 tests dans le gate ordinaire).
`replay/ground_weapon_pads_cluster_test.go` gagne `gwPadsClusterAssign` (l'assignation
apparition -> socle) et `gwPadsCycleFromGaps` (le verdict de stabilite sur des ecarts deja
calcules) : la phase 2 en a besoin, et une SECONDE regle de grappe ou de stabilite aurait
diverge de la premiere au premier correctif. Les deux nouvelles fonctions sont testees.

| seuil | valeur | source |
|---|---|---|
| identite retenue | mot MPP 32 bits dans `weaponv3.KnownWeaponHigh32` | phase 1, decouverte 2 |
| bornage de la disparition | recensement `ti=42` du walker durci d'images-cles (249/250), restreint a la vie `[creation, reprise de la cle (slot,gen))` | phase 2.1 amendee, correctif de revue des poses |
| « jamais ramasse » | encore recense a la DERNIERE image-cle du film | phase 2.1 amendee |
| datation | passage d'un bipede a < 1,5 m (`originDropMaxDist`) de la position de reference, LE PREMIER en temps ; aucun -> `unknown`, date = borne haute | decision 4 + 2.1 amendee |
| position de reference | dernier point de la piste delta si l'objet a bouge, position de creation sinon | brief phase 2 |
| temoin d'instant | joueur le plus proche a un instant tire au sort dans `[creation, date du ramassage)`, tolerance +-100 ms, graine 20260817 | ecrit ici |
| temoin de fenetre | fenetre de MEME largeur que l'intervalle, placee au hasard dans `[creation, derniere image-cle recensante]` : contient-elle aussi un passage < 1,5 m ? | ecrit ici |
| oracle des loadouts | famille (NOM canonique, alias replies) portee par le ramasseur a la PREMIERE image-cle suivant la date | item 2.2 |
| **GATE 2** | **accord >= 90 %**, denominateur = ramassages a ramasseur identifie ayant une image-cle suivante avant la fin du film | plan |
| cycle depuis le ramassage | >= 2 ecarts ET ecart-type <= 20 % de la mediane (memes regles que 1.3) | decision 3, item 2.4 |

**REGLE DE LECTURE DU TEMOIN DE FENETRE, ecrite avant de la connaitre** : si une fenetre
quelconque de meme largeur contient un passage < 1,5 m aussi souvent que le vrai intervalle,
alors le critere ne DATE pas le ramassage — il designe le premier passant, et cela s'ecrit comme
un negatif au lieu d'etre presente comme une datation.

- 2026-08-17 — **phase 2 CLOSE. GATE 2 NON ATTEINT (35,4 % contre >= 90 % exige), et la mesure
  separe deux populations que le gate confondait.** Tout ci-dessous a tourne dans la session, aux
  largeurs de chaque carte, `CGO_ENABLED=0`, UN FILM PAR PROCESSUS, memes 8 films que la phase 1.

### Controle d'ancrage : la phase 2 reproduit la phase 1 au chiffre pres

Avant toute mesure nouvelle, les quantites PARTAGEES avec la phase 1 se retrouvent a l'identique
par une chaine reecrite : **1 785 apparitions retenues** (identiques film par film : 220 / 159 /
291 / 213 / 166 / 372 / 86 / 278), **235 `at_rest`**, **57 socles**, **4 cycles d'APPARITION
etablis sur 57**. Rien n'a derive entre les deux lots. Les 5 power-ups de la phase 1 sont des
objets `ti=37` : le brief de la phase 2 porte sur `ti=42`, ils sont hors perimetre — et la
phase 1 avait deja etabli qu'ils sont tous `dropped` avec vie delta, donc sans socle.

### 2.1 — le bornage tient, la datation ne discrimine QUE sur les socles

| film | retenues | DATEES | unknown | jamais ramassees | jamais recensees a une image-cle |
|---|---:|---:|---:|---:|---:|
| `000d5950` | 220 | 128 (58,2 %) | 74 (33,6 %) | 18 (8,2 %) | 49 |
| `bcb6d393` | 159 | 85 (53,5 %) | 74 (46,5 %) | 0 (0,0 %) | 52 |
| `01e1f945` | 291 | 176 (60,5 %) | 94 (32,3 %) | 21 (7,2 %) | 76 |
| `75f1188f` | 213 | 112 (52,6 %) | 80 (37,6 %) | 21 (9,9 %) | 53 |
| `7f1bbf06` | 166 | 109 (65,7 %) | 34 (20,5 %) | 23 (13,9 %) | 26 |
| `b974a390` | 372 | 255 (68,5 %) | 106 (28,5 %) | 11 (3,0 %) | 107 |
| `b8d1fe0c` | 86 | 55 (64,0 %) | 20 (23,3 %) | 11 (12,8 %) | 14 |
| `00162144` | 278 | 191 (68,7 %) | 72 (25,9 %) | 15 (5,4 %) | 51 |
| **TOTAL** | **1 785** | **1 111 (62,2 %)** | **554 (31,0 %)** | **120 (6,7 %)** | **428 (24,0 %)** |

**LARGEUR DES INTERVALLES : mediane 20,00 s, p90 20,00 a 20,02 s, max 20,02 s** sur les huit
films — c'est l'espacement des images-cles, et rien d'autre. Le bornage ne peut pas etre plus
fin que le recensement ; il ne pretend pas l'etre. **428 apparitions sur 1 785 (24,0 %) ne sont
recensees a AUCUNE image-cle** : nees et disparues entre deux, elles sont bornees par leur
creation et l'image-cle suivante.

**LA DISTRIBUTION DES DISTANCES DU RAMASSEUR NE DIT RIEN, ET IL FAUT LE DIRE AVANT DE LA LIRE** :
mediane 1,46 a 1,49 m, p90 1,48 a 1,50 m, max 1,50 m — sur les huit films, sur tous les
sous-ensembles. C'est une consequence de la regle, pas une mesure : « le PREMIER passage sous
1,5 m » rend par construction la distance a laquelle le seuil est franchi. Seuls les TEMOINS
sont informatifs (decouverte 9).

**LES DEUX TEMOINS, ET CE QU'ILS SEPARENT** (denominateurs publies ; le temoin de fenetre est
compare a la MESURE sur le MEME denominateur) :

| jeu | temoin d'instant : joueur < 1,5 m | mediane du temoin d'instant | MESURE (le vrai intervalle date) | TEMOIN de fenetre |
|---|---:|---:|---:|---:|
| tous | 143 / 1 562 = **9,2 %** | 4,4 a 6,2 m | 221 / 292 = **75,7 %** | 158 / 292 = **54,1 %** |
| `at_rest` (socles) | 5 / 189 = **2,6 %** | 5,3 a 12,2 m | 120 / 134 = **89,6 %** | 30 / 134 = **22,4 %** |

**Sur les socles, le critere discrimine d'un facteur 4** (89,6 % contre 22,4 %) ; **sur
l'ensemble, il ne discrimine que d'un facteur 1,4** (75,7 % contre 54,1 %), et sur le seul film
Super Fiesta il s'inverse (`000d5950` : mesure 12/18 = 66,7 %, temoin 15/18 = 83,3 %). Le temoin
de fenetre n'est eligible que pour 292 apparitions sur 1 665 : il faut que la presence PROUVEE
soit plus large que l'intervalle a placer, ce qui exclut tout objet vu a une seule image-cle.

### 2.2 — l'oracle des loadouts : GATE 2 NON ATTEINT, et le detail dit ou

| jeu | denominateur | ACCORD | TEMOIN (autre joueur, meme image-cle) | rapport |
|---|---:|---:|---:|---:|
| **tous (le gate)** | **1 111** | **393 = 35,4 %** | 587 / 1 076 = **54,6 %** | **0,65** |
| `at_rest` (socles) | 177 | 111 = **62,7 %** | 6 / 170 = **3,5 %** | **17,8** |
| `dropped` | 770 | 247 = 32,1 % | 486 / 748 = 65,0 % | 0,49 |
| `echange` | 164 | 35 = 21,3 % | 95 / 158 = 60,1 % | 0,35 |

**LE SEUIL N'EST PAS REBAISSE : 35,4 % contre >= 90 % exige, le gate 2 tombe.** Et il tombe aussi
sur le sous-ensemble le plus favorable (62,7 % sur les socles). Mais la mesure separe nettement
deux populations que le gate traitait ensemble, et c'est le resultat :

- **sur les SOCLES, l'accord ECRASE son temoin — 62,7 % contre 3,5 %, un facteur 17,8.** Le
  ramasseur identifie porte l'arme du socle a l'image-cle suivante ; un autre joueur tire au sort
  ne la porte presque jamais. Et il ne la portait pas avant : « portait DEJA la famille » = 4 /
  177 = 2,3 %, donc l'accord est un accord sur des cas NOUVEAUX (107 / 173 = 61,8 %) ;
- **sur les armes `dropped` et les `echange`, l'accord est INFERIEUR a son temoin** (32,1 % contre
  65,0 % ; 21,3 % contre 60,1 %). Ce n'est pas un ramassage mal date : c'est que la disparition
  d'un `MA40 AR` lache a une mort est une DESPAWN, pas un ramassage, et que sa famille est portee
  par tout le monde — le temoin le mesure. Ces deux populations font 934 des 1 111 ramassages du
  denominateur du gate (84,1 %) : elles le gouvernent.

**PLAFOND STRUCTUREL, mesure** : 497 / 1 111 (44,7 %) des ramasseurs n'ont AUCUN loadout a
l'image-cle suivante — sur les socles, 51 / 177 (28,8 %). Cause : le slot d'un bipede MIGRE au
respawn (`offline_biped.go`), et 20 s suffisent a mourir. L'oracle ne peut donc pas depasser
55,3 % globalement ni 71,2 % sur les socles. Rapporte aux seuls cas ou il peut parler, l'accord
vaut **393 / 614 = 64,0 %** (tous) et **111 / 126 = 88,1 %** (socles) — chiffres publies pour ce
qu'ils sont, PAS substitues au denominateur du gate (decouverte 10).

**CONTROLE D'ESPACE DE SLOTS** (sans lui, un accord bas ne se distinguerait pas d'un decalage
d'index) : 931 / 962 = **96,8 %** des slots de bipede repliques a +-100 ms d'une image-cle y
portent aussi un loadout. Les deux chaines parlent bien du meme espace de slots.

### 2.3 — format PROPOSE pour la phase 3 (proposition, pas contrat)

Deux lignes, publiees telles quelles par l'instrument, une par objet et une par occupation de
socle. Le document de rejeu n'est PAS touche par ce lot.

```
PICKUP    film · carte · t (us) · x · y · z · famille · slot ramasseur|-1 · borne basse ·
          borne haute · statut(dated|unknown|never) · a bouge(bool)
PADSTATE  film · carte · famille · x · y · z (socle) · t apparition · t ou le socle redevient
          vide (-1 = jamais) · statut
```

`t` vaut l'instant du passage quand il existe, la borne haute sinon (regle du plan). `PADSTATE`
n'est emis que pour les SOCLES (>= 2 apparitions) du jeu `at_rest` : c'est l'etat present/vide
dans le temps que l'item 2.3 demande. **Reserve pour la phase 3** : `slot ramasseur` est le slot
du BIPEDE, qui migre au respawn — il ne designe pas un joueur, et le publier tel quel au document
demanderait d'abord l'attribution slot -> joueur (decouverte 10).

### 2.4 — LE CYCLE SE LIT DEPUIS LE RAMASSAGE, ET IL VAUT 30,5 s

| film | socles `at_rest` | cycle d'APPARITION etabli (1.3) | cycle depuis le RAMASSAGE etabli | GAGNES |
|---|---:|---:|---:|---:|
| `000d5950` | 0 | — | — | 0 |
| `bcb6d393` | 10 | 0 | **3 (30,0 %)** | 3 |
| `01e1f945` | 10 | 1 | **5 (50,0 %)** | 4 |
| `75f1188f` | 10 | 1 | **4 (40,0 %)** | 3 |
| `7f1bbf06` | 6 | 1 | **5 (83,3 %)** | 4 |
| `b974a390` | 7 | 1 | **4 (57,1 %)** | 3 |
| `b8d1fe0c` | 4 | 0 | 0 | 0 |
| `00162144` | 10 | 0 | **3 (30,0 %)** | 3 |
| **TOTAL** | **57** | **4 (7,0 %)** | **24 (42,1 %)** | **20** |

**20 socles passent de « non etabli » a « etabli »**, aux MEMES regles de stabilite que
l'item 1.3 (>= 2 ecarts, ecart-type <= 20 % de la mediane) — le seuil n'a pas bouge, c'est
l'horloge qui a change d'origine.

**LE MODE, MESURE SUR 142 ECARTS MIS EN COMMUN** : mediane 33,9 s, p10 19,8, p25 30,5, p75 77,5,
p90 120,2 — mais l'histogramme n'est pas une loi etalee, c'est **un pic** : **55 ecarts sur 142
(38,7 %) tombent dans la tranche 30-35 s**, et a l'interieur de cette tranche ils tiennent dans
**0,34 s** (min 30,30 · p25 30,43 · mediane 30,50 · p75 30,77 · max 33,87). Aucune autre tranche
de 5 s n'en porte plus de 8. Le pic est present sur les **7 films qui ont des socles** (23 % a
75 % de leurs ecarts). Les MEMES socles mesures d'APPARITION a apparition (item 1.3) donnent
mediane 99,1 s et seulement 20,0 % dans la tranche 30-35 s : **l'horloge repart bien du
ramassage, et non de l'apparition.**

**ET LES ARMES DE PUISSANCE SORTENT DU LOT, SANS QU'ON LE LEUR AIT DEMANDE.** Parmi les 24 socles
au cycle etabli : BR75, Bandit Evo, Mangler, Disruptor, Pulse Carbine, VK78 Commando -> 30,3 a
31,3 s ; **S7 Sniper 114,6 · 120,2 · 134,3 s ; Energy Sword 194,5 s ; Needler 100,9 s ; CQS48
Bulldog 40,1 et 80,0 s.** L'ordre de grandeur « 2-3 min pour une arme de puissance » que le plan
citait sans source se retrouve ici par la mesure seule. **Aucune comparaison a une source
officielle n'est faite** : il n'y en a pas hors ligne, et le plan interdit de comparer sans.
- 2026-08-17 — **arbitrage superviseur apres la phase 2** : gate 2 NON atteint (35,4 %) mais
  l'oracle suivait le slot de vie (plafond structurel 55,3 %) ; item 2.5 ajoute (oracle PAR
  JOUEUR via le pont `SlotXUID`, socles seulement, seuil 90 % inchange) ; perimetre de la phase 3
  tranche pour les deux issues (`weaponPads` + `padPickups`, xuid selon 2.5, jamais de `dropped`).
  Agent Opus lance sur 2.5.
- 2026-08-17 — **item 2.5 CLOS. GATE 2.5 NON ATTEINT (79,7 % contre >= 90 % exige), et la mesure
  REFUTE l'hypothese qui avait ouvert l'item.** Tout ci-dessous a tourne dans la session, aux
  largeurs de chaque carte, `CGO_ENABLED=0`, UN FILM PAR PROCESSUS, memes 8 films que les
  phases 1 et 2, meme chaine.

### Instrument et seuil (ecrits AVANT la mesure, commit `5546046b0` anterieur a la mesure)

`replay/ground_weapon_pickup_owner_test.go` : l'oracle par joueur, appele en DERNIER par
`TestGroundWeaponPickups` (garde `GW_PICKUP`). Trois tests sans garde entrent au gate ordinaire
(le loadout suit le joueur a travers le respawn ; deux slots d'un meme xuid ne fusionnent pas en
silence ; l'ordre du tirage du temoin est total).

**LE PONT EST CELUI DU CONSTRUCTEUR, PAS UN PONT MAISON** : `buildOwners` (owners.go) aux MEMES
entrees que `BuildFromFilm` — `ScanFilmDeaths`, `ScanFilmPlayerIndices` + `injectiveOrEmpty`,
`ScanFilmFireEvents` -> `fireRefs` (les fermetures), `indexBySlot` sur les positions deja lues.
C'est `own.SlotXUID` que le document publie sur `Track.XUID` (`build.go`, `nameTracks`) : mesurer
sur autre chose n'aurait rien dit de ce qui serait publie.

| seuil / regle | valeur | source |
|---|---|---|
| population | ramassages de SOCLE (`at_rest`) DATES | arbitrage, decouverte 11 |
| ramasseur | `SlotXUID[slot du passage]` ; absent => SANS PONT, hors denominateur | item 2.5 |
| vie courante | le slot du MEME xuid portant un loadout a l'image-cle visee ; plusieurs => le plus petit, et le cas est COMPTE | ecrit avant mesure |
| loadout observable | un tel slot existe ; sinon hors denominateur et publie a part | item 2.5 |
| temoin | un AUTRE xuid tire au sort parmi ceux qui ont un loadout observable a la MEME image-cle, graine 20260817 | item 2.5 |
| controle NOUVEAU | derniere image-cle PRECEDANT la date, meme regle de vie courante — publie SANS seuil | item 2.5 |
| **GATE 2.5** | **accord >= 90 %** sur le denominateur ci-dessus | plan (seuil du gate 2) |

### Controle d'ancrage : l'ajout de 2.5 ne deplace RIEN

Les 8 films ont d'abord ete rejoues avec l'instrument INCHANGE (avant tout ajout) : **177
ramassages de socle dates, 111 accords = 62,7 %, temoin 6 / 170 = 3,5 %** — les chiffres de la
phase 2, au chiffre pres. Apres l'ajout de 2.5, les sorties des items 2.1 a 2.4 sont **identiques
ligne pour ligne sur les 8 films** (diff = 0 ligne). C'est voulu et c'est construit : le rapport
2.5 est appele APRES les autres et son temoin a sa propre source a la meme graine — sans quoi il
aurait deplace les tirages de 2.1 et 2.2.

### 2.5 — l'oracle par joueur, et ce que son denominateur coute

| film | socles dates | SANS PONT | sans loadout obs. | DENOM | ACCORD | TEMOIN | rapport |
|---|---:|---:|---:|---:|---:|---:|---:|
| `000d5950` | 0 | — | — | 0 | — | — | — |
| `bcb6d393` | 19 | 1 (5,3 %) | 4 (21,1 %) | 14 | **14 (100 %)** | 1/14 = 7,1 % | 14,0 |
| `01e1f945` | 29 | 5 (17,2 %) | 3 (10,3 %) | 21 | 16 (76,2 %) | 2/21 = 9,5 % | 8,0 |
| `75f1188f` | 25 | 0 | 6 (24,0 %) | 19 | 16 (84,2 %) | 0/19 = 0,0 % | — |
| `7f1bbf06` | 19 | 0 | 4 (21,1 %) | 15 | 12 (80,0 %) | 1/15 = 6,7 % | 12,0 |
| `b974a390` | 39 | 1 (2,6 %) | 10 (25,6 %) | 28 | 24 (85,7 %) | 0/28 = 0,0 % | — |
| `b8d1fe0c` | 6 | 4 (66,7 %) | 1 (16,7 %) | 1 | 1 (100 %) | 0/1 = 0,0 % | — |
| `00162144` | 40 | 0 | 10 (25,0 %) | 30 | 19 (63,3 %) | 2/30 = 6,7 % | 9,5 |
| **TOTAL** | **177** | **11 (6,2 %)** | **38 (21,5 %)** | **128** | **102 = 79,7 %** | **6 / 128 = 4,7 %** | **17,0** |

`000d5950` est le Super Fiesta sur variante Forge : il n'a aucun socle depuis l'item 1.2, donc
aucun ramassage de socle a mesurer. Ce n'est pas une valeur manquante, c'est un zero mesure.

### 2.5 — le controle « ramassage NOUVEAU », et l'anomalie qu'on ne tait pas

| film | loadout obs. a l'image-cle PRECEDENTE | portait DEJA | NOUVEAUX | accord sur les NOUVEAUX | xuid a PLUSIEURS slots |
|---|---:|---:|---:|---:|---:|
| `bcb6d393` | 12 | 0 (0,0 %) | 12 | 12 (100 %) | 0 |
| `01e1f945` | 16 | 2 (12,5 %) | 14 | 9 (64,3 %) | 4 |
| `75f1188f` | 15 | 1 (6,7 %) | 14 | 12 (85,7 %) | 1 |
| `7f1bbf06` | 8 | 0 (0,0 %) | 8 | 5 (62,5 %) | 0 |
| `b974a390` | 19 | 0 (0,0 %) | 19 | 15 (78,9 %) | 0 |
| `b8d1fe0c` | 1 | 0 (0,0 %) | 1 | 1 (100 %) | 0 |
| `00162144` | 23 | 1 (4,3 %) | 22 | 14 (63,6 %) | 0 |
| **TOTAL** | **94** | **4 (4,3 %)** | **90** | **68 (75,6 %)** | **5** |

**LE CONTROLE TIENT** : 4 ramassages sur 94 portaient deja la famille a l'image-cle precedente
(4,3 %, a comparer aux 2,3 % mesures par slot a l'item 2.2). L'accord ne repose donc pas sur des
armes deja portees — il porte sur des ramassages NOUVEAUX. **L'anomalie est publiee telle
quelle** : 5 fois, un meme xuid porte un loadout sur PLUSIEURS slots a la meme image-cle, ce qui
ne devrait pas arriver. La regle ecrite avant la mesure retient le plus petit slot ; corriger ces
5 cas dans le sens le plus favorable donnerait 107 / 128 = 83,6 %, toujours sous le seuil.

### 2.5 — la qualite du pont, film par film (il n'est PAS la cause)

| film | morts | slots pontes | vies nommees | par lecture | fermetures tir / respawn | lectures d'index | desaccords | collisions | joueurs |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `000d5950` | 93 | 93 | 90/105 | 90 | 0 / 3 | 26 | 0 | 0 | 8 |
| `bcb6d393` | 49 | 54 | 48/58 | 48 | 1 / 5 | 19 | 0 | 0 | 11 |
| `01e1f945` | 105 | 100 | 97/113 | 97 | 1 / 2 | 28 | 0 | 0 | 8 |
| `75f1188f` | 80 | 82 | 79/85 | 79 | 1 / 2 | 22 | 0 | 0 | 8 |
| `7f1bbf06` | 60 | 67 | 60/71 | 60 | 1 / 6 | 17 | 0 | 0 | 8 |
| `b974a390` | 141 | 142 | 138/149 | 138 | 0 / 4 | 35 | 0 | 0 | 8 |
| `b8d1fe0c` | 31 | 33 | 31/44 | 31 | 0 / 2 | 12 | 0 | 0 | 7 |
| `00162144` | 92 | 98 | 92/108 | 92 | 0 / 6 | 23 | 0 | 0 | 8 |

**Zero collision de slot et zero desaccord d'index sur les huit films** : la table slot -> joueur
est licite partout ou elle existe. Les deux films qui perdent des ramasseurs sont ceux dont le fil
des morts nomme le moins de vies (`b8d1fe0c` 31/44 = 70,5 %, 4 des 6 ramassages sans pont ;
`01e1f945` 97/113 = 85,8 %, 5 sans pont sur 29).

### 2.5 — LA COMPARAISON QUI TRANCHE, sur la meme espece de restriction

| oracle | denominateur | accord | plafond (part du 177 mesurable) | temoin | rapport |
|---|---:|---:|---:|---:|---:|
| par SLOT de vie (item 2.2) | 126 | **111 = 88,1 %** | 126/177 = 71,2 % | 3,5 % | 17,8 |
| par JOUEUR (item 2.5) | 128 | **102 = 79,7 %** | 128/177 = 72,3 % | 4,7 % | 17,0 |

Les deux lignes sont restreintes de la MEME facon — aux cas ou l'oracle peut parler — et le
resultat est net : suivre le joueur gagne **un point de plafond** (71,2 % -> 72,3 %) et **perd
huit points d'accord** (88,1 % -> 79,7 %). Le rapport au temoin, lui, ne bouge pas (17,8 -> 17,0) :
les deux oracles mesurent bien quelque chose, mais celui par joueur ne mesure pas mieux.

**POURQUOI, ET C'EST LE RESULTAT DU LOT.** Un joueur qui meurt entre le ramassage et l'image-cle
suivante LACHE ce qu'il a ramasse. Par slot, ce cas etait MUET (le slot d'origine a disparu). Par
joueur, il n'est plus muet : la nouvelle vie est bien la, avec le loadout de REAPPARITION, qui ne
porte pas l'arme du socle. **Le pont ne transforme donc pas un silence en accord, il transforme un
silence en DESACCORD** — d'ou un plafond qui bouge a peine et un accord qui baisse. Le plafond des
55,3 % / 71,2 % de la decouverte 10 n'etait pas un artefact du pont : c'est l'espacement de 20 s
des images-cles, et la mort qui tient dedans.
- 2026-08-17 — **item 2.5 CLOS `[!]`** (`5546046b0`, `2c0c79842`) : 79,7 % contre 90 %, le pont
  n'est pas le plafond (l'espacement de 20 s des images-cles l'est) ; **ramasseur NON publiable
  (`null`)**. Perimetre de la phase 3 inchange (arbitrage apres la phase 2) : **phase 3 LANCEE**
  (agent Opus, principal) — `weaponPads` + `padPickups` sans ramasseur, schema 11.
