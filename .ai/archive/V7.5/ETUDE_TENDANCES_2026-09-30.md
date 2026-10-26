# Étude — pages « Tendances » solo et escouade (2026-09-30)

Statut : **étude du concept, pas d'implémentation** (consigne utilisateur). Décisions prises
(§5), maquette sur données réelles et trois décisions ouvertes (§6.6). Branche
`claude/trend-analysis-halo-stats-8771a8`.

Demande : s'inspirer de « Trend Analysis — 7/30/90/365 day performance trends » de
`github.com/gnossdrawkcab/halo-squad-stats`, une page solo et une page escouade (escouade
choisie par son nom enregistré), indépendantes ; vérifier la redondance avec les Séries
temporelles et l'Escouade ; dire ce que nos données ajoutent pour joueurs compétitifs et
occasionnels.

## 1. Ce que fait le dépôt d'exemple (lu sur pièces : `src/webapp.py` route `/trends`, `templates/trends.html`)

- Flask + pandas + PostgreSQL. Une page `/trends`, sans distinction solo / escouade : elle
  superpose TOUS les joueurs suivis.
- Fenêtre unique choisie parmi 7 / 30 / 90 / 180 / 365 j / tout (défaut 90), comptée depuis
  maintenant (`apply_trend_range`). Tout est ensuite **moyenné par jour et par joueur**
  (`build_trend_data`), avec une moyenne glissante de 7 j côté navigateur.
- 11 courbes quasi identiques : CSR (dernier CSR du jour), taux de victoire **cumulé**, KDA,
  score objectif, dégâts/min, écart de dégâts, précision, frags/match, morts/match, meilleure
  série, durée de match.
- Pastilles de note par courbe : classement des joueurs entre eux sur la DERNIÈRE valeur
  journalière (`grades.py:build_trend_grade_rows`).
- Corrélation de Pearson stat ↔ victoire (global et par joueur), heatmap jour × heure,
  « moments » (meilleurs et pires jours).
- Variation de CSR sur 7 / 30 / 90 j ailleurs (`compute_csr_window_delta` : dernier − premier).
- Tendances filtrées sur Ranked Arena uniquement.

Aucune comparaison entre périodes : on change de fenêtre et on regarde des courbes.

## 2. Redondance : ce qui existe déjà chez nous

| Question du joueur | Où c'est déjà traité | Tendances ? |
|---|---|---|
| Détail match par match d'un périmètre (axe = n° de match) | Séries temporelles (4 onglets, `features/timeseries/`) | non |
| Bilan d'un périmètre, heatmap jour × heure avec taux de victoire | Synthèse (`synthesis_service.go:208`) | lien seulement |
| Ce périmètre comparé à mon habitude | Explorer, bandeau « vs habituel » (`match_history_service_briefing.go:232`) | non |
| Où j'en suis face aux cibles de mon palier, pente LOWESS 30 j | Ascension, section Performance (`progression/profile/`) | cousin le plus proche |
| Mes 5 derniers matchs comparés aux 5 d'avant | Accueil (`home_canonical_kpis.go:144`) | non |
| CSR comparé à la saison précédente | Carrière (`CareerRankingBlock`) | non |
| **Suis-je en progrès, et à quelle échelle de temps ?** | **nulle part** | **oui** |
| Escouade : une session ou une période en profondeur | Escouade (Synergies, Contributions, Dynamique, Emprise) | non |
| **Escouade : évolution de la composition dans le temps** | frise des sessions (Synergies, historique complet) ; série `timeseries` calculée par le serveur mais **jamais affichée** | **oui** |

Conclusion du challenge :
- **Copier la page d'exemple serait redondant.** Choisir 7/30/90 j dans la pastille Période
  des Séries temporelles donne déjà ces courbes, match par match et plus détaillées. Il ne
  manque que le préréglage 1 an (`PERIOD_PRESETS`, `_hooks.ts:33-38`, et
  `filters_options.go:30-35` côté compteurs) et un axe par date.
- **Une page Tendances se justifie seulement si elle répond à la question que personne ne
  traite :** plusieurs horizons côte à côte, comptés depuis aujourd'hui, chacun comparé à une
  référence, avec un avertissement quand trop peu de matchs. Toutes nos pages actuelles
  regardent UN périmètre à la fois.
- Risque de doublon à maîtriser : Ascension (pente LOWESS 30 j, `TrendBadge`). Une seule
  définition de « en progrès » doit exister ; Tendances réutilise celle d'Ascension ou la
  remplace, pas une troisième.

## 3. Ce que nos données ajoutent (inventaire Go, à réutiliser)

