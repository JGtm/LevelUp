# PLAN — Damage model par titre (baseline « PV effectifs pour tuer »)

> **Créé** : 2026-06-20 (révélé en finalisant Canonical MatchEvents, avant activation Halo 5).
> **Branche** : `feat/multititre-peripherie` (séquencer avec l'activation 1b — c'est là que le gap mord).
> **Axe registre** : extension/concrétisation de **MT-13** (`PLAN_MULTITITRE_INDEX.md`) — qui ne capturait que les tables Halo **client-side** et différait « au 2e titre ». Ce plan ajoute le **backend** (sous-scopé par MT-13) + la baseline Halo 5 + la recalibration P80. Le « 2e titre » = maintenant (Halo 5).
> **Réfs** : `combat_yield.go` (`DAMAGE_EFFICIENCY_INTEGRATION.md`), mémoire `project_halo5_experimental_direction`.

## 0. ÉTAT VÉRIFIÉ (2026-06-20, audit code) — PHASES 0→2 DÉJÀ LIVRÉES

> ⚠ Le corps de ce plan (§2-§4) décrit l'état **avant** threading. Vérification code
> ce jour : **le seam + la quasi-totalité du compute sont déjà title-aware** (commits
> de cette journée). Ne PAS refaire ce travail à l'activation — ne reste que du
> Phase 3 **activation-couplé**.

**FAIT (vérifié) :**
- **Phase 0** ✅ : `[damage_model] effective_hp_to_kill` dans `constants.toml` (Infinite implicite=`225` via `games.DefaultEffectiveHpToKill` ; Halo 5=`115`). Loader `internal/games/mappings/loader_endpoints.go`. Getters `games.EffectiveHpToKill(slug)` / `EffectiveHpToKillFromResolver` + test oracle `damage_model_test.go`.
- **Phase 1** ✅ : `analysis/combat_yield.go` (`ComputeCombatYield(Float)`) prend `effectiveHpToKill` en **paramètre** (plus de littéral) ; **tous** les points d'entrée + callers service résolvent `games.EffectiveHpToKill(titleSlug)` : home (`home_kpis`/`home_recent`/`home_canonical_*`), `explorer_target_stats`, `kpi_stats`, `stats_canonical`, squad/synthesis, `compare_service`, match-view (`computeScoreboardRowCombatYield`/radar), `engagement_player_service`, `timeseries`, `teammates_*`, `patterns_repo`. Bootstrap expose `effective_hp_to_kill` par titre.
- **Phase 2 (post-sync SQL)** ✅ : `post_sync_progression_queries.go` utilise `games.EffectiveHpToKill(pdb.TitleSlug)` (plus de `225.0` inline).
- **Phase 3 front (copy)** ✅ : `help/i18n.ts` `withDamageBaseline` (token `{{HP}}`, title-aware via `effective_hp_to_kill`) ; `combat-yield-display`/`match-card` consomment les valeurs **calculées back** (déjà title-aware).

**Littéraux `225.0` restants = NON des gaps (à NE PAS toucher) :**
- `platform/duckdb/queries_career_encounters.go:172,177` (Q23 `LoadStatsMatches`) → **chemin LEGACY retiré** (`port/repository_sessions_home.go:21` : « LoadStatsMatches a été retiré, les services chargent canonical »). Les services live recalculent via `StatsMatchRowsFromCanonical(…, EffectiveHpToKill(slug))`. Le 225 SQL n'alimente plus rien de live.
- `internal/sync/skill_rating.go:335,338` (composite LUSR) → **LUSR = Infinite-only** (Halo 5 a un CSR natif, ne passe pas par LUSR). 225 y est correct par nature.
- `cmd/diag_lusr_player`, `cmd/lusr_v2_phase0` → outils **diag offline** (replay LUSR), hors chemin produit.

