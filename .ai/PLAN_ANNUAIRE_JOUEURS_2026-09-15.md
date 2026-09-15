# PLAN — Annuaire des joueurs : une clé d'identité, un chemin d'onboarding, un modèle de lecture

> Date : 2026-09-15. Branche : `wt/player-directory` (worktree `LevelUp-wt-player-directory`,
> base `feat/v75`). ADR : `docs/adr/0035-player-directory-single-identity-key.md` (lue AVANT
> toute étape). Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la
> fois, gate passé avant l'étape suivante, aucun report d'item exécutable, statuts `[x]` /
> `[~]` réf / `[!]` justifié, zéro fix hors périmètre — les découvertes vont en §10).
>
> Plan frère (non exécuté, autre session) : `.ai/PLAN_AMIS_PAR_JOUEUR_ET_INVITATIONS_2026-09-15.md`.
> Il touche `setup.go:119-126` et `xbox_auth_service.go` (invitations, `ProvisionGrant`). Ce
> plan-ci ne traite PAS les invitations ; il pose le helper de verrou et le chemin `Onboard`
> que l'étape 5 du plan frère devra utiliser (noté dans son §10 à la clôture).

## 1. Objectif et critères de succès

**Constat (vérifié sur pièces, journaux du VPS du 23/07/2026)** : un compte Xbox inconnu s'est
connecté par SSO (instance non verrouillée), a été ajouté au watcher, et un sync a écrit une
player DB + 25 matchs dans la base partagée sans qu'aucun profil n'existe ; post-sync, Prestige,
ownership, scheduler et pages admin ne l'ont jamais vu. Détail : ADR 0035 §Context.

**Décisions produit tranchées (utilisateur, 2026-09-15)** :
- **P1** — SSO hors verrou : le compte et les tokens sont créés, mais **aucun watcher ni sync
  tant que le profil n'existe pas** ; pas d'auto-provisioning implicite du profil.
- **P2** — La purge d'une identité **ne supprime jamais** les matchs de la base partagée.
- **P3** — Tests à chaque couche, revue adversariale par le pilote à la clôture (pas par lot).
- **P4** — Périmètre fermé : les 7 étapes ci-dessous, rien d'autre. P0 (verrouiller la prod
  maintenant) = action admin de l'utilisateur, hors code.

**Critères de succès** :
1. Un compte SSO sans profil ne déclenche ni poller, ni sync, ni écriture disque ; le refus est
   journalisé et compté.
2. Le verrou d'instance est lu par un seul helper ; ratchet vert.
3. `GET /admin/identities` et la section « Identités » de `/admin/management` montrent, par
   xuid, compte / profils / token / suivi live, avec anomalies typées.
