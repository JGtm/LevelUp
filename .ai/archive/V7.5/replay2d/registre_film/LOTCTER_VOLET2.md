# Lot C-ter — volet 2 : les FORMES de colline et le role `hill`

> Perimetre : CT.2.1, CT.2.2, CT.2.3 du plan (`PLAN_EXPLOITATION_REGISTRE_FILM.md` §« Lot C-ter »).
> Branche `wt/colline-formes`, base `feat/v75` `638c4d044`. Worktree `LevelUp-wt-colline-formes`.
> Gates : `LOTCTER_volet2_gates.log`. Mesures : `lotCter/volet2_<film>_*.log`.
> Corpus KOTH : `01e1f945` (Catalyst), `606d9844` (Chasm), `8076f97f` (Shogun), `0a247154`
> (Solitude - Ranked).

## 0. Le resultat, en une phrase

**Les collines de KOTH VIVENT DANS LA VARIANTE DE CARTE** — le meme `.mvar` que le catalogue lit
deja — sous une paire de labels non nommes ; elles sont au catalogue (21 formes sur les 4 cartes des
films, Solitude - Ranked entree au passage), servies par la table en KOTH sous le role `hill`, et
l'artefact les apparie par le meme chemin que Bastion (le repli sur les formes de Bastion/Extraction
a disparu, le balayage `ti=13` ne se paie plus qu'en Bastion/KOTH). **La clause « >= 90 % » du gate 2
n'est PAS tenue** sous la definition ecrite avant la mesure — 83,3 % / 57,1 % / 86,4 % (82,0 % sur
89 periodes de 3 films ; le 4e n'a aucune rampe) — mais les temoins montrent que c'est la
SEMANTIQUE de la jauge qui manque, pas les formes : deplacees de 6 m, elles tombent a 9-18 %
d'occupation contre 82-86 % en place, et la production apparie plus de rampes qu'avant sur les
trois films (13 des 20 periodes publiees de `01e1f945` changent de forme). Le repli d'affichage
(cercle synthetique) est laisse a la decision de l'utilisateur, non code.

## 1. CT.2.1 — L'INVENTAIRE : ou vivent les collines

### 1.1 Ce qui a ete fait

- `cmd/mapobj-build --dump-objects` etendu (`variant.go`) : le dump publie desormais les labels
  NON RESOLUS en hash brut (`unresolved_labels`) et l'`instance_id`. Sans cela l'inventaire etait
  aveugle : le dump ne montrait que ce que la table savait nommer — c'est-a-dire rien de KOTH.
  Test : `TestDumpPublieLesLabelsNonResolusEtLInstance` (Cliffhanger).
- Dumps joues sur les 8 variantes des 4 cartes (lecture seule de `.ai/re_dump/mapvar/` du
  principal) : `chasm_map` (348 objets) / `chasm_chasm` (326), `catalyst_map` (357) /
  `catalyst_catalyst` (337), `shogun_map` (5 397) / `shogun_fo11_blank` (100, le canevas Forge),
  `solitude_-_ranked_map` (3 227) / `solitude_-_ranked_fo11_blank` (100, canevas). NOTE : la
  variante de carte de `0a247154` (Solitude - Ranked, `4a5e5612`) est ABSENTE du catalogue
  actuel (son `.mvar` est dans le depot de dumps) — CT.2.2 l'y fait entrer.
- Instrument de chasse aux labels sous garde (`mapvar/label_hunt_test.go`,
  `MAPVAR_LABEL_HUNT_DIR`) : radicaux x suffixes, force brute sur l'alphabet.
- Instrument de mesure sous garde (`replay/hill_shapes_measure_test.go`, `ZONE_FILM` +
  `HILL_MVAR`/`HILL_LABEL`/`HILL_INCLUDE` ou catalogue role `hill`) : croise les formes candidates
  avec les rampes de jauge du film — definitions et seuils ecrits en tete du fichier AVANT la
  mesure (§1.4).

### 1.2 Les objets non resolus des variantes de carte

Sur les quatre `*_map.mvar` retenus par le catalogue, TROIS hashs non resolus sont communs et
forment un motif ferme (les autres hashs sont des filtres de mode : `landgrab_include`
-886053664 / `landgrab_zone` 996801386 craques au passage, `strongholds_exclude` -1191131830,
un couple `doubles_include` 390628080 / `doubles_exclude` 1081676545 sur Solitude - Ranked, cf.
§6) :

| hash | porte par | Catalyst | Chasm | Shogun | Solitude-R | lecture |
|---|---|---:|---:|---:|---:|---|
| **-767961569** | volumes (type `-1476457415`, boite/cylindre, neutres, TOUS avec forme) — ET 2 cylindres `minigame_include` par carte de developpeur (type `-1855279381`, r = 1,4 m, equipe 0/1, aux bases) | 6 (+2) | 5 (+2) | 5 | 5 | le ROLE (la colline) |
| **2133978317** | ces memes volumes + les marqueurs ponctuels (type `-877512201`, sans forme, un par colline, a la position de la colline) | 12 | 10 | 10 | 13 | le FILTRE DE MODE de KOTH (« include ») |
| **-1482301937** | les marqueurs ponctuels seulement | 6 | 5 | 5 | 8 | le marqueur de colline (nav/HUD ?) — non exploite |

Le motif est celui des autres modes du meme fichier : Land Grab = volumes `-1902667258`
`[landgrab_include, landgrab_zone]` + marqueurs `-877512201` `[landgrab_include, -941529218]` ;
Bastion = volumes `1818458590` `[strongholds_include, strongholds_zone]` + marqueurs
`[strongholds_include, -1246645531]`. KOTH = volumes `-1476457415` `[2133978317, -767961569]` +
marqueurs `[2133978317, -1482301937]`. Chaque mode a SON type de volume, son include, son role et
son marqueur.

Geometrie des volumes KOTH (demi-extents en metres, `up`/`down` depuis le centre) :

