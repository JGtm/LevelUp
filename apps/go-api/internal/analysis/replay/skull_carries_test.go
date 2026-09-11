package replay

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// skull_carries_test.go — LE PORTEUR DU CRANE, sur des enregistrements synthetiques (CI, sans
// film). On y prouve trois choses : les trains de tics deviennent des portages, un TROU de tics
// coupe un portage, et le porteur d'un train est nomme par l'identite de SA manche — donc le
// calque gere PLUSIEURS MANCHES sans melanger les porteurs.

// skullFixture fabrique un film Oddball a DEUX manches :
//   - manche 0 : slot 22 = "A" (deux portages, coupes par un trou), slot 20 = "C" ;
//   - manche 1 : slot 22 REATTRIBUE a "B".
//
// Le score de mode (`comp 0 A`) porte les tics ; le compteur de morts (`comp 2 B`) et le fil des
// morts nomment les slots par manche ; les prises (`comp 21 B`) alimentent la couverture.
func skullFixture() ([]objectiveevents.StatRecord, []objectiveevents.DeathInstant) {
	tick := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{0: {A: v}}}
	}
	death := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{2: {B: v}}}
	}
	grab := func(t, slot, round int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round,
			Comps: map[int]objectiveevents.StatValue{21: {B: v}}}
	}
	recs := []objectiveevents.StatRecord{
		// Manche 0 — slot 22 (A) : deux portages separes par un trou (4000 -> 9000, > 3 s).
		tick(1000, 22, 0, 1), tick(2000, 22, 0, 2), tick(3000, 22, 0, 3), tick(4000, 22, 0, 4),
		tick(9000, 22, 0, 5), tick(10000, 22, 0, 6),
		// Manche 0 — slot 20 (C).
		tick(5000, 20, 0, 1), tick(6000, 20, 0, 2), tick(7000, 20, 0, 3),
		// Manche 1 — slot 22 REATTRIBUE (B).
		tick(20000, 22, 1, 1), tick(21000, 22, 1, 2), tick(22000, 22, 1, 3),
		// Compteur de morts (identite par manche).
		death(4500, 22, 0, 1), death(4600, 22, 0, 2), death(4700, 22, 0, 3),
		death(7500, 20, 0, 1), death(7600, 20, 0, 2), death(7700, 20, 0, 3),
		death(22500, 22, 1, 1), death(22600, 22, 1, 2), death(22700, 22, 1, 3),
		// Prises (couverture) : 2 + 1 en manche 0, 1 en manche 1 = 4.
		grab(1000, 22, 0, 1), grab(9000, 22, 0, 2), grab(5000, 20, 0, 1), grab(20000, 22, 1, 1),
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].TimeMS < recs[j].TimeMS })
	deaths := []objectiveevents.DeathInstant{
		{XUID: "A", TimeMS: 4500}, {XUID: "A", TimeMS: 4600}, {XUID: "A", TimeMS: 4700},
		{XUID: "C", TimeMS: 7500}, {XUID: "C", TimeMS: 7600}, {XUID: "C", TimeMS: 7700},
		{XUID: "B", TimeMS: 22500}, {XUID: "B", TimeMS: 22600}, {XUID: "B", TimeMS: 22700},
	}
	return recs, deaths
}

// skullTestScan construit le scan de production a partir de la fixture.
func skullTestScan(recs []objectiveevents.StatRecord, deaths []objectiveevents.DeathInstant) SkullCarryScan {
	return SkullCarryScan{
		Scanned:  true,
		Records:  recs,
		Identity: objectiveevents.ResolveRoundIdentity(recs, deaths),
	}
}

