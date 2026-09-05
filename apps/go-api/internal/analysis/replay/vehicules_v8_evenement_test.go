package replay

// vehicules_v8_evenement_test.go — INSTRUMENT (lot V8) : LE VEHICULE D UN EPISODE, RESOLU PAR
// L EVENEMENT PLUTOT QUE PAR LA GEOMETRIE. AVANT / APRES, ET LES DESACCORDS UN PAR UN.
//
// CE QU IL MESURE, sur le MEME contexte de production que `TestV6EtatsOccupation` (socle V4) :
//
//	AVANT — le rattachement par la seule GEOMETRIE (deux ancres, rayon 3 m) : c est le 48 / 49
//	  du lot V6, re-mesure ici pour que l apres lui soit comparable ;
//	APRES — le rattachement par le NOM porte par la sortie, la geometrie en repli ;
//	L ACCORD des deux voies episode par episode, et le DEPOUILLEMENT de chaque desaccord : a
//	  quelle distance l ancre de debarquement est-elle du vehicule NOMME, et de celui que la
//	  distance designait ? C est la seule facon de dire QUI a raison ;
//	L AMBIGUITE GEOMETRIQUE (un SECOND vehicule sous le rayon a l ancre) et sa disparition sur
//	  les episodes nommes.
//
// LE TEMOIN, ecrit avant la mesure : PAR PERMUTATION. Le vehicule nomme est remplace par celui de
// l episode NOMME SUIVANT du meme film. L accord avec la geometrie doit s effondrer ; s il ne
// s effondre pas, c est que « nommer » ne vaut pas mieux que « tirer au sort dans la bande ».
//
// LECTURE SEULE, garde V4_ROOT / V4_FILMS :
//
//	CGO_ENABLED=0 V4_ROOT=<depot>/data/cache \
//	  V4_FILMS="0d76e8f1:Behemoth,21468645:Behemoth,4898d586:Behemoth,a89a3d23:Behemoth,fccc61cd:Launch Site" \
//	  go test ./internal/analysis/replay/ -run TestV8 -v -timeout 120m

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// v8Bilan accumule les comptes de tous les films.
type v8Bilan struct {
	episodes, nommes                        int
	avantGeom, apresTotal                   int
	parEvenement, parEvenementProche        int
	parGeometrie, perdus                    int
	gagnes, perdusParRapportAvant           int
	lesDeuxRepondent, accord, desaccord     int
	permAccord, nommeMuet                   int
	ambiguGeom, ambiguGeomNomme             int
	terminaux, terminauxNommes              int
	publiesAvant, publiesApres              int
	ridesAvant, ridesApres                  int
	nommesAvant, nommesApres, siegesApresNb int
	ambigusAvant, ambigusApres              int
	// parFamille ventile accord / desaccord par famille de chassis du vehicule que la GEOMETRIE
	// designe — c est lui qui porte le chassis, la vie nommee par l evenement etant parfois
	// MUETTE. [0] = accord, [1] = desaccord.
	parFamille map[string][2]int
}

