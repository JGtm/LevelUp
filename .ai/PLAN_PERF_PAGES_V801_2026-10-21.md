# Plan — pages : mesure et performance, résumés par match (v8.0.1, 2026-10-21)

Origine : frags assistés absents des tuiles de l'Accueil en prod le 2026-10-20 (Q26l non bornée
sous la fenêtre `_latest`, corrigée par `2af386c06`). Le user demande d'en traiter la cause de fond
de façon holistique et solide, sans régression, avec l'implémentation dans la v8.0.1. Le plan porte
DEUX VOLETS COMPLÉMENTAIRES sur un socle commun :
- **Volet 1 — mesure et performance** : ce que les pages mesurées ou signalées lentes paient pour
  rien (historique complet pour une fenêtre, fichiers complets pour quelques graphiques).
- **Volet 2 — changement de paradigme** : ce qui se recalcule à chaque requête depuis le journal
  du film et se décompose par match est calculé UNE fois, rangé en base, et lu par les pages.

Contrat d'exécution : skill `plan-execution` (fait foi). Statuts `[x]` fait, `[~]` couvert ailleurs
(référence), `[!]` non traité (justification écrite) ; aucune case vide à la clôture. Étape N close
(gate passé, cases statuées, entrée `thought_log`) avant N+1, dans l'ordre du §5. Découverte hors
périmètre : section « Découvertes », jamais traitée ici. Reprise de session : relire ce fichier
(cases cochées), puis `git log --oneline -15` dans le worktree.

