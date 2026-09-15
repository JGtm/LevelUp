package filmdec

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
// LE SECOND u32 N EST DONC PAS « un second entier » (c est ainsi que `film_major_version.go` le
// decrivait) : c est LA VERSION DE FORMAT DE `chunk_00`, et c est le seul parametre dont la
// lecture de l en-tete depende. Les deux fonctions de taille sont des recherches dans deux
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
	"encoding/binary"

	"levelup/go-api/internal/analysis/filmsource"
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
	return int(binary.LittleEndian.Uint32(chunk0[filmFormatVersionOffset:])), true
}

// FilmFormatVersion rend la version de format d un film DEJA CHARGE, lue dans son registre.
//
// ok=false quand le film ne porte pas son `chunk_00` (bobine partielle : `minifilm_000d5950`
// est exactement ce cas) ou que cet en-tete est trop court.
//
// ELLE NE DEPEND PAS DE LA SECTION D IDENTIFICATION : c est tout l interet de cette lecture.
// Les cinq films du cache qui n en portent pas rendent quand meme leur version (20).
func FilmFormatVersion(f *filmsource.Film) (int, bool) {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return FilmFormatVersionUnknown, false
	}
	return FilmFormatVersionFromHeader(reg)
}
