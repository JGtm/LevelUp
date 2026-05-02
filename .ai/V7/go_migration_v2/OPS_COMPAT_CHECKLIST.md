# OPS_COMPAT_CHECKLIST.md - Compat runtime et exploitation

> [!WARNING]
> Ne pas utiliser ce document seul.
> Lire aussi [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md), [MATRIX.md](MATRIX.md) et [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md).

## Role du document

Cette checklist porte ce que le plan maitre ne doit plus dupliquer :

1. compatibilite runtime ;
2. exigences d'exploitation ;
3. exigences auth / jobs / mode de test ;
4. realites packaging / deploy ;
5. pre-launch checklist executable.

## Decisions de compatibilite deja tranchees

1. Cible auth canonique : **MSAL**.
2. Support refresh tokens : **obligatoire** pour compatibilite et operations.
3. `src/app/media_watcher.py` et `src/utils/tailscale.py` : **sortis du scope du chantier Go principal**.
4. Le chantier Go ne s'ouvre qu'apres gel du corpus de reference Go : contrats, golden values et matrice.
5. La reference contractuelle de depart doit etre documentee explicitement dans le chantier Go lui-meme.
6. Aucun agent ne doit lancer un lot Go sans avoir relu ce document et `MATRIX.md`.

## Reference contractuelle de depart - explicite

### Precondition

1. La facade web cible et ses contrats sont geles avant le premier lot Go.
2. Les golden values et la matrice de couverture sont a jour avant toute implementation.

### Ce qui fait foi

1. `apps/web/` : comportement de la facade V7 et parcours reels consommes par l'utilisateur.
2. `apps/api/` et le schema OpenAPI versionne : contrat HTTP de reference.
3. Les suites de validation associees : Playwright, tests API et golden values relies au produit V7.
4. Le schema DuckDB v6, ses migrations et les fichiers de configuration actuels.

### Ce qui ne fait pas foi

1. Les anciennes pages Streamlit archivees.
2. Une intention orale non ecrite dans le plan ou les tests.
3. Une optimisation Go locale non couverte par la parite.

## Compat auth - obligatoire

### Cible canonique

1. Le chemin nominal Go doit etre MSAL Device Code Flow.
2. Le cache MSAL serialise dans `sync_meta` doit rester lisible ou explicitement migre.
3. Le portage Go doit conserver l'echange `access_token -> spartan_token + clearance_token`.

### Priorite de resolution auth

1. Par defaut, le chemin prefere reste MSAL.
2. Exception : si un refresh token exploitable existe deja, il est prioritaire sur un nouveau Device Code Flow interactif.
3. Priorite recommandee :
	a. `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_NORMALISE>`
	b. `oauth_refresh_token` persiste dans `sync_meta`
	c. `SPNKR_OAUTH_REFRESH_TOKEN`
	d. MSAL silent si un cache est present
	e. MSAL Device Code Flow interactif en dernier recours
4. Le plan Go doit documenter explicitement cette priorite au moment de l'implementation auth.

### Compatibilite a maintenir

1. Support de `SPNKR_OAUTH_REFRESH_TOKEN`.
2. Support de `SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_NORMALISE>`.
3. Support de `oauth_refresh_token` persiste dans `sync_meta` tant que le runbook et les jobs en dependent.
4. Le choix MSAL canonique ne justifie pas de casser les chemins refresh-token utilises par l'ops actuelle.

### Regle de portage

1. Pas de sprint auth termine sans tests MSAL.
2. Pas de sprint auth termine sans test de compatibilite refresh-token.
3. Pas de suppression d'un chemin refresh-token sans runbook explicite et mise a jour des scripts/ops concernes.

### Cas d'echec auth a traiter explicitement

1. Cache MSAL illisible ou invalide : journaliser, ignorer le cache, poursuivre avec la chaine de resolution auth.
2. Refresh token present mais refuse/revoque : le marquer comme non exploitable pour la tentative en cours, puis tenter la suite de la chaine sans boucle infinie.
3. Echec de l'echange `access_token -> spartan_token + clearance_token` : traiter comme echec auth bloquant, avec message explicite et sans etat semi-authentifie.
4. Aucun chemin auth valide : remonter un etat "reauth required" clair vers l'UI ou la CLI.
5. Un token d'environnement invalide ne doit jamais ecraser un token persiste valide sans decision explicite.

