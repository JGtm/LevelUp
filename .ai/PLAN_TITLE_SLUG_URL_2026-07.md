# PLAN — D7 : titre (et langue) dans l'URL (segments de route)

Statut : PLANIFIE v2 (aucune ligne de code écrite).
Date : v1 2026-07-13 (architecte Opus) ; **v2 2026-07-21** (revue Fable : 4 trous corrigés,
langue intégrée structurellement, décisions D-8..D-11 ajoutées — amendements validés par
l'utilisateur le 2026-07-21).
Principe APPROUVE par l'utilisateur (v1 le 2026-07-13, amendements v2 le 2026-07-21).

Branche cible d'implémentation : **`feat/title-slug-in-url`** (depuis `main` à jour).
Pré-requis : la branche en cours (`feat/frag-distribution-v2`) est livrée ou committée —
ne pas mélanger les chantiers.

> Contrat d'exécution : ce plan s'exécute sous le skill **`plan-execution`** (ordre strict,
> une étape close avant la suivante, aucun report d'action exécutable, statut sur chaque
> item, zéro fix hors périmètre). En cas de divergence, le présent plan fait foi ; à défaut
> le skill est le défaut. Avant de finaliser toute modification du plan : skill
> **`plan-review`**. Avant chaque commit : skill **`delivery-checklist`**.

---

## 1. Objectif et critères de succès (mesurables)

**Objectif.** Faire du titre actif (`halo_infinite` | `halo_5`, extensible) un état EXPLICITE
porté par l'URL, et non plus un état implicite résolu au bootstrap et porté par le client
API. L'URL devient la **source de vérité** du titre pour toutes les pages title-scoped ; le
store et le client API SUIVENT le segment d'URL (inversion de contrôle). La **structure**
d'URL accueille aussi la langue (segment optionnel) : le déplacement de routes se fait UNE
seule fois vers la forme finale ; la logique locale est livrée dans une phase dédiée du même
chantier (D-4).

**Pourquoi.** Aujourd'hui le titre est implicite : résolu au bootstrap
(`hydrateFromBootstrap`, `stores/appShellStore.ts:114-153`), mémorisé dans un module-level du
client API (`lib/api/client.ts:64` `_currentTitleSlug`), transporté par un header
(`client.ts:80-85` `getTitleHeader`, envoyé UNIQUEMENT si titre non-défaut), jamais présent
dans l'URL. Conséquences : URLs non auto-porteuses, et une **classe de bugs « titre
implicite »** dont la fuite deep-link `?f=` (PR #59, commit `0b2e5cdb8`) est le symptôme le
plus récent. Mettre le titre dans l'URL supprime la cause racine : au fresh-load, le titre
est connu AVANT le bootstrap, de façon synchrone, par lecture du segment — **à condition de
câbler ce chemin synchrone explicitement (Phase 1, trou n°2 de la revue v2)**.

**Critères de succès (tous vérifiables) :**

1. Toute page title-scoped est servie sous `/t/{slug}/…` (forme complète
   `/{-lang}/t/{slug}/players/{playerSlug}/…`) ; l'URL affichée porte le titre actif sur les
   deux titres.
2. `make check-types` = 0 — juge d'exhaustivité des littéraux TYPES uniquement (§2 : liste
   des surfaces qui y échappent → critère 3).
3. **Garde-rail grep vert** : plus aucun littéral `/players/` string (hors allowlist datée :
   splat redirect, regex legacy volontaires, module `title-routing`) — complément
   obligatoire du typecheck.
4. `make test-web` (vitest run) vert, y compris les tests PR #59
   (`createFilterStore.test.ts`, `appShellStore.switchTitle.test.ts`) et les nouveaux tests
   TDD du module `title-routing`.
5. `cd apps/web && npm run lint` = 0.
6. Toute ancienne URL `/players/{playerSlug}/…` redirige (`replace`) vers l'équivalent
   title-préfixé en préservant suffixe + `search` (`?f=`) + hash — **aucun lien mort** ; la
   matrice de redirection est couverte par des TESTS UNITAIRES table-driven (pas seulement
   une vérification navigateur).
7. Le changement de titre via `TitleSwitcher` change le SEGMENT d'URL ; données, filtres
   (reset) et cache basculent ; aucune fuite inter-titres ; l'échec de bascule ne crée PAS
   de divergence URL↔store (D-6 chemin d'erreur).
8. Un slug d'URL inconnu / `coming_soon` / `archived` mène à un gate propre (parité
   `RequireActiveTitle`, `require_active_title.go:48-81`), jamais une page blanche ; les
   strings du gate existent en FR **et** EN.
9. Le header `X-LevelUp-Title` est envoyé sur TOUTES les requêtes, pour TOUS les titres
   (title-agnostic, D-9) — plus de cas spécial `halo_infinite`.
10. Les specs Playwright `e2e/` sont migrées vers les nouvelles URLs + une spec dédiée
    vérifie la redirection legacy (leurs assertions d'URL actuelles casseraient sinon —
    `ascension-2tabs.spec.ts:64` etc.).
11. Segment langue : `/{lang}/t/{slug}/…` force la locale ; absence de segment = locale
    session/bootstrap (comportement actuel inchangé).
12. Non-régression PR #59 : deep-link legacy `?f=` (enveloppe v2 ou legacy) honoré / rejeté
    correctement selon le titre.
13. Vérification NAVIGATEUR (chrome-devtools) sur les 2 titres + redirections + segment
    langue, consignée au journal du plan.

---

## 2. Constat sur pièces — état actuel (re-vérifié le 2026-07-21)

- **Routing file-based, aucun segment titre.** `apps/web/src/routes/players/$playerSlug/…`
  (≈ 50 fichiers de route). `routeTree.gen.ts` généré — ne jamais l'éditer.
  Inventaire : **~70 fichiers / 186 occurrences** de `players/$playerSlug` hors
  `routeTree.gen.ts`.

- **Bootstrap = COMPOSANT, pas routeur (fondement de D-8).** `routes/__root.tsx:34-49` :
  `useQuery` bootstrap dans `RootLayout`, blocage par rendu conditionnel (`:86-124`). Il n'y
  a AUCUN `beforeLoad` root. Conséquence : un `beforeLoad` enfant s'exécute au matching,
  AVANT hydratation du store (`availableTitles` vide, `currentTitleSlug` = défaut). Toute
  validation/redirection dépendant du store doit être **déclarative dans le composant**,
  gatée sur `isBootstrapped` — pattern maison déjà éprouvé : `routes/index.tsx:38-50`
  (`<Navigate>` déclaratif) + résolveur pur `resolveIndexRedirect`
  (`shellNavigation.ts:123-136`).

- **Titre résolu au bootstrap puis porté par le client API.** `appShellStore.ts:114-116`
  (`hydrateFromBootstrap` → `setApiTitleSlug`), `client.ts:64-85` (module-level +
  `getTitleHeader` avec cas spécial `halo_infinite` = header non envoyé). Ce cas spécial est
  la cause de la contrainte d'ordre « POST session avant refetch » commentée dans
  `switchTitle` (`appShellStore.ts:171-177`) — supprimé par D-9.

- **Switch de titre piloté par un bouton.** `appShellStore.ts:165-218` `switchTitle()` ;
  `TitleSwitcher.tsx:40-58` navigue APRÈS le switch. Rollback store-only en cas d'échec
  (`:211-214`) — incompatible tel quel avec une URL source de vérité (divergence URL↔store,
  risque de boucle de réconciliation) → chemin d'erreur redéfini en D-6.

- **Backend déjà header/session-driven — aucun changement Go.**
  `middleware/title.go:55-72` : header `X-LevelUp-Title` accepté pour TOUT slug existant au
  registre (y compris `halo_infinite`) > session > défaut. `RequireActiveTitle` gate 503.
  D-9 (header systématique) est donc rétro-compatible sans toucher au Go. **D7 reste un
  chantier FRONTEND.**

- **Surfaces qui ÉCHAPPENT au typecheck (fondement du garde-rail, critère 3).** Le plan v1
  surestimait « le typecheck énumère l'exhaustif ». Échappent à tsc :
  - `shellNavigation.ts:2` — `ShellNavItem.to: string` (toute la nav L1/L2). Preuve : le
    mismatch `/players/$playerSlug/profile/citations` (`:48`, route réelle
    `career/citations`) vit depuis des mois sans erreur tsc.
  - `buildPlayerDestination` (`shellNavigation.ts:138-158`) — construit des strings.
  - `isCommunityPath` (`shellNavigation.ts:80-89`) — regex sur `pathname`.
  - `lib/pageTitle.ts` — patterns string.
  - `lib/page-scope/usePageScope.ts:32-35` — `to: string` + `params` non typés.
  - `e2e/*.spec.ts` (10 fichiers) — `page.goto('/players/…')` + assertions
    `expect(page.url()).toMatch(…)` (`ascension-2tabs.spec.ts:64`) → CASSENT après D7.

- **Garde deep-link `?f=` (PR #59).** `createFilterStore.ts:182-231` (enveloppe v2
  `encodeToUrl`/`decodeFromUrl`), `:298-309` `reconcileActiveTitle`, `:430-437`
  `onRehydrateStorage`. `decodeFromUrl` compare à `getApiTitleSlug()` → le câblage synchrone
  segment→client (Phase 1) rend ce décodage correct dès la réhydratation.

- **Locale.** Header `X-LevelUp-Locale` systématique (`client.ts:92-101`), normalisation
  serveur `title.go:37-44`. Un SEUL caller UI de `setLocale` :
  `features/settings/queries.ts:41-70` (mutation settings). Les query keys portent déjà
  `locale` là où les payloads sont localisés (`lib/query/keys.ts:98`).

- **Params optionnels TanStack.** `@tanstack/react-router` 1.170.16 installé ; router-core
  documente les segments `{-$optional}` (`node_modules/@tanstack/router-core/dist/esm/
  path.d.ts:53`). Support file-based à sanity-checker en Phase 0 (gate factuel, repli
  tranché en D-4).

---

## 3. Décisions PRÉ-TRANCHÉES (fermes — ne pas re-débattre en cours d'exécution)

### D-1 — Schéma d'URL final : `/{-$lang}/t/{titleSlug}/players/{playerSlug}/…`

Le titre est un segment préfixe sous le namespace `/t/`, au-dessus du joueur ; la langue est
un segment OPTIONNEL au-dessus du titre. Exemples : `/t/halo_infinite/players/jgtm/home`
(locale = session, comportement actuel), `/en/t/halo_5/players/jgtm/stats/timeseries`
(locale forcée EN).

- Hiérarchie : la locale est globale-app > le titre est un scope de données > le joueur
  appartient au titre (ADR 0008).
- Namespace `/t/` : évite la collision slug↔routes racine (`/admin`, `/settings`).
- Le segment est un SÉLECTEUR, pas un branchement de logique : le code métier reste branché
  capabilities (jamais `slug == "..."`). Le segment n'alimente que le client API (header) et
  la validation front.
- Langue optionnelle (`{-$lang}`) : `/t/…` reste valide pour toujours → **aucune deuxième
  génération d'URLs legacy** (c'était le coût caché du découpage v1 en deux chantiers).

