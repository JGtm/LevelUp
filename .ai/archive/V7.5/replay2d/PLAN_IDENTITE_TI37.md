# Plan — Mur de protection et capteur de menaces : l'IDENTITE des entites `ti=37`, par le record de creation

> Ecrit le 2026-08-17. Priorite utilisateur (bilan de la planche, F1) : « j'aimerais les murs
> de protection et capteurs de menaces, on reflechira a l'UI plus tard, sujet a pousser
> davantage niveau investigation ». Branche `feat/v75`, worktree principal (films), contrat
> `plan-execution`. Lot de MESURE + PUBLICATION ; le rendu (capteur = zone radar pulsee, mur =
> segment) est un lot ULTERIEUR, apres decision UI de l'utilisateur.

## Ce qui est acquis (15/08 et 16/08 — ne pas re-mesurer)

| acquis | ou | consequence |
|---|---|---|
| Les entites `ti=37` ont une POSITION fiable : 9 043 trajectoires / 628 368 echantillons, 97,2 % dans l'emprise du nuage des bipedes (aux largeurs de la carte depuis `4e2084d8e`) | `equipment_state_test.go`, `ScanFilmWorldObjects(dir, wr, 37)` | le OU est acquis, il manque le QUOI et le QUI |
| Aucun des 4 champs delta mesures (`deployed`, `activated`, `creator` R(5), `energy`) ne porte d'IDENTITE ; `creator` R(5) n'est ni un slot ni un index de joueur (0/1 328 dans la bande de slots ; sur `000d5950`, 8 joueurs, 15 valeurs distinctes, max 28) | `equipment_state.go`, PLAN_EQUIPEMENT_TI37 phase 0 | ne pas rejouer ces quatre champs pour l'identite |
| `ti=37` porte des objets qui NE SONT PAS des equipements de joueur (352 slots vs 74 bipedes en live ; `item-at-rest`, `item-ignore-player`) | dumps CE, phase 0 | l'identite doit DISCRIMINER : mur / capteur / autre chose |
| **Le RECORD DE CREATION (default state) de `ti=37` est deja porte, et il porte deux champs jetes** : `consumeDefaultStateTI37` = `V ; deser ti36 ; ECS_ReadEntityRefIndex5 (R(1) ; si 0 -> R(5)) ; g = R(1) ; si g -> R(32) « ability-enabled-id »` | `default_state_arch.go`, decompile FUN_1407f105c | **c'est la piste** : un identifiant 32 bits a la creation (une DEFINITION de capacite ?) et une reference d'entite sur 5 bits (le CREATEUR ?) |
| Le patron de publication d'une valeur jetee : hook sur le deser de production, statistiques de denominateur | `ability_rank.go`, `equipment_state.go` | pas de decodage nouveau : un hook sur le default state |
| L'identite du porteur par vie : `i48` (rang de palette, palette PAR FILM ; fam. A : 1 detecteur, 2 mur, 12 traqueur ; fam. B : 19 mur, 22 capteur) ; le pont slot -> joueur : `PlayerIndexTable` (`player_index.go`, index de joueur du FILM 0..N) | `ability_rank.go`, `replay/player_index.go` | les CONTROLES croisent l'entite creee avec le porteur |
| Deux mesures de la mesure du 16/08 : `activated` 1 transition / 3 films ; naissances `ti=37` 4,7-5,3 par seconde (fenetres saturees) | PLAN_ETAT_ACTIF phase C | la DATE de pose = le PREMIER record de la vie d'objet, pas `activated` |

## Objectif et critere de succes

Publier, par match, les POSES d'equipement — `{t0, t1, x, y, type ∈ {mur, capteur, autre}, createur}` —
avec un type MESURE (jamais devine) et un createur MESURE ou absent. Le critere est chiffre
et fixe d'avance (gates ci-dessous). Si l'identifiant de creation ne discrimine pas, le negatif
s'ecrit et rien n'est publie sous un nom.

## Decisions tranchees avant execution

