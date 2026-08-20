# V7 — Principes d'architecture auth, sécurité & onboarding

> ⛔ Prérequis : lire d'abord [PLAN_V7_ONBOARDING_MASTER.md](PLAN_V7_ONBOARDING_MASTER.md)
> Date : 2026-04-13
> Rôle : **principes et décisions** — ce document explique les "pourquoi", pas les "comment"

---

## Séparation des 5 responsabilités

### 1. Accès à l'instance

**Responsabilité** : décider si un navigateur a le droit d'entrer sur cette instance.

Exemples : htpasswd derrière reverse proxy, forward auth / SSO, réseau privé / Tailscale / VPN.

Ce niveau protège l'application elle-même. **Il ne doit pas être confondu avec l'auth applicative.**

### 2. Session web applicative

**Responsabilité** : maintenir un contexte web serveur après entrée dans l'application.

Exigences minimales :

- cookie httpOnly, Secure en production, SameSite=Lax
- contenu de session stocké côté serveur uniquement
- aucun token Halo dans le navigateur
- identité Halo liée et joueur courant gérés côté serveur

Ce niveau ne remplace pas la barrière externe ; il organise l'expérience web.

**Hypothèse MVP** : le `SessionStore` fichier JSON actuel est conçu pour une instance **single-user**. Il n'offre pas de compare-and-swap et n'est pas adapté à des accès concurrents multi-utilisateurs. Le passage à un store plus robuste (Redis, DB) est un prérequis de la Phase 4 (multi-user avec rôles).

### 3. Liaison du compte Halo

**Responsabilité** : obtenir et rafraîchir les credentials Halo nécessaires aux syncs.

Voie standard : Microsoft Device Code Flow. Cache/token stocké côté serveur, jamais dans le navigateur.

### 4. Provisioning local

**Responsabilité** : créer un profil joueur local, sa base, ses métadonnées et son contexte initial.

Règle : piloté par une **politique admin backend** (`can_self_provision`), pas laissé ouvert par défaut.

### 5. Sync initiale

**Responsabilité** : charger les premiers matchs et rendre l'état visible à l'utilisateur.

Règle : modélisé comme un **job asynchrone avec progression métier**, pas un spinner opaque.

---

## Décisions actées

### D1 — Ne pas remplacer le garde-barrière externe avant Phase 4

**Raison** : l'existant V7 a de bonnes briques (sessions signées, Device Code Flow, JobStore) mais il manque encore :

- séparation nette entre auth web et setup produit
- politique d'autorisation formalisée côté API
- durcissement mutatif complet
- gestion claire des sessions expirées, reconnects et révocations
- stratégie multi-worker / multi-restart cohérente pour les états longs

**Conditions moyen terme** pour remplacer la barrière externe :

- plusieurs comptes applicatifs distincts
- rôles (`admin`, `member`, `viewer`)
- auto-onboarding maîtrisé
- audit des actions sensibles
- politique d'autorisation côté API
- gestion claire des sessions expirées, reconnects et révocations

Sans ces conditions, on risque de déplacer la sécurité du proxy vers une implémentation plus faible.

**Conclusion** : améliorer l'onboarding applicatif maintenant, supprimer la protection externe uniquement quand Phase 4 est validée.

### D2 — Une seule machine d'état produit : `GET /bootstrap` + `setup_state`

Le shell React final ne doit pas arbitrer entre deux machines d'état concurrentes.

Règle :

- `setup_state` est la vérité produit pour l'onboarding final
- `GET /setup/status` et `next_blocking_step` restent des surfaces legacy / transitoires
- `POST /setup/smoke-test` n'est pas le contrat final de la première sync V7

### D3 — Politique de provisioning backend-first basée sur l'identité liée

Le backend expose `can_self_provision` dans `CapabilityMap`. Comportements :

- `false` → l'utilisateur peut lier son compte Halo, mais pas créer un profil sans intervention admin
- `true` → l'utilisateur peut finaliser la création de son profil local

**Variante utile (Phase 4)** : flag `auto_provision_from_halo_identity`. Déplacé en Phase 4 — pour le MVP single-user, l'auto-provision est le seul mode utile. Ce flag n'a de valeur que dans un scénario multi-user avec admin.

**Interdit** :

- laisser le frontend décider seul
- créer un profil Xbox à partir d'un simple champ texte sans comparaison backend avec l'identité liée
- exposer un mode admin purement côté client

### D4 — Le Device Code Flow doit marcher parfaitement du premier coup

C'est le moment où l'utilisateur décide si l'app est crédible. Critère : un utilisateur non-technique complète la liaison Xbox en < 2 minutes, sans ressaisie du gamertag.

Exigences minimales :

- une seule tentative active par session
- ownership strict de la tentative par la session qui l'a créée
- expiration et purge explicites
- retry UX claire

La persistance des tentatives Device Code au restart n'est **pas** un prérequis absolu du MVP onboarding, mais elle reste un prérequis avant Phase 4 ou avant une architecture multi-worker.

