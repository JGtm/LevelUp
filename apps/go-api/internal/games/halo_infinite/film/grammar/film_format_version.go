package grammar

// film_format_version.go — LA VERSION DE FORMAT DE `chunk_00` : LA CLE QUE LE FILM PORTE
// LUI-MEME (lot 1.9.1 ter, 2026-09-15).
//
// # LE FAIT QUI FONDE CE FICHIER
//
// Tranche par l utilisateur (mecanique de jeu, il fait autorite) : « le film est autoportant ;
// jamais de profil par build ; la cle est une version ECRITE dans le film ; la grammaire des
// anciens formats est dans l executable courant ». Ce fichier est cette cle, et elle est LUE
// chez l ecrivain — pas deduite du nom du build.
//
// # LE SERIALISEUR A DEUX FACES, ET ELLES NE SONT PAS SYMETRIQUES
//
// Releve du 2026-09-15 (Ghidra HTTP direct, `HaloInfinite.exe`, base `0x140000000`, LECTURE
// SEULE — aucun point d entree d ecriture appele).
//
//	ECRITURE  `FUN_14299b198(film, ecrivain)` — toutes les largeurs sont des LITTERAUX du build
//	          courant : W(0x20) film+0 · W(0x20) film+4 · W(0x659000) film+8 (le REGISTRE,
//	          50 blocs de 0x20800 bits) · W(0xF60) film+0xCB208 (LA TABLE PAR TYPE, 123 u32) ·
//	          puis version / build / saveur, `MOV R9D,0xf60` @14299b1e4.
//
//	LECTURE   `FUN_14299ab50(film, lecteur)` — les DEUX largeurs de tete sont CALCULEES :
//	            R(0x20) film+0 · R(0x20) film+4
//	            R(FUN_141cfff30(film+4)) film+0x8        <- largeur du REGISTRE, par la version
//	            R(FUN_141cffe20(film+4)) film+0xCB208    <- largeur de la TABLE, par la version
//	            puis les memes champs fixes, et `FUN_14299bcb0(film, film+4)` en queue.
//
// LE SECOND u32 N EST DONC PAS « un second entier » : c est LA VERSION DE FORMAT DE
// `chunk_00`, et c est le seul parametre dont la lecture de l en-tete depende.
// `film_major_version.go` le NOMME desormais et renvoie ici (corrige le 2026-09-16, revue de
// jalon M1 : son en-tete se contentait de dire que le second u32 « suit », ce qui laissait
// croire a un entier sans role).
// Les deux fonctions de taille sont des recherches dans deux
// `std::map<uint, int>` construites au demarrage, litteraux relus sur le desassemblage de
// `FUN_140268ec0` et `FUN_140268f40` :
//
//	blocs de registre  `DAT_1450fb368` : {13:47, 17:48, 18:49, 25:25}    defaut 50
//	entrees de table   `DAT_1450fb378` : {13:110, 16:113, 17:114, 21:117, 22:118, 24:121,
//	                                      25:122}                        defaut 123
//
// C est la forme mecanique du fait utilisateur : la grammaire des formats anciens EST dans
// l executable courant, indexee par ce u32. Aucun executable ancien n est necessaire.
//
// # CE QUE LA MESURE DIT, SUR LES 1 351 `chunk_00` DU CACHE (2026-09-15)
//
// Couple (version majeure, version de format), lu sur les huit octets de tete :
//
//	(41, 27) x 1123 · (40, 27) x 146 · (40, 25) x 39 · (39, 24) x 26 · (37, 24) x 10 ·
//	(31, 20) x 3 · (33, 20) x 2 · (38, 24) x 1 · (33, 21) x 1
//
// La version de format REPRODUIT EXACTEMENT la partition par build que `film_identity.go`
// derivait par arithmetique — 1 269 films a 50 blocs / 123 entrees, 39 a 122, 37 a 121, 1 a
// 116 — et elle la COUVRE LA OU LA DERIVATION S ARRETE : les CINQ films sans section
// d identification (`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`) portent tous
// `format = 20`, et ce sont les cinq SEULS du cache a le porter. La section d identification
// apparait donc au format 21, et l absence de chaine de build n est pas une anomalie : c est
// un format anterieur.
//
// # DEUX CONTRADICTIONS MESUREES, CONSIGNEES ET NON PORTEES
//
// Les deux tables de l ecrivain ci-dessus sont RELUES, mais elles ne sont PAS le lecteur de ce
// depot, et deux de leurs lignes contredisent la mesure :
//
//	format 21 : la table annonce 117 entrees, `a521164d` en porte 116 (fermeture arithmetique
//	            verifiee a l octet : chaine de build a 815 864, fin de registre a 815 368).
//	format 25 : la table annonce 25 blocs, les 39 films mesures en portent 49 (chaine de build
//	            a 815 888 sur `e5adf7b2` = exactement 49 blocs et 122 entrees).
//
// La derivation structurelle de `ReadFilmIdentity` (ancrage sur la chaine de build, fin de
// registre derivee du parse) reste donc LA lecture — elle est validee sur les 1 351 chunks —
// et ces deux lignes sont au §4 du plan. Les porter en dur serait poser la table de l ecrivain
// CONTRE la mesure, ce que D13 interdit dans les deux sens.

