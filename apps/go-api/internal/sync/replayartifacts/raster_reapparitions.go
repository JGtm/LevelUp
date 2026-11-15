package replayartifacts

// raster_reapparitions.go — LA PART « REAPPARITION » DU SIDECAR : la traduction des spawns de
// l'occupation vers le format du sidecar, et le compte des decisions par source.

import (
	"context"
	"math"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// arrondiCm arrondit une coordonnee au centimetre : une coordonnee issue d'un flottant 32 bits
// se serialise sinon avec une quinzaine de decimales parasites, et l'instantane de chaque
// reapparition en porte plusieurs dizaines.
func arrondiCm(v float64) float64 { return math.Round(v*100) / 100 }

// capabilityPrevisionArmee dit si le titre declare `film.spawn_forecast` (meme contrat que les portes
// jumelles : TOML illisible = incident, cle absente = configuration de titre).
func capabilityPrevisionArmee(ctx context.Context, d Deps) (armee, incident bool) {
	return porteCapability(ctx, d, games.CapFilmSpawnForecast, "prévision de réapparition", nil)
}

// spawnsDeLOccupation traduit les spawns d'un joueur vers le format du sidecar. X et Y du spawn
// restent a leur precision d'origine (les grappes les lisent telles quelles) ; tout ce que le
// schema 7 ajoute est arrondi au centimetre.
//
// La matiere de la prevision de reapparition (reapparition, source, debut de manche, mort,
// instantane) n'est portee que si `previsionArmee` : un titre qui n'a pas `film.spawn_forecast` n'en
// recoit aucune, le reste du spawn (frame, position, premiere vie) est le meme.
func spawnsDeLOccupation(spawns []tactical.SpawnPiste, previsionArmee bool) []domain.TacticalRasterSpawn {
	out := make([]domain.TacticalRasterSpawn, 0, len(spawns))
	for _, s := range spawns {
		sp := domain.TacticalRasterSpawn{
			Frame: s.Frame, X: s.X, Y: s.Y, Z: arrondiCm(s.Z), PremiereVie: s.PremiereVie,
		}
		if !previsionArmee {
			out = append(out, sp)
			continue
		}
		sp.Reapparition, sp.Source, sp.DebutDeManche = s.Reapparition, s.Source, s.DebutDeManche
		if s.Mort != nil {
			sp.Mort = &domain.TacticalRasterMort{
				Frame: s.Mort.Frame, X: arrondiCm(s.Mort.X), Y: arrondiCm(s.Mort.Y), Z: arrondiCm(s.Mort.Z),
			}
		}
		for _, v := range s.Instantane {
			sp.Instantane = append(sp.Instantane, domain.TacticalRasterVivant{
				XUID: v.XUID, Equipe: v.Equipe, X: arrondiCm(v.X), Y: arrondiCm(v.Y), Z: arrondiCm(v.Z),
			})
		}
		out = append(out, sp)
	}
	return out
}

// sourcesDesReapparitions compte, sur les joueurs d'un match, les reapparitions par source de
// decision et les debuts de manche. C'est la couverture que le sidecar publie : un match dont
// toutes les reapparitions viennent du repli se lit comme tel.
func sourcesDesReapparitions(joueurs []domain.TacticalRasterJoueur) domain.TacticalRasterSources {
	var c domain.TacticalRasterSources
	for _, j := range joueurs {
		for _, s := range j.Spawns {
			switch {
			case s.DebutDeManche:
				c.DebutsDeManche++
			case s.Reapparition && s.Source == tactical.SourceFilm:
				c.Film++
			case s.Reapparition && s.Source == tactical.SourceRepliEcart:
				c.RepliEcart++
			}
		}
	}
	return c
}

// EntreeOccupationDeLArtefact projette un document de rejeu vers l'entree pure de
// l'occupation. C'est l'UNIQUE projection : le sidecar de raster et les mesures de reapparition
// (`levelup spawn-model`) lisent le meme document de la meme facon.
//
// Le document ne porte pas encore de lecture de fin de vie ni de compte a rebours de
// reapparition : l'entree n'en a aucune, et la decision de reapparition tombe sur le repli
// nomme (cf. analysis/tactical, SourceRepliEcart).
func EntreeOccupationDeLArtefact(doc *replay.ReplayDocument) tactical.EntreeOccupation {
	intervalle := doc.FrameIntervalMS
	if intervalle <= 0 {
		intervalle = cadenceHistoriqueMS
	}
	return tactical.EntreeOccupation{
		MatchID:           doc.MatchID,
		IntervalleFrameMs: intervalle,
		Pistes:            pistesDeLArtefact(doc.Tracks),
		Embarquements:     embarquementsDeLArtefact(doc.Vehicles),
	}
}
