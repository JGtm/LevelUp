//go:build research

package objectives

// instruments_pied_research_test.go — LE PIED BRUT DES SONDES DE L'ASSAUT
// (`assaut_*_research_test.go`).
//
// Ces sondes relisent les blocs du pied de film au bit pres, sans le filtre `th == 10` de la
// production, pour chercher ce que l'Assaut y ecrit. La production de ce paquet ne lit aucun octet
// (ADR 0037, D-2 amende) : elle consomme `signaux.FooterEvents`. Ce fichier porte la COPIE DE TEST
// de la selection du pied (`signaux.footerData`) et des bornes de xuid de son balayage
// (`grammar/signaux/pied_de_film.go`) que les sondes appellent : un instrument de plus passe par
// ici, il ne recopie pas une troisieme fois (CLAUDE.md regle 6).

import (
	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// Bornes plausibles d'un xuid Halo (filtre anti-bruit du balayage du pied).
const (
	minXUID = uint64(2e15)
	maxXUID = uint64(3e15)
)

// footerData rend le contenu DECOMPRESSE du pied (chunk de plus haut numero, type des temps forts
// au manifeste), ou (nil, false).
func footerData(film *source.Film) ([]byte, bool) {
	pos, numero := -1, -1
	for _, c := range chunksDuManifeste(film) {
		if finalise.EstTempsForts(c.meta.ChunkType) && c.meta.Index > numero {
			pos, numero = c.pos, c.meta.Index
		}
	}
	if pos < 0 {
		return nil, false
	}
	return film.Chunk(pos), true
}
