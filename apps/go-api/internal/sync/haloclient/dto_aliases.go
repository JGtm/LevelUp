package haloclient

import "levelup/go-api/internal/domain"

// CareerRankData est l'alias historique (côté client Halo) du snapshot de rang
// carrière canonique. Extrait ici avec le client (K3e) ; sync le ré-exporte.
type CareerRankData = domain.CareerRankSnapshot
