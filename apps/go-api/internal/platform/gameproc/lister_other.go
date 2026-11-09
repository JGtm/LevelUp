//go:build !windows

package gameproc

// otherLister : hors Windows le client PC de Halo n'existe pas, aucun jeu n'est jamais détecté.
type otherLister struct{}

// NewSystemLister rend une énumération vide.
func NewSystemLister() Lister { return otherLister{} }

// Names ne rend aucun processus.
func (otherLister) Names() ([]string, error) { return nil, nil }
