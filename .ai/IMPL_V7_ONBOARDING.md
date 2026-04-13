# V7 — Implémentation onboarding — Sprints & état du code

> ⛔ Prérequis : lire d'abord (dans l'ordre) :
> 1. [PLAN_V7_ONBOARDING_MASTER.md](PLAN_V7_ONBOARDING_MASTER.md) — priorités et critères de succès
> 2. [PLAN_V7_AUTH_SECURITY_PRINCIPLES.md](PLAN_V7_AUTH_SECURITY_PRINCIPLES.md) — principes et décisions
> 3. [SPEC_V7_BOOTSTRAP_CONTRACT.md](SPEC_V7_BOOTSTRAP_CONTRACT.md) — contrats API cibles
>
> Date : 2026-04-13
> Rôle : **tâches concrètes** — fichiers à modifier, état fait/à faire, ordre d'exécution

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
| Ownership de tentative Device Code par session | `apps/api/app/services/setup_service.py` + `apps/api/app/routers/setup.py` | ❌ pas implémenté |
| `linked_halo_identity` en session / bootstrap | `apps/api/app/deps/auth.py` + `apps/api/app/schemas/bootstrap.py` | ❌ pas implémenté |
| Création de profil (`POST /setup/players`) | `apps/api/app/routers/setup.py` | ✅ |
| Guard `can_self_provision` sur `POST /setup/players` | `apps/api/app/routers/setup.py` | ✅ implémenté |
| Vérification backend de cohérence entre identité liée et corps de requête | `apps/api/app/routers/setup.py` + `apps/api/app/services/setup_service.py` | ❌ pas implémenté |
| Mise à jour `session.current_player_slug` après provisioning | `apps/api/app/routers/setup.py` | ❌ pas implémenté |
| Persistance / transfert du cache MSAL après création du profil | `apps/api/app/services/setup_service.py` | ❌ pas implémenté |
| `GET /setup/status` pilote encore le wizard React | `apps/web/src/features/setup/` | ⚠️ legacy encore active |
| `POST /setup/smoke-test` encore utilisé comme étape produit | `apps/web/src/features/setup/` | ⚠️ legacy encore active |
| `JobStore` singleton thread-safe + **persistant JSON** (`data/cache/jobs.json`) | `apps/api/app/services/job_store.py` | ✅ persistant + recovery `running→cancelled` au reload |
| `AsyncJobStatus` schéma enrichi (`phase_key`, `matches_done/total`, `warnings`, etc.) | `apps/api/app/schemas/common.py` | ✅ complet |
| Valeur `status="interrupted"` dans le code (distinct de `cancelled`) | `apps/api/app/services/job_store.py` | ❌ à ajouter |
| Marqueur persistant de sync initiale côté player DB | player DB / `sync_meta` | ❌ pas implémenté |
| Backfill des profils existants pour ce marqueur | migration dédiée | ❌ pas implémenté |
| Endpoint `POST /sync/initial` | — | ❌ pas implémenté |
| Frontend onboarding piloté par bootstrap | `apps/web/src/features/setup/` | ❌ pas implémenté |
| Rate limiting routes sensibles | — | ❌ pas implémenté |
| Vérification same-origin / CSRF des routes mutantes | — | ❌ pas implémenté |
| Logs structurés actions sensibles | — | ⚠️ partiel |

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
| **État** | 🔲 À faire |

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

### 2.3 — Assurer la continuité auth après création du profil

| | |
|-|-|
| **Fichiers** | `apps/api/app/routers/setup.py`, `apps/api/app/services/setup_service.py`, éventuels helpers auth |
| **Problème** | Le cache MSAL éphémère de l'attempt n'est pas transféré dans la player DB, alors que le service prétend déjà cette continuité dans sa docstring |
| **Action** | Persister explicitement le cache MSAL utile dans la player DB au moment du provisioning ou documenter un autre mécanisme serveur équivalent |
| **Test** | Profil créé puis première sync immédiate sans redemande d'auth |
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
| **Fichiers** | migration dédiée / script de backfill / lecture bootstrap |
| **Problème** | Sans migration, les profils historiques risquent d'être reclassés comme non synchronisés |
| **Action** | Backfiller le marqueur à partir des données historiques existantes avant rollout complet du nouvel onboarding |
| **Test** | Un profil existant avec historique pertinent arrive directement en `ready` |
| **Fichier de test** | `tests/api/test_bootstrap_setup_state.py` |
| **État** | 🔲 À faire |

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
| **Contrainte** | Single-flight par `player_slug` |
| **Réponse** | 202 + `AsyncJobStatus` |
| **Fichier de test** | `tests/api/test_sync_initial.py` |
| **État** | 🔲 À faire |

### 3.6 — Écran de progression frontend

| | |
|-|-|
| **Fichiers** | nouveau composant dans `apps/web/src/features/setup/` |
| **Action** | Après création profil → lancer sync → afficher progression avec compteurs métier. Réutiliser `useJobStatus` existant (poll 3s). |
| **Contrat UX** | Voir SPEC §8 |
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
1.2  linked_halo_identity en session    ──┼── Sprint 1 (P0)
1.3  ownership des attempts             ──┤
1.4  confirmation UI sans ressaisie     ──┘
           │
2.1  Guard can_self_provision (✅)      ──┐
2.2  Cohérence identité liée            ──┤
2.3  Continuité MSAL post-provisioning  ──┼── Sprint 2 (P1)
2.4  current_player_slug en session     ──┘
           │
3.1  Marqueur sync explicite            ──┐
3.2  Backfill profils existants         ──┤
3.3  Statut interrupted (schéma ✅)     ──┤
3.4  JobStore interrupted (persist. ✅) ──┼── Sprint 3 (P1)
3.5  POST /sync/initial                 ──┤
3.6  Écran progression frontend         ──┘
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
