package replay

// zone_states_hill_owner_test.go — LE PROPRIETAIRE DE LA COLLINE (2026-08-26).
//
// Le canal est le tag 4 du slot VOISIN du designateur. Son niveau de preuve — 88-89 % d'accord
// contre un temoin a 56 %, sous le seuil de 90 % que le plan s'etait fixe, accepte par decision
// utilisateur — est documente en tete de `hillStatesOf`, avec les trois campagnes qui l'ont
// mesure. Ces cas ne rejouent PAS cette mesure : ils tiennent la REGLE de publication, sur des
// enregistrements construits.
//
// CE QU'ILS PROTEGENT : la periode se subdivise aux changements de main ; le neutre est une
// mesure (`Owner` nil) et non une absence ; une valeur inconnue n'ouvre AUCUN intervalle et se
// compte ; un film qui n'emet qu'UN SEUL camp — le cas reel de `606d9844` et `8076f97f` — ne
// publie pas de camp la ou le canal se tait ; et le repli par les rampes ne publie jamais de
// proprietaire.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// hillOwnerCase monte un cas a designateur avec un canal de propriete SUR MESURE.
//
// Le designateur vit au slot 40 (deux bascules : frames 200 et 400, donc trois periodes
// [50 ; 199], [200 ; 399], [400 ; 599] une fois le premier contact pris a 50), le canal de
// propriete au slot 41. Les positions localisent les trois periodes.
func hillOwnerCase(owner []filmdec.ManagedPropertyRead) (ZoneInput, zoneCtx) {
	var reads []filmdec.ManagedPropertyRead
	reads = append(reads, zoneChainedReadAt(40, 200, filmdec.ManagedPropertyTagStringID, 0x78F81557))
	reads = append(reads, zoneChainedReadAt(40, 400, filmdec.ManagedPropertyTagStringID, 0x8727C0FF))
	reads = append(reads, owner...)
	in := zoneTestInput(reads)
	in.Hill = true
	var pts []Point
	for f := 60; f <= 70; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0)) // zone 1
	}
	for f := 250; f <= 260; f++ {
		pts = append(pts, pointAt(f, -19.5, 0, 0)) // zone 0
	}
	for f := 450; f <= 460; f++ {
		pts = append(pts, pointAt(f, 20.5, 0, 0)) // zone 1
	}
	return in, zoneTestCtx(nil, []Track{track("2533", pts...)})
}

