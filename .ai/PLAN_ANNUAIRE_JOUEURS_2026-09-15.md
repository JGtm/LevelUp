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

- [x] 4.1 `apps/web/src/lib/query/keys.ts:332-335` : `adminIdentities: ['admin', 'identities']`.
      **Le garde-rail `keys.title-slug.guard.test.ts` exige un classement explicite** de toute
      nouvelle clé (constat au gate, cf. §10) : classée AGNOSTIQUE avec justification écrite —
      une identité porte ses profils de TOUS les titres, et c'est la question posée ; la scoper
      par titre masquerait le profil d'un autre jeu, donc l'anomalie cherchée.
- [x] 4.2 `features/admin/management/identitiesQueries.ts` : `useAdminIdentities()`,
      `staleTime` 30 s, `retry: false` (même forme que `useAdminTokenHealth`).
- [x] 4.3 `features/admin/sections/IdentitiesSection.tsx` : TanStack Table, 7 colonnes
      (joueur, xuid copiable d'un clic, compte `username · rôle`, profils en badges de titre +
      états « en pause » / « auth seule » / « sans dossier », identifiants en un badge d'état,
      suivi live, anomalies en badges colorés par sévérité, `title` = le détail machine).
      Tri par défaut sur la colonne Anomalies, `desc` (les `warning` devant) ; ligne de
      compteurs au-dessus du tableau. **Zéro couleur en dur** : `tokenCssVar('warning'|'info'|
      'success'|'destructive')`, `npm run lint:colors` vert. Un code d'anomalie inconnu du web
      est affiché BRUT plutôt que rendre une ligne muette (le serveur peut en livrer un
      nouveau avant le front).
- [x] 4.4 `AdminManagementPage.tsx` : section « Identités » en PREMIER, au-dessus des comptes,
      avec le commentaire qui dit pourquoi. Le titre de section reste porté par la page (un
      `h2` par section, comme les deux autres) — la section ne le répète pas.
- [x] 4.5 i18n : 35 clés `admin.identities.*` FR + EN dans `manifests/admin.toml`, régénérées
      par `node scripts/build_i18n_manifests.mjs` (depuis `apps/web`, cf. §10) →
      `generated/admin.ts` versionné. Les six libellés d'anomalie vivent LÀ et nulle part
      ailleurs : le serveur ne rend que des codes.
