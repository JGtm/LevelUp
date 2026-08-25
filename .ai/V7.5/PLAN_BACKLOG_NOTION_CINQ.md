# PLAN — Backlog Notion : les 5 points (assistances, colonne rejeu, usages d'équipement, présence en jeu, 3 onglets match)

> Écrit le 2026-08-24. Branche : `wt/notion-cinq` (worktree unique
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-notion-cinq`, base `a995cf45d` =
> feat/v75). Contrat d'exécution : skill `plan-execution` — ordre strict, un lot à la
> fois, statuts `[x]` fait / `[~]` couvert ailleurs (référence) / `[!]` non traité
> (justification), aucune case vide à la clôture d'un lot. Zéro fix hors périmètre :
> toute trouvaille va en section « Découvertes ». Les exécuteurs ne touchent NI
> `.ai/thought_log.md` NI `.ai/V7.5/REGISTRE_REPORTS.md` (réservés au superviseur).
> JAMAIS `git add -A` — ajouter les fichiers nommément. Pas de push.

## Décisions produit TRANCHÉES (utilisateur, 2026-08-24)

- **D-P5** : page match découpée en **3 onglets « Général / Chronologie / Joueurs »**
  (option 3 du questionnaire).
- **D-P4** : compteur « amis en jeu » = la liste **`friend_gamertags` des Réglages**
  (pas PeopleHub).
- **D-P1** : graphe des assistances = **assistant → tueur assisté**, plus la métrique
  « kills volés » (assists où `assist_damage_pct > killer_damage_pct`).

## Ordre des lots (strict — lot N clos et gate passé avant lot N+1)

| Lot | Point Notion | Nature | Dépend de |
|---|---|---|---|
| A | 2 — colonne rejeu + filtre Explorer | Go + web | — |
| B | 5 — 3 onglets page match | web pur | — |
| C | 1 — assistances (match view + escouade) | Go + web | B (onglet Joueurs) |
| D | 3 — usages d'équipement (film) | web pur | B (onglet Chronologie) |
| E | 4 — présence en jeu | Go + web | — |

« Clos » = tous les items du lot statués, gate du lot vert, commit(s) posés sur
`wt/notion-cinq` avec message `notion5(<lot>): ...`.

## Règles transverses (rappel — le code du dépôt fait foi)

- i18n : toute string UI en FR **et** EN, parité par typage `Record<Locale, T>` ; FR
  sans anglicismes (« rejeu », pas « replay », dans l'UI FR).
- Couleurs : tokens sémantiques uniquement (`tokenCssVar`), zéro hex/classe Tailwind
  couleur dans `features/`.
- Query keys : `apps/web/src/lib/query/keys.ts`, jamais inline. Routes file-based,
  ne jamais éditer `routeTree.gen.ts`.
- Go : capabilities, jamais `slug ==` ; logging `slog.*Context` structuré ; types de
  réponse dans `internal/domain/` ; SQL dans `platform/duckdb/` ; pas de logique
  métier dans les handlers.
- Lectures kill events : vue `match_kill_events_latest` UNIQUEMENT (ADR 0026).
- `openapi.yaml` est GÉNÉRÉ : après tout changement de contrat, régénérer +
  `make generate-types` ; le gate contracttest openapi doit rester vert.
- Pas d'emojis dans les fichiers versionnés. Seuils : fichier ≤ 500 L, fonction
  ≤ 80 L (dette existante gelée : ne pas l'accroître).
- INTERDIT à tous les lots : `apps/go-api/internal/analysis/filmdec/` (worktree
  `wt/usages-equipement` en vol y travaille), `SchemaVersion` du document de rejeu
  (reste 18), toute re-cuisson d'artefact, toute nouvelle table DuckDB.

---

## LOT A — Point 2 : colonne rejeu dans les tableaux de matchs + filtre Explorer

État de l'art (recherche 2026-08-24) : le composant `ExplorerMatchesTable`
(`apps/web/src/features/explorer/ExplorerMatchesTable.tsx`) sert 5 pages (Explorer
matchs, Explorer mode joueur, Sessions, Carrière « matchs marquants », Timeseries
Progression) ; le tableau escouade `SquadSynergyHistoryTable` est une implémentation
séparée. Le seul lien vers le rejeu est `MatchHeader.replayLink.tsx` (page match).
Disponibilité d'un artefact : fichier `data/cache/replays/{titleSlug}/{short8}.json`,
`replay_service.go` (`IsAvailable` = os.Stat unitaire) ; AUCUNE table registre — et on
n'en crée pas (précédent : décision D2 de `PLAN_EXPLOITATION_REGISTRE_FILM.md`).
Précédent de listing en masse : `replay_purge_cron.go:137` (`os.ReadDir` unique).

### Items

- [x] A1 — Go : méthode bulk sur le service replay (`internal/service/replay_service.go`
  + port) : `AvailableSet(ctx) (map[string]struct{}, error)` — UN `os.ReadDir` de
  `ReplayArtifactsDir(titleSlug)`, clés = short8 (`FilmShortMatchID`). Chemins via
  `PathResolver` uniquement. Erreur : logger `slog.ErrorContext` puis set vide
  (dégradation propre, pas de 500).
  → Type nommé `port.ReplayAvailability` (`internal/port/replay_availability.go`) au
  lieu du `map[string]struct{}` nu : il porte `Has(matchID)` qui normalise le match_id
  complet en forme courte, une seule fois, pour tous les appelants.
- [x] A2 — Go : `HasReplay bool \`json:"has_replay,omitempty"\`` sur
  `domain.MatchHistoryRow` (au niveau de `MatchURL`) ; câblage
  `MatchHistoryService` : nouveau `WithReplay(...)`, `rowFormatters.hasReplay`
  (lookup O(1) dans le set construit UNE fois par requête dans `GetPage`, jamais un
  Stat par ligne) ; wiring dans `MatchHistoryCtx`
  (`internal/api/wire/registry_pages_home.go`) sur le modèle de `MatchViewCtx`
  (`registry_pages.go:112-115`).
  → `rowFormatters()` prend désormais l'ensemble en paramètre ; l'export CSV passe
  `nil` (pas de colonne rejeu dans le CSV → aucun listing payé).
- [x] A3 — Go : propagation `ExplorerMatchesRow.HasReplay` dans
  `BuildExplorerRowFromMatchHistory` (`handlers/projections.go`), comme `MatchURL`.
- [x] A4 — Go : filtre Explorer `replay_scope` (3 états `''|'with'|'without'`, patron
  exact du `squadScope`) : champ dans `ExplorerMatchesQueryRequest`
  (`domain/explorer.go`) + branche dans `applyExplorerMatchFilters`
  (`match_history_service_filters.go`). Le filtre s'applique côté Go, comme les 7
  existants.
  → L'ensemble est passé en PARAMÈTRE explicite à `applyExplorerMatchFilters` et aux
  5 compteurs cascade (`computeAvailable*`), pas caché dans le DTO de requête : les
  counts des autres dimensions restent cohérents quand le filtre rejeu est actif. Le
  filtre vit dans `match_history_filter_replay.go` (le fichier filtres était au seuil
  des 500 lignes).
- [x] A5 — Go : `SquadMatchHistoryRow.HasReplay` (`domain/teammates.go:344-369`) +
  câblage dans le service teammates (deuxième point d'ajout, service séparé).
- [x] A6 — contrats : régénérer `openapi.yaml` + `make generate-types` ; gate openapi
  contracttest vert. → `has_replay` publié sur les 3 schémas de ligne ; `replay_scope`
  n'apparaît pas au contrat car les request bodies `RawBody` ne sont décrits que
  partiellement par le fragment manuel (même traitement que `squad_scope` et les 6
  autres filtres Explorer — cf. Découvertes).
- [x] A7 — web : colonne « Rejeu » dans `ExplorerMatchesTable` (dans `baseColumns`,
  `columnHelper.display`) : icône `themedIconSrc('replay', theme)` + `<Link>` interne
  typé vers la route `.../matches/$matchId/replay` (gabarit STRUCTUREL = colonne
  Waypoint `ExplorerMatchesTable.tsx:406-441`, mais lien interne, pas externe) ;
  rien rendu si `has_replay` faux. En-têtes i18n FR « Rejeu » / EN « Replay ».
  → i18n via le MANIFESTE `lib/i18n/manifests/explorer.toml` (mécanisme de la feature
  Explorer, ADR 0003), pas un `i18n.ts`. L'en-tête de la colonne est VIDE, comme celui
  de son voisin Waypoint (colonne d'icône de 20 px) : le couple FR/EN vit donc dans
  l'aria-label + infobulle `explorer.matches.col_replay_aria` (« Ouvrir le rejeu 2D du
  match » / « Open the 2D replay of the match »), et la clé `col_replay` (« Rejeu » /
  « Replay ») a été retirée du manifeste faute d'appelant (règle 0 code mort).
- [x] A8 — web : même colonne dans `SquadSynergyHistoryTable`.
  → Cellule factorisée dès la 2e copie dans `lib/match-nav/MatchReplayLink.tsx`
  (partagé par les deux tableaux) ; i18n squad = `history.replayAriaLabel` (FR/EN).
- [x] A9 — web : filtre « Rejeu » dans la page Explorer : champ `replayScope` dans
  `ExplorerScope` + `EncodedExplorerScope` + encode/decode + `EXPLORER_URL_KEYS` +
  `explorerSearchSchema` + contrôle UI (3 états : tous / avec rejeu / sans rejeu),
  patron du contrôle `squadScope` existant ; branché dans la requête
  `matches-query`. → param URL `replay`. Sans compte par option (le backend n'expose
  pas de dimension cascade pour le rejeu, cf. A4).
- [x] A10 — tests : Go = test unitaire du set bulk (dossier fixture) + test du filtre
  `replay_scope` dans les tests du service match_history existants ; web = test du
  rendu conditionnel de la colonne + encode/decode du scope (vitest).
  → Go : `replay_service_test.go` (3 tests set bulk : listing/intrus, dossier absent,
  isolation par titre) + `match_history_replay_test.go` (has_replay, UN SEUL listing
  par requête, 3 états du filtre, double dégradation, ensemble vide). Web :
  `ExplorerMatchesTable.test.tsx` + `SquadSynergyHistoryTable.test.tsx` (rendu
  conditionnel) + `explorerScope.test.ts` (encode/decode/schéma 3 états).

Hors périmètre déclaré : cartes de la Home (`match-card.tsx` — pattern cards, pas un
tableau) → Découvertes si jugé souhaitable plus tard.

### Gate A

```
cd apps/go-api && go vet ./... && go test ./internal/service/... ./internal/api/... ./contracttest/...
cd apps/web && npx tsc -b --force && npx eslint <fichiers touchés> && npx vitest run <tests touchés + explorer>
```
0 erreur, 0 nouveau warning eslint.

---

## LOT B — Point 5 : la page match passe à 3 onglets « Général / Chronologie / Joueurs »

État de l'art : onglets = query param `tab` validé dans le layout
`routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId.tsx`
(`z.enum(['summary','details']).catch('summary')`) ; lu dans
`features/match-view/MatchViewPage.tsx` (523 L) l.110-118, barre l.309-328, contenu
`summary` l.331-400, `details` l.401-519 en 4 sous-sections titrées (`DetailSection`).
Précédent 3 onglets par query param : `stats/timeseries.tsx` ; précédent d'alias de
deep-links : `features/settings/tabs.ts` (`TAB_ALIASES`).

### Répartition décidée (D-P5)

- **Général** (`summary`) : INCHANGÉ.
- **Chronologie** (`chronology`) : les blocs de l'actuelle sous-section « Déroulé du
  match » — `MatchImpactBadgesBar` + `MatchKDCumulChart`, `MatchScoreCurveChart`,
  `MatchTugOfWarChart` + `MatchCadenceChart`, `MatchPositionsHeatmap`,
  `EngagementMatchSection`.
- **Joueurs** (`players`) : « Duels & confrontations » (`MatchNemesisCards`,
  `MatchAntagonistChart`, `MatchFragDiffChart`) + « Tableau des scores »
  (`MatchScoreboard` avec objectifs) + « Historique des rencontres »
  (`MatchEncountersTable`).

### Items

- [x] B1 — layout : `z.enum(['summary','chronology','players'])`, avec rétro-compat
  des deep-links : `tab=details` accepté et résolu vers `chronology` (alias au
  décodage, patron `TAB_ALIASES` de settings ; pas de redirect).
  → Ids + alias + résolveur dans `features/match-view/tabs.ts` (source unique
  partagée route/page, patron `features/settings/tabs.ts`). La route garde
  `matchViewTabSchema.optional().catch((ctx) => resolveMatchViewTab(ctx.value))` :
  `tab` reste OPTIONNEL (aucun `?tab=` ajouté aux liens match existants), une
  valeur inconnue retombe sur `summary`, `details` sur `chronology`.
- [x] B2 — `MatchViewPage.tsx` : table `TABS` à 3 entrées ; les CONTENUS des onglets
  Chronologie et Joueurs sont EXTRAITS en deux composants
  (`MatchViewTabChronology.tsx`, `MatchViewTabPlayers.tsx`) — `MatchViewPage.tsx`
  doit passer SOUS 500 lignes à la clôture (il est à 523 : l'extraction paie la
  dette au passage, sans changer le rendu des blocs déplacés).
  → 523 → **420 lignes**. Blocs déplacés à l'identique (JSX et commentaires) ;
  le helper `DetailSection` sort dans son propre fichier `DetailSection.tsx` pour
  être partagé par les deux onglets sans cycle d'import vers la page. Le ternaire
  summary/details devient trois blocs `{activeTab === '…' && (…)}` (aucun nœud DOM
  supplémentaire — le test de structure de l'onglet Général passe inchangé).
- [x] B3 — requêtes : `useMatchObjectiveEvents` et `useMatchPositions` ne doivent
  être actives que quand leur onglet les affiche (aujourd'hui tirées dès l'arrivée
  pour des composants de Détails ; utiliser `enabled:` selon l'onglet actif —
  l'artefact de rejeu et engagement sont déjà conditionnels par le rendu).
  → Vérifié sur pièces : `objectiveEvents` n'a que deux consommateurs
  (`MatchKDCumulChart`, `MatchTugOfWarChart`) et `matchPositions` un seul
  (`MatchPositionsHeatmap`) — tous les trois dans Chronologie. 3e paramètre
  `enabled = true` sur les deux hooks (`enabled: enabled && !!playerSlug && !!matchId`),
  passé `activeTab === 'chronology'` par la page.
- [x] B4 — i18n : `tabChronology` FR « Chronologie » / EN « Timeline »,
  `tabPlayers` FR « Joueurs » / EN « Players » ; `tabDetails` supprimé des DEUX
  tables (0 code mort) ; les sous-titres de sections conservés tels quels dans
  leurs onglets.
  → Pas de contrat i18n typé pour match-view (le seul `i18nContract.ts` du dépôt
  est celui de match-replay) : la parité FR/EN est portée par le typage
  `Record<MatchViewLocale, MatchViewText>`, `tabDetails` retiré de l'interface ET
  des deux tables. Commentaire « Sections de l'onglet Détails » corrigé (doc
  inversée) en « Sections des onglets Chronologie et Joueurs ».
- [x] B5 — tests : vitest — l'alias `details` → onglet Chronologie actif ; chaque
  onglet rend ses sections attendues (smoke par titre de section) ; typecheck.
  → `MatchViewTabs.test.tsx` (14 tests) : résolveur d'alias, schéma de recherche
  RÉEL de la route (`Route.options.validateSearch.parse({tab:'details'})` →
  `chronology`), barre à 3 onglets FR sans « Détails », contenu par onglet (smoke
  par titre de section + testids des blocs), et les trois cas d'activation des
  deux calques de film (Général/Joueurs désactivés, Chronologie activé).
- [x] B6 — NE PAS brancher `MatchNarrativeSection.tsx` (composant orphelin constaté)
  → Découvertes. → Re-vérifié : aucun import hors de son propre test ; non touché.

### Gate B

```
cd apps/web && npx tsc -b --force && npx eslint <fichiers touchés> && npx vitest run src/features/match-view
```

---

## LOT C — Point 1 : « qui est l'assistant de qui » — page match + page escouade

État de l'art : les trois identités (assistant nommé, tueur crédité, victime) sont
sur LA MÊME LIGNE de `match_kill_events_latest`, avec `killer_damage_pct` /
`assist_damage_pct` (entiers, NON bornés à 100 — ne jamais plafonner). Doctrine des
3 états : `assist_known=FALSE` = ON NE SAIT PAS ; `TRUE`+NULL = mesuré sans
assistant ; `TRUE`+nommé = l'assistant. Couverture ≈ 50 % des matchs (films expirés
— plafond définitif) : l'UI doit distinguer « non mesuré » de « 0 assistance ».
Patron chart : `MatchAntagonistChart.tsx` + `antagonistStackedSeries`
(`_chartSeries.ts:37-89`). Patron requête : Q21c (`queries_match.go:537-556`).
Patron agrégat multi-matchs scopé : `Q28RelationsScopedTpl` / `Q32cSquadKVPairsTemplate`.

### Items — page match

- [x] C1 — Go SQL : nouvelle requête (Q21d) sur `match_kill_events_latest` :
  `WHERE match_id = ? AND publishable AND assist_known AND assist_gamertag IS NOT
  NULL AND assist_xuid IS NOT NULL`, `GROUP BY assist_xuid, assist_gamertag,
  feed_killer_xuid` avec `COUNT(*)` et `COUNT(*) FILTER (assist_damage_pct >
  killer_damage_pct)` (kills volés). Agrégat par match_id direct — PAS de clé
  temporelle, donc pas de correction T0.
  → Q21d et son lecteur vivent dans un fichier DÉDIÉ
  (`platform/duckdb/match_view_repo_assist_pairs.go`) : `queries_match.go` (626 L) et
  `match_view_repo_extras.go` (530 L) sont tous deux au-delà du seuil des 500 lignes
  (dette gelée). Un renvoi de 3 lignes reste dans `queries_match.go` après Q21c pour
  la découvrabilité de la série Q. La requête porte DEUX dénominateurs et non un :
  `match_deaths` (toutes lignes du match) ET `measured_deaths`
  (`publishable AND assist_known`), joints aux paires par `LEFT JOIN … ON TRUE` —
  sans cette jointure, un match « mesuré, zéro assistant » ne rendrait AUCUNE ligne
  et le contrat perdrait la distinction au moment exact où elle est nécessaire.
  `feed_killer_xuid IS NOT NULL` ajouté aux filtres du plan (tueur BOT : une paire
  sans destinataire nommable, écartée au SQL et jamais normalisée en chaîne vide).
- [x] C2 — Go domain/service : type `MatchAssistPair` (`assist_xuid`,
  `assist_gamertag`, `killer_xuid`, `killer_gamertag`, `assist_count`,
  `stolen_count`) sur le modèle de `MatchKillerVictimPair` ; nouveau bloc
  `combat_tab.assist_pairs` + indicateur de mesure distinct (le contrat DOIT
  permettre de distinguer « non mesuré » de « mesuré, zéro paire » — s'aligner sur
  le patron de couverture du kill feed existant ; attention au piège huma
  nullable-arrays déjà documenté dans le dépôt). Gamertags du tueur résolus via le
  scoreboard comme `buildKillerVictimPairs`
  (`match_view_builders_combat.go:159-218`). Loader dans
  `match_view_data_loaders.go` à côté de `killAssists`.
  → Bloc = OBJET `*MatchAssistPairs { measured_deaths, pairs }` et non deux champs
  frères : c'est ce qui donne les TROIS états — bloc absent (aucune ligne de film →
  l'UI ne rend rien, ce qui couvre aussi le titre sans décodeur), `measured_deaths`
  à 0 (« non mesuré »), `pairs` vide avec `measured_deaths` > 0 (« aucune
  assistance »). Le seuil d'émission du bloc est `match_deaths > 0`, décidé par
  `buildAssistPairs` (`service/match_view_builders_assists.go`, fichier dédié —
  `match_view_builders_combat.go` est à 432 L). Piège huma CONFIRMÉ sur pièces :
  toute tranche Go sort `T[] | null` en TS (`pairs` compris) — comblé À LA
  FRONTIÈRE du composant, une seule fois. Un tueur absent du scoreboard garde son
  xuid et un `killer_gamertag` VIDE (aucun nom inventé, aucun xuid recopié dans un
  champ de nom).
- [x] C3 — contrats : openapi + `make generate-types` ; gate openapi vert.
  → `make openapi-gen` (+42 lignes : `MatchAssistPair`, `MatchAssistPairs`,
  `assist_pairs`) puis `make generate-types` ; `make openapi-check` vert (document à
  jour ET `generated.ts` dérivé). Ré-exports `MatchAssistPair` / `MatchAssistPairs`
  ajoutés à `lib/api/types.ts` SANS réécrire le `pairs: […] | null` du contrat.
- [x] C4 — web : `MatchAssistChart.tsx` (clone structurel de
  `MatchAntagonistChart`) : 1 barre par ASSISTANT, segments = tueurs assistés ;
  infobulle : nb d'assists + « dont volés : N » ; état vide « Assistance non
  mesurée pour ce match » ≠ « Aucune assistance » selon l'indicateur C2 ; série
  via un `assistStackedSeries` dans `_chartSeries.ts` (même tri, même palette de
  tokens). Monté dans l'onglet **Joueurs** (lot B), sous `MatchAntagonistChart`.
  i18n FR/EN complet.
  → L'infobulle a exigé UNE prop optionnelle sur le wrapper partagé
  `components/charts/BarStackedChart.tsx` : `tooltipComponentNote(category,
  component)`. Aucun appelant existant n'est affecté (sans la prop, le formateur
  natif d'ECharts reste en place ; le formateur personnalisé ne s'active que sur
  `tooltipHideZero` — comportement d'avant — ou sur la nouvelle note). README des
  wrappers mis à jour dans le même commit. Palette `ASSIST_TOKENS` = les mêmes 11
  tokens sémantiques que le graphe des antagonistes (aucune couleur en dur), pour
  qu'un joueur garde sa teinte d'un graphe à l'autre. i18n : `assistTitle`,
  `assistNotMeasured`, `assistNoData`, `assistStolenNote(n)` — FR « Éliminations
  volées » sans anglicisme (« dont N volée(s) » à l'infobulle), EN « stolen ».
- [x] C5 — web : ne PAS toucher au kill feed existant ni à `assist_state` (livré).
  → Vérifié sur pièces à la clôture : `git diff -- '*killfeed*' '*KillFeed*'` vide,
  aucune occurrence d'`assist_state` dans le diff. Le graphe lit un bloc SÉPARÉ
  (`assist_pairs`), jamais les events décorés du feed.

### Items — page escouade

- [x] C6 — Go : agrégat par paire scopé aux matchs de l'escouade dans le service
  teammates (patron Q32c/Q28Scoped) : par (assistant, tueur) au sein de l'escouade
  → `assist_count`, `stolen_count`, plus dénominateur de couverture
  `matches_measured` / `matches_total` (nb de matchs de la sélection ayant au
  moins une ligne `assist_known=TRUE`). Bloc ajouté à `TeammatesPageResponse`.
  → Q32d (`queries_squad.go`, à côté de Q32c) + lecteur
  `SquadRepo.LoadSquadAssistPairs` (fichier dédié, `squad_repo.go` est à 444 L).
  DEUX décisions non écrites au plan, toutes deux tranchées par la question posée :
  (a) les DEUX joueurs sont contraints à l'escouade — une assistance rendue à un
  allié de passage ne parle pas de l'escouade et gonflerait le dénominateur de la
  colonne « part » ; (b) la requête ne rend AUCUN gamertag — les deux xuids sont au
  roster de la page, dont les noms sont déjà résolus (alias compris) ; reprendre le
  nom écrit dans le film ferait apparaître un même joueur sous deux orthographes
  dans un seul tableau. `matches_total` n'est pas redemandé à la base : c'est la
  taille de la sélection, connue de l'appelant. Périmètre repris de `firstBloodScope`
  (helper partagé, mêmes matchs et mêmes joueurs que les blocs voisins). Bloc nil
  quand `matches_measured` vaut 0 — ce qui couvre le titre sans décodeur de film,
  par la DONNÉE et jamais par un test sur le slug. `TotalAssists` publié comme
  dénominateur de la part (le front ne le dérive pas de l'affichage).
- [x] C7 — web : tableau dans la page **Synergies** de l'escouade (TanStack Table) :
  colonnes assistant, bénéficiaire, assists (brut), part (% des assists mesurées
  de l'escouade), kills volés ; bandeau de couverture « mesuré sur N des M matchs »
  (patron `killFeedWeaponCoverage`). i18n FR/EN. Bascule %/brut : les DEUX colonnes
  affichées (pas de toggle).
  → `features/squad/SquadAssistPairsTable.tsx`, gabarit structurel
  `SquadSynergyHistoryTable` (mêmes classes d'en-tête et de cellule, mêmes helpers de
  tri `explorerMatchesClientSort`, même `HeaderLabelTooltip`) — sans pagination : une
  escouade a au plus quelques dizaines de paires. Le bandeau de couverture n'existait
  PAS côté web (aucun patron à reprendre : `killFeedWeaponCoverage` est un compteur de
  LOG Go, pas un composant) : rendu en `text-xs text-muted-foreground` au-dessus du
  tableau, et MAINTENU sur l'état vide — sans lui, « aucune assistance » se lirait
  « rien mesuré ». Libellé produit FR « Éliminations volées » / EN « Stolen kills ».
  Pourcentage via `Intl.NumberFormat(style: 'percent')` : le FR met une espace
  insécable avant le « % », pas l'EN — un `${x} %` en dur serait faux dans une des deux
  langues. Monté dans `SquadSynergiesPage` après le tableau d'historique ; bloc absent
  → section non montée (aucun cadre vide).
- [x] C8 — tests : Go = test du builder des paires (fixtures avec les 3 états
  d'assist ; kills volés ; unmeasured vs zéro) + test service teammates du bloc ;
  web = vitest chart (série, état vide double) + tableau escouade (couverture
  affichée).
  → Go, 21 tests : `platform/duckdb/match_view_repo_assist_pairs_test.go` (7 — Q21d
  sur le SCHÉMA DE PRODUCTION via `migration.EnsureMatchKillEvents`, pas une table de
  circonstance : la lecture passe par la vue `_latest` et son QUALIFY ; couvre les
  3 états, non publiable, part manquante, part à 228 non plafonnée, tueur bot),
  `platform/duckdb/squad_repo_assist_pairs_test.go` (5 — paires internes seules,
  couverture partielle, sélection vide, placeholders),
  `service/match_view_builders_assists_test.go` (3 — bloc absent, « non mesuré » vs
  « zéro », gamertag du tueur depuis le scoreboard sans invention),
  `service/teammates/teammates_squad_assist_pairs_test.go` (6 — couverture en matchs,
  rien mesuré, mesuré sans paire, paire hors roster exclue DU TOTAL, périmètre vide,
  repo absent/en erreur). Web, 18 tests : `MatchAssistChart.test.tsx` (11 — les trois
  états vides, `pairs: null`, agrégation par assistant, ennemis en tête, repli du nom
  masqué, table des volées) et `SquadAssistPairsTable.test.tsx` (7 — couverture
  affichée y compris sans paire, 5 colonnes FR, part sur le total SERVEUR, tri, EN).

Dégradation multi-titre : la table est peuplée par le pipeline film Halo Infinite ;
pour un titre sans données le bloc est absent → l'UI ne rend rien (double porte
donnée non vide). Pas de nouvelle capability requise ; ne pas brancher sur le slug.

### Gate C

```
cd apps/go-api && go vet ./... && go test ./internal/service/... ./internal/platform/duckdb/... ./internal/api/... ./contracttest/...
cd apps/web && npx tsc -b --force && npx eslint <fichiers touchés> && npx vitest run src/features/match-view src/features/squad
```

---

## LOT D — Point 3 : usages d'équipement Spartan par élément / joueur / équipe (web pur)

État de l'art (recherche 2026-08-24) : TOUT se calcule côté web depuis le document
de rejeu déjà servi (SchemaVersion 18 INCHANGÉ, zéro Go, zéro re-cuisson — patron
`PLAN_TEMPS_MORT_WEB.md`). Canaux attribués par joueur : `grappleLines[].slot`
(tractions de grappin — la seule ACTIVATION mesurée et attribuée),
`equipmentEpisodes[fam=camo|overshield]` (épisodes d'état actif — proxy attribué du
power-up ; la SOURCE socle-vs-capacité n'est PAS discriminée : réserve à afficher),
`equipmentPlacements[]` avec `owner >= 0` (`origin=deployed` = déploiements par
famille ; `origin=dropped` = lâchés à la mort), `grenades[]` (par type). Canal
ANONYME : `padPickups[]` croisé aux `weaponPads[]` de famille `powerup_*` (« socles
de power-up vidés » — `xuid` est null PAR MESURE, jamais « ramassé par X »).
Répulseur/propulseur : AUCUN canal publié (Phase 0 du chantier `wt/usages-equipement`
en cours) — HORS PÉRIMÈTRE, ne pas planifier, ne pas simuler (règle « aucun effet
sans donnée mesurée »). Pont slot → joueur → équipe déjà écrit :
`rosterLogic.ts` (`sideBySlot`) + `useSlotIdentity.ts` (`buildPlayers`) ;
`Track.Team` vaut toujours -1 — l'équipe passe par le scoreboard, un joueur sans
ligne de scoreboard est un trou AFFICHÉ, pas comblé.

### Items

- [x] D1 — `equipmentUsageLogic.ts` (features/match-replay) : agrégation PURE
  `doc → { byPlayer, byTeam }` des grandeurs : tractions de grappin, épisodes camo,
  épisodes surbouclier (nombre + durée cumulée), déploiements par famille, objets
  lâchés à la mort par famille, grenades par type ; et par match : socles de
  power-up vidés (anonyme). Dénominateurs de couverture repris de
  `doc.coverage` (`equipment.tracksTotal`, `grapple.pullLives`,
  `placements.byFamilyOrigin`, `groundWeapons.powerupPads`). Tests vitest complets
  sur fixtures synthétiques (y compris joueur hors scoreboard).
  → TROIS DÉCISIONS non écrites au plan, toutes trois imposées par la vérification sur
  pièces des canaux :
  (a) **les grenades ne se joignent PAS par `slot`**. `Grenade.Slot` est « le biped lanceur
  QUAND IL EST CONNU (0 sinon) » (`grenades.go`) ; l'auteur est `Grenade.i`, l'index de
  joueur ÉCRIT dans le film, et `grenadeFx.grenadeThrowActive` joint déjà par là. Mesuré sur
  4 témoins du cache : 65/70, 108/143 et 123/130 lancers portent un slot ABSENT des pistes —
  la jointure par slot en perdait la quasi-totalité, et les aurait TOUS versés au
  propriétaire du slot 0 sur un film où ce slot existe. Corrigé : pont
  `roster[].filmIndex → xuid`. Orphelins retombés de 70/108/4/125 à 5/17/4/3.
  (b) **les déploiements passent par `placementIsDeployedObject` croisé à
  `PLACEMENT_RENDER[family] != null`**, pas par `origin === 'deployed'` nu. Le premier
  dédoublonne le mur (un mur déployé publie DEUX poses : l'appareil et ses panneaux) ; le
  second écarte les familles que la table déclare explicitement « pas un objet posé sur le
  terrain » — les 4 grenades (leurs poses `deployed` sont des LANCERS, déjà comptés par
  `grenades[]` : deux colonnes pour un seul geste) et `grapple`/`thruster`/`repulsor`. Les
  lâchers passent de même par `placementIsDroppedPower` (décision produit du 18/08, déjà
  mesurée et écrite). Sans ces deux filtres, le tableau publiait des colonnes `repulsor` et
  `thruster` — exactement ce que le lot interdit — et des colonnes de grenades en double.
  (c) **une entrée de roster sans aucune vie n'a pas de ligne**, et ses gestes éventuels vont
  aux ORPHELINS : sans ce garde, un lancer attribué à une telle entrée disparaissait de
  l'écran sans entrer nulle part, et la somme des lignes mentait.
  Le module a été SCINDÉ avant d'atteindre le seuil (498 L / 500) : `equipmentUsageColumns.ts`
  porte la mise en colonnes et les libellés, `equipmentUsageLogic.ts` ne connaît aucune langue.
- [x] D2 — composant `MatchEquipmentUsageSection.tsx` (features/match-view ou
  match-replay selon l'existant du lot B) : tableau par joueur avec ligne « Total
  équipe » (patron `MatchObjectivesSection.tsx`), ligne à part pour l'anonyme
  (« Socles de power-up vidés : N » au niveau match), libellés des familles via les
  labels de rejeu existants (jamais en dur), réserve affichée pour camo/surbouclier
  (« état actif mesuré — source socle ou capacité non distinguée », en infobulle).
  Double porte : artefact présent (`header.replay_available` + donnée non vide)
  sinon rien. i18n FR/EN.
  → Posé dans **`features/match-replay/`** : chaque libellé qu'il écrit appartient au
  dictionnaire du rejeu (`placementFamily`, `padEquipmentFamily`, catalogue de grenades du
  document). Le poser dans `match-view` aurait forcé soit l'import du dictionnaire voisin,
  soit une SECONDE table de noms — celle qui diverge au premier ajout du manifeste. Le sens
  de l'import est déjà établi en face : `MatchScoreCurveChart` (match-view) lit
  `match-replay/queries`.
  Libellé FR « Socles de bonus de puissance vidés » et non « power-up » : c'est le mot déjà
  employé par `padPlacementNotePowerUp` (règle FR sans anglicismes). Le compte est un nombre
  de VIDAGES, pas de socles — un socle se vide plusieurs fois (témoin `00162144` : 7 vidages
  sur 1 socle) — d'où le dénominateur affiché « sur N socle(s) mesuré(s) ».
  DEUX libellés courts ajoutés (`equipmentUsage.activeFamily`) plutôt que réutilisés :
  `equipmentActive` porte des PHRASES d'infobulle de fiche (« Camouflage actif — le joueur
  est invisible à l'écran de jeu »), illisibles en tête de colonne ; `padEquipmentFamily`
  porte bien deux noms courts, mais ce sont ceux des SOCLES — les employer nommerait l'ÉTAT
  par la source que la réserve dit justement ne pas être établie. Justification écrite au
  contrat.
  Une phrase à l'écran (`notMeasured`) dit que répulseur et propulseur ne sont pas mesurés :
  aucune colonne vide (elle se lirait « zéro utilisation »), mais l'absence est NOMMÉE.
- [x] D3 — montage dans l'onglet **Chronologie** (lot B), après la courbe de score ;
  consomme `useMatchReplay` (déjà utilisé par `MatchScoreCurveChart` — même cache,
  aucun fetch additionnel).
  → Monté dans `MatchViewTabChronology.tsx` juste après `MatchScoreCurveChart`, avec les
  mêmes `playerSlug` / `matchId` / `replayAvailable` : la clé
  `queryKeys.matchReplay(playerSlug, titleSlug, matchId)` est identique, TanStack Query
  dédoublonne — aucun téléchargement de plus pour un artefact de 1,5 à 2,7 Mio.
- [x] D4 — INTERDITS respectés : aucun accès `filmdec/`, aucun bump de schéma,
  aucune modification de `ReplayCanvas.tsx` (cliquet 777), aucune table DB.
  → Vérifié sur pièces à la clôture : `git diff --name-only` sur le lot ne touche AUCUN
  fichier de `apps/go-api` (0), aucun `*filmdec*` (0), aucun `*ReplayCanvas*` (0) ;
  `SchemaVersion = 18` inchangé (`document.go:149`) ; zéro `CREATE TABLE` dans le diff ;
  zéro hex et zéro classe Tailwind de couleur dans les fichiers du lot (tokens sémantiques
  seuls) ; zéro emoji. Le cliquet de taille de `ReplayCanvas.tsx` (797, pas 777) n'est pas
  approché : le fichier n'est pas touché.
- [x] D5 — tests : vitest logique (D1) + rendu (tableau, porte artefact absent,
  ligne anonyme) ; typecheck.
  → 40 tests : `equipmentUsageLogic.test.ts` (24 — pont slot/joueur/équipe, joueur hors
  scoreboard, sans scoreboard du tout, roster sans vie, épisodes bornés et famille inconnue,
  dédoublonnage du mur, origine non établie, répulseur/propulseur sans grandeur, poseur -1,
  jointure des grenades par index de film et le slot menteur, socle hors bornes, couverture
  présente et absente, double porte) et `MatchEquipmentUsageSection.test.tsx` (16 — les trois
  portes fermées, groupes de colonnes, libellés pris aux tables existantes, total d'équipe
  = somme du CAMP, durée m:ss et « — », joueur hors scoreboard sous « Sans équipe », ligne
  anonyme HORS tableau avec son dénominateur, dénominateurs de couverture, absence de colonne
  répulseur/propulseur + phrase, réserve en infobulle d'en-tête, orphelins, parité EN).
  PLAUSIBILITÉ sur 4 artefacts réels du cache du dépôt principal (mesure jetable, non
  commitée) : 8 à 26 joueurs, 129 à 171 gestes attribués par match, 3 à 17 orphelins,
  17 à 23 grenades par joueur, épisodes de camouflage de 3,5 à 41,4 s cumulées,
  7 et 9 vidages de socle de bonus sur 1 socle mesuré.

### Gate D

```
cd apps/web && npx tsc -b --force && npx eslint <fichiers touchés> && npx vitest run src/features/match-replay src/features/match-view
```

---

## LOT E — Point 4 : présence en jeu (icône manette + compteur d'amis)

État de l'art : la présence Xbox Live EST implémentée et tourne (package
`internal/presence/`, `GET userpresence.xboxlive.com/users/xuid(N)?level=all`,
poller REST 10 s du watcher, backoffs, refresh XSTS ; `MultiUserTokenStore.AuthHeader()`
donne le header par xuid). Mapping titleId Xbox → slug : RÉSOLU
(`title/matcher.go` `MatchPresence`, Infinite `2043073184`, H5 `219630713`).
Manques : (1) le titre COURANT n'est pas stocké (`player_watcher.go` ne garde que
`inGame`) ; (2) piège daemon `daemon.go:466-480` — si titre détecté ≠ titre
configuré du joueur, le handler sort en « inactif » : capter le titre APRÈS le
`MatchPresence` réussi, AVANT le test de slug, sinon un joueur `halo_5` qui lance
Infinite paraît hors jeu ; (3) l'endpoint actuel `/watcher/status` est
`RequireAdmin` ; (4) la liste déroulante est un `<select>` HTML natif (une
`<option>` ne peut pas porter de SVG). « Amis » (D-P4) = `friend_gamertags` des
Réglages (résolution gamertag → xuid via `shared.v_gamertag_lookup`).

### Items

- [x] E1 — Go watcher : stocker le titre courant (`player_watcher.go` : champ +
  enregistrement au bon endroit du handler — cf. piège ci-dessus) ; le remonter
  dans `PlayerPresenceStatus` (`provider.go`) : `in_game`, `title_slug`,
  `title_name`.
  → Champs `currentTitleSlug`/`currentTitleName` + `SetCurrentTitle`/`CurrentTitle`.
  Le piège est traité DANS L'ORDRE : `pw.SetCurrentTitle(td.Slug, td.Name)` est posé
  juste après le `MatchPresence` réussi, AVANT le test « titre du watcher » qui sort
  en `OnPresenceInactive`+return ; les deux autres sorties (titre hors registre,
  payload sans titre) effacent le titre. DÉCISION non écrite au plan : `in_game` de
  `PlayerPresenceStatus` n'a PAS changé de sens — il reste la sémantique WATCHER
  (« joue-t-il au titre que CE watcher suit ? ») qui pilote la FSM et qu'affiche la
  carte admin. Le `in_game` du CONTRAT PUBLIC (E3) est dérivé du titre courant
  (`title_slug != ""`). Redéfinir le champ existant aurait changé l'affichage de
  `/watcher/status` sans que personne ne le demande, et fait mentir la FSM.
  Les deux accesseurs vivent dans `player_watcher_title.go` : `player_watcher.go`
  est à 508 lignes (dette gelée au-delà du seuil), l'ajout complet l'aurait porté à
  542 — il finit à 513, l'accroissement incompressible des deux champs.
- [x] E2 — Go présence des amis : appel batch
  `POST userpresence.xboxlive.com/users/batch` (nouvelle méthode du
  `PresenceClient`, même auth, même contract-version ; réutiliser
  `ParsePresencePayload` par élément) ; résolution des `friend_gamertags` → xuids
  via `v_gamertag_lookup` ; calcul « amis en jeu » = présence sur N'IMPORTE quel
  titre supporté (`titleReg.MatchPresence(titleID) != nil`) ; cache TTL en mémoire
  30-60 s (patron `privacyTTLCache`), calcul À LA DEMANDE (pas de poller dédié).
  Amis à la présence masquée (privacy) : ignorés silencieusement du compte, avec un
  `slog.DebugContext` — jamais d'erreur utilisateur.
  → `presence/batch_client.go` (fichier dédié : `rest_client.go` documente le poll
  unitaire). Le corps envoyé porte `"level":"all"` EN PLUS de `users` : sans lui la
  réponse se limite au niveau « user » et ne contient PAS `devices[].titles[]` — le
  compteur serait constamment à zéro. Un élément illisible du lot est ignoré (Debug)
  au lieu d'emporter les autres. Résolution inverse ajoutée au chokepoint existant :
  `GamertagRepo.ResolveXUIDsByGamertags` (insensible à la CASSE — la liste des
  Réglages est tapée à la main —, bots exclus, clés = les gamertags demandés).
  Compteur + cache dans `service/presence_friends.go`, TTL **45 s** (> les 30 s de
  poll du shell, pour qu'un onglet ouvert ne provoque pas un appel Xbox par tick).
  Un ÉCHEC n'est pas mis en cache : un incident d'une seconde ne doit pas geler
  45 s d'affichage.
- [x] E3 — Go endpoint : `GET /api/v1/presence` sous `RequireAuth` + `NoStore`
  (PAS admin), servi depuis `watcher.WatcherStateProvider` + le calcul E2 :
  `{ players: [{player_slug, gamertag, in_game, title_slug, title_name}],
  friends_in_game: N }` ; joueurs filtrés par `filterOwnedPlayers` (ADR 0029) ;
  si daemon absent/éteint → `players: []`, `friends_in_game: 0` (200, jamais 500).
  Handler sans logique métier : le calcul vit dans un service.
  → `service.PresenceService` (+ `domain/presence.go`), handler mince
  `handlers/presence.go`, wiring dans `api/server_presence.go` (server_apiv1.go est
  un assembleur exempté : on n'y ajoute pas d'adaptateurs). La liste rendue est
  l'INTERSECTION watcher × joueurs possédés — watcher éteint ⇒ liste vide, ce que
  demande le plan et qui est exact (on ne sait rien). `filterOwnedPlayers` n'a pas
  été recopié : la combinaison « joueurs du titre + visibles + possédés » est
  extraite en `BootstrapService.OwnedPlayers`, dont `BuildPlayersList` devient un
  appelant (source unique, règle des 2 copies). Le service ne dépend NI du daemon
  NI du client Xbox : il consomme des func/types neutres, ce qui le rend testable
  sans HTTP. `DaemonController` n'a PAS été élargi (son mock de test l'aurait été
  aussi) : le lot d'amis passe par un type-assert `presenceBatcher` sur la méthode
  `Daemon.PresenceBatch` — emprunter le client Xbox du daemon n'est pas le
  contrôler. `reg.FriendGamertags` expose le résolveur d'amis EXISTANT (celui de la
  page Escouade) plutôt qu'une seconde closure sur le settings store.
- [x] E4 — contrats : openapi + `make generate-types`.
  → `openapi-gen` (+57 lignes : `/presence`, `PresenceSnapshot`, `PlayerPresence`,
  et `title_slug`/`title_name` sur `PlayerPresenceStatus`) puis `generate-types`
  (+60 lignes) ; `openapi-gen -check` vert, `TestOpenAPIYAMLIsUpToDate` et
  `TestContractRoutesDocumented` verts. Ré-exports `PresenceSnapshot`/
  `PlayerPresence` dans `lib/api/types.ts` (le `players: [] | null` du contrat est
  comblé une seule fois, dans le hook).
- [x] E5 — web : hook `usePresence()` (clé dans `lib/query/keys.ts`,
  `refetchInterval: 30_000`, `staleTime` cohérent) ; remplacement du `<select>`
  natif de `NavL1.tsx` (l.411-432) par un dropdown custom sur le gabarit
  `SplitButton`/`SettingsSplitButton` du même fichier (`role="menu"`,
  click-outside, navigation clavier) — comportement de bascule joueur inchangé
  (`handlePlayerChange`) ; icône manette SVG inline à droite des users en jeu ;
  pour le joueur ACTIF : badge compteur à droite (« N » + libellé accessible
  « N amis en jeu » FR / « N friends in game » EN) rendu seulement si N > 0.
  Aucune couleur en dur — tokens.
  → Le dropdown vit dans `components/shell/PlayerSwitcher.tsx` et non dans
  `NavL1.tsx` : le fichier était à 449 lignes, l'ajout l'aurait poussé au-delà de
  500. Le GABARIT est bien celui des split buttons du fichier d'origine (wrapper
  `relative`, panneau `absolute/bg-popover/z-50/role=menu`, click-outside par
  `mousedown`), enrichi du clavier attendu d'un menu ARIA : ArrowDown ouvre depuis
  le déclencheur, flèches haut/bas bouclent entre les joueurs, Échap ferme et rend
  le focus. Hook `usePresence` à côté de son composant (précédent :
  `components/ui/useGamertagSuggestions.ts`), clé `queryKeys.presence(titleSlug)`
  title-scopée + entrée au garde-rail `keys.title-slug.guard.test.ts`.
  DEUX décisions non écrites au plan : (a) le joueur ACTIF porte AUSSI la manette
  quand il est en jeu — sinon il faudrait ouvrir le menu pour savoir si on est
  soi-même en partie ; (b) la pastille d'amis affiche une manette À CÔTÉ du nombre,
  un « 3 » nu à côté d'un gamertag n'étant pas interprétable. Le titre RÉEL est
  nommé dans l'infobulle (« En jeu sur Halo 5 ») : c'est tout l'intérêt du champ
  capté en E1. i18n par le manifeste `common.toml` (mécanisme en place dans NavL1),
  3 clés FR/EN dont le compteur en pluriel ICU. Le cas `availablePlayers.length <= 1`
  reste un simple libellé (pas de menu), manette et compteur compris.
- [x] E6 — tests : Go = parser batch + logique « amis en jeu » (fixtures presence)
  + handler httptest (daemon absent → réponse vide) ; web = vitest du dropdown
  (liste, icône conditionnelle, compteur conditionnel, bascule joueur) ;
  typecheck.
  → Go, 27 tests : `presence/batch_client_test.go` (6 — corps `users`+`level=all`,
  en-têtes, xuid vide écarté, liste vide sans appel réseau, élément illisible
  ignoré, 401 typé `*HTTPError`), `service/presence_friends_test.go` (10 — n'importe
  quel titre suivi compte, présence masquée et ami absent ignorés, gamertag inconnu,
  échec Xbox non mis en cache, cache dans le TTL, changement de liste = invalidation,
  dédoublonnage par xuid, xuid non demandé, aucun ami configuré = zéro appel,
  dépendance manquante = pas de compteur), `service/presence_service_test.go` (8 —
  dont le cas du lot : joueur halo_5 sur Infinite EN JEU avec le titre réel, deux
  watchers d'un même gamertag, watcher absent/arrêté, joueur non possédé exclu,
  erreur de chargement dégradée), `watcher/presence_title_test.go` (5 — l'ordre du
  handler, titre effacé sur titre hors registre et hors ligne, lot sans client),
  `api/handlers/presence_test.go` (3 — daemon absent → 200 vide, aucune source,
  cas nominal servi). `platform/duckdb/gamertag_resolve_test.go` (+1, tag
  `integration`) couvre le SQL de résolution inverse — exécuté à la main dans cette
  session (le gate du lot ne monte pas les tests d'intégration). Web, 16 tests :
  `PlayerSwitcher.test.tsx` (liste, bascule + fermeture, manette conditionnelle avec
  titre réel puis sans titre, compteur pluriel/singulier, absence de compteur,
  endpoint en erreur muet, aria haspopup/expanded, ouverture clavier, flèches qui
  bouclent, Échap, clic dehors, cas mono-joueur, parité EN, `players: null`).
  Handler MSW `/presence` ajouté aux fixtures partagées (défaut « personne en jeu »).

### Gate E

```
cd apps/go-api && go vet ./... && go test ./internal/presence/... ./internal/watcher/... ./internal/service/... ./internal/api/... ./contracttest/...
cd apps/web && npx tsc -b --force && npx eslint <fichiers touchés> && npx vitest run <tests shell/NavL1>
```

---

## LOT F — correctifs de revue adversariale (2026-08-25)

Origine : le gate global du chantier (2 garde-rails rouges) et QUATRE relectures
adversariales du diff A→E. Chaque item cite le constat de la revue ; le code du dépôt
fait foi et a été rouvert avant correction. Aucun élargissement de périmètre : toute
trouvaille annexe part en « Découvertes ».

### Garde-rails du dépôt (P1 — gate global rouge)

- [x] F1 — `platform/duckdb/match_view_repo_assist_pairs_test.go:54` : le fixture écrit
  les littéraux bruts de portée kill-events. Passer par les constantes de
  `internal/domain/killscope`. Gate : `go test ./internal/archlint/` vert.
  → `killscope.ReadPathFilmWalk` (`match_view_repo_assist_pairs_test.go:63,73`) et non
  l'une des trois constantes CRÉDIT citées par la revue : le seul littéral que le
  ratchet J4R-3 refusait était `'marche'`, la MARCHE du décodeur de film, et c'est la
  seule portée qui ait un sens ici — les producteurs crédit écrivent
  `OriginCreditOnly`, « le crédit et rien que le crédit », et ne connaissent PAS
  l'assistant que ce fixture pose. Écrire `ReadPathLiveFeed` aurait rendu le fixture
  faux. L'origine `credit-concordant` reste un littéral, NOMMÉ et commenté
  (`filmCreditOrigin`) : son propriétaire typé est `killsource.Origin`, paquet
  title-specific que `platform/duckdb` n'a pas à importer, elle n'est pas dans
  `killscope` (qui ne porte que le vocabulaire partagé des écrivains crédit) et le
  ratchet ne la couvre pas — même traitement que les dix autres fixtures du dépôt.
- [x] F2 — `internal/presence/batch_client.go` : appel HTTP sortant sans
  `netguard.Check` (le mode démo fuiterait). Poser le garde AVANT l'émission, erreur
  traitée par le chemin de dégradation existant. Gate :
  `go test ./internal/platform/netguard/` vert.
  → Surface `xbox_presence.batch` (`batch_client.go:48,116`), garde posé après le
  nettoyage de la liste et avant l'encodage du corps. DÉCISION : la dégradation est
  `(nil, nil)` et non `ErrOffline`. Le poll unitaire voisin est ALLOWLISTÉ parce qu'il
  appartient au daemon watcher, éteint en démo ; le lot, lui, part d'une requête
  utilisateur (`GET /api/v1/presence`, tiré toutes les 30 s par le shell) — il est donc
  bel et bien atteignable, d'où le garde plutôt qu'une entrée d'allowlist. Rendre une
  erreur ferait journaliser un `Warn` au compteur d'amis toutes les 45 s pour un refus
  ATTENDU, et armerait son backoff d'échec (F4) sans raison. Test
  `TestGetPresenceBatch_DemoMode_NoCall`.

### P1 de revue

- [x] F3 — `features/explorer/queries.ts:26-36` : `matchFiltersKey` ignore
  `replay_scope` (contrairement à `squad_scope`) — changer le filtre Rejeu ne
  déclenche aucun refetch et empoisonne le cache sous la même clé. Ajouter le champ à
  la clé + cas de test.
  → La clé devient une fonction PURE exportée, `matchFiltersKeyOf`
  (`explorer/queries.ts:15-41`) : elle était calculée en ligne dans le hook, donc
  intestable sans monter React — c'est ce qui a laissé le trou passer. Nouveau fichier
  `explorer/matchFiltersKey.test.ts` (4 cas : `replay_scope` distingue les 3 états,
  `squad_scope` idem, l'ordre de sélection ne change PAS la clé, vide == absent).
- [x] F4 — `service/presence_friends.go:94-106` : à cache froid ou en échec, chaque
  requête part vers Xbox. Ajouter (a) un singleflight (verrou tenu pendant le fetch)
  et (b) une mémoire d'échec courte (~30 s). Le test
  `TestFriendsCount_FetchErrorReturnsZeroAndIsNotCached` évolue : l'échec n'est
  toujours pas mis en cache COMME RÉSULTAT, mais il impose un backoff.
  → (a) `mu` est désormais tenu sur TOUT le calcul, appel Xbox compris
  (`presence_friends.go:83,130-151`) — le singleflight le plus simple qui tienne la
  promesse ; le coût (un appelant peut attendre le fetch d'un autre) est borné par le
  budget de 3 s posé en F7, et c'est écrit au godoc du champ. (b) `failedKey`/
  `failedAt` + `FriendsPresenceFailureBackoff = 30 s`, portés par la MÊME clé que le
  cache (changer la liste relance immédiatement) et effacés par tout succès. Le test
  est renommé `TestFriendsCount_FetchErrorReturnsZeroAndBacksOff` et vérifie EN PLUS
  que `cacheKey` reste vide — un échec n'est jamais un résultat. 4 tests neufs :
  reprise après expiration du backoff, effacement par un succès, contournement par un
  changement de liste, et 8 requêtes simultanées à cache froid → 1 seul appel sortant.
- [x] F5 — `service/presence_service_test.go:71-84` : le test du garde « le watcher
  porteur d'un titre gagne » est tautologique (l'entrée sans titre est énumérée en
  premier). Jouer LES DEUX ordres de fixture.
  → `t.Run` sur les deux ordres. Vérifié que le test ÉCHOUE bien sans le garde dans
  l'ordre « titre en premier » (c'est le seul des deux qui l'éprouve).

### P2 retenus (dans le périmètre du chantier)

- [x] F6 — `presence_friends.go:88-96` : la résolution gamertag→xuid (settings + requête
  DuckDB) s'exécute AVANT le test de cache à chaque appel. Clé de cache = la LISTE DE
  GAMERTAGS triée ; dans le TTL, aucun accès settings/DB.
  → Clé = `normalizedFriendList` (blancs retirés, dédoublonnée, TRIÉE) jointe par
  retour-ligne ; la résolution DuckDB passe DERRIÈRE la porte du cache, dans `measure`.
  ÉCART ASSUMÉ sur la lettre de l'item : la lecture des RÉGLAGES, elle, reste en amont
  du test de cache — c'est elle qui produit la clé, donc le seul moyen de détecter
  qu'un ami a été ajouté ; c'est un chargement local (`settingsStore.Load`), pas une
  requête de base, et le test d'invalidation par changement de liste en dépend. Écrit
  au godoc de `Count`. Deux tests : la résolution n'a lieu qu'UNE fois sur 3 appels, et
  réordonner/espacer/dupliquer la liste ne casse pas le cache.
- [x] F7 — `presence_service.go:75-95` : borner le comptage d'amis (contexte à timeout
  court, 3 s) — la réponse `/presence` ne doit jamais attendre Xbox 20 s.
  → `friendsCountBudget = 3 s` + `countFriendsWithinBudget` : le comptage tourne dans
  une goroutine et la réponse ne l'attend que le temps du budget. Un simple
  `context.WithTimeout` n'aurait PAS suffi depuis F4 — un appelant bloqué sur le verrou
  du singleflight n'observe pas l'annulation de son contexte (`sync.Mutex.Lock` ignore
  le ctx). Le canal est tamponné, la goroutine se termine sur l'annulation du contexte.
  Champ `friendsBudget` abaissé par le test (précédent : `RESTPoller.WithInterval`).
  Test : source amie qui bloque jusqu'à annulation → réponse immédiate,
  `friends_in_game = 0`, joueurs intacts.
- [x] F8 — `watcher/player_watcher_title.go:35` : `CurrentTitle()` exportée sans aucun
  appelant → SUPPRIMER (règle 0 code mort).
  → Supprimée. Vérifié sur pièces avant : zéro appelant dans tout le dépôt (l'unique
  lecteur, `StateProvider.GetStatus`, lit les deux champs directement sous `pw.mu`,
  avec une dizaine d'autres, d'un seul tenant). Un commentaire à sa place dit POURQUOI
  il n'y a pas d'accesseur en lecture — sans quoi le prochain passage le rajoute.
- [x] F9 — fraîcheur de la présence servie : VÉRIFIER d'abord la cadence des events. Si
  le handler du daemon est invoqué à CHAQUE poll réussi, blanchir titre + `in_game`
  quand `LastEventAt` date de plus de 3 minutes. Sinon, `[!]` avec preuve.
  → CONDITION VÉRIFIÉE, donc borne APPLIQUÉE. Preuve : `watcher/rest_poller.go:153-163`
  (`tickOnce` appelle `p.handler(event)` sur CHAQUE poll réussi, sans aucun filtre de
  changement d'état) à `restPollInterval = 10 s` (`rest_poller.go:38`), et
  `watcher/daemon.go:444` pose `pw.RecordEvent(time.Now())` AVANT tout filtrage. Le
  témoin avance donc tout seul toutes les 10 s tant que le poll vit.
  Implémentation : `presenceFreshnessWindow = 3 min` + `TrackedPresence.fresh(now)`,
  appliqué À L'INGESTION dans `trackedByGamertag` — et non après l'arbitrage : une
  entrée périmée PORTEUSE d'un titre aurait sinon battu une entrée fraîche disant
  « hors jeu » (test dédié). `LastEventAt` traverse l'adaptateur
  `server_presence.go` (parse RFC3339 ; vide ou illisible = temps zéro = pas en jeu,
  avec un `slog.Warn` sur l'illisible). 5 tests service + 2 tests de jonction.
- [x] F10 — colonne rejeu d'`ExplorerMatchesTable.tsx` : poser `enableSorting: false`
  comme sa jumelle `SquadSynergyHistoryTable.tsx:200`.
  → `ExplorerMatchesTable.tsx:452`. Le mécanisme d'exemption existait déjà (le merge
  des colonnes RESPECTE un `enableSorting: false` explicite, l.921-929) : un mot a
  suffi. Test formulé en INVARIANT — aucun en-tête vide ne porte de contrôle de tri —
  donc il couvre aussi la colonne Waypoint voisine.
- [x] F11 — `squad/i18n.ts` `assists.description` traduit mais jamais monté : l'afficher
  sous le titre de la section Assistances de `SquadSynergiesPage.tsx`.
  → `SquadSynergiesPage.tsx:170-173`, classe alignée sur le bandeau de couverture juste
  en dessous (`text-xs text-muted-foreground`). Deux tests (montée quand le bloc est
  là, absente sinon) ; le helper de mock du contexte accepte maintenant un `pageData`.
- [x] F12 — `match-replay/equipmentUsageColumns.ts:113` : une durée MESURÉE à zéro rend
  « — » alors que la cellule épisodes voisine écrit 0. Zéro mesuré = « 0:00 ».
  → Helper `durationCell`, jumeau d'`intCell` (`equipmentUsageColumns.ts:76-88`). Le
  repli d'absence reste à sa place — l'absence de la COLONNE, décidée en amont par
  `usage.columns.episodes`. Le test existant qui figeait le « — » est corrigé, un
  second couvre le cas `t1 == t0` (épisode observé, durée nulle).
- [x] F13 — état vide du graphe des assistances (match-view) : « non mesurée » est FAUX
  pour un film BTB (mesuré mais non publiable ligne à ligne). Reformuler FR/EN pour
  couvrir les deux cas, sans changer le contrat.
  → Clé renommée `assistNotMeasured` → `assistNotUsable` (le nom mentait autant que le
  texte). FR « Assistances non disponibles pour ce match (non mesurées ou non
  publiables). » / EN « Assists unavailable for this match (not measured or not
  publishable). » Contrat inchangé (`measured_deaths` reste le seul discriminant).
  Commentaires du contrat i18n ET de l'en-tête de `MatchAssistChart.tsx` corrigés — ils
  affirmaient tous deux « non mesurée » (doc inversée).
- [x] F14 — `match_view_builders_assists.go:50` : l'ASSISTANT est nommé par le gamertag
  du film alors que le tueur est résolu au scoreboard (deux orthographes possibles pour
  un même joueur dans un seul graphe). Résoudre AUSSI l'assistant par xuid, repli sur
  le gamertag du film.
  → `match_view_builders_assists.go:55-58`. Repli ASYMÉTRIQUE et voulu : le nom du film
  pour l'assistant (mieux vaut le nom d'hier que pas de nom, et il en a toujours un —
  Q21d exige `assist_gamertag IS NOT NULL`), la chaîne VIDE pour le tueur (contrat
  livré au lot C, le front a son masque « Joueur #### »). L'ancien test figeait
  explicitement le comportement fautif (« le scoreboard ne le corrige pas ») : corrigé,
  plus un test dédié (nom changé depuis le film, assistant absent du scoreboard,
  assistant présent mais anonyme).
- [x] F15 — tests manquants sur du code du chantier : (a) invoquer réellement
  `option.tooltip.formatter` de `BarStackedChart` (chemins `tooltipComponentNote` ET
  `tooltipHideZero`) ; (b) expiration du TTL amis ; (c) jonction httptest de
  `server_presence.go` (daemon vivant → JSON complet ; daemon arrêté → players vide).
  → (a) 6 cas dans `BarStackedChart.test.ts` : note sur la bonne paire (les arguments
  reçus par le rappel sont vérifiés un à un), zéros conservés quand seule la note est
  demandée, masquage préservé avec `tooltipHideZero`, infobulle VIDE quand tout est
  masqué, échappement HTML, et AUCUN formateur installé sans option (le comportement
  des appelants antérieurs). (b) `TestFriendsCount_CacheExpiresAfterTTL` — `cachedAt`
  vieilli à la main. (c) `internal/api/server_presence_test.go` (5 tests, package `api`
  pour atteindre `trackedPresenceFrom`) : daemon vivant → JSON complet, daemon arrêté →
  liste vide, `last_event_at` périmé → titre blanchi, `last_event_at` vide → idem,
  daemon absent → source nil et 200 vide.
- [x] F16 — `match_view_repo_assist_pairs.go:62` : le commentaire dit « 6 colonnes », la
  requête en rend 7. → Corrigé (`match_view_repo_assist_pairs.go:62`).

### Gate F

```
cd apps/go-api ; go vet ./... ; go test ./internal/archlint/... ./internal/platform/netguard/... ./internal/presence/... ./internal/watcher/... ./internal/service/... ./internal/api/... ./internal/platform/duckdb/... ./contracttest/...
cd apps/web ; npx tsc -b --force ; npx eslint <fichiers touchés> ; npx vitest run src/features/explorer src/features/squad src/features/match-view src/features/match-replay src/components
```
0 erreur, 0 nouveau warning.

Résultats (2026-08-25, worktree `wt/notion-cinq`) :

| Gate | Code | Sortie |
|---|---|---|
| `go vet ./...` | `EXIT_VET=0` | aucune ligne hors bruit CGO préexistant (paquets exclus par build constraints) |
| `go test` (8 paquets du gate) | `EXIT_GO_TESTS=0` | 17 `ok`, 3 `[no test files]`, 0 FAIL |
| `npx tsc -b --force` | `EXIT_TSC=0` | sortie vide |
| `npx eslint` (12 fichiers web touchés) | `EXIT_ESLINT=0` | 0 erreur, 1 warning PRÉEXISTANT (`react-hooks/incompatible-library` sur `useReactTable`, `ExplorerMatchesTable.tsx:950` — déjà consigné en Découvertes au lot A, fichier et ligne inchangés par F10) |
| `npx vitest run` (5 périmètres) | `EXIT_VITEST=0` | 229 fichiers, 2395 tests passés |

Baselines rouges AVANT correction, pour mémoire : `TestNoRawKillScopeLiteral` échouait
sur `match_view_repo_assist_pairs_test.go:54`, `TestOutboundCallsAreNetguarded` sur
`presence/batch_client.go`.

Contrôle de mutation sur F5 (le test était tautologique) : garde de préséance retiré à
la main → `TestPresenceSnapshot_TwoWatchersSameGamertag_TitleWins/titre_en_premier`
ÉCHOUE, `/sans_titre_en_premier` passe. Fichier restauré à l'identique (`diff` vide),
test re-vert.

---

## Clôture du chantier (superviseur)

- [x] Gate global dans le worktree (2026-08-25) : go vet 0 ; `go test` complet hors
  `himap` (lenteur locale documentée) — VERT après lot F (le premier passage avait
  attrapé archlint + netguard, corrigés en F1/F2) ; tsc 0 ; vitest complet 476
  fichiers / 4592 tests verts. Suite `-tags=integration` non rejouée : aucun
  fichier de `persist/`, `sync/` (écritures) ni `migration/` touché par le chantier.
- [x] Revue adversariale du diff complet (2026-08-25) : ronde 1 = 4 relecteurs
  aveugles (données, accès/concurrence, front, couverture de tests) → 0 P0, 4 P1,
  ~15 P2 ; P1 + 12 P2 corrigés au lot F ; ronde 2 sur les seuls correctifs →
  0 P0/P1, 4 P2 consignés (section Découvertes). Décroissance stricte, boucle close.
- [ ] Fusion `wt/notion-cinq` → `feat/v75` en `--no-ff` (superviseur), journal
  `.ai/thought_log.md` + entrée registre si reports ; suppression worktree+branche.
  BLOQUÉE en attente : 4 fichiers du chevauchement (`openapi.yaml`, `generated.ts`,
  `match-replay/i18n.ts`, `i18nContract.ts`) sont modifiés NON COMMITÉS dans le
  worktree principal (lots utilisateur du 24/08, commit « à sa demande »).
- [ ] Notion : barrer les items 1-5 au format maison (« TRAITÉ <date> — commit
  <sha> » + note d'une ligne), item par item quand son lot est clos et fusionné.
- [ ] Gate visuel utilisateur : liste des témoins à vérifier fournie au CR final.

## Découvertes (à consigner, ne pas traiter)

- Skill `db-schema` périmée : documente encore `killer_victim_pairs` (colonnes
  inexistantes) et ne mentionne pas `match_kill_events` (recherche 2026-08-24).
- `MatchNarrativeSection.tsx` (221 L) : composant orphelin jamais monté.
- Cartes Home (`match-card.tsx`) sans indicateur rejeu (hors périmètre lot A).
- Fragment OpenAPI manuel PÉRIMÉ sur les request bodies Explorer (lot A, 2026-08-24) :
  `ExplorerMatchesQueryRequest` y est décrit avec un `match_filters:
  ExplorerMatchFilters` que le handler Go NE LIT PLUS (le corps réel porte les filtres
  à plat : `squad_scope`, `experience_types`, `playlists`, `perf_tiers`, `match_ids`…).
  `MatchHistoryQueryRequest` est décrit de même de façon partielle (4 champs sur ~20).
  Conséquence : aucun des 8 filtres Explorer n'est au contrat publié. Non traité (hors
  périmètre) — `replay_scope` a reçu le même traitement que ses 7 voisins.
- `ExplorerMatchesTable.tsx` reste un god-file (1113 L avant le lot, 1132 après) : la
  colonne rejeu y ajoute ~19 lignes malgré l'extraction de la cellule dans
  `lib/match-nav/MatchReplayLink.tsx`. Découpe du fichier non tentée (hors périmètre).
- L'avertissement eslint `react-hooks/incompatible-library` sur `useReactTable()` est
  générique au dépôt (vérifié sur `MatchEncountersTable.tsx`, fichier non touché) :
  2 occurrences sur les fichiers du lot, aucune nouvelle.
- E2E `apps/web/e2e/match-view-combat.spec.ts` (lot B, 2026-08-24) : spec déjà
  neutralisée par `skipObsoleteSpec`, dont le commentaire et le message de skip
  nomment encore les onglets « Général / Détails » — périmés depuis le passage à 3
  onglets. Rien de fonctionnel (test skippé) ; non traité, hors périmètre.
- Flake Windows sur `TestStartImport_HappyPathReturns202WithJobID`
  (`internal/api/handlers`, lot C, 2026-08-25) : `t.TempDir()` échoue au nettoyage
  (« Le répertoire n'est pas vide ») parce qu'une goroutine de `jobs.Store` écrit
  encore `jobs.json` quand le test se termine — le log du même run porte
  « jobs.Store: write error … utilisé par un autre processus ». Rouge au premier
  passage du gate C, VERT au re-run isolé (`-count=3`) et au re-run du paquet entier.
  Aucun rapport avec les assistances (import OpenSpartan). Non traité, hors périmètre.
- Aucun bandeau de couverture n'existait côté web avant le lot C (vérifié sur
  `features/squad`, `match-view`, `match-replay`) : `killFeedWeaponCoverage` cité par le
  plan est un compteur de LOG Go, pas un composant. Le bandeau de C7 est donc une
  première — si une deuxième surface en demande un, c'est le moment de le factoriser
  (règle des 2 copies).
- `TeammatesPageResponse` (web) est une interface ÉCRITE À LA MAIN dans
  `lib/api/types.ts`, exemptée dans `response-types.guard.test.ts` (« page composite
  squad »). Tout nouveau bloc de la page escouade doit donc être déclaré à DEUX endroits
  (Go + cette interface), sans quoi il n'existe pas pour le front — le contrat généré ne
  suffit pas. Constaté au lot C, non traité (la migration de cette interface vers le
  contrat généré est un chantier à part entière).
- `Grenade.Slot` est un champ PIÈGE du document de rejeu (lot D, 2026-08-25) : le Go le
  documente « le biped lanceur quand il est connu (0 sinon) », et sur les 4 témoins mesurés du
  cache 65/70, 108/143 et 123/130 lancers portent un slot absent des pistes. Un seul consommateur
  existe côté web (`grenadeFx.grenadeThrowActive`) et il joint bien par `Grenade.i` — mais rien
  n'empêche le prochain lecteur de tomber dans le piège, le champ étant nommé comme les `slot`
  des trois autres calques, qui eux SONT des slots de piste. Un garde-rail (ou un renommage du
  champ à la prochaine montée de schéma) serait à sa place. Non traité, hors périmètre.
- Résolution des amis sur le shared du titre PAR DÉFAUT (lot E, 2026-08-25) :
  `friendXUIDResolverFrom` branche `GamertagRepo` sur `cfg.SharedProvider`, comme la
  recherche de gamertags existante — donc la vue `v_gamertag_lookup` du titre par
  défaut, pas celle du titre courant. Sans effet mesurable ici (un ami croisé sur
  n'importe quel titre y a son xuid, et le compte porte sur TOUS les titres suivis),
  mais c'est un raccourci mono-titre partagé avec `gamertagSvc`. Non traité, hors
  périmètre.
- Écriture non verrouillée de `Daemon.trackerRestClient` (lot E, 2026-08-25) : le
  champ est ASSIGNÉ dans `Start()` hors de `playersMu` alors que ses lecteurs
  (`GetStatus`, `initPlayers`, `AddPlayer`, et désormais `PresenceBatch`) le lisent
  sous ce verrou. Course théorique préexistante, non aggravée (le nouveau lecteur
  prend le verrou comme ses voisins). Non traité, hors périmètre.
- EMOJI VERSIONNÉ résiduel du lot E (`batch_client.go:75`, « U+26A0 » en godoc) :
  TRAITÉ par le superviseur (`995bfdc6e`) — remplacé par « ATTENTION : ». Aucun
  garde-rail ne rattrape ce cas (le gate F était vert avec) : un ratchet « pas
  d'emoji dans les fichiers versionnés » reste à créer, non traité ici.
- `formatDurationMMSS` confond « zéro » et « absent » pour TOUS ses appelants (lot F,
  2026-08-25) : son repli sort dès que la valeur est nulle. C'est juste pour une durée
  de MATCH (l'origine du helper) et faux partout où zéro est une mesure. F12 n'a corrigé
  que le tableau des usages d'équipement, par un helper local. Un autre appelant a la
  même forme et mériterait une vérification : `MatchScoreboard.tsx:110`
  (`avg_life_seconds`, repli « — »). Non traité, hors périmètre.
- La borne de fraîcheur de la présence (F9) REPOSE SUR UNE PROPRIÉTÉ DU POLLER : les
  events partent à chaque poll réussi, pas seulement aux changements d'état (vérifié sur
  `rest_poller.go:153-163`). Si un jour le poller se met à ne dispatcher que les
  transitions — une optimisation plausible —, `presenceFreshnessWindow` effacerait le
  titre d'un joueur bel et bien en partie au bout de 3 minutes. La dépendance est écrite
  au godoc de la constante ; elle n'a pas de garde-rail automatique.
- `MatchViewPage.tsx` coerce encore `locale === 'en' ? 'en' : 'fr'` sur une valeur
  déjà typée `Locale` (`'fr' | 'en'`) avant de la passer à `MatchMediaTab` et au
  breadcrumb : ternaires no-op héritées. Les deux composants d'onglets extraits
  reçoivent la `Locale` telle quelle. Nettoyage du reste non tenté (hors périmètre).

### Ronde 2 de revue adversariale (2026-08-25) — 0 P0, 0 P1, 4 P2 consignés

La ronde 2 a relu les seuls correctifs du lot F (diff `4450fa81d..995bfdc6e`) :
14 conditions vérifiées tiennent (singleflight sans interblocage, clé de cache par
gamertags saine, timeout sans fuite, ratchets propres, repli du nom d'assistant,
« 0:00 », tests F non tautologiques sauf un — ci-dessous). Les 4 P2, consignés
sans ronde 3 (borne du skill `adversarial-review`) :

- **Interaction budget 3 s × backoff d'échec (F7×F4)** : un `Count` qui dépasse le
  budget avorte le lot Xbox (même contexte), l'échec arme le backoff ~30 s — dans
  un régime où Xbox dépasse durablement 3 s, `friends_in_game` reste à 0 en
  permanence (avant lot F : juste mais lent). Dégradation ASSUMÉE, godoc de
  `friendsCountBudget` corrigé par le superviseur pour dire la vérité (le texte
  affirmait une poursuite en arrière-plan qui n'existe pas). Condition de
  reprise : si la pastille d'amis reste à zéro en usage réel, dissocier le
  contexte du fetch de celui de la requête (fetch détaché qui alimente le cache).
- **Test F9 partiellement tautologique** (`presence_service_test.go:203-222`) : la
  fixture « frais sans titre + périmé avec titre » ne discrimine pas le placement
  de la borne à l'INGESTION (la décision structurante de F9) d'un blanchiment
  post-arbitrage. Cas discriminant manquant : périmé+titré énuméré en PREMIER,
  frais+titré en second. Le garde lui-même est correct (vérifié sur pièces).
- **`enableSorting: false` sur la colonne rejeu = configuration inerte** : une
  colonne d'affichage sans `accessorFn` n'est jamais triable dans TanStack
  (`getCanSort` exige `accessorFn`) — le constat de ronde 1 était donc erroné sur
  la conséquence, le correctif est sans effet rendu, et son test ne peut pas
  rougir. Gardé pour l'explicite (parité avec la jumelle escouade).
- **`presenceFreshnessWindow` (3 min) < backoff 429 du poller (5 min)** : pendant
  un épisode 429 Xbox, un joueur réellement en partie est servi « hors jeu »
  ~2 min (avant F9 : titre conservé). Rien ne relie les deux constantes (paquets
  distincts, pas de garde-rail croisé). Arbitrage assumé : mieux vaut un faux
  « hors jeu » transitoire qu'un faux « en jeu » persistant ; à re-regarder si les
  429 deviennent fréquents.
