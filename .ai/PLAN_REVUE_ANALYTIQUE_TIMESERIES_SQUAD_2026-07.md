# Plan — Revue analytique Timeseries & Escouade (corrections, surlignage, nouveaux angles)

**Date** : 2026-07-12
**Branche Git** : `feat/analytics-review-ts-squad` (une branche, un commit par lot)
**Statut** : En attente validation (DEC-1..9 a trancher — voir tableau)
**Effort estime** : 4 a 7 j selon les lots retenus (detail par lot)
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

**Dependance** : le chantier momentum (`.ai/PLAN_MATCHVIEW_MOMENTUM_2026-07.md`)
introduit le patron « barres divergentes autour de zero ». F4 et F7 le reutilisent :
a la 2e utilisation, extraire un wrapper generique `DivergingBarChart` dans
`components/charts/` (regle ≤ 2 copies) — l'extraction se fait dans le PREMIER des
deux chantiers qui atteint la 2e occurrence.

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

---

## Decisions a trancher AVANT execution (DEC)

| # | Question | Options | Recommandation |
|---|---|---|---|
| DEC-1 | Lots retenus | A+B minimum ; C/D/E/F a la carte | Tout sauf E2/E3 (voir DEC-9) |
| DEC-2 | B2 rendement : le vrai Combat Yield remplace `TimeseriesEfficiency` ou s'affiche a cote ? | remplacer / cote a cote | Cote a cote + badge « Suppression ? » sur l'ancien (l'utilisateur tranche a l'ecran — coherent avec « rien supprimer ») |
| DEC-3 | B3 radar : recalibrage des seuils en P80 (comme survie/impact) | oui / seuils fixes revises | Oui, P80 uniforme |
| DEC-4 | B4 intensite : nouvelle echelle | (a) normalisation par le max GLOBAL de la periode (b) valeurs absolues + rampe `heatmap-freq-*` | (a), avec valeurs brutes au tooltip |
| DEC-5 | Data morte backend : brancher / marquer pour suppression ulterieure (tableau lot D) | par item | Brancher la heatmap jour×heure ; le reste → liste cleanup |
| DEC-6 | B7 reference « vs historique » (Escouade) | (a) tous les matchs du main, toutes compositions (b) statu quo + note explicative | (a) — une vraie baseline |
| DEC-7 | Sort du stack `squad_service_v2` (garder/substituer/supprimer) | — | HORS PERIMETRE : a instruire dans un chantier dedie apres E1 |
| DEC-8 | Outillage badges conserve apres le chantier ? | garder (manifest vide) / retirer | Garder : outil de revue reutilisable, inerte quand le manifest est vide |
| DEC-9 | Items optionnels : E2 (impact ranking), E3 (cadence agregee), F6 (breakdown mode), F7b (bande sigma) | par item | Differer E2/E3 (densite des pages) ; inclure F6 ; F7b selon dispo de sigma (verif E/F7) |

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
- [ ] B6 **Tri armes** : `SquadWeaponKillsChart` — tri DESC (plus utilisees en
      premier). Badge `verify`.
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

## Lot D — Data morte backend : brancher ou lister, rien supprimer — ~0,5 j

Champs calcules et servis par `timeseries_service` que le front ignore. Decision
par item (DEC-5) ; dans CE chantier on ne fait que brancher ce qui est retenu ;
les suppressions validees sont CONSIGNEES (liste ci-dessous) pour un chantier
cleanup ulterieur.

| Champ | Contenu | Recommandation |
|---|---|---|
| `intensity_tab.heatmap_data` | heatmap jour×heure (quand tu joues/gagnes) | **BRANCHER** (D1) |
| `intensity_tab.score_per_min_data` | score/min par periode | Cleanup ulterieur |
| `cumul_tab` | K/D cumule + rolling 5 | Cleanup ulterieur (recouvre les trends existants) |
| `outcomes_over_time` | V/D par periode | Cleanup ulterieur (recouvre Session Perf) |
| `distributions_tab.rolling_wr_buckets` | distribution WR glissant (fenetre 14) | Cleanup ulterieur |
| `distributions_tab.score_per_min_buckets` | distribution score/min | Cleanup ulterieur |
| `correlation_points` kills↔kd_ratio | scatter jamais affiche | Cleanup ulterieur |

- [ ] D1 Brancher la heatmap jour×heure : nouveau chart « Activite par jour et
      heure » (wrapper `Heatmap2DChart` existant, rampe `heatmap-freq-*` — c'est
      une heatmap de FREQUENCE, sans jugement bien/mal), onglet Summary. i18n
      FR+EN. Badge `new`.
- [ ] D2 Consigner le tableau ci-dessus (avec les choix DEC-5 finaux) dans la
      section Decouvertes de ce plan + une ligne thought_log, comme intrant du
      futur chantier cleanup.

**Gate D** : `make check-types` + `make test-web` verts ; heatmap visible avec badge.

## Lot E — Rehabilitation PRUDENTE du stack V2 — ~0,5-1 j

Regles de prudence (remarque utilisateur : « reliquat, prudence ») :
- On ne bascule PAS d'endpoint : les pages restent sur `POST /pages/teammates`.
- On PORTE la logique (code Go copie/adapte AVEC ses tests) dans le service
  `teammates` ; le code V2 d'origine n'est ni modifie ni supprime.
- Tout calcul porte est re-verifie sur pieces et couvert par un test unitaire
  AVANT branchement front (code jamais passe en prod = non fiable par defaut).

