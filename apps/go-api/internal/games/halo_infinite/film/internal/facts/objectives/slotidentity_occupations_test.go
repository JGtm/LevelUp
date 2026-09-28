package objectives

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// slotidentity_occupations_test.go — LE SIEGE RECYCLE (lot R1 du plan de suite d audit, constat C1
// du rapport G-corpus J11) : un slot statborg qui change d'occupant porte UN LIEN PAR OCCUPATION,
// et le pont s'abstient sur une occupation qu'il ne prouve pas.
//
// ROUGE OBSERVE AVANT LE LOT : `At(12, 10 000)` rendait le REMPLACANT — la serie publiee du slot
// (plus longue sous-suite non decroissante) garde le segment le plus long, et le lien resolu valait
// pour toute la manche. Meme forme que `bcb6d393` : le frag du premier occupant publie au nom d'un
// joueur entre dans la partie apres lui.

const (
	premierOccupant = "2535460750735339"
	remplacant      = "2535468064146356"
	voisin          = "2533274858283686"
	furtif          = "2535418713587213"
)

// siegeRecycle : une manche, le slot 12 tenu par `premierOccupant` (1 frag, 4 morts) puis, apres la
// retombee des compteurs a 40 000 ms, par `remplacant` (1 frag, 5 morts) ; le slot 10 tenu par
// `voisin` tout du long (3 morts). Le fil des morts nomme chaque mort.
func siegeRecycle() ([]types.StatRecord, []types.DeathInstant) {
	recs := []types.StatRecord{
		recKDA(10000, 12, 0, 1, 0, 0), recKDA(11000, 12, 0, 1, 1, 0), recKDA(20000, 12, 0, 1, 2, 0),
		recKDA(30000, 12, 0, 1, 3, 0), recKDA(35000, 12, 0, 1, 4, 0),
		recKDA(40000, 12, 0, 0, 0, 0), // retombee : le premier occupant quitte la partie
		recKDA(50000, 12, 0, 0, 1, 0), recKDA(60000, 12, 0, 0, 2, 0), recKDA(70000, 12, 0, 1, 2, 0),
		recKDA(80000, 12, 0, 1, 3, 0), recKDA(90000, 12, 0, 1, 4, 0), recKDA(95000, 12, 0, 1, 5, 0),
		recKDA(15000, 10, 0, 0, 1, 0), recKDA(25000, 10, 0, 0, 2, 0), recKDA(45000, 10, 0, 0, 3, 0),
	}
	var deaths []types.DeathInstant
	for _, t := range []int{11010, 20010, 30010, 35010} {
		deaths = append(deaths, types.DeathInstant{XUID: premierOccupant, TimeMS: t})
	}
	for _, t := range []int{50010, 60010, 80010, 90010, 95010} {
		deaths = append(deaths, types.DeathInstant{XUID: remplacant, TimeMS: t})
	}
	for _, t := range []int{15010, 25010, 45010} {
		deaths = append(deaths, types.DeathInstant{XUID: voisin, TimeMS: t})
	}
	return recs, deaths
}

// TestSiegeRecyclePorteUnLienParOccupation : chaque instant du slot recycle rend l'occupant de SON
// segment ; le slot voisin, sans retombee, garde son lien de manche entiere.
func TestSiegeRecyclePorteUnLienParOccupation(t *testing.T) {
	recs, deaths := siegeRecycle()
	id := ResolveRoundIdentity(recs, deaths, nil)
	for _, c := range []struct {
		slot, t int
		want    string
	}{
		{12, 10000, premierOccupant}, // son frag, avant la retombee
		{12, 39999, premierOccupant},
		{12, 40000, remplacant},
		{12, 70000, remplacant}, // le frag du remplacant
		{10, 10000, voisin},
		{10, 90000, voisin},
	} {
		if got := id.At(c.slot, c.t); got != c.want {
			t.Errorf("At(%d, %d) = %q, attendu %q", c.slot, c.t, got, c.want)
		}
	}
	occ := id.Occupations(0, 12)
	if len(occ) != 2 || !occ[0].OpenFrom || occ[0].ToMS != 40000 || occ[1].FromMS != 40000 || !occ[1].OpenTo {
		t.Fatalf("occupations du slot 12 : %+v, attendu deux occupations bornees a 40 000 ms", occ)
	}
	if occ[0].Origin != OriginDeathInstants || occ[1].Origin != OriginDeathInstants {
		t.Errorf("les deux occupations sont nommees par les instants de mort : %+v", occ)
	}
	if id.Occupations(0, 10) != nil {
		t.Error("un slot sans retombee ne porte pas d'occupation : son lien vaut pour la manche")
	}
	if id.UnprovenOccupantAt(12, 10000) || id.UnprovenOccupantAt(12, 70000) {
		t.Error("les deux occupations sont prouvees : aucune abstention")
	}
}

