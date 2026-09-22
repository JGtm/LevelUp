package replay

// film_scan_mouvement.go — LE BALAYAGE DES ETATS DE MOUVEMENT (schema 65, lot 5.3.6).
//
// SORTI DE `film_scan.go` PAR DEPLACEMENT PUR, dans le commit qui l ajoute : le fichier
// atteignait 504 lignes, et le seuil de CLAUDE.md (regle 5) est de 500. Aucune ligne de code
// n a change — la methode est la meme, sur le meme recepteur, appelee au meme endroit de
// `scanFilmInputs`.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// balayerEtatsDeMouvement lit les ETATS DE MOUVEMENT du Spartan a l'instant — accroupi (i29),
// glissade (i62), action de mobilite (i54).
//
// IL NE PARTAGE PAS LA MARCHE DES AUTRES CANAUX DE CAPACITE, et c'est mesure : celle-la est un
// CHERCHEUR D'ANCRES, qui ne retient que les records ressemblant a un en-tete de bipede — la
// population pauvre `{i0,i1,i21,i25}`. Sur `bfecd02b` elle annonce `i29` ZERO fois sur
// 162 444 records. La marche du FRAME-PROCESSEUR (`grammar.ScanMovementStates`) en rend
// 97 447 records `ti=35` dont 3 desynchronises, et 7 941 lectures d'etat. Detail et chiffres :
// l'en-tete de `grammar/movement_states.go` et la note 5.3 (section 2septdecies).
//
// ABSENCE NON FATALE : le rejeu sort sans intervalles d'etat, jamais avec des intervalles
// devines.
func (s *filmScan) balayerEtatsDeMouvement() {
	reads, st, err := grammar.ScanMovementStates(s.fc)
	if err != nil {
		slog.Warn("etats de mouvement illisibles — rejeu sans intervalles d etat",
			"err", err, "match_id", s.matchID)
		reads, st = nil, types.MovementStateStats{}
	} else {
		slog.Info("etats de mouvement lus",
			"records", st.Records, "desyncs", st.Desyncs, "lectures", st.Read,
			"paquets", st.Packets, "paquetsEvenements", st.EventPackets,
			"paquetsEvenementsLocalises", st.EventPacketsLocated,
			"paquetsEvenementsNonLocalises", st.EventPacketsUnlocated,
			"slotNonLie", st.SlotUnbound, "doublons", st.Duplicates,
			"largeursCarte", st.MapWidths, "absent", st.Absent)
	}
	s.in.MovementStates, s.in.MovementStateStats = reads, st
	s.opt.observe("movementStates", s.in.MovementStates)
	s.opt.observe("movementStates.stats", st)
}
