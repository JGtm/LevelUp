# Plan — Revue analytique Timeseries & Escouade (corrections, surlignage, nouveaux angles)

**Date** : 2026-07-12 — **revise le 2026-07-23** (revue UX avec l'utilisateur, elagage des ajouts)
**Branche Git** : `feat/analytics-review-ts-squad` (une branche, un commit par lot)
**Statut** : Valide, pret a executer (DEC tranchees le 2026-07-23 — voir tableau)
**Effort estime** : ~2,5 a 3 j (A 0,5 + B ~1,4 + C 0,1 + D 0,1 + F 0,5)
**Contrat d'execution** : skill `plan-execution` (ordre strict, gates, statuts `[x]`/`[~]`/`[!]`,
zero fix hors perimetre — decouvertes consignees en fin de fichier).

---

## Contexte

Revue critique du 2026-07-12 (voir thought_log a cette date) : trois passes (Timeseries,
Escouade, gisement backend) ont identifie des defauts de rendu, des redondances
possibles, un stack Escouade V2 dormant et des angles analytiques non exploites.

**Remarques utilisateur integrees (cadrage ferme)** :

1. **Rien n'est supprime dans ce chantier.** Les candidats a suppression sont
   SURLIGNES a l'ecran (lot A) ; l'utilisateur tranche apres verification visuelle.
   Les suppressions validees partent dans un chantier cleanup ulterieur.
2. **`expected_win_prob` est juge peu fiable** (constat utilisateur) : il reste
   cantonne a sa cellule de tableau. Aucune feature nouvelle ne s'appuie dessus.
   Les nouveautes « skill » se limitent aux donnees observees (`SkillRatingDelta`)
   et a l'incertitude (`sigma`), voir F7.
3. **`/pages/squad/v2` est un reliquat non identifie** : prudence. Aucune bascule
   d'endpoint ; on PORTE ponctuellement des calculs utiles vers le stack live
   (`teammates`), chaque port re-verifie sur pieces (le code V2 n'a jamais tourne
   en prod → repute non fiable par defaut). Le sort du stack V2 = DEC-7, chantier
   separe.
4. **Les « redondances » ne sont pas actees** : l'utilisateur n'en constate pas a
   l'usage. Elles recoivent un badge de doute avec une note posant la question
   (le rendu/narratif varie-t-il assez ?) — aucune suppression.
5. `.ai/V7/` est un dossier d'ARCHIVAGE : ce plan vit dans `.ai/`, comme
   `.ai/PLAN_MATCHVIEW_MOMENTUM_2026-07.md` (deplace le 2026-07-12).

