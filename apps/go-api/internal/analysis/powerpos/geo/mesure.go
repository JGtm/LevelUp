package geo

// mesure.go — L'ORCHESTRATION : DES TRIANGLES AUX NOEUDS MESURES.
//
// Chaque phase est chronometree et son compte publie : une mesure dont on ne sait pas
// combien de noeuds, de rayons ni de secondes elle a coute n'est pas reproductible.
// Le score n'est PAS calcule ici (cf. Score / Selectionne) : le calibrage rejoue les poids
// sans recalculer les rayons.

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Entrees rassemble ce que le paquet recoit d'une carte.
type Entrees struct {
	Cadre     Cadre
	Triangles []Triangle
	// ZMin, ZMax : la tranche verticale voxelisee.
	ZMin, ZMax float64
	Ancres     [][3]float64
	Ressources []Ressource
	// DansLArene : facultatif, ecarte les sols hors du volume de survie de la carte.
	DansLArene func([3]float64) bool
}

// Bilan chiffre la mesure.
type Bilan struct {
	Voxels          int
	VoxelsOccupes   int
	TrianglesUtiles int
	Sol             BilanSol
	Cibles          int
	Rayons          int
	// Ressources placees / sans noeud, par nature.
	RessourcesPlacees   map[string]int
	RessourcesSansNoeud map[string]int
	Durees              map[string]time.Duration
}

// Resultat est la mesure d'une carte.
type Resultat struct {
	Graphe *Graphe
	Noeuds []NoeudMesure
	Vis    *Visibilite
	// Volume : la grille d'occlusion, gardee pour le diagnostic (cf. Coutures).
	Volume *Volume
}

// Mesure calcule H, V, E, M et les distances de deplacement de chaque noeud. R est deduit
// des distances par Score (il depend d'un poids).
func Mesure(ctx context.Context, e Entrees, p Parametres) (*Resultat, Bilan, error) {
	b := Bilan{Durees: map[string]time.Duration{}, RessourcesPlacees: map[string]int{}, RessourcesSansNoeud: map[string]int{}}
	if len(e.Triangles) == 0 || len(e.Ancres) == 0 {
		return nil, b, fmt.Errorf("geo: %d triangles, %d ancres — rien a mesurer", len(e.Triangles), len(e.Ancres))
	}
	vol, g, err := solEtVolume(e, p, &b)
	if err != nil {
		return nil, b, err
	}
	if err := ctx.Err(); err != nil {
		return nil, b, err
	}
	t := time.Now()
	cibles := ChoisitCibles(g, p.PasCiblesCellules)
	vis := CalculeVisibilite(g, vol, cibles, p)
	b.Cibles, b.Rayons, b.Durees["visibilite"] = len(cibles), vis.Rayons, time.Since(t)

	t = time.Now()
	hauteurs := AltitudesRelatives(g, p)
	b.Durees["hauteur"] = time.Since(t)

	t = time.Now()
	dArmeForte := distancesVers(g, e.Ressources, map[string]bool{NatureArmeForte: true, NaturePowerup: true}, p, &b)
	dArme := distancesVers(g, e.Ressources, map[string]bool{NatureArmeForte: true, NaturePowerup: true, NatureArme: true}, p, &b)
	dObjectif := distancesVers(g, e.Ressources, map[string]bool{NatureObjectif: true}, p, &b)
	b.Durees["ressources"] = time.Since(t)

	t = time.Now()
	dCouvert := DistancesAuCouvert(g, vis, p)
	b.Durees["couvert"] = time.Since(t)

	noeuds := make([]NoeudMesure, len(g.Noeuds))
	nbCibles := float64(len(cibles))
	for i, n := range g.Noeuds {
		nm := NoeudMesure{Noeud: n, DArmeForte: dArmeForte[i], DArme: dArme[i], DObjectif: dObjectif[i], DCouvert: dCouvert[i]}
		nm.NbVisibles = vis.NbVisibles(i)
		nm.Brut.H = hauteurs[i]
		if nbCibles > 1 {
			nm.Brut.V = float64(nm.NbVisibles) / (nbCibles - 1)
		}
		nm.Brut.E = float64(vis.Secteurs(g, i, p)) / float64(max(p.NbSecteurs, 1))
		nm.Brut.M = Proximite(dCouvert[i], p.PorteeCouvertM)
		noeuds[i] = nm
	}
	return &Resultat{Graphe: g, Noeuds: noeuds, Vis: vis, Volume: vol}, b, nil
}

