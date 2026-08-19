package replay

// zone_states_test.go — LE CALQUE DE L'ETAT DES ZONES, sur des enregistrements CONSTRUITS.
//
// AUCUN FILM ICI, ET C'EST LE POINT : la regle d'appariement, la lecture du proprietaire et les
// degradations se testent sur des lectures fabriquees, donc a la milliseconde pres et sans
// dependre d'un fichier de 300 Mo. Les temoins reels (deux Bastion, un KOTH) sont cuits a part
// et publies au journal du lot ; ces fichiers verrouillent la REGLE.
//
// LE DECOUPAGE SUIT LES SOURCES : ce fichier porte le TRONC (assemblage, couverture, degradations)
// et les fabriques partagees ; `zone_states_owner_test.go` porte le volet du proprietaire et
// `zone_states_hill_test.go` le volet colline — a l'image de `zone_states_owner.go` et de
// `zone_states_hill.go`.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// zoneReadAt fabrique une lecture scalaire de `ti=13` posee sur une frame de la grille.
func zoneReadAt(slot uint32, frame, tag int, value uint64) filmdec.ManagedPropertyRead {
	return filmdec.ManagedPropertyRead{
		Slot: slot, TimestampUS: uint64(frame) * 100_000, Field: filmdec.ManagedPropertyScalar,
		FilmIndex: -1, Tag: tag, Value: value, HasValue: true,
	}
}

// gaugeQ rend le quantum d'une valeur de jauge donnee en MILLIEMES d'unite du jeu (0 = jauge au
// repos, 1 000 = pleine) — l'echelle de gaugeProgressOf.
func gaugeQ(milli uint64) uint64 {
	return zoneGaugeQuantZero + milli*zoneGaugeQuantUnit/1000
}

// zoneRampAt fabrique une rampe de jauge culminant a `peak` : trois emissions croissantes (0,001,
// 0,2 puis `topMilli` milliemes) et une amplitude tres au-dessus du seuil (4 096 quanta).
func zoneRampAt(slot uint32, peak int, topMilli uint64) []filmdec.ManagedPropertyRead {
	return []filmdec.ManagedPropertyRead{
		zoneReadAt(slot, peak-4, filmdec.ManagedPropertyTagQuant, gaugeQ(1)),
		zoneReadAt(slot, peak-2, filmdec.ManagedPropertyTagQuant, gaugeQ(200)),
		zoneReadAt(slot, peak, filmdec.ManagedPropertyTagQuant, gaugeQ(topMilli)),
	}
}

// zoneTestCtx est l'axe de temps commun aux cas : grille de 100 ms, 600 frames.
func zoneTestCtx(actions []ObjectiveAction, tracks []Track) zoneCtx {
	return zoneCtx{origin: 0, step: 100_000, frames: 600, intervalMS: 100,
		tracks: tracks, actions: actions, matchID: "test"}
}

// zoneTestInput monte l'entree d'un cas a deux zones.
func zoneTestInput(reads []filmdec.ManagedPropertyRead) ZoneInput {
	return ZoneInput{
		Scanned:    true,
		Reads:      reads,
		Zones:      []Zone{zoneAt(0, 101, -20, 0, 0), zoneAt(1, 102, 20, 0, 0)},
		Roles:      "strongholds_zone",
		TeamByXUID: map[string]int{"2533": 0, "2535": 1},
	}
}

