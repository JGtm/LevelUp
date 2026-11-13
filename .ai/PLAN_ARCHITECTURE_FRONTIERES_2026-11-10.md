# Plan architecture : frontières, ménage, front (2026-11-10)

**Statut : écrit le 2026-11-10 à partir du registre `.ai/AUDIT_ARCHITECTURE_2026-11-10.md`. Non relu
par un contexte frais. Décisions D1 à D9 validées par l'utilisateur le 2026-11-10. Go d'exécution donné le 2026-11-10 pour
les étapes 0 à 3, puis pour les étapes 4 à 8 le même jour. Lectures de production faites (0.4).**

**Rattachement : ce chantier fait partie de la v8.1.2 (utilisateur, 2026-11-10).** Le plan
`.ai/PLAN_V8_1_2_2026-11-09.md` est tenu par une autre session : il reste à y inscrire ce chantier.

Annexe : `.ai/annexes_architecture_2026-11-10/commandes_sans_utilisateur.txt`.

## 0. Objectif et critère de succès

Amener chaque ligne du tableau « changer l'UI ou une technologie » à « bon » ou « très bon », et
traiter les 25 P1 et les 22 P2 du registre. Le décodeur de film (`libs/film`) est hors périmètre :
projet à part, revu plus tard (message de l'utilisateur du 2026-11-10).

| Si on change... | Aujourd'hui | Cible | Mesure de clôture (étape 8) |
|---|---|---|---|
| Tout le front | Bon | Très bon | 0 type manuel homonyme d'un schéma du contrat ; garde étendu à tous les schémas |
| La bibliothèque de graphes | Moyen | Bon | 0 import de `echarts` hors `components/charts/` (types compris) ; tout graphe monté sous `ChartCard` |
| Le moteur de base | Moyen | Bon | 0 SQL et 0 `database/sql` dans `api/`, `service/`, `analysis/`, `progression/`, `scheduler/`, `prestige/` ; 0 type SQL dans `port/` |
| Le cadre HTTP | Moyen | Très bon | 0 import de `platform/duckdb`, `sync`, `persist`, `games/{slug}` dans `api/handlers` ; 0 SQL dans `api/wire` et à la racine `api/` ; aucun handler ne tient un type concret de `service` |
| Ajouter un titre | Moyen | Très bon | 0 import de `games/halo_infinite` ou `games/halo_5` dans `service/` et `api/handlers` |
| Le client HTTP du front | Bon | Très bon | 0 `fetch` hors `lib/api/` et `lib/desktop/` hors allowlist datée ; une seule source de la base URL |
| La coquille bureau, les briques | Très bon | Très bon | inchangé |

Deux cibles restent à « bon » et non « très bon » (décisions D6 et D7).

Succès =
- chaque item ci-dessous statué `[x]`, `[~]` (référence) ou `[!]` (justifié), aucune case vide ;
- les mesures de clôture ci-dessus tenues, chacune par un garde-rail nommé ;
- témoins avant et après identiques (étape 0) ;
- `make gate-push`, `go test -tags=integration -p 1 ./...` (depuis `apps/go-api`) et la CI de la
  branche verts ;
- une ronde de revue adversariale close sans P0 ni P1 ouvert.

## 1. Contrat d'exécution

Le skill `plan-execution` s'applique en entier. Propre à ce plan :

- **Aucun changement visible.** Ce plan déplace et supprime du code ; il ne change aucun écran ni
  aucune valeur, sauf le retrait de la probabilité de victoire (déjà masquée). Toute différence
  dans un témoin est un défaut, pas une amélioration.
- **Ordre strict.** Étape N close (items statués, gate passé, compte rendu relu sur pièces) avant
  N+1. Les étapes 1 à 3 sont indépendantes des étapes 4 à 6 mais s'exécutent quand même dans l'ordre.
- **Garde-rail dans le même commit.** Un item qui ramène un compte à zéro pose son garde-rail
  (plafond 0 ou allowlist vide, datée) dans le commit qui l'atteint (règle 6).
- **Rôles.** Le pilote ne code pas : il brieffe, relit sur pièces, commite, tient ce plan et
  `.ai/thought_log.md`. Exécutants Sonnet par défaut. Une seule ronde de revue adversariale par
  étape à risque (3, 4, 5), un relecteur Sonnet ; ronde 2 seulement s'il reste un P0.
- **Coût annoncé.** Environ 10 exécutants et 4 relecteurs Sonnet sur l'ensemble du plan.
- **Zéro fix hors périmètre.** Toute découverte va au §12, sans traitement.
- **Reprise de session.** Lire le §11 (avancement), puis `git log --oneline -15` sur la branche.

## 2. Décisions à valider avant exécution

| N° | Sujet | Recommandation | Validée |
|---|---|---|---|
| D1 | Branche et version | Chantier rattaché à la v8.1.2. Branche `feat/812-architecture` depuis `release/v8.1.2`, worktree dédié, une PR par étape vers `release/v8.1.2`. Ordonnancement : voir ci-dessous. | décidé par l'utilisateur le 2026-11-10 (« ce chantier fait partie de la 8.1.2 ») ; nom de branche proposé |
| D2 | Probabilité de victoire | Retirer l'affichage, la capability, les champs d'API et tous les lecteurs. Garder le calcul et la colonne `match_skill_rank.expected_win_prob` : c'est un sous-produit du calcul de la note, et retirer une colonne d'une table append-only impose une reconstruction. | validée le 2026-11-10 |
| D3 | `LEVELUP_USE_SHARED_PROVIDER` | Retirer l'interrupteur ; le lecteur hérité ne sert plus qu'aux tests. Mesure faite (0.4) : critère tenu. | validée le 2026-11-10 |
| D4 | Réparation automatique des trous de note (éteinte depuis le 2026-07-21) | Règle : jauge restée à 0 depuis le 21/07 : supprimer le code de réparation automatique (l'alerte et le recalcul manuel restent) ; sinon l'activer. Mesure faite (0.4) : jauge à 0 sur toute la période, donc suppression. | validée le 2026-11-10 |
| D5 | Sept interrupteurs sans échéance et une variable héritée | Retirer l'interrupteur et garder le comportement par défaut pour : `LEVELUP_POSTSYNC_BURST` (avec le chemin historique), `LEVELUP_LUSR_V2_MODE_COUPLING`, `LEVELUP_LUSR_V2_SQUAD_OFFSET`, `LEVELUP_SYNC_V2_DRYRUN` (avec ses bouchons), `LEVELUP_WATCHER_BROADCAST`, `LEVELUP_PERSIST_NO_FETCH_CACHE`, `PRESTIGE_ENABLED`, et la variable héritée `LEVELUP_LOG_JSON`. Mesure faite (0.4) : aucun des sept n'est posé en production ; `LEVELUP_LOG_JSON=true` l'est, par `Dockerfile:315`, à remplacer par `LEVELUP_LOG_FORMAT=json` dans le même commit. Conséquence : plus d'échappatoire sans redéploiement sur ces points. | validée le 2026-11-10 |
| D6 | Graphes : cible « bon » | « Très bon » voudrait que les pages ne construisent plus d'options ECharts : réécriture d'environ 67 fichiers, contraire au choix de l'ADR 0001 (`ChartCard` + `buildOption`). Non proposé. | validée le 2026-11-10 |
| D7 | Base : cible « bon » | « Très bon » voudrait sortir le SQL de `sync/` (216 fichiers, moteur d'écriture). Non proposé. | validée le 2026-11-10 |
| D8 | Commandes sans utilisateur | Supprimer les 98 commandes de l'annexe (85 sans aucune référence, 13 après relecture de la référence qui les nomme). Conserver 14 : instruments de mesure cartes et film (liés au décodeur, revu plus tard), items déjà au registre des reports, outils nommés par un message de test. | validée le 2026-11-10 (« je vais suivre tes reco, de toute façon tout est versionné ») |
| D9 | Contrat d'API | Le retrait des champs `expected_win_prob` change `openapi.yaml`. Sans effet de compatibilité : le front et le serveur sont livrés ensemble, y compris dans l'application Windows. | validée avec D2 (conséquence mécanique) |

**D8 révisée le 2026-11-10 (après exécution de 1.1).** Le critère « aucune citation » ne voit pas l'usage réel. La
comparaison du code inatteignable avant et après 1.1 (`deadcode`) montre que 17 commandes supprimées étaient le
seul point d'entrée de 63 fonctions du moteur ; elles sont rétablies. Décision de l'utilisateur le 2026-11-10
(« ok avec tes recos », et pour le classement mondial « le serveur lance les commandes en cron ») :
- GARDER et documenter dans `docs/COMMANDS.md` (deux langues) : Halo 5 (`h5-backfill`, `h5-events-backfill`,
  `h5-kill-kind-backfill`, `h5-csr-backfill`, `h5-roster-refetch`), calibrage du LUSR (`lusr_v2_ttt_batch`,
  `lusr_v2_squad_estimate`, `lusr_v2_replay`), et `backfill-csr-history` (saisons de rang PASSÉES d'un joueur :
  la sync n'écrit que la saison courante, vérifié dans `sync/engine_postsync_scoring.go`).
- SUPPRIMER avec leur code du moteur : classement mondial (`snapshot-world-leaderboard`,
  `backfill-world-player-stats`, `world-aliases-persist`). Vérifié en production le 2026-11-10 : aucune tâche
  planifiée de l'hôte ne les lance, c'est `scheduler.WorldLeaderboardCron` du serveur qui tourne chaque jour.
  Et `h5-read-smoke` (outil d'essai).
- GARDÉES (utilisateur, 2026-11-11 : « je n'ai lancé aucune de ces 4 commandes, mais elles me semblent
  utiles ») : `migrate-hls-audio`, `backfill-media-hls`, `purge_corrupt_records`, `backfill_time_played`. À
  documenter dans `docs/COMMANDS.md` (deux langues) avec l'étape 2. Texte d'origine de cette ligne : La recommandation acceptée est « supprimer si elles ont été passées en production » ;
  le journal n'en porte aucune trace. Question posée à l'utilisateur.

**Ordonnancement dans la v8.1.2 (D1).** Les quatre branches 8.1.2 (`feat/812-bureau`,
`feat/812-film`, `feat/812-web`, `feat/812-stats`) sont fusionnées dans `release/v8.1.2` (vérifié le
2026-11-10 : la base de ce chantier, `76635f08be`, les contient ; confirmé par le pilote de la
v8.1.2). Plus aucune attente entre les étapes de ce plan. Reste côté v8.1.2 : la PR des identités de
test (#187) et la bascule sur les modules publiés, qui réécrit des imports dans tout le dépôt : le
pilote de la v8.1.2 la lance APRÈS la fusion de la dernière étape de ce chantier, à lui signaler.
La branche est rebasée sur `release/v8.1.2` avant chaque étape.

**Plus de PR par étape (utilisateur, 2026-11-11 : « arrête de passer par des PR, je suis dev solo »).** L'étape 1 est
partie par la PR #191 (fusionnée, `91727b0c30`). À partir de l'étape 2, les commits sont poussés directement
sur `release/v8.2.0` (ancien nom `release/v8.1.2`, renommée le 2026-11-11) une fois le gate local passé ; la CI de la release se lit après le push.

**Changement de numéro (relais d'une autre session, 2026-11-11, décision de l'utilisateur).** Un hotfix prend le numéro
8.1.2 depuis `main` (PR #193) ; le travail de `release/v8.1.2` passe en 8.2.0 (branche renommée par son pilote,
puis rebasée sur `main`). Le hotfix porte la montée de Go en 1.26.11 et touche `sync/engine.go`, `sync/writes.go`,
`scheduler`, `platform/auth/pool`, `service/teammates`. Conséquence : l'étape 2 (qui touche `sync/engine.go` et
`scheduler`) attend que la release soit renommée et rebasée.

**Fenêtres convenues avec le pilote de la v8.1.2 (2026-11-10, sauf avis contraire de l'utilisateur).** Chaque étape
part en une PR et l'étape suivante ne commence qu'après sa fusion : entre deux étapes, aucun travail en vol.
(1) Après la fusion de l'étape 1 : prévenir le pilote, qui passe le renommage des chemins de module (environ
580 fichiers d'imports) ; attendre son signal de fusion, rebaser, puis étape 2. (2) Après la fusion de l'étape 3 :
le prévenir, il passe la republication des briques et la bascule (suppression de `libs/` et `go.work`,
`Dockerfile`, CI) ; attendre son signal, rebaser, puis étape 4. Après sa PR #189 (aides d'`archlint`) : rebaser.

**Vérifications à l'écran de l'utilisateur.** Elles vivent dans son artefact unique
(`https://claude.ai/artifact/P9VJKKbSDsQu1PUBjyGwTv`, collection `verifs`, documents `archi-tri` et
`archi-palette`, état `a_venir`). À la clôture de l'étape 7 : passer leur `etat` à `a_faire`, puis y
relire sa réponse.

**Mesures de production (0.4, lecture seule, 2026-11-10).**
- Jauge des trous de note : 298 relevés dans `scheduler.log` du 2026-07-22 au 2026-11-10, tous à 0,
  aucun relevé partiel.
- Fournisseur partagé : `LEVELUP_USE_SHARED_PROVIDER` absent des deux conteneurs ; 10 erreurs de
  restauration après bascule dans `duckdb.log`, toutes le 2026-07-11, aucune depuis (journal couvert
  du 2026-06-13 à ce jour). Le trimestre écoulé est propre.
- Interrupteurs posés en production parmi ceux visés : `LEVELUP_LOG_JSON=true` (deux conteneurs),
  `LEVELUP_NOTIFY_VERSIONS=1` (conteneur principal, par `docker-compose.yml:31`). Aucun autre.

## 3. Étape 0 — Préparation et témoins

- [x] 0.1 Créer `feat/812-architecture` depuis `release/v8.1.2` (D1) dans un worktree dédié.
- [x] 0.2 Capturer les témoins AVANT sur le serveur local, joueur et titre fixés, sortie JSON
      triée (`jq -S`) : accueil, carrière, progression (séries, records, jalons), campagne, profil
      joueur, escouade et synergies, détail de match, sessions, séries temporelles, explorateur,
      classements, réglages (données), monitoring (fraîcheur, couverture des armes), état de sync.
      Un fichier par route sous le dossier de session, liste des routes écrite dans le compte rendu.
- [x] 0.3 Relever les compteurs de départ des mesures de clôture du §0 (commandes du registre).

Résultat de 0.2 (2026-11-10). Harnais hors dépôt : `bash C:/t/812-archi/capture.sh <dossier>` (2 min, plus la
compilation). 90 routes, toutes en 200, sur le jeu de démo copié : 69 routes joueur et titre en profil démo,
21 routes admin, monitoring et réglages en profil dev sans authentification ; 10 variantes `halo_5`, 4 en
anglais. Référence : `C:/t/812-archi/avant/` ; `diff -r avant avant-bis` vide. Champs neutralisés et listes
triées : `C:/t/812-archi/normalize.txt`. Limites du témoin : il ne voit ni un changement d'ordre sur les listes
triées par le harnais, ni le contenu de `top_medals`, `top_citations`, `impact_roles`, `levers[].rank` ; il ne
couvre ni le rejeu (le jeu de démo n'a pas de film), ni `/squads`, ni l'export de l'historique.

Résultat de 0.3 (2026-11-10, base `76635f08be`).

| Mesure | Départ |
|---|---|
| fichiers de `api/handlers` important `platform/duckdb`, `persist` ou `sync` | 9 |
| fichiers de `service/` et `api/handlers` important `games/halo_infinite` ou `games/halo_5` | 16 |
| fichiers de `api/wire` et de la racine `api/` portant du SQL | 9 |
| occurrences de `*service.X` dans `api/handlers` | 20 |
| fichiers important `database/sql` dans `service`, `analysis`, `progression`, `scheduler`, `prestige`, `port` | 19 |
| fichiers lisant `os.Getenv` hors `internal/config` | 22 |
| fichiers web important `echarts` hors `components/charts/` | 72 |
| `fetch` hors `lib/api/` et `lib/desktop/` | 7 |
| types manuels homonymes d'un schéma du contrat | 31 |
| commandes de `cmd/` | 180 |
- [x] 0.4 Lectures de prod en lecture seule pour D3, D4 et D5 : faites le 2026-11-10, résultats au §2.

Gate : témoins rejoués deux fois de suite, identiques (sinon retirer la route instable et le dire).

## 4. Étape 1 — Ménage : suppressions pures

Aucun comportement ne change. Effort : moyen.

- [x] 1.1 (MORT-1, MORT-P2-1) Supprimer les 98 commandes de l'annexe, section par section. Pour les
      13 de la seconde section, relire d'abord la référence de code qui les nomme et corriger le
      commentaire. Retirer leurs entrées des allowlists (`archlint/*_test.go`,
      `platform/duckdb/no_attach_on_social_test.go`).
- [x] 1.2 (MORT-2) Supprimer `internal/presence/steam_poller.go`, ses tests et
      `DaemonConfig.SteamAPIKey` ; supprimer `internal/domain/chart/`.
- [x] 1.3 (MORT-P2-1) Supprimer `openspartan.IsOpenSpartanDB` et `campaign.MinMatchesForTrend`.
- [x] 1.4 (FLG-5) Supprimer `buildHomePlaylistRankItem`, `fillRankedMSRItem`, `fillPlacementItem`
      (`platform/duckdb/home_repo_playlist_ranks.go`), `errNoFallback`
      (`service/career_live_service.go:539`), `AuthHandler.WithUserStore` (`api/handlers/auth.go:76`).
- [x] 1.5 (STO-P2-3) Retirer l'import `duckdb-go` de `service/media_service.go` et
      `service/media_index_service.go` ; corriger le commentaire.
- [x] 1.6 (ART-5, D2, D9) Probabilité de victoire : retirer `CapExpectedWinProb` et son miroir web,
      `MatchWinProbCard`, les colonnes des tableaux Synergies et Progression, `lib/winProbCategory.ts`,
      les champs des réponses d'API et de `openapi.yaml`, `QSquadExpectedWinProbTpl` et
      `loadExpectedWinProbs`, l'entrée `queries_squad.go` de `no_raw_rating_reads_test.go`, les
      chaînes i18n devenues orphelines. Régénérer `generated.ts`. Ne toucher ni au calcul ni à la
      colonne. Lister dans le compte rendu les 34 fichiers Go relus et le sort de chacun.
- [x] 1.7 (FLG-4) Corriger les deux commentaires inversés (`scheduler/auto_sync.go:205`,
      `cmd/server/main.go:16`).
- [x] 1.8 (TEC-2, partie code mort) Supprimer le type manuel `MatchHistoryRow` de `lib/api/types.ts`.

- [x] 1.9 (règle 7, découvert en 1.1) Aucun code du moteur laissé sans appelant par les suppressions de
      commandes. Contrôle : `deadcode` avant et après, `comm -13` vide.

Résultat de 1.9 (2026-11-10). 17 commandes rétablies (seul point d'entrée de 63 fonctions), puis décision de
l'utilisateur appliquée (D8 révisée) : 4 supprimées avec leur code (`WorldCSRBatchEntries`,
`WorldCSRBestBatch`, `WorldCSRSeasonPlaylistPairs` et leurs types, `InsertWorldNoDataPlayers`,
`WorldSeasonEnrichedGamertags`, `logging.InstallCLI`, `ProvidesNativeKDA` et le réglage `no_native_kda` que
plus rien ne lisait), 9 gardées et documentées dans `docs/COMMANDS.md` et son pendant français, 4 en attente.
Bilan de `apps/go-api/cmd` : 180 au départ, 83 supprimées, 98 restantes (une arrivée par la release).
Témoins identiques. Le harnais neutralise désormais l'état du disque du poste (seuil de 80 % franchi en cours
de chantier : `open_detections`, `disk.status`, `volume.status`).

Résultat de 1.1 à 1.8 hors 1.6 (2026-11-10). 96 commandes supprimées et non 98 : `admin` (outil de secours des
comptes, seul moyen de créer un administrateur ou de réinitialiser un mot de passe sans l'API web) et `diag_q`
(requête SQL en lecture seule, outil courant des diagnostics) sont conservées et documentées dans
`docs/COMMANDS.md` et `docs/FR/COMMANDS.md` ; l'annexe est à jour. 84 commandes restent. 142 entrées
d'allowlist retirées dans 11 tests, aucune ajoutée ; plafond `nombreSitesAiAutorisesPlafond` descendu de 15 à 14.
Sont partis avec 1.2 : `STEAM_API_KEY` (exemple d'environnement, `docs/CONFIGURATION.md` et son pendant
français), l'entrée netguard de Steam, l'alias `handlers.UserLinker`. `docs/CHANGELOG.md` n'est pas modifié
(historique). Témoins : `diff -r avant apres-1a` vide. `go test ./internal/archlint/...` demande
`-timeout 30m` (4 min 30 s à vide, plus de 10 min sous charge).

Résultat de 1.6 (2026-11-10). Retirés : la capability, les champs de 5 schémas OpenAPI (`ExplorerMatchesRow`,
`MatchExpectedStats`, `MatchHistoryRow`, `SessionDetailMatchRow`, `SquadMatchHistoryRow`) et de `SkillSnapshot`,
les lecteurs de `platform/duckdb` (dont la lecture brute d'Escouade et son entrée d'allowlist : 5 vers 4), la
carte du détail de match, les colonnes des tableaux Synergies et Progression, `lib/winProbCategory.ts`, 14 clés
i18n dans chaque langue. Aucun cas où la probabilité alimente une autre valeur. Intacts : `sync/skill`,
`analysis/skill_v2`, `persist`, schéma et migrations (aucune ligne modifiée). Témoins : 303 lignes disparues
dans 10 fichiers, toutes porteuses de la clé retirée, aucune ligne ajoutée.

Gate de clôture de l'étape 1 (2026-11-10) : compilation, `go vet`, `gofmt`, `archlint`, `platform/duckdb` et
`sync` en série, `domain`, `analysis`, `service`, `api`, `games`, `ops`, `config`, `scheduler`, `cmd` verts ;
typecheck, lint, vitest (1 078 fichiers) et lint des imports entre features verts. La suite complète
`go test ./...` n'a pas été rejouée d'un bloc en local (poste chargé par plusieurs sessions) : c'est la CI de
la PR qui la porte.

Gate : `cd apps/go-api && go build ./... && go vet ./... && go test ./...` ; `make check-types`,
`npm run lint`, `make test-web` ; commande de reproduction de MORT-1 : 0 commande hors de la section
« conservées » ; témoins identiques, à l'exception des champs `expected_win_prob` disparus.

## 5. Étape 2 — Interrupteurs

Effort : moyen. D3, D4 et D5 sont validées, les lectures de 0.4 sont faites.

- [x] 2.1 (FLG-1, D3) Retirer `LEVELUP_USE_SHARED_PROVIDER` ; `LegacySharedReader` passe sous les
      tests ou disparaît si les tests se branchent sur le fournisseur.
- [x] 2.2 (FLG-2, D4) Supprimer la réparation automatique des trous de note : `isLUSRAutoHealEnabled`,
      `LEVELUP_LUSR_AUTOHEAL_ENABLED`, le candidat de réparation et son déclenchement
      (`scheduler/data_health_lusr.go`), leurs tests. La détection, la jauge, l'alerte et le recalcul
      manuel du panneau de monitoring restent.
- [x] 2.3 (FLG-3, FLG-P2-1, D5) Retirer les huit interrupteurs de D5 et leurs branches mortes ;
      `Dockerfile:315` passe de `LEVELUP_LOG_JSON=true` à `LEVELUP_LOG_FORMAT=json` dans le même
      commit, et le premier déploiement vérifie que les journaux restent en JSON. Pour
      les deux interrupteurs du LUSR : le comportement par défaut est conservé à l'identique, donc
      `LUSRRev` ne change pas ; le prouver par l'empreinte de `sync/scoring_revision_test.go`
      inchangée.
- [x] 2.4 (FLG-P2-3) Faire lire par `internal/config`, une seule fois, `LEVELUP_LOG_LEVEL`,
      `LEVELUP_OAUTH_CLIENT_ID`, `LEVELUP_LEGACY_FILM_CACHE_DIR`,
      `LEVELUP_EVENTS_CONVERGENCE_CHUNK`, `LEVELUP_EVENTS_CONVERGENCE_YIELD_MS`,
      `LEVELUP_NOTIFY_VERSIONS`.
- [x] 2.5 (FLG-P2-2) Dater ou retirer les deux branches de compatibilité
      (`handlers/media_paths.go:110-160`, `web/lib/title-routing/buildLegacyRedirect.ts`) : retrait
      si la mesure montre zéro usage, sinon date cible et critère en commentaire.
- [x] 2.6 Garde-rail : `archlint` interdit `os.Getenv` et `os.LookupEnv` hors `internal/config` et
      `cmd/`, allowlist datée pour les données de déploiement restantes (chemins système).
- [x] 2.7 Mettre à jour `docs/COMMANDS.md` et son pendant français, plus toute doc qui nomme une
      variable retirée (règle 15).

Résultat de l'étape 2 (2026-11-11). Dix commits. 2.1 : interrupteur retiré, le fournisseur partagé est le seul chemin.
2.2 : réparation automatique supprimée ; détection, jauge, alerte et recalcul manuel intacts. 2.3 : huit
interrupteurs retirés ; défauts vérifiés dans le code avant retrait (les deux du LUSR étaient ACTIFS par défaut,
leurs commentaires disaient l'inverse) ; `LUSRRev` et `sync/scoring_revision_test.go` inchangés ; l'image pose
`LEVELUP_LOG_FORMAT=json` (à vérifier au premier déploiement : journaux en JSON). 2.4 et 2.6 : six variables lues
une fois par `internal/config`, garde-rail `archlint` avec allowlist datée. 2.5 : les deux branches sont DATÉES
(retrait cible 2026-12-31), pas retirées : en production, 0 requête de page `/players/...` dans les journaux
d'accès, qui ne couvrent que 15 jours (2026-10-28 au 2026-11-11, 344 pages aux adresses actuelles) ; 0 avertissement
« aucun mapping trouvé » depuis le 2026-06-14, mais les chemins absolus en base ne sont pas comptés. 2.7 : aucune
mention restante hors historiques. Gate : `go vet ./...`, `archlint`, `-tags=integration -p 1` sur sync, persist,
platform/duckdb et scheduler, reste des paquets : code de sortie 0 ; `git grep` du gate vide ; code sans appelant :
0 nouveauté ; témoins : 90 routes, une seule différence, `ffmpeg_version` absent de la référence `avant-2` seule
(présent dans toutes les autres captures : sonde ffmpeg en échec pendant cette capture, poste chargé). Web : deux
commentaires seulement, contrôles laissés au pre-push. Exécutant coupé deux fois par les limites d'usage ; la fin
de 2.3, 2.5 et le gate ont été faits par le pilote.

Gate : gate de l'étape 1, plus `go test -tags=integration -p 1 ./...` (le post-sync change de
chemin) ; `git grep -nE 'LEVELUP_(USE_SHARED_PROVIDER|POSTSYNC_BURST|LUSR_V2_(MODE_COUPLING|SQUAD_OFFSET)|SYNC_V2_DRYRUN|WATCHER_BROADCAST|PERSIST_NO_FETCH_CACHE|LOG_JSON)' -- apps docs`
vide ; témoins identiques.

## 6. Étape 3 — Écritures : erreurs tracées et garde-rails

Touche `persist/` et `sync/` : revue adversariale obligatoire, tests d'intégration obligatoires.
Effort : moyen.

- [x] 3.1 (ART-2) Alias de joueurs : compter et journaliser chaque échec d'écriture ; ne poser le
      marqueur `psa_checked_at` qu'après un flush réussi, de sorte que le match soit repris au cycle
      suivant. Supprimer `upsertAliasesFromMatchJSON` (sans appelant). Test : un flush en échec
      laisse le match dans `selectMatchesMissingPSA`.
- [x] 3.2 (ART-1) `persistMatchIntensities` : journal et compteur sur chaque échec.
- [x] 3.3 (API-3, partie erreurs) `wire/registry_weapon_coverage.go:97-98` et
      `registry_catalog_expand.go:171-172` : journaliser les erreurs de `rows.Scan`.
- [x] 3.4 (ART-3) Étendre `archlint/no_bare_duckdb_rw_open_test.go` à `duckdb.OpenReadWrite`, avec
      l'allowlist datée de l'ADR 0030 D-3.
- [x] 3.5 (ART-4) Dater et rejustifier les entrées de `sync/shared_write_guard_test.go`,
      `sync/append_only_state_guard_test.go`, `platform/duckdb/no_raw_rating_reads_test.go` ; clé
      par chemin complet et non par nom de base ; corriger le motif périmé de `writes.go` et le
      commentaire de `writes.go:123`. Poser un test de forme : toute entrée d'allowlist de ces
      fichiers commence par une date.
- [x] 3.6 (ART-P2-1) Compléter la recette de `persist/doc.go` (liste des colonnes de la vue) et
      poser un test qui lie `pmeColumnStage` et `enrichmentColumnStage`.

Résultat de l'étape 3 (2026-11-12). Neuf commits sur `feat/812-architecture` (`f5729e4deb` à `e124f2423b`), dont un
correctif de revue (P1 sur 3.1, `3b44c03102`).
3.1 : `convergePSACollect` ne fait que le fetch et n'écrit plus rien ; `flushAliasPairs` écrit les alias ; `settlePSA`
écrit ENSUITE les awards puis le marqueur, par match, pour les seuls matchs dont tous les alias sont écrits. Un échec
d'alias ne laisse donc aucun award : le match reste dans `selectMatchesMissingPSA` (qui exclut tout match ayant des
awards) et la reprise ne réinsère jamais une génération d'awards. Première version (`f5729e4deb`) : awards écrits avant
le flush, d'où le P1 de la revue (match avec awards, alias en échec : hors sélection, marqueur NULL pour toujours) ;
reproduit par `TestRunPSAConvergence_FlushAliasEnEchecAvecAwards`, qui ÉCHOUE sur `cec4557d45` (`sélection au cycle
suivant = []`) et passe après. Chaque échec d'alias est journalisé et compté (`convergence_alias_upsert_failed_total`),
chaque match laissé à reprendre aussi (`convergence_psa_deferred_total`, tous réellement sélectionnés au cycle suivant) ;
`upsertAliasesFromMatchJSON` supprimée. Choix de conception, sur pièces : la sélection par le seul marqueur (demandée
par la revue) n'a PAS été retenue, parce que le chemin per-match écrit des awards sans jamais poser le marqueur (aucun
autre écrivain de `psa_checked_at`) : sur le jeu de démo (copie de `racine-source`, lecture seule), 59 matchs sur 68, 13
sur 25 et 14 sur 24 (trois joueurs Halo Infinite) ont des awards sans marqueur, qui entreraient tous dans la reprise
(refetch réseau, 50 par cycle) ; poser directement le marqueur pour eux changerait ce qui est écrit, hors périmètre.
Ordre awards après alias : aucun refetch en plus pour ces matchs, un refetch seulement pour un match dont l'alias a
échoué. Cas structurel : aucun profil de déploiement (hosted, desktop ; demo n'a ni sync ni réseau sortant) n'a de
base partagée en lecture seule par construction ; « RC-A » est un incident (handle ouvert en lecture), signalé par
`trackFatalErr` et qui fait échouer toutes les écritures shared ; `SharedAccess.Write` refuse un handle nil par une
erreur. Tout échec est donc traité comme transitoire (reprise, bornée à `convergenceHorizon` matchs par cycle), pas de
marqueur « impossibilité structurelle ». `TestConvergePSA_SharedNilAvecAlias` ne couvre qu'un enrobage de test (`convergePSA`
n'a aucun appelant de production et figurait déjà au code sans appelant avant l'étape ; gardée, à statuer au 8).
Preuve des dates d'allowlist : `git log --reverse --format='%as %h %s' -S'"<clé>"' -- <fichier de garde>` ; deux dates
reprises du texte d'origine ont été ramenées à celle de git (2026-10-20, squash v8.0.0).
3.2 : `persistMatchIntensities` journalise et compte (`match_intensity_write_failed_total`) ; SQL inchangé. 3.3 :
les deux sites du plan plus `classifyWeaponLabelsOnly` (même motif, même fichier) passent par
`signalerLigneIllisible` (journal et `wire_rows_scan_failed_total`) ; un `rows.Scan` ne peut pas être mis en échec
par les données d'un schéma valide : seul le compteur du helper est testé. 3.4 : second test d'`archlint` sur les
appels de `duckdb.OpenReadWrite` (alias d'import quelconque), allowlist de 43 fichiers et 49 appels, chaque entrée
datée et justifiée sur pièces ; mutation vérifiée (un appel de plus est refusé) ; `OpenReadWriteShared` non couvert.
3.5 : toutes les raisons des trois fichiers commencent par une date (tirée de git) puis « revu 2026-11-12 » ;
`writes.go` : motif périmé corrigé, commentaire de `InsertWeaponKills` corrigé ; `no_raw_rating_reads_test.go` :
clé par chemin complet (l'entrée `identity_annuaire.go` vivait sous `internal/analysis/`) ; trois tests de forme.
Les onze entrées d'allowlist existantes (cinq, deux et quatre) sont gardées, aucune retirée ; une porte la mention « non revérifié »
(`queries_career_encounters.go`, voir §12). 3.6 : recette de `persist/doc.go` portée à huit étapes (projection dans la
vue), les deux copies de `player_persister.go` y renvoient ; `TestEnrichmentColumnStage_LieALaVueMergeOnRead` échoue
sur un stage divergent et sur une colonne absente (mutations vérifiées). Gate (après correctif de revue) : `go build ./...` et `go vet ./...` code 0 ; `go test -tags=integration -p 1
./internal/sync/... ./internal/persist/... ./internal/platform/duckdb/... ./internal/scheduler/...` code 0 ; `go test -timeout 30m
./internal/archlint/...` code 0 ; avant correctif : `-tags=integration -p 1 ./...` code 0 (216 paquets `ok`) ; code sans appelant :
0 nouveauté ; témoins `apres-3b` : 90 routes, une seule différence avec `avant-3` (fenêtre `since`/`until` de
`activity_calendar.json` décalée d'un jour : changement de date UTC entre les deux captures ; `apres-3` et `apres-3b`
sont identiques entre elles). Revue
adversariale : en attente.

Gate : gate de l'étape 2 ; `go test -tags=integration -p 1 ./...` code de sortie 0.

## 7. Étape 4 — Frontières : la couche HTTP

Effort : lourd. Revue adversariale obligatoire. Chaque déplacement garde les signatures HTTP et les
réponses à l'identique (témoins).

- [x] 4.1 (DEP-P2-2) Sortir de `worldenrich/wiring.go` l'import de `service` (le câblage monte dans
      `api/wire` ou `cmd/server`), pour que `service` puisse importer le suivi en direct sans cycle.
- [x] 4.2 (API-2) Créer le port des jobs de synchronisation dans `internal/port/` (lancer une sync,
      un rattrapage, un alignement, un recalcul de sessions ; résolution du profil comprise) et son
      service. `handlers/backfill.go`, `sync_handler.go`, `sync_handler_align.go`,
      `settings_sessions_recalc.go` n'importent plus ni `sync` ni `games/halo_5/livesync`. La
      recherche de profil recopiée trois fois devient une fonction du service.
- [x] 4.3 (API-1) Ports et services pour la progression (séries, records, historique, jalons : la
      jointure catalogue × jalons va dans le service), la campagne, le profil joueur, et la limite
      mémoire des bases (`settings_data.go`). Les quatre handlers n'importent plus `platform/duckdb`.
- [x] 4.4 (API-3) Sortir le SQL des 9 fichiers de `api/wire` vers des dépôts de `platform/duckdb`
      derrière des ports ; les écritures de `registry_catalog_expand.go` (file de récupération,
      poids des playlists, DDL) vont dans `persist/` et `migration/`. Le calcul KPM et PSPM de
      `post_sync_progression_queries.go:231-236` passe par la fonction existante d'`analysis/`
      (vérifier l'existant, skill `go-features`). `ExpandPlaylistChildren` et la couverture des
      armes deviennent des services.
- [x] 4.5 (API-4) Déplacer `commendation_handler.go`, `spa_handler.go`, `static_files.go` et
      `serveStaticFile` dans `api/handlers/` ; `media_audience.go` dans un service ;
      `startSessionPurgeLoop` dans `scheduler/`, déclaré comme tâche de fond par profil (ADR 0044).
- [x] 4.6 (API-P2-1) Interfaces de `port/` pour `BootstrapService`, `PresenceService`, `SiteImport`,
      l'import OpenSpartan, `PlayerEngagementService`, `MediaIndexer`, la sync de titre.
- [x] 4.7 Garde-rails `archlint`, plafond 0 : (a) `api/handlers` n'importe ni `platform/duckdb`
      (hors types d'erreur listés), ni `sync`, ni `persist`, ni `games/{slug}` ; (b) aucun littéral
      SQL dans `api/wire` ni à la racine `api/` ; (c) aucun `*service.X` dans `api/handlers`.
- [x] 4.8 Tests : `httptest` avec faux port pour chaque handler touché ; faux dépôt pour chaque
      service créé ; DuckDB en mémoire pour chaque dépôt créé.

Résultat de l'étape 4 (2026-11-12). 4.1 à 4.4 : voir §12 (une ronde de revue adversariale, 0 P0, P1 corrigé).
4.5 : fichiers statiques, page web et distinctions dans `api/handlers` (62 requêtes comparées octet pour octet
entre l'ancien code et le code déplacé : statut, en-têtes, corps, aucune différence) ; règle d'audience des médias
dans `service.MediaAudienceService` ; purge des sessions = tâche de fond déclarée (`scheduler.RunSessionPurgeLoop`).
4.6 : les handlers tiennent des interfaces de `port/` (présence, import depuis le site, import OpenSpartan,
engagement, indexeur de médias, sync de titre, bootstrap). 4.7 : trois gardes à plafond 0 et allowlists vides
(stockage, `persist` et paquets de titre ; SQL dans `api` ; paquet `service`) ; `home.go` classe une panne de
lecture par `platform/dbfault`. 4.8 : faux ports et faux dépôts pour chaque handler et service touché. Gate :
build, vet (avec et sans `integration`), gofmt, `archlint`, `-tags=integration -p 1` sur api, service, port,
platform/duckdb, dbfault, session, scheduler, sync, cmd/server : code de sortie 0 ; code sans appelant : 4 entrées
de moins, aucune nouveauté ; mesures « cadre HTTP » : toutes à 0 (relues par le pilote) ; témoins : 90 routes,
`diff -r avant-4 apres-4e` vide. Non lancés en local : `golangci-lint` et la baseline de tests (laissés à la CI).

Gate : gate de l'étape 3 ; mesures « cadre HTTP » du §0 à zéro ; témoins identiques.

## 8. Étape 5 — Frontières : le stockage

Effort : lourd. Revue adversariale obligatoire.

- [x] 5.1 (STO-1, STO-P2-2) Import OpenSpartan : l'amorçage des lignes d'enrichissement passe par le
      persister existant (`persist/post_sync_enrichment_persister.go`), sans arrêt à la première
      erreur. Les `*sql.DB` disparaissent des signatures des services d'import (dépôts derrière
      ports) ; le champ `legacyDB` réservé aux tests disparaît.
- [x] 5.2 (STO-2) `progression/profile` : les 13 requêtes vont dans un dépôt de `platform/duckdb`
      derrière un port ; le service ne tient plus de `*duckdb.DB` ; `scanTemplate` n'existe plus
      qu'une fois.
- [x] 5.3 (STO-3) `scheduler` : les sondes de santé et la purge des rejeux lisent par un dépôt ; le
      scheduler n'importe plus `platform/duckdb`. Une sonde en erreur reste comptée dans
      `ProbeErrors` et le rapport le montre.
- [x] 5.4 (STO-4) Sortir d'`analysis/` `weapon_evidence.go` et `prestigetuning/collect.go` (la
      requête dans `platform/duckdb`, le calcul reste). Étendre le test de pureté à tout `analysis/`.
- [x] 5.5 (STO-P2-1) `port.DBExecutor` et `port.DBWriter` ne rendent plus de type SQL : la
      transaction média (ouverture, validation, CHECKPOINT) devient une méthode du dépôt.
- [x] 5.6 (STO-P2-4) Erreur de domaine « introuvable » dans `port/`, rendue par les dépôts ; les 6
      comparaisons à `sql.ErrNoRows` de la couche métier s'y branchent.
- [x] 5.7 (STO-P2-5) Déplacer les fragments SQL d'`analysis/` (`sql_fragments.go`, `sql_liste.go`,
      les requêtes d'`identity.go` et `identity_annuaire.go`, `campaign_exclusion.go:107`) dans un
      paquet de `platform/duckdb` ; les garde-rails qui tiennent le fragment de fuseau canonique
      (règle 8) suivent le déplacement.
- [x] 5.8 (DEP-P2-1) `config` n'importe plus `platform/duckdb` ni `prestige` : `ResolvePlayer` et le
      fournisseur partagé par titre montent dans le câblage.
- [x] 5.9 Garde-rails `archlint`, plafond 0 : ni `database/sql` ni littéral SQL dans `service/`,
      `analysis/`, `progression/`, `scheduler/`, `prestige/` ; aucun type `sql.` dans `port/`.

Résultat de l'étape 5 (2026-11-13). 5.1 : services d'import sans `*sql.DB` (ports, paquet `internal/importstore`),
amorçage par `PostSyncEnrichmentPersister.PrimeBaseline`, une transaction par match, chaque échec loggé et compté,
arrêt net sur contexte annulé ; `legacyDB` supprimé. 5.2 : 13 requêtes dans `platform/duckdb/profilerepo` derrière
`port.ProfileStore`, un seul `scanTemplate`. 5.3 : `internal/cronstore` et `platform/duckdb/worldrepo` ; le scheduler
n'importe plus `platform/duckdb` ; `ProbeErrors` testé sur un faux magasin. 5.4 : table de preuve des armes et collecte
Prestige dans `platform/duckdb`, pureté étendue à tout `analysis/`. 5.5 : `port.MediaTxOpener` / `port.MediaTx`
(CHECKPOINT après validation, writer rendu sur tous les chemins) ; `DBExecutor` et `DBWriter` dans `platform/dblease`.
5.6 : `domain.ErrRecordNotFound` (`port.ErrNotFound`), cinq comparaisons branchées (le plan en comptait six : cinq
trouvées sur pièces). 5.7 : fragments dans `platform/duckdb/sqlfrag`, noms inchangés, environ 200 appelants ;
`BuildNeighborsWhereClause` y part aussi (elle utilisait le fragment de fuseau). 5.8 : `internal/playerresolve` ;
`config` n'importe plus `platform/duckdb` ni `prestige`. 5.9 : `TestMetierSansSQL`, `TestPortSansTypeSQL`,
`TestSchedulerSansStockageDuckDB`, `TestAnalysisSansStockage`, plafond 0, allowlists vides. Mesures « moteur de base » :
fichiers important `database/sql` 18 puis 0 ; types `sql.` dans `port/` 4 puis 0 ; SQL détecté 41 occurrences puis 0.
Gate : build, vet (avec et sans `integration`), gofmt, `golangci-lint --new-from-rev` 0 défaut, `archlint`, code sans
appelant sans nouveauté : verts. `go test -tags=integration -p 1 ./...` complet : 3 échecs au premier passage, corrigés,
paquets concernés rejoués verts (pas de second passage complet en local). Témoins : 90 routes, `diff -r avant-4 apres-5b`
vide (aucune route d'écriture dans les témoins). `scripts/check_test_baseline.sh` : code 1 en local sur le délai de 300 s
de trois paquets (`ops`, `platform/duckdb`, `sync`), aucun test manquant ; la CI fait foi. Revue adversariale, une ronde,
deux relecteurs (écritures, lectures déplacées) : 0 P0, 0 P1, quatre P2 corrigés (test instable de `worldrepo`, arrêt de
l'amorçage sur contexte annulé, garde de la disposition démo aveugle au fichier déplacé, test de `decodeStringList`).

Gate : gate de l'étape 4 ; `go test -tags=integration -p 1 ./...` ; mesures « moteur de base » du §0
à zéro ; témoins identiques.

## 9. Étape 6 — Multi-titre

Effort : moyen.

- [x] 6.1 (DEP-1) Les 15 imports de `games/halo_infinite` dans `service/` disparaissent :
      `haloadapter.DefaultHaloProvider` est injecté au câblage (`NewHomeService` reçoit son
      fournisseur) ; `rankedplaylists` passe par l'adapter sémantique du titre ; `replaylabels`, déjà
      paramétré par titre (`Catalogue(repoRoot, titleSlug)`), quitte `games/halo_infinite` pour un
      paquet neutre ou passe par l'adapter (choix à écrire dans le compte rendu, sur pièces). Un
      titre sans rejeu rend `ErrCapabilityNotSupported`, jamais les libellés d'un autre titre.
- [x] 6.2 Garde-rail : étendre `no_title_package_in_analysis_test.go` à `service/` et
      `api/handlers`, plafond 0.
- [x] 6.3 Tests : cas de dégradation pour `halo_5` sur chaque service touché.

Résultat de l'étape 6 (2026-11-13). 6.1 : imports de paquets de titre dans `service/` hors tests : 15 de
`games/halo_infinite` (3 `haloadapter`, 2 `rankedplaylists`, 10 `replaylabels`) plus 1 de `games/halo_5/livesync`,
tous à 0. `NewHomeService(repo, provider)` reçoit son fournisseur au câblage. `replaylabels` : choix « par
l'adapter » pour le catalogue (il dépend du titre : icônes d'armes, clés d'armes du film), derrière
`port.ReplayLabelSource` dont l'implémentation de `api/wire` rend `games.ErrCapabilityNotSupported` pour un titre
sans rejeu ; la conversion de textes, indépendante du titre, part dans le paquet neutre `internal/replaytext`. Les
playlists classées sont appliquées par le `CatalogAdapter` du titre ; l'ensemble du classement mondial est fourni par
`cmd/server`. Le diagnostic d'apparence (verdicts) passe dans `domain`, son client Halo dans `api/wire`. 6.2 :
`TestServiceImporteAucunPaquetDeTitre`, plafond 0, allowlist vide, prouvé rouge par trois mutations ; `api/handlers`
reste tenu par `TestHandlersSansMoteurDeSync` (pas de doublon). 6.3 : cas `halo_5` sur chaque service touché. Gate :
gofmt, build, vet (avec et sans `integration`), `golangci-lint --new-from-rev` 0 défaut, `archlint`,
`-tags=integration -p 1` sur 53 paquets : code de sortie 0 ; code sans appelant : aucune nouveauté. Témoins : 90 routes
dont 12 `halo_5`, `diff -r avant-4 apres-6` vide après neutralisation de `since` / `until` du calendrier d'activité
(fenêtre glissante calculée depuis la date du jour : la référence datait de la veille).

Gate : gate de l'étape 5 ; mesure « ajouter un titre » du §0 à zéro ; témoins identiques pour les
deux titres.

## 10. Étape 7 — Front

Effort : lourd par le nombre de fichiers, faible en risque.

- [x] 7.1 (WEB-1) `tools/lint-cross-feature-imports.mjs` : reconnaître `_shared`, contrôler les
      imports relatifs, marqueur d'exemption par ligne et non par fichier. Corriger ce que le lint
      révèle : `_shared/useLocalFilterBar.tsx` (descendre `MultiSelectFilter`, `useActiveSeason` et
      `useFiltersPreview` dans `components/` ou `lib/`), `scoreTimelineKind` vers `lib/`,
      l'import de `ascension/AscensionCoachingTab.tsx:20`, les types de `squad/v2/types` lus par
      `_shared`.
- [x] 7.2 (WEB-2) `HomePrestigeSection.tsx` et `PrestigeSquadProgress.tsx` appellent les hooks de
      `prestige/hooks/`. Garde : aucun `useQuery` ni `useQueries` dans un `.tsx` de `features/`.
- [x] 7.3 (TEC-1) `CumulativeFragGapChart.tsx` et `OutcomeSequenceTape.tsx` passent par `ChartCard`
      (ou écoutent la palette). Garde : `<LazyECharts` monté seulement dans `components/charts/`.
      Contrôle : changer de palette, les deux graphes suivent sans rechargement.
- [x] 7.4 (TEC-2) Les 30 types manuels homonymes restants de `lib/api/types.ts` deviennent des
      réexports de `generated.ts` ; chaque écart de nullabilité ou de champ est corrigé chez les
      consommateurs. Étendre `response-types.guard.test.ts` à tous les schémas.
- [x] 7.5 (TEC-P2-3) `components/charts/` réexporte le type d'option ; les 72 fichiers l'importent
      de là. Garde : aucun import de `echarts` hors `components/charts/`.
- [x] 7.6 (TEC-P2-2) Les `fetch` de `admin/titles/queries.ts:94` et
      `spartan-customizer/SpartanCustomizerModal.tsx:85` passent par le client ;
      `explorer/explorerTableRows.ts:54` lit `API_BASE_URL`. Garde : `fetch` hors `lib/api/` et
      `lib/desktop/` sous allowlist datée (GitHub, sons et fichiers statiques du rejeu).
- [x] 7.7 (TEC-P2-1) Les 11 tableaux triés à la main passent sur TanStack Table. Garde : aucun
      `toggleSort` local dans `features/`.
- [x] 7.8 (WEB-P2-3) Moyenne, médiane, moyenne mobile : une seule implémentation dans `lib/charts/`
      (`meanOfValid` existe), les 9 moyennes inline et les deux moyennes mobiles s'y branchent ; la
      logique de `TimeseriesKdaDensity.tsx:57-105` va dans un fichier de logique testé.
- [x] 7.9 (WEB-P2-4) Le corps de `routes/.../replay.tsx` devient une page de `features/match-replay`
      ; la convergence de titre de `routes/.../$titleSlug.tsx:55-90` devient un hook.
- [x] 7.10 (WEB-P2-5, WEB-P2-6) Les 4 clés dérivées entrent dans `lib/query/keys.ts` ; un seul hook
      de bootstrap, les quatre sites s'y branchent.
- [x] 7.11 (WEB-P2-1, WEB-P2-2) `components/ui/match-card-assisted-frags.tsx` rejoint
      `features/_shared/assists/` (ou les quatre modules d'assistance descendent dans
      `components/`) ; `JoinPage.tsx` lit sa route par `getRouteApi` et n'importe plus `@/routes`.

Résultat de l'étape 7 (2026-11-13). 7.1 : lint des frontières à plafond 0, exemption par ligne, socle `_shared` sans
dépendance vers une feature (`MultiSelectFilter` dans `components/filters`, saisons dans `lib/seasons`). 7.2 : Accueil et
Prestige lisent par les hooks de Prestige ; garde « aucun `useQuery` dans un `.tsx` de `features/` ». 7.3 : les deux
graphes écoutent la palette (test automatique ; rendu à contrôler à l'écran). 7.4 : 21 types de `types.ts` puis 11
réponses rejoignent le contrat ; quatre corps de requête corrigés à la source dans le YAML du contrat (le serveur ne
change pas) ; homonymes hors `types.ts` de 78 à 5. Mesure « 0 type manuel homonyme » NON tenue au sens strict : restent,
en listes datées, 4 types d'entrée (`PeriodInput`, `SessionsInput`, `FilterContextInput`,
`EngagementTimeseriesRequest`), 2 réponses où le contrat diffère du serveur (`CareerHighlightMatchesResponse`,
`MediaPageResponse`) et 5 hors `types.ts` ; garde à plafond 0 hors listes. 7.5 : imports d'`echarts` hors
`components/charts/` 74 puis 0. 7.6 : `fetch` hors client 6 puis 4, tous en liste datée (GitHub, deux sons du rejeu, un
fichier statique). 7.7 : onze tableaux dans dix fichiers (`ApiHaloSection` en porte deux) convertis par `lib/table/useColumnSort`,
garde `noLocalSortToggle` à 0, dix tests d'ordre. 7.8 : `lib/charts/stats`.
7.9 : `ReplayPage` dans la feature, `useTitleConvergence`. 7.10 : 4 clés dérivées dans `queryKeys`, lecture unique du
bootstrap, deux gardes. 7.11 : les quatre modules d'assistance descendent dans `components/assists/`,
`REVERSE_SHARED_ALLOWED` supprimée, garde « aucune feature n'importe `@/routes` ». Gate (cache `node_modules/.tmp`
purgé) : typecheck, lint, 11979 tests, lint des frontières, knip, ratchet du contrat, couleurs, champs, build : code 0 ;
côté Go (YAML du contrat et un commentaire seulement) : `golangci-lint --new-from-rev`, `openapi-gen -check`, tests de
`internal/api` : code 0. Chaque garde prouvé rouge par mutation. Aucun contrôle à l'écran fait par les agents.

Gate : `make check-types` (cache `node_modules/.tmp` purgé), `npm run lint`, `make test-web`,
`node tools/lint-cross-feature-imports.mjs` ; mesures « front », « graphes » et « client HTTP » du
§0 tenues ; contrôle à l'écran par l'utilisateur de trois tableaux d'administration triés et des
deux graphes de 7.3.

## 11. Étape 8 — Clôture

- [x] 8.1 Rejouer les mesures du §0 et remplir le tableau « après » (tableau ci-dessous).
- [x] 8.2 Rejouer tous les témoins de 0.2 : identiques.
- [x] 8.3 `make gate-push`, `go test -tags=integration -p 1 ./...`, `make desktop-test`, CI verte. Réserves levées le
      2026-11-13 par le pilote (disque revenu à 179 Go libres) : `go test -tags=integration -p 1 -timeout 30m
      ./internal/platform/duckdb/...` code 0, 7 paquets verts (470 s pour `platform/duckdb`, au-delà du délai de 300 s du
      script de baseline) ; `make desktop-test` code 0 ; CI verte sur `07ceec0691`, les 29 jobs. Premier passage : voir le
      résultat : lint, replay-equiv, typecheck et lint web verts ; voie DuckDB de la baseline : 295 paquets sur 296 verts, un
      paquet (`platform/duckdb`) coupé par le délai local de 300 s ; `desktop-test` non lancé et reprise de
      `platform/duckdb` avec un délai large interrompue, l'un et l'autre pour cause de disque (moins de 4 Go libres) :
      reste au pilote. La CI est lue par le pilote.
- [x] 8.4 Mettre à jour `CLAUDE.md` et le skill `arch-rules` (nouveaux garde-rails, variables retirées),
      `docs/COMMANDS.md` et son pendant français, `.ai/project_map.md`. `docs/COMMANDS.md` et `docs/FR/COMMANDS.md` : rien à
      changer (couvert par 1.1 et 2.7 : toutes les commandes citées existent, aucune variable retirée n'y figure).
- [x] 8.5 Statuer chaque constat dans `.ai/AUDIT_ARCHITECTURE_2026-11-10.md` (section « Statut des constats » : 25 P1, 22 P2,
      12 écartés). `.ai/REGISTRE_REPORTS.md` : aucun item `[!]` des étapes 1 à 7 à y inscrire (les restes sont au §12).
- [x] 8.6 Entrée de `.ai/thought_log.md`.
- [x] 8.6 (suite) `scripts/disk-hygiene.ps1 -Lot 812-archi -Apply` : lancé par le pilote le 2026-11-13 après lecture de la
      CI, 33 Go libérés (`go-build-812-archi`, `golangci-812-archi`).

Résultat de 8.1 (2026-11-13, tête `3401008057` plus les commits de l'étape 8).

| Mesure | Départ (0.3) | Après | Tenue |
|---|---|---|---|
| fichiers de `api/handlers` important `platform/duckdb`, `persist` ou `sync` | 9 | 0 | oui (`TestHandlersSansStockageDuckDB`, `TestHandlersSansMoteurDeSync`) |
| fichiers de `service/` et `api/handlers` important `games/halo_infinite` ou `games/halo_5` | 16 | 0 | oui (`TestServiceImporteAucunPaquetDeTitre`, `TestHandlersSansMoteurDeSync`) |
| fichiers de `api/wire` et de la racine `api/` portant du SQL | 9 | 0 | oui (`TestApiSansSQL`) ; la recherche textuelle rend 2 commentaires (« SELECT courts »), pas du SQL |
| occurrences de `*service.X` dans `api/handlers` | 20 | 0 en code | oui (`TestHandlersSansTypeConcretDeService` : aucun import de `service`) ; 2 commentaires citent encore le nom d'un type |
| fichiers important `database/sql` dans `service`, `analysis`, `progression`, `scheduler`, `prestige`, `port` | 19 | 0 | oui (`TestMetierSansSQL`, `TestPortSansTypeSQL`) ; types `sql.` dans `port/` : 0 en code, 1 commentaire |
| fichiers lisant `os.Getenv` / `os.LookupEnv` hors `internal/config` | 22 | 13, tous en allowlist datée | oui au sens du garde (`TestAucuneLectureEnvHorsConfig`, 0 hors allowlist) ; non au sens « 0 fichier » |
| fichiers web important `echarts` hors `components/charts/` | 72 | 0 (tests compris) | oui (`chartsImportBoundary.guard.test.ts`) ; `<LazyECharts` hors `components/charts/` : 0 |
| `fetch` hors `lib/api/` et `lib/desktop/` | 7 | 4, tous en liste datée | oui au sens du garde (`rawFetch.guard.test.ts`) ; non au sens « 0 » : feedback GitHub, deux sons du rejeu, un fichier statique des véhicules |
| types manuels homonymes d'un schéma du contrat | 31 | 11 en listes datées (4 types d'entrée et 2 réponses dans `lib/api/types.ts`, 5 hors de ce fichier) | NON tenue au sens strict ; garde à plafond 0 hors listes (`response-types.guard.test.ts`) |
| commandes de `cmd/` | 180 | 98 | sans cible chiffrée ; décisions D8 |
| une seule source de la base URL du client | plusieurs | 1 (`lib/api/client.ts`) | oui |
| `toggleSort` local, `useQuery` dans un `.tsx` de `features/` | 11, 2 | 0, 0 | oui (`noLocalSortToggle`, `featureTsxNoQuery`) |

Résultat de 8.2 : `bash C:/t/812-archi/capture.sh C:/t/812-archi/apres-8`, 90 routes toutes en 200, `diff -r avant-4 apres-8` vide
(après `renorm.sh` de `avant-4` pour la neutralisation du calendrier d'activité, ajoutée le 2026-11-13).

Résultat de 8.3 (2026-11-13). `make gate-push` (GOWORK=off), étape par étape : `golangci-lint --new-from-merge-base=origin/main`
code 0, 0 défaut (une première passe avait rendu un `unused` sur l'enrobage de test de `convergePSA`, corrigé par le tag
`integration` de ses appelants) ; `replay-equiv` code 0 : 20 films identiques (deux premières passes à 20 échecs, causes du
poste : liste des xuids d'origine absente de ce worktree, puis parc de films non monté ; réglées par `LEVELUP_IDENTITES_ORIGINALES`
vers le fichier du dépôt principal et deux jonctions temporaires `data/cache/film_chunks` et `film_manifests`, retirées ensuite) ;
typecheck et lint web code 0 ; `check_test_baseline.sh tests` code 1 : voie des briques 218 s, tout présent, aucun échec ; voie
DuckDB 2 219 s, 295 paquets sur 296 verts, aucun test en échec ; le paquet `platform/duckdb` est coupé par le délai local de
300 s (`panic: test timed out after 5m0s`), d'où 71 tests de la baseline « absents » (tous encore paquets en pause au moment de la
coupure : `TestBuildPlaylistRankItem_*`, `TestCSRThresholdsRepo_*`, `TestUnrankedBadgeURL_*`, etc. ; ils existent tous dans le
code). Même cause que l'étape 5 (poste chargé, délai de 300 s pour ce paquet), non liée à ce chantier. Reprise de ce seul paquet
avec `-timeout 30m` : interrompue par mes soins quand le disque est passé sous 4 Go ; sa preuve reste à faire. Le
`go test -tags=integration -p 1 ./...` séparé n'a pas été lancé : la voie DuckDB du gate est la même commande (même
étiquette, `-p 1`, délai de 300 s). `make desktop-test` : non lancé (aucun dossier `target` Rust dans ce worktree, une
compilation à froid dépasse 4 Go). CI : lue par le pilote.

Résultat de 8.4 (2026-11-13). `CLAUDE.md` : section « Frontières de couches » (gardes Go et gardes web, où va le code), règle
multi-titre étendue à `service/` et `api/handlers`, règle ART 3 (dates d'allowlist, `OpenReadWrite`), recette d'enrichissement
à huit étapes (ART 7), modèle de kill-switch (`scheduler/auto_sync_convergence.go` : `shared_reader_legacy.go` n'en porte plus),
règle 13 (tri, bootstrap, graphes). Skill `arch-rules` : couches et paquets nouveaux (`importstore`, `cronstore`, `profilerepo`,
`worldrepo`, `sqlfrag`, `playerresolve`, `replaytext`, `dbfault`, `api/wire`), tableau des gardes, étape « ajouter un titre ».
`delivery-checklist` : même citation du modèle de kill-switch corrigée. `.ai/project_map.md` : îlot de fraîcheur du 2026-11-13.

Résultat de 8.5 (2026-11-13). 25 P1 et 22 P2 traités (TEC-2, FLG-P2-2 et TEC-P2-2 avec un reste nommé), 12 constats écartés
ou requalifiés statués, aucun sans statut. `convergePSA` : aucun appelant de production (relevé par l'étape 3, confirmé sur
pièces), seul enrobage de test des phases `convergePSACollect`, `flushAliasPairs`, `settlePSA` que `runPSAConvergence` enchaîne
en production ; décision : ne pas le garder dans la production (règle 7), il devient `convergence_psa_helper_test.go` (même
étiquette `integration` que ses quatre appelants), tests inchangés ; deux commentaires qui le citaient pointent vers
`settlePSA` et `runPSAConvergence`. Généralité des gardes multi-titre : `TestServiceImporteAucunPaquetDeTitre` était
général (tout répertoire d'`internal/games/` hors `paquetsInterTitres` est un titre) ; `TestHandlersSansMoteurDeSync`
nommait `halo_infinite` et `halo_5` : rendu général (`titresDeInternalGames`, `racinesInterditesAuxHandlers(titres)`),
`TestTitresDeInternalGames` empêche un parcours vide, mutation prouvée (un paquet `games/titre_hypothetique` importé par un
fichier de `api/handlers` puis de `service/` rougit les deux gardes ; retiré, vert). Plafonds et allowlists des gardes : inchangés.
Le paquet `archlint` a tourné vert dans la voie DuckDB du gate-push, après le dernier changement de code et les modifications de
documentation (`go test ./internal/archlint/... -timeout 30m` n'a pas été rejoué à part : disque sous 4 Go).

### Avancement

| Étape | État | Commit | Note |
|---|---|---|---|
| 0 | close | (commit de clôture de l'étape 0) | témoins et compteurs relevés le 2026-11-10 |
| 1 | close | `91727b0c30` (PR #191) | fusionnée le 2026-11-11 |
| 2 | close | poussée directement sur `release/v8.2.0` | close le 2026-11-11 |
| 3 | close | `dd5f4de69c` sur `release/v8.2.0` | close le 2026-11-12, CI verte ; une ronde de revue, P1 corrigé |
| 4 | close | poussée directement sur `release/v8.2.0` | close le 2026-11-12 ; une ronde de revue (lots 4.1 à 4.4), P1 corrigé |
| 5 | close | poussée directement sur `release/v8.2.0` | close le 2026-11-13 ; une ronde de revue, 0 P0, 0 P1, quatre P2 corrigés |
| 6 | close | poussée directement sur `release/v8.2.0` | close le 2026-11-13 |
| 7 | close | poussée directement sur `release/v8.2.0` | close le 2026-11-13 ; contrôle à l'écran par l'utilisateur en attente |
| 8 | close | poussée directement sur `release/v8.2.0` | close le 2026-11-13, CI verte ; réserves de 8.3 et 8.6 levées le même jour ; reste le contrôle à l'écran par l'utilisateur |

## 12. Découvertes d'exécution

- (étape 0) Ordres non déterministes dans des réponses d'API, à code constant : titres et capacités de
  `bootstrap`, listes `available_*` de la carrière, `commendations`, `career_csrs`, médailles et adversaires du
  détail de match, `patterns`, carte de chaleur des séries temporelles. Non traité.
- (étape 0) Départage aléatoire d'ex aequo : `top_medals` et `top_citations` de l'Accueil, `impact_roles` du
  détail de match, `levers[].rank`. Non traité.
- (étape 0) Le témoin ne couvre pas le rejeu. L étape 6 déplace `replaylabels` et touche `service/replayview` :
  son gate s'appuie donc en plus sur les tests de `service/replayview` et de `libs/film/replay` (équivalence),
  à lancer explicitement.
- (étape 1) `lusr_hyperparams_v2` et `player_squad_offset` ne sont écrits que par les outils de calibrage
  (`lusr_v2_ttt_batch`, `lusr_v2_squad_estimate`, gardés) ; `lusr_v2_squad_estimate` dépend de
  `LEVELUP_LUSR_V2_SQUAD_OFFSET=1`. À traiter en 2.3 : retirer l'interrupteur sans casser l'outil.
- (étape 1) La table `world_player_no_data` n'a plus d'écrivain de production ; son lecteur reste (masquage du
  classement). La description de la migration qui la crée parle encore d'un rattrapage. Non traité.
- (étape 1) `h5-roster-refetch` cite `cmd/h5-roster-topup`, supprimée. `lusr_v2_replay` porte quatre gamertags
  par défaut. `h5-events-backfill` fait un « DELETE puis INSERT ciblé » non vérifié contre les règles ART.
  Non traités.
- (étape 1) `TestBackfillRegistryNames_BaseTenueParUnAutreProcessus` (`cmd/levelup`) échoue sous charge et
  passe seul. Non traité.
- (étape 1) `domain.PlayerInfo.SteamID` (`steam_id` dans l'API) n'a aucun lecteur ; le retirer change
  `openapi.yaml`. Non traité.
- (étape 1) Des commentaires citent des commandes qui n'ont jamais existé dans `cmd/` : `diag_exec`,
  `diag_art_probe`, `h5-career-rank-xp`, `msal-poc`. Non traité.
- (étape 1) Le critère « aucune citation » de l'audit ne voit pas l'usage réel : deux commandes de la liste
  servaient (`admin`, `diag_q`). Toute suppression future de commande passe par une relecture de son rôle.
- (étape 1) `psa_index_repro_msr_fixture_test.go` (tag `psarepro`) nomme encore `QSquadExpectedWinProbTpl`,
  disparu : instantané d'une mesure datée, à retirer ou à garder. `SkillV2Service.PredictWin` n'a que son
  test pour appelant. Non traités.
- (étape 1) `scripts/disk-hygiene.ps1` teste encore `origin/feat/v75`, branche disparue, et affiche une
  erreur à chaque passage. Non traité.
- (étape 0) En profil dev, le premier démarrage rejoue les formules de note du jeu de démo (révision vide) :
  le harnais attend la fin de ce rejeu avant de capturer.
- (étape 3) `duckdb.OpenReadWriteShared` (même clé de cache que `OpenReadWrite`, pool de lecture : `internal/api/server.go`,
  `cmd/server/boot_providers.go`, `cmd/populate-career-rank-images`, `cmd/recompute_perfnote`, `internal/api/server_apiv1.go`)
  n'est couvert par aucun garde ; l'ADR 0030 D-3 ne nomme que `OpenReadWrite`. Non traité.
- (étape 3) `queries_career_encounters.go` (`Q24LUSRHistory`, lue par `StatsRepo.LoadLUSRHistory`) : lecture BRUTE de
  `match_skill_rank` sans filtre `rating_type` ni départage `written_at`. L'entrée d'allowlist (décision B7, 2026-07-10)
  est gardée, datée, avec la mention « non revérifié » : rien dans la requête ne dédoublonne les versions d'un match après
  un rejeu append-only, et le consommateur n'a pas été relu. Non traité.
- (étape 3) `engine_postsync_scoring.go` : si le burst d'écriture `match_intensity` est indisponible, le journal dit
  « retenté au prochain cycle ». Les intensités ne sont calculées que pour les matchs dont l'engagement est calculé
  dans la passe ; l'engagement tenté n'est pas recalculé (`force=false`), donc ce rejeu est vraisemblablement faux (même
  forme que ART-2). Non vérifié, non traité.
- (étape 3) `convergence.go` fait 691 lignes (666 au départ de l'étape, exemption connue du fichier) : le découpage vers un
  sous-paquet reste à faire (ADR 0027). Un alias dont l'upsert échoue en permanence fait désormais refetcher son match à chaque cycle jusqu'à `convergenceHorizon` ; le compteur
  `convergence_alias_upsert_failed_total` le rend visible. Non traité.
- (étape 3, revue) Le chemin per-match écrit des awards PSA mais ne pose jamais `psa_checked_at` (seul `settlePSA` l'écrit) :
  sur le jeu de démo, 59 matchs sur 68 (DEMO), 13 sur 25 et 14 sur 24 ont des awards sans marqueur, donc `blocked_psa` dans
  `snapshot_readiness` ne se lève pour eux que par vieillissement. Antérieur à l'étape ; le lever (poser le marqueur quand des
  awards existent) change ce qui est écrit. Non traité. Idem : si l'écriture du marqueur échoue après celle des awards, le
  match sort de la sélection sans marqueur (même état, antérieur).
- (étape 3) `CLAUDE.md` règle 7 (écritures) parle d'une recette d'enrichissement « en 3 étapes » : `persist/doc.go` en compte
  maintenant huit, dont la projection dans la vue. Non traité (étape 8.4).
- (étape 3) `internal/sync/schema.go` (`OpenPlayerDB`, `OpenSharedDB`, une trentaine d'appelants) : l'allowlist de
  `OpenReadWrite` ne dit rien de leur bail d'écriture ; aucun test ne vérifie qu'un appelant le prend. Non traité.

- Étape 4 (4.1, 4.2, 2026-11-12). `persist.BatchBuilder.SetLeadShare` n est appelée que par un test (arrivée par #200). `SyncJobService.WithProfiles` et `WithManualSyncCooldown` ne servent qu aux tests. Le moteur d auto-sync reste un type concret derrière `EngineBuilder` : le service ne se teste avec de faux lanceurs que par deux coutures non exportées. `notifications.EmitterFactory` a remplacé trois copies du type de fabrique sans garde-rail contre une nouvelle copie. `internal/port/post_sync_runner.go` cite encore `WithPostSyncDeltaHook`, supprimé. Garde-rail `archlint` du niveau de journal : les outils compilés sans cgo (`replay-build`, `replay-verite`) ne doivent pas importer `internal/config` (régression de 2.4, réparée le 2026-11-12). Non traités.

- Étape 4 (4.3, 2026-11-12). `api/handlers/home.go` importe encore `platform/duckdb` et `sharedprovider` pour classer une panne de lecture (allowlist datée du garde-rail) : à traiter en 4.7. L interface du profil joueur vit dans `progression/profile` et non dans `port/` (`profile` importe `port` : cycle) ; en sortir demande de déplacer les types de réponse vers `domain`. `settings.go` : 512 lignes. `SettingsHandler` sans `WithDatabaseMemory` n applique pas le plafond (avertissement au journal) : dépendance facultative, seul le serveur la branche. Pas de service pour la limite mémoire (aucune règle à y loger). Routes hors témoins : `GET /campaigns/{id}` et toutes les écritures (campagnes, `memory_limit_mb`, sauvegarde). Non traités.

- Étape 4 (4.4 et revue adversariale des lots 4.1 à 4.4, 2026-11-12). Revue : 0 P0 ; P1 « garde du writer partagé sur le chemin requête aveugle à `sync.NewSharedWriterAcquirer` » corrigé ; P1 « messages des 500 changés » réfuté sur pièces (`humacore.NewError` remplaçait déjà tout message 5xx par « internal error ») ; P2 corrigés : garde du pool (deux ouvertures nommées et inscrites en allowlist datée), étiquette Halo de la langue de repli liée en paramètre, garde anti-SQL élargi. Restent, non traités : `service.PlaylistExpansionService` ignore l erreur de `RecordWeight` (préexistant) ; `MediaRecipientsRepo.MatchParticipantXUIDs` ouvre la base partagée par `OpenReadForQuery` sous un nom que le garde du pool ne reconnaît pas ; aucune contrainte n impose `WithSessionsRecalc` et `WithDatabaseMemory` sur `SettingsHandler` ; `scripts/check_lease_enforcement.sh` cite deux chemins d avant `api/wire/` ; whitelist de `TestNoUnauthorizedSharedSocialMention` sans contrôle de péremption ; `port/dbexecutor.go` et `dbwriter.go` exposent `database/sql` (étape 5) ; KPM et PSPM : aucune fonction d `analysis` équivalente au bit près, formule déplacée telle quelle dans `analysis.PerMinute`.

- Étape 4 (4.5 à 4.8, 2026-11-12). `NewRouter` garde 9 paramètres et `ServiceRegistry` porte le magasin de sessions pour le seul relais vers `cmd/server`. La purge des sessions s'arrête désormais avec le contexte des tâches de fond (avant : contexte sans fin quand le watcher est absent), toujours hors du groupe d'attente. `port` importe maintenant `openspartan` et `platform/jobs` (`MediaIndexer` expose `*jobs.Store`) : à reprendre à l'étape 5 si `port` doit rester sans dépendance vers `platform`. `home.go` dépend encore de `platform/duckdb` par transitivité (`dbfault`) : le garde ne lit que les imports directs. Le passage `transfer.ErrImportRunning` vers `port.ErrSiteImportRunning` n'est testé que par un faux côté handler (pas de point d'injection dans le `Runner`). `port/repository_data.go` (649 lignes), `repository.go` (572) et `services.go` (544) dépassent 500 lignes : dette antérieure, non traitée. `docs/CHANGELOG.md` cite `NewSettingsHandlerWithIndexer` : historique, laissé. Non traités.

- Étape 5 (2026-11-13). Changements assumés, hors « à l'identique » : le dépôt de profil lit par `QueryRecovered` (une reprise sur handle invalidé, exigée par le garde de reprise) ; l'amorçage de l'import rend une erreur de lecture de la vue au lieu de l'ignorer ; les modèles de défis suggérés passent par `ListByTitle` (lecture des textes de tous les modèles du titre, réponse identique) ; clés de journal de l'après-import dérivées du nom d'étape (`post_import_metadata_unavailable` devient `post_import_open_metadata_failed`) ; une lecture en échec du classement mondial et de la fenêtre de profil est désormais journalisée ; le writer média est rendu avant la construction de la réponse. Deux messages d'erreur (Prestige « detach objectives », joueur de démo) ont été reformulés parce que le détecteur de SQL et le linter les refusaient : journal et 500 masqué seulement. Restes non traités : `port` importe `platform/jobs`, `openspartan` et `prestige` (`MediaIndexer` expose `*jobs.Store`, pas un type SQL) ; `config` importe encore `platform/duckdb/sharedprovider`, donc n'est pas importable sans cgo ; `scheduler` importe `sharedprovider` dans deux fichiers (`ErrDrainTimeout`) ; `analysis.SQLKnownMMR` reste dans `mmr_known.go`, invisible du détecteur ; `progression/profile/service.go` 580 lignes (dette antérieure, réduite) ; la vraie `mediaTx.Rollback` n'a pas de test d'échec en cours de transaction (faux seulement) ; le contrôle de contexte en cours de boucle de `PrimeBaseline` n'a pas de test ; 9 tests du scheduler importent `platform/duckdb` pour semer leurs bases ; `scripts/disk-hygiene.ps1` lit `origin/feat/v75`, branche disparue (erreur affichée, simulation rendue quand même). Disque du poste tombé à 1,7 Go libres pendant le lot (cache Go partagé : 57 Go).

- Étape 6 (2026-11-13). `TestHandlersSansMoteurDeSync` nomme `halo_infinite` et `halo_5` : un troisième titre ne serait pas refusé côté handlers (le garde de `service/` est-il général ? à vérifier à l'étape 8). Dix fichiers de test de `service/` et `api/handlers` importent des paquets de titre pour semer de vraies bases : les gardes ne lisent que le code hors tests. Le service de rejeu lit le catalogue mémoïsé (`Catalogue`) au lieu de le recharger à chaque requête. `WorldStatsEnricher.EnrichSeason` refuse un ensemble de playlists classées vide au lieu de retomber sur celui de Halo Infinite (le démarrage le fournit toujours). `scheduler` garde le même schéma `livesync` / `DeltaRunner`. `HomeService` accepte un fournisseur nul (cache puis « indisponible »). Non traités.

- Étape 7 (2026-11-13). Changements assumés : l'Accueil n'appelle plus les défis sans joueur choisi ; un palier inconnu retombe sur le palier calculé ; le brouillon TOML d'un titre envoie titre et langue ; une valeur non finie est traitée comme absente dans les moyennes (un ancien site pouvait la propager). Restes non traités : 11 types manuels en listes datées (voir le résultat de 7.4), dont 2 réponses où c'est le contrat qui est en retard sur le serveur ; le schéma de contrat `MediaItemRow` existe toujours, sans rapport avec le type renommé `NormalizedMediaItem` ; le lint des frontières ne balaie ni `lib/` ni `stores/` ; `medianSeconds` et `bucketMedian` ne sont pas branchés sur `lib/charts/stats` ; le fichier statique des véhicules est lu par `fetch` direct (liste datée) alors que celui des Spartan passe par le client ; `lib/playerProfile.ts` et `features/ascension/types.ts` dupliquent des définitions ; `heatmapColors.guard.test.ts` a un délai de 5 s fragile sous charge. À contrôler à l'écran par l'utilisateur : tri de trois tableaux (Carrière « rencontres », Classement, Admin « tâches »), les deux graphes de 7.3 sous changement de palette, le Rejeu 2D, `/join`, la connexion Xbox.

- Étape 8 (2026-11-13). Environnement : `make replay-equiv` (donc `gate-push`) exige dans un worktree la liste des xuids
  d'origine (`LEVELUP_IDENTITES_ORIGINALES` ou `data/film_identites/xuids_originaux.txt`) ET le parc de films
  (`data/cache/film_chunks`, `film_manifests`) : un worktree neuf n'a ni l'un ni l'autre et le gate rend 20 échecs. La voie DuckDB de `check_test_baseline.sh` coupe `platform/duckdb` à 300 s sur ce poste (aussi à l'étape 5) : le
  gate local ne passe pas sans réglage du délai. Disque : un `gate-push` complet a fait baisser C: d'environ 2 Go (observé) ; un dossier `go-build<N>` de 5,7 Go d'un autre processus traînait dans `%TEMP%` (non touché). `delivery-checklist` citait
  comme modèle de kill-switch un fichier qui n'en porte plus : corrigé avec 8.4. Non traités : `duckdb.OpenReadWriteShared`
  (déjà noté), `docs/COMMANDS.md` ne décrit pas les gardes d'`archlint`.

## 13. Hors périmètre

- L'intérieur de `libs/film` et les 14 commandes conservées (D8).
- Le SQL de `sync/` (D7) et la construction des options de graphes par les pages (D6).
- Les axes non passés par l'audit : duplication, erreurs avalées hors chemins d'écriture, couverture
  de tests, correction des données.
- `stores/createFilterStore.ts` et `lib/api/types.ts:1482`, qui importent `features/` : relevés
  hors périmètre par l'audit, à reprendre si le lint corrigé de 7.1 les fait remonter.
- (documentation des 4 commandes gardées, 2026-11-11) `backfill-media-hls` : `--delete-source` vaut `true` par
  défaut alors que `RunHLSTranscode` documente la conservation comme défaut sûr ; signalé en gras dans la
  documentation. Utilisateur, 2026-11-11 : « on garde pas les originaux, du moins pas en mode web app » : défaut
  voulu, rien à changer. La commande ne consulte pas le réglage de conservation des sources du bureau
  (`KeepMediaSources`) : sans objet tant qu'elle n'est lancée que sur le serveur. Non traité.
- (idem) `backfill_time_played` : UPDATE ligne à ligne sur `match_participants` hors `persist`, sans sauvegarde ;
  le commentaire « pas de risque ART » n'est pas vérifié. Non traité.
- (idem) `purge_corrupt_records` : `log.Fatalf` au milieu du parcours des joueurs (arrêt à mi-course), paquet
  `log` au lieu de `slog`, pas de sauvegarde. Non traité.
