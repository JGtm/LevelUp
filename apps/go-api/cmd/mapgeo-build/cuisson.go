package main

// cuisson.go — UNE CARTE, DU MODULE AUX POSITIONS.
//
// Le cadre est la boite des ancres d'objectif elargie de MargeCadreM ; la tranche verticale
// est celle du fond de carte (himap.TrancheDeJeu autour du niveau de jeu), resserree par
// ProfondeurSousJeuM / HauteurSurJeuM : on ne voxelise pas dix metres de sous-sol.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/himap"
)

// Marges du cadre et de la tranche, en metres. Le cadre s'arrete la ou l'arene s'arrete
// (les ancres la couvrent) plus de quoi voir ses bords ; la tranche va du sous-sol au toit.
const (
	MargeCadreM        = 15.0
	ProfondeurSousJeuM = 8.0
	HauteurSurJeuM     = 14.0
)

// Cuite est le resultat complet d'une carte.
type Cuite struct {
	Cible     *Cible
	Triangles BilanTriangles
	Bilan     geo.Bilan
	Resultat  *geo.Resultat
	Positions []geo.Position
	// FrontiereAppliquee : la coquille de mort a borne les noeuds.
	FrontiereAppliquee bool
	NiveauDeJeu        float64
	DureeTotale        time.Duration
	// MemoireSysMo / MemoireTasMo : `runtime.MemStats` a la fin de la mesure.
	MemoireSysMo float64
	MemoireTasMo float64
}

// Cuit mesure une carte et selectionne ses positions.
func Cuit(ctx context.Context, c *Cible, p geo.Parametres, r geo.Reglage) (*Cuite, error) {
	debut := time.Now()
	cadre, zMin, zMax, zJeu, err := cadreDe(c)
	if err != nil {
		return nil, err
	}
	tris, bt, err := ExtraitTriangles(ctx, c, cadre, zMin, zMax)
	if err != nil {
		return nil, err
	}
	ressources := append([]geo.Ressource(nil), c.Socles...)
	for _, a := range c.Ancres {
		ressources = append(ressources, geo.Ressource{X: a[0], Y: a[1], Z: a[2], Nature: geo.NatureObjectif})
	}
	e := geo.Entrees{Cadre: cadre, Triangles: tris, ZMin: zMin, ZMax: zMax, Ancres: c.Ancres, Ressources: ressources}
	frontiere := frontiereDe(ctx, c, zJeu)
	if frontiere != nil {
		e.DansLArene = frontiere.ContientFrontiere
	}
	res, b, err := geo.Mesure(ctx, e, p)
	if err != nil {
		return nil, fmt.Errorf("mesure de %s : %w", c.Carte, err)
	}
	geo.Score(res.Noeuds, r)
	positions := geo.Selectionne(res.Graphe, res.Noeuds, r)
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	out := &Cuite{
		Cible: c, Triangles: bt, Bilan: b, Resultat: res, Positions: positions,
		FrontiereAppliquee: frontiere != nil, NiveauDeJeu: zJeu, DureeTotale: time.Since(debut),
		MemoireSysMo: float64(ms.Sys) / (1 << 20), MemoireTasMo: float64(ms.HeapAlloc) / (1 << 20),
	}
	slog.InfoContext(ctx, "mapgeo: carte cuite", "carte", c.Carte, "module", c.Module,
		"cellules", cadre.NbCellules(), "noeuds", len(res.Noeuds), "cibles", b.Cibles, "rayons", b.Rayons,
		"germes", fmt.Sprintf("%d/%d", b.Sol.AncresPlacees, b.Sol.AncresPlacees+b.Sol.AncresSansNoeud),
		"ancres", len(c.Ancres), "socles", len(c.Socles), "positions", len(positions),
		"composantes", b.Sol.Composantes, "tailles", b.Sol.TaillesComposantes, "sans_retour", b.Sol.SansRetour,
		"arcs_coupes", b.Sol.ArcsCoupes,
		"arcs_saut", b.Sol.ArcsSaut, "arcs_obstacle", b.Sol.ArcsObstacle,
		"frontiere", frontiere != nil, "duree", out.DureeTotale.Round(time.Millisecond),
		"mem_sys_mo", fmt.Sprintf("%.0f", out.MemoireSysMo), "durees", b.Durees)
	return out, nil
}

// cadreDe rend le cadre, la tranche verticale et le niveau de jeu d'une cible.
func cadreDe(c *Cible) (geo.Cadre, float64, float64, float64, error) {
	if len(c.Ancres) == 0 {
		return geo.Cadre{}, 0, 0, 0, errors.New("aucune ancre")
	}
	minX, minY, maxX, maxY := c.Ancres[0][0], c.Ancres[0][1], c.Ancres[0][0], c.Ancres[0][1]
	for _, a := range c.Ancres[1:] {
		minX, maxX = min(minX, a[0]), max(maxX, a[0])
		minY, maxY = min(minY, a[1]), max(maxY, a[1])
	}
	cadre, err := geo.NouveauCadre(tactical.GrilleParDefaut(), minX-MargeCadreM, minY-MargeCadreM, maxX+MargeCadreM, maxY+MargeCadreM)
	if err != nil {
		return geo.Cadre{}, 0, 0, 0, err
	}
	zJeu := himap.MedianeZ(c.Ancres) - himap.AncrageDecalageSol
	trMin, trMax := himap.TrancheDeJeu(zJeu)
	return cadre, max(trMin, zJeu-ProfondeurSousJeuM), min(trMax, zJeu+HauteurSurJeuM), zJeu, nil
}

// frontiereDe lit la coquille de mort de la carte et la rend si elle garde toutes les
// ancres — la regle du fond de carte. Sinon nil, et le journal dit pourquoi.
func frontiereDe(ctx context.Context, c *Cible, zJeu float64) *himap.Sddt {
	chemin, err := himap.ModuleVariante(c.CheminModuleCarte, "any")
	if err != nil {
		slog.WarnContext(ctx, "mapgeo: pas de module any — arene non bornee", "err", err, "carte", c.Carte)
		return nil
	}
	s, err := himap.LitSddt(chemin)
	if err != nil {
		slog.WarnContext(ctx, "mapgeo: sddt illisible — arene non bornee", "err", err, "carte", c.Carte)
		return nil
	}
	if !himap.FrontiereGardeLesAncres(s, c.Ancres, zJeu) {
		slog.WarnContext(ctx, "mapgeo: la coquille de mort exclut des ancres — non appliquee",
			"carte", c.Carte, "plans", len(s.Frontieres))
		return nil
	}
	return &s
}
