package replay

// pads_proximity_research_test.go — LE RAFFINEMENT DE L'ETAT INCERTAIN D'UN SOCLE, MESURE AVANT
// D'ETRE IMPLANTE (item A-2 du plan .ai/V7.5/replay2d/PLAN_RETOURS_REJEU_2026-08-27.md).
//
// LA THESE A EPROUVER est celle de l'utilisateur (retour du 2026-08-27) : « si pas de joueurs
// ont pickup c'est que l'etat n'a pas du changer ». Autrement dit, dans l'intervalle [tLow,
// tHigh) ou le film ne dit RIEN, l'arme est encore la tant qu'aucun joueur n'a pu la prendre —
// et personne ne prend une arme sans passer dessus.
//
// LA REGLE QU'ON MESURE, ET RIEN D'AUTRE : une « approche » est un passage d'un joueur VIVANT a
// moins de R metres du socle (distance 2D, coordonnees monde) dans la fenetre [tLow, tHigh].
// Si la regle est juste, toute occupation ACHEVEE — dont l'absence est PROUVEE avant la fin du
// film — porte au moins une approche : l'arme a bien ete prise par quelqu'un qui est passe la.
// Une occupation achevee SANS aucune approche est une CONTRADICTION de la regle.
//
// LE SEUIL EST ECRIT AVANT LA MESURE (plan, item A-2) : la regle s'implante si la contradiction
// est <= 5 % au R retenu, et on retient le PLUS PETIT R qui tient ce seuil. Si aucun ne le
// tient, A-3 est REFUSE et le negatif chiffre reste ici.
//
// LES POINTS D'UNE TRACE SONT ECHANTILLONNES, et c'est le piege de la mesure : tester la seule
// distance AUX POINTS laisse passer un joueur qui traverse le socle entre deux echantillons.
// L'instrument teste donc la distance au SEGMENT entre deux points consecutifs, apres l'avoir
// coupe a la fenetre — c'est-a-dire la trajectoire telle que le rejeu la dessine.
//
// LE DENOMINATEUR EST CELUI QUE L'ARTEFACT PUBLIE, ET PAS UN AUTRE (corrige le 2026-08-27, revue
// adversariale) : `ReplayDocument.PadPickups` EST la liste des occupations ACHEVEES — le
// constructeur y ecrit tout statut SAUF `Never` (ground_weapon_pads.go, `gwBuildPad`). Le
// reconstruire depuis `Presence` etait une DEUXIEME regle, plus etroite, et sa legende etait
// fausse : « borne haute = fin du film » ne veut PAS dire « jamais videe », parce que le chemin
// `NoLaterKF` est ensuite rabattu par `lifeEnd` (ground_weapon_bounds.go). Cette reconstruction
// reste PUBLIEE comme LIGNE DE CONTROLE : l'ecart entre les deux listes dit quelque chose, et les
// occupations que la regle etroite ecartait sont en majorite des contradictions.
//
// DEUX ANOMALIES DE DONNEE, VUES PAR LA SONDE ET NON CORRIGEES ICI (elles appartiennent au
// producteur de l'artefact, pas a cet instrument) :
//   - ELARGIR LA FENETRE de +/- 60 images resout ~10 % des contradictions (26,71 % -> 23,89 % sur
//     la ligne de controle, R = 2 m) : une part du negatif est une LARGEUR DE BORNES, pas un
//     comportement de joueur ;
//   - sur `00162144` socle #10, l'occupation SUIVANTE a `t0 = 2628` alors que la precedente porte
//     `tHigh = 2631` : les fenetres se CHEVAUCHENT, la reapparition precede la preuve d'absence.
//
// LECTURE SEULE, aucune base, aucun decodage de film : l'instrument relit les ARTEFACTS JSON
// deja cuits (`ReplayDocument`) et ne touche a rien.
//
// USAGE (depuis apps/go-api) :
//
//	PADS_PROX_CORPUS=<repo>/data/cache/replays/halo_infinite \
//	  go test ./internal/analysis/replay/ -run '^TestPadsProximity$' -v -count=1

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// padsProxCorpusEnv designe le REPERTOIRE des artefacts JSON a relire (lecture seule).
const padsProxCorpusEnv = "PADS_PROX_CORPUS"

// LES RAYONS TESTES, en metres monde. Ecrits avant la mesure : 1,0 m est la portee de ramassage
// serree, 2,0 m la large ; 1,5 m est le defaut propose par la decision D2 du plan.
var padsProxRadii = []float64{1.0, 1.5, 2.0}

