package main

import (
	"context"
	"os"

	"levelup/go-api/internal/replayartifact"
)

// poserCompresse range un artefact de test par la porte (compressé, au chemin p) ; la signature suit
// os.WriteFile pour que les écritures des tests de ce paquet ne changent que de nom.
func poserCompresse(p string, raw []byte, _ os.FileMode) error {
	_, _, err := replayartifact.Write(context.Background(), p, raw)
	return err
}
