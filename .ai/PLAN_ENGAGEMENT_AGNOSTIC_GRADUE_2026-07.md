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
- [ ] E1a — `internal/games/canonical/fields.go` : FieldKey `engagement_score` (+ unité,
  bornes) — suivre la recette du skill `canonical-types` (FieldKey + TOML dans le MÊME commit).
- [ ] E1b — `config/titles/halo_infinite/mappings/fields.toml` + `halo_5/mappings/fields.toml` :
  section `engagement_score` (labels FR/EN, format, display_order). H5 le déclare (il l'expose
  à terme) — cohérent avec la règle F6 « sous-ensemble par capability-group ».
- [ ] E1c — Vérifier sur pièces où le score circule aujourd'hui HORS canonical
  (`player_match_enrichment.engagement_score`, service engagement, API) et brancher la
  lecture via le FieldKey (labels via `FieldMappingSet.Get`) là où un label est servi.
- Gate E1 : `go test ./internal/games/... ./internal/analysis/...` vert ; test de parité
  fields.toml vert ; ratchet anti-slug vert ; score Infinite inchangé (goldens).

### E2 — Vecteur de signaux + masque de présence (le cœur agnostic)
- [ ] E2a — CARTO sur pièces d'abord (30-60 min, consignée au Journal) : inventorier les
  inputs ACTUELS de `temporal.ComputeEngagementScore`/`ComputeEngagementCoefficient`
  (post-refonte lobby : pace joueur/équipe/lobby, bins d'intensité, damage model, deaths
  annotées…) + les signaux H5 DISPONIBLES non consommés (events carnage : kill mechanics,
  impulses, objectifs — cf. mémoires H5 events) + ce que Infinite expose d'équivalent.
- [ ] E2b — `internal/analysis/temporal/engagement_signals.go` (nouveau) : struct
  `EngagementSignals` = vecteur typé de signaux optionnels (pointeurs/`Valid bool`) +
  masque de présence + méthode `Sufficiency() SignalSufficiency` (enum : Insufficient /
  Partial / Full) selon l'ensemble minimal (à définir en E2a : au minimum pace + durée +
  frags/morts datés). AUCUNE référence à un titre dans ce package (analysis pur).
- [ ] E2c — `ComputeEngagementScore` consomme `EngagementSignals` (signature étendue ou
  struct d'input enrichie — suivre le style de la refonte lobby `EngagementScoreInput`).
  Les signaux absents ne pèsent PAS (poids nul, pas de valeur par défaut déguisée).
- [ ] E2d — Câblage adapters : côté collecte (sync/enrichment), chaque titre construit
  ses `EngagementSignals` — Infinite = mapping des inputs existants (byte-identique),
  H5 = mapping des inputs existants + signaux riches marqués présents quand disponibles.
  Le POINT de construction est title-owned (`games/{slug}/…` ou le livesync runner),
  jamais le moteur.
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
- (vide au démarrage)

## 7. Découvertes hors périmètre (à consigner, NE PAS traiter)
- (vide au démarrage)
