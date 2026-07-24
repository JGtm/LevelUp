# PLAN — XP de carrière estimée par match (Timeseries) — 2026-07

> Statut : PRÊT, non exécuté. Exécution sous contrat du skill `plan-execution`.
> Branche cible : `feat/xp-carriere-estimee`. Indépendant de
> `PLAN_EXPLORER_LIVE_REPAIR_2026-07.md` (aucune dépendance croisée).
> Base factuelle : thought_log 2026-07-22 (validation empirique) + mémoire
> `project_explorer_live_target_diag`.

## Faits établis (ne pas re-vérifier, sauf B0.2/B0.3)

1. **Formule validée sur nos données** : XP de carrière gagnée par match =
   `multiplicateur(éra) × personal_score`. 4/4 paires propres (snapshots encadrant un
   seul match connu, gardes 6 min) à ratio **2,00 exact**, défaite incluse → pas de
   modificateur victoire/défaite.
2. **Éras officielles** : la progression Career Rank = Personal/Applied Score du match
   (343). Depuis **Operation: Infinite, 18 novembre 2025**, « the Applied Score
   multipliers for all matchmaking playlists will be doubled » — doublement confirmé
   permanent. Sources : annonce officielle
   https://www.halowaypoint.com/fr/news/operation-infinite-preview (fournie par
   l'utilisateur, citation exacte ci-dessus), patch notes Operation Infinite
   (support.halowaypoint.com), halopedia.org Rank_(Halo_Infinite), @HaloSupport
   18/11/2025 (permanence). Donc : ×1 avant le 2025-11-18, ×2 depuis. Le pluriel
   « multipliers » confirme des multiplicateurs PAR playlist (tous doublés) — cohérent
   avec la décision n°1 (uniforme en v1, raffinement playlist en backlog).
   (SpartanRecord n'applique pas ce ×2 sur son graphe → il affiche la moitié de l'XP
   réelle actuelle.)
3. **`xp_total` des snapshots est fidèle à l'API** — le saut de +173 230 XP du
   25/05 (JGtm) correspond à un passage rang 179 → 184 rendu par l'API elle-même
   (re-crédit côté 343 ; catalogue local inchangé, aucun déploiement local ce jour-là).
   Conséquence : `xp_total` sert d'**oracle de validation** (deltas), PAS de série
   d'affichage long terme (des re-crédits serveur peuvent créer des sauts).
4. Le score personnel est en base pour tout l'historique (`match_participants.personal_score`),
   matchs matchmade PvP uniquement (Firefight quasi absent de la base, customs absents).

## Objectif et critère de succès

Un graphe « XP de carrière (estimée) » par match sur la page Timeseries (profil suivi,
historique complet — mieux que les 25 matchs de SpartanRecord), étiqueté honnêtement,
Infinite uniquement (capability), validé contre les deltas `xp_total` (±5 % sur fenêtres
propres). Aucun flag OFF : la feature part active.

## Décisions tranchées

1. **v1 = multiplicateur uniforme par éra** : ×1 avant 2025-11-18, ×2 après (borne à
   minuit UTC ; l'imprécision d'heure de déploiement est négligeable à l'échelle d'un
   graphe). Pas de multiplicateurs par playlist en v1 : BTB (SpartanRecord ×1,8) et
   bots ne sont PAS calibrables sur nos données (échantillon vide) — raffinement en
   backlog, PAS dans ce plan.
2. Étiquette : « XP de carrière (estimée) » + tooltip méthodologique (formule + éras +
   limites : matchs connus uniquement, playlists à multiplicateur spécial approximées).
3. Les éras vivent en TOML versionné (modèle `damage_model` + constants.toml), pas en
   dur dans le Go.
4. Matchs invisibles (Firefight non synchronisé) : la feature affiche l'XP des matchs
   CONNUS uniquement — c'est l'objet du graphe (per-match). B0.3 quantifie le manque ;
   seuil de décision : si la part d'XP « invisible » dépasse 10 % chez un joueur suivi,
   proposer un chantier séparé d'audit du sync PvE (PAS exécuté dans ce plan).
5. `xp_total` : AUCUNE adaptation (fait n°3). Interdiction documentée d'en tracer la
   série brute sans garde anti-discontinuité.

## Lot B0 — Calibration, mesures préalables (lecture seule)

