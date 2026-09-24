package replay

// positions_porte_vehicules_test.go — la porte des positions de vehicule (lot M1 des retours du
// rejeu, 2026-09-23) : les replis F-1 (emprise) et F-2 (silence avec deplacement), et la lacune
// publiee `VehicleSample.G`. Les temoins de l annexe `RAPPORT_positions_limbe.md` sont rejoues en
// fixtures, avec leurs deux negatifs : la chute continue (Behemoth) et le vehicule gare.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// porteVieVehicule assemble UNE vie de vehicule recensee de 2 s a la fin du film (recensement toutes
// les 20 s sur deux minutes), avec sa naissance a 1,5 s et les echantillons donnes.
func porteVieVehicule(t *testing.T, pos []grammar.BipedPosition) (VehicleTrack, VehicleCoverage) {
	t.Helper()
	key := types.EquipmentLifeKey{Slot: 770, Gen: 1}
	times := []uint64{2_000_000, 22_000_000, 42_000_000, 62_000_000, 82_000_000, 102_000_000}
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: vehKeyframes(times, key, times),
		Creations: []types.EquipmentCreation{vehCreation(key, 1_500_000, -42.07, -21.42, vehChassisKnown)},
		Positions: pos,
	}
	got, cov, _ := buildVehicleTracks(scan, nil, IdentityRegistry{}, vehClock())
	if len(got) != 1 {
		t.Fatalf("vies publiees = %d, attendu 1", len(got))
	}
	return got[0], cov
}

// porteVehSejour rend des echantillons immobiles en (x, y), un toutes les 100 ms de `deUS` a `aUS`.
func porteVehSejour(deUS, aUS uint64, x, y float32) []grammar.BipedPosition {
	var out []grammar.BipedPosition
	for t := deUS; t <= aUS; t += 100_000 {
		out = append(out, vehPos(770, t, x, y))
	}
	return out
}

// porteXs rend les X publies de la trajectoire.
func porteXs(tr VehicleTrack) map[float32]int {
	out := map[float32]int{}
	for _, s := range tr.Samples {
		out[s.X]++
	}
	return out
}

// F-2 — le temoin du Mongoose de 81c02726 (slot 770) : immobile au point d apparition, UN
// echantillon 70 s plus tard a 193 m, puis de retour au point d apparition. L aller-retour est
// ecarte ; les deux sejours qui s accordent restent, et le retour porte la lacune.
func TestPorteVehiculeAllerRetourAuTraversDUnSilenceEcarte(t *testing.T) {
	pos := porteVehSejour(3_000_000, 4_000_000, -42.07, -21.42)
	pos = append(pos, vehPos(770, 74_600_000, -201.16, 88.24))
	pos = append(pos, porteVehSejour(79_800_000, 80_500_000, -42.07, -21.42)...)
	tr, cov := porteVieVehicule(t, pos)
	if porteXs(tr)[-201.16] != 0 {
		t.Error("l echantillon de l aller-retour est encore publie")
	}
	if cov.EchantillonsAuTraversDUnSilence != 1 {
		t.Errorf("echantillonsAuTraversDUnSilence = %d, attendu 1", cov.EchantillonsAuTraversDUnSilence)
	}
	var retour VehicleSample
	for _, s := range tr.Samples {
		if s.T == vehClock().frame(79_800_000) {
			retour = s
		}
	}
	if retour.G != 75_800 {
		t.Errorf("lacune portee par le retour = %d ms, attendu 75800 (4,0 s -> 79,8 s)", retour.G)
	}
}

// F-2 — le DERNIER echantillon d une vie, atteint a travers un silence avec un deplacement
// (879a4dba, slot 821) : ecarte.
func TestPorteVehiculeDernierEchantillonAberrantEcarte(t *testing.T) {
	pos := porteVehSejour(3_000_000, 6_000_000, 10, 10)
	pos = append(pos, vehPos(770, 40_000_000, 217.3, -200.53))
	tr, cov := porteVieVehicule(t, pos)
	if porteXs(tr)[217.3] != 0 || cov.EchantillonsAuTraversDUnSilence != 1 {
		t.Errorf("dernier echantillon aberrant : publie %d fois, compte %d ; attendu 0 et 1",
			porteXs(tr)[217.3], cov.EchantillonsAuTraversDUnSilence)
	}
}

// F-2 — le PREMIER echantillon d une vie, quitte a travers un silence avec un deplacement :
// ecarte, et le sejour qui suit reste entier.
func TestPorteVehiculePremierEchantillonAberrantEcarte(t *testing.T) {
	pos := []grammar.BipedPosition{vehPos(770, 2_500_000, -6.79, 217.3)}
	pos = append(pos, porteVehSejour(20_000_000, 22_000_000, 10, 10)...)
	tr, cov := porteVieVehicule(t, pos)
	if porteXs(tr)[-6.79] != 0 || porteXs(tr)[10] != 21 || cov.EchantillonsAuTraversDUnSilence != 1 {
		t.Errorf("premier echantillon aberrant : %v, compte %d", porteXs(tr), cov.EchantillonsAuTraversDUnSilence)
	}
}

