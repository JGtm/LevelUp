package replay

// attachement_phase0_bord_test.go — ITEM 0.3 : LE JOUEUR À BORD.
//
// L'ORACLE EST GÉOMÉTRIQUE, ET LE PLAN L'A ÉCRIT AVANT LA MESURE (décision 4(b)) : une
// période où la position d'un bipède reste à moins de 1,5 m de celle d'un véhicule pendant au
// moins 3 s est une période « à bord ». Aucun événement du film ne dit qu'un joueur monte
// dans un véhicule : ni le statborg, ni les événements nommés, ni le fil des morts. Deux
// nuages de positions décodés par deux chaînes indépendantes sont tout ce dont on dispose, et
// c'est assez pour poser la question.
//
// L'ORACLE PORTE EN LUI UNE CONTRADICTION, ET ELLE EST MESURÉE PLUTÔT QUE SUPPOSÉE. Le modèle
// du plan dit que l'enfant attaché NE RÉPLIQUE PLUS sa position monde. Si le modèle est vrai,
// un bipède à bord n'émet PLUS de position — et l'oracle de coïncidence ne peut rien voir par
// construction. Le second volet mesure donc exactement cela : les TROUS du flux de position
// d'un bipède, et si ces trous s'ouvrent près d'un véhicule. Les deux lectures se publient
// ensemble ; c'est leur combinaison qui dit si le modèle tient.
//
// LE TÉMOIN EST OBLIGATOIRE et il est construit pour être comparable : à chaque période « à
// bord », on substitue au véhicule observé un AUTRE véhicule du même match, tiré de façon
// déterministe. Même nombre de périodes, mêmes lectures, même tolérance.

import (
	"math"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Les seuils de la décision 4(b), écrits avant la mesure.
const (
	// attBordRayonM : distance en plan sous laquelle bipède et véhicule sont « au même
	// endroit ». La distance est prise EN PLAN pour la même raison qu'aux socles : la
	// hauteur d'un occupant et celle du repère du véhicule ne se réfèrent pas au même point.
	attBordRayonM = 1.5
	// attBordDureeMS : durée minimale d'une période pour compter comme un embarquement.
	attBordDureeMS = 3000
	// attBordTolMS : tolérance d'appariement entre le début d'une période et une lecture
	// d'i10 attachée. Deux paquets delta valent ~1 s (écart médian mesuré 0,5 s) ; on prend
	// large, le biais joue CONTRE le signal quand il manque, jamais en sa faveur.
	attBordTolMS = 2000
	// attTrouMS : durée minimale d'une interruption du flux de position d'un bipède pour
	// compter comme un trou. Même ordre que la période minimale.
	attTrouMS = 3000
)

// attPeriode est une période « à bord » telle que l'oracle géométrique la voit.
type attPeriode struct {
	BipedeSlot, VehiculeSlot uint32
	T0, T1                   uint64 // horloge du FILM, microsecondes
}

// TestAttachementPhase0Vehicules — ITEM 0.3.
func TestAttachementPhase0Vehicules(t *testing.T) {
	root := attRequireRoot(t)
	joues := 0
	for _, id := range attFilmsVehicules() {
		if _, ok := objOpenFilm(t, root, id); !ok {
			t.Logf("%s : film absent du cache — sauté", id)
			continue
		}
		joues++
		attVehiculesFilm(t, root, id)
	}
	if joues == 0 {
		t.Skipf("aucun film à véhicules dans le cache (%s=%q)", attFilmEnv, root)
	}
}

// attVehiculesFilm mesure UN film.
func attVehiculesFilm(t *testing.T, root, id string) {
	t.Helper()
	veh, bip, ok := attNuages(t, root, id)
	if !ok {
		return
	}
	t.Logf("%s : %d vies de véhicule (ti=%d) · %d échantillons de bipède · %d slots de bipède",
		id, len(veh), attVehiculeTI, len(bip), len(attSlotsBipede(bip)))
	attControleNuage(t, root, id, veh)
	attControleEmprises(t, id, veh, bip)
	periodes := attPeriodesABord(veh, bip)
	t.Logf("%s : %d périodes « à bord » (bipède à moins de %.1f m d'un véhicule pendant au "+
		"moins %d ms)", id, len(periodes), attBordRayonM, attBordDureeMS)
	attConfronteI10(t, root, id, periodes, veh)
	attLogTrous(t, id, veh, bip)
}

// attNuages décode les deux nuages de positions d'un film, aux bornes de sa carte.
func attNuages(t *testing.T, root, id string) ([]filmdec.ProjectileTrack, []filmdec.BipedPosition, bool) {
	t.Helper()
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := attBornes(t, root, id)
	if !ok {
		t.Logf("%s : bornes de carte indisponibles — item 0.3 non mesurable sur ce film", id)
		return nil, nil, false
	}
	dir := objChunkDir(root, id)
	veh, err := filmdec.ScanFilmWorldObjects(dir, &wr, int(attVehiculeTI))
	if err != nil {
		t.Logf("%s : balayage des véhicules ti=%d : %v", id, attVehiculeTI, err)
		return nil, nil, false
	}
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &wr
	bip, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("%s : balayage des bipèdes : %v", id, err)
		return nil, nil, false
	}
	return veh, bip, true
}