import (
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// FilmFormatVersionUnknown : la valeur rendue quand l en-tete est trop court pour la porter.
const FilmFormatVersionUnknown = 0

// filmFormatVersionOffset : l octet du u32 de version de format, en tete du registre inflate.
// Il SUIT immediatement `filmMajorVersionOffset` (cf. `film_major_version.go`) — l ecrivain
// ecrit les deux a la file, `14299b1ab` puis `14299b1bd`.
const filmFormatVersionOffset = 4

// FilmFormatVersionFromHeader lit la version de format en tete du registre DECOMPRESSE.
//
// ok=false quand le chunk fait moins de huit octets. L appelant retombe alors sur
// [FilmFormatVersionUnknown] et DOIT le consigner : sans cette version, ni la largeur du
// registre ni celle de la table par type ne sont decidables.
func FilmFormatVersionFromHeader(chunk0 []byte) (int, bool) {
	if len(chunk0) < filmFormatVersionOffset+4 {
		return FilmFormatVersionUnknown, false
	}
	return int(source.U32LE(chunk0, filmFormatVersionOffset)), true
}

// FilmFormatVersion rend la version de format d un film DEJA CHARGE, lue dans son registre.
//
// ok=false quand le film ne porte pas son `chunk_00` (bobine partielle : `minifilm_000d5950`
// est exactement ce cas) ou que cet en-tete est trop court.
//
// ELLE NE DEPEND PAS DE LA SECTION D IDENTIFICATION : c est tout l interet de cette lecture.
// Les cinq films du cache qui n en portent pas rendent quand meme leur version (20).
func FilmFormatVersion(f *source.Film) (int, bool) {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return FilmFormatVersionUnknown, false
	}
	return FilmFormatVersionFromHeader(reg)
}

// MPPWidthsForFormat rend le decoupage du bloc `object-multiplayer-properties` d une version de
// format, ou [ErrUnknownFormat] enveloppe avec la version refusee.
//
// ELLE EXISTE POUR QUE LE CONSOMMATEUR PUISSE DISTINGUER LES DEUX « PAS DE PROFIL ». Un format
// CONNU dont la largeur est indeterminee (20, 21, 24, 25) rend `MPPWidths{}` et une erreur NULLE
// — c est l etat normal du parc ancien, et le repli calibre s y applique depuis le 1.9.1 bis. Un
// format INCONNU rend l erreur — c est l evenement « patch du jeu », et il se compte
// ([UnknownFormatExpvarPairs]). Sans cette frontiere, les deux se liraient pareil et un format
// neuf basculerait tout le parc sur la calibration sans que rien ne le dise.
func MPPWidthsForFormat(format int) (MPPWidths, error) {
	w, ok := mppWidthsPourFormat(format)
	if !ok {
		return MPPWidths{}, erreurFormatInconnu(format)
	}
	return w, nil
}

// ResolutionMPP est CE QUE LA GRAMMAIRE DIT du découpage MPP d'un film : la version de format
// lue, les largeurs qu'elle porte, et LEQUEL des deux silences on a rencontré.
//
// # POURQUOI UN TYPE, ET PAS TROIS RETOURS RECOPIÉS À CHAQUE SITE
//
// Deux sites de production installent ce découpage — les poses d'équipement
// (`ScanEquipmentPlacements`) et les socles d'armes / véhicules (`replay.gwWidthsForFilm`) — et
// ils l'ont longtemps résolu CHACUN DE LEUR CÔTÉ, tous deux par [BuildProfileFromFilm], donc par
// la clé BUILD. Le registre des replis, lui, déclare la condition `format_sans_profil_relu` : la
// clé est la VERSION DE FORMAT. Les deux clés ne coïncident pas — un film au format 27 (largeurs
// RELUES 9/5) dont le build est absent de la table des sept se repliait sur la calibration
// DEVANT une lecture disponible, et `formatSansProfil` rendait faux, donc ni compteur ni
// avertissement (constat 2 de la revue de jalon M1, 2026-09-15, lentille D13).
//
// Les deux sites passent désormais par CETTE porte, et par elle seule.
type ResolutionMPP struct {
	// FormatVersion : le u32 de `chunk_00+4`, ou [FilmFormatVersionUnknown].
	FormatVersion int
	// Widths : le découpage que porte cette version de format. Non valide quand la version est
	// connue mais sa largeur INDÉTERMINÉE (formats 20, 21, 24, 25), ou quand elle est inconnue.
	Widths MPPWidths
	// FormatInconnu : la version de format n'est PAS dans la table. C'est l'événement « patch du
	// jeu », et lui seul se compte ([UnknownFormatExpvarPairs]) — un format connu sans largeur
	// relue est l'état normal du parc ancien.
	FormatInconnu bool
}

