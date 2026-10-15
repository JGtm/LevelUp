package grammar

// malleable_du_film.go — LE DRAPEAU FINAL DU BLOC MALLEABLE DU BIPEDE, SELON LA VERSION QUE LE FILM
// DECLARE.
//
// # LE FAIT
//
// Le bloc de propriete malleable du bipede (i53 `biped-malleable-property-component`, lecteur
// `FUN_140ff6764` -> `FUN_1407efc5c`) se termine, dans l executable lu, par un drapeau R(1) (lu a
// 1407efeba). Le flux des films qui declarent la version majeure 0x25 (`chunk_00 + 0`,
// [FilmMajorVersion] ; build HI_1_8_0) n a PAS ce bit : lu, il decale d un bit tout ce qui suit et
// fait dire au R(5) d i53 une largeur fausse (+6), puis a l etiquette d i57 une branche fausse
// (+26). Sans lui, les records d image-cle de ces films ferment ; avec lui, ils ne ferment pas. La
// lecture equivalente « R(5) de n sur 4 bits » est indiscernable sur toutes les donnees mesurees ;
// celle-ci est retenue parce que le drapeau final est le dernier champ du bloc.
//
// # POURQUOI LA VERSION MAJEURE, ET POURQUOI LA SEULE VALEUR 0x25
//
// Entre un film 1.8 et un film 1.9, le registre de `chunk_00` est identique a l octet, sauf la
// version majeure (0x25 contre 0x26) : c est la donnee que le film declare et qui separe les deux
// grammaires (rapport `.ai/V7.5/RAPPORT_BIT_BUILDS_18_2026-10-09.md`, §2). L executable courant ne lit
// pas ce flux : son seul test de version majeure (`FUN_1428e219c`, `CMP dword ptr [RDX],0x29` a
// 1428e21a6) ne lance la lecture du film que pour la majeure 0x29 ; sinon il appelle
// `FUN_142988e98`, qui remet a zero l etat du lecteur de film (`+0x1488`) sans lire un octet du
// film. Il n y a donc pas de branche du jeu a porter : la regle est MESUREE, et elle se branche sur
// ce que le film declare, jamais sur une chaine de build.
//
// ELLE NE VAUT QUE POUR LA VALEUR DECLAREE 0x25 (decision de l utilisateur du 2026-10-15, option b) :
// c est la seule ou la mesure l etablit (les records ferment, aucun ne se perd). Les majeures plus
// anciennes (0x1f a 0x22, formats 20 a 23) n y suivent ni cette grammaire ni celle de l executable
// pour tous leurs records, et gardent celle de l executable ; les majeures >= 0x26 ont le bit.
//
// Une version majeure illisible (film sans `chunk_00`) garde la grammaire de l executable courant.

import "levelup/go-api/internal/games/halo_infinite/film/internal/source"

// majeureSansDrapeauFinalMalleable : la version majeure declaree dont le flux n a pas le drapeau
// final du bloc malleable du bipede (mesure, cf. l en-tete).
const majeureSansDrapeauFinalMalleable = 0x25

// malleableSansDrapeauFinal dit si le film declare la version majeure dont le bloc malleable du
// bipede n a pas de drapeau final.
func malleableSansDrapeauFinal(f *source.Film) bool {
	v, ok := FilmMajorVersion(f)
	return ok && v == majeureSansDrapeauFinalMalleable
}
