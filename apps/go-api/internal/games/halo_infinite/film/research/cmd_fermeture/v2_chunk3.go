//go:build research

package main

// v2_chunk3.go — LE COMPTE D EVENEMENTS DECLARE DU CHUNK DES TEMPS FORTS (item 1.5 de la carte v2).
//
// Le chunk de type 3 (temps forts) se decoupe en paquets comme un chunk de replication ; son
// paquet de type 9 commence par un u32 BIG-ENDIAN, le nombre d evenements que l ecrivain y a mis
// (port Rust, `parser/v41/chunks/summary/mod.rs:38-50`, rapport du 2026-10-01 §4.7). Notre lecteur
// (`grammar.ParseHighlightEvents`, balayage bit a bit des XUID) ne le lit pas : c est un oracle de
// completude gratuit. La carte le confronte a ce que le lecteur TROUVE dans le meme chunk — tous
// types, puis par type (kill-feed : `kill` ; fil des morts : `death`, et `grammar.ScanDeaths`).
//
// Le chunk est choisi PAR SON TYPE au manifeste (`finalise.EstTempsForts`), la regle de
// `grammar.ScanDeaths` ; un film sans manifeste type n est pas mesure (et le dit).

import (
	"errors"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// paquetTypeResume est le type du paquet qui porte le compte declare.
const paquetTypeResume = 9

// largeurCompteDeclare : le compte declare est un u32.
const largeurCompteDeclare = 4

// chunk3 est la confrontation d UN film.
type chunk3 struct {
	// mesure : le chunk des temps forts a ete trouve et lu.
	mesure bool
	refus  string
	chunk  int
	// paquets9 : paquets de type 9 portant un compte ; declares : la somme de leurs comptes.
	paquets9, declares int
	// trouves : evenements rendus par le lecteur, par type ; morts : `grammar.ScanDeaths`.
	trouves, kills, deaths, medailles, modes, autres, filDesMorts int
}

// errSansTempsForts : aucun chunk n est type « temps forts » au manifeste.
var errSansTempsForts = errors.New("aucun chunk type temps forts au manifeste")

// mesurerChunk3 recharge le film de `dir` AVEC son manifeste (`filmcache.LoadFilmDir`, le chemin
// de la cuisson : `<racine>/film_manifests/<short8>.json` type chaque chunk) et confronte le compte
// declare aux evenements trouves. Le contexte de la carte, lui, est charge sans manifeste
// (`grammar.ContexteDeFilm`) : il ne type aucun chunk.
func mesurerChunk3(dir string) chunk3 {
	film, ok, err := filmcache.LoadFilmDir(dir)
	switch {
	case err != nil:
		return chunk3{refus: err.Error()}
	case !ok:
		return chunk3{refus: "manifeste absent du cache"}
	}
	return confronterChunk3(film)
}

// confronterChunk3 confronte, sur un film charge avec son manifeste, le compte declare aux
// evenements trouves.
func confronterChunk3(film *source.Film) chunk3 {
	n := -1
	for _, m := range film.Meta() {
		if finalise.EstTempsForts(m.ChunkType) {
			n = m.Index
		}
	}
	if n < 0 {
		return chunk3{refus: errSansTempsForts.Error()}
	}
	raw, pks, ok := grammar.FilmChunkAt(film, n)
	if !ok {
		return chunk3{chunk: n, refus: "chunk des temps forts absent du film"}
	}
	out := chunk3{mesure: true, chunk: n}
	for _, pk := range pks {
		if pk.Type == paquetTypeResume && pk.Size >= largeurCompteDeclare {
			out.paquets9++
			out.declares += int(source.U32BE(pk.Payload(raw), 0))
		}
	}
	evs, err := grammar.ParseHighlightEvents(raw, grammar.HighlightProfileOfFilm(film).MajorVersion)
	if err != nil {
		out.refus = err.Error()
	}
	out.trouves = len(evs)
	for _, e := range evs {
		switch e.EventType {
		case highlightevent.EventTypeKill:
			out.kills++
		case highlightevent.EventTypeDeath:
			out.deaths++
		case highlightevent.EventTypeMedal:
			out.medailles++
		case highlightevent.EventTypeMode:
			out.modes++
		default:
			out.autres++
		}
	}
	if morts, err := grammar.ScanDeaths(film); err == nil {
		out.filDesMorts = len(morts)
	}
	return out
}
