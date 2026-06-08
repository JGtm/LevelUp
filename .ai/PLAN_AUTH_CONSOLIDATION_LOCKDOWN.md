# PLAN — Consolidation Auth (SISU vs MSAL) + 2ᵉ connexion fluide + Lockdown d'instance

> **Statut** : PLAN / à arbitrer. Aucun code. Fait suite à la PR bugfix `fix/session-cookie-secure-scheme` (issue #22).
> **Date** : 2026-06-08.
> **Pré-requis** : la PR bugfix (cookie Secure par schéma + récupération onboarding) doit être mergée d'abord — c'est elle qui débloque **tous** les flows interactifs (MSAL device, SISU device, redirect SSO passaient tous par le même cookie/AttemptStore cassé).

---

## 0. Objectifs (demande utilisateur)

1. **Lockdown d'instance** : pouvoir bloquer les nouvelles inscriptions, les nouvelles identités (connexions créant un compte) et la création de nouvelles BDD joueur — sans casser les utilisateurs/joueurs existants.
2. **Process d'auth clarifié** : rendre la **2ᵉ connexion (et suivantes) la plus fluide possible**. Le mot de passe est reconnu comme un irritant potentiel → à proposer en option, pas à imposer.
3. **Trancher MSAL vs SISU** : les deux coexistent, c'est redondant. Décision attendue sur conseil.
4. **Testabilité SISU/SSO Xbox** : recenser ce que Microsoft permet pour tester facilement.

---

## 1. État des lieux (cartographie factuelle)

### 1.1 Deux providers, mutuellement exclusifs au runtime

| | **MSALProvider** | **SISUProvider** |
|---|---|---|
| Fichier | `internal/platform/auth/provider.go:99` | `internal/platform/auth/sisu_provider.go:67` |
| App Azure requise | **OUI** (client_id à enregistrer) | **NON** — client_id Xbox natif `000000004c20a908` (`sisu_provider.go:24`) |
| Sélection | `app_settings.json:auth_provider` = `"msal"` (défaut) ou `"sisu"` → `cmd/server/main.go:buildTokenProvider()` (~l.68-85) | idem |
| Silent refresh (cache MSAL) | **OUI** (`TrySilentRefresh` via cache JSON) | **NON** (`sisu_provider.go:217` retourne `"",nil`) |
| Refresh via OAuth RT rotation | OUI (`login.microsoftonline.com`) | OUI (`login.live.com`, même mécanisme) |
| Flow redirect (Authorization Code, 1-clic) | OUI (`/auth/xbox/login` + `/callback`) — **exige** `OAuthRedirectURI` + plateforme « Web » Azure | **NON** (device code uniquement) |
| `GetFlowType()` | `"msal"` | `"sisu"` |

> Les deux implémentent la même interface `TokenProvider`. **Un seul est instancié au boot.** Ils ne tournent jamais ensemble ; le « both » est purement du code maintenu en double. `GetFlowType()` n'est lu que pour adapter l'UX front (`deviceFlowStartResponse`), aucun branchement métier.

### 1.2 Points d'entrée auth / inscription / création BDD

| Action | Route | Handler | Verrous actuels |
|---|---|---|---|
| Inscription password | `POST /auth/register` | `UserAuthHandler.Register` (`user_auth.go:110`) | 1er user→admin sans invite ; mode xbox→register bloqué hors bootstrap ; `RegistrationMode` closed/invite/open |
| Connexion password | `POST /auth/login` | `UserAuthHandler.Login` (`user_auth.go:52`) | mode xbox→non-admin = 403 `password_login_admin_only` |
| Device flow (Halo) | `POST /auth/device-flow/start` + `GET /…/{id}` | `AuthHandler` (`auth.go:72/120`) | `DemoMode`→422 |
| Redirect SSO | `GET /auth/xbox/login` + `/callback` | `XboxOAuthHandler` (`auth_xbox_oauth.go`) | `DemoMode`→422 ; câblé seulement si `AuthMode=xbox && OAuthRedirectURI!=""` (`server.go:624`) |
| Création user depuis SSO | (post-flow) | `XboxSSOLinkStrategy.OnAuthSuccess`→`CreateFromXbox` (`xbox_auth_service.go:83`, `userstore/store.go:368`) | aucun — crée le user si XUID inconnu |
| Création profil/BDD joueur | `POST /setup/players` | `SetupHandler.CreatePlayer` (`setup.go:51`) → `profile_service.go:46` (`os.MkdirAll` player dir) | `can_self_provision` (app_settings)→403 `provisioning_disabled` |

### 1.3 Switches de gating déjà présents (réutilisables)

- `DemoMode` (`LEVELUP_DEMO_MODE`) — bloque **tous** les flows auth (→422) **et** bascule sur fixtures. Trop lourd pour un simple lockdown.
- `RegistrationMode=closed` (`LEVELUP_REGISTRATION`) — bloque le register **password** uniquement.
- `can_self_provision=false` (app_settings) — bloque `POST /setup/players`.
- `config.Validate()` / `SecurityWarnings()` — garde-fou boot.
- **Pas de flag « instance fermée / maintenance » unique.** → à créer.

---

## 2. DÉCISION 1 — MSAL vs SISU

### 2.1 Analyse

L'audience cible est **auto-hébergée** (self-host : localhost, LAN, petit VPS), chaque utilisateur s'authentifie avec **son propre** compte Microsoft/Xbox.

- **La friction #1 de MSAL = l'enregistrement d'une app Azure** (créer l'app, récupérer le client_id, configurer une plateforme « Web » + redirect URI public). SISU **supprime entièrement** cette étape (client_id Xbox natif). C'est le plus gros levier de fluidité d'onboarding.
- **L'avantage UX de MSAL = le flow redirect 1-clic** (`/auth/xbox/login`). Mais il **exige un redirect URI public en HTTPS** → indisponible pour la majorité des self-hosters (localhost/LAN) → ils retombent de toute façon sur le device code. L'avantage est donc **caduc** pour la cible.
- **Refresh** : les deux reposent in fine sur le refresh_token `login.live.com`. SISU n'a pas le cache MSAL mais `TryOAuthRefreshWithRotation` capture/rote/persiste le RT — suffisant. Le wiring de refresh auto (ex-PR 2.5c / `RefreshUserXSTS`) est à faire **dans les deux cas**.

### 2.2 Recommandation

> **Consolider sur SISU comme provider interactif par défaut.** MSAL conservé en code, désactivé côté UI. Le flow **Authorization Code (redirect)** reste **opt-in opérateur** (activé seulement si un redirect URI Azure est configuré).

**Argument décisif = LevelUp est DISTRIBUÉ à d'autres self-hosters** (l'issue #22 vient d'un tiers, Du1Bz, qui a téléchargé le ZIP). On **ne peut pas** exiger de chaque utilisateur qu'il enregistre une app Azure + configure un redirect URI public. SISU (client_id Xbox natif, **zéro setup Azure**, marche sur localhost/LAN/VPS/Tailscale) est donc le bon **défaut**. L'Authorization Code reste disponible pour l'opérateur qui *a* un domaine stable et le veut (meilleure fluidité re-login, sans MDP), mais ce n'est pas le chemin par défaut.

**Décision end-user (cf. §3.6)** : SISU + **MDP opt-in** couvre la fluidité ; Auth Code = bonus opt-in pour instances à domaine stable.

**Pourquoi pas supprimer MSAL tout de suite** : SISU utilise un client_id Xbox **non officiel/non documenté** (même approche que les outils communautaires Halo : OpenSpartan, grunt-api…). Risque résiduel : Microsoft pourrait modifier ce client_id. Garder MSAL en filet de secours (derrière `auth_provider=msal`) pendant la transition évite un point de défaillance unique, sans coût de maintenance actif (SISU est le seul chemin testé/onboardé).

**Corollaires si SISU par défaut** :
- Le flow **redirect Authorization Code** (`/auth/xbox/login`, dépendant d'Azure) devient inutile pour la cible → le **device code** est le seul flow interactif. Décider : on garde le redirect derrière config Azure (power users) ou on le retire ? (cf. §6 points ouverts).
- Documenter le risque client_id + la procédure de bascule `auth_provider=msal` en cas de panne SISU.

⚠️ **À arbitrer** : OK pour « SISU défaut + MSAL fallback déprécié », ou tu veux **supprimer MSAL** franchement (zéro dual-maintenance, mais zéro filet) ?

---

## 3. DÉCISION 2 — Process d'auth & 2ᵉ connexion fluide

### 3.1 Comment marche la 2ᵉ connexion aujourd'hui

- **Session = cookie roulant 7 jours** (`session/store.go:37`, `Touch` à chaque requête). Tant que l'utilisateur revient dans les 7 jours → **rien à refaire**, il reste connecté.
- **Au-delà / nouvel appareil / logout** → il doit re-passer par Microsoft. En SISU = re-saisir un device code (friction). En MSAL+redirect = 1 clic (mais config Azure requise).
- Les **tokens de données Halo** (sync) se rafraîchissent **séparément et silencieusement** via le RT (rotation persistée) — indépendant de la session app.

### 3.2 Cible : 3 couches distinctes

| Couche | Mécanisme cible | Action utilisateur |
|---|---|---|
| **Session app** (gate l'UI) | cookie roulant 7j | aucune (retour < 7j) |
| **Re-login app** (session expirée) | **mot de passe optionnel** (instantané, sans round-trip MS) **OU** device code (fallback) | 1 saisie MDP, ou re-code |
| **Tokens données Halo** (sync) | refresh silencieux (RT rotation) | aucune ; re-consent Microsoft **seulement** si le RT meurt (révoqué / 90j inactivité / changement MDP MS) |

### 3.3 Le mot de passe optionnel (réponse à l'irritant)

- **Proposé** en fin d'onboarding (« Définir un mot de passe pour te reconnecter plus vite ? » → skippable). Primitives déjà là : bcrypt (`userstore/store.go:295`), `ResetPassword` admin.
- Si défini → relâcher `password_login_admin_only` pour cet utilisateur (login MDP autorisé même en mode xbox).
- **N'enlève jamais** le besoin de re-consent Microsoft périodique pour les données → prévoir une **bannière « reconnecte ton compte Xbox »** quand le RT est mort (rare). Avec SISU/MSAL, une seule ré-auth device code re-sème tout le RT.

### 3.5 Précision importante : ce que SISU fait *réellement* (vs intuition « redirect »)

SISU n'est **pas** un flow de redirection navigateur avec callback. C'est un **Device Code Flow** (`PollXboxDeviceCode`), avec deux variantes d'entrée fournies à l'utilisateur :
- un **code** (`user_code`) à saisir sur `login.live.com/devicelogin`, **et**
- une **URL cliquable** `MsaOauthRedirect` (`sisu_provider.go:150`) qui pré-porte le contexte de session SISU.

Déroulé d'une (re-)connexion SISU :
1. L'app affiche le lien cliquable (+ le code en repli).
2. L'utilisateur ouvre le lien → **Microsoft gère la session** : s'il a une session MS active, c'est juste un écran de **consentement à approuver** ; sinon il se connecte d'abord à Microsoft. *(Cette partie n'est effectivement « pas gérée par nous ».)*
3. L'app **détecte la complétion par polling** (pas de callback/redirect vers l'app).

**Conséquences (correction du modèle « autorise une fois puis session transparente »)** :
- ✅ **Vrai pour le régime permanent** : après la 1ʳᵉ autorisation, tant que l'utilisateur revient **dans les 7 jours** (cookie roulant) **et** que le refresh_token vit, il ne fait **rien** — la session app tient et les tokens données se rafraîchissent en silence.
- ⚠️ **Faux pour le « 1-clic si session MS active »** : ce confort-là (redirection instantanée qui rétablit la session app **sans** écran ni saisie) est le **flow Authorization Code (Azure)** qu'on retire. En SISU, **chaque (re-)connexion interactive demande au minimum un clic + une approbation**, et l'app apprend par polling. C'est « simple » (1 clic + approuver), pas « zéro interaction ».
- ⚠️ **Le RT persisté ne ressuscite PAS la session navigateur** : à l'expiration du cookie (> 7 j), le RT garde les **données** vivantes mais ne peut pas re-authentifier le **navigateur** tout seul (le cookie était le seul lien navigateur↔user). → il faut **une preuve d'identité** : SISU device code **ou** le **MDP opt-in**.

➡️ **C'est exactement pourquoi le MDP opt-in (décision #5) est le vrai levier de fluidité** pour les re-connexions post-expiration : instantané, sans round-trip Microsoft. SISU device-code reste le repli. Le flux est donc « optimal » au régime permanent, et fluide en re-login grâce au MDP — sans dépendre du redirect Azure.

### 3.6 SISU vs Authorization Code — recommandation end-user

Le vrai comparatif (corrige la confusion « code vs clic ») :

| | **SISU** (device code + lien cliquable) | **Authorization Code** (MSAL redirect) |
|---|---|---|
| Setup opérateur | **zéro** | app Azure + redirect URI **par instance** |
| Topologie | localhost / LAN / VPS / Tailscale | domaine stable HTTPS (ou localhost) requis |
| 1ʳᵉ auth end-user | ouvrir lien + confirmer | redirect → sign-in + consentement (1 fois) |
| Re-login (après 7 j) | ouvrir lien + confirmer | **souvent instantané** (0 écran si session MS active) |
| MDP nécessaire pour fluidité ? | oui (opt-in) le rend instantané | non |
| Légitimité | client_id Xbox **non officiel** (risque) | officiel/supporté |

**Règle de décision** : la seule vraie question = *« chaque instance a-t-elle un URL de redirect stable et enregistrable dans Azure ? »*
- App **distribuée** à des tiers (notre cas) → **NON garanti** → **SISU** (défaut).
- Instance **unique à domaine fixe** + opérateur OK pour 1 setup Azure + veut zéro MDP → **Authorization Code** (opt-in).

**Recommandation pour LevelUp : SISU par défaut + MDP opt-in.** Le surcoût end-user de SISU (un clic + confirmation) n'apparaît qu'au re-login post-7-jours ; le MDP l'annule pour qui le veut ; et le refresh des **données** est déjà silencieux (`RefreshLoop`). L'avantage « instantané » d'Auth Code est réel mais marginal et ne justifie pas d'imposer un setup Azure à chaque téléchargeur.

### 3.4 Refresh auto — état réel (corrigé) + ce qui manque

**Ce qui existe déjà (corrige une affirmation antérieure)** :
- `RefreshLoop` (`refresh_loop.go`) **est câblé** : goroutine 5 min qui, sur le `TokenStore` du tracker watcher, refresh l'access_token OAuth (< 10 min de marge) **et** le XSTS (< 20 min), avec **rotation du refresh_token persistée** (`ExchangeRefreshTokenWithRotation`) + **miroir** vers `MultiUserTokenStore` (PR 2.5b). → **le XSTS est déjà auto-géré**, l'utilisateur n'a rien à faire dessus.
- Par-utilisateur (multi-user), le refresh se fait à la demande via `RefreshHaloTokensViaStoreFirst` au moment du sync de chaque joueur.

**Ce qui manque** :
1. **Notification de mort du refresh_token** : aujourd'hui, RT révoqué/expiré → **simple WARN log** (`refresh_loop.go:149`), **aucune** alerte utilisateur. L'infra `internal/notify` (Discord) existe mais n'est PAS branchée sur l'auth.
2. **Bannière UI « reconnecte ton compte Xbox »** : aucune. L'état est seulement *pollable* via le statut watcher (`TokenValid`/`TokenExpiresAt`).
3. **Unification** : le bouton `/watcher/auth/start` (device flow watcher) ne capture pas le RT/cache → re-clic. À unifier sur le `MultiUserTokenStore` + capture à l'onboarding, plutôt que dupliquer la logique.
4. (`RefreshUserXSTS` `refresh_user_xsts.go:37` reste la variante per-user non câblée — utile seulement si on veut un loop per-user dédié plutôt que le refresh on-sync.)

➡️ **PR-B** doit donc surtout : (a) **détecter** la mort du RT (retour vide/`invalid_grant`) et la **propager** comme un état `reauth_required` par joueur, (b) le **notifier — notification IN-APP en primaire** (bannière persistante via `/bootstrap`, avec bouton « reconnecter ») **+ Discord opt-in en complément** (`internal/notify`), (c) unifier le circuit watcher. Le refresh XSTS lui-même est déjà là.

---

## 4. DÉCISION 3 — Lockdown d'instance

### 4.1 Sémantique visée

Une **instance fermée** : les utilisateurs/joueurs **déjà connus** continuent de fonctionner (sessions, login, données), mais **aucune nouvelle identité ni nouvelle BDD** ne peut être créée.

> ⚠️ À clarifier avec toi : « bloquer les connexions » =
> - **(A) Instance fermée** (recommandé) : bloquer la création de **nouvelles** identités/BDD ; les existants se connectent encore. **OU**
> - **(B) Maintenance dure** : bloquer **toute** connexion (même existante) + lecture seule. Plus radical, lock-out total.
>
> Je pars sur **(A)** par défaut ci-dessous ; (B) serait un mode `maintenance` séparé.

### 4.2 Conception (mode A — instance fermée)

Nouveau flag global unique : `instance_locked` (app_settings.json) + override env `LEVELUP_INSTANCE_LOCKED`. Lecture centralisée (comme `DemoMode`). Effets quand `true` :

| Point d'entrée | Comportement verrouillé |
|---|---|
| `POST /auth/register` | 403 `instance_locked` (tous modes, y compris bootstrap 1er user — ou exempter le tout 1er ? cf. points ouverts) |
| SSO `OnAuthSuccess` / `CreateFromXbox` | XUID **inconnu** → 403 `instance_locked` (pas de création) ; XUID **connu** → login normal |
| `POST /setup/players` | 403 `instance_locked` (en plus de `can_self_provision`) |
| Provisioning 1ʳᵉ BDD joueur (pool discovery) | refuse la création d'un nouveau `data/.../players/{gt}/` ; sert les joueurs existants |
| Sessions / login MDP existants / données | **inchangés** |

- **Découplage volontaire** de `DemoMode` (qui swap les fixtures) et de `RegistrationMode` (password only). `instance_locked` agit transversalement sur **toutes** les voies de création d'identité/BDD.
- **Toggle** : env (boot) + endpoint admin `PATCH /admin/instance-lock` (protégé `RequireAdmin`) pour basculer à chaud.
- **Observabilité** : log `instance_locked` à chaque refus + état exposé dans `/bootstrap` (pour un bandeau UI « inscriptions fermées »).

---

## 4bis. Contrôle d'ownership / anti-IDOR (remonté par revue collègue)

**Besoin** : garantir côté back qu'un appel API ne peut pas cibler les données d'un **autre** utilisateur que celui connecté (un attaquant connecté change le slug/identité dans la requête).

**Ce qui existe déjà** :
- `middleware.RequirePlayerOwnership` (ADR 0024, `require_player_ownership.go`) = **chokepoint unique** monté sur le groupe `/players/{player_slug}/…` (`server.go:789`). Résout slug→xuid (sans ouvrir DuckDB), autorise si profil **possédé** par l'utilisateur courant, **famille/amis**, ou **admin** ; sinon **403 `player_forbidden`**.
- Admin/Watcher : `RequireAuth` + `RequireAdmin` (`server.go:479,640,724,1052`).
- Capabilities par titre : `RequireCapability`.

**Limite / à auditer (gap réel)** :
1. L'enforcement est **actif seulement si `authz.Enforced(demoMode, authMode)`** = OFF en `AuthMode=none` (défaut dev) et en demo. → en passant l'instance en `xbox` + lockdown, c'est ON. **Documenter que `none` = aucun contrôle.**
2. Le chokepoint couvre les routes identifiées par le **path** `player_slug`. **Toute route qui prend l'identité cible dans le BODY ou la QUERY** (et pas le path) **échappe** au middleware. → **audit à faire** : recenser les handlers qui lisent un gamertag/xuid hors path et **re-valider** contre la session.
3. **Principe transverse à acter** : *ne jamais faire confiance à une identité fournie par le client* ; dériver l'identité de la **session** (`sess.LinkedHaloIdentity` / `CurrentPlayerSlug`), jamais du payload, pour tout ce qui est « mes » données.

➡️ **Ajout à PR-A** (ou audit dédié) : (a) test e2e « user A ne peut pas lire/écrire les données de user B » (path ET body), (b) revue des handlers body/query-scoped, (c) doc du principe ci-dessus.

---

## 5. Testabilité SISU / SSO Xbox

### 5.1 Contrainte dure (confirmée Microsoft Learn)

- **ROPC** (le seul flow password-based automatisable sans navigateur) **n'est supporté que sur les tenants Entra, jamais sur les comptes personnels (MSA)**. Or **Xbox Live exige un MSA personnel avec profil Xbox** (gamertag) → on ne peut PAS utiliser un tenant de test Entra pour la chaîne XSTS Xbox.
- Conséquence : **il n'existe aucun harnais d'intégration automatisé officiel** pour la chaîne complète Microsoft→XSTS→Spartan avec un compte personnel. Les scopes requis : `XboxLive.signin` + `XboxLive.offline_access`, tenant `consumers`.

### 5.2 Stratégie de test recommandée

1. **Unitaire (déjà en place, à étendre)** : stubber `TokenProvider` (`stubTokenProvider`, `NewStubDeviceFlow` `provider.go:49`). Couvre toute la logique back (attempts, link strategy, lockdown, refresh decisioning) **sans réseau**.
2. **Compte de test réel** : un **MSA secondaire « burner »** avec un gamertag Xbox gratuit. C'est le seul moyen de valider la vraie chaîne end-to-end. À documenter dans un runbook (pas en CI).
3. **Intégration semi-automatisée (gated)** : capturer **une fois** un refresh_token réel (via `cmd/token-capture`), le stocker en secret local, et écrire un test d'intégration `//go:build integration` qui **rejoue le silent/RT refresh** (`RefreshHaloTokensViaStoreFirst`) — skip par défaut si le secret est absent (pattern déjà utilisé par les fixtures `jgtm_full_match`). Valide la partie refresh **sans** interaction humaine répétée.
4. **Device code = le plus testable manuellement** : pas de redirect URI, marche sur localhost. Un humain saisit le code une fois ; le reste (polling, exchange, persistance) est observable via `logs/auth.log`.
5. **Ce que Microsoft NE fournit PAS** : pas de sandbox Xbox Live consumer, pas de comptes de test XSTS. (Les comptes sandbox Partner Center/XDP sont réservés aux éditeurs de titres et ne s'appliquent pas à un client non officiel.)

> Bottom line : **stub pour l'automatisé, burner MSA + RT capturé pour l'intégration manuelle/gated.** Ne pas chercher un CI vert sur la vraie chaîne Xbox — ce n'est pas supporté.

---

## 6. Découpage proposé (PRs) & points ouverts

### 6.1 Phasage

- **PR-A — Lockdown d'instance** (autonome, faible risque) : flag `instance_locked` + verrous (register, CreateFromXbox unknown-XUID, setup/players, provisioning) + endpoint admin + `/bootstrap` + tests stub.
- **PR-B — Refresh auto & unification tokens** : brancher `RefreshUserXSTS` (ReconnectManager) + unifier watcher sur `MultiUserTokenStore` + capture RT à l'onboarding. Tests refresh (stub + intégration gated).
- **PR-C — MDP optionnel & 2ᵉ connexion** : étape onboarding « définir un MDP » (skippable) + relâche `password_login_admin_only` pour users avec MDP + bannière « reconnecte Xbox » (RT mort). Tests front + back.
- **PR-D — Consolidation SISU** : SISU défaut, MSAL déprécié (env-only), doc risque client_id + runbook bascule. Décider du sort du flow redirect.

(Ordre indicatif ; A est livrable immédiatement et répond au besoin « bloquer les inscriptions ».)

### 6.2 Décisions arbitrées (2026-06-08)

1. **MSAL** : ✅ **conservé en code, désactivé côté UI** (pas de suppression). Le `auth_provider=msal` reste activable manuellement ; aucune entrée UI ne le propose.
2. **Flow redirect Authorization Code** : ✅ **non exposé par défaut** (SISU device-only) mais **reste activable en opt-in opérateur** si un redirect URI Azure est configuré (cf. §2.2 / §3.6). Code conservé (cohérent avec #1). Argument : LevelUp est distribué à des tiers → pas d'app Azure imposée.
3. **Lockdown** : ✅ **(A) instance fermée**. Activable via `LEVELUP_INSTANCE_LOCKED` (env.local) **ou** `instance_locked` (app_settings.json).
4. **Bootstrap 1er admin** : ✅ **exempté si 0 user existant** (sinon instance verrouillée à vide non amorçable).
5. **MDP** : ✅ **opt-in** en fin d'onboarding (jamais imposé).

---

## 7. Références code (ancrage)

- Providers : `internal/platform/auth/provider.go:99` (MSAL), `…/sisu_provider.go:67` (SISU) ; sélection `cmd/server/main.go:buildTokenProvider()`.
- Auth handlers : `internal/api/handlers/{auth.go, auth_xbox_oauth.go, user_auth.go}`.
- Link strategy / création user : `internal/service/xbox_auth_service.go:83`, `internal/platform/userstore/store.go:368` (`CreateFromXbox`).
- Création BDD : `internal/api/handlers/setup.go:51` → `internal/service/profile_service.go:46`.
- Refresh : `internal/platform/auth/refresh_user_xsts.go:37` (non câblé), `…/cli_refresh.go:48` (`RefreshHaloTokensViaStoreFirst`).
- Gating existant : `internal/config/config.go` (`DemoMode`, `RegistrationMode`, `Validate`/`SecurityWarnings`).
