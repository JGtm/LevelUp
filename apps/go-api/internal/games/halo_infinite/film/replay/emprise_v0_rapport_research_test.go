package replay

// emprise_v0_rapport_research_test.go — LOT V0 DU PLAN `.ai/PLAN_EMPRISE_VIES_2026-09-28.md` :
// LA COMPARAISON DES PORTAGES ET LE RAPPORT AGREGE.
//
// # LA MESURE DE FIDELITE, ECRITE AVANT LES CHIFFRES
//
// Chaque portage est un intervalle [debut, fin] en millisecondes du MATCH, par joueur. Pour un
// joueur, R est l'union des portages de la reference (document de la cuisson), C celle du
// candidat (voie a ou b). Avec D(X) = X dilate de 100 ms de chaque cote :
//
//	identique a +-100 ms  = |R ∩ D(C)|           (temps de reference que le candidat couvre)
//	en trop               = |C| − |C ∩ D(R)|     (temps que le candidat invente)
//	taux                  = identique / (|R| + en trop)
//
// Le seuil du plan (98 %) porte sur le taux AGREGE (sommes sur les films et les familles).
//
// Les frames d'un document se convertissent par `ms = (origine + f × pas) / 1000 − calage`, avec
// l'origine (premier paquet de position) et le calage (`coverage.bridge.deathOffsetMs`) de CE
// document — la convention de `match_clock.go`, lue a l'envers.
//
// `TestEmpriseV0Rapport` relit les resultats poses (volets collecteur et porteurs) et imprime les
// tableaux colles au journal du lot. Il se saute sans `EMPRISE_V0_DIR`.

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// v0Tolerance : la tolerance du seuil, en ms.
const v0Tolerance = 100

// v0Fid : les trois temps d'une comparaison, en ms.
type v0Fid struct{ RefMS, OkMS, ExtraMS int64 }

// Taux rend identique / (reference + en trop) ; 1 quand il n'y a rien a comparer.
func (f v0Fid) Taux() float64 {
	if den := f.RefMS + f.ExtraMS; den > 0 {
		return float64(f.OkMS) / float64(den)
	}
	return 1
}

func (f *v0Fid) ajouter(g v0Fid) { f.RefMS += g.RefMS; f.OkMS += g.OkMS; f.ExtraMS += g.ExtraMS }

// v0Resultat : la mesure V0.2 d'un film.
type v0Resultat struct {
	MatchID, Variante, Famille, RegistreEcart string
	PasseMS, CoutA, CoutStatborg, CoutPont    float64
	CoutDrapeau, CoutAssemblage, CoutB        float64
	CoutEquipes, CoutMonde, CoutBSansMonde    float64
	SurcoutA, SurcoutB                        float64
	PicPasse, PicBase, PicA, PicB             uint64
	CalageRef, CalageB, CalageCollecteur      int64
	CalageRefInconnu                          bool
	TenusParFamille                           map[string]int
	Fid                                       map[string]map[string]v0Fid
}