### D5 — Première sync = vrai job avec progression métier et sémantique de restart

Un spinner opaque pendant 3-4 minutes = l'utilisateur ferme l'onglet. Il faut :

- des compteurs concrets (37/200 matchs)
- des phases sémantiques (préparation, auth, récupération, enrichissement, finalisation)
- un état persistant
- une sémantique explicite si le serveur redémarre

Tant qu'une vraie reprise n'est pas implémentée, le contrat minimal est : **job rechargé comme interrompu, visible par l'UI, relançable proprement**.

### D6 — Continuité auth entre liaison Halo, provisioning et première sync

Une auth réussie avant création du profil doit rester exploitable après provisioning.

Cela impose :

- un état serveur `linked_halo_identity`
- un transfert explicite du cache MSAL vers la player DB au moment opportun
- une mise à jour du joueur courant en session après création du profil

Sans cela, le parcours “auth OK → profil créé → première sync” reste cassable malgré une bonne UX de surface.

---

## Architecture cible — 5 couches

| Couche | Source de vérité | Rôle |
|--------|-----------------|------|
| **1 — Garde-barrière** | Reverse proxy / réseau privé | Limiter l'exposition publique |
| **2 — Session web** | Backend FastAPI (`SessionData`) | Joueur courant, locale, flags UI, `auth_ready`, `linked_halo_identity` |
| **3 — Liaison Halo** | Couche auth Python (`src/auth/`) | Device Code Flow, résolution gamertag/xuid, cache MSAL |
| **4 — Provisioning** | Service setup (`setup_service.py`) | Créer profil, aligner session, assurer la continuité auth, clarifier l'init DB |
| **5 — Jobs longs** | Store de jobs persistant | Sync initiale, backfill, reindex médias, tâches longues similaires |

---

## Exigences de sécurité

### Secrets uniquement côté serveur

Interdits côté navigateur : refresh tokens, Spartan tokens, clearance tokens, cache MSAL complet.

### Cookies de session durcis

- httpOnly ✅ (déjà implémenté)
- Secure en production ✅ (déjà implémenté)
- SameSite=Lax ✅ (déjà implémenté)
- TTL 7 jours ✅ (déjà implémenté)

### Protection des routes mutantes

Routes sensibles : `POST /auth/device-flow/start`, `GET /auth/device-flow/{attempt_id}`, `POST /setup/players`, `POST /sync/initial`, `PATCH /settings`.

Mesures minimales :

- contrôle de capacité backend
- vérification same-origin / `Origin` ou `Referer` sur toutes les routes mutantes portées par cookie
- rate limiting basique

Si un cas cross-site légitime apparaît plus tard, il faudra ajouter une protection CSRF explicite. En l'état, le contrat minimal ne doit pas rester au stade de “si nécessaire”.

### Ownership des tentatives Device Code

Une tentative Device Code Flow appartient à la session qui l'a créée.

Conséquences :

- une autre session ne doit pas pouvoir la lire
- une tentative inconnue ou étrangère retourne comme inconnue
- le backend doit gérer expiration, purge et single-flight

### Journalisation

Tracer sans fuiter les secrets : démarrage/succès/échec Device Code Flow, provisioning profil, démarrage/fin sync, erreurs métier.

### Politique d'accès côté API

Une route sensible ne doit **jamais** dépendre uniquement de l'affichage ou non d'un bouton dans le frontend.

---

## Parcours utilisateur cible

### Premier lancement (instance déjà protégée)

1. L'utilisateur passe la barrière d'instance
2. Il arrive sur un écran d'entrée propre et non technique
3. L'app appelle `GET /bootstrap`
4. Elle déduit l'étape courante depuis `setup_state`
5. Elle n'utilise pas un deuxième endpoint produit pour arbitrer le flux

### États `setup_state`

| Valeur | Signification | Action UI |
|--------|--------------|-----------|
| `no_halo_link` | Aucun compte Halo lié | Afficher Device Code Flow (code, lien, QR code idéalement, explication simple) |
| `halo_linked_no_profile` | Auth OK, aucun profil local | Afficher confirmation profil à partir de `linked_halo_identity` |
| `profile_ready_no_sync` | Profil créé, sync jamais faite | Lancer sync initiale |
| `ready` | Tout opérationnel | Accès au dashboard |

### Écran d'entrée — blocs recommandés

- **Hero** : titre simple + état courant
- **Carte principale** : une seule action selon l'étape (connecter Xbox / confirmer profil / lancer sync / ouvrir LevelUp)
- **Panneau secondaire** : aide non technique (stockage local, durée sync, suite)
- **Footer** : état backend + joueur détecté + dernière erreur actionnable

### Détail important pour `halo_linked_no_profile`

L'écran ne doit pas redemander une identité libre. Il doit afficher :

- le `gamertag` déjà résolu
- éventuellement le `xuid`
- un bouton de confirmation
- un message d'erreur actionnable si le backend refuse le provisioning
