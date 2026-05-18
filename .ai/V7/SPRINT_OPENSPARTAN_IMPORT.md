# Plan : Import depuis une BDD OpenSpartan

> **Statut** : Plan d'implémentation futur, indépendant.
> **Dépendance** : profite du SSO Xbox (`SPRINT_XBOX_SSO.md`) pour valider l'identité, mais peut techniquement vivre sans.
> **Branche cible** : à créer (`feat/openspartan-import`), depuis `main`.
> **Auteur du plan** : Claude (session du 2026-05-16, révisé 2026-05-18).
> **Révision 2026-05-18** : périmètre resserré sur **vanilla OpenSpartan/grunt** après inspection d'une vraie DB. Voir [§2 Format réel](#2-format-réel-dune-bdd-openspartan).

---

## 0. Contexte

**OpenSpartan** (`github.com/OpenSpartan/openspartan` et `OpenSpartan/grunt`) est un client Halo Infinite tiers qui maintient une base SQLite locale par utilisateur. Beaucoup de joueurs Halo l'ont utilisé avant LevelUp et possèdent un historique de matchs important — souvent **plus ancien** que ce que l'API Halo de Microsoft expose actuellement (l'API tronque l'historique au-delà d'une fenêtre récente, le nombre exact dépendant du mode de jeu).

**Use case** :
1. User s'inscrit sur LevelUp via SSO Xbox.
2. Au lieu d'attendre 30 min de sync initial (et de perdre les matchs vieux que l'API ne renvoie plus), il importe sa BDD OpenSpartan.
3. LevelUp parse les réponses Halo API stockées en JSON brut par OpenSpartan et les insère dans `shared_matches_v2.duckdb` au format v6.