**Revision 2026-07-23 (revue UX avec l'utilisateur)** — le chantier est recentre sur
« rendre juste et lisible l'existant » ; les ajouts sont elagues :

- **Supprimes** : F2 fatigue intra-session (biais systematique, pas du bruit : une
  bonne session amene des adversaires plus forts en fin de soiree via le
  matchmaking — le « decrochage » serait un artefact, et le volume ne corrige pas
  un biais) ; F3 premier sang (causalite inversee : les equipes qui gagnent
  prennent le premier sang parce qu'elles sont meilleures — insight illusoire, et
  item le plus cher du lot) ; F4 part de contribution (valeur portee par le
  narratif, chart exigeant une interaction pour livrer) ; F5 solo vs escouade
  (aucune des deux pages n'est le bon emplacement — candidat futur chantier
  Synthese, consigne en Decouvertes) ; F6 breakdown par mode (breakdown statique
  sur une page dont l'objet est le temps) ; F7a delta rating (3e lecture du skill
  sur un onglet dont la redondance est deja questionnee en C2) ; F7b bande
  mu±sigma (contredit la regle « afficher la metrique connue de l'utilisateur —
  LUSR/CSR oui, mu/sigma non ») ; E1/E2/E3 (form score = metrique synthetique
  opaque recouvrant les trends existants ; stack V2 intact, DEC-7 inchangee) ;
  D1 heatmap jour×heure (fun-fact bruite → liste cleanup).
- **Remplace** : F1 comeback agrege (tuiles KPI abandonnees — les flags
  remontada/debandade/contre-remontada ne sont JAMAIS produits sans timeline de
  score, cf. `internal/analysis/comeback.go:218` : un compteur de periode serait
  silencieusement fausse par la couverture partielle) → nouveau lot F : dominance
  flags affiches PAR MATCH sur la `OutcomeSequenceTape` des deux pages (fiable
  match par match, l'absence de flag ne ment pas).
- **B6 tri des armes** : deja traite dans un chantier dedie (constat utilisateur
  2026-07-23) → statut `[~]`.

**Dependance** : OBSOLETE depuis la revision 2026-07-23 — F4 et F7 (barres
divergentes) sont supprimes, ce plan ne reutilise plus le patron du chantier
momentum. Aucune extraction `DivergingBarChart` a prevoir ici.

## Objectif & critere de succes

Corriger les defauts de lecture identifies, brancher les gains rapides, ajouter les
angles valides — et que CHAQUE graphe touche, ajoute ou suspecte porte un badge
visible a l'ecran (« A verifier » / « Nouveau » / « Suppression ? ») avec une note,
pour que l'utilisateur puisse balayer les deux pages et statuer sans rien rater.
Succes = tous les gates verts + tournee visuelle utilisateur ou chaque badge est
statue (l'entree du manifest est retiree une fois l'item valide).

## Hors perimetre (explicite)

- Toute suppression (chart, champ backend, stack V2) — surlignage seulement.
- Bascule des pages Escouade vers l'endpoint `/pages/squad/v2` (DEC-7, plus tard).
- Toute feature fondee sur `expected_win_prob` (y compris chart de calibration).
- Le chantier momentum Match View (plan separe).
- Nemesis/rivalites sur ces pages (reste sur Career/Palmares — non demande).
- Refonte de la definition de l'escouade (intersection sous-ensemble) : constat
  documente, pas de changement ici.
- Tous les items supprimes par la revision 2026-07-23 (ex-F2..F7, ex-E1..E3,
  ex-D1) — voir le bloc Revision dans le Contexte ; F5 consigne en Decouvertes
  comme candidat futur chantier Synthese.

---

## Decisions (DEC) — TRANCHEES le 2026-07-23 (revue UX avec l'utilisateur)

| # | Question | Decision |
|---|---|---|
| DEC-1 | Lots retenus | **A + B + C + D + F** (F redefini : dominance flags sur la tape ; E supprime) |
| DEC-2 | B2 rendement : Combat Yield remplace `TimeseriesEfficiency` ou cote a cote ? | **Cote a cote** + badge « Suppression ? » sur l'ancien — l'utilisateur tranche a l'ecran |
| DEC-3 | B3 radar : recalibrage des seuils | **Oui, P80 uniforme** (comme survie/impact) |
| DEC-4 | B4 intensite : nouvelle echelle | **(a) normalisation par le max GLOBAL de la periode**, valeurs brutes au tooltip |
| DEC-5 | Data morte backend (tableau lot D) | **Rien n'est branche** (D1 heatmap jour×heure abandonne — fun-fact bruite) ; TOUT part en liste cleanup (D2) |
| DEC-6 | B7 reference « vs historique » (Escouade) | **(a) tous les matchs du main, toutes compositions** — une vraie baseline |
| DEC-7 | Sort du stack `squad_service_v2` | **HORS PERIMETRE** : chantier dedie ulterieur (inchange ; E1 supprime n'y change rien) |
| DEC-8 | Outillage badges conserve apres le chantier ? | **Garder** : outil de revue reutilisable, inerte quand le manifest est vide |
| DEC-9 | Items optionnels E2/E3/F6/F7b | **Sans objet** : tous supprimes par la revision 2026-07-23 |

---

## Lot A — Outillage de surlignage (prerequis a tout le reste) — ~0,5 j

Mecanisme : un manifest central de revue + un badge accole aux titres de charts.
Cycle de vie d'une entree : ajoutee par ce chantier → l'utilisateur verifie a
l'ecran → l'entree est RETIREE du manifest (commit de cloture de la tournee).
Manifest vide = aucun badge rendu (mecanisme inerte, DEC-8).

- [ ] A1 `apps/web/src/lib/review/chart-review.ts` :
      `type ChartReviewStatus = 'verify' | 'new' | 'removal'` ;
      `interface ChartReview { status: ChartReviewStatus; note: string }` ;
      `const CHART_REVIEW: Record<string, ChartReview>` (cle = identifiant stable
      du chart, ex. `timeseries.kda_density`, `squad.synergy_radar`) ;
      helper `chartReview(key: string): ChartReview | undefined`.
- [ ] A2 `apps/web/src/components/charts/ReviewBadge.tsx` : badge compact
      (libelle + tooltip note). Couleurs par tokens : `verify` → `warning`,
      `new` → `info`, `removal` → `destructive` (aucun hex — skill color-tokens).
      Libelles i18n FR **et** EN (« A verifier »/« To verify », « Nouveau »/« New »,
      « Suppression ? »/« Remove? ») dans `lib/review/i18n.ts`
      (`Record<Locale, T>`).
- [ ] A3 `ChartCard` : prop optionnelle `review?: ChartReview` → badge rendu dans
      la barre de titre (a cote du `title`). Les surfaces non-ChartCard (tables,
      `OutcomeSequenceTape`, tuiles KPI) posent `<ReviewBadge>` directement.
- [ ] A4 Test vitest leger : `chartReview()` retourne l'entree, `ReviewBadge`
      rend le libelle FR/EN selon locale.

**Gate A** : `make check-types && make test-web` verts ; un badge de demonstration
visible sur un chart puis retire.

## Lot B — Corrections de rendu/donnee (chaque item pose un badge `verify`) — ~1,5 j

Chaque item commence par une verification sur pieces (le code a pu bouger depuis
la revue du 2026-07-12 — references ci-dessous a re-confirmer).

- [ ] B1 **Duree de vie reelle** (grave — constat valide utilisateur) :
      (1) verifier que `StatsMatchRow.AvgLifeSeconds` (`legacymatch/types.go:102`)
      est bien charge par la requete source et son taux de remplissage ;
      (2) backend : exposer `avg_life_seconds` dans `TimeseriesMatchRow`
      (`domain/timeseries.go`, `buildMatchRows`) et baser `buildLifeBuckets`
      dessus, fallback documente vers le proxy `time_played/(deaths+1)` si NULL
      (logge `slog.DebugContext`, jamais silencieux) ;
      (3) front : `TimeseriesAvgLifeTrend` + histogramme consomment le nouveau
      champ. Badges `verify` sur les deux charts.
- [ ] B2 **Rendement combat** (grave — constat valide utilisateur) :
      monter le composant orphelin `TimeseriesCombatYield`
      (`useCombatYieldHistory`, OC/DR calcules au sync) dans l'onglet Progression,
      selon DEC-2 a cote de `TimeseriesEfficiency`. Badges : `new` sur Combat
      Yield, `removal` sur Efficiency (note : « recalcul brut, le Combat Yield
      normalise est la version canonique — garder lequel ? »). Verifier au
      passage que l'endpoint consomme bien les champs sync
      (`OffensiveConversion`/`DefensiveResistance`).
- [ ] B3 **Radar synergie — axe Score a zero** (axe vise par l'utilisateur) :
      (1) diagnostiquer : valeurs `personal_score` reelles des coequipiers vs
      seuil `350×n` (`synergyRadarThresholds`) — trancher donnee manquante vs
      seuil ecrasant ; (2) recalibrer selon DEC-3 (P80 de l'historique, comme
      survie/impact) ; (3) tooltip : afficher la valeur BRUTE (deja dans le DTO,
      champ `Raw`) en plus du normalise. Badge `verify`.
- [ ] B4 **Heatmaps d'intensite** (Timeseries `intensity_rows` + Escouade
      `intensity_profile`) : appliquer DEC-4 — normalisation par le max GLOBAL de
      la periode (plus de max per-match qui detruit l'amplitude inter-matchs),
      valeurs brutes au tooltip. Point d'implementation : cote analysis
      (`NormalizeIntensityBuckets`) ou au niveau des builders — choisir l'endroit
      qui corrige LES DEUX pages sans dupliquer (≤ 2 copies). Badges `verify` ×2.
- [ ] B5 **Axes WR/MMR** : `TimeseriesSessionPerformance` — WR sur un axe %
      dedie [0,100] ; MMR sur son propre axe (offset ECharts), serie desactivable
      par legende. Verifier si `SquadSessionTimelineChart` partage le defaut
      (meme motif perf+WR+MMR) et corriger a l'identique si confirme. Badges
      `verify` (1 ou 2 selon constat).
- [~] B6 **Tri armes** : `SquadWeaponKillsChart` — tri DESC. COUVERT AILLEURS :
      deja traite dans un chantier dedie (constat utilisateur 2026-07-23).
      Verification rapide en passant sur la page (pas de code ici).
- [ ] B7 **Reference « vs historique »** (Escouade S1/S2) : appliquer DEC-6 —
      l'« historique » devient la baseline du main toutes compositions (au lieu
      du sur-ensemble de la meme composition). Backend :
      `LoadMapStatsForSquad` / son appelant dans `teammates_service.go`. Badges
      `verify` sur les deux charts (note expliquant le changement de reference).

**Gate B** : `make go-api-test` + `cd apps/go-api && go test ./...` verts ;
`make generate-types` rejoue (contrat modifie en B1) puis `make check-types` +
`make test-web` verts. Aucune ecriture DB touchee (lectures seules) → pas de
tests integration persist requis.

## Lot C — Redondances : surlignage SEULEMENT — ~0,1 j

Aucun changement de code hors manifest. Notes redigees en question ouverte
(l'utilisateur juge si le rendu/narratif differe assez).

- [ ] C1 Badge `verify` sur « Distribution FDA » ET « FDA (valeur) » (Summary) —
      note : « deux lectures de la meme metrique sur le meme onglet : distribution
      vs sequence temporelle. Les deux apportent-elles chacune quelque chose ? »
- [ ] C2 Badge `verify` sur « Progression CSR/LUSR » ET « Skill rank + Perf »
      (Progression) — note equivalente (courbe longue vs barres+perf).

**Gate C** : badges visibles, `make test-web` vert.

## Lot D — Data morte backend : consigner, rien brancher, rien supprimer — ~0,1 j

Champs calcules et servis par `timeseries_service` que le front ignore. DEC-5
(2026-07-23) : **rien n'est branche** — l'ex-D1 (heatmap jour×heure) est abandonne
(fun-fact bruite, petits effectifs par case). Tout part en liste cleanup pour un
chantier ulterieur.

| Champ | Contenu | Decision |
|---|---|---|
| `intensity_tab.heatmap_data` | heatmap jour×heure (quand tu joues/gagnes) | Cleanup ulterieur (ex-D1 abandonne 2026-07-23) |
| `intensity_tab.score_per_min_data` | score/min par periode | Cleanup ulterieur |
| `cumul_tab` | K/D cumule + rolling 5 | Cleanup ulterieur (recouvre les trends existants) |
| `outcomes_over_time` | V/D par periode | Cleanup ulterieur (recouvre Session Perf) |
| `distributions_tab.rolling_wr_buckets` | distribution WR glissant (fenetre 14) | Cleanup ulterieur |
| `distributions_tab.score_per_min_buckets` | distribution score/min | Cleanup ulterieur |
| `correlation_points` kills↔kd_ratio | scatter jamais affiche | Cleanup ulterieur |

- [ ] D2 Consigner le tableau ci-dessus dans la section Decouvertes de ce plan +
      une ligne thought_log, comme intrant du futur chantier cleanup.

**Gate D** : liste consignee dans Decouvertes + thought_log (aucun code).

## Lot E — SUPPRIME (revision 2026-07-23)

Rehabilitation du stack V2 abandonnee : E1 (form score LOWESS) = metrique
synthetique opaque pour l'utilisateur, recouvrant les trends de perf existants —
meme famille de probleme que mu/sigma (valeur non reliee au vecu de jeu).
E2/E3 etaient deja differes. Le stack `squad_service_v2` reste INTACT ; son sort
= DEC-7, chantier dedie ulterieur.

## Lot F — Dominance flags sur la bande d'outcomes (redefini 2026-07-23) — ~0,5 j

Seul survivant des « nouveaux angles », sous une forme differente : PAS de compteur
agrege (les flags remontada/debandade/contre-remontada ne sont jamais produits sans
timeline de score, cf. `internal/analysis/comeback.go:218` — un agregat de periode
serait silencieusement fausse par la couverture). Le flag s'affiche la ou il est
fiable : SUR le match, dans la `OutcomeSequenceTape` deja rendue sur Timeseries
(Summary) et Escouade. L'absence de marqueur = match ordinaire OU timeline absente :
aucun mensonge, le vocabulaire (badges Career/MatchView, colonne Dominance Explorer)
est deja connu de l'utilisateur.

Reutilisation maximale (verifie sur pieces le 2026-07-23) : cles i18n
`narrative.dominance.*` (FR+EN, manifests existants), tokens couleur
`narrative-domination|humiliation|remontada|debandade|contre-remontada`
(`ExplorerMatchesTable.tsx:222-235` — memes couleurs que la colonne Explorer).

- [ ] F1a Backend : exposer `dominance_flag` dans les rows qui alimentent les deux
      tapes (verif sur pieces des DTOs exacts : `TimeseriesMatchRow` +
      la reponse `teammates` qui nourrit la tape Escouade). Zero nouveau calcul —
      champ deja persiste dans `player_match_enrichment` (lecture via la vue/le
      chemin de lecture existant du service, pas de SQL nouveau si le champ est
      deja charge par `StatsMatchRow`).
- [ ] F1b Modele front : `OutcomePoint` gagne un champ optionnel
      `dominance?: 1|2|3|4|5` (`outcomeSequence.ts`) ; `toRuns` le propage tel
      quel. Champ ABSENT chez les autres consommateurs de la tape → leur rendu
      reste strictement identique (meme exigence de non-regression que
      `onMatchClick`).
- [ ] F1c Rendu (`OutcomeSequenceTape.renderItem`) : pour chaque match d'un run
      porteur d'un flag != none, dessiner un losange ~7 px centre sur la cellule
      du match, A L'INTERIEUR de la bande (yCenter — ne touche pas aux brackets
      de streaks au-dessus/en-dessous), rempli du token `narrative-*` du flag,
      lisere 1 px `CHART_BG` pour garantir la separation avec la couleur
      d'outcome dans les deux themes. Seuil de densite : ne rendre le losange que
      si la largeur par match >= 6 px (bande dense → le tooltip porte l'info).
- [ ] F1d Tooltip : suffixer la ligne du match avec le libelle du flag
      (« · Aquarius (Slayer) — Remontada »), cles `narrative.dominance.*`
      existantes. Verifier que ces cles sont dans un manifest charge par les DEUX
      pages (sinon les referencer depuis le manifest commun — pas de duplication
      de strings).
- [ ] F1e Aucune legende ajoutee (vocabulaire connu, marqueurs rares). Aucune
      couleur nouvelle, aucun hex.
- [ ] F1f Tests : `outcomeSequence.ts` (propagation du champ dans `toRuns`) +
      test de rendu leger (marqueur present si flag et largeur suffisante, absent
      sinon ; consommateur sans le champ → option identique a l'actuelle).
- [ ] F1g Badges review : `new` sur les deux tapes (note : « dominance flags par
      match — memes couleurs que la colonne Dominance de l'Explorer »).

**Gate F** : `make generate-types` (si contrat OpenAPI modifie en F1a) ;
`make go-api-test` + `cd apps/go-api && go test ./...` si Go touche ;
`make check-types` + `make test-web` verts ; verification visuelle des deux tapes
avec badge `new`.

## Cloture du chantier

- [ ] Z1 Tournee visuelle AVEC l'utilisateur (ou par lui) : les deux pages, badge
      par badge ; chaque item statue → entree retiree du manifest dans un commit
      de cloture. Les « Suppression ? » valides partent dans la liste cleanup.
- [ ] Z2 Skill `delivery-checklist` ; entree thought_log (statut Complete) ;
      demander validation avant tout commit ; rappel : push `main` = deploiement
      prod → prevenir l'utilisateur.

**Gate final** :
```bash
make go-api-lint
make go-api-test
cd apps/go-api && go test ./...
make generate-types && make check-types
make test-web
```

## Protocole de reprise de session

Lire ce fichier (statuts) + la derniere entree thought_log mentionnant « revue
analytique ». Reprendre a la premiere case non statuee de la premiere phase non
close (une phase est close = items statues ET gate passe). Le manifest
`chart-review.ts` reflete l'etat de la tournee visuelle (entrees restantes = a
verifier).

## Decouvertes hors perimetre (a consigner, ne pas traiter)

- Liste cleanup ulterieur (DEC-5 2026-07-23 ; a completer par la tournee Z1) :
  - `intensity_tab.heatmap_data` (heatmap jour×heure — ex-D1 abandonne)
  - `intensity_tab.score_per_min_data`
  - `cumul_tab` (K/D cumule + rolling 5)
  - `outcomes_over_time`
  - `distributions_tab.rolling_wr_buckets`
  - `distributions_tab.score_per_min_buckets`
  - `correlation_points` kills↔kd_ratio
- Candidat futur chantier Synthese (consigne 2026-07-23) : **Solo vs escouade**
  (ex-F5) — baseline KDA/WR/perf du main avec vs sans escouade (flag
  `is_with_friends`), barres groupees avec effectifs n + phrase narrative.
  Emplacement pressenti : page Synthese (insight de profil « toi avec/sans
  escouade ») — ni Timeseries (objet = temps) ni Escouade (cadree sur les matchs
  joues ensemble). Noter que B7 (baseline toutes compositions) porte deja une
  partie de cette information cote Escouade.
- Autres : (vide)
