// Package observability — titled.go : dimension TITRE dans les collecteurs
// (MT-05 / PMT-10).
//
// La factorisation route par titre SANS casser la parité Halo : le titre par
// défaut (halo_infinite) ET le titre vide (helpers legacy, ctx background)
// collapsent vers "" → clé/bucket NU → octets `/debug/vars` + buckets erreurs
// + stats player byte-identiques au comportement pré-seam. Un 2e titre obtient
// sa propre dimension (`<title>.<name>`, `<title>|level|message`, etc.).
//
// `obsDefaultTitle` est une const UNIQUE (pas de comparaison littérale dispersée
// → archlint no_slug_comparison vert ; le titre transite par paramètre, jamais
// par un `slug == "..."`).
package observability

const obsDefaultTitle = "halo_infinite"

// obsEffectiveTitle collapse le titre par défaut / vide vers "" (parité Halo).
// Tous les collecteurs et le filtre des endpoints passent par ici → un seul
// endroit définit « le défaut == pas de dimension titre ».
func obsEffectiveTitle(title string) string {
	if title == "" || title == obsDefaultTitle {
		return ""
	}
	return title
}

// obsKey construit la clé expvar titre-aware : nom NU pour le titre effectif
// vide (Halo/legacy), "<title>.<name>" sinon.
func obsKey(title, name string) string {
	if et := obsEffectiveTitle(title); et != "" {
		return et + "." + name
	}
	return name
}

// ─── Variantes titrées des helpers expvar (MT-05) ───────────────────────────
//
// Les helpers historiques (IncCounter, AddInt, SetInt, RecordDurationMS,
// LoadCounter, LoadDurationStats) restent inchangés = titre effectif vide
// (clé nue). Les call-sites migrent progressivement vers ces variantes en
// passant ctxkeys.TitleSlug(ctx).

// IncCounterT incrémente de 1 le compteur (titre-aware).
func IncCounterT(title, name string) { AddIntT(title, name, 1) }

// AddIntT augmente le compteur de delta (titre-aware).
func AddIntT(title, name string, delta int64) { AddInt(obsKey(title, name), delta) }

// SetIntT fixe la valeur du compteur (titre-aware).
func SetIntT(title, name string, value int64) { SetInt(obsKey(title, name), value) }

// RecordDurationMST enregistre une durée ms (titre-aware).
func RecordDurationMST(title, name string, ms int64) { RecordDurationMS(obsKey(title, name), ms) }

// LoadCounterT lit le compteur (titre-aware).
func LoadCounterT(title, name string) int64 { return LoadCounter(obsKey(title, name)) }

// LoadDurationStatsT lit le snapshot de durées (titre-aware).
func LoadDurationStatsT(title, name string) (count, sumMS, avgMS, maxMS int64) {
	return LoadDurationStats(obsKey(title, name))
}

// ─── Accès filtrés par titre (pour les endpoints monitoring, PMT-10 PR-2) ────

// EffectiveTitle expose le titre effectif (défaut/vide → "") au package api,
// pour peupler le champ DTO `Title` (omitempty → absent pour Halo).
func EffectiveTitle(title string) string { return obsEffectiveTitle(title) }

// PlayerAPIStatsForTitle retourne les stats player du titre effectif `title`.
func PlayerAPIStatsForTitle(title string) []PlayerAPIStat {
	et := obsEffectiveTitle(title)
	all := PlayerAPIStats()
	out := make([]PlayerAPIStat, 0, len(all))
	for _, s := range all {
		if s.Title == et {
			out = append(out, s)
		}
	}
	return out
}
