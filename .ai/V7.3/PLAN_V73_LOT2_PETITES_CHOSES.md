# PLAN v7.3 — Lot 2 « petites choses » (section Notion, hors Replay 2D)

> Écrit le 2026-08-02 après reconnaissance (3 agents Explore) et 3 questionnaires
> utilisateur. La v7.3 reste ouverte (tag non posé) : ce lot s'y ajoute.
> **Exécution sous contrat du skill `plan-execution`** (ordre strict, une étape à la
> fois, gate passé avant l'étape suivante, zéro report d'étape exécutable).

## Objectif et critère de succès

Traiter tous les points non barrés de la section Notion « Pour la v7.3 » hors Replay 2D,
hors POSTPONED, hors décisions de report ci-dessous. Succès = chaque item de ce plan
statué (`[x]` fait / `[~]` couvert ailleurs avec référence / `[!]` non traité justifié),
gates verts, revue navigateur des changements visuels faite, section Notion mise à jour
(points barrés + notes), thought_log à jour.

**Branche** : `feat/v7.3-notion-lot2` depuis `main` (à créer au démarrage de
l'exécution — 1 branche, N commits). Effort global : moyen-lourd (~6 j-agent).

## Décisions utilisateur verrouillées (questionnaires du 2026-08-02)

| Sujet | Décision |
|---|---|
| Killsource | **Re-différé** (branche `feat/killsource-prod` vivante, handoff du 31/07 — ne pas perdre ; inclut la validation « précision Infinite exploitable ? ») |
| Artefacts ≥3 propositions | **Rendement & Résistance** (Dynamique) + **losanges séquence V/D** uniquement. FDA/intensité et Rythme des rencontres = correctifs directs |
| Tableaux d'historique | Colonne **Score personnel** maintenant ; colonne **Replay à la livraison du Replay 2D** (pas dans ce lot) |
| Suppression de vidéos | **Propriétaire + admin**, définitive, avec confirmation |
| i18n TOML/TS en double | **Supprimer le repli** : flag `MULTI_TITLE_API_ENABLED` invariant, fallbacks supprimés, `metricLabel` migré vers `useFieldLabel` |
| Note de perf modes objectifs | **Chantier isolé futur** (hors lot). Consigner : scission ranked par famille = bonne piste validée ; subtilité — ne pas récompenser l'absence de combat, mais un joueur écrasé peut quand même jouer l'objectif |
| Kills véhicules | **H5 maintenant** (classes déjà en base), Infinite au killsource. Exigence : le sous-niveau du sunburst doit distinguer **chaque véhicule**, pas un bucket unique |
| Sessions escouade | Règle unique **« matchs commencés ensemble »** (tous les sélectionnés au roster) partout en contexte escouade ; « composition exacte » devient une option désactivée par défaut ; échecs de chargement visibles |
| Précision par arme | **Question répondue, rien à implémenter** : H5 = données déjà en base (`weapon_accuracy`, tirs tirés/touchés par arme) ; Infinite = travail en cours (killsource), à valider si exploitable |
| Bonus (i) | Cible = graphe **« Frags / Morts » de la page Escouade** |
| Découvertes d'exécution | Délégué (message du 2026-08-02 en cours de lot) : triage et traitement **selon les recommandations de l'orchestrateur** — celles qui croisent un item du lot s'y traitent, les autres en passe dédiée post-lot avec recommandation par découverte |
| Dependabot au fil de l'eau | Délégué (même message) : traiter **au mieux jugé** — constat du 2026-08-02 : rien de neuf (alerte echarts = D3, PR #70 = D4, runs « Dependabot Updates » rouges = re-tentatives echarts sans objet car PR #49 fermée manuellement en juin) |

## Ordre d'exécution

Phase 0 en premier (les artefacts partent chez l'utilisateur, sa décision tourne en
parallèle), puis 1 → 2 → 3 (3.4/3.5 dès les choix d'artefacts rendus) → 4.
Une phase est close quand tous ses items sont statués ET son gate est vert.

Le volet D (branches Dependabot) vit sur des branches SÉPARÉES du lot : D1 et D2 sont
exécutables dès le démarrage (indépendants), D3 s'exécute APRÈS le merge du lot
(séquencement imposé, voir D3), D4 est une vérification de statut à la clôture.

---

## Phase 0 — Artefacts de propositions visuelles (gate utilisateur)

Charger les skills `dataviz` puis `artifact-design` AVANT d'écrire chaque page.
Données réalistes tirées des formes réelles des payloads (lire les builders), rendu
light/dark, chaque proposition faisable en ECharts avec les tokens sémantiques du repo.

- [x] 0.1 Artefact **Rendement & Résistance** (page Dynamique escouade) — publié le
      2026-08-02 (4 propositions : A axe retourné log, B indice de vies, C écart à la
      frontière élite par piste, D grille de session ; light/dark, données réelles) :
      https://claude.ai/code/artifact/2c1ae601-d755-40f9-b487-e6784dc9f33f
- [x] 0.2 Artefact **marqueurs de dominance** de la séquence V/D — publié le
      2026-08-02 (4 propositions A/B/C/D, rendus 20/60/150/400 matchs + bac d'essai
      12→600, clair/sombre) :
      https://claude.ai/code/artifact/cf902165-3994-457f-9715-1e155ee1335b
      (`components/charts/OutcomeSequenceTape.tsx`, losanges 7x7 px en dur,
      invisibles sous 6 px/match) : >= 3 formes alternatives, avec le comportement
      aux faibles largeurs traité dans chaque proposition.

**Gate 0** : 2 artefacts publiés (privés), URLs remises à l'utilisateur, décisions
consignées dans ce fichier (section Décisions d'artefacts en bas). L'implémentation
(3.4/3.5) ne démarre pas sans décision.

## Phase 1 — Bugs

- [ ] 1.1 (code livré le 2026-08-02, gates re-vérifiés orchestrateur : vitest 3237 ok,
      tsc ok — RESTE revue navigateur au gate 1 avant de cocher. Cause : triple maillon
      garde composition-seule / pas de useFollowLatestSession en escouade /
      followLatest éteint par les chemins techniques → fix `lastAnchoredLatestSession`
      persisté + règle « session jamais ancrée → ré-ancrage ». Reproduction 31/07
      impossible en local, DBs au 23/07 — validé sur la dernière session locale.)
      **Autosnap escouade** : la page ne s'est pas ouverte sur la session du 31/07.
      Diag sur pièces : `features/squad/SquadLayout.tsx:434-461` +
      `features/squad/squadPending.ts` (`decideCompositionReanchor`) + persistance
      `picked_sessions` par LABEL et `isAutoSnappingToLatest`
      (`stores/createFilterStore.ts`, clé `levelup-squad-filter-v1`). Reproduire avec
      les données réelles de la session du 31/07, corriger, tests vitest sur
      `decideCompositionReanchor`, revue navigateur.
- [ ] 1.2 (code livré le 2026-08-02, gates re-vérifiés orchestrateur — RESTE revue
      navigateur au gate 1. Cause du 11/8/6/5 : 4 populations empilées ; livré :
      `match_count` roster dans `composition_sessions` + `mergeSessionCounts` front,
      `filter_exact_composition` optionnel défaut off (contrat +49 lignes, toggle
      FR/EN persisté, query key), heatmap sans re-filtrage privé, collecteur
      `data_issues` slog.ErrorContext + bandeau UI FR/EN.)
      **Compteurs de sessions unifiés** (le 11/8/6/5) :
      `internal/service/teammates/teammates_service.go:162-301`. Règle canonique
      « commencés ensemble » = intersection roster (population B). (a) Le compteur de
      la liste des sessions en contexte escouade affiche le compte « ensemble » ;
      (b) `filterExactComposition` devient un paramètre API optionnel, défaut off
      (contrat + toggle UI FR/EN) ; (c) le heatmap perd son re-filtrage privé
      (`teammates_squad_charts_sessions_maps.go:300-333`) et consomme la même
      population ; (d) tout échec de chargement best-effort (LoadMainTeamParticipants,
      LoadFor) cesse d'être silencieux : `slog.ErrorContext` + état d'erreur visible
      côté UI (fin des chiffres non reproductibles). Tests service (mock port) +
      httptest si contrat touché + vitest + revue navigateur.
- [ ] 1.3 (code livré le 2026-08-02, gates re-vérifiés orchestrateur — RESTE revue
      navigateur au gate 1. Double cause prouvée : AT TIME ZONE 'UTC' post-COALESCE
      qui annulait le fuseau de session + COALESCE mal parenthésé décalant le repli ;
      fix = fragment canonique `StartTimeCanonicalSQL`, 3 tests DuckDB :memory:
      Paris été/hiver + tooltip 3 lignes i18n FR/EN, 5 tests web.)
      **Rythme des rencontres — heures fausses** :
      `internal/platform/duckdb/queries_relations_moments.go:41-42` — le
      `AT TIME ZONE 'UTC'` explicite annule le fuseau de session et livre des heures
      UTC sous l'alias `hour_local`. Corriger pour rendre l'heure dans
      `cfg.UserTimezone` en respectant le fragment canonique (règle CLAUDE.md n°8 :
      COALESCE conservé). Test DuckDB `:memory:` avec fuseau fixé non-UTC.
      + **Tooltip refait** (`features/palmares/RelationsMomentsHeatmap.tsx:88-100`) :
      contenu lisible (joueur, jour/heure, n matchs), i18n FR/EN.
- [ ] 1.4 (code livré le 2026-08-02, gates re-vérifiés orchestrateur — RESTE revue
      navigateur au gate 1. Normalisation canonique `commendation_category.go`
      (7 clés stables, patron medal_category) branchée aux 3 frontières H5 +
      service + analysis ; audit (c) Infinite : trou avéré (libellés FR en dur du
      seed) corrigé par normalisation à la lecture, chemin sync/citations.go:215
      confirmé sans consommateur aval ; 7 clés citations.category.* FR/EN + 
      labels.ts patron médailles ; 7 tests Go + 4 tests web parité.)
      **Catégories de citations en anglais** : (a) H5 — normaliser les catégories
      brutes de l'API (`"MULTIPLAYER"`, `"GAME MODE"`...) en clés stables côté Go
      (`platform/duckdb/halo5/halo5_commendation_defs.go:68`,
      `service/.../commendation_totals.go:69`) ; (b) clés
      `citations.category.<key>` FR/EN dans
      `apps/web/src/lib/i18n/manifests/citations.toml` + résolution dans
      `features/citations/CitationsView.tsx:49` (patron médailles,
      `medal_category_table.go` + `medals.toml`) ; (c) audit Infinite :
      `citation_mappings.category` (`internal/sync/citations.go:215`) — vérifier que
      les clés servies sont stables et couvertes par le manifeste. Tests Go
      (normalisation) + parité typée `Record<Locale, T>`.
- [ ] 1.5 (CONTEXTE UTILISATEUR 2026-08-02, en cours d'exécution : régressions
      RÉCURRENTES sur les likes médias, « en général à cause de l'enregistrement en
      BDD qui ne se fait pas de manière instantanée » — la fenêtre d'écriture
      asynchrone SharedSocialPersister (Collect→Persist + CHECKPOINT différé) entre
      la réponse HTTP et la visibilité en lecture `_latest` est le suspect
      prioritaire, à croiser avec les pistes 3/4. Exigence ajoutée : le correctif
      doit CADRER cette classe de régression — sémantique lecture-après-écriture
      explicite + test d'intégration qui reproduit la fenêtre, pas un fix du seul
      symptôme.)
      (code livré le 2026-08-02 — RESTE revue navigateur au gate 1. Diagnostic :
      piste 1 écartée sur pièces mais garde livrée ; piste 2 PROUVÉE (chemin absolu
      vs `file_path` relatif forward-slash 219/219 → UPDATE 0 ligne → 404) ; piste 3
      PROUVÉE (réponse = chemin stocké, cache indexé par URL servable → onSuccess
      muet) ; cause racine de la RÉCURRENCE = `tx.Commit()` nu, like en WAL jusqu'à
      5 min, effacé par tout redémarrage/déploiement — `CommitWithCheckpoint`
      existait, documenté pour les likes, jamais appelé. Correctifs : conversion
      URL unifiée 3 endpoints + garde-rail, écho du file_path reçu, 401
      `like_requires_session` + toast FR/EN, CommitWithCheckpoint avec garantie
      commentée, media.go 596→485 L. 11 tests Go + 2 web dont
      `SurvivesWALLoss` au pouvoir discriminant prouvé rouge-sans/vert-avec.)
      **Likes médias cassés** : diagnostic hiérarchisé sur pièces AVANT tout code,
      dans cet ordre : (1) session absente => `liker_slug` vide => like fantôme sans
      401 (`api/handlers/media.go:312-372`) ; (2) `urlToFilePath` désaligné après un
      changement de préfixe d'URL média ; (3) forme de cache divergente
      (`features/media/queries.ts:140-175`) ; (4) 503 `ErrDBLocked` masqué par
      l'optimistic update + `refetchType: 'none'`. Corriger la cause prouvée +
      garde anti-silence : sans session le like doit échouer visiblement (401 gaté ou
      erreur UI explicite — décision technique sur pièces, contrat mis à jour si
      besoin). Tests + `go test -tags=integration -p 1` (shared_social touché).
- [ ] 1.6 (code livré le 2026-08-02 — RESTE revue navigateur au gate 1 (vignettes
      Home démo). `image_url` ajouté aux 4 items des DEUX fixtures vers
      `/static/prestige-assets/Objectives-badges/*.png` (assets existants du repo),
      garde-rail `TestGetChallenges_DemoMode_ImagesServable` (clé + fichier
      existant), vérifié en démo réelle :8123 — 4 URLs en 200 image/png, parité
      FR/EN.)
      **Démo : images des défis absentes** : les fixtures
      `internal/service/demo_fixtures/challenges.json` + `challenges.en.json` n'ont
      pas de clé `image_url` (le mode démo bypasse le cache DB —
      `home_service_battlepass.go:88-89`). Ajouter `image_url` aux items des DEUX
      fixtures vers un asset servi par `/static/` (vérifier qu'un asset embarqué
      existe, sinon en ajouter un). Vérification en démo locale.

**Gate 1** : `cd apps/go-api && go test ./...` exit 0 sur les paquets touchés ;
`go test -tags=integration -p 1 ./...` exit 0 (1.5 touche shared_social) ;
`make check-types` ; `make test-web` ; contrat régénéré si routes/params modifiés
(`openapi` + `make generate-types`, 0 chemin perdu) ; revue navigateur des correctifs
1.1, 1.2, 1.3, 1.6.

## Phase 2 — Petites UI

- [ ] 2.1 **Médailles/citations à peu d'éléments** :
      `features/match-view/MatchSummaryMedalsAndCitations.tsx` — la co-localisation
      2 colonnes existe déjà ; le problème est le plancher `CARD_HEIGHT = 280`.
      Hauteur adaptative quand peu d'éléments (les deux cartes se compactent sur la
      rangée). Revue navigateur sur un match pauvre en médailles.
- [ ] 2.2 **(i) Bonus** sur le graphe « Frags / Morts » de la page Escouade
      (`features/squad/charts/squadPerformanceLineCharts.ts`, série `Bonus` masquée
      par défaut) : InfoTooltip FR/EN expliquant Bonus = assistances/3 (ADR 0006).
      Si le même composant sert d'autres pages, elles en héritent (pas de travail
      supplémentaire hors périmètre).
- [ ] 2.3 **FDA gap + intensité — pédagogie sans refonte** (choix utilisateur : pas
      d'artefact) : (a) InfoTooltip pédagogique sur les 2 instances du FDA gap
      (`features/timeseries/TimeseriesFdaGapTrend.tsx`,
      `features/session-detail/SessionFdaGapCumulative.tsx`) — il n'en existe aucun
      aujourd'hui ; (b) réécrire celui du profil d'intensité
      (`SessionIntensityProfile.tsx:62-71`) en langage non technique ; (c) affordance
      de survol : symboles visibles au hover (le rendu canvas sans aucun point
      survolable est ce qui donne l'impression d'une image figée) ; (d) profil
      d'intensité escouade : ajouter la courbe agrégée de l'équipe quand >= 3 joueurs
      sélectionnés (`features/squad/charts/squadIntensityProfileChart.ts`).
      Validation en revue navigateur avec l'utilisateur.
- [ ] 2.4 **Tableaux — colonne Rang en image + tooltips d'en-tête** : (a) ajouter
      `skill_rank_image_url` au contrat `ExplorerMatchRow` servi par le backend via
      l'adaptateur d'assets du titre (précédent : `RecentMatchItem.skill_rank_image_url`
      pour Home) — dégradation propre : champ null => texte localisé actuel (H5) ;
      assets déjà présents (`static/ranks/halo_infinite/unranked_0..9.png` + CSR) ;
      (b) réduction des largeurs de colonnes : Explorer d'abord, puis passe sur
      Escouade, Timeseries, Session, Carrière ; (c) tooltip d'en-tête porté par le
      LABEL : étendre `InfoTooltip` (`components/ui/info-tooltip.tsx`) avec un trigger
      non-bouton (piège documenté bouton-dans-bouton, `lib/table/columnMeta.tsx:11-15`),
      retirer les icônes ⓘ des en-têtes. Contrat + `make generate-types`.
- [ ] 2.5 **Colonne Score personnel** dans les tableaux d'historique : vérifier
      d'abord quel(s) composant(s) constituent « l'historique » (Explorer +
      MatchHistoryPage — confirmer par grep avant code) ; champ contrat à ajouter si
      absent d'`ExplorerMatchRow` (mutualiser avec la modification 2.4a) ; tri
      TanStack ; en-tête FR/EN.

**Gate 2** : `make check-types` ; `make test-web` ; contrat régénéré (2.4/2.5) ;
`make go-api-test` si le backend a bougé ; revue navigateur de chaque item.

## Phase 3 — Features et unifications

- [ ] 3.1 **Suppression de médias** (propriétaire + admin, définitive, confirmation) :
      (a) design sur pièces AVANT code : modèle de stockage réel (fichiers disque +
      `media_likes` append-only sur shared_social) => sémantique de suppression des
      likes du média supprimé compatible ADR 0022/0026 (pas d'UPDATE/DELETE sur
      tables critiques — événements ou orphelins invisibles, décision consignée ici) ;
      (b) gate admin : réutiliser le mécanisme d'admin existant s'il y en a un
      (vérifier ce qui a été livré par `feat/admin-retours-diag`) — s'il n'existe
      pas, livrer propriétaire seul et statuer la part admin `[!]` avec justification ;
      (c) endpoint DELETE gaté auth + ownership (ratchet bare_routes), handler sans
      logique métier, service + port ; (d) UI : action supprimer dans le visualiseur
      média + modale de confirmation, invalidation des query keys `mediaBase`.
      Tests httptest + service + `go test -tags=integration -p 1`. Contrat openapi.
- [ ] 3.2 **Kills véhicules H5** : VÉRIFICATION PRÉALABLE BLOQUANTE — confirmer que
      les kills véhicule H5 portent un identifiant PAR véhicule (registre) et non le
      seul sentinel `VehicleWeaponID = 2` ; si tout s'écrase sur le sentinel, STOP et
      rapporter à l'utilisateur avant d'implémenter (son exigence : le sous-niveau
      distingue chaque véhicule). Si OK : sortir `vehicle`/`turret` de
      `nonCombatFragClasses` (`internal/domain/frag_distribution.go:42-49`) pour le
      breakdown H5, libellés FR/EN via TOML mappings H5 si nouvelles clés, gardé par
      les données (pas de branche slug). Tests analysis/domain.
- [ ] 3.3 **i18n source unique** : (a) vérifier `MULTI_TITLE_API_ENABLED` actif dans
      TOUS les environnements (compose prod + démo, `.env.example`, CI, dev) puis le
      rendre invariant (retrait du flag, ou toujours-on documenté kill-switch daté —
      règle 11) ; (b) supprimer `features/home/fallback.i18n.ts`,
      `features/media/fallback.i18n.ts`, `features/prestige/fallback.i18n.ts` et les
      dictionnaires de `lib/i18n/metricLabel.ts` ; (c) migrer les consommateurs vers
      `useFieldLabel`/`useAssetLabel`/`useOutcomeLabel` (remonter dans le rendu les
      appels actuellement hors composants) en gardant `humanizeMetricKey` comme
      unique repli d'affichage ; (d) garde-rail (règle 6) : test grep interdisant les
      dictionnaires de libellés de field-keys hors TOML. Hors périmètre :
      `lib/skillTiers.ts` (couplage Go<->TS distinct, garde-rail déjà existant).
- [ ] 3.4 **Implémentation du choix d'artefact Rendement & Résistance** (gate 0.1).
- [ ] 3.5 **Implémentation du choix d'artefact dominance V/D** (gate 0.2) — mêmes
      surfaces que les losanges actuels (Home, Timeseries, Relations, showcase Lab).

**Gate 3** : suite Go complète `go test ./...` + `-tags=integration -p 1` exit 0 ;
`make go-api-lint` 0 issue nouvelle ; `make check-types` ; `make test-web` ; contrat +
types front régénérés ; revue navigateur 3.1/3.2/3.4/3.5.

## Phase 4 — Ops prod et clôture

- [ ] 4.1 **Re-mesure lot K** (fetch films hors verrou, déployé le 26/07 mais jamais
      mesuré en conditions réelles — 0 match ingéré alors) : la session du 31/07
      fournit enfin la mesure. Compter dans les logs VPS les avertissements
      « writer RW tenu > seuil » par cycle d'auto-sync depuis le 31/07 (avant :
      3-5/cycle ; attendu : ~0). **Préavis utilisateur avant toute op VPS.**
- [ ] 4.2 **Cut de snapshot prod** : vérifier si le snapshot de lecture a été recoupé
      depuis le correctif G1 (sinon le repli lecture live reste actif) ; recouper si
      nécessaire, avec préavis.
- [ ] 4.3 **MAJ Notion** : barrer chaque point traité avec commit + notes sous les
      points ; consigner les réponses aux questions posées dans la section
      (précision par arme = H5 déjà couvert / Infinite au killsource ; « pourquoi
      cette représentation est une image » = rendu canvas ECharts sans symbole ni
      aide, corrigé en 2.3) ; callout Suivi du lot 2.
- [ ] 4.4 **Clôture** : delivery-checklist (skill), tous les items de ce plan statués,
      entrée thought_log, découvertes consignées (section ci-dessous).
- [ ] 4.5 Rappel côté utilisateur : poser le **tag v7.3.0** quand tu considères la
      v7.3 close (déclenche notification de release + « Quoi de neuf »). Le Replay 2D
      étant ton chantier, ce lot ne conditionne pas le tag.

**Gate 4** : `make gate-push` vert avant proposition de merge (merge main = deploy
prod auto — prévenir l'utilisateur).

## Volet D — Branches Dependabot (orchestration, branches séparées du lot)

Détail des étapes : `.ai/PLAN_DEPS_ECHARTS_TS7_2026-07-27.md` (lots A/B/C, contrat
plan-execution propre). Ce volet ne duplique pas ces étapes : il fixe QUAND et QUOI
exécuter, avec l'état constaté le 2026-08-02. Chaque merge sur main = déploiement prod
(prévenir avant).

- [x] D1 **Triage des 2 nouvelles PR (#71, #72)** — fait le 2026-08-02 : CI re-vérifiée
      (14/14 checks verts chacune), #71 puis #72 mergées en squash avec CI main vérifiée
      verte après chacune (go utilisateur reçu). Incident intercalé : la CI main est
      passée rouge après #71 sur un TODO(expiry:2026-08-01) échu (bombe datée sans lien
      avec #71) + un flake persist qualifié (10/10 PASS local) → hotfix PR #73 mergée
      (voir Découvertes). Deploys prod déclenchés et verts (#71, #73, #72).
- [x] D2 **Lot A — PR #67 (go-minor-patch, 10 paquets dont duckdb-go 2.10505)** —
      clos le 2026-08-02 (agent Opus A1-A6, orchestrateur A7/A8) : changelog duckdb-go
      2.10505/DuckDB 1.5.5 audité, AUCUNE mention ART/index ; fix allowlist = 2 entrées
      datées (`QUERY /static/*` + `QUERY /static/commendations/*`) ; gates locaux
      `go test ./...` et `-tags=integration -p 1` exit 0 (persist + sync verts) ;
      conflit avec le hotfix #73 résolu en reprenant la version main (TODO 2026-09-15) ;
      CI branche 15/15 verte, mergée squash sur go utilisateur, **CI main post-merge
      verte**, thought_log fait. `main` rapatriée dans la branche du lot.
- [ ] D3 **Lot B — echarts 5.6.0 → 6.1.0 (CVE-2026-45249, XSS)** — effort moyen.
      Décision du 27/07 maintenue : ne pas re-différer une 3e fois. SÉQUENCEMENT
      IMPOSÉ : après le merge du lot 2, depuis un `main` à jour (branche
      `fix/echarts-6-security-bump`) — le harnais Playwright `toHaveScreenshot` (B2)
      doit capturer les visuels FINAUX, or 2.3 et 3.4/3.5 modifient des graphes dont
      `OutcomeSequenceTape`, qui fait partie des wrappers à couvrir. Exécuter B1-B8 ;
      le gros du travail est le harnais, pas le bump ; sign-off visuel utilisateur (B7)
      requis.
- [ ] D4 **Lot C — TS7 (PR #70) : statuer, ne pas exécuter** — vérifié le 2026-08-02 :
      `typescript-eslint@latest` = 8.65.0, peer `typescript >=4.8.4 <6.1.0`, TS 7.0.2
      toujours HORS range (seules des 8.65.1-alpha existent). Report justifié `[!]`
      (dépendance externe). À la clôture du lot : revérifier une dernière fois
      `npm view typescript-eslint@latest peerDependencies`, consigner le statut dans
      Notion, laisser la PR #70 ouverte avec un commentaire de blocage daté si le range
      n'a pas bougé. Ne PAS exécuter C2+ d'ici là.

**Gate D** : gates propres à chaque lot du plan dédié (CI GitHub verte sur branche
AVANT merge, pas seulement un rejeu local).

---

## Hors périmètre (consigné, ne pas traiter dans ce lot)

| Sujet | Statut |
|---|---|
| Replay 2D (tout le bloc) | Chantier utilisateur en cours |
| Killsource (branchement + validation précision Infinite + assets icônes armes) | Re-différé (décision 2026-08-02) — branche `feat/killsource-prod`, entrer par `.ai/HANDOFF_KILLSOURCE_REPRISE.md` |
| Note de perf modes objectifs | Chantier isolé futur — voir décision verrouillée (subtilité écrasement/participation) |
| 4 items `[POSTPONED]` de la section (assistants par kill, spartan abilities, profil Ascension, drawer lobby) | Marqués reportés par l'utilisateur |
| 2 items `[ATTENTE REPLAY 2D]` (replay unique démo, niveaux de bleu cartes) | Attendent le Replay 2D |
| Colonne Replay dans l'historique | À la livraison du Replay 2D |
| TypeScript 7 (PR #70) | Bloqué par dépendance externe (typescript-eslint) — suivi en D4, exécution interdite tant que le range peer n'inclut pas TS7 |
| `lib/skillTiers.ts` (couplage Go<->TS) | Garde-rail existant, hors du périmètre i18n TOML |

## Protocole de reprise de session

1. Lire ce fichier : les statuts `[ ]`/`[x]`/`[~]`/`[!]` font foi.
2. `git log --oneline -10` sur `feat/v7.3-notion-lot2` (chaque item livré = commit
   dédié référencé dans la case).
3. `.ai/thought_log.md` (entrées 2026-08-XX du lot).
4. Interdiction de fixes opportunistes hors périmètre : toute découverte va dans la
   section ci-dessous, rien d'autre.

## Découvertes en cours d'exécution (append-only)

- [2026-08-02, agent 0.2] **Contraste des marqueurs de dominance** : aucun token de
  dominance n'atteint 3:1 contre la couleur d'issue qu'il surmonte (domination 1,40:1,
  remontada 2,04:1, contre-remontada 1,47:1, humiliation 1,13:1, débandade 1,03:1) —
  c'est le liseré `tooltipBg` 1 px qui rend le losange actuel visible, pas sa couleur.
  Toute forme retenue en 3.5 doit conserver un liseré/gouttière.
- [2026-08-02, agent 0.2] **Collision de tokens** : `--ac-outcome-dnf` et
  `--ac-narrative-humiliation` partagent exactement `#8B5CF6` (hors périmètre du lot,
  à statuer plus tard).
- [2026-08-02, agent 0.2] `#00DC82` et `#33D6FF` (palette dominance) sortent de la
  bande de clarté et tombent sous 3:1 contre la surface claire (1,75 / 1,66) — un
  marqueur de ces couleurs posé sur le fond de page en thème clair exige un cerne.
- [2026-08-02, agent 0.2] `EChartsThemeColors` (`lib/echarts/themeColors.ts`) n'expose
  aucune couleur de surface (`CHART_BG = 'transparent'`) : une découpe couleur-fond en
  3.5 exigerait d'ajouter un champ (`--background`/`--card`), sinon seam gris
  `tooltipBg`.
- [2026-08-02, agent 0.2] Contrainte dirimante pour tout rail sous la bande : sur
  Relations le composant est monté avec `height=64` et une grille 32+32 → hauteur de
  plot nulle (aucune place sous les brackets).
- [2026-08-02, agent 0.1] **Palette joueurs non discriminable** (tous les charts
  multi-joueurs Escouade) : ΔE 6,7 entre `narrative-dominant` et `divergent-pos`
  (deux verts quasi identiques, plancher 15), ΔE 5,7 en protanopie entre
  `divergent-pos` et `perf-tier-3` ; contraste < 3:1 sur fond de carte en clair pour
  les 4 tokens joueur.
- [2026-08-02, agent 0.1] **Deux définitions du « rendement »** : la carte Rendement
  trace `damage_dealt / kills` (sans assistances) alors que `match-card.tsx:103`
  utilise `effectiveDmgPerFrag(...)` aligné ADR 0006 — deux écrans, deux nombres
  sous le même mot. À trancher en 3.4.
- [2026-08-02, agent 0.1] `rendement_offensif`/`resistance_defensive` sont servis
  dans le payload mais ne servent qu'à `hasEfficiencyData()` — le graphe recalcule
  autre chose ; un joueur `damage_dealt=0` passe le test sans courbe offensive.
- [2026-08-02, agent 0.1] `divergent-neutral` est un bleu (#60A5FA) utilisé comme
  point milieu du dégradé rouge↔vert (`oneLifeDamageGradient.ts`) — il manque un
  token gris neutre pour les encodages divergents.
- [2026-08-02, agent 0.1] `DefensiveResistanceP80 = 1,65` est une constante Go alors
  que le pendant offensif est déclaré par titre dans `constants.toml` (asymétrie
  title-agnostic).
- [2026-08-02, agent 1.3/1.4] **Le garde-rail du fragment timezone n'existe pas** :
  `player_matches_repo.go:145-146` référence `analysis/start_time_canonical_test.go`
  qui est introuvable — rien n'interdit de réécrire le fragment à la main (c'est
  ainsi que le bug 1.3 est né). Recommandation : ratchet grep sur `start_time_utc`
  hors helper (candidat passe post-lot, règle 6 CLAUDE.md).
- [2026-08-02, agent 1.3/1.4] Le seed Infinite écrit toujours des libellés FR dans
  `citation_mappings.category` (neutre : normalisation à la lecture) — écrire les
  clés stables serait une migration de données, hors lot.
- [2026-08-02, agent 1.3/1.4] `CitationFullMapping.Category` (`sync/citations.go`)
  est scanné mais jamais consommé — champ mort candidat au nettoyage (règle 7).
- [2026-08-02, agent 1.3/1.4] `routeTree.gen.ts` se fait régénérer (réordonnancement
  603 lignes) au passage de tsc/vitest — décalage de version du plugin TanStack
  Router à investiguer (bruit de diff récurrent).
- [2026-08-02, agent 1.5/1.6] **Les tests du chemin atomique des likes sont
  désactivés depuis le 2026-05-15** (`media_service_atomic_integration_test.go`,
  tag `atomic_legacy`) : le chemin nominal de prod n'avait AUCUN test actif —
  c'est ce qui a laissé passer les régressions successives. Recommandation :
  réactiver ou supprimer (anti-pattern dead code museum).
- [2026-08-02, agent 1.5/1.6] **`/players/{player_slug}` sans RequireAuth** :
  l'ownership laisse passer `sess == nil` sur toutes les routes joueur. Traité
  pour le like uniquement (401 gaté) — audit des autres routes du groupe à
  recommander.
- [2026-08-02, agent 1.5/1.6] Like historique orphelin (event du 26/04/2026 en
  chemin absolu, sans correspondance `media_files`) — réparable par INSERT d'un
  event sous la clé canonique.
- [2026-08-02, agent 1.5/1.6] **`media_files.liked` est global** (partagé entre
  tous les viewers) alors que `media_likes_history` est par liker — le cœur d'un
  média est commun à tous. À trancher avec l'item 3.1 (suppression de médias).
- [2026-08-02, agent 1.5/1.6] Placeholder « Défi » en dur non i18n
  (`HomeChallengesList.tsx:119`) ; compat `LegacyMediaItemRow` sans date
  d'expiration (`features/media/queries.ts`).
- [2026-08-02, agent 1.5/1.6] Flakes qualifiés (verts au rejeu isolé) :
  `TestStartImport_HappyPathReturns202WithJobID` (cleanup TempDir Windows),
  `TestCareerLive_NilAPIResponse_NotCached` (timeout 2 s sous charge).
- [2026-08-02, orchestrateur] Retombée du fix 1.3 corrigée par l'orchestrateur :
  `TestCareerRepo_GetRelationsHeatmap` (integration) attendait des heures UTC et
  dépendait du fuseau machine → ouvert via le chemin production épinglé UTC
  (les non-UTC restent couverts par les tests dédiés 1.3).
- [2026-08-02, orchestrateur] **CI main rouge post-merge #71** (run 30745523574),
  2 causes sans lien avec #71 : (1) `TODO(expiry:2026-08-01)` échu dans
  `season_pass_repo_tracks.go:297` → hotfix PR #73 (échéance 2026-09-15, critère
  mesurable : 0 ligne kind='track-def' dans asset_index prod, à vérifier avec les
  ops Phase 4) ; (2) `TestWorker_Run_PersistsAndACKs` = flake qualifié (10/10 PASS
  local `-tags=integration`). Merge #72 suspendu jusqu'à CI main verte.

## Décisions d'artefacts (à remplir au gate 0)

- Rendement & Résistance : (en attente)
- Marqueurs dominance V/D : (en attente)
