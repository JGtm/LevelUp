package handlers

// settings_replay_sound.go — validation des réglages « Sons du rejeu 2D ».
//
// Fichier séparé pour la même raison que settings_backup.go : settings.go dépasse déjà le
// seuil de 500 lignes, on n'y ajoute pas.

import (
	"net/http"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// validerPourcentagesSonsRejeu refuse une variation ou une distance hors de [0, 100].
//
// Les deux réglages sont des POURCENTAGES servis par un curseur : hors bornes, ils ne
// veulent plus rien dire — une fourchette du jeu multipliée par 3 n'est plus la variation
// du jeu. On refuse donc plutôt que de corriger en silence : un curseur qui affiche 100
// alors que le serveur a retenu autre chose est un mensonge d'interface.
func validerPourcentagesSonsRejeu(req *domain.UpdateSettingsRequest) error {
	champs := []struct {
		valeur *int
		code   string
		nom    string
	}{
		{req.ReplaySoundVariationPercent, "invalid_replay_sound_variation", "replay_sound_variation_percent"},
		{req.ReplaySoundDistancePercent, "invalid_replay_sound_distance", "replay_sound_distance_percent"},
	}
	for _, c := range champs {
		if c.valeur == nil {
			continue
		}
		if *c.valeur < 0 || *c.valeur > 100 {
			return humacore.NewError(http.StatusBadRequest, c.code,
				c.nom+" doit être compris entre 0 et 100.")
		}
	}
	return nil
}
