package objectiveevents

import "sort"

// statborg.go — le decodage des ENREGISTREMENTS D'ENTITE des paquets FRAME, d'ou sortent
// le score de mode, le score personnel et les compteurs de combat, horodates a la ms.
//
// # Ce qu'est un enregistrement
//
// Chaque entite du match (2 equipes + 8 joueurs) est serialisee dans les paquets FRAME par
// un enregistrement portant une liste CREUSE de composants : l'enregistrement DIT quels
// composants il transporte, on n'a donc pas a parcourir les 58 de l'archetype. Un composant
// n'est reemis QUE lorsqu'il change — c'est ce qui rend la resolution temporelle bien plus
// fine que le tick de 5 s du footer.
//
// # La grammaire, MESUREE (jamais derivee du binaire seul)
//
//	[1 bit = 1][14 bits slot][2 bits = 10][1 bit forme = 0][3 bits N][N x 6 bits index]
//	puis, par index : [5 bits = 0][5 bits = 0][valeur A][valeur B][2 drapeaux][conditionnelles]
//
// Chaque constante a ete lue sur 1 078 en-tetes et 2 708 lectures de composant issus d'une
// capture Cheat Engine, pas supposee (.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md §15) :
//   - le bit qui precede le slot vaut 1 dans 1 077/1 078 ;
//   - les slots valent 6 et 8 (equipes) et 10..24 pairs (les 8 joueurs), soit
//     2 x (identifiant runtime - 0x40000000) ;
//   - les 2 bits qui suivent le slot valent 10 dans 1 077/1 078 ;
//   - les deux en-tetes de 5 bits du composant valent 0 sur **2 683 / 2 708 lectures
//     reelles (99,1 %)**, tous composants confondus, contre une poignee chez les faux
//     positifs — sans cette contrainte, 151 faux positifs par film.
//
// Piege consigne : les 2 bits qui PRECEDENT le bit de presence ne sont pas un champ de
// type. Ils sont statistiquement independants (22 co-occurrences pour 21,6 attendues sous
// independance) : c'est la queue de l'enregistrement precedent. Ne pas les contraindre.
//
// # L'horloge
//
// L'horodatage vient du `us` du paquet, recale par chunk sur le `start_ms` du manifeste.
// Les deux concordent a moins de 4 ms sur 573 s de film, et le footer (donc les
// ObjectiveEvent) est sur la meme base : tout est superposable sans recalage. Prendre pour
// origine le premier paquet OU L'ON TROUVE QUELQUE CHOSE au lieu du manifeste decale toute
// la courbe (140 s mesures sur un CTF).

const (
	// statIDBits est la largeur du champ de slot d'entite dans l'en-tete d'enregistrement.
	statIDBits = 14
	// statGenBits / statGenValue : les 2 bits qui suivent le slot, constants.
	statGenBits  = 2
	statGenValue = 0b10
	// statHdrBits est la largeur de chacun des deux en-tetes d'un composant.
	statHdrBits = 5
	// statMaxComp borne les index de composant de l'archetype (58 composants).
	statMaxComp = 58
	// statCompIndexBits est la largeur d'un index de composant dans la liste creuse.
	statCompIndexBits = 6
	// statMaxCompPerRecord : un enregistrement porte de 1 a 7 composants (compte sur
	// 3 bits ; mediane mesuree 2, jamais les 58 de l'archetype).
	statMaxCompPerRecord = 7
	// statSlotMin / statSlotMax / statTeamSlotMax delimitent les slots d'entite :
	// 6 et 8 pour les deux equipes, 10 a 24 (pairs) pour les huit joueurs.
	statSlotMin     = 6
	statSlotMax     = 24
	statTeamSlotMax = 8
	// statTailBits : marge de fin de paquet sous laquelle on n'ancre plus.
	statTailBits = 64
)

// StatValue est la paire de valeurs d'un composant. Le sens de A et de B depend du
// composant : le score de mode est en A, le score personnel en B (cf. score.go).
type StatValue struct{ A, B int64 }

// StatRecord est un enregistrement d'entite decode d'un paquet FRAME.
type StatRecord struct {
	// TimeMS est l'instant de l'emission sur l'horloge du film (meme base que le
	// TimeMS des ObjectiveEvent).
	TimeMS int
	// Slot identifie l'entite : 6 et 8 sont les deux equipes, 10..24 les huit joueurs.
	Slot int
	// Comps porte les composants effectivement transportes par cet enregistrement.
	Comps map[int]StatValue
}

// IsTeamSlot dit si un slot designe une entite d'equipe (par opposition a un joueur).
func IsTeamSlot(slot int) bool { return slot <= statTeamSlotMax }

