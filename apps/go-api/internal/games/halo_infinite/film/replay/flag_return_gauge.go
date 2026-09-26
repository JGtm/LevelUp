package replay

// flag_return_gauge.go — LA JAUGE DE RETOUR D UN DRAPEAU LACHE (schema 63), et comment un slot de
// `ti=13` se reconnait comme celle d UN drapeau precis.
//
// # LE RETOUR N EST PAS UN MINUTEUR, C EST UNE JAUGE — ET C EST LE FILM QUI LE DIT
//
// Le calque du drapeau a cherche DEUX FOIS un compte a rebours, et les deux voies sont des
// negatifs MESURES : le bassin du moteur (`ti=11 i0` = « aucun minuteur », 446 records sur 446,
// note 3.7 § 6 bis.3) et le minuteur manuel du navpoint (`ti=12 i11`/`i12` nul sur 588 lectures
// sur 588, lot 5.1.2, dont 172 A L INTERIEUR d un lacher). Le modele etait faux : le script du
// mode (`parcel_deliver_object.lua`) ne decompte pas, il REMPLIT — au taux `1/reset + H(n)/solo`,
// ou `n` est le nombre de defenseurs dans le rayon et `H` la serie harmonique. Le jeu nomme
// lui-meme la fonction `CalculateReturnRateHarmonic`.
//
// La jauge est `ti=13 i1` TAG 3, mode A, voie DELTA — le MEME canal que la jauge de capture des
// zones, en production depuis le schema 18 (cf. zone_states_gauge.go). Un slot par drapeau.
//
// # L APPARIEMENT EST UNE CORRELATION, ET ELLE EST TOTALE (lot 5.1.6, 2026-09-18)
//
// Le slot ne se NOMME pas : `i0` (le `StringId` du nom de propriete) est marche mais jamais
// recolte, et le hacher contre des noms Lua candidats serait une identification par supposition.
// Il se RECONNAIT, par la part de ses echantillons qui tombent dans un lacher DU MEME drapeau :
//
//	film        slot   drapeau   echantillons dans un lacher DE CE drapeau
//	bcb6d393    1490      0       111 / 111  = 100,0 %
//	fb1a1a72    1614      0       453 / 453  = 100,0 %
//	fb1a1a72    1619      1       320 / 320  = 100,0 %
//
// 884 echantillons, tous dans un lacher de LEUR drapeau, zero en dehors. Le contraste est fourni
// par le film lui-meme : le slot 1495 de `bcb6d393` n y tombe qu a 15,8 %. Le seuil vit dans
// l ecart entre 15,8 % et 100 %, et il est ecrit une seule fois ci-dessous.
//
// L ORACLE QUI VALIDE LA LECTURE EST BINAIRE ET SANS EXCEPTION, par lacher, selon CE QUI LE
// TERMINE : retour automatique (`home`) 13 / 13 atteignent le plein (>= 0,99) ; repris par un
// joueur avant la fin 0 / 14. Separation totale sur 27 lachers, trois jauges, deux films.
//
// # DEUX PIEGES, ET ILS SONT ECRITS ICI PARCE QU ILS SE PAIENT EN SILENCE
//
// L ECHELLE EST 0 -> 1, PAS 0 -> 100 : [-100, +100] est la plage de SERIALISATION du tag 3, la
// valeur portee est une fraction normalisee. `gaugeProgressOf` (zone_states.go) porte deja cette
// conversion, MESUREE sur les jauges de zone, et c est elle qui sert ici — une seconde echelle
// serait une seconde verite.
//
// LE TAUX N EST PAS CONSTANT et la jauge SE VIDE : deux lachers de meme duree ne montent pas
// pareil (18,2 s -> 1,0000 ; 26,9 s -> 0,3209), et jusqu a 20 redescentes sur un seul lacher
// quand plus personne n est dans la zone. RIEN DE CELA N EST MODELISE : la jauge se LIT. C est
// aussi pourquoi le client recoit la SERIE et non un sommet — un sommet tenu pendant tout
// l intervalle se lirait « le retour est imminent » alors que la jauge est retombee a zero
// (c est exactement l erreur que la v18 a corrigee sur les zones).
//
// # CE QUI N EST PAS PUBLIE, ET POURQUOI
//
// La CONTESTATION (`Contested` / `ContestedRefilling` du script) : ni observee par l utilisateur,
// ni lisible dans le script (constantes dedupliquees), et la mesure explique le silence — a 1,3 m
// d un drapeau tombe, un ennemi ne conteste pas, il RAMASSE (cf. FlagReturnZone).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

