# Plan — Durcissement Auth & Consolidation Azure (optionnel, post-incident SSO)

> Contexte : après la soirée du 2026-06-12/13 (connexion Xbox réparée : routes OAuth
> racine, race d'hydratation login, identité XSTS xboxlive.com, anti-clobber du cookie
> d'auth + nettoyage sessions). Chantiers OPTIONNELS restants, **numérotés dans l'ordre
> d'exécution réel** (la Phase 1 — seam — est volontairement faite avant l'audit car
> elle ne change aucun comportement et matérialise l'inventaire code/env).

## État des lieux (acquis)

- SSO Xbox fonctionnel (Authorization Code Flow), device flow OK.
- 4 joueurs : `jgtm`/`jgtm_xbox` (admin), `madina97294`, `xxdaemongamerxx` (user).
- `instance_locked=false`. Sessions 7 j glissants + persistance conditionnelle + purge 6 h.
- App canonique cible décidée par l'user : **`e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca`** (LevelUp Halo).

---

## Phase 1 — Seam unique pour les credentials Azure — ✅ FAIT (commit `7071aef86`, non déployé)

Refactor SANS changement de comportement : `internal/platform/auth/azure_credentials.go`
= source unique des lectures `SPNKR_AZURE_CLIENT_ID`/`SECRET`. 3 usages explicites
(`ResolveAzureOAuthClient` SSO+refresh, `DeviceFlowClientID`, `TokenCaptureClientID`).
9 golden tests figent le comportement actuel (filet avant/après). Sentinelle = seam
seul lecteur prod du secret.
- **Reste pour clore la phase** : déployer + vérif live (SSO + device + refresh watcher)
  que rien n'a bougé (les unit tests ne prouvent pas l'acceptation Microsoft/XSTS).

## Phase 2 — Audit Azure (gate de la Phase 3, non destructif)

But : savoir **où** opérer sans casser. Les refresh tokens sont liés à leur app émettrice.

- **2.1 Inventaire portail Azure** (toi seul) : pour chaque app → client_id, plateforme
  (Web/Mobile/SPA), redirect URIs, permission `Xboxlive.signin`, secrets actifs/expirés,
  public-client activé.
- **2.2 Inventaire code/env** : ✅ déjà matérialisé par le seam (Phase 1) — voir
  `azure_credentials.go`.
- **2.3 Couplage tokens (read-only)** : déterminer sous quelle app les RT des 4 joueurs
  ont été émis (le cache MSAL dans `watcher_tokens/{xuid}.json` porte le client_id en
  clair). Conclusion → coût de bascule (re-login des joueurs).
- Sortie : `.ai/AUDIT_AZURE_APPS.md` + décision Go/No-Go.

## Phase 3 — Uniformiser vers `e1cb35ab` + supprimer les apps mortes (changement EXPLICITE)

- **3.1** Introduire `LEVELUP_OAUTH_CLIENT_ID` (défaut = `e1cb35ab`) consommé par le seam ;
  aligner SSO/refresh ET token-capture sur l'app canonique. Le **diff des golden tests
  = le changement assumé** (« après »). Gérer public/confidentiel (retry AADSTS90023).
- **Pré-requis portail** : redirect `https://lvelup.info/auth/xbox/callback` (racine) en
  plateforme **Web** sur `e1cb35ab`.
- **Impact tokens** : les RT actuels (émis sous l'ancienne app) deviennent invalides →
  chaque joueur re-login une fois (cheap). Vérif watcher OK après re-login.
- **3.2** Supprimer les apps mortes (portail) APRÈS 3.1 stable + filet (garder les
  secrets retirés hors-ligne 1–2 semaines).

## Phase 4 — Durcissement sécurité OAuth (indépendant, peut se faire dès maintenant)

- **PKCE** (S256) : `code_challenge` à l'authorize + `code_verifier` (stocké en session
  à côté de `OAuthState`) au token exchange. Parade si le `code` fuit (il apparaît dans
  l'URL de callback).
- **Logs nginx** : exclure la query de `/auth/xbox/` de l'`access_log`.
- Tests : flux SSO e2e avec PKCE.

## Phase 5 — Verrouillage d'instance

- Pré-requis : tous les joueurs légitimes présents dans `users.json`.
- `LEVELUP_INSTANCE_LOCKED=true` → xuid inconnu refusé au login. Rollback = false.

## Phase 6 — Fusion des comptes JGtm (faible priorité)

- `jgtm` (admin, mot de passe) + `jgtm_xbox` (admin, SSO) = doublon bénin.
- Recommandé : **garder les deux** (le mot de passe = secours admin si SSO casse).
- Sinon fusionner (transférer le `password_hash` sur le compte xuid, supprimer le doublon).

## Phase 7 — Vérification fonctionnelle finale (à faire en dernier)

- **Bouton de déconnexion** : cliquer « Se déconnecter » → la session est bien invalidée
  (POST /auth/logout), retour `/login`, et la revisite ne reconnecte PAS automatiquement
  (cookie d'auth bien effacé). Important vu tous les changements session de cette soirée.
- **Admin JGtm** : `jgtm_xbox` (admin) voit bien la section/onglet admin et y accède
  (users, invites, monitoring) ; un compte `user` (madina/xxdaemon) ne la voit pas.

---

### Limite de test à assumer (honnêteté)

Les unit tests couvrent le **choix du client** et la **forme des requêtes** (déterministe),
PAS l'acceptation Microsoft/XSTS (nécessite un vrai compte). → **checklist de vérif live
obligatoire à chaque déploiement** (SSO e2e + device + refresh watcher) en complément.

Chaque phase : 1 branche, build+tests verts, déploiement, vérif live, entrée thought_log.
Les opérations portail Azure / `.env.local` VPS sont manuelles ou via SSH avec préavis.
