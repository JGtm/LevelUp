# Plan — Refactor Spartan Identity (bannière / emblème / Spartan ID)

**Date** : 2026-05-25
**Branche** : `fix/art-eradication-and-home-resilience`
**Statut** : RÉVISÉ 2026-05-25 PM — bascule LIVE pur (voir §11) + bug token mismatch suspect (voir §12)

---

## 1. Le problème en une phrase

Sur la home, certains joueurs (Madina, Chocoboflor) n'ont pas leur bannière
Spartan affichée alors qu'ils en ont déjà eu une — et on n'a aucun moyen
fiable de garantir que tous les joueurs aient toujours la leur.

---

## 2. État des lieux (mesuré, pas supposé)

J'ai écrit un petit diag Go (`cmd/diag_customization_population`) qui scanne
toutes les bases joueurs. Résultat ce matin :

| Joueur | total rows | banner stocké | emblem stocké | dernière bannière |
|--------|-----------:|--------------:|--------------:|-------------------|
| JGtm | 86 | **1 / 86** | 13 / 86 | aujourd'hui |
| Madina97294 | 65 | **0 / 65** | 9 / 65 | **jamais** |
| XxDaemonGamerxX | 0 | 0 | 0 | jamais |
| Chocoboflor | ? (DB lockée par serveur) | ? | ? | ? |

Constats :
- Même pour JGtm qui a une sync récente, seulement 1 row sur 86 contient une
  bannière. Les 13 rows avec emblème sont les rares moments où l'API
  customisation a renvoyé quelque chose.
- Madina n'a JAMAIS eu de bannière en DB (dernière sync customisation :
  2026-05-08, soit 17 jours).
- XxDaemonGamerxX : aucune sync n'a jamais réussi pour lui (tokens absents
  ou autre).

---

## 3. Pourquoi c'est cassé — la racine

La table actuelle s'appelle `career_progression` et stocke **deux choses
différentes mélangées** :

- **Rank + XP** : change à chaque match. Append-only justifié (historique).
- **Customisation** (bannière, emblème, Spartan ID) : change très rarement.
  Stockée dans les mêmes lignes append-only, ce qui force à faire un
  hack SQL (`ARG_MAX FILTER WHERE NOT NULL`) pour récupérer la dernière
  valeur non-vide.

Et **3 chemins d'écriture différents** écrivent dans cette table, chacun avec
ses propres règles bizarres :

1. **Sync de match (legacy)** — écrit rank + XP seulement. Jamais bannière.
2. **Visite home avec tokens** — écrit tout, mais seulement si le cache
   mémoire est chaud (donc seulement à partir de la 2ᵉ visite dans une
   fenêtre de 6h).
3. **Mon ajout d'hier (`CustomizationRefresher`)** — écrit tout, toutes
   les 6h, pour les joueurs présents dans le pool de tokens.

Résultat : impossible de savoir pourquoi un joueur n'a pas sa bannière. Est-ce
que l'API a échoué ? Le cache n'a jamais chauffé ? Les tokens manquaient ?
On ne sait pas, on ne peut pas debugger, et le UI montre juste un placeholder.

**À noter** : le fallback nameplate (`ResolveNameplateURL`) qui dérive une
bannière depuis l'emblème est CORRECT et fonctionne. Le bug n'est pas là.

---

## 4. Solution proposée — table dédiée + un seul écrivain

### Le concept (en français, pas en Go)

Au lieu d'avoir une table qui mélange historique de rang et customisation,
on crée **une deuxième table** dédiée à la customisation. Cette table aura
**une seule ligne par joueur** (pas un historique de snapshots), qu'on
remplace à chaque rafraîchissement (UPSERT). Une seule fonction écrit dans
cette table, elle tourne toutes les 6h, et elle note explicitement le statut
du dernier essai (`ok`, `api a rendu vide`, `tokens manquants`, etc.).

La home, au lieu de faire un hack SQL avec `ARG_MAX`, lit juste cette ligne
unique. Si elle est vide, on sait pourquoi grâce à `last_attempt_status`.

