package main

// score.go — mode -score : décode + imprime (et persiste si -write) la timeline
// de score per-équipe des modes à objectif (Strongholds, KOTH) via le package
// objectivescore (ancre bit-level 0x7B6 dans le payload TYPE_2).
//
// Pour chaque match résolu (-match/-all), si game_variant ∈ {strongholds, koth} :
// charge les chunks du cache disque (décompressés) + les finals DB (calibration),
// appelle DecodeScoreTimeline, imprime la courbe (t, team0, team1) et compare le
// dernier point décodé au final DB. -write persiste via ObjectiveScoreRepo
// (DELETE-then-INSERT par match ; DRY-RUN par défaut ; copie temp si la DB est
// verrouillée, comme l'existant).

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"

	"levelup/go-api/internal/analysis/objectivescore"
	"levelup/go-api/internal/platform/duckdb"
)

// runScore traite le panel en mode SCORE.
func runScore(ctx context.Context, conn *conn, cfg runConfig, ids []matchRef) error {
	for _, m := range ids {
		if err := processMatchScore(ctx, conn, cfg, m); err != nil {
			fmt.Printf("[%s] ERREUR: %v\n\n", m.short, err)
		}
	}
	return nil
}

// processMatchScore décode + imprime (et persiste si write) la timeline d'un match.
func processMatchScore(ctx context.Context, c *conn, cfg runConfig, m matchRef) error {
	reg, ok, err := loadRegistry(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("absent de match_registry")
	}
	mode := objectivescore.ClassifyMode(reg.variant)
	if mode == objectivescore.ModeUnsupported {
		fmt.Printf("[%s] %s — variant=%q : mode non géré par objectivescore (skip)\n\n",
			m.short, m.full, reg.variant)
		return nil
	}

	chunks, err := loadScoreChunks(cfg.cacheDir, m.short)
	if err != nil {
		return err
	}
	frames := objectivescore.DecodeScoreTimeline(reg.variant, chunks, reg.team0Score, reg.team1Score)
	printScoreTimeline(m, reg, frames)

	if cfg.write {
		if err := writeScoreFrames(ctx, c, m.full, frames); err != nil {
			return err
		}
		fmt.Printf("  [write] %d frame(s) persistées sur match_objective_score_timeline\n", len(frames))
	}
	fmt.Println()
	return nil
}

// loadScoreChunks lit + décompresse les chunks du cache disque vers []ChunkInput.
func loadScoreChunks(cacheDir, short string) ([]objectivescore.ChunkInput, error) {
	src, ok, err := newDiskFilmSource(cacheDir, short)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("manifest cache absent (%s)", short)
	}
	var chunks []objectivescore.ChunkInput
	for _, meta := range src.Chunks() {
		raw, ok := src.ChunkData(meta.Index)
		if !ok {
			continue
		}
		chunks = append(chunks, objectivescore.ChunkInput{
			Data: decompressScoreChunk(raw), StartMS: meta.StartMS, ChunkType: meta.ChunkType,
		})
	}
	return chunks, nil
}

// decompressScoreChunk = zlib si magic 0x78, sinon raw (miroir des décodeurs film).
func decompressScoreChunk(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		if z, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			defer z.Close()
			if d, e2 := io.ReadAll(z); e2 == nil {
				return d
			}
		}
	}
	return raw
}

// printScoreTimeline imprime la courbe de score + la comparaison final vs DB.
func printScoreTimeline(m matchRef, reg *registryRow, frames []objectivescore.ScoreFrame) {
	fmt.Printf("[%s] %s — variant=%q (mode=%s)\n", m.short, m.full, reg.variant,
		modeLabel(objectivescore.ClassifyMode(reg.variant)))
	if len(frames) == 0 {
		fmt.Println("  Aucune frame de score décodée (chunks non ancrés / film absent).")
		return
	}
	fmt.Println("  Timeline score (t, team0, team1) :")
	for _, f := range frames {
		fmt.Printf("    t=%-8.1fs  team0=%-4d team1=%-4d src=%s conf=%s\n",
			float64(f.TimeMS)/1000.0, f.Team0, f.Team1, f.Source, f.Confidence)
	}
	printScoreCompare(reg, frames[len(frames)-1])
}

// printScoreCompare confronte le dernier point décodé au final DB.
func printScoreCompare(reg *registryRow, last objectivescore.ScoreFrame) {
	fmt.Println("  --- Comparaison final décodé vs DB ---")
	if !reg.hasScores {
		fmt.Printf("    Final décodé : %d-%d — DB sans score final (NULL)\n", last.Team0, last.Team1)
		return
	}
	match := last.Team0 == reg.team0Score && last.Team1 == reg.team1Score
	fmt.Printf("    Final décodé : %d-%d\n", last.Team0, last.Team1)
	fmt.Printf("    Final DB     : %d-%d\n", reg.team0Score, reg.team1Score)
	fmt.Printf("    MATCH=%s\n", okStr(match))
}

// modeLabel rend un libellé court du Mode objectivescore.
func modeLabel(mode objectivescore.Mode) string {
	switch mode {
	case objectivescore.ModeStrongholds:
		return "strongholds"
	case objectivescore.ModeKOTH:
		return "koth"
	default:
		return "unsupported"
	}
}

// writeScoreFrames persiste via ObjectiveScoreRepo (PlayerDB minimal : seul Shared
// est requis ; SharedReadDB() retombe sur LegacySharedReader en RW).
func writeScoreFrames(ctx context.Context, c *conn, matchID string, frames []objectivescore.ScoreFrame) error {
	if c.rwDB == nil {
		return fmt.Errorf("connexion non-RW (write impossible)")
	}
	pdb := &duckdb.PlayerDB{Shared: c.rwDB}
	repo := duckdb.NewObjectiveScoreRepo(pdb)
	return repo.WriteMatch(ctx, matchID, frames)
}
