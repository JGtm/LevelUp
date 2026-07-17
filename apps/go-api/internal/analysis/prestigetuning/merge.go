package prestigetuning

// merge.go — agrégation multi-joueurs. Les comptes d'une même clé
// (source, métrique, fenêtre) — resp. (source) pour l'acceptance — provenant de
// plusieurs player DBs sont sommés. Fonctions pures.

type mwKey struct {
	source, metric, windowType, windowValue string
}

// MergeCounts somme les comptes de plusieurs player DBs par
// (source, métrique, fenêtre).
func MergeCounts(in []MetricWindowCount) []MetricWindowCount {
	acc := map[mwKey]*MetricWindowCount{}
	var order []mwKey
	for _, c := range in {
		k := mwKey{c.Source, c.Metric, c.WindowType, c.WindowValue}
		cur, ok := acc[k]
		if !ok {
			cp := c
			acc[k] = &cp
			order = append(order, k)
			continue
		}
		cur.Created += c.Created
		cur.Completed += c.Completed
		cur.Expired += c.Expired
		cur.Abandoned += c.Abandoned
	}
	out := make([]MetricWindowCount, 0, len(order))
	for _, k := range order {
		out = append(out, *acc[k])
	}
	return out
}

// MergeAcceptance somme created/rejected par source sur plusieurs player DBs et
// recalcule le taux d'acceptation.
func MergeAcceptance(in []SourceAcceptance) []SourceAcceptance {
	acc := map[string]*SourceAcceptance{}
	var order []string
	for _, a := range in {
		cur, ok := acc[a.Source]
		if !ok {
			acc[a.Source] = &SourceAcceptance{Source: a.Source, Created: a.Created, Rejected: a.Rejected}
			order = append(order, a.Source)
			continue
		}
		cur.Created += a.Created
		cur.Rejected += a.Rejected
	}
	out := make([]SourceAcceptance, 0, len(order))
	for _, s := range order {
		a := acc[s]
		a.AcceptanceRate = rate(a.Created, a.Created+a.Rejected)
		out = append(out, *a)
	}
	return out
}
