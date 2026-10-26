# Plan : statistiques par carte regroupées sous la carte de base (variantes Heavies, - Ranked)

> Rédigé le 2026-10-26 à partir de `.ai/NOTE_CARTES_VARIANTES_STATS_2026-10-23.md`, relue sur pièces.
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, statut de chaque
> item `[x]` / `[~]` réf / `[!]` justifié, aucune case vide à la clôture, zéro fix hors périmètre).
> Branche : `feat/cartes-variantes-stats`, créée depuis `release/v8.0.7` (06531b561) dans le worktree
> dédié `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-cartes-variantes` (décision user 2026-10-26).
> Chantier voisin : `feat/briefing-seuils` (plan `.ai/PLAN_BRIEFING_EXPLORER_SEUILS_2026-10-26.md`,
> copie principale) touche `match_history_service_briefing*.go` : y limiter l'intervention de E2 aux
> lignes passées à `breakdown.ByMap`, rien d'autre.

## Objectif et critère de succès

Une carte et ses variantes déclarées (`map_variants.toml`) forment UN groupe pour toute statistique
par carte : une vignette dans la Tactique, un taux de victoire « sur cette carte » dans l'Explorateur
et l'Escouade, une ligne par carte dans le briefing, les motifs, les synergies et les séries
temporelles. Critère : sur la base réelle, « Live Fire » et « Live Fire - Ranked » n'apparaissent
plus séparément sur aucune de ces surfaces, et la lecture de placement d'une vignette regroupée
porte sur les matchs de tout le groupe (pied « N sur M » cohérent avec la vignette).

## Relecture de la note du 2026-10-23 (ce qui est juste, ce qui ne l'est pas)