// LE SEUIL D'IMPLANTATION, ecrit avant la mesure (item A-2) : au-dela, la regle contredit trop
// souvent la donnee pour etre appliquee a l'ecran.
const padsProxMaxContradiction = 0.05

// padsProxDoc est un artefact relu : ce que la mesure lui prend, et rien de plus.
type padsProxDoc struct {
	name    string
	frameMs float64
	end     int
	doc     ReplayDocument
}

// padsProxStat accumule la mesure d'UN rayon sur tout le corpus.
type padsProxStat struct {
	radius float64
	// achevees : le DENOMINATEUR — les occupations achevees, telles que l'artefact les publie.
	achevees int
	// sans : le NUMERATEUR — celles qu'aucune approche n'explique.
	sans int
	// offsets : (tFirst - tLow) en secondes, pour les occupations qui portent une approche.
	offsets []float64
	// retreci : (tFirst - tLow) / (tHigh - tLow), la part de la fenetre que le raffinement
	// rendrait a l'etat PLEIN.
	retreci []float64
	// parFilm : [contradictions, total] par artefact — c'est lui qui dit si le negatif est
	// REPARTI ou PORTE PAR QUELQUES FILMS.
	parFilm map[string][2]int
}

// padsProxPickupWindows rend le denominateur AUTORITAIRE : une fenetre par occupation ACHEVEE,
// telle que `padPickups` la publie. L'index de socle vient de l'artefact ; un index hors bornes
// serait un artefact corrompu, il est compte a part plutot qu'ignore en silence.
func padsProxPickupWindows(d padsProxDoc) ([]padsProxWindow, int) {
	out, aberrants := make([]padsProxWindow, 0, len(d.doc.PadPickups)), 0
	for _, p := range d.doc.PadPickups {
		if p.Pad < 0 || p.Pad >= len(d.doc.WeaponPads) {
			aberrants++
			continue
		}
		pad := d.doc.WeaponPads[p.Pad]
		out = append(out, padsProxWindow{
			cx: float64(pad.X), cy: float64(pad.Y),
			lo: float64(p.TLow), hi: float64(p.THigh),
		})
	}
	return out, aberrants
}

// padsProxPresenceWindows rend la LIGNE DE CONTROLE : les occupations qu'une lecture reconstruite
// depuis `Presence` retiendrait (absence prouvee ET borne haute avant la fin du rejeu). Plus
// etroite que `padPickups` — l'ecart est publie, il fait partie du resultat.
func padsProxPresenceWindows(d padsProxDoc) ([]padsProxWindow, int) {
	out := make([]padsProxWindow, 0, len(d.doc.WeaponPads))
	for _, pad := range d.doc.WeaponPads {
		for _, o := range pad.Presence {
			if o.THigh <= o.TLow || o.THigh >= d.end {
				continue
			}
			out = append(out, padsProxWindowOf(pad, o))
		}
	}
	return out, 0
}

func TestPadsProximity(t *testing.T) {
	dir := os.Getenv(padsProxCorpusEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", padsProxCorpusEnv)
	}
	docs := padsProxLoad(t, dir)
	if len(docs) == 0 {
		t.Fatalf("aucun artefact avec des socles sous %s", dir)
	}
	padsProxLogCorpus(t, docs)

	stats := make([]padsProxStat, 0, len(padsProxRadii))
	for _, r := range padsProxRadii {
		stats = append(stats, padsProxMeasure(docs, r, padsProxPickupWindows))
	}
	padsProxReport(t, stats)
	// LA LIGNE DE CONTROLE : la meme mesure sur la liste RECONSTRUITE depuis `Presence`. Elle ne
	// decide rien ; elle montre de combien une regle plus etroite flatte le resultat.
	t.Logf("CONTROLE (denominateur reconstruit depuis Presence, plus etroit) :")
	ctrl := make([]padsProxStat, 0, len(padsProxRadii))
	for _, r := range padsProxRadii {
		st := padsProxMeasure(docs, r, padsProxPresenceWindows)
		ctrl = append(ctrl, st)
		t.Logf("CONTROLE\tR=%.1f m\t%d/%d\tcontradiction %.2f %%",
			r, st.sans, st.achevees, padsProxRatio(st.sans, st.achevees)*100)
	}
	padsProxByFilm(t, "AUTORITAIRE", padsProxLargest(stats))
	padsProxByFilm(t, "CONTROLE", padsProxLargest(ctrl))
	padsProxControl(t, docs)
}

