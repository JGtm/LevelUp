# Plan — Correction sémantique progression des citations (per-match)

> Date : 2026-05-21
> Branche cible : à créer depuis `main` (proposition : `fix/citations-progression-semantic`)
> Origine : bug observé sur match `b8c1b220-5ef4-4dee-9e92-77d3ff55d6d3` — composites "Maîtrise en …" affichés alors qu'aucun enfant n'a été masterisé.

---

## 0. Synthèse exécutive

Le calcul des citations per-match est **sémantiquement faux** dans la version Go actuelle :

1. Le moteur principal ([`citations_composite.go:68`](../../apps/go-api/internal/analysis/citations_composite.go#L68)) compare la **valeur per-match d'un enfant** (ex: 5 kills BR75) au **seuil cumulatif `max(tier_targets)`** (ex: 500). La comparaison ne fire jamais ⇒ aucun composite n'est calculé par le sync autonome.
2. Pour combler ce trou, un mode CLI séparé `RunBackfillCompositeOnlyCitations` ([`citations_backfill.go:140`](../../apps/go-api/internal/sync/citations_backfill.go#L140)) applique une autre logique (`val > 0`) qui transforme tout match utilisant au moins une arme d'une famille en "progression composite" — c'est **encore faux** sémantiquement : ça compte des usages, pas des maîtrises.
3. Les **leaves déjà masterisées** continuent d'écrire des deltas dans chaque nouveau match (over-incrémentation cumulée).

**Sémantique cible** (rappel utilisateur) :

> Le moteur ne compte pas la valeur d'un enfant. Il vérifie si une citation enfant **est maîtrisée** (dernier palier atteint). Dans le match courant, on progresse d'un palier sur le composite parent si un enfant a été nouvellement maîtrisé. Les méta‑citations suivent la même règle en cascade. Dans le match, on n'affiche/comptabilise **que les citations qui ont progressé** durant le match — atteinte d'un palier, maîtrise complète, ou simple incrément. Une citation déjà maîtrisée n'est plus comptée (la composite l'a déjà absorbée).

**Conséquence** : le moteur doit raisonner sur le **cumulé pre-match** et le **cumulé post-match** des feuilles pour décider des deltas à écrire en `match_citations`, et propager les transitions de palier sur les composites/métas par cascade. Le mode backfill composite-only doit disparaître — le sync devient pleinement autonome.

---

## 1. Bug actuel — diagnostic détaillé

### 1.1 Chaîne de calcul actuelle (sync autonome)

```
sync.BackfillMatchCitations(matchIDs)
  ↓
  pour chaque match :
    buildCitationContext  → medals/stats/weapon_kills/awards/events PER-MATCH
    ↓
    analysis.ComputeFullMatchCitations(ctx, mappings)
      ↓
      dispatchFull(m, ctx)             // leaves → val per-match
      ↓
      computeCompositeCitations(totals, mappings)  // ← BUG ICI
        pour chaque composite :
          pour chaque enfant :
            val = totals[child]        // valeur PER-MATCH (ex: 5 kills BR75)
            si val >= max(tier_targets)  // seuil CUMULATIF (ex: 500)
              count++                  // ne fire jamais
          totals[composite] += count   // toujours 0
    ↓
    writeCitations(matchID, deltas)    // composites = 0 → non écrits
```

Fichiers concernés :
- [`apps/go-api/internal/analysis/citations_engine.go`](../../apps/go-api/internal/analysis/citations_engine.go) — `ComputeFullMatchCitations`, `dispatchFull`.
- [`apps/go-api/internal/analysis/citations_composite.go`](../../apps/go-api/internal/analysis/citations_composite.go) — `computeCompositeCitations`, `compositeChildMasterised`, `ApplyCompositeCitationsPerMatch`.
- [`apps/go-api/internal/sync/citations.go`](../../apps/go-api/internal/sync/citations.go) — `BackfillMatchCitations`, `buildCitationContext`.
- [`apps/go-api/internal/sync/citations_backfill.go`](../../apps/go-api/internal/sync/citations_backfill.go) — `RunBackfillCompositeOnlyCitations`.

### 1.2 Chaîne de calcul actuelle (mode backfill composite-only)

```
CLI : levelup backfill --citations-composite-only
  ↓
  SyncEngine.RunBackfillCompositeOnlyCitations()
    ↓
    loadNonCompositeCitationsByMatch  → totals leaves par match (val per-match)
    ↓
    pour chaque match :
      analysis.ApplyCompositeCitationsPerMatch(totals, mappings)
        ↓
        applyCompositesPass (passes itératives, max 5)
          pour chaque composite :
            count = nb d'enfants où totals[child] > 0   // ← BUG ICI
            si count > 0 : totals[composite] = count
    ↓
    writeCitations(matchID, composites > 0)
```

C'est ce chemin qui a produit les `human_weapons_mastery`, `paria_weapons_mastery`, etc. visibles sur `b8c1b220-…`. Il faut le supprimer.

### 1.3 Bug secondaire — leaves non capées

Aucun mécanisme actuel n'empêche d'écrire `value` per-match pour une leaf déjà maîtrisée cumulativement. Exemple : si BR75 cumulé = 500 (max tier), un nouveau match avec 10 kills BR75 va écrire `value=10` ⇒ cumulé devient 510 ⇒ faux signal aux composites parents si un jour ils utilisent le cumulé.

Confirmer le comportement actuel : il n'y a aucune dédup ni cap dans `BackfillMatchCitations`. À fixer en même temps.

### 1.4 Bug tertiaire — pas d'ordre chronologique garanti

`BackfillMatchCitations(matchIDs []string)` itère dans l'ordre passé par l'appelant. Si l'ordre n'est pas chronologique (ascendant par `start_time` ou `end_time`), le calcul pre/post cumulé sera incorrect en mode rejeu / import OpenSpartan. À fixer : l'appelant doit garantir l'ordre, ou la fonction trie en interne.

Appelants à vérifier :
- [`apps/go-api/internal/service/openspartan_post_import_service.go:207`](../../apps/go-api/internal/service/openspartan_post_import_service.go#L207)
- [`apps/go-api/internal/sync/citations_backfill.go:271`](../../apps/go-api/internal/sync/citations_backfill.go#L271)

---

## 2. Sémantique cible

### 2.1 Règles invariantes

| # | Règle | Conséquence concrète |
|---|---|---|
| R1 | Une **leaf** (medal/stat/weapon_stat/pve_stat/award/custom) écrit un delta per-match **uniquement si** elle progresse réellement dans le match | `value > 0` ET `cumul_pre < max(tier_targets)` |
| R2 | Le delta écrit est **capé** au dernier palier | `value = min(val_brut_match, max(tier_targets) − cumul_pre)` |
| R3 | Une leaf **déjà maîtrisée** (cumul_pre ≥ max(tier_targets)) n'écrit **rien** | Évite over-incrémentation cumulée et faux signal aux composites |
| R4 | Un **composite** progresse de **+1 par enfant nouvellement maîtrisé** dans ce match | Condition par enfant : `cumul_pre_enfant < max_tier_enfant ∧ cumul_post_enfant ≥ max_tier_enfant` |
| R5 | Un composite n'écrit rien si aucun enfant n'a traversé son palier final dans ce match | `value > 0` sinon ne pas insérer |
| R6 | Une **méta** (composite de composites) suit R4–R5 en cascade | Passe itérative jusqu'à stabilisation (déjà implémentée mais avec la mauvaise condition) |
| R7 | Pour les composites/métas sans `tier_targets`, le composite parent **se maîtrise** quand tous ses enfants sont maîtrisés | Comportement par défaut si `tier_targets` vide |
| R8 | Les composites/métas ont aussi leurs propres `tier_targets` éventuels et sont eux-mêmes capés | Même logique R2 sur le composite parent |
| R9 | Match-view UI : on affiche uniquement les lignes présentes dans `match_citations` pour ce `match_id` | Pas de filtrage côté frontend ; le stockage est déjà la source de vérité |

### 2.2 Conséquences sur le stockage

- `match_citations(match_id, citation_name_norm, value)` reste un **journal d'événements** : chaque ligne = "progression réelle survenue dans ce match". `SUM(value) GROUP BY citation_name_norm` = cumul total cohérent par citation.
- Pas besoin de table séparée pour les composites — ils sont stockés au même niveau que les leaves.
- L'idempotence est garantie par `ON CONFLICT (match_id, citation_name_norm) DO NOTHING` (déjà en place). Le **recalcul** demande donc un `DELETE` préalable, ce qui est OK puisque c'est ce que fait déjà le mode `--citations` en re-sync forcé.

### 2.3 Conséquences sur l'agrégation cumulative

- `CitationsPage` lit `SUM(value)` par citation. La règle R2 garantit que ce cumul n'excède jamais `max(tier_targets)`. Plus besoin de `LEAST(SUM, max)` côté requête (à vérifier dans les queries existantes).
- La progression composite = `SUM(value)` cohérent avec le nb d'enfants maîtrisés — plus de comptage gonflé par les workarounds backfill.

---

## 3. Plan d'implémentation

### Phase 1 — Refactor du moteur `analysis/citations`

**Objectif** : `ComputeFullMatchCitations` prend en entrée le cumulé pre-match et produit des deltas correctement capés / transitionnels.

**Fichiers modifiés** :
- `apps/go-api/internal/analysis/citations_engine.go`
- `apps/go-api/internal/analysis/citations_composite.go`
- `apps/go-api/internal/analysis/citations_test.go`
- `apps/go-api/internal/analysis/citations_composite_test.go`

**Nouveau contrat** :

```go
// CitationProgressInput agrège l'état nécessaire pour calculer la progression
// d'un match : contexte stats + cumulé pre-match.
type CitationProgressInput struct {
    Ctx        domain.CitationContext   // stats/medals/awards/events DU match
    CumulPre   map[string]int           // citation_name_norm → cumul AVANT ce match
}

// ComputeFullMatchCitations applique R1–R8.
// Retourne uniquement les deltas correspondant à une progression effective.
func ComputeFullMatchCitations(
    in CitationProgressInput,
    mappings []domain.CitationFullMapping,
) []domain.CitationMatchDelta
```

**Algorithme** :

```
1. Phase A — Leaves (mapping_type ≠ "composite") :
   - Pour chaque leaf m :
     - raw = dispatchFull(m, in.Ctx)
     - max = max(tier_targets) ou +inf si tier_targets vide
     - capRoom = max - in.CumulPre[m.NameNorm]
     - if capRoom <= 0 : skip (R3)
     - delta = min(raw, capRoom)        (R1, R2)
     - if delta > 0 : leafDeltas[m.NameNorm] = delta
2. Phase B — État post-match des leaves :
   - cumulPost = copie(cumulPre)
   - cumulPost[name] += delta pour chaque leaf
3. Phase C — Composites & métas par passes itératives :
   - jusqu'à stabilisation (max 5 passes) :
     - pour chaque composite m :
       - children = parseCompositeChildren(m)
       - newlyMastered = 0
         pour chaque child :
           maxChild = max(child.tier_targets) ou défini par règle R7
           if cumulPre[child] < maxChild AND cumulPost[child] >= maxChild :
             newlyMastered++  (R4)
       - if newlyMastered == 0 : continue
       - max_composite = max(m.tier_targets) ou len(children) si R7
       - capRoom_composite = max_composite - cumulPre[m.NameNorm]
       - delta_composite = min(newlyMastered, capRoom_composite)
       - if delta_composite > 0 :
         - compositeDeltas[m.NameNorm] = delta_composite
         - cumulPost[m.NameNorm] += delta_composite  (alimente la passe suivante pour les métas)
4. Retourne leafDeltas ∪ compositeDeltas filtrés > 0
```

**Helpers à introduire** :
- `parseTierTargetsCSV(*string) []int` — déjà partiellement présent.
- `maxTier(tiers []int, fallback int) int`.
- `isChildNewlyMastered(pre, post, maxTier int) bool`.

**Suppressions** :
- `ApplyCompositeCitationsPerMatch` (workaround val>0)
- `applyCompositesPass` (lié)
- `compositeChildMasterised` (remplacé par `isChildNewlyMastered`)

### Phase 2 — Adapter `sync/citations.go`

**Fichiers modifiés** :
- `apps/go-api/internal/sync/citations.go`
- `apps/go-api/internal/sync/citations_backfill.go` (nettoyage)

**Changements `BackfillMatchCitations`** :

```go
func BackfillMatchCitations(
    ctx context.Context,
    metadataDB, sharedDB, playerDB *sql.DB,
    xuid string,
    matchIDs []string,                 // ordre garanti chronologique par l'appelant
) error {
    // 1. Charger mappings + weapon_labels (déjà fait)
    // 2. Charger cumulé pre depuis match_citations en EXCLUANT les matchIDs traités
    //    (pour rester idempotent en cas de re-sync forcé)
    cumulPre, err := loadCumulExcluding(ctx, playerDB, matchIDs)
    // 3. Trier matchIDs par start_time ASC (depuis shared.match_registry)
    matchIDs, err = sortMatchIDsChrono(ctx, sharedDB, matchIDs)
    // 4. Pour chaque match (ordre chrono) :
    for _, matchID := range matchIDs {
        citCtx := buildCitationContext(...)
        in := analysis.CitationProgressInput{Ctx: citCtx, CumulPre: cumulPre}
        deltas := analysis.ComputeFullMatchCitations(in, mappings)
        // 5. AVANT writeCitations : DELETE FROM match_citations WHERE match_id = ?
        //    pour garantir le recalcul propre (utile en re-sync forcé)
        if err := deleteCitationsForMatch(ctx, playerDB, matchID); err != nil { ... }
        if err := writeCitations(ctx, playerDB, matchID, deltas); err != nil { ... }
        // 6. Updater cumulPre avec les deltas qu'on vient d'écrire
        for _, d := range deltas {
            cumulPre[d.NameNorm] += d.Value
        }
    }
}
```

**Helpers à introduire** :
- `loadCumulExcluding(ctx, playerDB, excludedMatchIDs) (map[string]int, error)` — `SELECT name_norm, SUM(value) FROM match_citations WHERE match_id NOT IN (…) GROUP BY name_norm`.
- `sortMatchIDsChrono(ctx, sharedDB, matchIDs) ([]string, error)` — `SELECT match_id FROM match_registry WHERE match_id IN (…) ORDER BY start_time ASC`.
- `deleteCitationsForMatch(ctx, playerDB, matchID) error`.

**Refactor (pas suppression)** :
- `SyncEngine.RunBackfillCompositeOnlyCitations` ([`citations_backfill.go:140`](../../apps/go-api/internal/sync/citations_backfill.go#L140)) est **conservée** comme outil de rescue (filet de sécurité en cas de dérive des composites sans toucher aux leaves). Mais elle doit être **refactorisée pour réutiliser le moteur corrigé** :
  1. Charger `cumulPre` depuis `match_citations` (exclure les composites).
  2. Itérer les matchs en ordre chrono.
  3. Reconstruire `cumulPost` au fil de l'eau et appliquer la logique de transition (R4).
  4. N'écrire que les lignes composites (DELETE composites pour le matchID concerné avant INSERT).
- Helpers à supprimer (devenus inutiles) : `ApplyCompositeCitationsPerMatch`, `applyCompositesPass`, `compositeChildMasterised`, `loadNonCompositeCitationsByMatch`, `buildCompositeNameSet` (tous remplacés par le moteur unifié).
- La CLI associée dans [`cmd/levelup/cmd_backfill.go:762`](../../apps/go-api/cmd/levelup/cmd_backfill.go#L762) reste, mais sa doc/help précise qu'elle est un outil d'urgence — le sync autonome fait le boulot en temps normal.

### Phase 3 — Migration one-shot des données existantes

**Objectif** : purger toutes les lignes composite/méta actuelles (issues du workaround) et recalculer correctement sur l'historique du joueur.

**Approche** : un sous-commande CLI dédié, idempotent.

**Fichier** : `apps/go-api/cmd/levelup/cmd_backfill.go` — ajouter un mode `--citations-recompute-all`.

```go
// 1. DELETE FROM match_citations  (purge totale — décision Q1)
// 2. Lister tous les match_id du joueur depuis shared.match_registry
//    (BackfillMatchCitations triera en interne par start_time ASC — décision Q8)
// 3. Lancer BackfillMatchCitations(ctx, …, matchIDs)
// 4. Vérif post-compute obligatoire (décision Q4) : exécuter automatiquement
//    le bloc SQL §4.3 et imprimer le résumé ; échouer si une invariante est violée.
```

**Décision à prendre** :
- **Option A** : ne purger que les composites/métas. Les leaves existantes restent (potentiellement over-incrémentées, mais moindre impact UI).
- **Option B** (recommandée) : purger toutes les lignes `match_citations` et recalculer tout. Plus propre, plus long. Sur un joueur moyen (~2000 matchs) c'est encore quelques secondes.

→ Recommander B. À chiffrer lors de l'implémentation.

### Phase 4 — Cleanup & docs

**Fichiers modifiés** :
- `CLAUDE.md` (optionnel) — pas de mention citations.
- `docs/ARCHITECTURE_V6.md` — ajouter une section "Sémantique progression citations" si pas présente.
- `.ai/thought_log.md` — entrée obligatoire avant commit.
- `.ai/V7/PLAN_CITATIONS_GO_PORTAGE.md` — ajouter un encart "Amendement 2026-05-21" pointant vers ce plan.

**Suppressions** :
- Tests obsolètes liés à `ApplyCompositeCitationsPerMatch` ([`apps/go-api/internal/sync/citations_backfill.go`](../../apps/go-api/internal/sync/citations_backfill.go) test associé).
- Tests `TestComputeFullMatchCitations_CompositeSkipped` à mettre à jour (la sémantique change).

### Phase 5 — Tests

**Tests unitaires `analysis/`** :

| # | Scénario | Setup | Attendu |
|---|---|---|---|
| T1 | Leaf en progression simple, pas de tier | `cumulPre={br75:50}` ; match: `weapon_kills:BR75=5` ; tier=`25,50,100,200,500` | delta = 5 |
| T2 | Leaf cap par max tier | `cumulPre={br75:498}` ; match: 10 kills | delta = 2 (capé à 500-498) |
| T3 | Leaf déjà maîtrisée | `cumulPre={br75:500}` ; match: 5 kills | delta = 0 (R3) |
| T4 | Composite — enfant traverse | `cumulPre={br75:495}` ; match: 10 kills BR75 → cumulPost=500 ; composite `human_weapons_mastery` enfant=[br75,…] | delta br75=5 (capé), delta human_weapons_mastery=+1 |
| T5 | Composite — enfant déjà maîtrisé pre, pas de transition | `cumulPre={br75:500}` ; match: 10 kills BR75 | delta br75=0, composite=0 |
| T6 | Composite — 2 enfants traversent dans le même match | `cumulPre={br75:495, ma40:498}` ; match: BR75=10, MA40=10 | composite human_weapons_mastery=+2 |
| T7 | Méta — cascade | `cumulPre={…toutes UNSC à max-1}` ; match maîtrise les 10 UNSC + 1 Paria | composite UNSC=+10 (capé à son max), méta `all_weapons_mastery` = +1 si UNSC complet |
| T8 | Composite sans tier_targets (R7) | composite `human_weapons_mastery` sans tier_targets ; 10 enfants ; 1 enfant traverse | delta composite = 1, max_composite = len(children) = 10 |
| T9 | Re-calcul idempotent | Appel x2 sur même `cumulPre` et même match | mêmes deltas |
| T10 | Order independence | Mappings shuffles aléatoirement | mêmes deltas |

**Tests d'intégration `sync/`** :
- Reprendre `TestPipelineFixture_Citations` ([`sync_pipeline_fixture_test.go:831`](../../apps/go-api/internal/sync/sync_pipeline_fixture_test.go#L831)) et adapter aux nouvelles attentes.
- Nouveau test : 3 matchs chronologiques où le 2ème fait traverser un palier ⇒ vérifier que seul le 2ème match a la ligne composite.

**Tests E2E `api/handlers/`** :
- Vérifier que la `CitationsPage` (`GET /api/v1/players/{slug}/pages/citations`) reste cohérente après recalcul.

---

## 4. Migration des données existantes — checklist opérationnelle

### 4.1 Procédure

1. **Backup** : copier `data/players/{gamertag}/stats.duckdb` vers `data/players/{gamertag}/stats.duckdb.bak.YYYYMMDD`.
2. **Lancer la CLI de recompute** : `levelup backfill --citations-recompute-all --player {gamertag}`.
3. Le recompute exécute **automatiquement** les vérifications §4.3 à la fin et imprime un résumé. Il échoue (exit ≠ 0) si une invariante est violée.

Pour un déploiement multi-joueur : itérer la commande sur tous les `data/players/*/`.

### 4.2 Logique de la commande

```
1. DELETE FROM match_citations  (purge totale, Q1)
2. SELECT match_id FROM v_player_matches WHERE xuid = ?  (liste brute)
3. BackfillMatchCitations(ctx, …, matchIDs)  (tri chrono interne, Q8)
4. Exécuter §4.3 ; en cas d'échec : log + exit 1
```

### 4.3 Vérifications post-compute (obligatoires, Q4)

À implémenter directement dans la commande `--citations-recompute-all` :

| # | Requête | Invariante |
|---|---|---|
| V1 | `SELECT citation_name_norm, SUM(value) AS cumul FROM match_citations GROUP BY 1` joiné sur `citation_mappings` | `cumul ≤ max(tier_targets)` pour toute leaf avec tier_targets ; `cumul ≤ len(composite_children)` pour tout composite (R7) |
| V2 | `SELECT match_id, citation_name_norm, COUNT(*) FROM match_citations GROUP BY 1,2 HAVING COUNT(*) > 1` | Doit retourner 0 ligne (PK respectée) |
| V3 | `SELECT COUNT(*) FROM match_citations WHERE value <= 0` | Doit retourner 0 (jamais de delta nul/négatif persisté) |
| V4 | Pour chaque composite C avec enfants \[c1…cN\] : `SUM(value pour C) ≤ Σᵢ ⌊SUM(value pour cᵢ) / max_tier(cᵢ)⌋` (nb d'enfants maîtrisés) | Le composite ne progresse jamais plus que le nb d'enfants ayant traversé leur palier final |
| V5 | `SELECT match_id, citation_name_norm FROM match_citations mc JOIN match_registry r USING (match_id) WHERE r.start_time IS NULL` | Doit retourner 0 (intégrité référentielle) |

Toute violation → log détaillé + exit 1 + suggestion de restore depuis le backup §4.1.

### 4.4 Validation manuelle UI

- Ouvrir le match `b8c1b220-…` dans la match-view ⇒ plus de "Maîtrise des armes UNSC" parasites.
- Ouvrir la page Citations ⇒ progression composite cohérente avec le nb d'enfants maîtrisés affiché.

---

## 5. Risques & questions ouvertes

### 5.1 Risques

| # | Risque | Mitigation |
|---|---|---|
| K1 | Perf : lecture cumul + tri chrono à chaque batch | Acceptable : 1 requête `SUM GROUP BY` + 1 requête `ORDER BY` au début, pas dans la boucle |
| K2 | Migration one-shot longue sur gros historique | Mesurer sur un joueur réel (~2000 matchs). Si > 30s, paginer. |
| K3 | Régression sur la lecture cumulé par la `CitationsPage` (cap déjà appliqué ailleurs ?) | Auditer les queries existantes dans `home_repo_medals_citations.go` et `queries_home_citations.go` pour vérifier qu'aucun `LEAST(SUM, max)` n'était requis |
| K4 | Comportement OpenSpartan post-import : ordre chronologique ? | Vérifier `openspartan_post_import_service.go:207` — ajouter tri si absent |
| K5 | Citations custom (`custom_function`) — leur valeur per-match est-elle bornée ? | Auditer `citations_custom.go` : si certaines retournent > 1, R2 s'applique normalement |

### 5.2 Décisions figées (2026-05-21)

| # | Question | Décision |
|---|---|---|
| Q1 | Migration : purge totale ou composites seulement ? | **Purge totale** — `DELETE FROM match_citations` avant recompute. Ça coûte rien et garantit la propreté (élimine aussi les éventuelles over-incrémentations historiques sur les leaves). |
| Q2 | R7 — défaut composite sans tier_targets : `max = len(children)` ? | **Oui**. Les 7 composites du seed sont tous sans `tier_targets` → R7 est la règle universelle. |
| Q3 | Stockage composites : table ou VIEW ? | **Table** — `match_citations` reste la source de vérité unique, leaves + composites stockés au même niveau. |
| Q4 | Mode `--dry-run` sur le recompute ? | **Non**. Pas de dry-run, mais **vérif post-compute obligatoire** : le recompute exécute automatiquement les requêtes de validation (§4.3) et échoue si une invariante est violée. |
| Q5 | ~~Leaves sans tier_targets~~ | ✓ Résolu — seed aligné (`brute_slayer`, `skimmer_slayer` → `"10,20,30,50,100"`). Re-seed UPSERT au prochain boot. |
| Q6 | `tier_targets` des leaves `award` cohérents avec la fréquence ? | **On garde les seuils actuels**. Recalibration éventuelle en ticket séparé après recompute (on aura les cumuls réels). |
| Q7 | Découpage commits | **6 commits sur branche unique** `fix/citations-progression-semantic` (§6). |
| Q8 | Tri chrono : moteur ou appelant ? | **Moteur** — `BackfillMatchCitations` trie en interne via `ORDER BY start_time ASC` sur `match_registry`. Tous les callers protégés automatiquement. |
| Q9 | Suppression de `RunBackfillCompositeOnlyCitations` ? | **Non, refactor**. On la garde comme outil d'urgence/rescue (refactorisée pour utiliser le moteur corrigé), mais elle n'est plus un pré-requis du fonctionnement normal — le sync autonome fait le boulot. |

---

## 6. Découpage en commits proposé

Branche unique `fix/citations-progression-semantic`, commits chronologiques :

1. `refactor(citations): introduce CitationProgressInput + new ComputeFullMatchCitations contract` — Phase 1 sans changer les callers (compat shim si besoin).
2. `feat(citations): cap leaves and propagate composite/meta transitions per-match` — Phase 1 algo correct + tests unitaires T1–T10.
3. `feat(sync): wire cumulPre + chrono order in BackfillMatchCitations` — Phase 2 callers adaptés + tests d'intégration.
4. `refactor(sync): rewrite RunBackfillCompositeOnlyCitations on top of fixed engine (rescue tool)` — Phase 2 refactor (Q9).
5. `feat(cli): add --citations-recompute-all with mandatory post-compute checks` — Phase 3 + §4.3.
6. `docs(citations): document new semantic + amend PLAN_CITATIONS_GO_PORTAGE.md` — Phase 4.

---

## 7. Critères d'acceptation

- [ ] Sur `b8c1b220-…`, plus aucun composite/méta n'apparaît, sauf si un enfant a effectivement traversé son max tier dans ce match précis.
- [ ] `SELECT SUM(value) FROM match_citations WHERE citation_name_norm = X` ≤ `max(tier_targets[X])` pour toute citation X avec tier_targets non vide.
- [ ] Plus aucun appel à `ApplyCompositeCitationsPerMatch` (code supprimé). `RunBackfillCompositeOnlyCitations` existe encore mais utilise le nouveau moteur (rescue tool).
- [ ] `BackfillMatchCitations` est appelée avec des matchs triés chronologiquement (vérifié dans les 2 callers).
- [ ] Couverture analysis/citations ≥ niveau pré-refacto (baseline `coverage_pre_migration.txt`).
- [ ] Aucune régression sur les pages Citations / Career / Home (test E2E + revue visuelle).
- [ ] Entrée `.ai/thought_log.md` documentant la décision sémantique.
