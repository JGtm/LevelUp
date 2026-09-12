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
- [x] B.2 Correctif `locateThrow` (auteur d'abord, toutes les naissances, rayon 4 m, `Slot` publié).
- [x] B.3 Garde-fou pas projectile > 10 m (`Rest=false`, compteur tronqués) ; mesure |Δy| − étendue Y.
- [x] B.4 Props Forge par carte (`MapGeometryDir(slug, module)`, `UNATTRIBUTED/` + README, test PathResolver).
- [x] B.5 Bump SchemaVersion, goldens, gate corpus, recuisson `backfill-replay --only-existing`.

## Lot C — robustesse (bilan 4a, 4b) + littéraux FR hors i18n (bilan 6)
- [x] C.1 `IsFileLockError` libellé Windows EN + test.
- [x] C.2 `IsFilmGoneErr` + test ; brancher sur les appelants qui retentent (à vérifier).
- [x] C.3 Les ~15 emplacements web du bilan passés par `Record<Locale, T>`.

## Lot D — tâches Notion 7 puis 6+8 (machine, serveur de dev arrêté)
- [x] D.1 `levelup backfill-killsource` (325 matchs sur l'ancien décodeur) ; contrôle par `_latest`.
- [x] D.2 `levelup seed citation-mappings` puis `backfill --all --citations-recompute-all`.
- [x] D.3 Cocher Notion 6, 7, 8.

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
- feat/citations-artilleur-vehicules était en retard de 161 commits sur feat/v75 ; le pilote s'est
  replacé sur feat/v75 (worktree `LevelUp-wt-v75` détaché, à supprimer au lot F).

## Journal
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
- 2026-09-11 : lot A rendu (`wt/badge-schema`, `40bf29a5c` `097a764ca` `5f82450bc`). Version
  courante du producteur en EN-TÊTE HTTP `X-Replay-Latest-Schema-Version` (le jumeau
  `domain/replaydoc` n'importe pas `analysis/replay` et la parité champ à champ est verrouillée
  par `replayview/parity_test.go`) ; badge `ReplaySchemaBadge` dans le h1 de `replay.tsx`, gate
  `isAdmin`, trois états (à jour / à recuire / inconnu sans en-tête), FR+EN. Revue A.4 du pilote :
  handler document Huma sans ETag (l'en-tête est toujours sur le 200 ; l'ETag ne concerne que
  l'image de fond) ; pas de CORS (même origine) ; test non-admin = DOM vide ; gates Go complet,
  tsc purgé, lint 0, vitest match-replay 2 720. CI et revue adversariale : une fois par vague
  (A+B+C), au moment de la fusion dans feat/v75.
- 2026-09-11 : lot B rendu (`wt/decodeur-fork`, `0288a35bd`..`b3f92f86d`, 34 fichiers). B.1/B.2 :
  médiane lancer -> lanceur 25-27 m -> 0,00 m sur Live Fire, pire cas 14,46 -> 0,56 m sur
  `000d5950`, 0 lancer perdu (repli biped), `slot` publié sur la branche projectile. B.3 : garde
  10 m + `coverage.projectiles` ; **cause caractérisée : un BIT qui bascule** (saut = étendue /
  2^k, k 1..7, 97 % des pas ; bit de poids fort de Y sur Live Fire `sgh_interlock`, axe X sur
  les cartes Forge) — cause racine NON traitée, à décider. B.4 : CSV attribué à Cliffhanger
  (`map_geometry/ridgeline/`, 90,8 % de l'emprise, commit d'origine = 2 .mvar Cliffhanger) ;
  les autres cartes n'ont plus de props (faux décor retiré). B.5 : schéma 52, gate corpus 7
  témoins toutes pertes voulues. `[~]` B.5 : la recuisson du parc est jouée par le pilote après
  fusion, serveur arrêté. Découvertes : `build.go` et `replaybuild.go` > 500 L grossissent ;
  17 films du parc absents de l'instantané v93 du registre ; le client ne dessine pas d'arcs.

## Lot B-bis — cause racine du bit qui bascule (déquantification projectile, filmdec) — décidé
## par l'utilisateur le 2026-09-12, AVANT la recuisson (`wt/bit-projectile`, base `wt/decodeur-fork`)
- [x] BB.1 Instruction : nommer le bit fautif (axe, rang) sur Live Fire `sgh_interlock` et une carte
      Forge ; distinguer signe / débordement / largeur de champ fausse / bornes fausses / delta.
- [x] BB.2 Correctif filmdec si la cause est prouvée ; sinon rapport et `[!]`.
- [x] BB.3 Mesure avant/après (pas coupés par film), schéma 53 si le contenu cuit change,
      goldens, gate corpus.

- 2026-09-12 : lot C rendu (`wt/robustesse-i18n`, `8c13b3b4c` `90e97c240` `efba88bbe`) : verrou
  Windows EN ; `IsFilmGoneErr` (= `isNotFoundErr` exporté, 404/410 manifeste OU blob) branché
  dans `killcollector/collector.go` — un blob mort remontait en Errors, jamais en
  `OutcomeNoFilm`, donc le match restait candidat à vie aux passes `--online` ; ~20 littéraux
  FR migrés + garde-rail `no-hardcoded-locale-fallback.guard.test.ts` ; `ChartsShowcasePage`
  `[!]` (labo, faible priorité). Gates : Go unit + intégration `-p 1` sync verts, tsc 0, eslint
  0 erreur, vitest 7 373. Découverte : 13 replis identiques sur Frags/Morts/FDA dans les
  fichiers timeseries (non cités par le bilan). Revue pilote : prédicat = 404/410 typés
  seulement, biais documenté ; OK.
- 2026-09-12 : lot B-bis lancé (Opus, effort maximal) sur la cause racine.
- 2026-09-12 : lot B-bis rendu (`wt/bit-projectile`, `1a93b34f2`..`a2dde7890`). **Cause prouvée
  (b) largeur de champ fausse** : la porte de `object-position-component` des objets du monde
  lisait un littéral de 3 bits (1 precHigh + 1 index-sel + 1 bit d'index de région) alors que
  la largeur d'index est PAR CARTE (`regionIndexBits` du catalogue : 1 sur 78 cartes, **2 sur
  Live Fire**) — les trois axes étaient lus un bit trop tôt, le MSB de Y était le LSB de X
  (40/40 pas vérifiés bit à bit). Correctif : porte = 2 + `IndexW`, index comparé à la région
  jouée (`world_object_gate_region_test.go`). Avant/après Live Fire : vols tronqués 239 -> 4,
  144 -> 1, 162 -> 0, 69 -> 0 ; points x6 ; grenades par projectile 1 -> 83 et 0 -> 137 à
  0,44 m de médiane ; autres cartes identiques au point près. Schéma 53, golden `000d5950` ne
  diffère que par la ligne de schéma, gate corpus 7/7 sans perte. Découvertes : le corpus
  témoin n'a AUCUN témoin Live Fire (ajouter `0797ce72`) ; queue Forge = faux positif du
  balayage ; `document_chronicle.go` 1 190 L ; records des autres régions refusés proprement
  (158/16 880). Incident : un `git stash`/`pop` par l'exécutant, sans perte, consigné.
- 2026-09-12 : revue adversariale UNIQUE de la vague A+B+B-bis+C lancée (3 relecteurs
  aveugles : correction décodeur, couverture de tests par mutation, killcollector + badge).
- 2026-09-12 : revue de vague rendue — ronde 1 (3 relecteurs) : 0 P0, 0 P1, 6 P2 tous corrigés
  dans les lots (`3395430b8` échappatoire --geometry + doc porte + banc ; `9e73145b5` en-tête
  exposé sous CORS ; `13866c98c` libellé WARN) ; ronde 2 (corrections seules) : 0 P0/P1, 1 P2
  (aucun test ne traversait CORS) corrigé par `0bc007fec`. Fusion dans feat/v75 décidée par
  l'utilisateur : `c44aa87ca` (bit-projectile, contient decodeur-fork), `380adae61` (badge),
  `4a1dcfdef` (robustesse-i18n) ; plan du pilote conservé aux conflits, journal en union.
  Gates sur la branche fusionnée : Go complet exit 0, tsc purgé 0, eslint 0 erreur, vitest
  7 385 (1 rouge corrigé `af0af06f2` : le test de porte de la route mockait `api.get`),
  contrat généré stable. Témoin Live Fire `0797ce72` ajouté au corpus (`4694733c4`). Push
  feat/v75 pour la CI de vague.
- 2026-09-12 : recuisson du parc au schéma 53 jouée (`backfill-replay --only-existing`, serveur
  arrêté, 76 artefacts, 0 erreur, journal scratchpad `recuisson_53.log`) : 76/76 au schéma 53,
  338 vols tronqués comptés en couverture, lancers par projectile 6 891 dont 6 549 avec slot
  (95 %), props seulement sur Cliffhanger (382). B.5 `[x]`. Lot D lancé en chaîne :
  `backfill-killsource` (1 210 films : `IsolationDecoderRev` montée le 10-09, tout le parc est
  périmé sur l'unité « faits d'isolement », pas 325) puis `seed citation-mappings` puis
  `backfill --all --citations-recompute-all` (journal `lot_d_chaine.log`).
- 2026-09-12 : lot D clos. `backfill-killsource` 12:10 -> 14:50 (1 210 films décodés, 1 384 matchs
  crédités, 138 806 morts), `seed citation-mappings` (2 insérées, 104 mises à jour),
  `backfill --all --citations-recompute-all` (JGtm 1 147, Madina97294 1 268, Chocoboflor 580,
  XxDaemonGamerxX 39 matchs ; V1-V4 OK ; 5 profils sans base sautés). Serveur relancé (health
  200). Notion 6, 7, 8 cochés avec les chiffres. **Contrôle `_latest` : 27 807 morts sans source,
  INCHANGÉ** (12 047 = 10,3 % sur le décodeur courant ; 15 326 sur 102 matchs restés en
  `killsource-2026-07-31`, sans film en cache, datés 2023 -> 03/2026, films expirés). Les 223
  matchs repassés n'ont gagné que 8 % d'attribution : l'hypothèse Notion du 10-09 n'est pas
  confirmée — à instruire (Découverte), pas dans ce plan.
