package himap

// SONDE BOUILLIE (2026-08-13, INVESTIGATION_BOUILLIE_FORGE) — instrument sous garde,
// versionne le 2026-08-20 sur decision utilisateur (il skippe sans les fichiers du jeu :
// aucune execution en CI ; lancement CIBLE uniquement, cf. ci-dessous).
//
// Le gate visuel utilisateur a REFUSE EN BLOC les 33 fonds Forge de masse (« bouillie au
// milieu, forme patatoide gribouillee », Domicile « trop de toits ») pendant que Vagabond,
// Corpo et les natives sont validees et Starboard/Dredge « plutot ok ». La sonde cuit les
// deux populations avec la voie de reference ARMEE mais NON appliquee, et mesure ce qui les
// separe : profil vertical de la matiere au-dessus de la reference (le couvercle), part de
// matiere hors portee des ancres (la coquille non substituable), residu apres substitution
// simulee, rugosite du champ substitue (le gribouillage), qualite de la reference d'ancres.
//
// Lancement CIBLE :
//
//	go test ./internal/himap/ -run TestSondeBouillie -timeout 45m -v
//
// AUCUNE ecriture d'asset : mesure pure.

import (
	"context"
	"math"
	"testing"
)

var cartesSondeBouillie = []struct{ nom, verdict string }{
	{"Vagabond", "VALIDEE (ouverte)"},
	{"Corpo", "VALIDEE"},
	{"Starboard", "pilote plutot ok — toits residuels"},
	{"Dredge", "pilote plutot ok — toits residuels"},
	{"Goliath", "tolerable a la rigueur"},
	{"Domicile", "REFUSEE — trop de toits (covered=false)"},
	{"The Pit", "REFUSEE masse (covered=false)"},
	{"Fortress", "REFUSEE masse (covered=false)"},
	{"Empyrean", "REFUSEE masse (covered=false)"},
	{"Dynasty", "REFUSEE masse (covered=true)"},
	{"Absolution", "REFUSEE masse (covered=true)"},
	{"Smallhalla", "REFUSEE masse (covered=true)"},
}

func TestSondeBouillie(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	parNom := map[string]CarteForge{}
	for _, c := range CartesForge {
		parNom[c.Nom] = c
	}
	for _, cs := range cartesSondeBouillie {
		c, ok := parNom[cs.nom]
		if !ok {
			t.Errorf("%s : absente de CartesForge", cs.nom)
			continue
		}
		cs := cs
		t.Run(cs.nom, func(t *testing.T) {
			mesureBouillie(t, root, c, cs.verdict)
		})
	}
}