## Jobs longs - obligatoire

Le modele Go ne doit pas se contenter de goroutines en memoire.

Exigences minimales :

1. Persistance hors memoire du statut des jobs.
2. Au redemarrage, un job `running` doit devenir `interrupted` plutot que disparaitre.
3. Le bootstrap doit pouvoir exposer `active_sync_job_id`.
4. `POST /sync/initial` et `GET /jobs/{job_id}` doivent rester coherents avec ce modele.
5. Les jobs de sync doivent etre exclusifs : pas de sync concurrente.

Regle : un modele "goroutine seule + map memoire" n'est pas acceptable comme etat final.

### Exclusivite des syncs

1. Une seule sync a la fois pour toute l'application.
2. Si une sync est deja en `queued` ou `running`, toute nouvelle demande de sync doit etre refusee, pas mise en file d'attente implicite.
3. La reponse de refus doit inclure l'identifiant du job actif et son statut courant.
4. `active_sync_job_id` dans le bootstrap doit refleter cette exclusivite globale.
5. Apres redemarrage, un job `interrupted` n'est pas repris automatiquement ; une relance explicite est requise.

## DuckDB / write lease - obligatoire

### Principe de migration

1. La premiere implementation Go doit d'abord reproduire fidelement la semantique Python actuelle avant tout durcissement.
2. Le write lease Go ne doit pas promettre plus que ce que fournit aujourd'hui `_write_lease.py` tant qu'une strategie inter-process explicite n'a pas ete decidee.

### Semantique write lease a reproduire

1. Un lease par chemin absolu de DB.
2. Un seul writer logique par DB path a la fois.
3. Attente courte equivalente au Python : environ 5 secondes maximum.
4. En cas de timeout, warning explicite puis tentative d'ouverture quand meme ; ne pas changer cette semantique silencieusement dans Go.
5. Cette coordination est process-locale ; elle ne constitue pas a elle seule un verrou distribue ou inter-process.

### Consequence d'architecture

1. Tous les opens `read_write` DuckDB du runtime Go doivent passer par un composant central unique.
2. Aucun handler, job ou helper ne doit ouvrir une connexion `read_write` "en direct" hors de ce composant.
3. Les syncs et writers metier doivent etre orchestrés dans un seul runtime Go actif a la fois.
4. L'exclusivite globale des syncs complete la protection du write lease, elle ne la remplace pas.

### Recommandations pool / connexions

1. Pas de pool read-only illimite.
2. Pool read-only borne des le debut, avec bornes modestes et explicites par DB.
3. `ATTACH shared_matches_v2` une seule fois par connexion, jamais par requete.
4. Separer clairement les chemins `read_only` et `read_write`.
5. Une connexion writer ne doit pas etre recyclee comme connexion read-only generaliste.

### POC DuckDB minimal a exiger

1. Ouvrir metadata/shared/player sans migration implicite non voulue.
2. **Verifier la version DuckDB embarquee par `duckdb-go`** : doit etre compatible avec les fichiers crees par DuckDB Python 1.4.4 (meme majeure+mineure). Si l'ouverture declenche une migration de format, c'est un risque bloquant.
3. **Verifier le comportement de `database/sql` avec ATTACH** : `database/sql` gere son pool de connections de facon transparente. Une requete suivante peut s'executer sur une connection differente de celle qui a fait l'ATTACH. Valider la strategie choisie (`sql.Conn` pinee, `ConnInitFunc`, ou pool custom).
4. Verifier `ATTACH` player -> shared en read-only.
5. Verifier 10 lectures paralleles + 1 writer sur le meme path.
6. Verifier 2 writers sur le meme path : serialisation ou echec controle, jamais corruption.
7. Verifier 2 writers sur des DB path differents : independance effective.
8. Verifier types critiques : UBIGINT, TIMESTAMP WITH TIME ZONE, BOOLEAN, VARCHAR.
9. **Documenter le toolchain CGo Windows utilise** (ex : `w64devkit`, `tdm-gcc`, MSYS2 ucrt64) pour reproduction CI.

