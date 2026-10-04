package grammar

// slot_band_releve.go — LE RELEVE DES SLOTS D UN ARCHETYPE AUX IMAGES-CLES, un canal de la phase des
// images-cles (lot 2.2 du plan de l etape 2 de la representation intermediaire).
//
// Les bandes de slots des objets du monde ([worldObjectSlotBand], [observedSlotBand]) et le
// recensement ([ScanWorldObjectKeyframes]) partent de la MEME matiere : les slots qu un archetype
// occupe aux images-cles, et ceux qu un autre archetype y occupe. Le releve la lit dans la structure
// de chaque paquet ([lecture.Paquet]) — l identite des records, sans parcourir leur corps.

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"

// releveDesSlots releve, dans la phase des images-cles, les slots de l archetype `ti` (`vus`) et ceux
// d un autre archetype (`autres`). Un slot peut etre dans les deux : il a porte l archetype a une
// image-cle et un autre a une autre.
type releveDesSlots struct {
	ti          int
	vus, autres map[uint32]bool
}

// nouveauReleveDesSlots rend un releve vide des slots de l archetype `ti`.
func nouveauReleveDesSlots(ti int) *releveDesSlots {
	return &releveDesSlots{ti: ti, vus: map[uint32]bool{}, autres: map[uint32]bool{}}
}

// releverLesSlots joue la phase des images-cles du film pour le seul releve de l archetype `ti`.
func releverLesSlots(fc *FilmContext, ti int) *releveDesSlots {
	r := nouveauReleveDesSlots(ti)
	distribuerLesImagesClesSeules(fc, []Canal{r})
	return r
}

func (*releveDesSlots) Interets() []Interet { return nil }

func (r *releveDesSlots) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	r.releverLesRecords(p.Records)
}

func (*releveDesSlots) Clore(BilanDeMarche) {}

// releverLesRecords range les records d UNE image-cle.
func (r *releveDesSlots) releverLesRecords(recs []lecture.Record) {
	for _, rec := range recs {
		if int(rec.TI) == r.ti {
			r.vus[rec.Vie.Slot] = true
		} else {
			r.autres[rec.Vie.Slot] = true
		}
	}
}
