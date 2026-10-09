# Plan — Fonds de carte par map_id : debloquer les cartes Forge communautaires

> Decision utilisateur du 2026-08-13 : perimetre v7.5 (« il nous en manque plein »).
> Execution sous le contrat du skill `plan-execution` (ordre strict, une etape a la fois,
> statuts [x]/[~]/[!], zero fix hors perimetre — les decouvertes se notent en §Decouvertes).
> Branche : `feat/v75`, worktree principal. Commits par phase, PAS de push (orchestrateur).
> JAMAIS deux commandes Go en parallele. Tests himap par `-run` ancre + `-timeout 45m`.

## Probleme (mesure, 2026-08-13)

La cle d'un fond de carte est le DOSSIER INSTALLE du jeu. Une carte Forge communautaire vit
sur un CANEVAS (8 installes) partage par des dizaines de cartes : une seule carte par canevas
peut donc avoir un fond (`fo08_wetland` = Vagabond, `fo11_blank` = Corpo — collision
documentee `cuisson_forge.go:63-66`). ~35 cartes Forge jouees restent sans fond. Les BORNES
de dequantification, elles, sont PAR CANEVAS et se resolvent par nom (`map_quant_bounds.json`,
plusieurs noms -> meme module, pas de collision) : seul le FOND collisionne.

## Architecture cible

- Fond d'une carte FORGE : fichiers `{map_id}.png` / `{map_id}.json` (map_id = asset UGC de
  `match_registry`, present sur chaque match). Fond d'une carte NATIVE : inchange (dossier).
- `CartesForge` (`cuisson_forge.go:83-96`) : declarations PAR CARTE — {map_id, canevas,
  fichier .mvar du depot, nom}. Plus jamais keye par canevas.
- Service `replay_map_background.go` : essaie d'abord la cle map_id du match, puis la cle
  module (natives). `PathResolver` : chemins par les helpers existants, jamais de join manuel.
- Vagabond et Corpo MIGRENT vers la cle map_id (re-cuisson deterministe) — pas de doublon
  module-keye laisse derriere (regle 0 code mort ; le fallback module ne sert que les natives).
- Bornes : entree `map_quant_bounds.json` par NOM de carte -> module canevas, prouvee par
  level_id (methode du lot bornes, sonde `TestPreuveLevelIDCartes` a etendre).
- Ancres : entree `map_objectives.json` par carte (map_id UGC + ancres + objects_n) — la
  cuisson Forge l'exige (jointure anti-mauvaise-carte, `cmd/mapfond-build/cuisson.go:195-222`).
  Outil existant : `cmd/mapobj-build` (verifier ses modes sur pieces avant usage).

## Phase A — Plomberie de la cle (fermee)

- [x] A1. `CartesForge` par carte (map_id, canevas, mvar, nom) ; `CuitCarteForge` sort
      `{map_id}.png/json` ; adapter `cmd/mapfond-build` (selection `--maps` accepte map_id
      OU dossier natif). Fait : `CarteForge{MapID, Nom, FichierMvar, ModuleCanevas}`,
      `EstCarteForge` -> `EstCanevasForge` (la chaine native ecarte le CANEVAS).
- [x] A2. Service : resolution map_id d'abord (match -> map_id -> fond), repli module pour
      les natives. Aucun changement de contrat HTTP ni front. Fait : port
      `MapKeysForMatch` rend `{MapID, Names}` (le map_id sans nom reste exploitable —
      map_name NULL mesure sur un match Vagabond), service `resolveBackgroundKey`.
- [x] A3. Migration Vagabond + Corpo : re-cuisson sous cle map_id, suppression des fichiers
      `fo08_wetland.*` / `fo11_blank.*` module-keyes, MAJ des tests/oracles qui les citent.
      Vagabond : PNG IDENTIQUE AU BIT (sha256 b5f21976..., calage et stats identiques).
      Corpo : calage IDENTIQUE, PNG mis a niveau — l'ancien fo11_blank.png datait d'avant
      le saut bloc/scen/mach du lot B (objets dessines 1725 -> 1976, sansModele 260 -> 9).
- [x] A4. Tests : unitaires himap (`TestCartesForgeDeclarations`), service
      (`TestMapBackground_ForgeParMapID`, `_ForgeJamaisViaCanevas`, oracle
      `TestMapBackground_TousLesFondsMapID`), garde-rail `TestFondForgeJamaisSousCleModule`
      (3 assertions : jamais de fond sous cle canevas, declaration => fond publie, fond
      uuid => declaration).

Gate A TENU (2026-08-13) : himap cibles + service + integration duckdb verts ;
`TestRenduForgeVagabond` vert sous la cle map_id (4/4 ancres, ecart median -0,01 m) ;
endpoint local `GET /api/v1/players/JGtm/matches/{id}/replay/background` : Vagabond
7344d24f -> 200 calage identique (65.6035/112.4380, 1332x1287, 0.092 m/px), Corpo
52fc79ef -> 200 calage identique (-50/77.068, 1088x1379). Commit.

## Phase B — Pilotes Starboard et Dredge (fermee)

Seules cartes jouees SEULES sur leur canevas (fo03_space, fo06_deepsea) — zero risque de
collision, chaine complete a blanc.

- [x] B1. Preuve level_id des 2 cartes (etendre la sonde) + entrees `map_quant_bounds.json`
      (via `mapquant-build`, table `mapModule` completee avec preuve citee). Mesure :
      Starboard -747133697 (0xD377A4FF) -> fo03_space, Dredge 2123870979 (0x7E97B303) ->
      fo06_deepsea, unicite 1/1 chacun ; catalogue regenere (23 cartes, diff +36 lignes =
      les 2 entrees seules).
