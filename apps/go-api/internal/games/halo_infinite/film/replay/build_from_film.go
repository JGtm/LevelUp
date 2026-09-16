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
// recoivent un `*source.Film` deja charge (une seule decompression par cuisson) et les
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

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/profile"
	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// BuildFromFilm décode les positions bipeds des SEULS chunks du film DEJA CHARGE et en
// assemble le document de rejeu 2D. Aucune entrée Cheat Engine.
//
// LE FILM EST UN PARAMETRE DEPUIS LE LOT 1 (2026-09-02, PLAN_CUISSON_PERF item 1.2), et c'est
// tout le gain : les ~27 balayages de `scanFilmInputs` relisaient et redecompressaient le film
// ENTIER chacun leur tour. Ils consomment desormais le meme `*source.Film`, charge UNE fois
// par l'appelant (`replaybuild.BuildBytes`). Aucun d'eux ne touche plus le disque.
//
// HORS LIGNE par construction — ne jamais appeler depuis un chemin de requête ; l'API sert
// l'artefact pré-construit.
func BuildFromFilm(matchID, titleSlug string, film *source.Film, opt Options) (ReplayDocument, error) {
	if opt.MapQuant == nil {
		return ReplayDocument{}, fmt.Errorf("%w (match %s) : le document de rejeu exige l'entrée de catalogue de la carte",
			profile.ErrUnknownMapBounds, matchID)
	}
	// PLUS DE VERROU DE PAQUET ICI (lot 2.3) : `grammar` n'a plus aucune variable de paquet
	// ecrite, et cette cuisson porte son propre etat de decodage — le profil de balayage et
	// l'observation vivent dans le `FilmContext` construit ci-dessous, donc dans CET appel. Deux
	// films peuvent se cuire en parallele dans le meme processus ; le verrou INTER-PROCESSUS
	// `filmproc.AcquireSolo`, qui borne la memoire de la machine, reste a la charge des outils.
	// LE COMPTEUR DE REPLIS NAIT ICI, AVANT LE PREMIER BALAYAGE (lot 1.9.0, D14) : les replis du
	// BALAYAGE (largeurs par defaut, plafond de grenades) et ceux de l'ASSEMBLAGE tombent dans le
	// meme compte, celui de cette cuisson, publie dans `coverage.fallbacks`.
	if opt.Fallbacks == nil {
		opt.Fallbacks = fallback.NouveauCompteur()
	}
	// L AVERTISSEMENT UNIQUE PAR FILM quand la version de format de `chunk_00` est inconnue de
	// la table de profil (lot 1.9.1 ter) : ici, avant tout balayage, pour que la ligne PRECEDE
	// les consequences qu elle explique. Les COMPTEURS, eux, tombent aux deux sites du repli.
	avertirFormatSansProfil(film, matchID)
	// LE CONTEXTE DU FILM EST OUVERT ICI DEPUIS LE LOT 2.1, ET PAS DANS `scanFilmInputs` : c est
	// lui qui porte le PROFIL, resolu a la construction (D1), et c est le profil que
	// `installWorldObjectPrecision` lit juste apres. L ouvrir plus bas obligerait a resoudre le
	// profil DEUX fois par cuisson — une recopie de plus, alors que le lot s appelle « resolu une
	// fois ». Rien d autre ne bouge : les trois derivations memorisees (bande de slots, decoupage
	// d i0, registre) restent PARESSEUSES, donc calculees au premier balayage qui les demande,
	// donc apres l installation ci-dessous et apres le demarrage de l horloge des etapes.
	fc := grammar.NewFilmContextForMap(film, opt.MapQuant, decoupageForce(opt))
	poserProfilPuisCarte(fc, matchID, opt)
	in, err := scanFilmInputs(matchID, film, fc, opt)
	if err != nil {
		return ReplayDocument{}, err
	}
	in.applyTo(&opt)
	return BuildFromPositions(matchID, titleSlug, in.Positions, in.Fire, opt), nil
}