// bastionCase monte le cas nominal : deux zones, deux jauges, deux canaux de proprietaire,
// QUATRE captures nommees posees DANS leur zone — deux par zone.
//
// DEUX CAPTURES PAR ZONE, ET C'EST LE SEUIL QUI L'EXIGE (revue R1) : un canal n'est elu
// proprietaire qu'a partir de deux concordances avec le roster (zoneOwnerMinAgreements). Une
// zone prise une seule fois dans tout le match n'a pas de proprietaire publie — c'est la regle,
// et le cas nominal doit la franchir plutot que de vivre dessous.
func bastionCase() (ZoneInput, zoneCtx) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(10, 100, 900)...) // zone 101, prise par l'equipe 0
	reads = append(reads, zoneRampAt(10, 300, 950)...) // zone 101, reprise par l'equipe 1
	reads = append(reads, zoneRampAt(20, 200, 800)...) // zone 102, prise par l'equipe 1
	reads = append(reads, zoneRampAt(20, 400, 820)...) // zone 102, reprise par l'equipe 0
	reads = append(reads,
		zoneReadAt(11, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(11, 101, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(11, 301, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(21, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(21, 201, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(21, 401, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(10, 5, filmdec.ManagedPropertyTagStringID, 0x67F43AC3),
	)
	actions := []ObjectiveAction{action("2533", 100), action("2535", 200), action("2535", 300),
		action("2533", 400)}
	tracks := []Track{
		track("2533", pointAt(100, -19.5, 0, 0), pointAt(400, 20.5, 0, 0)),
		track("2535", pointAt(200, 20.5, 0, 0), pointAt(300, -19.5, 0, 0)),
	}
	return zoneTestInput(reads), zoneTestCtx(actions, tracks)
}

func intPtr(v int) *int { return &v }

// TestZoneStatesPublieUnEtatParZoneAppariee est le cas nominal : chaque zone que les captures
// rattachent a un slot de jauge sort avec ses intervalles de propriete.
func TestZoneStatesPublieUnEtatParZoneAppariee(t *testing.T) {
	in, c := bastionCase()
	states, cov := buildZoneStates(in, c)
	if len(states) != 2 {
		t.Fatalf("%d zone(s) publiee(s), attendu 2 : %+v", len(states), states)
	}
	if states[0].ZoneRef != 0 || states[1].ZoneRef != 1 {
		t.Errorf("zoneRef publies %d et %d, attendu 0 et 1 — l'index doit etre celui du catalogue",
			states[0].ZoneRef, states[1].ZoneRef)
	}
	if states[0].Key != 0x67F43AC3 {
		t.Errorf("cle du slot de jauge %#x, attendu 0x67F43AC3", states[0].Key)
	}
	if cov.Method != ZoneMethodCaptures {
		t.Errorf("methode %q, attendu %q", cov.Method, ZoneMethodCaptures)
	}
	if cov.Paired != 2 || cov.Unpaired != 0 {
		t.Errorf("apparies %d / non apparies %d, attendu 2 / 0", cov.Paired, cov.Unpaired)
	}
	if cov.Captures != 4 || cov.Attributed != 4 {
		t.Errorf("captures %d, attribuees %d — attendu 4 et 4", cov.Captures, cov.Attributed)
	}
	if cov.OwnerUnpaired != 0 {
		t.Errorf("zones sans proprietaire elu %d, attendu 0", cov.OwnerUnpaired)
	}
}

// TestZoneStatesProgressionSurLEchelleDuJeu : la progression est la fraction de capture sur
// l'echelle du JEU (0 = le zero quantifie de [-100, +100], 1 = +1,0 unite), ecretee — pas la part
// d'une excursion mesuree, qu'une emission aberrante sous zero suffisait a fausser (lot C-ter
// volet 3, temoin `7344d24f`).
func TestZoneStatesProgressionSurLEchelleDuJeu(t *testing.T) {
	cas := []struct {
		q    uint64
		want float32
	}{
		{zoneGaugeQuantZero, 0}, {gaugeQ(1000), 1}, {gaugeQ(500), 0.5},
		{zoneGaugeQuantZero - 4_000_000, 0}, {gaugeQ(1000) + 2_000_000, 1},
	}
	for _, c := range cas {
		if got := gaugeProgressOf(c.q); got < c.want-0.001 || got > c.want+0.001 {
			t.Errorf("progression de %d = %v, attendu %v", c.q, got, c.want)
		}
	}
}

// TestZoneStatesSansCatalogueNePublieRien : sans zone de carte, aucun intervalle n'a de cible.
// La couverture, elle, est publiee — c'est elle qui distingue les silences.
func TestZoneStatesSansCatalogueNePublieRien(t *testing.T) {
	in, c := bastionCase()
	in.Zones = nil
	states, cov := buildZoneStates(in, c)
	if states != nil {
		t.Errorf("%d etat(s) publie(s) sans catalogue de zones", len(states))
	}
	if cov == nil || cov.Catalog != 0 {
		t.Fatalf("couverture attendue avec un catalogue vide, obtenu %+v", cov)
	}
}

// TestZoneStatesNonBalayeNePublieAucuneCouverture : « rien n'a ete lu » et « rien n'existait »
// ne sont pas la meme chose, et l'absence de couverture porte la premiere.
func TestZoneStatesNonBalayeNePublieAucuneCouverture(t *testing.T) {
	in, c := bastionCase()
	in.Scanned = false
	states, cov := buildZoneStates(in, c)
	if states != nil || cov != nil {
		t.Errorf("balayage absent : attendu (nil, nil), obtenu (%d etats, %+v)", len(states), cov)
	}
}

// TestZoneStatesTientLeVolumeDUnVraiFilm — LE COUT DU CALQUE, sur les ordres de grandeur
// MESURES d'un vrai match de Bastion (`7344d24f`, 2026-08-18) : 26 slots dont 5 de jauge a
// ~1 000 emissions et 10 de propriete, 246 captures nommees, 8 joueurs, ~5 000 frames.
//
// POURQUOI CE TEST EXISTE. Le calque tourne DANS l'assemblage d'artefact, derriere une dizaine
// de balayages de film : une regle quadratique y serait invisible en unitaire et couterait des
// minutes en production. Le volume est donc reproduit ici, sans film — si une boucle devient
// quadratique, ce test ne rend plus la main.
func TestZoneStatesTientLeVolumeDUnVraiFilm(t *testing.T) {
	const frames, captures = 5000, 246
	var reads []filmdec.ManagedPropertyRead
	for slot := uint32(10); slot < 15; slot++ { // 5 slots de jauge, ~1 000 emissions chacun
		for i := 0; i < 330; i++ {
			reads = append(reads, zoneRampAt(slot, 8+i*15, uint64(500+i))...)
		}
	}
	for slot := uint32(30); slot < 40; slot++ { // 10 slots de propriete
		for i := 0; i < 40; i++ {
			v := uint64(i % 2)
			if i%7 == 0 {
				v = zoneNeutralOwner
			}
			reads = append(reads, zoneReadAt(slot, 20+i*120, filmdec.ManagedPropertyTagU32, v))
		}
	}
	in := zoneTestInput(reads)
	actions := make([]ObjectiveAction, 0, captures)
	pts := make([]Point, 0, frames)
	for f := 0; f < frames; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0))
	}
	for i := 0; i < captures; i++ {
		actions = append(actions, action("2533", 8+i*20))
	}
	c := zoneCtx{origin: 0, step: 100_000, frames: frames, intervalMS: 100,
		tracks: []Track{track("2533", pts...)}, actions: actions, matchID: "volume"}
	states, cov := buildZoneStates(in, c)
	if cov == nil {
		t.Fatalf("aucune couverture rendue sur le cas de volume")
	}
	t.Logf("volume : %d lectures, %d slots, %d captures -> %d zone(s), %d intervalle(s)",
		len(reads), cov.Slots, cov.Captures, len(states), cov.Spans)
}