// v0MesurerFilm : la base, les lectures des deux voies, puis les comparaisons.
func v0MesurerFilm(t *testing.T, e v0Entree) v0Resultat {
	t.Helper()
	fam := v0FamilleDuMode(e.ref.Variante)
	r := v0Resultat{MatchID: e.id, Variante: e.ref.Variante, Famille: fam, PasseMS: e.col.PasseMedMS,
		Fid: map[string]map[string]v0Fid{}}
	for _, p := range e.col.PicPasse {
		r.PicPasse = max(r.PicPasse, p)
	}
	b, picBase := v0ChargerBase(t, e)
	r.PicBase, r.RegistreEcart = picBase, v0RegistreConforme(b)
	r.CalageCollecteur = b.reg.DeathOffsetMS()
	if fam == "" {
		// HORS MODE A PORTEUR, LA GARDE DE MODE NE LIT RIEN : ni la voie (a) ni la voie (b).
		return r
	}
	l := v0LireEnPlus(t, e, b, fam)
	var docB ReplayDocument
	var picAsm uint64
	r.CoutAssemblage, picAsm = v0Chrono(e.tours, func() {
		docB = BuildFromPositions(e.id, title.DefaultSlug, b.positions, nil, v0OptionsSync(e, b, l, fam))
	})
	r.CoutA, r.CoutStatborg, r.CoutPont, r.CoutDrapeau = l.coutA, l.coutStatborg, l.coutPont, l.coutDrap
	r.CoutEquipes, r.CoutMonde = l.coutEquipes, l.coutMonde
	r.CoutB, r.PicB = v0CoutB(r, l, picAsm, fam)
	r.PicA, r.PicB = max(r.PicBase, l.picA), max(r.PicBase, r.PicB)
	r.SurcoutA, r.SurcoutB = 100*r.CoutA/r.PasseMS, 100*r.CoutB/r.PasseMS
	ref := v0IntervallesRef(e, &r)
	r.CalageB = r.CalageCollecteur
	if docB.Coverage != nil && docB.Coverage.Bridge.DeathOffsetMs != nil {
		r.CalageB = *docB.Coverage.Bridge.DeathOffsetMs
	}
	cb := v0IntervallesDoc(v0DocPortages(docB), v0PremierPaquet(b.positions),
		uint64(docB.FrameIntervalMS)*1000, r.CalageB)
	ca := v0IntervallesTenus(t, b, l.held, &r)
	r.Fid["a"], r.Fid["b"] = v0ComparerFamilles(ref, ca), v0ComparerFamilles(ref, cb)
	if fam == "drapeau" {
		v0SansObjetsDuMonde(e, b, l, ref, &r)
	}
	return r
}

// v0CoutB compose le cout de la voie (b) : ce que CHAQUE calque exige, et rien de plus.
func v0CoutB(r v0Resultat, l v0Lectures, picAsm uint64, fam string) (float64, uint64) {
	switch fam {
	case "drapeau":
		return r.CoutStatborg + r.CoutPont + r.CoutDrapeau + r.CoutAssemblage,
			max(l.picStat, l.picDr, picAsm)
	case "crane":
		return r.CoutStatborg + r.CoutPont + r.CoutAssemblage, max(l.picStat, picAsm)
	case "vip":
		// Le calque VIP resout son pont lui-meme (`attachVipCrown`) : il est dans l'assemblage.
		return r.CoutStatborg + r.CoutAssemblage, max(l.picStat, picAsm)
	default: // bombe : `bombInput` ne lit rien de plus que le canal des armes tenues
		return r.CoutA + r.CoutAssemblage, max(l.picA, picAsm)
	}
}

// v0Iv : un intervalle [a, b] en ms du match.
type v0Iv struct{ a, b int64 }

// v0Portages : les portages d'un document, par famille, en frames.
type v0Portages map[string][]v0Periode

// v0DocPortages extrait les quatre calques d'un document.
func v0DocPortages(doc ReplayDocument) v0Portages {
	p := v0Portages{}
	for _, fc := range doc.FlagCarries {
		for _, s := range fc.Spans {
			if s.State == FlagStateCarried && s.XUID != nil {
				p["drapeau"] = append(p["drapeau"], v0Periode{XUID: *s.XUID, T0: s.T0, T1: s.T1})
			}
		}
	}
	for _, c := range doc.SkullCarries {
		p["crane"] = append(p["crane"], v0Periode{XUID: c.XUID, T0: c.T0, T1: c.T1})
	}
	for _, c := range doc.BombCarries {
		p["bombe"] = append(p["bombe"], v0Periode{XUID: c.XUID, T0: c.T0, T1: c.T1})
	}
	for _, c := range doc.VipCrown {
		p["vip"] = append(p["vip"], v0Periode{XUID: c.XUID, T0: c.T0, T1: c.T1})
	}
	return p
}

// v0IntervallesDoc convertit des portages en frames vers les ms du match.
func v0IntervallesDoc(p v0Portages, origine, pas uint64, calage int64) map[string]map[string][]v0Iv {
	ms := func(f int) int64 { return int64((origine+uint64(f)*pas)/1000) - calage }
	out := map[string]map[string][]v0Iv{}
	for fam, list := range p {
		out[fam] = map[string][]v0Iv{}
		for _, q := range list {
			out[fam][q.XUID] = append(out[fam][q.XUID], v0Iv{ms(q.T0), ms(q.T1)})
		}
	}
	return out
}

