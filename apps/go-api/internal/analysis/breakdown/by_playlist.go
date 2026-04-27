package breakdown

import "sort"

// PlaylistAggregate agrege par playlist.
type PlaylistAggregate struct {
	PlaylistName string
	Counts
	AvgPerformanceScore *float64
}

// ByPlaylist groupe par PlaylistName.
// Les rows avec PlaylistName vide sont ignorees.
// Tri : WinRate desc puis PlaylistName asc.
func ByPlaylist(rows []Row) []PlaylistAggregate {
	grouped := make(map[string][]Row)
	for _, r := range rows {
		if r.PlaylistName == "" {
			continue
		}
		grouped[r.PlaylistName] = append(grouped[r.PlaylistName], r)
	}
	out := make([]PlaylistAggregate, 0, len(grouped))
	for k, items := range grouped {
		out = append(out, PlaylistAggregate{
			PlaylistName:        k,
			Counts:              computeCounts(items),
			AvgPerformanceScore: avgPerformanceScore(items),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].WinRate != out[j].WinRate {
			return out[i].WinRate > out[j].WinRate
		}
		return out[i].PlaylistName < out[j].PlaylistName
	})
	return out
}