**MAJ 2026-06-23** : items 1 (115 design), 3 (front title-aware), 4 (KDA h5 = `h5NetFDA` DÉJÀ correct à l'ingestion) RÉGLÉS ; item 2 (P80) **CALIBRÉ data** = h5 OC P80 **1.264** (config `offensive_conversion_p80` + getter livrés). Reste UNIQUEMENT le threading display radar/barre (activation 1b ; DR N/A car cryptum sans `damage_taken`).

**RESTE (Phase 3, activation-couplé — voir §7) :**
1. ✅ **`115` Halo 5 = valeur DESIGN AUTORITAIRE** (confirmé user 2026-06-23 ; bouclier 70 + armure 45). NE PAS la dériver de la moyenne empirique `Σdmg/Σkills` (≈136 sur 8856 lignes h5) : cette moyenne inclut overkill + dégâts d'assistances, avec un facteur title-spécifique NON constant (Infinite ×1.6 = 358.8/225 ; h5 ×1.18 = 136/115) → un « scale-match » sur l'empirique FAUSSE la valeur (un essai à 86 le 2026-06-23 = ERREUR, **reverté à 115**). Le PV-pour-tuer DESIGN fait foi.
2. ✅ **P80 OC h5 CALIBRÉ = 1.264** (config `[damage_model].offensive_conversion_p80` + getter `games.OffensiveConversionP80(slug)` livrés, défaut/Infinite 0.90 + test). Mesuré 2026-06-23 = **80e percentile de l'OC** sur la vraie distribution h5 (12891 lignes `match_participants`, hp=115). MÊME méthode que le 0.90 d'Infinite (= son propre 80e percentile, vérifié 0.918≈0.90) → « mêmes proportions relatives » par construction (ratio P80/médiane : h5 1.28 ≈ Infinite 1.25). NE PAS réutiliser 0.90 pour h5 (médiane h5 0.99 > 0.90 → tous élite). **Résistance (DR) = N/A** : le carnage cryptum h5 ne fournit PAS `damage_taken` (vérifié sur le retour BRUT live, 0/13241 — uniquement des dégâts INFLIGÉS). **Threading display LIVRÉ (2026-06-23)** : `offensive_conversion_p80` exposé au bootstrap (`TitleSummary` + openapi + getter `games.OffensiveConversionP80`) ; barres OC front (`combat-yield-bar`, `TimeseriesCombatYield`, `SessionOcdrBars`) via hook `useOffensiveConversionP80` ; radars backend **match-view** (`BuildMatchRadarFromScoreboard`) + **squad** (`synergyRadarThresholds`) via `games.OffensiveConversionP80(slug)`. Le chemin LUSR garde la const `analysis.OffensiveConversionP80` (Infinite-only). **SEULE exception** : le radar **session-compare** (`buildSessionParticipationProfile`) gardé sur la const — son threading passe par 3 niveaux (`buildCompareEntry`→`WithObjectives`→profile) + ~13 fixtures de test, disproportionné pour cette surface secondaire (h5 `not_exposed`) ; passe dédiée si besoin. Tests Go (games/service/api) + front (typecheck/eslint/vitest) verts.
3. ✅ **Front title-aware LIVRÉ (2026-06-23)** : `ONE_LIFE_DAMAGE` reste le défaut Infinite (225) mais `offensiveDamageGradient`/`defensiveDamageGradient`/`damageAxisBounds` prennent un param `oneLife` ; `TimeseriesEfficiency` (+ `EfficiencyTooltipText`) résolvent le PV-pour-tuer du titre courant via `useEffectiveHpToKill()` (`lib/damage/effectiveHp.ts`) + jeton `{{HP}}` dans `common.charts.efficiency_tooltip` + `timeseries.progression.ref_one_life`. Charts escouade (`squadEfficiencyChart`) gardent le défaut 225 (escouade = Infinite-only, h5 not_exposed). Tests : `oneLifeDamageGradient.test` (param oneLife) + `effectiveHp.test` (substituteHpToken) verts.
4. **KDA/FDA = stat API, JAMAIS calculée pour Infinite (relevé user 2026-06-20)** — `internal/sync/transforms.go:342` LIT le `KDA` natif de l'API Infinite (`row.KDA = floatPtrFrom(core, "KDA")`, jamais recalculé). **Formule native (concepteur ranking Halo 5)** : `FDA = ((k + ⅓·a) − d) / games` — forme **NET / différence** (PAS un ratio), par-match = ÷1, donc **peut être NÉGATIVE**. Ça explique vraisemblablement (a) les KDA négatifs côté API et (b) **pourquoi on n'a JAMAIS reproduit le FDA Infinite en local** (on calculait le ratio `(k+a/3)/d` au lieu de la différence `k+a/3−d`). **RÈGLE ABSOLUE** : le calcul local du KDA/FDA est une **exception**, **JAMAIS appliqué à Infinite** (Infinite lit toujours l'API).
   - ✅ **h5 KDA = RÉGLÉ** (le paragraphe ci-dessus décrit l'ANCIEN état, corrigé depuis) : `mapping_carnage.go:135` `h5NetFDA(k,a,d) = (k + a/3) − d` est calculé À L'INGESTION (forme NATIVE FDA NET, peut être négative), stocké dans `match_participants.kda`, testé (`capture_test.go`). Par match games=1 → on stocke le NUMÉRATEUR ; les lecteurs agrègent (AVG = métrique /games). Garde-fou respecté : Infinite lit toujours le KDA de l'API (jamais calculé). RIEN à faire.
   - **Poids d'assist** : la formule FDA native utilise `⅓` (pas `½`). Le « 1 assist = ½ frag » mentionné serait un AUTRE contexte (scoring/contribution in-game ?) — **à clarifier**. Le seul endroit où NOUS calculons avec un poids d'assist = le **rendement** (`OffensiveConversion`, métrique LevelUp absente de l'API) → c'est LÀ (et seulement là) que `assist_frag_weight` title-aware s'applique, PAS au KDA. Résistance non affectée (pas d'assist).
   - **Fix rendement** = `[damage_model] assist_frag_weight` (Infinite `3` ; h5 = à confirmer) + getter `games.AssistFragWeight(slug)` threadé dans `FragEquivalents`. **N'impacte PAS le sync** (K/D/A bruts persistés). À traiter AVEC les items 1-3.

## 1. Objectif + critère de succès

**Objectif** : rendre la baseline « dégâts pour tuer un Spartan » (et tout KPI dégâts-normalisé : rendement offensif / résistance défensive) **paramétrable par titre**, au lieu de `225` (Halo Infinite) câblé en dur dans les calculs ET l'affichage.

**Critère de succès** : pour Halo 5, rendement/résistance sont calculés et affichés **sur la bonne échelle** (baseline Halo 5 validée sur données réelles, ≈ `115` = bouclier 70 + armure 45) ; **zéro littéral `225`** dans les chemins de calcul ; le copy d'aide n'affirme plus « convention Halo Infinite » pour un autre titre. Halo Infinite reste **byte-identique**.

## 2. Problème (constat code, 2026-06-20)

`225` = PV effectifs Infinite (90 vie + 135 bouclier, échelle de dégâts de l'API). Formules : `offensive_conversion = 225·(kills + assists/3)/dmg_dealt` ; `defensive_resistance = dmg_taken/(225·deaths)`. **Câblé en dur, dupliqué** :

| Site | Type | Détail |
|---|---|---|
| `apps/go-api/internal/analysis/combat_yield.go:59-67` | **compute (source canonique)** | `225.0` ×3 + P80 calibrés Infinite (`OffensiveConversionP80=0.83`, `DefensiveResistanceP80=1.59`, 4 joueurs Infinite) |
| `apps/go-api/internal/analysis/squad_breakdown_canonical.go:79,84` | **compute** | OC/DR agrégés escouade |
| `apps/go-api/internal/api/post_sync_progression_queries.go:232,237,337-342` | **compute (SQL inline)** | `225.0` en dur dans DuckDB (le plus pénible) |
| `apps/web/src/components/ui/{combat-yield-bar,combat-yield-display,match-card}.tsx` | front display | inverse `225 / dmgPerKill`, commentaires + affichage |
| `apps/web/src/components/charts/EfficiencyTooltipText.tsx` | front tooltip | « 1 vie ≈ 225 pts » |
| `apps/web/src/features/help/i18n.ts:91-100` | **copy user-facing** | « un Spartan possède 225 PV (90+135), convention officielle de Halo Infinite » — **faux pour Halo 5** |

**Impact** : la donnée `damage_dealt`/`damage_taken` brute est correcte par titre, mais le **diviseur 225 est une hypothèse Infinite**. Pour Halo 5 (115), conversion ≈ ×(225/115)≈1,96 **trop haute**, résistance ≈ ×2 **trop basse**. Pas un crash — un **mauvais cadrage** dès que Halo 5 est live.

**Nuance critique** : `225` Infinite est **empirique** (calibré data), pas le PV affiché en jeu. Le `115` Halo 5 est la valeur **design** → **à VALIDER** contre les vrais `damage_dealt`/kills d'un match Halo 5 (l'unité de dégâts de l'API cryptum peut ne pas être 1:1 avec les PV). On a JGtm en Halo 5 → 1 match suffit (`Σdmg/Σkills ≈ ?`).

## 3. Architecture (title-aware)

| Élément | Emplacement | Rôle |
|---|---|---|
| Constante par titre | `config/titles/{slug}/constants.toml` → section `[damage_model]` (`effective_hp_to_kill`, + optionnel `shield`/`health`/`melee` pour le copy) | `225` Infinite / `115` Halo 5 (validé) |
| Lecture | `internal/games/mappings` (loader constants) → exposé via l'adapter sémantique / un getter titre | injecté au caller, jamais lu en dur |
| Compute pur | `analysis/combat_yield.go` + `squad_breakdown_canonical.go` | `ComputeCombatYieldFloat(..., effectiveHpToKill float64)` — paramètre, plus de littéral |
| SQL | `post_sync_progression_queries.go` | baseline en **bind param** (`?`) OU calcul remonté en Go |
| Calibration | P80 par titre (ou dégradation « non calibré ») | empirique, dépend de la data du titre |
| Front | combat-yield display/tooltip + `help/i18n.ts` | baseline exposée (field-mappings ou endpoint constante titre) → copy/affichage title-aware |

**Discipline** : tout champ adossé à une source réelle ; baseline Halo 5 validée data, pas supposée.

## 4. Phases (risque/effort croissant ; séquencer avec activation 1b)

### Phase 0 — Constante + validation data (petit, central)
- `[damage_model]` dans `constants.toml` Infinite (`225`) + Halo 5 (`115` provisoire) + loader + getter. **Valider `115` Halo 5** sur un vrai match (JGtm) : `Σdmg/Σkills` ≈ ? Ajuster la valeur réelle.
- Oracle : Infinite lit `225` depuis config = byte-identique ; getter testé.

### Phase 1 — Paramétrer les fonctions pures (compute) — ⚠ CASCADE MESURÉE 2026-06-20
- `ComputeCombatYield(Float)` + `ComputeSynthesisKPIsFromCanonical`/`extKPIAcc.applyTo` prennent `effectiveHpToKill` (param, restent pures) ; le caller résout `games.EffectiveHpToKill(pdb.TitleSlug)` et l'injecte.
- **Cascade réelle (compiler-driven)** : combat-yield est calculé dans **9 points d'entrée analysis**, chacun appelé par un service qui a le slug → threading du param à chaque niveau :
  - `combat_yield.go` (ComputeCombatYield / ComputeCombatYieldFloat) — cœur.
  - `squad_breakdown_canonical.go` (ComputeSynthesisKPIsFromCanonical + applyTo, inline 225).
  - `home_kpis.go:62`, `home_recent.go:128`, `home_canonical_kpis.go:101`, `home_canonical_recent.go:144` (page Home, 4 sites).
  - `explorer_target_stats.go:84` (profil combat Explorer).
  - `kpi_stats.go:143` ; `stats_canonical.go:91`.
  - **Callers service** (résolvent le slug) : home, explorer, stats, synthesis/squad, compare (`combatYieldOf`), match-view (`computeScoreboardRowCombatYield`), sync (`performance_helpers`).
  - **Tests** à mettre à jour (passent `225` = byte-identique) : `combat_yield_test.go` (8 appels), `synthesis_canonical_test.go`, etc.
  - **Total ~30-40 sites**, mécanique, interdépendant (fonctions analysis partagées → threading séquentiel, pas parallélisable sans casser l'invariant).
- Oracle byte-identique : TOUS les tests combat/stats existants verts en passant `225` (Infinite inchangé) ; Halo 5 = échelle 115.

### Phase 2 — SQL inline
- `post_sync_progression_queries.go` : baseline en bind param ou remontée Go. **Ne pas régresser** le post-sync Infinite.

### Phase 3 — P80 + front
- P80 par titre (table calibration) OU dégradation « non calibré » (barre normalisée masquée / valeur brute + flag) tant que Halo 5 manque de data.
- Front : display inverse + tooltips + **copy d'aide title-aware** (ne plus dire « Halo Infinite » pour un autre titre).

## 5. Tests (par couche)
- `analysis/combat_yield` : baseline param (225 → parité ; 115 → échelle H5) + P80 dégradé.
- `analysis/squad_breakdown` : OC/DR agrégés param.
- SQL : test repo post-sync (baseline param, Infinite parité).
- front : tooltip/help title-aware (vitest).
- Oracle double : Halo golden (225, byte-identique) + `synthetic_title_b` (baseline custom).

## 6. Blockers / dépendances
- **Validation baseline Halo 5** : besoin de vraies données damage Halo 5 (JGtm) → trivial une fois Halo 5 live/fetché.
- **Calibration P80 Halo 5** : besoin d'un échantillon de matchs Halo 5 → dégrader « non calibré » en attendant (honnête).
- **Dégradation naturelle déjà là** : sans `damage_dealt/taken`, ces KPI s'annulent (`if damageDlt > 0`) — la baseline ne mord que si la donnée existe.

## 7. Done definition / sequencing
- **À faire AVEC l'activation 1b** (le trigger : Halo 5 live = moment où le mauvais cadrage devient visible ; la vérif live de 1b le révélerait).
- Done = Halo 5 affiche rendement/résistance sur la bonne échelle (validée data) ; 0 littéral `225` compute ; copy title-aware ; P80 calibré OU dégradé proprement ; Infinite byte-identique.
- Thought_log à chaque phase. Met à jour MT-13 (`PLAN_MULTITITRE_INDEX.md`) à la clôture.