### D-2 — Valeurs des segments = identifiants INTERNES verbatim

Titres : `halo_infinite`, `halo_5` (underscore, pas d'alias hyphénisé — pas de table de
mapping). Langues : `fr`, `en` (type `Locale` existant). Alias cosmétiques = chantier
ultérieur distinct, hors D7.

### D-3 — Pages title-agnostiques : HORS segments (racine, inchangées)

`admin/*`, `settings`, `setup`, `login`, `register`, `changelog`, `help`, `groups`, `join`,
`onboarding.openspartan`, `lab/*`, index `/` : sans segment titre NI langue. Leur titre
actif = session serveur (le flux title-scoped commit `POST /session/context` à chaque
bascule). Sélecteurs in-page des pages admin : non modifiés par D7 (toute dépendance cassée
= Découverte).

### D-4 — Langue : structure DANS D7, logique en phase dédiée (amende la v1)

L'utilisateur a CONFIRME (2026-07-21) que la langue ira dans l'URL. Reporter la structure
aurait coûté un deuxième déplacement de ~50 routes + une deuxième passe sur ~70 fichiers de
littéraux + des redirections en chaîne. Décision :

- Le déplacement structurel (Phase 2) vise directement la forme finale D-1
  (`routes/{-$lang}/t/$titleSlug/players/…`).
