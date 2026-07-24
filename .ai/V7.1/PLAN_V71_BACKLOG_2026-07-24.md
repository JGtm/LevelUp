# PLAN — Backlog Notion « Pour la v7.1 » — 2026-07-24

> Statut : EN COURS. Exécution sous contrat du skill `plan-execution`.
> Mode : supervision (Fable pilote, agents Opus/Sonnet/Haiku exécutent).
> Source : Notion « Backlog LevelUp », section « Pour la v7.1 »
> (https://app.notion.com/p/39a7ae87e7a3809e8e03e4ffedcf5086).
> Échange avec l'utilisateur : questionnaires interactifs pour les décisions ;
> retours/infos écrits SOUS l'item Notion concerné avec une ligne « Réponse : » ;
> item traité = barré dans Notion.
> Branche : décision utilisateur (proposée : feat/v7.1-backlog). Push main = deploy prod
> → JAMAIS sans feu vert explicite.

## Inventaire des items (IDs stables)

| ID | Item Notion (résumé) | Nature |
|----|----------------------|--------|
| I1 | H5 : stat véhicules détruits + hijacks (« Vol à la tire » H5 / « Dépositaire » Infinite) ; KPI cards H5 Synthesis (coup au sol, charge, assassinats) à gauche du breakdown d'armes | Feature |
| I2 | Médias : déclarer la piste voix/jeu/autres, config PAR JOUEUR, manuel OU algo acoustique | Feature |
| I3 | Path to hero (Carrière) title-agnostic — voir PLAN_XP_CARRIERE_ESTIMEE (décision user) | Feature |
| I4 | Synthesis : légende « Outils de destruction » en bas, centrée | UI |
| I5 | Sessions : « Répartition des frags » % en légende + hauteurs alignées avec « Outils de destruction » (drawer compare déployé) | UI |
| I6 | Escouade : affichage % sur « Outils de destruction » (forme = décision user) | UI |
| I7 | Citations JGtm à zéro : lister (noms+descriptions) → tableau Notion avec colonne commentaires user ; diag mapping/récupération | Diag |
| I8 | Purge Docker au déploiement (saturation VPS du 23/07) | Ops |
| I9 | Escouade : axe Y « Taux de victoire session vs historique » affiche un nombre inconnu | Bug |
| I10 | Slug de la langue active absent de l'URL | Feature |
| I11 | Écart cumulé FDA (Timeseries + Sessions) : ajouter courbe « FDA attendu » | Feature |
| I12 | « Performance par carte — Session vs Historique » + « Taux de victoire session vs historique » : toujours pas en ordre chronologique | Bug |
| I13 | « Performance d'escouade par session » : historique NON exclusif à la compo exacte (fausse tous les historiques) | Bug |
| I14 | Enregistrement compo d'escouade non player-agnostic (doublons, membre manquant chez les autres joueurs) | Bug |
| I15 | Notifs : « PB » anglicisme + sweep complet des anglicismes FR | Fix |
| I16 | Vérifier que TOUS les tableaux sont triables par colonnes | Audit |
| I17 | [POSTPONED par l'user] premier frag/première mort normalisé — HORS PÉRIMÈTRE | — |
| I18 | Titres d'onglet navigateur absents/instables (« LevelUp » seul) | Bug |
| I19 | Option colonne « ouvrir sur HaloWaypoint » (2e colonne, logo thème clair/sombre — asset : C:\Users\Guillaume\Pictures\Screenpresso\2026-07-24_13h49_49.png à détourer) | Feature |
| I20 | Explorer « Sur XX matchs ensemble » : + écart de frags cumulé, donuts TdV, sparkline, FDA avec/contre (scope = décision user) | Feature |
| I21 | Élucider `perf="1"` dans l'URL Explorer (+ fix si reliquat) | Diag |
| I22 | [FINAL] What's new README FR/EN + changelogs FR/EN + notes de version in-app (depuis v7.0.0) | Docs |

## Vagues d'exécution

### V0 — Investigation (lecture seule, 11 agents parallèles) — EN COURS
- [ ] V0.1 (opus) I13+I14 bugs escouade compo — causes racines + fixes proposés
- [ ] V0.2 (sonnet) I9+I12 graphes escouade (axe Y, ordre chrono)
- [ ] V0.3 (sonnet) I7 citations JGtm à zéro (données + diag) — SEUL agent autorisé aux commandes go
- [ ] V0.4 (sonnet) I18+I21+I10 front/routing (titres onglet, perf=1, locale URL)
- [ ] V0.5 (sonnet) I15 sweep anglicismes
- [ ] V0.6 (sonnet) I16 audit tri tableaux
- [ ] V0.7 (sonnet) I1 dispo données H5 (véhicules, kill mechanics) — SANS commandes go
- [ ] V0.8 (sonnet) I2 design pistes audio médias
- [ ] V0.9 (sonnet) I19 plan colonne Waypoint
- [ ] V0.10 (sonnet) I11 plan courbe FDA attendu
- [ ] V0.11 (sonnet) I8 pipeline deploy + mesures VPS (lecture seule stricte)
Gate V0 : rapports reçus, vérifiés sur pièces (spot-check superviseur), plan V1+ affiné.

### V1 — Correctifs et UI courts (après décisions user)
I9, I12, I4, I5, I6 (forme décidée), I18, I21, I15. Découpage en agents à l'issue de V0.
Gate : tsc + vitest ciblés + lint ; suite complète en fin de lot (doctrine batch gates).

### V2 — Bugs escouade (I13, I14) — potentiellement le plus gros correctif
Gate : tests Go + front verts, scénario de repro utilisateur validé.

### V3 — Features (I1, I2, I10, I11, I19, I20 selon décisions, I3 selon décision)
Gate : par feature — tsc/vitest/go test ; -tags=integration si persist touché.

### V4 — Ops (I8) : modif pipeline deploy (repo) ; AUCUNE écriture VPS sans feu vert.

### V5 — [FINAL] I22 docs release — UNIQUEMENT quand tout le reste est statué.

## Décisions utilisateur
| Sujet | Décision | Date |
|-------|----------|------|
| I6 forme % Outils de destruction | Étiquettes % sur segments (valeurs brutes conservées, petits segments au tooltip) | 2026-07-24 |
| I3 Path to hero | EXÉCUTER le plan XP CARRIÈRE ESTIMÉE d'abord, puis refonte Path to hero dessus | 2026-07-24 |
| I20 scope Explorer | Écart de frags cumulé + donuts TdV ensemble/face à lui (PAS sparkline, PAS FDA avec/contre) | 2026-07-24 |
| Nom de branche | feat/v7.1-backlog | 2026-07-24 |

## Décisions superviseur (autonomie, consignées)
- I12 : cap 20 cartes conservé, sélection = top-20 par match_count, AFFICHAGE en ordre
  chronologique de première apparition (aligné heatmap). Suffixe « (n) » de l'axe Y (I9)
  expliqué par InfoTooltip (pattern standard) plutôt que supprimé.
- I13 : pool d'exclusivité = amis configurés en priorité, top teammates en repli ;
  `selected ⊆ composition` toujours forcé.
- I2 : v1 = rôles déclarés par piste (manuel > auto NNLS), rendition `voices` = voix∪autres
  (pas de 3e toggle lecteur, effort L différé) ; pas de ré-application rétroactive.
- I21 : aucun changement de code (comportement TanStack par défaut, round-trip sans perte) —
  réponse documentée dans Notion, item barré.
- I15 : lots A-E traités ; coaching_tips.toml = relecture éditoriale SÉPARÉE (noté Notion).
- I8 : modifs scripts/deploy.sh dans la branche ; unités systemd VPS = proposition Notion,
  AUCUNE écriture VPS sans feu vert.

## V0 — résultats (rapports complets dans les transcripts agents)
Tous reçus 2026-07-24. Points saillants : I13 = prédicat subset (3 maillons :
intersectSquadRowsByMatchID, briefing IntersectByMatchID, Q42 HAVING COUNT) ; I14 =
résolution par slug/gamertag au lieu de xuid + créateur legacy absent → backfill
append-only requis ; I7 = 4 familles de bugs (A pve_stat non câblé, B grenade_kills
omis, C collision medal_id 3169118333 driver/road_trip, D collision award carrier_killed) ;
I1 volet 2 DÉJÀ LIVRÉ (vérifier couverture backfill kill mechanics) ; I18 = double
mécanisme concurrent __root vs effets locaux, resolvePageTitle non locale-aware ;
I10 = mécanisme {-$lang} DÉJÀ LIVRÉ dormant (option A = l'activer) ; I8 = --keep-storage
no-op (déprécié) + cron.d mort (pas de daemon cron).

### V0 statuts
- [x] V0.1 à V0.11 : rapports reçus et exploités (spot-check superviseur au fil des vagues).

## Découvertes hors périmètre (règle 7 — noter, ne pas traiter)
- winRateVsHistoryChart.ts (non-bullet) = builder MORT (aucun caller) → suppression incluse
  dans le périmètre I12 (même famille, décision superviseur). FAIT (W1b). Reste la clé i18n
  orpheline charts.winRateVsHistoryTitle → à purger au gate V1 (superviseur).
- buildSoloMapBreakdown (timeseries_service_aggregations.go, page Stats solo) trie encore
  par MatchCount desc — même biais visuel potentiel que I12 mais AUTRE page, hors périmètre
  du signalement. À proposer à l'utilisateur en fin de lot.
- Couverture kill mechanics H5 mesurée (parquet 23/07) : 21277/24208 lignes (88 %),
  sommes assn=1539 gp=353 sb=1175 → cards alimentées ; ~12 % de matchs anciens sans
  re-sync (backfill optionnel).
- Incohérence libellé catégorie coach « lusr_tier_approach » front vs Go (relevé I15, lot D).
- coaching_tips.toml : ~80 entrées jargon esport — relecture éditoriale dédiée à cadrer.
- personal_score_awards : citation player_vs_everything décrit « gagner » mais compte des
  kills (incohérence de contenu, à trancher avec l'utilisateur via Notion).
- deploy.sh : down avant build (prod down si build échoue) — fix radical (build avant down)
  signalé mais hors périmètre I8.

## Journal
- [2026-07-24] Chantier ouvert. V0 lancée (11 agents). Questionnaire décisions envoyé.
- [2026-07-24] V0 close (11/11 rapports). Décisions user reçues (I6, I3, I20, branche).
  Branche feat/v7.1-backlog créée depuis main (f389de218). Lancement Vague 1 :
  W1a bugs escouade (opus), W1b graphes escouade (sonnet), W1c UI charts I4/I5/I6 (sonnet),
  W1d FDA attendu (sonnet), W1e titres d'onglet (sonnet). Agents SANS commandes go/npm ;
  gates par le superviseur en fin de vague.
- [2026-07-24] Vagues 1+2 CLOSES et committées : I13+I14 [x] (ea84e242f), I9+I12 [x]
  (59e705690), I7 code [x] (cd91858f5 — recalcul data à lancer ; road_trip/flag_defender
  [!] décision user), I11 [x] (5ba252be5), I18 [x] (36b9735e3), I4+I5+I6 [x] (bf21c7180 —
  vérif visuelle à faire), I19 [x] (ef80b0e53), I8 repo [x] (unités systemd VPS [!] feu
  vert user). Gates : tsc purgé 0 err ; vitest 2914 ; eslint 0 err/9 warn baseline+1 ;
  go build/vet/test OK ; intégration sync -p 1 103 s OK ; golangci-lint ratchet 0 issue.
  Corrections superviseur au gate : 3 mocks SquadContext (currentPlayerXuid),
  FeatureUnavailable (entrée capability), garde-rail waypoint (exclusion .test),
  gofmt 2 fichiers, emoji deploy.sh, allowlist shared_social (justifiée datée).
  Restent : I21 [x] (réponse Notion, barré), I17 [~] POSTPONED user, vagues 3-4, backfills
  data, passe visuelle, I22 [FINAL].
