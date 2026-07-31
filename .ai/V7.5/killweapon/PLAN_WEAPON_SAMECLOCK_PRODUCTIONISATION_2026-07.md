# PLAN — Productionisation de l'attribution d'arme SAME-CLOCK (damage-source)

> Créé 2026-07-11. Branche cible : `feat/filmdec-continuation` (worktree filmdec-continuation) —
> ou une branche dédiée `feat/weapon-sameclock` depuis main si on repart propre.
> Contrat d'exécution : skill `plan-execution` (ordre strict, statuer chaque item, zéro report exécutable).
> ⚠️ Merge main = **deploy prod** sur `weapon_kills` : prévenir l'utilisateur AVANT le merge.

## 0. Contexte & justification

L'attribution d'arme de PROD (`internal/sync/backfill_weapons.go` → `internal/analysis/weapon_correlation.go`)
utilise la méthode **fire-events** (réclame le fire event le plus proche dans [t-5s, t] pour le player_index du
tueur). Cette méthode est la voie **legacy** (portée de `_global_correlation.py`), contraire à la doctrine
« l'arme = la SOURCE DE DÉGÂT fatale » (mémoires `feedback_no_fire_events_weaponv3`, `project_killfeed_damage_source_goal`).

La méthode **damage-source same-clock** a été RE'd et **VALIDÉE** dans le tool `cmd/tmp_kwval` (RE_LOG §7ter.12) :
arme = dernier record de dégât 0xd2 (attaquant==tueur) avant l'instant du kill, cross-paquet.
- Couverture-ARME 70% → **~82%** (+5 à +12 pts sur 4 films).
- **Accuracy per-paire 94% vs capture live** (9b191a7f, 65/69) ; distribution ⊆ live (capte le Fuel Rod SPNKr,
  zéro sur-attribution). C'est l'INVERSE de l'expérience full-scan `augcov2` (rejetée, 71%, ratait le Fuel Rod).
- Prérequis prouvé : **player_index == slot == index-local du film par IDENTITÉ** (RE_LOG §7ter.11, outil `idxmap`).

**But** : porter la méthode damage-source same-clock dans le pipeline de prod, remplacer le fire-events,
sans régression de couverture ni d'accuracy, en préservant les invariants d'écriture ART (`weapon_kills`).