### Structure de la nouvelle table

```sql
CREATE TABLE spartan_identity (
  xuid VARCHAR PRIMARY KEY,           -- 1 ligne par joueur
  spartan_id VARCHAR,                  -- "OKLM", "X4G", etc.
  banner_image_url VARCHAR,            -- URL nameplate
  emblem_image_url VARCHAR,            -- URL emblème
  backdrop_image_url VARCHAR,          -- URL backdrop
  last_refreshed_at TIMESTAMP,         -- dernier fetch réussi (au moins partiel)
  last_attempt_at TIMESTAMP,           -- dernier essai (réussi ou non)
  last_attempt_status VARCHAR          -- 'ok' / 'api_empty' / 'auth_missing' / 'failed'
);
```

### Bénéfice immédiat

| Question | Aujourd'hui | Après |
|---|---|---|
| Qui écrit dans cette table ? | 3 endroits | 1 endroit |
| Comment je sais pourquoi un joueur n'a pas de bannière ? | Je sais pas | `SELECT last_attempt_status FROM spartan_identity WHERE xuid = ?` |
| Combien de temps avant qu'un nouveau joueur ait sa bannière ? | Aléatoire (cache + sync + visite home) | Max 6h (cycle scheduler garanti) |
| Le code SQL contient des hacks ? | Oui (`ARG_MAX FILTER WHERE NOT NULL`) | Non (`SELECT * WHERE xuid = ?`) |
| Le cache mémoire et la DB peuvent se désynchroniser ? | Oui | Non (cache mémoire supprimé) |

---

## 5. Phases d'implémentation

Chaque phase est livrable séparément (commit propre, tests verts). Si tu
veux arrêter à la fin d'une phase, le système reste fonctionnel.

### Phase 0 — Diag (FAIT)

J'ai déjà écrit `cmd/diag_customization_population` pour mesurer l'état
actuel. À la fin du refactor, on le garde (utile pour vérifier la santé du
système périodiquement) OU on le supprime.

**Effort** : 0 (déjà fait).

### Phase 1 — Créer la table + repository (1h)

- Nouvelle migration au boot : crée la table `spartan_identity` dans chaque
  player DB si elle n'existe pas.
- Nouveau fichier `internal/platform/duckdb/spartan_identity_repo.go` avec :
  - `Load(ctx, xuid)` → retourne 1 ligne ou nil
  - `Upsert(ctx, xuid, data, status)` → insère ou met à jour
- Tests unitaires : DB en mémoire, UPSERT idempotent.

**Risque** : aucun. Code nouveau, ne touche rien d'existant.

### Phase 2 — Renommer mon refresher + le faire écrire dans la nouvelle table (30 min)

Ce que j'ai créé hier (`CustomizationRefresher` dans
`internal/scheduler/customization_refresh.go`) devient
`SpartanIdentityRefresher`. Le code change très peu : au lieu d'écrire dans
`career_progression`, il écrit dans `spartan_identity` via le nouveau repo.

Il continue à tourner toutes les 6h via le scheduler, et on ajoute aussi
un refresh au boot du serveur (pour ne pas attendre 6h).

**Risque** : faible. Si ça plante, l'ancien chemin existe encore.

### Phase 3 — Faire lire la home depuis la nouvelle table (30 min)

Modifier `LoadSpartanIdentity` pour qu'elle lise `spartan_identity` au lieu
de faire le hack `ARG_MAX` sur `career_progression`.

**Risque** : moyen. Il faut bien tester que la home affiche toujours la
bannière correctement.

### Phase 4 — Backfill au boot depuis l'ancienne table (30 min)

Pour ne rien perdre, au premier boot après le refactor, on remplit
`spartan_identity` depuis les valeurs qui étaient dans `career_progression`
(via le même `ARG_MAX FILTER`). Idempotent : si la nouvelle table contient
déjà une ligne pour ce xuid, on skip.

