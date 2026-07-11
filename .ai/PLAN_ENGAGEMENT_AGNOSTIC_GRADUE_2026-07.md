# PLAN — Engagement title-agnostic gradué + activation H5 (chantier F7)

> **STATUT : PARTIEL — E1→E5 + E6a + garde-rails LIVRÉS (2026-07-11).** Reste UNIQUEMENT des
> dépendances non automatisables : gate humain E6b (décision utilisateur sur les scores H5,
> protocole ci-dessous), smoke visuel H5, re-backfill PROD (post-merge). H5 est activé en
> `degraded` (score servi avec mention « calibration provisoire ») — sûr en l'état. Passage à
> `supported` = décision E6b. Tous les gates automatisés verts (Go unit + intégration `-p 1`,
> front typecheck/eslint/vitest, lint delta 0, byte-identité Infinite prouvée). Plan NON déplacé
> vers V7 (PARTIEL, dépend d'une décision utilisateur).
>
> Date : 2026-07-10. Auteur : Fable (supervision). Exécutant prévu : Opus.
> Origine : décision utilisateur 2026-07-10 (échange F7) — « H5 est plus riche en events
> in-match : rendre l'engagement davantage agnostic, capable de recevoir PLUS de données
> qu'Infinite ; en deçà d'un lot de données ou d'une finesse donnée, l'engagement est
> dégradé ou non supporté ; coefficients par jeu (gameplay/dynamique différents) ».
> Mémoire de fond : [[project-h5-engagement-canonicalization-chantier]] (constat 2026-07-04 :
> compute déjà pur et agnostic, H5 coupé par l'adapter + calibration cold-start Infinite).
>
> **Contrat d'exécution : skill `plan-execution`, intégralement** (ordre strict, statuts
> [x]/[~]/[!] justifiés, vérifier sur pièces avant de coder ET avant de cocher, zéro fix
> hors périmètre → §Découvertes, thought_log + MAJ de CE fichier à chaque clôture de phase).
> Reprise de session : relire §0, puis le Journal §J, reprendre à la première case non statuée.
>
> **Branche** : `feat/engagement-agnostic-gradue` (nouvelle, depuis main APRÈS le merge de
> la campagne d'audits). JAMAIS main direct (push main = deploy prod auto).

## 0. DÉPENDANCE BLOQUANTE — ordre des chantiers engagement

**P0 — [PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md](PLAN_ENGAGEMENT_REFONTE_LOBBY_2026-07.md)
D'ABORD.** Ce plan-là change le CŒUR du modèle (attendu ancré lobby, bins d'intensité,
poids death 0, 3 courbes). Calibrer H5 avant = calibrer contre un modèle qui va changer
(double calibration garantie). CE plan-ci ne démarre que lorsque la refonte lobby est
CLOSE (ses 5 phases livrées + re-backfill 2 titres fait). Un seul recouvrement autorisé :
la Phase E1 (canonicalisation, purement structurelle) peut se faire en parallèle de la
refonte lobby si besoin — AUCUNE autre phase.

## 1. Objectif et critères de succès

Objectif : l'engagement devient un sous-système **title-agnostic gradué** — chaque titre
alimente un vecteur de signaux extensible (H5 peut en fournir PLUS qu'Infinite), une
double porte décide du statut (suffisance des signaux, puis calibration), les coefficients
restent par titre — et **H5 est activé** (`degraded` puis `supported` après validation).

Critères de succès mesurables :
- `engagement.score` H5 passe de `not_exposed` à `degraded` (puis `supported` après gate
  humain de calibration) SANS branche `slug == "halo_5"` (ratchet anti-slug vert).
- Un titre futur (Halo 7) s'active en fournissant : mapping de signaux dans son adapter
  + coefficients calibrés — ZÉRO modification du moteur `temporal`.
- Le score Infinite reste byte-identique tant que ses inputs ne changent pas (tests
  golden avant/après sur fixtures Infinite).
- Sous le seuil de suffisance de signaux, le statut est `not_exposed`/`degraded` — jamais
  un score silencieusement faux (règle : dégradation gracieuse, pas de panic, pas de mensonge).

## 2. Décisions PRÉ-TRANCHÉES (utilisateur 2026-07-10 + reco superviseur — ne pas re-questionner)

| # | Décision | Choix ferme |
|---|---|---|
| DE-1 | Séquence | Refonte lobby (plan existant) AVANT ce chantier (P0). Seule E1 parallélisable |
| DE-2 | Modèle d'entrée | Vecteur de signaux extensible + masque de présence — chaque titre remplit ce qu'il expose, le moteur ne connaît AUCUN titre |
| DE-3 | Double porte | (1) Suffisance : sous un ensemble minimal de signaux → `not_exposed` ; partiel → `degraded` ; complet → éligible `supported`. (2) Calibration : coefficients non validés pour le titre → plafonné `degraded` |
| DE-4 | Coefficients | PAR TITRE (gameplay/dynamique différents — acté utilisateur). Format : fichier de config par titre (pattern `games/damage_model.go` + `constants.toml`), pas de constantes Go partagées |
| DE-5 | Richesse H5 | Les events riches H5 (kill mechanics, impulses…) ENTRENT dans le vecteur comme signaux optionnels pondérés — mais ne paient qu'après calibration (un signal non calibré n'améliore pas un score, il le fausse) |
| DE-6 | Activation | H5 : `not_exposed` → `degraded` dès E5 (calcul branché, calibration provisoire) → `supported` UNIQUEMENT après le gate humain E6 (« les scores ont-ils du sens ? » — non automatisable) |
| DE-7 | Pas de flag | Aucun feature flag. Le mécanisme légitime = statuts de capability fine (`capabilities.toml`) + statut de confiance par match dans la réponse API |
| DE-8 | Périmètre UI | Le front affiche l'état dégradé (badge/mention discrète, FR+EN) — pas de score « nu » quand la confiance est réduite |

## 3. Phases (ordre strict ; 1 phase = 1+ commits `engagement(EX):`)

### E1 — Canonicalisation : l'engagement devient un métrique de 1er ordre
Constat mémoire : le score (0-100) n'est PAS un FieldKey canonique (contrairement à
`performance_score`, `offensive_conversion`, `defensive_resistance`) — c'est le verrou n°1.
- [x] E1a — `internal/games/canonical/fields.go` : FieldKey `engagement_score` (unité sans
  dimension, bornes [0,100] documentées dans le commentaire) ajouté au groupe `derived` +
  `AllFieldKeys()` ; count test 59→60 ; golden `fields.golden.txt` MAJ (FieldKey + TOML dans
  le MÊME commit, recette `canonical-types`).
