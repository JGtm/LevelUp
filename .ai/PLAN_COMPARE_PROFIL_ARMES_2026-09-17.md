# PLAN — Profil d'armes du Face-à-face (2026-09-17)

> Date : 2026-09-17. Cadrage validé par l'utilisateur en conversation le 2026-09-17 (trois blocs,
> grain rôle pour la portée, médiane + P10/P90 en barre, min/max en infobulle, aucun vainqueur).
> Branche d'exécution : `wt/compare-armes`, worktree dédié `../LevelUp-wt-compare-armes`
> créé depuis `feat/v75` (le worktree principal est PARTAGÉ entre agents, ne jamais y coder).
> Clôture : commits sur `wt/compare-armes`, fusion dans `feat/v75` sur signal de l'utilisateur.
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report, tout item statué,
> zéro fix hors périmètre). Ce fichier est la source de vérité de l'avancement.

## Objectif et critère de succès

Ajouter à la page Face-à-face (`/community/compare`, `features/compare/ComparePage.tsx`) une
section « Profil d'armes » à trois blocs, pour deux ou trois joueurs :

1. **Part des frags par classe** (épaule, poing, lourde, mêlée, grenade, et les seaux hors
   arsenal quand ils existent), en pourcentage, côte à côte.
2. **Portée par rôle** (précision, automatique, sniper, fusil à pompe, puissance, spéciale,
   poing, grenade...) : frags ET morts, bâton P10 à P90, losange sur la médiane, A et B
   superposés ; min et max observés, effectif et dénivelé dans l'infobulle.
3. **Top 3 armes** par joueur, icône + nom + frags.

**Critère de succès** (amendé au lot 3-bis, 2026-09-17 — voir amendement D2) : la section
s'affiche pour A vs B dès que les deux sont présents dans la base partagée, chacun sur le
scope lifetime de ses propres matchs, avec la note « sur N matchs » tirée de `matches` POUR
LES DEUX ; elle reste absente proprement pour un B absent de la base (ou sans xuid) ; elle se
dégrade bloc par bloc sur un titre sans positions par kill (Halo 5 : classes et top 3
présents, portée absente, aucune erreur). Aucun vainqueur n'est élu. Gates Go et web verts, CI de branche
verte au niveau job, gate visuel de l'utilisateur passé.

## Ce qui existe et se réutilise (vérifié sur pièces le 2026-09-17)

| Besoin | Existant | Fichier |
|---|---|---|
| Frags mesurés (arme, côté, distance, dénivelé) | `port.WeaponRangeRepository.LoadWeaponRange` (filtres `MatchIDs` + `XUIDs`/`Gamertag`) | `internal/port/weapon_range.go`, `platform/duckdb/weapon_range_repo.go` |
| Agrégat P10/médiane/P90 + seuil 8 + dénivelé | `analysis.WeaponRangeAggregate`, `WeaponRangeMinMeasured` | `internal/analysis/weapon_range.go` |
| Bloc de réponse portée (lignes, médianes globales, couverture, sous-seuil) | `domain.SynthesisWeaponRange`, `buildWeaponRangeBlock`, `mergeWeaponSides` | `domain/synthesis_weapon_range.go`, `service/weapon_range_section_build.go` |
| Dimensions d'une clé d'arme (class/role/family) | `resolveWeaponKeyDimensions` (non exposé par un port) | `platform/duckdb/weapon_resolver.go:284` |
| Kills par arme d'un joueur sur un scope, avec class/role | `port.WeaponKillsRepository.LoadWeaponKillsAggregated` (`ResolveRoles: true`) | `port/weapon_kills.go`, wire `weaponKillsRepoFor` |
| Répartition classe→rôle | `fragdist.Build(rows, counts, hasMechanics)` | `service/fragdist/fragdist.go:101` |
| Top armes triées (kills desc, départage label) | `buildTopWeaponKills` (sort inline) | `service/synthesis_service_builders.go:226` |
| Icône d'arme (URL + mode masque) | closure `WithWeaponImageURL` | `api/server.go:307` |
| Scope et stats d'un joueur (lifetime, campagne exclue) | `CompareRepo.GetLocalStats`, `GetCrossMatchSample` (SANS liste de match_id) | `platform/duckdb/compare_repo.go` |
| Graphe bâton + losange | `buildWeaponRangeOption`, `weaponRangeLines`, `weaponRangeChartHeight` | `features/synthesis/_weaponRangeChart.ts` |
| Ligne de comparaison sans vainqueur | `CompareBar` (`winner={null}` rendu comme égalité), `CompareMirrorRow` | `features/compare/` |
| Libellés classe / rôle | manifeste `frags.toml` (`frags.class.*`, `frags.role.*`), `fragRoleDisplayLabel` | `components/charts/fragRoleLabel.ts` |
| Icône côté web | `WeaponIcon` (`imageUrl`, `tinted`) | `components/ui/WeaponIcon.tsx` |

## Décisions tranchées AVANT exécution (fermes, ne pas re-décider)

**D1 — Grain.** Portée agrégée par **rôle** (jamais par famille : une trentaine de lignes
presque toutes sous le seuil ; jamais par classe : « lourde » mélange sniper et épée). Part des
frags par **classe**. Top **3** armes.

**D2 — Scope par joueur, même doctrine que les métriques existantes.**
- Joueur local (A toujours ; B si `IsLocal`) : TOUS ses matchs de `shared.match_participants`,
  campagne exclue (`excludeCampaignByMatchID`), comme `GetLocalStats`.
- B non local avec xuid résolu : les matchs COMMUNS avec A (`IsSample = true`, note
  « sur N matchs » côté front, même règle d'affichage que `sample_size_b`).
- B sans xuid (jamais croisé, résolution live en échec) : pas de bloc (`weapons` absent).
- La période des filtres (`CompareRequest.Filters`) n'est PAS appliquée : le service ne
  l'applique à aucune métrique aujourd'hui (constat, voir Découvertes).

> **AMENDEMENT D2 — 2026-09-17, lot 3-bis (revue du superviseur).** Le scope CROISÉ est
> RETIRÉ, et avec lui le drapeau `IsSample`. Doctrine effective :
>
> - **B présent dans la base partagée** = scope LIFETIME de ses lignes `match_participants`
>   (les mêmes matchs que ses métriques, `IsLocal` de la page), campagne exclue. Identique au
>   scope de A.
> - **B absent de la base partagée** (ou sans xuid) = pas de profil (`weapons` absent).
> - **Le scope croisé a été retiré parce qu'il était inatteignable** : sa requête lisait la
>   MÊME table `match_participants` avec la MÊME clause d'exclusion de campagne (portée sur
>   `a.match_id`, égal à `b.match_id` par la jointure) et n'y ajoutait qu'un `EXISTS(a.xuid =
>   A)`. Son résultat était donc un SOUS-ENSEMBLE du scope lifetime de B : « croisé non vide »
>   impliquait « lifetime non vide », la branche de repli de `weaponScopeB` n'était jamais
>   prise, `IsSample` n'était jamais vrai, et un test vert entretenait l'illusion (règle « 0
>   code mort », anti-pattern n°1 du dépôt). Vérification sur pièces faite avant suppression :
>   aucun cas où le croisé est non vide alors que le lifetime est vide.
> - **`matches` remplace ce que `is_sample` prétendait dire.** Il est publié pour les DEUX
>   joueurs et le front l'affiche pour les deux (décision prise pour le lot 4) : un joueur peu
>   vu montre un petit N, ce qui dit au lecteur sur quoi le profil repose, sans qualifier.

**D3 — Aucun vainqueur.** Toutes les lignes sont descriptives (`winner` null). Une distance
plus longue n'est pas meilleure, une part de frags décrit un style.

**D4 — Grammaire graphique.** Celle validée pour la Synthèse (D6 du plan portée) : bâton P10 à
P90, losange sur la médiane, tri par médiane croissante conservé du back. Deux graphes par
comparaison : « Où ils fraguent » (côté tueur) et « Où ils meurent » (côté victime) ; dans
chaque graphe, une ligne par rôle, deux bâtons (A en haut, B en bas), couleurs `compare-a` /
`compare-b`. Pas de graphe de dénivelé : la ventilation dénivelé va dans l'infobulle.

