# V7 — Implémentation onboarding — Sprints & état du code

> ⛔ Prérequis : lire d'abord (dans l'ordre) :
> 1. [PLAN_V7_ONBOARDING_MASTER.md](PLAN_V7_ONBOARDING_MASTER.md) — priorités et critères de succès
> 2. [PLAN_V7_AUTH_SECURITY_PRINCIPLES.md](PLAN_V7_AUTH_SECURITY_PRINCIPLES.md) — principes et décisions
> 3. [SPEC_V7_BOOTSTRAP_CONTRACT.md](SPEC_V7_BOOTSTRAP_CONTRACT.md) — contrats API cibles
>
> Date : 2026-04-13
> Rôle : **tâches concrètes** — fichiers à modifier, état fait/à faire, ordre d'exécution

> ⚠️ **Révision majeure le 2026-04-13** : ajout des §9 erreurs/recovery dans SPEC, nouvelles tâches 1.5, 3.6, 3.7, 3.8 dans IMPL, précisions sur SessionData (§1.2), transfert MSAL (§2.3), backfill concret (§3.2), `active_sync_job_id` dans bootstrap. **Tout agent IA reprenant ce chantier doit relire SPEC §9 + IMPL complet avant de coder.**

---

## État des briques existantes

Avant de coder, vérifier ce qui est **déjà implémenté** :

| Brique | Fichier | État |
|--------|---------|:----:|
| Session web httpOnly + cookie sécurisé | `apps/api/app/deps/auth.py` | ✅ |
| `SessionData` (session_id, auth_ready, locale, player_slug) | `apps/api/app/deps/auth.py` | ✅ partiel |
| `BootstrapResponse` avec `setup_state` | `apps/api/app/schemas/bootstrap.py` | ✅ partiel |
| `_compute_setup_state()` (3 états sur 4) | `apps/api/app/services/bootstrap_service.py` | ⚠️ manque `profile_ready_no_sync` |
| `CapabilityMap` avec `can_self_provision`, `can_start_initial_sync`, `can_manage_instance` | `apps/api/app/schemas/common.py` | ✅ schéma |
| `_build_capabilities()` branchée sur app_settings | `apps/api/app/services/bootstrap_service.py` | ✅ partiel |
| Device Code Flow complet (start + poll + résolution gamertag) | `apps/api/app/services/setup_service.py` | ✅ |
| `DeviceFlowStatusResponse` avec `gamertag` / `xuid` | `apps/api/app/schemas/setup.py` | ✅ |
| `session.auth_ready = True` au polling `GET /auth/device-flow/{attempt_id}` | `apps/api/app/routers/setup.py` | ✅ |
| Ownership de tentative Device Code par session | `apps/api/app/services/setup_service.py` + `apps/api/app/routers/setup.py` | ✅ implémenté |
| `linked_halo_identity` en session / bootstrap | `apps/api/app/deps/auth.py` + `apps/api/app/schemas/bootstrap.py` | ✅ implémenté |
| Création de profil (`POST /setup/players`) | `apps/api/app/routers/setup.py` | ✅ |
| Guard `can_self_provision` sur `POST /setup/players` | `apps/api/app/routers/setup.py` | ✅ implémenté |
| Vérification backend de cohérence entre identité liée et corps de requête | `apps/api/app/routers/setup.py` + `apps/api/app/services/setup_service.py` | ✅ implémenté |
| Mise à jour `session.current_player_slug` après provisioning | `apps/api/app/routers/setup.py` | ✅ implémenté |
| Persistance / transfert du cache MSAL après création du profil | `apps/api/app/services/setup_service.py` | ✅ implémenté |
| `GET /setup/status` pilote encore le wizard React | `apps/web/src/features/setup/` | ✅ endpoint supprimé (Sprint 5.2) |
| `POST /setup/smoke-test` encore utilisé comme étape produit | `apps/web/src/features/setup/` | ✅ endpoint supprimé (Sprint 5.2) |
| `JobStore` singleton thread-safe + **persistant JSON** (`data/cache/jobs.json`) | `apps/api/app/services/job_store.py` | ✅ persistant + recovery `running→interrupted` au reload |
| `AsyncJobStatus` schéma enrichi (`phase_key`, `matches_done/total`, `warnings`, etc.) | `apps/api/app/schemas/common.py` | ✅ complet |
| Valeur `status="interrupted"` dans le code (distinct de `cancelled`) | `apps/api/app/services/job_store.py` | ✅ implémenté |
| Marqueur persistant de sync initiale côté player DB | player DB / `sync_meta` (`initial_sync_completed_at`) | ✅ implémenté |
| Backfill des profils existants pour ce marqueur | migration `add_initial_sync_completed_at` | ✅ migration créée |
| Endpoint `POST /sync/initial` | `apps/api/app/routers/sync.py` | ✅ implémenté |
| Frontend onboarding piloté par bootstrap | `apps/web/src/features/setup/SetupPage.tsx` | ✅ implémenté — wizard legacy supprimé |
| Rate limiting routes sensibles | `apps/api/app/deps/rate_limit.py` | ✅ implémenté |
| Vérification same-origin / CSRF des routes mutantes | `apps/api/app/deps/csrf.py` | ✅ implémenté |
| Logs structurés actions sensibles | `sync_service.py`, `setup_service.py` | ✅ `initial_sync_started`, `initial_sync_succeeded`, `device_flow_*` |

