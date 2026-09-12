package filmdec

// film_major_version.go — L'INDICATEUR QUI DIT COMMENT LE FILM EST CONSTRUIT.
//
// # CE QU'IL EST
//
// Les quatre premiers octets du REGISTRE (`chunk_00.bin`), lus en u32 little-endian, sont le
// `FilmMajorVersion` du film — la MEME valeur que l'API publie dans le manifeste de spectate
// (`CustomData.FilmMajorVersion`, cf. `sync/haloclient/halo_client_film.go`). Le second u32 le
// suit (25 / 24 / 27 sur les films de version 40 / 37 / 41), puis la chaine
// `game-engine-team-mapping-component` ouvre le premier bloc d'archetype.
//
// Verifie sur pieces le 2026-09-12, en-tete brut de trois films du cache :
//
//	e5adf7b2 (2025-07)  28 00 00 00 | 19 00 00 00 | "game-engine-team-mapping..."  -> v40
//	a26dbcdb (2024-10)  25 00 00 00 | 18 00 00 00 | idem                           -> v37
//	5676a9ba (2026-07)  29 00 00 00 | 1b 00 00 00 | idem                           -> v41
//
// # POURQUOI IL VIT ICI
//
// `filmsource` est volontairement AVEUGLE au contenu (« les chunks bruts, et rien d'autre », et
// `archlint/filmsource_leaf_test.go` le maintient feuille) ; `filmdec` est le paquet qui porte la
// SEMANTIQUE de `chunk_00` — `registry.go` l'analyse, [FilmRegistryChunk] le localise. La lecture
// de l'en-tete du registre appartient donc a `filmdec`, et a lui seul.
//
// # CE QU'IL CORRIGE
//
// `analysis.ParseHighlightEvents(chunk, version)` decoupe le gamertag du bloc d'event a l'octet 0
// (`version <= 38 || version >= 41`) ou a l'octet 12 (versions 39-40). Trois appelants passaient
// 0 en dur faute de manifeste : sur les films 39-40 — 211 des 1 351 du cache, mars a novembre
// 2025 — ils lisaient du rembourrage, le roster humain s'effondrait a 2 noms distincts pour 24 a
// 27 joueurs et les portes « indice < nPlay » du decodeur de source de degat rejetaient les trois
// quarts des morts (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md). Ils lisent desormais la
// version ICI plutot que de la deviner.
//
// # NOTE D'HISTOIRE
//
// `registry.go` (fonction `looksZlib`) avait deja MESURE cette valeur sur les 1 378 `chunk_00` du
// cache — 0x29 sur 1 117 films, 0x28 sur 204, 0x27 sur 34, 0x25 sur 13 — mais la lisait comme « le
// `kind` u32 du premier slot ». Le comptage etait juste, l'interpretation non : c'est la version
// du film. Le commentaire de `looksZlib` renvoie desormais ici.

import (
	"encoding/binary"

	"levelup/go-api/internal/analysis/filmsource"
)

// FilmMajorVersionUnknown : la valeur que porte une version non lue. C'est aussi celle que les
// appelants passaient en dur avant le 2026-09-12, et le decoupage « gamertag en tete » que
// `analysis.ParseHighlightEvents` lui applique reste le comportement historique.
const FilmMajorVersionUnknown = 0

// filmMajorVersionOffset : l'octet ou commence l'u32 de version, en tete du registre inflate.
const filmMajorVersionOffset = 0

// FilmMajorVersionFromHeader lit le `FilmMajorVersion` en tete du registre DECOMPRESSE
// (`chunk_00.bin` : le cache n'en porte aucun encore compresse — 0 sur 1 351 mesures le
// 2026-09-12 — mais un appelant qui tient des octets bruts passe par `filmsource.Inflate`, qui
// rend le tampon inchange quand il n'est pas zlib).
//
// ok=false quand le chunk fait moins de quatre octets : l'appelant retombe alors sur
// [FilmMajorVersionUnknown] et DOIT le consigner — une version illisible change les lignes
// produites en aval, elle ne se tait pas.
func FilmMajorVersionFromHeader(chunk0 []byte) (int, bool) {
	if len(chunk0) < filmMajorVersionOffset+4 {
		return FilmMajorVersionUnknown, false
	}
	return int(binary.LittleEndian.Uint32(chunk0[filmMajorVersionOffset:])), true
}

// FilmMajorVersion rend la version d'un film DEJA CHARGE, lue dans son registre.
//
// ok=false quand le film ne porte pas son registre (bobine partielle, fixture sans `chunk_00` —
// `replay/testdata/minifilm_000d5950` est exactement ce cas) ou que cet en-tete est trop court.
func FilmMajorVersion(f *filmsource.Film) (int, bool) {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return FilmMajorVersionUnknown, false
	}
	return FilmMajorVersionFromHeader(reg)
}
