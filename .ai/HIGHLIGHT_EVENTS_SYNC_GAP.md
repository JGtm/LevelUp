# HIGHLIGHT_EVENTS — GAP CRITIQUE SYNC + BACKFILL

> Créé le 2026-04-24  
> Priorité : **ULTRA HAUTE** — données manquantes à chaque sync depuis le portage Go

---

## 1. Diagnostic — Ce qui ne fonctionne pas

### 1.1 La sync Go ne collecte JAMAIS les highlight_events

Le moteur de sync Go (`apps/go-api/internal/sync/engine.go`) exécute `processMatch` pour chaque nouveau match. Voici ce que cette fonction fait aujourd'hui :

| Étape | Fait ? | Fonction Go |
|-------|--------|-------------|
| Fetch stats JSON | ✅ | `GetMatchStats` |
| Insert `match_registry` | ✅ | `InsertRegistryIfNotExists` |
| Insert `match_participants` | ✅ (si `WithParticipants`) | `InsertParticipants` |
| Insert `medals_earned` | ✅ (si `WithMedals`) | `InsertMedals` |
| Fetch film / highlight_events | ❌ **ABSENT** | — |
| Insert `highlight_events` | ❌ **ABSENT** | — |
| Set `events_loaded = TRUE` | ❌ **ABSENT** | — |
| Insert `killer_victim_pairs` | ❌ **ABSENT** | — |
| Mettre à jour `xuid_aliases` depuis events | ❌ **ABSENT** | — |

En conséquence, **tous les matchs syncronisés via Go ont `events_loaded = FALSE`** dans `match_registry` et `highlight_events` est toujours vide pour ces matchs.

### 1.2 Ce que fait la sync Python (v7/cockpit — référence)

En Python, `_match_processing.py::_process_new_match` fait en parallèle :

```python
stats_json, skill_json, highlight_events = await self._fetch_match_data(
    match_id, options
)
```

`_fetch_match_data` appelle concurremment :
1. `get_match_stats(match_id)` — endpoint `/hi/matches/{id}/stats`
2. `get_skill_stats(match_id, xuids)` — endpoint skill
3. `get_highlight_events(match_id)` → `spnkr.film.read_highlight_events(client, match_id=match_id)`

Les highlight_events viennent de l'**API film Halo** (UGC spectate), pas du JSON stats. C'est un endpoint séparé :

```
GET https://discovery-infiniteugc.svc.halowaypoint.com/hi/films/matches/{id}/spectate
```

Côté Go, **cet endpoint est bien implémenté** dans `GetMatchFilm` (`halo_client.go`), mais il n'est **jamais appelé pendant `processMatch`**.

Après fetch, Python appelle `_insert_new_match_shared` qui fait :
1. `_insert_shared_events(event_rows)` → insère dans `highlight_events`
2. `_upsert_event_aliases(event_rows)` → peuple `xuid_aliases` avec les gamertags des events (source la plus fiable)
3. `_insert_shared_killer_victim_pairs(match_id, highlight_events)` → peuple `killer_victim_pairs`
4. `UPDATE match_registry SET events_loaded = TRUE`
5. `UPDATE match_registry SET backfill_completed = ... | BACKFILL_FLAGS["events"]`

**Rien de tout cela n'existe dans `processMatch` Go.**

### 1.3 `SyncOptions` — champ manquant

`domain.SyncOptions` (Go) n'a pas de champ `WithHighlightEvents`.  
`SyncOptions` Python a `with_highlight_events: bool = True` (activé par défaut).

`DefaultSyncOptions()` Go ne contient donc aucune option pour les events.

---

## 2. Impact — Champs critiques cassés

### 2.1 `highlight_events` vide → fonctionnalités brisées

| Fonctionnalité | Fichier Go consommateur | Symptôme |
|----------------|------------------------|---------|
| **Tug-of-War** (onglet Combat) | `match_view_service.go:419` via `Q20KVPairs` | Graphique vide / plat |
| **Highlight events liste** (onglet match) | `queries_match.go:Q15` et `Q21` | Tableau vide |
| **Dominance flag** (header match) | `analysis/comeback.go` via `BuildScoreSnapshots` | Toujours `false`/0 |
| **KillerVictim / Encounters** (onglet Team) | `queries_match.go:Q20`, `queries_squad.go:L104` | Tableaux vides |
| **xuid_aliases** incomplet | `_upsert_event_aliases` absente | Gamertags manquants depuis les events |
| **⚠️ Weapon kills backfill** | `backfill_weapons.go:getKillsForPlayer` | `getKillsForPlayer` lit `highlight_events (event_type='Killed')` — si vide, aucun kill à corréler → `weapon_kills` reste vide |

