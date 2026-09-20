package replay

// filmfacts_gardes.go — LES QUATRE CANAUX GARDES PAR L APPELANT, ET RIEN D AUTRE.
//
// Extrait de `filmfacts_canaux.go` le 2026-09-18 : celui-ci a franchi les 500 lignes en gagnant
// les morts d objet des vehicules, et la table de `film_file_size_test.go` est DATEE ET FERMEE au
// 2026-09-16 — un fichier neuf ne s y inscrit pas, il se coupe.
//
// LA COUPE SUIT UNE FRONTIERE REELLE : dans `filmfacts_canaux.go`, les canaux que le BALAYAGE
// remplit toujours ; ici, les quatre que seule une GARDE DE MODE de l appelant remplit — le
// marqueur de portage du drapeau (CTF), l etat des zones (KOTH / Strongholds) et l anneau
// d armement de la bombe (Assaut).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ---------------------------------------------------------------------------
// LES QUATRE CANAUX GARDES PAR L APPELANT (lot 4.1.1-b, 2026-09-17)
// ---------------------------------------------------------------------------

// encodeGardesDeMode / decodeGardesDeMode serialisent les QUATRE canaux que le codec ne portait
// pas : le marqueur de portage du drapeau, l etat des zones avec son temoin, et l anneau
// d armement de la bombe.
//
// # POURQUOI ILS N Y ETAIENT PAS, ET POURQUOI C ETAIT UN TROU
//
// La table `champsNonTransportes` les declarait absents « parce que le fixture ne fournit AUCUNE
// garde » : les trois calques ne se balaient que si `Options.Flag` / `.Zone` / `.Bomb` portent la
// garde de mode, et le fixture d entrees n en fournit pas. C etait vrai POUR UN FIXTURE et FAUX
// EN PRODUCTION — tout CTF remplit `FlagMarks`, tout KOTH/Strongholds `ZoneReads`, tout Assaut
// armable `BombReads`. Le codec passant en production au lot 4.1.1-a, l absence de ces canaux
// aurait rendu un artefact rejoue SANS calque de drapeau, SANS zones et SANS armement sur les
// films qui en portent : la table est donc VIDEE et son mecanisme SUPPRIME avec sa derniere
// entree (meme doctrine que les trois allowlists de `film_layers_deps_test.go`).
//
// `ZoneScanned` VOYAGE AVEC `ZoneReads`, et il le faut : une liste vide et un balayage QUI N A
// PAS EU LIEU ne disent pas la meme chose, et la couverture publie la difference (meme lecon que
// le temoin `Scanned` de la v13).
func encodeGardesDeMode(w *gwriter, in FilmInputs) {
	encodeCarrierMarkScan(w, in.FlagMarks)
	encodeManagedPropertyReads(w, in.ZoneReads)
	w.bool8(in.ZoneScanned)
	// LA JAUGE DE RETOUR DU DRAPEAU VOYAGE ICI, ET PAS AVEC `ZoneReads` : c est le MEME canal
	// de film (`ti=13`), mais ce sont DEUX entrees, gardees par deux modes differents et lues
	// par deux calques. Les fondre ferait publier a un CTF la couverture de zones d un KOTH.
	encodeManagedPropertyReads(w, in.FlagGauge)
	w.bool8(in.FlagGaugeScanned)
	w.u(uint64(len(in.BombReads)))
	for _, b := range in.BombReads {
		w.u(uint64(b.Slot))
		w.i(int64(b.TMS))
		w.byte8(b.Q)
		w.bool8(b.Chained)
	}
}

func decodeGardesDeMode(r *greader, in *FilmInputs) {
	in.FlagMarks = decodeCarrierMarkScan(r)
	in.ZoneReads = decodeManagedPropertyReads(r)
	in.ZoneScanned = r.bool8()
	in.FlagGauge = decodeManagedPropertyReads(r)
	in.FlagGaugeScanned = r.bool8()
	n := int(r.u())
	in.BombReads = make([]types.NavpointRadialRead, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		in.BombReads = append(in.BombReads, types.NavpointRadialRead{
			Slot: uint32(r.u()), TMS: int32(r.i()), Q: r.byte8(), Chained: r.bool8(),
		})
	}
	if len(in.BombReads) == 0 {
		in.BombReads = nil
	}
}

// encodeCarrierMarkScan / decodeCarrierMarkScan : les marques de portage du drapeau ET leur
// DENOMINATEUR (les instants de chaque image-cle balayee, marque ou non). Sans le denominateur,
// « 3 portages confirmes » ne dit pas si les autres ont ete observes — c est ecrit en tete de
// `grammar.CarrierMarkScan`, et c est pourquoi les quatre champs voyagent ensemble.
func encodeCarrierMarkScan(w *gwriter, s grammar.CarrierMarkScan) {
	w.u(uint64(len(s.Marks)))
	var last uint64
	for _, m := range s.Marks {
		w.u(m.TimestampUS - last) // instants non decroissants dans l ordre du film
		last = m.TimestampUS
		w.u(uint64(m.Slot))
	}
	w.u(uint64(len(s.KeyframeUS)))
	last = 0
	for _, ts := range s.KeyframeUS {
		w.u(ts - last)
		last = ts
	}
	w.u(uint64(s.Records))
	w.u(uint64(s.BipedRecords))
}

func decodeCarrierMarkScan(r *greader) grammar.CarrierMarkScan {
	var s grammar.CarrierMarkScan
	n := int(r.u())
	s.Marks = make([]grammar.CarrierMark, 0, n)
	var last uint64
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		s.Marks = append(s.Marks, grammar.CarrierMark{TimestampUS: last, Slot: uint32(r.u())})
	}
	if len(s.Marks) == 0 {
		s.Marks = nil
	}
	n = int(r.u())
	s.KeyframeUS = make([]uint64, 0, n)
	last = 0
	for k := 0; k < n && r.err == nil; k++ {
		last += r.u()
		s.KeyframeUS = append(s.KeyframeUS, last)
	}
	if len(s.KeyframeUS) == 0 {
		s.KeyframeUS = nil
	}
	s.Records, s.BipedRecords = int(r.u()), int(r.u())
	return s
}

// encodeManagedPropertyReads / decodeManagedPropertyReads serialisent UNE liste de lectures de
// `ti=13`. DEUX CANAUX LES EMPLOIENT — l etat des zones et la jauge de retour du drapeau — et
// c est exactement pourquoi la boucle est ecrite ICI, une seule fois : deux copies de huit champs
// divergeraient au premier champ ajoute, et le fichier de faits se relirait decale.
func encodeManagedPropertyReads(w *gwriter, rs []grammar.ManagedPropertyRead) {
	w.u(uint64(len(rs)))
	for _, z := range rs {
		w.u(uint64(z.Slot))
		w.u(z.TimestampUS)
		w.i(int64(z.Field))
		w.i(int64(z.FilmIndex))
		w.i(int64(z.Tag))
		w.u(z.Value)
		w.bool8(z.HasValue)
		w.bool8(z.Chained)
	}
}

func decodeManagedPropertyReads(r *greader) []grammar.ManagedPropertyRead {
	n := int(r.u())
	out := make([]grammar.ManagedPropertyRead, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		out = append(out, grammar.ManagedPropertyRead{
			Slot:        uint32(r.u()),
			TimestampUS: r.u(),
			Field:       grammar.ManagedPropertyField(r.i()),
			FilmIndex:   int(r.i()),
			Tag:         int(r.i()),
			Value:       r.u(),
			HasValue:    r.bool8(),
			Chained:     r.bool8(),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