// TestV8VehiculeParEvenement — LE BILAN. Une section par film, un total.
func TestV8VehiculeParEvenement(t *testing.T) {
	root := v4Root(t)
	tot := v8Bilan{parFamille: map[string][2]int{}}
	for _, f := range v4Corpus(t) {
		v8UnFilm(t, root, f, &tot)
	}
	t.Logf("== V8 TOTAL (%d episodes d evenement) ==", tot.episodes)
	t.Logf("  nommes par la sortie : %s", v8Pc(tot.nommes, tot.episodes))
	t.Logf("  AVANT (geometrie seule)  : rattaches %s", v8Pc(tot.avantGeom, tot.episodes))
	t.Logf("  APRES (evenement d abord): rattaches %s — par l evenement %d · par la vie la plus"+
		" proche %d · par la geometrie %d · PERDUS %d",
		v8Pc(tot.apresTotal, tot.episodes), tot.parEvenement, tot.parEvenementProche,
		tot.parGeometrie, tot.perdus)
	t.Logf("  DELTA : gagnes (l evenement rattache la ou la geometrie echouait) %d · perdus %d",
		tot.gagnes, tot.perdusParRapportAvant)
	t.Logf("  ACCORD : les deux voies repondent %d fois — MEME vie %d · DESACCORD %d "+
		"(TEMOIN PERMUTATION : meme vie %d) · dont vie nommee MUETTE (non dessinable) %d",
		tot.lesDeuxRepondent, tot.accord, tot.desaccord, tot.permAccord, tot.nommeMuet)
	t.Logf("  AMBIGUITE GEOMETRIQUE (2e vehicule sous %.1f m a l ancre) : %d episodes, dont %d"+
		" sont NOMMES par leur sortie (l ambiguite y tombe)",
		vehicleEventAnchorRadiusM, tot.ambiguGeom, tot.ambiguGeomNomme)
	for _, f := range v8FamillesTriees(tot.parFamille) {
		t.Logf("  famille %-10q : accord %d · DESACCORD %d", f, tot.parFamille[f][0],
			tot.parFamille[f][1])
	}
	t.Logf("  SILENCES TERMINAUX : %d, dont nommes %d (un embarquement ne nomme pas son vehicule)",
		tot.terminaux, tot.terminauxNommes)
	t.Logf("  ARTEFACT : vies publiees %d -> %d · episodes %d -> %d · occupants nommes %d -> %d"+
		" · episodes qui se chevauchent %d -> %d · avec siege (apres) %d",
		tot.publiesAvant, tot.publiesApres, tot.ridesAvant, tot.ridesApres,
		tot.nommesAvant, tot.nommesApres, tot.ambigusAvant, tot.ambigusApres, tot.siegesApresNb)
}

// v8Pc formate « x (p %) ».
func v8Pc(x, n int) string {
	if n == 0 {
		return fmt.Sprintf("%d (-)", x)
	}
	return fmt.Sprintf("%d/%d (%.1f %%)", x, n, 100*float64(x)/float64(n))
}

// v8FamillesTriees rend les familles d une ventilation, triees : la sortie est deterministe.
func v8FamillesTriees(m map[string][2]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// v8UnFilm depouille un film.
func v8UnFilm(t *testing.T, root string, f v0Film, tot *v8Bilan) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	ctx, ok := v4Decode(t, root, f)
	if !ok {
		return
	}
	in := vehicleRideInputs{
		vehBySlot: ctx.vehBySlot, bipeds: ctx.bip, events: ctx.scan.Events,
		own: ctx.own, lives: ctx.lives, clock: ctx.clock,
		drawable: vehicleDrawableLives(ctx.lives, ctx.spawns, ctx.vehBySlot),
	}
	boards, exits := vehicleEventsByOccupant(ctx.scan.Events)
	bySlot := vehiclePositionsBySlot(ctx.bip)
	eps := vehicleEventEpisodes(boards, exits, bySlot)
	t.Logf("== V8 %s (%s) — %d episodes d evenement ==", f.ID, f.Carte, len(eps))
	v8Muettes(t, ctx)
	v8Episodes(t, eps, bySlot, in, ctx.spawns, tot)
	v8Artefact(t, ctx, tot)
}

// v8Muettes recense les vies `ti=40` MUETTES — ni echantillon de position, ni record de creation.
// Ce sont elles que le desaccord met en cause : la sortie NOMME une vie muette la ou la geometrie
// designe une vie voisine, pleine, du meme recensement. L instrument dit s il existe un VOISIN
// immediat (slot +/- 1, meme generation, MEME fenetre de recensement) qui, lui, porte une
// trajectoire — c est-a-dire si les deux entites sont les deux faces d un meme vehicule.
func v8Muettes(t *testing.T, ctx v4Ctx) {
	t.Helper()
	pleine := map[filmdec.EquipmentLifeKey]bool{}
	for _, l := range ctx.lives {
		_, hasSpawn := ctx.spawns[l.key]
		if hasSpawn || len(ctx.vehBySlot[l.key.Slot]) > 0 {
			pleine[l.key] = true
		}
	}
	muettes, avecVoisin := 0, 0
	for _, l := range ctx.lives {
		if pleine[l.key] {
			continue
		}
		muettes++
		voisins := ""
		for _, d := range []int{1, -1} {
			v := filmdec.EquipmentLifeKey{Slot: uint32(int(l.key.Slot) + d), Gen: l.key.Gen}
			for _, o := range ctx.lives {
				if o.key != v || !pleine[v] {
					continue
				}
				meme := o.loUS == l.loUS && o.hiUS == l.hiUS
				voisins += fmt.Sprintf(" [%+d -> slot %d, %d echantillons, memeFenetre=%v, %s]",
					d, v.Slot, len(ctx.vehBySlot[v.Slot]), meme, v8Famille(ctx, v))
				if meme {
					avecVoisin++
				}
			}
		}
		if voisins == "" {
			voisins = " AUCUN voisin plein"
		}
		t.Logf("  VIE MUETTE slot %d gen %d fenetre [%d..%d] ·%s",
			l.key.Slot, l.key.Gen, l.loUS, l.hiUS, voisins)
	}
	t.Logf("  vies recensees %d — muettes (ni echantillon ni naissance) %d, dont %d ont un voisin"+
		" immediat PLEIN a la MEME fenetre", len(ctx.lives), muettes, avecVoisin)
}