---

## Sprint 1 — Contrat unique & Device Code Flow (P0)

> **Objectif** : unifier la machine d'état produit, fiabiliser l'identité Halo liée, supprimer la dépendance produit à `setup/status`.
> **Contexte requis** : SPEC §0, §1, §3

### 1.1 — Faire de bootstrap la source de vérité produit

| | |
|-|-|
| **Fichiers** | `apps/api/app/schemas/bootstrap.py`, `apps/api/app/services/bootstrap_service.py`, `apps/web/src/features/setup/SetupPage.tsx`, `apps/web/src/features/setup/queries.ts` |
| **Problème** | Le wizard React route encore sur `GET /setup/status` / `next_blocking_step`, alors que le contrat cible impose `GET /bootstrap` / `setup_state` |
| **Action** | Brancher l'onboarding React final sur bootstrap. Reclasser `GET /setup/status` comme surface legacy / transitoire. |
| **Test** | L'écran d'onboarding route correctement à partir d'un seul `GET /bootstrap` |
| **Fichier de test** | `tests/api/test_bootstrap_setup_state.py` |
| **État** | � En cours |

**Stratégie de transition frontend** : le wizard React legacy (5 étapes via `next_blocking_step`) est remplacé par le nouveau parcours piloté par `setup_state`. L'ancien wizard est **supprimé à la fin du plan** (Sprint 4 ou nettoyage post-Sprint 4) — pas de coexistence durable. Pendant la transition, les tests legacy restent opérationnels mais ne pilotent plus l'UX de production.

**Mode démo** : en mode démo, le parcours auth est entièrement court-circuité. `setup_state` renverra directement `"ready"` (avec les données démo préchargées). Le mode démo existe pour explorer les capacités de l'app et les stats, pas pour tester l'onboarding. Le Device Code Flow, le provisioning et la sync initiale ne sont **pas accessibles** en mode démo.

### 1.2 — Stocker `linked_halo_identity` côté session et l'exposer via bootstrap

| | |
|-|-|
| **Fichiers** | `apps/api/app/deps/auth.py`, `apps/api/app/schemas/bootstrap.py`, `apps/api/app/services/bootstrap_service.py`, `apps/api/app/routers/setup.py` |
| **Problème** | Le backend sait résoudre `gamertag` / `xuid`, mais la session ne stocke que `auth_ready` ; bootstrap ne peut donc pas piloter une confirmation de profil 100 % serveur |
| **Action** | Ajouter un état session explicite pour l'identité Halo liée, le persister côté serveur, puis le renvoyer dans `BootstrapResponse` |
| **Test** | Après succès du Device Code Flow puis refresh navigateur, le bootstrap renvoie toujours l'identité Halo liée |
| **Fichier de test** | `tests/api/test_bootstrap_setup_state.py` |
| **État** | 🔲 À faire |

