# PLAN — D7 : le titre dans l'URL (titre = segment de route)

Statut : PLANIFIE (aucune ligne de code écrite).
Date : 2026-07-13.
Auteur du plan : architecte Opus (worktree isolé, lecture seule).
Principe APPROUVE par l'utilisateur le 2026-07-13.

Branche cible d'implémentation : **`feat/title-slug-in-url`** (depuis `main` à jour).
Ne PAS implémenter sur la branche worktree de rédaction de ce plan.

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
store et le client API SUIVENT le segment d'URL (inversion de contrôle).

**Pourquoi.** Aujourd'hui le titre est implicite : résolu au bootstrap
(`hydrateFromBootstrap`, `appShellStore.ts:114-153`), mémorisé dans un module-level du client
API (`client.ts:64` `_currentTitleSlug`), transporté par un header
(`client.ts:80-85` `getTitleHeader`), jamais présent dans l'URL. Conséquences : URLs non
auto-porteuses (un lien partagé ne dit pas son titre), et une **classe de bugs « titre
implicite »** dont la fuite deep-link `?f=` (corrigée en défense par PR #59, commit
`0b2e5cdb8`) est le symptôme le plus récent. Mettre le titre dans l'URL supprime la cause
racine de cette classe : au fresh-load, le titre est connu AVANT le bootstrap, de façon
synchrone, par lecture du segment.

**Critères de succès (tous vérifiables) :**

1. Toute page title-scoped est servie sous un préfixe `/t/{slug}/…` ; l'URL affichée dans la
   barre d'adresse porte le titre actif sur les deux titres.
2. `make check-types` = 0 erreur (le typecheck TanStack est le juge d'exhaustivité de la
   migration des littéraux de route — voir §5, Phase 1).
3. `make test-web` (vitest run) vert, y compris les tests PR #59
   (`createFilterStore.test.ts`, `appShellStore.switchTitle.test.ts`).
4. `cd apps/web && npm run lint` = 0.
5. Toute ancienne URL `/players/{playerSlug}/…` (bookmark, lien partagé, notification, lien
   interne) redirige (301-like, `replace`) vers l'équivalent `/t/{slug}/players/{playerSlug}/…`
   en préservant le suffixe complet, le `search` (`?f=`) et le hash — **aucun lien mort**.
6. Le changement de titre via le sélecteur (`TitleSwitcher`) change le SEGMENT d'URL ; les
   données, filtres (reset) et cache basculent ; aucune fuite inter-titres.
7. Un `slug` d'URL inconnu / `coming_soon` / `archived` mène à un gate propre (redirection
   vers le titre actif OU écran d'indisponibilité — parité avec `RequireActiveTitle`,
   `require_active_title.go:48-81`), jamais une page blanche.
8. Non-régression PR #59 : un deep-link legacy `?f=` (enveloppe v2 ou legacy) reste honoré /
   rejeté correctement selon le titre.
9. Vérification NAVIGATEUR (chrome-devtools) exécutée sur les 2 titres + les redirections,
   captures consignées au journal du plan.

---

## 2. Constat sur pièces — état actuel (titre implicite)

Vérifié par lecture directe (fichier:ligne réels au 2026-07-13) :

- **Routing file-based, aucun segment titre.** `apps/web/src/routes/` : pages title-scoped
  toutes sous `players/$playerSlug/…` (≈ 50 fichiers de route). L'index `routes/index.tsx:30-41`
  redirige `/` → `/players/$playerSlug/home`. Layout joueur `routes/players/$playerSlug.tsx:16-50`
  (guard `beforeLoad`, sync joueur↔slug). `routeTree.gen.ts` généré — **ne jamais l'éditer**
  (CLAUDE.md §13, ADR routing).

