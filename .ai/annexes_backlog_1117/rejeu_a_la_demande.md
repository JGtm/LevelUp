# Rejeu à la demande : conception (item G.0 du plan des entrées du backlog du 2026-11-17)

Conception à relire par le pilote avant G.1 et G.2 ; écrite le 2026-11-17 sur `bl1117/g`, par lecture seule. Cadre
décidé (D-4) : bureau et développement seulement, propriétaire ou administrateur, jeton du joueur de la page, file
existante, action d'administration de l'hébergé inchangée. Chemins : `go/` = `apps/go-api/internal/`, `cmd/` =
`apps/go-api/cmd/`, `web/` = `apps/web/src/`.

## 1. Parcours d'une demande, du bouton au rejeu rangé

| # | Étape | Existe | À écrire |
|---|---|---|---|
| 1 | Le web sait que l'action existe : `capabilities.can_generate_replays` du bootstrap | lecture des capacités : `web/lib/capabilities/deploymentCapabilities.ts:18-63` | la capacité (§7) |
| 2 | Le web lit l'état du match : `GET .../matches/{match_id}/replay/generation` | rien | route, port, service (§2, §6) |
| 3 | Bouton « Générer le rejeu » si l'état est `generatable` (ou `failed` réessayable) | état vide nu : `web/features/match-replay/ReplayPage.tsx:245-249` ; lien seul : `web/features/match-view/MatchHeader.replayLink.tsx:36` | composant partagé (§8) |
| 4 | `POST .../replay/generate` passe les gardes du groupe joueur | titre actif, propriété (`go/api/server_apiv1.go:805`, `:825`), participation (`:829`), capacité de rejeu du titre (`:884-886`) | montage sous la capacité de profil (§6) |
| 5 | Le handler résout le titre (`ctxkeys.TitleSlug`) et l'identifiant, appelle le port | patron `ServiceFactory` (`go/api/handlers/career.go:25`) | `handlers/replay_generation.go` |
| 6 | Le service contrôle : capacité `film.replay_artifact`, match connu, mode géré, bit « film absent », rejeu déjà là, jeton présent, plafond | prédicats existants : `port/replay_registry.go:18`, `port/analysis_probe.go:27`, `platform/duckdb/analysis_probe_repo.go:91` | `service/replay_generation_service.go` |
| 7 | La demande entre dans la file, persistée, idempotente ; réponse 202 | file en mémoire du seul lot de régénération : `go/replayregen/service.go:98-125` | demandes explicites (§3) |
| 8 | Le travailleur attend son tour : pause, mode jeu, synchronisation | `waitTurn`, `go/replayregen/service.go:379-400` | réutilisé tel quel |
| 9 | Film absent de l'archive : jeton du joueur, manifeste, morceaux, archivage | chaîne de téléchargement `go/sync/filmfetch/films.go:63-112`, archivage `go/filmarchive/write.go:125` | `wire/registry_replay_generation_film.go` (§4, §5) |
| 10 | Film introuvable (404/410) : bit `MBitFilmAbsent` par l'écrivain existant | `go/sync/killcollector/registry_flags.go:53-63` (`marquerRegistre`) | exporter une entrée (§4) |
| 11 | Reconstruction hors processus, sous `film_decode.lock` | `Deps.Rebuild` = `regenerateReplay` (`go/api/wire/registry_replay_regen.go:87-99`) -> `RunReplayBuild` (`registry_replay_build.go:39-73`), verrou `go/filmrun/solo.go:64-73`, occupé -> `ErrBusy` (`registry_replay_regen.go:102-107`) et nouvel essai (`replayregen/service.go:459-476`) | rien |
| 12 | Rejeu rangé, dérivations rejouées | `deriverArtefactRange` (`registry_replay_regen.go:97`) | rien |
| 13 | Archive désactivée : film retiré après la cuisson | même règle que `retirerFilmNonArchive` (`go/sync/replayartifacts/cuisson.go:392-402`) | appel dans la file (§4) |
| 14 | Le web voit `ready`, invalide rejeu, vue match, accueil ; `replay_available` devient vrai | `os.Stat` à chaque lecture : `go/service/match_view_builders_header.go:245` | invalidations (§8) |

## 2. Le port `port.ReplayGeneration` et les états

Fichier `go/port/replay_generation.go` (types de données dans `go/domain/replay_generation.go`, sans dépendance).

```go
// port : ce que le handler demande (une instance par requête, liée au joueur de la page par la fabrique).
type ReplayGeneration interface {
    // RequestGeneration met le match en file. created=false : déjà en file ou en cours (idempotence).
    RequestGeneration(ctx context.Context, titleSlug, matchID string) (domain.ReplayGenerationStatus, bool, error)
    // GenerationStatus rend l'état du match, sans rien lancer.
    GenerationStatus(ctx context.Context, titleSlug, matchID string) (domain.ReplayGenerationStatus, error)
}
// ReplayGenerationQueue : la file (satisfaite par *replayregen.Service).
type ReplayGenerationQueue interface {
    Enqueue(d domain.ReplayGenerationDemand) (domain.ReplayGenerationStatus, bool, error)
    DemandStatus(titleSlug, matchID string) (domain.ReplayGenerationStatus, bool)
}
// ReplayMatchProbe : ce que la base et le disque disent du match (implémenté au câblage, sans SQL dans wire).
type ReplayMatchProbe interface {
    ProbeReplayMatch(ctx context.Context, titleSlug, matchID string) (ReplayMatchProbeResult, error)
}
type ReplayMatchProbeResult struct {
    FullID string // match_id complet du registre
    Known, UnmanagedMode, FilmAbsent, ArtifactPresent, FilmArchived bool
}
// ReplayTokenPresence : le joueur a-t-il un jeton de rafraîchissement au magasin (lecture, aucun échange).
type ReplayTokenPresence func(xuid string) bool
```

