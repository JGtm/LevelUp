//go:build research && campagne_overlay

package grammar

// r_comb_research_test.go — RECHERCHE R-COMB DE LA CAMPAGNE DE GRAMMAIRE (2026-10-02, PLAN §6.0
// « Mesures ouvertes » et §6.1, critique n° 2 point N10) : les leviers de la phase 2 mesures
// ENSEMBLE, puis la combinaison privee d UN levier a la fois. Un instrument de recherche : aucun
// fichier de production n est touche, aucune sortie ne change, `grammar.Rev` ne bouge pas.
//
// CE FICHIER EXIGE LA SURCOUCHE DE RECHERCHE (tag `campagne_overlay`) : copie des mesures bis 2
// dans `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/r_comb_overlay/` (fichiers identiques a
// l octet a `mesures_bis2_overlay/`, `overlay.json` reecrit sur ce worktree).
//
// Les leviers (chacun est la MEME copie de recherche que sa mesure separee) :
//
//	L1   oracle des naissances, regions (i)+(ii) de la sonde M1 (MESURES_BIS_1 §4)
//	L9   oracle P1-pont : eid de l image-cle incomplete lies au debut du chunk (BIS_3 §2)
//	L8   `ti=3 i0` low-frequency porte + `ti=3 i1` a 26 bits ([b3CrochetTi3], BIS_3 §6)
//	L2   grammaire `ti=43` de T7 ([b2vLireTi43], BIS_2 §5)
//	L6a  largeurs de la ligne de l index de plage lu (Live Fire, contexte de production, BIS_2 §3.3)
//	L6b  sites `flock-position` et `tacmap-displayasset` lus comme le jeu (BIS_2 §3.1, BIS_4 §3)
//
// Les oracles L1 et L9 se DERIVENT de la marche qui porte les memes composants : marche A
// (composants seuls) -> oracle L9 et oracle L1 « sans L9 » ; marche B (composants + L9) -> oracle
// L1 « dans un monde ou L9 est pose ». Chaque marche de derivation est aussi une configuration
// mesuree. Chaque paquet ferme passe au juge des invariants de l ecrivain ([cmContredit]).
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	CAMPAGNE_CATALOGUE=<map_quant_bounds.json> CAMPAGNE_CARTES="0797ce72=Live Fire;..." \
//	CAMPAGNE_BORNES_sgh_interlock="0:...;2:...;3:..." \
//	  go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 240m \
//	  -run '^TestRComb$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rcComps : les leviers de composant d une marche.
type rcComps struct{ l8, l2, l6a, l6b bool }

// rcAgg : les comptes d une marche.
type rcAgg struct {
	paquets, fermes, sains, utilesFermes, utilesSains, utilesLus, horsCadre int
	entreesUtiles, entreesUtilesSaines, rejets                              int
}

// rcObs ecoute une marche : statut de chaque paquet (0 non ferme, 1 ferme contredit, 2 ferme sain).
type rcObs struct {
	chk    *cmCollecteur
	statut map[[2]int]uint8
	a      rcAgg
}

func (o *rcObs) debutDeChunk(c int, _ []byte, _ []FilmPacket, _ *World) {
	o.chk.chunk = c
	if bis2C3 != nil {
		bis2C3.evenements = bis2C3.evenements[:0]
	}
}

func (o *rcObs) finDeFilm() {}

func (o *rcObs) paquet(_ int, p *cmPaquet, _ *World) {
	cle := [2]int{p.d.Chunk, p.d.Index}
	if bis2C3 != nil {
		bis2C3.evenements = bis2C3.evenements[:0]
	}
	if !p.d.Fermee {
		o.statut[cle] = 0
		return
	}
	eu := 0
	for _, e := range p.d.VueC.Entrees {
		if e.Bloc {
			eu++
		}
	}
	o.a.fermes++
	o.a.utilesFermes += p.utilesFermes
	o.a.entreesUtiles += eu
	if len(cmContredit(o.chk, p)) > 0 {
		o.statut[cle] = 1
		return
	}
	o.statut[cle] = 2
	o.a.sains++
	o.a.utilesSains += p.utilesFermes
	o.a.entreesUtilesSaines += eu
}