// solEtVolume voxelise puis reconstruit le sol.
func solEtVolume(e Entrees, p Parametres, b *Bilan) (*Volume, *Graphe, error) {
	if !(e.ZMax > e.ZMin) || math.IsNaN(e.ZMin) || math.IsNaN(e.ZMax) {
		return nil, nil, fmt.Errorf("geo: tranche verticale invalide [%v, %v]", e.ZMin, e.ZMax)
	}
	t := time.Now()
	vol := NouveauVolume(e.Cadre, e.ZMin, e.ZMax, p.PasVoxelXYM, p.PasVoxelZM)
	b.TrianglesUtiles = vol.Voxelise(e.Triangles)
	b.Voxels, b.VoxelsOccupes = vol.NbVoxels(), vol.Occupes()
	b.Durees["voxelisation"] = time.Since(t)

	t = time.Now()
	// Les GERMES d'atteignabilite sont les ancres d'objectif ET les ressources : un socle
	// d'arme est pose la ou l'on va le chercher, il est du terrain joue par definition.
	germes := append([][3]float64(nil), e.Ancres...)
	for _, r := range e.Ressources {
		germes = append(germes, [3]float64{r.X, r.Y, r.Z})
	}
	g, bs := ConstruitSol(e.Cadre, vol, e.Triangles, germes, e.DansLArene, p)
	g, bs.SansRetour = sansRetourEcartes(g, e.Ressources, p)
	b.Sol, b.Durees["sol"] = bs, time.Since(t)
	if len(g.Noeuds) == 0 {
		return nil, nil, fmt.Errorf("geo: aucun noeud praticable (candidats %d, libres %d, ancres placees %d/%d)",
			bs.Candidats, bs.SolsLibres, bs.AncresPlacees, bs.AncresPlacees+bs.AncresSansNoeud)
	}
	return vol, g, nil
}

// sansRetourEcartes ecarte les noeuds d'ou AUCUN objectif n'est atteignable. Un lieu d'ou
// l'on ne peut rejoindre aucun objectif n'est pas joue : mesure du 2026-09-20, 74 % des
// noeuds de Streets et 66 % de Bazaar etaient des toits et des terrasses hors jeu, atteints
// par une chaine de sauts sur du decor, sans retour — ils ecrasaient la normalisation
// (V median 0,66 sur Bazaar : la moitie des noeuds voyaient tout, depuis dehors).
func sansRetourEcartes(g *Graphe, ressources []Ressource, p Parametres) (*Graphe, int) {
	sources, _ := PlaceRessources(g, ressources, map[string]bool{NatureObjectif: true}, p)
	if len(sources) == 0 {
		return g, 0
	}
	dist := DistancesVers(g, sources)
	garde := make([]bool, len(g.Noeuds))
	ecartes := 0
	for i, d := range dist {
		garde[i] = !math.IsInf(d, 1)
		if !garde[i] {
			ecartes++
		}
	}
	if ecartes == 0 {
		return g, 0
	}
	return g.Restreint(garde), ecartes
}

// distancesVers place les ressources d'un ensemble de natures et rend les distances.
func distancesVers(g *Graphe, ressources []Ressource, natures map[string]bool, p Parametres, b *Bilan) []float64 {
	sources, sans := PlaceRessources(g, ressources, natures, p)
	cle := ""
	for n := range natures {
		if cle == "" || n < cle {
			cle = n
		}
	}
	b.RessourcesPlacees[cle] += len(sources)
	b.RessourcesSansNoeud[cle] += sans
	return DistancesVers(g, sources)
}
