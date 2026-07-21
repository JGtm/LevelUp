# Fix — boucle de retour incessante vers `/login` (session anonyme transitoire)

## Contexte

L'utilisateur est renvoyé « sans cesse » vers la page `/login` alors qu'il s'est connecté
le jour même et que sa session doit être valide. Indice décisif fourni : **supprimer
`/login` de l'URL (rechargement plein sur `/`) le reconnecte** — donc la session backend est
bien **vivante**, et quelque chose l'éjecte **à tort**. Régression apparue « il y a ~1 semaine »,
sur **prod ET local**.

Diagnostic établi et **vérifié sur pièces** (lecture code + git). Cause racine à **deux couches** :

### Couche 1 — Backend : race de lecture « torn read » sur le fichier de session (LE DÉCLENCHEUR)

- `internal/platform/session/store.go` `Save` (L92-99) écrit via `os.WriteFile` : **non atomique**
  (truncate puis write), **aucun verrou**, **aucun rename**.
- `Touch`->`Save` est appelé en fin de **chaque** requête authentifiée
  (`internal/api/middleware/session.go:58-60`).
- `Load` (L74-90) fait `os.ReadFile` + `json.Unmarshal` et renvoie **`nil`** au moindre échec
  -> le middleware crée alors une **session anonyme** (`loadOrCreate` -> `store.New()`).
- `/bootstrap` lit `sess.Username` -> si session anonyme, renvoie `current_username: null`
  (toujours **HTTP 200**, jamais 401).
- **La race** : `refetchOnWindowFocus` (défaut React Query `true`, non surchargé dans
  `app/queryClient.ts`) déclenche au retour sur l'onglet une **rafale** de requêtes concurrentes ;
  chacune `Touch`/`Save` la même session pendant que `/bootstrap` fait son `Load` -> `Load` lit un
  fichier **tronqué** -> `nil` -> anonyme transitoire. Concurrence **intra-process** -> reproductible
  sur **prod (VPS mono-instance) comme en local**, pire sous Windows.

### Couche 2 — Frontend : éjection sur anonyme transitoire (LA RÉGRESSION VISIBLE)

- `apps/web/src/routes/__root.tsx:55-84` : `useEffect([data])` re-tourne à **chaque** résolution de
  `/bootstrap` (y compris refetch focus) ; sur `!current_username` il fait `navigate({to:'/login'})`
  (L68-71) **et** `hydrateFromBootstrap` rabat le store `currentUsername` à `null` (L57 ->
  `appShellStore.ts:140`).
- `aa5458fb7` (2026-07-16, **sur `main` = déployé prod**) a ajouté le vecteur **déclaratif**
  `/` -> `<Navigate to="/login">` via `resolveIndexRedirect` (`shellNavigation.ts:128`,
  `routes/index.tsx`). Amplificateur préalable : `d96daf182` (2026-06-25) a passé
  `refetchOnWindowFocus` à `true`. La date du 16/07 colle à l'onset « ~1 semaine ».
- Toutes les redirections `/login` automatiques exigent un état anonyme (les 6 vecteurs tracés :
  `__root:68`, `index.tsx`/`resolveIndexRedirect:128`, `players/$playerSlug.tsx:35`) : **aucun** chemin
  n'envoie vers `/login` avec une session valide -> le seul déclencheur est le bootstrap anonyme
  de la couche 1.

## Objectif & critère de succès