| Affirmation de la note | Verdict sur pièces |
|---|---|
| Filtres regroupés, commit `0fff7a5f0`, fichiers cités | Juste |
| Garde-rail `no_group_by_registry_name_test.go` | Juste, mais il ne couvre que le SQL : `computeMapWinRates` regroupe en Go sur le NOM EN brut du registre (`r.MapName`), le défaut même que le garde-rail vise |
| « computeMapWinRates regroupe par libellé affiché » | Faux : clé = nom EN brut du registre (NULL ou map_id recopié possibles), et c'est une COLONNE par ligne (« taux sur cette carte »), pas un tableau par carte |
| Liste des surfaces | Incomplète. Manquent : briefing de l'Explorateur (`breakdown.ByMap`), Escouade (`computeMapBreakdown`, `LoadMapStatsForSquad`, `squadMapWinRate`), synergies v2 (`breakdown.ByMap`), séries temporelles (`timeseries_service_aggregations.go`), et toute la Tactique au-delà de la grille (lecture de placement, cellule, séquence, équipe, isolé, spawn — tous via l'univers) |
| « Non relevés : citations, carrière, engagement » | Vérifié : aucun agrégat par carte (les citations lisent `map_id` par match) |
| Version légère : « positions, zones, rejeu intacts » | Incohérente pour la Tactique : une vignette regroupée ouvre une lecture par UN map_id, donc compte de la vignette ≠ « N sur M » de la lecture, et mini-plan d'une seule variante. La version légère ne tient que pour la colonne de l'Explorateur |
| Risque « dix fichiers, régression Tactique » | Surestimé : toute la Tactique passe par un seul point, la clause carte de l'univers (`tactical_repo_univers.go`, `? = '' OR mr.map_id = ?`). Passer d'un map_id à une liste de map_id y suffit |
| Argument du mélange statistique | Faible pour « - Ranked » (même carte ; c'est le cas qui pèse : Live Fire 445 parties) ; pour « Heavies », le mode (`BTB Heavies:*`) sépare déjà les parties et le filtre de modes permet de les isoler |

Non vus par la note :
1. **Deuxième mécanisme de variantes, codé en dur** : `games/halo_infinite/adapter_asset_urls.go`
   (`rankedMapSuffix = " - Ranked"`, repli de miniature ajouté le 2026-10-26 ; `" Heavies"` dans
   `mapVariantSuffixes`). Ce plan ajoute un 3e consommateur : règle n°6 (centraliser + garde-rail).
2. **Détection par ligne sur le nom du registre** (`rattacherCartesDeBase*`) : une ligne dont le
   nom est NULL ou le map_id recopié n'est pas rattachée, alors qu'une autre ligne du même map_id
   l'est. Le rattachement doit être une table map_id -> map_id de base.
3. **Repère spatial** : regrouper des positions suppose que variante et base partagent les mêmes
   coordonnées (même niveau). Plausible, non vérifié.
4. **Plusieurs map_asset_id pour un même nom de base** (cartes republiées) : `idsDeCartesParNom`
   prend le plus petit ; le groupe doit contenir TOUS les identifiants de la base.

## Décisions produit (tranchées par le user le 2026-10-26)

- **D1 — Les deux suffixes** (« - Ranked » et « Heavies ») sont regroupés dans les statistiques.
- **D2 — Toutes les surfaces** de E2-E4, Synthèse comprise (signalée par le user). Tendances
  vérifiée le 2026-10-26 : aucun agrégat par carte (`trends_service*.go`, `analysis/trends/`,
  `teammates_service_trends.go`) — rien à faire, re-vérifié au grep de clôture de E4.
- **D3 — Aucune mention** de la variante sur une vignette ou une ligne regroupée : aucune chaîne UI
  nouvelle.

## Étapes

### E0 — Vérifications sur pièces (copie des bases, serveur arrêté ou copie de nuit)
- [x] Compter les parties par map_id variante et par base (relevé à jour du 2026-10-23) -> Mesures, M1
- [x] Les map_id variantes sont-ils au catalogue (`maps_catalog`) ? Sinon, la table se construit
      sur le nom retenu du registre (`nomDeCarteRetenuSQL`), jamais sur le nom par ligne -> M2 : NON,
      la table se construit sur le registre
- [x] Bases à plusieurs `map_asset_id` : lister -> M3
- [x] Repère spatial (22 paires mesurées, M4, minimum 90,3 %, aucun arrêt) : pour 3 paires (Live Fire / - Ranked, Fragmentation / Heavies, une 3e),
      comparer les boîtes englobantes des positions de morts. Recouvrement < 90 % sur une paire =
      ARRÊT et décision user (exclusion spatiale de la paire)
- [x] Fond, zones, callouts (M5) : la base a-t-elle un fond (`MapBackgroundForMap`) ? la variante
      résout-elle la même clé ?
- [x] Filtre de cartes de la Synthèse et de l'Escouade (M6 : défaut CONFIRMÉ par test) (`squadagg.FilterRowsByCascade`, compare
      `assetLabelForFilter(r.Summary.Map)` = libellé PROPRE de la ligne) : choisir « Thunderhead »
      dans le filtre ramène-t-il les parties « Du lourd sur Thunderhead » ? Établir par un test (ou
      sur la page), pas par lecture seule. Si non : défaut de la livraison du 23/10, corrigé en E2
      (même résolveur), pas ailleurs
- Contrainte : le serveur local tient les bases (copie de fichier refusée). Ne PAS l'arrêter (autres
  sessions) : lire par l'API locale (`:8000`, ex. grille Tactique = comptes et bornes par map_id) ou
  par un outil Go passant par `OpenReadForQuery`. Si aucune voie ne marche : ARRÊT, demander au user.
- Gate : relevé écrit dans ce plan (section Mesures), aucune case ouverte.

### E1 — Le résolveur unique (socle)
- [x] `domain` : type `GroupesDeCartes` (map_id -> map_id de base ; `Groupe(id)` rend l'id lui-même
      hors variante ; `Membres(base)` rend base + variantes, triés) -> `domain/map_groups.go` ; le calcul pur est `analysis.GroupMapsByBase`
