//go:build research

package grammar

// va_v2_corr_research_test.go — LOT VA, ETAPE V2, CORRECTIONS DU CONTROLE (2026-10-05) : LES
// LECTURES DE LA MARCHE DES MORTS QUE LA FIN DE LA VUE A CHANGE, INSTRUITES PAQUET PAR PAQUET.
//
// Par film (carte du catalogue, contexte de cuisson, largeurs MPP relues posees comme l etape
// `vehicles` du rejeu les pose), chaque paquet a evenements dont le debut de la marche des morts
// change — la fin de la vue A E contre le localisateur S, sur le MEME monde : la marche restaure le
// monde apres chaque paquet ([marchRecordsOf]) — est marche depuis les deux debuts. Chaque mort
// `ti=40` et chaque lecture d occupation qui n appartient qu a l une des deux marches est ecrite
// avec le bit de son record (avant E : des bits que l ecrivain a ecrits dans la vue A), la position
// de S contre E, l arret de la marche depuis E, et si la vie (slot, generation) est relue ailleurs
// par la marche de production.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	VA_CATALOGUE=<map_quant_bounds.json> VA_CARTES="id=Carte;..." VA_PROFILS=<profils> \
//	  go test -tags=research -count=1 -run '^TestVAV2CorrMorts$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// vaLecture est une lecture de la marche des morts publiee par l etape `vehicles` : sa cle
// (genre, instant, slot, generation, valeur) et le bit de l en-tete de son record.
type vaLecture struct {
	cle, vie string
	bit      int
}

// vaLecturesDe rend les morts `ti=40` acceptees et les lectures d occupation de `recs`.
func vaLecturesDe(h *objectDeathHarvest, recs []FrameRecord, at uint64) []vaLecture {
	var out []vaLecture
	for i := range recs {
		r := &recs[i]
		if o, ok := occupancyFromRecord(r, at); ok {
			out = append(out, vaLecture{fmt.Sprintf("occupation\t%d\t%d\t%d\t%+v", at, o.Slot, o.Gen, o),
				fmt.Sprintf("o%d:%d", o.Slot, o.Gen), r.HeaderBit})
		}
		if r.TypeIndex != uint32(VehicleTypeIndex) || r.Trace.Dead == nil || !r.Trace.Dead.Mort {
			continue
		}
		if _, ok := h.accept(r); ok {
			out = append(out, vaLecture{fmt.Sprintf("mort\t%d\t%d\t%d\t%+v", at, r.Slot, r.ID>>30, *r.Trace.Dead),
				fmt.Sprintf("m%d:%d", r.Slot, r.ID>>30), r.HeaderBit})
		}
	}
	return out
}

// vaArretDeLaMarche decrit ou la marche partie de `debut` s arrete : records lus, fin du dernier,
// son archetype et le composant de sa desynchronisation ; et si un record y commence au bit `s`.
func vaArretDeLaMarche(reg *Registry, recs []FrameRecord, s int) string {
	traverse := false
	for i := range recs {
		traverse = traverse || recs[i].HeaderBit == s
	}
	if len(recs) == 0 {
		return rnTab(0, -1, "-", "-", traverse)
	}
	r := &recs[len(recs)-1]
	comp := "-"
	if a, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt >= 0 && r.DesyncAt < len(a.Components) {
		comp = fmt.Sprintf("i%d %s", r.DesyncAt, a.Components[r.DesyncAt])
	}
	return rnTab(len(recs), r.FinBit, fmt.Sprintf("ti=%d slot %d", r.TypeIndex, r.Slot), comp, traverse)
}

// vaViesDe rend les vies (slot, generation) de toutes les lectures d une marche complete.
func vaViesDe(f MarchFacts) map[string]bool {
	out := map[string]bool{}
	for _, d := range f.Deaths {
		if d.TypeIndex == uint32(VehicleTypeIndex) {
			out[fmt.Sprintf("m%d:%d", d.Slot, d.Gen)] = true
		}
	}
	for _, o := range f.Occupancy {
		out[fmt.Sprintf("o%d:%d", o.Slot, o.Gen)] = true
	}
	return out
}

