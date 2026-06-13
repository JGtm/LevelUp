# Audit Azure — App registrations LevelUp (Phase 2.1)

## ✅ FINDINGS (relevés portail 2026-06-13)

### App canonique — « LevelUp Halo » `e1cb35ab` (créée 03/06/2026)
- Object ID `4cfaf2b7-8651-45cc-969c-45e85b7d5682`.
- **Account types : Personal account only** ✅ (cohérent avec l'authority `/consumers`).
- **Allow public client flows : Enabled** → **client PUBLIC**.
- Redirects **Web** : `https://lvelup.info/api/v1/auth/xbox/callback` (⚠️ chemin **/api/v1**, PAS racine) + `https://localhost`. Plateforme Mobile/desktop : nativeclient.
- Permissions : `offline_access` + `User.Read` (PAS de `Xboxlive.signin` listé — non bloquant, le scope est accepté dynamiquement, le device flow le prouve).
- 1 secret (331ba791) — **non utilisé pour notre usage** (public + PKCE → on n'envoie pas de secret).

### App actuelle SSO/refresh — « Spartan Graph » `39829f7a` (créée 19/01/2026)
- Object ID `629c7679-…`, tenant `c3cf26f9-…`.
- **Account types : All Microsoft account users** (multi-tenant + perso).
- **Allow public client flows : Enabled** (public, mais possède des secrets).
- Redirects **Web** : `https://lvelup.info/auth/xbox/callback` (chemin **RACINE**) + `https://localhost`.
- **2 secrets** : `SPNKR` (6816804e, exp 19/01/2027, `kOI…`) + `SPNKR2` (516094d5, exp 19/01/2027, `aPk…`). L'un = `SPNKR_AZURE_CLIENT_SECRET` du VPS (len 41).
- Permissions : Microsoft Graph `User.Read`.

### Conséquences pour la bascule (Phase 3)
- **e1cb35ab est PRÊT** (c'est l'app du device flow, public, redirect Web déjà présent) — MAIS son redirect est `/api/v1/auth/xbox/callback`, alors que la prod utilise le chemin **racine** (enregistré sur 39829f7a).
- **Deux chemins pour basculer** :
  - **(A) Zéro changement Azure** : `LEVELUP_OAUTH_REDIRECT_URI=https://lvelup.info/api/v1/auth/xbox/callback` (e1cb35ab l'a déjà ; notre Go a gardé l'alias `/api/v1/auth/xbox/callback` ; nginx `location /` le proxifie). Marche tel quel.
  - **(B) Plus propre (1 action portail)** : ajouter le redirect **racine** `https://lvelup.info/auth/xbox/callback` à la plateforme Web de e1cb35ab → garde le chemin racine + le `access_log off` nginx s'applique.
- **e1cb35ab = public + PKCE (déployé)** → on n'envoie **AUCUN secret** (le seam retourne déjà `SecretToSend()=""` pour LevelUpClientID). Config moderne correcte.
- **Coût tokens** : les RT des 4 joueurs sont émis sous `39829f7a` → invalides après bascule → **re-login des 4 joueurs** (le watcher récupère après). Confirmé.
- token-capture : aligner aussi sur e1cb35ab (le seam `TokenCaptureClientID` défaut HaloTools → à pointer sur e1cb35ab en Phase 3).

### Reste à confirmer
- **Y a-t-il d'AUTRES apps** (le « mélange » de tests) que ces 2 ? Si oui, les lister (candidates suppression). Si non → la consolidation = juste basculer SSO/refresh/capture sur e1cb35ab, puis éventuellement supprimer `39829f7a` (« Spartan Graph ») quand tout est stable.

---



> But : cartographier TOUTES les app registrations Azure (le « mélange » créé pendant
> les tests) avant de consolider vers l'app canonique **`e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca`**
> (LevelUp Halo) et de supprimer les apps mortes. Document à REMPLIR en parcourant le portail.
>
> Portail : https://entra.microsoft.com (ou portal.azure.com → « Microsoft Entra ID » →
> « App registrations »). Onglet **All applications** ET **Owned applications**.

## Connu d'avance (à confirmer)

| Client ID | Nom probable | Rôle actuel | Public/Confidentiel |
|---|---|---|---|
| `e1cb35ab-c41a-4ee5-a7a1-22ea4e94cdca` | LevelUp Halo | **Cible canonique** ; device flow (code 9 car.) | Public (device flow) |
| `39829f7a-5262-4d22-a387-795c488f7102` | halo-tools | SSO web + refresh en prod (`SPNKR_AZURE_CLIENT_ID`) ; token-capture | Public (selon code) MAIS un secret est posé |

Côté VPS `.env.local` : `SPNKR_AZURE_CLIENT_ID=39829f7a…`, `SPNKR_AZURE_CLIENT_SECRET` (longueur 41). Il faut identifier QUEL secret du portail correspond.

---

## A. Inventaire global (lister TOUTES les apps)

Pour chaque app registration trouvée (y compris les oubliées/tests) :

- [ ] **Display name** : ______
- [ ] **Application (client) ID** : ______
- [ ] **Object ID** : ______
- [ ] **Créée le** / par qui (Owner) : ______
- [ ] **Supported account types** : (Personal Microsoft accounts only ? AAD+personal ? Single tenant ?) → pour Xbox Live il faut les **comptes Microsoft personnels** (consumers).
- [ ] Repérage : est-ce une **app de test à supprimer** ? (oui/non/incertain)

> Astuce : note tout ce qui ressemble à un doublon « halo / levelup / xbox / test … ».

---

## B. Pour CHAQUE app (détail) — surtout `e1cb35ab` et `39829f7a`

### B.1 Authentication
- [ ] **Plateformes** configurées : Web / SPA / Mobile & desktop / (aucune)
- [ ] **Redirect URIs** (exacts, par plateforme) : ______
  - Présence de `https://lvelup.info/auth/xbox/callback` ? sur quelle plateforme ? (doit être **Web**)
  - Présence de `http://localhost:8000/...` (dev) ? autres ?
- [ ] **Allow public client flows** : Yes / No  → détermine public (Yes) vs confidentiel (No)
- [ ] **Front-channel logout URL** : ______ (souvent vide)
- [ ] **Implicit grant / tokens** (access tokens, ID tokens cochés ?) : ______

### B.2 Certificates & secrets
- [ ] **Client secrets** : nombre, et pour chacun → Description, **Expiration**, Statut (actif/expiré), **Secret ID** (les 2 premiers caractères de la valeur si visibles).
- [ ] Lequel correspond à `SPNKR_AZURE_CLIENT_SECRET` du VPS (longueur 41) ?
- [ ] **Certificates** : présents ? (normalement non)
- [ ] Secrets **expirés** à nettoyer.

### B.3 API permissions
- [ ] Présence de **`Xboxlive.signin`** (type *Delegated*) ?
- [ ] Statut : *Granted* (consenti) ou non ?
- [ ] Permissions **superflues** (Graph User.Read par défaut, etc.) à noter.

### B.4 Autres
- [ ] **Owners** : qui a accès à l'app.
- [ ] **Branding & properties** : nom d'éditeur, logo (aide à identifier les apps de test).
- [ ] **Expose an API** / **App roles** : (normalement vide pour nous) — noter si configuré.

---

## C. Spécifique à la bascule Phase 3 (cible `e1cb35ab`)

Vérifier précisément sur **`e1cb35ab` (LevelUp Halo)** :
- [ ] A-t-il la plateforme **Web** avec le redirect **exact** `https://lvelup.info/auth/xbox/callback` ? → **SI NON : c'est l'action n°1** (l'ajouter) avant de basculer le SSO web dessus, sinon `AADSTS50011`.
- [ ] Est-il **public** (Allow public client flows = Yes) ou **confidentiel** (a un secret) ? → conditionne l'envoi ou non d'un `client_secret` (garde anti-AADSTS90023 dans le code).
- [ ] A-t-il **`Xboxlive.signin`** (delegated) + consenti ?
- [ ] Account types = **comptes Microsoft personnels** (consumers) ?

Et sur **`39829f7a` (halo-tools)** :
- [ ] Lister ses redirect URIs (pour savoir tout ce qui pointe dessus aujourd'hui).
- [ ] Confirmer son/ses secret(s) (dont celui en prod).

---

## D. Corrélation lvelup.info (je peux aider côté VPS)

- [ ] `SPNKR_AZURE_CLIENT_ID` (VPS) = `39829f7a…` → confirmé.
- [ ] `SPNKR_AZURE_CLIENT_SECRET` (VPS, len 41) ↔ quel secret du portail ?
- [ ] Tokens du parc (`data/auth/watcher_tokens/{xuid}.json`) : émis sous l'app SSO prod (39829f7a) → re-login requis après bascule vers e1cb35ab (confirmé au switch : 1er refresh échoue → re-login).

---

## E. Décision finale (à remplir après l'inventaire)

- [ ] **App canonique** retenue : `e1cb35ab` (confirmé) — prête ? (Web redirect + Xboxlive.signin + account type OK)
- [ ] **Apps à SUPPRIMER** (liste des client_id morts/tests) : ______
- [ ] **Secrets à révoquer/rotater** : ______
- [ ] **Pré-requis avant bascule** : ajouter le redirect Web sur e1cb35ab ; décider public vs confidentiel.

> ⚠️ Ne RIEN supprimer avant que la bascule (Phase 3) soit déployée et vérifiée en prod,
> et que les tokens du parc soient confirmés non liés aux apps supprimées (ou re-capturés).
