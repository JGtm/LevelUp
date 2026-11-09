# Registre de revue — chantier annuaire des joueurs (ADR 0035)

> Pilote, 2026-09-15 soir. Périmètre : commits `a7d6838e9`, `55d8b0b0b`, `b9010c56f`,
> `5d16e16f9` (étapes 1-4), relus fichier par fichier pendant l'exécution de l'étape 5-6.
> Chaque constat porte un statut à la clôture (étape 7) : corrigé / accepté / reporté.

## Constats

| # | Sévérité | Fichier | Constat | Statut |
|---|---|---|---|---|
| R1 | **haute** | `service/playerdirectory/collect.go` `addAccounts` | Deux comptes portant le même xuid (cas RÉEL en prod : `JGtm` et `jgtm_xbox`, xuid `2533274823110022`) : le second écrase silencieusement `rec.Account`, aucune anomalie, aucun test. L'annuaire ment sur le compte affiché. Correction attendue : garder le premier, ajouter l'anomalie `account_duplicate` (warning, détail = username écrasé) OU porter une liste ; test avec deux comptes même xuid. | à traiter (7.1) |
| R2 | basse | `web/features/setup/setupRouting.ts` `needsOwnProfile` | Un utilisateur non admin sans profil propre mais co-membre d'un groupe (`available_players` > 0 par accès aux profils d'amis) n'est pas conduit au wizard : il consulte ses amis, sans sync pour lui. Cohérent avec ADR 0029 (accès par groupe) ; à documenter dans le commentaire de la fonction, pas de changement de comportement. | à statuer (7.1) |
| R3 | basse | `service/playerdirectory/collect.go` `resolve` | Fusion par gamertag (insensible à la casse) quand une source n'a pas de xuid (profil legacy sans `xuid`, dossier orphelin). Deux personnes distinctes ne fusionnent que si l'une n'a pas de xuid nulle part — acceptable, mais le commentaire doit dire que le gamertag n'est un repli que pour les sources SANS xuid (ADR D1). | à statuer (7.1) |
| R4 | basse | `service/bootstrap_service.go` | Alignement des champs du littéral `Capabilities` : `gofmt -l` vide, donc conforme ; rien à faire. | clos |
| R5 | info | `api/server_player_directory.go` | `daemon.(playerdirectory.WatchedReader)` sur une interface portant un `*watcher.Daemon` nil typé : `WatchedPlayers` est nil-safe (`d == nil`), OK. | clos |
| R6 | **moyenne** (règle 3) | `service/profile_service.go` `PurgeIdentityData` | `removed[slug] = os.RemoveAll(...) == nil` : l'erreur système (verrou Windows, EBUSY, droits) est réduite à un booléen sans journal ; la purge remonte « entree retiree mais dossier joueur non supprime » sans la cause. Correction : `slog.Warn`/`ErrorContext` avec `err`, `path`, `title_slug` avant de poser `false`. Test : dossier non supprimable → journal présent (ou au minimum l'erreur portée dans le rapport). | à traiter (7.1) |

Étapes 5-6 relues (commits `53d19824c`, `ffd51c600`) : `Onboard` (profil AVANT watcher, échec de notification journalisé et non propagé, `DBCreated` depuis le FS), ratchet `CreatePlayer(` à allowlist unique, `Purge` (refus admin, dry-run sans effet, `errors.Join`, étapes indépendantes), `RemovePlayer` sous lock avec cancel du poller, CLI `--yes` parsé après l'argument positionnel, aucune valeur de token en sortie (booléen `HasRefreshToken` seulement).

## Revue adversariale — ronde 1 (relecteur à contexte frais, lentilles accès / anti-patterns / multi-titre, 2026-09-16)

Contrat, périmètre et décisions transmis selon le skill `adversarial-review`. 5 constats recevables,
0 jeté ; 8 conditions sur 10 tiennent, la 9e (deux comptes même xuid) était déjà corrigée dans
l'arbre (R1), la 10e (settings illisibles ⇒ non verrouillé) est le repli assumé par l'ADR.

