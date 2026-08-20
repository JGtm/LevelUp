# V7 — Contrats API : Bootstrap, Setup, Capabilities & Jobs

> ⛔ Prérequis : lire d'abord :
> 1. [PLAN_V7_ONBOARDING_MASTER.md](PLAN_V7_ONBOARDING_MASTER.md)
> 2. [PLAN_V7_AUTH_SECURITY_PRINCIPLES.md](PLAN_V7_AUTH_SECURITY_PRINCIPLES.md)
>
> Date : 2026-04-13
> Rôle : **contrats d'API** — ce que le frontend consomme et ce que le backend doit exposer

---

## 0. Statut de transition

Le contrat produit cible de l'onboarding V7 est centré sur `GET /api/v1/bootstrap`.

Règle de transition :

- `GET /api/v1/setup/status` et `POST /api/v1/setup/smoke-test` peuvent subsister comme endpoints **legacy / transitoires / dev**
- le shell React final ne doit plus dépendre de `next_blocking_step` pour piloter l'onboarding produit
- tout nouveau comportement produit doit être exprimé dans bootstrap, `setup_state`, les capabilities et les endpoints dédiés (`/auth/device-flow/*`, `/setup/players`, `/sync/initial`, `/jobs/{job_id}`)

---

## 1. Bootstrap — `GET /api/v1/bootstrap`

Point d'entrée unique du shell React. Le frontend n'a **jamais** besoin de déduire un état à partir de plusieurs appels.

### Schéma `BootstrapResponse` — contrat cible

```
apps/api/app/schemas/bootstrap.py → BootstrapResponse
```

| Champ | Type | Rôle | Statut |
|-------|------|------|:------:|
| `setup_required` | `bool` | Bool legacy de compatibilité | ✅ actuel |
| `auth_state` | `"missing" \| "partial" \| "ready"` | État de l'auth Halo | ✅ actuel |
| `setup_state` | `"no_halo_link" \| "halo_linked_no_profile" \| "profile_ready_no_sync" \| "ready"` | Machine d'état produit de l'onboarding | ✅ partiel |
| `current_player` | `PlayerSummary \| None` | Joueur actif dans la session | ✅ actuel |
| `available_players` | `list[PlayerSummary]` | Tous les joueurs configurés | ✅ actuel |
| `locale` | `str` | Langue courante | ✅ actuel |
| `hints_visible_default` | `bool` | Affichage des hints | ✅ actuel |
| `feature_flags` | `FeatureFlags` | Flags de déploiement | ✅ actuel |
| `capabilities` | `CapabilityMap` | Capacités serveur | ✅ partiel |
| `settings_excerpt` | `SettingsExcerpt` | Préférences minimales pour le shell | ✅ actuel |
| `linked_halo_identity` | `HaloIdentitySummary \| None` | Identité Halo liée à la session pour confirmer le provisioning | ❌ cible Sprint 1 |
| `active_sync_job_id` | `str \| None` | ID du job sync actif (permet au frontend de reprendre le polling après refresh) | ❌ cible Sprint 3 |

### Champ réservé Phase 4 — `access_state`

`access_state` n'est **pas** dans le contrat MVP actuel exposé par le code. Si ce champ est ajouté plus tard, il devra modéliser l'accès applicatif quand la barrière externe sera remplacée ou complétée.

### `HaloIdentitySummary`

| Champ | Type | Rôle |
|-------|------|------|
| `gamertag` | `str` | Nom joueur résolu côté backend |
| `xuid` | `str` | Identifiant Xbox résolu côté backend |

### `setup_state` — machine d'état

```
no_halo_link ──[Device Code Flow OK]──→ halo_linked_no_profile
                                                                     │
                                             [Profil créé]───┘
                                                                     ▼
                                                      profile_ready_no_sync
                                                                     │
                                             [Sync initiale OK]──┘
                                                                     ▼
                                                                ready
```