// v8Famille rend la famille de chassis d une vie, ou « - ».
func v8Famille(ctx v4Ctx, key filmdec.EquipmentLifeKey) string {
	sp, ok := ctx.spawns[key]
	if !ok || !sp.MPPPresent[filmdec.MPPWord32] {
		return "famille -"
	}
	return "famille " + vehicleFamilyOf(uint32(sp.MPPVal[filmdec.MPPWord32]))
}

// v8Episodes compare les deux voies, episode par episode.
func v8Episodes(
	t *testing.T, eps []vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition,
	in vehicleRideInputs, spawns map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation,
	tot *v8Bilan,
) {
	t.Helper()
	nommes := v8SlotsNommes(eps)
	for i, ep := range eps {
		tot.episodes++
		if ep.openEnd {
			tot.terminaux++
			if ep.vehValid {
				tot.terminauxNommes++
			}
		}
		if ep.vehValid {
			tot.nommes++
		}
		geo, geoOK := vehicleLifeFromGeometry(ep, bySlot[ep.slot], in)
		// DEUX LECTURES : ce que l evenement DIT (resolution nue) et ce que la PRODUCTION en
		// retient (le nom, filtre par la publiabilite de la vie nommee).
		evLife, evSrc := vehicleLifeNamedByEvent(ep, in.lives)
		_, src := vehicleLifeFromEvent(ep, in)
		if geoOK {
			tot.avantGeom++
		}
		v8CompteApres(src, geoOK, tot)
		if n := v8CountWithinAnchor(ep, bySlot, in); n > 1 {
			tot.ambiguGeom++
			if ep.vehValid {
				tot.ambiguGeomNomme++
			}
		}
		if evSrc == vehicleResolvedNone || !geoOK {
			continue
		}
		tot.lesDeuxRepondent++
		if !in.drawable[evLife.key] {
			tot.nommeMuet++
		}
		fam := "-"
		if sp, ok := spawns[geo.key]; ok && sp.MPPPresent[filmdec.MPPWord32] {
			fam = vehicleFamilyOf(uint32(sp.MPPVal[filmdec.MPPWord32]))
		}
		c := tot.parFamille[fam]
		if evLife.key == geo.key {
			tot.accord++
			c[0]++
		} else {
			tot.desaccord++
			c[1]++
			v8Desaccord(t, ep, evLife, geo, bySlot, in, spawns)
		}
		tot.parFamille[fam] = c
		// TEMOIN : le vehicule de l episode NOMME suivant, au meme instant.
		if perm, ok := v8Permute(ep, nommes, i); ok {
			if l, s := vehicleLifeNamedByEvent(perm, in.lives); s != vehicleResolvedNone &&
				l.key == geo.key {
				tot.permAccord++
			}
		}
	}
}

