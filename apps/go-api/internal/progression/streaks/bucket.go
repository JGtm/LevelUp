package streaks

import "time"

// bucket.go — primitives de date/semaine pour le découpage en buckets streak.
//
// Toutes les opérations sont en UTC pour cohérence cross-fuseaux. L'utilisateur
// final voit ses streaks dans son fuseau local (rendu côté frontend), mais la
// vérité métier reste en UTC pour éviter les ambiguïtés de bucket à minuit
// dans des timezones différentes.

// DayStart retourne minuit (00:00:00 UTC) du jour calendaire contenant t.
func DayStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// DayEnd retourne 23:59:59.999999999 UTC du jour calendaire contenant t.
func DayEnd(t time.Time) time.Time {
	return DayStart(t).Add(24*time.Hour - time.Nanosecond)
}

// DaysBetween retourne le nombre de jours calendaires entre a et b (a < b).
// 0 si même jour, 1 si jours consécutifs, etc. Signé : négatif si a > b.
func DaysBetween(a, b time.Time) int {
	da := DayStart(a)
	db := DayStart(b)
	return int(db.Sub(da).Hours() / 24)
}

// WeekStart retourne le lundi 00:00 UTC de la semaine ISO contenant t.
// Convention ISO : la semaine commence le lundi.
func WeekStart(t time.Time) time.Time {
	d := DayStart(t)
	// time.Weekday : Sunday=0, Monday=1, ..., Saturday=6. ISO veut lundi en première position.
	wd := int(d.Weekday())
	if wd == 0 {
		wd = 7 // dimanche traité comme jour 7 ISO
	}
	return d.AddDate(0, 0, -(wd - 1))
}

// WeekEnd retourne dimanche 23:59:59.999999999 UTC de la semaine ISO contenant t.
func WeekEnd(t time.Time) time.Time {
	return WeekStart(t).AddDate(0, 0, 7).Add(-time.Nanosecond)
}

// WeeksBetween retourne le nombre de semaines ISO entre a et b (a < b).
// 0 si même semaine, 1 si semaines consécutives. Signé : négatif si a > b.
func WeeksBetween(a, b time.Time) int {
	wa := WeekStart(a)
	wb := WeekStart(b)
	return int(wb.Sub(wa).Hours() / (24 * 7))
}

// SameMonth retourne true si a et b sont dans le même mois calendaire UTC.
// Utilisé pour la régénération mensuelle des shields.
func SameMonth(a, b time.Time) bool {
	a, b = a.UTC(), b.UTC()
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// BucketStart retourne le début de bucket (jour ou semaine) selon le type de streak.
func BucketStart(t time.Time, st StreakType) time.Time {
	if isWeeklyType(st) {
		return WeekStart(t)
	}
	return DayStart(t)
}

// BucketEnd retourne la fin du bucket (jour ou semaine) selon le type.
func BucketEnd(t time.Time, st StreakType) time.Time {
	if isWeeklyType(st) {
		return WeekEnd(t)
	}
	return DayEnd(t)
}

// BucketsBetween retourne le nombre de buckets (jours ou semaines) entre a et b
// selon le type de streak.
func BucketsBetween(a, b time.Time, st StreakType) int {
	if isWeeklyType(st) {
		return WeeksBetween(a, b)
	}
	return DaysBetween(a, b)
}

// isWeeklyType retourne true pour les types streak à granularité hebdomadaire.
func isWeeklyType(st StreakType) bool {
	return st == StreakTypeWeeklyPlay || st == StreakTypeWeeklyKDAThreshold
}
