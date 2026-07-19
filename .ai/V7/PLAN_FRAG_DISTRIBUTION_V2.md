# PLAN_FRAG_DISTRIBUTION_V2.md — « Répartition des frags » v2, sunburst hiérarchique classe→rôle

> Rédigé le 2026-07-19. Branche cible : `feat/frag-distribution-v2`.
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, statuer chaque item,
> vérifier sur pièces, zéro fix hors périmètre). Ce fichier fait foi ; si le code le contredit, corriger
> ce fichier dans le même commit.
>
> Contexte discuté et validé avec l'utilisateur (mockup publié, direction actée). Prédécesseur du surfaçage
> UI différé de `.ai/V7/PLAN_WEAPON_TAXONOMY.md` §9 (dim `class`/`role` du registre).

---

## 0. TL;DR

Fusionner les deux donuts de frags actuels (« Répartition des frags » kill-type + « Frags par type d'arme »
rôle) en **une carte hiérarchique** : donut **sunburst ECharts** à deux anneaux — **classe** au centre (axe
manipulation : Épaule / Poing / Lourde / Mêlée / Grenade [+ Capacités spartanes H5]), **rôle** à l'extérieur
(fonction de combat) — accompagné d'un **breakdown par arme en barres** recoloré par classe. Le total et les
classes API (mêlée/grenade/spartan) sont **exacts** ; la ventilation par arme est **estimée** (registre) ;
tout écart de parsing retombe dans une tranche **Non attribué** calculée. **Une couleur par classe, réutilisée
dans le breakdown par arme** → cohérence visuelle inter-pages.

**Critère de succès** : le sunburst v2 + breakdown par arme remplacent les graphes cibles sur Synthesis, Match
view, Timeseries, Sessions ; l'Escouade passe à des barres empilées par classe ; Explorer garde la v1 ; le
tout title-agnostic (Infinite + H5), accessible (CVD validé), réconcilié (Σ classes = total), zéro régression.

**Effort** : LOURD (7 phases, 5 surfaces, couche Go + composant + couleurs). Phases 0-1 = prérequis ; chaque
surface ensuite est livrable indépendamment.

---

## 1. Décisions produit — TRANCHÉES (ne pas rouvrir en exécution)

| # | Décision | Statut |
|---|---|---|
| D1 | Forme = **sunburst** 2 anneaux classe→rôle (pas donut simple) | ACTÉ (mockup) |
| D2 | Niveau 1 = axe **`class`** du registre (Épaule/Poing/Lourde/Mêlée/Grenade) | ACTÉ |
| D3 | **Non attribué** = résidu calculé serveur (Σ classes = total), tranche hachurée neutre | ACTÉ |
| D4 | **Mêlée** : sous-parts niveau 2 = **Assassinat** + **Corps-à-corps direct** (jamais « mêlée » répété). Feuille sur Infinite (pas de compteur natif). | ACTÉ |
| D5 | **Capacités spartanes** (H5-only, cap `native_kill_mechanics`) = classe à part = **Frappe au sol** + **Charge d'épaule** (disjointes de la mêlée) | ACTÉ |
| D6 | **Match view** : remplacer « Frags par arme » (pie) ET « Frags par technique » (donut kill-type, H5-only) par le **binôme sunburst + breakdown par arme en barres** | ACTÉ |
| D7 | **Timeseries** : monter sunburst + « Outils de destruction » sur le **même onglet = Résumé** ; recolorer « Outils de destruction » par classe | ACTÉ |
| D8 | **Escouade** : barres empilées horizontales **par CLASSE** (5-6 segments), pas par rôle (trop fin pour 4 joueurs) ; même échelle couleur | ACTÉ |
| D9 | **Explorer** : garde la v1 (donut SVG partagé) — INCHANGÉ, vérif iso | ACTÉ |
| D10 | Couleurs = **1 token sémantique par classe**, palette CVD-safe (Okabe-Ito), rôles = teintes de la classe, validées par script | ACTÉ |

