package replayartifact

// head.go — le début décompressé d'un fichier : de quoi lire l'en-tête d'un document sans le décompresser en entier.

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// ReadHead rend au plus maxBytes octets du DÉBUT du JSON décompressé du fichier désigné par `path`. Seul le début du
// flux compressé est lu et décompressé : le coût ne dépend pas de la taille du document. Une erreur enveloppe
// fs.ErrNotExist pour un fichier absent ; un fichier sans en-tête gzip ou vide est une erreur. Aucun journal ni compteur :
// l'appelant qui retombe sur [Read] en est le témoin.
func ReadHead(path string, maxBytes int) ([]byte, error) {
	f, err := Locate(path)
	if err != nil {
		return nil, err
	}
	fh, err := os.Open(f.Path) //nolint:gosec // chemin résolu par la porte
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	zr, err := gzip.NewReader(fh)
	if err != nil {
		return nil, fmt.Errorf("début de %s : %w", f.Path, err)
	}
	defer func() { _ = zr.Close() }()
	head, err := io.ReadAll(io.LimitReader(zr, int64(maxBytes)))
	if err != nil && len(head) == 0 {
		return nil, fmt.Errorf("début de %s : %w", f.Path, err)
	}
	return head, nil
}
