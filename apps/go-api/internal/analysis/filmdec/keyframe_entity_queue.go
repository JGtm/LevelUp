package filmdec

// keyframe_entity_queue.go — lot R6 (2026-08-17). PONT entre la capture live « keyframe »
// de juillet et les paquets REELS d'un film, et marche de la boucle de records portee sur
// le PREMIER paquet delta d'une session.
//
// POURQUOI CE FICHIER EXISTE. Le lot R5 avait laisse une condition de reprise : « decompiler
// le consommateur du payload type-2, qui alimenterait une file par-entite ». La lecture du
// jeu (journal RE `.ai/V7.5/killweapon/WALK_PORT_NOTES.md`, section « file par entite ») a
// tranche autrement :
//
//   - la file par-entite n'est PAS une transformation : son item porte une COPIE du tampon
//     du paquet et le BIT DE DEPART du record ; le drain rejoue la MEME grammaire ;
//   - l'aiguillage par type de paquet du lecteur de film n'a AUCUN handler pour le type 2 ;
//     le handler du type 1 saute le payload type-1 PUIS le payload suivant — c'est-a-dire le
//     bloc type-2. Le jeu ne relit donc jamais l'image-cle du film.
//
// Ce que le jeu utilise a la place est le PREMIER paquet de type 0 d'une session, et c'est
// lui que ce fichier permet de mesurer. Aucune adresse de binaire n'est cablee ici : les
// adresses vivent dans le journal RE.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// KFQPacketRef designe un paquet de film retrouve par son CONTENU (reconciliation d'un
// tampon capture en memoire avec un objet nomme du film).
type KFQPacketRef struct {
	Film   string
	Chunk  int
	Packet FilmPacket
}

// FindPackets balaie UNE SEULE FOIS les films presents sous root et rend, pour chaque
// predicat, les paquets qui le satisfont. Lecture seule ; les chunks illisibles sont
// ignores (un cache de films contient des telechargements partiels).
//
// UN SEUL BALAYAGE, ET C'EST LE POINT : le cache complet fait des dizaines de gigaoctets
// et chaque chunk_00 se decompresse. Poser deux questions en deux appels doublait le cout
// pour rien.
//
// C'est l'instrument de reconciliation C0 : il repond « ce tampon capture en RAM est le
// paquet #N de type T du film F », ou bien il ne rend rien — et l'absence est un resultat.
func FindPackets(root string, preds []func([]byte) bool) ([][]KFQPacketRef, error) {
	films, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("racine de films illisible %s : %w", root, err)
	}
	out := make([][]KFQPacketRef, len(preds))
	for _, f := range films {
		if !f.IsDir() {
			continue
		}
		dir := filepath.Join(root, f.Name())
		for c := 0; c <= CountFilmChunks(dir); c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				pay := pk.Payload(data)
				for i, ok := range preds {
					if ok(pay) {
						out[i] = append(out[i], KFQPacketRef{Film: f.Name(), Chunk: c, Packet: pk})
					}
				}
			}
		}
	}
	return out, nil
}

// KFQEqual rend le predicat « payload EXACTEMENT egal a want ».
func KFQEqual(want []byte) func([]byte) bool {
	return func(pay []byte) bool { return len(pay) == len(want) && bytes.Equal(pay, want) }
}

// KFQPrefix rend le predicat « payload COMMENCE par want ». Utile quand la coincidence
// exacte echoue : il dit si la structure est la meme (meme tete de paquet) sans que le
// contenu le soit.
func KFQPrefix(want []byte) func([]byte) bool {
	return func(pay []byte) bool {
		return len(want) > 0 && len(pay) >= len(want) && bytes.Equal(pay[:len(want)], want)
	}
}

// FirstPacketOfType rend le chunk decompresse et le PREMIER paquet de type typ d'un film,
// en parcourant les chunks dans l'ordre. chunk == 0 (le registre) est inclus : certains
// films y logent deja des paquets.
func FirstPacketOfType(dir string, typ uint16) ([]byte, FilmPacket, int, error) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, FilmPacket{}, 0, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == typ {
				return data, pk, c, nil
			}
		}
	}
	return nil, FilmPacket{}, 0, fmt.Errorf("aucun paquet de type %d dans %s", typ, dir)
}

