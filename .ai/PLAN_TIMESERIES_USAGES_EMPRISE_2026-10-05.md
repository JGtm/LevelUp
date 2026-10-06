# Plan : Séries temporelles › Usages → l'Emprise du périmètre solo — 2026-10-05

> Sources, à lire avant tout lot, qui FONT FOI pour le rendu :
> - maquette validée par l'utilisateur le 2026-10-05, position « Après » :
>   https://claude.ai/artifact/BR8veZfoaQrbhqNKuU8Uk2 (v4), copie
>   `.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html` (script lisible : `renderApres` l. 862,
>   `renderEquip` l. 1124, `renderMine` l. 1149, `renderFil` l. 1187, `gridColumns`/`renderGrid`
>   l. 1233-1292) ; les lignes « remplace : … » et les encarts « Maquette. » ne se portent pas ;
> - relevés : `.ai/V7.5/MESURES_TIMESERIES_USAGES_2026-10-05.md` ;
> - `.ai/thought_log.md`, entrée « [2026-10-05] Sessions, Séries temporelles, Vue match : relevé des
>   rendus v1… » et ses compléments 1 à 3 ;
> - plan précédent de la même famille : `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md`
>   (décisions D2-D12, spécification S1-S13 reprises ici quand elles s'appliquent).
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, un lot à la fois, gate passé avant le
> suivant, aucun item sans statut, zéro fix hors périmètre, découvertes consignées §8). Statuts :
> `[x]` fait et vérifié, `[~]` couvert ailleurs (référence), `[!]` non fait (justification écrite).
> Aucune case vide à la clôture d'un lot. « Clos » = les 5 actions de la règle 6 du skill.
>
> Statut du plan : **GO du superviseur le 2026-10-06 (phase 2, lot par lot, compte rendu et
> « continue » à chaque clôture)**. Branche : `feat/ts-usages-emprise` (créée sur `origin/feat/v75` = `65c99b669`),
> worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-ts-usages`.

## 0. Objectif, critère de succès, hors périmètre

**Objectif.** L'onglet « Usages » des Séries temporelles devient l'Emprise appliquée aux matchs
SOLO du périmètre filtré (même liste de matchs que le reste de la page : `filteredCanon`,
`match_context` forcé à `solo` par `features/timeseries/TimeseriesPage.tsx:85-89`), dans l'ordre et
les formes de la maquette « Après ». « Riposte » quitte l'onglet Progression. Tout ce qui perd son
dernier lecteur est supprimé (règle n° 7), Go et web, contrat compris.

**Critère de succès.** (1) Les 8 blocs du §3 rendus dans l'ordre, conformes à la maquette (S1-S12) ;
(2) chaque carte se retire seule sans donnée et sur Halo 5 (jamais un 500, jamais un zéro inventé) ;
(3) tous les gates des lots verts, dernière exécution dans la session ; (4) inventaire §4 supprimé,
chaque preuve grep à 0 ; (5) docs du lot clôture à jour.

**Hors périmètre** (consigné, non traité) :
- Page Sessions (`features/session-detail/`) : rien de ce qu'elle lit ne bouge (`_shared/usage/*`
  qu'elle importe, bloc `coordination` riposte comprise, `SessionUsageBlock`).
- Escouade › Synergies (retrait de « Rôles de hauteur », « Riposte », « Temps de riposte », « Morts
  ripostées ») : décision utilisateur du 2026-10-05, lot à part non lancé.
- Vue match : son bloc d'élévation est `MatchElevationBlock` (`domain/match_view.go:527`), distinct.
- Le DTO `CoordinationBlock` garde sa riposte : la page Sessions la lit (`SessionDetailPage.tsx:408`).
  Les Séries temporelles continuent de recevoir le bloc entier (même producteur
  `buildCoordinationBlock`) ; seule la carte web Riposte disparaît.
- Escouade › Emprise : comportement INCHANGÉ ; seules ses briques deviennent paramétrables (L4).

## 1. Décisions

### 1.1 Validées par l'utilisateur (2026-10-05, maquette v4) — fermes

- **V1** Ordre et contenu de l'onglet : §3 (8 blocs).
- **V2** Grille « carte par carte » : une colonne par carte jouée, les 12 plus jouées puis
  « Autres cartes » qui somme le reste (maquette `gridColumns`, l. 1233-1243 : repli seulement
  au-delà de 13 cartes) ; en-tête : nom, nombre de matchs, bilan V / D (+ A s'il y en a).
- **V3** « Mes vies : près d'un coéquipier ou seul » : distance au coéquipier le plus proche À
  L'INSTANT DE LA MORT, seuil = portée du radar du match ; barre épaisse = mes vies terminées par une
  mort, barre fine = mes frags pendant ces vies ; frags par vie de chaque côté ; vies sans
  coéquipier situé ou d'un match sans portée connue : écartées ET comptées (le ⓘ le dit). Source
  `match_death_context_latest` ⨝ `match_lives_latest` ⨝ `match_kill_events_latest`, jamais
  `match_life_placement`.
- **V4** « Ma part à l'objectif » : la SEULE fiche est celle du joueur affiché (gamertag + emblème),
  pas de « Reste de mon camp » ; la barre d'une action = sa part du total de son camp ; familles
  côte à côte dans la fiche.
- **V5** « Hauteur d'engagement » retirée de la page ; « Riposte » retirée de Progression ;
  « Appui reçu » reste tel quel.
- **V6** Sur les pages solo on écrit « Mon camp », jamais « Notre camp ».

### 1.2 Tranchées par le planificateur — FERMES (confirmées par le superviseur le 2026-10-06 : D1-D14 confirmées, D7 / L7 maintenu en avant-dernier lot — gate qui résiste = arrêt et compte rendu, jamais forcé —, D15 : ligne d'attribution de la session de l'exécuteur)

- **D1 — Pas de déplacement vers `_shared`, import direct de `features/squad/*`.** Vérifié sur
  pièces : `tools/lint-cross-feature-imports.mjs:157` déclare la paire `'timeseries=>squad'` dans
  `ALLOWED_CROSS_IMPORTS` ; les imports timeseries → squad ne comptent donc PAS dans le plafond 7
  (`RATCHET_THRESHOLD`, l. 386), et la page en importe déjà quatre modules
  (`TimeseriesPage.summary.tsx:40-41`, `TimeseriesRangeRolesCard.tsx:38,47`,
  `TimeseriesSquadAdapted.tsx:37`). Déplacer ~35 fichiers de `squad/emprise` ferait du bruit sans
  gain de gate. Les briques réutilisées prennent leurs textes en paramètre (`t: EmpriseText`,
  `t: ObjectifText`) : le contexte solo passe ses propres textes (D10). *Contredit le brief §3.*
- **D2 — Le bloc solo est un type à part qui EMBARQUE l'Emprise.** `domain.SoloEmpriseBlock` =
  `SquadEmpriseBlock` embarqué (champs aplatis dans le JSON, même mécanisme que
  `EquipmentUsageFamilyLine` qui embarque `SessionUsageOutcomes`, `domain/equipment_usage.go:70-75`)
  + `maps` (grille par carte) + `equipment` (carte Équipement) ; champ de réponse
  `TimeseriesPageResponse.Emprise` (`json:"emprise,omitempty"`). Côté web le type généré est
  structurellement assignable à `SquadEmpriseBlock` : toutes les briques squad le lisent tel quel.
  Pas de placement (`Placement` nil), pas d'habitude (`Timeline` vide → `Habit` nil, `build.go:88`).
- **D3 — La grille par carte se calcule en Go** (`squademprise.BuildMaps`), pas dans le web : elle
  agrège des états de mesure par match (film, camp, niveaux de socle, véhicules) — règle métier. Le
  repli « 12 + Autres cartes » est aussi en Go (constante `domain.EmpriseGridMaxMaps = 12`). Chaque
  colonne est un « soirée » (`soiree`, `build.go:14-58`) sur les matchs de la carte : une seule règle
  de lisibilité pour la soirée, le match et la carte. V / D / autres par carte : depuis
  `canonical.MatchSummary.Outcome` (`games/canonical/match.go:31`), résolu par le service et passé
  dans `squademprise.Match` (nouveaux champs `MapKey`, `MapLabel`, `Outcome`).
- **D4 — Carte « Équipement » : périmètre et familles.**
  - Périmètre : les matchs MESURÉS du bloc (filmés à camp connu, `matchTally.bonusMeasured`,
    `match.go:93`) pour MOI comme pour le reste de mon camp — un seul périmètre des deux côtés.
  - « Servi / gardé / lâché » et « pris sur la carte » : `sessionusage.PlayerOutcomeCounts`
    (`usage_outcomes_counts.go:34`), donc la bascule unique `equipmentUsedOf` (mur = posé, autres =
    charge consommée).
  - Familles « mesurées » : `equipmentusage.EquipmentOutcomeFamilies()` privé des deux bonus
    (`sessionusage.PowerupFamilies()`), dans l'ordre de la table (`domain/equipmentusage/families.go:54-63`).
  - Familles « non mesurées » (ligne « Non mesuré », compte de MES lâchers) : grappin et propulseur,
    exportés par `domain/equipmentusage` (nouvelles constantes `EquipmentFamilyGrapple`,
    `EquipmentFamilyThruster`, liste `EquipmentUnmeasuredLineFamilies()`) ; le décodeur
    (`games/halo_infinite/film/replay/usage_summary_families.go:43-44`) RELIT ces constantes au lieu
    de ses littéraux — même patron que mur / capteur (l. 45-48) ; valeur identique, aucune révision.
    Vérifié : `domain/equipmentusage` n'est dans le périmètre d'aucune couche révisée (aucun
    `*_perimetre.golden` ne le cite ; couches révisées = `film/revision/couches.go:22-30`).
    Répulseur : aucune ligne (décision P4, maquette).
  - Ordre d'affichage : celui de la maquette (`equip` des données : grappin, mur, capteur,
    translocateur, écran, traqueur, champ, propulseur) = non mesurées encadrant les mesurées ; publié
    par le Go dans cet ordre (`EquipmentUnmeasuredLineFamilies()[0]`, mesurées, puis `[1]`).
- **D5 — Carte « Mes vies » : unité et règles.**
  - Unité : une vie de `match_lives_latest` du joueur avec `end_cause = 'death'`
    (`migration/steps_shared_match_lives.go:69-74` : seule cause qui dise qu'il est mort).
  - Fenêtre d'une vie : `[start_ms, start_ms de sa vie suivante)` — LA règle de rattachement du
    décodeur (`replay/placement_des_vies.go:36-41`, `rattacherLesFrags` l. 335-360). La fin du film
    (`end_ms`) n'est PAS l'instant de la mort au journal (`lives_export.go:66-69` : fin de réplication
    convertie), d'où la fenêtre et non une égalité de clé.
  - Mort de la vie : la ligne de `match_death_context_latest` (victime = joueur) dont `time_ms` tombe
    dans la fenêtre ; « près » si `nearest_teammate_m ≤ portée` (borne INCLUSIVE, même comparaison
    que `coordination.accompagnee`, `analysis/coordination/isolation.go:72-74`, factorisée).
  - Écartées et comptées : (a) `ExcludedUnlocated` = pas de ligne de contexte dans la fenêtre, ou
    `nearest_teammate_m` NULL (aucun coéquipier visible, équipe à terre comprise) ; (b)
    `ExcludedNoRadar` = vie d'un match dont la variante n'a pas de portée.
  - Frags d'une vie : lignes de `match_kill_events_latest` `publishable`, `feed_killer_xuid` = le
    joueur, victime d'un AUTRE camp (camps de `match_participants`) ; trahison ou camp inconnu :
    écarté, compté au journal (pas au contrat).
  - Non réutilisé, et pourquoi : `TacticalRepo.MortsAvecContexte`
    (`platform/duckdb/tactical_repo_isolement.go:87`) exige la position de la victime (INNER JOIN
    `kill_positions`) et charge un univers par carte (`TacticalQuery`), et ne rend ni vies ni frags.
    Réutilisés : `mappings.PorteeDuRadar` (via D6), la comparaison de `accompagnee`.
- **D6 — Portée du radar par match : 3e copie → helper.** Deux boucles « variantes → portée,
  compte des sans-portée » existent (`service/tactical_service_isolement.go:159-174`
  `rayonsParMatch` ; `service/teammates/teammates_service_emprise_placement.go:124-136`
  `rayonParMatchDuScope`) ; les Séries temporelles en feraient une 3e. Règle n° 6 : nouveau
  `mappings.PorteesDuRadarParMatch(table map[string]int, variantes map[string]string)
  (map[string]float64, int)` à côté de `PorteeDuRadar` (`games/mappings/portee_du_radar.go:24`), les
  deux copies migrées, garde-rail étendu (`archlint/no_local_radar_range_lookup_test.go`).
- **D7 — `formes_retenues` réduit à ce que l'objectif lit.** Les 9 cartes solo de
  `features/squad/formes/` étaient les DERNIERS lecteurs des champs non-objectif du bloc (grep §4.F).
  Règle n° 7 : `Lobby`, `PadNamed`, `PadUnnamed`, `WeaponPads`, `DurationSeconds`, `TeamSize`,
  `LobbySize`, `Measured` de `SquadFormesMatch`, `Weapons` du bloc, leurs types et leur production
  (`analysis/squadformes/formes.go:56-71,114,121-126,166-180,210-262,389-436`,
  `port/session_usage.go:47-49` `LoadUsageFilmPads`, sa mise en œuvre DuckDB) sortent du code et du
  contrat. Les champs que lisent les cartes d'objectif restent : `available`, `unavailable_reason`,
  `matches_total`, `matches_measured`, `main_xuid`, `squad`, `matches[].{match_id, start_time,
  mode_label, map_label, player_team, objective}`. *Lot à part (L7), le plus risqué ; si le
  superviseur préfère le différer, L7 passe `[!]` avec cette justification et le reste du plan tient.*
- **D8 — Chaîne `equipment_usage` supprimée (Go).** Seul producteur restant :
  `service/timeseries_service_sections.go:104` ; seul lecteur web : `TimeseriesPage.usages.tsx:57,96`.
  Sortent : `domain/equipment_usage.go`, `squadagg.BuildEquipmentUsageBlock` et ses aides
  (`squadagg/equipment_usage.go:44-64,97-196` ; `LireUsage`/`LecturesUsage` l. 66-95 RESTENT : lus par
  l'Emprise et les formes), l'alias `service/squadagg_reexport.go:19,28`,
  `sessionusage/usage_overview.go` (+ test) et la comparaison `metricKeys`/`overviewFamilies` de
  `usage_outcomes_test.go:334-385` (le critère `subjectBilanFamilies` reste, lu par `metricKeys`),
  le résolveur d'amis du service (`timeseries_service.go:106`, `timeseries_service_sections.go:47-54,297-303`).
  `squadagg.NommerArmesDesNiveaux` RESTE (lu par `session_page_usage_labels.go:127`).
- **D9 — Chaîne « nuage d'élévation » supprimée (Go + web).** Lecteurs uniques :
  `ElevationCard.tsx` / `_elevationCloudChart.ts` côté web, `TimeseriesPageResponse.Elevation`
  (`domain/timeseries.go:340`) côté Go. Sortent : `domain/elevation_cloud.go`,
  `analysis/elevation_cloud.go` (+ test), `service/elevation_cloud_section.go` (+ test), le second
  retour de `buildWeaponRangeSections` (`weapon_range_section.go:55-94,151-172` : `hydrateLabels`
  ne nomme plus que la portée).
- **D10 — Textes solo.** Fichier `features/timeseries/usages/usagesText.ts` : `EMPRISE_TEXT_SOLO`
  et `OBJECTIF_TEXT_SOLO` (`Record<Locale, …>`) construits par surcharge de `EMPRISE_TEXT`
  (`squad/emprise/empriseStrings.ts:500`) et `OBJECTIF_TEXT` ; textes FR de la maquette mot pour mot
  (titres, ⓘ, légendes), EN traduits ; « Mon camp » / « My side », « matchs du périmètre » /
  « matches in scope ». Plus les textes propres aux cartes neuves (Mes prises, Équipement, Mes vies,
  grille par carte, Ma part).
- **D11 — Disposition.** « Portée par arme » reste seule dans sa grille à deux colonnes
  (demi-largeur, maquette `renderRange` l. 444-484 : `grid2` sans voisine) ; « Appui reçu » reste
  seule dans `TimeseriesCoordinationSection` (`lg:grid-cols-2`, demi-largeur, « tel quel »).
- **D12 — « Au fil des matchs » sur une période.** Même graphe que l'Escouade
  (`squad/emprise/empriseCharts.ts:165-212`), nouveau mode d'axe `period` : sous l'axe, la date du
  premier match de chaque mois (et un trait), légende « n matchs, dont m filmés » à droite ; points
  plus petits au-delà de 120 matchs (maquette `renderFil` l. 1188, 1204) ; pas d'encoche de
  dominance (la maquette n'en dessine pas). Résultat, date et carte joints depuis `match_rows`
  (`TimeseriesMatchRow` : `outcome`, `start_time`, `map_name(_fr)`, `domain/timeseries.go:191-256`).
- **D13 — Lignes « en attente » de la maquette non portées.** Les lignes « 0 prise… », « Non mesuré
  sur les N matchs » du solo réel suivent le comportement des briques de l'Emprise : une ressource
  sans prise n'a pas de ligne (`emprise.logic.ts:65-72`), les véhicules suivent leur couverture
  (lot L7 de l'Escouade). La phrase conditionnelle du ⓘ de « Frags obtenus » (maquette l. 928) n'est
  pas portée (elle décrit les données d'illustration).
- **D14 — Emblème de la fiche « Ma part ».** Publié par le Go sur la réponse
  (`TimeseriesPageResponse.PlayerEmblemURL`, `json:"player_emblem_url,omitempty"`), lu par le même
  chargeur que l'Escouade (`SquadV2LoaderAdapter.LoadEmblemURLs`,
  `platform/duckdb/squad_v2_adapter.go:358`) derrière un port étroit (`port.EmblemURLLoader`) ;
  absent → l'initiale (repli existant de `SquadSheetAvatar`).
- **D15 — Attribution des commits** : ligne système de cette session
  (`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`), le brief en citait une autre (signalé
  au compte rendu de phase 1).

## 2. Spécification de rendu (non négociable)

Reprend S1-S13 du plan du 2026-09-26 (§2) ; en cas de doute, la maquette fait foi, puis ce §2.

- **S1** Titre factuel, ⓘ si besoin (3 phrases au plus, texte de la maquette), graphique, légende
  centrée en bas — rien d'autre (aucune phrase de lecture, aucun pied de carte).
- **S2** Valeurs DANS les segments si elles tiennent (mesure au pixel,
  `components/charts/segmentLabelFit.ts`), repli au-dessus sinon, jamais seulement en infobulle.
- **S3** `team-ally` / `team-enemy` à la place des mots ; hachure réservée à « sans film ».
- **S4** Trait 50 % pointillé `warning`, « 50 % : autant que l'adversaire » ; « parité » n'existe plus.
- **S5** Aucun vert / rouge sur une répartition interne (moi / reste de mon camp : `squad-player-1`
  / `team-rest`).
- **S6** Couleurs de ressource `resource-*` ; pastille devant chaque nom de ressource.
- **S7** FR + EN pour toute chaîne (`Record<Locale, …>`), « FDA » jamais « KDA », aucun anglicisme
  (garde `lib/i18n/no-anglicisms.guard.test.ts`), aucun emoji.
- **S8** Aucune couleur en dur ni classe Tailwind de couleur (skill `color-tokens`,
  `tools/lint-no-hardcoded-colors.mjs`).
- **S9** « Mon camp » (V6).
- **S10** Chaque carte se retire seule sans donnée ; onglet sans rien → l'état vide existant
  (`timeseries.usages.empty_*`, `TimeseriesPage.usages.tsx:61-68`).

## 3. Cartes cibles (ordre à l'écran)

| # | Bloc / carte | Source Go (contrat) | Composant web |
|---|---|---|---|
| 1 | « Portée des engagements » : 4 tuiles, « Portée par arme », « Rôles de portée » | INCHANGÉ (`weapon_range`, `range_profiles`) ; `elevation` retiré | `WeaponRangeSection` sans `ElevationCard`, `TimeseriesRangeRolesCard` |
| 2a | « Bilan du périmètre » › « Contrôle des ressources » | `emprise.resources` | `squad/emprise/ResourceControlCard` + `EMPRISE_TEXT_SOLO` |
| 2b | « Contrôle des ressources au fil des matchs » | `emprise.matches` + `match_rows` (résultat, date) | `squad/emprise/ResourceFilCard`, axe `period` (D12) |
| 3 | « Carte par carte » › « Contrôle des ressources, carte par carte » | `emprise.maps` (D3, NEUF) | `timeseries/usages/ResourceMapGridCard` sur `squad/emprise/ResourceGridTable` (extrait) |
| 4 | « Mes prises » › « Mes prises dans mon camp » | `emprise.objects[].squad` (moi, reste) + `resources[powerup].outcomes` (bonus perdus) | `timeseries/usages/MinePickupsCard` (NEUF), modèle `buildPickupSheets` existant |
| 5 | « Prendre, et s'en servir » › « Frags obtenus avec les ressources » + « Rendement face à l'adversaire » | `emprise.production` | `squad/emprise/ProductionCard`, `YieldCard` |
| 6 | « Près d'un coéquipier ou seul » › « Mes vies : près d'un coéquipier ou seul » | `lives_near_teammate` (D5, NEUF) | `timeseries/usages/LivesNearTeammateCard` (NEUF) |
| 7a | « Objectif » › « Rapport de force par famille de mode » | `formes_retenues` (objectif) | `squad/objectif/ObjectiveBalanceCard` + `OBJECTIF_TEXT_SOLO` |
| 7b | « Ma part à l'objectif » | `formes_retenues` + `player_emblem_url` (D14) | `squad/objectif/ObjectiveSoloSheetCard` (NEUF) |
| 8 | « Équipement » › « Équipement pris, et ce que j'en ai fait » | `emprise.equipment` (D4, NEUF) | `timeseries/usages/EquipmentOutcomesCard` (NEUF) |
| P | Progression : « Appui reçu » seule | `coordination` inchangé | `TimeseriesCoordinationSection` sans `CarteRiposte` |

Contrats neufs (Go, `internal/domain/`, fichier `solo_emprise.go` et `timeseries_lives.go`) :

```go
// SoloEmpriseBlock — l'Emprise du périmètre solo des Séries temporelles (D2).
type SoloEmpriseBlock struct {
    SquadEmpriseBlock
    Maps      []EmpriseMapColumn `json:"maps"`               // D3, plus jouée à gauche
    Equipment *EmpriseEquipment  `json:"equipment,omitempty"` // D4 ; nil sans film
}
type EmpriseMapColumn struct {
    MapKey, MapLabel  string                      // vides pour « Autres cartes »
    OtherMaps         int                         // > 0 : colonne de repli, nb de cartes sommées
    Matches, MatchesFilmed, MatchesMeasured, MatchesTiers, VehiclesMeasured int
    Wins, Losses, Others int
    Resources         []SquadEmpriseMatchResource // mêmes objets (squad = moi, reste) que match par match
    PowerWeaponKills  *SquadEmpriseCount
}
type EmpriseEquipment struct {
    MatchesMeasured int
    Families        []EmpriseEquipmentFamily      // ordre D4
}
type EmpriseEquipmentFamily struct {
    Family    string
    Measured  bool
    Me, Rest  *EmpriseEquipmentOutcomes           // Measured seulement
    DroppedMe int                                 // non mesurées seulement
}
type EmpriseEquipmentOutcomes struct{ Taken, Used, Kept, Dropped int }

// TimeseriesLivesNearTeammate — « Mes vies : près d'un coéquipier ou seul » (D5).
type TimeseriesLivesNearTeammate struct {
    Near, Alone         LivesSideCount // {Lives, Kills int}
    ExcludedUnlocated   int
    ExcludedNoRadar     int
    MatchesRead         int
    MatchesWithoutRadar int
}
```
(tags JSON en snake_case, `omitempty` sur les pointeurs ; noms définitifs fixés au lot, ce bloc
fige la FORME.)

## 4. Inventaire des suppressions — preuves par grep (relevées le 2026-10-05, à REJOUER avant de supprimer)

Chaque preuve se rejoue par `Grep` (outil) sur `apps/web/src` (hors `lib/api/generated.ts`) ou
`apps/go-api`. Attendu APRÈS suppression : 0 occurrence hors fichiers supprimés.

- **A. `EquipmentUsageSection` et ses cinq cartes (web).** `EquipmentUsageSection` n'est importé que
  par `features/timeseries/TimeseriesPage.usages.tsx:26`. Ses dépendances propres
  (`_shared/usage/EquipmentUsageSection.tsx:39-52`) : `UsageCountsGrid`, `usageCountsModel`,
  `UsageEquipmentDonutCard`, `usageEquipmentPartiesModel` — aucun autre lecteur (grep des noms hors
  `_shared/usage/`) ; `usagePadTiersModel`, `usageAvailability`, `usageCardTitle`,
  `UsageEmptyNotice`, `usageI18n` RESTENT (lus par `session-detail`). Clés `usageI18n` propres aux
  cartes retirées : celles que knip / le grep laissent sans lecteur. Gardes à adapter :
  `_shared/usage/usageEmptyStateCanonical.guard.test.ts`, `noLocalUsageCopies.guard.test.ts`
  (vérifier qu'ils ne citent pas un fichier supprimé ; les adapter, jamais les désactiver).
- **B. « Les formes retenues » (web).** Importée par `TimeseriesPage.usages.tsx:29` seulement.
  Survivants de `features/squad/formes/` (importés hors du dossier, grep `../formes/`) :
  `cardsI18n.ts` (familles, lu par `SquadObjectiveSection.tsx:28`), `colors.ts` (Emprise),
  `i18n.ts` (`columns`, lu par les cartes d'objectif), `format.ts`, `model/objectives.ts`. Sortent :
  `FormesRetenuesSection.tsx` (+ test), `FormesCard.tsx`, `viewModel.ts`, `richText.ts`, `scales.ts`,
  `cards/*` (4), `forms/*` (8 dont `GrilleForm.test.tsx`), `model/access.ts`, `aggregates.ts`
  (+ test), `display.ts` (+ test), `pads.ts`, `padsObjectives.test.ts` (la partie objectif migre dans
  `objectivesOptional.test.ts` si elle teste un survivant), `formes.fixtures.ts` si plus aucun test ne
  l'importe. Dans les survivants : clés et types sans lecteur (`EquipmentAxis`, `WeaponClass`, textes
  des cartes de `cardsI18n.ts` hors `families`, encres d'axes de `colors.ts`), et
  `aggregateColumns` réduit à `{team, lobby}` (seuls champs lus par `objectif.logic.ts:79-81`),
  `aggregateRole` supprimé s'il n'a plus de lecteur. Juge de paix : knip 0 / 0 / 0.
- **C. `ElevationCard` (web) et nuage d'élévation (Go).** Web : `ElevationCard.tsx`,
  `_elevationCloudChart.ts` (+ test), import et rendu dans `WeaponRangeSection.tsx:52,362-369`, props
  `elevation` / `matchRows` (`WeaponRangeSection.tsx:296-302`, lus seulement par la carte), clés
  `synthesis.weapon_range.*` propres à la carte (manifest `lib/i18n/manifests/synthesis.toml`,
  régénéré par `node apps/web/scripts/build_i18n_manifests.mjs`), cas de tests
  `WeaponRangeSection.test.tsx` / `.options.test.tsx` qui l'exercent. Go : D9.
- **D. Carte Riposte (web).** `CarteRiposte` (`TimeseriesCoordinationSection.tsx:174-235`),
  `secFmt` (l. 73-76), `delaiMedianS` (`timeseriesCoordination.logic.ts:115-118`) si plus lu, chaînes
  de riposte de `timeseriesCoordinationStrings.ts` (`riposteTitle`, `covered`, `iRiposte`, `delay`,
  `riposteTooltip`, `volMyDeaths`, `volTeamDeaths`), cas de `TimeseriesCoordinationSection.test.tsx`.
- **E. `equipment_usage` (Go + contrat).** D8 ; web : `EquipmentUsageBlock` et alias de
  `lib/api/types.ts:2327-2330`, fixture `test/handlers.ts:242`, cas
  `TimeseriesPage.sections.test.tsx:133-216`.
- **F. Champs de `formes_retenues` (Go + contrat).** D7 ; lecteurs web après B : `objectif/*` et
  `formes/model/objectives.ts` seulement (grep `SquadFormes` : 19 fichiers relevés, dont 11 sortent
  en B).
- **G. Bouts devenus morts** : `timeseries.usages.equipment_title` (`timeseries.toml:914`) ; types
  TS retirés du snapshot `lib/api/contract-surface.snapshot.json` par la procédure documentée
  (`UPDATE_CONTRACT_SURFACE=1`), disparitions listées au journal du lot.

## 5. Organisation et gates communs

- Exécuteur seul, dans le worktree, lots SÉQUENTIELS. Aucun sous-agent, aucun push, aucun merge,
  aucun `git stash`, aucun `git add -A` (stager fichier par fichier), aucun `--no-verify`, aucun
  Python, aucune base de `data/` ouverte, aucun serveur arrêté ou relancé, une commande `go` à la fois.
- Environnement Go, à chaque appel PowerShell :
  `$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"`.
  Web : `npm ci` dans `apps/web` du worktree (node_modules réel) avant le premier gate web.
- **Gate Go** (depuis `apps/go-api`) : `go build ./...` ; `go vet` des paquets touchés ;
  `go test` des paquets touchés puis `go test ./...` ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...`
  dès que `platform/duckdb` bouge (et `./internal/sync/...`, `./internal/persist/...`,
  `./internal/migration/...` s'ils bougent — aucun lot ne les prévoit) ; `go test ./internal/archlint/...` ;
  `make go-api-lint` (Git Bash) ; contrat : `go run ./cmd/openapi-gen`, puis
  `go run ./cmd/openapi-gen -check`, `npm run generate-types` (dans `apps/web`) et
  `node tools/check-generated-types-fresh.mjs` (racine).
- **Gate web** (depuis `apps/web`, vitest hors sandbox) : purge `node_modules\.tmp` ;
  `npx tsc -b --force` ; `npm run lint` (0 erreur) ; `npx vitest run --pool=forks` ; depuis la
  racine : `node tools/knip-ratchet.mjs` (0 / 0 / 0), `node tools/lint-no-hardcoded-colors.mjs`
  (0), `node tools/lint-cross-feature-imports.mjs` (≤ 7, aucune dérogation morte),
  `npx lefthook run pre-push`.
- **Seuils** (CLAUDE.md règle 5) : fichier ≤ 500 L, fonction ≤ 80 L, ≤ 5 paramètres, complexité ≤ 12
  pour tout fichier créé ou modifié ; mesure jointe au journal du lot
  (`git diff --name-only <base>.. | ForEach-Object { "{0} {1}" -f (Get-Content $_).Count, $_ }`).
  Attention : `service/timeseries_service.go` est à 424 L, `domain/timeseries.go` à 379 L,
  `squad/emprise/empriseStrings.ts` à 501 L (ne pas l'agrandir : D10 dans un autre fichier).
- **TDD** : chaque règle neuve a son test ROUGE écrit et vu rouge AVANT le code ; puis vert ; puis
  au moins UNE MUTATION par règle (modification volontaire du code qui doit faire rougir le test,
  annulée ensuite), consignée au journal du lot (« mutation : … → rouge »).
- **Clôture de lot** = gate vert + items statués + section du lot mise à jour ici + entrée en FIN de
  `.ai/thought_log.md` + commit local `feat(ts-usages/<lot>): …` (fichiers stagés un par un) + point
  d'étape au superviseur.

## 6. Lots

### L1 — Go : portée du radar par match et lectures de l'Emprise factorisées · moyen

Refactorisation sans changement de comportement, préalable à L2 et L3.

- [x] L1.1 `games/mappings/portee_du_radar.go` : `PorteesDuRadarParMatch(table, variantes)` (rend
  la table par match et le nombre de matchs sans portée) ; test pur (`portee_du_radar_test.go` :
  variante connue, inconnue, blanc de tête, portée nulle) écrit rouge d'abord.
- [x] L1.2 Migrer `teammates/teammates_service_emprise_placement.go:73,117-136`
  (`rayonParMatchDuScope` supprimé, son test `teammates_service_emprise_placement_test.go:162-170`
  déplacé vers L1.1) et `service/tactical_service_isolement.go:61,159-174` (`rayonsParMatch` bâtit
  la carte des variantes des matchs `Mesure` puis appelle le helper).
- [x] L1.3 Garde-rail : `archlint/no_local_radar_range_lookup_test.go` — nouvelle empreinte « appel à
  `PorteeDuRadar(` hors du helper » sur les fichiers NON test, allowlist nommée
  `sync/killcollector/capture.go` (résolution d'UNE variante à l'écriture, `capture.go:102`) ;
  auto-test qui prouve que l'empreinte reconnaît les deux anciennes boucles (littéraux copiés de
  L1.2) ; mutation : réintroduire une boucle dans un fichier de test temporaire hors allowlist →
  rouge.
- [x] L1.4 `service/squadagg/emprise_lectures.go` (NEUF) : sortir de `TeammatesService` les trois
  lectures de l'Emprise en fonctions libres paramétrées (repo, joueur, page pour les journaux) —
  `LireFeuilleEmprise` (`teammates_service_emprise.go:89-106`), `LireFilmEmprise` + `raisonFilm`
  (l. 108-172), `LireVehiculesEmprise` (`teammates_service_emprise_vehicles.go:37-63`). Noms
  d'événements : `emprise_*` avec l'attribut `page` (`teammates` / `timeseries`) ; les tests qui
  citent `teammates_emprise_*` (5 fichiers, 27 occurrences relevées) sont mis à jour.
  `TeammatesService` appelle ces fonctions ; comportement inchangé
  (`teammates_service_emprise_test.go`, `_vehicles_test.go`, `_placement_test.go` verts sans
  changement d'assertion autre que le nom d'événement).
- Gate : gate Go (sans contrat : aucun type public ne change) ; preuve
  `Grep "PorteeDuRadar\(" apps/go-api/internal --glob !*_test.go` → helper + `capture.go` +
  `PorteesDuRadarParMatch` seulement.

Journal L1 (2026-10-06, exécuteur, `feat/ts-usages-emprise`) :
- **L1.1** `mappings.PorteesDuRadarParMatch(table, variantes)` (`games/mappings/portee_du_radar.go`) ; test `TestPorteesDuRadarParMatch` écrit d'abord, vu ROUGE (symbole indéfini), puis vert. En-tête du fichier réécrit au présent (il racontait les deux copies, règle 17).
- **L1.2** `rayonParMatchDuScope` supprimée (`teammates_service_emprise_placement.go`), son test supprimé (couvert par L1.1) ; `TacticalService.rayonsParMatch` bâtit la carte des variantes des matchs `Mesure` et appelle le helper.
- **L1.3** Empreinte 3 du garde-rail `archlint/no_local_radar_range_lookup_test.go` : tout appel `PorteeDuRadar(` dans un fichier de production hors du helper est refusé, allowlist nommée `internal/sync/killcollector/capture.go` (une variante à l'écriture) ; auto-test sur les deux anciennes boucles et deux faux positifs (`AvecPorteeDuRadar(`, appel du helper).
- **L1.4** `service/squadagg/emprise_lectures.go` : `EmpriseLecteur{Page, Player, RepoRoot, TitleSlug}` avec `Feuille`, `Film` (+ `ajouterHabitude`, `raisonFilm`), `Vehicules`, et `EmpriseMatchIDs` ; `TeammatesService` les appelle (`lireFeuilleEmprise`, `lireFilmEmprise`, `raisonFilm`, `lireVehiculesEmprise`, `matchIDsOf` supprimés). Événements renommés `emprise_*` avec l'attribut `page` ; seuls les trois `teammates_emprise_vehicules_*` étaient assertés (`teammates_service_emprise_vehicles_test.go`), mis à jour (+ assertion `"page":"teammates"`) ; les journaux du placement restent `teammates_emprise_placement_*` (non déplacés). Écart au plan, assumé : la lecture du film n'avait AUCUN test de l'habitude (mutation « habitude jamais lue » VERTE sur la suite teammates) — ajouté `squadagg/emprise_lectures_test.go` (habitude lue en plus du périmètre et niveaux sur les deux, échec de l'habitude qui la dégrade seule, sans repo → `film_unsupported`).
- **Mutations** (script `mutation.ps1` du scratchpad, restauration garantie, diff vérifié après) : `sans++` retiré du helper → ROUGE (`TestPorteesDuRadarParMatch`) ; appel `PorteeDuRadar` réintroduit dans `tactical_service_isolement.go` → ROUGE (`TestNoLocalRadarRangeLookup`) ; capability non supportée de la feuille rendue en `sheet_load_failed` → ROUGE (`TestTeammatesService_GetPage_EmpriseCapabilityNonSupportee`) ; habitude jamais lue → ROUGE (`TestEmpriseLecteurFilm_LHabitudeEstLueEnPlusDuPerimetre`) ; échec de l'habitude ignoré → ROUGE (`TestEmpriseLecteurFilm_LHabitudeEnEchecDegradeSeule`, après avoir rendu le double mordant : l'échec porte sur la lecture des joueurs, les films se lisant).
- **Gate** (CGO, une commande `go` à la fois, avant-plan) : `go build ./...` sortie 0 ; `go vet` mappings, squadagg, teammates, service, archlint sortie 0 ; `go test -count=1` des paquets touchés : 5 ok ; `go test -count=1` du module en six lots couvrant tout `go list ./...` (cmd + contracttest + analysis + api + domain + port + archlint : 67 ok en 79 s ; games : 39 ok en 62 s ; platform + service : 28 ok en 67 s ; sync + persist + migration : 13 ok en 91 s ; reste de internal : 45 ok en 36 s ; pkg + scripts + tests : 3 ok) — aucun FAIL ; `make go-api-lint` : 0 issues ; `go run ./cmd/openapi-gen -check` : à jour ; `gofmt -l` muet ; preuve grep `PorteeDuRadar\(` hors tests : déclaration et appel dans le helper, `capture.go` seulement. `-tags=integration` non requis (aucun paquet `platform/duckdb`, `sync`, `persist`, `migration` modifié).
- Seuils : plus gros fichier touché `tactical_service_isolement.go` 244 L ; aucune fonction neuve au-delà de 45 L ; `Film` a 5 paramètres (ctx compris).

### L2 — Go : le bloc Emprise du périmètre solo · lourd

Périmètre : `analysis/squademprise/{input.go, maps.go (NEUF), equipment.go (NEUF)}` (+ tests),
`domain/solo_emprise.go` (NEUF), `domain/equipmentusage/families.go`,
`games/halo_infinite/film/replay/usage_summary_families.go` (2 constantes), `domain/timeseries.go`,
`service/timeseries_service*.go`, `port/` (EmblemURLLoader), `api/wire/registry_pages.go`, contrat.

- [x] L2.1 `domain/equipmentusage` : `EquipmentFamilyGrapple`, `EquipmentFamilyThruster`,
  `EquipmentUnmeasuredLineFamilies()` (copie défensive comme `EquipmentOutcomeFamilies`, l. 89-93) ;
  `replay/usage_summary_families.go:43-44` relit ces constantes. Garde-rail :
  `replay/usage_summary_families_guard_test.go` vérifie que chaque famille de
  `EquipmentUnmeasuredLineFamilies()` est dans `usageCarriedCapacityFamilies` et que le répulseur n'y
  est pas ; mutation : ajouter `repulsor` à la liste → rouge.
- [x] L2.2 `squademprise.Match` : `MapKey`, `MapLabel`, `Outcome` (`canonical.Outcome` en chaîne) ;
  l'Escouade ne les renseigne pas (aucun changement de son bloc : `build_test.go` vert inchangé).
- [x] L2.3 `squademprise.BuildMaps(in Input) []domain.EmpriseMapColumn` (`maps.go`) : une `soiree` par
  carte (réutilise `tallyMatch` / `soiree.add` / `objets.publier`), tri matchs décroissants puis
  libellé ; au-delà de `EmpriseGridMaxMaps + 1` cartes, les 12 premières puis une colonne « Autres
  cartes » (`OtherMaps` = nombre sommé) ; V / D / autres depuis `Match.Outcome`. Tests ROUGES d'abord
  (`maps_test.go`) : tri, repli à 13 / 14 cartes (maquette : 13 cartes = pas de repli), sommes de la
  colonne de repli = somme des cartes repliées, carte sans film (`MatchesFilmed = 0`), carte filmée
  sans niveaux (`MatchesTiers = 0`), objets « qui chez moi » (moi / reste), frags aux armes spéciales
  hors film. Mutations : seuil de repli ±1, tri inversé → rouges.
- [x] L2.4 `squademprise.BuildEquipment(in Input) *domain.EmpriseEquipment` (`equipment.go`), D4.
  Tests ROUGES d'abord (`equipment_test.go`) : servi = posé pour le mur et consommé pour le capteur
  (via `PlayerOutcomeCounts`), moi / reste de mon camp séparés, adversaire exclu, match à camp
  inconnu exclu des deux côtés, grappin / propulseur non mesurés avec MES lâchers, répulseur absent,
  ordre D4, nil sans film. Témoin chiffré : les comptes de la maquette (illustration, mesures §9 :
  mur moi 52 · 0 · 32, reste 146 · 7 · 151 ; capteur moi 6 · 1 · 54 ; grappin 84 lâchés) sur une
  fixture minimale qui les reproduit. Mutation : compter `deployed` pour le capteur → rouge.
- [x] L2.5 `domain/solo_emprise.go` (types §3) ; `TimeseriesPageResponse.Emprise` et
  `PlayerEmblemURL` (`domain/timeseries.go`, après `RangeProfiles`, commentaire de contrat court).
- [x] L2.6 Service : `service/timeseries_service_emprise.go` (NEUF) — `attachEmprise` depuis
  `filteredCanon` : `Match` par ligne canonique (`MapKey` = `Summary.Map.ID`, `MapLabel` =
  `labelPourLocale(Summary.Map, locale)`, `timeseries_service_sections.go:308-316`), `Players` = le
  joueur seul (`squadagg.SquadPlayers(xuid, gamertag, participants, nil)`), `Timeline` vide, lectures
  par L1.4, `Build` + `BuildMaps` + `BuildEquipment` ; une section de durée `emprise` (ADR 0036 I6) ;
  journaux `emprise_*` page `timeseries`. Les lectures du résumé d'usage du périmètre sont faites UNE
  fois (`squadagg.LireUsage`) et partagées avec `BuildSquadFormesBlock` (`SquadFormesQuery.Lectures`,
  `squadagg/squad_formes.go:61-63`) — ADR 0036 I4. Seuil : les dépendances neuves (feuille,
  véhicules, emblèmes, vies, portées) vivent dans une struct `usagesDeps` déclarée dans ce fichier
  et embarquée par UNE ligne dans `TimeseriesService` (`timeseries_service.go:56-123`, fichier à
  424 L) ; leurs `With*` aussi.
- [x] L2.7 Emblème (D14) : `port.EmblemURLLoader` ; `WithEmblemLoader` ; lecture best-effort
  journalisée (Debug si absent).
- [x] L2.8 Câblage `api/wire/registry_pages.go:392-446` : `WithEmprise(duckdb.NewSquadEmpriseRepo(pdb))`
  inconditionnel (feuille de match, tous titres) ; repo d'usage sous `CapFilmUsageSummary` (le même
  `NewSessionUsageRepo`, renommer `WithEquipmentUsage` en `WithUsageSummary(repo, repoRoot)` — la
  suppression du résolveur d'amis se fait en L6) ; véhicules sous `CapFilmVehicleUsage` ; chargeur
  d'emblèmes comme `TeammatesCtx` (`registry_pages_home.go`). Garde-rail de câblage
  `registry_pages_timeseries_wiring_test.go` (NEUF, patron `registry_pages_home_teammates_wiring_test.go:80-170`) :
  `WithEmprise` inconditionnel, véhicules sous leur seule porte ; mutation : mettre `WithEmprise` sous
  condition → rouge.
- [x] L2.9 Tests service (mocks de port, `timeseries_service_emprise_test.go`) : périmètre = les
  matchs filtrés ; Halo 5 (repo d'usage nil → `film_unavailable = film_unsupported`, seule la feuille) ;
  `ErrCapabilityNotSupported` d'une source → source absente avec raison ; lecture en échec →
  `*_load_failed`, jamais d'erreur de page ; scope vide → `emprise` nil ; une seule lecture du résumé
  d'usage pour l'Emprise et les formes (compteur de mock).
- [x] L2.10 Contrat régénéré (openapi + `generated.ts`), diff additif ; garde
  `contract-surface.guard.test.ts` verte (ajouts tolérés).
- Gate : gate Go + contrat ; `no_title_package_in_analysis_test.go`,
  `no_analysis_type_in_http_body_test.go`, `no_slug_comparison_test.go` rejoués nommément.

Journal L2 (2026-10-06, exécuteur, `feat/ts-usages-emprise`) :
- **L2.1** `domain/equipmentusage` : `EquipmentFamilyGrapple`, `EquipmentFamilyThruster`, `EquipmentUnmeasuredLineFamilies()` ; `replay/usage_summary_families.go` relit les deux constantes (valeur identique, aucune révision). Garde-rail `TestFamillesNonMesureesSontDesCapacitesPortees` (`usage_summary_families_guard_test.go`).
- **L2.2** `squademprise.Match` : `MapKey`, `MapLabel`, `Outcome` (`canonical.Outcome`) ; l'Escouade ne les renseigne pas (`build_test.go` vert inchangé).
- **L2.3** `squademprise.BuildMaps` (`maps.go`) : une `soiree` par carte, tri matchs décroissants / libellé / clé, repli au-delà de `domain.EmpriseGridMaxMaps + 1` ; tests `maps_test.go` (tri et sommes, repli à 13 / 14 / 20 cartes, carte sans film).
- **L2.4** `squademprise.BuildEquipment` (`equipment.go`), D4 ; tests `equipment_test.go` avec le témoin chiffré de la maquette (mur moi 23 pris, 52 · 0 · 32, reste 146 · 7 · 151 ; capteur moi 12 pris, 6 · 1 · 54, reste 16 · 4 · 183 ; grappin 84 et propulseur 65 lâchés) et les exclusions (adversaire, camp inconnu, poses de capteur, répulseur).
- **L2.5** `domain/solo_emprise.go` (`SoloEmpriseBlock` embarquant `SquadEmpriseBlock` — schéma OpenAPI vérifié aplati —, `EmpriseMapColumn`, `EmpriseEquipment*`, `EmpriseGridMaxMaps`) ; `TimeseriesPageResponse.Emprise`, `.PlayerEmblemURL`.
- **L2.6** `service/timeseries_service_emprise.go` : `usagesDeps` embarqué par une ligne dans `TimeseriesService`, `attachEmprise` (section de durée `emprise`, journaux `emprise*` page `timeseries`), `timeseriesEmpriseMatches` (carte et résultat depuis le canonique). Les trois lectures du résumé d'usage se font UNE fois (`lireUsageDuScope`, section `usage_summary`) et nourrissent le bloc d'usage, les formes et l'Emprise (avant ce lot : le bloc d'usage et les formes lisaient chacun de leur côté).
- **L2.7** `port.EmblemURLLoader`, `WithEmblemLoader`, `attachEmblem` (Debug sans chargeur).
- **L2.8** Câblage : `registry_pages.go` (déjà à 620 lignes, au-delà du seuil) n'est PAS agrandi — taille avant / après : 620 / 620 — : la factory appelle `r.cablerUsagesTimeseries(svc, pdb)`, nouveau fichier `api/wire/registry_pages_timeseries.go` (feuille inconditionnelle, emblème inconditionnel, véhicules sous `CapFilmVehicleUsage`). Le lecteur d'appels du test de câblage de l'Escouade est généralisé (`appelsDansFactory`, une seule copie). Nouveau `registry_pages_timeseries_wiring_test.go`. Renommage `WithEquipmentUsage` → `WithUsageSummary` : fait en L6.2, quand le résolveur d'amis disparaît (la signature ne change qu'une fois) — dépendance de plan, pas un report.
- **L2.9** `timeseries_service_emprise_test.go` : fenêtre, un seul joueur, ni habitude ni placement, grille et équipement ; sans film (film_unsupported, feuille seule) ; dégradations nommées (feuille non supportée / en échec, film en échec) ; fenêtre vide ; une lecture du résumé d'usage pour trois blocs ; emblème.
- **L2.10** Contrat : `openapi.yaml` +179 lignes, 0 retrait ; `generated.ts` +67, 0 retrait ; `check-generated-types-fresh` OK ; `contract-surface.guard.test.ts` vert SANS régénérer le snapshot (snapshot non modifié).
- **Rouge avant vert** : pour `BuildMaps`, `BuildEquipment` et `attachEmprise`, le code a été écrit avant les tests ; le rouge a été obtenu en rejouant les tests contre un bouchon (corps remplacé par `return nil` / bloc non posé) — 3, 2 et 3 tests rouges respectivement —, puis vert. Écart de méthode consigné.
- **Mutations** (toutes ROUGES, restauration vérifiée) : répulseur ajouté aux familles non mesurées ; seuil de repli −1 et +1 ; tri des cartes inversé ; DNF compté en défaite ; capteur lu sur les poses (`equipmentUsedOf`) ; adversaire compté dans le reste du camp ; match à camp inconnu compté ; lecture du résumé d'usage non partagée (2 lectures au lieu d'1) ; résultat non transmis ; coéquipiers sélectionnés sur la page solo ; câblage : `WithEmprise` déplacée sous `CapFilmUsageSummary`, `WithEquipmentUsage` sortie de sa porte (`if true`), `cablerUsagesTimeseries` mise sous condition.
- **Gate** : `go build ./...` 0 ; `go vet` des 8 paquets touchés 0 ; `gofmt -l` muet (un fichier reformaté, fin de ligne) ; `go test -count=1` du module en lots couvrant tout `go list ./...` : 67 + 39 + 28 + 13 + 48 ok, 0 FAIL ; après le déplacement du câblage : build, vet, `./internal/api/...` 5 ok ; `make go-api-lint` 0 issues (deux fois) ; `openapi-gen -check` à jour ; garde-rails rejoués nommément : `TestAucunTypeAnalysisEnCorpsHuma`, `TestNoNewSlugComparison`, `TestAnalysisImporteAucunPaquetDeTitre`, `TestOpenAPIYAMLIsUpToDate` PASS. Web (premier passage) : `npm ci` (node_modules réel, 508 paquets) ; `npm run generate-types` ; `npx tsc -b --force` 0 ; vitest `src/lib/api` 5 fichiers / 36 tests verts. `-tags=integration` non requis (aucun paquet `platform/duckdb`, `sync`, `persist`, `migration` modifié).
- Seuils : fichiers neufs ≤ 188 L ; `timeseries_service.go` 458 L (+2) ; `domain/timeseries.go` 385 L ; plus longue fonction neuve `BuildEquipment` (~35 L) ; `attachEmprise` 5 paramètres (ctx compris).

### L3 — Go : « Mes vies : près d'un coéquipier ou seul » · moyen

- [ ] L3.1 `analysis/coordination/vies_pres_ou_seul.go` (NEUF) : types de lecture (`ViesLues` :
  vies, morts situées, frags avec camps, variantes) et `ViesPresOuSeul(lues, rayonParMatch)` → domaine
  (D5). `accompagnee` (`isolation.go:72-74`) délègue à une comparaison commune `aPortee(d *float64,
  rayon float64) bool` utilisée par les deux. Tests ROUGES d'abord : fenêtre `[début, début suivant)`
  (frag posthume rattaché à la vie qui finit), borne inclusive (d = portée → près), d NULL → écartée,
  pas de contexte dans la fenêtre → écartée, match sans portée → écartée (et compté), vie
  `film_end` / `cut` ignorée, trahison et camp inconnu exclus, frags par vie. Mutations : `<` au lieu
  de `≤`, fenêtre bornée par `end_ms` → rouges. Témoin : une fixture qui reproduit 721 / 152 vies et
  604 / 168 frags n'est PAS exigée (base non lisible ici) ; un témoin à 6 vies chiffrées à la main.
- [ ] L3.2 `port/timeseries_lives.go` : `SoloLivesRepository.LoadLivesNearTeammate(ctx, matchIDs
  []string, xuid string) (coordination.ViesLues, error)`.
- [ ] L3.3 `platform/duckdb/solo_lives_repo.go` (NEUF) : trois lectures sur `match_lives_latest`,
  `match_death_context_latest`, `match_kill_events_latest` (+ `match_participants`, `match_registry`
  tables ordinaires), liste des matchs liée en constante sur le `match_id` de CHAQUE vue
  (`clauseListeMatchs`, ADR 0036 I2), joueur filtré après la fenêtre, jamais `v_gamertag_lookup` ;
  table absente → `games.ErrCapabilityNotSupported` (patron `squad_life_placement_repo.go:73-78`).
  Test `:memory:` migré (patron `squad_life_placement_repo_test.go`) : dernière passe entière par
  match, matchs et joueur demandés seulement, NULL conservés, `exigerFenetresBornees` sur les trois
  vues ; mutation : lier la liste par sous-requête → rouge.
- [ ] L3.4 Service `service/timeseries_service_lives.go` (NEUF) : `attachLives` — repo nil →
  Debug « capability absente » ; lecture ; `mappings.PorteesDuRadarParMatch` (L1.1) ; calcul ;
  journal Info du bilan (vies, écartées, frags écartés) ; section de durée `lives`. Tests mocks :
  capability absente, échec de lecture (bloc absent + ErrorContext), table des portées vide (toutes
  écartées et comptées, bloc publié avec Near/Alone à zéro ET `ExcludedNoRadar` > 0 — le web décide
  de l'afficher, voir L5.6).
- [ ] L3.5 Câblage sous `CapFilmKillPositions` (même porte que `WithLifePlacement`,
  `registry_pages_home.go:272-274`) + `WithRadarRange(r.radarRangeFor(pdb))` ; ajout au garde-rail de
  câblage L2.8.
- [ ] L3.6 `TimeseriesPageResponse.LivesNearTeammate` (`json:"lives_near_teammate,omitempty"`),
  contrat régénéré ; ADR 0036 (EN) : ajouter le nouveau test à la liste I2
  (`docs/adr/0036-page-reads-are-scoped.md:155-162` et tableau l. ~421).
- Gate : gate Go + `go test -tags=integration -p 1 ./internal/platform/duckdb/...` + contrat.

### L4 — Web : briques de l'Emprise paramétrables (Escouade inchangée) · moyen

- [ ] L4.1 `squad/emprise/emprise.logic.ts:89-104,149,359` : `buildResourceFil` et `buildMatchGrid`
  prennent un index `Map<string, EmpriseMatchInfo>` ; `empriseMatchIndexFromHistory(history)` pour
  l'Escouade ; appelants (`useEmpriseModels.ts:40,42`, `empriseContent.ts:69`) migrés ;
  `emprise.logic.test.ts` vert inchangé (mêmes valeurs).
- [ ] L4.2 `empriseCharts.ts` : mode d'axe `period` de `buildResourceFilOption` (D12) ;
  `ResourceFilCard` reçoit `axis` (défaut `match`) ; tests `empriseCharts.test.ts` : labels de mois,
  légende « n matchs, dont m filmés », pas d'encoche, rayons réduits au-delà de 120 matchs ;
  mutation : étiquette sur chaque match → rouge.
- [ ] L4.3 Extraire de `ResourceMatchGridCard.tsx:62-390` la table (`SectionRows`, `GridLine`,
  `Cell`, `cellTip`, `ResourceDot`, `SummaryLabel`) dans `squad/emprise/ResourceGridTable.tsx`,
  colonnes génériques `{ key, head: ReactNode, tipHead: string }` ; `ResourceMatchGridCard` garde
  `MatchHead` ; `SquadEmprisePage.test.tsx` vert inchangé.
- RÈGLE DU LOT : uniquement des refactorisations dont chaque export a un lecteur dans le même lot
  (knip 0 / 0 / 0 au gate) ; les composants neufs naissent en L5 avec leur lecteur.
- Gate : gate web (aucun Go) ; `SquadEmprisePage.test.tsx`, `SquadContributionsPage.test.tsx`,
  `SquadObjectiveSection.test.tsx` rejoués nommément.

### L5 — Web : l'onglet Usages reconstruit · lourd

Périmètre : `features/timeseries/TimeseriesPage.usages.tsx`, `features/timeseries/usages/*` (NEUF),
`squad/objectif/` (fiche solo), `WeaponRangeSection.tsx`, `TimeseriesCoordinationSection.tsx`,
`TimeseriesPage.progression.tsx` (rien à changer si la section garde sa signature), tests, puis les
SUPPRESSIONS WEB devenues mortes (knip 0 / 0 / 0 au gate de ce lot l'exige). Aucune requête neuve,
aucune clé de requête neuve (`lib/query/keys.ts` non touché) : tout arrive avec
`useTimeseriesPage`. Libellés de résultat et de dominance : `squad/emprise/useOutcomeLabels.ts` et
`lib/narrative/dominance.ts` (existants).

- [ ] L5.0a `ObjectiveSoloSheetCard.tsx` (NEUF, `squad/objectif/`) + `buildSoloObjectiveSheet` dans
  `objectif.logic.ts` (une fiche : MA valeur, part = ma valeur / somme de la ligne sur les fiches du
  camp de `buildObjectiveSheets`, familles côte à côte, rôle dominant, pied Prendre / Défendre /
  Tenir, zéro atténué) ; tests ROUGES d'abord (part, zéro, rôle dominant, emblème / initiale) ;
  mutation : part sur le max de la ligne (règle de l'Escouade) → rouge.
- [ ] L5.0b `features/timeseries/usages/usagesText.ts` (D10) : `EMPRISE_TEXT_SOLO`,
  `OBJECTIF_TEXT_SOLO`, textes des cartes neuves ; test : FR = maquette pour chaque titre et ⓘ
  (chaînes copiées), parité FR / EN par le typage, aucun « Notre camp » / « Our side ».
- [ ] L5.1 `usages/usages.logic.ts` (+ test) : index des matchs depuis `match_rows`
  (date, carte `map_name_fr || map_name` comme `matchLabels.ts:18`, résultat
  `outcomeCodeToValue`), prédicat `usagesSections(data, caps)` (un seul prédicat pour la page et
  l'état vide : portée, bilan, carte, mes prises, prendre, vies, objectif, équipement ; réutilise
  `gridHasFilmRows` / `sheetsHaveLines` de `empriseContent.ts:20-27`), modèles de « Mes prises »
  (depuis `buildPickupSheets`), de la grille par carte (depuis `emprise.maps`), de l'équipement et des
  vies. Tests ROUGES d'abord, une mutation par règle.
- [ ] L5.2 `TimeseriesPage.usages.tsx` : ordre §3, intertitres de la maquette (« Portée des
  engagements », « Bilan du périmètre », « Carte par carte », « Mes prises », « Prendre, et s'en
  servir », « Près d'un coéquipier ou seul », « Objectif », « Équipement »), un bloc sans donnée se
  retire intertitre compris, état vide inchangé ; aucune logique dans le composant.
- [ ] L5.3 `ResourceMapGridCard.tsx` : en-tête nom, « n cartes · » pour la colonne de repli,
  « n matchs », « x V · y D (· z A) » en couleurs `outcome-*` ; cases « sans film » (hachure),
  « non classé », « — » ; infobulle « Chez moi : <gamertag> n, reste du camp m » ; râteliers repliés ;
  ligne des frags aux armes spéciales (feuille).
- [ ] L5.4 `MinePickupsCard.tsx` (maquette `renderMine`) : groupes par ressource, objets pris par mon
  camp triés par volume, barre à l'échelle du plus gros, segments moi (`squad-player-1`) / reste
  (`team-rest`) avec comptes dedans (S2), « moi n · camp m » au bout, râteliers repliés, ligne
  « Bonus perdus » des deux camps en pastilles d'équipe ; sans prise : la carte se retire (D13).
- [ ] L5.5 `EquipmentOutcomesCard.tsx` (maquette `renderEquip`) : une ligne par famille D4,
  segments servi / gardé / lâché (`divergent-pos` / `divergent-neutral` / `divergent-neg`) avec
  comptes, sous-libellé « n objets, dont m pris sur la carte », barre fine du reste de mon camp et
  sa ligne de parts, ligne « Non mesuré : ni prise ni usage publiés pour cette famille » + « n lâchés »
  pour grappin et propulseur, axe 0-100 %.
- [ ] L5.6 `LivesNearTeammateCard.tsx` (maquette l. 994-1026, encart « Maquette. » non porté) :
  barre épaisse vies près (`squad-player-1`) / seul (`extreme`), barre fine frags, ligne « frags :
  n · p % · x par vie … y par vie · m » ; ⓘ avec le compte des vies écartées (les deux causes) ;
  carte retirée si Near + Alone = 0.
- [ ] L5.7 Objectif : `ObjectiveBalanceCard` (texte solo) puis `ObjectiveSoloSheetCard` (L5.0a)
  alimentés par `formes_retenues` et `player_emblem_url` ; section retirée sans match à objectif
  (`objectiveMatches`).
- [ ] L5.8 `WeaponRangeSection.tsx` : `ElevationCard` retirée, props `elevation` / `matchRows`
  retirées, « Portée par arme » demi-largeur seule (D11) ; tests adaptés.
- [ ] L5.9 `TimeseriesCoordinationSection.tsx` : `CarteRiposte` retirée, « Appui reçu » seule
  (D11) ; tests adaptés (la carte Riposte n'est plus montée).
- [ ] L5.10 Tests de page : `TimeseriesPage.sections.test.tsx` réécrit (ordre des blocs, intertitres,
  retrait par bloc, état vide, anglais, Halo 5 sans film : seule la barre épaisse des armes
  spéciales si la feuille la porte) ; une fixture `usages/usages.fixtures.ts` tirée des chiffres
  d'illustration de la maquette (bilan 111 / 87 bonus, 229 / 231 armes spéciales ; vies 1 558 / 301).
- [ ] L5.11 Rejouer CHAQUE preuve grep de §4.A-D avant de supprimer ; écart → Découvertes, arrêt
  propre si un lecteur inattendu existe.
- [ ] L5.12 Web §4.A : `EquipmentUsageSection` et dépendances propres, tests, clés i18n orphelines.
- [ ] L5.13 Web §4.B : cartes et formes, survivants réduits, `aggregateColumns` réduit.
- [ ] L5.14 Web §4.C, §4.D, §4.G (manifests régénérés par `node apps/web/scripts/build_i18n_manifests.mjs`).
- [ ] L5.15 Ratchets : knip 0 / 0 / 0 ; imports croisés ≤ 7 et aucune dérogation morte (la paire
  `timeseries=>squad` reste servie) ; si un plafond baisse, l'abaisser.
- Gate : gate web ; preuves §4.A-D rejouées → 0 (côté web).

### L6 — Suppressions Go (D8, D9) et contrat · moyen

Le web ne lit plus `equipment_usage` ni `elevation` depuis L5.

- [ ] L6.1 Rejouer les preuves §4.E et la partie Go de §4.C (producteurs et lecteurs Go).
- [ ] L6.2 Go D8 : chaîne `equipment_usage` (+ `WithUsageSummary` sans résolveur d'amis, wire
  `registry_pages.go:428` sans `friendGamertagsResolver`), tests supprimés avec leur code
  (`service/equipment_usage_block_test.go`, `sessionusage/usage_overview_test.go`, cas
  `timeseries_service_test.go` qui les citent).
- [ ] L6.3 Go D9 : chaîne d'élévation ; `weapon_range_section_test.go:494` et
  `elevation_cloud_section_test.go` suivent.
- [ ] L6.4 Contrat régénéré ; snapshot `contract-surface` régénéré par la procédure, disparitions
  listées ; fixture `test/handlers.ts:242` retirée ; `types.ts:2327-2330` retiré.
- Gate : gate Go + contrat + gate web ; preuves §4.A-E rejouées → 0.

### L7 — Contrat `formes_retenues` réduit à l'objectif (D7) · moyen, le plus risqué

- [ ] L7.1 Rejouer §4.F (lecteurs web des champs après L6).
- [ ] L7.2 Go : champs D7 retirés de `domain/squad_formes.go:59-159` ; `squadformes.Build` ne
  publie plus que les matchs à objectif et les champs gardés (le compte `MatchesMeasured` reste,
  calculé comme aujourd'hui) ; `FilmPads`, `WeaponPad`, `WeaponInfo.Class/Role` s'ils n'ont plus de
  lecteur (vérifier `squademprise/input.go:108-109` qui lit `squadformes.WeaponInfo` :
  `Label`/`WeaponKey` restent), `WeaponClassOf`, `buildWeapons`, `WallFamilyKey`
  (`squadagg/squad_formes.go:108-111`), `LoadUsageFilmPads` (port + DuckDB + tests).
- [ ] L7.3 Tests Go adaptés : `squadformes/formes_test.go`, `squadagg/squad_formes_prises_nettes_test.go`,
  `teammates_service_usage_test.go`, `timeseries_service_equipes_test.go` ; l'objectif publié est
  identique avant / après (test de non-régression sur la fixture existante, écrit AVANT la coupe).
- [ ] L7.4 Contrat régénéré ; web : types TS suivent, fixtures `objectif.fixtures.ts` /
  `objectivesOptional.test.ts` réduites ; `SquadObjectiveSection` et les cartes d'objectif des deux
  pages inchangées à l'écran.
- Gate : gate Go + contrat + gate web.

### L8 — Clôture · rapide

- [ ] L8.1 Docs : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md` (bloc `[7.5.0]`, corriger les phrases
  contredites — EN l. 40, 54, 71 ; FR l. 40, 54, 71 — et ajouter l'entrée) ; `docs/RELEASE_NOTES.md`
  + `docs/FR/RELEASE_NOTES.md` (bloc 7.5 : corriger EN l. 32, 49, 55 / FR l. 32, 49, 55 — riposte sur
  les Séries temporelles, « formes retenues » solo, contenu de l'onglet Usages — et ajouter
  l'entrée) ; lignes re-vérifiées au moment d'écrire.
- [ ] L8.2 `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` §4 (l. 256-345, lecteurs des Séries
  temporelles l. 299-307) : `equipment_usage` supprimé, Emprise solo et carte Équipement (D4) ;
  ligne du tableau l. 482.
- [ ] L8.3 ADR 0036 (fait en L3.6, vérifié ici) ; `docs/adr/0034` non concerné.
- [ ] L8.4 Statut de chaque item du plan ; §8 Découvertes relues ; entrée finale du journal.
- [ ] L8.5 Revue adversariale du diff cumulé : à demander au SUPERVISEUR (l'exécuteur n'a pas de
  sous-agent) — lots à risque : L2 (agrégats), L3 (lecture bornée), L7 (contrat).
- Gate : gate Go complet + gate web complet + contrat, rejoués après les docs.

## 7. Reprise de session

Relire le skill `plan-execution`, puis ce fichier (cases, journaux de lot), puis les dernières
entrées de `.ai/thought_log.md` du worktree et `git -C <worktree> log --oneline -10`. Reprendre à la
première case non statuée du lot courant. Les décisions du §1 sont fermes une fois le « go » donné.

## 7 bis. Relecture plan-review (2026-10-05, phase 1)

Grille `.claude/skills/plan-review/SKILL.md`, passée sur ce fichier :
- §1 structure : objectif et critère (§0), lots ordonnés du refactor sans effet (L1) au contrat
  le plus risqué (L7), effort par lot, branche nommée, bloqueur documenté (D7 différable) — OK.
- §2 couches Go : calculs dans `analysis/` (squademprise, coordination), types dans `domain/`,
  orchestration dans `service/`, ports neufs (`SoloLivesRepository`, `EmblemURLLoader`), aucun SQL
  hors `platform/duckdb`, aucun handler modifié (la réponse de page existante porte les blocs) — OK.
- §3 multi-titre : capabilities `film.usage_summary`, `film.vehicle_usage`, `film.kill_positions`,
  feuille de match inconditionnelle ; `ErrCapabilityNotSupported` testé (L2.9, L3.4) ; aucun
  `slug ==` ; pas de nouveau champ de stats ni d'asset — OK.
- §4 adapters : lectures par repos de port (patron déjà en place pour ces blocs, pas de
  `TitleDataAdapter`) ; types de domaine, pas de type de titre — OK, écart assumé et conforme à
  l'existant (Emprise de l'Escouade).
- §5 tests : purs (L1.1, L2.3, L2.4, L3.1), service avec mocks (L2.9, L3.4), DuckDB `:memory:` avec
  fenêtres bornées (L3.3), câblage (L2.8), web (L4, L5) ; mutation par règle — OK. Handlers : aucun
  changement, aucun test httptest ajouté.
- §6 logs : `slog.*Context` à chaque dégradation (L1.4, L2.6, L3.4) — OK.
- §7 front : aucune route, aucune clé de requête ; chaînes FR + EN `Record<Locale>` (D10), libellés
  de carte venus du Go, de résultat par `useOutcomeLabels` ; jetons seulement — OK.
- §8 livraison : critères par lot (gates), journal par lot, aucune dépendance externe — OK.
- §9 exécutabilité : périmètres fermés (listes, preuves grep, knip comme juge des morts), gates à
  commandes exactes, statuts et règle « aucune case vide », ordre strict, Découvertes, reprise,
  renvoi au skill — OK.
Défaut trouvé et CORRIGÉ à la relecture : la première version créait en L4 des exports sans lecteur
(fiche solo, textes) et laissait des fichiers web morts jusqu'à L6 — le gate knip 0 / 0 / 0 aurait
rougi en L4 et en L5. Les composants neufs naissent désormais en L5 avec leur lecteur, et les
suppressions web sont dans L5 (L5.11-L5.15) ; L6 ne garde que le Go.

## 8. Découvertes (à consigner ici, pas à traiter)

- (phase 1) `domain.SquadEmpriseBlock` porte « Squad » dans son nom alors qu'il sert aussi le solo
  (D2) : renommage non fait (hors périmètre, bruit de contrat).
- (phase 1) Le bloc `coordination` des Séries temporelles calcule encore la riposte que plus aucune
  carte de la page n'affiche ; DTO partagé avec Sessions, donc conservé.
- (phase 1) `match_life_placement` et `match_vehicle_takes` vides dans la copie de base du
  2026-10-05 18 h 26 alors que le journal note leur rattrapage (complément 2 du journal) : cause non
  instruite.
</content>
</invoke>
- (L2) Les Séries temporelles lisent les participants de la fenêtre deux fois : `lireEquipesDuScope` (`timeseries_service_sections.go`, section `participants`) et `squadagg.LireUsage` (résumé d'usage partagé depuis L2). Les deux portent sur la même fenêtre ; les fusionner toucherait la courbe d'équipe et la coordination (hors périmètre).
