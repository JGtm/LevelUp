package main

// triangles.go — DES `.module` DU JEU AUX TRIANGLES MONDE.
//
// C'est la boucle de `himap.PeupleRendu`, avec UNE difference assumee : elle ne rasterise
// pas, elle RESTITUE les triangles. Un fond de carte n'a besoin que du sommet de chaque
// colonne ; une analyse d'occlusion a besoin des murs, des plafonds et des dessous.
//
// DEUX ECARTS AU PRODUCTEUR DE FONDS, tous deux mesures et journalises :
//
//  1. LE FILTRE DE DECOR N'EST PAS APPLIQUE. `EstDecorGrossier` ecarte les maillages dont
//     le triangle median est trop grand — une regle d'IMAGE (une dalle de terrain grossiere
//     salit un plan). Pour l'occlusion, une dalle grossiere ARRETE quand meme les balles.
//     Les deux comptes sont rendus : on sait combien de matiere le fond jette.
//  2. LE BORNAGE A LA BOITE DE L'INSTANCE ECARTE LE TRIANGLE ENTIER au lieu de le rogner.
//     Meme intention que `AddMeshBorne` (quelques instances des modules globaux debordent
//     d'un facteur 42,8 de leur diagonale), mais a la maille du triangle : un triangle dont
//     le centre sort de la boite dilatee n'est pas de cette instance.
//
// C'est la DEUXIEME copie de cette boucle dans le depot (la premiere est `PeupleRendu`).
// A la troisieme, la regle du depot impose de la centraliser dans `himap` avec un
// garde-rail — elle n'est pas centralisee ici parce que le banc de non-regression du fond
// de carte (`TestBancCliffhanger`, tag `gamefiles`) ne peut pas etre rejoue dans cette
// session, et qu'un refactor non verifie de la production de fonds coute plus qu'il ne rend.

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/himap"
)

// BilanTriangles chiffre une extraction. Publie : une geometrie dont on ne sait pas
// combien d'instances ont ete lues n'est pas verifiable.
type BilanTriangles struct {
	BSPs          int
	BSPInstances  int
	Lues          int
	SansGeometrie int
	// DecorGrossier : instances que le producteur de FONDS aurait ecartees. Comptees, pas
	// ecartees — cf. l'en-tete.
	DecorGrossier int
	Triangles     int
	HorsBoite     int
	HorsCadre     int
	// MinZ, MaxZ : l'emprise verticale des triangles retenus.
	MinZ, MaxZ float64
}

// ExtraitTriangles rend les triangles MONDE d'une carte, bornes au cadre `cadre` (monde) et
// a la tranche verticale [zMin, zMax].
//
// Le cadre borne le TRAVAIL, pas la verite : un triangle hors cadre ne peut occulter que
// des rayons hors cadre, et les noeuds sont tous dedans par construction.
func ExtraitTriangles(ctx context.Context, c *Cible, cadre geo.Cadre, zMin, zMax float64) ([]geo.Triangle, BilanTriangles, error) {
	var b BilanTriangles
	racine, err := himap.DeployRoot()
	if err != nil {
		return nil, b, err
	}
	chemins, err := himap.GeometrySearchPath(racine, c.CheminModule)
	if err != nil {
		return nil, b, fmt.Errorf("chemins de geometrie : %w", err)
	}
	idx, err := himap.NewModuleIndex(chemins...)
	if err != nil {
		return nil, b, fmt.Errorf("index des modules : %w", err)
	}
	defer func() {
		if errFerme := idx.Close(); errFerme != nil {
			slog.ErrorContext(ctx, "mapgeo: fermeture de l'index des modules", "err", errFerme)
		}
	}()
	bsps, err := himap.ReadModuleInstances(c.CheminModule)
	if err != nil {
		return nil, b, fmt.Errorf("instances du module : %w", err)
	}
	bsp := himap.ChoisitBSP(bsps, c.Ancres)
	b.BSPs, b.BSPInstances = len(bsps), len(bsp.Instances)

	tris := collecte(ctx, idx, bsp, cadre, zMin, zMax, &b)
	if len(tris) == 0 {
		return nil, b, fmt.Errorf("aucun triangle dans le cadre sur %d instances", len(bsp.Instances))
	}
	slog.InfoContext(ctx, "mapgeo: triangles extraits", "carte", c.Carte, "module", c.Module,
		"bsps", b.BSPs, "instances", b.BSPInstances, "lues", b.Lues, "decor_grossier", b.DecorGrossier,
		"triangles", b.Triangles, "hors_cadre", b.HorsCadre, "hors_boite", b.HorsBoite,
		"z", fmt.Sprintf("%.2f..%.2f", b.MinZ, b.MaxZ))
	return tris, b, nil
}

