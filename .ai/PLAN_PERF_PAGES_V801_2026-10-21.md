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
- **D-7 Familles du volet 2, grain et colonnes figés à l'étape 6 avant tout code** (fait : l'ADR
  0039 fait foi sur les colonnes ; écarts listés au gate de l'étape 6). Chaque famille
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
- [x] 2b.2 Compactage automatique HORS du serveur (décision du user du 2026-10-21, option (d) du
      pilote : pas dans le processus serveur — risque mémoire, ~1,6 Go sur le banc pour un VPS de
      1,8 Go, et pas de 503 en journée ; la conception « tâche de fond sous l'écrivain partagé »
      initiale est caduque). Fait : `scripts/compact-passes-nuit.sh` + unités
      `scripts/systemd/levelup-compact-passes.{service,timer}` (installation : `scripts/systemd/README.md`
      et `docs/RUNBOOK_DEPLOY_CHECKLIST.md`, les unités ne sont pas déployées par `deploy.sh`).
      **Note de conception (2026-10-21).** Décision sur la jauge 2b.3 (fichier
      `data/global/admin_state/passes_par_match.txt`, lu depuis l'hôte : aucun port, aucune route,
      rien d'ouvert à Internet ; `/debug/vars` exige un admin, `/health` ne porte que `{status}`).
      Seuil `migration.SeuilPassesParMatch` = 1,5 brutes/gardées, porté par le fichier (une seule
      source) : une table compactée vaut exactement 1 (dénominateur = règle du registre, pas la vue,
      sinon une vue plus filtrante qu'une passe laisserait une table compacte au-dessus de 1 et
      arrêterait le serveur chaque nuit) ; A.3 : coût de page ≈ proportionnel aux lignes brutes
      (`match_kill_events` à ~19 → pages ÷2 à ÷10), 1,5 borne le surcoût à +50 % entre deux nuits.
      Horaire 03:30 UTC et non 04:00 : la sauvegarde restic (`levelup-restic-backup.timer`) tourne à
      04:00 et arrête/redémarre le serveur elle aussi ; le ménage Docker de 03:00 supprime les
      conteneurs arrêtés depuis plus de 24 h. Pas de `Persistent` (un rattrapage au démarrage
      tomberait avec celui de restic) ; `TimeoutStartSec=25min` (passage mesuré le plus long ~90 s).
      Séquence : verrou `flock` (sinon 75), restic actif (sinon 75), jauge absente / illisible /
      > 48 h (2 cycles du contrôle de santé) → erreur sans rien toucher, 0 table → sortie, place
      disque ≥ taille des bases partagées + 2 Gio, `stop`, `compact-passes --backup-dir
      /app/data/compaction-backups` (hors `data/titles` : ni restic ni ménage n'y touchent une base),
      `start` toujours (trap si l'arrêt échoue), attente bornée de `/health` (120 s). Multi-titre :
      la CLI traite chaque titre dont la base existe ; le script compte la place par glob
      `data/titles/*/…`, aucun slug. Échec au milieu : la table en cours est annulée par sa
      transaction, les précédentes restent compactées et vérifiées, la CLI s'arrête à la 1re erreur
      (`refuserOrphelin` en tête de table) ; le script redémarre le serveur, garde les sauvegardes
      (état `echec`), sort en 1. Ménage `*.avant-compaction-*` > 24 h du seul dossier dédié, après un
      passage réussi seulement, joué en FIN de passage (sinon la sauvegarde de la veille, 23 h 59,
      survivrait 48 h).
- [x] 2b.3 Jauge « passes par match » : `migration.MesurerPasses` (`compaction_jauge.go`, lecture
      seule) ; contrôle de santé `scheduler/data_health_passes.go` : une lecture courte du provider
      B-swap PAR TABLE (`WithSharedReader`, câblé dans `cmd/server` par `healthSharedReader`), APRÈS
      l'audit du titre (le handle `OpenReadOnly` de l'audit est rendu) ; expvar
      `levelup.compaction.passes_par_match` (`<titre>/<table>`) et
      `levelup.compaction.tables_au_dessus_du_seuil` ; WARN par table au-delà du seuil ; fichier
      (`PathResolver.PassesGaugePath`, écriture atomique). Mesure partielle → rien publié. Démo : la
      tâche « santé données » est déjà `demoCut` dans `demoPolicies` (aucune tâche nouvelle).
- [x] 2b.4 Documentation : ADR 0026 amendé (EN, « Automatic compaction » : jauge, seuil, tâche hôte,
      pourquoi pas dans le serveur avec les durées prod du 2026-10-21, filet) ; en-têtes de
      `cmd/levelup/cmd_compact_passes.go` et `internal/migration/compaction.go` corrigés ;
      `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` ; changelog 8.0.1 EN + FR (section Ops) ; installation
      dans `scripts/systemd/README.md` et `docs/RUNBOOK_DEPLOY_CHECKLIST.md` (post-déploiement).
- [x] 2b.5 Tests : jauge (`migration/compaction_jauge_test.go` : 6/3 → 2, 1 après compaction, table
      absente, hors registre, seuil strict ; `scheduler/data_health_passes_test.go` : fichier + expvar,
      une lecture injectée par table, horloge injectée, mesure partielle → aucun fichier) ; script
      (`archlint/compact_passes_nuit_script_test.go`, faux `docker`/`curl`/`df`/`systemctl`/`flock`) :
      sous le seuil → rien, au-dessus → stop/run/start dans l'ordre, compaction en échec → serveur
      redémarré + code 1, `/health` muet → 1, arrêt en échec → redémarrage tenté, disque insuffisant,
      verrou, restic, jauge périmée ou absente, ménage (motif seul, après réussite seulement).
      `[~]` pour les cas de la conception caduque : refus pendant un sync et écrivain obligatoire →
      sans objet (serveur arrêté, la CLI prend le bail dblease) ; empreintes identiques après
      compaction → couvert par `internal/games/halo_infinite/migrations/compaction_e2e_test.go`
      (corpus partagé complet, inchangé). `no_art_patterns_test.go` : aucune entrée ajoutée.
- Gate : [x] `go test` et `-tags=integration -p 1` sur `./internal/migration/...`,
  `./internal/scheduler/...`, `./cmd/server/...`, `./internal/domain/title/...`,
  `./internal/archlint/...`, `./cmd/levelup/...` verts ; [x] `golangci-lint
  --new-from-merge-base=origin/main` 0 problème ; [x] `shellcheck` + `bash -n` du script ;
  [x] revue adversariale : ronde 1 (1 P1 restic, 2 P1 redémarrage et jauge partielle, tests faibles) corrigée en `a3ff5e1bb` ; ronde 2 sur les seules corrections (2026-10-21) : aucun défaut recevable, 4/4 corrections complètes. Étape 2b close (installation des unités systemd sur le VPS : au déploiement, avec l'accord du user).
- Revue adversariale ronde 1 (pilote, 2026-10-21 ; VPS vérifié en `Etc/UTC`, 03:30 UTC retenu),
  chaque correction avec un test rouge sans elle (vérifié en rejouant le test sur la version de
  `373e8b62b`) :
  - [x] P1 garde restic : l'unité restic est `oneshot` sans `RemainAfterExit`, `is-active` rend 3
        pendant la sauvegarde (`activating`) → lecture de `systemctl show -p ActiveState --value`
        (`activating`/`active`/`reloading`/`deactivating` = en cours) ; faux `systemctl` qui modélise
        `activating` (cas `activating` et `deactivating`).
  - [x] P1 redémarrage : `START_ESSAIS=3` essais de start + `/health` borné, `START_DELAI_S=15` entre
        deux, ERROR à chaque échec, sortie 1 si tous échouent ; le filet reste armé jusqu'à un
        redémarrage RÉUSSI (il ne rejoue pas des essais épuisés) ; tests : start raté puis réussi,
        start toujours raté (3 essais exactement), `/health` muet (3 essais).
  - [x] P1 jauge d'un titre illisible : la jauge est mesurée pour tout titre dont la base partagée
        existe, que l'audit du titre ait réussi ou non ; illisible → tables non mesurées → fichier
        non réécrit (WARN) ; test à deux titres dont un à base illisible.
  - [x] Tests qui mordent : place disque (base de 8 Mio, disque = marge + moitié de la base →
        refus ; mutant sans `bases_ko` rouge), âge de la jauge à 23 h 30, 47 h 50 et 48 h 10 (mutant
        72 h rouge), jauge illisible (4 formes), SIGTERM pendant la compaction → filet, serveur
        relancé, code 143 (`trap … TERM INT HUP` explicite ; mutant sans lui rouge).
  - [~] P2 verrou commun avec `scripts/deploy.sh` : consigné en Découvertes, non corrigé (consigne).

### Étape 3 — Accueil en fenêtre, serveur (volet 1)

- [x] 3.1 Extraction `buildTiles(ctx, tilesInput)` depuis `GetHomePage` (`service/home_service_tiles.go`) :
      traductions d'assets (`EnrichCanonicalAssetTranslations`), construction, placements de performance,
      rejeux, enrichissements par liste (une section par étape), analyse en cours. `tilesInput` porte
      l'historique complet (placements), la fenêtre, les favoris, la locale, le nom de liste et `translated`
      (la tête a déjà traduit TOUT l'historique : les sessions et faits marquants lisent aussi les libellés).
      `GetHomePage` : 64 lignes. Chargement déplacé dans `home_service_load.go`, page dans
      `home_service_page.go` (`home_service.go` passe de 536 à 202 lignes).
- [x] 3.2 `GetHomePage` sert la tête (30 tuiles) + `recent_matches_has_more` + `recent_matches_next_before` +
      `recent_matches_total` (= nombre de lignes à `MatchID` non vide, pas `CountPlayerMatches`). Ordre
      d'affichage = instant canonique décroissant puis `match_id` décroissant (`analysis/home_tiles_window.go`,
      une seule fonction pour la tête et les pages ; la tranche de l'appelant n'est jamais réordonnée).
- [x] 3.3 Endpoint Huma `GET /players/{player_slug}/pages/home/matches?kind=&before=&limit=` (limite 1-60,
      défaut 30 ; `kind=favorites` : favoris en entier) : `handlers/home_matches.go` (sans logique ; 400
      `invalid_cursor`), `HomeService.GetHomeMatches` ajouté au port et aux quatre faux
      (`health_home_test.go`, `home_test.go`, `og_inject_test.go`, `season_pass_service_test.go`). Même
      factory `HomeCtxWithAuth` que `/pages/home` (aucune nouvelle factory : le garde de parité
      `home_factories_parity_test.go` reste vert). `openapi.yaml` régénéré (`make openapi-gen`), fragment
      manuel pour le 200 opaque (comme `/pages/home`), `generated.ts` régénéré, plafond de `limit` inscrit
      dans `pagination_guard_test.go`. Cache : `private, no-cache` + ETag/304 (un favori basculé doit se voir).
- [x] 3.4 `buildFavoriteMatchListCanonical` supprimée (construisait toutes les tuiles puis filtrait) :
      remplacée par `analysis.FavoriteHomeTileRows` (lignes favorites seulement) + `buildTiles`.
- [x] 3.5 `last_session_outcomes` : tranche non nulle de `{match_id, outcome_tone, detail}` (`domain.HomeSessionOutcome`),
      même règle que `HomePage.tsx` l.204-213 (première tuile dont le libellé de session est celui de la
      session solo ou escouade résumée ; sinon les 20 plus récentes), calculée sur les lignes complètes,
      dans l'ordre d'affichage (le web inverse).
- [x] 3.6 Retrait de `recent_media` : `port.HomeRepository.LoadRecentMedia`, `noopHomeRepo`, mocks,
      `Q28RecentMedia`, `analysis.BuildRecentMedia`, `domain.RecentMediaItem`, `domain.HomeMediaRow`,
      `HomeRepo.LoadRecentMedia`, section `home_recent_media`, `lib/api/types.ts` (`RecentMediaItem`,
      `recent_media`), `test/handlers.ts`, fixtures de `HomePage.test.tsx`, `HomePage.nav.test.tsx`,
      `HomeRankingStates.test.tsx` (grep `recent_media|RecentMediaItem|LoadRecentMedia|BuildRecentMedia|
      HomeMediaRow|Q28RecentMedia` : 0 hors historique git). La rail de médias lit son propre endpoint.
- [x] 3.7 Retrait de `HomeMatchesCache` (cache qui n'était plus lu depuis P4.3) : `home_matches_cache.go` +
      test, `WithMatchesCache` -> `WithXUID` (le xuid de la page reste le sujet de l'identité Spartan et des
      coéquipiers de session), `wire/registry.go`, `registry_auth.go`, `registry_pages_home.go`,
      `post_sync_deltas.go` (+ `invalidateHomeCacheAfterSync`), `post_sync_home_cache_test.go`, les 2 tests
      `TestHomeService_GetHomePage_Cache*` (déjà skippés). `home_factories_parity_test.go` : aucune référence
      au cache, inchangé. Baseline de tests : 17 paires retirées (78 lignes), nommées dans
      `scripts/check_test_baseline.sh` (RETRAIT DU 2026-10-21).
- [x] 3.8 ADR 0036 : invariant I8 + ligne de garde-rails (« Eight invariants ») ; garde-rail
      `TestGetHomePage_FenetreBornee` (200 matchs -> 30 tuiles, totaux inchangés) joué sous `halo_infinite`
      ET `halo_5` (aucune comparaison de slug).
- [x] 3.9 Parité : `TestGetHomeMatches_ParitePagesEtListeComplete` (200 matchs, tailles de page 0/1/17/30/
      60/1000 : tête + pages = liste construite d'un bloc, `reflect.DeepEqual` tuile par tuile, même ordre
      que le chargeur) ; `TestGetHomeMatches_SynchroEntreDeuxPages` (3 matchs arrivés en tête entre deux
      pages : ni doublon ni trou) ; `analysis/home_tiles_window_test.go` (instants égaux, bornes, curseur).
      Parité sur la copie des bases du banc : voir l'étape 4.4 (test Q25 / ordre de l'Accueil, trois comptes).
- Décision d'exécution (étape 3, 2026-10-21) : la tête a gardé `favorite_matches` jusqu'à l'étape 4 (le web
  n'avait pas encore sa requête dédiée) ; retiré en 4.6.
- Gate : [x] `go test` défaut puis `-tags=integration -p 1` sur service, analysis, domain, port, api/...,
  platform/duckdb, archlint ; [x] `golangci-lint run --new-from-merge-base=origin/main` 0 problème ;
  [x] `tsc -b` + `vitest run src/features/home src/lib/api src/lib/analysis-pending src/test` ;
  [x] `bash scripts/check_test_baseline.sh tests` ; [x] banc avant/après (pilote, 2026-10-21, copie compactée) : tête de l'Accueil 94-101 Ko (3,1-18,3 Mo avant), durée 0,14 / 0,39 / 0,25 s pour JGtm / Nuzzles / Madina97294 (0,31 / 1,04 / 0,51 s avant), `home_assisted_frags_recent` 6-16 ms ;
  [x] revue adversariale : voir le bloc « Revue adversariale » de l'étape 4 (rondes 1 et 2).

### Étape 4 — Accueil en fenêtre, web (volet 1)

- [x] 4.1 La requête de tête garde `useQuery` sous `queryKeys.home` (prefetch, loader de route et
      invalidations existantes intacts) ; pages suivantes sous `queryKeys.homeMatchesPage` (une clé PAR PAGE,
      le curseur en dernier segment : une tête qui change de curseur après une synchro repart de zéro, les
      anciennes pages ne se recollent pas) et favoris sous `queryKeys.homeFavorites`, toutes sous le préfixe
      `home` (toute invalidation de l'accueil les recharge). Garde `keys.title-slug.guard.test.ts` mis à jour.
- [x] 4.2 `components/ui/carousel.tsx` : `onNearEnd` (appelé à 5 tuiles de la fin, `carousel.logic.ts`
      `isNearEnd`/`itemSpanOf`), `loadingMore` (indicateur `role="status"`, chaîne FR/EN
      `common.carousel.loading_more`) ; il ne monte que les enfants reçus. Erreur de page :
      `home.matches.load_error` / `load_retry` (FR + EN) dans la section.
- [x] 4.3 Section des matchs extraite dans `features/home/HomeMatchesSection.tsx` ; `HomePage.tsx` 572 -> 458
      lignes, exemption `max-lines` de la ligne 1 retirée ; `SectionTitle` (garde `detail-section`, plafond
      de `HomePage.tsx` abaissé de 5 à 4).
- [x] 4.4 Navigation (D-9) : `goToMatch` sans `matchIds` pour `home_recent` (contexte
      `{ source, matchIds: [], contextDescriptor: { kind: 'recent' } }`, conservé en sessionStorage :
      `hasNavContent`) ; `useMatchNeighborsResolved` : contexte sans liste = voisins et total de Q25 (tout
      l'historique), libellé du descripteur, contexte propagé à prev/next. Favoris : liste d'ids conservée
      (bornée). Q25 lit le lecteur partagé LIVE (`neighborsRead`), plus le snapshot immuable (revue ronde 1, P1 : sur la copie de prod, les 253 matchs les plus récents de Nuzzles étaient absents du snapshot : voisins vides et total 6 989 au lieu de 7 242 ; `TestMatchNeighbors_LisentLeLiveJamaisLeSnapshot`). Parité Q25 / ordre de l'Accueil : `TestHomeTiles_OrdreIdentiqueAQ25` (schéma DuckDB réel,
      ex aequo compris) a REVÉLÉ que Q25 n'avait pas de départage -> `ORDER BY instant DESC, match_id DESC`
      dans `Q25NeighborMatches` et `Q25NeighborMatchesTemplate` (le test échoue sans). [!] La parité « trois
      comptes sur la copie des bases du banc » n'est pas exécutable ici : la copie vit dans le worktree
      `LevelUp-wt-chronos`, `data/` de ce worktree n'a que `metadata-prebuilt.zip` (consigne : ne toucher
      aucun autre dossier) ; FAIT par le pilote (2026-10-21) après la correction P1-1 : parcours complet par curseur des trois comptes (JGtm 1 182, Nuzzles 7 242, Madina97294 1 292 tuiles, 0 doublon, total juste) et voisins + total de Q25 conformes sur 10 positions par compte (récents, bords de page, milieu, plus ancien) : 0 écart. Avant la correction : Nuzzles 10 écarts sur 10 (snapshot).
- [x] 4.5 Analyse en cours : `tilesPageRefetchInterval` / `tilesPageStaleTime` (60 s tant que CETTE page
      porte une tuile en cours) sur chaque page de tuiles et sur les favoris ; `homeRefetchInterval` /
      `homeStaleTime` ne lisent plus que la tête. Test : deux pages, 60 s seulement pour celle qui porte
      une analyse (`queries.tiles.test.tsx`).
- [x] 4.6 Favori : `useSetMatchFavorite` pose le drapeau sur place dans les réponses déjà en cache (tête,
      pages : `patchFavoriteFlag`, même référence si rien ne change) et ne recharge QUE la liste des
      favoris (+ l'historique Explorer). Retrait de `favorite_matches` de la réponse de la tête (Go :
      `HomePageResponse`, `GetHomePage`) : les favoris ont leur propre lecture (décision D-8, fin de la
      décision d'exécution de l'étape 3).
- [x] 4.7 `OutcomeSequenceTape` lit `last_session_outcomes` (ordre inversé pour la bande, libellé de tuile
      en infobulle) ; plus de calcul dans `HomePage.tsx`.
- [x] 4.8 Tests vitest : `homeTiles.logic.test.ts`, `carousel.logic.test.ts`, `carousel.test.tsx`,
      `queries.tiles.test.tsx` (page suivante, idempotence, reset de tête, erreur/réessai, rechargement par
      page, favoris), `HomeMatchesSection.test.tsx` (montage limité à 30, page à 5 tuiles de la fin,
      navigation sans liste, favoris, favori sans rechargement de la tête), `HomePage.lastSession.test.tsx`,
      `useMatchNeighborsResolved.test.tsx` / `navContext.test.ts` (contexte sans liste). Miroir du contrat :
      `types.ts` (`recent_matches_total`, `_has_more`, `_next_before`, `last_session_outcomes`,
      `HomeMatchesPage`, `HomeSessionOutcome`, `favorite_matches` retiré), `test/handlers.ts` (+ handler
      `pages/home/matches`). [!] E2E local : serveur d'API de démo et Vite lancés sur 8020 / 5180 (le serveur
      du port 8000 n'a pas été touché), `npx playwright test --project=chromium` : 38 réussis, 62 sautés
      (fixtures de démo absentes : `slice-5-home` saute), 7 échecs d'environnement SANS rapport avec
      l'Accueil (`legacy-redirect` H5, `slice-4b` 404, `slice-4c`, 4 de `slice-9-onboarding` : joueur de
      démo / état d'installation absents du bac isolé). À rejouer sur un environnement de démo garni.
- Gate : [x] `go test` défaut puis `-tags=integration -p 1` (service, analysis, domain, api/..., archlint,
  platform/duckdb) ; [x] `golangci-lint --new-from-merge-base=origin/main` 0 problème ; [x] `tsc -b`,
  `eslint .` (0 erreur, 26 avertissements existants), `vitest run` complet (987 fichiers, 10 226 tests),
  `lint:colors`, `lint:fields`, `knip` ; [x] banc : critères Accueil du §2 tenus (voir étape 3) ;
  [x] revue adversariale : ronde 1 (6 P1) corrigée en `f128a0141` ; ronde 2 (pilote, 2026-10-21) : 5/6 corrections complètes, 1 P1 proposé RECLASSÉ P2 par le pilote (la page de la vue match rend déjà `null` pendant son propre chargement, comportement antérieur au lot ; Q25 en live mesuré à 7-10 ms contre 300-800 ms pour la vue : les voisins arrivent avant la page), 2 P2 → Découvertes. P0+P1 : 6 → 0.

- Revue adversariale ronde 1 (pilote, 2026-10-21) : P1-1 Q25 en lecture live [x] ; P1-2 favoris chargement / erreur [x] ; P1-3 voisins gardés pendant le pas [x] ; P1-4 commentaires [x] ; P1-5 `HomeMatchesKindRecent` utilisé [x] ; P1-6 tests qui mordent (fetchMore, Carousel + hook, `useNavigateToMatch`) [x] ; P2 défilement ramené à 30 tuiles : consigné en Découvertes. Ronde 2 : par le pilote.

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

- [x] 6.1 ADR 0039 (EN) : règle du paradigme (agrégat décomposable par match → résumé écrit avec sa
      source ; lignes brutes → lecture à la demande) ; D-6 (point d'écriture unique, normalisations,
      `now` partagé, couplage de disponibilité) ; D-7 ; tête qui décide la passe ; `summary_pass` /
      `summary_rev` ; recalcul (D-10) ; jauge « sans tête » ; garde-rail anti-retour ; compaction ;
      snapshots et démo ; multi-titre (Halo Infinite et Halo 5 écrivent `match_kill_events` : migration
      générique, aucun branchement par slug).
- [x] 6.2 Colonnes exactes de S0, S1, S2 figées dans l'ADR, chacune justifiée par les lecteurs qui la
      lisent (table « lecteur → colonnes » reprise de l'inventaire et RE-VÉRIFIÉE sur pièces, dont les
      cas : bots sans xuid, victime bot dans `frags_film` de Q26l, tueur NULL dans
      `/monitoring/weapon-coverage`, `stolen` et portée de Q21d, « mesurée sans assistant » et victime
      du cas bot-bot de la Coordination, jeton Campagne sur `match_id`).
- [x] 6.3 Mesures sur la copie des bases, consignées : lignes de S1 et S2 projetées par rapport au
      journal (décision « S1 unique ou scindée ») ; hétérogénéité de `publishable` dans une même passe
      (`SELECT decode_pass FROM match_kill_events GROUP BY decode_pass HAVING COUNT(DISTINCT publishable) > 1`).
- Gate : ADR relu par un contexte frais ; aucune contradiction avec les ADR 0019, 0026, 0030, 0036 ;
  chaque lecteur de 9.1-9.4 a ses colonnes dans 6.2.
- Statut du gate : [x] (2026-10-21). Relecture à contexte frais faite ; corrections appliquées :
  écrivains réels (dont le seed démo synthétique), fonction d'insertion unique
  `insertKillSummaryRows`, jauge « sans tête » = porte de release, I2 appliqué aux vues S, périmètre
  du garde-rail calqué sur `TestNoRawAppendOnlyReads`, empreinte de révision (modèle ADR 0034 D-6),
  section Rollout, procédure d'ajout d'un lecteur ou d'un agrégat, liens avec 0026/0030/0036.
  Mesures 6.3 (copie de prod, 2026-10-21) : 0 passe à `publishable` hétérogène (73 729 passes
  BRUTES, avant compactage) ; 0 assistant nommé sans `assist_known` ; 0 part sans gamertag
  d'assistant ; S1 245 547 lignes et S2 86 901 pour 419 858 lignes `_latest` (Halo Infinite), S1
  96 178 pour 268 337 (Halo 5) → S1 unique, pas de scission.
  ÉCARTS À D-7, assumés par l'ADR (qui fait foi) : S1 sans gamertags (aucun lecteur agrégat ne les
  lit) et sans `kills_publishable` (`publishable` constant par passe, porté par la tête) ; S0 =
  `publishable` + 4 comptes ; S2 sans ligne séparée « mesurée sans assistant » (grain à assistant
  NULL) ; `qSoloFrags` (lignes individuelles) rejoint l'allowlist de 9.5 ; les lectures du journal
  côté `sync/` sont hors décision et hors garde-rail.

### Étape 7 — Familles S0 à S2 : schéma, calcul, écriture (volet 2)

- [x] 7.1 Migration générique (ordre canonique `migration/order.go`) : tables append-only S0-S2,
      séquences `id` propres, vues `_latest` jointes à la tête sur `summary_pass` (recette ADR 0026,
      `append_only_rebuild.go`), vues recréées après tout `ADD COLUMN` ; tables et vues ajoutées à
      `EnsureMatchKillEvents` (sinon `insertKillEventRows` échoue sur une base ou une fixture non migrée).
      Fait (2026-10-21) : `shared_match_kill_summaries_v1` (`migration/steps_shared_kill_events_summaries.go`),
      rangée juste après `shared_registry_killsource_sans_killfeed_rev_v1`, donc après tous les
      `steps_shared_kill_events_*` ; colonnes de l'ADR 0039 à la lettre ; séquence et index
      (`match_id`, `written_at`) par table ; `written_at NOT NULL` SANS défaut SQL (horodatage Go de la
      passe, D-6). Tête = dernière génération (`written_at DESC, id DESC`), filles jointes à la tête sur
      (`match_id`, `summary_pass`) avec son `publishable`. Règle « vues recréées après `ADD COLUMN` »
      écrite dans l'en-tête de la migration. `EnsureMatchKillEvents` pose les trois tables et leurs vues.
      `append_only_rebuild.go` non utilisé : tables nettes-neuves (patron `steps_shared_usage_summary.go`).
- [x] 7.2 Compaction : S0-S2 dans `migration/compaction_registry.go` (`dernierePasse` pour la tête,
      passe de la vue parente pour les filles, modèle `match_usage_players`) + signatures de vue dans
      `compaction_test.go`.
      Fait : `TestRegistre_SignaturesDesResumesSurLaBaseMigree` relit le SQL des vues tel que DuckDB le
      rend après migration (et refuse une règle modifiée). Le bout-à-bout
      `games/halo_infinite/migrations/compaction_e2e_test.go` sème et compacte aussi les trois tables
      (tête : une ligne par passe ; le semeur ne pose un xuid que sur une colonne VARCHAR, car
      `n_killer_xuid` est un compte).
- [x] 7.3 Garde-rails ART : S0-S2 dans `tablesProtegees` de `sync/no_art_patterns_test.go` et de
      `sync/append_only_state_guard_test.go` ; aucune nouvelle entrée d'allowlist.
      Fait, et en plus (même famille, enrôlement à la création, ADR 0030 D-4) : motif de
      `TestNoRawAppendOnlyReads` (`platform/duckdb/no_raw_rating_reads_test.go`).
- [x] 7.4 Fonction pure dans `internal/analysis/` (type de ligne neutre → S0, S1, S2) et ses deux
      adaptateurs (lignes en mémoire, lignes lues en base) ; écriture dans `insertKillEventRows` (D-6).
      Fait : `analysis/killsummary` (`Summarize(publishable, []Row)`, `Rev = "ks1"`, tranches par
      `domain.AssistTierLowMaxPct` / `AssistTierMidMaxPct`, texte vide = NULL, SourceTag 0 = NULL) ;
      `persist/kill_summaries.go` : adaptateur mémoire `killSummaryRowsOf` et insertion UNIQUE
      `insertKillSummaryRows`, appelée par `insertKillEventRows` dans la même transaction avec le même
      `now` ; `persist/kill_summaries_relecture.go` : adaptateur base `readLatestKillPass` (vue
      `match_kill_events_latest`) et `PersistLatestKillPassSummaries` (propre transaction ; tête datée
      du `written_at` de la passe relue, pour que l'ordre des têtes suive celui des passes — choix non
      écrit dans l'ADR, consigné ici). Appelants de cette dernière : le seed démo synthétique
      (`ops/seed_demo_synthetic_shared.go`), les tests ; le rattrapage (étape 8) s'y branchera.
- [~] 7.5 Snapshots et démo : S0-S2 dans `ops/snapshot_export.go` (`sharedSnapshotMatchKeyedRaw`) et
      dans la lecture de snapshot ; seed démo (`ops/seed_demo_shared_tables.go`, `seed_demo.go`) avec
      anonymisation des colonnes d'identité et `TestSharedTablesDeclareEveryIdentityColumn` à jour ;
      fixtures de tests qui écrivent `match_kill_events` (`sync/testutil/fixture.go` et les tests
      `INSERT INTO match_kill_events`) passent par l'écriture des résumés ou les posent.
      Fait : snapshot (tables REQUISES ; vues posées par `EnsureMatchKillEvents` ; snapshots antérieurs
      refusés jusqu'au prochain cut, comme l'annonce le Rollout de l'ADR) ; seed démo : liste
      `killSummaryTables`, identités déclarées (`colFeedKillerXUID` centralisé, garde-rail du littéral
      étendu), exemption datée `match_kill_heads.n_killer_xuid` (compte INTEGER, pas une identité) avec
      contrôle de péremption ; `TestCopyAnonymizedTables_ResumesDuJournal` (déclaration réelle).
      `sync/testutil/fixture.go` n'écrit pas `match_kill_events` (vérifié).
      Fixtures `INSERT INTO match_kill_events` brutes : couvert par 9.x (décision du pilote Q2,
      2026-10-21) — chaque fixture est convertie avec le lecteur qui l'utilise.
      Revue du 2026-10-21 : le contrat de schéma de la lecture de snapshot cite S0-S2 et leurs vues, et
      `TestOpenSnapshotShared_ResumesDuJournalVoyagent_integration` vérifie qu'ils voyagent (prêt
      seulement) ; `TestSeedDemoSynthetic_ResumesCoherents` tient la tête et S1 de chaque match semé
      (échoue sans l'appel du seed) ; S0-S2 sont dans `detSeededTables`, `summary_pass` (tiré au hasard)
      exclu du dump explicitement.
- [x] 7.6 Migrations historiques qui ont écrit `match_kill_events` (`steps_shared_kill_events_credit_base.go`,
      `_from_pairs.go`, `_sans_killfeed.go`) : non rejouées, nommées dans l'ADR ; le rattrapage (étape 8)
      couvre leurs passes.
      Vérifié sur pièces : seules `_credit_base.go` et `_from_pairs.go` écrivent le journal (nommées par
      l'ADR 0039, décision 2) ; `_sans_killfeed.go` n'écrit que `match_registry` (écart au texte de
      l'item, l'ADR fait foi). Aucune n'est rejouée ; leurs passes restent sans tête jusqu'à l'étape 8.
- [x] 7.7 Tests : purs (bots, assistant anonyme, cas bot-bot, mesurée sans assistant, NULL, passe crédit
      puis passe film) ; parité adaptateur mémoire = adaptateur base ; un test d'intégration par
      écrivain (sync primaire, complétion, collecteur killsource, `BatchBuilder`, backfill CLI) : tête
      S0 `src_decode_pass` = passe de `match_kill_events_latest` ; deux passes successives → vues
      `_latest` = dernière passe ; même passe relue à révision nouvelle → aucun double compte ;
      injection d'erreur dans le calcul → passe annulée ; Halo 5 ; test des seuils de tranche liés à
      `summary_rev` ; test de cohérence PERMANENT : somme des `kills` de S1 = nombre de lignes de la
      passe courante.
      Fait : purs `analysis/killsummary/killsummary_test.go` ; empreinte `rev_test.go` (jetons des
      sources du paquet + bornes de tranche, couple figé `ks1`) et `TestRev_BornesDeTrancheLieesALaRevision` ;
      `persist/kill_summaries_integration_test.go` (sync primaire Halo Infinite ET Halo 5, complétion,
      BatchBuilder, deux passes, relecture à révision nouvelle puis rattrapage depuis la base, parité
      des adaptateurs relue jusque dans les vues, cohérence permanente sur un corpus de 5 matchs) ;
      `sync/killcollector/kill_summaries_integration_test.go` (collecteur crédit ; le test du film réel,
      sauté sans `KILLSOURCE_FIXTURES`, n'est pas touché : plafond de taille figé) ;
      `cmd/levelup/cmd_backfill_killsource_resumes_integration_test.go` (passe crédit du backfill).
      Constats partagés dans `sync/testutil/kill_summaries.go`. Injection d'erreur : la fonction pure
      est totale (ADR décision 2) ; l'erreur injectée est une erreur BASE sur l'insertion de la tête
      (table retirée), après le journal dans la même transaction : passe annulée pour chaque écrivain
      (le sync primaire perd le match, la complétion ne pose pas `events_loaded`, le backfill compte
      l'erreur). Coût de la passe (`TestPasseCredit_OuVaLeTemps`, CollectMatch de 3 morts, base
      fichier) : sans résumés 9,8 / 13,7 / 11,1 ms (+ une valeur aberrante à 56,5), avec 11,6 / 14,7 /
      12,3 ms : dans le bruit.
- [x] 7.8 Plans d'exécution : `EXPLAIN ANALYZE` des vues S0-S2 sous une liste de matchs (le filtre
      descend-il sous la tête et ses filles ?) consigné ; les tests ADR 0036 qui exigent des fenêtres
      bornées (`exigerFenetresBornees`) restent verts ou sont réécrits pour les vues S, et le tableau
      d'exceptions I2 de l'ADR 0036 mis à jour.
      Consigné (`platform/duckdb/resumes_fenetres_test.go`, 10 matchs x 2 générations = 20 têtes, liste
      `VARCHAR[]` de 2 matchs) : S0 sous la liste → fenêtre 4 lignes (bornée) ; S1/S2 pour UN match
      (`match_id = ?`) → 2 (l'égalité traverse la jointure) ; S1/S2 sous la liste → 20 : la liste NE
      descend PAS sous la fenêtre de la tête (toute la table des têtes, jamais les lignes des filles).
      Variantes mesurées : `IN (?, ?)` et `IN` littéral → 20 ; liste sur la vue fille + semi-jointure sur
      la vue tête → 20 et 4 (la fenêtre interne reste entière) ; table fille BRUTE jointe à la vue tête
      sous la liste → 4 (bornée, mais lecture brute d'une table append-only, interdite par ADR 0026 et
      `TestNoRawAppendOnlyReads`). Les tests `exigerFenetresBornees` existants restent verts (aucun
      lecteur ne lit encore S).
      Décision Q1 du pilote (2026-10-21), option (a) : la fenêtre de la tête n'est pas bornée sous une
      liste quand on lit une vue fille ; acceptée (une ligne par match et génération, ~6 500 lignes
      compactées en prod). ADR 0039 décision 7 amendée, ligne ajoutée au tableau d'exceptions I2 de l'ADR
      0036 ; le test `resumes_fenetres_test.go` tient la taille de cette fenêtre.
- Gate : doctrine §4 (dont `-tags=integration` persist/sync) ; revue adversariale close.
  Statut (2026-10-21) : `go test` (défaut) vert sur analysis/killsummary, persist, migration, ops,
  sync/..., platform/duckdb/..., migrations Halo Infinite, cmd/levelup ; `-tags=integration -p 1` vert
  sur les mêmes paquets ; `go test ./internal/archlint/` vert ; golangci-lint
  `--new-from-merge-base=origin/main` 0 problème, avec et sans tag `integration` ;
  `scripts/check_test_baseline.sh` (mode autonome, suite d'intégration complète du module, 360
  paquets, 21 416 tests) : tous les tests de la baseline présents, aucun test ni paquet en échec ;
  contrôle de COUVERTURE rouge (total 65,8 % contre 73,0 % de baseline). Écart de 7 points sur le
  module entier, sans rapport mesurable avec ~600 lignes testées de ce lot ; NON vérifié sur le
  commit de base (35 min de suite) — à confirmer par le pilote ou la CI (la baseline de couverture est
  un gate Linux). Revue adversariale : par le pilote.
  Clôture (2026-10-21, après les décisions Q1 et Q2 du pilote et les 10 corrections de revue) : chaque
  test ajouté a été vu ÉCHOUER sous la mutation qu'il garde (null écrits en sentinelle, comptes de
  la tête permutés ou faux, appui volé à égalité, erreur de S1 ou de S2 avalée, relecture de la
  table au lieu de la vue, appel du seed démo retiré, tables retirées de l'export du snapshot),
  puis la mutation annulée. Gate rejoué sur les paquets touchés : `go test` (défaut) vert sur
  analysis/killsummary, persist, ops, migration, migrations Halo Infinite, cmd/levelup, sync/...,
  platform/duckdb/..., archlint ; `-tags=integration -p 1` vert sur les mêmes ; golangci-lint
  `--new-from-merge-base=origin/main` 0 problème, avec et sans tag `integration`. Le contrôle de
  couverture global de `check_test_baseline.sh` reste jugé par la CI. Étape CLOSE.

### Étape 8 — Rattrapage du parc (volet 2, D-10 validée avant)

- [x] 8.1 Commande `levelup backfill-match-summaries --title <slug>` : pour chaque match dont la passe
      courante n'a pas de tête à la révision courante (anti-join), relit la passe en base (adaptateur
      base), calcule S0-S2, écrit par lots bornés via `persist` ; serveur arrêté exigé (ouverture en
      écriture), message clair sinon. Ne décode aucun film.
      Fait (2026-10-21) : `cmd/levelup/cmd_backfill_match_summaries.go` (`--title`, `--dry-run`, `--batch`, lot par
      défaut `defaultSummariesBatch` = 250). La liste des matchs à rattraper est UNE anti-jointure
      (`persist.PendingKillSummaryMatches`, `persist/kill_summaries_catchup.go`) : aucune tête, tête à
      une autre révision, OU tête qui résume une autre passe que la courante (ajout à la lettre du plan,
      consigné dans l'ADR 0039 décision 5) ; calculée une fois (fenêtre de la vue du journal), parcourue
      par lots avec un journal INFO de progression, jauge finale relue. Écriture par
      `PersistLatestKillPassSummaries` (une transaction par match) ; un échec se compte, se journalise
      et la commande se termine en erreur. Refus clair si la base est tenue (`erreurOuvertureEcriturePartagee`).
      Ouverture par `duckdb.OpenReadWrite` (aucune entrée d'allowlist). Tests d'intégration
      (`cmd_backfill_match_summaries_integration_test.go`) : sans tête / ancienne révision / passe
      supplantée réécrits, à jour intact, second passage vide, `--dry-run` n'écrit rien, échec compté.
- [x] 8.2 Jauge « matchs de la passe courante sans tête » : au boot et dans
      `scheduler/data_health_check.go` (lit déjà le journal), journalisée en WARN si non nulle.
      Fait (2026-10-21) : `scheduler/data_health_kill_summaries.go` (même texte SQL que 8.1 via
      `persist.CountPendingKillSummaries`), appelée par `auditTitle` ; expvar `levelup.kill_summaries_pending`
      (une entrée par titre, jamais écrasée par une mesure en échec), WARN par titre quand non nulle,
      champ `KillSummariesPending` du résultat, hors `WarningsTotal` (signal distinct, comme les trous LUSR ;
      aucun changement du DTO admin). Pas au boot : le contrôle périodique seul (coût de la fenêtre du
      journal). Pas de jauge des passes dans ce worktree : rien à côté de quoi la poser. Test :
      `data_health_kill_summaries_e2e_test.go` (0 sur base complète ; 3 pour tête absente, ancienne
      révision, passe supplantée ; 0 après rattrapage).
- [ ] 8.3 Passage complet sur la COPIE des bases du banc, Halo Infinite et Halo 5 ; volume (matchs,
      lignes S par match), durée et pic mémoire consignés et ANNONCÉS au user avant 8.4.
- [ ] 8.4 Passage sur les bases locales réelles (`LevelUp/data/`), serveur local arrêté avec l'accord
      du user et des sessions actives prévenues ; jauge à 0 pour les deux titres.
- Gate : doctrine §4 ; jauge à 0 sur la copie et sur les bases locales ; volume et pic mémoire consignés.
  Statut (2026-10-21) : 8.1 et 8.2 faits (code, tests, docs COMMANDS EN/FR, changelog 8.0.1 EN/FR, ADR 0039
  décisions 5-6). Gate de code rejoué sur persist, scheduler, cmd/levelup : `go test` (défaut) vert ;
  `-tags=integration -p 1` vert ; `go test ./internal/archlint/` vert ; gardes ART de sync et de
  platform/duckdb verts ; golangci-lint `--new-from-merge-base=origin/main` 0 problème avec et sans tag
  `integration` ; chaque test vu échouer sous sa mutation (condition de révision ou de passe retirée de
  l'anti-jointure, `--dry-run` qui écrit, échec non compté). RESTENT OUVERTS, réservés au pilote : 8.3
  (copie des bases du banc) et 8.4 (bases locales réelles) ; le gate de l'étape (jauge à 0, volume, pic
  mémoire) ne peut se fermer qu'avec eux. Étape NON close.

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

- Revue ronde 2 piste A (P2, non traités) : (1) `MatchViewPage.tsx:71` rend `null` pendant le chargement de la vue, donc la barre de navigation est absente à chaque pas vers un match non en cache (comportement antérieur au lot ; les voisins, à 7-10 ms, arrivent avant la vue) ; (2) `MatchHeader.tsx:146` `pendingStep` n'est remis à zéro ni au changement de match ni sur erreur des voisins : un pas retenu peut être joué sur un autre match si l'utilisateur change de match avant la réponse (rare) ; (3) `match_view_repo_neighbors_skill.go:27,85` : Q25 lit le live pour tous ses appelants (vue match ouverte hors Accueil comprise) et attend le provider pendant un écrit, borné à 10 s (500 `neighbors_error` au-delà), comme la liste de l'Accueil.
- `analysis/identity.go` (`v_gamertag_lookup`, annuaire des noms) lit encore `match_kill_events_latest` pour résoudre les noms : hors du critère du volet 2 (lecture de noms, pas un agrégat), allowlistée en 9.5 ; à reprendre dans un chantier « annuaire » si mesuré coûteux.
- Banc (2026-10-21) : un serveur gardé entre deux passages ralentit de 6 à 15 % par passage, et sa mémoire privée atteint ~1,6 Go après UN passage de 34 pages (2 fils DuckDB, 512 Mo). Le VPS de prod a 1,8 Go de RAM au total : pression mémoire à surveiller (non traité ici).
- Comparaison : 5,9 s sur 6,2 passés HORS sections (frags entre joueurs non bornés, `QKillsBetweenPlayers`) ; seule la partie servie par S1 est dans le périmètre (9.2).
- Étape 7 (2026-10-21) : le modèle d'empreinte cité par l'ADR 0039 (`decoder_rev_fingerprint_test.go`) n'existe plus (retiré le 2026-09-16 au profit de `film/revision`, propre aux couches du décodeur et à leur fermeture d'imports). L'empreinte de `killsummary.Rev` hache le flux de jetons (`go/scanner`, sans commentaires) des sources du paquet et les bornes de tranche ; elle ne réunit pas les trois marqueurs du ratchet `archlint/no_ad_hoc_source_fingerprint_test.go`. À trancher par le pilote si une centralisation est voulue.
- Étape 7 : deux affirmations fausses corrigées dans les commentaires touchés (règle 17) : `insertKillEventRows` n'était pas « le seul écrivain » du journal (seed démo synthétique et deux migrations historiques l'écrivent en SQL) ; la liste des appelants de `EnsureMatchKillEvents` omettait la lecture du snapshot partagé.
- Étape 7 : le test du collecteur sur film réel (`TestKillSourceCollecteFilmReelEtRelitParLaVue`) se saute sans `KILLSOURCE_FIXTURES` et son fichier est au plafond figé par `archlint/film_file_size_test.go` : aucune assertion de tête n'y est ajoutée ; le chemin `PersistPass` du collecteur est couvert sans film par le collecteur crédit.
- Étape 7 (revue du 2026-10-21) : le test d'empreinte de `killsummary.Rev` (`rev_test.go`) refait un hachage de jetons (copie du mécanisme de `film/revision/jetons.go`) ; à reprendre si un mécanisme partagé hors film est créé.
- Piste C (2b) — contrôle de santé des données : `scheduler/data_health_check.go` (`openDBShared`) ouvre la base partagée par `duckdb.OpenReadOnly`, pas par le provider ni `OpenReadForQuery` (règle CLAUDE.md n° 4). D'après le contrat de `sharedprovider.Provider`, une référence tenue hors du provider fait refuser l'`OpenReadWrite` d'un swap tant qu'elle n'est pas rendue : pendant chaque audit de titre (toutes les 24 h), une écriture du sync peut échouer. La jauge 2b.3 ne prolonge pas cette fenêtre (mesurée après l'audit, par le provider) ; l'audit lui-même n'est pas modifié.
- Piste C (2b) — sauvegardes de 2b.1 : les fichiers `…avant-compaction-20261021T1017…Z.duckdb` (2,5 Go) sont à côté des bases, sous `data/titles/`, donc embarqués par la sauvegarde restic de chaque nuit, et la tâche de nuit ne les supprime pas (elle ne touche que `data/compaction-backups/`). À retirer à la main après vérification par le user.
- Piste C (2b) — `scripts/restic-backup.sh` ne prend aucun verrou : la tâche de nuit refuse de tourner pendant une sauvegarde restic, mais une sauvegarde restic lancée à la main pendant une compaction arrêterait/redémarrerait le serveur sous elle. Couvert par les horaires (03:30 contre 04:00) et `TimeoutStartSec=25min`, pas par un verrou commun.
- Piste C (2b) — l'ADR 0026 est rédigé en français alors que CLAUDE.md (règle 15) veut les ADR en anglais seul ; l'amendement est en anglais, le corps n'est pas traduit.
- Piste C (2b), revue ronde 1, P2 — aucun verrou partagé entre `scripts/compact-passes-nuit.sh` et `scripts/deploy.sh` : un déploiement (push `main`) pendant la fenêtre de compaction échouerait sur le verrou DuckDB tenu par `compact-passes` (pas de corruption : la CLI tient le bail et le verrou de fichier). Consigné, non corrigé.
- Piste C (2b) — au timeout systemd (`TimeoutStartSec=25min`), le SIGTERM atteint le client `docker compose run` ; que le conteneur de compaction s'arrête aussi (relais de signal de `compose run`) n'est pas vérifié. S'il survivait, le filet relancerait le serveur pendant que la CLI tient la base : le serveur échouerait à l'ouvrir (verrou DuckDB, pas de corruption) et le script sortirait en erreur après ses 3 essais. À vérifier sur le VPS si ce timeout se produit un jour (passage mesuré ~90 s).
- Piste C (2b) — la jauge a jusqu'à 24 h de retard sur une recuisson faite serveur démarré (le contrôle de santé tourne au démarrage puis toutes les 24 h) : une telle recuisson peut attendre deux nuits avant d'être compactée. Une recuisson faite serveur arrêté est mesurée au redémarrage.
- (piste A, 2026-10-21) `HomePage.tsx` lit `data.privacy_warning` (`PrivacyBanner`, type web `privacy_warning`) mais `domain.HomePageResponse` ne l'émet pas : le bandeau « compte privé » de l'Accueil ne peut jamais s'afficher. Non traité (hors périmètre).
- (piste A) `Q25NeighborMatches` n'avait pas de départage des ex aequo (corrigé en 4.4 : il bloquait la parité avec l'ordre de l'Accueil, le test échoue sans le correctif).
- (piste A) Sans fixtures de démo (générées sur l'hôte de prod seulement), 62 specs E2E data-dépendantes sautent, dont `slice-5-home` : la couverture E2E de l'Accueil n'a pas pu être exercée dans le bac isolé de la piste A.
- (piste A, revue ronde 1, P2 consigné, non corrigé) Défilement ramené à 30 tuiles quand la tête rechargée change de curseur (`features/home/queries.ts` : `cursors` repart de zéro si `recent_matches_next_before` change) : rare (nouveau match pendant un défilement profond), assumé pour ne jamais recoller des pages à une autre tête.
- (à compléter pendant l'exécution)
