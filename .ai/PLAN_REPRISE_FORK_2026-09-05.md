# PLAN — Reprise du fork ChaseWoodhams : deux chantiers indépendants

> Créé le 2026-09-05, après revue sur pièces des branches `feat/study-path-resolution` et
> `fix/leaderboard-ratchet-entries-non-nil` du fork `ChaseWoodhams/LevelUp`.
>
> **Deux volets sans dépendance entre eux, sur DEUX BRANCHES DIFFÉRENTES.** Le volet A va
> sur `main` (c'est là que vit le code visé), le volet B sur `feat/v75`. Ils peuvent être
> menés dans n'importe quel ordre, mais chacun se clôt entièrement avant qu'on ouvre
> l'autre (skill `plan-execution` : une étape à la fois).
>
> Contrat d'exécution : skill `plan-execution`. Skill `delivery-checklist` avant tout
> commit. Statut par item : `[x]` fait · `[~]` couvert ailleurs (référence) · `[!]` non
> traité (justification écrite). Aucune case vide à la clôture.
>
> **Effort** : volet A = rapide (une heure de code, plus `make gate-push` ~25 min) ;
> volet B = moyen (un à deux jours de code, puis sept jours d'observation prescrite).
>
> **Reprise de session** : lire les statuts de ce fichier (le premier item non coché du
> volet en cours est le point de reprise), puis la dernière entrée `.ai/thought_log.md`
> qui cite ce plan. Ne jamais reprendre au milieu d'un lot dont le gate n'a pas été
> rejoué.
>
> Revue du 2026-09-05 (grille `plan-review`) appliquée : décision B.2 réécrite (l'ancienne
> n'était pas implémentable), Lot 0 rendu exécutable, véhicule de la réparation B.3.1
> corrigé (mono-process), journalisation ajoutée en B.2.3, gate A précisé.

**Hors périmètre, définitivement** (décidé avec l'utilisateur le 2026-09-05) : la note
`docs/testing.md` sur la chaîne cgo UCRT64/libstdc++, le commit `45fc3ce52` du fork, et
toute reprise de ses modifications de `CLAUDE.md` (notamment sa règle « English going
forward », décision locale à son fork). Également hors périmètre : `film_paths.go`,
`study_paths.go`, `cmd/study-archiver`, `cmd/study-server`, `apps/study` — voir §4.

---

# VOLET A — Classements : `entries` / `seasons` / `playlists` à `null`

**Branche** : `fix/leaderboard-collections-non-nil`, depuis `main`.
**Source** : commit `ad3f013d8` du fork (le second commit, `45fc3ce52`, est hors périmètre).

## A.1 — Le constat, vérifié sur `feat/v75` et sur `main`

Le bug est intact des deux côtés :

| Pièce | État |
|---|---|
| [domain/leaderboard.go:184-187](../apps/go-api/internal/domain/leaderboard.go#L184-L187) | `Seasons` / `Playlists` sans `omitempty` : un nil sérialise `null`, jamais `[]` |
| [leaderboard_service.go:88-92](../apps/go-api/internal/service/leaderboard_service.go#L88-L92) | `GetCatalog` rend `domain.LeaderboardCatalog{}` pour un titre sans la capability |
| [leaderboard_world_repo.go:512](../apps/go-api/internal/platform/duckdb/leaderboard_world_repo.go#L512) | `scanCatalogColumn` construit sur `var out []…` : base sans snapshot → `seasons: null` |
| [leaderboard_service.go:80-84](../apps/go-api/internal/service/leaderboard_service.go#L80-L84) | `resp.Entries = entries` écrase la garantie non-nil posée à la construction |

Les deux chemins sont RÉELS, pas théoriques : Halo 5 est un titre **actif** qui exclut
`world.leaderboard` (`config/titles/halo_5/title.toml`), et une base fraîche avant le
premier scrape n'a aucun snapshot. Ce sont les constats que notre propre plan
`.ai/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md` avait laissés en `[!]` (M1 + angle mort
du ratchet, qui n'exerçait que le chemin `halo_infinite`).

## A.2 — Ce qui a été vérifié avant d'écrire ce plan

- `git apply --check` du patch `leaderboard_service.go` : **passe** sur `feat/v75`.
- `git apply --check` du patch `jsonshape_dto_smoke_test.go` : **passe** sur `feat/v75`.
- `internal/api/handlers/leaderboard_test.go` : **absent de `feat/v75`**, présent sur
  `main` (arrivé avec `b68cb4c53`, lot 4, jamais mergé dans v75). Sur `main`, le commit
  s'applique donc intégralement.
- Le fork est posé exactement sur `98bd7c143` = tip actuel de `origin/main` : le
  cherry-pick est un **fast-forward**.

## A.3 — Étapes

- [x] A.3.1 Créer `fix/leaderboard-collections-non-nil` depuis `origin/main` à jour.
      NE PAS travailler sur `main` (règle 16), NE PAS quitter le worktree en cours.
- [x] A.3.2 Cherry-pick `ad3f013d8` **sans committer** (`-n`), puis retirer du staging
      les fichiers hors périmètre : `CLAUDE.md`, `.ai/thought_log.md`,
      `.ai/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md`.
      Restent exactement 3 fichiers : `internal/service/leaderboard_service.go`,
      `internal/service/jsonshape_dto_smoke_test.go`,
      `internal/api/handlers/leaderboard_test.go`.
- [x] A.3.3 TRADUIRE les commentaires repris : ils sont en ANGLAIS (vérifié — « A scan
      with no rows can return a NIL slice… », le bloc au-dessus de `GetCatalog`, celui
      de `normalizeLeaderboardCatalog`, et les commentaires des deux fichiers de test).
      Règle 1 du dépôt, pas celle de son fork. C'est un item de TRAVAIL, pas une
      relecture : trois blocs dans le service, plus les tests. Aucun renvoi à des tickets
      ou branches qui n'existent pas chez nous.
- [x] A.3.4 Vérifier que `normalizeLeaderboardCatalog` est bien le point UNIQUE de la
      garantie : les deux chemins servis de `GetCatalog` passent par lui, et le chemin
      d'erreur rend un zéro jamais sérialisé (le handler en fait un 500).
- [x] A.3.5 Statuer le constat symétrique de notre plan : `LeaderboardResponse.total`
      n'est lu nulle part côté web. Le NOTER en §5 « Découvertes », **ne pas le traiter**
      (zéro fix opportuniste).
- [x] A.3.6 Mettre à jour `.ai/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md` : M1 et
      l'angle mort du ratchet passent de `[!]` à `[x]`, avec renvoi à ce plan.
- [x] A.3.7 Entrée `.ai/thought_log.md` (date, décision, résultat, prochaine étape).

**Gate A** — commandes exactes :
```
cd apps/go-api && go test ./internal/service/... ./internal/api/handlers/... -count=1
cd apps/go-api && go test ./... -count=1
make gate-push
```
Plus une vérification manuelle du contrat, sur le serveur local — le titre vient du
contexte (middleware `TitleExtractor`, en-tête `X-LevelUp-Title`), pas du chemin :
```
curl -s -H "X-LevelUp-Title: halo_5" \
  http://localhost:8000/api/v1/players/<slug déclaré>/pages/leaderboard/catalog
```
attendu `{"seasons":[],"playlists":[]}`, jamais `null`. Même appel sans en-tête
(halo_infinite) : les deux tableaux sont présents, vides ou non.

## A.4 — Livraison : ATTENTION

`main` est déployé en prod automatiquement au push. **Prévenir l'utilisateur et obtenir
son accord explicite avant le merge**, conformément à la règle du dépôt. Le merge se fait
sur `main` d'abord, puis `feat/v75` récupère `main` (elle est déjà 9 commits en retard).

Deux pièges connus pour ce merge retour vers `feat/v75`, à anticiper et non à découvrir :
- `.ai/thought_log.md` sera en conflit (entrée du volet A côté `main`, entrées v7.5 côté
  branche) : résolution en BASH sur octets bruts, jamais en PowerShell (double-encodage
  UTF-8).
- `.ai/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md` est **non suivi** (`??`) dans le
  worktree principal de `feat/v75` alors qu'il est suivi sur `main` : git refusera le
  merge tant que la copie locale n'est pas déplacée ou supprimée. Le faire AVANT le merge.
- **CE fichier tombe dans le même piège** : `.ai/PLAN_REPRISE_FORK_2026-09-05.md` est
  committé par le volet A (il est le journal du lot), et il existe aussi en copie NON
  SUIVIE dans le worktree principal de `feat/v75` et dans celui du volet B. Les deux
  copies locales doivent être déplacées ou supprimées avant le merge retour, sans quoi git
  le refusera exactement de la même manière.

## A.5 — Journal d'exécution du volet A

**2026-09-05 — VOLET A CLOS, en attente d'accord utilisateur pour le merge.** Worktree
dédié `LevelUp-wt-leaderboard-nonnil`, branche `fix/leaderboard-collections-non-nil`
ancrée sur `origin/main` = `98bd7c143` (merge-base vérifiée).

- Constats A.1 re-vérifiés sur pièces AVANT de coder : `domain/leaderboard.go:193-194`
  (`Seasons`/`Playlists` sans omitempty), `leaderboard_service.go` (`GetCatalog` rendait
  `domain.LeaderboardCatalog{}` ; `resp.Entries = entries` écrasait la garantie),
  `leaderboard_world_repo.go:504` (`var out []domain.LeaderboardCatalogRef`).
- Cherry-pick `-n` de `ad3f013d8`, 3 fichiers hors périmètre retirés du staging et du
  disque (`CLAUDE.md`, `.ai/thought_log.md`, l'ancien plan leaderboard) : il ne restait
  bien que les 3 fichiers Go (`git status --short` vérifié).
- A.3.3 : les 8 blocs de commentaires anglais repris ont été RÉÉCRITS en français, ainsi
  que les 5 libellés de sous-tests en prose et les 3 messages d'assertion en prose (la
  forme télégraphique `x = %v, want N` du fichier est conservée). Aucun renvoi à un
  ticket ou une branche du fork ne subsiste.
- A.3.4 vérifié sur pièces : `GetCatalog` a un seul appelant
  (`handlers/leaderboard.go:110`), ses deux chemins servis passent par
  `normalizeLeaderboardCatalog`, et son chemin d'erreur rend `nil` + un 500 Huma
  (`leaderboard.go:112`) — la valeur zéro n'est jamais sérialisée. Aucun second
  propriétaire de la garantie : le repo DuckDB ne fait `make(...)` que sur `playlists`.
- **Le garde-rail MORD** (vérifié en neutralisant temporairement le correctif sur une
  copie, puis restauration) : sans lui, 4 des 5 nouveaux sous-tests du ratchet et 3 des
  4 cas HTTP rougissent. Le cas « page, titre sans capability » reste vert des deux
  côtés — il était déjà couvert par la garantie de construction du lot 4.1, c'est
  attendu.

- **Gate A, résultats mesurés** :
  `go test ./internal/service/... ./internal/api/handlers/... -count=1` → 0, 19 s, 0
  `--- FAIL:` · `go test ./... -count=1` → 0, 76 s, 0 `--- FAIL:` (une PREMIÈRE passe avait
  rougi sur un flake `internal/persist`, voir §5) · `go vet ./...` → 0, 5 s ·
  `make gate-push` → vert sur ses 4 étapes ; la dernière
  (`bash scripts/check_test_baseline.sh tests`) a dû être rejouée SEULE, le `make` complet
  dépassant le plafond de 10 min de l'outil d'exécution : sortie 0, 8 761 tests baseline
  tous présents, 11 722 tests au run courant, aucun test ni package en échec ; eslint
  0 erreur / 19 avertissements préexistants.
- **Vérification manuelle du contrat** : binaire du worktree lancé avec
  `LEVELUP_REPO_ROOT` sur le checkout principal (aucun autre serveur sur :8000 — modèle
  mono-process respecté) et `LEVELUP_AUTH_MODE=none` (voir §5).
  `curl -s -H "X-LevelUp-Title: halo_5" .../players/Chocoboflor/pages/leaderboard/catalog`
  → `{"seasons":[],"playlists":[]}`. Le même appel sans en-tête (halo_infinite) → les deux
  tableaux présents, `seasons` peuplé de 4 saisons avec leurs `playlist_ids`. Process
  arrêté et binaire temporaire supprimé, `netstat` ne montre plus aucun LISTENING sur 8000.

**Reste à faire, hors périmètre de l'exécuteur** : A.4 — merge vers `main` (deploy prod
automatique) sur accord explicite de l'utilisateur, puis retour de `main` vers `feat/v75`
avec les deux pièges listés ci-dessus.

---

# VOLET B — Films dont les blobs sont expirés : arrêter de courir après un lien mort

**Branche** : worktree DÉDIÉ `wt/film-blobs-expires`, lots = commits, merge dans
`feat/v75`.
**Origine** : l'observation portée par `IsFilmGoneErr` (commit `4b88ea1a2` du fork). On ne
reprend PAS son câblage — chez lui il sert un archiveur qui n'existe pas ici — mais son
constat désigne un trou réel chez nous.

## B.1 — Le constat, sur pièces

Le manifeste d'un film et ses blobs pré-signés meurent sur **deux calendriers séparés**.
`fetchFilmManifest` convertit déjà un manifeste 404/410 en `found=false` : cas nominal,
traité. Mais un manifeste encore vivant dont les **blobs** rendent 404/410 ressort en
**erreur de transport**, et à partir de là notre chaîne le traite comme une panne
passagère :

| Emplacement | Ce qui se passe |
|---|---|
| [collector.go:296-298](../apps/go-api/internal/sync/killcollector/collector.go#L296-L298) | toute erreur de `FilmChunksForMatch` → `OutcomeNoFilm, 0, err` |
| [roster.go:210-219](../apps/go-api/internal/sync/killcollector/roster.go#L210-L219) | `if err != nil { sum.Errors++; continue }` — **avant** l'appel à `c.marquerFilm(...)` |
| [registry_flags.go:75-79](../apps/go-api/internal/sync/killcollector/registry_flags.go#L75-L79) | `MBitFilmAbsent` n'est donc jamais posé sur ce match |

Conséquence : ce match repart à chaque cycle des rattrapages 1.57/1.58, et il n'aboutira
jamais. Il coûte un aller-retour CDN par passe, indéfiniment.

**Notre sens d'erreur est le bon.** Le commentaire de `roster.go` dit explicitement
pourquoi le marqueur est posé dans la boucle et pas dans `collect` (« seule cette boucle
distingue une erreur d'un outcome »). On n'enterre jamais un film vivant. On paie
simplement l'autre moitié : on court après des morts.

**Deuxième bénéficiaire, gratuit.** `internal/sync/replayartifacts` est un LECTEUR du bit :
[backlog.go:133](../apps/go-api/internal/sync/replayartifacts/backlog.go#L133) exclut du
retard de cuisson les matchs marqués `bitFilmAbsent`, et
[cuisson.go:209-214](../apps/go-api/internal/sync/replayartifacts/cuisson.go#L209-L214)
retente à la passe suivante sur erreur. Un bit posé au bon endroit ferme les deux fuites
avec une seule écriture.

## B.2 — Décisions produit, TRANCHÉES avant l'exécution

Aucune de ces questions ne se rouvre en cours de route.

1. **L'asymétrie commande tout.** Un faux « transitoire » se corrige seul à la passe
   suivante. Un faux `MBitFilmAbsent` est **définitif** : le bit est persisté dans
   `match_registry.backfill_completed` et retire le match des deux rattrapages pour
   toujours. **En cas de doute : transitoire.**
2. **Un seul blob 404/410 = film mort.** Un blob pré-signé qui a disparu ne revient pas,
   et un film incomplet ne se décode pas : le verdict est définitif même si un blob
   voisin rendait 503 au même moment. On ne peut d'ailleurs pas faire autrement :
   `fetchFilmChunks` remonte la PREMIÈRE erreur de son errgroup, `collect()` ne voit
   jamais les autres. (La version initiale de ce plan disait « partielle = transitoire »
   — c'était inapplicable sans réécrire le téléchargeur, ce qui est hors périmètre.)
   **La prémisse « un 404 de blob est définitif » se MESURE au Lot 0 (B.0.4) avant
   d'être codée** : si un match a déjà donné `downloadBlob HTTP 404` puis a été décodé à
   une passe ultérieure, elle est fausse, et le volet B change de forme (garde d'âge
   sur le match — sujet à re-planifier, pas à improviser).
3. **Jamais de verdict définitif sous délai dépassé.** Si `ctx.Err() != nil`, on ne classe
   pas : un abandon sur limite de temps n'est pas une disparition.
4. **Un seul propriétaire du prédicat.** `IsFilmGoneErr` s'APPUIE sur `isNotFoundErr`
   (déjà présent, [halo_client_film.go:365](../apps/go-api/internal/sync/haloclient/halo_client_film.go#L365)),
   il ne le duplique pas. Deux copies de la même règle 404/410 = règle 6 violée.
5. **On ne type pas l'erreur de `downloadBlob`.** Ce serait changer le branchement du POOL
   (cooldowns 429/503, marquage 401/403) : hors périmètre. Le repli textuel reste, confiné
   aux erreurs formatées dans le paquet `haloclient`.

## B.3 — Lot 0 : MESURER (aucun code)

Sans chiffre, on ne sait pas si le problème pèse 3 matchs ou 3 000. La ligne de log existe
déjà et porte `err` : `killsource: decodage du film — echec`.

- [ ] B.0.1 Sur le VPS (PRÉVENIR avant toute opération prod, lecture seule), dans
      `{LogsDir}/general.log` ET ses archives `general.log.1 … general.log.N` — PAS
      `sync.log` : le module est déduit du NOM DE PAQUET (`killcollector`), qui n'est pas
      dans `packageToModuleMap`, et l'heuristique de repli rend `ModuleGeneral`
      (`observability/logging/module.go`, `mapPackageToModule` → `moduleFromPath` →
      `default`). Vérifié sur pièces le 2026-09-05 après une première version fausse de
      cet item. La rotation est PAR TAILLE, donc la fenêtre temporelle est ce que les
      archives couvrent — la lire sur les horodatages de la première et de la dernière
      ligne, et l'écrire avec le chiffre. Compter les occurrences de
      `killsource: decodage du film — echec` dont le champ `err` contient
      `downloadBlob HTTP 404` ou `downloadBlob HTTP 410`.
- [ ] B.0.2 Compter les `match_id` DISTINCTS parmi elles, et combien reviennent sur
      plusieurs passes — c'est la répétition qui prouve la fuite, pas le volume brut.
- [ ] B.0.3 Écrire les chiffres en §6 « Mesures », datés, avec la fenêtre d'observation.
- [ ] B.0.4 **Tester la prémisse** (décision 2) : parmi ces `match_id`, y en a-t-il UN
      qui apparaît ENSUITE dans `killsource: decodage du film — fin` avec
      `resultat=ecrit` ou `resultat=sans-killfeed` ? Un seul suffit à prouver qu'un 404
      de blob n'est pas définitif. Écrire le résultat (« 0 sur N » ou la liste) en §6.

**Gate B.0** : trois nombres écrits et datés (B.0.1, B.0.2, B.0.4).
**Go/no-go de B.0.4** : si un match a ressuscité après un 404, STOP — ne pas entrer au
Lot 2 ; consigner en §5 et revenir à l'utilisateur avec la mesure, le volet doit être
re-planifié (garde d'âge). Un faux `MBitFilmAbsent` étant permanent, ce n'est pas
négociable.
**Sortie possible** : si le compte est 0 sur une fenêtre significative, le volet B
s'arrête ici — clôture dans `.ai/thought_log.md` et `.ai/V7.5/REGISTRE_REPORTS.md`, aucun
code écrit. Si les logs sont rotés et ne permettent pas de conclure, passer au Lot 1 (qui
EST l'instrument) et revenir mesurer après soak.

## B.4 — Lot 1 : rendre le compteur honnête (l'instrument)

`metricDecodeError` (`killsource_erreurs_decodage`) est incrémenté à
[collector.go:297](../apps/go-api/internal/sync/killcollector/collector.go#L297) pour un
échec **réseau** ET à [collector.go:317](../apps/go-api/internal/sync/killcollector/collector.go#L317)
pour un vrai échec de **décodage**. Le compteur ne peut donc ni mesurer le problème, ni
vérifier le Lot 2 après déploiement.

- [ ] B.1.1 Ajouter `metricFetchError = "killsource_erreurs_reseau"`, l'utiliser
      ligne 297 ; `killsource_erreurs_decodage` ne couvre plus que le décodage.
- [ ] B.1.2 Statuer les consommateurs de l'ancien agrégat :
      `grep -rn "killsource_erreurs_decodage" apps/ docs/ .ai/` — aucun, ou mis à jour.
- [ ] B.1.3 Test : la passe compte séparément une erreur réseau et une erreur de décodage.

**Gate B.1** : `cd apps/go-api && go test ./internal/sync/killcollector/... -count=1` vert,
grep B.1.2 statué.

## B.5 — Lot 2 : le verdict définitif, et son écriture

- [ ] B.2.1 Exporter `IsFilmGoneErr` dans `internal/sync/haloclient/halo_client_film.go` :
      `errors.As` sur `*HTTPError` (404/410), puis repli sur `isNotFoundErr`. Commentaire
      RÉÉCRIT en français : pourquoi le prédicat existe ici (manifeste et blobs expirent
      séparément), et pourquoi chez nous seul le cas blob remonte en pratique.
- [ ] B.2.2 Reprendre le test du fork `halo_client_isfilmgone_test.go` (il s'applique
      proprement, vérifié) en retirant la mention `cmd/study-archiver`. Conserver ses cas
      401/403 (refus d'auth ≠ disparition) et `nil`.
- [ ] B.2.3 Dans `collect()`, AVANT le `return OutcomeNoFilm, 0, err` de la ligne 298 :
      si `haloclient.IsFilmGoneErr(err) && ctx.Err() == nil` (le `ctx` reçu est le
      `matchCtx` borné par `CollectMatch` — c'est le bon), alors :
      `slog.InfoContext(ctx, "killsource: film expire cote CDN — marqueur terminal",
      "match_id", matchID, "err", err)` (règle 3 : jamais de dégradation silencieuse,
      et c'est cette ligne que B.7.1 relira), incrémenter un nouveau compteur
      `killsource_films_expires`, et rendre `OutcomeNoFilm, 0, nil` (décisions 1 et 3).
      **Ne RIEN changer aux lignes 317 et 323** : un échec de décodage ou de roster n'est
      pas une disparition de film.
      Effet assumé : ces matchs comptent désormais dans `sum.NoFilm` de la boucle
      `roster.go` (et non plus `sum.Errors`) — le total « traités » est inchangé.
      Précondition VÉRIFIÉE le 2026-09-05, ne pas la re-vérifier : la chaîne d'erreur
      conserve le texte et le type jusqu'à `collect()` — `PooledHaloClient.doPublic`
      rend `callErr` tel quel, `FilmChunksForMatch` enveloppe avec `%w`, `RemoteFilms`
      passe l'erreur sans la toucher.
- [ ] B.2.4 Tests `killcollector` avec un faux `filmChunkFetcher` — liste exhaustive :
      blob 404 → `(OutcomeNoFilm, 0, nil)` · blob 410 → idem · 503 → erreur non nulle ·
      blob 404 avec `ctx` annulé → erreur non nulle (décision 3) · échec de décodage →
      erreur non nulle (ligne 317 inchangée).
- [ ] B.2.5 Test de la boucle `CollectMatches` (`roster.go:186`) : sur un blob 404,
      `marquerFilm` est appelé et pose `MBitFilmAbsent` ; sur un 503, il ne l'est pas.
      Le harnais à ÉTENDRE est `registry_flags_integration_test.go`
      (`TestRunPostSync_FilmAbsent_PoseLeMarqueurEtDraineLeBacklog` : base DuckDB réelle,
      faux fetcher `filmsTraces`, lecture du bit dans `match_registry`) — il est derrière
      `-tags=integration`, donc ce test ne tourne QU'au gate d'intégration `-p 1`. Le
      test unitaire `registry_flags_test.go` ne couvre que la fonction pure
      `marquerFilmParOutcome`, il ne suffit pas.
- [ ] B.2.6 Vérifier SUR PIÈCES (rouvrir le fichier) que `replayartifacts/backlog.go`
      exclut bien les matchs nouvellement marqués — lire la requête, ne pas supposer.

**Gate B.2** — commandes exactes :
```
cd apps/go-api && go test ./internal/sync/... ./internal/domain/... -count=1
cd apps/go-api && go test -tags=integration -p 1 ./... -count=1
```
L'étape intégration est OBLIGATOIRE : on touche `sync`. Pas de `-race` (incompatible
DuckDB).

## B.6 — Lot 3 : filet de réparation et clôture

Le Lot 2 écrit un bit dont l'effet est permanent. On ne livre pas ça sans savoir le
défaire.

- [ ] B.3.1 Écrire et VÉRIFIER la procédure de retrait du bit sur un ensemble de matchs
      (`backfill_completed & ~4194304`), avec la requête de contrôle avant/après.
      **Véhicule imposé par le modèle mono-process** (ADR 0013/0016) : PAS de `duckdb`
      CLI en RW pendant que le serveur tient la shared. Deux options, à trancher à cet
      item et à écrire dans le runbook : (a) une commande `cmd/levelup` qui passe par
      `dblease` comme les autres écritures du registre ; (b) serveur arrêté, fenêtre
      annoncée. Vérifier d'abord si un geste équivalent existe déjà
      (`grep -rn "backfill_completed" apps/go-api/cmd/levelup/`) avant d'en créer un.
      La consigner dans le runbook ops qui héberge déjà les gestes DuckDB CLI.
- [ ] B.3.2 Requête de surveillance : nombre de matchs portant `MBitFilmAbsent` avant et
      après la première passe post-déploiement. Un saut anormal = signal de faux positif.
- [ ] B.3.3 Entrée `.ai/thought_log.md` : date, décision (l'asymétrie), mesure du Lot 0,
      résultat observé, prochaine étape.
- [ ] B.3.4 `.ai/V7.5/REGISTRE_REPORTS.md` mis à jour si un item sort en `[!]`.
- [ ] B.3.5 Skill `delivery-checklist` passé avant le commit de clôture.

**Gate B.3** : `make gate-push` vert, procédure B.3.1 rejouée à blanc sur une copie.

## B.7 — Observation prescrite après déploiement

Report LÉGITIME (délai d'observation, pas un report de complaisance) :

- [ ] B.7.1 À J+7 : `killsource_films_expires` non nul et `killsource_erreurs_reseau` en
      baisse sur les mêmes matchs. Si `killsource_films_expires` reste à 0 alors que le
      Lot 0 comptait des occurrences, le câblage ne prend pas — rouvrir.
- [ ] B.7.2 À J+7 : le compte de `MBitFilmAbsent` (B.3.2) a augmenté d'un ordre de
      grandeur cohérent avec la mesure du Lot 0, pas davantage.

---

# 4. Ce que ce plan ne fait pas — et pourquoi

- **`film_paths.go` du fork : rejeté.** `internal/games/halo_infinite/film/filmcache/` est
  déjà notre propriétaire canonique du chemin de chunks (const `chunksDir`, 17 appelants,
  `filmcache_guard_test.go` avec allowlist datée). Le bug qu'il corrige dans
  `cmd/replay-build` est **déjà corrigé chez nous**, et mieux :
  [replay-build/main.go:81](../apps/go-api/cmd/replay-build/main.go#L81) passe par
  `filmcache.ChunkDir(cacheRoot, …)`, qui accepte une racine de cache arbitraire là où son
  `PathResolver` est ancré sur `repoRoot`. Un second propriétaire serait l'anti-pattern
  « factorisation abandonnée ». Son garde-rail rougirait d'ailleurs immédiatement chez
  nous, avec un faux positif sur
  [pooled_client.go:242](../apps/go-api/internal/sync/pooled_client.go#L242), où
  `"film_chunks"` est un label de métrique et non un chemin.
- **`study_paths.go` : rejeté.** Quatre méthodes sans aucun appelant hors de l'outil
  d'étude = code mort (règle 7).
- **`cmd/study-archiver`, `cmd/study-server`, `apps/study` : hors décision.** Reprendre son
  architecture engage (copie de notre moteur de rejeu dans une seconde app). Sujet séparé.
- **`docs/testing.md`, `CLAUDE.md`, chaîne cgo UCRT64 : écartés** par l'utilisateur le
  2026-09-05.

# 5. Découvertes (à remplir pendant l'exécution — NE PAS corriger)

- (A.3.5) `LeaderboardResponse.total` n'est lu nulle part côté web : champ de contrat sans
  consommateur. Constaté, non traité. **Re-vérifié sur pièces le 2026-09-05** : grep de
  `total` dans `apps/web/src/features/leaderboard/{queries.ts,LeaderboardBlock.tsx}` et
  `features/palmares/PalmaresLeaderboardPage.tsx` → seule occurrence `coverage.total`, un
  autre objet. Le champ reste figé par `TestLeaderboardPage_TotalFieldNameOnTheWire`.
- (volet A, exécuteur) **`internal/persist` est FLAKY sous parallélisme.**
  `TestBatchQueue_Drain_PartialFailure_OnlySomeACKed` a rougi une fois pendant
  `go test ./... -count=1` (« PendingCount = 4, want 2 »), puis passé 3 fois isolé
  (`-count=3`) et 2 fois en package seul, et la re-passe complète est verte. Aucun rapport
  avec le volet A (le diff ne touche ni `persist/` ni `sync/`). Piège connu du dépôt sur
  `-p 1` — non traité.
- (volet A, exécuteur) **La route catalogue est inatteignable en anonyme en local.**
  `LEVELUP_AUTH_MODE=xbox` dans `.env.local` active `RequirePlayerOwnership` : sans
  session, `Chocoboflor` rend 403 `player_forbidden` (« authz: accès joueur refusé »). La
  vérification manuelle du gate a donc tourné avec `LEVELUP_AUTH_MODE=none` passé au
  LANCEMENT du binaire (aucun fichier modifié) — le contrat vérifié est la forme du JSON,
  pas l'authz. À savoir pour toute vérification manuelle future sur ce poste.
- (volet A, exécuteur) Le slug joueur d'URL est le gamertag EXACT, casse comprise
  (`config_players.go:142`, `PlayerSlug: gamertag`) : `chocoboflor` en minuscules donne un
  403 « slug joueur inconnu refusé », pas un 404. Non traité.
- (volet A, exécuteur, hors code) Au démarrage, le serveur a émis
  `disk_watch: espace disque CRITIQUE sur le volume data` pour
  `LevelUp-go-migration\data\titles\halo_infinite`. Environnement, pas le diff — signalé à
  l'utilisateur.
- (volet A, revue adversariale 2026-09-05, contexte frais, lentille L6) **0 constat
  recevable, 14 conditions vérifiées qui tiennent.** Mutation ligne à ligne : chaque `if`
  ajouté est gardé aux deux étages (ratchet + handler). Un P2 pré-existant, hors diff :
  la fixture `csrWorld: []domain.LeaderboardEntry{}` du sous-test
  `jsonshape_dto_smoke_test.go:155` est MORTE — sans saison ni playlist la requête sort
  en amont (`leaderboard_service.go:72-78`) et n'atteint jamais le repo ; le cas « slice
  vide non-nil » n'est donc pas exercé (dominé par le cas nil, sans conséquence
  observable). Non traité.

# 6. Mesures

_(B.0.3 écrit ici)_
