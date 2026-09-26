# Plan V7 — Onboarding, Auth & Sécurité — Document Maître

> Date : 2026-04-13
> Statut : actif
> Périmètre : façade FastAPI + React de V7

---

## ⛔ Règle de lecture obligatoire

Tout agent IA ou développeur qui touche au domaine **auth, setup, onboarding, bootstrap, Device Code Flow, provisioning joueur ou sync initiale** doit lire les documents ci-dessous **dans l'ordre** avant de coder.

### Ordre de lecture

| # | Document | Contenu | Obligatoire |
|---|----------|---------|:-----------:|
| 1 | **Ce fichier** | Vue d'ensemble, priorités, critères de succès | ✅ |
| 2 | [PLAN_V7_AUTH_SECURITY_PRINCIPLES.md](PLAN_V7_AUTH_SECURITY_PRINCIPLES.md) | Principes d'architecture, décisions, exigences sécurité | ✅ |
| 3 | [SPEC_V7_BOOTSTRAP_CONTRACT.md](SPEC_V7_BOOTSTRAP_CONTRACT.md) | Contrats API (bootstrap, setup_state, provisioning, jobs) | ✅ |
| 4 | [IMPL_V7_ONBOARDING.md](IMPL_V7_ONBOARDING.md) | Sprints, tâches concrètes, état réel du code | ✅ |

### Documents complémentaires

| # | Document | Contenu | Obligatoire |
|---|----------|---------|:-----------:|
| 5 | [TABLE_V7_ONBOARDING_CONTRACTS.md](TABLE_V7_ONBOARDING_CONTRACTS.md) | Matrice des contrats à figer (source de vérité, owner, tests, sortie) | Recommandé |
| 6 | [CHECKLIST_V7_ONBOARDING_GO_NO_GO.md](CHECKLIST_V7_ONBOARDING_GO_NO_GO.md) | Checklist de lancement / revue / merge du chantier | Recommandé |

**Ne pas commencer à implémenter sans avoir lu les 4 documents coeur.**

**Source de vérité** : ce corpus remplace les décisions dispersées dans `PLAN_V7_AUTH_SECURITY_ONBOARDING.md`. Ce document historique peut rester utile comme archive de travail, mais il ne fait plus foi s'il diverge du présent corpus.

---

## Résumé exécutif

L'onboarding V7 sépare 5 responsabilités distinctes :

1. **Accès instance** — garde-barrière externe (htpasswd / proxy / VPN)
2. **Session web** — cookie httpOnly et état serveur côté backend FastAPI
3. **Liaison Halo** — Device Code Flow Microsoft → identité Halo liée (`gamertag` + `xuid`)
4. **Provisioning** — création profil local + continuité auth + joueur courant
5. **Sync initiale** — job long avec progression métier visible

Les deux décisions structurantes sont :

1. **ne pas remplacer le garde-barrière externe** tant que la Phase 4 n'est pas validée
2. **faire de `GET /bootstrap` + `setup_state` la machine d'état produit unique** pour l'onboarding React

---

## Décisions de cadrage actées

### 1. Machine d'état produit unique

Le shell React final route l'onboarding uniquement depuis `GET /bootstrap` et `setup_state`.

Conséquence :

- `GET /setup/status` reste une surface **legacy / transitoire / dev**
- `POST /setup/smoke-test` ne pilote pas le parcours produit final
- aucune nouvelle logique produit ne doit dépendre de `next_blocking_step`

### 2. Identité Halo authoritative côté backend

Après succès du Device Code Flow, l'identité Halo liée devient un état serveur explicite :

- stockée côté session backend
- exposée au frontend via bootstrap pour confirmation UI
- utilisée comme source de vérité pour `POST /setup/players`

Conséquence : le backend ne fait **jamais** confiance à un `gamertag` ou `xuid` librement fourni par le client pour créer un profil Xbox.

