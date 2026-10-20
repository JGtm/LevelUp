package grammar

// etats_de_mort_balayes.go — LE GABARIT DU DEAD-STATE DE BIPEDE, CHERCHE A CHAQUE BIT DES TRAMES
// (RE_LOG 7ter.60, verifie 7ter.61). C est le balayage de `killsource`, descendu de
// `facts/killsource/scan.go` au lot 2.7.c1 de la representation intermediaire, a l identique :
// la grammaire rend les positions ou le gabarit se lit, killsource garde ce qu il en fait (la porte
// du catalogue des tags, la redondance avec sa marche, la multiplicite, l appariement au kill-feed).
//
// GRAMMAIRE BALAYEE (forme BIPED, archetype 0x23) :
//
//	p+0        R(1)  Mort            doit valoir 1
//	p+1        R(1)  garde du tag    doit valoir 1
//	p+2..p+33  R(32) tag             accepte par l appelant
//	p+34..p+41 R(8)
//	p+42       R(1)  garde victime   doit valoir 0
//	p+43..p+47 R(5)  victime         < nombre de participants
//	p+48       R(1)  garde tueur     doit valoir 0
//	p+49..p+53 R(5)  tueur           < nombre de participants
//	p+54..p+57 R(4)  categorie       <= 9

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

// LargeurDuGabaritDeMort est la longueur du gabarit balaye, en bits.
const LargeurDuGabaritDeMort = 58

// categorieDeMortMax est la plus grande categorie de l enumeration du dead-state.
const categorieDeMortMax = 9

// EtatDeMortBalaye est une position de bit d une trame delta ou le gabarit se lit.
type EtatDeMortBalaye struct {
	// PositionDuChunk et Index situent la trame : la position de son chunk dans la source (celle de
	// `types.Packet.Chunk`), son rang dans le chunk ; TS est son horodatage.
	PositionDuChunk, Index int
	TS                     uint64
	// Bit est le premier bit du gabarit dans le payload.
	Bit int
	// Tag, Victime, Tueur et Categorie sont les champs lus.
	Tag                       uint32
	Victime, Tueur, Categorie int
}

// BalayerLesEtatsDeMort rend, dans l ordre de la source puis des bits, les positions des trames delta
// du film — celles qui annoncent une liste d evenements si `avecEvenements`, les autres sinon — ou le
// gabarit se lit avec des indices de victime et de tueur inferieurs a `nParticipants` et un tag que
// `tagAccepte` accepte (nil : tout tag).
func BalayerLesEtatsDeMort(f *source.Film, nParticipants int, tagAccepte func(uint32) bool,
	avecEvenements bool,
) []EtatDeMortBalaye {
	var out []EtatDeMortBalaye
	for _, p := range f.AllPackets() {
		if p.Type != int(PacketTypeDelta) || (source.BitAt(p.Payload, 1) != 0) != avecEvenements {
			continue
		}
		for _, e := range balayerUnPayload(p.Payload, nParticipants, tagAccepte) {
			e.PositionDuChunk, e.Index, e.TS = p.Chunk, p.Index, p.TS
			out = append(out, e)
		}
	}
	return out
}

// balayerUnPayload rend les positions de `pl` ou le gabarit se lit.
func balayerUnPayload(pl []byte, nParticipants int, tagAccepte func(uint32) bool) []EtatDeMortBalaye {
	var out []EtatDeMortBalaye
	nb := len(pl) * 8
	for p := 0; p+LargeurDuGabaritDeMort <= nb; p++ {
		if source.BitAt(pl, p) == 0 || source.BitAt(pl, p+1) == 0 { // Mort + garde du tag
			continue
		}
		tag := uint32(source.BitsAt(pl, p+2, 32)) //nolint:gosec // R(32)
		if tagAccepte != nil && !tagAccepte(tag) {
			continue
		}
		e, ok := lireLesIndicesDuGabarit(pl, p, nParticipants)
		if !ok {
			continue
		}
		e.Tag = tag
		out = append(out, e)
	}
	return out
}

// lireLesIndicesDuGabarit lit la queue du gabarit qui commence au bit `p` — victime, tueur,
// categorie — et ses trois tests.
func lireLesIndicesDuGabarit(pl []byte, p, nParticipants int) (EtatDeMortBalaye, bool) {
	q := p + 42
	if source.BitAt(pl, q) != 0 { // garde victime
		return EtatDeMortBalaye{}, false
	}
	vic := int(source.BitsAt(pl, q+1, 5)) //nolint:gosec // R(5)
	if vic >= nParticipants {
		return EtatDeMortBalaye{}, false
	}
	if source.BitAt(pl, q+6) != 0 { // garde tueur
		return EtatDeMortBalaye{}, false
	}
	kil := int(source.BitsAt(pl, q+7, 5)) //nolint:gosec // R(5)
	if kil >= nParticipants {
		return EtatDeMortBalaye{}, false
	}
	cat := int(source.BitsAt(pl, q+12, 4)) //nolint:gosec // R(4)
	if cat > categorieDeMortMax {
		return EtatDeMortBalaye{}, false
	}
	return EtatDeMortBalaye{Bit: p, Victime: vic, Tueur: kil, Categorie: cat}, true
}