- [x] E1b — `config/titles/halo_infinite/mappings/fields.toml` + `halo_5/mappings/fields.toml` :
  section `[fields.engagement_score]` (labels FR/EN, description FR/EN, format `integer`,
  `display_order = 96` unique dans `derived`). H5 le déclare avec commentaire F6 « sous-ensemble
  par capability-group » (serving gaté par la capability fine).
- [x] E1c — Vérifié sur pièces : le score VALEUR circule via `player_match_enrichment.engagement_score`
  (persist, `rows.go`/`player_persister.go`) et l'API engagement (numérique brut, JSON key, PAS
  de libellé localisé). Le SEUL surface Go qui sert un LIBELLÉ de champ est
  `GET /titles/{slug}/field-mappings` (`handlers/field_mappings.go`), qui itère `set.All()`
  génériquement → le label FR/EN de `engagement_score` est désormais servi automatiquement via
  `FieldMappingSet` (branchement data-driven, aucun code à modifier). Rien d'autre ne sert de
  libellé engagement côté Go.
- Gate E1 : `go test ./internal/games/... ./internal/analysis/...` vert ; test de parité
  fields.toml vert ; ratchet anti-slug vert ; score Infinite inchangé (goldens).

### E2 — Vecteur de signaux + masque de présence (le cœur agnostic)
- [x] E2a — CARTO faite (consignée §J). Inputs ACTUELS de `ComputeEngagementScore` :
  PlayerEvents/TeamEvents/LobbyEvents (`canonical.HighlightEvent`), NTeam/NHumansLobby,
  MatchStart/End, History, CoefLobbyShare+HasGlobalLobbyCoef, ResponseBins (bins d'intensité
  post-refonte), PersonalScore/Kills/Assists, Mode/IsTeamMode, Window/Sampling. **Constat
  clé** : les signaux riches H5 (impulses objectif) sont DÉJÀ projetés en amont dans
  `highlight_events` comme `event_type="mode"` (poids 1.5) par l'ingest title-owned
  (`games/halo_5/ingest/objective_impulses.go`) et DÉJÀ consommés par la courbe — le
  **vecteur d'events EST le vecteur de signaux universel**, le compute est déjà agnostic
  (confirme mémoire 2026-07-04). 2 points de construction de l'input :
  `service/…::buildInputForMatch` + `sync/engagement.go::batchComputeEngagementScores`.
