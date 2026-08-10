package himap

import (
	"fmt"
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
	var sansCoquille, absentes, nonMesurables []string

	for _, mod := range modules {
		// UN SOUS-TEST PAR CARTE : un `t.Fatal` dans `peupleRendu` (module illisible) ne doit tuer
		// QUE sa carte. Le premier balayage complet est mort en entier sur une seule — meme
		// piege que celui deja documente pour `TestBalayageDesCartes`.
		mod := mod
		t.Run(mod, func(t *testing.T) {
			chemin, ok := ChercheModuleInstalle(mod)
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
			// EXCEPTION CONNUE, signalee et non avalee : `live_fire` (sgh_interlock) ne porte AUCUN
			// tag sbsp — instruit au §1 ter du handoff, sa geometrie n'est pas la ou celle des
			// 26 autres se trouve. `peupleRendu` y ferait `t.Fatal` ; on la declare non mesurable
			// AVEC SA RAISON plutot que de rougir sur un cas deja tranche.
			if _, err := ReadModuleInstances(chemin); err != nil {
				nonMesurables = append(nonMesurables, fmt.Sprintf("%s (%v)", mod, err))
				return
			}
			b := bilan{carte: mod}
			rendu := cadreSurAncres(t, ancres)
			zJeu := MedianeZ(ancres) - AncrageDecalageSol
			rendu.Tranche(TrancheDeJeu(zJeu))
			peupleRendu(t, rendu, racine, chemin, ancres)
			b.ancresAvant, b.pxAvant = compteAncresEtPixels(rendu, ancres)

			fr := frontiereDeLaCarte(t, chemin)
			if len(fr.Frontieres) == 0 {
				sansCoquille = append(sansCoquille, mod)
				return
			}
			b.plans = len(fr.Frontieres)
			if !FrontiereGardeLesAncres(fr, ancres, zJeu) {
				b.ancresApres, b.pxApres = b.ancresAvant, b.pxAvant
				b.refusee = true
				bilans = append(bilans, b)
				return
			}
			rendu.RestreintALaFrontiere(fr, zJeu)
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
		t.Logf("%s %-34s %d tris · ancres %2d -> %2d · pixels %7d -> %7d (%.1f %% retires)",
			marque, b.carte, b.plans, b.ancresAvant, b.ancresApres, b.pxAvant, b.pxApres,
			100*float64(b.pxAvant-b.pxApres)/float64(max(b.pxAvant, 1)))
	}
	if len(sansCoquille) > 0 {
		t.Logf("%d cartes SANS coquille exploitable : %s", len(sansCoquille), strings.Join(sansCoquille, ", "))
	}
	if len(absentes) > 0 {
		t.Logf("%d modules absents de l'installation", len(absentes))
	}
	if len(nonMesurables) > 0 {
		t.Logf("%d cartes NON MESURABLES : %s", len(nonMesurables), strings.Join(nonMesurables, " · "))
	}
	t.Logf("BILAN : %d cartes mesurees · %d coquilles REFUSEES (elles excluaient des ancres) · %d perdent des ancres",
		len(bilans), refusees, perdantes)

	// LE GATE, ET IL MANQUAIT (pose le 2026-08-10, lot A). Ce test ne faisait que JOURNALISER :
	// « 0 ancre perdue » n'etait verifie que par un humain qui lisait la sortie, et un run sans
	// `-v` rendait `ok` quoi qu'il arrive. Un gate qui ne peut pas rougir n'est pas un gate —
	// c'est la faute que ce chantier paie depuis le debut, sous une autre forme.
	if len(bilans) == 0 {
		t.Fatal("AUCUNE carte mesuree — le balayage n'a rien fait (reference du 2026-08-10 : 25 cartes)")
	}
	if perdantes > 0 {
		t.Errorf("%d cartes PERDENT des ancres a cause de la coquille (reference du 2026-08-10 : 0) "+
			"— une ancre perdue est de la zone jouable effacee", perdantes)
	}
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

func frontiereDeLaCarte(t *testing.T, cheminModule string) Sddt {
	t.Helper()
	chemin, err := ModuleVariante(cheminModule, "any")
	if err != nil {
		return Sddt{}
	}
	s, err := LitSddt(chemin)
	if err != nil {
		return Sddt{}
	}
	return s
}
