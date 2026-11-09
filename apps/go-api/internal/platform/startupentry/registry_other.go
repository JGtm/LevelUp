//go:build !windows

package startupentry

// otherStore : hors Windows (compilation et CI Linux seulement, l'application de bureau est Windows seul), il n'y
// a pas de registre : aucune entrée n'existe.
type otherStore struct{}

// NewRegistryStore rend un [Store] sans aucune entrée.
func NewRegistryStore() Store { return otherStore{} }

func (otherStore) RunValue(string) (string, bool, error) { return "", false, nil }
func (otherStore) DeleteRunValue(string) error           { return nil }
func (otherStore) DeleteApprovedValue(string) error      { return nil }
