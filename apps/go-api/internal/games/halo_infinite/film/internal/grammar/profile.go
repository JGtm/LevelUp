package grammar

// profile.go — CE QUI VA CHERCHER LE PROFIL D UN FILM DANS LE FILM (D1 du PLAN_DECODEUR_FILM,
// D-3 d ADR 0034).
//
// # LE PROFIL LUI-MEME N EST PLUS ICI DEPUIS LE LOT 2.5.b
//
// Le TYPE [profile.Profile], ses sous-profils, la table par build et par format, le catalogue de
// cartes et la composition [profile.Resoudre] vivent dans la couche `profile` : ce sont des
// DONNEES. Ce fichier garde les TROIS fonctions qui lisent le `chunk_00` d un film pour en tirer
// les cles, et rien d autre.
//
// C EST L INVERSION DE DEPENDANCE DU LOT, ET ELLE SE LIT DANS LE SENS DES IMPORTS : `grammar`
// importe `profile`, jamais l inverse (ratchet `archlint/film_layers_deps_test.go`, regle R1).
// Avant le lot, la lecture et la table etaient dans la MEME fonction — c est cette couture qui
// aurait fait remonter `profile` vers `grammar`, et c est elle que ce fichier defait.
//
// # LE PROFIL NE MET AUCUN FILM DE COTE
//
// D-4 d ADR 0034 interdit de lire un film au profil du build voisin — pas de le lire du tout.
// Une cle absente de la table rend une ERREUR TYPEE ([profile.Profile.Err], `errors.Is` sur
// [profile.ErrUnknownFormat] / [profile.ErrUnknownBuild]) que l appelant consulte, et le reste
// du profil est pose : les invariants et la carte ne dependent d aucune cle. Les replis
// existants (calibration MPP, largeurs d axe par defaut) continuent de tourner, comptes comme
// aujourd hui.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ResolveProfile resout le profil d un film DEJA CHARGE, sous l entree de catalogue de sa carte.
//
// ELLE LIT, ELLE NE COMPOSE PAS : les trois cles sortent du `chunk_00` ici, et
// [profile.Resoudre] les range dans la table. C est une fonction pure de (octets du film, entree
// de carte) : aucune variable de paquet n est lue ni ecrite, aucun fichier n est ouvert.
//
// `film` nil et `entry` nil sont ACCEPTES : le profil rend alors ses invariants, sa carte nulle,
// et [profile.Profile.Err] porte les cles manquantes.
func ResolveProfile(film *source.Film, entry *profile.MapQuantEntry) profile.Profile {
	reg, ok := FilmRegistryChunk(film)
	if !ok {
		// Bobine partielle ou fixture sans `chunk_00` : AUCUNE cle n est lisible. Les
		// sentinelles des deux lecteurs disent « non lue » ; `RegistrePresent` a faux fait
		// rendre les deux erreurs, dans le vocabulaire des cles et non dans celui du fichier.
		return profile.Resoudre(profile.ClesDuFilm{
			Format:  FilmFormatVersionUnknown,
			Majeure: FilmMajorVersionUnknown,
		}, entry)
	}
	cles := profile.ClesDuFilm{RegistrePresent: true}
	// LA VERSION DE FORMAT SE RANGE TELLE QUE LE LECTEUR LA REND : il rend deja
	// [FilmFormatVersionUnknown] quand l en-tete est trop court, donc le drapeau de lecture
	// n ajoute rien ici — la sentinelle EST l absence.
	cles.Format, _ = FilmFormatVersionFromHeader(reg)
	cles.Majeure, cles.MajeureLue = FilmMajorVersionFromHeader(reg)
	// L IDENTITE EST LUE APRES LE FORMAT, et l ordre compte : un `chunk_00` sans section
	// d identification (5 films du cache) porte quand meme sa version de format. Lire le build
	// d abord ferait perdre le format de ces cinq-la.
	if id, err := ReadFilmIdentity(reg); err == nil {
		cles.Identite, cles.IdentiteLue = id, true
	}
	return profile.Resoudre(cles, entry)
}

// HighlightProfileOfFilm rend l implantation du gamertag d un film DEJA CHARGE : la seule part
// du profil dont les lecteurs de temps forts aient besoin.
//
// POURQUOI UNE PORTE ETROITE, ET PAS [ResolveProfile] : les trois sites qui decoupent un bloc
// d evenement de temps fort n ont pas de carte, et deux d entre eux n ont pas de film complet.
// Leur faire resoudre le profil ENTIER — donc lire la section d identification et la table par
// type — couterait une analyse de registre de plus par appel pour une valeur qui tient dans les
// quatre premiers octets. La VALEUR est la meme : c est la meme fonction
// ([profile.HighlightDepuisMajeure]) qui la compose ici et dans [profile.Resoudre].
func HighlightProfileOfFilm(f *source.Film) profile.HighlightProfile {
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		return profile.HighlightDepuisMajeure(FilmMajorVersionUnknown, false)
	}
	return HighlightProfileFromHeader(reg)
}

// HighlightProfileFromHeader rend l implantation du gamertag depuis un `chunk_00` DECOMPRESSE.
//
// C est la forme des appelants qui tiennent des octets bruts et pas un film (le backfill du flux
// de medailles lit son registre au cache). Un chunk trop court rend l implantation historique
// avec [profile.HighlightProfile.Lue] a faux : la degradation se NOMME, elle ne se tait pas.
func HighlightProfileFromHeader(chunk0 []byte) profile.HighlightProfile {
	majeure, lue := FilmMajorVersionFromHeader(chunk0)
	return profile.HighlightDepuisMajeure(majeure, lue)
}