// TestSiegeRecycleSAbstientSurLOccupationNonProuvee : un occupant intermediaire qui meurt UNE fois ne
// se nomme pas (trois coincidences au moins) — le pont s'abstient sur son intervalle, et ne le
// donne ni au premier occupant ni au dernier.
func TestSiegeRecycleSAbstientSurLOccupationNonProuvee(t *testing.T) {
	recs, deaths := siegeRecycle()
	// Le remplacant ne prend le siege qu'a 60 000 : entre 40 000 et 60 000, `furtif` l'occupe
	// (une mort), puis le siege retombe une seconde fois.
	var out []types.StatRecord
	for _, r := range recs {
		if r.Slot == 12 && r.TimeMS >= 40000 {
			continue
		}
		out = append(out, r)
	}
	out = append(out,
		recKDA(40000, 12, 0, 0, 0, 0), recKDA(45500, 12, 0, 1, 1, 0), // furtif : un frag, une mort
		recKDA(55000, 12, 0, 0, 0, 0), // seconde retombee
		recKDA(60000, 12, 0, 0, 1, 0), recKDA(70000, 12, 0, 1, 1, 0), recKDA(80000, 12, 0, 1, 2, 0),
		recKDA(90000, 12, 0, 1, 3, 0), recKDA(95000, 12, 0, 1, 4, 0),
	)
	deaths = append(sansMortsDe(deaths, remplacant), types.DeathInstant{XUID: furtif, TimeMS: 45510})
	for _, t := range []int{60010, 80010, 90010, 95010} {
		deaths = append(deaths, types.DeathInstant{XUID: remplacant, TimeMS: t})
	}
	id := ResolveRoundIdentity(sortedByTime(out), deaths, nil)
	if got := id.At(12, 50000); got != "" {
		t.Errorf("At(12, 50 000) = %q : l'occupant intermediaire n'est pas prouve, le pont doit s'abstenir", got)
	}
	if !id.UnprovenOccupantAt(12, 50000) {
		t.Error("l'abstention sur un siege recycle doit se dire (UnprovenOccupantAt)")
	}
	if got := id.At(12, 10000); got != premierOccupant {
		t.Errorf("At(12, 10 000) = %q, attendu %q", got, premierOccupant)
	}
	if got := id.At(12, 70000); got != remplacant {
		t.Errorf("At(12, 70 000) = %q, attendu %q", got, remplacant)
	}
}

