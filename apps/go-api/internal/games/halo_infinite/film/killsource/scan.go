package killsource

// scan.go — LE SCAN DIRECT ET SES QUATRE TESTS (RE_LOG 7ter.60, verifie 7ter.61).
//
// L IDEE VIENT DE L UTILISATEUR : *<< le fichier ne se lit pas en continu, on n a pas a
// l approcher de maniere sequentielle >>*. La sequentialite n est PAS imposee par le format,
// elle est imposee par la MARCHE — un record ECS n a ni longueur ni separateur, donc un
// composant EN AMONT de largeur inconnue fait perdre tout ce qui suit. Or le record de mort est
// la, a quelques centaines de bits.
//
// On balaie donc TOUTES les positions de bit et on ne retient que ce qui passe QUATRE tests
// simultanes. Cela contourne entierement le probleme des largeurs : pour la question << quelle
// arme a tue >>, la calibration, les keyframes et le localisateur deviennent FACULTATIFS.
//
//	T1  victime   = comp+0x04, R(1) gate a 0 puis R(5)   -> indice < nb de participants
//	T2  tueur     = comp+0x08, idem                      -> indice < nb de participants
//	T3  categorie = comp+0x0c, R(4)                      -> dans l enum (0..9)
//	T4  TAG       = comp+0x00, R(1) gate a 1 puis R(32)  -> entree du groupe `jpt!`
//
// GRAMMAIRE BALAYEE (forme BIPED, typeIndex 0x23) :
//
//	p+0        R(1)  Mort            doit valoir 1
//	p+1        R(1)  gate tag        doit valoir 1
//	p+2..p+33  R(32) tag             doit etre dans `jpt!`
//	p+34..p+41 R(8)
//	p+42       R(1)  gate victime    doit valoir 0
//	p+43..p+47 R(5)  victime         < nParticipants
//	p+48       R(1)  gate tueur      doit valoir 0
//	p+49..p+53 R(5)  tueur           < nParticipants
//	p+54..p+57 R(4)  categorie       <= 9
//
// LE SCAN N INVENTE PAS DE MORTS, ET C EST MESURE, PAS AFFIRME :
//
//	controle negatif   1 seul candidat sur 208 913 712 bits de paquets SANS event
//	concordance        346/346 AU MEME BIT avec la marche, desaccord 0
//	leurre             468 identifiants tires AU HASARD -> 0 candidat
//	regime             dans les paquets a events les tests passent 85 fois plus souvent
//
// T4 porte l essentiel du pouvoir : ~1 faux positif sur 9 000 000 pour les valeurs >= 0x10000.
// Sous 0x10000 le taux monte a 1.15 %, et TROIS tags REELS y vivent : on ne peut donc pas
// exclure ce regime du scan, seulement refuser de melanger les deux dans une mesure.

// deadStateBits : longueur du gabarit balaye, en bits.
const deadStateBits = 58

// candidate : un dead-state i11 accepte par les quatre tests. C est l unite commune aux DEUX
// voies : la marche convertit ses records dans cette forme, position en bits comprise.
type candidate struct {
	chunk, pidx int
	ms          int
	bit         int // -1 quand la position n a pas ete enregistree (record de la marche)
	tag         uint32
	victim      int
	killer      int
	cat         int
}

// scanPayload : toutes les positions de bit d un paquet ou un dead-state i11 valide se decode.
func scanPayload(pl []byte, nParticipants int) []candidate {
	var out []candidate
	nb := len(pl) * 8
	for p := 0; p+deadStateBits <= nb; p++ {
		if bitAt(pl, p) == 0 || bitAt(pl, p+1) == 0 { // Mort + gate du tag
			continue
		}
		tag := bits32(pl, p+2)
		if !isCatalogued(tag) { // T4
			continue
		}
		c, ok := readIndices(pl, p, nParticipants)
		if !ok {
			continue
		}
		c.tag = tag
		out = append(out, c)
	}
	return out
}

