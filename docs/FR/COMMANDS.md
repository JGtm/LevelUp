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

Démarrage : le serveur Go écoute dès sa configuration chargée, avant d'ouvrir ses bases. Tant
qu'il n'est pas prêt, `/health` et `/api/*` répondent `503` avec le code `server_starting`
(l'étape en cours dans `details.step`) et la page affiche « Démarrage du serveur… » en
réinterrogeant chaque seconde. `make dev` et `make go-api-dev` considèrent un serveur en
démarrage comme déjà lancé.

Requêtes lentes : une requête d'API d'au moins `LEVELUP_SLOW_REQUEST_MS` millisecondes (défaut `1000`, lu
une fois au démarrage du serveur) est journalisée dans `logs/http.log` avec `slow: true`, au moins en INFO.
Si elle a chronométré des sections (pages Escouade, Synthèse, Sessions et Séries temporelles, filtres),
une ligne `http_timings` suit et montre où est passé le temps (`total_ms`, `sections` = jusqu'à 15
`nom=ms`, les plus longues d'abord, `calls`) ; si le niveau DEBUG est actif (`LEVELUP_LOGS_FILE_LEVEL=debug`),
toute requête qui a chronométré des sections en reçoit une.

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
comme tout le reste (D4, plan robustesse du sync). Le cron de personnalisation Spartan (toutes
les 8 h, premier passage au démarrage du serveur) préfère le jeton propre du joueur, qui ouvre
la vue privée `/customization/appearance` (403 pour un tiers, mesuré). Quand ce jeton est
inutilisable (jeton de rafraîchissement refusé, créneau malsain), le joueur est lu avec le jeton
de l'utilisateur de l'instance seulement : le compte dont le xuid est lié à un compte de rôle
`admin` parmi les comptes de l'instance (`data/auth/users.json`, lus par le store des comptes).
Le jeton d'un autre utilisateur ne porte jamais la lecture. Si ce jeton est lui aussi
inutilisable, ou les comptes illisibles, rien n'est lu ni écrit (journalisé, compté dans
`spartan_cron_no_bearer_total`). Avec le jeton admin, le client se replie sur la vue publique
`/customization?view=public`, qui porte le même emblème, le même fond et le même tag de service
pour n'importe quel joueur. La ligne est écrite dans la base du joueur lu ; aucun jeton n'est
capturé ni rafraîchi hors du pool.

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

`--title <slug>` (défaut : le titre par défaut) choisit le titre de chaque passe locale ci-dessus
(`--engagement-scores`, `--citations`, `--citations-recompute-all`, `--composite-only`, `--lusr`,
`--perf`, `--assists-model`, `--compare-formulas`) ; `--all` prend alors les profils de CE titre.
`--csr` / `--shared-csr` restent sur le titre par défaut et refusent tout autre. Une passe dont le
titre ne déclare pas la capability (citations sur Halo 5 : pas de `citations.engine`, citations
natives à la place) répond « non supporte pour le titre » et n'exécute rien. Les passes choisies
tournent toutes, dans l'ordre `--engagement-scores`, `--citations`, `--lusr`, `--perf`,
`--assists-model`, `--composite-only`, `--citations-recompute-all`. Le job HTTP prend le même choix
dans le champ `title` de `POST /backfill/start`.

```bash
go run ./cmd/levelup backfill --all --title halo_5 --engagement-scores --force
go run ./cmd/levelup backfill --all --title halo_5 --perf --force
go run ./cmd/levelup backfill --all --title halo_5 --assists-model --force
go run ./cmd/levelup backfill --all --title halo_5 --lusr
```

Un match est classé ssi le `RankRecap.PostMatchCsr.InitialMeasurementMatches` de sa charge skill est
> 0 (règle d'OpenSpartan Workshop ; un seul point de décision, `internal/sync/csr_writes.go`).
`backfill-is-ranked` répare `match_registry.is_ranked` depuis le CSR par match déjà écrit (faux/NULL
vers vrai seulement, ne déclasse jamais, idempotente). **Serveur arrêté**, `--dry-run` compris ;
à lancer dans cet ordre, car le LUSR exclut les matchs classés et l'engagement saute les matchs déjà
calculés et dérive son mode de `is_ranked` :
```bash
go run ./cmd/levelup backfill --all --shared-csr
go run ./cmd/levelup backfill --all --csr
go run ./cmd/levelup backfill-is-ranked --dry-run
go run ./cmd/levelup backfill-is-ranked
go run ./cmd/levelup backfill --all --lusr                       # recalcul LUSR v2 canonique
go run ./cmd/levelup backfill --all --engagement-scores --force  # le mode d'engagement suit is_ranked
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
Une exécution qui décode (ni `--dry-run` ni `--credit-only`) prend le verrou solo de décodage de sa racine de données en refus
immédiat, avant d'ouvrir la base : quand un autre décodage tient cette racine (un serveur qui cuit un rejeu, la file de
régénération des rejeux, une autre passe), elle nomme le détenteur et sort avec le code **3** (2 reste le code d'une ligne de
commande refusée). Le verrou est le fichier `<racine de données>/cache/film_decode.lock` : il est par racine de données, pas par
machine. Une commande lancée depuis le dépôt (`data/` de la copie de travail) et l'application installée (`%LOCALAPPDATA%\LevelUp`, ou
`LEVELUP_DATA_DIR`) tiennent deux verrous différents et ne se voient pas ; pointer `LEVELUP_DATA_DIR` sur la même racine pour n'en partager qu'un.

```bash
go run ./cmd/levelup backfill-killsource --dry-run      # bilan : total, déjà à jour, chunks, ETA
go run ./cmd/levelup backfill-killsource                # tout : films puis crédit, 3 ouvriers
go run ./cmd/levelup backfill-killsource --workers 1    # la boucle en série d'avant le lot 5.24
go run ./cmd/levelup backfill-killsource --limit 20     # les 20 films les moins chers
go run ./cmd/levelup backfill-killsource --credit-only  # la passe SQL → SQL seule
go run ./cmd/levelup backfill-killsource --force        # redécode même ce qui est à jour
go run ./cmd/levelup backfill-killsource --match 1c4c63c2,ee90570b --dry-run   # seulement ces matchs (préfixes de 8+ caractères, univoques)
go run ./cmd/levelup backfill-killsource --status       # DANS UN AUTRE TERMINAL : où elle en est
```

**`--match ID[,ID...]` — borne la passe hors ligne à des matchs nommés.** Identifiants séparés
par des virgules ; un préfixe court (8 caractères ou plus) est accepté s'il désigne UN seul match
du registre, et refusé avec une erreur claire s'il est ambigu (les candidats sont listés) ou
inconnu. La règle de fraîcheur ne change pas : un match nommé déjà à jour est sauté sauf
`--force` ; `--limit` s'applique après le filtre ; `--dry-run` liste chaque film retenu. Refusée
avec `--online` (cette passe choisit ses films dans l'historique du joueur, pas dans le registre)
et avec `--credit-only` (la passe SQL → SQL ne lit pas cette sélection).

**Tables écrites par la passe d'isolement.** Outre la source des frags, la passe des films écrit,
par match et en UNE transaction partageant une seule `decode_pass` : `match_lives`,
`match_death_context` et `match_death_mates` (l'état et la position de chaque coéquipier de la
victime à l'instant de chaque mort ; INSERT-only, lue par `match_death_mates_latest`). Un match
est périmé tant que son `IsolationDecoderRev` n'a pas atteint `isolement-2026-10-17-positions` (le bump du 2026-10-17 réécrit aussi `kill_positions` : joueurs renommés, tueurs posthumes, et un côté sans échantillon à l'instant, tenu à son dernier échantillon hors d'un véhicule ou posé au siège du véhicule qu'il occupe ; la victime de `match_death_context` est placée par la même lecture) :
le premier passage après ce bump redécode tout le parc (environ 1 600 films, reprise par
`decoder_rev`). La passe est atomique par match : si les lignes de coéquipiers d'un match
échouent à la validation, rien de ce match n'est publié pour la passe (`match_lives` et
`match_death_context` compris).

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

Deux passes lisent les artefacts de rejeu DÉJÀ cuits (`data/cache/replays/{slug}/{short8}.json.gz`, ou `.json` dans un parc pas encore converti)
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

# 4 ter. Ressource VEHICULES de l Emprise (prises, temps a bord, frags apparies aux
#    episodes du tueur) -> match_vehicle_takes (append-only). Meme motif que (4) : elle LIT
#    les artefacts TELS QU ILS SONT, sans decodage, SANS RECUISSON ; la seule lecture en
#    base est celle des frags du match dans match_kill_events_latest. Un artefact anterieur
#    au schema 67 (aucune occupation lue) s ecrit « non mesure », jamais zero : c est l etat
#    de tout le parc tant qu il n est pas recuit. LA REPRISE SE CLE SUR LA PRESENCE, donc
#    APRES UNE RECUISSON (ou l arrivee des evenements de mort) --force EST OBLIGATOIRE : la
#    ligne en base continuerait de dire « non mesure ». SERVEUR ARRETE, y compris pour
#    --dry-run (elle joue les migrations).
go run ./cmd/levelup backfill-vehicle-takes --dry-run
go run ./cmd/levelup backfill-vehicle-takes [--force] [--match ID[,ID...]] [--limit N] [--title S]
#    --match : identifiants de match séparés par des virgules, ou préfixes univoques de 8+
#    caractères, résolus contre le registre comme backfill-killsource ; inconnu / ambigu /
#    trop court = refusé, rien d écrit.

# 4 ter bis. MAINMISE (part du temps joué pendant laquelle un camp est seul en tête du score, et
#    nombre de retournements) -> match_lead_share (append-only). Même motif que (4 ter) : elle
#    LIT les artefacts TELS QU'ILS SONT, sans décodage, SANS RECUISSON, un artefact à la fois ; la
#    seule lecture en base est le mode et la durée du match au registre. Un match que la mesure
#    refuse (mode à peu de points : drapeau, colline, bombe ; score non couvert ; horloge sans
#    origine ; ...) s'écrit « non mesuré » avec sa raison, jamais zéro. LA REPRISE SE CLE SUR LA
#    PRÉSENCE, donc APRÈS UNE RECUISSON (ou un changement de règle) --force EST OBLIGATOIRE.
#    SERVEUR ARRÊTÉ. --dry-run ouvre la base partagée en LECTURE SEULE (rien d'écrit, aucune
#    migration, la table peut ne pas exister) : une ligne par match et un bilan avec le taux de
#    matchs mesurés et les refus par raison.
go run ./cmd/levelup backfill-lead-share --dry-run [--limit N]
go run ./cmd/levelup backfill-lead-share [--force] [--match ID[,ID...]] [--limit N] [--title S]

# 4 quater. Zones nommées (callouts) des cartes Forge jouées qui n'en ont pas -> le catalogue
#    GÉNÉRÉ reference/generated/map_callouts.json (ignoré par git ; le map_callouts.json
#    versionné n'est jamais écrit). La variante (.mvar) de chaque carte est lue au cache
#    (data/cache/mvar/<map_id>/), sinon téléchargée UNE fois par le pool de jetons de la CLI et
#    déposée au cache. AUCUNE RECUISSON : les zones se résolvent au service, à la lecture.
#    Idempotente (cartes couvertes sautées, ajout seul). Une ligne par carte et un bilan
#    chiffré, avec les string_id de lieu absents du lexique. La synchronisation fait de même
#    pour chaque nouvelle carte (étape post-sync, après le rattrapage des socles). Capability
#    map.forge_callouts. SERVEUR ARRÊTÉ (lecture du registre + pool de jetons).
go run ./cmd/levelup backfill-map-callouts --dry-run
go run ./cmd/levelup backfill-map-callouts [--hors-ligne] [--carte ID[,ID...]] [--rps N] [--cache-dir D] [--title S]

# 5. Rasters d'occupation tactique -> fichiers sidecar JSON sous
#    data/cache/replays/{slug}/rasters/. AUCUNE base n'est ouverte, pas même en lecture :
#    le sidecar est par match et anonyme, il n'y a rien à demander à DuckDB.
go run ./cmd/levelup tactical-rasters --backfill [--dry-run] [--limit N] [--title S]
```

`backfill-replay` ne cuit jamais un match d'un mode que le rejeu ne gère pas (Firefight PvE —
décision du 2026-10-15 ; le verdict est le drapeau du registre `match_registry.is_firefight`,
lu via `analysis.SQLModeNonGere`, jamais un nom de mode ni un slug). Ces matchs sont comptés
« mode non gere » dans le résumé, sous toutes les formes de la passe (`--force`,
`--only-existing`, `--repair-impoverished`) ; `--one <id>` sur l'un d'eux est refusé avec une
erreur explicite. L'étape post-sync, les actions admin de construction / mise en file et le
choix des rejeux de la démo appliquent la même exclusion, et le service de rejeu répond « pas de
rejeu » (404 `replay_not_available`) pour ces matchs même si un artefact traîne encore dans le
cache. Pour retirer les artefacts cuits avant cette décision (artefact, marque de dérivation et
sidecar de raster ; le cache de films n'est jamais touché) :

```bash
go run ./cmd/levelup backfill-replay --purge-non-geres --dry-run   # liste, ne supprime rien
go run ./cmd/levelup backfill-replay --purge-non-geres             # supprime
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

**Résumés par match du journal des morts** (`match_kill_heads`, `match_kill_pairs`,
`match_assist_pairs`, ADR 0039). Chaque passe écrite par la synchronisation porte ses résumés ;
cette commande les écrit pour les passes qui n'en ont pas (écrites avant les résumés) ou dont la
tête date d'une ancienne révision de la règle. Elle ne décode aucun film : elle relit la passe
courante de chaque match dans `match_kill_events_latest`. **Serveur arrêté** (elle ouvre la base
partagée en `OpenReadWrite` et joue les migrations partagées, `--dry-run` compris ; si un autre
processus tient la base, elle refuse et le dit). Reprenable : la liste est une anti-jointure
« passe courante sans tête à la révision courante », donc un second passage ne trouve rien. Elle
journalise une ligne de progression par lot et se termine sur le nombre de matchs restant à faire,
qui doit valoir 0 (le contrôle de santé des données publie la même jauge,
`levelup.kill_summaries_pending`).

```bash
go run ./cmd/levelup backfill-match-summaries --title halo_infinite --dry-run   # compte, n'écrit rien
go run ./cmd/levelup backfill-match-summaries --title halo_infinite [--batch N]  # lot par défaut : 250 matchs
go run ./cmd/levelup backfill-match-summaries --title halo_5
```

### Backup / restore

```bash
go run ./cmd/levelup backup  --gamertag X [--output-dir D] [--compression-level 9]
go run ./cmd/levelup restore --gamertag X --backup-dir D [--replace] [--dry-run] [--tables T1,T2]
go run ./cmd/levelup restore-csr --gamertag X --backup PATH [--dry-run]
go run ./cmd/levelup backup restore --from D:\Sauvegardes\levelup-20261103-0930 --to D:\DonneesLevelUp [--dry-run] [--force]
levelup-server.exe -mode restore --from D:\Sauvegardes\levelup-20261103-0930 --to "%LOCALAPPDATA%\LevelUp" [--dry-run] [--force]
```

`backup restore` restaure un jeu écrit par la sauvegarde périodique en Parquet de l'application de bureau (Réglages > Application,
`backup_dir` ; voir `docs/RUNBOOK_DESKTOP.md`). **`levelup-server.exe -mode restore` est la même commande** (mêmes drapeaux, même fonction) :
l'installeur livre `levelup-server.exe` et non `levelup.exe`, un PC neuf restaure donc avec le serveur qu'il a déjà (aucune configuration chargée,
aucun port lié, aucune base ouverte hors de la restauration) ; `levelup backup restore` en est l'alias depuis un dépôt. **Serveur arrêté** : la
commande prend le verrou d'instance de la racine de données cible (`--to`, le dossier qui contient `titles/` et `auth/`) et refuse tant qu'il est
tenu ; **tout serveur, quel que soit son profil, le tient pendant qu'il tourne, et `--force` ne le contourne jamais** (il autorise seulement le
remplacement de bases qui existent). Un `<base>.wal` résiduel dans la cible est refusé sans `--force` (lancer l'application une fois pour un arrêt
propre, ou supprimer le WAL si perdre les dernières écritures est acceptable) ; un jeu dont le dossier `db/` contient un fichier que le manifeste
ne déclare pas est refusé. Elle vérifie d'abord chaque somme SHA-256 du manifeste, recrée
chaque base depuis ses Parquet dans l'état de son époque (le démarrage suivant applique les migrations manquantes), puis remet en place les
fichiers de configuration et d'authentification. Une cible qui contient déjà une base du jeu est refusée sans `--force` ; `--dry-run` vérifie le jeu
et liste ce qui serait restauré, sans rien écrire. Les films archivés ne font pas partie d'un jeu.

```bash
levelup-server.exe -mode uninstall-cleanup      # appelé par le désinstalleur ; aucun drapeau
```

`-mode uninstall-cleanup` retire ce que l'application a posé HORS de son dossier, puis sort : l'entrée de démarrage de l'utilisateur (la valeur `LevelUp` de
`HKCU\...\Run` et sa valeur `StartupApproved`, seulement si sa commande vise le dossier du programme désinstallé) et la tâche planifiée
`LevelUp\Refresh-<8 hex>` de la racine de données de `LEVELUP_DATA_DIR` (seulement si son action lance le serveur de ce même dossier : deux installations
d'un même utilisateur partagent la racine par défaut, donc le nom de la tâche, et la tâche de l'autre reste en place, `tache=foreign`). Il n'ouvre aucune
base, ne lie aucun port, ne prend aucun verrou et ne touche jamais à la racine de données. Il retire aussi le paquet de ressources installé et les dossiers `web` et `config` du moteur (ces
derniers seulement quand `web/index.html` existe, et en entier : le front nomme ses fichiers par un hachage de leur contenu, donc après une mise à jour le
désinstalleur ne liste plus les anciens ; un fichier posé à la racine du dossier du programme reste). Sortie 0 quand rien n'existait ou que tout était étranger, 1 quand
une lecture ou une suppression a échoué (le crochet NSIS ignore le code : la désinstallation n'en dépend jamais) ; le bilan va sur stderr
(`demarrage=… tache=… ressources=… moteur=… ok=… exit=…`). `LEVELUP_UNINSTALL_WITNESS` posée à un chemin écrit aussi la ligne de bilan dans un fichier (ce que vérifie
`e2e-install.ps1`, code de sortie compris). Le `NSIS_HOOK_PREUNINSTALL` de l'installeur vérifie d'abord que l'application ne tourne pas (« Annuler » à
« LevelUp est en cours d'exécution » ne retire rien), puis le lance avant l'effacement des fichiers. Il ne tourne pas dans le flux de l'updater
(`/UPDATE` : l'updater n'appelle jamais l'ancien désinstalleur), mais une mise à jour MANUELLE (installeur lancé à la main, case « désinstaller avant »
cochée par défaut) ou une réinstallation passive (`/P`) appelle l'ancien désinstalleur sans `/UPDATE` : l'entrée et la tâche sont alors retirées, puis
reposées au prochain lancement si le réglage est actif. Après le nettoyage, le modèle de Tauri supprime la valeur `Run` du produit sans la regarder ; le
crochet la lit avant et la réécrit quand elle ne visait pas le dossier désinstallé.

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

### Mise en commun des jetons (consentement — ADR 0038)

Le jeton d'un joueur n'entre dans le pool partagé qui sert la synchro de tous que si ce joueur
l'a décidé (Réglages, onglet Synchronisation ; question posée à la première connexion Xbox).
Aucune décision enregistrée vaut refus. `pool-sharing list` montre où en est chaque joueur ;
`pool-sharing grandfather` enregistre l'accord de ceux qui l'ont donné hors de l'application,
avant l'existence du réglage.

```bash
go run ./cmd/levelup pool-sharing list                      # état par joueur : granted / declined / undecided
go run ./cmd/levelup pool-sharing grandfather --players GT1,GT2        # SIMULATION : liste ceux qui seraient marqués
go run ./cmd/levelup pool-sharing grandfather --players GT1,GT2 --yes  # enregistre l'accord (base grandfathered_explicit)
go run ./cmd/levelup pool-sharing grandfather --all [--yes]            # tous les joueurs sans décision : seulement si TOUS ont accepté
```

- **Ne lancer `grandfather` que pour des accords réellement recueillis** : la commande écrit un
  consentement au nom des joueurs. Elle n'écrase jamais une décision existante, refus compris.
- Le serveur en marche applique tout de suite une décision prise dans l'application ; une
  décision écrite par la CLI est reprise au prochain scan du pool (ou au redémarrage).
- **En production**, la CLI est dans l'image : la lancer dans le conteneur du serveur, par ex.
  `docker compose exec -T -u appuser levelup levelup pool-sharing list` (depuis `/opt/levelup`). Elle ne lit
  et n'écrit que `data/auth/watcher_tokens/`, jamais une base DuckDB : le serveur peut rester démarré.
  La lancer en `appuser` (`-u appuser`) : le conteneur durci retire à root les capacités qui lui
  permettaient d'écrire les fichiers de l'application, et la commande échoue sinon en `permission denied`.

### Archive des films (recompression et déplacement — ADR 0045 D-12, décision D-18)

L'archive contient chaque film téléchargé (`film_chunks/` et `film_manifests/`, sous
`film_archive_dir` ou `<données>/cache` par défaut). Un film expire côté Halo et ne se retélécharge
jamais : les deux opérations ci-dessous sont **à lancer soi-même, serveur arrêté** ; rien ne les
lance seul (ni cron, ni tâche au démarrage).

Les nouveaux chunks s'écrivent en zstd niveau 3 (environ 75 % plus petits). Les chunks bruts et zlib
existants se recompressent par `film recompress`, un film à la fois : chaque chunk est compressé,
redécodé et comparé à l'original avant que le fichier ne soit remplacé atomiquement. Un chunk déjà en
zstd est sauté : une passe interrompue reprend en relançant la commande. Les lecteurs acceptent les
chunks bruts, zlib et zstd à leur en-tête, et les faits décodés d'un film ne dépendent pas de la forme
de stockage.

```bash
go run ./cmd/levelup film recompress --dry-run               # lit et compresse en mémoire, affiche le gain, n'écrit rien
go run ./cmd/levelup film recompress                         # convertit toute l'archive
go run ./cmd/levelup film recompress --films 000d5950,01e1f945   # seulement ces films (courts)
go run ./cmd/levelup film archive-move --to D:\Films          # range l'archive dans un dossier vide, puis enregistre film_archive_dir
go run ./cmd/levelup film archive-move --default             # la ramène sous <données>/cache
go run ./cmd/levelup film archive-move --to D:\Films --resume   # reprend un déplacement interrompu
```

- L'écran des réglages (Réglages, onglet Données) affiche la taille de l'archive et l'espace libre, et
  l'assistant de premier lancement propose de conserver les films ou non. Y changer le dossier est
  refusé tant que l'archive contient des films ou que la cible n'est pas vide ; `archive-move` fait le
  déplacement (serveur arrêté ; le dossier enregistré sert au démarrage suivant). Sur une archive vide
  (assistant de premier lancement), le nouveau dossier sert tout de suite.
- Sans archive (`archive_films` désactivé), les rejeux sont construits et les faits persistés, les
  chunks sont retirés une fois cuits, et la rétention des rejeux doit rester illimitée (la purge suppose
  une reconstruction depuis l'archive).
- Pour mesurer le gain sur son poste : `du -sh data/cache/film_chunks` avant et après `film recompress`.

### Artefacts de rejeu (compression)

Un artefact de rejeu se range `data/cache/replays/<titre>/<short8>.json.gz` (gzip, niveau 6 de la
bibliothèque standard Go). Son contenu décompressé est le document de rejeu, inchangé d'un octet : toute
lecture et toute écriture des octets d'un artefact passent par une porte unique (`internal/replaybuild`),
qui reconnaît la forme aux octets magiques de gzip, pas au nom. Un parc ancien porte des `<short8>.json`
bruts et reste lisible ; un match qui a les deux formes est lu dans le `.json.gz` (un avertissement et un
compteur le signalent). L'en-tête `.meta` et la marque `.derived.json` gardent leurs noms, et leur `bytes`
est la taille stockée (celle du fichier sur disque).

`replay compress` convertit un parc brut sur place. Il est **à lancer soi-même** (rien ne le lance seul :
ni cron, ni tâche au démarrage). Chaque artefact est écrit en `.json.gz` atomiquement, relu et comparé au
JSON d'origine (identique, octet pour octet) avant que le `.json` ne soit supprimé ; son `.meta` est
réécrit, et une marque de dérivation à jour suit la nouvelle taille stockée : aucune dérivation n'est
rejouée. Un fichier en échec est compté, journalisé et laissé en place. La commande est idempotente (un
artefact compressé est sauté) et rend le compte : convertis, déjà compressés, doublons (les deux formes,
laissées telles quelles), échecs, octets avant et après.

```bash
go run ./cmd/levelup replay compress --dry-run          # lit et compresse en mémoire, affiche le gain, n'écrit rien
go run ./cmd/levelup replay compress                    # convertit le parc du titre par défaut
go run ./cmd/levelup replay compress --title halo_5     # un autre titre
go run ./cmd/levelup replay compress --dir D:\copie\parc # un dossier d'artefacts explicite
```

### Référentiels / seed / migration

```bash
go run ./cmd/levelup seed career-ranks | citation-mappings | medals | rank-translations
go run ./cmd/levelup seed-demo            # génère les données démo anonymisées (data/demo/), rejeux figés compris
go run ./cmd/levelup seed-demo --emit-replay-picks   # choisit un rejeu par mode de jeu dans config/demo/<gt>/<slug>.json (relire, puis committer)
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

### Langues et traductions (ADR 0040)

Les langues de l'application sont listées dans `config/locales.toml` (source unique). Ajouter une langue,
pas à pas : [Guide des fondations § 3.5](FOUNDATIONS_GUIDE.md).

```bash
# régénère apps/web/src/lib/i18n/generated/locales.ts, apps/go-api/internal/i18n/locales_gen.go et les
# modules de manifests depuis config/locales.toml ; s'arrête à la première feuille de manifest qui n'a pas
# une langue listée
node apps/web/scripts/build_i18n_manifests.mjs
# langue de preuve : fabrique `xx` depuis `en` (texte entre ⟦ et ⟧) dans --out (un dossier temporaire par
# défaut) ; ne touche ni config/locales.toml ni aucun fichier versionné
node apps/web/scripts/build_i18n_manifests.mjs --pseudo xx [--out DOSSIER]
# les types TypeScript générés ET les listes de langues générées sont à jour (aussi lancé par les tests web)
node tools/check-generated-types-fresh.mjs
# noms Halo des cartes, playlists, paires et variantes dans asset_translations (API Discovery UGC)
cd apps/go-api
go run ./cmd/levelup populate-assets [--types map,playlist,pair,game_variant] [--langs es-ES,es-MX] \
    [--dry-run] [--force] [--concurrency N] [--freshness JOURS] [--title-id halo_infinite]
```

Défauts de `populate-assets` : les quatre types, les 14 étiquettes de `domain.TargetLanguages`, concurrence
10, fraîcheur 30 jours ; `--langs` ignore (avec un avertissement) une étiquette hors de cette liste.

### Chaînes de fabrication des assets versionnés

Onze chaînes hors ligne, toutes sous `apps/go-api/cmd/`, produisent des fichiers commités
(`data/titles/{slug}/reference/`, `static/`, ou un fichier Go généré). Aucune n'est câblée
dans `cmd/server` — le décodage du jeu et le code GPLv3 (le paquet `himap` de `halo-infinite-map-images`, le paquet `ooz` de `halo-infinite-files-reader`,
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
  `.ai/archive/V7.5/icones/ETAT_DE_L_ART_ICONES.md`.

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
  mise à jour du jeu » selon son propre en-tête) ; une carte Forge relue en revue est à
  promouvoir au catalogue versionné (`--forge-fetch`). Une NOUVELLE carte Forge n'en a plus
  besoin : la synchronisation et `levelup backfill-map-callouts` rangent ses callouts au
  catalogue généré.

#### mapfond-build

```bash
CGO_ENABLED=1 go run ./cmd/mapfond-build [--maps "Cliffhanger,Catalyst"] [--title slug] \
  [--out-dir DIR] [--style jeu] [--natives=false] [--forge=false] [--rapport FILE]
```

- Sortie : `data/titles/{slug}/reference/map_backgrounds/{cle}.png` + `{cle}.json` (sidecar de
  calage) par carte — 218 fichiers aujourd'hui.
- Maillage de navigation : une carte native est cuite avec le maillage de navigation de son
  module `any` (`himap.ChargeMaillageNatif`) : la matière hors du maillage, dilaté de 1,5 m, est
  effacée, et le sidecar publie un `plan` (les arêtes de bord du maillage, en mètres monde, au
  millimètre). Un module sans maillage (Live Fire, le tutoriel de l'Académie) est cuit comme avant
  et porte « aucun maillage de navigation dans le module any » dans `degradations`, sans `plan`.
  Leviers par carte de `map_fond_reglages.json` (chaque entrée garde sa `raison` et son `gateLe`) :
  `rogneAuNavmesh` (sur une carte native : actif par défaut, `false` garde le maillage comme
  surface de référence sans rogner ; sur une carte Forge : inactif par défaut), `margeNavmesh`
  (dilatation, en mètres), `toleranceNavmesh`, `maillageNiveauHaut` ; `sansNavmesh: true` (cartes
  natives) retire le maillage entier (ni référence, ni rognage, ni plan). Une carte Forge reçoit
  aussi `navmeshReference`, et publie un `plan` dès que le dépôt porte son `navmesh.blob`
  (`mapnav-fetch`), que la cuisson s'en serve ou non.
- Après la cuisson, lancer `go run ./cmd/mapfond-webp -convertir` : il réécrit chaque sidecar par
  `internal/mapfondsidecar`, qui conserve le `plan` (le `replay.MapBackground` du décodeur de film
  ne le connaît pas).
- Prérequis : jeu installé — TOUJOURS, aucun flag ne permet de l'éviter, même en Forge seul ;
  chaîne cgo/GPLv3 (`halo-infinite-map-images/himap` -> `halo-infinite-files-reader/himodule` -> `halo-infinite-files-reader/ooz`, jamais liée
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
  (`.ai/archive/V7.5/v2/PLAN_V2_REJEU_FILM_2026-09-05.md` item A.3 — suivi séparément, hors de cette chaîne).
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
  retrait DIFFÉRÉ (`.ai/REGISTRE_REPORTS.md`) — encore lu par deux fichiers web —
  vérifier cette entrée avant de supposer cet outil sans risque à supprimer.

#### mappos-build

```bash
go run ./cmd/mappos-build --cle <mapId> [--carte NOM] [--title slug] [--pas M] \
  [--min-matchs N] [--min-occurrences N] <rejeu.json>...
```

- Sortie : `data/titles/{slug}/reference/map_positions_jouees.json` (fusionne dans le
  catalogue existant, une clé de carte à la fois).
- Prérequis : ni jeu installé, ni cgo — post-traitement pur sur des artefacts de rejeu déjà
  décodés (`data/cache/replays/{title}/{matchId}.json.gz` ou `.json`), passés en arguments positionnels.
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
`.ai/archive/V7.5/film_re/*.md` pour l'état courant avant de rejouer) :

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
  `.ai/archive/V7.5/film_re/V4_RAPPORT_SPRITES_2026-08-31.md` §9 et les notes suivantes du même
  dossier.

#### weapon-sounds (mode `livrer`, dernière étape d'une recette plus large)

```bash
go run ./cmd/weapon-sounds -mode livrer -donnees <chantier>/_donnees [-sons <chantier>] [-depot <depot>]
```

- Sortie : `static/sounds/halo_infinite/hinf_*.ogg` (26 fichiers, Opus 96 kbit/s en Ogg, encodés à la livraison par
  le même code que `sounds-encode` ; mixage et troncature restent en PCM) +
  `apps/web/src/features/match-replay/weaponSoundVariations.ts`.
- Prérequis : les étapes antérieures de la recette, encore hors dépôt (extraction, analyse
  des banks, vote humain), doivent déjà avoir produit `_donnees/*.json` et l'arborescence de
  `.wav` sources/rendus par arme. Aucun jeu installé n'est nécessaire pour cette étape finale
  (le mode n'ouvre aucun module du jeu), mais cgo EST requis pour COMPILER le binaire :
  `cmd/weapon-sounds` importe `halo-infinite-map-images/himap` -> `halo-infinite-files-reader/himodule` -> `halo-infinite-files-reader/ooz`
  (décompression Kraken) pour ses autres modes.
- À rejouer : un vote d'arme est finalisé, ou la recette complète est rejouée (mise à jour du
  jeu, nouvelle arme). Recette complète : `.ai/archive/V7.5/RECETTE_SONS_ARMES.md`.

#### sounds-encode (sons du rejeu, WAV -> Opus)

```bash
go run ./cmd/sounds-encode [-src static/sounds] [-dst <src>] [-check] [-remove-sources]
```

- Convertit chaque `.wav` sous `-src` en Opus 96 kbit/s dans un conteneur Ogg (`.ogg`, même arborescence ; nombre de
  canaux conservé). ffmpeg et ffprobe sont requis (résolus par `internal/media` : `LEVELUP_FFMPEG_DIR`, puis le
  dossier de l'exécutable, puis le PATH). Idempotent : un `.ogg` au moins aussi récent que son `.wav` est sauté.
  Sortie reproductible.
- `-check` vérifie, pour chaque son (ceux déjà à jour compris), que le `.ogg` se décode en exactement le nombre
  d'échantillons du `.wav` ; `-remove-sources` supprime chaque `.wav` dont le `.ogg` est en place (et contrôlé avec
  `-check`). Code de sortie 1 si un son a échoué (chaque échec est journalisé).
- À utiliser quand : un son est déposé à la main (véhicules, fin de partie...). L'arbre ne garde jamais de `.wav`
  (`replaySoundAssets.guard.test.ts`) : lancer `-check -remove-sources`.

### Médias

```bash
go run ./cmd/levelup index-media --gamertag X [--force-rescan] [--buffer-min N]
go run ./cmd/levelup media-capture-times --dry-run [--title slug] [--db CHEMIN]
go run ./cmd/levelup media-capture-times [--title slug] [--db CHEMIN]
```

L'heure lue pour un média (nom du fichier, sinon date du fichier) est le DÉBUT ou la FIN de la
capture selon sa source (table `captureTimeSources`, `internal/ops/media_capture_source.go`) :
un « Replay » d'OBS, un « record that » de la Game Bar ou de la console Xbox, un Instant Replay
de ShadowPlay et une date de fichier sont des FINS ; le nom d'un enregistrement manuel OBS ou
ShadowPlay est un DÉBUT. Les médias indexés avant cette règle sont recalculés par le serveur
lui-même à chaque indexation (envoi, balayage après synchronisation, bouton « Scanner » des
réglages). `media-capture-times` simule ce recalcul (`--dry-run` : comptes par source, sur une
COPIE de `shared_social.duckdb` si le serveur tourne) ou l'écrit serveur arrêté. Idempotent.

### Diagnostic & ops

```bash
go run ./cmd/levelup healthcheck [--verbose]
go run ./cmd/levelup diagnose --db PATH [--verbose]
go run ./cmd/levelup check-env
go run ./cmd/levelup gate-check [--gamertag X] [--json]
go run ./cmd/levelup compare-db --go-db PATH --python-db PATH [--json]
go run ./cmd/diag_q DB_PATH "SQL"                       # SQL en lecture seule, sortie TSV ; à lancer sur une COPIE d'une base tenue par le serveur
go run ./cmd/admin create-admin --username U --password P   # outil de secours des comptes, sans l'API web
go run ./cmd/admin reset-password --username U --password P
go run ./cmd/admin list-users
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

#### Compaction des passes de décodage supersédées (`compact-passes`)

Chaque redécodage d'un film AJOUTE une passe entière aux tables du film de la base partagée des
matchs (`match_kill_events`, `match_lives`, `match_death_context`, `match_death_mates`, `kill_openings`,
`kill_positions`, `match_weapon_shots`, `match_player_positions`, le résumé d'usage, les niveaux
de socle, les prises de drapeau, les statistiques d'Assaut — INSERT-only, ADR 0019 / 0026). Les
vues `_latest` ne servent que la dernière passe, mais toute lecture qui les évalue parcourt
toutes les passes : sur la base locale du 2026-09-26, 90 % de ces lignes étaient des passes
mortes. `compact-passes` reconstruit chaque table avec les seules lignes que sa vue retient —
jamais de `DELETE` (bug DuckDB ART #23645) : un échange transactionnel qui vérifie, avant le
COMMIT, le nombre de lignes, le DDL et les index de la table, et que la vue rend exactement le
même résultat. Les séquences ne sont pas touchées (aucun `id` n'est réutilisé). Idempotente : une
seconde passe rend `deja-compacte`.

Quand : **automatiquement chaque nuit en production** quand une table dépasse le seuil (ci-dessous),
et à la main pour compacter tout de suite, typiquement après une campagne de redécodage (recuisson,
`backfill-killsource`). Jamais au boot, ni après un sync, ni dans le processus serveur. **Serveur
arrêté** : la commande refuse une base qu'un autre processus tient.

**Compaction automatique de nuit (VPS).** Le contrôle de santé des données du serveur (au démarrage
puis toutes les 24 h) mesure pour chaque table du registre et chaque titre le rapport lignes
brutes / lignes gardées (une table compactée vaut 1) et l'écrit dans
`data/global/admin_state/passes_par_match.txt` (aussi en expvar :
`levelup.compaction.passes_par_match`, `levelup.compaction.tables_au_dessus_du_seuil`). À 03:30 UTC,
le timer systemd `levelup-compact-passes.timer` lance `scripts/compact-passes-nuit.sh` : si aucune
table d'aucun titre ne dépasse 1,5 (`migration.SeuilPassesParMatch`), il sort sans rien toucher ;
sinon il vérifie la place disque, arrête `levelup`, lance
`levelup compact-passes --backup-dir /app/data/compaction-backups`, redémarre `levelup` (même si la
compaction échoue, jusqu'à 3 essais) et attend `/health`. Journal : `data/logs/compact-passes-nuit.log`. Les
sauvegardes de plus de 24 h de `data/compaction-backups/` sont supprimées après un passage réussi
seulement. Installation des unités (une fois, root) : `scripts/systemd/README.md` ; décision et
mesures : ADR 0026, amendement « Automatic compaction ».

```bash
go run ./cmd/levelup compact-passes --dry-run                # lecture seule : brutes / _latest / gain par table
go run ./cmd/levelup compact-passes                          # sauvegarde, puis compaction, tous les titres
go run ./cmd/levelup compact-passes --title halo_5           # un seul titre
go run ./cmd/levelup compact-passes --rewrite-file           # + réécriture du fichier pour rendre la place au disque
go run ./cmd/levelup compact-passes --backup-dir D:\sauvegardes  # dossier des sauvegardes (défaut : celui de la base)
```

- **Sauvegarde (obligatoire)** : copie octet pour octet du fichier de la base, prise après un
  `CHECKPOINT`, fichier fermé, affichée `sauvegarde : <chemin>`
  (`<nom>.<slug>.avant-compaction-<UTC>.duckdb`). Restaurer = remettre ce fichier en place, serveur
  arrêté. L'outillage de sauvegarde Parquet (`backup`) n'est pas utilisé : il perd séquences,
  vues et index.
- **`--rewrite-file`** : DuckDB réutilise les blocs libérés mais ne rétrécit jamais un fichier.
  L'option recopie la base dans un fichier neuf (`COPY FROM DATABASE`), relit les DEUX fichiers
  seuls et ne les échange que si le catalogue (tables, vues, index, séquences avec leur prochaine
  valeur, macros, types), le compte de chaque table et l'empreinte de chaque vue compactée sont
  identiques. La commande TIENT le verrou DuckDB de la base du début à la fin (aucun autre
  processus ne peut l'ouvrir entre-temps) : une seule connexion fait le `CHECKPOINT`, la copie vers
  le fichier neuf et la sauvegarde de l'ancien (`<nom>.<slug>.avant-reecriture-<UTC>.duckdb`, écrite par
  `COPY FROM DATABASE` dans `--backup-dir`, n'importe quel volume — sous Windows, les octets d'un
  fichier tenu par DuckDB ne se lisent pas), et les deux copies sont vérifiées sous ce verrou.
  Puis UN rename met le fichier neuf en place — le chemin de la base porte toujours une base
  complète, quel que soit l'instant où la commande s'arrête. Sous Linux, le rename a lieu pendant
  que la connexion est encore ouverte ; sous Windows, qui refuse de remplacer un fichier tenu, la
  connexion est fermée et le rename suit aussitôt : un processus qui a ouvert la base entre les
  deux la tient encore, le rename échoue et la commande refuse, base intacte. Seule perte
  théorique : un processus qui ouvrirait, écrirait, ferait son checkpoint et fermerait ENTIÈREMENT
  en ces quelques microsecondes. Un `.wal` laissé au lancement par un processus
  tué avant son checkpoint est rejoué par la première ouverture de la commande (récupération
  normale de DuckDB, sans perte) et se retrouve dans la sauvegarde et la compaction ; l'étape de
  réécriture refuse, elle, un `.wal` non vide qui apparaîtrait entre la compaction et la
  réécriture. Les sauvegardes portent le slug du titre (`<nom>.<slug>.avant-…`) et un fichier
  existant n'est jamais écrasé ni retiré. Sur toute erreur, le
  fichier temporaire et la sauvegarde sont retirés. Mesuré sur une copie : 1 264 Mio -> 351 Mio.
- Supprimer les sauvegardes à la main une fois l'application vérifiée.

#### Réparation des temps forts en double (`repair-highlight-phantoms`)

Avant `grammar-2026-10-16`, le lecteur des temps forts pouvait décoder un même bloc de 60 octets
DEUX FOIS sous deux XUID (un faux motif de XUID dans la fiche d'un joueur, plusieurs milliers de
bits avant le vrai). Les deux lignes sont dans `highlight_events` (86 matchs sur la copie locale du
2026-10-15, un seul joueur touché), s'apparient à tort dans `killer_victim_pairs` (« X tue X ») et
arrivent dans les morts servies. Depuis la correction, aucune nouvelle ligne n'est doublée ; les
existantes se réparent par cette séquence exacte (**serveur arrêté**, prévenir avant de la jouer
en production) :

```bash
cd apps/go-api
go run ./cmd/levelup repair-highlight-phantoms --dry-run     # lecture seule : lignes fantômes et matchs touchés
go run ./cmd/levelup repair-highlight-phantoms               # sauvegarde, purge, recalcul des couples qui diffèrent
go run ./cmd/levelup backfill-killsource --credit-only      # reconstruit la base crédit (noms de l'annuaire)
go run ./cmd/levelup backfill-killsource                    # puis la passe film par-dessus
go run ./cmd/levelup compact-passes                          # facultatif : retire les passes supersédées
```

- Une ligne fantôme est une ligne de `highlight_events` dont le XUID n'est PAS un participant du
  match et dont le jumeau (même match, type, instant, type_hint, raw_json) est porté par un
  participant. Un match sans participants chargés n'en produit aucune : dans le doute, rien n'est retiré.
- `highlight_events` porte des index ART (ADR 0019) : pas de `DELETE`. La table est reconstruite à
  côté sans les lignes fantômes et échangée en une transaction (même cœur que `compact-passes`,
  ADR 0026) ; les `id` et la séquence ne bougent pas. Les couples de tous les matchs dont les
  couples stockés diffèrent de l'appariement courant (fenêtre de 5 ms, ordre total des événements)
  sont recalculés et écrits par échange de `killer_victim_pairs` en une transaction, sans `DELETE`.
  Les noms viennent de l'annuaire de lecture (`v_gamertag_lookup`), puis des couples existants du
  match, jamais du XUID. Une exécution interrompue se reprend d'elle-même : les matchs purgés gardent
  des couples qui diffèrent. La migration partagée `shared_killer_victim_pairs_drop_legacy_indexes_v1`
  retire les trois index ART que les bases anciennes portent sur `killer_victim_pairs`
  (`idx_kv_match`, `idx_kv_killer`, `idx_kv_victim`).
- **Sauvegarde (obligatoire)** : copie octet pour octet avant toute écriture
  (`<nom>.<slug>.avant-fantomes-<UTC>.duckdb`), restauration = la remettre en place, serveur arrêté.
- **Idempotente** : sans ligne fantôme, une seconde exécution n'écrit rien.
- La fusion refuse aussi, depuis `killsource-2026-10-16`, d'ajouter une seconde mort à un instant que
  le kill-feed de l'API ne compte qu'une fois, et contrôle les morts servies de chaque joueur contre
  `match_participants.deaths` (compteur `killsource_fusion_joueurs_au_dela_officiel_residuel`, journal
  WARN avec `match_id`, joueur, officielles, servies). Le contrôle ne retire jamais une ligne : en
  retirer une pour tomber juste reviendrait à choisir, sans preuve, quelle mort est fausse. Les morts fantômes des autres cas (p. ex. un joueur servi 10 fois pour 7 morts
  officielles) sont hors des 86 matchs : elles disparaissent quand `backfill-killsource` (sans
  `--match`, tous les films périmés sous la nouvelle révision `killsource`) les re-fusionne, à
  annoncer et jouer comme une campagne.

#### Morts servies deux fois : appariement, sièges de bot, fiches de kill, morts de l'API

Depuis `killsource-2026-10-16`, quatre causes d'une mort servie deux fois sont fermées :

- **Appariement du crédit** (`analysis.ComputeKillerVictimPairs`) : kills et morts s'apparient par
  écart croissant (l'instant exact d'abord), un kill et une mort au plus une fois chacun, jamais un
  joueur avec lui-même. L'ancien appariement glouton dans l'ordre des kills laissait un kill sur un
  bot (aucune mort dans le kill-feed de l'API) prendre une mort humaine tombée 1 à 4 ms plus loin ;
  34 couples « X tue X » étaient sur la copie du 2026-10-15. La fenêtre est
  `analysis.KillDeathPairToleranceMS` (5 ms), la seule du dépôt (ratchet
  `archlint/kill_death_tolerance_test.go`).
- **Siège de bot repris par un humain** (décodeur) : un bot ne tient son indice de joueur que pendant
  ses déclarations `BOT_METADATA` ; après la dernière, la lecture du motif de XUID nomme son
  successeur.
- **Fiche de kill** (décodeur) : un kill-event va au kill de son tueur le plus proche, et au kill dont
  la mort de la victime est voisine dans le flux.
- **Fusion** (`persist.MergeCreditAndFilm`) : toute mort d'humain servie s'adosse à UN événement
  `death` de l'API (même XUID, à 5 ms près), les morts de crédit d'abord ; une orpheline de film à
  victime humaine sans événement libre est refusée et comptée
  (`killsource_fusion_orphelins_sans_mort_api`). Le contrôle des morts officielles compare chaque
  joueur au plus grand de ses morts officielles et de ses événements `death` : un joueur parti puis
  revenu n'a de morts officielles que sur une partie du match.

Les couples déjà dans `killer_victim_pairs` et la base crédit gardent l'ancien appariement jusqu'à
cette séquence (**serveur arrêté**, prévenir avant de la jouer en production ; les deux dernières
étapes redécodent tous les films périmés, une campagne de plusieurs heures) :

```bash
cd apps/go-api
go run ./cmd/levelup repair-highlight-phantoms --dry-run   # lecture seule : fantômes, matchs dont les couples diffèrent
go run ./cmd/levelup repair-highlight-phantoms             # sauvegarde, fantômes, puis les couples de ces matchs
go run ./cmd/levelup backfill-killsource --credit-only               # base crédit de tous les matchs, nouvel appariement
go run ./cmd/levelup backfill-killsource                             # passe film par-dessus (tous les films périmés)
go run ./cmd/levelup compact-passes                                  # facultatif : retire les passes supersédées
```

La réparation ne réécrit que les matchs dont les couples stockés diffèrent de l'appariement courant
(477 sur 4 010 sur la copie du 2026-10-15) et est idempotente.

### Rattrapages de l'historique Halo 5 et rangs des saisons passées (outils autonomes, serveur arrêté)

Binaires autonomes (`apps/go-api/cmd/<nom>`), à lancer depuis la racine du dépôt ; ils demandent la
compilation CGO (gcc) comme le serveur. Chacun ouvre directement la base qu'il écrit : le **serveur
doit être arrêté** (un seul processus écrivain par base). Les outils Halo 5 lisent `db_profiles.json`
et le magasin de jetons (`data/auth/watcher_tokens/`) ; quand le jeton de rafraîchissement de la
cible est mort, empruntez celui d'un compte sain avec `LEVELUP_H5_AUTH_AS=<Gamertag>` (l'historique,
les événements et les dossiers de service Halo 5 sont servis pour n'importe quel gamertag à n'importe
quel jeton valide). Le gamertag positionnel vaut `JGtm` par défaut.

```bash
go run ./apps/go-api/cmd/h5-backfill [Gamertag] [tailleDePage]              # tailleDePage : 25 par défaut
go run ./apps/go-api/cmd/h5-events-backfill [Gamertag-auth] [maxMatchs]     # 0 ou absent = tous les matchs
go run ./apps/go-api/cmd/h5-kill-kind-backfill [Gamertag-auth] [maxMatchs] [force]
go run ./apps/go-api/cmd/h5-csr-backfill [Gamertag]
go run ./apps/go-api/cmd/h5-roster-refetch [--commit] [--match ID] [--limit N] [--title halo_5] [--resolver]
go run ./apps/go-api/cmd/backfill-csr-history -xuid XUID -player-db CHEMIN [-season CsrSeason12-1] [-dry-run]
```

- `h5-backfill` : historique Halo 5 complet d'un joueur (de 2016 à aujourd'hui), en complément de la
  synchronisation incrémentale. Chaque page est persistée dès qu'elle est lue et les matchs connus
  sont sautés : relancer après une interruption reprend sans doublon (INSERT seul). Affiche à la fin
  les effectifs de la base partagée Halo 5.
- `h5-events-backfill` : relit les événements par match des matchs Halo 5 collectés avant l'existence
  des surfaces dérivées et ne réécrit que celles-ci (précision des armes, événements marquants
  d'assistance et d'objectif). Idempotent. Le compte d'authentification est le premier argument (ou
  `LEVELUP_H5_AUTH_AS`).
- `h5-kill-kind-backfill` : relit les événements et recalcule `weapon_kills` avec la mécanique du
  frag (`kill_kind`), écrite en nouvelle génération (INSERT seul, `v_weapon_kills` lit la dernière).
  Sans `force`, seuls les matchs dont `kill_kind` est vide sont traités ; `force` (troisième argument
  `force`, `1` ou `true`, ou `LEVELUP_KK_FORCE=1`) recalcule tous les matchs, nécessaire pour
  recueillir une valeur de `kill_kind` nouvellement introduite.
- `h5-csr-backfill` : écrit les instantanés de CSR de carrière en arène d'un joueur, par liste de
  lecture classée, dans `player_csr_snapshots` de sa base joueur Halo 5 (la cible doit être déclarée
  dans `db_profiles.json`).
- `h5-roster-refetch` : matchs Halo 5 dont l'effectif est incomplet et non reconstructible hors
  ligne (BTB 8 contre 8, matchs avec remplacements) : relit le match et ajoute les participants
  MANQUANTS avec leurs statistiques complètes (INSERT seul, garde `NOT EXISTS` sur
  `(match_id, xuid)`, `match_registry` intact). **Simulation par défaut** : rien n'est écrit sans
  `--commit`. Les identités viennent d'abord du fil des frags local du match ; `--resolver` ajoute le
  résolveur universel en repli (désactivé par défaut).
- `backfill-csr-history` : instantanés de rang (CSR) des saisons PASSÉES d'un joueur Halo Infinite,
  qui alimentent le menu des saisons de la page Carrière : la synchronisation n'écrit que la saison
  courante. Les saisons viennent de `csr_season_calendars` dans `-metadata-db` (toutes, sauf si
  `-season` est donné) ; une saison jamais jouée en classé n'a pas de palier réel et n'écrit rien.
  `-dry-run` liste les saisons visées sans appeler l'API ni écrire. Autres drapeaux : `-metadata-db`,
  `-watcher-tokens-dir`, `-title` (`halo_infinite` par défaut), `-gamertag` (journaux seulement),
  `-env-file` (`.env.local` par défaut), `-rate-limit` (requêtes par minute, 60 par défaut). Le jeton
  de rafraîchissement du joueur doit se trouver dans le magasin de jetons.

### Calibrage du LUSR (outils autonomes, serveur arrêté)

Estimations hors ligne qui alimentent la note LUSR v2 à partir de l'historique des matchs de la base
partagée Halo Infinite. À lancer depuis la racine du dépôt (le `--db` par défaut est le chemin
relatif `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`), **serveur arrêté** : ils
ouvrent la base directement. Commencer par `--dry-run` quand il existe.

```bash
go run ./apps/go-api/cmd/lusr_v2_ttt_batch [--dry-run] [--db CHEMIN] [--smooth [--write-smoothed]]
go run ./apps/go-api/cmd/lusr_v2_squad_estimate [--dry-run] [--db CHEMIN] [--weeks 4] [--min-matches 10] [--gain 6.0]
go run ./apps/go-api/cmd/lusr_v2_replay [--db CHEMIN] [--reset] [Gamertag ...]
```

- `lusr_v2_ttt_batch` : hyperparamètres empiriques par groupe de listes de lecture (probabilité de
  match nul, moyenne et écart type des frags et des morts, nombre de matchs) et matrice de couplage
  entre modes, ajoutés à `lusr_hyperparams_v2` sous une source datée `batch_AAAA_MM_JJ` ; le
  calcul de fond les relit par les vues `_latest`. Idempotent (une relance ajoute une ligne, la vue
  dédoublonne). `--smooth` estime en plus un tau par joueur et par groupe et l'écrit dans
  `ttt_tau_empirical` ; `--write-smoothed` (avec `--smooth`) écrit en outre le mu lissé terminal dans
  `player_skill_state_v2`. Estimation de première passe, pas le lissage complet « Through-Time ».
- `lusr_v2_squad_estimate` : décalage de synergie d'escouade (`player_squad_offset`) de chaque paire
  de coéquipiers ayant au moins `--min-matches` matchs ensemble sur les `--weeks` dernières
  semaines, par groupe de listes de lecture : moyenne de (issue réelle - probabilité de victoire
  prédite par les notes SOLO), convertie en mu par `--gain`. Première passe prudente (les notes solo
  courantes incluent déjà ces matchs, le décalage est donc sous-estimé). La note lit ces
  décalages dès qu'ils existent (aucun interrupteur côté serveur).
- `lusr_v2_replay` : rejoue le LUSR v2 sur tout l'historique des joueurs nommés (par défaut quatre
  gamertags figés : passez les vôtres) et imprime sur la sortie standard un rapport markdown à
  comparer aux niveaux cibles validés. Le rejeu écrit lui-même l'état de note
  (`player_skill_state_v2`). **`--reset` supprime toutes les lignes de `player_skill_state_v2`**, pour
  tous les joueurs, avant le rejeu (état neuf depuis les priors).

### Réparations de données (outils autonomes, serveur arrêté)

Deux outils de réparation des données existantes, jamais lancés par le serveur. Chacun ouvre
directement les bases qu'il écrit : le **serveur doit être arrêté**. La synchronisation produit déjà
des données justes pour les nouveaux matchs ; ces outils ne concernent que des lignes écrites avant.

```bash
cd apps/go-api
go run -tags cgo ./cmd/purge_corrupt_records [--data-root CHEMIN] [--apply]
go run ./cmd/backfill_time_played --db CHEMIN [--commit]
```

- `purge_corrupt_records` : retire les records personnels invraisemblables : valeurs hors des bornes
  de vraisemblance (une précision de « 7333 % », un `best_kda` de 107) ou métriques hors du
  catalogue suivi. Il couvre `player_records_history` (et sa vue `player_records_latest`) dans
  `warehouse/shared_social.duckdb`, et `record_history` dans le `stats.duckdb` de chaque dossier
  joueur sous `players/`, le tout sous `--data-root` (défaut `../../data/titles/halo_infinite`,
  relatif à `apps/go-api` ; passer `data/titles/{slug}` pour un autre titre). Jamais de `DELETE`
  brut : chaque table est reconstruite avec les seules lignes plausibles et échangée dans une seule
  transaction (recette de l'ADR 0026), avec un contrôle du nombre de lignes avant le COMMIT et un
  retour arrière complet à la moindre erreur ; clé primaire, index et vue sont recréés.
  **Simulation par défaut** : sans `--apply`, il imprime seulement, par table, les lignes à retirer
  regroupées par (métrique, valeur). Il ouvre chaque base en lecture-écriture dans les deux modes :
  le serveur doit donc être arrêté même pour la simulation. Il ne fait aucune sauvegarde de
  lui-même.
- `backfill_time_played` : recalcule `time_played_seconds` de `match_participants` à partir des
  horodatages de participation (`first_joined_time`, `last_leave_time`), en gérant les joueurs partis
  avant la fin et les arrivés en cours de match, dans la fenêtre de jeu du match (de
  `real_start_time` au début plus la durée). `--db` est la base de matchs partagée
  (`shared_matches_v2.duckdb`). **Simulation par défaut** (base ouverte en lecture seule) : il
  imprime la répartition par qualité, l'écart entre la valeur recalculée et celle de l'API, et le
  garde-fou des matchs complets (la médiane de `time_played` des joueurs présents du début à la fin
  doit être proche de la durée de jeu, cible : plus de 95 % à 5 % près). `--commit` écrit, dans une
  seule transaction, uniquement les lignes de qualité `ok` ; les lignes sans `first_joined_time`
  gardent leur valeur de l'API. Serveur en marche, viser une copie de la base pour la simulation.
  Réserve : les nouveaux matchs sont déjà corrects, la synchronisation calcule elle-même cette valeur
  (`timeline.ComputeTimePlayedFor`, appelé depuis `internal/sync/transforms.go`) ; la commande ne sert
  qu'aux anciens matchs. Elle dépend de l'heure d'arrivée du joueur (`first_joined_time`), dont le
  fuseau horaire est une dette connue : lire un échantillon du rapport de simulation avant de
  l'appliquer en masse.

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

### Maintenance des médias HLS (outils autonomes)

Deux binaires pour la version HLS des médias (clips servis par segments). Le serveur fait déjà le
travail courant lui-même : il transcode les médias sans version HLS après chaque scan ou
synchronisation des médias. Ces outils servent aux gros rattrapages et à la réparation d'arbres
existants. Les deux ont besoin de `ffmpeg` et `ffprobe` (sauf mention contraire).

```bash
cd apps/go-api
go run ./cmd/backfill-media-hls --db CHEMIN [--captures-base DOSSIER] [--settings app_settings.json] [--slug GT] [--limit N] [--dry-run] [--retry-failed] [--delete-source=false]
go run ./cmd/migrate-hls-audio --root DOSSIER [--slug GT] [--limit N] [--dry-run] [--collapse-redundant]
go run ./cmd/migrate-hls-audio --collapse-dir DOSSIER [--dry-run]
go run ./cmd/migrate-hls-audio --analyze FICHIER
```

- `backfill-media-hls` : convertit en HLS les vidéos médias qui n'ont pas encore de version HLS, et
  reprend les transcodages laissés à `processing` sans `hls_path` après un plantage. C'est une fine
  enveloppe autour de la routine que le serveur lance après chaque scan. **Serveur arrêté** : il écrit
  `shared_social.duckdb` (`--db`, requis) en lecture-écriture. `--captures-base` vaut par défaut
  `media_captures_base_dir` du fichier donné par `--settings`. `--slug` limite à un joueur, `--limit`
  borne le nombre de clips transcodés. `--dry-run` liste sans transcoder. `--retry-failed` réarme
  d'abord les transcodages en échec (`transcode_status` `failed` remis à NULL, dans le périmètre de
  `--slug`), puis balaie ; les échecs ne sont plus rejoués automatiquement, c'est donc un geste
  explicite de l'opérateur (ignoré avec `--dry-run`). **`--delete-source` vaut vrai par défaut ici** :
  après un transcodage HLS réussi, le fichier source est SUPPRIMÉ ; passer `--delete-source=false`
  pour garder les originaux.
- `migrate-hls-audio` : maintenance sur place des arbres HLS multipistes EXISTANTS. Il ne touche que
  des fichiers sous la racine des médias (aucune base). Les modes sont exclusifs ; les modes
  d'écriture remplacent des fichiers servis : lancer **serveur arrêté** (ou à faible trafic : un clip
  lu pendant l'échange pourrait saccader). Tous sont idempotents et relançables, et tous sauf
  `--analyze` acceptent `--dry-run`.
  - défaut (`--root` ; `--slug`, `--limit`) : rend mono-codec AAC le groupe audio des arbres dont les
    renditions mélangent des codecs (jeu en Opus, voix et mix complet en AAC), pour que le changement
    de piste fonctionne sur Firefox et Safari. Ni la vidéo ni le master ne sont touchés.
  - `--collapse-redundant` (`--root`) : sur les clips dont les renditions `game` et `voices` portent le
    MÊME son (pas de voix isolée, par exemple des sessions solo OBS), réécrit le master pour n'exposer
    que `game`. Gardé par la corrélation d'enveloppe : un vrai clip jeu et voix n'est pas touché.
  - `--collapse-dir DOSSIER` : réduction FORCÉE d'UN seul arbre (dossier contenant `master.m3u8`) sans
    le critère de corrélation, pour des clips anciens dont la redondance est structurelle mais dont la
    corrélation passe sous le seuil à cause de la voix, et dont les sources sont supprimées. À réserver
    à un arbre dont on sait la redondance. N'a pas besoin de `ffmpeg`.
  - `--analyze FICHIER` : inspection en lecture seule de la disposition audio d'un fichier SOURCE
    (détection de la piste de mix complet, composante jeu, mesures) ; imprime la décision, n'écrit rien.
  - Code de sortie 1 si un arbre a échoué ; `--root` ou l'un des deux autres modes est requis.

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
`.ai/archive/V7.5/PLAN_CUISSON_PERF.md`). Ils lisent le cache local de films ; les deux outils hors ligne
n'ont besoin d'aucune base et décodent un film par processus enfant borné (plafond mémoire dur,
priorité CPU basse, verrou solo).

```bash
cd apps/go-api
go run ./cmd/levelup replay-facts-export --out <copie du décodeur>/replay/testdata/equivalence \
  [--title slug] [--oracle] <short8|match_id>...
```

Écrit un `<short8>.facts.json` par match — lignes de match, scores des deux camps, variante,
identités de carte candidates — dans la forme que `replay-build --facts` lit déjà. Sans ces faits,
zones, actions d'objectif, VIP/crâne/bombe, socles et points d'apparition sont court-circuités et
une passe d'équivalence serait vacuante. Lecture seule (`OpenReadForQuery`) ; la commande échoue
franchement au lieu d'écrire des faits vides — arrêter un serveur qui tient la base partagée en
écriture. `--oracle` écrit aussi `<short8>.oracle.json` : les vérités officielles que la cuisson ne
lit jamais (score personnel, tirs, kills par catégorie, présence booléenne, stats d'objectif de
`match_objective_stats_latest`), l'oracle du banc de vérité (`cmd/replay-verite`). C'est un fichier
SÉPARÉ, exprès : les entrées de la cuisson restent identiques à l'octet. Un oracle vide est refusé
comme des faits vides.

```bash
make replay-equiv                                  # idem, depuis la racine ; joué aussi par `make gate-push` (exige le parc local de films, ~12 min, code 1 = écart)
go run ./cmd/replay-equiv                          # tout le corpus (CORPUS.txt), comparaison seule
go run ./cmd/replay-equiv -films 000d5950 -update  # (re)fige les références d'un seul film
# flags : -corpus F  -films a,b (remplace le corpus)  -update  -mem-gib N (défaut 3, 0 = désarmé)
#         -title slug  -out-dir D (conserve les TSV des enfants au lieu d'un temporaire effacé)
# -update écrit dans la brique du décodeur : à lancer sur un clone de son dépôt, relié par l'espace de travail local,
# où qu'il soit sur le disque :
#   bash scripts/briques-workspace.sh ../halo-infinite-film-decoder
#   export GOWORK="$PWD/levelup-briques.work"   # PowerShell : $env:GOWORK = "$PWD\levelup-briques.work"
# Sans GOWORK la brique est résolue vers le cache de modules (lecture seule) : -update refuse et nomme le script.
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
`replay/testdata/equivalence/<short8>.tsv` du décodeur de film (`halo-infinite-film-decoder`, à la version qu'exige `go.mod`), chacune ouverte par son marqueur
`# digest-grammar: N` : une référence figée sous une autre grammaire est une panne
d'infrastructure (« re-figer par `-update` »), jamais un écart de décodage. `-update` réécrit ces
références au lieu de les comparer — pour une correction déclarée seulement. Le mode `-walkers`
(divergence des grammaires de découpage sur tout le cache de films) a été **retiré** en 2026-09 :
il portait en copie trois marcheurs de paquets historiques dont les originaux n'existent plus, il
ne se comparait donc plus qu'à lui-même. Sa mesure reste figée au §2 de
`.ai/archive/V7.5/MESURES_CUISSON_PERF.md` et rejouée en CI par le test de la mini-bobine de
`internal/analysis/filmsource`.

Le harnais est **hermétique pour le catalogue des socles d'armes** : la construction qu'il cuit lit
le catalogue versionné plus la fixture `cmd/replay-equiv/testdata/map_weapon_pads_overlay.json` (la
seule carte avec laquelle les références ont été figées), jamais l'overlay non versionné que le
runtime d'un poste remplit ; le verdict ne dépend donc plus de la machine. Une fixture absente est
une erreur, pas un repli silencieux.

```bash
levelup replay-parc-vies --faits <dossier des faits> --cache <depot>/data/cache --sortie <racine de travail> \
  [--ids <fichier>] [--garder-documents] [--title slug]
levelup replay-parc-vies --comparer <racine avant> <racine après>
```

Les vies sans identité sur le parc des films. Pour chaque film nommé par un fichier de faits
(`replay-facts-export`), la commande construit le rejeu dans un enfant borné
(`internal/replaychild` : verrou solo en attente bornée à 10 min sur le cache des films, sentinelle
mémoire, priorité basse), un film à la fois, **sans ouvrir de base ni rien écrire sous le dépôt**
(l'artefact passe par le fichier temporaire de l'enfant, les faits de film ne sont pas rangés).
Elle écrit `films.tsv` (une ligne par film : match, build, issue, variante, pistes, pistes sans
identité, compteurs du pont, `slot:début-fin` de chaque piste sans nom) et `identites/<id>.tsv`
(slot, bornes, équipe, identité et provenance du lien de chaque piste publiée) ;
`--garder-documents` dépose aussi `documents/<id>.json`. Un film déjà présent dans `films.tsv` est
sauté : on reprend en relançant. `--comparer` relit deux racines et rend les vies nommées, les vies
nommées perdues, les noms changés, les équipes changées et les pistes redécoupées — la preuve qu'un
correctif du nommage doit apporter sur le parc entier. Nommer la racine de travail `parc*` dans le
scratchpad (hygiène du disque).

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
cd <copie du décodeur>   # le décodeur vit dans son propre dépôt (halo-infinite-film-decoder)
# la ligne de base commise (à régénérer seulement sur un changement déclaré)
go test -bench . -run '^$' -count 10 ./internal/grammar/ \
  > internal/grammar/testdata/bench_baseline.txt

# comparer après un changement, sur la MÉDIANE (ce que benchstat rapporte)
go test -bench . -run '^$' -count 10 ./internal/grammar/ > /tmp/apres.txt
benchstat internal/grammar/testdata/bench_baseline.txt /tmp/apres.txt
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

## Coquille desktop (Tauri 2, Rust)

L'application Windows vit dans `apps/desktop` (ADR 0045). C'est une fenêtre autour du même serveur Go,
lancé comme processus fils avec `-mode desktop -arret-sur-stdin` ; Rust n'est admis qu'ici, sans aucune
logique métier (D-18). Prérequis : `cargo` et `rustc` (stable, 1.88 ou plus), Node 24, Go avec CGO (gcc de
msys64) pour le vrai serveur.

```bash
make desktop-build   # vrai serveur (scripts/desktop-dev.ps1) -> src-tauri/binaries/, puis cargo build
make desktop-test    # cargo test + clippy -D warnings + fmt --check, puis vet et tests du faux serveur
make desktop-dev     # compile le serveur, puis `npx tauri dev` sur la racine du dépôt et une racine de données de développement
make desktop-installer  # installeur NSIS (front, serveur, ffmpeg LGPL, ressources), sans clé de signature
make desktop-e2e        # installe, lance, ferme, relance et désinstalle cet installeur sur une racine de données jetable
```

CI : le job `desktop-test` (Windows) joue les tests de la coquille, clippy et le format ; le job `desktop-audit` (Linux) lance `cargo audit` sur
`apps/desktop/src-tauri/Cargo.lock` avec un `cargo-audit` épinglé : une **vulnérabilité** connue le rougit, les avis « non maintenu », « unsound » et « yanked »
de l'arbre transitif restent des avertissements. Les deux suivent le filtre de chemins `desktop`.

`desktop-dev` prend `%LOCALAPPDATA%\LevelUp-dev` pour racine de données (`-DataDir` pour la changer), jamais
celle de l'application installée. La coquille lit trois surcharges facultatives dans son propre environnement,
`LEVELUP_RESOURCES_DIR`, `LEVELUP_DATA_DIR` et `LEVELUP_WEB_DIST` (chemins absolus ; le dernier est le front
compilé que le serveur sert, `scripts/desktop-dev.ps1 -BuildWeb` le construit) ; toute autre variable `LEVELUP_*` ou `SPNKR_*`
est retirée avant le lancement du serveur. Le journal de la coquille est `<racine des données>\logs\desktop.log`
(5 Mo, un fichier `.1`) : la sortie d'erreur du serveur relayée ligne à ligne, et les lignes propres de la coquille.

`cargo test` tourne sans fenêtre. Ses tests de contrat (`src-tauri/tests/contract.rs`) compilent le faux serveur
`apps/desktop/testdata/fake-sidecar` (son propre module, `GOWORK=off`, aucun import de `levelup/go-api`) par
`go build` dans un dossier temporaire et jouent contre lui chaque cas du contrat : `FAKE_VARIANT`
(`never_ready`, `failed`, `failed_no_port`, `late_stopped`, `crash`, `crash_after_ready`),
`FAKE_CHATTY_BYTES`, `FAKE_*_DELAY_MS`. `cargo audit` (`cargo install cargo-audit --locked`) contrôle les
dépendances de `Cargo.lock`. Une compilation de release (`--release`) refuse une clé publique d'updater vide et
un binaire serveur absent.

### Installeur, updater et test de bout en bout (L17)

`apps/desktop/scripts/` (PowerShell et bash, les mêmes scripts que les jobs `desktop-build` et `desktop-e2e` de
`.github/workflows/release.yml`) :

```powershell
powershell -File apps/desktop/scripts/build-installer.ps1 [-SkipWeb] [-SigningKey <fichier> [-SigningPassword <mdp>]] [-OutDir <dossier>]
powershell -File apps/desktop/scripts/stage-bundle.ps1 -WebDist <apps/web/dist> -FfmpegDir <dossier> [-ArchiveDir <dossier>] [-Version <X.Y.Z>]
powershell -File apps/desktop/scripts/e2e-install.ps1 -Installer <LevelUp_X.Y.Z_x64-setup.exe> [-EngineInstaller <LevelUp_X.Y.Z_x64-engine-setup.exe>]
bash apps/desktop/scripts/check-version.sh <X.Y.Z> [<serveur.exe> [<empreinte du paquet>]]
cd apps/go-api && go run ./cmd/resources-manifest -dir <dossier du paquet> [-version <X.Y.Z>] [-zip <archive.zip>]
```

- L'installation est faite de deux unités (ADR 0045 D-30) : le **moteur** (coquille, serveur, front compilé sans `web/titles`, `config/`) et le **paquet de
  ressources** (`static/`, `web/titles/`, référentiel du rejeu, ffmpeg, `resources.manifest.json`), identifié par son **empreinte** (SHA-256 du manifeste trié, calculée au build, aucune
  version à incrémenter). `stage-bundle.ps1` assemble les deux (`src-tauri/bundle-staging` et `src-tauri/pack-staging`, ignorés par git), lance
  `cmd/resources-manifest` (le seul générateur de manifeste, aussi utilisé par le Dockerfile), écrit l'empreinte dans `pack-staging.fingerprint` et, avec
  `-ArchiveDir`, l'archive `levelup-resources-<empreinte>.zip`.
- `build-installer.ps1` compile le front, télécharge le ffmpeg LGPL épinglé (`fetch-ffmpeg.ps1` : l'URL et la somme SHA-256 sont des constantes du script, une somme
  différente arrête tout), assemble moteur et paquet, compile le serveur (`-ldflags "-s -w -X main.version=<version de Cargo.toml> -X main.resourcesFingerprint=<empreinte>"` : ni table des
  symboles ni DWARF, traces de panique conservées, et l'empreinte de SON paquet) et lance `npx tauri build` deux fois : `tauri.bundle.conf.json` (moteur + paquet) puis
  `tauri.engine.conf.json` (moteur seul). Trois fichiers sortent sous `-OutDir` (défaut `src-tauri/target/release/installers`) :
  `LevelUp_<version>_x64-setup.exe` (installeur **complet** : première installation et page de téléchargement), `LevelUp_<version>_x64-engine-setup.exe` (**moteur seul** : celui que
  `latest.json` désigne à l'updater) et `levelup-resources-<empreinte>.zip`. Sans `-SigningKey` aucun secret n'est nécessaire (`createUpdaterArtifacts` reste
  faux) ; avec, l'installeur moteur seul et le paquet sont aussi signés (fichiers `.sig`, même clé). L'installation est par utilisateur : programme sous `%LOCALAPPDATA%ProgramsLevelUp`, données sous
  `%LOCALAPPDATA%LevelUp`, jamais supprimées par le désinstalleur.
- `e2e-install.ps1` refuse de s'exécuter si LevelUp est déjà installé. Il installe en silencieux (`/S`), vérifie le paquet installé contre son manifeste, lance
  l'application avec un `LEVELUP_DATA_DIR` jetable, attend `<données>desktopport` et `GET /health` = 200, la ferme
  par `WM_CLOSE`, vérifie la ligne de journal `arrêt: terminé` avec `clean=true` et l'absence de fichier `*.wal`,
  relance (même port), ferme de nouveau. Avec `-EngineInstaller` il joue ensuite une **mise à jour moteur seul** (`/S /UPDATE`) après avoir supprimé un fichier du moteur et un fichier du
  paquet (celui du moteur revient, celui du paquet ne doit pas revenir, tous les autres fichiers du paquet gardent leur SHA-256) et lance l'application une troisième fois ; il désinstalle enfin en silencieux, vérifie
  que le dossier du programme disparaît entièrement (le serveur livré retire le paquet, que le désinstalleur réécrit ne connaît plus) et que la racine de données existe encore.
  Code de sortie 0 ou 1 ; en cas d'échec les journaux sont copiés sous `-ArtifactsDir`.
- `check-version.sh` refuse tout écart entre la version attendue (le tag git sans `v`), `Cargo.toml`,
  `tauri.conf.json` et les `-ldflags` embarqués dans le binaire serveur (`go version -m`) ; avec un troisième argument il refuse aussi un serveur qui n'embarque pas
  `-X main.resourcesFingerprint=<cette empreinte>`.
- Workflow de release : un tag `vX.Y.Z` compile et signe l'installeur (secrets `TAURI_SIGNING_PRIVATE_KEY` et, si la
  clé a un mot de passe, `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` ; sans la clé le job échoue avant de compiler), joue le
  test de bout en bout, puis publie l'installeur moteur, le paquet (`levelup-resources-<empreinte>.zip`) avec sa signature et `latest.json` (le fichier que lit l'updater : il
  désigne l'installeur moteur et porte un bloc `resources` avec l'empreinte, l'URL, la somme SHA-256, la taille et la signature du paquet) dans la publication unique `updates`. La page de la version porte l'installeur complet et les serveurs. Un push sur `release/**` compile l'installeur sans signature et ne joue que le test de bout en bout.

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
make go-api-lint       # golangci-lint ratchet sur apps/go-api ; go vet en repli
```

#### Briques Go : modules publiés, développés dans leurs dépôts dédiés (ADR 0041, 0043)

Quatre briques sont des modules Go publiés depuis leur propre dépôt, et `apps/go-api/go.mod` les exige à une version
publiée (v1.0.1 à la bascule de la 8.1.2 ; ce dépôt ne porte plus ni `libs/` ni aucun `go.work`) : « fichiers du
jeu », `github.com/JGtm/halo-infinite-files-reader` (paquets `himodule`, `ooz`, `hinavmesh`, `mapvar`), « images de
carte », `github.com/JGtm/halo-infinite-map-images` (paquet `himap`, qui n'importe que la brique des fichiers du jeu),
le « décodeur de film », `github.com/JGtm/halo-infinite-film-decoder` (la façade `decfilm`, la couche de publication
`replay`, les couches fermées sous `internal/`, ses catalogues ; n'importe que les deux briques ci-dessus, ADR 0034),
et l'« API Halo », `github.com/JGtm/halo-api` (paquets `xblauth`, `infinite`, `halo5`, `xboxlive` ; n'importe aucune
autre brique, ADR 0042). Trois conséquences pour toute commande :

- Le `./...` de `apps/go-api` ne couvre que LevelUp : les tests d'une brique se jouent dans son dépôt (sa CI les
  joue). LevelUp garde les tests qui exigent ses données (`internal/gamefilescorpus/`, `internal/filmcorpus/`,
  `cmd/replay-equiv`, `cmd/replay-corpus-gate`).
- Une brique se monte comme toute dépendance : un nouveau tag sur son dépôt, puis la ligne `require` de
  `apps/go-api/go.mod` (voir « Développer une brique » ci-dessous). `make tidy` dans `apps/go-api` lance
  `go mod tidy` avec `GOWORK=off`, jamais contre un espace de travail local.
- Un paquet de brique se nomme par son chemin d'import : `cd apps/go-api && go test
  github.com/JGtm/halo-infinite-map-images/himap/ -run ...` joue un paquet de la version qu'exige `go.mod` (le
  Makefile fait ainsi pour `BRIQUE_FILM` et pour le corpus `gamefiles`). `make gate-push` pose `GOWORK=off` : le filet
  juge les versions publiées, jamais un espace de travail local.

##### Développer une brique

Le code d'une brique se modifie dans son dépôt dédié, jamais dans LevelUp :

```bash
# 1. cloner le dépôt dédié à côté de LevelUp
git clone https://github.com/JGtm/halo-infinite-film-decoder ../halo-infinite-film-decoder

# 2. voir le changement dans LevelUp avant de publier : engendrer l'espace de travail LOCAL
#    (levelup-briques.work, ignoré par git ; un garde-rail, archlint/no_go_work_versionne_test.go,
#    interdit de commiter un go.work), puis y pointer Go
bash scripts/briques-workspace.sh ../halo-infinite-film-decoder [../halo-api ...]
export GOWORK="$PWD/levelup-briques.work"      # PowerShell : $env:GOWORK = "$PWD\levelup-briques.work"

# 3. tester dans le dépôt de la brique (ses propres gates), puis dans LevelUp
cd ../halo-infinite-film-decoder && go test ./... -count=1
cd ../LevelUp/apps/go-api && go test ./internal/... -count=1

# 4. retour aux versions publiées
unset GOWORK                                   # ou supprimer levelup-briques.work
```

Seul `GOWORK` désigne l'espace de travail : sans la variable, tout outil Go juge les versions publiées. Publier se fait
en deux temps : (1) le mainteneur pousse le changement de la brique sur son dépôt et y pose le tag (`vX.Y.Z`, un numéro
par dépôt ; un tag ne se déplace jamais), (2) une branche de LevelUp monte le `require` (`cd apps/go-api && go get
github.com/JGtm/<brique>@vX.Y.Z && make tidy`) et joue `make gate-push`. Une pseudo-version (`@<sha>`) est permise sur
une branche de travail, un tag est obligatoire avant `main` (ADR 0043, amendement du 2026-11-12). Les tags de
l'ancien chemin `libs/<brique>/vX.Y.Z` sont dépréciés et ne reçoivent plus de version ; `github.com/JGtm/halo-api`
retire v0.1.0 et v1.0.0.

Une brique n'a ni base de données ni donnée de LevelUp : ses tests ne lisent que leur `testdata/` et le jeu installé.
Un test du film qui exige les données versionnées du dépôt hôte en prend la racine dans
`DECODEUR_FILM_RACINE_DONNEES` (une copie de travail de LevelUp) et se saute sans elle ; une valeur qui ne mène nulle
part fait échouer le test.

#### Corpus du tag de build `gamefiles` (cartes **et** catalogues commis)

Deux familles de tests lisent le **jeu installé**, toutes deux derrière `//go:build gamefiles` :

- les fichiers `*_gamefiles_test.go` du paquet `himap` de `halo-infinite-map-images` (34 au 2026-10-28, après le lot 2c) décodent les
  modules du jeu, et ceux de `internal/gamefilescorpus/` (25, un paquet de tests seuls : depuis
  le plan de découplage D1, les tests de `himap` qui lisent des données de LevelUp y vivent et
  appellent `himap` par son API exportée) balaient les 26 cartes du catalogue. Ils sont longs par nature — mesuré le 2026-09-05,
  `TestBalayageCoquille` prend à lui seul **203 s** pour 26 cartes (1 246 s avant le passage du
  lecteur de modules en projection mémoire, le même jour). Le tag garde un `go test
  ./himap/` nu (depuis le dépôt `halo-infinite-map-images`) utilisable (2,8 s) ;
- trois fabricants de catalogues sous `cmd/` re-dérivent leur catalogue **commis** depuis le jeu
  installé et le comparent à l'octet : `cmd/film-profiles-build/`, `cmd/mapfond-build/`,
  `cmd/mapstruct-build/` (9,0 s à eux trois, mesuré le 2026-09-17).

`make go-api-test-gamefiles` joue **les six paquets** (le paquet `himap` de la brique des images de carte et le paquet `hinavmesh` de la brique des fichiers du jeu, par chemin d'import, les quatre autres depuis
`apps/go-api`). Jusqu'au 2026-09-17 la cible ne jouait que la brique : les trois tests `cmd/` n'étaient joués par AUCUNE commande du dépôt
(découverte D1 (3.1.2) ; le plus ancien est tagué depuis le 2026-09-05). La liste des paquets est
explicite et non `./cmd/...` : sous le tag, un paquet sans fichier `gamefiles` n'apporte que du
temps de compilation. `archlint.TestCibleMakefileGamefilesCouvreLeCorpus` rougit si un paquet
entre au corpus sans entrer dans la cible.

```bash
make go-api-test-gamefiles                       # corpus entier (~6 min, exige le jeu)
cd apps/go-api && CGO_ENABLED=1 go test -tags=gamefiles -count=1 -timeout 3600s github.com/JGtm/halo-infinite-map-images/himap/ -v
cd apps/go-api && CGO_ENABLED=1 go test -tags=gamefiles -count=1 -timeout 3600s \
  ./internal/gamefilescorpus/ \
  ./cmd/film-profiles-build/ ./cmd/mapfond-build/ ./cmd/mapstruct-build/ -v

# Les catalogues commis seuls (des secondes, pas des minutes) :
cd apps/go-api && CGO_ENABLED=1 go test -tags=gamefiles -count=1 \
  ./cmd/film-profiles-build/ ./cmd/mapfond-build/ ./cmd/mapstruct-build/ -v

# Une seule carte (beaucoup plus rapide) :
BALAYAGE_CARTES=aquarius_map go test -tags=gamefiles -timeout 300s \
  ./internal/gamefilescorpus/ -run TestBalayageCoquille -v

# Jeu installé ailleurs :
cd apps/go-api && LEVELUP_HALO_DEPLOY=/chemin/vers/Halo Infinite/deploy go test -tags=gamefiles github.com/JGtm/halo-infinite-map-images/himap/
```

Les tests natifs de `hinavmesh` ne cherchent pas l'installation eux-mêmes : ils exigent
`LEVELUP_HALO_DEPLOY` (le dossier `deploy` du jeu), que la cible transmet depuis l'environnement
(elle affiche un avertissement s'il n'est pas posé), et prennent leur `t.Skip` sinon.

Sans installation du jeu, chaque test prend son `t.Skip` et le corpus est vide en une
seconde — c'est exactement ce qui se passe en CI. La CI se contente donc de le **compiler**
(`go vet -tags=gamefiles` du paquet `himap` et de `./internal/gamefilescorpus/`, depuis `apps/go-api`, job `go-build` ; la CI de la brique compile le reste) ; elle ne l'exécute jamais. Le
tag lui-même est tenu sur tout l'espace de travail par `internal/archlint/gamefiles_tag_test.go`, qui
tourne dans le build par défaut.

**Test rouge connu** (devenu un instrument de recherche depuis le lot 2c de D1 : `sddt_research_test.go`, joué avec `-tags=gamefiles,research`) : `TestBancCliffhanger` échoue (accord 64,4 % contre une référence
re-basée à 64,7 %). Il est *préexistant*, pas une régression — vérifié le 2026-09-05 en le
rejouant sur le commit précédent, qui rend des chiffres identiques au bit près. Personne ne
pouvait le voir : le corpus ne terminait jamais, et la CI ne l’exécute pas. Consigné dans
`.ai/REGISTRE_REPORTS.md`.

**Instruments de recherche des briques (D1 lot 2c, 2026-10-28).** Les sondes de rétro-ingénierie des briques
`himap`, `hinavmesh` et `mapvar` qui lisent `.ai/re_dump` (le dump de rétro-ingénierie non versionné) ou des
données du dépôt hôte (`data/...`) sont *conservées* comme instruments de recherche, hors du build par défaut
ET de la suite `gamefiles` ci-dessus. Ce sont des fichiers `*_research_test.go` ; ceux qui ouvrent aussi le jeu
portent `//go:build gamefiles && research` (les autres `//go:build research`). Ils tournent à la demande et
exigent le jeu installé et, pour certains, `.ai/re_dump` :

```bash
cd <copie images de carte> && CGO_ENABLED=1 go test -tags=gamefiles,research -count=1 ./himap/ -run <Sonde>   # un instrument ; sans -run, toute la suite gamefiles plus les instruments
cd <copie fichiers du jeu> && go test -tags=research -count=1 ./hinavmesh/ ./mapvar/
```

La CI de chaque brique les compile (`go vet -tags=gamefiles,research ./...` dans son dépôt) ;
le `go vet -tags=research ./...` de tout le module laisse de côté, volontairement, ceux en `gamefiles && research`
(les deux tags sont requis). La règle est tenue par `internal/archlint/gamefiles_tag_test.go` (« un instrument qui
ouvre le jeu déclare `gamefiles` ») et `research_tag_test.go`.

#### Gate de non-régression du rejeu sur corpus témoin (`cmd/replay-corpus-gate`)

Trois régressions de données (28/08, 30/08, 02/09 — pont d'identité par manche, actions de
drapeau non attribuées, « une piste = une vie ») ont traversé des goldens SYNTHÉTIQUES verts
pendant dix-neuf schémas, faute d'un différentiel sur des films réels. `cmd/replay-diff` (déjà
utilisé pour le balayage ponctuel du parc, `.ai/archive/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`) devient ici
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
`coverage.decoder.{sourceRev, profileRev, grammarRev, killsourceRev, objectivesRev}` (`factsRev` jusqu'au schéma 71), `coverage.decoder.build`,
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
| `--mem-gib` | `4` | plafond mémoire souple (Gio) armé sur CHAQUE cuisson enfant, HEAD et base (`0` désarme). Le défaut du gate est volontairement AU-DESSUS de celui de production (`memguard.DefaultLimitGiB` = 3) : D6 a mesuré deux témoins BTB à 3,779 et 3,807 Gio côté base, soit juste au-dessus du plafond dur de 3,75 Gio — ils échouaient au hasard d'un run à l'autre. |
| `--source-root` | `git rev-parse --show-toplevel` | dépôt dont le code/la config AU HEAD est testé — **pas** basé sur `db_profiles.json` : fonctionne depuis n'importe quel worktree, y compris un sans copie locale de ce fichier |
| `--parc-root` | `source-root` s'il porte déjà la base partagée du titre, sinon auto-détecté via le `.git` commun | le parc de développement (chunks de film, artefacts `--reference=parc`) |
| `--lock-root` | `CacheRootDir()` du parc | où vit le verrou de décodage partagé |
| `--work-root` | un dossier temporaire jetable | racine de travail de la ou des cuissons fraîches |
| `--keep-work` | `false` | conserver la racine de travail après l'exécution (débogage) |
| `--json` | (aucun) | chemin où écrire aussi le rapport complet en JSON |

**À exécuter avant tout merge qui touche** `replaybuild`, le `replay` ou le `internal/grammar` du décodeur (par une nouvelle version du décodeur), ou qui
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
`.ai/archive/V7.5/v2/CORPUS_TEMOIN_2026-09-06.md`** pour l'exécution exacte et sa durée.

Si le parc local est plus ancien que le HEAD, les écarts attendus en `--reference=parc` sont des
GAINS (calques neufs, correctifs documentés) ; toute PERTE est un fait à rapporter, jamais à
masquer en resserrant le manifeste ou en filtrant le rapport.


#### Banc de vérité (`cmd/replay-verite`, 2026-09-30)

Juge un artefact de rejeu APRÈS contre un artefact AVANT du même témoin, contre des ORACLES (les
vérités officielles du match) et par des COMPTES DE VIOLATIONS (ce qu'aucun match réel ne peut
contenir), au lieu d'attribuer à la main des centaines de métriques de couverture. Conception :
`.ai/archive/V7.5/film_re/BANC_DE_VERITE_CONCEPTION_2026-09-30.md` ; bibliothèque : `internal/replayverite`.
Il ne cuit rien, ne décode aucun film et n'ouvre aucune base.

```bash
cd apps/go-api
go run ./cmd/replay-verite -avant <avant.json> -apres <apres.json> -faits <short8>.facts.json \
  [-temoin id] [-registre-avant registre.json]
go run ./cmd/replay-verite -registre > registre.json   # registre des replis de CETTE révision
```

Verdict (sortie 1 hors `ok`) : `FAUX` quand un faux positif d'oracle ou une classe de violation
monte, ou qu'un repli NOUVEAU se déclenche (sauf si le registre d'avant dit son compteur non encore
branché) ; `MANQUE` quand un faux négatif d'oracle monte ou qu'une preuve interne se dégrade ;
`ok` sinon. Seul le delta décide : les valeurs absolues s'affichent à titre d'information. Les
unités appariées SUR l'oracle lui-même (slots statborg `triplet_feuille`, camps rattachés par
`teamIdentity` `a`/`a0`) sont exclues, jamais comptées en vrais positifs. `-oracle
<short8>.oracle.json` ajoute O-S3 (score personnel).

**Dans `replay-corpus-gate` (2026-09-30), le verdict EST celui du banc.** Chaque témoin comparé est
jugé par le banc (référence et HEAD, mêmes faits, même oracle — le gate exporte désormais
`<short8>.oracle.json` à côté des faits par `replay-facts-export --oracle`). Le statut du témoin est
`FAUX` / `MANQUE` (banc), puis `PERTE` pour deux filets seulement : une perte `replaydiff` dans un
bloc qu'aucune mesure du banc ne couvre, ou un calque de premier niveau qui disparaît. Toutes les
autres différences `replaydiff` (pertes couvertes, changements, gains) restent au rapport à titre
d'information ; `CHANGEMENT` n'est plus un statut. Le rapport texte imprime une section
`BANC DE VERITE` par témoin AVANT le détail `replaydiff` ; le JSON porte un objet `verite` par
témoin. La règle R-1 exige le registre des replis de la base : le gate compile `cmd/replay-verite`
dans le worktree de base et lance `-registre` (une fois par passage, sans cache : il ne dépend que
du SHA de la base). Une base antérieure au banc n'a pas cet outil : le registre est alors inconnu et
tout repli nouveau est `FAUX`, ce que le rapport dit.

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

Les images Halo 5 (`apps/web/public/titles/halo_5`) sont en WebP. Pour en ajouter ou en rafraîchir une à partir d'un PNG
(ffmpeg avec `libwebp` dans le `PATH`, ou `LEVELUP_FFMPEG_DIR`) :

```bash
node scripts/convert-halo5-images.mjs [--keep] [--dir <racine>] [--jobs <n>]
```

Le script convertit par famille (plaques et fond : avec perte, qualité 85 ; emblèmes et vignettes : sans perte, car le
personnalisateur Spartan lit leurs canaux R/V/B comme des zones de couleur et le sous-échantillonnage des chrominances
d'un codage avec perte fait baver une zone sur la suivante), supprime chaque PNG converti (`--keep` les garde), est idempotent
et affiche le poids avant et après. `teams/` et tout `halo_infinite/` restent en PNG. Un garde-rail vitest
(`halo5Images.guard.test.ts`) échoue si un PNG revient.

### Gate local avant merge (`gate-push`)

```bash
make gate-push               # ratchet lint Go + harnais replay-equiv + typecheck/lint web + baseline de tests, voies film et DuckDB en parallèle (~25 min + ~12 min)
```

`make gate-push` joue `make replay-equiv` (le harnais d'équivalence du rejeu) : les références de
`replay/testdata/equivalence` du décodeur de film (lues à la version qu'exige `go.mod`) ont dérivé sans bruit de la
v8.0.0 au re-figeage du 2026-10-30 faute de gate qui les joue. Il exige le parc local de films ; pour re-figer après
une preuve d'équivalence : `go run ./cmd/replay-equiv -update` depuis `apps/go-api`, avec `GOWORK` pointant sur une
copie du décodeur (voir « Développer une brique » ; la copie peut être n'importe où sur le disque, seul le cache de
modules est refusé), puis publier la nouvelle version du décodeur et monter le `require`.

La baseline de tests joue deux voies en parallèle (ADR 0041) : la voie des briques (les paquets des briques publiées
`halo-infinite-film-decoder` et `halo-api`, joués depuis `apps/go-api` par chemin d'import, sans dépendance à DuckDB —
garde-rail `internal/archlint/film_lane_duckdb_free_test.go` — jugés contre
`.ai/baselines/tests_halo-infinite-film-decoder.jsonl` et `.ai/baselines/tests_halo-api.jsonl`) et la voie DuckDB
(tout `apps/go-api`, `-p 1`, jugée contre `.ai/baselines/tests_pre_migration.jsonl`). La voie DuckDB fixe la durée ;
`scripts/gate-push.ps1` ci-dessous exécute les deux suites l'une après l'autre.

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

### Voie CI Windows avec CGO (`go-test-windows-cgo`)

Le job CI `go-test-windows-cgo` (`.github/workflows/ci.yml`) tourne sur `windows-latest` avec `CGO_ENABLED=1` (le runner
fournit gcc, pas d'action MinGW). Il compile `./cmd/server` (DuckDB lié), puis joue `./cmd/server/...`,
`./internal/platform/diskfree/...`, `./internal/platform/filelock/...`, `./internal/filmrun/...` et `./cmd/levelup/...` depuis `apps/go-api`,
puis, toujours depuis `apps/go-api` par chemin d'import, `internal/decodelock/` et `internal/filelock/` du décodeur de film et `himodule/` de la brique des fichiers du jeu : les paquets qui portent les fichiers `*_windows.go` /
`*_win.go` (arrêt propre, verrous de fichiers, espace disque libre, priorité de processus, projection mémoire des
modules). Aucun test sous les tags `gamefiles` ou `research` n'y est joué : le jeu n'est pas installé sur le runner.
Il se déclenche sur les pushs `release/**`, les pull requests vers `main` (les seules que ce workflow reçoit). Les runners Windows coûtent
cher : les branches de travail et les pushs sur `main` le sautent. Le budget est de 15 minutes. C'est le gate
d'autorité des comportements propres à Windows ; les jobs Linux ne compilent jamais ces fichiers. Pour le rejouer en
local (PowerShell, depuis `apps/go-api`, gcc dans le `PATH`) : `go build ./cmd/server`, puis les trois appels
`go test -count=1` ci-dessus.

### Hygiène du disque (`disk-hygiene.ps1`)

Go ne retire de son cache que les entrées inutilisées depuis 5 jours, et seulement quand une
commande réutilise ce même cache : un `GOCACHE` dédié laissé à la clôture d'un lot ne rétrécit
jamais, pas plus que les racines de gate gardées dans les scratchpads des sessions.
`scripts/disk-hygiene.ps1` rend cette place. Il **simule par défaut** ; rien n'est supprimé sans
`-Apply`.

```powershell
powershell -File scripts/disk-hygiene.ps1                    # simulation : ce qui serait libéré, rapport des worktrees
powershell -File scripts/disk-hygiene.ps1 -Apply             # passe réelle (celle de la tâche planifiée)
powershell -File scripts/disk-hygiene.ps1 -Lot <lot> -Apply  # clôture de lot : supprime go-build-<lot> et golangci-<lot>
```

- Périmètre : `%LOCALAPPDATA%\go-build-*` et `golangci-*` (hors `golangci-lint`, le cache par
  défaut), `%USERPROFILE%\gocache-*`, `C:\*-gocache`, les dossiers de session sous
  `%TEMP%\claude`, `C:\t\*`. Jamais `data\`, un worktree, `.git` ni le cache des modules.
- Seuils : cache dédié inactif depuis 48 h ; cache partagé vidé par `go clean -cache` au-delà de
  25 Go ; dossier de session dont le transcript et le contenu n'ont pas bougé depuis 72 h ; dans une
  session active, `gate_work*` / `gocache*` / `golangci*` / `parc*` inactifs depuis 24 h ;
  `C:\t\*` inactifs depuis 72 h. Âge = l'écriture la plus récente dans tout l'arbre.
- Sécurité : aucune suppression tant qu'un processus `go`, `compile`, `link`, `*.test`,
  `golangci-lint`, `gate`, `replay-*` ou `levelup*` tourne (rapport seul). `-Lot` refuse un cache
  écrit dans les 10 dernières minutes. Un arbre qui contient une jonction ou un lien n'est jamais
  supprimé, seulement signalé. Les worktrees sont signalés, jamais retirés.
- Journal : `%LOCALAPPDATA%\levelup-disk-hygiene\disk-hygiene.log` (espace libre avant et après,
  chaque suppression et chaque refus ; `ALERTE` sous 60 Go libres).
- Tâche planifiée `LevelUp-disk-hygiene` : chaque jour à 4 h, utilisateur courant, sans élévation,
  `-Apply`.

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