**État actuel de `SessionData`** (dans `apps/api/app/deps/auth.py`) :

```python
@dataclass
class SessionData:
    session_id: str                           # UUID auto-généré
    created_at: float                         # time.time()
    last_seen_at: float                       # mis à jour par .touch()
    current_player_slug: str | None = None    # joueur actif
    locale: str = "fr"
    hints_visible: bool = True
    auth_ready: bool = False                  # ← True après Device Code Flow réussi
    linked_halo_identity: dict[str, str] | None = None  # ← EXISTE DÉJÀ mais jamais écrit
    _extra: dict[str, Any]                    # champs inconnus préservés au désérialisé
```

Le champ `linked_halo_identity` **existe déjà** dans le dataclass. Le problème n'est pas le schéma — c'est que personne ne l'écrit. La tâche concrète est :

1. Dans `setup_service.py` → `_complete_device_flow_bg()` : après résolution `gamertag` + `xuid`, **écrire dans la session** `linked_halo_identity = {"gamertag": gamertag, "xuid": xuid}` et sauvegarder la session via `SessionStore.save()`
2. Dans `bootstrap_service.py` → `_build_bootstrap()` : lire `session.linked_halo_identity` et le mapper vers `HaloIdentitySummary` dans la réponse

**Persistance** : `SessionStore` stocke chaque session dans un fichier JSON dans `data/sessions/{session_id}.json`, avec TTL configurable. Le champ `linked_halo_identity` sera automatiquement sérialisé/désérialisé via `to_dict()` / `from_dict()` — aucune modification du `SessionStore` n'est nécessaire.

### 1.3 — Lier chaque tentative Device Code à la session qui l'a créée

| | |
|-|-|
| **Fichiers** | `apps/api/app/services/setup_service.py`, `apps/api/app/routers/setup.py` |
| **Problème** | Les tentatives sont seulement indexées par `attempt_id` en mémoire process ; aucune ownership par session n'est imposée |
| **Action** | Stocker `session_id`, gérer single-flight par session, expiration et purge, retourner 404 pour une tentative étrangère |
| **Test** | Deux sessions différentes : la seconde ne peut pas lire le statut de la première |
| **Fichier de test** | `tests/api/test_device_flow_ownership.py` |
| **État** | 🔲 À faire |

### 1.4 — Supprimer la ressaisie du gamertag

| | |
|-|-|
| **Fichiers** | `apps/web/src/features/setup/SetupPage.tsx`, nouveau composant onboarding bootstrap si nécessaire |
| **Problème** | Après `status=provisioned`, le backend connaît déjà `gamertag` / `xuid`, mais l'UI actuelle garde une saisie quasi libre |
| **Action** | Afficher une carte de confirmation à partir de `linked_halo_identity`, pas un champ vide |
| **Test** | Après auth réussie, l'utilisateur ne ressaisit pas son gamertag |
| **Fichier de test** | Test E2E (Playwright ou manuel) |
| **État** | 🔲 À faire |

### 1.5 — Contrat d'erreur et UX de timeout du Device Code Flow

| | |
|-|-|
| **Fichiers** | `apps/web/src/features/setup/` (composant Device Code Flow) |
| **Problème** | Le backend gère les erreurs async (attempt → `failed`/`expired`), mais le frontend n'a pas de contrat pour : afficher le compte à rebours, arrêter le polling, permettre le retry |
| **Action** | Implémenter dans le frontend : (1) compte à rebours visible basé sur `expires_in_seconds`, (2) arrêt du polling quand `status ∉ {"pending"}`, (3) message d'erreur contextuel selon `error.code` (voir SPEC §9.2), (4) bouton "Recommencer" qui relance `POST /auth/device-flow/start` |
| **Contrat** | Voir SPEC §9.2 — table des `error.code` et CTA frontend attendus |
| **Test** | (1) Le compte à rebours est visible et décrémente. (2) À expiration, le message "Code expiré" s'affiche sans attendre le backend. (3) Après `status=failed`, le bon message d'erreur s'affiche. (4) "Recommencer" lance un nouveau flow. |
| **Fichier de test** | Test E2E / Playwright |
| **État** | 🔲 À faire |

