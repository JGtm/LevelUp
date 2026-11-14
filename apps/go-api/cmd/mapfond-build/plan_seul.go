package main

// Le mode -plan-seul : refait le plan des murs et l'image plan des sidecars déjà publiés, sans
// recuire le fond.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// bilanPlanSeul compte ce qu'a fait ajoutePlans et garde une ligne de rapport par carte traitée.
type bilanPlanSeul struct {
	Ajoutes  int
	SansBlob []string
	Echecs   int
	Bilans   []bilanAsset
}

// carteDuPlan est une carte dont on refait le plan : son maillage, ses ancres, l'origine du
// maillage pour `plan.source`.
type carteDuPlan struct {
	cle      string
	noms     []string
	source   string
	ancres   [][3]float64
	maillage *hinavmesh.Maillage
}

// ajoutePlans refait, dans le sidecar publié de chaque carte retenue, le plan tiré de son
// maillage de navigation (cartes Forge : `navmesh.blob` ; cartes natives : module `any`), et
// l'image plan à côté du fond. Le plan ne dépend que du maillage, des ancres et de la matière du
// fond publié : l'image du fond, le calage, les statistiques, les dégradations et la date de
// génération du sidecar restent ceux de la cuisson qui les a produits. Une carte sans maillage ou
// sans sidecar est laissée intacte. L'avis (`-avis`) ne décide pas ici : le fond n'est pas touché.
func (e *environnement) ajoutePlans(ctx context.Context) bilanPlanSeul {
	var b bilanPlanSeul
	if !e.sansForge {
		for _, carte := range himap.CartesForge {
			if e.retenue(carte.MapID) && !e.ignoree(ctx, carte.MapID, []string{carte.Nom}, &b) {
				e.ajoutePlanForge(ctx, carte, &b)
			}
		}
	}
	if !e.sansNatives {
		for _, c := range e.ciblesNatives() {
			if !e.ignoree(ctx, c.cle, c.noms, &b) {
				e.ajoutePlanNatif(ctx, c, &b)
			}
		}
	}
	return b
}

// ignoree dit si l'avis demande de laisser la carte entièrement telle qu'elle est (ni fond ni
// plan) et, dans ce cas, la note au rapport.
func (e *environnement) ignoree(ctx context.Context, cle string, noms []string, b *bilanPlanSeul) bool {
	d := e.decisionFond(cle)
	if !d.Ignore {
		return false
	}
	slog.InfoContext(ctx, "mapfond: carte ignoree par l'avis, rien n'est recalcule", "carte", cle)
	b.Bilans = append(b.Bilans, bilanSaute(cle, noms, d))
	return true
}

// ajoutePlanForge refait le plan d'une carte Forge.
func (e *environnement) ajoutePlanForge(ctx context.Context, carte himap.CarteForge, b *bilanPlanSeul) {
	m, err := e.chargeNavmesh(carte)
	if err == nil {
		var entree = carteDuPlan{cle: carte.MapID, noms: []string{carte.Nom}, source: sourcePlanForge, maillage: m}
		entree.ancres, err = e.ancresDuCatalogue(carte.MapID)
		if err == nil {
			err = e.refaitLePlan(ctx, entree, b)
		}
	}
	e.noteResultat(ctx, b, carte.MapID, []string{carte.Nom}, err)
}

// ajoutePlanNatif refait le plan d'une carte native.
func (e *environnement) ajoutePlanNatif(ctx context.Context, c cible, b *bilanPlanSeul) {
	m, _, err := e.maillageNatif(ctx, c.cle)
	if err == nil && m == nil {
		err = fmt.Errorf("aucun maillage de navigation : %w", os.ErrNotExist)
	}
	if err == nil {
		err = e.refaitLePlan(ctx, carteDuPlan{cle: c.cle, noms: c.noms, source: sourcePlanNatif, ancres: c.ancres, maillage: m}, b)
	}
	e.noteResultat(ctx, b, c.cle, c.noms, err)
}

