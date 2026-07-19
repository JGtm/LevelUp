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
- **Mêlée niveau 2** (H5) : `assassination = total_assassinations` ; `direct_melee = total_melee_kills`. Les deux
  compteurs API sont **DISJOINTS** (probe empirique tranchée 2026-07-19 : `melee_kills` n'inclut PAS les assassinats).
  Total de la classe Mêlée = `melee + assass` (SANS soustraction ni clamp) ⇒ `Σ rôles == total classe` (invariant b).
  Preuve : match `3066a511-ebd0-428f-9555-50422caebaba`/xuid `2535421586125737` → melee=6, assass=4, per-kill 6+4 ;
  match `9bb09267…` melee=2 < assass=4 (impossible sous inclusion) ; 1212/1213 couples disjoints. (Ancienne formule
  `direct = melee − assassination` supposait à tort l'inclusion — corrigée.)

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

- [x] P1.1 Échelle couleur classe : `apps/web/src/lib/accessibility/scales/fragClass.ts` (SOURCE UNIQUE) +
      `fragClassColors.ts` (NOUVEAU — hex fixes). Ordre FIXE `FRAG_CLASS_ORDER` ; `fragClassColor(class)` sert des
      **hex fixes CVD-safe INDÉPENDANTS de la palette active** (option A, tranchée user 2026-07-19) : shoulder
      #0072B2 / sidearm #E69F00 / heavy #56B4E9 / melee #D55E00 / grenade #009E73 / spartan_ability #F0E442 (teintes
      Okabe-Ito), `unattributed` = neutre #888888 rendu HACHURÉ côté sunburst. **Motif** : sous la palette DÉFAUT,
      `chart-series-1..5` sont une rampe indigo (ΔE ~5.4, SOUS le plancher CVD) → 5 classes indistinguables ; sortir
      de la palette active (précédent `rarity.ts`) garantit la distinction quelle que soit la palette. **7 hex
      distincts** (collision mêlée=grenade éradiquée). `fragClassScale`/`fragClassColorToken` (palette-dépendants)
      SUPPRIMÉS (0 code mort). Exporté via `scales/index.ts`. Hex UNIQUEMENT dans `fragClassColors.ts` (exception
      documentée), zéro hex dans features/components. VÉRIFIÉ : check-types + guard test verts.
- [x] P1.2 Rôles = teintes de luminosité (`fragRoleColor(class, index, count)` via `shiftLightness`, spread 0.32,
      1er clair→dernier foncé). Double encodage couleur + label + position (anneau/segment).
- [x] P1.3 **Validation CVD par le validateur dataviz `validate_palette.js`** (all-pairs, sur les 6 hex FIXES) :
      **INCONDITIONNELLE** (hex hors palette active) — normal-vision 15.58 PASS ; protan 11.38 PASS ; deutan **10.98**
      (worst, melee/grenade) PASS (cible validateur 8). Cible plan ≥12 non atteinte (max structurel des 6 teintes Okabe ;
      voir §6 D-P1-1) — couvert par l'encodage secondaire (label+position+hachure). Report consigné (thought_log).
      Garde-fou in-repo : `fragClass.guard.test.ts` (anti-collision hex + pin des 6 hex Okabe + palette-indépendance +
      min all-pairs ΔE ≥ 8, math Machado 2009). RÉSOUT l'exposition à la rampe indigo de `palettes/default.ts`.
- [x] P1.4 `apps/web/src/components/charts/FragSunburst.tsx` (NOUVEAU) : ECharts `sunburst` 2 anneaux, consomme le
      type généré `FragDistribution`. Anneau interne=classe, externe=rôle, `unattributed` hachuré (decal), centre=total
      (graphic centré), tooltip = valeur + % du total + % de la classe parente (niv.2) + badge autorité (exact/estimé),
      `sort:undefined` (ordre canonique conservé). Rend `null` si total 0. Builder pur `buildFragSunburstOption` exporté.
- [x] P1.5 `apps/web/src/components/charts/FragWeaponBreakdown.tsx` (NOUVEAU) : barres horizontales par arme,
      **couleur = `fragClassColor(w.class)`** (per-datum), classe dans le tooltip. Go : `SynthesisWeaponKillEntry`
      (`domain/synthesis.go`) enrichi de `class`/`role` (json omitempty) ; peuplé dans `buildTopWeaponKills`
      (rows déjà `ResolveRoles=true`) ; schéma `openapi.yaml` + `make generate-types` (class?/role? dans generated.ts).
- [x] P1.6 i18n FR+EN **manifest PARTAGÉ NOUVEAU** `apps/web/src/lib/i18n/manifests/frags.toml` (→ `fragsManifest`,
      26 clés) : classes (`frags.class.*` Épaule/Poing/Lourde/Mêlée/Grenade/Capacités spartanes/Non attribué) + rôles
      registre (`frags.role.precision..special`) + rôles neufs (`assassination`/`direct_melee`/`ground_pound`/
      `shoulder_bash`) + titres cartes + badge autorité + fragments tooltip. Réutilisable par les 4 surfaces.
- [x] P1.7 Tests front (`FragSunburst.test.tsx`, `FragWeaponBreakdown.test.tsx`, `fragClass.guard.test.ts`, 12 tests) :
      builders purs (hiérarchie, résidu hachuré, null si total 0, recolor par classe) ; rendu composant (mock
      `echarts-for-react`) ; **anti-collision** (7 tokens distincts) + pin du mapping validé CVD + CVD ≥ 8.

**Gate P1** : `go test ./internal/service/... ./internal/domain/...` PASS ; `make generate-types` PASS ;
`make check-types` PASS ; `make test-web` **278 fichiers, 2392 tests PASS / 14 skipped** ; validateur couleur
all-pairs deutan 11.0 PASS (report ci-dessus). Composant NON monté sur page (c'est P2). Aucun commit.

### P2 — Rollout Synthesis (surface de référence)

> **Note de fusion (vaut pour P3-P6)** : le sunburst v2 unifié REMPLACE **tous** les anciens graphes de frags que
> chaque surface montait (donut kill-type + donut rôle), et `FragWeaponBreakdown` remplace l'ancien graphe par-arme.
> Donc chaque surface RETIRE les anciennes cartes que le sunburst/breakdown subsument — **pas seulement celle
> nommée dans l'item**. Corollaire du §0 (« fusionner les deux donuts ») acté avec l'utilisateur.

- [x] P2.1 Les DEUX anciens donuts de frags remplacés par le sunburst unifié `<FragSunburst>` (alimenté par
      `data.frag_distribution`), dans la nouvelle carte de composition `SynthesisFragCard.tsx` (feature) : sunburst +
      insight coach PRÉSERVÉ (lit `kills_by_role` via `weaponRoleInsight`, blind_spot/over_reliance inchangés).
      SUPPRIMÉS (0 réf restante hors commentaires, grep vert) : `SynthesisRoleKillsDonut.tsx` (+ son test) ET
      `SynthesisKillTypesDonut.tsx` (donut kill-type « Répartition des frags », subsumé par le sunburst — pas de test
      dédié). Partagés `KillTypesDonutCard`/`KillTypesDonut` CONSERVÉS (utilisés par Timeseries/Match/Explorer).
      `weaponRoleInsight.ts` conservé (consommé par `SynthesisFragCard`). Champ `frag_distribution?` ajouté à
      l'interface hand-written `SynthesisPageResponse` (types.ts) — absent après P0 (cf. §6 D-P2-1). `KillsByRole` DTO
      CONSERVÉ (retrait P7).
- [x] P2.2 `FragWeaponBreakdown` monté SOUS le sunburst (dans `SynthesisFragCard`), alimenté par `data.top_weapon_kills`
      (enrichi class/role en P1) — barres recolorées par classe. L'ancien bar chart par-arme `SynthesisWeaponKillsChart`
      (« Frags par arme », subsumé) SUPPRIMÉ (0 réf restante hors commentaires, pas de test dédié). La section frags de
      Synthesis = `SynthesisFragCard` UNIQUEMENT.
- [x] P2.3 **Gate G2.3 — TRANCHÉ : compteurs DISJOINTS** (probe empirique 2026-07-19, verdict opposé à l'hypothèse
      initiale d'inclusion). `melee_kills` et `assassination_kills` sont des compteurs API SÉPARÉS : `melee_kills`
      n'inclut PAS les assassinats. Preuve per-kill : match `3066a511-ebd0-428f-9555-50422caebaba`/xuid
      `2535421586125737` → melee_kills=6, assassination_kills=4, décompte per-kill = 6 mêlées + 4 assassinats
      (aucun chevauchement) ; contre-preuve d'inclusion : match `9bb09267…` a melee_kills=2 < assassination_kills=4
      (impossible si assass ⊆ melee) ; 1212/1213 couples disjoints sur l'échantillon. **Correctif appliqué**
      (`fragdist.go`) : total classe Mêlée = `melee + assass` ; `assassination = counts.Assassination`,
      `direct_melee = counts.Melee` SANS soustraction ni clamp ; `Σ rôles == total classe`. Conséquence : les
      assassinats — auparavant perdus (non comptés dans aucune classe → gonflaient « Non attribué ») — sont désormais
      attribués → « Non attribué » DIMINUE (correct). D-P2-2 CLOS.
- [x] P2.4 Explorer INCHANGÉ : `git diff` ne touche AUCUN fichier Explorer ni `KillTypesDonut.tsx` (composant SVG
      partagé) ni `ExplorerTargetSampleStats` — byte-équivalents. Seul `types.ts` change (champ optionnel additif, 0
      impact comportemental Explorer).

**Gate P2** : gates P0+P1 verts ; `make check-types` PASS ; `make test-web` **277 fichiers, 2390 PASS / 14 skipped**
(re-gate après retrait des doublons ; counts inchangés — `SynthesisKillTypesDonut`/`SynthesisWeaponKillsChart`
n'avaient pas de test dédié ; le −1 fichier vs P1 = `SynthesisRoleKillsDonut.test.tsx`). D-P2-3 RÉSOLU (fusion :
section frags = `SynthesisFragCard` seule). Revue visuelle Synthesis (Infinite + H5) RESTE À FAIRE (gate humain).
Probe consignée (D-P2-2). Aucun commit.

### P3 — Rollout Match view

- [x] P3.1 `class`/`role` plombés dans `BulkWeaponKillRaw` (`domain/match_view_raw.go`) + `weaponMetaEntry`
      (class/role) + `GetMatchBulkWeaponKills` (`platform/duckdb/match_view_repo_weapons.go`) : `lookupWeaponMeta`
      porte désormais class/role (résolus par `resolveWeaponMeta`, déjà class+role — P0.1) et les pose sur chaque row.
- [x] P3.2 `buildViewerFragDistribution` (`service/match_view_builders_combat.go`) RÉUTILISE `buildFragDistribution`
      (P0). Signature du builder GÉNÉRALISÉE : `stats domain.SynthesisDetailedStats + totalKills int` →
      `counts domain.FragKillTypeCounts{Melee,Grenade,Assassination,GroundPound,ShoulderBash,Total}` (struct neutre,
      §6 D-P3-1) ; appelant Synthesis + 5 tests adaptés (0 duplication, règle ≤2 copies). gun classes = bulk weapon
      kills du viewer ; melee/grenade/spartan + total = ligne scoreboard native `MatchScoreboardRow` (is_me).
      hasMechanics = `titleHasNativeKillMechanics` (capability). Champ `frag_distribution` sur `MatchCombatTab` :
      Go domain + `openapi.yaml` + `generated.ts` + **interface hand-written `MatchCombatTab` types.ts** (D-P2-1) ;
      câblé sur les DEUX voies (repo `buildMatchViewFromData` + canonique live `buildMatchViewFromCanonical`).
      `class` ajouté à `MatchWeaponKill` (Go+openapi+types.ts) pour recolorer le breakdown.
- [x] P3.3 `MatchWeaponPieChart` ET `MatchKillTypesDonut` remplacés par `MatchFragCard` (composition sunburst +
      breakdown, calque `SynthesisFragCard`) dans `MatchViewPage.tsx`. Non gaté (Infinite classes sans Spartan ;
      H5 avec). `killTypeFallback` + `weaponData` + imports orphelins retirés. Cas vide géré (rend null).
- [x] P3.4 `MatchWeaponCharts.tsx` (seul export `MatchWeaponPieChart`) + `MatchKillTypesDonut.tsx` SUPPRIMÉS (0 réf,
      pas de test dédié). i18n orphelins retirés (`chartWeaponPieTitle`/`chartKillTypesTitle`/`labelPowerWeapon`/
      `labelMelee`/`labelOtherKills`/`weaponOtherGroup`) ; CONSERVÉS `labelGrenade`/`labelAssassination`/
      `labelGroundPound`/`labelShoulderBash` (encore utilisés par `MatchScoreboard.tsx`) + `weaponUnknownPrefix`
      (dette pré-existante hors périmètre, §6 D-P3-2). Partagés `KillTypesDonut`/`KillTypesDonutCard` NON touchés.

**Gate P3** : `go test ./internal/service/... ./internal/domain/...` PASS (service 10.7s + domain OK ; duckdb build OK) ;
`make generate-types` PASS ; `make check-types` PASS (tsc exit 0) ; `make test-web` **277 fichiers, 2390 PASS / 14 skipped**
(counts inchangés vs P2 : composants supprimés sans test dédié, MatchFragCard couvert par les tests P1 des composants
partagés). Revue visuelle match Infinite + H5 RESTE À FAIRE (gate humain). Aucun commit.

### P4 — Rollout Timeseries

- [x] P4.1 `timeseries_service.go` : filtres weapon_kills passés à `ResolveRoles=true` (résout Role+Class en
      une passe, `rows` hoistées `weaponRows`) ; nouvelle méthode `buildTimeseriesFragDistribution(ctx, weaponRows, kt)`
      qui RÉUTILISE `buildFragDistribution` (0 duplication) — `counts` = `FragKillTypeCounts` rempli depuis
      `resp.KillTypes` (`buildTimeseriesKillTypes`), gun classes = `weaponRows`, `hasMechanics = titleHasNativeKillMechanics`
      (capability, jamais slug==). slog Debug (compteurs) + Warn sur sur-comptage (jamais avalé). Champ
      `FragDistribution` ajouté à `TimeseriesPageResponse` : Go domain + `openapi.yaml` + `make generate-types`
      (généré, pas d'interface hand-written — `TimeseriesPageResponse` est un alias `components['schemas'][...]`).
- [x] P4.2 `TimeseriesKillTypesDonut` (Résumé) ET le montage `KillTypesDonutCard` (Progression) remplacés par un
      UNIQUE `<FragSunburst distribution={data.frag_distribution} />` sur l'onglet **Résumé** (D7). Title-agnostic
      (FragSunburst rend null si total 0 ; Infinite sans spartan, H5 avec — décidé backend par capability).
- [x] P4.3 « Outils de destruction » (`TimeseriesTopWeapons`) recoloré PAR CLASSE : `class` ajouté à
      `TimeseriesWeaponKill` (Go+openapi+généré), peuplé dans `buildTopWeapons` (passe registre P4.1) ; barres colorées
      per-datum via `fragClassColor(w.class)` (remplace le token unique `chart-series-1`). Chart CONSERVÉ sur Résumé.
- [x] P4.4 `TimeseriesKillTypesDonut.tsx` SUPPRIMÉ (0 réf après P4.2, pas de test dédié) ; montage kill-type retiré de
      Progression (var `hasKillMechanics` + import `KillTypesDonutCard` retirés). Partagés `KillTypesDonutCard`/
      `KillTypesDonut` NON supprimés : `KillTypesDonut` reste utilisé par Explorer (`ExplorerTargetSampleStats`) ;
      `KillTypesDonutCard` devient orphelin (Timeseries était son dernier consommateur) mais laissé en place (§6 D-P4-1,
      retrait = P7). `data.detailed_stats` n'est plus consommé côté Timeseries (backend inchangé, §6 D-P4-2).

**Gate P4** : `go test ./internal/service/... ./internal/domain/...` PASS (service 10.1s + domain OK) ; gofmt Go clean ;
`make generate-types` PASS ; `make check-types` PASS (tsc exit 0) ; `make test-web` **277 fichiers, 2390 PASS / 14 skipped**
(counts inchangés vs P3 : `TimeseriesKillTypesDonut` supprimé sans test dédié, `FragSunburst`/`FragWeaponBreakdown`/recolor
couverts par les tests P1). Revue visuelle Timeseries (onglet Résumé, Infinite + H5) RESTE À FAIRE (gate humain). Aucun commit.

### P5 — Rollout Sessions (LE PLUS LOURD — nouveau chemin de données)

- [x] P5.1 L'endpoint `sessions/detail` est servi par `SessionPageService.GetPage` (`session_page_service.go`,
      PAS `sessions_service.go` qui sert la LISTE — cf. §6 D-P5-1). Nouveau chemin de données câblé sans dupliquer
      le builder : `session_page_frag_distribution.go` (NOUVEAU) agrège, sur les match IDs de la session
      sélectionnée, (a) les compteurs kill-type API (melee/grenade/assassination/ground_pound/shoulder_bash + total)
      depuis les rows canoniques (déjà chargées dans `GetPage`, filtrées par ID) via
      `buildSynthesisDetailedStatsFromCanonical` ; (b) les classes/rôles d'arme + top armes via
      `LoadWeaponKillsAggregated(MatchIDs, Gamertag, ResolveRoles:true)`. Puis RÉUTILISE `buildFragDistribution`
      (0 duplication). `hasMechanics = titleHasNativeKillMechanics(slug)` (capability, jamais slug==). Champs
      `FragDistribution` + `TopWeaponKills` ajoutés à `SessionCompareEntry` : Go domain + `openapi.yaml` +
      `make generate-types`. **Interface hand-written** : `SessionCompareEntry` (types.ts:1849) EST
      `components['schemas']['SessionCompareEntry'] & {...}` → les 2 champs arrivent AUTOMATIQUEMENT via le schéma
      généré (contraste avec D-P2-1 où `SynthesisPageResponse` était 100 % hand-written) — 0 édition types.ts.
      Repo câblé `WithWeaponKillsRepo` (`registry_pages.go`). Attaché à la session courante ET à la session comparée.
- [x] P5.2 Front : `SessionFragCard.tsx` (NOUVEAU, calque `SynthesisFragCard`/`MatchFragCard` : `<FragSunburst>` +
      `<FragWeaponBreakdown>`, rend null si pas de données) alimentée par `entry.frag_distribution` +
      `entry.top_weapon_kills`. Insérée **JUSTE APRÈS** `SessionDamageComposite` (`damage`) aux DEUX branches de
      `SessionChartStack.tsx` : branche dense (après `{damage}`) ET branche défaut (après `{damage}`).
- [x] P5.3 Query key inchangée (même endpoint `/pages/sessions/detail`, `queries.ts` non touché). Titre de carte
      i18n FR+EN via le manifeste PARTAGÉ `frags` (`frags.charts.sunburst_title` / `weapon_breakdown_title`, défauts
      de `FragSunburst`/`FragWeaponBreakdown`) — comme la carte de référence Synthesis (pas de clé `session.toml` ajoutée).

**Gate P5** : `go test ./internal/service/... ./internal/domain/...` PASS (service 10.4s, domain cached) ; gofmt Go
touchés clean ; `make generate-types` PASS (generated.ts +2 lignes = les 2 champs SessionCompareEntry) ;
`make check-types` PASS (tsc exit 0) ; `make test-web` **277 fichiers, 2390 PASS / 14 skipped** (counts inchangés vs
P4 : `SessionFragCard` couvert par les tests P1 des composants partagés, pas de test dédié). Revue visuelle Sessions
(les 2 branches : drawer compact + pleine page, Infinite + H5) RESTE À FAIRE (gate humain). Aucun commit.

### P6 — Rollout Escouade (barres empilées par classe)

- [x] P6.1 Backend : ventilation PAR CLASSE par gamertag exposée en `frag_classes` (map gamertag→`[]FragClassEntry`,
      niveau 1 seulement, D8) sur **`TeammatesPageResponse`** (PAS `SquadPerformanceSeriesPoint` ni `squad_service_v2`
      — §6 D-P6-3 : le chart Escouade est servi par `TeammatesService`/`/pages/teammates`, code fait foi). RÉUTILISE
      `fragdist.Build` par joueur (0 duplication) : gun classes = weapon kills par joueur (`buildSquadWeaponKills` +
      `ResolveRoles:true` pour peupler `class`) ; counts melee/grenade/total = agrégat de `PerformanceSeries` du joueur
      (`aggregateFragCounts`) ; `hasMechanics=false` (§6 D-P6-2). Champ : Go domain (`teammates.go`) + `openapi.yaml` +
      `make generate-types` + interface hand-written `TeammatesPageResponse` (`types.ts`, D-P2-1). **Builder extrait**
      package leaf `internal/service/fragdist` (§6 D-P6-1) — 5 appelants service + test migrés vers `fragdist.Build`.
- [x] P6.2 `squadFragBreakdownChart.ts` généralisé : de 4 segments FIGÉS à **N classes DYNAMIQUES** (union présente,
      ordre canonique `FRAG_CLASS_ORDER`). Input = `Record<string, FragClassEntry[]>` (`frag_classes`). Couleurs
      `fragClassColor(class)` (per-datum), labels via `classLabel` (manifeste `frags`). Forme GARDÉE : barres
      horizontales empilées, `inverse` main-en-haut, `stack:'frags'`, `barMaxWidth:18`, tooltip Total. `aggregate()`
      remplacé par `killsByClass`/`presentClasses`. Test adapté à la taxonomie par classe (6 tests, verts).
- [x] P6.3 i18n : les 4 labels FR/EN figés de `squad/i18n.ts` (meleeLabel/powerWeaponLabel/grenadeLabel/otherLabel)
      SUPPRIMÉS (type + FR + EN) ; labels de classe résolus dans `SquadPerformanceCharts` via le manifeste PARTAGÉ
      `frags` (`frags.class.*`) — même source que le sunburst. `fragBreakdownTitle` (titre de carte) CONSERVÉ.

**Gate P6** : `go test ./internal/service/... ./internal/domain/...` PASS (service 8.0s + teammates 0.44s + domain cached ;
`fragdist` [no test files] — builder couvert par les tests P0 restés en package service) ; gofmt Go touchés clean ;
`make generate-types` PASS (`frag_classes` dans generated.ts) ; `make check-types` PASS (tsc exit 0) ; `make test-web`
**277 fichiers, 2390 PASS / 14 skipped** (counts inchangés vs P5 : test frag breakdown réécrit 6→6 tests). Revue
visuelle Escouade (1 à 4 joueurs, Infinite + H5) RESTE À FAIRE (gate humain). Aucun commit.

### P7 — Nettoyage, garde-fous, livraison

- [x] P7.1 Code mort retiré (0 zombie) : (a) **`KillsByRole` DTO éradiqué** (préféré au maintien) — `weaponRoleInsight`
      migré pour dériver les rôles des GUN CLASSES de `frag_distribution` (logique EXACTEMENT préservée via
      `insightFromRoles` : NON_COMBAT_WEAPON_ROLES + MIN_KILLS + power<3% + top>70%) ; supprimés `SynthesisRoleKillEntry`
      + `buildKillsByRole` + `synthesis_role_kills_test.go` + champ openapi/types/generated + prop front `killsByRole`.
      `SynthesisRoleKillsDonut`/`SynthesisKillTypesDonut` déjà supprimés en P2 (pas de reliquat). (b) **D-P4-1**
      `KillTypesDonutCard.tsx` orphelin SUPPRIMÉ (`KillTypesDonut` SVG conservé — Explorer+Match). (c) **D-P4-2**
      `TimeseriesPageResponse.DetailedStats` (0 consommateur back+front) RETIRÉ (domain + peuplement + openapi + generated).
      (d) **Note P6** 5 tests builder co-localisés dans `fragdist/fragdist_test.go` (package `fragdist`).
- [x] P7.2 Garde-fous ajoutés : (i) `fragClass.colorSource.guard.test.ts` (grep — aucun hex de classe de frags en dur
      sous features/components, source unique `fragClassColor`) ; (ii) `frag_distribution_log_guard_test.go` (grep — log
      FragDistribution centralisé dans `logFragDistribution`, marqueurs interdits ailleurs, D-P5-2). Anti-collision
      `fragClass.guard.test.ts` maintenu et vert.
- [x] P7.3 `internal/sync/no_art_patterns_test.go` INCHANGÉ (git clean) et vert dans la suite — aucune écriture per-match.
- [~] P7.4 Couche persist NON touchée (lectures/agrégations only ; edits migrations = commentaires seuls) → intégration
      NON requise. Couvert par la suite Go COMPLÈTE (`go test ./...`) verte + `no_art_patterns` inchangé (P7.3).
- [x] P7.5 `.ai/thought_log.md` : entrée P7 ajoutée. `.ai/project_map.md` NON modifié — doc explicitement GELÉE
      (« HISTORIQUE — NE FAIT PLUS FOI », 0 réf frags) : le code + thought_log + ADRs font foi (carto ne le mérite pas).
- [x] P7.6 Skill `delivery-checklist` invoqué → verdict **GO (local)** consigné (thought_log). CI branche N/A (non poussé).

**Gate P7** (suite COMPLÈTE, séquentiel, exécuté 2026-07-19) : `cd apps/go-api && go test ./...` **PASS** (tous packages
`ok`, 0 FAIL, exit 0) ; `make generate-types` PASS ; `make check-types` PASS (tsc exit 0) ; `make test-web` **278 fichiers,
2398 PASS / 14 skipped** ; `go vet ./...` exit 0 ; `golangci-lint` packages touchés **baseline-neutre** (1 gofmt introduit
sur synthesis.go → CORRIGÉ). RESTE (gates humains, hors code) : revue visuelle des 5 surfaces + Explorer iso ; probe
pré-prod P2.3 (`[!]`, D-P2-2). Aucun commit.

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
- **D-P1-1 (palette DÉFAUT non catégorielle sur 6 séries) — RÉSOLU (révision option A, 2026-07-19)** : le problème
  était que `fragClassColor` sourçait `chart-series-1..6` via la palette ACTIVE ; sous la palette **DÉFAUT**,
  `chart-series-1..5` sont une RAMPE indigo (all-pairs ΔE 5.4 — FAIL catégoriel), donc 5 classes de frags se
  ressemblaient. **Correctif** : les classes de frags ont désormais leur PROPRE jeu de **hex fixes Okabe-Ito
  INDÉPENDANT de la palette active** (`fragClassColors.ts`, précédent `rarity.ts`) → l'exposition à la rampe indigo
  du défaut DISPARAÎT pour ce chart, quelle que soit la palette. La rampe indigo de `palettes/default.ts` reste une
  propriété du défaut partagée par les autres charts (HORS périmètre), mais ne touche plus les frags.
- **D-P1-2 (cible CVD 12 vs 11.0) — RÉSOLU** : sur les 6 hex FIXES, worst all-pairs deutan **10.98** (protan 11.38,
  normal 15.58) PASSE le validateur (cible 8) de façon INCONDITIONNELLE (plus de dépendance palette). Cible plan ≥12
  = plafond structurel des 6 teintes Okabe (la dépasser exigerait de dropper Vert/Vermillion → casse le floor
  normal-vision). Décision retenue : 10.98 + encodage secondaire (label + position d'anneau + hachure du résidu).
- **D-P2-1 (interface front `SynthesisPageResponse` non mise à jour en P0)** : P0.8 a ajouté `frag_distribution` au
  schéma OpenAPI et au type GÉNÉRÉ (`SynthesisPageV2Response` dans `generated.ts`), mais l'interface HAND-WRITTEN
  `SynthesisPageResponse` (types.ts:1347) — celle réellement consommée par `queries.ts`/`SynthesisPage` — ne portait
  PAS le champ. Ajout du champ optionnel `frag_distribution?: FragDistribution` en P2 (nécessaire pour
  `data.frag_distribution`). Micro-fix in-scope (câblage requis par P2.1). À noter : les surfaces P3-P6 consommant
  leurs propres DTOs hand-written devront faire de même.
- **D-P2-2 (formule `direct_melee` — vérif exacte différée)** : `probe-h5` est exécutable ici (auth OK, 343 sert le
  live H5), mais ne compte pas les Death `IsMelee`. Preuve agrégée (service record JGtm arena : melee=2 ≥ assass=1 ;
  H5 = melee compteur-parapluie) → cohérente avec `assassination ⊆ melee`, formule conservative conservée. La vérif
  NUMÉRIQUE exacte (fetch `StatsMatchDetails` d'un match, count des Death events `IsMelee` vs `TotalAssassinations`
  par joueur) exigerait d'ÉTENDRE probe-h5 (parse Death events) = HORS P2 + risque sonde throwaway. À traiter AVANT
  prod (P7 ou lot dédié H5). Aucune action de code P2.
- **D-P3-1 (signature `buildFragDistribution` généralisée) — TRAITÉ (in-scope P3.2)** : la signature P0
  (`rows, stats domain.SynthesisDetailedStats, totalKills int, hasMechanics bool`) couplait le builder au DTO Synthesis.
  Le Match view n'a pas de `SynthesisDetailedStats` (ses compteurs natifs vivent sur `MatchScoreboardRow`). Conformément
  à la NOTE P3.2, l'entrée kill-type a été généralisée en `domain.FragKillTypeCounts{Melee,Grenade,Assassination,
  GroundPound,ShoulderBash,Total}` (struct neutre) → `buildFragDistribution(rows, counts, hasMechanics)`. Appelant
  Synthesis (`loadTopWeaponKills`) + `buildAPIFragClasses` + 5 tests adaptés. ZÉRO duplication de logique.
- **D-P3-2 (voie canonique H5 = pas de bulk weapon kills)** : `buildMatchViewFromCanonical` (H5 live-only) ne charge
  AUCUN bulk weapon kill (pas de substrat DuckDB). Sa FragDistribution du viewer est donc construite avec `rows=nil` :
  melee/grenade/spartan + total servis par la ligne scoreboard native, la ventilation gun retombant intégralement dans
  « Non attribué » (résidu, conforme D3 + D-P0-2). Sur la voie repo (Infinite persisté) les classes gun sont pleines.
  Le breakdown par arme est donc VIDE sur H5 live (ChartCard placeholder). Cohérent avec la dégradation gracieuse ;
  à valider à la revue visuelle H5. `weaponUnknownPrefix` (i18n match-view) reste déclaré mais SANS consommateur —
  dette PRÉ-EXISTANTE (déjà orpheline avant P3), laissée en place (hors périmètre P3).
- **D-P4-1 (`KillTypesDonutCard` orphelin après P4.4)** : le retrait du montage kill-type de l'onglet Progression
  fait de Timeseries le DERNIER consommateur de `KillTypesDonutCard` (grep : plus aucune réf hors sa propre def après
  P4.4). Le fichier est LAISSÉ en place (consigne P4.4 : ne pas casser les partagés). `KillTypesDonut` (donut SVG
  interne) reste bien utilisé par Explorer (`ExplorerTargetSampleStats`). Retrait éventuel de `KillTypesDonutCard` =
  P7 (nettoyage code mort), hors périmètre P4.
- **D-P4-2 (`TimeseriesPageResponse.DetailedStats` sans consommateur front)** : le donut Progression était le seul
  lecteur front de `data.detailed_stats` (via `KillTypesDonutCard`). Après P4.4, le champ backend `DetailedStats`
  (encore peuplé dans `GetPage`, `timeseries_service.go`) n'a plus de consommateur côté Timeseries. Laissé en place
  (retrait DTO + arrêt du peuplement = P7, hors périmètre P4 ; risque nul, champ `omitempty`). À trancher au P7.
- **D-P5-1 (endpoint sessions/detail = SessionPageService, pas sessions_service.go)** : le plan pointait
  `sessions_service.go:27` (`SessionsService.GetSessions`) comme cible P5.1. VÉRIFICATION SUR PIÈCES : ce service sert
  la LISTE des sessions (calcul de découpage) ; l'endpoint `POST /pages/sessions/detail` (le DÉTAIL, avec les charts)
  est servi par `SessionPageService.GetPage` (`session_page_service.go`) qui construit `SessionCompareEntry`. C'est là
  qu'a été câblé le chemin de données (canonicalRows déjà chargées + repo weapon_kills injecté). Micro-correction de
  cible tranchée par le code (le code fait foi).
- **D-P5-2 (centralisation `logFragDistribution` — 3e copie)** : P5 introduisait une 3e copie du pattern de log
  d'agrégation FragDistribution (Debug compteurs + Warn sur-comptage) après Synthesis (`synthesis_service.go`, méthode)
  et Timeseries (`timeseries_service.go`, inline). CLAUDE.md règle 6 (≤2 copies → centraliser à la 3e) → extrait en
  helper PARTAGÉ `logFragDistribution(ctx, surface, title, player, fd)` (`synthesis_service_builders.go`) ; les 2
  copies existantes migrées vers lui (Synthesis : clé log `gamertag`→`player` uniformisée ; Timeseries : suppression
  du calcul `sumClasses` inline). Aucun test n'assertait ces logs (grep vert) → migration sans risque, gate service
  vert. **Garde-fou grep** (interdiction du littéral) NON ajouté ici → laissé à P7.2 (étape garde-fous du plan), pour
  ne pas sur-étendre P5.
- **D-P5-3 (top_weapon_kills ajouté à l'entry session, in-scope P5.2)** : pour rendre le binôme complet
  (sunburst + breakdown par arme) comme la carte de référence, `TopWeaponKills` a été ajouté à `SessionCompareEntry`
  en plus de `FragDistribution` (P5.1 ne nommait que `frag_distribution`). Coût nul : les mêmes `WeaponKillRow`
  chargées pour les gun classes alimentent `buildTopWeaponKills`. Conforme à P5.2 (« + top weapons si dispo »).
- **D-P6-1 (builder inaccessible depuis package teammates → extraction leaf) — TRAITÉ (in-scope P6.1)** : le chart
  Escouade est servi par `TeammatesService` (package `internal/service/teammates`), or `buildFragDistribution` était
  UNEXPORTED dans le package PARENT `internal/service` (qui importe teammates → l'inverse serait un cycle). Impossible
  de « réutiliser » sans dupliquer OU extraire. Décision : builder + helpers déplacés dans un package LEAF
  `internal/service/fragdist` (exporté `Build`), importable par service ET teammates. 5 appelants du package service
  (Synthesis/Match/Session/Timeseries + `synthesis_frag_distribution_test.go`) migrés vers `fragdist.Build` (mécanique,
  gate service vert). Les tests du builder RESTENT en package service (ils exercent `fragdist.Build` ; `fragdist` sans
  test dédié) — co-localisation possible en P7, hors périmètre P6.
- **D-P6-2 (mécaniques spartanes par joueur dans l'Escouade) — RÉSOLU (2026-07-19)** : `frag_classes` est désormais
  construit avec `hasMechanics = titleHasNativeKillMechanics(slug)` (capability `native_kill_mechanics`, jamais slug==).
  Les compteurs assassination/ground_pound/shoulder_bash sont chargés par gamertag via `loadSquadMechanicsByGT`
  (`LoadKillMechanics`, mêmes matchs/xuids que `buildSquadWeaponKills`) puis fusionnés dans les `FragKillTypeCounts`
  de chaque joueur avant `fragdist.Build`. Conséquence H5 : classe « Capacités spartanes » (ground_pound + shoulder_bash)
  + split Mêlée (Assassinat / Corps-à-corps direct) par joueur, comme les autres surfaces ; les frags spartanes ne
  retombent plus dans « Non attribué ». Infinite (cap off) : `mechByGT` nil, `hasMechanics=false` → INCHANGÉ. Le chart
  Escouade (`squadFragBreakdownChart`, niveau classe dynamique) rend `spartan_ability` sans modif de forme. Le chart
  dédié `SquadKillMechanicsChart` reste inchangé. Reste : revue visuelle H5.
- **D-P6-3 (cible service = TeammatesService, pas squad_service_v2) — cadrage corrigé sur pièces** : le plan/brief
  pointait `squad_service_v2.go` (`loadWeapons`, package service). VÉRIFICATION SUR PIÈCES : le sous-chart
  « Répartition des frags » de l'Escouade consomme `frag_classes`/`performance_series` de `TeammatesPageResponse`,
  servi par `POST /pages/teammates` → `TeammatesService.GetPage`. `SquadPageV2Response` (squad_service_v2) N'EST PAS
  consommé par ce chart (le frontend fetch `/pages/teammates`, cf. `squad/queries.ts`). Câblage fait dans
  `TeammatesService` (précédent D-P5-1 : le code fait foi). Le champ `frag_classes` est donc porté par
  `TeammatesPageResponse`, pas `SquadPageV2Response`. `squad_service_v2` inchangé (migration front non effectuée).
- **D-P6-4 (comptage counts par joueur depuis PerformanceSeries)** : `counts.Total/Melee/Grenade` par gamertag sont
  agrégés depuis `PerformanceSeries[gt]` (Σ `kills`/`melee_kills`/`grenade_kills`, valeurs canoniques API). Un joueur
  présent dans `playersOrdered` mais absent de `PerformanceSeries` (aucun match commun servi) → counts nuls → si des
  rows d'arme existent quand même, invariant (a) `Σ==total` peut ne pas tenir localement (Total=0). Cas dégénéré non
  observé en pratique (weapon kills ⇒ kills>0). `fragdist.Build` clampe le résidu ≥ 0 (invariant c préservé). Aucune
  action P6.
- **D-P2-3 (doublons de cartes après P2.1/P2.2) — RÉSOLU (fusion tranchée avec l'utilisateur, 2026-07-19)** : les
  titres i18n de `SynthesisKillTypesDonut` (« Répartition des frags ») et `SynthesisWeaponKillsChart` (« Frags par
  arme ») étaient IDENTIQUES au sunburst/breakdown → doublons. Décision actée : LE sunburst unifié remplace LES DEUX
  anciens donuts, et `FragWeaponBreakdown` remplace l'ancien par-arme. `SynthesisKillTypesDonut.tsx` et
  `SynthesisWeaponKillsChart.tsx` SUPPRIMÉS (0 réf restante hors commentaires). Partagés `KillTypesDonutCard`/
  `KillTypesDonut` conservés (Timeseries/Match/Explorer). La section frags de Synthesis = `SynthesisFragCard` seule.
  Note de fusion générale ajoutée en tête de la section Phases (vaut pour P3-P6 : chaque surface retire les anciennes
  cartes subsumées, pas seulement celle nommée).
