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
(gate passé, cases statuées, entrée `thought_log`) avant N+1, dans l'ordre du §5 ; après l'étape 2, DEUX PISTES PARALLÈLES (accord user 2026-10-21), chacune en ordre strict : piste A = étapes 3, 4 (agent Sonnet, worktree dédié) ; piste B = étapes 6, 7, 8, 9 (ADR par le pilote, puis agents Opus, worktree dédié) ; piste C = étape 2b (compactage automatique, agent Opus, worktree dédié) ; l'étape 10 clôt les deux. Découverte hors
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

À valider par le user, par message daté, AVANT l'étape indiquée. D-8, D-9 et D-10 VALIDÉES par le user le 2026-10-21 (« Vas y go », en réponse au message qui les listait explicitement, avec le feu vert sur le plan). Le 2026-10-21 après l'étape 2, le user a re-confirmé D-8 (« 30 tuiles par pages c'est bien, on chargera davantage si nécessaire, en le faisant proprement pour que ça reste fluide ») et D-9 (« Ok avec ta proposition pour la navigation fournie par le serveur »), validé D-10 (« Ok pour le rattrapage en local et copie vers le VPS ») et REJETÉ D-11 (« on abandonne la vue match allégée, le gain est trop faible ») : l'étape 5 est statuée `[~]`.
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

- [x] 1.1 Recenser les tâches lancées au démarrage du serveur (`cmd/server/main.go`,
      `background_tasks.go`) et, pour chacune, comment elle est neutralisée sur le banc.
      Constat (2026-10-21) : appels Halo (crons catalogue, Spartan, classement mondial, balayage des
      noms d'assets, pool de jetons, auto-sync) → aucun jeton (dossier d'auth du banc sans
      `watcher_tokens*`) : ils s'arrêtent avant tout appel réseau (« aucune entrée store »), et le
      proxy fermé reste en second rempart ; Discord (notification de rejeux, `app_release`) →
      webhook vidé, notifications coupées ; watcher, auto-sync, surveillance médias, sauvegardes →
      coupés dans `app_settings.json` ; écritures (checkpoint social, relance des citations,
      récupération WAL, janitor, purge des rejeux, migration du groupe par défaut → `groups.json`
      dans le dossier d'auth du banc) → toutes sous la racine du worktree (`cfg.RepoRoot`). Aucune
      tâche n'écrit hors du worktree ni ne joint un tiers.
- [x] 1.2 Place disque : 330 Go libres sur `C:`.
- [x] 1.3 Copie (liste `scripts/bench-pages.files`). ÉCART CONSIGNÉ : la source est la PROD
      (`/opt/levelup`, flux `ssh … tar` en lecture seule, rien d'écrit sur le VPS), pas les bases
      locales : sous Windows, le serveur local des autres sessions tient ses bases en exclusivité
      (`cp` : « Device or resource busy ») et D-5 interdit de l'arrêter ; la prod est en outre la
      référence la plus fidèle. Copie le 2026-10-21 de 09:56 à 09:57 (bases + `.wal`), snapshot
      `v…228`, `db_profiles.json` restreint aux trois comptes Infinite (profils Halo 5 gardés),
      joueurs Halo 5 copiés (étapes 8-9), `app_settings.json` assaini dans le flux (le webhook réel
      n'est jamais écrit sur disque), artefacts des trois matchs de référence. `data/auth` absent.
- [x] 1.4 `scripts/bench-pages.sh` (`fetch`, `serve`, `measure`, `pass`, `check`, `stop`, `clean`) ;
      `pass` redémarre le serveur à neuf à chaque passage (un serveur gardé dérivait de 6 à 15 % par
      passage, mémoire privée ~1,6 Go — Découvertes).
- [x] 1.5 Endpoints mesurés (12) ; matchs de référence : JGtm `8ba9251a-53f3-4367-aa55-9c5df983120f`,
      Nuzzles `37c6e818-fc07-4b2e-90ed-f2f84c904933`, Madina97294 `a853258c-3027-4144-85c5-7309b9dc0031`.
      Les POST portent `Origin: http://localhost:5173` (CSRF par origine) ; Comparaison :
      `{"target_gamertag":"Madina97294"}`.
- Gate : [x] passages 2 et 3 à serveur neuf : 0 page hors ±20 % (max +7,5 %, hors Relations Nuzzles
  bimodale 4,3 / 6,2 / 5,0 s, à ±20 % entre les passages 2 et 3) ; `bench-pages.sh check` : `data/auth`
  absent, aucun jeton dans le dossier d'auth du banc, aucun fichier suivi modifié sous `data/`,
  webhook vide, aucune tentative `halo_api` journalisée, aucun compteur `halo_api_*` exposé.

### Étape 2 — Mesure « avant » et classement des pages

- [x] 2.1 Annexe A (A.1 durées et octets, A.2 ventilation par section).
- [x] 2.2 Classement (lente = médiane > 2 s sur un compte) :
      lentes — Accueil (Nuzzles 6,2 s), vue match (Nuzzles 4,1 s, JGtm 2,1 s), Comparaison (6,3 s),
      Carrière rencontres (Nuzzles 2,0 s), Explorer (Nuzzles 2,4 s), et en « tout l'historique »
      Relations (Nuzzles 5,0 s), Escouade (Nuzzles 3,6 s), Timeseries (7,8 à 27,6 s) ;
      ok — Carrière, rivaux (< 2 s), artefact et contexte du rejeu (< 0,15 s).
      Conséquences, avec les réponses du user du 2026-10-21 : étape 5 `[~]` (D-11 rejetée ; le temps
      de la vue match vient de Q23b et Q28c, servis par 9.2 et 9.4) ; Comparaison, Carrière
      rencontres et rivaux accélérés par le volet 2 (9.2) — le reste de la Comparaison est un
      candidat non prioritaire (« compréhensible ») ; Explorer `[~]` (« plutôt ok ») ; modes « tout
      l'historique » `[~]` (« peu utilisé, lenteur compréhensible »).
- [x] 2.3 Cibles. Accueil après l'étape 4 : réponse de tête < 300 Ko décodés, section
      `home_assisted_frags_recent` ≤ 0,2 s, durée serveur ≤ `home_favorite_weapon` + 0,5 s (l'arme
      favorite reste sur tout l'historique jusqu'à 9.3). Après l'étape 9 : Accueil et vue match
      < 0,6 s pour les trois comptes sur le banc ; Carrière rencontres et rivaux < 0,3 s.
- Gate : [x] annexe A complète pour les trois comptes ; candidats du §6 statués.

### Étape 2b — Compactage des passes : prod, puis automatisation (piste C, ajoutée le 2026-10-21)

Origine : découverte de l'annexe A.3 (bases de prod jamais compactées depuis les recuissons
d'octobre ; compactage seul = -56 à -93 % sur la plupart des pages). Accord du user le 2026-10-21 :
« ok pour l'automatisation et pour le faire dès maintenant ». Complémentaire des volets 1 et 2 : le
compactage retire les passes périmées de TOUTES les tables append-only (y compris les futures tables
de résumés S0-S2) ; il ne remplace ni la fenêtre de l'Accueil ni les résumés.

- [x] 2b.1 Compactage de la prod (2026-10-21, 10:16:58 → 10:18:46 UTC, serveur arrêté ~1 min 50,
      `docker compose run --rm --no-deps -u appuser --entrypoint levelup levelup compact-passes`) :
      halo_infinite 26,6 M → 3,74 M lignes (`match_kill_events` 7,98 M → 0,42 M), halo_5 0,83 M →
      0,57 M ; sauvegardes `…avant-compaction-20261021T1017…Z.duckdb` à côté des bases (2,3 Go +
      0,2 Go) — à supprimer après vérification par le user ; santé OK 11 s après redémarrage,
      aucune erreur journalisée.
- [ ] 2b.2 Compactage automatique dans le serveur : tâche de fond (coupée en démo, déclarée dans
      `demoPolicies`) qui, la nuit (fenêtre fixe, hors cycle de sync), mesure le rapport lignes
      brutes / lignes `_latest` de chaque table du registre de compaction et, au-delà d'un seuil
      nommé (constante, valeur justifiée par mesure), appelle `migration.CompactSupersededPasses`
      SOUS l'écrivain partagé obtenu par le chemin standard (rafale courte, comme le LUSR v2 ; jamais
      une ouverture RW hors provider/lease — règle « bare connect »), avec vérification d'empreinte
      de vue avant/après (déjà dans `echangerTable`). Pas de sauvegarde fichier en processus (la base
      est ouverte) : filet = transaction + empreintes + sauvegarde restic quotidienne du VPS ; le CLI
      `compact-passes` reste l'outil manuel avec sauvegarde. Journal INFO par table (avant, après,
      durée), WARN si un seuil est dépassé mais la compaction refusée.
      **Note de conception (2026-10-21, piste C) — BLOQUÉE sur une décision du user, aucun code écrit.**
      Acquisition : `sync.AcquireSharedWriterStandalone(ctx, cfg.SharedProviderForTitle(slug), pr.SharedDBPath(slug))`
      → `sharedprovider.(*providerImpl).AcquireWriter` (provider_writer.go), le chemin de
      `SharedAccess.Write` du LUSR ; une rafale PAR TABLE (chaque table a déjà sa transaction).
      Mesure avant rafale sous le lecteur (`Provider.Get`), sans écrivain. Sync : refus si
      `Coordinator.InFlightCount() > 0` à l'entrée de chaque rafale (TOCTOU bénin : le bail dblease
      sérialise de toute façon les écrivains). Fenêtre 04:00-05:00 UTC, tirage une fois par nuit.
      Seuil : 1,5 brutes/`_latest` par table (une table compactée vaut 1 ; A.3 : coût de page ≈
      proportionnel au rapport, `match_kill_events` à ~19 → pages ÷2 à ÷10 ; 1,5 borne le surcoût à
      +50 % entre deux nuits). Multi-titre : boucle sur `reg.All()` dont la base partagée existe,
      provider du titre, aucune comparaison de slug. Échec au milieu : la table en cours est
      annulée par sa transaction (base intacte), les précédentes restent compactées et vérifiées ;
      `refuserOrphelin` en tête de chaque table (un orphelin ne naît pas d'une transaction DuckDB) ;
      ERROR journalisée, la tâche s'arrête pour la nuit.
      POINT BLOQUANT : la rafale d'une table dure le temps de sa RECONSTRUCTION (trois lectures
      fenêtrées de la table brute + empreinte de vue + index), pas ≤ 2 s comme une rafale du LUSR.
      Pendant ce temps le provider est en RW : chaque lecture de page attend puis répond 503 au-delà
      de 3 s (`middleware.DefaultUserFacingReadBudget`), le sync attend, le chien de garde (2 s)
      journalise un WARN. Ordre de grandeur : 26,6 M lignes en ~33 s sur le poste (A.3), donc
      plusieurs secondes par grosse table en régime établi sur le VPS (2 vCPU), et des dizaines de
      secondes par table la nuit qui suit une recuisson. Décision à prendre par le user : (a) accepter
      cette indisponibilité nocturne (503 le temps de chaque table) ; (b) borner l'automatique aux
      tables dont la reconstruction tient dans le budget, WARN sinon et CLI manuel serveur arrêté ;
      (c) ne pas compacter dans le serveur : jauge seule (2b.3) + `compact-passes` ajouté à la
      séquence de recuisson.
- [ ] 2b.3 Jauge « passes par match » (rapport brutes / `_latest` par table) exposée dans le contrôle
      de santé des données (`scheduler/data_health_check.go`) et en expvar ; WARN au-delà du seuil.
- [ ] 2b.4 Documentation : ADR 0026 amendé (la compaction est automatique, conditions, filet),
      `docs/COMMANDS.md` FR + EN (le CLI reste pour l'usage manuel), changelog 8.0.1 FR + EN.
- [ ] 2b.5 Tests : déclenchement sous et au-dessus du seuil (horloge et seuil injectables), refus
      pendant un cycle de sync, écrivain partagé obligatoire (aucune ouverture directe), démo coupée,
      empreintes identiques après compaction en processus sur un corpus append-only réel
      (`-tags=integration`), `no_art_patterns_test.go` sans nouvelle entrée d'allowlist.
- Gate : doctrine §4 (dont `-tags=integration -p 1` sur `./internal/migration/...`,
  `./internal/scheduler/...`, `./cmd/server/...`) ; revue adversariale close.

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

- [~] 5.1 Si l'étape 2 mesure la vue match « ok » (rencontres déjà bornées par `6cb130a4f`) ET
      l'artefact de rejeu non dominant : `[~]` « mesuré, rien à faire ». Sinon, D-11 validée, puis :
- [~] 5.2 Inventaire des champs lus par `MatchScoreCurveChart`, `MatchScoreEventsChart`,
      `MatchImpactBadgesBar`, `MatchViewTabArsenal`, `MatchEquipmentUsageSection` (montée par
      l'Arsenal), y compris `coverage.score.modeSupported` ; consigné ici.
- [~] 5.3 Type dédié + endpoint de projection + stratégie de lecture serveur (calques écrits à la
      cuisson, ou cache par (match, date de l'artefact)) ; parité champ pour champ avec le document.
- [~] 5.4 Les composants lisent la projection ; la page de rejeu inchangée ; prévenir levelup-76
      avant de toucher au rejeu s'il est actif.
- Statut : étape `[~]` (2026-10-21). Banc : l'artefact de rejeu sort en 0,05-0,09 s, le temps de la vue match (1,2-3,5 s) vient de Q23b `match_encounter_stats` et de Q28c `mv_encounter_assists` (tout l'historique, journal du film), servis par le volet 2 (9.2, 9.4). D-11 rejetée par le user. Items 5.2-5.4 couverts par 9.2 et 9.4.

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
| Explorer : `page_size: 10000`, ligne de 38 champs (le user la juge « plutôt ok ») | audit | `[~]` mesuré 0,5-2,4 s, « plutôt ok » (user) |
| Relations : `relations[]` sans limite (25 affichées ; filtres et compteurs lus côté client) | audit | `[~]` 1,3-5,0 s en « tout l'historique », lenteur acceptée (user) ; sa lecture Q28 passe sur S1 (9.2) |
| Timeseries : `cumul_tab`, `intensity_tab`, `outcomes_over_time` calculés sans lecteur | audit | `[~]` mode « tout l'historique » peu utilisé (user) ; non prioritaire, registre des reports |
| Carrière `current_season`, session `compare_metrics` calculés sans lecteur | audit | `[~]` non mesuré isolément (sections absentes), coût faible attendu ; registre des reports |
| Bloc `coverage` du rejeu (~333 noms sur 500 sans lecteur ; badge admin, contrat, `parity_test.go`) | audit | `[~]` artefact servi en 0,05-0,09 s : aucun gain mesurable ; registre des reports |
| `GET /pages/sessions`, `POST /pages/match-history/query` sans appelant web | audit | `[~]` hors mesure (code mort à confirmer) ; registre des reports |
| `match_measured_kills` (Tactique, distances), `match_life_windows` (vies près d'un coéquipier) | inventaire | `[~]` 5,5-14,7 s mais seulement en « tout l'historique » de Timeseries ; registre des reports (second rang) |

## Annexe A — Mesures du banc

### A.1 « Avant » (2026-10-21, passage 3 à serveur neuf, code `affab87ed` + sections `9d40cc1b9`)

Copie des bases de PROD du 2026-10-21 09:57 ; médiane de 5 appels après chauffe, octets décodés.
`timeseries_tout`, `escouade_tout`, `relations` : périmètre « tout l'historique » (corps `{}`),
pas le réglage par défaut du web (dernière session) — pire cas. Stabilité : passages 2 et 3 à serveur
neuf à ±8 % (Relations Nuzzles 4,3 → 6,2 → 5,0 s, bimodale, consignée), mémoire privée du serveur de
banc ~1,56 Go après un passage.

| Page | JGtm | Nuzzles | Madina97294 |
|---|---|---|---|
| accueil | 2.77 s · 3.1 Mo | 6.20 s · 18.3 Mo | 3.08 s · 3.4 Mo |
| vue_match | 2.07 s · 0.1 Mo | 4.06 s · 0.2 Mo | 1.31 s · 0.1 Mo |
| rejeu_artefact | 0.05 s · 2.3 Mo | 0.08 s · 2.5 Mo | 0.06 s · 1.7 Mo |
| rejeu_contexte | 0.08 s · 0.1 Mo | 0.09 s · 0.1 Mo | 0.12 s · 0.1 Mo |
| explorer | 0.73 s · 1.0 Mo | 2.38 s · 5.7 Mo | 0.49 s · 1.1 Mo |
| relations | 1.81 s · 1.0 Mo | 5.04 s · 5.6 Mo | 1.32 s · 2.2 Mo |
| carriere | 0.04 s · 0.3 Mo | 0.10 s · 1.8 Mo | 0.04 s · 0.4 Mo |
| carriere_rencontres | 0.83 s · 0.0 Mo | 2.04 s · 0.0 Mo | 0.60 s · 0.0 Mo |
| carriere_rivaux | 0.78 s · 0.0 Mo | 1.93 s · 0.0 Mo | 0.53 s · 0.0 Mo |
| timeseries_tout | 11.35 s · 5.6 Mo | 27.57 s · 22.2 Mo | 7.80 s · 5.9 Mo |
| escouade_tout | 1.50 s · 5.1 Mo | 3.63 s · 11.8 Mo | 1.24 s · 4.6 Mo |
| comparaison | 6.25 s · 0.0 Mo | — | — |

### A.2 Ventilation (sections, passage 3)

| Lecture | JGtm | Nuzzles | Madina97294 | Famille |
|---|---|---|---|---|
| Accueil — `home_favorite_weapon` (arme favorite, tout l'historique) | 1,58 s | 1,98 s | 1,89 s | volet 2, S1 armes (9.3) |
| Accueil — `home_assisted_frags_recent` (Q26l, toutes les tuiles) | 1,12 s | 3,49 s | 0,87 s | volet 1 (fenêtre) + volet 2 S2 (9.4) |
| Accueil — `home_spartan_identity` | 0,12 s | 0,51 s | 0,34 s | — |
| Vue match — `match_encounter_stats` (Q23b, tout l'historique) | 2,00 s | 3,42 s | 1,11 s | volet 2, S1 paires (9.2) |
| Vue match — `mv_encounter_assists` (Q28c, tout l'historique) | 1,77 s | 3,91 s | 1,15 s | volet 2, S2 (9.4) |
| Comparaison — hors sections (frags entre joueurs non bornés) | 5,9 s | — | — | volet 2, S1 paires (9.2) |
| Timeseries tout — `weapon_range` + `range_profiles` + `lives` | 7,8 s | 14,7 s | 5,5 s | candidats `match_measured_kills` / `match_life_windows` (§6) |


### A.3 Après compactage des passes (DÉCOUVERTE, 2026-10-21, même copie, `levelup compact-passes --title halo_infinite`)

La copie de prod n'avait jamais été compactée : 26,6 M lignes brutes pour 3,7 M lues dans les
tables append-only du film (`match_kill_events` 7,97 M → 0,42 M, ~20 passes par match laissées par
les recuissons d'octobre). Compactage : ~33 s, sauvegarde préalable. Passage à serveur neuf :

| Page | JGtm | Nuzzles | Madina97294 |
|---|---|---|---|
| accueil | 2.77 → 0.31 s | 6.20 → 1.04 s | 3.08 → 0.51 s |
| vue_match | 2.07 → 0.92 s | 4.06 → 0.34 s | 1.31 → 1.15 s |
| explorer | 0.73 → 0.16 s | 2.38 → 1.25 s | 0.49 → 0.21 s |
| relations | 1.81 → 0.29 s | 5.04 → 0.70 s | 1.32 → 0.37 s |
| carriere_rencontres | 0.83 → 0.12 s | 2.04 → 0.19 s | 0.60 → 0.15 s |
| carriere_rivaux | 0.78 → 0.07 s | 1.93 → 0.13 s | 0.53 → 0.08 s |
| timeseries_tout | 11.35 → 1.66 s | 27.57 → 8.58 s | 7.80 → 2.11 s |
| escouade_tout | 1.50 → 0.44 s | 3.63 → 1.33 s | 1.24 → 0.48 s |
| comparaison | 6.25 → 0.83 s | — | — |

Après compactage, l'arme favorite coûte 60-150 ms (1,6-2,0 s avant), les frags assistés 55-205 ms,
Q23b/Q28c 250-1 080 ms (Madina97294 reste à ~1 s sur la vue match).

## Découvertes

- `analysis/identity.go` (`v_gamertag_lookup`, annuaire des noms) lit encore `match_kill_events_latest` pour résoudre les noms : hors du critère du volet 2 (lecture de noms, pas un agrégat), allowlistée en 9.5 ; à reprendre dans un chantier « annuaire » si mesuré coûteux.
- Banc (2026-10-21) : un serveur gardé entre deux passages ralentit de 6 à 15 % par passage, et sa mémoire privée atteint ~1,6 Go après UN passage de 34 pages (2 fils DuckDB, 512 Mo). Le VPS de prod a 1,8 Go de RAM au total : pression mémoire à surveiller (non traité ici).
- Comparaison : 5,9 s sur 6,2 passés HORS sections (frags entre joueurs non bornés, `QKillsBetweenPlayers`) ; seule la partie servie par S1 est dans le périmètre (9.2).
- (à compléter pendant l'exécution)
