# Attribution des kills par arme — Référence technique

Ce document décrit comment LevelUp attribue chaque kill d'un match à l'arme qui
l'a causé. L'attribution est dérivée **hors-ligne** des chunks du film (théâtre)
récupérés depuis le CDN des films Halo — les API de stats Halo n'exposent pas
l'arme par kill. Tout le pipeline est implémenté en Go sous `apps/go-api`.

> Portée : l'attribution arme-par-kill est **best-effort, non autoritative**.
> Elle reconstruit l'arme depuis les données de réplication du film et la
> réconcilie avec les totaux de l'API. Lire [Niveaux de confiance](#niveaux-de-confiance)
> et [Limites connues](#limites-connues) avant de s'y fier.

---

## Table des matières

1. [Vue d'ensemble](#vue-densemble)
2. [Carte du code](#carte-du-code)
3. [Pipeline](#pipeline)
4. [Chemins d'attribution](#chemins-dattribution)
5. [Niveaux de confiance](#niveaux-de-confiance)
6. [Structure d'un Weapon ID (WID)](#structure-dun-weapon-id-wid)
7. [Stockage — `weapon_kills` (append-only)](#stockage--weapon_kills-append-only)
8. [Lecture — `v_weapon_kills` et labels](#lecture--v_weapon_kills-et-labels)
9. [Lancer le backfill](#lancer-le-backfill)
10. [Ajouter / résoudre un Weapon ID](#ajouter--résoudre-un-weapon-id)
11. [Limites connues](#limites-connues)
12. [Approfondissement : rétro-ingénierie du film](#approfondissement--rétro-ingénierie-du-film)

---

## Vue d'ensemble

Pour un match donné, le moteur télécharge le film, scanne les chunks binaires de
réplication pour les événements de tir et d'arme tenue, corrèle chaque kill
(depuis `highlight_events`) à une arme, réconcilie le résultat avec les
décomptes de kills de l'API, puis écrit une ligne par kill dans la table
partagée `weapon_kills`.

Le résultat agrégé (kills par arme, avec labels EN/FR) est relu via la vue
`v_weapon_kills` et la table `metadata.weapon_labels`, et exposé dans la vue
match et le KPI « arme favorite » de l'accueil.

---

## Carte du code

| Sujet | Emplacement |
|-------|-------------|
| Orchestration pipeline (par match / tous participants / batch) | `internal/sync/backfill_weapons.go` |
| Écriture DB (INSERT append-only) | `internal/sync/writes.go` — `InsertWeaponKills`, `MarkWeaponKillsDone` |
| Scan des chunks (fire events, timeline arme tenue) | `internal/analysis/weapon_scanner.go`, `internal/analysis/weapon_parser.go` |
| Map des Weapon IDs, timings, fusions, sentinels | `internal/analysis/weapon_data.go` |
| Corrélation kill -> arme | `internal/analysis/weapon_correlation.go` |
| Réconciliation API | `internal/analysis/weapon_reconciliation.go` |
| Struct résultat d'attribution | `internal/analysis/kill_attribution.go` |
| Repository de lecture agrégée | `internal/platform/duckdb/weapon_kills_repo.go` |
| Résolution label / rôle (metadata) | `internal/platform/duckdb/weapon_resolver.go` |
| Schéma (table + vue) | `internal/games/halo_infinite/migrations/steps_shared_core.go` (`add_weapon_kills`, `add_weapon_kills_reconciled_as`) |
| Conversion append-only | `internal/migration/steps_shared_append_only_weapon_kills.go` |
| Backfill CLI | `apps/go-api/cmd/levelup` — `backfill --weapons` |
| Seeding des labels | `apps/go-api/cmd/seed-weapon-labels` |

---

## Pipeline

Point d'entrée : `sync.BackfillWeaponKillsForMatchAll(ctx, client, sharedDB, matchID)`
dans `internal/sync/backfill_weapons.go` (la variante `...ForMatch` traite un
seul joueur ; `...ForMatches` est la méthode batch sur `SyncEngine` qui acquiert
le lease).

Étapes :

1. **Télécharger le film** — `client.GetMatchFilm(ctx, matchID)` retourne les
   chunks binaires. Si le film a disparu (404/410), le match est marqué
   `weapon_kills_no_film` et ignoré (les films des matchs anciens expirent).
2. **Construire les timelines d'arme tenue** —
   `analysis.BuildWeaponTimelines(chunks)` produit, par chunk et par index de
   joueur, les octets de l'arme tenue au moment du snapshot du chunk
   (`Timeline`), plus un set de détection de swap par chunk (`SwapPIs`) et les
   intervalles temporels des chunks (`Timing`).
3. **Scanner les fire events** — `analysis.ScanFireEventsAll(...)` sur chaque
   chunk collecte les événements de tir de tous les joueurs avec timestamps
   estimés.
4. **Charger les kills** — `getAllKillsForMatch` lit `highlight_events`
   (`event_type LIKE '%kill%'`, avec flags melee/grenade) depuis la DB partagée,
   plus un mapping `xuid -> player_index` (`getXuidToPI`, ordonné par
   `team_id, rank`).
5. **Corréler** — `analysis.CorrelateKillsGlobal(...)` attribue chaque kill à
   une arme (cf. [Chemins d'attribution](#chemins-dattribution)).
6. **Réconcilier** — `analysis.ReconcileAPIAggregates` ajuste attributions et
   confiance contre les totaux de kills API de `match_participants`.
7. **Écrire** — `InsertWeaponKills` écrit une génération append-only de lignes
   par `(match_id, xuid)` ; `MarkWeaponKillsDone` ne pose le bit registre
   **que si au moins une ligne a été insérée** (garde-fou ajouté après un
   incident de mai 2026 où poser le bit sur une extraction à 0 ligne a vidé
   silencieusement ~1010 matchs).

Les écritures sont sérialisées par le lease d'écriture partagé +
`MaxOpenConns(1)` ; la corrélation et le téléchargement du film sont
parallélisés network-only (`weaponBackfillParallelism = 24`).

---

## Chemins d'attribution

Chaque kill reçoit un `attribution_path` (constantes dans
`internal/analysis/kill_attribution.go`) :

| Chemin | Portée | Source | Remarque |
|--------|--------|--------|----------|
| `fire_event` | Par joueur, basé timing | Événements de tir scannés des chunks, associés au fire event précédent le plus proche dans la fenêtre de timing de l'arme | Précision maximale quand un fire event est trouvé |
| `formula_a` | Par joueur, basé snapshot | L'arme *tenue* par le joueur (`Timeline`) au chunk couvrant l'instant du kill, avec repli sur le chunk précédent | Plus grossier (granularité chunk) ; dégradé à `medium` si un swap a été détecté dans ce chunk |
| `none` | Repli | Ni fire event ni snapshot d'arme tenue exploitable | Stocké avec `confidence = none` |

Les kills melee et grenade sont identifiés via le type d'event de
`highlight_events` et ne sont **pas** attribués par Weapon ID via le film ;
leurs décomptes par match viennent des colonnes `melee_kills` / `grenade_kills`
de `match_participants` (cf. [Lecture](#lecture--v_weapon_kills-et-labels)).

---

## Niveaux de confiance

Constantes dans `internal/analysis/weapon_correlation.go`
(`confidenceHigh/Medium/Low/None`). `ComputeConfidence(weaponID, deltaMS)`
utilise la fenêtre de timing de l'arme (`GetTiming`, depuis `weapon_data.go`) :

| Valeur | Signification |
|--------|---------------|
| `high` | Fire event dans le `SwapMS` de l'arme, ou snapshot confirmé réconcilié avec le total API |
| `medium` | Correspondance mais ambiguïté de swap/timing (delta dans `TravelMax`) |
| `low` | Correspondance faible (delta au-delà de `TravelMax`) |
| `none` | Kill non attribuable |

`ReconcileAPIAggregates` (`weapon_reconciliation.go`) compare les totaux
attribués à `match_participants.kills` et promeut/dégrade la confiance et les
Weapon IDs pour que les décomptes par arme restent cohérents avec les totaux
autoritatifs de l'API.

---

## Structure d'un Weapon ID (WID)

Un WID, ce sont les 8 octets d'arme filmshell lus comme un **`uint64`
big-endian** (`hexToUint64` dans `internal/analysis/weapon_data.go`). DuckDB le
stocke en `UBIGINT` — certains WID réels (ex. `f408190f42c9679f`) ont le bit 63
activé et dépassent `2^63`, raison pour laquelle l'écriture caste une chaîne
décimale en `UBIGINT` plutôt que de binder un `uint64` Go (le driver duckdb-go
rejette les uint64 à bit de poids fort activé).

Structure des 8 octets :

- **Octets 1-4 (32 bits de poids fort) : l'identité de l'arme** — unique par
  type/variante.
- **Octets 5-8 (32 bits de poids faible) : un suffixe famille/variante.** Le
  suffixe commun `42c9679f` (`CommonWeaponSuffix` dans `weapon_data.go`) couvre
  la plupart des armes standard ; les familles spéciales partagent leurs octets
  de poids fort mais diffèrent par le suffixe :

| Famille | Octets de poids fort (identité) | Comportement |
|---------|----------------------------------|--------------|
| Energy Sword | `4ff3937e` | Même identité, suffixe différent par skin (Duelist, Bloodblade, Infected) |
| Gravity Hammer | `841ac5e5` | Même identité, suffixe différent par variante (Diminisher of Hope, Rushdown) |

Les variantes cosmétiques sont repliées sur leur arme canonique via
`WeaponFusionMap` / `WeaponFusionMapID`. Les IDs sentinels `0` (grenade),
`1` (melee), `2` (véhicule) sont réservés et exclus de l'agrégation des armes.

La liste autoritative des WID (hex confirmé -> nom) vit dans `weaponEntries` au
sein de `weapon_data.go`, et est dupliquée en notes de recherche dans
`.ai/REFERENCE_WEAPON_IDS.md`.

---

## Stockage — `weapon_kills` (append-only)

La table `weapon_kills` vit dans la DB partagée
(`data/warehouse/shared_matches_v2.duckdb`). Colonnes de base
(`steps_shared_core.go`, migrations `add_weapon_kills` +
`add_weapon_kills_reconciled_as`) :

| Colonne | Type | Description |
|---------|------|-------------|
| `match_id` | VARCHAR | UUID du match Halo |
| `xuid` | VARCHAR | XUID du tueur |
| `time_ms` | INTEGER | Horodatage du kill (ms depuis le début du match) |
| `weapon_id` | UBIGINT | WID attribué via film |
| `reconciled_as` | UBIGINT | Override réconcilié via API (NULL sinon) |
| `delta_ms` | INTEGER | Écart fire event -> kill (NULL si chemin snapshot) |
| `confidence` | VARCHAR | `high` / `medium` / `low` / `none` |
| `attribution_path` | VARCHAR | `fire_event` / `formula_a` / `none` |
| `swap_detected` | BOOLEAN | Un swap d'arme a eu lieu près du kill |
| `delayed_damage` | BOOLEAN | Le vol du projectile a pu gonfler le delta |
| `player_index` | INTEGER | Index de joueur film résolu |

**Append-only (durcissement #23046).** La table a été convertie en append-only
(`internal/migration/steps_shared_append_only_weapon_kills.go`) : deux colonnes
ajoutées — `generation_id BIGINT` et `written_at TIMESTAMP`. Chaque appel à
`InsertWeaponKills` alloue une génération depuis `weapon_kills_generation_seq`
partagée par toutes les lignes de ce write `(match_id, xuid)`, et fait un INSERT
(pas de `DELETE`). Ceci a éliminé l'ancien DELETE-puis-INSERT qui déclenchait le
bug d'index ART de DuckDB sur la DB partagée multi-writer. Il n'y a pas de PK
technique ; la justesse vient de la séquence de génération + le superséding à la
lecture, pas de contraintes au niveau ligne.

> Certains commentaires inline de `backfill_weapons.go` décrivent encore le
> modèle legacy DELETE-puis-INSERT — la conversion append-only les remplace.

---

## Lecture — `v_weapon_kills` et labels

Les lecteurs ne lisent jamais `weapon_kills` directement. La surface de lecture
canonique est la vue **`v_weapon_kills`**, qui :

- ajoute `effective_weapon_id = COALESCE(reconciled_as, weapon_id)`, et
- ne garde, par `(match_id, xuid)`, **que les lignes de la dernière
  génération** (`DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY
  generation_id DESC)`).

`internal/platform/duckdb/weapon_kills_repo.go` (`LoadWeaponKillsAggregated`)
agrège `v_weapon_kills` par `(xuid, effective_weapon_id)` avec `COUNT(*)`, en
excluant les sentinels `effective_weapon_id NOT IN (0,1,2)`. Quand
`IncludeGrenadeMelee = true`, il fait un UNION ALL des totaux `grenade_kills` /
`melee_kills` de `match_participants` sous les IDs sentinels `0` et `1`.

Les labels (EN/FR) et rôles sont attachés côté Go depuis
`metadata.weapon_labels` (`weapon_id UBIGINT`, `name_en`, `name_fr`) via
`weapon_resolver.go` — la DB metadata est séparée, elle ne peut donc pas être
jointe en SQL à la DB partagée. Le nom d'affichage est `name_fr` d'abord, puis
`name_en`.

Si `weapon_kills` / `v_weapon_kills` est absente (ex. un titre qui ne la
supporte pas), le repo retourne `games.ErrCapabilityNotSupported`.

---

## Lancer le backfill

Depuis `apps/go-api` (toolchain CGO requise pour le driver DuckDB) :

```bash
# Backfill weapon_kills pour les matchs manquants de tous les joueurs éligibles
go run ./cmd/levelup backfill --weapons

# Retraiter les matchs même déjà marqués done
go run ./cmd/levelup backfill --weapons --force
```

La sélection des matchs est pilotée par bitmask sur
`match_registry.backfill_completed` : bit `1<<21` = weapon_kills fait, bit
`1<<22` = no-film. Sans `--force`, les matchs ayant l'un de ces bits sont
ignorés (`findMissingWeaponMatches`).

Si `metadata.weapon_labels` est vide (certaines DB prébuilties), réparer-la
(arrêter le serveur API d'abord — il tient metadata.duckdb en RW) :

```bash
go run -tags cgo ./cmd/seed-weapon-labels
```

---

## Ajouter / résoudre un Weapon ID

Quand une nouvelle arme arrive, ou qu'un WID non résolu est positivement
identifié :

1. Ajouter l'entrée dans `weaponEntries` dans
   `apps/go-api/internal/analysis/weapon_data.go` (hex -> nom). La placer dans
   le bon groupe (standard / famille Energy Sword / famille Gravity Hammer /
   grenade). Si la classe d'arme est nouvelle, ajouter une entrée
   `WeaponTimingByName`.
2. Ajouter le label EN/FR à `metadata.weapon_labels` (seed via
   `cmd/seed-weapon-labels`, ou insertion directe) pour que la lecture puisse
   l'afficher.
3. Relancer `go run ./cmd/levelup backfill --weapons --force` pour les matchs
   concernés si l'on veut ré-attribuer les lignes existantes ; les nouveaux
   matchs prennent le mapping automatiquement.

> Attention : ne pas ajouter de WID de façon spéculative par similarité de
> famille d'octets. N'ajouter que des WID positivement identifiés via des
> sources d'assets fiables ou une inspection directe de chunks — une mauvaise
> entrée provoque des attributions erronées sur tous les matchs.

---

## Limites connues

- **Les films expirent.** Les matchs anciens n'ont plus de film (404/410) et ne
  pourront jamais être attribués ; ils sont marqués `no_film`.
- **`fire_event` est par tir, `formula_a` est un snapshot par chunk.**
  L'attribution par snapshot est grossière et dégradée sur les swaps détectés
  dans un chunk.
- **La réconciliation aligne les totaux, pas les kills individuels.** La
  réconciliation API garantit que les décomptes par arme correspondent au total
  de kills API, pas que chaque kill soit correctement attribué.
- **Le type d'arme melee/grenade n'est pas résolu via le film** au-delà de la
  distinction `MELEE`/`GRENADE` ; les décomptes viennent de
  `match_participants`.
- L'attribution est donc une **reconstruction statistique**, adaptée aux
  répartitions d'usage d'arme et à l'arme favorite, pas à des affirmations
  forensiques par kill.

---

## Approfondissement : rétro-ingénierie du film

La structure binaire des chunks du film, le décodage dead-state / kill-feed, et
le catalogue des Weapon IDs sont documentés séparément et **non dupliqués ici** :

- `.ai/RESEARCH_THEATER_RE.md` — structure des chunks film/théâtre, notes de
  rétro-ingénierie dead-state et kill-feed.
- `.ai/REFERENCE_WEAPON_IDS.md` — catalogue des Weapon IDs et recherche de
  résolution.