### Logique de dérivation — contrat cible

| Condition | Résultat |
|-----------|----------|
| `auth_state == "missing"` | `no_halo_link` |
| `auth_state == "ready"` ET aucune identité Halo liée disponible pour la session / aucun profil local lié | `halo_linked_no_profile` |
| Profil local prêt ET marqueur de sync initiale absent dans `sync_meta` | `profile_ready_no_sync` |
| Profil local prêt ET marqueur de sync initiale présent | `ready` |

**Règle importante** : `profile_ready_no_sync` et `ready` reposent sur un marqueur persistant de sync réussie côté player DB, **pas** sur un simple count en `shared.match_participants`.

### Comportement en mode démo

En mode démo, l'onboarding est **entièrement court-circuité**. Le mode démo sert à explorer les capacités de l'app et visualiser des stats, pas à tester l'onboarding.

| Champ | Valeur en mode démo |
|-------|---------------------|
| `setup_state` | `"ready"` (toujours) |
| `auth_state` | `"ready"` (simulé) |
| `linked_halo_identity` | `null` |
| `capabilities.can_run_sync` | `false` |
| `capabilities.can_use_live_halo` | `false` |
| `capabilities.can_start_initial_sync` | `false` |

Le Device Code Flow, le provisioning et la sync initiale ne sont **pas accessibles** en mode démo.

### Migration des profils existants

Le rollout V7 doit prévoir un backfill des profils historiques pour initialiser le marqueur de sync réussie quand les données existent déjà. Sans cela, des joueurs déjà configurés seraient renvoyés à tort vers `profile_ready_no_sync`.

---

## 2. Capabilities — `CapabilityMap`

```
apps/api/app/schemas/common.py → CapabilityMap
```

| Champ | Type | Source | Statut |
|-------|------|--------|:------:|
| `can_read_local_data` | `bool` | Toujours `True` | ✅ actuel |
| `can_run_sync` | `bool` | `not demo_mode` | ✅ actuel |
| `can_use_live_halo` | `bool` | `not demo_mode` | ✅ actuel |
| `can_manage_settings` | `bool` | Toujours `True` | ✅ actuel |
| `can_reset_media_index` | `bool` | Toujours `True` | ✅ actuel |
| `can_view_media` | `bool` | `app_settings.media_enabled` | ✅ actuel |
| `can_self_provision` | `bool` | `app_settings.can_self_provision` | ✅ actuel |
| `can_start_initial_sync` | `bool` | `not demo_mode` | ✅ actuel |
| `can_manage_instance` | `bool` | Toujours `True` | ✅ actuel |

> **`auto_provision_from_halo_identity`** : déplacé en **Phase 4**. Pour le MVP single-user, l'auto-provision via l'identité Halo liée est le seul mode utile. Ce flag n'a de valeur que dans un scénario multi-user avec admin.

**Règle** : les capabilities sont informatives pour le frontend, mais l'autorisation réelle est toujours vérifiée côté API.

---

## 3. Device Code Flow — endpoints

### `POST /api/v1/auth/device-flow/start`

**Réponse** : `DeviceFlowStartResponse`

| Champ | Type |
|-------|------|
| `attempt_id` | `str` (UUID) |
| `user_code` | `str` |
| `verification_uri` | `str` |
| `verification_uri_complete` | `str \| None` |
| `expires_in_seconds` | `int` |
| `poll_interval_seconds` | `int` |

### Règles serveur

- requiert une session web valide
- vérifie la politique same-origin des routes mutantes
- applique un **single-flight** : une seule tentative pending active par session
- si une tentative pending active existe déjà pour la session, le backend peut retourner la tentative existante plutôt que d'en créer une nouvelle

### `GET /api/v1/auth/device-flow/{attempt_id}`

**Réponse** : `DeviceFlowStatusResponse`

