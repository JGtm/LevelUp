# Commandes utiles — LevelUp

English version: [../COMMANDS.md](../COMMANDS.md)

> Aide-mémoire de la stack actuelle : backend Go (`apps/go-api`) + frontend React/Vite (`apps/web`).
> L'outillage d'exploitation est le CLI `levelup` (`apps/go-api/cmd/levelup`). Les cibles `make`
> sont dans le `Makefile` racine. L'accès DuckDB exige CGO (voir [Tests](#tests)).

---

## Lancement

```bash
make dev          # API Go (air, :8000) + frontend Vite (:5173) — Ctrl+C arrête tout
make go-api-dev   # API Go seule (hot-reload air)
make web          # Frontend seul (Vite, :5173)
make stop         # Arrête les serveurs dev (kill par port, API + 5173)
make restart      # stop + dev
```

Ouvrir http://localhost:5173 une fois `make dev` lancé.

---

## Build

```bash
make go-api-build   # CGO_ENABLED=1 go build -> apps/go-api/bin/server
make install-web    # npm install dans apps/web
make generate-types # Types TypeScript depuis apps/go-api/api/openapi.yaml
make check-types    # tsc -b (typecheck seul)
```

---

## CLI `levelup`

Compilé depuis `apps/go-api/cmd/levelup`. Lancer via `go run` (CGO requis) ou builder un binaire.
Utiliser `LEVELUP_REPO_ROOT` pour pointer le repo de données (auto-détecté si absent).

```bash
cd apps/go-api
CGO_ENABLED=1 go run ./cmd/levelup <commande> [options]
# Aide par commande :
CGO_ENABLED=1 go run ./cmd/levelup <commande> --help
```

### Synchronisation (API Halo)

```bash
# Sync delta — nouveaux matchs uniquement
go run ./cmd/levelup sync-delta --gamertag MonGamertag
go run ./cmd/levelup sync-delta --all --max-matches 25
# options : --match-type all|matchmaking|custom|local  --rps N  --token-pool-size N

# Sync complète — parcourt les N derniers matchs API, insère les manquants (comble les trous)
go run ./cmd/levelup sync-full --gamertag MonGamertag --max-matches 500

# Backfill des achievements Xbox (admin one-shot)
go run ./cmd/levelup sync-achievements --all [--dry-run]
```

**Aucun joueur n'a besoin de son propre jeton.** Tous les chemins de synchronisation —
`--gamertag` comme `--all` — passent par le pool de jetons : l'historique des matchs, les
statistiques, les films et les CSR sont des points d'accès PUBLICS que n'importe quel jeton du
parc sert (`PolicyAnyPublic`). Un profil suivi qui ne s'est jamais connecté par le SSO Xbox se
synchronise comme les autres. Il suffit que le pool tienne au moins un jeton sain.

Le rang de carrière ne fait PAS partie de la synchronisation : il est servi par le flux
séparé de carrière en direct (`service.CareerLiveService`), et `career_synced` vaut toujours
`false` dans le résumé du sync, jeton ou pas. `/careerranks` est lui-même PUBLIC : mesuré le
2026-09-16 avec trois jetons prêteurs différents sur un xuid tiers, il rend le même rang et la
même XP que l'appel du propriétaire — le client poolé l'acquiert donc en `PolicyAnyPublic`
comme tout le reste (D4, plan robustesse du sync). Le cron de personnalisation Spartan est le
seul appelant qui exige le jeton propre du joueur (403 pour un tiers, mesuré), et garde pour
cette raison son contrôle `HasPlayer`.

Les passes `backfill --csr` / `--shared-csr` et les commandes de films (`archive-films`,
`backfill-killsource --online`, `replay-events`) suivent la même doctrine : `--gamertag` nomme le
joueur traité, pas un prêteur de jeton.

**Note d exploitation.** Une passe de synchronisation en ligne de commande tient la base
partagée en ÉCRITURE et applique les migrations shared du titre avant sa première insertion :
à lancer **serveur arrêté** (un seul writer, ADR 0013). Elle fait aussi tourner les jetons de
rafraîchissement de TOUT le parc via le pool — ne jamais faire tourner les jetons du parc
pendant qu un serveur tourne, sinon ce serveur garde les anciens jetons en mémoire et finit en
`reauth_required` sur N comptes.

`--token-pool-size N` plafonne le nombre de slots SAINS, pas le nombre de sources tentées : le
scan est parcouru en entier, dans l'ordre alphabétique des gamertags, et une source dont le
jeton de rafraîchissement ne se résout pas ne consomme pas le quota. `0` prend tous les jetons
sains du parc. Avant le 2026-09-16, le plafond tronquait le scan AVANT de résoudre : avec
`--token-pool-size 1`, un seul compte révoqué pouvait être tenté et la commande échouait sur
« aucun slot créé ».

### Backfill (local Go ; CSR/weapons nécessitent des tokens Halo)

```bash
go run ./cmd/levelup backfill --gamertag X --citations        [--force]
go run ./cmd/levelup backfill --all          --lusr           [--force]
go run ./cmd/levelup backfill --gamertag X --perf             [--force]
go run ./cmd/levelup backfill --gamertag X --engagement-scores
go run ./cmd/levelup backfill --gamertag X --csr             [--force]   # tokens Halo
go run ./cmd/levelup backfill --all          --shared-csr     [--dry-run] # tokens Halo
go run ./cmd/levelup backfill --all          --weapons        [--force]   # film CDN
go run ./cmd/levelup backfill --gamertag X --citations-recompute-all
```

Manches des modes qui se décident aux manches (ADR 0032) — une colonne que seule l'API peut
remplir, donc qu'aucune resynchronisation ne répare. **Serveur arrêté** pour `--apply` (un
seul writer, ADR 0013) :

```bash
# répétition à blanc : aucune écriture, aucun droit d'écriture demandé
go run ./cmd/backfill-team-rounds --gamertag X

# application — restreinte PAR DÉFAUT aux variantes déclarées dans regulation.toml
# [rounds_decide] (26 matchs, ~7 s). --all couvre tout le corpus (~1 900 appels d'API).
go run ./cmd/backfill-team-rounds --gamertag X --apply [--all] [--limit N] [--match ID]
```

#### Remplir `match_kill_events` / `match_weapon_shots` depuis les films — `backfill-killsource`

**100 % hors ligne** (films du cache local : ni réseau, ni jetons, ni CDN), **serveur arrêté**
(elle tient la base partagée en écriture — un seul writer, ADR 0013). Elle joue deux passes dans
cet ordre : les FILMS (décodage), puis le CRÉDIT (transformation SQL → SQL, quelques minutes).

```bash
go run ./cmd/levelup backfill-killsource --dry-run      # bilan : total, déjà à jour, chunks, ETA
go run ./cmd/levelup backfill-killsource                # tout : films puis crédit, 3 ouvriers
go run ./cmd/levelup backfill-killsource --workers 1    # la boucle en série d'avant le lot 5.24
go run ./cmd/levelup backfill-killsource --limit 20     # les 20 films les moins chers
go run ./cmd/levelup backfill-killsource --credit-only  # la passe SQL → SQL seule
go run ./cmd/levelup backfill-killsource --force        # redécode même ce qui est à jour
go run ./cmd/levelup backfill-killsource --status       # DANS UN AUTRE TERMINAL : où elle en est
```

**`--workers` (défaut 3) — N films décodés en parallèle, UN SEUL qui touche la base.** La
décomposition du coût (lot 5.24.1) mesure **93 à 99 % du temps d'un film en CPU hors base** :
c'est la seule raison d'être du parallélisme ici. Les écritures et les deux lectures par match
passent toutes par un jeton unique (`PorteDeLaBase`), donc à tout instant au plus un goroutine
parle à la base. Le plafond est une MESURE, pas un réglage : le pire film du corpus culmine à
**422 Mio**, la passe se plafonne à 4 Gio, soit **9 ouvriers au maximum** — au-delà, la commande
refuse au démarrage en citant le chiffre. Gain mesuré à 3 ouvriers sur 9 films : **9,7 s → 3,5 s
(×2,7)**, lignes écrites identiques (`TestOuvriers_MemesLignesQuUnSeulOuvrier`).

**`--status` — la question « il en est où ? », depuis un autre terminal.** Elle LIT le fichier
d'état et l'affiche une fois, **sans ouvrir aucune base** (c'est la seule façon d'interroger un
processus qui tient la base partagée en écriture). Le fichier est réécrit **après chaque film**,
dans `data/global/admin_state/backfill_killsource_{slug}.json` :

