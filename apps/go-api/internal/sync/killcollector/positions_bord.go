package killcollector

// positions_bord.go — LA LECTURE DU BORD DE LA PASSE DES POSITIONS (E8.1, decision D-16), sortie de
// positions.go pour tenir le seuil de 500 lignes.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// lireLeBordDuCollecteur rend la lecture du bord de la passe : l occupation du decodage killsource et
// le nuage des vehicules ([replay.PositionsDesVehicules]). Sans decodage, rien n est lu (ni tenue ni
// siege). Un nuage illisible se journalise et laisse les occupants sans lieu ; l occupation reste lue.
func lireLeBordDuCollecteur(ctx context.Context, fc *decfilm.FilmContext, entry decfilm.MapQuantEntry,
	res *decfilm.Result, matchID string,
) replay.LectureDuBord {
	if res == nil {
		return replay.LectureDuBord{}
	}
	rng := entry.Range()
	vehicules, err := replay.PositionsDesVehicules(fc, &rng)
	if err != nil {
		slog.WarnContext(ctx, "killsource: positions — nuage des vehicules illisible, les occupants restent sans lieu",
			"match_id", matchID, "err", err)
		vehicules = nil
	}
	// Partagee : les deux lecteurs de la passe (placement des morts, contexte de mort) lisent ce bord
	// contre les memes positions et ne l'indexent qu'une fois.
	return replay.LectureDuBord{Lue: true, Occupation: res.Occupation, Vehicules: vehicules}.Partagee()
}
