# PLAN v7.3 â€” Lot 2 Â« petites choses Â» (section Notion, hors Replay 2D)

> Ã‰crit le 2026-08-02 aprÃ¨s reconnaissance (3 agents Explore) et 3 questionnaires
> utilisateur. La v7.3 reste ouverte (tag non posÃ©) : ce lot s'y ajoute.
> **ExÃ©cution sous contrat du skill `plan-execution`** (ordre strict, une Ã©tape Ã  la
> fois, gate passÃ© avant l'Ã©tape suivante, zÃ©ro report d'Ã©tape exÃ©cutable).

## Objectif et critÃ¨re de succÃ¨s

Traiter tous les points non barrÃ©s de la section Notion Â« Pour la v7.3 Â» hors Replay 2D,
hors POSTPONED, hors dÃ©cisions de report ci-dessous. SuccÃ¨s = chaque item de ce plan
statuÃ© (`[x]` fait / `[~]` couvert ailleurs avec rÃ©fÃ©rence / `[!]` non traitÃ© justifiÃ©),
gates verts, revue navigateur des changements visuels faite, section Notion mise Ã  jour
(points barrÃ©s + notes), thought_log Ã  jour.

**Branche** : `feat/v7.3-notion-lot2` depuis `main` (Ã  crÃ©er au dÃ©marrage de
l'exÃ©cution â€” 1 branche, N commits). Effort global : moyen-lourd (~6 j-agent).

## DÃ©cisions utilisateur verrouillÃ©es (questionnaires du 2026-08-02)

| Sujet | DÃ©cision |
|---|---|
| Killsource | **Re-diffÃ©rÃ©** (branche `feat/killsource-prod` vivante, handoff du 31/07 â€” ne pas perdre ; inclut la validation Â« prÃ©cision Infinite exploitable ? Â») |
| Artefacts â‰¥3 propositions | **Rendement & RÃ©sistance** (Dynamique) + **losanges sÃ©quence V/D** uniquement. FDA/intensitÃ© et Rythme des rencontres = correctifs directs |
| Tableaux d'historique | Colonne **Score personnel** maintenant ; colonne **Replay Ã  la livraison du Replay 2D** (pas dans ce lot) |
| Suppression de vidÃ©os | **PropriÃ©taire + admin**, dÃ©finitive, avec confirmation |
| i18n TOML/TS en double | **Supprimer le repli** : flag `MULTI_TITLE_API_ENABLED` invariant, fallbacks supprimÃ©s, `metricLabel` migrÃ© vers `useFieldLabel` |
| Note de perf modes objectifs | **Chantier isolÃ© futur** (hors lot). Consigner : scission ranked par famille = bonne piste validÃ©e ; subtilitÃ© â€” ne pas rÃ©compenser l'absence de combat, mais un joueur Ã©crasÃ© peut quand mÃªme jouer l'objectif |
| Kills vÃ©hicules | **H5 maintenant** (classes dÃ©jÃ  en base), Infinite au killsource. Exigence : le sous-niveau du sunburst doit distinguer **chaque vÃ©hicule**, pas un bucket unique |
| Sessions escouade | RÃ¨gle unique **Â« matchs commencÃ©s ensemble Â»** (tous les sÃ©lectionnÃ©s au roster) partout en contexte escouade ; Â« composition exacte Â» devient une option dÃ©sactivÃ©e par dÃ©faut ; Ã©checs de chargement visibles |
| PrÃ©cision par arme | **Question rÃ©pondue, rien Ã  implÃ©menter** : H5 = donnÃ©es dÃ©jÃ  en base (`weapon_accuracy`, tirs tirÃ©s/touchÃ©s par arme) ; Infinite = travail en cours (killsource), Ã  valider si exploitable |
| Bonus (i) | Cible = graphe **Â« Frags / Morts Â» de la page Escouade** |
| DÃ©couvertes d'exÃ©cution | DÃ©lÃ©guÃ© (message du 2026-08-02 en cours de lot) : triage et traitement **selon les recommandations de l'orchestrateur** â€” celles qui croisent un item du lot s'y traitent, les autres en passe dÃ©diÃ©e post-lot avec recommandation par dÃ©couverte |
| Dependabot au fil de l'eau | DÃ©lÃ©guÃ© (mÃªme message) : traiter **au mieux jugÃ©** â€” constat du 2026-08-02 : rien de neuf (alerte echarts = D3, PR #70 = D4, runs Â« Dependabot Updates Â» rouges = re-tentatives echarts sans objet car PR #49 fermÃ©e manuellement en juin) |

## Ordre d'exÃ©cution

Phase 0 en premier (les artefacts partent chez l'utilisateur, sa dÃ©cision tourne en
parallÃ¨le), puis 1 â†’ 2 â†’ 3 (3.4/3.5 dÃ¨s les choix d'artefacts rendus) â†’ 4.
Une phase est close quand tous ses items sont statuÃ©s ET son gate est vert.

Le volet D (branches Dependabot) vit sur des branches SÃ‰PARÃ‰ES du lot : D1 et D2 sont
exÃ©cutables dÃ¨s le dÃ©marrage (indÃ©pendants), D3 s'exÃ©cute APRÃˆS le merge du lot
(sÃ©quencement imposÃ©, voir D3), D4 est une vÃ©rification de statut Ã  la clÃ´ture.

---

## Phase 0 â€” Artefacts de propositions visuelles (gate utilisateur)

Charger les skills `dataviz` puis `artifact-design` AVANT d'Ã©crire chaque page.
DonnÃ©es rÃ©alistes tirÃ©es des formes rÃ©elles des payloads (lire les builders), rendu
light/dark, chaque proposition faisable en ECharts avec les tokens sÃ©mantiques du repo.

- [x] 0.1 Artefact **Rendement & RÃ©sistance** (page Dynamique escouade) â€” publiÃ© le
      2026-08-02 (4 propositions : A axe retournÃ© log, B indice de vies, C Ã©cart Ã  la
      frontiÃ¨re Ã©lite par piste, D grille de session ; light/dark, donnÃ©es rÃ©elles) :
      https://claude.ai/code/artifact/9775c1ab-54b8-4b28-9fe1-0ba6106da39e (republication orchestrateur — les publications des agents avaient échoué en silence)
- [x] 0.2 Artefact **marqueurs de dominance** de la sÃ©quence V/D â€” publiÃ© le
      2026-08-02 (4 propositions A/B/C/D, rendus 20/60/150/400 matchs + bac d'essai
      12â†’600, clair/sombre) :
      https://claude.ai/code/artifact/b0cee06c-c152-455c-84e5-dd677a39dcea (republication orchestrateur — idem)
      (`components/charts/OutcomeSequenceTape.tsx`, losanges 7x7 px en dur,
      invisibles sous 6 px/match) : >= 3 formes alternatives, avec le comportement
      aux faibles largeurs traitÃ© dans chaque proposition.

**Gate 0** : 2 artefacts publiÃ©s (privÃ©s), URLs remises Ã  l'utilisateur, dÃ©cisions
consignÃ©es dans ce fichier (section DÃ©cisions d'artefacts en bas). L'implÃ©mentation
(3.4/3.5) ne dÃ©marre pas sans dÃ©cision.

## Phase 1 â€” Bugs

- [x] 1.1 (commit 3862ff083 â€” revue navigateur PASSÃ‰E le 2026-08-02 : scÃ©nario
      discriminant Playwright Â« ancre pÃ©rimÃ©e + sÃ©lection manuelle ancienne â†’ reload Â»
      = rÃ©-ancrage sur la derniÃ¨re session (reanchorOK) ET contre-Ã©preuve Â« ancre
      valide + choix manuel Â» = choix respectÃ© (respectOK). Cause : triple maillon
      garde composition-seule / pas de useFollowLatestSession en escouade /
      followLatest Ã©teint par les chemins techniques â†’ fix `lastAnchoredLatestSession`
      persistÃ© + rÃ¨gle Â« session jamais ancrÃ©e â†’ rÃ©-ancrage Â». Reproduction 31/07
      impossible en local, DBs au 23/07 â€” validÃ© sur la derniÃ¨re session locale.)
      **Autosnap escouade** : la page ne s'est pas ouverte sur la session du 31/07.
      Diag sur piÃ¨ces : `features/squad/SquadLayout.tsx:434-461` +
      `features/squad/squadPending.ts` (`decideCompositionReanchor`) + persistance
      `picked_sessions` par LABEL et `isAutoSnappingToLatest`
      (`stores/createFilterStore.ts`, clÃ© `levelup-squad-filter-v1`). Reproduire avec
      les donnÃ©es rÃ©elles de la session du 31/07, corriger, tests vitest sur
      `decideCompositionReanchor`, revue navigateur.
- [x] 1.2 (commit 3862ff083 â€” revue navigateur + preuve API PASSÃ‰ES le 2026-08-02 :
      badges du sÃ©lecteur = compte roster (9/4/5/4) = lignes du tableau = en-tÃªte
      Â« 9 matchs Â» ; preuve API du contraste suffixe/roster (Â« (7) Â»â†’4, Â« (12) Â»â†’5,
      Â« (6) Â»â†’3) ; strict dÃ©cochÃ© par dÃ©faut, persistant au reload, et Â« 0 match Â»
      sous strict prouvÃ© vÃ©ritÃ© serveur (filter_exact_composition=true â†’ history=0) ;
      data_issues=[] sur le chemin nominal. Cause du 11/8/6/5 : 4 populations
      empilÃ©es ; livrÃ© : `match_count` roster + `mergeSessionCounts`,
      `filter_exact_composition` optionnel dÃ©faut off (contrat, toggle FR/EN, query
      key), heatmap sans re-filtrage privÃ©, collecteur `data_issues`.)
      **Compteurs de sessions unifiÃ©s** (le 11/8/6/5) :
      `internal/service/teammates/teammates_service.go:162-301`. RÃ¨gle canonique
      Â« commencÃ©s ensemble Â» = intersection roster (population B). (a) Le compteur de
      la liste des sessions en contexte escouade affiche le compte Â« ensemble Â» ;
      (b) `filterExactComposition` devient un paramÃ¨tre API optionnel, dÃ©faut off
      (contrat + toggle UI FR/EN) ; (c) le heatmap perd son re-filtrage privÃ©
      (`teammates_squad_charts_sessions_maps.go:300-333`) et consomme la mÃªme
      population ; (d) tout Ã©chec de chargement best-effort (LoadMainTeamParticipants,
      LoadFor) cesse d'Ãªtre silencieux : `slog.ErrorContext` + Ã©tat d'erreur visible
      cÃ´tÃ© UI (fin des chiffres non reproductibles). Tests service (mock port) +
      httptest si contrat touchÃ© + vitest + revue navigateur.
- [x] 1.3 (commits 24720944f + ef7d32cbc â€” revue navigateur PASSÃ‰E le 2026-08-02 :
      heures locales Ã  l'Ã©cran (pics 21h-23h soir / 11h-13h midi, l'UTC aurait montrÃ©
      19h-21h), tooltip 3 lignes FR (Â« Joueur / CrÃ©neau : 16h / Matchs communs Â»)
      ET EN (Â« Player / Time slot / Shared matches Â»), bascule Par heure/Par jour
      changeant l'Ã©tiquette. Double cause prouvÃ©e : AT TIME ZONE 'UTC' post-COALESCE
      + COALESCE mal parenthÃ©sÃ© ; fix = fragment canonique `StartTimeCanonicalSQL`,
      3 tests DuckDB :memory: + 5 tests web + test heatmap Ã©pinglÃ© UTC.)
      **Rythme des rencontres â€” heures fausses** :
      `internal/platform/duckdb/queries_relations_moments.go:41-42` â€” le
      `AT TIME ZONE 'UTC'` explicite annule le fuseau de session et livre des heures
      UTC sous l'alias `hour_local`. Corriger pour rendre l'heure dans
      `cfg.UserTimezone` en respectant le fragment canonique (rÃ¨gle CLAUDE.md nÂ°8 :
      COALESCE conservÃ©). Test DuckDB `:memory:` avec fuseau fixÃ© non-UTC.
      + **Tooltip refait** (`features/palmares/RelationsMomentsHeatmap.tsx:88-100`) :
      contenu lisible (joueur, jour/heure, n matchs), i18n FR/EN.
- [x] 1.4 (commit 24720944f â€” la revue navigateur du gate 1 ne listait pas cet item ;
      paritÃ© FR/EN verrouillÃ©e par le garde-rail de labels.test.ts et les 7 tests Go.
      Normalisation canonique `commendation_category.go`
      (7 clÃ©s stables, patron medal_category) branchÃ©e aux 3 frontiÃ¨res H5 +
      service + analysis ; audit (c) Infinite : trou avÃ©rÃ© (libellÃ©s FR en dur du
      seed) corrigÃ© par normalisation Ã  la lecture, chemin sync/citations.go:215
      confirmÃ© sans consommateur aval ; 7 clÃ©s citations.category.* FR/EN + 
      labels.ts patron mÃ©dailles ; 7 tests Go + 4 tests web paritÃ©.)
      **CatÃ©gories de citations en anglais** : (a) H5 â€” normaliser les catÃ©gories
      brutes de l'API (`"MULTIPLAYER"`, `"GAME MODE"`...) en clÃ©s stables cÃ´tÃ© Go
      (`platform/duckdb/halo5/halo5_commendation_defs.go:68`,
      `service/.../commendation_totals.go:69`) ; (b) clÃ©s
      `citations.category.<key>` FR/EN dans
      `apps/web/src/lib/i18n/manifests/citations.toml` + rÃ©solution dans
      `features/citations/CitationsView.tsx:49` (patron mÃ©dailles,
      `medal_category_table.go` + `medals.toml`) ; (c) audit Infinite :
      `citation_mappings.category` (`internal/sync/citations.go:215`) â€” vÃ©rifier que
      les clÃ©s servies sont stables et couvertes par le manifeste. Tests Go
      (normalisation) + paritÃ© typÃ©e `Record<Locale, T>`.
- [x] 1.5 (CONTEXTE UTILISATEUR reÃ§u en cours d'exÃ©cution et transmis Ã  l'agent :
      rÃ©gressions rÃ©currentes Â« en gÃ©nÃ©ral Ã  cause de l'enregistrement en BDD qui ne
      se fait pas de maniÃ¨re instantanÃ©e Â» â€” confirmÃ© par le diagnostic : c'Ã©tait le
      `tx.Commit()` sans CHECKPOINT. Exigence de cadrage honorÃ©e : sÃ©mantique
      lecture-aprÃ¨s-Ã©criture commentÃ©e sur place + test reproduisant la fenÃªtre WAL.)
      (commit ea5611397 â€” revue navigateur IMPOSSIBLE en local : l'API sert 0
      mÃ©dia pour tous les joueurs, les fichiers vivent sur le VPS ; le gate 1 ne
      l'exigeait pas pour cet item. Couverture : test discriminant
      `SurvivesWALLoss` prouvÃ© rouge-sans/vert-avec + intÃ©gration -p 1 verte ;
      vÃ©rification visuelle en prod Ã  faire avec les ops Phase 4 (4.1).
      Diagnostic :
      piste 1 Ã©cartÃ©e sur piÃ¨ces mais garde livrÃ©e ; piste 2 PROUVÃ‰E (chemin absolu
      vs `file_path` relatif forward-slash 219/219 â†’ UPDATE 0 ligne â†’ 404) ; piste 3
      PROUVÃ‰E (rÃ©ponse = chemin stockÃ©, cache indexÃ© par URL servable â†’ onSuccess
      muet) ; cause racine de la RÃ‰CURRENCE = `tx.Commit()` nu, like en WAL jusqu'Ã 
      5 min, effacÃ© par tout redÃ©marrage/dÃ©ploiement â€” `CommitWithCheckpoint`
      existait, documentÃ© pour les likes, jamais appelÃ©. Correctifs : conversion
      URL unifiÃ©e 3 endpoints + garde-rail, Ã©cho du file_path reÃ§u, 401
      `like_requires_session` + toast FR/EN, CommitWithCheckpoint avec garantie
      commentÃ©e, media.go 596â†’485 L. 11 tests Go + 2 web dont
      `SurvivesWALLoss` au pouvoir discriminant prouvÃ© rouge-sans/vert-avec.)
      **Likes mÃ©dias cassÃ©s** : diagnostic hiÃ©rarchisÃ© sur piÃ¨ces AVANT tout code,
      dans cet ordre : (1) session absente => `liker_slug` vide => like fantÃ´me sans
      401 (`api/handlers/media.go:312-372`) ; (2) `urlToFilePath` dÃ©salignÃ© aprÃ¨s un
      changement de prÃ©fixe d'URL mÃ©dia ; (3) forme de cache divergente
      (`features/media/queries.ts:140-175`) ; (4) 503 `ErrDBLocked` masquÃ© par
      l'optimistic update + `refetchType: 'none'`. Corriger la cause prouvÃ©e +
      garde anti-silence : sans session le like doit Ã©chouer visiblement (401 gatÃ© ou
      erreur UI explicite â€” dÃ©cision technique sur piÃ¨ces, contrat mis Ã  jour si
      besoin). Tests + `go test -tags=integration -p 1` (shared_social touchÃ©).
- [x] 1.6 (commit ea5611397 â€” vÃ©rifiÃ© le 2026-08-02 sur serveur dÃ©mo rÃ©el lancÃ© par
      l'orchestrateur (LEVELUP_DEMO_MODE=true, :8000) : l'endpoint season-pass sert
      les 4 `image_url` et les 4 assets rÃ©pondent 200 image/png (6,9-7,2 Ko). Le
      rendu Home dÃ©mo complet exige la fixture DB DemoPlayer, absente du poste
      (Â« Sync in progress Â») â€” le rendu passe par le mÃªme composant que la prod,
      couverture CI Â« Simulation regen demo Â». `image_url` ajoutÃ© aux 4 items des
      DEUX fixtures vers
      `/static/prestige-assets/Objectives-badges/*.png` (assets existants du repo),
      garde-rail `TestGetChallenges_DemoMode_ImagesServable` (clÃ© + fichier
      existant), vÃ©rifiÃ© en dÃ©mo rÃ©elle :8123 â€” 4 URLs en 200 image/png, paritÃ©
      FR/EN.)
      **DÃ©mo : images des dÃ©fis absentes** : les fixtures
      `internal/service/demo_fixtures/challenges.json` + `challenges.en.json` n'ont
      pas de clÃ© `image_url` (le mode dÃ©mo bypasse le cache DB â€”
      `home_service_battlepass.go:88-89`). Ajouter `image_url` aux items des DEUX
      fixtures vers un asset servi par `/static/` (vÃ©rifier qu'un asset embarquÃ©
      existe, sinon en ajouter un). VÃ©rification en dÃ©mo locale.

**Gate 1** : `cd apps/go-api && go test ./...` exit 0 sur les paquets touchÃ©s ;
`go test -tags=integration -p 1 ./...` exit 0 (1.5 touche shared_social) ;
`make check-types` ; `make test-web` ; contrat rÃ©gÃ©nÃ©rÃ© si routes/params modifiÃ©s
(`openapi` + `make generate-types`, 0 chemin perdu) ; revue navigateur des correctifs
1.1, 1.2, 1.3, 1.6.
> GATE 1 PASSÃ‰ le 2026-08-02 (orchestrateur) : go test ./... exit 0 ;
> -tags=integration -p 1 exit 0 (aprÃ¨s correction de la retombÃ©e 1.3 sur le test
> heatmap, commit ef7d32cbc) ; tsc exit 0 ; vitest 3248/3248 (14 skips
> prÃ©existants) ; openapi-gen -check sans dÃ©rive ; matrice navigateur Playwright
> (scripts .tmp.mjs supprimÃ©s, captures au scratchpad) : 1.1 reanchor+contre-Ã©preuve
> PASS, 1.2 compteurs+strict+preuve API PASS, 1.3 FR/EN PASS, 1.6 vÃ©rifiÃ© en dÃ©mo
> rÃ©elle (API + assets ; rendu complet = fixture DemoPlayer absente du poste),
> 1.5 non vÃ©rifiable en local (0 mÃ©dia servi) â†’ vÃ©rif prod aux ops 4.1.

## Phase 2 â€” Petites UI

- [x] 2.1 (commit 610aaeb23 â€” revue navigateur PASSÃ‰E : cas extrÃªme vÃ©rifiÃ© sur un
      match sans mÃ©daille ni citation, les deux cartes se compactent en fines
      rangÃ©es Ã©galisÃ©es, plus de bloc vide 280 px. Plancher
      remplacÃ© par MEDALS_CARD_MIN_BODY_HEIGHT=96 + flex, Ã©galisation par la grille
      du parent conservÃ©e ; 4 tests.)
      **MÃ©dailles/citations Ã  peu d'Ã©lÃ©ments** :
      `features/match-view/MatchSummaryMedalsAndCitations.tsx` â€” la co-localisation
      2 colonnes existe dÃ©jÃ  ; le problÃ¨me est le plancher `CARD_HEIGHT = 280`.
      Hauteur adaptative quand peu d'Ã©lÃ©ments (les deux cartes se compactent sur la
      rangÃ©e). Revue navigateur sur un match pauvre en mÃ©dailles.
- [x] 2.2 (commit 610aaeb23 â€” revue navigateur PASSÃ‰E : lÃ©gende Â« Frags Â· Morts Â·
      Bonus barrÃ© (masquÃ© par dÃ©faut) Â· â“˜ Â» constatÃ©e Ã  l'Ã©cran, aide adjacente au
      bouton qu'elle explique. â“˜ posÃ© sur
      le bouton de lÃ©gende Â« Bonus Â» (l'Ã©lÃ©ment expliquÃ©), prop info optionnelle de
      SquadToggleLegendChart, textes FR/EN ADR 0006 validÃ©s orchestrateur ; 4 tests.)
      **(i) Bonus** sur le graphe Â« Frags / Morts Â» de la page Escouade
      (`features/squad/charts/squadPerformanceLineCharts.ts`, sÃ©rie `Bonus` masquÃ©e
      par dÃ©faut) : InfoTooltip FR/EN expliquant Bonus = assistances/3 (ADR 0006).
      Si le mÃªme composant sert d'autres pages, elles en hÃ©ritent (pas de travail
      supplÃ©mentaire hors pÃ©rimÃ¨tre).
- [x] 2.3 (commit 610aaeb23 â€” revue navigateur PASSÃ‰E : â“˜ sur le titre IntensitÃ©,
      3 panneaux joueurs avec la courbe d'Ã©quipe pointillÃ©e distincte du repÃ¨re
      10 % ; symboles au survol couverts par le helper testÃ©. (a) texte
      partagÃ© FdaGapTooltipText branchÃ© sur les 2 instances + (fold dÃ©couverte,
      dÃ©lÃ©gation utilisateur) la 3e surface SquadFdaGapCumulativeCard ; (b) tooltip
      intensitÃ© rÃ©Ã©crit sans jargon + (fold) la 3e instance Timeseries alignÃ©e sur
      le mÃªme texte ; (c) helper hoverRevealSymbol centralisÃ© (cause du figÃ© :
      symbol 'none' ne s'affiche pas Ã  l'emphase) appliquÃ© aux 4 courbes ; (d)
      courbe Ã‰quipe dÃ¨s 3 panneaux via rows['all'] du back (population sans filtre
      joueur, mÃªme helper phaseProfile â€” pas de moyenne de mÃ©dianes), dÃ©gradation
      silencieuse sans 'all'. +25 tests.)
      **FDA gap + intensitÃ© â€” pÃ©dagogie sans refonte** (choix utilisateur : pas
      d'artefact) : (a) InfoTooltip pÃ©dagogique sur les 2 instances du FDA gap
      (`features/timeseries/TimeseriesFdaGapTrend.tsx`,
      `features/session-detail/SessionFdaGapCumulative.tsx`) â€” il n'en existe aucun
      aujourd'hui ; (b) rÃ©Ã©crire celui du profil d'intensitÃ©
      (`SessionIntensityProfile.tsx:62-71`) en langage non technique ; (c) affordance
      de survol : symboles visibles au hover (le rendu canvas sans aucun point
      survolable est ce qui donne l'impression d'une image figÃ©e) ; (d) profil
      d'intensitÃ© escouade : ajouter la courbe agrÃ©gÃ©e de l'Ã©quipe quand >= 3 joueurs
      sÃ©lectionnÃ©s (`features/squad/charts/squadIntensityProfileChart.ts`).
      Validation en revue navigateur avec l'utilisateur.
- [x] 2.4 (commit 008d4cf1f â€” revue navigateur PASSÃ‰E : 19 badges de rang en image
      sur 20 lignes Explorer (+1 dÃ©gradation texte), zÃ©ro icÃ´ne â“˜ dans les th, le
      survol du libellÃ© ouvre l'aide, le clic trie toujours (ordre changÃ© constatÃ©).
      (a) champ
      `skill_rank_image_url` via chokepoint extrait `analysis.SkillBadgeURL` +
      `WithSkillBadgeResolver` (chemin Home rÃ©utilisÃ©, zÃ©ro slug) ; (b) largeurs :
      Explorer px-1.5 + icÃ´nes w-8 + carte tronquÃ©e 12c + rang image (~-126 px),
      Escouade px-2 (~-104 px), Timeseries/Session/CarriÃ¨re hÃ©ritent d'Explorer,
      th nowrap = aucun libellÃ© coupable FR/EN ; (c) InfoTooltip mode trigger span
      sans onClick (piÃ¨ge bouton-dans-bouton Ã©vitÃ©), HeaderInfoTooltip supprimÃ©,
      8 surfaces columnMeta + SortableTh (9 consommateurs) migrÃ©es. Contrat : +10
      openapi, +6 generated, 203/203 chemins. NOTE nommage : le type Go rÃ©el est
      `ExplorerMatchesRow`, le schÃ©ma openapi `ExplorerMatchRow` est un orphelin
      (dÃ©couverte). RÃ©serve agent sur un premier go test exit 1 non capturÃ© :
      TRANCHÃ‰E flake â€” rejeu orchestrateur complet exit 0.)
      **Tableaux â€” colonne Rang en image + tooltips d'en-tÃªte** : (a) ajouter
      `skill_rank_image_url` au contrat `ExplorerMatchRow` servi par le backend via
      l'adaptateur d'assets du titre (prÃ©cÃ©dent : `RecentMatchItem.skill_rank_image_url`
      pour Home) â€” dÃ©gradation propre : champ null => texte localisÃ© actuel (H5) ;
      assets dÃ©jÃ  prÃ©sents (`static/ranks/halo_infinite/unranked_0..9.png` + CSR) ;
      (b) rÃ©duction des largeurs de colonnes : Explorer d'abord, puis passe sur
      Escouade, Timeseries, Session, CarriÃ¨re ; (c) tooltip d'en-tÃªte portÃ© par le
      LABEL : Ã©tendre `InfoTooltip` (`components/ui/info-tooltip.tsx`) avec un trigger
      non-bouton (piÃ¨ge documentÃ© bouton-dans-bouton, `lib/table/columnMeta.tsx:11-15`),
      retirer les icÃ´nes â“˜ des en-tÃªtes. Contrat + `make generate-types`.
- [x] 2.5 (commit 008d4cf1f + harmonisation registre orchestrateur â€” revue
      navigateur PASSÃ‰E : colonne prÃ©sente et triable, tooltip lisible ; registre
      FR harmonisÃ© en tutoiement (convention dominante 111 vs 37 constatÃ©e).
      Â« L'historique Â» confirmÃ© par grep = UN composant ExplorerMatchesTable montÃ©
      5 fois (Explorer matchs, Explorer joueur alliÃ©/ennemi, Session, Timeseries
      Progression, CarriÃ¨re marquants) â€” MatchHistoryPage n'existe pas cÃ´tÃ© web.
      `personal_score` = donnÃ©e dÃ©jÃ  lue puis JETÃ‰E par enrichRow (fuite, zÃ©ro SQL
      nouveau). En-tÃªte Â« Score personnel Â»/Â« Personal score Â», tri TanStack, aide
      ajoutÃ©e sur Â« Score Â» (Ã©quipe) pour lever l'ambiguÃ¯tÃ© ; Session : score
      personnel sorti de score_label, colonne Score (Ã©quipe, inexistante lÃ ) masquÃ©e.
      Part Escouade `[!]` : SquadSynergyHistoryTable volontairement sans stats
      perso â€” hors Â« historique Â».)
      **Colonne Score personnel** dans les tableaux d'historique : vÃ©rifier
      d'abord quel(s) composant(s) constituent Â« l'historique Â» (Explorer +
      MatchHistoryPage â€” confirmer par grep avant code) ; champ contrat Ã  ajouter si
      absent d'`ExplorerMatchRow` (mutualiser avec la modification 2.4a) ; tri
      TanStack ; en-tÃªte FR/EN.

**Gate 2** : `make check-types` ; `make test-web` ; contrat rÃ©gÃ©nÃ©rÃ© (2.4/2.5) ;
`make go-api-test` si le backend a bougÃ© ; revue navigateur de chaque item.
> GATE 2 PASSÃ‰ le 2026-08-02 (orchestrateur) : go test ./... exit 0 (rÃ©serve agent
> tranchÃ©e flake), tsc exit 0, vitest 3285 (puis re-suite aprÃ¨s harmonisation
> registre), contrat 203/203 chemins, matrice navigateur Playwright : 2.1 cas
> extrÃªme compactÃ©, 2.2 â“˜ lÃ©gende Bonus, 2.3 â“˜ titre + courbe Ã©quipe Ã— 3 panneaux,
> 2.4 badges 19/20 + aide au libellÃ© + tri conservÃ©, 2.5 colonne triable.
> DÃ©couverte registre : 37 vouvoiements rÃ©siduels dans les manifests (dette mixte
> prÃ©existante) â€” candidat passe post-lot.

## Phase 3 â€” Features et unifications

- [ ] 3.1 **Suppression de mÃ©dias** (propriÃ©taire + admin, dÃ©finitive, confirmation) :
      (a) design sur piÃ¨ces AVANT code : modÃ¨le de stockage rÃ©el (fichiers disque +
      `media_likes` append-only sur shared_social) => sÃ©mantique de suppression des
      likes du mÃ©dia supprimÃ© compatible ADR 0022/0026 (pas d'UPDATE/DELETE sur
      tables critiques â€” Ã©vÃ©nements ou orphelins invisibles, dÃ©cision consignÃ©e ici) ;
      (b) gate admin : rÃ©utiliser le mÃ©canisme d'admin existant s'il y en a un
      (vÃ©rifier ce qui a Ã©tÃ© livrÃ© par `feat/admin-retours-diag`) â€” s'il n'existe
      pas, livrer propriÃ©taire seul et statuer la part admin `[!]` avec justification ;
      (c) endpoint DELETE gatÃ© auth + ownership (ratchet bare_routes), handler sans
      logique mÃ©tier, service + port ; (d) UI : action supprimer dans le visualiseur
      mÃ©dia + modale de confirmation, invalidation des query keys `mediaBase`.
      Tests httptest + service + `go test -tags=integration -p 1`. Contrat openapi.
- [ ] 3.2 **Kills vÃ©hicules H5** : VÃ‰RIFICATION PRÃ‰ALABLE BLOQUANTE â€” confirmer que
      les kills vÃ©hicule H5 portent un identifiant PAR vÃ©hicule (registre) et non le
      seul sentinel `VehicleWeaponID = 2` ; si tout s'Ã©crase sur le sentinel, STOP et
      rapporter Ã  l'utilisateur avant d'implÃ©menter (son exigence : le sous-niveau
      distingue chaque vÃ©hicule). Si OK : sortir `vehicle`/`turret` de
      `nonCombatFragClasses` (`internal/domain/frag_distribution.go:42-49`) pour le
      breakdown H5, libellÃ©s FR/EN via TOML mappings H5 si nouvelles clÃ©s, gardÃ© par
      les donnÃ©es (pas de branche slug). Tests analysis/domain.
- [ ] 3.3 **i18n source unique** : (a) vÃ©rifier `MULTI_TITLE_API_ENABLED` actif dans
      TOUS les environnements (compose prod + dÃ©mo, `.env.example`, CI, dev) puis le
      rendre invariant (retrait du flag, ou toujours-on documentÃ© kill-switch datÃ© â€”
      rÃ¨gle 11) ; (b) supprimer `features/home/fallback.i18n.ts`,
      `features/media/fallback.i18n.ts`, `features/prestige/fallback.i18n.ts` et les
      dictionnaires de `lib/i18n/metricLabel.ts` ; (c) migrer les consommateurs vers
      `useFieldLabel`/`useAssetLabel`/`useOutcomeLabel` (remonter dans le rendu les
      appels actuellement hors composants) en gardant `humanizeMetricKey` comme
      unique repli d'affichage ; (d) garde-rail (rÃ¨gle 6) : test grep interdisant les
      dictionnaires de libellÃ©s de field-keys hors TOML. Hors pÃ©rimÃ¨tre :
      `lib/skillTiers.ts` (couplage Go<->TS distinct, garde-rail dÃ©jÃ  existant).
- [ ] 3.4 **ImplÃ©mentation du choix d'artefact Rendement & RÃ©sistance** (gate 0.1).
- [ ] 3.5 **ImplÃ©mentation du choix d'artefact dominance V/D** (gate 0.2) â€” mÃªmes
      surfaces que les losanges actuels (Home, Timeseries, Relations, showcase Lab).

**Gate 3** : suite Go complÃ¨te `go test ./...` + `-tags=integration -p 1` exit 0 ;
`make go-api-lint` 0 issue nouvelle ; `make check-types` ; `make test-web` ; contrat +
types front rÃ©gÃ©nÃ©rÃ©s ; revue navigateur 3.1/3.2/3.4/3.5.

## Phase 4 â€” Ops prod et clÃ´ture

- [ ] 4.1 **Re-mesure lot K** (fetch films hors verrou, dÃ©ployÃ© le 26/07 mais jamais
      mesurÃ© en conditions rÃ©elles â€” 0 match ingÃ©rÃ© alors) : la session du 31/07
      fournit enfin la mesure. Compter dans les logs VPS les avertissements
      Â« writer RW tenu > seuil Â» par cycle d'auto-sync depuis le 31/07 (avant :
      3-5/cycle ; attendu : ~0). **PrÃ©avis utilisateur avant toute op VPS.**
- [ ] 4.2 **Cut de snapshot prod** : vÃ©rifier si le snapshot de lecture a Ã©tÃ© recoupÃ©
      depuis le correctif G1 (sinon le repli lecture live reste actif) ; recouper si
      nÃ©cessaire, avec prÃ©avis.
- [ ] 4.3 **MAJ Notion** : barrer chaque point traitÃ© avec commit + notes sous les
      points ; consigner les rÃ©ponses aux questions posÃ©es dans la section
      (prÃ©cision par arme = H5 dÃ©jÃ  couvert / Infinite au killsource ; Â« pourquoi
      cette reprÃ©sentation est une image Â» = rendu canvas ECharts sans symbole ni
      aide, corrigÃ© en 2.3) ; callout Suivi du lot 2.
- [ ] 4.4 **ClÃ´ture** : delivery-checklist (skill), tous les items de ce plan statuÃ©s,
      entrÃ©e thought_log, dÃ©couvertes consignÃ©es (section ci-dessous).
- [ ] 4.5 Rappel cÃ´tÃ© utilisateur : poser le **tag v7.3.0** quand tu considÃ¨res la
      v7.3 close (dÃ©clenche notification de release + Â« Quoi de neuf Â»). Le Replay 2D
      Ã©tant ton chantier, ce lot ne conditionne pas le tag.

**Gate 4** : `make gate-push` vert avant proposition de merge (merge main = deploy
prod auto â€” prÃ©venir l'utilisateur).

## Volet D â€” Branches Dependabot (orchestration, branches sÃ©parÃ©es du lot)

DÃ©tail des Ã©tapes : `.ai/PLAN_DEPS_ECHARTS_TS7_2026-07-27.md` (lots A/B/C, contrat
plan-execution propre). Ce volet ne duplique pas ces Ã©tapes : il fixe QUAND et QUOI
exÃ©cuter, avec l'Ã©tat constatÃ© le 2026-08-02. Chaque merge sur main = dÃ©ploiement prod
(prÃ©venir avant).

- [x] D1 **Triage des 2 nouvelles PR (#71, #72)** â€” fait le 2026-08-02 : CI re-vÃ©rifiÃ©e
      (14/14 checks verts chacune), #71 puis #72 mergÃ©es en squash avec CI main vÃ©rifiÃ©e
      verte aprÃ¨s chacune (go utilisateur reÃ§u). Incident intercalÃ© : la CI main est
      passÃ©e rouge aprÃ¨s #71 sur un TODO(expiry:2026-08-01) Ã©chu (bombe datÃ©e sans lien
      avec #71) + un flake persist qualifiÃ© (10/10 PASS local) â†’ hotfix PR #73 mergÃ©e
      (voir DÃ©couvertes). Deploys prod dÃ©clenchÃ©s et verts (#71, #73, #72).
- [x] D2 **Lot A â€” PR #67 (go-minor-patch, 10 paquets dont duckdb-go 2.10505)** â€”
      clos le 2026-08-02 (agent Opus A1-A6, orchestrateur A7/A8) : changelog duckdb-go
      2.10505/DuckDB 1.5.5 auditÃ©, AUCUNE mention ART/index ; fix allowlist = 2 entrÃ©es
      datÃ©es (`QUERY /static/*` + `QUERY /static/commendations/*`) ; gates locaux
      `go test ./...` et `-tags=integration -p 1` exit 0 (persist + sync verts) ;
      conflit avec le hotfix #73 rÃ©solu en reprenant la version main (TODO 2026-09-15) ;
      CI branche 15/15 verte, mergÃ©e squash sur go utilisateur, **CI main post-merge
      verte**, thought_log fait. `main` rapatriÃ©e dans la branche du lot.
- [ ] D3 **Lot B â€” echarts 5.6.0 â†’ 6.1.0 (CVE-2026-45249, XSS)** â€” effort moyen.
      DÃ©cision du 27/07 maintenue : ne pas re-diffÃ©rer une 3e fois. SÃ‰QUENCEMENT
      IMPOSÃ‰ : aprÃ¨s le merge du lot 2, depuis un `main` Ã  jour (branche
      `fix/echarts-6-security-bump`) â€” le harnais Playwright `toHaveScreenshot` (B2)
      doit capturer les visuels FINAUX, or 2.3 et 3.4/3.5 modifient des graphes dont
      `OutcomeSequenceTape`, qui fait partie des wrappers Ã  couvrir. ExÃ©cuter B1-B8 ;
      le gros du travail est le harnais, pas le bump ; sign-off visuel utilisateur (B7)
      requis.
- [ ] D4 **Lot C â€” TS7 (PR #70) : statuer, ne pas exÃ©cuter** â€” vÃ©rifiÃ© le 2026-08-02 :
      `typescript-eslint@latest` = 8.65.0, peer `typescript >=4.8.4 <6.1.0`, TS 7.0.2
      toujours HORS range (seules des 8.65.1-alpha existent). Report justifiÃ© `[!]`
      (dÃ©pendance externe). Ã€ la clÃ´ture du lot : revÃ©rifier une derniÃ¨re fois
      `npm view typescript-eslint@latest peerDependencies`, consigner le statut dans
      Notion, laisser la PR #70 ouverte avec un commentaire de blocage datÃ© si le range
      n'a pas bougÃ©. Ne PAS exÃ©cuter C2+ d'ici lÃ .

**Gate D** : gates propres Ã  chaque lot du plan dÃ©diÃ© (CI GitHub verte sur branche
AVANT merge, pas seulement un rejeu local).

---

## Hors pÃ©rimÃ¨tre (consignÃ©, ne pas traiter dans ce lot)

| Sujet | Statut |
|---|---|
| Replay 2D (tout le bloc) | Chantier utilisateur en cours |
| Killsource (branchement + validation prÃ©cision Infinite + assets icÃ´nes armes) | Re-diffÃ©rÃ© (dÃ©cision 2026-08-02) â€” branche `feat/killsource-prod`, entrer par `.ai/HANDOFF_KILLSOURCE_REPRISE.md` |
| Note de perf modes objectifs | Chantier isolÃ© futur â€” voir dÃ©cision verrouillÃ©e (subtilitÃ© Ã©crasement/participation) |
| 4 items `[POSTPONED]` de la section (assistants par kill, spartan abilities, profil Ascension, drawer lobby) | MarquÃ©s reportÃ©s par l'utilisateur |
| 2 items `[ATTENTE REPLAY 2D]` (replay unique dÃ©mo, niveaux de bleu cartes) | Attendent le Replay 2D |
| Colonne Replay dans l'historique | Ã€ la livraison du Replay 2D |
| TypeScript 7 (PR #70) | BloquÃ© par dÃ©pendance externe (typescript-eslint) â€” suivi en D4, exÃ©cution interdite tant que le range peer n'inclut pas TS7 |
| `lib/skillTiers.ts` (couplage Go<->TS) | Garde-rail existant, hors du pÃ©rimÃ¨tre i18n TOML |

## Protocole de reprise de session

1. Lire ce fichier : les statuts `[ ]`/`[x]`/`[~]`/`[!]` font foi.
2. `git log --oneline -10` sur `feat/v7.3-notion-lot2` (chaque item livrÃ© = commit
   dÃ©diÃ© rÃ©fÃ©rencÃ© dans la case).
3. `.ai/thought_log.md` (entrÃ©es 2026-08-XX du lot).
4. Interdiction de fixes opportunistes hors pÃ©rimÃ¨tre : toute dÃ©couverte va dans la
   section ci-dessous, rien d'autre.

## DÃ©couvertes en cours d'exÃ©cution (append-only)

- [2026-08-02, agent 0.2] **Contraste des marqueurs de dominance** : aucun token de
  dominance n'atteint 3:1 contre la couleur d'issue qu'il surmonte (domination 1,40:1,
  remontada 2,04:1, contre-remontada 1,47:1, humiliation 1,13:1, dÃ©bandade 1,03:1) â€”
  c'est le liserÃ© `tooltipBg` 1 px qui rend le losange actuel visible, pas sa couleur.
  Toute forme retenue en 3.5 doit conserver un liserÃ©/gouttiÃ¨re.
- [2026-08-02, agent 0.2] **Collision de tokens** : `--ac-outcome-dnf` et
  `--ac-narrative-humiliation` partagent exactement `#8B5CF6` (hors pÃ©rimÃ¨tre du lot,
  Ã  statuer plus tard).
- [2026-08-02, agent 0.2] `#00DC82` et `#33D6FF` (palette dominance) sortent de la
  bande de clartÃ© et tombent sous 3:1 contre la surface claire (1,75 / 1,66) â€” un
  marqueur de ces couleurs posÃ© sur le fond de page en thÃ¨me clair exige un cerne.
- [2026-08-02, agent 0.2] `EChartsThemeColors` (`lib/echarts/themeColors.ts`) n'expose
  aucune couleur de surface (`CHART_BG = 'transparent'`) : une dÃ©coupe couleur-fond en
  3.5 exigerait d'ajouter un champ (`--background`/`--card`), sinon seam gris
  `tooltipBg`.
- [2026-08-02, agent 0.2] Contrainte dirimante pour tout rail sous la bande : sur
  Relations le composant est montÃ© avec `height=64` et une grille 32+32 â†’ hauteur de
  plot nulle (aucune place sous les brackets).
- [2026-08-02, agent 0.1] **Palette joueurs non discriminable** (tous les charts
  multi-joueurs Escouade) : Î”E 6,7 entre `narrative-dominant` et `divergent-pos`
  (deux verts quasi identiques, plancher 15), Î”E 5,7 en protanopie entre
  `divergent-pos` et `perf-tier-3` ; contraste < 3:1 sur fond de carte en clair pour
  les 4 tokens joueur.
- [2026-08-02, agent 0.1] **Deux dÃ©finitions du Â« rendement Â»** : la carte Rendement
  trace `damage_dealt / kills` (sans assistances) alors que `match-card.tsx:103`
  utilise `effectiveDmgPerFrag(...)` alignÃ© ADR 0006 â€” deux Ã©crans, deux nombres
  sous le mÃªme mot. Ã€ trancher en 3.4.
- [2026-08-02, agent 0.1] `rendement_offensif`/`resistance_defensive` sont servis
  dans le payload mais ne servent qu'Ã  `hasEfficiencyData()` â€” le graphe recalcule
  autre chose ; un joueur `damage_dealt=0` passe le test sans courbe offensive.
- [2026-08-02, agent 0.1] `divergent-neutral` est un bleu (#60A5FA) utilisÃ© comme
  point milieu du dÃ©gradÃ© rougeâ†”vert (`oneLifeDamageGradient.ts`) â€” il manque un
  token gris neutre pour les encodages divergents.
- [2026-08-02, agent 0.1] `DefensiveResistanceP80 = 1,65` est une constante Go alors
  que le pendant offensif est dÃ©clarÃ© par titre dans `constants.toml` (asymÃ©trie
  title-agnostic).
- [2026-08-02, agent 1.3/1.4] **Le garde-rail du fragment timezone n'existe pas** :
  `player_matches_repo.go:145-146` rÃ©fÃ©rence `analysis/start_time_canonical_test.go`
  qui est introuvable â€” rien n'interdit de rÃ©Ã©crire le fragment Ã  la main (c'est
  ainsi que le bug 1.3 est nÃ©). Recommandation : ratchet grep sur `start_time_utc`
  hors helper (candidat passe post-lot, rÃ¨gle 6 CLAUDE.md).
- [2026-08-02, agent 1.3/1.4] Le seed Infinite Ã©crit toujours des libellÃ©s FR dans
  `citation_mappings.category` (neutre : normalisation Ã  la lecture) â€” Ã©crire les
  clÃ©s stables serait une migration de donnÃ©es, hors lot.
- [2026-08-02, agent 1.3/1.4] `CitationFullMapping.Category` (`sync/citations.go`)
  est scannÃ© mais jamais consommÃ© â€” champ mort candidat au nettoyage (rÃ¨gle 7).
- [2026-08-02, agent 1.3/1.4] `routeTree.gen.ts` se fait rÃ©gÃ©nÃ©rer (rÃ©ordonnancement
  603 lignes) au passage de tsc/vitest â€” dÃ©calage de version du plugin TanStack
  Router Ã  investiguer (bruit de diff rÃ©current).
- [2026-08-02, agent 1.5/1.6] **Les tests du chemin atomique des likes sont
  dÃ©sactivÃ©s depuis le 2026-05-15** (`media_service_atomic_integration_test.go`,
  tag `atomic_legacy`) : le chemin nominal de prod n'avait AUCUN test actif â€”
  c'est ce qui a laissÃ© passer les rÃ©gressions successives. Recommandation :
  rÃ©activer ou supprimer (anti-pattern dead code museum).
- [2026-08-02, agent 1.5/1.6] **`/players/{player_slug}` sans RequireAuth** :
  l'ownership laisse passer `sess == nil` sur toutes les routes joueur. TraitÃ©
  pour le like uniquement (401 gatÃ©) â€” audit des autres routes du groupe Ã 
  recommander.
- [2026-08-02, agent 1.5/1.6] Like historique orphelin (event du 26/04/2026 en
  chemin absolu, sans correspondance `media_files`) â€” rÃ©parable par INSERT d'un
  event sous la clÃ© canonique.
- [2026-08-02, agent 1.5/1.6] **`media_files.liked` est global** (partagÃ© entre
  tous les viewers) alors que `media_likes_history` est par liker â€” le cÅ“ur d'un
  mÃ©dia est commun Ã  tous. Ã€ trancher avec l'item 3.1 (suppression de mÃ©dias).
- [2026-08-02, agent 1.5/1.6] Placeholder Â« DÃ©fi Â» en dur non i18n
  (`HomeChallengesList.tsx:119`) ; compat `LegacyMediaItemRow` sans date
  d'expiration (`features/media/queries.ts`).
- [2026-08-02, agent 1.5/1.6] Flakes qualifiÃ©s (verts au rejeu isolÃ©) :
  `TestStartImport_HappyPathReturns202WithJobID` (cleanup TempDir Windows),
  `TestCareerLive_NilAPIResponse_NotCached` (timeout 2 s sous charge).
- [2026-08-02, agent 2.1-2.3] `TimeseriesKdaTrend.tsx` est le pendant exact de 2.2
  cÃ´tÃ© Timeseries (sÃ©rie Bonus masquÃ©e, libellÃ© 'Bonus' en dur non i18n, aucune
  aide) â€” candidat passe post-lot, mÃªme patron SquadToggleLegendChart.
- [2026-08-02, agent 2.1-2.3] Le libellÃ© Â« Bonus Â» sert aussi de clÃ© de masquage
  dans les builders (`hiddenTypes.has('Bonus')`) : couplage clÃ©/label documentÃ© en
  commentaire, casserait si quelqu'un localisait la clÃ© â€” garde-rail grep candidat.
- [2026-08-02, agent 2.1-2.3, TRAITÃ‰ES en fold par l'orchestrateur (dÃ©lÃ©gation
  dÃ©couvertes)] : 3e instance du tooltip d'intensitÃ© (Timeseries) alignÃ©e sur le
  registre simple ; 3e surface FDA gap (SquadFdaGapCumulativeCard) branchÃ©e sur
  FdaGapTooltipText â€” 2 registres ne coexistent plus sur les mÃªmes charts.
- [2026-08-02, agent 2.4/2.5] Le schÃ©ma OpenAPI `ExplorerMatchRow` est un ORPHELIN
  (aucun path ne le rÃ©fÃ©rence) et diverge du `ExplorerMatchesRow` rÃ©ellement servi
  â€” candidat suppression du fragment manuel.
- [2026-08-02, agent 2.4/2.5] `wire.csrBadgeURL` reste une implÃ©mentation parallÃ¨le
  de la normalisation tier/sous-palier non migrÃ©e vers `analysis.SkillBadgeURL`
  (sert la CarriÃ¨re, ne capitalise pas le tier) â€” Ã  basculer dans une passe dÃ©diÃ©e
  avec test de non-rÃ©gression (rÃ¨gle 6 : 2 copies max atteintes).
- [2026-08-02, agent 2.4/2.5] Tri non atteignable au clavier sur MatchScoreboard,
  MatchEncountersTable, DetectionsPanel (onClick sur th sans bouton, dette
  prÃ©existante) ; clÃ©s i18n mortes `explorer.matches.col_tier` / `col_delta_rank`.
- [2026-08-02, orchestrateur] RetombÃ©e du fix 1.3 corrigÃ©e par l'orchestrateur :
  `TestCareerRepo_GetRelationsHeatmap` (integration) attendait des heures UTC et
  dÃ©pendait du fuseau machine â†’ ouvert via le chemin production Ã©pinglÃ© UTC
  (les non-UTC restent couverts par les tests dÃ©diÃ©s 1.3).
- [2026-08-02, orchestrateur] **CI main rouge post-merge #71** (run 30745523574),
  2 causes sans lien avec #71 : (1) `TODO(expiry:2026-08-01)` Ã©chu dans
  `season_pass_repo_tracks.go:297` â†’ hotfix PR #73 (Ã©chÃ©ance 2026-09-15, critÃ¨re
  mesurable : 0 ligne kind='track-def' dans asset_index prod, Ã  vÃ©rifier avec les
  ops Phase 4) ; (2) `TestWorker_Run_PersistsAndACKs` = flake qualifiÃ© (10/10 PASS
  local `-tags=integration`). Merge #72 suspendu jusqu'Ã  CI main verte.

## DÃ©cisions d'artefacts (Ã  remplir au gate 0)

- Rendement & RÃ©sistance : (en attente)
- Marqueurs dominance V/D : (en attente)

