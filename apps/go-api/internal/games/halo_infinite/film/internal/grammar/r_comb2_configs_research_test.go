//go:build research && campagne_overlay

package grammar

// r_comb2_configs_research_test.go — R-COMB-2 (cf. r_comb2_research_test.go) : le pilote. Par film
// (un a la fois, sentinelle `filmproc` 4 Gio) : la reference, la combinaison complete, chaque
// levier seul, les variantes de verification, puis la combinaison privee d UN levier a la fois.
// L oracle L9 se derive de la marche qui porte les memes autres leviers (marche « base », avec le
// diagnostic [b3Diag]). Chaque paquet ferme passe au juge des invariants de l ecrivain
// ([cmContredit], decision D2).
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	CAMPAGNE_CATALOGUE=<map_quant_bounds.json> CAMPAGNE_CARTES="0797ce72=Live Fire;..." \
//	CAMPAGNE_BORNES_sgh_interlock="0:...;2:...;3:..." \
//	  go test -tags=research,campagne_overlay -overlay=<surcouche_unique_postj12/overlay.json> \
//	  -count=1 -timeout 240m -run '^TestRComb2$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rc2Res : une configuration mesuree sur un film.
type rc2Res struct {
	nom, leviers, effectifs, rejoue string
	nl9, actifs, desav              int
	a                               rcAgg
	st                              []uint8
	ut                              []int32
	debut                           []int32
	sig                             map[string]int
	score                           map[int][2]int
}

// rc2Cmp : une marche contre une base, paquet par paquet.
type rc2Cmp struct{ g, p, gs, ps, psContre, psUt, gsUt int }

func rc2Comparer(r, base *rc2Res) rc2Cmp {
	var c rc2Cmp
	if len(r.st) != len(base.st) {
		return rc2Cmp{g: -1, p: -1, gs: -1, ps: -1}
	}
	for i, s := range r.st {
		b := base.st[i]
		switch {
		case s > 0 && b == 0:
			c.g++
		case s == 0 && b > 0:
			c.p++
		}
		switch {
		case s == 2 && b != 2:
			c.gs++
			c.gsUt += int(r.ut[i])
		case s != 2 && b == 2:
			c.ps++
			c.psUt += int(base.ut[i])
			if s == 1 {
				c.psContre++
			}
		}
	}
	return c
}

// marcher joue une configuration ; `diag` branche le diagnostic qui derive l oracle L9.
func (x *rc2Film) marcher(k rc2L, oracle map[[2]int][]cmLiaison, diag bool) (*rc2Res, *b3Diag) {
	fv := x.filmDe(k)
	x.poser(k)
	defer rc2Lever()
	o := &rc2Obs{chk: nouveauCollecteur(fv, x.b, nil), b: x.b, lp: k.lp, score: map[int][2]int{},
		sig: map[string]int{}, extra: motFacultatifDEnTete(fv.cfg), garder: k.ls && !k.l1a && !k.l1aRef}
	if k.ls {
		o.tr = &rlocTrace{}
	}
	mux := cmMux{o}
	var d *b3Diag
	if diag {
		d = b3NouveauDiag(fv, x.b, false)
		mux = append(mux, d)
	}
	r, obs, _ := cmMarcher(fv, cmVariante{oracle: oracle, tete: x.tete(k, fv, o)}, mux)
	o.a.paquets, o.a.utilesLus = r.Paquets, r.Utiles.Records
	o.a.horsCadre, o.a.rejets = r.Bloquants[CauseHorsCadre].Paquets, obs.RejetsHorsDatum
	return &rc2Res{nl9: b3Compter(oracle), actifs: o.actifs, desav: o.desav, a: o.a, st: o.st, ut: o.ut,
		debut: o.debut, sig: o.sig, score: o.score}, d
}

// jouer mesure `k` sous le nom `nom` ; `base` : marche qui derive aussi l oracle L9.
func (x *rc2Film) jouer(k rc2L, nom string, base bool) *rc2Res {
	e := x.effectif(k)
	_, aOracle := x.oracles[e]
	if r, ok := x.cache[e]; ok && (!base || aOracle) {
		c := *r
		c.nom, c.leviers = nom, k.nom()
		c.rejoue = "non (identique a " + r.nom + ")"
		if e != k {
			c.rejoue = "non (identique par construction a " + r.nom + ")"
		}
		return &c
	}
	var oracle map[[2]int][]cmLiaison
	if e.l9 {
		b := e
		b.l9 = false
		if _, ok := x.oracles[b]; !ok {
			x.jouer(b, "base:"+b.nom(), true)
		}
		oracle = x.oracles[b]
	}
	r, d := x.marcher(e, oracle, base && !e.l9)
	if d != nil {
		x.oracles[e] = d.imageCle
	}
	r.nom, r.leviers, r.effectifs, r.rejoue = nom, k.nom(), e.nom(), "oui"
	if _, ok := x.cache[e]; !ok {
		x.cache[e] = r
	}
	return r
}