// padsProxLargest rend la mesure du plus GRAND rayon : c'est celle qui donne au negatif sa
// meilleure chance, donc celle sur laquelle une repartition par film se lit sans complaisance.
func padsProxLargest(stats []padsProxStat) padsProxStat {
	var out padsProxStat
	for _, st := range stats {
		if st.radius > out.radius {
			out = st
		}
	}
	return out
}

// padsProxByFilm publie la REPARTITION du negatif : sans elle, un taux global se lit comme un
// comportement general alors qu'il peut n'etre porte que par quelques artefacts.
//
// CE QU'IL FAUT EN RETENIR (mesure du 2026-08-27) : deux films concentrent l'essentiel des
// contradictions, avec des distances minimales medianes en DIZAINES de metres la ou le reste du
// corpus est a ~0,25 m — c'est la signature d'une population de socles douteuse cote artefact,
// pas d'un comportement de joueur. Le verdict ne change pas pour autant : meme la MEDIANE des
// taux par film reste au-dessus du seuil.
func padsProxByFilm(t *testing.T, base string, st padsProxStat) {
	t.Helper()
	if st.parFilm == nil {
		return
	}
	noms := make([]string, 0, len(st.parFilm))
	for nom := range st.parFilm {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	taux := make([]float64, 0, len(noms))
	var pires []string
	var sansPires, totalSansPires int
	for _, nom := range noms {
		c := st.parFilm[nom]
		part := padsProxRatio(c[0], c[1])
		taux = append(taux, part)
		if part > padsProxFilmSuspect {
			pires = append(pires, fmt.Sprintf("%s (%d/%d = %.1f %%)", nom, c[0], c[1], part*100))
			continue
		}
		sansPires, totalSansPires = sansPires+c[0], totalSansPires+c[1]
	}
	hautSans, hautTotal, toutSans, toutTotal := padsProxTop(st, 2)
	t.Logf("PAR FILM [%s] (R = %.1f m) — mediane des taux %.2f %% · %d films au-dessus de %.0f %% · les 2 pires portent %.1f %% des contradictions",
		base, st.radius, padsProxQuantile(taux, 0.5)*100, len(pires), padsProxFilmSuspect*100,
		padsProxRatio(hautSans, toutSans)*100)
	for _, p := range pires {
		t.Logf("PAR FILM [%s]\tSUSPECT\t%s", base, p)
	}
	t.Logf("PAR FILM [%s] — hors les 2 PIRES films : %d/%d = %.2f %% · hors les %d SUSPECTS : %d/%d = %.2f %%",
		base, toutSans-hautSans, toutTotal-hautTotal,
		padsProxRatio(toutSans-hautSans, toutTotal-hautTotal)*100,
		len(pires), sansPires, totalSansPires, padsProxRatio(sansPires, totalSansPires)*100)
}

// padsProxTop rend, pour les `n` PIRES films (par nombre de contradictions), leurs contradictions
// et leur total d'occupations — puis les memes deux quantites sur TOUT le corpus. C'est de ces
// quatre nombres que se lit si un negatif global est un comportement ou une poignee d'artefacts.
func padsProxTop(st padsProxStat, n int) (int, int, int, int) {
	comptes := make([][2]int, 0, len(st.parFilm))
	var toutSans, toutTotal int
	for _, c := range st.parFilm {
		comptes = append(comptes, c)
		toutSans, toutTotal = toutSans+c[0], toutTotal+c[1]
	}
	sort.Slice(comptes, func(i, j int) bool { return comptes[i][0] > comptes[j][0] })
	var hautSans, hautTotal int
	for i := 0; i < n && i < len(comptes); i++ {
		hautSans, hautTotal = hautSans+comptes[i][0], hautTotal+comptes[i][1]
	}
	return hautSans, hautTotal, toutSans, toutTotal
}

// Au-dela de ce taux, un film est SIGNALE a part : ce n'est pas un seuil de decision, c'est le
// niveau a partir duquel un artefact merite d'etre regarde pour lui-meme.
const padsProxFilmSuspect = 0.20

// LES RAYONS DE CONTROLE, en metres : ils ne decident RIEN, ils disent seulement jusqu'ou il
// faudrait ouvrir la regle pour qu'elle cesse de contredire la donnee.
var padsProxControlRadii = []float64{1.0, 1.5, 2.0, 3.0, 5.0, 10.0}

// padsProxControl est LE TEMOIN qui separe deux negatifs tres differents : « la regle est
// fausse » et « la donnee ne permet pas de la tester ».
//
// CE QU'IL PUBLIE : la distance MINIMALE du socle a une trajectoire pendant la fenetre, sa
// distribution, et la part des fenetres qu'AUCUNE trajectoire ne couvre. Si le plus proche
// passage median se compte en metres et non en dizaines de metres, la regle vise juste et c'est
// le rayon qui est trop serre ; si des fenetres entieres sont sans trajectoire, l'instrument
// mesure un trou de couverture et non un comportement de joueur.
//
// LA COUVERTURE SE COMPTE EN SEGMENTS, PAS EN POINTS (corrige le 2026-08-27, revue
// adversariale) : le temoin jetait les fenetres sans aucun ECHANTILLON a l'interieur, alors
// qu'un segment qui les TRAVERSE donne une distance parfaitement definie — il ecartait donc des
// mesures qu'il venait de calculer, et toutes du meme cote (des passages lointains).
func padsProxControl(t *testing.T, docs []padsProxDoc) {
	t.Helper()
	dists, sansEch, sansTraj := padsProxDistances(docs, padsProxPickupWindows)
	t.Logf("TEMOIN — %d fenetres SANS aucune trajectoire (exclues) · %d couvertes par un segment TRAVERSANT sans echantillon interieur (comptees)",
		sansTraj, sansEch)
	t.Logf("TEMOIN — passage le plus proche : median %.2f m · p90 %.2f m (sur %d fenetres couvertes)",
		padsProxQuantile(dists, 0.5), padsProxQuantile(dists, 0.9), len(dists))
	for _, r := range padsProxControlRadii {
		dedans := padsProxCountWithin(dists, r)
		t.Logf("TEMOIN\tR=%.1f m\tapproches %d/%d\tcontradiction %.2f %%",
			r, dedans, len(dists), (1-padsProxRatio(dedans, len(dists)))*100)
	}
	// LE MEME TEMOIN SUR LA BASE ETROITE, pour que l'ecart entre les deux denominateurs soit
	// VERIFIABLE et non pas seulement affirme : c'est la base sur laquelle la premiere version de
	// cet instrument publiait ses chiffres.
	ctrl, _, _ := padsProxDistances(docs, padsProxPresenceWindows)
	t.Logf("TEMOIN [base Presence] — median %.2f m · p90 %.2f m · plancher R=10 m %.2f %% (sur %d fenetres)",
		padsProxQuantile(ctrl, 0.5), padsProxQuantile(ctrl, 0.9),
		(1-padsProxRatio(padsProxCountWithin(ctrl, 10), len(ctrl)))*100, len(ctrl))
}

// padsProxDistances rend la distance du passage le plus proche pour chaque fenetre COUVERTE, le
// nombre de fenetres couvertes sans echantillon interieur, et celles qu'aucune trajectoire ne
// couvre (seules exclues).
func padsProxDistances(
	docs []padsProxDoc, source func(padsProxDoc) ([]padsProxWindow, int),
) ([]float64, int, int) {
	dists := make([]float64, 0, 1024)
	var sansEchantillon, sansTrajectoire int
	for _, d := range docs {
		fenetres, _ := source(d)
		for _, w := range fenetres {
			dist, n, couverte := padsProxMinDistance(d.doc.Tracks, w)
			if !couverte {
				sansTrajectoire++
				continue
			}
			if n == 0 {
				sansEchantillon++
			}
			dists = append(dists, dist)
		}
	}
	return dists, sansEchantillon, sansTrajectoire
}

// padsProxLoad relit les artefacts JSON du repertoire. Un fichier illisible FAIT ECHOUER la
// mesure : un corpus silencieusement ampute rendrait un taux faux sans le dire.
func padsProxLoad(t *testing.T, dir string) []padsProxDoc {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("corpus illisible : %v", err)
	}
	sort.Strings(paths)
	out := make([]padsProxDoc, 0, len(paths))
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s illisible : %v", filepath.Base(p), err)
		}
		var doc ReplayDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s non deserialisable : %v", filepath.Base(p), err)
		}
		if len(doc.WeaponPads) == 0 || doc.FrameCount <= 1 || doc.DurationMS <= 0 {
			continue
		}
		out = append(out, padsProxDoc{
			name:    filepath.Base(p),
			frameMs: float64(doc.DurationMS) / float64(doc.FrameCount),
			end:     doc.FrameCount - 1,
			doc:     doc,
		})
	}
	return out
}

