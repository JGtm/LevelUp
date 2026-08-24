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

- [ ] B1 — layout : `z.enum(['summary','chronology','players'])`, avec rétro-compat
  des deep-links : `tab=details` accepté et résolu vers `chronology` (alias au
  décodage, patron `TAB_ALIASES` de settings ; pas de redirect).
- [ ] B2 — `MatchViewPage.tsx` : table `TABS` à 3 entrées ; les CONTENUS des onglets
  Chronologie et Joueurs sont EXTRAITS en deux composants
  (`MatchViewTabChronology.tsx`, `MatchViewTabPlayers.tsx`) — `MatchViewPage.tsx`
  doit passer SOUS 500 lignes à la clôture (il est à 523 : l'extraction paie la
  dette au passage, sans changer le rendu des blocs déplacés).
- [ ] B3 — requêtes : `useMatchObjectiveEvents` et `useMatchPositions` ne doivent
  être actives que quand leur onglet les affiche (aujourd'hui tirées dès l'arrivée
  pour des composants de Détails ; utiliser `enabled:` selon l'onglet actif —
  l'artefact de rejeu et engagement sont déjà conditionnels par le rendu).
- [ ] B4 — i18n : `tabChronology` FR « Chronologie » / EN « Timeline »,
  `tabPlayers` FR « Joueurs » / EN « Players » ; `tabDetails` supprimé des DEUX
  tables (0 code mort) ; les sous-titres de sections conservés tels quels dans
  leurs onglets.
- [ ] B5 — tests : vitest — l'alias `details` → onglet Chronologie actif ; chaque
  onglet rend ses sections attendues (smoke par titre de section) ; typecheck.
- [ ] B6 — NE PAS brancher `MatchNarrativeSection.tsx` (composant orphelin constaté)
  → Découvertes.

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

- [ ] C1 — Go SQL : nouvelle requête (Q21d) sur `match_kill_events_latest` :
  `WHERE match_id = ? AND publishable AND assist_known AND assist_gamertag IS NOT
  NULL AND assist_xuid IS NOT NULL`, `GROUP BY assist_xuid, assist_gamertag,
  feed_killer_xuid` avec `COUNT(*)` et `COUNT(*) FILTER (assist_damage_pct >
  killer_damage_pct)` (kills volés). Agrégat par match_id direct — PAS de clé
  temporelle, donc pas de correction T0.
- [ ] C2 — Go domain/service : type `MatchAssistPair` (`assist_xuid`,
  `assist_gamertag`, `killer_xuid`, `killer_gamertag`, `assist_count`,
  `stolen_count`) sur le modèle de `MatchKillerVictimPair` ; nouveau bloc
  `combat_tab.assist_pairs` + indicateur de mesure distinct (le contrat DOIT
  permettre de distinguer « non mesuré » de « mesuré, zéro paire » — s'aligner sur
  le patron de couverture du kill feed existant ; attention au piège huma
  nullable-arrays déjà documenté dans le dépôt). Gamertags du tueur résolus via le
  scoreboard comme `buildKillerVictimPairs`
  (`match_view_builders_combat.go:159-218`). Loader dans
  `match_view_data_loaders.go` à côté de `killAssists`.
- [ ] C3 — contrats : openapi + `make generate-types` ; gate openapi vert.
- [ ] C4 — web : `MatchAssistChart.tsx` (clone structurel de
  `MatchAntagonistChart`) : 1 barre par ASSISTANT, segments = tueurs assistés ;
  infobulle : nb d'assists + « dont volés : N » ; état vide « Assistance non
  mesurée pour ce match » ≠ « Aucune assistance » selon l'indicateur C2 ; série
  via un `assistStackedSeries` dans `_chartSeries.ts` (même tri, même palette de
  tokens). Monté dans l'onglet **Joueurs** (lot B), sous `MatchAntagonistChart`.
  i18n FR/EN complet.
- [ ] C5 — web : ne PAS toucher au kill feed existant ni à `assist_state` (livré).

### Items — page escouade

- [ ] C6 — Go : agrégat par paire scopé aux matchs de l'escouade dans le service
  teammates (patron Q32c/Q28Scoped) : par (assistant, tueur) au sein de l'escouade
  → `assist_count`, `stolen_count`, plus dénominateur de couverture
  `matches_measured` / `matches_total` (nb de matchs de la sélection ayant au
  moins une ligne `assist_known=TRUE`). Bloc ajouté à `TeammatesPageResponse`.
- [ ] C7 — web : tableau dans la page **Synergies** de l'escouade (TanStack Table) :
  colonnes assistant, bénéficiaire, assists (brut), part (% des assists mesurées
  de l'escouade), kills volés ; bandeau de couverture « mesuré sur N des M matchs »
  (patron `killFeedWeaponCoverage`). i18n FR/EN. Bascule %/brut : les DEUX colonnes
  affichées (pas de toggle).
- [ ] C8 — tests : Go = test du builder des paires (fixtures avec les 3 états
  d'assist ; kills volés ; unmeasured vs zéro) + test service teammates du bloc ;
  web = vitest chart (série, état vide double) + tableau escouade (couverture
  affichée).

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

- [ ] D1 — `equipmentUsageLogic.ts` (features/match-replay) : agrégation PURE
  `doc → { byPlayer, byTeam }` des grandeurs : tractions de grappin, épisodes camo,
  épisodes surbouclier (nombre + durée cumulée), déploiements par famille, objets
  lâchés à la mort par famille, grenades par type ; et par match : socles de
  power-up vidés (anonyme). Dénominateurs de couverture repris de
  `doc.coverage` (`equipment.tracksTotal`, `grapple.pullLives`,
  `placements.byFamilyOrigin`, `groundWeapons.powerupPads`). Tests vitest complets
  sur fixtures synthétiques (y compris joueur hors scoreboard).
- [ ] D2 — composant `MatchEquipmentUsageSection.tsx` (features/match-view ou
  match-replay selon l'existant du lot B) : tableau par joueur avec ligne « Total
  équipe » (patron `MatchObjectivesSection.tsx`), ligne à part pour l'anonyme
  (« Socles de power-up vidés : N » au niveau match), libellés des familles via les
  labels de rejeu existants (jamais en dur), réserve affichée pour camo/surbouclier
  (« état actif mesuré — source socle ou capacité non distinguée », en infobulle).
  Double porte : artefact présent (`header.replay_available` + donnée non vide)
  sinon rien. i18n FR/EN.
- [ ] D3 — montage dans l'onglet **Chronologie** (lot B), après la courbe de score ;
  consomme `useMatchReplay` (déjà utilisé par `MatchScoreCurveChart` — même cache,
  aucun fetch additionnel).
- [ ] D4 — INTERDITS respectés : aucun accès `filmdec/`, aucun bump de schéma,
  aucune modification de `ReplayCanvas.tsx` (cliquet 777), aucune table DB.
- [ ] D5 — tests : vitest logique (D1) + rendu (tableau, porte artefact absent,
  ligne anonyme) ; typecheck.

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

- [ ] E1 — Go watcher : stocker le titre courant (`player_watcher.go` : champ +
  enregistrement au bon endroit du handler — cf. piège ci-dessus) ; le remonter
  dans `PlayerPresenceStatus` (`provider.go`) : `in_game`, `title_slug`,
  `title_name`.
- [ ] E2 — Go présence des amis : appel batch
  `POST userpresence.xboxlive.com/users/batch` (nouvelle méthode du
  `PresenceClient`, même auth, même contract-version ; réutiliser
  `ParsePresencePayload` par élément) ; résolution des `friend_gamertags` → xuids
  via `v_gamertag_lookup` ; calcul « amis en jeu » = présence sur N'IMPORTE quel
  titre supporté (`titleReg.MatchPresence(titleID) != nil`) ; cache TTL en mémoire
  30-60 s (patron `privacyTTLCache`), calcul À LA DEMANDE (pas de poller dédié).
  Amis à la présence masquée (privacy) : ignorés silencieusement du compte, avec un
  `slog.DebugContext` — jamais d'erreur utilisateur.
- [ ] E3 — Go endpoint : `GET /api/v1/presence` sous `RequireAuth` + `NoStore`
  (PAS admin), servi depuis `watcher.WatcherStateProvider` + le calcul E2 :
  `{ players: [{player_slug, gamertag, in_game, title_slug, title_name}],
  friends_in_game: N }` ; joueurs filtrés par `filterOwnedPlayers` (ADR 0029) ;
  si daemon absent/éteint → `players: []`, `friends_in_game: 0` (200, jamais 500).
  Handler sans logique métier : le calcul vit dans un service.
- [ ] E4 — contrats : openapi + `make generate-types`.
- [ ] E5 — web : hook `usePresence()` (clé dans `lib/query/keys.ts`,
  `refetchInterval: 30_000`, `staleTime` cohérent) ; remplacement du `<select>`
  natif de `NavL1.tsx` (l.411-432) par un dropdown custom sur le gabarit
  `SplitButton`/`SettingsSplitButton` du même fichier (`role="menu"`,
  click-outside, navigation clavier) — comportement de bascule joueur inchangé
  (`handlePlayerChange`) ; icône manette SVG inline à droite des users en jeu ;
  pour le joueur ACTIF : badge compteur à droite (« N » + libellé accessible
  « N amis en jeu » FR / « N friends in game » EN) rendu seulement si N > 0.
  Aucune couleur en dur — tokens.
- [ ] E6 — tests : Go = parser batch + logique « amis en jeu » (fixtures presence)
  + handler httptest (daemon absent → réponse vide) ; web = vitest du dropdown
  (liste, icône conditionnelle, compteur conditionnel, bascule joueur) ;
  typecheck.

### Gate E

```
cd apps/go-api && go vet ./... && go test ./internal/presence/... ./internal/watcher/... ./internal/service/... ./internal/api/... ./contracttest/...
cd apps/web && npx tsc -b --force && npx eslint <fichiers touchés> && npx vitest run <tests shell/NavL1>
```

---

## Clôture du chantier (superviseur)

- [ ] Gate global dans le worktree : `cd apps/go-api && go test ./...` +
  `cd apps/web && npx tsc -b --force && npx vitest run` — verts.
- [ ] Revue adversariale du diff complet (skill `adversarial-review`) avant fusion.
- [ ] Fusion `wt/notion-cinq` → `feat/v75` en `--no-ff` (superviseur), journal
  `.ai/thought_log.md` + entrée registre si reports ; suppression worktree+branche.
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
