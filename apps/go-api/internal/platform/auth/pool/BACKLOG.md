# Backlog — Pool de tokens

## Optimisations futures

### 1. Pré-validation légère au démarrage

**Idée** : Valider les CredentialSources au moment de `NewPool()` plutôt qu'au premier `Acquire()`.

**Bénéfice** : Détecter immédiatement les tokens invalides (401/403) avant que le premier sync démarre. Échecs rapides et informatifs.

**Coût** : ~500ms-1s de latence au boot (3 tokens × latence réseau).

**Décision** : À évaluer une fois le pool en production. Probablement un flag `--token-pool-validate-boot` pour opt-in.

---

### 2. Couplage intelligent avec TokenProvider

**Idée** : Permettre à Discovery de pré-échanger les tokens au scan time au lieu d'attendre `Acquire()`.

**Détail** :
- Au lieu de : scan → store sources → (plus tard) Resolve à chaud
- Faire : scan → pré-Exchange en parallèle → cache chaud au boot

**Bénéfice** : Boot plus rapide si tous les tokens sont valides (un seul "exchange burst" à démarrage).

**Coût** : 
- Complexité : Discovery doit connaître TokenProvider (couche A parle à auth.Provider)
- Risque : Échange échoue pendant le scan → faut-il bloquer le boot ou retrying en arrière-plan ?

**Architecture recommandée** :
```go
// Nouvelle interface dans pool/
type PreWarmer interface {
    PreWarmTokens(ctx context.Context, sources []CredentialSource) map[string]*ResolvedTokens
}

// Discovery l'utiliserait optionnellement
discovery.ScanWithPreWarm(ctx, prewarmer)
```

**Status** : ✅ Dossier pour future PR après que le pool soit en prod quelques jours.

---

### 3. Refresh stratégique en arrière-plan

**Idée** : Refresher anticipativement les tokens avant leur expiration réelle (~3h30) au lieu d'attendre la première requête 401/403.

**Détail** : Une goroutine `refresherLoop` qui :
- Surveille `expiresAt` de chaque token
- ~30 min avant l'expiration, déclenche `Refresh()` silencieusement
- Log l'issue (succès / erreur non-bloquante)

**Status** : Déjà structuré pour ça — existe un `Close()` pour stopper la goroutine. À impl au besoin.

---

### 4. Collecte de métriques

**Idée** : Exposer des métriques sur le pool :
- Cache hit ratio (pour tuner le TTL)
- Refresh latencies par token
- Token health status (healthy / unhealthy / refreshing)

**Où stocker** : probablement un expvar ou une interface `Metrics()` que le pool expose.

**Status** : Nice-to-have pour observabilité production.

---

## Notes architecturales

- **Séparation Discovery ↔ Resolver** : Intentionnelle et bien testée. N'a pas besoin de TokenProvider.
- **Thread-safety** : Validée via `-race` sur les tests. Pas de regressions prévues pour l'instant.
- **Fallback smart** : MSAL → OAuth → Exchange déjà implémenté. Pattern portable si on ajoute d'autres sources (ex: SISU).