// F-2 — a TAILLE EGALE (vehicule gare replique une fois par silence), c est le sejour que son
// autre voisin ne soutient pas qui tombe : jamais le vrai.
func TestPorteVehiculeEgaliteTrancheeParLeVoisin(t *testing.T) {
	pos := []grammar.BipedPosition{
		vehPos(770, 3_000_000, 5, 5),
		vehPos(770, 30_000_000, -201.16, 88.24),
		vehPos(770, 50_000_000, 5, 5),
	}
	tr, cov := porteVieVehicule(t, pos)
	if xs := porteXs(tr); xs[-201.16] != 0 || xs[5] != 2 || cov.EchantillonsAuTraversDUnSilence != 1 {
		t.Errorf("echantillons publies %v (compte %d) : attendu les deux vrais, sans l aberrant",
			xs, cov.EchantillonsAuTraversDUnSilence)
	}
}

// NEGATIF — un vehicule GARE 100 s : deux echantillons au meme point, separes par un silence.
// Rien n est ecarte ; le second porte la lacune, que le client TIENT sans interpoler.
func TestPorteVehiculeGareConserve(t *testing.T) {
	pos := []grammar.BipedPosition{vehPos(770, 3_000_000, 5, 5), vehPos(770, 103_000_000, 5, 5)}
	tr, cov := porteVieVehicule(t, pos)
	if len(tr.Samples) != 2 || cov.EchantillonsAuTraversDUnSilence != 0 {
		t.Fatalf("vehicule gare : %d echantillons publies (compte %d), attendu 2 et 0",
			len(tr.Samples), cov.EchantillonsAuTraversDUnSilence)
	}
	if tr.Samples[0].G != 0 || tr.Samples[1].G != 100_000 {
		t.Errorf("lacunes = %d / %d, attendu 0 / 100000", tr.Samples[0].G, tr.Samples[1].G)
	}
}

// NEGATIF — la CHUTE CONTINUE d un vehicule (Behemoth 771, t 3371-3384 : z -> -45) : replication
// sans silence, rien n est ecarte, aucune lacune.
func TestPorteVehiculeChuteContinueConservee(t *testing.T) {
	var pos []grammar.BipedPosition
	for i := 0; i <= 130; i++ {
		p := vehPos(770, 3_000_000+uint64(i)*100_000, 12, 30)
		p.Z = -float32(i) * 45 / 130
		pos = append(pos, p)
	}
	tr, cov := porteVieVehicule(t, pos)
	if len(tr.Samples) != 131 || cov.EchantillonsAuTraversDUnSilence != 0 {
		t.Fatalf("chute continue : %d echantillons (compte %d), attendu 131 et 0",
			len(tr.Samples), cov.EchantillonsAuTraversDUnSilence)
	}
	for _, s := range tr.Samples {
		if s.G != 0 {
			t.Fatalf("lacune publiee sur une replication continue : %+v", s)
		}
	}
}

