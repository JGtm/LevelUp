package replay

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// deaths_source.go — LE FIL DES MORTS, LU DANS LE FILM.
//
// OÙ IL VIT. Le chunk des « highlight events » (type de manifeste 3, les TEMPS FORTS) : un
// enregistrement par événement de match, dont les morts, chacune portant le XUID de la
// victime et son instant. Il est DANS LE FILM — aucune base n'intervient, ce qui préserve
// la propriété que tout le rejeu tient hors ligne. Il est choisi PAR SON TYPE quand le
// manifeste est là (lot L3, 2026-09-23) : « le dernier numéro » désignait un morceau de
// réplication sur un film archivé avant sa finalisation.
//
// POURQUOI CE FICHIER EST SÉPARÉ. C'est le seul point du paquet qui fait des I/O disque et
// qui dépend du paquet `analysis` (pour son parseur, déjà en production et éprouvé). Le
// reste de `replay` reste pur et testable sans fichier : `BuildFromPositions` reçoit les
// morts par `Options.Deaths`, comme il reçoit déjà les loadouts et les grenades.
//
// CE QU'ON NE FAIT PAS ICI, et c'est délibéré : on ne recopie pas le parseur. Il vit dans
// `grammar.ParseHighlightEvents`, il est testé là-bas, et une seconde implémentation
// divergerait — la règle du dépôt sur les copies vaut aussi pour les décodeurs.

// ErrFilSansTempsForts : les morceaux du film sont TYPES par son manifeste, et aucun n est celui des
// temps forts — la signature d un film archive avant sa finalisation (`ab526724`, 2026-09-22 : le
// dernier numero etait un morceau de replication), ou d un morceau des temps forts absent du
// cache. DISTINCTE d un morceau illisible ou sans mort, et de la famille
// [filmcache.ErrFilmNonFinalise] : `errors.Is` repond vrai pour les deux.
var ErrFilSansTempsForts = fmt.Errorf("fil des morts : aucun morceau des temps forts parmi les "+
	"morceaux types du film : %w", filmcache.ErrFilmNonFinalise)

// ScanFilmDeaths lit le fil des morts du film de filmDir.
//
// HORS LIGNE (I/O disque) — jamais depuis un chemin de requête ; l'API sert l'artefact
// pré-construit.
//
// ENVELOPPE D2, HORS PRODUCTION (lot 1, 2026-09-02) : la cuisson appelle [ScanDeaths] sur un
// film déjà chargé.
func ScanFilmDeaths(filmDir string) ([]Death, error) {
	film, err := source.LoadDir(filmDir, nil)
	if err != nil {
		return nil, err
	}
	return ScanDeaths(film)
}

// numeroDesTempsForts rend le NUMERO du morceau des temps forts.
//
// MANIFESTE PRESENT (au moins un morceau type) : le morceau dont le type est celui des temps forts
// ([filmcache.EstTempsForts]) — le dernier s il y en avait plusieurs, ce que le parc ne montre
// pas. Aucun : [ErrFilSansTempsForts], le film n est pas finalise.
//
// MANIFESTE ABSENT (film charge sans metadonnees — enveloppes D2, instruments, tests —, ou
// repertoire sans manifeste, 0 au parc du 2026-09-23) : le DERNIER numero, la regle d avant le
// lot. Ce n est pas un repli de publication : la cuisson ne charge jamais un film sans manifeste
// qu elle n ait deja juge (`replaybuild.refuserManifesteNonFinalise`).
func numeroDesTempsForts(film *source.Film, nums []int) (int, error) {
	n, type3, typee := -1, false, false
	for _, m := range film.Meta() {
		if m.ChunkType != 0 {
			typee = true
		}
		if filmcache.EstTempsForts(m.ChunkType) {
			n, type3 = m.Index, true
		}
	}
	switch {
	case type3:
		return n, nil
	case typee:
		return 0, ErrFilSansTempsForts
	default:
		return nums[len(nums)-1], nil
	}
}

// ScanDeaths lit le fil des morts d'un film DEJA CHARGE.
//
// LES OCTETS SONT DEJA DECOMPRESSES, et `grammar.ParseHighlightEvents` l'accepte : il tente un
// `zlib.NewReader` et, s'il echoue, traite l'entree comme du clair — c'est la double tolerance
// qu'il porte depuis l'incident du 2026-05-22 (le cache historique stockait les chunks
// compresses, les telechargements recents ne le font plus). Lui donner le chunk deja inflate
// rend donc EXACTEMENT les memes evenements, sans une seconde decompression du plus gros chunk
// du film.
//
// LA VERSION DU FILM EST LUE DANS SON REGISTRE (2026-09-12), plus passee a 0 en dur : elle
// commande le decoupage du gamertag du bloc d event, decale de douze octets sur les versions
// 39-40 (mars a novembre 2025). Les `Death.Gamertag` publies dans l artefact de rejeu en
// dependent — cf. .ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md. Film sans registre : version 0,
// decoupage historique, et c est L APPELANT qui consigne la degradation (voir le corps).
func ScanDeaths(film *source.Film) ([]Death, error) {
	nums := grammar.FilmChunkNumbers(film)
	if len(nums) == 0 {
		return nil, grammar.ErrNoReadableFilmChunk
	}
	n, err := numeroDesTempsForts(film, nums)
	if err != nil {
		return nil, err
	}
	raw, _, ok := grammar.FilmChunkAt(film, n)
	if !ok {
		return nil, fmt.Errorf("chunk highlight (%d) : absent du film", n)
	}
	// LE WARN DU REGISTRE ABSENT MONTE CHEZ L APPELANT, ET C EST DELIBERE (revue adversariale du
	// 2026-09-12, constat P2-4). Deux raisons : cette fonction ne connait pas le `match_id` — elle
	// recoit un film deja charge — alors que tous les WARN voisins de la cuisson le portent ; et
	// elle est appelee DEUX FOIS par cuisson (`replaybuild.lireMorts` puis `BuildFromFilm`), ce
	// qui doublait la ligne de journal pour un seul fait. [BuildFromFilm] lit deja cette meme
	// version pour la publier dans la couverture : c est lui qui consigne, une fois, avec le match.
	// LA VERSION VIENT DU PROFIL DU FILM DEPUIS LE LOT 2.1.4 : `HighlightProfileOfFilm` lit le
	// MEME u32 que `FilmMajorVersion` et NOMME l implantation qu il selectionne. La porte etroite
	// plutot que `ResolveProfile` : cette fonction est appelee deux fois par cuisson et n a pas
	// de carte — lui faire resoudre le profil entier couterait une analyse de registre par appel
	// pour une valeur qui tient dans les quatre premiers octets.
	evs, err := grammar.ParseHighlightEvents(raw, grammar.HighlightProfileOfFilm(film).MajorVersion)
	if err != nil {
		return nil, fmt.Errorf("chunk highlight (%d) : %w", n, err)
	}
	out := make([]Death, 0, len(evs))
	for _, e := range evs {
		if e.EventType != highlightevent.EventTypeDeath {
			continue
		}
		out = append(out, Death{XUID: e.XUID, Gamertag: e.Gamertag, TimeMS: int64(e.TimeMS)})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("chunk highlight (%d) : aucune mort", n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out, nil
}
