# Plan V2 — Spartan Identity + Rang Carrière + Barre XP

**Date** : 2026-05-26
**Branche** : `fix/art-eradication-and-home-resilience`
**Statut** : EN ATTENTE DE VALIDATION
**Supersede** : `.ai/PLAN_SPARTAN_IDENTITY_REFACTOR.md` (V1, abandonné)

---

## 0. Pourquoi V2 ?

V1 partait sur une table dédiée `spartan_identity` avec UPSERT 1-ligne. Mise en
place hier soir, ce refactor a **empiré le système** : la home flicker
(bannière apparaît/disparaît selon la visite), le rang carrière change
arbitrairement, la barre XP saute. La cause racine est le UPSERT qui écrase
des valeurs connues par des valeurs vides quand l'API rend partiellement.

V2 prend en compte tes 3 corrections d'architecture :

1. **Append-only avec timestamp** (pas UPSERT 1-ligne) — propre, auditable, garde l'historique
2. **Per-field non-empty wins** — chaque champ est indépendant ; l'API qui rend vide sur un champ ne touche pas la valeur connue
3. **Stale-while-revalidate** — la home affiche immédiatement la dernière valeur DB connue non-vide, le live tourne en background, met à jour la DB si nouveau, et le front re-render

Cette logique existait DÉJÀ dans le code original (`career_progression` +
`mergeCareerRow`). Mon refactor §11 l'a cassée. V2 la **restaure proprement**
et l'étend aux champs rang/XP qui souffrent du même bug.

---

## 1. État des lieux factuel (mesuré aujourd'hui)

### Côté DB (avant refactor §11, mesure d'hier matin)

| Joueur | total rows | banner stocké | emblem stocké | dernière bannière |
|--------|-----------:|--------------:|--------------:|-------------------|
| JGtm | 86 | 1 / 86 | 13 / 86 | aujourd'hui |
| Madina97294 | 65 | 0 / 65 | 9 / 65 | jamais |
| XxDaemonGamerxX | 0 | 0 | 0 | jamais |
| Chocoboflor | DB lockée | — | — | — |

### Côté tokens (logs pool.log dernières 24h)

| Joueur | source | Exchange Halo | Customization endpoint |
|--------|--------|---|---|
| JGtm | duckdb_oauth | ✅ OK | ✅ rend données complètes |
| Chocoboflor | duckdb_oauth | ✅ OK | ✅ (présumé OK) |
| Madina97294 | env_oauth (KO invalid_grant) + duckdb_oauth (OK) | ✅ via duckdb | ❌ aucune trace récente d'appel |
| XxDaemonGamerxX | duckdb_oauth | ✅ OK | ❌ 5× 403 sur 24h |

### Symptômes post-refactor V1 (signalés par toi)

- Bannière apparaît/disparaît d'une visite à l'autre (toutes joueurs)
- Rang carrière change : JGtm voit 179 alors que Halo Waypoint affiche plus
- Barre XP saute (parfois remplie, parfois vide)
- Adornment et nameplate vus puis disparus
- Fallback "dernière valeur connue" inopérant

### Hypothèses confirmées par logs

1. **§12 — token mismatch suspect** : non confirmé. JGtm a son propre token valide. Le rang stale vient probablement du cache Halo serveur OU d'un autre bug
2. **403 XxDaemonGamer** : token valide pour les autres endpoints, seul customization 403 → 99% c'est sa privacy Halo Waypoint
3. **Madina** : aucun appel customization dans les logs récents → le live fetch n'est pas déclenché pour elle (à investiguer)
4. **Flicker tous champs** : pollution par UPSERT du refactor §11 (cause identifiée, fix planifié ci-dessous)

---

## 2. Principes architecturaux (les 3 règles d'or)

### Règle 1 — Append-only avec timestamp

```sql
-- Table conservée telle quelle (career_progression) ou nouvelle, peu importe.
-- Ce qui compte : 1 ligne = 1 sync, jamais d'écrasement.
CREATE TABLE career_progression (
  recorded_at TIMESTAMP,
  xuid VARCHAR,
  rank INTEGER NULLABLE,             -- NULL si pas màj
  current_xp INTEGER NULLABLE,
  xp_for_next_rank INTEGER NULLABLE,
  is_max_rank BOOLEAN NULLABLE,
  rank_name VARCHAR NULLABLE,
  rank_tier VARCHAR NULLABLE,
  spartan_id VARCHAR NULLABLE,
  banner_image_url VARCHAR NULLABLE,
  emblem_image_url VARCHAR NULLABLE,
  backdrop_image_url VARCHAR NULLABLE,
  adornment_path VARCHAR NULLABLE
)
```