- La LOGIQUE locale (réconciliation locale←segment) est livrée en Phase 5, APRÈS que le
  flux titre est vert — même mécanique factorisée (D-10), périmètre locale minuscule
  (un seul caller `setLocale`).
- **Repli tranché** si le sanity-check Phase 0 révèle un support file-based `{-$lang}`
  cassé : segment `$lang` OBLIGATOIRE (`/fr/t/…`), les redirections legacy visant
  `/{localeCourante}/t/…`. Pas d'autre alternative à explorer en cours de route.

### D-5 — Redirections legacy : splat DÉCLARATIF, préservation intégrale

`routes/players/**` est physiquement DÉPLACÉ (git mv) vers la forme finale. La racine
`routes/players/` libérée héberge un splat (`routes/players/$.tsx`) dont le **composant**
(PAS un `beforeLoad` — D-8) rend un `<Navigate replace>` calculé par la fonction pure
`buildLegacyRedirect(pathname, search, hash, activeSlug)` du module `title-routing`, gaté
sur `isBootstrapped` (état `wait` → null, `__root` affiche déjà le loader). Préservation
suffixe + `search` (`?f=`) + hash. Les redirections internes existantes
(objectifs→ascension, palmares→community, compare→community) sont re-pointées vers les
cibles title-préfixées. L'index `/` redirige vers `/t/{activeSlug}/players/{player}/home`.