---

## Sprint 2 — Provisioning backend-authoritative (P1)

> **Objectif** : le provisioning repose sur les guards backend et l'identité liée, pas sur la confiance au client.
> **Contexte requis** : SPEC §2, §4 ; PRINCIPES §D3, §D6

### 2.1 — Guard backend `can_self_provision`

| | |
|-|-|
| **Fichier** | `apps/api/app/routers/setup.py` |
| **Action** | Avant `create_player_profile()`, vérifier `capabilities.can_self_provision`. Si `false` → 403. |
| **Règle** | Le frontend masque éventuellement le bouton, mais le backend refuse **indépendamment**. |
| **Test** | `POST /setup/players` avec `can_self_provision=false` dans app_settings → 403 |
| **Fichier de test** | `tests/api/test_setup_guards.py` |
| **État** | ✅ Déjà implémenté — vérifier que le guard renvoie bien 403 et ajouter le test |

### 2.2 — Faire respecter l'identité Halo liée au moment du provisioning

| | |
|-|-|
| **Fichiers** | `apps/api/app/routers/setup.py`, `apps/api/app/services/setup_service.py` |
| **Problème** | Le service accepte aujourd'hui `gamertag` / `xuid` depuis le client après simple validation de format |
| **Action** | En mode `profile_mode="xbox"`, comparer le corps de requête avec `session.linked_halo_identity` et refuser toute incohérence |
| **Test** | `POST /setup/players` avec un `gamertag` différent de l'identité liée → 409 |
| **Fichier de test** | `tests/api/test_setup_guards.py` |
| **État** | 🔲 À faire |

### 2.3 — Transférer le cache MSAL dans la player DB au provisioning

| | |
|-|-|
| **Fichiers** | `apps/api/app/services/setup_service.py`, `src/auth/_msal.py` |
| **Problème** | Le Device Code Flow dans `setup_service.py` crée un `SerializableTokenCache` éphémère en mémoire (`_msal_module.SerializableTokenCache()`). Ce cache contient le refresh token Microsoft obtenu. Mais il est stocké uniquement dans l'objet `_DeviceFlowAttempt` en mémoire process — il n'est **jamais persisté**. Si le serveur redémarre ou si la sync arrive plus tard, le refresh token est perdu et l'utilisateur doit refaire le Device Code Flow. |
| **Patron existant** | Le projet dispose déjà des fonctions `load_msal_cache(db_path)` et `save_msal_cache_if_changed(db_path, cache)` dans `src/auth/_msal.py`. Elles sérialisent le cache dans la table `sync_meta` de la player DB (clé `msal_token_cache`). C'est exactement le pattern utilisé par le moteur de sync Streamlit. |
| **Action** | Au moment du provisioning (`POST /setup/players`), après création/récupération de la player DB : (1) récupérer le `SerializableTokenCache` depuis l'attempt Device Code Flow actif de la session, (2) appeler `save_msal_cache_if_changed(player_db_path, cache)` pour persister le refresh token dans `sync_meta`. Le cache est alors disponible pour la sync initiale (Sprint 3.5) via `load_msal_cache(player_db_path)`. |
| **Implémentation concrète** | Dans `_DeviceFlowAttempt`, le cache est déjà stocké dans `attempt._cache`. Au provisioning : `from src.auth._msal import save_msal_cache_if_changed` → `save_msal_cache_if_changed(player_db_path, attempt._cache)`. |
| **Test** | Profil créé puis première sync immédiate sans redemande d'auth. Vérifier que `sync_meta` contient la clé `msal_token_cache` après provisioning. |
| **Fichier de test** | `tests/api/test_provisioning_continuity.py` |
| **État** | 🔲 À faire |

### 2.4 — Mettre à jour le joueur courant en session