// Relue dit si la grammaire porte le découpage de ce film : c'est la condition qui fait DÉCIDER
// la lecture plutôt que la calibration.
func (r ResolutionMPP) Relue() bool { return r.Widths.Valid() }

// MPPWidthsForFilm résout le découpage du bloc `object-multiplayer-properties` d'un film DÉJÀ
// CHARGÉ, par sa VERSION DE FORMAT et par elle seule.
//
// ELLE NE LIT PAS LA SECTION D'IDENTIFICATION, et c'est le point : le nom de build ne key pas
// cette grammaire (lot 1.9.1 ter), donc le faire passer par [BuildProfileFromFilm] — qui refuse
// tout build hors de la table des sept — éteignait la lecture sur des films dont le format la
// portait. Un `chunk_00` absent rend `(FilmFormatVersionUnknown, FormatInconnu)` : l'absence de
// version est, elle aussi, une absence de grammaire, et elle se compte sous
// `filmdec_unknown_format_0`.
// ELLE PASSE PAR [MPPWidthsForFormat] et non par la table brute : c'est cette fonction qui porte
// la frontière entre les deux « pas de profil », et la dédoubler ici les ferait diverger au
// premier format ajouté — le défaut même que ce type existe pour fermer.
func MPPWidthsForFilm(f *source.Film) ResolutionMPP {
	format, ok := FilmFormatVersion(f)
	if !ok {
		return ResolutionMPP{FormatVersion: FilmFormatVersionUnknown, FormatInconnu: true}
	}
	w, err := MPPWidthsForFormat(format)
	return ResolutionMPP{FormatVersion: format, Widths: w, FormatInconnu: err != nil}
}

// unknownFormatCounterName rend le nom expvar du compteur de version de format inconnue.
//
// NOMMAGE (ADR 0009), ET C EST LE MEME PATRON QUE [unknownBuildCounterName] : `<categorie>_
// <sous_cle>` en snake_case, la cause DANS le nom. La sous-cle est ici un ENTIER, donc elle n a
// pas besoin du nettoyage que le nom de build exige — un nom de build vient du film et peut
// porter n importe quoi, une version de format est deja un nombre.
func unknownFormatCounterName(format int) string {
	return "filmdec_unknown_format_" + strconv.Itoa(format)
}

// UnknownFormatExpvarPairs rend le compteur a publier quand la version de format d un film est
// absente de la table de profil ([ErrUnknownFormat]).
//
// # POURQUOI CE COMPTEUR EXISTE, ET POURQUOI IL N ATTEND PAS LE COMPTAGE DU REGISTRE
//
// Le repli `repli_largeurs_mpp_calibrees_sur_le_film` est le BON defaut : un format que ce depot
// ne connait pas encore bascule sur les largeurs CALIBREES sur le film, il n eteint pas le
// decodeur. Mais un repli silencieux sur tout le parc neuf est un incident invisible — au
// prochain patch du jeu, la seule trace serait une derive de qualite sans cause. Le comptage des
// REPLIS du registre est differe au pas 2 de M2 (il attend le porteur par `FilmContext`) ;
// CELUI-CI ne l attend pas, parce qu il ne compte pas un repli ordinaire mais l evenement
// « le jeu a change de format ».
//
// `filmdec` NOMME ses compteurs et ne depend PAS d `internal/observability` — meme patron que
// [UnknownBuildExpvarPairs], dont le cableur vit dans `replay`.
func UnknownFormatExpvarPairs(format int) []ExpvarPair {
	return []ExpvarPair{{Name: unknownFormatCounterName(format), Value: 1}}
}
