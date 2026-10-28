//go:build research

package grammar

// durees_effets_lien_research_test.go — ETAPE B.1/B.2 du plan des durees d effets : QUEL LIEN
// STRUCTUREL relie (1) la pose de mur publiee (creation ti=37 designee par un 103) a l objet suivi du
// mur (pile i28), et (2) la vie ti=38 nee a l impact de l ecran a son objet suivi ? Lecture seule.
// Reutilise les aides de durees_effets_research_test.go.
//
//	CGO_ENABLED=0 EQUIPPORTE_FILMS=<repo>/data/cache/film_chunks EQUIPPORTE_ID=1ea71def \
//	  EQUIPPORTE_CARTE=Deadlock EQUIPPORTE_OUT=<scratchpad>/lien go test -tags=research \
//	  ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestLienDuMur$' -count=1 -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestLienDuMur(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	art, err := deLireArtefact(filepath.Join(racine, "..", "replays", "halo_infinite", id+".json"))
	if err != nil {
		t.Fatalf("%s : artefact : %v", id, err)
	}
	famille := func(r int) string {
		if l, ok := art.AbilityLabels[strconv.Itoa(r)]; ok && l.Family != "" {
			return l.Family
		}
		return "?"
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatal(err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	wr := bornesDeLaCarte(t)
	lectures, _, err := ScanEquipementPorte(fc, &wr)
	if err != nil {
		t.Fatal(err)
	}
	fc2, _ := epContexte(filepath.Join(racine, id))
	ch, _, err := ScanEquipmentChanges(fc2, nil)
	if err != nil {
		t.Fatal(err)
	}
	rangs := map[[2]uint64]int{}
	for _, e := range ch {
		if e.Kind == types.EquipmentTaken || e.Kind == types.EquipmentSpawned {
			rangs[[2]uint64{e.TimestampUS, uint64(e.Slot)}] = e.Rank
		}
	}
	usages, suivis := deLireLesUsages(lectures, rangs, famille)
	deRattacher(usages, suivis)

	fc3, _ := epContexte(filepath.Join(racine, id))
	poses, _, err := ScanEquipmentPlacements(fc3, &wr)
	if err != nil {
		t.Fatal(err)
	}
	fc4, _ := epContexte(filepath.Join(racine, id))
	evs, _, err := ScanEquipmentSpawnEvents(fc4)
	if err != nil {
		t.Fatal(err)
	}
	ms := func(ts uint64) float64 { return float64(int64(ts)-int64(origine)) / 1000 }

	var b strings.Builder
	fmt.Fprintf(&b, "# %s : %d usages, %d suivis, %d poses ti=37, %d evenements 103\n", id, len(usages), len(suivis), len(poses), len(evs))
	// Poses designees par un 103 (les panneaux) : cle de vie, instant, evenement, Source.
	type panneau struct {
		p  types.EquipmentPlacement
		ev types.EquipmentSpawnEvent
	}
	var panneaux []panneau
	for _, p := range poses {
		for _, e := range evs {
			if e.SpawnedValid && e.Spawned == p.Life && e.TimestampUS >= p.T0US && e.TimestampUS-p.T0US <= 200_000 {
				panneaux = append(panneaux, panneau{p, e})
				break
			}
		}
	}
	sort.Slice(panneaux, func(i, j int) bool { return panneaux[i].p.T0US < panneaux[j].p.T0US })
	for _, pn := range panneaux {
		fmt.Fprintf(&b, "PANNEAU\tt=%.3f\tlife=%d/%d\tgid=%08x\tpos=%.2f,%.2f,%.2f\tsource=%d/%d valid=%v\tpoints=%d\n",
			ms(pn.p.T0US), pn.p.Life.Slot, pn.p.Life.Gen, pn.p.GlobalID, pn.p.X, pn.p.Y, pn.p.Z,
			pn.ev.Source.Slot, pn.ev.Source.Gen, pn.ev.SourceValid, pn.p.Points)
	}
	for _, u := range usages {
		if u.famille != "wall" {
			continue
		}
		for _, s := range u.suivis {
			fmt.Fprintf(&b, "SUIVI_MUR\tusage=%d\tt=%.3f\tobjet=%d/%d\tporteur=%d\tsuivi=%d/%d ti=%d\tnomme=%.3f\tfin=%.3f\tcreation=%s\n",
				u.idx, ms(u.ts), u.objet.Slot, u.objet.Gen, u.porteur, s.cle.Slot, s.cle.Gen, s.ti, ms(s.nomme),
				func() float64 {
					if s.fin == 0 {
						return -1
					}
					return ms(s.fin)
				}(), strings.ReplaceAll(dePos(s.creation, s.aCreation), "\t", ","))
			// Liens candidats : meme vie qu un panneau ? meme Source que l objet porte ?
			for _, pn := range panneaux {
				memeVie := pn.p.Life == s.cle
				memeSource := pn.ev.SourceValid && pn.ev.Source == u.objet
				proche := deAbs64(int64(pn.p.T0US)-int64(s.nomme)) <= 3_000_000
				if memeVie || memeSource || proche {
					fmt.Fprintf(&b, "  LIEN?\tpanneau t=%.3f life=%d/%d memeVie=%v memeSource=%v dt=%.0fms\n", ms(pn.p.T0US),
						pn.p.Life.Slot, pn.p.Life.Gen, memeVie, memeSource, float64(int64(pn.p.T0US)-int64(s.nomme))/1000)
				}
			}
		}
	}
	if err := os.WriteFile(filepath.Join(out, id+".lien.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + b.String())
}

// ecranCanal note, pour chaque record NEW d un archetype (EQUIPPORTE_TI, defaut 38) : instant, cle,
// composants presents (index/bits) et la lecture i10 (porte « attache », parent, identifiant libre).
type ecranCanal struct {
	ti      int16
	lignes  []string
	origine uint64
}

func (c *ecranCanal) Interets() []Interet                      { return nil }
func (c *ecranCanal) Clore(BilanDeMarche)                      {}
func (c *ecranCanal) Brancher(*Observation, *MarcheDistribuee) {}

func (c *ecranCanal) Trame(p *lecture.Paquet) {
	for _, r := range p.Records {
		if r.Genre != lecture.GenreNeuf || r.TI != c.ti {
			continue
		}
		var cs, i10 strings.Builder
		for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
			fmt.Fprintf(&cs, " i%d/%d", k.Index, k.Bits)
			if k.Index == 10 && k.Bits > 0 {
				for _, param := range []uint32{3, 1} {
					br := LecteurSur(p.Payload)
					br.SetBitPos(int(k.Debut))
					st := decodeObjectParentState(br, param, uint32(c.ti))
					fmt.Fprintf(&i10, " [param%d attache=%v quant16=%d free=%v id=%d/%v alt11=%d bits=%d]", param,
						st.Attached, st.Quant16, st.FreeRead, st.FreeID, st.HasFreeID, st.Alt11, br.BitPos()-int(k.Debut))
				}
			}
		}
		c.lignes = append(c.lignes, fmt.Sprintf("NEW ti=%d t=%.3f vie=%d/%d comps:%s i10:%s", r.TI,
			float64(int64(p.TS)-int64(c.origine))/1000, r.Vie.Slot, r.Vie.Gen, cs.String(), i10.String()))
	}
}

// TestLienDeLEcran : les creations ti=38 du film et la lecture de leur i10 (le parent de l objet).
func TestLienDeLEcran(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatal(err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	ti := 38
	if v := os.Getenv("EQUIPPORTE_TI"); v != "" {
		ti = deEntier(v, 38)
	}
	c := &ecranCanal{ti: int16(ti), origine: origine}
	if err := Distribuer(fc, c, nouveauCanalDeLEquipementPorte(fc)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, id+".ecran.txt"), []byte(strings.Join(c.lignes, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s : %d creations ti=%d", id, len(c.lignes), ti)
}

// TestAssocEcran : pour chaque objet suivi d ecran occultant, les vies ti=38 nees dans [fin - 1,5 s,
// fin + 0,3 s] — instant relatif a la fin de la grenade, a l instant ou la piste de la grenade se pose
// (AtRest) et a son dernier point, distance au repos, duree, et le nombre de rivales (ti=38 non nommees).
func TestAssocEcran(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	art, err := deLireArtefact(filepath.Join(racine, "..", "replays", "halo_infinite", id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	famille := func(r int) string {
		if l, ok := art.AbilityLabels[strconv.Itoa(r)]; ok && l.Family != "" {
			return l.Family
		}
		return "?"
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatal(err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	wr := bornesDeLaCarte(t)
	cf := &cfCanal{actives: map[uint32]*cfVie{}, fc: fc, posTI: 38, wr: wr}
	prod := nouveauCanalDeLEquipementPorte(fc)
	if err := Distribuer(fc, cf, prod); err != nil {
		t.Fatal(err)
	}
	lectures, _, err := ScanEquipementPorte(fc, &wr)
	if err != nil {
		t.Fatal(err)
	}
	fc2, _ := epContexte(filepath.Join(racine, id))
	ch, _, err := ScanEquipmentChanges(fc2, nil)
	if err != nil {
		t.Fatal(err)
	}
	rangs := map[[2]uint64]int{}
	for _, e := range ch {
		if e.Kind == types.EquipmentTaken || e.Kind == types.EquipmentSpawned {
			rangs[[2]uint64{e.TimestampUS, uint64(e.Slot)}] = e.Rank
		}
	}
	usages, suivis := deLireLesUsages(lectures, rangs, famille)
	deRattacher(usages, suivis)
	fc3, _ := epContexte(filepath.Join(racine, id))
	pistes, err := ScanProjectiles(fc3, &wr)
	if err != nil {
		t.Fatal(err)
	}
	deSituerLesRepos(suivis, pistes)
	nommes := map[types.LifeKey]bool{}
	for _, s := range suivis {
		nommes[s.cle] = true
	}
	ms := func(ts uint64) float64 { return float64(int64(ts)-int64(origine)) / 1000 }
	var b strings.Builder
	for _, u := range usages {
		if u.famille != "shroud_screen" || len(u.suivis) == 0 {
			continue
		}
		s := u.suivis[0]
		var piste *types.ProjectileTrack
		for i, p := range pistes {
			if p.Slot == s.cle.Slot && p.Gen == s.cle.Gen && len(p.Pts) > 0 &&
				deAbs64(int64(p.Pts[0].TimestampUS)-int64(s.nomme)) <= deToleranceNaissanceUS {
				piste = &pistes[i]
			}
		}
		atRest, dernier := uint64(0), uint64(0)
		if piste != nil {
			dernier = piste.Pts[len(piste.Pts)-1].TimestampUS
			for _, p := range piste.Pts {
				if p.AtRest && atRest == 0 {
					atRest = p.TimestampUS
				}
			}
		}
		fmt.Fprintf(&b, "USAGE %d t=%.3f grenade=%d/%d nomme=%.3f fin=%.3f dernierPt=%.3f atRest=%.3f repos=%s\n", u.idx, ms(u.ts),
			s.cle.Slot, s.cle.Gen, ms(s.nomme), ms(s.fin), ms(dernier), ms(atRest), dePos(s.repos, s.reposEtat != "absent"))
		for _, v := range cf.vies {
			if v.ti != 38 || nommes[types.LifeKey{Slot: v.slot, Gen: v.gen}] {
				continue
			}
			if int64(v.ne) < int64(s.fin)-1_500_000 || int64(v.ne) > int64(s.fin)+300_000 {
				continue
			}
			d, dist := -1.0, -1.0
			if v.fin > 0 {
				d = float64(v.fin-v.ne) / 1000
			}
			if v.aPos && s.reposEtat != "absent" {
				dist, _ = deDist(v.pos, s.repos)
			}
			fmt.Fprintf(&b, "   ti38 vie=%d/%d ne=%.3f vsFin=%.0fms vsAtRest=%.0fms vsDernierPt=%.0fms duree=%.0fms dist=%.2f\n", v.slot, v.gen,
				ms(v.ne), float64(int64(v.ne)-int64(s.fin))/1000, float64(int64(v.ne)-int64(atRest))/1000,
				float64(int64(v.ne)-int64(dernier))/1000, d, dist)
		}
	}
	if err := os.WriteFile(filepath.Join(out, id+".assoc.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDiagEffets : les lectures d effet que le canal de production rend (naissance : cle, instant, position ; fin).
func TestDiagEffets(t *testing.T) {
	racine, id := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID")
	if racine == "" || id == "" {
		t.Skip("EQUIPPORTE_FILMS ou EQUIPPORTE_ID absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatal(err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	wr := bornesDeLaCarte(t)
	ls, _, err := ScanEquipementPorte(fc, &wr)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range ls {
		switch l.Kind {
		case types.CarriedTrackedEnd:
			t.Logf("FIN suivi %d/%d a %.3f", l.Tracked[0].Slot, l.Tracked[0].Gen, float64(int64(l.TimestampUS)-int64(origine))/1e6)
		case CarriedEffectBirth:
			n := l.Spawned[0]
			t.Logf("  NAISSANCE pour %d/%d : %d/%d ne %.3f pos %v (%.2f,%.2f,%.2f)", l.Tracked[0].Slot, l.Tracked[0].Gen, n.Life.Slot, n.Life.Gen,
				float64(int64(n.TimestampUS)-int64(origine))/1e6, n.HasPosition, n.X, n.Y, n.Z)
		case CarriedEffectEnd:
			t.Logf("  FIN D EFFET %d/%d a %.3f", l.Tracked[0].Slot, l.Tracked[0].Gen, float64(int64(l.TimestampUS)-int64(origine))/1e6)
		}
	}
}