| | |
|-|-|
| **Fichiers** | `apps/api/app/routers/setup.py`, `apps/api/app/deps/auth.py` |
| **Problème** | Le provisioning ne fixe pas `session.current_player_slug`, ce qui laisse le joueur courant implicite |
| **Action** | Après création ou récupération idempotente du profil, écrire `session.current_player_slug` |
| **Test** | Le bootstrap suivant renvoie le profil créé comme `current_player` |
| **Fichier de test** | `tests/api/test_setup_guards.py` |
| **État** | 🔲 À faire |

> **Note** : la tâche 2.5 (`auto_provision_from_halo_identity`) a été déplacée en Phase 4. Pour un MVP single-user, l'auto-provision via l'identité Halo liée est le seul mode utile. Le flag n'a de valeur que dans un scénario multi-user avec admin (Phase 4).

---

## Sprint 3 — Sync initiale & persistance des jobs (P1)

> **Objectif** : la première sync est un vrai job produit, avec source de vérité persistante et comportement clair au restart.
> **Contexte requis** : SPEC §5, §6 ; PRINCIPES §D5

### 3.1 — Définir et écrire le marqueur persistant de sync initiale

| | |
|-|-|
| **Fichiers** | couche player DB / `sync_meta`, futur `sync_service.py`, `bootstrap_service.py` |
| **Problème** | `profile_ready_no_sync` n'a pas encore de source de vérité fiable |
| **Action** | Choisir un marqueur persistant (`initial_sync_completed_at` ou `last_successful_sync_at`), l'écrire à la fin d'une première sync réussie et le lire dans bootstrap |
| **Test** | Un joueur fraîchement provisionné passe à `ready` seulement après écriture du marqueur |
| **Fichier de test** | `tests/api/test_bootstrap_setup_state.py` |
| **État** | 🔲 À faire |

**Logique de dérivation pendant la période de migration** :

```
if marqueur_sync_present:
    return "ready"
elif has_any_synced_matches():  # fallback pour profils historiques non-encore migrés
    return "ready"
else:
    return "profile_ready_no_sync"
```

Le fallback `_has_any_synced_matches()` est conservé comme filet de sécurité pendant la migration (Sprint 3.2). Une fois le backfill terminé pour tous les profils existants, ce fallback sera supprimé et seul le marqueur persistant fera foi.

### 3.2 — Backfiller les profils existants pour ce marqueur

| | |
|-|-|
| **Fichiers** | script de migration + `src/data/sync/migrations.py` |
| **Problème** | Sans migration, les profils historiques qui ont déjà des matchs seraient reclassés à tort comme `profile_ready_no_sync` au lieu de `ready` |
| **Action** | Créer une migration idempotente qui initialise `initial_sync_completed_at` pour tous les profils existants ayant un historique de sync |
| **Test** | Un profil existant avec historique pertinent arrive directement en `ready` |
| **Fichier de test** | `tests/api/test_bootstrap_setup_state.py` |
| **État** | 🔲 À faire |

**Schéma cible** : le marqueur est une clé dans la table `sync_meta` (key-value VARCHAR) de chaque player DB :

| Clé | Valeur | Signification |
|-----|--------|--------------|
| `initial_sync_completed_at` | ISO8601 timestamp | Première sync terminée avec succès |

**Script de backfill** (pseudo-SQL exécuté sur chaque `data/players/{gamertag}/stats.duckdb`) :

```sql
-- Condition : la clé n'existe pas encore ET le joueur a déjà une sync réussie
INSERT INTO sync_meta (key, value, updated_at)
SELECT
    'initial_sync_completed_at',
    sm.value,                    -- copier la valeur de last_sync_at
    CURRENT_TIMESTAMP
FROM sync_meta sm
WHERE sm.key = 'last_sync_at'
  AND sm.value IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM sync_meta WHERE key = 'initial_sync_completed_at'
  );
```

**Logique Python** (dans le système de migration existant `src/data/migration/`) :

