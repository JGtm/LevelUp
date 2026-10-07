package replay

// zone_states_etat_initial_test.go — L'ETAT D'UNE ZONE AVANT SON PREMIER CHANGEMENT, sur des
// lectures CONSTRUITES (cf. zone_states_etat_initial.go). Aucun film : le temoin reel vit dans
// `zone_etat_initial_temoin_test.go`.

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// bastionSansNeutreDeDepart est le cas nominal de `bastionCase` sans les emissions neutres de la
// frame 0 : chaque canal de propriete n'emet qu'a sa premiere reprise (frame 101 et 201), comme le
// fait le film.
func bastionSansNeutreDeDepart() (ZoneInput, zoneCtx) {
	in, c := bastionCase()
	reads := in.Reads[:0:0]
	for _, r := range in.Reads {
		if r.Tag == grammar.ManagedPropertyTagU32 && r.TimestampUS == 0 {
			continue
		}
		reads = append(reads, r)
	}
	in.Reads = reads
	return in, c
}

// etatDImageCle fabrique un etat d'image-cle du canal de propriete.
func etatDImageCle(slot uint32, frame int, value uint64) grammar.ManagedPropertyRead {
	r := zoneReadAt(slot, frame, grammar.ManagedPropertyTagU32, value)
	r.Chained = true
	return r
}

// spansOfRef rend les intervalles publies de la zone `ref`.
func spansOfRef(t *testing.T, states []ZoneState, ref int) []ZoneSpan {
	t.Helper()
	for _, st := range states {
		if st.ZoneRef == ref {
			return st.Spans
		}
	}
	t.Fatalf("zone %d absente de l'etat publie : %+v", ref, states)
	return nil
}

// TestEtatInitialZonePreTenueOuvreALImageCle : une zone que l'image-cle donne au camp 1 a la
// frame 20, et dont le canal n'emet qu'a sa reprise (frame 101), publie le camp 1 de la frame 20
// a la frame 100 ; la suite est inchangee.
func TestEtatInitialZonePreTenueOuvreALImageCle(t *testing.T) {
	in, c := bastionSansNeutreDeDepart()
	avant, covAvant := buildZoneStates(context.Background(), in, c)
	in.KeyReads = []grammar.ManagedPropertyRead{etatDImageCle(11, 20, 1)}
	apres, covApres := buildZoneStates(context.Background(), in, c)

	sAvant, sApres := spansOfRef(t, avant, 0), spansOfRef(t, apres, 0)
	if sAvant[0].T0 != 101 {
		t.Fatalf("sans image-cle, premier intervalle a %d, attendu 101 (premiere emission)", sAvant[0].T0)
	}
	if len(sApres) != len(sAvant)+1 {
		t.Fatalf("%d intervalle(s) avec l'image-cle, attendu %d", len(sApres), len(sAvant)+1)
	}
	premier := sApres[0]
	if premier.T0 != 20 || premier.T1 != 100 || premier.Owner == nil || *premier.Owner != 1 {
		t.Fatalf("premier intervalle %+v (owner %v), attendu [20, 100] tenu par le camp 1", premier, ownerOf(premier))
	}
	if !reflect.DeepEqual(sApres[1:], sAvant) {
		t.Errorf("les intervalles suivants changent :\n  avant %+v\n  apres %+v", sAvant, sApres[1:])
	}
	if !reflect.DeepEqual(spansOfRef(t, apres, 1), spansOfRef(t, avant, 1)) {
		t.Errorf("la zone sans image-cle change")
	}
	if covAvant.OwnerChecked != covApres.OwnerChecked || covAvant.OwnerAgreed != covApres.OwnerAgreed ||
		covAvant.UnknownOwner != covApres.UnknownOwner || covAvant.Paired != covApres.Paired {
		t.Errorf("la couverture change : avant %+v, apres %+v", covAvant, covApres)
	}
	if covApres.Spans != covAvant.Spans+1 {
		t.Errorf("intervalles publies %d, attendu %d", covApres.Spans, covAvant.Spans+1)
	}
}

// TestEtatInitialZoneNeutreResteInchangee : une image-cle qui dit la zone neutre avant sa
// premiere emission n'ouvre rien.
func TestEtatInitialZoneNeutreResteInchangee(t *testing.T) {
	in, c := bastionSansNeutreDeDepart()
	avant, covAvant := buildZoneStates(context.Background(), in, c)
	in.KeyReads = []grammar.ManagedPropertyRead{etatDImageCle(21, 20, zoneNeutralOwner),
		etatDImageCle(21, 120, zoneNeutralOwner)}
	apres, covApres := buildZoneStates(context.Background(), in, c)
	if !reflect.DeepEqual(apres, avant) || !reflect.DeepEqual(covApres, covAvant) {
		t.Errorf("une zone neutre au depart change :\n  avant %+v\n  apres %+v", avant, apres)
	}
}