// spansTries rend tous les intervalles publies, toutes zones confondues, tries par T0.
func spansTries(states []ZoneState) []ZoneSpan {
	var out []ZoneSpan
	for _, st := range states {
		out = append(out, st.Spans...)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].T0 < out[j-1].T0; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// TestCollineProprietaireSubdiviseLaPeriode — la colline change de main PENDANT une periode :
// l'intervalle se coupe a la bascule, et chaque morceau porte SON camp.
func TestCollineProprietaireSubdiviseLaPeriode(t *testing.T) {
	in, c := hillOwnerCase([]filmdec.ManagedPropertyRead{
		zoneReadAt(41, 60, filmdec.ManagedPropertyTagU32, 0),
		zoneReadAt(41, 300, filmdec.ManagedPropertyTagU32, 1),
	})
	states, cov := buildZoneStates(in, c)
	if cov.Method != ZoneMethodDesignator {
		t.Fatalf("methode %q, attendu %q", cov.Method, ZoneMethodDesignator)
	}
	// La periode [200 ; 399] se coupe a 300 : [200 ; 299] au camp 0, [300 ; 399] au camp 1.
	veut := []struct {
		t0, t1 int
		owner  *int
	}{
		{60, 199, ptr(0)}, // premier contact du canal a 60, jusqu'a la bascule du designateur
		{200, 299, ptr(0)},
		{300, 399, ptr(1)},
		{400, 599, ptr(1)},
	}
	got := spansTries(states)
	if len(got) != len(veut) {
		t.Fatalf("%d intervalle(s), attendu %d : %+v", len(got), len(veut), got)
	}
	for i, sp := range got {
		if sp.T0 != veut[i].t0 || sp.T1 != veut[i].t1 {
			t.Errorf("intervalle %d : [%d ; %d], attendu [%d ; %d]", i, sp.T0, sp.T1,
				veut[i].t0, veut[i].t1)
		}
		if sp.Owner == nil || *sp.Owner != *veut[i].owner {
			t.Errorf("intervalle %d [%d ; %d] : proprietaire %v, attendu %d", i, sp.T0, sp.T1,
				sp.Owner, *veut[i].owner)
		}
		if !sp.Active {
			t.Errorf("intervalle %d : non marque ACTIF", i)
		}
	}
}

// TestCollineProprietaireNeutreEstUneMesure — la valeur neutre du canal publie un intervalle
// SANS camp. « Personne ne la tient » n'est pas « on ne sait pas » : le premier se dessine.
func TestCollineProprietaireNeutreEstUneMesure(t *testing.T) {
	in, c := hillOwnerCase([]filmdec.ManagedPropertyRead{
		zoneReadAt(41, 60, filmdec.ManagedPropertyTagU32, zoneNeutralOwner),
		zoneReadAt(41, 250, filmdec.ManagedPropertyTagU32, 1),
	})
	states, _ := buildZoneStates(in, c)
	got := spansTries(states)
	if len(got) == 0 {
		t.Fatal("aucun intervalle publie")
	}
	if got[0].Owner != nil {
		t.Errorf("premier intervalle : proprietaire %d, attendu nil (valeur neutre du canal)",
			*got[0].Owner)
	}
	// ...et le camp qui suit est bien publie : le neutre n'eteint pas le canal.
	var vus []int
	for _, sp := range got {
		if sp.Owner != nil {
			vus = append(vus, *sp.Owner)
		}
	}
	if len(vus) == 0 {
		t.Error("aucun camp publie apres la valeur neutre — le neutre ne doit pas eteindre le canal")
	}
}

// TestCollineProprietaireUnSeulCampEmis — LE CAS REEL DE `606d9844` ET `8076f97f` : le film ne
// replique qu'UN camp sur le canal. Ce qui precede la premiere emission n'a PAS de camp, et le
// producteur ne le devine pas.
func TestCollineProprietaireUnSeulCampEmis(t *testing.T) {
	in, c := hillOwnerCase([]filmdec.ManagedPropertyRead{
		// DEUX emissions, UN SEUL camp — et la distinction compte : l'election du designateur
		// EXIGE que son voisin porte au moins deux emissions (`hillDesignatorMinOwnerSamples`).
		// Un canal qui ne parlerait qu'une fois n'elirait aucun designateur et retomberait sur
		// les rampes ; ce n'est pas le cas mesure sur `606d9844` (13 emissions) ni sur
		// `8076f97f` (35). Les deux emissions valent le MEME camp : `mergeZoneRuns` n'en fait
		// qu'un seul intervalle, ouvert a 300 — tout ce qui precede reste sans camp.
		zoneReadAt(41, 300, filmdec.ManagedPropertyTagU32, 1),
		zoneReadAt(41, 500, filmdec.ManagedPropertyTagU32, 1),
	})
	states, _ := buildZoneStates(in, c)
	got := spansTries(states)
	if len(got) == 0 {
		t.Fatal("aucun intervalle publie")
	}
	avant, apres := 0, 0
	for _, sp := range got {
		if sp.T1 < 300 {
			if sp.Owner != nil {
				t.Errorf("intervalle [%d ; %d] AVANT la premiere emission : proprietaire %d, "+
					"attendu aucun — le producteur ne devine pas un camp que le canal ne dit pas",
					sp.T0, sp.T1, *sp.Owner)
			}
			avant++
			continue
		}
		if sp.Owner == nil || *sp.Owner != 1 {
			t.Errorf("intervalle [%d ; %d] APRES la premiere emission : proprietaire %v, attendu 1",
				sp.T0, sp.T1, sp.Owner)
		}
		apres++
	}
	if avant == 0 || apres == 0 {
		t.Errorf("%d intervalle(s) avant la premiere emission et %d apres, attendu au moins 1 de "+
			"chaque — sans les deux, le cas ne teste rien", avant, apres)
	}
	// LA COLLINE RESTE ACTIVE MEME SANS CAMP : c'est le comportement d'avant le canal, et il ne
	// doit pas se perdre.
	for _, sp := range got {
		if !sp.Active {
			t.Errorf("intervalle [%d ; %d] non marque ACTIF", sp.T0, sp.T1)
		}
	}
}

// TestCollineProprietaireValeurInconnueNOuvreRien — une valeur qui n'est ni le neutre ni un camp
// connu n'ouvre AUCUN intervalle et se COMPTE. Publier un camp qu'aucun joueur n'occupe serait
// une invention ; le taire empecherait de le voir arriver.
func TestCollineProprietaireValeurInconnueNOuvreRien(t *testing.T) {
	in, c := hillOwnerCase([]filmdec.ManagedPropertyRead{
		zoneReadAt(41, 60, filmdec.ManagedPropertyTagU32, 7),
		zoneReadAt(41, 300, filmdec.ManagedPropertyTagU32, 0),
	})
	states, cov := buildZoneStates(in, c)
	if cov.UnknownOwner != 1 {
		t.Errorf("UnknownOwner = %d, attendu 1 — une valeur hors referentiel doit se compter",
			cov.UnknownOwner)
	}
	for _, sp := range spansTries(states) {
		if sp.Owner != nil && *sp.Owner == 7 {
			t.Errorf("intervalle [%d ; %d] publie le camp 7, qui n'existe pas", sp.T0, sp.T1)
		}
	}
}

// ptr rend un pointeur sur un entier — les camps du DTO sont des pointeurs, parce que le camp 0
// existe et doit se distinguer de « aucun camp ».
func ptr(v int) *int { return &v }