func TestSkullCarriesTwoRounds(t *testing.T) {
	recs, deaths := skullFixture()
	// step = 1000 us/frame => 1 frame par ms (frame = instant en ms). frames grand : tout ferme.
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 100000}, carrierPresence{})

	if cov == nil || !cov.SkullFilm {
		t.Fatalf("couverture absente ou SkullFilm faux : %+v", cov)
	}
	if cov.Grabs != 4 {
		t.Errorf("prises = %d, attendu 4", cov.Grabs)
	}
	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4 : %+v", len(carries), carries)
	}
	// Ordre TOTAL par instant : A(1000), C(5000), A(9000), B(20000).
	//
	// LES BORNES PORTENT LA DEMI-FENETRE DE TIC (lot 6.7-B1, item 2) : la cadence mesuree de la
	// fixture vaut 1 000 ms, l'axe 1 000 us par image, donc 499 images de part et d'autre (cf.
	// [skullHalfTickFrames]). Les instants des TICS, eux, sont inchanges.
	want := []struct {
		xuid   string
		t0, t1 int
	}{
		{"A", 1000 - testHalfTickFrames, 4000 + testHalfTickFrames},
		{"C", 5000 - testHalfTickFrames, 7000 + testHalfTickFrames},
		{"A", 9000 - testHalfTickFrames, 10000 + testHalfTickFrames},
		{"B", 20000 - testHalfTickFrames, 22000 + testHalfTickFrames},
	}
	for i, w := range want {
		if carries[i].XUID != w.xuid || carries[i].T0 != w.t0 || carries[i].T1 != w.t1 {
			t.Errorf("portage %d = %+v, attendu {%s %d %d}", i, carries[i], w.xuid, w.t0, w.t1)
		}
	}
	// LE POINT DU LOT : en manche 1 le porteur est B (slot reattribue), pas A.
	if carries[3].XUID != "B" {
		t.Errorf("porteur manche 1 = %q, attendu \"B\" (le pont plat aurait dit A)", carries[3].XUID)
	}
	if cov.Trains != 4 || cov.Carries != 4 || cov.NoBridge != 0 || cov.OutOfWindow != 0 {
		t.Errorf("couverture = %+v, attendu 4 trains / 4 portages / 0 rejet", cov)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestSkullCarriesOpenAtAxisEnd — un portage dont le dernier tic bute sur la fin de l'axe n'est
// pas ferme (borne haute), les autres le sont.
func TestSkullCarriesOpenAtAxisEnd(t *testing.T) {
	recs, deaths := skullFixture()
	// frames = 23000 : le dernier portage (fin 22000) tombe dans le mou de fin (3 s) -> ouvert.
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 23000}, carrierPresence{})
	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4", len(carries))
	}
	if carries[3].Closed {
		t.Errorf("le portage de fin d'axe devrait etre OUVERT (borne haute)")
	}
	for i := 0; i < 3; i++ {
		if !carries[i].Closed {
			t.Errorf("portage %d devrait etre FERME (un fait le borne avant la fin)", i)
		}
	}
	if cov.Open != 1 || cov.Closed != 3 {
		t.Errorf("couverture ouverts/fermes = %d/%d, attendu 1/3", cov.Open, cov.Closed)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestSkullCarriesUnscanned — hors Oddball (Scanned faux), ni calque ni couverture.
func TestSkullCarriesUnscanned(t *testing.T) {
	carries, cov := buildSkullCarries(SkullCarryScan{Scanned: false}, matchClock{step: 1000, frames: 100}, carrierPresence{})
	if carries != nil || cov != nil {
		t.Errorf("film non-Oddball : attendu (nil, nil), obtenu (%v, %v)", carries, cov)
	}
}

// TestSkullCarriesCarrierAbsent — le gate de PRESENCE : un portage attribue a un joueur ABSENT de la
// carte pendant l'intervalle est ecarte (fantome), et un portage qui deborde de la presence est
// rogne a elle. Contre-epreuve : sans presence (nil), les memes trains sortent tous (cf.
// TestSkullCarriesTwoRounds, 4 portages).
func TestSkullCarriesCarrierAbsent(t *testing.T) {
	recs, deaths := skullFixture()
	// A n'est present que [1500,3500] : couvre en PARTIE son 1er portage (1000-4000) mais PAS son
	// 2e (9000-10000). C et B couvrent leurs portages.
	presence := carrierPresence{named: map[string][]presenceSpan{
		"A": {{1500, 3500}},
		"C": {{5000, 7000}},
		"B": {{20000, 22000}},
	}}
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 100000}, presence)

	if len(carries) != 3 {
		t.Fatalf("portages = %d, attendu 3 (le fantome A@9000 ecarte) : %+v", len(carries), carries)
	}
	if cov.CarrierAbsent != 1 {
		t.Errorf("CarrierAbsent = %d, attendu 1", cov.CarrierAbsent)
	}
	// Le 1er portage de A est ROGNE a sa presence [1500,3500] (au lieu de 1000-4000).
	if carries[0].XUID != "A" || carries[0].T0 != 1500 || carries[0].T1 != 3500 {
		t.Errorf("portage A rogne = %+v, attendu {A 1500 3500}", carries[0])
	}
	// C et B intacts (leur presence couvre tout l'intervalle).
	if carries[1].XUID != "C" || carries[1].T0 != 5000 || carries[1].T1 != 7000 {
		t.Errorf("portage C = %+v, attendu {C 5000 7000}", carries[1])
	}
	if carries[2].XUID != "B" {
		t.Errorf("porteur restant = %q, attendu \"B\"", carries[2].XUID)
	}
	if cov.Trains != 4 || cov.Carries != 3 {
		t.Errorf("couverture = %+v, attendu 4 trains / 3 portages", cov)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestSkullCarrierPresence — l'index de presence groupe les vies bipedes NOMMEES par xuid et
// RETIENT a part celles qui ne prouvent l'absence de personne : une vie sans identite, et une
// vie dont l'identite est DEDUITE (cf. carrierPresenceOf). Elles ne sont pas jetees, elles sont
// la trace de l'ignorance.
func TestSkullCarrierPresence(t *testing.T) {
	tracks := []Track{
		{Slot: 1, XUID: "1", StartFrame: 10, EndFrame: 40},
		{Slot: 1, XUID: "1", StartFrame: 100, EndFrame: 130},
		{Slot: 3, XUID: "", StartFrame: 0, EndFrame: 999},                  // sans identite : RETENUE
		{Slot: 4, XUID: "", Bot: "Ciri [bot]", StartFrame: 5, EndFrame: 8}, // bot : IDENTIFIE
		{Slot: 2, XUID: "2", StartFrame: 50, EndFrame: 70},
	}
	// Aucune identite deduite ici : les quatre xuids viennent de la lecture.
	deduites := map[int]bool{}
	p := carrierPresenceOf(tracks, deduites)
	if len(p.named) != 2 {
		t.Fatalf("xuids indexes = %d, attendu 2 : %+v", len(p.named), p.named)
	}
	if len(p.unnamed) != 1 || p.unnamed[0] != (presenceSpan{0, 999}) {
		t.Errorf("vies anonymes = %+v, attendu une seule [0,999]", p.unnamed)
	}
	if len(p.named["1"]) != 2 || len(p.named["2"]) != 1 {
		t.Errorf("A=%d vies, B=%d vies, attendu 2 et 1", len(p.named["1"]), len(p.named["2"]))
	}
	if _, ok := unionOverlap(p.named["1"], 20, 25); !ok {
		t.Errorf("[20,25] devrait recouvrir la vie A [10,40]")
	}
	if _, ok := unionOverlap(p.named["1"], 60, 90); ok {
		t.Errorf("[60,90] ne devrait recouvrir aucune vie A (trou entre [10,40] et [100,130])")
	}
}

// TestSkullCarriesVieAnonymeNEstPasUneAbsence — LE FAIT RESIDUEL n° 1 du balayage du parc
// (`d9781168`, 36 portages -> 30). Un portage dont le porteur n'a AUCUNE vie nommee sur
// l'intervalle, mais qu'une vie ANONYME recouvre, ne doit etre ni ecarte ni rogne : le pont
// d'identite n'a pas nomme cette vie, et une presence sans identite n'est pas une absence.
//
// Verite terrain de `d9781168` : en Oddball le score EST le temps de portage ; la feuille de match
// donne 191 s / 196 s par equipe, l'artefact rejetant les anonymes n'en publiait que 60,1 s /
// 147,4 s. Le gate ecartait de la donnee vraie en croyant ecarter des fantomes.
func TestSkullCarriesVieAnonymeNEstPasUneAbsence(t *testing.T) {
	recs, deaths := skullFixture()
	// Meme presence nommee que TestSkullCarriesCarrierAbsent : A n'est nomme que [1500,3500],
	// donc son 1er portage (1000-4000) serait ROGNE et son 2e (9000-10000) ECARTE.
	base := map[string][]presenceSpan{
		"A": {{1500, 3500}},
		"C": {{5000, 7000}},
		"B": {{20000, 22000}},
	}
	// Une seule vie ANONYME, qui couvre les DEUX intervalles de A.
	p := carrierPresence{named: base, unnamed: []presenceSpan{{500, 11000}}}
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 100000}, p)

	if len(carries) != 4 {
		t.Fatalf("portages = %d, attendu 4 (aucun ecarte) : %+v", len(carries), carries)
	}
	if cov.CarrierAbsent != 0 {
		t.Errorf("CarrierAbsent = %d, attendu 0 : une vie anonyme couvre l'intervalle", cov.CarrierAbsent)
	}
	// Le 1er portage de A n'est PAS rogne a [1500,3500] : il garde ses bornes, demi-fenetre
	// de tic comprise (lot 6.7-B1, item 2).
	if carries[0].XUID != "A" || carries[0].T0 != 1000-testHalfTickFrames ||
		carries[0].T1 != 4000+testHalfTickFrames {
		t.Errorf("portage A = %+v, attendu {A %d %d} NON rogne", carries[0],
			1000-testHalfTickFrames, 4000+testHalfTickFrames)
	}
	if !cov.Balanced() {
		t.Errorf("couverture desequilibree : %+v", cov)
	}
}

