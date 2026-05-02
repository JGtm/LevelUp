# HALO_PROVIDER_ERROR_TAXONOMY.md — Taxonomie d'erreurs provider -> produit

> Dernier livrable de cadrage avant implémentation.
> Ce document fixe comment les erreurs et dégradations venant de la couche Halo doivent être traduites dans l'API produit.

## Rôle du document

Ce document sert à éviter que le futur backend Go traite les erreurs Halo au cas par cas dans chaque handler.

Il fixe :

1. la différence entre erreur bloquante, limitation et warning ;
2. les familles d'erreurs provider à reconnaître ;
3. leur traduction vers le langage produit ;
4. leur projection vers le contrat HTTP actuel basé sur `ApiErrorSchema`.

## Ce document ne définit pas

1. les payloads métiers de succès ;
2. le détail des DTOs Halo Infinite ;
3. les retries techniques exacts du client HTTP ;
4. les erreurs purement formulaire ou validation UI qui restent côté API produit générale.

## Position dans l'architecture

```text
provider Halo
  -> erreur provider normalisée
    -> classification produit
      -> ApiErrorSchema OU limitations/warnings
        -> frontend
```

## Distinction obligatoire : erreur vs limitation

### Erreur bloquante

La route ne peut pas satisfaire son contrat utile.
On renvoie une erreur HTTP normalisée.

### Limitation

La route peut encore renvoyer une réponse utile, mais une partie de la surface est dégradée, absente ou partielle.
On renvoie `200` avec une limitation ou un warning explicite dans le payload concerné.

### Warning technique interne

L'information n'apporte rien au consommateur produit.
Elle reste dans les logs/metrics et ne sort pas dans le contrat.

## Contrat HTTP cible existant

La projection d'une erreur bloquante doit rester compatible avec le contrat actuel de l'API FastAPI :

```json
{
  "code": "halo_upstream_timeout",
  "message": "Le provider Halo n'a pas répondu à temps.",
  "retryable": true,
  "details": {
    "provider": "spnkr",
    "title": "halo_infinite"
  },
  "field_errors": null
}
```

Référence de forme : `apps/api/app/schemas/common.py` et `apps/api/app/core/errors.py`.

## Dimensions de classification à figer

| Dimension | Rôle |
|----------|------|
| `kind` | famille canonique de l'erreur provider |
| `retryable` | indique si le frontend ou l'orchestrateur peut retenter |
| `http_status` | projection HTTP cible |
| `product_code` | code fonctionnel stable lisible par machine |
| `exposure` | `hard_error`, `limitation`, `silent_log` |
| `details` | contexte utile sans fuite d'informations sensibles |

## Familles d'erreurs à reconnaître

| `kind` | Cause typique | Projection produit | HTTP | `retryable` | `product_code` recommandé |
|--------|---------------|--------------------|:----:|:-----------:|---------------------------|
| `auth_required` | session Halo absente, expirée ou clearance invalide | erreur bloquante | 401 | non | `halo_auth_required` |
| `access_denied` | provider refuse l'accès malgré une session présente | erreur bloquante | 403 | non | `halo_access_denied` |
| `resource_not_found` | joueur, match, film ou ressource introuvable côté provider | erreur bloquante | 404 | non | `halo_resource_not_found` |
| `capability_not_available` | surface non supportée pour le titre/provider courant | erreur bloquante ou limitation selon le parcours | 422 | non | `halo_capability_unavailable` |
| `rate_limited` | throttling provider | erreur bloquante | 429 | oui | `halo_rate_limited` |
| `upstream_timeout` | timeout réseau ou provider | erreur bloquante | 504 | oui | `halo_upstream_timeout` |
| `upstream_unavailable` | indisponibilité distante, maintenance, DNS, 5xx distant | erreur bloquante | 503 | oui | `halo_upstream_unavailable` |
| `upstream_invalid_response` | payload incomplet, shape incohérent, contrat distant cassé | erreur bloquante | 502 | oui | `halo_upstream_invalid_response` |
| `partial_data` | la route reste servable mais un sous-bloc manque ou est fragile | limitation | 200 | n/a | pas d'erreur HTTP par défaut |
| `internal_mapping_error` | bug de mapping canonique ou d'adaptation produit | erreur bloquante | 500 | non | `internal_error` |

## Règles de décision

### 1. Préférer `limitation` à `hard_error` si le contrat utile reste servable

Exemples :

1. skill snapshot partiel dans match view ;
2. film accessible mais extraction d'armes fragile ;
3. bloc economy indisponible alors que le bootstrap shell reste exploitable.

### 2. Utiliser `hard_error` si la ressource centrale du parcours est absente

Exemples :

