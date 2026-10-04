//go:build research

package grammar

// morts_marche_unique_research_test.go — INSTRUMENT : CE QUE LA MARCHE DES TRAMES REND DES MORTS
// D OBJET ET DE L OCCUPATION, A COTE DE LA MARCHE A HUIT VUES (mesure prealable du lot 2.7.a de la
// representation intermediaire).
//
// Film par film, sous le contexte de la cuisson (entree de carte du catalogue) :
//  1. la marche des morts d objet ([ScanMarchFacts]) : le cadre calibre et ses denominateurs, les
//     morts par archetype, l occupation ;
//  2. la marche des trames (trois vues par rangs, monde de la phase des images-cles, `IDLowBits`
//     de l en-tete) : les memes recoltes, sous la meme regle d acceptation ([objectDeathHarvest]) ;
//  3. la difference des deux ensembles — morts de vehicule, morts de bipede, occupation : communes,
//     propres a chaque marche, et quelques exemples des morts de vehicule propres a l une.
//
//	FILM_CACHE_ROOT=<depot>/data/cache MORTS_FILMS='084a804d=Fortitude Heavies;a349fea8=Fragmentation Heavies' \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/grammar/ -run MortsMarcheUnique -v

import (
	"cmp"
	"os"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestMortsMarcheUnique imprime, film par film, les trois mesures de l en-tete.
func TestMortsMarcheUnique(t *testing.T) {
	racine, liste := os.Getenv("FILM_CACHE_ROOT"), os.Getenv("MORTS_FILMS")
	if racine == "" || liste == "" {
		t.Skip("FILM_CACHE_ROOT ou MORTS_FILMS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		fcHuit := contexteDesMortsDeLaCuisson(t, id, film, &entree)
		huit, err := ScanMarchFacts(fcHuit)
		if err != nil {
			t.Fatalf("%s : marche a huit vues : %v", id, err)
		}
		trois, evenements, localises := recolteDeLaMarcheDesTrames(t, contexteDeLaCuisson(film, &entree), false)
		st := huit.Stats
		t.Logf("%s : cadre calibre idLow=%d (par defaut %v ; localises %d contre %d au dauphin, sur %d) ; "+
			"en-tete idLow=%d", id, st.Config.IDLowBits, st.CadreParDefaut, st.CadreLocalises, st.CadreDauphin,
			st.CadreEvenements, idLowBitsPresume)
		t.Logf("%s : paquets a evenements localises : huit vues %d/%d, trames %d/%d", id, st.LocatedPackets,
			st.EventPackets, localises, evenements)
		for _, ti := range []uint32{uint32(VehicleTypeIndex), uint32(BipedTypeIndex)} {
			communes, seulHuit, seulTrois := diffDesMorts(huit.Deaths, trois.Deaths, ti)
			t.Logf("%s : morts ti=%d : huit vues %d, trames %d — communes %d, huit seules %d, trames seules %d "+
				"(masques declarant : huit %d, trames %d)", id, ti, compteDe(huit.Deaths, ti), compteDe(trois.Deaths, ti),
				communes, len(seulHuit), len(seulTrois), st.MaskDeclared[ti], trois.Stats.MaskDeclared[ti])
			if ti == uint32(VehicleTypeIndex) {
				exemplesDeMorts(t, id, "huit seules", seulHuit)
				exemplesDeMorts(t, id, "trames seules", seulTrois)
				vies := viesDesVehicules(fcHuit)
				for _, s := range []struct {
					nom string
					ds  []types.ObjectDeath
				}{{"huit vues", huit.Deaths}, {"trames", trois.Deaths}, {"huit seules", seulHuit}, {"trames seules", seulTrois}} {
					var dsTI []types.ObjectDeath
					for _, d := range s.ds {
						if d.TypeIndex == ti {
							dsTI = append(dsTI, d)
						}
					}
					dans, fin := confirmees(vies, dsTI)
					t.Logf("%s :   confirmation par le recensement (%d vies) : %s %d, dans une vie %d, a moins d une "+
						"minute de sa fin %d", id, len(vies), s.nom, len(dsTI), dans, fin)
				}
			}
		}
		communes, seulHuit, seulTrois := diffDeLOccupation(huit.Occupancy, trois.Occupancy)
		t.Logf("%s : occupation : huit vues %d, trames %d — communes %d, huit seules %d, trames seules %d", id,
			len(huit.Occupancy), len(trois.Occupancy), communes, seulHuit, seulTrois)
		// Variantes : la marche des trames sous les largeurs MPP des vehicules, sans puis avec la
		// recuperation des listes qu elle ne localise pas.
		vies := viesDesVehicules(fcHuit)
		for _, recuperer := range []bool{false, true} {
			fcMPP := contexteDeLaCuisson(film, &entree)
			fcMPP.PoserMPP(fcHuit.ProfilDeBalayage().MPP)
			tMPP, _, _ := recolteDeLaMarcheDesTrames(t, fcMPP, recuperer)
			vti := uint32(VehicleTypeIndex)
			c2, h2, t2 := diffDesMorts(huit.Deaths, tMPP.Deaths, vti)
			oc, oh, ot := diffDeLOccupation(huit.Occupancy, tMPP.Occupancy)
			d2, f2 := confirmees(vies, filtrerTI(t2, vti))
			dh, fh := confirmees(vies, filtrerTI(h2, vti))
			t.Logf("%s : trames aux largeurs MPP des vehicules (recuperation des listes non localisees %v) : morts "+
				"ti=40 %d (communes %d, huit seules %d [confirmees %d/%d], trames seules %d [confirmees %d/%d]) ; "+
				"occupation %d (communes %d, huit seules %d, trames seules %d)", id, recuperer, compteDe(tMPP.Deaths, vti),
				c2, len(h2), dh, fh, len(t2), d2, f2, len(tMPP.Occupancy), oc, oh, ot)
		}
	}
}

// contexteDesMortsDeLaCuisson ouvre le contexte sous lequel la cuisson marche les morts : les
// largeurs MPP des vehicules, celles de la version de format quand la grammaire les relit, sinon
// celles que la calibration des poses mesure sur le film (`replay.gwWidthsForFilm`).
func contexteDesMortsDeLaCuisson(t *testing.T, id string, film *source.Film, entree *profile.MapQuantEntry) *FilmContext {
	t.Helper()
	fc := contexteDeLaCuisson(film, entree)
	if res := MPPWidthsForFilm(film); res.Relue() {
		fc.PoserMPP(res.Widths)
		t.Logf("%s : largeurs MPP relues au format %d : %s", id, res.FormatVersion, res.Widths.String())
		return fc
	}
	wr := entree.Range()
	_, pst, err := ScanEquipmentPlacements(fc, &wr)
	if err == nil && pst.Calibration.Widths.Valid() {
		fc.PoserMPP(pst.Calibration.Widths)
	}
	t.Logf("%s : format sans largeurs relues ; calibration des poses : %s (err %v)", id,
		pst.Calibration.Widths.String(), err)
	return fc
}

// recolteDeLaMarcheDesTrames recolte morts et occupation sur les records de la marche des trames,
// sous la regle de la marche des morts, et compte ses paquets a evenements et ceux qu elle localise.
func recolteDeLaMarcheDesTrames(t *testing.T, fc *FilmContext, recuperer bool) (MarchFacts, int, int) {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	m, err := fc.nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	st := newObjectDeathStats()
	h := &objectDeathHarvest{reg: reg, idx: map[uint32]int{}, st: &st}
	evenements, localises := 0, 0
	m.parcourir(func(tr *trameLue) bool {
		if tr.paquet.VueA.Etat == lecture.VueArretee {
			evenements++
			if tr.debut >= 0 {
				localises++
			} else if recuperer {
				h.harvest(recupererLaListe(tr.paquet.Payload, m.monde, m.cfg), tr.paquet.TS)
			}
		}
		h.harvest(tr.lecture.recs, tr.paquet.TS)
		return true
	})
	return MarchFacts{Deaths: dedupObjectDeaths(h.out), Occupancy: dedupOccupancy(h.rides), Stats: st},
		evenements, localises
}

// recupererLaListe lit la vue B d une liste que la marche des trames n a pas localisee, depuis le
// debut que le localisateur unique trouve dans l ordre des marches qui lisent les morts (signature,
// puis largeur libre), sous le monde de la marche, rendu intact.
func recupererLaListe(pay []byte, w *World, cfg FrameConfig) []FrameRecord {
	s, _ := LocaliserBoucleDeRecords(pay, w, cfg, SignaturePuisLargeurLibre)
	if s < 0 {
		return nil
	}
	snap := w.Snapshot()
	defer w.Restore(snap)
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.Skip(s)
	recs, _ := DecodeFrameRecords(br, w, cfg)
	return recs
}

// cleDeMort identifie une mort : l entite et l instant du paquet qui la porte.
type cleDeMort struct {
	slot, gen uint32
	at        uint64
}

// diffDesMorts rend, pour l archetype `ti`, le nombre de morts communes et celles propres a chaque
// marche.
func diffDesMorts(a, b []types.ObjectDeath, ti uint32) (int, []types.ObjectDeath, []types.ObjectDeath) {
	dans := func(ds []types.ObjectDeath) map[cleDeMort]types.ObjectDeath {
		out := map[cleDeMort]types.ObjectDeath{}
		for _, d := range ds {
			if d.TypeIndex == ti {
				out[cleDeMort{d.Slot, d.Gen, d.TimestampUS}] = d
			}
		}
		return out
	}
	ma, mb := dans(a), dans(b)
	communes := 0
	var seulA, seulB []types.ObjectDeath
	for k, d := range ma {
		if _, ok := mb[k]; ok {
			communes++
		} else {
			seulA = append(seulA, d)
		}
	}
	for k, d := range mb {
		if _, ok := ma[k]; !ok {
			seulB = append(seulB, d)
		}
	}
	return communes, seulA, seulB
}

// compteDe rend le nombre de morts de l archetype `ti`.
func compteDe(ds []types.ObjectDeath, ti uint32) int {
	n := 0
	for _, d := range ds {
		if d.TypeIndex == ti {
			n++
		}
	}
	return n
}

// exemplesDeMorts imprime au plus cinq morts de la liste.
func exemplesDeMorts(t *testing.T, id, nom string, ds []types.ObjectDeath) {
	t.Helper()
	for i, d := range ds {
		if i == 5 {
			t.Logf("%s :   %s : ... %d de plus", id, nom, len(ds)-5)
			return
		}
		t.Logf("%s :   %s : slot %d gen %d a %d us (queue inconnue %v)", id, nom, d.Slot, d.Gen, d.TimestampUS,
			d.TailDesync)
	}
}

// diffDeLOccupation rend le nombre de lectures d occupation communes et propres a chaque marche.
func diffDeLOccupation(a, b []types.VehicleOccupancy) (int, int, int) {
	dans := func(lectures []types.VehicleOccupancy) map[types.VehicleOccupancy]bool {
		out := map[types.VehicleOccupancy]bool{}
		for _, o := range lectures {
			out[o] = true
		}
		return out
	}
	ma, mb := dans(a), dans(b)
	communes := 0
	for k := range ma {
		if mb[k] {
			communes++
		}
	}
	return communes, len(ma) - communes, len(mb) - communes
}

// TestMortsMarcheUniqueDiagnostic : pour chaque mort de vehicule que la marche a huit vues lit et
// que la marche des trames ne lit pas, la vue ou la marche a huit vues l a trouvee, et ce que la
// marche des trames a fait du meme paquet (localisation, sortie de la vue B, bit d arret). Un film :
//
//	FILM_CACHE_ROOT=<depot>/data/cache MORTS_DIAG='084a804d=Fortitude Heavies' \
//	  go test -tags research ./internal/games/halo_infinite/film/internal/grammar/ -run MortsMarcheUniqueDiagnostic -v
func TestMortsMarcheUniqueDiagnostic(t *testing.T) {
	racine, couple := os.Getenv("FILM_CACHE_ROOT"), os.Getenv("MORTS_DIAG")
	if racine == "" || couple == "" {
		t.Skip("FILM_CACHE_ROOT ou MORTS_DIAG absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	id, carte, _ := strings.Cut(couple, "=")
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("%s : carte %q : %v", id, carte, err)
	}
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film %s : %v (present %v)", id, err, ok)
	}
	huit := mortsParVue(t, contexteDesMortsDeLaCuisson(t, id, film, &entree))
	trames := etatDesTrames(t, contexteDeLaCuisson(film, &entree))
	parVue, parCas := map[int]int{}, map[string]int{}
	for k, m := range huit {
		if m.mort.TypeIndex != uint32(VehicleTypeIndex) {
			continue
		}
		tr, vu := trames[k.at]
		if vu && tr.morts[k] {
			continue
		}
		parVue[m.vue]++
		cas := "paquet absent de la marche des trames"
		switch {
		case !vu:
		case tr.debut < 0:
			cas = "liste non localisee par la marche des trames"
		case tr.debut != m.debut:
			cas = "debut different"
		case m.bit >= tr.finVueB && tr.finVueB >= 0:
			cas = "record au-dela de la fin de la vue B (" + tr.sortie + ")"
		default:
			cas = "record dans la vue B, non lu"
		}
		parCas[cas]++
		if parCas[cas] <= 3 {
			t.Logf("%s : slot %d gen %d a %d : vue %d, bit %d (debut huit vues %d) ; trames : debut %d, fin de vue B %d "+
				"(%s), %d record(s), cas %q", id, k.slot, k.gen, k.at, m.vue, m.bit, m.debut, tr.debut, tr.finVueB,
				tr.sortie, tr.records, cas)
		}
	}
	t.Logf("%s : morts de vehicule des huit vues absentes des trames, par vue : %v", id, parVue)
	t.Logf("%s : par cas : %v", id, parCas)
}

// mortVue : une mort lue par la marche a huit vues, la vue qui la porte, le bit de son record et
// le debut de la marche de son paquet.
type mortVue struct {
	mort       types.ObjectDeath
	vue        int
	bit, debut int
}

// mortsParVue rejoue [ScanMarchFacts] en gardant, pour chaque mort acceptee, la vue qui la porte.
func mortsParVue(t *testing.T, fc *FilmContext) map[cleDeMort]mortVue {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	kfs, deltas := marchPacketsOf(fc)
	cfg, _, _, _ := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	st := newObjectDeathStats()
	h := &objectDeathHarvest{reg: reg, idx: map[uint32]int{}, st: &st}
	tl := newMarchTimeline(reg, kfs)
	out := map[cleDeMort]mortVue{}
	for _, d := range deltas {
		w := tl.advanceTo(d.timestampUS)
		start, _, ok, _ := marchDebut(d.payload, w, cfg)
		if !ok {
			continue
		}
		snap := w.Snapshot()
		br := LecteurSur(d.payload)
		br.poserCadre(cfg)
		br.Skip(start)
		for v := 0; v < marchViews && br.Remaining() >= 8; v++ {
			recs, err := DecodeFrameRecords(br, w, cfg)
			for i := range recs {
				r := &recs[i]
				if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
					continue
				}
				tail, ok := h.accept(r)
				if !ok {
					continue
				}
				k := cleDeMort{r.Slot, r.ID >> 30, d.timestampUS}
				if _, deja := out[k]; !deja {
					out[k] = mortVue{mort: types.ObjectDeath{TimestampUS: d.timestampUS, Slot: r.Slot, Gen: r.ID >> 30,
						TypeIndex: r.TypeIndex, TailDesync: tail}, vue: v, bit: r.HeaderBit, debut: start}
				}
			}
			if err != nil {
				break
			}
		}
		w.Restore(snap)
	}
	return out
}

// trameVue : ce que la marche des trames a fait d un paquet.
type trameVue struct {
	debut, finVueB, records int
	sortie                  string
	morts                   map[cleDeMort]bool
}

// etatDesTrames marche les trames et rend, par instant de paquet, ce que la marche en a fait.
func etatDesTrames(t *testing.T, fc *FilmContext) map[uint64]trameVue {
	t.Helper()
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	m, err := fc.nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	st := newObjectDeathStats()
	out := map[uint64]trameVue{}
	m.parcourir(func(tr *trameLue) bool {
		h := &objectDeathHarvest{reg: reg, idx: map[uint32]int{}, st: &st}
		h.harvest(tr.lecture.recs, tr.paquet.TS)
		tv := trameVue{debut: tr.debut, finVueB: tr.lecture.finVueB, records: len(tr.lecture.recs),
			sortie: nomDeSortie(tr.lecture.sortieVueB), morts: map[cleDeMort]bool{}}
		for _, d := range h.out {
			tv.morts[cleDeMort{d.Slot, d.Gen, d.TimestampUS}] = true
		}
		out[tr.paquet.TS] = tv
		return true
	})
	return out
}

// nomDeSortie nomme une sortie de la vue B.
func nomDeSortie(s lecture.SortieVueB) string {
	return [...]string{"non atteinte", "terminateur", "rejet hors datum", "rejet de vue", "record infranchissable",
		"fin de payload", "plafond"}[s]
}

// vieRecensee : une vie de vehicule du recensement des images-cles et sa fenetre, sous la regle de
// la cuisson (`replay.assignVehicleWindows` : 20 s de tolerance avant le premier recensement, la
// premiere image-cle qui ne la recense plus, la frontiere de la vie suivante du meme slot).
type vieRecensee struct {
	cle                  types.LifeKey
	premier, fin, lo, hi uint64
}

// viesDesVehicules rend les vies de vehicule du film, triees par slot puis premier recensement.
func viesDesVehicules(fc *FilmContext) []vieRecensee {
	kf := ScanWorldObjectKeyframes(fc, VehicleTypeIndex)
	var out []vieRecensee
	for cle, vus := range kf.SeenUS {
		if len(vus) == 0 {
			continue
		}
		v := vieRecensee{cle: cle, premier: vus[0]}
		for _, t := range kf.TimesUS {
			if t > vus[len(vus)-1] {
				v.fin = t
				break
			}
		}
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b vieRecensee) int {
		return cmp.Or(cmp.Compare(a.cle.Slot, b.cle.Slot), cmp.Compare(a.premier, b.premier),
			cmp.Compare(a.cle.Gen, b.cle.Gen))
	})
	for i := range out {
		v := &out[i]
		v.lo, v.hi = v.premier-min(v.premier, 20_000_000), v.fin
		if v.hi == 0 {
			v.hi = ^uint64(0)
		}
		if i > 0 && out[i-1].cle.Slot == v.cle.Slot && out[i-1].hi > v.lo {
			v.lo = out[i-1].hi
		}
		if i+1 < len(out) && out[i+1].cle.Slot == v.cle.Slot && v.hi > out[i+1].premier {
			v.hi = out[i+1].premier
		}
	}
	return out
}