Pour tout le monde (formules ADR 0006, `analysis/indicators.go`) : taux de victoire, KDA
(formule agrégée, jamais le quotient), KDR, précision, frags/match, morts/match, score de
performance 0-100 (déjà relatif à l'historique du joueur, 50 = habituel), durée de vie, CSR et
LUSR (`_latest`), variation de rang sur un périmètre (`ComputeKPIStats` → `RankDelta`).

Joueurs occasionnels :
- une phrase de verdict par horizon, en français courant (« en progrès sur 30 jours, stable
  sur l'année ») ;
- rythme : matchs par semaine, jours joués ;
- records battus sur 30 / 90 j (`progression/records`, fenêtres `30d | 90d | all_time`) ;
- meilleurs créneaux : lien vers la heatmap de la Synthèse, pas de copie.

Joueurs compétitifs (ce que le dépôt d'exemple n'a pas) :
- **progression à difficulté égale** : frags et morts réels comparés aux attendus
  (`expected_fda.go`, capacité `expected_stats`), en regard du MMR du lobby. Un KDA qui baisse
  parce que le lobby monte n'est pas une régression ; c'est le signal le plus utile pour un
  joueur classé ;
- dégâts nets par match (infligés − subis). Pas de valeur par minute : tout se lit par match
  (décision utilisateur du 13/09, « cadence c'est par match, pas par minutes ») ;
- conversion offensive et résistance défensive (`combat_yield.go`) ;
- premier frag, intensité par phase de match (début contre fin) ;
- trajectoire CSR / LUSR et variation par horizon ;
- filtre playlist / mode (le dépôt d'exemple ne regarde que Ranked Arena : mélanger BTB et
  classé fausse toute tendance).

Escouade :
- taux de victoire de la composition, nombre de sessions et de matchs (compte unique ADR 0033
  `composition_sessions[].match_count`), MMR d'équipe, score escouade (`squad_score.go`) ;
- forme de chaque membre (score de performance par horizon) ;
- coordination (riposte, appui) dans le temps ;
- séries déjà calculées et inutilisées : `ComputeSquadTimeseries`
  (`teammates_service_sections.go:82`) et Form Score LOWESS (`squad_service_v2_timeline.go:77`).

À ne PAS reprendre du dépôt d'exemple : courbes journalières superposées (illisibles sur un an,
un jour à 2 matchs = bruit), taux de victoire cumulé (converge et masque le récent), notes qui
classent les amis entre eux, 11 graphiques identiques.

Piste à trancher : corrélation stat ↔ victoire (« ce qui fait gagner »). Rien d'équivalent
chez nous (seuls des nuages de points sans coefficient, `timeseries_service_correlation.go`).
Intéressant pour les compétitifs, mais corrélation n'est pas cause, et en jeu d'équipe elle
est très confondue.

## 4. Briques à réutiliser (rien à réinventer)

Go :
- agrégat d'un périmètre : `analysis.ComputeKPIStats` (`kpi_stats.go:27`) appelé une fois par
  horizon ;
- courbe par date : `temporal.BucketByGranularity` + `ResolveAdaptive` + `LowessSmooth`.
  Limite : `temporal.Period` ne connaît que `all | 2y | 1y | 1m | 1w` (1m / 1y calendaires) ;
  pas de 90 j. À étendre, ou passer des bornes de dates ;
- comparaison à une référence : modèle `buildBriefingBaseline` (Explorer) et
  `breakdown.CompareToHistorical` ;
- escouade : résolution de la composition du service teammates (ADR 0033, composition
  stricte), `ComputeSquadTimeseries`, Form Score ;
- escouades enregistrées : `GET /players/{slug}/squads?user_id=` (module Prestige, actif par
  défaut, `config.go:145-148`) ; champs `Squad{id, name}` + membres par xuid. Pas de
  `GET /squads/{id}` : la liste suffit.
- contraintes ADR 0036 : un seul chargement par requête (I4), lecture `_latest` bornée (I2),
  cache invalidé au sync (I3), sections de durée déclarées (I6). Un chargement borné (365 j, ou
  730 j si la référence est la période précédente) sert tous les horizons.

Web :
- `DeltaCard` (`components/ui/delta-card.tsx`, valeur + variation signée colorée,
  `lowerIsBetter`) : **aucun consommateur aujourd'hui**, fait exactement la cellule voulue ;
- `MetricWithTrend` ▲▼ (`components/ui/metric-trend.tsx`), `KpiCard`, `StatCard`,
  `SectionCard`, `ChartCard` + wrappers du README charts, `EmptyStateNotice` ;
- échelles `perfScale`, `skillDeltaScale` ; `lib/baseline.ts` (`formatSignedPoints`) ;
- filtre local playlist / mode : `features/_shared/useLocalFilterBar.tsx` (modèle Synthèse,
  exclue de la barre de période globale) ;
- navigation : section Stats de `navL1Sections.tsx:93-103` ; une route hors layout Escouade
  (suffixe `_`, comme `career_.tsx`) évite la lourde requête `POST /pages/teammates` du
  layout.

## 5. Décisions utilisateur (2026-09-30)

1. **Principe** : comparer les horizons (7 / 30 / 90 / 365 j côte à côte). Pas de copie du
   dépôt d'exemple.
2. **Référence d'une variation** : la période précédente de même durée (7 j comparés aux 7 j
   d'avant, etc.).
3. **Corrélation stat ↔ victoire** : dès le départ.
4. **Navigation** : question retournée par l'utilisateur — « de nouveaux onglets seraient
   dépendants des filtres pris en L2, ça casserait le principe des tendances ». Réponse
   (§6.1) : oui, et c'est structurel ; section dédiée proposée puis **validée** (onglets Solo /
   Escouade, seul filtre playlist / mode).
5. **Pas d'implémentation** : on étudie le concept et sa mise en place.

## 6. Mise en place (concept, pas de code)

### 6.1 Navigation : une section « Tendances » à part

Vérifié sur pièces : toute route sous `/stats/` hérite de la barre de filtres solo
(`routeShowsSoloFilters`, `components/shell/shellNavigation.ts:55-59`), et un enfant du layout
Escouade hérite de la barre escouade, du choix de composition et de la requête
`POST /pages/teammates`. La Synthèse est exclue à la main (`SYNTHESIS_RE`) et porte sa propre
barre locale : c'est faisable techniquement, mais un onglet au milieu de voisins pilotés par la
période fait croire que changer la période le change.

Proposition : une section L1 « Tendances » (`navL1Sections.tsx`) avec deux onglets, Solo et
Escouade, sur des routes `/tendances/solo` et `/tendances/escouade` que `STATS_RE` ne
reconnaît pas : l'indépendance est structurelle, pas une exception à maintenir.

Filtres permis (orthogonaux au temps, ils ne cassent pas le principe) :
- solo : playlist / mode seulement (classé seul, par exemple). Indispensable : mélanger BTB et
  classé fausse toute tendance. Ni période, ni sessions, ni saison ;
- escouade : l'escouade enregistrée (par son nom), la composition stricte, playlist / mode.

### 6.2 Ce que calcule le serveur (une requête par page)

- **Fenêtres** : pour chaque horizon H ∈ {7, 30, 90, 365} j, la fenêtre courante
  `[aujourd'hui − H, aujourd'hui]` et la précédente `[aujourd'hui − 2H, aujourd'hui − H]`.
  Un seul chargement borné à 730 j sert les huit fenêtres (ADR 0036 I2 / I4), mis en cache
  jusqu'au prochain sync (I3), sections de durée déclarées (I6). Fenêtres glissantes en jours,
  pas calendaires (au contraire de `temporal.Period` 1m / 1y).
- **Valeurs** : `ComputeKPIStats` sur chaque fenêtre (taux de victoire, KDA agrégé ADR 0006,
  précision, valeurs par match, score de performance, OC / DR, `RankDelta`), plus réel contre
  attendu (`expected_stats`) et MMR moyen du lobby. Jamais de valeur par minute (décision du
  13/09).
- **Fiabilité** : chaque cellule porte le nombre de matchs des deux fenêtres ; sous un seuil
  nommé (constante, pas de nombre magique), la variation n'est pas affichée. Le seuil exact se
  fixe en regardant le volume réel de 7 j des joueurs suivis.
- **Verdict** : une phrase par horizon, tirée d'un nombre borné de métriques (taux de victoire,
  score de performance, réel contre attendu). Une seule définition de « en progrès » : celle
  d'Ascension (pente LOWESS, `progression/profile/lowess.go`) est réutilisée ou remplacée, pas
  dupliquée.
- **Courbe** : 365 j regroupés par semaine (`temporal.BucketByGranularity` + `LowessSmooth`),
  axe par date, bornes des horizons marquées.
- **Titre** : chaque bloc est gaté par capacité (`expected_stats`, `ranked`, `lusr`...), jamais
  par slug ; Halo 5 reçoit une page partielle propre.

### 6.3 Escouade

- Sélecteur : `GET /players/{slug}/squads?user_id=` (escouades dont le joueur connecté est
  membre, avec noms et membres par xuid). Pas de saisie de gamertags.
- Appartenance d'un match à l'escouade : la règle de l'ADR 0033 (le joueur principal l'a joué
  et chaque membre choisi était dans son équipe, peu importe qui était là à la fin ; composition
  stricte en option), en réutilisant la résolution du service teammates, pas une copie. Les
  comptes de matchs affichés viennent de cette seule source.
