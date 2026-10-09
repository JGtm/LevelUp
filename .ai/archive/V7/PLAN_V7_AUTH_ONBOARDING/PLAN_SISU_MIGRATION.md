# Plan de migration MSAL → SISU/PoP (RTA + auth Xbox native)

> Référence externe : [dend/conch](https://github.com/dend/conch) — MIT — TypeScript/C#  
> Branche cible : à créer depuis `main` — `feat/sisu-auth-provider`  
> Statut : **Planifié** — non démarré

---

## Contexte

Le flow d'authentification actuel repose sur MSAL (Microsoft Authentication Library for Go)
pour obtenir l'`access_token` Microsoft, lequel alimente ensuite deux chaînes :

```
MSAL Device Code Flow → access_token
  ├─ → UserToken → XSTS (Halo RP)      → Spartan + Clearance  [sync/API Halo]
  └─ → UserToken → XSTS (XboxLive RP)  → header RTA           [watcher présence]
```

**Problèmes MSAL actuels :**
- Dépendance à une Azure App Registration (`LevelUpClientID`) — révocable par Microsoft
- Cache sérialisé MSAL stocké en DuckDB (`sync_meta`) — format opaque, fragile
- UX onboarding en deux étapes : l'utilisateur doit copier un code court sur `microsoft.com/devicelogin`

**Ce que SISU apporte :**
- Flow 100% Xbox-natif (endpoints `sisu.xboxlive.com`) — aucune Azure app
- UX plus simple : un lien OAuth direct à cliquer, pas de code à saisir
- Le XSTS est retourné directement par SISU (`AuthorizationToken`) — une étape de moins
- Nécessite un Device Token signé (Proof-of-Possession ECDSA P-256) — portage en Go stdlib

---

## Stratégie : coexistence avec switch admin

`AppSettings.auth_provider` (`"msal"` | `"sisu"`) pilote l'instanciation du `TokenProvider`
dans `cmd/server/main.go`. Valeur par défaut : `"msal"` → **aucune régression** pour les
installations existantes. SISU est activable explicitement par un admin.

### Roadmap

```
Phase 1  Primitives PoP Signing              (pop_signing.go)
Phase 2  Device Token                        (device_token.go)
Phase 3  SISU Client                         (sisu_client.go)
Phase 4  Device Code Flow natif Xbox         (xbox_device_code.go)
Phase 5  SISUProvider + switch AppSettings   (provider.go + settings/store.go)
Phase 6  Adaptation frontend                 (Settings UI + page onboarding)
Phase 7  Tests en conditions réelles
Phase 8  Basculer le défaut sur "sisu"
Phase 9  Déprécier puis supprimer MSAL
```

---

## Détail des phases

### Phase 1 — Primitives PoP Signing

**Fichier** : `apps/go-api/internal/platform/auth/pop_signing.go`

Portage de `pop-crypto-provider.ts` (Conch) + `xbox-authentication-client.ts#signRequest` en Go stdlib.

#### Structures

```go
type PoPKeyPair struct {
    privateKey *ecdsa.PrivateKey
    proofKey   ProofKey
}

// ProofKey est le JWK public à inclure dans les requêtes Device/SISU.
type ProofKey struct {
    Kty string `json:"kty"` // "EC"
    Alg string `json:"alg"` // "ES256"
    Crv string `json:"crv"` // "P-256"
    Use string `json:"use"` // "sig"
    X   string `json:"x"`   // base64url coordonnée X
    Y   string `json:"y"`   // base64url coordonnée Y
}
```

#### Fonctions

```go
// GeneratePoPKeyPair génère une paire de clés ECDSA P-256 éphémère.
func GeneratePoPKeyPair() (*PoPKeyPair, error)

// ProofKey retourne le JWK public (à inclure dans le body des requêtes Device/SISU).
func (kp *PoPKeyPair) GetProofKey() ProofKey

// SignRequest construit et signe le header "Signature" pour les endpoints Xbox PoP.
// uri     : URL complète de la requête (ex: "https://device.auth.xboxlive.com/...")
// authToken : "" pour Device/SISU (non authentifié à ce stade)
// body    : corps JSON de la requête
// Retourne : header Signature encodé en Base64 standard.
func (kp *PoPKeyPair) SignRequest(uri, authToken, body string) (string, error)
```

#### Algorithme de signature (fidèle à Conch)

```
payload = [4 octets policy_version=1 BE] + [0x00]
        + [8 octets Windows FILETIME BE]  + [0x00]
        + UTF-8("POST\0" + pathAndQuery + "\0" + authToken + "\0" + body + "\0")

Windows FILETIME = (unixSeconds + 11_644_473_600) × 10_000_000

sig_der  = ecdsa.Sign(rand, privateKey, sha256(payload))
sig_p1363 = convertDER_to_P1363(sig_der)  // r||s, 32+32 octets

header_bytes = [4B policy_version BE] + [8B timestamp BE] + sig_p1363
Signature    = base64.StdEncoding.EncodeToString(header_bytes)
```

> **Point de vigilance** : Go génère des signatures ECDSA au format ASN.1 DER.
> Conch attend du IEEE P1363 (r‖s, 64 octets pour P-256). La conversion DER→P1363
> est non-triviale et doit être testée unitairement en premier.

#### Tests unitaires requis

- Vecteur fixe : clé connue + payload → signature attendue (vecteur tiré de Conch)
- `GeneratePoPKeyPair` : x, y sont des base64url valides
- `SignRequest` : output est du Base64 standard valide, longueur = ceil((4+8+64)/3)*4

---

### Phase 2 — Device Token

**Fichier** : `apps/go-api/internal/platform/auth/device_token.go`

```go
// RequestDeviceToken obtient un Device Token Xbox avec signature PoP.
func RequestDeviceToken(ctx context.Context, client *http.Client, kp *PoPKeyPair) (string, error)
```

**Endpoint** : `POST https://device.auth.xboxlive.com/device/authenticate`

**Headers** :
```
Content-Type: application/json
Accept: application/json
x-xbl-contract-version: 2
Signature: <kp.SignRequest(url, "", body)>
```

**Body** :
```json
{
  "RelyingParty": "http://auth.xboxlive.com",
  "TokenType": "JWT",
  "Properties": {
    "AuthMethod": "ProofOfPossession",
    "DeviceType": "Win32",
    "Id": "{<UUID-MAJUSCULE>}",
    "ProofKey": { "kty": "EC", "alg": "ES256", "crv": "P-256", "use": "sig", "x": "...", "y": "..." },
    "Version": "10.0.22000"
  }
}
```

Retourne `resp["Token"].(string)`.

---

### Phase 3 — SISU Client

**Fichier** : `apps/go-api/internal/platform/auth/sisu_client.go`

#### Étape 3a — Initialiser la session SISU

```go
type SISUSession struct {
    SessionID       string // header "X-SessionId" de la réponse
    MsaOauthRedirect string // URL à présenter à l'utilisateur
}

// InitSISUSession ouvre une session SISU et retourne l'URL OAuth à afficher.
func InitSISUSession(
    ctx context.Context,
    client *http.Client,
    kp *PoPKeyPair,
    deviceToken string,
    appID, titleID string,           // ex: "000000004c20a908", "144209987"
    codeChallenge, codeChallengeState string,
) (*SISUSession, error)
```

**Endpoint** : `POST https://sisu.xboxlive.com/authenticate`

**Headers** :
```
Content-Type: application/json
Accept: application/json
x-xbl-contract-version: 2
Signature: <kp.SignRequest(url, "", body)>
```

**Body** :
```json
{
  "AppId": "<appID>",
  "TitleId": "<titleID>",
  "DeviceToken": "<deviceToken>",
  "Offers": ["service::user.auth.xboxlive.com::MBI_SSL"],
  "ProofKey": { ... },
  "RedirectUri": "https://login.live.com/oauth20_desktop.srf",
  "Sandbox": "RETAIL",
  "TokenType": "code",
  "Query": {
    "display": "touch",
    "code_challenge": "<codeChallenge>",
    "code_challenge_method": "S256",
    "state": "<state>"
  }
}
```

`SessionID` ← header `X-SessionId` de la réponse.

#### Étape 3b — Compléter le flow SISU

```go
// CompleteSISUFlow échange l'access_token OAuth contre le XSTS directement (via SISU).
// Retourne un XSTSResult prêt à l'emploi (même struct qu'AcquireXSTSForRTA).
func CompleteSISUFlow(
    ctx context.Context,
    client *http.Client,
    kp *PoPKeyPair,
    deviceToken, accessToken string,
    appID, sessionID string,
    codeVerifier string,
) (*XSTSResult, error)
```

**Endpoint** : `POST https://sisu.xboxlive.com/authorize`

**Headers** :
```
Content-Type: application/json
Accept: application/json
x-xbl-contract-version: 2
Signature: <kp.SignRequest(url, "", body)>
```

**Body** :
```json
{
  "AppId": "<appID>",
  "DeviceToken": "<deviceToken>",
  "ProofKey": { ... },
  "Sandbox": "RETAIL",
  "AccessToken": "t=<accessToken>",
  "UseModernGamertag": true,
  "SiteName": "user.auth.xboxlive.com",
  "SessionId": "<sessionID>"
}
```

**Réponse** : `AuthorizationToken` = XSTS direct (champs `Token`, `DisplayClaims.xui[0]`).
Extraire avec les fonctions existantes `extractUserHash`, `extractDisplayClaims`, `extractNotAfter`.

> **Note** : SISU retourne le XSTS directement dans `AuthorizationToken` — pas besoin de
> l'étape User Token → XSTS séparée (contrairement au flow `halo_exchange.go`).

---

### Phase 4 — Device Code Flow natif Xbox (sans MSAL)

**Fichier** : `apps/go-api/internal/platform/auth/xbox_device_code.go`

RFC 8628 pur sur `login.live.com`, sans dépendance externe.

```go
// XboxDeviceCodeResult contient les données du Device Code Flow Xbox.
type XboxDeviceCodeResult struct {
    UserCode        string
    VerificationURL string // "https://login.live.com/oauth20_remoteconnect.srf"
    DeviceCode      string // opaque, pour le polling
    ExpiresIn       int
    Interval        int    // délai de polling en secondes
}

// StartXboxDeviceCode initie un Device Code Flow sur login.live.com.
func StartXboxDeviceCode(ctx context.Context, clientID string) (*XboxDeviceCodeResult, error)
    // POST https://login.live.com/oauth20_connect/device
    // Form: client_id, scope="Xboxlive.signin Xboxlive.offline_access", response_type=device_code

// PollXboxDeviceCode attend la complétion du Device Code Flow.
// Bloquant — à appeler dans une goroutine.
func PollXboxDeviceCode(ctx context.Context, clientID, deviceCode string, interval int) (accessToken, refreshToken string, err error)
    // Boucle POST https://login.live.com/oauth20_token.srf
    // Form: grant_type=urn:ietf:params:oauth:grant-type:device_code, client_id, device_code
    // Gère: authorization_pending (continuer), slow_down (augmenter interval), success, erreur fatale
```

**`clientID`** : appID Xbox officiel (ex. `000000004c20a908` — Halo Waypoint mobile).
À confirmer en testant le flow réel contre `login.live.com`.

---

### Phase 5 — SISUProvider + switch AppSettings

#### 5a — Nouveau provider (`provider.go`)

> **Problème de couplage à résoudre ici** :
>
> `pollDeviceFlow` (Halo auth) suit le chemin :
> ```
> flow.AcquireToken() → access_token → provider.Exchange(accessToken) → ExchangeResult
> ```
> `pollWatcherAuth` (RTA auth) suit le chemin :
> ```
> flow.AcquireToken() → access_token → AcquireXSTSForRTA(accessToken) → XSTSResult
> ```
> Pour MSAL, `Exchange` est stateless (access_token → XBL → XSTS → Spartan) — OK.
> Pour SISU, `CompleteSISUFlow` a besoin du contexte de session (`kp`, `deviceToken`,
> `sessionID`, `codeVerifier`) initialisé dans `InitDeviceFlow`. Ces données ne transitent
> pas dans l'access_token.
>
> **Fix** : `SISUProvider` stocke le contexte de flow en tant qu'état éphémère (protégé
> par mutex) : posé par `InitDeviceFlow`, consommé et effacé par `Exchange`. L'accès
> concurrent est peu probable (single-flight via `AttemptStore`) mais le mutex garantit
> la safety. `pollDeviceFlow` et `pollWatcherAuth` ne changent **pas** de signature —
> le couplage est transparent pour les appelants.

