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
- [x] E.1 Ratchet « `analysis/` n'importe pas `games/{slug}` » posé avant.
- [x] E.2 Commit de déplacement seul ; suite Go complète + goldens identiques.
- [x] E.3 Cocher Notion 9.

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
- 2026-09-12 (correction utilisateur) : les films des 102 matchs restés en `killsource-2026-07-31`
  ne sont PAS expirés — 79 datent de 2025, 15 de 2024, 7 de 2023, 1 de 2026 ; le code ne connaît
  aucune date butoir. Mon affirmation « films expirés, irrécupérables » était fausse (déduite du
  seul cache local). Passe `backfill-killsource --online --gamertag JGtm` lancée (93 films à
  télécharger, du plus récent au plus vieux), serveur arrêté ; journal `killsource_online.log`.
  Reformulation : « morts sans source » = film non décodé (absent ou pas encore repassé) OU
  abstention du décodeur — pas une propriété des morts elles-mêmes.
- 2026-09-12 : passe `--online` finie (15:20 -> 16:38) : 65 films récupérés et décodés, 24 disparus
  (404), 4 sans kill feed. État `_latest` : 1 347 matchs sur le décodeur courant, 34 sur l'ancien
  (films perdus), 3 temps forts. **Morts sans source 27 807 -> 27 807 : le re-décodage n'en résout
  aucune.** Distribution (décodeur courant) : 82 matchs à > 50 % sans source portent 15 296 morts,
  dont 72 BTB de mars à novembre 2025 (88 % non attribués) ; 339 matchs à 10-49 % (6 247) ;
  98 à 1-9 % ; 828 à 0 %. L'hypothèse Notion du 10-09 est réfutée : c'est une abstention du
  décodeur sur les films BTB 2025, pas un retraitement manquant. DÉCOUVERTE MAJEURE, à instruire
  dans un lot dédié (décision utilisateur) — non traitée ici. Notion 7 reformulé avec ces chiffres.
- 2026-09-12 : lot E rendu (`wt/decodeur-sous-titre`, `5a0d1ec91` ratchet + `e64bf77f0`
  déplacement pur 981 renommages, 0 ajout/suppression ; `KillSourceDecoderRev` non montée,
  justifié). Gates verts sauf gate corpus (base tenue) : rejoué par le pilote après la passe.
- 2026-09-12 : gate corpus du lot E joué par le pilote (base libre, `--base=feat/v75`,
  `--parc-root` go-migration) : **8 témoins sur 8 `ok`, 53 -> 53, 0 gain, 0 perte** (journal
  `gate_corpus_lotE.log`). Lot E vérifié : déplacement pur prouvé. Revue adversariale : AUCUNE
  (renommage, calibrage du skill). Fusion dans feat/v75 DIFFÉRÉE jusqu'au retour de l'instruction
  BTB 2025 (branche partie de feat/v75 avant le déplacement — fusionner E en dernier évite de
  rebaser une branche vivante à travers 981 renommages). Branche poussée pour la CI. E.3 Notion 9
  se coche à la fusion. Serveur relancé (health 200). Instruction BTB 2025 lancée
  (`wt/btb-2025-abstention`, Opus).

## Lot G — abstention du décodeur sur les BTB 2025 (décidé par l'utilisateur le 2026-09-12,
## `wt/btb-2025-abstention`, base feat/v75 `2f5d165be`)
- [x] G.1 Instruction : cause (a) format du film — versions 39-40 (mars -> nov. 2025), gamertag à
      l'octet 12 ; `killsource`/`deaths_source`/`medal_feed_backfill` passaient version 0 = « en
      tête » ; 2 gamertags lus sur 25, portes `indice < nPlay` fermées. 92/92 BTB séparés sans
      recouvrement ; 8 films instrumentés (2025 : 5-17 % -> 82-100 % ; témoins 2024/2026 inchangés).
- [x] G.2 Correctif : résolution par mesure quand la version est inconnue (`2a6265ac4`),
      `KillSourceDecoderRev` -> `killsource-2026-09-12`, 3 tests CI + banc env + non-régression.