### Critère de succès (mesurable)
1. Sur les 4 films de référence (000d5950, 00502e52, 0014603f, 9b191a7f), la sortie prod ≡ la sortie du tool RE
   `augcovsc` (couverture-ARME et distribution d'armes identiques à ±1 kill).
2. Accuracy per-paire ≥ 90% vs live sur 9b191a7f (le tool donne 94%).
3. `go test ./...` + `go test -tags=integration -p 1 ./...` verts (persist/sync touchés).
4. Zéro `INSERT ... ON CONFLICT DO UPDATE` concurrent introduit sur `weapon_kills` ; écritures sérialisées
   comme aujourd'hui (lease shared + MaxOpenConns(1), DELETE-then-INSERT par match_id).

## 1. Pièces à PORTER (tool RE `cmd/tmp_kwval` → `internal/analysis/`)

Toutes en algos PURS (analysis/, 0 accès DB, 0 IO) — entrée = `[]analysis.ChunkData`, sortie = structs.

| # | Source RE (tmp_kwval) | Cible analysis/ | Rôle |
|---|---|---|---|
| P1 | itération paquets (marker/type/size/ts) dans `collectRaw` (main.go:100-126) | `film_packets.go` `IterFilmPackets(chunk) []Packet{Marker,Type,Payload,TS}` | découpe un chunk en paquets type-0 |
| P2 | `parsePreamble` (deserlen.go:1016) + `dmgRecord` | `damage_source.go` `ParseDamageRecord(pl []byte) (DamageRecord,bool)` | 0xd2 base=24 → attaquant(slot) + family(u32) |
| P3 | `weaponName` (deserlen.go:976) + `firearmDmgMk` (killfeed.go:628) | `damage_source.go` (réutiliser le loader `weapon_labels` prod existant) | family u32 → nom d'arme |
| P4 | `keReadOpt`/`validKE`/`decodeKE`/`keCandidates` (deserlen.go:600-635,1129) | `kill_event_film.go` `DecodeKillEvents(pkt) []FilmKill{Killer,Victim,TS}` | 0xE6 : tueur/victime (slot) via R1-gate+R5, cap<16 |
| P5 | `runAugCovSC` corrélation (killfeed.go) | `weapon_correlation_sameclock.go` `CorrelateKillsSameClock(kills, damages, xuidToPI)` | pour chaque kill (pi,ts) : arme = dernier dégât attaquant==pi, ts le plus proche avant |

Notes :
- **Roster** : PAS de portage — `player_index` (déjà fourni via `xuidToPI` dans le pipeline) EST le slot
  (identité prouvée §7ter.11). La sortie same-clock est indexée par player_index directement.
- **Kills** : deux options (à trancher en P5) : (a) kills du FILM (P4, 0xE6) mappés aux kills DB par
  (killer,victim,ts) — comme le tool RE ; (b) kills de la DB (`getKillsForPlayer`) directement, en alignant
  time_ms sur l'horloge frame (déjà résolu par le pipeline fire-events actuel — RÉUTILISER cet alignement).
  **Recommandation : (b)** pour réutiliser l'infra clock existante et éviter de re-décoder/mapper les kills ;
  P4 (kills film) reste utile en validation croisée mais pas dans le chemin de prod.

## 2. Phases (ordre strict, risque croissant)

### Phase 1 — Portage algos purs + tests unitaires (branche feature, ZÉRO risque prod)
- [ ] P1 `IterFilmPackets` + test (un chunk connu → N paquets, marqueurs attendus).
- [ ] P2 `ParseDamageRecord` + test (records 0xd2 d'un chunk de référence → attaquant/family attendus,
      comparés aux valeurs du tool RE).
- [ ] P3 `weaponName` branché sur le loader `weapon_labels` de prod (identifier lequel : `metadata.duckdb`
      `weapon_labels`, chargé par… — à repérer dans le pipeline actuel) + test.
- [ ] P5 `CorrelateKillsSameClock` (pur) + test (kills + damages synthétiques → attributions attendues,
      dataset hétérogène réaliste — mémoire `feedback_integration_tests_realistic_datasets`).
- **Gate Phase 1** : `go test ./internal/analysis/...` vert ; un test de PARITÉ compare la sortie
  `CorrelateKillsSameClock` sur les 4 films de référence à la sortie `augcovsc` du tool RE (±1 kill).

### Phase 2 — Câblage sync (branche feature, ZÉRO deploy)
- [ ] `backfill_weapons.go` : remplacer `BuildWeaponTimelines`+`ScanFireEventsAll`+`CorrelateKillsGlobal` par
      `ParseDamageRecord`(scan chunks) + `CorrelateKillsSameClock`. Conserver `getKillsForPlayer`,
      `ReconcileAPIAggregates` (si pertinent), `InsertWeaponKills`, `MarkWeaponKillsDone`, le lease/parallélisme.
- [ ] Logging : `slog.InfoContext(ctx, "weapon_sameclock", "match_id", …, "covered", …, "named", …)` ;
      `slog.ErrorContext` sur échec décode ; jamais d'erreur avalée (mémoire logger AVANT dégradation).
- [ ] Supprimer le code fire-events DÉBRANCHÉ (weapon_correlation.go fire-event path, ScanFireEventsAll,
      BuildWeaponTimelines) AVEC ses tests (`weapon_correlation_test.go`, `kill_attribution_test.go` parties
      fire-event) et imports — doctrine « 0 code mort » (CLAUDE.md). Garder ce qui reste utilisé.
- **Gate Phase 2** : `go test ./...` + `go vet ./...` verts ; **`go test -tags=integration -p 1 ./...`**
      (persist/sync touchés — OBLIGATOIRE, sérialisé, code sortie 0 vérifié) ; CI branche verte
      (`gh run list --branch`).

### Phase 3 — Validation & livraison
- [ ] Rejouer un backfill weapon sur un échantillon de matchs réels (dev local) ; comparer la distribution
      d'armes avant/après (fire-events → same-clock) : attendue = +couverture, distribution plausible.
- [ ] Entrée `../../thought_log.md` + MAJ RE_LOG §7ter.12 (statut : productionisé).
- [ ] delivery-checklist (skill) : go/no-go complet.
- [ ] **PRÉVENIR L'UTILISATEUR** avant merge main (deploy prod auto sur `weapon_kills`). Merge = étape gated,
      jamais autonome.

## 3. Architecture (couches Go — arch-rules)
- Algos purs (P1-P5) → `internal/analysis/` (entrée ChunkData/structs, 0 DB, 0 slog sauf debug).
- Orchestration (télécharge film, scanne, corrèle, écrit) → `internal/sync/backfill_weapons.go`.
- Écritures `weapon_kills` : INCHANGÉES (DELETE-then-INSERT par match_id, sérialisées lease+MaxOpenConns(1)).
  Ne PAS introduire d'UPSERT concurrent (invariant ART).
- Title-agnostic : le décodage film est **Halo Infinite-only** (capability, pas slug==) — garder derrière la
  capability existante du pipeline weapon (le pipeline actuel est déjà HI-only ; ne pas régresser).

## 4. Tests par couche
- `analysis/` : unit tests purs P2/P4/P5 + **test de parité vs tool RE** (4 films, la preuve).
- `sync/` : test d'orchestration avec fixtures (mock HaloClient GetMatchFilm) — réutiliser
  `sync_pipeline_fixture_test.go` / `backfill_weapons_test.go` existants, adaptés au nouvel algo.
- Intégration : gate `-tags=integration -p 1` (écritures weapon_kills).

## 5. Découvertes / à trancher pendant l'exécution (ne PAS élargir le périmètre)
- **D1** : localiser le loader `weapon_labels` de prod (P3) — le pipeline fire-events actuel nomme-t-il déjà
  les armes via un map u32→nom ? Réutiliser.
- **D2** : alignement horloge kill DB time_ms ↔ film frame ts (P5 option b) — extraire la logique du pipeline
  fire-events actuel (il aligne déjà kills et fire events).
- **D3** : `ReconcileAPIAggregates` (ajustement depuis les counts API) — pertinent pour le same-clock ? Le
  garder si la validation le montre utile, sinon le débrancher proprement (avec ses tests).
- **D4** : marqueurs de dégât (`firearmDmgMk` = 0xD2/C0/C2/C3/CA/E9) — mêlée/grenade sont des events distincts
  (0xD3/0x4c0c00, cf. tool RE `scanEvents`) : décider si le same-clock couvre firearm SEUL (et garder la
  logique mêlée/grenade existante) ou unifie. **Recommandation : firearm same-clock d'abord, mêlée/grenade
  inchangés** (périmètre fermé, non-régression).

## 6. Reprise / exécutabilité agent
- Preuve de référence : `cd apps/go-api && CGO_ENABLED=0 go build -o $TEMP/kwval.exe ./cmd/tmp_kwval` puis
  `$TEMP/kwval.exe <film> augcovsc` (couverture + accuracy + distribution vs live). Le test de parité Phase 1
  doit reproduire ces chiffres.
- Films de référence : 000d5950, 00502e52, 0014603f, 9b191a7f (seul 9b191a7f a la capture live d'arme).
- Statuts d'item : `[x]` fait / `[~]` couvert ailleurs (réf) / `[!]` non traité (justif). Aucune case vide à
  la clôture. Étape N close+gate passé avant N+1.