```go
// sisuFlowContext regroupe le contexte éphémère d'un SISU flow en cours.
// Durée de vie : de InitDeviceFlow() à Exchange() (puis effacé).
type sisuFlowContext struct {
    kp           *PoPKeyPair
    deviceToken  string
    sessionID    string
    codeVerifier string
}

// SISUProvider implémente TokenProvider via SISU/PoP (sans MSAL).
type SISUProvider struct {
    appID   string
    titleID string

    mu      sync.Mutex        // protège current
    current *sisuFlowContext  // nil entre deux flows
}

func NewSISUProvider(appID, titleID string) *SISUProvider

// InitDeviceFlow démarre un Device Code Flow Xbox natif + initialise la session SISU.
// Stocke le sisuFlowContext dans p.current pour que Exchange puisse l'utiliser.
// Retourne un sisuDeviceFlow (privé) qui implémente DeviceFlow.
// GetVerificationURL() = MsaOauthRedirect (URL OAuth directe, pas de code à saisir).
// GetFlowType() = "sisu".
func (p *SISUProvider) InitDeviceFlow(ctx context.Context) (DeviceFlow, error)

// Exchange complète le flow SISU après poll OAuth.
// Lit p.current (sessionID, kp, deviceToken, codeVerifier) — posé par InitDeviceFlow.
// Appelle CompleteSISUFlow(ctx, kp, deviceToken, accessToken, appID, sessionID, codeVerifier)
// → XSTSResult → requestSpartanToken → requestClearanceToken → ExchangeResult.
// Efface p.current après usage.
// ⚠️ Panics si appelé sans InitDeviceFlow préalable (bug d'utilisation, pas d'erreur runtime).
func (p *SISUProvider) Exchange(ctx context.Context, accessToken string) (*ExchangeResult, error)

// TrySilentRefresh délègue à TryOAuthRefresh (pas de cache MSAL).
func (p *SISUProvider) TrySilentRefresh(ctx context.Context, cacheJSON string) (string, error)

// TryOAuthRefresh : identique à MSALProvider (même endpoint refresh_token).
func (p *SISUProvider) TryOAuthRefresh(ctx context.Context, refreshToken string) (string, error)
```

