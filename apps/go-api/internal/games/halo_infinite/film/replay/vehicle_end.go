package replay

// vehicle_end.go — LA FIN DE VIE D UN VEHICULE, LUE AU LIEU D ETRE INFEREE (lot 1.9.10, D13).
//
// CE QUI DECIDAIT AVANT. `VehicleTrack.End` ne prenait qu une valeur, `unknown`, et la fenetre
// d une vie que le recensement ne fermait jamais etait posee a « dernier recensement + 20 s »
// (`vehicleCensusTolUS`) — un REPLI qui decidait DEVANT une lecture disponible
// (`repli_fin_de_vie_vehicule_par_recensement`, registre, retire par ce lot).
//
// CE QUI DECIDE MAINTENANT : le film. Il ECRIT la destruction dans le composant
// `object-dead-state` de l entite `ti=40` (21 a 27 par film, mesure V13 du 2026-09-05), et le
// seul verrou etait le filtre `DesyncAt == -1` qui jetait des morts parfaitement lues — la
// grammaire et le detail du verrou vivent dans `filmdec/object_deaths.go`.
//
// TROIS FINS, ET PAS UNE DE PLUS. Toutes les trois sont des FAITS, jamais une supposition :
//
//	destroyed  le film ecrit la mort de CETTE vie : l instant est publie (`TEnd`) ;
//	film_end   aucune image-cle ne cesse de la recenser : la vie finit AVEC le film. Ce n est
//	           pas une ignorance, c est une lecture — et c est exactement la population que le
//	           repli retire couvrait (mesure : 88 vies sur 295, 20 artefacts du parc) ;
//	unknown    la vie cesse d etre recensee sans que le film n ecrive sa mort. Le champ garde
//	           ici son sens d origine et son role de contrat : le client ne doit PAS lire
//	           l effacement du sprite comme une explosion.
//
// LE COMPTE DE `unknown` EST LE RESTE A FAIRE, et il se publie (`VehicleCoverage.EndUnknown`) :
// c est la part des vies dont la disparition n est pas datee.
//
// CE FICHIER EST PUR : il ne lit pas le film, il range ce que `VehicleScan.Deaths` a rendu.

import "levelup/go-api/internal/games/halo_infinite/film/grammar"

// vehicleDeathTally est le bilan de l attribution des morts ecrites aux vies recensees.
type vehicleDeathTally struct {
	// read : morts `ti=40` rendues par la marche ; matched : celles qu une vie a reprises ;
	// unmatched : celles qu AUCUNE vie ne reprend.
	read, matched, unmatched int
	// tailDesync : parmi `matched`, celles dont le record a rompu APRES le dead-state.
	tailDesync int
}

// assignVehicleDeaths pose, sur chaque vie, l instant de la mort que le film ECRIT pour elle.
//
// L ATTRIBUTION SE FAIT PAR LA VIE, PAS PAR LE SLOT : le pool de slots reboucle et la generation
// ne fait que 2 bits, donc deux vies distinctes peuvent partager `(slot, gen)`. La fenetre
// [`loUS`, `hiUS`] de la vie — deja posee par `assignVehicleWindows` — departage ; la mort la
// plus PRECOCE de la fenetre est retenue, parce que le film re-replique le dead-state sur
// plusieurs ticks (sur-comptage mesure : 84 dead-states bipedes pour ~60 morts reelles).
//
// UNE MORT QUE PERSONNE NE REPREND EST COMPTEE, JAMAIS JETEE : c est le signal qu une vie
// manque au recensement, et le compte le dit (`VehicleCoverage.DeathsUnmatched`).
func assignVehicleDeaths(lives []vehicleLife, deaths []grammar.ObjectDeath) vehicleDeathTally {
	t := vehicleDeathTally{read: len(deaths)}
	if len(lives) == 0 || len(deaths) == 0 {
		t.unmatched = len(deaths)
		return t
	}
	for _, d := range deaths {
		i := indexOfVehicleLifeAt(lives, d)
		if i < 0 {
			t.unmatched++
			continue
		}
		t.matched++
		if d.TailDesync {
			t.tailDesync++
		}
		if lives[i].deathUS == 0 || d.TimestampUS < lives[i].deathUS {
			lives[i].deathUS, lives[i].deathTailDesync = d.TimestampUS, d.TailDesync
		}
	}
	return t
}

// indexOfVehicleLifeAt rend l index de la vie a qui la mort `d` appartient, ou -1. La vie doit
// porter la MEME cle `(slot, gen)` et sa fenetre doit contenir l instant.
func indexOfVehicleLifeAt(lives []vehicleLife, d grammar.ObjectDeath) int {
	for i := range lives {
		l := &lives[i]
		if l.key.Slot != d.Slot || l.key.Gen != d.Gen {
			continue
		}
		if d.TimestampUS < l.loUS || d.TimestampUS > l.hiUS {
			continue
		}
		return i
	}
	return -1
}

// vehicleEndOf rend la CAUSE de fin d une vie et, quand elle est datee, la frame de cet instant.
//
// L ORDRE EST FIXE ET C EST D14 (b) : on LIT d abord (la mort ecrite), on constate ensuite (la
// vie court jusqu au bout du film), et `unknown` n est pas un repli mais l aveu compte qu aucun
// des deux ne s applique.
func vehicleEndOf(l vehicleLife, clock replayClock) (cause string, tEnd *int) {
	if l.deathUS > 0 {
		f := clock.frame(l.deathUS)
		return VehicleEndDestroyed, &f
	}
	if l.goneByUS == 0 {
		return VehicleEndFilmEnd, nil
	}
	return VehicleEndUnknown, nil
}

// tallyVehicleEnds ventile les vies PUBLIEES par cause de fin, et compte les echantillons
// posterieurs a une fin datee.
//
// CE DERNIER COMPTE EST UNE CONTRADICTION, PAS UN DEFAUT D AFFICHAGE : un vehicule que le film
// declare mort ne devrait plus repliquer sa position. Il se publie plutot que de se corriger en
// silence — couper la trajectoire a la mort masquerait precisement le desaccord qui dirait que
// l attribution est fausse.
func tallyVehicleEnds(tracks []VehicleTrack, cov *VehicleCoverage) {
	for i := range tracks {
		tr := &tracks[i]
		switch tr.End {
		case VehicleEndDestroyed:
			cov.EndDestroyed++
		case VehicleEndFilmEnd:
			cov.EndFilmEnd++
		default:
			cov.EndUnknown++
		}
		if tr.TEnd == nil {
			continue
		}
		for _, s := range tr.Samples {
			if s.T > *tr.TEnd {
				cov.SamplesAfterEnd++
			}
		}
	}
}