4. `POST /setup/players` passe par `PlayerDirectory.Onboard` ; ratchet sur `CreatePlayer(`.
5. `levelup identity purge <xuid>` retire compte + token + profils + dossiers + groupes, jamais
   la base partagée (test à l'octet près).
6. Gates : `go test ./...` vert (dont `-tags=integration ./internal/sync/...`), `make check-types`,
   `make test-web`, `make go-api-lint` sans nouvelle dette.

**Effort** : étape 1 rapide, 2 moyen, 3 moyen, 4 moyen, 5 moyen (risque auth), 6 moyen, 7 rapide.

## 2. Étape 0 — Préparation (rapide) — pilote

- [x] 0.1 Worktree `LevelUp-wt-player-directory` sur `wt/player-directory` depuis `feat/v75`.
- [x] 0.2 ADR 0035 écrite.
- [x] 0.3 Jonction `apps/web/node_modules` → `LevelUp-go-migration/apps/web/node_modules`
      (`cmd /c mklink /J`), JAMAIS `npm ci`. Faite le 2026-09-15 21:36 ; `.bin/tsc`,
      `.bin/vite` et `.bin/vitest` visibles depuis le worktree.
- [x] 0.4 Baseline (2026-09-15, 21:33 → 22:04) : tous les paquets verts —
      `internal/service` 0, `internal/watcher` 3,3 s, `internal/api/handlers` 71,8 s,
      `internal/platform/settings` 3,0 s, `internal/authz` 2,0 s, sous-paquets
      `internal/sync/*` verts. **`internal/sync` (paquet racine) : 601 s au premier
      passage = dépassement du timeout PAR DÉFAUT de `go test` (600 s), build CGO
      DuckDB à froid inclus ; relancé seul avec `-timeout 30m` → `ok 501,057 s`.**
      Baseline réelle : 0 échec. Conséquence pour les gates suivants : toute commande
      touchant `./internal/sync/...` porte `-timeout 30m` (cf. §10).
      `make check-types` → 0 (54 s).

**Gate G0** : 0.3 et 0.4 verts. ✅

## 3. Étape 1 — Verrou centralisé + défauts sûrs (rapide) — agent A

- [x] 1.1 `internal/authz/authz.go:101-133` : `func InstanceLocked(envLocked bool, load func() (bool, error)) bool`.
      `envLocked` court-circuite (load non appelé) ; `load` nil ⇒ seule la source env ;
      `err != nil` ⇒ `slog.Warn("instance_locked: settings illisibles, repli sur non verrouillé", "err", err)`
      puis `false` (journal repris mot pour mot de l'ancien `server_apiv1.go`).
      Tests `authz_test.go:TestInstanceLocked` (5 sous-cas dont « load non appelé »).
- [x] 1.2 `internal/api/server_apiv1.go:158-172` : `instanceLockedFn` = closure sur
      `authz.InstanceLocked`. **Déplacée plus haut dans `mountAPIV1`** (avant le montage du
      handler bootstrap) : le bloc d'origine (ligne ~308) était postérieur au montage du
      bootstrap, qui en a désormais besoin aussi. Injectée dans `XboxSSOLinkStrategy`
      (`:349` inchangé), `UserAuthHandler` (`:379` inchangé), **`SetupHandler`**
      (`:511`, nouveau `WithInstanceLock` + `WithUserLookup(users)`) et **`BootstrapService`**
      (`:161`, 4e copie découverte — cf. §10).
- [x] 1.3 `internal/api/handlers/setup.go` : la double garde est extraite dans
      `guardProvisioning(ctx, canSelfProvision, actorIsAdmin)` (`:251-270`) et l'admin est
      résolu par `actorIsAdmin(ctx)` (`:232-247`) — `authz.CurrentUser` quand
      `WithUserLookup` est câblé (un admin rétrogradé depuis l'ouverture de sa session perd
      l'exemption), sinon repli sur `sess.Role`, même source que `middleware.RequireAdmin`.
      Journal `slog.InfoContext(ctx, "setup: création profil par admin", "gamertag", ...)`
      posé après validation du gamertag. `h.instanceLocked` nil ⇒ non verrouillé (même
      convention que `XboxSSOLinkStrategy`/`UserAuthHandler`, documentée sur le champ).
- [x] 1.4 `internal/platform/settings/` : `NewStore(path)` inchangé,
      `WithEnforcedDefaults(enforced bool) *Store` ajouté. `applyAbsentDefaults` prend un 3e
      paramètre `enforced` : `instance_locked` absent ⇒ `true`, `can_self_provision` absent
      ⇒ `false` ; hors mode appliqué, défauts historiques stricto sensu. `defaultSettings()`
      INCHANGÉ ; la branche « fichier absent » de `Load` passe désormais par
      `applyAbsentDefaults(cfg, nil, …)` — sans quoi un app_settings.json absent restait
      ouvert en mode appliqué (même trou, autre porte). Aucune écriture du fichier.
      **Les trois fonctions concernées ont été SORTIES dans `settings/defaults.go`** :
      `store.go` passait à 598 L, il est redescendu à 516 L (556 L avant l'étape) — la dette
      de seuil baisse au lieu de monter (CLAUDE.md règle 5).
      Câblage : `internal/api/server.go:658` et `cmd/server/main.go:720`, tous deux
      `.WithEnforcedDefaults(authz.Enforced(cfg.DemoMode, cfg.AuthMode))`.
      Tests `defaults_enforced_test.go` (5 cas : clés absentes appliqué / non appliqué,
      valeurs explicites gagnantes, fichier absent, overlay par titre).
      Doc des deux champs mise à jour dans le MÊME commit (anti « doc inversée »).
- [x] 1.5 Ratchet `internal/archlint/no_bare_instance_lock_read_test.go` : en-tête
      POURQUOI / PORTÉE, allowlist datée du 2026-09-15 (`internal/authz/`, `internal/config/`,
      `internal/platform/settings/`, `internal/domain/`, `internal/api/handlers/settings.go`,
      `internal/api/server_apiv1.go`). Balayage du MODULE entier (pas seulement `internal/`),
      `_test.go` exclus, lignes de commentaire ignorées, `authz.InstanceLocked(` jamais une
      violation. Vert.
- [x] 1.6 Tests : `setup_admin_exempt_test.go` — matrice des 4 combinaisons du plan
      (admin+verrou → 201, user+verrou → 403 `instance_locked`, user+provisioning coupé →
      403 `provisioning_disabled`, admin+provisioning coupé → 201) plus le cas anonyme, et
      chaque cas vérifie AUSSI que `CreatePlayer` n'est appelé que sur un 201.
      `TestSetupHandler_CreatePlayer_AdminFromUserStore` : rôle du store prioritaire sur
      celui de la session. `setup_test.go:80` : le test de verrou existant injecte le
      résolveur (le handler ne lit plus `cfg.InstanceLocked`). `user_auth_test.go` et
      `xbox_auth_service_test.go` inchangés — le résolveur injecté garde le type `func() bool`.

**Gate G1** ✅ (2026-09-15, 22:16 → 22:21) :
- `go test -count=1 ./internal/authz/... ./internal/api/handlers/... ./internal/platform/settings/...
  ./internal/archlint/... ./internal/service/...` → **9 paquets `ok`, 0 échec** (authz 7,8 s ·
  api/handlers 51,5 s · platform/settings 5,1 s · archlint 65,7 s · service 52,8 s + 4
  sous-paquets).
- `grep -rn "InstanceLocked ||" apps/go-api/internal` → **0 ligne de code de production**
  (3 occurrences, toutes dans des `_test.go` : le commentaire du ratchet qui cite le motif
  interdit, et 2 assertions `!cfg.InstanceLocked || cfg.CanSelfProvision`).
- `go vet ./...` → **0**.

**Réserve G1 (consignée, non masquée)** : au PREMIER passage du gate (22:00 → 22:05),
`internal/service` avait échoué, pendant que le rattrapage `internal/sync` (501 s) tournait en
parallèle sur la même machine. Le paquet est repassé vert **deux fois** ensuite, dont une avec
`-count=1` (cache désactivé), seul puis dans le gate complet. Le nom du test en échec n'a pas pu
être récupéré : la sortie du premier passage avait été tronquée par un `| tail -40`. Hypothèse
retenue : test sensible au temps sous contention CPU (`internal/service` compte 230 `t.Parallel()`
et plusieurs tests à TTL/deadline — `remote_stats_cache_test.go`, `career_live_cache_test.go`,
`squad_service_v2_test.go`). Aucun test n'a été désactivé ni skippé. À re-vérifier à l'étape 7,
gate complet machine au repos.

## 4. Étape 2 — Portes « profil suivi » sur le watcher et le coordinateur (moyen) — agent A

- [x] 2.1 `internal/domain/identity.go` (nouveau) : `type ProfileGate func(ctx, titleSlug, xuid string) bool`,
      avec l'en-tête de paquet qui pose la clé d'identité (le xuid, ADR 0035 D1) et le
      récit de l'incident du 2026-07-23. Le fichier est volontairement le point d'ancrage
      des types de l'annuaire : l'étape 3 l'étendra (`IdentityRecord`, anomalies…).
- [x] 2.2 `internal/config/config_players.go:198-223` :
      `func (c *AppConfig) HasTrackedProfile(titleSlug, xuid string) (bool, error)` —
      `LoadPlayers(titleSlug)` → `domain.SyncablePlayers` → recherche par **xuid**.
      xuid vide ⇒ false sans lecture ; erreur de lecture REMONTÉE (c'est au caller de
      décider de sa dégradation ; les trois portes refusent, en journalisant).
      `config_has_tracked_profile_test.go` : 7 cas (normal, auth_only, pause, inconnu, vide,
      autre titre, même profil sur SON titre) + un test dédié « un gamertag n'ouvre jamais
      la porte » + fichier absent (pas une erreur) + fichier illisible (erreur remontée).
- [x] 2.3 `internal/sync/coordinator.go` : `WithProfileGate(domain.ProfileGate) *Coordinator`
      (`:222-232`) ; dans `Submit` (`:246-259`), la porte se ferme AVANT le claim in-flight —
      un joueur refusé n'occupe ni slot de dédup, ni sémaphore, ni goroutine. Refus =
      `slog.WarnContext` (gamertag, xuid, title_slug, match_count, event) +
      `observability.IncCounter("sync_refused_no_profile")` + `return false`.
      Le compteur est déclaré avec les trois compteurs expvar existants du gate
      (`metricGate*`, en tête de fichier) — c'est le registre `internal/observability`
      de l'ADR 0009, aucun second mécanisme. Le titre est normalisé sur
      `titlePkg.DefaultSlug` avant d'interroger la porte, comme le fait `gateKey` juste
      après (même piège qu'en 2.4 : un titre vide serait lu « tous les titres »).
      **Vérifié sur pièces** : `CoordinatorRequest` portait DÉJÀ `XUID` et `TitleSlug`,
      et `consumeQueue` les renseigne — rien à ajouter.
- [x] 2.4 `internal/watcher/daemon.go` : `WithProfileGate(domain.ProfileGate) *Daemon`
      (`:160-172`) ; `AddPlayer` (`:345-364`) rend la sentinelle exportée
      `ErrPlayerNotTracked` et journalise en WARN ; `initPlayers` inchangé (liste déjà
      filtrée par `SyncablePlayers`). Le titre est normalisé sur `title.DefaultSlug` AVANT
      d'interroger la porte, comme le fait `playerKey` juste après : un titre vide serait
      sinon lu « tous les titres » par le chargeur de profils, et le profil d'un AUTRE jeu
      ouvrirait le suivi de celui-ci.
- [x] 2.5 Câblage. `cmd/server/main.go:2238-2254` : la porte est construite après
      `watcher.NewDaemon` et AVANT `daemon.Start` ; erreur de lecture de `db_profiles.json`
      ⇒ `slog.ErrorContext` puis refus (on ne synchronise pas « dans le doute »).
      `internal/api/server_apiv1.go:355-367` : porte du SSO (même forme, journal dédié).
      Ces deux closures sont les DEUX seules copies du motif « HasTrackedProfile + refus
      journalisé » : seuil de la règle 6 non atteint (une 3e imposerait un helper + un
      garde-rail). Elles ne sont pas fusionnables sans traverser la frontière
      `cmd/server` ↔ `internal/api` pour deux lignes.
      **Écart assumé par rapport au plan** : le plan prévoyait DEUX poses en `main.go`
      (coordinateur puis daemon). Sur pièces, le `Coordinator` est construit DANS
      `watcher.NewDaemon` (`daemon.go:149`) et n'est exposé que derrière l'interface
      `SyncGate` — il n'est pas atteignable depuis `main.go`. `Daemon.WithProfileGate` pose
      donc les DEUX portes (elle délègue à `d.coordinator.WithProfileGate`) : c'est le point
      de câblage unique du chemin watcher, et le lien est tenu par un test
      (`TestWithProfileGate_PoseAussiLaPorteDuCoordinateur`). Exposer le coordinateur
      seulement pour satisfaire la forme du gate aurait été le contraire d'un progrès.
- [x] 2.6 `internal/service/xbox_auth_service.go` : `WithProfileGate` (`:118-124`) ;
      `OnAuthSuccess` n'appelle `notifyWatcher` que si `watcherAllowedFor` s'ouvre pour
      (`title.DefaultSlug`, xuid) — sinon
      `slog.InfoContext(ctx, "xbox_sso: compte sans profil suivi — watcher non notifié, provisioning requis", …)`.
      Documentation d'en-tête du fichier (`:1-20`) et du type (`:64-77`) mises à jour : ce que
      la stratégie NE fait PAS, et pourquoi le compte + les tokens restent créés.
      Câblage `server_apiv1.go:366`.
- [x] 2.7 Tests. `sync/coordinator_profile_gate_test.go` : refus (Submit false, porte
      interrogée avec le bon couple, compteur à 1, aucun `RunSync`, aucun claim résiduel),
      porte ouverte (comportement d'origine, compteur à 0), porte nil (seam).
      `watcher/daemon_profile_gate_test.go` : `ErrPlayerNotTracked`, aucun `PlayerWatcher`,
      aucun cancel de poller enregistré ; porte ouverte ; titre vide normalisé ; porte nil ;
      et le test qui garde le lien daemon → coordinateur.
      `service/xbox_auth_profile_gate_test.go` : porte fermée ⇒ login OK, compte créé,
      refresh token persisté, session câblée, **`AddPlayer` jamais appelé** ; porte ouverte ⇒
      appelé ; porte nil ⇒ appelé. Les fakes existants (`mockDaemon`, `newXboxStore`) sont
      réutilisés, aucun n'a été modifié.
- [x] 2.8 Front. **Vérifié sur pièces : la redirection N'EXISTAIT PAS** pour le cas qui nous
      occupe. `setup_required` (`bootstrap_service.go:144`) et `setup_state`
      (`:462-485`) décrivent l'INSTANCE (« existe-t-il au moins un profil ? »), alors que
      `available_players` est filtré par propriété (ADR 0029). Sur une instance déjà peuplée,
      un compte SSO sans profil recevait donc `setup_state: 'ready'` : `postLoginDestination`
      (`apps/web/src/features/auth/postLoginDestination.ts:29`) l'envoyait sur
      `/onboarding/openspartan`, page qui annonce « on synchronise tes derniers matchs » et
      offre « Continuer → » vers l'accueil — alors que depuis l'étape 2 plus rien ne tourne
      pour lui. Et `SetupPage.tsx` le renvoyait à l'accueil (`setupState === 'ready' || !setupRequired`),
      donc une simple garde de route aurait bouclé.
      Ajouté : `apps/web/src/features/setup/setupRouting.ts` — quatre fonctions PURES
      (`needsOwnProfile`, `setupRedirectPath`, `resolveSetupStep`, `shouldLeaveSetup`), aucune
      logique dans les composants (CLAUDE.md règle 7) ; garde dans `routes/__root.tsx:160-178`
      (et bypass du shell sur `/setup` pour ce compte) ; `SetupPage.tsx` pilote son étape par
      `resolveSetupStep` et ne rend la main que via `shouldLeaveSetup`.
      Tests `setupRouting.test.ts` : 17 cas (admin exclu, auth non activée exclue, non
      connecté exclu, anti-boucle sur `/setup`, pages d'auth et pages anonymes épargnées,
      étape « profil » imposée malgré une instance « prête », et la garde anti-boucle de
      sortie). Aucune chaîne d'interface nouvelle ⇒ aucun manifeste i18n à régénérer.

**Gate G2** ✅ (2026-09-15) :
- `go test -count=1 -timeout 30m ./internal/sync/... ./internal/watcher/... ./internal/service/...
  ./internal/config/... ./internal/domain/... ./internal/archlint/...` (les deux derniers
  paquets en plus du gate, pour couvrir `domain/identity.go` et les ratchets) → **23 paquets
  `ok`, 0 échec**, exit 0 (22:42 → 22:45 ; `internal/sync` 122,4 s, `watcher` 1,3 s,
  `service` 14,1 s, `config` 1,2 s, `domain` 0,3 s, `archlint` 18,5 s).
- `go test -count=1 -tags=integration -timeout 30m ./internal/sync/...` → **10 paquets `ok`,
  0 échec**, exit 0 (22:45 → 22:49 ; `internal/sync` 249,1 s, `replayartifacts` 63,0 s,
  `killcollector` 20,2 s).
- `grep -n "WithProfileGate" apps/go-api/cmd/server/main.go apps/go-api/internal/api/server_apiv1.go`
  → 3 lignes : `main.go:2241` (commentaire du câblage), `main.go:2254` (daemon **et**
  coordinateur), `server_apiv1.go:366` (SSO). Deux points d'appel pour les trois portes,
  cf. l'écart assumé en 2.5.
- Hors gate mais vérifié : `make check-types` → 0 ; `npx vitest run src/features/setup
  src/routes/__root.test.tsx` → 3 fichiers, 27 tests, 0 échec ; `npx eslint` sur les 4
  fichiers web touchés → 0.

## 5. Étape 3 — Domaine + service `PlayerDirectory` + `GET /admin/identities` (moyen) — agent B

- [x] 3.1 `internal/domain/identity.go` (`:36-159`) : les six types du plan, aux champs et
      sévérités prévus. **Deux ajouts assumés, tous deux exigés par la suite du plan** :
      `OrphanDirRef{TitleSlug, Name}` + le champ `IdentityRecord.OrphanDirs` — sans lui
      `computeAnomalies(rec)` ne pourrait pas rester la fonction PURE que demande 3.3
      (l'information « dossier sans profil » ne serait pas dans la ligne) ; et
      `WatchedPlayerRef{XUID, Gamertag, TitleSlug}`, type de lecture du watcher (cf. 3.3).
      `Detail` porte un CONTEXTE machine (slug, nom de dossier), jamais une phrase : les
      libellés FR/EN sont posés à l'étape 4 (et le ratchet `no_french_label_literal` interdit
      de toute façon un littéral accentué dans un fichier neuf de `internal/service` ou
      `internal/api/handlers` — cf. §10).
- [x] 3.2 `internal/port/player_directory.go` (nouveau) : `PlayerDirectory{List, Get,
      HasTrackedProfile}` + `ErrIdentityNotFound`. `Onboard`/`Purge` NON déclarées, comme le
      plan l'impose (elles arrivent avec leur implémentation aux étapes 5 et 6 ; un
      commentaire sur l'interface le dit, pour que l'agent C n'ait pas à le redécouvrir).
      **Pas de `noopPlayerDirectory`** : vérifié sur pièces, le pattern noop de
      `internal/port` n'existe QUE dans `repository.go` (`:76-95`, `:377-378`) pour les
      repositories ; `services.go` n'en porte aucun, et une impl nulle non utilisée serait du
      code mort (CLAUDE.md règle 7). Le contrôle de compilation est tenu à sa vraie place :
      `var _ port.PlayerDirectory = (*Directory)(nil)` dans le service (`directory.go:100`).
- [x] 3.3 `internal/service/playerdirectory/` (nouveau paquet, 4 fichiers) : `directory.go`
      (types, `New`, `List`/`Get`/`HasTrackedProfile`, tri, compteurs), `collect.go`
      (l'agrégateur), `anomalies.go` (`computeAnomalies`, PURE), `fs.go` (témoin disque via
      `PathResolver`). Aucun import DuckDB, aucune écriture.
      `XUID string json:"xuid,omitempty"` ajouté à `AdminUserSummary` (`domain/user.go:90-100`)
      et renseigné dans `userstore.Store.List` (`store.go:226`), avec son test (3.6).
      **Trois écarts de forme, tranchés sur pièces** :
      (a) `AccountsReader` ne porte QUE `List()` — `GetByXUID` n'est appelée par aucun chemin
      de l'annuaire (la jointure se fait sur la liste déjà chargée) ; l'ajouter aurait forcé
      chaque double de test à l'implémenter pour rien.
      (b) `ProfilesReader` porte AUSSI `HasTrackedProfile(titleSlug, xuid)` : le port doit
      répondre à cette question (3.2) et `*config.AppConfig` sait déjà y répondre depuis
      l'étape 2 — la redéclarer dans le service aurait fait une 2e définition de « suivi »,
      exactement ce que l'ADR 0035 D3 interdit.
      (c) le watcher rend `WatchedPlayers() []domain.WatchedPlayerRef` et non
      `WatchedKeys() []string` : `PlayerWatcher` porte déjà `xuid`, `gamertag` ET `titleSlug`
      (`player_watcher.go:48-56`), donc l'annuaire rattache le suivi live par XUID au lieu de
      re-découper la chaîne `gamertag|titre` de `playerKey` — un format interne qui aurait
      silencieusement cassé l'annuaire s'il changeait. Méthode nil-safe, sous `playersMu`
      (`watcher/daemon_watched.go`, fichier séparé : `daemon.go` est à 633 L).
      Dossiers orphelins : `PathResolver.PlayersRootDir(slug)` (et non `PlayersDir`, qui
      n'existe pas) pour chaque titre de `title.DefaultRegistry().All()`, comparaison de clé
      insensible à la casse comme `dbprofiles.File.FindKey` (`store.go:291`). Un dossier
      qu'AUCUN registre ne réclame produit une ligne à xuid vide plutôt que d'être perdu
      (cf. §10).
- [x] 3.4 `internal/api/handlers/admin_identities.go` (nouveau) : handler Huma sans logique
      (`h.directory.List`), monté dans le bloc `r.Route("/admin", …)` de `server_apiv1.go`
      (`:430-435`) — donc sous les MÊMES `RequireAuth` + `RequireAdmin` que les autres routes
      admin (`:413-414`), avec `middleware.NoStore` comme `/admin/token-health` et
      `/admin/monitoring/*` : une anomalie corrigée doit disparaître au rafraîchissement, pas
      au bout d'un cache. Annuaire non câblé ⇒ 503 typé (jamais un corps vide qui se lirait
      « aucune identité »). Câblage extrait dans `internal/api/server_player_directory.go`
      (modèle `server_presence.go`) : `server_apiv1.go` est un assembleur déjà exempté du
      seuil, on n'y ajoute pas d'adaptateurs. Le daemon n'y est atteignable que par
      `watcher.DaemonController` : il est lu par assertion sur la petite interface
      `WatchedReader` plutôt qu'en élargissant `DaemonController`, ce qui aurait forcé tous
      ses doubles de test à implémenter une méthode dont ils n'ont pas l'usage.
- [x] 3.5 OpenAPI : `make openapi-gen` → +162 lignes (`/admin/identities` +
      `AdminIdentitiesResponse`, `IdentityRecord`, `ProfileRef`, `AccountRef`, `TokenRef`,
      `OrphanDirRef`, `IdentityAnomaly`) ; `make generate-types` → +94 lignes dans
      `generated.ts` ; `make openapi-check` → **0** (contrat à jour ET `generated.ts` dérivé).
- [x] 3.6 Tests : `directory_test.go` (14 tests, fakes des 5 lecteurs) — identité complète → 0
      anomalie ; compte sans profil (l'état exact du 2026-07-23 : compte + token + dossier,
      pas de profil) ; token seul → `token_orphan` ; dossier orphelin sans AUCUN registre ;
      dossier de casse différente → PAS orphelin ; profil d'ami → 2 `info` ; suivi live sur un
      titre sans profil ; tri warning d'abord + compteurs ; erreur de registre remontée (3
      sous-cas) ; balayage disque en échec → dégradation sans conclure d'orphelin ; sans
      watcher ; `Get` ; `HasTrackedProfile` délégué (4 sous-cas).
      `anomalies_test.go` : la fonction pure seule, 8 cas + l'ordre warning-avant-info.
      `admin_identities_test.go` : 200 + forme, 500, 503 sans annuaire, et 401/403/200 sur la
      MÊME chaîne de middlewares que `server.go` (harnais repris de `admin_titles_test.go:160`).
      `watcher/daemon_watched_test.go` : couples rendus, titre vide normalisé, récepteur nil,
      aucun joueur. `userstore/store_test.go:TestList_PorteLeXUID` : xuid présent après
      `LinkIdentity`, vide sinon.

**Gate G3** ✅ (2026-09-15, 23:09 → 23:11) :
- `go test -count=1 ./internal/domain/... ./internal/service/playerdirectory/...
  ./internal/api/handlers/... ./internal/platform/userstore/... ./internal/watcher/...` →
  **9 paquets `ok` (8 testés + 1 sans test), 0 échec**, exit 0, 14 s (domain 0,3 s ·
  domain/title 8,7 s · playerdirectory 0,3 s · api/handlers 10,0 s · userstore 4,4 s ·
  watcher 1,2 s).
- `make openapi-check` → **0** (openapi-gen -check « à jour » + `generated.ts` dérivé).
- `make check-types` → **0** (19 s).
- Hors gate, parce que le câblage touche `internal/api` et que deux ratchets balaient les
  fichiers neufs : `go test -count=1 ./internal/api/... ./internal/archlint/...` → **0 échec**
  (36 s) ; `golangci-lint run ./internal/service/playerdirectory/... ./internal/port/...` →
  **0 issue**, et aucun constat portant sur `admin_identities.go`,
  `server_player_directory.go`, `daemon_watched.go` ni `identity.go` dans les paquets déjà
  endettés (les 19 + 3 constats restants sont la baseline, fichiers non touchés).

## 6. Étape 4 — Web : section « Identités » sur `/admin/management` (moyen) — agent B

- [ ] 4.1 `apps/web/src/lib/query/keys.ts` : `adminIdentities: ['admin', 'identities'] as const`.
- [ ] 4.2 `apps/web/src/features/admin/management/identitiesQueries.ts` : `useAdminIdentities()`
      (`api.get<AdminIdentitiesResponse>('/admin/identities')`, types depuis `generated.ts`).
- [ ] 4.3 `apps/web/src/features/admin/sections/IdentitiesSection.tsx` : TanStack Table
      (règle 13) — colonnes gamertag, xuid (monospace, copiable), compte (username · rôle),
      profils (badges titre + « pause » / « auth seule »), token (ok / ré-auth requise / erreur),
      suivi live (titres), anomalies (badges par sévérité via tokens sémantiques, tooltip =
      détail). Tri par défaut : anomalies warning d'abord. Ligne d'en-tête avec compteurs.
      Aucune couleur hex ni classe Tailwind couleur (skill `color-tokens`).
- [ ] 4.4 `AdminManagementPage.tsx` : section « Identités » AU-DESSUS de `UsersSection` (c'est la
      vue d'ensemble ; les comptes en sont un sous-ensemble).
- [ ] 4.5 i18n : clés `admin.identities.*` (titre, colonnes, sévérités, libellés d'anomalies
      FR + EN) dans `apps/web/src/lib/i18n/manifests/admin.toml` puis
      `node apps/web/scripts/build_i18n_manifests.mjs` (vérifier l'invocation dans le script) ;
      `generated/admin.ts` régénéré et versionné.
- [ ] 4.6 Tests vitest : `IdentitiesSection.test.tsx` (rend 2 identités, badges d'anomalies,
      état vide) ; `identitiesDisplay.ts` + test si une fonction de formatage est extraite
      (sévérité → token, code → clé i18n) — logique hors composant (règle 7).
- [ ] 4.7 Toggle admin « Instance fermée » (ajouté le 2026-09-15 après constat : le backend
      accepte `PATCH /settings {instance_locked}` sous rôle admin — `handlers/settings.go:~228` —
      mais AUCUNE page ne l'expose ; la prod a dû être verrouillée à la main dans le fichier).
      Sur `/admin/management`, en tête de la section « Identités » : interrupteur « Instance
      fermée » (état lu depuis `useAppShellStore.instanceLocked` / bootstrap ; écriture via le
      hook existant `useUpdateSettings` de `features/settings/queries.ts` avec
      `{ instance_locked }` — vérifier sur pièces qu'il accepte ce champ, sinon l'ajouter au
      type de requête) + texte d'aide FR/EN (« Un compte Xbox inconnu ne peut plus créer de
      compte ni de profil ; les comptes existants ne sont pas affectés. »). Après succès :
      invalider `bootstrap` (le verrou est servi par `/bootstrap`) pour que l'état affiché suive.
      Aucune logique dans le composant : `identitiesDisplay.ts` ou un hook `useInstanceLock()`
      pour l'état + mutation. Test vitest : rendu de l'état, clic → mutation appelée avec la
      bonne valeur, désactivé pendant l'envoi.

**Gate G4** : `make check-types` → 0 ; `cd apps/web && npx vitest run src/features/admin src/features/settings` → 0 échec ;
`npm run lint:colors` → 0 ; `npm run lint` → 0 nouvelle erreur ; `grep -rn "instance_locked" apps/web/src/features/admin`
→ ≥ 1 ligne de production (le toggle existe).

## 7. Étape 5 — Chemin d'onboarding unique (moyen, risque auth) — agent C

- [ ] 5.1 `port.PlayerDirectory` étendu : `Onboard(ctx, domain.OnboardRequest) (domain.OnboardResult, error)` ;
      `OnboardRequest{TitleSlug, Gamertag, XUID, InitialMaxMatches, ActorUsername}` ;
      `OnboardResult{PlayerKey, DBPath, DBCreated, WatcherNotified bool, Warnings []string}`.
- [ ] 5.2 `service/playerdirectory/onboard.go` : `Onboard` = `ProfileService.CreatePlayer`
      (injecté par interface `ProfileCreator{CreatePlayer(req) (key, warnings, err)}`) → si
      daemon injecté et `IsRunning()` → `AddPlayer(PlayerSummary{...})` ; échec AddPlayer →
      `slog.ErrorContext` + `WatcherNotified=false` (pas d'erreur : le profil est la vérité,
      `initPlayers` le reprend au boot). Journal `slog.InfoContext(ctx, "directory: profil créé", "xuid", "gamertag", "title_slug", "actor")`.
- [ ] 5.3 `handlers/setup.go:173-200` : remplacer `h.profileSvc.CreatePlayer(req)` par
      `h.directory.Onboard(...)` (setter `WithDirectory` ; `profileSvc` retiré du handler si
      plus utilisé — vérifier `SetTitleSyncEnabled`/`PurgeTitleData` : s'ils passent par ce
      handler, garder l'injection pour eux uniquement). Réponse inchangée (contrat OpenAPI).
- [ ] 5.4 Ratchet `internal/archlint/no_direct_profile_create_test.go` : `\.CreatePlayer\(`
      interdit hors `service/playerdirectory/` ; allowlist datée : tests.
- [ ] 5.5 Vérifier sur pièces les autres créateurs de profil : `cmd/token-capture`,
      `cmd/token-import`, `cmd/levelup seed*`, `AddFriendFlow` (web → `/setup/players`, donc
      couvert). Un CLI qui écrit `db_profiles.json` directement → le faire passer par
      `dbprofiles.Store` (pas par `Onboard` : pas de daemon en CLI) et le noter `[~]`.
- [ ] 5.6 Tests : `onboard_test.go` (profil créé + daemon notifié ; daemon arrêté → non
      notifié, pas d'erreur ; `CreatePlayer` échoue → erreur propagée, daemon non appelé) ;
      `setup_test.go` : 201 via directory (fake), warnings propagées.

**Gate G5** : `go test ./internal/service/playerdirectory/... ./internal/api/handlers/...
./internal/archlint/...` → 0 ; `grep -rn "\.CreatePlayer(" apps/go-api --include=*.go | grep -v _test | grep -v playerdirectory`
→ 0 ligne (hors la méthode elle-même dans `profile_service.go`).

## 8. Étape 6 — Purge d'identité + CLI (moyen) — agent C

- [ ] 6.1 `port.PlayerDirectory` étendu : `Purge(ctx, xuid string, opts domain.PurgeOptions) (domain.PurgeReport, error)` ;
      `PurgeOptions{DryRun bool}` ; `PurgeReport{XUID, Gamertag, Steps []PurgeStep{Kind, Target, Done bool, Err string}}`.
- [ ] 6.2 `service/playerdirectory/purge.go` : ordre — (1) watcher `RemovePlayer` (vérifier
      l'existence d'une méthode de retrait sur le daemon — `playerCancels` suggère un W2 ;
      sinon l'ajouter : cancel du poller + retrait de la map, sous lock) ; (2) pour chaque
      profil du xuid : `ProfileService.PurgeTitleData(title, key)` (retire l'entrée + dossier,
      handles évincés) ; (3) dossiers orphelins portant ce gamertag (insensible à la casse) ;
      (4) `tokenStore.Remove(xuid)` ; (5) `groups.RemoveMember(id, xuid)` pour chaque groupe
      de `ListForXUID` ; (6) `users.Delete(username)` si compte. Refus : compte `admin`
      (`ErrPurgeAdminRefused`), xuid vide. `DryRun` : construit le rapport sans exécuter.
      Chaque étape journalisée (`slog.InfoContext` done / `slog.ErrorContext` err) ; une étape
      en échec n'arrête pas les suivantes (rapport complet), l'erreur globale agrège.
      **Jamais** d'accès à `shared_*.duckdb` (ni import de `platform/duckdb` dans ce package —
      ratchet `no_duckdb_import` existant à vérifier/étendre).
- [ ] 6.3 CLI `apps/go-api/cmd/levelup/cmd_identity.go` : `levelup identity list` (table
      xuid / gamertag / compte / profils / token / anomalies) et
      `levelup identity purge <xuid> [--yes]` (sans `--yes` = dry-run, imprime le rapport et
      sort 0 ; `--yes` exécute). Construction du directory en mode CLI : readers fichiers, pas
      de daemon (nil-safe). Refuse si le serveur tient les player DB ? — non : `PurgeTitleData`
      évince les handles du process courant seulement ; documenter « arrêter le serveur ou
      utiliser la purge alors que le joueur n'est pas suivi » dans l'aide de la commande.
- [ ] 6.4 Tests : `purge_test.go` sur un `t.TempDir()` avec les 4 fichiers + un dossier joueur +
      un fichier `shared_matches_v2.duckdb` factice : après purge, compte/token/profil/dossier
      absents, groupes sans le membre, **sha256 du fichier shared inchangé** ; dry-run → rien
      supprimé ; admin → refus. `cmd_identity_test.go` : dry-run sans `--yes`, sortie contient
      le rapport.
- [ ] 6.5 `docs/COMMANDS.md` (EN) + `docs/COMMANDS.fr.md` (ou la variante FR existante — vérifier
      le nom) : section `identity list` / `identity purge`, même PR (règle 15).

**Gate G6** : `go test ./internal/service/playerdirectory/... ./cmd/levelup/...` → 0 ;
`go build ./cmd/levelup` → 0 ; `go run ./cmd/levelup identity list` sur le worktree local →
tableau des 4 joueurs locaux sans panique.

## 9. Étape 7 — Clôture (rapide) — pilote

- [ ] 7.1 Revue adversariale du diff complet (skill `adversarial-review`, contexte frais) ;
      constats corrigés dans la branche.
- [ ] 7.2 Gates complets : `cd apps/go-api && go test ./...` ; `go test -tags=integration ./internal/sync/... ./internal/persist/...` ;
      `make go-api-lint` (pas de nouvelle dette) ; `make check-types` ; `make test-web` ;
      `make openapi-check`.
- [ ] 7.3 Docs : `CLAUDE.md` (section « Architecture des Données » : ligne « Identités
      joueurs : 4 registres, clé xuid, lecture/écriture via `PlayerDirectory` — ADR 0035 » ;
      liste des ADR : `0035`) ; `docs/ARCHITECTURE_V6.md` EN + FR (paragraphe registres
      d'identité) ; ADR 0035 amendée avec le résultat mesuré ; §10 de ce plan renseigné ;
      plan frère : note en §10 (helper de verrou + `Onboard` à utiliser en 5.3/5.4).
- [ ] 7.4 `.ai/thought_log.md` : entrée `[2026-09-15]` (statut, décisions, résultats, suite).
- [ ] 7.5 Commit(s) sur `wt/player-directory` ; PAS de push, PAS de merge sans l'utilisateur.
- [ ] 7.6 Prod (à la main de l'utilisateur, après merge/déploiement) : verrou déjà posé (P0) ;
      `levelup identity purge 2533274796795729 --yes` sur le VPS ; vérifier
      `GET /admin/identities` → 0 anomalie warning.

## 10. Découvertes hors périmètre (ne pas traiter ici)

- `sync-full --gamertag` (CLI) exige le refresh token du joueur alors que le serveur emprunte
  au pool (déjà noté par le plan frère).
- `AdminUserSummary` sans xuid : corrigé ici (3.3) car nécessaire ; le reste du panel Users
  n'affiche toujours pas le xuid — à l'appréciation UI plus tard.
- `applyAbsentDefaults` ignore `show_progression` etc. (hors sujet).
- **[agent A, étape 1] 4e copie du calcul du verrou** : `internal/service/bootstrap_service.go`
  recalculait `s.cfg.InstanceLocked || getBoolSetting(appSettings, "instance_locked", false)`
  pour l'exposer au front — le plan n'en comptait que trois. TRAITÉE (elle bloquait le
  ratchet 1.5, donc le gate G1) : `BootstrapService.WithInstanceLock(func() bool)`, câblé
  dans `server_apiv1.go` juste avant le montage du handler bootstrap.
- **[agent A, étape 1] 2e défaut de `can_self_provision`** : `buildCapabilities`
  (`bootstrap_service.go`) lisait la clé dans la map brute avec son propre défaut `true`,
  indépendant de `settings.Store`. Aligné sur la même règle (`!authz.Enforced(...)`) dans
  le périmètre de 1.4 : sans cela le front aurait proposé une création de profil que
  `POST /setup/players` refuse en 403.
- **[agent A, étape 0] `internal/sync` dépasse le timeout par défaut de `go test`** (600 s)
  sur ce poste : 501 s de tests + build CGO DuckDB à froid. Ce n'est PAS une régression
  (relance verte). Tout gate qui inclut `./internal/sync/...` doit porter `-timeout 30m`.
- **[agent A, étape 1] `POST /setup/players` en `profile_mode="xbox"` reste verrouillé sur
  l'identité de la session** (409 `identity_mismatch` si le gamertag diffère) : un admin
  exempté des deux gardes ne peut donc pas créer le profil d'un AMI en mode xbox, seulement
  en mode `manual`. NON TRAITÉE (hors périmètre 1.3, qui ne lève que `can_self_provision`
  et le verrou) — à arbitrer avec le plan frère (invitations / `AddFriendFlow`).
- **[agent A, étape 2] `setup_state` et `setup_required` décrivent l'INSTANCE, pas
  l'utilisateur** (`bootstrap_service.go:144` et `:462-485`), alors que `available_players`
  est filtré par propriété (ADR 0029). Sur une instance déjà peuplée, un compte SSO sans
  profil recevait donc `setup_state: 'ready'`. TRAITÉE côté web seulement (item 2.8 :
  `setupRouting.ts` + garde de route + wizard) : rendre `setup_state` par-utilisateur côté
  serveur toucherait la garde `setup_required` de tout le monde — à arbitrer avec l'étape 3
  (l'annuaire donnera le bon signal : « ce xuid a-t-il un profil ? »).
- **[agent A, étape 2] `notifyWatcher` (SSO) ne renseigne pas `TitleSlug`** sur la
  `PlayerSummary` qu'il passe à `AddPlayer` : le titre tombait donc à vide. Sans
  normalisation, un titre vide est lu « TOUS les titres » par `LoadPlayers` — le profil
  Halo 5 d'un joueur aurait ouvert son suivi Halo Infinite. Traité DANS les portes
  (normalisation sur `title.DefaultSlug` dans `Coordinator.Submit` et `Daemon.AddPlayer`,
  avec un test chacun) plutôt qu'en changeant la charge utile du SSO — hors périmètre, et
  la valeur vide a la même signification partout ailleurs (`playerKey`, `gateKey`).
- **[agent A, étape 2] commentaire orphelin en fin de `config_players.go`** :
  `// LoadAppSettings charge app_settings.json…` traîne seul en fin de fichier (la fonction
  vit ailleurs). Laissé en place, simplement repoussé après le nouveau code pour qu'il ne
  soit pas lu comme la doc de `HasTrackedProfile`. À supprimer un jour.
- **[agent B, étape 3] le ratchet `no_duckdb_import` de l'étape 6.2 N'EXISTE PAS.**
  `internal/archlint/` n'a aucun test de ce nom (63 fichiers, vérifiés un à un) : l'item 6.2
  dit « existant à vérifier/étendre », il faudra le CRÉER. Rien à corriger pour l'étape 3 —
  `internal/service/playerdirectory/` n'importe aucun paquet DuckDB (le témoin disque
  `fs.go` ne fait que `os.Stat`/`os.ReadDir` via `PathResolver`, il n'OUVRE jamais une player
  DB) — mais l'agent C doit prévoir l'écriture du garde-rail, pas son extension.
- **[agent B, étape 3] le ratchet `no_french_label_literal` contraint tout fichier NEUF de
  `internal/{service,analysis,api/handlers,notify,games}`** : un littéral accentué hors
  argument direct de `slog.*`/`fmt.Errorf` y est interdit (allowlist par fichier avec compte
  du jour, jamais de nouvelle entrée). Conséquence tenue ici : les `Detail` d'anomalie et les
  messages d'erreur HTTP de l'annuaire sont des codes/contextes machine, les libellés vivent
  côté web (item 4.5). À savoir pour les étapes 5 et 6 (`onboard.go`, `purge.go`,
  `cmd_identity.go` — ce dernier hors périmètre du ratchet, qui ne balaie pas `cmd/`).
- **[agent B, étape 3] un dossier joueur orphelin peut n'appartenir à AUCUN xuid.** L'annuaire
  est keyé par xuid (D1), mais un dossier n'a qu'un nom. Choix retenu : rattachement par
  gamertag insensible à la casse quand un registre le connaît, SINON une ligne à xuid vide
  portant le nom du dossier. Le masquer aurait reproduit le trou que l'annuaire ferme (un
  dossier que personne ne réclame est précisément ce qu'on cherche). Même traitement pour un
  profil ou un compte sans xuid. Côté web (étape 4), ces lignes doivent rester lisibles : la
  colonne xuid y est vide, jamais la ligne absente.
- **[agent B, étape 3] `GET /admin/users` expose désormais `xuid`** (ajout additif à
  `AdminUserSummary`, exigé par 3.3). Contrat OpenAPI régénéré ; aucun consommateur web ne
  lit ce champ aujourd'hui (le panel Users ne l'affiche toujours pas — déjà noté plus haut).
- **[agent B, étape 3] `internal/service` compte 6 sous-paquets** (`demo_fixtures`,
  `fragdist`, `replayview`, `squadagg`, `teammates`, `testdata`) : `playerdirectory` en est le
  7e, la forme « sous-paquet de service » est bien la convention du dépôt.

## Avancement

| Étape | Statut | Agent | Gate | Note |
|---|---|---|---|---|
| 0 | **terminée** | pilote/A | G0 ✅ | jonction node_modules OK ; baseline 0 échec (`internal/sync` exige `-timeout 30m`) |
| 1 | **terminée** | A | G1 ✅ | 6/6 items `[x]` ; verrou = `authz.InstanceLocked` + ratchet module-wide ; 4e copie trouvée et traitée (§10) ; réserve : un flake `internal/service` sous contention, non reproduit |
| 2 | **terminée** | A | G2 ✅ | 8/8 items `[x]` ; portes posées sur le coordinateur, le daemon et le SSO ; compteur `sync_refused_no_profile` ; garde web ajoutée (la redirection n'existait pas) ; 1 écart de forme assumé en 2.5 |
| 3 | **terminée** | B | G3 ✅ | 6/6 items `[x]` ; port + paquet `playerdirectory` (4 fichiers, 0 import DuckDB) + `GET /admin/identities` ; 3 écarts de forme assumés en 3.3 (dont le suivi live rendu par xuid) ; xuid ajouté à `AdminUserSummary` ; 4 découvertes en §10 |
| 4 | à faire | B | G4 | |
| 5 | à faire | C | G5 | |
| 6 | à faire | C | G6 | |
| 7 | à faire | pilote | — | |