> **Chemin watcher (RTA)** : `pollWatcherAuth` appelle `flow.AcquireToken()` → access_token →
> `AcquireXSTSForRTA(ctx, accessToken)`. Ce chemin n'appelle pas `provider.Exchange` et n'a
> donc **pas** besoin du contexte SISU. Il fonctionne identiquement pour MSAL et SISU —
> aucun changement dans `watcher_handler.go`.



#### 5b — Interface `DeviceFlow` (remplace le struct `DeviceCodeFlow` concret)

> **Raison** : `DeviceCodeFlow` est aujourd'hui un struct concret contenant `dc public.DeviceCode`
> (type MSAL) et `AcquireToken()` appelle directement `f.dc.AuthenticationResult()`. Ajouter des
> champs SISU dans ce même struct crée une discriminated-union masquée par `FlowType string` — l'
> anti-pattern exact à éviter. La solution : abstraire en interface.

```go
// DeviceFlow abstrait un flow interactif d'authentification (MSAL ou SISU).
// Chaque provider retourne sa propre implémentation privée.
// Défini dans provider.go (ou device_flow.go si le fichier grossit).
type DeviceFlow interface {
	// Accesseurs pour l'UI / sérialisation HTTP.
	GetMessage() string
	GetUserCode() string         // vide si SISU
	GetVerificationURL() string  // microsoft.com/devicelogin (MSAL) ou URL OAuth directe (SISU)
	GetExpiresIn() int
	GetFlowType() string         // "msal" | "sisu" — pour adapter le frontend

	// AcquireToken bloque jusqu'à l'authentification de l'utilisateur et retourne
	// l'access_token Microsoft. Bloquant — à appeler dans une goroutine.
	// Pour SISU : effectue le polling OAuth natif Xbox + appel CompleteSISUFlow.
	// Pour MSAL : délègue à dc.AuthenticationResult().
	AcquireToken(ctx context.Context) (string, error)
}
```

