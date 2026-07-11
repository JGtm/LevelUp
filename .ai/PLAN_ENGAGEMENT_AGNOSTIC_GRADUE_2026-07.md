# PLAN — Engagement title-agnostic gradué + activation H5 (chantier F7)

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
- [ ] E3a — Porte statique (par titre) : la capability fine `engagement.score` du
  `capabilities.toml` reflète ce que le titre PEUT fournir : `not_exposed` (signaux
  minimaux impossibles), `degraded` (signaux OK, calibration non validée), `supported`.
  Documenter la sémantique dans le header du TOML.
- [ ] E3b — Porte dynamique (par match) : la réponse engagement gagne un champ de
  confiance (`signal_basis` : full/partial + calibration : provisional/validated) dérivé
  de `Sufficiency()` + du statut de calibration du titre. Openapi + `make generate-types`.
- [ ] E3c — Règle de service : capability `not_exposed` → 204/section absente (comme
  aujourd'hui) ; `degraded` → score servi AVEC le champ de confiance ; jamais d'erreur 500.
  Réutiliser `MapCapabilityError`/les patterns B15.
- Gate E3 : tests service (mock port) sur les 3 statuts ; drift openapi vert ; front
  typecheck (types régénérés).

### E4 — Harnais de calibration par titre (outil, pas de magie)
- [ ] E4a — CLI `cmd/engagement-calibrate` : sur les données d'un titre (copie locale ou
  dev), calcule les distributions des composantes du score (par bin d'intensité post-
  refonte), propose des coefficients par titre (méthode simple et EXPLICABLE : normalisation
  des échelles par percentiles vs référence Infinite — pas de ML opaque), écrit un rapport
  chiffré + le fichier de config candidat. NE l'applique PAS lui-même.
- [ ] E4b — Format de config coefficients par titre : `config/titles/{slug}/engagement.toml`
  (pattern damage_model/constants.toml) + loader + fallback documenté (titre sans fichier =
  cold-start actuel). Infinite y migre ses valeurs ACTUELLES (byte-identique, prouvé golden).
- [ ] E4c — Exécuter le harnais sur H5 (données dev/copie prod) → rapport + coefficients
  candidats H5 commités avec le rapport dans `.ai/ENGAGEMENT_CALIBRATION_H5_<date>.md`.
- Gate E4 : goldens Infinite inchangés après migration de config ; rapport H5 produit ;
  `go test ./internal/analysis/temporal/...` vert.

### E5 — Activation H5 en `degraded`
- [ ] E5a — Adapter H5 : `CapEngagement` passe de `CapNotExposed` à la surface réelle ;
  `capabilities.toml` halo_5 : `engagement.score = degraded` (commentaire : calibration
  provisoire E4, validation humaine en attente).
- [ ] E5b — Vérifier le pipeline complet H5 sur pièces : le sync H5 calcule et persiste
  déjà l'enrichment engagement (mémoire) — VALIDER sur données réelles (dev) que les
  scores H5 sortent avec les coefficients E4, et que les surfaces (courbe engagement,
  profil) rendent le champ de confiance.
- [ ] E5c — Front : badge/mention « calibration provisoire » (FR+EN, i18n manifest) sur
  les surfaces engagement quand `degraded`/provisional (DE-8) ; masqué quand validated.
- [ ] E5d — Backfill H5 : recompute engagement des matchs H5 existants avec les
  coefficients E4 (chemin recompute existant post-refonte-lobby, append-only). Dev
  d'abord ; prod = fenêtre convenue.
- Gate E5 : `-tags=integration -p 1 ./internal/sync/... ./internal/api/...` vert ; front
  typecheck+vitest verts ; smoke visuel H5 (courbe + profil) par l'utilisateur.

### E6 — GATE HUMAIN de calibration puis `supported`
- [ ] E6a — Protocole de validation écrit (10 lignes) : quels matchs H5 regarder
  (intenses vs calmes, victoires nettes vs serrées), à quoi un score « qui a du sens »
  ressemble. L'utilisateur juge sur ses propres parties (non automatisable — mémoire).
- [ ] E6b — Si validé : `engagement.score` H5 → `supported`, retrait du badge provisoire,
  MAJ mémoire projet + ce plan. Si non validé : itération coefficients (E4c) — le plan
  reste en E6, consigner les constats.
- Gate E6 : décision utilisateur consignée ; capabilities cohérentes (test miroir
  coarse↔fine vert — le livrer ici s'il n'existe pas encore, cf. reliquat L2-(3)).

## 4. Garde-rails à livrer avec (règle 6 — pas de factorisation sans ratchet)
- Test « le moteur temporal n'importe aucun package games/titre » (archlint, calque
  no_duckdb_import).
- Test miroir coarse↔fine engagement (générique tous titres — ferme le reliquat F15-12/L2-(3)
  pour cette capability).
- Goldens engagement par titre (fixtures Infinite existantes + fixtures H5 à créer en E5 —
  ferme M5 pour ce sous-système).

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

## 7. Découvertes hors périmètre (à consigner, NE PAS traiter)
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