### 3. Première sync réussie = marqueur persistant explicite

Le passage à `profile_ready_no_sync` puis `ready` ne doit pas reposer sur une heuristique fragile du type “il existe déjà des matchs dans shared”.

La source de vérité cible est un marqueur persistant en DB joueur, par exemple dans `sync_meta` :

- `initial_sync_completed_at`
- ou `last_successful_sync_at`

Conséquence : un backfill de migration est requis pour les profils déjà existants, afin que les joueurs historiques ne repassent pas artificiellement par l'onboarding.

### 4. Jobs longs avec sémantique explicite au restart

La persistance d'un job ne suffit pas. Il faut aussi définir ce qu'il devient après redémarrage du serveur :

- `interrupted`
- `restart_required`
- ou vraie reprise si des checkpoints existent et sont testés

Tant que la vraie reprise n'est pas implémentée, la sémantique minimale attendue est : **job rechargé, marqué comme interrompu, relançable proprement**.

### 5. Continuité auth entre setup et première sync

Une auth réussie avant création du profil doit rester exploitable après provisioning.

Conséquence : la création du profil doit gérer explicitement :

- le transfert / la persistance du cache MSAL côté joueur
- la mise à jour du joueur courant en session
- la capacité de lancer la première sync sans demander une nouvelle auth implicite

---

## Priorités

| Priorité | Étape | Risque si raté |
|:--------:|-------|----------------|
| **P0** | Machine d'état unique via `GET /bootstrap` | UX incohérente, double logique, bugs en cascade |
| **P0** | Device Code Flow impeccable + ownership de tentative par session | Abandon utilisateur immédiat, fuite d'état inter-session |
| **P0** | Provisioning backend-authoritative basé sur l'identité Halo liée | Faille de sécurité, profil créé sur identité incohérente |
| **P1** | Marqueur explicite de sync initiale + migration des profils existants | Routage faux entre `profile_ready_no_sync` et `ready` |
| **P1** | Jobs sync persistants avec progression métier et sémantique de restart | UX dégradée, perte d'état, relances floues |
| **P2** | Durcissement mutatif (same-origin / CSRF, rate limit, logs) | Surface d'attaque inutilement large |
| **P2** | Garde-barrière externe maintenu jusqu'à évaluation Phase 4 | Exposition prématurée de l'instance |

---

## Critères de succès par sprint

### Sprint 1 — Fondations (P0)

- [ ] Un utilisateur non-technique complète la liaison Xbox en < 2 minutes, sans ressaisie
- [ ] Le frontend route vers le bon écran avec un seul `GET /bootstrap`
- [ ] `GET /setup/status` n'est plus la source de vérité du shell d'onboarding final
- [ ] L'identité Halo liée est disponible côté serveur puis renvoyée par bootstrap
- [ ] Une tentative Device Code Flow n'est lisible que par la session qui l'a créée

### Sprint 2 — Sécurité provisioning (P1)

- [ ] `POST /setup/players` refuse avec 403 si `can_self_provision=false`, indépendamment du frontend
- [ ] `POST /setup/players` refuse une identité incohérente avec l'identité Halo liée en session
- [ ] Le gamertag résolu par le Device Code Flow est pré-rempli côté UI comme confirmation, pas comme saisie libre
- [ ] La création du profil met à jour `session.current_player_slug` et rend la première sync immédiatement possible

### Sprint 3 — Sync initiale (P1)

- [ ] `profile_ready_no_sync` et `ready` reposent sur un marqueur persistant explicite en DB joueur
- [ ] Les profils existants sont migrés / backfillés pour ne pas casser l'expérience au rollout
- [ ] L'écran affiche des compteurs métier réels (37/200 matchs récupérés)
- [ ] Une seule sync initiale active par joueur peut exister à la fois
- [ ] Si le serveur redémarre pendant une sync, le job rechargé est visible comme interrompu et relançable proprement

### Sprint 4 — Durcissement (P2)