**Implémentations privées :**

| Type (privé) | Fichier | Notes |
|---|---|---|
| `msalDeviceFlow` | `msal_client.go` | Wraps l'actuel `DeviceCodeFlow`; `AcquireToken` → `dc.AuthenticationResult` |
| `sisuDeviceFlow` | `sisu_client.go` | Contient `kp`, `deviceToken`, `session`, `codeVerifier`, chan résultat ; `AcquireToken` → poll OAuth + `CompleteSISUFlow` |

**Conséquences en cascade (à appliquer dans la même PR) :**

```go
// TokenProvider.InitDeviceFlow — signature mise à jour
InitDeviceFlow(ctx context.Context) (DeviceFlow, error)  // était: *DeviceCodeFlow

// Attempt (attempt_store.go)
DevFlow DeviceFlow  // était: *DeviceCodeFlow

// AuthHandler.pollDeviceFlow (auth.go)
func (h *AuthHandler) pollDeviceFlow(attemptID string, flow DeviceFlow)  // était: *DeviceCodeFlow
// flow.ExpiresIn → flow.GetExpiresIn()
// flow.AcquireToken(ctx) — inchangé

// stub_provider_test.go + auto_sync_test.go — adapter la valeur de retour
// (retourner un DeviceFlow interface au lieu de *DeviceCodeFlow)
```

