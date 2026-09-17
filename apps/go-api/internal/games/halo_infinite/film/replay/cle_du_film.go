package replay

// cle_du_film.go — LA PORTE UNIQUE DE LA POLITIQUE « CLE INCONNUE = FILM MIS DE COTE »
// (lot 3.1.1, D-4 d ADR 0034, decision V20 (2) de l utilisateur du 2026-09-17).
//
// # CE QUE CE FICHIER FERME
//
// D-4 dit qu un film dont la cle est absente de la table de profil est MIS DE COTE, jamais lu au
// profil d un voisin. La seconde moitie tenait depuis le lot 1.5 ; la premiere, non : mesure du
// 2026-09-17 a la cloture de M2, AUCUN appelant de production ne lisait
// [FilmContext.ProfileErr] — la seule occurrence hors tests etait sa propre declaration
// (decouverte D1 (cloture M2)). La cuisson continuait donc sur le profil des INVARIANTS.
//
// Ce fichier rend le verdict ; les deux orchestrateurs — `sync/killcollector` et `replaybuild`,
// les seuls que D-4 autorise a journaliser et a publier des compteurs — l appliquent.
//
// # POURQUOI UNE PORTE PARTAGEE, ET PAS DEUX GARDES RECOPIEES
//
// Deux orchestrateurs qui composeraient chacun leur verdict divergeraient au premier build
// ajoute : l un ecarterait un film que l autre cuirait, et le parc porterait des artefacts sans
// faits killsource (ou l inverse) sans que rien ne le dise. Le verdict se compose ICI, une fois.
//
// # POURQUOI LE VERDICT N EST PAS `Profile.Err() != nil`
//
// MESURE DU 2026-09-17. `profile.Resoudre` rend [profile.ErrUnknownBuild] enveloppe avec un
// build VIDE sur les films dont `chunk_00` ne porte AUCUNE section d identification — cinq films
// du cache, versions majeures 31 et 33. Or le profil les CONNAIT : leur cle est `majeure=31` ou
// `majeure=33`, que la table porte. Mettre de cote sur `Profile.Err()` aurait ecarte ces cinq
// films, dont `50247b26` et `a349fea8`, DEUX TEMOINS du corpus gate : deux pertes mesurees pour
// une politique censee n en produire aucune. Le verdict porte donc sur la CLE ECRITE
// ([profile.CleConnue]), et `profile.Resoudre` n est pas touche — sa valeur d erreur commande
// `coverage.decoder.build`, que ce lot ne doit pas deplacer (V15 (15)).
//
// # CE QUE LA PORTE NE FAIT PAS
//
// Elle ne CHARGE pas le film (R4 du ratchet des couches) : elle le RECOIT, deja charge par
// l orchestrateur. Elle ne journalise pas et ne decide pas d un statut de synchronisation : ces
// deux gestes appartiennent a l orchestrateur, qui seul connait le vocabulaire de son pipeline.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/observability"
)

// CleFilm porte les trois cles qu un film ECRIT et le verdict de la table de profil.
//
// LES TROIS CLES SONT PUBLIEES MEME QUAND LA CLE EST CONNUE : c est ce que l orchestrateur
// journalise le jour du refus, et un journal qui dirait « cle inconnue » sans dire LAQUELLE
// n aiderait personne a ajouter la ligne qui manque (runbook `docs/RUNBOOK_FILM_PROFILES.md`).
type CleFilm struct {
	// Format est la version de format (`chunk_00+4`), ou [grammar.FilmFormatVersionUnknown].
	Format int
	// Build est la chaine en clair de la section 2, ou "" quand le film n en porte pas.
	Build string
	// Majeure est la version majeure (`chunk_00+0`) ; `MajeureLue` dit si elle a ete LUE.
	Majeure    int
	MajeureLue bool
	// Ecrite est la cle SELECTIONNANTE sous la forme du catalogue — `build=HI_1_13_0`,
	// `majeure=31` — ou "" quand le film n en ecrit aucune de lisible.
	Ecrite string
	// Lisible dit si le film porte son `chunk_00` ET une cle nommable. Faux = bobine partielle,
	// fixture sans registre, `chunk_00` encore compresse : il n y a RIEN a refuser, et chaque
	// balayage rend deja son propre diagnostic a sa place ([grammar.ErrNoFilmChunk]).
	Lisible bool
	// Err est l erreur TYPEE quand la cle est absente de la table ([profile.ErrUnknownBuild],
	// [profile.ErrUnknownFormat], enveloppees avec la valeur refusee), sinon nil.
	Err error
}

// Refusee dit si ce film doit etre MIS DE COTE. C est la porte que les orchestrateurs testent.
func (c CleFilm) Refusee() bool { return c.Err != nil }

