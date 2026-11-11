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

- [ ] 2.1 (FLG-1, D3) Retirer `LEVELUP_USE_SHARED_PROVIDER` ; `LegacySharedReader` passe sous les
      tests ou disparaît si les tests se branchent sur le fournisseur.
- [ ] 2.2 (FLG-2, D4) Supprimer la réparation automatique des trous de note : `isLUSRAutoHealEnabled`,
      `LEVELUP_LUSR_AUTOHEAL_ENABLED`, le candidat de réparation et son déclenchement
      (`scheduler/data_health_lusr.go`), leurs tests. La détection, la jauge, l'alerte et le recalcul
      manuel du panneau de monitoring restent.
- [ ] 2.3 (FLG-3, FLG-P2-1, D5) Retirer les huit interrupteurs de D5 et leurs branches mortes ;
      `Dockerfile:315` passe de `LEVELUP_LOG_JSON=true` à `LEVELUP_LOG_FORMAT=json` dans le même
      commit, et le premier déploiement vérifie que les journaux restent en JSON. Pour
      les deux interrupteurs du LUSR : le comportement par défaut est conservé à l'identique, donc
      `LUSRRev` ne change pas ; le prouver par l'empreinte de `sync/scoring_revision_test.go`
      inchangée.
- [ ] 2.4 (FLG-P2-3) Faire lire par `internal/config`, une seule fois, `LEVELUP_LOG_LEVEL`,
      `LEVELUP_OAUTH_CLIENT_ID`, `LEVELUP_LEGACY_FILM_CACHE_DIR`,
      `LEVELUP_EVENTS_CONVERGENCE_CHUNK`, `LEVELUP_EVENTS_CONVERGENCE_YIELD_MS`,
      `LEVELUP_NOTIFY_VERSIONS`.
- [ ] 2.5 (FLG-P2-2) Dater ou retirer les deux branches de compatibilité
      (`handlers/media_paths.go:110-160`, `web/lib/title-routing/buildLegacyRedirect.ts`) : retrait
      si la mesure montre zéro usage, sinon date cible et critère en commentaire.
- [ ] 2.6 Garde-rail : `archlint` interdit `os.Getenv` et `os.LookupEnv` hors `internal/config` et
      `cmd/`, allowlist datée pour les données de déploiement restantes (chemins système).
- [ ] 2.7 Mettre à jour `docs/COMMANDS.md` et son pendant français, plus toute doc qui nomme une
      variable retirée (règle 15).

Gate : gate de l'étape 1, plus `go test -tags=integration -p 1 ./...` (le post-sync change de
chemin) ; `git grep -nE 'LEVELUP_(USE_SHARED_PROVIDER|POSTSYNC_BURST|LUSR_V2_(MODE_COUPLING|SQUAD_OFFSET)|SYNC_V2_DRYRUN|WATCHER_BROADCAST|PERSIST_NO_FETCH_CACHE|LOG_JSON)' -- apps docs`
vide ; témoins identiques.

## 6. Étape 3 — Écritures : erreurs tracées et garde-rails

Touche `persist/` et `sync/` : revue adversariale obligatoire, tests d'intégration obligatoires.
Effort : moyen.

- [ ] 3.1 (ART-2) Alias de joueurs : compter et journaliser chaque échec d'écriture ; ne poser le
      marqueur `psa_checked_at` qu'après un flush réussi, de sorte que le match soit repris au cycle
      suivant. Supprimer `upsertAliasesFromMatchJSON` (sans appelant). Test : un flush en échec
      laisse le match dans `selectMatchesMissingPSA`.
- [ ] 3.2 (ART-1) `persistMatchIntensities` : journal et compteur sur chaque échec.
- [ ] 3.3 (API-3, partie erreurs) `wire/registry_weapon_coverage.go:97-98` et
      `registry_catalog_expand.go:171-172` : journaliser les erreurs de `rows.Scan`.
- [ ] 3.4 (ART-3) Étendre `archlint/no_bare_duckdb_rw_open_test.go` à `duckdb.OpenReadWrite`, avec
      l'allowlist datée de l'ADR 0030 D-3.
- [ ] 3.5 (ART-4) Dater et rejustifier les entrées de `sync/shared_write_guard_test.go`,
      `sync/append_only_state_guard_test.go`, `platform/duckdb/no_raw_rating_reads_test.go` ; clé
      par chemin complet et non par nom de base ; corriger le motif périmé de `writes.go` et le
      commentaire de `writes.go:123`. Poser un test de forme : toute entrée d'allowlist de ces
      fichiers commence par une date.