| # | Gravité | Fichier | Constat | Correction | Statut |
|---|---|---|---|---|---|
| A1 | **P1** | `handlers/title_sync.go` SetSync/Purge | Mettre un titre en pause ou le purger laissait le `PlayerWatcher` et son poller vivants : chaque match détecté était refusé par la porte, incrémentait `sync_refused_no_profile` (le signal d'intrusion) et l'annuaire affichait `watched_without_profile` en warning — une action admin légitime déclenchait l'alarme. | `watcher.Daemon.RemovePlayerTitle` (nouveau, par couple xuid×titre), `TitleSyncHandler.WithWatcher/WithPlayerLookup` : pause et purge retirent le couple (xuid résolu AVANT la purge), réactivation le remet via `AddPlayer`. Tests `daemon_remove_title_test.go` (2), `title_sync_watcher_test.go` (4). | corrigé |
| A2 | P1 (règle 7) | `port/player_directory.go`, `playerdirectory/directory.go` | `PlayerDirectory.HasTrackedProfile` sans aucun appelant de production (les portes lisent `config.AppConfig.HasTrackedProfile`) — code mort avec test vert. | Méthode retirée du port, de l'implémentation et du faux ; doc du port et de `ProfilesReader` : la définition de « suivi » est `config.AppConfig.HasTrackedProfile`, l'annuaire la réutilise sans la ré-exposer. ADR D2 amendée. | corrigé |
| A3 | P1 | `web/…/useInstanceLock.ts` + `handlers/settings.go` | Verrou forcé par `LEVELUP_INSTANCE_LOCKED` : décocher l'interrupteur écrivait `false` sur disque, `/bootstrap` rendait toujours `true`, la case se recochait seule sans message. | Backend : `PATCH /settings {instance_locked:false}` sous verrou env ⇒ 409 `instance_lock_forced`, rien d'écrit (test `settings_lock_forced_test.go`). Web : `useInstanceLock.errorCode`, message spécifique `admin.identities.lock_forced` FR/EN (tests hook + section). | corrigé |
| A4 | P1 | `playerdirectory/directory.go` `sortRecords` | Deux comptes sans gamertag ni xuid : ordre de `userstore.List` (map) ⇒ le tableau permutait à chaque rafraîchissement, contre l'invariant documenté « ordre total et stable ». | Dernier critère de tri = nom du compte principal. Test `TestList_OrdreStable_ComptesSansIdentiteXbox`. | corrigé |
| A5 | P1 | `cmd/levelup/cmd_identity.go`, `playerdirectory/purge.go` | Un dossier orphelin sans aucun registre (xuid vide) était signalé par `identity list` mais impossible à retirer par `identity purge` (xuid exigé). | `Directory.Get` accepte le gamertag d'une identité SANS xuid (jamais celui d'une identité avec xuid) ; la purge ne traite alors que le dossier ; CLI `purge <xuid|gamertag>`. Tests `TestGet_ParGamertagSeulementSansXuid`, `TestPurge_DossierOrphelinSansXuid_ParGamertag`. | corrigé |
| A6 | P1 (règle 6) | `cmd/server/main.go`, `api/server.go`, `cmd/levelup/cmd_identity.go` | 3e copie de `filepath.Join(cfg.AuthDir, "users.json")`. | `config.AppConfig.UsersFilePath()` + `config.UsersFilePathIn(dir)` (4e copie trouvée par le ratchet dans `cmd/admin`) ; ratchet `no_users_json_literal_test.go`. | corrigé |
| A7 | P2 (doc) | `settings/defaults.go` | Commentaire promettant le seuil ≤ 500 L alors que `store.go` reste à 516. | Reformulé : dette réduite, pas résorbée. | corrigé |
| A8 | P2 (hors diff) | `handlers/settings.go:232` | Le toggle du verrou fait confiance à `sess.Role` là où `setup.go` résout le rôle par le store. Préexistant. | Consigné §10 du plan, non traité. | consigné |

## Revue adversariale — ronde 2 (contexte frais, lentilles tests / front + re-vérification des 8 corrections, 2026-09-16)

Résultat : **0 P0/P1** (5 P1 en ronde 1 → 0 : la boucle converge, borne du skill respectée, pas de
ronde 3). 2 P2 recevables et 3 réserves, tous sur du code de ce lot — traités ; 25 conditions
vérifiées qui tiennent (portes, boot, sync HTTP intact, sentinelles, sha256 de la base
partagée, 4 ratchets non vides, front : tokens, i18n FR/EN, query key, anti-boucle).