Domaine (`go/domain/replay_generation.go`) : `ReplayGenerationStatus{State, Reason, Phase string; Position int;
WaitingForGame, Paused, Retryable bool}` (étiquettes JSON `state`, `reason`, `phase`, `position`,
`waiting_for_game`, `paused`, `retryable`, avec `enum:` et `doc:` comme `domain/replay_regen.go:9-28`) ;
`ReplayGenerationDemand{TitleSlug, MatchID, PlayerSlug, XUID, Gamertag string; RequestedAt time.Time}` ;
erreurs `ErrReplayGenerationQueueFull`, `ErrReplayFilmUnavailable`, `ErrReplayGenerationNotSupported`,
`ErrReplayGenerationUnbound`, `ErrReplayFilmGone`, `ErrReplayNoToken`. Le domaine porte les erreurs pour que
`replayregen`, `service` et `handlers` les partagent par `errors.Is` sans s'importer.

États servis (codes seulement, le web écrit les textes, comme `ReplayRegenStatus`) :

| `state` | Sens | `reason` possibles | Bouton |
|---|---|---|---|
| `generatable` | rien n'empêche de générer | - | Générer |
| `queued` | en file ; `position` (1 = suivant), `waiting_for_game`, `paused` | - | aucun |
| `running` | en cours ; `phase` = `downloading` puis `building` | - | aucun |
| `failed` | dernier essai en échec dans ce processus | `film_not_ready`, `fetch_failed`, `token_unavailable`, `build_failed` (réessayables) ; `map_not_in_catalog`, `unknown_film_key` (non) | Réessayer si `retryable` |
| `film_unavailable` | bit `MBitFilmAbsent` posé, ou 404/410 constaté | - | aucun |
| `not_supported` | ce match ne se génère pas ici | `mode_unmanaged`, `match_unknown`, `no_token` | aucun |
| `ready` | l'artefact est rangé | - | lien vers le rejeu |

`ready` s'ajoute aux six états demandés : sans lui, une lecture après succès retomberait sur `generatable`.
Un titre sans `film.replay_artifact` ne reçoit pas d'état : `games.ErrCapabilityNotSupported` -> 503.

Ordre d'évaluation de `GenerationStatus` (service) : capacité du titre (sinon 503) ; demande connue de la file
(`queued`, `running`, `failed` récent) ; artefact présent -> `ready` ; match inconnu -> `not_supported/match_unknown` ;
mode non géré -> `not_supported/mode_unmanaged` ; bit posé -> `film_unavailable` ; film non archivé ET aucun jeton ->
`not_supported/no_token` (un film déjà archivé se génère sans jeton) ; sinon `generatable`.

Transitions : `generatable` -POST-> `queued` -tour-> `running/downloading` -> `running/building` -> `ready` ;
`downloading` -404/410-> `film_unavailable` (bit posé) ; `running` -erreur-> `failed` -POST-> `queued` ;
`queued` ou `running` -arrêt-> `queued` à l'ouverture (le film recommence) ; `film_unavailable`, `not_supported`, `ready` : terminaux.

## 3. Extension de `replayregen.Service`

Aujourd'hui : un lot de candidats tirés des rejeux déjà rangés (`scan.go:57-83`), un film à la fois (`process`,
`service.go:424`), pause (`:240`), reprise par un drapeau (`:192-211`), aucune entrée pour un match sans artefact.

- **Structure.** Dans `Service` : `demands []demandEntry` (FIFO, `entry{Demand; phase string}`), `results
  map[string]demandResult` (issues terminales du processus, clé `titre/short8`, 50 au plus, la plus ancienne sort),
  `demandLooping bool`, `slot sync.Mutex` (le jeton « un film à la fois », tenu pendant `processDemand` ET pendant
  `process`). `Deps` gagne `FetchFilm func(ctx, d domain.ReplayGenerationDemand) error`, `ReleaseFilm func(ctx,
  titleSlug, matchID string)` (archive désactivée), `Demands DemandStore`, `MaxDemands int` (défaut 10).
  `Deps` et ses défauts (`service.go:40-95`) déménagent tels quels dans `replayregen/deps.go` : `service.go` fait
  477 lignes et dépasserait 500.
- **Persistance.** `DemandStore{LoadDemands() ([]Demand, error); SaveDemands([]Demand) error}` ; implémentation
  fichier `replayregen/demand_store.go` sur `title.PathResolver.ReplayGenerationQueuePath()` =
  `data/global/admin_state/replay_generation_queue.json` (à créer dans `go/domain/title/paths_replay_generation.go`,
  sous `AdminStateDir`, `registry.go:688-690`), écrit par `platform/atomicfile.WriteFile`. Le fichier porte titre,
  match complet, slug, xuid, gamertag, date ; aucun secret. Réécrit sous `s.mu` à chaque ajout et à chaque sortie
  de file, comme `persistPendingLocked` (`service.go:273-281`) ; un échec d'écriture se journalise en Warn.
- **Reprise.** `ResumeDemands()` relit le fichier, remet chaque demande en `queued` dans l'ordre, lance la boucle.
  Une demande en cours à l'arrêt recommence entière (l'archivage écrit le manifeste en dernier,
  `filmarchive/write.go:117-124` : une coupure ne laisse jamais un film à moitié lisible).