- [x] B2. Ancres : entrees `map_objectives.json` pour les 2 map_id (`mapobj-build
      --from-file` sur `<carte>_map.mvar` — PAS le rack). Starboard 7a9265af... : 3964
      objets, 8 objectifs (emprise 24,9 x 21,2 m, z 80,6-85,5 — pas un rack) ; Dredge
      e4bb06db... : 5479 objets, 8 objectifs (15,8 x 27,0 m, z 76,1-79,8). Roles complets
      (CTF + oddball + 3 strongholds) sur les deux.
- [x] B3. Declarations `CartesForge` + cuisson ciblee + oracle des ancres. Starboard
      3902/3964 objets (98,4 %), Dredge 5410/5479 (98,7 %), ancres 16/16.
- [x] B4. Artefacts : `Desktop/gate_cartes_v75/mapid_pilotes/{starboard,dredge}.png`
      (146 Ko / 254 Ko). GATE VISUEL UTILISATEUR PENDANT.

Gate B TENU (2026-08-13) : oracle ancres 8/8 + 8/8 ; endpoint 200 — Starboard match
1af26997 (calage -58.5500/-31.5662, 1358x1318), Dredge match 113195e6 (-62.2092/64.5070,
1259x1381) ; determinisme : 2 cuissons -> SHA256 identiques (c3157d68..., aa0d0a62...).
Commit.

## Phase C — La masse, par nombre de matchs (fermee par liste)

Cartes jouees, canevas connu, `.mvar` au depot — ordre : The Pit (22), Snowbound (23),
Empyrean (29), Origin (25), Absolution (21), Curfew (20), Dynasty (20), Shiro (19),
Cliffside (18), Nemesis (18), Domicile (17), Fortress (17), Goliath (17), Isolation (17),
Solitude (17), Houseki (16), High Ground (15), Salvation (15), Takamanohara (15),
Elevation (14), Kiken'na (13), Banished Narrows (12), Kaiketsu (12), Obituary (12),
Opulence (12), Command (11), Fortitude (11), Refuge (11), Critical Dewpoint (10),
Perilous (10), Shogun (10), Sylvanus (10), Smallhalla (9). (Le reste <9 matchs : meme
chaine si le temps le permet, sinon liste au registre avec la commande de reprise.)

- [x] C1. Pour chaque carte, la MEME chaine que B — **33/33 cuites, 0 echec**. Preuves
      level_id 45/45 vertes (un level_id par canevas, unicite 1/1 par carte via son
      fichier-lien) ; bornes : catalogue 56 cartes ; ancres : 33 entrees map_objectives
      (catalogue 72 cartes) ; oracle des ancres 438/441 — 3 cartes a 1 ancre sans sol
      pres (The Pit 14/15, Absolution 16/17, Goliath 7/8), detail au rapport.
- [x] C2. Rapport final : `.ai/V7.5/cartes/RAPPORT_FONDS_MAP_ID_2026-08-13.md` (tableau
      complet + reliquat <9 matchs avec commande de reprise).
- [x] C3. Artefacts : 33 PNG copies dans `Desktop/gate_cartes_v75/mapid_masse/` (nommes
      par carte) — GATE VISUEL UTILISATEUR PENDANT, en une passe avec mapid_pilotes/.

Gate C TENU (2026-08-13) : oracle ancres par carte (rapport) ; `git status` = uniquement
les 66 nouveaux fichiers map_id + catalogues + chaine ; tests himap/service cibles verts
(+ suites service et port completes) ; golangci-lint 0 issue (goconst corrige par les
constantes Canevas*) ; endpoint 200 verifie sur The Pit (match 2b8aa0b9). Commit.

## Hors perimetre (ne pas traiter, deja au registre ou en file)

Live Fire (natif sans sbsp — investigation dediee), Detachment/Argyle (canevas inconnu —
investigation dediee), toile du canevas sous les objets, seuil des toits, variantes
Sentry Defense/Firefight, cartes <9 matchs non traitees en C.

## Decouvertes

1. **Le fond publie de Corpo etait perime** (pre-saut bloc/scen/mach du lot B) : l'ancien
   `fo11_blank.png` rendait 1725/1988 objets avec 260 sans modele ; la re-cuisson de la
   migration A rend 1976/1988 (9 sans modele), calage identique. Corrige DE FAIT par la
   migration — pas un fix hors perimetre.
2. **3 cartes ont 1 ancre d'objectif sans sol dessine** (The Pit 14/15, Absolution 16/17,
   Goliath 7/8). L'oracle faible dit « trou de reconstruction sous cette ancre », pas ou.
   A regarder au gate visuel utilisateur ; non bloquant (438/441 = 99,3 %).
3. **Des cartes traitees ont des assets UGC SECONDAIRES du meme nom** (Dynasty, Houseki,
   Shiro, Shogun, Salvation, Starboard — 1 a 3 matchs chacun) : autre map_id, pas de fond.
   Reponse actuelle : absence propre (jamais la carte d'un autre). Au registre.
4. **map_name est NULL sur certains matchs** alors que map_id est present (1 match
   Vagabond mesure) : la resolution map_id-d'abord les sert desormais — le repli par nom
   seul ne l'aurait jamais fait.

## Reprise de session

Avancement = ce fichier (statuts) + `git log --oneline` sur feat/v75 + le rapport C2.
Discipline de cloture : thought_log avant chaque commit, reports au REGISTRE_REPORTS.md
avec condition de reprise, CI de branche = a l'orchestrateur.