- [ ] E1 **Form score lisse (LOWESS)** : porter la logique de `BuildFormScore`
      (`squad_service_v2_timeline.go`) dans `teammates` → nouveau champ
      `form_score` de `TeammatesPageResponse` + chart ligne lissee sur l'onglet
      Synergies (l'i18n `form_score_title` existe deja — verifier et reutiliser).
      Test unitaire porte. Badge `new`.
- [ ] E2 [OPTIONNEL — DEC-9, reco : differer] Impact ranking 8 roles
      (`BuildImpactRanking`) en complement du `SquadImpactScoreboard`.
- [ ] E3 [OPTIONNEL — DEC-9, reco : differer] Cadence agregee multi-matchs
      (recouvrement probable avec l'intensite — a instruire seulement apres B4).

**Gate E** : `make go-api-test` + suite Go complete verts ; `make generate-types`
+ `make check-types` + `make test-web` verts ; chart visible avec badge.

## Lot F — Nouveaux angles (badge `new` partout) — ~2-3 j selon DEC-9

Reponse a « je n'arrive pas a visualiser » : chaque item specifie Donnees /
Representation / Narratif / Emplacement. Penchant narratif assume (remarque
utilisateur) : chaque item porte une PHRASE de synthese FR+EN en plus (ou a la
place) d'un chart. Zero dependance a `expected_win_prob`.

- [ ] F1 **Comeback agrege** (Timeseries, Summary, pres de la tape) :
      - Donnees : `dominance_flag` (deja stocke par match dans
        `player_match_enrichment`) → propager dans `TimeseriesMatchRow`.
      - Representation : 3 tuiles KPI compactes « Remontadas / Debacles /
        Dominations » (compte + % de la periode). Pas de nouveau chart.
      - Narratif : « 4 remontadas sur la periode — tu convertis 31 % des matchs
        mal engages. » / EN equivalent.
- [ ] F2 **Echauffement / fatigue intra-session** (Timeseries, Progression) :
      - Donnees : `session_label` existant + `perf_score` par match → index du
        match dans sa session (calcul backend trivial, agregat par index).
      - Representation : barres « perf moyenne par position dans la session »
        (x = 1er, 2e, … 10e+ match ; y = perf moyenne ; effectif n en tooltip),
        colorees `perf-tier-*`.
      - Narratif : reutiliser la logique `detectSessionFatigue`
        (`analysis/patterns/behavioral.go`) — « au-dela du 6e match d'une
        session, ta perf chute de 12 % en moyenne ».
- [ ] F3 **Premier sang** (Timeseries, Progression, pres de la distribution des
      timings existante) :
      - Donnees : ATTENTION, les `first_events` actuels = 1er kill DU JOUEUR vs
        sa 1re mort. Le taux « premier sang du match » demande le premier kill
        toutes equipes → agregat backend nouveau sur `highlight_events`
        (kills synthetises inclus pour H5, cf. `applyKVSynthesisIfNeeded`).
      - Representation : 2 tuiles KPI (« Premier sang obtenu : X % des matchs » /
        « concede : Y % ») + barres groupees « winrate avec premier sang obtenu
        vs concede ».
      - Narratif : « Quand ton equipe prend le premier sang, vous gagnez 64 %
        des matchs (contre 41 % sinon). »
- [ ] F4 **Part de contribution dans le temps** (Escouade, Contributions) :
      - Donnees : `performance_series` (kills/degats/perf par match × joueur,
        deja servi) → ecart du joueur a la moyenne de l'escouade, par match.
      - Representation : barres divergentes autour de zero par match (joueur
        selectionnable) — reutilise le patron momentum ; 2e occurrence →
        extraire `DivergingBarChart` (cf. Dependance en tete de plan). PAS d'aire
        empilee 100 % (masquerait l'amplitude — defaut qu'on eradique).
      - Narratif : « Sur les 8 derniers matchs, X a porte l'escouade 5 fois. »
- [ ] F5 **Solo vs escouade** (Escouade, Synergies) :
      - Donnees : flag `is_with_friends` + agregats KDA/WR/perf du main sur la
        meme periode, calcules en deux passes (avec / sans). Petit ajout backend
        cote `teammates` (baseline solo).
      - Representation : barres groupees 2 series (Solo / En escouade) sur
        3 metriques, effectifs n affiches.
      - Narratif : « En escouade, ton KDA gagne +0,4 et ton taux de victoire
        +9 points (n=62 vs n=141). »
- [ ] F6 [OPTIONNEL — DEC-9, reco : inclure] **Par categorie de mode**
      (Timeseries, Summary) : WR/KDA par `ByModeCategory`
      (`internal/analysis/breakdown/`, deja pret) — barres groupees. Badge `new`.
- [ ] F7 **Skill — donnees observees uniquement** (Timeseries, Progression) :
      - F7a : barres divergentes « points de rating gagnes/perdus par match »
        (`SkillRatingDelta`, deja stocke — a exposer dans `TimeseriesMatchRow`).
        Badge `new`.
      - F7b [OPTIONNEL — DEC-9] : bande d'incertitude mu±sigma sur
        `TimeseriesSkillProgression` — SEULEMENT si sigma est disponible dans
        `match_skill_rank_latest` sans nouveau calcul (verif sur pieces en tete
        d'item ; sinon differer).
      - Exclusion ferme : rien base sur `expected_win_prob` (remarque
        utilisateur : fiabilite douteuse — reste cantonne a sa cellule).

**Gate F** : memes gates que lot E + verification visuelle de chaque nouveaute
avec badge `new` a l'ecran.

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

- Liste cleanup ulterieur (alimentee par DEC-5 et la tournee Z1) : (vide)
- Autres : (vide)