// v0IntervallesRef rend la reference en ms du match (calage de la cuisson, ou a defaut celui du
// collecteur — signale).
func v0IntervallesRef(e v0Entree, r *v0Resultat) map[string]map[string][]v0Iv {
	calage := e.col.Calage
	if e.ref.CalageRefMs != nil {
		calage = *e.ref.CalageRefMs
	} else {
		r.CalageRefInconnu = true
	}
	r.CalageRef = calage
	p := v0Portages{"drapeau": e.ref.Drapeau, "crane": e.ref.Crane, "bombe": e.ref.Bombe, "vip": e.ref.VIP}
	return v0IntervallesDoc(p, e.ref.OrigineUS, uint64(e.ref.FrameIntervalMS)*1000, calage)
}

// v0PremierPaquet : l'origine d'un document construit sur ces positions (`assemblage.ouvrir`).
func v0PremierPaquet(pos []grammar.BipedPosition) uint64 {
	var min uint64
	for i := range pos {
		if ts := pos[i].TimestampUS; min == 0 || ts < min {
			min = ts
		}
	}
	return min
}

// v0EvenementsTenus : `bombHeldEventsOf`, pour une famille quelconque (meme protocole B2 :
// prise = transition VERS la famille, lacher = transition DEPUIS).
func v0EvenementsTenus(changes []types.HeldWeaponChange, fam uint32, calage int64) []HeldObjectEvent {
	var out []HeldObjectEvent
	for _, ch := range changes {
		matchMS := int(int64(ch.TimestampUS)/1000 - calage)
		if ch.Family == fam {
			out = append(out, HeldObjectEvent{TimeMS: matchMS, Slot: ch.Slot, Pickup: true})
		}
		if ch.Previous == fam && ch.Family != fam {
			out = append(out, HeldObjectEvent{TimeMS: matchMS, Slot: ch.Slot, Pickup: false})
		}
	}
	return out
}

// v0IntervallesTenus : la voie (a) — `BuildHeldObjectCarry` par famille, ponte par le registre du
// collecteur a l'instant. Une periode restee ouverte court jusqu'a la derniere position du film.
func v0IntervallesTenus(t *testing.T, b v0Base, held []types.HeldWeaponChange,
	r *v0Resultat) map[string]map[string][]v0Iv {
	t.Helper()
	calage := b.reg.DeathOffsetMS()
	if got, want := v0EvenementsTenus(held, v0FamBombe, calage), bombHeldEventsOf(held, calage); len(got) != len(want) {
		t.Fatalf("filtre generique %d evenements contre %d pour la bombe", len(got), len(want))
	}
	var dernier uint64
	for i := range b.positions {
		dernier = max(dernier, b.positions[i].TimestampUS)
	}
	fin := int64(dernier/1000) - calage
	r.TenusParFamille = map[string]int{}
	out := map[string]map[string][]v0Iv{}
	for nom, fam := range map[string]uint32{"drapeau": v0FamDrapeau, "crane": v0FamCrane, "bombe": v0FamBombe} {
		ev := v0EvenementsTenus(held, fam, calage)
		r.TenusParFamille[nom] = len(ev)
		carry := BuildHeldObjectCarry(ev, occupantParMatchMS(b.reg), b.deaths)
		out[nom] = map[string][]v0Iv{}
		for _, p := range carry.Periods {
			if p.XUID == 0 {
				continue
			}
			f := int64(p.FinMS)
			if p.Ouverte {
				f = fin
			}
			x := strconv.FormatUint(p.XUID, 10)
			out[nom][x] = append(out[nom][x], v0Iv{int64(p.DebutMS), f})
		}
	}
	return out
}

// v0ComparerFamilles compare famille par famille (celles ou la reference OU le candidat porte).
func v0ComparerFamilles(ref, cand map[string]map[string][]v0Iv) map[string]v0Fid {
	out := map[string]v0Fid{}
	for _, fam := range []string{"drapeau", "crane", "bombe", "vip"} {
		if len(ref[fam]) == 0 && len(cand[fam]) == 0 {
			continue
		}
		var f v0Fid
		joueurs := map[string]bool{}
		for x := range ref[fam] {
			joueurs[x] = true
		}
		for x := range cand[fam] {
			joueurs[x] = true
		}
		for x := range joueurs {
			rr, cc := v0Union(ref[fam][x]), v0Union(cand[fam][x])
			f.RefMS += v0Longueur(rr)
			f.OkMS += v0Inter(rr, v0Union(v0Dilater(cc)))
			f.ExtraMS += v0Longueur(cc) - v0Inter(cc, v0Union(v0Dilater(rr)))
		}
		out[fam] = f
	}
	return out
}