### Vues materialisees et post-sync

1. Le refresh des `mv_*` doit vivre dans le meme domaine transactionnel logique que le post-sync.
2. Le refresh ne doit pas contourner le write lease.
3. Un echec de refresh doit laisser un etat detectable et rejouable, pas une corruption silencieuse.

## Mode de demo/test - obligatoire

Le portage Go doit conserver un mode de demo/test utile a la boucle de dev et aux E2E.

### Objectif

1. Permettre de valider la facade V7 contre l'API Go sans dependance a Microsoft, Halo ou des donnees personnelles reelles.
2. Fournir une base deterministe pour comparer Python et Go sur les memes cas.

### Donnees minimales obligatoires

1. Un joueur demo "nominal" avec assez de matchs pour Career, History, Explorer et Match View.
2. Un cas sparse ou partiellement vide : zero match, peu de donnees ou champs absents.
3. Un cas PvE/Firefight.
4. Un cas multi-joueurs avec au moins deux profils relies.
5. Un cas avec medias indexes et un cas sans media.

### Surfaces minimales a couvrir

1. Bootstrap et players.
2. Resolveur de filtres.
3. Career, Match History, Explorer et Match View.
4. Settings read/write non destructifs.
5. Jobs read-only ou statuts simulables.

### Invariants de parite

1. Meme shape JSON, memes noms de champs, memes statuts HTTP.
2. Meme semantique de tri, pagination, valeurs nulles et degradations gracieuses.
3. Bypass auth explicite et visible.
4. Aucun appel reseau live requis vers Microsoft ou Halo dans ce mode.

### Regles d'execution

1. Le dataset de demo/test doit etre versionne.
2. Son mode de regeneration ou reset doit etre documente.
3. Toute nouvelle surface Go visible par la facade V7 doit avoir une couverture demo/test correspondante.

Regle : aucun socle Go n'est considere pret sans mode demo/test fonctionnel et deterministe.

## Surfaces explicitement sorties du scope Go principal

Ces surfaces existent dans le repo mais ne bloquent pas la feuille de route backend Go :

1. `src/app/media_watcher.py` - watcher Linux inotify.
2. `src/utils/tailscale.py` - helper Tailscale funnel.
3. scheduling legacy de `src/app/media_background.py`.

Sorti du scope ne veut pas dire oublie : la decision doit rester explicite dans `MATRIX.md` et le runbook.

## Packaging et deploiement - realites a respecter

1. Binaire unique avec sous-commandes : recommandation valide.
2. **Taille binaire** : le binaire CGo+DuckDB+web embarque pese **100-200 MB** (`duckdb-go` lie DuckDB en statique, plus `go:embed` des assets React). C'est normal pour ce type de stack ; ne pas cibler <50 MB.
3. Si media indexing est conserve, le systeme n'est pas "zero dependance runtime" au sens strict : ffprobe/ffmpeg restent a traiter.
4. Si Docker reste supporte, la strategie permissions/bind-mount doit rester explicite.
5. Pas de promesse de `scratch`/`distroless` si le runtime final a encore besoin de binaires auxiliaires ou d'un filesystem mutable non trivial.
6. **Versioning** : injecter version, commit SHA et date de build via `-ldflags` (`-X main.version=... -X main.commit=... -X main.buildDate=...`). Exposer dans `/health` et `--version`.

## CI/CD - strategie de build

### Contraintes CGo

Le build Go+DuckDB utilise CGo (`#cgo LDFLAGS: -lduckdb`). Cela impose :

1. **Build matrix** : GitHub Actions avec matrix `os: [ubuntu-latest, windows-latest]` × `arch: [amd64]`.
2. **Windows** : installer un compilateur C (MinGW via `w64devkit` ou `tdm-gcc`) dans le step de build. Documenter la version exacte.
3. **Linux** : `gcc` et `g++` suffisent (pre-installes sur `ubuntu-latest`).
4. **Pre-built DuckDB** : telecharger `libduckdb.a` + headers depuis les releases DuckDB plutot que compiler depuis les sources (D7). Matcher la version 1.4.4.

