package main

// positions.go — mode POSITIONS du CLI diag_weapons_v3.
//
// Pour chaque match (cache film présent), décode les positions joueurs keyframe
// (positions.DecodeKeyframePositions, §N de .ai/RESEARCH_THEATER_RE.md) depuis le
// cache disque (chunks BRUTS → décompressés zlib ici), affiche un résumé par
// match (nb positions, bornes x/y/z, split équipe best-effort), et persiste via
// PlayerPositionsRepo si -write (DELETE-then-INSERT par match).
//
// Décodage MATCH-LEVEL : pas d'attribution xuid (la delta-compression bloque
// l'index par joueur, cf. positions/positions.go). team est best-effort (-1 si
// inconnu).

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"math"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
)

// runPositions traite le panel en mode POSITIONS, match par match.
func runPositions(ctx context.Context, c *conn, cfg runConfig, ids []matchRef) error {
	for _, m := range ids {
		if err := processMatchPositions(ctx, c, cfg, m); err != nil {
			fmt.Printf("[%s] ERREUR: %v\n\n", m.short, err)
		}
	}
	return nil
}

// processMatchPositions décode + résume (et persiste si cfg.write) les positions
// d'un match.
func processMatchPositions(ctx context.Context, c *conn, cfg runConfig, m matchRef) error {
	if _, ok, err := loadRegistry(ctx, c.sqlDB, m.full); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("absent de match_registry")
	}
	src, ok, err := filmcache.Open(cfg.cacheDir, m.short)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("manifest cache absent (%s)", m.short)
	}

	chunks := collectPositionChunks(src)
	pos := positions.DecodeKeyframePositions(chunks)
	printPositionsSummary(m, pos)

	if cfg.write {
		if err := writePositions(ctx, c, m.full, pos); err != nil {
			return err
		}
		fmt.Printf("  [write] %d position(s) persistées sur match_player_positions\n", len(pos))
	}
	fmt.Println()
	return nil
}

// collectPositionChunks lit + décompresse chaque chunk du film et le présente sous forme de
// positions.ChunkInput (Data = contenu DÉCOMPRESSÉ).
//
// CE CHEMIN N'EST PAS CELUI DE LA CUISSON, et c'est délibéré (§7 de PLAN_CUISSON_PERF) :
// `analysis/positions` sert la vue de match côté serveur, il porte son propre marcheur de
// paquets et il n'est PAS migré vers `filmsource`. D'où l'inflate local, à l'allowlist datée du
// garde-rail (item 1.9). L'indice est ici la POSITION dans le manifeste, que
// [filmcache.Source.Chunk] et [filmcache.Source.Meta] partagent.
func collectPositionChunks(src *filmcache.Source) []positions.ChunkInput {
	meta := src.Meta()
	out := make([]positions.ChunkInput, 0, len(meta))
	for i, m := range meta {
		raw, err := src.Chunk(i)
		if err != nil {
			// Chunk absent du cache : le film est partiel. On le DIT (l'erreur porte le film et
			// le chemin) et on continue — c'est la dégradation d'avant, sans le silence.
			fmt.Printf("  chunk illisible : %v\n", err)
			continue
		}
		out = append(out, positions.ChunkInput{
			Data:      decompressZlib(raw),
			StartMS:   m.StartMS,
			ChunkType: m.ChunkType,
		})
	}
	return out
}

// decompressZlib renvoie le contenu décompressé d'un chunk film (zlib, magic
// 0x78). Un chunk non compressé est renvoyé tel quel.
func decompressZlib(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0x78 {
		if z, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
			if d, err2 := io.ReadAll(z); err2 == nil {
				return d
			}
		}
	}
	return raw
}

// printPositionsSummary affiche le résumé d'un match : nb positions, bornes
// x/y/z, split équipe best-effort.
func printPositionsSummary(m matchRef, pos []positions.PlayerPosition) {
	fmt.Printf("[%s] %s — %d position(s) full-state décodées\n", m.short, m.full, len(pos))
	if len(pos) == 0 {
		fmt.Println("  Aucune position décodée (mode non filmé, footer absent, ou chunks TYPE_2 vides).")
		return
	}
	b := boundsOf(pos)
	fmt.Printf("  Bornes : x[%.1f,%.1f] y[%.1f,%.1f] z[%.1f,%.1f]\n",
		b.xmin, b.xmax, b.ymin, b.ymax, b.zmin, b.zmax)
	t0, t1, unknown := positionTeamSplit(pos)
	fmt.Printf("  Split équipe (best-effort) : team0=%d team1=%d inconnu=%d\n", t0, t1, unknown)
}

// posBounds regroupe les bornes spatiales d'un ensemble de positions.
type posBounds struct {
	xmin, xmax, ymin, ymax, zmin, zmax float32
}

// boundsOf calcule les bornes min/max sur x/y/z. pos est supposé non vide.
func boundsOf(pos []positions.PlayerPosition) posBounds {
	b := posBounds{
		xmin: pos[0].X, xmax: pos[0].X,
		ymin: pos[0].Y, ymax: pos[0].Y,
		zmin: pos[0].Z, zmax: pos[0].Z,
	}
	for _, p := range pos {
		b.xmin, b.xmax = minF(b.xmin, p.X), maxF(b.xmax, p.X)
		b.ymin, b.ymax = minF(b.ymin, p.Y), maxF(b.ymax, p.Y)
		b.zmin, b.zmax = minF(b.zmin, p.Z), maxF(b.zmax, p.Z)
	}
	return b
}

// positionTeamSplit compte les positions par team (0/1/inconnu).
func positionTeamSplit(pos []positions.PlayerPosition) (t0, t1, unknown int) {
	for _, p := range pos {
		switch p.Team {
		case 0:
			t0++
		case 1:
			t1++
		default:
			unknown++
		}
	}
	return
}

// writePositions persiste les positions via PlayerPositionsRepo (PlayerDB minimal :
// seul Shared est requis, SharedReadDB() retombe sur LegacySharedReader).
func writePositions(ctx context.Context, c *conn, matchID string, pos []positions.PlayerPosition) error {
	if c.rwDB == nil {
		return fmt.Errorf("connexion non-RW (write impossible)")
	}
	pdb := &duckdb.PlayerDB{Shared: c.rwDB}
	repo := duckdb.NewPlayerPositionsRepo(pdb)
	return repo.WriteMatch(ctx, matchID, pos)
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float32) float32 {
	if math.IsNaN(float64(a)) {
		return b
	}
	if a > b {
		return a
	}
	return b
}
