# Verification adverse V-GO-D1

> Contexte frais, lecture seule. HEAD reel au moment de la verif : `081871f09` (2026-09-05) ;
> `736ccf3c3` verifie ancetre de HEAD (`git merge-base --is-ancestor` = 0), seul un commit de
> plan par-dessus. Toutes les mesures ci-dessous sont refaites, jamais reprises de G9.

## Constat 1 — `diag_weapons_v3 -write` UNIQUE ecrivain de `match_objective_events` : REFUTE

- Ce que j'ai verifie :
  - `rg -n "match_objective_events" apps/go-api --type go` (et non `rg "WriteMatch\("`) →
    **`apps/go-api/internal/persist/bomb_stats_persister.go:243` : `INSERT INTO match_objective_events (`**.
    C'est un SECOND ecrivain, et il est sur le chemin canonique `persist/`, en INSERT pur.
  - Chaine d'appel vivante : `apps/go-api/internal/sync/replayartifacts/artifacts.go:352`
    (`persisterStatsBombe(ctx, d, b.usage)`) dans `func Run` (`:294`) — le cycle POST-SYNC.
    Second appelant : `apps/go-api/cmd/levelup/cmd_backfill_bomb_stats.go:194`
    (`persist.NewBombStatsPersister(db)`), sous-commande `levelup backfill-bomb-stats`
    (`cmd/levelup/main.go:131`).
  - `apps/go-api/cmd/diag_weapons_v3/main.go:10-23` : « SECURITE ECRITURE : DRY-RUN par defaut »
    et « IMPORTANT : avec -write, **stopper le serveur Go** (lock ecriture exclusif DuckDB
    sur shared_matches_v2) ».
  - `apps/go-api/internal/sync/no_art_patterns_test.go:66-69` : `tablesProtegees` = « tables que
    **ce PR a migrees en append-only** ». Les 25 entrees ont toutes une vue `_latest`.
- Ce que l'auditeur n'a pas vu :
  1. **Le second ecrivain.** Sa commande de reproduction (`rg -n "WriteMatch\(" …`) ne peut,
     par construction, trouver que des appelants de `WriteMatch` — jamais un autre ecrivain de
     la table. La conclusion « UNIQUE » est un artefact de la methode.
  2. **La coexistence est ecrite, datee et volontaire.**
     `internal/persist/bomb_stats_persister.go:25-26` : « (!) `ObjectiveEventsRepo.WriteMatch`
     (platform/duckdb) fait, LUI, un DELETE-then-INSERT, et son en-tete le documente comme
     "hors chemin live". **Il n'est NI appele NI modifie ici.** » (lot E3 Assaut, 2026-09-04).
  3. **L'absence de `tablesProtegees` est definitionnelle, pas un trou.**
     `bomb_stats_persister.go:30` : « `match_objective_events` a une PK NATURELLE
     `(match_id, seq)` et **AUCUNE vue `_latest`** : elle n'a pas de mecanique de generation. »
     Une table non migree append-only n'est pas eligible a une liste dont le contrat est
     « tables migrees en append-only ».
  4. **Le handle « de lecture » est documente comme RW dans ce mode.**
     `platform/duckdb/objective_events_repo.go:10-18` : « En mode legacy / CLI backfill
     (SharedReader == LegacySharedReader), ce handle est RW … Le backfill v3 tourne HORS chemin
     live (MaxOpenConns(1), serialise) — pas de pression concurrente ART (cf.
     `.ai/PLAN_WEAPON_ATTRIBUTION_V3.md` §10) », plus le precedent nomme `InsertWeaponKills`
     et le jumeau `player_positions_repo.go:14,45`.
  5. **Le defaut est deja classe et date.** `internal/sync/comeback_objective_test.go:87-89` :
     « rien dans la sync n'alimente `match_objective_events` (**audit 2026-08-06, P0**) » — et
     les lecteurs ont ete adaptes pour degrader (repli historique) plutot que rendre un 0 faux.
