# V7 — Matrice des contrats à figer avant implémentation

> Date : 2026-04-13
> Statut : actif
> Rôle : **vue architecte** — lister les contrats à figer, leur source de vérité, leur owner, leur test minimal et leur critère de sortie

---

## Mode d'emploi

Cette matrice sert à éviter le démarrage de plusieurs chantiers en parallèle sur un contrat encore flou.

Règles d'usage :

- une ligne marquée **P0** ou **P1** doit être figée avant d'ouvrir plusieurs implémentations dépendantes
- si la source de vérité n'est pas claire, la ligne est bloquante
- si le test minimal n'est pas exprimable simplement, le contrat n'est pas encore assez net

---

## Matrice

| Sujet | Source de vérité | Owner principal | Test minimal | Critère de sortie | Priorité |
|-------|------------------|-----------------|--------------|-------------------|:--------:|
| Machine d'état produit onboarding | `GET /api/v1/bootstrap` + `setup_state` | API + Web | L'onboarding route correctement après un seul appel bootstrap | Le shell React final ne dépend plus de `GET /setup/status` pour le produit | P0 |
| Statut legacy `setup/status` / `smoke-test` | Contrat de transition documenté | API + Web | Les tests legacy passent encore, sans piloter le shell produit | Les endpoints legacy sont explicitement reclassés ou débranchés de l'UX finale | P1 |
| Identité Halo liée en session | `SessionData` côté backend | API | Après auth réussie puis refresh, bootstrap renvoie encore `linked_halo_identity` | `linked_halo_identity` est persistée côté session et consommable par le frontend | P0 |
| Ownership d'une tentative Device Code | Session backend + `attempt_id` | API | Une seconde session ne peut pas lire le statut d'une tentative étrangère | Toute tentative est liée à une session, expirée et purgée proprement | P0 |
| Politique de provisioning | `CapabilityMap.can_self_provision` + guard router | API | `POST /setup/players` renvoie 403 quand `can_self_provision=false` | Le backend refuse indépendamment de l'UI | P0 |
| Autorité de l'identité au provisioning | `session.linked_halo_identity` | API | Corps de requête incohérent avec l'identité liée → 409 | `POST /setup/players` ne truste plus `gamertag` / `xuid` client en mode Xbox | P0 |
| Continuité auth après provisioning | Cache MSAL / état auth côté player DB ou store serveur explicite | API | Auth réussie → profil créé → première sync sans redemande d'auth | La continuité auth est documentée, testée et stable | P1 |
| Joueur courant après provisioning | `session.current_player_slug` | API | Après création profil, bootstrap renvoie ce profil comme `current_player` | Le joueur courant n'est plus implicite | P1 |
| Marqueur de sync initiale réussie | `sync_meta` dans la player DB | API | Profil neuf sans marqueur → `profile_ready_no_sync` ; profil avec marqueur → `ready` | La transition `profile_ready_no_sync` → `ready` ne repose plus sur une heuristique faible | P1 |
| Migration des profils existants | Backfill / migration dédiée | API + Ops | Un profil historique déjà peuplé n'est pas renvoyé vers l'onboarding de sync | Le rollout ne casse pas les utilisateurs existants | P1 |
| Single-flight de sync initiale | Store de jobs + verrou par `player_slug` | API | Deux appels concurrents pour le même joueur ne créent pas deux jobs | Une seule sync initiale active par joueur | P1 |
| Sémantique au restart des jobs | Store de jobs persistant | API | Redémarrage pendant une sync → job rechargé avec statut explicite | `running` fantôme impossible après restart | P1 |
| Contrat de reprise ou relance | `AsyncJobStatus.status` + warnings | API + Web | L'UI affiche “interrompu / relancer” au lieu d'un faux progrès | La reprise ou la relance est compréhensible pour l'utilisateur | P1 |
| Protection mutative same-origin / CSRF | Middleware / dépendance FastAPI | API | Requête cross-site mutante rejetée | La protection ne reste pas formulée en “si nécessaire” | P2 |
| Rate limiting des routes sensibles | Middleware / dépendance FastAPI | API + Ops | Dépassement 5 req/min → refus propre | Device Flow, provisioning et sync initiale sont protégés contre l'abus trivial | P2 |
| Logging structuré des actions sensibles | Couche service / routeurs | API + Ops | Logs présents sans fuite de secrets | Les actions critiques sont traçables et actionnables | P2 |
| Mode démo court-circuité | `bootstrap_service.py` | API | En mode démo, `setup_state="ready"` sans auth/provisioning/sync | Le mode démo ne déclenche jamais l'onboarding | P0 |
| Suppression du wizard legacy | `apps/web/src/features/setup/` | Web | L'ancien wizard (5 étapes via `next_blocking_step`) est supprimé | Aucun code frontend ne dépend encore de `setup/status` pour le produit | P2 |
| Rollback par sprint | Documentation + feature flags | Ops | Chaque sprint peut être reverté sans casser l'état existant | Les migrations sont idempotentes et dry-run-ables | P1 |

---

## Ordre recommandé de verrouillage

1. Machine d'état produit
2. Identité Halo liée et ownership des attempts
3. Politique de provisioning et continuité auth
4. Marqueur de sync initiale et migration des profils existants
5. Jobs persistants, single-flight et sémantique de restart
6. Durcissement mutatif

---

## Rappel pratique

Si une tâche d'implémentation touche une ligne encore floue dans cette matrice, la bonne action n'est pas de “coder pour voir”, mais de figer d'abord le contrat dans les documents coeur.
