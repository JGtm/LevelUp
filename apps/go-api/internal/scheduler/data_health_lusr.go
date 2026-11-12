package scheduler

// data_health_lusr.go — volet « garde-fou trous LUSR » du HealthScheduler :
// scan read-only par joueur (port.DataHealthShared.ScanLUSRGaps), qui alimente la jauge et l'alerte.
// Aucune réparation automatique : le recalcul d'un joueur est l'action manuelle du
// panneau de monitoring. Extrait de data_health_check.go pour tenir la limite de
// taille de fichier (CLAUDE.md n°5).

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// auditTitleLUSRGaps scanne les trous LUSR de tous les joueurs d'un titre
// (ScanLUSRGaps : read-only, croise candidats/watermark sur la shared DB avec les
// lignes LUSR notées de chaque player DB). AGRÈGE les compteurs dans res. La chaîne
// LUSR étant title-aware, on injecte le slug dans le ctx. Best-effort, sémantique
// « unmeasured ≠ sain » (ne casse jamais le cycle) :
//   - joueur sans xuid.txt → répertoire non résolu, PAS un joueur mesurable :
//     skip silencieux (Debug), ni ProbeErrors ni Unmeasured ;
//   - player DB présente mais inouvrable (typiquement tenue RW par un sync) OU
//     os.ReadDir du répertoire joueurs KO → WARN + res.LUSRPlayersUnmeasured : le
//     scan devient PARTIEL (la jauge ne sera pas republiée, cf. runCycle) ;
//   - scan SQL en échec → WARN + res.LUSRPlayersUnmeasured ET res.ProbeErrors (une
//     sonde SQL ratée = cron non « sain », en plus du scan partiel).
func (s *HealthScheduler) auditTitleLUSRGaps(ctx context.Context, pr *titlePkg.PathResolver, slug string, sharedDB port.DataHealthShared, res *DataHealthCheckResult) {
	titleCtx := ctxkeys.WithTitleSlug(ctx, slug)
	entries, err := os.ReadDir(pr.PlayersRootDir(slug))
	if err != nil {
		// Répertoire joueurs illisible → AUCUN joueur du titre n'a pu être scanné. Un
		// seul +1 (et non 1/joueur, nombre inconnu ici) suffit à marquer le scan
		// partiel : il gèle la republication de la jauge (runCycle), l'invariant visé.
		slog.WarnContext(ctx, "data_health: répertoire joueurs illisible — scan LUSR du titre non mesuré",
			"titleSlug", slug, "err", err)
		res.LUSRPlayersUnmeasured++
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		xuid := readXUIDFromDir(pr.PlayerDir(slug, gamertag))
		if xuid == "" {
			continue // pas de xuid.txt → joueur non résolu, skip silencieux
		}
		playerPath := pr.PlayerDBPath(slug, gamertag)
		if _, err := os.Stat(playerPath); err != nil {
			continue
		}
		scanCtx, cancel := context.WithTimeout(titleCtx, 60*time.Second)
		rep, serr := sharedDB.ScanLUSRGaps(scanCtx, playerPath, xuid)
		cancel()
		if errors.Is(serr, port.ErrDataHealthUnavailable) {
			// Player DB présente mais inouvrable (typiquement tenue RW par un sync en
			// cours) → joueur NON mesuré ce cycle. WARN + compteur : le scan devient
			// partiel (jauge gelée par runCycle) plutôt qu'un sous-comptage muet. Pas
			// de ProbeErrors : une DB tenue RW est transitoire, elle ne fait pas
			// échouer le cron.
			slog.WarnContext(ctx, "data_health: player DB inouvrable pour scan LUSR (non mesuré)",
				"titleSlug", slug, "gamertag", gamertag, "err", serr)
			res.LUSRPlayersUnmeasured++
			continue
		}
		if serr != nil {
			// Sonde SQL ratée : cron non « sain » (ProbeErrors) ET joueur non mesuré
			// (Unmeasured → jauge gelée). Les deux compteurs sont incrémentés.
			slog.WarnContext(ctx, "data_health: scan trous LUSR échoué",
				"titleSlug", slug, "gamertag", gamertag, "err", serr)
			res.ProbeErrors++
			res.LUSRPlayersUnmeasured++
			continue
		}
		res.LUSRInteriorGaps += rep.InteriorGaps
		res.LUSRPendingRecent += rep.PendingRecent
		res.LUSRPlayersScanned++
	}
}

// readXUIDFromDir lit le xuid.txt d'un répertoire joueur (1 ligne). Vide si absent
// ou illisible — le caller skippe alors le joueur.
func readXUIDFromDir(playerDir string) string {
	data, err := os.ReadFile(filepath.Clean(filepath.Join(playerDir, "xuid.txt")))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
