package killcollector

// bridge.go — PONT ENTRE LE TELECHARGEMENT DE FILM ET LE DECODEUR `killsource`.
//
// Il fait UNE chose : rendre, pour un match, la SEQUENCE de chunks que le decodeur attend. Il ne
// decode pas (c est `games/halo_infinite/film/killsource`), il n ecrit pas (c est
// `internal/persist`), et il ne decide pas QUAND telecharger (c est le collecteur).
//
// # QUELS CHUNKS, ET POURQUOI LA SYNCHRO NE SUFFISAIT PAS TELLE QUELLE
//
// Le manifeste `spectate` d un film declare trois types de chunks :
//
//	type 1  HEADER              en-tete du film
//	type 2  REPLICATION_DATA    le gros du film — les enregistrements de mort ET BOT_METADATA
//	type 3  HIGHLIGHT_EVENTS    le kill-feed — sans lui, aucun couple tueur/victime
//
// Avant ce pont, DEUX appels separes filtraient chacun sur UN type — `GetMatchFilm` le type 2,
// `GetHighlightEventsChunk` le type 3 — et **personne ne telechargeait l en-tete**. La decouverte
// J0.1 (2026-07-31) l a paye comptant : `cmd/fetch_film_chunks` ne le prenait pas non plus, et il
// a fallu recuperer a la main le chunk 0 (type 1) et le footer (type 3) pour amorcer le World du
// decodeur de rejeu. Le pont prend donc la SEQUENCE COMPLETE, via `GetFilmChunks` (une seule
// lecture de manifeste au lieu de deux).
//
// CE QUE `killsource` FAIT DE CHAQUE TYPE, verifie dans le code du decodeur :
//   - il lit TOUS les chunks du film qu on lui donne, sans borne d index (piege historique :
//     l outillage de RE bornait au chunk 41, et le HIGHLIGHT d un film BTB est le n62 — le
//     kill-feed y etait purement introuvable). La decompression et le decoupage en paquets, eux,
//     sont faits UNE fois par `filmsource` avant l appel (lot 1 de PLAN_CUISSON_PERF) ;
//   - il localise le HIGHLIGHT PAR SON CONTENU (le chunk qui produit le plus d evenements
//     `kill`), donc sa position dans la sequence est libre ;
//   - il n a pas besoin de l en-tete. Le lui donner est sans effet pour lui, et c est ce que le
//     collecteur veut : UNE source de chunks, complete, qui servira aussi les autres decodeurs
//     du film (rejeu) sans re-telecharger.
//
// # CE QUE LE PONT NE FAIT PAS
//
// Il ne juge pas l absence de film : `found == false` est un cas NORMAL (tous les matchs n ont
// pas de film, et les films Theater EXPIRENT cote serveur — le manque est definitif). C est au
// collecteur de decider quoi en faire, et sa reponse est « rien, on passe au suivant ».

import (
	"context"
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/sync/haloclient"
)

// filmChunkFetcher : le SEUL besoin du pont cote client. Interface etroite (patron du paquet,
// cf. `highlightChunkFetcher`) — testable sans implementer tout `HaloClient`.
type filmChunkFetcher interface {
	GetFilmChunks(ctx context.Context, matchID string) ([]haloclient.FilmChunk, bool, error)
}

// FilmForMatch assemble et CHARGE le film d un match, dans l ordre du manifeste (en-tete,
// replication, kill-feed).
//
// Le booleen rendu vaut false quand le film n existe pas (404/410 cote API, ou plus aucun chunk
// disponible) : c est un cas NORMAL, pas une erreur.
//
// ⚠ Le decodeur EXIGE le chunk HIGHLIGHT : sans lui il rend `ErrNoKillFeed`, et c est l erreur
// la plus probable d un branchement qui n aurait telecharge que la replication. Le pont ne peut
// pas la prevenir (le manifeste peut legitimement ne pas en declarer), mais il compte les types
// pour que le diagnostic soit lisible cote appelant.
func FilmForMatch(
	ctx context.Context, client filmChunkFetcher, matchID string,
) (*filmsource.Film, bool, error) {
	chunks, found, err := FilmChunksForMatch(ctx, client, matchID)
	if err != nil || !found {
		return nil, found, err
	}
	film, err := FilmOf(chunks)
	if err != nil {
		return nil, true, fmt.Errorf("chargement du film %s: %w", matchID, err)
	}
	return film, true, nil
}