// vaCorrMortsDuFilm rend les lignes d un film et journalise ses comptes publies (base, V2).
func vaCorrMortsDuFilm(t *testing.T, id string, fc *FilmContext) []string {
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	avec, err1 := ScanMarchFacts(fc)
	sans, err2 := vaMarchFactsSans(fc, VueADuFilm{})
	if err1 != nil || err2 != nil {
		t.Fatalf("%s : %v / %v", id, err1, err2)
	}
	ma, oa := vaLignesDeVehicules(avec)
	ms, osa := vaLignesDeVehicules(sans)
	t.Logf("%s : morts ti=40 %d -> %d (retirees %d, ajoutees %d) ; occupations %d -> %d (retirees %d, ajoutees %d)",
		id, len(ms), len(ma), len(vaDiffDesLignes(ms, ma)), len(vaDiffDesLignes(ma, ms)), len(osa), len(oa),
		len(vaDiffDesLignes(osa, oa)), len(vaDiffDesLignes(oa, osa)))
	viesV2 := vaViesDe(avec)
	kfs, deltas := marchPacketsOf(fc)
	cfg, _, _, _ := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	h := &objectDeathHarvest{reg: reg, idx: map[uint32]int{}, st: &ObjectDeathStats{}}
	tl := newMarchTimeline(reg, kfs)
	g := fc.grammaireDeLaVueA()
	var lignes []string
	for _, d := range deltas {
		w := tl.advanceTo(d.timestampUS)
		if !marchHasEvents(d.payload) {
			continue
		}
		e, _ := DebutDeLaVueB(d.payload, w, cfg, VueADuFilm{g: g})
		s, _ := LocaliserBoucleDeRecords(d.payload, w, cfg, SignaturePuisLargeurLibre)
		if e == s {
			continue
		}
		var recsE, recsS []FrameRecord
		if e >= 0 {
			recsE = marchRecordsOf(d.payload, w, cfg, e)
		}
		if s >= 0 {
			recsS = marchRecordsOf(d.payload, w, cfg, s)
		}
		le, ls := vaLecturesDe(h, recsE, d.timestampUS), vaLecturesDe(h, recsS, d.timestampUS)
		p := vaPaquetCorr{id: id, e: e, s: s, arret: vaArretDeLaMarche(reg, recsE, s), vies: viesV2}
		lignes = append(lignes, p.seulement("retiree", ls, le)...)
		lignes = append(lignes, p.seulement("ajoutee", le, ls)...)
	}
	return lignes
}

// vaPaquetCorr porte ce qu une ligne dit du paquet : film, debuts E et S, arret de la marche depuis
// E, vies relues par la marche de production.
type vaPaquetCorr struct {
	id    string
	e, s  int
	arret string
	vies  map[string]bool
}

// seulement ecrit les lectures de `a` absentes de `b`.
func (p vaPaquetCorr) seulement(cote string, a, b []vaLecture) []string {
	dans := map[string]int{}
	for _, x := range b {
		dans[x.cle]++
	}
	var out []string
	for _, x := range a {
		if dans[x.cle] > 0 {
			dans[x.cle]--
			continue
		}
		pos := "S apres E"
		switch {
		case p.s < 0:
			pos = "S absent"
		case p.s < p.e:
			pos = "S avant E"
		}
		lieu := "vue B"
		if x.bit < p.e {
			lieu = "vue A"
		}
		out = append(out, rnTab(p.id, cote, x.cle, x.bit, lieu, p.e, p.s, pos, p.arret, p.vies[x.vie]))
	}
	return out
}

// vaPoserLeProfilDeLaCuisson pose sur `fc`, comme `replay.poserProfilPuisCarte`, le profil que
// `killsource` a calibre sur le film (VA_PROFILS/<id>.profil.json, ecrit par
// `killsource/va_v2_profil_research_test.go`), puis les largeurs d objet du monde de la carte.
// Sans VA_PROFILS, le contexte garde son profil.
func vaPoserLeProfilDeLaCuisson(t *testing.T, id string, fc *FilmContext) {
	rep := os.Getenv("VA_PROFILS")
	if rep == "" {
		return
	}
	blob, err := os.ReadFile(filepath.Join(rep, id+".profil.json"))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	var p ProfilDeBalayage
	if err := json.Unmarshal(blob, &p); err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	fc.PoserProfilDeBalayage(p)
	bal := fc.ProfilDeBalayage()
	fc.NoterReplis(bal.PoserLargeursObjetDuMondeDepuisDecoupage(fc.Profile().Map().Layout()))
	fc.PoserProfilDeBalayage(bal)
}

// TestVAV2CorrMorts : par film de CAMPAGNE_FILMS, les lectures de la marche des morts que la fin de
// la vue A change, instruites (cf. l en-tete).
func TestVAV2CorrMorts(t *testing.T) {
	racine, sortie, films := b2Env(t)
	lignes := []string{"film\tcote\tlecture\tinstant_us\tslot\tgen\tvaleur\tbit_record\tlieu\tE\tS\tS_contre_E\t" +
		"records_depuis_E\tfin_du_dernier\tdernier\tdesync\ttraverse_S\tvie_relue_en_V2"}
	for _, id := range films {
		e, ok := vaCarte(t, id)
		if !ok {
			t.Fatalf("%s : carte inconnue", id)
		}
		film, err := source.LoadDir(filepath.Join(racine, id), nil)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		fc := NewFilmContextForMap(film, &e, DefaultScanFilmOptions().Layout)
		vaPoserLeProfilDeLaCuisson(t, id, fc)
		if res := MPPWidthsForFilm(fc.Film()); res.Relue() {
			fc.PoserMPP(res.Widths)
		} else {
			t.Logf("%s : largeurs MPP non relues, cadre du contexte garde", id)
		}
		lignes = append(lignes, vaCorrMortsDuFilm(t, id, fc)...)
	}
	b2Ecrire(t, sortie, "va_v2_corr_morts.tsv", lignes)
}