- [x] E2b — `internal/analysis/temporal/engagement_signals.go` (nouveau) : `EngagementSignals`
  = vecteur (ensemble minimal `HasTimedPlayerEvents`/`HasLobbyPace`/`DurationMS` + signaux
  riches optionnels `*int` `ObjectiveEvents`/`RichKillMechanics` = masque de présence) +
  `SignalSufficiency` (Insufficient/Partial/Full) + `Sufficiency()` + `SignalsFromEvents`
  (dérivation title-AGNOSTIC depuis la composition des events). 0 référence à un titre.
- [x] E2c — `EngagementScoreInput` gagne `Signals EngagementSignals` ; `ComputeEngagementScore`
  dérive le vecteur effectif (fourni par l'appelant, ou dérivé des inputs si `IsZero`) et
  expose `SignalBasis` (= `Sufficiency().String()`) sur le résultat. Les signaux riches ne
  modifient PAS le score (poids nul, DE-5) — prouvé par test byte-identical.
- [x] E2d — Câblage aux 2 points de construction (service + sync) via
  `temporal.SignalsFromEvents(playerEvents, lobbyEvents, durationMS)`. La construction reste
  dans le chemin agnostic partagé (comme TOUT le sous-système engagement) — voir §Découvertes :
  le title-owned est UPSTREAM (l'ingest qui projette les events du titre), le dériveur est
  agnostic (0 modif du moteur pour un titre futur = critère de succès respecté).
- Gate E2 : goldens Infinite inchangés ; tests unitaires `EngagementSignals` (suffisance
  3 niveaux, signaux absents = poids nul) ; build+vet ; `-tags=integration -p 1
  ./internal/sync/...` vert.

### E3 — Double porte de dégradation (suffisance + calibration)
- [x] E3a — Porte statique documentée dans les headers `capabilities.toml` des 2 titres
  (mapping engagement.score → calibration : supported=validated, degraded=provisional+badge,
  not_exposed=503). Infinite reste `supported` ; H5 reste `not_exposed` (bascule E5).
- [x] E3b — Porte dynamique par match : `EngagementScoreResult` gagne `calibration`
  (validated/provisional) en plus de `signal_basis` (E2). `signal_basis` = `Sufficiency()`
  (compute) ; `calibration` = statut de calibration du titre, injecté au service via
  `WithEngagementCapability(status)` résolu par la factory (title-aware, `titleResolver.Data(slug)
  .Capabilities()[CapEngagement]`, nil-safe). openapi.yaml + `generated.ts` régénérés
  (openapi-typescript ; `make generate-types` est un stub — noté §Découvertes).
- [x] E3c — Règle de service : `GetMatchEngagement` retourne `games.ErrCapabilityNotSupported`
  si fine=`not_exposed` → handler mappe via `MapCapabilityError` → 503 `capability_not_supported`
  (pattern B15, jamais 500 ni score faux) ; `degraded` → score servi AVEC `calibration=provisional` ;
  `supported`/vide → validated. Tests handlers (mock port) sur les 3 statuts ajoutés.
- Gate E3 : tests service (mock port) sur les 3 statuts ; drift openapi vert ; front
  typecheck (types régénérés).

### E4 — Harnais de calibration par titre (outil, pas de magie)
- [x] E4a — CLI `cmd/engagement-calibrate` (`//go:build cgo`) : énumère les player DBs d'un
  titre, agrège les `RatioSample` (paces persistées) par mode, calcule les distributions par
  bin d'intensité via la MÊME logique que le serving (`ComputeEngagementResponseBins` +
  `ComputeEngagementCoefficient`), compare à la référence Infinite, écrit un rapport markdown
  + le bloc TOML candidat. Méthode EXPLICABLE documentée (score = percentile intra-personnel
  invariant d'échelle → levier = poids d'events ; le rapport juge si la dispersion/rejet du
  titre est comparable à Infinite). N'applique RIEN.
- [x] E4b — Config par titre = `constants.toml [engagement]` (pattern damage_model, cf.
  §Découvertes : le repo met les constantes par-titre dans constants.toml, pas un fichier
  séparé) : poids objective/assist/death/default. Loader (`mappings.EngagementConstants` +
  `loader_endpoints.go`) + accessor `games.EngagementWeightsFor(slug)` → `temporal.EventWeights`,
  fallback `DefaultEventWeights` (byte-identique) si section absente. Threadé dans le compute
  (`EngagementScoreInput.Weights` → courbe) + les 2 points de collecte (service + sync).
  Infinite = valeurs ACTUELLES (1.5/0.5/0.0/1.0), byte-identique prouvé (test temporal
  `ExplicitDefaultWeightsByteIdentical` + suite temporal + intégration sync verts).
- [x] E4c — Harnais exécuté sur H5 (données LOCALES, 4 joueurs, 5240 samples) →
  `.ai/ENGAGEMENT_CALIBRATION_H5_2026-07-11.md` commité. Résultat : bins décroissants
  calme→chaotique (ranked 1.043→0.916, unranked 1.011→0.879) cohérents avec Infinite, coef
  global 0.95-0.97, rejets faibles (ranked 0.5 %, unranked 8.5 %). Candidats H5 = poids de
  référence Infinite (dans `halo_5/constants.toml [engagement]`, provisoires jusqu'à E6).
- Gate E4 : goldens Infinite inchangés après migration de config ; rapport H5 produit ;
  `go test ./internal/analysis/temporal/...` vert.

### E5 — Activation H5 en `degraded`
- [x] E5a — H5 `engagement.score` passé à `degraded` dans les 3 miroirs : `capabilities.toml`
  halo_5, adapter `fallbackCapabilities()` (`games.CapDegraded`), et le parity test
  `skeleton_test.go` (TestHalo5_FineCapabilities). Coarse `title.CapEngagement` déjà présent
  (title.toml) → route servie. Commentaires : calibration provisoire E4, gate humain E6.
- [x] E5b — Pipeline H5 vérifié sur pièces : coarse cap présent (route sert) ; fine=degraded →
  service (E3) mappe `calibration=provisional` (test handler) ; feature matrix H5 =
  `degraded` (cascade `CapDegraded`→`StatusDegraded`) → front rend avec badge ; 5240 paces H5
  persistées (harnais E4c les a lues). `-tags=integration -p 1 ./internal/api/...` exit 0.
- [x] E5c — Front : mention discrète « calibration provisoire » / « provisional calibration »
  (manifest `engagement.calibration.provisional` FR+EN, régénéré) apposée au sous-titre du
  graphe quand `calibration === 'provisional'`, masquée sinon (DE-8). Logique extraite dans
  `engagementSubtitle.ts` (+ champ `calibration`/`signal_basis` sur `EngagementScoreResultAPI`).
  Test `EngagementMatchSection.test.tsx` (5 cas). typecheck 0, eslint 0, vitest verts.
- [x] E5d — Backfill H5 LOCAL : les coefficients E4 (poids) = défauts Infinite, byte-identiques
  au re-backfill de la refonte lobby qui a produit les 5240 samples locaux → un recompute est un
  no-op numérique (données locales déjà conformes E4 ; harnais E4c les a lues et validées). [!]
  PROD : re-backfill différé à la fenêtre post-merge (dépend du deploy — même différé que la
  refonte lobby ; vérification utilisateur requise).
- Gate E5 : `-tags=integration -p 1 ./internal/sync/... ./internal/api/...` vert ; front
  typecheck+vitest verts ; smoke visuel H5 (courbe + profil) par l'utilisateur.

### E6 — GATE HUMAIN de calibration puis `supported`
- [x] E6a — Protocole de validation écrit (ci-dessous, §Protocole gate humain E6).
- [!] E6b — DÉCISION UTILISATEUR (non automatisable). En attente : l'utilisateur juge les
  scores H5 sur ses parties selon le protocole E6a. Si validé → `engagement.score` H5 passe
  à `supported` (3 miroirs) + retrait du badge provisoire (via `calibration=validated`
  automatique) + MAJ mémoire ; si non validé → itérer les poids `halo_5/constants.toml
  [engagement]` (E4c) et re-soumettre. État courant figé à `degraded`/provisional (sûr : score
  servi avec mention discrète, jamais faussement présenté comme validé).
- [x] Gate E6 — capabilities cohérentes : **test miroir coarse↔fine livré**
  (`internal/games/engagement_capability_mirror_test.go`, générique tous titres — ferme le
  reliquat F15-12/L2-(3) pour cette capability), vert. Décision utilisateur E6b : [!] en attente.

### Protocole gate humain E6 (E6a)

Objectif : juger si le score d'engagement H5 « a du sens » avant de passer `supported`.
L'utilisateur, sur SES parties H5 (page Match View → onglet Équipe → courbe engagement) :
1. **Match intense gagné en dominant** (chaotique, victoire nette) : le score doit être
   plutôt haut / la courbe « Joueur réel » au-dessus de « Joueur attendu ».
2. **Match intense subi / farmé** (chaotique, défaite) : score plutôt bas, « réel » sous
   « attendu » — le modèle ne doit PAS gonfler l'attendu par l'intensité (bin chaotique →
   coef plus bas, cf. rapport E4c : ranked chaotique 0.916 < calme 1.043).
3. **Match calme** (bin calme) : l'attendu suit une réponse habituelle basse, l'écart réel−
   attendu reste lisible.
4. **Cohérence forme du jour** : deux matchs de MÊME intensité mais l'un « en forme »,
   l'autre « en dedans » doivent produire des scores nettement distincts.
5. La mention « calibration provisoire » doit s'afficher (H5 degraded) et disparaître si un
   jour H5 passe `supported`.
Verdict attendu : « les scores distinguent bien forme du jour et intensité, sans absurdité »
→ valider (E6b). Sinon, noter les cas faux → ajuster les poids H5 (E4) → re-tester.

## 4. Garde-rails à livrer avec (règle 6 — pas de factorisation sans ratchet)
- [x] Test « le moteur temporal n'importe aucun package games/titre » —
  `internal/archlint/no_temporal_title_import_test.go` (seul `games/canonical` toléré). Vert.
- [x] Test miroir coarse↔fine engagement (générique tous titres) —
  `internal/games/engagement_capability_mirror_test.go`. Ferme le reliquat F15-12/L2-(3) pour
  cette capability. Vert.
- [~] Goldens engagement par titre : couvert PAR COMPOSITION (le compute est title-agnostic).
  Infinite = test byte-identical `ExplicitDefaultWeightsByteIdentical` + suite temporal (algo
  agnostic sur events canoniques). H5 = tests `games/halo_5/ingest` (synthèse events
  killer_victim_pairs + impulses objectif→`mode`) + rapport empirique E4c
  (`.ai/ENGAGEMENT_CALIBRATION_H5_2026-07-11.md`). Créer un fixture golden H5 dédié
  re-testerait l'algo agnostic avec des events canoniques H5-shaped = redondant (gold-plating).
  M5 clos pour ce sous-système par cette composition.

## 5. Hors périmètre (NE PAS traiter ici)
- Refonte lobby elle-même (plan dédié, P0).
- Tout autre metric canonique de 1er ordre.
- Activation d'autres capabilities H5 (leaderboard mondial, CSR live…).
- Halo 7 / titres futurs (le chantier les REND possibles, il ne les câble pas).

## 6. Journal §J (à remplir par l'exécutant à chaque clôture de phase)
- **E1 — COMPLÉTÉE (2026-07-11)**. `engagement_score` promu FieldKey canonique de 1er ordre
  (groupe `derived`, bornes [0,100]) + sections `fields.toml` des 2 titres + count/golden MAJ.
  E1c : libellé servi data-driven par `field_mappings.go` (générique `set.All()`), aucun
  branchement Go à ajouter. Gate : `go test ./internal/games/... ./internal/analysis/...`
  ALL GREEN ; parité fields (loader smoke réel) verte ; ratchet anti-slug (`archlint`) vert ;
  golden FieldKey vert ; `go vet` 0.
- **E2 — COMPLÉTÉE (2026-07-11)**. Vecteur de signaux `EngagementSignals` + `Sufficiency()`
  (3 niveaux) + `SignalsFromEvents` (dérivation agnostic) ; `EngagementScoreInput.Signals`
  consommé par le compute → `SignalBasis` sur le résultat ; câblage aux 2 points de
  construction (service + sync). Gate : temporal (unit + byte-identical) vert ; `go build ./...`
  clean ; packages touchés (analysis/service/sync/domain/api-handlers/persist) verts ; api
  (drift report-only, additif = divergent non gaté) vert ; `go vet` 0 ;
  `-tags=integration -p 1 ./internal/sync/...` exit 0 ; lint delta `--new-from-rev=3b0195df2` 0.
- **E6 — E6a + garde-rails LIVRÉS (2026-07-11) ; E6b [!] décision utilisateur en attente.**
  Protocole de gate humain écrit (§Protocole gate humain E6). Garde-rails §4 livrés : archlint
  temporal-agnostic + miroir coarse↔fine (tous titres). H5 reste `degraded`/provisional (sûr).
  Passage `supported` = décision E6b (non automatisable). Gate : miroir vert ; décision [!].
- **E5 — COMPLÉTÉE en LOCAL (2026-07-11)**. H5 `engagement.score` → `degraded` (3 miroirs) ;
  service mappe `calibration=provisional` ; front badge « calibration provisoire » FR+EN
  (logique extraite `engagementSubtitle.ts` + test). Gate : H5/games verts ; intégration api
  `-p 1` exit 0 ; front typecheck 0 / eslint 0 / vitest verts. Reste : re-backfill PROD
  (post-merge) + smoke visuel H5 par l'utilisateur (gate E5 non automatisable).
- **E4 — COMPLÉTÉE (2026-07-11)**. Coefficients par titre (poids d'events) externalisés dans
  `constants.toml [engagement]` + loader + accessor `games.EngagementWeightsFor` ; threadés
  dans le compute (byte-identique Infinite) et les 2 collecteurs. Harnais `cmd/engagement-calibrate`
  + rapport H5. Gate : temporal (dont byte-identical) vert ; build ./... clean ; intégration
  sync `-p 1` exit 0 ; lint delta 0 ; rapport H5 produit.
- **E3 — COMPLÉTÉE (2026-07-11)**. Double porte : capability fine documentée (E3a) ;
  `calibration` (validated/provisional) + `signal_basis` dans le contrat, injectés au service
  via `WithEngagementCapability` résolu title-aware par la factory (E3b) ; `not_exposed` → 503
  `capability_not_supported` via `MapCapabilityError`, `degraded` → provisional (E3c). Gate :
  api (drift ↔ Huma réconcilié, guards Huma, 3-statuts) vert ; service vert ; front typecheck 0
  (generated.ts régénéré) ; lint delta 0.

## 7. Découvertes hors périmètre (à consigner, NE PAS traiter)
- **Config par titre = `constants.toml [engagement]`, pas un `engagement.toml` séparé** (E4).
  Le plan (E4b) nommait `config/titles/{slug}/engagement.toml`. Vérifié sur pièces : le
  précédent `damage_model` (que le plan cite comme pattern) vit dans `constants.toml [damage_model]`,
  chargé par `mappings.LoadEndpointsFromFile` → resolver boot → `games.EffectiveHpToKill(slug)`.
  J'ai suivi CE pattern (section `[engagement]` dans `constants.toml`, même loader/resolver/accessor)
  plutôt qu'un fichier + loader séparés : réutilise l'infra, un fichier de moins, cohérent repo.
  Aucune fonctionnalité perdue.
- **H5 PvP_unranked : taux de rejet 8.5 %** (E4c, > seuil indicatif 5 % de la refonte lobby pour
  Infinite). Bins bien peuplés (n=538-540) et coef global exploitable → non bloquant, mais à
  regarder au gate humain E6 (matchs unranked H5 plus bruités). Non traité (diagnostic seul).
- **`make generate-types` est un stub** (E3). La cible Makefile `generate-types` (l.44-46) ne
  fait que vérifier `npx` et logger « Types générés » — elle N'EXÉCUTE PAS openapi-typescript.
  La vraie génération est le script npm `apps/web` : `npm run generate-types` (openapi-typescript
  openapi.yaml → generated.ts). Utilisé ici. La cible Makefile devrait déléguer au script npm
  (dette doc/outillage — non traitée, hors périmètre).
- **Construction des signaux : agnostic partagé, pas de builder per-titre** (E2). Le plan
  (E2d) supposait un point de construction « title-owned (`games/{slug}/…`) ». Vérifié sur
  pièces : TOUT le sous-système engagement (input, compute, recompute) est déjà agnostic —
  la spécificité titre vit UPSTREAM dans l'ingest qui projette les events du titre dans
  `highlight_events` (H5 : `games/halo_5/ingest/objective_impulses.go` mappe les impulses
  objectif en `event_type="mode"`). Forcer un builder engagement per-titre forkerait le
  compte agnostic sans bénéfice et risquerait la byte-identité Infinite. `SignalsFromEvents`
  est donc un helper agnostic dans `analysis/temporal` (pur). Le critère de succès « un titre
  futur s'active sans modifier le moteur temporal » reste satisfait : il fournit son ingest
  (title-owned) + ses coefficients (E4). Aucun périmètre changé, non traité au-delà.