// TestRetombeeSansRepriseNeCoupePas : une retombee que rien ne suit (fin de match) et une retombee
// que l'emission suivante PROLONGE (valeurs d'avant reprises) ne changent pas le siege — le lien de
// la manche entiere est garde, octet pour octet.
func TestRetombeeSansRepriseNeCoupePas(t *testing.T) {
	recs := []types.StatRecord{
		recKDA(10000, 12, 0, 1, 1, 0), recKDA(20000, 12, 0, 1, 2, 0),
		recKDA(25000, 12, 0, 0, 0, 0), // emission a zero ...
		recKDA(30000, 12, 0, 1, 3, 0), // ... que la suivante prolonge
		recKDA(40000, 12, 0, 1, 4, 0),
		recKDA(99000, 12, 0, 0, 0, 0), // fin de match : aucune reprise
	}
	var deaths []types.DeathInstant
	for _, t := range []int{10010, 20010, 30010, 40010} {
		deaths = append(deaths, types.DeathInstant{XUID: premierOccupant, TimeMS: t})
	}
	id := ResolveRoundIdentity(recs, deaths, nil)
	if occ := id.Occupations(0, 12); occ != nil {
		t.Fatalf("aucun changement de siege attendu, occupations %+v", occ)
	}
	if got := id.At(12, 99500); got != premierOccupant {
		t.Errorf("At(12, 99 500) = %q, attendu le lien de manche entiere %q", got, premierOccupant)
	}
}

// TestLeTripletNommeLeDernierOccupant : le triplet de la feuille (totaux FINAUX) nomme la DERNIERE
// occupation d'un siege recycle quand les morts ne la nomment pas — jamais une occupation
// anterieure.
func TestLeTripletNommeLeDernierOccupant(t *testing.T) {
	recs := []types.StatRecord{
		recKDA(10000, 12, 0, 1, 1, 0), recKDA(20000, 12, 0, 1, 2, 0), recKDA(30000, 12, 0, 2, 3, 0),
		recKDA(40000, 12, 0, 0, 0, 0),
		recKDA(50000, 12, 0, 1, 0, 0), recKDA(60000, 12, 0, 2, 1, 0),
		recKDA(15000, 10, 0, 0, 1, 0), recKDA(25000, 10, 0, 0, 2, 0), recKDA(45000, 10, 0, 0, 3, 0),
	}
	deaths := []types.DeathInstant{
		{XUID: premierOccupant, TimeMS: 10010}, {XUID: premierOccupant, TimeMS: 20010},
		{XUID: premierOccupant, TimeMS: 30010}, {XUID: remplacant, TimeMS: 60010},
		{XUID: voisin, TimeMS: 15010}, {XUID: voisin, TimeMS: 25010}, {XUID: voisin, TimeMS: 45010},
	}
	lines := []types.PlayerLine{
		{XUID: premierOccupant, Kills: 2, Deaths: 3}, {XUID: remplacant, Kills: 2, Deaths: 1},
		{XUID: voisin, Deaths: 3},
	}
	id := ResolveRoundIdentity(sortedByTime(recs), deaths, nil)
	if got := id.At(12, 55000); got != "" {
		t.Fatalf("avant la feuille, le remplacant (une mort) n'est pas prouve : At = %q", got)
	}
	complet := id.CompletedByLines(sortedByTime(recs), lines)
	if got := complet.At(12, 55000); got != remplacant {
		t.Errorf("At(12, 55 000) = %q, attendu %q (triplet final = dernier occupant)", got, remplacant)
	}
	if got := complet.At(12, 15000); got != premierOccupant {
		t.Errorf("At(12, 15 000) = %q, attendu %q", got, premierOccupant)
	}
	if got := id.At(12, 55000); got != "" {
		t.Error("la completion ne modifie jamais l'identite de son appelant")
	}
	occ := complet.Occupations(0, 12)
	if len(occ) == 0 || occ[len(occ)-1].Origin != OriginSheetTriplet {
		t.Errorf("la derniere occupation est nommee par le triplet : %+v", occ)
	}
}

// sortedByTime ordonne des enregistrements par instant, comme les rend le balayage.
func sortedByTime(recs []types.StatRecord) []types.StatRecord {
	out := append([]types.StatRecord(nil), recs...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].TimeMS < out[j-1].TimeMS; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// sansMortsDe retire du fil les morts d'un joueur.
func sansMortsDe(deaths []types.DeathInstant, xuid string) []types.DeathInstant {
	var out []types.DeathInstant
	for _, d := range deaths {
		if d.XUID != xuid {
			out = append(out, d)
		}
	}
	return out
}