**Risque** : faible.

### Phase 5 — Nettoyer (30 min — APRÈS validation visuelle)

Une fois que tu confirmes que les 4 joueurs ont bien leur bannière :

- Supprimer le code qui écrit la customisation dans `career_progression`
  depuis `CareerLiveService` (les colonnes restent en DB pour rollback,
  on les drop dans 1 mois si tout va bien).
- Supprimer le cache mémoire customisation (devient inutile).
- Supprimer le fix `kickoffBackgroundRefresh persist` que j'ai mis ce
  matin (devient obsolète).

**Risque** : faible si Phase 3 a bien marché.

### Phase 6 — Tests + livraison (1h)

- `go test ./...` + `go vet ./...`
- Lancer le serveur, attendre 1 cycle, vérifier que les 4 joueurs ont
  une ligne dans `spartan_identity` avec un `last_attempt_status` explicite.
- Vérification visuelle home pour chaque joueur.
- Thought log.

---

## 6. Effort total

| Phase | Temps | Risque |
|-------|------:|--------|
| 0 — Diag | fait | — |
| 1 — Table + repo | 1h | aucun |
| 2 — Refresher | 30 min | faible |
| 3 — Reader | 30 min | moyen |
| 4 — Backfill | 30 min | faible |
| 5 — Nettoyage | 30 min | faible |
| 6 — Tests | 1h | — |
| **Total** | **~4h** | |

---

## 7. Les 2 questions que tu n'as pas comprises ce matin

### Q2 : "Le CustomizationRefresher — on le garde ou on supprime ?"

Ce que ça voulait dire : hier j'ai créé un fichier qui fait tourner un
refresh toutes les 6h (`internal/scheduler/customization_refresh.go`). Ce
fichier est utile, il est presque parfait pour la nouvelle architecture.
La seule chose à changer : il écrit aujourd'hui dans `career_progression`,
il faudra le faire écrire dans `spartan_identity`.

**Ma décision (que je propose)** : on le garde, on le renomme
`spartan_identity_refresh.go`, et on change 5 lignes pour qu'il écrive
dans la nouvelle table.

### Q3 : "Tes commits récents — on garde ou on revert ?"

J'ai fait 3 changements aujourd'hui sur cette branche :
1. **Frontend** : remplacé la bannière par défaut bizarre par ton image
   `banner-default.png`. → **Utile, on garde** (sert de filet ultime
   pour un joueur tout neuf sans aucune sync).
2. **Scheduler** : ajouté `CustomizationRefresher`. → **Devient
   `SpartanIdentityRefresher`** (cf. Q2 ci-dessus).
3. **CareerLiveService** : fait persister la bannière depuis le background
   refresh. → **Devient inutile** après le refactor, je le supprimerai
   en Phase 5.

**Ma décision (que je propose)** : on garde tout. Le 1 et 2 servent au
refactor, le 3 sera supprimé en Phase 5 sans douleur.

---

## 8. Ce qui ne change PAS

- L'API Halo elle-même (`GetSpartanCustomization`, `ResolveNameplateURL`).
- La table `career_progression` continue d'exister pour rank + XP.
- Le scheduler global (`AutoSyncScheduler`) garde son cycle de 6h.
- Le frontend (`HomeSpartanIdentityBanner.tsx`) ne change pas.
- Le contrat API JSON `spartan_identity.banner_image_url` ne change pas.

---

## 9. Décision attendue de ta part

Avant que je touche au code, j'ai besoin que tu valides :

1. **L'approche** : on part bien sur table dédiée `spartan_identity` ?
2. **L'ordre des phases** : ok ou tu veux que je commence par une autre ?
3. **Les commits d'hier/ce matin** : ok pour les garder et bâtir au-dessus,
   ou tu veux un revert total avant de commencer ?

Si tu valides, je commence par Phase 1 (créer la table + repo) et je te
montre le diff avant de passer à la Phase 2.

---

## 10. Ce que je NE ferai PAS sans te demander

