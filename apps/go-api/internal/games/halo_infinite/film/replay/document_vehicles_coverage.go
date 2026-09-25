package replay

// document_vehicles_coverage.go — CE QUE LE CALQUE DES VEHICULES DIT DE LUI-MEME : le decompte
// des vies publiees, celui des occupations, et le journal de cuisson qui les rend.
//
// DEPLACEMENT PUR depuis `document_vehicles.go` (lot 2.7 volet publication, 2026-09-16) : le
// fichier passait 500 lignes en portant DEUX sujets — la FORME publiee du calque (les types et
// ce que le film prouve de chacun de leurs champs, restee la-bas) et sa MESURE (ici). Aucune
// ligne n a change : memes fonctions, meme ordre, memes seuils.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// tallyVehicleCoverage compte, sur les vies PUBLIEES, ce que la couverture annonce. Un compteur
// qui se remplirait ailleurs qu ici finirait par diverger du tableau qu il decrit.
//
// `fb` (nil-safe) EST LE COMPTEUR DE REPLIS DE LA CUISSON (lot 1.9.9, D14 (c)) : une vie dont le
// chassis est LU mais absent de la table des familles declenche
// `repli_chassis_vehicule_marqueur_neutre`. Il se compte ICI et pas au site de la lecture
// (`vehicleTrackOf`) pour deux raisons : la limite de cinq parametres du depot y est deja
// atteinte, et surtout la couverture decrit les vies PUBLIEES — une vie assemblee puis fondue
// dans un relais (`mergeVehicleRelays`) ne doit pas compter un repli que l artefact ne porte pas.
func tallyVehicleCoverage(tracks []VehicleTrack, cov *VehicleCoverage, fb *fallback.Compteur) {
	cov.Published = len(tracks)
	for _, tr := range tracks {
		if tr.Spawn != nil {
			cov.WithSpawn++
		}
		if tr.Chassis != "" {
			cov.WithChassis++
			switch {
			case tr.Family != "":
				cov.FamilyResolved++
			case tr.Part != "":
				// UNE PIECE MONTEE N EST PAS UN CHASSIS INCONNU (schema 69) : la table des pieces
				// la nomme (`vehicle_turrets.go`) et `coverage.vehicles.turrets` la compte. La
				// ranger ici ferait declencher le marqueur neutre sur une vie qui n est pas dessinee.
			default:
				cov.FamilyUnknown++
				cov.UnknownChassis[tr.Chassis]++
				// D14 (b) : la LECTURE a eu lieu (`tr.Chassis` est le mot d identite lu dans le
				// record de creation) et c est la TABLE qui se tait — le repli entre APRES elle.
				fb.Declenche(fallback.NomChassisVehiculeMarqueurNeutre)
			}
		}
		cov.Samples += len(tr.Samples)
		for _, s := range tr.Samples {
			if s.H != 0 {
				cov.WithHeading++
			}
		}
		tallyVehicleRides(tr.Rides, cov)
	}
}

// tallyVehicleRides compte les episodes d UNE vie et releve leurs chevauchements.
func tallyVehicleRides(rides []VehicleRide, cov *VehicleCoverage) {
	if len(rides) == 0 {
		return
	}
	cov.VehiclesRidden++
	cov.Rides += len(rides)
	for i, r := range rides {
		if r.XUID != "" {
			cov.RidesNamed++
		}
		if r.Seat != nil {
			cov.RidesWithSeat++
		}
		// LA FENETRE EST INCLUSIVE aux deux bouts (`T0` et `T1` sont deux frames affichees) :
		// un episode d une seule frame en couvre UNE, pas zero.
		cov.AimRideFrames += r.T1 - r.T0 + 1
		if len(r.Aim) > 0 {
			cov.RidesWithAim++
			cov.AimSamples += len(r.Aim)
		}
		if r.Src == VehicleRideSrcFilm {
			cov.RidesRead++
		} else {
			cov.RidesProximity++
		}
		// Les episodes d une vie sont TRIES par T0 : un chevauchement se voit sur le voisin.
		// UN ARTILLEUR REPORTE D UNE TOURELLE (`Turret`, schema 69) N EST PAS UNE AMBIGUITE : sa
		// place a bord est designee par la piece qu il sert, il ne dispute rien au conducteur.
		if i > 0 && r.T0 <= rides[i-1].T1 && r.Turret == nil && rides[i-1].Turret == nil {
			cov.Ambiguous++
		}
	}
}