// attSlotsBipede rend les slots de bipède distincts d'un nuage.
func attSlotsBipede(bip []filmdec.BipedPosition) map[uint32]bool {
	out := map[uint32]bool{}
	for _, b := range bip {
		out[b.Slot] = true
	}
	return out
}

// attPeriodesABord construit les périodes de coïncidence prolongée.
//
// LA RÈGLE EST CELLE DU PLAN, appliquée telle quelle : pour chaque échantillon de bipède, on
// cherche le véhicule le plus proche À CET INSTANT (interpolation par le plus proche voisin
// temporel de sa trajectoire, sans extrapolation au-delà d'une seconde) ; les échantillons
// consécutifs qui restent sous le rayon avec LE MÊME véhicule forment une période.
func attPeriodesABord(veh []filmdec.ProjectileTrack, bip []filmdec.BipedPosition) []attPeriode {
	parBipede := map[uint32][]filmdec.BipedPosition{}
	for _, b := range bip {
		if b.HasWorld {
			parBipede[b.Slot] = append(parBipede[b.Slot], b)
		}
	}
	slots := make([]uint32, 0, len(parBipede))
	for s := range parBipede {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []attPeriode
	for _, s := range slots {
		ech := parBipede[s]
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		out = append(out, attPeriodesDuBipede(s, ech, veh)...)
	}
	return out
}

// attPeriodesDuBipede découpe la suite d'échantillons d'UN bipède en périodes « à bord ».
func attPeriodesDuBipede(slot uint32, ech []filmdec.BipedPosition,
	veh []filmdec.ProjectileTrack) []attPeriode {
	var out []attPeriode
	var cour attPeriode
	ouvert := false
	for _, e := range ech {
		v, ok := attVehiculeLePlusProche(e, veh)
		switch {
		case ok && ouvert && v == cour.VehiculeSlot:
			cour.T1 = e.TimestampUS
		case ok:
			if ouvert {
				out = attFermePeriode(out, cour)
			}
			cour, ouvert = attPeriode{BipedeSlot: slot, VehiculeSlot: v,
				T0: e.TimestampUS, T1: e.TimestampUS}, true
		default:
			if ouvert {
				out = attFermePeriode(out, cour)
			}
			ouvert = false
		}
	}
	if ouvert {
		out = attFermePeriode(out, cour)
	}
	return out
}

// attFermePeriode retient une période si elle atteint la durée minimale.
func attFermePeriode(out []attPeriode, p attPeriode) []attPeriode {
	if int64(p.T1-p.T0)/1000 >= attBordDureeMS {
		return append(out, p)
	}
	return out
}

// attExtrapolationUS borne l'écart temporel accepté entre un échantillon de bipède et
// l'échantillon de véhicule le plus proche. Au-delà, on ne compare pas deux positions
// simultanées mais deux positions distantes dans le temps — et l'oracle mesurerait le
// déplacement du véhicule, pas la présence du joueur à son bord.
const attExtrapolationUS = uint64(1_000_000)

// attVehiculeLePlusProche rend le slot du véhicule sous le rayon à cet instant, s'il y en a.
func attVehiculeLePlusProche(e filmdec.BipedPosition, veh []filmdec.ProjectileTrack) (uint32, bool) {
	best, found := uint32(0), false
	bestD := math.MaxFloat64
	for _, v := range veh {
		s, gap, ok := attEchantillonLePlusProche(v.Pts, e.TimestampUS)
		if !ok || gap > attExtrapolationUS {
			continue
		}
		d := math.Hypot(float64(e.X-s.X), float64(e.Y-s.Y))
		if d <= attBordRayonM && d < bestD {
			best, bestD, found = v.Slot, d, true
		}
	}
	return best, found
}

// attEchantillonLePlusProche rend l'échantillon de trajectoire le plus proche d'un instant,
// et l'écart. `ScanFilmWorldObjectsForBand` rend les points d'une vie DÉJÀ TRIÉS (tri total
// sur instant puis position) : la recherche dichotomique est donc licite ici.
func attEchantillonLePlusProche(pts []filmdec.ProjectileSample, atUS uint64) (
	filmdec.ProjectileSample, uint64, bool) {
	if len(pts) == 0 {
		return filmdec.ProjectileSample{}, 0, false
	}
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS >= atUS })
	best := i
	if i >= len(pts) {
		best = len(pts) - 1
	} else if i > 0 && attEcartUS(pts[i-1].TimestampUS, atUS) < attEcartUS(pts[i].TimestampUS, atUS) {
		best = i - 1
	}
	return pts[best], attEcartUS(pts[best].TimestampUS, atUS), true
}

