# PLAN — Axe « Objectifs » du profil de participation : index par opportunité (stats v7.2)

> Statut : ACTIF. Branche : `fix/quick-wins-post-v721`. Contrat : skill `plan-execution`
> (ordre strict, aucun report d'étape exécutable, statuts `[x]`/`[~]`/`[!]` obligatoires,
> zéro fix hors périmètre — consigner en « Découvertes »).
> Origine : demande utilisateur 2026-07-26 (« Ajouter les nouvelles stats d'objectifs de la
> v7.2 au profil de participations pour l'axe Objectifs (radar et barre), attention à la
> pondération et la calibration »). Diagnostic complet : session 2026-07-26 (agent diag).

## Objectif et critère de succès

L'axe « Objectifs » des 4 surfaces vivantes (barres Session Compare/Detail, radar synergie
Escouade, radar Match View, radar profil Ascension) est alimenté par un **index par
opportunité** construit sur `match_objective_stats_latest` (41 colonnes v7.2/v7.2.1), au
lieu de la somme PSA diluée par le nombre total de matchs. Succès = (a) un joueur dont les
matchs à objectif sont bons n'est plus écrasé par ses matchs Slayer ; (b) l'axe disparaît
proprement quand aucun match à objectif (au lieu d'afficher 0) ; (c) constantes de
calibration MESURÉES (P80 réels) pour les familles à volume suffisant ; (d) gates verts.

## Décisions produit TRANCHÉES (ne pas rouvrir en cours d'exécution)

1. **Remplacement, pas complément** : l'axe Objectif abandonne `Σ PSA objective` comme
   source. Le chargement PSA est CONSERVÉ uniquement pour le résiduel de l'axe Score
   (`match_view_radar.go` ~:273 : `ps − kills×100 − assists×50 − objectiveScore`) — ne pas
   le casser.
2. **Squad V2 (`charts.radar`, payload mort sans consommateur front)** : NE PAS toucher.
   Consigner en Découvertes (code mort + 2 clés i18n orphelines) pour un lot de nettoyage.
3. **Sémantique de l'index** (voir formules) : sous-score par famille sur les seuls matchs
   de la famille ; agrégation pondérée par n_f ; idiome intensif P80×1.25 (un joueur P80
   marque 80/100). Familles : `ctf`, `zones_koth`, `zones_strongholds` (split par
   `zone_scoring_ticks > 0` au niveau ligne), `oddball`, `stockpile`, `extraction`, `vip`.
4. **Axe retiré si n_obj == 0** (patron `dropUncomputableRadarAxes` / `skipSurvivalAxis`),
   et retiré pour un titre sans capability objectifs (H5) via
   `HasCapability`/CapabilityKey — JAMAIS `slug == "..."` (ratchet no_slug_comparison).
5. **Pas de changement de schéma API** : `raw` reste une scalaire (l'index brut, ratio
   ~[0,1.25]), `Value` reste 0..100. Aucun openapi-gen. Tooltips/glossaire réécrits.
6. **Calibration** : P80 mesurés localement (diag_q) pour ctf / zones_koth /
   zones_strongholds / oddball. Stockpile/extraction/vip : backfill local ciblé d'abord
   (étape 1) ; si le backfill échoue (auth/lease), constantes PROVISOIRES documentées
   (`provisional`, critère de re-mesure post-backfill prod) — fallback défini, pas un
   report libre.
7. **Poids `w` des actions** : repris 1:1 de `config/titles/halo_infinite/mappings/awards.toml`
   quand l'équivalent existe (flag_captured 5.0, zone_captured 3.0, power_seed_secured 3.0,
   extraction_converted/completed 3.0, flag_stolen 2.0, zone_secured 2.0, extraction_denied
   2.0, flag_returned 1.5, ball_taken 1.5, extraction_initiated 1.5…). Colonnes sans
   équivalent : flag_secures 1.0, flag_carriers_killed 1.5, flag_returners_killed 1.5,
   zone_defensive_kills 1.0, zone_offensive_kills 1.0, zone_scoring_ticks 0.5 (KOTH
   seulement), skull_carriers_killed 1.5, skull_scoring_ticks 0.5, power_seeds_stolen 2.0,
   power_seed_carriers_killed 1.5, extraction_conversions_denied 2.0, vip_kills 3.0,
   kills_as_vip 2.0 (÷ times_selected_as_vip), vip_assists 1.0.
   ÉCARTÉES (signal nul ou nature inadaptée) : kills_as_flag_carrier, kills_as_skull_carrier,
   kills_as_power_seed_carrier, longest_time_as_*, max_killing_spree_as_vip,
   times_selected_as_vip (dénominateur), extraction_initiations_denied, flag_grabs.
