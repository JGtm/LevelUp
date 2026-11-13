package main

// Le mode -plan-seul : ajoute le plan des murs aux sidecars Forge déjà publiés, sans recuire.

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// bilanPlanSeul compte ce qu'a fait ajoutePlans.
type bilanPlanSeul struct {
	Ajoutes  int
	SansBlob []string
	Echecs   int
}

// ajoutePlans pose, dans le sidecar publié de chaque carte Forge retenue, le plan tiré de son
// navmesh.blob. Le plan ne dépend que du maillage : l'image, le calage, les statistiques, les
// dégradations et la date de génération du sidecar restent ceux de la cuisson qui les a produits.
// Une carte sans blob ou sans sidecar est laissée intacte.
func (e *environnement) ajoutePlans(ctx context.Context) bilanPlanSeul {
	var b bilanPlanSeul
	for _, carte := range himap.CartesForge {
		if !e.retenue(carte.MapID) {
			continue
		}
		chemin := filepath.Join(e.sortieDir, carte.MapID+".json")
		switch err := e.ajoutePlan(ctx, carte, chemin); {
		case errors.Is(err, os.ErrNotExist):
			b.SansBlob = append(b.SansBlob, carte.Nom+" ("+carte.MapID+")")
			slog.InfoContext(ctx, "mapfond: pas de plan, blob ou sidecar absent", "carte", carte.Nom, "map_id", carte.MapID)
		case err != nil:
			b.Echecs++
			slog.ErrorContext(ctx, "mapfond: plan impossible", "err", err, "carte", carte.Nom, "map_id", carte.MapID)
		default:
			b.Ajoutes++
		}
	}
	return b
}

// ajoutePlan réécrit le sidecar `chemin` avec le plan du blob de `carte`. Rend une erreur qui
// satisfait os.ErrNotExist quand le blob ou le sidecar n'existe pas.
func (e *environnement) ajoutePlan(ctx context.Context, carte himap.CarteForge, chemin string) error {
	m, err := e.chargeNavmesh(carte)
	if err != nil {
		return err
	}
	s, err := mapfondsidecar.Load(chemin)
	if err != nil {
		return err
	}
	plan := planDeMaillage(ctx, carte.MapID, sourcePlanForge, m)
	if plan == nil {
		return errors.New("plan des murs impossible")
	}
	s.Plan = plan
	return mapfondsidecar.Write(chemin, *s)
}

// executePlanSeul lance ajoutePlans, journalise le bilan et rend le code de sortie.
func executePlanSeul(ctx context.Context, env *environnement) int {
	b := env.ajoutePlans(ctx)
	slog.InfoContext(ctx, "plan seul termine", "ajoutes", b.Ajoutes, "sansBlobOuSidecar", len(b.SansBlob), "echecs", b.Echecs)
	if b.Echecs > 0 {
		return 1
	}
	return 0
}