Un utilisateur authentifié n'est **jamais** éjecté vers `/login` par une réponse `/bootstrap`
anonyme transitoire (focus d'onglet, navigation multi-requêtes). La vraie déconnexion
(`LogoutButton` -> `POST /auth/logout` + reload plein sur `/`) continue d'atterrir sur `/login`.

**Succès mesurable :**
1. Test Go concurrent `Load`-pendant-`Save` : `Load` d'une session vivante ne renvoie **jamais**
   `nil` (rouge avant / vert après).
2. Test vitest : store authentifié + `/bootstrap` refetch anonyme => **pas** de `navigate('/login')`,
   `currentUsername` préservé ; montage frais anonyme => redirige bien vers `/login`.
3. Repro manuelle : onglet en arrière-plan > 5 min puis refocus répété => reste connecté.

## Correctif (2 étapes ordonnées, gate à chaque étape)

### Étape 1 — Backend : persistance atomique des sessions (supprime le déclencheur)

Fichier : `apps/go-api/internal/platform/session/store.go`

- [x] `Save` : écriture dans un fichier temporaire du **même répertoire** (`os.CreateTemp(s.dir,
  sanitizeID(id)+"-*.tmp")`) -> `Write` -> `Close` -> `os.Rename(tmp, target)` (atomic-replace
  cross-plateforme). En cas d'erreur : `os.Remove(tmp)` + retour de l'erreur (pas d'avalage).
- [x] `PurgeExpired` : nettoie les `*.tmp` orphelins (guard `orphanTmpTTL = 1h` pour ne jamais
  supprimer le `.tmp` d'un `Save` en vol) ; non comptés comme sessions.
- [x] **RWMutex** sur le `Store` (au lieu du simple `sync.Mutex` sur `Save` — écart au plan justifié) :
  `Save` prend `Lock`, `Load` prend `RLock`. NÉCESSAIRE et non optionnel : sous **Windows**, le rename
  atomique seul ne suffit pas intra-process — `os.Rename` échoue et `os.ReadFile` prend une *sharing
  violation* si un handle concurrent tient le fichier (empiriquement 109 Load nil / 177 Save err au 1er
  run avec rename seul). Le RWMutex sérialise lecture/écriture intra-process (les deux plateformes) ; le
  rename atomique reste la protection **cross-process** (doublon `air`). Couvre aussi les lost-updates
  login/OAuth. `Delete` reste sans verrou (appelé sous `RLock` par `Load` sur expiry -> pas de deadlock).

Fichier : `apps/go-api/internal/api/middleware/session.go`
- [x] L59 `store.Touch(sess)` : échec logué via
  `slog.ErrorContext(r.Context(), "session touch failed", "err", err)`.

Test garde-rail : `apps/go-api/internal/platform/session/store_concurrent_test.go` (nouveau)
- [x] 4 writers (`Load`->`Touch` sur copie fraîche par itération) + 4 readers (`Load` en boucle
  ~200 ms) sur une session `Username="alice"`. **Assertion** : chaque `Load` non-`nil` + `Username`
  préservé. **ROUGE** au 1er run (rename atomique seul, sans RLock : 109 nil) -> **VERT** après le
  RWMutex (`-race`, `-count=3`).

**Gate étape 1 :**
```
cd apps/go-api && go test ./internal/platform/session/... ./internal/api/middleware/...
cd apps/go-api && go test -race ./internal/platform/session/...   # OK ici : pas de driver DuckDB
cd apps/go-api && go test ./...
```
Résultat (2026-07-21) : session + middleware **verts** ; `-race` session **vert** (count=3) ;
`go build ./...` + `go vet` (paquets touchés) **verts** ; `go test ./...` **vert sauf** un flake
`internal/sync` (GET réseau réel vers halostats.svc.halowaypoint.com -> `context deadline exceeded`
sous charge parallèle) qui **passe en isolation** (`go test ./internal/sync/` = ok). Sans lien avec
session/middleware — cf. Découvertes.

### Étape 2 — Frontend : ne pas éjecter un utilisateur authentifié sur un anonyme transitoire

Fichier : `apps/web/src/routes/__root.tsx` (effet L55-84)
- [x] Avant `hydrateFromBootstrap(data)`, capture `wasAuthenticated =
  useAppShellStore.getState().currentUsername`.
- [x] Si `!data.current_username && wasAuthenticated` (rétrogradation suspecte via refetch) :
  `return` **avant** hydrate ET redirection + `log.warn('bootstrap:anon_downgrade', ...)`
  (`@/components/shell/_logger`). La redirection `/login` ne s'exécute que sur un anonyme
  **autoritaire** (`!wasAuthenticated` : chargement frais, reload post-logout).
- [x] **Variante choisie (écart aux 2 options du plan, justifié)** : `return` early = **ne pas
  hydrater du tout** sur downgrade suspect, plutôt que « hydrater puis restaurer `currentUsername` ».
  Raison : en mode xbox un bootstrap anonyme renvoie AUSSI `available_players: []`, `current_player:
  null`, `is_admin: false` — hydrater en ne préservant QUE `currentUsername` laisserait un état
  mi-anonyme incohérent (shell sans joueurs). Le skip préserve l'état authentifié **complet**. Comme
  `index.tsx`/`resolveIndexRedirect` et `players/$playerSlug.tsx` lisent le **store**, préserver le
  store corrige **tous** les vecteurs d'un coup.
- [x] `refetchOnWindowFocus: true` conservé (inchangé). Un refetch AUTHENTIFIÉ (cas nominal) hydrate
  toujours normalement, bannière reauth incluse ; seul le downgrade **anonyme** est ignoré.
- [x] (hors items plan, in-scope) `RootLayout` exporté pour testabilité ;
  `vite.config.ts` : `routeFileIgnorePattern: '\\.test\\.tsx?$'` pour exclure les tests colocalisés
  du codegen de routes (sinon warning « does not export a Route »).

Tests : `apps/web/src/routes/__root.test.tsx`
- [x] store authentifié (`currentUsername='alice'`) + `/bootstrap` refetch anonyme => `navigateMock`
  **jamais** appelé, `currentUsername` reste `'alice'`.
- [x] montage frais (store vide) + bootstrap anonyme => `navigate({ to: '/login' })` appelé,
  `currentUsername` null (chemin logout intact).

**Gate étape 2 :**
```
make check-types
make test-web        # vitest hors sandbox : dangerouslyDisableSandbox=true
```
Résultat (2026-07-21) : `check-types` (tsc -b) **vert** ; `__root.test.tsx` 2/2 **vert** ;
`make test-web` (suite complète) **vert** : 276 fichiers, 2382 tests, 14 skipped, 0 échec.

## Vérification end-to-end

1. **Avant fix (repro)** : `make dev`, se connecter, laisser l'onglet en arrière-plan > 5 min,
   revenir et re-focus plusieurs fois => observer l'éjection vers `/login`. Via chrome-devtools MCP :
   repérer le `GET /api/v1/bootstrap` **200** avec `current_username: null` dans la rafale de focus.
2. **Après fix** : même manip => reste connecté ; plus aucun `/bootstrap` anonyme dans la rafale
   (Étape 1) et, même si un anonyme transitoire survenait, aucune éjection (Étape 2).
3. **Non-régression logout** : cliquer « Se déconnecter » => reload sur `/` => atterrit sur `/login`.
4. **Sanity local** (couvre le « les deux ») : vérifier qu'un seul serveur tourne (pas de doublon
   `air`+`server.exe` orphelin avec un `LEVELUP_SESSION_SECRET` divergent) ; `LEVELUP_SESSION_SECRET`
   inchangé dans `.env.local`.

## Fichiers critiques

- `apps/go-api/internal/platform/session/store.go` — `Save`/`Touch` atomiques + `PurgeExpired`.
- `apps/go-api/internal/api/middleware/session.go` — log de l'échec `Touch` (L59).
- `apps/go-api/internal/platform/session/store_concurrent_test.go` — **nouveau** garde-rail.
- `apps/web/src/routes/__root.tsx` — garde anti-éjection transitoire.
- `apps/web/src/routes/__root.test.tsx` — tests frontend.
- Réutilisés (ne pas recréer) : `hydrateFromBootstrap` (`stores/appShellStore.ts:114`),
  `resolveIndexRedirect` (`components/shell/shellNavigation.ts:123`), logger client
  (`components/shell/_logger`).

## Découvertes hors périmètre (notées, NON traitées)

- `apps/web/src/lib/api/client.ts:148` dispatch `levelup:auth-required` mais **aucun listener**
  n'existe (code mort / câblage incomplet). Serait le **bon** canal pour une vraie expiration en cours
  de session (401 sur appel API réel), complémentaire du fix. À câbler dans un lot séparé.
- **Flake `internal/sync`** : ~~un test tente un `GET` réseau réel...~~ **RÉSOLU (lot complémentaire, cf.
  ci-dessous)**. Coupable : `TestHaloClient_GetMatchHistory_ParamsValides` (+ `TestPooledHaloClientGetMatchHistory`)
  faisaient de vrais appels réseau. Rendus hermétiques (contexte déjà annulé → échec HTTP instantané,
  `duration_ms=0`, zéro I/O réseau). Skip `-short` obsolète retiré. `time` import retiré.

## Lot complémentaire (demande utilisateur) — couverture logging + dette préexistante

Au-delà du plan initial, sur demande : couverture de logging dans le dossier `logs/` dédié + fix des
échecs/dettes préexistants. Réalisé et testé (2026-07-21) :

- **`session.Store` — logs des erreurs avalées** (module auto-détecté = `session` → `logs/session.log`) :
  - `Load(ctx, id)` (signature enrichie du ctx) : trace un WARN sur read IO / JSON corrompu (le retour
    `nil` silencieux était le point aveugle du bug) ; un fichier ABSENT reste silencieux (cas nominal).
  - `NewStore` : log ERROR si `MkdirAll` échoue (sessions non persistables → login cassé).
  - `PurgeExpired` : log ERROR si `ReadDir` échoue (fuite disque sinon).
- **Handlers auth — 3 `Save` avalés (`_ =`) corrigés** (`auth.go`, `auth_xbox_oauth.go` ×2) : log
  `slog.ErrorContext` (anti-pattern #10 « swallowed error »).
- **Tests** : `TestStore_Load_CorruptFile_LogsWarnAndReturnsNil` (nil + WARN sur corruption),
  `TestStore_Load_NotFound_NoLog` (absent = silencieux). Middleware `loadOrCreate` passe `r.Context()`.
- **Dette flake** : 2 tests `internal/sync` rendus hermétiques (cf. ci-dessus).
- Gates : `go test ./...` **vert** (flake éliminé) ; `go vet` défaut + `-tags=integration` **verts** ;
  `-race` session/middleware **vert**.

## Branche & livraison

- **Tâche distincte** du travail frags en cours (`feat/frag-distribution-v2`, WIP non commité).
  Créer une **nouvelle branche depuis `main`** : `fix/session-login-loop`. Le WIP frags est en cours
  sur la branche active -> **ne pas** `git stash` ; commiter le WIP ou coordonner avant switch
  (CLAUDE.md règle 16 / mémoire). 2 commits : backend (Étape 1), frontend (Étape 2).
- **`push main` = déploiement prod automatique** : prévenir avant merge/push. PR recommandée.
- Entrée `.ai/thought_log.md` avant commit ; passer `delivery-checklist` avant livraison.
- Exécution sous contrat `plan-execution` (ordre strict, gate par étape, statut par item).
