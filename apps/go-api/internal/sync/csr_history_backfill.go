// Package sync — csr_history_backfill.go : backfill des CSR de saisons PASSÉES
// dans player_csr_snapshots, via l'endpoint par-playlist (Grunt GetPlaylistCsr —
// fonctionne pour n'importe quelle saison).
//
// Alimente le menu déroulant saison de la page Carrière : une saison n'apparaît
// que si le joueur y a des snapshots, donc on ne persiste QUE les entrées avec un
// tier réel (pas les "Non classé" des saisons jamais jouées qui pollueraient le
// menu). Réutilisable par le CLI cmd/backfill-csr-history.
package sync

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// BackfillCSRHistory récupère, pour chaque saison demandée × chaque playlist
// classée connue, le CSR du joueur (xuid) via GetPlaylistCsr, et persiste les
// snapshots à tier réel dans player_csr_snapshots (append-only). Best-effort par
// (saison, playlist) : une erreur est loggée puis ignorée. Retourne le nombre
// total de snapshots écrits.
//
// playerDB doit être ouverte en RW. client peut utiliser un token de service
// (l'endpoint skill par-playlist est public — le xuid est un paramètre).
func BackfillCSRHistory(ctx context.Context, client HaloClient, playerDB *sql.DB, xuid string, seasons []string) (int, error) {
	if strings.TrimSpace(xuid) == "" {
		return 0, nil
	}
	all := rankedplaylists.All()
	total := 0
	for _, season := range seasons {
		season = strings.TrimSpace(season)
		if season == "" {
			continue
		}
		csrs := fetchSeasonRankedCSRs(ctx, client, xuid, season, all)
		if len(csrs) == 0 {
			slog.InfoContext(ctx, "backfill CSR: saison sans classement, ignorée", "season", season, "xuid", xuid)
			continue
		}
		n, err := saveCSRSnapshots(ctx, playerDB, csrs, season)
		if err != nil {
			slog.WarnContext(ctx, "backfill CSR: persistance saison échouée", "season", season, "err", err)
			continue
		}
		total += n
		slog.InfoContext(ctx, "backfill CSR: saison persistée", "season", season, "snapshots", n)
	}
	return total, nil
}

// fetchSeasonRankedCSRs interroge GetPlaylistCsr pour toutes les playlists
// classées d'une saison et ne retient que les entrées à tier réel.
func fetchSeasonRankedCSRs(
	ctx context.Context,
	client HaloClient,
	xuid, season string,
	playlists []rankedplaylists.Playlist,
) []PlayerPlaylistCSR {
	var csrs []PlayerPlaylistCSR
	for _, pl := range playlists {
		res, err := client.GetPlaylistCsr(ctx, pl.AssetID, xuid, season)
		if err != nil {
			slog.WarnContext(ctx, "backfill CSR: GetPlaylistCsr échoué",
				"season", season, "playlist", pl.AssetID, "err", err)
			continue
		}
		if res == nil || !hasRealCSR(*res) {
			continue
		}
		res.PlaylistName = pl.NameEN
		res.Queue = pl.Queue
		res.Input = pl.Input
		csrs = append(csrs, *res)
	}
	return csrs
}

// hasRealCSR retourne true si l'entrée porte un tier réel (≠ placement/non classé).
// Évite de persister des snapshots vides pour des saisons jamais jouées.
func hasRealCSR(c PlayerPlaylistCSR) bool {
	return strings.TrimSpace(c.Current.Tier) != "" ||
		strings.TrimSpace(c.Season.Tier) != "" ||
		strings.TrimSpace(c.AllTime.Tier) != ""
}
