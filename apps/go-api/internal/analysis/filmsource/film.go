package filmsource

// film.go — LE FILM CHARGE : CHUNKS DECOMPRESSES UNE FOIS, PAQUETS DECOUPES UNE FOIS.
//
// La grammaire de decoupage (D3 REVISEE du 2026-09-02) et la mesure qui l'a etablie sont
// documentees en tete de paquet (doc.go) : `.ai/V7.5/PLAN_CUISSON_PERF.md` §3 D3 et
// `.ai/V7.5/MESURES_CUISSON_PERF.md` §2b (1 378 films).

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// packetHeaderSize : [u16 type][2 octets][u32 taille][u64 horodatage], little-endian.
	packetHeaderSize = 16
	// packetTypeChunkEnd : CHUNK_END, le terminateur d'un chunk de donnees. Il est EMIS, puis la
	// marche s'arrete (regle 3). Sur les chunks de donnees il porte une taille 0 et se tient en
	// derniere position, sans un octet apres — mesure sur 1 378 films.
	packetTypeChunkEnd = 7
)

const (
	// inflateRatioHint : ratio de decompression suppose, pour pre-dimensionner le tampon de
	// sortie. Ce n'est PAS une borne : un ratio reel superieur (7,6 mesure sur le chunk_01 de la
	// mini-bobine) coute simplement une croissance de plus.
	inflateRatioHint = 6
	// inflateHintCap : borne haute du pre-dimensionnement, 64 Mio par chunk. Sans elle, un chunk
	// de 89 Mio (le plus gros du cache) reserverait un demi-gigaoctet d'un coup.
	inflateHintCap = 64 << 20
)

// ChunkMeta : metadonnees d'un chunk, telles que le manifeste du film les porte. Ce paquet ne les
// LIT pas (il ne connait ni le cache ni le manifeste) : l'appelant les FOURNIT a [Load].
type ChunkMeta struct {
	Index     int
	ChunkType int
	StartMS   int
}

// Packet : un paquet de replication dans un chunk decompresse.
//
// Payload est une SOUS-TRANCHE du buffer du chunk, JAMAIS une copie. Deux consequences a
// connaitre : ecrire dans le payload modifie le chunk (et reciproquement), et garder un seul
// Packet retient tout le chunk decompresse. C'est le choix qui rend le decodage unique abordable
// en memoire — copier les payloads doublerait le pic.
type Packet struct {
	// Chunk : indice du chunk dans la source ; Index : rang du paquet dans ce chunk (0-based).
	Chunk, Index int
	// Type : le type du paquet (0 = delta/replication, 2 = image-cle, 7 = CHUNK_END...).
	Type int
	// TS : horodatage moteur du paquet, en microsecondes (horloge du film).
	TS uint64
	// Payload : les octets du paquet, sous-tranche du chunk decompresse.
	Payload []byte
}

// Film : un film Theater charge — chunks decompresses et paquets decoupes, une fois pour toutes.
// Un Film est en LECTURE SEULE apres [Load] : ses accesseurs rendent les tranches internes sans
// copie, et personne ne doit y ecrire. Il n'est pas protege contre l'usage concurrent en ecriture,
// ce qui n'a pas d'objet ici (aucune methode ne mute).
type Film struct {
	chunks [][]byte // par indice de chunk, decompresses
	all    []Packet // tous les paquets, ordre chunk PUIS index
	bounds [][2]int // par indice de chunk : [debut, fin) dans all
	meta   []ChunkMeta
}

// NumChunks : nombre de chunks du film.
func (f *Film) NumChunks() int { return len(f.chunks) }

// Chunk : les octets DECOMPRESSES du chunk `i`, ou nil hors bornes.
func (f *Film) Chunk(i int) []byte {
	if i < 0 || i >= len(f.chunks) {
		return nil
	}
	return f.chunks[i]
}

// Packets : les paquets du chunk `i` dans l'ordre du chunk, ou nil hors bornes. La tranche rendue
// est une vue sur le stockage interne : la lire est gratuit, y ajouter est interdit (la capacite
// est bornee a sa longueur, un append allouera au lieu d'ecraser le chunk suivant).
func (f *Film) Packets(i int) []Packet {
	if i < 0 || i >= len(f.bounds) {
		return nil
	}
	b := f.bounds[i]
	if b[0] == b[1] {
		return nil
	}
	return f.all[b[0]:b[1]:b[1]]
}