### D-6 — Inversion de contrôle : layout `t/$titleSlug` déclaratif + chemin d'erreur défini

Layout `routes/{-$lang}/t/$titleSlug.tsx`, parent du sous-arbre joueur. Mécanisme
DÉCLARATIF (D-8) : résolveur pur `resolveTitleGate(slug, availableTitles, isBootstrapped)`
→ `wait | valid | unknown | coming_soon | archived`, projeté par le composant :

1. `wait` → null (loader `__root` déjà affiché) ;
2. `unknown`/`coming_soon`/`archived` → gate (redirection titre actif OU écran
   d'indisponibilité, strings FR+EN) — parité `RequireActiveTitle` ;
3. `valid` + `titleSlug !== currentTitleSlug` → `applyActiveTitle(slug)` (extraction de
   `switchTitle`) : `setApiTitleSlug` + `set(currentTitleSlug)` + `POST /session/context` +
   reset filtres + `cancelQueries`/`clear()` + re-bootstrap + filet joueur.

**Chemin d'erreur (nouveau, ferme)** : en échec d'`applyActiveTitle`, PAS de rollback
store-only (divergence URL↔store → boucle de réconciliation). À la place : `navigate`
retour vers le segment précédent + message d'erreur (toast/inline). **Course
back/forward** : pendant un `applyActiveTitle` en vol (`isTitleSwitching`), le layout ne
déclenche PAS de seconde bascule ; à la complétion, il re-compare segment↔store et re-boucle
si divergence (convergence garantie, testée).

### D-7 — Articulation `?f=` : segment = source de vérité, garde PR #59 conservée

Inchangé v1 : la garde PR #59 (enveloppe v2 + `reconcileActiveTitle`) reste INTACTE en
défense en profondeur + rétro-compat des `?f=` partagés. Ne PAS la refactorer dans D7.
Commentaire de code documentant la coexistence segment↔`?f=` et le critère de retrait.

### D-8 — Mécanisme de résolution : DÉCLARATIF composant, jamais `beforeLoad` dépendant du store (nouveau)

Le bootstrap étant composant-level (`__root.tsx:34-49`, §2), tout `beforeLoad` lisant
`appShellStore` verrait un store non hydraté (trou n°1 de la revue v2 : un bookmark H5
legacy serait redirigé vers `halo_infinite`). Règle ferme : validation de segment,
redirection legacy et réconciliation = **résolveurs purs + `<Navigate>`/effets déclaratifs
dans les composants, gatés sur `isBootstrapped`** (pattern `resolveIndexRedirect`).
Déplacer le bootstrap dans le router context = refactor hors de proportion, écarté.

### D-9 — Title-agnostic intégral : header `X-LevelUp-Title` TOUJOURS envoyé (nouveau)

Demande explicite utilisateur (2026-07-21) : tous les jeux traités PAREIL, aucun cas
spécial pour le titre par défaut.

- `getTitleHeader()` (`client.ts:80-85`) : envoyer le header dès que `_currentTitleSlug`
  est non vide — suppression du `!== 'halo_infinite'`. Vérifié compatible backend sans
  changement Go (`title.go:55-61` accepte tout slug existant).
- Câblage synchrone au boot (trou n°2) : au point d'entrée de l'app, AVANT la première
  requête, parser `location.pathname` (fonction pure du module D-10) ; si segment titre
  présent → `setApiTitleSlug(segment)` ; si segment langue présent → `setApiLocale`. Aucun
  test sur la VALEUR du slug (pas de « si halo_infinite alors… ») : le segment est posé
  verbatim, le backend valide contre son registre.
- Le `POST /session/context` reste (il tient la session à jour pour les pages agnostiques,
  D-3) mais n'est PLUS sur le chemin critique de l'ordre des refetch (le header fait foi
  partout) — simplifier le commentaire d'ordonnancement de `switchTitle` en conséquence.
- MAJ de la doc du header dans `client.ts` (doc inversée interdite, anti-pattern n°9).

### D-10 — Factorisation : module unique `lib/title-routing/` + garde-rail (nouveau)

Toute interprétation de segment vit dans UN module `apps/web/src/lib/title-routing/` :

- `parseRouteSegments(pathname)` → `{ lang?: Locale, titleSlug?: string }` — pur ;
- `resolveTitleGate(slug, availableTitles, isBootstrapped)` — pur (D-6) ;
- `buildLegacyRedirect(pathname, search, hash, activeSlug)` — pur (D-5) ;
- `applyActiveTitle(slug)` — l'unique fonction effectful (extraite de `switchTitle`).

Garde-rail (règle CLAUDE.md n°6) : test vitest « ratchet » qui scanne `apps/web/src` et
interdit tout littéral string `/players/` et tout parsing de segment `/t/` hors allowlist
EXPLICITE datée (module `title-routing`, splat `routes/players/$.tsx`, regex legacy
volontaires type `isCommunityPath`). C'est le critère de succès n°3.

### D-11 — TDD ciblé : tests d'abord sur la logique, tsc sur le mécanique (nouveau)

Les tests unitaires du module `title-routing` (parse, gate, matrice de redirection
complète en table-driven) et d'`applyActiveTitle` s'écrivent AVANT leur implémentation
(Phase 1). La matrice de redirection §Phase 3 est d'abord une TABLE DE TESTS, la
vérification navigateur ne fait que confirmer. Le déplacement mécanique des ~70 fichiers de
littéraux typés n'est pas TDD-isé : tsc est le harnais (complété par le garde-rail D-10
pour les strings).