8. **Durées (`hold`)** : time_as_flag_carrier_seconds ; time_in_zones_seconds ;
   time_as_skull_carrier_seconds ; time_as_power_seed_carrier_seconds +
   time_as_power_seed_driver_seconds ; time_as_vip_seconds (÷ times_selected_as_vip).
   Extraction : pas de durée → poids actions = 1.0 (pas de terme hold).

## Formules (référence d'implémentation)

Pour une famille f et un scope de matchs (session, fenêtre, escouade, ou 1 match) :

```
actions_f = Σ_i (w_i × stat_i) / n_f              # cadence pondérée par match de la famille
hold_f    = Σ durées_f / Σ time_played_f          # part du temps de jeu sur l'objectif [0,1]
r_f       = min(1.25, 0.65×actions_f/P80_actions_f + 0.35×hold_f/P80_hold_f)
            # extraction : r_f = min(1.25, actions_f/P80_actions_f)
raw       = Σ_f (n_f × r_f) / n_obj               # n_obj = Σ_f n_f ; si n_obj == 0 → axe retiré
threshold.Objective = 1.25                        # ComputeParticipationProfile fait ×100+clamp
```

P80 = QUANTILE_CONT(0.8) de la distribution PAR MATCH-JOUEUR (population : lignes
`match_objective_stats_latest ⋈ match_participants`, `time_played > 30`, hors bots
`xuid LIKE 'bid(%'`). Commentaire de constante au format du dépôt (n, p50/p80/p90 —
modèle : `internal/analysis/combat_yield.go:49-51`).

## Étapes (ordre strict — gate par étape)

### Étape 0 — Vérifications sur pièces
- [x] Relire les points d'injection cités (les lignes ont pu bouger) :
      `session_compare_participation_helpers.go` ~:122-140,
      `teammates/teammates_squad_charts_synergy.go` ~:28-38 + ~:161-167,
      `match_view_radar.go` ~:218-279 + ~:288-306, `match_view_data_loaders.go` ~:418,
      `progression/profile/queries.go` ~:86-241 + `profile/service.go` ~:203-257.
- [x] Vérifier l'état des colonnes v7.2.1 en base locale (stockpile/extraction/vip
      non-NULL ?) via diag_q.
- Gate : notes écrites des écarts constatés vs diagnostic — cf. « Journal d'exécution »
  (2026-07-26, étape 0) en fin de fichier.

### Étape 1 — Backfill local ciblé v7.2.1 (prérequis calibration 3 nouveaux modes)
- [!] Suivre la procédure du plan V721-02 item 02.6 (`.ai/V7.2.1/PLAN_V721_NOTION_BATCH.md`) :
      reset ciblé `MBitObjectiveStats` sur les matchs des modes Stockpile/Extraction/VIP
      sans ligne `match_objective_stats`, puis `cmd/backfill_objective_stats`.
      NON EXÉCUTABLE dans cette session — échec de lease documenté (journal 2026-07-26,
      étape 1) : la shared DB est tenue RW-exclusive par le serveur dev (PID 29964),
      interdiction de l'arrêter ; le reset ciblé exigerait en outre une écriture ad hoc
      non outillée sur la DB partagée (interdite). → fallback décision 6.