| carte | idx | forme | centre (x ; y ; z) |
|---|---:|---|---|
| Catalyst | 311 | boite 3,00 x 3,20, +1,79/-0,50 | (-14,75 ; -0,01 ; 22,32) |
| Catalyst | 328 | boite 2,25 x 1,75, +2,00/-0,00 | (-13,73 ; -8,95 ; 23,85) |
| Catalyst | 313 | cylindre r 2,00, +3,00 | (0,00 ; 14,70 ; 24,70) |
| Catalyst | 314 | boite 1,00 x 4,65, +2,05/-0,23 | (0,00 ; 0,00 ; 27,30) |
| Catalyst | 312 | cylindre r 3,20, +2,33 | (12,97 ; -0,01 ; 22,08) |
| Catalyst | 315 | boite 2,06 x 1,58, +1,77 | (14,12 ; -0,04 ; 25,31) |
| Chasm | 129 | boite 1,65 x 4,00, +4,00 | (-73,62 ; -53,97 ; -136,43) |
| Chasm | 132 | cylindre r 0,80, +2,10 | (-67,20 ; -62,20 ; -133,97) |
| Chasm | 133 | cylindre r 3,00, +1,80 | (-67,18 ; -62,20 ; -138,70) |
| Chasm | 130 | boite 5,25 x 1,50, +1,80 | (-54,40 ; -70,19 ; -138,83) |
| Chasm | 131 | boite 2,50 x 1,90, +1,90 | (-50,68 ; -54,05 ; -138,83) |
| Shogun | 4220 | boite 2,04 x 2,80, +1,96 | (-45,80 ; 0,00 ; 64,85) |
| Shogun | 4225 | cylindre r 3,55, +3,44 | (-37,95 ; 7,75 ; 63,05) |
| Shogun | 4218 | boite 1,89 x 2,00, +1,80 | (-37,95 ; 16,10 ; 63,95) |
| Shogun | 4223 | cylindre r 1,65, +2,00 | (-21,75 ; 14,60 ; 63,45) |
| Shogun | 2787 | boite 1,25 x 1,25, +2,00 | (-19,15 ; 1,10 ; 66,75) |
| Solitude-R | 964 | boite 1,90 x 2,92, +2,00 | (-15,78 ; -12,11 ; 80,58) |
| Solitude-R | 1431 | boite 1,67 x 4,50, +1,85 | (-3,79 ; -15,90 ; 77,88) |
| Solitude-R | 963 | boite 1,82 x 1,88, +1,65/-0,05 | (-3,77 ; -32,16 ; 78,87) |
| Solitude-R | 965 | boite 1,42 x 1,67, +1,75/-1,00 | (1,16 ; -7,81 ; 79,52) |
| Solitude-R | 962 | boite 2,61 x 2,25, +1,70 | (3,15 ; -26,67 ; 80,65) |

Les tailles sont de l'ordre de celles des zones de Bastion/Extraction du meme fichier (meme
lecture « tailles pleines », shape.go) ; sur Catalyst trois collines COINCIDENT avec une zone d'un
autre role (311 ~ Bastion 316, 312 ~ Extraction 335, 313 = Bastion 318, 328 = Land Grab 322) — c'est
ce qui a permis au repli de la phase 2b d'apparier une partie des periodes.

Les secondes variantes (`catalyst_catalyst.mvar`, `chasm_chasm.mvar`) portent les MEMES volumes,
aux memes positions, avec des `instance_id` NON NULS ; les variantes `*_map.mvar` que le catalogue
retient (plus d'objectifs) ont `instance_id = 0` partout — c'est la raison du « instance_id = 0 sur
toutes les entrees » du §6 du plan. Les canevas Forge `fo11_blank` portent 5 collines de RACK
(r = 1 m, a 2 m d'ecart, equipe 0, z = 50,5) : le piege du canevas s'applique aux collines comme aux
zones, et `isParkedPalette` les ecarte deja.

VARIANTE DE MODE : non ouverte, sans objet — les collines sont dans la variante de CARTE.

### 1.3 Le nom des labels : NON RESOLU (liste testee)

Aucun radical de KOTH ne retombe sur les trois hashs. Teste (murmur3 x86_32 seed 0,
`mapvar.LabelHash`) :

- radicaux nus et composes aux suffixes `_include _exclude _zone _spawn _hill _area _volume _marker
  _navpoint _socket _objective _capture _point _1 _2 _3 _a _b _c _01 _02 _03 _include_zone
  _zone_include _neutral _neutral_include _active _inactive _default _multi _multi_exclude
  _neutral_exclude` : `koth koth_hill koth_zone koth_hill_zone koth_zone_include koth_hills hill
  hill_zone hills hill_include hill_zone_include king_of_the_hill king_of_the_hill_hill
  king_of_the_hill_zone kingofthehill king crown crown_include crown_zone zone zones capture_zone
  capture_hill control_zone control_point territory territories territory_zone total_control
  total_control_zone land_grab land_grab_zone landgrab koth_neutral koth_moving moving_hill
  rotating_hill sequence_hill hill_sequence koth_sequence koth_order hill_order hill_marker
  koth_marker crown_hill the_hill objective_hill objective_zone mode_zone` (1 585 candidats) ;
- 54 radicaux de tous les modes connus x 103 suffixes (5 612 candidats) ; 144 variantes de casse
  (`KOTH`, `Koth`, `KingOfTheHill`, `KOTH_Hill`, ...) x 57 suffixes (8 311) ;
- dictionnaire de 2 mots (200 mots : king, of, the, hill, koth, crown, zone, capture, control,
  moving, sequence, territory, throne, ...) combines avec et sans `_`, x 11 suffixes (237 936) ;
- FORCE BRUTE : toutes les chaines sur `[a-z_]` de longueur <= 7 (10,9 milliards de radicaux),
  filtrees par `radical + _include` puis confirmees par un second suffixe de role : aucune paire
  coherente (les seules doubles correspondances sont des chaines aleatoires : `csdr_ef`,
  `arfedzm`).

Consequence pour CT.2.2 : le role `hill` se pose par HASH (les deux hashs, role + filtre), avec
la preuve ci-dessus, dans une table distincte de `labelNames` — la table nommee ne recoit que des
noms qui re-hashent sur leur cle (`TestLabelTableIsSelfConsistent`).

### 1.4 Le croisement avec l'oracle (`01e1f945`, Catalyst) — trois passes, la definition avant

Instrument `hill_shapes_measure_test.go`, formes = `catalyst_map.mvar`, role -767961569 ET filtre
2133978317 (6 volumes) ; film decode par le chemin de production (`p2bBuild`), 113 vies, 5 343
frames de 100 ms, 60 rampes de jauge (3 slots), premiere rampe a 36,8 s. Definitions ecrites en
tete du fichier avant la mesure : (D1) periode = rampe brute (`zoneRampsOf`) ; (D2) forme appariee
= celle qui compte le plus de positions a <= 0,5 m ; (D3) « dedans » = >= 50 % d'occupation ;
(D4) >= 90 % des periodes ; (D5) temoins formes permutees et periodes +20 s, hasard publie.

| passe | ce qui a change | dedans / 60 | temoin permute | temoin +20 s |
|---|---|---:|---:|---:|
| 1 | 8 formes (le seul hash de role : 2 objets minigame inclus), occupation par FRAME | 42 = 70,0 % | 65,0 % | 73,3 % |
| 2 | R1 positions TENUES ; R2 filtre de mode (6 formes) | 42 = 70,0 % | 58,3 % | 76,7 % |
| 3 | R3 occupation aux EMISSIONS CROISSANTES de la jauge (± 0,5 s) | **50 = 83,3 %** | 83,3 % | 75,0 % |

- R1 s'est averee sans effet : les points sont DENSES (ecart entre deux points d'une vie : p50 1,
  p99 2 frames) — l'hypothese « une vie immobile n'emet rien » etait fausse, la correction est
  restee (elle est neutre et documentee).
- Un balayage de decalage temporel rampes/occupation (-10 s..+10 s) culmine a 0 (occupation
  moyenne 62 % contre 45 % au hasard) : pas de defaut d'horloge.
- R3 : `findZoneRamps` retient une suite NON DECROISSANTE, une « rampe » englobe donc les
  PLATEAUX (la jauge tenue quand la colline se vide) — d'ou l'occupation par frame bornee. Aux
  seuls instants ou la jauge MONTE, 44 rampes sur 60 sont occupees a 100 %.
- Les 10 rampes non « dedans » sous R3 : 4 micro-rampes (0,3 a 1,0 s, 0 vote a moins de 0,5 m
  d'aucune forme : rien ne s'y garde) et 6 rampes de 10 emissions dont AUCUNE emission n'est
  occupee alors que 30 a 72 % de leurs frames le sont — la jauge monte AVANT que quiconque
  n'entre (une animation d'apparition ou de remise a zero de la colline, a confirmer par le
  volet 1). Dans les six cas la forme appariee est bien celle ou la grappe finit par se poser :
  la geometrie n'est pas en cause, la semantique de la jauge l'est.
