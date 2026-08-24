package replay

// zone_states_lettres_test.go — LE RANG DE LETTRE des zones a bases simultanees
// (`ZoneState.LetterRank`), sur des enregistrements CONSTRUITS.
//
// CE QUE CES CAS VERROUILLENT, et pourquoi chacun existe :
//
//	l'ORDRE       le rang suit le NUMERO DE SLOT `ti=13`, jamais le `zoneRef`. C'est TOUT le
//	              fallback : sur les 8 cartes mesurees (phase 0.2), les deux ordres ne coincident
//	              jamais. Un cas ou le slot de la zone 1 est INFERIEUR a celui de la zone 0 fait
//	              donc echouer toute implementation qui rangerait par index de zone.
//	la BIJECTION  une zone du catalogue que rien n'apparie et PLUS AUCUNE lettre n'est publiee :
//	              sans cette garde, la zone suivante heriterait de la lettre de la muette.
//	l'ALPHABET    au-dela de trois zones, le HUD n'a plus de lettre — le fallback se tait.
//	la COLLINE    un mode a colline ne porte aucune lettre, par le chemin ET par la ceinture.
//	le COMPTEUR   `coverage.zones.letters` dit combien de lettres sont publiees, pour que « aucune
//	              lettre » ne se confonde pas avec « artefact anterieur au champ ».
//
// Les fabriques partagees (`zoneReadAt`, `zoneRampAt`, `bastionCase`, ...) vivent dans
// `zone_states_test.go`.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// lettreDe rend la lettre publiee d'une zone, ou "-" quand elle n'en porte pas.
func lettreDe(st ZoneState) string {
	if st.LetterRank == nil {
		return "-"
	}
	return string(rune('A' + *st.LetterRank))
}

// lettresPubliees rend la lettre de chaque zone, indexee par `zoneRef`.
func lettresPubliees(states []ZoneState) map[int]string {
	out := map[int]string{}
	for _, s := range states {
		out[s.ZoneRef] = lettreDe(s)
	}
	return out
}

// TestZoneLettresCasNominal : deux zones appariees, deux lettres, dans l'ordre des slots.
//
// Dans `bastionCase`, la zone 0 porte le slot de jauge 10 et la zone 1 le slot 20 : l'ordre des
// slots et l'ordre des zones coincident ici, et c'est le cas le plus simple.
func TestZoneLettresCasNominal(t *testing.T) {
	in, c := bastionCase()
	states, cov := buildZoneStates(in, c)
	got := lettresPubliees(states)
	if got[0] != "A" || got[1] != "B" {
		t.Errorf("lettres %v, attendu zone 0 = A (slot 10) et zone 1 = B (slot 20)", got)
	}
	if cov.Letters != 2 {
		t.Errorf("coverage.zones.letters = %d, attendu 2", cov.Letters)
	}
}

// TestZoneLettresSuiventLeSlotPasLeZoneRef — LE CAS QUI PORTE LE FALLBACK.
//
// Memes zones, memes captures, memes canaux de proprietaire : seuls les SLOTS DE JAUGE sont
// echanges, la zone 1 prenant le slot 10 et la zone 0 le slot 20. Les lettres doivent suivre les
// slots — zone 1 = A, zone 0 = B —, exactement l'inverse du cas nominal. Une implementation qui
// rangerait par `zoneRef` rendrait A et B dans l'autre sens et ferait echouer ce cas.
func TestZoneLettresSuiventLeSlotPasLeZoneRef(t *testing.T) {
	in, c := bastionCaseSlotsEchanges()
	states, cov := buildZoneStates(in, c)
	got := lettresPubliees(states)
	if got[1] != "A" || got[0] != "B" {
		t.Errorf("lettres %v, attendu zone 1 = A (slot 10) et zone 0 = B (slot 20) —"+
			" le rang suit le SLOT, pas l'index de zone", got)
	}
	if cov.Letters != 2 {
		t.Errorf("coverage.zones.letters = %d, attendu 2", cov.Letters)
	}
}