- [x] G.3 Revue adversariale (1 relecteur, algo) — question centrale : bump `SchemaVersion` 53 -> 54
      (deaths_source du rejeu lit le même parseur) ; puis fusion feat/v75.
- [x] G.4 Re-décodage des 210 films 39-40 (killsource, backlog par révision) + recuisson des
      artefacts touchés ; contrôle `_latest` par mois et playlist.
- Découvertes : 136 films 39-40 hors BTB touchés (arène) ; empreinte `killSourceDecoderFingerprint`
  ne hache que `killsource/` (ce correctif d'amont ne l'aurait pas fait sonner) ; manifestes du
  cache sans `FilmMajorVersion`.
- 2026-09-12 (décision utilisateur) : l'heuristique « résolution par mesure » de `2a6265ac4` est
  REFUSÉE — un décodeur lit l'indicateur clé, il ne le devine pas. Vérifié sur pièces par le
  pilote : `FilmMajorVersion` = u32 LE à l'offset 0 de `chunk_00.bin` (40 / 37 / 41 sur les
  3 films instrumentés) ; corrélation parfaite sur 92 BTB (v39-40 : 68-97 % sans source ;
  autres : 0-24 %) ; cache : v39 x26, v40 x185, v41 x1123, v<=38 x17. L'API porte la même
  valeur (`CustomData.FilmMajorVersion`), la synchro en direct la passe déjà ; seuls les chemins
  du cache passaient 0. Revue de la version heuristique arrêtée.
- [x] G.2bis Correctif retenu (2026-09-12, `wt/btb-2025-abstention`, 6 commits) : helper canonique
      `filmdec.FilmMajorVersionFromHeader` / `FilmMajorVersion` (u32 LE en tête de `chunk_00`) ;
      3 appelants branchés (`killsource/feed.go`, `replay/deaths_source.go`,
      `ops/medal_feed_backfill.go`) avec WARN si le registre manque ; heuristique de `2a6265ac4`
      SUPPRIMÉE (0 code mort) ; `haloclient.fetchFilmManifest` ne force plus 0 depuis le cache
      (le chemin de synchro EN DIRECT était touché lui aussi) ; `SchemaVersion` 53 -> 54 avec
      chronique datée, golden à une seule ligne d'écart ; Lot G étendu fait :
      `coverage.filmMajorVersion` publié (champ optionnel, contrat + types web regénérés).
      `[!]` justifié : pas de champ `film_major_version` dans le manifeste sérialisé (redondant
      avec l'en-tête, ne couvrirait aucun des 1 351 films déjà en cache).
      Parc par version lue : 1 351 films, 0 registre illisible, v31 x3, v33 x3, v37 x10, v38 x1,
      v39 x26, v40 x185, v41 x1123 ; croisement mesure/version : 1 divergence (`007d53a4`, film
      sans aucun highlight event — la mesure y est indéfinie).
      Gates : go build/vet/test ./... (171 paquets, 0 échec), integration sync, lint 0 issue,
      gate corpus 8/8 ok 0 perte (exit 0, joué deux fois). `tsc`/vitest non joués (pas de
      `node_modules` dans le worktree) — CI gate d'autorité.
- 2026-09-12 (décision utilisateur) : la version de film devient une DIMENSION du décodeur.
  Constat : seul le parseur des temps forts consomme `FilmMajorVersion` ; filmdec et le
  constructeur de rejeu l'ignorent (hypothèse implicite « tout est en 41 » ; cache : 1 123 films
  en 41, 211 en 39-40, 17 en 31-38 ; le seul artefact cuit d'un film 40 décode correctement,
  84/84 pistes nommées). (1) Lot G étendu : la version lue en tête est portée par le film chargé
  et publiée dans la couverture de l'artefact (champ optionnel) ; (2) ensuite une session de
  mesure par version (bancs identité/tirs/projectiles/objectifs sur 3 films 39, 3 films 40,
  2 films 31-38 contre témoins 41) — lot H, à lancer après G.