### D-12 — Locale par segment : périmètre minimal (nouveau)

Segment `lang` présent → il FORCE la locale : `setLocale(lang)` + invalidation des queries
localisées (PAS de `queryClient.clear()` complet — les keys portent déjà `locale`,
`keys.ts:98`). Segment absent → locale session/bootstrap, comportement actuel strictement
inchangé. Le sélecteur de langue (settings) met à jour le segment s'il est présent dans
l'URL courante, sinon comportement actuel. Pas de redirection legacy côté langue (le
segment est optionnel, il n'y a pas d'« ancienne URL » de langue).

---

## 4. Périmètre

**Dans le périmètre (frontend `apps/web`) :** module `lib/title-routing/` + tests TDD +
garde-rail ; câblage synchrone au boot + header systématique (`client.ts`) ; structure de
routes finale (`routes/{-$lang}/t/$titleSlug/players/…`) ; layout titre déclaratif ;
migration de tous les littéraux (typés ET strings échappées §2) ; helpers centralisés
(`shellNavigation` — y compris typage de `ShellNavItem.to`, `buildPlayerDestination`,
`pageTitle`, `isCommunityPath`, `usePageScope` callers) ; redirections legacy ;
`switchTitle`/`TitleSwitcher` ; chemin d'erreur + course back/forward ; logique locale par
segment ; strings i18n FR+EN du gate ; tests vitest ; migration des specs Playwright + spec
de redirection ; vérification navigateur.

**Hors périmètre (noter en Découvertes si rencontré) :** refactor/suppression garde `?f=`
PR #59 (D-7) ; tout changement Go (vérifié inutile, D-9/§2) ; sélecteurs in-page admin
(D-3) ; alias cosmétiques de slugs (D-2) ; segment langue sur les pages agnostiques (D-3) ;
dette lint pré-existante (baseline gelée).

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Rappel plan-execution : clôture d'étape = gate passé (commandes exactes, sorties propres)
> + tous les items statués `[x]`/`[~]`/`[!]` + plan mis à jour + entrée thought_log + point
> d'étape utilisateur. Aucune case vide à la clôture. Zéro fix hors périmètre.
> Note tests : vitest `apps/web` tourne HORS sandbox → `dangerouslyDisableSandbox=true`
> (mémoire projet `reference_vitest_outside_sandbox`).

### Phase 0 — Cadrage, baseline, sanity-check `{-$lang}` (rapide)

- [ ] Créer `feat/title-slug-in-url` depuis `main` à jour (`git fetch` + vérifier
      `git log --oneline -1 origin/main`).
- [ ] Baselines au journal : (a) `grep -rn "players/\$playerSlug" apps/web/src
      --include=*.tsx --include=*.ts | wc -l` (attendu ≈ 186 hors routeTree) ; (b)
      inventaire des strings échappées (grep `"/players/` + liste §2) ; (c) `grep -rn
      "/players/" apps/web/e2e` (specs à migrer, Phase 6).
- [ ] **Sanity-check `{-$lang}` file-based** : créer `routes/{-$lang}/t/$titleSlug.tsx`
      minimal (layout `<Outlet/>`), régénérer `routeTree.gen.ts` (dev ou build), vérifier
      que tsc accepte `navigate({to: '/t/$titleSlug', params})` SANS lang et AVEC lang. Si
      cassé → appliquer le repli D-4 (segment `$lang` obligatoire) et le consigner.
- [ ] Relire les décisions §3 (fermes). Ne rien re-décider.

Gate Phase 0 : `git branch --show-current` = `feat/title-slug-in-url` ; baselines
consignées ; verdict `{-$lang}` consigné (support OK ou repli D-4 acté).

### Phase 1 — Module `title-routing` en TDD + câblage synchrone + header agnostic (moyen)