| Champ | Type | Rôle |
|-------|------|------|
| `attempt_id` | `str` | ID de la tentative |
| `status` | `"pending" \| "authorized" \| "provisioned" \| "failed" \| "expired"` | État courant |
| `gamertag` | `str \| None` | Résolu quand `status=provisioned` |
| `xuid` | `str \| None` | Résolu quand `status=provisioned` |
| `error` | `ApiErrorSchema \| None` | Détails d'erreur si `status=failed` |

### Règles serveur

- une tentative n'est lisible que par la session qui l'a créée
- une tentative étrangère ou inconnue retourne comme inconnue (`404` recommandé)
- quand `status = "provisioned"`, le backend met à jour :
   - `session.auth_ready = True`
   - `session.linked_halo_identity = {gamertag, xuid}`

### Persistance des tentatives

État cible : ownership, expiration, purge et retry UX clairs.

Décision de phase :

- MVP onboarding : tentatives process-level acceptables si l'UX de retry est propre
- avant Phase 4 ou avant multi-worker : store persistant ou partagé requis

---

## 4. Création de profil — `POST /api/v1/setup/players`

**Requête** : `CreatePlayerProfileRequest`

| Champ | Type | Validation / usage |
|-------|------|--------------------|
| `gamertag` | `str` | 1-50 chars, alphanum + tirets + espaces ; sert de confirmation en mode `xbox` |
| `xuid` | `str \| None` | optionnel ; ignoré ou vérifié en mode `xbox`, jamais trusté seul |
| `profile_mode` | `"xbox" \| "azure_manual"` | mode de provisioning |

**Réponse** : `CreatePlayerProfileResponse`

| Champ | Type |
|-------|------|
| `player` | `PlayerSummary` |
| `db_created` | `bool` |
| `warnings` | `list[str]` |

### Guards requis

- `403` si `can_self_provision = false`
- `409` si `profile_mode = "xbox"` mais qu'aucune identité Halo liée n'est présente en session
- `409` si `gamertag` / `xuid` ne correspondent pas à l'identité Halo liée en session

### Effets de bord requis

En cas de succès, le backend doit :

- créer ou idempotemment retrouver le profil local
- mettre à jour `session.current_player_slug`
- rendre l'identité Halo liée exploitable pour la première sync
- persister le cache MSAL / état auth utile dans la player DB si nécessaire
- clarifier le contrat de `db_created` : DB réellement créée maintenant, ou initialisation garantie avant la première sync

### Idempotence

Deux appels successifs pour la même identité Halo ne doivent pas créer de doublons. Le contrat recommandé est un retour du profil existant si l'identité est déjà provisionnée.

---

## 5. Jobs longs — modèle enrichi

### Schéma `AsyncJobStatus` — contrat cible

```
apps/api/app/schemas/common.py → AsyncJobStatus
```

| Champ | Type | Statut |
|-------|------|:------:|
| `job_id` | `str` | ✅ actuel |
| `job_type` | `str` | ✅ actuel |
| `status` | `"queued" \| "running" \| "succeeded" \| "failed" \| "cancelled" \| "interrupted"` | ⚠️ valeur `interrupted` à ajouter dans la logique |
| `progress_pct` | `int \| None` | ✅ actuel |
| `current_step` | `str \| None` | ✅ actuel |
| `phase_key` | `str \| None` | ✅ actuel |
| `phase_label` | `str \| None` | ✅ actuel |
| `matches_done` | `int \| None` | ✅ actuel |
| `matches_total` | `int \| None` | ✅ actuel |
| `subtasks_done` | `int \| None` | ✅ actuel |
| `subtasks_total` | `int \| None` | ✅ actuel |
| `eta_seconds` | `int \| None` | ✅ actuel (optionnel — voir note ci-dessous) |
| `warnings` | `list[str]` | ✅ actuel |
| `started_at` | `datetime \| None` | ✅ actuel |
| `finished_at` | `datetime \| None` | ✅ actuel |
| `result` | `dict \| None` | ✅ actuel |
| `error` | `ApiErrorSchema \| None` | ✅ actuel |

