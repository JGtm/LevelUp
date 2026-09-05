# Vérification adverse V-GO-A1

Cadre : lecture seule, dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`,
branche `feat/v75`, HEAD code `736ccf3c3`. Biais assumé du mandat : en cas de doute, RÉFUTÉ.

## Constat 1 — `KillSourceDecoderRev` figé alors que le décodeur a changé : TIENT

- **Ce que j'ai vérifié** :
  - `apps/go-api/internal/sync/killcollector/collector.go:60-65` — contrat écrit
    (« LA FAIRE EVOLUER a chaque changement de decodage qui change les lignes produites »)
    + `const KillSourceDecoderRev = "killsource-2026-07-31"` ; écriture sur chaque ligne à
    `collector.go:474` (`DecoderRev: KillSourceDecoderRev`).
  - `git log --oneline v7.3.0..HEAD -- apps/go-api/internal/games/halo_infinite/film/killsource/`
    → **14 commits**. `git log -G'KillSourceDecoderRev = ' --oneline v7.3.0..HEAD -- .../collector.go`
    → **1 seul**, `36fc76835`, et c'est un déplacement de fichier (« le collecteur va dans son
    sous-paquet »), pas un bump. 0 bump réel.
  - Prédicat de reprise : `apps/go-api/internal/sync/killcollector/postsync.go:369-377`
    (`conditionBacklog`, `NOT EXISTS (… e.decoder_rev = ? AND e.read_path <> ?)`), args
    `postsync.go:399`. Copie CLI : `apps/go-api/cmd/levelup/cmd_backfill_killsource.go:400-404`
    (`matchsAJour`).
  - `git log -1 328d83232` : « HARNAIS D'EQUIVALENCE … **`killsource` bouge sur ces 4 films et
    sur EUX SEULS. Ce dernier ecart est la CORRECTION DECLAREE du lot V9** … le kill-feed des
    matchs a vehicules change dans le sens de la correction. »
- **Ce que j'ai cherché pour réfuter, et qui n'existe pas** :
  - Aucune entrée dans `.ai/V7.5/REGISTRE_REPORTS.md` (grep `KillSourceDecoderRev|decoder_rev|
    revision du decodeur` → 0 ligne) ; aucune décision datée dans `.ai/thought_log.md` (les
    5 occurrences de `decoder_rev` concernent `whd-v1`, un rejeu à venir, une autre table).
  - Aucun échappatoire côté CLI : `cmd_backfill_killsource.go:139-150` n'expose que
    `-title -cache -limit -force -dry-run -films-only -credit-only -online -gamertag -rps`.
    **Pas de `--match-id`** : la seule voie de redécodage ciblé n'existe pas, il ne reste que
    `--force` sur tout le corpus.
  - Aucune migration ni aucun chemin ne remet à zéro `decoder_rev` ou le bit
    `backfill_completed` pour ces matchs (grep sur les 11 sites de `KillSourceDecoderRev` :
    3 non-test, tous en lecture/écriture de la valeur courante).
- **Conséquence réelle** : les lignes déjà en base des 4 matchs à véhicule portent la révision
  courante, le prédicat de backlog les exclut définitivement, et `match_kill_events_latest`
  continue de servir la lecture d'avant la correction V9 à toutes les surfaces qui en dérivent.

## Constat 2 — Les trois dérivations câblées seulement sur la branche « locale » : TIENT (gravité → P2, latent)

- **Ce que j'ai vérifié** :
  - `apps/go-api/internal/sync/replayartifacts/artifacts.go:321-322` : `if d.Placement ==
    replaybuild.PlacementWorker { enqueueAll(ctx, d, work); return }` — le `return` précède
    bien `:344 reporterT0Film`, `:347 persisterResumesUsage`, `:352 persisterStatsBombe`.
  - `apps/go-api/internal/replaybuild/placement.go:78-84` : défaut d'instance en production =
    `worker`. Et `:66-70` : `local` est **refusé** en production (`ErrLocalBuildInProduction`
    → `PlacementOff`). La branche des dérivations est donc **inatteignable en prod par
    construction**, pas seulement « par défaut ».
  - `apps/go-api/internal/api/wire/registry_build_queue.go:207-245` (`StoreBuildArtifact`) :
    valide, range, compte, journalise, `return`. Aucune dérivation.
  - Piste de réfutation la plus sérieuse — le chemin `persist` : `combined_persister.go:128`
    câble bien `NewBombStatsPersister(sharedDB).Persist(ctx, batch)` dans la fenêtre de lease
    du sync primaire. **C'est un NO-OP** : `bomb_stats_persister.go:156-162` sort si
    `batch.Shared.BombStats == nil`, et `grep -v _test` sur `SetBombStats` ne rend **aucun
    appelant de production** (seulement `builder.go:126`, la déclaration). Idem `SetWeaponShots`,
    `SetKillSource`.
  - `apps/go-api/cmd/replay-worker/*.go` : aucune trace d'usage / bombe / T0 — l'ouvrier pousse
    des octets, il n'écrit dans aucune base.
- **Ce que l'auditeur n'a pas vu (et qui abaisse la gravité, sans réfuter)** :
  1. **Le tiers T0-film est DÉJÀ AU REGISTRE**, daté et avec sa condition de reprise :
     `.ai/V7.5/REGISTRE_REPORTS.md` ligne 18 — « **Report T0-film au fil de l'eau limite au
     placement `local`** — le chemin `worker` (defaut PROD) ne cuit rien en process, donc ne
     reporte rien au registre | lot C T0-film, **2026-09-02** | … | Lot dedie : accrocher le
     report a `replaybuild.SetArtifactStoredSink` ». C'est mot pour mot le constat, déjà
     consigné. Par la propre règle de tri de l'audit, ce tiers va en « constats écartés ».
  2. **La limitation est écrite dans le code**, section dédiée :
     `apps/go-api/internal/sync/replayartifacts/usage.go:42-47` — « Seuls les artefacts CUITS
     DANS CE CYCLE … **et le chemin « ouvrier » ne cuit rien localement**. Le corpus déjà sur
     disque relève du backfill CLI (`levelup backfill-usage-summary`), hors ligne et sous le
     contrôle de l'opérateur. » Ce n'est pas un oubli muet.
  3. **Le mécanisme nommé comme correctif EXISTE déjà** :
     `apps/go-api/internal/replaybuild/artifact_events.go:59` (`SetArtifactStoredSink`), câblé au
     boot en `apps/go-api/internal/api/wire/registry_replay_notify.go:57` — mais uniquement vers
     la notification Discord groupée, jamais vers les persisters.
  4. **Le symptôme décrit (« bloc vide permanent en prod ») ne se produit pas aujourd'hui** :
     l'ouvrier distant n'est **pas activé en production** — `.ai/V7.5/PLAN_OUVRIER_DISTANT.md`
     titre : « **ce qui reste avant l'activation** » ; `:383` « COMMANDE DE REMEDIATION — A
     LANCER AVANT LA PREMIERE ACTIVATION PROD DE L'OUVRIER » ; `REGISTRE_REPORTS.md` l. 277
     « l'ACTIVATION reste non faite (garde local, arbitrage utilisateur) ». Sans jeton ouvrier,
     `DecidePlacement` dégrade en `PlacementOff` (`placement.go:71-77`) : **aucun artefact n'est
     produit en prod**, donc les tables seraient vides même si les dérivations étaient câblées.
     Le défaut est LATENT — il surgit le jour de l'activation, pas maintenant.
- **Conséquence réelle** : le jour où l'ouvrier distant sera activé en production, `match_usage_*`
  et `match_bomb_stats` (+ faits `bomb`) resteront vides et `real_start_time` sur le T0 estimé,
  parce que les projections sont accrochées à la cuisson locale et non au rangement d'artefact —
  défaut latent, dont le tiers T0-film est déjà consigné au registre.

## Constat 3 — Les dérivés n'ont aucun rattrapage : TIENT (gravité → P2)

- **Ce que j'ai vérifié** (tous les faits chiffrés se reproduisent exactement) :
  - `cuisson.go:275-276` : `if aJour && complet { b.dejaAJour++; return }` — sortie sans
    projection. `cuisson.go:309` / `:314` : seuls les matchs effectivement cuits alimentent
    `b.t0Film` / `b.usage`.
  - `backlog.go:153` + `:185-188` : le rattrapage teste `artefactPresent()` = `os.Stat` +
    taille > 0. Contre `replaybuild/artifact_digest.go:53` : `UpToDate()` =
    `d.SchemaVersion == replay.SchemaVersion`.
  - `usage.go:134-150` (writer absent / indisponible → WARN + `CompteurUsageEchecs`, aucune
    reprise) et `:151-163` (échec d'écriture par match → `echecs++`, `continue`). Confirmé.
  - Corpus mesuré sur la machine : `ls data/cache/replays/halo_infinite/*.json | wc -l` = **106** ;
    versions = 6·1, 20·27, 21·4, 23·7, 28·4, 31·1, 32·9, 34·51, 38·2 → **9 versions, 0 à la
    version courante 39** ; `grep -l t0FilmMs … | wc -l` = **2**. Chiffres exacts.
- **Ce que l'auditeur n'a pas vu (abaisse la gravité)** :
  1. **Le prédicat `os.Stat` porte sa justification écrite et mesurée** :
     `backlog.go:26-34` — « Le prédicat le moins cher qui existe : un `os.Stat`. Le prédicat
     complet … lit et désérialise l'artefact ENTIER — acceptable sur les quelques matchs insérés
     d'un cycle, **ruineux sur soixante-quatre à chaque cycle**. Le rattrapage répond donc à
     « ce match n'a AUCUN rejeu », **qui est exactement le défaut mesuré** ». La portée est
     déclarée, pas subie ; ce n'est pas une « factorisation abandonnée » mais un périmètre choisi.
  2. **L'état « 0 artefact à la version 39 » est un arbitrage utilisateur daté** :
     `REGISTRE_REPORTS.md` ligne 17 — « **Recuisson `backfill-replay --only-existing` pour publier
     `t0FilmMs` dans les artefacts (~106)** | chantier T0-film, 2026-09-02 | **Decision user 02/09 :
     « la recuisson attendra »** ». L'audit le range lui-même en « constats écartés » pour le
     `t0FilmMs`, puis le remobilise ici comme preuve d'un défaut de rattrapage.
  3. **Le rattrapage du corpus est explicitement assigné à l'opérateur**, pas oublié :
     `usage.go:45-47` (backfill CLI), `bombstats.go` + `cmd/levelup/cmd_backfill_bomb_stats.go`,
     `cmd/backfill_t0_film/`.
- **Conséquence réelle** : le fil de l'eau ne resélectionne jamais un artefact périmé mais présent,
  donc les dérivés des 104 artefacts périmés ne s'écriront que par un backfill opérateur — un
  périmètre déclaré dans les en-têtes et déjà arbitré par l'utilisateur, pas une lacune ignorée.

## Constat 4 — `match_weapon_shots` : passe + écriture pour zéro lecteur : RÉFUTÉ

- **Ce que j'ai vérifié** :
  - Le fait brut est exact : `grep -rn "match_weapon_shots" apps/go-api --include='*.go' |
    grep -v _test` → **aucun `SELECT`** hors `internal/migration/`, `internal/persist/` et
    la chaîne d'aide `cmd/levelup/main.go:188`. Rien non plus hors Go (`*.sql`, `*.ts`, `*.tsx`,
    `*.toml`, `docs/`) sauf le commentaire de `capabilities.toml:102`.
  - `collector.go:346` appelle bien `c.collectShots`, gaté par
    `games.CapFilmWeaponShots` (`collector.go:371-376`), déclaré `CapSupported`
    (`games/halo_infinite/adapter_data.go:218`, `capabilities.toml:108`).
- **Ce que l'auditeur n'a pas vu** :
  1. **C'est un report AU REGISTRE, daté, avec une décision utilisateur ferme et une condition de
     reprise nominative.** `.ai/V7.5/REGISTRE_REPORTS.md` ligne 24 : « **Precision PAR ARME (film)
     REMISEE — capability d'exposition RETIREE (decision FERME user)** | Remise precision-arme,
     **2026-09-01** | … On REMISE : **les acquis BACKEND sont conserves** (fix d'index
     `ShooterIndex5`/filmdec, decodeur `weapon_hits*`, resolveur distance, table/migration
     `match_weapon_hit_distance`, persister, mapper `weapon_accuracy_film.go`, passe
     `killcollector/hits.go`, **capability de STOCKAGE `film.weapon_shots`**), seule l'EXPOSITION
     est retiree … | **Piste compteur ECS validee** … Alors seulement re-declarer la capability ».
     La conservation de `film.weapon_shots` en `supported` est **nommément** l'objet de la
     décision. L'audit écarte deux constats frères sur ce motif exact (`hits.go` l. 15,
     `match_weapon_hit_distance` l. 24) et retient celui-ci : incohérence de son propre tri.
  2. **La doc dit bien ce que le code fait**, contrairement à ce qu'affirme le constat :
     `capabilities.toml:106-108` — « (!) **CETTE CLÉ GOUVERNE LE STOCKAGE, PAS UNE PUBLICATION** :
     la table stocke des COMPTES, elle ne publie aucun TAUX » ; `shots.go:15-16` — « **La table
     STOCKE ; elle ne publie pas.** »
  3. **« Une passe de décodage supplémentaire » est faux** : `shots.go:6-12` et `collector.go:288-293`
     documentent que les tirs sont greffés sur **la même** passe (film déjà chargé, chunks déjà
     décompressés) précisément pour éviter un second décodage — mesure citée : **1,65 s CPU en
     autonome contre ~0,5 s greffé**. Le coût réel est un balayage `ScanFireEventsB5` sur des
     chunks déjà en mémoire, pas une seconde passe chère.
- **Résidu honnête, non registré** : contrairement à `hits.go` (dont le registre chiffre le coût à
  « nul — la passe ne s'execute pas »), `collectShots` **tourne** et écrit ~0,5 s CPU + un burst
  writer par match pour une table sans lecteur. Ce coût résiduel n'est chiffré nulle part.
- **Conséquence réelle** : ce n'est pas un défaut neuf mais un report déjà consigné, daté et
  arbitré par l'utilisateur ; seul le coût résiduel de la passe (non nul, contrairement à sa
  jumelle `hits.go`) mériterait d'être ajouté à la ligne 24 du registre.

## Constat 5 — Quatre lectures attribuent l'arme sans la porte `publishable` : RÉFUTÉ

- **Ce que j'ai vérifié, requête par requête** — les quatre lectures « sans porte » sont **toutes**
  des agrégations pures, aucune n'affiche une mort :
  - `platform/duckdb/match_view_repo_weapons_source.go:34-45` : `SELECT k.feed_killer_xuid,
    k.source_tag, COUNT(*) … GROUP BY k.feed_killer_xuid, k.source_tag`.
  - `platform/duckdb/killsource_weapon_kills_repo.go:227-243` : idem, filtré par lot de matchs.
  - `platform/duckdb/killsource_weapon_scope.go:128-149` : `SELECT k.source_tag, COUNT(*) …
    WHERE k.feed_killer_xuid = ? … GROUP BY k.source_tag`.
  - `sync/citations_weapons_source.go:84-88` : `SELECT k.source_tag, COUNT(*) … GROUP BY k.source_tag`.
  Et les cinq lectures « avec porte » affichent bien **une mort nommée** : Q21b/Q21c = icône
  d'arme et assistant du kill feed (`queries_match.go:501`, `:540`), `kill_distance_repo.go:127`
  = distance d'UN kill, `match_view_repo_assist_pairs.go:91` (« **une paire nomme DEUX joueurs :
  c'est une lecture ligne à ligne** », `:49`), `queries_squad.go:257`/`:270`.
- **Ce que l'auditeur n'a pas vu — l'arbitrage écrit du dépôt dit exactement l'inverse de sa lecture** :
  - `internal/migration/steps_shared_kill_events.go:203-206`, arbitrage (B) du DDL :
    « `publishable = FALSE` veut dire : les lignes sont justes EN AGREGAT et fausses
    INDIVIDUELLEMENT (marge de bijection nulle). **Un lecteur de cumul ne filtre pas sur
    `publishable` ; un lecteur qui affiche UNE mort nommee — kill feed, duel, timeline — exige
    `publishable = TRUE`.** » L'asymétrie 4 / 5 n'est pas un oubli : c'est la règle, appliquée.
  - `platform/duckdb/kill_events_source.go:50` : « La colonne se lira le jour où une surface
    affichera **l'ARME d'UNE MORT** » — singulier. Un total « X frags au MA40 » n'est pas l'arme
    d'une mort.
  - **Décision datée, sur cette population précise** : `.ai/thought_log.md:9854-9857` —
    « La couverture n'est pas les 34,3 % du kill feed : ce taux vaut pour une publication LIGNE
    PAR LIGNE, **or l'usage est AGREGE** et le schema dit qu'une passe `publishable = FALSE` porte
    des lignes « justes en AGREGAT ». **Le tueur vient du kill-feed avec son xuid, sans
    bijection.** » Le constat P0 repose sur la thèse contraire (« agréger par joueur est une
    attribution ligne par ligne »), qui contredit un arbitrage écrit et daté sans le citer ni le
    réfuter sur mesure.
  - `killsource/bijection.go:17-21` confirme le mécanisme : la marge nulle rend deux joueurs
    interchangeables **dans la bijection indice → joueur**, qui sert à apparier le dead-state ;
    le crédit du tueur, lui, vient du feed en clair.
- **Conséquence réelle** : les quatre lectures sont dans la catégorie « cumul » que la doctrine du
  dépôt exempte explicitement de la porte ; le constat est un désaccord avec un arbitrage écrit et
  daté, pas un défaut non vu — et l'audit lui-même écarte d'autres constats sur ce motif
  (« décision datée et justifiée »).

## Constat 6 — Deux décodages, deux clés de fraîcheur indépendantes : RÉFUTÉ (comme constat distinct)

- **Ce que j'ai vérifié** : les faits bruts sont exacts. `grep -rn "killsource.Decode" apps/go-api
  | grep -v _test` → deux sites de production, `sync/killcollector/collector.go:314` et
  `replaybuild/kills.go:34`. `replaybuild/artifact_digest.go:44-53` : `Digest{MatchID,
  SchemaVersion, Players, Tracks, Bytes}` et `UpToDate() = d.SchemaVersion == replay.SchemaVersion`
  — aucune trace du décodeur. `grep -rn "KillSourceDecoderRev" internal/replaybuild
  internal/analysis/replay` → **0**.
- **Ce que l'auditeur n'a pas vu** :
  1. **C'est un doublon interne à la campagne.** L'audit G2 range lui-même « Deux décodages du même
     film par cycle (1.57 puis 1.58) » en **constats écartés** — « dette de performance assumée
     (audit cuisson 2026-09-02, C1-C7) ; le fait *structurel* — deux versionnages pour un décodeur
     — **est porté par le constat P0** ». G4-6 rejoue donc le constat 1 sous un autre angle.
  2. **La conséquence annoncée est INVERSÉE dans les faits.** Le constat dit : « un changement du
     décodeur fait redécoder la base (via `decoder_rev`) mais **pas** les artefacts ». Mesure :
     `const SchemaVersion` vaut **4** à `c8002abd2`, **8** à `efec87d71`, **31** à `4bfa9e383`,
     **39** à `328d83232` et à HEAD — soit 43 incréments sur la période où `KillSourceDecoderRev`
     n'a pas bougé une seule fois. C'est donc **la base** qui garde l'ancien décodage et **les
     artefacts** qui sont marqués périmés : l'inverse exact du scénario décrit. Le « rejeu 2D reste
     sur l'ancien décodage pendant que la fiche de match sert le nouveau » ne s'est jamais produit
     et ne peut pas se produire tant que la clé d'artefact bouge plus vite que celle du décodeur.
  3. Le second décodage est par ailleurs **bordé et documenté** : `replaybuild/kills.go:20-32`
     (« décode killsource UNE SEULE FOIS par match », sur le film déjà chargé par `BuildBytes`,
     lot 1 de `PLAN_CUISSON_PERF` item 1.4).
- **Résidu honnête** : « l'artefact n'enregistre pas `KillSourceDecoderRev` » est vrai, et devient
  un vrai risque **le jour où le constat 1 sera corrigé** (un bump de révision sans bump de schéma
  laisserait alors les artefacts intacts). À traiter comme corollaire du constat 1, pas comme un
  constat autonome.
- **Conséquence réelle** : rien d'observable aujourd'hui ; la seule divergence effective entre les
  deux horloges est celle que le constat 1 décrit déjà, et dans l'autre sens.

## Bilan : 3 tiennent, 3 réfutés, 2 requalifiés

| # | Constat | Verdict |
|---|---|---|
| 1 | `KillSourceDecoderRev` figé | **TIENT** (P0 maintenu) |
| 2 | Dérivations câblées sur la seule branche locale | **TIENT, gravité → P2** (tiers T0-film déjà au registre l. 18 ; défaut latent tant que l'ouvrier n'est pas activé en prod) |
| 3 | Aucun rattrapage des dérivés | **TIENT, gravité → P2** (prédicat `os.Stat` justifié et mesuré ; corpus périmé = arbitrage user daté, registre l. 17) |
| 4 | `match_weapon_shots` sans lecteur | **RÉFUTÉ** (registre l. 24, décision user ferme du 2026-09-01 nommant `film.weapon_shots` ; « passe supplémentaire » factuellement faux) |
| 5 | Quatre lectures sans la porte `publishable` | **RÉFUTÉ** (les quatre sont des cumuls, catégorie explicitement exemptée par l'arbitrage (B) du DDL et par une décision datée du thought_log) |
| 6 | Double décodage / deux clés de fraîcheur | **RÉFUTÉ comme constat distinct** (doublon du n°1, dette déjà écartée par G2 lui-même, conséquence inversée par la mesure : 43 bumps de schéma contre 0 bump de révision) |