1. **Un nom (mur / capteur) ne se pose que par DOUBLE chaine** : (a) l'identifiant de creation
   est STABLE par type (peu de valeurs distinctes, les memes d'un film a l'autre) ET (b) il se
   croise avec le rang `i48` du createur (un objet cree par un porteur de rang « mur » porte
   l'identifiant du mur ; par un porteur de rang « capteur » celui du capteur), avec un temoin
   interne (les porteurs d'autres rangs ne creent aucun objet de ces deux identifiants).
   Regle du depot, RECETTE §14 : un temoignage isole ne vaut pas.
2. **Le createur se prouve par la GEOMETRIE et le TEMPS**, pas par la seule reference : a
   l'instant du premier record de l'objet, le createur candidat doit etre a moins de X m de
   l'objet (X mesure, seuil enonce avant : mediane < 3 m, 90 % < 6 m — un mur se pose devant
   soi, un capteur se lance a quelques metres). Une reference qui ne satisfait pas la
   geometrie n'est pas un createur.
3. **La date de pose est le premier record de la vie d'objet** ; la fin est la disparition
   (dernier record) — les deux publies tels quels, sans remanence.
4. **Le rendu n'est pas dans ce lot** ; la publication (schema 9) l'est, pour que le lot UI
   n'ait qu'a dessiner. Un type `autre` est publie SANS nom (jamais « mur » par defaut).
5. Un seul decodage filmdec par process ; instruments gardes ; `CGO_ENABLED=0` ; aucune base
   DuckDB en ecriture ; JAMAIS `git add -A` (fichiers d'une autre session dans l'arbre) ;
   jamais de pause d'attente passive.

## Phases

### Phase 0 — LIRE le record de creation (hook, zero decodage nouveau) — CLOSE le 2026-08-17

- [x] 0.1 Hook sur `consumeDefaultStateTI37` (`filmdec/equipment_creation.go`,
      `SetEquipmentCreationHook`) : les deux feuilles publient au lieu de jeter, largeurs
      INCHANGEES. Le record de creation est atteint par un balayage d'en-tete NEW dans les
      paquets delta — `[0][01][slot:13][gen:2][ti:6=37]` (24 bits), puis le default-state, la
      porte has-components, le masque, et i0. Denominateurs publies par
      `EquipmentCreationStats` (ancres, debordement, masque invalide, position rejetee,
      acceptes, portes ouvertes).
- [x] 0.2 DISTRIBUTION. **`abilityEnabledID` n'est JAMAIS transmis** : porte fermee sur
      274 records de creation sur 274 (`000d5950`) et 229 sur 229 (`00162144`).
      `entityRef5` non plus : 0 sur 274. La prediction du plan est REFUTEE, et la refutation
      est propre — le default-state est lu BIT-EXACT (profil de bits :
      `equipment_creation_offset_test.go` mesure la distance en-tete -> masque a 85 bits
      = 24 + 60 + 1, exactement ce que le deser porte predit, sur 91 records localises par un
      oracle de position independant).
- [x] 0.3 **L'IDENTITE EST LA, mais dans un AUTRE champ du MEME record** : le mot de 32 bits
      INCONDITIONNEL du bloc `object-multiplayer-properties` (`FUN_14080d6f0`, publie sous
      `MPPWord32`) prend **8 valeurs distinctes sur 274 records** (`000d5950`) et **6 sur 229**
      (`00162144`), dont **3 communes aux deux films**. Croisement avec les tags du jeu
      (`himap/sonde_ti37_gamefiles_test.go`) : **11 valeurs sur 11 se resolvent, et TOUTES
      dans le groupe `eqip`** — 105 tags `eqip` sur 148 097 tags parcourus. Ce n'est ni du
      bruit ni une enumeration opaque : c'est le **GlobalID du tag d'equipement de l'objet**.

**Gate 0 : PASSE**, avec correction du plan. L'identite est dans le record de creation, mais
pas dans le champ que le plan designait. `ability-enabled-id` et `entity-ref-index5` sont des
NEGATIFS mesures (portes fermees, 503 records sur 503) — ils vont au registre.

**CE QUE LE CORPUS N'A PAS COUVERT, et pourquoi.** Le plan demandait les 12 films du 15/08.
Quatre ont ete balayes (`000d5950`, `00162144`, `00ba2e1c`, `06dfe6d9`) et le balayage a ete
ARRETE la : sur les deux derniers, la largeur du default-state n'est pas celle du defaut
(cf. phase 2), et un balayage a la mauvaise largeur ne rend pas une mesure, il rend du bruit —
le publier comme « distribution du film » aurait fabrique un resultat. Les huit films restants
se mesureront quand la largeur sera detectee de facon fiable, dans la meme passe que la
phase 2. Les deux films retenus suffisent au gate : ils portent 503 records lus bit-exact, la
verite terrain Theater, et les deux familles de palette.

### Phase 1 — PROUVER le type et le createur (double chaine) — CLOSE le 2026-08-17

- [x] 1.1 Table `eqip` x rang `i48` — **et non `abilityEnabledID` x rang**, puisque le premier
      n'est jamais transmis. Le createur ne se LIT pas (aucune reference a la naissance) : le
      porteur candidat est le bipede le PLUS PROCHE dans une fenetre de 250 ms.
      Instrument : `filmdec/equipment_creation_owner_test.go`.
      **`000d5950` (palette famille B — 19 mur, 20 grappin, 21 propulseur, 22 capteur)**,
      diagonale calculee sur les rangs CONNUS (i48 ne transmet qu'une fois par vie ; une vie
      sans lecture rend un rang inconnu, compte a part) :

          0x008e2dc574   19:9  20:1                ->  90 %   MUR DE PROTECTION
          0x0072199cba   22:12 21:1                ->  92 %   CAPTEUR DE MENACES
          0x008c77ffe7   20:20 19:1                ->  95 %   grappin
          0x00eef5d48d   21:19 19:1 20:1 22:1      ->  86 %   propulseur
          0x00528fce46   19:8  21:1 22:1           ->  80 %   mur (2e identifiant)
          0x000f5716ff / 0x00aada07f3 / 0x00bcabbe43 / 0x00caaadcb0  ->  27 a 45 %  PLAT

      **`00162144` (palette famille A — 2 = mur)** : `0x00686b40c9` 2:9 sur 9 rangs connus
      (**100 %**), `0x002974c233` 2:10 / 9:1 (**91 %**), `0x00273fe0eb` 4:8 / 9:1 (89 %,
      grappin), et les MEMES `0x00bcabbe43` / `0x00caaadcb0` / `0x000f5716ff` restent PLATS.
      Les identifiants plats sont les objets du monde (bonus, socles, `item-at-rest`) : ils
      n'ont pas de poseur, et c'est la mesure qui le dit.
- [x] 1.2 CONTROLE GEOMETRIQUE, en METRES (bornes `cliffhanger` du catalogue
      `map_quant_bounds.json`, emprise 113,2 x 113,8 x 137,6 m) :
      **porteur le plus proche mediane 0,575 m · p90 1,341 m** contre
      **temoin (autre bipede vivant au meme instant) mediane 14,127 m · p90 31,969 m**.
      Le seuil du plan (mediane < 3 m, p90 < 6 m) est tenu avec une marge de 4x, et le temoin
      est 25 fois plus loin.
- [!] 1.3 CONTROLE TEMPOREL par `i57` v==1 : **NON TRAITE**. La double chaine est deja etablie
      par 1.1 (deux canaux independants) et 1.2 (geometrie + temoin). Ajouter un troisieme
      canal n'aurait pas change le verdict et aurait ouvert le nommage d'`i57`, qui est un
      autre sujet. Report au registre.
- [x] 1.4 VERITE TERRAIN `000d5950` (releve Theater du 27/07 : 1 mur, 1 capteur parmi
      8 joueurs) : **2/2**. Un identifiant domine au rang 22 (capteur, 92 %) et un au rang 19
      (mur, 90 %) ; aucun autre film-objet ne les revendique.

**Gate 1 : PASSE** — diagonale 90 % et 92 % sur les DEUX types demandes (le seuil du plan),
temoin interne 8 a 10 % (les rangs autres que le rang dominant), temoin externe plat,
geometrie 0,575 / 1,341 m, verite terrain 2/2.

### Phase 2 — PUBLIER (schema 9) — NON TRAITE

- [!] 2.1 / 2.2 / 2.3 **BLOQUE PAR LA COUVERTURE, et le blocage est mesure.** Le
      default-state de ti=37 ne fait pas la meme largeur d'un film a l'autre : 60 bits sur
      `000d5950` et `00162144`, **57 sur `06dfe6d9` et `00ba2e1c`** (mesure par oracle de
      position : la distance en-tete -> masque passe de 85 a 82 bits, avec 150 records a 82 et
      98 a 426 sur `06dfe6d9`). L'ecart de 3 bits est porte par le PREMIER champ du bloc MPP
      (`FUN_141fd72c0`, `R(9)` au decompile) : c'est une largeur de configuration de
      REPLICATION, du meme genre que `WorldObjectPrecision` — posee au chargement de la carte,
      absente de l'executable. `mppLeadBits` + `DetectMPPLeadBits` sont en place et la
      detection retient bien 6 quand on la force, mais elle ECHOUE en automatique sur
      `06dfe6d9` (moins de 20 records dans les 3 premiers chunks). Publier `equipmentPlacements`
      dans cet etat livrerait un champ plein sur les films d'arene et vide ailleurs, sans que
      l'artefact le dise : c'est exactement la demi-livraison que le depot refuse.

**Condition de reprise** (registre) : rendre `DetectMPPLeadBits` fiable — elargir la fenetre
de detection (chunks et seuil), ou mieux, SOURCER la largeur comme les largeurs d'axe (une
entree de catalogue par carte, controlee par la detection). Une fois la largeur juste sur les
12 films du corpus, la phase 2 se deroule telle qu'ecrite.

