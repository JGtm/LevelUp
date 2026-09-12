# Plan — bilan du fork + séquence de release (tâches Notion 6-10) + badge admin (2026-09-11)

> Arbitré par l'utilisateur le 2026-09-11 (questionnaire) : ordre A, B, C, D, E, F ; liste des
> véhicules FIGÉE (Warthog, Gungoose, Scorpion) ; nettoyage limité au fusionné dans feat/v75,
> bascule LevelUp guidée ; mesure ti=9 lancée en parallèle du lot A.
> Base : feat/v75 `81f15be30`. Un lot = un worktree `LevelUp-wt-<slug>` + branche `wt/<slug>`.
> Source des points : `.ai/BILAN_FORK_CHASEWOODHAMS_2026-09-11.md`. Contrat : skill plan-execution.

## Lot A — badge admin « version de schéma » sur la page rejeu (`wt/badge-schema`)
- [x] A.1 API : la réponse du document de rejeu porte la version COURANTE du producteur à côté
      de `schemaVersion` (artefact lu), sans bump de `SchemaVersion`, sans casser la parité
      `analysis/replay` ↔ `domain/replaydoc`.
- [x] A.2 Web : badge visible SEULEMENT si `isAdmin` : « Schéma N · à jour » ou « Schéma N ·
      à recuire (dernier : M) », FR + EN via i18n de la feature, tokens sémantiques.
- [x] A.3 Tests : Go (handler/service), vitest (admin/non-admin, à jour/périmé), contrat
      `make generate-types` sans dérive, tsc, lint.
- [x] A.4 Journal + revue sur pièces par le pilote.

## Mesure (parallèle du lot A) — bilan point 3, en-tête keyframe 47 bits ti=9 (`wt/mesure-ti9`)
- [x] M.1 Banc `keyframe_fullstate_loop` avec HeaderBits=47 restreint aux entités ti=9 sur 3 films.
- [x] M.2 Critères : 8 entités, 4-4 stables sur tout le film ; comparaison à la table des scores.
- [x] M.3 Rapport `.ai/MESURE_ENTETE_TI9_47BITS_2026-09-11.md` ; aucun code de production.

## Lot B — corrections du décodeur (bilan 1, 2 garde-fou, 5) + recuisson (23 min)
- [x] B.1 Mesure grenades sur 3 films (médiane distance lancer→lanceur, part > 4 m).
      Banc `grenade_ecart_research_test.go` gardé par env (`0288a35bd`). Le film `36e80b83` du
      fork n'est pas dans notre parc ; retenus : `000d5950` (Cliffhanger) + `0797ce72` et
      `21ece4d8` (Live Fire).