### Pipeline cible

```yaml
# Simplifie - structure a adapter
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - name: Install DuckDB pre-built
        run: # telecharger libduckdb.a + headers pour la plateforme
      - name: Build
        run: go build -ldflags="-X main.version=${{ github.ref_name }}" -o levelup ./cmd/levelup
      - name: Test
        run: go test ./...
      - uses: actions/upload-artifact@v4
```

### Regles CI

1. Pas de merge sans build vert sur les deux OS.
2. Tests unitaires + golden values dans le pipeline.
3. Tests E2E Playwright en job separe (sur Linux uniquement, contre le binaire Go).
4. Cache des modules Go et du pre-built DuckDB entre runs.

## Configuration - strategie native Go

La configuration du produit Go utilise un modele natif sans dependance `viper` :

### Ordre de resolution

1. **Struct Go avec valeurs par defaut** : `type Config struct` avec tags JSON, valeurs par defaut dans le constructeur.
2. **Fichier JSON** : `app_settings.json` lu au demarrage — meme format que Python pour la compatibilite.
3. **Variables d'environnement** : surcharge du fichier par `os.Getenv()` avec prefixe `LEVELUP_` (ex : `LEVELUP_PORT`, `LEVELUP_DATA_DIR`).
4. **CLI flags** : `flag.StringVar` pour les sous-commandes sync/backfill.

### Regles

1. Pas de `viper`, pas de `envconfig`, pas de YAML. Struct Go + JSON + env vars = suffisant.
2. Le parsing JSON utilise `encoding/json` de la stdlib.
3. La validation se fait dans un `Validate() error` sur la struct Config.
4. Si le nombre de variables d'environnement depasse 15, reconsiderer `envconfig` (pas avant).
5. `db_profiles.json` et `.env.local` restent compatibles avec le format Python actuel.

## CORS - liste d'origines explicite

### Regles

1. La liste des origines CORS autorisees est **explicite** dans `app_settings.json`, pas wildcardee.
2. Origines actuelles Python a reproduire : `http://localhost:3000`, `http://localhost:5173` (dev Vite), plus les origines de prod si deployees.
3. En mode dev (`LEVELUP_ENV=dev`), ajouter automatiquement `http://localhost:*`.
4. En mode prod, n'autoriser que les origines nommees dans la config.
5. Middleware CORS dans Chi : `github.com/go-chi/cors` avec `AllowedOrigins`, `AllowCredentials: true`, `AllowedMethods: GET, POST, PATCH, DELETE, OPTIONS`.

## Hot reload - workflow developpeur

### En developpement

1. **Air** (`github.com/air-verse/air`) pour le hot reload du binaire Go. Configuration `.air.toml` a la racine.
2. Surveiller `cmd/`, `internal/`, `pkg/` — ignorer `apps/web/`, `data/`, `*.duckdb`.
3. Temps de rebuild cible : <5s (le CGo ralentit le build ; pre-compiler `libduckdb.a` une fois).

### En production

1. Pas de hot reload. Binaire statique, restart propre via le graceful shutdown existant.
2. Le binaire expose `--version` pour verifier la version deployee.

## Pool multi-joueurs - degradation explicite

### Probleme

LevelUp gere N joueurs, chacun avec sa propre `stats.duckdb`. En Python, chaque `DuckDBRepository` ouvre une connexion dediee. En Go avec un pool `database/sql`, il faut gerer la scalabilite.

### Regles de degradation

1. **Limite de connexions simultanees** : borner a `max_open_conns` par DB path (defaut : 4 read-only, 1 read-write).
2. **Limite de joueurs actifs** : si >10 joueurs sont accedes dans les 5 dernieres minutes, les connexions des joueurs les moins recemment accedes sont fermees (LRU eviction).
3. **Timeout d'acquisition** : si aucune connexion n'est disponible dans le pool en <2s, retourner HTTP 503 `"too many concurrent requests"` plutot que bloquer indefiniment.
4. **Monitoring** : exposer dans `/debug/stats` le nombre de connexions actives par DB path, le nombre de joueurs charges et le nombre d'evictions LRU.
5. **ATTACH shared** : la connexion ATTACH vers `shared_matches_v2.duckdb` est partagee par toutes les connexions read-only, initialisee une seule fois dans `ConnInitFunc` du pool.

