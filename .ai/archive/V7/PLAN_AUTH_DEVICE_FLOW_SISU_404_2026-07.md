# PLAN — Onboarding Device Code Flow cassé (SISU endpoint 404) + fix UX spinner infini

> **STATUT : COMPLÉTÉ — 2026-07-13.** Tous les lots soldés : lot ops item 0/0b
> (URL device-code `.srf` + race single-flight + lien de vérif court, commits
> `a94fa3269`/`ba37cbe56`), lot A (UI d'erreur StepDeviceCode), lot D (garde-rail
> réseau opt-in + doc `auth_provider` FR/EN + D3 vérifié). Lot B SANS OBJET
> (endpoint jamais retiré — l'URL du code était fausse). Tracker §4 intégralement
> statué. Branche `fix/auth-deviceflow-lots-ad`.
>
> Date de rédaction : 2026-07-11. Exécutant prévu : Opus. Superviseur : Guillaume.
> Origine : investigation du 2026-07-11 — wizard bloqué sur « Démarrage du Device Code Flow… »
> au premier lancement sur un PC neuf. Cause racine tracée sur pièces (cf. §1).
> Entrée thought_log associée : `[2026-07-11] Onboarding bloqué … SISU endpoint 404`.
>
> Objectif : rétablir un onboarding Device Code Flow fiable pour l'app distribuée, avec
> garde-rails empêchant la régression de repasser inaperçue, + corriger le bug UX qui masque
> l'échec derrière un spinner infini.
> Critère de succès global : onboarding device-flow fonctionnel vérifié bout-en-bout ;
> tout échec de `start` est surfacé côté UI ; un garde-rail détecte la panne d'endpoint ;
> chaque case du tracker (§4) porte un statut final `[x]`/`[~]`/`[!]`.

---

## 0. CONTRAT D'EXÉCUTION (rappel — le skill `plan-execution` fait foi)

1. **Ordre strict des lots** (§3). Ne pas commencer le lot N+1 tant que le lot N n'est pas
   CLOS. Exception : le lot B (décision produit) peut nécessiter un aller-retour utilisateur.
2. **Vérifier sur pièces avant de coder ET avant de cocher** — rouvrir chaque fichier/ligne
   cité ci-dessous ; le code a pu bouger, et surtout l'endpoint MS peut avoir re-changé.
3. **Statuts** : `[x]` fait+vérifié / `[~]` couvert ailleurs (référence) / `[!]` non traité
   (justification écrite obligatoire au Journal §5).
4. **Zéro fix opportuniste hors périmètre** : noter en §6 « Découvertes », ne pas traiter.
5. **Clôture d'un lot** : gate passé → cases statuées → ce fichier MAJ → entrée thought_log →
   point d'étape utilisateur.
6. **Git** : 1 branche dédiée (ex. `fix/auth-device-flow-sisu-404`), N commits ; jamais sur
   `main` ; demander avant tout commit/push (push main = deploy prod auto).
7. **Pas de nouveau flag laissant une feature OFF** (règle projet §11) — la correction se
   livre active. Le champ `auth_provider` existe déjà et n'est PAS un kill-switch de feature.

---

## 1. CONSTAT — cause racine (vérifiée sur pièces le 2026-07-11)

**Symptôme utilisateur** : premier `make restart` sur un PC neuf → le wizard reste bloqué
sur le spinner « Démarrage du Device Code Flow… », indéfiniment.

**Chaîne observée** :

1. `POST /api/v1/auth/device-flow/start` → **HTTP 500 `msal_init_error`**
   (logs `logs/general.log`, plusieurs occurrences 2026-07-11 20:31→20:39).
2. Provider actif au boot = **SISU** (défaut) — logs « buildTokenProvider: SISU provider
   activé (défaut) » ; sélection dans
   [cmd/server/main.go:87-111](../apps/go-api/cmd/server/main.go#L87-L111).
3. `SISUProvider.InitDeviceFlow`
   ([sisu_provider.go:112-168](../apps/go-api/internal/platform/auth/sisu_provider.go#L112-L168))
   enchaîne 3 appels réseau :
   - `requestDeviceToken` (`device.auth.xboxlive.com`) → **OK** (« Device Token obtenu »),
   - `StartXboxDeviceCode` → **ÉCHEC**,
   - `initSISUSession` (non atteint).
4. `StartXboxDeviceCode`
   ([xbox_device_code.go:49-116](../apps/go-api/internal/platform/auth/xbox_device_code.go#L49-L116))
   fait `POST https://login.live.com/oauth20_connect/device`
   (const `xboxDeviceCodeURL`, [xbox_device_code.go:25](../apps/go-api/internal/platform/auth/xbox_device_code.go#L25))
   → **HTTP 404, corps vide** (header `PPServer` présent = Passport atteint, chemin introuvable).

**Reproductions curl (indépendantes du binaire)** :
- `POST login.live.com/oauth20_connect/device` avec `scope=Xboxlive.signin Xboxlive.offline_access`
  → **404**.
- même endpoint avec `scope=service::user.auth.xboxlive.com::MBI_SSL` → **404**.
- Comparaison : `POST login.microsoftonline.com/consumers/oauth2/v2.0/devicecode` avec le
  client MSAL `e1cb35ab-…` → **200** (renvoie `user_code`/`device_code`/`verification_uri`).

**Conclusions** :
- **A** — L'endpoint natif Xbox `oauth20_connect/device` est retiré/déplacé côté Microsoft.
  La panne est **globale** (tout onboarding SISU distribué), PAS spécifique au PC de Guillaume.
- **B** — Le chemin MSAL (`consumers/oauth2/v2.0/devicecode`) fonctionne. Le contournement
  local `auth_provider=msal` (déjà posé dans `app_settings.json`, gitignored) débloque
  l'onboarding sur cette machine après restart.
- **C** — Bug UX indépendant : `useStartDeviceFlow` est appelé sans `onError` dans
  [StepDeviceCode.tsx:58-71](../apps/web/src/features/setup/StepDeviceCode.tsx#L58-L71) ; le
  garde `startFlow.isPending || (!status && !deviceFlowUserCode)`
  ([StepDeviceCode.tsx:119](../apps/web/src/features/setup/StepDeviceCode.tsx#L119)) laisse le
  spinner tourner à l'infini quand `start` échoue → l'erreur est avalée. Même schéma à
  vérifier dans `XboxLoginPage.tsx`.
- **D** — Angle mort de test : `xbox_device_code_test.go` et `sisu_provider_test.go` injectent
  des URLs `httptest` mockées → ils ne touchent JAMAIS l'endpoint réel, donc n'ont pas pu
  détecter le 404. Aucun smoke/health ne valide la joignabilité réelle du device endpoint.

---

## 2. DÉCISION PRODUIT REQUISE (à trancher avec Guillaume avant le lot C)

L'app est distribuée à des self-hosters qui ne peuvent pas tous enregistrer une app Azure —
c'est la raison d'être historique de SISU (cf. commentaire
[main.go:79-86](../apps/go-api/cmd/server/main.go#L79-L86)). Trois stratégies non exclusives :

- **Option 1 — Réparer SISU** : trouver le nouvel endpoint natif Xbox device-code (ou le
  nouveau flux) et mettre à jour `xboxDeviceCodeURL` + le polling associé. Préserve le « zéro
  app Azure ». Coût : recherche endpoint MS non documenté, risque de re-cassure.
- **Option 2 — Basculer le défaut sur MSAL** : `buildTokenProvider` renvoie MSAL par défaut.
  Fiable aujourd'hui, mais impose l'app Azure `e1cb35ab` à tous les self-hosters (couplage au
  client_id LevelUp) — à valider vs modèle de distribution.
- **Option 3 — Fallback automatique SISU→MSAL sur échec de start** : SISU reste le défaut ;
  si `InitDeviceFlow` échoue (404/erreur réseau), bascule transparente sur MSAL pour la
  tentative courante + log `legacy_source_used`-style. Robuste, mais garde le code SISU vivant.

> Recommandation exécutant : **Option 3** (robustesse immédiate + préserve SISU si MS le
> réactive) couplée à un ticket de recherche pour l'Option 1. À CONFIRMER par Guillaume —
> ne pas coder le lot C avant décision.

---

## 3. LOTS (ordre d'exécution)

### Lot A — Fix UX : surfacer l'échec de `start` (indépendant de la décision produit)
Priorité haute : corrige le « spinner infini » quelle que soit la stratégie retenue.
- Ajouter un `onError` (ou dériver `startFlow.isError`) dans
  [StepDeviceCode.tsx](../apps/web/src/features/setup/StepDeviceCode.tsx) : un échec de
  `useStartDeviceFlow` bascule sur l'UI d'erreur existante (§bloc `status==='failed'`) avec
  bouton « Réessayer » (`handleRetry` existe déjà), au lieu de rester sur le `Spinner`.
- Revoir la même condition de garde L119 pour ne pas masquer un état d'erreur.
- Vérifier/appliquer le même correctif dans
  [XboxLoginPage.tsx](../apps/web/src/features/auth/XboxLoginPage.tsx) si le schéma s'y répète.
- Tests : compléter `SetupPage.test.tsx` / `XboxLoginPage.test.tsx` avec un cas « start renvoie
  500 → message d'erreur + bouton Réessayer affichés » (garde-rail anti-régression du spinner).
- **Gate** : `make check-types` + `make test-web` verts.

### Lot B — Décision produit (§2)
- Présenter §2 à Guillaume, acter l'option (1/2/3). Consigner la décision au Journal §5.
- **Gate** : décision écrite + option retenue notée dans ce fichier.

### Lot C — Implémentation de la stratégie retenue (dépend du lot B)
Selon l'option :
- **Si Opt.1** : identifier le nouvel endpoint natif (curl exploratoire), mettre à jour
  `xboxDeviceCodeURL` + `pollXboxDeviceCode`, ré-tester bout-en-bout.
- **Si Opt.2** : `buildTokenProvider` défaut → MSAL ; MAJ commentaire main.go + doc onboarding ;
  vérifier que l'app Azure est documentée comme pré-requis self-host.
- **Si Opt.3** : wrapper qui tente SISU puis MSAL sur erreur d'`InitDeviceFlow`, avec log
  structuré du basculement. Pas de swallow d'erreur (logger la cause SISU avant fallback).
- Dans tous les cas : logger explicitement l'erreur d'`InitDeviceFlow` dans
  [auth.go:127-135](../apps/go-api/internal/api/handlers/auth.go#L127-L135) (aujourd'hui
  stockée dans l'attempt mais jamais loggée — swallowed error, anti-pattern CLAUDE.md §10).
- **Gate** : `cd apps/go-api && go test ./...` vert + onboarding device-flow réussi en réel
  (obtenir un `user_code`, aller au bout jusqu'à `authorized`).

### Lot D — Garde-rail anti-régression endpoint + doc
- Ajouter un check de joignabilité de l'endpoint device-code réel : soit un test taggé
  `integration` (opt-in réseau) qui asserte HTTP 2xx sur le start, soit une sonde de santé
  au boot qui logge un WARN si l'endpoint device-code répond 404/5xx (sans bloquer le boot).
  Objectif : la prochaine cassure MS est visible immédiatement, pas via un spinner utilisateur.
- MAJ doc onboarding (`docs/INSTALL.md` / `docs/FR/INSTALL.md`, `docs/CONFIGURATION.md`) :
  documenter `auth_provider` (`sisu` défaut vs `msal`) et le contournement.
- **Gate** : test/sonde en place et vert ; doc FR+EN à parité (règle §15).

---

## 4. TRACKER (aucune case vide à la clôture)

- [x] A1 — `onError` surfacé dans StepDeviceCode via helper `startDeviceFlow()` +
      état `startError` (message i18n `common.setup.device_start_failed`, ou
      `err_demo` si `demo_mode`) → bascule sur l'UI d'erreur + « Réessayer ».
      Bonus DRY : les 3 copies d'`onSuccess` (montage/récup/retry) factorisées.
- [x] A2 — garde spinner (ex-L119) revue : early-return `if (startError)` placé
      AVANT `if (startFlow.isPending || (!status && !deviceFlowUserCode))` — le
      spinner ne peut plus masquer un start en échec.
- [x] A3 — XboxLoginPage surfe DÉJÀ l'échec de start (`startError`, vérifié sur
      pièces + au navigateur 2026-07-13 : 500 simulé → message + « Réessayer »,
      pas de spinner) ; garde-rail de régression ajouté (`XboxLoginPage.test.tsx`).
      Le spinner single-flight a été corrigé côté SERVEUR le 2026-07-13 (Journal).
- [x] A4 — tests web « start 500 → erreur + Réessayer » : `SetupPage.test.tsx`
      (StepDeviceCode) + `XboxLoginPage.test.tsx`. Asserte aussi l'absence du
      spinner de démarrage.
- [x] A5 — gate lot A vert : check-types (tsc -b) OK ; eslint 0 erreur/0 warning
      sur les fichiers touchés ; vitest complet 2159 passés / 14 skipped ;
      vérif navigateur `/login` (happy path + échec simulé).
- [~] B1 — décision SANS OBJET : la prémisse « endpoint retiré » était fausse (cf. Journal
      2026-07-13) — l'endpoint natif existe, c'est l'URL du code qui n'a jamais été la bonne.
      Option 1 exécutée de fait ; Options 2/3 sans objet. À confirmer par Guillaume (rayer D3
      de l'ETAT consolidé).
- [x] C1 — Option 1 de fait : `xboxDeviceCodeURL` corrigée `/oauth20_connect/device` →
      `oauth20_connect.srf` (vérif curl : .srf → 200 + device_code ; /device → 404 ;
      les DEUX variantes de scopes passent). Commit lot ops item 0 (2026-07-13).
- [x] C2 — log explicite `slog.ErrorContext` de l'erreur InitDeviceFlow ajouté dans
      handleStartDeviceFlow (même commit).
- [~] C3 — vérifié jusqu'à la page Microsoft « Se connecter » incluse (code généré,
      polling pending, page login.live.com rendue) — la complétion `authorized` exige les
      identifiants Microsoft de l'utilisateur : à confirmer par Guillaume à sa reconnexion.
- [x] C4 — gates du commit item 0 : go build/vet + tests handlers/auth + platform/auth
      verts ; lint --new-from-rev=origin/main 0 issue.
- [x] D1 — garde-rail joignabilité endpoint device-code : test taggé `integration`
      + opt-in réseau (env `LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK`) dans
      `xbox_device_code_reachability_integration_test.go` (package `auth`), qui
      exerce la constante RÉELLE `xboxDeviceCodeURL` via `StartXboxDeviceCode` et
      asserte 2xx + `device_code`. CHOIX justifié (en-tête du fichier) : test opt-in
      plutôt que sonde de santé au boot — coût runtime nul, zéro faux WARN offline,
      aucune dépendance réseau au démarrage, double-gate (tag + env) qui SKIP dans
      le CI anti-ART sans jamais flaker. Vérifié : SKIP sans env ; PASS en réel
      (endpoint `oauth20_connect.srf` joignable, user_code obtenu, expires_in=900 s).
- [x] D2 — doc onboarding MAJ en parité FR+EN (règle §15, même commit) sur 4 fichiers :
      `docs/INSTALL.md`, `docs/FR/INSTALL.md` (note « fournisseur d'auth SISU défaut vs
      MSAL »), `docs/CONFIGURATION.md`, `docs/FR/CONFIGURATION.md` (sous-section
      « fournisseur de tokens » + ligne `auth_provider` dans la table `app_settings`).
- [x] D3 — contournement local `auth_provider=msal` (posé le 11/07 dans
      `app_settings.json`, gitignored) VÉRIFIÉ SUR PIÈCES : la clé vaut aujourd'hui
      `""` (= SISU défaut) — le contournement a déjà été retiré (SISU refonctionne).
      Rien à supprimer ; consigné au rapport de clôture.

---

## 5. JOURNAL

- **2026-07-11** — Investigation initiale. Cause racine tracée (SISU `oauth20_connect/device`
  → 404 ; MSAL `consumers/…/devicecode` → 200). Contournement local `auth_provider=msal`
  posé dans `app_settings.json` (gitignored) — actif au prochain `make restart`. Plan rédigé.
  Aucun code applicatif modifié à ce stade.
- **2026-07-13** — REQUALIFICATION (lot ops/qualité item 0, branche `chore/lot-ops-qualite`,
  vérification du mode auth local `xbox`). La conclusion A du §1 (« endpoint retiré/déplacé,
  recherche coûteuse ») était fausse sur un point clé : l'endpoint natif Xbox device-code
  EXISTE — c'est `https://login.live.com/oauth20_connect.srf` ; l'URL du code
  (`/oauth20_connect/device`) n'a JAMAIS été la bonne (introduite telle quelle par le commit
  SISU `16e7d2922`, jamais exercée en réel : les tests injectent des URLs mockées, conclusion
  D du §1). Vérifié par POST direct : `.srf` → 200 + device_code/user_code/verification_uri
  (avec `service::user.auth.xboxlive.com::MBI_SSL` ET `Xboxlive.signin Xboxlive.offline_access`) ;
  `/device` → 404 corps vide. **Fix livré** (= Option 1, triviale) : constante corrigée,
  flow re-testé bout-en-bout au navigateur (page /login anonyme → code + lien
  `login.live.com/oauth20_authorize.srf` SISU/PKCE → page Microsoft « Se connecter » rendue).
  **Découverte + fix bonus (même commit)** : 2e cause de spinner infini sur /login — le
  single-flight de `handleStartDeviceFlow` répondait au start concurrent (double-fire des
  effets React en dev, 2e onglet) AVANT que la requête créatrice ait rempli la tentative →
  200 avec user_code VIDE, l'UI écrasait le code et restait sur « Génération du code… ».
  Fix : `waitDeviceFlowReady` (attente bornée 15 s, lecture par Snapshot, propagation de
  l'échec créateur, 503 retryable au timeout) + 2 tests de régression
  (`auth_device_flow_singleflight_test.go`). C2 (swallowed error) réglé au passage.
  Restent ouverts : lot A (StepDeviceCode), D1/D2 (garde-rail + doc) — hors périmètre du
  lot ops. Le lot B (décision produit) est SANS OBJET.
- **2026-07-13 (soir)** — LOT A livré (branche `fix/auth-deviceflow-lots-ad`).
  `StepDeviceCode.tsx` : le start en échec (500 `msal_init_error` / 503 retryable du
  single-flight) était avalé — la garde spinner `startFlow.isPending || (!status &&
  !deviceFlowUserCode)` restait vraie indéfiniment. Correctif : état `startError`
  alimenté par un `onError` centralisé dans le helper `startDeviceFlow()` (qui remplace
  les 3 copies inline d'`onSuccess`), et early-return de l'UI d'erreur + « Réessayer »
  AVANT la garde spinner. Reset de `startError` dans `handleRetry` (event handler) —
  PAS dans le helper, pour éviter un setState synchrone dans l'effet de montage
  (warning `react-hooks/set-state-in-effect`). Nouvelle clé i18n FR+EN
  `common.setup.device_start_failed`. Tests : `SetupPage.test.tsx` +
  `XboxLoginPage.test.tsx` (start 500 → message + « Réessayer », pas de spinner).
  Gate vert (check-types, eslint 0/0, vitest 2159 passés). Vérif navigateur `/login` :
  happy path (code + `microsoft.com/link`) OK ; échec simulé (fetch override 500) →
  message d'erreur + « Réessayer », aucun spinner infini ; reload propre restaure le code.
- **2026-07-13 (soir)** — LOT D livré + PLAN CLOS. D1 : garde-rail réseau opt-in
  `xbox_device_code_reachability_integration_test.go` (tag `integration` + env
  `LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK`) — exerce la vraie constante `xboxDeviceCodeURL`,
  referme le blind spot « tests mockés » (conclusion D §1). Choisi vs sonde de boot :
  coût nul, pas de faux WARN offline, pas de dépendance réseau au démarrage, SKIP
  garanti dans le gate anti-ART. Vérifié SKIP-sans-env + PASS-en-réel. D2 : doc
  `auth_provider` (SISU défaut / MSAL fallback config-only) en parité FR+EN sur
  INSTALL + CONFIGURATION (4 fichiers, même commit). D3 : contournement local
  `auth_provider=msal` déjà retiré (`app_settings.json` = `""`), rien à faire.
  Gate : go vet `-tags=integration` + tests package auth verts ; golangci-lint
  `--new-from-rev=feat/explorer-briefing-cards --build-tags=integration` = 0 issue.
  Lot B (décision produit) SANS OBJET (endpoint jamais retiré, cf. requalif du 13/07).
  Tracker §4 intégralement statué. Plan déplacé en `.ai/V7/`.

## 6. DÉCOUVERTES (hors périmètre — à ne pas traiter ici)

- **Mock MSW `device-flow/start` sur l'alias déprécié** : le contrat réel est sain — le
  backend renvoie `expires_in` (vérifié au navigateur : compte à rebours actif) et le front
  lit `data.expires_in`. En revanche le mock de test (`handlers.ts`) renvoie
  `expires_in_seconds` (l'alias marqué `@deprecated` dans `types.ts`), donc les tests
  vitest n'exercent jamais le compte à rebours (part de `null`). Cosmétique/test-only,
  pré-existant, hors périmètre A/D — aligner le mock sur `expires_in` un jour.
