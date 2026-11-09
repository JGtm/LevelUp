// Package procmem mesure la mémoire résidente du processus (lot L22, plafond mémoire du profil desktop).
//
// La mesure sert à l'affichage (« mémoire actuelle : N Mo ») et à l'expvar `levelup.process.rss_mb` : elle
// ne décide d'aucun arrêt ni d'aucune limite. Sous Windows, l'ensemble de travail du processus
// (GetProcessMemoryInfo) ; ailleurs, la mémoire que le runtime Go a obtenue du système (approximation, la
// mesure exacte n'étant demandée que sous Windows).
package procmem

import "expvar"

// RSSMB rend la mémoire résidente du processus, en Mo entiers (0 si la mesure est indisponible).
func RSSMB() int {
	b, ok := residentBytes()
	if !ok {
		return 0
	}
	return int(b >> 20)
}

func init() {
	// Lue à chaque lecture de /debug/vars : une jauge, jamais un échantillonneur en tâche de fond.
	expvar.Publish("levelup.process.rss_mb", expvar.Func(func() any { return RSSMB() }))
}