## Lot H — mesure par version de film (décidé le 2026-09-12, à lancer après G)
Consigne utilisateur : couvrir TOUS les calques, pas seulement le kill feed — socles (armes,
power-ups, équipement), armes au sol, trajectoires et positions (bipèdes, projectiles,
véhicules), vies/identité, tirs, grenades, objectifs (drapeaux, zones, crâne, bombe), équipement
(usages, poses, lâchers, ramassages, charges), médailles, score et manches, sons/événements.
Méthode : pour chaque banc/calque, 3 films v39, 3 films v40, 2 films v31-38 contre des témoins
v41 de même mode et carte quand c'est possible ; tableau calque x version = décodé / vide /
aberrant, avec les compteurs de couverture de l'artefact et les oracles existants (API, table
des scores, `swap.sh`, corpus). Livrable : rapport + liste des points où le profil de
déchiffrage doit brancher sur la version, chiffrés.
- [x] H.1 Corpus par version (choix des films, cache complet vérifié).
- [x] H.2 Mesure calque par calque (tableau).
- [x] H.3 Rapport et plan des divergences ; témoins v39/v40 ajoutés au corpus gate.

## Lot I — architecture « profil de déchiffrage » (proposé le 2026-09-12, APRÈS la release, sur
## feu vert utilisateur après H)
Relevé du 2026-09-12 : 60 000 L de décodage (filmdec 25 800 / 110 fichiers, replay 34 500,
killsource 4 700, objectiveevents 4 500, filmsource 500) ; deux lecteurs de bits (`filmdec.BitReader`,
`killsource.evReader`) ; six lecteurs d'octets bruts hors filmsource/filmdec ; ~12 globales de
paquet installées par film (install/restore) au lieu d'un profil ; `filmsource.Film` sans version
ni profil ; empreinte du décodeur limitée à killsource ; 6 fichiers filmdec > 500 L (traverse 1 380).
Cible : (1) `Film` porte version + profil immuable résolu au chargement (en-tête + catalogue de
carte) ; (2) profil passé explicitement, globales supprimées ; (3) porte d'entrée unique aux octets
bruts + ratchet archlint ; (4) empreinte étendue à filmdec/filmsource/parseur ; (5) équivalence
prouvée (goldens, corpus gate avec témoins 39/40, bascule bit à bit). Conçu d'après le tableau du
lot H, pas avant. Effort L.
- 2026-09-12 : lot G.2bis rendu (`bda707ac0`..`bd4eb83f5`, 7 commits) ; revue adversariale
  ronde 1 (2 relecteurs) : 0 P0, **4 P1** (les trois appelants corrigés ne sont couverts que par
  des bancs gardés par env — mutation « version 0 » verte en CI ; la montée de
  `KillSourceDecoderRev` n'est pas gatée par l'empreinte), 5 P2 (3 docs inversées/inexactes, WARN
  sans `match_id`, couverture non assertée). Corrections lancées (fixture v40 réduite + 3 tests CI,
  golden rev+empreinte, docs, WARN, test de couverture) ; ronde 2 ensuite. Doc d'architecture
  cible écrit : `.ai/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md` (`74f320235`).
- 2026-09-12 : ronde 1 corrigée (`fc7084801` fixture v40 888 Kio + 3 tests CI mordants ;
  `c0ee67a8b` golden rev+empreinte gatant les deux gestes ; `4ea83abd5` docs + WARN avec
  match_id + tests de couverture) ; ronde 2 : 0 constat, 6 mutations rougissent. **Fusion du lot G
  dans feat/v75** (merge de `wt/btb-2025-abstention`), gates sur la branche fusionnée : Go
  paquets touchés verts, tsc 0, vitest 3 015 (replay/api/match-replay), contrat généré stable ;
  push pour la CI. G.4 lancé : serveur arrêté, `backfill-killsource` (révision montée = tout le
  parc au backlog, ~2 h 30) puis `backfill-replay --only-existing` (schéma 54, ~23 min).