```
backfill-killsource [halo_infinite] — phase films, PID 11480
  demarree     2026-09-22T13:19:13+02:00 (il y a 0s)
  mise a jour  2026-09-22T13:19:14+02:00 (il y a 0s)
  revisions    morts killsource-2026-09-22.2 | isolement isolement-2026-09-15-decoupage-du-catalogue
  films        1 / 3 traites — 5 chunks / 100
               1 ecrits (42 morts), 0 sans film, 0 sans kill-feed, 0 cle inconnue, 0 abandons sur delai, 0 erreurs
               1500 deja a jour au demarrage (sautes : c est la REPRISE, et elle se decide en base)
               3 ouvrier(s), 0.00 films/min, 0.400 s/chunk mesure — reste ~13s
               fin estimee vers 2026-09-22T13:19:26+02:00
  dernier fini petit (5 chunks) en 2.00 s — ecrit
  EN COURS     gros (65 chunks) depuis 12.0 s
```

L'**ETA compte des CHUNKS, pas des films**, et il est divisé par le nombre d'ouvriers : les gros
films passent en dernier, donc un reste compté en nombre de films annoncerait une fin proche
juste avant la queue la plus chère. Le coût par chunk est celui **mesuré depuis le début de la
passe**, pas une constante. Un état non mis à jour depuis plus de 10 minutes est signalé comme
tel. Le terminal de la passe, lui, reçoit une ligne de progression **tous les 25 films OU toutes
les 60 s**, avec les mêmes chiffres.

**Reprise — la clé est `decoder_rev`, EN BASE.** Un match dont toutes les passes courantes (vues
`_latest`) portent leur révision de décodeur courante est sauté : interrompre et relancer **la
même commande** repart au dernier état. Le fichier d'état n'est **pas** une source de vérité pour
la reprise : le supprimer ne perd qu'un affichage. `Ctrl-C` (ou `SIGTERM`) arrête la
DISTRIBUTION des films ; ceux qui sont en vol vont au bout et sont écrits, l'état est fermé avec
sa cause, et la commande sort avec le code **130** (distinct du 0 d'une passe finie et du 1 d'une
panne). Un **second** `Ctrl-C` tue le processus immédiatement.

#### Projeter les artefacts de rejeu en base — L'ORDRE DE RELEASE N'EST PAS INTERCHANGEABLE

Deux passes lisent les artefacts de rejeu DÉJÀ cuits (`data/cache/replays/{slug}/{short8}.json`)
et les projettent vers les tables partagées. **Aucune des deux ne décode de film.** Ce sont des
tâches de RELEASE, et toutes deux exigent le **serveur arrêté** : elles prennent `OpenReadWrite`
sur la DB partagée et jouent elles-mêmes les migrations — y compris sous `--dry-run`.

```bash
# 1. RE-CUIRE les artefacts d'abord — le schéma 39 périme tout artefact antérieur, et c'est
#    cette passe-là qui fait NAÎTRE `bombStats` dedans.
go run ./cmd/levelup backfill-replay [--dry-run] [--force] [--limit N] [--only-existing]

# 2. Usages d'équipement et de socles -> match_usage_players + match_usage_films.
go run ./cmd/levelup backfill-usage-summary [--dry-run] [--force] [--match ID] [--limit N] [--title S]

# 3. Statistiques d'Assaut -> match_bomb_stats (append-only) + faits datés dans
#    match_objective_events. Répétition à blanc D'ABORD : elle imprime les compteurs par
#    match et n'écrit rien.
go run ./cmd/levelup backfill-bomb-stats --dry-run
go run ./cmd/levelup backfill-bomb-stats [--force] [--match ID] [--limit N] [--title S]

# 4. Prises nettes de drapeau -> match_flag_grabs_net (append-only). Elle lit les artefacts
#    TELS QU'ILS SONT : aucun décodage, AUCUNE RECUISSON — tout artefact de schéma 14 ou
#    plus porte déjà la chronologie de portage du drapeau. Indépendante de la passe (1),
#    elle peut tourner avant elle.
#    La fenêtre de jonglage vient de regulation.toml ; LA CHANGER EXIGE --force, car les
#    lignes déjà écrites portent l'ancienne fenêtre et la reprise ne les reverrait jamais.
go run ./cmd/levelup backfill-flag-grabs-net --dry-run
go run ./cmd/levelup backfill-flag-grabs-net [--force] [--match ID] [--limit N] [--title S]

# 4 bis. NIVEAUX D ARME des prises de socle -> match_pad_pickups_by_tier (append-only).
#    Meme motif que (4) : elle LIT les artefacts TELS QU ILS SONT, sans decodage, SANS
#    RECUISSON. Le niveau vient de la CARTE (l emplacement Forge que le socle du match
#    confirme, reference map_weapon_pads.json) et des equipements de depart du film ; jamais
#    du nom de l arme. AJOUTER UNE CARTE A LA REFERENCE EXIGE --force : les lignes deja
#    ecrites portent l ANCIEN croisement (des prises restees en `non_classe` qui deviendraient
#    `terrain` ou `puissance`), et la reprise ne les reverrait jamais.
go run ./cmd/levelup backfill-pad-tiers --dry-run
go run ./cmd/levelup backfill-pad-tiers [--force] [--match ID] [--limit N] [--title S]

# 5. Rasters d'occupation tactique -> fichiers sidecar JSON sous
#    data/cache/replays/{slug}/rasters/. AUCUNE base n'est ouverte, pas même en lecture :
#    le sidecar est par match et anonyme, il n'y a rien à demander à DuckDB.
go run ./cmd/levelup tactical-rasters --backfill [--dry-run] [--limit N] [--title S]
```

La passe (5) est idempotente : un sidecar n'est réécrit que s'il manque, si son propre
`schema_version` n'est plus le courant, ou si son `artifact_schema_version` ne correspond
plus à celui de l'artefact dont il a été projeté (donc après une re-cuisson). Une seconde
passe immédiate écrit zéro fichier. **Le schéma 3 du sidecar** ajoute les morts (avec la
distance à chaque autre joueur nommé vivant à cet instant), les routes de sortie de spawn et
le drapeau de spawn de départ : tout sidecar v2 est périmé, et cette passe le réécrit.

Lancer (3) avant (1) est un **no-op SILENCIEUX** : un artefact antérieur au schéma 39 ne porte
aucun `bombStats`, rien n'est écrit et chaque match tombe dans le compteur « sans calque ». Les
deux passes sont reprenables — un match déjà présent dans la vue `_latest` est sauté, sauf
`--force`. `backfill-usage-summary` re-résume en plus quand la révision de projection ou le
schéma de l'artefact a bougé.

### Backup / restore

```bash
go run ./cmd/levelup backup  --gamertag X [--output-dir D] [--compression-level 9]
go run ./cmd/levelup restore --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
go run ./cmd/levelup restore-csr --gamertag X --backup PATH [--dry-run] [--mode preserve|overwrite]
```

### Identités joueur (annuaire et purge — ADR 0035)

Quatre registres décrivent un joueur : le compte (`data/auth/users.json`), les identifiants
(`data/auth/watcher_tokens/{xuid}.json`), le profil de suivi (`db_profiles.json`) et le suivi
live du daemon watcher. La seule clé qui les relie est le **xuid**. `identity list` les lit
ensemble et signale ce qui ne colle pas ; `identity purge` retire une identité de tous.

```bash
go run ./cmd/levelup identity list                          # l'annuaire, anomalies comprises
go run ./cmd/levelup identity purge <xuid>                  # SIMULATION : imprime le rapport, ne supprime rien
go run ./cmd/levelup identity purge <xuid> --yes            # exécute
```

- **La base partagée des matchs n'est jamais touchée.** Les matchs déjà persistés dans
  `shared_matches_v2.duckdb` portent aussi les données des adversaires et des coéquipiers du
  joueur purgé, et l'entrepôt est append-only par construction (ADR 0026). La purge ne
  l'ouvre même pas.
- **La simulation est le défaut.** Sans `--yes`, la commande imprime le rapport complet de ce
  qu'elle ferait et sort 0.
- **Ordre** (ADR 0035 D6) : suivi live, puis entrées de profil et dossiers joueur, dossiers
  orphelins, identifiants, appartenances aux groupes, et enfin le compte. Une étape en échec
  n'arrête jamais les suivantes — le rapport est toujours complet et nomme chaque échec.
- **Un compte administrateur est refusé.** Retirer le dernier administrateur fermerait
  l'administration à clef ; cela se fait délibérément, à la main.
- **Pré-requis : le serveur ne doit pas tenir la player DB.** La purge supprime le dossier du
  joueur avec son fichier DuckDB. Elle n'évince que les handles du *processus courant* : si le
  serveur tient le fichier, la suppression échoue (verrou Windows) et l'étape est rendue en
  échec dans le rapport. Arrêter le serveur, ou purger un joueur qui n'est pas suivi.