// confirmees compte les morts qui tombent dans la fenetre d une vie de meme (slot, gen), et parmi
// elles celles qui precedent la fin de cette vie de moins d une minute.
func confirmees(vies []vieRecensee, ds []types.ObjectDeath) (dansUneVie, presDeLaFin int) {
	for _, d := range ds {
		for _, v := range vies {
			if v.cle.Slot != d.Slot || v.cle.Gen != d.Gen || d.TimestampUS < v.lo || d.TimestampUS > v.hi {
				continue
			}
			dansUneVie++
			if v.fin != 0 && v.fin >= d.TimestampUS && v.fin-d.TimestampUS < 60_000_000 {
				presDeLaFin++
			}
			break
		}
	}
	return dansUneVie, presDeLaFin
}

// contexteDeLaCuisson ouvre le contexte d un film comme la cuisson le pose : le profil de balayage
// que `killsource` calibre (`ProfilDeDepartPourCarte` — generation stricte, largeurs world-object de
// la carte — puis le controle de corruption du film ; le mot de poignee reste l invariant, non
// discrimine sur les films a vehicules du corpus), puis les largeurs de la carte
// (`replay.installWorldObjectPrecision`).
func contexteDeLaCuisson(film *source.Film, entree *profile.MapQuantEntry) *FilmContext {
	fc := NewFilmContextForMap(film, entree, nil)
	p := ProfilDeBalayageParDefaut()
	p.Grammaire.GenerationStricte = true
	_ = p.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	p, _ = GrammaireSousFilm(p, film)
	_ = p.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(p)
	return fc
}

// filtrerTI rend les morts de l archetype `ti`.
func filtrerTI(ds []types.ObjectDeath, ti uint32) []types.ObjectDeath {
	var out []types.ObjectDeath
	for _, d := range ds {
		if d.TypeIndex == ti {
			out = append(out, d)
		}
	}
	return out
}
