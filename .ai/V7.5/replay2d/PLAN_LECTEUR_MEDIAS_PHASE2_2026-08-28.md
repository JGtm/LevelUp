# Plan — Lecteur phase 2 : medias sur la frise (donnee) + repli des pistes

> Ecrit le 2026-08-28 par la session pilote, sur cartographie complete de l'existant medias
> (agent d'exploration, faits fichier:ligne verifies). Contrat : skill `plan-execution`
> (ordre strict, statuts [x]/[~]/[!], zero fix hors perimetre). Grille : skill `plan-review`.
> Branche : `wt/lecteur-medias` (worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-lecteur`,
> base origin/feat/v75 @ f87b4bf29). 1 commit par lot. AUCUN merge/push sans decision pilote.
> Solde le report « Medias du rejeu : la DONNEE » du REGISTRE_REPORTS.

## Decision d'architecture (tranchee) — OPTION A : enrichir `match_view.media_tab`

PAS de nouvel endpoint. La page rejeu appelle DEJA `useMatchView` (replay.tsx:68) et le
match sert DEJA ses medias associes : Q24 (`platform/duckdb/queries_match_detail.go:175`)
-> `GetMatchMedia` (`match_view_repo_extras.go:322`) -> `buildMediaTab`
(`service/match_view_builders_team.go:276`) -> DTO `MatchAssociatedMedia`
(`domain/match_view.go:708`) -> URLs transformees (`handlers/match_view.go:240`).
Zero appel reseau de plus, cache partage. Deux trous a combler (bug latent connu) :
`duration_seconds` JAMAIS peuple (le champ DTO existe, Q24 ne le SELECT pas) et
`capture_time` = capture_END (poserait chaque clip decale de sa duree).

Regle temporelle (decision utilisateur du 2026-08-28 : « soustraire la duree ») :
- clip : debut = `capture_start_utc` si present, sinon `capture_end_utc - duration` ;
- image : instant = `capture_start_utc` (== end) sinon `capture_end_utc` ;
- axe du rejeu (cote CLIENT, comme le fil — killFeedLogic.ts:16) :
  `replayMs = (captureStart - header.start_time) - (doc.originMs ?? 0)`
  (capture - start_time_utc == event_time_ms + t0Ms ; approximation assumee, la fenetre
  de gameplay ecarte ce qui tombe hors bornes via placeMedia). Pas de recalage cote Go —
  on sert les timestamps absolus + duree, le client soustrait UNE fois (meme doctrine que
  le fil, useReplayTimeline.ts:10-14).

Vignettes : les « GIFs » auto sont aujourd'hui des WebP ANIMES
(`ops/media_thumbnails.go:159`, `{stem}.webp`, legacy .gif encore servis) — un `<img>`
les anime nativement : la piste et la bande de la lightbox les reprennent TELS QUELS via
`thumbnail_path` transforme en URL. Rien a generer.

## LOT 1 — Go : Q24 + DTO (les deux champs manquants)

- [x] 1-1 `queries_match_detail.go` Q24 : ajouter `mf.capture_start_utc, mf.duration_seconds`
      au SELECT (lecture via `media_match_associations_latest` INCHANGEE — regle ART).
- [x] 1-2 `match_view_repo_extras.go` `GetMatchMedia` : scanner les 2 colonnes ;
      `domain/match_view.go` `MatchAssociatedMedia` : peupler `DurationSeconds` (champ
      existant, bug latent solde) + nouveau champ `CaptureStartTime *string` RFC3339
      (`capture_start_time,omitempty`) — `CaptureTime` (end) reste tel quel, l'onglet
      medias du match le consomme.
- [x] 1-3 `match_view_builders_team.go` `buildMediaTab` : assigner les 2 champs.
- [x] 1-4 Tests : etendre `match_view_repo_media_test.go` (colonnes start+duration servies ;
      cas duration NULL ; le cas TeammateOnly existant reste vert).
- [x] 1-5 Contrat : `make openapi-gen` + `make generate-types` + `make openapi-check`
      (JAMAIS d'edition manuelle d'openapi.yaml ni de generated.ts).
- [x] 1-G Gate : `go test ./internal/platform/duckdb/... ./internal/service/... ./internal/domain/...`
      + `go vet ./...` (apps/go-api, CGO msys64, PAS de builds go concurrents) ; typecheck
      web VERT apres generate-types. Commit lot 1.

Journal lot 1 (2026-08-28) : les 5 ancres du plan verifiees sur pieces, toutes exactes
(Q24 l.175, GetMatchMedia l.322, buildMediaTab l.276, DTO l.709, transform URL l.240).
Une DEVIATION assumee : `media_files.duration_seconds` est un DOUBLE en base
(`ops/media_store.go:45`) alors que le DTO expose `*int` — scan en `sql.NullFloat64` puis
`math.Round`. Changer le type du DTO aurait ete un 3e changement de contrat, exclu par le
perimetre ferme ; le cout est au pire une demi-seconde de placement, sous l'approximation
du recalage lui-meme. Fixture partagee NON modifiee (les scenarios de galerie s'appuient
dessus) : le nouveau test pose start+duree par UPDATE cible sur med-A1 et laisse med-C1 en
NULL/NULL pour couvrir l'absence. Diff openapi = UN champ (`capture_start_time`) ;
`duration_seconds` etait deja au schema, jamais peuple — c'est tout le bug latent.
Gates : go build exit 0 ; `go test -tags=integration -run TestMatchViewRepo_GetMatchMedia`
6/6 PASS (dont TeammateOnly, reste vert) ; `go test ./internal/platform/duckdb/...
./internal/service/... ./internal/domain/...` 13 packages ok ; `go vet ./...` exit 0 ;
openapi-gen + generate-types + openapi-check (2 maillons) exit 0 ; `npm run typecheck`
exit 0.

## LOT 2 — Web : mapper pur + branchement de la prop

- [x] 2-1 `replayMediaLogic.ts` (nouveau, match-replay) : `buildReplayMedia(mediaTab,
      header, originMs)` -> `ReplayMediaItem[]` pur. Regles : mapping kind
      (`video`->'clip', `image`->'image') ; regle temporelle ci-dessus ; `durationMs =
      duration_seconds*1000` (clip sans duree ET sans start = pose ponctuelle a end,
      assume) ; `id = String(file_id)` (JAMAIS file_path — colonne MUTEE, piege n°7 de la
      carto) ; `thumbUrl = thumbnail_url ?? (image ? url : '')` ; `label = basename`
      (+ auteur si le DTO le porte — verifier sur pieces) ; item sans le moindre timestamp
      = ECARTE (jamais une pose inventee).
- [x] 2-2 Tests `replayMediaLogic.test.ts` : soustraction duree (clip end-only), start
      prioritaire, image ponctuelle, originMs absent, duration NULL, id stable, ecartes.
- [x] 2-3 Branchement : `replay.tsx` memo `buildReplayMedia(...)` -> prop `media` de
      `ReplayCanvas` -> option `media` de `useReplayTimeline` (remplace `EMPTY_MEDIA` a
      l'unique ligne 99-104 ; la constante devient la VALEUR PAR DEFAUT de l'option, son
      commentaire mis a jour — phase 2 livree).
- [x] 2-4 Capability : la piste Medias ne se rend que si le titre porte `media`
      (le rejeu n'est garde que par `matchmaking` — piege releve par la carto) : prop
      `showMediaTrack` sur ReplayTimelineTracks, servie par le mecanisme capability web
      existant (FeatureGate/hook — verifier sur pieces lequel), rangée ABSENTE sinon
      (pas un etat vide menteur). Cross-joueur assume : un clip d'un coequipier apparait
      (meme regle que l'onglet medias du match).
- [x] 2-G Gate : typecheck ; vitest match-replay + routes ; ESLint. Commit lot 2.

Journal lot 2 (2026-08-28) : ancres verifiees sur pieces — `EMPTY_MEDIA` bien a
useReplayTimeline.ts:99-104, `useMatchView` bien a replay.tsx:68, doctrine de recalage
bien a killFeedLogic.ts:16. Mecanisme capability retenu : le hook `useCapability('media')`
(lib/capabilities), appele DANS `useReplayTimeline` — `FeatureGate` est un composant, il
aurait impose une rangee enveloppee dans la grille a 2 colonnes. Fail-open assume (c'est
le contrat du hook) : bootstrap non charge = piste affichee, no-op mono-titre.
DEUX DECISIONS non ecrites au plan, prises en lecture conservatrice : (a) en-tete sans
`start_time` = AUCUN media place (rien a recaler, on n'invente pas) ; (b) `originMs` absent
= 0, la pose se degrade du decalage de l'image zero plutot que de faire disparaitre la
piste — meme choix que le reste du lecteur.
CLIQUET ReplayCanvas : la prop `media` coutait 3 lignes sur un fichier a 672/672. Compense
DANS LE PERIMETRE : le bloc de commentaire de `feedEntries` couvre desormais les deux listes
(meme doctrine, meme phrase) et celui de l'appel au hook perd sa ligne de rappel historique.
Fichier a 672 exactement, cliquet vert.
Gates : `npm run typecheck` (cache purge) exit 0 ; vitest `src/features/match-replay` +
`src/routes` = 104 fichiers / **1598 tests, 0 echec** (+16 : 15 mappeur, 1 rangee absente) ;
ESLint 0 erreur sur les 9 fichiers touches, 1 warning PRE-EXISTANT (exhaustive-deps
objectiveObjects, deja consigne au lot 5 du chantier lecteur).

## LOT 3 — Web : lightbox compatible HLS

Piege carto n°5 : les clips transcodés ont `file_path` mue vers `master.m3u8` — le
`<video src>` nu de la lightbox ne les lira PAS sur Chrome/Firefox. La galerie a deja
resolu (CoverFlowModal + hls.js, teste par `CoverFlowModal.hls.test.tsx`).

- [ ] 3-1 Reutiliser le mecanisme HLS de la galerie dans `ReplayMediaLightbox` — par
      EXTRACTION d'un helper/hook partage si le code de la galerie n'en expose pas un
      (regle 6 : pas de 2e copie divergente ; si extraction, la galerie migre dessus dans
      le meme lot). MKV/AVI remux sans Range (pas de seek) : assume, commentaire.
- [ ] 3-2 Tests : patron du test hls existant de la galerie applique a la lightbox
      (m3u8 -> hls.js attache ; mp4 -> src direct).
- [ ] 3-G Gate : typecheck ; vitest match-replay + media ; ESLint. Commit lot 3.

## LOT 4 — Web : repli des pistes (retouche utilisateur)

- [ ] 4-1 Bouton de repli sur la frise (chevron, zone des libelles) : replie = SEULE la
      barre de progression (+ horloges de bornes) reste ; deplie = les 4 pistes.
      Preference PERSISTEE (patron `usePersistedFlag`, useReplaySettings.ts:254 — meme
      mecanisme que les autres reglages du lecteur), defaut DEPLIE. aria-expanded +
      libelles FR/EN (i18nContract + 2 tables, parite typee).
- [ ] 4-2 Tests : repli masque les pistes et garde le curseur ; persistance ; aria ;
      le garde-fou TIMELINE_SHORTCUT_ATTR reste vert (le curseur ne bouge pas de place).
- [ ] 4-G Gate : typecheck ; vitest match-replay ; ESLint. Commit lot 4.

## LOT 5 — Cloture

- [ ] 5-1 REGISTRE_REPORTS : solder « Medias du rejeu : la DONNEE » (reference commits) ;
      le report « gate visuel lecteur » s'enrichit des points medias + repli a verifier.
- [ ] 5-2 Journal chantier + entree thought_log racine (patron des entrees existantes).
- [ ] 5-G Gates transverses : typecheck cache purge ; vitest complet match-replay +
      routes + media ; `go test ./...` apps/go-api VERT ; `make openapi-check` propre ;
      ESLint 0 sur touches. Commit docs. PAS de merge, PAS de push.

## Perimetre ferme (ce plan NE fait PAS)

- Pas de nouvel endpoint HTTP, pas de changement d'openapi au-dela des 2 champs.
- Pas de recalage temporel cote Go ; pas de retouche de l'association media<->match
  (fenetre 2 min, priorites) ni du pipeline vignettes ; pas de reintroduction de
  colonnes droppees (liked/liked_at — piege carto).
- Pas de likes sur la frise ; pas de multi-vignettes par clip (thumbUrls[] reste le
  point d'extension documente) ; pas de seek custom sur les remux WebM.
- Taxonomie de modes hard-codee Halo Infinite de MediaRepo (follow-up documente
  wire/registry_media.go:25-27) : HORS perimetre, ne pas toucher.

## Environnement

- vitest HORS sandbox ; `npm run typecheck` fait foi (purger node_modules/.tmp avant
  conclusion) ; Go : CGO msys64, jamais deux builds go en parallele ; worktree deja
  npm-installe (chantier lecteur).
- Le depot bouge (sessions paralleles sur feat/v75) : rebaser/verifier avant tout gate
  final si origin/feat/v75 avance.

## Protocole de reprise

Ce plan (statuts), derniere entree du journal replay2d, `git log --oneline -6` sur
wt/lecteur-medias. Un seul lot ouvert a la fois.

## Decouvertes (a consigner, ne pas traiter)

- (carto 2026-08-28) `MatchMediaTab.tsx:40` s'appuie sur `duration_seconds` toujours
  null — le lot 1 le repare de fait ; verifier qu'aucun affichage ne casse en le voyant
  arriver non-null.
  VERIFIE au lot 1 : `duration_seconds` n'y sert que de REPLI quand `item.kind` est absent,
  et Q24 sert toujours `mf.kind` — la branche n'est atteinte que par un cache navigateur
  d'avant le fix kind. Rien ne casse ; le repli devient meme juste (un clip y etait classe
  'screenshot' faute de duree). Defaut PRE-EXISTANT laisse tel quel (hors perimetre) : le
  test est `!== null` alors que le champ est `omitempty`, donc `undefined` quand absent —
  une image sans `kind` est classee 'clip'. Comportement INCHANGE par le lot 1 (une image
  n'a toujours pas de duree).
- (lot 2, 2026-08-28) DEUX schemas media coexistent au contrat : `MatchAssociatedMedia`
  (celui que `media_tab.media_items` sert reellement) et `AssociatedMediaItem` (un autre
  schema Go, sans `capture_start_time`, avec `duration_seconds` en float). `MatchMediaTab.tsx`
  se type sur le SECOND alors qu'il recoit le premier : les champs coincident aujourd'hui,
  donc rien ne casse et `tsc` ne voit rien. HORS PERIMETRE — le mappeur du rejeu, lui, se
  type sur `MatchMediaTab['media_items']`, c'est-a-dire sur ce qui arrive vraiment.
