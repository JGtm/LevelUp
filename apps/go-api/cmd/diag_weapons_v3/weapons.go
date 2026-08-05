package main

// weapons.go — mode ARMES du CLI diag_weapons_v3 (flag -weapons).
//
// Décode l'attribution d'arme v3 (weaponv3.BuildV3Attributions) à partir des
// chunks film cachés + des kills highlight_events + du roster, lit la baseline v2
// (table weapon_kills GELÉE, lecture seule), et imprime le rapport de comparaison
// §4 du plan (.ai/PLAN_WEAPON_ATTRIBUTION_V3.md). DRY-RUN par défaut ; -write
// persiste UNIQUEMENT weapon_kills_v3 (jamais weapon_kills v2).
//
// IO uniquement ici (chargement chunks/kills/roster/baseline + écriture v3) ; le
// calcul des métriques + l'impression vivent dans weapons_report.go.

import (
	"bytes"
	"compress/zlib"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// processMatchWeapons décode l'attribution d'arme v3 d'un match, imprime le
// rapport §4 (v3 vs v2 baseline + agrégats), agrège les métriques sur le panel,
// et persiste weapon_kills_v3 si cfg.write. Renvoie les compteurs du match pour
// l'agrégat panel.
func processMatchWeapons(ctx context.Context, c *conn, cfg runConfig, m matchRef, agg *panelAgg) error {
	reg, ok, err := loadRegistry(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("absent de match_registry")
	}

	chunks, err := loadV3Chunks(cfg.cacheDir, m.short)
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return fmt.Errorf("aucun chunk film décompressé (%s)", m.short)
	}

	kills, err := loadKillsV3(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}
	roster, err := loadRosterXuids(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}

	in := weaponv3.V3Input{
		MatchID:     m.full,
		Kills:       kills,
		RosterXuids: roster,
		Chunks:      chunks,
	}
	applyMeasureOverrides(&in, cfg)
	attrs := weaponv3.BuildV3Attributions(in)

	baseline, err := loadV2Baseline(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}
	aggregates, err := loadParticipantAggregates(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}

	rep := buildWeaponReport(attrs, baseline, aggregates)
	printWeaponReport(m, reg, rep)
	agg.add(rep)

	if cfg.write {
		n, err := writeV3Attributions(ctx, c, m.full, attrs)
		if err != nil {
			return err
		}
		fmt.Printf("  [write] %d ligne(s) persistées sur weapon_kills_v3\n", n)
	}
	fmt.Println()
	return nil
}

// applyMeasureOverrides applique les overrides de MESURE -firepi/-relax3 sur le
// V3Input (isolation des leviers §8/§9). Vide = défaut orchestrateur (auto-layout +
// relax3 par défaut). N'a aucun effet en prod : c'est de l'instrumentation CLI.
func applyMeasureOverrides(in *weaponv3.V3Input, cfg runConfig) {
	// NOTE: FirePi4High == zéro-valeur → l'orchestrateur le réinterprète en AUTO
	// (firePiAutoLayout). "4high" est donc équivalent à "auto" via cette entrée ; pour
	// FORCER vraiment le 4-bit sur un BTB il faudrait un sentinel, hors scope mesure.
	switch cfg.firePi {
	case "5span":
		in.FirePiLayout = weaponv3.FirePi5SpanBefore
	case "5highb5":
		in.FirePiLayout = weaponv3.FirePi5HighInB5
	case "5lowb5":
		in.FirePiLayout = weaponv3.FirePi5LowInB5
	case "", "auto", "4high":
		// défaut auto (selon taille roster) — laissé tel quel
	}
	switch cfg.relax3 {
	case "on":
		in.FireRelax3Set, in.FireRelax3 = true, true
	case "off":
		in.FireRelax3Set, in.FireRelax3 = true, false
	case "", "default":
		// défaut orchestrateur — laissé tel quel
	}
}