### Phase 3 — PROPOSER l'UI (pas la coder) — CLOSE le 2026-08-17

- [x] 3.1 Note de proposition : voir « Ce que la donnee permet a l'ecran » ci-dessous.

## Ce que la donnee permet a l'ecran (note pour l'utilisateur, phase 3)

**Ce qui est disponible, par pose** : l'instant de la pose (le record de creation est date, et
il precede le premier record delta de la vie dans 268 cas sur 271), la POSITION en coordonnees
monde, le TYPE (mur de protection / capteur de menaces / grappin / propulseur / objet du monde
non revendique), et le POSEUR par proximite (0,575 m median). La fin de vie se lit deja par le
dernier record de la trajectoire (`ScanFilmWorldObjects`), donc `[t0, t1]` est complet.

**Ce qui manque, et il faut le dire avant de dessiner** :

1. **L'ORIENTATION du mur.** Le record de creation ne porte aucun cap : le mur ne peut pas
   etre dessine comme un segment oriente sans l'inventer. Deux sorties honnetes : (a) un
   rectangle/disque centre sur la position, sans direction ; (b) deduire le cap du VISEE du
   poseur au meme instant (`i2 forward`, deja decode par `CaptureDirs`) — c'est une mesure
   supplementaire, pas une supposition, mais elle n'est pas faite.
