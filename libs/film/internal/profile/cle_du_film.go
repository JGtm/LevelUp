package profile

// cle_du_film.go — LA CLE QU UN FILM ECRIT, ET CE QUE LA TABLE DU PROFIL EN SAIT (lot 3.1.1).
//
// # POURQUOI CE FICHIER EXISTE, ET CE QU IL CORRIGE
//
// D-4 d ADR 0034 dit qu un film dont la cle est absente du profil est MIS DE COTE. Jusqu au lot
// 3.1.1 la moitie « mis de cote » n etait pas tenue : [Profile.Err] portait l erreur typee, mais
// aucun appelant de production ne la lisait (D1 (cloture M2), mesure du 2026-09-17). Ce lot la
// fait lire par les deux orchestrateurs — `sync/killcollector` et `replaybuild`.
//
// # LA MESURE QUI A OBLIGE A UNE FONCTION A PART, ET NON A `Profile.Err()`
//
// [Profile.Err] MELANGE DEUX CHOSES, et l une des deux n est pas une cle inconnue :
//
//	film sans section d identification  `Resoudre` rend [ErrUnknownBuild] enveloppe avec un build
//	                                    VIDE. Or ces films-la sont CONNUS du profil : leur cle
//	                                    est `majeure=31` ou `majeure=33`, que la table porte
//	                                    (amorce de grenade du lot 3.3.1, empreintes de registre
//	                                    du lot 3.2.1). Cinq films du cache sont dans ce cas, et
//	                                    deux d entre eux sont des temoins du corpus gate.
//	build hors table                    LA vraie cle inconnue : le film ECRIT un nom que ce depot
//	                                    ne connait pas. Zero film du cache au 2026-09-17.
//
// Mettre de cote sur `Profile.Err() != nil` aurait donc ecarte cinq films que le depot sait lire
// — dont `50247b26` et `a349fea8`, deux temoins — et le corpus gate aurait compte deux PERTES.
// [CleConnue] tranche sur la CLE ECRITE, comme [AmorceGrenadePour] : le build quand le film en
// ecrit un, la version majeure sinon.
//
// `Resoudre` N EST PAS CORRIGE, et c est delibere : sa valeur d erreur commande
// `coverage.decoder.build` (V15 (15) : chaine vide et bloc present dans les DEUX cas), et la
// changer deplacerait le contenu cuit d un lot qui ne doit pas y toucher.
//
// # LES TROIS CLES, ET L ORDRE DANS LEQUEL ELLES SE LISENT
//
//	format    `chunk_00+4`. Absent de la table = le jeu a change de format de bits.
//	build     la chaine en clair de la section 2. Absent de la table = mise a jour du jeu.
//	majeure   `chunk_00+0`. Elle ne sert de cle que lorsque le film n ecrit PAS de build.

import "errors"

// CleConnue dit si la table du profil connait la cle ECRITE par un film.
//
// Les trois cles sont testees dans l ordre de ce qu elles commandent : la grammaire des bits
// (format), puis la taille des structures de contenu (build), puis — et seulement quand le film
// n ecrit pas de build — la version majeure.
//
// ELLE NE REND PAS D ERREUR ET N EN JOURNALISE AUCUNE : c est une lecture de table. L erreur
// TYPEE que l orchestrateur remonte est composee par [ErreurCleInconnue], sur les memes
// arguments, pour que les deux ne puissent pas diverger.
func CleConnue(format int, build string, majeure int) bool {
	if _, connu := MPPPourFormat(format); !connu {
		return false
	}
	if build != "" {
		_, connu := PersonnalisationOctets(build)
		return connu
	}
	return MajeureSansSectionConnue(majeure)
}

// MajeureSansSectionConnue dit si la version majeure d un film SANS section d identification est
// une cle de la table du profil.
//
// LES DEUX VALEURS SONT CELLES DE LA TABLE DES AMORCES DE GRENADE (lot 3.3.1) : les tables du
// profil portent les MEMES neuf clefs, sept builds et deux majeures. Le garde-rail qui l exige
// est `TestLesTablesDuProfilPortentLesMemesClefs` — sans lui, une table gagnerait une clef que
// les autres ignoreraient, et « le profil connait ce film » n aurait plus de sens unique.
func MajeureSansSectionConnue(majeure int) bool {
	switch majeure {
	case majeureSansSection31, majeureSansSection33:
		return true
	}
	return false
}

// ErreurCleInconnue rend l erreur TYPEE d une cle absente de la table, ou nil quand la cle est
// connue. C est la valeur que les orchestrateurs testent par `errors.Is` avant de mettre un film
// de cote (D-4).
//
// ELLE ENVELOPPE LES MEMES SENTINELLES QUE `Resoudre` — [ErrUnknownFormat], [ErrUnknownBuild] —
// avec la cle refusee : un journal qui dit « cle inconnue » sans dire LAQUELLE ne sert a rien le
// jour du patch du jeu.
//
// UN FILM SANS SECTION DONT LA MAJEURE EST INCONNUE rend [ErrUnknownBuild] enveloppe avec un
// build VIDE : c est la forme que `Resoudre` emploie deja pour cette population, et la reprendre
// evite deux vocabulaires pour un meme refus.
func ErreurCleInconnue(format int, build string, majeure int) error {
	var errs []error
	if _, connu := MPPPourFormat(format); !connu {
		errs = append(errs, ErreurFormatInconnu(format))
	}
	switch {
	case build != "":
		if _, connu := PersonnalisationOctets(build); !connu {
			errs = append(errs, ErreurBuildInconnu(build))
		}
	case !MajeureSansSectionConnue(majeure):
		errs = append(errs, ErreurBuildInconnu(""))
	}
	return errors.Join(errs...)
}