> Ordre TDD (D-11) : 1a AVANT 1b. Aucun déplacement de route dans cette phase — tout est
> pur/testable sans toucher au routing.

- [ ] **1a. Tests d'abord** (rouges) : `lib/title-routing/*.test.ts` —
      `parseRouteSegments` (avec/sans lang, avec/sans titre, chemins agnostiques, cas
      tordus `/t/`, `/t/x`, trailing slash) ; `resolveTitleGate` (wait/valid/unknown/
      coming_soon/archived) ; `buildLegacyRedirect` en TABLE-DRIVEN — la matrice complète :
      `home`, `stats/{timeseries,sessions,synthesis,index}`, `career/*`, `squad/*`,
      `community/*`, `media`, `explorer`, `matches/$matchId(/replay)`, `ascension/*`,
      `notifications`, `citations`/`commendations`, legacy `palmares/*` + `objectifs` +
      `compare` + `synthesis`, chacun avec préservation `?f=` + hash ; bare `/players` et
      `/players/{slug}` sans suffixe → `/t/{active}/players/{slug}/home`.
- [ ] **1b.** Implémenter le module (D-10) jusqu'à tests verts. `applyActiveTitle(slug)` :
      extraction de `switchTitle` (`appShellStore.ts:165-218`), testée (adapter/étendre
      `appShellStore.switchTitle.test.ts` — ne pas affaiblir les assertions PR #59).
- [ ] **1c.** Câblage synchrone au boot (D-9) : au point d'entrée (localiser sur pièces :
      `main.tsx` / init du router), `parseRouteSegments(location.pathname)` →
      `setApiTitleSlug` / `setApiLocale` si segments présents, AVANT la première requête.
      Test unitaire du helper d'init (fonction pure appelée par le point d'entrée).
- [ ] **1d.** `client.ts` : `getTitleHeader` envoie TOUJOURS le header (suppression du cas
      spécial `halo_infinite`) ; MAJ des commentaires (`client.ts:59-63`, ordonnancement
      dans `switchTitle`/`applyActiveTitle`) ; adapter les mocks/tests impactés.
- [ ] **1e.** Garde-rail ratchet (D-10) : test vitest scannant `apps/web/src` — à ce stade
      il documente l'allowlist BASELINE (les ~70 fichiers pas encore migrés y figurent en
      liste à faire fondre) OU il est écrit pour ne s'armer qu'en Phase 2 ; choisir la
      forme la plus simple et la consigner.

Gate Phase 1 : `make test-web` vert (nouveaux tests inclus) ; `make check-types` = 0 ;
`npm run lint` = 0 ; smoke navigateur : l'app fonctionne À L'IDENTIQUE (aucune route
déplacée), header `X-LevelUp-Title` visible en network sur les requêtes du titre PAR DÉFAUT
aussi (preuve D-9).

### Phase 2 — Déplacement structurel + littéraux + layout (LOURD — cœur du chantier)

- [ ] **2a.** `git mv routes/players/** → routes/{-$lang}/t/$titleSlug/players/**` (forme
      Phase 0). Compléter le layout `t/$titleSlug.tsx` : projection de `resolveTitleGate`
      (D-6, D-8) + `applyActiveTitle` sur divergence + écran d'indisponibilité (strings
      FR **et** EN dans le manifest i18n approprié).
- [ ] **2b.** Régénérer `routeTree.gen.ts`, corriger CHAQUE littéral typé cassé jusqu'à
      `tsc -b` = 0 (Link `to=`, `navigate({to})`, `redirect({to})`, `Route.useParams()`).
- [ ] **2c.** Surfaces échappant au typecheck (§2) : typer `ShellNavItem.to` avec le type
      de route du routeur (fait entrer la nav L1/L2 dans tsc — le mismatch
      `profile/citations` `shellNavigation.ts:48` sera FORCÉMENT corrigé vers la route
      réelle `career/citations` : correction mécanique in-périmètre, la consigner) ;
      `buildPlayerDestination` (préfixe titre+lang) ; `isCommunityPath` ; `pageTitle.ts` ;
      callers `usePageScope`.
- [ ] **2d.** Re-cibler l'index `/` (`routes/index.tsx`) et le guard joueur
      (ex-`players/$playerSlug.tsx:16-50`) ; **joueur inconnu sur le titre cible** : après
      réconciliation/re-bootstrap, si `playerSlug` absent de `availablePlayers` → premier
      joueur disponible (même titre) sinon index — avec test.
