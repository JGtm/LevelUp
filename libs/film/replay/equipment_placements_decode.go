package replay

// equipment_placements_decode.go — LE BALAYAGE DES POSES ET DES EVENEMENTS 103, journalises (hors ligne).
// DEPLACEMENT PUR depuis `equipment_placements.go` (plan des durees d effets, B.1) pour tenir le seuil de
// 500 lignes : aucune ligne de logique n a change.

import (
	"context"
	"log/slog"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// decodeFilmPlacements décode les poses du film et JOURNALISE ce qu'il en est.
//
// TROIS SORTIES, TROIS PHRASES — et la distinction est le point. Un balayage impossible (film
// illisible) n'est pas un film sans équipement, et un film dont le découpage du bloc de
// réplication n'a pas été tranché n'est ni l'un ni l'autre : il aurait des poses, mais les lire
// à une largeur devinée rendrait du bruit. Les trois se lisent au journal, et la troisième se
// relit ensuite dans l'artefact (`coverage.placements.calibrated`).
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPlacements(ctx context.Context,
	fc *grammar.FilmContext, matchID string, worldRange *profile.Vec3Range,
) ([]types.EquipmentPlacement, grammar.EquipmentPlacementStats) {
	pl, st, err := grammar.ScanEquipmentPlacements(fc, worldRange)
	if st.FormatSansProfil {
		// SITE 1 DU REPLI `repli_largeurs_mpp_calibrees_sur_le_film` : le compteur, pas le
		// journal — l avertissement est emis UNE FOIS PAR FILM par `avertirFormatSansProfil`.
		publierFormatSansProfil(st.FormatVersion)
	}
	switch {
	case err != nil:
		slog.WarnContext(ctx, "poses d'equipement illisibles — rejeu sans equipement pose",
			"err", err, "match_id", matchID)
		return nil, st
	case !st.Calibration.Widths.Valid():
		slog.WarnContext(ctx, "poses d'equipement : le decoupage du bloc de replication n'a pas ete tranche"+
			" sur ce film — AUCUNE pose publiee plutot que du bruit",
			"match_id", matchID, "ancres", st.Calibration.Anchors,
			"vies", st.Calibration.Lives, "chunksLus", st.Calibration.Chunks)
	default:
		slog.InfoContext(ctx, "poses d'equipement : records de creation ti=37",
			"decoupage", st.Calibration.Widths.String(), "accords", st.Calibration.Agree,
			"ancres", st.Anchors, "acceptes", st.Accepted,
			"confirmes", st.Confirmed, "poses", st.Placements)
	}
	return pl, st
}

// decodeFilmSpawnEvents lit les evenements 103 « une PIECE a ete engendree » et JOURNALISE ce
// que le balayage a vu.
//
// UN FILM SANS AUCUN EVENEMENT N'EST PAS UNE ERREUR, ET CE N'EST PAS NON PLUS UN SILENCE : les
// builds les plus anciens du corpus n'en rendent aucun (`a521164d`, HI_1_4_1 : 0 sur 4 956
// listes non vides) la ou les recents en rendent des dizaines. Le journal publie donc les DEUX
// denominateurs — listes traversees et evenements lus —, et la couverture de l'artefact les
// publie TOUS DEUX (`coverage.placements.spawnLists` et `.spawnEvents`).
//
// HORS LIGNE — appelee par le balayage, sous le meme verrou que le reste de la cuisson.
func decodeFilmSpawnEvents(ctx context.Context,
	fc *grammar.FilmContext, matchID string,
) ([]types.EquipmentSpawnEvent, types.EquipmentSpawnStats) {
	ev, st, err := grammar.ScanEquipmentSpawnEvents(fc)
	if err != nil {
		slog.WarnContext(ctx, "evenements de piece engendree illisibles — l'origine des poses retombe sur ses replis",
			"err", err, "match_id", matchID)
		return nil, st
	}
	slog.InfoContext(ctx, "poses d'equipement : evenements 103 (piece engendree)",
		"match_id", matchID, "chunks", st.Chunks, "paquetsDelta", st.Packets,
		"listesNonVides", st.Lists, "evenements", st.Events,
		"refSource", st.WithSource, "refEngendree", st.WithSpawned, "ref2", st.Ref2)
	return ev, st
}