**Helper de test :**

```go
// NewStubDeviceFlow crée un DeviceFlow minimal pour les tests (pas de polling réseau).
// Exporté — remplace NewDeviceCodeFlow dans les stubs de test.
func NewStubDeviceFlow(userCode, verificationURL, message string, expiresIn int, flowType string) DeviceFlow
```

`NewDeviceCodeFlow` (actuellement utilisé dans les stubs) devient `NewStubDeviceFlow` et retourne
`DeviceFlow`. La suppression de l'export `DeviceCodeFlow` n'est pas un breaking change puisque le
type n'est jamais instancié en dehors du package `auth`.

#### 5c — `AppSettings` + switch instanciation

```go
// Dans AppSettings (settings/store.go)
AuthProvider string `json:"auth_provider"` // "msal" (défaut) | "sisu"
```

```go
// Dans cmd/server/main.go
var provider auth.TokenProvider
switch appSettings.AuthProvider {
case "sisu":
    provider = auth.NewSISUProvider(sisuAppID, sisuTitleID)
default: // "msal" ou vide
    provider = auth.NewMSALProvider()
}
```

#### 5d — `domain.UpdateSettingsRequest` + `domain.SettingsResponse`

Ajouter `AuthProvider *string` dans les deux structs.
Appliquer dans `settings.Apply()` et exposer dans `settings.ToResponse()`.

---

### Phase 6 — Adaptation frontend

#### Settings admin (section Synchronisation)

Nouveau champ dans le panneau Settings :

```
── Authentification Xbox ────────────────────────────────
  Méthode d'authentification
    ○ MSAL (Device Code classique)  [défaut]
    ○ SISU (natif Xbox) — expérimental
  ⚠ Le changement prend effet au prochain redémarrage du serveur.
```

#### Page onboarding / modal Device Code Flow

Adapter selon `flow_type` retourné par `GET /auth/device-flow/{id}` :

| `flow_type` | Affichage |
|-------------|-----------|
| `"msal"` | Code court + lien `microsoft.com/devicelogin` (comportement actuel) |
| `"sisu"` | Bouton "Se connecter avec Xbox" → ouvre `VerificationURL` directement (pas de code) |

---

### Phase 7 — Tests unitaires et non-régression

#### 7a — Tests unitaires par fichier (hors réseau)

**`pop_signing_test.go`** — le plus critique, tester en premier :

| Test | Entrée | Assertion |
|------|--------|-----------|
| `TestDERtoP1363_KnownVector` | Signature DER d'un message fixe avec clé fixe | r‖s = 64 octets exactement ; vecteur attendu tiré de Conch |
| `TestDERtoP1363_LengthVariants` | Signatures DER avec r ou s < 32 octets (padding) | Sortie toujours 64 octets, r et s zero-padded à gauche |
| `TestGeneratePoPKeyPair` | — | `x`, `y` sont base64url valides ; `kty="EC"`, `alg="ES256"`, `crv="P-256"` |
| `TestSignRequest_OutputFormat` | URI + body quelconques | Base64 standard décodable ; longueur header = 4+8+64 = 76 octets |
| `TestSignRequest_TimestampBounds` | Avant/après appel | FILETIME ≥ `(now-1s + 11_644_473_600) × 10_000_000` |

**`xbox_device_code_test.go`** — comportement de polling sans réseau :