// attConfronteI10 confronte les périodes « à bord » aux lectures d'i10 des bipèdes.
//
// SEUIL DU PLAN : au moins 90 % des périodes doivent porter, dans les `attBordTolMS` autour
// de leur début, une lecture d'i10 ATTACHÉE sur le slot du bipède dont le champ candidat
// désigne CE véhicule. Témoin : le même compte avec un autre véhicule du match.
func attConfronteI10(t *testing.T, root, id string, periodes []attPeriode,
	veh []filmdec.ProjectileTrack) {
	t.Helper()
	lectures, _ := attScanOf(t, root, id)
	parSlot := map[uint32][]attI10{}
	for _, l := range lectures {
		if l.TI == uint32(filmdec.BipedTypeIndex) {
			parSlot[l.Slot] = append(parSlot[l.Slot], l)
		}
	}
	autres := make([]uint32, 0, len(veh))
	for _, v := range veh {
		autres = append(autres, v.Slot)
	}
	sort.Slice(autres, func(i, j int) bool { return autres[i] < autres[j] })
	apparie, temoin, attachees := 0, 0, 0
	for i, p := range periodes {
		if attTrouveAttache(parSlot[p.BipedeSlot], p.T0, p.VehiculeSlot) {
			apparie++
		}
		if len(autres) > 0 {
			faux := autres[i%len(autres)]
			if faux == p.VehiculeSlot {
				faux = autres[(i+1)%len(autres)]
			}
			if faux != p.VehiculeSlot && attTrouveAttache(parSlot[p.BipedeSlot], p.T0, faux) {
				temoin++
			}
		}
		attachees += attCompteAttachees(parSlot[p.BipedeSlot], p.T0)
	}
	t.Logf("ITEM 0.3 %s — %d/%d périodes appariées à une lecture i10 attachée désignant LE "+
		"véhicule (%.1f %%, seuil %.0f %%) ; témoin (autre véhicule) %d (%.1f %%, seuil %.0f %%) ; "+
		"%d lectures attachées présentes dans les fenêtres, tolérance %d ms",
		id, apparie, len(periodes), 100*attPart(apparie, len(periodes)), 100*attSeuilTaux,
		temoin, 100*attPart(temoin, len(periodes)), 100*attSeuilTemoin, attachees, attBordTolMS)
}