> **Cascade critique** : `highlight_events` est une dépendance de `backfill_weapons`. Sans events, le pipeline weapon kills est aveugle même si le film est bien téléchargé.

### 2.2 `events_loaded` jamais positionné

- Tous les matchs Go ont `events_loaded = FALSE`
- Le backfill `scope.Events` détecte tous les matchs Go comme "à retraiter"
- Si un backfill se lance, il re-fetch inutilement tous les matchs récents

### 2.3 `killer_victim_pairs` vide

- Calculé à partir des `highlight_events` lors de l'insertion
- `Q20KVPairs` retourne toujours vide
- L'onglet Team / Encounters et le Tug-of-War sont cassés en cascade

---

## 3. Analyse comparative Python ↔ Go

### 3.1 Appel API film

| | Python | Go |
|-|--------|----|
| Endpoint | `spnkr.film.read_highlight_events(client, match_id)` → `GET /hi/films/matches/{id}/spectate` + parsing binaire | `GetMatchFilm(ctx, matchID)` — **implémenté**, télécharge les mêmes chunks binaires |
| Parsing structuré (events typés) | `spnkr.film` parse les chunks → objets Python avec `event_type`, `xuid`, `victim_xuid`, `time_ms`, `gamertag` | **❌ ABSENT** — Go n'a pas l'équivalent de ce parseur |
| Parsing fire events (armes) | Non utilisé pour highlight_events | `analysis.ScanFireEventsAll` — **uniquement pour weapon_kills**, ne produit pas de highlight_events |

> **Distinction fondamentale** : le film binaire contient deux types d'information :  
> 1. Des **événements de gameplay typés** (Killed, Assist, Medal, ...) → `spnkr.film.read_highlight_events` en Python, **non implémenté en Go**  
> 2. Des **fire events bas-niveau** (position, arme, timing) → `analysis.ScanFireEventsAll` en Go, pour weapon_kills uniquement

### 3.2 Structure `highlight_events` (schéma DB)

D'après `steps_shared.go` et les transformers Python, la table `highlight_events` attend :

```sql
CREATE TABLE highlight_events (
    id         INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
    match_id   VARCHAR  NOT NULL,
    event_type VARCHAR  NOT NULL,
    time_ms    INTEGER,
    xuid       VARCHAR,
    type_hint  INTEGER,
    raw_json   VARCHAR
);
```