// AllPackets : tous les paquets du film, ordre chunk PUIS index.
func (f *Film) AllPackets() []Packet { return f.all }

// Meta : les metadonnees de chunk, INDEXEES PAR POSITION dans le film (`Meta()[i]` decrit le
// chunk `i`), ou nil si l'appelant n'en a pas donne et que la source n'en synthetise pas.
//
// [LoadDir] EN SYNTHETISE TOUJOURS : `Meta()[i].Index` y est le numero du fichier `chunk_NN.bin`
// (cf. l'en-tete de paquet, « L'INDEXATION DES CHUNKS »). Meta vide n'arrive donc que pour un
// [Load] sur des chunks en memoire sans manifeste — a ce compte, l'appelant qui a besoin du role
// des chunks doit le dire par une erreur, pas par un resultat vide.
//
// La tranche est une copie faite au chargement (l'appelant peut disposer de la sienne), mais elle
// n'est pas a modifier.
func (f *Film) Meta() []ChunkMeta { return f.meta }

// errNoChunk : une source sans chunk n'est pas un film.
var errNoChunk = errors.New("filmsource: aucun chunk dans la source")

// Load : decompresse tous les chunks de `src` et decoupe leurs paquets, une seule lecture de la
// source. `meta` est POSITIONNEL (`meta[i]` decrit le chunk `i` de `src`) et LICITE A NIL
// (enveloppes de compatibilite, tests) : le film est alors charge sans metadonnees, et c'est au
// consommateur qui en a besoin (numero, type de chunk, start_ms) de le dire par une erreur
// explicite plutot que par un resultat vide. Les appelants qui lisent un REPERTOIRE passent par
// [LoadDir], qui construit cet alignement lui-meme depuis les noms de fichiers.
func Load(src Source, meta []ChunkMeta) (*Film, error) {
	if src == nil {
		return nil, errors.New("filmsource: source nulle")
	}
	n := src.NumChunks()
	if n <= 0 {
		return nil, errNoChunk
	}
	f := &Film{chunks: make([][]byte, n), bounds: make([][2]int, n)}
	for ch := 0; ch < n; ch++ {
		raw, err := src.Chunk(ch)
		if err != nil {
			return nil, fmt.Errorf("filmsource: chunk %d: %w", ch, err)
		}
		d := inflate(raw)
		f.chunks[ch] = d
		start := len(f.all)
		f.all = appendPackets(f.all, d, ch)
		f.bounds[ch] = [2]int{start, len(f.all)}
	}
	if len(meta) > 0 {
		f.meta = append(make([]ChunkMeta, 0, len(meta)), meta...)
	}
	return f, nil
}

// LoadDir : [Load] sur un repertoire `chunk_NN.bin` ([DirSource]), avec les metadonnees ALIGNEES
// SUR LES FICHIERS PRESENTS.
//
// L'INDEX EST TOUJOURS SYNTHETISE DEPUIS LE NOM DU FICHIER, `meta` ou pas : `Meta()[i].Index` est
// le NN de `chunk_NN.bin` a la position `i`. C'est ce qui tranche le piege documente en tete de
// paquet — sur une bobine sans `chunk_00`, la position 0 n'est PAS le registre, et un balayage qui
// confondrait les deux marcherait le premier chunk de donnees comme un registre.
//
// `meta` (le manifeste du film, quand l'appelant l'a) est FUSIONNE PAR NUMERO, jamais par
// position : une entree de manifeste dont le fichier manque au cache est ignoree, et un fichier
// absent du manifeste garde son index synthetise avec un type et un debut a zero. Aligner par
// position serait faux des le premier telechargement partiel — le manifeste liste ce que le
// serveur a, le repertoire porte ce qui est descendu.
func LoadDir(dir string, meta []ChunkMeta) (*Film, error) {
	src, err := newDirSource(dir)
	if err != nil {
		return nil, err
	}
	return Load(src, alignMetaOnNumbers(src.nums, meta))
}

