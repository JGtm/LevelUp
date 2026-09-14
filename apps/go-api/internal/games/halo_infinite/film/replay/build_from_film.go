package replay

// build_from_film.go — LE DECODAGE D'UN FILM, ET RIEN QUE LUI.
//
// DEPLACEMENT PUR depuis `build.go` (lot 1 de PLAN_CUISSON_PERF, 2026-09-02), et c'est la
// resorption que le plan avait prevue (note N-G du §8). `build.go` melangeait deux travaux de
// nature differente : DECODER un film (~290 lignes de balayages sequentiels, un par calque) et
// ASSEMBLER un document a partir de positions deja decodees (`BuildFromPositions`, PUR et
// testable sans fichier). Le fichier pesait 928 lignes ; la separation rend chacun lisible seul.
//
// LE SECOND DECOUPAGE (lot 1.0 du PLAN_DECODEUR_FILM, 2026-09-14) : la SEQUENCE de balayages
// n'est plus dans le corps de `BuildFromFilm` mais dans `scanFilmInputs`, qui rend un
// [FilmInputs] — le type de ce que l'assemblage consomme (cf. film_inputs.go). `BuildFromFilm`
// est desormais exactement « cet etage, puis `BuildFromPositions` ». CE QUE CELA FERME : le
// fixture d'entrees (`golden_inputs_film_test.go`) RECOPIAIT la sequence a la main, et les deux
// copies avaient diverge de cinq canaux entiers (decouverte D7 du lot 0.D). Il appelle
// desormais le meme etage — il n'y a plus de sequence a recopier.
//
// C'EST ICI QUE LE FILM EST CONSOMME, ET NULLE PART AILLEURS DANS `replay` : les balayages
// recoivent un `*filmsource.Film` deja charge (une seule decompression par cuisson) et les
// enveloppes `dir` sont interdites en production (garde-rail
// `internal/archlint/no_film_reread_test.go`).
//
// GARDE-RAILS QUI LISENT CE FICHIER : `observe_test.go` (la liste fermee des etapes observees,
// dans l'ordre du source, en descendant de `scanFilmInputs` dans les phases de `film_scan.go`) et
// `world_object_precision_guard_test.go` (l'installation des largeurs d'axe sous le verrou, dans
// `BuildFromFilm`). Les deux le parsent PAR SON NOM — s'il demenage, ils le suivent.