Le transformer Python (`transformers/_events.py`) produit :
- `event_type` — type d'événement (ex: `"Killed"`, `"Assist"`, `"Medal"`, ...)
- `time_ms` — timestamp en ms dans le match
- `xuid` — XUID du joueur concerné
- `gamertag` — gamertag (source fiable, différente de l'API stats)
- `type_hint` — entier de type
- `raw_json` — JSON brut de l'event

### 3.3 Ce que fait `spnkr.film.read_highlight_events`

L'API retourne un manifest film avec des chunks binaires Azure. La lib `spnkr.film` parse ces chunks pour extraire des **événements structurés** (pas des fire events pour les armes, mais des événements de gameplay : kills, assists, etc.). Ces événements sont **distincts** des fire events de la Section 2 des chunks binaires utilisés pour le pipeline weapon_kills.

En Go, **ce parsing d'événements typés (highlight events) depuis les chunks film n'est pas implémenté**. Seul le parsing pour les weapon kills est présent (`analysis/weapon_scanner.go`).

---

## 4. Plan de correction

### Phase 1 — Implémenter le parseur d'événements typés depuis le film binaire

**Priorité : IMMÉDIATE — bloquante pour tout le reste**

C'est le travail de fond le plus important. En Python, `spnkr.film.read_highlight_events` extrait les événements typés (Killed, Assist, Medal, etc.) depuis les chunks binaires. Go n'a pas cet équivalent. Il faut l'implémenter.

**Approche recommandée** : analyser la librairie `spnkr` open source et le format film Halo pour implémenter `ParseHighlightEvents(chunks map[int]filmChunkData) []HighlightEventRow` dans `internal/analysis/` (ou `internal/sync/`).

Les événements attendus ont la structure :
```go
type HighlightEventRow struct {
    MatchID   string
    EventType string  // "Killed", "Assist", "Medal", ...
    TimeMS    int
    XUID      string
    TypeHint  *int
    RawJSON   string
    // Pour killer_victim_pairs :
    VictimXUID    *string
    KillerGamertag *string
    VictimGamertag *string
}
```

> **Alternative** : contacter / inspecter le code source de `spnkr` (open source sur GitHub : `acurtis166/SPNKr`) pour comprendre le format binaire des highlight events.

### Phase 2 — Ajouter `WithHighlightEvents` à `SyncOptions`

Fichier : `apps/go-api/internal/domain/sync.go`

```go
type SyncOptions struct {
    MatchType            string
    MaxMatches           int
    WithParticipants     bool
    WithMedals           bool
    WithHighlightEvents  bool  // ← NOUVEAU
    RequestsPerSecond    int
}

func DefaultSyncOptions() SyncOptions {
    return SyncOptions{
        MatchType:           "matchmaking",
        MaxMatches:          200,
        WithParticipants:    true,
        WithMedals:          true,
        WithHighlightEvents: true,  // activé par défaut comme Python
        RequestsPerSecond:   10,
    }
}
```

### Phase 3 — Ajouter `InsertHighlightEvents` + `InsertKillerVictimPairs` dans `sync/writes.go`

```go
func InsertHighlightEvents(db *sql.DB, rows []HighlightEventRow) (int, error) { ... }
func InsertKillerVictimPairsFromEvents(db *sql.DB, matchID string, events []HighlightEventRow) error { ... }
```

### Phase 4 — Connecter dans `processMatch` (`engine.go`)

Ajouter après l'insertion des médailles :

```go
// ─── highlight_events ──────────────────────────────────────────────────────
if opts.WithHighlightEvents {
    rawChunks, found, err := client.GetMatchFilm(ctx, matchID)
    if err != nil {
        result.AddWarning(fmt.Sprintf("GetMatchFilm(%s): %v", matchID, err))
    } else if found {
        events := ParseHighlightEvents(rawChunks)  // Phase 1
        if len(events) > 0 {
            n, _ := InsertHighlightEvents(sharedDB, events)
            _ = InsertKillerVictimPairsFromEvents(sharedDB, matchID, events)
            // xuid_aliases depuis les events (source la plus fiable)
            for _, ev := range events {
                if ev.XUID != "" && ev.Gamertag != "" {
                    _ = UpsertXUIDAlias(sharedDB, ev.XUID, ev.Gamertag)
                }
            }
            if n > 0 {
                _, _ = sharedDB.Exec(
                    "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?", matchID)
                result.EventsInserted += n
            }
        }
    }
}
```

### Phase 5 — Ajouter `EventsInserted` à `SyncResult`

```go
type SyncResult struct {
    MatchesInserted  int
    MatchesSkipped   int
    MedalsInserted   int
    ParticipantsDone int
    EventsInserted   int  // ← NOUVEAU
    // ...
}
```

### Phase 6 — Tests

- [ ] Test unitaire `ParseHighlightEvents` avec chunks binaires réels ou fixtures
- [ ] Test E2E `engine_e2e_test.go` : vérifie `highlight_events` peuplé après `processMatch`
- [ ] Test `InsertHighlightEvents` + `InsertKillerVictimPairsFromEvents`
- [ ] Vérifier `events_loaded = TRUE` après sync
- [ ] Vérifier que `backfill_weapons::getKillsForPlayer` retourne des kills non vides

---

## 6. Fichiers impactés par la correction

| Fichier | Modification |
|---------|-------------|
| `internal/domain/sync.go` | Ajouter `WithHighlightEvents`, `EventsInserted` |
| `internal/sync/engine.go` | `processMatch` : appel extract + insert events |
| `internal/sync/writes.go` | `InsertHighlightEvents`, `InsertKillerVictimPairs` |
| `internal/sync/transforms.go` | `ExtractHighlightEvents(matchJSON)` si Option B |
| `internal/sync/halo_client.go` | Potentiellement nouvel endpoint si Option C |
| `internal/analysis/` | Potentiellement parseur film events si Option A |
| `internal/sync/engine_e2e_test.go` | Nouveaux tests |

---

## 7. Entrée thought_log

```
[2026-04-24] highlight_events — GAP CRITIQUE SYNC GO
Statut : En cours — investigation technique
Décision : La sync Go (processMatch) ne collecte jamais les highlight_events.
           GetMatchFilm existe mais n'est appelé que pour weapon_kills (backfill).
           SyncOptions n'a pas WithHighlightEvents.
Résultats : Tug-of-War, KillerVictim, DominanceFlag, EventsList tous cassés.
Prochaine étape : Inspecter JSON stats pour Option B avant implémentation.
```
