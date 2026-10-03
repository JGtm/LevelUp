package replay

// porteurs_entrees.go — LES GARDES DE MODE ET LES ENTREES DES QUATRE CALQUES DE PORTEUR
// (drapeau, crane, couronne VIP, bombe), POUR LA CUISSON ET POUR LE SYNC.
//
// DEPLACE depuis `replaybuild/matchfacts.go` et `replaybuild/zones.go` le 2026-09-28 (lot V1.4
// du plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`), sans changement de regle. Le plan exige que le
// collecteur lise les porteurs « sous la meme garde de mode que la cuisson » : une garde recopiee
// au sync aurait diverge au premier ajustement d'un nom de variante. Elle est ecrite ICI, une fois,
// et `replaybuild` comme `PortagesAuSync` l'appellent.
//
// # LE CALQUE NE DEVINE TOUJOURS AUCUN MODE
//
// La doctrine des calques est intacte : `attachVipCrown`, `attachSkullCarries` et
// `attachBombCarries` ne lisent que le drapeau `Scanned` de leur entree. Ce qui a demenage, c'est
// le PREDICAT que l'appelant applique pour le poser — il est desormais partage par les deux
// appelants au lieu de vivre chez un seul.

import (
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// GardesDesPorteurs dit, pour une variante, quels calques de porteur se lisent.
type GardesDesPorteurs struct {
	// Drapeau : la variante est de la famille CTF (`ObjectiveTypeOf`). LA CUISSON NE S'EN SERT
	// PAS : elle pose l'entree du drapeau sur tout film et laisse les trois signaux du FILM
	// trancher (cf. [withFlagIdentity]). Le SYNC s'en sert pour ne pas PAYER les lectures du
	// drapeau (statborg, equipes, objets du monde) hors CTF — les signaux du film tranchent
	// ensuite de la meme facon.
	Drapeau bool
	// Crane : Oddball — `comp 0 A` est le score de mode de tout mode, donc lu sur un film d'un
	// autre mode il rendrait de faux porteurs. MEME predicat canonique que la colline.
	Crane bool
	// Bombe : TOUTE la famille bomb, One Bomb comprise depuis le 2026-09-04 (les 4 formes du
	// registre, releve du 2026-08-31 : « Assault:One Bomb », « Assault:Neutral Bomb »,
	// « Assault:Neutral Bomb Squad », « Husky Raid:Assault »). Ce qui protege l'ARMEMENT n'est pas
	// un nom mais la confrontation locale aux explosions du film (`bomb_armings.go`,
	// tout-ou-rien) ; le PORTAGE lit le canal des armes tenues, present dans les 9 films d'Assaut
	// de B1.
	Bombe bool
	// VIP : la couronne — `comp 22 A` vaut `flag_grabs` en CTF, donc elle n'est lue que sur un
	// film reconnu VIP par son nom de variante.
	//
	// CRITERE : le jeton `vip` dans le nom de variante, MEME approche par mot-clef que les autres
	// modes. Le marqueur canonique du mode est `GameVariantCategory=23` (verifie sur les payloads
	// bruts, `VIP_COURONNE_PROTOCOLE.md`), mais la categorie n'est pas portee par les faits de
	// match — le nom l'est. LA GARDE ECHOUE FERMEE : un film VIP dont le nom ne porterait pas
	// `vip` ne montre simplement pas de couronne ; la seule erreur dangereuse — une couronne sur
	// un film non-VIP — exigerait un nom non-VIP contenant `vip`, ce qu'aucune variante ne fait.
	VIP bool
}

// Aucune dit qu'aucun calque de porteur ne se lit pour cette variante.
func (g GardesDesPorteurs) Aucune() bool {
	return !g.Drapeau && !g.Crane && !g.Bombe && !g.VIP
}

// jetonVIP : le jeton de la variante VIP, cherche sans casse.
const jetonVIP = "vip"

// GardesDeLaVariante applique les predicats de mode a `game_variant_name`.
func GardesDeLaVariante(variante string) GardesDesPorteurs {
	famille := objectives.ObjectiveTypeOf(variante)
	return GardesDesPorteurs{
		Drapeau: famille == objectives.ObjectiveTypeFlag,
		Crane:   famille == objectives.ObjectiveTypeSkull,
		Bombe:   famille == objectives.ObjectiveTypeBomb,
		VIP:     strings.Contains(strings.ToLower(variante), jetonVIP),
	}
}

// EntreeDuDrapeau assemble ce que le calque du DRAPEAU VIVANT lit dans le film : les
// enregistrements d'entite (evenements nommes et progressions du compteur de morts) et les BURSTS
// DE CAPTURE — sans eux le discriminant de mode ne tient pas (la table d'emplacements du drapeau,
// appliquee a un film Oddball, rend 1 470 « prises » et 994 « vols »).
//
// LE PONT D'IDENTITE DESCEND JUSQU'ICI DEPUIS LE 2026-09-06 (schema 42). Le calque le resolvait
// lui-meme par les seuls INSTANTS DE MORT, qui exigent TROIS instants coincidents : un joueur qui
// meurt moins de trois fois — le meilleur, celui qui porte le drapeau — lui echappait par
// construction (`c0a82e88` : 3 prises, 3 `noBridge`, 0 portage). Le pont COMPLETE est le meme que
// celui des actions d'objectif.
//
// AUCUN FAIT DE MATCH N'ENTRE DANS LE CALQUE : ce qui descend est une TABLE slot -> xuid. Les
// SOCLES s'y ajoutent chez l'appelant (ils viennent du catalogue de carte, pas du film).
func EntreeDuDrapeau(recs []types.StatRecord, bursts []int, pont *PontParManche) FlagInput {
	return withFlagIdentity(FlagInput{Scanned: true, Records: recs, Bursts: bursts}, pont)
}

// withFlagIdentity pose le pont COMPLETE sur l'entree du calque — et SEULEMENT sur un film que les
// trois signaux reconnaissent comme du CTF, la MEME garde que le calque (`attachFlagCarries`) :
// hors CTF le pont n'est pas resolu du tout (protection du 2026-08-18). Coeur PUR, sans film.
func withFlagIdentity(in FlagInput, pont *PontParManche) FlagInput {
	if flagFilmSignalsOf(in, pont.Consultations()).IsFlagFilm() {
		in.Identity = pont.Identite()
	}
	return in
}

// EntreeDuCrane assemble ce que le PORTEUR DU CRANE lit dans le film — les memes enregistrements
// d'entite que la courbe de score —, gardee par le mode. Hors Oddball elle rend une entree VIDE
// (ni records ni Scanned) et NE TOUCHE PAS au pont : un appelant qui n'aurait pas d'autre raison
// de le reveiller ne le paye pas.
//
// LE PONT DESCEND JUSQU'ICI comme pour le drapeau : le calque le resolvait par les seuls instants
// de mort, et un porteur qui meurt moins de trois fois dans la manche lui echappait (mesure du
// 2026-09-10 sur les quatre films Oddball du parc : `43716616` perdait les 62,3 s de son plus gros
// porteur).
func EntreeDuCrane(recs []types.StatRecord, crane bool, pont *PontParManche) SkullInput {
	if !crane {
		return SkullInput{}
	}
	return SkullInput{Scanned: true, Records: recs, Identity: pont.Identite()}
}

// EntreeDeLaCouronne assemble ce que la COURONNE VIP lit dans le film — les memes enregistrements
// d'entite —, gardee par le mode. Hors VIP, une entree VIDE : ni calque ni couverture.
func EntreeDeLaCouronne(recs []types.StatRecord, vip bool) VipInput {
	if !vip {
		return VipInput{}
	}
	return VipInput{Scanned: true, Records: recs}
}

// EntreeDeLaBombe assemble ce que LA BOMBE lit hors film, sous UNE SEULE garde de mode :
//
//	l'ARMEMENT (schema 33)  l'horloge du manifeste (start_ms par chunk), que le balayage de
//	                        l'anneau demande pour dater sur la meme base que les explosions ;
//	le PORTAGE (schema 34)  aucune donnee de plus (le canal des armes tenues) : la garde seule.
//
// Hors de la famille bomb, une entree VIDE : ni balayage, ni calque, ni couverture.
func EntreeDeLaBombe(horloge map[int]int, bombe bool) BombInput {
	if !bombe {
		return BombInput{}
	}
	return BombInput{CarryScanned: true, Scanned: true, ChunkStartMS: horloge}
}
