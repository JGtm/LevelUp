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
