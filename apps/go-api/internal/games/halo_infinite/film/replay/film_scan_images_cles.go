package replay

// film_scan_images_cles.go — LA MARCHE UNIQUE DES IMAGES-CLES POUR LES BIPEDES (lot D1.1 de 2.7.d1,
// `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`).
//
// Armes portees, inventaire et marque de portage sortent d UNE distribution de la phase des
// images-cles (`grammar.ScanEtatsDesImagesCles`), lus par la grammaire sur les records admis. Elle
// est jouee une fois par cuisson, au rang des armes portees ([filmScan.balayerPositions]) ;
// l inventaire ([filmScan.balayerInventaire]) et la marque ([filmScan.balayerCalquesGardes]) la
// reprennent sans remarcher le film.
//
// LES RECORDS RECUPERES (D1.3, 2026-10-09). Un record non admis dont une fenetre de bits a rendu une
// valeur est marque recupere par la grammaire (`grammar.RecordRecupere`, ADR 0037 IR-6), avec ses
// methodes. Sa valeur entre dans les memes listes que les lectures de la grammaire, dans l ordre
// du film : un client n a qu une grandeur par champ. Ce que la cuisson PUBLIE de la recuperation,
// ce sont les comptes des trois replis (`coverage.fallbacks`, verses avec le rapport du contexte et
// persistes avec lui dans les faits) ; la marque par record ne voyage pas (decision E-8 du plan :
// la forme des faits ne s elargit pas), elle se JOURNALISE ici, par methode.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// lireLesImagesCles joue la marche unique des images-cles et journalise ce que la regle
// d admission a juge et ce que les fenetres ont recupere.
func (s *filmScan) lireLesImagesCles() {
	s.etats, s.errEtats = grammar.ScanEtatsDesImagesCles(s.fc, loadoutFamilies(), 0)
	a := s.etats.Admission
	r := recuperesParMethode(s.etats.Recuperes)
	slog.InfoContext(s.ctx, "image-cle : etat complet des bipedes lu par la grammaire",
		"match_id", s.matchID, "bipedes", a.Bipedes, "admis", a.Admis, "sansCorps", a.SansCorps,
		"refusNiFermeNiI22", a.RefusNiFermeNiI22, "refusT1", a.RefusT1, "refusT2", a.RefusT2,
		"debordements", a.Debordements, "capaciteHorsDomaine", a.CapaciteHorsDomaine,
		"recuperes", len(s.etats.Recuperes), "recuperesArmes", r.armes,
		"recuperesInventaire", r.inventaire, "recuperesMarque", r.marque)
}

// comptesDesRecuperes compte les records recuperes par methode : un record compte dans chaque
// fenetre qui lui a rendu une valeur.
type comptesDesRecuperes struct{ armes, inventaire, marque int }

// recuperesParMethode compte les records recuperes par les fenetres, par methode.
func recuperesParMethode(recs []grammar.RecordRecupere) comptesDesRecuperes {
	var c comptesDesRecuperes
	for _, r := range recs {
		c.armes += unSiVrai(r.Armes)
		c.inventaire += unSiVrai(r.Inventaire)
		c.marque += unSiVrai(r.Marque)
	}
	return c
}

// unSiVrai rend 1 pour vrai, 0 pour faux.
func unSiVrai(b bool) int {
	if b {
		return 1
	}
	return 0
}