1. match principal introuvable pour `/matches/{match_id}` ;
2. joueur inconnu pour une page player-scoped ;
3. auth Halo obligatoire mais absente pour un parcours live-only.

### 3. Ne pas mélanger incapacité provider et interdiction produit

Exemple :

`can_start_initial_sync = false` est un guard produit/configuration, pas une erreur provider Halo.

## Projection des limitations

Quand `kind = partial_data` ou `capability_not_available` mais que la route peut encore répondre utilement, exposer une limitation plutôt qu'une erreur HTTP.

Shape minimale recommandée :

```json
{
  "code": "halo_partial_data",
  "message": "Les données de skill sont partielles pour ce match.",
  "severity": "warning",
  "retryable": false,
  "details": {
    "capability_key": "match.skill.snapshot",
    "provider": "spnkr",
    "title": "halo_infinite"
  }
}
```

Règle :

une limitation n'est pas une mini-exception. Elle doit aider le consommateur à dégrader proprement, pas à reproduire toute la stack d'erreur.

## Champs `details` recommandés

| Champ | Usage |
|------|-------|
| `provider` | identifiant du provider courant |
| `title` | identifiant du titre courant |
| `capability_key` | surface concernée si connue |
| `resource_type` | `player`, `match`, `film`, etc. |
| `resource_id` | identifiant concerné si non sensible |
| `upstream_status` | statut HTTP distant si utile |
| `retry_after_seconds` | délai de retry sur throttling |
| `bridge_name` | nom du bridge transitoire si concerné |

À exclure :

1. URLs Waypoint complètes ;
2. tokens, headers ou secrets ;
3. payloads bruts volumineux ;
4. stack traces internes.

## Codes HTTP à conserver côté produit

| HTTP | Usage recommandé |
|------|------------------|
| 401 | auth Halo absente ou expirée |
| 403 | accès refusé ou interdit |
| 404 | ressource introuvable |
| 409 | conflit produit local, pas une erreur provider par défaut |
| 422 | capability demandée mais non disponible/supportée |
| 429 | throttling provider |
| 500 | bug interne de mapping/adaptation |
| 502 | réponse provider invalide |
| 503 | provider indisponible ou bridge transitoire indisponible |
| 504 | timeout provider |

## Règles spécifiques par parcours MVP

### Bootstrap

1. Le shell doit préférer un bootstrap `200` dégradé à un échec global si le cœur du produit peut démarrer.
2. Une capability indisponible doit être visible via `halo.capabilities` ou `halo.limitations`, pas via un 5xx global, sauf si le bootstrap lui-même devient inutilisable.

### Match View / History / Career

1. Une ressource principale manquante -> erreur bloquante.
2. Un sous-bloc partiel -> limitation locale.

### Explorer / Search

1. Recherche vide ou sans résultat -> succès vide, pas erreur.
2. Échec provider -> erreur normalisée seulement si la recherche dépend réellement du provider au moment de l'appel.

## Types Go documentaires recommandés

```go
type ProviderErrorKind string

const (
    ErrAuthRequired         ProviderErrorKind = "auth_required"
    ErrAccessDenied         ProviderErrorKind = "access_denied"
    ErrResourceNotFound     ProviderErrorKind = "resource_not_found"
    ErrCapabilityUnavailable ProviderErrorKind = "capability_not_available"
    ErrRateLimited          ProviderErrorKind = "rate_limited"
    ErrUpstreamTimeout      ProviderErrorKind = "upstream_timeout"
    ErrUpstreamUnavailable  ProviderErrorKind = "upstream_unavailable"
    ErrUpstreamInvalid      ProviderErrorKind = "upstream_invalid_response"
    ErrPartialData          ProviderErrorKind = "partial_data"
    ErrBridgeUnavailable    ProviderErrorKind = "bridge_unavailable"
    ErrInternalMapping      ProviderErrorKind = "internal_mapping_error"
)
```

## Documents liés

1. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) pour les limitations côté modèle canonique.
2. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) pour les surfaces supportées ou dégradées.
3. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) pour la projection bootstrap des capabilities et limitations.
4. [HALO_PRODUCT_CONTRACT_ADAPTERS.md](HALO_PRODUCT_CONTRACT_ADAPTERS.md) pour la couche d'adaptation canonique -> API produit.
5. [OPENAPI_MVP_P0_P1.md](OPENAPI_MVP_P0_P1.md) pour l'usage de cette taxonomie dans les premiers contrats HTTP.

## Règle d'arrêt documentaire

Après ce document, aucune nouvelle taxonomie d'erreurs préalable n'est requise avant le code.
Les écarts restants doivent être découverts et corrigés pendant le Sprint 0 ou l'implémentation, puis reportés comme deltas ciblés.
