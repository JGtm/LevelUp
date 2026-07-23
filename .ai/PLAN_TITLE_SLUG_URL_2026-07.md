# PLAN — D7 : titre (et langue) dans l'URL (segments de route)

Statut : EN COURS D'EXECUTION (Phase 0 close le 2026-07-22).
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

- [x] Créer `feat/title-slug-in-url` — DÉVIATION consignée : créée depuis
      `refactor/ascension-ux-2026-07` (demande utilisateur du 2026-07-22 : cette branche
      contient l'intégralité d'`origin/main` — vérifié `git log HEAD..origin/main` vide —
      PLUS les correctifs Ascension non encore mergés ; partir de main re-créerait les
      mêmes warnings/erreurs déjà réglés).
- [x] Baselines au journal : (a) 192 occurrences / 70 fichiers hors routeTree (attendu
      ≈186 — écart +6 dû au chantier Ascension postérieur à la v2 du plan) ; (b) 131
      strings `/players/` échappées (hors occurrences typées) ; (c) e2e : 26 fichiers
      contiennent `/players/` (25 specs + `_helpers/demoData.ts`) — `ascension-2tabs.spec.ts`
      cité §2 a été SUPPRIMÉ entre-temps (mini-lot F6 Ascension).
- [x] **Sanity-check `{-$lang}` file-based** : SUPPORT OK — route
      `routes/{-$lang}/t/$titleSlug.tsx` créée, routeTree régénéré (vite build),
      `tsc -b` = 0 avec Link/navigate SANS lang et AVEC lang. Repli D-4 NON nécessaire.
      Précision de forme : le `to` typé est `'/{-$lang}/t/$titleSlug'` (segment optionnel
      DANS le template) ; la forme courte `'/t/$titleSlug'` est REFUSÉE par tsc (vérifié
      par `@ts-expect-error` consommé). Tous les littéraux `to` des phases 2+ utilisent
      donc la forme longue ; l'URL RENDUE sans lang reste `/t/…`. Artefacts de sanity
      supprimés, routeTree restauré (zéro diff de code après Phase 0).
- [x] Relire les décisions §3 (fermes). Ne rien re-décider. — Fait, rien à re-décider.

Gate Phase 0 : `git branch --show-current` = `feat/title-slug-in-url` ; baselines
consignées ; verdict `{-$lang}` consigné (support OK ou repli D-4 acté).

### Phase 1 — Module `title-routing` en TDD + câblage synchrone + header agnostic (moyen)

> Ordre TDD (D-11) : 1a AVANT 1b. Aucun déplacement de route dans cette phase — tout est
> pur/testable sans toucher au routing.

