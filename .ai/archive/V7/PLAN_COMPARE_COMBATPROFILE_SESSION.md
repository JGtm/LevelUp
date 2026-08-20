# PLAN — Face a face / Profil de combat / Session (polish)

> Statut : PLAN VALIDE (analyse terminee, implementation a venir).
> Branche : **branche courante** (`feat/skill-progression-magnitude-scale`) — pas de nouvelle branche, commits sequentiels.
> Date analyse : 2026-06-02.

Plan issu de l'analyse des 5 points demandes par l'utilisateur. Aucune modification de code
a ce stade ; ce document sert de suivi tracable avant et pendant l'implementation.

---

## Decisions verrouillees (arbitrages utilisateur)

1. **Rendement / Resistance (Face a face)** : alignement **metrique + format** sur la KPI bar home
   (OC/DR canonique), pas seulement le format.
2. **Donnees live a brancher dans le Face a face** : CSR classe + top medailles + **echantillon des
   20 derniers matchs fetch en live**.
3. **20 derniers matchs** : fetch **live**, **persiste temporairement en cache memoire** (PAS d'ecriture
   DuckDB), TTL **20 min** (fourchette 15-30 demandee), modele `CachedStatsProvider`.
4. **Graphes du profil de combat** : **live uniquement pour les non-locaux** ; les joueurs suivis
   conservent la lecture locale DuckDB existante (gratuite).
5. **Banniere aleatoire (point 4)** : priorite backdrop > pool conservee ; on corrige seulement le cas
   "aucune identite" qui laisse le non-local sans banniere.

---

## Etat des lieux (cartographie confirmee)

### Terminologie
- **Face a face** = ComparePage / CompareService — comparaison joueur A (local) vs joueur B.
  - Front : `apps/web/src/features/compare/ComparePage.tsx`
  - Back : `apps/go-api/internal/service/compare_service.go`, handler `internal/api/handlers/compare.go`
- **Profil de combat** = encart cible de l'Explorer (identite + graphes).
  - Front : `apps/web/src/features/explorer/ExplorerCombatProfile.tsx`, `CombatFdaChart.tsx`,
    `CombatScorePlacementChart.tsx`, `combatChartOptions.ts`, `ExplorerTargetIdentityBanner.tsx`
  - Back : `internal/service/explorer_service.go`, `explorer_service_target.go`
- **Session** = page detail session.
  - Front : `apps/web/src/features/session-detail/SessionSummaryCard.tsx`, `SessionKillsDonut.tsx`,
    `SessionOutcomeDonut.tsx`, wrapper `components/charts/DonutChart.tsx`

### Cache / TTL existant (a reutiliser)
- `internal/service/remote_stats_cache.go:34` → `DefaultRemoteStatsTTL = 5 * time.Minute`.
  Cache process-level en memoire (map + singleflight + expvar), aligne sur `CareerLiveCache`.
  C'est le pattern "persiste temporairement" (RAM, pas BDD) a dupliquer pour les 20 matchs.

### Capacites HTTP live existantes (couche sync, persistent aujourd'hui)
- Liste matchs : `internal/sync/halo_client.go:214` (`GET /hi/players/{..}/matches`).
- Stats par match : `internal/platform/halo/discovery_client.go:71` (`FetchMatchStats`).
- A reutiliser **hors pipeline persist** pour un fetch read-only.

---

## Point 1 — Rang carriere N/A pour un non-local (Face a face)

**Cause racine.** `compare_service.go:117-135` (`loadPlayerB`) fetch un non-local via
`provider.FetchRemoteStats` (service record Waypoint) qui **n'expose pas le career rank**. Double verrou :
`compare_service.go:261-264` (`metricAvailability`) exige `isLocal && value>0` pour `career_rank`.
L'Explorer, lui, obtient le rang via `explorer_service_target.go:50-59` (`FetchLiveIdentity(xuid)`).

**Fix.**
1. Port : reutiliser l'interface live identity de l'Explorer (`FetchLiveIdentity(ctx, xuid)`).
2. `loadPlayerB` (branche non-locale) : resoudre xuid B (cf. risque), fetch live identity,
   `remote.CareerRank = id.CareerRank`. Best-effort (skip sans auth).