// rcCmp : une marche comparee a une base, paquet par paquet.
type rcCmp struct{ g, p, gs, ps int }

func rcComparer(st, base map[[2]int]uint8) rcCmp {
	var c rcCmp
	for k, s := range st {
		b := base[k]
		switch {
		case s > 0 && b == 0:
			c.g++
		case s == 0 && b > 0:
			c.p++
		}
		switch {
		case s == 2 && b != 2:
			c.gs++
		case s != 2 && b == 2:
			c.ps++
		}
	}
	return c
}

// rcRes : une configuration mesuree sur un film.
type rcRes struct {
	nom, leviers string
	rejoue       bool
	nl1, nl9     int
	a            rcAgg
	vsRef        rcCmp
	vsFull       rcCmp
	statut       map[[2]int]uint8
}

// rcFilm : un film ouvert et son contexte de leviers.
type rcFilm struct {
	f        *cmFilm
	b        *cmBlocs
	contexte string
	bornes   map[int][3][2]float32
}

// rcCrochet : l intercepteur de composants des leviers L8 et L2.
func rcCrochet(l8, l2 bool) func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
	ti3 := b3CrochetTi3(true, true)
	return func(br *Lecteur, name string, typeIndex, level uint32) (bool, bool) {
		switch {
		case l8 && typeIndex == 3:
			return ti3(br, name, typeIndex, level)
		case l2 && typeIndex == 43:
			return b2vLireTi43(br, name)
		}
		return false, false
	}
}

// poser installe les leviers de composant dans la surcouche ; `lever` les retire.
func (x *rcFilm) poser(k rcComps) {
	rcLever()
	b3Ti4Temoin = false
	if k.l8 || k.l2 {
		bis2Intercepteur = rcCrochet(k.l8, k.l2)
	}
	bis2SitesJeu[bis2SiteFlockPosition] = k.l6b
	bis2SitesJeu[bis2SiteDisplayAsset] = k.l6b
	if k.l6a && len(x.bornes) > 0 {
		bis2C3 = &bis2EtatC3{parIndex: true, bornes: x.bornes}
	}
}

func rcLever() {
	bis2Intercepteur = nil
	bis2SitesJeu = [bis2NombreDeSites]bool{}
	bis2WaypointQueue, bis2QueuesSansCondition = false, false
	bis2C3 = nil
}

// marcher joue une configuration ; `col` / `diag` : brancher les derivations d oracle.
func (x *rcFilm) marcher(k rcComps, oracle map[[2]int][]cmLiaison, col, diag bool) (*rcObs, *cmCollecteur, *b3Diag) {
	x.poser(k)
	defer rcLever()
	o := &rcObs{chk: nouveauCollecteur(x.f, x.b, nil), statut: map[[2]int]uint8{}}
	mux := cmMux{o}
	var c *cmCollecteur
	var d *b3Diag
	if col {
		wr := profile.QuantRangeCEBiped()
		cre, _, _ := ScanVehicleCreations(x.f.fc, &wr)
		c = nouveauCollecteur(x.f, x.b, cre)
		mux = append(mux, c)
	}
	if diag {
		d = b3NouveauDiag(x.f, x.b, false)
		mux = append(mux, d)
	}
	r, obs, _ := cmMarcher(x.f, cmVariante{oracle: oracle}, mux)
	o.a.paquets, o.a.utilesLus = r.Paquets, r.Utiles.Records
	o.a.horsCadre, o.a.rejets = r.Bloquants[CauseHorsCadre].Paquets, obs.RejetsHorsDatum
	return o, c, d
}

// rcL1 : l oracle L1 (regions (i) et (ii)) d un collecteur.
func rcL1(c *cmCollecteur) map[[2]int][]cmLiaison {
	return cmUnion(c.parRegion[cmNomsDeRegion[0].region], c.parRegion[cmNomsDeRegion[1].region])
}

