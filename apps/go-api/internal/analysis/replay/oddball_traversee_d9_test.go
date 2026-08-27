package replay

// oddball_traversee_d9_test.go — D9 : LE PORTAGE PAR LA PREMIERE TRAVERSEE.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_
// 2026-08.md`, section « D9 »). Ce qui suit l'applique.
//
// # LE PREDICAT CHANGE, ET C'EST CE QUI DISTINGUE CETTE CAMPAGNE D'UN CINQUIEME REGLAGE
//
// D6 demandait « qui est la QUAND l'objet se tait » — 45,5 et 47,8 % des trous. D8 a mesure que
// 84,8 a 93,3 % des trous voient un joueur TRAVERSER le lieu de repos pendant leur duree, et a
// REFUTE le sommeil : l'objet est porte, pas endormi. Il ne manquait donc qu'un point d'entree.
// Ici la question est « qui passe le PREMIER sur l'objet pose ».
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM` + `ODDBALL_ORACLE`, UN FILM PAR PROCESSUS, lecture
// seule, AUCUNE base ouverte.

import (
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/filmproc"
)

const (
	// d9FenetreMS : le delai maximal entre le silence et la traversee retenue. HUIT SECONDES,
	// et la valeur sort de la mesure D8, pas d'un reglage : les q90 du delai valent 7,90 / 3,40
	// / 6,00 s sur les trois films sains, donc 8 s couvre le q90 de CHACUN. Au-dela s'etend une
	// queue jusqu'a 22,6 s — la zone ou « quelqu'un passe » cesse d'etre un ramassage pour
	// devenir une coincidence.
	d9FenetreMS = 8000
	// d9QueueMS : la queue complete de D8, rejouee en DIAGNOSTIC seulement. Elle ne valide rien.
	d9QueueMS = 22600
	// d9DecalageM : le decalage du temoin SPATIAL. Douze metres — le decalage de temoin deja
	// etabli au depot, huit fois le rayon de ramassage, et assez peu pour rester sur du sol foule.
	d9DecalageM = 12.0
	// d9Graine fixe les deux tirages : deux executions rendent la MEME sortie.
	d9Graine = 20260829
)

// TestOddballTraverseeD9 — LA MESURE FINALE. Un film par processus.
func TestOddballTraverseeD9(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", d4FilmEnv)
	}
	oracle, ok := d6Oracle(t)
	if !ok {
		return
	}
	g := filmproc.Arm("d9-traversee", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	e, ok := d8Charge(t, root, id)
	if !ok {
		return
	}
	dir := objChunkDir(root, id)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}

	opt := d9Options{fenetreMS: d9FenetreMS}
	rec, stats := d9Reconstruit(e, deaths, opt)
	t.Logf("D9 %s : %d trou(s) — %d ATTRIBUE(s), %d retour, %d sans traversee dans %d ms ; "+
		"%.0f s de portage attribuees", id, stats.trous, stats.attribues, stats.retours,
		stats.sansTraversee, d9FenetreMS, stats.secondes)

	rngJ := rand.New(rand.NewSource(d9Graine)) //nolint:gosec // temoin reproductible
	rngS := rand.New(rand.NewSource(d9Graine)) //nolint:gosec // temoin reproductible
	temJ, _ := d9Reconstruit(e, deaths, d9Options{fenetreMS: d9FenetreMS, hasard: rngJ})
	temS, _ := d9Reconstruit(e, deaths, d9Options{fenetreMS: d9FenetreMS, decale: true, hasard: rngS})
	d9Verdict(t, id, rec, temJ, temS, oracle)

	// DIAGNOSTIC HORS GATE : ce que la queue de D8 ajouterait. Il ne valide RIEN.
	queue, qs := d9Reconstruit(e, deaths, d9Options{fenetreMS: d9QueueMS})
	t.Logf("DIAGNOSTIC %s (HORS GATE, fenetre %d ms) : %d attribue(s), %.0f s, "+
		"recouvrement %.1f %%", id, d9QueueMS, qs.attribues, qs.secondes,
		100*d9Taux(queue, oracle))
}

// d9Options porte ce qui change entre la mesure et ses deux temoins (regle des 5 parametres).
type d9Options struct {
	fenetreMS int
	// decale bascule le TEMOIN SPATIAL : la position de repos est deplacee de d9DecalageM.
	decale bool
	// hasard NON NIL bascule le TEMOIN DE JOUEUR : un present tire au sort remplace le premier
	// traversant. Meme chaine, meme cloture, meme metrique.
	hasard *rand.Rand
}

// d9Stats compte ce que la reconstruction a rencontre. Sans ces denominateurs, un total de
// secondes ne se juge pas.
type d9Stats struct {
	trous, attribues, retours, sansTraversee int
	secondes                                 float64
}

// d9Reconstruit attribue chaque trou au PREMIER joueur qui traverse le lieu de repos.
func d9Reconstruit(e d8Etat, deaths []Death, opt d9Options) (map[string]float64, d9Stats) {
	out, st := map[string]float64{}, d9Stats{}
	for i := 0; i+1 < len(e.vies); i++ {
		fin := e.vies[i+1].T0US
		if fin <= e.vies[i].T1US {
			continue
		}
		st.trous++
		x, y := e.vies[i].Last()
		if opt.decale {
			x, y = x+d9DecalageM, y+d9DecalageM
		}
		debut := e.vies[i].T1US
		xuid, at, ok := d9PremierTraversant(e, debut, fin, x, y, opt.fenetreMS)
		if !ok {
			// UN TROU SANS TRAVERSEE N'EST PAS DEVINE. S'il se termine au socle, c'est un
			// RETOUR ; sinon il reste non attribue, et il pese au denominateur dans les deux cas.
			if d6NaitAuSocle(e.vies[i+1], e.socles) {
				st.retours++
			} else {
				st.sansTraversee++
			}
			continue
		}
		if opt.hasard != nil {
			xuid = d6TireAuHasard(e.pont, xuid, opt.hasard)
		}
		st.attribues++
		d := float64(d9FinPortage(xuid, at, fin, deaths, e.pont)-at) / 1e6
		if d <= 0 {
			continue
		}
		st.secondes += d
		out[strconv.FormatUint(xuid, 10)] += d
	}
	return out, st
}

// d9PremierTraversant rend le PREMIER joueur nomme a passer a portee du lieu de repos, et quand.
//
// LE PAS EST L'IMAGE, comme en D8 : on cherche un PASSAGE, et un joueur qui traverse est a portee
// pendant plusieurs images consecutives. La fenetre borne la recherche — au-dela, non attribue.
func d9PremierTraversant(e d8Etat, debutUS, finUS uint64, x, y float32, fenetreMS int,
) (uint64, uint64, bool) {
	limite := debutUS + uint64(fenetreMS)*1000
	if finUS < limite {
		limite = finUS
	}
	for at := debutUS; at <= limite; at += d7ImageUS {
		xuid, d, second := d6PlusProche(e.tracks, e.pont, at, x, y)
		if xuid == 0 || d > d6RayonRamassageM {
			continue
		}
		// AMBIGUITE : deux joueurs a portee et separes de moins de d6AmbiguiteM — le porteur est
		// null, doctrine inchangee des occupations de socle. Le trou reste NON attribue.
		if second-d < d6AmbiguiteM {
			return 0, 0, false
		}
		return xuid, at, true
	}
	return 0, 0, false
}

// d9FinPortage borne le portage : la fin du trou, ou la MORT du porteur si elle tombe avant.
// Regle fondee par la mesure D6 : 22 morts de porteur sur 24 sont suivies d'un lacher.
func d9FinPortage(xuid, debutUS, finUS uint64, deaths []Death, pont objBridge) uint64 {
	fin := finUS
	for _, d := range deaths {
		if d.XUID != xuid {
			continue
		}
		if at := uint64(d.TimeMS+pont.OffsetMS) * 1000; at > debutUS && at < fin {
			fin = at
		}
	}
	return fin
}

// d9Verdict confronte la reconstruction a l'oracle API et a SES DEUX TEMOINS.
func d9Verdict(t *testing.T, id string, rec, temJ, temS, oracle map[string]float64) {
	t.Helper()
	xuids := make([]string, 0, len(oracle))
	for x := range oracle {
		xuids = append(xuids, x)
	}
	sort.Strings(xuids)
	for _, x := range xuids {
		t.Logf("  %s : API %.0f s · reconstruit %.0f s · temoin joueur %.0f s · temoin spatial %.0f s",
			x[len(x)-4:], oracle[x], rec[x], temJ[x], temS[x])
	}
	r, tj, ts := d9Taux(rec, oracle), d9Taux(temJ, oracle), d9Taux(temS, oracle)
	principal := d6MemePrincipal(rec, oracle)
	t.Logf("SIGNAL %s : recouvrement %.1f %% (seuil %.0f %%) ; temoin JOUEUR %.1f %% ; "+
		"temoin SPATIAL %.1f %% (seuil <= %.0f %%) ; porteur principal %s",
		id, 100*r, 100*d6SeuilRecouvrement, 100*tj, 100*ts, 100*d6SeuilTemoin, d6Oui(principal))
	switch {
	case r >= d6SeuilRecouvrement && tj <= d6SeuilTemoin && ts <= d6SeuilTemoin:
		t.Logf("VERDICT %s : recouvrement ET les DEUX temoins sont tenus sur ce film.", id)
	case r >= d6SeuilRecouvrement:
		t.Logf("VERDICT %s : le recouvrement tient mais un TEMOIN aussi — le signal ne se "+
			"distingue pas de son controle.", id)
	default:
		t.Logf("VERDICT %s : le recouvrement est SOUS le seuil.", id)
	}
}

// d9Taux rend le recouvrement : somme des min(reconstruit, oracle) sur le total de l'oracle.
func d9Taux(rec, oracle map[string]float64) float64 {
	var total float64
	for _, s := range oracle {
		total += s
	}
	if total <= 0 {
		return 0
	}
	return d6Recouvrement(rec, oracle) / total
}
