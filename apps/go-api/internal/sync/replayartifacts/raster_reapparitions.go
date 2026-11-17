package replayartifacts

// raster_reapparitions.go — LA PART « REAPPARITION » DU SIDECAR : la traduction des spawns de
// l'occupation vers le format du sidecar, et la projection du document de rejeu vers l'entree pure.

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
// La matiere de la prevision de reapparition (reapparition, debut de manche, mort,
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
		sp.Reapparition, sp.DebutDeManche = s.Reapparition, s.DebutDeManche
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

// EntreeOccupationDeLArtefact projette un document de rejeu vers l'entree pure de
// l'occupation. C'est l'UNIQUE projection : le sidecar de raster et les mesures de reapparition
// (`levelup spawn-model`) lisent le meme document de la meme facon.
//
// Les lectures de fin de vie du film y passent telles que le document les publie : la fin de
// chaque piste (`endReason`, `releaseFrame`) et les comptes a rebours de reapparition
// (`respawnCountdowns`), rattaches a un xuid par l'identite des joueurs. Un document qui n'en
// porte aucune (artefact anterieur a ces champs) n'a aucune reapparition (cf. tactical.LectureDuFilm).
func EntreeOccupationDeLArtefact(doc *replay.ReplayDocument) tactical.EntreeOccupation {
	intervalle := doc.FrameIntervalMS
	if intervalle <= 0 {
		intervalle = cadenceHistoriqueMS
	}
	debuts, sansJoueur := tactical.JoindreDebutsDeCompte(comptesDuDocument(doc.RespawnCountdowns), xuidParIndexFilm(doc.Identity))
	return tactical.EntreeOccupation{
		MatchID:           doc.MatchID,
		IntervalleFrameMs: intervalle,
		Pistes:            pistesDeLArtefact(doc.Tracks),
		Embarquements:     embarquementsDeLArtefact(doc.Vehicles),
		DebutsDeCompte:    debuts,
		ComptesSansJoueur: sansJoueur,
	}
}

// finDeLaPiste traduit la fin de vie que le document publie pour une piste. `released` porte la
// frame du lacher de commande ; sans elle, la lecture est incomplete et n'est pas reprise.
// `deleted` ne porte pas de frame : le decodeur rattache la suppression au corps de CETTE piste,
// avant la piste suivante du slot, et la frame retenue est donc la fin de la piste (meme regle
// que `tactical.NouvelIndexVies` : la fin declaree si elle est posee, sinon le dernier point).
// Toute autre valeur, ou l'absence, ne dit rien.
func finDeLaPiste(t replay.Track) *tactical.FinLue {
	switch t.EndReason {
	case replay.EndReasonReleased:
		if t.ReleaseFrame == nil {
			return nil
		}
		return &tactical.FinLue{Raison: tactical.RaisonLiberee, Frame: *t.ReleaseFrame}
	case replay.EndReasonDeleted:
		fin := t.EndFrame
		if t.EndFrame <= t.StartFrame && len(t.Points) > 0 {
			fin = t.Points[len(t.Points)-1].T
		}
		return &tactical.FinLue{Raison: tactical.RaisonSupprimee, Frame: fin}
	}
	return nil
}

// comptesDuDocument rend le debut de chaque compte a rebours de reapparition, avec l'index film
// de son joueur. `from` est la frame ou le compte vaut sa duree entiere : celle de la mort.
func comptesDuDocument(comptes []replay.RespawnCountdown) []tactical.CompteBrut {
	out := make([]tactical.CompteBrut, 0, len(comptes))
	for _, c := range comptes {
		out = append(out, tactical.CompteBrut{IndexFilm: c.FilmIndex, Debut: c.From})
	}
	return out
}

// xuidParIndexFilm rend l'index film -> xuid des joueurs humains que le film place. Un joueur
// que le film ne place nulle part (index -1) ou un bot (sans xuid) n'y entre pas ; un index
// publie deux fois garde sa premiere identite.
func xuidParIndexFilm(id *replay.IdentitySection) map[int]string {
	out := map[int]string{}
	if id == nil {
		return out
	}
	for _, p := range id.Players {
		if p.FilmIndex < 0 || p.XUID == "" {
			continue
		}
		if _, deja := out[p.FilmIndex]; !deja {
			out[p.FilmIndex] = p.XUID
		}
	}
	return out
}