// v8CompteApres ventile le rattachement de la voie NOUVELLE et son delta a l ancienne.
func v8CompteApres(src vehicleResolvedBy, geoOK bool, tot *v8Bilan) {
	switch src {
	case vehicleResolvedByEvent:
		tot.parEvenement++
	case vehicleResolvedByEventNearest:
		tot.parEvenementProche++
	default:
		if geoOK {
			tot.parGeometrie++
		}
	}
	apres := src != vehicleResolvedNone || geoOK
	if apres {
		tot.apresTotal++
	}
	switch {
	case apres && !geoOK:
		tot.gagnes++
	case !apres && geoOK:
		tot.perdusParRapportAvant++
	case !apres:
		tot.perdus++
	}
}

// v8SlotsNommes rend les slots de vehicule nommes par les episodes, dans l ordre : la matiere du
// temoin par permutation.
func v8SlotsNommes(eps []vehicleEpisode) []uint32 {
	var out []uint32
	for _, ep := range eps {
		if ep.vehValid {
			out = append(out, ep.vehSlot)
		}
	}
	return out
}

// v8Permute rend l episode dont le vehicule a ete remplace par celui du suivant de la liste.
func v8Permute(ep vehicleEpisode, nommes []uint32, i int) (vehicleEpisode, bool) {
	if !ep.vehValid || len(nommes) < 2 {
		return ep, false
	}
	autre := nommes[(i+1)%len(nommes)]
	if autre == ep.vehSlot {
		autre = nommes[(i+2)%len(nommes)]
	}
	if autre == ep.vehSlot {
		return ep, false
	}
	ep.vehSlot = autre
	return ep, true
}

// v8Desaccord depouille UN desaccord : les deux vies candidates — leur chassis, leur famille,
// leur naissance, leur nuage — et la distance de chaque ancre a chacune d elles. L ancre de FIN
// (le premier point replique APRES la sortie) est la piece qui tranche : on descend A COTE du
// vehicule qu on quitte.
func v8Desaccord(
	t *testing.T, ep vehicleEpisode, ev, geo vehicleLife,
	bySlot map[uint32][]filmdec.BipedPosition, in vehicleRideInputs,
	spawns map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation,
) {
	t.Helper()
	pts := bySlot[ep.slot]
	a0, h0 := vehicleAnchorAt(pts, ep.startUS, false)
	a1, h1 := vehicleAnchorAt(pts, ep.endUS, true)
	siege := "nil"
	if ep.seat != nil {
		siege = fmt.Sprintf("%d", *ep.seat)
	}
	t.Logf("  DESACCORD occupant %d siege %s [%d..%d us]", ep.slot, siege, ep.startUS, ep.endUS)
	t.Logf("     EVENEMENT -> %s", v8Vie(ev, in, spawns))
	t.Logf("     GEOMETRIE -> %s", v8Vie(geo, in, spawns))
	t.Logf("     ancre de DEBUT : evenement %s · geometrie %s",
		v8DistTo(a0, h0, ev, in, spawns), v8DistTo(a0, h0, geo, in, spawns))
	t.Logf("     ancre de FIN   : evenement %s · geometrie %s",
		v8DistTo(a1, h1 && !ep.openEnd, ev, in, spawns),
		v8DistTo(a1, h1 && !ep.openEnd, geo, in, spawns))
}

// v8Vie decrit une vie candidate : identite, chassis, famille, nuage, fenetre, naissance.
func v8Vie(
	l vehicleLife, in vehicleRideInputs,
	spawns map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation,
) string {
	chassis, famille, naissance := "-", "-", "aucune"
	if sp, ok := spawns[l.key]; ok {
		if sp.MPPPresent[filmdec.MPPWord32] {
			id := uint32(sp.MPPVal[filmdec.MPPWord32])
			chassis, famille = formatChassisID(id), vehicleFamilyOf(id)
		}
		naissance = fmt.Sprintf("(%.1f, %.1f)", sp.X, sp.Y)
	}
	return fmt.Sprintf("vie (slot %d, gen %d) chassis %s famille %q · %d echantillons · "+
		"fenetre [%d..%d] · naissance %s", l.key.Slot, l.key.Gen, chassis, famille,
		len(in.vehBySlot[l.key.Slot]), l.loUS, l.hiUS, naissance)
}

