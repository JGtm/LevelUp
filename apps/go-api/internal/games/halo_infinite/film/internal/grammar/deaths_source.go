package grammar

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"levelup/go-api/internal/domain/highlightevent"

	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
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
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM, DU-3 = S1) :
// c est une LECTURE du film, et l ADR 0034 D-1 dit que la couche de publication ne decode rien.
// Deplacement pur ; `Death` vit en `types.Death`. `replay` recoit les morts par
// `Options.Deaths`, comme il recoit deja les loadouts et les grenades. Ce fichier fait entrer
// `film/filmcache` (le predicat unique du type des temps forts, `finalise.EstTempsForts`) dans
// le perimetre de la couche : c est la seule facon de choisir le morceau PAR SON TYPE sans
// recopier le predicat (ratchet `archlint/film_finalise_predicate_test.go`).
//
// CE QU'ON NE FAIT PAS ICI, et c'est délibéré : on ne recopie pas le parseur. Il vit dans
// `ParseHighlightEvents`, il est testé là-bas, et une seconde implémentation
// divergerait — la règle du dépôt sur les copies vaut aussi pour les décodeurs.

// ErrFilSansTempsForts : les morceaux du film sont TYPES par son manifeste, et aucun n est celui des
// temps forts — la signature d un film archive avant sa finalisation (`ab526724`, 2026-09-22 : le
// dernier numero etait un morceau de replication), ou d un morceau des temps forts absent du
// cache. DISTINCTE d un morceau illisible ou sans mort, et de la famille
// [finalise.ErrFilmNonFinalise] : `errors.Is` repond vrai pour les deux.
var ErrFilSansTempsForts = fmt.Errorf("fil des morts : aucun morceau des temps forts parmi les "+
	"morceaux types du film : %w", finalise.ErrFilmNonFinalise)

// ErrFilDesMortsSansMort : le morceau des temps forts a ete LU et ne porte aucune mort. C est une
// MESURE, pas une panne — `coverage.bridge.deathsFeed` la publie `empty`, distincte de
// `unreadable` (lot M5.2 des retours rejeu, 2026-09-23). Le texte du message est celui d avant.
var ErrFilDesMortsSansMort = errors.New("aucune mort")

// ScanFilmDeaths lit le fil des morts du film de filmDir.
//
// HORS LIGNE (I/O disque) — jamais depuis un chemin de requête ; l'API sert l'artefact
// pré-construit.
//
// ENVELOPPE D2, HORS PRODUCTION (lot 1, 2026-09-02) : la cuisson appelle [ScanDeaths] sur un
// film déjà chargé.
func ScanFilmDeaths(filmDir string) ([]types.Death, error) {
	film, err := source.LoadDir(filmDir, nil)
	if err != nil {
		return nil, err
	}
	return ScanDeaths(film)
}

// numeroDesTempsForts rend le NUMERO du morceau des temps forts.
//
// MANIFESTE PRESENT (au moins un morceau type) : le morceau dont le type est celui des temps forts
// ([finalise.EstTempsForts]) — le dernier s il y en avait plusieurs, ce que le parc ne montre
// pas. Aucun : [ErrFilSansTempsForts], le film n est pas finalise.
//
// MANIFESTE ABSENT (film charge sans metadonnees — enveloppes D2, instruments, tests —, ou
// repertoire sans manifeste, 0 au parc du 2026-09-23) : le DERNIER numero, la regle d avant le
// lot, et c est un REPLI NOMME, `repli_temps_forts_dernier_numero` (registre `facts/fallback`,
// critere de retrait ecrit la-bas). En cuisson il n est atteint que par un repertoire VRAIMENT
// sans manifeste, que `replaybuild.jugerFilmSansManifeste` laisse passer en le journalisant ; un
// manifeste present mais illisible y est refuse (constat L3-R8 de la revue adverse du lot).
func numeroDesTempsForts(film *source.Film, nums []int) (int, error) {
	n, type3, typee := -1, false, false
	for _, m := range film.Meta() {
		if m.ChunkType != 0 {
			typee = true
		}
		if finalise.EstTempsForts(m.ChunkType) {
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
// LES OCTETS SONT DEJA DECOMPRESSES, et `ParseHighlightEvents` l'accepte : il tente un
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
func ScanDeaths(film *source.Film) ([]types.Death, error) {
	nums := FilmChunkNumbers(film)
	if len(nums) == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	n, err := numeroDesTempsForts(film, nums)
	if err != nil {
		return nil, err
	}
	raw, _, ok := FilmChunkAt(film, n)
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
	evs, err := ParseHighlightEvents(raw, HighlightProfileOfFilm(film).MajorVersion)
	if err != nil {
		return nil, fmt.Errorf("chunk highlight (%d) : %w", n, err)
	}
	out := make([]types.Death, 0, len(evs))
	for _, e := range evs {
		if e.EventType != highlightevent.EventTypeDeath {
			continue
		}
		out = append(out, types.Death{XUID: e.XUID, Gamertag: e.Gamertag, TimeMS: int64(e.TimeMS)})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("chunk highlight (%d) : %w", n, ErrFilDesMortsSansMort)
	}
	trierMortsDuFil(out)
	return out, nil
}

// trierMortsDuFil range le fil des morts dans un ordre TOTAL (lot J10.1, 2026-09-27, DT-9) :
// instant, puis xuid, puis gamertag. Deux morts de la meme milliseconde (un double a la grenade)
// ne se departageaient pas : leur rang, qui ordonne le journal publie et les appariements aux
// vies, dependait du tri. Deux morts que ce comparateur ne separe pas sont identiques.
func trierMortsDuFil(out []types.Death) {
	slices.SortFunc(out, func(a, b types.Death) int {
		return cmp.Or(cmp.Compare(a.TimeMS, b.TimeMS), cmp.Compare(a.XUID, b.XUID),
			strings.Compare(a.Gamertag, b.Gamertag))
	})
}
