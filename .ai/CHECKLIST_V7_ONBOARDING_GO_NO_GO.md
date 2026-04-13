# Checklist V7 — Go / No-Go Onboarding, Auth & Première Sync

> Date : 2026-04-13
> Statut : actif
> Périmètre : revue de préparation, lancement de sprint, revue avant merge

---

## Mode d'emploi

Cette checklist sert à répondre rapidement à trois questions :

1. le contrat est-il assez clair pour démarrer le sprint ?
2. le résultat est-il assez robuste pour ouvrir d'autres chantiers en parallèle ?
3. peut-on merger sans réintroduire une dette de contrat majeure ?

---

## A. Contrats figés

- [ ] `GET /api/v1/bootstrap` est la source de vérité produit de l'onboarding
- [ ] `GET /api/v1/setup/status` est explicitement limité à un rôle legacy / transitoire / dev
- [ ] `linked_halo_identity` a un owner clair, une persistance claire et un schéma clair
- [ ] `POST /setup/players` a une politique d'autorisation backend explicite
- [ ] Le contrat dit clairement si `POST /setup/players` crée réellement la DB joueur ou garantit seulement son initialisation avant la première sync
- [ ] La définition de `profile_ready_no_sync` repose sur un marqueur persistant, pas sur une heuristique durablement ambiguë
- [ ] En mode démo, `setup_state = "ready"` directement — aucun accès au Device Code Flow, provisioning ou sync initiale

---

## B. Device Code Flow

- [ ] Une tentative Device Code est liée à une session backend
- [ ] Une session étrangère ne peut pas lire une tentative qu'elle n'a pas créée
- [ ] L'expiration et la purge des tentatives sont explicites
- [ ] Le retry UX est clair côté frontend
- [ ] Après auth réussie, l'identité Halo liée est récupérable après refresh navigateur

---

## C. Provisioning

- [ ] `can_self_provision` est vérifié côté backend
- [ ] `POST /setup/players` refuse une identité incohérente avec l'identité liée
- [ ] Le provisioning est idempotent pour une même identité Halo
- [ ] Le joueur courant en session est mis à jour après provisioning
- [ ] La continuité auth jusqu'à la première sync est testée
- [ ] Le frontend ne demande plus de ressaisie libre du gamertag après auth réussie

---

## D. Sync initiale

- [ ] Un endpoint dédié `POST /api/v1/sync/initial` existe
- [ ] Une seule sync initiale active par joueur est possible
- [ ] La progression expose phases + compteurs métier + warnings
- [ ] La fin de sync écrit le marqueur persistant attendu
- [ ] Les profils existants ont une stratégie de migration / backfill

---

## E. Jobs & restart

- [ ] Le store de jobs n'est plus purement en mémoire process
- [ ] Un redémarrage de serveur ne laisse pas de job fantôme en `running`
- [ ] La sémantique `interrupted` / relance est explicite dans le contrat API
- [ ] Le frontend affiche un état compréhensible après restart

---

## F. Sécurité mutative

- [ ] Les cookies prod ont été validés derrière reverse proxy
- [ ] Une protection same-origin / CSRF explicite existe sur les routes mutantes portées par cookie
- [ ] Un rate limiting simple protège les routes sensibles
- [ ] Aucun secret n'est loggé en clair

---

## F’. Rollback & nettoyage

- [ ] Chaque sprint peut être reverté indépendamment sans casser l'état existant
- [ ] Les migrations DB sont idempotentes et dry-run-ables
- [ ] L'ancien wizard React (5 étapes via `next_blocking_step`) est supprimé après stabilisation du Sprint 4
- [ ] Les endpoints legacy (`GET /setup/status`, `POST /setup/smoke-test`) sont supprimés ou reclassés en route admin/debug
- [ ] Le fallback `_has_any_synced_matches()` est supprimé après backfill complet des profils existants

---

## G. Tests minimaux attendus

**Stratégie** : chaque tâche IMPL a un fichier de test cible désigné. Les tests sont écrits dans `tests/api/` (tests unitaires/intégration API) sauf le E2E.

- [ ] Test API : bootstrap retourne le bon `setup_state` selon les cas principaux (`tests/api/test_bootstrap_setup_state.py`)
- [ ] Test API : bootstrap renvoie `setup_state="ready"` en mode démo
- [ ] Test API : tentative Device Flow étrangère rejetée (`tests/api/test_device_flow_ownership.py`)
- [ ] Test API : `POST /setup/players` → 403 si `can_self_provision=false` (`tests/api/test_setup_guards.py`)
- [ ] Test API : `POST /setup/players` → 409 si identité incohérente (`tests/api/test_setup_guards.py`)
- [ ] Test API : création profil met à jour le joueur courant en session (`tests/api/test_setup_guards.py`)
- [ ] Test API : sync initiale active unique par joueur (`tests/api/test_sync_initial.py`)
- [ ] Test API : restart job → statut `interrupted`, pas faux `running` (`tests/api/test_job_store.py`)
- [ ] Test E2E : parcours non-technique complet “connecter Xbox → confirmer profil → lancer sync → entrer dans l'app”

---

## Verdict

### Go

- [ ] Tous les points P0 sont verts
- [ ] Aucun point P1 bloquant n'est laissé implicite
- [ ] Le sprint suivant peut démarrer sans dépendre d'un contrat oral ou tacite

### No-Go immédiat si

- [ ] Deux machines d'état produit coexistent encore sans hiérarchie explicite
- [ ] Le backend continue de trust un `gamertag` / `xuid` client pour le provisioning Xbox
- [ ] La première sync n'a pas de marqueur persistant clair
- [ ] Un restart serveur laisse l'UI croire qu'une sync tourne encore alors qu'elle est morte
- [ ] Une route mutante sensible reste protégée uniquement par l'affichage d'un bouton frontend