- Toucher au schéma `career_progression` (sauf pour ajouter des commentaires
  "deprecated").
- Supprimer des colonnes.
- Changer le contrat API frontend.
- Faire un PR ou merger quoi que ce soit.
- Ajouter des features non listées ici.

---

## 11. Révision 2026-05-25 PM — bascule LIVE pur (pas de scheduler 6h)

### Pourquoi cette révision

Le plan original (§5 Phases 2-5) déplaçait la customisation vers un
**scheduler 6h** (`CustomizationRefresher`). C'est l'inverse de l'intention
produit : **customisation ET rank/progress doivent être en LIVE, pas au sync**.

Le scheduler est confortable (cycle garanti même si user ne visite jamais
sa home) mais introduit une cadence indépendante de l'usage réel — et
duplique le chemin live déjà en place dans `CareerLiveService`.

### Architecture révisée

**Le seul écrivain dans `spartan_identity` devient `CareerLiveService`**
(chemin home visit, via `kickoffBackgroundRefresh`). Le scheduler 6h est
supprimé. Le cache mémoire (TTL 6h) est conservé comme throttle anti-DDoS
Halo, pas comme source de vérité.

| Aspect | Plan original | Plan révisé |
|---|---|---|
| Écrivain de `spartan_identity` | Scheduler `CustomizationRefresher` toutes les 6h | `CareerLiveService.kickoffBackgroundRefresh` (déclenché par visite home) |
| Cadence de rafraîchissement | Cycle garanti max 6h | Cycle = à chaque visite home (avec throttle cache mémoire 6h) |
| Cache mémoire | À supprimer (devient inutile car DB toujours fraîche) | À garder (throttle anti-DDoS Halo) |
| `CustomizationRefresher` (mon ajout d'hier) | Renommé `SpartanIdentityRefresher` et gardé | **Supprimé** + son tick scheduler |
| Cas "joueur jamais connecté" | Couvert par scheduler 6h | Non couvert (acceptable : pas de bannière utile si pas de visite) |

### Phases révisées (~3h)

| Phase | Action | Durée |
|-------|--------|------:|
| 1 | Table `spartan_identity` (UPSERT 1 row/xuid) + repo `Load/Upsert` + tests `:memory:` | 1h |
| 2 | `kickoffBackgroundRefresh` UPSERT dans `spartan_identity` au lieu d'écrire les champs custom dans `career_progression` | 30 min |
| 3 | `LoadSpartanIdentity` (`home_repo_identity.go`) lit `spartan_identity` direct + fallback `career_progression` ARG_MAX si row absente | 30 min |
| 4 | Backfill one-shot au boot : pour chaque player DB, si `spartan_identity` vide → copy depuis `career_progression` ARG_MAX | 30 min |
| 5 | **Supprimer** `internal/scheduler/customization_refresh.go` + son tick + tests associés | 10 min |
| 6 | `go test ./... && go vet ./...`, smoke test 4 joueurs, thought_log | 30 min |

### Grille plan-review appliquée à la version révisée

- **Architecture Go** : nouveau repo dans `platform/duckdb/spartan_identity_repo.go`, lecture/écriture via interface étroite (`Load(ctx, xuid)`, `Upsert(ctx, xuid, data, status)`). Service `CareerLiveService` continue d'orchestrer. Pas de SQL dans le service. ✓
- **Multi-titres** : table créée par migration au boot via `pdb.Player.Path()` (déjà PathResolver). Pas de hardcoding `halo_infinite`. ✓
- **Adapters** : ne touche pas aux adapters canonical (c'est de la persistence pure). ✓
- **Tests** : tests `:memory:` pour repo + tests service mock pour le UPSERT dans `kickoffBackgroundRefresh`. ✓
- **Logging** : `slog.InfoContext(ctx, "spartan_identity: upsert ok", "xuid", xuid, "status", status)`. ✓
- **Frontend** : zéro changement (contrat API JSON `spartan_identity` inchangé). ✓
- **Livraison** : 6 phases livrables séparément, thought_log à chaque phase. ✓

### Risques résiduels identifiés par la revue

1. **Joueur visite sa home pour la première fois sans aucune row `spartan_identity`** → fallback `career_progression` ARG_MAX en Phase 3 couvre. Phase 4 (backfill au boot) garantit qu'au démarrage serveur, tous les joueurs avec historique ont déjà une row.
2. **Cas Madina97294 et XxDaemonGamerxX (tokens absents)** → `last_attempt_status` = `'auth_missing'`. La home affiche le placeholder ou la dernière valeur DB connue. À documenter dans le log Phase 6.
3. **Race condition UPSERT concurrent** : 2 requêtes home simultanées du même joueur → 2 UPSERT en parallèle. DuckDB gère le `INSERT OR REPLACE` atomiquement. À vérifier en test (`TestSpartanIdentityRepo_ConcurrentUpsert`).

---

## 12. Bug parallèle suspecté — token mismatch dans le live fetch

### Symptôme observé

JGtm affiche rang=179 ("Général 2 Platine") alors que Halowaypoint montre
plus. Logs serveur : `career_live: new snapshot inserted` aujourd'hui
13:58 avec rank=179. Hier 24/05 15:45 : rank=183 (avant ART bug + restore).
Aucune erreur auth dans les 120 warnings du log.

### Hypothèse

Le live fetch reçoit silencieusement des données stale parce que le
**token utilisé ne correspond pas au xuid queried**. L'API Halo Economy
`/hi/careerranks/careerRank1?players=xuid(X)` peut renvoyer les données
du token holder (ou des données figées) si les deux ne matchent pas.

Cette hypothèse est cohérente avec :
- Le commit `03322560` du 24/05 01:00 : "fix(auth): E.v1 legacy store ne
  s'attribue qu'à UN seul joueur" — qui explicitement note "mismatch
  xuid/token, erreurs API potentielles ou PIRE (écrasement de données)".
- Le fait que seuls **Chocoboflor et JGtm** ont des tokens valides
  aujourd'hui (Madina expirés, XxDaemon jamais eu). Si JGtm tombe sur le
  RT legacy au lieu de son env token, on a un mismatch silencieux.

### Action A (avant tout refactor) — diagnostic

Ajouter dans `fetchProgressCached` (et `fetchCustomizationCached`) un
log qui :
1. Décode le JWT spartan token pour extraire le xuid du token holder
2. Log : `token_xuid` vs `query_xuid` et `mismatch` (bool)

Si mismatch confirmé sur les logs après une visite home JGtm → fix dans
`enrichWithHaloTokens` pour refuser les tokens de session qui ne
correspondent pas au `pdb.XUID` visité, et forcer un refresh depuis
le bon store.

### Fix proposé (Phase 12.2, si A confirme)

```go
// internal/api/registry.go
func (r *ServiceRegistry) enrichWithHaloTokens(ctx context.Context, pdb *duckdb.PlayerDB) context.Context {
    existing := ctxkeys.HaloTokens(ctx)
    existingXUID := ctxkeys.HaloXUID(ctx)
    // STRICT : les tokens de session HTTP ne sont réutilisés QUE si le
    // xuid associé correspond au pdb visité. Sinon, on force un refresh
    // depuis le store du bon joueur — sinon l'API Halo renvoie les
    // données du token holder au lieu de celles du xuid queried.
    if existing != nil && existingXUID == pdb.XUID {
        return ctx
    }
    // ... reste inchangé (cached / refreshTokensFromDB pour pdb.XUID)
}
```

### Critère de complétion §12

- Log diagnostic ajouté + rebuild server
- Visite home JGtm → log indique `token_xuid == query_xuid` OU mismatch
- Si mismatch : fix `enrichWithHaloTokens` appliqué + retest
- Si pas de mismatch : on a éliminé cette hypothèse, on creuse autre chose
  (probable cache server-side Halo)
- Thought log avec le verdict