// rc2Configurations joue les configurations d un film, dans l ordre des derivations.
func rc2Configurations(x *rc2Film) []*rc2Res {
	var res []*rc2Res
	add := func(k rc2L, nom string, base bool) *rc2Res {
		r := x.jouer(k, nom, base)
		res = append(res, r)
		return r
	}
	ref := add(rc2L{}, "reference", true)
	x.refActifs = map[int]bool{}
	acc := [2]int{}
	for _, c := range x.f.fc.ChunkNumbers() {
		x.refActifs[c] = rc2Actif(acc)
		s := ref.score[c]
		acc[0], acc[1] = acc[0]+s[0], acc[1]+s[1]
	}
	// CAMPAGNE_RCOMB2_RETIRES : leviers retires de la combinaison complete (passe 2 : « L7 », rejete par
	// la passe 1) ; la passe 2 ne rejoue ni les leviers seuls ni les variantes.
	tous := rc2Tous()
	retires := strings.Split(os.Getenv("CAMPAGNE_RCOMB2_RETIRES"), ",")
	for _, r := range retires {
		tous = tous.sans(strings.TrimSpace(r))
	}
	add(tous.sans("L9"), "full-L9", true)
	add(tous, "full", false)
	if retires[0] == "" {
		for _, l := range rc2Noms[:rc2NLots] {
			add(rc2Avec(l.nom), l.nom, false)
		}
		for _, v := range [][]string{{"L1a-ref"}, {"WO"}, {"L6b-flock"}, {"LS", "L8"}, {"LM", "L6a"}, {"L6a", "WO"}} {
			k := rc2Avec(v...)
			add(k, k.nom(), false)
		}
	}
	for _, l := range rc2Noms[:rc2NLots-1] {
		b := tous.sans(l.nom).sans("L9")
		add(b, "full-"+l.nom+"-L9", true)
		add(tous.sans(l.nom), "full-"+l.nom, false)
	}
	wo := tous
	wo.wo = true
	add(wo.sans("L9"), "full+WO-L9", true)
	add(wo, "full+WO", false)
	return res
}

const rc2Entete = "film\tbuild\tformat\tcontexte\tconfig\tleviers\teffectifs\trejoue\tliaisons_l9\tchunks_l1a_actifs\t" +
	"liaisons_desavouees_lp\tpaquets\tfermes\tfermes_sains\tutiles_fermes\tutiles_fermes_sains\tutiles_lus\thors_cadre\t" +
	"rejets_hors_datum\tg_ref\tp_ref\tgs_ref\tps_ref\tps_ref_vers_contredit\tps_ref_utiles\tgs_ref_utiles\t" +
	"g_full\tp_full\tgs_full\tps_full"

// TestRComb2 joue R-COMB-2 sur les films de CAMPAGNE_FILMS.
func TestRComb2(t *testing.T) {
	racine, sortie, films := b2Env(t)
	cartes := map[string]string{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_CARTES"), ";") {
		if id, carte, ok := strings.Cut(x, "="); ok {
			cartes[strings.TrimSpace(id)] = strings.TrimSpace(carte)
		}
	}
	var cat *profile.MapQuantCatalog
	if len(cartes) > 0 {
		var err error
		if cat, err = profile.LoadMapQuantCatalog(os.Getenv("CAMPAGNE_CATALOGUE")); err != nil {
			t.Fatalf("catalogue : %v", err)
		}
	}
	if os.Getenv("CAMPAGNE_RLOC_LS") != "" || os.Getenv("CAMPAGNE_RCOMB2_KS") != "" || os.Getenv("CAMPAGNE_RCOMB2_L7") != "" {
		t.Fatal("bascules de binaire posees dans l environnement : la marche de reference ne serait pas la production")
	}
	utiles := cmUtiles(t)
	out := map[string][]string{"configs": {rc2Entete}, "signatures": {"film\tbuild\tconfig\tcle\tpaquets"},
		"ls_l8": {"film\tbuild\tpaquets_a_evenements\tdebut_identique\tdebut_different\tlocalise_ls_seul\tlocalise_ls_l8_seul"}}
	defer rc2Lever()
	for _, id := range films {
		rc2UnFilm(t, racine, id, cartes[id], cat, utiles, out)
	}
	for nom, l := range out {
		b2Ecrire(t, sortie, "r_comb2_"+nom+".tsv", l)
	}
}

