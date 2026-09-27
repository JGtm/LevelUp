package replay

// film_scan_pont.go — LA PREMIERE LECTURE DE L ETAGE DE BALAYAGE : LE PONT D IDENTITE (lot J4.3,
// 2026-09-26). Sortie de `balayerPositions` (film_scan.go) au seuil de longueur du depot ; meme
// etage, meme ordre d etapes observees.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// lirePontDIdentite lit, d'un seul appel a l'etage unique, les six lectures du pont d'identite, et
// pose les TELEPORTATIONS (premiere etape observee). Les autres lectures sont posees par les
// phases, a leur place d'avant.
func (s *filmScan) lirePontDIdentite() {
	// TÉLÉPORTATIONS DU TRANSLOCATEUR : lues AVANT les positions, parce qu'elles servent
	// deux fois — le calque `translocations` du document, et l'EXEMPTION du filtre de
	// vitesse (décision D2) : une arrivée de téléportation part à 193-1540 m/s, le filtre à
	// 100 m/s la rejetait à tort (R3 : 51/51 rejets mesurés, tous à ±200 ms d'un événement
	// 117 du même slot). Sur un film sans tête 117, la liste est vide et le filtre est
	// bit à bit identique à l'actuel — invariance prouvée par test.
	//
	// L'ENTRÉE DE CATALOGUE Y DESCEND parce que la CHARGE de l'événement porte les deux
	// positions du va-et-vient, quantifiées aux bornes de la carte (R6 §1, validé 18/18) :
	// sans elle le scanner rendrait des quanta invérifiables, donc rien. Elle est garantie
	// non nulle ici (refus en tête de BuildFromFilm).
	//
	// L'ETAGE UNIQUE DU PONT D'IDENTITE (lot J4.3, 2026-09-26) : teleportations, positions avec
	// leurs exemptions, creations de bipede, fil des morts, table d'index et origine d'horloge sont
	// lus ICI, d'un seul appel — le MEME que le collecteur killsource. Chaque lecture est POSEE ou
	// elle l'etait (etapes observees dans le meme ordre) ; seules les POLITIQUES de la cuisson
	// restent ici : le roster de l'index (fil des morts + roster de l'appelant), la capture des
	// directions (dans `s.scan`), l'injectivite et la fatalite des erreurs.
	s.pont = grammar.ScanPontDIdentite(s.fc, grammar.OptionsDuPont{
		Balayage: s.scan, Carte: s.opt.MapQuant,
		RosterDesMorts: func(d []types.Death) []uint64 { return rosterOf(d, s.opt.RosterXUIDs) },
	})
	s.in.Translocations = s.pont.Translocations
	if len(s.in.Translocations) > 0 {
		slog.Info("translocateur : teleportations lues", "evenements", len(s.in.Translocations))
	}
}