// bastionCaseSlotsEchanges reprend `bastionCase` en INVERSANT les slots de jauge : la zone 1 est
// desormais portee par le slot 10 et la zone 0 par le slot 20. Les canaux de proprietaire ne
// bougent pas (leur election se fait sur l'accord avec le roster, pas sur le voisinage de slot).
func bastionCaseSlotsEchanges() (ZoneInput, zoneCtx) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(20, 100, 900)...) // zone 101 (ref 0), desormais slot 20
	reads = append(reads, zoneRampAt(20, 300, 950)...)
	reads = append(reads, zoneRampAt(10, 200, 800)...) // zone 102 (ref 1), desormais slot 10
	reads = append(reads, zoneRampAt(10, 400, 820)...)
	reads = append(reads,
		zoneReadAt(11, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(11, 101, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(11, 301, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(21, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(21, 201, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(21, 401, filmdec.ManagedPropertyTagU32, 0),
	)
	actions := []ObjectiveAction{action("2533", 100), action("2535", 200), action("2535", 300),
		action("2533", 400)}
	tracks := []Track{
		track("2533", pointAt(100, -19.5, 0, 0), pointAt(400, 20.5, 0, 0)),
		track("2535", pointAt(200, 20.5, 0, 0), pointAt(300, -19.5, 0, 0)),
	}
	return zoneTestInput(reads), zoneTestCtx(actions, tracks)
}

// TestZoneLettresBijectionExigee : une zone du catalogue que rien n'apparie, et AUCUNE lettre
// n'est publiee — pas meme sur la zone qui, elle, est bien appariee.
//
// LE DECALAGE EST LE DANGER, ET IL EST INVISIBLE : si la zone appariee gardait « A » alors que la
// muette est peut-etre celle que le jeu nomme A, l'ecran afficherait une lettre credible et
// fausse. Le silence est la seule reponse honnete.
func TestZoneLettresBijectionExigee(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(10, 100, 900)...)
	reads = append(reads, zoneRampAt(10, 300, 950)...)
	reads = append(reads,
		zoneReadAt(11, 0, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(11, 101, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(11, 301, filmdec.ManagedPropertyTagU32, 1),
	)
	in := zoneTestInput(reads)
	// Seules les captures de la zone 0 : la zone 1 du catalogue reste muette sur ce match.
	actions := []ObjectiveAction{action("2533", 100), action("2535", 300)}
	tracks := []Track{
		track("2533", pointAt(100, -19.5, 0, 0)),
		track("2535", pointAt(300, -19.5, 0, 0)),
	}
	states, cov := buildZoneStates(in, zoneTestCtx(actions, tracks))
	if len(states) != 1 {
		t.Fatalf("%d zone(s) publiee(s), attendu 1 (seule la zone 0 est appariee) : %+v",
			len(states), states)
	}
	if l := lettreDe(states[0]); l != "-" {
		t.Errorf("lettre %q publiee sur une carte a l'appariement INCOMPLET (1 zone sur 2) —"+
			" attendu aucune", l)
	}
	if cov.Letters != 0 {
		t.Errorf("coverage.zones.letters = %d, attendu 0", cov.Letters)
	}
}

// TestZoneLettresJamaisSurUneColline : un mode a colline ne publie aucune lettre.
func TestZoneLettresJamaisSurUneColline(t *testing.T) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneRampAt(40, 100, 900)...)
	reads = append(reads, zoneRampAt(40, 400, 900)...)
	in := zoneTestInput(reads)
	in.Hill = true
	var pts []Point
	for f := 96; f <= 100; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0))
	}
	for f := 396; f <= 400; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0))
	}
	states, cov := buildZoneStates(in, zoneTestCtx(nil, []Track{track("2533", pts...)}))
	if len(states) == 0 {
		t.Fatalf("aucune periode de colline publiee : le cas ne juge rien")
	}
	for _, s := range states {
		if s.LetterRank != nil {
			t.Errorf("zone %d porte la lettre %q sur une COLLINE — le mode n'en a pas",
				s.ZoneRef, lettreDe(s))
		}
	}
	if cov.Letters != 0 {
		t.Errorf("coverage.zones.letters = %d sur un mode a colline, attendu 0", cov.Letters)
	}
}

// TestZoneLettresRegleDuRang verrouille la regle pure, y compris ses portes fermees.
func TestZoneLettresRegleDuRang(t *testing.T) {
	cas := []struct {
		nom     string
		gauge   map[int]uint32
		catalog int
		hill    bool
		want    map[int]int
	}{
		{
			nom:     "trois zones, rang par slot croissant",
			gauge:   map[int]uint32{0: 1542, 1: 1532, 2: 1537},
			catalog: 3,
			want:    map[int]int{1: 0, 2: 1, 0: 2}, // la table mesuree de Vagabond
		},
		{
			nom:     "appariement incomplet : aucune lettre",
			gauge:   map[int]uint32{0: 1532, 1: 1537},
			catalog: 3,
			want:    nil,
		},
		{
			nom:     "au-dela de trois zones : le HUD n'a pas la lettre suivante",
			gauge:   map[int]uint32{0: 10, 1: 20, 2: 30, 3: 40},
			catalog: 4,
			want:    nil,
		},
		{
			nom:     "mode a colline : ceinture fermee",
			gauge:   map[int]uint32{0: 10, 1: 20},
			catalog: 2,
			hill:    true,
			want:    nil,
		},
		{
			nom:     "catalogue vide",
			gauge:   map[int]uint32{},
			catalog: 0,
			want:    nil,
		},
	}
	for _, c := range cas {
		got := zoneLetterRanks(c.gauge, c.catalog, c.hill)
		if len(got) != len(c.want) {
			t.Errorf("%s : %v rangs, attendu %v", c.nom, got, c.want)
			continue
		}
		for ref, rank := range c.want {
			if got[ref] != rank {
				t.Errorf("%s : zone %d au rang %d, attendu %d (%v)", c.nom, ref, got[ref], rank, got)
			}
		}
	}
}
