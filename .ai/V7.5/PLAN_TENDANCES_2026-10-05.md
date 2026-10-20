# Plan d'implémentation — page « Tendances » (2026-10-05)

Source produit : `.ai/MAQUETTE_TENDANCES_2026-10-04.html` (v15) et `.ai/HANDOFF_TENDANCES_2026-10-05.md`.
Branche : `claude/tendances-mockup-review-c21607` (worktree dédié, part de `65aff8ebd`), fusion ultérieure dans
`feat/v75`. Exécution : un exécutant Sonnet à la fois, piloté par un superviseur ; contrat du skill
`plan-execution` (ordre strict, une étape close avant la suivante, aucun report d'une action faisable).

## 1. Objectif et critère de succès

Un 6e onglet « Tendances » dans Ascension montre l'évolution des statistiques d'un joueur sur 7 / 30 / 90 / 365
jours, chaque horizon comparé à la période d'avant de même durée, en vue Solo et en vue Escouade.

Succès : l'onglet s'affiche pour un joueur réel avec tous les blocs de la section 4 ; l'API renvoie pour JGtm
des valeurs cohérentes avec la maquette à date égale (étape 8) ; tous les gates de la section 7 sont verts ;
la revue adversariale (étape 8) n'a plus de constat bloquant ouvert.

## 2. Décisions tranchées (fermes pendant l'exécution)

Décisions de l'utilisateur :
- Horizons 7 / 30 / 90 / 365 j, comparaison à la période d'avant, seuil de 10 matchs de part et d'autre.
- Un seul sélecteur d'horizon pour tout ce qui est sous la matrice ; le pas de temps reste propre à « Évolution ».
- Jamais de « soirée » ni de session ; pas de valeurs attendues du matchmaking.
- CSR pour le classé, LUSR pour le non classé ; le contenu s'adapte aux types de partie que le joueur joue.
- Sens de « Solo » : celui de l'app (matchs sans amis, `match_context = solo`).
- Titre sans MMR ou sans objectifs : page plus pauvre, pilotée par les capacités, jamais par le slug.
- **Interface : uniquement ce qui existe déjà** (blocs, graphiques, titres, bascules). Graphique centré dans
  son bloc, légende en bas et centrée.
- Pas de barre de dégradé en légende (matrice, calendrier).

Décisions du superviseur (conséquences de la contrainte d'interface, relevé du 05/10) :

| Bloc de la maquette | Brique existante retenue | Écart assumé avec la maquette |
|---|---|---|
| Onglet | `features/ascension/AscensionLayout.tsx` (lien en dur) + `navL1Sections.tsx` + `pageTitle.ts` | aucun |
| Bascule Solo / Escouade, horizon, pas | boutons `aria-pressed` au gabarit de `components/shell/FilterOmnibar.tsx:324-345` (aucun composant partagé n'existe) | un seul composant local à la feature, classes recopiées |
| Barre « Horizon » collante | gabarit `sticky top-2 z-10 rounded-lg border bg-card` de `features/ascension/campaign/CampaignTracker.tsx:57` | aucun |
| Filtre « type de partie » | `components/ui/select.tsx` | options = chaînes de performance réellement jouées |
| Matrice | `components/charts/Heatmap2DChart.tsx` en rampe divergente, un graphe par groupe d'indicateurs dans une `SectionCard` | pas de groupes repliables, pas de clic vers le graphique, pas de case grise « pas de comparaison » (case neutre + infobulle) |
| Graphiques « face au MMR adverse » | builder local sur le modèle de `features/squad/charts/squadSessionTimelineChart.ts` (statistique sur l'axe 0, MMR en pointillé sur l'axe 1), rendu par `ChartCard` / `features/timeseries/ChartFromOption.tsx` | courbe à points de taille fixe : ni nuage à taille variable ni tendance lissée |
| Autres courbes d'évolution | même builder, sans axe de droite | aucun |
| Calendrier | `Heatmap2DChart` semaines × jours, rampe divergente sur le taux de victoire, cases vides masquées, sans réglette | pas de case pâle sous 3 matchs |
| Victoires et défaites, médailles | builder local sur le modèle de `features/session-detail/SessionMmrDumbbell.tsx` (deux séries de points + segment) | le coefficient r passe dans l'infobulle |
| Matchs par type de partie | `components/charts/BarStackedChart.tsx` | aucun |
| Sélecteur d'escouade | `components/ui/GamertagCombobox.tsx` + `features/squad/useSquadPresets.tsx`, case « Composition stricte » au gabarit de `features/squad/SquadFilterBar.tsx` | aucun |
| Titres, bulles d'information, états vides | `SectionTitle`, `SectionCard` + `titleWithInfo`, `InfoTooltip`, `EmptyStateNotice`, états de `ChartCard` | aucun |

Autres décisions techniques :
- **Un chargement, une réponse** : l'API renvoie en une fois la matrice, les séries à tous les pas sur 365 j,
  le calendrier, les blocs par horizon. Changer d'horizon ou de pas ne refait aucune requête : le web découpe.
  Un changement de vue, de type de partie ou d'escouade refait la requête.
- **Type de partie** = chaîne de performance (`sync.GetPerformanceChain(slug, pairName, isRanked, isPvE)`), comme
  `service/home_service.go:390`. Les matchs PvE sont hors de la page.
- **Unités** (ADR 0006) : taux et parts en 0..1 dans l'API, mis en pourcentage par le web.
- **Fuseau** : jours, semaines et mois découpés dans `cfg.UserTimezone` ; `now` injectable (tests).
- **Libellés** : clé de champ canonique (`config/titles/*/mappings/fields.toml`) quand elle existe, via
  `useMetricLabel` / `useFieldLabel` ; sinon manifeste `lib/i18n/manifests/tendances.toml` (FR + EN).
- **Points d'état non décidés par l'utilisateur, laissés comme dans la maquette** : écart chiffré absent des
  cases ; graphique « MMR des adversaires et de l'équipe » conservé.
- **Commits** : un par étape, préparé par le superviseur. Le dépôt exige l'accord de l'utilisateur avant chaque
  `git commit` ; sans accord en cours de route, le superviseur garde un patch par étape close (scratchpad) et
  propose les commits à la clôture. Aucun exécutant ne commite.

## 3. Contrat d'API

`POST /api/v1/players/{player_slug}/pages/trends` (sous `r.Route("/players/{player_slug}")`, lecture seule :
le préfixe `/pages/` est déjà exempté par `readOnlyPostPrefixes`). Opération Huma `postTrendsPage`.

Requête `domain.TrendsQueryRequest` :
`view` (`"solo"` par défaut, `"squad"`), `game_type` (vide = tous), `selected_gamertags []string` et
`exact_composition bool` (vue Escouade), `locale` (noms de médailles).

Réponse `domain.TrendsPageResponse` :
- `as_of`, `view`, `game_type`, `timezone` ;
- `game_types []{key, matches}` : chaînes jouées sur 365 j, triées par nombre de matchs décroissant ;
- `capabilities {mmr, csr, lusr, objectives, equipment bool}` ;
- `months []string` : 12 clés `AAAA-MM`, mois courant en dernier ;
- `indicators []TrendsIndicator` : `key`, `variant` (chaîne LUSR, file classée ou gamertag ; vide sinon),
  `group` (`level|results|combat|style|objectives|activity|squad|members`), `unit`
  (`number|ratio|seconds|hours`), `decimals`, `better` (-1, 0, 1), `in_matrix bool`,
  `months []{value *float64, matches int, z *float64}` (12),
  `horizons []{days int, value *float64, matches int, prev_value *float64, prev_matches int, z *float64}`
  (365, 90, 30, 7), `series {match, day, week, month []{t time, value float64, matches int}}` ;
- `calendar []{date, matches, wins, losses, win_rate *float64, performance_score *float64}` (jours joués, 365 j) ;
- `win_loss []{days, matches, required, rows []{key, group, matches, win_mean, loss_mean, z_win, z_loss, r}}` ;
- `medals []{days, compared bool, rows []{medal_id, name, rate, prev_rate *float64}}` ;
- `mix {day, week, month []{t, counts map[string]int}}`.

Règles de calcul (reprises de la maquette) :
- Horizon courant = `]now − d, now]`, période d'avant = `]now − 2d, now − d]`. `value` dès 1 match ;
  `prev_value` et `z` seulement si les deux fenêtres ont au moins 10 matchs.
- Mois : valeur si le mois compte au moins 5 matchs. `z` d'un mois = `better × (valeur − moyenne des mois) /
  écart-type des mois` ; `z` d'un horizon = `better × (valeur − prev_value) / écart-type des mois` ;
  écart-type nul remplacé par 1 ; `better = 0` donne `z = 0`. Le web borne à ± 2,5.
- Séries : `match` sur les 30 derniers jours seulement ; minimum de matchs par point 1 / 2 / 3 / 5 pour
  match / jour / semaine / mois ; point daté du début de son intervalle.
- Découpe par horizon (web) : un point appartient à l'horizon quand le début de son intervalle est dans
  `]now − d, now]`. L'intervalle à cheval sur le début de l'horizon n'est donc pas tracé.
- Taux de victoire = victoires / tous les matchs de la fenêtre. Indicateur absent de `series.match`.
- FDA agrégé = `analysis.AggregateKDA` ; précision = `analysis.Accuracy` sur les totaux ; rendement et
  résistance = `analysis.ComputeCombatYieldFloat` sur les totaux avec `games.EffectiveHpToKill(slug)`.
- CSR et LUSR d'une fenêtre = dernière valeur connue (`SkillSnapshot.RatingValue`), une ligne par
  `PlaylistGroup` ; les matchs de placement n'ont pas de valeur.
- Victoires et défaites : au moins 30 matchs gagnés ou perdus sur l'horizon, sinon `rows` vide ; axe commun en
  écarts-types de la statistique sur l'horizon ; `r` = corrélation de Pearson avec l'issue.
- Médailles : les dix plus fortes variations de taux par match entre l'horizon et la période d'avant ; sans
  période d'avant comparable (`compared = false`), les dix plus fréquentes.
- Ligne sans aucune valeur : omise de la réponse.

Indicateurs de la vue Solo :

| Groupe | Clé (variante) | Définition sur un ensemble de matchs | better |
|---|---|---|---|
| level | `enemy_mmr`, `team_mmr` | moyenne | 1 |
| level | `csr_value` (file), `lusr_value` (chaîne) | dernière valeur connue | 1 |
| results | `win_rate` | victoires / matchs | 1 |
| results | `performance_score` | moyenne | 1 |
| combat | `kda` | FDA agrégé | 1 |
| combat | `damage_balance` | moyenne de (infligés − subis) | 1 |
| combat | `accuracy` | touchés / tirés | 1 |
| combat | `assists_per_match`, `avg_max_killing_spree` | moyenne | 1 |
| combat | `defensive_resistance`, `offensive_conversion` | sur les totaux | 1 |
| style | `avg_life_seconds` | moyenne | 1 |
| style | `headshot_share` | frags à la tête / frags | 1 |
| style | `power_weapon_share` | frags à l'arme lourde / frags | 0 |
| style | `equipment_used_share` | règle de `sessionusage` (étape 3) | 1 |
| objectives | `objective_take_share`, `objective_defend_share`, `objective_hold_share` | part du joueur dans le total de son équipe (étape 3) | 1 |
| activity | `match_count`, `hours_played`, `days_played` | compte, somme de `Self.TimePlayed` / 3600, jours locaux distincts | 0 |
| activity | `dnf_rate` | abandons / matchs | -1 |
| hors matrice | `kills_per_match`, `deaths_per_match`, `avg_damage_dealt`, `avg_damage_taken`, `objective_parity` | moyenne ; parité = moyenne de 1 / effectif du camp | 0 |

Statistiques de « Victoires et défaites » (valeur par match) : `avg_life_seconds`, `damage_balance`, `deaths`,
`kills`, `assists`, `accuracy`, `headshot_kills`, `max_killing_spree` (groupe `stats`) ; `performance_score`,
`kda` natif, `offensive_conversion`, `defensive_resistance` (groupe `composite`) ; `mmr_gap` = MMR de
l'équipe − MMR adverse (groupe `context`).

## 4. Contenu de la page

Solo : (1) matrice par groupe ; (2) barre « Horizon » ; (3) « Évolution » avec boutons de pas, grille
« Face au MMR adverse » (FDA, frags, morts, taux de victoire, score de performance, précision, durée de vie,
rendement + résistance) et grille « Niveau, combat et style » (CSR, LUSR, MMR adversaires / équipe, dégâts
infligés / subis, objectifs + parité) ; (4) calendrier, « Victoires et défaites », médailles, matchs par type
de partie. Pas proposés : 7 j match / jour ; 30 j match / jour / semaine ; 90 j jour / semaine / mois ;
365 j semaine / mois ; défauts match / jour / semaine / mois. Un graphique de moins de 2 points n'est pas
rendu. Sans capacité MMR : pas de groupe « Niveau » MMR, et la première grille se trace sur un seul axe.

Escouade : sélecteur d'escouade + « Composition stricte », matrice, barre « Horizon », six graphiques
(taux de victoire avec et sans l'escouade, FDA par membre, part des frags de l'escouade par membre en barres
empilées, part des frags de l'équipe, écart de MMR, matchs joués ensemble).

## 5. Étapes

Statuts : `[ ]` à faire, `[x]` fait et vérifié, `[~]` couvert par un autre item (lequel), `[!]` non traité
(justification au journal). Aucune case vide à la clôture d'une étape.

Charge estimée : étape 1 lourde, 2 moyenne, 3 moyenne, 4 lourde, 5 moyenne, 6 moyenne, 7 lourde, 8 moyenne.
Une étape est close quand son gate est vert, rejoué par le superviseur, et que tous ses items sont statués.
Aucune dépendance externe bloquante : tout se joue dans le worktree, sur du code et des tests locaux.

### Étape 1 — Go : domaine et analyse pure
Fichiers créés : `internal/domain/trends.go`, `internal/analysis/trends/*.go` (+ `_test.go`).
- [x] 1.1 Types de la section 3 dans `internal/domain/trends.go` (tags JSON de la section 3).
- [x] 1.2 `trends.Match` (ligne aplatie) et `trends.FromCanonical(rows, opts)` : champs lus sur
      `canonical.PlayerMatchRow`, PvE écarté. Le paquet reste pur : la chaîne de performance arrive par une
      fonction injectée `ChainOf(pairName string, isRanked, isPvE bool) string` (le service y branchera
      `sync.GetPerformanceChain`), le seuil de dégâts par frag par un `float64`, le fuseau par un `*time.Location`.
      Aucun import de `internal/sync`, `internal/service` ni `internal/platform`.
- [x] 1.3 Fenêtres : horizon courant / période d'avant, 12 mois locaux, seuils (10, 5).
- [x] 1.4 Registre des indicateurs de la vue Solo de la section 3, hors objectifs et équipement.
- [x] 1.5 Matrice : mois, horizons, `z`, omission des lignes vides.
- [x] 1.6 Séries aux quatre pas via `temporal.BucketByGranularity` (heures converties dans `loc`), minimums par point.
- [x] 1.7 Calendrier, « Victoires et défaites » (écarts-types, Pearson, seuil 30), matchs par type de partie, liste des types joués.
- [x] 1.8 Tests unitaires de chaque fonction exportée, dont les bornes de fenêtre, le seuil de 10, l'écart-type nul, le changement de jour local et un jeu vide.
- [x] 1.9 Test de référence `maquette_test.go` : les 1 158 matchs de la maquette (`testdata/maquette_matchs.tsv`) passés à `BuildSolo` à `now = 2026-09-27T02:00Z` redonnent ce que le code de la maquette calcule (`testdata/maquette_attendu.json` : matrice, filtre de type, victoires et défaites, séries, calendrier, types joués), à 1e-9 près. Les deux fichiers sont produits par le superviseur en exécutant la maquette sous jsdom.
Gate : `go vet ./internal/analysis/trends/... ./internal/domain/...` ; `go test ./internal/analysis/trends/... ./internal/domain/... -count=1` ; `go test ./internal/archlint/... -count=1`.

### Étape 2 — Go : service, port, handler, contrat
- [x] 2.1 `port.TrendsService` (`internal/port/services.go`).
- [x] 2.2 `internal/service/trends_service.go` : `LoadPlayerMatches` par le dépôt en cache, contexte Solo par `Enrichment.IsWithFriends`, filtre `game_type`, capacités, sections de durée (`timing.FromContext(ctx).Section`), horloge et fuseau injectés.
- [x] 2.3 `internal/api/handlers/trends.go` sur le modèle de `timeseries.go`, avec `MapCapabilityError` ; montage dans `internal/api/server_apiv1.go` à côté de `timeseries.Mount` ; fabrique `ServiceRegistry.Trends` dans `internal/api/wire/registry_pages.go`.
- [x] 2.4 Validation de la requête (vue, type de partie inconnu, 4 gamertags au plus) : 400 typé.
- [x] 2.5 `make openapi-gen`, puis `make openapi-check` vert ; chemin ajouté à `TestContractPathsCoverage`.
- [x] 2.6 Tests : service (dépôt simulé : solo seul, filtre, capacité MMR absente), handler (`httptest` : 200, 400, 404, 503 de capacité).
Gate : `go vet -tags=integration ./...` ; `go test ./internal/service/... ./internal/api/... ./internal/port/... ./contracttest/... ./internal/archlint/... -count=1` ; `make openapi-check`.

### Étape 3 — Go : objectifs, médailles, équipement (vue Solo)
- [x] 3.1 Objectifs : rôles par `LoadObjectiveRoleRows` + `LoadFlagGrabsNet`, contexte de camp par `sessionusage.BuildTeamContext`, parts par `sessionusage.ComputeObjectives` appliqué à chaque fenêtre ; lignes et séries `objective_*`, parité ; câblage gardé par `games.CapMatchObjectiveStats`.
- [x] 3.2 Médailles : `LoadMedalsForMatchesByXUID` sur les matchs des fenêtres, noms par `MedalDefinitionsRepo.LookupByIDs`, bloc `medals` aux quatre horizons.
- [x] 3.3 Équipement : lire d'abord `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`, puis réutiliser la lecture d'usage déjà câblée par `TimeseriesService.WithEquipmentUsage` ; ligne `equipment_used_share`, gardée par `games.CapFilmUsageSummary`.
- [x] 3.4 Dégradation : chaque lecture en échec est journalisée (`slog.WarnContext`) et son bloc omis, la page reste servie.
- [x] 3.5 Tests : analyse (parts, parité, médailles comparées ou non), service (chaque capacité absente, lecture en échec).
Gate : celui de l'étape 2.

### Étape 4 — Web : socle, matrice, barre d'horizon
- [x] 4.1 `make generate-types` ; types `TrendsQueryRequest` / `TrendsPageResponse` réexportés dans `lib/api/types.ts`.
- [x] 4.2 Route `routes/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tendances.tsx` ; onglet dans `AscensionLayout.tsx` (dont l'exclusion `isProfile`) ; `features/ascension/i18n.ts` ; `components/shell/navL1Sections.tsx` + `lib/i18n/manifests/common.toml` régénéré ; `lib/pageTitle.ts`.
- [x] 4.3 Tests existants mis à jour : `AscensionLayout.test.tsx`, `NavL1.test.tsx`, `pageTitle.labels.guard.test.ts`, et `buildLegacyRedirect.test.ts` s'il liste les chemins d'Ascension.
- [x] 4.4 Feature `features/tendances/` : manifeste `tendances.toml` + génération, clé `queryKeys.trends` (titre en 2e segment, locale incluse), hook `useTrendsPage`.
- [x] 4.5 Page : bascule Solo / Escouade, filtre « type de partie » (`Select`, libellés par `lusrChainLabel` ou manifeste), états de chargement / erreur / vide existants.
- [x] 4.6 Logique pure `tendances.logic.ts` : découpe par horizon, pas proposés et pas par défaut, formatage par unité, libellé d'un indicateur.
- [x] 4.7 Matrice : un `Heatmap2DChart` par groupe dans une `SectionCard`, valeur affichée par `detail.count`, `valueRange` ± 2,5, infobulle complète, sans réglette.
- [x] 4.8 Barre « Horizon » collante et composant local de bascule (gabarit `FilterOmnibar`).
- [x] 4.9 Tests : logique pure, builder de la matrice, rendu de la page (echarts simulé) dans ses états.
Gate : `npx tsc -b --force` ; `npm run lint` ; `npm run lint:colors` ; `npm run lint:fields` ; `node ../../tools/lint-cross-feature-imports.mjs` ; `node ../../tools/knip-ratchet.mjs` ; `node_modules/.bin/vitest run --pool=forks src/features/tendances src/features/ascension src/components/shell src/lib/pageTitle.labels.guard.test.ts src/lib/query`.

### Étape 5 — Web : « Évolution »
- [x] 5.1 Builder pur `buildTendancesLinesOption` : séries sur l'axe des temps, repère horizontal, moyenne de la période d'avant en pointillé, axe de droite optionnel pour le MMR adverse, légende en bas centrée (`getLegendBase`), jetons `stat-kills` / `stat-deaths` pour frags et morts.
- [x] 5.2 Définitions des graphiques des deux grilles (`tendancesCharts.logic.ts`), règle des 2 points, masquage du taux de victoire au pas « match », dégradation sans MMR.
- [x] 5.3 Section « Évolution » : boutons de pas, deux grilles de `ChartCard` avec titre et bulle d'information.
- [x] 5.4 Tests : builder (axes, repères, légende, couleurs par jeton), définitions (présence selon capacités et pas).
Gate : celui de l'étape 4.

### Étape 6 — Web : calendrier, victoires et défaites, médailles, types de partie
- [x] 6.1 Calendrier : `Heatmap2DChart` semaines × jours sur l'horizon, rampe divergente, `emptyCells="hidden"`, sans réglette, infobulle (date, matchs, victoires, défaites, taux, score de performance).
- [x] 6.2 Builder pur d'haltères (modèle `SessionMmrDumbbell`), couleurs passées en jetons.
- [x] 6.3 « Moyenne par match en défaite et en victoire » (jetons `outcome-win` / `outcome-loss`, valeurs réelles en étiquette, r dans l'infobulle, état « pas assez de matchs »).
- [x] 6.4 « Médailles par match » avec le même builder (horizon et période d'avant ; sans comparaison, un seul point).
- [x] 6.5 « Matchs par type de partie » : `BarStackedChart`, un bâton par pas (par jour au pas « match »), couleurs `LUSR_GROUP_TOKENS`.
- [x] 6.6 Tests : builders, états vides, découpe par horizon.
Gate : celui de l'étape 4.

### Étape 7 — Vue Escouade
- [x] 7.1 (superviseur) Relevé de `internal/service/teammates/` et conception : où vit le calcul, quelles lectures existantes il réutilise, contrat des indicateurs `squad` / `members` ; consignée en section 8 avant 7.2.
- [x] 7.2 Go : matchs de la composition (ADR 0033, option stricte), indicateurs et séries de la section 4, tests.
- [x] 7.3 Web : sélecteur d'escouade, case « Composition stricte », matrice et six graphiques avec les builders des étapes 5 et 6, couleurs `squad-player-1..4`.
- [x] 7.4 Tests Go et web de la vue.
Gate : ceux des étapes 2 et 4.

### Étape 8 — Clôture
- [x] 8.1 Gates complets de la section 7.
- [x] 8.2 Vérification sur données réelles : serveur du worktree lancé avec `LEVELUP_REPO_ROOT` sur le checkout principal (aucun autre serveur sur les mêmes bases), appel de l'API pour JGtm, comparaison aux chiffres de la maquette à date égale par un test d'analyse à `now` fixé si la date diffère.
- [x] 8.3 Revue adversariale du diff par un agent Opus (skill `adversarial-review`, demande de l'utilisateur), puis correction des constats confirmés.
- [x] 8.4 `.ai/thought_log.md`, handoff mis à jour, ce plan clos (cases et journal).

## 6. Règles pour les exécutants

- Lire avant d'agir : ce plan, puis les skills du domaine (`arch-rules`, `go-features`, `canonical-types`,
  `db-schema` côté Go ; `frontend-patterns`, `foundations-usage`, `color-tokens` côté web) dans `.claude/skills/`.
- Périmètre fermé : les items de l'étape, rien d'autre. Toute découverte (bug, dette, idée) va dans le rapport,
  section « Découvertes », sans être traitée, sauf si elle bloque le gate.
- Une décision qui n'est pas dans ce plan ne s'invente pas : s'arrêter et la rapporter.
- Interdits : `git commit`, `git add`, `git stash`, `git checkout`, `git restore`, `git reset` ; toute
  modification sous `.ai/` ; `routeTree.gen.ts` et `apps/go-api/api/openapi.yaml` édités à la main ; un test
  désactivé ou une liste d'exceptions agrandie pour faire passer un gate.
- Commandes en avant-plan uniquement, une seule commande `go` à la fois, jamais de sortie de gate lue à
  travers un tube : chaque commande de gate se termine par `; echo EXIT=$?`.
- Go : `slog.*Context` structuré, aucune erreur avalée, fichiers ≤ 500 lignes, fonctions ≤ 80 lignes,
  ≤ 5 paramètres, pas de comparaison de slug, lecture des tables append-only par les vues `_latest`.
- Web : aucune couleur en dur, aucune chaîne d'interface hors i18n (FR et EN), clés de requête dans
  `lib/query/keys.ts`, FR sans anglicismes, infobulles de trois phrases au plus.

## 7. Gates complets (étape 8)

Depuis `apps/go-api` : `go vet -tags=integration ./...` ; `go test ./... -count=1` ;
`golangci-lint run --timeout 20m --new-from-rev=65aff8ebd ./internal/analysis/trends/... ./internal/domain/... ./internal/port/... ./internal/service/... ./internal/api/...` ;
`make -C ../.. openapi-check`.
Depuis `apps/web` : `npx tsc -b --force` ; `npm run lint` ; `npm run lint:colors` ; `npm run lint:fields` ;
`node ../../tools/lint-cross-feature-imports.mjs` ; `node ../../tools/knip-ratchet.mjs` ; `node_modules/.bin/vitest run --pool=forks`.
Le diff ne touche ni `persist/`, ni `sync/`, ni `migration/` : la suite d'intégration `-p 1` n'est pas requise ;
elle le deviendrait si une étape y touchait.

## 8. Journal et découvertes

Reprise de session : lire la section 2, puis ce journal, reprendre à la première case non statuée.

### Journal
- 2026-10-05 : relevés Go et web (deux agents Sonnet, lecture seule), plan écrit et passé à la grille `plan-review`.
- 2026-10-05, étape 1 : items 1.1 à 1.8 livrés par un exécutant Sonnet (9 fichiers, 4 fichiers de tests). Gates
  rejoués par le superviseur : `go vet` 0, `go test` 0, `gofmt` propre, `archlint` 0 ; `golangci-lint` 0 sur
  le paquet neuf (les 3 constats sur `internal/domain` sont antérieurs, dans d'autres fichiers). Relecture du
  code : conforme à la section 3. Deux retouches demandées (plafond de 64 variantes à retirer, taux d'abandon
  calculé sans `analysis.WinRate`) et item 1.9 ajouté : test de référence contre la maquette.
- 2026-10-05, étape 1 CLOSE : test de référence vert du premier coup (416 cellules de matrice, 39 lignes de victoires et
  défaites, 547 points de série, 146 jours, 5 types joués ; aucun écart avec la maquette, donc aucune correction
  de formule). Contrôle du superviseur : trois corruptions des valeurs attendues font bien échouer le test,
  fichier restauré à l'identique. Retouches faites. Gates rejoués : `go vet` 0, `go test` 0, `golangci-lint` 0.
- 2026-10-05, étape 2 CLOSE : service, port, handler, câblage et contrat de la vue Solo (exécutant Sonnet). Écarts
  acceptés : fabrique dans `internal/api/wire/registry_trends.go` (`registry_pages.go` dépasse déjà 500 lignes) ;
  corps de requête optionnel par `humacore.MarkRequestBodyOptional`, comme `citations.go`. Le contrat régénéré
  ajoute un chemin et 15 schémas, ne retire rien. Les dix gates rejoués en série par le superviseur : tous à 0.

- 2026-10-05, amendement d'ordonnancement : les étapes web (4 à 6) ne dépendent que du contrat de l'étape 2,
  que l'étape 3 ne change pas (elle remplit des blocs déjà typés). L'étape 4 démarre donc pendant l'étape 3 :
  deux exécutants au plus, l'un sans aucune commande `go`, sur des dossiers disjoints (`apps/go-api`, `apps/web`).
  Référence régénérée pour l'étape 3 (lignes d'objectifs et d'équipement de la maquette, 26 lignes de matrice).
- 2026-10-05, étape 3 CLOSE : rôles d'objectif, équipement et médailles de la vue Solo (exécutant Sonnet), sans
  SQL nouveau ni changement de forme du contrat. Le test de référence couvre maintenant 480 cellules et 762
  points, objectifs et équipement compris, sans écart. Choix de l'exécutant acceptés : prises nettes illisibles
  ou noms de médailles illisibles = bloc omis avec avertissement ; une ligne sans valeur de mois ni d'horizon
  reste omise même si une série courte existerait ; locale normalisée en `fr` / `en`. Les dix gates rejoués en
  série par le superviseur : tous à 0.
- 2026-10-05, étape 4 CLOSE : onglet, route, navigation, données, matrice et barre « Horizon » (exécutant Sonnet).
  Écarts acceptés : `TrendsQueryRequest` écrit à la main côté web (le corps de la requête n'a pas de schéma dans
  le contrat) ; réponse précédente gardée pendant un changement de type de partie, au même joueur et au même titre ;
  clé de manifeste en plus pour l'horizon non comparé faute de matchs récents ; libellés d'indicateur résolus par
  une fonction unique (`useIndicatorLabeler`). Les attributs `data-horizon` / `data-step` de la page sont à retirer
  à l'étape 5. Gates rejoués par le superviseur : typage, lints, exports inutilisés et 1 716 tests, tous à 0.
  Les gates web du plan citaient `tools/` depuis `apps/web` : chemin corrigé en `../../tools/`, ratchet des
  exports inutilisés ajouté.
- 2026-10-05, étape 5 CLOSE : section « Évolution » (exécutant Sonnet). Ajouts acceptés : titre de l'état vide ;
  dates des infobulles dans le fuseau de la réponse ; graphiques CSR, LUSR, MMR et objectifs de la seconde grille
  affichés selon les capacités. Gates rejoués par le superviseur : typage, lints, exports inutilisés et 2 177 tests,
  tous à 0. À regarder au contrôle visuel : un repère horizontal hors de l'étendue des données ne se voit pas
  (l'axe est à l'échelle des données).
- 2026-10-05, item 7.2 CLOS : vue Escouade côté Go (exécutant Sonnet, repris après son arrêt sur l'écart de MMR).
  Référence : 3 compositions, 496 cellules et 581 points sans écart. Écarts acceptés : `SquadTrendsCtx` appelle
  `TeammatesCtx` puis retrouve le service concret (un garde-rail lit le corps de `TeammatesCtx` par son nom et
  interdit de l'extraire) ; le handler décode et valide le corps avant de construire un service (corps invalide
  sur joueur inconnu : 400 au lieu de 404). Gates rejoués par le superviseur, élargis à tout `internal/service`
  et `internal/api` : tous à 0. À corriger en clôture : `GetSquadTrends` suppose `WithTrends` appelé (horloge nil
  sinon).
- 2026-10-05, étape 6 CLOSE : calendrier, victoires et défaites, médailles, matchs par type de partie (exécutant
  Sonnet). Écart accepté : la carte des types de partie est une `SectionCard` autour du wrapper sans cadre (le
  wrapper ne prend qu'un titre texte). Le test `langSegmentInheritance.test.ts` avait dépassé son délai chez
  l'exécutant pendant que les gates Go tournaient : rejoué deux fois seul sur machine au calme, 3 tests verts en
  1,6 s. Gates rejoués par le superviseur : typage, lints, exports inutilisés et 2 232 tests, tous à 0.
- 2026-10-05, item 8.2 FAIT (superviseur) : serveur de la branche lancé sur une COPIE isolée des bases réelles
  (614 Mo, sans jetons : aucune synchronisation possible), port 8017. Vue Solo de JGtm : 200 en 0,47 s à froid puis
  0,13 s, 293 Ko ; vue Escouade : 200 en 0,23 s, 98 Ko ; 400 sans gamertag ; 404 joueur inconnu. Les lectures sur
  730 jours d'identifiants ne posent pas de problème de durée. Comparaison des horizons aux formules de la maquette
  à la même date (`as_of` de la réponse), sur ses matchs extraits :
  - Solo : 377 valeurs identiques, 15 à moins de 0,5 % (LUSR arrondi dans la maquette), 8 écarts tous expliqués :
    heures jouées (temps joué du joueur côté app, durée du match côté maquette) ; part de l'équipement (884 matchs
    mesurés aujourd'hui contre 84 à l'extraction de la maquette) ; une valeur d'avant du LUSR que l'API ne publie
    pas quand l'horizon courant n'a pas de valeur.
  - Escouade non stricte : 171 identiques, 5 à moins de 0,5 %, aucun écart (434 matchs, comme la maquette).
  - Escouade stricte : 304 matchs sur 365 j contre 388 dans la maquette. Voulu : la composition stricte de l'app
    écarte tout coéquipier connu hors sélection, la maquette seulement les trois coéquipiers suivis. La route
    légère de la page Escouade compte 310 matchs stricts sur tout l'historique : même population.
- 2026-10-05, items 7.3 et 7.4 CLOS : vue Escouade côté web (exécutant Sonnet). Sélection locale à l'onglet,
  coéquipiers fréquents par la liste légère de l'onglet Tactique, bouton « Enregistrer » inactif faute du xuid
  du joueur. Gates rejoués par le superviseur : typage, lints, exports inutilisés à 0 ; suite web complète
  (9 028 tests) verte sauf trois tests de garde sans rapport, en dépassement de délai pendant que les relecteurs
  tournaient, verts rejoués seuls (7 tests en 1,1 s).
- 2026-10-05, revue adversariale, ronde 1 (item 8.3) : trois relecteurs Opus aveugles les uns aux autres.
  - Données Go : aucun constat recevable, 30 conditions vérifiées.
  - Architecture Go : 2 constats, 22 conditions vérifiées. P1 : deux fonctions équivalentes de test de capacité
    de titre dans `internal/api/wire`. P2 corrigé quand même : doublons de gamertags acceptés en vue Escouade
    (parts supérieures à 1 par appel direct de l'API).
  - Web : 8 constats, 20 conditions vérifiées. P0 : le filtre « type de partie » affiche « Toutes les parties »
    alors qu'un type absent de la liste reste appliqué (changement de vue ou de joueur). P1 : clé brute
    « ranked » affichée pour le CSR ; « matchmaking » dans deux textes français ; barres empilées d'escouade
    rendues avec un seul bâton ; `TendancesTab` de plus de 80 lignes. P2 corrigés quand même : « −0 » affiché ;
    total d'un bâton à 99 % ; bulle d'« Évolution » qui parle de moyenne pour le CSR et le LUSR.
  - Demandes de l'utilisateur ajoutées au lot de corrections : la sélection d'escouade reprend celle de la page
    Escouade ; le bouton « Enregistrer » devient actif (la réponse de la vue Escouade porte ses membres avec
    leur xuid).
- 2026-10-05 — Corrections Go de la ronde 1 closes (exécutant Sonnet, diff relu par le superviseur). Une seule
  fonction de test de capacité dans `internal/api/wire` (`titleHasExpectedStats` appelle `titleHasCapability`) ;
  `Validate` refuse deux gamertags égaux (espaces de bord retirés, casse ignorée) ; `GetSquadTrends` ne garde
  qu'un membre par xuid et ignore le xuid du joueur principal ; erreur explicite si `WithTrends` n'a pas été
  appelé ; la réponse porte `members` (xuid et gamertag ; joueur principal d'abord en vue Escouade, vide en vue
  Solo). Tests neufs dans trois fichiers, aucun test existant modifié, références de la maquette inchangées.
  Gates rejoués par le superviseur, en série : `go build ./...`, `go vet -tags=integration` (internal,
  contracttest, openapi-gen), tests de `analysis`, `domain`, `port`, `service/teammates`, `service`,
  `api/handlers`, `api/wire`, `contracttest`, `archlint`, `openapi-gen -check`, `golangci-lint
  --new-from-rev=65aff8ebd` (0 issue), `gofmt -l` (rien) : dix codes de sortie à 0.
- 2026-10-06 — Corrections web de la ronde 1 et demandes de l'utilisateur closes (exécutant Sonnet, diff relu
  par le superviseur). Filtre « Type de partie » : `TendancesTab` devient une enveloppe qui remonte
  `TendancesPage` par couple (titre, joueur) ; `gameTypeOptions` garde toujours le type appliqué dans la liste.
  `variantLabel` écrit « Classé » / « Ranked » pour le groupe de file du CSR (matrice et courbe). Plus de
  « matchmaking » en français (garde `tendances.manifest.test.ts`). Barres empilées : `MIN_POINTS` partagé.
  `TendancesTab` découpé en `TendancesControls`, `TendancesBody`, `useTendancesSquadSelection`. Zéro sans
  signe dans `formatTrendValue`. Parts d'un bâton au plus fort reste (`roundSharesToTotal`). Bulle
  d'« Évolution » : « valeur sur la période d'avant ». Sélection d'escouade partagée avec la page Escouade :
  `features/squad/squadSelectionStorage.ts` (coéquipiers) et `exactComposition.ts` (option stricte) sont les
  seuls lecteurs et écrivains des deux clés `localStorage`, `SquadLayout` et `useSquadSessionSelection` les
  appellent sans changement de comportement. « Enregistrer » actif : `TendancesSquadPicker` reçoit
  `members`, `useSquadPresets` prend `readonly SquadPresetRow[]`. Test existant modifié :
  `localStorage.clear()` au `beforeEach` de `TendancesTab.squad.test.tsx` (la sélection persiste désormais).
  Gates rejoués par le superviseur : `tsc -b --force`, eslint (0 erreur, aucun avertissement dans les dossiers
  touchés), `lint:colors`, `lint:fields`, imports inter-features (7, plafond 7), knip (0) : six codes de sortie
  à 0 ; vitest ciblé par l'exécutant (1 196 tests).
- 2026-10-06 — Revue adversariale, ronde 2 (un relecteur Opus neuf, lecture seule, sur le diff des seules
  corrections : instantané d'avant corrections contre l'arbre de travail). 13 corrections sur 15 tiennent
  (C1 à C5, W1 à W4, W7 à W10). Deux constats recevables, P0 + P1 = 2 (6 en ronde 1) : (1) `roundedParts`
  arrondit par `toFixed`, qui travaille sur la valeur binaire : 1,45 s'écrit « 1,4 » là où Intl (le reste de
  l'app) écrit « 1,5 » — valeur d'affichage changée d'une unité du dernier chiffre sur les cas de mi-chemin ;
  (2) deux fonctions de plus de 80 lignes restent dans `features/tendances/` (`getTendancesText`, 128 lignes,
  et `buildTendancesDumbbellOption`, 86 lignes). Les deux sont corrigés dans le lot (exécutant Sonnet) ; pas
  de ronde 3 : le superviseur relit ces deux corrections lui-même et le dit à l'utilisateur.
- 2026-10-06 — Corrections de la ronde 2 closes (exécutant Sonnet, relues par le superviseur). `tendances.logic.ts` :
  plus aucun `toFixed` ; `numberParts` écrit `|valeur × échelle|` par `formatNumber` (Intl arrondit) et le signe
  suit le texte (nul quand aucun chiffre de 1 à 9) ; `formatTrendValue` et `formatTrendDelta` le partagent ;
  tests 1,45 → « 1,5 », 1,005 → « 1,01 », 2,675 → « 2,68 », écart 0,145 → « +0,15 », −0,004 → « 0,00 », les cas
  « zéro sans signe » gardés. `i18n.ts` : `getTendancesText` assemble quatre constructeurs (`pageMatrixTexts`,
  `evolutionTexts`, `blockTexts`, `squadTexts`, 30 à 36 lignes chacun), même type `TendancesText`.
  `tendancesDumbbell.logic.ts` : `dumbbellAxes`, `segmentsOf`, `segmentSeries` extraits,
  `buildTendancesDumbbellOption` à 49 lignes, test existant inchangé et vert. Gates web de l'exécutant à 0
  (tsc, eslint, couleurs, champs, imports, knip, vitest `features/tendances` : 202 tests).
- 2026-10-06, étape 8 CLOSE. 8.1 : gates complets de la section 7 rejoués par le superviseur sur l'état final,
  en série. Go : `go vet -tags=integration ./...` (0), `go test ./... -count=1` (0 ; 196 paquets ok, 153 sans
  test), `golangci-lint --new-from-rev=65aff8ebd` sur les paquets touchés (0 issue, rejoué après les corrections
  Go, code Go inchangé depuis), `openapi-gen -check` à jour. Web : `tsc -b --force` (cache purgé), eslint
  (0 erreur, aucun avertissement dans les dossiers touchés), `lint:colors`, `lint:fields`, imports
  inter-features (7, plafond 7), knip (0), `vitest run --pool=forks` complet : 846 fichiers et 9 035 tests passés,
  5 fichiers / 23 tests sautés, tous préexistants et hors du lot. 8.3 : deux rondes (entrées ci-dessus).
  8.4 : entrée `.ai/thought_log.md` du 2026-10-06, handoff mis à jour (section « Implémentation »), mémoire
  projet mise à jour. Rien n'est commité : les commits sont à proposer à l'utilisateur. Hors plan, pour
  l'utilisateur : gate visuel dans l'app après fusion, CI de la branche.

### Conception de la vue Escouade (item 7.1, superviseur, 2026-10-05)

- **Où vit le calcul** : la population d'une composition reste celle de la page Escouade (ADR 0033). Le service
  `internal/service/teammates` gagne `GetSquadTrends(ctx, playerXUID, req)` (nouveau fichier), qui réutilise
  `LoadTopTeammates`, `lireComposition`, le filtre de composition exacte et `lireEquipeAlliee`. Aucun SQL nouveau.
- **Une seule lecture de l'équipe alliée par requête** : elle sert à la fois au filtre « composition stricte » et
  aux statistiques des membres. Le filtre de `appliquerCompositionExacte` est extrait dans une fonction qui reçoit
  les camps déjà lus ; le comportement de la page Escouade ne change pas.
- **Matchs** : lignes canoniques du joueur principal (`LoadPlayerMatches`, même variante en cache), restreintes aux
  matchs de la composition ; les matchs sans amis servent au taux de victoire « sans l'escouade ».
- **Calcul pur** : `trends.BuildSquad`. Indicateurs du groupe `squad` : `win_rate`, `win_rate_alone` (matchs sans
  amis, mêmes fenêtres), `match_count`, `squad_share_of_team_kills` (frags des membres / frags de l'équipe),
  `mmr_gap` (MMR de l'équipe − MMR adverse). Groupe `members`, variante = gamertag : `kda` (FDA agrégé du membre),
  `member_share_of_squad_kills`. Calendrier, victoires et défaites, médailles et types de partie par pas : vides.
- **Port et handler** : `port.SquadTrendsService` ; le handler des tendances reçoit une seconde fabrique et route
  sur `view`. `Validate` accepte la vue Escouade avec 1 à 3 gamertags.
- **Dépendances injectées par le câblage** (fuseau, horloge, classement en type de partie, seuil de dégâts par
  frag, capacités) : le paquet `teammates` n'importe pas `internal/sync`.
- **Référence** : `testdata/maquette_attendu.json` porte un bloc `squads` (trois compositions de la maquette, dont
  une stricte) ; le test de référence le compare à `BuildSquad`.

### Découvertes (non traitées)
- `Enrichment.FriendsXUIDs` de `canonical.PlayerMatchRow` est déclaré mais jamais renseigné.
- `LoadObjectiveRoleRows` et `LoadFlagGrabsNet` lient leurs `match_id` en `IN (?, ?, …)`, pas en un paramètre
  `VARCHAR[]` (ADR 0036, invariant 2).
- Le handler des séries temporelles n'appelle pas `MapCapabilityError`.
- Le pseudo-code du KDA dans l'ADR 0006 contredit `analysis.AggregateKDA`.
- Le README de `internal/analysis/temporal` décrit `StartTime()` ; le code attend `GetStartTime()`.
- Le gabarit de bascule segmentée est recopié à quatre endroits sans composant partagé ; `PeriodFilter` n'a aucun appelant.
- La maquette calcule l'écart de MMR par `m.tm - m.em` : un match sans MMR y compte pour 0 (`null - null` en
  JavaScript). Deux matchs sont concernés. Le Go et la référence de la vue Escouade les écartent ; la maquette
  n'est pas corrigée.
- Les lectures d'objectifs, d'équipement, de participants et de médailles reçoivent jusqu'à 730 jours
  d'identifiants de match en `IN (?, …)` : coût à mesurer sur un joueur réel (étape 8).
- `lireComposition` (`teammates_service_composition_legere.go`, partagée avec `CompositionSessions`) charge les
  matchs de chaque gamertag avant tout dédoublonnage : un gamertag qui se résout vers le joueur principal, mêlé
  à de vrais coéquipiers, vide l'intersection, donc la composition. Atteignable par appel direct de l'API
  seulement (le sélecteur du web exclut le joueur) ; la page Escouade a le même comportement.