// TestSkullCarriesFantomeResteEcarte — la CONTRE-EPREUVE du test precedent : sans vie anonyme sur
// l'intervalle, les pistes publiees rendent compte de tout, et le portage attribue a un joueur
// qui n'y est pas reste un FANTOME. Le correctif retrecit le gate, il ne le supprime pas.
func TestSkullCarriesFantomeResteEcarte(t *testing.T) {
	recs, deaths := skullFixture()
	p := carrierPresence{
		named: map[string][]presenceSpan{
			"A": {{1500, 3500}},
			"C": {{5000, 7000}},
			"B": {{20000, 22000}},
		},
		// Une vie anonyme LOIN des portages de A : elle ne couvre rien de [9000,10000].
		unnamed: []presenceSpan{{30000, 40000}},
	}
	carries, cov := buildSkullCarries(skullTestScan(recs, deaths),
		matchClock{origin: 0, step: 1000, frames: 100000}, p)
	if len(carries) != 3 || cov.CarrierAbsent != 1 {
		t.Fatalf("portages = %d / CarrierAbsent = %d, attendu 3 et 1 (le fantome A@9000 ecarte)",
			len(carries), cov.CarrierAbsent)
	}
	if carries[0].T0 != 1500 || carries[0].T1 != 3500 {
		t.Errorf("portage A = %+v, attendu rogne a [1500,3500]", carries[0])
	}
}