### Référentiels / seed / migration

```bash
go run ./cmd/levelup seed career-ranks | citation-mappings | medals | rank-translations
go run ./cmd/levelup seed-demo            # génère les données démo anonymisées (data/demo/)
go run ./cmd/levelup migrate              # migre les données vers le namespace multi-titres
go run ./cmd/levelup add-title --name "Halo MCC" [--slug s] [--capabilities matchmaking,media] [--xbox-id X] [--steam-id S]
```

#### Référentiel d'icônes de médailles (`static/medals/{slug}/{medal_id}.png`)

La page Médailles sert un PNG par identifiant de médaille. `refresh-metadata medal-images`
compare le catalogue officiel GameCMS (`hi/Waypoint/file/medals/metadata.json`) aux icônes
versionnées et découpe les manquantes dans la feuille de sprites officielle
(`hi/Waypoint/file/medals/images/medal_sheet_xl.png`, 4096×4096, tuiles de 256 px,
16 colonnes). Les jetons viennent du store watcher (ADR 0023 — jamais de re-capture) ;
**aucun fichier DuckDB n'est ouvert**, la commande se lance donc serveur allumé.

```bash
cd apps/go-api
# rapport seul (défaut) : décomptes et les deux listes d'écart
go run ./cmd/refresh-metadata medal-images --player JGtm
# découpe toutes les médailles du catalogue dont l'icône manque
go run ./cmd/refresh-metadata medal-images --player JGtm --download
# la feuille DEVANCE le catalogue : l'auditer tuile par tuile, puis ancrer un id à la main
go run ./cmd/refresh-metadata medal-images --player JGtm --audit-sheet
go run ./cmd/refresh-metadata medal-images --player JGtm --extract-tiles 55,56 --extract-dir /tmp/tuiles
go run ./cmd/refresh-metadata medal-images --player JGtm --pin 1053114074:55
# options : --title-id  --out-dir  --dump-raw FICHIER  --metadata-path  --sprite-sheet-path  --tile PX
```

Référence des points d'accès et de la disposition des `spriteIndex` : den.dev, *Halo Infinite
Medal API: Infection, VIP, Extraction* (2023-10-11) —
<https://den.dev/blog/halo-infinite-medals-api/>. La feuille contient des tuiles que le JSON
ne liste pas : le JSON reste la référence, `--pin` est la porte de sortie, et le garde-rail
`internal/games/halo_infinite/medal_icons_test.go` échoue dès qu'une médaille de la taxonomie
n'a pas d'icône.

### Chaînes de fabrication des assets versionnés

Onze chaînes hors ligne, toutes sous `apps/go-api/cmd/`, produisent des fichiers commités
(`data/titles/{slug}/reference/`, `static/`, ou un fichier Go généré). Aucune n'est câblée
dans `cmd/server` — le décodage du jeu et le code GPLv3 (`internal/himap`, `internal/ooz`,
Kraken/Oodle) restent isolés dans ces binaires. Lancées depuis `apps/go-api` sauf mention
contraire. `--title`/`-title` vaut `halo_infinite` par défaut partout.

#### weapon-icons (build + table)

```bash
go run ./cmd/weapon-icons-build                      # racine du jeu auto-détectée
go run ./cmd/weapon-icons-build -deploy "D:/SteamLibrary/.../Halo Infinite/deploy"
# flags : -out DIR  -max N (images par atlas)  -probe N (profondeur de recalage descripteur→ressource)
go run ./cmd/weapon-icons-table                      # derive la table Go depuis index.json
```

