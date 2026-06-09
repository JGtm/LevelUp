# Plan — Chiffrement at-rest des watcher tokens

> **Créé le** : 2026-06-09
> **Statut** : Proposé — **conditionnel** (déclencheur produit requis avant implémentation)
> **Priorité** : 🟢 Basse aujourd'hui (outil single-user local, threat model accepté par ADR 0023)
> **Origine backlog** : `[auth/security] Chiffrement at-rest des watcher tokens`

## Contexte & état vérifié

Source unique des tokens (ADR 0023) : `data/auth/watcher_tokens/{xuid}.json` via
`*auth.MultiUserTokenStore`. Vérifié dans le code :
- Écriture **JSON en clair**, atomique, perms **0600** fichier / **0700** répertoire
  (`multi_user_token_store.go:431,453,456`).
- **Aucun** chiffrement : `grep Encrypt|cipher|aes|go-keyring|DPAPI` dans `internal/platform/auth/` → 0.
- Champs sensibles en clair : `oauth_refresh_token`, `msal_cache_json`, `xsts_token`, `access_token`.

C'est une **décision design explicite** d'ADR 0023 (sécurité = permissions FS, pas de crypto at-rest),
**pas un oubli**. Ce plan n'est donc à exécuter **que** si le déclencheur ci-dessous se réalise.

## Déclencheur (condition d'activation)

Implémenter **uniquement si** l'un survient :
- 🔴 Distribution publique grand public (users non-techniciens qui ne protègent pas leur HOME), ou
- 🔴 Incident de fuite de token, ou
- 🔴 Bascule vers un déploiement multi-tenant / cloud.

Tant que l'outil reste single-user local : **ne pas implémenter** (complexité + risque de régression
auth sans bénéfice réel ; les tokens sont TTL court et révocables côté Microsoft).

## Conception (le jour où le déclencheur tombe)

### Principe : couche de chiffrement transparente avec fallback clair
Nouveau sous-package `internal/platform/auth/securestore/` qui **wrappe** Read/Write/Delete du store
existant, sans changer son interface publique. Le `MultiUserTokenStore` délègue le (dé)chiffrement à
cette couche.

### Backend natif OS via `github.com/zalando/go-keyring` (ou équivalent)
- **Windows** : DPAPI (`CryptProtectData`) — cible principale (cf. environnement projet).
- **macOS** : Keychain.
- **Linux** : libsecret / GNOME Keyring.

Modèle recommandé : **clé de chiffrement** stockée dans le keychain OS, **données** chiffrées
(AES-GCM) dans le fichier `{xuid}.json` existant. Évite de mettre des blobs volumineux (MSAL cache)
dans le keychain ; ne stocke que la clé.

### Fallback & migration
- **Fallback clair** : si aucun keychain dispo (CI headless, Linux minimal), retomber sur l'écriture
  claire actuelle **avec warn log** (`encryption_unavailable`) — ne jamais bloquer l'auth.
- **Migration douce** : au boot, détecter les fichiers en clair, les ré-écrire chiffrés si keychain
  dispo. Lecture tolérante des deux formats (marqueur de version/format dans le JSON).
- **Réversibilité** : commande/outil pour déchiffrer (export) en cas de besoin de debug.

### Garde-fous
- ⛔ Pas de logique métier dans le package auth (règle ADR 0023) — la couche securestore reste pure
  (pas de dépendance DuckDB).
- ✅ Invalidation cache process (`halo.InvalidateCachedPlayerTokens`) inchangée.
- ✅ Tests : round-trip chiffré, fallback clair sans keychain, lecture mixte clair/chiffré, migration
  boot idempotente.

## Estimation
~3h une fois le déclencheur acté (1 package securestore + intégration store + tests). Effort
concentré ; risque principal = casser l'auth → tests round-trip + fallback obligatoires avant merge.

## Références
- `internal/platform/auth/multi_user_token_store.go` (store actuel, perms 0600/0700)
- ADR 0023 — `docs/adr/0023-auth-tokens-single-source.md` (décision de ne PAS chiffrer + threat model)
- `.ai/V7/AUDIT_TOKEN_STORAGE.md`
