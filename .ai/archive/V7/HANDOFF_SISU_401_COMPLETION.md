# HANDOFF — SISU device-flow : 401 à la complétion (auth bout-en-bout cassée)

> Rédigé le 2026-07-15 par le superviseur (Fable) après diagnostic. Destiné à un agent
> disposant de contexte auth/SISU. Objectif : faire fonctionner le SSO Xbox natif (SISU)
> bout-en-bout pour la PREMIÈRE fois, puis retirer MSAL.

## Symptôme (reproduit par l'utilisateur, local, `LEVELUP_AUTH_MODE=xbox`)

Après avoir saisi le user_code sur `microsoft.com/link` et **validé le login Microsoft**,
l'UI affiche « Échec de l'authentification. ». Log serveur :

```
17:19:08 [ERROR] sisu: échec complétion status=401
```

Le 401 est renvoyé par le serveur Xbox à l'appel `CompleteSISUFlow`
(`apps/go-api/internal/platform/auth/sisu_client.go:222`).

## Contexte établi (à ne pas re-débattre)

1. **SISU device-flow n'a JAMAIS fonctionné bout-en-bout.** Depuis son introduction, il
   était cassé (URL device-code 404). L'utilisateur se connectait via le contournement
   `auth_provider=msal`. Les chantiers récents (voir commits ci-dessous) l'ont amené
   jusqu'à la complétion, qui échoue. Ce n'est donc PAS une régression à annuler : c'est
   une mise au point d'un flux Xbox non documenté, à faire aboutir.