// padsProxLogCorpus publie ce que le corpus contient AVANT toute regle : films, socles,
// occupations, le denominateur AUTORITAIRE et l'ecart avec la ligne de controle.
func padsProxLogCorpus(t *testing.T, docs []padsProxDoc) {
	t.Helper()
	var pads, occ, tracks, pickups, aberrants, controle int
	for _, d := range docs {
		pads += len(d.doc.WeaponPads)
		tracks += len(d.doc.Tracks)
		for _, p := range d.doc.WeaponPads {
			occ += len(p.Presence)
		}
		fen, hors := padsProxPickupWindows(d)
		pickups, aberrants = pickups+len(fen), aberrants+hors
		ctrl, _ := padsProxPresenceWindows(d)
		controle += len(ctrl)
	}
	t.Logf("CORPUS — %d films · %d socles · %d traces · %d occupations publiees", len(docs), pads, tracks, occ)
	t.Logf("DENOMINATEUR AUTORITAIRE — %d occupations ACHEVEES (`padPickups`, tout statut sauf Never)",
		pickups)
	t.Logf("CONTROLE — %d retenues par la lecture reconstruite depuis Presence, soit %d de MOINS",
		controle, pickups-controle)
	if aberrants > 0 {
		t.Logf("ANOMALIE — %d occupations designent un socle hors bornes (artefact incoherent)", aberrants)
	}
}