- Contenu par horizon : taux de victoire de la composition, nombre de sessions et de matchs,
  MMR d'équipe, score escouade (`squad_score.go`), score de performance de chaque membre,
  coordination (riposte, appui).
- Courbe : `ComputeSquadTimeseries` et Form Score, déjà calculés et aujourd'hui jamais affichés
  (§7.1) ; la page les consomme au lieu d'en écrire d'autres.
- À trancher à la maquette : un membre de l'escouade peut-il voir la page si un autre membre ne
  partage pas ses données (propriété des données joueur, ADR 0029) ?

### 6.4 Corrélation stat ↔ victoire (retenue dès le départ)

- Mesure : corrélation de Pearson entre chaque statistique d'un match et le résultat
  (victoire = 1, défaite = 0) ; les matchs sans résultat (égalité, abandon) sont écartés.
- **Uniquement sur 90 j et 365 j**, sous un seuil de matchs nommé : sur 7 j, le coefficient
  n'a pas de sens.
- Statistiques **par match** (décision du 13/09 : jamais par minute). Le biais de durée (un
  match long donne plus de frags) est tenu par les écarts à l'attendu (l'attendu du matchmaking
  intègre le temps joué) et par des lectures groupées : « ce que tu contrôles », « indicateurs
  composés », « contexte ».
- Présentation prudente : « associé aux victoires », pas « fait gagner » ; en jeu d'équipe, la
  corrélation reflète aussi le niveau des coéquipiers.
- Code : deux copies privées de `pearson` existent déjà
  (`analysis/skill_v2/mode_correlation_matrix.go:107`, `media/hls_audio_analyze.go:90`). Une
  troisième impose la règle n° 6 : un helper unique dans `internal/analysis/` qui remplace les
  deux copies, plus un garde-rail qui interdit une nouvelle copie.

### 6.5 Web (briques existantes)

Cellule « valeur + variation » : `DeltaCard` (sans consommateur aujourd'hui) ; tendances ▲▼ :
`MetricWithTrend` ; sections : `SectionCard` ; courbe : `ChartCard` + wrapper du README
charts ; échelles `perfScale` / `skillDeltaScale` ; textes via manifest i18n FR + EN ; clés de
requête dans `lib/query/keys.ts`.

### 6.6 Maquette (2026-09-30) et constats sur données réelles

Navigation validée par l'utilisateur : section « Tendances » dédiée, onglets Solo / Escouade,
seul filtre playlist / mode. Maquette : `.ai/MAQUETTE_TENDANCES_2026-09-30.html`, sur les
données réelles d'une COPIE locale (shared_matches_v2, shared_social, base JGtm ; dernier match
le 27/09/2026), requêtes par `cmd/diag_q` en lecture seule.

Constats mesurés (ils conditionnent le plan) :
- **7 j contre les 7 j d'avant est fragile.** Mesuré jour par jour du 01/11/2025 au
  27/09/2026 : les deux fenêtres ont au moins 10 matchs 66 % des jours pour JGtm, 41 % pour
  Chocoboflor, 35 % pour Madina97294 (au moins 5 : 79 / 58 / 59 %). Le 27/09, aucun des quatre
  joueurs suivis n'a de semaine précédente jouée. À 30 j : 94 à 96 % des jours.
- **365 j contre l'année d'avant n'existe pas encore** pour JGtm et Chocoboflor (historique
  réel depuis octobre 2025) ; Madina97294 : 110 matchs l'année d'avant.
- **Classé quasi absent** : 8 matchs classés sur 1 142 dans l'année pour JGtm (1 117 en Partie
  rapide). La page ne se centre pas sur le CSR ; bloc Classement gaté `ranked`, masqué vide.
- **Une escouade = environ une soirée par semaine** (Big bosses : 426 matchs, 53 soirées sur
  l'année). Sur 7 j, l'escouade, c'est une soirée.
- **Composition stricte** : « Les deux BGs » = 450 matchs sur l'année au moins à deux, 24 sans
  Madina97294 ni XxDaemonGamerxX dans l'équipe (ADR 0033).
- **Difficulté égale, le récit que la page d'exemple ne voit pas** (JGtm, 8 premières contre
  8 dernières semaines, pondéré par les matchs) : KDA −2,54 → +0,46, MMR adverse ~1 140 →
  ~1 240, écart aux frags attendus +2,06 → −0,40 par match.
- **Corrélation (JGtm, 365 j, 1 100 matchs)** : durée de vie +0,34, dégâts nets +0,29, meilleure
  série +0,28, morts −0,32, précision +0,01 (nulle) ; composés : score de perf +0,46, KDA
  +0,43 ; contexte : écart de MMR équipe − adversaires +0,37. Même ordre sur 90 j (184 matchs).
  `gamertag` de `match_participants` est souvent NULL : la jointure se fait par xuid.