// F-1 — au travers de l assemblage PUBLIC : l emprise est celle des joueurs, et elle ecarte un
// echantillon et une naissance hors carte. La vie dont la seule naissance lue est fausse, et qui
// n a aucun echantillon, n est plus publiee (`NoPosition`) ; la vie dont la naissance fausse
// precedait la vraie garde la vraie.
func TestPorteVehiculeHorsEmpriseAuTraversDeLAssemblage(t *testing.T) {
	vraie, fantome := types.EquipmentLifeKey{Slot: 770, Gen: 1}, types.EquipmentLifeKey{Slot: 771, Gen: 1}
	times := []uint64{2_000_000, 22_000_000}
	kf := vehKeyframes(times, vraie, times)
	kf.Band[771] = true
	kf.SeenUS[fantome] = times
	// Les z hors carte sont ceux de l annexe (motifs A et B : -390,99, -370,22, -235,46) ; la
	// carte de la foule fait 10 m de haut.
	fausse := vehCreation(vraie, 1_000_000, -201.16, 88.26, vehChassisUnknown) // plus precoce
	fausse.Z = -390.99
	naissanceFantome := vehCreation(fantome, 1_500_000, 217.3, -200.53, vehChassisUnknown)
	naissanceFantome.Z = -370.22
	horsCarte := vehPos(770, 3_100_000, -201.16, 88.24)
	horsCarte.Z = -235.46
	scan := VehicleScan{
		Scanned:   true,
		Keyframes: kf,
		Creations: []types.EquipmentCreation{fausse, vehCreation(vraie, 1_500_000, 40, 40, vehChassisKnown), naissanceFantome},
		Positions: []grammar.BipedPosition{vehPos(770, 3_000_000, 40, 40), horsCarte},
	}
	opt := Options{FrameIntervalMS: 100, Vehicles: scan}
	doc := BuildFromPositions("m", "halo_infinite", porteFoule(300, 0), nil, opt)
	if len(doc.Vehicles) != 1 {
		t.Fatalf("vies publiees = %d, attendu 1 (la fantome n a plus aucune position)", len(doc.Vehicles))
	}
	v := doc.Vehicles[0]
	if v.Spawn == nil || v.Spawn.X != 40 || v.Family != "ghost" || len(v.Samples) != 1 {
		t.Errorf("vie publiee = spawn %+v famille %q, %d echantillon(s) : attendu la vraie naissance, "+
			"sa famille, et le seul echantillon dans l emprise", v.Spawn, v.Family, len(v.Samples))
	}
	c := doc.Coverage.Vehicles
	if c.EchantillonsHorsEmprise != 1 || c.SpawnsHorsEmprise != 2 || c.NoPosition != 1 {
		t.Errorf("couverture = echantillons %d, naissances %d, sansPosition %d ; attendu 1, 2, 1",
			c.EchantillonsHorsEmprise, c.SpawnsHorsEmprise, c.NoPosition)
	}
	if got := porteRepli(doc, fallback.NomPositionHorsEmpriseEcartee); got != 3 {
		t.Errorf("repli publie = %d declenchements, attendu 3", got)
	}
}

// NEGATIF DE F-1 — le corpus temoin `50247b26` : une Wasp, un Ghost, un Warthog TOMBENT dans un
// vide, un echantillon par image, et la fin de leur chute sort de l emprise : elle reste publiee.
// Un vehicule largue d en haut (naissance hors emprise, puis descente continue) garde sa naissance.
// Trois echantillons IDENTIQUES hors carte, eux (la Wraith de 0a44c6cc), sont isoles et ecartes.
func TestPorteVehiculeContinuiteHorsEmprise(t *testing.T) {
	vie := types.EquipmentLifeKey{Slot: 770, Gen: 1}
	largue := types.EquipmentLifeKey{Slot: 771, Gen: 1}
	times := []uint64{2_000_000, 22_000_000}
	kf := vehKeyframes(times, vie, times)
	kf.Band[771] = true
	kf.SeenUS[largue] = times
	var positions []grammar.BipedPosition
	for i := 0; i <= 64; i++ { // chute : z 5 -> -148,6, 2,4 m par image (24 m/s, corpus temoin)
		p := vehPos(770, 3_000_000+uint64(i)*100_000, 20, 20)
		p.Z = 5 - 2.4*float32(i)
		positions = append(positions, p)
	}
	for i := 0; i < 3; i++ { // faux en-tete repete, isole
		p := vehPos(770, 12_000_000+uint64(i)*100_000, -19.41, -346.69)
		p.Z = 323.74
		positions = append(positions, p)
	}
	naissance := vehCreation(largue, 2_500_000, 30, 30, vehChassisKnown)
	naissance.Z = 400
	for i := 0; i <= 156; i++ { // descente du largage : z 395 -> 5, 2,5 m par image
		p := vehPos(771, 2_550_000+uint64(i)*100_000, 30, 30)
		p.Z = 395 - 2.5*float32(i)
		positions = append(positions, p)
	}
	scan := VehicleScan{Scanned: true, Keyframes: kf, Positions: positions,
		Creations: []types.EquipmentCreation{vehCreation(vie, 2_900_000, 20, 20, vehChassisKnown), naissance}}
	doc := BuildFromPositions("m", "halo_infinite", porteFoule(300, 0), nil, Options{FrameIntervalMS: 100, Vehicles: scan})
	par := map[uint32]VehicleTrack{}
	for _, v := range doc.Vehicles {
		par[v.Slot] = v
	}
	if got := len(par[770].Samples); got != 65 {
		t.Errorf("chute : %d echantillons publies, attendu 65 (les 3 faux ecartes, la chute gardee)", got)
	}
	if v := par[771]; v.Spawn == nil || v.Spawn.Z != 400 || len(v.Samples) != 157 {
		t.Errorf("largage : naissance %+v, %d echantillons ; attendu la naissance a z=400 et 157 echantillons",
			v.Spawn, len(v.Samples))
	}
	if c := doc.Coverage.Vehicles; c.EchantillonsHorsEmprise != 3 || c.SpawnsHorsEmprise != 0 {
		t.Errorf("couverture : echantillons %d, naissances %d ; attendu 3 et 0", c.EchantillonsHorsEmprise, c.SpawnsHorsEmprise)
	}
}