3. Relacher `metricAvailability` pour `career_rank` afin d'accepter une valeur live non-locale
   (flag explicite type `CareerRankLive` pour ne pas confondre avec l'ATH local).
4. Wiring `registry_pages.go` : injecter le resolver dans `CompareService`.

**Depend de** : le cablage live des commits 4-5 (mutualise).

---

## Point 2 — Rendement / Resistance alignes sur la KPI bar (Face a face)

**Cause racine (double divergence).**
- Formules differentes :
  - Face a face `compare_service.go:370-382` : rendement = `damage/kill/225` (semantique INVERSEE,
    `lessIsBetter:true`).
  - Home `analysis/combat_yield.go:33` : OC = `225*(kills+assists/3)/damage` (plus haut = mieux).
- Format different :
  - Face a face `ComparePage.tsx` : `v*100` a 1 decimale.
  - Home `components/ui/off-def-composite.tsx:58-61` : `(off*100).toFixed(0)` et `((def-1)*100).toFixed(0)`.

**Fix (alignement complet).**
1. Service : remplacer `computeRendement`/`computeResistance` par OC/DR canonique
   (reutiliser `analysis.ComputeCombatYield` agrege, ou recalculer depuis les per-game de
   `NormalizedPlayerStats`). Exposer `AvgOC`/`AvgDR` (parite avec `SessionCompareEntry.avg_oc/avg_dr`).
2. `buildMetrics` : OC passe `lessIsBetter:false` → adapter le calcul de vainqueur.
3. Front `ComparePage.tsx` : reutiliser le format `OffDefComposite` (idealement le composant).
4. Nuance a documenter : home moyenne par-match (round2) vs compare per-game agrege
   (moyenne-de-ratios != ratio-de-moyennes) → ecart marginal accepte.

---

## Point 3 — 20 derniers matchs live + graphes profil de combat

**Etat actuel.** Le champ `combat_profile` (`[]ExplorerTargetRecentMatch`, limite 20 via
`explorer_service.go:365` `explorerCombatProfileLimit = 20`) qui alimente les graphes est aujourd'hui
une **lecture locale DuckDB** : `explorer_repo.go:238` (`GetTargetRecentMatches`, SharedReader,
`Q19cTargetRecentMatches`). Pour un non-local sans matchs communs → vide/partiel.

**Fix.**
1. Nouveau provider read-only `RecentMatchesProvider.FetchRecentMatches(ctx, xuid, 20)` :
   reutilise les primitives HTTP (liste `halo_client.go:214` + `discovery_client.FetchMatchStats`),
   **sans passer par persist/BatchBuilder** (zero ecriture DuckDB).
2. Cache memoire TTL **20 min** (`DefaultTargetRecentMatchesTTL`), cle `titleSlug|xuid`,
   singleflight + compteurs expvar (modele `remote_stats_cache.go`).
3. Branchement graphes profil de combat : **live si non-local**, lecture locale DuckDB si joueur suivi.
4. Branchement Face a face : stats du non-local calculees sur cet echantillon live (au lieu de
   l'echantillon croise biaise de `enrichRemotePlayerBWithCrossSample`), + CSR (`SeasonCSRs` Explorer)
   + top medailles (deja fetchees par l'Explorer).

**Caveat critique** : endpoint `/matches` doit utiliser le format **`xuid(NNN)`** (pas le gamertag brut),
sinon reponse stale silencieuse (cf. memoire `reference_halo_api_xuid_format`). Auditer
`halo_client.go:214` qui passe `url.PathEscape(gamertag)`.

---

## Point 4 — Banniere aleatoire absente pour un non-local (Profil de combat)

**Cause racine.** La nameplate deterministe (`explorer_service_target.go:70-96`) n'est appliquee que si
une identite existe deja (gate `if id == nil ... return`). Or pour un non-local **sans auth** (ou si le
fetch live echoue), `fetchTargetIdentityRaw` (`explorer_service_target.go:43-63`) retourne `nil`
→ `identityRaw=nil` → `buildTargetProfile:349` produit `identity=nil` → front bascule sur le placeholder
"Identite Spartan indisponible" sans aucune banniere (`ExplorerTargetIdentityBanner.tsx:55-77`).

**Fix.**
1. Quand ni local ni live n'est exploitable, construire une identite minimale (`&HomeSpartanIdentityRow{}`)
   et lui appliquer `applyBannerFallbacks(ctx, id, targetXUID)` au lieu de retourner `nil`.
2. Conserver priorite backdrop > pool (defaut). Pool vide (aucun joueur suivi avec banniere) → log + degrade propre.

---

## Point 5 — Session : valeur FDA au centre du donut + precision moyenne dans la card

**Etat.** Card "FDA" = tuile KDA `SessionSummaryCard.tsx:66-70` (`labelOf('kda')`, valeur `entry.kda`).
Donut "F/D/A" `SessionKillsDonut.tsx` : **pas de label central**. Pattern centre disponible
(`centerValue`/`centerLabel` du `DonutChart`, deja utilise par `SessionOutcomeDonut.tsx`).
Precision moyenne de session : **MANQUANTE** sur `SessionCompareEntry` (`avg_oc`/`avg_dr` existent → meme chemin).

**Fix.**
1. Back : ajouter `AvgAccuracy *float64` a `SessionCompareEntry` (`domain/session_compare.go`),
   calcule la ou `avg_oc/avg_dr` sont agreges (unite 0..1).
2. Types : `avg_accuracy: number | null` (`apps/web/src/lib/api/types.ts`).
3. Donut `SessionKillsDonut.tsx:68` : `centerValue={formatNumber(kda,2)}` + `centerLabel` i18n ("F/D/A").
   Passer `kda` en prop depuis le parent (le composant ne recoit aujourd'hui que `matches`).
4. Card `SessionSummaryCard.tsx:66-70` : remplacer la tuile KDA par precision
   (`labelOf('accuracy')`, `formatPercent(entry.avg_accuracy)`).
5. i18n FR+EN pour le centerLabel.

---

## Ordre d'execution (branche courante, commits sequentiels)

| # | Commit | Coeur | Effort |
|---|--------|-------|--------|
| 1 | Point 4 — banniere non-local | identite minimale + nameplate pool quand `identityRaw==nil` | faible |
| 2 | Point 5 — donut center + accuracy | `AvgAccuracy` (back) + centerValue donut + card precision | faible-moyen |
| 3 | Point 2 — alignement OC/DR | OC/DR canonique + format `OffDefComposite` + sens vainqueur | moyen |
| 4 | Point 3a — provider live read-only | `RecentMatchesProvider` (hors persist) + cache TTL 20 min, format `xuid(NNN)` | moyen-lourd |
| 5 | Point 3b — branchements | graphes combat profile (live si non-local) ; Face a face (sample live + CSR + medailles) | moyen |
| 6 | Point 1 — career rank live | piggyback cablage live + relacher `metricAvailability` | moyen |

Commits 4-5-6 mutualisent l'injection des providers live (`LiveIdentity`, `SeasonCSRs`,
`RecentMatchesProvider`) dans `CompareService`.

---

## Garde-fous (delivery-checklist)

- Fetchs live bornes par budget (modele `explorerTargetLiveBudget`) + best-effort (une source qui echoue
  ne bloque pas les autres).
- `slog.*Context` sur erreurs non triviales et fetchs significatifs ; pas de `fmt.Println`.
- Tests par couche : mock providers (career rank live, recent matches) ; dataset heterogene pour les
  samples (cf. memoire `feedback_integration_tests_realistic_datasets`) ; donut center
  (`SessionNewCharts.test.tsx`) ; agregation `avg_accuracy`.
- `go test ./... && go vet ./...` (CGO/DuckDB requis) + `typecheck`/`lint`/`vitest` (vitest hors sandbox).
- Strings UI FR+EN ; couleurs via tokens (pas de hex/Tailwind couleur).
- Entree `thought_log.md` par commit (regle CLAUDE.md).

## Risques residuels a lever en cours d'implementation

- **Resolution gamertag→xuid dans `CompareService`** (points 1 et 3) : le `RecentMatchesProvider` et le
  fetch career rank live exigent le xuid ; `ResolveXUID` renvoie "" pour un non-local. Confirmer le chemin
  reutilisable (resolution Halo) avant le commit 4.
- **Format `xuid(NNN)`** sur `/matches` (point 3) : a forcer dans le nouveau provider.
- **Latence** : les fetchs live du Face a face doivent rester bornes et best-effort.