- [ ] 3.6 (ART-P2-1) Compléter la recette de `persist/doc.go` (liste des colonnes de la vue) et
      poser un test qui lie `pmeColumnStage` et `enrichmentColumnStage`.

Gate : gate de l'étape 2 ; `go test -tags=integration -p 1 ./...` code de sortie 0.

## 7. Étape 4 — Frontières : la couche HTTP

Effort : lourd. Revue adversariale obligatoire. Chaque déplacement garde les signatures HTTP et les
réponses à l'identique (témoins).

- [ ] 4.1 (DEP-P2-2) Sortir de `worldenrich/wiring.go` l'import de `service` (le câblage monte dans
      `api/wire` ou `cmd/server`), pour que `service` puisse importer le suivi en direct sans cycle.
- [ ] 4.2 (API-2) Créer le port des jobs de synchronisation dans `internal/port/` (lancer une sync,
      un rattrapage, un alignement, un recalcul de sessions ; résolution du profil comprise) et son
      service. `handlers/backfill.go`, `sync_handler.go`, `sync_handler_align.go`,
      `settings_sessions_recalc.go` n'importent plus ni `sync` ni `games/halo_5/livesync`. La
      recherche de profil recopiée trois fois devient une fonction du service.
- [ ] 4.3 (API-1) Ports et services pour la progression (séries, records, historique, jalons : la
      jointure catalogue × jalons va dans le service), la campagne, le profil joueur, et la limite
      mémoire des bases (`settings_data.go`). Les quatre handlers n'importent plus `platform/duckdb`.
- [ ] 4.4 (API-3) Sortir le SQL des 9 fichiers de `api/wire` vers des dépôts de `platform/duckdb`
      derrière des ports ; les écritures de `registry_catalog_expand.go` (file de récupération,
      poids des playlists, DDL) vont dans `persist/` et `migration/`. Le calcul KPM et PSPM de
      `post_sync_progression_queries.go:231-236` passe par la fonction existante d'`analysis/`
      (vérifier l'existant, skill `go-features`). `ExpandPlaylistChildren` et la couverture des
      armes deviennent des services.
- [ ] 4.5 (API-4) Déplacer `commendation_handler.go`, `spa_handler.go`, `static_files.go` et
      `serveStaticFile` dans `api/handlers/` ; `media_audience.go` dans un service ;
      `startSessionPurgeLoop` dans `scheduler/`, déclaré comme tâche de fond par profil (ADR 0044).
- [ ] 4.6 (API-P2-1) Interfaces de `port/` pour `BootstrapService`, `PresenceService`, `SiteImport`,
      l'import OpenSpartan, `PlayerEngagementService`, `MediaIndexer`, la sync de titre.
- [ ] 4.7 Garde-rails `archlint`, plafond 0 : (a) `api/handlers` n'importe ni `platform/duckdb`
      (hors types d'erreur listés), ni `sync`, ni `persist`, ni `games/{slug}` ; (b) aucun littéral
      SQL dans `api/wire` ni à la racine `api/` ; (c) aucun `*service.X` dans `api/handlers`.
- [ ] 4.8 Tests : `httptest` avec faux port pour chaque handler touché ; faux dépôt pour chaque
      service créé ; DuckDB en mémoire pour chaque dépôt créé.

Gate : gate de l'étape 3 ; mesures « cadre HTTP » du §0 à zéro ; témoins identiques.

## 8. Étape 5 — Frontières : le stockage

Effort : lourd. Revue adversariale obligatoire.

- [ ] 5.1 (STO-1, STO-P2-2) Import OpenSpartan : l'amorçage des lignes d'enrichissement passe par le
      persister existant (`persist/post_sync_enrichment_persister.go`), sans arrêt à la première
      erreur. Les `*sql.DB` disparaissent des signatures des services d'import (dépôts derrière
      ports) ; le champ `legacyDB` réservé aux tests disparaît.
- [ ] 5.2 (STO-2) `progression/profile` : les 13 requêtes vont dans un dépôt de `platform/duckdb`
      derrière un port ; le service ne tient plus de `*duckdb.DB` ; `scanTemplate` n'existe plus
      qu'une fois.
- [ ] 5.3 (STO-3) `scheduler` : les sondes de santé et la purge des rejeux lisent par un dépôt ; le
      scheduler n'importe plus `platform/duckdb`. Une sonde en erreur reste comptée dans
      `ProbeErrors` et le rapport le montre.
- [ ] 5.4 (STO-4) Sortir d'`analysis/` `weapon_evidence.go` et `prestigetuning/collect.go` (la
      requête dans `platform/duckdb`, le calcul reste). Étendre le test de pureté à tout `analysis/`.
