package himap

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// LE GATE DE LA COQUILLE — elle transfere-t-elle aux 27 cartes, ou seulement aux deux qui
// l'ont vue naitre ?
//
// Methode du §1 ter : la regle se calibre sur la carte a oracle FORT (Cliffhanger, 29 221
// positions) et se VALIDE ailleurs avec l'oracle FAIBLE des ancres. Ici la question est
// binaire et sans gout : la coquille fait-elle PERDRE des ancres a une carte ?
//
// Une ancre perdue = de la zone jouable effacee. C'est disqualifiant, et ca doit se voir avant
// de livrer, pas apres.
func TestBalayageCoquille(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := chargeCatalogue(p)
	if err != nil {
		t.Fatal(err)
	}
	var modules []string
	vus := map[string]bool{}
	for _, e := range cat {
		if e.Module != "" && !vus[e.Module] {
			vus[e.Module] = true
			modules = append(modules, e.Module)
		}
	}
	sort.Strings(modules)
	if s := os.Getenv("BALAYAGE_CARTES"); s != "" {
		modules = strings.Split(s, ",")
	}

	type bilan struct {
		carte                    string
		ancresAvant, ancresApres int
		pxAvant, pxApres         int
		plans                    int
		refusee                  bool
	}
	var bilans []bilan
	var sansCoquille, absentes []string

	for _, mod := range modules {
		// UN SOUS-TEST PAR CARTE : un `t.Fatal` dans `peupleRendu` (module illisible) ne doit tuer
		// QUE sa carte. Le premier balayage complet est mort en entier sur une seule — meme
		// piege que celui deja documente pour `TestBalayageDesCartes`.
		mod := mod
		t.Run(mod, func(t *testing.T) {
			chemin, ok := chercheModule(racine, mod)
			if !ok {
				absentes = append(absentes, mod)
				return
			}
			var ancres [][3]float64
			for _, e := range ancresDuModule(t, mod) {
				for _, o := range e.Objectives {
					ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
				}
			}
			if len(ancres) == 0 {
				return
			}
			b := bilan{carte: mod}
			rendu := cadreSurAncres(t, ancres)
			rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
			zJeu := medianeZ(ancres) - AncrageDecalageSol
			peupleRendu(t, rendu, racine, chemin, ancres)
			b.ancresAvant, b.pxAvant = compteAncresEtPixels(rendu, ancres)

			coq := coquilleDeLaCarte(t, chemin)
			if len(coq) == 0 {
				sansCoquille = append(sansCoquille, mod)
				return
			}
			b.plans = len(coq)
			if !CoquilleGardeLesAncres(coq, ancres, zJeu) {
				b.ancresApres, b.pxApres = b.ancresAvant, b.pxAvant
				b.refusee = true
				bilans = append(bilans, b)
				return
			}
			rendu.RestreintALaCoquille(coq, zJeu)
			b.ancresApres, b.pxApres = compteAncresEtPixels(rendu, ancres)
			bilans = append(bilans, b)
		})
	}

	perdantes, refusees := 0, 0
	for _, b := range bilans {
		perdu := b.ancresAvant - b.ancresApres
		marque := "  "
		if b.refusee {
			marque = "--"
			refusees++
		}
		if perdu > 0 {
			marque = "!!"
			perdantes++
		}
		t.Logf("%s %-34s %d plans · ancres %2d -> %2d · pixels %7d -> %7d (%.1f %% retires)",
			marque, b.carte, b.plans, b.ancresAvant, b.ancresApres, b.pxAvant, b.pxApres,
			100*float64(b.pxAvant-b.pxApres)/float64(max(b.pxAvant, 1)))
	}
	if len(sansCoquille) > 0 {
		t.Logf("%d cartes SANS coquille exploitable : %s", len(sansCoquille), strings.Join(sansCoquille, ", "))
	}
	if len(absentes) > 0 {
		t.Logf("%d modules absents de l'installation", len(absentes))
	}
	t.Logf("BILAN : %d cartes mesurees · %d coquilles REFUSEES (elles excluaient des ancres) · %d perdent des ancres",
		len(bilans), refusees, perdantes)
}

func compteAncresEtPixels(r *Rendu, ancres [][3]float64) (int, int) {
	avecSol := 0
	for _, a := range ancres {
		i := int((a[0] - r.Min[0]) / r.Cell)
		j := int((a[1] - r.Min[1]) / r.Cell)
		if i < 0 || i >= r.NX || j < 0 || j >= r.NY {
			continue
		}
		if _, ok := r.Altitude(i, j); ok {
			avecSol++
		}
	}
	px := 0
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			if _, ok := r.Altitude(i, j); ok {
				px++
			}
		}
	}
	return avecSol, px
}

func coquilleDeLaCarte(t *testing.T, cheminModule string) CoquilleSddt {
	t.Helper()
	chemin, err := ModuleVariante(cheminModule, "any")
	if err != nil {
		return nil
	}
	s, err := LitSddt(chemin)
	if err != nil {
		return nil
	}
	return s.Coquille()
}
