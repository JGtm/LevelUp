package relations

// Counts agrège les compteurs d'aperçu (distinct / allies / rivals / core)
// à partir de la liste des relations.
type Counts struct {
	DistinctPlayers int
	AlliesCount     int
	RivalsCount     int
	CoreCount       int
}

// ComputeCounts calcule les compteurs d'aperçu sur l'ensemble des relations.
func ComputeCounts(rels []RelationStats) Counts {
	c := Counts{DistinctPlayers: len(rels)}
	for _, s := range rels {
		if s.TeammateMatches > 0 {
			c.AlliesCount++
		}
		if s.EnemyMatches > 0 {
			c.RivalsCount++
		}
		if IsCore(s) {
			c.CoreCount++
		}
	}
	return c
}

// IsCore : "noyau dur" — total>=20 ET teammate>=3 ET enemy>=3. Exporté : source
// unique consommée aussi par le service (flag is_core du DTO) pour que le front
// n'ait pas à dupliquer les seuils.
func IsCore(s RelationStats) bool {
	return s.TotalMatches >= CoreMinTotalMatches &&
		s.TeammateMatches >= CoreMinTeammate &&
		s.EnemyMatches >= CoreMinEnemy
}

// TopRef : référence binôme/bête noire (sortie de SelectTopAlly/SelectTopNemesis).
type TopRef struct {
	Gamertag string
	WinRate  *float64
	Matches  int
}

// SelectTopAlly retourne le binôme : meilleur teammate_win_rate parmi les
// relations avec teammate_matches >= 8. nil si aucun candidat. Tiebreak :
// plus de matchs en allié, puis gamertag pour le déterminisme.
func SelectTopAlly(rels []RelationStats) *TopRef {
	var best *RelationStats
	for i := range rels {
		s := &rels[i]
		if s.TeammateMatches < TopAllyMinTeammateMatches || s.TeammateWinRate == nil {
			continue
		}
		if best == nil || allyBetter(s, best) {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	return &TopRef{Gamertag: best.Gamertag, WinRate: best.TeammateWinRate, Matches: best.TeammateMatches}
}

// allyBetter : a est un meilleur binôme que b (taux de victoire allié plus
// haut ; tiebreak plus de matchs, puis gamertag croissant).
func allyBetter(a, b *RelationStats) bool {
	if *a.TeammateWinRate != *b.TeammateWinRate {
		return *a.TeammateWinRate > *b.TeammateWinRate
	}
	if a.TeammateMatches != b.TeammateMatches {
		return a.TeammateMatches > b.TeammateMatches
	}
	return a.Gamertag < b.Gamertag
}

// SelectTopNemesis retourne la bête noire : pire enemy_win_rate parmi les
// relations avec enemy_matches >= 8. nil si aucun candidat. Tiebreak : plus de
// matchs en ennemi, puis pire duel_ratio, puis gamertag.
func SelectTopNemesis(rels []RelationStats) *TopRef {
	var worst *RelationStats
	for i := range rels {
		s := &rels[i]
		if s.EnemyMatches < TopNemesisMinEnemyMatches || s.EnemyWinRate == nil {
			continue
		}
		if worst == nil || nemesisWorse(s, worst) {
			worst = s
		}
	}
	if worst == nil {
		return nil
	}
	return &TopRef{Gamertag: worst.Gamertag, WinRate: worst.EnemyWinRate, Matches: worst.EnemyMatches}
}

// nemesisWorse : a est une pire bête noire que b (taux de victoire ennemi plus
// bas ; tiebreak plus de matchs, puis duel_ratio plus bas, puis gamertag).
func nemesisWorse(a, b *RelationStats) bool {
	if *a.EnemyWinRate != *b.EnemyWinRate {
		return *a.EnemyWinRate < *b.EnemyWinRate
	}
	if a.EnemyMatches != b.EnemyMatches {
		return a.EnemyMatches > b.EnemyMatches
	}
	ar := duelRatioOrInf(a)
	br := duelRatioOrInf(b)
	if ar != br {
		return ar < br
	}
	return a.Gamertag < b.Gamertag
}

// duelRatioOrInf : duel_ratio numérique pour le tri (nil → très grand, donc
// jamais choisi comme "pire").
func duelRatioOrInf(s *RelationStats) float64 {
	if s.DuelRatio == nil {
		return 1e18
	}
	return *s.DuelRatio
}