> **Note** : le schéma enrichi est déjà complètement implémenté dans le code. Il reste à ajouter la valeur `"interrupted"` dans la logique du `JobStore` (distinct de `"cancelled"` qui est une annulation explicite).

### `eta_seconds` — champ optionnel

`eta_seconds` est présent dans le schéma mais **ne doit pas être promis comme fiable** dans les critères de succès. Sans un gros corpus de logs et vu les variations de puissance de calcul entre machines, un ETA faux est pire que pas d'ETA. Le frontend peut afficher `eta_seconds` s'il est non-null, mais ne doit pas en dépendre pour le routage ou la progression.

### Phases visibles de la sync initiale

Les phases `prepare`, `auth` et `verify` sont quasi-instantanées (< 1s). Pour l'UX, seules 3 phases sont réellement visibles :

| # | `phase_key` | `phase_label` (fr) | Contenu | Visible |
|---|-------------|---------------------|---------|:-------:|
| 1 | `prepare` | Préparation | Vérification DB, config, préconditions | ⚠️ très bref |
| 2 | `auth` | Connexion au service Halo | Vérification / acquisition des tokens | ⚠️ très bref |
| 3 | `fetch_matches` | Récupération de vos matchs | Appels API Halo | ✅ phase principale |
| 4 | `enrich` | Analyse des statistiques | Enrichissements et écritures secondaires | ✅ phase principale |
| 5 | `verify` | Vérification | Contrôles post-sync | ⚠️ très bref |
| 6 | `finalize` | Finalisation | Index, vues matérialisées, marqueur de fin | ✅ visible |

Le frontend garde les 6 phases dans le contrat mais peut regrouper visuellement 1-2 et 5 en une barre de progression générique, et ne mettre les compteurs détaillés que sur `fetch_matches` et `enrich`.

### Persistance et restart

Le `JobStore` est déjà **persistant** (`data/cache/jobs.json`, thread-safe, recovery au reload, purge lazy).

Ce qui reste à faire :

- Remplacer la recovery `running → cancelled` par `running → interrupted` avec warning explicite
- L'UI peut proposer "Relancer la sync" plutôt qu'afficher un faux état terminal
- Tant qu'une vraie reprise n'est pas implémentée, le job rechargé est marqué `interrupted`

## 6. Sync initiale — `POST /api/v1/sync/initial`

**Requête** : `StartInitialSyncRequest`

| Champ | Type | Rôle |
|-------|------|------|
| `player_slug` | `str` | Joueur à synchroniser |
| `max_matches` | `int` | Taille initiale de chargement |

**Réponse** : `AsyncJobStatus` (`202`)

### Guards requis

- `403` si `can_start_initial_sync = false`
- `404` si `player_slug` inconnu
- refus ou réutilisation contrôlée si une sync initiale active existe déjà pour ce joueur

### Single-flight recommandé

Une seule sync initiale active par `player_slug` à la fois.

Comportement recommandé : si une sync initiale existe déjà pour ce joueur, retourner le job actif existant plutôt que créer un doublon.

### Effet de succès

À la fin de la sync, le backend écrit le marqueur persistant qui fait passer le joueur de `profile_ready_no_sync` à `ready`.

---

## 7. Endpoints legacy de transition

| Endpoint | Rôle actuel | Rôle cible |
|----------|-------------|------------|
| `GET /api/v1/setup/status` | Wizard React existant + tests legacy | Compatibilité transitoire, pas source de vérité produit |
| `POST /api/v1/setup/smoke-test` | Vérification simplifiée post-installation | Dev / diagnostic, pas étape produit finale |

---

## 8. Écran d'entrée — contrat UX

Le frontend affiche du contenu conditionnel basé sur `setup_state` :

