package killcollector

// positions_bord.go — LA LECTURE DU BORD DE LA PASSE DES POSITIONS (E8.1, decision D-16), sortie de
// positions.go pour tenir le seuil de 500 lignes.

import (
	"context"
	"log/slog"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

// lireLeBordDuCollecteur rend la lecture du bord de la passe : l occupation du decodage killsource, le
// nuage et les records de creation des vehicules ([replay.LireLesVehicules]). Sans decodage, rien n est lu
// (ni tenue ni siege). Une lecture illisible se journalise ; ce qui a ete lu sert (un nuage illisible
// laisse les occupants sans lieu, des creations illisibles laissent les pieces montees et postes fixes
// sans lieu, l occupation reste lue dans les deux cas).
func lireLeBordDuCollecteur(ctx context.Context, fc *decfilm.FilmContext, entry decfilm.MapQuantEntry,
	res *decfilm.Result, matchID string,
) replay.LectureDuBord {
	if res == nil {
		return replay.LectureDuBord{}
	}
	rng := entry.Range()
	vehicules, err := replay.LireLesVehicules(fc, &rng)
	if err != nil {
		slog.WarnContext(ctx, "killsource: positions — vehicules illisibles, les occupants restent en partie sans lieu",
			"match_id", matchID, "err", err)
	}
	// Partagee : les deux lecteurs de la passe (placement des morts, contexte de mort) lisent ce bord
	// contre les memes positions et ne l'indexent qu'une fois.
	return replay.LectureDuBord{Lue: true, Occupation: res.Occupation, Vehicules: vehicules.Positions,
		Creations: vehicules.Creations}.Partagee()
}