const (
	// flagReturnGaugeMinShare est la part MINIMALE des echantillons d un slot qui doivent tomber
	// dans un lacher DU MEME drapeau pour que le slot soit reconnu comme sa jauge de retour.
	//
	// LE SEUIL VIT DANS UN ECART MESURE, il n est pas choisi : les trois jauges du corpus sont a
	// 100,0 % (111/111, 453/453, 320/320) et le meilleur slot NON-jauge a 15,8 %. Tout seuil
	// entre les deux rend le meme verdict ; 0,80 laisse de la marge a un film ou quelques
	// echantillons tomberaient dans un trou du calque (une fin de portage non datee borne mal son
	// lacher) sans jamais s approcher du contraste.
	flagReturnGaugeMinShare = 0.80
	// flagReturnGaugeMinSamples est le nombre d echantillons en deca duquel un slot ne se juge
	// PAS, quelle que soit sa part. Deux echantillons tombes par hasard dans un lacher font
	// 100 % ; les jauges reelles en portent 111, 320 et 453. Huit est un plancher, pas une
	// mesure — il ecarte la coincidence sans jamais approcher les populations observees.
	flagReturnGaugeMinSamples = 8
)

// flagGaugeSlot est ce qu un slot candidat porte : ses echantillons dates sur l axe du rejeu.
type flagGaugeSlot struct {
	slot    uint32
	samples []zoneSample
}

// attachFlagReturnGauges pose la SERIE de la jauge de retour sur chaque intervalle `dropped` des
// drapeaux publies, et rend les denominateurs du calque.
//
// L ORDRE EST CELUI DE LA PREUVE : on ne peut apparier un slot a un drapeau qu une fois les
// intervalles `dropped` CONNUS, puisque c est la correlation a ces intervalles qui identifie le
// slot. La jauge se pose donc APRES l assemblage des vies, jamais pendant.
func attachFlagReturnGauges(
	carries []FlagCarry, in FlagInput, c matchClock, cov *FlagCarriesCoverage,
) {
	if cov != nil {
		cov.GaugeScanned = in.GaugeScanned
	}
	if !in.GaugeScanned || len(carries) == 0 || c.step == 0 {
		return
	}
	slots := flagGaugeSlotsOf(in.Gauge, c)
	if cov != nil {
		cov.GaugeSlots = len(slots)
		for _, s := range slots {
			cov.GaugeReads += len(s.samples)
		}
	}
	bound := bindFlagGauges(slots, carries)
	gap := zoneGaugeGapFrames(int(c.step / 1000))
	for f := range carries {
		s, ok := bound[f]
		if !ok {
			continue
		}
		if cov != nil {
			cov.GaugePaired++
		}
		fillFlagReturnProgress(carries[f].Spans, s.samples, gap, cov)
	}
}

// flagGaugeSlotsOf retient les seules lectures qui PEUVENT etre une jauge — le variant scalaire
// (mode A, `i1`) au tag 3, charge presente — et les date sur l axe du rejeu, par slot et triees.
//
// LES AUTRES TAGS NE SONT PAS DES CANDIDATS ECARTES, ILS SONT D UN AUTRE TYPE : le tag dit le
// TYPE de la propriete reseau (cf. components_managed_property.go). Un booleen ou un identifiant
// de chaine ne se compare pas a une fraction.
func flagGaugeSlotsOf(reads []grammar.ManagedPropertyRead, c matchClock) []flagGaugeSlot {
	bySlot := map[uint32][]zoneSample{}
	for _, r := range reads {
		if r.Field != grammar.ManagedPropertyScalar || r.Tag != grammar.ManagedPropertyTagQuant || !r.HasValue {
			continue
		}
		f := frameOf(r.TimestampUS, c.origin, c.step)
		if f < 0 || f >= c.frames {
			continue
		}
		bySlot[r.Slot] = append(bySlot[r.Slot], zoneSample{t: f, v: r.Value})
	}
	out := make([]flagGaugeSlot, 0, len(bySlot))
	for slot, ss := range bySlot {
		sort.SliceStable(ss, func(i, j int) bool { return ss[i].t < ss[j].t })
		out = append(out, flagGaugeSlot{slot: slot, samples: ss})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].slot < out[j].slot })
	return out
}