| `setup_state` | Carte principale | Action |
|---------------|-----------------|--------|
| `no_halo_link` | "Connecter mon compte Xbox" | → Device Code Flow |
| `halo_linked_no_profile` | "Créer mon profil" | → confirmation à partir de `linked_halo_identity` |
| `profile_ready_no_sync` | "Préparer mes données" | → `POST /sync/initial` |
| `ready` | "Ouvrir LevelUp" | → Dashboard |

### Informations de progression sync

- phase courante + label localisé
- compteurs concrets (37/200 matchs récupérés)
- estimation grossière si disponible (optionnel, pas fiable sans corpus de logs)
- warning non bloquant quand nécessaire
- résultat final : nombre de matchs importés + CTA d'entrée dans l'app

---

## 9. Scénarios d'erreur et recovery

> Ce chapitre spécifie le comportement attendu pour chaque classe d'erreur que l'onboarding peut rencontrer. Il couvre les erreurs synchrones (réponses HTTP immédiates) et asynchrones (erreurs dans les jobs longs).

### 9.1 Convention d'erreur API

Toute erreur est renvoyée dans l'enveloppe `ApiErrorSchema` déjà standardisée :

```json
{
  "code": "device_flow_expired",
  "message": "Le code a expiré. Veuillez relancer la connexion Xbox.",
  "retryable": true,
  "details": null,
  "field_errors": null
}
```

Le champ `retryable` est la **source de vérité** pour le frontend. Si `retryable=true`, le frontend affiche un CTA "Réessayer". Si `retryable=false`, le frontend affiche un message d'erreur final avec une action alternative (contacter l'admin, revenir en arrière, etc.).

### 9.2 Device Code Flow — erreurs

#### Erreurs synchrones (`POST /auth/device-flow/start`)

| Code HTTP | `code` | `retryable` | Cause | CTA frontend |
|:---------:|--------|:-----------:|-------|--------------|
| 422 | `demo_mode_unsupported` | `false` | Tentative en mode démo | Masquer le bouton (ne devrait pas arriver) |
| 429 | `rate_limit_exceeded` | `true` | Trop de tentatives consécutives | "Réessayez dans X secondes" (afficher `details.retry_after_seconds`) |
| 503 | `msal_unavailable` | `false` | MSAL non installé ou cassé | "Service d'authentification indisponible — contacter l'administrateur" |
| 500 | `internal_error` | `true` | Erreur inattendue serveur | "Réessayer" |

#### Erreurs asynchrones (polling `GET /auth/device-flow/{attempt_id}`)

Le polling retourne `DeviceFlowStatusResponse` avec `status` et optionnellement `error: ApiErrorSchema`.

| `status` renvoyé | `error.code` | Cause | CTA frontend |
|-------------------|--------------|-------|--------------|
| `expired` | — | L'utilisateur n'a pas scanné le code avant expiration | "Votre code a expiré." + bouton "Recommencer" (→ nouveau `POST start`) |
| `failed` | `device_flow_denied` | L'utilisateur a refusé l'autorisation sur Microsoft | "Connexion refusée." + bouton "Réessayer" |
| `failed` | `device_flow_error` | Erreur Microsoft pendant le flow (réseau, service MS down) | "Erreur de connexion Xbox." + bouton "Réessayer" |
| `failed` | `halo_exchange_failed` | Token Microsoft OK mais échange Halo échoué | "Impossible de contacter le service Halo." + bouton "Réessayer" |
| `failed` | `identity_resolution_failed` | Token Halo OK mais résolution gamertag/xuid échouée | "Impossible de résoudre votre identité Xbox." + bouton "Réessayer" |

#### Contrat de polling frontend

