package grammar

// morts_de_la_marche.go — LES MORTS QUE LA MARCHE DES TRAMES LIT, AVEC LEUR POSITION, POUR
// `killsource`.
//
// Le canal des morts ([canalDesMorts]) recolte les records de la vue B que la marche lit, et ceux des
// listes qu elle ne localise pas, qu il recupere (signature, puis largeur libre). Sur demande, il
// garde chaque dead-state `Mort` avec sa trame, la position de son composant et la qualite de sa
// lecture : c est ce que killsource lit des morts d un film. killsource y applique sa regle (records
// entierement portes, archetype bipede) et son filtre de credibilite.

import "levelup/go-api/internal/games/halo_infinite/film/types"

// EtatDeMortLu est un dead-state `Mort` qu un record lu porte.
type EtatDeMortLu struct {
	// PositionDuChunk et Index situent la trame : la position de son chunk dans la source (celle de
	// `types.Packet.Chunk`), son rang dans le chunk ; TS est son horodatage.
	PositionDuChunk, Index int
	TS                     uint64
	// Slot, Gen et TypeIndex identifient l entite et son archetype, celui que la marche lie.
	Slot, Gen, TypeIndex uint32
	// Bit est le premier bit du dead-state dans le payload, -1 s il n est pas dans la trace.
	Bit int
	// Propre dit que le record est entierement porte (aucune rupture) ; Recupere, que sa liste n a
	// pas ete localisee par la marche et que le canal l a recuperee.
	Propre, Recupere bool
	// Dead est le dead-state lu.
	Dead types.DeadState
}

// MortsDeLaMarche est ce que la marche des trames lit des morts d un film pour `killsource`.
type MortsDeLaMarche struct {
	// Lus sont les dead-states `Mort` des records lus et recuperes, dans l ordre du flux.
	Lus []EtatDeMortLu
	// Stats sont les denominateurs du canal des morts ; LargeurLibre, les listes recuperees par le
	// repli a largeur libre (`repli_localisation_largeur_libre`).
	Stats        ObjectDeathStats
	LargeurLibre int
}

// LireLesMortsDeLaMarche distribue le seul canal des morts sur la marche des trames du film et rend
// ses dead-states lus avec leur position. La cuisson lit les morts par [ScanMarcheDesTramesAvec].
func LireLesMortsDeLaMarche(fc *FilmContext) (MortsDeLaMarche, error) {
	reg, err := fc.Registry()
	if err != nil {
		return MortsDeLaMarche{}, err
	}
	c := nouveauCanalDesMorts(reg)
	c.garder = true
	if err := Distribuer(fc, c); err != nil {
		return MortsDeLaMarche{}, err
	}
	return MortsDeLaMarche{Lus: c.mortsLues, Stats: c.st, LargeurLibre: c.largeurLibre}, nil
}