- [ ] 5.5 (STO-P2-1) `port.DBExecutor` et `port.DBWriter` ne rendent plus de type SQL : la
      transaction média (ouverture, validation, CHECKPOINT) devient une méthode du dépôt.
- [ ] 5.6 (STO-P2-4) Erreur de domaine « introuvable » dans `port/`, rendue par les dépôts ; les 6
      comparaisons à `sql.ErrNoRows` de la couche métier s'y branchent.
- [ ] 5.7 (STO-P2-5) Déplacer les fragments SQL d'`analysis/` (`sql_fragments.go`, `sql_liste.go`,
      les requêtes d'`identity.go` et `identity_annuaire.go`, `campaign_exclusion.go:107`) dans un
      paquet de `platform/duckdb` ; les garde-rails qui tiennent le fragment de fuseau canonique
      (règle 8) suivent le déplacement.
- [ ] 5.8 (DEP-P2-1) `config` n'importe plus `platform/duckdb` ni `prestige` : `ResolvePlayer` et le
      fournisseur partagé par titre montent dans le câblage.
- [ ] 5.9 Garde-rails `archlint`, plafond 0 : ni `database/sql` ni littéral SQL dans `service/`,
      `analysis/`, `progression/`, `scheduler/`, `prestige/` ; aucun type `sql.` dans `port/`.

Gate : gate de l'étape 4 ; `go test -tags=integration -p 1 ./...` ; mesures « moteur de base » du §0
à zéro ; témoins identiques.

## 9. Étape 6 — Multi-titre

Effort : moyen.

- [ ] 6.1 (DEP-1) Les 15 imports de `games/halo_infinite` dans `service/` disparaissent :
      `haloadapter.DefaultHaloProvider` est injecté au câblage (`NewHomeService` reçoit son
      fournisseur) ; `rankedplaylists` passe par l'adapter sémantique du titre ; `replaylabels`, déjà
      paramétré par titre (`Catalogue(repoRoot, titleSlug)`), quitte `games/halo_infinite` pour un
      paquet neutre ou passe par l'adapter (choix à écrire dans le compte rendu, sur pièces). Un
      titre sans rejeu rend `ErrCapabilityNotSupported`, jamais les libellés d'un autre titre.
- [ ] 6.2 Garde-rail : étendre `no_title_package_in_analysis_test.go` à `service/` et
      `api/handlers`, plafond 0.
- [ ] 6.3 Tests : cas de dégradation pour `halo_5` sur chaque service touché.

Gate : gate de l'étape 5 ; mesure « ajouter un titre » du §0 à zéro ; témoins identiques pour les
deux titres.

## 10. Étape 7 — Front

Effort : lourd par le nombre de fichiers, faible en risque.

- [ ] 7.1 (WEB-1) `tools/lint-cross-feature-imports.mjs` : reconnaître `_shared`, contrôler les
      imports relatifs, marqueur d'exemption par ligne et non par fichier. Corriger ce que le lint
      révèle : `_shared/useLocalFilterBar.tsx` (descendre `MultiSelectFilter`, `useActiveSeason` et
      `useFiltersPreview` dans `components/` ou `lib/`), `scoreTimelineKind` vers `lib/`,
      l'import de `ascension/AscensionCoachingTab.tsx:20`, les types de `squad/v2/types` lus par
      `_shared`.
- [ ] 7.2 (WEB-2) `HomePrestigeSection.tsx` et `PrestigeSquadProgress.tsx` appellent les hooks de
      `prestige/hooks/`. Garde : aucun `useQuery` ni `useQueries` dans un `.tsx` de `features/`.
- [ ] 7.3 (TEC-1) `CumulativeFragGapChart.tsx` et `OutcomeSequenceTape.tsx` passent par `ChartCard`
      (ou écoutent la palette). Garde : `<LazyECharts` monté seulement dans `components/charts/`.
      Contrôle : changer de palette, les deux graphes suivent sans rechargement.
- [ ] 7.4 (TEC-2) Les 30 types manuels homonymes restants de `lib/api/types.ts` deviennent des
      réexports de `generated.ts` ; chaque écart de nullabilité ou de champ est corrigé chez les
      consommateurs. Étendre `response-types.guard.test.ts` à tous les schémas.