| Règle | Valeur |
|-------|--------|
| Intervalle de polling | `poll_interval_seconds` retourné par `POST start` (typiquement 5s) |
| Arrêt du polling | Dès que `status ∉ {"pending"}` (tout statut terminal arrête le polling) |
| Timeout côté frontend | `expires_in_seconds` retourné par `POST start` — le frontend affiche un compte à rebours visible. À expiration, il arrête le polling et affiche "Code expiré" même si le backend n'a pas encore répondu `expired`. |
| Retry après erreur | Nouveau `POST /auth/device-flow/start` (nouvelle tentative, nouveau code) |
| Refresh navigateur pendant polling | Le frontend refait `GET /bootstrap` → si `setup_state` est toujours `no_halo_link`, il n'y a pas de tentative active retrouvable (process-level). L'utilisateur doit relancer. |

#### Cas particulier : fermeture du navigateur pendant le Device Code Flow

Le thread backend continue jusqu'à résolution. Si l'utilisateur revient :

- **Cas A — l'attempt a réussi** : `session.linked_halo_identity` est renseigné, `setup_state` sera `halo_linked_no_profile` → le frontend passe directement à la confirmation profil
- **Cas B — l'attempt a échoué/expiré** : `setup_state` reste `no_halo_link` → le frontend propose de relancer

### 9.3 Création de profil — erreurs

Erreurs synchrones de `POST /setup/players` :

| Code HTTP | `code` | `retryable` | Cause | CTA frontend |
|:---------:|--------|:-----------:|-------|--------------|
| 403 | `provisioning_disabled` | `false` | `can_self_provision=false` en config | "L'auto-provisioning est désactivé. Contactez l'administrateur." (bouton masqué normalement) |
| 409 | `no_halo_identity` | `false` | Aucune identité Halo liée en session | "Vous devez d'abord vous connecter à Xbox." + retour à l'étape Device Code Flow |
| 409 | `identity_mismatch` | `false` | Le gamertag/xuid envoyé ne correspond pas à `linked_halo_identity` | "L'identité ne correspond pas à votre compte Xbox connecté." (ne devrait pas arriver si le frontend utilise les données de bootstrap) |
| 500 | `internal_error` | `true` | Erreur lors de la création DB ou du profil | "Réessayer" |

### 9.4 Sync initiale — erreurs

#### Erreurs synchrones (`POST /sync/initial`)

| Code HTTP | `code` | `retryable` | Cause | CTA frontend |
|:---------:|--------|:-----------:|-------|--------------|
| 403 | `sync_disabled` | `false` | `can_start_initial_sync=false` (mode démo) | Masquer le bouton |
| 404 | `not_found` | `false` | `player_slug` inconnu | Retour à la création de profil |
| 409 | `sync_already_active` | `false` | Une sync tourne déjà pour ce joueur. Réponse inclut `details.active_job_id`. | Afficher la progression du job actif (basculer sur le polling du `active_job_id` retourné) |
| 500 | `internal_error` | `true` | Erreur inattendue | "Réessayer" |

#### Erreurs asynchrones (job sync via `GET /jobs/{job_id}`)

Le job atteint `status="failed"` avec `error: ApiErrorSchema` renseigné.

| `error.code` | `retryable` | Cause | CTA frontend |
|--------------|:-----------:|-------|--------------|
| `sync_auth_expired` | `true` | Token Halo expiré pendant la sync, refresh silent MSAL échoué | "La connexion Xbox a expiré." + bouton "Relancer la sync" (→ nouveau `POST /sync/initial`) |
| `sync_halo_api_error` | `true` | Erreur Halo API (timeout, 5xx, quota dépassé) | "Le service Halo est temporairement indisponible." + bouton "Relancer" |
| `sync_halo_api_quota` | `true` | Rate limit Halo API atteint (429 répétés) | "Quota d'appels Halo atteint. Réessayez dans quelques minutes." + bouton "Relancer" avec délai suggéré |
| `sync_db_error` | `true` | Erreur d'écriture DuckDB (disque plein, corruption) | "Erreur lors de l'enregistrement des données." + bouton "Relancer" |
| `internal_error` | `true` | Exception inattendue dans le worker | "Erreur inattendue." + bouton "Relancer" |

