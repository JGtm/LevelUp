package replay

// filmfacts_statsdepose.go — LES DENOMINATEURS DE POSE, ET LES MAPS DANS UN ORDRE EXPLICITE.
//
// Extrait de `filmfacts_encode.go` le 2026-09-18 : celui-ci franchissait les 500 lignes, et la
// table de `film_file_size_test.go` est DATEE ET FERMEE au 2026-09-16 — un fichier neuf ne s y
// inscrit pas, il se coupe. La coupe suit une frontiere reelle : la SEQUENCE des sections reste
// dans `filmfacts_encode.go`, les denominateurs de pose et les deux codecs de map viennent ici.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// encodeStatsDePose / decodeStatsDePose : LES ONZE CHAMPS DES DENOMINATEURS DE POSE.
//
// # CE QUE L ANCIENNE VERSION PERDAIT, ET CE QUE CA COUTAIT (2026-09-18, gate S8)
//
// Elle portait SEPT champs sur onze, et parmi les absents il y avait `Scanned` — le TEMOIN qui dit
// que le balayage a eu lieu. Un artefact rejoue publiait donc `coverage.placements.scanned: false`
// la ou la cuisson publie `true` : le document affirmait « on n a pas regarde » sur un film qu on
// avait regarde. Mesure : c est l ecart de UN OCTET (`true` contre `false`) qui restait sur les
// cinq films d ARENE du gate S8, les seuls dont rien d autre ne divergeait.
//
// `ByWidths` et `ByID` sont des MAPS : elles voyagent par leur cardinal puis leurs couples, dans
// l ordre TRIE — une map Go ne s itere pas deux fois pareil, et un fichier de faits doit etre le
// meme a chaque ecriture.
func encodeStatsDePose(w *gwriter, st grammar.EquipmentPlacementStats) {
	w.bool8(st.Scanned)
	w.i(int64(st.Calibration.Widths.Lead))
	w.i(int64(st.Calibration.Widths.Index))
	w.i(int64(st.Calibration.Agree))
	w.i(int64(st.Calibration.Runner.Lead))
	w.i(int64(st.Calibration.Runner.Index))
	w.i(int64(st.Calibration.RunnerAgree))
	w.i(int64(st.Calibration.Anchors))
	w.i(int64(st.Calibration.Chunks))
	w.i(int64(st.Calibration.Lives))
	encodeCouplesTriesParLargeurs(w, st.Calibration.ByWidths)
	w.i(int64(st.Lives))
	w.i(int64(st.Slots))
	w.i(int64(st.Anchors))
	w.i(int64(st.Accepted))
	w.i(int64(st.Confirmed))
	w.i(int64(st.Placements))
	encodeCouplesTriesParID(w, st.ByID)
	w.i(int64(st.FormatVersion))
	w.bool8(st.FormatSansProfil)
}

func decodeStatsDePose(r *greader) grammar.EquipmentPlacementStats {
	var st grammar.EquipmentPlacementStats
	st.Scanned = r.bool8()
	st.Calibration.Widths.Lead = int(r.i())
	st.Calibration.Widths.Index = int(r.i())
	st.Calibration.Agree = int(r.i())
	st.Calibration.Runner.Lead = int(r.i())
	st.Calibration.Runner.Index = int(r.i())
	st.Calibration.RunnerAgree = int(r.i())
	st.Calibration.Anchors = int(r.i())
	st.Calibration.Chunks = int(r.i())
	st.Calibration.Lives = int(r.i())
	st.Calibration.ByWidths = decodeCouplesTriesParLargeurs(r)
	st.Lives, st.Slots, st.Anchors = int(r.i()), int(r.i()), int(r.i())
	st.Accepted, st.Confirmed, st.Placements = int(r.i()), int(r.i()), int(r.i())
	st.ByID = decodeCouplesTriesParID(r)
	st.FormatVersion = int(r.i())
	st.FormatSansProfil = r.bool8()
	return st
}

// encodeCouplesTriesParLargeurs / encodeCouplesTriesParID : une map, dans un ORDRE EXPLICITE.
func encodeCouplesTriesParLargeurs(w *gwriter, m map[profile.MPPWidths]int) {
	cles := make([]profile.MPPWidths, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i].Lead != cles[j].Lead {
			return cles[i].Lead < cles[j].Lead
		}
		return cles[i].Index < cles[j].Index
	})
	w.u(uint64(len(cles)))
	for _, k := range cles {
		w.i(int64(k.Lead))
		w.i(int64(k.Index))
		w.i(int64(m[k]))
	}
}

func decodeCouplesTriesParLargeurs(r *greader) map[profile.MPPWidths]int {
	n := r.compte(3)
	if n == 0 {
		return nil
	}
	m := make(map[profile.MPPWidths]int, n)
	for k := 0; k < n && r.err == nil; k++ {
		cle := profile.MPPWidths{Lead: int(r.i()), Index: int(r.i())}
		m[cle] = int(r.i())
	}
	return m
}

func encodeCouplesTriesParID(w *gwriter, m map[uint32]int) {
	cles := make([]uint32, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return cles[i] < cles[j] })
	w.u(uint64(len(cles)))
	for _, k := range cles {
		w.u(uint64(k))
		w.i(int64(m[k]))
	}
}

func decodeCouplesTriesParID(r *greader) map[uint32]int {
	n := r.compte(2)
	if n == 0 {
		return nil
	}
	m := make(map[uint32]int, n)
	for k := 0; k < n && r.err == nil; k++ {
		cle := uint32(r.u())
		m[cle] = int(r.i())
	}
	return m
}