L'INSERT n'écrit qu'**une nouvelle ligne**. Aucune écriture ne touche
les lignes existantes. Aucun risque de "perte" sur écrasement.

### Règle 2 — Per-field non-empty wins

À l'écriture : chaque champ est **set indépendamment**. Si l'API a rendu
banner mais pas emblem, la nouvelle ligne aura `banner_image_url = "url"` et
`emblem_image_url = NULL`.

À la lecture : on consulte chaque champ via :
```sql
ARG_MAX(banner_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url), '') IS NOT NULL)
```

Ce filter garantit qu'on récupère la **dernière valeur non-vide par champ**,
indépendamment des autres champs et de l'ordre des lignes. Une ligne polluée
(tous NULL ou empty) n'affecte la lecture d'aucun champ.

**Distinction critique** : "API rend 0 explicite" vs "API n'a rien rendu" :
- Si `progress == nil` (API n'a pas répondu) → on n'écrit RIEN
- Si `progress != nil` avec `CurrentXP == 0` → on écrit `current_xp = 0`
  (vraie valeur du joueur en début de palier)
- Si `progress != nil` avec `BannerImageURL == ""` → on écrit `NULL` (pas vide string)

### Règle 3 — Stale-while-revalidate, front-first

Flow de chaque visite home :
1. Lecture DB synchrone (<5ms) : `SELECT ARG_MAX FILTER WHERE NOT NULL` par champ → identité complète des dernières valeurs connues, par champ indépendamment
2. Retour IMMÉDIAT au frontend → bannière + rang + XP affichés avec ce que la DB sait
3. Background goroutine (détachée du request context) : appel live Halo
4. Si le live rend des données différentes ET non-vides → INSERT nouvelle ligne
5. Aucun overlay synchrone qui peut écraser ce qu'on a déjà servi
6. La prochaine visite servira les nouvelles valeurs automatiquement (cf. Règle 2)

Le frontend ne voit JAMAIS le flicker parce qu'il reçoit toujours du cohérent
synchrone, et la mise à jour est asynchrone et acquise à la prochaine fetch.

---

## 3. Scope du refactor V2

### Dans le scope

| Concern | Champs | Source live | Triggers d'écriture |
|---------|--------|------------|-------------------|
| **Customisation** | spartan_id, banner_image_url, emblem_image_url, backdrop_image_url | `GetSpartanCustomization` + `ResolveNameplateURL` | visite home (background detached) + scheduler optionnel 6h |
| **Rang carrière** | rank, rank_name, rank_tier, is_max_rank, adornment_path | `GetCareerProgress` + metadata enrichment | visite home (background detached) + match sync |
| **Barre XP** | current_xp, xp_for_next_rank, xp_total | `GetCareerProgress` | visite home (background detached) + match sync |

**Tous écrits dans la même table** via le même helper. Pas 3 chemins, pas 3 tables.

### Hors scope (mais lié — à traiter séparément après V2)

- Bug §12 "JGtm rank stale" : non lié au refactor V2 (cause à investiguer
  séparément : cache serveur Halo, throttle, ou autre)
- Privacy 403 XxDaemonGamerxX : V2 surface le `last_attempt_status = 'forbidden_403'`
  et le frontend affiche le fallback gris. Pas de fix possible côté code (réglage joueur Halo)
- Bug "Madina pas d'appel customization" : V2 inclut un diag explicite pour
  comprendre. Si root cause hors V2, ticket dédié

---

## 4. Migration — comment passer de l'état actuel à V2

### Constat sur l'état actuel

Le refactor §11 a introduit :
- Une table `spartan_identity` (UPSERT 1-ligne) → **à virer**
- Un repo `SpartanIdentityRepo` Load/Upsert → **à virer**
- Un `kickoffBackgroundRefresh` qui UPSERT dans cette nouvelle table → **à virer**
- Un `ApplySpartanIdentityOverlay` qui patch identity avec spartan_identity → **à virer**
- Tous les tests associés → **à virer**

Mais le refactor a aussi gardé en place :
- `CustomizationRefresher` (mon ajout d'avant-hier) → **soit virer, soit restaurer pour écrire dans career_progression via le helper V2**
- Le frontend `FALLBACK_BANNER_URL = '/banner-default.png'` → **à garder** (ultime filet)

### Choix de migration : revert ciblé + audit

Plutôt qu'un revert massif risqué, on procède par chirurgie ciblée :

- Identifier les commits §11 EXACTS qui ont introduit le refactor (3-5 commits)
- Reverter SEULEMENT ces commits (git revert, pas reset)
- Tout le reste (frontend fallback, scheduler, diag_customization_population) reste

Après revert : on est revenu au comportement original `career_progression`
+ `mergeCareerRow` + `Q26cHomeSpartanIdentity`. Ce comportement avait des
bugs (la bannière de Madina, etc.) MAIS PAS le flicker. C'est un état stable
sur lequel construire V2 proprement.

---

## 5. Phases d'implémentation V2

Chaque phase est livrable séparément (commit propre, tests verts). Tu peux
stopper après n'importe quelle phase et le système reste stable.

### Phase 1 — Revert chirurgical du refactor §11 (30 min)

- `git log --oneline` pour lister les commits §11 (3-5 commits identifiés)
- `git revert <hash>` pour chacun (NE PAS faire de reset, garder l'historique propre)
- Build + test
- **Livrable** : système revenu à l'état "stable mais imparfait" (bannière de Madina manque, mais pas de flicker)
- **Risque** : faible (revert atomique, restauration garantie)

### Phase 2 — Audit + fix `mergeCareerRow` (1h)

Audit ligne par ligne de `mergeCareerRow` (career_live_service.go ligne 512+) :

- [ ] Cas "live==nil" → on n'écrit aucune ligne (déjà OK)
- [ ] Cas "live!=nil, champ vide" → on n'écrit PAS ce champ dans la nouvelle ligne (currently buggy : `current_xp=0` est écrit comme valeur)
- [ ] Cas "live!=nil, champ non-vide" → on écrit ce champ
- [ ] Cas "live==nil, dbLast==nil" → return nil, pas d'INSERT

**Fix concret** :

```go
// mergeCareerRow doit retourner (row, fieldsToWrite map[string]bool)
// L'INSERT ne set que les champs présents dans fieldsToWrite.
// Les autres restent NULL dans la nouvelle ligne.
```

**Livrable** : tests unitaires pour les 4 cas + tests pour distinguer `Rank=0` API explicit vs API absent
**Risque** : moyen (touche au cœur du merge logic)

### Phase 3 — INSERT field-aware (45 min)

Modifier `InsertCareerProgressionIfChanged` pour accepter un `map[string]bool` de
champs à set, et n'inclure que ceux-là dans le INSERT (les autres → NULL).

```go
// Avant : INSERT INTO career_progression (rank, current_xp, ...) VALUES (?, ?, ...)
// Après : INSERT INTO career_progression (rank, current_xp) VALUES (?, ?)
//         seulement si rank ET current_xp sont dans le map fieldsToWrite
```

**Livrable** : test `TestInsert_OnlyBanner_LeavesOtherFieldsNull`
**Risque** : faible

### Phase 4 — Stale-while-revalidate strict dans `GetSpartanIdentity` (30 min)

Vérifier que :
- `serveDBFallback(ctx, xuid)` est appelé EN PREMIER et retourne synchrone
- Le live tourne en BACKGROUND uniquement
- Aucun overlay synchrone qui écrase le dbFallback par du vide
- `overlayIdentityFromFallback` est appelé après merge mais avec la règle "non-empty wins" garantie

C'est probablement déjà OK depuis le revert Phase 1, mais on audite explicitement.

**Livrable** : test `TestGetSpartanIdentity_ReturnsDBFirst_LiveDoesNotEraseExisting`
**Risque** : faible

### Phase 5 — Diag pourquoi Madina n'a pas de customization fetch (30 min)

Ajouter un log Info dans `fetchAndMerge` qui dit :
- xuid
- hasAuth
- cache hit/miss (progress, custom)
- needRefresh
- kickoffBackgroundRefresh fired ? (oui/non/raison)

Visite home Madina → lire les logs → comprendre pourquoi le fetch n'est jamais déclenché.

**Livrable** : verdict écrit dans thought_log
**Risque** : aucun (juste du log, pas de fix)

### Phase 6 — Surface `last_attempt_status` pour les 403 (30 min)

Ajouter une colonne `last_fetch_status` dans `career_progression` (NULLABLE) :
- `NULL` : pas d'info (compat ascendant)
- `'ok'` : dernière API a rendu données complètes
- `'forbidden_403'` : API a rendu 403 (privacy Halo)
- `'auth_missing'` : aucun token disponible
- `'timeout'` : budget dépassé

L'INSERT inclut ce champ. La home peut le surfacer côté API JSON pour que
le frontend affiche un message explicite (ex: "ce joueur a son profil
Spartan en privé").

**Livrable** : XxDaemonGamerxX → forbidden_403 visible dans le JSON
**Risque** : faible (ajout colonne, pas modification)

### Phase 7 — Tests E2E par scénario API (1h)

Tests d'intégration avec mock `CareerFetcher` qui simule :

| Scénario | Live rend | Attendu en DB après visite | Attendu côté home |
|----------|----------|--------------------------|------------------|
| Cold start | tout vide | rien | placeholder |
| First success | progress + custom complets | 1 ligne avec tous champs set | tout affiché |
| Partial — custom only | custom OK, progress nil | 1 ligne avec banner/emblem/spartan_id, rank/xp NULL | banner+rang vu |
| Partial — progress only | progress OK, custom nil | 1 ligne avec rank/xp, banner/spartan_id NULL | rang+xp vus, banner gardée si previous |
| Regression | live rend tout vide | rien (pas d'INSERT) | dernières valeurs servies |
| API 403 | custom retourne 403 | ligne avec last_fetch_status='forbidden_403', autres champs NULL | progress affichées, custom fallback gris |

**Livrable** : 6 tests E2E verts
**Risque** : aucun (tests pur, pas de code prod)

### Phase 8 — Cleanup + livraison (30 min)

- `go test ./...` + `go vet ./...`
- Vérification visuelle home pour les 4 joueurs
- Thought log avec verdict de chaque phase
- Diag `cmd/diag_customization_population` re-run pour comparer avant/après

**Livrable** : `.ai/thought_log.md` à jour
**Risque** : aucun

---

## 6. Effort total

| Phase | Action | Durée | Risque |
|-------|--------|------:|--------|
| 1 | Revert chirurgical §11 | 30 min | faible |
| 2 | Audit + fix `mergeCareerRow` | 1h | moyen |
| 3 | INSERT field-aware | 45 min | faible |
| 4 | Stale-while-revalidate strict | 30 min | faible |
| 5 | Diag Madina | 30 min | aucun |
| 6 | Surface last_fetch_status | 30 min | faible |
| 7 | Tests E2E par scénario | 1h | aucun |
| 8 | Cleanup + livraison | 30 min | aucun |
| **Total** | | **~5h** | |

---

## 7. Critères de succès (par phase, vérifiables)

### Phase 1 (revert)
- [ ] Plus de fichier `spartan_identity_repo.go`
- [ ] Plus de méthode `WithSpartanIdentityRepo` dans `CareerLiveService`
- [ ] Plus de table `spartan_identity` dans la migration au boot
- [ ] `go test ./...` passe
- [ ] Home affiche les bannières connues (JGtm OK, Madina/XxDaemon affichent fallback gris ou dernière connue de career_progression)

### Phase 2 (merge fix)
- [ ] Test : live rend `BannerImageURL=""`, dbLast a `BannerImageURL="url"` → INSERT n'inclut PAS banner_image_url
- [ ] Test : live rend `Rank=0` (API absent), dbLast a `Rank=183` → INSERT n'inclut PAS rank
- [ ] Test : live rend `CurrentXP=0` (API explicit, début palier) → INSERT inclut current_xp=0

### Phase 3 (INSERT field-aware)
- [ ] Lecture après INSERT partiel : ARG_MAX FILTER retourne valeurs anciennes pour champs NULL nouveaux
- [ ] Aucune régression sur les tests existants `InsertCareerProgressionIfChanged`

### Phase 4 (SwR strict)
- [ ] Test : home visit avec live qui timeout → réponse ≤200ms avec dbFallback complet
- [ ] Test : live qui rend partiel → 1ère réponse = dbFallback, 2ème réponse = nouvelles valeurs

### Phase 5 (diag Madina)
- [ ] Log explicite dans logs/service.log : "career_live: kickoff_skipped reason=X xuid=Madina_xuid"
- [ ] Verdict écrit dans thought_log

### Phase 6 (last_fetch_status)
- [ ] JSON home pour XxDaemonGamerxX contient `spartan_identity.fetch_status = 'forbidden_403'`
- [ ] Frontend reçoit le statut et peut décider l'affichage

### Phase 7 (E2E)
- [ ] 6 scénarios verts
- [ ] Couverture branchée dans CI

### Phase 8 (livraison)
- [ ] Diag avant/après montre population améliorée
- [ ] Thought log signé

---

## 8. Ce qui ne change PAS dans V2

- Le schéma `career_progression` original (juste ajout colonne `last_fetch_status` optionnelle)
- L'API Halo elle-même (`GetSpartanCustomization`, `GetCareerProgress`, `ResolveNameplateURL`)
- Le contrat JSON `spartan_identity.banner_image_url` côté frontend
- `HomeSpartanIdentityBanner.tsx` côté frontend (sauf si on veut afficher le statut explicite — décision à prendre Phase 6)
- Le `FALLBACK_BANNER_URL = '/banner-default.png'` côté frontend (filet ultime conservé)
- Le `CustomizationRefresher` scheduler 6h — soit on le garde tel quel (continue à écrire dans career_progression via le helper V2), soit on le supprime si on juge que la visite home suffit. Décision Phase 5 après diag.

---

## 9. Risques et mitigation

### Risque 1 — Le revert §11 casse les tests refactor

**Mitigation** : on revert aussi les tests qui dépendent de `spartan_identity`.
Aucun test métier ne devrait dépendre du refactor V1.

### Risque 2 — Le fix mergeCareerRow casse une convention silencieuse

`current_xp=0 live` est aujourd'hui interprété comme "vraie valeur joueur".
Si on bascule sur "0 = pas écrit", on perd l'info "joueur vraiment à 0 XP".

**Mitigation** : distinguer via le wrapper sync.CareerRankData. Si l'API rend
un objet avec `CurrentXP=0` ET un autre champ non-zéro (par exemple `Rank=185`),
alors `CurrentXP=0` est réel. Si l'objet est entièrement zéro, alors l'API
n'a probablement rien rendu. Ajouter ce test explicitement.

### Risque 3 — Race condition INSERT parallèle

Visite home + scheduler tick en même temps → 2 INSERT en parallèle.
DuckDB gère mais on peut avoir 2 lignes identiques.

**Mitigation** : `CareerRankRowEqualForInsert` existe déjà et skip si dernière
ligne identique. Ajouter test concurrent.

### Risque 4 — Backfill historique

Les anciennes lignes `career_progression` ont des champs vides mélangés.
ARG_MAX FILTER les ignore correctement, donc pas de migration nécessaire.
La donnée historique reste exploitable telle quelle.

---

## 10. Ce que je NE ferai PAS sans te demander

- Faire un `git reset --hard` ou un revert massif sans liste explicite
- Modifier le schéma de `career_progression` au-delà d'ajouter `last_fetch_status`
- Supprimer une table existante
- Toucher au contrat JSON API (sauf ajout champ optionnel comme `fetch_status`)
- Modifier `HomeSpartanIdentityBanner.tsx` ou autre composant frontend
- Faire un PR ou merger quoi que ce soit
- Démarrer le serveur ou tester en live (sauf si tu me demandes explicitement)

---

## 11. Décision attendue de ta part

Avant que je touche au code, valide ou corrige :

1. **Le scope V2** : on couvre bien customisation + rang + XP dans le même refactor ?
2. **L'approche append-only** : on garde la table `career_progression` existante (1 ligne par sync, ARG_MAX FILTER en lecture) — pas de nouvelle table ?
3. **L'ordre des phases** : revert §11 d'abord, puis fix `mergeCareerRow`, puis le reste ?
4. **Les tests E2E** : tu veux que je liste les 6 scénarios attendus avant que je code Phase 7 ?
5. **Le scheduler 6h `CustomizationRefresher`** : garder ou supprimer ? Je propose de décider après Phase 5 (diag Madina)

Si tu valides, je commence par te lister les commits exacts à reverter (avec hashes, dates, descriptifs) en Phase 1, et je te montre avant d'exécuter quoi que ce soit.

---

## 12. Note sur le bug §12 (JGtm rank stale)

Hors scope V2. À investiguer dans un ticket dédié après V2 livré.
Causes possibles à explorer :
- Cache HTTP Halo Waypoint serveur-side
- Throttle ou freshness lag de l'endpoint `/hi/careerranks/careerRank1`
- Bug `enrichCareerRankFromMetadata` qui réécraserait

Reprendre §12 ancien plan après V2 si JGtm rank reste stale.
