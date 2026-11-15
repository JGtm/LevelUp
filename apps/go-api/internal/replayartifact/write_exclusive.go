package replayartifact

// write_exclusive.go — LA POSE EXCLUSIVE d'un artefact compressé : un fichier déjà présent n'est jamais remplacé.
//
// DEUX MÉCANISMES, le second n'existe que parce que le premier peut être refusé :
//
//  1. LE LIEN DUR (chemin normal). Le contenu est écrit dans un temporaire du même dossier, fsync, puis posé par un
//     lien dur : atomique (le fichier final apparaît entier ou pas du tout) et qui échoue si la cible existe.
//  2. LA CRÉATION EXCLUSIVE (repli). Les systèmes de fichiers sans lien dur (exFAT, FAT32 : disque externe, carte)
//     refusent le lien. Le dossier est alors mémorisé pour le processus, et le fichier final est créé par
//     `O_CREATE|O_EXCL` (échoue s'il existe), écrit, fsync. Cette création N'EST PAS atomique : pendant l'écriture un
//     lecteur peut ouvrir un fichier incomplet. Elle est sûre parce que la porte REFUSE un gzip incomplet (en-tête
//     absent, flux tronqué, somme de contrôle manquante : [Decode] rend une erreur) et lit alors le `.json` voisin
//     s'il existe ([Read]). Un échec d'écriture supprime le fichier partiel ; un arrêt brutal en laisse un, que la
//     conversion du parc reconnaît et reprend (`replaybuild.CompressDir`).
//
// Toute erreur du lien dur autre que « la cible existe » fait replier CE fichier (le repli est lui-même exclusif, donc
// sans danger si l'erreur avait une autre cause : il rend alors la sienne). Mais le DOSSIER n'est mémorisé « sans lien
// dur » que sur une erreur qui signifie « non pris en charge » ([lienNonPrisEnCharge]) : une erreur transitoire
// (antivirus, violation de partage) ne condamne pas le lien dur pour tout le processus ; elle est journalisée (WARN) et
// comptée ([CounterLinkError]).

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"syscall"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// CounterLinkFallback : artefacts posés par création exclusive parce que le dossier refuse le lien dur.
const CounterLinkFallback = "replay_artifact_link_fallback_total"

// CounterLinkError : liens durs refusés par une erreur qui NE signifie PAS « non pris en charge » (le fichier est posé par
// le repli, le dossier n'est pas mémorisé).
const CounterLinkError = "replay_artifact_link_error_total"

// Points d'injection des tests ; la production n'a qu'une valeur de chacun.
var (
	// linkFile : la pose par lien dur (échoue si la cible existe).
	linkFile = os.Link
	// ecrireEtSynchroniser écrit tous les octets d'un fichier du repli et le synchronise.
	ecrireEtSynchroniser = func(f *os.File, data []byte) error {
		if _, err := f.Write(data); err != nil {
			return err
		}
		return f.Sync()
	}
)

// Codes Windows d'un lien dur refusé par le système de fichiers : ERROR_INVALID_FUNCTION (exFAT, FAT32 renvoient
// « Incorrect function »), ERROR_NOT_SUPPORTED, ERROR_CALL_NOT_IMPLEMENTED.
var codesWindowsSansLien = []syscall.Errno{1, 50, 120}

// Codes POSIX : EPERM (FAT et exFAT sous Linux refusent le lien par « Operation not permitted »), ENOTSUP et
// EOPNOTSUPP (même valeur ou non selon l'OS), ENOSYS.
var codesPosixSansLien = []syscall.Errno{syscall.EPERM, syscall.ENOTSUP, syscall.EOPNOTSUPP, syscall.ENOSYS}

// lienNonPrisEnCharge dit si l'erreur d'un lien dur signifie « ce système de fichiers ne sait pas faire de lien dur » :
// `errors.ErrUnsupported` (que la bibliothèque standard fait correspondre à ENOSYS, ENOTSUP, EOPNOTSUPP et
// ERROR_NOT_SUPPORTED) ou l'un des codes ci-dessus de l'OS courant. Tout le reste (accès refusé, violation de partage,
// disque plein) est traité comme transitoire.
func lienNonPrisEnCharge(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errors.ErrUnsupported) {
		return true
	}
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	codes := codesPosixSansLien
	if runtime.GOOS == "windows" {
		codes = codesWindowsSansLien
	}
	return slices.Contains(codes, errno)
}