// attTrouveAttache cherche une lecture attachée dont le champ candidat désigne `cible`, dans
// la fenêtre de tolérance autour de t0.
func attTrouveAttache(ls []attI10, t0 uint64, cible uint32) bool {
	for _, l := range ls {
		if !l.St.Attached || l.ParentSlot != cible {
			continue
		}
		if attEcartUS(l.TS, t0)/1000 <= attBordTolMS {
			return true
		}
	}
	return false
}

// attCompteAttachees compte les lectures attachées présentes dans la fenêtre de tolérance,
// quel que soit le champ — le dénominateur SANS lequel « 0 apparié » ne se distingue pas de
// « aucune lecture à apparier ».
func attCompteAttachees(ls []attI10, t0 uint64) int {
	n := 0
	for _, l := range ls {
		if l.St.Attached && attEcartUS(l.TS, t0)/1000 <= attBordTolMS {
			n++
		}
	}
	return n
}

// attEcartUS rend l'écart absolu entre deux instants microseconde.
func attEcartUS(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// attLogTrous mesure les INTERRUPTIONS du flux de position des bipèdes et leur voisinage.
//
// C'EST LE VOLET QUI ÉPROUVE LE MODÈLE DU PLAN, et il ne dépend d'aucun champ d'i10 : si un
// enfant attaché cesse de répliquer sa position, un embarquement se voit comme un TROU dont
// les deux bords sont près d'un véhicule. Si les trous ne sont pas plus près des véhicules
// que le reste du temps, c'est le MODÈLE qui ne tient pas ici — et c'est une information de
// même valeur que son contraire.
func attLogTrous(t *testing.T, id string, veh []filmdec.ProjectileTrack, bip []filmdec.BipedPosition) {
	t.Helper()
	parBipede := map[uint32][]filmdec.BipedPosition{}
	for _, b := range bip {
		if b.HasWorld {
			parBipede[b.Slot] = append(parBipede[b.Slot], b)
		}
	}
	trous, presVehicule := 0, 0
	for _, ech := range parBipede {
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		for i := 1; i < len(ech); i++ {
			if int64(ech[i].TimestampUS-ech[i-1].TimestampUS)/1000 < attTrouMS {
				continue
			}
			trous++
			if _, ok := attVehiculeLePlusProche(ech[i-1], veh); ok {
				presVehicule++
			}
		}
	}
	t.Logf("ITEM 0.3 %s — %d trous d'au moins %d ms dans le flux de position des bipèdes, "+
		"dont %d dont le DERNIER point avant le trou est à moins de %.1f m d'un véhicule "+
		"(%.1f %%)", id, trous, attTrouMS, presVehicule, attBordRayonM,
		100*attPart(presVehicule, trous))
}

// attEtalementSerre / attEtalementLarge — les deux bornes du contrôle d'étalement, reprises
// TELLES QUELLES de la refutation des armes au sol du 2026-08-12, pour que les deux mesures
// se comparent : une vie dont les points tiennent dans 0,5 u est une pose, une vie qui
// s'etale au-dela de 20 u n'est pas une trajectoire d'objet mais un melange de slots.
const (
	attEtalementSerre  = 0.5
	attEtalementLarge  = 20.0
	attMinEchantillons = 3
)

// attControleNuage juge le nuage de positions des vehicules AVANT de s'en servir.
//
// POURQUOI CE CONTROLE EST OBLIGATOIRE ICI. La position des objets du monde lue sur le chemin
// delta a deja ete REFUTEE une fois dans ce depot, pour les armes au sol : un record delta ne
// dit pas son archetype, la bande de slots comblee est contaminee par les archetypes voisins,
// et 62,4 % des slots s'etalaient au-dela de 20 u. Employer le meme decodeur sur `ti=40` sans
// refaire le controle serait reproduire l'erreur en changeant de numero.
//
// LE TEMOIN FANTOME passe par le MEME decodeur, sur une bande de MEME cardinalite faite de
// slots jamais vus porter cet archetype. S'il rend autant de vies et le meme etalement, le
// signal est sous le bruit — et l'oracle geometrique de l'item 0.3 ne mesure rien.
func attControleNuage(t *testing.T, root, id string, veh []filmdec.ProjectileTrack) {
	t.Helper()
	vus, autres := attBandesKeyframe(objChunkDir(root, id))
	serre, large, comptes := attEtalement(veh)
	t.Logf("%s : CONTROLE du nuage ti=%d — %d slots vus en image-cle, %d vies decodees, "+
		"%d vies a >= %d echantillons dont %d tiennent dans %.1f u (%.1f %%) et %d s'etalent "+
		"au-dela de %.0f u (%.1f %%)",
		id, attVehiculeTI, len(vus), len(veh), comptes, attMinEchantillons, serre,
		attEtalementSerre, 100*attPart(serre, comptes), large, attEtalementLarge,
		100*attPart(large, comptes))
	fantome := attBandeFantome(vus, autres)
	if len(fantome) == 0 {
		t.Logf("%s : aucun slot libre pour une bande fantome — controle temoin impossible", id)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := attBornes(t, root, id)
	if !ok {
		return
	}
	fveh, err := filmdec.ScanFilmWorldObjectsForBand(objChunkDir(root, id), &wr, fantome)
	if err != nil {
		t.Logf("%s : bande fantome : %v", id, err)
		return
	}
	fs, fl, fc := attEtalement(fveh)
	t.Logf("%s : TEMOIN FANTOME (%d slots jamais vus porter ti=%d) — %d vies, %d a >= %d "+
		"echantillons dont %d serrees (%.1f %%) et %d etalees (%.1f %%)",
		id, len(fantome), attVehiculeTI, len(fveh), fc, attMinEchantillons, fs,
		100*attPart(fs, fc), fl, 100*attPart(fl, fc))
}

// attBandesKeyframe releve, dans les images-cles, les slots vus porter `attVehiculeTI` et
// ceux vus porter un AUTRE archetype.
func attBandesKeyframe(dir string) (vus, autres map[uint32]bool) {
	vus, autres = map[uint32]bool{}, map[uint32]bool{}
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(p.Payload(data)) {
				if r.Slot < 0 {
					continue
				}
				if uint32(r.TI) == attVehiculeTI {
					vus[uint32(r.Slot)] = true
					continue
				}
				autres[uint32(r.Slot)] = true
			}
		}
	}
	return vus, autres
}

