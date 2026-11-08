package resourcepack

import (
	"errors"
	"os"
)

// syncFile force f sur le disque. Variable pour que les tests voient quels fichiers sont rendus
// durables et fassent échouer le geste ; jamais réaffectée hors tests.
var syncFile = func(f *os.File) error { return f.Sync() }

// writeFileSync écrit data dans path puis le force sur le disque avant de rendre la main : après un retour sans
// erreur, une coupure de courant ne laisse pas un fichier de contenu nul.
func writeFileSync(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm) //nolint:gosec // chemin du dossier de travail ou du paquet
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	if werr == nil {
		werr = syncFile(f)
	}
	return errors.Join(werr, f.Close())
}

// writeFileAtomic écrit path d'un bloc et durablement : le contenu complet est forcé sur le disque dans un fichier
// voisin, puis renommé sur path. Le fichier est ancien ou entier, jamais tronqué.
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := writeFileSync(tmp, data, 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