- **Titre résolu au bootstrap puis porté par le client API.**
  `appShellStore.ts:114-116` : `hydrateFromBootstrap` lit `data.current_title_slug ??
  'halo_infinite'` puis `setApiTitleSlug(titleSlug)`. `client.ts:64-85` : module-level
  `_currentTitleSlug`, `setApiTitleSlug()`, `getApiTitleSlug()`, `getTitleHeader()` (header
  `X-LevelUp-Title` envoyé UNIQUEMENT si titre ≠ `halo_infinite`).

- **Switch de titre piloté par un bouton, pas par l'URL.**
  `appShellStore.ts:165-218` `switchTitle()` : POST `/session/context {title_slug}` → `setApiTitleSlug`
  → `set({currentTitleSlug})` → reset filtres solo/squad → `queryClient.cancelQueries()` +
  `clear()` → re-`GET /bootstrap` → réhydrate → filet joueur. Puis `TitleSwitcher.tsx:40-58`
  `navigate({to:'/players/$playerSlug/home'})` APRÈS le switch (l'URL portait encore l'ancien
  joueur).

- **Backend déjà header/session-driven — aucun changement Go attendu.**
  `middleware/title.go:55-72` `resolveTitleSlug` : header `X-LevelUp-Title` > session
  `CurrentTitleSlug` > défaut `halo_infinite`. `require_active_title.go:48-81` : gate 503
  `title_unavailable` pour titre non actif. Le segment d'URL n'a PAS besoin d'être lu par le
  back : le front continue de poser le header `X-LevelUp-Title` (dérivé du segment). **D7 est
  un chantier FRONTEND.** (Tout besoin Go = Découverte.)