- Consequence reelle reformulee : `match_objective_events` a deux ecrivains — un INSERT-only sur
  le chemin `persist` qui sert le live (Assaut), et un DELETE+INSERT de backfill hors ligne
  documente serveur arrete ; il reste un ecart de doctrine a fermer, pas un P0 d'unique ecrivain
  hors garde-rail.

## Constat 2 — `cmd/vs-measure` « a supprimer », toujours la : REFUTE

- Ce que j'ai verifie :
  - `apps/go-api/cmd/vs-measure/main.go:1` « OUTIL JETABLE » et `:7` « A SUPPRIMER apres usage
    (pas de code mort) » — exact (l'auditeur cite `:8`, c'est `:7`).
  - `wc -l` sur les 8 fichiers = **1 987** exact. `rg -l "vs-measure" Makefile scripts docs
    .github packaging apps/go-api/internal apps/web/src` = **0** exact.
  - Dernier commit `cmd/vs-measure/` : 2026-09-02 (`97d8ec488`) ; HEAD 2026-09-05.
- Ce que l'auditeur n'a pas vu : **l'item est un report assume, entre le jour meme de HEAD.**
  - `.ai/V7.5/REGISTRE_REPORTS.md:12`, date **2026-09-05**, avec condition de reprise ecrite :
    « decision de l'auteur : garder avec un en-tete "outil de recherche, hors production" ou
    supprimer (regle 7) », et proprietaire nomme (« l'auteur du chantier »).
  - `.ai/V7.5/PLAN_INTEGRATION_BRANCHES_2026-09-05.md:478` — item **N-5** du plan d'integration.
  - `.ai/thought_log.md:6372` — « si OK, commit du lot et suppression du driver jetable
    `cmd/vs-measure` ».
  - **Balayage exhaustif du depot** (`grep -rl "vs-measure" . --exclude-dir=.git`, hors son propre
    repertoire) : **14 fichiers, TOUS sous `.ai/`** — 11 notes de chantier `V7.5/film_re/`, le
    `thought_log`, le plan d'integration et le registre. Zero Makefile, script, `docs/`,
    `.github/` ou `packaging/`. Le fait « 0 reference hors `.ai/` » est donc confirme sur
    l'ensemble du depot, et les seules references qui existent sont precisement le report date
    et sa condition de reprise.
  - **Incoherence interne de G9** : dans sa propre table « Constats ecartes », l'auditeur ecarte
    `cmd/replay-equiv` au motif « Dette assumee, `.ai/V7.5/REGISTRE_REPORTS.md:13`, avec
    condition de reprise ». `vs-measure` est a la **ligne 12 du meme registre**, avec une
    condition de reprise. Deux lignes adjacentes, deux standards opposes.
- Consequence reelle reformulee : le constat re-signale, en P1, un report deja inscrit la veille
  au registre avec sa condition de reprise et son proprietaire — c'est un doublon du registre,
  pas un oubli de la regle 7.

## Constat 3 — maillon de livraison des sons en Python hors depot : TIENT (une piece a charge sur deux refutee)

- Ce que j'ai verifie :
  - `git ls-files | grep -iE "livraison|akpk|_outils"` → **0 fichier** (seuls 3 faux positifs de
    noms de tests Go). `git ls-files "*.py"` → **3** (conformes a CLAUDE.md).
  - `git ls-files "*static/sounds*" | wc -l` = **177** — exact.
  - Aucun mode `livrer` parmi les 39 `case` de `cmd/weapon-sounds/main.go` ; aucun ecrivain Go de
    `static/sounds/` (`rg "static/sounds" apps/go-api --type go` = 1 hit, un test).
  - Les deux citations TS existent : `replaySound.ts:16`, `weaponSoundVariations.ts:2`.
- Ce qui confirme, et **ce qui est faux dans la piece a charge** :
  - **`akpk_unpack.py` est deja porte en Go.** `cmd/weapon-sounds/pck_dump.go:5-10` dit
    l'inverse de ce que l'auditeur en tire : « Jusqu'ici l'etape `.pck -> .wem` passait par un
    script Python hors depot (`_outils/akpk_unpack.py`), alors que `lirePck` (pck.go) sait deja
    lire le conteneur a l'octet pres. **Ce mode ferme le trou : la chaine complete
    (`pck -> wem -> vgmstream -> wav`) reste dans le depot, en Go.** » (commit `97d8ec488`,
    2026-09-02). Le chiffre « 2 scripts Python hors depot » est donc faux.
  - **La recette EST versionnee** et citee par les deux en-tetes TS :
    `.ai/V7.5/RECETTE_SONS_ARMES.md` (2026-08-16). Elle nomme **six** scripts, pas deux :
    `akpk_unpack.py:29`, `conv_lot.py:32`, `coups_lot.py:88`, `manifeste2.py:102`,
    `reinjecte.py:103`, `livraison.py:115`. L'auditeur en sous-compte quatre tout en en comptant
    un qui n'existe plus dans la chaine.
  - Un garde-rail existe : `apps/web/src/features/match-replay/replaySoundAssets.guard.test.ts`
    rejoue le manifeste contre le dossier livre.
- Consequence reelle reformulee : le dernier maillon (`livraison.py`) est bien hors depot et sans
  equivalent Go, donc 177 assets et `weaponSoundVariations.ts` restent non regenerables — mais le
  perimetre reel est de cinq scripts residuels, pas deux, et la recette qui les orchestre, elle,
  est versionnee.

## Constat 4 — dix chaines d'assets non documentees : TIENT (gravite -> P2)

- Ce que j'ai verifie :
  - `grep -oE "cmd/[A-Za-z0-9_-]+" docs/COMMANDS.md | sort -u` → **8** exactement
    (`backfill-team-rounds`, `levelup`, `migrate-media-paths`, `prestige-tuning-analyze`,
    `replay-build`, `replay-equiv`, `replay-worker`, `weapon-icons-build`).
    `docs/FR/COMMANDS.md` → **8** aussi : **la parite R15 est respectee**.
  - Balayage des 11 outils cites : **0 occurrence sous `docs/`** pour chacun (RUNBOOKs et
    READMEs compris) ; entre 1 et 27 occurrences sous `.ai/` chacun. Le fait est etabli.
- Ce que l'auditeur n'a pas vu :
  1. **La « regle enfreinte » n'existe pas.** La doctrine citee entre guillemets — « les assets
     versionnes sont produits par des outils Go (pas Python) et documentes dans
     `docs/COMMANDS.md` » — est introuvable : recherchee dans `CLAUDE.md`, `docs/`, `.claude/`
     et `.ai/V7.5/README.md`, **zero occurrence**. Elle est citee comme norme dans les constats
     3 ET 4. La regle reellement ecrite (CLAUDE.md R15) est une regle de **parite** sur les
     modifications de quatre guides nommes — parite qui est tenue (8/8).
  2. **`docs/COMMANDS.md` ne pretend pas etre un inventaire.** Son preambule se declare
     « Cheat-sheet for the current stack ». Le ratio « 8 sur 176 » mesure un document contre une
     promesse qu'il ne fait pas.
  3. **La consequence « un asset perime ne se detecte pas » est fausse.** Des oracles existent
     sur les assets versionnes : `internal/assets/static/static_test.go`,
     `internal/service/replay_map_background_test.go` (« L'ORACLE SUR LES ASSETS VERSIONNES »),
     `internal/service/replay_vehicle_labels_test.go`, `apps/web/src/lib/staticAssets.test.ts`,
     `weaponFullIcon.guard.test.ts`, `replaySoundAssets.guard.test.ts`.
- Consequence reelle reformulee : onze chaines de fabrication ne sont documentees que dans les
  notes de chantier `.ai/` et pas dans `docs/` — un confort de reprise a ameliorer, pas la
  violation d'une regle du depot ni un risque d'asset perime silencieux.

## Constat 5 — `vehicle-sprite` mal classe au registre : REFUTE

- Ce que j'ai verifie :
  - `git ls-files "*static/vehicles-assets*" | wc -l` = **20** exact ; consommateurs reels
    confirmes (`useReplayVehicles.ts`, `assets/static/layout.go`,
    `analysis/replay/vehicle_families.go`, + 6 autres).
  - `.ai/V7.5/REGISTRE_REPORTS.md:12` **en entier** (l'auditeur n'en cite qu'un fragment) :
    « outillage de recherche (sons, mesures, sprites) sans consommateur de production ; leur
    statut (**outil maintenu** / a supprimer **apres extraction des assets**) appartient a
    l'auteur du chantier | **decision de l'auteur : garder** avec un en-tete "outil de recherche,
    hors production" **ou supprimer** (regle 7) ».
- Ce que l'auditeur n'a pas vu :
  1. **« Sans consommateur de production » qualifie l'OUTIL, pas sa sortie** — aucun code de prod
     n'invoque `cmd/vehicle-sprite` (mesure : 0 reference, l'auditeur la fournit lui-meme). Rien
     dans la phrase ne parle des fichiers produits.
  2. **La clause suivante nomme explicitement la production d'assets** : « a supprimer **apres
     extraction des assets** ». Le registre sait donc parfaitement que ces outils extraient des
     assets — c'est meme la condition qu'il pose. L'auditeur coupe la citation avant.
  3. **« Suivre sa condition de reprise ferait supprimer le producteur des sprites » est faux** :
     la condition offre « **garder** … ou supprimer », dans cet ordre, et delegue la decision a
     l'auteur. De meme « la condition proposee est inapplicable a un outil de classe (b) » est
     faux : « garder avec un en-tete "outil de recherche, hors production" » s'applique
     parfaitement a un outil de fabrication.
- Consequence reelle reformulee : le registre classe les trois outils par leur absence
  d'appelant de production, prevoit nommement l'extraction d'assets et laisse le choix de garder
  — il n'y a pas de qualification erronee a corriger.

## Constat 6 — troisieme implementation de la sentinelle memoire : TIENT (gravite a RELEVER)

- Ce que j'ai verifie, en cherchant la justification qui sauverait les copies :
  - **`internal/filmproc` n'a AUCUN import projet non-stdlib** (`rg "levelup/go-api"
    internal/filmproc/*.go` hors tests = 0). Aucun cout de dependance.
  - **`cmd/levelup/backfill_memlimit.go:48` importe deja `levelup/go-api/internal/filmproc`** et
    l'appelle a `:133-134` (`filmproc.EmitPeak(v)`, `os.Exit(filmproc.CodeMemory)`) — alors que
    l'en-tete du MEME fichier (repris a `cmd/replay-worker/memlimit.go:22-27`) affirme :
    « Go ne permet pas d'importer un paquet main. **Factoriser exigerait d'ouvrir un paquet
    interne partage** … la duplication mesuree est le cout accepte ». Le paquet est ouvert, et
    le fichier qui porte la justification en depend deja.
  - Idem cote ouvrier : `cmd/replay-worker/job.go:30` importe `internal/filmproc`
    (`filmproc.AcquireSoloWait` a `:283`).
  - **Le contrat de `filmproc.Arm` prevoit nommement les deux doctrines** :
    `internal/filmproc/memguard.go:12-15` — « L'ARRET N'EST PAS DECIDE ICI, ET C'EST CE QUI REND
    LE PAQUET PARTAGEABLE. La sentinelle APPELLE un callback ; … l'enfant d'une passe hors ligne
    rend un code de sortie a son parent, l'ouvrier rapporte d'abord au serveur puis s'arrete.
    Les deux doctrines divergent legitimement ; la mesure de tas, elle, est la meme. »
  - **Les deux doctrines sont deja servies par `filmproc.Arm` ailleurs** : code de sortie enfant
    → `cmd/replay-build/main.go:156`, `cmd/replay-equiv/child.go:83` ; passe de PRODUCTION au
    plafond par defaut → `internal/replaychild/replaychild.go:103`
    (`filmproc.Arm(outilPostSync, filmproc.DefaultLimitGiB, …)`). Plus `zone-attribution`,
    `oddball-terrain`, `statnames-sweep` et une quinzaine de tests.
  - Dates : `memguard.go` cree le **2026-08-27** (`fefa158c1`) ; `backfill_memlimit.go` modifie
    le **2026-09-03** (`aa694442f`), une semaine apres, sans migration.
  - Le garde-rail `internal/archlint/no_unbounded_film_loop_test.go:168-172` autorise
    explicitement les deux copies « avec leur doctrine d'arret propre documentee sur place » —
    il valide donc une justification que le code dement.
- Ce qui confirme : je n'ai trouve aucune incompatibilite de contrat, aucun cout de dependance et
  aucune justification datee qui resiste. La seule justification ecrite est refutee par le bloc
  d'import de son propre fichier.
- Consequence reelle reformulee : trois exemplaires du meme double plafond, dont deux maintenus
  au nom d'un obstacle technique qui n'existe plus et que les fichiers concernes franchissent
  deja — et le garde-rail entérine l'erreur au lieu de l'attraper.

## Constat 7 — douze commandes, 20 308 L, zero reference : REFUTE

- Ce que j'ai verifie (recompte integral, ligne a ligne) :
  ```
  vs-measure  1987 refs=0 | vehicle-sprite  1367 refs=0 | weapon-sounds 12709 refs=2
  mapplafond-mesure 1414 refs=0 | mapfond-inventaire 699 refs=0 | mapfond-planche 519 refs=0
  repair_psa_index 506 refs=0 | recompute_perfnote 480 refs=0 | mapfond-cadrage 246 refs=0
  diag_film_avail 224 refs=0 | diag_matchstats_dump 127 refs=0 | diag_deaths 30 refs=0
  TOTAL 20308
  ```
  Les lignes et la somme sont exactes. **La premisse ne l'est pas.**
- Ce que l'auditeur n'a pas vu — dans sa propre sortie :
  1. **`cmd/weapon-sounds` a 2 references hors `.ai/`** :
     `apps/web/src/features/match-replay/replaySound.ts:359` et `weaponSoundLogic.ts:8`.
     Sa **propre** ligne 19 du tableau l'ecrit (« 2 (prose web) ») — puis il le range parmi les
     « douze … aucune reference hors `.ai/` ». Ce sont **onze**, pas douze.
  2. **L'erreur emporte 63 % du chiffre.** `weapon-sounds` pese 12 709 des 20 308 L. Le total
     reel des commandes a zero reference est **7 599 L**, soit **8 %** de `cmd/` — pas 22 %.
     Surestimation d'un facteur 2,7, reprise telle quelle dans « Chiffres mesures » et dans la
     « Vue d'ensemble ».
  3. La consequence « rien n'empeche de lancer » ignore les contraintes ecrites :
     `cmd/recompute_perfnote/main.go:5` « Outil jetable, **a executer SERVEUR ARRETE** (modele
     mono-process : un writer par …) » ; `cmd/diag_perfsim/main.go:10` « Outil JETABLE :
     **aucune ecriture** (DB ouvertes en `access_mode=read_only`) » — contredit « diag_perfsim …
     que rien n'empeche de lancer » et le rangement en outil a risque.
  4. Trois des douze (`vs-measure`, `vehicle-sprite`, `weapon-sounds`) sont l'entree
     `REGISTRE_REPORTS.md:12` datee 2026-09-05 avec condition de reprise (cf. constats 2 et 5) :
     des reports assumes, pas des oublis.
- Consequence reelle reformulee : onze commandes totalisant 7 599 lignes n'ont pas d'appelant
  hors `.ai/`, dont trois deja inscrites au registre — le stock de sondes a ranger est reel mais
  trois fois plus petit que ce que le constat annonce.

## Constat 8 — `.golangci.yml` exempte `^cmd/` en bloc : TIENT

- Ce que j'ai verifie :
  - `apps/go-api/.golangci.yml:152-154` (commentaire « cmd/ (CLI tools / scripts ponctuels) :
    permissifs sur tout / Les scripts de diag/migration/seed/repair n'ont pas vocation a etre
    production-grade (utilises rarement, manipules manuellement) ») puis `:155`
    `- path: "^cmd/"` desactivant **12** linters : `funlen, lll, gocyclo, noctx, goconst,
    errcheck, prealloc, revive, unused, unparam, ineffassign, staticcheck`. Exact.
  - Binaires de production couverts, verifie sur pieces : `cmd/server` construit et deploye par
    `.github/workflows/release.yml:62` (et `ci.yml:554`, `Makefile:132`) ;
    `cmd/replay-worker` par `scripts/deploy-worker.sh:100` et
    `packaging/systemd/levelup-worker.service:59` (`ExecStart=/opt/levelup/bin/replay-worker …`).
- Ce que j'ai cherche pour refuter, et qui ne sauve pas la configuration :
  - **Une exemption plus fine existe bien** — `:211` (`cmd/server/main.go` : `funlen`, `gocyclo`)
    et `:246` (`cmd/levelup/main.go`) — mais elle est **morte** : `^cmd/` a `:155` couvre deja
    ces deux linters sur ces fichiers. Leur presence montre au contraire qu'une intention plus
    fine a existe avant d'etre avalee par la regle en bloc.
  - **Aucune baseline lint par fichier n'existe pour `cmd/`.** Le gate est un ratchet
    (`golangci-lint run --timeout 5m --new-from-merge-base=origin/main`, `Makefile:307` et
    `:237`) avec « la dette baseline (~479 issues gelees) » (`Makefile:224`). L'exemption ne
    protege donc pas seulement du legacy : elle desarme `errcheck`, `staticcheck` et `unused`
    sur le **code neuf** des deux binaires de production.
- Consequence reelle reformulee : le serveur HTTP et l'ouvrier systemd, tous deux en production,
  ecrivent du code neuf sans `errcheck`, `staticcheck` ni `unused`, au nom d'un motif ecrit
  (« utilises rarement, manipules manuellement ») qui ne les decrit pas.

## Bilan : 3 tiennent, 4 refutes, 1 requalifie

| # | Constat | Verdict |
|---|---|---|
| 1 | P0 unique ecrivain `match_objective_events` | **REFUTE** — 2e ecrivain INSERT-only sur le chemin `persist`, vivant en post-sync |
| 2 | `vs-measure` a supprimer | **REFUTE** — report assume au registre (ligne 12, 2026-09-05) avec condition de reprise |
| 3 | Maillon de livraison des sons en Python | **TIENT** — mais `akpk_unpack.py` est deja porte en Go ; 5 scripts residuels, pas 2 |
| 4 | Dix chaines d'assets non documentees | **TIENT, gravite -> P2** — fait exact, « regle enfreinte » inexistante dans le depot |
| 5 | `vehicle-sprite` mal classe au registre | **REFUTE** — citation tronquee ; le registre nomme l'extraction d'assets et offre « garder » |
| 6 | 3e sentinelle memoire | **TIENT, gravite a RELEVER** — la justification est refutee par le bloc d'import du fichier lui-meme |
| 7 | 12 commandes / 20 308 L sans reference | **REFUTE** — 11 et 7 599 L ; `weapon-sounds` a 2 refs, ecrites dans le tableau de G9 |
| 8 | `^cmd/` exempte 12 linters | **TIENT** — exemption fine morte, aucune baseline, mord sur le code neuf de prod |

Deux defauts de methode transverses expliquent trois des quatre refutations :
une **commande de reproduction qui ne peut pas trouver le contre-exemple** (constat 1 :
`rg "WriteMatch\("` cherche des appelants, pas des ecrivains de la table), et des
**citations tronquees ou contredites par leur propre source** (constat 5 : registre coupe avant
« apres extraction des assets » ; constat 3 : `pck_dump.go` dit que le script est porte ;
constat 7 : chiffre dementi par la ligne 19 du tableau de G9). S'y ajoute une **norme citee entre
guillemets qui n'existe nulle part dans le depot** (constats 3 et 4).
