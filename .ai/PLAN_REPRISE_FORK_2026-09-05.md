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

- [ ] A.3.1 Créer `fix/leaderboard-collections-non-nil` depuis `origin/main` à jour.
      NE PAS travailler sur `main` (règle 16), NE PAS quitter le worktree en cours.
- [ ] A.3.2 Cherry-pick `ad3f013d8` **sans committer** (`-n`), puis retirer du staging
      les fichiers hors périmètre : `CLAUDE.md`, `.ai/thought_log.md`,
      `.ai/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md`.
      Restent exactement 3 fichiers : `internal/service/leaderboard_service.go`,
      `internal/service/jsonshape_dto_smoke_test.go`,
      `internal/api/handlers/leaderboard_test.go`.
- [ ] A.3.3 TRADUIRE les commentaires repris : ils sont en ANGLAIS (vérifié — « A scan
      with no rows can return a NIL slice… », le bloc au-dessus de `GetCatalog`, celui
      de `normalizeLeaderboardCatalog`, et les commentaires des deux fichiers de test).
      Règle 1 du dépôt, pas celle de son fork. C'est un item de TRAVAIL, pas une
      relecture : trois blocs dans le service, plus les tests. Aucun renvoi à des tickets
      ou branches qui n'existent pas chez nous.
- [ ] A.3.4 Vérifier que `normalizeLeaderboardCatalog` est bien le point UNIQUE de la
      garantie : les deux chemins servis de `GetCatalog` passent par lui, et le chemin
      d'erreur rend un zéro jamais sérialisé (le handler en fait un 500).
- [ ] A.3.5 Statuer le constat symétrique de notre plan : `LeaderboardResponse.total`
      n'est lu nulle part côté web. Le NOTER en §5 « Découvertes », **ne pas le traiter**
      (zéro fix opportuniste).
- [ ] A.3.6 Mettre à jour `.ai/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md` : M1 et
      l'angle mort du ratchet passent de `[!]` à `[x]`, avec renvoi à ce plan.
- [ ] A.3.7 Entrée `.ai/thought_log.md` (date, décision, résultat, prochaine étape).

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

---

# VOLET B — Films dont les blobs sont expirés : arrêter de courir après un lien mort

**ÉTAT : FERMÉ le 2026-09-05 au Lot 0, sans code — voir §6.**
**Condition de reprise** : rouvrir uniquement si `downloadBlob HTTP 404|410` apparaît dans
les logs (prod : `/opt/levelup/data/logs/*.log*` ; local : `apps/go-api/logs/`), auquel cas
reprendre à B.0.4 avec les ids observés.

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

- [x] B.0.1 Sur le VPS (PRÉVENIR avant toute opération prod, lecture seule), dans
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
- [x] B.0.2 Compter les `match_id` DISTINCTS parmi elles, et combien reviennent sur
      plusieurs passes — c'est la répétition qui prouve la fuite, pas le volume brut.
- [x] B.0.3 Écrire les chiffres en §6 « Mesures », datés, avec la fenêtre d'observation.
- [x] B.0.4 **Tester la prémisse** (décision 2) : parmi ces `match_id`, y en a-t-il UN
      qui apparaît ENSUITE dans `killsource: decodage du film — fin` avec
      `resultat=ecrit` ou `resultat=sans-killfeed` ? Un seul suffit à prouver qu'un 404
      de blob n'est pas définitif. Écrire le résultat (« 0 sur N » ou la liste) en §6.
      **Résultat : 0 sur 0** — aucun `match_id` à tester, le compte de blobs 404/410 est
      nul sur les deux fenêtres (§6). La prémisse n'est ni confirmée ni infirmée : elle
      n'a pas d'occurrence à laquelle s'appliquer.

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

- [!] B.1.1 Ajouter `metricFetchError = "killsource_erreurs_reseau"`, l'utiliser
      ligne 297 ; `killsource_erreurs_decodage` ne couvre plus que le décodage.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.1.2 Statuer les consommateurs de l'ancien agrégat :
      `grep -rn "killsource_erreurs_decodage" apps/ docs/ .ai/` — aucun, ou mis à jour.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.1.3 Test : la passe compte séparément une erreur réseau et une erreur de décodage.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.

**Gate B.1** : `cd apps/go-api && go test ./internal/sync/killcollector/... -count=1` vert,
grep B.1.2 statué.

## B.5 — Lot 2 : le verdict définitif, et son écriture

