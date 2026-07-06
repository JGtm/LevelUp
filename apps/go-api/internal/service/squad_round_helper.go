package service

import "math"

// round2 arrondit à 2 décimales. Réintroduit côté service-root (K3b) après le
// départ de teammates_service_kpis.go vers le sous-package teammates : le seul
// consommateur prod restant est timeseries_service_aggregations.go. Helper
// trivial (miroir de analysis.round2 / teammates.round2 — littéral math pur).
func round2(v float64) float64 { return math.Round(v*100) / 100 }