// alignMetaOnNumbers construit les metadonnees POSITIONNELLES d'un repertoire : un [ChunkMeta] par
// fichier, dans l'ordre de la source, portant le numero du fichier et — s'il figure au manifeste —
// le type et le debut que celui-ci lui donne.
func alignMetaOnNumbers(nums []int, meta []ChunkMeta) []ChunkMeta {
	byIndex := make(map[int]ChunkMeta, len(meta))
	for _, m := range meta {
		if _, seen := byIndex[m.Index]; !seen {
			byIndex[m.Index] = m
		}
	}
	out := make([]ChunkMeta, len(nums))
	for i, n := range nums {
		out[i] = ChunkMeta{Index: n}
		if m, ok := byIndex[n]; ok && n != ChunkNumberUnknown {
			out[i].ChunkType, out[i].StartMS = m.ChunkType, m.StartMS
		}
	}
	return out
}

// Inflate decompresse UN chunk brut, exactement comme [Load] le fait pour chacun des siens : une
// entree deja decompressee traverse telle quelle, un flux tronque rend le partiel.
//
// EXPOSEE POUR LES LECTEURS D'UN SEUL CHUNK — les enveloppes de compatibilite (`filmdec`) et les
// outils de recherche qui ouvrent un `chunk_NN.bin` isole sans charger le film entier. Le chemin
// de production, lui, ne l'appelle pas : il charge le film UNE fois par [Load].
func Inflate(raw []byte) []byte { return inflate(raw) }

// inflate : decompresse un chunk zlib. Une entree deja decompressee traverse telle quelle, et un
// flux TRONQUE rend le PARTIEL — un film Theater se termine parfois net (doctrine reprise de
// `killsource`, et la mesure du 2026-09-02 n'a trouve aucun flux tronque sur 1 378 films : c'est
// une garantie de robustesse, pas un cas courant).
//
// Le tampon de sortie est PRE-DIMENSIONNE (Grow) puis rempli par io.Copy, et non par un io.ReadAll
// nu : l'audit a mesure le churn de la croissance par doublement — a chaque realloc, tout le
// tampon est RECOPIE, soit ~2N octets deplaces pour N octets utiles, sur des chunks de plusieurs
// megaoctets et pour chacun des ~40 decodages d'un film.
func inflate(raw []byte) []byte {
	if len(raw) < 2 || raw[0] != 0x78 {
		return raw
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return raw
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if hint := len(raw) * inflateRatioHint; hint > 0 {
		buf.Grow(min(hint, inflateHintCap))
	}
	// io.Copy passe par bytes.Buffer.ReadFrom, qui lit directement dans la capacite reservee :
	// aucun tampon intermediaire, aucune recopie tant que le pre-dimensionnement suffit.
	_, err = io.Copy(&buf, zr)
	out := buf.Bytes()
	if err != nil && len(out) == 0 {
		return raw
	}
	return out
}

// appendPackets : decoupe un chunk decompresse et ajoute ses paquets a `dst`. UNE tranche pour
// tout le film (les vues par chunk sont des sous-tranches), donc une seule croissance amortie.
//
// La grammaire est celle de D3 REVISEE, dans l'ordre exact de ses quatre regles :
//
//	(1) arret si l'en-tete deborde du chunk ;
//	(2) le paquet est emis, taille 0 comprise ;
//	(3) arret APRES un paquet de type 7 (CHUNK_END) ;
//	(4) arret AVANT emission sur une taille 0 qui n'est pas de type 7 (en-tete degenere).
func appendPackets(dst []Packet, d []byte, ch int) []Packet {
	off, k := 0, 0
	for off+packetHeaderSize <= len(d) {
		typ := int(binary.LittleEndian.Uint16(d[off:]))
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		// (1) `size < 0` n'est atteignable que sur une plateforme 32 bits, ou un u32 ne tient pas
		// dans un int ; le garder rend la borne vraie partout.
		if size < 0 || off+packetHeaderSize+size > len(d) {
			break
		}
		// (4) l'en-tete degenere n'existe que dans chunk_00, le registre — jamais marche comme un
		// flux de paquets par un consommateur legitime.
		if size == 0 && typ != packetTypeChunkEnd {
			break
		}
		// (2) emission, taille 0 comprise.
		dst = append(dst, Packet{
			Chunk:   ch,
			Index:   k,
			Type:    typ,
			TS:      ts,
			Payload: d[off+packetHeaderSize : off+packetHeaderSize+size],
		})
		off += packetHeaderSize + size
		k++
		// (3) le terminateur est emis, puis c'est fini pour ce chunk.
		if typ == packetTypeChunkEnd {
			break
		}
	}
	return dst
}