// KFQVariant est une combinaison de CADRE de la boucle de records — les trois grandeurs qui
// ne sont pas dans le corps d'un record et que le film ne porte pas :
//
//	IDLowBits   largeur du champ id bas (valeur de RUNTIME, FUN_1406d310c)
//	Preamble    bits d'amorce en tete de paquet (DefaultPacketPreambleBits = 2)
//	ExtraFields prologue de MODE FILM : R(32) en tete de chaque iteration de record, plus
//	            R(1)[+R(8)] avant les branches NEW et DEL. Lu dans FUN_1406cd128 sous la
//	            garde FUN_14076cea8() ; porte par FrameConfig.HasExtraFields.
type KFQVariant struct {
	IDLowBits   int
	Preamble    int
	ExtraFields bool
}

// KFQWalk est le resultat MESURE d'une traversee de paquet par la boucle de records portee.
// Tous les champs sont des denominateurs publiables : c'est le seul point de ce fichier.
type KFQWalk struct {
	// Variant est la combinaison de cadre utilisee pour cette marche.
	Variant KFQVariant
	// Records / New / Del / Delta comptent les records rendus par la boucle.
	Records, New, Del, Delta int
	// CleanNew compte les records NEW dont la traversee n'a PAS desynchronise.
	CleanNew int
	// EndBit est la position atteinte, TotalBits la taille du paquet en bits.
	EndBit, TotalBits int
	// Overrun dit que la marche a lu AU-DELA de la fin du tampon. Le BitReader rend des
	// zeros passe la fin (rembourrage de queue du moteur) : une marche qui deborde ne
	// progresse pas, elle broute du vide. Sans ce drapeau une combinaison absurde gagne le
	// balayage (mesure du 2026-08-17 sur `0014603f` : 12 317 % de « couverture »).
	Overrun bool
	// Stop nomme la cause d'arret ("fin propre" ou le message d'erreur de la boucle).
	Stop string
	// ByTI compte les NEW propres par typeIndex.
	ByTI map[uint32]int
}

// Coverage rend la fraction du paquet consommee (0..1). Une marche qui deborde rend 0 :
// elle n'a rien consomme de reel, elle a lu du rembourrage.
func (w KFQWalk) Coverage() float64 {
	if w.TotalBits == 0 || w.Overrun {
		return 0
	}
	return float64(w.EndBit) / float64(w.TotalBits)
}

// SortedTIs rend les typeIndex de ByTI, tries — pour une sortie de mesure stable.
func (w KFQWalk) SortedTIs() []int {
	out := make([]int, 0, len(w.ByTI))
	for ti := range w.ByTI {
		out = append(out, int(ti))
	}
	sort.Ints(out)
	return out
}

// WalkPacketRecords rejoue la boucle de records PORTEE (DecodeFrameRecords) sur un payload
// de paquet, avec la largeur d'id donnee, et rend la mesure. Le World est neuf : c'est
// exactement la situation du jeu au premier paquet d'une session.
//
// Aucune bascule globale n'est touchee ici : l'appelant reste maitre des hooks.
func WalkPacketRecords(reg *Registry, pay []byte, v KFQVariant) KFQWalk {
	return WalkPacketRecordsWithWorld(NewWorld(reg), pay, v)
}

// WalkPacketRecordsWithWorld est la meme marche, sur un World DEJA peuple — la seule facon
// de tester l'hypothese « le premier paquet suppose un etat initial venu d'ailleurs ».
// Le World passe est MODIFIE par la marche (liaisons), comme dans le jeu.
func WalkPacketRecordsWithWorld(w *World, pay []byte, v KFQVariant) KFQWalk {
	cfg := DefaultFrameConfig()
	cfg.IDLowBits = v.IDLowBits
	cfg.PacketPreambleBits = v.Preamble
	cfg.HasExtraFields = v.ExtraFields
	br := NewBitReader(pay)
	recs, err := DecodeFrameRecords(br, w, cfg)

	out := KFQWalk{
		Variant:   v,
		Records:   len(recs),
		EndBit:    br.BitPos(),
		TotalBits: len(pay) * 8,
		Stop:      "fin propre",
		ByTI:      map[uint32]int{},
	}
	if err != nil {
		out.Stop = err.Error()
	}
	if out.EndBit > out.TotalBits {
		out.Overrun = true
		out.Stop = "depassement du tampon : " + out.Stop
	}
	for _, r := range recs {
		switch r.Type {
		case recNew:
			out.New++
			if r.DesyncAt == -1 {
				out.CleanNew++
				out.ByTI[r.TypeIndex]++
			}
		case recDel:
			out.Del++
		case recDelta:
			out.Delta++
		}
	}
	return out
}