- [!] B.2.1 Exporter `IsFilmGoneErr` dans `internal/sync/haloclient/halo_client_film.go` :
      `errors.As` sur `*HTTPError` (404/410), puis repli sur `isNotFoundErr`. Commentaire
      RÉÉCRIT en français : pourquoi le prédicat existe ici (manifeste et blobs expirent
      séparément), et pourquoi chez nous seul le cas blob remonte en pratique.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.2.2 Reprendre le test du fork `halo_client_isfilmgone_test.go` (il s'applique
      proprement, vérifié) en retirant la mention `cmd/study-archiver`. Conserver ses cas
      401/403 (refus d'auth ≠ disparition) et `nil`.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.2.3 Dans `collect()`, AVANT le `return OutcomeNoFilm, 0, err` de la ligne 298 :
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
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.2.4 Tests `killcollector` avec un faux `filmChunkFetcher` — liste exhaustive :
      blob 404 → `(OutcomeNoFilm, 0, nil)` · blob 410 → idem · 503 → erreur non nulle ·
      blob 404 avec `ctx` annulé → erreur non nulle (décision 3) · échec de décodage →
      erreur non nulle (ligne 317 inchangée).
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.2.5 Test de la boucle `CollectMatches` (`roster.go:186`) : sur un blob 404,
      `marquerFilm` est appelé et pose `MBitFilmAbsent` ; sur un 503, il ne l'est pas.
      Le harnais à ÉTENDRE est `registry_flags_integration_test.go`
      (`TestRunPostSync_FilmAbsent_PoseLeMarqueurEtDraineLeBacklog` : base DuckDB réelle,
      faux fetcher `filmsTraces`, lecture du bit dans `match_registry`) — il est derrière
      `-tags=integration`, donc ce test ne tourne QU'au gate d'intégration `-p 1`. Le
      test unitaire `registry_flags_test.go` ne couvre que la fonction pure
      `marquerFilmParOutcome`, il ne suffit pas.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.2.6 Vérifier SUR PIÈCES (rouvrir le fichier) que `replayartifacts/backlog.go`
      exclut bien les matchs nouvellement marqués — lire la requête, ne pas supposer.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.

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

- [!] B.3.1 Écrire et VÉRIFIER la procédure de retrait du bit sur un ensemble de matchs
      (`backfill_completed & ~4194304`), avec la requête de contrôle avant/après.
      **Véhicule imposé par le modèle mono-process** (ADR 0013/0016) : PAS de `duckdb`
      CLI en RW pendant que le serveur tient la shared. Deux options, à trancher à cet
      item et à écrire dans le runbook : (a) une commande `cmd/levelup` qui passe par
      `dblease` comme les autres écritures du registre ; (b) serveur arrêté, fenêtre
      annoncée. Vérifier d'abord si un geste équivalent existe déjà
      (`grep -rn "backfill_completed" apps/go-api/cmd/levelup/`) avant d'en créer un.
      La consigner dans le runbook ops qui héberge déjà les gestes DuckDB CLI.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.3.2 Requête de surveillance : nombre de matchs portant `MBitFilmAbsent` avant et
      après la première passe post-déploiement. Un saut anormal = signal de faux positif.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.3.3 Entrée `.ai/thought_log.md` : date, décision (l'asymétrie), mesure du Lot 0,
      résultat observé, prochaine étape.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.3.4 `.ai/V7.5/REGISTRE_REPORTS.md` mis à jour si un item sort en `[!]`.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.3.5 Skill `delivery-checklist` passé avant le commit de clôture.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.

**Gate B.3** : `make gate-push` vert, procédure B.3.1 rejouée à blanc sur une copie.

## B.7 — Observation prescrite après déploiement

Report LÉGITIME (délai d'observation, pas un report de complaisance) :

- [!] B.7.1 À J+7 : `killsource_films_expires` non nul et `killsource_erreurs_reseau` en
      baisse sur les mêmes matchs. Si `killsource_films_expires` reste à 0 alors que le
      Lot 0 comptait des occurrences, le câblage ne prend pas — rouvrir.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.
- [!] B.7.2 À J+7 : le compte de `MBitFilmAbsent` (B.3.2) a augmenté d'un ordre de
      grandeur cohérent avec la mesure du Lot 0, pas davantage.
      Justification : Lot 0 = 0 occurrence sur les deux fenêtres ; clause de sortie du
      plan appliquée le 2026-09-05, volet fermé sans code.

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
  consommateur. Constaté, non traité.
- (B.0, P2) **Un 304 sur un blob est journalisé comme une erreur de téléchargement.**
  `apps/go-api/internal/sync/haloclient/halo_client_http.go`, fonction `downloadBlob`,
  lignes 50-53 (rouvertes le 2026-09-05) :
  `if resp.StatusCode != http.StatusOK {` … `slog.WarnContext(ctx, "halo_api: downloadBlob HTTP error", …)` …
  `return nil, fmt.Errorf("downloadBlob HTTP %d", resp.StatusCode)`.
  Toute réponse non-200 est donc traitée comme un échec, et c'est bien le statut **304**
  qui domine les échecs observés : 17 fois en prod, 5 fois en local (§6). Un 304
  « Not Modified » sur un blob CDN n'est pas un échec de téléchargement. Constaté, NON
  TRAITÉ ici (zéro fix opportuniste hors périmètre) : à qualifier dans un lot dédié —
  déterminer d'abord si ce 304 vient d'un en-tête conditionnel que nous envoyons, puis
  décider s'il doit être un succès (corps réutilisé) ou un cas neutre non journalisé en
  WARN.
  **Traité depuis, par le VOLET C** (ouvert le 2026-09-05 sur cette découverte) : nous
  n'envoyons aucun en-tête conditionnel, le 304 est un artefact d'edge, il est désormais
  RETENTÉ (avec `Cache-Control: no-cache`) au lieu d'être un verdict d'échec.
- (C.3.8-i) **Le paquet `sync` garde une copie TEXTUELLE de `isNotFoundErr`** :
  `apps/go-api/internal/sync/halo_client_test_helpers_test.go` (helper de test, exercé par
  `sync_coverage_extra_test.go`, qui lui envoie `errors.New("HTTP 404")`). Elle est
  indépendante du prédicat de production, désormais typé. Décision C.2.7 : hors périmètre,
  constaté, NON TRAITÉ.
- (C.3.8-ii) **Prod `main` : `weapon_kills: erreur match` retente les MÊMES 5 matchs des
  milliers de fois** (`BackfillWeaponKillsForMatch film(5da6fd30…)` × 1 909, `35f5cdea…`
  × 1 274 sur la fenêtre du §6). Vérifié sur pièces le 2026-09-05 dans ce worktree : le
  producteur de ces lignes n'existe plus sur `feat/v75` (il ne reste que des mentions en
  commentaire — `service/backfill_orchestrator.go:158`, `sync/events_replay.go:249`,
  `sync/haloclient/halo_client_film.go:140`). Constaté, NON TRAITÉ : le code est mort côté
  v7.5, la fuite s'éteindra au déploiement de la v7.5.
- (C.3.3, découvert à l'exécution) **La décision 4 se trompait sur un point de fait** :
  `contains` / `containsStr` n'étaient PAS morts une fois `isNotFoundErr` typé — `isAuthErr`
  (`halo_client_career.go:222-228`) les utilisait aussi, pour « HTTP 401 » / « HTTP 403 ».
  La décision a été appliquée quand même (helpers SUPPRIMÉS, règle 7) en basculant
  `isAuthErr` sur `strings.Contains` : sémantique strictement identique (`contains` était
  une réimplémentation à la main de `strings.Contains`), zéro changement de comportement.
  `isAuthErr` reste donc un prédicat TEXTUEL — 401/403 sont hors de la liste de la
  décision 4, et le garde-rail ne les couvre pas. Constaté, NON TRAITÉ ici : fermer aussi
  ce repli-là (doGet rend déjà un `*HTTPError` typé sur 401/403) est un lot à part.
  **TRAITÉ depuis, en ronde 1 de revue (C.3-r1.P2-h)** : `isAuthErr` est supprimé,
  `doPlayerGatedGet` passe par `IsAuthError`, et le garde-rail couvre 401/403 et les
  indirections.

## 5.1 — Ronde 1 de revue adversariale du volet C (2026-09-05) : les 11 constats

Deux relecteurs indépendants, périmètre `sync/`. Triage complet, aucun constat sans suite.

**Recevables et CORRIGÉS dans le lot C.3-r1** (8) : P1-a échec de lecture du corps traité
comme fatal · P1-b caractère collant du `no-cache` sans garde-rail · P2-a backoff après la
dernière tentative (temps perdu + abandon masqué par un `ctx` expiré) · P2-b `Retry-After`
ignoré sur les blobs · P2-c corps non drainé sur un statut non-200 (connexion jetée au lieu
d'être rendue au pool) · P2-d commentaire du champ `Attempts` faux · P2-e table
`retryableBlobStatus` couverte à moitié · P2-f 200 au corps non-zlib sans test · P2-g borne
basse de `retryBaseDelay` perdue au passage `const` → `var`. (Le décompte des corrections
est de 10 avec P1-b et P2-h, qui portent aussi du code.)

**Amendement de décision** (1) : P2-h, `isAuthErr` textuel exempté du garde-rail sans que
ce soit écrit ni daté → décision C.2.7 amendée, prédicat supprimé (voir C.2.7 et C.3-r1).

**Consignés, NON TRAITÉS** (2) — hors périmètre de la ronde, à reprendre dans un lot dédié
à `doGet` :
- (r1, `doGet`) **Même défaut de backoff terminal** : la boucle de `doGet`
  (`halo_client_http.go`) appelle `c.backoff` sur sa DERNIÈRE itération, avant de sortir
  vers le log `GET échec définitif` — jusqu'à 6,4 s de sommeil qui ne précèdent aucune
  tentative, sur CHAQUE appel API épuisé. `doGet` est explicitement hors périmètre du volet
  C (décision C.2.7) : constaté, NON TRAITÉ.
- (r1, `doGet`) **Corps non drainé sur les statuts retentés** : `doGet` lit le corps par
  `io.ReadAll` puis `Close` — il draine donc bien le corps dans le cas nominal (contrairement
  à l'ancien `downloadBlob`), SAUF quand `readErr != nil`, où la connexion est perdue. Effet
  marginal (une lecture en erreur signifie déjà une connexion cassée) : constaté, NON TRAITÉ.

# 6. Mesures

**Mesure du Lot 0 (B.0.1 à B.0.4), faite le 2026-09-05 en lecture seule.** Aucune ligne
`downloadBlob HTTP 404` ni `downloadBlob HTTP 410` n'existe, sur aucune des deux fenêtres.

| Périmètre | Fenêtre | Fichiers lus | `downloadBlob HTTP 404\|410` | `downloadBlob` au total | Statuts HTTP observés sur ces échecs |
|---|---|---|---|---|---|
| **Prod** (VPS, `/opt/levelup/data/logs`, binaire = `main` `98bd7c143`, v7.3.1) | 2026-06-13T22:08Z → 2026-09-05T18:54Z | TOUS les `*.log*` : 34 fichiers (general / sync / auth / provider + archives `.1` … `.3`) | **0** | 553 (general.log 29, general.log.1 18, sync.log 26, sync.log.1 480) | **304 × 17, 502 × 2** — aucun 404, aucun 410 |
| **Local** (poste de dev, code `feat/v75`), `logs/` | general.log 2026-05-20 → 2026-09-04 | `logs/` | **0** | — | 304 × 4, 502 × 1 |
| **Local** (poste de dev, code `feat/v75`), `apps/go-api/logs/` | general.log 2026-05-20 → 2026-09-05 | `apps/go-api/logs/` | **0** | — | 304 × 1 |

Répartition des messages sur les 553 lignes `downloadBlob` de prod :
`halo_api: downloadBlob échec réseau` 471 · `weapon_kills: erreur match` 63 ·
`halo_api: downloadBlob HTTP error` 19.

- **B.0.1** — `killsource: decodage du film` + `echec` en prod : **0**. Attendu : le
  collecteur v7.5 n'est pas déployé en prod (binaire `main` 98bd7c143, v7.3.1).
  `downloadBlob HTTP 404|410`, tous messages confondus : **0**.
- **B.0.2** — `match_id` distincts concernés : **0** (aucune occurrence à ventiler), donc
  aucun match revenant sur plusieurs passes. La fuite décrite en B.1 n'a aucune trace
  mesurée.
- **B.0.4** — **0 sur 0** : aucun id à tester. La prémisse « un 404 de blob est définitif »
  n'a pas d'occurrence à laquelle s'appliquer.
- **Local, chaîne v7.5 réellement exercée** : `apps/go-api/logs/general.log` porte 5 952
  lignes `killsource`, dont **1 860 passes `decodage du film — debut` / 1 860 `— fin`, et
  0 `— echec`** (résultat `sans-killfeed` × 1 860). Le chemin du volet B a donc tourné
  1 860 fois sans jamais produire l'échec qu'il visait.

**Conclusion (clause de sortie du Lot 0).** Le compte est 0 sur deux fenêtres
significatives — près de trois mois en prod, trois mois et demi en local. La prémisse du
volet B (un manifeste vivant dont les blobs rendent 404/410) n'a été observée nulle part.
Le volet se ferme ici, sans code : tous les items des lots 1, 2, 3 et de l'observation
prescrite passent en `[!]`. Ils restent écrits, avec leurs décisions déjà tranchées, pour
être repris tels quels le jour où la condition de reprise (en tête du volet B) se réalise.

---

# VOLET C — `downloadBlob` : le 304 du CDN n'est pas un échec, et un blob ne se retente jamais

> Ouvert le 2026-09-05 à la demande de l'utilisateur, sur la découverte P2 du volet B.
> **Branche** : worktree dédié `LevelUp-wt-blob-304`, branche `wt/blob-304-retry` depuis
> `feat/v75`, lots = commits, merge dans `feat/v75`. Revue adversariale OBLIGATOIRE à la
> fin (`sync/`, 2 relecteurs).

## C.1 — Le constat, sur pièces (2026-09-05)

[halo_client_http.go:50-53](../apps/go-api/internal/sync/haloclient/halo_client_http.go#L50-L53) :
`downloadBlob` traite TOUT statut non-200 comme un échec définitif de la tentative, sans
aucun retry — alors que `doGet`, dix lignes plus bas, retente les 5xx et les échecs réseau
avec `c.backoff`. Et il rend une erreur **texte** (`fmt.Errorf("downloadBlob HTTP %d")`),
ce qui oblige `isNotFoundErr` ([halo_client_film.go:365-370](../apps/go-api/internal/sync/haloclient/halo_client_film.go#L365-L370))
à faire du `contains(s, "HTTP 404")` sur des chaînes.

Ce que les logs montrent (mesure du 2026-09-05, prod + local) :
- Statuts d'échec des blobs sur trois mois : **304 × 22** (17 prod, 5 local), 502 × 3,
  « échec réseau » × 471. Jamais de 404/410.
- Les 304 frappent **toujours les mêmes 5-6 films**, à répétition pendant des mois
  (`9bdc0275…/filmChunk16` : 15/06, 22/07, 04/08, 22/08 en prod, 12/07 en local ;
  `dbdbdee9…` : 12/08, 16/08, 02/09 ; `25ede6f8…` : 15/06, 16/06, 04/07, 05/09), chaque fois
  en 0,6-2,8 s. Le match derrière est redemandé à chaque passe de rattrapage, et
  échoue à chaque fois sur ce 304.
- **Le blob est VIVANT.** Sondé le 2026-09-05 à 19:00Z, sans aucun token :
  `HEAD` → `200`, 870 413 octets, `ETag "0x8DD721A77E0269A"`, `Last-Modified` 02/04/2025,
  `Cache-Control: public, max-age=604800, s-maxage=3600`, `x-azure-ref` (Azure Front
  Door), `x-ms-last-access-time` = le matin même à 03:01 (une passe de rattrapage l'a
  touché). `GET` avec `Range: 0-0` → `206`.
- L'URL n'a **aucune query string** : ce ne sont pas des liens pré-signés, le CDN est
  **public et non authentifié** (`https://blobs-infiniteugc.svc.halowaypoint.com/ugcstorage/film/<asset>/<version>/filmChunkNN`).
- Notre requête n'envoie aucun en-tête conditionnel (`If-None-Match` / `If-Modified-Since`
  : 0 occurrence dans `internal/sync`). Un 304 à une requête inconditionnelle est un
  **artefact du CDN** (revalidation d'edge mal propagée), pas une information sur le blob.

Conséquence produit : un film vivant est déclaré illisible à chaque passe, pour toujours,
à cause d'un statut que nous n'avons jamais demandé. Et comme `fetchFilmChunks` abandonne
TOUS les autres chunks à la première erreur (errgroup), un seul 304 sur un chunk coûte le
film entier.

## C.2 — Décisions TRANCHÉES

1. **Un blob se retente, avec la même politique que `doGet`** : jusqu'à `maxRetries`
   tentatives, `c.backoff(ctx, attempt, retryAfter)` entre deux. Liste FERMÉE des statuts
   retentés : `304, 408, 429, 500, 502, 503, 504`, plus les échecs réseau (`c.http.Do`
   en erreur — aujourd'hui non retentés dans `downloadBlob`, alors que `doGet` le fait).
   Pas de retry : `404, 410` (absent), `401, 403` (n'a aucun sens sur un CDN public, et ne
   changera pas), tout autre 4xx.
2. **Sur retry après un 304, la requête porte `Cache-Control: no-cache`** (forcer la
   revalidation à l'origine chez Front Door). Un seul en-tête, posé UNIQUEMENT dans ce
   cas — pas sur la première requête, pas sur les autres statuts.
3. **Erreur TYPÉE, et d'un type DISTINCT de `*HTTPError`** :
   `BlobHTTPError{StatusCode int; URL string; Attempts int}` dans `haloclient`. Distinct
   PARCE QUE le CDN est non authentifié : `notifyPoolOnError` marque un token `unhealthy`
   sur `*HTTPError` 401/403 et pose un cooldown global sur 503 — un 403 ou un 503 du CDN
   ne doit ni poisonner un token valide ni geler le pool de l'API Halo. `errors.As` sur
   `*HTTPError` ne DOIT PAS matcher un `*BlobHTTPError`. Message :
   `downloadBlob HTTP <code> après <n> tentative(s): <url>`.
4. **`isNotFoundErr` passe au typé et perd son repli textuel** : `errors.As` sur
   `*HTTPError` ET `*BlobHTTPError`, statut 404/410. Les helpers `contains` /
   `containsStr` deviennent morts → **supprimés** (règle 7). Garde-rail : un test
   `haloclient` qui interdit, hors `_test.go`, les littéraux `"downloadBlob HTTP %d"`,
   `"HTTP 404"`, `"HTTP 410"` et `"ressource absente"` en position de PRÉDICAT — le
   texte ne redevient jamais une API.
5. **Mesurable** : `slog.DebugContext` à chaque retry (`url`, `status`, `attempt`),
   `slog.WarnContext` à l'abandon (`url`, `status`, `attempts`), et
   `slog.InfoContext(ctx, "halo_api: downloadBlob 304 puis succès", "url", "attempts")`
   quand un retry réussit APRÈS un 304 — c'est cette ligne qui dira à J+7 si l'artefact
   CDN cède au retry. Deux compteurs `observability.AddInt` :
   `halo_api_blob_retry_success`, `halo_api_blob_retry_exhausted` (vérifier la
   convention de nommage des compteurs existants dans `internal/observability` avant de
   les poser, et s'y conformer si elle diffère).
6. **`retryBaseDelay` devient surchargeable en test** (var de paquet, ou champ du client
   — choisir le moins invasif ; `TestBackoff` à `halo_client_extra_test.go:220` doit
   rester vrai). Sans ça, un test « 4 tentatives épuisées » dure 12 s.
7. **Hors périmètre** : le paquet `sync` garde sa copie de test `isNotFoundErr`
   (`halo_client_test_helpers_test.go`, testée par `sync_coverage_extra_test.go`) — à
   noter en §5, pas à toucher. Le comportement errgroup « premier échec = film abandonné »
   reste (les retries vivent DANS `downloadBlob`). `doGet` n'est pas modifié.
   `backfill_weapons.go` (code `main` uniquement, table supprimée sur v7.5) : rien.

   **Amendé le 2026-09-05 en ronde 1 de revue** (le texte d'origine ci-dessus reste, il
   dit ce qui a été décidé avant la revue) : le point « le paquet `haloclient` garde son
   prédicat textuel `isAuthErr` » NE TIENT PLUS. Les deux relecteurs ont constaté que le
   garde-rail `no_text_predicate_test.go` prétendait interdire le prédicat textuel sur les
   erreurs HTTP tout en exemptant, sans l'écrire ni la dater, la seule occurrence restante
   — et que le message de `BlobHTTPError.Error()` contient précisément « HTTP 401 » /
   « HTTP 403 », donc qu'un 403 du CDN PUBLIC pouvait faire dégrader une réponse
   player-gated en « token insuffisant ». Décision amendée : `isAuthErr` est SUPPRIMÉ,
   `doPlayerGatedGet` appelle `IsAuthError` (`errors.As` sur `*HTTPError`, vérifié sur
   pièces : `doGet` rend bien un `*HTTPError` typé sur 401/403), et le garde-rail couvre
   désormais « HTTP 401 » / « HTTP 403 » ET les indirections (littéral interdit nommé par
   une `const`/`var` puis passé à un `strings.Contains`). Reste hors périmètre, inchangé :
   la copie de test `isNotFoundErr` du paquet `sync`, l'errgroup, `doGet`.

## C.3 — Lot unique, items

- [x] C.3.1 Rouvrir `halo_client_http.go`, `halo_client_film.go:340-350` (consommateur
      highlight) et `:240-270` (phase 2 errgroup), `pooled_client.go:84-140`, avant
      d'écrire une ligne — le code a pu bouger depuis la mesure.
      **Fait le 2026-09-05, tout est conforme à la mesure** (`downloadBlob` sans retry
      lignes 24-71, errgroup inchangé, `notifyPoolOnError` lignes 101-134). Un écart
      trouvé, hors des lignes citées : `contains` a un SECOND appelant, `isAuthErr`
      (`halo_client_career.go:227`) — cf. §5 et C.3.3.
- [x] C.3.2 `BlobHTTPError` + boucle de retry dans `downloadBlob` (décisions 1, 2, 3, 5).
      `downloadBlob` reste ≤ 80 lignes : extraire `retryableBlobStatus(code) bool` et, si
      besoin, la lecture/inflation du corps dans une sous-fonction.
      **Fait** : `BlobHTTPError`, `retryableBlobStatus`, `blobAttempt`/`fetchBlobOnce`
      (une tentative + en-tête `Cache-Control: no-cache` après un 304), `inflateBlob`,
      `blobAbandon`. `downloadBlob` = 52 lignes, `halo_client_http.go` = 377 lignes.
      Compteurs `halo_api_blob_retry_success` / `halo_api_blob_retry_exhausted` : la
      convention `internal/observability` (« `<categorie>_<sous_cle>` en snake_case »,
      commentaire de `IncCounter`) est respectée telle quelle, les noms du plan sont
      gardés. Messages de log inchangés là où les mesures du §6 les grepent
      (`downloadBlob échec réseau`, `downloadBlob HTTP error`).
- [x] C.3.3 `isNotFoundErr` typé, `contains`/`containsStr` supprimés, garde-rail posé
      (décision 4).
      **Fait** ; `isAuthErr` bascule sur `strings.Contains` pour que la suppression des
      helpers soit possible (même sémantique, cf. §5). Garde-rail :
      `no_text_predicate_test.go` (scan AST des `.go` non-test du paquet, littéral
      interdit en argument d'un test de sous-chaîne ou en comparaison d'égalité).
      VÉRIFIÉ ROUGE une fois, littéral temporaire réintroduit puis retiré :
      `--- FAIL: TestGardeRail_PasDePredicatTextuelSurLesErreursHTTP` /
      `halo_client_film.go:376 : littéral "HTTP 404" utilisé comme prédicat`.
- [x] C.3.4 `retryBaseDelay` surchargeable (décision 6).
      **Fait** : `const` → `var` de paquet (le moins invasif ; aucun autre lecteur, aucun
      usage en contexte constant). Surcharge par le seul helper
      `avecRetryBaseDelayCourt(t)`, qui restaure la valeur de production via `t.Cleanup`.
      `TestBackoff_Short` / `TestBackoff_CancelledContext` restent verts et inchangés.
- [x] C.3.5 Tests `httptest` dans `haloclient`, liste FERMÉE :
      (a) 304, 304, puis 200 → données rendues, 3 requêtes reçues, la 2e et la 3e
      portent `Cache-Control: no-cache`, la 1re non, compteur `retry_success` +1 ;
      (b) 304 × `maxRetries` → `*BlobHTTPError{304, Attempts: maxRetries}`, compteur
      `retry_exhausted` +1 ; (c) 404 → `*BlobHTTPError{404}` en UNE requête (aucun
      retry) ; (d) 403 → une requête, et `notifyPoolOnError` reçoit l'erreur SANS appeler
      `MarkUnhealthy` (test côté `pooled_client_test.go`, avec le mock de pool existant) ;
      (e) 503 → retenté, puis succès ; (f) échec réseau (serveur fermé) → retenté ;
      (g) `ctx` annulé pendant le backoff → retour immédiat, erreur de contexte ;
      (h) `isNotFoundErr` : `*BlobHTTPError` 404/410 → vrai, 304 → faux ; `*HTTPError`
      404 → vrai ; `errors.New("HTTP 404")` → **faux** (le texte n'est plus une API).
      **Fait, les 8 cas** : `halo_client_blob_retry_test.go` pour (a) à (c), (e) à (h) et
      la moitié client de (d) — plus, dans (d), la preuve que
      `errors.As(err, &*HTTPError)` NE matche PAS un `*BlobHTTPError`. Moitié pool de (d) :
      `TestPooledHaloClientNotifyError_BlobHTTPError_NePoisonnePasLePool`
      (`internal/sync/pooled_client_test.go`), sur 401/403/429/503 : ni `MarkUnhealthy`,
      ni `OnHTTPError`, ni `On429ForToken`. (h) inclut aussi le cas ENVELOPPÉ
      (`fmt.Errorf("%w")` de l'errgroup), qui est le vrai chemin de production.
- [x] C.3.6 Les tests existants qui font échouer un blob (`TestGetMatchFilm_DownloadFails`,
      `TestGetMatchFilm_OneChunkFails_ReturnsError`, et tout autre trouvé par
      `grep -rn "downloadBlob\|StatusInternalServerError\|StatusNotFound" internal/sync/haloclient/*_test.go`)
      restent verts ET rapides (< 3 s chacun) : un fixture qui rend 500 est désormais
      retenté — passer à un statut non retenté ou à `retryBaseDelay` court.
      **Fait** : le grep ne rend que ces deux-là plus `spartan_nameplate_diagnosis_test.go`
      (500/404), qui passe par `doGet` et n'est pas touché. Les deux tests prennent
      `avecRetryBaseDelayCourt(t)` : mesuré 0,02 s chacun.
- [x] C.3.7 `go vet`, seuils (fichier ≤ 500 L, fonction ≤ 80 L), aucun `fmt.Println`.
      **Fait** : `go vet ./internal/sync/...` code 0 ; `halo_client_http.go` 377 L,
      `halo_client_film.go` 389 L ; `downloadBlob` 52 L, toutes les nouvelles fonctions
      ≤ 5 paramètres ; 0 `fmt.Println` / `log.Printf` ; `gofmt -l` vide.
- [x] C.3.8 §5 « Découvertes » : (i) la copie de test `isNotFoundErr` du paquet `sync` ;
      (ii) prod `main` : `weapon_kills: erreur match` retente les MÊMES 5 matchs des
      milliers de fois (`BackfillWeaponKillsForMatch film(5da6fd30…)` × 1 909,
      `35f5cdea…` × 1 274) — code absent de v7.5, à ne pas traiter.
      **Fait**, plus une troisième entrée (la décision 4 se trompait sur `contains`).
      (ii) re-vérifié sur pièces ici : il ne reste du producteur que des mentions en
      commentaire.
- [x] C.3.9 Entrée `.ai/thought_log.md` `[2026-09-05]` + statuts de ce volet + skill
      `delivery-checklist`.

**Gate C** — commandes exactes, en série, jamais deux `go` en parallèle :
```
cd apps/go-api && go test ./internal/sync/... -count=1
cd apps/go-api && go vet ./internal/sync/...
cd apps/go-api && go test -tags=integration -p 1 ./... -count=1     # OBLIGATOIRE : sync/ touché
make gate-push
```
Verdict = code de sortie + `grep '^--- FAIL:'` ancré.

**Résultat du gate C — 2026-09-05, worktree `LevelUp-wt-blob-304`, en série** (mutex
respecté : aucune autre commande `go` sur la machine pendant chaque étape) :

| Commande | Code de sortie | Durée | `^--- FAIL:` |
|---|---|---|---|
| `go test ./internal/sync/... -count=1` | 0 | 1 min 20 s | 0 |
| `go vet ./internal/sync/...` | 0 | 11 s | 0 |
| `go test -tags=integration -p 1 ./... -count=1` | 0 | 16 min 28 s | 0 (160 paquets `ok`) |
| `make gate-push` | 0 | 15 min 23 s | 0 (eslint : 0 erreur, 23 warnings préexistants ; baseline 8 586/8 586 présents) |

**RESTE AVANT MERGE dans `feat/v75`** : la revue adversariale annoncée en tête du volet C
(`sync/`, 2 relecteurs, skill `adversarial-review`) n'a PAS été faite — elle n'est pas un
item du lot C.3 et demande des contextes frais. Elle est le dernier verrou avant le merge.

## C.3-r1 — Ronde 1 de corrections après revue adversariale (2026-09-05)

Deux relecteurs indépendants ont rendu 11 constats sur le lot C.3. Triage : 2 P1 et 6 P2
recevables et CORRIGÉS ici, 1 amendement de décision (C.2.7, ci-dessus), 2 constats
consignés NON TRAITÉS (hors périmètre, cf. §5). Liste FERMÉE, aucun autre changement.

- [x] r1.P1-a L'échec de LECTURE du corps (`io.ReadAll` après un 200 : corps coupé,
      `unexpected EOF`, `connection reset`) n'est plus `fatal` : c'est un échec de
      TRANSPORT, retenté comme les autres (DEBUG à chaque retry, compté dans
      `blobAbandon` avec `status == 0`). Restent fatals : la construction de requête
      impossible, et un corps 200 complet non-zlib (`inflateBlob` en erreur — retenter
      une page d'erreur HTML coûterait 4 × 870 Ko). Test
      `TestDownloadBlob_CorpsCoupeEstRetente` (le serveur doit `Flush()` avant d'abandonner,
      sinon l'échec retombe sur le transport et le test ne prouve rien) — VU ROUGE sous
      mutation (`fatal: true` réintroduit) : `downloadBlob read: unexpected EOF`.
- [x] r1.P1-b Caractère COLLANT de `Cache-Control: no-cache` gardé par un test :
      304 → 503 → 200, 3 requêtes, en-tête absent sur la 1re et posé sur les 2 suivantes,
      compteur `retry_success` +1 (`TestDownloadBlob_NoCacheCollantApres304`). VU ROUGE
      sous mutation (`revalider = at.status == 304` au lieu de `revalider || …`).
- [x] r1.P2-a Plus de backoff après la DERNIÈRE tentative (`attempt < maxRetries-1`) :
      un 304 épuisé coûte 5,6 s au lieu de 12 s, et un `ctx` qui expirait pendant ce
      sommeil final ne masque plus l'abandon (le compteur `retry_exhausted` et le WARN
      partaient à la trappe). La sortie de boucle passe TOUJOURS par `blobAbandon`. Test
      déterministe `TestDownloadBlob_AbandonNonMasqueParUnCtxExpire` (transport factice
      qui annule le contexte pile après la 4e tentative) — VU ROUGE sous mutation
      (backoff inconditionnel restauré). `doGet` porte le même défaut : NON TRAITÉ, §5.
- [x] r1.P2-b `Retry-After` honoré sur les blobs : `fetchBlobOnce` le lit via le
      `parseRetryAfter` existant et le rend dans `blobAttempt`, la boucle le passe à
      `c.backoff` (déjà plafonné par `backoffCeiling`). Test
      `TestDownloadBlob_RetryAfterHonore` (429 + `Retry-After: 1` puis 200, borne BASSE
      mesurée entre les deux requêtes) — VU ROUGE sous mutation (`c.backoff(ctx, attempt, 0)`).
- [x] r1.P2-c Corps drainé (`io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))`)
      avant la fermeture sur un statut non-200, pour que `net/http` rende la connexion au
      pool au lieu de la jeter. Commentaire d'une ligne sur le pourquoi ; pas de test
      (non observable simplement).
- [x] r1.P2-d Commentaire du champ `Attempts` corrigé : « nombre de requêtes réellement
      envoyées, dans 1..maxRetries » — deux 503 puis un 404 donne 3, pas 1. Doc seule,
      aucun changement de code.
- [x] r1.P2-e Table `retryableBlobStatus` couverte EN ENTIER par
      `TestDownloadBlob_TableDesStatuts` : 304/408/429/500/502/503/504 → 2 requêtes et
      succès ; 400/401/403/404/410/418 → 1 requête et `*BlobHTTPError{N, Attempts: 1}`.
      VU ROUGE dans les DEUX sens sous une mutation unique (408 retiré de la liste, 404
      ajouté) : `retente_408` et `definitif_404` rouges.
- [x] r1.P2-f 200 au corps non-zlib : `TestDownloadBlob_200CorpsNonZlib` — 1 requête,
      erreur non nulle, PAS un `*BlobHTTPError`, les deux compteurs à delta 0. VU ROUGE
      sous mutation (erreur d'inflation traitée comme retentable → 4 requêtes).
- [x] r1.P2-g Borne basse de `retryBaseDelay` (`TestRetryBaseDelay_BorneBasse`, ≥ 500 ms) :
      remplace la garantie de compilation perdue au passage `const` → `var`. Lit la
      variable sans surcharge (aucun test du paquet n'est parallèle, `avecRetryBaseDelayCourt`
      restaure via `t.Cleanup`). VU ROUGE sous mutation (`retryBaseDelay = 100ms`).
- [x] r1.P2-h `isAuthErr` SUPPRIMÉ, `doPlayerGatedGet` appelle `IsAuthError` (amendement
      de la décision C.2.7). Garde-rail étendu : « HTTP 401 » / « HTTP 403 » ajoutés aux
      littéraux interdits, et l'INDIRECTION (littéral en initialiseur d'une `const`/`var`)
      est désormais un constat. VU ROUGE sous mutation (`const marqueurTemporaireRevue =
      "HTTP 404"` + `strings.Contains` dans `halo_client_career.go` : `littéral "HTTP 404"
      utilisé comme prédicat (initialiseur de const/var)`), essai retiré. Cas de non-régression
      ajouté à `TestIsAuthError` : un `*BlobHTTPError` 403 (CDN public) n'est PAS une erreur
      d'auth, alors que son message contient « HTTP 403 ».

**Gate C-r1** — mêmes commandes que le gate C, en série, résultats :

| Commande | Code de sortie | Durée | `^--- FAIL:` |
|---|---|---|---|
| `go test ./internal/sync/... -count=1` | 0 | 59 s | 0 |
| `go vet ./internal/sync/...` | 0 | 2 s | 0 |
| `go test -tags=integration -p 1 ./... -count=1` | 0 | 17 min 26 s | 0 |
| `make gate-push` | 0 | 17 min 25 s | 0 (eslint : 0 erreur, 28 warnings préexistants ; baseline 8 586/8 586 présents, run courant 14 625 tests) |

Seuils après ronde 1 : `downloadBlob` = 60 lignes (≤ 80), `halo_client_http.go` = 404
lignes (≤ 500), `gofmt -l` vide.

## C.4 — Observation prescrite (J+7 après déploiement)

- [!] C.4.1 Compter `halo_api: downloadBlob 304 puis succès` : > 0 = l'artefact cède au
      retry, les films `9bdc0275…`, `dbdbdee9…`, `25ede6f8…` doivent être passés.
      = 0 avec des `retry_exhausted` 304 → le 304 est collant côté Front Door ; sujet
      suivant = contourner l'edge (variation d'URL, ou requête directe à l'origine).
      **Report VALIDE** (délai d'observation prescrit, skill `plan-execution` règle 3) :
      la ligne INFO et les deux compteurs n'existent que depuis ce lot, et le seul juge
      est la prod — rien n'est observable avant le déploiement de la v7.5. Inscrit à
      `.ai/V7.5/REGISTRE_REPORTS.md` le 2026-09-05 avec sa condition de reprise.
