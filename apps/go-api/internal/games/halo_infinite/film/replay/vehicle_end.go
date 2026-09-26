package replay

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

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

// vehicleDeathTally est le bilan de l attribution des morts ecrites aux vies recensees.
type vehicleDeathTally struct {
	// read : morts `ti=40` rendues par la marche ; matched : celles qu une vie a reprises ;
	// unmatched : celles qu AUCUNE vie ne reprend.
	read, matched, unmatched int
	// tailDesync : parmi `matched`, celles dont le record a rompu APRES le dead-state.
	tailDesync int
	// closed : les vies DISTINCTES qu une mort a effectivement fermees. IL NE VAUT PAS
	// `matched`, et c est tout l interet de le compter a part : le film RE-REPLIQUE le
	// dead-state sur plusieurs ticks, donc trois morts appariees peuvent ne fermer qu une seule
	// vie. Mesure du 2026-09-18 sur `a349fea8` : 3 appariees, 1 vie fermee.
	closed int
	// LA VENTILATION DES NON-APPARIEES, ET POURQUOI ELLE EXISTE. `unmatched` seul dit qu une
	// mort lue n a trouve personne ; il ne dit pas OU la chaine la perd, et sans cela le
	// correctif se choisit au hasard. Les trois causes sont exclusives et couvrent tout :
	//
	//	noSlot    aucune vie recensee ne porte ce slot — la mort est lue, la vie ne l est pas ;
	//	noGen     des vies portent ce slot, aucune avec cette generation ;
	//	window    une vie porte bien `(slot, gen)`, mais l instant tombe hors de sa fenetre.
	noSlot, noGen, window int
	// windowAvant / windowApres / windowEcartMaxMS precisent le debordement : de quel cote de la
	// fenetre la mort tombe, et de combien au pire. Une fenetre trop etroite d un cheveu et une
	// mort qui appartient a une autre vie ne se corrigent pas de la meme facon.
	windowAvant, windowApres int
	windowEcartMaxMS         int64
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
func assignVehicleDeaths(lives []vehicleLife, deaths []types.ObjectDeath) vehicleDeathTally {
	t := vehicleDeathTally{read: len(deaths)}
	if len(lives) == 0 || len(deaths) == 0 {
		t.unmatched = len(deaths)
		return t
	}
	for _, d := range deaths {
		i := indexOfVehicleLifeAt(lives, d)
		if i < 0 {
			t.unmatched++
			t.compterLaPerte(lives, d)
			continue
		}
		t.matched++
		if d.TailDesync {
			t.tailDesync++
		}
		if lives[i].deathUS == 0 {
			t.closed++
		}
		if lives[i].deathUS == 0 || d.TimestampUS < lives[i].deathUS {
			lives[i].deathUS, lives[i].deathTailDesync = d.TimestampUS, d.TailDesync
		}
	}
	logVehicleDeathTally(t)
	return t
}

// compterLaPerte ventile UNE mort non appariee par sa cause. Elle ne decide rien : elle nomme
// l endroit exact ou la chaine perd une donnee qu elle a su lire.
func (t *vehicleDeathTally) compterLaPerte(lives []vehicleLife, d types.ObjectDeath) {
	slot, gen := false, false
	for i := range lives {
		if lives[i].key.Slot != d.Slot {
			continue
		}
		slot = true
		if lives[i].key.Gen == d.Gen {
			gen = true
		}
	}
	switch {
	case !slot:
		t.noSlot++
	case !gen:
		t.noGen++
	default:
		t.window++
		t.mesurerLEcart(lives, d)
	}
}

// mesurerLEcart retient, pour une mort qui a sa vie mais tombe hors de sa fenetre, DE QUEL COTE
// elle deborde et DE COMBIEN. C est cette mesure, et elle seule, qui dit si la fenetre est trop
// etroite d un cheveu ou si la mort appartient a une autre vie.
func (t *vehicleDeathTally) mesurerLEcart(lives []vehicleLife, d types.ObjectDeath) {
	meilleur := int64(-1)
	apres := false
	for i := range lives {
		l := &lives[i]
		if l.key.Slot != d.Slot || l.key.Gen != d.Gen {
			continue
		}
		var ecart int64
		cote := false
		if d.TimestampUS < l.loUS {
			ecart = int64(l.loUS - d.TimestampUS)
		} else {
			ecart, cote = int64(d.TimestampUS-l.hiUS), true
		}
		if meilleur < 0 || ecart < meilleur {
			meilleur, apres = ecart, cote
		}
	}
	if meilleur < 0 {
		return
	}
	if apres {
		t.windowApres++
	} else {
		t.windowAvant++
	}
	if ms := meilleur / 1000; ms > t.windowEcartMaxMS {
		t.windowEcartMaxMS = ms
	}
}

// logVehicleDeathTally dit ce que l attribution a perdu, et par quelle cause.
//
// UN COMPTEUR MUET EST UNE ERREUR AVALEE (grille de revue, anti-patron 10) : jusqu ici une mort
// lue que personne ne reprenait disparaissait dans un entier, et le lot 3.7 a du la
// re-instruire film par film pour decouvrir qu elle valait 17 sur 20. Le niveau est `Warn` des
// qu une mort se perd : c est une donnee du film que le document ne portera pas.
func logVehicleDeathTally(t vehicleDeathTally) {
	if t.read == 0 {
		return
	}
	niveau := slog.Info
	if t.unmatched > 0 {
		niveau = slog.Warn
	}
	niveau("rejeu : attribution des morts de vehicule",
		"lues", t.read, "appariees", t.matched, "viesFermees", t.closed,
		"nonAppariees", t.unmatched, "queueRompue", t.tailDesync,
		"perdues_slotAbsent", t.noSlot, "perdues_generationAbsente", t.noGen,
		"perdues_horsFenetre", t.window, "horsFenetre_avant", t.windowAvant,
		"horsFenetre_apres", t.windowApres, "horsFenetre_ecartMaxMS", t.windowEcartMaxMS)
}

// indexOfVehicleLifeAt rend l index de la vie a qui la mort `d` appartient, ou -1. La vie doit
// porter la MEME cle `(slot, gen)` et sa fenetre doit contenir l instant.
func indexOfVehicleLifeAt(lives []vehicleLife, d types.ObjectDeath) int {
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