| Test | Mécanique |
|------|-----------|
| `TestPollXboxDeviceCode_AuthorizationPending` | serveur HTTP test retournant `authorization_pending` × N puis succès → vérifie que la boucle continue et retourne le token |
| `TestPollXboxDeviceCode_SlowDown` | réponse `slow_down` → vérifie que `interval` augmente d'au moins 5s |
| `TestPollXboxDeviceCode_FatalError` | réponse `access_denied` → vérifie retour d'erreur immédiat (pas de retry) |
| `TestPollXboxDeviceCode_ContextCancel` | `ctx.Cancel()` pendant le polling → pas de goroutine leak |

**`sisu_client_test.go`** — avec `httptest.NewServer` :

| Test | Mécanique |
|------|-----------|
| `TestInitSISUSession_ExtractsSessionID` | serveur test avec header `X-SessionId: test-123` → `session.SessionID == "test-123"` |
| `TestCompleteSISUFlow_ExtractsXSTSFields` | réponse JSON avec `AuthorizationToken.Token + DisplayClaims` → `XSTSResult` correctement rempli |
| `TestCompleteSISUFlow_MissingToken` | réponse sans `Token` → erreur non-nil |

**`provider_test.go`** — non-régression interface :

```go
// Vérifications compile-time (ajout dans le fichier existant)
var _ auth.TokenProvider = (*auth.MSALProvider)(nil)
var _ auth.TokenProvider = (*auth.SISUProvider)(nil)
var _ auth.DeviceFlow    = auth.NewStubDeviceFlow("", "", "", 0, "msal")

// TestSISUProvider_ExchangeWithoutInit — erreur explicite si Exchange appelé sans InitDeviceFlow
func TestSISUProvider_ExchangeWithoutInit(t *testing.T) {
    p := auth.NewSISUProvider("appid", "titleid")
    _, err := p.Exchange(context.Background(), "some-token")
    if err == nil {
        t.Fatal("attendu une erreur si Exchange appelé sans InitDeviceFlow")
    }
}

// TestSISUProvider_CurrentClearedAfterExchange — p.current est nil après Exchange
// (nécessite un Exchange stub via httptest, pas de réseau réel)
```

**`auth_handler_test.go` (non-régression handlers)** :

| Test | Ce qui est protégé |
|------|-------------------|
| `TestStartDeviceFlow_MSALProvider_CompatStub` | `stubTokenProvider` retourne un `DeviceFlow` via `NewStubDeviceFlow("CODE", "https://...", "", 300, "msal")` → réponse JSON contient `user_code` et `verification_uri` |
| `TestStartDeviceFlow_SISUProvider_NoUserCode` | `NewStubDeviceFlow("", "https://sisu.example.com/...", "", 300, "sisu")` → réponse JSON contient `verification_uri` non vide, `user_code` vide ou absent |
| `TestGetDeviceFlowStatus_FlowTypeInResponse` | `flow_type` présent dans la réponse `GET /auth/device-flow/{id}` |

#### 7b — Tests d'intégration (réseau réel, tag `//go:build integration`)

Ces tests ne tournent pas en CI standard — exécution manuelle ou CI dédiée.

```go
//go:build integration

// TestSISUFullFlow_RealNetwork — flow complet SISU sans onboarding UI :
// 1. GeneratePoPKeyPair
// 2. RequestDeviceToken → token non vide
// 3. InitSISUSession → MsaOauthRedirect valide (HTTP 200 sur l'URL)
// 4. (manuel) l'opérateur ouvre l'URL et autorise
// 5. PollXboxDeviceCode → access_token + refresh_token
// 6. CompleteSISUFlow → XSTSResult.NotAfter futur
// 7. ExchangeAccessToken → SpartanToken + ClearanceToken non vides
```

#### 7c — Checklist de validation manuelle (avant Phase 8)

- [ ] Device Token obtenu sans erreur 400/401
- [ ] SISU session initiée — `MsaOauthRedirect` accessible
- [ ] OAuth poll complété — `access_token` + `refresh_token` obtenus
- [ ] `CompleteSISUFlow` → `XSTSResult` valide (token non vide, `NotAfter` futur)
- [ ] `RefreshLoop` fonctionne avec le `refresh_token` SISU (même endpoint que MSAL)
- [ ] RTA WebSocket se connecte avec le XSTS SISU
- [ ] Reconnexion automatique après expiration XSTS
- [ ] Onboarding complet nouveau joueur (SISU activé)
- [ ] Switch `auth_provider` MSAL → SISU → MSAL sans perte de données
- [ ] Tous les tests unitaires passent avec les deux providers (`go test ./internal/platform/auth/...`)

