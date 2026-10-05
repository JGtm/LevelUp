package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// vue_a_versions.go — LA VERSION DE CHAQUE GENRE DE MESSAGE : CELLE DU JEU, CELLE DU FILM, ET LA
// REGLE QUI DIT SI LA GRAMMAIRE DU JEU LIT LA VUE A D UN FILM (lot LN, regle en deux classes du lot
// VA de la campagne de grammaire).
//
// # CE QUE LE JEU LIT
//
// Le lecteur d un message consulte la version de SON genre par `FUN_141102ed0(genre)` : en rejeu
// de film (`FUN_1404f2b4c`), `FUN_1428e1c64` rend le u32 `film + 0xCB208 + genre * 4` — la table
// par genre de `chunk_00` ([profile.FilmIdentity.TypeVersions]) ; hors film, la table NATIVE
// `DAT_14474cd90` (paires {version, taille de la structure du message}, la taille etant nulle
// exactement pour les treize genres vides de [tableDesGenresVueA]). Dix lecteurs la consultent (35,
// 36, 40, 48, 89, 90, 91, 93, 97, 114), et seulement pour separer une version ancienne de la leur.
//
// # LA REGLE, EN DEUX CLASSES (decision de l utilisateur du 2026-10-04)
//
// La grammaire de l executable est celle de SES versions : un ecrivain d une autre version d un genre
// a pu ecrire autre chose, et le lecteur du jeu ne sait pas le lire (les branches qu il porte
// s arretent a sa propre version). Deux tables lues se comparent, sans seuil ni build nomme :
//
//	EGALE     le film declare les [GenresVueA] genres, chacun a sa version native : l ecrivain du
//	          film est le lecteur porte. Film RECENT : la fin de sa vue A est le debut de sa vue B.
//	PREFIXE   le film declare MOINS de genres, chacun a sa version native : un ecrivain plus ancien,
//	          que la table ne suffit pas a identifier au lecteur porte (le tir a composantes des
//	          films a 121 genres en differe, mesure). Film ANCIEN : la fin de sa vue A ne vaut que
//	          prouvee paquet par paquet.
//	ILLISIBLE section d identification absente, table vide ou plus longue, une version differente :
//	          la vue A n est pas lue au-dela de sa tete.
//
// Les deux classes lisent la vue A jusqu a son terminateur ; ce que la marche fait de sa fin depend
// de la classe ([debutParLaVueA]). Un genre au-dela du cardinal du film n existe pas
// chez son ecrivain : le lire arrete la vue A.

// classeDeLaVueA est la classe de la table des genres que le film declare.
type classeDeLaVueA uint8

// Les classes de la table des genres d un film.
const (
	// vueAIllisible : la vue A du film ne se lit pas au-dela de sa tete.
	vueAIllisible classeDeLaVueA = iota
	// vueAPrefixe : la table du film est un prefixe STRICT de la table native.
	vueAPrefixe
	// vueAEgale : la table du film est EGALE a la table native.
	vueAEgale
)

// grammaireDeLaVueA est ce que le film declare de la grammaire de sa vue A, et ce que la carte de son
// match en fait connaitre. Elle vient du film ([grammaireDeLaVueASousFilm]) et de nulle part
// ailleurs : ce n est pas un reglage du balayage, et elle ne voyage pas avec le profil de balayage
// ([ProfilDeBalayage]) — seulement avec le lecteur de la vue A ([Lecteur.vueA]).
type grammaireDeLaVueA struct {
	// classe et genres : la classe de la table des genres du film et son cardinal
	// ([classeDesGenres]) ; [vueAIllisible] : la vue A ne se lit que jusqu a sa tete.
	classe classeDeLaVueA
	genres int
	// script : ce que le film declare du prefixe du message Script ([scriptDuFilm]).
	script etatDuScript
	// positions : les tables de position de la region jouee de la carte du match, pour les
	// positions a index des genres 5 et 6 ([tablesDeLaRegionJouee]) ; vides sans carte.
	positions tablesDePosition
}

// grammaireDeLaVueASousFilm derive d un profil la grammaire de la vue A que le film declare : sa table
// des genres et la simulation de son enregistreur, lues dans la section d identification de
// `chunk_00`, et les tables de la region jouee de l entree de catalogue du profil.
func grammaireDeLaVueASousFilm(p profile.Profile) grammaireDeLaVueA {
	g := grammaireDeLaVueA{script: scriptDuFilm(p), positions: tablesDeLaRegionJouee(p.Map())}
	g.classe, g.genres = tableDesGenresDuFilm(p)
	return g
}

// versionsNativesDesGenres est la colonne version de `DAT_14474cd90` (123 paires de 8 octets), un
// chiffre par genre dans l ordre des genres.
const versionsNativesDesGenres = "" +
	"1111111111" + "1111111121" + "1111111111" + "3111143111" + "2111112121" + // 0-49
	"1111112111" + "2611111112" + "1111111111" + "1515111112" + "3312111411" + // 50-99
	"1111111311" + "1111111111" + "111" // 100-122

// versionNative rend la version native d un genre (0 <= genre < GenresVueA).
func versionNative(genre int) uint32 {
	return uint32(versionsNativesDesGenres[genre] - '0')
}

// tableDesGenresDuFilm applique la regle a l identite d un profil : la classe de la table du film et
// son cardinal (0 pour un film illisible).
func tableDesGenresDuFilm(p profile.Profile) (classeDeLaVueA, int) {
	if !p.IdentityRead() {
		return vueAIllisible, 0
	}
	return classeDesGenres(p.Identity().TypeVersions)
}

// classeDesGenres est la regle sur la table que le film declare (`film + 0xCB208`) : sa classe et
// son cardinal.
func classeDesGenres(declarees []uint32) (classeDeLaVueA, int) {
	if len(declarees) == 0 || len(declarees) > GenresVueA {
		return vueAIllisible, 0
	}
	for genre, v := range declarees {
		if v != versionNative(genre) {
			return vueAIllisible, 0
		}
	}
	if len(declarees) == GenresVueA {
		return vueAEgale, GenresVueA
	}
	return vueAPrefixe, len(declarees)
}