```python
def apply_schema(conn: duckdb.DuckDBPyConnection) -> None:
    """Migration : backfill initial_sync_completed_at pour profils existants."""
    conn.execute("""
        INSERT INTO sync_meta (key, value, updated_at)
        SELECT 'initial_sync_completed_at', value, CURRENT_TIMESTAMP
        FROM sync_meta
        WHERE key = 'last_sync_at' AND value IS NOT NULL
          AND NOT EXISTS (
            SELECT 1 FROM sync_meta WHERE key = 'initial_sync_completed_at'
          )
    """)
```

**Critères de validation après backfill** :

1. Pour chaque player DB dans `data/players/*/stats.duckdb` :
   - Si `sync_meta` contient `last_sync_at` non null → `initial_sync_completed_at` doit aussi exister
   - Si `sync_meta` ne contient pas `last_sync_at` → profil neuf, pas de marqueur attendu
2. Script de vérification : `SELECT key, value FROM sync_meta WHERE key IN ('last_sync_at', 'initial_sync_completed_at')` sur chaque player DB
3. **Critère Go/No-Go** : 100% des player DBs ayant `last_sync_at` ont aussi `initial_sync_completed_at` après exécution de la migration

**Déploiement** : cette migration est exécutée **avant** que le nouveau code de bootstrap ne soit activé, via le système de migration automatique existant (`launcher.py → _run_migrations()`). Elle est idempotente — exécuter 2 fois ne produit aucun effet (clause `NOT EXISTS`).

### 3.3 — Ajouter le statut `interrupted` dans le code

| | |
|-|-|
| **Fichier** | `apps/api/app/services/job_store.py` |
| **Problème** | Le schéma `AsyncJobStatus` est déjà complet (tous les champs enrichis sont présents : `phase_key`, `phase_label`, `matches_done/total`, `subtasks_done/total`, `eta_seconds`, `warnings`). En revanche, la valeur `"interrupted"` n'est pas encore distincte de `"cancelled"` dans la logique du `JobStore`. |
| **Action** | Ajouter la valeur `"interrupted"` comme statut distinct, utilisé lors du rechargement de jobs incomplets au redémarrage (différent de `"cancelled"` qui est une annulation explicite) |
| **Test** | Un job `running` au moment d'un restart est rechargé comme `interrupted`, pas `cancelled` |
| **Fichier de test** | `tests/api/test_job_store.py` |
| **État** | 🔲 À faire (schéma ✅, logique ❌) |

### 3.4 — Ajouter la sémantique `interrupted` au JobStore

| | |
|-|-|
| **Fichier** | `apps/api/app/services/job_store.py` |
| **Problème** | Le `JobStore` est déjà **persistant** (`data/cache/jobs.json`, thread-safe, avec recovery `running → cancelled` au reload et purge lazy). Cependant, les jobs interrompus par un restart sont marqués `cancelled` au lieu de `interrupted`, ce qui ne reflète pas la sémantique réelle. |
| **Action** | Remplacer la recovery `running → cancelled` par `running → interrupted` avec un warning explicite. L'UI peut alors proposer "Relancer la sync" plutôt qu'afficher un état terminal ambigu. |
| **Test** | Redémarrage du serveur pendant une sync : le job rechargé apparaît avec `status=interrupted` et un warning |
| **Fichier de test** | `tests/api/test_job_store.py` |
| **État** | 🔲 À faire (persistance ✅, sémantique interrupted ❌) |

### 3.5 — Créer l'endpoint `POST /api/v1/sync/initial`

| | |
|-|-|
| **Fichiers** | nouveau router `apps/api/app/routers/sync.py` + service `apps/api/app/services/sync_service.py` |
| **Action** | Créer un job `initial_sync`, lancer l'orchestrateur sync, mettre à jour le job avec les compteurs métier, écrire le marqueur de fin |
| **Guard** | Vérifier `can_start_initial_sync` → 403 si `false` |
| **Contrainte** | Single-flight par `player_slug`. Si un job `initial_sync` actif existe pour ce joueur → retourner `409` avec `details.active_job_id` |
| **Réponse** | 202 + `AsyncJobStatus` |
| **Fichier de test** | `tests/api/test_sync_initial.py` |
| **État** | 🔲 À faire |