- [x] `port` : interface de lecture des groupes (une méthode, ctx + slug implicite du PlayerDB)
- [x] `platform/duckdb/map_variants.go` : construction de la table depuis le catalogue (+ repli nom
      retenu selon E0), règle `analysis.BaseMapName`, TOUS les ids de la base ;
      `rattacherCartesDeBase` et `rattacherCartesDeBaseHistorique` réécrits dessus (pas de 2e copie)
- [x] `adapter_asset_urls.go` : le repli « - Ranked » passe par `analysis.BaseMapName` + suffixes du
      TOML (plus de littéral)
- [x] Garde-rail `archlint` : interdit `HasSuffix`/`CutSuffix`/`TrimSuffix` avec un littéral
      `"- Ranked"` ou `"Heavies"` dans le Go non-test hors `analysis/map_variant.go`
- [x] Tests : unitaires du type domaine ; DuckDB `:memory:` du résolveur (variante sans base, base à
      deux ids, nom NULL au registre, titre sans suffixe)
- Gate : `go test ./internal/domain/ ./internal/analysis/ ./internal/platform/duckdb/ ./internal/archlint/ ./internal/games/...`
      E1 CLOS : table construite sur le NOM RETENU DU REGISTRE (pas le catalogue, M2) ; une variante a une
      base quand un map_id du registre porte le nom de base ; groupe = tous les map_id de ce nom + les
      variantes ; représentant = le plus joué (à égalité le plus petit). Deux identifiants de même nom SANS
      variante ne forment pas de groupe (hors périmètre). `rattacherCartesDeBase*` réécrits sur
      `analysis.GroupMapsByBase` (la ligne sans nom est rattachée par son identifiant ; les identifiants de la
      base reçoivent le même libellé). Le garde-rail couvre l'appel direct (littéral, constante) ET la liste
      de suffixes de paquet ; 3 exceptions datées dans `archlint/no_map_variant_suffix_literal_test.go` : la
      liste d'exclusion du tiroir de l'adaptateur (autre notion), et les deux copies du décodeur de film
      (couche fermée, ne peut pas importer analysis) -> `[!]` convergence à statuer avec le décodeur, hors
      périmètre de ce plan (Découvertes). Gate : 6 commandes vertes (domain, analysis, platform/duckdb, archlint,
      games/... ; vet api/port/service ; golangci-lint ratchet 0 issue).