Règle de non-sur-correction (user, 2026-10-21) : une page n'a d'étape de correction que si le banc
la mesure lente ou si le user la signale. Un constat d'audit sans mesure reste un CANDIDAT (§6) ;
mesuré sans problème, il est statué `[~]` « mesuré, rien à faire ».
EXCEPTION ASSUMÉE : le volet 2 n'est pas une correction de page, c'est le changement de paradigme
demandé par le user (2026-10-20 : « intéressant de sauver certaines choses à chaque match » ;
2026-10-21 : « le point initial de ce plan c'était justement ce changement de paradigme »). Il
s'applique à tous les lecteurs de classe A, y compris ceux d'un seul match (classe C servie par la
même famille : une seule source pour un même chiffre), avec deux exigences chiffrées : parité exacte,
et aucune page plus lente qu'avant (médiane du banc ≤ +10 %).

Branche `feat/v801-perf-pages`, worktree `LevelUp-wt-chronos`, partie de `origin/feat/v75`
`affab87ed` (rebasée avant l'étape 1 si `feat/v75` a avancé). Intégration dans `feat/v75` :
DEMANDÉE AU USER avant chaque intégration (mémoire « accord donné : revenir pour la fusion
feat/v75 »), puis fast-forward de la ref distante (`git push origin feat/v801-perf-pages:feat/v75`
depuis le checkout principal, après `git merge-base --is-ancestor origin/feat/v75 feat/v801-perf-pages`),
sinon rebase dans le worktree ; jamais de modification de l'arbre de travail d'une autre session.
Fusion vers `main` + tag v8.0.1 : geste du user (session de release). Push `main` : jamais.

Environnement Go : `GOCACHE=C:\Users\Guillaume\AppData\Local\go-build-perf-pages`,
`GOLANGCI_LINT_CACHE=C:\Users\Guillaume\AppData\Local\golangci-perf-pages`, gcc
`C:\msys64\ucrt64\bin`. Jamais `go build ./...`. Un seul `go` à la fois. Intégration
`-tags=integration -p 1`. Push depuis le checkout principal si le crochet knip échoue dans le
worktree (pas de `node_modules`) ; jamais `--no-verify`.

## 1. Pièces du dossier

- Journal prod 2026-10-20 : Accueil 15,2 s en moyenne (16 requêtes, max 29,2 s), vue match 14,6 s
  (5, max 21,8 s) ; frags assistés en échec de délai à chaque Accueil de JGtm et Madina97294.
- Mesure navigateur levelup-76 (prod, JGtm, iPhone émulé) : Accueil 28,6 s puis 20 s, 2,9 Mo de JSON
  décodé, `recent_matches` = 1 182 éléments, 158 000 nœuds DOM, 161 Mo de tas JS ; rechargement
  « analyse en cours » de 60 s sur toute la liste.
- Vue match : rencontres bornées par `6cb130a4f` (local 4,9 s → 1,3-1,5 s, levelup-76) ; à re-mesurer.
- `.ai/AUDIT_SURCHARGEMENT_PAGES_2026-10-21.md` : données envoyées vs lues (candidats du volet 1).
- `.ai/INVENTAIRE_RECALCULS_FILM_2026-10-21.md` : les lecteurs qui recomptent le journal du film,
  leurs doublons, le modèle du résumé d'usage (volet 2).
- Réglages DuckDB de prod : défauts du code (`platform/duckdb/db.go` : 2 fils, 512 Mo), lus dans le
  conteneur le 2026-10-21 ; VPS 2 vCPU, 1,8 Go.
- Hors périmètre, fait : page de rejeu allégée (levelup-76, `6eb8ffae8`) ; sections de durée
  `home_*` / `mv_<source>` (`9d40cc1b9`).

## 2. Critères de succès

Socle : banc reproductible (±20 % entre deux passages), sans aucun effet hors du worktree.
Volet 1 : Accueil (JGtm, Nuzzles, Madina97294) — réponse de tête < 300 Ko décodés, au plus 30
cartes montées au premier rendu, durée serveur réduite d'un facteur fixé APRÈS la ventilation par
section du banc (étape 2) ; vue match — selon la mesure de l'étape 2 (§5, étape 5).
Volet 2 : aucun lecteur de requête de classe A ne lit plus `match_kill_events_latest` (garde-rail au
niveau de la requête) ; chiffres identiques avant/après pour les trois comptes Infinite ET les comptes
Halo 5 de la copie (parité sur copie des bases) ; aucune page plus lente (médiane ≤ +10 %) ; jauge
« matchs sans tête de résumé » à 0 après rattrapage.
Après le déploiement : `http_timings` de prod relevées en lecture seule et comparées à l'annexe A.

## 3. Décisions

Fermes (tranchées sur pièces) :
- **D-1 Fenêtre.** Une réponse de page ne porte pas une liste qui croît avec l'historique du joueur
  quand la page n'en affiche qu'une fenêtre. Invariant ADR 0036 I8 + garde-rail.
- **D-2 Un seul constructeur de tuiles.** Tête et pages suivantes passent par la MÊME fonction de
  construction (`buildTiles(window)`), traductions d'assets comprises.
- **D-3 Totaux sur l'historique.** Héros, faits marquants, sessions, placements de performance
  restent calculés sur les lignes canoniques complètes (en cache) ; seule la liste servie est fenêtrée.
- **D-4 Parité avant bascule, suppression dans la même étape.** Ancien et nouveau calcul comparés
  par test ; l'ancien chemin et ses tests supprimés dans la même étape (règle 7), entrées de la
  baseline de tests retirées dans le même commit.
- **D-5 Banc isolé.** Copie des bases dans le worktree, port 8010, réglages DuckDB de prod,
  `LEVELUP_AUTH_MODE=none`, `LEVELUP_AUTH_DIR` sur un dossier vide, AUCUN fichier de `data/auth/`
  copié, `app_settings.json` assaini (webhook Discord vidé, watcher / auto-sync / surveillance médias
  coupés), `HTTPS_PROXY` et `HTTP_PROXY` vers un port fermé. Le serveur local des autres sessions
  n'est jamais arrêté.
- **D-6 Résumés écrits avec la passe, en un point.** `insertKillEventRows`
  (`internal/persist/kill_events_persister.go:258`) est « le seul écrivain de match_kill_events du
  dépôt » ; les deux producteurs (film `PersistPass`, crédit `persistCreditKillEvents`) et leurs
  appelants (sync primaire Halo Infinite et Halo 5, complétion des événements, collecteur killsource,
  `BatchBuilder`, CLI de backfill) y passent. Les résumés sont calculés en Go depuis les lignes de la
  passe (passe entière en mémoire : `len(Deaths) >= CreditBaseCount`), APRÈS les mêmes normalisations
  que `killEventArgs` (`""` → NULL, `SourceTag == 0` → NULL), et insérés dans la même transaction, avec
  le même `now` Go que la passe (jamais un `DEFAULT now()` SQL) et la même règle de tri que la vue
  (`written_at DESC, id DESC`). Une erreur de calcul annule la passe (couplage assumé, testé par
  injection d'erreur). Fonction pure sur un type de ligne neutre, avec deux adaptateurs : lignes en
  mémoire (écriture) et lignes lues en base (rattrapage), tenus égaux par un test de parité.
- **D-7 Familles du volet 2, grain et colonnes figés à l'étape 6 avant tout code.** Chaque famille
  porte `summary_pass` (une génération), `summary_rev` (révision de la règle), `src_decode_pass`,
  `written_at` ; vues `_latest` jointes à la tête sur `summary_pass` (modèle `match_usage_*`), pour
  qu'une relecture à révision nouvelle sur la même passe ne compte jamais deux fois.
  - S0 `match_kill_heads` (match) : comptes de la passe — morts, publiables, publiables avec tueur
    nommé (dénominateur de `match_range`), publiables avec assistance lue (porteuses), drapeau de journal.
  - S1 `match_kill_pairs` (match, tueur xuid+gamertag, victime xuid+gamertag, `source_tag`) : `kills`,
    `kills_publishable` ; NULL conservés (bots, environnement, source inconnue). Sert les paires
    (somme sur la source), les armes (somme sur la victime) et le journal d'armes de l'Emprise
    (`LoadJournalWeaponKills`). Une seule table au lieu de deux : à confirmer à l'étape 6 par le
    nombre de lignes sur la copie (si la table approche la taille du journal, scinder par usage).
  - S2 `match_assist_pairs` (match, assistant xuid+gamertag, tueur xuid+gamertag, victime xuid
    seulement si assistant et tueur sont sans xuid) : `assists`, `stolen`, somme et nombre des parts
    non NULL, tranches basse/moyenne/haute, ET une ligne « mesurée sans assistant » par tueur (état
    « personne n'a assisté » lu par la Coordination).
  - Seuils de tranche liés à `summary_rev` : test qui échoue si `AssistTierLowMaxPct` /
    `AssistTierMidMaxPct` changent sans montée de révision.
  - Candidats de grain par frag ou par vie (`match_measured_kills`, `match_life_windows`) : §6.

À valider par le user, par message daté, AVANT l'étape indiquée. D-8, D-9 et D-10 VALIDÉES par le user le 2026-10-21 (« Vas y go », en réponse au message qui les listait explicitement, avec le feu vert sur le plan). D-11 reste à valider après l'étape 2.
- **D-8 (avant étape 3)** 30 tuiles par page ; page suivante demandée à 5 tuiles de la fin ;
  pagination par curseur (instant canonique `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`
  via le fragment partagé — règle 8 —, puis `match_id`) avec `has_more` (pas de décalage : une synchro
  entre deux pages créerait doublons ou trous) ; favoris servis en entier (liste choisie par le joueur,
  bornée par son geste) avec leur propre requête.
- **D-9 (avant étape 3)** Navigation précédent / suivant depuis une tuile de l'Accueil : la liste
  d'ids n'est plus transportée ; les voisins viennent de l'API (Q25, tout l'historique), et
  `useMatchNeighborsResolved` rend le libellé de contexte (« Matchs récents X/Y » avec Y = total).
- **D-10 (avant étape 8)** Rattrapage du parc et mise en service : les résumés des matchs existants
  (Halo Infinite ET Halo 5) sont calculés EN LOCAL, serveur local arrêté avec l'accord du user (le CLI
  ouvre la base en écriture), puis les bases rejoignent le VPS selon la séquence de release : arrêt de
  la prod, copie des bases, déploiement du nouveau code, démarrage (comme la v8.0.0). Les lecteurs
  basculent dans la même release : pas de double chemin en prod. Une montée de `summary_rev` ensuite
  suit la même séquence (recalcul local + copie). Alternative refusée par défaut : rattrapage
  automatique en prod avec lecture hybride (double chemin temporaire).
- **D-11 (avant étape 5, si l'étape a lieu)** Vue match : projection serveur des seuls calques lus
  par ses graphiques, type dédié (précédent `ReplayMatchContext`), lecture serveur sans décoder tout
  l'artefact à chaque requête ; le partage de cache vue match ↔ rejeu disparaît pour ces graphiques.

## 4. Doctrine anti-régression (commune)

Pour chaque étape qui change du code, AVANT de cocher son gate :
1. Banc avant/après (mêmes comptes, mêmes matchs), consigné en annexe A.
2. Parité : test qui compare l'ancien et le nouveau résultat sur un corpus ; liste paginée :
   concaténation des pages = ancienne liste, même ordre, champ pour champ ; résumé : lecteur ancien
   et nouveau égaux sur corpus ET sur la copie des bases pour les trois comptes.
3. Contrat : OpenAPI régénéré (`make generate-types`) pour les DTO décrits ; pour l'Accueil (corps
   servi en octets, pas de schéma) : miroir `apps/web/src/lib/api/types.ts`, `test/handlers.ts`,
   test `jsonshape` et `RequireNoNilSlicesWithoutOmitempty` ; `make check-types`.
4. Suites : `go test ./internal/<paquets touchés>/...` puis `-tags=integration -p 1` sur les mêmes ;
   persist/sync touchés : `go test -tags=integration ./internal/persist/... ./internal/sync/...`
   (anti-ART obligatoire) ; `npx vitest run <dossiers touchés>` ; lint
   `golangci-lint run --new-from-merge-base=origin/main <paquets>` sans problème ;
   `bash scripts/check_test_baseline.sh` si des tests sont supprimés ou renommés.
5. Web touché : `npx playwright test` en local (la CI ne lance l'E2E que sur les PR vers `main`).
6. Revue adversariale (skill `adversarial-review`) : étapes 3, 4, 7, 8, 9 (et 5 si elle a lieu).
7. Entrée `thought_log` par étape.

## 5. Étapes (ordre d'exécution)

### Étape 1 — Banc de mesure (socle)

- [ ] 1.1 Recenser les tâches lancées au démarrage du serveur (`cmd/server/main.go`,
      `background_tasks.go`) et, pour chacune, comment elle est neutralisée sur le banc (réglage,
      variable, absence de jeton, proxy fermé). Tâche qui écrirait hors du worktree ou contacterait
      un tiers malgré le proxy : `[!]` et arrêt, on remonte au user.
- [ ] 1.2 Place disque : au moins 10 Go libres sur le volume du worktree (`df -h`), sinon arrêt.
- [ ] 1.3 Copie dans `LevelUp-wt-chronos/data/` (fichiers ignorés par git uniquement, liste écrite
      dans `scripts/bench-pages.files`) : `titles/halo_infinite/warehouse/*.duckdb` ET leurs `.wal`,
      `titles/halo_infinite/players/{JGtm,Nuzzles,Madina97294}/`, `global/`, `db_profiles.json`,
      `app_settings.json` assaini (D-5), les artefacts de rejeu des trois matchs de référence (copie,
      jamais de jonction). Jamais `data/auth/`. Copie faite quand aucun sync local n'écrit (heure
      consignée).
- [ ] 1.4 `scripts/bench-pages.sh` (versionné) : lance le serveur du worktree (D-5,
      `LEVELUP_DUCKDB_THREADS=2`, `LEVELUP_DUCKDB_MEMORY_LIMIT=512MB`, `LEVELUP_SLOW_REQUEST_MS=1`,
      `LEVELUP_LOGS_DIR` dans le worktree) ; pour chaque compte et endpoint : 1 appel de chauffe +
      5 appels ; médiane, max, octets décodés ; extrait les `http_timings`.
- [ ] 1.5 Endpoints : Accueil, vue match (un match avec rejeu par compte, identifiants consignés),
      artefact de rejeu, Explorer `matches-query` (requête par défaut du web), Relations, Carrière +
      rencontres + rivaux, Timeseries, Escouade, Comparaison (JGtm vs Madina97294).
- Gate : deux passages à ±20 % ; `ls data/auth` → absent ; compteurs expvar `halo_api_*` à 0 après
  le passage (`curl -s localhost:8010/debug/vars`) ; aucun message parti (webhook vide) ;
  `git status --short data/` ne montre aucun fichier suivi modifié.

### Étape 2 — Mesure « avant » et classement des pages

- [ ] 2.1 Banc complet, annexe A remplie (durées, octets, sections `home_*` / `mv_<source>`).
- [ ] 2.2 Classement : chaque page « lente » (médiane > 2 s sur un compte) ou « ok » ; chaque
      lecteur de l'inventaire chiffré via ses sections. Conséquences consignées : étape 5 (vue match)
      et candidats §6 statués « à faire » ou `[~]`.
- [ ] 2.3 Cible chiffrée du volet 1 pour l'Accueil fixée depuis la ventilation (critère §2).
- Gate : annexe A complète pour les trois comptes ; chaque candidat du §6 a un statut.

### Étape 3 — Accueil en fenêtre, serveur (volet 1)

- [ ] 3.1 Extraction `buildTiles(ctx, rows, window, favorites)` depuis `GetHomePage` (≤ 80 lignes
      après extraction) : traductions d'assets (`EnrichCanonicalAssetTranslations`) sur la fenêtre,
      construction, enrichissements par liste, placements, rejeux, analyse en cours.
- [ ] 3.2 `GetHomePage` sert la tête (D-8) + `has_more` + `recent_matches_total` (= nombre de tuiles
      que le constructeur produirait, lignes à `MatchID` non vide, pas `CountPlayerMatches`).
- [ ] 3.3 Endpoint Huma `GET /players/{player_slug}/pages/home/matches?before=&limit=` (limite max
      60, curseur D-8) et `?kind=favorites` ; factory authentifiée `HomeCtxWithAuth` (parité
      `home_factories_parity_test.go`) ; handler sans logique ; méthode ajoutée au port et aux faux
      (`health_home_test.go`, `home_test.go`, `og_inject_test.go`, `season_pass_service_test.go`).
- [ ] 3.4 `buildFavoriteMatchListCanonical` : construit seulement les favoris.
- [ ] 3.5 `last_session_outcomes` : tranche non nulle de `{match_id, outcome_tone, detail}`, même règle
      que `HomePage.tsx` l.204-213, calculée sur les lignes complètes.
- [ ] 3.6 Retrait de `recent_media` : `port.HomeRepository.LoadRecentMedia`, `noopHomeRepo`, mocks,
      `Q28RecentMedia`, `analysis.BuildRecentMedia`, `domain.RecentMediaItem`, `HomeMediaRow`,
      `lib/api/types.ts`, `test/handlers.ts`, tests web concernés (grep consigné).
- [ ] 3.7 Retrait de `HomeMatchesCache` : `home_service.go`, `wire/registry.go`,
      `registry_auth.go`, `registry_pages_home.go`, `post_sync_deltas.go`,
      `post_sync_home_cache_test.go`, `home_factories_parity_test.go` (grep consigné).
- [ ] 3.8 ADR 0036 : invariant I8 (D-1) ; garde-rail `TestGetHomePage_FenetreBornee` (200 matchs →
      au plus 30 tuiles) ; test de la fenêtre aussi avec le slug `halo_5` (aucune comparaison de slug).
- [ ] 3.9 Parité : historique de 200 matchs, concaténation des pages = ancienne `recent_matches`,
      champ pour champ ; une synchro simulée entre deux pages ne crée ni doublon ni trou.
- Gate : doctrine §4 ; revue adversariale close.

### Étape 4 — Accueil en fenêtre, web (volet 1)

- [ ] 4.1 La requête de tête garde `useQuery` sous `queryKeys.home` (prefetch, loader de route,
      invalidations existantes intacts) ; pages suivantes et favoris sous des clés dédiées de
      `lib/query/keys.ts`.
- [ ] 4.2 `components/ui/carousel.tsx` : rappel « proche de la fin » et état de chargement (chaînes
      FR et EN) ; le carrousel ne monte que les tuiles chargées.
- [ ] 4.3 Extraction de la section des matchs hors de `HomePage.tsx` (exemption `max-lines` de la
      ligne 1 : « le découpage revient au lot qui touchera ce fichier »).
- [ ] 4.4 Navigation (D-9) : `goToMatch` sans `matchIds` pour `home_recent` ;
      `useMatchNeighborsResolved` rend le libellé de contexte ; test de parité Q25 / ordre de
      l'Accueil sur les trois comptes (copie des bases).
- [ ] 4.5 Analyse en cours : le rechargement de 60 s porte sur chaque page chargée qui contient une
      tuile en attente, et seulement sur elle (`homeRefetchInterval`, `homeStaleTime`).
- [ ] 4.6 Favori : mise à jour locale de la tuile, rechargement de la seule liste des favoris.
- [ ] 4.7 `OutcomeSequenceTape` lit `last_session_outcomes`.
- [ ] 4.8 Tests vitest (montage limité, page suivante, navigation, rechargement, favoris) ; E2E local.
- Gate : doctrine §4 ; banc : critères Accueil du §2 tenus ; revue adversariale close.

### Étape 5 — Vue match (volet 1, conditionnelle à l'étape 2)

- [ ] 5.1 Si l'étape 2 mesure la vue match « ok » (rencontres déjà bornées par `6cb130a4f`) ET
      l'artefact de rejeu non dominant : `[~]` « mesuré, rien à faire ». Sinon, D-11 validée, puis :
- [ ] 5.2 Inventaire des champs lus par `MatchScoreCurveChart`, `MatchScoreEventsChart`,
      `MatchImpactBadgesBar`, `MatchViewTabArsenal`, `MatchEquipmentUsageSection` (montée par
      l'Arsenal), y compris `coverage.score.modeSupported` ; consigné ici.
- [ ] 5.3 Type dédié + endpoint de projection + stratégie de lecture serveur (calques écrits à la
      cuisson, ou cache par (match, date de l'artefact)) ; parité champ pour champ avec le document.
- [ ] 5.4 Les composants lisent la projection ; la page de rejeu inchangée ; prévenir levelup-76
      avant de toucher au rejeu s'il est actif.
- Gate : doctrine §4 ; captures avant/après des cinq composants sur les trois matchs de référence.

### Étape 6 — ADR 0039 « résumés par match » : règle, grain, colonnes (volet 2)

- [ ] 6.1 ADR 0039 (EN) : règle du paradigme (agrégat décomposable par match → résumé écrit avec sa
      source ; lignes brutes → lecture à la demande) ; D-6 (point d'écriture unique, normalisations,
      `now` partagé, couplage de disponibilité) ; D-7 ; tête qui décide la passe ; `summary_pass` /
      `summary_rev` ; recalcul (D-10) ; jauge « sans tête » ; garde-rail anti-retour ; compaction ;
      snapshots et démo ; multi-titre (Halo Infinite et Halo 5 écrivent `match_kill_events` : migration
      générique, aucun branchement par slug).
- [ ] 6.2 Colonnes exactes de S0, S1, S2 figées dans l'ADR, chacune justifiée par les lecteurs qui la
      lisent (table « lecteur → colonnes » reprise de l'inventaire et RE-VÉRIFIÉE sur pièces, dont les
      cas : bots sans xuid, victime bot dans `frags_film` de Q26l, tueur NULL dans
      `/monitoring/weapon-coverage`, `stolen` et portée de Q21d, « mesurée sans assistant » et victime
      du cas bot-bot de la Coordination, jeton Campagne sur `match_id`).
- [ ] 6.3 Mesures sur la copie des bases, consignées : lignes de S1 et S2 projetées par rapport au
      journal (décision « S1 unique ou scindée ») ; hétérogénéité de `publishable` dans une même passe
      (`SELECT decode_pass FROM match_kill_events GROUP BY decode_pass HAVING COUNT(DISTINCT publishable) > 1`).
- Gate : ADR relu par un contexte frais ; aucune contradiction avec les ADR 0019, 0026, 0030, 0036 ;
  chaque lecteur de 9.1-9.4 a ses colonnes dans 6.2.

### Étape 7 — Familles S0 à S2 : schéma, calcul, écriture (volet 2)

- [ ] 7.1 Migration générique (ordre canonique `migration/order.go`) : tables append-only S0-S2,
      séquences `id` propres, vues `_latest` jointes à la tête sur `summary_pass` (recette ADR 0026,
      `append_only_rebuild.go`), vues recréées après tout `ADD COLUMN` ; tables et vues ajoutées à
      `EnsureMatchKillEvents` (sinon `insertKillEventRows` échoue sur une base ou une fixture non migrée).
- [ ] 7.2 Compaction : S0-S2 dans `migration/compaction_registry.go` (`dernierePasse` pour la tête,
      passe de la vue parente pour les filles, modèle `match_usage_players`) + signatures de vue dans
      `compaction_test.go`.
- [ ] 7.3 Garde-rails ART : S0-S2 dans `tablesProtegees` de `sync/no_art_patterns_test.go` et de
      `sync/append_only_state_guard_test.go` ; aucune nouvelle entrée d'allowlist.
- [ ] 7.4 Fonction pure dans `internal/analysis/` (type de ligne neutre → S0, S1, S2) et ses deux
      adaptateurs (lignes en mémoire, lignes lues en base) ; écriture dans `insertKillEventRows` (D-6).
- [ ] 7.5 Snapshots et démo : S0-S2 dans `ops/snapshot_export.go` (`sharedSnapshotMatchKeyedRaw`) et
      dans la lecture de snapshot ; seed démo (`ops/seed_demo_shared_tables.go`, `seed_demo.go`) avec
      anonymisation des colonnes d'identité et `TestSharedTablesDeclareEveryIdentityColumn` à jour ;
      fixtures de tests qui écrivent `match_kill_events` (`sync/testutil/fixture.go` et les tests
      `INSERT INTO match_kill_events`) passent par l'écriture des résumés ou les posent.
- [ ] 7.6 Migrations historiques qui ont écrit `match_kill_events` (`steps_shared_kill_events_credit_base.go`,
      `_from_pairs.go`, `_sans_killfeed.go`) : non rejouées, nommées dans l'ADR ; le rattrapage (étape 8)
      couvre leurs passes.
- [ ] 7.7 Tests : purs (bots, assistant anonyme, cas bot-bot, mesurée sans assistant, NULL, passe crédit
      puis passe film) ; parité adaptateur mémoire = adaptateur base ; un test d'intégration par
      écrivain (sync primaire, complétion, collecteur killsource, `BatchBuilder`, backfill CLI) : tête
      S0 `src_decode_pass` = passe de `match_kill_events_latest` ; deux passes successives → vues
      `_latest` = dernière passe ; même passe relue à révision nouvelle → aucun double compte ;
      injection d'erreur dans le calcul → passe annulée ; Halo 5 ; test des seuils de tranche liés à
      `summary_rev` ; test de cohérence PERMANENT : somme des `kills` de S1 = nombre de lignes de la
      passe courante.
- [ ] 7.8 Plans d'exécution : `EXPLAIN ANALYZE` des vues S0-S2 sous une liste de matchs (le filtre
      descend-il sous la tête et ses filles ?) consigné ; les tests ADR 0036 qui exigent des fenêtres
      bornées (`exigerFenetresBornees`) restent verts ou sont réécrits pour les vues S, et le tableau
      d'exceptions I2 de l'ADR 0036 mis à jour.
- Gate : doctrine §4 (dont `-tags=integration` persist/sync) ; revue adversariale close.

### Étape 8 — Rattrapage du parc (volet 2, D-10 validée avant)

- [ ] 8.1 Commande `levelup backfill-match-summaries --title <slug>` : pour chaque match dont la passe
      courante n'a pas de tête à la révision courante (anti-join), relit la passe en base (adaptateur
      base), calcule S0-S2, écrit par lots bornés via `persist` ; serveur arrêté exigé (ouverture en
      écriture), message clair sinon. Ne décode aucun film.
- [ ] 8.2 Jauge « matchs de la passe courante sans tête » : au boot et dans
      `scheduler/data_health_check.go` (lit déjà le journal), journalisée en WARN si non nulle.
- [ ] 8.3 Passage complet sur la COPIE des bases du banc, Halo Infinite et Halo 5 ; volume (matchs,
      lignes S par match), durée et pic mémoire consignés et ANNONCÉS au user avant 8.4.
- [ ] 8.4 Passage sur les bases locales réelles (`LevelUp/data/`), serveur local arrêté avec l'accord
      du user et des sessions actives prévenues ; jauge à 0 pour les deux titres.
- Gate : doctrine §4 ; jauge à 0 sur la copie et sur les bases locales ; volume et pic mémoire consignés.

### Étape 9 — Bascule des lecteurs, famille par famille (volet 2)

Pour chaque lecteur : test de parité ancien/nouveau (corpus + copie des bases, trois comptes Infinite
et comptes Halo 5), bascule, suppression de l'ancienne requête et de ses tests (entrées de baseline
retirées dans le même commit, `bash scripts/check_test_baseline.sh`), mesure banc. Liste fermée
(re-vérifiée à l'étape 6.2) :
- [ ] 9.1 S0 (couverture et dénominateurs) : `QTacticalUnivers.mesure`, `qSoloJournalNonPubliable`,
      portée de Q21d, `matches_measured` de Q32d, `porteuses` de Q28c et Q26l, dénominateur de
      `LoadMatchRangeKills` (`matchRangeKillsTotalSQL` : publiables avec tueur nommé).
- [ ] 9.2 S1 (frags par paire, somme sur la source) : Q23b `GetMatchEncounterStats`, Q26
      `GetTopEncountersGlobal`, Q27 `GetRivals`, Q28 `GetRelations`, Q30 `GetRivalTimeline`,
      `CompareRepo.GetEncounterStats`, `ExplorerRepo.GetKillerVictimBetween`, `frags_film` de Q26l ;
      `QKillsBetweenPlayers*` et `TestUneSeuleRequeteDeFragsEntreDeuxJoueurs` remplacés par 9.5.
- [ ] 9.3 S1 (frags par arme, somme sur la victime ; tueurs NULL conservés) :
      `bulkWeaponKillsFromSource`, `GetTopWeaponsForMatches`, `LoadFavoriteWeapon`,
      `LoadWeaponKillsAggregated`, `LoadKillSourceCategoryKills`, `qSquadVehicleFrags`,
      `EventsRead` de l'Emprise (lignes à source et tueur non NULL), `LoadJournalWeaponKills`,
      `/monitoring/weapon-coverage`.
- [ ] 9.4 S2 (assistances par paire) : Q21d `GetMatchAssistPairs`, Q28c `GetRelationAssists` /
      `GetMatchEncounterAssists`, tranches de Q26l `LoadMatchAssistedFrags`, `QCoordinationAppuis`,
      Q32d `LoadSquadAssistPairs`.
- [ ] 9.5 Garde-rail archlint au niveau de la REQUÊTE (constante `KillEventsCanonicalTable` et
      littéral `match_kill_events_latest`, commentaires retirés, modèle ADR 0030 D-4) : seules les
      requêtes d'une allowlist datée et décroissante les référencent. Allowlist initiale = lecteurs de
      classe B : Q20 `GetMatchKVPairs`, Q21b `GetMatchKillSources`, Q21c `GetMatchKillAssists`, Q32e
      `LoadSquadKillLog`, replis `loadSyntheticKillEvents` / Q32c, `kill_measured.go`,
      `kill_distance_repo.go`, `weapon_range_repo.go`, `match_range_repo.go` (lignes mesurées),
      `solo_lives_repo.go`, `tactical_repo*.go`, `halo5_match_events_source.go`, et la jambe du
      kill-feed de `v_gamertag_lookup` / annuaire (Découvertes).
- Gate : doctrine §4 ; parité verte pour chaque lecteur sur tous les comptes de la copie ; aucune page
  plus lente (médiane ≤ +10 %) ; revue adversariale close.

### Étape 10 — Clôture

- [ ] 10.1 `make gate-push` vert.
- [ ] 10.2 CI verte sur la branche ; intégration dans `feat/v75` DEMANDÉE au user puis faite ; CI verte sur `feat/v75`.
- [ ] 10.3 Changelog 8.0.1 FR + EN ; `docs/COMMANDS.md` FR + EN (banc, rattrapage) ; ADR à jour ;
      séquence de release complétée (rattrapage local puis bases vers le VPS, D-10) et transmise à la
      session de release.
- [ ] 10.4 `.ai/REGISTRE_REPORTS.md` alimenté (candidats non faits, familles de second rang) ; plan
      archivé sous `.ai/V7.5/`.
- [ ] 10.5 Après le déploiement : `http_timings` de prod relevées (lecture seule), comparées à
      l'annexe A, consignées.
- [ ] 10.6 Nettoyage : seulement les fichiers listés dans `scripts/bench-pages.files` (jamais `data/`
      entier : 524 fichiers suivis) ; caches `go-build-perf-pages` / `golangci-perf-pages` purgés
      (`scripts/disk-hygiene.ps1 -Lot perf-pages -Apply`).

Charge : étapes 1-2 rapides ; 3-4 lourdes ; 5 conditionnelle ; 6 rapide ; 7-9 lourdes. Si la date de
la 8.0.1 ne permet pas tout : le user arbitre ce qui glisse en 8.0.2, à la fin de l'étape 4 ou 7 ;
une étape commencée n'est jamais livrée à moitié.

## 6. Candidats (mesurés à l'étape 2, traités seulement si la mesure le justifie)

| Candidat | Source | Statut |
|---|---|---|
| Explorer : `page_size: 10000`, ligne de 38 champs (le user la juge « plutôt ok ») | audit | à mesurer |
| Relations : `relations[]` sans limite (25 affichées ; filtres et compteurs lus côté client) | audit | à mesurer |
| Timeseries : `cumul_tab`, `intensity_tab`, `outcomes_over_time` calculés sans lecteur | audit | à mesurer |
| Carrière `current_season`, session `compare_metrics` calculés sans lecteur | audit | à mesurer |
| Bloc `coverage` du rejeu (~333 noms sur 500 sans lecteur ; badge admin, contrat, `parity_test.go`) | audit | à mesurer |
| `GET /pages/sessions`, `POST /pages/match-history/query` sans appelant web | audit | à re-vérifier |
| `match_measured_kills` (Tactique, distances), `match_life_windows` (vies près d'un coéquipier) | inventaire | à mesurer |

## Annexe A — Mesures du banc

(étape 2, puis après chaque étape de code)

## Découvertes

- `analysis/identity.go` (`v_gamertag_lookup`, annuaire des noms) lit encore `match_kill_events_latest` pour résoudre les noms : hors du critère du volet 2 (lecture de noms, pas un agrégat), allowlistée en 9.5 ; à reprendre dans un chantier « annuaire » si mesuré coûteux.
- (à compléter pendant l'exécution)