- [x] **1a. Tests d'abord** (rouges) : `lib/title-routing/*.test.ts` — écrits AVANT le
      module (red vérifié : 5 fichiers échouent module absent), matrice
      `buildLegacyRedirect` table-driven complète (home, stats/*, career/*, squad/*,
      community/*, media, explorer, matches, ascension/*, notifications, citations,
      commendations, palmares/*, objectifs, compare, synthesis, `?f=` + hash, bare
      `/players` → null [aucune route index legacy n'existe], `/players/{slug}` → home).
- [x] **1b.** Module implémenté (D-10) : `parseRouteSegments`, `resolveTitleGate`,
      `buildLegacyRedirect` (remaps internes CONTRE-VÉRIFIÉS sur pièces par
      l'orchestrateur : objectifs→ascension/objectifs, palmares(/*)→community(/*),
      compare→community/compare, synthesis→stats/synthesis, citations→career/citations,
      commendations→career/commendations), `applyActiveTitle` (extraction ISO, THROW sans
      rollback interne — chemin d'erreur à l'appelant, D-6), `switchTitle` = wrapper
      rollback store-only (comportement bouton identique, tests PR #59 inchangés verts).
      + `locales.ts` (KNOWN_LOCALES runtime — voir Découverte type Locale §7).
- [x] **1c.** Câblage synchrone au boot : `initTitleFromLocation(window.location.pathname)`
      dans `main.tsx` AVANT createRoot/installGlobalCapture ; testé unitairement. No-op
      runtime en Phase 1 (aucune route à segment), câblage prêt.
- [x] **1d.** Commentaires `client.ts` réécrits (3 sources : segment > hydratation/bascule
      > null/session). `[~]` partie « suppression du cas spécial halo_infinite » : DÉJÀ
      LIVRÉE avant D7 par `df7f13775` (getTitleHeader envoie le header pour tous les
      titres) — constat sur pièces, aucun mock/test à adapter.
- [x] **1e.** Garde-rail ratchet créé : `no-title-literals.ratchet.test.ts` — forme
      choisie (consignée) : règle `/t/` ARMÉE dès Phase 1 (allowlist datée = module seul,
      0 hit baseline) ; règle `/players/` ajoutée en Phase 2e (échelonnement du plan,
      commentaire daté dans le test). Pattern quote/backtick/template/regex — la prose
      des commentaires n'est pas matchée.

Gate Phase 1 : `make test-web` vert (nouveaux tests inclus) ; `make check-types` = 0 ;
`npm run lint` = 0 ; smoke navigateur : l'app fonctionne À L'IDENTIQUE (aucune route
déplacée), header `X-LevelUp-Title` visible en network sur les requêtes du titre PAR DÉFAUT
aussi (preuve D-9).

### Phase 2 — Déplacement structurel + littéraux + layout (LOURD — cœur du chantier)

- [x] **2a.** 36 fichiers de route déplacés en RENAMES git vers
      `routes/{-$lang}/t/$titleSlug/players/**`. Layout `t/$titleSlug.tsx` livré :
      projection PURE de `resolveTitleGate` + effet `applyActiveTitle` sur divergence
      (gardes anti-double-bascule : `isTitleSwitching`, `applyingRef` StrictMode,
      `applyFailed`) + `TitleGateScreen` (unknown/coming_soon/archived/switch_failed,
      10 clés i18n `common.title_gate.*` FR+EN, tokens sémantiques). RÈGLE ABSOLUE
      implémentée : l'Outlet ne rend JAMAIS en `wait`/divergence/bascule en vol — ce
      layout ferme la fenêtre pré-hydratation que `__root` laisse ouverte (il rend
      l'Outlet NU quand `!isBootstrapped`, constat §7).
- [x] **2b.** routeTree régénéré (vite build) ; ~100 littéraux typés corrigés, 108
      erreurs tsc → 0. Verdict params : TanStack EXIGE `titleSlug` dans `params` hors
      `from`/`beforeLoad`/`Route.useParams()` → helper central `useTitleSlug()`
      (module title-routing) au lieu de ~30 copies. Les redirects internes déplacés
      (objectifs/palmares/compare/synthesis/citations/commendations) retypés par tsc
      vers les cibles title-préfixées (⇒ Phase 3b déjà couverte, cf. journal).
- [x] **2c.** `ShellNavItem.to` typé `RouteTo = FileRouteTypes['to']` (nav L1/L2 dans
      tsc) — mismatch `profile/citations` → `career/citations` corrigé MÉCANIQUEMENT
      (commentaire in-code, effet attendu §7). `buildPlayerDestination` SUPPRIMÉ
      (zéro code mort), remplacé par `resolvePlayerSwitch` (pur) + `navigate({to:'.'})`
      qui préserve section/titre/langue. Matchers de pathname déportés sur helpers
      uniques `playerRelativePath`/`routeTemplateSuffix` (module title-routing,
      allowlisté) : `isCommunityPath`, `pageTitle.ts`, `navL1Sections`, `NavL2` SANS
      littéral `/players/`. `usePageScope.to` typé `FileRouteTypes['to']`.
- [x] **2d.** Index `/` re-ciblé (Navigate typé + `useTitleSlug`). Guard joueur :
      beforeLoad conservé pour les nav SPA (cibles retypées, spread `...params`) +
      filet fresh-load DÉCLARATIF `resolvePlayerFallback` (pur, testé) projeté par
      `PlayerLayout` — commentaire expliquant la division (trou n°1 revue v2 fermé).
- [x] **2e.** Ratchet `/players/` ARMÉ (allowlist datée 2026-07-23 : module
      title-routing, `routes/players/` splat Phase 3 inscrit par avance,
      feedback-drawer regex legacy volontaire, générés exclus) — 0 offender.
      Étendu (lot 2-C) à la forme backtick en contexte `to:`/`href:` (limite
      résiduelle documentée dans le test).
- [x] **2f.** Tests layout titre : 7 cas (wait→null, valid→Outlet, divergence→
      applyActiveTitle mocké puis convergence, unknown/coming_soon/archived→gate FR,
      switch_failed→retry). Tests nav adaptés sans affaiblissement.
- [x] **2g.** Héritage `lang` PROUVÉ par test avec vrai routeur + memory history
      (`langSegmentInheritance.test.ts`) : depuis `/en/t/…`, un Link sans `lang`
      préserve `/en` ; sans segment, reste sans segment. Aucun correctif nécessaire.
- [x] **2-C (complément découvert en 2-B).** Dernière famille de littéraux : cibles de
      route en template-literal backtick (échappent à tsc ET au ratchet) — 5 fichiers
      migrés (notifications/navigation.ts ~11 cibles typées + params, filterLink,
      PlayerDetailPanel, MediaViewer, CoverFlowModal) via `useTitleSlug()` et helper
      `playerScopedHref` (title-routing, hrefs pleine page — lang omis = session).
      Cycle d'import barrel→store évité (import du module feuille, documenté).

Gate Phase 2 : `make check-types` = 0 ; `make test-web` vert ; `npm run lint` = 0 ;
garde-rail vert ; smoke navigateur : `/t/halo_infinite/players/{p}/home` ET
`/t/halo_5/players/{p}/home` rendent la bonne page (header cohérent en network).

### Phase 3 — Redirections legacy (moyen)

- [x] **3a.** Splat `routes/players/$.tsx` DÉCLARATIF livré : gaté `isBootstrapped`,
      projection de `buildLegacyRedirect` via `router.history.replace(href)` (le href
      complet `?f=`/`#` ne passe pas dans un `<Navigate to>` typé sans cast/double
      parsing — choix documenté ; round-trip `searchStr` du sérialiseur par défaut
      VÉRIFIÉ byte-identique, y compris base64 `?f=`). BUG DE COURSE détecté par la
      vérification navigateur de l'orchestrateur puis corrigé : après le replace, le
      splat re-rend avec la location transitoire `/t/…` → sans garde, la branche
      « hors matrice → index » écrasait la redirection (perte suffixe/`?f=`/hash).
      Garde `isLegacyPath` sur LES DEUX branches + test de non-régression (rouge
      avant fix, vert après). `/players` nu → index (aucune route legacy n'existait).
- [~] **3b.** Re-pointage des redirections internes : DÉJÀ COUVERT par 2b (tsc a forcé
      le retypage des redirects déplacés vers les cibles title-préfixées). Vérifié par
      grep : aucun `to: '/players/` ne subsiste sous `routes/`.
- [x] **3c.** Matrice navigateur exécutée par l'orchestrateur (dev server relancé
      proprement) : legacy home → `/t/halo_infinite/…/home` direct ; suffixe +
      `?f=TESTVALUE123` + `#deep` préservés BYTE-IDENTIQUES ; `objectifs` →
      `ascension/objectifs` et `palmares` → `community` en UN hop ; `/players` nu →
      index → home ; **session H5 + bookmark legacy → `/t/halo_5/…` (trou n°1 fermé
      en conditions réelles)**. Trace des navigations consignée au journal.

Gate Phase 3 : `make check-types` = 0 ; `make test-web` vert ; navigateur : redirections
vérifiées, suffixe + `?f=` + hash préservés, aucune page morte.

### Phase 4 — `switchTitle` réduit, chemin d'erreur, courses (moyen)

- [x] **4a.** `TitleSwitcher.onSelect` → navigate typé vers le segment cible (le layout
      bascule sur divergence). Cas limite SANS joueur : `applyActiveTitle` direct +
      navigate `/` (pas de route de titre nue — documenté, échec loggé). `switchTitle`
      ET `setCurrentTitle` SUPPRIMÉS du store (plus aucun caller — grep vérifié) ;
      assertions PR #59 MIGRÉES vers `appShellStore.applyActiveTitle.test.ts` (même
      séquence load-bearing protégée : POST→set→resetFilters→cancel+clear→bootstrap→
      filet ; justification : switchTitle n'était qu'un wrapper, séquence observable
      identique). `createFilterStore.test.ts` INCHANGÉ (git diff vide).
- [x] **4b.** Chemin d'erreur D-6 complet : échec avec joueur courant → toast sonner
      (clé i18n `common.title_gate.switch_failed_toast` FR+EN) + navigate REPLACE vers
      le titre courant ; sans joueur → écran `switch_failed` (retry) conservé.
      Anti-boucle par convergence segment==store. Constat consigné : le layout reste
      MONTÉ au changement de param → `applyFailed` persisterait → le chemin avec
      joueur ne le pose JAMAIS (design). Testé (mock rejet → toast + navigate replace,
      pas de re-tentative ; sans joueur → écran).
- [x] **4c.** Test de course back/forward écrit AVANT — il a RÉVÉLÉ une fragilité
      réelle du layout Phase 2 : `applyingRef` relâché dans `.finally` APRÈS le
      re-render SYNCHRONE Zustand (isTitleSwitching→false) → convergence calée si le
      segment avait changé pendant le vol. CORRIGÉ dans le layout (relâchement en tête
      d'effet quand aucune bascule live ; dédup StrictMode préservée — applyActiveTitle
      pose isTitleSwitching=true synchrone). Après fix : exactement 2 appels [cible,
      retour], convergence finale. Point §7 `refetchOnWindowFocus` : test dédié vert,
      la re-comparaison absorbe une réhydratation pendant bascule — pas de patch ad hoc.
- [x] **4d.** Garde PR #59 INTACTE (zéro refactor, `createFilterStore.test.ts` intact
      vert) ; commentaire de coexistence segment↔`?f=` ajouté (défense en profondeur +
      rétro-compat) avec critère de retrait daté mesurable (≥ 1 release majeure post-D7
      + télémétrie ≥ 1 mois sans lien legacy sans segment).

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

- [2026-07-22] Clés de cache TanStack `filtersResolve`/`filtersPreview` SANS titre
  (`lib/query/keys.ts:29-32`), alors que la clé `home` (keys.ts:98) documente précisément ce
  piège (« un switch de titre servait les données périmées du titre précédent »). Risque
  résiduel : depuis df7f13775 le client n'affirme AUCUN header titre avant hydratation
  (session serveur autoritaire) et `useFiltersResolve` est `enabled` dès le playerSlug
  d'URL → une réponse `/filters/resolve` du titre de session peut être mise en cache sous
  une clé sans titre et servie après hydratation sur un autre titre. Découvert pendant le
  diagnostic « filtres H5 vides » (branche fix/h5-filters-asset-names — la cause racine de
  ce bug-là était backend, corrigée là-bas). Décision : différé vers CE plan — D7 rend le
  titre déterministe dès le premier rendu (segment d'URL → header affirmé d'emblée), ce qui
  ferme la fenêtre de course au boot ; lors de la Phase 1, AJOUTER le titre aux clés
  `filtersResolve`/`filtersPreview` (même motif que `home`) en défense en profondeur, et
  vérifier au passage les autres clés player-scoped title-dépendantes sans titre
  (`career`, `timeseries`, `synthesis`, `teammates`, …) — même audit, même commit.
  **TRAITÉ Phase 1 (2026-07-22)** : `filtersResolve`/`filtersPreview` enrichies du
  titre (+ callers `features/filters/queries.ts` ; `filtersResolveAll` reste broad
  par joueur, documenté). Audit des autres clés : AUCUNE fuite résiduelle nécessitant
  un changement (`queryClient.clear()` au switch + rendu gaté bootstrap + segment
  synchrone Phase 2) ; groupes « loader » (`career_`, `matches/$matchId`) et « poll »
  (`media`, `notifications`, `progression`) identifiés comme candidats à un
  durcissement OPTIONNEL — décision reportée au lot final.

- [2026-07-22] Pas de type `Locale` central dans `apps/web` (le plan §Phase 1 citait
  `lib/i18n.ts (type Locale)` qui n'existe pas) : `'fr' | 'en'` est redéclaré en 3
  alias nommés (`ManifestLocale`, `FieldMappingLocale`, `MetricLocale`) + inline
  (`client.ts`, `appShellStore.ts`). Le module `title-routing` porte désormais
  `KNOWN_LOCALES`/`Locale` (besoin runtime du parsing). Décision : centralisation d'un
  `Locale` unique = candidat LOT FINAL (règle des ≤ 2 copies déjà dépassée).

- [2026-07-22] Cohérence clé-de-cache (store `currentTitleSlug`) vs header (client
  `_currentTitleSlug`) pendant une bascule par URL : fermée structurellement si le
  layout D-6 ne rend PAS l'Outlet pendant `wait`/divergence/bascule en vol — précision
  d'implémentation INTÉGRÉE au brief Phase 2 (2a). Pas de changement de design.
  **TRAITÉ Phase 2** (layout livré avec cette règle + constat aggravant : `__root` rend
  l'Outlet NU quand `!isBootstrapped`, __root.tsx:175-177 — le layout titre est le seul
  rempart du sous-arbre title-scoped).

- [2026-07-23] (lot 2-B) Cibles de route front en template-literal backtick
  (`` `/players/${slug}/…` ``) échappant à tsc ET au ratchet : 5 fichiers
  (notifications/navigation, filterLink, PlayerDetailPanel, MediaViewer,
  CoverFlowModal). L'audit §2 du plan les avait manquées. **TRAITÉ lot 2-C** (même
  phase) : migration typée + `playerScopedHref` + extension du ratchet au contexte
  `to:`/`href:` backtick.

- [2026-07-23] (lot 2-C) `notif.target_route` ÉMIS PAR LE BACKEND porte encore l'ancien
  format `/players/…` (donnée runtime, pas un littéral source). Couvert par le splat
  legacy Phase 3 (1 hop). Mise à jour de l'émission backend = chantier Go HORS D7
  (documenté in-code navigation.ts). Décision : DIFFÉRÉ (signalé utilisateur).

- [2026-07-23] (lot 2-B, mineur) `ExplorerPage.tsx:168` : `eslint-disable-next-line
  set-state-in-effect` MAL PLACÉ (désactive la ligne suivante, pas les setState
  165-167) → warning baseline. Décision : lot final (trivial).

- [2026-07-23] (lot 2-B, mineur) `NavL1MobileActions.test.tsx:42` : fixture pathname
  ancien style `/players/test-player/home` (passe par sous-chaîne). Décision : lot
  final (cohérence des fixtures).

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

- **[2026-07-22] Phase 0 CLOSE.** Gate : branche `feat/title-slug-in-url` active (créée
  depuis `refactor/ascension-ux-2026-07` = origin/main + fixes Ascension, demande
  utilisateur) ; baselines consignées (192 occ / 70 fichiers typés, 131 strings échappées,
  26 fichiers e2e) ; verdict `{-$lang}` : SUPPORT OK, repli D-4 non nécessaire. Forme
  canonique des `to` typés : `'/{-$lang}/t/$titleSlug/…'` (forme courte refusée par tsc).
  Exécution pilotée : Fable orchestre + vérifie, agents Opus implémentent (phases 1-6).
  Zéro diff de code à la clôture (artefacts sanity supprimés, routeTree restauré).

- **[2026-07-22] Phase 1 CLOSE.** Agent Opus (module + tests TDD red→green), revue
  orchestrateur sur pièces (remaps legacy contre-vérifiés contre les 6 redirects
  existants — exacts ; extraction `applyActiveTitle` ISO-comportement vérifiée ligne à
  ligne). Gate re-exécuté par l'orchestrateur : vitest 291 fichiers / 2573 passés /
  14 skip pré-existants / 0 échec (+81 tests du module) ; `tsc -b` = 0 ; eslint fichiers
  touchés = 0. Smoke navigateur (script Playwright ad hoc, serveurs dev relancés) :
  app à l'identique sur `/players/Chocoboflor/home`, 0 erreur page, TOUTES les requêtes
  API JSON portent `X-LevelUp-Title: halo_infinite` (titre par défaut — preuve D-9) ;
  `/bootstrap` sans header (pré-hydratation, session autoritaire, documenté) ; assets
  `<img>` sans header (fetch navigateur, hors client API — attendu). Découvertes
  consignées §7 (type Locale absent, durcissement optionnel clés loader/poll → lot
  final). Binaires Playwright chromium (ré)installés au passage (requis Phase 6).

- **[2026-07-23] Phase 2 CLOSE** (3 lots Opus : 2-A structure+layout, 2-B surfaces
  échappées, 2-C backtick emitters — chaque lot revu sur pièces par l'orchestrateur).
  Gate re-exécuté par l'orchestrateur : `tsc -b` = 0 ; vitest **295 fichiers / 2605
  passés / 14 skip pré-existants / 0 échec** ; `eslint .` = 0 erreur (68 warnings
  baseline pré-existante) ; ratchet `/t/` + `/players/` vert, 0 offender. Smoke
  navigateur (Playwright) : `/t/halo_infinite/players/Chocoboflor/home` rendu, headers
  API uniformément `halo_infinite` ; `/t/halo_5/players/Chocoboflor/home` FRESH-LOAD
  avec session Infinite → convergence D-6 observée (URL stable, `?f=` estampillé
  `halo_5`, re-bootstrap, contenu H5) — les requêtes shell pré-convergence partent avec
  l'ancien titre puis sont purgées par `clear()` (conforme D-3 : le shell suit le
  store) ; `/t/inconnu/…` → écran gate « Titre introuvable » + lien retour, ZÉRO fuite
  de contenu joueur sous le gate, aucune erreur console. Legacy `/players/…` = 404
  TanStack à ce stade : ATTENDU, le splat arrive Phase 3. Verdicts consignés : params
  `titleSlug` requis hors from/beforeLoad/useParams (helper `useTitleSlug`) ; héritage
  `lang` OK (test routeur réel).

- **[2026-07-23] Phase 3 CLOSE.** Splat legacy déclaratif (agent Opus) ; la première
  livraison passait tsc/vitest mais la VÉRIFICATION NAVIGATEUR de l'orchestrateur a
  détecté un bug de course post-replace (toutes les URLs legacy perdaient
  suffixe/`?f=`/hash via un détour par `/`) — diagnostic orchestrateur, fix agent
  (garde `isLegacyPath` double-branche) + test de non-régression rouge→vert. Gate
  re-exécuté : tsc 0 ; vitest 296 fichiers / 2609 passés / 0 échec ; matrice
  navigateur INTÉGRALE verte après redémarrage propre du dev server (A1 home direct,
  A2 `?f=`+hash byte-préservés, A3 objectifs→ascension/objectifs 1 hop, A4
  palmares→community, A5 `/players` nu→index, B1 session H5 → `/t/halo_5/…`).
  Leçon consignée : les tests de composant du splat ne simulaient pas le re-render
  en transition — la vérification navigateur end-to-end reste OBLIGATOIRE à chaque
  étape (elle a payé ici).

- **[2026-07-23] Phase 4 CLOSE.** TitleSwitcher navigate-first (agent Opus) ;
  `switchTitle`/`setCurrentTitle` supprimés (0 code mort), assertions PR #59 migrées
  sans affaiblissement. Le test de course 4c (écrit d'abord) a détecté une fragilité
  de convergence du layout Phase 2 (verrou vs re-render synchrone Zustand) — corrigée
  dans le layout, pas dans le test. Chemin d'erreur D-6 complet (toast + navigate
  replace / écran fallback sans joueur). Gate re-exécuté orchestrateur : tsc 0 ;
  vitest 296 fichiers / 2613 passés / 0 échec ; smoke UI navigateur : switch
  Infinite→H5→Infinite par clics réels, segment + `?f=` ré-estampillés, 0 erreur.

- Découverte (2026-07-23, Phase 4, mineure) : `createFilterStore.test.ts:142` porte
  une référence commentée à l'ex-`switchTitle` — laissée pour honorer la contrainte
  « INCHANGÉ » du gate 4d. Rafraîchissement de commentaire → lot final.