- [x] B.2 Correctif `locateThrow` (auteur d'abord, toutes les naissances, rayon 4 m, `Slot` publié).
      Pire cas 14,46 → 0,56 m sur Cliffhanger, médiane 25-27 m → 0,00 m sur Live Fire, aucun
      lancer perdu sur ces deux-là. Golden : 70 → 69 lancers posés, justifié.
- [x] B.3 Garde-fou pas projectile > 10 m (`Rest=false`, compteur tronqués) ; mesure |Δy| − étendue Y.
      Forme mesurée et CONTRAIRE à celle du fork : le saut vaut étendue / 2^k (un bit), pas
      l'étendue entière. Cause racine caractérisée ici, TRANCHÉE et corrigée au lot B-bis.
- [x] B.4 Props Forge par carte (`MapGeometryDir(slug, module)`, `UNATTRIBUTED/` + README, test PathResolver).
      Le CSV est ATTRIBUÉ à `ridgeline` (Cliffhanger) sur deux preuves indépendantes, donc placé
      sous `map_geometry/ridgeline/` et non sous `UNATTRIBUTED/`. README + test de falsification.
- [x] B.5 Bump SchemaVersion, goldens, gate corpus, recuisson `backfill-replay --only-existing`.
      Schéma 51 → 52 avec chronique datée ; goldens mis à jour délibérément et comptés ; gate
      corpus `--reference=base` joué sur les 7 témoins, toutes les pertes voulues (verdict ligne
      par ligne au rapport). `[!]` **RECUISSON NON LANCÉE** — réservée au pilote, après fusion,
      serveur arrêté (consigne du lot).

## Lot B-bis — la cause racine du pas impossible (`wt/bit-projectile`, base `wt/decodeur-fork`)
- [x] BB.1 INSTRUCTION : du point publié aberrant jusqu'aux BITS, sur 2 films, 20 pas chacun.
      **Verdict (b) — largeur de champ fausse.** La porte d'`object-position-component` était
      écrite en dur à 3 bits ; l'index de région qu'elle porte fait DEUX bits sur Live Fire
      (seule carte du catalogue à `regionIndexBits = 2`). Les trois axes y étaient lus un bit
      trop tôt : le bit de poids faible de X devenait le bit de poids fort de Y. 40 pas sur 40,
      sur `0797ce72` et `21ece4d8`, portent la bascule du MSB de Y et d'aucun autre bit.
      (a) signe, (c) bornes, (d) delta : réfutées sur pièces — tableau au rapport.
- [x] BB.2 CORRECTIF `filmdec` : la porte suit `WorldObjectPrecision.IndexW` et l'index lu est
      COMPARÉ à la région jouée (ce que le jumeau bipède `decodeBipedI0Pos` faisait déjà).
      Records porteurs d'un pas impossible : 50,7 % -> 0,04 % (`0797ce72`), 52,8 % -> 0,02 %
      (`21ece4d8`), 48,5 % -> 0,02 % (`c88ec007`). Garde-rail `world_object_gate_region_test.go`.
- [x] BB.3 MESURE avant/après sur 10 films cuits deux fois (racine jetable, zéro écriture au parc) :
      Live Fire 614 vols tronqués -> 5, 1 067 points -> 8 378 ; les 5 cartes à index d'un bit et
      le témoin Cliffhanger IDENTIQUES au point près. SchemaVersion 52 -> 53, chronique datée,
      golden régénéré (il ne diffère que par la ligne de schéma). Gate corpus `--reference=base`
      `--base=wt/decodeur-fork` : sortie 0, 7/7 ok, 0 perte, verdict ligne par ligne au rapport.
      Contrôle croisé indépendant du seuil : la branche projectile des lancers de grenade
      retrouve 83 et 137 lancers à 0,44 m de médiane (elle était à 1 et 0 au lot B).
      `[!]` **RECUISSON DU PARC NON LANCÉE** — consigne explicite du lot, elle reste au pilote.
      Rapport : `.ai/RAPPORT_LOT_BBIS_BIT_PROJECTILE_2026-09-12.md`.

## Lot C — robustesse (bilan 4a, 4b) + littéraux FR hors i18n (bilan 6)
- [ ] C.1 `IsFileLockError` libellé Windows EN + test.
- [ ] C.2 `IsFilmGoneErr` + test ; brancher sur les appelants qui retentent (à vérifier).
- [ ] C.3 Les ~15 emplacements web du bilan passés par `Record<Locale, T>`.

## Lot D — tâches Notion 7 puis 6+8 (machine, serveur de dev arrêté)
- [ ] D.1 `levelup backfill-killsource` (325 matchs sur l'ancien décodeur) ; contrôle par `_latest`.
- [ ] D.2 `levelup seed citation-mappings` puis `backfill --all --citations-recompute-all`.
- [ ] D.3 Cocher Notion 6, 7, 8.

## Lot E — tâche Notion 9 : déplacement pur du décodeur sous `internal/games/halo_infinite/film/`
- [ ] E.1 Ratchet « `analysis/` n'importe pas `games/{slug}` » posé avant.
- [ ] E.2 Commit de déplacement seul ; suite Go complète + goldens identiques.
- [ ] E.3 Cocher Notion 9.

## Lot F — tâche Notion 10 : nettoyage worktrees/branches, bascule dossier LevelUp
- [ ] F.1 Inventaire daté (fusionné dans feat/v75 / non fusionné).
- [ ] F.2 Suppression du fusionné (worktrees, branches locales, distantes) ; `wt/ti11-cadre` conservée.
- [ ] F.3 Archivage `.ai/` racine → `V7.5/` ; sort de `cmd/investigate_matches`.
- [ ] F.4 Bascule vers `LevelUp` avec l'utilisateur ; cocher Notion 10.

## Découvertes (non traitées)
- Le corpus témoin (`config/replay_corpus.toml`) n'a AUCUN témoin Live Fire, donc aucun témoin
  d'une carte à index de région de 2 bits : le gate est sorti vert sur un correctif qu'il ne
  voyait pas. Ajouter `0797ce72` fermerait le trou (lot B-bis).
- La queue de pas impossibles des cartes Forge (612 pas, 5 films) n'est PAS la porte : leurs
  artefacts sont identiques avant/après. Signature d'un FAUX POSITIF du balayage par position de
  bit (Y figé au quantum près pendant que X saute d'une puissance de deux exacte). Le garde-fou
  de v52 la couvre et devient rare (lot B-bis).
- `document_chronicle.go` 1 119 -> 1 190 L : la convention de chronique fait grossir à chaque
  bump un fichier déjà au-delà du seuil de 500 L (lot B-bis).
- feat/citations-artilleur-vehicules était en retard de 161 commits sur feat/v75 ; le pilote s'est
  replacé sur feat/v75 (worktree `LevelUp-wt-v75` détaché, à supprimer au lot F).
- **(lot B) La cause du pas de projectile impossible est un basculement d'UN BIT du champ
  quantifié**, pas le repli de plage décrit par le fork : le saut vaut l'étendue de la carte sur
  un axe divisée par 2^k (97 % des 4 901 pas du parc, k de 1 à 7). Concentré sur `sgh_interlock`
  (Live Fire) au bit de poids fort de Y — exactement la moitié de l'étendue, 31,89 m sur
  63,775 m, 3 907 pas ; sur les cartes Forge l'axe touché est X et k vaut plutôt 7. Chantier
  `filmdec`, avec `coverage.projectiles.truncated` comme témoin par artefact.
- **(lot B) `build.go` (604 L) et `replaybuild.go` (569 L)** dépassaient déjà le seuil de 500 L
  avant ce lot et y gagnent 18 et 37 lignes. Une extraction est due, hors périmètre.
- **(lot B) 17 des 78 films du parc sont absents de l'instantané du registre partagé** (v93,
  2026-09-09) alors que leur artefact est cuit — dont `30724141`, dont la carte a dû être
  identifiée par ses bornes.
- **(lot B) Le client ne dessine pas les arcs de lancer** : ni `grenadeArcs.ts` ni
  `ARC_ORIGIN_RADIUS_M` dans `apps/web/src/features/match-replay/`. Le jumeau du rayon de 4 m
  n'existe donc pas encore côté web ; le commentaire Go le dit.

## Journal
- 2026-09-12 : lot B-bis rendu (`wt/bit-projectile`, `1a93b34f2` `fb71e9b3c` `c5a71dcbd`).
  La cause racine du point 2 du bilan est TRANCHÉE et corrigée : un bit de porte de trop peu sur
  la seule carte du catalogue à deux bits d'index de région, découvert en remontant du point
  publié jusqu'aux quanta bruts. Gates : `go build` / `go vet` / `go test ./...` verts (CGO),
  lint 0 issue, aucun fichier > 500 L ni fonction > 80 L introduit. Recuisson NON lancée.
- 2026-09-11 : plan écrit, worktrees `LevelUp-wt-badge-schema` et `LevelUp-wt-mesure-ti9` créés.
- 2026-09-11 (pilote, pré-mesure lot B sur les 76 artefacts `data/cache/replays/halo_infinite/*.json`,
  tous schéma 51) : grenades 8 938 dont **7 012 (78 %) situées par projectile, toutes avec
  `slot = 0`** (point 1 confirmé : aucun lanceur publié) ; **382 props identiques sur chacun
  des 76 artefacts** (point 5 confirmé) ; projectiles 15 735 trajectoires, **947 (6 %) portent
  au moins un pas > 10 m** (4 901 pas), concentrés sur quelques films (`0797ce72` 239/305 dont
  322 pas = étendue Y exacte ; `21ece4d8` 144/148 ; `30724141` 162/200 ; ailleurs 0-9 par film)
  — point 2 confirmé, avec une forme PAR FILM à instruire (le saut = étendue Y n'est net que
  sur `0797ce72`). La distance lancer -> lanceur (B.1) exige `slotFor` côté Go.
- 2026-09-11 : mesure M rendue (`wt/mesure-ti9`, `949bf64a2` + `d2861944d`) — **thèse confirmée** :
  en-tête keyframe PROPRE AU TYPE, ti=9 = 47 bits (seul préfixe sur 301 tenant le 4-4 sur 3
  films, 1 151 désignateurs, 0 vide ; biped 13 449 -> 0 composants à 47). Lien entité -> joueur
  non résolu (deux pistes réfutées). Exploitation NON recommandée (la base a déjà l'équipe).
  Découvertes : piège « 0 désync avec 0 composant » réel chez nous (bancs de calibration à
  vérifier) ; `document.go:345,558` affirment à tort « le film ne porte pas l'équipe ».
  Vérifié sur pièces par le pilote : bancs sautent sans films, `go test`/`go vet` verts.
- 2026-09-11 : lot B rendu (`wt/decodeur-fork`, 6 commits de `0288a35bd` à la clôture). Trois
  correctifs mesurés (grenades attribuées à leur lanceur, vols de projectile coupés au premier
  pas impossible, props Forge par carte), schéma 52, gate corpus joué sur les 7 témoins —
  sortie 1, toutes pertes voulues et cantonnées aux trois calques du lot. Suite Go + vet verts,
  lint 0 issue. Découverte majeure : la cause racine du pas impossible est un bit qui bascule
  (étendue / 2^k), pas le repli de plage décrit par le fork. Recuisson NON lancée (pilote).
  Rapport : `.ai/RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`.
- 2026-09-11 : lot A rendu (`wt/badge-schema`, `40bf29a5c` `097a764ca` `5f82450bc`). Version
  courante du producteur en EN-TÊTE HTTP `X-Replay-Latest-Schema-Version` (le jumeau
  `domain/replaydoc` n'importe pas `analysis/replay` et la parité champ à champ est verrouillée
  par `replayview/parity_test.go`) ; badge `ReplaySchemaBadge` dans le h1 de `replay.tsx`, gate
  `isAdmin`, trois états (à jour / à recuire / inconnu sans en-tête), FR+EN. Revue A.4 du pilote :
  handler document Huma sans ETag (l'en-tête est toujours sur le 200 ; l'ETag ne concerne que
  l'image de fond) ; pas de CORS (même origine) ; test non-admin = DOM vide ; gates Go complet,
  tsc purgé, lint 0, vitest match-replay 2 720. CI et revue adversariale : une fois par vague
  (A+B+C), au moment de la fusion dans feat/v75.