- [ ] Cookies : httpOnly + Secure en prod + SameSite=Lax + TTL explicite
- [ ] Protection same-origin / CSRF clairement définie sur toutes les routes mutantes portées par cookie
- [ ] Rate limiting sur routes sensibles (5 req/min)
- [ ] Logs structurés sans secrets

---

## Points de vigilance

- **Pas de deuxième machine d'état produit** : pas de coexistence durable entre `setup_state` et `next_blocking_step`
- **Pas de trust client sur l'identité** : `gamertag` / `xuid` ne sont pas des vérités backend
- **Pas d'heuristique silencieuse pour la première sync** : `shared.match_participants` peut servir de migration, pas de source de vérité pérenne
- **Pas de faux “job persistant”** : un job rechargé sans sémantique claire de reprise reste un contrat incomplet
- **Pas de promesse ambiguë sur le provisioning** : le contrat doit dire explicitement si `POST /setup/players` crée la DB joueur immédiatement ou garantit seulement sa création avant la première sync- **Mode démo court-circuité** : en mode démo, `setup_state = "ready"` directement — pas de Device Code Flow, provisioning ni sync initiale
- **Rollback par sprint** : chaque sprint doit pouvoir être reverté indépendamment sans casser l'état existant. Les guards backend refusent proprement, pas de migration irréversible sans dry-run préalable
- **Wizard legacy supprimé à la fin du plan** : l'ancien wizard React (5 étapes via `next_blocking_step`) est supprimé après stabilisation du Sprint 4, pas de coexistence durable
- **Hypothèse single-user** : le `SessionStore` fichier JSON n'est pas adapté au multi-user concurrent — Phase 4 si besoin
---

## Facteurs de réussite et d'échec

### Produit

**Facteur de réussite** : un seul parcours compréhensible, une seule prochaine action affichée, une machine d'état lisible pour l'utilisateur.

**Facteur d'échec** : deux parcours parallèles, un onboarding trop technique, ou une promesse produit floue sur ce qui se passe après la liaison Xbox.

**À anticiper** : valider très tôt les 4 états visibles, le texte exact des CTA, et le temps réel du parcours complet sur un utilisateur non-technique.

### Sécurité

**Facteur de réussite** : garde-barrière externe conservé, guards backend réels, identité Halo authoritative côté serveur, protection same-origin claire sur les routes mutantes.

**Facteur d'échec** : confiance implicite au frontend, bouton masqué pris pour une autorisation, tentative Device Code visible d'une autre session, ou provisioning accepté sur une identité incohérente.

**À anticiper** : tests `403` et `409` systématiques, petite revue de menace avant chaque sprint sensible, et politique explicite sur ce qui est stocké en session et en base joueur.

### Backend / API

**Facteur de réussite** : bootstrap comme source de vérité unique, marqueur persistant de sync réussie, idempotence sur provisioning et sync, single-flight par joueur.

**Facteur d'échec** : heuristiques fragiles, schémas qui divergent du code réel, jobs fantômes après restart, ou continuité auth supposée mais non câblée.

**À anticiper** : verrouiller les schémas avant d'implémenter, écrire les effets de bord attendus de chaque endpoint, et décider noir sur blanc ce que signifie `interrupted`, relançable ou repris.

### Frontend / UX

**Facteur de réussite** : UI pilotée par bootstrap, confirmation au lieu de ressaisie, progression métier visible, erreurs actionnables.

**Facteur d'échec** : frontend qui reconstruit sa propre logique, spinner opaque, polling qui masque les erreurs, dépendance résiduelle au flow legacy.

**À anticiper** : un E2E complet du premier onboarding, un refresh navigateur en plein parcours, et un restart serveur pendant l'affichage de la progression.

### Migration / Données

**Facteur de réussite** : migration des profils existants avant bascule produit, source de vérité claire pour `profile_ready_no_sync` et `ready`, rollout réversible.