// collecte parcourt les instances du bsp et accumule les triangles retenus.
func collecte(ctx context.Context, idx *himap.ModuleIndex, bsp himap.BSPInstances,
	cadre geo.Cadre, zMin, zMax float64, b *BilanTriangles) []geo.Triangle {
	assets := map[uint32]*himap.RuntimeGeoAsset{}
	var tris []geo.Triangle
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != himap.GroupeRtgo {
			continue
		}
		a, deja := assets[id]
		if !deja {
			a = ouvreAsset(ctx, idx, id)
			assets[id] = a
		}
		if a == nil {
			continue
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil {
			b.SansGeometrie++
			continue
		}
		if himap.EstDecorGrossier(m, in, himap.AireMaxTriangleJouable) {
			b.DecorGrossier++
		}
		b.Lues++
		tris = ajouteMaillage(tris, m, in, cadre, zMin, zMax, b)
	}
	return tris
}

// ajouteMaillage transforme un maillage en monde et retient ses triangles utiles.
func ajouteMaillage(tris []geo.Triangle, m *himap.Mesh, in himap.Instance,
	cadre geo.Cadre, zMin, zMax float64, b *BilanTriangles) []geo.Triangle {
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	marge := himap.MargeBornageInstance
	lo := [3]float64{in.AABBMin[0] - marge, in.AABBMin[1] - marge, in.AABBMin[2] - marge}
	hi := [3]float64{in.AABBMax[0] + marge, in.AABBMax[1] + marge, in.AABBMax[2] + marge}
	for _, t := range m.Triangles {
		tri := geo.Triangle{monde[t[0]], monde[t[1]], monde[t[2]]}
		if !dansLaBoite(tri, lo, hi) {
			b.HorsBoite++
			continue
		}
		if !cadre.ToucheTriangle(tri, zMin, zMax) {
			b.HorsCadre++
			continue
		}
		if b.Triangles == 0 {
			b.MinZ, b.MaxZ = tri[0][2], tri[0][2]
		}
		for _, p := range tri {
			b.MinZ, b.MaxZ = min(b.MinZ, p[2]), max(b.MaxZ, p[2])
		}
		b.Triangles++
		tris = append(tris, tri)
	}
	return tris
}

// dansLaBoite dit si le CENTRE du triangle tombe dans la boite monde de son instance.
func dansLaBoite(t geo.Triangle, lo, hi [3]float64) bool {
	for a := 0; a < 3; a++ {
		c := (t[0][a] + t[1][a] + t[2][a]) / 3
		if c < lo[a] || c > hi[a] {
			return false
		}
	}
	return true
}

// ouvreAsset extrait et decode un tag `rtgo`. Un tag illisible est LOGGE puis saute : une
// carte ne doit pas mourir sur un maillage, mais une extraction muette masquerait un trou.
func ouvreAsset(ctx context.Context, idx *himap.ModuleIndex, id uint32) *himap.RuntimeGeoAsset {
	tag, blob, err := idx.ExtractWithResources(id)
	if err != nil {
		slog.DebugContext(ctx, "mapgeo: tag de geometrie illisible", "id", fmt.Sprintf("%08x", id), "err", err)
		return nil
	}
	a, err := himap.NewRuntimeGeoAsset(tag, blob)
	if err != nil {
		slog.DebugContext(ctx, "mapgeo: tag de geometrie indecodable", "id", fmt.Sprintf("%08x", id), "err", err)
		return nil
	}
	return a
}