// rc2UnFilm mesure un film.
func rc2UnFilm(t *testing.T, racine, id, carte string, cat *profile.MapQuantCatalog, utiles UsagesProduit,
	out map[string][]string) {
	garde := filmproc.Arm("campagne/r-comb2", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	x := &rc2Film{contexte: "instruments", cache: map[rc2L]*rc2Res{}, oracles: map[rc2L]map[[2]int][]cmLiaison{}}
	var controle *rc2Res
	if carte != "" {
		entry, err := cat.Lookup(carte)
		if err != nil {
			t.Errorf("%s : carte %q hors catalogue (%v)", id, carte, err)
			return
		}
		if fi, ok := cmOuvrir(t, racine, id, utiles); ok {
			xi := &rc2Film{f: fi, b: cmLireBlocs(fi), hf: rlocArchetypesHF(fi.reg)}
			controle, _ = xi.marcher(rc2L{}, nil, false)
			controle.nom, controle.leviers, controle.effectifs, controle.rejoue = "controle:reference-instruments", "aucun", "aucun", "oui"
		}
		f, ok := b2pOuvrirProduction(t, racine, id, entry, utiles)
		if !ok {
			return
		}
		x.f, x.contexte, x.bornes = f, "production:"+carte, b2pBornes(entry.Module)
	} else {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			return
		}
		x.f = f
	}
	x.b, x.hf = cmLireBlocs(x.f), rlocArchetypesHF(x.f.reg)
	x.format, _ = FilmFormatVersion(x.f.fc.Film())
	res := rc2Configurations(x)
	var ref, full *rc2Res
	for _, r := range res {
		switch r.nom {
		case "reference":
			ref = r
		case "full":
			full = r
		}
	}
	if controle != nil {
		res = append([]*rc2Res{controle}, res...)
	}
	for _, r := range res {
		vr, vf := rc2Comparer(r, ref), rc2Comparer(r, full)
		if r == controle {
			vr, vf = rc2Cmp{}, rc2Cmp{}
		}
		a := r.a
		out["configs"] = append(out["configs"], strings.Join([]string{id, x.f.build, fmt.Sprint(x.format), x.contexte,
			r.nom, r.leviers, r.effectifs, r.rejoue, rc2Champs(r.nl9, r.actifs, r.desav, a.paquets, a.fermes, a.sains,
				a.utilesFermes, a.utilesSains, a.utilesLus, a.horsCadre, a.rejets, vr.g, vr.p, vr.gs, vr.ps, vr.psContre,
				vr.psUt, vr.gsUt, vf.g, vf.p, vf.gs, vf.ps)}, "\t"))
		if r.rejoue == "oui" || r == ref {
			for cle, n := range r.sig {
				out["signatures"] = append(out["signatures"], fmt.Sprintf("%s\t%s\t%s\t%s\t%d", id, x.f.build, r.nom, cle, n))
			}
		}
	}
	out["ls_l8"] = append(out["ls_l8"], rc2LsL8(id, x.f.build, res))
	t.Logf("%s %s (%s, format %d) : %d configurations ; pic %d Mio, %s", id, x.f.build, x.contexte, x.format,
		len(res), garde.Peak()>>20, time.Since(debut).Round(time.Second))
}

// rc2Champs met des entiers en champs TSV.
func rc2Champs(v ...int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, "\t")
}

// rc2LsL8 : point 3 de la mission — le debut de liste de chaque paquet a evenements sous LS et
// sous LS+L8 (la signature high-frequency lue par NOM reste-t-elle la meme quand `ti=3 i1` est
// route vers sa table ?).
func rc2LsL8(id, build string, res []*rc2Res) string {
	var ls, ls8 *rc2Res
	for _, r := range res {
		switch r.nom {
		case "LS":
			ls = r
		case "L8+LS":
			ls8 = r
		}
	}
	if ls == nil || ls8 == nil || len(ls.debut) != len(ls8.debut) {
		return fmt.Sprintf("%s\t%s\t-1\t-1\t-1\t-1\t-1", id, build)
	}
	var ev, egal, diff, seulLS, seulLS8 int
	for i, d := range ls.debut {
		d8 := ls8.debut[i]
		if d == -2 && d8 == -2 {
			continue
		}
		ev++
		switch {
		case d == d8:
			egal++
		default:
			diff++
			if d >= 0 && d8 < 0 {
				seulLS++
			}
			if d < 0 && d8 >= 0 {
				seulLS8++
			}
		}
	}
	return fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d", id, build, ev, egal, diff, seulLS, seulLS8)
}