#### Politique de retry dans le worker sync

Le worker sync ne fait **pas** de retry automatique global. La politique est :

| Niveau | Stratégie |
|--------|-----------|
| **Appel API Halo unitaire** | Retry automatique : 3 tentatives, backoff exponentiel (1s, 2s, 4s), abandon après 3 échecs consécutifs |
| **Phase complète** | Pas de retry automatique. Si une phase échoue après épuisement des retries unitaires, le job passe à `status="failed"` avec le code d'erreur correspondant |
| **Job complet** | Le frontend propose à l'utilisateur de relancer manuellement via un nouveau `POST /sync/initial`. Le nouveau job reprend depuis le début (pas de reprise partielle en V7). |

Justification : une reprise partielle (reprendre à partir du match N) complexifie énormément le code pour un gain marginal sur la première sync (quelques minutes). La reprise se fait depuis le début — les matchs déjà en base sont détectés par dédoublication et ne sont pas ré-écrits.

#### Comportement au restart serveur pendant la sync

1. Le job en cours est marqué `interrupted` (pas `cancelled`) au reload du `JobStore`
2. L'UI récupère le job via `GET /bootstrap` ou `GET /jobs/{job_id}` et voit `status="interrupted"`
3. Le frontend affiche : "La synchronisation a été interrompue (redémarrage du serveur)." + bouton "Relancer la sync"
4. L'utilisateur relance manuellement → nouveau job, reprise depuis le début avec dédoublication

#### Données partielles après échec

Un échec en phase `fetch_matches` ou `enrich` peut laisser des données partielles en base (matchs déjà écrits dans `shared_matches`). Ce n'est **pas un problème** :

- Les matchs écrits sont valides et complets individuellement
- La relance de la sync détecte les matchs existants par `match_id` et ne les ré-écrit pas
- Le marqueur `initial_sync_completed_at` n'est écrit qu'à la fin de `finalize` — tant qu'il est absent, `setup_state` reste `profile_ready_no_sync`
- L'utilisateur ne voit pas de données partielles dans le dashboard (il est bloqué par le guard-barrier tant que `setup_state ≠ "ready"`)

### 9.5 Session web — expiration et recovery

| Scénario | Comportement |
|----------|-------------|
| Cookie absent ou invalide | `get_or_create_session` crée une session vierge. `setup_state` recalculé : si un profil avec marqueur sync existe → `ready` (l'utilisateur entre directement). Sinon → `no_halo_link` (l'onboarding recommence). |
| Cookie expiré (TTL 7j) | Identique à "cookie absent". La session fichier est supprimée. |
| Refresh navigateur mid-onboarding | Session conservée (cookie valide). `setup_state` recalculé depuis la session + la DB. Le frontend reprend à la bonne étape. |
| Refresh navigateur pendant Device Code Flow | Session conservée → si le thread a terminé et renseigné `linked_halo_identity`, le `setup_state` est avancé. Si le thread tourne encore, pas de tentative retrouvable côté frontend (process-level) → l'utilisateur voit `no_halo_link` et peut relancer. |
| Refresh navigateur pendant sync initiale | Session conservée. Le job sync tourne en background (thread daemon). `GET /bootstrap` → `setup_state` reste `profile_ready_no_sync`. Le frontend peut retrouver le job actif si son `job_id` est connu (via un champ `active_sync_job_id` optionnel dans bootstrap ou via un `GET /jobs?type=initial_sync&player_slug=X`). |

> **Point ouvert** : pour que le frontend retrouve le job actif après un refresh, il faut soit (a) stocker `active_sync_job_id` dans la session et l'exposer dans bootstrap, soit (b) ajouter un endpoint de recherche de job par type+joueur. L'option (a) est plus simple et recommandée.

### 9.6 Halo API indisponible pendant l'onboarding

