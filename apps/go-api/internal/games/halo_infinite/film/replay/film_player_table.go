package replay

// film_player_table.go — LA TABLE DES JOUEURS DU FILM, JOURNALISEE ET COMPTEE (lot 1.6.0).
//
// LA LECTURE EST DESCENDUE EN `grammar` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM,
// DU-3 = S1) : `grammar.ScanFilmPlayerTable` lit `chunk_00` et rend la table, ou un refus NOMME
// avec l'erreur typee qui l'a motive — son en-tete porte la mesure du 2026-09-14 et ce qui se
// refuse. Restent ici les deux gestes que cette couche d'orchestration fait d'un refus et que la
// grammaire ne fait pas (ADR 0034 D-4) : le JOURNAL, qui porte le `match_id`, et le COMPTEUR
// expvar du build inconnu.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/observability"
)

// consignerLaTableDuFilm journalise la lecture de la table des joueurs du film
// ([grammar.ScanFilmPlayerTable], appelee une fois par cuisson dans l etage de balayage) et, sur
// un refus pour build inconnu, incremente le compteur que `grammar` nomme. Rend la table telle
// quelle.
//
// ELLE NE REND JAMAIS D ERREUR, et ce n est pas une erreur avalee : chaque cause d echec est
// TYPEE chez `grammar`, traduite en [grammar.FilmTableRefusal] NOMMEE, journalisee ici avec
// l erreur qui l a motivee, et publiee dans la couverture de l artefact. Un refus se compte ; il
// ne se tait pas.
func consignerLaTableDuFilm(ctx context.Context, t grammar.FilmPlayerTable, err error, matchID string) grammar.FilmPlayerTable {
	if t.Refusal != grammar.FilmTableRead {
		if t.Refusal == grammar.FilmTableUnknownBuild {
			// D-4 : le film est mis de cote AVEC son compteur, pour que le refus se voie en
			// production et non seulement dans le journal du jour de la cuisson.
			publierBuildInconnu(t.Build)
		}
		slog.WarnContext(ctx, "rejeu : table des joueurs du film NON LUE — le registre d'identite retombe sur "+
			"la lecture des chunks de replication", "match_id", matchID, "build", t.Build,
			"cause", string(t.Refusal), "err", err)
		return t
	}
	slog.InfoContext(ctx, "rejeu : table des joueurs du film lue", "match_id", matchID, "build", t.Build,
		"sieges", t.Occupied, "vacants", t.Vacant, "vacantIntercale", t.InterleavedVacant)
	if t.InterleavedVacant {
		slog.WarnContext(ctx, "rejeu : siege VACANT INTERCALE dans la table du film — le rang absolu et "+
			"l'index parmi les occupes divergent, le lien direct n'est PAS affirme",
			"match_id", matchID, "build", t.Build)
	}
	return t
}

// publierBuildInconnu expose sur `/debug/vars` le refus d'un film pour build inconnu (D-4 d'ADR
// 0034, compteur nomme par `grammar.UnknownBuildExpvarPairs`).
//
// C'EST ICI QUE LE COMPTEUR DU LOT 1.5 TROUVE SON CABLEUR, et c'etait ecrit : « le premier
// appelant de production de ReadPlayerTable est le registre d'identite du lot 1.6, et c'est lui
// qui publiera cette paire » (player_table_profile.go). Meme patron que
// `publierFermetureImageCle` : `grammar` NOMME ses compteurs, le consommateur les CABLE.
func publierBuildInconnu(build string) {
	for _, p := range grammar.UnknownBuildExpvarPairs(build) {
		observability.AddInt(p.Name, p.Value)
	}
}