// readIndices : la queue du gabarit (victime, tueur, categorie) et ses trois tests. Extraite
// pour que le scan et la sonde a T4 relache partagent EXACTEMENT le meme code — une copie
// divergerait en silence, et la sonde cesserait de mesurer ce qu elle croit mesurer.
func readIndices(pl []byte, p, nParticipants int) (candidate, bool) {
	q := p + 42
	if bitAt(pl, q) != 0 { // gate victime
		return candidate{}, false
	}
	vic := bitsN(pl, q+1, 5)
	if vic >= nParticipants { // T1
		return candidate{}, false
	}
	if bitAt(pl, q+6) != 0 { // gate tueur
		return candidate{}, false
	}
	kil := bitsN(pl, q+7, 5)
	if kil >= nParticipants { // T2
		return candidate{}, false
	}
	cat := bitsN(pl, q+12, 4)
	if cat > 9 { // T3
		return candidate{}, false
	}
	return candidate{bit: p, victim: vic, killer: kil, cat: cat}, true
}

// scanFilm : le scan de tous les paquets type-0 A EVENTS.
//
// POPULATION, ET C EST LA LECON DE METHODE LA PLUS CHERE DU CHANTIER : les morts sont 93/93
// dans les paquets A EVENTS. Trois sections entieres ont mesure sur les paquets SANS event —
// une population qui ne contient PAS la cible. Avant d optimiser une metrique, prouver que la
// cible est dans la population.
func scanFilm(f *film, nParticipants int) []candidate {
	var out []candidate
	for i := range f.t0 {
		p := &f.t0[i]
		if !hasEvents(p) {
			continue
		}
		cs := scanPayload(p.payload, nParticipants)
		ms := f.ms(p)
		for k := range cs {
			cs[k].chunk, cs[k].pidx, cs[k].ms = p.chunk, p.idx, ms
		}
		out = append(out, cs...)
	}
	return dedup(out)
}

// dedup : un meme dead-state est repete dans les VUES de replication d un paquet. Deux
// candidats du MEME paquet portant le MEME quadruplet (tag, victime, tueur, categorie) sont une
// seule et meme mort : on ne garde que le premier bit. Sans cela une voie gonflerait sa propre
// population et son gate (b) cesserait d etre comparable a celui de l autre.
func dedup(cs []candidate) []candidate {
	seen := map[[6]int]bool{}
	out := make([]candidate, 0, len(cs))
	for _, c := range cs {
		k := [6]int{c.chunk, c.pidx, int(c.tag), c.victim, c.killer, c.cat}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, c)
	}
	return out
}

// multKey : le triplet dont les faux positifs se repetent.
type multKey struct {
	bit    int
	tag    uint32
	victim int
}

// multiplicity : pour chaque triplet (bit, tag, victime), le nombre de PAQUETS DISTINCTS qui
// portent un candidat. Les vues de replication d un meme paquet ne comptent qu une fois.
//
// C EST UN DISCRIMINANT STRUCTUREL, PAS UNE HEURISTIQUE : un vrai dead-state tombe a un bit
// quelconque ; un champ fixe lu comme un dead-state se repete au MEME bit sur des paquets
// differents (11 fois au bit 746, meme tag, meme joueur, sur un film). Son cout en vrais
// positifs est ZERO sur les quatre films, a tous les seuils 2..6.
func multiplicity(cs []candidate) map[multKey]int {
	pkts := map[multKey]map[[2]int]bool{}
	for _, c := range cs {
		k := multKey{c.bit, c.tag, c.victim}
		if pkts[k] == nil {
			pkts[k] = map[[2]int]bool{}
		}
		pkts[k][[2]int{c.chunk, c.pidx}] = true
	}
	out := make(map[multKey]int, len(pkts))
	for k, v := range pkts {
		out[k] = len(v)
	}
	return out
}

func multOf(m map[multKey]int, c candidate) int { return m[multKey{c.bit, c.tag, c.victim}] }

// scanRelaxedPayload : LE MEME GABARIT, AMPUTE DE SA SEULE PORTE FORTE (T4). Sert uniquement a
// la metrique de sante : c est la seule facon de voir ce que la porte du catalogue CACHE.
// Il partage [readIndices] avec le scan reel, donc il ne peut pas rendre MOINS de candidats.
func scanRelaxedPayload(pl []byte, nParticipants int) []candidate {
	var out []candidate
	nb := len(pl) * 8
	for p := 0; p+deadStateBits <= nb; p++ {
		if bitAt(pl, p) == 0 || bitAt(pl, p+1) == 0 {
			continue
		}
		c, ok := readIndices(pl, p, nParticipants)
		if !ok {
			continue
		}
		c.tag = bits32(pl, p+2)
		out = append(out, c)
	}
	return out
}