- [x] `[!]`-fallback autorisé UNIQUEMENT si échec auth/lease documenté → constantes
      provisoires (décision 6). APPLIQUÉ : stockpile/extraction/vip en `provisional`,
      critère de re-mesure consigné dans les constantes (étape 2).
- Gate : COUNT(*) par famille dans `match_objective_stats_latest` avant/après, consigné :
  identiques (aucune écriture) — ctf 3579 · zones 2120 · oddball 58 · stockpile 0 ·
  extraction 0 · vip 0.

### Étape 2 — Calibration mesurée
- [x] Requêtes diag_q : distributions actions_f et hold_f par famille (population définie
      ci-dessus), quantiles p50/p80/p90, n par famille. Résultats : journal 2026-07-26
      étape 2.
- [x] Choisir P80 par famille ; familles à n < 30 matchs : constante `provisional` avec
      justification (précédent : paliers citations v7.2.1). Mesurées : ctf, strongholds,
      koth. Provisoires : oddball (7 matchs — valeurs mesurées 57 lignes retenues comme
      estimation), stockpile, extraction, vip (0 ligne, backfill non exécutable étape 1).
- Gate : tableau des quantiles collé dans le rapport + constantes écrites (journal +
  constantes Go étape 3).

### Étape 3 — Primitive pure `internal/analysis/narrative/objective_participation.go` (nouveau)
- [x] Types d'entrée agrégés par famille (n_f, sommes pondérables, durées, time_played,
      times_selected_as_vip), table des poids, constantes P80, `ComputeObjectiveIndex`
      (retourne raw + n_obj). AUCUN I/O. Réutilise `ComputeParticipationProfile` en aval.
      Tables exportées `ObjectiveFamilyActionWeights` / `ObjectiveFamilyHoldColumns`
      (clés = colonnes de la table — source unique pour la couche repo, décision D du
      plan étape 4).
- [x] Tests unitaires purs : cas nominal par famille, familles mixtes, n_obj=0,
      VIP times_selected=0, clamp 1.25, extraction sans hold (+ colonnes inconnues
      ignorées, normalisation 80/100 via ComputeParticipationProfile, cohérence
      poids/hold/calibration 7 familles).
- Gate : `go test ./internal/analysis/...` vert (exit 0, 2026-07-26).

### Étape 4 — Lecture repo (shared)
- [x] `internal/platform/duckdb/objective_index_repo.go` (nouveau fichier, méthodes sur
      `ObjectiveStatsRepo`) : `LoadObjectiveIndexInputs(ctx, matchIDs, xuids)` → par
      xuid × famille : n_f, sommes, durées, Σ time_played (JOIN match_participants,
      filtre population `time_played_seconds > 30` aligné calibration). Split
      KOTH/Strongholds par `zone_scoring_ticks > 0` au niveau ligne. Lecture via vue
      `_latest` UNIQUEMENT. + variante `LoadObjectiveIndexInputsByGamertag` (résolution
      `xuid_aliases`, coéquipiers non suivis — patron WeaponKillsRepo).
