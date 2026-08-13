# Rapport — fonds de carte par map_id (2026-08-13)

> Execution du plan `PLAN_FONDS_PAR_MAP_ID.md` (phases A/B/C). Cle d'un fond FORGE = le
> `map_id` (asset UGC de `match_registry`) ; les cartes NATIVES restent keyees par module.
> Chaine par carte : preuve level_id (sonde `TestPreuveLevelIDCartes`) -> bornes
> (`mapquant-build`) -> ancres (`mapobj-build --from-file` sur `<carte>_map.mvar`) ->
> declaration `CartesForge` -> cuisson (`mapfond-build --maps <map_id>`) -> oracle des
> ancres. Style `jeu`, 0,0920 m/px, voie de reference anti-toits active.

## Bilan global

- **37 fonds Forge publies sous cle map_id** : 2 migres (Vagabond, Corpo), 2 pilotes
  (Starboard, Dredge), 33 de masse. **0 echec de cuisson.**
- Oracle des ancres : **462/465 (99,4 %)** — 3 cartes a 1 ancre sans sol pres (The Pit
  14/15, Absolution 16/17, Goliath 7/8), toutes les autres a 100 %.
- Determinisme verifie au bit sur les pilotes (2 cuissons -> sha256 identiques) ;
  Vagabond re-cuit IDENTIQUE AU BIT a l'ancien `fo08_wetland.png`.
- Garde-rails : `TestFondForgeJamaisSousCleModule` (aucun fond sous cle canevas,
  declaration <-> fond publie), `TestCartesForgeDeclarations` (map_id unique, `.mvar` de
  carte et non de rack), oracle service `TestMapBackground_TousLesFondsMapID` (37 fonds
  servis par la cle map_id).

## Migration (phase A)

| carte | map_id | ancres | resultat |
|---|---|---|---|
| Vagabond | 105f5d84-8de1-4908-af3a-1c4f3bf9d642 | 4/4 | PNG identique au bit, calage identique |
| Corpo | 8be179f7-8940-4868-b881-44cad1ca8711 | 4/4 | calage identique ; PNG mis a niveau (pre-saut bloc/scen/mach : objets 1725 -> 1976, sansModele 260 -> 9) |

## Pilotes (phase B)

| carte | map_id | matchs | canevas | objets rendus | ancres | endpoint |
|---|---|---|---|---|---|---|
| Starboard | 7a9265af-a880-487b-8829-68d88fcfb145 | 23 | fo03_space | 3902/3964 (98,4 %) | 8/8 | 200 (match 1af26997) |
| Dredge | e4bb06db-065f-4902-b93b-d8dac315eac4 | 16 | fo06_deepsea | 5410/5479 (98,7 %) | 8/8 | 200 (match 113195e6) |

## Masse (phase C) — 33 cartes, ordre du plan

`matchs` = compte `match_registry` du map_id majoritaire au 2026-08-13. `couverte` = la
voie de reference anti-toits a substitue l'etage de jeu (>1/3 de matiere couvrante).

