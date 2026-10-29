package killcollector

// positions_entrees.go — CE QUE LA PASSE DES POSITIONS DONNE AUX DEUX LECTEURS DU LIEU : le placement
// des morts (`kill_positions`) et le contexte de mort. Les deux entrees se composent ici, a partir du
// meme materiau, pour que le cablage (l'origine d'horloge, le bord, le compteur de replis) se teste
// sans film et ne puisse pas diverger d'un lecteur a l'autre.

import (
	"levelup/go-api/internal/persist"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

// entreeDesPositions compose l'entree de [replay.BuildKillPositions] : les positions et le registre du
// materiau, les couples du journal, l'origine d'horloge, la lecture du bord et le compteur de replis
// de la passe.
func entreeDesPositions(mat materiauDIsolement, journal []persist.KillEventInsert, ids MatchIdentities,
) replay.EntreeDesPositions {
	morts, _ := journalDesMorts(journal)
	return replay.EntreeDesPositions{
		Positions: mat.positions, Registre: mat.registre, Kills: killRefsFromDeaths(journal), Morts: morts,
		OrigineUS: mat.origineUS, Fallbacks: ids.replis, Bord: mat.bord,
	}
}

// entreeContexteMorts compose l'entree de [replay.ContextesDesMorts]. LE LIEU DE LA VICTIME EST CELUI DE
// `kill_positions` (D-7, D-16) : meme instant (l'origine), meme lecture (le bord, partage avec le
// placement des morts), meme compteur de replis (lot J8.7).
func entreeContexteMorts(mat materiauDIsolement, journal []replay.MortDuJournal, equipes map[uint64]int,
	ids MatchIdentities,
) replay.EntreeContexteMorts {
	return replay.EntreeContexteMorts{
		Positions: mat.positions, Registre: mat.registre, Journal: journal, Equipes: equipes,
		Fallbacks: ids.replis, OrigineDuFilmUS: mat.origineUS, Bord: mat.bord,
	}
}