- [ ] **2e.** Armer le garde-rail (1e) : allowlist réduite à sa forme finale datée (module,
      splat — créé Phase 3 : l'y inscrire par avance —, regex legacy volontaires).
- [ ] **2f.** Adapter les tests de route/nav cassés ; tests du layout titre (wait → null ;
      valide → rendu ; divergence → bascule ; inconnu/coming_soon → gate).
- [ ] **2g.** Vérifier que les Links conservent le segment lang courant quand il est
      présent (héritage des params TanStack sur param optionnel) — test ou vérification
      consignée ; si non hérité, corriger via les helpers centralisés (pas au cas par cas).

Gate Phase 2 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 ;
garde-rail vert ; smoke navigateur : `/t/halo_infinite/players/{p}/home` ET
`/t/halo_5/players/{p}/home` rendent la bonne page (header cohérent en network).

### Phase 3 — Redirections legacy (moyen)

- [ ] **3a.** Splat `routes/players/$.tsx` DÉCLARATIF (D-5, D-8) : composant gaté
      `isBootstrapped`, `<Navigate replace>` calculé par `buildLegacyRedirect` (déjà
      testée en 1a — le splat n'est qu'une projection).
- [ ] **3b.** Re-pointer les redirections internes (objectifs→ascension,
      palmares→community, compare→community) vers les cibles title-préfixées.
- [ ] **3c.** Confirmer la matrice par les tests 1a (déjà verts) + navigateur : 4+
      anciennes URLs (dont une avec `?f=`, dont une H5 — le fresh-load H5 doit rediriger
      vers `/t/halo_5/…` grâce au gate `isBootstrapped`, PAS vers le défaut).

Gate Phase 3 : `make check-types` = 0 ; `make test-web` vert ; navigateur : redirections
vérifiées, suffixe + `?f=` + hash préservés, aucune page morte.

### Phase 4 — `switchTitle` réduit, chemin d'erreur, courses (moyen)

- [ ] **4a.** `TitleSwitcher.onSelect` → simple `navigate` vers
      `/t/{slug}/players/{player}/home` (le layout fait la bascule) ; réduire/supprimer
      `switchTitle` bouton ; purge du code mort (POST+rebootstrap+navigate dupliqués) et de
      ses tests obsolètes — règle « 0 code mort ».
- [ ] **4b.** Chemin d'erreur D-6 : échec `applyActiveTitle` → navigate retour segment
      précédent + message ; test (mock POST en échec → URL revenue, pas de boucle).
- [ ] **4c.** Course back/forward : test popstate pendant bascule en vol → convergence
      (re-comparaison à la complétion), pas de double bascule.
- [ ] **4d.** Articulation `?f=` (D-7) : garde PR #59 intacte ; commentaire de coexistence
      + critère de retrait ; tests PR #59 passent inchangés OU adaptés (justification au
      journal, assertions non affaiblies).