### 3.6 — Gestion des erreurs dans le worker sync

| | |
|-|-|
| **Fichiers** | `apps/api/app/services/sync_service.py` (worker background) |
| **Problème** | Le worker sync doit capturer les erreurs, les classer par type, et les persister dans le job avec le bon `error.code` |
| **Action** | Implémenter dans le worker : (1) try/except par phase, (2) retry automatique unitaire (3×, backoff exponentiel) pour les appels API Halo, (3) classification de l'erreur (`sync_auth_expired`, `sync_halo_api_error`, `sync_halo_api_quota`, `sync_db_error`, `internal_error`) selon le type d'exception, (4) écriture dans le `JobStore` via `store.update(job_id, status="failed", error=ApiErrorSchema(code=..., retryable=True))` |
| **Contrat** | Voir SPEC §9.4 — table des `error.code` et politique de retry |
| **Test** | (1) Erreur API Halo mockée → job `failed` avec `error.code="sync_halo_api_error"`. (2) Erreur auth → job `failed` avec `error.code="sync_auth_expired"`. (3) Exception inattendue → job `failed` avec `error.code="internal_error"`. |
| **Fichier de test** | `tests/api/test_sync_initial.py` |
| **État** | 🔲 À faire |

### 3.7 — Retrouver le job sync actif après refresh navigateur

| | |
|-|-|
| **Fichiers** | `apps/api/app/deps/auth.py`, `apps/api/app/services/bootstrap_service.py`, `apps/api/app/schemas/bootstrap.py` |
| **Problème** | Si l'utilisateur rafraîchit le navigateur pendant la sync, le frontend perd le `job_id` et ne peut plus afficher la progression |
| **Action** | Stocker `active_sync_job_id` dans la session quand un job sync est lancé. L'exposer dans `BootstrapResponse` (champ optionnel). Le frontend le récupère au rechargement et reprend le polling. Le champ est remis à `null` quand le job atteint un statut terminal. |
| **Test** | Lancer sync → refresh navigateur → `GET /bootstrap` renvoie `active_sync_job_id` → le frontend reprend le polling |
| **Fichier de test** | `tests/api/test_sync_initial.py` |
| **État** | 🔲 À faire |

### 3.8 — Écran de progression frontend avec gestion d'erreur

| | |
|-|-|
| **Fichiers** | nouveau composant dans `apps/web/src/features/setup/` |
| **Action** | Après création profil → lancer sync → afficher progression avec compteurs métier. Réutiliser `useJobStatus` existant (poll 3s). **Ajouter** : (1) affichage du message d'erreur basé sur `error.code` si `status="failed"`, (2) bouton "Relancer la sync" si `error.retryable=true`, (3) message spécifique si `status="interrupted"` ("Synchronisation interrompue — redémarrage serveur"), (4) récupération du job actif via `active_sync_job_id` de bootstrap au rechargement de page |
| **Contrat UX** | Voir SPEC §8 + §9.4 |
| **État** | 🔲 À faire |

---

## Sprint 4 — Durcissement sécurité (P2)

> **Objectif** : durcir les routes mutantes portées par cookie et valider le comportement prod derrière proxy.
> **Contexte requis** : PRINCIPES §Exigences de sécurité

### 4.1 — Vérifier les cookies en production

| | |
|-|-|
| **Fichier** | `apps/api/app/deps/auth.py` |
| **Checklist** | httpOnly ✅, Secure si HTTPS ✅, SameSite=Lax ✅, TTL 7j ✅ |
| **Action** | Valider la détection HTTPS en prod derrière reverse proxy |
| **État** | ⚠️ À vérifier en conditions réelles |

### 4.2 — Ajouter une protection same-origin / CSRF explicite

| | |
|-|-|
| **Fichiers** | middleware ou dépendance FastAPI dédiée |
| **Routes** | `POST /auth/device-flow/start`, `POST /setup/players`, `POST /sync/initial`, `PATCH /settings` |
| **Action** | Valider `Origin` / `Referer` pour les routes mutantes basées sur cookie ; documenter le comportement et les exceptions |
| **État** | 🔲 À faire |