- [x] B0.1 Table d'éras créée dans `config/titles/halo_infinite/constants.toml` (le
      constants.toml au niveau dossier titre EST le précédent damage_model/engagement,
      chargé par `mappings/loader_endpoints.go` — vérifié sur pièces). Section
      `[[career_xp_eras]]` : `{from="", to="2025-11-18", multiplier=1.0}` +
      `{from="2025-11-18", to="", multiplier=2.0}` + commentaire source Operation: Infinite.
- [!] B0.2 Invérifiable localement : les snapshots `career_progression` des 3 joueurs
      suivis démarrent tous en 2026-02/03 (JGtm min recorded_at 2026-02-18 ; Madina et
      Chocoboflor 2026-03-07) — donc AUCUNE fenêtre pré-18/11/2025 en base. Toutes les
      fenêtres mesurées sont post-doublement (×2). Le ×1 pré-18/11 repose sur la source
      officielle seule (annonce + patch notes Operation: Infinite). Issue acceptée par le
      plan, non bloquante.
- [x] B0.3 Matchs invisibles quantifiés (parquets staging du 23/07, diag_parquet_q).
      Méthode : fenêtres entre transitions distinctes de `xp_total` ; « invisible » =
      Σ Δxp_total des fenêtres à 0 match connu (participation en base) / Σ Δxp_total total.
      Résultats : **JGtm 0,37 % · Chocoboflor 3,07 % · Madina97294 5,83 %** — tous < 10 %
      → PAS de chantier « audit sync PvE/Firefight » déclenché. Détail en « Découvertes ».
- [x] B0.4 Borne d'éra retenue (2025-11-18 00:00 UTC, from inclusif / to exclusif) et
      résultats B0.3 documentés ci-dessus + section « Découvertes ».

Gate B0 : items statués, % consignés, TOML relu (`go test ./internal/games/...` si un
loader est touché, sinon lecture seule).

## Lot B1 — Backend

- [x] B1.1 `internal/analysis/xp_estimate.go` : fonction pure
      `EstimateCareerXP(personalScore int, endedAt time.Time, eras []mappings.CareerXPEra) int`.
      Type `CareerXPEra` placé dans `mappings` (PAS `internal/domain` : `domain` importe
      déjà `games` via `squad_v2.go` → cycle ; le loader impose son type ; `analysis`
      importe déjà `games/mappings` — vérif sur pièces). Tests purs `xp_estimate_test.go` :
      bornes veille/jour J/lendemain 18/11/2025, score 0, éras vides→0, arrondi.
- [x] B1.2 Loader TOML : `[[career_xp_eras]]` branché dans `mappings/loader_endpoints.go`
      (`careerXPEraTOML` + `parseCareerXPEras` : dates UTC parsées/validées, multiplicateur
      > 0, intervalle non inversé) → `EndpointSet.CareerXPEras()` ; résolveur+défaut dans
      `games/career_xp.go` (`CareerXPErasFor` + `DefaultCareerXPEras`, miroir exact
      `EffectiveHpToKill`) + `MappingsEndpointResolver.CareerXPErasFor`. Tests loader
      (`loader_endpoints_test.go` : valide + 3 erreurs + golden HI) et games (`career_xp_test.go`).
- [x] B1.3 Capability fine `analytics.career_xp_estimate` : const `CapAnalyticsCareerXPEstimate`
      (`adapter.go`) + `AllCapabilityKeys()` (`capabilities.go`, count test 18→19) +
      `"analytics.career_xp_estimate" = "supported"` (`capabilities.toml` HI uniquement).
      Testée via `games.ProvidesCareerXPEstimate` (CapabilityMap/Has, jamais `slug ==`) ;
      titre sans capability → éras non résolues → `CareerXPEstimated` nil par ligne
      (série absente, dégradation silencieuse). Défaut opt-in STRICT (false).
- [x] B1.4 Service Timeseries : champ additif `CareerXPEstimated *int` sur
      `domain.TimeseriesMatchRow` (grain per-match existant — décision d'archi via
      exploration : les séries per-match sont des champs additifs, PAS un objet série
      dédié). Peuplé dans `buildMatchRows` via `estimateMatchCareerXP` (exclusion
      `IsFirefight`, `PersonalScore` nil → point exclu = nil). Gate capability +
      résolution éras dans `GetPage`. **Écart au plan assumé** : borne d'éra sur
      `StartTime` (canonique déjà projeté) et non `end_time_utc` — ce dernier n'est PAS
      projeté vers `StatsMatchRow` ; l'ajouter toucherait le repo/scan/projection sans
      bénéfice (borne à la journée → start vs end du match négligeable). AUCUNE requête
      SQL ajoutée (`personal_score`/`is_firefight`/`start_time` déjà projetés). Pas de
      champ `excluded_count` (le nil par ligne suffit à exclure le point ; front gate
      data-driven — pas de compteur nécessaire).