- [x] Pas de 6e liste de colonnes : SELECT GÉNÉRÉ depuis
      `narrative.ObjectiveFamilyActionWeights`/`ObjectiveFamilyHoldColumns` (source
      unique, aucun cycle d'import) + test de cohérence clés ⊆ schéma
      (`TestObjectiveIndexRepo_WeightKeysSubsetOfSchema` sur les vraies migrations).
- [x] Tests DuckDB `:memory:` (harnais migrations réelles partagé) : split familles +
      sommes/durées/tp, filtre ≤30 s + scopes, by-gamertag via alias, vue absente →
      dégradation silencieuse, chaîne complète vers ComputeObjectiveIndex.
- Gate : `go test ./internal/platform/duckdb/...` vert (exit 0, 2026-07-26).

### Étape 5 — Injection dans les 4 surfaces
- [x] Session Compare : `buildSessionParticipationProfile` prend
      `narrative.ObjectiveIndexInput` (seuil `ObjectiveIndexThreshold`, axe retiré si
      n_obj=0) ; câblage `WithObjectiveIndexRepo(repo, xuid)` capability-gated
      (`registry_pages.go`). Ancien chemin PSA session SUPPRIMÉ (provider
      `objectiveScoreProvider`, `PlayerMatchesRepo.ObjectiveScores`, délégation adapter
      + test associé — règle 0 code mort).
- [x] Escouade synergie : `loadSynergyObjectiveRaw` par gamertag via
      `LoadObjectiveIndexInputsByGamertag` (source shared → coéquipiers non suivis
      couverts, décision 3 assumée) ; axe retiré de TOUTES les séries si repo absent
      (capability) ou scope sans match à objectif ; `LoadObjectiveScores` SUPPRIMÉ de
      `squadagg.SquadV2Loader` + adapter + fakes ; câblage capability-gated
      (`TeammatesCtx`).
- [x] Match View : raw depuis `me.Obj` (Q12, zéro requête en plus) via
      `objectiveIndexInputFromScoreboard` ; seuil ; `objective` ajouté à
      `dropUncomputableRadarAxes` (aucun bloc objectif → axe retiré). PSA conservé
      UNIQUEMENT pour le résiduel Score (décision 1) — commentaires re-synchronisés.
- [x] Ascension : `computeObjectiveIndexAxis` (fenêtre via `windowMatchIDs` factorisé,
      gating `games.ProvidesObjectiveStats` — helper capability neuf, patron
      live_service_source) ; awards.toml conservé pour les AUTRES axes
      (`applyAwardsRadarAxes` skippe objective) ; seuil spécifique par surcharge de
      `DefaultThresholds("custom")`.
- [x] `DefaultThresholds` : doc mise à jour (champ Objective plus consommé par les
      surfaces vivantes — surchargé par ObjectiveIndexThreshold).
- [x] Tests service : session (P80→80, axe retiré sans données), teammates (mock
      `fakeObjectiveIndexRepo` : P80→80 + coéquipier sans ligne à 0 aligné ; axe retiré
      si repo absent [capability] ou n_obj=0), match view (mapping ⊆ poids par famille +
      split KOTH ; P80→80 ; résiduel Score PSA ; axe retiré sur Slayer), profil
      (intégration : axe retiré malgré awards objective mappés, présent avec lignes
      shared, support toujours boosté par awards).
- Gate : `go build ./... && go vet ./...` (exit 0+0) + `go test ./internal/service/...
      ./internal/progression/...` (exit 0) + intégration profile `-tags=integration`
      rejouée (exit 0) — 2026-07-26.

### Étape 6 — Web (textes uniquement)
- [x] Tooltips/glossaire réécrits FR+EN : `features/squad/i18n.ts` (tooltip
      synergyRadar.objective FR :503 + EN :812), `features/help/i18n.ts` (entrée
      glossaire « Objectif »/« Objective » réécrite avec formule + exemple ; formule
      résiduel Score :201/:587 VÉRIFIÉE toujours exacte — le PSA reste soustrait,
      décision 1), + `features/match-view/i18n.ts` (radarTooltipObjective FR :303 +
      EN :568 — même axe, même source, doc inversée sinon). Manifests
      session.toml/squad.toml NON touchés (labels d'axes seulement, toujours valides)
      → pas de rebuild manifests. Grep : plus aucune occurrence « PersonalScoreAwards »
      côté web.
- [x] AUCUN changement de rendu vérifié sur pièces :
      `SessionParticipationBars.tsx` :111-126 mappe `entry.participation` dynamiquement ;
      `squadSynergyRadarChart.ts` :57-61 aligne les indicateurs sur series[0].axes (le
      backend retire l'axe de TOUTES les séries via skipObjectiveAxis scope-level) ;
      `MatchSummaryCharts.tsx` :329-337 mappe mySeries.axes dynamiquement.
- Gate : purge `node_modules\.tmp` + `npm run typecheck` (exit 0) + `npm run lint`
      (exit 0, 15 warnings = baseline incompatible-library) ; vitest ciblé N/A (aucune
      logique touchée, aucun test n'asserte les strings modifiées — grep vérifié) —
      2026-07-26.

### Étape 7 — Non-régression des effets en cascade
- [x] Vérifié : `Strengths`/`ImprovementAreas`/`DominantRole` (profile/service.go)
      tolèrent un radar à 5 axes (topN borné par len ; ImprovementAreas requiert ≥ 6
      axes → vide quand l'axe Objectif est retiré, comportement existant piloté par le
      nombre d'axes, inchangé). Coach escouade : `perfComponentAxis`
      (api/wire/prestige_squad_profile.go:124 — pas prestige/squad_coach.go) n'émet
      toujours ni objective ni support → l'axe focal ne peut pas être objective ; NON
      « corrigé » conformément au plan (déjà en Découvertes héritées).
- Gate : `go test ./internal/prestige/... ./internal/progression/...` vert (exit 0,
      + ./internal/api/... rejoué vert — wiring touché) — 2026-07-26.

## Interdits spécifiques à ce lot
- Pas d'UPSERT / pas d'écriture hors backfill officiel (étape 1 = CLI existant uniquement).
- Lectures append-only via `_latest` uniquement.
- Pas de `slug == "..."` ; pas de hex/couleur en dur côté web ; pas de nouvelle string UI
  sans paire FR+EN ; `api/openapi.yaml` jamais édité à la main.
- Builds Go strictement séquentiels ; CGO env (`CGO_ENABLED=1`, PATH msys64 ucrt64).

## Découvertes (à remplir par l'exécutant — ne pas traiter)
- (héritées du diagnostic, déjà connues : payload mort Squad V2 `charts.radar` + 2 clés
  i18n orphelines ; `BuildMatchRadar`/`haloAwardToAxis` sans appelant prod ; dette D-15
  agrégat 9 colonnes ; `objectiveStatColumns` citations sans les 18 colonnes v7.2.1 ;
  `matchModeFamilyFromMeta` ignore KOTH/Stockpile/Extraction/VIP ; `perfComponentAxis`
  n'émet jamais objective/support ; PSA manquants dégradés en silence dans 3 lecteurs.)
- 2026-07-26 (exécution) :
  - Le reset ciblé `MBitObjectiveStats` (pré-requis du backfill objectifs pour les
    modes ajoutés après coup) n'est PAS outillé : c'est un UPDATE ad hoc sur
    match_registry. Candidat : étendre l'action admin `lying-bits/reset` (qui clear
    déjà des bits menteurs, writer sérialisé dblease) ou un flag `--reset-modes` sur
    `cmd/backfill_objective_stats`. Tant que ce n'est pas outillé, tout agent sous
    interdiction d'écriture ad hoc est bloqué sur ce backfill.
  - Tous les CLI backfill RW sont inopérables tant que le serveur dev tient la shared
    DB (lock process exclusif) — le message d'erreur DuckDB donne le PID, utile.
  - Oddball local : 7 matchs / 58 lignes seulement → P80 `provisional` malgré des
    mesures réelles. À re-mesurer avec le pool prod (le VPS a plus d'historique).
  - Bloc de doc ORPHELIN décrivant `synergyRadarThresholds` en FIN d'un autre fichier
    (`teammates_squad_charts_weapons_perf.go:647`, la fonction vit dans
    teammates_squad_charts_synergy.go) — il était déjà périmé pour l'axe Score avant ce
    lot. Mis à jour ici (doc inversée sur mon changement) mais l'emplacement reste à
    rapatrier près de la fonction lors d'un lot de nettoyage.
  - La lecture d'index dégrade en silence (warn) sur erreur SQL : une clé de poids
    fantôme ne casserait pas la page mais éteindrait l'axe — couvert par le test de
    cohérence clés ⊆ schéma, à maintenir si on ajoute une famille.

## Protocole de reprise
Avancement = cases de ce fichier + rapport d'agent. Reprendre à la première étape non
close. Le superviseur passe les gates finaux complets et commit.

## Journal d'exécution

### [2026-07-26] Étape 0 — notes de vérification sur pièces (gate)

Écarts vs diagnostic : AUCUN écart bloquant, lignes quasi inchangées.
- `session_compare_participation_helpers.go` : `buildSessionParticipationProfile` :80-150 ;
  accumulation `objTotal` :122-124 ; seuil `Objective: 350.0 * n` :136. Conforme.
- `teammates/teammates_squad_charts_synergy.go` : `synergyRadarThresholds` :28-38
  (Objective 350×n :35) ; PSA via `LoadObjectiveScores` :161-167. Conforme.
- `match_view_radar.go` : `BuildMatchRadarFromScoreboard` :193-236 (seuil Objective 350
  :223), `computeMatchRadarRawAxes` :241-279 (résiduel Score :273),
  `dropUncomputableRadarAxes` :288-306. Conforme. CONFIRMÉ : Q12
  (`match_view_repo_scoreboard.go` :77-117) charge déjà `ScoreboardRaw.Obj`
  (`domain.ObjectiveRaw`, 41 colonnes + discriminants `HasCTF/…/HasVip`) pour TOUS les
  joueurs → l'index single-match se calcule sans requête supplémentaire.
- `match_view_data_loaders.go` : chargement `d.objectiveScore` (PSA) :88-92, câblage
  radar :417-418. Conforme.
- `progression/profile/queries.go` : `computeRadarAxes` :86-99, base :103-161,
  `applyAwardsRadarAxes` :170-241 (Objective via awards.toml). `profile/service.go` :
  `aggregateNarrative` :203-257, `DefaultThresholds("custom")` :211. Conforme.
- État base locale (diag_q, `match_objective_stats_latest`) : 5757 lignes.
  Lignes par famille : ctf 3579 · zones 2120 (dont 375 lignes koth ticks>0) · oddball 58
  · stockpile 0 · extraction 0 · vip 0.
  Matchs distincts : ctf 234 · strongholds 136 · koth 47 · oddball 7 · stockpile 0 ·
  extraction 0 · vip 0. → backfill étape 1 requis pour les 3 familles v7.2.1 ;
  oddball (7 matchs) sera < 30 matchs → constante provisoire prévue (décision 6).
- Serveur dev local actif (PID 29964, port 8000, `apps/go-api/tmp/server.exe`) — tient la
  shared DB ; consigne : ne pas l'arrêter.

### [2026-07-26] Étape 1 — backfill non exécutable, fallback décision 6 (gate)

Tentative RÉELLE : `go run ./cmd/backfill_objective_stats --dry-run --limit 5` → exit 1 :
`IO Error: Cannot open file "...shared_matches_v2.duckdb": Le processus ne peut pas accéder
au fichier [...] File is already open in apps\tmp\server.exe (PID 29964)`.
Le CLI exige l'accès RW exclusif (serveur arrêté) ; consigne de session : ne pas arrêter ni
redémarrer le serveur dev. Par ailleurs le pré-requis « reset ciblé MBitObjectiveStats »
(43/44 matchs Stockpile/Extraction/VIP portent déjà le bit, cf. thought_log 2026-07-25) est
une écriture ad hoc non outillée sur la DB partagée — improvisation interdite.
→ Fallback décision 6 : constantes `provisional` pour stockpile / extraction / vip.
Re-mesure post-backfill : rejouer les requêtes de calibration de l'étape 2 quand
`match_objective_stats_latest` comptera ≥ 30 matchs distincts pour la famille (reset +
backfill à jouer par le superviseur, serveur coupé). Rappel volumes attendus localement :
39 matchs Stockpile / 2 Extraction / 3 VIP → extraction et vip resteraient `provisional`
même après backfill (n < 30).

### [2026-07-26] Étape 2 — calibration mesurée (gate)

Population : `match_objective_stats_latest o JOIN match_participants mp` (même match_id +
xuid), `mp.time_played_seconds > 30` (NB : colonne réelle `time_played_seconds`, pas
`time_played`), `o.xuid NOT LIKE 'bid(%'`. actions = somme pondérée décision 7 par ligne ;
hold = durée_f / time_played_seconds par ligne. Quantiles QUANTILE_CONT.

| Famille | n lignes | n matchs | actions p50/p80/p90 | hold p50/p80/p90 |
|---|---|---|---|---|
| ctf | 3419 | 234 | 3 / 11 / 16.5 | 0.0016 / 0.0434 / 0.0807 |
| zones_strongholds | 1639 | 136 | 15 / 29 / 36 | 0.1075 / 0.1809 / 0.2194 |
| zones_koth | 361 | 47 | 35.5 / 51 / 60.5 | 0.1348 / 0.2178 / 0.2591 |
| oddball | 57 | 7 | 21.5 / 35 / 43.1 | 0.0614 / 0.1187 / 0.1651 |
| stockpile / extraction / vip | 0 | 0 | — | — |

Constantes retenues (P80) :
- MESURÉES : ctf actions 11.0 / hold 0.0434 · strongholds actions 29.0 / hold 0.1809 ·
  koth actions 51.0 / hold 0.2178.
- PROVISOIRES (`provisional`, re-mesure quand la famille atteint ≥ 30 matchs distincts
  en local après reset+backfill) :
  - oddball actions 35.0 / hold 0.1187 (mesures réelles sur 57 lignes / 7 matchs —
    retenues comme meilleure estimation, sous le seuil de significativité) ;
  - stockpile actions 20.0 / hold 0.10 (analogie porteur-objet : cadence CTF × porteurs
    multiples, hold entre oddball et strongholds) ;
  - extraction actions 12.0, pas de hold (décision 8 : r = actions/P80 seul) ;
  - vip actions 18.0 / hold 0.12 (vip_kills ~ zone_captures en fréquence, temps VIP
    normalisé par sélection ~ hold oddball).

Note d'interprétation décision 7 (consignée, non rouverte) : `flag_capture_assists`
pondérée 2.0 (équivalent awards.toml `flag_capture_assist` — liste « … » non exhaustive) ;
`kills_as_flag_returner` écartée par analogie explicite avec `kills_as_flag_carrier`
(kill-en-rôle, signal combat). `extraction_conversions_denied` = `extraction_denied` 2.0
(double mention du même poids dans la décision).

### [2026-07-26] Étapes 3-7 — implémentation + gates (résumé)

Étape 3 : `internal/analysis/narrative/objective_participation.go` + tests (pur, 0 I/O).
Étape 4 : `internal/platform/duckdb/objective_index_repo.go` + tests :memory: (SELECT
généré depuis les tables narrative, source unique).
Étape 5 : 4 surfaces injectées + suppressions PSA session/squad (0 code mort) +
`games.ProvidesObjectiveStats` + port `ObjectiveIndexRepository` + wiring capability-gated.
Étape 6 : textes FR/EN squad + help + match-view ; rendu inchangé vérifié sur pièces.
Étape 7 : cascades vérifiées (profil 5 axes OK, coach non modifié).
Gates finaux de lot : `go build` 0 · `go vet` 0 · `go test` analysis + platform/duckdb +
service + progression + prestige + api + port + games + archlint + sync → exit 0 (aucun
`--- FAIL:`), intégration profile `-tags=integration` exit 0 ; web : typecheck 0, lint 0
(15 warnings baseline). gofmt propre sur les répertoires touchés.
