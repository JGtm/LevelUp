package squadagg

// Types d'expérience canoniques (labels FR) partagés par les helpers d'agrégation
// d'escouade. Dupliqués depuis service/match_history_service.go (2 copies tolérées :
// packages disjoints après extraction K3b ; ce sont des littéraux de vocabulaire,
// pas de la logique).
const (
	ExpTypePVPRanked   = "PVP classé"
	ExpTypePVPUnranked = "PVP non classé"
	ExpTypePVE         = "PVE"
)
