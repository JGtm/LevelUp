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

- [ ] B1. Preuve level_id des 2 cartes (etendre la sonde) + entrees `map_quant_bounds.json`
      (via `mapquant-build`, table `mapModule` completee avec preuve citee).
- [ ] B2. Ancres : entrees `map_objectives.json` pour les 2 map_id (outil `mapobj-build`,
      variante `.mvar` du depot ; piege du rack de canevas — la variante aux objectifs
      dans <5 % de l'emprise est le RACK, pas la carte).
- [ ] B3. Declarations `CartesForge` + cuisson ciblee + oracle des ancres (aucune perdue).
- [ ] B4. Artefacts : `Desktop/gate_cartes_v75/mapid_pilotes/{starboard,dredge}.png`.

Gate B : oracle ancres 100 % sur les 2 cartes ; endpoint 200 sur un match reel de chaque ;
determinisme (2 cuissons = PNG identiques). Commit.

## Phase C — La masse, par nombre de matchs (fermee par liste)

Cartes jouees, canevas connu, `.mvar` au depot — ordre : The Pit (22), Snowbound (23),
Empyrean (29), Origin (25), Absolution (21), Curfew (20), Dynasty (20), Shiro (19),
Cliffside (18), Nemesis (18), Domicile (17), Fortress (17), Goliath (17), Isolation (17),
Solitude (17), Houseki (16), High Ground (15), Salvation (15), Takamanohara (15),
Elevation (14), Kiken'na (13), Banished Narrows (12), Kaiketsu (12), Obituary (12),
Opulence (12), Command (11), Fortitude (11), Refuge (11), Critical Dewpoint (10),
Perilous (10), Shogun (10), Sylvanus (10), Smallhalla (9). (Le reste <9 matchs : meme
chaine si le temps le permet, sinon liste au registre avec la commande de reprise.)

- [ ] C1. Pour chaque carte, la MEME chaine que B (preuve, bornes, ancres, declaration,
      cuisson, oracle). Une carte qui echoue N'ARRETE PAS le lot : ligne au rapport avec la
      raison mesuree, et on continue.
- [ ] C2. Rapport final : tableau carte / matchs / ancres / cuite ou raison d'echec —
      `.ai/V7.5/cartes/RAPPORT_FONDS_MAP_ID_2026-08-13.md`.
- [ ] C3. Artefacts : tous les PNG cuits copies dans `Desktop/gate_cartes_v75/mapid_masse/`
      — GATE VISUEL UTILISATEUR EN UNE PASSE a la fin (regle : batch, pas carte par carte).

Gate C : oracle ancres par carte ; `git diff --stat` = uniquement les nouveaux fichiers
map_id + catalogues ; tests himap/service cibles verts ; lint/vet 0. Commit.

## Hors perimetre (ne pas traiter, deja au registre ou en file)

Live Fire (natif sans sbsp — investigation dediee), Detachment/Argyle (canevas inconnu —
investigation dediee), toile du canevas sous les objets, seuil des toits, variantes
Sentry Defense/Firefight, cartes <9 matchs non traitees en C.

## Decouvertes

(a remplir en cours d'execution — rien d'autre ne se corrige dans ce lot)

## Reprise de session

Avancement = ce fichier (statuts) + `git log --oneline` sur feat/v75 + le rapport C2.
Discipline de cloture : thought_log avant chaque commit, reports au REGISTRE_REPORTS.md
avec condition de reprise, CI de branche = a l'orchestrateur.