| Moment | Effet | Détection | Recovery |
|--------|-------|-----------|----------|
| Pendant `POST /auth/device-flow/start` | Le Device Code Flow est Microsoft, pas Halo → **aucun impact** | — | — |
| Pendant l'échange token (`_complete_device_flow_bg`) | Attempt passe à `status="failed"`, `error.code="halo_exchange_failed"` | Polling frontend | Bouton "Réessayer" → nouveau Device Code Flow |
| Pendant `fetch_matches` (sync) | Retry automatique unitaire (3×). Si échec persistant → job `failed`, `error.code="sync_halo_api_error"` | Polling job | Bouton "Relancer la sync" |
| Pendant `enrich` (sync) | Idem. Les enrichissements contactant l'API Halo suivent la même politique de retry unitaire | Polling job | Bouton "Relancer la sync" |

### 9.7 Codes d'erreur — registre complet onboarding

| `error.code` | Contexte | HTTP | `retryable` |
|--------------|----------|:----:|:-----------:|
| `demo_mode_unsupported` | Device Flow start en mode démo | 422 | `false` |
| `rate_limit_exceeded` | Trop de requêtes | 429 | `true` |
| `msal_unavailable` | Module MSAL absent/cassé | 503 | `false` |
| `device_flow_denied` | Utilisateur a refusé sur Microsoft | — (async) | `false` |
| `device_flow_error` | Erreur Microsoft pendant le flow | — (async) | `true` |
| `halo_exchange_failed` | Échange token Halo échoué | — (async) | `true` |
| `identity_resolution_failed` | Résolution gamertag/xuid échouée | — (async) | `true` |
| `provisioning_disabled` | Auto-provisioning désactivé | 403 | `false` |
| `no_halo_identity` | Pas d'identité Halo en session | 409 | `false` |
| `identity_mismatch` | Gamertag/xuid ne match pas la session | 409 | `false` |
| `sync_disabled` | Sync désactivée (mode démo) | 403 | `false` |
| `sync_already_active` | Sync déjà en cours pour ce joueur | 409 | `false` |
| `sync_auth_expired` | Token Halo expiré pendant sync | — (job) | `true` |
| `sync_halo_api_error` | Erreur API Halo (5xx, timeout) | — (job) | `true` |
| `sync_halo_api_quota` | Rate limit Halo API (429) | — (job) | `true` |
| `sync_db_error` | Erreur DuckDB (disque, corruption) | — (job) | `true` |
| `internal_error` | Exception inattendue | 500 / job | `true` |

### 9.8 Flux non couverts en V7 — exclusions explicites

Les scénarios suivants sont formellement **hors scope V7**. Ils sont documentés ici pour éviter toute ambiguïté :

| Scénario | Raison d'exclusion | Phase cible |
|----------|--------------------|-------------|
| **Re-onboarding / reset d'un profil** | Single-user MVP, pas de bouton "Reset" | Phase 4+ |
| **Changement de compte Xbox (account switch)** | Nécessite une déconnexion Halo + nouveau Device Code Flow + gestion de l'ancien profil | Phase 4+ |
| **Gamertag rename post-setup** | Le xuid reste stable ; le gamertag est résolu via `xuid_aliases` existant dans shared DB. Si l'alias est stale, il sera mis à jour à la prochaine sync delta. Pas de détection proactive en V7. | Amélioration incrémentale |
| **Multi-joueur / player switching** | SessionStore single-user, pas de CAS | Phase 4 |
| **Reprise partielle de sync** (checkpoint) | Complexité disproportionnée vs bénéfice. Dédoublication des matchs suffit. | Phase 4+ |
| **Accès aux settings/langue pendant l'onboarding** | `locale` est dans la session et le cookie. Le frontend peut techniquement permettre un switch de langue sur l'écran d'onboarding sans endpoint dédié (juste `POST /session/context` avec `locale`). Non spécifié car trivial. | Implicitement supporté |