2. **Le mur, c'est DEUX identifiants** (`0x008e2dc574` et `0x00528fce46` sur `000d5950`,
   `0x00686b40c9` et `0x002974c233` sur `00162144`), tous deux au rang du mur. Vraisemblable-
   ment l'appareil lance et les panneaux deployes. A l'ecran, ce sont deux calques possibles :
   un seul marqueur par pose, ou la silhouette reelle des panneaux.
3. **Le capteur n'a qu'un identifiant** et sa position est un point : la « zone radar » est un
   choix de RENDU (rayon arbitraire), pas une donnee.

**Proposition de rendu** (a trancher par l'utilisateur) : capteur = disque pulse a la position,
de t0 a t1, a la couleur du poseur ; mur = rectangle plein sans direction (ou segment oriente
si la mesure du cap de visee est faite) ; les identifiants PLATS ne sont pas dessines du tout
tant qu'ils ne sont pas nommes. Bascule dans le tiroir, a cote des effets de tir et de mort.
Sons `Drop Wall - Activate` / `Threat Sensor - Activate` a t0, depuis la bibliotheque de
l'utilisateur.

## Regles dures

Aucun nom sans double chaine ; un type non prouve se publie `autre` ; un createur non prouve
se publie -1 ; les seuils ne se rebaissent pas ; zero fix hors perimetre ; decouvertes au
registre.

## Statuts et cloture

`[x]` / `[~]` / `[!]` ; aucune case vide ; journal date ; registre (la ligne « Identite de
l'objet ti=37 » sort si le gate 1 passe) ; commits sur `feat/v75`, pas de push.

## Journal

### 2026-08-17 — phases 0, 1 et 3 closes ; phase 2 reportee

**Ce que le lot a change au plan lui-meme.** Le plan pariait sur `ability-enabled-id`. Le pari
est PERDU et la perte est nette : sur les 503 records de creation lus bit-exact des deux films
d'arene, la porte de ce champ est fermee 503 fois sur 503, et celle d'`entity-ref-index5`
aussi. Ce n'est pas un echec de decodage — le profil de bits du record concorde champ par champ
avec le deser porte (`consumeDefaultStateTI37`), et la distance en-tete -> masque tombe a
85 bits = 24 + 60 + 1 sur 91 records localises par un oracle de POSITION independant du
decodage. Le champ existe, il n'est simplement jamais transmis pour ces objets.

**Ce qui l'a remplace, et pourquoi c'est mieux.** Le MEME record porte le bloc
`object-multiplayer-properties`, dont le mot de 32 bits inconditionnel est le **GlobalID du tag
`eqip` de l'objet** : 11 valeurs observees sur 11 se resolvent dans le groupe `eqip` du jeu
(105 tags sur 148 097 parcourus). Le plan cherchait une enumeration a interpreter ; on a une
REFERENCE DE DEFINITION, qui se nomme par le jeu et non par une table de correspondance.

**Le nommage tient par les deux chaines exigees** : (a) l'identifiant est stable — 3 valeurs
communes aux deux films, et les memes identifiants PLATS (objets du monde) reviennent d'un film
a l'autre ; (b) il se croise avec le rang `i48` du porteur le plus proche a 90 % (mur) et 92 %
(capteur), temoin externe plat, geometrie 0,575 m mediane contre 14,127 m pour le temoin.

**Ce qui bloque la publication** : la largeur du premier champ du bloc MPP varie d'un film a
l'autre (9 bits sur les films d'arene, 6 sur `06dfe6d9`/`00ba2e1c`). Elle est desormais
parametrable (`mppLeadBits`) et detectable (`DetectMPPLeadBits`), mais la detection automatique
echoue sur les gros films. Tant qu'elle n'est pas fiable, `equipmentPlacements` serait plein
sur deux films et vide sur les autres.

**Instruments livres** (tous gardes par `EQUIP_CREATION_FILM`, sautes en CI) :
`filmdec/equipment_creation_test.go` (distributions et temoin fantome),
`equipment_creation_offset_test.go` (oracle de position -> largeur du default-state),
`equipment_creation_owner_test.go` (croisement `eqip` x rang, geometrie, temoin),
`himap/sonde_ti37_gamefiles_test.go` (resolution des GlobalID contre les modules du jeu).

**Decouvertes non traitees** (hors perimetre, portees au registre) : le balayage des records de
creation sur `00ba2e1c` et `06dfe6d9` rend surtout du bruit tant que la largeur MPP n'est pas
detectee ; le `sofd` ne porte PAS les GlobalID `eqip` en clair (0 reference trouvee sur les
119 palettes) — la chaine rang -> `eqip` passe donc par la mesure du film, pas par le tag.