Gate Phase 4 : `make check-types` = 0 ; `make test-web` vert (incl. PR #59) ; grep
confirmant l'absence du code de switch dupliqué ; navigateur : switch via sélecteur →
segment change, données/filtres basculent ; échec simulé → retour propre.

### Phase 5 — Locale par segment (rapide)

- [ ] **5a.** Réconciliation locale←segment (D-12) dans le layout : `lang` présent et ≠
      locale courante → `setLocale` + invalidation des queries localisées ; absent →
      no-op strict. Tests.
- [ ] **5b.** Sélecteur de langue (`features/settings/queries.ts:41-70`) : si un segment
      lang est présent dans l'URL courante, le mettre à jour (navigate replace) ; sinon
      comportement actuel. Test.
- [ ] **5c.** Navigateur : `/en/t/halo_infinite/players/{p}/home` force EN (payloads +
      header `X-LevelUp-Locale`) ; retour sans segment → locale session.

Gate Phase 5 : `make check-types` = 0 ; `make test-web` vert ; vérification navigateur
consignée.

### Phase 6 — E2E + vérification navigateur complète + livraison (moyen)

- [ ] **6a.** Migrer les 10 specs `e2e/*.spec.ts` vers les URLs `/t/…` (leurs assertions
      d'URL casseraient sinon — critère 10) ; ajouter UNE spec `legacy-redirect.spec.ts`
      exerçant la matrice principale (bookmark, `?f=`, H5).
- [ ] **6b.** chrome-devtools, 2 titres : home / stats / career / squad / community /
      explorer / un match — URL `/t/{slug}/`, header network, données du bon titre.
- [ ] **6c.** Scénario fuite PR #59 : filtre session sur un titre, switch, vérifier reset ;
      deep-link `?f=` legacy honoré/rejeté correctement.
- [ ] **6d.** Segment invalide `/t/inconnu/players/…` + titre `coming_soon` si dispo →
      gate propre, pas de page blanche, strings FR/EN.
- [ ] **6e.** Suite complète : `make check-types`, `make test-web`, `npm run lint`,
      `npm run test:e2e` (nécessite `make dev`).
- [ ] **6f.** `thought_log` (clôture), MAJ `project_map` (la cartographie des routes
      change), statut final de chaque item du plan, delivery-checklist.

Gate Phase 6 : tous les gates verts ; captures/observations consignées au journal.

---

## 6. Protocole de reprise de session

1. Relire le contrat `plan-execution` puis ce plan (§3 décisions fermes, §5 phases).
2. Le fichier plan est la source de vérité de l'avancement : reprendre à la première case
   non statuée de l'étape courante (celle dont le gate n'est pas coché).
3. Vérifier `git branch --show-current` = `feat/title-slug-in-url` ; `git log --oneline
   -10` pour situer les commits d'étape.
4. Ne pas re-décider ce qui est tranché en §3 (D-1..D-12). Toute nouvelle décision =
   signalée immédiatement à l'utilisateur.
5. Vérifier sur pièces avant de coder (rouvrir les fichier:ligne du §2 — re-vérifiés le
   2026-07-21, mais le code bouge).

---

## 7. Découvertes (à remplir pendant l'exécution — ne PAS traiter hors périmètre)

> Format : `[YYYY-MM-DD] découverte — décision (différé / signalé / bloquant traité)`.

- (aucune à ce jour)

Points de vigilance connus :
- Le mismatch `shellNavigation.ts:48` (`profile/citations` vs route réelle
  `career/citations`) sera corrigé MÉCANIQUEMENT par le typage de `ShellNavItem.to` (2c) —
  ce n'est plus une Découverte, c'est un effet attendu ; le consigner au journal.
- Héritage du param optionnel `{-$lang}` par les Links (2g) : comportement TanStack à
  vérifier sur pièces ; corriger via helpers centralisés uniquement.
- `refetchOnWindowFocus: true` sur le bootstrap (`__root.tsx:42`) : un refocus pendant une
  bascule en vol re-hydrate le store — la convergence D-6 (re-comparaison segment↔store)
  doit couvrir ce cas ; si un test le montre fragile → Découverte, pas de patch ad hoc.

---

## 8. Effort estimé

| Phase | Charge | Nature |
|---|---|---|
| 0 — Cadrage + sanity `{-$lang}` | Rapide | branche, baselines, verdict optionnel param |
| 1 — Module TDD + boot synchrone + header | Moyen | pur/testable, zéro déplacement de route |
| 2 — Déplacement structurel + littéraux + layout | **LOURD** | ~50 routes déplacées, ~70 fichiers de littéraux, échappées typecheck, garde-rail |
| 3 — Redirections legacy | Rapide/Moyen | projection déclarative d'une fonction déjà testée |
| 4 — switchTitle réduit + erreurs/courses | Moyen | rewire sélecteur, purge, chemins d'erreur |
| 5 — Locale par segment | Rapide | périmètre minuscule (D-12) |
| 6 — E2E + navigateur + livraison | Moyen | migration specs + spec redirect + matrice navigateur |

**Total : chantier LOURD** (2 sessions denses probables). Le risque de la Phase 2 est borné
par tsc + garde-rail grep ; la logique risquée (gate, redirect, bascule) est testée AVANT le
déplacement (Phase 1, TDD). Backend non touché.

---

## 9. Conformité plan-review (auto-contrôle)

- Objectif + critères mesurables : §1 (13 critères vérifiables).
- Phases ordonnées, périmètre FERMÉ par étape : §5 (items cochables, pas de « etc. »).
- Gates par commandes exactes : §5 (`make check-types`, `make test-web`, `npm run lint`,
  garde-rail vitest, `npm run test:e2e`, chrome-devtools).
- Statuts `[x]`/`[~]`/`[!]` + aucune case vide à la clôture : §5 (rappel).
- Ordre strict N close avant N+1 : §5 (rappel).
- Fixes hors périmètre interdits + section Découvertes : §4, §7.
- Décisions produit TRANCHÉES avant exécution : §3 (D-1..D-12, y compris replis — aucun
  « à décider en cours de route »).
- Protocole de reprise : §6.
- Branche cible nommée : `feat/title-slug-in-url`.
- Renvoi explicite `plan-execution` : en-tête.
- i18n FR+EN des nouvelles strings UI : critère 8, items 2a/6d.
- Vérification navigateur 2 titres + redirections + langue : Phases 3-6.

---

## 10. Journal du plan (rempli à l'exécution)

- (vide — à alimenter à chaque clôture d'étape : date, étape, gate, statut des items,
  observations navigateur.)
