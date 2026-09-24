package replay

// tirs_index_fiable.go — L INDEX DE TIREUR EST-IL LA PLACE SUR CE FILM ? (lot M4b.4, 2026-09-24)
//
// LE FAIT MESURE. Le record 36 porte DEUX designations de son tireur : l index `d` (cinq bits, la
// PLACE du tireur, sonde P4) et la reference 0 (l UNITE qui tire : son bipede). Sur les sept builds
// des fixtures de rejeu ou l index est la place, l index BRUT et l index du joueur dont le bipede
// est l unite s accordent a 88,4 % au moins — l ecart est exactement le compte des tirs de
// remplacants, qui portent la place du partant (177, 57, 50, 124 : `coverage.seats.tirsParPlace`).
// Sur la build HI_1_4_1 (`a521164d`), l accord vaut 0 % : le champ lu comme tireur y est une
// constante d un tireur a l autre (`18` pour les unites 518 et 520, bits bruts au 2026-09-24).
//
// LA REGLE. Quand l accord tombe sous `seuilAccordIndexUnite`, l index n est pas la place sur ce
// film : ni la lecture des places par les tirs (`sieges_tirs.go`), ni le rattachement par la
// place ou par l index (`tirs_par_place.go`, `slotFor`) ne s en servent — seule la reference 0 pose
// un tir. C est un REPLI NOMME : le drapeau `coverage.seats.tirsIndexNonPlace` le publie, le
// journal donne l accord mesure. Critere de retrait : une lecture de l index propre aux builds
// anciennes (le champ `d` de HI_1_4_1 n est pas encore identifie).
//
// PUR : aucune I/O.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// seuilAccordIndexUnite : sous cet accord, l index de tireur n est pas la place. MESURE du
// 2026-09-24 sur les huit fixtures de build : 88,4 % a 100 % quand il l est, 0 % quand il ne l est
// pas — le seuil tombe dans le vide entre les deux familles.
const seuilAccordIndexUnite = 0.5

// accordIndexUnite est la mesure : tirs dont l unite est un bipede d index connu, et parmi eux
// ceux dont l index de tireur est celui de ce bipede.
type accordIndexUnite struct {
	accord, total int
}

// mesurerIndexDeTireur compare, tir par tir, l index de tireur a l index du joueur dont le bipede
// est l unite tireuse.
func mesurerIndexDeTireur(fire []grammar.FireEvent, owner map[uint32]int) accordIndexUnite {
	var m accordIndexUnite
	for _, e := range fire {
		if !e.Unit.Present || !e.HasShooter {
			continue
		}
		pi, ok := owner[e.Unit.Slot]
		if !ok {
			continue
		}
		m.total++
		if pi == e.FilmIndex {
			m.accord++
		}
	}
	return m
}

// estLaPlace dit si l index de tireur designe la place sur ce film. Sans aucune mesure possible,
// il est pris pour la place : c est le rattachement d avant, et rien ne le contredit.
func (m accordIndexUnite) estLaPlace() bool {
	return m.total == 0 || float64(m.accord)/float64(m.total) >= seuilAccordIndexUnite
}

// journaliser dit la mesure quand elle ecarte l index.
func (m accordIndexUnite) journaliser(matchID string) {
	if m.estLaPlace() {
		return
	}
	slog.Warn("rejeu : l index de tireur n est pas la place sur ce film — seule la reference 0 pose les tirs",
		"match_id", matchID, "accord", m.accord, "tirs", m.total, "seuil", seuilAccordIndexUnite)
}

// sansIndexDeTireur rend une COPIE des evenements ou l index de tireur est retire : seule leur
// reference 0 les designe.
func sansIndexDeTireur(fire []grammar.FireEvent) []grammar.FireEvent {
	out := make([]grammar.FireEvent, len(fire))
	copy(out, fire)
	for i := range out {
		out[i].HasShooter, out[i].FilmIndex = false, -1
	}
	return out
}
