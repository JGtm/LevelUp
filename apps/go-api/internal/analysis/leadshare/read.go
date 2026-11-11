package leadshare

// read.go — LA LECTURE DE LA MAINMISE : de la passe retenue d'un match à la part de l'équipe du
// joueur, et la moyenne de plusieurs matchs. Pur, comme le reste du paquet : le dépôt de lecture
// (platform/duckdb) ne fait que ranger les lignes de `match_lead_share_latest` dans [MatchRead].

// MatchRead est la passe MESURÉE retenue d'un match, telle que la base la porte.
type MatchRead struct {
	// MeasuredMS : durée de la partie jouée.
	MeasuredMS int64
	// LeadChanges : retournements du match.
	LeadChanges int
	// SilentCamps : camps du roster sans série de score (à zéro, jamais en tête).
	SilentCamps int
	// LeadMS : temps passé seul en tête, par camp publié.
	LeadMS map[int]int64
}

// PlayerShare est la Mainmise d'un match pour l'équipe d'un joueur. Share est une FRACTION 0..1
// (ADR 0006), jamais un pourcentage.
type PlayerShare struct {
	Share   float64
	Changes int
}

// ShareOfCamp rend la Mainmise de `camp` dans le match, ou faux quand elle n'est pas lisible :
// durée nulle, ou camp absent de la passe sans qu'un camp muet explique l'absence. Un camp muet
// (jamais marqué) ne figure pas dans les lignes de camp : sa part vaut zéro, et c'est une mesure.
func ShareOfCamp(m MatchRead, camp int) (PlayerShare, bool) {
	if m.MeasuredMS <= 0 {
		return PlayerShare{}, false
	}
	lead, ok := m.LeadMS[camp]
	if !ok {
		if m.SilentCamps <= 0 {
			return PlayerShare{}, false
		}
		lead = 0
	}
	share := float64(lead) / float64(m.MeasuredMS)
	return PlayerShare{Share: min(1, max(0, share)), Changes: m.LeadChanges}, true
}

// Average rend la moyenne des parts MESURÉES et leur nombre. Un match non mesuré n'entre ni au
// numérateur ni au dénominateur : l'appelant ne passe que des parts lues. n == 0 : pas de moyenne.
func Average(shares []float64) (avg float64, n int) {
	if len(shares) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, s := range shares {
		sum += s
	}
	return sum / float64(len(shares)), len(shares)
}