func mesureBouillie(t *testing.T, root string, carte CarteForge, verdict string) {
	t.Helper()
	v, entree := chargeCarteForge(t, carte)
	var ancres [][3]float64
	for _, o := range entree.Objectives {
		ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	if len(ancres) == 0 {
		t.Skip("aucune ancre")
	}

	s := NewSurfaceReference(ancres)
	r := CadreSurAncres(ancres)
	zJeu := MedianeZ(ancres) - AncrageDecalageSol
	r.Tranche(TrancheDeJeu(zJeu))
	r.NiveauDeJeu(zJeu)
	r.ArmeReference(s)
	opts := OptionsCuissonForge{
		RacineDeploy:        root,
		Objets:              v.Objects,
		Ancres:              ancres,
		CheminModuleCanevas: CheminCanevasForge(carte),
		Cle:                 carte.MapID,
	}
	idx, forge, err := indexForge(opts)
	if err != nil {
		t.Skipf("index Forge : %v", err)
	}
	var b BilanCuisson
	// `opts` porte deja Objets = v.Objects et laisse a zero les quatre reglages que l'appel
	// passait explicitement a nil/0 (types exclus, minceur, plafond, drapeaux) ; zJeu reste 0
	// comme avant.
	poseObjetsForge(context.Background(), r, &b, idx, forge, opts, 0)
	if b.ObjetsDessines == 0 {
		t.Fatal("aucun objet dessine")
	}

	t.Logf("VERDICT UTILISATEUR : %s", verdict)
	t.Logf("objets %d/%d · zJeu %.1f · cadre %dx%d px", b.ObjetsDessines, b.ObjetsForge, zJeu, r.NX, r.NY)
	mesureAncresQualite(t, s, ancres, r)
	mesureProfilEtResidu(t, r, s)
	mesureAncresToits(t, r, ancres)
}

// mesureAncresQualite chiffre la reference : nombre d'ancres, emprise, dispersion en z, et
// erreur d'interpolation leave-one-out (chaque ancre confrontee a la surface interpolee des
// autres).
func mesureAncresQualite(t *testing.T, s *SurfaceReference, ancres [][3]float64, r *Rendu) {
	t.Helper()
	lo := [2]float64{math.Inf(1), math.Inf(1)}
	hi := [2]float64{math.Inf(-1), math.Inf(-1)}
	var zs []float64
	for _, a := range ancres {
		lo[0], lo[1] = math.Min(lo[0], a[0]), math.Min(lo[1], a[1])
		hi[0], hi[1] = math.Max(hi[0], a[0]), math.Max(hi[1], a[1])
		zs = append(zs, a[2])
	}
	var loo []float64
	for i := range ancres {
		autres := make([][3]float64, 0, len(ancres)-1)
		autres = append(autres, ancres[:i]...)
		autres = append(autres, ancres[i+1:]...)
		sr := NewSurfaceReference(autres)
		if sr.Vide() {
			continue
		}
		loo = append(loo, math.Abs(sr.At(ancres[i][0], ancres[i][1])-(ancres[i][2]-AncrageDecalageSol)))
	}
	t.Logf("REFERENCE : %d ancres · emprise %.0fx%.0f m (cadre %.0fx%.0f m) · z [%.1f;%.1f] IQR %.1f · "+
		"erreur LOO mediane %.1f m · max %.1f m",
		len(ancres), hi[0]-lo[0], hi[1]-lo[1],
		float64(r.NX)*r.Cell, float64(r.NY)*r.Cell,
		Centile(zs, 0), Centile(zs, 1), Centile(zs, 0.75)-Centile(zs, 0.25),
		Centile(loo, 0.5), Centile(loo, 1))
}

// mesureProfilEtResidu mesure, par pixel de matiere :
//   - le profil vertical de la surface HAUTE contre la reference (le couvercle) ;
//   - le taux de couverture de production, entier / dans la portee / hors portee ;
//   - le residu APRES substitution simulee (dans la portee : zRef ; au-dela : zHaut) ;
//   - la rugosite des champs zHaut et substitue (gribouillage).
func mesureProfilEtResidu(t *testing.T, r *Rendu, s *SurfaceReference) {
	t.Helper()
	bandes := []float64{-2, 2, 6, 10, 15, 20}
	nBandes := make([]int, len(bandes)+1)
	nMat, nDedans, nDehors := 0, 0, 0
	cacheProd, cacheDedans, cacheDehors := 0, 0, 0
	couvercleDedans, couvercleDehors := 0, 0
	resid4, resid8, resid4Dedans, resid4Dehors := 0, 0, 0, 0
	sousSol4Dedans := 0
	dedans := make([]bool, r.NX*r.NY)
	final := make([]float64, r.NX*r.NY)
	for j := 0; j < r.NY; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		for i := 0; i < r.NX; i++ {
			k := j*r.NX + i
			final[k] = math.Inf(-1)
			zHaut, zRef, ref, ok := r.CandidatsReference(i, j)
			if !ok {
				continue
			}
			nMat++
			d := s.DistanceAncre(r.Min[0]+(float64(i)+0.5)*r.Cell, y)
			in := d <= PorteeAncre
			dedans[k] = in
			if in {
				nDedans++
			} else {
				nDehors++
			}
			dz := zHaut - ref
			bi := len(bandes)
			for x, b := range bandes {
				if dz < b {
					bi = x
					break
				}
			}
			nBandes[bi]++
			cache := zHaut-zRef >= EcartPlafondMin && math.Abs(zRef-ref) <= TolSolReference
			if cache {
				cacheProd++
				if in {
					cacheDedans++
				} else {
					cacheDehors++
				}
			}
			if dz >= 6 {
				if in {
					couvercleDedans++
				} else {
					couvercleDehors++
				}
			}
			zFin := zHaut
			if in {
				zFin = zRef
			}
			final[k] = zFin
			if zFin-ref >= 4 {
				resid4++
				if in {
					resid4Dedans++
				} else {
					resid4Dehors++
				}
			}
			if zFin-ref >= 8 {
				resid8++
			}
			if in && zRef-ref <= -4 {
				sousSol4Dedans++
			}
		}
	}
	pc := func(n, d int) float64 {
		if d == 0 {
			return math.NaN()
		}
		return 100 * float64(n) / float64(d)
	}
	t.Logf("MATIERE %d px · dans portee ancres %.1f %% · hors portee %.1f %%",
		nMat, pc(nDedans, nMat), pc(nDehors, nMat))
	t.Logf("PROFIL zHaut-ref : <-2m %.1f%% · -2..2 %.1f%% · 2..6 %.1f%% · 6..10 %.1f%% · "+
		"10..15 %.1f%% · 15..20 %.1f%% · >=20 %.1f%%",
		pc(nBandes[0], nMat), pc(nBandes[1], nMat), pc(nBandes[2], nMat), pc(nBandes[3], nMat),
		pc(nBandes[4], nMat), pc(nBandes[5], nMat), pc(nBandes[6], nMat))
	t.Logf("TAUX COUVERTURE (prod, cadre entier) %.1f %% [seuil %.1f] · dans portee %.1f %% · hors %.1f %%",
		pc(cacheProd, nMat), 100*SeuilCarteCouverte, pc(cacheDedans, nDedans), pc(cacheDehors, nDehors))
	t.Logf("COUVERCLE (zHaut-ref>=6m) : dans portee %.1f %% · hors portee %.1f %%",
		pc(couvercleDedans, nDedans), pc(couvercleDehors, nDehors))
	t.Logf("RESIDU apres substitution simulee : >=4m %.1f %% (dedans %.1f %% · dehors %.1f %%) · >=8m %.1f %% · "+
		"sous-sol<=-4m dedans %.1f %%",
		pc(resid4, nMat), pc(resid4Dedans, nDedans), pc(resid4Dehors, nDehors), pc(resid8, nMat),
		pc(sousSol4Dedans, nDedans))
	mesureRugosite(t, r, final, dedans)
}

