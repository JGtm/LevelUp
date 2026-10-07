package killsource

// chunks.go — L ENTREE DU PAQUET : UN FILM DEJA CHARGE, ET RIEN D AUTRE.
//
// # CE QUE CE FICHIER A CESSE DE FAIRE (lot 1, PLAN_CUISSON_PERF item 1.4, 2026-09-02)
//
// Il portait sa propre source de chunks (`ChunkSource`, `MemoryChunks`, `DirChunks`), son propre
// inflate zlib et son propre marcheur de paquets (`splitPackets`) — la TROISIEME copie des trois,
// a cote de `grammar` et d `objectives`, et les trois DIVERGEAIENT. Une cuisson d artefact
// payait donc une lecture disque et une decompression du film ENTIER rien que pour ce decodeur,
// en plus de celles des balayages.
//
// Tout cela vit desormais dans `internal/games/halo_infinite/film/internal/source`, paquet FEUILLE : l appelant charge
// le film UNE fois et le passe a [Decode]. Ce fichier ne fait plus que poser l origine des instants
// du film et la version de son kill-feed : killsource ne lit aucun octet de paquet, la marche des
// trames, les balayages et le kill-feed passent par la grammaire.
//
// PIEGE HISTORIQUE, ET IL A COUTE UN KILL-FEED ENTIER : tout l outillage de RE bornait la lecture
// au chunk 41. Un film BTB en compte 63, et son chunk HIGHLIGHT est le n62 — le kill-feed y etait
// purement introuvable (RE_LOG 7ter.52). La garantie est intacte, elle a seulement change de
// domicile : `source` ne borne rien, et killsource prend TOUS les chunks du film charge.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// packetType0 : le type des paquets de replication (events + records ECS ; les morts y sont 93/93).
const packetType0 = 0

// film : le film charge, l origine de ses instants et la version de son kill-feed.
type film struct {
	// src : LE FILM CHARGE, et la SEULE porte aux octets de ce paquet (lot 2.4.2, ADR 0034
	// D-2) ; killsource le passe a la grammaire, qui en lit les octets.
	src *source.Film
	// tsBase : l horodatage du premier paquet de replication, l instant 0 de killsource.
	tsBase uint64

	// majorVersion : le FilmMajorVersion LU dans l en-tete du registre (`chunk_00`), et
	// `versionLue=false` quand le film n en porte pas (bobine partielle, fixture). Il commande
	// le decoupage du gamertag dans le kill-feed : sur les versions 39-40 le gamertag vit a
	// l octet 12 du bloc d event et non a l octet 0. Le passer en dur a 0 — ce que faisait
	// `loadKillFeed` avant le 2026-09-12 — lisait du rembourrage sur ces films et effondrait le
	// roster humain a deux noms distincts pour 24 a 27 joueurs.
	majorVersion int
	versionLue   bool
}

// loadFilm : pose l origine des instants du film et la version de son kill-feed. AUCUNE lecture
// disque, AUCUN inflate : tout est fait.
func loadFilm(src *source.Film) (*film, error) {
	if src == nil || src.NumChunks() == 0 {
		return nil, ErrNoChunk
	}
	f := &film{src: src}
	// LA VERSION VIENT DU PROFIL DU FILM DEPUIS LE LOT 2.1.4 : `grammar.HighlightProfileOfFilm`
	// porte la MEME valeur que `FilmMajorVersion` — c est la meme lecture — mais elle la rend
	// avec le NOM de l implantation qu elle selectionne, et c est le profil qui en est
	// desormais la source unique (item 2.1.4 du PLAN_DECODEUR_FILM).
	hl := grammar.HighlightProfileOfFilm(src)
	f.majorVersion, f.versionLue = hl.MajorVersion, hl.Lue
	base, ok := origineDesInstants(src)
	if !ok {
		return nil, ErrNoPacket
	}
	f.tsBase = base
	return f, nil
}

// origineDesInstants rend le plus petit horodatage des paquets de replication du film, faux quand
// il n en porte aucun.
func origineDesInstants(src *source.Film) (uint64, bool) {
	var base uint64
	vu := false
	for _, p := range src.AllPackets() {
		if p.Type == packetType0 && (!vu || p.TS < base) {
			base, vu = p.TS, true
		}
	}
	return base, vu
}

// msDe : un horodatage du film, en millisecondes depuis le premier paquet type-0.
func (f *film) msDe(ts uint64) int { return int((ts - f.tsBase) / 1000) }
