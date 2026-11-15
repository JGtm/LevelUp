package coalesce

// IsolateTracking donne au process un suivi des recalculs en vol neuf et rend de quoi rétablir
// l'ancien. Drain refuse définitivement toute nouvelle demande du suivi qu'il draine : un test
// qui exerce l'arrêt d'un paquet appelant (écriture suivie d'un service) isole le suivi pour ne
// pas fermer celui des autres tests du binaire. Réservé aux tests ; jamais appelé hors d'eux.
func IsolateTracking() (restore func()) {
	previous := inflight
	inflight = &inflightRuns{}
	return func() { inflight = previous }
}
