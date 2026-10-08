# Diagnostic — cartes en double de `/tactical/maps` (2026-10-07)

Diagnostic en lecture seule (agent Opus, copie de `shared_matches_v2.duckdb` du 07/10 21:25,
traductions de `metadata-prebuilt.zip` du 02/10). Aucune écriture.

## 1. Chaîne de lecture

`api/handlers/tactical.go:153-162` -> `service/tactical_service.go:97-127` (`MapsPlayed`, plancher
`Matchs < 10`, `domain/tactical.go:494`) -> `platform/duckdb/tactical_repo.go:89-147`
(`QTacticalMaps`, `match_registry ⨝ match_participants`). **Clé : `GROUP BY mr.map_id, mr.map_name`**
(`tactical_repo.go:120`) : une rangée par variante de `map_name` (nom, NULL, ou map_id recopié).
Le libellé affiché est résolu par `map_id` (`map_labels.go`), d'où des doublons au même nom.

## 2. Nature

Seul `map_name` diffère (même `map_id`, même `map_version_id`). Base : 9 216 matchs, 298 cartes,
67 cartes à plusieurs noms ; 34 rangées NULL ; 1 338 rangées `map_name = map_id`.
JGtm : 103 vignettes pour 77 cartes (23 en double dont 3 en triple, 26 fantômes) ; 25 matchs NULL
(joués 23/04-05/05/2026, inscrits 06/05 14:49) + 6 matchs nom = id (10/06/2026).
Autres profils : Madina97294 140/114, Chocoboflor 93/68, Nuzzles 233/213 (1 053 matchs touchés).

## 3. Causes

Seul écrivain : `persist/shared_persister.go:155` ; sans nom API, `sync/transforms.go:81-84` recopie l'id.
- **A (NULL, 34 rangées)** : écrites avec l'id avant `EnrichRegistryFromMetadata` (09/05), puis
  `cmd/repair_data_consistency` chantier 2 (08/05) a posé `map_name = NULL` / `pair_name = NULL`
  (`main.go:199`, `:206`). Outil toujours dans HEAD. `sync/backfill_registry_names.go` ne traite que
  nom = id, pas NULL.
- **B (09-10/06, 7 rangées)** : chemin V2 sans enrichissement du nom (`sync/engine_v2bridge.go:87-91`),
  corrigé le 13/06 (`3c8a005c4`).
- **C (Nuzzles 16/09, 1 331 rangées)** : `levelup sync-full` via pool ; `cmd/levelup/pool_engine.go:90`
  (`newPooledEngine`) sans `.WithAssetNameResolution(pool)` (seul `scheduler/auto_sync_engine.go:125`
  le branche). 1 332/1 332 rangées nom = id écrites avant la traduction, 5 848/5 848 nommées après.
- **Défaut commun** : `ResolveUnresolvedAssetNames` remplit `asset_translations` mais ne réécrit
  jamais `match_registry` ; plafond `defaultMaxAssets = 64` (`assetnames/resolver.go:33`) figerait un
  premier passage chargé (déduction).
- Plus produit depuis le 17/09 (81 rangées, toutes nommées), mais une sync CLI d'un profil neuf
  reproduirait C.

## 4. Effets

- Tactique (JGtm) : 103 -> 77 vignettes, 61 -> 33 sous le plancher ; Elevation (11) et Critical
  Dewpoint (10) à tort sous le plancher ; comptes sous-estimés (Illusion 54/56) alors que le plan
  lit l'univers par carte (`tactical_repo_univers.go:100`) ; vignettes fantômes peintes avec la
  chaleur de toute la carte (`tactical_service_vignettes.go:48`) ; clés React en double
  (`TacticalMapsColumn.tsx:84`, `:142`, `:89`).
- Rejeu : 25 matchs de JGtm à `pair_name` NULL (dont 7 Bases, 6 CTF) sans calque d'objectifs ni
  niveaux d'armes (`replay_map_objectives.go:63`, `replay_weapon_tiers.go:41`).
- Explorateur, filtres, répartitions par carte : non touchés (résolution par map_id).
- Lectures dormantes au nom brut : `match_history_repo.go:321`, `queries_squad.go:344`,
  `replay_map_repo.go:116` (`LIMIT 1` sans tri).

## 5. Correction recommandée (ordre : c, a1/a2/a4, b, a3)

- **(c) Garde à la lecture** : `tactical_repo.go` groupe sur `map_id` seul, nom par
  `arg_max(map_name, début canonique) FILTER (WHERE map_name IS NOT NULL AND map_name <> map_id)` ;
  tests dépôt `:memory:` + service (plancher sur compte fusionné) ; garde-rail archlint interdisant un
  GROUP BY sur `map_name`/`pair_name`/`playlist_name`/`game_variant_name` hors clé par match.
- **(a) Source** : (1) `pool_engine.go:90` + `.WithAssetNameResolution(pool)` (+ vérifier
  `cmd_backfill.go`) ; (2) pas de plafond à la première écriture (`assetnames_wiring.go:59`, `:106`) ;
  (3) convergence : persisteur `registry_names_persister.go` qui réinscrit les noms après balayage,
  une rangée à la fois, `WHERE match_id = ? AND (x_name IS NULL OR x_name = x_id)` ;
  (4) retirer le chantier 2 de `repair_data_consistency` (ou l'outil).
- **(b) Données** : étendre `backfill_registry_names.go` (NULL ou = id, une rangée à la fois, pas
  `mode_category`, mode simulation) ; serveur arrêté, sauvegarde, relance à vide (idempotence).
  Taille : carte 1 372, paire 2 072 (1 566 + reconstruction), playlist 2 300, variante 1 113 ;
  3 348 matchs. Colonnes non indexées : risque ART faible ; risque principal = écrivain concurrent.

## Questions ouvertes (user)

1. Prod : mêmes rangées ? (A et C sont locaux) — diagnostic lecture seule VPS avant toute réparation.
2. Registre porteur de noms justes (b + a3) ou identifiants seuls, toutes lectures par traductions ?
3. Supprimer `cmd/repair_data_consistency` en entier ?
4. Arrêt du serveur local pour (b) ?