- **Garde deep-link `?f=` (PR #59) à articuler.**
  `createFilterStore.ts:182-223` : enveloppe v2 `{t: titre, c: contexte}` (`encodeToUrl` /
  `decodeFromUrl`), `:298-309` `reconcileActiveTitle` (reset one-shot si le titre du deep-link
  ≠ titre actif), `:428-442` `onRehydrateStorage` (mémorise `urlHydratedTitleSlug`).
  `appShellStore.ts:151-152` appelle `reconcileActiveTitle` depuis le bootstrap. Avec le titre
  dans l'URL, le titre est connu **synchrone au fresh-load** (segment) : l'estampille `t` du
  `?f=` n'est plus nécessaire pour la CORRECTION, mais reste conservée en **défense en
  profondeur** + rétro-compat des `?f=` déjà partagés.

- **Helpers de navigation centralisés à rendre title-aware.**
  `shellNavigation.ts:13-71` (nav L1/L2, littéraux `to`), `:91-111` `buildPlayerDestination`
  (préserve le suffixe au switch de joueur), `pageTitle.ts:12-40` (règles de titre d'onglet
  par pattern de route). Pattern de redirection legacy réutilisable :
  `routes/players/$playerSlug/objectifs/index.tsx` (`throw redirect({to, params, replace})`).

- **Ampleur mécanique.** ≈ **69 fichiers** référencent `players/$playerSlug` en littéraux de
  route typés (Link `to=`, `navigate({to})`, `redirect({to})`, `Route.useParams()`). Chaque
  littéral est vérifié à la compilation contre `routeTree.gen.ts` : ajouter un param
  `titleSlug` au-dessus de `players` casse le typecheck partout — **c'est une propriété de
  sûreté** (le compilateur énumère chaque littéral à corriger ; rien ne peut être oublié
  silencieusement).

---

## 3. Décisions PRÉ-TRANCHÉES (fermes — ne pas re-débattre en cours d'exécution)

### D-1 — Schéma d'URL retenu : `/t/{slug}/players/{playerSlug}/…`

Le titre est un segment préfixe, sous le namespace court `/t/`, au-dessus du joueur. Exemple :
`/t/halo_infinite/players/jgtm/home`, `/t/halo_5/players/jgtm/stats/timeseries`.

**Pourquoi ce schéma :**
- Hiérarchie correcte : un joueur appartient à un titre (DBs par titre,
  `data/titles/{slug}/players/{gamertag}/`, ADR 0008) → titre AU-DESSUS de joueur.
- Namespace `/t/` : évite toute collision avec les routes agnostiques de racine (`/admin`,
  `/settings`, …). Sans préfixe (`/{slug}/players/…`), `/admin` serait ambigu avec un slug.
- Segment = SÉLECTEUR de titre, PAS un branchement de logique. Le code métier reste branché
  sur les **capabilities** (`HasCapability`, jamais `slug == "..."` — ratchet
  `no_slug_comparison_test.go`, CLAUDE.md « Multi-titre »). Le segment n'alimente que
  `setApiTitleSlug` (donc le header) et la validation front.

**Alternatives écartées :** `/{slug}/players/…` (collision racine) ; `/game/{slug}/` ou
`/titles/{slug}/` (plus verbeux, `/titles` prête à confusion avec `/admin/titles`).

### D-2 — Valeur du slug dans l'URL = slug INTERNE verbatim

`halo_infinite`, `halo_5` (avec underscore) tels quels, PAS d'alias hyphénisé. Évite une
couche de mapping URL↔slug interne (table de traduction + risque de désync). Le segment est un
sélecteur direct. Tradeoff esthétique (underscore) accepté ; un alias cosmétique éventuel est
un chantier ultérieur distinct, hors D7.

### D-3 — Pages title-agnostiques : HORS segment (restent à la racine)

Restent à la racine, SANS segment titre : `admin` + `admin/*`, `settings`, `setup`, `login`,
`register`, `changelog`, `help`, `groups`, `join`, `onboarding.openspartan`, `lab/*`, et
l'index `/`. Elles ne consomment pas de donnée title-scoped ; leur ajouter le segment ajoute
du bruit et double la matrice de redirection sans valeur.

- Le titre « actif » pour ces pages = dernier titre connu, via la session serveur / header
  (`resolveTitleSlug` session-fallback). Le layout titre (D-6) commit `POST /session/context`
  à chaque visite title-scoped, gardant la session serveur à jour pour ces pages.
- Cas admin per-titre (`admin/sync`, `admin/data`, `admin/titles`) : gardent leurs sélecteurs
  in-page / le titre actif de session — **non modifiés par D7**. Si l'un dépend silencieusement
  du titre actif d'une façon cassée par le déplacement → **Découverte**, pas un fix in-line.

### D-4 — Langue dans la route : HORS PÉRIMÈTRE de D7 (pointeur explicite)

La langue (idée sœur backlog Notion) N'est PAS traitée dans D7.

**Justification :** (a) valeur indépendante — D7 corrige la classe de bugs « titre implicite »
maintenant ; langue-dans-URL est un confort (liens localisés partageables) ; (b) bundler double
le blast-radius (deux params sur chaque littéral, double matrice de redirection) sur un
refactor mécanique déjà lourd — contraire à un périmètre borné (plan-review §9) ; (c) la
mécanique « segment = source de vérité » livrée par D7 est réutilisable telle quelle pour la
langue dans un chantier jumeau. **Guidance pour ce futur chantier :** la langue se placerait
AU-DESSUS du titre (`/{lang}/t/{slug}/…`), car la locale est globale-app alors que le titre est
un scope de données. Aujourd'hui la locale est portée par header `X-LevelUp-Locale`
(`client.ts:99-101`, `title.go:37-44`) + `setLocale` — même pattern que le titre, donc
migration analogue le moment venu.

### D-5 — Stratégie de redirection : splat redirect legacy, préservation intégrale

Le sous-arbre `routes/players/…` est physiquement DÉPLACÉ sous `routes/t/$titleSlug/players/…`.
La racine `routes/players/` libérée héberge un **splat redirect** (`routes/players/$playerSlug/$`
ou équivalent TanStack) dont le `beforeLoad` fait `throw redirect(...)` vers
`/t/{activeSlug}/players/{playerSlug}/{suffixe}`, où `activeSlug` = `appShellStore
.getState().currentTitleSlug` (hydraté par le bootstrap, garanti par le blocage de rendu de
`__root`). Préserve suffixe + `search` (`?f=`) + hash. `replace: true`.

- Les redirections internes déjà présentes (objectifs→ascension, palmares→community,
  compare→community) sont re-pointées vers leurs cibles title-préfixées.
- L'index `/` redirige vers `/t/{activeSlug}/players/{playerSlug}/home`.

### D-6 — Inversion de contrôle : layout `t/$titleSlug` pilote le store

Nouveau layout `routes/t/$titleSlug.tsx`, parent du sous-arbre joueur. Il :
1. valide `titleSlug` contre `availableTitles` (`appShellStore`, hydraté par bootstrap) ;
2. gate `coming_soon`/`archived`/inconnu (parité `RequireActiveTitle`) → redirection titre
   actif ou écran d'indisponibilité ;
3. **réconcilie store ← URL** : si `titleSlug !== currentTitleSlug`, applique la bascule
   (extraction de la logique de `switchTitle`) : `setApiTitleSlug` + `set(currentTitleSlug)` +
   `POST /session/context` + reset filtres + `clear()`/refetch — mais DÉCLENCHÉE PAR LE
   SEGMENT, pas par un bouton.

Le `TitleSwitcher` (D-7 / Phase 3) se réduit alors à un `navigate` vers le segment cible.

### D-7 — Articulation `?f=` : segment = source de vérité, PR #59 gardée en défense

Le segment d'URL porte désormais le titre → l'estampille `t` du `?f=` (PR #59) n'est plus
nécessaire à la correction. **Décision : conserver la garde PR #59 intacte** (enveloppe v2 +
`reconcileActiveTitle`) en défense en profondeur et rétro-compat des `?f=` déjà partagés. Ne
PAS refactorer/supprimer cette garde dans D7 (risque de régression de la correction #59
disproportionné vs le gain). Documenter dans le code (commentaire) : la garde reste ; critère
de retrait éventuel = « quand tous les `?f=` en circulation portent un segment titre » (non
mesurable court terme → pas de retrait planifié ici). Toute simplification = Découverte.

---

## 4. Périmètre

**Dans le périmètre (frontend `apps/web`) :** structure de routes (`routes/t/$titleSlug/…`),
layout titre, migration de tous les littéraux de route typés, helpers centralisés
(`shellNavigation`, `buildPlayerDestination`, `pageTitle`), `appShellStore.switchTitle` +
`TitleSwitcher`, redirections legacy, articulation `?f=` (conservation), tests vitest,
vérification navigateur.

**Hors périmètre (ne pas traiter — noter en Découvertes si rencontré) :** langue dans la route
(D-4) ; refactor/suppression de la garde `?f=` PR #59 (D-7) ; tout changement Go (le back est
déjà header/session-driven, D-2 §2) ; sélecteurs de titre in-page des pages admin (D-3) ;
alias cosmétique de slug (D-2) ; toute dette lint pré-existante (baseline gelée).

