// Package relations — daypart.go : bucketing horaire en 6 tranches (day-parts)
// pour le heatmap agrégé « Quand tu les croises ». Pur, sans DB. Les bornes
// sont des constantes nommées (pas de magic number).
package relations

// Daypart : tranche horaire de la journée (heatmap relation × tranche).
type Daypart int

// Tranches (ordre d'affichage croissant : Nuit → Tard). 6 buckets larges pour
// garder le heatmap lisible (réserve produit : ne pas surcharger).
const (
	DaypartNight     Daypart = iota // Nuit       : 00h–05h
	DaypartMorning                  // Matin      : 06h–10h
	DaypartNoon                     // Midi       : 11h–13h
	DaypartAfternoon                // Après-midi : 14h–17h
	DaypartEvening                  // Soir       : 18h–21h
	DaypartLateNight                // Tard       : 22h–23h
	DaypartCount
)

// DaypartFromHour mappe une heure 0..23 vers sa tranche. Les bornes :
//
//	00–05 Nuit · 06–10 Matin · 11–13 Midi · 14–17 Après-midi · 18–21 Soir · 22–23 Tard
func DaypartFromHour(hour int) Daypart {
	switch {
	case hour < 6:
		return DaypartNight
	case hour < 11:
		return DaypartMorning
	case hour < 14:
		return DaypartNoon
	case hour < 18:
		return DaypartAfternoon
	case hour < 22:
		return DaypartEvening
	default:
		return DaypartLateNight
	}
}