- **Idempotence.** Clé `titre/FilmShortMatchID(matchID)`. Déjà en file ou en cours : rend l'état, `created=false`.
  Échec récent dans `results` : la demande le remplace (nouvel essai). L'artefact à jour est revérifié au moment du
  tour (`replaybuild.ArtifactUpToDate`) : une demande devenue inutile sort en `ready` sans rien lancer.
- **Plafond.** `len(demands) >= MaxDemands` -> `domain.ErrReplayGenerationQueueFull`, vérifié sous `s.mu` dans le même
  geste que l'ajout. Dix films : quelques minutes de travail et au plus quelques centaines de Mo de films.
- **Ordre avec la régénération du lot.** Le film en cours va à son terme. `run` (`service.go:284-308`) appelle
  `s.serveDemands(ctx)` avant chaque candidat : les demandes passent avant le candidat suivant. Hors lot,
  `Enqueue` lance `demandLoop` si aucune boucle ne tourne ; `start()` du lot pendant `demandLoop` : les deux boucles
  se partagent `slot`, `demandLoop` sort dès que `running` est vrai ; `finish` (`:316-345`) relance `demandLoop` s'il
  reste des demandes. Le décompte du lot (`Stale`, `Total`, `Done`) ne compte jamais les demandes.
- **Portes.** Avant le téléchargement ET avant la reconstruction : `waitTurn` (pause, mode jeu
  `gameproc.Mode.HeavyWorkAllowed` `go/platform/gameproc/mode.go:70-72`, synchronisation). La pause n'existe que
  pendant un lot (`Pause` rend `ErrNotRunning` sinon, `:240-245`) : un lot en pause retient aussi les demandes
  (`paused=true` dans l'état servi), sa reprise les relâche.
- **Classement.** `nil` -> `ready` ; `ErrReplayFilmGone` -> `film_unavailable` ; `finalise.ErrFilmNonFinalise` ->
  `failed/film_not_ready` ; `ErrReplayNoToken` -> `failed/token_unavailable` ; `ErrMapNotInCatalog`,
  `ErrUnknownFilmKey` (`service.go:410-412`) -> `failed` non réessayable ; `ErrModeNonGere` -> `not_supported` ;
  `ctx` annulé -> reste en file persistée ; autre -> `failed/build_failed` ou `fetch_failed`. Chaque issue :
  compteur expvar (`replay_generation_{queued,done,failed,film_gone}_total`) et une ligne `slog` avec `titleSlug`,
  `match_id`, `player`, `err`.
- **Liaison.** `Bind` inchangé ; `cmd/server/boot_replayregen.go:20-29` lie si `ReplayRegen || GenerateReplays`,
  appelle `ResumeIfPending` et `Rescan` sous `ReplayRegen` seulement, `ResumeDemands` sous `GenerateReplays`
  seulement, après `armGameWatch` (ordre écrit `boot_replayregen.go:6-7`). Le profil `dev` n'a que les demandes.
- Doc du paquet (`replayregen/scan.go:1-13`) : ajouter la seconde entrée (demande explicite d'un match).

## 4. Téléchargement et archivage du film d'un match

Fonction `ServiceRegistry.fetchFilmForDemand(ctx, d)` dans `go/api/wire/registry_replay_generation_film.go`,
branchée sur `Deps.FetchFilm` par `ReplayRegenDeps` (`registry_replay_regen.go:28-42`).

1. Racine `paths.FilmArchiveDir()` (`go/domain/title/roots.go:97`, réglage `film_archive_dir`), clé
   `title.FilmShortMatchID(d.MatchID)`.
2. Film complet à l'archive (`filmcache.FilmComplet`, comme `cmd/levelup/cmd_archive_films.go:319-327`) : rien à faire,
   AUCUN jeton n'est pris.
3. Sinon : jetons du joueur (§5), client `syncpkg.NewHaloAPIClient(spartan, clearance, replayManifestRPS)`
   (`registry_build_queue.go:36`, `:163`), `GetFilmChunks(ctx, d.MatchID)` : un manifeste puis tous les morceaux
   (`filmfetch/films.go:63-65`, cache disque d'abord puis API, `manifeste.go:25-47`) ; `GetFilmChunks` et non
   `GetMatchFilm` (raison écrite `cmd_archive_films.go:235-237`).
4. Écriture `filmarchive.Write(ctx, root, short, chunks)` (copie de la conversion de `killcollector/remote_films.go:138-145`).
   Échec : `fetch_failed`, Error journalisé, jamais muet.
5. **Film expiré** : `found == false`, zéro morceau, ou `filmfetch.IsFilmGoneErr(err)` (`filmfetch/types.go:69`, même
   classement que `killcollector/collector_run.go:118-122`) -> `killcollector.MarquerFilmAbsent(ctx, acquire,
   d.MatchID)` puis `ErrReplayFilmGone`. `MarquerFilmAbsent` (nouveau `go/sync/killcollector/film_absent_mark.go`)
   n'est qu'une entrée exportée sur `marquerRegistre(ctx, db, matchID, matchflags.MBitFilmAbsent)`
   (`registry_flags.go:53-63`) : même requête, même lease, aucune écriture neuve. `acquire` =
   `syncpkg.NewSharedWriterAcquirer(r.sharedProviderForTitle(slug), paths.SharedDBPath(slug),
   "replay_generation_film_absent", 8*time.Second)` (`go/sync/engine_acquire.go:168-177`,
   `wire/registry_build_queue_derivations.go:78`). Un échec du marqueur se journalise ; l'état reste
   `film_unavailable` pour ce processus.
6. **Film non finalisé** (`finalise.ErrFilmNonFinalise`, `filmfetch/manifeste.go:96-103`) : rien n'est écrit ni
   marqué, `failed/film_not_ready`, réessayable.
7. Après une reconstruction réussie, `ReleaseFilm` retire le film si le réglage « archiver les films » est faux,
   comme la cuisson (`cuisson.go:392-402` ; lecture du réglage : patron `replayartifacts/artifacts.go:129-131`).

## 5. Le jeton, pas à pas, et pourquoi aucun autre jeton ne sert

1. Le slug vient du CHEMIN, jamais du corps. La fabrique `ServiceRegistry.ReplayGeneration(ctx, slug)` appelle
   `r.resolve(ctx, slug)` (comme `wire/registry_pages.go:145-151`) et lie le service à `pdb.XUID` et
   `pdb.Gamertag` : le profil de la page, déjà admis par `RequirePlayerOwnership` (écriture : propriétaire ou
   administrateur, `middleware/require_player_ownership.go:34-47`).
2. Au POST, le service vérifie `ReplayTokenPresence(xuid)` : `MultiUserTokenStore.Load(xuid)` et refresh token non
   vide (lecture du fichier, aucun échange). Absent et film non archivé : `not_supported/no_token`, 409.
3. La demande persistée porte ce xuid. Rien d'autre n'en choisit un.
4. Au tour du film (§4 étape 3) : `tokencache.GetCachedPlayerTokens(d.XUID)`
   (`platform/auth/tokencache/player_token_cache.go:91-99`, lecture seule) ; à défaut
   `auth.RefreshHaloTokensViaStoreFirst(ctx, auth.NewMultiUserTokenStore(paths.WatcherTokensDir()), r.provider,
   d.XUID, d.Gamertag)` (`platform/auth/cli_refresh.go:40-99` ; le magasin prend le chiffreur du processus, donc
   DPAPI sur le bureau, `multi_user_token_store.go:121-128`), puis `tokencache.SetCachedPlayerTokens(d.XUID, ...)`
   (`:130-137`) pour que les autres chemins du processus ne rafraîchissent pas le même RT dans la foulée.
5. `(nil, nil)` -> `ErrReplayNoToken` ; erreur -> `ErrReplayNoToken` enveloppant la cause, journalisée en Warn avec
   `xuid` et `err`, message sans aucun conseil de recapture (contraire de `registry_catalog_drain.go:169`).
6. Les jetons ne servent qu'au client de ce film, puis sont oubliés (aucune trace dans la file, l'état, le journal).

Preuve qu'aucun jeton d'un autre joueur ne sert : (a) un seul appel de jeton, paramétré par `d.XUID` ; (b) aucune
boucle sur `cfg.LoadPlayers`, ni `haloTokensForDrain`/`jetonsCatalogue` (`registry_catalog_drain.go:151-175`), ni
`AnyPlayerTokens` (`registry_auth.go:336-350`), ni le pool (`publicReadTokenSrc`, `registry_auth.go:211-226`) :
interdits par une garde neuve (§10, archlint) sur les fichiers de la piste ; (c) un film archivé ne demande aucun
jeton ; (d) test du câblage : la fonction de jeton factice enregistre ses appels, un seul, avec le xuid de la demande,
y compris quand un autre profil a un jeton valide. Le RT n'est jamais recapturé ; une rotation est persistée par
`RefreshFromStoreEntry` (`cli_refresh.go:136-141`).

## 6. Routes et contrat

Montage dans `go/api/server_replay_generation.go` : `mountReplayGeneration(r, cfg, reg, participated, playerOpt)`,
appelé dans le groupe du rejeu (`server_apiv1.go:884-895`, après `MountReplayContext`), qui ne monte rien hors
`cfg.Capabilities.GenerateReplays`. Les routes héritent donc : titre actif, budget de lecture, auth des écritures
(`:819`), propriété (`:825`), `RequireCapability(CapReplay)` (503 `capability_unavailable`,
`middleware/require_capability.go:56`), participation (`r.With(participated)`, 404 `match_not_participant`).

- `POST /api/v1/players/{player_slug}/matches/{match_id}/replay/generate` -> corps
  `{"created": bool, "status": ReplayGenerationStatus}`. 202 : `queued` ou `running` (créée ou déjà là) ; 200 :
  `ready` (rien à faire).
- `GET /api/v1/players/{player_slug}/matches/{match_id}/replay/generation` -> `ReplayGenerationStatus`, 200.

Route de lecture NEUVE : (1) pour un match sans artefact, la page du rejeu n'a qu'un 404 (`replayLoadState.ts:28-34`) ;
(2) l'en-tête de la vue match se relit au mieux toutes les 60 s (`analysisPending.logic.ts:21`), trop lent, et y porter
la file lierait la vue match à un état absent de l'hébergé ; (3) lecture légère (mémoire, un `Stat`, deux SELECT
courts), relue toutes les 3 s seulement en `queued`/`running`.

Codes (messages fixes, `humacore.FixedMessage` ou `NewErrorFrom` : aucun littéral accentué dans `handlers`,
`archlint/no_french_label_literal_test.go:14`) : 400 `invalid_match_id` (`title.IsValidMatchID`) ;
401 `auth_required`, 403 `player_forbidden`, 404 `match_not_participant` (gardes) ; 409
`replay_film_unavailable`, `replay_generation_not_supported` (raison dans `details.reason`) ; 429
`replay_generation_queue_full` (`retryable: true`) ; 503 `capability_not_supported` par `MapCapabilityError`
(`handlers/capability_error.go:25-33`, garde anti-copie), 503 `replay_generation_unavailable` (file non liée).
Route absente (404) hors placement local, comme `desktop_replay_regen_mount_test.go:13-44`.

OpenAPI : `openapi-gen` décrit le routeur du profil démo (`cmd/openapi-gen/main.go`, `web/lib/api/response-types.guard.test.ts:71-76`),
qui ne monte pas ces routes. G.3 n'ajoute donc aucune route à `openapi.yaml` ; `make generate-types` ne change que
`CapabilityMap` (champ `can_generate_replays`). Les deux types de réponse s'écrivent à la main côté web et entrent
dans `MANUAL_CAPABILITY_TYPES` (`response-types.guard.test.ts:77-83`), comme `ReplayRegenStatus`. Opérations Huma :
`postPlayerMatchReplayGenerate`, `getPlayerMatchReplayGeneration`, étiquette `replay`.

## 7. La capacité `can_generate_replays`

1. `go/config/deployment.go` : champ `GenerateReplays` de `DeploymentCapabilities` (commentaire : génération à la
   demande d'un rejeu absent, placement local, film téléchargé avec le jeton du joueur) ; dans `capabilitiesOf`
   (`:275-319`) `GenerateReplays: desktop || cfg.Deployment == DeploymentDev`. La comparaison vit dans le résolveur,
   seul endroit où elle est permise (ADR 0044 D-4 ; précédent `ResourcePackFatal`, `:306`). `ReplayRegen` (`:312`)
   n'est pas touché.
2. `go/domain/bootstrap.go` : `CanGenerateReplays bool `json:"can_generate_replays" doc:"..."`` après
   `CanRegenerateReplays` (`:60`).
3. `go/service/bootstrap_service.go` : `buildCapabilities` (`:410-460`) reçoit un booléen de plus,
   `CanGenerateReplays: dc.GenerateReplays && titleHasReplay`, calculé dans `Build` (`:161`) par
   `s.replayArtifactTitle(currentTitleSlug)`, fonction injectée par `WithReplayArtifactTitleCheck(fn func(string) bool)`
   (nil = faux). Câblage dans `cmd/server/boot_providers.go:159-193` : fonction mémoïsée par titre sur
   `games.LoadCapabilityMap(cfg.Roots().Resources, slug)` et `caps.Has(games.CapFilmReplayArtifact)` (recette
   `registry_replay_regen.go:67-83` ; `LoadCapabilityMap` ne mémorise pas, `games/capabilities.go:101-103`).
Côté web : `can_generate_replays: false` dans `DEPLOYMENT_CAPABILITY_DEFAULTS` (`deploymentCapabilities.ts:43`).
Aucun consommateur ne lit `deployment`.

## 8. Le web

- Code partagé par deux features (rejeu, vue match), donc sous `web/features/_shared/replayGeneration/` (la
  frontière entre features l'impose : `node tools/lint-cross-feature-imports.mjs`).
- **Clés** (`web/lib/query/keys.ts`) : `matchReplayGeneration(playerSlug, titleSlug, matchId)` (codes seuls, pas de
  langue) et le préfixe `matchViewAllLocales(playerSlug, titleSlug, matchId)` (la clé `matchView`, `:164-165`, porte
  la langue).
- **Hook** `queries.ts` : `useReplayGenerationStatus(playerSlug, matchId, enabled)` (relecture 3 s si `queued` ou
  `running`, sinon aucune, patron `components/replay-regen/queries.ts:37-52`) et `useRequestReplayGeneration()`
  (POST ; succès : pose l'état au cache ; 409 ou 429 : relit l'état). Au passage à `ready` : invalidation de
  `matchReplay`, `matchViewAllLocales`, `homeAll`. Types `ReplayGenerationStatus` et `ReplayGenerationRequestResponse`
  écrits à la main dans ce fichier.
- **Logique pure** `logic.ts` : `generationView(status)` -> `{ action: 'generate' | 'retry' | 'open' | null,
  pending: 'queued' | 'downloading' | 'building' | null, messageKey }`. Règle « aucun bouton quand l'action est
  impossible » : `action` n'est non nul que pour `generatable`, `failed` réessayable, `ready`.
- **Composant** `ReplayGenerateAction.tsx`, deux variantes. `panel` : remplace le bloc vide de `ReplayPage`
  (`:245-249`) quand `useDeploymentCapability('can_generate_replays')` est vrai ; sinon le bloc actuel reste. Le bloc
  garde sa bordure ; `pending` affiche `AnalysisPendingSlot` (`web/components/ui/analysis-pending-slot.tsx:11-24`,
  pointillés et indicateur tournant, l'indicateur « analyse en cours » de la vue match) ; les états terminaux sans
  action affichent leur phrase sans bouton. `inline` : dans `ReplayLink` (`MatchHeader.replayLink.tsx`), quand
  `available` est faux et la capacité vraie : bouton contour « Générer le rejeu » à côté du bouton de copie
  (`MatchHeader.card.tsx:288-293`), et pendant l'attente une pastille compacte qui réutilise le même indicateur :
  le cercle tournant de `AnalysisPendingSlot` est extrait en `AnalysisPendingSpinner` exporté par le même fichier
  (une copie, pas deux). La requête de statut ne part que sous la capacité et quand `replay_available` est faux.
- **Textes** `text.ts`, `Record<Locale, ...>` (fr, en) : générer, réessayer, ouvrir le rejeu ; « Aucun rejeu pour ce
  match » ; « Le film du match sera téléchargé puis analysé sur ce poste » ; en attente (position {n}) ;
  téléchargement du film ; génération en cours ; attente de la fin de la partie ; file en pause ; échec ; film plus
  disponible ; mode sans rejeu ; aucun compte Xbox relié à ce profil ; file pleine. Mots : « générer », jamais
  « cuire ». Couleurs : jetons sémantiques (`info`, `muted`, `border`) seulement.
- Commentaire faux à corriger : `MatchHeader.replayLink.tsx:23-29` (« En production aucun artefact n'est produit ») :
  le composant rend le lien quand l'artefact existe, sinon l'action de génération sous la capacité, sinon rien.

## 9. Liste fermée des fichiers

**G.1 (Go)**, à créer :
1. `go/domain/replay_generation.go` : statut, demande, constantes d'états et de raisons, erreurs (§2).
2. `go/port/replay_generation.go` : `ReplayGeneration`, `ReplayGenerationQueue`, `ReplayMatchProbe`, `ReplayTokenPresence`.
3. `go/service/replay_generation_service.go` (+ `_test.go`) : contrôles et ordre d'évaluation (§2), journal de chaque refus.
4. `go/replayregen/deps.go` : `Deps` déplacé de `service.go`, plus `FetchFilm`, `ReleaseFilm`, `Demands`, `MaxDemands`.
5. `go/replayregen/demand.go` (`Enqueue`, `DemandStatus`, plafond, idempotence, `results`), `demand_run.go`
   (`demandLoop`, `serveDemands`, `processDemand`, classement, `ResumeDemands`), `demand_store.go` (fichier
   atomique), et leurs tests `demand_test.go`, `demand_store_test.go`.
6. `go/domain/title/paths_replay_generation.go` (+ test) : `ReplayGenerationQueuePath`.
7. `go/sync/killcollector/film_absent_mark.go` (+ `_integration_test.go`) : `MarquerFilmAbsent`.
8. `go/api/wire/registry_replay_generation.go` : fabrique `ReplayGeneration`, `ProbeReplayMatch` (registre :
    `duckdb.NewReplayRegistryRepo(shared, nil).MatchesByIDOrPrefix` sous `titleBases().ReadShared` comme `:49-54` de
    `registry_replay_regen.go` ; bit : `duckdb.NewAnalysisProbeRepo(pdb).LoadAnalysisProbes` ; artefact :
    `Stat(paths.ReplayArtifactPath)` ; film : `filmcache.FilmComplet`), présence du jeton.
9. `go/api/wire/registry_replay_generation_film.go` (+ `_test.go`) : `fetchFilmForDemand`, `releaseFilmAfterBuild` (§4, §5).
10. `go/api/handlers/replay_generation.go` (+ `_test.go`) : deux routes, traduction des erreurs.
11. `go/api/server_replay_generation.go` (+ `replay_generation_mount_test.go`) : montage sous la capacité.
12. `go/archlint/replay_generation_token_source_test.go` : garde de §5 (b), avec sa preuve de détection.

À modifier :
13. `go/config/deployment.go`, `go/config/deployment_test.go` (§7).
14. `go/domain/bootstrap.go` ; `go/service/bootstrap_service.go` ; `go/service/bootstrap_capabilities_profile_test.go`
    (cas `TestBuildCapabilities_*` : vrai en desktop et dev avec titre, faux sans titre, faux hébergé et démo).
15. `cmd/server/boot_providers.go` (vérification du titre), `cmd/server/boot_replayregen.go` (§3 Liaison).
16. `go/replayregen/service.go` (champs, appel `serveDemands` dans `run`, relance dans `finish`), `go/replayregen/scan.go` (doc).
17. `go/api/wire/registry_replay_regen.go` (`ReplayRegenDeps` : `FetchFilm`, `ReleaseFilm`, `Demands`).
18. `go/api/server_apiv1.go` : un appel de `mountReplayGeneration` dans le groupe `:884-895`.
19. `go/archlint/no_shared_writer_on_request_path_test.go` : entrée datée `"internal/api/wire/registry_replay_generation_film.go": {1, "2026-11-17 : file de génération à la demande (capacité GenerateReplays, bureau et poste de développement) : bit film absent posé depuis la goroutine de la file, jamais sous la requête, un film à la fois, attente bornée à 8 s"}` (`:52-78`).

**G.2 (web)**, à créer : `web/features/_shared/replayGeneration/{queries.ts, logic.ts, text.ts,
ReplayGenerateAction.tsx, logic.test.ts, queries.test.ts, ReplayGenerateAction.test.tsx}`.
À modifier : `web/lib/query/keys.ts` ; `web/lib/capabilities/deploymentCapabilities.ts` (+ son test) ;
`web/features/match-replay/ReplayPage.tsx` (bloc vide) ; `web/features/match-view/MatchHeader.replayLink.tsx`
(action, commentaire) ; `web/components/ui/analysis-pending-slot.tsx` (extraction du cercle) ;
`web/lib/api/response-types.guard.test.ts` (deux entrées `MANUAL_CAPABILITY_TYPES`).
Hors G.1/G.2 (pilote, G.3) : `openapi.yaml`, `generated.ts`, notes de version, `docs/CHANGELOG.md`.

## 10. Tests

**Domaine, config, bootstrap** : profils `desktop`, `dev` -> `GenerateReplays` vrai ; `hosted`, `demo` -> faux ;
`ReplayRegen` inchangé (témoin T3 de l'ADR 0044 : seule différence, le champ neuf) ; `can_generate_replays` faux si le
titre courant n'a pas `film.replay_artifact` (Halo 5 : `not_exposed`) ou si la fonction n'est pas câblée.
**PathResolver** : chemin sous `AdminStateDir`, dérivé de la racine de données.
**replayregen** (faux `Rebuild`, `FetchFilm`, magasin en mémoire, portes réglables) :
(1) doublon -> `created=false`, une entrée, un `Rebuild` ; (2) onzième demande -> `ErrReplayGenerationQueueFull`,
rien d'écrit ; (3) un film à la fois : concurrence de `Rebuild`/`FetchFilm` jamais > 1, lot et demandes mêlés ;
(4) ordre : lot de trois candidats, demande arrivée pendant le premier -> passe en deuxième ; (5) mode jeu faux ->
rien ne part, `waiting_for_game` ; synchronisation en cours -> attente ; (6) lot en pause -> `queued` avec `paused`,
reprise -> part ; (7) l'ajout écrit le fichier, la sortie le réécrit, `ResumeDemands` reprend dans l'ordre, fichier
illisible -> Warn et file vide ; (8) contexte annulé pendant la reconstruction -> la demande reste au fichier ;
(9) `ErrReplayFilmGone` -> `film_unavailable`, `ErrFilmNonFinalise` -> `failed/film_not_ready` réessayable,
`ErrMapNotInCatalog` -> non réessayable, `ErrBusy` -> nouvel essai ; (10) échec puis nouvelle demande -> acceptée,
artefact à jour au tour -> `ready` sans `Rebuild` ; (11) `results` bornés à 50.
**service** (faux ports) : capacité absente -> `ErrCapabilityNotSupported` sur GET et POST ; mode non géré, match
inconnu -> `not_supported` et POST refusé ; bit posé -> `film_unavailable` et POST refusé ; artefact présent ->
`ready`, rien en file ; pas de jeton et film non archivé -> `no_token` ; pas de jeton et film archivé ->
`generatable` ; file pleine propagée ; chaque refus produit une ligne `slog` (gestionnaire de test qui capture).
**handlers** (`httptest`, faux port) : 202 créée, 202 doublon, 200 `ready`, 400 identifiant, 409 film indisponible,
409 non pris en charge (`details.reason`), 429, 503 capacité, 503 non liée, GET 200 pour chaque état.
**montage** (`api`) : sans `GenerateReplays` -> 404 ; avec -> route présente ; propriété appliquée (mode xbox) :
non-propriétaire et co-membre -> 403 au POST, administrateur -> passe ; non-participant -> 404 (patron
`ambiguous_path_guards_test.go:347-368`).
**câblage du film** (faux résolveur de jeton, faux client, archive en dossier temporaire) : film archivé -> aucun
appel de jeton ; film absent -> un seul appel de jeton, avec le xuid de la demande, alors qu'un autre profil a un
jeton ; `(nil, nil)` -> `ErrReplayNoToken` ; 404 -> `MarquerFilmAbsent` appelé, `ErrReplayFilmGone` ; non finalisé ->
aucun marqueur, rien écrit ; archive désactivée -> film retiré après succès seulement.
**killcollector** (tag `integration`) : `MarquerFilmAbsent` pose le bit, idempotent, ne touche aucun autre bit.
**archlint** : la garde neuve refuse `LoadPlayers`, `haloTokensForDrain`, `jetonsCatalogue`, `AnyPlayerTokens`,
`publicReadTokenSrc`, `ResolveAny` dans les fichiers `registry_replay_generation*.go`, avec un extrait fautif qui la
fait rougir ; `TestNoSharedWriterOnRequestPath` vert avec l'entrée datée.
**web** : `logic.test.ts` (une ligne par état et raison, `action` nul quand impossible) ; `queries.test.ts`
(relecture seulement en `queued`/`running`, invalidations au passage à `ready`, 409 relit l'état) ;
`ReplayGenerateAction.test.tsx` (rendu par état, deux variantes, rien sans capacité) ; test du bloc vide de
`ReplayPage` avec et sans capacité ; défauts des capacités ; gardes `keys`, `featureTsxNoQuery`, `response-types`.

## 11. Risques et parades

| Risque | Parade |
|---|---|
| Jeton d'un autre joueur | §5 : un seul appel paramétré, garde archlint, test d'appel unique |
| Deux rafraîchissements concurrents du même RT dans le processus (pool et cache font déjà chacun le leur ; un RT consommé rend `invalid_grant`, `pool/resolver.go:60-65`) ; `RefreshHaloTokensViaStoreFirst` marque alors `reauth_required` (`cli_refresh.go:84-90`), à tort | cache du processus lu d'abord, rempli après (§5 étape 4) : au plus un échange par heure et par joueur ; question Q1 |
| Recapture conseillée à l'utilisateur | aucun message ne la propose ; texte web neutre |
| File bloquée par une partie, un cycle de synchronisation ou une pause | mêmes portes que le lot, signalées dans l'état (`waiting_for_game`, `paused`) |
| Deux décodages en parallèle | `slot` en processus ; `film_decode.lock` entre processus (`filmrun/solo.go`), `ErrBusy` puis nouvel essai borné |
| Prise de l'écrivain partagé sous une requête | prise seulement depuis la goroutine de la file, bornée à 8 s, une fois par film expiré ; entrée datée de la garde |
| Demande perdue à l'arrêt | fichier atomique réécrit à chaque changement ; reprise à l'ouverture |
| Disque, film à moitié écrit | plafond de 10 ; manifeste écrit en dernier, film partiel réparé au prochain essai ; échec d'écriture `fetch_failed` journalisé ; réglage « archiver les films » respecté |
| Hébergé exposé | capacité fausse : route non montée (404), bootstrap faux ; l'action d'administration n'est pas touchée |
| Bit « film absent » faux sur une panne transitoire | seul 404/410 ou manifeste vide le pose (`IsFilmGoneErr`) ; 503, limite de débit, réseau : `fetch_failed` |

## 12. Vérification du pilote (préparée, NON lancée ; avec l'accord de l'utilisateur, sur UN match)

But : le manifeste du film d'un match se résout avec le jeton du joueur de la page, sans télécharger un morceau ni
écrire quoi que ce soit, hors la rotation du RT que `RefreshHaloTokensViaStoreFirst` persiste obligatoirement
(ADR 0023 : ne pas la persister brûlerait le RT).
Préalables : serveur de développement (air) ET application de bureau ARRÊTÉS (un second processus qui fait tourner le
RT désynchroniserait celui que le serveur garde en mémoire) ; un match du poste, de préférence sans film à l'archive ;
le slug du profil qui l'a joué.

Programme jetable (non versionné, supprimé après), tag `dev` comme `cmd/get-token/main.go:1` : `cmd/sonde-manifeste-g0/main.go` :

```go
//go:build dev

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/titleseams"
	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"
)

func main() {
	titleseams.RegisterAll("")
	slug, match := flag.String("player", "", "slug du profil"), flag.String("match", "", "match_id complet")
	flag.Parse()
	cfg, err := config.Load() // .env.local chargé ici ; profil dev, magasin en clair
	if err != nil || *slug == "" || !titlePkg.IsValidMatchID(*match) {
		fmt.Fprintln(os.Stderr, "usage : -player <slug> -match <match_id complet>", err)
		os.Exit(2)
	}
	players, err := cfg.LoadPlayers(titlePkg.DefaultSlug) // db_profiles.json, aucune base ouverte
	var xuid, gt string
	for _, p := range players {
		if p.PlayerSlug == *slug {
			xuid, gt = p.XUID, p.Gamertag
		}
	}
	if err != nil || xuid == "" {
		fmt.Fprintln(os.Stderr, "profil introuvable ou sans xuid", err)
		os.Exit(2)
	}
	ctx, store := context.Background(), auth.NewMultiUserTokenStore(cfg.PathResolver().WatcherTokensDir())
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), xuid, gt)
	if tokens := auth.HaloTokensFromExchange(res); err != nil || tokens == nil {
		fmt.Println("JETON : indisponible :", err) // cause seule, aucun conseil de recapture
		os.Exit(1)
	} else {
		client := gosync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 1) // sans cache disque
		refs, found, err := client.GetFilmChunkURLs(ctx, *match)                      // manifeste seul
		types := map[int]int{}
		for _, r := range refs {
			types[r.ChunkType]++
		}
		fmt.Printf("MANIFESTE : err=%v trouvé=%v morceaux=%d par type %v\n", err, found, len(refs), types)
	}
}
```

Commande : `cd apps/go-api && GOWORK=off go run -tags dev ./cmd/sonde-manifeste-g0 -player <slug> -match <match_id>`.
Attendu : `err=<nil> trouvé=true` avec au moins un morceau de type 1, 2 et 3 (`trouvé=false` : film expiré,
404/410). Le programme n'imprime ni jeton, ni
URL pré-signée, ni xuid. `GetFilmChunkURLs` ne lit que le manifeste (`filmfetch/films.go:142-162`) ; le client sans
cache disque ne lit pas l'archive locale (`haloadapter/client_films.go:29-33`, `filmfetch/local_film_cache.go:98-101`).
Retirer le dossier ensuite (`rm -r apps/go-api/cmd/sonde-manifeste-g0`), rien à valider.

## 13. Questions ouvertes (ce que le code ne tranche pas)

1. **Rafraîchissement concurrent du RT** (§11, ligne 2). Le cache lu d'abord réduit le risque sans l'annuler.
   L'équivalent serveur `tokencache.ResolveFreshPlayerTokens(ctx, xuid)` (`player_token_cache.go:163-201` :
   même magasin, même joueur, échange dédupliqué, aucun marquage `reauth_required` sur un échec,
   `registry_auth.go:303-331`) l'annulerait dans le processus. Recommandation : garder la décision
   (`RefreshHaloTokensViaStoreFirst` derrière le cache), et demander à l'utilisateur s'il accepte de basculer vers
   `ResolveFreshPlayerTokens` si G.4 ou la relecture adversariale montre un marquage à tort.
2. **Profil suivi sans compte relié** (cas du poste de développement) : prêter le jeton du pool consenti ?
   Recommandation : non ; état `not_supported/no_token`, conforme à D-4.
3. **Plafond** : 10 en vol, 50 issues en mémoire. Recommandation : garder, en constantes nommées.
4. **Archive désactivée** : retirer un film téléchargé exprès ? Recommandation : oui, même règle que la cuisson.
5. **Échec après redémarrage** : non persisté, le match redevient `generatable`. Recommandation : garder.
6. **Bouton de l'en-tête** : contour avec libellé (recommandé) ou icône seule ; contrôle à l'écran en G.4.