// StatRecords decode tous les enregistrements d'entite d'un film, tries par temps puis
// par slot. L'ancrage est DIRECT : les contraintes de l'en-tete suffisent a localiser un
// enregistrement, aucune traversee de la chaine n'est necessaire.
func StatRecords(src FilmSource) []StatRecord {
	var out []StatRecord
	for _, meta := range src.Chunks() {
		raw, ok := src.ChunkData(meta.Index)
		if !ok {
			continue
		}
		data := decompressChunk(raw)
		frames := walkFrames(data)
		if len(frames) == 0 {
			continue
		}
		base := frames[0].us
		for _, f := range frames {
			tMS := meta.StartMS + int((f.us-base)/1000)
			out = append(out, scanFrameForRecords(data[f.off:f.off+f.size], tMS)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// scanFrameForRecords balaie un paquet FRAME et rend les enregistrements qu'il porte.
func scanFrameForRecords(pay []byte, tMS int) []StatRecord {
	var out []StatRecord
	lim := len(pay)*8 - statTailBits
	for b := 1; b < lim; b++ {
		slot, idx, at, ok := matchRecordHeader(pay, b)
		if !ok {
			continue
		}
		comps := decodeComponents(pay, at, idx)
		if len(comps) == 0 {
			continue
		}
		out = append(out, StatRecord{TimeMS: tMS, Slot: slot, Comps: comps})
	}
	return out
}

// matchRecordHeader teste l'en-tete d'enregistrement d'entite a la position b. Il rend le
// slot, la liste creuse d'index de composants et la position du premier composant.
func matchRecordHeader(pay []byte, b int) (slot int, idx []int, compAt int, ok bool) {
	if readBitsBE(pay, b-1, 1) != 1 {
		return 0, nil, 0, false
	}
	slot = int(readBitsBE(pay, b, statIDBits))
	if slot < statSlotMin || slot > statSlotMax || slot%2 != 0 {
		return 0, nil, 0, false
	}
	if readBitsBE(pay, b+statIDBits, statGenBits) != statGenValue {
		return 0, nil, 0, false
	}
	m := b + statIDBits + statGenBits
	if readBitsBE(pay, m, 1) != 0 {
		return 0, nil, 0, false
	}
	n := int(readBitsBE(pay, m+1, 3))
	if n < 1 || n > statMaxCompPerRecord {
		return 0, nil, 0, false
	}
	// La liste creuse doit etre strictement croissante et bornee : c'est elle qui porte
	// l'essentiel de la contrainte dure.
	idx = make([]int, n)
	prev := -1
	for i := 0; i < n; i++ {
		idx[i] = int(readBitsBE(pay, m+4+statCompIndexBits*i, statCompIndexBits))
		if idx[i] >= statMaxComp || idx[i] <= prev {
			return 0, nil, 0, false
		}
		prev = idx[i]
	}
	return slot, idx, m + 4 + statCompIndexBits*n, true
}

// decodeComponents lit les composants listes, dans l'ordre. Le PREMIER doit satisfaire la
// contrainte mesuree « les deux en-tetes de 5 bits valent 0 » : c'est elle qui separe un
// vrai enregistrement d'un ancrage fortuit. Les suivants ne sont pas re-contraints — leurs
// largeurs sont chainees, une lecture qui derape s'arrete d'elle-meme.
func decodeComponents(pay []byte, at int, idx []int) map[int]StatValue {
	if readBitsBE(pay, at, statHdrBits) != 0 || readBitsBE(pay, at+statHdrBits, statHdrBits) != 0 {
		return nil
	}
	out := make(map[int]StatValue, len(idx))
	q := at
	for _, i := range idx {
		v, w, ok := decodeStatComponent(pay, q)
		if !ok {
			break
		}
		out[i] = v
		q += w
	}
	return out
}

// decodeStatComponent lit un composant et rend sa largeur consommee. Reproduit
// FUN_140C18794 : deux en-tetes de 5 bits, deux valeurs a longueur variable, deux drapeaux
// commandant chacun une valeur conditionnelle.
func decodeStatComponent(pay []byte, p int) (StatValue, int, bool) {
	q := p + 2*statHdrBits
	a, n1, ok := readStatVarWidth(pay, q)
	if !ok {
		return StatValue{}, 0, false
	}
	b, n2, ok := readStatVarWidth(pay, q+n1)
	if !ok {
		return StatValue{}, 0, false
	}
	q += n1 + n2
	if q+2 > len(pay)*8 {
		return StatValue{}, 0, false
	}
	flags := [2]uint64{readBitsBE(pay, q, 1), readBitsBE(pay, q+1, 1)}
	q += 2
	for _, f := range flags {
		if f != 1 {
			continue
		}
		_, n, ok := readStatVarWidth(pay, q)
		if !ok {
			return StatValue{}, 0, false
		}
		q += n
	}
	return StatValue{A: a, B: b}, q - p, true
}

// readStatVarWidth reproduit le lecteur a longueur variable FUN_140C18A1C : un selecteur
// de 2 bits donne la largeur (8 << selecteur), la valeur est signee.
func readStatVarWidth(pay []byte, p int) (int64, int, bool) {
	if p < 0 || p+2 > len(pay)*8 {
		return 0, 0, false
	}
	w := 8 << uint(readBitsBE(pay, p, 2))
	if w > 32 || p+2+w > len(pay)*8 {
		return 0, 0, false
	}
	v := readBitsBE(pay, p+2, w)
	iv := int64(v)
	if w < 32 && v&(1<<uint(w-1)) != 0 {
		iv = int64(v) - (1 << uint(w))
	}
	return iv, 2 + w, true
}
