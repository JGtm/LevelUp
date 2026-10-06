package replay

// film_scan_mouvement.go — LA MARCHE DU FRAME-PROCESSEUR : les ETATS DE MOUVEMENT (schema 65, lot
// 5.3.6) et, depuis le lot M4b (2026-09-24), le TIR CONTINU de la vue de controle.
//
// SORTI DE `film_scan.go` PAR DEPLACEMENT PUR, dans le commit qui l ajoute : le fichier
// atteignait 504 lignes, et le seuil de CLAUDE.md (regle 5) est de 500.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// balayerEtatsDeMouvement lit les ETATS DE MOUVEMENT du Spartan a l'instant — accroupi (i29),
// glissade (i62), action de mobilite (i54) — et le TIR CONTINU (vue C), dans la MEME marche.
//
// C EST LA MARCHE DU FRAME-PROCESSEUR (`grammar.ScanMarcheDesTramesAvec`), et non un chercheur
// d ancres : un chercheur ne retient que les records ressemblant a un en-tete de bipede, et c est
// la population pauvre `{i0,i1,i21,i25}` (detail : l en-tete de `grammar/movement_states.go`). Le
// tir continu y est lu parce que la vue C est le dernier rang de CHAQUE trame que cette marche
// deroule deja. LES MORTS D OBJET ET L OCCUPATION aussi (canal des morts), quand le calque des
// vehicules a ete balaye : il les prend ici ([mortsDeVehicule]). Un film sans vehicule ne paie pas
// leur lecture. Elle recueille enfin les lectures bipedes que les huit lecteurs de composants du
// portage et des capacites rejouent ensuite, l ancrage d en-tete passant derriere elle : c est
// pourquoi elle les precede (`build_from_film.go`).
//
// LA MARCHE TOURNE AUX LARGEURS MPP DU CONTEXTE, et non a celles que les socles et les vehicules
// calibrent sur les poses des formats sans largeur relue : une largeur mesuree, et non lue dans le
// jeu, n entre pas dans la lecture de toutes les entites — les corrections de la grammaire sont
// generales et lues dans le jeu.
//
// ABSENCE NON FATALE : le rejeu sort sans intervalles d'etat ni rafales, jamais avec des
// intervalles devines, et les fins de vie de vehicule a la seule borne de recensement.
func (s *filmScan) balayerEtatsDeMouvement() {
	m, err := grammar.ScanMarcheDesTramesAvec(s.fc, grammar.LecturesDeLaMarche{Morts: s.in.Vehicles.Scanned})
	st, tc := m.MovementStateStats, m.ContinuousFireStats
	if err != nil {
		slog.WarnContext(s.ctx, "etats de mouvement et tir continu illisibles — rejeu sans intervalles d etat ni rafales",
			"err", err, "match_id", s.matchID)
		m, st, tc = grammar.MarcheDesTrames{}, types.MovementStateStats{}, types.ContinuousFireStats{}
	} else {
		slog.InfoContext(s.ctx, "etats de mouvement lus",
			"records", st.Records, "desyncs", st.Desyncs, "lectures", st.Read,
			"paquets", st.Packets, "paquetsEvenements", st.EventPackets,
			"paquetsEvenementsLocalises", st.EventPacketsLocated,
			"paquetsEvenementsNonLocalises", st.EventPacketsUnlocated,
			"slotNonLie", st.SlotUnbound, "doublons", st.Duplicates, "liaisonsOubliees", st.LiaisonsOubliees, "neufsContreUnVivant", st.NeufsContreUnVivant,
			"largeursCarte", st.MapWidths, "absent", st.Absent)
		slog.InfoContext(s.ctx, "tir continu lu (vue de controle)",
			"paquets", tc.Packets, "vueCFermee", tc.Closed, "trous", tc.Holes, "suitesDeTrous", tc.HoleRuns,
			"rafales", tc.Bursts, "rafalesTouchees", tc.BurstsWithHole, "tirTenuTuMs", tc.HeldHoleMS,
			"match_id", s.matchID)
	}
	s.opt.Fallbacks.DeclencheN(fallback.NomLiaisonParAnticipation, m.LiaisonsParRepliDAnticipation)
	s.opt.Fallbacks.DeclencheN(fallback.NomDebutDeListeFermeAuBit, m.DebutsDeListeParRepliFermeAuBit)
	s.in.MovementStates, s.in.MovementStateStats = m.MovementStates, st
	s.in.ContinuousFire, s.in.ContinuousFireStats = m.ContinuousFire, tc
	if s.in.Vehicles.Scanned {
		s.in.Vehicles.Deaths, s.in.Vehicles.Occupancy, s.in.Vehicles.DeathStats = mortsDeVehicule(s.ctx, s.matchID, m)
	}
	s.opt.observe(s.ctx, "movementStates", s.in.MovementStates)
	s.opt.observe(s.ctx, "movementStates.stats", st)
	s.opt.observe(s.ctx, "continuousFire", s.in.ContinuousFire)
	s.opt.observe(s.ctx, "continuousFire.stats", tc)
	s.opt.observe(s.ctx, "vehicleDeaths", mortsEtOccupation{Deaths: s.in.Vehicles.Deaths, Occupancy: s.in.Vehicles.Occupancy})
	s.opt.observe(s.ctx, "vehicleDeaths.stats", s.in.Vehicles.DeathStats)
}

// mortsEtOccupation est ce que l etape `vehicleDeaths` observe : les morts de vehicule et les
// lectures d occupation que le calque des vehicules a pris de la marche des trames.
type mortsEtOccupation struct {
	Deaths    []types.ObjectDeath
	Occupancy []types.VehicleOccupancy
}
