package replay

// film_scan_mouvement.go — LA MARCHE DU FRAME-PROCESSEUR : les ETATS DE MOUVEMENT (schema 65, lot
// 5.3.6) et, depuis le lot M4b (2026-09-24), le TIR CONTINU de la vue de controle.
//
// SORTI DE `film_scan.go` PAR DEPLACEMENT PUR, dans le commit qui l ajoute : le fichier
// atteignait 504 lignes, et le seuil de CLAUDE.md (regle 5) est de 500.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// balayerEtatsDeMouvement lit les ETATS DE MOUVEMENT du Spartan a l'instant — accroupi (i29),
// glissade (i62), action de mobilite (i54) — et le TIR CONTINU (vue C), dans la MEME marche.
//
// IL NE PARTAGE PAS LA MARCHE DES AUTRES CANAUX DE CAPACITE, et c'est mesure : celle-la est un
// CHERCHEUR D'ANCRES, qui ne retient que les records ressemblant a un en-tete de bipede — la
// population pauvre `{i0,i1,i21,i25}`. Sur `bfecd02b` elle annonce `i29` ZERO fois sur
// 162 444 records. La marche du FRAME-PROCESSEUR (`grammar.ScanMarcheDesTrames`) en rend
// 97 447 records `ti=35` dont 3 desynchronises, et 7 941 lectures d'etat. Detail et chiffres :
// l'en-tete de `grammar/movement_states.go` et la note 5.3 (section 2septdecies). Le tir continu
// y est lu parce que la vue C est le dernier rang de CHAQUE trame que cette marche deroule deja.
//
// ABSENCE NON FATALE : le rejeu sort sans intervalles d'etat ni rafales, jamais avec des
// intervalles devines.
func (s *filmScan) balayerEtatsDeMouvement() {
	m, err := grammar.ScanMarcheDesTrames(s.fc)
	st, tc := m.MovementStateStats, m.ContinuousFireStats
	if err != nil {
		slog.Warn("etats de mouvement et tir continu illisibles — rejeu sans intervalles d etat ni rafales",
			"err", err, "match_id", s.matchID)
		m, st, tc = grammar.MarcheDesTrames{}, types.MovementStateStats{}, types.ContinuousFireStats{}
	} else {
		slog.Info("etats de mouvement lus",
			"records", st.Records, "desyncs", st.Desyncs, "lectures", st.Read,
			"paquets", st.Packets, "paquetsEvenements", st.EventPackets,
			"paquetsEvenementsLocalises", st.EventPacketsLocated,
			"paquetsEvenementsNonLocalises", st.EventPacketsUnlocated,
			"slotNonLie", st.SlotUnbound, "doublons", st.Duplicates, "liaisonsOubliees", st.LiaisonsOubliees, "neufsContreUnVivant", st.NeufsContreUnVivant,
			"largeursCarte", st.MapWidths, "absent", st.Absent)
		slog.Info("tir continu lu (vue de controle)",
			"paquets", tc.Packets, "vueCFermee", tc.Closed, "trous", tc.Holes, "suitesDeTrous", tc.HoleRuns,
			"rafales", tc.Bursts, "rafalesTouchees", tc.BurstsWithHole, "tirTenuTuMs", tc.HeldHoleMS,
			"match_id", s.matchID)
	}
	s.in.MovementStates, s.in.MovementStateStats = m.MovementStates, st
	s.in.ContinuousFire, s.in.ContinuousFireStats = m.ContinuousFire, tc
	s.opt.observe("movementStates", s.in.MovementStates)
	s.opt.observe("movementStates.stats", st)
	s.opt.observe("continuousFire", s.in.ContinuousFire)
	s.opt.observe("continuousFire.stats", tc)
}