- Le temoin « formes permutees » ne DISCRIMINE PAS en KOTH (83,3 % = le reel) : les formes
  permutees sont posees sur d'autres collines REELLES, et D2 re-apparie ; c'est un temoin de
  taille, pas de lieu. Le temoin temporel (+20 s) tombe a 75,0 % ; le hasard (une colline
  occupee a une frame quelconque) vaut 83,2 %.
- Fait saillant : 49 rampes sur 60 portent EXACTEMENT 9 ou 10 emissions croissantes, quelle que
  soit leur duree (1 s a 45 s) — la jauge de KOTH monte par 10 pas fixes.
- PRODUCTION (`buildHillStates`, le chemin de la phase 2b) avec ces 6 formes : **21 periodes sur 6
  zones, 4 rampes non appariees, 4 975/5 343 frames actives = 93,1 %** — contre AVANT
  (Bastion+Extraction, 8 formes) : 20 periodes, 6 zones, **8** rampes non appariees.
- AVANT/APRES sur les 20 periodes publiees par l'artefact : **13 sur 20** designent une colline
  DISTINCTE de la forme de Bastion/Extraction sur laquelle elles etaient posees (a 1,3 a 24,8 m) ;
  7 sont confondues (les collines qui coincident avec une zone d'un autre role).

### 1.5 Verdict CT.2.1

Les collines vivent dans la variante de CARTE, sous le couple de hashs `[2133978317, -767961569]`
(volumes de type `-1476457415`), 6 / 5 / 5 / 5 sur les quatre cartes, formes coherentes avec les
zones des autres modes ; leur nom snake_case n'est pas retrouve. Sur `01e1f945` elles expliquent
plus de rampes que le repli actuel (4 non appariees contre 8) et 13 des 20 periodes publiees
changent de forme. La clause « >= 90 % » du gate 2 se mesure en CT.2.3 sur les 4 films et le
catalogue servi ; sur ce premier film et sous la definition ecrite avant la mesure, elle vaut
83,3 % (R3) — le negatif est ecrit tel quel, l'analyse des 10 manques est ci-dessus.

## 2. CT.2.2 — Le role `hill` : decodeur, table, catalogue, artefact, service

### 2.1 Ce qui est ecrit

| fichier | ce qu'il porte |
|---|---|
| `internal/analysis/replay/mapvar/objectives.go` | `RoleHill = "hill"` ; `LabelHashHillRole` (-767961569), `LabelHashHillInclude` (2133978317), `LabelHashHillMarker` (-1482301937, documente, sans role) ; `classify` pose `hill` quand l'objet porte LA PAIRE role + filtre et qu'aucun label NOMME ne donne de role ; les deux hashs restent comptes NON RESOLUS (aucun nom devine — `TestLabelTableIsSelfConsistent` intact) |
| `mapvar/mapvar_test.go` | `TestHillRoleParLaPaireDeHashs` (paire / role seul = minigame / filtre seul = marqueur / priorite du role nomme) ; `TestCollinesDeCliffhanger` (fixture versionnee : 5 collines, neutres, avec forme, 2 cylindres) |
| `internal/games/mappings/loader_objective_roles.go` | `hill` admis par la validation stricte |
| `config/titles/halo_infinite/mappings/objective_roles.toml` | entree `match = ["King of the Hill", "KOTH"]`, `roles = ["hill"]`, `neutral = true` — jetons releves sur les 26 libelles KOTH du registre local (« Arena:King of the Hill on X », « Ranked:King of the Hill on X », `game_variant_name` « KOTH:Arena ») ; en-tete des roles admis mis a jour |
| `loader_objective_roles_test.go` | fichier versionne : 7 modes, `hill` neutre |
| `data/titles/halo_infinite/reference/map_objectives.json` | regenere HORS LIGNE pour les 4 cartes des films (`--from-file`, `.mvar` du depot de dumps) : Catalyst 6 collines, Chasm 5, Shogun 5, **Solitude - Ranked 5 (carte NOUVELLE au catalogue, `4a5e5612`)** — 21 collines, 100 % avec forme, 100 % neutres ; les 69 autres cartes ne bougent pas (le diff ne retire que `generated_at` et 3 compteurs de couverture) |
| `cmd/mapobj-build/main.go` | `--from-file` sur une carte DEJA au catalogue conserve ses metadonnees reseau (`version_id`, `public_name`, `fetched_at`) au lieu de dater le re-parse local |
| `cmd/mapobj-build/variant_test.go` | rack de Vagabond : 25 objectifs (20 + 5 collines de rack), toujours ecarte |
| `internal/analysis/replay/objectives_catalog_test.go` | l'invariant « role surfacique = 100 % avec forme » couvre `hill` (21 collines) |
| `internal/service/replay_map_objectives_test.go` | `TestMapObjectives_KOTH_DonneesReelles` : la table versionnee reconnait les deux libelles, sert `[{hill neutre}]`, Catalyst rend 6 zones / 0 marqueur ; CTF n'en sert aucune |
| `internal/replaybuild/zones.go` | `hillFallbackRoles` et le repli SUPPRIMES ; `heldZoneRoles = {strongholds_zone, hill}` : le catalogue du match (donc le balayage ti=13) ne se paie que sur les roles de zone TENUE — plus en CTF (18 cartes a `flag_delivery` avec forme) ni en Extraction ; en-tete reecrit ; `isHillVariant` reste la porte de la METHODE (`ZoneInput.Hill`) |
| `internal/replaybuild/zones_test.go` | `TestMatchZonesModesSansZone` etendu a Catalyst (CTF dans les deux ordres, Extraction, Oddball, Slayer : 0 zone) ; `TestMatchZonesKOTHViennentDuRoleHill` (3 libelles -> 6 collines, `roles = "hill"`, meme liste que le service) ; `TestMatchZonesBastionSansCollines` (Bastion sur Catalyst : 3 zones, pas les collines) ; le test du repli supprime avec le repli |
| `internal/analysis/replay/build_zones.go`, `document_zones.go` | les deux commentaires faux de la revue ronde 2 (P2) corriges : le balayage se decide par le ROLE tenu (`heldZoneRoles`), plus par la presence de formes ; `zoneRef` se joint sur `hill` en KOTH |
| `hill_shapes_measure_test.go` | lit desormais `mapvar.RoleHill` du catalogue par defaut (mode CT.2.3) |

Le SERVICE n'a pas change de code : il projette la meme table (`objectiveRoleSpecs`) — l'entree
KOTH suffit, et le test sur donnees reelles le prouve. Le WEB n'a pas change (interdit au volet, et
sans besoin : `objectivesLayer.ts` dessine les zones par FORME, le role ne pilote que l'anneau des
livraisons ; `zoneStatesLayer.ts` teinte par `zoneRef`) — a confirmer au gate visuel utilisateur.

