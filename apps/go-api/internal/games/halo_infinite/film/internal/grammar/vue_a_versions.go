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
// # LA REGLE, EN DEUX CLASSES (decisions de l utilisateur du 2026-10-04 ; garde de version majeure,
// decision du pilote du 2026-10-06)
//
// La grammaire de l executable est celle de SES versions : un ecrivain d une autre version d un genre
// a pu ecrire autre chose, et le lecteur du jeu ne sait pas le lire (les branches qu il porte
// s arretent a sa propre version). Deux tables lues se comparent, sans seuil ni build nomme, sous la
// version majeure que le jeu joue (ci-dessous) :
//
//	EGALE     le film declare les [GenresVueA] genres, chacun a sa version native, ET sa version
//	          majeure est celle que le jeu joue ([versionMajeureJouee]) : l ecrivain du film est le
//	          lecteur porte. Film RECENT : la fin de sa vue A est le debut de sa vue B.
//	PREFIXE   le film declare MOINS de genres, chacun a sa version native — un ecrivain plus ancien,
//	          que la table ne suffit pas a identifier au lecteur porte (le tir a composantes des
//	          films a 121 genres en differe, mesure) —, OU sa table est egale sous une autre version
//	          majeure. La fin de sa vue A ne vaut que prouvee paquet par paquet.
//	ILLISIBLE section d identification absente, table vide ou plus longue, une version differente :
//	          la vue A n est pas lue au-dela de sa tete.
//
// Les deux classes lisent la vue A jusqu a son terminateur ([lireLaVueA]), et sa fin est rangee avec
// elle (`lecture.Paquet.VueA`). Ce que la marche fait de cette fin depend de la classe
// ([debutParLaVueA]) : le debut de la vue B. Un genre au-dela du cardinal du film n existe pas chez son
// ecrivain : le lire arrete la vue A.
//
// # LA GARDE DE VERSION MAJEURE : LE JEU NE JOUE QUE LA SIENNE (lu ; decision du pilote, 2026-10-06)
//
// Avant de jouer un film, le jeu lit sa version majeure, le u32 de tete de `chunk_00`
// ([FilmMajorVersion]) : `FUN_1428e219c` (appele par `FUN_140ba23e4` @1423b9bd7) ne lit le film, et
// ne reporte son bit de controle de corruption (`singleton + 0x1AE` <- `film + 0xCB45C`), que sous
// `*film == 0x29` ; sinon il passe a `FUN_142988e98` (releve sous `va_ghidra/FUN_1428e219c.c`).
// L executable lu (HI_1_13_0) ne joue donc que les films de majeure 0x29 : pour un film d une autre
// majeure, « l ecrivain du film est le lecteur porte » n est pas lu dans le jeu, meme quand sa table
// des genres est egale a la native. C est le cas des films HI_1_12_0 (majeure 0x28, table de 123
// genres aux versions natives ; 147 films du cache, mesure du 2026-10-06,
// `va_ghidra/sonde_majeures_parc_2026-10-06.txt`). Ils suivent la regle des films PREFIXE : E
// seulement si la marche depuis E ferme le paquet sans regle de l ecrivain contredite, sinon le chemin
// d avant ([tableDesGenresDuFilm], [classeSousLaMajeure]). Une version majeure non lue n est pas la
// version jouee.
//
// # LA NUMEROTATION DES GENRES, PRESUMEE POUR LES FILMS PREFIXE
//
// Comparer les versions genre par genre suppose que le genre i du film est le message i de
// l executable : des genres ajoutes en fin de table, aucun insere. Ce n est pas lu (seul l executable
// HI_1_13_0 est lu, `FUN_140e453b4` y enregistre ses descripteurs). La colonne des versions vaut 1 sur
// 104 genres sur 123, et la derniere version differente de 1 est au genre 107
// ([dernierGenreAVersionDistinctive]) : une insertion avant lui deplacerait sa version et rendrait la
// table illisible, une insertion entre 108 et 120 decalerait les genres d un film PREFIXE sans changer
// sa classe. Pour un film PREFIXE, les genres 108 a `genres - 1` sont donc lus sous les charges
// natives sur cette seule presomption, et seule la preuve de fermeture ([debutParLaVueA]) protege
// l usage de E. Aucun canal de production ne lit un genre de vue A au-dela de la tete (lu : [teteDe]
// prend le premier, [listeAnnoncee] le compte) ; la suite des genres rangee dans `lecture.Paquet.VueA`
// dit ou la presomption commence (`lecture.VueA.PremierPresume`, [premierGenrePresume]).