// v0Union trie et fusionne.
func v0Union(in []v0Iv) []v0Iv {
	s := append([]v0Iv(nil), in...)
	sort.Slice(s, func(i, j int) bool { return s[i].a < s[j].a })
	var out []v0Iv
	for _, iv := range s {
		if iv.b < iv.a {
			iv.b = iv.a
		}
		if n := len(out); n > 0 && iv.a <= out[n-1].b {
			out[n-1].b = max(out[n-1].b, iv.b)
			continue
		}
		out = append(out, iv)
	}
	return out
}

func v0Dilater(in []v0Iv) []v0Iv {
	out := make([]v0Iv, len(in))
	for i, iv := range in {
		out[i] = v0Iv{iv.a - v0Tolerance, iv.b + v0Tolerance}
	}
	return out
}

func v0Longueur(in []v0Iv) int64 {
	var n int64
	for _, iv := range in {
		n += iv.b - iv.a
	}
	return n
}

// v0Inter : longueur de l'intersection de deux unions triees.
func v0Inter(a, b []v0Iv) int64 {
	var n int64
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		lo, hi := max(a[i].a, b[j].a), min(a[i].b, b[j].b)
		if hi > lo {
			n += hi - lo
		}
		if a[i].b < b[j].b {
			i++
		} else {
			j++
		}
	}
	return n
}

// v0Echantillon : le pic d'empreinte d'une phase (`filmproc.Footprint`, 10 ms) ; au-dela de
// 10 Gio le processus s'arrete pour proteger la machine de travail.
type v0Echantillon struct {
	pic        atomic.Uint64
	stop, fini chan struct{}
}

func v0Echantillonner() *v0Echantillon {
	runtime.GC()
	debug.FreeOSMemory()
	s := &v0Echantillon{stop: make(chan struct{}), fini: make(chan struct{})}
	s.noter(filmproc.Footprint())
	go func() {
		defer close(s.fini)
		tk := time.NewTicker(10 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-tk.C:
				v := filmproc.Footprint()
				s.noter(v)
				if v > 10<<30 {
					fmt.Fprintf(os.Stderr, "EMPRISE V0 : empreinte %.2f Gio > plafond — arret\n", v0G(v))
					os.Exit(3)
				}
			}
		}
	}()
	return s
}

func (s *v0Echantillon) noter(v uint64) {
	for {
		old := s.pic.Load()
		if v <= old || s.pic.CompareAndSwap(old, v) {
			return
		}
	}
}

func (s *v0Echantillon) arreter() uint64 {
	s.noter(filmproc.Footprint())
	close(s.stop)
	<-s.fini
	return s.pic.Load()
}

// v0SansObjetsDuMonde : SENSIBILITE, PAS UNE VOIE. La voie (b) du drapeau paie surtout les objets
// du monde (poses et socles, d'ou sortent les vies libres qui ferment un lacher volontaire). Ce
// que coute leur absence en fidelite est mesure ici, avec le meme assembleur et les memes autres
// entrees ; le verdict du lot reste celui de la voie (b) telle que definie.
func v0SansObjetsDuMonde(e v0Entree, b v0Base, l v0Lectures, ref map[string]map[string][]v0Iv,
	r *v0Resultat) {
	l.pads = PadScans{}
	doc := BuildFromPositions(e.id, title.DefaultSlug, b.positions, nil, v0OptionsSync(e, b, l, "drapeau"))
	calage := r.CalageCollecteur
	if doc.Coverage != nil && doc.Coverage.Bridge.DeathOffsetMs != nil {
		calage = *doc.Coverage.Bridge.DeathOffsetMs
	}
	c := v0IntervallesDoc(v0DocPortages(doc), v0PremierPaquet(b.positions),
		uint64(doc.FrameIntervalMS)*1000, calage)
	r.Fid["b-sans-monde"] = v0ComparerFamilles(ref, c)
	r.CoutBSansMonde = r.CoutStatborg + r.CoutPont + r.CoutEquipes + r.CoutAssemblage
}