### E2 — Explorateur
- [x] `computeMapWinRates` : clé = `Groupe(map_id)` (plus le nom brut) ; `enrichRow` lit par la même clé
- [x] Briefing (`match_history_service_briefing.go`) : lignes regroupées avant `breakdown.ByMap`
- [x] `breakdown.ByMap` prend le regroupement en paramètre (signature changée = chaque appelant est
      forcé de décider à la compilation ; c'est le garde-rail de E2-E4)
- [x] Synthèse : `buildBreakdownsFromCanonical` (`synthesis_service_legacy.go`, « cartes les plus
      jouées », clé = libellé FR/défaut) regroupe par groupe de map_id, libellé de la base
- [x] Filtre de cartes en cascade (`squadagg.FilterRowsByCascade`) : selon E0, une base retenue
      retient ses variantes
- [x] Tests service : une variante et sa base donnent un seul taux ; carte sans nom au registre ;
      Synthèse : une seule entrée pour base + variante
- Gate : `go test ./internal/analysis/... ./internal/service/...`
      E2 CLOS. Mécanisme : `port.MapGroupReader` injecté dans chaque service (`WithMapGroups`, câblé dans
      `wire/registry_map_variants.go: mapGroupsFor`, mêmes suffixes que les filtres), lu UNE fois par requête
      via la feuille `service/mapgroups.Load` (échec journalisé puis dégradé : variantes seules). La table
      porte le libellé de la base par locale (`GroupesDeCartes.Libelle`, EN + FR) : une ligne regroupée
      n'affiche jamais le libellé d'une variante (D3). `breakdown.ByMap(rows, groups, locale)` : la
      locale choisit la langue du libellé de base ; les 3 appels des synergies (E4) passent la valeur zéro tant
      qu'E4 n'est pas faite (compilent, comportement inchangé, remplacés en E4). `FilterRowsByCascade`
      prend `(rows, domain.CascadeFilter, groups)` : câblé pour la Synthèse, l'Escouade
      (`SquadServiceV2`) et le briefing des coéquipiers, M6 corrigé pour les trois. Clé du taux par
      carte : groupe du map_id, à défaut `name:<nom>` (ligne sans identifiant) ; une ligne sans nom au
      registre mais avec identifiant est désormais comptée. Gate : `go test ./internal/analysis/...
      ./internal/service/...` vert (29 paquets) ; archlint, vet api/platform/port, golangci-lint ratchet 0 issue.

### E3 — Tactique
- [x] `domain.TacticalQuery.MapID` -> liste d'identifiants (groupe) ; clause de l'univers en `IN`
      (`tactical_repo_univers.go`) ; refus de liste vide conservé dans `KillPositions`
- [x] `TacticalService.MapsPlayed` : agrège les rangées par groupe (somme matchs / victoires /
      défaites, MapID = base, libellé de la base) ; `SousPlancher` recalculé après somme
- [x] Vignettes : `MortsParCarte` replié par groupe avant rasterisation
- [x] Toutes les entrées par carte (lecture, cellule, séquence, équipe, spawn, voisines) résolvent
      la base reçue en groupe au même endroit (`requeteDuScope` ou équivalent unique)
- [x] Fond, zones, callouts : clé = map_id de base (E0 a vérifié qu'elle résout)
- [~] Lien « Voir les matchs de cette carte » : porte le libellé de la base ; l'Explorateur ramène le
      groupe (déjà le cas via `MapFilterLabel`)
- [x] Tests : service (deux rangées fusionnées, plancher) ; repo `:memory:` (univers sur deux map_id)
- Gate : `go test ./internal/service/ ./internal/platform/duckdb/ ./internal/api/handlers/` +
  vérification à l'écran par le user sur Live Fire (témoin à nommer par le user)
      DÉCISION USER 2026-10-26 : la vérification à l'écran de E3 est `[~]` REPORTÉE à la vérification finale,
      en une seule fois en fin de chantier (liste dans la section « À vérifier à l'écran »). E3 est close sur
      son gate automatisé.
      E3 : TESTS AUTOMATISÉS VERTS (gate écran reporté, cf. ci-dessus ; avant le report :
      étape non close sans témoin). Mécanisme : `TacticalQuery.MapIDs` (liste du groupe) ; la clause carte de
      l'univers est le jeton `%CARTES%` (`clauseCartes`, `IN (...)`) ; le service résout la carte reçue en
      groupe en UN endroit, `TacticalService.cartesDe` (table lue une fois par service = par requête,
      `tactical_service_groupes.go`), et l'injecte dans `requeteDuScope` / `requeteAvecRetention`, dans les
      4 `TacticalQuery` construites à la main (contextes, mates, journal, univers de la séquence) et dans le
      filtre de matchs canoniques du détail d'une zone. `MapsPlayed` somme les rangées par groupe
      (`fusionnerLesCartes`, MapID = base, libellés = base, plancher recalculé) ; `pointsDuGroupe` réunit les
      morts des membres pour la vignette. Fond/zones/callouts : aucune modification, la carte porte
      l'identifiant de BASE (M5 : le fond de base résout pour 34 groupes ; les zones d'une carte Forge à ids
      multiples ne sont garanties que pour ce représentant). Lien « Voir les matchs » `[~]` : la vignette porte
      `map_name_fr` = libellé de la base (`fusionnerLesCartes`) et l'Explorateur retient le groupe
      (E1/E2 : `MapFilterLabel`). Gate automatisé : `go test ./internal/service/ ./internal/platform/duckdb/
      ./internal/api/handlers/ ./internal/api/wire/` vert ; archlint, golangci-lint ratchet 0 issue.

### E4 — Autres agrégats par carte
- [x] Escouade : `computeMapBreakdown`, `LoadMapStatsForSquad` (repli Go par groupe),
      `squadMapWinRate`
- [x] Synergies v2 (`squad_service_v2_synergies.go`, 3 appels `ByMap`)
- [x] Motifs contextuels (`patterns.analyzeByMap`) : lignes regroupées en amont
- [x] Séries temporelles (`timeseries_service_aggregations.go`)
- [x] Re-grep de clôture : `grep -rn "MapID\]" internal/service internal/analysis` et
      `GROUP BY.*map_id` — tout résultat restant statué
- Gate : `go test ./internal/...` (hors tag gamefiles)
      E4 CLOS. Mécanisme commun : `service/mapgroups` (`Load`, `RefKey`, `RefLabel`) + `squadagg.FoldMapStats`
      (stats « avec l'escouade » repliées par groupe, performance pondérée par `MapSquadStats.PerfCount`,
      champ ajouté et rempli par `LoadMapStatsForSquad`). Escouade V2 : groupes lus une fois par page
      (`GetSquadPage`), lollipop, bullet, perf vs historique, heatmap joueur x carte (`Build*(…, groups)`) ;
      Coéquipiers : `computeMapBreakdown(rows, groups)`, `squadStatsToWinTotal` (chaque identifiant du groupe lit
      le même taux, donc `squadMapWinRate` est inchangé), heatmap joueur x carte. Motifs : `PatternsRepo`
      remplace l'identifiant de carte par celui de la base (`regrouperLesCartes`, suffixes injectés via
      `wire.PatternsRepo`). Séries temporelles : `StatsMatchRow.MapID`, `buildSoloMapBreakdown(…, groups)`,
      Emprise par carte (`timeseriesEmpriseMatches(…, groups)` ; la page Sessions passe la valeur zéro : son
      Emprise n'a pas de découpage par carte, `WithMaps` faux).
      Re-grep de clôture, tout statué : `squad_service_v2_synergies.go:43,79,184,276,323`,
      `teammates_squad_charts_sessions_maps.go:345`, `timeseries_service_aggregations.go:99`,
      `analysis/breakdown/compare.go:28,32`, `patterns/context.go:31` -> `[x]` traités ci-dessus (clés = groupe) ;
      `teammates_service_assets.go:58,214` (libellé par ligne, tri de la clé de groupe), `stats_canonical.go:149`,
      `home_canonical_converters.go:62` (champs par match) -> `[~]` pas des agrégats ; SQL `GROUP BY map_id` :
      `map_variants.go` (le résolveur), `tactical_repo*.go` (repliés par le service, E3) -> `[~]` ;
      `ops/catalog_refresh.go:120`, `sync/replayartifacts/zones_cartes.go:35` -> `[~]` catalogue et zones de
      rejeu, pas des statistiques.
      `[!]` NON TRAITÉ : `analysis/home_canonical_highlights_tiles.go:179` (Accueil, tuile « carte favorite »,
      `sliceFavoriteMapCanonical`). Surface par carte absente de la liste du plan et de D2, que le user n'a pas
      citée ; le traiter ajoute un lecteur de groupes au service de l'Accueil (périmètre du plan : règle n°5).
      Conséquence visible : la carte favorite peut nommer « Live Fire - Ranked » ou « ... Heavies ». À décider
      par le user (REGISTRE_REPORTS). Gate : `go test ./internal/...` vert (169 paquets ok, 0 échec) ;
      golangci-lint ratchet sur `./internal/...` : 0 issue.

### E5 — Web et clôture
- [ ] Contrat OpenAPI inchangé attendu ; si changé : `make generate-types`
- [ ] `make check-types`, `make test-web`, Playwright Tactique en local
- [ ] `make go-api-lint`, `make gate-push`
- [ ] Note du 2026-10-23 close (renvoi vers ce plan) ; `.ai/thought_log.md` ;
      `.ai/REGISTRE_REPORTS.md` pour tout `[!]`

## Mesures (rempli en E0)

Sources (le serveur local tient toutes les bases en écriture : aucune n'a été copiée ni ouverte, et l'API
exige une session SSO) : instantané parquet du 2026-10-26 12:53 (`warehouse/snapshots/v...260/shared/
match_registry.parquet`, 9 592 matchs prêts, 304 map_id ; les 9 540 matchs « partiels » n'y sont pas) pour
les comptes ; sauvegarde `backups/avant_rattrapage_csr_2026-10-15/shared_matches_v2.duckdb` (fichier sans
verrou, ouvert en lecture seule, 268 084 positions) pour le repère spatial ; liste publique
`GET /api/v1/assets/halo_infinite/maps` et sauvegarde `metadata.duckdb` du 2026-10-08 pour le catalogue ;
sidecars de `reference/map_backgrounds/` pour les fonds. Outil jetable `cmd/_e0sql` (non versionné).

**M1 - Comptes (instantané du 26/10).** 42 groupes (carte de base + variante) : 32 en « - Ranked »
(32 map_id, 2 993 matchs), 11 en « Heavies » (11 map_id, 132 matchs), Refuge porte les deux. 67 map_id de
base (4 036 matchs). Plus gros : Live Fire 507 base (3 map_id) + 512 Ranked ; Recharge 486 + 507 ; Streets
469 + 310 ; Solitude 94 + 282 ; Aquarius 376 + 277 ; Lattice 187 et Vacancy 150 en Ranked SANS carte de base
au registre (restent seules). Heavies : Fragmentation 30, Oasis 21, Breaker 19, Highpower 18, Deadlock 11,
Thunderhead 8, Scarr 6, Fortitude 6, Obituary 5, Insolence 5, Refuge 3. Rien ne contredit la note du 23/10
(Live Fire - Ranked : 445 alors, 512 maintenant).

**M2 - Catalogue.** Incomplet : la liste publique ne porte que 101 cartes dont 3 « - Ranked » ; la
sauvegarde du 2026-10-08 (298 lignes, 277 noms) porte 28 variantes mais pas toutes celles du registre, et
UN seul map_id là où le registre en a plusieurs (Live Fire : 2 au catalogue, 3 au registre). Argyle,
Beaver Canyon, Kusini Bay (base) n'y ont pas leur base. CONSÉQUENCE POUR E1 : la table map_id -> base se
construit sur le NOM RETENU DU REGISTRE (`nomDeCarteRetenuSQL`) + `analysis.BaseMapName` ; une variante a une
base quand un map_id du registre porte le nom de base (pas quand le catalogue la porte). Le catalogue ne
sert plus qu'au libellé FR.

**M3 - Bases à plusieurs map_id (registre).** Catalyst 4 ; Live Fire, Recharge, Forbidden, Kusini Bay,
Vallaheim, Exiled 3 ; Streets, Aquarius, Argyle, Fragmentation, Oasis, Breaker, Highpower, Forest, Deadlock,
Scarr, Illusion, The Pit 2. Aujourd'hui chacun est une vignette Tactique à part (une rangée par map_id) : le
groupe doit contenir TOUS les map_id du nom de base (point 4 du plan confirmé, et plus large que « cartes
republiées »).

**M4 - Repère spatial** (positions de mort et de tueur, boîte p1-p99 de chaque ensemble ; recouvrement =
surface commune / surface de la plus petite boîte). 22 paires, toutes >= 90,3 %. Les plus basses : Highpower
Heavies 90,3, Oasis Heavies 92,6, Refuge Heavies 94,0, Forest - Ranked 96,3, Fragmentation Heavies 96,6 ;
les trois paires du plan : Live Fire - Ranked 99,4, Fragmentation Heavies 96,6, Recharge - Ranked 98,6. Les
boîtes min/max brutes sont faussées par des points isolés (Catalyst : un point à x = -249 ; paire Catalyst -
Ranked 87,4 % en brut contre 98,4 % en p1-p99) : la boîte robuste est la bonne mesure. Autre lecture (part
des points de la variante dans la boîte de la base) : 88,4 % pour Highpower Heavies (les véhicules vont plus
loin), 91,3 % à 98,4 % ailleurs. Bases à plusieurs map_id mesurables (>= 100 positions) : Deadlock 98,2 %,
Behemoth 100 % ; les autres map_id doublons n'ont pas de film. Conclusion : même repère pour toutes les
paires ; Highpower Heavies est la paire limite (extension de l'emprise, pas un autre repère : origines à
quelques mètres sur une carte de 70 x 80 m). CAVEAT : si le user préfère exclure cette paire du regroupement
spatial, c'est l'« exclusion spatiale de la paire » prévue par le plan.

**M5 - Fonds, zones, callouts.** Les fonds sont clés par map_id (sidecar) puis par NOM via l'index des
identités, avec repli sur la carte de base (`identiteDeBase`, déjà mesuré : variante et base partagent le
repère à 0,02 m près). Sur les 42 groupes : 34 ont un fond de base ; chaque variante de ces 34 résout un fond (sidecar
propre pour 7 : Fortitude, Insolence, Obituary, Origin, Refuge, Solitude, Thunderhead ; même clé que la base
pour les autres) ; 2 groupes sans carte de base au registre ont un fond de variante (Lattice : sidecar propre ;
Vacancy : repli sur l'identité de base) ; 6 groupes n'ont de fond ni en base ni en variante (Ascension, Gyre,
Inquisitor, Kusini Bay, Rendezvous, Vallaheim) : aucun changement pour eux. Lecture faite par NOM sur les
sidecars (la clé map_id des sidecars n'a pas été rejouée par le service) ; les ids d'une même base y résolvent
la même clé. POINT D'ATTENTION E3 : `mapcatalog/callouts_resolution.go` pose « aucun rapprochement par nom entre
deux map_id » (deux assets de même nom sont souvent deux versions d'une carte Forge dont les zones ne se
superposent pas) ; le groupe est donc résolu sur UN map_id représentatif de la base (le plus joué), pas sur le
nom, et les zones d'une carte Forge à ids multiples ne sont garanties que pour ce représentant.

**M6 - Filtre en cascade (Synthèse, Escouade, briefing des coéquipiers).** DÉFAUT CONFIRMÉ par un test jetable
sur `squadagg.FilterRowsByCascade` (supprimé après lecture) : avec des lignes « Thunderhead » (fr
Thunderhead), « Thunderhead Heavies » (fr « Du lourd sur Thunderhead »), « Live Fire - Ranked » et « Live Fire »,
la sélection « Thunderhead » ne retient que la ligne de base, la sélection « Live Fire » aussi. Le filtre compare
le libellé PROPRE de la ligne alors que les options des filtres sont celles de la carte de base (livraison du
23/10). Corrigé en E2 par le même résolveur. Appelants : `squad_service_v2.go:91`, `synthesis_service.go:204`,
`teammates_service_briefing.go:66`.

## Découvertes (hors périmètre, non traitées)

- **Quatre copies des suffixes de variante, pas deux** : `analysis.BaseMapName` (TOML), `adapter_asset_urls.go`
  (`rankedMapSuffix`, `mapVariantSuffixes`), `film/internal/profile/map_bounds.go` (`variantSuffixes` en
  minuscules, bornes de déquantification, preuve par level_id) et `film/replay/map_background_index.go`
  (`suffixesVariante` sous forme d'identité `_-_ranked`, plus `_firefight`). Seule la 2e est dans le périmètre
  de E1 ; les deux autres vivent dans le décodeur de film (formes et preuves différentes) : à statuer en E1
  (garde-rail ou `[!]` justifié), pas en E0 -> statué en E1 : exceptions datées du garde-rail, convergence `[!]`.
- `computeMapWinRates` sur le nom brut existe depuis avant ce plan ; corrigé en E2 parce que c'est
  la surface visée, pas par opportunisme.

## Reprise de session

Lire ce fichier (cases cochées), `git log --oneline -10` sur la branche, puis reprendre à la
première étape non close.

## À vérifier à l'écran (vérification finale du user, une seule fois ; décision du 2026-10-26)

Base réelle, joueur JGtm (le seul à avoir des films, donc des positions). Rien n'a été vu à l'écran par
l'exécutant : les serveur et bases locaux tiennent la copie principale.

1. **Tactique, grille des cartes.** « Live Fire » ne doit apparaître qu'UNE fois (plus de vignette séparée
   « Live Fire - Ranked »), avec le total des parties des deux (instantané du 26/10 : 507 + 512 parties
   avant filtres). Idem « Recharge », « Streets », « Aquarius », « Solitude », « Empyrean », « Origin »,
   « Forbidden », « Fortress », « Argyle ». Aucune vignette nommée « ... - Ranked » ni « ... Heavies » (hors
   Lattice et Vacancy en Ranked, sans carte de base au registre : elles restent seules). « Fragmentation »,
   « Oasis », « Breaker », « Highpower », « Deadlock », « Thunderhead » : une seule vignette, sans mention
   « Heavies » ni « Du lourd sur ».
2. **Tactique, ouverture de Live Fire.** Le fond s'affiche ; le pied de carte « N sur M » est cohérent avec
   le compte de la vignette ; ouvrir « Où je meurs » puis un détail de zone ne déclenche aucune erreur ;
   « Voir les matchs de cette carte » ramène les parties des deux variantes dans l'Explorateur.
3. **Tactique, Highpower.** Paire limite (90,3 % de recouvrement) : les points de la variante « Heavies »
   doivent tomber sur le fond de « Highpower », sans décalage visible.
4. **Explorateur.** Colonne « taux de victoire sur cette carte » : une ligne « Live Fire - Ranked » et une
   ligne « Live Fire » affichent le MÊME taux et le même nombre de parties. Filtre de cartes : « Live Fire »
   ne présente qu'une option, et la choisir ramène les parties des deux. Briefing (si l'Explorateur l'affiche
   avec un filtre assez large) : une seule carte « Live Fire » dans la dimension carte.
5. **Synthèse.** « Cartes les plus jouées » : « Live Fire » une seule fois, jamais « Live Fire - Classée » ni
   « Du lourd sur ... ». Le filtre de cartes de la Synthèse sur « Thunderhead » retient aussi les parties
   « Heavies » (défaut M6 corrigé).
6. **Escouade et page Coéquipiers** (avec au moins un coéquipier sélectionné) : graphiques par carte
   (lollipop, bullet, perf vs historique, heatmap joueur x carte, heatmap de la page Coéquipiers) : une seule
   colonne « Live Fire ». Filtre de cartes : même comportement qu'en Synthèse.
7. **Séries temporelles.** « Cartes » (découpage par carte) : une seule ligne « Live Fire » avec son
   historique ; l'onglet Usages (Emprise par carte) : une seule carte « Live Fire ».
8. **Motifs** (page Progression / Patterns). Les motifs « par carte » ne listent plus « Live Fire - Ranked » à
   part (n'apparaissent que si le seuil de parties par carte est atteint).
9. **Non couvert, à ne pas attendre** : l'Accueil, tuile « carte favorite » (voir `[!]` E4) ; le rejeu 2D et
   la page de vue d'un match gardent le nom de la variante réellement jouée.