// loadV3Chunks charge tous les chunks du cache, les décompresse (zlib magic 0x78)
// et les convertit en []weaponv3.ChunkInput (avec chunk_type/start_ms/duration_ms
// du manifest). Le manifest est obligatoire (dégradation gracieuse si absent →
// liste vide). Tous les chunks sont chargés (le scanner melee scanne large) ;
// l'orchestrateur v3 filtre lui-même les chunks gameplay (type 2).
func loadV3Chunks(cacheDir, short string) ([]weaponv3.ChunkInput, error) {
	metas, ok, err := loadManifestChunks(cacheDir, short)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("manifest cache absent (%s)", short)
	}
	out := make([]weaponv3.ChunkInput, 0, len(metas))
	for _, mc := range metas {
		p := filepath.Join(cacheDir, "film_chunks", short, fmt.Sprintf("chunk_%02d.bin", mc.index))
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			continue // chunk manquant sur disque → on saute (dégradation gracieuse)
		}
		out = append(out, weaponv3.ChunkInput{
			Index:      mc.index,
			Data:       decompressFilmChunk(raw),
			StartMS:    mc.startMS,
			DurationMS: mc.durationMS,
			ChunkType:  mc.chunkType,
		})
	}
	return out, nil
}

// manifestChunkMeta — méta d'un chunk lu du manifest (incl. duration_ms, requis
// par l'orchestrateur v3 pour le FormulaA per-chunk).
type manifestChunkMeta struct {
	index      int
	chunkType  int
	startMS    int
	durationMS int
}

// loadManifestChunks lit film_manifests/<short>.json. (nil,false,nil) si absent.
func loadManifestChunks(cacheDir, short string) ([]manifestChunkMeta, bool, error) {
	mfPath := filepath.Join(cacheDir, "film_manifests", short+".json")
	raw, err := os.ReadFile(mfPath)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read manifest %s: %w", short, err)
	}
	var mf struct {
		Chunks []struct {
			Index      int `json:"index"`
			ChunkType  int `json:"chunk_type"`
			StartMS    int `json:"start_ms"`
			DurationMS int `json:"duration_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, false, fmt.Errorf("unmarshal manifest %s: %w", short, err)
	}
	out := make([]manifestChunkMeta, 0, len(mf.Chunks))
	for _, c := range mf.Chunks {
		out = append(out, manifestChunkMeta{
			index: c.Index, chunkType: c.ChunkType, startMS: c.StartMS, durationMS: c.DurationMS,
		})
	}
	return out, true, nil
}

// decompressFilmChunk décompresse un chunk zlib (magic 0x78) ; renvoie le brut si
// non compressé (robustesse). Même règle que objectiveevents.decompressChunk et
// le loadChunk de cmd/tmp_p2valid : l'orchestrateur v3 attend des Data DÉCOMPRESSÉS.
func decompressFilmChunk(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		if r, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
			defer r.Close()
			if inf, err := io.ReadAll(r); err == nil {
				return inf
			}
		}
	}
	return raw
}

// loadKillsV3 charge les kills de TOUS les participants via la MÊME requête que
// internal/sync getAllKillsForMatch (event_type LIKE '%kill%', GROUP BY
// xuid,time_ms, IsMelee/IsGrenade via LIKE '%melee%'/'%grenade%').
func loadKillsV3(ctx context.Context, db *sql.DB, matchID string) ([]analysis.Kill, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			xuid, time_ms,
			MAX(CASE WHEN LOWER(COALESCE(event_type, '')) LIKE '%melee%' THEN TRUE ELSE FALSE END) AS is_melee,
			MAX(CASE WHEN LOWER(COALESCE(event_type, '')) LIKE '%grenade%' THEN TRUE ELSE FALSE END) AS is_grenade
		FROM highlight_events
		WHERE match_id = ?
		  AND LOWER(COALESCE(event_type, '')) LIKE '%kill%'
		GROUP BY xuid, time_ms
		ORDER BY time_ms`, matchID)
	if err != nil {
		return nil, fmt.Errorf("loadKillsV3(%s): %w", matchID, err)
	}
	defer rows.Close()

	var kills []analysis.Kill
	for rows.Next() {
		var (
			xuid              string
			timeMS            *int
			isMelee, isGrenad bool
		)
		if err := rows.Scan(&xuid, &timeMS, &isMelee, &isGrenad); err != nil {
			continue
		}
		tms := 0
		if timeMS != nil {
			tms = *timeMS
		}
		kills = append(kills, analysis.Kill{
			MatchID: matchID, XUID: xuid, TimeMS: tms, IsMelee: isMelee, IsGrenade: isGrenad,
		})
	}
	return kills, rows.Err()
}