| # | Gravité | Constat | Correction | Statut |
|---|---|---|---|---|
| B1 | P2 | `collect.go addAccounts` : `userstore.List` itère une map → « le premier compte lu » d'un xuid n'était pas stable, le compte principal affiché alternait (annuaire, CLI). Le faux de test (slice ordonnée) masquait le cas. | Tri total avant la boucle : `created_at` puis nom — le plus ancien compte est le principal. Test `TestList_DeuxComptesMemeXuid_PrincipalDeterministe` (deux ordres d'arrivée, même résultat). | corrigé |
| B2 | P2 | `settings` : un fichier existant porte déjà `instance_locked:false` / `can_self_provision:true` (Save réécrit la struct) → les défauts appliqués n'y changent rien. | Comportement VOULU par l'ADR D5 (une instance existante est verrouillée par son admin, pas par un déploiement) ; figé par `TestLoad_EnforcedDefaults_PermissiveKeysAlreadyPresent` pour qu'on ne le prenne pas pour un oubli. | accepté + testé |
| B3 | réserve (a) | Câblage `daemon.(handlers.TitleWatcher)` par assertion de type non gelée à la compilation. | `internal/api/wiring_assertions_test.go` : `*watcher.Daemon` doit implémenter `handlers.TitleWatcher` et `playerdirectory.WatchedReader` (erreur de build sinon). | corrigé |
| B4 | réserve (b) | `mockDirectory.HasTrackedProfile` / `fakeDirectory.HasTrackedProfile` encore déclarés dans les tests handlers (méthode retirée du port). | Supprimés (règle 7). | corrigé |
| B5 | réserve (h) | Branche « `os.RemoveAll` échoue » de `PurgeIdentityData` non testée (R6 : le journal existait, rien ne l'exerçait). | Seam `ProfileService.WithRemoveAll` ; `profile_service_purge_identity_test.go` : refus injecté → `removed=false`, profil retiré quand même, journal avec la cause ; chemin nominal → dossier supprimé. | corrigé |

## Corrections apportées (2026-09-16, pilote, hors commit des agents)

- **R1 corrigé** : `IdentityRecord.DuplicateAccounts` + anomalie `account_duplicate` (warning,
  détail = username du doublon) ; le premier compte lu reste `Account`, rien n'est écrasé
  (`collect.go` journalise le doublon) ; la purge retire TOUS les comptes du xuid et refuse si
  l'un d'eux est admin (`purge.go`). Tests : `TestList_DeuxComptesMemeXuid`,
  `TestPurge_DeuxComptesMemeXuid_LesDeuxSupprimes`, `TestPurge_RefuseSiUnDoublonEstAdministrateur`.
  Web : mapping + clé i18n FR/EN `admin.identities.anomaly_account_duplicate` + test. OpenAPI
  et `generated.ts` régénérés (`openapi-check` OK).
- **R2 statué** : accepté, comportement conforme à l'ADR 0029 ; pas de changement.
- **R3 corrigé** : commentaire de `identityBuilder.resolve` (repli gamertag réservé aux sources
  sans xuid).
- **R6 corrigé** : `PurgeIdentityData` journalise l'erreur de `os.RemoveAll` (`slog.Warn`, `err`,
  `path`, `title_slug`, `gamertag`) avant de poser `false`.

## Points vérifiés sans constat

- `authz.InstanceLocked` : env court-circuite, `load` nil → false, erreur journalisée avant repli.
- `setup.go` : admin exempté des deux gardes ; user → `provisioning_disabled` avant `instance_locked` (ordre historique conservé).
- `settings/defaults.go` : clés absentes + mode appliqué → `instance_locked=true`, `can_self_provision=false` ; valeur explicite gagne ; fichier jamais réécrit.
- Portes : `Coordinator.Submit` (avant le claim in-flight, compteur `sync_refused_no_profile`), `Daemon.AddPlayer` (`ErrPlayerNotTracked`), SSO (`watcherAllowedFor`, titre par défaut) ; `HasTrackedProfile` par xuid via `SyncablePlayers`, titre vide normalisé.
- `admin_identities.go` : handler sans logique, 503 si annuaire non câblé, 500 journalisé.
- `useInstanceLock` : mutation via `useUpdateSettings`, invalidation de `bootstrap` au succès, état grisé pendant l'envoi.
- Aucun accès DuckDB dans `playerdirectory` (imports vérifiés).
- `gofmt -l ./internal ./cmd` → vide.