// mesureRugosite compare la rugosite du champ HAUT et du champ SUBSTITUE, dans la portee des
// ancres : ecart absolu moyen entre voisins horizontaux et part de paires en rupture (>0,5 m,
// SeuilAreteMetres) — le proxy chiffre du « gribouillage ».
func mesureRugosite(t *testing.T, r *Rendu, final []float64, dedans []bool) {
	t.Helper()
	var sumH, sumF float64
	nH, nF, rupH, rupF := 0, 0, 0, 0
	for j := 0; j < r.NY; j++ {
		for i := 0; i+1 < r.NX; i++ {
			k := j*r.NX + i
			if !dedans[k] || !dedans[k+1] {
				continue
			}
			if !math.IsInf(r.z[k], -1) && !math.IsInf(r.z[k+1], -1) {
				d := math.Abs(r.z[k] - r.z[k+1])
				sumH += d
				nH++
				if d > SeuilAreteMetres {
					rupH++
				}
			}
			if !math.IsInf(final[k], -1) && !math.IsInf(final[k+1], -1) {
				d := math.Abs(final[k] - final[k+1])
				sumF += d
				nF++
				if d > SeuilAreteMetres {
					rupF++
				}
			}
		}
	}
	if nH == 0 || nF == 0 {
		t.Log("RUGOSITE : echantillon vide")
		return
	}
	t.Logf("RUGOSITE (portee ancres, paires horizontales) : champ HAUT %.2f m/paire · ruptures %.1f %% ; "+
		"champ SUBSTITUE %.2f m/paire · ruptures %.1f %%",
		sumH/float64(nH), 100*float64(rupH)/float64(nH),
		sumF/float64(nF), 100*float64(rupF)/float64(nF))
}
