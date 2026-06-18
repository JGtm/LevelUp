// labels_resolver.go — resolver de libellés Discord PARTAGÉ (PMT-11). Câblé une
// fois au boot (server.go) avec une closure qui construit les NotifyLabels d'un
// titre depuis son adapter sémantique (outcomes.toml) + son nom de registre. Les
// call-sites (settings, orchestrator, CLIs) injectent `cfg.Labels = LabelsForSlug
// (slug)` sans connaître les packages games/title — la closure encapsule tout.
//
// Nil tant que non câblé (tests/CLI) → LabelsForSlug retombe sur HaloLabels
// (byte-identique Halo).
package notify

import "sync"

var (
	defaultLabelsMu       sync.RWMutex
	defaultLabelsResolver func(slug string) NotifyLabels
)

// SetDefaultLabelsResolver câble le resolver de libellés partagé (appelé au boot).
func SetDefaultLabelsResolver(fn func(slug string) NotifyLabels) {
	defaultLabelsMu.Lock()
	defaultLabelsResolver = fn
	defaultLabelsMu.Unlock()
}

// LabelsForSlug retourne les libellés title-aware d'un titre via le resolver
// partagé. Failsafe : resolver non câblé OU retour nil → HaloLabels (byte-identique
// Halo). Aucune comparaison de slug : le routage est encapsulé dans la closure.
func LabelsForSlug(slug string) NotifyLabels {
	defaultLabelsMu.RLock()
	fn := defaultLabelsResolver
	defaultLabelsMu.RUnlock()
	if fn == nil {
		return HaloLabels()
	}
	if l := fn(slug); l != nil {
		return l
	}
	return HaloLabels()
}