---

## 2. DTO canonique (title-agnostic)

Nouveau type domaine, réutilisé par toutes les surfaces (sauf Escouade qui n'a besoin que du niveau classe).

```go
// internal/domain/frag_distribution.go (NOUVEAU)
type FragDistribution struct {
    TotalKills int                `json:"total_kills"`
    Classes    []FragClassEntry   `json:"classes"` // ordonné ; Σ Kills == TotalKills (inclut unattributed)
}
type FragClassEntry struct {
    Class         string           `json:"class"`          // shoulder|sidearm|heavy|melee|grenade|spartan_ability|unattributed
    Kills         int              `json:"kills"`
    Authoritative bool             `json:"authoritative"`  // true = totaux API canoniques ; false = estimé registre
    Roles         []FragRoleEntry  `json:"roles,omitempty"`// nil = feuille ; sinon Σ Kills(roles) == Kills
}
type FragRoleEntry struct {
    Role  string `json:"role"`  // precision|automatic|sniper|shotgun|special|sidearm|assassination|direct_melee|ground_pound|shoulder_bash
    Kills int    `json:"kills"`
}
```

**Invariants (testés)** : (a) `Σ Classes[i].Kills == TotalKills` ; (b) pour toute classe non-feuille,
`Σ Roles[j].Kills == Class.Kills` ; (c) `unattributed.Kills >= 0` ; (d) capability off → aucune classe
`spartan_ability`, Mêlée feuille (byte-équivalent à l'absence de mécaniques natives).

**Provenance des données** (le point anti-double-source) :
- Classes **`melee` / `grenade` / `spartan_ability`** + total : **stats canoniques API** (`total_melee_kills`,
  `total_grenade_kills`, `total_assassinations`, `total_ground_pound_kills`, `total_shoulder_bash_kills`,
  `total_kills`) → `Authoritative=true`. Voir `buildSynthesisDetailedStatsFromCanonical`
  (`internal/service/synthesis_service_builders.go:127-179`).
- Classes **`shoulder` / `sidearm` / `heavy`** + tous les rôles d'arme : **registre**, via `WeaponKillRow`
  (agrégation `v_weapon_kills`, exclut `IsGrenadeMelee` comme aujourd'hui pour role) → `Authoritative=false`.
- **`unattributed`** = `TotalKills − Σ(classes ci-dessus)`, ajouté seulement si `> 0`.
- **Mêlée niveau 2** (H5) : `assassination = total_assassinations` ; `direct_melee = melee_class_total − assassination`
  (formule à CONFIRMER par probe, cf. P2 gate G2.3).

---

## 3. Phases

> Ordre strict. Une étape close (gate vert) avant la suivante. Statuts : `[x]` fait / `[~]` couvert ailleurs
> (réf) / `[!]` non traité (justification). Aucune case vide à la clôture d'une phase.

### P0 — Fondation données Go (title-agnostic) — la brique partagée

Objectif : `Class` disponible dans la résolution d'arme partagée + `buildFragDistribution` pur + câblage Synthesis.

- [x] P0.1 `internal/platform/duckdb/weapon_resolver.go` : `class string` ajouté à `weaponResolved`,
      `COALESCE(w.class,'') AS class` au SELECT + scan (colonne appended en fin, ordre scan aligné). Jointure
      `LEFT JOIN weapons w` réutilisée → +1 colonne, 0 round-trip DB. VÉRIFIÉ : go test duckdb vert.
- [x] P0.2 `internal/port/weapon_kills.go` : `Class string` (json `class,omitempty`) ajouté à `WeaponKillRow` ;
      nom `ResolveRoles` conservé, doc mise à jour (« peuple Role ET Class en une passe »).
- [x] P0.3 `internal/platform/duckdb/weapon_kills_repo.go` : `attachWeaponMeta` pose `rows[i].Class = m.class`
      quand `withRoles` (bloc Role+Class regroupé ; sentinels 0/1 → Class "" comme avant).
- [x] P0.4 `internal/domain/frag_distribution.go` (NOUVEAU) : types §2 exacts (tags JSON) + constantes de classes/
      rôles canoniques + doc invariants.
- [x] P0.5 `internal/service/synthesis_service_builders.go` : `buildFragDistribution` PUR + helpers
      (`buildGunFragClasses`/`buildAPIFragClasses`/`meleeRoles`/`spartanRoles`/`rolesFromMap`). Ordre classes
      déterministe (`canonicalFragClassOrder`). **Signature étendue** : `+ totalKills int` (voir §6 Découverte D-P0-1).
- [x] P0.6 `internal/service/synthesis_service.go` : `loadTopWeaponKills` renvoie `*FragDistribution` (3e valeur) ;
      chargement rows extrait dans `loadWeaponKillRows` ; `hasMechanics` via `titleHasNativeKillMechanics(slug)`
      = `DefaultRegistry().Get(slug).HasCapability(CapNativeKillMechanics)` (pattern leaderboard, jamais slug==) ;
      câblé dans la réponse.
- [x] P0.7 `internal/domain/synthesis.go` : champ `FragDistribution *FragDistribution` (`frag_distribution,omitempty`)
      sur `SynthesisPageV2Response`. **`KillsByRole` conservé** (retrait P7).
- [x] P0.8 `apps/go-api/api/openapi.yaml` : schémas `FragDistribution`/`FragClassEntry`/`FragRoleEntry` + champ
      `frag_distribution` sur `SynthesisPageV2Response`. VÉRIFIÉ : présents dans `generated.ts` après generate-types.
- [x] P0.9 Tests (`synthesis_frag_distribution_test.go`, 5 tests) : invariants (a)(b)(c) via `assertFragInvariants` ;
      dataset hétérogène Infinite (cap off = pas de spartan + Mêlée feuille = invariant d) ; H5 (cap on = spartan +
      Mêlée niv.2) ; ordre canonique ; clamp assassination>melee ; résidu nul si attribution exacte. Tous verts.
- [x] P0.10 `logFragDistribution` : `slog.DebugContext` (compteurs classes/rôles/unattributed) + `slog.WarnContext`
      sur sur-comptage (Σ classes > total) — anomalie SIGNALÉE, jamais avalée.

**Gate P0** (séquentiel, PAS de builds go concurrents) :
```
cd apps/go-api && go test ./internal/service/... ./internal/platform/duckdb/... ./internal/domain/...
make generate-types   # openapi.yaml -> generated.ts (le front voit FragDistribution)
```

### P1 — Composant sunburst + couleurs accessibles (front, partagé)

- [ ] P1.1 Échelle couleur classe : helper `fragClassColor(class)` (lib/accessibility ou util chart) via
      `makeCategoricalScale` sur palette **Okabe-Ito** ; ordre FIXE par classe, **1 token/classe, zéro collision**
      (corrige le doublon mêlée=grenade=`chart-series-8` de `SynthesisRoleKillsDonut.tsx:28-29`). Non attribué =
      neutre hachuré. **Aucun hex dans features/components** (règle color-tokens).
- [ ] P1.2 Rôles = teintes de luminosité de la couleur de classe (double encodage : couleur + label + position).
- [ ] P1.3 **Validation CVD par script** (dataviz `validate_palette.js` + palette okabe-ito) sur light ET dark ;
      cible CVD ≥ 12 ; consigner le rapport dans le commit.
- [ ] P1.4 `FragSunburst.tsx` (NOUVEAU) : ECharts `sunburst`, consomme `FragDistribution`. Anneau interne=classe,
      externe=rôle, Non attribué hachuré, centre=total (ou segment survolé), tooltip = valeur + % du total + % de
      la classe parente + badge autorité (exact/estimé). Rendu `null` si total 0.
- [ ] P1.5 `FragWeaponBreakdown.tsx` (NOUVEAU ou généralisation de `SynthesisWeaponKillsChart.tsx`) : barres par
      arme, **couleur = `fragClassColor(class)`**. Nécessite `class` sur l'entrée arme → enrichir
      `SynthesisWeaponKillEntry` (`domain/synthesis.go:149-152`) de `class`/`role` (même passe `resolveWeaponMeta`).
- [ ] P1.6 i18n FR+EN partagé : labels de classe (`class_shoulder`=Épaule/Shoulder, `_sidearm`=Poing, `_heavy`=
      Lourde, `_melee`=Mêlée, `_grenade`=Grenade, `_spartan_ability`=Capacités spartanes, `_unattributed`=Non
      attribué) + rôles neufs (`assassination`=Assassinat, `direct_melee`=Corps-à-corps direct, `ground_pound`=
      Frappe au sol, `shoulder_bash`=Charge d'épaule). Emplacement partagé (réutilisé par 4 surfaces).
- [ ] P1.7 Tests front : typecheck ; rendu composant (mocker `echarts-for-react`, cf. reference echarts+jsdom) ;
      test garde-fou anti-collision (aucune classe → même token qu'une autre).

**Gate P1** : `make check-types && make test-web` (vitest hors sandbox, `dangerouslyDisableSandbox`) + rapport validateur couleur joint.

### P2 — Rollout Synthesis (surface de référence)

- [ ] P2.1 Remplacer `SynthesisRoleKillsDonut` (« Frags par type d'arme ») par `<FragSunburst>` dans
      `SynthesisPage.tsx` (`:313,736`). Adapter l'insight coach `weaponRoleInsight.ts` (garder blind_spot/over_reliance).
- [ ] P2.2 Brancher `FragWeaponBreakdown` (breakdown par arme recoloré par classe) sous le sunburst.
- [ ] P2.3 **Gate G2.3 — probe H5 (CLI TESTÉE, pas de sonde throwaway)** : via `probe-h5`, mesurer par joueur
      `Σ assassinats` vs count des Death `IsMelee` → trancher `assassination ⊆ melee` (formule `direct_melee`).
      Ajuster `buildFragDistribution` si disjoint. Consigner le verdict dans thought_log + §2.
- [ ] P2.4 Explorer INCHANGÉ : vérifier que `ExplorerTargetSampleStats`/`KillTypesDonut` restent byte-équivalents.

**Gate P2** : gates P0+P1 verts ; revue visuelle Synthesis (Infinite + H5) ; probe consignée.

### P3 — Rollout Match view

- [ ] P3.1 Plomber `class` dans le repo SÉPARÉ `MatchViewRepo.GetMatchBulkWeaponKills`
      (`internal/service/match_view_repo_weapons.go:107,161-166`) — `BulkWeaponKillRaw` expose déjà label ; ajouter
      class/role (son `resolveWeaponMeta` renvoie déjà role → étendre à class).
- [ ] P3.2 Construire `FragDistribution` par-match (viewer) : gun classes = bulk weapon kills ; melee/grenade/
      spartan = `MatchScoreboardRow` (`types.ts:1621-1677`, colonnes natives déjà présentes).
- [ ] P3.3 Remplacer `MatchWeaponPieChart` (`MatchViewPage.tsx:353`) ET `MatchKillTypesDonut` (`:356`, H5-only) par
      `<FragSunburst> + <FragWeaponBreakdown>`.
- [ ] P3.4 Supprimer le code mort : `MatchWeaponPieChart`, `MatchKillTypesDonut` + leurs i18n si plus référencés (règle 0 code mort).

**Gate P3** : `make check-types && make test-web` ; revue visuelle match Infinite + H5.

### P4 — Rollout Timeseries

- [ ] P4.1 `timeseries_service.go` (`:216-234`) : passer `ResolveRoles=true` ; ajouter class ; construire
      `FragDistribution` (kill-type via `buildTimeseriesKillTypes` `timeseries_service_tabs.go:248-269` + gun via registre).
- [ ] P4.2 Remplacer les DEUX variantes kill-type (`TimeseriesKillTypesDonut` H5 `summary.tsx:140` + `KillTypesDonutCard`
      Infinite `progression.tsx:147`) par `<FragSunburst>` sur l'onglet **Résumé** (D7).
- [ ] P4.3 Recolorer « Outils de destruction » (`TimeseriesTopWeapons`) par classe (enrichir `TimeseriesWeaponKill`
      de `class`) — même onglet Résumé, sous/à côté du sunburst.
- [ ] P4.4 Supprimer le montage kill-type de l'onglet Progression (code mort si plus utilisé).

**Gate P4** : `make check-types && make test-web` ; revue visuelle Timeseries (onglet Résumé, Infinite + H5).

### P5 — Rollout Sessions (LE PLUS LOURD — nouveau chemin de données)

- [ ] P5.1 L'endpoint `sessions/detail` (`internal/service/sessions_service.go:27`) **ne charge AUCUNE donnée
      d'arme ni total kill-type** aujourd'hui. Ajouter : agrégation `WeaponKillRow` (gun classes) + totaux
      kill-type (melee/grenade/spartan) sur le scope de la session → nouveau champ `FragDistribution` sur
      `SessionCompareEntry`/`SessionPageResponse` (`generated.ts:7695`).
- [ ] P5.2 Front : nouvelle carte `<FragSunburst> (+ breakdown)` insérée **après** `SessionDamageComposite`
      (dernier de la pile, `SessionChartStack.tsx:159` branche dense ET `:193` branche défaut — insérer aux DEUX).
- [ ] P5.3 Query key inchangée (même endpoint) ; i18n titre de carte.

**Gate P5** : gate go (service+duckdb) + `make check-types && make test-web` ; revue visuelle Sessions (les 2 branches : drawer compact + pleine page).

### P6 — Rollout Escouade (barres empilées par classe)

- [ ] P6.1 Backend : `kills_by_class` PAR gamertag sur `SquadPerformanceSeriesPoint` (`generated.ts:8167-8214`),
      via le `loadWeapons` squad (`squad_service_v2.go:272-304`, `IncludeGrenadeMelee:true`) + `class`.
- [ ] P6.2 Généraliser `squadFragBreakdownChart.ts` : de 4 segments figés (`SegmentKey` `:40-48`) à **N classes
      dynamiques** ; labels de classe (i18n P1.6) ; couleurs `fragClassColor` (remplace tokens kill-type `:43-48`).
      GARDER la forme (barres horizontales, 4 joueurs, `inverse` main-en-haut, `stack:'frags'`, `barMaxWidth:18`).
- [ ] P6.3 Wiring i18n squad (`squad/i18n.ts:473-476`) : labels classe.

**Gate P6** : `make check-types && make test-web` ; revue visuelle Escouade (1 à 4 joueurs).

### P7 — Nettoyage, garde-fous, livraison

- [ ] P7.1 Retirer `KillsByRole` du DTO + `SynthesisRoleKillsDonut` + `weaponRoleInsight` si absorbés (0 code mort,
      supprimer tests+imports).
- [ ] P7.2 Garde-fou (règle ≤2 copies) : test grep interdisant tout mapping direct classe→token hors
      `fragClassColor` ; test anti-collision de token maintenu.
- [ ] P7.3 `internal/sync/no_art_patterns_test.go` inchangé (aucune écriture per-match introduite) — vérifier.
- [ ] P7.4 Tests intégration (`go test -tags=integration ./...`, `-p 1`, filtre ancré) si toute couche persist touchée — sinon `[~]`.
- [ ] P7.5 `.ai/thought_log.md` : entrée par phase (obligatoire). `.ai/project_map.md` MAJ si structure.
- [ ] P7.6 Skill `delivery-checklist` avant « livré ».

**Gate P7** : suite complète `cd apps/go-api && go test ./...` + `make check-types && make test-web` + revue visuelle des 5 surfaces + Explorer iso.

---

## 4. Taxonomie classe → rôle (référence d'exécution)

| Classe (niv.1) | Label FR | Rôles (niv.2) | Source | Autorité |
|---|---|---|---|---|
| `shoulder` | Épaule | precision, automatic, (shotgun=Bulldog), (special=Needler) | registre | estimé |
| `sidearm` | Poing | — (feuille) | registre | estimé |
| `heavy` | Lourde | power, sniper, (shotgun=Heatwave), (special=Sentinel Beam) | registre | estimé |
| `melee` | Mêlée | Infinite: feuille · H5: assassination + direct_melee | API canonique | exact |
| `grenade` | Grenade | — (feuille) | API canonique | exact |
| `spartan_ability` | Capacités spartanes | ground_pound + shoulder_bash | API canonique | exact |
| `unattributed` | Non attribué | — (feuille, hachuré) | résidu calculé | — |

- `spartan_ability` + rôles de `melee` : **capability-gated `native_kill_mechanics`** (H5 only). Gating via
  `HasCapability`, JAMAIS `slug ==` (ratchet `no_slug_comparison_test.go`). Infinite : Mêlée feuille, pas de Spartan.
- **Rôles trans-classes** (`shotgun`, `special` apparaissent sous shoulder ET heavy) : normal — ce sont des arcs
  distincts sous des parents différents (Bulldog sous Épaule, Heatwave sous Lourde). Tooltip désambiguïse par parent.
- Source registre : `weapon_registry.go` colonnes `class`/`role` (`:71-72`), déjà en RAM (`weaponregistry/registry.go:15`).

---

## 5. Exécution & reprise

- **Contrat** : skill `plan-execution` (fait foi). Ordre strict, gate vert avant étape suivante, statuer chaque item.
- **Interdiction** : fix opportuniste hors périmètre → consigner en §6 Découvertes, ne pas traiter.
- **Builds go** : JAMAIS concurrents (cache Windows) — un seul `go test`/`build` à la fois, séquentiel.
- **Reprise de session** : lire les cases `[x]/[~]/[!]` de la phase courante + dernière entrée `.ai/thought_log.md`
  + `git log --oneline` de `feat/frag-distribution-v2`. Reprendre à la première case vide de la première phase non close.
- **Pilotage** : exécution déléguée à des agents Opus (le superviseur garde git/merges/CI/mémoire) ; revue au merge.

## 6. Découvertes (à remplir en cours d'exécution — ne pas traiter hors périmètre)

- **D-P0-1 (signature `buildFragDistribution`)** : la signature littérale du plan
  (`rows, stats domain.SynthesisDetailedStats, hasMechanics bool`) NE PORTE PAS le total : `SynthesisDetailedStats`
  n'a aucun champ `total_kills`, or §2 exige le total depuis l'API (`total_kills`) pour calculer le résidu
  `unattributed = total − Σ classes`. Ajouté un paramètre `totalKills int` (câblé depuis `overview.TotalKills`,
  déjà calculé). Micro-décision tranchée par §2 (contrat DTO fait foi) — signaler pour cohérence des phases suivantes.
- **D-P0-2 (buckets non-combat H5 → Non attribué)** : le registre porte des classes hors sunburst
  (`vehicle`/`turret`/`environmental`/`unattributed`/`other`, H5 hors-arsenal). `buildFragDistribution` ne retient
  comme classes gun que `shoulder`/`sidearm`/`heavy` (§2) ; ces frags non-combat retombent donc dans « Non attribué »
  (résidu). Conforme à la taxonomie §4 (7 classes only) et à D3, mais à valider visuellement en P2 (part de « Non
  attribué » potentiellement notable sur les scopes H5 riches en véhicules). NON traité (hors P0).
- **D-P0-3 (over-count possible)** : si `Σ classes attribuées > total` (anomalie de données), le résidu serait
  négatif → non ajouté (clamp, invariant c préservé) mais l'invariant (a) `Σ==total` n'est alors PAS tenu.
  `logFragDistribution` émet un `WARN` dédié. À surveiller en prod ; aucune action P0.