- [x] 4.6 Tests vitest : `identitiesDisplay.test.ts` (10 tests — tokens de sévérité, les 6
      codes + un code inconnu, compte de warnings, clé de ligne quand le xuid est vide, états
      de profil, états d'identifiants et le fait que « absent » n'a PAS de couleur : c'est le
      cas NORMAL d'un ami servi par le pool) ; `IdentitiesSection.test.tsx` (8 tests — deux
      identités rendues dans l'ordre warning-d'abord, libellés FR, compteurs, état vide sans
      tableau, code inconnu affiché brut, et les 4 cas de l'interrupteur).
- [x] 4.7 Interrupteur « Instance fermée » livré en tête de la section, dans un hook dédié
      `features/admin/management/useInstanceLock.ts` (état + mutation + invalidation — aucune
      logique dans le composant). **Vérifié sur pièces, et ce n'était PAS le cas** : le type TS
      `SettingsResponse` (écrit à la main, `lib/api/types.ts:382`) ne portait pas
      `instance_locked`, donc `UpdateSettingsRequest = Partial<Omit<SettingsResponse, …>>` le
      refusait — alors que le Go l'expose ET l'accepte (`domain/settings.go:87` et `:166`).
      Champ ajouté au type TS (tous les usages sont des `Partial<>`, aucun littéral complet à
      mettre à jour). Après succès : `invalidateQueries({ queryKey: queryKeys.bootstrap })`,
      la seule source de l'état affiché. Texte d'aide FR/EN au mot près du plan. Tests :
      `useInstanceLock.test.tsx` (3 cas : lecture depuis le store, charge utile
      `{ instance_locked: true }` + invalidation de `bootstrap` via le rappel de succès, envoi
      en cours / échec remontés) et les 4 cas d'interface dans `IdentitiesSection.test.tsx`
      (état rendu, clic → bascule avec la bonne valeur, désactivé pendant l'envoi, échec
      affiché — jamais un silence).
**Gate G4** ✅ (2026-09-15, 23:28 → 23:45) :
- `make check-types` → **0** (deux passages : après la section, puis après le classement de la
  query key).
- `cd apps/web && npx vitest run src/features/admin src/features/settings` → **38 fichiers,
  233 tests, 0 échec** (10 s).
- `npm run lint:colors` → **0 violation**.
- `npm run lint` → **0 erreur**, 26 warnings. Le seul warning porté par un fichier neuf est
  `react-hooks/incompatible-library` sur `useReactTable` : il est INHÉRENT à TanStack Table et
  identique sur chaque tableau existant (vérifié en lançant eslint sur
  `monitoring/DetectionsPanel.tsx` seul → le même, à la ligne près). Aucune nouvelle famille.
- `grep -rn "instance_locked" apps/web/src/features/admin` → **2 lignes de production**
  (`useInstanceLock.ts:4` la doc, `:41` la charge utile) + 3 lignes de test.
- Hors gate, et il a payé : `npx vitest run` (suite COMPLÈTE) → **718 fichiers, 7704 tests,
  0 échec** (3 min 35). Le premier passage avait révélé un vrai échec — le garde-rail de
  classement des query keys (cf. 4.1 et §10) — noyé parmi 7 garde-rails qui expirent par
  contention (5 s de timeout par défaut pour un balayage de l'arbre des sources) ; tous verts
  relancés isolément, et verts aussi au second passage complet.

## 7. Étape 5 — Chemin d'onboarding unique (moyen, risque auth) — agent C

- [x] 5.1 `port.PlayerDirectory` étendu : `Onboard(ctx, domain.OnboardRequest) (domain.OnboardResult, error)`.
      Les deux types sont posés dans `internal/domain/identity.go` (`:161-196`), fichier
      d'ancrage de l'annuaire, aux champs prévus par le plan. `ActorUsername` est
      explicitement documenté comme JOURNAL SEUL : aucune décision d'autorisation ne s'y
      prend, les gardes restent dans le handler (ADR 0035 D5).
- [x] 5.2 `service/playerdirectory/onboard.go` (nouveau) : `ProfileCreator` +
      `WatcherNotifier` (deux petites interfaces locales, comme les cinq lecteurs de
      l'étape 3), `Onboard` et `notifyWatcher` ; `Deps.Creator` / `Deps.Watcher` ajoutés.
      Journal `slog.InfoContext(ctx, "directory: profil créé", …)` avec xuid, gamertag,
      player_key, title_slug et acteur. Échec `AddPlayer` → `slog.ErrorContext` +
      `WatcherNotified=false`, jamais d'erreur. **Un cas de plus que le plan, tranché sur
      pièces** : `AddPlayer` REFUSE un xuid vide (`daemon.go:336`), donc un profil manuel
      (mode `azure_manual`, sans identité Xbox) aurait produit un ERROR à chaque création.
      Il est court-circuité en amont avec un journal INFO — le watcher suit PAR xuid, sans
      xuid il n'y a rien à suivre, et c'est un cas NORMAL, pas une panne.
      `FS` gagne `PlayerDBPath(titleSlug, key)` : `OnboardResult.DBPath` est le chemin que
      la sync utilisera, et il se lit par `PathResolver` comme tous les autres.
- [x] 5.3 `handlers/setup.go` : `h.directory.Onboard(...)` (setter `WithDirectory`).
      **Vérifié sur pièces : `profileSvc` ne servait QU'À `CreatePlayer`** — ni
      `SetTitleSyncEnabled` ni `PurgeTitleData` ne passent par ce handler (ils vivent dans
      `TitleSyncHandler`, câblé sur le même `*service.ProfileService` partagé). Le champ, le
      paramètre de `NewSetupHandler` et l'interface `port.ProfileService` (plus aucun
      consommateur) sont donc SUPPRIMÉS (CLAUDE.md règle 7), ainsi que le helper
      `fileExists` et ses deux tests, devenus du code mort à tests verts dès que le chemin
      de la player DB est passé dans l'annuaire. Annuaire non câblé ⇒ 503
      `directory_unavailable` typé (jamais un 201 qui ferait croire à un profil créé).
      Réponse et contrat OpenAPI inchangés (`make openapi-check` → 0). Le câblage construit
      UNE instance d'annuaire (`server_apiv1.go:409-421`, `playerDirectoryDeps`) servant la
      lecture admin ET l'écriture setup : le créateur qu'elle porte est le writer unique de
      `db_profiles.json`. `guardLinkedXboxIdentity` extrait au passage —
      `handleCreatePlayer` passait à 94 L (> 80, dette gelée à ~85 L), il redescend à 71 L :
      la dette de seuil baisse au lieu de monter.
- [x] 5.4 Ratchet `internal/archlint/no_direct_profile_create_test.go` : en-tête
      POURQUOI / PORTÉE, balayage du MODULE entier, `_test.go` exclus, commentaires ignorés,
      allowlist datée du 2026-09-16 à UNE entrée (`internal/service/playerdirectory/`). Le
      motif exige un sélecteur (`\.CreatePlayer\(`) : la DÉFINITION de la méthode n'est
      jamais une violation. Vert.
- [x] 5.5 Autres créateurs de profil — inventaire VÉRIFIÉ SUR PIÈCES, aucun n'en crée :
      `cmd/token-capture` et `cmd/token-import` LISENT `db_profiles.json` et exigent que le
      joueur y soit déjà déclaré (en-têtes de leurs `main.go`) ; `levelup seed-demo` ne fait
      que lire (`cmd_data.go:240-258`, `ops.TitlesForGamertag`) ; `levelup add-title` écrit
      une SECTION DE TITRE vide, pas une entrée joueur, et le fait déjà par
      `dbprofiles.Store` (writer atomique) — `[~]` couvert par 5.4, que le ratchet laisse
      passer à juste titre ; `AddFriendFlow` (web) passe par `POST /setup/players`
      (`features/setup/queries.ts:46`), donc par `Onboard`. Aucun CLI à faire migrer.
- [x] 5.6 Tests : `onboard_test.go` (10 tests) — profil créé + watcher notifié (charge utile
      vérifiée) ; **l'ORDRE tenu par un test** (`TestOnboard_ProfilAvantWatcher` : le double
      du watcher rejoue la porte profil de `AddPlayer`, il refuse tant que le profil n'est
      pas là) ; daemon arrêté ; échec `AddPlayer` non propagé ; échec `CreatePlayer` propagé
      avec watcher jamais appelé ; sans watcher (CLI) ; sans xuid ; titre vide normalisé ;
      sans créateur ; gamertag vide. `setup_test.go` : 201 via l'annuaire (clé de profil et
      warnings rendus, demande transmise) et 503 sans annuaire ; les 3 fichiers de tests du
      handler passent du double `mockProfileService` au double `mockDirectory`.