### Scenarios de stress a tester

1. 5 joueurs accedes simultanement en read-only → pas de degradation visible.
2. 15 joueurs accedes → eviction LRU des moins actifs, latence <100ms supplementaire.
3. Sync active + 3 lecteurs sur la meme DB player → write lease respecte, lecteurs non bloques.
4. Pool sature → 503 explicite, pas de deadlock.

## Migration des donnees utilisateurs

### A verifier absolument

1. Ouverture de fichiers DuckDB existants sans migration implicite non voulue.
2. Compatibilite type/schema avec la version DuckDB embarquee cote Go.
3. Lisibilite du cache MSAL existant.
4. Strategie explicite pour les sessions HTTP existantes.
5. Compatibilite `db_profiles.json`, `app_settings.json`, `.env.local`.

### Acceptable

1. Invalidation des sessions HTTP au premier demarrage Go.
2. Re-authentification utilisateur explicite.

### Non acceptable

1. Ouvrir une DB utilisateur et declencher une migration implicite non maitrisee.
2. Casser un cache auth existant sans runbook explicite.

## Discipline d'evolution pendant le portage

1. Pas de freeze total du produit.
2. Pas de retard silencieux de plusieurs semaines entre Python et Go sur OpenAPI, golden values ou schema DuckDB.
3. Les bugfixes Python de production doivent etre repercutes dans les golden values avant d'etre consideres "portes" en Go.
4. Toute migration DuckDB Python doit avoir un statut explicite cote Go dans la semaine.

## Checklist pre-lancement

### Gates minimaux avant la premiere ligne de Go

- [ ] Le corpus de reference Go est explicite et gele
- [ ] La reference contractuelle de depart est nommee explicitement et relue
- [ ] `MATRIX.md` initialise avec les surfaces touchees par le premier lot
- [ ] POC DuckDB Go valide sur Windows et Linux
- [ ] Version DuckDB `duckdb-go` compatible avec les fichiers Python 1.4.4 (pas de migration implicite)
- [ ] Strategie pool `database/sql` + ATTACH validee
- [ ] Toolchain CGo Windows documente et reproductible en CI
- [ ] Strategie auth confirmee : MSAL canonique + refresh tokens supportes
- [ ] Strategie de compatibilite cache MSAL Python → Go documentee (lecture directe ou invalidation)
- [ ] Strategie jobs persistants definie avant tout travail setup/sync
- [ ] Mode de demo/test Go defini des le socle
- [ ] Mecanisme de feature flags choisi pour la bascule progressive

### Gates minimaux avant toute bascule read-only

- [ ] OpenAPI versionne et compare au backend Python de reference
- [ ] golden values a jour sur les surfaces touchees
- [ ] ecarts documentes ou nuls
- [ ] latence et erreurs mesurables

### Gates minimaux avant toute bascule auth/jobs

- [ ] tests MSAL verts
- [ ] tests refresh-token verts
- [ ] priorite MSAL / refresh tokens couverte par tests
- [ ] jobs persistes au redemarrage
- [ ] `active_sync_job_id` expose correctement
- [ ] refus explicite d'une deuxieme sync concurrente

### Gates minimaux avant le premier sync Go sur donnees reelles

- [ ] backup automatique de `shared_matches_v2.duckdb` et des DB player execute et verifie
- [ ] script de restore rapide teste et documente
- [ ] les fichiers DuckDB n'ont pas de WAL accessible externement ni de snapshot natif — un sync Go defaillant peut corrompre les donnees sans rollback simple

### Gates minimaux avant extinction Python

- [ ] runbook utilisateur mis a jour
- [ ] scripts critiques d'exploitation couverts
- [ ] migrations DuckDB, sync, backfill et maintenance critiques ont leur equivalent Go ou une decision explicite de maintien hors scope
- [ ] aucune surface active n'est absente de `MATRIX.md`