### 2.2 Decisions

- **Cle d'instance** : `instance_id` reste celui du fichier (0 sur toutes les variantes `*_map.mvar`
  retenues — c'est le fichier, pas le decodeur) et la cle SERVIE reste le RANG SPATIAL dans le role
  (`ZonesOfRole` trie x, y, z, puis instance_id), exactement comme les zones de Bastion : `zoneRef`
  = index dans `mapObjectives.zones`, l'artefact et le service construisent la meme liste (test de
  garde). Rien de nouveau a porter.
- **Role par HASH, pas par nom** : le nom des labels n'etant pas retrouve (§1.3), `labelNames` ne les
  recoit pas ; une paire de constantes typees + `isHill` dans le decodeur, avec la preuve en
  commentaire (motif, comptes, contre-exemple minigame, film). Le jour ou le nom tombe, il entre
  dans `labelNames`/`roleByLabel` et la paire de constantes se retire — la regle « on ne devine pas
  de libelle » est tenue.
- **`neutral = true`** : le team_index des collines vaut -1 dans les 21 cas ; le drapeau est pose par
  coherence avec Bastion/Extraction (l'etat vivant appartient a `zoneStates`).
- **Modes a zone TENUE** = `{strongholds_zone, hill}` dans `replaybuild/zones.go` (un ensemble de
  roles du decodeur, pas un nouveau champ TOML : le plan demandait de conditionner « aux ROLES de
  zone tenue »). Total Control n'a pas de role au catalogue (variante de MODE, lot 5) : rien a
  declarer tant qu'il n'existe pas.
- Les instruments des phases 2a/2b (`zone_state_p2a_*`, `zone_state_p2b_temoin_test.go`) gardent
  leur union `strongholds_zone,extraction_zone` en KOTH : ce sont les mesures de leur phase, leurs
  chiffres doivent rester reproductibles ; la mesure KOTH du lot vit dans
  `hill_shapes_measure_test.go`.
- `zone_states_hill.go` (en-tete : « le catalogue ne connait AUCUN role de colline ») est
  INTERDIT au volet (volet 1 en parallele) : trois lignes de doc a rafraichir a la fusion,
  consignees pour le superviseur (§6).

## 3. CT.2.3 — La MESURE sur les 4 films, depuis le catalogue servi

Instrument `hill_shapes_measure_test.go`, formes = catalogue (`ZonesOfRole(hill)`, l'ordre servi),
un film par processus (`ZONE_FILM`), logs `lotCter/volet2_<film>_catalogue.log`. Definitions D1-D5
et seuils : en tete du fichier, ecrits avant la premiere mesure (§1.4), jamais abaisses ; les
corrections d'instrument (R1 positions tenues — sans effet ; R2 filtre de mode ; R3 emissions
croissantes ; R4 temoin deplace) sont datees et motivees dans le meme en-tete.

### 3.1 Le tableau

| film (carte) | collines | rampes | DEDANS emissions (D4) | dedans frames | temoin permute | temoin +20 s | temoin DEPLACE (+6 m ; +6 m) | hasard reel / permute / deplace |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| `01e1f945` (Catalyst) | 6 | 60 | **50/60 = 83,3 %** | 42/60 = 70,0 % | 83,3 % | 75,0 % | **13,3 %** | 83,2 / 71,2 / 9,5 % |
| `606d9844` (Chasm) | 5 | 7 | **4/7 = 57,1 %** | 5/7 = 71,4 % | 71,4 % | 85,7 % | **42,9 %** | 81,9 / 71,1 / 12,4 % |
| `8076f97f` (Shogun) | 5 | 22 | **19/22 = 86,4 %** | 17/22 = 77,3 % | 86,4 % | 90,9 % | **22,7 %** | 85,6 / 84,6 / 18,4 % |
| `0a247154` (Solitude - Ranked) | 5 | **0** | sans objet | sans objet | — | — | — | — |
| ensemble (3 films juges) | 16 | 89 | **73/89 = 82,0 %** | 64/89 = 71,9 % | | | | |

`0a247154` : AUCUNE rampe de jauge (tag 3) sur ce film — la phase 1 l'avait deja mesure (0 rampe) —,
donc aucun denominateur : le calque vivant y est vide quelles que soient les formes ; les 5 collines
sont servies (statiques). Ecrit comme absence d'oracle, pas comme echec des formes.

Manques (D3 emissions) : Catalyst 4 micro-rampes non appariees (duree mediane 1,1 s, aucune
position a moins de 0,5 m d'aucune forme) + 5 rampes appariees a une forme occupee >= 30 % de leurs
frames mais VIDE a chaque emission croissante + 1 autre ; Chasm 1 micro-rampe + 2 « vides aux
emissions » (deux rampes de 24 et 54 s, occupees 74 % et 86 % de leurs frames) ; Shogun 3
micro-rampes. Soit 15 des 16 manques = micro-rampe sans grappe nulle part, ou jauge qui monte
AVANT que quiconque n'entre (apparition / remise a zero de la colline — a lire par le volet 1) :
la geometrie n'est en cause dans aucun des 16 cas (la forme appariee est bien celle ou la grappe
se pose ensuite).

Ce que disent les temoins : le temoin PERMUTE (celui du plan) ne discrimine PAS en KOTH — il pose
chaque forme sur une AUTRE colline reelle, ou les joueurs vont aussi, et D2 re-apparie (83,3 % =
le reel sur Catalyst) ; le temoin DEPLACE de 6 m tombe a 13 / 43 / 23 % et le hasard des formes
deplacees a 9,5-18,4 % contre 82-86 % pour les vraies : les formes sont bien LA ou l'action est.

### 3.2 La production (le chemin de la phase 2b) avec les collines

| film | AVANT (Bastion+Extraction) | APRES (collines) |
|---|---|---|
| `01e1f945` | 8 formes, 20 periodes, 6 zones, **8** rampes non appariees, 93,1 % actif | 6 formes, 21 periodes, 6 zones, **4** non appariees, 93,1 % actif |
| `606d9844` | 8 formes, 5 periodes, 4 zones, 0 non appariee | 5 formes, 5 periodes, 3 zones, 2 non appariees, 83,8 % actif |
| `8076f97f` | **3 formes, 1 periode, 1 zone**, 1 non appariee | 5 formes, **7 periodes, 4 zones**, 2 non appariees, 80,4 % actif |

Sur `01e1f945`, tableau des 20 periodes publiees (AVANT -> colline APRES) : **13 sur 20 DISTINCTES**
(la colline est a 1,3 a 24,8 m de la forme de Bastion/Extraction sur laquelle la periode etait
posee), 7 confondues (les collines qui coincident avec une zone d'un autre role) — le detail
periode par periode est dans `lotCter/volet2_01e1f945_catalogue.log`. Sur Chasm 3 sur 5, sur
Shogun 1 sur 1.

### 3.3 Verdict du gate 2

- Formes TROUVEES : oui (§1) ; SERVIES : oui (§2, table + catalogue + service + artefact).
- Clause « >= 90 % des periodes » sous la definition ecrite avant la mesure : **NON TENUE** —
  83,3 % / 57,1 % / 86,4 % (82,0 % sur 89 periodes), un film sans denominateur.
- Le negatif est ecrit tel quel. Ce qu'il mesure, d'apres les temoins et le bilan des manques :
  la clause est couplee a la SEMANTIQUE de la jauge (rampes d'activation colline vide,
  micro-rampes d'une seconde), pas a la geometrie — le temoin deplace s'effondre a 9-18 % la ou les
  vraies formes tiennent 82-86 %, et la production apparie plus de rampes qu'avant sur les trois
  films.
- Le repli d'affichage du plan (cercle synthetique au barycentre de la grappe) est une DECISION
  UTILISATEUR : il n'est pas code. Recommandation de l'executeur : servir les formes reelles (elles
  sont livrees et jointes ; le cercle synthetique serait posee au barycentre d'une grappe que
  ces formes contiennent deja).

## 4. Statut des items

- [x] CT.2.1 — inventaire, chasse, croisement `01e1f945`, verdict ecrit (§1).
- [x] CT.2.2 — role `hill` (decodeur par paire de hashs), table, catalogue 4 cartes (+ Solitude - Ranked), artefact sans repli, balayage restreint aux zones tenues, service (§2).
- [x] CT.2.3 — mesure jouee sur les 4 films depuis le catalogue (3 juges, 1 sans denominateur), tableau AVANT/APRES de `01e1f945` (13/20 distinctes) ; clause >= 90 % NON TENUE (82,0 % sur 89 periodes), negatif ecrit, repli = decision utilisateur (§3).

## 5. Cout machine

- Films decodes (un par processus, avant-plan, ~180-210 s chacun) : `01e1f945` x 5 (3 passes
  candidat, 2 catalogue), `606d9844` x 2, `8076f97f` x 2, `0a247154` x 1 — 10 decodages,
  ~35 min ; force brute murmur3 N <= 7 : ~24 min CPU un
  coeur (N <= 6 : 52 s) ; gates : ~15 min.

## 6. Decouvertes hors perimetre (non traitees) et consignes pour la fusion

- **Artefacts KOTH cuits AVANT ce lot** (`01e1f945.json` schema 16 en cache prod, `606d9844`
  schema 6) : leur `zoneStates[].zoneRef` indexe l'ancienne liste (Bastion+Extraction, 8 zones sur
  Catalyst) alors que le service sert desormais 6 collines. Le web se TAIT (`useZoneStates.ts`
  compare `coverage.zones.catalog` au nombre de zones servies : 8 != 6 => `joinable = false`) —
  degradation propre, mais le calque vivant KOTH ne se verra qu'apres RECUISSON des artefacts
  KOTH (`replay-build`). Une carte ou l'ancien compte egalerait le nombre de collines teinterait
  la mauvaise zone : le web pourrait comparer `coverage.zones.roles` (publie pour cela) en plus
  du compte — a evaluer par le lot web.
- `zone_states_hill.go` (INTERDIT au volet, volet 1 en parallele) : l'en-tete dit « le catalogue de
  formes ne connait AUCUN role de colline » et « appariement contre les zones que la carte declare
  sous d'autres roles » — trois lignes a rafraichir a la fusion (le catalogue porte `hill`, la
  grappe s'apparie aux collines).
- Instruments des phases 2a/2b (`zone_state_p2a_corpus_test.go` `p2aRolesZones`,
  `zone_state_p2b_temoin_test.go` `p2bRoles`) : gardent l'union Bastion+Extraction en KOTH (leurs
  chiffres historiques restent reproductibles) ; la mesure KOTH du lot vit dans
  `hill_shapes_measure_test.go`. A l'occasion d'un temoin recuit KOTH (volet 3, CT.3.3), lire
  `RoleHill`.
- Labels craques au passage, aucun n'est un role : `landgrab_include` = -886053664 et
  `landgrab_zone` = 996801386 (9 zones sur Catalyst et Chasm) ; `strongholds_exclude` =
  -1191131830 ; `doubles_include` = 390628080 / `doubles_exclude` = 1081676545 (Solitude - Ranked —
  le filtre du mode Doubles). A ajouter a `labelNames` dans un lot de couverture (le lot 5 en
  avait deja 7 en attente).
- Marqueurs ponctuels `-877512201` `[<mode>_include, <hash de marqueur>]` : Bastion -1246645531,
  Land Grab -941529218, KOTH -1482301937 — un par zone, a sa position (repere HUD/nav ?). Non
  exploites, non nommes.
- La jauge KOTH (tag 3) monte par 9-10 pas fixes quelle que soit la duree de la rampe (49 rampes
  sur 60 sur `01e1f945`) ; 5 + 2 rampes montent colline VIDE puis la colline se remplit (apparition
  / remise a zero ?), 8 micro-rampes d'une seconde n'ont aucune grappe : matiere pour le volet 1
  (semantique d'activation).
- `0a247154` (Ranked KOTH, Solitude - Ranked) : aucune rampe de jauge tag 3 dans le film (deja vu
  en phase 1) — le variant Ranked emet-il la jauge sur un autre tag ? (volet 1.)
- Les variantes `*_map.mvar` retenues par le catalogue ont `instance_id = 0` partout ; les
  variantes de mode (`catalyst.mvar`, `chasm.mvar`, `ctf_breaker.mvar`) portent des identifiants
  non nuls — si un jour la cle d'instance devient necessaire, elle est dans l'autre fichier.
- Le premier commit du volet (`3d6fe97e7`) a ete pose hooks desactives par erreur d'outillage ;
  gitleaks (0 fuite), gofmt et le controle de marqueurs de conflit ont ete rejoues a la main sur
  ce commit, les commits suivants passent les hooks.
- Plan et registre : les cases CT.2.1/CT.2.2/CT.2.3 du plan et l'entree `thought_log` sont a
  poser par le superviseur (regle du lot) — §4 ci-dessus est la source.
- **DETTE PREEXISTANTE, consignee et NON traitee (revue ronde 1, 2026-08-19)** :
  `cmd/mapobj-build/main.go:274` `ingestLocal` (le chemin `--from-file`) n'applique PAS
  `isParkedPalette` — seul `ingestRemote` le fait (`main.go:221`). Le piege du rack Forge est
  donc ouvert sur le chemin hors ligne : un `.mvar` de canevas ingere a la main entre au
  catalogue avec ses objectifs ranges hors terrain. La demonstration est faite au passage de
  R1-3 : sonde dans un faux depot, `empyrean_fo11_blank.mvar` (le canevas, 100 objets) rend
  **5 collines de rack** la ou la carte jouee n'en a qu'une. Le lot n'a emprunte ce chemin que
  sur des `*_map.mvar` surs (les 4 cartes des films, puis les 23 de R1-3, tous choisis parce
  que le catalogue les avait DEJA retenus apres `isParkedPalette` a l'ingestion reseau).
  **Condition de reprise** : le jour ou `--from-file` est offert a un fichier non deja retenu
  (nouvelle carte, dump manuel), poser `isParkedPalette` dans `ingestLocal` — ou un drapeau
  explicite `--accept-parked` — AVANT de s'en servir.

## 7. Revue adversariale ronde 1 — constats et corrections

Ronde 1 jouee le 2026-08-19 sur le diff `638c4d044..2951d7f55`. Cinq constats, un commit par
correction, gates rejoues en entier (section « Ronde 1 » de `LOTCTER_volet2_gates.log` : tous
les EXIT_* a 0).

### R1-1 (P1) — les gardes centrales du lot ne tournaient PAS en CI

**Constat.** `zonesTestBuilder` (`internal/replaybuild/zones_test.go`) et les deux tests sur
donnees reelles du service localisaient le depot par `title.FindRepoRoot()` : marqueur
`db_profiles.json`, GITIGNORE donc absent d'un checkout CI, ou `LEVELUP_REPO_ROOT`, jamais
pose en CI. Les gardes du lot sortaient en SKIP silencieux — « KOTH = 6 collines, meme liste
que le service », « CTF/Extraction/Oddball/Slayer = 0 zone sur Catalyst », « Bastion = 3
zones », « specs reels = [{hill neutre}] ». Le worktree du lot n'a pas `db_profiles.json` non
plus : elles n'ont tourne qu'avec la variable posee a la main — c'est ce que dit la premiere
section du log de gates, « LEVELUP_REPO_ROOT=worktree ».

**Correction.** Mecanisme CANONIQUE unique : `internal/testutil/repo_root.go`,
`testutil.RepoRoot()` deduit la racine de l'EMPLACEMENT DE SON PROPRE FICHIER SOURCE
(`runtime.Caller`) — l'arbre versionne, donc trouve quel que soit le repertoire courant, sans
variable ni fichier ignore — puis verifie un repertoire versionne. Les cinq copies maison sont
migrees, et un fichier VERSIONNE absent echoue desormais au lieu de skipper :

| fichier | avant | apres |
|---|---|---|
| `internal/replaybuild/zones_test.go:153` | `FindRepoRoot` + 2 `t.Skipf` | `testutil.RepoRoot()` + `t.Fatalf` |
| `internal/service/replay_map_objectives_test.go:177` et `:224` | `FindRepoRoot` + `t.Skipf` | idem |
| `internal/analysis/replay/objectives_catalog_test.go:172` | echelle `../../../..` + `t.Skip` | idem |
| `internal/analysis/replay/callouts_catalog_test.go:65` | echelle + `t.Skip` | idem |
| `internal/games/mappings/loader_objective_roles_test.go:135` | `reposRootDepuisTests` (remontee cwd) | idem, helper local supprime |
| `internal/games/mappings/loader_awards_test.go:143` | chemin relatif en dur + `t.Skipf` | idem |

**Garde-rail** (regle CLAUDE.md n.6 : centraliser ET interdire l'ancien litteral) :
`internal/archlint/no_repo_root_walk_test.go:111` `TestNoAdHocRepoRootLadderInTests` interdit
l'echelle de remontee ecrite a la main dans un `_test.go`. Allowlist DATEE de deux fichiers
ANTERIEURS et hors du perimetre de gate du lot :
`internal/analysis/filmdec/map_bounds_test.go` (echelle + Skip sur `map_quant_bounds.json`,
versionne) et `internal/ops/seed_citation_assets_test.go` (const `citationRepoRoot`).
Condition de reprise ecrite dans le fichier : au prochain passage sur ces paquets, migrer et
retirer l'entree.

**Preuve (phrase corrigee en ronde 2).** `go test -v ./internal/replaybuild/
./internal/service/` SANS `LEVELUP_REPO_ROOT`, **restreint aux tests migres du tableau
ci-dessus** : 12 tests, 0 SKIP, tous verts. La redaction d'origine — « 12 tests, **0 SKIP** »
— ne disait pas de QUOI elle parlait et se lisait comme un verdict de paquet : au niveau des
PAQUETS la meme commande rendait alors 1302 PASS / **14 SKIP**, dont 5 encore du motif racine,
dans trois fichiers que la ronde 1 n'avait pas migres. Le compte reel avant/apres est au §8.
Controle negatif du garde-rail (allowlist videe) : FAIL sur les deux fichiers attendus, et sur
eux seuls.

### R1-2 (P1) — `hill_shapes_measure_test.go` a 615 lignes (seuil 500)

Scission par RESPONSABILITE : le fichier de mesure (449 L) garde l'instrument entier
(definitions et seuils ecrits avant la mesure, appariement, temoins, `TestHillShapesMeasure`) ;
`hill_shapes_report_test.go` (190 L) recoit les fonctions qui IMPRIMENT un rapport a partir
d'une mesure deja faite : `hillTableau`, `hillMark`, `hillProduction`, `hillAvantApres`,
`p2aZonesOrNil`, `hillShapeDesc`, `hillManques`.

Deplacement PUR, verifie octet pour octet : le bloc deplace est identique aux lignes 447-615
de l'ancien fichier, la tete conservee identique aux lignes 1-445 — a 4 lignes de renvoi pres
ajoutees a l'en-tete (ou sont passes les rapports). Un seul `func Test` dans le paquet avant
comme apres : les chiffres de `lotCter/volet2_*.log` restent reproductibles a l'identique.

### R1-3 (P2) — les 19 autres cartes KOTH du registre entrent au catalogue

**Inventaire** (lecture seule, `duckdb -readonly` sur `shared_matches_v2.duckdb` du principal) :
34 `map_id` distincts joues en King of the Hill / KOTH ; 27 sont au catalogue, 7 n'y sont pas
(Chasm `fc1ced39`, Ecotone, Vallaheim Firefight, Argyle, Cliffhanger, Live Fire `b6aca0c7`,
Recharge) ; 23 des 27 n'avaient aucune colline.

**Regeneration** par le MEME chemin que les 4 cartes du lot : `--from-file` sur le `.mvar`
depose (`.ai/re_dump/mapvar/` du principal, lecture seule), une carte a la fois, catalogue du
worktree (`LEVELUP_REPO_ROOT` explicite). AUCUN appel reseau : les 23 fichiers etaient deposes.
Vagabond a exige une copie de travail nommee `map.mvar` (son nom de fichier est partage avec
Highpower ; le depot le range sous `vagabond_map.mvar`) — la copie garde `mvar_file`/`module` a
l'identique. Le compte d'objets parses coincide avec l'`objects_n` deja au catalogue sur les 23
cartes : chaque fichier est bien celui de sa carte.

**Garde-fou, verifie sur le diff** : 20 lignes supprimees en tout — `generated_at` et 19
compteurs de couverture —, ZERO autre suppression ; 92 lignes `"role": "hill"` ajoutees et 92
`unresolved_labels` qui les accompagnent, rien d'autre. 54 entrees identiques octet pour octet
(les 50 cartes non touchees + les 4 sans colline). Aucune carte ajoutee ni retiree (73 avant,
73 apres).

| collines | cartes |
|---:|---|
| 6 | Catalyst, Prism, Salvation |
| 5 | Absolution, Curfew, Goliath, Live Fire `6c01f693`, Shogun, Snowbound, Vagabond, Banished Narrows, Bazaar, Behemoth, Chasm `a455572d`, Elevation, Fortress, Isolation, Nemesis, Opulence, Solitude - Ranked, Streets |
| 4 | Dredge |
| 1 | **Empyrean** |
| 0 | **Forbidden, Illusion, Oasis, Oasis Firefight** |

Total : **113 collines sur 23 cartes**, 100 % avec forme, aucune forme degeneree (invariant
`TestCatalogueLivreEstExploitable`, dont le commentaire est mis a jour).

**Verdict sur les cartes SANS colline.** Quatre cartes KOTH du registre n'ont AUCUNE colline
dans leur variante, et Empyrean n'en a qu'une. Ce n'est pas un `.mvar` perime : le fichier gele
(2026-07-26 / 08-13) est POSTERIEUR au dernier match KOTH de ces cartes (2026-01-18 au plus
tard). Ce n'est pas non plus la variante de MODE : sondees a part, dans un faux depot et sans
toucher au catalogue, `forbidden_ctf_forbidden`, `illusion_ctf_illusion`, `oasis_btb_exiled` et
`oasis_firefight_btb_exiled` rendent **0 colline** — avec temoin POSITIF `catalyst_catalyst.mvar`
= 6 collines, qui valide la methode. La colline de ces matchs n'est donc pas dans les fichiers
dont on dispose. **Condition de reprise** : re-tirer l'asset UGC de ces cinq cartes
(`--player <GT> --map-id <uuid>` — une version plus recente peut exister) ; si elle rend encore
0, la colline de ces variantes vient d'ailleurs que du motif `[2133978317, -767961569]` et
l'inventaire de CT.2.1 est a rouvrir sur elles.

### R1-4 (L6) — la preservation des metadonnees reseau par `--from-file` n'avait aucun test

`ingestLocal` (`cmd/mapobj-build/main.go:300`) conserve `version_id`, `public_name` et
`fetched_at` d'une carte deja au catalogue. Sans ce bloc, un catalogue gele serait date du jour
de sa RELECTURE et perdrait le nom public et la version de l'asset, sans un mot. La meme regle
cote `--refresh-from` etait deja gardee ; celle-ci ne l'etait pas — alors que R1-3 vient de
l'emprunter 23 fois.

`TestIngestLocalPreserveLesMetadonneesReseau` (`cmd/mapobj-build/refresh_test.go:249`) : carte
deja au catalogue = les trois metadonnees survivent, `mvar_file`/`module` suivent le fichier
parse ; carte neuve = rien d'herite et date du parse. Temoin = le `.mvar` VERSIONNE de
Cliffhanger, localise par `testutil.RepoRoot()`. **Controle negatif joue** : bloc retire de
`main.go` -> FAIL sur les trois champs ; bloc remis -> vert.

### R1-5 (doc inversee) — `internal/service/replay_map_objectives.go:53`

Le commentaire disait « mode sans objectifs statiques (Slayer, KOTH...) » alors que la table du
titre sert `hill` en King of the Hill depuis le lot : il decrivait l'etat d'AVANT le fichier de
config qu'il commente. Exemples remplaces par trois modes reellement absents de la table :
Slayer, Land Grab, Total Control.

### Ce qui n'a PAS ete fait (et pourquoi)

- La dette `ingestLocal` / `isParkedPalette` est CONSIGNEE, pas corrigee (§6, entree du
  2026-08-19) : la revue la classe hors correction.
- `internal/analysis/filmdec/map_bounds_test.go` et `internal/ops/seed_citation_assets_test.go`
  gardent leur echelle de remontee : hors perimetre de gate du lot, allowlist datee avec
  condition de reprise (R1-1).
- Le web, `zone_states_hill.go`/`zone_state_scan.go` (volet 1) et `document.go`/schema
  (volet 3) n'ont pas ete touches : interdits au perimetre.

## 8. Revue adversariale ronde 2 — le garde-rail n'attrapait pas son propre motif

Ronde 2 jouee le 2026-08-19 sur le resultat de la ronde 1 (HEAD `397d341c3`). Un constat P1 et
son P2 associe, un commit par correction, gates rejoues (section « Ronde 2 » de
`LOTCTER_volet2_gates.log`).

### Le constat

Le garde-rail pose en R1-1 (`TestNoAdHocRepoRootLadderInTests`) ne cherchait QUE le litteral de
l'echelle `../../../..`. Il n'attrapait donc NI `title.FindRepoRoot()` + `t.Skip` — le motif
EXACT qui a cause R1-1 — NI la lecture directe de `LEVELUP_REPO_ROOT`. Une garde qui n'attrape
pas le defaut qu'elle documente ne garde rien.

Mesure : `go test -v ./internal/replaybuild/ ./internal/service/` sans variable rendait
**14 SKIP** au niveau des PAQUETS, dont **5 du motif racine**, dans trois fichiers que la ronde
1 n'avait pas migres — deux d'entre eux portant, AU-DESSUS du skip, un commentaire affirmant
« il ne se declare pas SKIP en silence ». Les catalogues versionnes `map_callouts.json` et
`map_quant_bounds.json` n'avaient donc, en fait, aucune couverture CI. P2 associe : la phrase
de preuve du §7 (« 12 tests, 0 SKIP ») etait vraie des seuls fichiers migres — elle est
corrigee ci-dessus.

### R2-1 — les cinq derniers appels migres (commit `8fe1d1b5c`)

Verification prealable exigee par la revue, `git ls-files` (2026-08-19) : TOUT ce que ces tests
lisent est VERSIONNE — `map_quant_bounds.json`, `map_callouts.json`, les 112 entrees de
`map_backgrounds/`, `weapon_names.toml` et `replay_labels.toml` (charges par `NewBuilder`).
Aucun skip n'est donc conserve : `t.Skipf` -> `t.Fatalf` sur les cinq.

| test | fichier:ligne (avant) |
|---|---|
| `TestResolveMapEntry_SurLeCatalogueLivre` | `internal/replaybuild/replaybuild_test.go:45` |
| `TestMapBackground_DonneesReelles` | `internal/service/replay_map_background_test.go:308` |
| `TestMapBackground_TousLesModulesDuCatalogue` | `internal/service/replay_map_background_test.go:358` |
| `TestMapBackground_TousLesFondsMapID` | `internal/service/replay_map_background_test.go:408` |
| `TestMapCallouts_DonneesReelles` | `internal/service/replay_map_callouts_test.go:122` |

**Le compte AU NIVEAU DES PAQUETS**, `go test -count=1 -v ./internal/replaybuild/
./internal/service/` sans `LEVELUP_REPO_ROOT` :

| | PASS (racine) | SKIP (racine) | PASS (tous niveaux) | SKIP (tous niveaux) |
|---|---|---|---|---|
| avant R2-1 | 1302 | **14** | 1446 | 14 |
| apres R2-1 | 1307 | **9** | 1507 | 68 |

Zero SKIP du motif racine apres. Les 9 restants sont d'une AUTRE nature, tous legitimes :
3 dans `openspartan_import_realdb_manual_test.go` (`OPENSPARTAN_DB_PATH` non pose), 3 dans
`home_service_test.go` (TODO P4.4, cache canonical-aware), 2 dans `grenade_join_corpus_test.go`
(`REPLAY_CORPUS`, corpus d'artefacts NON versionne), 1 dans `grenade_ambigu_sweep_test.go`
(`FILM_SWEEP`). Les 59 SKIP de sous-tests qui apparaissent sont le cas normal documente « pas
de fond fige pour X » de `TestMapBackground_TousLesModulesDuCatalogue`, que son parent borne
par `avecFond == 0` : ils etaient invisibles avant parce que le parent lui-meme skippait.

### R2-2 — le garde-rail etendu (commit `922b0d424`)

`TestNoProdRepoRootHelperInTests` (`internal/archlint/no_repo_root_walk_test.go:206`) interdit
desormais, hors commentaires, dans tout `_test.go` sous `internal/` et `cmd/` :

- `title.FindRepoRoot` — l'APPEL. L'import de `title` reste libre : le paquet sert aussi
  `DefaultSlug` et `PathResolver`, l'interdire ferait tomber des dizaines de tests legitimes,
  et un test ne peut pas appeler le helper sans ecrire cet appel.
- `os.Getenv("LEVELUP_REPO_ROOT")`. Le motif est ECRIT AVEC son `os.Getenv` : `t.Setenv` dans
  `internal/config/config_extra_test.go` (qui teste le paquet config) et l'injection
  d'environnement d'un sous-process dans `internal/ops/seed_demo_cli_test.go` ne sont pas
  visees — elles ne situent aucune racine.

Le motif (c) — boucle `filepath.Dir` ad hoc — n'est PAS implemente : (a) et (b) couvrent les
cinq tests d'avant migration, et plus aucun helper de ce genre ne subsiste dans les tests
(`reposRootDepuisTests` a disparu en R1-1). Ne pas sur-ingenier.

Le fichier de garde s'exclut par SON CHEMIN (`runtime.Caller`), pas par une entree d'allowlist :
il CITE les motifs dans ses messages d'erreur et sa table sans en appeler aucun — meme
intention que le motif construit de `TestNoAdHocRepoRootLadderInTests`, exprimee une seule fois.

Allowlist DATEE (2026-08-19), quatre entrees, chacune avec sa condition de reprise ecrite dans
le fichier :

| fichier | raison |
|---|---|
| `internal/analysis/replay/ctf_research_test.go` | instrument de recherche sous garde `CTF_RESEARCH_FILMS` : ne tourne jamais en CI |
| `internal/himap/carte_oracle_gamefiles_test.go` | lit `data/cache/replays/...`, un CACHE **non versionne** : ici le skip est le comportement JUSTE |
| `internal/mapdecoupe/oracle_corpus_test.go` | meme defaut que R2-1 sur un dump VERSIONNE, mais hors perimetre du volet 2 |
| `cmd/mapcallouts-build/classify_test.go` | idem |

Les deux entrees de l'echelle `../` posees en R1-1 restent inchangees.

### Controles negatifs joues

1. **Allowlist videe** -> FAIL sur les 4 fichiers attendus ET EUX SEULS
   (`cmd/mapcallouts-build/classify_test.go`, `internal/analysis/replay/ctf_research_test.go`,
   `internal/himap/carte_oracle_gamefiles_test.go`, `internal/mapdecoupe/oracle_corpus_test.go`),
   chacun avec le motif qui l'a fait tomber. Le fichier de garde ne s'y signale pas : il
   s'exclut par son chemin.
2. **Fichier de test mute** : `internal/replaybuild/replaybuild_test.go` remis en
   `title.FindRepoRoot()` — mutation COMPILABLE, verifiee par `go vet` CGO=1 exit 0, donc une
   vraie regression et pas une simple mutation de texte -> FAIL sur ce seul fichier.

### Ce qui reste ouvert (ronde 2)

- `internal/mapdecoupe/oracle_corpus_test.go` et `cmd/mapcallouts-build/classify_test.go`
  portent le MEME defaut que R2-1 sur un dump VERSIONNE
  (`.ai/V7.5/dumps/callout_zones_ridgeline_clipped.json`, `git ls-files` non vide) : ils
  skippent en silence en CI. Hors perimetre du volet 2 — les migrer sans rejouer leurs paquets
  serait un fix aveugle. Reprise au prochain passage sur ces deux paquets ; l'allowlist datee
  porte la condition.