2. **DÉCISION PRODUIT (utilisateur, 2026-07-15)** : un seul provider de tokens Halo =
   **SISU**. MSAL sera **supprimé** une fois SISU validé. `auth_mode=xbox` (comment
   l'utilisateur se connecte) est un axe distinct et reste inchangé.
3. **MSAL reste un filet TEMPORAIRE** le temps du diagnostic (l'utilisateur peut remettre
   `auth_provider=msal` en local pour travailler). Sa suppression = tout dernier commit,
   après confirmation SISU par l'utilisateur.
4. **TU NE PEUX PAS VALIDER SEUL** : la complétion exige un login Microsoft réel (identité
   de l'utilisateur). Méthode obligatoire = instrumenter → l'utilisateur fait un essai →
   diagnostiquer sur ses logs → itérer.

## Branche & état

- Branche de travail : **`fix/revue-adversariale`** (tête d'un train de merge GELÉ — ne
  rien merger/pousser vers main). Le working tree contient une suppression NON-STAGÉE de
  `.ai/PLAN_WEAPON_ATTRIBUTION_V3.md` (voulue par l'utilisateur) — NE PAS y toucher.
- Serveur dev local : air :8000 (mode xbox) + vite :5173. Redémarrage : PATH msys64 (gcc)
  prépendu au Start-Process, sinon air sert un binaire périmé en silence. Relancer à la fin.

## Fichiers clés (vérifiés sur pièces le 2026-07-15)

- `internal/platform/auth/sisu_client.go`
  - `InitSISUSession` (~l.100) : crée la session SISU (renvoie `SessionID`).
  - `CompleteSISUFlow` (l.156) → `completeSISUFlowWithURL` (l.167) : POST vers
    `sisuAuthorizeURL`. **Corps** (l.178) : `AppId`, `DeviceToken`, `ProofKey` (PoP),
    `Sandbox`, `AccessToken: "t=" + accessToken`, `UseModernGamertag`, `SiteName`,
    `SessionId`. Signature PoP via `kp.SignRequest(targetURL, "", body)` (l.194). 401 loggé
    l.222 — **ne logge QUE le status, pas le corps `raw`** (premier point d'instrumentation).
- `internal/platform/auth/sisu_provider.go`
  - `InitDeviceFlow` : construit le `sisuFlowContext{kp, deviceToken, sessionID,
    codeVerifier}` et le porte DANS l'objet `sisuDeviceFlow` (champ `flowCtx`).
  - `sisuDeviceFlow.ExchangeFlow` → `completeSISUExchange(ctx, flowCtx, accessToken)` →
    `CompleteSISUFlow(...)`. `Exchange` (provider) est désormais TOUJOURS stateless.
- `internal/api/handlers/auth.go`
  - `exchangeAfterAcquire` (l.264) : si le flow implémente `FlowExchanger` → `ExchangeFlow`
    (chemin SISU interactif), sinon `provider.Exchange`. Routage vérifié CORRECT.
  - `pollDeviceFlow` (~l.276) : `flow.AcquireToken(ctx)` (polling device-code Xbox) puis
    `exchangeAfterAcquire`.
  - `waitDeviceFlowReady` (l.181) + single-flight **clé = SessionID** (commentaire l.123) :
    dédoublonne les `start` concurrents (double-fire React dev).

## Commits de la file ayant touché ce flux (pour bisect logique)

- `a94fa3269` (lot ops) — URL device-code corrigée `oauth20_connect/device` →
  `oauth20_connect.srf` ; fix race single-flight (user_code vide → spinner) ; log de
  l'erreur InitDeviceFlow.
- `ba37cbe56` (lot ops) — inversion `verification_uri` vs `MsaOauthRedirect` (le lien
  affiché pointe la page de SAISIE du code) + raccourcissement d'URL.
- `73797cd63` (revue adversariale) — **slot global `SISUProvider.current` supprimé** au
  profit d'un contexte per-flow (`sisuDeviceFlow.flowCtx`). Vérifie que l'objet `flow`
  utilisé par `pollDeviceFlow` est bien CELUI dont le `flowCtx` correspond au user_code
  affiché à l'utilisateur (interaction avec le single-flight clé=SessionID).

## Hypothèses techniques du 401 (à instrumenter/vérifier, non ordonnées)

1. **Audience/format de l'access_token** : `AccessToken: "t=" + accessToken`. L'access_token
   obtenu par le device-flow Xbox (`AcquireToken`) doit avoir l'audience attendue par SISU
   authorize (`user.auth.xboxlive.com` / scope `service::user.auth.xboxlive.com::MBI_SSL`).
   Si le fix d'URL a changé le scope/l'audience, la complétion rejette en 401. Le préfixe
   `"t="` est peut-être à revoir selon le type de token.
2. **Cohérence session/device** : `SessionId` + `DeviceToken` + `ProofKey` doivent provenir
   du MÊME `InitSISUSession`/`InitDeviceFlow` que le user_code validé. Vérifier qu'un
   double-init (single-flight) ne fait pas compléter avec un `flowCtx` d'un autre flow.
3. **Signature PoP** : `kp.SignRequest(targetURL, "", body)` — vérifier URL signée, corps
   exact, timestamp, format de la clé (cross-référencer une implémentation SISU connue).
4. **Corps/headers** : `x-xbl-contract-version`, `SiteName`, `Sandbox` — comparer à un
   wrapper de référence.

## Méthode imposée

1. **Cross-référencer un wrapper SISU connu-bon** AVANT d'accuser les tokens (mémoire projet
   « Cross-référencer Grunt+SPNKr avant l'auth » ; voir aussi `internal/platform/auth/halo_exchange.go`
   qui référence Grunt/SPNKr). Le flux SISU natif est documenté dans OpenSpartan/Grunt.
2. **Instrumenter d'abord** : logger le CORPS `raw` de la réponse 401 (le serveur Xbox y met
   souvent un code d'erreur précis : `XErr`, message), et tracer chaque étape de la chaîne
   avec les valeurs NON-SENSIBLES (longueurs, présence, sessionID, audience du token décodé
   — JAMAIS le token brut ni de secret). Committer cette instrumentation.
3. **Demander un essai à l'utilisateur** (il refait login réel) → récupérer les logs →
   diagnostiquer → corriger → itérer jusqu'au « sisu: ExchangeFlow OK » + session admin.
4. Quand SISU marche bout-en-bout (confirmé utilisateur) : **retirer MSAL** (MSALProvider +
   son câblage `buildTokenProvider`, msal_client si plus référencé) avec ses tests/imports,
   MAJ doc onboarding (`docs/INSTALL.md` + FR, `docs/CONFIGURATION.md`) : SISU = seul
   provider. Vérifier que le pool auto-sync / refresh (Exchange stateless) reste couvert par
   SISU (son `Exchange` stateless = `ExchangeAccessToken`, équivalent MSAL — à confirmer).

## Contraintes projet (non négociables)

- ADR 0023 : tokens source unique `data/auth/watcher_tokens/{xuid}.json` ; JAMAIS de
  re-capture pour « réparer » une auth ; `AADSTS70000/90023` ≠ raison de re-capturer.
- Aucun secret (token, webhook, access_token brut) dans les logs.
- Builds/tests Go SÉQUENTIELS (cache Windows corruptible) ; `-tags=integration -p 1` si
  persist/sync touchés ; `golangci-lint --new-from-rev=origin/main` = 0 ; front check-types
  purgé + eslint + vitest si touché ; gofmt avant commit (hook rejette sinon).
- Commits FR + `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` ; thought_log EN
  HAUT avant chaque commit ; pas d'emojis ; jamais de push/merge vers main ; pas de PR
  (le train sera réassemblé par le superviseur APRÈS validation SISU).

## Livrable

SISU device-flow fonctionnel bout-en-bout (login réel → session admin), prouvé par un
essai utilisateur ; MSAL retiré ; gates verts ; commits sur `fix/revue-adversariale`.
Rapport final : cause racine du 401, correctif, ce que l'utilisateur a validé, état du
retrait MSAL, points restants.