import (
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// BuildFromFilm décode les positions bipeds des SEULS chunks du film DEJA CHARGE et en
// assemble le document de rejeu 2D. Aucune entrée Cheat Engine.
//
// LE FILM EST UN PARAMETRE DEPUIS LE LOT 1 (2026-09-02, PLAN_CUISSON_PERF item 1.2), et c'est
// tout le gain : les ~27 balayages de `scanFilmInputs` relisaient et redecompressaient le film
// ENTIER chacun leur tour. Ils consomment desormais le meme `*filmsource.Film`, charge UNE fois
// par l'appelant (`replaybuild.BuildBytes`). Aucun d'eux ne touche plus le disque.
//
// HORS LIGNE par construction — ne jamais appeler depuis un chemin de requête ; l'API sert
// l'artefact pré-construit.
func BuildFromFilm(matchID, titleSlug string, film *filmsource.Film, opt Options) (ReplayDocument, error) {
	if opt.MapQuant == nil {
		return ReplayDocument{}, fmt.Errorf("%w (match %s) : le document de rejeu exige l'entrée de catalogue de la carte",
			filmdec.ErrUnknownMapBounds, matchID)
	}
	// UN SEUL decodage filmdec a la fois par process (verrou de paquet partage avec
	// killsource.Decode) : les balayages de `scanFilmInputs` lisent et ecrivent les globaux de
	// filmdec (dont compWidthObs, sans verrou propre). Tenu jusqu'au retour : l'assemblage
	// pur qui suit est negligeable devant le decodage, et relacher plus tot inviterait un
	// entrelacement entre deux sous-balayages du MEME film.
	release := filmdec.LockProcessDecode()
	defer release()
	// Les largeurs d'axe du chemin WORLD-OBJECT sont un global de paquet : installées ici,
	// sous le verrou, pour TOUT le decodage du film, et restaurees au retour.
	defer installWorldObjectPrecision(*opt.MapQuant, matchID)()
	in, err := scanFilmInputs(matchID, film, opt)
	if err != nil {
		return ReplayDocument{}, err
	}
	in.applyTo(&opt)
	return BuildFromPositions(matchID, titleSlug, in.Positions, in.Fire, opt), nil
}

// filmScan porte ce que les cinq phases de balayage se partagent : le film et son contexte, les
// reglages de decodage, les bornes de la carte, les options de l'appelant (l'observateur et les
// gardes de mode) et les entrees accumulees.
//
// UN RECEPTEUR PLUTOT QUE DES PARAMETRES : les phases se passeraient sinon six a huit valeurs
// chacune, au-dela de la limite du depot (5 parametres), et une phase qui en oublierait une
// lirait un zero sans que rien ne le dise.
type filmScan struct {
	matchID string
	film    *filmsource.Film
	fc      *filmdec.FilmContext
	scan    filmdec.ScanFilmOptions
	world   filmdec.Vec3Range
	// opt porte ce que l'APPELANT a fourni : l'observateur, son horloge, et les gardes de mode
	// des trois calques qui ne se balaient que sur demande (drapeau, zones, bombe). Les
	// balayages n'y ECRIVENT jamais — leurs sorties vont dans `in`.
	opt Options
	in  FilmInputs
}

// scanFilmInputs EST L'ETAGE DE BALAYAGE : il lit le film et rend ce que l'assemblage consomme.
//
// PRE-REQUIS : l'appelant detient `filmdec.LockProcessDecode` et a installe les largeurs d'axe
// de la carte (`installWorldObjectPrecision`). Les deux sont des globaux de paquet, et
// `BuildFromFilm` — l'unique appelant de production — les tient pour toute la duree du decodage.
// Le ratchet `archlint/decode_lock_held_test.go` verifie cette couverture par point fixe.
//
// L'ORDRE DES PHASES EST L'ORDRE DU FILM, et il n'est pas libre : les positions exemptent le
// filtre de vitesse aux teleportations, les changements d'arme se qualifient sur les loadouts
// deja lus, les changements d'equipement sur les naissances lues dans les positions, et les
// socles comme les vehicules heritent des largeurs MPP calibrees par les poses.
func scanFilmInputs(matchID string, film *filmsource.Film, opt Options) (FilmInputs, error) {
	s := &filmScan{matchID: matchID, film: film, opt: opt, world: opt.MapQuant.Range()}
	s.scan = filmdec.DefaultScanFilmOptions()
	if opt.Scan != nil {
		s.scan = *opt.Scan
	}
	s.scan.WorldRange = &s.world
	// Le cap de visée (Point.H) se lit dans le MÊME record que la position : la capture des
	// directions est donc toujours active pour l'artefact. Elle n'altère aucune position
	// (lecture seule après le vec3 d'i0).
	s.scan.CaptureDirs = true
	// LE CONTEXTE DU FILM EST OUVERT UNE FOIS, ICI, ET TOUS LES BALAYAGES LE PARTAGENT (lot 2
	// de PLAN_CUISSON_PERF, 2026-09-03). Il porte les trois derivations qui ne dependent que du
	// film — bande de slots bipede, decoupage d'i0, registre chunk_00 — que huit, six et douze
	// balayages recalculaient chacun pour leur compte sur le film pourtant deja charge. Il ne
	// LIT rien a la construction : chaque derivation est calculee au premier balayage qui la
	// demande, donc a l'endroit exact ou elle l'etait avant (cf. filmdec/film_context.go), et
	// les suivants la lisent.
	//
	// LE DÉCOUPAGE d'i0 vient du CATALOGUE, comme les bornes dont il est déduit — le
	// découpage lu dans le film (DetectI0Layout) est le contrôle, jamais l'entrée (doctrine
	// écrite sur WorldObjectPrecision, appliquée au bipède depuis le lot C catalogues
	// 2026-08-27 : sur une carte à plus de 2 régions, l'auto-détection lit l'index de
	// région comme un bit d'axe et le décodeur rejetterait tous les records — Live Fire).
	// Un opt.Scan qui force déjà son Layout (instruments) reste maître ; une entrée sans
	// largeurs (catalogue antérieur au champ) laisse l'auto-détection, comme le chemin
	// world-object laisse son défaut — jamais des largeurs nulles.
	//
	// CETTE REGLE EST DESORMAIS CELLE DU CONTEXTE, ET DONC CELLE DE TOUS LES BALAYAGES (lot 3,
	// 2026-09-03) : elle est écrite une seule fois, dans `filmdec.NewFilmContextForMap`, et les
	// positions la lisent au MEME endroit que les six canaux delta. Avant, les positions seules
	// l'appliquaient et les canaux delta re-detectaient — sur Live Fire, 27 enregistrements
	// d'une AUTRE region de compression passaient leur porte (mesure du 2026-09-03, item 3.2).
	s.fc = filmdec.NewFilmContextForMap(film, opt.MapQuant, s.scan.Layout)
	s.scan.Layout = s.fc.ImposedLayout()
	// L'HORLOGE DES BALAYAGES PART ICI, et pas a l'entree de la fonction : ce qui precede est
	// l'attente du verrou process et la lecture du catalogue, qui ne sont le temps d'aucun
	// balayage. A partir d'ici, chaque `opt.observe` ferme le balayage qu'il annonce
	// (cf. observe.go).
	s.opt.clock = &stepClock{last: time.Now()}
	if err := s.balayerPositions(); err != nil {
		return FilmInputs{}, err
	}
	s.balayerPortage()
	s.balayerCapacites()
	s.balayerMonde()
	s.balayerPont()
	return s.in, nil
}