// CleDuFilm rend la cle d un film DEJA CHARGE et le verdict de la table de profil.
//
// UN FILM SANS `chunk_00` N EST JAMAIS REFUSE, et c est la meme regle que
// `grammar.journaliserProfilIncomplet` : une bobine partielle ou une fixture sans registre n a
// pas de cle a chercher, et la refuser transformerait un diagnostic de lecture deja nomme
// ([grammar.ErrNoFilmChunk], « decoupage i0 illisible », ...) en « cle inconnue », qui serait
// faux.
func CleDuFilm(film *source.Film) CleFilm {
	chunk0, ok := grammar.FilmRegistryChunk(film)
	if !ok {
		return CleFilm{Format: grammar.FilmFormatVersionUnknown}
	}
	return CleDeChunk0(chunk0)
}

// CleDeChunk0 rend la cle d un film a partir des octets DECOMPRESSES de son `chunk_00`, et rien
// d autre — aucun flux de bits, aucun autre chunk, aucun decodage.
//
// ELLE EST EXPORTEE POUR LE RECENSEMENT : mesurer combien de films du cache portent une cle
// inconnue ne doit pas coûter un decodage (1 351 films au 2026-09-17). C est aussi la porte que
// la procedure d ajout d un build emploie (`docs/RUNBOOK_FILM_PROFILES.md` §4.5).
func CleDeChunk0(chunk0 []byte) CleFilm {
	format, _ := grammar.FilmFormatVersionFromHeader(chunk0)
	majeure, majeureLue := grammar.FilmMajorVersionFromHeader(chunk0)
	c := CleFilm{Format: format, Majeure: majeure, MajeureLue: majeureLue}
	if id, err := grammar.ReadFilmIdentity(chunk0); err == nil {
		c.Build = id.Build
	}
	switch {
	case c.Build != "", majeureLue:
		c.Ecrite = profile.CleEcrite(c.Build, c.Majeure)
	default:
		// NI BUILD NI MAJEURE : le `chunk_00` est la mais rien de nommable n en sort. Le
		// refuser nommerait une cle qu on n a pas lue ; les lecteurs rendent deja leur erreur.
		return c
	}
	c.Lisible = true
	c.Err = profile.ErreurCleInconnue(c.Format, c.Build, c.Majeure)
	return c
}

// PublierCleInconnue incremente sur `/debug/vars` le compteur de la cle refusee (D-4 : erreur
// TYPEE **et** compteur).
//
// LES DEUX COMPTEURS SONT CEUX QUI EXISTENT DEJA, et c est voulu : `grammar` NOMME
// `filmdec_unknown_build_<build>` ([grammar.UnknownBuildExpvarPairs], cable depuis le lot 1.6 par
// `publierBuildInconnu`) et `filmdec_unknown_format_<n>` ([grammar.UnknownFormatExpvarPairs]).
// En ouvrir un troisieme aurait donne deux comptes du meme evenement, qui divergeraient des que
// l un des deux chemins change.
//
// UN APPEL SUR UNE CLE CONNUE NE COMPTE RIEN : la garde est ici plutot que chez les deux
// appelants, pour qu un orchestrateur qui oublierait de tester ne salisse pas `/debug/vars`.
func PublierCleInconnue(c CleFilm) {
	if !c.Refusee() {
		return
	}
	for _, p := range compteursDeCleInconnue(c) {
		observability.AddInt(p.Name, p.Value)
	}
}

// compteursDeCleInconnue rend les paires a publier pour une cle refusee. Separee de la
// publication pour que le choix des compteurs se teste sans `expvar`, meme patron que
// [grammar.UnknownBuildExpvarPairs].
func compteursDeCleInconnue(c CleFilm) []grammar.ExpvarPair {
	var pairs []grammar.ExpvarPair
	if _, connu := profile.MPPPourFormat(c.Format); !connu {
		pairs = append(pairs, grammar.UnknownFormatExpvarPairs(c.Format)...)
	}
	if !buildOuMajeureConnus(c) {
		// La cle de CONTENU — le build, ou la majeure des films sans section — se compte sous le
		// compteur de build : un build VIDE s y ecrit `sans_section`, et c est exactement la
		// population d une majeure refusee.
		pairs = append(pairs, grammar.UnknownBuildExpvarPairs(c.Build)...)
	}
	return pairs
}

// buildOuMajeureConnus dit si la cle de CONTENU du film est dans la table — le build quand le
// film en ecrit un, la version majeure sinon. Le format n entre pas : il a son propre compteur.
func buildOuMajeureConnus(c CleFilm) bool {
	if c.Build != "" {
		_, connu := profile.PersonnalisationOctets(c.Build)
		return connu
	}
	return profile.MajeureSansSectionConnue(c.Majeure)
}