**D5 — Min et max en infobulle seulement.** Ajoutés au contrat (`min_m`, `max_m` sur
`WeaponRangeSide`), calculés dans l'agrégat, JAMAIS tracés (deux accidents ne sont pas une
portée). La Synthèse ignore les deux champs (aucun changement d'affichage là-bas).

**D6 — Seuil.** `analysis.WeaponRangeMinMeasured` (8) par couple (rôle, côté) et par joueur.
Les rôles sous le seuil sont NOMMÉS avec leur effectif, jamais cachés (D9 du plan portée).

**D7 — Mode miroir (3 joueurs).** Même section, disposition B | A | C : classes via
`CompareMirrorRow`, portée = deux paires de graphes côte à côte (A vs B, A vs C), top 3 sur
trois colonnes. A est identique dans les deux réponses ; le front lit A dans la réponse gauche.

**D8 — Clé de ligne = rôle, libellé côté web.** Le bloc portée réutilise
`domain.SynthesisWeaponRange` tel quel ; `WeaponRangeRow.weapon_key` porte la clé de rôle
(`precision`, `automatic`, ...), `label`/`label_en` restent vides. Le front résout
`frags.role.<clé>` puis `frags.class.<clé>` (les clés qui sont leur propre rôle : `grenade`,
`melee`, `sidearm`, `equipment`, `vehicle`, `turret`, `environmental`), jamais une clé brute.
Une clé d'arme sans rôle dans le registre est ÉCARTÉE et comptée au journal (Debug).

**D9 — Dégradation par bloc, jamais par slug.** Portée absente si le repo rend
`games.ErrCapabilityNotSupported` ou zéro frag mesuré ; classes absentes si aucun kill par
arme ; top 3 absent si aucune arme résolue. Chaque bloc est nil indépendamment. Aucun
`slug ==` (ratchet `no_slug_comparison_test.go`).

**D10 — Aucun libellé FR/EN en dur côté Go.** Rôles et classes = clés ; noms d'armes = chaîne
`weapon_names.toml` déjà portée par `WeaponKillRow.Label/LabelEN`. Toute string UI neuve en FR
ET EN dans `features/compare/i18n.ts` (parité testée par `i18n.test.ts`).

**D11 — Règle des copies.** Le tri « kills desc, départage label » existe dans
`buildTopWeaponKills` et `buildWeaponAccuracy` : la troisième copie est INTERDITE. Extraire
un helper `topWeaponKillRows(rows, n)` utilisé par `buildTopWeaponKills` ET par le compare,
avec un garde-rail grep (une seule occurrence du littéral de tri dans `internal/service/`).
Même règle pour la closure d'icône de `server.go:307` : exposée par UNE méthode du registry,
pas recopiée.

## Contrat de réponse (additif, `CompareResponse.weapons`, omis si absent)

```go
// domain/compare_weapons.go
type CompareWeaponProfile struct {
	PlayerA CompareWeaponSide `json:"player_a"`
	PlayerB CompareWeaponSide `json:"player_b"`
}

type CompareWeaponSide struct {
	Matches     int                   `json:"matches"`              // taille du scope (D2)
	IsSample    bool                  `json:"is_sample,omitempty"`  // true = échantillon croisé (B non local)
	TotalKills  int                   `json:"total_kills"`          // dénominateur des parts
	FragClasses []CompareFragClass    `json:"frag_classes"`         // [] jamais null
	Range       *SynthesisWeaponRange `json:"range,omitempty"`      // weapon_key = clé de RÔLE (D8) ; Opening toujours nil
	TopWeapons  []CompareTopWeapon    `json:"top_weapons"`          // [] jamais null, 3 max
}

type CompareFragClass struct {
	Class    string  `json:"class"`     // clé domain.FragClass* (frags.class.<clé> côté web)
	Kills    int     `json:"kills"`
	SharePct float64 `json:"share_pct"` // 0..100, convention *Pct du dépôt
}

type CompareTopWeapon struct {
	Label       string `json:"label"`              // FR-first (chaîne weapon_names.toml)
	LabelEN     string `json:"label_en,omitempty"`
	Kills       int    `json:"kills"`
	Class       string `json:"class,omitempty"`
	Role        string `json:"role,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	ImageTinted bool   `json:"image_tinted,omitempty"`
}

// domain : WeaponRangeSide gagne MinM / MaxM (D5)
MinM float64 `json:"min_m"`
MaxM float64 `json:"max_m"`