// attBandeFantome construit une bande de MEME cardinalite que la bande reelle, faite de slots
// jamais vus porter le moindre archetype en image-cle.
func attBandeFantome(vus, autres map[uint32]bool) map[uint32]bool {
	out := map[uint32]bool{}
	for s := uint32(0); s <= attSlotMask && len(out) < len(vus); s++ {
		if vus[s] || autres[s] {
			continue
		}
		out[s] = true
	}
	return out
}

// attEtalement rend, sur les vies d'au moins `attMinEchantillons` points, combien tiennent
// dans `attEtalementSerre` et combien depassent `attEtalementLarge` (diagonale en plan de
// leur boite englobante), plus le denominateur.
func attEtalement(tracks []filmdec.ProjectileTrack) (serre, large, comptes int) {
	for _, tr := range tracks {
		if len(tr.Pts) < attMinEchantillons {
			continue
		}
		comptes++
		minX, maxX := tr.Pts[0].X, tr.Pts[0].X
		minY, maxY := tr.Pts[0].Y, tr.Pts[0].Y
		for _, p := range tr.Pts {
			minX, maxX = min(minX, p.X), max(maxX, p.X)
			minY, maxY = min(minY, p.Y), max(maxY, p.Y)
		}
		d := math.Hypot(float64(maxX-minX), float64(maxY-minY))
		if d <= attEtalementSerre {
			serre++
		}
		if d > attEtalementLarge {
			large++
		}
	}
	return serre, large, comptes
}