---

## 5. Phases (ordre strict — une étape CLOSE avant la suivante)

> Rappel plan-execution : clôture d'étape = gate passé (commandes exactes, sorties propres) +
> tous les items statués `[x]`/`[~]`/`[!]` + plan mis à jour + entrée thought_log + point
> d'étape utilisateur. Aucune case vide à la clôture. Zéro fix hors périmètre.
> Note tests : vitest `apps/web` tourne HORS sandbox → invoquer avec
> `dangerouslyDisableSandbox=true` (mémoire projet `reference_vitest_outside_sandbox`).

### Phase 0 — Cadrage & baseline (rapide)

- [ ] Créer la branche `feat/title-slug-in-url` depuis `main` à jour (`git fetch` +
      vérifier `git log --oneline -1 origin/main`).
- [ ] Figer l'inventaire de complétude : `grep -rn "players/\$playerSlug" apps/web/src
      --include=*.tsx --include=*.ts` → consigner le nombre de fichiers (attendu ≈ 69) dans le
      journal du plan comme baseline. Le typecheck (Phase 1) est le gate d'exhaustivité réel ;
      cet inventaire est un repère.
- [ ] Relire les décisions §3 (fermes). Ne rien re-décider.

Gate Phase 0 : branche créée et courante (`git branch --show-current` = `feat/title-slug-in-url`) ;
baseline consignée au journal.

### Phase 1 — Sous-arbre titre + résolution URL→store (LOURD — cœur du chantier)

- [ ] **1a.** Créer `routes/t/$titleSlug.tsx` (layout) : `beforeLoad`/composant validant
      `titleSlug` vs `availableTitles` ; gate `coming_soon`/`archived`/inconnu (parité
      `RequireActiveTitle` — redirection titre actif OU écran indisponibilité title-agnostic) ;
      réconciliation store←URL (D-6, point 3).
- [ ] **1b.** Extraire de `switchTitle` (`appShellStore.ts:165-218`) une fonction réutilisable
      pilotée par le slug (ex. `applyActiveTitle(slug)`) : `setApiTitleSlug` + `set` +
      `POST /session/context` + reset filtres + `clear()`/refetch + filet joueur. Utilisée par
      le layout 1a. (La forme bouton de `switchTitle` sera réduite en Phase 3.)
- [ ] **1c.** Déplacer TOUT `routes/players/**` → `routes/t/$titleSlug/players/**` (déplacement
      de fichiers, `git mv`). Ne pas éditer `routeTree.gen.ts` (régénéré au build/dev).
- [ ] **1d.** Régénérer `routeTree.gen.ts` (`cd apps/web && npm run build` ou dev une fois),
      puis corriger CHAQUE littéral de route typé cassé jusqu'à `tsc -b` = 0 : Link `to=`,
      `navigate({to})`, `redirect({to})`, `Route.useParams()` (expose désormais `titleSlug`),
      en injectant le param `titleSlug`. Le typecheck énumère l'exhaustif.
- [ ] **1e.** Rendre title-aware les helpers centralisés : `shellNavigation.ts` (`to` + param
      `titleSlug` au rendu), `buildPlayerDestination` (`:91-111` préfixer le titre),
      `pageTitle.ts` (`:12-40` patterns `/t/$titleSlug/players/…`).
- [ ] **1f.** Re-cibler l'index `/` (`routes/index.tsx:30-41`) et les guards
      (`routes/t/$titleSlug/players/$playerSlug.tsx` ex-`players/$playerSlug.tsx:16-50`,
      redirections internes `__root.tsx`) vers les cibles title-préfixées.
- [ ] **1g.** Tests : adapter les tests de route/nav cassés ; ajouter un test du layout titre
      (slug valide → rendu ; slug inconnu → gate ; réconciliation store←URL).

Gate Phase 1 : `make check-types` = 0 (exhaustivité) ; `make test-web` vert ; `cd apps/web &&
npm run lint` = 0 ; smoke navigateur (chrome-devtools) : `/t/halo_infinite/players/{p}/home` ET
`/t/halo_5/players/{p}/home` rendent la bonne page avec le bon titre (header `X-LevelUp-Title`
cohérent en network).

### Phase 2 — Redirections legacy (moyen)

- [ ] **2a.** Ajouter le splat redirect `routes/players/$playerSlug/$` (ou forme TanStack
      équivalente) : `beforeLoad` → `throw redirect` vers `/t/{activeSlug}/players/{playerSlug}/
      {suffixe}`, `activeSlug = appShellStore.getState().currentTitleSlug`, préservant
      `search` (`?f=`) + hash, `replace: true` (réutiliser le pattern
      `objectifs/index.tsx`).
- [ ] **2b.** Vérifier la matrice complète des anciennes URLs → chaque famille redirige :
      `home`, `stats/{timeseries,sessions,synthesis,index}`, `career/*`, `squad/*`,
      `community/*`, `media`, `explorer`, `matches/$matchId(/replay)`, `ascension/*`,
      `notifications`, `citations`/`commendations`, legacy `palmares/*` + `objectifs` +
      `compare` + `synthesis`.
- [ ] **2c.** Re-pointer les redirections internes existantes (objectifs→ascension,
      palmares→community, compare→community) vers leurs cibles title-préfixées.

Gate Phase 2 : `make check-types` = 0 ; `make test-web` vert ; navigateur : ouvrir 4+ anciennes
URLs bookmark (dont une avec `?f=`, dont une sur H5) → toutes redirigées, suffixe + `?f=`
préservés, aucune page morte.

### Phase 3 — `switchTitle` réduit + articulation `?f=` (moyen)

- [ ] **3a.** Réécrire `TitleSwitcher.onSelect` (`:40-58`) + réduire `switchTitle`
      (`:165-218`) : le sélecteur fait `navigate({to:'/t/$titleSlug/players/$playerSlug/home',
      params:{titleSlug:newSlug, playerSlug}})` ; le layout (1a) fait la bascule. Supprimer le
      code devenu mort (POST+rebootstrap+navigate dupliqués) — **0 code mort** (CLAUDE.md §7),
      avec ses éventuels tests obsolètes.
- [ ] **3b.** Articuler `?f=` (D-7) : confirmer que le segment porte le titre et que la garde
      PR #59 (`createFilterStore.ts` v2 + `reconcileActiveTitle`) reste INTACTE en défense ;
      ajouter/mettre à jour le commentaire de code expliquant la coexistence segment↔`?f=` et
      le critère de retrait. Ne PAS refactorer la garde (hors périmètre, D-7).
- [ ] **3c.** Vérifier que les tests PR #59 (`createFilterStore.test.ts`,
      `appShellStore.switchTitle.test.ts`) passent inchangés OU sont adaptés au nouveau flux de
      switch (si adaptés : documenter pourquoi au journal, ne pas affaiblir l'assertion).

Gate Phase 3 : `make check-types` = 0 ; `make test-web` vert (incl. tests PR #59) ;
`grep -rn` confirmant l'absence du code de switch dupliqué mort ; navigateur : switch de titre
via le sélecteur → segment d'URL change, données/filtres basculent.

### Phase 4 — Vérification navigateur complète + livraison (moyen)

- [ ] **4a.** chrome-devtools, sur les 2 titres : parcourir home / stats / career / squad /
      community / explorer / un match ; vérifier l'URL porte `/t/{slug}/`, le header
      `X-LevelUp-Title` en network, et des données du bon titre.
- [ ] **4b.** Switch via le sélecteur → segment change, données basculent, filtres reset, pas
      de fuite (relire scénario PR #59 : appliquer un filtre session sur un titre, switcher,
      vérifier reset).
- [ ] **4c.** Redirections : anciennes URLs bookmarkées (Infinite ET H5) redirigent ; deep-link
      `?f=` legacy honoré/rejeté correctement (non-régression PR #59).
- [ ] **4d.** Segment invalide : `/t/inconnu/players/…` et (si dispo) un titre `coming_soon` →
      gate propre (redirection/écran indispo), pas de page blanche.
- [ ] **4e.** Suite complète : `make check-types`, `make test-web`, `cd apps/web && npm run
      lint` ; optionnel si pertinent `npm run test:e2e` (Playwright — nécessite `make dev`).
- [ ] **4f.** `thought_log` (entrée de clôture), MAJ `project_map` si la cartographie route en
      dépend, statut final de chaque item du plan.

Gate Phase 4 : tous les gates ci-dessus verts ; captures/observations navigateur consignées au
journal du plan ; delivery-checklist passée.

---

## 6. Protocole de reprise de session

1. Relire le contrat `plan-execution` puis ce plan (§3 décisions fermes, §5 phases).
2. Le fichier plan est la source de vérité de l'avancement : reprendre à la **première case
   non statuée de l'étape courante** (celle dont le gate n'est pas coché).
3. Vérifier `git branch --show-current` = `feat/title-slug-in-url` ; `git log --oneline -10`
   pour situer les commits d'étape.
4. Ne pas re-décider ce qui est tranché en §3. Toute nouvelle décision produit = signalée
   immédiatement à l'utilisateur (plan-execution §3).
5. Vérifier sur pièces avant de coder (le code a pu bouger depuis la rédaction — rouvrir les
   fichier:ligne du §2).

---

## 7. Découvertes (à remplir pendant l'exécution — ne PAS traiter hors périmètre)

> Toute dette, bug, incohérence ou idée rencontrée qui n'est pas dans le périmètre §4 se note
> ICI et n'est pas traitée (sauf si elle bloque le gate de l'étape courante). Format :
> `[YYYY-MM-DD] découverte — décision (différé / signalé / bloquant traité)`.

- (aucune à ce jour)

Points de vigilance connus à surveiller (candidats Découvertes) :
- Mismatch nav pré-existant : `shellNavigation.ts:47` pointe `/players/$playerSlug/profile/
  citations` alors que les routes réelles sont `career/citations` + `citations.tsx` (aucun
  dossier `profile/`). Ne pas corriger dans D7 — noter si le typecheck le fait ressortir.
- Ordonnancement bootstrap↔segment : au fresh-load d'un `/t/halo_5/…`, `__root` bootstrap
  d'abord (titre session, possiblement `halo_infinite`), puis le layout titre réconcilie vers
  `halo_5` (POST session/context + refetch). Transitoire acceptable (identique au switch
  actuel). Option d'optimisation (poser le header titre issu du segment DÈS la requête
  `/bootstrap`) = amélioration, PAS requise pour le gate — si tentée, la traiter comme item
  explicite, sinon Découverte.

---

## 8. Effort estimé

| Phase | Charge | Nature |
|---|---|---|
| 0 — Cadrage & baseline | Rapide | branche + inventaire |
| 1 — Sous-arbre + résolution URL→store | **LOURD** | déplacement ≈ 50 routes + correction de tous les littéraux typés (≈ 69 fichiers) jusqu'à tsc=0 + layout + helpers |
| 2 — Redirections legacy | Moyen | splat redirect + matrice + re-pointage internes |
| 3 — switchTitle réduit + `?f=` | Moyen | rewire sélecteur, purge code mort, articulation garde |
| 4 — Vérif navigateur + livraison | Moyen | chrome-devtools 2 titres + redirections + suite complète |

**Total : chantier LOURD** (1 grosse session dense ou 2). Le risque est concentré en Phase 1 ;
il est BORNÉ par le typecheck (aucune omission silencieuse possible). Backend non touché.

---

## 9. Conformité plan-review (auto-contrôle §9 exécutabilité)

- Objectif + critères de succès mesurables : §1 (9 critères vérifiables).
- Phases ordonnées, périmètre FERMÉ par étape (items cochables, pas de « etc. ») : §5.
- Gates vérifiables par commandes exactes (`make check-types`, `make test-web`,
  `npm run lint`, chrome-devtools) : §5.
- Statuts d'item `[x]`/`[~]`/`[!]` + règle « aucune case vide à la clôture » : §5 (rappel).
- Règle d'ordre explicite (N close avant N+1, définition de « clos ») : §5 (rappel).
- Interdiction fixes hors périmètre + section Découvertes : §4, §7.
- Décisions produit TRANCHÉES avant exécution (schéma URL, agnostiques, langue in/out,
  redirection, `?f=`) : §3 (D-1..D-7).
- Protocole de reprise : §6.
- Branche cible nommée : `feat/title-slug-in-url` (en-tête + Phase 0).
- Renvoi explicite `plan-execution` : en-tête.
- Vérification navigateur 2 titres + redirections : Phase 4.

---

## 10. Journal du plan (rempli à l'exécution)

- (vide — à alimenter à chaque clôture d'étape : date, étape, gate, statut des items,
  observations navigateur.)
</content>
</invoke>
