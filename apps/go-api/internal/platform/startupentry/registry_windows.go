//go:build windows

package startupentry

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey      = `Software\Microsoft\Windows\CurrentVersion\Run`
	approvedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
)

// registryStore : les valeurs de démarrage de HKCU (jamais HKLM : l'entrée est par utilisateur). Les deux
// clés sont des champs pour que le test vise une clé privée à nom aléatoire, jamais la vraie clé Run.
type registryStore struct {
	run      string
	approved string
}

// NewRegistryStore rend le [Store] du registre de l'utilisateur courant.
func NewRegistryStore() Store { return registryStore{run: runKey, approved: approvedKey} }

func (s registryStore) RunValue(name string) (string, bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, s.run, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("ouverture de la clé %s : %w", s.run, err)
	}
	defer k.Close()
	value, _, err := k.GetStringValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lecture de %s : %w", name, err)
	}
	return value, true, nil
}

func (s registryStore) DeleteRunValue(name string) error { return deleteValue(s.run, name) }

func (s registryStore) DeleteApprovedValue(name string) error { return deleteValue(s.approved, name) }

// deleteValue supprime la valeur name de la clé de HKCU ; clé ou valeur absente = succès.
func deleteValue(key, name string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("ouverture de la clé %s : %w", key, err)
	}
	defer k.Close()
	if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("suppression de %s : %w", name, err)
	}
	return nil
}