// flagGaugeCouple est un appariement CANDIDAT slot -> drapeau, avec de quoi le departager.
type flagGaugeCouple struct {
	flag   int
	slot   int
	share  float64
	inside int
}

// bindFlagGauges apparie AU PLUS UN slot par drapeau, et au plus un drapeau par slot.
//
// UN SLOT NE PEUT PAS SERVIR DEUX DRAPEAUX, ET DEUX SLOTS NE PEUVENT PAS SERVIR LE MEME : le jeu
// attache la propriete a l OBJET, et la mesure le confirme (un slot par drapeau sur les deux
// films). L appariement prend donc les couples par part DECROISSANTE, et un couple dont l un des
// deux cotes est deja pris est ecarte — la meme discipline que l appariement des zones.
func bindFlagGauges(slots []flagGaugeSlot, carries []FlagCarry) map[int]flagGaugeSlot {
	var cs []flagGaugeCouple
	for si, s := range slots {
		if len(s.samples) < flagReturnGaugeMinSamples {
			continue
		}
		for f := range carries {
			inside := flagSamplesInsideDrops(s.samples, carries[f].Spans)
			share := float64(inside) / float64(len(s.samples))
			if share < flagReturnGaugeMinShare {
				continue
			}
			cs = append(cs, flagGaugeCouple{flag: f, slot: si, share: share, inside: inside})
		}
	}
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].share != cs[j].share {
			return cs[i].share > cs[j].share
		}
		return cs[i].inside > cs[j].inside
	})
	out := map[int]flagGaugeSlot{}
	pris := map[int]bool{}
	for _, c := range cs {
		if _, deja := out[c.flag]; deja || pris[c.slot] {
			continue
		}
		out[c.flag] = slots[c.slot]
		pris[c.slot] = true
	}
	return out
}

// flagSamplesInsideDrops compte les echantillons qui tombent dans un intervalle `dropped` de ce
// drapeau. Les spans sont tries par T0 et disjoints : une recherche dichotomique suffit.
func flagSamplesInsideDrops(ss []zoneSample, spans []FlagSpan) int {
	var n int
	for _, s := range ss {
		i := sort.Search(len(spans), func(k int) bool { return spans[k].T1 >= s.t })
		if i < len(spans) && spans[i].State == FlagStateDropped && s.t >= spans[i].T0 {
			n++
		}
	}
	return n
}

// fillFlagReturnProgress pose la serie sur chaque intervalle `dropped`, ALLEGEE comme celle des
// zones et bornee a l intervalle.
//
// LA CLE EST ABSENTE QUAND LE FILM NE PORTE RIEN SUR CET INTERVALLE, et c est la meme regle que
// partout dans ce document : un tableau vide se lirait « la jauge est restee a zero », alors que
// l absence dit « le film n a rien emis ici ».
func fillFlagReturnProgress(spans []FlagSpan, ss []zoneSample, gap int, cov *FlagCarriesCoverage) {
	for i := range spans {
		if spans[i].State != FlagStateDropped {
			continue
		}
		pts, _ := appendGaugeThinned(nil, ss, spans[i].T0, spans[i].T1, gap)
		if len(pts) == 0 {
			continue
		}
		spans[i].ReturnProgress = pts
		if cov != nil {
			cov.GaugeSpans++
			cov.GaugePoints += len(pts)
		}
	}
}
