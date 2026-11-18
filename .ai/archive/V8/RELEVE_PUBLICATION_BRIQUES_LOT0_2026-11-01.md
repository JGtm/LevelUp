# Releve lot 0 - quatre clients HTTP Halo/Xbox restes hors brique

Depot : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-publication`, branche `refactor/publication-briques`, HEAD 8f0b512c58 (2026-11-01).
Lecture seule. Aucun build/test lance. Les numeros de ligne sont ceux des fichiers au HEAD ci-dessus.
Chemins abreges : `go-api` = `apps/go-api`.

## 0. Methode D3 telle que lue (rappel utile pour les tableaux)

- ADR 0042 D-2 : API = sans etat, sans politique, sans interpretation. Exploitation = qui appelle, quand, avec quel jeton, ou va le resultat, ce que ca veut dire.
- Brique `haloapi` : `Doer` recu (jamais de client HTTP construit), `Garde` optionnelle (consultee AVANT limiteur et attente), `ErrDefinitif` / `NonRetryable` arretent la reprise ; `Retry` = valeur (9 champs) ; `Endpoints` + cles ; `Auth{Spartan, Clearance}` passe a chaque appel ; `Crochets` pour les compteurs ; imports permis : stdlib, `internal/haloapi/...`, `errgroup`, `rate` (tiers autorises par sous-arbre : `archlint/briques_internes_test.go`, `tiersHaloapi`, cle `.../haloapi/infinite` seulement).
- `infinite/transport.go` : `get` = GET JSON uniquement (`Accept: application/json`, `Accept-Language` si donne, `x-343-authorization-spartan` et `343-clearance` si non vides), corps lu en entier, erreur typee `*HTTPError{StatusCode, URL, Err, RetryAfter, Body}` (`Is(ErrAuth|ErrFilmGone)`), reprise via `Retry`. `getUnique` (nameplate.go:124) = un seul GET sans reprise, echeance 8 s. Aucun `User-Agent` ni `Origin` poses (sauf `ScraperUserAgent` dans leaderboard.go:72). Pas de GET binaire generique (le seul binaire est `DownloadBlob`, CDN films, sans jeton).
- Garde netguard cote LevelUp : un `netguardDoer{surface, next}` par surface (8 surfaces, `games/halo_infinite/haloadapter/doer.go`), exige par `archlint/haloapi_doer_garde_test.go` (litteral `netguardDoer{...}` a l'appel de `infinite.New`).

---

## 1. `go-api/internal/games/halo_5/client.go` (462 L)

### 1.A Tableau symbole par symbole

| symbole | lignes | nature | couplages internes | destination proposee |
|---|---|---|---|---|
| commentaire de paquet/recette (hotes, segment `h5`, en-tetes, `?auth=st`, UA, gzip) | 1-20 | protocole (doc de la recette) | - | `libs/haloapi/halo5` (doc.go) ; la mention "Pool Infinite" est du contexte |
| imports | 22-37 | - | `platform/netguard` ; tiers `golang.org/x/time/rate` | voir B |
| `const TitleSlug = "halo_5"` | 39-41 | **exploitation** (identite du titre ; ~90 usages hors client : `cmd/h5-*`, `internal/api`, `config`, `migrations`, `livesync`, `adapter_*`) | aucun | RESTE dans `games/halo_5` (ne doit pas suivre le client) |
| `h5StatsHost` | 45 | protocole (adresse par defaut) | duplique `config/titles/halo_5/constants.toml` `[endpoints].stats` mais n'en lit rien | `halo5.Endpoints` (defaut) |
| `h5ProfilesHost` | 50 | protocole | duplique `constants.toml` `[endpoints].nameplate` (haloplayer) | `halo5.Endpoints` (defaut) |
| `h5UserAgent` = `cpprestsdk/2.4.0` | 52 | protocole (en-tete exige cote 343) | - | `halo5` (constante) |
| `h5MaxRetries`=4, `h5RetryBaseDelay`=800 ms, `h5MaxBackoff`=10 s | 53-55 | protocole (politique de reprise) | - | `halo5.Retry` en valeur (politique d'aujourd'hui) |
| `type HTTPError{StatusCode, URL, Err}` + `Error()` + `Unwrap()` | 60-67 | protocole (erreur typee) mais **lu par l'exploitation** : `errors.As` dans `halo_5/adapter_data_loaders.go:149` ; litteraux construits dans `adapter_data_test.go:123,137`, `events_test.go:166` | aucun | `halo5.HTTPError` ; reexport (alias) a creer dans `games/halo_5` pour garder `*HTTPError` stable ; pas de `RetryAfter`, `Body`, `Is(...)` contrairement a `infinite.HTTPError` |
| `type Client{http *http.Client, spartanToken, statsBaseURL, profilesBaseURL, limiter *rate.Limiter}` | 70-76 | mixte : `http` construit en interne et jeton stocke dans le client = exploitation ; URL override + limiter = protocole | `rate.Limiter` (tiers) | `halo5.Client` immuable (doer, endpoints, retry, limiter interface) ; le jeton passe en parametre (`Auth`) ou reste dans l'adaptateur |
| `NewClient(spartanToken, requestsPerSecond)` | 80-89 | **mixte** : `&http.Client{Timeout: 20s}` (ligne 85) = exploitation (la brique ne construit jamais de client) ; `rate.NewLimiter(rps, 1)` avec defaut 10 rps (lignes 81-83, 87) = exploitation (budget) | aucun interne | adaptateur a creer (`games/halo_5/haloadapter` ou dans `halo_5`) : construit le `Doer` garde 20 s + le limiteur ; `halo5.New(doer, retry)` |
| `WithHTTPClient` | 93-98 | exploitation (couture de test) ; **aucun appelant** du `WithHTTPClient` de halo_5 (le grep ne trouve que celui de `haloadapter`/`syncpkg`, `cmd/bench-rps`) | - | disparait au profit du `Doer` |
| `WithLimiter(*rate.Limiter)` | 101-106 | protocole (accepte un limiteur) ; **aucun appelant** hors definition (grep `.WithLimiter(` : seuls les clients Infinite) | `rate` | `halo5.Client.WithLimiter(Limiter)` (interface `Wait(ctx)`) |
| `WithStatsBaseURL` / `statsHost()` | 110-122 | protocole (adresse) ; surcharge utilisee par tests `client_test.go` (4 appels) | - | `Endpoints.Avec(CleStats, ...)` |
| `WithProfilesBaseURL` / `profilesHost()` | 126-138 | protocole ; surcharge utilisee par `client_appearance_test.go` (4 appels) | - | `Endpoints.Avec(CleProfils, ...)` |
| `GetPlayerMatches(ctx, gt, start, count)` | 142-162 | protocole : URL `…/h5/players/{gt}/matches?start&count&include-times=true`, decode `H5MatchesResponse` | DTO `H5MatchesResponse` (dto.go) | `halo5` ; DTO deplaces avec |
| `GetServiceRecords(ctx, gt, recordType)` | 167-185 | protocole : `…/h5/servicerecords/{type}?players={gt}` | `H5ServiceRecordResponse` (dto.go) | `halo5` |
| `GetMatchDetail(ctx, matchID, mode)` -> `map[string]any` | 190-205 | protocole ; **aucun appelant** dans le depot (ni prod ni test) | - | `halo5` ou a ecarter : decision du pilote (je note seulement l'absence d'appelant) |
| `GetMatchCarnage` | 210-225 | protocole : `…/h5/{mode}/matches/{id}` | `H5CarnageResponse` (dto_carnage.go) | `halo5` |
| `GetMatchEvents` | 231-245 | protocole : `…/h5/matches/{id}/events` (SANS segment de mode) | `h5MatchEventsResponse` **non exporte** (events_dto.go:63) | `halo5` ; le type devra etre exporte pour rester atteignable par `games/halo_5` (interface `h5Source`, adapter_data.go:38) |
| `GetAppearance` | 251-265 | protocole : `…/h5/profiles/{gt}/appearance` (hote profils) | `H5Appearance` (dto.go:129) | `halo5` |
| `GetSpartanRenderPNG` | 271-281 | protocole : `…/h5/profiles/{gt}/spartan` (302 vers CDN, suivi par le client HTTP) | - | `halo5` |
| `GetEmblemPNG` | 288-298 | protocole : `…/h5/profiles/{gt}/emblem` (idem) | - | `halo5` |
| `doGet` | 303-361 | **mixte** : en-tetes, `?auth=st`, boucle de reprise = protocole ; `netguard.Check` (327) et `c.http.Do` direct = exploitation | `platform/netguard` | `halo5` (`get`) ; `netguard.Check` -> `Doer` garde ; classement de statuts dans la brique |
| `doGetBinary` | 368-427 | **mixte** : copie de `doGet` (Accept image, corps vide = erreur, retourne Content-Type) ; `netguard.Check` (391) | `platform/netguard` | `halo5` (`getBinaire`) ; **meme code duplique** (2 copies de la boucle) |
| `waitRetry` | 431-443 | protocole | - | `halo5` |
| `parseRetryAfterSeconds` | 448-462 | protocole (secondes seulement, borne a 10 s) | - | `halo5` ; **different** de `infinite.ParseRetryAfter` (voir C) |

Types de DTO H5 (hors `client.go`, a deplacer avec lui) : `dto.go` (154 L : `H5MatchesResponse`, `H5MatchResult`, `H5MatchID`, `H5ISODate`, `H5Team`, `H5MatchPlayer`, `H5PlayerRef`, `H5ServiceRecordResponse/Result/Body`, `H5ArenaStats`, `H5ArenaPlaylistStat`, `H5Csr`, `H5Appearance`, `H5AppearanceEmblem`, `H5AppearanceCompany`), `dto_carnage.go` (103 L : `H5CarnageResponse/Team/Player`, `H5CommendationDelta`, `H5XpInfo`), `events_dto.go` (65 L : `h5EventPlayer`, `h5WorldLocation`, `h5GameEvent`, `h5MatchEventsResponse`). Ils sont consommes par `mapping*.go`, `events.go`, `capture.go`, `career_sr.go` et `livesync/*` (`halo5.H5Csr`, `halo5.H5ArenaStats`, etc.). Je n'ai pas verifie leurs imports internes ligne a ligne (`dto.go` ne montre aucun `import` interne dans ses 20 premieres lignes) : a confirmer avant la coupe.

### 1.B Imports

- stdlib : `context, encoding/json, errors, fmt, io, net/http, net/url, strconv, strings, time`.
- tiers : `golang.org/x/time/rate` (limiteur concret `*rate.Limiter` dans `Client.limiter`, `NewClient`, `WithLimiter`).
- internes : `levelup/go-api/internal/platform/netguard` uniquement : `netguard.Check(ctx, "halo5_api.get")` (327) et `netguard.Check(ctx, "halo5_api.get_binary")` (391).
- Note : le tiers `rate` n'est permis par `briques_internes_test.go` que sous `.../haloapi/infinite` ; un futur sous-paquet `halo5` demanderait sa propre entree dans `tiersHaloapi` (ou l'interface `Limiter` seule).

### 1.C Politique de reprise (doGet 303-361, doGetBinary 368-427, waitRetry 431-443)

- 4 essais (`h5MaxRetries`, ligne 53), boucle `for attempt := 0; attempt < 4` (311, 376).
- Limiteur : `limiter.Wait(ctx)` avant CHAQUE essai (312-316) ; son erreur est rendue telle quelle, sans enveloppe (314, 379).
- `netguard.Check` APRES le `Wait` du limiteur et APRES la construction de la requete, a chaque essai (326-329) ; refus rendu brut (`ErrOffline`, non marque `NonRetryable`), la fonction sort aussitot (donc aucune reprise), mais un jeton du limiteur a deja ete consomme.
- Erreur de transport (`c.http.Do`) : retentee, `waitRetry(ctx, attempt, 0)` (332-336). Corps coupe sur un 200 : retente (343-347).
- 200 : retourne ; 401 et 403 : terminal, `HTTPError` "token invalide/expire" (349-350) ; 404 et 410 : terminal "ressource absente" (351-352) ; **tout autre statut (429, 5xx, mais aussi 400, 408...) est retente** (353-358).
- Attente : `delay = 800 ms * 2^attempt`, remplace par `Retry-After` s'il est plus grand, borne a 10 s (432-438) ; **attente aussi apres le dernier essai** (waitRetry appele sans condition sur `attempt`) ; `waitRetry` rend sur `ctx.Done()` mais la boucle continue (pas d'arret sur contexte : l'essai suivant echoue sur le limiteur ou sur `Do`).
- `parseRetryAfterSeconds` (448-462) : entier de secondes uniquement (date HTTP -> 0), <= 0 -> 0, plafond 10 s.
- Message final : `doGet %s: %d tentatives echouees: %w` (360) ; `doGetBinary` idem (426).

Comparaison avec `infinite/retry.go` :

| champ `Retry` | halo_5 actuel | RetryProvider | RetryDiscovery | RetryHaloClient |
|---|---|---|---|---|
| Essais | 4 | 3 | 3 | 4 |
| statut retente | tout non-200 hors 401/403/404/410 (429 compris) | idem | 5xx seuls | tout sauf 429 (429 rendu tout de suite) |
| HonoreRetryAfter | oui | non | non | oui |
| DelaiBase / Plafond | 800 ms / 10 s | idem | 800 ms / sans plafond | idem |
| AttenteFinale | oui | oui | non | oui |
| ArretSurContexte | non | non | oui | non |
| LimiteurUneFois | non (avant chaque essai) | non | oui | non |
| IgnoreErreurLimiteur | non (erreur rendue) | non | non | oui |
| Garde (netguard) | APRES limiteur, a chaque essai | avant limiteur (brique) | avant limiteur | avant limiteur |

Verdict : **differente des trois**. Elle est "RetryProvider avec 4 essais et Retry-After honore" : combinaison absente de `infinite/retry.go` (RetryHaloClient a 4 essais + Retry-After mais renvoie le 429 et ignore l'erreur du limiteur ; RetryProvider retente le 429 et rend l'erreur du limiteur mais a 3 essais et ignore Retry-After). Les 9 champs de `Retry` suffisent a l'exprimer (Retente: toujours vrai ; HonoreRetryAfter: vrai ; AttenteFinale: vrai ; IgnoreErreurLimiteur: faux). Differences de forme a noter : (a) `Retry-After` en date HTTP ignore ici, lu par `infinite.ParseRetryAfter` (plafond 5 min puis plafond de politique 10 s) ; (b) le `HTTPError` de halo_5 ne porte ni `RetryAfter` ni `Body` ; (c) position de la garde (apres le limiteur ici) ; (d) l'erreur finale (`tentatives echouees`) vs `doGet %s: %d tentatives échouées` d'infinite (accents).

### 1.D Surfaces netguard

| appel | ligne | surface |
|---|---|---|
| `netguard.Check(ctx, "halo5_api.get")` dans `doGet` | 327 | `halo5_api.get` |
| `netguard.Check(ctx, "halo5_api.get_binary")` dans `doGetBinary` | 391 | `halo5_api.get_binary` |

Les deux surfaces ne figurent dans aucune table ni test (grep `halo5_api` : seul `client.go`). Ratchet `platform/netguard/netguard_coverage_test.go` : le fichier contient `.Do(req)` ET `netguard.Check` => conforme.

### 1.E URL et en-tetes produits

Hotes (defauts du paquet, lignes 45 et 50) : stats = `https://spartanstats.svc.halowaypoint.com` ; profils = `https://haloplayer.svc.halowaypoint.com`. Aucun passage par `games.DefaultEndpointResolver` (les hotes sont des constantes ; `config/titles/halo_5/constants.toml` lignes 25-32 porte les memes chaines mais le client ne le lit pas).

Chaque URL recoit en plus `auth=st` ajoute en fin (303-308 / 368-373) : `&auth=st` si l'URL a deja un `?`, sinon `?auth=st`.

| methode | URL produite (avant ajout `auth=st`) | lignes |
|---|---|---|
| `GetPlayerMatches` | `{stats}/h5/players/{PathEscape(gt)}/matches?start={n}&count={n}&include-times=true` -> final `...&include-times=true&auth=st` | 151-152 |
| `GetServiceRecords` | `{stats}/h5/servicerecords/{PathEscape(type)}?players={QueryEscape(gt)}` -> `...&auth=st` | 174-175 |
| `GetMatchDetail` | `{stats}/h5/{PathEscape(mode)}/matches/{PathEscape(id)}` -> `?auth=st` | 194-195 |
| `GetMatchCarnage` | meme chemin que `GetMatchDetail` | 214-215 |
| `GetMatchEvents` | `{stats}/h5/matches/{PathEscape(id)}/events` -> `?auth=st` | 235 |
| `GetAppearance` | `{profils}/h5/profiles/{PathEscape(gt)}/appearance` -> `?auth=st` | 255 |
| `GetSpartanRenderPNG` | `{profils}/h5/profiles/{PathEscape(gt)}/spartan` -> `?auth=st` | 275 |
| `GetEmblemPNG` | `{profils}/h5/profiles/{PathEscape(gt)}/emblem` -> `?auth=st` | 292 |

En-tetes (JSON, `doGet` 321-324) : `Accept: application/json` ; `X-343-Authorization-Spartan: <spartanToken>` (toujours pose, meme vide) ; `User-Agent: cpprestsdk/2.4.0` ; AUCUN `343-clearance` ; `Accept-Encoding` non pose (gzip transparent, test `TestClient_GzipTransparent`) ; pas d'`Accept-Language`.
En-tetes (binaire, `doGetBinary` 386-388) : `Accept: image/png,image/*` ; `X-343-Authorization-Spartan` ; `User-Agent: cpprestsdk/2.4.0`.
Redirections : politique par defaut de `http.Client` (suivi automatique, commentaire 367) ; le `?auth=st` n'est pas rajoute sur la cible. Fait Go a verifier dans le temoin : les en-tetes personnalises (donc le jeton Spartan) sont renvoyes sur la redirection vers `image.halocdn.com` (seuls Authorization/Cookie/WWW-Authenticate sont retires entre domaines).
Casse : `Header.Set` canonicalise (`X-343-Authorization-Spartan`), identique octet pour octet a `x-343-authorization-spartan` d'Infinite en HTTP/1.1.
Timeout `http.Client` : 20 s (85).

### 1.F expvar / metriques

Aucun (`grep expvar|metrics|counter` : 0). Seuls journaux : aucun `slog` dans ce fichier.

### 1.G Appelants et tests

Symboles exportes du client : `Client`, `NewClient`, `HTTPError`, `TitleSlug`, `WithHTTPClient`, `WithLimiter`, `WithStatsBaseURL`, `WithProfilesBaseURL`, `Get*`.
- `NewClient(` : un seul appelant de production, `games/halo_5/adapter_data.go:76` (dans `NewSpartanTokenSource`). Les autres constructions viennent des tests (`client_test.go`, `client_appearance_test.go`).
- Sources produites par `NewSpartanTokenSource` / `NewCaptureSource` (capture.go:42) / `NewAppearanceSource` (capture.go:~70) : `internal/api/wire/server_titles_additional.go:126` (`halo5.NewDataAdapter(halo5.NewSpartanTokenSource, ...)`), `cmd/server/main.go:1196` (`NewAppearanceSource`), `cmd/levelup/cmd_backfill_h5_kill_mechanics.go:198`, et les CLI `cmd/h5-backfill`, `h5-csr-backfill`, `h5-csr-match-backfill`, `h5-events-backfill`, `h5-kill-kind-backfill`, `h5-roster-refetch`, `h5-teamscore-backfill` (`NewCaptureSource`), `cmd/h5-appearance-backfill` (`NewAppearanceSource`).
- Satisfactions d'interface : `var _ h5Source = (*Client)(nil)` (adapter_data.go:41), `CaptureSource` (capture.go:36), `AppearanceCapableSource` (capture.go:65), `livesync/appearance_persist.go:49` (`var _ AppearanceSource = (*halo5.Client)(nil)`). Les commentaires de `livesync/csr_match.go:22` mentionnent `*halo5.Client`.
- `HTTPError` : `adapter_data_loaders.go:149` (`errors.As`), tests ci-dessus.
- `TitleSlug` : cf. tableau (hors client en fait).
- Mention dans `archlint/no_halowaypoint_literal_test.go:65` : `internal/games/halo_5/client.go` est dans l'allowlist `halowaypointAllowlist` (a retirer/deplacer si le fichier part).
- Tests couvrant le client : `halo_5/client_test.go` : `TestClient_RequestRecipe` (chemin, jeton, UA, absence de clearance, `auth=st`), `TestClient_GzipTransparent`, `TestClient_PlayerMatches_IncludeTimes`, `TestClient_Unauthorized` ; `halo_5/client_appearance_test.go` : `TestClient_GetAppearance`, `TestClient_GetEmblemPNG_FollowsRedirect`, `TestClient_GetSpartanRenderPNG_FollowsRedirect`, `TestClient_GetEmblemPNG_NotFound`. Aucun test de la reprise (retries, 429, Retry-After, backoff) : seul le 401 terminal est teste. Aucun test de `GetMatchCarnage`, `GetMatchEvents` (URL) ni `GetMatchDetail`. Les tests passent par `WithStatsBaseURL/WithProfilesBaseURL` + `httptest`.

### 1.H Secret / jeton

Le jeton est un champ du client (`spartanToken`, ligne 72) pose par `NewClient(spartanToken, ...)` (80). Sa source : `ctxkeys.HaloTokens(ctx).SpartanToken` lu dans `games/halo_5/adapter_data.go:71-76` (`NewSpartanTokenSource`) ; erreur si absent. Le client lui-meme ne lit ni contexte, ni store, ni env. Pose a chaque requete dans `X-343-Authorization-Spartan` (322, 387). Le journal ne l'imprime pas (pas de `slog` dans le fichier) mais l'URL (`auth=st` seulement) est incluse dans les messages d'erreur ; `HTTPError.Error()` imprime `HTTP %d: %s` avec l'URL (donc pas de jeton).

---

## 2. `go-api/internal/mapcatalog/ugc.go` (185 L)

### 2.A Tableau symbole par symbole

| symbole | lignes | nature | couplages internes | destination proposee |
|---|---|---|---|---|
| commentaire de paquet `mapcatalog` + de fichier | 1-19 | exploitation (explique pourquoi le fichier est dans `mapcatalog`) ; le "temoin d'authentification" (anonyme -> 401) est du protocole | - | le commentaire de paquet reste dans `mapcatalog` (le paquet porte d'autres fichiers) ; la recette d'auth va dans la doc du service |
| `discoveryHost` = `https://discovery-infiniteugc.svc.halowaypoint.com` | 38 | protocole (adresse) ; **identique** a `infinite.CleDiscoveryUGC` | - | `infinite.Endpoints` (deja la) |
| `requestTimeout` = 30 s | 39 | exploitation (delai du client HTTP) | - | adaptateur (Doer) |
| `maxMvarBytes` = 64 MiB | 40 | mixte : garde-fou de taille de lecture (protocole si limite par requete) | - | brique (option de lecture) ou adaptateur |
| `politeUserAgent` = `LevelUp/1.0 (dashboard stats Halo, usage personnel)` | 41 | mixte : identite de l'appelant (valeur LevelUp, mais en-tete = protocole) | - | parametre (`Auth`/options) ; la valeur reste LevelUp |
| `type Asset` (AssetId, VersionId, PublicName `string`, Files{Prefix, FileRelativePaths}, CustomData{NumOfObjectsOnMap, TagLevelId, IsBaked}) | 46-59 | protocole (DTO brut) ; **different** de `infinite.DiscoveryAssetDTO` : `PublicName` est `string` ici (le DTO infinite le lit en `json.RawMessage`), `CustomData` absent d'infinite | - | DTO dans `infinite` (nouveau ou etendu) |
| `type Client{http, tokens *domain.HaloTokens}` | 61-64 | mixte : `tokens` portes par le client = exploitation | `domain.HaloTokens` | brique : `Client` + `Auth` par appel ; `mapcatalog` garde une facade `NewClient(tokens)` |
| `NewClient(tokens)` | 66-71 | mixte : construit `&http.Client{Timeout: 30 s}` (exploitation) | `domain.HaloTokens` | adaptateur construit le Doer garde |
| `(*Client).FetchAsset(ctx, assetID, versionID)` | 75-96 | **mixte** : URL + decode = protocole ; validation `Files` vide = interpretation (exploitation) ; log INFO `mapobj: asset résolu` = exploitation | - | service UGC de `infinite` (`FetchMapAsset`) rend le DTO ; la validation et le log restent dans `mapcatalog` |
| `(*Asset).MvarPaths()` | 99-107 | exploitation (filtre `.mvar` = choix metier) | - | reste dans `mapcatalog` |
| `(*Client).FetchMvar(ctx, asset, relPath)` | 110-118 | mixte : delegue a `FetchFileAt` ; log INFO = exploitation | - | reste dans `mapcatalog` (appelle la brique) |
| `(*Client).FetchFileAt(ctx, prefix, relPath)` | 128-133 | protocole (GET du stockage blob sans jeton) | netguard (via `get`) | `infinite` (telechargement de fichier du stockage blob, sans jeton) |
| `(*Client).get(ctx, endpoint, withAuth)` | 145-185 | **mixte** : netguard (146), en-tetes (153-164), `Do`, statut non-200 -> erreur avec extrait de 512 octets, lecture bornee = transport (protocole) ; message d'erreur `aucun jeton Spartan disponible` = exploitation | `platform/netguard` | transport de `infinite` ou petit transport dedie ; garde -> Doer |

### 2.B Imports

- stdlib : `context, encoding/json, fmt, io, log/slog, net/http, net/url, strings, time`.
- tiers : aucun.
- internes : `internal/domain` : uniquement `domain.HaloTokens` (champ `Client.tokens`, `SpartanToken` 156/159, `ClearanceToken` 160-161) ; `internal/platform/netguard` : `netguard.Check(ctx, "mapcatalog_ugc.get")` (146).
- Observation : les AUTRES fichiers du paquet `mapcatalog` importent `games/halo_infinite/replaylabels`, `i18n`, `domain/title` (callouts_*.go) ; le fichier `ugc.go` ne partage aucun import avec eux.

### 2.C Politique de reprise

**Aucune** : un seul essai (`get`, 145-185). Pas de limiteur, pas de `Retry-After`, pas d'attente, pas de distinction 401/404/5xx : tout non-200 devient `fmt.Errorf("GET %s: HTTP %d (%s)", ...)` avec les 512 premiers octets du corps (175-179). Delai global : timeout du `http.Client` (30 s). Echec de transport : `GET %s: %w` (167). Comparaison : plus proche de `getUnique` d'`infinite` (un GET, pas de reprise, nameplate.go:124) que d'une des trois politiques ; differences avec `getUnique` : echeance 30 s portee par le client vs 8 s par contexte, erreur non typee (pas de `*HTTPError`), lecture bornee a 64 MiB, `Accept` + `User-Agent` + `Origin` poses. Cote appelants : "les deux appelants traitent deja tout echec reseau en best-effort" (commentaire 140-143), ce qui est vrai pour `haloadapter/client_mvar.go` (via le client de sync, qui reprend ailleurs) et `cmd/mapobj-build` (echec journalise, carte suivante).

### 2.D Surfaces netguard

| appel | ligne | surface |
|---|---|---|
| `netguard.Check(ctx, "mapcatalog_ugc.get")` en tete de `get` (donc pour Discovery ET stockage blob) | 146 | `mapcatalog_ugc.get` |

Une seule surface pour deux hotes. Ratchet couverture : conforme (`.Do(req)` ligne 166 avec `netguard.Check` dans le fichier).

### 2.E URL et en-tetes produits

| appel | URL | lignes |
|---|---|---|
| `FetchAsset(assetID, "")` | `https://discovery-infiniteugc.svc.halowaypoint.com/hi/Maps/{PathEscape(assetID)}` | 76 |
| `FetchAsset(assetID, versionID != "")` | `.../hi/Maps/{PathEscape(assetID)}/versions/{PathEscape(versionID)}` | 76-79 |
| `FetchFileAt(prefix, relPath)` / `FetchMvar` | `{TrimSuffix(prefix, "/")}/{relPath}` (prefixe = `Files.Prefix` du DTO ou `BlobPrefix` de l'inventaire ; hote blob, non fige dans le fichier) ; `relPath` NON echappe | 132 |

Points distinctifs par rapport a `infinite.FetchAsset` : segment `Maps` (majuscule M, `maps` chez infinite), prefixe de jeu **en dur `hi`** (infinite : `Endpoints.PrefixeJeu()`), `versionID` optionnel (infinite : requis, erreur sinon).

En-tetes (toutes requetes, 153-154) : `User-Agent: LevelUp/1.0 (dashboard stats Halo, usage personnel)` ; `Accept: application/json` (meme pour le blob `.mvar`). Si `withAuth` (Discovery seulement, 155-164) : `x-343-authorization-spartan: <SpartanToken>` (erreur avant envoi si jetons nil ou Spartan vide, 156-158) ; `343-clearance: <ClearanceToken>` si non vide ; `Origin: https://www.halowaypoint.com` (163, literal `halowaypoint` autorise par l'allowlist `no_halowaypoint_literal_test.go:92`). Pas d'`Accept-Language`. Stockage blob : aucun en-tete d'auth.
Taille : `io.LimitReader(resp.Body, 64 MiB)` (180) ; erreur non-200 : extrait 512 octets (176).

### 2.F expvar / metriques

Aucun. Journaux : `slog.InfoContext "mapobj: asset résolu"` (91-94), `slog.InfoContext "mapobj: mvar téléchargé"` (115-116), `slog.WarnContext "mapobj: fermeture du corps de réponse"` (172).

### 2.G Appelants et tests

Symboles exportes : `Asset`, `Client`, `NewClient`, `FetchAsset`, `MvarPaths`, `FetchMvar`, `FetchFileAt`.
- `internal/games/halo_infinite/haloadapter/client_mvar.go:30-42` : `mapcatalog.NewClient(&domain.HaloTokens{Spartan, Clearance})`, `ugc.FetchAsset(ctx, mapID, "")`, `asset.MvarPaths()`, `ugc.FetchMvar(...)`. Appelant de production du runtime (`FetchMvarForMap`, repris par `sync/pooled_client.go:282`, `sync/fetch_cache.go:110`, rattrapage `sync/replayartifacts/mvar_rattrapage.go`). Un nouveau client est construit A CHAQUE appel.
- `cmd/mapobj-build/main.go:117` (`NewClient(tokens)`), `:192-209` (`ingestRemote` : `FetchAsset`, `MvarPaths`, `FetchMvar`) ; jetons via `cmd/mapobj-build/auth.go:25` (`SPARTAN_TOKEN`/`CLEARANCE_TOKEN` env, sinon store via `RefreshHaloTokensViaStoreFirst`).
- `cmd/mapcallouts-build/forge_fetch.go:97-104` : `NewClient(nil)` + `FetchFileAt` (sans jeton, blob).
- Autres importeurs du paquet `mapcatalog` qui n'utilisent PAS le client : `cmd/levelup/cmd_backfill_map_callouts*.go`, `cmd/mapcallouts-build/*`, `cmd/mapopads-build/*`, `service/map_module_resolver.go`, `service/replay_map_callouts.go`, `sync/replayartifacts/{mvar_rattrapage,zones_rattrapage}*.go`.
- **Tests du client : aucun** (aucun `ugc_test.go` ; aucun test ne reference `NewClient`, `FetchFileAt`, `FetchMvar`, `discoveryHost`, `politeUserAgent`). Le ratchet `no_halowaypoint_literal_test.go:92` nomme le fichier.
- Rapport avec `haloapi/infinite/discovery.go` : **perimetre different, pas un doublon strict**. Memes hote et jetons, mais : (i) URL `…/hi/Maps/{id}[/versions/{v}]` ici vs `{hote}/{prefixe}/{maps|playlists|mapModePairs|ugcGameVariants}/{id}/versions/{v}` ; (ii) DTO different (`CustomData`, `PublicName` string vs RawMessage + `Description`) ; (iii) retour : ici le DTO Files complet pour telecharger les `.mvar`, la-bas un `DiscoveryAsset` (nom localise + `ImageURL` de miniature) ; (iv) `infinite.FetchAsset` passe par la reprise `RetryDiscovery` + limiteur + `Accept-Language`, ce client n'a rien de cela ; (v) le telechargement du `.mvar` depuis le stockage blob n'a pas d'equivalent dans `infinite` (`DownloadBlob` cible le CDN des films, avec `Cache-Control: no-cache` sur reprise 304, sans `Accept`). `infinite.GetPlaylistConfig` (playlist.go:45-51) est un troisieme chemin Discovery (`/playlists/{id}/versions/{v}`).

### 2.H Secret / jeton

Parametre de construction : `NewClient(tokens *domain.HaloTokens)` (66) ; stocke dans `Client.tokens` (63) ; lu a la requete (156-161) uniquement si `withAuth`. Sources reelles : `haloadapter/client_mvar.go:30-32` (jetons de la session de sync, `c.spartanToken`/`c.clearanceToken`) et `cmd/mapobj-build/auth.go`. `nil` accepte pour le chemin blob. Pas de lecture de contexte ni de store dans le fichier.

---

## 3. `go-api/internal/assets/fetcher_gamecms.go` (356 L)

### 3.A Tableau symbole par symbole

Types/symboles de meme paquet utilises sans import (couplages "implicites" a `assets`) : `Fetcher`, `Kind*` (kinds.go), `Ref`, `Payload`, `BinaryPayload`, `JSONPayload`, `URLPayload` (fetcher_chain.go), `ErrNotFound`, `ErrUpstreamUnavailable`, `ErrUnsupportedKind` (errors.go), `MimeImagePNG`, `contentHash` (store_localfs.go:16,150), `TokenProvider` (fetcher.go:11).

| symbole | lignes | nature | couplages internes | destination proposee |
|---|---|---|---|---|
| `defaultGameCMSBase` = `https://gamecms-hacs.svc.halowaypoint.com` | 16 | protocole (adresse) ; **utilise aussi** par `fetcher_chain.go:67` ; duplique `infinite.CleGameCMS` | - | `infinite.Endpoints` ; la constante disparait des deux fichiers de `assets` |
| `gamecmsHostFor(ctx, fallback)` | 21-28 | **exploitation** : lit le singleton `games.DefaultEndpointResolver()` + `ctxkeys.TitleSlug(ctx)` + `games.EndpointGameCMS` | `games`, `ctxkeys` | reste dans `assets` -> construit des `Endpoints` (adaptateur "EndpointsDepuisResolveur" du plan) ; aussi appelee par `fetcher_chain.go:92` |
| `(*GameCMSFetcher).base(ctx)` | 31-33 | exploitation (choix d'hote par titre) | idem | reste dans `assets` |
| `gamecmsPrefixFor(ctx)` | 38-40 | exploitation : `games.GamePrefix(ctxkeys.TitleSlug(ctx))` ; appelee aussi par `fetcher_chain.go:92` | `games`, `ctxkeys` | reste dans `assets` |
| `(*GameCMSFetcher).gamePrefix(ctx)` | 43-45 | exploitation | idem | reste |
| `type GameCMSFetcher{httpClient *http.Client, tokens TokenProvider, baseURL}` | 50-54 | mixte (client + fournisseur de jetons + base) | `TokenProvider` -> `domain.HaloTokens` | reste cote `assets` (compose le service de la brique) |
| `NewGameCMSFetcher(client, tokens, baseURL)` | 58-66 | mixte : `client == nil -> http.DefaultClient` (63) = exploitation | - | reste (adaptateur) |
| `Supports(k)` | 69-83 | exploitation (quels `Kind` GameCMS sert) | `Kind*` | reste |
| `Fetch(ctx, ref)` | 86-104 | exploitation (dispatch par `Kind`) | `Ref`, `Payload` | reste |
| `fetchMedalImage` | 109-131 | **mixte** : URL + GET = protocole ; mapping statut->`ErrNotFound/ErrUpstreamUnavailable`, `BinaryPayload`, `contentHash` = exploitation | `ErrNotFound`, `BinaryPayload`, `contentHash`, `MimeImagePNG` | brique : `FetchMedalImage(auth?) -> []byte` ; `assets` fait le reste |
| `fetchChallengeBadge` | 135-161 | mixte (idem ; jetons obligatoires) | idem + `resolveTokens` | idem |
| `fetchGameCMSImage` | 167-196 | mixte (jetons facultatifs ; `http.DetectContentType`) | idem | idem |
| `buildGameCMSImageFetchURL(base, prefix, raw)` | 198-227 | protocole (construction d'URL pure) ; **quasi-doublon** de `infinite.buildCustomizationImageURL` (appearance.go:164-178) mais plus large (rejette `.json`, gere `progression/file/` et `waypoint/file/`) | - | brique (fonction pure testable) |
| `fetchMedalMetadata` | 230-245 | mixte | `JSONPayload` | idem (`Metadata/Metadata.json`) |
| `fetchChallengeDefinition` | 249-272 | mixte | idem | idem |
| `fetchRewardTrackDefinition` | 276-299 | mixte (**meme corps** que le precedent et le suivant) | idem | idem |
| `fetchBPItemDefinition` | 303-326 | mixte | idem | idem |
| `(*GameCMSFetcher).doGet(ctx, url, tokens)` | 329-348 | **mixte** : `netguard.Check` (334) = exploitation ; en-tetes + `httpClient.Do` = protocole | `platform/netguard`, `domain.HaloTokens` | transport de la brique ; garde -> Doer |
| `(*GameCMSFetcher).resolveTokens(ctx)` | 351-356 | exploitation | `TokenProvider` | reste |

Remarque de forme (a titre de fait) : les 5 fonctions `fetch*` partagent un corps quasi identique (GET, 404 -> NotFound, != 200 -> Upstream, ReadAll) ; trois d'entre elles (`fetchChallengeDefinition`, `fetchRewardTrackDefinition`, `fetchBPItemDefinition`) ne different que par le libelle d'erreur.

### 3.B Imports

- stdlib : `context, fmt, io, net/http, strings`.
- tiers : aucun.
- internes : `ctxkeys` -> `ctxkeys.TitleSlug(ctx)` (lignes 23 et 39 ; valeur "halo_infinite" si absente) ; `domain` -> `domain.HaloTokens` (signature de `doGet` 329 et `resolveTokens` 351 ; champs `SpartanToken`, `ClearanceToken`) ; `games` -> `games.DefaultEndpointResolver()` (22), `res.HostFor(slug, games.EndpointGameCMS)` (23), `games.GamePrefix(slug)` (39) ; `platform/netguard` -> `netguard.Check(ctx, "gamecms_assets.get")` (334).

### 3.C Politique de reprise

**Aucune** : un seul GET par appel (`doGet`, 329-348). Pas de limiteur, pas de `Retry-After`, pas de delai propre. Le `http.Client` est celui du constructeur : `http.DefaultClient` si `nil` (63), donc **sans timeout** (en production `assets.AssetConfig.HTTPClient` n'est jamais renseigne : `server_apiv1.go:1450` pose `TokenProvider`, pas `HTTPClient`). Classement : 404 -> `ErrNotFound` (images, definitions) ; tout autre non-200 -> `ErrUpstreamUnavailable` (sans reprise) ; `fetchMedalMetadata` n'a PAS de branche 404 (237). Comparaison avec `infinite/retry.go` : n'en est aucune des trois ; plus proche de `getUnique` (un seul GET) mais sans echeance ni erreur typee. Les reprises eventuelles se font plus haut (`ChainFetcher`/resolveur), hors de ce fichier.

### 3.D Surfaces netguard

| appel | ligne | surface |
|---|---|---|
| `netguard.Check(ctx, "gamecms_assets.get")` en tete de `doGet` (toutes les `fetch*`) | 334 | `gamecms_assets.get` |

Garde prouvee par `assets/demo_offline_no_persist_test.go` (0 requete, cache vide en demo) et par le ratchet de couverture ; `archlint` historique : `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md` ligne 1363 (B-C3).

### 3.E URL et en-tetes produits

Base = `f.base(ctx)` : `gamecmsHostFor(ctx, f.baseURL)` -> si le resolveur partage est cable ET renvoie un hote pour (slug du ctx, `games.EndpointGameCMS`) : cet hote sans `/` final ; sinon `f.baseURL` (defaut `https://gamecms-hacs.svc.halowaypoint.com`, `/` final retire a la construction, 65). Prefixe = `games.GamePrefix(slug)` (`hi`/`h5`, defaut `hi`).

| fonction | URL | lignes |
|---|---|---|
| `fetchMedalImage` | `{base}/{prefix}/Progression/file/medals/{ref.TitleID}/{ref.ID}.png` | 110 |
| `fetchChallengeBadge` | `{base}/{prefix}/waypoint/file/images/{ref.ID}.png` | 140 |
| `fetchGameCMSImage` | `buildGameCMSImageFetchURL(base, prefix, ref.ID)` : `""` si vide ou suffixe `.json` (-> `ErrNotFound`) ; URL absolue `http(s)://` renvoyee telle quelle ; sinon (apres `TrimLeft "/"`) : si commence par `{p}/images/file/`, `{p}/progression/file/` ou `{p}/waypoint/file/` -> `{base}/{cleaned}` ; si commence par `images/file/`, `progression/file/` ou `waypoint/file/` -> `{base}/{prefix}/{cleaned}` ; sinon `{base}/{prefix}/images/file/{cleaned}` (comparaisons en minuscules) | 171, 198-227 |
| `fetchMedalMetadata` | `{base}/{prefix}/Progression/file/Metadata/Metadata.json` | 231 |
| `fetchChallengeDefinition` | `{base}/{prefix}/Progression/file/{TrimLeft(ref.ID,"/")}` | 255 |
| `fetchRewardTrackDefinition` | `{base}/{prefix}/Progression/file/{TrimLeft(ref.ID,"/")}` | 282 |
| `fetchBPItemDefinition` | `{base}/{prefix}/Progression/file/{TrimLeft(ref.ID,"/")}` | 309 |
| (fetcher_chain.go:92, hors fichier) | `{gamecmsHostFor}/{gamecmsPrefixFor}/Progression/file/medals/sprites/{TitleID}.png` | - |

`ref.ID` et `ref.TitleID` ne sont NI echappes NI valides dans ce fichier (verification eventuelle : `ref_validate.go`, non lu).

En-tetes (`doGet` 341-346) : **uniquement** si `tokens != nil` : `x-343-authorization-spartan: <SpartanToken>` (pose meme vide) et `343-clearance: <ClearanceToken>` si non vide. **Aucun `Accept`, aucun `User-Agent`, aucun `Accept-Language`** (valeurs par defaut de Go : `User-Agent: Go-http-client/1.1`). Jetons envoyes : `fetchMedalImage` et `fetchMedalMetadata` -> `nil` (publics) ; `fetchChallengeBadge`, `fetchChallengeDefinition`, `fetchRewardTrackDefinition`, `fetchBPItemDefinition` -> obligatoires (erreur `ErrUpstreamUnavailable` "tokens unavailable" si le fournisseur echoue ; si `f.tokens == nil` la requete part SANS en-tetes) ; `fetchGameCMSImage` -> facultatifs (`tokens, _ := ...`, erreur du fournisseur ignoree, ligne 170, sans journal).

### 3.F expvar / metriques

Aucun dans le fichier. (`assets/metrics.go` : interface `Metrics` + `NoopMetrics`, utilisee par le resolveur/la file d'ecriture, pas par ce fichier ; `wire.go:80` pose `NoopMetrics`.)

### 3.G Appelants et tests

Symboles exportes : `GameCMSFetcher`, `NewGameCMSFetcher`, `Supports`, `Fetch`. Non exportes utilises hors fichier : `defaultGameCMSBase` (fetcher_chain.go:67), `gamecmsHostFor`, `gamecmsPrefixFor` (fetcher_chain.go:92).
- Production : `assets/wire.go:72` (`NewGameCMSFetcher(cfg.HTTPClient, cfg.TokenProvider, cfg.GameCMSBaseURL)`, enchaine dans `SpritesheetFallbackFetcher` + `multiFetcher`) ; `cmd/refresh-career-ranks/main.go:96` (`NewGameCMSFetcher(http.DefaultClient, tokenProvider, "")`, `Fetch` de `KindRewardTrackDefinition`) ; `scripts/warm_bp_assets/main.go:183` (via `assets.New`).
- Jetons de prod : `internal/api/server_apiv1.go:1450` (`TokenProvider: reg.AnyPlayerTokens`, `internal/api/wire/registry_auth.go:338` : premier joueur du pool dont le rafraichissement reussit).
- Tests : `assets/fetcher_gamecms_test.go` (18 tests : `TestGameCMSFetcher_Supports`, `_Fetch_UnsupportedKind`, `_MedalImage_OK/_NotFound/_ServerError`, `_ChallengeBadge_OK/_TokenError`, `_MedalMetadata_OK`, `_BPImage_OK/_NotFound`, `_SpartanIdentityImages_OK`, `_ChallengeDefinition_OK`, `_RewardTrackDefinition_NotFound`, `_BPItemDefinition_OK/_NotFound/_TokensRequired/_TokensUnavailable_Fails`, `_DoGet_SetsHeaders`) ; `assets/demo_offline_no_persist_test.go` (garde demo ; commentaire en tete : "Seule source d'un payload BINAIRE : GameCMSFetcher") ; `assets/fetcher_chain_test.go` (non lu). Pas de test de `buildGameCMSImageFetchURL` isole ni des chemins `waypoint/progression/images` (a verifier dans `_SpartanIdentityImages_OK`, 206-253, non lu en detail).
- Ratchet : `archlint/no_halowaypoint_literal_test.go:61` (`internal/assets/fetcher_gamecms.go` dans l'allowlist ; `assets/kinds.go` y est aussi).
- **Rapport avec `haloapi/infinite/gamecms.go` (105 L)** : **perimetres differents, aucun recouvrement d'URL**. `infinite/gamecms.go` : `FetchMedalsMetadata` -> `{GameCMS}/{jeu}/Progression/file/medals/metadata.json`, `FetchSeasonCalendar` -> `.../Progression/file/SeasonCalendar.json`, `FetchCSRSeasonCalendar` -> `.../Progression/file/CsrSeasonCalendar.json` (GET JSON, reprise via la politique du client, DTO bruts, jetons par `Auth`). `assets` : images binaires (PNG) + definitions JSON (`Progression/file/{path}`) + `Metadata/Metadata.json` (casse et dossier differents de `medals/metadata.json`). Les deux partagent l'hote GameCMS et le segment de jeu, jamais un chemin. Zones proches ailleurs dans `infinite` : `appearance.go:134,164-178` (`/progression/file/{path}` en minuscules + `buildCustomizationImageURL`, quasi-doublon de `buildGameCMSImageFetchURL`) et `nameplate.go:24,56,114,119` (`/Waypoint/file/...`). Contraintes transport d'`infinite` a connaitre : `get` impose `Accept: application/json` et une reprise, et lit en `[]byte` : il ne sert pas tel quel a des PNG.

### 3.H Secret / jeton

`TokenProvider func(ctx) (*domain.HaloTokens, error)` (assets/fetcher.go:11) injecte a la construction (`tokens`, ligne 52), appele par `resolveTokens` (351-356) a chaque requete concernee ; puis pose en en-tetes dans `doGet` (341-346). Production : `reg.AnyPlayerTokens` (jetons du premier joueur du pool, hors rattachement a un joueur precis). `nil` admis (assets publics). Aucun acces au store `platform/auth` depuis le fichier ; aucun jeton journalise.

---

## 4. `go-api/internal/sync/xbox_client.go` (271 L)

### 4.A Tableau symbole par symbole

| symbole | lignes | nature | couplages internes | destination proposee |
|---|---|---|---|---|
| commentaire de fichier (endpoint, contract-version, pagination) | 1-5 | protocole (doc) | - | `libs/haloapi/xboxlive` |
| `xboxAchievementsBaseURL` = `https://achievements.xboxlive.com` | 23 | protocole (adresse) | - | `xboxlive` (Endpoints/defaut) |
| `interface XboxAchievementsClient{GetPlayerAchievements(ctx, xuid, lang)}` | 27-29 | exploitation (couture de test de `sync`, `SyncAchievements`) ; retourne `[]PlayerAchievementRaw` | `PlayerAchievementRaw` | RESTE dans `sync` ; l'adaptateur de `sync` convertit le DTO brut en `PlayerAchievementRaw` |
| `type PlayerAchievementRaw` (15 champs : ID, Name, Description, LockedDesc, Gamerscore, ImageURL, IsSecret, RarityCategory, RarityPercent, Unlocked, UnlockedAt, CurrentProgress, TargetProgress, XboxTitleID, ServiceConfigID) | 32-51 | **exploitation** (type de domaine LevelUp : `Unlocked`, `UnlockedAt time.Time`, progression 1er requirement, gamerscore extrait) ; utilise par `achievements.go` (merge, i18n) | - | RESTE dans `sync` (le DTO brut `xboxlive` est `xboxAchievementItem`) |
| `type xboxHTTPClient{authHeader, httpClient, xboxTitleID}` | 54-58 | mixte : en-tete d'auth stocke dans le client, `http.Client` construit en interne | - | `xboxlive.Client` (Doer, Endpoints, Retry) ; `authHeader` en parametre par appel |
| `NewXboxHTTPClient(xstsResult *auth.XSTSResult, xboxTitleID)` | 63-69 | **mixte** : `xstsResult.AuthHeader()` (65) ; `&http.Client{Timeout: 30 s}` (66) = exploitation | `platform/auth.XSTSResult` (**alias de `haloapi/xblauth.XSTSResult`**, `platform/auth/xbl_xsts_rta.go:17`) | adaptateur dans `sync` (Doer garde 30 s) ; `xblauth` fournit deja `XSTSResult.AuthHeader()` (xsts_rta.go:37) donc la brique `xboxlive` peut prendre un `string` d'autorisation sans importer `xblauth` |
| `(*xboxHTTPClient).GetPlayerAchievements` (pagination) | 74-99 | mixte : boucle de pagination `skipItems += len(batch)` jusqu'a `continuationToken == ""` = protocole ; logs = exploitation | - | `xboxlive` (pagination) ou `sync` : a trancher ; la boucle depend de `fetchPage` |
| `xboxAchievementsResponse` | 103-109 | protocole (DTO brut) | - | `xboxlive` |
| `xboxAchievementItem` | 111-156 | protocole (DTO brut, 45 L) ; reference directement par `sync/achievements_test.go` (6 tests) | - | `xboxlive` (export requis : aujourd'hui non exporte, le test de `sync` construit le litteral) |
| `(*xboxHTTPClient).fetchPage` | 159-213 | **mixte** : URL, en-tetes, GET, statut != 200 -> erreur avec corps entier, decode = protocole ; `netguard.Check` (183) ; la conversion `parseAchievementItem` (206-210) = exploitation | `platform/netguard` | `xboxlive` (`GetAchievementsPage` -> DTO + continuation) |
| `parseAchievementItem` | 216-271 | **exploitation** (interpretation : gamerscore du reward `Gamerscore`, icone `Type == "Icon"`, 1er titleAssociation, `TimeUnlocked` zero-value `0001-01-01T00:00:00Z`, 1er requirement) | - | RESTE dans `sync` (conversion DTO -> `PlayerAchievementRaw`) |

### 4.B Imports

- stdlib : `context, encoding/json, fmt, io, log/slog, net/http, net/url, strconv, time`.
- tiers : aucun.
- internes : `platform/auth` -> uniquement le type `*auth.XSTSResult` et sa methode `AuthHeader()` (ligne 63-65) ; `platform/netguard` -> `netguard.Check(ctx, "xbox_achievements.get")` (183).
- Note : `auth.XSTSResult = xblauth.XSTSResult` (alias). Le paquet `sync` importe donc `platform/auth` ici pour un alias de type de la brique ; ce n'est pas un couplage au magasin de jetons. (Dans `sync`, `platform/auth` est importe ailleurs, p. ex. `engine_postsync_csr.go`, pour `AcquireXSTSForRTA`.)

### 4.C Politique de reprise

**Aucune** : un essai par page (`fetchPage`, 159-213). Pas de limiteur, pas de `Retry-After`. Tout statut != 200 : `xbox_client: HTTP %d: %s` avec **le corps complet** de la reponse (197-199). Echec transport : `xbox_client: HTTP do: %w`. Delai : timeout 30 s du `http.Client` (66). Une page en erreur abandonne tout (`return nil, err`, 82-85) ; la reprise est a l'etage superieur (cycle de sync). Comparaison : aucune des trois politiques d'`infinite` ; differences notables : pas de typage d'erreur (`*HTTPError`), message avec le corps complet, pas de distinction 401/404.

### 4.D Surfaces netguard

| appel | ligne | surface |
|---|---|---|
| `netguard.Check(ctx, "xbox_achievements.get")` dans `fetchPage`, APRES construction de la requete et AVANT `Do` | 183 | `xbox_achievements.get` |

Verifie aussi : le GET de pagination repasse par `fetchPage`, donc la garde est consultee a chaque page. Ratchet couverture : conforme. Aucune autre mention de la surface dans le depot.

### 4.E URL et en-tetes produits

URL (160-171) : `url.Parse("https://achievements.xboxlive.com/users/xuid({xuid})/achievements")` (le `xuid` est insere TEL QUEL par `Sprintf`, non echappe) puis `RawQuery = q.Encode()` avec : `titleId={xboxTitleID}` si `xboxTitleID != ""` ; `skipItems={n}` si `n > 0`. `q.Encode()` trie les cles alphabetiquement : `skipItems=..&titleId=..` (ordre `skipItems` puis `titleId`). Exemples : page 1 : `.../users/xuid(2535...)/achievements?titleId=1144039928` ; page 2 : `...?skipItems=100&titleId=1144039928`. Note : les parentheses de `xuid(...)` passent par `url.Parse`/`String()` ; la forme exacte sur le fil est a figer par le temoin (Go ne reecrit pas `(` `)` dans le chemin).
En-tetes (177-180) : `Authorization: <authHeader>` (`XBL3.0 x=<UserHash>;<Token>`, xblauth/xsts_rta.go:36-39) ; `x-xbl-contract-version: 2` ; `Accept-Language: <lang>` (pose meme vide) ; `Accept: application/json`. Pas de `User-Agent` explicite (Go par defaut).
`lang` : etiquette Halo principale de chaque langue de `config/locales.toml` (`i18n.HaloTags(l)[0]`, `sync/achievements.go:98-102`), p. ex. `en-US`, `fr-FR`. Un appel par langue (boucle `fetchAchievementsByLocale`), pagination par appel.

### 4.F expvar / metriques

Aucun. Journaux `slog` : Debug "xbox_client: début récupération achievements" (78), Warn "xbox_client: erreur fetchPage" (82-83), Debug "xbox_client: page achievements récupérée" (87-91).

### 4.G Appelants et tests

Symboles exportes : `XboxAchievementsClient`, `PlayerAchievementRaw`, `NewXboxHTTPClient`.
- Production : `sync/engine_postsync_csr.go:197` (`NewXboxHTTPClient(xstsResult, titlePkg.XboxTitleIDFor(e.titleSlug))` puis `SyncAchievements(ctx, client, ...)`) ; `sync/achievements.go` (interface `XboxAchievementsClient` lignes 52 et 91 ; `PlayerAchievementRaw` lignes 92, 112, 113, 115). Appelants amont : `RunAchievementsOnly`, `RunAchievementsHook` (engine_postsync_csr.go), CLI `sync-achievements`, hook post-sync des runners live (dont Halo 5).
- Tests : `sync/achievements_test.go` (19 occurrences : `TestMergeAchievements_*`, `TestParseAchievementItem_GamerscoreFromRewards`, `_UnlockedFromProgressState`, `_ImageFromMediaAssets`, `_ProgressionRequirement`, `_RarityPreserved`, `_TimeUnlocked_ZeroValue_Ignored`, `TestWarmAchievementImages_*`) ; `sync/achievements_mocks_integration_test.go` (`mockXboxClient` implemente `XboxAchievementsClient`) ; `sync/achievements_integration_test.go`, `achievements_outcome_test.go`, `achievements_race_test.go`. **Aucun test du chemin HTTP** (`xboxHTTPClient`, `fetchPage`, URL, en-tetes, pagination) : aucune reference a `xboxHTTPClient` ni a `achievements.xboxlive` dans les tests.
- Remarque sur le plan : le plan dit "Xbox : profils, presence... lire sur pieces". **Sur pieces, ce client ne fait QUE les succes (achievements v2)** ; presence/profils sont ailleurs (`internal/presence/rest_client.go`, resolveurs de profil dans `xblauth`/`platform/auth/xbl_resolvers.go`), hors de ce fichier.

### 4.H Secret / jeton

Parametre de construction : `NewXboxHTTPClient(xstsResult *auth.XSTSResult, ...)` (63) ; l'en-tete complet est calcule UNE fois (`xstsResult.AuthHeader()`, 65) et stocke dans `xboxHTTPClient.authHeader` (55). Le jeton XSTS vient de `engine_postsync_csr.go:150` : `auth.AcquireXSTSForRTA(ctx, accessToken)` (facade `platform/auth/xbl_xsts_rta.go:53`, qui delegue a `xblauth.AcquireXSTSForRTA` avec `chaineDepuisCtx`), a partir de `accessToken` resolu par `e.resolveAchievementsAccessToken(ctx)` (store `watcher_tokens`, ADR 0023). Le client n'accede ni au store ni au contexte pour un jeton. Pas de journalisation du jeton (le log imprime `xuid` et `lang`).

---

## 5. Synthese par client

**halo_5/client.go (462 L).** Part protocole estimee ~85 % (~390 L : URL, en-tetes, reprise, decode, `HTTPError`) ; exploitation ~15 % (~70 L : `TitleSlug` const, `NewClient` qui construit `http.Client` 20 s + limiteur 10 rps, jeton stocke, 2 `netguard.Check`). Difficulte moyenne : le client est simple et deja stateless, mais (1) les ~16 DTO (`dto*.go`, `events_dto.go`) doivent suivre et `h5MatchEventsResponse` n'est pas exporte, (2) `TitleSlug` (~90 usages) et `HTTPError` (lu par `adapter_data_loaders.go`) restent atteignables depuis `games/halo_5`, (3) la politique de reprise (4 essais, tout statut retente 429 compris, Retry-After seconde, limiteur avant chaque essai, garde APRES le limiteur) ne correspond a aucune des 3 politiques d'`infinite` : elle s'exprime en `Retry` mais la position de la garde change (avant le limiteur dans la brique). Risques : temoin d'URL (`auth=st`, `include-times`, `players=` encode en query), redirections 302 avec jeton envoye (a figer), tiers `rate` a autoriser sous le futur sous-arbre `halo5`, deux surfaces netguard (`halo5_api.get`, `halo5_api.get_binary`) a conserver, `GetMatchDetail` sans appelant.

**mapcatalog/ugc.go (185 L).** Protocole ~55 % (~100 L : URL, en-tetes, `Asset` DTO, `get`, `FetchFileAt`) ; exploitation ~45 % (`MvarPaths`, validation `Files` vide, logs INFO, `NewClient` + timeout 30 s, jetons `domain.HaloTokens`). Difficulte moyenne-haute : ce n'est PAS un doublon de `infinite.FetchAsset` (segment `Maps`, prefixe `hi` en dur, version optionnelle, DTO different, aucune reprise, `Origin`/UA propres, telechargement blob sans jeton) ; aucun test existant => les temoins seraient a creer de zero. Risques : `Origin` + `User-Agent` custom absents du transport d'`infinite` ; une seule surface netguard (`mapcatalog_ugc.get`) pour deux hotes ; lecture bornee a 64 MiB ; 3 appelants (runtime `haloadapter/client_mvar.go`, `mapobj-build`, `mapcallouts-build`) ; entree dans l'allowlist `halowaypoint`.

**assets/fetcher_gamecms.go (356 L).** Protocole ~35 % (~125 L : URL, `buildGameCMSImageFetchURL`, `doGet`) ; exploitation ~65 % (resolution d'hote par titre via singleton + `ctxkeys`, dispatch `Kind`, `Payload`, `contentHash`, mapping d'erreurs `assets.Err*`, `TokenProvider`). Difficulte moyenne : peu de protocole propre ; 5 fonctions `fetch*` quasi identiques ; transport d'`infinite` ne sait ni binaire ni `Accept` absent (il impose `Accept: application/json` et une reprise) ; aucun recouvrement d'URL avec `infinite/gamecms.go` mais `buildGameCMSImageFetchURL` quasi-double `infinite.buildCustomizationImageURL`. Risques : pas d'`Accept` ni de `User-Agent` poses aujourd'hui (en poser change la requete) ; `http.DefaultClient` sans timeout en prod ; erreur du `TokenProvider` ignoree en silence dans `fetchGameCMSImage` (170) ; `defaultGameCMSBase` et `gamecmsHostFor/PrefixFor` partages avec `fetcher_chain.go` ; test demo `demo_offline_no_persist_test.go` et allowlist `halowaypoint`.

**sync/xbox_client.go (271 L).** Protocole ~60 % (~160 L : URL, en-tetes, pagination, DTO brut 55 L, `fetchPage`) ; exploitation ~40 % (`PlayerAchievementRaw`, `XboxAchievementsClient`, `parseAchievementItem` 55 L, `NewXboxHTTPClient`, timeout 30 s). Difficulte faible-moyenne : un seul appelant de production, jeton passe en parametre (deja le schema cible) ; `platform/auth` n'est importe que pour l'alias `XSTSResult` = `xblauth.XSTSResult`. Risques : le chemin HTTP n'a AUCUN test (temoin a creer : URL `xuid(...)` non echappe, ordre `skipItems`/`titleId`, en-tetes) ; `xboxAchievementItem` non exporte mais construit en litteral par 6 tests de `sync` ; le plan parle de "profils, presence" alors que le fichier ne fait que les succes ; reprise inexistante.

---

## 6. Ecarts avec le plan du 2026-10-30

Plan (`.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md` lignes 52-55) :

| affirmation du plan | etat au HEAD 8f0b512c58 |
|---|---|
| `halo_5/client.go` 462 L, n'importe que `platform/netguard` | **Exact pour les imports internes** (un seul : `platform/netguard`) et la taille (462 L). Precision omise : il importe aussi le tiers `golang.org/x/time/rate` (`*rate.Limiter` dans le type), non permis hors `infinite` par `briques_internes_test.go`. Autres precisions : les DTO (`dto.go` 154 L, `dto_carnage.go` 103 L, `events_dto.go` 65 L) sont HORS `client.go` mais indissociables ; `h5MatchEventsResponse` est non exporte ; `TitleSlug` (const du meme fichier) est utilisee par ~90 fichiers et ne suit pas le client ; le client ne lit pas `config/titles/halo_5/constants.toml` (hotes en dur). |
| `mapcatalog/ugc.go` 185 L, importe `domain`, `netguard` | **Exact** (185 L ; `domain` = `HaloTokens` seulement ; `netguard` = une surface). Precision : aucun test du client ; l'`Origin` `https://www.halowaypoint.com` est un litteral `halowaypoint` allowliste (`no_halowaypoint_literal_test.go:92`). Le plan parle d'un "troisieme client UGC [qui] rejoint `FetchAsset`/`GetPlaylistConfig`" : exact en nombre de clients Discovery (infinite `FetchAsset`, `GetPlaylistConfig`, celui-ci) mais les URL differ (`Maps` vs `maps`, prefixe en dur, version optionnelle) : ce n'est pas la meme requete. |
| `assets/fetcher_gamecms.go` 356 L, importe `ctxkeys`, `domain`, `games`, `netguard` | **Exact** (356 L ; `ctxkeys.TitleSlug`, `domain.HaloTokens`, `games.DefaultEndpointResolver/GamePrefix/EndpointGameCMS`, `netguard`). Precisions : le fichier depend aussi sans import de nombreux symboles du paquet `assets` (`Kind*`, `Ref`, `Payload*`, `Err*`, `contentHash`, `MimeImagePNG`, `TokenProvider`) ; `defaultGameCMSBase`, `gamecmsHostFor` et `gamecmsPrefixFor` sont aussi consommes par `fetcher_chain.go` (hors fichier) ; "GameCMS -> service de `infinite`" : `infinite/gamecms.go` couvre d'AUTRES fichiers (medailles `medals/metadata.json`, calendriers de saisons), pas les images ni les definitions de `Progression/file/{path}`. |
| `sync/xbox_client.go` 271 L, importe `platform/auth`, `netguard` | **Exact** (271 L ; `platform/auth` = `*auth.XSTSResult` + `AuthHeader()`, **alias** de `xblauth.XSTSResult` ; `netguard` = `xbox_achievements.get`). Ecart de fond : le plan decrit ce client comme "Xbox : profils, presence..." ; le fichier ne fait que les **succes Xbox (achievements v2)** + pagination. |
| "Aucun README sous `libs/`, aucun tag" / ADR 0042 D-9 "Four other clients are not folded" | Coherent avec le HEAD. |

Autres constats a signaler au pilote (faits, sans proposition) :
1. Trois fichiers sur quatre sont dans l'allowlist `halowaypoint` (`no_halowaypoint_literal_test.go:61,65,92`) ; `xbox_client.go` n'y est pas (hote `xboxlive.com`).
2. Le ratchet `platform/netguard/netguard_coverage_test.go` exempte `haloapi/` par prefixe ; un client deplace vers `haloapi/halo5` ou `haloapi/xboxlive` perdrait son `netguard.Check` local et dependrait du `Doer` garde + de `archlint/haloapi_doer_garde_test.go` (qui n'exige aujourd'hui le litteral `netguardDoer{...}` que pour `infinite.New` / `infinite.NewClassement`, table `argumentsDoerDesConstructeurs`).
3. Les 5 surfaces netguard de ces fichiers (`halo5_api.get`, `halo5_api.get_binary`, `mapcatalog_ugc.get`, `gamecms_assets.get`, `xbox_achievements.get`) ne sont dans aucune table de surfaces ; la liste D-6 de l'ADR 0042 en compte huit (toutes Infinite).
4. Tests du chemin HTTP : halo_5 oui (8 tests, sans test de reprise) ; assets oui (18 tests) ; mapcatalog/ugc.go NON ; xbox_client.go (HTTP) NON.
5. `GetMatchDetail` (halo_5/client.go:190) n'a aucun appelant dans le depot.
6. Aucun des quatre fichiers ne declare de compteur expvar ni ne depend de `observability`.