**Gate G5** ✅ (2026-09-16, 00:07 → 00:08) :
- `go test -count=1 -timeout 30m ./internal/service/playerdirectory/... ./internal/api/handlers/...
  ./internal/archlint/...` → **3 paquets `ok`, 0 échec**, exit 0 (playerdirectory 0,3 s ·
  api/handlers 9,5 s · archlint 24,9 s).
- `grep -rn "\.CreatePlayer(" apps/go-api --include=*.go | grep -v _test | grep -v playerdirectory`
  → **0 ligne** (la définition dans `profile_service.go` ne matche pas : pas de sélecteur).
- Hors gate, parce que le câblage touche `internal/api` et que le contrat est en jeu :
  `go vet ./...` → 0 ; `go test -count=1 ./internal/api/... ./internal/domain/... ./internal/port/...`
  → **11 paquets `ok`, 0 échec** (26 s) ; `make openapi-check` → **0** ;
  `golangci-lint run --new-from-merge-base=origin/main ./internal/...` → **0 issue**, et
  `setup.go` ne porte plus AUCUN constat (il en portait un de funlen avant ce lot).

## 8. Étape 6 — Purge d'identité + CLI (moyen) — agent C

- [x] 6.1 `port.PlayerDirectory` étendu : `Purge(ctx, xuid, domain.PurgeOptions) (domain.PurgeReport, error)`.
      Types dans `internal/domain/identity.go` (`:198-248`), aux champs prévus + `DryRun` sur le
      rapport (la simulation et l'exécution ont la MÊME forme, donc le rapport doit dire
      laquelle il est) et six constantes de `Kind` (codes machine). `PurgeOptions.DryRun`
      porte un avertissement explicite : la valeur ZÉRO exécute, le défaut « simulation » est
      tenu par l'unique appelant (`--yes`), là où la décision d'un humain se prend.
