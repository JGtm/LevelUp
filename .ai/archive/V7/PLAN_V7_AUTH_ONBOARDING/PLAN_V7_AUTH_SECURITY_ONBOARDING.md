# Plan V7 — Sécurité d'accès, onboarding et première sync

> Statut : proposition cadrée
> Date : 2026-04-13
> Périmètre : façade FastAPI + React de V7

---

## TL;DR

La bonne direction pour V7 n'est pas de remplacer immédiatement le garde-barrière serveur par un simple login applicatif maison.

La trajectoire recommandée est :

1. conserver une barrière d'accès externe pour l'instance (`htpasswd` aujourd'hui, idéalement un vrai proxy auth/SSO plus tard)
2. ajouter dans l'application un écran d'entrée moderne qui gère la liaison du compte Halo, la création du profil local et la première sync
3. traiter la première sync comme un job long suivi côté API, avec une progression exploitable par le frontend et des messages pensés pour un utilisateur final
4. n'envisager une auth applicative complète en remplacement du garde-barrière externe que si l'on assume vraiment un produit multi-utilisateur avec rôles, politique d'accès et audit

---

## Problème produit

Le besoin réel mélange aujourd'hui quatre sujets différents :

1. Contrôle d'accès à l'instance : qui peut ouvrir l'application sur ce serveur
2. Connexion Halo : quel compte Microsoft / Xbox est lié à l'application
3. Provisioning local : quels profils joueurs et quelles bases locales existent
4. Première ingestion de données : comment charger les premiers matchs sans laisser l'utilisateur dans le flou

Le risque principal serait de fusionner ces sujets dans un faux login unique qui serait joli en façade, mais fragile en sécurité et confus en architecture.

---

## Séparation des responsabilités

### 1. Accès à l'instance

Responsabilité : décider si un navigateur a le droit d'entrer sur cette instance.

Exemples :

- htpasswd derrière reverse proxy
- forward auth / SSO du proxy
- réseau privé / Tailscale / VPN

Ce niveau protège l'application elle-même.

### 2. Session web applicative

Responsabilité : maintenir un contexte web serveur après entrée dans l'application.

Exigences minimales :

- cookie httpOnly
- cookie Secure en production
- SameSite=Lax a minima
- contenu de session stocké côté serveur uniquement

Ce niveau ne remplace pas forcément la barrière externe ; il organise surtout l'expérience web dans l'application.

### 3. Liaison du compte Halo

Responsabilité : obtenir et rafraîchir les credentials Halo nécessaires aux syncs.

Voie standard recommandée :

- Microsoft Device Code Flow
- cache/token stocké côté serveur, jamais dans le navigateur

### 4. Provisioning local

Responsabilité : créer un profil joueur local, sa base, ses métadonnées et son contexte initial.

Cette capacité doit être pilotée par une politique admin, pas laissée ouverte par défaut.

### 5. Sync initiale

Responsabilité : charger les premiers matchs, enrichir les données et rendre l'état visible à l'utilisateur.

Cela doit être modélisé comme un job asynchrone avec progression, pas comme un spinner opaque.

---

## Recommandation produit

## Recommandation court terme

Conserver le garde-barrière externe pour l'accès à l'instance, et construire dans V7 un vrai flux applicatif pour :

- l'accueil visuel
- la connexion Halo
- la création du profil local
- la première sync guidée

En pratique, cela donne :

1. l'utilisateur passe la barrière d'instance
2. il arrive sur un écran Bienvenue dans LevelUp
3. l'application détecte s'il manque une liaison Halo, un profil joueur, ou une sync initiale
4. elle l'emmène dans le bon sous-flux avec des étapes simples

## Recommandation moyen terme

Ne remplacer la barrière externe par une auth applicative complète que si les exigences suivantes deviennent vraies :

- plusieurs comptes applicatifs distincts
- rôles (`admin`, `member`, `viewer`)
- auto-onboarding maîtrisé
- audit des actions sensibles
- politique d'autorisation côté API
- gestion claire des sessions expirées, reconnects et révocations

Sans cela, on risque de déplacer la sécurité du proxy vers une implémentation plus faible.

---

## Pourquoi ne pas remplacer immédiatement htpasswd

L'existant V7 web contient déjà de bonnes briques, mais ce n'est pas encore un remplacement complet du garde-barrière serveur.

### Ce qui est déjà bon

- sessions serveur + cookie signé côté backend
- machine d'état de setup
- Device Code Flow côté API
- création de profil côté API
- jobs asynchrones avec polling

### Ce qui manque encore pour parler de vraie sécurité d'accès applicative

1. L'auth web et le setup sont encore trop confondus

Le système actuel raisonne surtout en termes de setup_required, pas en termes d'accès applicatif autorisé / refusé.

2. Les états critiques sont encore process-level en mémoire

Les attempts de Device Code Flow et les jobs longs sont conservés en mémoire du process. En cas de redémarrage, l'état disparaît.

3. La politique d'autorisation n'est pas encore formalisée

Il manque des capacités explicites du type :

- can_self_provision
- can_start_initial_sync
- can_manage_instance

4. Le flux setup n'est pas encore pleinement cohérent côté UX

Exemples de dette actuelle :

- l'écran web n'affiche pas correctement le vrai code Device Code
- après résolution du compte Halo, l'utilisateur peut encore devoir ressaisir son gamertag alors que le backend l'a déjà

Conclusion :

Le bon mouvement immédiat est d'améliorer l'onboarding applicatif, pas de supprimer trop tôt la protection externe.

---

## Architecture cible recommandée

## Couche 1 — Garde-barrière d'instance

Source de vérité : reverse proxy / réseau privé / SSO infra.

Rôle :

- limiter l'exposition publique
- éviter qu'une auth applicative naissante soit la seule ligne de défense

## Couche 2 — Session web applicative

Source de vérité : backend FastAPI.

Rôle :

- stocker le joueur courant
- stocker la locale
- stocker les flags UI utiles
- mémoriser l'état de liaison Halo côté serveur

Règles :

- aucun token Halo dans le navigateur
- aucune donnée sensible dans localStorage
- rotation/expiration gérées côté serveur

## Couche 3 — Liaison Halo

Source de vérité : couche auth Python existante.

Rôle :

- lancer le Device Code Flow
- récupérer gamertag et xuid résolus
- persister le cache côté serveur / base joueur

## Couche 4 — Provisioning local

Source de vérité : service de setup.

Rôle :

- créer le profil dans db_profiles.json
- créer la structure data/players/{slug}
- initialiser la base DuckDB du joueur

## Couche 5 — Jobs longs

Source de vérité : store de jobs persistant.

Rôle :

- première sync
- smoke test
- backfill
- réindexation médias

Recommandation importante :

Le store de jobs ne doit pas rester purement en mémoire si l'on veut une UX robuste en production.

---

## Politique de provisioning admin

Le provisioning automatique in-app est une bonne idée si et seulement si il est gouverné par une politique explicite.

### Règle proposée

Le backend expose une capacité booléenne dans le bootstrap :

- can_self_provision

Comportements :

- false : l'utilisateur peut lier son compte Halo, mais pas créer un nouveau profil local sans intervention admin
- true : l'utilisateur peut finaliser lui-même la création de son profil local

### Variante utile

Ajouter un second flag optionnel :

- auto_provision_from_halo_identity

Si actif, après succès du Device Code Flow :

- le backend récupère gamertag et xuid
- propose une confirmation
- crée directement le profil sans ressaisie manuelle

### Ce qu'il faut éviter

- laisser le frontend décider seul s'il peut créer un profil
- autoriser la création par simple champ texte sans politique backend
- exposer un mode admin purement côté client

---

## Parcours utilisateur cible

## A. Premier lancement sur une instance déjà protégée

1. L'utilisateur ouvre LevelUp
2. L'app affiche un écran d'entrée propre et non technique
3. L'app détecte l'étape bloquante courante
4. L'utilisateur suit un flux court

États possibles :

- no_halo_link : aucun compte Halo lié
- halo_linked_no_profile : compte Halo lié, aucun profil local
- profile_ready_no_sync : profil créé, sync initiale non faite
- ready : application prête

## B. Connexion Halo

1. L'utilisateur clique sur Connecter mon compte Xbox
2. L'application affiche le Device Code, un lien, idéalement un QR code, et une explication simple
3. Le backend poll le flow
4. À succès, l'app affiche l'identité résolue : gamertag + confirmation

## C. Création du profil local

Si can_self_provision=true :

1. l'utilisateur confirme le profil détecté
2. l'app crée la DB locale et le profil
3. l'app passe automatiquement à l'étape de sync

Si can_self_provision=false :

1. l'app indique clairement que le compte est lié mais que l'activation doit être faite par l'admin
2. elle n'affiche pas un faux bouton cassé

## D. Première sync

1. l'utilisateur voit un écran dédié Préparation de vos données
2. l'app lance un job de sync initiale
3. l'écran montre la progression réelle
4. à la fin, l'app redirige vers l'accueil

---

## Première sync — UX recommandée

Le retour utilisateur actuel ne doit pas être un simple spinner ou une barre floue.

Il faut un écran orienté résultat, avec une sémantique métier compréhensible.

## Informations à afficher

### 1. Phase courante

Exemples :

- Connexion au service Halo
- Récupération de vos matchs récents
- Analyse des statistiques détaillées
- Préparation de votre tableau de bord

### 2. Compteurs concrets

Exemples :

- 37 / 200 matchs récupérés
- 12 / 37 matchs enrichis

### 3. Estimation grossière

Exemples :

- Environ 2 à 4 minutes restantes
- La première sync est la plus longue ; les suivantes seront plus rapides

### 4. Ce qui se passe vraiment

Une ligne d'explication sobre :

- Nous téléchargeons vos matchs, puis nous calculons les indicateurs utilisés dans l'application.

### 5. Gestion des warnings

Exemples :

- Le service Halo répond lentement, la sync continue
- Certaines données secondaires seront enrichies ensuite

### 6. Issue finale claire

Résultats finaux compréhensibles :

- nombre de matchs importés
- éventuels enrichissements différés
- bouton Ouvrir l'application

---

## Modèle de job recommandé pour la sync initiale

Le contrat générique actuel (`progress_pct`, `current_step`) est une bonne base, mais il faut l'enrichir pour un vrai onboarding.

## Champs recommandés

- job_id
- job_type
- status
- progress_pct
- phase_key
- phase_label
- current_step
- matches_done
- matches_total
- subtasks_done
- subtasks_total
- eta_seconds
- warnings
- result
- error

## Phases suggérées

1. prepare
2. auth
3. fetch_matches
4. enrich
5. verify
6. finalize

## Recommandation de stockage

Pour la prod :

- persistance en DuckDB ou en fichiers JSON robustes côté serveur
- pas uniquement en mémoire process-level

---

## Exigences de sécurité minimales

## 1. Secrets uniquement côté serveur

Interdits côté navigateur :

- refresh tokens
- Spartan tokens
- clearance tokens
- contenu complet du cache MSAL

## 2. Cookies de session durcis

Minimum :

- httpOnly
- Secure en production
- SameSite=Lax
- TTL explicite

## 3. Protection des routes mutantes

Les routes suivantes doivent être traitées comme sensibles :

- démarrage Device Code Flow
- création de profil
- démarrage sync/backfill
- modifications settings d'instance

Mesures recommandées :

- contrôle de capacité backend
- vérification d'origine / CSRF si nécessaire
- rate limiting basique

## 4. Journalisation utile

Tracer sans fuiter les secrets :

- démarrage d'un flow auth
- succès/échec du flow
- provisioning profil
- démarrage/fin de sync
- erreurs métier principales

## 5. Politique d'accès côté API

Une route sensible ne doit jamais dépendre uniquement de l'affichage ou non d'un bouton dans le frontend.

---

## Contrat fonctionnel minimal à ajouter côté bootstrap

Pour rendre le frontend V7 cohérent, le bootstrap devrait idéalement exposer :

- access_state: open | blocked | limited
- auth_state: missing | partial | ready
- setup_state: no_halo_link | halo_linked_no_profile | profile_ready_no_sync | ready
- capabilities.can_self_provision
- capabilities.can_start_initial_sync
- capabilities.can_manage_instance

Le frontend n'aurait alors plus à déduire des états implicites à partir de plusieurs indices partiels.

---

## Contrat UX recommandé pour l'écran d'entrée V7

L'écran d'entrée peut reprendre une direction visuelle plus ambitieuse, mais il doit rester très lisible.

## Blocs recommandés

### Hero

- titre simple
- promesse produit
- état courant de l'instance

### Carte principale

Affiche une seule action principale selon l'étape :

- connecter mon compte Xbox
- confirmer mon profil
- lancer la préparation initiale
- ouvrir LevelUp

### Panneau secondaire

Texte d'aide non technique :

- ce qui est stocké localement
- pourquoi la première sync prend du temps
- ce que fera l'app ensuite

### Footer d'état

- état du backend
- joueur détecté si connu
- dernière erreur actionnable si présente

---

## Feuille de route recommandée

## Phase 1 — Consolider l'onboarding actuel

Objectif : rendre l'existant crédible sans toucher à la sécurité externe.

À faire :

1. corriger l'affichage réel du Device Code dans l'écran setup
2. éviter la ressaisie du gamertag si l'API l'a déjà résolu
3. finaliser proprement le contexte de session après succès
4. clarifier les états setup vs ready

## Phase 2 — Ajouter la politique admin de provisioning

Objectif : permettre le wizard créer mon compte / ma DB sans trou de sécurité.

À faire :

1. ajouter can_self_provision au bootstrap
2. verrouiller POST /setup/players côté backend selon cette capacité
3. ajouter la confirmation de l'identité Halo avant création

## Phase 3 — Remplacer le smoke test par une vraie sync initiale produit

Objectif : rendre la première expérience réellement utilisable.

À faire :

1. créer un job initial_sync
2. exposer des compteurs métier dans le statut du job
3. afficher un écran de progression orienté utilisateur final

## Phase 4 — Évaluer le remplacement du garde-barrière externe

Objectif : décider en connaissance de cause si l'app peut devenir sa propre porte d'accès.

Condition préalable :

- sessions stables
- états persistants
- capacités d'autorisation claires
- logs suffisants
- UX d'erreur/reconnexion solide

---

## Décision recommandée

La décision la plus pragmatique pour V7 est :

- oui à une belle page d'entrée moderne dans l'application
- oui à un wizard in-app pour créer le profil et la DB si l'admin l'autorise
- oui à une vraie UX de première sync avec progression compréhensible
- non, pas tout de suite, au remplacement pur et simple de htpasswd par un pseudo-login applicatif partiel

Autrement dit :

on améliore fortement l'expérience produit maintenant, sans affaiblir la sécurité de l'instance pendant la transition.

1. Ne pas confondre "accès instance" et "auth applicative"
C'est la décision structurante. Si vous supprimez htpasswd trop tôt sans avoir des sessions robustes, une politique d'autorisation backend, et un audit minimum, vous ouvrez l'instance. Le plan le dit bien, mais dans la pratique la tentation de "simplifier" en fusionnant les deux est forte.

Critère de succès : le garde-barrière externe reste en place jusqu'à ce que la Phase 4 soit explicitement validée.

2. Le Device Code Flow doit marcher parfaitement du premier coup
C'est le moment où l'utilisateur décide si l'app est crédible ou non. Aujourd'hui le plan note que :

le code n'est pas affiché correctement côté web
le gamertag est re-demandé alors que le backend l'a déjà
Si ce flux est bancal, aucun utilisateur ne terminera l'onboarding. C'est la Phase 1, et elle conditionne tout.

Critère de succès : un utilisateur non-technique complète la liaison Xbox en < 2 minutes, sans ressaisie.

3. can_self_provision côté backend, pas frontend
La politique de provisioning doit vivre côté API. Si le frontend décide seul qui peut créer un profil, c'est un trou de sécurité.

Critère de succès : POST /setup/players refuse avec 403 si can_self_provision=false, indépendamment de ce que le frontend envoie.

4. La première sync doit être un vrai job avec progression métier
Un spinner opaque pendant 3-4 minutes = l'utilisateur ferme l'onglet. Le plan propose des compteurs concrets (37/200 matchs récupérés) — c'est exactement ce qu'il faut, mais ça implique :

un store de jobs persistant (pas en mémoire process)
des phases sémantiques exposées via l'API
un frontend qui poll et affiche la progression
Critère de succès : si le serveur redémarre pendant une sync, l'état du job survit et l'utilisateur peut reprendre.

5. Le bootstrap expose un contrat d'état clair
Le bloc proposé (access_state, auth_state, setup_state, capabilities) est le cœur de la cohérence frontend. Sans ça, chaque composant React va réinventer sa propre détection d'état.

Critère de succès : le frontend n'a jamais besoin de déduire un état à partir de plusieurs appels — un seul GET /bootstrap suffit.

Résumé par priorité
Priorité	Étape	Risque si raté
P0	Device Code Flow impeccable (Phase 1)	Abandon utilisateur immédiat
P0	Bootstrap avec contrat d'état unique	UX incohérente, bugs en cascade
P1	Politique provisioning côté backend (Phase 2)	Faille de sécurité
P1	Jobs sync persistants avec progression (Phase 3)	UX dégradée, perte d'état au restart
P2	Garde-barrière externe maintenu (Phase 4)	Exposition prématurée de l'instance
Les phases 1 et 2 sont le socle. Si elles sont solides, les phases 3 et 4 deviennent des améliorations progressives plutôt que des reconstructions.

ok mais si tu devais me traduire ça en étapes actionnables ?

Let me d'abord regarder l'état actuel du code V7 pour ancrer les étapes dans la réalité.

Voici le plan d'action concret, ancré dans l'état réel du code V7 :

Étapes actionnables — V7 Auth, Sécurité & Onboarding
Phase 1 — Device Code Flow impeccable + Bootstrap cohérent
Prérequis : c'est ce que l'utilisateur voit en premier. Si c'est cassé, tout le reste est inutile.

1.1 — Corriger auth_ready après Device Code Flow

Fichier : setup_service.py
Problème : après succès du flow, session.auth_ready n'est pas positionné → le frontend peut boucler sur /setup
Action : dans le callback de succès du background thread, écrire session.auth_ready = True et persister la session
1.2 — Supprimer la ressaisie du gamertag

Fichier : SetupPage.tsx
Problème : après status=provisioned, le backend connaît déjà le gamertag+xuid (résolu dans _DeviceFlowAttempt) mais le frontend demande quand même une saisie manuelle
Action :
Côté API : inclure gamertag et xuid dans DeviceFlowStatusResponse quand status=provisioned
Côté frontend : pré-remplir le champ, proposer une confirmation au lieu d'une saisie
1.3 — Enrichir le bootstrap avec setup_state

Fichier : bootstrap.py + bootstrap_service.py
Action : ajouter un champ setup_state avec les valeurs du plan :
setup_state: Literal["no_halo_link", "halo_linked_no_profile", "profile_ready_no_sync", "ready"]
Logique : dériver l'état depuis auth_state + présence de profils + présence de matchs sync
Impact frontend : le React shell route vers le bon écran en un seul test
Phase 2 — Politique de provisioning backend
2.1 — Ajouter can_self_provision aux capabilities

Fichier : common.py (CapabilityMap)
Action : ajouter le flag, piloté par un setting dans app_settings.json
can_self_provision: bool  # default True pour single-user, False pour instance partagée
2.2 — Guard backend sur POST /setup/players

Fichier : route setup dans apps/api/app/routes/
Action : vérifier can_self_provision côté serveur avant de créer le profil → 403 si interdit
Règle : le frontend masque le bouton, mais le backend refuse indépendamment
2.3 — Auto-provision depuis l'identité Halo

Action : quand le Device Code Flow résout le gamertag, proposer directement la création avec les infos pré-remplies (pas de champ texte vide)
Phase 3 — Sync initiale comme vrai job
3.1 — Persister le JobStore

Fichier : job_store.py
Problème actuel : dict en mémoire, perdu au restart
Action : sauvegarder les jobs dans un fichier JSON ou une table DuckDB (data/warehouse/jobs.json ou table job_history dans metadata.duckdb)
Au démarrage : recharger les jobs non-terminés
3.2 — Créer l'endpoint POST /api/v1/sync/initial

Action : nouvelle route qui :
Crée un job initial_sync dans le JobStore
Lance un background thread qui appelle l'orchestrateur sync existant (sync.py ou équivalent via import Python)
Met à jour le job avec des compteurs métier
3.3 — Enrichir le modèle de job pour la sync

Fichier : job_store.py schemas
Ajouter au modèle Job :
phase_key: str | None          # "fetch_matches", "enrich", "finalize"
phase_label: str | None        # "Récupération de vos matchs"
matches_done: int | None
matches_total: int | None
eta_seconds: int | None
warnings: list[str]
3.4 — Écran de progression frontend

Fichier : nouveau composant dans setup
Action : après création du profil → lancer la sync → afficher la progression avec les compteurs métier (réutiliser useJobStatus existant avec poll 3s)
Phase 4 — Durcissement sécurité
4.1 — Vérifier les cookies en production

Fichier : auth.py
Checklist :
 httpOnly=True ✅ (déjà fait)
 Secure=True si HTTPS (vérifier la détection)
 SameSite=Lax minimum
 TTL explicite (7j — déjà fait)
4.2 — Rate limiting sur routes sensibles

Routes : POST /auth/device-flow/start, POST /setup/players, POST /sync/initial
Action : middleware simple (compteur par IP/session, 5 req/min sur ces routes)
4.3 — Logging des actions sensibles

Action : ajouter des logs structurés pour :
Démarrage/succès/échec Device Code Flow
Création profil joueur
Démarrage/fin sync
Aucun secret dans les logs (masquer tokens)
Ordre d'exécution recommandé
1.1  auth_ready après Device Code    ──┐
1.2  Supprimer ressaisie gamertag    ──┼── Sprint 1 (fondations)
1.3  Bootstrap setup_state           ──┘
         │
2.1  can_self_provision capability   ──┐
2.2  Guard backend POST /players     ──┼── Sprint 2 (sécurité provisioning)
2.3  Auto-provision Halo identity    ──┘
         │
3.1  Persister JobStore              ──┐
3.2  POST /sync/initial              ──┼── Sprint 3 (sync initiale)
3.3  Modèle job enrichi              ──│
3.4  Écran progression frontend      ──┘
         │
4.1  Cookies prod                    ──┐
4.2  Rate limiting                   ──┼── Sprint 4 (durcissement)
4.3  Logging structuré               ──┘