// TestSkullCarrySecondsByXUID — la GRANDEUR DU GATE ORACLE (porteur principal) : la somme des
// durees de portage par joueur, toutes manches confondues. Le temoin sur film reel skip en CI
// (pas de corpus) ; ce test synthetique gate la fonction que le temoin re-cuit, sans film.
//
// Attendus de `skullFixture` : A = 3 s (1000-4000) + 1 s (9000-10000) = 4 s ; C = 2 s
// (5000-7000) ; B = 2 s en manche 1 (20000-22000, slot 22 reattribue).
func TestSkullCarrySecondsByXUID(t *testing.T) {
	recs, deaths := skullFixture()
	got := skullCarrySecondsByXUID(recs, objectiveevents.ResolveRoundIdentity(recs, deaths))
	want := map[string]float64{"A": 4, "C": 2, "B": 2}
	if len(got) != len(want) {
		t.Fatalf("%d porteur(s), attendu %d : %+v", len(got), len(want), got)
	}
	for x, w := range want {
		if got[x] != w {
			t.Errorf("duree de portage de %q = %.1f s, attendu %.1f s (%+v)", x, got[x], w, got)
		}
	}
	// Le porteur PRINCIPAL (argmax) est A — c'est ce que le gate oracle confronte a l'oracle fige.
	best := ""
	for x, s := range got {
		if best == "" || s > got[best] {
			best = x
		}
	}
	if best != "A" {
		t.Errorf("porteur principal = %q, attendu \"A\"", best)
	}
}

