package service

// replay_spawn_candidats.go — LES REAPPARITIONS DES AUTRES MATCHS DE LA CARTE, d'ou la prevision
// deduit ses points candidats.
//
// Elles se lisent dans les sidecars de raster (un par match, schema courant), qui portent chaque
// reapparition apres mort avec sa position : c'est la meme matiere que la mesure du modele
// (`levelup spawn-model`, qui la lit dans les artefacts). La lecture est MAIGRE : seuls l'equipe du
// joueur et, par spawn, la position, le drapeau de reapparition et la mort sont decodes ; les
// instantanes, les cellules et les routes (l'essentiel du fichier) sont ignores.
//
// LE MATCH PREVU N'EST JAMAIS LU : une prevision ne voit pas la reponse (cf. analysis/tactical,
// PrevoirLesReapparitions, qui l'ecarte aussi de son cote).
//
// Un sidecar absent (match pas encore cuit), perime (schema anterieur) ou illisible n'arrete rien :
// le match ne fournit pas de candidat, et chaque cas se compte et se journalise.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// sidecarReapparitions est la part du sidecar que la prevision lit.
type sidecarReapparitions struct {
	SchemaVersion    int     `json:"schema_version"`
	PasM             float64 `json:"pas_m"`
	PasEchantillonMs int     `json:"pas_echantillon_ms"`
	Joueurs          []struct {
		Equipe int `json:"equipe"`
		Spawns []struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
			// Apres est le drapeau `reapparition` du sidecar ; sa lecture est celle de reapparitionApresMort.
			Apres bool `json:"reapparition"`
			Mort  *struct {
				Frame int `json:"frame"`
			} `json:"mort"`
		} `json:"spawns"`
	} `json:"joueurs"`
}

// bilanCandidats compte ce que la lecture des sidecars de la carte a vu et ecarte.
type bilanCandidats struct {
	Lus, Absents, Perimes, Illisibles int
}

// reapparitionsDesAutresMatchs rend les reapparitions de la famille sur la carte, hors le match
// `exclu`, et le bilan de la lecture. Un match d'une autre famille (ou sans famille) n'en fournit
// aucune : les poids et les frequences sont ceux de la famille.
func (s *replayService) reapparitionsDesAutresMatchs(ctx context.Context, matchs []port.SpawnMatch,
	familles *mappings.SpawnFamilySet, famille, exclu string) ([]tactical.EvenementReapparition, bilanCandidats) {
	var (
		out   []tactical.EvenementReapparition
		bilan bilanCandidats
	)
	chemins := title.NewPathResolver(s.repoRoot)
	for _, m := range matchs {
		if m.MatchID == exclu || familles.FamilyOf(m.Variant, m.Category) != famille {
			continue
		}
		sc, err := lireSidecarReapparitions(chemins.TacticalRasterPath(s.titleSlug, m.MatchID))
		switch {
		case errors.Is(err, os.ErrNotExist):
			bilan.Absents++
			continue
		case err != nil:
			slog.WarnContext(ctx, "prevision de reapparition : sidecar illisible, match ecarte",
				"err", err, "match_id", m.MatchID, "titleSlug", s.titleSlug)
			bilan.Illisibles++
			continue
		case !domain.SidecarRasterCourant(&domain.TacticalRasterSidecar{
			SchemaVersion: sc.SchemaVersion, PasM: sc.PasM, PasEchantillonMs: sc.PasEchantillonMs}):
			bilan.Perimes++
			continue
		}
		bilan.Lus++
		out = append(out, evenementsDuSidecar(sc, m)...)
	}
	return out, bilan
}

// lireSidecarReapparitions lit la part du sidecar que la prevision decode. Un fichier absent se
// reconnait par errors.Is(err, os.ErrNotExist).
func lireSidecarReapparitions(path string) (*sidecarReapparitions, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture du sidecar %s: %w", path, err)
	}
	var sc sidecarReapparitions
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, fmt.Errorf("deserialisation du sidecar %s: %w", path, err)
	}
	return &sc, nil
}

// evenementsDuSidecar projette les reapparitions apres mort d'un sidecar vers les evenements du
// modele. Comme la mesure : la mort est connue, l'equipe du joueur l'est aussi, et les positions
// sont finies.
func evenementsDuSidecar(sc *sidecarReapparitions, m port.SpawnMatch) []tactical.EvenementReapparition {
	var out []tactical.EvenementReapparition
	for _, j := range sc.Joueurs {
		if j.Equipe < 0 {
			continue
		}
		for _, sp := range j.Spawns {
			spawn := domain.TacticalRasterSpawn{Reapparition: sp.Apres}
			if sp.Mort != nil {
				spawn.Mort = &domain.TacticalRasterMort{Frame: sp.Mort.Frame}
			}
			if !reapparitionApresMort(spawn) {
				continue
			}
			reel := tactical.Point3{X: sp.X, Y: sp.Y, Z: sp.Z}
			if !reel.Fini() {
				continue
			}
			out = append(out, tactical.EvenementReapparition{MatchID: m.MatchID, CarteID: m.MapID, Reel: reel})
		}
	}
	return out
}