- [ ] 7.5 (TEC-P2-3) `components/charts/` réexporte le type d'option ; les 72 fichiers l'importent
      de là. Garde : aucun import de `echarts` hors `components/charts/`.
- [ ] 7.6 (TEC-P2-2) Les `fetch` de `admin/titles/queries.ts:94` et
      `spartan-customizer/SpartanCustomizerModal.tsx:85` passent par le client ;
      `explorer/explorerTableRows.ts:54` lit `API_BASE_URL`. Garde : `fetch` hors `lib/api/` et
      `lib/desktop/` sous allowlist datée (GitHub, sons et fichiers statiques du rejeu).
- [ ] 7.7 (TEC-P2-1) Les 11 tableaux triés à la main passent sur TanStack Table. Garde : aucun
      `toggleSort` local dans `features/`.
- [ ] 7.8 (WEB-P2-3) Moyenne, médiane, moyenne mobile : une seule implémentation dans `lib/charts/`
      (`meanOfValid` existe), les 9 moyennes inline et les deux moyennes mobiles s'y branchent ; la
      logique de `TimeseriesKdaDensity.tsx:57-105` va dans un fichier de logique testé.
- [ ] 7.9 (WEB-P2-4) Le corps de `routes/.../replay.tsx` devient une page de `features/match-replay`
      ; la convergence de titre de `routes/.../$titleSlug.tsx:55-90` devient un hook.
- [ ] 7.10 (WEB-P2-5, WEB-P2-6) Les 4 clés dérivées entrent dans `lib/query/keys.ts` ; un seul hook
      de bootstrap, les quatre sites s'y branchent.
- [ ] 7.11 (WEB-P2-1, WEB-P2-2) `components/ui/match-card-assisted-frags.tsx` rejoint
      `features/_shared/assists/` (ou les quatre modules d'assistance descendent dans
      `components/`) ; `JoinPage.tsx` lit sa route par `getRouteApi` et n'importe plus `@/routes`.

Gate : `make check-types` (cache `node_modules/.tmp` purgé), `npm run lint`, `make test-web`,
`node tools/lint-cross-feature-imports.mjs` ; mesures « front », « graphes » et « client HTTP » du
§0 tenues ; contrôle à l'écran par l'utilisateur de trois tableaux d'administration triés et des
deux graphes de 7.3.

## 11. Étape 8 — Clôture

- [ ] 8.1 Rejouer les mesures du §0 et remplir le tableau « après ».
- [ ] 8.2 Rejouer tous les témoins de 0.2 : identiques.
- [ ] 8.3 `make gate-push`, `go test -tags=integration -p 1 ./...`, `make desktop-test`, CI verte.
- [ ] 8.4 Mettre à jour `CLAUDE.md` et le skill `arch-rules` (nouveaux garde-rails, variables
      retirées), `docs/COMMANDS.md` et son pendant français, `.ai/project_map.md`.
- [ ] 8.5 Statuer chaque constat dans `.ai/AUDIT_ARCHITECTURE_2026-11-10.md` (champ « Décision ») ;
      alimenter `.ai/REGISTRE_REPORTS.md` avec tout item `[!]`.
- [ ] 8.6 Entrée de `.ai/thought_log.md` ; `scripts/disk-hygiene.ps1 -Lot <lot> -Apply`.

### Avancement

| Étape | État | Commit | Note |
|---|---|---|---|
| 0 | close | (commit de clôture de l'étape 0) | témoins et compteurs relevés le 2026-11-10 |
| 1 | close | `91727b0c30` (PR #191) | fusionnée le 2026-11-11 |
| 2 | à faire | | |
| 3 | à faire | | |
| 4 | à faire | | |
| 5 | à faire | | |
| 6 | à faire | | |
| 7 | à faire | | |
| 8 | à faire | | |

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

## 13. Hors périmètre

- L'intérieur de `libs/film` et les 14 commandes conservées (D8).
- Le SQL de `sync/` (D7) et la construction des options de graphes par les pages (D6).
- Les axes non passés par l'audit : duplication, erreurs avalées hors chemins d'écriture, couverture
  de tests, correction des données.
- `stores/createFilterStore.ts` et `lib/api/types.ts:1482`, qui importent `features/` : relevés
  hors périmètre par l'audit, à reprendre si le lint corrigé de 7.1 les fait remonter.
