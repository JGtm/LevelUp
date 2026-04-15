package analysis

// weapon_reconciliation.go — Réconciliation post-traitement API.
//
// Port de src/analysis/reconciliation.py.
// Applique les agrégats API pour résoudre les attributions à faible confidence.
// N'écrase JAMAIS weapon_id — écrit dans reconciled_as.

// ReconcileAPIAggregates compare film vs API et assigne reconciled_as pour les low/none.
func ReconcileAPIAggregates(
	attributions []KillAttribution,
	apiWeaponCounts map[uint64]int,
) []KillAttribution {
	filmCounts := countConfidentAttributions(attributions)
	surplus := computeSurplus(apiWeaponCounts, filmCounts)

	if len(surplus) == 0 {
		return attributions
	}

	results := make([]KillAttribution, len(attributions))
	copy(results, attributions)

	for i := range results {
		a := &results[i]
		if a.Confidence != "low" && a.Confidence != "none" {
			continue
		}
		if a.WeaponID != nil && SentinelIDs[*a.WeaponID] {
			continue
		}
		if a.AttributionPath == "none" {
			continue
		}

		bestWID := findBestSurplus(surplus)
		if bestWID == nil {
			break
		}

		a.ReconciledAs = bestWID
		surplus[*bestWID]--
		if surplus[*bestWID] <= 0 {
			delete(surplus, *bestWID)
		}
	}

	return results
}

// AssignSentinels mappe les sentinels (melee/grenade/vehicle) via reconciled_as.
func AssignSentinels(
	attributions []KillAttribution,
	sentinelMap map[string]uint64, // key = "{xuid}_{time_ms}"
) []KillAttribution {
	results := make([]KillAttribution, len(attributions))
	copy(results, attributions)

	for i := range results {
		a := &results[i]
		key := a.XUID + "_" + itoa(a.TimeMS)
		if sentinel, ok := sentinelMap[key]; ok && a.ReconciledAs == nil {
			a.ReconciledAs = &sentinel
		}
	}

	return results
}

// countConfidentAttributions compte les high/medium par weapon_id.
func countConfidentAttributions(attributions []KillAttribution) map[uint64]int {
	counts := make(map[uint64]int)
	for _, a := range attributions {
		if (a.Confidence == "high" || a.Confidence == "medium") && a.WeaponID != nil {
			counts[*a.WeaponID]++
		}
	}
	return counts
}

// computeSurplus calcule API - film_confident par arme.
func computeSurplus(apiCounts map[uint64]int, filmCounts map[uint64]int) map[uint64]int {
	surplus := make(map[uint64]int)
	for wid, apiN := range apiCounts {
		if SentinelIDs[wid] {
			continue
		}
		filmN := filmCounts[wid]
		diff := apiN - filmN
		if diff > 0 {
			surplus[wid] = diff
		}
	}
	return surplus
}

// findBestSurplus retourne le weapon_id avec le plus grand surplus.
func findBestSurplus(surplus map[uint64]int) *uint64 {
	if len(surplus) == 0 {
		return nil
	}
	var bestWID uint64
	bestCount := 0
	for wid, count := range surplus {
		if count > bestCount {
			bestWID = wid
			bestCount = count
		}
	}
	return &bestWID
}

// itoa convertit un int en string sans import supplémentaire de strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		i--
		n /= 10
	}
	if neg {
		buf[i] = '-'
		i--
	}
	return string(buf[i+1:])
}