---

### Phase 7bis — Plan de logging

Chaque composant nouveau doit émettre des `slog` structurés avec les niveaux suivants.
Convention déjà en vigueur dans `provider.go` : utiliser `slog.DebugContext`, `slog.InfoContext`,
`slog.WarnContext`, `slog.ErrorContext`.

#### `pop_signing.go`

```go
slog.DebugContext(ctx, "pop_signing: clé PoP générée", "kty", kp.proofKey.Kty)
slog.DebugContext(ctx, "pop_signing: signature construite", "uri", uri, "sig_len", len(sig))
// Pas de log du body (peut contenir device_token) — uniquement URI et longueur
```

#### `device_token.go`

```go
slog.DebugContext(ctx, "device_token: requête Device Token Xbox")
slog.InfoContext(ctx, "device_token: Device Token obtenu")
slog.ErrorContext(ctx, "device_token: échec", "status", resp.StatusCode, "err", err)
```

#### `sisu_client.go`

```go
// InitSISUSession
slog.DebugContext(ctx, "sisu: initialisation session", "app_id", appID)
slog.InfoContext(ctx, "sisu: session initiée", "session_id", session.SessionID)
slog.ErrorContext(ctx, "sisu: échec init session", "status", resp.StatusCode, "err", err)

// CompleteSISUFlow
slog.DebugContext(ctx, "sisu: complétion flow", "session_id", sessionID)
slog.InfoContext(ctx, "sisu: XSTS obtenu", "not_after", result.NotAfter, "gamertag", result.Gamertag)
slog.ErrorContext(ctx, "sisu: échec complétion", "err", err)
// Ne jamais logger device_token, access_token, xsts_token
```

#### `xbox_device_code.go`

```go
slog.DebugContext(ctx, "xbox_device_code: démarrage Device Code Flow", "client_id", clientID)
slog.InfoContext(ctx, "xbox_device_code: Device Code Flow initialisé", "expires_in", res.ExpiresIn)
slog.DebugContext(ctx, "xbox_device_code: poll en attente")
// Sur slow_down :
slog.WarnContext(ctx, "xbox_device_code: slow_down reçu", "new_interval", interval)
slog.InfoContext(ctx, "xbox_device_code: Device Code Flow complété")
slog.ErrorContext(ctx, "xbox_device_code: erreur fatale", "error_code", code)
```

#### `provider.go` — `SISUProvider`

```go
// InitDeviceFlow
slog.DebugContext(ctx, "sisu_provider: démarrage Device Code Flow")
slog.InfoContext(ctx, "sisu_provider: flow prêt", "verification_url", flow.GetVerificationURL())

// Exchange
slog.DebugContext(ctx, "sisu_provider: échange access_token → XSTS SISU")
slog.InfoContext(ctx, "sisu_provider: Exchange OK", "gamertag", result.Gamertag, "xuid", result.XUID)
slog.ErrorContext(ctx, "sisu_provider: Exchange échoué", "err", err)
// Si current == nil au moment d'Exchange :
slog.ErrorContext(ctx, "sisu_provider: Exchange appelé sans contexte de flow actif")
```

#### Règle générale (aucune exception)

> **Ne jamais logger** : `access_token`, `refresh_token`, `device_token`, `xsts_token`,
> `spartan_token`, `clearance_token`, ni aucune valeur du `ProofKey` (x, y). Logger uniquement
> les longueurs, les statuts HTTP, les gamertags, les timestamps d'expiration.

---

### Phase 8 — Basculer le défaut sur `"sisu"`

```go
// defaultSettings() dans settings/store.go
AuthProvider: "sisu",
```

Documenter dans `CHANGELOG.md` + `docs/CONFIGURATION.md`.

---

### Phase 9 — Déprécier puis supprimer MSAL

Fichiers à supprimer :

| Fichier | Action |
|---------|--------|
| `internal/platform/auth/msal_client.go` | Supprimer |
| `internal/platform/auth/msal_cache_test.go` | Supprimer |
| `cmd/msal-poc/main.go` | Supprimer |
| `go.mod` / `go.sum` | Retirer `github.com/AzureAD/microsoft-authentication-library-for-go` |
| `sync_meta` key `msal_token_cache` | Ignorer à la lecture (migration douce) |

