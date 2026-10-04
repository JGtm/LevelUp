package objectives

// instruments_statborg_test.go — LA LECTURE BRUTE DES INSTRUMENTS DE MESURE DE CE PAQUET.
//
// Les instruments de la phase 0 du lot A (`score_measure_*_test.go`) et du lot 1.9.11
// (`e1911_manches_mesure_research_test.go`) relisent le statborg au bit pres pour le confronter a
// l oracle — la grammaire etendue de `score_measure_rounds_test.go` est une grammaire PARALLELE.
// Ils vivent ici parce que l oracle et l analyse des manches y vivent. La PRODUCTION de ce paquet
// ne lit aucun octet (ADR 0037, D-2 amende) : elle consomme `signaux.LireLeStatborg`.
//
// Ce fichier porte donc une COPIE DE TEST des primitives de `grammar/signaux/statborg.go` et de
// `grammar/signaux/statborg_film.go` que ces instruments appellent. C est la seconde copie, et la
// derniere (CLAUDE.md regle 6) : un instrument de plus passe par ici ; une primitive qui changerait
// dans la grammaire se recopie ici dans le meme commit, sinon l instrument mesurerait une autre
// lecture que la production.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Les constantes de la grammaire d ancrage que les instruments lisent
// (`grammar/signaux/statborg.go`).
const (
	statHdrBits          = 5
	statDenseMaskBits    = 64
	statMaxComp          = 58
	statCompIndexBits    = 6
	statMaxCompPerRecord = 7
	statTailBits         = 64
)

// chunkDuManifeste = un chunk que le manifeste decrit : sa position dans le film et ses
// metadonnees (`signaux.manifestChunk`).
type chunkDuManifeste struct {
	pos  int
	meta types.ChunkMeta
}

// chunksDuManifeste rend les chunks du film decrits par le manifeste, dans l ordre du film : un
// type de chunk nul est un fichier hors manifeste (`signaux.manifestChunks`).
func chunksDuManifeste(film *source.Film) []chunkDuManifeste {
	if film == nil {
		return nil
	}
	var out []chunkDuManifeste
	for i, m := range film.Meta() {
		if m.ChunkType != 0 {
			out = append(out, chunkDuManifeste{pos: i, meta: m})
		}
	}
	return out
}

// tramesDe rend les paquets FRAME (type 0) du chunk a la position `pos` (`signaux.framesOf`).
func tramesDe(film *source.Film, pos int) []types.Packet {
	var out []types.Packet
	for _, p := range film.Packets(pos) {
		if p.Type == 0 {
			out = append(out, p)
		}
	}
	return out
}

// decodeStatComponent lit un composant et rend sa largeur consommee : deux en-tetes de 5 bits,
// deux valeurs a longueur variable, deux drapeaux commandant chacun une valeur conditionnelle
// (FUN_140C18794, `signaux.decodeStatComponent`).
func decodeStatComponent(pay []byte, p int) (types.StatValue, int, bool) {
	q := p + 2*statHdrBits
	a, n1, ok := readStatVarWidth(pay, q)
	if !ok {
		return types.StatValue{}, 0, false
	}
	b, n2, ok := readStatVarWidth(pay, q+n1)
	if !ok {
		return types.StatValue{}, 0, false
	}
	q += n1 + n2
	if q+2 > len(pay)*8 {
		return types.StatValue{}, 0, false
	}
	flags := [2]uint64{source.BitsTronques(pay, q, 1), source.BitsTronques(pay, q+1, 1)}
	q += 2
	out := types.StatValue{A: a, B: b}
	for i, f := range flags {
		if f != 1 {
			continue
		}
		v, n, ok := readStatVarWidth(pay, q)
		if !ok {
			return types.StatValue{}, 0, false
		}
		if i == 0 {
			out.C, out.HasC = v, true
		} else {
			out.D, out.HasD = v, true
		}
		q += n
	}
	return out, q - p, true
}

// readStatVarWidth lit une valeur a longueur variable : un selecteur de 2 bits donne la largeur
// (8 << selecteur), la valeur est signee (FUN_140C18A1C, `signaux.readStatVarWidth`).
func readStatVarWidth(pay []byte, p int) (int64, int, bool) {
	if p < 0 || p+2 > len(pay)*8 {
		return 0, 0, false
	}
	w := 8 << uint(source.BitsTronques(pay, p, 2))
	if w > 32 || p+2+w > len(pay)*8 {
		return 0, 0, false
	}
	v := source.BitsTronques(pay, p+2, w)
	iv := int64(v)
	if w < 32 && v&(1<<uint(w-1)) != 0 {
		iv = int64(v) - (1 << uint(w))
	}
	return iv, 2 + w, true
}
