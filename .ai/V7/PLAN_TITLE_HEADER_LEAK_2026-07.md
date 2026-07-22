# Patch — Fuite directionnelle H5 → Infinite (titre non affirmé par les requêtes)

> Rédigé 2026-07-21. Périmètre validé : **header seul** (durcissement des query keys
> renvoyé au chantier « slug du titre dans l'URL », semaine suivante).

## Contexte

Fuite récurrente : en switchant **Halo 5 → Halo Infinite**, des données Halo 5 restent
affichées sur l'accueil Infinite (Spartan ID, playlists, rangs… « plein de trucs »).
**Toujours dans ce sens, jamais l'inverse.** Cette asymétrie est le signal diagnostique
clé. Le plan « slug du titre dans l'URL » (semaine prochaine) durcira la structure ; **ce
patch corrige dès maintenant la cause racine** de la fuite directionnelle, en une
poignée de lignes.

## Cause racine (vérifiée sur pièces)

Le client API n'envoie le header `X-LevelUp-Title` **que pour les titres non-défaut** :

```ts
// apps/web/src/lib/api/client.ts:80-85
function getTitleHeader(): Record<string, string> {
  if (_currentTitleSlug && _currentTitleSlug !== 'halo_infinite') {
    return { 'X-LevelUp-Title': _currentTitleSlug }
  }
  return {}   // ← halo_infinite : AUCUN header (« rétrocompatibilité »)
}
```

Côté backend, `resolveTitleSlug`
([title.go:55-72](apps/go-api/internal/api/middleware/title.go#L55-L72)) résout :
**header > session > défaut**. Conséquence directionnelle :

- Sur **Halo 5**, chaque requête porte `X-LevelUp-Title: halo_5` → le titre est affirmé,
  aucune donnée d'un autre titre ne peut fuiter.
- Sur **Halo Infinite**, les requêtes **n'affirment aucun titre** → le backend retombe sur
  la **session serveur**. Si la session vaut encore `halo_5` (course avec le commit
  `/session/context`, refetch en arrière-plan `refetchOnWindowFocus`/`refetchInterval`, ou
  **un autre onglet ouvert sur Halo 5** — le cookie de session est partagé), Infinite
  **sert du Halo 5**.

Le piège de fausse confiance : la clé `home` inclut pourtant `titleSlug`
([keys.ts:98](apps/web/src/lib/query/keys.ts#L98)), mais la **requête HTTP** sous cette clé
n'affirme pas le titre → la réponse Halo 5 est mise en cache **sous la clé
`['home', slug, 'halo_infinite', locale]`**. Le Spartan ID « vieille v1 Halo 5 » que tu
vois = le champ `spartan_id` backend de Halo 5 servi sous la clé Infinite. Idem pour toutes
les autres clés par-joueur (career, explorer, timeseries, citations, media…) qui, elles,
n'incluent même pas le titre.

### Pourquoi toujours envoyer le header est sûr (vérifié)

- Le registre backend contient `halo_infinite` → `resolveTitleSlug` honore un header
  `X-LevelUp-Title: halo_infinite` explicite (test existant
  [title_test.go:25](apps/go-api/internal/api/middleware/title_test.go#L25)).
- Le handler `/bootstrap` dérive `current_title_slug` de la **session**, PAS du header
  ([bootstrap_service.go:112-114](apps/go-api/internal/service/bootstrap_service.go#L112-L114))
  → la **reprise sur le dernier titre** est préservée même si le bootstrap initial envoie
  `halo_infinite` par défaut.
- Aucune requête par-joueur ne part avant le bootstrap (toutes `enabled: !!playerSlug`,
  playerSlug venant du bootstrap) → pas de fenêtre pré-bootstrap qui affirmerait un mauvais
  titre. API créditée (`credentials: include`) → pas de cache CDN keyé sur le header.

## Correctif (core) : toujours affirmer le titre courant

### 1. `getTitleHeader` — envoyer le header pour TOUS les titres
[apps/web/src/lib/api/client.ts:80-85](apps/web/src/lib/api/client.ts#L80-L85)

```ts
function getTitleHeader(): Record<string, string> {
  // Toujours affirmer le titre courant : sans header, le backend retombe sur la
  // session serveur (partagée entre onglets) → une session périmée sur un autre
  // titre fait fuiter ses données sur le titre affiché. Cf. resolveTitleSlug
  // (header > session > défaut).
  if (_currentTitleSlug) {
    return { 'X-LevelUp-Title': _currentTitleSlug }
  }
  return {}
}
```

Mettre à jour le commentaire de `_currentTitleSlug`
([client.ts:59-63](apps/web/src/lib/api/client.ts#L59-L63)) : retirer la mention
« header pas envoyé pour le défaut (rétrocompatibilité) », remplacée par la raison anti-fuite.

### 2. Corriger la doc désormais fausse (CLAUDE.md : doc == code)
[appShellStore.ts:171-180](apps/web/src/stores/appShellStore.ts#L171-L180) — le commentaire
de `switchTitle` justifie le POST `/session/context` par « pour le défaut le front n'envoie
pas de header → la session fait autorité ». Le POST reste utile (le bootstrap lit la
session), mais reformuler : le header affirme désormais le titre sur chaque requête ; le
commit session sert la persistance/reprise, plus la résolution per-requête.

## Tests

- **Nouveau** `apps/web/src/lib/api/client.test.ts` : spy sur `global.fetch`.
  1. `setApiTitleSlug('halo_infinite')` → une requête `api.get(...)` porte
     `X-LevelUp-Title: halo_infinite` (le cœur de la non-régression).
  2. `setApiTitleSlug('halo_5')` → header `halo_5` (inchangé).
- Suite front : `make check-types` + `make test-web`.
- Go : `cd apps/go-api && go test ./internal/api/...` (non-régression middleware titre —
  le comportement backend ne change pas, on ne fait que l'exercer avec le header explicite).

## Vérification end-to-end (skill `run` / MCP browser)

1. `make dev`. **Cas multi-onglets (repro la plus fiable)** : onglet A sur **Halo 5**,
   onglet B sur **Halo Infinite**. Rafraîchir/naviguer dans l'onglet B → l'accueil Infinite
   doit montrer les données Infinite (Spartan ID, playlists, rangs), **jamais** du Halo 5.
2. **Switch simple** Halo 5 → Halo Infinite : tout bascule sur Infinite, aucun résidu H5.
3. **Reprise de session** : se placer sur Halo 5, recharger l'app (F5) → on revient bien
   sur Halo 5 (bootstrap lit la session, non régressé).
4. Inspecter l'onglet Réseau : chaque requête `/api/v1/...` porte bien `X-LevelUp-Title`
   (y compris `halo_infinite`).

## Secondaire — durcissement structurel (renvoyé au chantier URL, NON traité ici)

Avec le core + le `queryClient.clear()` du switch
([appShellStore.ts:198-199](apps/web/src/stores/appShellStore.ts#L198-L199)), le switch est
correct. Restent des risques latents à traiter avec le slug-dans-l'URL de la semaine
prochaine (ne PAS élargir le patch maintenant, juste consigner) :

- **Query keys par-joueur sans `titleSlug`** (career, matchHistory, explorer, timeseries,
  synthesis, citations, commendationTotals, media, leaderboard, sessionDetail, comparePlayer,
  engagement*, achievements, matchView… — [keys.ts](apps/web/src/lib/query/keys.ts)) :
  collisionnent entre titres, séparées seulement par le `clear()`. Ajouter `titleSlug` à la
  clé = garde structurelle indépendante du timing.
- **`setCurrentTitle`** (setter sans purge de cache,
  [appShellStore.ts:160-163](apps/web/src/stores/appShellStore.ts#L160-L163)) : inutilisé en
  prod (tests only) — à supprimer/sécuriser.
- **Bandeau Spartan synthétisé** (`useCapability('spartan_customizer')` fail-open dans
  [HomeSpartanIdentityBanner.tsx:53](apps/web/src/features/home/HomeSpartanIdentityBanner.tsx#L53)
  et [ExplorerTargetIdentityBanner.tsx:51](apps/web/src/features/explorer/ExplorerTargetIdentityBanner.tsx#L51))
  : vecteur localStorage distinct, difficilement atteignable (nécessite `availableTitles`
  non résolu). Passer ces portes sur un `useCapabilityStrict` (fail-closed) est un
  durcissement propre si le visuel persiste après le core — non central.

## Livraison

- Branche courante `feat/frag-distribution-v2` = travail frags **sans rapport** → créer une
  branche dédiée depuis `main` (ex. `fix/title-header-leak`) avant tout commit (1 tâche = 1
  branche). **Demander avant de committer.**
- Entrée `.ai/thought_log.md` obligatoire avant commit.
- Skill `delivery-checklist` avant « c'est livré ». `make check-types` + `make test-web` +
  `go test ./internal/api/...` verts.

## Statut d'exécution (2026-07-21, branche `fix/title-header-leak` depuis `main`)

**Core**
- `[x]` `getTitleHeader` affirme le header pour TOUS les titres
  ([client.ts:82-90](apps/web/src/lib/api/client.ts#L82-L90)) — `if (_currentTitleSlug)`.
- `[x]` Commentaire `_currentTitleSlug` réécrit (raison anti-fuite, plus « rétrocompat »)
  ([client.ts:59-66](apps/web/src/lib/api/client.ts#L59-L66)).
- `[x]` Commentaire `switchTitle` reformulé
  ([appShellStore.ts:171-180](apps/web/src/stores/appShellStore.ts#L171-L180)). Les DEUX
  sous-commentaires du bloc (étape 1 ET étape 2) affirmaient « pour le défaut pas de
  header » → tous deux corrigés (doc==code, anti-pattern #9). Le POST `/session/context`
  reste requis pour la persistance/reprise (bootstrap lit la session).

**Tests**
- `[x]` `apps/web/src/lib/api/client.test.ts` — 2 tests (défaut `halo_infinite` porte le
  header = cœur non-régression ; `halo_5` inchangé). Vert (spy `globalThis.fetch`,
  `mockRestore` rend la main à MSW).
- `[x]` `make check-types` (tsc `-b`) vert.
- `[x]` `make test-web` : 276 fichiers / **2382 passed** / 14 skipped (+2 = nouveaux tests).
- `[x]` `cd apps/go-api && go test ./internal/api/...` vert (middleware titre non régressé).

**Vérification end-to-end (checks #1-4)**
- `[!]` **Gate visuel utilisateur — non exécuté in-session.** Le MCP `browser` n'est pas
  connecté dans cette session (SDK headless) ; WebFetch ne peut ni s'authentifier ni
  inspecter les headers XHR runtime. Le repro multi-onglets (#1) exige de plus
  l'environnement deux-titres réel + des yeux humains. Étapes détaillées § « Vérification
  end-to-end » ci-dessus. Le mécanisme est prouvé par voie automatisée (test unitaire =
  header affirmé sur la requête sortante pour les 2 titres ; test backend existant = header
  `halo_infinite` explicite honoré). À valider par l'utilisateur.

**Secondaire (durcissement structurel)**
- `[~]` Hors périmètre — NON traité, conforme au plan (renvoyé au chantier « slug dans
  l'URL »). Query keys par-joueur sans `titleSlug`, `setCurrentTitle`, bandeau Spartan
  `useCapabilityStrict` : consignés, intacts.

**Livraison**
- `[x]` Branche `fix/title-header-leak` créée depuis `main` (arbre propre au départ). Le
  plan vivait uniquement sur `feat/frag-distribution-v2` → rapatrié via
  `git checkout <frag> -- <plan>` (path-scoped, sans le code frags).
- `[x]` Entrée `.ai/thought_log.md` [2026-07-21].
- `[x]` Commit initial `71b7c97b3` (front + test + plan + log) posé après accord — non poussé.

## Addendum observabilité + couverture (2026-07-21, hors périmètre initial « header seul »)

Demandé par l'utilisateur après le core. Le fix front n'a pas de surface de log ; la `SlogLogger`
trace déjà `title_slug` par requête (logs/http.log). Trou comblé côté backend :

- `[x]` **Logging** — `resolveTitleSlug` avalait silencieusement un header `X-LevelUp-Title`
  non-vide pointant un titre INCONNU (anti-pattern #10). Ajout `slog.WarnContext` nommant le titre
  demandé ([title.go:62-70](apps/go-api/internal/api/middleware/title.go#L62-L70)) — rare par
  construction, rend une confusion de titre visible dans logs/.
- `[x]` **Tests** (3, via `InjectSession`) — WARN sur header inconnu ; **header bat une session
  divergente** (invariant anti-fuite backend, jusque-là non testé) ; session fait autorité sans
  header ([title_test.go](apps/go-api/internal/api/middleware/title_test.go)).
- `[x]` Gates : `gofmt`/`go vet` clean, `go test ./internal/api/...` vert (middleware 8/8), front
  vert. golangci-lint absent du PATH (gate local `make go-api-lint` = `go vet`).
- `[!]` 2e commit (title.go + title_test.go) — EN ATTENTE de l'accord utilisateur.
  → RÉSOLU : commité `002260798`, mergé via `ee53afd11` (branche `feat/monitoring-lusr-fixes`).

## Addendum revue 2026-07-22 (post-merge)

La revue complète de la branche a réfuté sur pièces l'hypothèse « aucune requête par-joueur ne part
avant le bootstrap » (§ Pourquoi toujours envoyer le header est sûr) : `useFiltersResolve` est
`enabled: !!playerSlug` avec un slug venant de l'URL, donc part AVANT l'hydratation. Le module
valant `'halo_infinite'` au boot, le header affirmait le défaut sur une session `halo_5` (header >
session côté backend) = fuite INVERSE, cachée sous une clé sans titre.

- `[x]` Correctif 2026-07-22 : `_currentTitleSlug: string | null = null` au boot — aucun header
  avant hydratation (la session serveur reste autoritaire, comportement sûr du boot) ; slug affirmé
  sur chaque requête dès `hydrateFromBootstrap`/`switchTitle` (le cœur anti-fuite H5→Infinite est
  préservé). `setApiTitleSlug` accepte `null` (reset tests) ; `getApiTitleSlug()` coalesce vers le
  défaut (contrat share-link inchangé). Test « aucun header avant hydratation » ajouté
  (`client.test.ts`). Le durcissement structurel (titleSlug dans les query keys) reste renvoyé au
  chantier « slug dans l'URL », inchangé.