// loadRosterXuids charge les xuids NUMÉRIQUES du roster (team_id IS NOT NULL),
// pour la résolution pi bit-level (§8). Les bots (xuid non numérique, ex. bidXXXX)
// sont ignorés : ResolveBest ne peut pas bit-searcher un xuid non 64-bit.
func loadRosterXuids(ctx context.Context, db *sql.DB, matchID string) ([]uint64, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid FROM match_participants
		WHERE match_id = ? AND team_id IS NOT NULL`, matchID)
	if err != nil {
		return nil, fmt.Errorf("loadRosterXuids(%s): %w", matchID, err)
	}
	defer rows.Close()

	var out []uint64
	for rows.Next() {
		var xuid string
		if err := rows.Scan(&xuid); err != nil {
			continue
		}
		if n, ok := parseNumericXUID(xuid); ok {
			out = append(out, n)
		}
	}
	return out, rows.Err()
}

// parseNumericXUID extrait l'entier 64-bit d'un xuid. Accepte la forme nue
// ("2533...") et la forme "xuid(2533...)". Renvoie ok=false pour un bot.
func parseNumericXUID(xuid string) (uint64, bool) {
	s := xuid
	if len(s) > 6 && s[:5] == "xuid(" && s[len(s)-1] == ')' {
		s = s[5 : len(s)-1]
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// writeV3Attributions persiste les attributions v3, regroupées par xuid (le repo
// écrit par (match, joueur), DELETE-then-INSERT). Renvoie le total de lignes écrites.
func writeV3Attributions(ctx context.Context, c *conn, matchID string, attrs []weaponv3.AttributionV3) (int, error) {
	if c.rwDB == nil {
		return 0, fmt.Errorf("connexion non-RW (write impossible)")
	}
	pdb := &duckdb.PlayerDB{Shared: c.rwDB}
	repo := duckdb.NewWeaponKillsV3Repo(pdb)

	byXUID := map[string][]domain.WeaponKillV3Row{}
	for _, a := range attrs {
		byXUID[a.XUID] = append(byXUID[a.XUID], attributionToV3Row(a))
	}
	total := 0
	for xuid, rows := range byXUID {
		if err := repo.WriteMatch(ctx, matchID, xuid, rows); err != nil {
			return total, fmt.Errorf("WriteMatch(%s/%s): %w", matchID, xuid, err)
		}
		total += len(rows)
	}
	return total, nil
}

// attributionToV3Row convertit une weaponv3.AttributionV3 en domain.WeaponKillV3Row.
// ReconciledAs reste nil (la v3 a DROPPÉ la réconciliation API, cf. plan §2).
func attributionToV3Row(a weaponv3.AttributionV3) domain.WeaponKillV3Row {
	return domain.WeaponKillV3Row{
		TimeMS:          a.TimeMS,
		WeaponID:        a.WeaponID,
		ReconciledAs:    nil,
		DeltaMS:         a.DeltaMS,
		Confidence:      a.Confidence,
		AttributionPath: a.AttributionPath,
		SwapDetected:    false,
		DelayedDamage:   false,
		PlayerIndex:     a.PlayerIndex,
		SourceSignal:    a.SourceSignal,
		HighWeaponID:    a.HighWeaponID,
		KillingShotHit:  a.KillingShotHit,
		BurstFinal:      a.BurstFinal,
		ShotCounter:     a.ShotCounter,
	}
}