**Facteur d'échec** : profils historiques renvoyés à tort vers la première sync, doublons de profils, ou DB joueur considérée prête alors qu'elle ne l'est pas.

**À anticiper** : backfill explicite, dry-run sur un échantillon de vrais profils, métriques avant / après migration, et règle de rollback claire.

### Exploitation / Run

**Facteur de réussite** : comportement clair derrière reverse proxy, logs structurés, redémarrage serveur traité comme cas normal, runbook simple.

**Facteur d'échec** : HTTPS mal détecté, cookie mal posé en prod, rate limit absent, job rechargé dans un faux état `running`, diagnostic impossible sur incident réel.

**À anticiper** : test de restart volontaire, test prod-like derrière proxy, validation des cookies en conditions réelles, et journalisation lisible des actions critiques.

### Pilotage

**Facteur de réussite** : owner clair par contrat, ordre des sprints respecté, pas d'implémentations parallèles sur un contrat encore flou.

**Facteur d'échec** : plusieurs développements avancent sur bootstrap, provisioning et sync en même temps alors que les invariants ne sont pas figés.

**À anticiper** : revue Go / No-Go avant chaque sprint, usage systématique de la matrice de contrats, et refus d'ouvrir un sous-chantier si sa source de vérité n'est pas tranchée.

---

## Matrice des risques opérationnels

| ID | Domaine | Risque | Gravité | Probabilité | Mitigation |
|----|---------|--------|:-------:|:-----------:|------------|
| R1 | Produit | Double machine d'état (`bootstrap` vs `setup/status`) | Haute | Haute | Faire de `GET /bootstrap` la seule source de vérité produit et reclasser explicitement les endpoints legacy |
| R2 | Sécurité | Provisioning Xbox accepté sur une identité client incohérente | Haute | Haute | Comparer toute requête de provisioning à `linked_halo_identity` côté backend et refuser en `409` |
| R3 | Backend | `profile_ready_no_sync` dérivé par heuristique fragile | Haute | Haute | Introduire un marqueur persistant en `sync_meta` et backfiller les profils historiques |
| R4 | Jobs | Redémarrage serveur laissant un faux job `running` | Haute | Moyenne | Persister les jobs et convertir les jobs incomplets en `interrupted` tant qu'il n'y a pas de vraie reprise |
| R5 | Auth | Rupture de continuité auth entre Device Code Flow, provisioning et première sync | Haute | Moyenne | Persister / transférer explicitement l'état MSAL utile au moment du provisioning |
| R6 | Frontend | Ressaisie inutile du gamertag après auth réussie | Moyenne | Haute | Afficher une carte de confirmation pilotée par `linked_halo_identity` |
| R7 | Migration | Profils existants renvoyés à tort dans l'onboarding de sync | Haute | Moyenne | Prévoir un dry-run et un backfill avant rollout complet |
| R8 | Sécurité mutative | Routes mutantes protégées seulement par l'UI | Haute | Moyenne | Ajouter guards backend, protection same-origin et rate limiting minimal |
| R9 | Exploitation | Mauvaise détection HTTPS derrière proxy | Moyenne | Moyenne | Valider les cookies et la config reverse proxy en conditions prod-like |
| R10 | Pilotage | Implémentations parallèles sur des contrats encore flous | Moyenne | Haute | Utiliser la matrice de contrats et faire un Go / No-Go avant chaque sprint |

---

## Décision clé

> On améliore fortement l'expérience produit maintenant, sans affaiblir la sécurité de l'instance pendant la transition.

- ✅ Belle page d'entrée moderne
- ✅ Onboarding piloté par bootstrap
- ✅ Confirmation de profil depuis l'identité Halo déjà résolue
- ✅ Vraie UX de première sync avec progression métier
- ❌ Pas de remplacement de htpasswd par un pseudo-login applicatif partiel (Phase 4, conditionnel)