// rcNomLeviers nomme une configuration par ses leviers.
func rcNomLeviers(k rcComps, l1, l9 bool) string {
	var n []string
	for _, x := range []struct {
		on  bool
		nom string
	}{{l1, "L1"}, {k.l8, "L8"}, {k.l2, "L2"}, {l9, "L9"}, {k.l6a, "L6a"}, {k.l6b, "L6b"}} {
		if x.on {
			n = append(n, x.nom)
		}
	}
	if len(n) == 0 {
		return "aucun"
	}
	return strings.Join(n, "+")
}

// TestRComb joue R-COMB sur les films de CAMPAGNE_FILMS.
func TestRComb(t *testing.T) {
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
	utiles := cmUtiles(t)
	lignes := []string{rcEntete}
	defer rcLever()
	for _, id := range films {
		lignes = append(lignes, rcUnFilm(t, racine, id, cartes[id], cat, utiles)...)
	}
	b2Ecrire(t, sortie, "r_comb_configs.tsv", lignes)
}

const rcEntete = "film\tbuild\tcontexte\tconfig\tleviers\trejoue\tliaisons_l1\tliaisons_l9\tpaquets\tfermes\t" +
	"fermes_sains\tutiles_fermes\tutiles_fermes_sains\tutiles_lus\thors_cadre\trejets_hors_datum\t" +
	"entrees_utiles_fermees\tentrees_utiles_saines\tg_ref\tp_ref\tgs_ref\tps_ref\tg_full\tp_full\tgs_full\tps_full"

// rcUnFilm mesure un film : toutes les configurations, dans l ordre des derivations.
func rcUnFilm(t *testing.T, racine, id, carte string, cat *profile.MapQuantCatalog, utiles UsagesProduit) []string {
	garde := filmproc.Arm("campagne/r-comb", 4, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
		os.Exit(3)
	})
	defer garde.Disarm()
	debut := time.Now()
	x := &rcFilm{contexte: "instruments"}
	var controle *rcObs
	if carte != "" {
		entry, err := cat.Lookup(carte)
		if err != nil {
			t.Errorf("%s : carte %q hors catalogue (%v)", id, carte, err)
			return nil
		}
		if fi, ok := cmOuvrir(t, racine, id, utiles); ok {
			xi := &rcFilm{f: fi, b: cmLireBlocs(fi)}
			controle, _, _ = xi.marcher(rcComps{}, nil, false, false)
		}
		f, ok := b2pOuvrirProduction(t, racine, id, entry, utiles)
		if !ok {
			return nil
		}
		x.f, x.contexte, x.bornes = f, "production:"+carte, b2pBornes(entry.Module)
	} else {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			return nil
		}
		x.f = f
	}
	x.b = cmLireBlocs(x.f)
	res := rcConfigurations(x)
	var out []string
	ligne := func(r *rcRes) {
		rj := "oui"
		if !r.rejoue {
			rj = "non (identique par construction)"
		}
		a := r.a
		out = append(out, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
			id, x.f.build, x.contexte, r.nom, r.leviers, rj, r.nl1, r.nl9, a.paquets, a.fermes, a.sains,
			a.utilesFermes, a.utilesSains, a.utilesLus, a.horsCadre, a.rejets, a.entreesUtiles, a.entreesUtilesSaines,
			r.vsRef.g, r.vsRef.p, r.vsRef.gs, r.vsRef.ps, r.vsFull.g, r.vsFull.p, r.vsFull.gs, r.vsFull.ps))
	}
	if controle != nil {
		ligne(&rcRes{nom: "controle:reference-instruments", leviers: "aucun", rejoue: true, a: controle.a})
	}
	for _, r := range res {
		ligne(r)
	}
	t.Logf("%s %s (%s) : %d configurations ; pic %d Mio, %s", id, x.f.build, x.contexte, len(res),
		garde.Peak()>>20, time.Since(debut).Round(time.Second))
	return out
}