// FilmChunksForMatch : les chunks TYPES d un film, en une seule lecture de manifeste.
//
// C est le point d entree du collecteur, parce que lui a besoin du TYPE : les morts se
// decodent sur la sequence complete, les tirs sur la REPLICATION_DATA seule. Rendre
// un `*filmsource.Film` ici perdrait cette information — d ou deux fonctions et pas une.
func FilmChunksForMatch(
	ctx context.Context, client filmChunkFetcher, matchID string,
) ([]haloclient.FilmChunk, bool, error) {
	chunks, found, err := client.GetFilmChunks(ctx, matchID)
	if err != nil {
		return nil, found, fmt.Errorf("chunks du film %s: %w", matchID, err)
	}
	if !found || len(chunks) == 0 {
		return nil, false, nil
	}
	return chunks, true, nil
}

// FilmOf CHARGE le film a partir de chunks DEJA telecharges : decompression et decoupage en
// paquets, une fois pour toutes (`filmsource`, lot 1 de PLAN_CUISSON_PERF).
//
// Elle existe separement parce qu une passe de collecte lit le film PLUSIEURS FOIS — les morts
// (`killsource`), les tirs (`analysis.ScanFireEventsB5`) et les positions (quatre balayages) — et
// que telecharger plusieurs fois serait payer plusieurs fois le seul cout reseau du chantier.
// L appelant recupere les chunks TYPES une fois, en derive CE film pour les morts et les
// positions, et selectionne lui-meme la REPLICATION_DATA pour les tirs (qui scanne des octets
// bruts, pas des paquets).
//
// L INDEX DU MANIFESTE EST LA POSITION DANS LE FILM, et les deux coincident ici parce que la
// sequence part du chunk 0 : `film.Chunk(0)` est donc l en-tete, celui que `killsource` lit comme
// registre ECS. Les index ne sont pas garantis contigus : on dimensionne sur le maximum observe
// plutot que sur le nombre d entrees, et on laisse les trous VIDES (un chunk vide ne rend aucun
// paquet). Les metadonnees rendues sont POSITIONNELLES, comme `filmsource.Load` les attend, et
// portent le type et le debut que le manifeste donne a chaque chunk present.
func FilmOf(chunks []haloclient.FilmChunk) (*filmsource.Film, error) {
	maxIdx := 0
	for _, c := range chunks {
		if c.Index > maxIdx {
			maxIdx = c.Index
		}
	}
	seq := make([][]byte, maxIdx+1)
	meta := make([]filmsource.ChunkMeta, maxIdx+1)
	for i := range meta {
		meta[i] = filmsource.ChunkMeta{Index: i}
	}
	for _, c := range chunks {
		if c.Index >= 0 {
			seq[c.Index] = c.Data
			meta[c.Index] = filmsource.ChunkMeta{Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS}
		}
	}
	return filmsource.Load(filmsource.MemoryChunks(seq), meta)
}

// ReplicationChunks : les seuls chunks que la passe de TIRS a le droit de scanner.
//
// LA POPULATION EST LA MESURE. La loi « un fire-event = un tir » (mediane 0,994 sur 3 129
// observations de mode standard) a ete etablie sur la REPLICATION_DATA seule — c est ce que
// rendait `GetMatchFilm`. Scanner en plus l en-tete et le kill-feed ajouterait des evenements
// hors de cette population : la porte de publication (+-10 % du `shots_fired` de l API) les
// verrait comme de la SUR-attribution, sans rien qui dise pourquoi.
func ReplicationChunks(chunks []haloclient.FilmChunk) [][]byte {
	out := make([][]byte, 0, len(chunks))
	for _, c := range chunks {
		if c.ChunkType == haloclient.FilmChunkTypeReplicationData && len(c.Data) > 0 {
			out = append(out, c.Data)
		}
	}
	return out
}

// CountFilmChunkTypes compte les chunks par type de manifeste. Sert le diagnostic du collecteur :
// « 29 chunks de replication et AUCUN kill-feed » est une phrase qui explique un `ErrNoKillFeed`,
// la ou l erreur seule laisse chercher.
func CountFilmChunkTypes(chunks []haloclient.FilmChunk) map[int]int {
	out := make(map[int]int, 3)
	for _, c := range chunks {
		out[c.ChunkType]++
	}
	return out
}
