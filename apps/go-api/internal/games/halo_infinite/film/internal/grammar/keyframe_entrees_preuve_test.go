package grammar

// keyframe_entrees_preuve_test.go — LA PREUVE D IMAGE-CLE LIT LES SLOTS LIBERES QUI SUIVENT UN RECORD
// (plan `.ai/V7.5/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, R1.3), sur les octets reels des bobines par build.

import "testing"

// recordProuveSurEntreesLiberees est un record que la preuve du film prouve alors que des entrees sans
// archetype le separent du record suivant.
type recordProuveSurEntreesLiberees struct {
	film       string
	pay        []byte
	bit, slot  int
	liberees   int
	preuve     *PreuveDImageCle
	slotSuivie int
}

// premierRecordProuveSurEntreesLiberees cherche, sur les bobines par build, le premier record de la
// marche du film que la preuve prouve et que suivent au moins une entree sans archetype.
func premierRecordProuveSurEntreesLiberees(t *testing.T) (recordProuveSurEntreesLiberees, int) {
	t.Helper()
	trouves := 0
	var premier recordProuveSurEntreesLiberees
	for _, court := range closureMiniFilms() {
		fc := NewFilmContext(bobineParBuild(t, court))
		pv, reg := fc.PreuveDImageCle(), (*Registry)(nil)
		if pv == nil {
			continue
		}
		reg = pv.reg
		for p, err := range fc.ImagesCles() {
			if err != nil {
				t.Fatalf("%s : %v", court, err)
			}
			total, enTete := len(p.Payload)*8, pv.ctx.Profil.Cadre.EnTeteBits
			for _, r := range p.Records {
				if !pv.prouve(p.Payload, int(r.Debut), int(r.Vie.Slot)) {
					continue
				}
				tr := WalkKeyframeFullState(p.Payload, int(r.Debut), reg, pv.ctx)
				pos, k := sauterLesEntreesSansCorps(p.Payload, tr.EndBit, total, enTete)
				if k == 0 {
					continue
				}
				h, _ := readKeyframeHeader(p.Payload, pos, total)
				if trouves++; trouves == 1 {
					premier = recordProuveSurEntreesLiberees{film: court, pay: p.Payload, bit: int(r.Debut),
						slot: int(r.Vie.Slot), liberees: k, preuve: pv, slotSuivie: h.Slot}
				}
			}
		}
	}
	return premier, trouves
}

// TestLaPreuveLitLesSlotsLiberesQuiSuivent : sur les bobines, des records sont prouves par-dessus des
// entrees sans archetype, et le record qui les suit porte exactement le slot `slot + 1 + k` ; le meme
// record, pris pour un slot de plus, ne se prouve plus (l ecart de slots est le nombre d entrees).
// MUTATIONS — ne plus sauter les entrees sans archetype, ou comparer le slot suivant par `>` : ROUGE.
func TestLaPreuveLitLesSlotsLiberesQuiSuivent(t *testing.T) {
	r, n := premierRecordProuveSurEntreesLiberees(t)
	if n == 0 {
		t.Fatal("aucun record prouve par-dessus des slots liberes sur les sept bobines")
	}
	if r.slotSuivie != r.slot+1+r.liberees {
		t.Errorf("%s bit %d : slot %d, %d entree(s) liberee(s), record suivant au slot %d", r.film, r.bit,
			r.slot, r.liberees, r.slotSuivie)
	}
	if r.preuve.prouve(r.pay, r.bit, r.slot+1) {
		t.Errorf("%s bit %d : prouve aussi au slot %d — l ecart de slots n est plus lu", r.film, r.bit, r.slot+1)
	}
	t.Logf("%d record(s) prouves par-dessus des slots liberes ; premier : %s bit %d slot %d, %d entree(s)",
		n, r.film, r.bit, r.slot, r.liberees)
}