**Périmètre v1 (décidé 2026-05-18)** : **vanilla OpenSpartan/grunt uniquement**. Les anciennes bases SQLite LevelUp v3/v4 (qui réutilisent l'infra grunt + ajoutent leurs propres tables) ne sont pas une cible — leur cas est traité ad-hoc par un script de migration interne. Le service détecte les tables canoniques OpenSpartan et **ignore** toute extension propriétaire qu'il pourrait croiser.

---

## 1. Décisions design

### Quoi importer depuis OpenSpartan

| Source OpenSpartan (vanilla) | Cible LevelUp v6 | Priorité | Note |
|---|---|---|---|
| `MatchStats.ResponseBody` (JSON `match-stats` Halo API) | `shared.match_registry` + `shared.match_participants` | **P0** | Source primaire — tous les joueurs du match sont dans `$.Players[]` |
| `PlayerMatchStats.ResponseBody` (JSON `player-match-stats` Halo API) | Complément stats détaillées du owner (shots fired/hit, awards éventuels) | **P0** | Croise avec MatchStats par `MatchId` |
| `$.Players[].PlayerTeamStats[].Stats.CoreStats.Medals` (embedded JSON) | `shared.medals_earned` | **P1** | Dans le JSON, pas dans une table dédiée vanilla |
| `HighlightEvents.ResponseBody` | `shared.highlight_events` | **P1** | Table existe vanilla, JSON brut |
| `XuidAliases` (Xuid, Gamertag, LastSeen) | `shared.xuid_aliases` | **P0** | Critique pour résolution identité cross-DB |
| `Friends` (owner_xuid, friend_xuid, friend_gamertag) | **Stash JSON** dans `players/{gamertag}/stash/friends.json` | **P2** | Pas de table SQL — réservé pour MULTIUSER_ACL futur. Décision 2026-05-18. |
| Killer/victim — n'existe pas en vanilla OpenSpartan | — | — | Sera repeuplé par sync API si l'event est dans `HighlightEvents` |
| `ServiceRecordSnapshots`, `PlaylistCSRSnapshots` | **Skip** | — | Cumulatifs recalculés depuis matchs |
| `Maps`, `Playlists`, `GameVariants`, `PlaylistMapModePairs`, `EngineGameVariants` | **Skip** | — | Référentiels gérés par `metadata.duckdb` LevelUp |
| `InventoryItems`, `OwnedInventoryItems`, `OperationRewardTracks` | **Skip** | — | Hors scope LevelUp |
| `MedalsAggregate`, `TeammatesAggregate`, `PerformanceScores`, `MatchCache` (si présentes) | **Ignorer explicitement** | — | Extensions LevelUp legacy — recalculées côté v6 pour cohérence |
| Médias (screenshots) | **Skip** | — | Pas dans OpenSpartan en standard |

### Stratégie de conflit

- **Match déjà présent dans `shared.match_registry`** → skip silencieux (l'API est source de vérité pour les matchs récents).
- **Match présent uniquement dans OpenSpartan** → insert.
- **Stats joueur incomplètes côté OpenSpartan** (ex: shots_fired absent du JSON) → insert quand même, colonnes manquantes restent NULL, le sync API peut les compléter plus tard.
- **Réimport d'un même fichier** → idempotent par contrainte d'unicité sur `match_id` (shared.match_registry), `(match_id, xuid)` (match_participants), `(match_id, xuid, medal_id)` (medals_earned).

---

## 2. Format réel d'une BDD OpenSpartan

**Constats d'inspection (DB exemple : `{xuid}.db`, 27 MB, 451 matchs)** :

### Fichier
- **Extension : `.db`** (SQLite standard), **pas** `.osdb`. Nom du fichier = `{xuid}.db` (XUID du owner).
- `PRAGMA user_version` = 0 → **inutilisable pour détecter le schéma**. Détection à faire par signature de tables (présence de `MatchStats` avec colonne `ResponseBody`).
- Magic header SQLite standard, pas de signature propriétaire.

### Tables canoniques vanilla (toutes JSON brut)

```sql
CREATE TABLE MatchStats (
  ResponseBody TEXT,                                            -- JSON Halo API
  MatchId   TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchId'))   VIRTUAL,
  MatchInfo TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchInfo')) VIRTUAL,
  Teams     TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.Teams'))     VIRTUAL,
  Players   TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.Players'))   VIRTUAL
);

CREATE TABLE PlayerMatchStats (
  ResponseBody TEXT,
  MatchId TEXT,
  PlayerStats TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.Value')) VIRTUAL
);

CREATE TABLE HighlightEvents (
  MatchId TEXT NOT NULL,
  ResponseBody TEXT NOT NULL,
  EventType TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.event_type')) VIRTUAL,
  TimeMs    INTEGER GENERATED ALWAYS AS (json_extract(ResponseBody, '$.time_ms')) VIRTUAL,
  Xuid      TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.xuid'))       VIRTUAL,
  Gamertag  TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.gamertag'))   VIRTUAL,
  TypeHint  INTEGER GENERATED ALWAYS AS (json_extract(ResponseBody, '$.type_hint')) VIRTUAL
);

CREATE TABLE XuidAliases (
  Xuid TEXT PRIMARY KEY,
  Gamertag TEXT NOT NULL,
  LastSeen TEXT,
  Source TEXT,                -- typiquement 'api'
  UpdatedAt TEXT
);

CREATE TABLE Friends (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_xuid TEXT NOT NULL,
  friend_xuid TEXT NOT NULL,
  friend_gamertag TEXT,
  nickname TEXT,
  added_at TEXT
);

CREATE TABLE CacheMeta ( key TEXT PRIMARY KEY, value TEXT, updated_at TEXT );
CREATE TABLE SyncMeta  ( key TEXT PRIMARY KEY, value TEXT, updated_at TEXT );
```

### Implications pour l'implémentation

1. **Tout le mapping passe par parsing JSON** — il n'y a pas de tables structurées "OSMatch / OSParticipant" prêtes à l'emploi. Les structs Go doivent modéliser la réponse Halo API officielle (`match-stats` v1).
2. **Le owner n'est `MatchStats` que via tous les Players** — `MatchStats.ResponseBody.Players[]` contient TOUS les joueurs du match (humains + bots). Le owner se déduit par croisement avec `PlayerMatchStats.ResponseBody.MatchId` ou via le nom du fichier `{xuid}.db`.
3. **Médailles enfouies** — `medal_id` + `count` par joueur sont dans `$.Players[i].PlayerTeamStats[j].Stats.CoreStats.Medals[]`. Pas dans une table dédiée. Clé v6 : `(match_id, xuid, medal_id)`.
4. **Killer/victim absent** — `killer_victim_pairs` ne se reconstruit qu'à partir des `HighlightEvents` de type kill, si on veut. Reco : **skip en v1**, l'API repeuplera lors du prochain sync.
5. **Heuristique de détection du owner** :
   - Primaire : nom du fichier `{xuid}.db` (string match `^\d{16}\.db$`)
   - Secondaire : `SELECT PlayerId FROM (json_each Players where PlayerType='human') GROUP BY PlayerId ORDER BY COUNT(*) DESC LIMIT 1` (le xuid présent dans le plus de matchs = le owner)
   - Tertiaire : `SELECT value FROM CacheMeta WHERE key LIKE '%xuid%'` si présent

---

## 3. PR 1 — Reader SQLite + parser JSON Halo API

**Périmètre** : ouvrir le SQLite OpenSpartan, exposer un itérateur sur les matchs.

- Nouveau package `apps/go-api/internal/import/openspartan/`
  - `reader.go` : ouvre le `.db` en read-only via **`modernc.org/sqlite`** (pur Go, pas de CGO, build Windows propre).
  - `detector.go` : vérifie la signature minimale (présence de tables `MatchStats`, `PlayerMatchStats` avec colonne `ResponseBody`) et le contenu non vide. Retourne `ErrNotOpenSpartanDB` sinon.
  - `models.go` : structs Go pour la réponse Halo `match-stats` v1 (`MatchInfo`, `Team`, `Player`, `PlayerTeamStats`, `CoreStats`, `MedalAward`, ...) — modélisation **directe de l'API Microsoft**, pas d'invention OpenSpartan.
  - `iterator.go` : `func (r *Reader) MatchesByOwner(ctx, ownerXUID) iter.Seq2[*ParsedMatch, error]` — stream, n'instancie pas tout en RAM.
  - `owner.go` : `func (r *Reader) DetectOwner(filenameHint string) (xuid string, confidence Confidence, err error)` — applique les 3 heuristiques ci-dessus.

- **Test fixtures** : créer 2-3 `.db` réels minimaux (5-10 matchs, mix matchmaking 4v4 / BTB / ranked / custom) dans `testdata/openspartan/`. Réutiliser ma vraie DB (anonymisée si nécessaire) comme golden test pour le throughput.

- **Note SQLite Go** : la règle "SQLite interdit" de `CLAUDE.md` vise le code Python (`src/`). Le code Go n'est pas concerné — donc pas d'exception à documenter, juste choisir le bon driver.

---

## 4. PR 2 — Mapper Halo API JSON → LevelUp v6

**Périmètre** : transformer un `ParsedMatch` (sortie PR 1) vers les structures DuckDB v6.

- Nouveau package `apps/go-api/internal/import/openspartan/mapper/`
  - `registry.go` : `func MapRegistry(pm ParsedMatch) (domain.MatchRegistryRow, error)` — peuple `shared.match_registry` (mode, map, dates, playlist, scores équipe).
  - `participants.go` : `func MapParticipants(pm ParsedMatch) ([]domain.MatchParticipantRow, error)` — itère sur `$.Players[]`, génère 1 row par humain (filtre `PlayerType='human'`, ignore les bots dans v6 / TBD).
  - `medals.go` : `func MapMedals(pm ParsedMatch) ([]domain.MedalEarnedRow, error)` — itère `$.Players[i].PlayerTeamStats[j].Stats.CoreStats.Medals[]`, génère N rows par joueur.
  - `highlights.go` : `func MapHighlight(hl ParsedHighlight) (domain.HighlightEventRow, error)` — mapping direct, plus simple.
  - `mode.go` : utilise la skill `halo-modes` pour normaliser les modes (variant_id → canonical mode).

- **Edge cases** :
  - Match avec `PlayerType=Bot` → ignoré (v6 = stats humaines uniquement).
  - `MatchInfo.StartTime` dans le futur (corruption) → skip + log warning.
  - Mode inconnu (DLC futur) → log warning + insert avec `mode=NULL`. Le post-traitement pourra le compléter quand le mode sera ajouté à `metadata.duckdb`.
  - JSON malformé sur un match → log + skip ce match, continuer les autres. Ne **jamais** abort l'import entier sur 1 mauvais row.
  - Champ stat manquant côté Halo API (ex: anciens matchs sans `ShotsFired`) → NULL, sync API ultérieur complétera.

- **Tests** : pour chaque famille de mode (matchmaking 4v4, BTB, ranked, custom, FFA), un fixture JSON + assertions sur les rows générées. Vérifier l'unicité `(match_id, xuid)` côté participants.

---

## 5. PR 3 — Service d'import + endpoint

**Périmètre** : sécurité, orchestration, persistance v6.

- Service `apps/go-api/internal/service/openspartan_import_service.go`
  - `func ImportFromOpenSpartan(ctx, ownerXUID, dbPath string, opts ImportOptions) (ImportResult, error)`
  - **Validation XUID stricte** :
    1. Appeler `reader.DetectOwner(filenameHint=dbPath)` → `(detectedXUID, confidence)`.
    2. Si `detectedXUID != ownerXUID` (le XUID de la session SSO) → refuser avec `ErrXUIDMismatch`. Pas de bypass.
    3. Si `confidence == Low` (3 heuristiques contradictoires) → refuser avec un message clair : "Impossible de vérifier que cette BDD t'appartient."
  - **Insertion DuckDB en bulk** via l'API **Appender** (10-100× plus rapide que des INSERT prepared). Batch implicite. Une transaction par type de row (`match_registry`, `match_participants`, `medals_earned`, `highlight_events`, `xuid_aliases`).
  - **Stash Friends** : si la table existe et est non vide, sérialiser en JSON → écrire dans `data/players/{gamertag}/stash/openspartan_friends.json`. Pas de schéma DuckDB créé. Le futur sprint MULTIUSER_ACL pourra le consommer.
  - **Dry-run mode** : `opts.DryRun=true` → tout parser, tout valider, ne rien écrire. Renvoie le `ImportResult` avec les compteurs prévus.
  - Le fichier `.db` est **traité depuis un tmp puis supprimé** (jamais conservé).

- Endpoint `POST /import/openspartan` :
  - Upload multipart (50-500 MB typique, quota max 1 GB).
  - Renvoie `ImportResult { detected_owner_xuid, added_matches, skipped_existing, added_medals, added_highlights, stashed_friends, errors[] }`.
  - **Long-running** : utiliser le système de jobs existant. Le handler crée le job, retourne immédiatement un `job_id`. Le frontend poll via `useJobToasts` côté `apps/web`.
  - **Progress** : le service écrit son avancement (matchs parsés / matchs total) dans le job → progress bar côté UI.

- **Sécurité** :
  - Quota taille upload : 1 GB max.
  - Le service tourne avec les perms de l'user SSO connecté uniquement. Pas de path query string, le `dbPath` est toujours un tmp serveur.
  - L'XUID owner est **toujours** pris depuis la session SSO, jamais depuis le payload client.

---

## 6. PR 3.5 — Post-import recompute

**Périmètre** : remettre les dérivés v6 cohérents après l'insertion brute.

Insertion brute = `match_registry`, `match_participants`, `medals_earned`, `highlight_events`, `xuid_aliases` peuplés. Mais les **calculs v6 dépendants** ne sont pas faits :

| Recompute | Source de vérité | Comment |
|---|---|---|
| `player_match_enrichment.session_id`, `session_label` | clustering temporel par owner | Réutiliser la logique existante (`backfill --sessions` côté Python, ou son équivalent Go si déjà migré) |
| `player_match_enrichment.performance_score` | formule v6 | Réutiliser le batch perf_score Go (voir `apps/go-api/internal/analysis/`) |
| `match_citations` | médailles importées + référentiel `citation_mappings` | Recompute par joueur après import |
| `mv_*` (vues matérialisées) | tables source | Si `mv_*` sont matérialisées physiquement → REFRESH ; si VIEW SQL pures → rien à faire |
| Stats agrégées (KDA cumulé, etc.) | `mv_player_matches` | Auto, via les `mv_` |

**Implémentation** : à la fin de `ImportFromOpenSpartan`, déclencher le post-traitement en **async** (le job d'import est déjà long, on ne bloque pas). Ce post-traitement peut soit :
- (option A) appeler en-process les services Go de recompute déjà existants.
- (option B) émettre un event "owner_xuid X needs recompute" sur le bus interne, qu'un worker existant consomme.

Reco : **option A** au début (synchrone dans le job), B plus tard si le post-traitement devient lent.

---

## 7. PR 4 — UI onboarding "Mode avancé"

**Périmètre** : exposer l'option dans le flow d'onboarding Xbox SSO, derrière un disclosure. Décision 2026-05-18 : **pas d'onglet Settings** en v1.

- Étape post-inscription Xbox, après le succès de `XboxLoginPage` :
  - Card par défaut : "Sync initial en cours — on récupère tes derniers matchs depuis Halo Waypoint." (UX standard, action principale)
  - Lien discret en bas : **"Options avancées →"** (collapse fermé par défaut)
  - Dans le collapse : `OpenSpartanImportCard` avec drag & drop `.db` + texte explicatif court ("Si tu as déjà utilisé OpenSpartan, importe ta BDD pour récupérer des matchs plus anciens que l'API.")

- Composant `apps/web/src/features/auth/onboarding/OpenSpartanImportCard.tsx` :
  - Drag & drop / file input `.db`
  - POST multipart → `job_id`
  - Affiche progress bar via `useJobToasts`
  - Sur succès : affiche `ImportResult` ("X matchs importés, Y déjà connus, Z ignorés. Détails →")
  - Sur échec XUID mismatch : message clair "Cette BDD n'appartient pas à ton compte Xbox."

- **Pas affiché aux users existants** (déjà onboardés) en v1. Si demande future, ajouter une tab Settings dans une PR séparée.

- Tests : composant avec MSW + fixture `.db` minimal (peut être un fichier vide qui mock le 400 côté API).

---

## 8. Pièges à éviter

1. **Détection de schéma sans `user_version`** — `PRAGMA user_version` est à 0. Détecter via présence des tables canoniques + une colonne attendue. Si tu croises une vieille version qui n'a pas `MatchStats.ResponseBody` (très vieux grunt) → refuser proprement avec message "version trop ancienne, contactez le support".

2. **Validation XUID stricte** — sans ça, n'importe qui peut importer la BDD d'un autre. Le XUID owner vient TOUJOURS de la session SSO, jamais du payload. Voir [§5 Sécurité](#5-pr-3--service-dimport--endpoint).

3. **Pas de fusion implicite si réimport** — `match_registry` PK = `match_id`, `match_participants` PK = `(match_id, xuid)`, `medals_earned` PK = `(match_id, xuid, medal_id)`. Insertion via `INSERT OR IGNORE` ou Appender avec contrainte d'unicité. Pas de update silencieux.

4. **Performance** — 10 000+ matchs = quelques minutes via Appender DuckDB. Le faire en job background avec progress visible, **jamais** dans le request handler. Pour 451 matchs (taille typique observée), compter ~30s parse + 5s insert.

5. **Pas de backup pré-import** — décision 2026-05-18 : à l'onboarding la DB joueur n'existe pas encore (ou est vide), donc rien à backuper. Si plus tard on ouvre l'import à des users déjà onboardés, **alors** réintroduire un backup auto. Pour l'instant : noop.

6. **`shared` vs `player`** — les matchs OpenSpartan vont dans `shared.*` (centralisé). Bien faire attention à NE PAS écrire dans `data/players/{gamertag}/stats.duckdb` côté participants/médailles/highlights (qui ne contient plus de tables matchs depuis v5.1). Seul écrit côté player : créer un `stats.duckdb` vide avec les tables d'enrichissement v6 (`player_match_enrichment`, `match_citations`, ...) si le user vient de s'inscrire.

7. **xuid_aliases : merge, pas replace** — la table `shared.xuid_aliases` peut déjà contenir un gamertag plus récent pour un xuid donné (renames Xbox). L'import doit faire `INSERT OR IGNORE` puis update conditionnel `WHERE updated_at < ?`, jamais écraser une entrée plus fraîche.

8. **Mode bot / custom** — `PlayerType=Bot` → ignoré côté participants. Modes custom avec score perso peuvent avoir des colonnes manquantes → NULL, pas erreur.

9. **JSON malformé** — log + skip ce match, continuer. Le `ImportResult.errors[]` liste les `match_id` skippés pour transparence.

10. **Concurrency Appender DuckDB** — l'Appender prend un lock writer sur la DB. Bien sérialiser : pas 2 imports en parallèle sur la même `shared_matches_v2.duckdb`. Le job system doit garantir la sérialisation (mutex global ou file d'attente).

---

## 9. Estimation

| PR | Effort | Bloque la suite ? |
|---|---|---|
| PR 1 (reader + parser JSON Halo API) | 1.5j | Oui |
| PR 2 (mapper → v6) | 1.5j | Oui (dépend PR 1) |
| PR 3 (service + endpoint + sécurité) | 1.5j | Oui (dépend PR 1+2) |
| PR 3.5 (post-import recompute) | 1j | Oui (dépend PR 3) |
| PR 4 (UI onboarding mode avancé) | 0.5j | Non (peut être stub `curl` au début) |

**Total** : **~6 jours** de dev focused (révisé depuis 4j initiaux après prise en compte du parsing JSON, des structs Halo API à modéliser, et du post-traitement).

---

## 10. Avant de démarrer

- [ ] Récupérer 2-3 `.db` OpenSpartan réels (le mien + idéalement un autre joueur) pour tester le parsing sur des JSON réels variés.
- [ ] Confirmer que les `MatchId` OpenSpartan sont bien les UUIDs Halo Infinite Microsoft (très probable vu que `MatchStats.ResponseBody.MatchId` vient de l'API directement). Un quick check `SELECT MatchId FROM MatchStats LIMIT 5` suffira.
- [ ] Décider du quota taille upload max (proposé : 1 GB).
- [ ] Vérifier qu'il existe un driver SQLite Go pur (`modernc.org/sqlite`) dans le `go.mod` ou l'ajouter.
- [ ] Vérifier l'existence d'un service Go de recompute (sessions, perf_score) pour PR 3.5, ou décider de déléguer au script Python `backfill_data.py` en attendant.
- [ ] Trancher option A vs B pour le post-recompute (synchrone dans le job vs event sur bus).

---

## Annexe — Différences avec la v1 du plan (2026-05-16)

| Point | v1 | v2 (2026-05-18) | Raison |
|---|---|---|---|
| Extension fichier | `.osdb` | `.db` | Inspection : pas d'extension propriétaire |
| Détection version | `PRAGMA user_version` | Signature de tables | `user_version` = 0 dans une vraie DB |
| Mapper input | "rows OpenSpartan structurées" | Parsing JSON `ResponseBody` | Réalité : tout est JSON brut |
| Médailles | Table `MedalsAggregate` | Embedded dans `$.Players[].PlayerTeamStats[].Stats.CoreStats.Medals` | `MedalsAggregate` est une extension LevelUp legacy, vide en vanilla |
| Participants | "via mapping table" | Parsing `$.Players[]` du JSON | Idem |
| Killer/victim | P1 | Skip v1 (à reconstruire depuis HighlightEvents si besoin) | Pas de table dédiée vanilla |
| Friends | Non mentionné | Stash JSON pour MULTIUSER_ACL | Décision user 2026-05-18 |
| XuidAliases | Non explicite | P0 explicite | Critique pour résolution identité |
| Backup pré-import | "proposer ou forcer" | Skip (onboarding-only) | Décision user 2026-05-18 |
| UI | Tab Settings ou onboarding | Onboarding uniquement, derrière "Mode avancé" | Décision user 2026-05-18 |
| Post-recompute | Non mentionné | PR 3.5 dédiée | Cohérence v6 (sessions, perf_score, citations) |
| Driver SQLite Go | Non précisé | `modernc.org/sqlite` (pur Go) | Pas de CGO sur Windows |
| Insertion DuckDB | Batch 1000 INSERT | API Appender | 10-100× plus rapide |
| Estimation | 4j | 6j | Parsing JSON + post-recompute + structs API |