- [x] 6.2 `service/playerdirectory/purge.go` (nouveau) : l'ordre du plan, tenu et testé.
      (1) `watcher.Daemon.RemovePlayer` — **la méthode n'existait pas** : ajoutée dans
      `internal/watcher/daemon_remove.go`, PAR XUID (`UpdateSubscriptions` travaille par
      gamertag, ce qui convient à une liste d'abonnement mais pas à une identité — un gamertag
      se renomme). Cancel du REST poller + `stopPoller` + retrait des deux maps sous
      `playersMu`, exactement comme le retrait existant : sans le cancel, la goroutine du
      poller survivrait (fuite W2). (2) profils : **`PurgeTitleData` NE CONVIENT PAS** —
      `Store.RemoveEntry` refuse le DERNIER titre actif d'un gamertag (`ErrLastActiveTitle`),
      invariant qui protège un joueur QUI RESTE ; appliqué titre par titre il échouerait
      systématiquement sur le dernier et laisserait le profil en place. `ProfileService
      .PurgeIdentityData(gamertag, titleSlugs)` ajouté : UNE mutation atomique pour tous les
      titres, puis suppression des dossiers (handles évincés d'abord). Un test dédié le prouve.
      (3) dossiers orphelins par `FS.RemovePlayerDir` (nouvelle méthode, chemin par
      `PathResolver`, nom refusé s'il porte un séparateur ou `..`) ; (4) `tokens.Remove` ;
      (5) `groups.RemoveMember` pour chaque groupe de `ListForXUID` ; (6) `users.Delete`.
      Refus : compte admin (`ErrPurgeAdminRefused`), xuid vide (`ErrPurgeInvalidXUID`), xuid
      inconnu (`port.ErrIdentityNotFound`). Chaque étape journalisée, une étape en échec
      n'arrête pas les suivantes, `errors.Join` agrège. Aucun accès à `shared_*.duckdb`.
- [x] 6.2bis Ratchet `internal/archlint/no_duckdb_import_playerdirectory_test.go` — **ÉCRIT**
      (il n'existait pas, cf. §10) : en-tête POURQUOI / PORTÉE, `internal/service/playerdirectory/`
      récursif, interdit `internal/platform/duckdb` ET `github.com/duckdb/duckdb-go`, `_test.go`
      exclus, commentaires ignorés. Vérifié EN ÉCHEC sur un import ajouté volontairement, puis
      remis vert — un garde-rail qu'on n'a pas vu rougir ne garde rien.
- [x] 6.3 CLI `cmd/levelup/cmd_identity.go` (nouveau) + dispatch `case "identity"` et aide
      dans `main.go`. `identity list` (xuid / gamertag / compte / profils / jeton / anomalies)
      et `identity purge <xuid> [--yes]`. Annuaire CLI = lecteurs fichiers, daemon nil.
      L'aide documente la précondition « le serveur ne doit pas tenir la player DB » (la purge
      n'évince que les handles du processus courant ; un fichier tenu ailleurs fait échouer
      l'étape, et le rapport le dit). **Piège corrigé sur pièces** : `flag` s'arrête au premier
      argument non-flag, donc `purge <xuid> --yes` — l'ordre documenté, celui qu'un humain
      écrit — aurait laissé `--yes` non lu, c'est-à-dire une SIMULATION là où l'on croyait
      exécuter ; le positionnel est extrait avant `Parse` et un test joue cet ordre exact.
      Sortie utilisateur par `io.Writer` (testable) avec la première erreur d'écriture retenue
      et remontée : un rapport tronqué ne passe pas pour un rapport complet.
- [x] 6.4 Tests. `purge_test.go` (7 tests) sur un `t.TempDir()` avec les VRAIS stores
      (db_profiles.json, users.json, groups.json, watcher_tokens/{xuid}.json, dossier joueur +
      player DB, dossier orphelin sur un second titre, `shared_matches_v2.duckdb` factice) :
      purge complète → **sha256 du shared inchangé**, dossiers/credentials/compte absents,
      groupe sans le membre, profil de l'admin intact ; dernier titre actif retiré quand même ;
      dry-run → rapport complet et RIEN supprimé (les six vérifications disque) ; admin →
      `ErrPurgeAdminRefused` sans aucune suppression ; étape en échec → les suivantes
      s'exécutent et l'erreur agrège ; xuid vide et inconnu ; ordre watcher-avant-profils tenu
      par un double qui note ses appels. `daemon_remove_test.go` (4 tests) : retrait par xuid
      tous titres sans toucher l'homonyme, cancel du poller consommé, idempotence + nil-safe,
      titre vide normalisé. `cmd_identity_test.go` (5 tests) : `list`, simulation sans `--yes`
      (rien supprimé), exécution avec `--yes`, xuid inconnu, routage.
- [x] 6.5 `docs/COMMANDS.md` (EN) + `docs/FR/COMMANDS.md` (**c'est le nom réel de la variante
      FR** — `docs/COMMANDS.fr.md` n'existe pas) : section « Player identities / Identités
      joueur » avant « Metadata / seed / migration », avec les cinq points qui comptent
      (entrepôt partagé jamais touché, simulation par défaut, ordre, refus de l'admin,
      précondition serveur arrêté).

**Gate G6** ✅ (2026-09-16, 00:38) :
- `go test -count=1 -timeout 30m ./internal/service/playerdirectory/... ./cmd/levelup/...` →
  **2 paquets `ok`, 0 échec**, exit 0 (playerdirectory 0,2 s · cmd/levelup 0,9 s).
- `go build ./cmd/levelup` → **0**.
- `go run ./cmd/levelup identity list` sur les registres locaux (LEVELUP_REPO_ROOT du dépôt
  principal, commande STRICTEMENT en lecture) → **12 identités, aucune panique** : les 4
  joueurs à profils multi-titres (JGtm admin, Chocoboflor, Madina97294, XxDaemonGamerxX), les
  6 amis `auth_only`, et 3 lignes `token_orphan` (fixtures `000000000000000{0,1,2}.json`,
  cf. §10). Exit 0.
- Hors gate : `go vet ./...` → 0 ; `./internal/archlint/... ./internal/watcher/...
  ./internal/domain/... ./internal/port/... ./internal/service/...` → **14 paquets `ok`,
  0 échec** ; `golangci-lint run --new-from-merge-base=origin/main ./...` → **0 issue**.

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
- **[agent B, étape 4] toute nouvelle query key doit être CLASSÉE** dans
  `apps/web/src/lib/query/keys.title-slug.guard.test.ts` (title-scopée ou agnostique, avec
  justification) : le garde-rail échoue sur une clé non classée. Il n'est pas joué par le gate
  G4 (`src/features/admin src/features/settings`) — il ne s'est manifesté qu'à la suite
  complète. À savoir pour les étapes suivantes qui ajouteraient une clé.
- **[agent B, étape 4] 7 garde-rails web expirent sous contention** (`Test timed out in
  5000ms`) quand la suite complète tourne en parallèle sur ce poste : ce sont ceux qui
  balaient l'arbre des sources (`noLocalUsageCopies`, `keys.guard`, `typeEquality`,
  `lab-removal`, `xuidMeta`, `fragClass.colorSource`, `heatmapColors`). Verts isolément et au
  second passage complet. Ce n'est PAS une régression, mais un premier passage de
  `make test-web` sur machine chargée peut afficher jusqu'à 30 fichiers en échec — ne pas s'y
  fier sans relancer (même piège que le flake `internal/service` de G1).
- **[agent B, étape 4] le type TS `SettingsResponse` est écrit À LA MAIN** (`lib/api/types.ts`)
  et dérive AU FIL DE L'EAU du `domain.SettingsResponse` Go : `instance_locked` y manquait
  depuis que le verrou existe (2026-06-08), ce qui rendait le champ inatteignable par
  `UpdateSettingsRequest`. Ajouté ici (4.7). D'autres champs peuvent manquer — non audité,
  hors périmètre.
- **[agent C, étape 5] `watcher.Daemon.AddPlayer` refuse un xuid vide** (`daemon.go:336`,
  antérieur à ce chantier). Un profil créé en mode `azure_manual` n'a pas de xuid : notifier
  le watcher aurait produit un `slog.Error` à CHAQUE création manuelle, c'est-à-dire du bruit
  sur un cas parfaitement normal. TRAITÉE dans le périmètre de 5.2 : `notifyWatcher`
  court-circuite en amont avec un journal INFO. La sentinelle du daemon n'est pas touchée.
- **[agent C, étape 5] `port.ProfileService` n'avait plus qu'un consommateur**, le
  `SetupHandler`. Une fois celui-ci passé par `Onboard`, l'interface devenait du code mort
  (CLAUDE.md règle 7) : supprimée. `service.ProfileService` (le TYPE) reste, consommé par
  `TitleSyncHandler` (pause/purge par titre) et par l'annuaire via l'interface locale
  `ProfileCreator` — la convention du paquet, comme ses cinq lecteurs.
- **[agent C, étape 5] `fileExists` (handlers) n'était plus utilisé QUE par ses propres
  tests** une fois le chemin de la player DB passé dans l'annuaire : c'est le « dead code
  museum » à tests verts du diagnostic de revue. Helper et tests supprimés.
- **[agent C, étape 6] `ProfileService.PurgeTitleData` ne peut pas purger une identité.**
  `Store.RemoveEntry` refuse le DERNIER titre actif d'un gamertag (`ErrLastActiveTitle`) —
  l'invariant « au moins un titre actif » protège un joueur QUI RESTE. Appliquée titre par
  titre, une purge échouerait donc systématiquement sur le dernier et laisserait le profil en
  place : purge incomplète, silencieuse côté fichier. TRAITÉE dans le périmètre de 6.2 :
  `PurgeIdentityData(gamertag, titleSlugs)` retire tout en UNE mutation atomique.
  `PurgeTitleData` est inchangée — elle sert le réglage par titre (`TitleSyncHandler`), où
  l'invariant a tout son sens.
- **[agent C, étape 6] `watcher.Daemon` n'avait AUCUNE méthode de retrait par joueur.**
  `UpdateSubscriptions` retire par GAMERTAG, ce qui convient à une liste d'abonnement mais pas
  à une identité (un gamertag se renomme, un xuid non). `RemovePlayer(ctx, xuid)` ajoutée
  (`daemon_remove.go`), pas sur l'interface `DaemonController` : l'y mettre forcerait tous ses
  doubles de test à l'implémenter sans usage — même raisonnement que `WatchedReader` à
  l'étape 3, et le câblage la prend par assertion.
- **[agent C, étape 6] un groupe DONT l'identité purgée est PROPRIÉTAIRE ne se quitte pas.**
  `groupstore.RemoveMember` rend `ErrCannotRemoveOwner`. L'étape est rendue EN ÉCHEC avec
  l'identifiant du groupe plutôt que de supprimer le groupe : celui-ci porte les accès
  d'autres joueurs, et le supprimer est une décision d'administrateur, pas d'une purge. Non
  traité au-delà (aucun groupe de ce type en local).
- **[agent C, étape 6] 3 `token_orphan` sur l'instance locale** :
  `data/auth/watcher_tokens/000000000000000{0,1,2}.json` (datés du 2026-08-20) — des fixtures
  à xuid factice qu'aucun compte ni profil ne réclame. C'est exactement ce que l'annuaire est
  fait pour montrer. NON TRAITÉ (hors périmètre) : à supprimer par
  `levelup identity purge 0000000000000000 --yes` quand l'utilisateur le décidera.
- **[agent C, étape 6] la variante FR de `COMMANDS.md` est `docs/FR/COMMANDS.md`**, pas
  `docs/COMMANDS.fr.md` comme l'item 6.5 le supposait. Tout `docs/FR/` suit cette forme.
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
| 4 | **terminée** | B | G4 ✅ | 7/7 items `[x]` ; section « Identités » en tête de la page Gestion (TanStack Table, 7 colonnes, tokens sémantiques, 35 clés FR+EN) + interrupteur « Instance fermée », que le backend acceptait mais qu'aucune page n'exposait ; 21 tests vitest neufs ; suite web complète verte (7704 tests) ; 3 découvertes en §10 |
| 5 | **terminée** | C | G5 ✅ | 6/6 items `[x]` ; `Onboard` seul chemin de création (profil PUIS watcher, ordre tenu par un test) ; ratchet `no_direct_profile_create` (allowlist à 1 entrée) ; `port.ProfileService` + `fileExists` supprimés (code mort) ; contrat OpenAPI inchangé ; 3 découvertes en §10 |
| 6 | **terminée** | C | G6 ✅ | 6/6 items `[x]` (+ 6.2bis) ; purge ordonnée, dry-run par défaut, refus admin, sha256 du shared inchangé ; `Daemon.RemovePlayer` et `ProfileService.PurgeIdentityData` ajoutés (les deux manquaient) ; ratchet `no_duckdb_import_playerdirectory` ÉCRIT et vu rougir ; CLI `levelup identity list/purge` + COMMANDS EN & FR ; 5 découvertes en §10 |
| 7 | à faire | pilote | — | |
