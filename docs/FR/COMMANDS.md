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
```

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

### Référentiels / seed / migration

```bash
go run ./cmd/levelup seed career-ranks | citation-mappings | medals | rank-translations
go run ./cmd/levelup seed-demo            # génère les données démo anonymisées (data/demo/)
go run ./cmd/levelup migrate              # migre les données vers le namespace multi-titres
go run ./cmd/levelup add-title --name "Halo MCC" [--slug s] [--capabilities matchmaking,media] [--xbox-id X] [--steam-id S]
```

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
  (`.ai/PLAN_V2_REJEU_FILM_2026-09-05.md` item A.3 — suivi séparément, hors de cette chaîne).
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

- Sortie : `static/vehicles-assets/halo_infinite/replay/` — 20 fichiers (18 PNG +
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
  `.wav` sources/rendus par arme ; aucun cgo/jeu installé pour cette seule étape finale.
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
go run ./cmd/levelup recompute-friends [--dry-run]          # recompute is_with_friends sur les player DBs
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
go run ./cmd/levelup replay-facts-export --out internal/analysis/replay/testdata/equivalence \
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
#         -title slug
```

Le harnais d'équivalence de la construction : il hache la sortie de **chaque** balayage, pas
seulement l'artefact final, ce qui localise une divergence au balayage près. Parent et enfant vivent
dans le même binaire — le parent planifie et ne décode rien, chaque film naît dans un enfant borné
(verrou solo en attente bornée, sentinelle) et meurt avec sa RAM. Les références vivent dans
`internal/analysis/replay/testdata/equivalence/<short8>.tsv`, chacune ouverte par son marqueur
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

#### Corpus de rétro-ingénierie des cartes (tag de build `gamefiles`)

Les 59 fichiers `*_gamefiles_test.go` de `internal/himap/` décodent les modules du **jeu
installé** et balaient les 26 cartes du catalogue. Ils sont longs par nature — mesuré le
2026-09-05, `TestBalayageCoquille` prend à lui seul **203 s** pour 26 cartes (1 246 s avant
le passage du lecteur de modules en projection mémoire, le même jour).
Ils vivent derrière `//go:build gamefiles` pour qu'un `go test ./internal/himap/` nu reste
utilisable (2,8 s).

```bash
make go-api-test-gamefiles                       # corpus entier (~6 min, exige le jeu)
cd apps/go-api && go test -tags=gamefiles -count=1 -timeout 3600s ./internal/himap/ -v

# Une seule carte (beaucoup plus rapide) :
BALAYAGE_CARTES=aquarius_map go test -tags=gamefiles -timeout 300s \
  ./internal/himap/ -run TestBalayageCoquille -v

# Jeu installé ailleurs :
LEVELUP_HALO_DEPLOY=/chemin/vers/Halo Infinite/deploy go test -tags=gamefiles ./internal/himap/
```

Sans installation du jeu, chaque test prend son `t.Skip` et le corpus est vide en une
seconde — c'est exactement ce qui se passe en CI. La CI se contente donc de le **compiler**
(`go vet -tags=gamefiles ./internal/himap/`, job `go-test`) ; elle ne l'exécute jamais. Le
tag lui-même est tenu par `internal/himap/corpus_tag_test.go`, qui tourne dans le build par
défaut.

**Test rouge connu** : `TestBancCliffhanger` échoue (accord 64,4 % contre une référence
re-basée à 64,7 %). Il est *préexistant*, pas une régression — vérifié le 2026-09-05 en le
rejouant sur le commit précédent, qui rend des chiffres identiques au bit près. Personne ne
pouvait le voir : le corpus ne terminait jamais, et la CI ne l’exécute pas. Consigné dans
`.ai/V7.5/REGISTRE_REPORTS.md`.

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