// padsProxMeasure rejoue la regle a UN rayon sur tout le corpus, sur le denominateur que
// `source` designe (les occupations publiees, ou la ligne de controle reconstruite).
func padsProxMeasure(
	docs []padsProxDoc, r float64, source func(padsProxDoc) ([]padsProxWindow, int),
) padsProxStat {
	st := padsProxStat{radius: r, parFilm: make(map[string][2]int, len(docs))}
	for _, d := range docs {
		fenetres, _ := source(d)
		for _, w := range fenetres {
			st.achevees++
			compte := st.parFilm[d.name]
			compte[1]++
			first, ok := padsProxFirstApproach(d.doc.Tracks, w, r)
			if !ok {
				st.sans++
				compte[0]++
				st.parFilm[d.name] = compte
				continue
			}
			st.parFilm[d.name] = compte
			offset := first - w.lo
			st.offsets = append(st.offsets, offset*d.frameMs/1000)
			// UNE FENETRE DE LARGEUR NULLE N'A PAS DE RETRECISSEMENT a offrir : `padPickups` en
			// publie (borne haute rabattue sur la basse), et les compter diviserait par zero.
			if w.hi > w.lo {
				st.retreci = append(st.retreci, offset/(w.hi-w.lo))
			}
		}
	}
	return st
}

// padsProxReport publie le tableau R x (contradiction, offsets, retrecissement) et le VERDICT
// contre le seuil ecrit avant la mesure.
func padsProxReport(t *testing.T, stats []padsProxStat) {
	t.Helper()
	t.Logf("SEUIL — la regle s'implante si contradiction <= %.0f %% (le plus petit R qui tient)",
		padsProxMaxContradiction*100)
	t.Logf("R(m)\tachevees\tsans approche\tcontradiction\toffset median(s)\toffset p90(s)\tretreci median")
	retenu := math.NaN()
	for _, st := range stats {
		part := padsProxRatio(st.sans, st.achevees)
		t.Logf("%.1f\t%d\t%d\t%.2f %%\t%.2f\t%.2f\t%.3f",
			st.radius, st.achevees, st.sans, part*100,
			padsProxQuantile(st.offsets, 0.5), padsProxQuantile(st.offsets, 0.9),
			padsProxQuantile(st.retreci, 0.5))
		if math.IsNaN(retenu) && st.achevees > 0 && part <= padsProxMaxContradiction {
			retenu = st.radius
		}
	}
	if math.IsNaN(retenu) {
		t.Logf("VERDICT — AUCUN rayon ne tient le seuil : la regle est REFUSEE (A-3 non implante)")
		return
	}
	t.Logf("VERDICT — seuil tenu au plus petit R = %.1f m : la regle s'implante a ce rayon", retenu)
}

// LA GEOMETRIE ET LES RESUMES DE DISTRIBUTION VIVENT A COTE (`pads_proximity_geometry_test.go`) :
// approche, distance a un segment coupe, quantiles. Ils ne connaissent ni artefact ni seuil.
