package main

// Un fond dont le sidecar publié porte un plan des murs a un maillage de navigation : le plan en est
// tiré. Une cuisson qui ne le trouve pas (blob absent du dépôt copié sur le poste, ou illisible) ne
// décrit pas la carte : publier alors un relief « sans maillage », ou un fond sans plan, ferait perdre
// à une carte ce qu'elle a. La carte est mise en échec nommé et rien n'est écrit pour elle. Les cartes
// qui n'ont vraiment pas de maillage (les deux Thunderhead) n'ont pas de plan publié : elles passent.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// errPlanPublieSansMaillage est l'échec d'une carte dont le fond publié a un plan et dont la cuisson n'a
// pas de maillage.
var errPlanPublieSansMaillage = errors.New("le fond publie porte un plan des murs mais la cuisson n'a trouve aucun maillage de navigation (blob absent ou illisible)")

// verifieLeMaillageDuPlanPublie rend errPlanPublieSansMaillage quand le sidecar publié `s` porte un plan
// (ou son image) et que la cuisson n'a pas de maillage `m`.
func verifieLeMaillageDuPlanPublie(cle string, s *mapfondsidecar.MapBackground, m *hinavmesh.Maillage) error {
	if m != nil || (s.Plan == nil && s.PlanImage == "") {
		return nil
	}
	return fmt.Errorf("carte %s : %w ; rien n'est ecrit", cle, errPlanPublieSansMaillage)
}

// verifieLeMaillageAvantPublication est la garde de la cuisson complète : le sidecar déjà présent dans
// le dossier de sortie, quand il y en a un, est celui que la cuisson va remplacer. Un sidecar absent ou
// illisible ne porte pas de plan à protéger ; l'illisible est journalisé.
func (e *environnement) verifieLeMaillageAvantPublication(ctx context.Context, cle string, m *hinavmesh.Maillage) error {
	if m != nil {
		return nil
	}
	s, err := mapfondsidecar.Load(filepath.Join(e.sortieDir, cle+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		slog.WarnContext(ctx, "mapfond: sidecar existant illisible, garde du maillage non appliquee", "err", err, "carte", cle)
		return nil
	}
	return verifieLeMaillageDuPlanPublie(cle, s, m)
}