- 2026-09-12 (21:40) : feat/v75 (lot G) fusionné DANS `wt/decodeur-sous-titre` (`1ba55d468`) :
  4 fichiers de G suivis sous `games/halo_infinite/film`, 7 autres imports de G réécrits, chemin
  relatif d'une fixture corrigé, golden d'empreinte régénéré à révision inchangée (2 lignes
  d'historique). Gates : Go 171 ok, archlint, lint 0, openapi sans dérive, tsc 0. Reste pour E :
  gate corpus base libre (après la chaîne G.4), fusion dans feat/v75, Notion 9. Hygiène : 42 node
  et 1 git bloqué (sessions précédentes / sous-agent) coupés ; règle consignée en mémoire.
- 2026-09-13 : chaîne G.4 finie (killsource 20:26 -> 00:34, 1 351 films, 0 erreur ; recuisson 54
  00:34 -> 00:57, 76 artefacts, 0 erreur). **Contrôle `_latest` : morts sans source sur le
  décodeur courant 21 898 -> 7 677 (5,9 %)** ; BTB par année : 2023 22 % (9 matchs, v31-33),
  2024 4 %, **2025 27 % (102 matchs, était 88 %)**, 2026 10,6 %. Résidu 2025 supérieur aux
  années voisines : à instruire au lot H (par version 39 vs 40, par calque). Notion 7 mis à jour.
  Gate corpus du lot E lancé base libre.
- 2026-09-13 : gate corpus du lot E après intégration de G : 8/8 `ok`, 54 -> 54, 0 gain, 0 perte.
  **Lot E fusionné dans feat/v75** (`a3e4e516b`) ; `internal/analysis/{filmdec,replay}` n'existent
  plus, tout est sous `internal/games/halo_infinite/film/`. Serveur relancé (air, health 200).
  Notion 9 coché. Push feat/v75 pour la CI. Prochain : lot H.
- 2026-09-13 : lot H rendu (`wt/mesure-versions`, `d32b710a1` `b87cfb0e4` `ec02f1dfc`, aucun code
  de production). **Résultat central : la clé de déchiffrage est le BUILD (lu en clair dans la
  section d'identification de chunk_00 : HI_1_4_1 .. HI_1_13_0), pas la version majeure** — la
  frontière la plus nette tombe à l'intérieur de la v40 (HI_1_11_0 vs HI_1_12_0). Cinq
  divergences nommées : D1 lancers de grenade VIDES sur tous les builds < HI_1_12_0 (liste
  blanche `GrenadeTypeIDsByRank` datée d'un seul build ; 82 films, 5 000-10 000 lancers) ; D2
  identité éteinte sur 5 films sans section d'identification (v31/v33) ; D3 registre ECS par
  build (empreinte inconnue sur 228 films, journalisé en WARN sans conséquence) ; D4 marche des
  morts calibrée à vide sur les vieux films (= résidu BTB 2023) ; D5 bande de slots bipède
  [512,767] -> [512,8064] selon le build. Identiques prouvés : pied de film (la version ne
  commande QUE le gamertag), trajectoires, projectiles, armes au sol, socles, équipement,
  inventaire, véhicules, médailles, score. 4 témoins ajoutés au corpus (v39, v40 HI_1_11_0, v37,
  v33 sans identification) = 12. Gate corpus joué par le pilote base libre le 13-09 matin.
- 2026-09-13 : CI feat/v75 ROUGE sur `e528e347f` (job Coverage + Baseline) : la baseline de
  tests (`.ai/baselines/tests_pre_migration.jsonl`) nommait encore `internal/analysis/replay`
  (868 entrées) et `replay/mapvar` (56), déplacés par le lot E ; le lot E avait corrigé le
  script mais pas la baseline. Réécriture pure des chemins, poussée ; CI à surveiller.
- 2026-09-13 : gate corpus H : **12 témoins sur 12 `ok`, 54 -> 54, 0 gain, 0 perte** (dont les 4
  nouveaux par version). Lot H fusionné dans feat/v75 et poussé. Serveur relancé. Lot H clos ;
  les 5 points de branchement (P1-P5) relèvent du plan d'architecture (lot I), pas de ce plan.
  Suite : lot F (nettoyage worktrees/branches, bascule LevelUp) avec l'utilisateur.