// attControleEmprises confronte les EMPRISES des deux nuages.
//
// POURQUOI CETTE VERIFICATION PRECEDE TOUTE DISTANCE. Les deux chaines dequantifient avec des
// largeurs d'axe de sources DIFFERENTES : le bipede avec le decoupage DETECTE dans le flux
// (`DetectI0Layout`), l'objet du monde avec les largeurs du CATALOGUE de bornes. Le depot
// affirme que les deux coincident (accord 7 films sur 7) — mais l'affirmation vaut pour les
// films deja mesures, pas pour celui-ci. Si les deux nuages n'occupent pas le meme volume,
// une distance entre eux ne mesure pas une proximite mais un desaccord de repere, et l'oracle
// geometrique de l'item 0.3 est nul et non avenu AVANT d'avoir rien conclu.
func attControleEmprises(t *testing.T, id string, veh []filmdec.ProjectileTrack,
	bip []filmdec.BipedPosition) {
	t.Helper()
	var vb, bb attBoite
	for _, tr := range veh {
		for _, p := range tr.Pts {
			vb.ajoute(p.X, p.Y, p.Z)
		}
	}
	for _, b := range bip {
		if b.HasWorld {
			bb.ajoute(b.X, b.Y, b.Z)
		}
	}
	t.Logf("%s : EMPRISES — vehicules x[%.1f, %.1f] y[%.1f, %.1f] z[%.1f, %.1f] (%d points) ; "+
		"bipedes x[%.1f, %.1f] y[%.1f, %.1f] z[%.1f, %.1f] (%d points)",
		id, vb.minX, vb.maxX, vb.minY, vb.maxY, vb.minZ, vb.maxZ, vb.n,
		bb.minX, bb.maxX, bb.minY, bb.maxY, bb.minZ, bb.maxZ, bb.n)
	t.Logf("%s : part des points de vehicule DANS l'emprise des bipedes : %.1f %% "+
		"(un nuage qui ne recouvre pas l'autre est un desaccord de repere, pas une distance)",
		id, 100*attPartDansEmprise(veh, bb))
}

// attBoite est une boite englobante en cours de construction.
type attBoite struct {
	minX, maxX, minY, maxY, minZ, maxZ float32
	n                                  int
}

func (b *attBoite) ajoute(x, y, z float32) {
	if b.n == 0 {
		b.minX, b.maxX, b.minY, b.maxY, b.minZ, b.maxZ = x, x, y, y, z, z
	}
	b.minX, b.maxX = min(b.minX, x), max(b.maxX, x)
	b.minY, b.maxY = min(b.minY, y), max(b.maxY, y)
	b.minZ, b.maxZ = min(b.minZ, z), max(b.maxZ, z)
	b.n++
}

// attPartDansEmprise rend la part des points de trajectoire qui tombent dans une boite.
func attPartDansEmprise(tracks []filmdec.ProjectileTrack, b attBoite) float64 {
	dedans, total := 0, 0
	for _, tr := range tracks {
		for _, p := range tr.Pts {
			total++
			if p.X >= b.minX && p.X <= b.maxX && p.Y >= b.minY && p.Y <= b.maxY &&
				p.Z >= b.minZ && p.Z <= b.maxZ {
				dedans++
			}
		}
	}
	return attPart(dedans, total)
}