// TestPortageAChevalSurDeuxViesNommeesGardeSesBornes — LE RESIDU A, instruit le 2026-09-06.
//
// VERDICT : CONFIRME, l'exemption « lecteur deja rattrape » ne le couvrait PAS. Le correctif du
// schema 43 a traite le REJET (« l'ignorance passe avant le rognage ») ; le ROGNAGE, lui, est
// reste sur `bestOverlap` — la vie de recouvrement MAXIMAL. C'est exactement le defaut que
// `windowFor` portait avant le schema 45 et que `spanFor` a ferme pour les episodes
// d'equipement : un portage qu'un trou de replication de plus de `lifeGapUS` coupe en deux vies
// NOMMEES du meme porteur etait tronque a la moitie la plus longue, l'instant de PRISE compris.
//
// LE CAS : le porteur a deux vies nommees, [10..30] et [80..120], separees par un trou. Le
// portage mesure court de 20 a 100 : il enjambe le trou. L'union rend [10..120], le clamp
// ramene a [20..100] — les bornes MESUREES, intactes.
//
// MUTATION : revenir a `bestOverlap` rougit — le portage sort [80..100], ampute de 60 frames
// (la vie [80..120] recouvre 21 frames du portage contre 11 pour [10..30]).
func TestPortageAChevalSurDeuxViesNommeesGardeSesBornes(t *testing.T) {
	p := carrierPresence{named: map[string][]presenceSpan{
		"A": {{f0: 10, f1: 30}, {f0: 80, f1: 120}},
	}}
	f0, f1, ok := p.gate("A", 20, 100)
	if !ok {
		t.Fatal("portage ecarte : deux vies nommees le recouvrent")
	}
	if f0 != 20 || f1 != 100 {
		t.Errorf("bornes publiees [%d..%d], attendu [20..100] — un trou de replication ne doit "+
			"pas amputer une duree mesuree", f0, f1)
	}
}

// TestPortageResteRogneHorsDesViesNommees — LA CONTRE-EPREUVE : l'union ne deborde JAMAIS les
// vies du porteur. Un portage qui commence avant sa premiere vie et finit apres la derniere est
// toujours ramene a ce que les pistes rendent compte.
func TestPortageResteRogneHorsDesViesNommees(t *testing.T) {
	p := carrierPresence{named: map[string][]presenceSpan{
		"A": {{f0: 10, f1: 30}, {f0: 80, f1: 120}},
	}}
	f0, f1, ok := p.gate("A", 0, 200)
	if !ok {
		t.Fatal("portage ecarte a tort")
	}
	if f0 != 10 || f1 != 120 {
		t.Errorf("bornes publiees [%d..%d], attendu [10..120] — l'union est bornee par les vies", f0, f1)
	}
	// Et un portage qu'AUCUNE vie nommee ne recouvre reste un FANTOME : la regle de rejet ne
	// bouge pas.
	if _, _, ok := p.gate("A", 300, 400); ok {
		t.Error("portage hors de toute vie nommee : il devait rester ecarte")
	}
}