// domain : le scope d'un côté, lu par le repo
type CompareWeaponScope struct {
	MatchIDs                               []string
	Matches                                int
	Kills, Deaths, MeleeKills, GrenadeKills int
}
```

## Lot 0 — Poste et vérification sur pièces (rapide)

- [x] 0.1 `git worktree add ../LevelUp-wt-compare-armes -b wt/compare-armes feat/v75` ; toute
      la suite s'exécute dans ce worktree. `git branch --show-current` = `wt/compare-armes`.
- [x] 0.2 Rouvrir chaque fichier du tableau « Ce qui existe » et confirmer les symboles cités
      (le code a pu bouger depuis le 2026-09-17). Noter tout écart au journal.
- [x] 0.3 Lire `.ai/PLAN_DUELS_PORTEE_2026-09-06.md` §« Décisions » (D6, D9, D10).

**Gate 0** : worktree créé, `go build ./...` et `npm run typecheck` verts à la base.

## Lot 1 — Analyse pure : regroupement et min/max (rapide)

Fichiers : `internal/analysis/weapon_range.go` (+ `_test.go`), `domain/synthesis_weapon_range.go`,
`service/weapon_range_section_build.go`.

- [x] 1.1 `analysis.WeaponRange` gagne `Min, Max float64`, calculés dans `weaponRangeOf` sur la
      même série triée que les percentiles. Test : nominal, une seule mesure (min = max =
      médiane), série non triée.
- [x] 1.2 `analysis.RegroupMeasuredKills(kills []MeasuredKill, keyOf func(weaponKey string) string) (out []MeasuredKill, dropped int)` :
      pur, remplace `WeaponKey` par `keyOf(WeaponKey)` ; clé vide = écarté et compté. Ne
      touche à rien d'autre (côté, distance, dénivelé conservés). Tests : regroupement de deux
      armes vers un rôle, clé inconnue écartée, entrée vide.
- [x] 1.3 `domain.WeaponRangeSide` gagne `MinM`/`MaxM` ; `weaponRangeSideOf` les remplit.
      Test existant de `weapon_range_section_build` étendu sur les deux champs.
- [x] 1.4 `weapon_range_guard_test.go` (seuils définis une seule fois) reste vert sans
      modification : ne PAS introduire de nouveau littéral de seuil.

**Gate 1** :
```
cd apps/go-api && go test ./internal/analysis/... ./internal/domain/... ./internal/service/... && go vet ./internal/analysis/... ./internal/domain/...
```

## Lot 2 — Ports et DuckDB : scopes et dimensions (moyen)

Fichiers : `port/repository_data.go` (CompareRepository), `port/weapon_range.go`,
`platform/duckdb/compare_repo.go`, `platform/duckdb/weapon_range_repo.go`,
`domain/compare_weapons.go`.

- [x] 2.1 `port.CompareRepository` gagne :
      `GetWeaponScope(ctx, xuid, titleSlug string) (*domain.CompareWeaponScope, error)` (tous
      les matchs du xuid, campagne exclue, mêmes clauses que `GetLocalStats`) et
      `GetCrossWeaponScope(ctx, xuidA, xuidB, titleSlug string) (*domain.CompareWeaponScope, error)`
      (matchs communs, totaux de B, même jointure que `GetCrossMatchSample`). Les deux via
      `SharedReadDB().Get` (SharedReader, jamais `OpenReadOnly`), timeout 15 s, `(nil, nil)` si
      zéro match. Totaux : `SUM(kills)`, `SUM(deaths)`, `SUM(melee_kills)`, `SUM(grenade_kills)`
      depuis `match_participants` (vérifier les noms de colonnes sur pièces, skill `db-schema`).
      **Lot 3-bis (2026-09-17) : `GetCrossWeaponScope` RETIRÉ** — sous-ensemble du premier,
      branche appelante morte (voir amendement D2). Seul `GetWeaponScope` subsiste.
- [x] 2.2 Les mocks existants de `compare_service_test.go` (`mockCompareRepo`,
      `mockCompareRepoAB`) implémentent les deux méthodes (retour `nil, nil` par défaut).
      — `mockCompareRepoAB` : fait. `mockCompareRepo` : `[~]` couvert par l'analyse ci-contre —
      il n'implémente PAS `port.CompareRepository` aujourd'hui (ni `GetPlayerATH`, ni
      `GetPlayerATHFor`, ni `GetEncounterStats`, ni `GetCrossMatchSample`) et n'est jamais
      passé à `NewCompareService` ; lui ajouter les deux méthodes serait du code mort.
      Vérifié : `go vet -tags=integration ./...` ne le signale pas.
- [x] 2.3 Test DuckDB `:memory:` des deux méthodes (nouveau `compare_repo_weapons_test.go`,
      même régime de tag que les autres tests du paquet) : trois matchs dont un de campagne
      exclu, un commun A/B, totaux exacts.
      **Lot 3-bis** : les deux tests du scope croisé (`TestGetCrossWeaponScope_*`) sont
      SUPPRIMÉS avec la méthode. Restent `TestGetWeaponScope_LifetimeCampagneExclue` (trois
      matchs dont un de Campagne, totaux exacts) et `TestGetWeaponScope_AucunMatch`.
- [x] 2.4 `port.WeaponRangeRepository` gagne
      `ResolveWeaponDimensions(ctx, titleSlug string, keys []string) (map[string]WeaponDimensions, error)`
      avec `type WeaponDimensions struct{ Class, Role, Family string }` ; implémentation =
      délégation à `resolveWeaponKeyDimensions` (aucune requête neuve). Metadata absente =
      map vide, pas d'erreur. Le fake de `weapon_range_section_test.go` (ou équivalent)
      implémente la méthode.
- [x] 2.5 Ratchets du dépôt inchangés : `no_art_patterns_test.go`, `no_slug_comparison_test.go`,
      `shared_read_recovery_routing_test.go` verts sans modification d'allowlist.

**Gate 2** :
```
cd apps/go-api && go test ./internal/port/... ./internal/platform/duckdb/... ./internal/service/... && go vet ./...
```

## Lot 3 — Service et câblage (moyen)

Fichiers : `service/compare_weapons.go` (nouveau, ≤ 500 L ; scinder en
`compare_weapons_build.go` si besoin), `service/compare_service.go`,
`service/synthesis_service_builders.go`, `api/wire/registry_pages_home.go`, `api/server.go`,
`domain/compare.go`.

- [x] 3.1 `domain.CompareResponse` gagne `Weapons *CompareWeaponProfile` (tag json
      `weapons,omitempty`).
- [x] 3.2 `CompareService.WithWeaponProfile(kills port.WeaponKillsRepository, rng port.WeaponRangeRepository, image func(weaponID int64) (string, bool)) *CompareService`
      (nil-safe : sans les deux repos, `Weapons` reste nil).
- [x] 3.3 `buildWeaponProfile(ctx, statsA, statsB, xuidB)` appelé dans `GetPage` après
      `buildMetrics`, best-effort (jamais d'erreur remontée, log Warn sur anomalie SQL, Debug
      sur capability absente, parité `logWeaponRangeFailure`). Scope par côté selon D2 :
      A → `GetWeaponScope(xuidA)` ; B `IsLocal` → `GetWeaponScope(xuidB)` ; B non local avec
      xuid → `GetCrossWeaponScope(xuidA, xuidB)` et `IsSample = true` ; B sans xuid → `nil`.
      **Lot 3-bis (2026-09-17)** : `buildWeaponProfile(ctx, xuidB)` appelle `GetWeaponScope`
      pour A ET pour B, sans repli ; xuid B vide → `nil` sans toucher la base.
- [x] 3.4 Par côté, `buildWeaponSide(ctx, scope, xuid)` :
      (a) `LoadWeaponKillsAggregated(slug, {MatchIDs, XUIDs:[xuid], ResolveRoles:true})` ;
      (b) classes = `fragdist.Build(rows, FragKillTypeCounts{Melee, Grenade, Total: scope.Kills}, titleHasNativeKillMechanics(slug))`
      → `Classes` projetées en `CompareFragClass` avec `SharePct = kills*100/TotalKills`
      (classes à 0 kill omises, `unattributed` CONSERVÉ : les parts somment à 100) ;
      **Lot 3-ter (2026-09-17, P0 de la revue adversariale)** : sur un titre à capability
      `native_kill_mechanics`, les trois compteurs `Assassination/GroundPound/ShoulderBash`
      sont CHARGÉS (`loadKillMechanicsForXUID`) — les laisser à zéro sous-évaluait « Mêlée »,
      faisait disparaître « Capacités spartanes » et gonflait « Non attribué » d'autant ;
      (c) portée = `LoadWeaponRange(slug, {MatchIDs, XUIDs:[xuid]})` → clés distinctes →
      `ResolveWeaponDimensions` → `analysis.RegroupMeasuredKills(kills, role)` →
      `buildWeaponRangeBlock(regrouped, nil, weaponRangeScopeInfo{matchIDs, scope.Kills, scope.Deaths})` ;
      `hydrateWeaponRangeLabels` NON appelé (D8) ; `Opening` nil ; bloc nil si zéro mesure
      ou capability absente ;
      (d) top 3 = `topWeaponKillRows(rows, 3)` (helper D11) → `CompareTopWeapon`, icône via
      la fonction injectée (URL vide = pas d'icône).
- [x] 3.5 D11 : extraire `topWeaponKillRows(rows []port.WeaponKillRow, n int) []port.WeaponKillRow`
      dans `synthesis_service_builders.go`, `buildTopWeaponKills` l'appelle ; garde-rail
      `compare_weapons_guard_test.go` : le littéral de tri choisi apparaît UNE fois dans
      `internal/service/` hors tests.
- [x] 3.6 Câblage : dans `registry_pages_home.go` `Compare()`, `.WithWeaponProfile(r.weaponKillsRepoFor(pdb), duckdb.NewWeaponRangeRepo(pdb, r.killSourceClassifierFor(pdb)), r.weaponImageURLFor(pdb))`.
      `weaponImageURLFor` = méthode NEUVE du registry qui porte la closure aujourd'hui inline
      dans `server.go:307` ; `server.go` l'appelle à son tour (une seule définition, D11). Le
      câblage est INCONDITIONNEL (le repo décide de la capability, jamais un `if` ici, même
      motif que Timeseries `registry_pages.go:398`).
- [x] 3.7 Tests service (fakes, sans DuckDB) dans `compare_weapons_test.go` :
      A local + B local (deux scopes lifetime) ; B non local croisé (`IsSample`, matches du
      scope croisé) ; B sans xuid (`Weapons` nil) ; range repo rend
      `ErrCapabilityNotSupported` (classes et top 3 présents, `Range` nil) ; clé d'arme sans
      rôle écartée ; parts qui somment à 100 ± 0,01 ; top 3 borné et trié.
      **Lot 3-bis** : le test « B non local croisé » est SUPPRIMÉ (il couvrait un chemin
      mort). Il est remplacé par `TestBuildWeaponProfile_BAbsentDeLaBase_ProfilAbsent`, et
      `TestBuildWeaponProfile_DeuxJoueursLocaux` vérifie désormais que la base est lue
      exactement DEUX fois — une par joueur, un seul chemin.
- [x] 3.8 Ratchet identité `registry_auth_page_identity_ratchet_test.go` vert (le xuid A reste
      celui du constructeur).
- [x] 3.9 `make generate-types` : `apps/web/src/lib/api/generated.ts` porte
      `CompareWeaponProfile`, `CompareWeaponSide`, `CompareFragClass`, `CompareTopWeapon` et
      `WeaponRangeSide.min_m/max_m`.

**Gate 3** :
```
cd apps/go-api && go test ./... && go vet ./... && golangci-lint run ./internal/service/... ./internal/api/... ./internal/platform/duckdb/... ./internal/analysis/... ./internal/domain/... ./internal/port/...
make generate-types && git diff --stat apps/web/src/lib/api/generated.ts
```
Aucun `-tags=integration` requis (aucune écriture, pas de persist/sync/migration touchés) ;
le dire explicitement au journal.

## Lot 4 — Web : section « Profil d'armes » (moyen à lourd)

Fichiers : `features/compare/ComparePage.tsx`, `features/compare/i18n.ts` (+ test),
`features/compare/CompareWeaponsSection.tsx` (nouveau), `features/compare/compareWeapons_logic.ts`
(nouveau, pur, + test), `features/synthesis/_weaponRangeChart.ts` (+ test),
`features/synthesis/SynthesisWeaponRangeSection.tsx`, `features/synthesis/SynthesisWeaponRangeTable.tsx`,
`lib/api/types.ts`.

- [x] 4.1 `types.ts` : `CompareResponse.weapons?: components['schemas']['CompareWeaponProfile']`
      (interface manuscrite existante, champ typé depuis le généré ; exporter les alias
      `CompareWeaponSide`, `CompareFragClass`, `CompareTopWeapon` depuis `components['schemas']`).
- [x] 4.2 Généralisation de `_weaponRangeChart.ts` sans changer le rendu Synthèse :
      `WeaponRangeLine.kills/deaths` renommés `top/bottom` ; `WeaponRangeOptionInput`
      reçoit `topColor/bottomColor` et `labels.top/labels.bottom` (à la place de
      kills/deaths) ; l'infobulle ajoute une ligne « min – max observés » quand
      `min_m`/`max_m` sont présents (libellé injecté `labels.observed`). `weaponRangeLines`
      (Synthèse) mappe kills→top, deaths→bottom. `buildWeaponElevationOption`,
      `SynthesisWeaponRangeSection`, `SynthesisWeaponRangeTable` et leurs tests suivent le
      renommage. Les tests Synthèse existants restent verts (même option produite à
      libellés égaux).
- [x] 4.3 `compareWeapons_logic.ts` (pur, testé) :
      **Lot 4** : `sampleNote` N'EXISTE PAS — `is_sample` a été retiré au lot 3-bis. La note
      « sur N matchs » se lit de `matches`, pour CHAQUE joueur et sans condition, dans le
      composant. Le module rend en plus `coverageOf` et `hasWeaponProfile`.
      `fragClassRows(sideA, sideB)` (union des classes, part 0 pour la classe absente d'un
      côté, ordre = ordre de A puis B) ;
      `roleRangeLines(sideA, sideB, side: 'kills'|'deaths', locale)` → `WeaponRangeLine[]`
      (union des rôles, `top` = A, `bottom` = B, tri par médiane de A croissante puis B) ;
      `roleLabel(key, locale)` : `frags.role.<key>` → `frags.class.<key>` → clé (garde-rail
      : jamais une clé `frags.` brute affichée, même contrat que `fragRoleDisplayLabel`) ;
      `belowThresholdText(rows, locale)` ; `sampleNote(side, text)` (note « sur N matchs »
      seulement si `is_sample`).
- [x] 4.4 `CompareWeaponsSection.tsx` : props `{ left: CompareResponse; right?: CompareResponse; text; locale }`.
      **Lot 4** : TROIS fichiers et non un — `CompareWeaponsSection.tsx` (classes, armes,
      orchestration), `CompareWeaponsRange.tsx` (le bloc portée) et `compareWeaponsShared.ts`
      (encres, formateurs, résolveur de libellé). Seuils du dépôt respectés (fichier ≤ 500 L,
      fonction ≤ 80 L) — voir le journal.
      Bloc 1 : une `CompareBar` par classe (`winner={null}`, valeurs « 42 % », `sampleNote`),
      `CompareMirrorRow` en miroir. Bloc 2 : deux `ChartCard` (frags / morts) par comparaison,
      option via `buildWeaponRangeOption` avec `resolveToken('compare-a'/'compare-b')` (et
      `compare-c` pour la paire de droite), hauteur `weaponRangeChartHeight` ; sous chaque
      graphe : couverture « N frags mesurés sur M » par joueur et liste nommée des rôles sous
      le seuil (`WEAPON_RANGE_MIN_MEASURED`) ; en miroir, les deux paires côte à côte.
      Bloc 3 : colonnes top 3 (`WeaponIcon` + nom via locale + frags), B | A | C en miroir.
      Section entière absente si `weapons` absent des deux réponses ; chaque bloc absent
      indépendamment si son côté est vide.
- [x] 4.5 `ComparePage.tsx` : la section s'insère après « Bilan & Rang » dans les deux modes.
      Titre `text.catWeapons`. Aucune couleur hex ni classe Tailwind couleur (skill
      `color-tokens`) ; tokens `compare-a/b/c` uniquement.
- [x] 4.6 `i18n.ts` : nouvelles clés FR + EN (`catWeapons`, `weaponsClassesTitle`,
      `weaponsRangeKills`, `weaponsRangeDeaths`, `weaponsTopTitle`, `weaponsCoverage(n, m)`,
      `weaponsBelowThreshold(list)`, `weaponsObserved`, `weaponsNoRange`, `weaponsPercentiles`,
      `weaponsNoMeasure`) ; `i18n.test.ts` (parité des clés) vert. FR sans anglicisme
      (« frags », « morts », « portée », jamais « kills »/« range »).
      **Lot 4** : une DOUZIÈME clé, `weaponsMatches(n)` — « sur N matchs », affichée pour
      chaque joueur depuis le retrait d'`is_sample` (lot 3-bis). `i18n.test.ts` gagne trois
      témoins : parité du dictionnaire ENTIER (le test historique ne couvrait que `metrics`),
      présence des douze clés dans les deux langues, et absence d'anglicisme dans les
      libellés FR du bloc.
- [x] 4.7 Tests : `compareWeapons_logic.test.ts` (union, parts, tri, libellé jamais brut,
      note « sur N matchs » — et non « note d'échantillon », `is_sample` ayant été retiré au
      lot 3-bis) ; `CompareWeaponsSection.test.tsx` (rendu 2 joueurs, rendu miroir,
      section absente sans `weapons`, bloc portée absent sans `range`) avec
      `echarts-for-react` mocké (référence : tests de `SynthesisWeaponRangeSection`).
- [x] 4.8 Garde-rails existants verts : lint anti-anglicismes, `lint-no-hardcoded-fields`,
      `metric-key-guardrail.test.ts`.

**Gate 4** :
```
cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp ; npm run typecheck && npm run lint && npm run test
```
(vitest hors sandbox ; typecheck après purge du cache incrémental.)

## Lot 5 — Clôture (rapide)

- [x] 5.1 Gate complet : `cd apps/go-api && go test ./... && go vet ./...` ; gate 4 rejoué ;
      `make go-api-lint` sur les paquets touchés.
- [ ] 5.2 Entrée `.ai/thought_log.md` (date, titre, Complété, décision, résultats, suite).
- [!] 5.3 Commits par lot sur `wt/compare-armes` (préfixe `compare-armes(lotN):`), push de la
      branche, `gh run list --branch wt/compare-armes` vert au niveau job.
      **Lot 5 (2026-09-17)** : commits FAITS (sept), push FAIT (EXIT_PUSH=0). Le verdict CI
      est IMPOSSIBLE sur cette branche — `wt/**` n'est dans les declencheurs ni de `ci.yml`
      ni de `gitleaks.yml`, et `ci.yml` n'a pas de `workflow_dispatch`. Ni echec ni rouge :
      aucun run n'existe. Elargir les declencheurs est HORS PERIMETRE (voir journal).
- [ ] 5.4 Gate visuel de l'utilisateur (serveur local, `/community/compare` avec un B local,
      un B croisé non local, un B jamais croisé, et le titre Halo 5) : l'utilisateur nomme
      les témoins. Aucune fusion avant son verdict.
- [ ] 5.5 Fusion dans `feat/v75` sur signal explicite de l'utilisateur ; ce plan mis à jour
      (toutes cases statuées) et inclus dans le commit de fusion.

## Découvertes (à consigner, NE PAS traiter)

- `CompareRequest.Filters` est validé (`Validate`) mais jamais appliqué par `CompareService`
  (aucune lecture de `req.Filters` dans `compare_service.go`) : champ mort du contrat, à
  trancher hors plan (retrait ou application réelle aux métriques).
- `WEAPON_KEYS_WITHOUT_RANGE` (`_weaponRangeChart.ts:59`) filtre `hinf_environment` côté
  front, pis-aller documenté ; la place durable est le classifieur Go. Avec le grain rôle,
  `environmental` devient une clé de rôle légitime (D8) : la ligne `environmental` est
  PUBLIÉE au compare (elle a un sens de distance : chute, explosif de carte).
- `CompareResponse` est une interface manuscrite dans `types.ts` alors que `CompareMetricRow`
  vient du généré : migrer l'ensemble vers `components['schemas']` est un lot à part.

**Découvertes des lots 0 à 3 (2026-09-17) — consignées, NON traitées**

- `CompareRepo.GetCrossMatchSample` n'applique PAS `excludeCampaignByMatchID`, alors que
  `GetLocalStats` le fait sur la même table. Les quatre métriques locale-only d'un joueur B
  croisé (série max, durée de vie, tués parfaits, tirs à la tête) comptent donc les matchs de
  Campagne de `halo_5`, ce que la colonne de A n'a jamais fait. `GetCrossWeaponScope` a choisi
  la clause d'exclusion ; la métrique voisine reste comme elle est — à trancher hors plan.
- Une TROISIÈME variante du tri « armes par frags décroissants » vit dans
  `timeseries_service_aggregations.go:360` : même comparateur, mais départage sur le
  `WeaponID` et non sur le libellé. Ce n'est pas une copie du littéral visé par D11 (le
  garde-rail ne la signale pas, et c'est voulu — son témoin licite épingle justement cette
  forme), mais c'est bien une DEUXIÈME doctrine de départage pour le même classement
  d'armes : une même arme à frags égaux se range différemment selon la page. À unifier hors
  plan.
- `make generate-types` ne régénère PAS `openapi.yaml` (généré par `cmd/openapi-gen`). Lancé
  seul après un changement de contrat Go, il produit un `generated.ts` périmé ET un gate vert.
  Seul `make openapi-check` attrape le décalage. Un `generate-types` qui dépendrait de
  `openapi-gen` fermerait le piège — modification de Makefile hors périmètre.
- `api/server.go` construit son `AssetMetadataHandler` avec un `hiAssetURL` passé en paramètre
  alors qu'un `*wire.ServiceRegistry` existe déjà au même point d'appel (`server_apiv1.go`,
  `reg` ligne 1255). Deux chemins vers le même adaptateur d'assets — hors périmètre.

**Découverte du lot 3-bis (2026-09-17) — consignée, À NE PAS TRAITER dans ce chantier**

- Le raisonnement qui a tué le scope croisé s'applique TEL QUEL à la branche
  `enrichRemotePlayerBWithCrossSample` / `IsLocalSample` de `compare_service.go`. Elle n'est
  appelée que lorsque `GetLocalStats(xuidB)` a échoué ou rendu vide (chemin « B remote ») ;
  or `GetCrossMatchSample` lit la même table `match_participants` avec une auto-jointure qui
  n'y ajoute qu'un `EXISTS(a.xuid = A)` — son résultat est donc un sous-ensemble des lignes
  que `GetLocalStats` aurait comptées. `GetLocalStats` vide ⇒ `GetCrossMatchSample` vide, donc
  `IsLocalSample` n'est jamais vrai en nominal, et les quatre métriques locale-only qu'il
  débloque (série max, durée de vie, tués parfaits, tirs à la tête) ne s'affichent jamais pour
  un B remote. NUANCE à vérifier avant tout retrait : `GetLocalStats` échoue aussi sur
  `sql.ErrNoRows` transformé en erreur — le chemin remote est donc atteint pour un joueur
  ABSENT, pas pour un joueur présent ; c'est bien le même argument, mais il touche une branche
  LIVRÉE et visible en prod. Chantier à part, avec sa propre vérification sur pièces.

**Découvertes du lot 3-ter (2026-09-17) — consignées, NON traitées**

- La distinction Debug / Warn de `logCompareWeaponFailure` (absence légitime de capability vs
  anomalie SQL) n'est couverte par AUCUN test. C'est pourtant ce qui empêche un titre sans
  décodeur de film de noyer les vrais bugs sous des Warn. Un témoin sur le niveau journalisé
  demanderait un handler `slog` de test — à faire dans un lot de durcissement de la
  journalisation, pas ici.
- `make openapi-check` (`cmd/openapi-gen -check`, qui détecte une dérive Go -> `openapi.yaml`)
  n'est joué NI par la CI, NI par `make gate-push`, NI par lefthook. Seul
  `generated-types-fresh.guard.test.ts` tourne, et il compare `generated.ts` au YAML — pas le
  YAML au code Go. Conséquence : un changement de contrat Go non suivi d'un `make openapi-gen`
  passe tous les gates, et le front reste typé sur l'ancien contrat avec un `tsc` vert. C'est
  exactement le piège rencontré au lot 3 (item 3.9). Le refermer = ajouter `openapi-check` à la
  CI ou au hook — modification d'outillage hors périmètre.
- Le dictionnaire `features/compare/i18n.ts` n'est PAS dans le périmètre du garde-rail
  anti-anglicismes (`lib/i18n/no-anglicisms.guard.test.ts`, qui ne scanne que cinq
  dictionnaires manuscrits et dix manifestes TOML). Les libellés FR de la page ne sont donc
  contrôlés que par relecture. Le lot 4 y répond LOCALEMENT (un témoin dans `i18n.test.ts`
  sur les douze libellés du profil d'armes) ; élargir le périmètre du garde-rail global au
  dictionnaire entier reste un lot à part.


**Découverte du lot 5 (2026-09-17) — consignée, NON traitée**

- **Les branches `wt/**` ne déclenchent AUCUNE CI.** `ci.yml` et `gitleaks.yml` partagent une
  liste de préfixes (`main, feat/**, feature/**, fix/**, hotfix/**, refactor/**, perf/**,
  docs/**, chore/**, integration/**`) où `wt/**` est absent, et `ci.yml` n'expose pas de
  `workflow_dispatch`. Or `wt/*` est la convention des worktrees dédiés du dépôt
  (`wt/ci-garde-rails-tstr`, `wt/vehicules-tourelles`, `wt/compare-armes`...) : tout chantier
  mené sous ce préfixe pousse sans CI et sans scan de secrets, et ne s'en aperçoit qu'en
  cherchant un run qui n'arrivera pas. C'est exactement le piège que le commentaire de
  `gitleaks.yml` documente pour `feat/**` (« absent de `shared-social-gate.yml` pendant 45
  runs »), reproduit un préfixe plus loin. Le refermer touche au moins deux workflows, doit se
  faire simultanément des deux côtés, et croise l'invariant D29 (`paths-ignore` de `ci.yml`
  sous-ensemble de celui de `deploy.yml`, gardé par
  `archlint.TestCIPathsIgnoreSubsetOfDeployPathsIgnore`). Décision utilisateur.

**Découvertes du lot 4 (2026-09-17) — consignées, NON traitées**

- `features/synthesis/_weaponRangeChart.ts` est désormais partagé par DEUX features. Sa place
  durable est `components/charts/`, où vivent les wrappers de graphe partagés (doctrine du
  linter cross-feature et du skill `foundations-usage`). Le déplacement emmène
  `resolveWeaponLabel` (aujourd'hui dans `synthesis/weaponRange_logic`, qui ne peut pas
  descendre seul) et retouche les quatre fichiers Synthèse qui en dépendent : chantier à part.
  En attendant, la dépendance est DÉCLARÉE au module près dans `ALLOWED_CROSS_IMPORTS`.
- `CompareResponse` reste une interface MANUSCRITE dans `types.ts` alors que le champ neuf
  `weapons` est typé depuis le généré : les deux moitiés du même type n'ont pas la même
  source. Migrer l'interface entière vers `components['schemas']` est le lot déjà consigné
  plus haut ; ce lot-ci n'a fait que ne pas aggraver la dette.
- Le test `components/shell/NavL1.test.tsx` a expiré (5 000 ms) sur une exécution complète de
  la suite (487 s) et passe seul en 9,4 s : c'est un flake de charge, pas un échec de
  contenu. Le durcir (timeout explicite ou allègement du rendu) est hors périmètre.

## Journal d'exécution

_(une ligne par clôture de lot : date, lot, gate, découvertes, écarts au plan)_

### 2026-09-17 — Lot 0 clos

**Gate 0** — worktree `../LevelUp-wt-compare-armes` sur `wt/compare-armes` (base `feat/v75`
`05723dce2`).

```
cd apps/go-api && go build ./...        -> EXIT_BUILD=0 (aucune sortie)
cd apps/web && npm install              -> EXIT=0 (node_modules absent du worktree neuf)
cd apps/web && npm run typecheck        -> tsc -b, aucune erreur, EXIT_TYPECHECK=0
```

**Écarts relevés à la vérification sur pièces (0.2)** — tous les symboles du tableau
existent ; quatre différences de signature ou d'emplacement par rapport au texte du plan :

1. `buildWeaponRangeBlock(kills, openings []analysis.MeasuredKill, scope weaponRangeScopeInfo)`
   vit dans `service/weapon_range_section_build.go` ; `weaponRangeScopeInfo` est défini dans
   `service/weapon_range_section.go` et ses champs sont NON exportés
   (`matchIDs`, `totalKills`, `totalDeaths`). Le littéral positionnel du plan (3.4c) reste
   valide car l'appelant est dans le même paquet.
2. La closure d'icône de `api/server.go:307` a la signature
   `func(titleID string, weaponID int64) (string, bool)` (le plan écrit
   `func(weaponID int64) (string, bool)`) et capture `hiAssetURL`, un paramètre de
   `buildAssetMetadataHandler`, pas un champ du `ServiceRegistry` (`hiAssetURL` vit dans
   `titleRuntime`, `server.go:411`). Le registry accède à l'adapter par `assetURLFor(slug)`
   (`registry.go:506`). Conséquence pour 3.6 : la définition canonique sera une fonction
   EXPORTÉE du paquet `wire` (server.go est `package api` et ne peut pas appeler une
   méthode non exportée), la méthode du registry et `server.go` l'appelant tous deux.
3. `resolveWeaponKeyDimensions` rend `weaponKeyResolved{class, role, family, label, labelEN,
   numericID}` (le plan ne cite que class/role/family) — le port n'exposera que les trois
   dimensions demandées (2.4).
4. `GetCrossMatchSample` n'applique PAS `excludeCampaignByMatchID` (contrairement à
   `GetLocalStats`). `GetCrossWeaponScope` l'appliquera quand même, pour que le scope croisé
   soit homogène avec celui de A (D2 : campagne exclue) — écart assumé, noté ici.

**0.3** — D6 (bâton p10→p90, losange sur la médiane, tri par médiane croissante), D9 (seuil 8
défini une seule fois dans `internal/analysis/`, armes sous le seuil publiées et non cachées)
et D10 (aucun filtre temporel dans le repo : le scope arrive par `MatchIDs`) relus dans
`.ai/PLAN_DUELS_PORTEE_2026-09-06.md`.

**Note de tenue du plan** : le fichier de plan était UNTRACKED dans le worktree principal
(partagé). Il a été COPIÉ dans ce worktree — le worktree principal n'a pas été modifié — et
c'est cette copie qui est versionnée sur `wt/compare-armes`.

### 2026-09-17 — Lot 1 clos (1.1 à 1.4 tous `[x]`)

**Gate 1** — commande exacte du plan :

```
cd apps/go-api && go test ./internal/analysis/... ./internal/domain/... ./internal/service/... \
  && go vet ./internal/analysis/... ./internal/domain/...
```

```
ok      levelup/go-api/internal/domain          2.201s
ok      levelup/go-api/internal/domain/title    52.415s
ok      levelup/go-api/internal/service         20.406s
ok      levelup/go-api/internal/service/fragdist        0.930s
...
EXIT_TEST=0
EXIT_VET=0
```

Détail des tests neufs (exécution ciblée `-v`) : `TestWeaponRangeMinMaxNominal`,
`TestWeaponRangeMinMaxUneSeuleMesure`, `TestWeaponRangeMinMaxSerieNonTriee`,
`TestRegroupMeasuredKillsFusionneVersLeRole`, `TestRegroupMeasuredKillsClefInconnueEcartee`,
`TestRegroupMeasuredKillsConserveCoteDistanceDenivele`,
`TestRegroupMeasuredKillsNeMutePasLEntree`,
`TestRegroupMeasuredKillsEntreeVideEtResolveurNil`,
`TestWeaponRangeSideOf_MinEtMaxPortesJusquAuContrat`,
`TestWeaponRangeSideOf_MinEtMaxCoteMorts` — tous PASS. Le garde-rail
`TestSeuilsPorteeDefinisUneSeuleFois` (1.4) reste PASS **sans modification** : aucun nouveau
littéral de seuil n'a été introduit.

**Écarts / précisions d'implémentation**

- `RegroupMeasuredKills` accepte un `keyOf` NIL (tout écarté et compté) plutôt que de paniquer :
  un câblage manquant ne doit pas faire tomber une page. Couvert par un test.
- Le test min/max du service est bâti sur une fixture à distances VARIÉES et non sur `wrKills`
  (qui pose n frags à la même distance : min == max == p10 == p90 == médiane, donc un test
  bâti dessus resterait vert même avec min/max câblés sur les percentiles).
- `internal/analysis/weapon_range.go` passe de 323 à 380 lignes (seuil 500 respecté).

### 2026-09-17 — Lot 2 clos (2.1, 2.3, 2.4, 2.5 `[x]` ; 2.2 `[x]` pour AB, `[~]` pour `mockCompareRepo`)

**Gate 2** — commande exacte du plan :

```
cd apps/go-api && go test ./internal/port/... ./internal/platform/duckdb/... ./internal/service/... \
  && go vet ./...
```

```
ok      levelup/go-api/internal/port                            9.213s
ok      levelup/go-api/internal/platform/duckdb                 256.687s
ok      levelup/go-api/internal/platform/duckdb/indexcheck      3.276s
ok      levelup/go-api/internal/platform/duckdb/prestige        28.244s
ok      levelup/go-api/internal/platform/duckdb/sharedprovider  0.443s
ok      levelup/go-api/internal/service                         37.454s
ok      levelup/go-api/internal/service/fragdist                2.330s
...
EXIT_TEST=0
EXIT_VET=0
```

**Le gate du plan ne couvre PAS le test 2.3** : `compare_repo_weapons_test.go` porte
`//go:build integration` (régime du paquet), et la commande de gate n'a pas le tag. Il a donc
été exécuté séparément, dans cette session :

```
go test -tags=integration ./internal/platform/duckdb/ -run 'TestGetWeaponScope|TestGetCrossWeaponScope' -v
--- PASS: TestGetWeaponScope_LifetimeCampagneExclue (0.55s)
--- PASS: TestGetWeaponScope_AucunMatch (0.31s)
--- PASS: TestGetCrossWeaponScope_MatchsCommunsTotauxDeB (0.52s)
--- PASS: TestGetCrossWeaponScope_JamaisCroise (0.34s)
ok      levelup/go-api/internal/platform/duckdb 1.907s   EXIT=0
```

Et `go vet -tags=integration ./...` -> EXIT=0 (c'est lui qui a prouvé que les deux mocks du
paquet `service` satisfont bien les interfaces élargies).

**2.5 — ratchets** : `go test ./internal/archlint/... ./internal/sync/` -> `ok` sur les deux
paquets ; `git diff --stat` sur `internal/sync/no_art_patterns_test.go`,
`internal/archlint/no_slug_comparison_test.go` et
`internal/sync/shared_read_recovery_routing_test.go` rend une sortie VIDE — aucune allowlist
touchée.

**Écarts / décisions d'implémentation**

- Les deux scopes vivent dans un fichier NEUF, `platform/duckdb/compare_repo_weapons.go`
  (le plan citait `compare_repo.go`, déjà à 341 lignes : y ajouter ~130 lignes l'aurait mené
  à ~470, trop près du plafond de 500).
- UNE SEULE requête par scope, rendant une ligne par match : `MatchIDs` et les totaux sont
  agrégés en Go sur les MÊMES lignes. Deux requêtes liraient deux instantanés de la base
  partagée (un sync peut écrire entre les deux), et la couverture publiée afficherait un
  dénominateur qui ne décrit plus le corpus de son numérateur.
- `GetCrossWeaponScope` applique `excludeCampaignByMatchID` sur `a.match_id`, alors que
  `GetCrossMatchSample` — dont il reprend l'auto-jointure — ne l'applique pas (écart noté au
  lot 0). Motif : le côté A du même profil est mesuré campagne exclue, deux scopes aux règles
  différentes rendraient les deux colonnes non comparables.
- Les tests 2.3 passent par le titre `halo_5` et non le titre par défaut : `halo_5` est le seul
  à déclarer des variants de Campagne, donc le seul où la clause d'exclusion n'est pas un
  no-op. Menés sur le titre par défaut, ils seraient verts quelle que soit l'implémentation.
- `ResolveWeaponDimensions` prend le slug EN PARAMÈTRE (comme les deux lectures du même repo)
  et non `r.pdb.TitleSlug` : résoudre sous le titre du PlayerDB rendrait les dimensions d'un
  autre jeu dès que les deux diffèrent. Le mock du service mémorise le slug reçu.
- `port.WeaponDimensions` n'expose que Class/Role/Family : `weaponKeyResolved` porte aussi
  label/labelEN/numericID, hors du besoin du port (les libellés passent par
  `WeaponLabelResolver`).
- `domain/compare_weapons.go` ne porte AU LOT 2 que `CompareWeaponScope`. Le reste du contrat
  (`CompareWeaponProfile` et ses blocs) arrive au lot 3, avec son producteur — publier des
  types que personne n'assemble en ferait du code mort le temps d'un lot.

### 2026-09-17 — Lot 3 clos (3.1 à 3.9 tous `[x]`)

**Gate 3** — commandes exactes du plan, exécutées dans cette session :

```
cd apps/go-api && go test ./...        -> EXIT_TEST=0 (aucun paquet hors "ok"/"no test files")
cd apps/go-api && go vet ./...         -> EXIT_VET=0
make generate-types                    -> EXIT_GEN=0
git diff --stat apps/web/src/lib/api/generated.ts apps/go-api/api/openapi.yaml
  apps/go-api/api/openapi.yaml      | 90 +++++++++++++++++
  apps/web/src/lib/api/generated.ts | 36 ++++++++
```

Aucun `-tags=integration` requis pour ce lot (aucune écriture, aucun persist/sync/migration
touché) — dit explicitement comme le demande le plan. Les tests `integration` du lot 2 ont
tout de même été rejoués plus haut.

**Le maillon lint du gate 3 demande une lecture précise.** La commande littérale du plan

```
golangci-lint run ./internal/service/... ./internal/api/... ./internal/platform/duckdb/... \
  ./internal/analysis/... ./internal/domain/... ./internal/port/...
```

sort en **EXIT=1 avec 132 issues** — c'est la DETTE GELÉE du dépôt sur ces six arbres
(funlen 33, goconst 38, gocyclo 12, lll 10, prealloc 3, revive 14, staticcheck 3, unparam 6,
unused 13), pas un défaut de ce lot : `make go-api-lint` n'invoque jamais golangci nu, il
invoque le RATCHET `--new-from-merge-base=origin/main`. Le même ratchet joué contre la base de
la branche :

```
golangci-lint run --timeout 5m --new-from-rev=05723dce2 <mêmes paquets>
0 issues.   EXIT_LINT_NEW=0
```

Et aucune des 132 issues ne porte sur un fichier NEUF de ce chantier (vérifié par grep sur la
sortie : les seules lignes touchant des fichiers modifiés sont `buildMetrics is too long
(81 > 80)` et `buildWeaponAccuracy - n always receives ...`, deux issues PRÉEXISTANTES dont le
numéro de ligne a seulement glissé).

**3.9 — écart de procédure, important** : `make generate-types` ne fait que
`openapi-typescript api/openapi.yaml -> generated.ts`. Or `openapi.yaml` est lui-même GÉNÉRÉ
(`cmd/openapi-gen`) : lancé seul, `make generate-types` aurait régénéré `generated.ts` depuis
un `openapi.yaml` périmé et le gate serait passé au vert SANS les nouveaux schémas.
`make openapi-gen` a donc été joué d'abord. Vérifié dans `generated.ts` :
`CompareWeaponProfile`, `CompareWeaponSide`, `CompareFragClass`, `CompareTopWeapon`,
`CompareResponse.weapons?`, et `WeaponRangeSide.min_m` / `.max_m`.

**Effet de bord du contrat sur le web, RÉPARÉ dans ce lot** : `min_m`/`max_m` étant REQUIS
dans le schéma généré, quatre fixtures de test Synthèse qui construisaient un
`WeaponRangeSide` sans eux ne compilaient plus (`SynthesisWeaponRangeSection.test.tsx`,
`SynthesisWeaponRangeSection.options.test.tsx`, `_weaponRangeChart.test.ts`,
`_weaponElevationChart.test.ts`). Ce n'est PAS un fix opportuniste : c'est une casse causée par
ce lot, et la laisser rendrait la branche rouge au typecheck. Les deux champs y sont ajoutés
avec un commentaire disant qu'ils ne sont jamais tracés. Vérifié :

```
cd apps/web && npm run typecheck                     -> EXIT_TYPECHECK=0
cd apps/web && npm run test -- --run src/features/synthesis
  Test Files  10 passed (10) · Tests 102 passed | 14 skipped (116)   EXIT_VITEST=0
```

**3.8** — `go test ./internal/api/wire/ -run 'Identity|identity'` : 4 tests PASS
(`TestForcePageIdentityXUID_*`, `TestEnrichCallersForcePageIdentity`).

**Écarts / décisions d'implémentation**

- `WithWeaponProfile` prend un `weaponImageFunc` (type nommé local) plutôt qu'un littéral
  `func(int64) (string, bool)` : le plan écrivait le littéral, le type nommé porte la doctrine
  (URL et « masque à teinter » sortent TOUJOURS du même appel — les séparer donne une
  silhouette noire).
- `buildWeaponProfile(ctx, xuidB)` et non `(ctx, statsA, statsB, xuidB)` : les deux `stats` ne
  servaient à rien — le scope de chaque côté se lit par xuid, et `IsLocal` de `statsB` n'est
  PAS le bon test (un joueur peut être `IsLocal` sans scope d'armes lisible). Le service
  essaie donc le scope lifetime de B, et retombe sur le scope croisé — plus robuste, même
  résultat produit.
- D11, closure d'icône : la définition canonique est `wire.WeaponImageURLFromAdapter(adapter)`,
  fonction EXPORTÉE du paquet `wire`, et non une simple méthode du registry — `api/server.go`
  vit dans le paquet `api` et ne peut pas appeler une méthode non exportée. La méthode
  `(*ServiceRegistry).weaponImageURLFor(pdb)` l'utilise pour le compare ; `server.go` l'appelle
  aussi, en gardant sa garde de titre. UNE seule écriture du couple URL + masque.
- Le garde-rail D11 (`compare_weapons_guard_test.go`) vise la FORME COMPLÈTE du comparateur
  (frags décroissants PUIS libellé croissant) et non la seule comparaison de frags :
  `internal/service/` porte plusieurs tris par frags décroissants légitimes et différents
  (répartition des frags, séries temporelles qui départagent sur l'identifiant d'arme). Les
  signaler ferait désactiver le garde-rail. Il porte son contrôle positif ET son témoin
  licite.
- `compare_service.go` passe de 587 à 597 lignes (fichier DÉJÀ au-dessus du seuil de 500 avant
  ce chantier) : +6 lignes de champs et +4 à `GetPage`. Croissance minimale et inévitable pour
  le câblage ; scinder ce fichier est un chantier à part, hors périmètre.
- `service/compare_weapons.go` : 347 lignes, fonctions toutes sous 80 lignes.

### 2026-09-17 — Lot 3-bis : retrait du scope croisé (revue du superviseur, P1)

**Constat, vérifié sur pièces avant toute suppression.** Les deux requêtes lisaient la MÊME
table :

```sql
-- GetWeaponScope(B)
FROM match_participants mp WHERE mp.xuid = B  <exclusion sur mp.match_id>

-- GetCrossWeaponScope(A, B)
FROM match_participants a
JOIN match_participants b ON b.match_id = a.match_id AND b.xuid = B
WHERE a.xuid = A                              <exclusion sur a.match_id>
```

`a.match_id = b.match_id` par la jointure, donc l'exclusion de campagne est le MÊME prédicat
sur les MÊMES lignes. Le croisé ne fait qu'ajouter `EXISTS(a.xuid = A)` : ses lignes sont un
sous-ensemble strict de celles du lifetime de B. Conclusion : **croisé non vide ⇒ lifetime non
vide**, donc la branche de repli de `weaponScopeB` ne pouvait jamais être prise, `IsSample`
jamais valoir vrai, et `TestBuildWeaponProfile_BNonLocalEstUnEchantillon` était un test vert
sur un chemin mort. Aucun cas contraire trouvé — le constat du superviseur est confirmé et la
suppression faite.

**Supprimé** : `port.CompareRepository.GetCrossWeaponScope` (+ impl `noopCompareRepo`),
`CompareRepo.GetCrossWeaponScope` (+ le helper `scanWeaponScope`, devenu un intermédiaire à un
seul appelant, réintégré dans `GetWeaponScope`), `CompareService.weaponScopeB` (réduit à
l'appel direct), le paramètre `echantillon` de `buildWeaponSide`, le champ
`domain.CompareWeaponSide.IsSample`, les méthodes de mock correspondantes, et TROIS tests
(`TestGetCrossWeaponScope_MatchsCommunsTotauxDeB`, `TestGetCrossWeaponScope_JamaisCroise`,
`TestBuildWeaponProfile_BNonLocalEstUnEchantillon`).

**Ajouté** : `TestBuildWeaponProfile_BAbsentDeLaBase_ProfilAbsent` (le seul cas « pas de profil
pour B » qui subsiste) ; `TestBuildWeaponProfile_DeuxJoueursLocaux` compte désormais les appels
au scope (exactement 2, un par joueur) ; `TestBuildWeaponProfile_BSansXUID_ProfilAbsent`
vérifie que la base n'est PAS interrogée sans xuid.

`Matches` est conservé et devient le seul porteur du « sur N matchs », publié pour les deux
joueurs (décision du lot 4).

**Gates rejoués dans cette session**

```
cd apps/go-api && go build ./...                              EXIT_BUILD=0
cd apps/go-api && go vet -tags=integration ./...              EXIT_VET_INT=0
make openapi-gen && make generate-types                       EXIT_GEN=0
  git diff --stat : openapi.yaml -2 lignes, generated.ts -1 ligne
  grep is_sample sur les deux fichiers : AUCUNE occurrence
cd apps/go-api && go test ./...                               EXIT_TEST=0
cd apps/go-api && go vet ./...                                EXIT_VET=0
go test -tags=integration ./internal/platform/duckdb/ -run TestGetWeaponScope -v
  --- PASS: TestGetWeaponScope_LifetimeCampagneExclue (0.34s)
  --- PASS: TestGetWeaponScope_AucunMatch (0.21s)               EXIT=0
golangci-lint run --timeout 5m --new-from-rev=05723dce2 <6 arbres>
  0 issues.                                                   EXIT_LINT_NEW=0
cd apps/web && npm run typecheck                              EXIT_TYPECHECK=0
```

`CompareWeaponSide` dans `generated.ts` après régénération : `frag_classes`, `matches`,
`range?`, `top_weapons`, `total_kills` — plus de `is_sample`.

### 2026-09-17 — Lot 3-ter : mécaniques natives (P0) et commentaire du contrat

**P0 — les mécaniques natives n'étaient pas chargées.** Sur un titre à capability
`native_kill_mechanics`, `fragdist.Build` RETRANCHE les frags de mécanique des classes d'arme
(un assassinat est inscrit dans `weapon_kills` au compte de l'ARME TENUE, `MechanicKills`) en
supposant que `FragKillTypeCounts.Assassination/GroundPound/ShoulderBash` les rapportent. Le
compare les laissait à zéro : « Mêlée » sous-évaluée du nombre d'assassinats, « Capacités
spartanes » absente, « Non attribué » gonflé d'autant. **Le total bouclait quand même** —
erreur arithmétiquement cohérente, donc invisible sur la page comme dans une assertion de
somme des parts.

**Correctif, sans troisième copie (règle n°6).** Le chargement vivait chez l'Explorer
(`ExplorerService.loadTargetKillMechanics`) ; le compare en aurait été une seconde écriture,
et l'Escouade en porte déjà une variante. Extrait en fonction LIBRE
`loadKillMechanicsForXUID(ctx, repo, xuid, matchIDs)` dans un fichier neuf
`service/kill_mechanics_loader.go`, avec l'interface de type-assertion renommée
`explorerKillMechanicsLoader` -> `killMechanicsLoader`. L'Explorer l'appelle désormais
directement (sa méthode thin est SUPPRIMÉE, pas laissée en façade) et le compare aussi,
gardé par `titleHasNativeKillMechanics(slug)` — capability, jamais un slug.

`teammates.loadSquadMechanicsByGT` N'EST PAS touchée, et c'est motivé : elle vit dans
`service/teammates` (qui ne peut pas importer son parent `service`), passe par un AUTRE port
(`squadLoader.LoadKillMechanics(ctx, slug, filters)`, trois arguments), charge N joueurs en une
requête et rend une map keyée par GAMERTAG. La plier à cette signature changerait son
comportement pour un gain nul.

**Témoins** (`service/compare_weapons_mechanics_test.go`, fichier neuf) : fixture calibrée pour
que le retrait de l'appel se voie sur TROIS classes à la fois — avec les mécaniques, Mêlée 10 /
Capacités spartanes 2 / Non attribué 2 ; sans elles, Mêlée 6 / classe absente / Non attribué 8.
Les parts somment à 100 dans les deux cas. Deux types de fake (avec et sans la méthode) plutôt
qu'un drapeau, parce que ce qui est testé est une TYPE-ASSERTION. Un titre porteur de la
capability est enregistré par swap+restore du registre par défaut (motif de
`skill_v2_shadow_test.go`) — sans quoi les tests mesureraient l'autre branche.

**Contrôle de mutation exécuté** : avec `if false && hasMechanics`, le témoin ROUGIT —
`classe Mêlée = &{Class:melee Kills:6 SharePct:20}, attendu 10 frags`. Code restauré et
vérifié après.

**Commentaire inversé corrigé** (`domain/compare_weapons.go`, et son pendant sur
`buildWeaponSide`). Il annonçait « un joueur dont aucune arme n'est résolue n'a pas de top 3
mais a ses compteurs natifs », alors que `FragClasses` et `TopWeapons` naissent de la MÊME
lecture et tombent donc ENSEMBLE. Le commentaire décrit maintenant le code réel et dit
pourquoi ce choix est assumé (une répartition réduite à trois seaux dont le plus gros est
« Non attribué » se lit comme une panne, pas comme un style — même arbitrage que l'Explorer).
Aucun changement de comportement.

**Gate 3-ter** — exécuté dans cette session :

```
cd apps/go-api && go test ./...        -> EXIT_TEST=0, aucun FAIL
cd apps/go-api && go vet ./...         -> EXIT_VET=0
golangci-lint run --timeout 5m --new-from-rev=05723dce2 <6 arbres>
  0 issues.                            -> EXIT_LINT_NEW=0
make openapi-gen && make generate-types -> EXIT_GEN=0
  git diff --stat openapi.yaml generated.ts : SORTIE VIDE (le contrat ne bouge pas)
```

**Non faits, sur décision du superviseur** : pas de test DuckDB pour
`ResolveWeaponDimensions` ; pas de nettoyage des champs `dimCalls`/`lastDimSlug`/`dimsErr` du
mock `mockWeaponRangeRepo`.

### 2026-09-17 — Lot 4 clos (4.1 à 4.8 tous `[x]`)

**Le renommage `top`/`bottom` est le cœur du lot, et il n'est pas cosmétique.** Le module de
rendu dessine deux bâtons superposés sur une bande. À la Synthèse ce sont « mes frags » et
« mes morts » ; au Face-à-face ce sont DEUX JOUEURS, sur un graphe qui ne montre qu'un côté de
mesure (le titre de carte le nomme). Garder `kills`/`deaths` aurait obligé le compare à ranger
le joueur B dans un champ nommé « morts ». Le module ne connaît donc plus que la GÉOMÉTRIE ;
le sens vit chez l'appelant, qui fournit aussi les deux libellés et les deux encres.

**Rendu Synthèse strictement inchangé**, et c'est vérifié : les dix fichiers de test de
`features/synthesis` restent verts (102 tests). Le seul écart apparu pendant le renommage a
été corrigé plutôt qu'accepté — les noms de PILE du graphe de dénivelé (`stack: 'kills'` /
`'deaths'`) étaient devenus `'top'`/`'bottom'`, ce qui ne changeait rien au rendu mais faisait
rougir un test ; ils sont restaurés, le graphe de dénivelé étant propre à la Synthèse.

**Infobulle min–max (D5)** : `labels.observed` est OPTIONNEL. Le compare le fournit, la
Synthèse non — son infobulle est donc inchangée. Les deux extrêmes ne sont JAMAIS tracés.

**Import cross-feature : le ratchet n'a PAS été relevé.** `compare` importe le graphe de
`synthesis` ; le linter `tools/lint-cross-feature-imports.mjs` plafonne à 7 violations NON
DÉCLARÉES et en comptait 7. La paire a été ajoutée à `ALLOWED_CROSS_IMPORTS` — mécanisme prévu
pour une dépendance DURABLE — en la nommant AU MODULE (`compare=>synthesis/_weaponRangeChart`)
et non à la feature entière, avec justification datée. Le compteur reste à 7, le plafond
inchangé. La place durable du module est `components/charts/` : consigné en Découverte.

**Gate 4** — commande exacte du plan (`Remove-Item node_modules\.tmp` rendu en `rm -rf`, shell
bash) :

```
cd apps/web && rm -rf node_modules/.tmp
npm run typecheck   -> tsc -b, aucune erreur           EXIT_TYPECHECK=0
npm run lint        -> 27 problems (0 errors, 27 warnings)  EXIT_LINT=0
                       AUCUN avertissement sur features/compare/ (vérifié par grep) ;
                       les 27 sont la dette `react-hooks/incompatible-library` préexistante
                       de TanStack Table, sur des fichiers non touchés.
npm run test        -> voir ci-dessous
```

Premier passage de `npm run test` : **1 échec sur 7 885**, `components/shell/NavL1.test.tsx`
— un `Test timed out in 5000ms` sur une suite complète de 487 s (machine saturée), pas une
assertion. Rejoué SEUL : **12 tests PASS en 9,4 s**. Aucun fichier de `components/shell/`
n'est touché par ce lot.

Deuxième passage (suite entière) : **732 fichiers, 7 868 tests, 0 échec, EXIT_VITEST=0** — le
flake ne s'est pas reproduit.

Passage FINAL, après la scission en trois fichiers décrite plus bas (le deuxième passage avait
démarré avant elle) :

```
cd apps/web && rm -rf node_modules/.tmp
npm run typecheck   -> EXIT_TYPECHECK=0                        (log persistant, ligne 5)
npm run lint        -> 27 problems (0 errors, 27 warnings)
                       EXIT_LINT=0                             (ligne 244)
npm run test        -> Test Files  732 passed | 1 skipped (733)
                       Tests  7868 passed | 17 skipped (7885)
                       Duration 432,96 s · EXIT_VITEST=0        (ligne 459)
grep -nE '^ FAIL ' sur le log persistant -> AUCUNE ligne
```

UN SEUL FICHIER A BOUGÉ APRÈS LE DÉPART DE CE RUN (`compare/i18n.ts`, 19:39:04 contre un
départ à 19:38:03), et la modification est un COMMENTAIRE JSDoc : celui de
`weaponsBelowThreshold` annonçait « Sous le seuil de 8 mesures » alors que la chaîne publiée
ne chiffre aucun seuil (doc inversée, anti-pattern n°9 du dépôt — corrigé, voir plus bas). La
sous-suite concernée a donc été rejouée EN AVANT-PLAN sur les bytes finaux :

```
npm run test -- --run src/features/compare src/features/synthesis
  Test Files  14 passed (14) · Tests  149 passed | 14 skipped (163)   EXIT_SUBSET=0
```

**Garde-rails d'outillage (4.8)** :

```
node tools/lint-no-hardcoded-fields.mjs  -> 1976 fichiers, aucune violation   EXIT=0
node tools/lint-no-hardcoded-colors.mjs  -> clean (0 violation)               EXIT=0
node tools/lint-cross-feature-imports.mjs -> 7 <= plafond 7                   EXIT=0
grep hex + classes Tailwind couleur dans features/compare/ -> aucun (hors tests)
```

**Écarts / décisions d'implémentation**

- `sampleNote` de l'item 4.3 n'a pas été écrit : `is_sample` a disparu au lot 3-bis. La note
  « sur N matchs » se lit de `matches`, pour CHAQUE joueur et sans condition.
- Le module pur rend deux fonctions de plus que le plan : `coverageOf` (les deux dénominateurs
  de couverture, qui ne sont pas interchangeables) et `hasWeaponProfile` (la garde de section).
- `roleRangeLines` prend un résolveur de libellé INJECTÉ plutôt qu'une `locale` : le module
  pur ne connaît alors aucun manifeste, et son test n'a pas à en charger un.
- Les classes du bloc 1 sont nommées par le MÊME résolveur que les rôles (`frags.role.<clé>`
  puis `frags.class.<clé>`) : les clés de classe tombent naturellement dans le second essai,
  et deux résolveurs pour deux familles de clés auraient divergé.
- Le bloc « armes les plus utilisées » affiche les armes servies par le back (trois au plus,
  déjà triées) : aucun tri côté front, qui serait une seconde doctrine.
- **La phrase « sous le seuil » ne CHIFFRE pas le seuil**, là où l'item 4.4 citait
  `WEAPON_RANGE_MIN_MEASURED`. Cette constante est un MIROIR de la valeur Go, et le fichier qui
  la porte (`synthesis/weaponRange_logic.ts`) documente qu'elle n'existe qu'une fois ; l'importer
  depuis `compare` coûterait une seconde déclaration cross-feature, la recopier en ferait une
  troisième copie d'un seuil produit — ce que la règle n°6 interdit. La phrase publiée reste
  informative parce qu'elle NOMME chaque rôle écarté AVEC son effectif (« Puissance (6) ») : le
  lecteur voit l'ordre de grandeur sans qu'on lui annonce la borne.
- **Trois fichiers de composant, pas un.** La première écriture faisait 508 lignes avec deux
  fonctions à 83 et 92 lignes — au-dessus des seuils du dépôt (500 / 80). Scindé :
  `CompareWeaponsSection.tsx` (278 L, classes + armes + orchestration),
  `CompareWeaponsRange.tsx` (233 L, le bloc portée) et `compareWeaponsShared.ts` (71 L, les
  encres, les formateurs et le résolveur de libellé que les deux consomment). Les recopier
  aurait donné deux palettes et deux façons d'écrire « 12,4 m ». Fonction la plus longue après
  scission : 73 lignes.

### 2026-09-17 — Lot 5 partiel : 5.1 clos, 5.3 BLOQUÉ par un trou de déclencheur CI

**5.1 — gate complet, toutes les commandes jouées en avant-plan, logs persistants.**

```
cd apps/go-api && go test ./...        EXIT_GO_TEST=0   (aucun paquet hors ok/no-test-files)
cd apps/go-api && go vet ./...         EXIT_GO_VET=0
make go-api-lint  (DEPUIS LA RACINE)   « golangci-lint présent — lint complet (ratchet CI,
                                         dette gelée exclue). »
                                       golangci-lint run --timeout 5m
                                         --new-from-merge-base=origin/main
                                       0 issues.        EXIT_GO_LINT=0
cd apps/web && rm -rf node_modules/.tmp
npm run typecheck                      EXIT_TYPECHECK=0
npm run lint                           27 problems (0 errors, 27 warnings)   EXIT_LINT=0
                                       0 avertissement sur features/compare/ (grep)
npm run test                           Test Files 732 passed | 1 skipped (733)
                                       Tests 7868 passed | 17 skipped (7885)
                                       0 ligne ^ FAIL     EXIT_VITEST=0
```

`origin/main` a été rafraîchi avant le lint (`cf333a388`) et vérifié ANCÊTRE de HEAD : le
ratchet avait donc une vraie merge-base, et son « 0 issues » porte bien sur le diff du
chantier. Aucune correction n'a été nécessaire — pas de commit `compare-armes(lot5)` de code.

**5.3 — commits et push faits ; la vérification CI est IMPOSSIBLE sur cette branche.**

Faits : sept commits `compare-armes(lotN)` sur `wt/compare-armes`, puis
`git push -u origin wt/compare-armes` -> `* [new branch]`, EXIT_PUSH=0.

Bloqué : `gh run list --branch wt/compare-armes` rend une liste VIDE, et le restera. Cause
vérifiée sur pièces, ce n'est pas une latence — elle a été sondée en avant-plan pendant cinq
minutes avant diagnostic :

- `.github/workflows/ci.yml`, bloc `on.push.branches` :
  `[main, feat/**, feature/**, fix/**, hotfix/**, refactor/**, perf/**, docs/**, chore/**,
  integration/**]`. Le préfixe `wt/**` N'Y EST PAS.
- `.github/workflows/gitleaks.yml` reprend cette liste « À L'IDENTIQUE » (son propre
  commentaire le dit) — même trou.
- `ci.yml` n'expose AUCUN `workflow_dispatch` : le run ne peut pas être déclenché à la main.
- Le second déclencheur de `ci.yml` est `pull_request: branches: [main]` — ouvrir une PR vers
  `main` déclencherait la CI, mais la cible de fusion de ce chantier est `feat/v75`, et aucune
  PR n'a été autorisée.

CE N'EST PAS UN ÉCHEC DE LA BRANCHE. La base est saine : `gh run list --branch feat/v75
--limit 3` rend `CI`, `Secrets (gitleaks)` et `Deploy Pre-Check` tous en `completed success`
(run CI 35254287872, 2026-09-17T17:41). Rien n'est rouge — il n'y a simplement rien à lire
pour `wt/**`.

HORS PÉRIMÈTRE, NON CORRIGÉ : ajouter `wt/**` aux deux workflows touche la CI du dépôt
entier, doit se faire DES DEUX CÔTÉS en même temps (le commentaire de `gitleaks.yml` cite le
piège déjà payé : `feat/**` absent de `shared-social-gate.yml` pendant 45 runs) et croise
l'invariant D29 avec le `paths-ignore` de `deploy.yml`, gardé par
`archlint.TestCIPathsIgnoreSubsetOfDeployPathsIgnore`. Décision utilisateur.

Le verdict CI au niveau job reste donc À OBTENIR — il le sera mécaniquement à la fusion dans
`feat/v75` (préfixe couvert), ou plus tôt si l'utilisateur veut élargir les déclencheurs.


## Reprise de session

1. Relire le skill `plan-execution`, puis ce journal.
2. `git worktree list` : reprendre dans `../LevelUp-wt-compare-armes` sur `wt/compare-armes`.
3. Reprendre à la première case non statuée du lot courant. Ne pas re-décider D1 à D11.