// classeDeLaVueA est la classe de la table des genres que le film declare.
type classeDeLaVueA uint8

// Les classes de la table des genres d un film.
const (
	// vueAIllisible : la vue A du film ne se lit pas au-dela de sa tete.
	vueAIllisible classeDeLaVueA = iota
	// vueAPrefixe : la fin de la vue A ne vaut que prouvee paquet par paquet — la table du film est
	// un prefixe STRICT de la table native, ou elle lui est egale sous une version majeure que le jeu
	// ne joue pas.
	vueAPrefixe
	// vueAEgale : la table du film est EGALE a la table native, sous la version majeure que le jeu
	// joue.
	vueAEgale
)

// versionMajeureJouee est la seule version majeure de film que l executable lu joue : `FUN_1428e219c`
// ne lit le film que sous `*film == 0x29` (cf. l en-tete, « la garde de version majeure »).
const versionMajeureJouee = 0x29

// dernierGenreAVersionDistinctive est le plus haut genre dont la version native differe de 1
// ([versionsNativesDesGenres]) : au-dela, la numerotation des genres d un film PREFIXE est presumee
// (cf. l en-tete).
const dernierGenreAVersionDistinctive = 107

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
	// variante : ce que la variante de partie du film decide des genres 85 et 116 ([varianteDuFilm]).
	variante varianteDeLaVueA
}

// grammaireDeLaVueASousFilm derive d un profil la grammaire de la vue A que le film declare : sa table
// des genres et la simulation de son enregistreur, lues dans la section d identification de
// `chunk_00`, sa variante de partie, lue dans le corps de `chunk_00`, et les tables de la region
// jouee de l entree de catalogue du profil.
func grammaireDeLaVueASousFilm(p profile.Profile) grammaireDeLaVueA {
	g := grammaireDeLaVueA{script: scriptDuFilm(p), positions: tablesDeLaRegionJouee(p.Map()),
		variante: varianteDuFilm(p)}
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

// tableDesGenresDuFilm applique la regle au profil d un film : la classe de sa table des genres sous
// sa version majeure, lue en tete de `chunk_00` ([profile.HighlightProfile] porte la lecture), et son
// cardinal (0 pour un film illisible). C est le seul site ou la classe d un film est decidee.
func tableDesGenresDuFilm(p profile.Profile) (classeDeLaVueA, int) {
	if !p.IdentityRead() {
		return vueAIllisible, 0
	}
	classe, genres := classeDesGenres(p.Identity().TypeVersions)
	majeure := p.Highlight()
	return classeSousLaMajeure(classe, majeure.MajorVersion, majeure.Lue), genres
}

// classeSousLaMajeure applique a la classe de la table la garde de version majeure du jeu
// (`FUN_1428e219c`, cf. l en-tete) : une table EGALE ne vaut lecteur porte que sous la version
// majeure jouee, LUE ; sinon la fin de la vue A du film ne vaut que prouvee, comme celle d un film
// PREFIXE (decision du pilote du 2026-10-06).
func classeSousLaMajeure(classe classeDeLaVueA, majeure int, lue bool) classeDeLaVueA {
	if classe == vueAEgale && (!lue || majeure != versionMajeureJouee) {
		return vueAPrefixe
	}
	return classe
}

// premierGenrePresume rend le rang, dans la suite `genres` lue dans la vue A d un film de classe
// `classe`, du premier genre dont la numerotation est presumee et non lue : le premier genre au-dela
// de [dernierGenreAVersionDistinctive] d un film PREFIXE (cf. l en-tete) : le message de
// l executable qu il est est presume, et la position des genres qui le suivent depend de sa charge,
// lue sous la presomption. `len(genres)` quand aucun ne l est.
func premierGenrePresume(classe classeDeLaVueA, genres []int) int {
	if classe == vueAPrefixe {
		for i, g := range genres {
			if g > dernierGenreAVersionDistinctive {
				return i
			}
		}
	}
	return len(genres)
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