- Sortie : `static/weapons-assets/halo_infinite/jeu/` — 168 PNG (icônes d'armes en contour et
  en silhouette, plus l'atlas du kill feed) + `index.json` (build) ;
  `internal/games/halo_infinite/weapon_icons_table.go`, généré — NE PAS ÉDITER (table).
- Prérequis : jeu installé + cgo (Kraken) pour `weapon-icons-build`. `weapon-icons-table` n'a
  besoin ni de l'un ni de l'autre — il ne lit que `index.json`, déjà versionné : il tourne
  partout, y compris en CI.
- À rejouer : après une mise à jour de contenu qui fait grandir les tables (build) ; après
  chaque exécution de `weapon-icons-build`, pour garder la table à jour (table).
- Chaîne complète, tables de correspondance et pistes réfutées :
  `.ai/V7.5/icones/ETAT_DE_L_ART_ICONES.md`.

#### mapquant-build

```bash
CGO_ENABLED=1 go run ./cmd/mapquant-build [--levels DIR] [--title slug] [--out FILE]
```

- Sortie : `data/titles/{slug}/reference/map_quant_bounds.json` — les bornes monde par carte
  qui transforment les coordonnées quantifiées du film en coordonnées monde.
- Prérequis : jeu installé (sauf `--levels` explicite) + cgo. Le lien nom affiché -> module est
  une table codée en dur dans l'outil : une carte absente de cette table est absente du
  catalogue par construction (refus de publier une coordonnée devinée).
- À rejouer : quand le lien module d'une nouvelle carte est établi, ou si le jeu change ses
  modules/BSP.

#### film-profiles-build

```bash
go run ./cmd/film-profiles-build [--title slug] [--check] [--out FILE] [--bounds FILE]
```

- Sortie : `data/titles/{slug}/reference/film_profiles.json` — le catalogue des profils de film
  (ce que le dépôt sait de la grammaire d'un film, indexé par les trois clefs que le film
  *écrit*). L'outil produit le bloc `derived` **seulement** — l'empreinte de
  `map_quant_bounds.json`, pour que le profil dise de quelle dérivation des fichiers du jeu il
  est solidaire. Les blocs saisis — `entries` et `registryFingerprints` (une empreinte de
  registre ECS par clef écrite : le build, ou la version majeure pour les films sans section
  d'identification) — sont écrits à la main, avec provenance, et recopiés tels quels.
- Prérequis : ni jeu installé, ni réseau, ni cgo. `--check` n'écrit rien et sort en 1 si le
  fichier commis n'est pas celui que l'outil produirait. Depuis un worktree, exporter
  `LEVELUP_REPO_ROOT=<le worktree>` ou passer `--out`/`--bounds` (`db_profiles.json` est
  gitignoré et n'existe que dans le checkout principal).
- À rejouer : après `mapquant-build` (mise à jour du jeu, nouvelle carte), ou après l'ajout
  d'une entrée de profil. Procédure d'ajout d'un build : `docs/RUNBOOK_FILM_PROFILES.md`
  (EN, les runbooks sont EN-only).
- Gate avec le jeu installé :
  `CGO_ENABLED=1 go test -tags=gamefiles ./cmd/film-profiles-build/ -count=1` — rejoue la chaîne
  entière (bornes régénérées depuis les `.module`, puis catalogue commis égal à l'octet à ce que
  la chaîne produit). Skip là où Halo Infinite n'est pas installé.

#### mapcallouts-build

```bash
CGO_ENABLED=1 go run ./cmd/mapcallouts-build                            # passe native seule
CGO_ENABLED=1 go run ./cmd/mapcallouts-build --forge-only --forge-fetch # passe Forge seule
CGO_ENABLED=1 go run ./cmd/mapcallouts-build --lexique --forge-only     # + lexique de chaînes
```

- Sortie : `data/titles/{slug}/reference/map_callouts.json` (zones nommées natives + Forge) ;
  `--lexique` écrit en plus `callouts_lexique.csv` à côté. Lit en entrée le
  `callouts_i18n.csv` versionné (816 libellés).
- Prérequis : jeu installé pour la passe native et pour `--lexique` ; cgo dans tous les cas
  (pour compiler) ; réseau uniquement avec `--forge-fetch` (récupération anonyme des `.mvar`
  UGC, sans jeton). Un garde-fou bloque l'écriture d'une carte qui perdrait des sommets par
  rapport au fichier déjà commité (`--accepte-perte` pour outrepasser).
- À rejouer : mise à jour du jeu (passe native, ou `--lexique`, qui « ne se rejoue qu'à une
  mise à jour du jeu » selon son propre en-tête) ; une nouvelle carte Forge a besoin de ses
  callouts (`--forge-fetch`).

#### mapfond-build

```bash
CGO_ENABLED=1 go run ./cmd/mapfond-build [--maps "Cliffhanger,Catalyst"] [--title slug] \
  [--out-dir DIR] [--style jeu] [--natives=false] [--forge=false] [--rapport FILE]
```

- Sortie : `data/titles/{slug}/reference/map_backgrounds/{cle}.png` + `{cle}.json` (sidecar de
  calage) par carte — 218 fichiers aujourd'hui.
- Prérequis : jeu installé — TOUJOURS, aucun flag ne permet de l'éviter, même en Forge seul ;
  chaîne cgo/GPLv3 (`internal/himap` -> `internal/himodule` -> `internal/ooz`, jamais liée
  dans `cmd/server`) ; exige `map_objectives.json` déjà construit (dépendance dure, échoue
  sans lui) ; utilise `map_quant_bounds.json` / `map_callouts.json` / `map_positions_jouees.json`
  / `map_fond_reglages.json` s'ils existent, dégrade avec un avertissement sinon.
- À rejouer : non documenté dans l'outil lui-même ; en pratique, une nouvelle carte (native ou
  Forge) a besoin de son fond cuit.

#### mapobj-build

```bash
go run ./cmd/mapobj-build --player <Gamertag> --map-id <uuid> [--map-id <uuid>...]
go run ./cmd/mapobj-build --player <Gamertag> --all                # tout match_registry
go run ./cmd/mapobj-build --from-file <chemin.mvar> --map-id <uuid> # hors ligne
go run ./cmd/mapobj-build --refresh-from <dossier de .mvar>         # hors ligne, tout le catalogue
```

- Sortie : `data/titles/{slug}/reference/map_objectives.json`, écriture atomique (fichier
  temporaire + renommage). `map_objects.csv` et `forge_object_types.csv`
  (`data/titles/{slug}/reference/map_geometry/`) ne sont produits par AUCUN outil — vérifié :
  zéro producteur dans `cmd/` — ils ont été importés à la main et n'ont pas de commande de
  rejeu.
- Prérequis : jeu installé NON requis ; réseau requis sauf `--from-file`/`--refresh-from`
  (authentification Xbox Live/Halo selon l'ADR 0023 — jamais de re-capture de jeton) ;
  `--all` ouvre en plus `shared_matches_v2.duckdb` en lecture seule ; cgo nécessaire pour
  compiler (driver DuckDB).
- À rejouer : une nouvelle carte est jouée en matchmaking (un `--map-id`) ; `--all` pour
  resynchroniser tout le registre ; `--refresh-from` après un dépôt local de `.mvar`,
  entièrement hors ligne.

#### mapopads-build

```bash
go run ./cmd/mapopads-build --from <dossier de .mvar> [--title slug] [--dry-run]
go run ./cmd/mapopads-build --from <dossier> --refresh-drifted   # re-valide contre des .mvar frais
```

- Sortie : `data/titles/{slug}/reference/map_weapon_pads.json` (socles d'arme et de
  power-up), écriture atomique via le même helper `mapcatalog.WriteAtomic` que le chemin de
  rattrapage Forge de la synchro écrit dans ce même fichier
  (`.ai/V7.5/v2/PLAN_V2_REJEU_FILM_2026-09-05.md` item A.3 — suivi séparément, hors de cette chaîne).
- Prérequis : ni jeu installé, ni réseau, ni cgo ; exige `map_objectives.json` (lien map_id ->
  nom de fichier) et un dépôt local de `.mvar` (`--from`).
- À rejouer : `--refresh-drifted` — le `.mvar` d'une carte UGC ne concorde plus avec le
  catalogue commité (dérive mesurée ; c'est la voie normale de re-validation depuis la
  décision du 2026-09-01).

#### mapstruct-build

```bash
CGO_ENABLED=1 go run ./cmd/mapstruct-build [--levels DIR] [--maps "Cliffhanger,Streets"] \
  [--title slug] [--out-dir DIR]
```

- Sortie : `data/titles/{slug}/reference/map_structure/{module}.json` (2 fichiers
  aujourd'hui — le `--maps` par défaut ne couvre que les deux cartes à 100 % de couverture
  mesurée, pas « toutes »).
- Prérequis : jeu installé (variante deploy `pc`, pas `ds`) sauf `--levels` ; cgo ; exige
  `map_quant_bounds.json` (lien module <-> nom affiché).
- À rejouer : quand le décodage des instances de maillage d'une autre carte atteint 100 % de
  couverture. **Avertissement** : le champ `structure` de l'artefact est sous une décision de
  retrait DIFFÉRÉ (`.ai/V7.5/REGISTRE_REPORTS.md`) — encore lu par deux fichiers web —
  vérifier cette entrée avant de supposer cet outil sans risque à supprimer.

#### mappos-build

```bash
go run ./cmd/mappos-build --cle <mapId> [--carte NOM] [--title slug] [--pas M] \
  [--min-matchs N] [--min-occurrences N] <rejeu.json>...
```

- Sortie : `data/titles/{slug}/reference/map_positions_jouees.json` (fusionne dans le
  catalogue existant, une clé de carte à la fois).
- Prérequis : ni jeu installé, ni cgo — post-traitement pur sur des artefacts de rejeu déjà
  décodés (`data/cache/replays/{title}/{matchId}.json`), passés en arguments positionnels.
- À rejouer : quand plus ou de plus récents matchs doivent affiner le masque de positions
  jouées d'une carte.

#### mapnav-fetch

```bash
go run ./cmd/mapnav-fetch -toutes [-out-dir DIR] [-rate-ms N] [-refaire]
go run ./cmd/mapnav-fetch -map-id <uuid> [-map-id <uuid>...] [-dry-run]
```

- Sortie : `<out-dir, defaut .ai/re_dump/navmesh>/<mapID>.blob` — **pas un asset versionné en
  soi** : `.ai/re_dump/` est ignoré par git. C'est le cache de travail local que la passe
  Forge de `mapfond-build` relit (`cuisson.go`) ; cité ici parce qu'il alimente une chaîne
  versionnée.
- Prérequis : PAS le jeu installé — une requête HTTP anonyme vers les pages UGC publiques de
  halowaypoint.com (deux requêtes, sans authentification) ; reprenable (saute les blobs déjà
  présents) et limité en débit.
- À rejouer : une nouvelle carte Forge a besoin de son navmesh avant que `mapfond-build` ne
  puisse cuire son fond ; `-refaire` force un nouveau téléchargement.

#### vehicle-sprite

CLI à sous-commandes (`inventaire`/`render`/`variantes`/`diag`/`assemble`/`compose2d`), pas
une invocation unique. Fragment vérifié de la recette derrière le jeu actuel (couvre 13 des
18 véhicules ; des passes ultérieures ont ajouté le reste — vérifier
`.ai/V7.5/film_re/*.md` pour l'état courant avant de rejouer) :

```bash
go build -o v4tool.exe ./cmd/vehicle-sprite
v4tool.exe render -variant=any -cote=256 -out=<dir> \
  -modules="pc:globals-rtx-new.module,globals-rtx-new.module,common-rtx-new.module,multiplayer-rtx-new.module" \
  -curate="0x00002705:warthog,0x000025aa:mongoose,0x0000d3db:scorpion,0xb65b3b4a:wasp"
```

- Sortie : `static/vehicles-assets/halo_infinite/replay/` — 38 fichiers (18 sprites + 18 `*_outline.png` +
  `index.json` + `files_list.txt`), consommés par `useReplayVehicles.ts`. Rien ne passe par
  le `PathResolver` — les chemins sont de simples flags `-out`/`-curate`.
- Prérequis : jeu installé, cgo/GPLv3 (jamais lié dans `cmd/server`) ; aucun réseau.
- À rejouer : un nouveau véhicule pilotable sort. Recette complète :
  `.ai/V7.5/film_re/V4_RAPPORT_SPRITES_2026-08-31.md` §9 et les notes suivantes du même
  dossier.

#### weapon-sounds (mode `livrer`, dernière étape d'une recette plus large)

```bash
go run ./cmd/weapon-sounds -mode livrer -donnees <chantier>/_donnees [-sons <chantier>] [-depot <depot>]
```

- Sortie : `static/sounds/halo_infinite/hinf_*.wav` (26 fichiers) +
  `apps/web/src/features/match-replay/weaponSoundVariations.ts`.
- Prérequis : les étapes antérieures de la recette, encore hors dépôt (extraction, analyse
  des banks, vote humain), doivent déjà avoir produit `_donnees/*.json` et l'arborescence de
  `.wav` sources/rendus par arme. Aucun jeu installé n'est nécessaire pour cette étape finale
  (le mode n'ouvre aucun module du jeu), mais cgo EST requis pour COMPILER le binaire :
  `cmd/weapon-sounds` importe `internal/himap` -> `internal/himodule` -> `internal/ooz`
  (décompression Kraken) pour ses autres modes.
- À rejouer : un vote d'arme est finalisé, ou la recette complète est rejouée (mise à jour du
  jeu, nouvelle arme). Recette complète : `.ai/V7.5/RECETTE_SONS_ARMES.md`.

### Médias

```bash
go run ./cmd/levelup index-media --gamertag X [--force-rescan] [--buffer-min N]
```

### Diagnostic & ops

```bash
go run ./cmd/levelup healthcheck [--verbose]
go run ./cmd/levelup diagnose --db PATH [--verbose]
go run ./cmd/levelup check-env
go run ./cmd/levelup gate-check [--gamertag X] [--json]
go run ./cmd/levelup compare-db --go-db PATH --python-db PATH [--json]
```

### Prestige — analyseur de tuning de la grammaire coach

Analyseur en LECTURE SEULE (jamais d'ouverture RW). Produit des **recommandations**
d'ajustement de la grammaire de synthèse du coach
(`config/coach_advisor/synthesis_grammar.toml`) à partir de la télémétrie Prestige
(taux de complétion par métrique de grammaire). L'application reste **manuelle** : un
humain lit le rapport et édite le TOML — aucune PR automatique, aucun override runtime.

```bash
# Tous les joueurs d'un titre (défaut halo_infinite), rapport texte :
go run ./cmd/prestige-tuning-analyze
# Un seul joueur, sortie JSON :
go run ./cmd/prestige-tuning-analyze --player JGtm --format json
# Seuils personnalisés (règle : complétion < min-completion sur >= min-sample défis coach acceptés) :
go run ./cmd/prestige-tuning-analyze --min-completion 0.30 --min-sample 50 --source coach
# flags : --format text|json  --player SLUG|GAMERTAG  --title SLUG
#         --min-completion 0..1  --min-sample N  --source coach|user|pilot_mode  --grammar PATH
```

Sous `--min-sample` : « données insuffisantes » (aucune reco sur du bruit). Une métrique
de télémétrie absente de la grammaire est signalée comme orpheline (dérive de nommage /
défi legacy).

### Maintenance (serveur arrêté pour les rebuilds ART/alias)

```bash
go run ./cmd/levelup rebuild-pme-art --all | --gamertag X   # reconstruit l'index ART player_match_enrichment
go run ./cmd/levelup consolidate-aliases                    # merge xbox_aliases dans shared.xuid_aliases
go run ./cmd/levelup recompute-friends [--dry-run]          # recompute is_with_friends, chaque joueur avec SES amis
go run ./cmd/levelup replay-events --gamertag X             # re-parse les highlight events
go run ./cmd/levelup reset-bitmasks                         # reset des bits de backfill skill/participants/PVE
go run ./cmd/levelup engagement-coefs [--with-scores]      # recompute des coefficients d'engagement
```

### Migration des chemins média (one-shot, binaire autonome)

Convertit les chemins média **absolus** (legacy) en chemins relatifs portables
`{owner_slug}/{rel}` dans `shared_social.duckdb` (`media_files.file_path` /
`thumbnail_path`, propagé à la PK `media_likes.media_path`). Idempotent — les chemins déjà
relatifs sont ignorés, une miniature cassée est mise à NULL pour que le prochain
`BackfillThumbnailPaths` la repointe. À lancer **serveur arrêté** (ouvre
`shared_social.duckdb` en RW). Déjà exécuté en prod pour les titres existants ; conservé
pour de futurs imports legacy qui réintroduiraient des chemins absolus.

```bash
go run ./cmd/migrate-media-paths --db data/titles/{slug}/warehouse/shared_social.duckdb [--dry-run]
# flags : --db PATH (requis)  --captures-base DIR  --settings app_settings.json  --dry-run
# --captures-base : défaut = media_captures_base_dir de app_settings.json
```

### Rejeu 2D — où se construit un artefact (`replay_build_location`)

Réglage d'`app_settings.json`, relu à **chaque** cycle de synchronisation (un
`PATCH /api/v1/settings` prend effet sans redémarrage). Il arbitre les chemins de *service* —
l'étape post-sync et l'action d'administration — jamais la commande d'opérateur ci-dessous.

| Valeur | Ce que fait le serveur | Quand elle s'applique |
|---|---|---|
| `local` | Ce processus décode le film lui-même, dans un **processus enfant borné** (plafond mémoire dur, priorité CPU basse). | Défaut en développement. **Refusée en production** : un décodage dure ~50 s et son pic mémoire vaut des centaines de fois la taille du film ; le VPS web ne décode jamais. Un `PATCH` qui la demande en production est refusé par un `400 invalid_replay_build_location`. |
| `worker` | Ce processus **met en file** et ne décode jamais. Un `cmd/replay-worker` distant prend le travail, télécharge les morceaux par URL pré-signées, décode, et repousse l'artefact. | Défaut en production. Exige `LEVELUP_BUILD_WORKER_TOKEN` sur l'instance web ; **sans lui le placement dégrade en `off`** (enfiler quand personne ne vide la file résoudrait un manifeste Halo par match, à chaque cycle, pour rien). |
| `off` | Aucune construction. La page de rejeu se contente des artefacts déjà présents. | Renoncement explicite. C'est le seul placement *silencieux* — les deux dégradations ci-dessus journalisent chacune un `WARN`. |

Valeur vide = défaut de l'instance (`worker` en production, `local` en développement). La
décision vit à un seul endroit : `replaybuild.DecidePlacement`.

L'étape post-sync (1.58) prend d'abord les matchs **insérés** du cycle, puis rattrape les
matchs les plus récents de la fenêtre de rétention qui n'ont pas encore d'artefact — le film
Theater se publie *après* la partie, et une tentative unique à l'instant de l'insertion ne
rattraperait jamais un film arrivé en retard. Plafonds : le rattrapage n'ajoute jamais plus de
5 matchs par cycle, une construction locale n'en traite jamais plus de 5, et l'un comme l'autre
s'arrêtent entre deux matchs dès que le cycle a consommé 5 minutes. Le retard restant est
publié en `postsync_replay_backlog_restant` sur `/debug/vars`, avec
`postsync_replay_cycles_total` (zéro alors que les synchronisations tournent = l'étape est
éteinte ou non câblée).

`replay_retention_months` borne cette même fenêtre : l'étape ne construit jamais — et la purge
récurrente supprime — les artefacts plus anciens. `0` = illimité.

La commande d'opérateur ignore volontairement ce réglage (cf. `cmd/levelup backfill-replay`) :
celui qui la tape a déjà décidé où il construit, sur sa machine, avec ses films en cache.

### Rejeu 2D — outillage de construction (faits, équivalence, profils)

Outils d'opérateur de la chaîne de construction des artefacts (« cuisson » dans le plan
`.ai/V7.5/PLAN_CUISSON_PERF.md`). Ils lisent le cache local de films ; les deux outils hors ligne
n'ont besoin d'aucune base et décodent un film par processus enfant borné (plafond mémoire dur,
priorité CPU basse, verrou solo).

```bash
cd apps/go-api
go run ./cmd/levelup replay-facts-export --out internal/games/halo_infinite/film/replay/testdata/equivalence \
  [--title slug] <short8|match_id>...
```

Écrit un `<short8>.facts.json` par match — lignes de match, scores des deux camps, variante,
identités de carte candidates — dans la forme que `replay-build --facts` lit déjà. Sans ces faits,
zones, actions d'objectif, VIP/crâne/bombe, socles et points d'apparition sont court-circuités et
une passe d'équivalence serait vacuante. Lecture seule (`OpenReadForQuery`) ; la commande échoue
franchement au lieu d'écrire des faits vides — arrêter un serveur qui tient la base partagée en
écriture.

```bash
go run ./cmd/replay-equiv                          # tout le corpus (CORPUS.txt), comparaison seule
go run ./cmd/replay-equiv -films 000d5950 -update  # (re)fige les références d'un seul film
# flags : -corpus F  -films a,b (remplace le corpus)  -update  -mem-gib N (défaut 3, 0 = désarmé)
#         -title slug  -out-dir D (conserve les TSV des enfants au lieu d'un temporaire effacé)
```

Le harnais d'équivalence de la construction : il hache la sortie de **chaque** balayage, pas
seulement l'artefact final, ce qui localise une divergence au balayage près. **Depuis le
2026-09-17, il nomme TOUTES les étapes divergentes d'un film, pas seulement la première** (D2),
avec le compte et le sha attendus / obtenus par étape et une ligne d'en-tête
`ECART sur N etape(s) sur M` : trois étapes divergentes valaient jusque-là trois décodages
complets (une à trois minutes chacun) pour les découvrir une à une, et une divergence locale ne
se distinguait pas d'une divergence générale. `-out-dir D` conserve les TSV des enfants au lieu
d'effacer un temporaire, pour comparer les digests obtenus aux références sans re-décoder.
`-update` et le format des TSV de référence sont inchangés. Parent et enfant vivent
dans le même binaire — le parent planifie et ne décode rien, chaque film naît dans un enfant borné
(verrou solo en attente bornée, sentinelle) et meurt avec sa RAM. Les références vivent dans
`internal/games/halo_infinite/film/replay/testdata/equivalence/<short8>.tsv`, chacune ouverte par son marqueur
`# digest-grammar: N` : une référence figée sous une autre grammaire est une panne
d'infrastructure (« re-figer par `-update` »), jamais un écart de décodage. `-update` réécrit ces
références au lieu de les comparer — pour une correction déclarée seulement. Le mode `-walkers`
(divergence des grammaires de découpage sur tout le cache de films) a été **retiré** en 2026-09 :
il portait en copie trois marcheurs de paquets historiques dont les originaux n'existent plus, il
ne se comparait donc plus qu'à lui-même. Sa mesure reste figée au §2 de
`.ai/V7.5/MESURES_CUISSON_PERF.md` et rejouée en CI par le test de la mini-bobine de
`internal/analysis/filmsource`.

```bash
LEVELUP_LOG_LEVEL=debug go run ./cmd/replay-build --map "<nom de carte>" --facts <f>.facts.json \
  --cpuprofile tmp/<f>.cpu.prof --memprofile tmp/<f>.heap.prof <short8> [dossierFilm]
```

Mesure d'une construction unitaire (protocole §6 du plan) : `LEVELUP_LOG_LEVEL=debug` fait
apparaître la durée de chaque balayage (le binaire installe un handler slog) ; `--cpuprofile` et
`--memprofile` écrivent des profils pprof (`go tool pprof`), celui du tas après la construction. Les
trois sont inertes par défaut, et les options doivent précéder `<matchId>` — le paquet flag arrête
l'analyse au premier argument positionnel.

#### Budget de temps du décodeur — bancs et `benchstat` (lot 0.A.5)

`replay-equiv` imprime déjà une durée **par film** (sa propre colonne) : c'est le budget de bout
en bout. Il ne dit pas **où** le temps est passé. Trois bancs isolent les étages que la révision
structurelle (M2) va déplacer, pour qu'un ralentissement se localise au lieu de se constater :
`BenchmarkBitReaderReadBits` (le primitif, sans grammaire), `BenchmarkTraverseEntity` (la boucle
de composants sur des records d'image-clé réels) et `BenchmarkKeyframeClosure` (le balayage chaud
d'une bobine entière).

Ils tournent sur la mini-bobine par build `minifilm_bcb6d393`, jamais sur `data/` : un banc qui
dépendrait du cache de films ne tournerait pas en CI. (`ScanBipedPositions`, que le plan nommait,
ne s'exécute PAS sur une mini-bobine — il dérive sa bande de slots bipède des images-clés et
refuse une bobine dont les images sont concaténées hors de leur continuité.)

```bash
cd apps/go-api
# la ligne de base commise (à régénérer seulement sur un changement déclaré)
go test -bench . -run '^$' -count 10 ./internal/games/halo_infinite/film/internal/grammar/ \
  > internal/games/halo_infinite/film/internal/grammar/testdata/bench_baseline.txt

# comparer après un changement, sur la MÉDIANE (ce que benchstat rapporte)
go test -bench . -run '^$' -count 10 ./internal/games/halo_infinite/film/internal/grammar/ > /tmp/apres.txt
benchstat internal/games/halo_infinite/film/internal/grammar/testdata/bench_baseline.txt /tmp/apres.txt
# benchstat n'est pas vendorisé : go install golang.org/x/perf/cmd/benchstat@latest
```

**Le budget de +10 % ne s'applique QU'AUX DEUX BANCS SERRÉS, `BitReaderReadBits` et
`TraverseEntity`.** Le premier a une médiane à 0,5 % de son minimum (un outlier isolé peut porter
son max à +56 % : c'est la MÉDIANE qui se lit, et c'est ce que `benchstat` compare) ; le second
disperse de +7 %.

`BenchmarkKeyframeClosure` est **informatif, pas un gate**. Mesuré SANS aucun changement de code :
71 % d'écart au sein d'une même passe, et +21 % de médiane d'une passe à l'autre sur le même
commit ; deux passes ici ont rendu +62 % et +5 % de dispersion. L'écart suit la charge de la
machine, pas le décodeur. Trancher un budget de +10 % dessus ferait rougir des lots innocents et
laisserait passer de vrais ralentissements — il sert à voir un ordre de grandeur bouger (un
facteur 2), rien de plus.

`-count 10` donne à `benchstat` une distribution et non un point unique ; `-run '^$'` tient les
tests hors du chronomètre.

### Notifications

```bash
go run ./cmd/levelup notify-version --version v1.2.3
go run ./cmd/levelup notify-sync --gamertag X --op sync_delta --duration 120s [--matches N]
```

Liste complète : `go run ./cmd/levelup help`.

---

## Tests

### Go (voir [../testing.md](../testing.md))

```bash
# Rapide, sans DuckDB (CGO off)
make go-api-test
# ou directement :
cd apps/go-api && CGO_ENABLED=0 go test ./internal/domain/... ./internal/analysis/... ./contracttest/... -count=1

# Suite complète avec DuckDB (CGO on — toolchain C / MinGW requis sur Windows)
cd apps/go-api && CGO_ENABLED=1 LEVELUP_DEMO_MODE=true go test ./... -timeout 5m -count=1

make go-api-coverage   # rapport de couverture
make go-api-lint       # go vet
```

#### Corpus du tag de build `gamefiles` (cartes **et** catalogues commis)

Deux familles de tests lisent le **jeu installé**, toutes deux derrière `//go:build gamefiles` :

- les 59 fichiers `*_gamefiles_test.go` de `internal/himap/` décodent les modules du jeu et
  balaient les 26 cartes du catalogue. Ils sont longs par nature — mesuré le 2026-09-05,
  `TestBalayageCoquille` prend à lui seul **203 s** pour 26 cartes (1 246 s avant le passage du
  lecteur de modules en projection mémoire, le même jour). Le tag garde un `go test
  ./internal/himap/` nu utilisable (2,8 s) ;
- trois fabricants de catalogues sous `cmd/` re-dérivent leur catalogue **commis** depuis le jeu
  installé et le comparent à l'octet : `cmd/film-profiles-build/`, `cmd/mapfond-build/`,
  `cmd/mapstruct-build/` (9,0 s à eux trois, mesuré le 2026-09-17).

`make go-api-test-gamefiles` joue **les quatre paquets**. Jusqu'au 2026-09-17 la cible ne jouait
que `./internal/himap/` : les trois tests `cmd/` n'étaient joués par AUCUNE commande du dépôt
(découverte D1 (3.1.2) ; le plus ancien est tagué depuis le 2026-09-05). La liste des paquets est
explicite et non `./cmd/...` : sous le tag, un paquet sans fichier `gamefiles` n'apporte que du
temps de compilation. `archlint.TestCibleMakefileGamefilesCouvreLeCorpus` rougit si un paquet
entre au corpus sans entrer dans la cible.

```bash
make go-api-test-gamefiles                       # corpus entier (~6 min, exige le jeu)
cd apps/go-api && CGO_ENABLED=1 go test -tags=gamefiles -count=1 -timeout 3600s \
  ./internal/himap/ \
  ./cmd/film-profiles-build/ ./cmd/mapfond-build/ ./cmd/mapstruct-build/ -v

# Les catalogues commis seuls (des secondes, pas des minutes) :
cd apps/go-api && CGO_ENABLED=1 go test -tags=gamefiles -count=1 \
  ./cmd/film-profiles-build/ ./cmd/mapfond-build/ ./cmd/mapstruct-build/ -v

# Une seule carte (beaucoup plus rapide) :
BALAYAGE_CARTES=aquarius_map go test -tags=gamefiles -timeout 300s \
  ./internal/himap/ -run TestBalayageCoquille -v

# Jeu installé ailleurs :
LEVELUP_HALO_DEPLOY=/chemin/vers/Halo Infinite/deploy go test -tags=gamefiles ./internal/himap/
```

Sans installation du jeu, chaque test prend son `t.Skip` et le corpus est vide en une
seconde — c'est exactement ce qui se passe en CI. La CI se contente donc de le **compiler**
(`go vet -tags=gamefiles ./internal/himap/`, job `go-test`) ; elle ne l'exécute jamais. Le
tag lui-même est tenu sur tout le module par `internal/archlint/gamefiles_tag_test.go`, qui
tourne dans le build par défaut.

**Test rouge connu** : `TestBancCliffhanger` échoue (accord 64,4 % contre une référence
re-basée à 64,7 %). Il est *préexistant*, pas une régression — vérifié le 2026-09-05 en le
rejouant sur le commit précédent, qui rend des chiffres identiques au bit près. Personne ne
pouvait le voir : le corpus ne terminait jamais, et la CI ne l’exécute pas. Consigné dans
`.ai/V7.5/REGISTRE_REPORTS.md`.

#### Gate de non-régression du rejeu sur corpus témoin (`cmd/replay-corpus-gate`)

Trois régressions de données (28/08, 30/08, 02/09 — pont d'identité par manche, actions de
drapeau non attribuées, « une piste = une vie ») ont traversé des goldens SYNTHÉTIQUES verts
pendant dix-neuf schémas, faute d'un différentiel sur des films réels. `cmd/replay-diff` (déjà
utilisé pour le balayage ponctuel du parc, `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`) devient ici
un gate répétable : `config/replay_corpus.toml` fige un film témoin par famille de mode (CTF
mono- et multi-manche, Oddball, Assaut, Slayer, un match à deux manches, un match riche en
véhicules), chacun choisi parce qu'il porte déjà un calque ou un défaut mesuré (portages de
crâne, pont d'identité multi-manche, forte occupation de véhicules...). Tous les axes que
connaît `cmd/replay-diff` sont comparés (dont l'axe « somme des durées par calque », qui
attrape un intervalle rogné qu'un simple compte d'éléments ne voit pas).

**Deux modes de référence** (décidé le 2026-09-06, après une première version qui comparait le
HEAD au parc de développement et rendait PERTE sur les 7 témoins au meilleur état connu — un
gate qui échoue toujours ne gate rien) :

- `--reference=base` (**défaut**) : cuit chaque témoin DEUX FOIS — une fois avec le code du
  HEAD, une fois avec le code d'une **révision de base** (défaut : `origin/feat/v75` si le
  HEAD en diffère, sinon `HEAD^` — un worktree détaché temporaire est créé pour la révision de
  base et retiré ensuite, même en échec) — puis compare les deux artefacts frais. Toute perte
  sort en code 1. C'est le gate à lancer avant tout merge : le signal est binaire, une perte ne
  peut venir que du diff en cours de revue, jamais de l'âge du parc.
- `--reference=parc` : compare le HEAD à l'artefact déjà cuit dans le parc local (méthode
  historique, balayage de release). **Informatif par défaut** (imprime le tableau, sort en 0) —
  `--strict` le rend aussi bloquant sur une perte.

Dans les deux modes, la racine de travail est jetable (entrées copiées, config/catalogues
depuis la branche extraite ou le worktree de base, chunks de film depuis le parc de
développement — **jamais d'écriture dans le parc**), et le gate ne bumpe jamais un schéma — il
ne fait que comparer.

```bash
make replay-corpus-gate                                      # defaut : mode base, manifeste complet
cd apps/go-api && go run ./cmd/replay-corpus-gate             # idem, toutes les options disponibles
cd apps/go-api && go run ./cmd/replay-corpus-gate \
  --reference=parc                                            # balayage informatif contre le parc
cd apps/go-api && go run ./cmd/replay-corpus-gate \
  --base=HEAD~3                                               # revision de base explicite
```

**Plancher de couverture (2026-09-07, CORPUS-R1 C3)** : par défaut, **tous** les témoins du
manifeste doivent être cuits et comparés — un cache de film purgé ou partiel rendait
auparavant tous les témoins ABSENT, et le gate sortait silencieusement en 0 sans rien comparer
(`codeSortie` saute les lignes ABSENT). Un ou plusieurs témoins ABSENT sortent désormais en
code 4 quand rien d'autre n'est à signaler, en nommant lesquels et pourquoi (voir la règle de
priorité ci-dessous) ; `--allow-missing` restaure l'ancien comportement (un
avertissement `slog` seul, jamais un échec) pour une exécution partielle délibérée.

**Statut par témoin, et sa règle de priorité (2026-09-17)** : un témoin porte UN SEUL statut,
dans la dernière colonne du tableau et dans le champ `statut` du JSON. La première règle qui
s'applique gagne :

| Statut | Signification |
|---|---|
| `ABSENT` / `ERREUR` | rien n'a été mesuré : film, faits ou artefact de référence manquants (`ABSENT`, avec sa cause), ou cuisson/comparaison en échec (`ERREUR`, avec sa cause). Exclusifs l'un de l'autre par construction. |
| `PERTE` | au moins une mesure a baissé ou disparu. **Prime sur `CHANGEMENT`** : un témoin qui porte les deux est un témoin en perte, et c'est la perte qu'on instruit. |
| `CHANGEMENT` | aucune perte, mais au moins une valeur publiée a BOUGÉ (réattribution, voie de nommage qui cède à une autre — `replaydiff/polarite.go`). Statut à lui depuis le 2026-09-17 : jusque-là un changement sortait `PERTE`, ce qui envoyait chercher une régression là où une valeur avait seulement changé de main. Il reste **bloquant** : un changement se justifie (divergence prouvée) ou il se corrige, jamais il ne se tait. |
| `ok` | ni perte ni changement. Des GAINS peuvent s'y trouver : un gain n'est jamais un échec. |

**Codes de sortie (constantes nommées, 2026-09-17)** : chacun dit UNE chose. Avant cette date,
le `2` disait à la fois « manifeste invalide » et « témoin absent » — un appelant ne pouvait
pas distinguer « ce gate n'a pas démarré » de « ce gate a démarré mais n'a pas tout comparé » —
et une erreur de cuisson se confondait avec une perte sous le `1`.

| Code | Constante | Signification |
|---|---|---|
| 0 | `codeOK` | tout le manifeste a été comparé, aucun témoin bloquant |
| 1 | `codePerte` | au moins un témoin comparé porte une `PERTE` ou un `CHANGEMENT` bloquant — le verdict de ce gate |
| 2 | `codeUsage` | le gate n'a pas DÉMARRÉ (drapeau invalide, manifeste illisible, racine ou capability absente, worktree de base impossible) ; rien n'a été mesuré du diff sous revue |
| 3 | `codeErreurCuisson` | le gate a démarré, mais un témoin CUIT a échoué à la cuisson ou à la comparaison — distinct du 1 : la question n'a pas pu être posée, la réponse n'est pas « il a perdu » |
| 4 | `codeCouvertureIncomplete` | au moins un témoin ABSENT sans `--allow-missing` (CORPUS-R1 C3), **et rien d'autre à signaler** — distinct du 1 ET du 2 : le manifeste est valide, tout témoin qui A ÉTÉ comparé est à zéro, il en manque |

**Le verdict des témoins présents prime sur la couverture (2026-09-16)** : la couverture était
vérifiée AVANT le verdict, si bien qu'un seul témoin ABSENT — l'aléa d'export des faits
ci-dessous, un cache de film partiel — faisait sortir le gate en 4 et MASQUAIT une perte, un
changement ou une erreur de cuisson sur tous les autres. Le gate tranche désormais d'abord sur
les témoins qu'il a comparés : une perte ou un changement sort en 1 et figure dans le tableau
et le JSON, l'avertissement de couverture étant journalisé en plus ; une erreur de cuisson sort
en 3 de la même façon. Le code 4 reste pour le seul cas où « il en manque » est tout ce qu'il y
a à dire. Le rapport JSON porte les deux informations — `couverture_incomplete` à la racine, à
côté des compteurs par témoin sous `temoins` — pour qu'un lecteur automatique ne confonde
jamais « tout est à zéro » avec « tout ce qui a été comparé est à zéro ». `--allow-missing` est
inchangé : il éteint la vérification de couverture, jamais le verdict.

**Export des faits robuste (2026-09-17, D2)** : `levelup replay-facts-export` ouvre la base
partagée en lecture seule, et échoue quand le serveur local la tient en écriture à cette
seconde-là (« `… serveur en ecriture ? reessayer` »). Au gate du lot 2.1, cela a coûté deux
témoins sur quatorze — `2/14 absent(s)` pour un aléa de quelques secondes, rejoué à la main
avec un manifeste réduit à ces deux-là. Le gate **réessaie désormais 3 fois, à 2 s d'écart**,
un échec qui porte le marqueur de base tenue, et ne réessaie JAMAIS un échec permanent (id
inconnu du registre, faits vides) : re-poser une question dont la réponse ne peut pas changer
ne fait qu'allonger un gate de 25 min. Un témoin toujours manquant ensuite sort `ABSENT` en
code 4 si les témoins comparés sont propres, distinct d'une perte ; `--temoins a,b` rejoue les
seuls concernés.

**Changements nommés dans le rapport JSON (2026-09-17, D5)** : le JSON porte désormais un
`changementsDetail` (axe, métrique, ancien, nouveau) symétrique de `pertesDetail`, plus un
`statut` et un `absentCause` sur chaque ligne, et le tableau imprimé gagne une section
`DETAIL DES CHANGEMENTS` à côté de `DETAIL DES PERTES`. Jusque-là le rapport disait
« 2 changements » sans jamais dire LESQUELS — la clôture M1 a dû relancer `replay-diff` à la
main sur les artefacts conservés pour les nommer — pendant qu'un témoin en ERREUR s'écrivait
`{"gains":0,"pertes":0,"changements":0}`, donc, lu du seul JSON, comme un témoin propre.

**La télémétrie et les compteurs de rejet ne sont pas des pertes (2026-09-17, lot 3.3.3)** : le
verdict comptait deux familles qu'il n'avait pas à compter. **La télémétrie** —
`coverage.decoder.{sourceRev, profileRev, grammarRev, factsRev}`, `coverage.decoder.build`,
`coverage.decoder.registry.fingerprint` — dit quelle VERSION du décodeur a cuit l'artefact, pas ce
que le match contient. Ces feuilles étaient NEUVES au lot 2.6, donc comptées en gains ; depuis le
schéma 61 elles sont partagées, si bien que tout lot qui fait monter une révision les faisait
« bouger » sur chaque témoin et le gate sortait 1 sans que rien d'autre n'ait changé (lot 3.3.2 :
51 des 59 changements étaient ces trois chaînes). Elles s'impriment désormais dans leur propre
section `TELEMETRIE` et ne comptent nulle part. **Les compteurs de rejet** — `noSlot`, `unread`,
`truncated`, `unnamedLives`… — sont déjà lus à l'envers par `replaydiff/polarite.go` (une baisse
est un gain) ; ce que cette couche ne voit pas, c'est leur DÉNOMINATEUR. Une hausse n'est une
`PERTE` que si le RAPPORT au dénominateur se dégrade, avec un dénominateur non nul des deux côtés ;
un compteur qui monte de zéro parce que le dénominateur passe de 0 à N est un `CHANGEMENT` —
imprimé, instruit par le pilote, toujours bloquant. La table des compteurs et de leurs
dénominateurs est `cmd/replay-corpus-gate/verdict_metriques.go`, une ligne par bloc de couverture,
chacune citant où le dénominateur a été lu. Le reste ne change pas : une perte réelle et un
changement hors télémétrie sortent tous deux en 1.