// TestUneIdentiteDEDUITENeProuveLAbsenceDePersonne — L'INTERACTION P0-0 x GATE DE PRESENCE
// (2026-09-07), attrapee par la cuisson des temoins.
//
// Depuis le nommage final (unnamed_lives.go), plus aucune vie n'est publiee sans identite :
// `carrierPresence.unnamed` serait donc VIDE, et l'abstention n 2 du gate — la moitie la plus
// couteuse du correctif du schema 43 — mourrait avec elle. Une vie nommee PAR DEDUCTION
// (l'occupation du slot dans le temps) etablit qu'un joueur etait probablement la ; elle
// n'etablit JAMAIS qu'un autre n'y etait pas. La compter comme une preuve d'absence
// transformerait une deduction en refutation.
//
// MESURE : sur `d9781168` (Oddball, dont le score EST le temps de portage), la compter coutait
// un portage et 101 frames, en S'ELOIGNANT de la feuille de match — 387 s reelles, 331,3 s
// publiees avec cette garde contre 321,2 s sans.
//
// MUTATION : retirer la branche `if !identityIsRead(...)` rougit (« portage ecarte »).
func TestUneIdentiteDEDUITENeProuveLAbsenceDePersonne(t *testing.T) {
	// Le slot 7 porte une vie NOMMEE PAR LECTURE (joueur B, fil des morts) et une vie que seul
	// le nommage final a nommee — a B aussi, par occupation.
	tracks := []Track{
		{Slot: 7, StartFrame: 0, EndFrame: 50, XUID: "2"},
		{Slot: 7, StartFrame: 100, EndFrame: 200, XUID: "2"}, // identite DEDUITE
		{Slot: 9, StartFrame: 0, EndFrame: 300, XUID: "1"},
	}
	// La piste d'indice 1 est celle que le nommage final a nommee.
	p := carrierPresenceOf(tracks, map[int]bool{1: true})
	if len(p.unnamed) != 1 {
		t.Fatalf("vies sans identite LUE = %d, attendu 1 (la vie deduite du slot 7)", len(p.unnamed))
	}

	// Un portage de A sur [120..180] : la vie deduite le recouvre. Le gate doit S'ABSTENIR,
	// pas rejeter — la deduction ne refute rien.
	f0, f1, ok := p.gate("1", 120, 180)
	if !ok || f0 != 120 || f1 != 180 {
		t.Errorf("portage [%d..%d] ok=%v ; attendu [120..180] conserve : une identite DEDUITE "+
			"ne prouve l'absence de personne", f0, f1, ok)
	}
}

// TestUneIdentiteLUEProuveToujoursUnePresence — LA CONTRE-EPREUVE : le gate garde ses dents.
// Une vie nommee PAR LECTURE rend compte de l'occupation du slot, et un portage attribue a un
// joueur qu'aucune de SES vies ne recouvre reste un fantome.
func TestUneIdentiteLUEProuveToujoursUnePresence(t *testing.T) {
	tracks := []Track{
		{Slot: 7, StartFrame: 100, EndFrame: 200, XUID: "2"},
		{Slot: 9, StartFrame: 0, EndFrame: 50, XUID: "1"},
	}
	p := carrierPresenceOf(tracks, nil) // aucune identite deduite : tout vient de la lecture
	if len(p.unnamed) != 0 {
		t.Fatalf("aucune identite deduite ici : unnamed = %d, attendu 0", len(p.unnamed))
	}
	if _, _, ok := p.gate("1", 120, 180); ok {
		t.Error("portage de A sur un intervalle que seule une vie LUE de B recouvre : " +
			"il devait rester ecarte")
	}
}