// rcConfigurations joue les configurations d un film. Ordre : reference, chaine de la
// combinaison complete, leviers seuls, puis la combinaison privee de chaque levier de composant.
func rcConfigurations(x *rcFilm) []*rcRes {
	lf := len(x.bornes) > 0
	var res []*rcRes
	var ref, full *rcRes
	add := func(nom string, k rcComps, l1, l9 bool, o *rcObs, nl1, nl9 int) *rcRes {
		r := &rcRes{nom: nom, leviers: rcNomLeviers(k, l1, l9), rejoue: true, nl1: nl1, nl9: nl9, a: o.a, statut: o.statut}
		res = append(res, r)
		return r
	}
	// Une chaine de derivation pour un jeu de composants : A (composants seuls), B (+ L9), C (+ L9 + L1).
	chaine := func(pref string, k rcComps) (a, b, c *rcRes, l1SansL9 map[[2]int][]cmLiaison) {
		oA, colA, diagA := x.marcher(k, nil, true, true)
		l9 := diagA.imageCle
		l1SansL9 = rcL1(colA)
		a = add(pref+"composants", k, false, false, oA, 0, 0)
		oB, colB, _ := x.marcher(k, l9, true, false)
		b = add(pref+"+L9", k, false, true, oB, 0, b3Compter(l9))
		l1 := rcL1(colB)
		oC, _, _ := x.marcher(k, cmUnion(l9, l1), false, false)
		c = add(pref+"+L9+L1", k, true, true, oC, b3Compter(l1), b3Compter(l9))
		return a, b, c, l1SansL9
	}
	tous := rcComps{l8: true, l2: true, l6a: lf, l6b: true}
	ref, _, _, l1Ref := chaine("ref:", rcComps{})
	ref.nom = "reference"
	res[1].nom, res[2].nom = "L9", "L1+L9"
	o, _, _ := x.marcher(rcComps{}, l1Ref, false, false)
	add("L1", rcComps{}, true, false, o, b3Compter(l1Ref), 0)
	compsRes, sansL1, fullR, l1Full := chaine("full:", tous)
	compsRes.nom, sansL1.nom, fullR.nom = "full-L1-L9 (composants)", "full-L1", "full"
	full = fullR
	o, _, _ = x.marcher(tous, l1Full, false, false)
	add("full-L9", tous, true, false, o, b3Compter(l1Full), 0)
	seuls := []struct {
		nom string
		k   rcComps
		ok  bool
	}{{"L8", rcComps{l8: true}, true}, {"L2", rcComps{l2: true}, true}, {"L6a", rcComps{l6a: true}, lf},
		{"L6b", rcComps{l6b: true}, true}}
	for _, s := range seuls {
		if !s.ok {
			c := *ref
			c.nom, c.leviers, c.rejoue = s.nom, "L6a (sans borne : aucun effet)", false
			res = append(res, &c)
			continue
		}
		o, _, _ := x.marcher(s.k, nil, false, false)
		add(s.nom, s.k, false, false, o, 0, 0)
	}
	moins := []struct {
		nom string
		k   rcComps
		ok  bool
	}{{"L8", rcComps{l2: true, l6a: lf, l6b: true}, true}, {"L2", rcComps{l8: true, l6a: lf, l6b: true}, true},
		{"L6a", rcComps{l8: true, l2: true, l6b: true}, lf}, {"L6b", rcComps{l8: true, l2: true, l6a: lf}, true}}
	for _, m := range moins {
		noms := [3]string{"full-" + m.nom + "-L1-L9", "full-" + m.nom + "-L1", "full-" + m.nom}
		if !m.ok {
			for i, src := range []*rcRes{compsRes, sansL1, full} {
				c := *src
				c.nom, c.rejoue = noms[i], false
				res = append(res, &c)
			}
			continue
		}
		a, b, c, _ := chaine("", m.k)
		a.nom, b.nom, c.nom = noms[0], noms[1], noms[2]
	}
	for _, r := range res {
		r.vsRef = rcComparer(r.statut, ref.statut)
		r.vsFull = rcComparer(r.statut, full.statut)
	}
	return res
}