// logVehicleCoverage journalise le calque avec les MEMES denominateurs que l artefact.
//
// LE SILENCE QU IL FAUT ROMPRE : des vies publiees dont AUCUNE ne resout de famille n est pas
// « un film sans vehicule reconnaissable », c est une lecture qui a echoue en bloc — largeurs du
// bloc MPP non reinstallees, ou grammaire du default-state qui a bouge. Sans ce warn, un film
// entier sortirait avec zero sprite sans que rien ne le signale.
func logVehicleCoverage(c *VehicleCoverage) {
	if c == nil {
		return
	}
	slog.Info("rejeu : vehicules",
		"balaye", c.Scanned, "viesRecensees", c.Lives, "publiees", c.Published,
		"relaisFusionnes", c.Merged,
		"sansPosition", c.NoPosition, "avecNaissance", c.WithSpawn, "avecChassis", c.WithChassis,
		"famillesResolues", c.FamilyResolved, "famillesInconnues", c.FamilyUnknown,
		"echantillons", c.Samples, "avecCap", c.WithHeading)
	// LA PORTE DES POSITIONS (lot M1 des retours du rejeu) : ce qu elle a ecarte, et les silences
	// avec deplacement qu elle a laisses publies faute de preuve pour trancher.
	if c.EchantillonsHorsEmprise+c.SpawnsHorsEmprise+c.EchantillonsAuTraversDUnSilence+c.SilencesNonTranches > 0 {
		slog.Info("rejeu : porte des positions de vehicule",
			"echantillonsHorsEmprise", c.EchantillonsHorsEmprise, "spawnsHorsEmprise", c.SpawnsHorsEmprise,
			"echantillonsAuTraversDUnSilence", c.EchantillonsAuTraversDUnSilence,
			"silencesNonTranches", c.SilencesNonTranches)
	}
	slog.Info("rejeu : occupation des vehicules",
		"episodes", c.Rides, "vehiculesOccupes", c.VehiclesRidden, "occupantsNommes", c.RidesNamed,
		"lus", c.RidesRead, "parProximite", c.RidesProximity,
		"avecSiege", c.RidesWithSeat, "ambigus", c.Ambiguous,
		"lecturesDeViseeBrutes", c.AimReads, "episodesAvecVisee", c.RidesWithAim,
		"pointsDeVisee", c.AimSamples, "framesDEpisode", c.AimRideFrames)
	// LE CYCLE DE REAPPARITION porte ses deux moities de denominateur au journal comme il les
	// porte a l artefact : `emplacements` et `ecarts` disent ce que le film offrait, `cycles` ce
	// qui a passe la regle de stabilite, `manques` les occasions perdues faute d une fin datee.
	// Sans eux, « 0 cycle » ne distinguerait pas un film sans emplacement d un film dont tous les
	// ecarts etaient trop disperses.
	slog.Info("rejeu : cycle de reapparition des vehicules",
		"emplacements", c.CycleLocations, "cycles", c.Cycles,
		"ecarts", c.CycleGaps, "manques", c.CycleMissing)
	// LE SILENCE QU IL FAUT ROMPRE, et il est le pendant exact du warn de `logVehicleCoverage` :
	// des episodes publies dont AUCUN ne porte de visee n est pas « un film ou personne ne
	// regardait », c est le balayage `i21` sans position qui n a rien rendu. La mesure V11 rend
	// 35 episodes attestes sur 35 porteurs d au moins une lecture, sur 5 films.
	if c.Rides > 0 && c.RidesWithAim == 0 {
		slog.Warn("rejeu : AUCUN episode d occupation ne porte de visee alors que des episodes"+
			" existent — le balayage des records de visee SANS position n a rien rendu, le cone"+
			" retombe partout sur le cap du chassis",
			"episodes", c.Rides, "lecturesBrutes", c.AimReads)
	}
	// UN CHASSIS INCONNU EST UN AVERTISSEMENT DEPUIS LE LOT 1.9.9 (2026-09-16), une ligne par
	// chassis et par cuisson. Decision utilisateur du 2026-09-14 : le parc d assets vehicules est
	// COMPLET, donc un chassis absent de la table est un MISMATCH a nommer — pas une information
	// de routine. Ce journal est le pendant lisible du repli
	// `repli_chassis_vehicule_marqueur_neutre`, compte par `tallyVehicleCoverage`.
	//
	// `slog.Warn` ET NON `slog.WarnContext` : toute la chaine d assemblage du calque est PURE et
	// ne porte aucun `context.Context` (meme convention que les douze autres journaux de ce
	// fichier et de `build_vehicles.go`). Lui en faire traverser un pour cette seule ligne
	// changerait la signature de six fonctions du lot voisin 1.9.10.
	for id, n := range c.UnknownChassis {
		slog.Warn("rejeu : chassis de vehicule ABSENT DE LA TABLE DES FAMILLES — vies publiees"+
			" sans sprite, dessinees en marqueur neutre ; le mot d identite est LU, c est la table"+
			" qui ne le nomme pas",
			"chassis", id, "vies", n, "repli", string(fallback.NomChassisVehiculeMarqueurNeutre))
	}
	if c.WithChassis > 0 && c.FamilyResolved == 0 {
		slog.Warn("rejeu : AUCUN chassis de vehicule resolu alors que le mot d identite a ete lu"+
			" — table de familles a completer, ou lecture du bloc MPP a verifier",
			"chassisLus", c.WithChassis)
	}
}
