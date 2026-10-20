package signaux

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// statborg_film.go — la lecture du statborg sur un FILM DEJA CHARGE : les chunks que le
// manifeste decrit, leurs paquets FRAME, l'horloge du manifeste, le plafond et le tri.

// LectureDuStatborg est ce que la methode du statborg rend d'un film : les enregistrements, le
// temoin de troncature, de quoi dire qu'un film n'est pas datable, et les comptes de ses deux
// replis — en donnees, comme le rapport du contexte (`grammar.ComptesDesReplis`) : le registre des replis
// vit au-dessus de cette couche, et la couche des faits porte ces comptes jusqu'au versement.
type LectureDuStatborg struct {
	// Records : les enregistrements d'entite, tries par temps, puis slot, puis manche.
	Records []types.StatRecord
	// Tronque : le plafond [StatborgEnregistrementsMax] a ete atteint ; la lecture s'est arretee
	// au chunk [LectureDuStatborg.ChunkTronque] et Records porte ce qui avait ete lu.
	Tronque bool
	// ChunkTronque : le numero au manifeste du chunk ou le plafond a ete atteint ; sans objet sans
	// troncature.
	ChunkTronque int
	// ChunksDatables : le nombre de chunks que le manifeste decrit. Zero : le film porte peut-etre
	// des paquets mais aucun `start_ms` — « on ne sait pas dater ce film », un fait distinct de
	// « ce film ne porte rien », et le second se repare (le manifeste se retelecharge).
	ChunksDatables int
	// EnregistrementsAbandonnes : `repli_enregistrement_statborg_abandonne` — en-tetes
	// d enregistrement reconnus dont aucun composant ne se decode ou dont un compteur sort du
	// domaine, abandonnes ([scanFrameAvecReplis]).
	EnregistrementsAbandonnes int
	// ComposantsArretes : `repli_composants_statborg_arretes` — enregistrements GARDES dont la
	// lecture des composants s est arretee avant le dernier annonce ([decodeComponentsAvecArret]).
	ComposantsArretes int
}

// LireLeStatborg decode tous les enregistrements d'entite d'un film, tries par temps puis par
// slot, sous le plafond [StatborgEnregistrementsMax]. L'ancrage est DIRECT : les contraintes de
// l'en-tete suffisent a localiser un enregistrement, aucune traversee de la chaine n'est
// necessaire.
//
// LE FILM ARRIVE DEJA CHARGE, et seuls les chunks du MANIFESTE sont balayes ([manifestChunks]) :
// chaque paquet FRAME est date par le `start_ms` de son chunk, l'origine etant le premier paquet
// FRAME du chunk.
func LireLeStatborg(film *source.Film) LectureDuStatborg {
	chunks := manifestChunks(film)
	l := LectureDuStatborg{ChunksDatables: len(chunks)}
	var out []types.StatRecord
	for _, c := range chunks {
		frames := framesOf(film, c.pos)
		if len(frames) == 0 {
			continue
		}
		base := frames[0].TS
		for _, f := range frames {
			tMS := c.meta.StartMS + int((f.TS-base)/1000)
			out = append(out, scanFrameAvecReplis(f.Payload, tMS, &l)...)
			if len(out) >= StatborgEnregistrementsMax {
				l.Records, l.Tronque, l.ChunkTronque = sortRecords(out), true, c.meta.Index
				return l
			}
		}
	}
	l.Records = sortRecords(out)
	return l
}

// sortRecords ordonne les enregistrements par temps puis par slot.
func sortRecords(out []types.StatRecord) []types.StatRecord {
	slices.SortStableFunc(out, func(a, b types.StatRecord) int {
		return cmp.Or(cmp.Compare(a.TimeMS, b.TimeMS), cmp.Compare(a.Slot, b.Slot), cmp.Compare(a.Round, b.Round))
	})
	return out
}

// Types de chunk du MANIFESTE, tels que [types.ChunkMeta.ChunkType] les porte. Le pied (3) est
// celui des temps forts, et il se reconnait par `finalise.EstTempsForts` ([footerData]).
//
// ZERO N'EST PAS UN TYPE : c'est ce que `source.LoadDir` synthetise pour un `chunk_NN.bin`
// PRESENT au cache mais ABSENT du manifeste. Mesure du 2026-09-02 sur les 1 380 manifestes du
// cache : trois valeurs seulement — 1 pour l'en-tete (`chunk_00`, le registre), 2 pour les
// chunks de jeu, 3 pour le pied — et jamais 0. Un chunk hors manifeste n'a donc ni type ni
// `start_ms` connus, et ces lectures ne le consomment pas (cf. [manifestChunks]).
const (
	chunkTypeInconnu = 0
	chunkTypeJeu     = 2
)

// typeDeTrame est le type des paquets FRAME — l'etat horodate, delta de replication — dans le
// decoupage de `source` (`grammar.PacketTypeDelta`, que ce paquet n'importe pas).
const typeDeTrame = 0

// manifestChunk = un chunk du film QUE LE MANIFESTE DECRIT : sa position dans le film (l'index
// que prennent [source.Film.Chunk] et [source.Film.Packets]) et ses metadonnees.
type manifestChunk struct {
	pos  int
	meta types.ChunkMeta
}

// manifestChunks rend les chunks du film decrits par le manifeste, dans l'ordre du film.
//
// POURQUOI CE FILTRE EXISTE, ET CE QU'IL PRESERVE. Le film charge par `source.LoadDir` porte TOUS
// les fichiers presents au cache ; un fichier de chunk absent du manifeste n'a pas de `start_ms`,
// et serait balaye date FAUX. Un film du cache A ETE dans ce cas (`7b0d89c4`, chunks 31 et 32 au
// 2026-09-02 : un film archive avant sa finalisation, restaure complet le 2026-09-16) ; depuis le
// lot L3 du 2026-09-23, la cuisson refuse un tel film avant de l'ouvrir, et ce filtre reste la
// garde de datation des films charges hors d'elle.
//
// Un film sans aucune metadonnee de manifeste rend une liste VIDE : c'est le seul resultat
// honnete (rien n'est datable), et [LectureDuStatborg.ChunksDatables] le dit.
func manifestChunks(film *source.Film) []manifestChunk {
	if film == nil {
		return nil
	}
	meta := film.Meta()
	out := make([]manifestChunk, 0, len(meta))
	for i, m := range meta {
		if m.ChunkType == chunkTypeInconnu {
			continue
		}
		out = append(out, manifestChunk{pos: i, meta: m})
	}
	return out
}

// framesOf rend les paquets FRAME ([typeDeTrame]) du chunk a la POSITION `pos` dans
// le film, dans l'ordre du chunk : le decoupage est fait une fois par `source`, il ne reste qu'a
// choisir le type.
func framesOf(film *source.Film, pos int) []types.Packet {
	pks := film.Packets(pos)
	out := make([]types.Packet, 0, len(pks))
	for _, p := range pks {
		if p.Type == typeDeTrame {
			out = append(out, p)
		}
	}
	return out
}