// ancresDuCatalogue rend les ancres d'objectifs d'une carte, lues au catalogue d'objectifs.
func (e *environnement) ancresDuCatalogue(cle string) ([][3]float64, error) {
	entree, err := e.catalogue.Lookup(cle)
	if err != nil {
		return nil, err
	}
	out := make([][3]float64, 0, len(entree.Objectives))
	for _, o := range entree.Objectives {
		out = append(out, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	return out, nil
}

// noteResultat classe le résultat d'un plan dans le bilan : absent (maillage ou sidecar), échec,
// ou ajouté.
func (e *environnement) noteResultat(ctx context.Context, b *bilanPlanSeul, cle string, noms []string, err error) {
	switch {
	case errors.Is(err, os.ErrNotExist):
		b.SansBlob = append(b.SansBlob, cle)
		slog.InfoContext(ctx, "mapfond: pas de plan, maillage ou sidecar absent", "carte", cle, "err", err)
	case err != nil:
		b.Echecs++
		b.Bilans = append(b.Bilans, bilanAsset{Cle: cle, Noms: noms, Err: err})
		slog.ErrorContext(ctx, "mapfond: plan impossible", "err", err, "carte", cle)
	default:
		b.Ajoutes++
	}
}

// refaitLePlan réécrit le sidecar de la carte avec son plan et écrit l'image plan. Rend une erreur
// qui satisfait os.ErrNotExist quand le sidecar ou le fond publié n'existe pas.
func (e *environnement) refaitLePlan(ctx context.Context, c carteDuPlan, b *bilanPlanSeul) error {
	chemin := filepath.Join(e.sortieDir, c.cle+".json")
	s, err := mapfondsidecar.Load(chemin)
	if err != nil {
		return err
	}
	fond, err := lisFondPublie(e.sortieDir, s)
	if err != nil {
		return err
	}
	cal := himap.CalageFond{
		OrigineX: s.Calibration.OriginX, OrigineY: s.Calibration.OriginY,
		MetresParPixel: s.Calibration.MetersPerPixel, Largeur: s.Calibration.WidthPx, Hauteur: s.Calibration.HeightPx,
	}
	if fond.Bounds().Dx() != cal.Largeur || fond.Bounds().Dy() != cal.Hauteur {
		return fmt.Errorf("fond %s de %dx%d px, calage de %dx%d px", s.Image,
			fond.Bounds().Dx(), fond.Bounds().Dy(), cal.Largeur, cal.Hauteur)
	}
	pf, err := calculePlan(c.source, c.maillage, c.ancres, cal, matiereDeImage(fond, cal))
	if err != nil {
		return err
	}
	if pf == nil {
		return errors.New("plan des murs impossible")
	}
	poseLePlan(ctx, e.sortieDir, c.cle, pf, s)
	if s.Plan == nil {
		return errors.New("image plan impossible")
	}
	suivi := suiviFond{Decision: e.decisionFond(c.cle), Action: actionPlanSeul, Plan: pf}
	if e.rognePublie {
		if err := e.rogneLePublie(ctx, c, s, fond, cal, &suivi); err != nil {
			return err
		}
	}
	if err := mapfondsidecar.Write(chemin, *s); err != nil {
		return err
	}
	b.Bilans = append(b.Bilans, bilanAsset{Cle: c.cle, Noms: c.noms, Suivi: suivi})
	return nil
}

// executePlanSeul lance ajoutePlans, journalise le bilan, écrit les rapports demandés et rend le
// code de sortie.
func executePlanSeul(ctx context.Context, env *environnement, rapports rapportsDemandes) int {
	b := env.ajoutePlans(ctx)
	slog.InfoContext(ctx, "plan seul termine", "ajoutes", b.Ajoutes, "sansMaillageOuSidecar", len(b.SansBlob), "echecs", b.Echecs)
	echecs := b.Echecs
	if rapports.tsv != "" {
		if err := ecritRapportTSV(rapports.tsv, rapports.avantDir, b.Bilans, env); err != nil {
			slog.ErrorContext(ctx, "rapport tsv", "err", err, "path", rapports.tsv)
			echecs++
		}
	}
	if echecs > 0 {
		return 1
	}
	return 0
}
