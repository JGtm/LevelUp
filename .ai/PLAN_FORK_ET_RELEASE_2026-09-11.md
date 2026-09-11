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
- [~] B.5 Bump SchemaVersion, goldens, gate corpus, recuisson `backfill-replay --only-existing`.

## Lot C — robustesse (bilan 4a, 4b) + littéraux FR hors i18n (bilan 6)
- [x] C.1 `IsFileLockError` libellé Windows EN + test.
- [x] C.2 `IsFilmGoneErr` + test ; brancher sur les appelants qui retentent (à vérifier).
- [x] C.3 Les ~15 emplacements web du bilan passés par `Record<Locale, T>`. `[!]`
      `ChartsShowcasePage.tsx` (page de labo, faible priorité, ~45 lignes accentuées,
      non traitée — cf. journal).

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
- feat/citations-artilleur-vehicules était en retard de 161 commits sur feat/v75 ; le pilote s'est
  replacé sur feat/v75 (worktree `LevelUp-wt-v75` détaché, à supprimer au lot F).
- (lot C, C.3) Même défaut « repli `?? '<littéral FR>'` quand `fieldMappings` n'est pas chargé »
  sur des CHAMPS NON cités par le bilan, dans les mêmes fichiers déjà touchés :
  `TimeseriesPage.distributions.tsx` (7 occurrences `?? 'Frags'/'Morts'/'FDA'`),
  `TimeseriesPage.summary.tsx` (6 occurrences, mêmes champs). Non traité (hors liste du bilan,
  volume trop grand pour rester mécanique dans ce lot — 13 occurrences supplémentaires).
- (lot C, C.3) `eslint-rules/no-hardcoded-strings.js` a deux angles morts documentés dans le
  nouveau garde-rail `apps/web/src/lib/i18n/no-hardcoded-locale-fallback.guard.test.ts` : un
  littéral FR assigné à une variable avant usage JSX (non vu par le visiteur `JSXAttribute`), et
  un littéral court (< 3 mots ET < 15 caractères, ex. « Précédent ») sous le seuil
  `looksLikeUserContent`. Le garde-rail ajouté ferme la lacune sur les fichiers de ce lot
  uniquement — une passe eslint dédiée (scope analysis ou lint TS custom) resterait à faire pour
  fermer la lacune partout.

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
- 2026-09-11 : lot C rendu (`wt/robustesse-i18n`, base feat/v75 `81f15be30`). C.1 :
  `IsFileLockError` reconnaît le message OS Windows EN « process cannot access the file because
  it is being used by another process » (insensible à la casse), en plus du marqueur DuckDB
  « File is already open in » qui ne couvre QUE le cas où DuckDB identifie lui-même le détenteur
  du verrou. C.2 : `haloclient.IsFilmGoneErr` exporte `isNotFoundErr` (déjà typé
  `*HTTPError`/`*BlobHTTPError`, manifeste ET blobs, 404/410) ; appelant branché =
  `killcollector.collect()` — un manifeste vivant dont un blob CDN pré-signé rend 404/410
  (expiration partielle) remontait une ERREUR non-nil, jamais classée `OutcomeNoFilm`, donc
  `MBitFilmAbsent` n'était JAMAIS posé et le match restait candidat à vie aux passes
  `backfill-killsource --online` ; reclassé en `OutcomeNoFilm` sans erreur, `slog.WarnContext`
  avec `match_id`. Une panne transitoire (503, rate-limit) reste une erreur retentée (biais
  assumé, bilan pt 4b). Callers vérifiés sans retry à brancher : `GetFilmChunkURLs`
  (build-queue, résout le manifeste seul, erreur immédiate à l'admin, pas de retry en boucle),
  `GetHighlightEventsChunk` (déjà typé côté blob, cf. commentaire d'origine), `LocalCacheFilms`
  (hors ligne, aucun réseau). C.3 : ~20 littéraux FR en dur migrés vers les manifests TOML
  existants (`common`, `palmares`, `feedback_drawer`, `synthesis`, `timeseries`) sur les 10
  fichiers cités par le bilan — deux catégories : (a) littéral direct en JSX (carousel.tsx,
  StepPlayer.tsx, XboxLoginPage.tsx) ; (b) littéral assigné à une variable AVANT usage JSX
  (ThemeToggle.tsx — angle mort de la règle eslint `no-hardcoded-strings`, qui ne suit pas les
  variables) ; (c) corps d'issue GitHub entièrement en dur (buildIssueUrl.ts, ~20 lignes,
  `locale` maintenant un paramètre requis) ; (d) replis `?? '<littéral>'` quand `fieldMappings`
  n'est pas chargé (SynthesisPage, TimeseriesPage×4) — repli = clé manifest de la feature, plus
  un littéral. Garde-rail neuf `lib/i18n/no-hardcoded-locale-fallback.guard.test.ts` (grep
  source, ferme les deux angles morts eslint documentés en tête de fichier). `[!]`
  `ChartsShowcasePage.tsx` non traité (faible priorité déclarée par le plan). Découverte non
  traitée : 13 occurrences du même défaut sur des champs non cités par le bilan (Frags/Morts/
  FDA) dans les fichiers timeseries déjà touchés — notée ci-dessus, hors liste du bilan. Gates :
  Go `go test ./internal/platform/duckdb/... ./internal/sync/...` (vert, ~140 s) + `go vet ./...`
  (vert) + `go test -tags=integration -p 1 ./internal/sync/...` (vert, ~230 s, touché
  `killcollector` au-delà de `haloclient`) ; web `tsc -b` (0 erreur), `eslint .` (0 erreur, 31
  warnings pré-existants hors périmètre), `vitest run` (695 fichiers / 7373 tests, 1 skip
  jsdom canvas). Commits : `8c13b3b4c` (C.1), `90e97c240` (C.2), commits C.3 à suivre.