**Tous les drapeaux** (`cd apps/go-api && go run ./cmd/replay-corpus-gate -h` pour la liste à
jour) :

| Drapeau | Défaut | Signification |
|---|---|---|
| `--reference` | `base` | `base` (cuisson fraîche contre une révision de base) ou `parc` (contre l'artefact déjà cuit) |
| `--base` | auto (voir plus haut) | révision de base explicite, en mode `--reference=base` |
| `--strict` | `false` | en mode `--reference=parc`, une perte ou un changement sort aussi en code 1 (sans effet en mode base, déjà bloquant) |
| `--allow-missing` | `false` | tolérer un témoin ABSENT (avertissement seul) au lieu de sortir en code 4 |
| `--manifest` | `<source-root>/config/replay_corpus.toml` | chemin du manifeste |
| `--temoins` | (aucun) | rejouer les SEULS témoins nommés (ids séparés par des virgules) — le manifeste versionné reste le corpus, aucun manifeste réduit à écrire. Un id inconnu est une erreur (code 2), jamais une exécution tronquée en silence. |
| `--mem-gib` | `4` | plafond mémoire souple (Gio) armé sur CHAQUE cuisson enfant, HEAD et base (`0` désarme). Le défaut du gate est volontairement AU-DESSUS de celui de production (`filmproc.DefaultLimitGiB` = 3) : D6 a mesuré deux témoins BTB à 3,779 et 3,807 Gio côté base, soit juste au-dessus du plafond dur de 3,75 Gio — ils échouaient au hasard d'un run à l'autre. |
| `--source-root` | `git rev-parse --show-toplevel` | dépôt dont le code/la config AU HEAD est testé — **pas** basé sur `db_profiles.json` : fonctionne depuis n'importe quel worktree, y compris un sans copie locale de ce fichier |
| `--parc-root` | `source-root` s'il porte déjà la base partagée du titre, sinon auto-détecté via le `.git` commun | le parc de développement (chunks de film, artefacts `--reference=parc`) |
| `--lock-root` | `CacheRootDir()` du parc | où vit le verrou de décodage partagé |
| `--work-root` | un dossier temporaire jetable | racine de travail de la ou des cuissons fraîches |
| `--keep-work` | `false` | conserver la racine de travail après l'exécution (débogage) |
| `--json` | (aucun) | chemin où écrire aussi le rapport complet en JSON |

**À exécuter avant tout merge qui touche** `games/halo_infinite/film/replay`, `replaybuild`, `film/internal/grammar`, ou qui
bumpe `SchemaVersion`. **Exige** : le parc local de développement (chunks de film ; + artefacts
déjà cuits sous `data/cache/replays` en mode `--reference=parc`) et l'accès en lecture à la base
partagée du titre (pour les faits du match, via `levelup replay-facts-export` lancé en
sous-processus, PAR témoin — la seule étape qui exige CGO/gcc ; un témoin inconnu du registre ne
saute que lui, jamais tout le lot). **N'exige PAS le jeu installé** : la cuisson elle-même (un
binaire `cmd/replay-build`, compilé à la volée pour le HEAD et, en mode base, pour la révision
de base) ne lit que des catalogues versionnés (`data/titles/{slug}/reference`), contrairement au
corpus `gamefiles` ci-dessus — le gate n'en partage que l'ESPRIT (une ressource locale
volumineuse, absente en CI, qui dégrade proprement plutôt que d'échouer). Mesuré le
2026-09-06/07 sur le manifeste à 7 témoins, dans les deux modes : **cf.
`.ai/V7.5/v2/CORPUS_TEMOIN_2026-09-06.md`** pour l'exécution exacte et sa durée.

Si le parc local est plus ancien que le HEAD, les écarts attendus en `--reference=parc` sont des
GAINS (calques neufs, correctifs documentés) ; toute PERTE est un fait à rapporter, jamais à
masquer en resserrant le manifeste ou en filtrant le rapport.


### Frontend (`apps/web`)

```bash
make test-web        # vitest run
make test-e2e        # Playwright (nécessite `make dev` en cours)
make test-e2e-ui     # Playwright en mode UI
# ou via npm dans apps/web :
npm run test:run
npm run test:coverage
npm run lint
```

### Gate local avant merge (`gate-push`)

```bash
make gate-push               # ratchet lint Go + typecheck/lint web + baseline de tests (~25 min)
```

Sur certains postes Windows, l'environnement git-bash casse le lien des
binaires de test Go embarquant `libduckdb_static` (`undefined reference
__emutls_v._ZSt11__once_call`), ce qui fait échouer le maillon baseline de
tests de `make gate-push` alors que le code lui-même est sain — PowerShell
natif lie correctement. Contournement validé (documenté dans
`.ai/HANDOFF_POST_LOT2_V73.md`) : lancer `scripts/gate-push.ps1` à la place.
Il reproduit les 4 mêmes maillons (lint Go, tests Go d'intégration, typecheck
web, lint web) mais produit le JSONL `go test -json` depuis PowerShell natif,
puis le fait vérifier par `scripts/check_test_baseline.sh tests --from-jsonl
<fichier>` (mode consommateur — parse le JSONL, ne relance pas la suite). La
CI reste l'autorité ; ce script est un filet local propre à cette
particularité d'environnement.

```powershell
powershell -File scripts/gate-push.ps1
```

---

## Variables d'environnement

| Variable | Rôle |
|----------|------|
| `LEVELUP_REPO_ROOT` | Racine du repo de données (auto-détectée si absente) |
| `LEVELUP_API_PORT` | Port de l'API Go (défaut `8000`) |
| `LEVELUP_DEMO_MODE` | Mode démo (utilisé par les cibles de test) |
| `LEVELUP_NOTIFY_VERSIONS` | Mettre à `1` pour activer les notifs de version en prod |
| `DISCORD_WEBHOOK_URL` | Webhook Discord (prévaut sur `app_settings.json`) |
| `CGO_ENABLED` | Doit valoir `1` pour tout build/test touchant DuckDB |

---

## Chemins des données

```
data/
  warehouse/metadata.duckdb         # référentiels (maps, playlists, médailles)
  warehouse/shared_matches_v2.duckdb # matchs/médailles/events/aliases partagés
  warehouse/shared_pve.duckdb       # stats Firefight
  players/{gamertag}/stats.duckdb   # enrichissements par joueur
  players/{gamertag}/archive/       # archives Parquet
db_profiles.json                    # profils joueurs (multi-titres)
app_settings.json                   # paramètres app
.env.local                          # tokens Azure / secrets
```

Voir [ARCHITECTURE_V6.md](../ARCHITECTURE_V6.md) pour le modèle de données complet.