// poserProfilPuisCarte installe sur le contexte, DANS CET ORDRE, le profil de balayage calibre
// par la passe precedente puis les largeurs d'axe de la carte.
//
// # L ORDRE EST LA REGLE, ET IL N EST PAS COMMUTATIF
//
// [grammar.FilmContext.PoserProfilDeBalayage] remplace le profil ENTIER. Poser les largeurs de la
// carte d'abord et le profil ensuite les EFFACERAIT sans un mot : les objets du monde seraient
// dequantifies aux largeurs par defaut — celles d'UNE carte, `cliffhanger` — sur toutes les
// autres. Dans le bon ordre, `installWorldObjectPrecision` n'ecrase que le descripteur
// world-object du profil, et laisse la traversee, la largeur d'axe absolue et le `param_4` que
// `killsource` a calibres.
//
// # D OU VIENT LE PROFIL
//
// Des OPTIONS (lot 2.3, condition D1 du lot 2.2.a). `killsource` calibre sur CE film un
// descripteur de traversee, une largeur d'axe absolue et un `param_4` dont la cuisson heritait
// jusqu'ici par l'etat du processus ; `replaybuild` les passe desormais explicitement. Options
// sans profil (appelant qui n'a pas decode le kill-feed) : le contexte garde le sien.
//
// Garde-rail : `TestRouteDuProfilCalibreJusquAuContexte` epingle les deux — le profil calibre
// ARRIVE, et les largeurs de carte SURVIVENT. Intervertir les deux appels le fait rougir.
func poserProfilPuisCarte(fc *grammar.FilmContext, matchID string, opt Options) {
	if opt.ProfilDeBalayage != nil {
		fc.PoserProfilDeBalayage(*opt.ProfilDeBalayage)
	}
	installWorldObjectPrecision(fc, matchID, opt.Fallbacks)
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
	film    *source.Film
	fc      *grammar.FilmContext
	scan    grammar.ScanFilmOptions
	world   profile.Vec3Range
	// opt porte ce que l'APPELANT a fourni : l'observateur, son horloge, et les gardes de mode
	// des trois calques qui ne se balaient que sur demande (drapeau, zones, bombe). Les
	// balayages n'y ECRIVENT jamais — leurs sorties vont dans `in`.
	opt Options
	in  FilmInputs
}

// decoupageForce rend le decoupage d'i0 que l'APPELANT impose, ou nil.
//
// C'est la seule partie de `grammar.DefaultScanFilmOptions()` / `*opt.Scan` dont le CONSTRUCTEUR
// du contexte a besoin (`NewFilmContextForMap` la prend en troisieme parametre). Le defaut du
// paquet ne force aucun decoupage — c'est le sens de la regle du catalogue —, donc l'absence
// d'`opt.Scan` rend nil, exactement comme `DefaultScanFilmOptions().Layout`.
//
// GARDE-RAIL : `TestDecoupageForceSuitLesOptions` compare cette fonction au champ que
// `scanFilmInputs` calcule pour son propre compte ; les deux ne peuvent pas diverger en silence.
func decoupageForce(opt Options) *profile.I0Layout {
	if opt.Scan == nil {
		return grammar.DefaultScanFilmOptions().Layout
	}
	return opt.Scan.Layout
}

// scanFilmInputs EST L'ETAGE DE BALAYAGE : il lit le film et rend ce que l'assemblage consomme.
//
// PRE-REQUIS : l'appelant a pose sur le CONTEXTE DU FILM le profil de balayage calibre puis
// les largeurs d'axe de la carte (`installWorldObjectPrecision`), DANS CET ORDRE — poser le
// profil ECRASE les largeurs, cf. `BuildFromFilm`. Ce ne sont plus des globaux de paquet depuis
// le lot 2.3 : ils voyagent avec le contexte, donc avec cet appel.
//
// L'ORDRE DES PHASES EST L'ORDRE DU FILM, et il n'est pas libre : les positions exemptent le
// filtre de vitesse aux teleportations, les changements d'arme se qualifient sur les loadouts
// deja lus, les changements d'equipement sur les naissances lues dans les positions, et les
// socles comme les vehicules heritent des largeurs MPP calibrees par les poses.
func scanFilmInputs(matchID string, film *source.Film, fc *grammar.FilmContext,
	opt Options) (FilmInputs, error) {
	s := &filmScan{matchID: matchID, film: film, fc: fc, opt: opt, world: opt.MapQuant.Range()}
	s.scan = grammar.DefaultScanFilmOptions()
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
	// 2026-09-03) : elle est écrite une seule fois, dans `grammar.NewFilmContextForMap`, et les
	// positions la lisent au MEME endroit que les six canaux delta. Avant, les positions seules
	// l'appliquaient et les canaux delta re-detectaient — sur Live Fire, 27 enregistrements
	// d'une AUTRE region de compression passaient leur porte (mesure du 2026-09-03, item 3.2).
	// LE CONTEXTE VIENT DE `BuildFromFilm` DEPUIS LE LOT 2.1 (cf. son commentaire) : il y est
	// ouvert avec le MEME decoupage force (`decoupageForce`, qui lit `opt.Scan.Layout` comme les
	// deux lignes ci-dessus) et la meme entree de carte. Ce qui suit est inchange : la regle du
	// catalogue est tranchee dans le constructeur, et les balayages la lisent ici.
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
