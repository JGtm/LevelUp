package service

// replay_map_background_plan.go — L'IMAGE PLAN D'UN FOND DE CARTE.
//
// L'image plan (`<cle>.plan.webp`) est le plan des composantes du maillage de navigation que la
// cuisson garde, dessiné dans le cadre et au calage de l'image du fond : les deux se superposent
// pixel pour pixel. Son nom vit dans le sidecar (`planImage`), à côté de celui de l'image du fond ;
// elle se sert comme elle, sous la même clé, la même cascade de résolution et les mêmes gardes sur
// le nom de fichier. Une carte sans plan n'a pas de `planImage` : même absence que sans fond.

import (
	"context"

	"levelup/go-api/internal/port"
)

// MapBackgroundPlanImage retourne les octets de l'image plan du fond de la carte du match, et son
// type MIME. Retourne ErrMapBackgroundNotAvailable quand la carte n'a pas de fond, ou pas de plan.
func (s *replayService) MapBackgroundPlanImage(ctx context.Context, matchID string) ([]byte, string, error) {
	key, err := s.resolveBackgroundKey(ctx, matchID)
	if err != nil {
		return nil, "", err
	}
	return s.readPlanImage(ctx, key)
}

// MapBackgroundPlanImageForMap est la jumelle de MapBackgroundPlanImage keyée par CARTE (map_id).
func (s *replayService) MapBackgroundPlanImageForMap(ctx context.Context, mapID string) ([]byte, string, error) {
	key, err := s.resolveBackgroundKeyForMap(ctx, mapID)
	if err != nil {
		return nil, "", err
	}
	return s.readPlanImage(ctx, key)
}

// readPlanImage lit l'image plan d'une clé de fond. Le sidecar doit être lisible : une image sans
// calage ne se superpose à rien.
func (s *replayService) readPlanImage(ctx context.Context, key string) ([]byte, string, error) {
	full, err := s.loadFullSidecar(ctx, key)
	if err != nil {
		return nil, "", err
	}
	if full.PlanImage == "" {
		return nil, "", port.ErrMapBackgroundNotAvailable
	}
	chemin, mime, err := s.imageFileNamed(ctx, key, full.PlanImage)
	if err != nil {
		return nil, "", err
	}
	return lisFichierImage(chemin, mime, key)
}