---

## Récapitulatif des fichiers touchés

| Action | Fichier |
|--------|---------|
| Nouveau | `internal/platform/auth/pop_signing.go` + `pop_signing_test.go` |
| Nouveau | `internal/platform/auth/device_token.go` + `device_token_test.go` |
| Nouveau | `internal/platform/auth/sisu_client.go` + `sisu_client_test.go` |
| Nouveau | `internal/platform/auth/xbox_device_code.go` + `xbox_device_code_test.go` |
| Modifié | `internal/platform/auth/provider.go` — ajout interface `DeviceFlow` + `SISUProvider` ; `InitDeviceFlow` retourne `(DeviceFlow, error)` |
| Modifié | `internal/platform/auth/msal_client.go` — `DeviceCodeFlow` devient `msalDeviceFlow` privé implémentant `DeviceFlow` ; `NewDeviceCodeFlow` → `NewStubDeviceFlow` (retourne `DeviceFlow`) |
| Modifié | `internal/platform/auth/attempt_store.go` — `Attempt.DevFlow *DeviceCodeFlow` → `DevFlow DeviceFlow` |
| Modifié | `internal/api/handlers/auth.go` — `pollDeviceFlow(..., flow DeviceFlow)` ; `flow.ExpiresIn` → `flow.GetExpiresIn()` |
| Modifié | `internal/api/handlers/watcher_handler.go` — `flow *DeviceCodeFlow` → `flow DeviceFlow` si nécessaire |
| Modifié | `internal/api/handlers/stub_provider_test.go` — retourner `DeviceFlow` au lieu de `*DeviceCodeFlow` ; ajouter `TestStartDeviceFlow_SISUProvider_NoUserCode` + `TestGetDeviceFlowStatus_FlowTypeInResponse` |
| Modifié | `internal/scheduler/auto_sync_test.go` — idem |
| Modifié | `internal/platform/settings/store.go` — ajout `AuthProvider string` |
| Modifié | `internal/domain/` — `UpdateSettingsRequest` + `SettingsResponse` |
| Modifié | `cmd/server/main.go` — switch instanciation provider |
| Modifié | `apps/web/src/` — Settings UI + onboarding modal |
| Supprimé (Phase 9) | `internal/platform/auth/msal_client.go`, `msal_cache_test.go`, `cmd/msal-poc/` |
| Modifié (Phase 9) | `go.mod` / `go.sum` |

---

## Points de vigilance

1. **Conversion DER → P1363** : Go (`crypto/ecdsa`) signe en ASN.1 DER ; Xbox attend IEEE P1363
   (`r‖s`, 32+32 octets). Implémenter et tester unitairement **avant** toute requête réseau.

2. **AppID Xbox** : l'appID officiel à utiliser avec SISU doit être validé expérimentalement.
   Valeur candidate : `000000004c20a908` (Halo Waypoint mobile — utilisé par Grunt/Conch).
   Tester avec `titleId: "144209987"` (Halo Infinite).

3. **PKCE cohérent** : le `code_challenge` envoyé dans `InitSISUSession` et le `code_verifier`
   utilisé dans `PollXboxDeviceCode` doivent être générés ensemble et persistés le temps du flow.

4. **`refresh_token` compatible** : le refresh_token obtenu via SISU utilise le même endpoint
   `login.live.com/oauth20_token.srf` que MSAL → `TryOAuthRefresh` est réutilisable sans changement.

5. **Cache MSAL legacy** : les installations MSAL existantes ont un `msal_token_cache` dans
   `sync_meta`. En mode SISU, ignorer ce champ silencieusement (ne pas le supprimer).

6. **Coexistence watcher** : `watcher_handler.go` appelle aussi `InitDeviceFlow`. Le switch via
   `TokenProvider` s'applique automatiquement — aucun changement dans le handler.

7. **Découplage `DeviceFlow` — point de bascule critique** : le passage de `*DeviceCodeFlow`
   (struct concret) à `DeviceFlow` (interface) impacte en cascade `attempt_store.go`,
   `auth.go`, `watcher_handler.go`, et tous les stubs de test. Ces changements doivent
   tous être dans le **même commit** que la Phase 5 pour que le code compile.
   Vérifier que `var _ DeviceFlow = (*msalDeviceFlow)(nil)` compile avant de merger.
