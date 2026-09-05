package killsource

// chunks.go — L ENTREE DU PAQUET : UN FILM DEJA CHARGE, ET RIEN D AUTRE.
//
// # CE QUE CE FICHIER A CESSE DE FAIRE (lot 1, PLAN_CUISSON_PERF item 1.4, 2026-09-02)
//
// Il portait sa propre source de chunks (`ChunkSource`, `MemoryChunks`, `DirChunks`), son propre
// inflate zlib et son propre marcheur de paquets (`splitPackets`) — la TROISIEME copie des trois,
// a cote de `filmdec` et d `objectiveevents`, et les trois DIVERGEAIENT. Une cuisson d artefact
// payait donc une lecture disque et une decompression du film ENTIER rien que pour ce decodeur,
// en plus de celles des balayages.
//
// Tout cela vit desormais dans `internal/analysis/filmsource`, paquet FEUILLE : l appelant charge
// le film UNE fois et le passe a [Decode]. Ce fichier ne fait plus que TRADUIRE ce film dans le
// vocabulaire interne du decodeur (`packet`, `film`) et trier les paquets type-0 par horodatage.
//
// PIEGE HISTORIQUE, ET IL A COUTE UN KILL-FEED ENTIER : tout l outillage de RE bornait la lecture
// au chunk 41. Un film BTB en compte 63, et son chunk HIGHLIGHT est le n62 — le kill-feed y etait
// purement introuvable (RE_LOG 7ter.52). La garantie est intacte, elle a seulement change de
// domicile : `filmsource` ne borne rien, et ce fichier lit TOUS les chunks du film charge.
//
// # LE CHUNK 0 EST LE PREMIER DE LA SOURCE, PAS « LE CHUNK NUMERO 0 »
//
// `f.chunks` est indexe par POSITION dans la source, exactement comme l ancien `ChunkSource` :
// `f.chunks[0]` est le PREMIER chunk que la source donne, et c est lui que `newTimeline`
// (world.go) lit comme registre ECS. Sur un cache complet ou sur une sequence telechargee, la
// position 0 porte bien `chunk_00`, le registre ; sur une bobine partielle qui commence a
// `chunk_01`, elle porte un chunk de donnees — et c etait DEJA le cas avant. Le contrat est donc
// conserve tel quel : `film.Chunk(i)` de `filmsource` est indexe par la meme position.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// packet : un paquet de replication, tel qu il se presente dans un chunk decompresse.
// En-tete de 16 octets LITTLE-ENDIAN : [u16 type][2 o][u32 taille][u64 horodatage] — decode par
// `filmsource`, dont ce type est la vue interne du decodeur.
type packet struct {
	chunk, idx int
	typ        int
	ts         uint64
	payload    []byte
}

// packetType0 / packetTypeKeyframe / packetTypeBotMeta : les trois seuls types lus ici.
const (
	packetType0        = 0  // replication : events + records ECS. Les morts y sont 93/93.
	packetTypeKeyframe = 2  // declaration complete du monde a un instant
	packetTypeBotMeta  = 12 // BOT_METADATA : nbBots, slot, identifiant, nom (RE_LOG 7ter.62)
)

// packetTypeChunkEnd : CHUNK_END, le terminateur d un chunk de donnees. AUCUN consommateur de ce
// paquet ne le lit — il est filtre a l entree, cf. [packetsOf].
const packetTypeChunkEnd = 7

// film : les octets d un film, deja decompresses, prets a decoder.
type film struct {
	chunks  [][]byte // par POSITION de chunk dans la source, decompresses
	packets []packet
	t0      []packet // paquets type-0, tries par horodatage
	tsBase  uint64
}

// loadFilm : traduit un film deja charge par `filmsource` dans le vocabulaire du decodeur, et
// trie les paquets type-0 par horodatage. AUCUNE lecture disque, AUCUN inflate : tout est fait.
func loadFilm(src *filmsource.Film) (*film, error) {
	if src == nil || src.NumChunks() == 0 {
		return nil, ErrNoChunk
	}
	n := src.NumChunks()
	f := &film{chunks: make([][]byte, n), packets: packetsOf(src)}
	for ch := 0; ch < n; ch++ {
		f.chunks[ch] = src.Chunk(ch)
	}
	for i := range f.packets {
		if f.packets[i].typ == packetType0 {
			f.t0 = append(f.t0, f.packets[i])
		}
	}
	if len(f.t0) == 0 {
		return nil, ErrNoPacket
	}
	sort.Slice(f.t0, func(i, j int) bool { return f.t0[i].ts < f.t0[j].ts })
	f.tsBase = f.t0[0].ts
	return f, nil
}

// packetsOf : les paquets du film dans la forme interne, LE TERMINATEUR EXCLU.
//
// LE FILTRE DE TYPE 7 EST LA POUR L IDENTITE, et il est le seul ecart entre les deux grammaires :
// `filmsource` EMET le paquet CHUNK_END (regle 3 de D3 revisee, comme `filmdec` le faisait), la ou
// l ancien `splitPackets` de ce paquet s arretait sur `taille <= 0` et ne l emettait donc jamais
// (sur les chunks de donnees, « taille 0 » et « CHUNK_END » sont LE MEME paquet, en derniere
// position — mesure sur 1 378 films, cf. `filmsource/doc.go`). Le filtrer ici reproduit
// EXACTEMENT l ancien jeu de paquets, jusqu au rang `idx` : le terminateur etant le dernier paquet
// de son chunk, le retirer ne decale aucun rang.
//
// Ce filtre est une precaution, pas une necessite : les trois consommateurs de `f.packets`
// selectionnent deja par type (`packetType0` pour `t0` et le scan, `packetTypeKeyframe` pour la
// timeline, `packetTypeBotMeta` pour les bots), donc un type 7 ne traverserait de toute facon
// aucun d eux. Il est ecrit pour que l identite tienne AUSSI sur les comptes intermediaires
// (`len(f.packets)`), et pour qu un futur lecteur de `f.packets` sans filtre de type herite du
// meme jeu qu avant.
func packetsOf(src *filmsource.Film) []packet {
	all := src.AllPackets()
	out := make([]packet, 0, len(all))
	for i := range all {
		p := &all[i]
		if p.Type == packetTypeChunkEnd {
			continue
		}
		out = append(out, packet{chunk: p.Chunk, idx: p.Index, typ: p.Type, ts: p.TS, payload: p.Payload})
	}
	return out
}

// ms : instant d un paquet, en millisecondes depuis le premier paquet type-0 du film.
func (f *film) ms(p *packet) int { return int((p.ts - f.tsBase) / 1000) }

// hasEvents : le paquet porte-t-il une liste d evenements ? Le bit 1 du payload le dit.
func hasEvents(p *packet) bool { return bitAt(p.payload, 1) != 0 }

// bitAt : lecture MSB-first d un bit. Hors tampon = 0, comme le lecteur de bits du moteur.
func bitAt(d []byte, p int) int {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return int(d[p>>3]>>uint(7-(p&7))) & 1
}

// bits32 : lecture MSB-first de 32 bits a la position `p`.
func bits32(d []byte, p int) uint32 {
	i, sh := p>>3, uint(p&7)
	var v uint64
	for k := 0; k < 5; k++ {
		v <<= 8
		if i+k < len(d) {
			v |= uint64(d[i+k])
		}
	}
	return uint32(v >> (8 - sh))
}

// bitsN : lecture MSB-first de n bits (n <= 8) a la position `p`.
func bitsN(d []byte, p, n int) int {
	v := 0
	for k := 0; k < n; k++ {
		v = v<<1 | bitAt(d, p+k)
	}
	return v
}