**Maquette v2 (même jour, retour utilisateur « trop basique, des graphiques qui parlent plutôt
que des tableaux »)** : tableau supprimé, tout en graphiques ECharts (version de l'app), calculés
dans la page sur les 1 158 matchs de JGtm (une ligne par match, stats des trois coéquipiers
suivis dans son équipe, LUSR par chaîne, équipement du film). Blocs et constats réels :
- cartes KPI : valeur 30 j + variation, « échelle des horizons » (7 / 30 / 90 / 365 j, période
  d'avant contre maintenant) et courbe hebdomadaire sur 12 mois ;
- trajectoire mois par mois (MMR adverse × écart aux frags attendus) : 1 091 → 1 184 de MMR
  adverse, +2,27 → −0,34 frag par match (octobre → septembre) ;
- niveau LUSR par chaîne (le classé n'existe presque pas) : arène assassin 1 490 → 1 561,
  chaos 1 495 → 1 666, arène objectif 1 197 → 1 554 ;
- forme glissante sur 50 matchs avec bande d'incertitude à 90 % (Wilson) : 52 % [41 ; 63] ; les
  creux et pics de l'année (34 % à 68 %) restent en grande partie dans le bruit ;
- calendrier de l'année (couleur = taux de victoire du jour) : 146 jours joués, 7,8 matchs par
  jour joué, 11 jours d'affilée au plus. Jours découpés en UTC dans la maquette ; l'app devra
  découper en heure locale (voir §7.4) ;
- victoires contre défaites sur un axe COMMUN (écart à la moyenne du joueur, en écarts-types) :
  durée de vie 28,2 → 36,9 s (lien le plus net), morts 12,7 → 10,5, précision inchangée
  (42,8 → 42,9 %) ;
- soirées : score de perf 49,9 sur les trois premiers matchs, 53,1 à partir du 9e (plutôt un
  effet d'échauffement que de fatigue) ; après une défaite 50 % (430 cas), après une victoire
  54 % (450 cas) : écart dans l'incertitude, pas d'effet net de la défaite précédente ;
- style (3 premiers contre 3 derniers mois) : part des frags à la tête 43 → 26 %, à l'arme
  lourde 12 → 21 %, grappins par match 0,4 → 1,0, grenades lancées 8,1 → 12,8 ;
- médailles qui bougent (par match, 90 j contre 90 j) et mélange des types de partie par mois ;
- escouade (Big bosses) : 53 soirées (24 gagnées, 23 perdues, 6 à l'équilibre), Madina97294
  fait 47 % des frags de l'escouade, JGtm gagne 50 % avec elle contre 47 % sans coéquipier suivi
  mais avec un score de perf plus bas (49,0 contre 51,8), équipe à +12 de MMR sur ses
  adversaires ; composition stricte fonctionnelle (426 → 388 matchs sur l'année).
Pistes non maquettées faute de données dans la copie : frags par arme (classificateur Go du
film), vies isolées (placement des vies de l'Emprise), coordination.

**Décisions utilisateur du 2026-10-01** :
- **Horizons des cartes : dernière soirée · 30 · 90 · 365 j.** 7 j est retiré : sur 7 j, JGtm
  a 13 matchs, soit ± 23 points de marge à 90 % sur le taux de victoire (± 15 / ± 6 / ± 2 sur
  30 / 90 / 365 j), et la semaine d'avant est souvent non jouée. La soirée (`session_id`) est
  comparée aux soirées des 90 jours qui la précèdent, à titre INDICATIF, avec son propre seuil
  (5 matchs ; 10 pour la référence). Les 30 / 90 / 365 j restent comparés à la période d'avant.
- **Repris de la Synthèse et de Sessions : résistance défensive et rendement offensif**
  (`analysis/combat_yield.go`, assists à 1/3, 225 PV par frag via `games.EffectiveHpToKill`),
  en petits graphiques dans « Ton style évolue », à la place des parts de frags au corps à corps
  et à la grenade (plates). Constats JGtm : résistance 1,25 → 1,33, rendement 0,87 → 0,85
  (3 premiers contre 3 derniers mois). Proposés et non retenus : participation aux frags
  (24 → 30 %), lien vers la heatmap de la Synthèse.
- **Retours utilisateur du 2026-10-02, intégrés en maquette v3** :
  1. pas de nouvelle entrée L1. Proposition retenue en maquette : 6e onglet « Tendances » dans
     **Ascension** (progression ; pas sous `/stats/`, donc aucune barre de période héritée),
     sélecteur Solo / Escouade dans la page. Réserve : Ascension est gatée `lusr` et le
     réglage `show_progression`. L'alternative (un onglet sous Solo et un sous Escouade) hérite
     des barres de filtres et coupe la page en deux : écartée ;
  2. la trajectoire en nuage relié (zigzags) devient deux courbes alignées par mois : MMR des
     adversaires, écart aux frags attendus ;
  3. titres factuels, neutres, concis, sans tutoiement ni vouvoiement ; en FR « FDA », jamais
     « KDA » ;
  4. le mot « forme » est retiré (« moyenne glissante sur 50 matchs », « écart aux frags
     attendus par membre ») ;
  5. équipement en général, pas le grappin : bilan « utilisé / lâché en mourant / gardé sans
     servir » par la règle de `sessionusage.equipmentUsedOf` sur `EquipmentOutcomeFamilies`
     (répulseur, grappin, propulseur hors bilan) ; « grenades lancées » retiré. Couverture de
     la copie : 84 matchs résumés `us6` (juillet à septembre) ;
  6. cartes du haut : cinq propositions à choisir — A position dans l'année (valeur 30 j
     dans la distribution des fenêtres de 30 j de l'année : étendue, moitié centrale,
     médiane, point d'avant), B écart à l'année (sucettes 90 j / 30 j / soirée autour du
     niveau des 365 j), C courbe et paliers (12 mois + moyennes 365 / 90 / 30 j en paliers
     + dernière soirée), D écart mensuel à la moyenne de l'année (barres divergentes),
     E échelle des horizons (v2) ;
  7. moins de texte : chaque bloc = un titre, une icône (i) si nécessaire, le graphique, la
     légende centrée en bas. Phrases de lecture et sous-titres supprimés ; notes d'étude
     repliées sous la maquette.
- **Réponses utilisateur du 2026-10-02** : **Ascension VALIDÉ** (6e onglet, sélecteur Solo /
  Escouade dans la page). Cartes : D plaît mais perd les horizons (« si ça ne marche pas avec
  ces horizons, aucune solution ne me va ») ; A, B, C, E écartées. Deux nouvelles propositions
  en maquette : **F · variation par horizon** (une barre par horizon 365 j / 90 j / 30 j /
  soirée, hauteur = variation par rapport à la période d'avant, valeur actuelle sous la barre,
  « n.d. » sans période d'avant, soirée pâle car indicative) et **G · horizons et mois** (F
  au-dessus, l'écart mensuel D en dessous). Choix en attente.
- **Retour utilisateur du 2026-10-02 (suite)** : F et G ne conviennent pas non plus ; « l'idée
  des mini-cartes n'est peut-être pas adaptée pour voir l'évolution sur plusieurs périodes en
  même temps ». Les cartes sont REMPLACÉES par UN graphique pleine largeur, tous indicateurs ×
  toutes périodes, deux propositions :
  - **Matrice** (recommandée) : une ligne par indicateur ; 12 colonnes de mois (couleur = écart
    du mois à la moyenne des mois) puis 4 colonnes d'horizons 365 j / 90 j / 30 j / soirée
    (couleur = variation par rapport à la période d'avant ; soirée par rapport aux soirées
    des 90 jours d'avant) ; valeur écrite dans la case ; gris = pas de comparaison. Échelle
    commune : écarts-types des valeurs mensuelles de l'indicateur, orientés « + = mieux ».
  - **Trajectoires** : une ligne par indicateur, points 90 j / 30 j / soirée placés par rapport
    au niveau des 365 j sur un axe commun (les comptes, comme les soirées jouées, n'y figurent
    pas : un compte de 90 j ne se compare pas à celui de 365 j).
  Piège de maquette rencontré : ECharts n'interpole pas les couleurs `oklch` des jetons ; la
  rampe de la matrice passe par trois couleurs hexadécimales par thème.
- **Décision utilisateur du 2026-10-02 : la MATRICE est retenue** (« c'est la matrice qui me
  plaît, les trajectoires bof ») ; Trajectoires retirées de la maquette. Ajout demandé, en
  complément : un **graphique de détail avec un menu déroulant** pour la stat choisie (mêmes
  données, plus parlant). Maquetté : un point par mois (vert / rouge par rapport à la moyenne
  des mois, taille = matchs), la moyenne des mois en pointillé, les paliers 90 j et 30 j
  (couleur = variation) à côté du palier de même durée juste avant (pointillé, au survol), la
  dernière soirée en losange ; un compte (soirées jouées) est ramené à un équivalent par
  30 jours. Cliquer une ligne de la matrice sélectionne la stat du détail. Même bloc sur
  l'onglet Escouade.
- **Retour utilisateur du 2026-10-02** : les paliers horizontaux superposés aux mois ne se
  comprenaient pas (« mis à part la moyenne des mois, je ne comprends pas ces barres »).
  Décision utilisateur : **barres groupées « avant / maintenant »**. Le détail se coupe en
  deux : « Par mois » (points et moyenne des mois) et « Par horizon » (365 j, 90 j, 30 j,
  soirée ; barre grise = période d'avant de même durée, barre colorée = maintenant, vert /
  rouge selon la variation, bleu sans période d'avant ; valeurs écrites sur les barres ;
  soirées jouées ramenées à un rythme par 30 jours). Leçon : un encodage qu'il faut expliquer
  par la légende ne « parle pas de lui-même » ; préférer une forme connue (barres groupées) à
  une superposition astucieuse.
- **Recadrage utilisateur du 2026-10-02** : « ça sert à quoi ces barres ? On n'a pas besoin
  d'un troisième graphique. » Constat : les barres avant / maintenant répétaient exactement
  les colonnes d'horizons de la matrice. **Le détail n'apporte que ce que la matrice montre
  mal : l'ampleur de l'évolution mois par mois.** Il redevient UN graphique : une barre par
  mois au-dessus ou en dessous de la moyenne des mois (couleurs de la matrice), valeur réelle
  sur la barre. Les horizons vivent dans la matrice seule. Règle à garder pour la suite : un
  bloc ne répète pas une information qu'un autre bloc de la page porte déjà.
- **Questions utilisateur du 2026-10-02 sur « Difficulté des lobbys et écart à l'attendu »** :
  1. *Un graphe à plusieurs axes plutôt que deux ?* Non : deux échelles verticales sur un même
     graphique (MMR ~1 100-1 250, écart ~−1 à +2) font lire des croisements et des pentes qui
     dépendent du choix arbitraire des deux échelles (erreur n° 1 des graphiques). Deux
     panneaux empilés sur le même axe des temps racontent la même chose sans ce piège ; le
     nuage relié (une seule zone) avait été refusé pour ses zigzags.
  2. *Pourquoi les frags plutôt que le FDA ?* Retenu : le FDA attendu existe déjà côté Go
     (`analysis.ExpectedFDA` = frags attendus + assists attendues / 3 − morts attendues, écart
     `FDADiff`, déjà tracé par les Séries temporelles). Sur JGtm, l'écart aux frags plonge
     (+2,27 → −0,34) quand l'écart au FDA reste positif l'été (+0,56 en juillet) parce que les
     morts baissent sous l'attendu (−1,02 en juillet) : le FDA dit plus juste. La maquette
     prend les assists attendues du modèle POPULATIONNEL global (`assists_model_coefs`,
     `__global__`) ; l'app utilise le modèle personnel puis par variante.
  3. *Une liste déroulante ?* Oui, limitée aux trois écarts à l'attendu (FDA par défaut,
     frags, morts) : seules ces mesures sont corrigées de la difficulté ; toute autre stat se
     parcourt déjà dans le détail sous la matrice. Le haut reste le MMR adverse.
  Dans la matrice, « Écart aux frags attendus » devient « Écart au FDA attendu ».
- **Suite (2026-10-02)** : l'utilisateur veut aussi les valeurs RÉELLES (« les attendus c'est
  bien mais le réel aussi ») : le menu gagne « FDA », « Frags par match », « Morts par match »,
  chacun avec l'attendu en pointillé sur le même axe (même unité, pas de double axe). Puis
  « on ne peut pas avoir les deux stats sur un graphe X, Y, Z ? » : si — double axe faisable
  mais trompeur (lecture dépendante des échelles), 3D illisible sans rotation, troisième
  dimension EN COULEUR propre. Décision utilisateur : maquetter les deux. Bouton « Deux
  panneaux / Un graphique » : la variante « un graphique » porte la mesure en Y et colore
  chaque mois selon le MMR adverse (rampe séquentielle bleue clair → foncé, barre de couleur
  « lobbys faciles → lobbys durs »). Choix en attente.
  **Corrigé le même jour** sur retour utilisateur (« en mode un graphique tu n'as pas mis le
  MMR ») : la couleur ne se lisait pas comme une donnée. La variante « un graphique » est
  désormais un **double axe** : mesure choisie à gauche (bleu, graduations bleues), MMR adverse
  à droite (courbe et graduations orange `--c3`), attendu en pointillé gris le cas échéant.
  Leçon : quand l'utilisateur demande « les deux stats sur le même graphe », il veut voir les
  deux courbes ; ne pas substituer un encodage indirect sans le lui montrer d'abord.
  **Décision utilisateur du 2026-10-02 : « un graphique c'est nickel, on garde ça ».** La forme
  « deux panneaux » et le bouton sont retirés de la maquette ; le bloc est un graphique unique à
  double axe avec le menu (écarts au FDA / frags / morts attendus ; FDA, frags, morts réels avec
  l'attendu en pointillé).

- **« Moyenne par match en défaite et en victoire » (question utilisateur du 2026-10-02 : un
  papillon ? ça exprime le poids des stats sur les chances de victoire ?)** — Réponse : le
  graphique montrait une ASSOCIATION (moyenne en défaite contre en victoire, axe en
  écarts-types, r), pas un poids sur la probabilité ; un papillon sur ces moyennes ne marchait
  pas (unités différentes, dégâts nets négatifs, écart illisible entre deux barres presque
  égales). Recadrage : **taux de victoire selon le niveau de la statistique** — pour chaque
  stat, taux de victoire dans le quart des matchs où elle est la plus basse (≤ Q1) et dans le
  quart où elle est la plus haute (≥ Q3) ; unité commune (% de victoires), écart en points.
  Deux formes maquettées (bouton) : **papillon** (quart bas à gauche, quart haut à droite,
  barres depuis le centre) et **haltères** (axe commun 0-100 %). JGtm, 365 j : durée de vie
  24 → 74 % (+50), morts 73 → 30 % (−43), dégâts nets 31 → 73 % (+43), frags 35 → 66 %
  (+31), précision 50 → 51 % (+1) ; contexte : écart de MMR équipe − adversaires 27 → 77 %
  (+49). Choix de forme en attente.
  **Retour utilisateur (même jour) : « j'aime pas le nouveau papillon, c'était mieux avant ».**
  Retour à la version précédente : « Moyenne par match en défaite et en victoire » (axe commun en
  écarts-types, valeurs réelles, r), sans bouton de forme.
- **Recadrage utilisateur du 2026-10-02 : « pourquoi 50 matchs arbitrairement ? La page sert à
  voir les stats sur des horizons précis » (même remarque pour « Soirées »).** Règle appliquée à
  toute la page : un bloc montre les 12 mois et / ou les horizons, ou apporte ce que la matrice ne
  peut pas montrer. Décisions utilisateur :
  - **retirés** : moyennes glissantes sur 50 matchs (fenêtre arbitraire, doublon de la matrice),
    section Soirées (comportement intra-soirée : piste pour la page Sessions), calendrier
    (doublon du calendrier d'activité d'Ascension > Réalisations) ;
  - **passés en lignes de la matrice** (et donc du détail) : résistance défensive, rendement
    offensif, part des frags à la tête, part des frags à l'arme lourde, part de l'équipement
    utilisé, LUSR arène assassin / chaos / arène objectif (niveau = dernière valeur de la
    fenêtre ; horizon = progression contre la période d'avant) ;
  - **médailles** : choix d'horizon 30 / 90 / 365 j contre la période d'avant (365 j : pas
    d'année d'avant chargée, médailles les plus fréquentes) ;
  - **escouade** : mêmes règles ; matrice = taux de victoire, taux de victoire sans coéquipier
    suivi, soirées jouées, part des frags de l'équipe, MMR équipe − adversaires, écart au FDA
    attendu par membre, part des frags de l'escouade par membre ; retirés : bilan par soirée,
    répartition, écart par membre, effet escouade, MMR (devenus lignes de la matrice).
  - gardé : « Matchs par type de partie » (contexte).
- **Demandes utilisateur du 2026-10-02 (suite)** : « remets le calendrier » → calendrier des
  résultats rétabli (avant « Victoires et défaites ») malgré le doublon avec Ascension >
  Réalisations ; et un **nuage de points en complément** de la vue globale, statistique choisie
  dans une liste. Un nuage brut « stat × victoire » s'empilerait sur deux lignes (résultat
  binaire) : maquetté « Taux de victoire selon la valeur » — chaque match en petit point
  (victoires en haut, défaites en bas, dispersion déterministe), et le taux de victoire de chaque
  dixième des matchs rangés par valeur en gros points reliés, un seul axe 0-100 %, r dans le nom
  d'axe ; suit l'horizon 365 / 90 j de la section. JGtm 365 j : durée de vie de ~20 % à ~80 % de
  victoires, précision plate (r ≈ 0).
  **Refusé aussitôt (« il est dégueu, les points ne sont que sur 0 % et 100 % »)** : un point par
  match ne peut pas faire un nuage, le résultat d'un match est binaire. Remplacé par **un point par
  SOIRÉE** (« Taux de victoire des soirées selon la statistique ») : abscisse = moyenne de la stat
  sur la soirée, ordonnée = taux de victoire de la soirée (continu), taille = matchs, droite de
  tendance par moindres carrés pondérés par les matchs, soirées d'au moins 3 matchs. JGtm 365 j :
  154 soirées ; r entre soirées : écart de MMR +0,49, score de perf +0,42, meilleure série +0,36,
  durée de vie +0,34, morts −0,34, frags +0,15, précision +0,01.
  **Refusé aussi (« qu'est-ce que tu me parles de soirée ? »)** : regroupement non demandé.
  Version retenue : **un point par MATCH, « Matchs gagnés et perdus selon la statistique »** —
  abscisse = statistique choisie, ordonnée = écart de score final en % du score le plus haut
  (positif = gagné, vert ; négatif = perdu, rouge ; manches pour les modes de
  `regulation.toml [rounds_decide]`, Oddball ; signe cohérent avec le résultat sur 1 157 / 1 157
  matchs), droite de tendance. JGtm 365 j, r stat × écart : score de perf +0,45, FDA +0,41, durée
  de vie +0,40, écart de MMR +0,39, morts −0,31, frags +0,21, précision −0,03. Dans l'app, l'écart
  passera par `analysis.ReadTeamScore` (jamais une comparaison de score à la main).
  **Refusé aussi (« toujours de la merde »)**, puis un essai de copie des nuages de l'onglet
  Distributions (deux stats au choix en X et Y) : hors sujet, la demande est une stat face au
  résultat. **Version retenue (demande utilisateur du 2026-10-02 : « une stat par rapport au
  résultat du match »)** : « Statistique selon le résultat du match » — abscisse = statistique
  choisie, une rangée Victoire et une rangée Défaite (refusé : « pourquoi avoir séparé la victoire
  et la défaite ? »). Puis **un seul nuage** : un point par match, date en abscisse sur l'horizon
  de la section, statistique choisie en ordonnée, couleur = résultat ; style des points de
  `TimeseriesScatterWithTrend` (taille 5, opacité 0,7, couleurs victoire / défaite). Rien d'autre
  (une boîte à quartiles ajoutée sans demande a été retirée).
  **Nuage abandonné (décision utilisateur du 2026-10-02 : « le nuage, c'est pas le bon choix »).**
  Question reposée : voir, sur les horizons, l'influence d'une statistique sur le résultat.
  Recommandation en attente de validation : une matrice statistiques × horizons (365 / 90 / 30 j),
  case = écart de taux de victoire entre les matchs au-dessus et au-dessous de la médiane du
  joueur, en points ; marge à 90 % au survol, case grisée quand la marge dépasse l'écart ; pas de
  colonne soirée. JGtm : durée de vie +34 / +33 / +54 pts (marge ± 5 / 12 / 24), morts −37 / −35 /
  −48, FDA +37 / +39 / +48, précision −1 / +6 / +3 ; 30 j = 32 matchs seulement.
  **Matrice refusée (« pas de matrice ; pour UNE SEULE stat » ; les haltères font déjà le
  détail).** Validé et maquetté le 2026-10-02 : « Taux de victoire selon la statistique, par
  horizon » — statistique en liste déroulante, horizons 365 / 90 / 30 j en abscisse, deux barres par
  horizon (taux de victoire au-dessus / en dessous de la médiane du joueur sur cet horizon), survol :
  médiane, matchs, marge à 90 %. Pas de soirée (2 à 5 matchs par barre). Durée de vie JGtm :
  67 / 33 % (365 j), 66 / 32 % (90 j), 67 / 13 % (30 j, 15 et 16 matchs).
  Retour utilisateur du 2026-10-02 : horizon 7 j ajouté (seul bloc de la page qui l'a, à la
  demande) et valeur de la statistique en **courbe sur un second axe** (moyenne de la stat sur
  l'horizon, axe de droite en orange), à la place des libellés médiane / moyenne. Durée de vie
  moyenne JGtm : 32,6 s (365 j), 38,1 s (90 j), 35,2 s (30 j, 32 matchs), 35,2 s (7 j, 13 matchs).
- **Recadrage utilisateur du 2026-10-02 : les stats des blocs retirés (glissant 50 matchs,
  Soirées) gardaient leur intérêt ; les suivre proprement dans la logique de la page.** Fait :
  - **Soirées → lignes de la matrice** (groupe « Soirées », donc aussi dans le détail) : taux de
    victoire après une victoire, taux de victoire après une défaite (match suivant de la même
    soirée, au moins 5 enchaînements), score de perf fin − début de soirée (2e moitié moins 1re
    moitié de chaque soirée d'au moins 4 matchs notés : pas de seuil « 9e match » arbitraire),
    matchs par soirée. JGtm 365 j : 54 % après une victoire, 50 % après une défaite, +1,6 de
    perf en fin de soirée, 5,8 matchs par soirée.
  - **Glissant 50 matchs** : taux de victoire et score de perf étaient déjà dans la matrice ; ce
    qu'ils apportaient en plus, l'incertitude, passe dans le survol des cases de taux de
    victoire (« marge à 90 % : ± x pts », Wilson) pour chaque mois et chaque horizon.
  - La matrice reçoit des intertitres de groupe (Résultats, Combat et style, Soirées, Niveau
    LUSR ; Escouade, Membres) ; lignes ramenées à 34 px.

### 6.7 État des décisions (2026-10-02)

| Sujet | Décision |
|---|---|
| Emplacement | 6e onglet « Tendances » d'Ascension, sélecteur Solo / Escouade dans la page |
| Horizons | dernière soirée (vs soirées des 90 j d'avant, indicatif, seuil 5), 30, 90, 365 j (vs période d'avant, seuil 10) |
| Principe | chaque bloc montre les 12 mois et / ou les horizons ; pas de fenêtre arbitraire, pas de doublon |
| Indicateurs clés | matrice (Solo : 14 lignes, dont style, équipement, LUSR par type de partie ; Escouade : 11 lignes, dont membres) × (12 mois + 4 horizons) + détail à liste déroulante (barres mensuelles autour de la moyenne) |
| Victoires / défaites | « Moyenne par match en défaite et en victoire » (version d'origine) + « Taux de victoire selon la statistique, par horizon » (une stat au choix ; 365 / 90 / 30 / 7 j ; barres au-dessus / en dessous de la médiane + courbe de la moyenne de la stat sur un second axe). Nuage abandonné, 365 / 90 j |
| Calendrier | rétabli à la demande de l'utilisateur |
| Médailles | choix d'horizon 30 / 90 / 365 j |
| Soirées | lignes de matrice : victoire après une victoire / une défaite, perf fin − début de soirée, matchs par soirée |
| Incertitude | marge à 90 % au survol des taux de victoire (remplace les moyennes glissantes) |
| Retirés | graphiques « glissant 50 matchs » et « Soirées » (leurs stats vivent dans la matrice), graphiques mensuels d'escouade |
| Niveau | « Difficulté des lobbys et écart à l'attendu » : un graphique, double axe, menu réel / écarts |
| Corrélation stat ↔ victoire | dès le départ (90 / 365 j), présentée en victoires / défaites sur axe commun |
| Équipement | bilan utilisé / lâché / gardé (règle Sessions), pas un équipement isolé |
| Repris de Synthèse / Sessions | résistance défensive et rendement offensif |
| Textes | titre neutre + (i) + graphique + légende en bas ; FDA, jamais KDA |

Aucune implémentation (consigne utilisateur).
- Laissés à leur page (bilans, pas des tendances, ou plats) : heatmap jour × heure, meilleurs
  et pires matchs, totaux et records, solo contre escouade, semaines, premier frag (8 à 12 %,
  le niveau du hasard), portée (conversion d'unités côté Go), engagement, intensité par phase.

Décisions à prendre sur la maquette :
1. seuil de matchs par fenêtre (5 / 10 / 20) ;
2. période précédente vide ou sous le seuil : ne rien afficher (maquette) ou retomber sur
   l'année comme référence (variante sélectionnable dans la maquette) ;
3. escouade : libeller les horizons en soirées.

Ensuite : plan d'implémentation soumis à `plan-review` (pas avant accord).

## 7. Découvertes hors périmètre (notées, non traitées)

1. Série escouade `timeseries` calculée à chaque `POST /pages/teammates` et jamais lue ;
   `features/squad/charts/timelineChart.ts` n'est importé que par son test ; Form Score
   servi seulement par `/pages/squad/v2`, que le web n'appelle pas (hors engagement). Soit
   Tendances escouade les consomme, soit c'est du code mort (règle 7).
2. `components/filters/PeriodFilter.tsx` : importé seulement par son test.
3. `DeltaCard` et `features/squad/components/PlayerScoreCard.tsx` : aucun consommateur.
4. Heatmap jour × heure calculée sur l'heure UTC (`squad_breakdown_canonical.go:322`) : vérifier
   si le web convertit en heure locale.
5. Histogramme de perf des Séries temporelles : seuils 20/40/60/80, différents de `perfScale`
   (80/65/50/35).
6. Composition stricte : cochée par défaut côté web, `false` par défaut côté serveur, et
   l'ADR 0033 la décrit désactivée par défaut.
7. `components/charts/README.md` cite un EWMA qui n'existe plus ; le skill
   `frontend-patterns` décrit l'ancien modèle i18n `features/<f>/i18n.ts`.
8. Synthèse : réécrit à la main le switch de période au lieu de `temporal.Period`.

## Annexe — entrée du journal du 2026-09-30 (jamais versionnée, reprise le 2026-10-19 avec l étude)


### [2026-09-30] Étude pages « Tendances » solo et escouade (inspiration halo-squad-stats) — En cours (`claude/trend-analysis-halo-stats-8771a8`)

**Décision technique principale** : ne pas copier la page `/trends` du dépôt d'exemple (fenêtre unique + moyennes journalières superposées), redondante avec les Séries temporelles + la pastille Période. Une page Tendances ne se justifie que par la comparaison de plusieurs horizons (7/30/90/365 j) côte à côte, depuis aujourd'hui, chacun face à une référence, avec un seuil de matchs ; escouade choisie parmi les escouades enregistrées (Prestige, `GET /squads`). Tout se construit sur l'existant : `ComputeKPIStats` par horizon, `temporal` (bucket + LOWESS), `DeltaCard` (sans consommateur), `ComputeSquadTimeseries` et Form Score (calculés, jamais affichés).

**Résultats observés** : lecture sur pièces du dépôt d'exemple (`webapp.py` route `/trends`, `grades.py`) ; inventaire web, escouade et Go par trois explorations, affirmations clés revérifiées (préréglages 7d/30d/90d/all sans 1 an, `temporal.Period` sans 90 j, série escouade non lue, `DeltaCard` sans consommateur, Prestige actif par défaut). Étude complète : `.ai/ETUDE_TENDANCES_2026-09-30.md` (redondance, métriques pour compétitifs et occasionnels, briques, 8 découvertes hors périmètre non traitées).

**Conclusion / prochaine étape** : décisions utilisateur prises : comparer les horizons ; variation contre la période précédente de même durée ; corrélation stat ↔ victoire dès le départ (90 / 365 j seulement, valeurs par minute, un helper Pearson unique qui remplace les deux copies privées existantes). Navigation : section L1 « Tendances » dédiée (des onglets sous Stats ou Escouade hériteraient des filtres L2), validée ; seul filtre playlist / mode. Maquette `.ai/MAQUETTE_TENDANCES_2026-09-30.html` sur données réelles (copie locale des bases, `cmd/diag_q` en lecture seule) : 7 j contre les 7 j d'avant comparable seulement 35 à 66 % des jours (≥ 10 matchs de chaque côté), aucune année précédente pour JGtm, 8 matchs classés sur 1 142, une escouade = environ une soirée par semaine ; récit « à difficulté égale » de JGtm (KDA −2,54 → +0,46 pendant que le MMR adverse monte de ~100 et que l'écart aux frags attendus retombe de +2,06 à −0,40). Correction : plus de valeur « par minute » dans l'étude (décision utilisateur du 13/09, tout par match). Décisions ouvertes : seuil de matchs, repli quand la période précédente est vide, horizons d'escouade en soirées. Maquette v2 le même jour (retour utilisateur : « trop basique, des graphiques qui parlent plutôt que des tableaux ») : tableau supprimé, 25 graphiques ECharts calculés dans la page sur les 1 158 matchs réels (trajectoire MMR × écart attendu, LUSR par chaîne, forme glissante avec bande de Wilson, calendrier, victoires/défaites sur un axe commun en écarts-types, soirées et « après une défaite », style, médailles, mélange des parties ; escouade : soirées, qui porte l'équipe, forme de chacun, effet escouade, matchmaking, composition stricte). Détails et constats chiffrés : étude §6.6. Le 2026-10-01, décisions utilisateur : horizons « dernière soirée · 30 · 90 · 365 j » (7 j retiré : ± 23 points de marge sur 13 matchs ; soirée comparée aux soirées des 90 jours d'avant, indicative, seuil 5) ; repris de la Synthèse et de Sessions : résistance défensive et rendement offensif (combat_yield, 225 PV), à la place de deux petits graphiques plats ; participation aux frags et lien heatmap proposés, non retenus. Le 2026-10-02, maquette v3 sur retours utilisateur : pas de L1 (proposé : onglet « Tendances » dans Ascension, sélecteur Solo / Escouade dans la page), trajectoire remplacée par deux courbes alignées, titres factuels sans tu/vous, « FDA » et non « KDA », « forme » retiré, équipement en général (utilisé / lâché / gardé, règle `sessionusage.equipmentUsedOf`, 84 matchs couverts dans la copie) à la place du grappin et des grenades lancées, cinq propositions pour les cartes du haut (position dans l'année, écart à l'année, courbe et paliers, écart mensuel, échelle), texte réduit à titre + (i) + graphique + légende. Puis réponses utilisateur du 2026-10-02 : Ascension validé ; cartes : D apprécié mais perd les horizons, A/B/C/E écartées ; ajout de F (une barre par horizon = variation par rapport à la période d'avant, valeur sous la barre) et G (F + écart mensuel), puis refusées aussi (« les mini-cartes ne sont peut-être pas adaptées ») : cartes remplacées par un graphique unique tous indicateurs × toutes périodes, deux propositions, Matrice (12 mois + 4 horizons, couleur = écart en écarts-types orienté « mieux », valeur dans la case) et Trajectoires (90 j / 30 j / soirée par rapport à l'année). Matrice RETENUE par l'utilisateur (Trajectoires retirées) ; ajout demandé et maquetté : graphique de détail avec menu déroulant (mois colorés par rapport à leur moyenne, paliers 90 j / 30 j contre le palier de même durée juste avant, dernière soirée), sélection aussi par clic sur une ligne de la matrice. Paliers horizontaux du détail incompris par l'utilisateur : remplacés (décision utilisateur) par un détail en deux graphiques, « Par mois » et « Par horizon » en barres groupées avant / maintenant, puis recadrage utilisateur (« pas besoin d'un troisième graphique ») : ces barres répétaient les colonnes d'horizons de la matrice ; le détail redevient un seul graphique, barres mensuelles autour de la moyenne des mois, horizons dans la matrice seule. « Difficulté des lobbys » : deux panneaux gardés (pas de double axe), écart au FDA attendu (`analysis.ExpectedFDA`, assists attendues du modèle populationnel dans la maquette) à la place des frags, menu limité aux trois écarts à l'attendu ; matrice alignée. Puis, sur retour utilisateur : valeurs réelles ajoutées au menu (réel en trait plein, attendu en pointillé, même axe) et deux formes maquettées à comparer, « deux panneaux » et « un graphique » où la couleur des points porte le MMR adverse, remplacé aussitôt (retour utilisateur : le MMR n'y apparaissait pas) par un double axe, mesure à gauche en bleu, MMR adverse à droite en orange ; RETENU par l'utilisateur (« un graphique c'est nickel »), forme à deux panneaux retirée. État des décisions : étude §6.7. Préférences de présentation enregistrées en mémoire (feedback_dataviz_page_conventions). Bloc victoires / défaites recadré (question utilisateur sur un papillon) : taux de victoire dans le quart bas et le quart haut de chaque statistique (unité commune, écart en points), formes papillon et haltères maquettées, puis refusées (« c'était mieux avant ») : version d'origine rétablie. Recadrage utilisateur « horizons précis » : moyennes glissantes 50 matchs, Soirées et calendrier retirés ; style, équipement et LUSR par type de partie passés en lignes de la matrice ; médailles avec choix d'horizon ; escouade réduite à matrice + détail (lignes par membre). Puis calendrier rétabli (demande utilisateur) et nuage ajouté en complément ; première version (un point par match) refusée car binaire (points sur 0 % et 100 % seulement), remplacée par un point par soirée, refusée à son tour (regroupement non demandé), puis par un point par match avec l'écart de score final en ordonnée (gagnés au-dessus de 0, perdus en dessous, tendance). Puis, sur recadrage utilisateur, les stats des blocs retirés reviennent dans la logique de la page : groupe « Soirées » dans la matrice (victoire après victoire / défaite, perf fin − début de soirée par moitiés, matchs par soirée) et marge d'incertitude à 90 % au survol des taux de victoire ; intertitres de groupe dans la matrice. Nuage à écart de score refusé aussi, puis demande utilisateur du 2026-10-02 (« une stat par rapport au résultat du match ») : « Statistique selon le résultat du match » ; rangées Victoire / Défaite séparées refusées, devenues un seul nuage : un point par match, date en abscisse, stat choisie en ordonnée, couleur = résultat (style des points de `TimeseriesScatterWithTrend`), sans ajout. Nuage abandonné ensuite (décision utilisateur du 2026-10-02) ; matrice d'influence proposée puis refusée (une seule stat demandée) ; validé et maquetté : « Taux de victoire selon la statistique, par horizon », une stat au choix, deux barres par horizon 365 / 90 / 30 / 7 j (au-dessus / en dessous de la médiane du joueur), courbe de la moyenne de la stat sur un second axe, marge au survol (étude §6.6). Étude §6.6-6.7. Consigne utilisateur : étude du concept seulement, pas d'implémentation. Rien de commité.