### 4.3 — Rate limiting routes sensibles

| | |
|-|-|
| **Fichiers** | middleware ou dépendance FastAPI |
| **Routes** | `POST /auth/device-flow/start`, `POST /setup/players`, `POST /sync/initial` |
| **Seuil** | 5 req/min par IP ou session |
| **État** | 🔲 À faire |

### 4.4 — Logging structuré des actions sensibles

| | |
|-|-|
| **Fichiers** | `apps/api/app/services/setup_service.py`, futur `sync_service.py`, routeurs concernés |
| **Actions à tracer** | Démarrage/succès/échec Device Code Flow, création profil, démarrage/fin sync, jobs interrompus au restart |
| **Règle** | Aucun secret dans les logs (masquer tokens / cache brut) |
| **État** | ⚠️ Partiellement fait |

---

## Ordre d'exécution

```
1.1  Bootstrap = source produit         ──┐
1.2  linked_halo_identity en session    ──┤
1.3  ownership des attempts             ──┼── Sprint 1 (P0)
1.4  confirmation UI sans ressaisie     ──┤
1.5  UX erreur/timeout Device Flow      ──┘
           │
2.1  Guard can_self_provision (✅)      ──┐
2.2  Cohérence identité liée            ──┤
2.3  Continuité MSAL post-provisioning  ──┼── Sprint 2 (P1)
2.4  current_player_slug en session     ──┘
           │
3.1  Marqueur sync explicite            ──┐
3.2  Backfill profils existants         ──┤
3.3  Statut interrupted (schéma ✅)     ──┤
3.4  JobStore interrupted (persist. ✅) ──┤
3.5  POST /sync/initial + single-flight ──┼── Sprint 3 (P1)
3.6  Erreurs worker sync (retry+codes)  ──┤
3.7  active_sync_job_id en bootstrap    ──┤
3.8  Écran progression + erreur         ──┘
           │
4.1  Cookies prod derrière proxy        ──┐
4.2  Same-origin / CSRF                 ──┤
4.3  Rate limiting                      ──┼── Sprint 4 (P2)
4.4  Logging structuré                  ──┘
           │
5.1  Supprimer wizard legacy            ──┐
5.2  Supprimer endpoints legacy         ──┼── Nettoyage post-Sprint 4
5.3  Supprimer fallback has_matches()   ──┘
```

Chaque sprint suppose le précédent suffisamment stabilisé pour éviter d'ouvrir plusieurs couches de dette de contrat en parallèle.

---

## Nettoyage post-Sprint 4

> **Objectif** : supprimer le code legacy de l'onboarding une fois le nouveau parcours stable.

### 5.1 — Supprimer le wizard React legacy

| | |
|-|-|
| **Fichiers** | `apps/web/src/features/setup/SetupPage.tsx` (composants `StepChooseMode`, `StepSmokeTest`, orchestrateur via `next_blocking_step`), `apps/web/src/features/setup/queries.ts` (hooks liés à `setup/status`) |
| **Action** | Supprimer les composants du wizard legacy (5 étapes) et le routing via `next_blocking_step`. Ne conserver que le parcours piloté par `setup_state`. |
| **Condition** | Sprint 4 terminé et nouveau parcours stable |

### 5.2 — Reclasser ou supprimer les endpoints legacy

| | |
|-|-|
| **Fichiers** | `apps/api/app/routers/setup.py` |
| **Action** | `GET /setup/status` → supprimer ou déplacer en route admin/debug. `POST /setup/smoke-test` → supprimer ou déplacer en route dev. |

### 5.3 — Supprimer le fallback `_has_any_synced_matches()`

| | |
|-|-|
| **Fichier** | `apps/api/app/services/bootstrap_service.py` |
| **Condition** | Sprint 3.2 (backfill) terminé et vérifié — tous les profils existants ont le marqueur persistant |
| **Action** | Retirer le fallback et ne garder que la lecture du marqueur persistant |
