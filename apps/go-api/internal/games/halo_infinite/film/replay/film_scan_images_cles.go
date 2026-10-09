package replay

// film_scan_images_cles.go — LA MARCHE UNIQUE DES IMAGES-CLES POUR LES BIPEDES (lot D1.1 de 2.7.d1,
// `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`).
//
// Armes portees, inventaire et marque de portage sortent d UNE distribution de la phase des
// images-cles (`grammar.ScanEtatsDesImagesCles`), lus par la grammaire sur les records admis. Elle
// est jouee une fois par cuisson, au rang des armes portees ([filmScan.balayerPositions]) ;
// l inventaire ([filmScan.balayerInventaire]) et la marque ([filmScan.balayerCalquesGardes]) la
// reprennent sans remarcher le film.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// lireLesImagesCles joue la marche unique des images-cles et journalise ce que la regle
// d admission a juge.
func (s *filmScan) lireLesImagesCles() {
	s.etats, s.errEtats = grammar.ScanEtatsDesImagesCles(s.fc, loadoutFamilies(), 0)
	a := s.etats.Admission
	slog.InfoContext(s.ctx, "image-cle : etat complet des bipedes lu par la grammaire",
		"match_id", s.matchID, "bipedes", a.Bipedes, "admis", a.Admis, "sansCorps", a.SansCorps,
		"refusNiFermeNiI22", a.RefusNiFermeNiI22, "refusT1", a.RefusT1, "refusT2", a.RefusT2,
		"debordements", a.Debordements, "capaciteHorsDomaine", a.CapaciteHorsDomaine)
}