// v8DistTo rend la distance EN PLAN d une ancre a une vie de vehicule, a la regle de fraicheur de
// la production (1 s). A DEFAUT D ECHANTILLON, la distance a la NAISSANCE est rendue : une vie
// qui ne replique jamais sa position (objet attache, vehicule jamais conduit) n a que celle-la, et
// la taire ferait passer pour « introuvable » un vehicule dont on sait exactement ou il est.
func v8DistTo(
	e filmdec.BipedPosition, has bool, l vehicleLife, in vehicleRideInputs,
	spawns map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation,
) string {
	if !has {
		return "pas d ancre"
	}
	if p, gap, ok := vehicleSampleNear(in.vehBySlot[l.key.Slot], e.TimestampUS); ok {
		return fmt.Sprintf("%.1f m (echantillon a %.1f s)", planDist(e.X, e.Y, p.X, p.Y),
			float64(gap)/1e6)
	}
	sp, ok := spawns[l.key]
	if !ok {
		return "aucun echantillon, aucune naissance"
	}
	return fmt.Sprintf("%.1f m (NAISSANCE, aucun echantillon)", planDist(e.X, e.Y, sp.X, sp.Y))
}

// v8CountWithinAnchor compte les vehicules FRAIS sous le rayon de l ancre d evenement — la mesure
// d AMBIGUITE du lot V6, reprise a l identique.
func v8CountWithinAnchor(
	ep vehicleEpisode, bySlot map[uint32][]filmdec.BipedPosition, in vehicleRideInputs,
) int {
	pts := bySlot[ep.slot]
	a0, h0 := vehicleAnchorAt(pts, ep.startUS, false)
	n := v6CountWithin(a0, h0, in, vehicleEventAnchorRadiusM)
	if n == 0 && !ep.openEnd {
		a1, h1 := vehicleAnchorAt(pts, ep.endUS, true)
		n = v6CountWithin(a1, h1, in, vehicleEventAnchorRadiusM)
	}
	return n
}

// v8Artefact compte ce que le calque PUBLIE, dans les deux regimes. Le regime AVANT est obtenu en
// effacant la reference de vehicule des evenements : un seul parametre change, tout le reste du
// contexte est identique.
func v8Artefact(t *testing.T, ctx v4Ctx, tot *v8Bilan) {
	t.Helper()
	avant := ctx.scan
	avant.Events = v8SansReference(ctx.scan.Events)
	_, ca, _ := buildVehicleTracks(avant, ctx.bip, ctx.own, ctx.clock)
	_, cb, st := buildVehicleTracks(ctx.scan, ctx.bip, ctx.own, ctx.clock)
	t.Logf("  ARTEFACT avant/apres : vies publiees %d -> %d · episodes %d -> %d · nommes %d -> %d"+
		" · sieges %d -> %d · AMBIGUS (episodes qui se chevauchent) %d -> %d",
		ca.Published, cb.Published, ca.Rides, cb.Rides, ca.RidesNamed, cb.RidesNamed,
		ca.RidesWithSeat, cb.RidesWithSeat, ca.Ambiguous, cb.Ambiguous)
	t.Logf("  RATTACHEMENT (journal de production) : episodes %d · nommes %d · par evenement %d ·"+
		" par vie la plus proche %d · par geometrie %d · non rattaches %d · repli %d",
		st.episodes, st.nommes, st.parEvenement, st.parEvenementProche, st.parGeometrie,
		st.perdus, st.repli)
	tot.publiesAvant += ca.Published
	tot.publiesApres += cb.Published
	tot.ridesAvant += ca.Rides
	tot.ridesApres += cb.Rides
	tot.nommesAvant += ca.RidesNamed
	tot.nommesApres += cb.RidesNamed
	tot.siegesApresNb += cb.RidesWithSeat
	tot.ambigusAvant += ca.Ambiguous
	tot.ambigusApres += cb.Ambiguous
}

// v8SansReference rend les memes evenements, prives de leur reference de vehicule : le regime
// D AVANT le lot V8, ou seule la geometrie pouvait repondre.
func v8SansReference(evs []filmdec.VehicleEvent) []filmdec.VehicleEvent {
	out := make([]filmdec.VehicleEvent, len(evs))
	copy(out, evs)
	for i := range out {
		out[i].VehicleSlot, out[i].VehicleSlotValid, out[i].VehicleGen = 0, false, 0
	}
	return out
}