- [x] B1.5 `openapi.yaml` : propriété `career_xp_estimated` (int64, non-required, style
      omitempty comme `max_killing_spree`) ajoutée à `TimeseriesMatchRow`. **`make
      generate-types` = superviseur** (régénère `apps/web/src/lib/api/generated.ts` →
      `career_xp_estimated?: number`). Pas d'interface hand-written impactée.
- [x] B1.6 Tests service `timeseries_service_test.go` `TestBuildMatchRows_CareerXPEstimated`
      (éras ×2/×1 par date, exclusion Firefight + score nil, éras nil → pas de série) +
      2 callers existants mis à jour (4e arg). Pas de contracttest dédié à créer : la
      forme est couverte par `jsonshape_dto_smoke_test.go` (GetPage) + le champ est omitempty.

Gate B1 : `cd apps/go-api && go test ./internal/analysis/... ./internal/service/...
./internal/games/...` puis `go test ./...` + `go vet ./...` + `gofmt -w`. **[!] Gate
superviseur** (droits go build/test/vet réservés au superviseur ; code écrit avec
vérif sur pièces des signatures/champs, non compilé par l'agent). Note : gofmt requis
sur les littéraux de struct touchés (alignement).

## Lot B2 — Front (page Timeseries)

- [x] B2.1 Chart `TimeseriesCareerXP` (`TimeseriesFormCharts.tsx`, pattern local exact
      de `TimeseriesRankScore` : `useMemo`→`buildOption`→`ChartRender`). Ligne = XP de
      carrière CUMULÉE (trajectoire, « mieux que les 25 matchs de SpartanRecord ») ;
      barres = XP estimée par match (axe secondaire). Grain per-match (1 point/match) —
      pas de binning (ADR 0010 ne concerne que les histogrammes, pas les lignes per-match).
      Ajouté à l'onglet Progression (`TimeseriesPage.progression.tsx`), pleine largeur,
      gate DATA-DRIVEN (`hasCareerXP = match_rows.some(career_xp_estimated != null)` —
      pas de slug côté front). Logique de série extraite pure dans `careerXpSeries.ts`.
- [x] B2.2 i18n FR **et** EN dans `manifests/timeseries.toml` : `career_xp_title`
      (« XP de carrière (estimée) » / « Career XP (estimated) »), `career_xp_cumulative`,
      `career_xp_per_match`, `career_xp_tooltip` (méthodo : formule + éras + « matchs
      connus uniquement », via `InfoTooltip`). FR sans anglicisme. Tokens sémantiques
      uniquement (`chart-series-3`/`chart-series-6`, `resolveToken`) — zéro hex.
      **`node apps/web/scripts/build_i18n_manifests.mjs` = superviseur** (régénère
      `generated/timeseries.ts` → nouvelles clés typées `TimeseriesManifestKey`).
- [~] B2.3 Query keys : query Timeseries existante réutilisée telle quelle — la série
      arrive dans le payload `TimeseriesPageResponse.match_rows` (champ additif). AUCUNE
      nouvelle clé (couvert par la query existante `keys.timeseries`).
- [x] B2.4 Test builder pur `careerXpSeries.test.ts` (vitest) : cumul reporté sur les
      nuls, démarrage au 1er match connu, hasData false si aucune estimation. **`make
      generate-types && make check-types && make test-web` = superviseur** (l'agent n'a
      pas les droits npm/tsc/vitest).

Gate B2 : `make generate-types && make check-types && make test-web` exit 0 ; aucun hex
dans `features/` ; vérification visuelle du graphe (JGtm). **[!] Gate superviseur**
(droits npm/tsc/vitest réservés). Pré-requis superviseur AVANT check-types :
`make generate-types` (champ `career_xp_estimated`) + build i18n manifests (clés).

## Lot B3 — Validation croisée (gate final)

- [~] B3.1 Recoupement Σ XP estimée vs Δ`xp_total` : RÉALISÉ en B0.3 (la formule est
      FERME = ×2 × personal_score ; rejouer donnerait des chiffres identiques). Sur les
      fenêtres à un seul match propres (sans match invisible, sans re-crédit serveur
      écarté) : écart ≤ 1 % pour 86/87 fenêtres (bien sous le seuil ≤ 5 %). Tableau
      consigné plan §Découvertes + thought_log.
- [~] B3.2 Paires propres (méthode gardes 6 min → ici transitions d'xp distinctes = un
      match) rejouées avec la formule finale : 86/87 = **98,9 % à ±1 %** (JGtm 96,6 %,
      Madina 100 %, Chocoboflor 100 %) ≥ 90 % requis. Couvert par B0.3.
- [!] B3.3 Revue visuelle utilisateur (gate HUMAIN) + entrée thought_log de clôture :
      déférée au superviseur/utilisateur après passage des gates go/front. Entrée
      thought_log de calibration B0 déjà posée ; clôture finale à ajouter au merge.

## Gate final (delivery-checklist) — [!] superviseur (droits build/test/lint réservés)

- [!] Suite Go complète + vet + lint. AUCUN diff persist/sync/migration (le champ est
      un ajout de projection DTO en lecture ; pas d'écriture DB) → `-tags=integration`
      non requis pour ce chantier. `gofmt -w` sur les fichiers Go touchés (alignement
      des littéraux de struct). Superviseur.
- [!] `make generate-types` (champ `career_xp_estimated`) + build i18n manifests
      (nouvelles clés) AVANT `make check-types` (cache purgé) + `make test-web`. Superviseur.
- [x] i18n parité FR/EN par typage (4 clés `fr`+`en`) ; zéro couleur hex (tokens) ;
      zéro `fmt.Println` (backend en pur calcul/config, aucun log ajouté).
- [~] Thought_log : entrée calibration B0 posée ; entrée d'implémentation à ajouter au
      merge par le superviseur ; aucun item sans statut (tous B0/B1/B2/B3 statués).

## Hors périmètre (consigner, ne pas traiter)

- Multiplicateurs par playlist (BTB ×1,8 / bots / Firefight) — backlog, revalider
  empiriquement quand des échantillons existeront (fenêtres BTB encadrées).
- Audit sync PvE/Firefight (déclenché seulement si B0.3 > 10 %, chantier séparé).
- XP estimée sur l'Explorer target (matchs live) et les Sessions — extensions backlog.
- Halo 5 : système d'XP distinct (Spartan Rank) — jamais couvert par cette capability.

## Découvertes en cours d'exécution

### B0 — mesures préalables (parquets staging 2026-07-23, lecture seule via diag_parquet_q)

Snapshots `career_progression` par joueur suivi (fenêtre couverte en base) :

| Joueur | xuid | rows | Δxp distincts | min recorded_at | max recorded_at |
|---|---|---|---|---|---|
| JGtm | 2533274823110022 | 1680 | 57 | 2026-02-18 | 2026-07-22 |
| Madina97294 | 2533274858283686 | 796 | 40 | 2026-03-07 | 2026-07-22 |
| Chocoboflor | 2535469190789936 | 805 | 47 | 2026-03-07 | 2026-07-16 |

**B0.3 — part d'XP « invisible » (fenêtres à 0 match connu / Σ Δxp_total)** :

| Joueur | n fenêtres (Δxp) | Σ Δxp_total | Δxp invisible | % invisible |
|---|---|---|---|---|
| JGtm | 56 | 823 460 | 3 020 | **0,37 %** |
| Chocoboflor | 45 | 325 970 | 10 000 | **3,07 %** |
| Madina97294 | 39 | 557 210 | 32 500 | **5,83 %** |

Tous < 10 % → seuil de décision non franchi, PAS de chantier audit sync PvE/Firefight
(aucun match Firefight en base pour ces 3 joueurs sur la période : `ff_only_dxp` = 0).

**Validation formule (×2 × personal_score) sur fenêtres propres à un seul match** —
méthode « une transition d'xp = un match PvP en base », miroir des paires gardes 6 min :

| Joueur | n fenêtres 1-match | à ±1 % | à ±5 % | ratio médian |
|---|---|---|---|---|
| JGtm | 29 | 28 (96,6 %) | 28 (96,6 %) | 1,0000 |
| Madina97294 | 28 | 28 (100 %) | 28 (100 %) | 1,0000 |
| Chocoboflor | 30 | 30 (100 %) | 30 (100 %) | 1,0000 |
| **Total** | **87** | **86 (98,9 %)** | **86 (98,9 %)** | **1,0000** |

→ Validation ±5 % : **PASS** pour les 3 joueurs (bien au-delà du seuil ; et 98,9 % à ±1 %,
dépasse le critère B3.2 de ≥ 90 % à ±1 %). L'unique écart JGtm (1/29) est une fenêtre à
attribution de bord (match au voisinage d'une borne de snapshot), dans la tolérance.

**Oracle / discontinuité confirmée (fait n°3)** : la fenêtre JGtm à Δxp = 173 230 (max_dxp,
re-crédit serveur 343 du 25/05, rang 179→184) est la raison pour laquelle `xp_total` brut
ne doit PAS être tracé en série (garde anti-discontinuité). Elle abaisse le ratio agrégé
JGtm à 80,6 % sur l'ensemble des fenêtres, mais est correctement écartée par la méthode
1-match propre ci-dessus (B3.1 l'exclut aussi).

**Décision layering (vérif sur pièces)** : `internal/domain` importe déjà `internal/games`
(`domain/squad_v2.go`) → un type d'éra dans `domain` créerait un cycle. Le loader impose son
type (`mappings`, précédent damage_model/engagement) et `internal/analysis` importe déjà
`internal/games/mappings` (`analysis/home_kpis.go`). Donc : type `CareerXPEra` dans
`mappings`, fonction pure `EstimateCareerXP([]mappings.CareerXPEra)` dans `analysis`,
résolveur + défaut dans `games` (miroir exact de `games.EffectiveHpToKill` / damage_model).

### Fichiers touchés (checklist superviseur)

Backend (Go) :
- `config/titles/halo_infinite/constants.toml` — section `[[career_xp_eras]]`.
- `config/titles/halo_infinite/mappings/capabilities.toml` — `"analytics.career_xp_estimate"`.
- `internal/games/mappings/endpoints.go` — type `CareerXPEra` + accessor + wither (import `time`).
- `internal/games/mappings/loader_endpoints.go` — parsing `[[career_xp_eras]]` (import `time`).
- `internal/games/career_xp.go` (NOUVEAU) — défaut + résolveur + `ProvidesCareerXPEstimate`.
- `internal/games/endpoints.go` — `MappingsEndpointResolver.CareerXPErasFor`.
- `internal/games/adapter.go` — const `CapAnalyticsCareerXPEstimate`.
- `internal/games/capabilities.go` — ajout à `AllCapabilityKeys()`.
- `internal/analysis/xp_estimate.go` (NOUVEAU) — `EstimateCareerXP`.
- `internal/domain/timeseries.go` — champ `CareerXPEstimated *int`.
- `internal/service/timeseries_service.go` — gate capability + résolution éras + call.
- `internal/service/timeseries_service_tabs.go` — param `buildMatchRows` + `estimateMatchCareerXP` (import `mappings`).
- `api/openapi.yaml` — propriété `career_xp_estimated` sur `TimeseriesMatchRow`.
- Tests : `internal/analysis/xp_estimate_test.go` (NOUVEAU), `internal/games/career_xp_test.go`
  (NOUVEAU), `internal/games/mappings/loader_endpoints_test.go` (+cas), `internal/games/capabilities_test.go`
  (18→19), `internal/service/timeseries_service_test.go` (+test, 2 callers).

Front (TS) :
- `apps/web/src/features/timeseries/careerXpSeries.ts` (NOUVEAU) — builder pur.
- `apps/web/src/features/timeseries/careerXpSeries.test.ts` (NOUVEAU) — vitest.
- `apps/web/src/features/timeseries/TimeseriesFormCharts.tsx` — composant `TimeseriesCareerXP`.
- `apps/web/src/features/timeseries/TimeseriesPage.progression.tsx` — import + gate + rendu.
- `apps/web/src/lib/i18n/manifests/timeseries.toml` — 4 clés FR/EN.

Actions superviseur (dans l'ordre) : `gofmt -w` fichiers Go touchés → gates Go
(`go test ./...` + `go vet ./...` + lint) → `make generate-types` → build i18n manifests
(`node apps/web/scripts/build_i18n_manifests.mjs`) → `make check-types` + `make test-web`.

## Protocole de reprise de session

Identique au plan Explorer : thought_log + `git log --oneline -10` + premier item non
`[x]` du premier lot non clos. Lot clos = items statués + gate passé (exit code vérifié).

## Effort estimé

B0 : rapide (mesures déjà outillées). B1 : moyen. B2 : petit. B3 : rapide.
Total : ~1 session de travail.