// dossiersSansLienDur : les dossiers dont le lien dur a été refusé dans ce processus (clé : le dossier).
var dossiersSansLienDur sync.Map

// errLienRefuse enveloppe l'erreur d'un lien dur refusé (autre chose que « existe déjà »).
type errLienRefuse struct{ err error }

func (e errLienRefuse) Error() string { return "lien dur refusé : " + e.err.Error() }
func (e errLienRefuse) Unwrap() error { return e.err }

// WriteIfAbsent écrit le JSON d'un artefact, compressé, sous son nom canonique, SANS JAMAIS remplacer un fichier déjà
// présent (cf. l'en-tête du fichier : lien dur, ou création exclusive sur un dossier qui le refuse). `created` dit si
// le fichier a été posé ; faux = un `.json.gz` existait déjà et n'a pas été touché (un serveur actif en a posé un
// plus récent, ou une conversion précédente a été interrompue) : l'appelant juge alors le doublon. Aucun temporaire
// ne survit.
func WriteIfAbsent(ctx context.Context, outPath string, doc []byte) (written string, stored []byte, created bool, err error) {
	written, _ = title.ReplayArtifactCandidates(outPath)
	stored, err = Encode(doc)
	if err != nil {
		return written, nil, false, fmt.Errorf("compression de l'artefact : %w", err)
	}
	dir := filepath.Dir(written)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return written, nil, false, err
	}
	if _, sans := dossiersSansLienDur.Load(dir); !sans {
		created, err = poserParLien(ctx, dir, written, stored)
		var refus errLienRefuse
		if !errors.As(err, &refus) {
			return written, orNil(stored, err), created, err
		}
		if lienNonPrisEnCharge(refus.err) {
			dossiersSansLienDur.Store(dir, struct{}{})
			slog.InfoContext(ctx, "replayartifact: ce dossier refuse le lien dur (exFAT, FAT32 ?) — les artefacts y sont "+
				"posés par création exclusive", "dir", dir, "err", refus.err)
		} else {
			observability.IncCounter(CounterLinkError)
			slog.WarnContext(ctx, "replayartifact: lien dur refusé pour une raison qui n'est pas « non pris en charge » — "+
				"cet artefact est posé par création exclusive, le dossier n'est pas mémorisé", "path", written, "err", refus.err)
		}
	}
	created, err = poserParCreationExclusive(ctx, written, stored)
	if err == nil && created {
		observability.IncCounter(CounterLinkFallback)
	}
	return written, orNil(stored, err), created, err
}

func orNil(stored []byte, err error) []byte {
	if err != nil {
		return nil
	}
	return stored
}

// poserParLien : temporaire, fsync, lien dur. created = faux sans erreur : la cible existait.
func poserParLien(ctx context.Context, dir, written string, stored []byte) (bool, error) {
	tmp, err := os.CreateTemp(dir, ".levelup-atomic-*")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	defer func() {
		if rerr := os.Remove(tmpName); rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
			slog.WarnContext(ctx, "replayartifact: temporaire non supprimé", "path", tmpName, "err", rerr)
		}
	}()
	if _, err = tmp.Write(stored); err == nil {
		err = tmp.Sync()
	}
	if err == nil {
		err = tmp.Chmod(0o644) // le mode d'un artefact écrit par la porte, comme atomicfile.WriteFile
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return false, err
	}
	if err = linkFile(tmpName, written); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, errLienRefuse{err: err}
	}
	return true, nil
}

// poserParCreationExclusive : le repli. Crée le fichier final exclusivement, l'écrit, le synchronise ; un échec le
// supprime (jamais de partiel laissé par un échec que ce processus voit).
func poserParCreationExclusive(ctx context.Context, written string, stored []byte) (bool, error) {
	f, err := os.OpenFile(written, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // chemin canonique de la porte
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	err = ecrireEtSynchroniser(f, stored)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		if rerr := os.Remove(written); rerr != nil && !errors.Is(rerr, fs.ErrNotExist) {
			slog.ErrorContext(ctx, "replayartifact: artefact partiel non supprimé après un échec d'écriture",
				"path", written, "err", rerr)
		}
		return false, fmt.Errorf("écriture exclusive de l'artefact : %w", err)
	}
	return true, nil
}