| carte | map_id | matchs | canevas | objets rendus | ancres | couverte | statut |
|---|---|---|---|---|---|---|---|
| The Pit | 648ae7aa | 21 | fo09_academy | 5326/5384 | 14/15 | non (24,8 %) | cuite — 1 ancre sans sol |
| Snowbound | 410f1c01 | 23 | fo13_frost | 4444/4607 | 30/30 | oui (61,1 %) | cuite |
| Empyrean | d035fc3e | 27 | fo11_blank | 5255/5297 | 9/9 | non (26,9 %) | cuite |
| Origin | b302eb62 | 23 | fo08_wetland | 5201/5288 | 13/13 | oui (36,4 %) | cuite |
| Absolution | 78da545f | 21 | fo09_academy | 5371/5494 | 16/17 | oui (80,5 %) | cuite — 1 ancre sans sol |
| Curfew | 63d634be | 19 | fo11_blank | 4539/4564 | 13/13 | oui (63,9 %) | cuite |
| Dynasty | cfd90b63 | 19 | fo08_wetland | 5506/5540 | 8/8 | oui (73,3 %) | cuite |
| Shiro | 2890782c | 17 | fo05_desert | 5466/5513 | 7/7 | oui (43,9 %) | cuite |
| Cliffside | 4bffd021 | 18 | fo05_desert | 5274/5384 | 9/9 | oui (39,1 %) | cuite |
| Nemesis | 2be34415 | 18 | fo08_wetland | 5120/5180 | 21/21 | oui (50,2 %) | cuite |
| Domicile | 921aebb1 | 17 | fo05_desert | 5478/5539 | 8/8 | non (16,2 %) | cuite |
| Fortress | 0d1c9255 | 17 | fo09_academy | 5399/5466 | 24/24 | non (27,4 %) | cuite |
| Goliath | 504ebf22 | 17 | fo11_blank | 5473/5495 | 7/8 | oui (53,7 %) | cuite — 1 ancre sans sol |
| Isolation | 01af558d | 15 | fo08_wetland | 4945/5042 | 15/15 | oui (37,3 %) | cuite |
| Solitude | f1cc3b4e | 12 | fo11_blank | 3153/3223 | 14/14 | oui (63,4 %) | cuite |
| Houseki | cf034ec8 | 15 | fo09_academy | 5206/5299 | 8/8 | oui (36,5 %) | cuite |
| High Ground | bb7b78ae | 15 | fo08_wetland | 5260/5344 | 15/15 | oui (44,3 %) | cuite |
| Salvation | cd08bc7a | 12 | fo11_blank | 4805/4873 | 6/6 | oui (38,2 %) | cuite |
| Takamanohara | edcd4467 | 15 | fo11_blank | 5499/5542 | 8/8 | oui (41,1 %) | cuite |
| Elevation | 76043dc6 | 12 | fo11_blank | 5186/5277 | 15/15 | oui (58,8 %) | cuite |
| Kiken'na | df7dbf08 | 11 | fo08_wetland | 5505/5570 | 8/8 | oui (49,6 %) | cuite |
| Banished Narrows | 9ad226d8 | 12 | fo05_desert | 4807/4897 | 8/8 | non (30,9 %) | cuite |
| Kaiketsu | 98a83f87 | 12 | fo05_desert | 5404/5497 | 8/8 | oui (35,8 %) | cuite |
| Obituary | a289bafe | 9 | fo09_academy | 5291/5397 | 16/16 | oui (54,8 %) | cuite |
| Opulence | 255bbe78 | 12 | fo11_blank | 3564/3613 | 8/8 | oui (46,2 %) | cuite |
| Command | 2c9f3490 | 11 | fo09_academy | 5368/5421 | 25/25 | oui (40,3 %) | cuite |
| Fortitude | 1ede38fa | 7 | fo05_desert | 5331/5418 | 23/23 | oui (60,2 %) | cuite |
| Refuge | 41217472 | 10 | fo08_wetland | 5273/5356 | 26/26 | oui (36,9 %) | cuite |
| Critical Dewpoint | bae4df14 | 9 | fo11_blank | 5098/5133 | 6/6 | oui (42,2 %) | cuite |
| Perilous | c5ac9f12 | 9 | fo08_wetland | 4881/4959 | 4/4 | oui (37,4 %) | cuite |
| Shogun | 33075df7 | 9 | fo11_blank | 5351/5397 | 8/8 | oui (59,5 %) | cuite |
| Sylvanus | 95b69e4b | 10 | fo05_desert | 5300/5394 | 11/11 | oui (47,5 %) | cuite |
| Smallhalla | 98783453 | 9 | fo08_wetland | 3751/3901 | 27/27 | oui (73,1 %) | cuite |

Preuves level_id (un level_id par CANEVAS, unicite 1/1 par carte via son fichier-lien) :
fo05_desert 1804860316 · fo08_wetland 88891201 · fo09_academy 1437677928 ·
fo11_blank 426470249 · fo13_frost -992358985 · fo03_space -747133697 ·
fo06_deepsea 2123870979.

## Reliquat (non traite, commande de reprise)

Cartes Forge jouees, `.mvar` au depot, <9 matchs — la MEME chaine s'applique telle
quelle : etendre `preuvesLevelID` + `mapModule` (+ sonde), `mapobj-build --from-file
.ai/re_dump/mapvar/<carte>_map.mvar --map-id <uuid>`, declaration `CartesForge`,
`mapfond-build --natives=false --maps <uuid>` :

Ecotone (8, 8816f240..., fo11_blank) · Insolence (7, d5c5eb4f..., fo09_academy) ·
Solution (7, ee43d273..., fo05_desert) · Flood Gulch (6, 7097bc4f..., fo05_desert) ·
Threshold (5, ddbb3a00..., fo11_blank) · Solitude - Ranked (5, 4a5e5612..., fo11_blank) ·
Thunderhead (4, 28a3ac28..., fo08_wetland) · Pharaoh (3, 88d45250..., fo11_blank) ·
Credence (2, 0cc728d2..., fo11_blank) · Disciple (2, 525451ca..., fo11_blank) ·
Merchant's Square (2, 7dfec55d..., fo09_academy) · Urban Raid (2, be848f91...,
fo09_academy) · <=1 match : Nadair, Outlook, Rat's Nest, Ronin, Scarlett's Landing,
Warehouse, Dawnbreaker, Cole Protocol, Lattice - Ranked (+ 944396dd sans nom).

Hors perimetre (deja au registre ou au plan) : variantes Heavies / Sentry Defense /
Firefight ; assets SECONDAIRES d'une carte deja cuite (Dynasty 90cd321d, Houseki
6439625e, Shiro 2962c4e0, Shogun 8f51ccb9, Salvation f633db01, Starboard 50771a22 —
1 a 3 matchs chacun : leur fond retombe sur l'absence propre, jamais sur la carte d'un
autre) ; `TFF | Night Of The Undead` (canevas inconnu) ; Live Fire (natif sans sbsp) ;
Detachment/Argyle (canevas inconnu). Les map_id secondaires des cartes NATIVES
(Aquarius 33c0766c, Cliffhanger 81274d6f, Recharge 8420410b, ...) sont deja servis par
le repli nom -> module.

## Artefacts du gate visuel utilisateur

- `Desktop/gate_cartes_v75/mapid_pilotes/{starboard,dredge}.png`
- `Desktop/gate_cartes_v75/mapid_masse/*.png` (33 fichiers, nommes par carte)

Le gate visuel juge l'ASPECT — l'oracle des ancres ne le fait pas (lecon Catalyst).