// KFQFrameVariants enumere les combinaisons de cadre a prober : les largeurs d'id de from a
// to, les amorces 0/1/2, avec et sans le prologue de mode film.
func KFQFrameVariants(from, to int) []KFQVariant {
	var out []KFQVariant
	for _, extra := range []bool{false, true} {
		for pre := 0; pre <= 2; pre++ {
			for n := from; n <= to; n++ {
				out = append(out, KFQVariant{IDLowBits: n, Preamble: pre, ExtraFields: extra})
			}
		}
	}
	return out
}

// BestVariant balaie les combinaisons de cadre et rend celle qui consomme le plus de bits
// sans desync, avec la liste complete. Ces grandeurs sont du RUNTIME (elles ne sont pas dans
// le film) : les balayer et PUBLIER la retenue est la seule facon honnete de les fixer.
// seed, s'il n'est pas nil, fabrique le World de depart (test de l'hypothese « etat initial
// venu d'ailleurs ») ; nil = World neuf.
func BestVariant(reg *Registry, pay []byte, vs []KFQVariant, seed func() *World) (KFQWalk, []KFQWalk) {
	var all []KFQWalk
	var best KFQWalk
	for _, v := range vs {
		w0 := NewWorld(reg)
		if seed != nil {
			w0 = seed()
		}
		w := WalkPacketRecordsWithWorld(w0, pay, v)
		all = append(all, w)
		if !w.Overrun && w.EndBit > best.EndBit {
			best = w // une marche qui deborde n'est pas un progres : elle lit du rembourrage
		}
	}
	return best, all
}

// AllPacketsOfType rend tous les paquets de type typ d'un film, avec le chunk decompresse
// qui les porte. Sert les contre-listes : un film peut contenir plusieurs sessions.
func AllPacketsOfType(dir string, typ uint16) ([][]byte, []FilmPacket, []int) {
	var chunks [][]byte
	var pkts []FilmPacket
	var idx []int
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == typ {
				chunks = append(chunks, data)
				pkts = append(pkts, pk)
				idx = append(idx, c)
			}
		}
	}
	return chunks, pkts, idx
}

// KFQAnchorShape mesure la FORME des ancres de la table type-2 : le mot de 32 bits a +32 de
// chaque ancre retenue par le balayeur, et l'ecart en bits jusqu'a l'ancre suivante.
// Sert le controle H2 du plan R6 (« bitstream ou table d'octets ? ») : un ecart toujours
// multiple de 8 dirait table d'octets.
type KFQAnchorShape struct {
	Records    int
	GapMod8    map[int]int // ecart entre ancres consecutives, modulo 8
	GapValues  map[int]int // ecart exact -> compte
	BitAligned int         // ancres dont la position est multiple de 8
}

// MeasureKeyframeAnchors applique le balayeur de la table type-2 a un payload et rend la
// forme de ses ancres. Il REUTILISE WalkKeyframeWorld : aucun second balayeur.
func MeasureKeyframeAnchors(pay []byte) KFQAnchorShape {
	recs := WalkKeyframeWorld(pay)
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	out := KFQAnchorShape{
		Records:   len(recs),
		GapMod8:   map[int]int{},
		GapValues: map[int]int{},
	}
	for i, r := range recs {
		if r.Bit%8 == 0 {
			out.BitAligned++
		}
		if i+1 < len(recs) {
			gap := recs[i+1].Bit - r.Bit
			out.GapMod8[gap%8]++
			out.GapValues[gap]++
		}
	}
	return out
}