// TestEtatInitialSansImageCleRienNeChange : sans etat d'image-cle sur les canaux elus (ici, un
// etat sur un slot qui n'est le canal d'aucune zone), le calque est identique.
func TestEtatInitialSansImageCleRienNeChange(t *testing.T) {
	in, c := bastionSansNeutreDeDepart()
	avant, covAvant := buildZoneStates(context.Background(), in, c)
	in.KeyReads = []grammar.ManagedPropertyRead{etatDImageCle(99, 20, 1)}
	apres, covApres := buildZoneStates(context.Background(), in, c)
	if !reflect.DeepEqual(apres, avant) || !reflect.DeepEqual(covApres, covAvant) {
		t.Errorf("un etat d'image-cle hors canal elu change le calque")
	}
}

// TestEtatInitialImageCleDiscordanteNeChangeRien : une image-cle POSTERIEURE a la premiere
// emission ne touche pas aux intervalles ; son desaccord avec l'etat reconstitue se compte.
func TestEtatInitialImageCleDiscordanteNeChangeRien(t *testing.T) {
	in, c := bastionSansNeutreDeDepart()
	avant, _ := buildZoneStates(context.Background(), in, c)
	in.KeyReads = []grammar.ManagedPropertyRead{etatDImageCle(11, 250, 1)}
	apres, _ := buildZoneStates(context.Background(), in, c)
	if !reflect.DeepEqual(apres, avant) {
		t.Errorf("une image-cle posterieure a la premiere emission change les intervalles")
	}

	delta := []zoneSample{{t: 101, v: 0}, {t: 301, v: 1}}
	var k zoneKeyTally
	tallyKeyAgreement(delta, []zoneSample{{t: 250, v: 1}, {t: 350, v: 1}, {t: 301, v: 0}, {t: 50, v: 1}}, &k)
	if k.checked != 2 || k.agreed != 1 {
		t.Errorf("controle %d compare(s) / %d concordant(s), attendu 2 / 1 (250 discordante, 350 "+
			"concordante, 301 a la frame d'une emission et 50 avant la premiere ne se comparent pas)",
			k.checked, k.agreed)
	}
}

// TestEtatInitialValeurInconnueEcarteLEtatInitial : un etat d'image-cle qui n'est ni neutre ni un
// camp du roster ecarte l'etat initial, sans compter de valeur inconnue.
func TestEtatInitialValeurInconnueEcarteLEtatInitial(t *testing.T) {
	in, c := bastionSansNeutreDeDepart()
	avant, covAvant := buildZoneStates(context.Background(), in, c)
	in.KeyReads = []grammar.ManagedPropertyRead{etatDImageCle(11, 20, 7)}
	apres, covApres := buildZoneStates(context.Background(), in, c)
	if !reflect.DeepEqual(apres, avant) || !reflect.DeepEqual(covApres, covAvant) {
		t.Errorf("une valeur hors roster ouvre un intervalle ou se compte")
	}
}

// TestEtatInitialImageCleAvantLOrigine : une image-cle anterieure a l'origine de l'axe se pose a
// la frame 0 — la DERNIERE avant l'origine, et seulement si aucune emission du canal ne tombe
// entre elle et l'origine. Jamais avant la frame 0, jamais une frame inventee.
func TestEtatInitialImageCleAvantLOrigine(t *testing.T) {
	c := zoneCtx{origin: 1_000_000, step: 100_000, frames: 600}
	at := func(slot uint32, ts uint64, v uint64) grammar.ManagedPropertyRead {
		return grammar.ManagedPropertyRead{Slot: slot, TimestampUS: ts, Field: grammar.ManagedPropertyScalar,
			FilmIndex: -1, Tag: grammar.ManagedPropertyTagU32, Value: v, HasValue: true, Chained: true}
	}
	key := []grammar.ManagedPropertyRead{
		at(1, 200_000, 0), at(1, 600_000, 1), at(1, 1_000_000, 1), at(1, 3_000_000, 1), // slot 1
		at(2, 500_000, 1),     // slot 2 : une emission delta suit avant l'origine
		at(3, 999_999_999, 1), // slot 3 : au-dela de l'axe
	}
	reads := []grammar.ManagedPropertyRead{at(2, 800_000, 0)}
	got := zoneKeyOwnerOf(key, reads, c)
	want1 := []zoneSample{{t: 0, v: 1}, {t: 0, v: 1}, {t: 20, v: 1}}
	if !reflect.DeepEqual(got[1], want1) {
		t.Errorf("slot 1 : %+v, attendu %+v (la derniere image-cle avant l'origine, a la frame 0, "+
			"puis celles de l'axe)", got[1], want1)
	}
	if len(got[2]) != 0 {
		t.Errorf("slot 2 : %+v — une emission delta entre l'image-cle et l'origine la perime", got[2])
	}
	if len(got[3]) != 0 {
		t.Errorf("slot 3 : %+v — une image-cle hors de l'axe est ecartee", got[3])
	}
}

// ownerOf rend le camp d'un intervalle pour un message, -1 pour personne.
func ownerOf(s ZoneSpan) int {
	if s.Owner == nil {
		return -1
	}
	return *s.Owner
}
