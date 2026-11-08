package resourcepack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// fileOps : les seules opérations d'écriture de la bascule. Tout passe par elles pour que les tests
// puissent couper le processus (ou faire échouer une suppression) à n'importe quelle étape.
type fileOps struct {
	rename    func(oldPath, newPath string) error
	remove    func(path string) error
	removeAll func(path string) error
	// writeFile écrit path d'un bloc : le fichier est entier ou absent.
	writeFile func(path string, data []byte) error
}

var osOps = fileOps{rename: os.Rename, remove: os.Remove, removeAll: os.RemoveAll, writeFile: writeFileAtomic}

// journal : ce qu'une bascule en cours a l'intention de faire, relu au démarrage suivant si le
// processus est mort au milieu.
type journal struct {
	To    string   `json:"to"`
	Units []string `json:"units"`
	// New : les unités que le paquet prêt apporte (les autres ne font que disparaître).
	New []string `json:"new"`
	// Committed : toutes les unités sont basculées. Écrit (durablement, d'un bloc) avant la moindre suppression : à
	// partir de là rien ne défait la bascule, la reprise ne fait que finir le ménage. Faux : la reprise défait.
	Committed bool `json:"committed"`
}

func journalPath(root string) string { return filepath.Join(WorkDir(root), journalName) }

// activation : ce que partagent les gestes d'une bascule ou de sa reprise.
type activation struct {
	ctx   context.Context
	ops   fileOps
	root  string
	stage string
	old   string
}

func newActivation(ctx context.Context, ops fileOps, root, fp string) activation {
	return activation{ctx: ctx, ops: ops, root: root, stage: incomingDir(root, fp), old: filepath.Join(WorkDir(root), oldDirName)}
}

// Activate bascule le paquet prêt d'empreinte fp à la place du paquet installé. Chaque unité est
// renommée (l'ancienne vers `.pack/old`, la nouvelle à sa place) ; un échec défait ce qui a été
// fait et laisse l'ancien paquet actif. Le journal, écrit avant le premier renommage et validé après le
// dernier, permet à [Recover] de défaire une bascule interrompue ou de finir le ménage d'une bascule
// validée. Une fois validée la bascule est réussie : un ménage qui échoue (fichier tenu par un autre
// processus) est journalisé et repris au démarrage suivant, il ne la défait ni ne la déclare échouée.
// À appeler au démarrage, avant que le serveur n'ouvre un seul fichier du paquet, et seulement quand
// rien d'autre ne les tient.
func Activate(ctx context.Context, root, fp string) error { return activateWith(ctx, osOps, root, fp) }

func activateWith(ctx context.Context, ops fileOps, root, fp string) error {
	if !Staged(root, fp) {
		return fmt.Errorf("resourcepack: aucun paquet prêt pour l'empreinte %s", fp)
	}
	if exists(journalPath(root)) {
		return errors.New("resourcepack: une bascule précédente n'est pas résolue (journal présent), reprise requise")
	}
	j, err := planJournal(root, fp)
	if err != nil {
		return err
	}
	if err := writeJournal(ops, root, j); err != nil {
		return err
	}
	a := newActivation(ctx, ops, root, fp)
	if err := a.swapUnits(j); err != nil {
		return err
	}
	j.Committed = true
	if err := writeJournal(ops, root, j); err != nil {
		return a.abort(j, fmt.Errorf("resourcepack: validation de la bascule: %w", err))
	}
	if err := a.finish(fp); err != nil {
		slog.WarnContext(ctx, "resources: ménage de la bascule incomplet, repris au prochain démarrage",
			"err", err, "fingerprint", fp)
	}
	return nil
}

// planJournal compose le journal de la bascule vers le paquet prêt fp : l'union des unités du paquet installé et du
// paquet prêt, le manifeste en dernier (c'est lui qui dit « activé »).
func planJournal(root, fp string) (journal, error) {
	next, err := ReadManifest(incomingDir(root, fp))
	if err != nil {
		return journal{}, err
	}
	incoming := map[string]struct{}{}
	unitsOf(next, incoming)
	all := map[string]struct{}{}
	unitsOf(next, all)
	if cur, err := ReadManifest(root); err == nil {
		unitsOf(cur, all)
	}
	return journal{To: fp, Units: append(sortedKeys(all), ManifestFileName), New: append(sortedKeys(incoming), ManifestFileName)}, nil
}

// swapUnits fait les renommages ; au premier échec, défait les unités déjà basculées.
func (a activation) swapUnits(j journal) error {
	for i, u := range j.Units {
		if err := a.swapUnit(u); err != nil {
			partial := j
			partial.Units = j.Units[:i+1]
			return a.abort(partial, fmt.Errorf("resourcepack: bascule de %s: %w", u, err))
		}
	}
	return nil
}

// abort défait les unités de j. Si tout est revenu, retire le journal (l'ancien paquet est actif) ; sinon le
// journal reste, la reprise du démarrage suivant défait le reste. Rend cause, complétée de l'échec du retour arrière.
func (a activation) abort(j journal, cause error) error {
	if err := a.rollback(j.Units, j.New); err != nil {
		return errors.Join(cause, fmt.Errorf("resourcepack: retour arrière incomplet, journal conservé pour la reprise: %w", err))
	}
	if err := a.ops.remove(journalPath(a.root)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.ErrorContext(a.ctx, "resources: journal de bascule non retiré après retour arrière, la reprise le rejouera", "err", err)
	}
	return cause
}

// swapUnit déplace l'unité u : le contenu installé vers old, le contenu prêt à sa place.
func (a activation) swapUnit(u string) error {
	live, staged, kept := unitPath(a.root, u), unitPath(a.stage, u), unitPath(a.old, u)
	if exists(live) {
		if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
			return err
		}
		if err := a.ops.rename(live, kept); err != nil {
			return err
		}
	}
	if exists(staged) {
		if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
			return err
		}
		if err := a.ops.rename(staged, live); err != nil {
			return err
		}
	}
	return nil
}

// rollback défait les unités dans l'ordre inverse : le contenu venu du paquet prêt y retourne, l'ancien
// contenu reprend sa place. Idempotent : une unité déjà revenue est laissée. Chaque échec est journalisé et
// rendu ; les unités suivantes sont tout de même tentées.
func (a activation) rollback(units, incoming []string) error {
	inPack := make(map[string]struct{}, len(incoming))
	for _, u := range incoming {
		inPack[u] = struct{}{}
	}
	var errs []error
	for i := len(units) - 1; i >= 0; i-- {
		_, fromPack := inPack[units[i]]
		errs = append(errs, a.rollbackUnit(units[i], fromPack)...)
	}
	return errors.Join(errs...)
}

func (a activation) rollbackUnit(u string, fromPack bool) []error {
	live, staged, kept := unitPath(a.root, u), unitPath(a.stage, u), unitPath(a.old, u)
	var errs []error
	move := func(step, from, to string) {
		err := os.MkdirAll(filepath.Dir(to), 0o755)
		if err == nil {
			err = a.ops.rename(from, to)
		}
		if err != nil {
			slog.ErrorContext(a.ctx, "resources: retour arrière d'une unité échoué", "unit", u, "step", step, "err", err)
			errs = append(errs, fmt.Errorf("%s (%s): %w", u, step, err))
		}
	}
	// 1. Le contenu venu du paquet prêt retourne à son dossier d'extraction.
	if fromPack && exists(live) && !exists(staged) {
		move("paquet prêt", live, staged)
	}
	// 2. L'ancien contenu reprend sa place.
	if exists(kept) && !exists(live) {
		move("ancien contenu", kept, live)
	}
	return errs
}

// finish retire l'ancien contenu, le dossier extrait, le marqueur puis, en dernier, le journal : s'il reste une
// erreur le journal reste, et la reprise du démarrage suivant refait ce ménage.
func (a activation) finish(fp string) error {
	var errs []error
	for _, p := range []string{a.old, a.stage} {
		if err := a.ops.removeAll(p); err != nil {
			errs = append(errs, err)
		}
	}
	if err := a.ops.remove(readyMarker(a.root, fp)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if err := a.ops.remove(journalPath(a.root)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Recovery : ce que [Recover] a fait d'une bascule interrompue.
type Recovery int

const (
	// RecoveryNone : aucun journal, rien à reprendre.
	RecoveryNone Recovery = iota
	// RecoveryUndone : la bascule n'était pas validée, chaque unité a retrouvé son ancien contenu.
	RecoveryUndone
	// RecoveryFinished : la bascule était validée, son ménage a été mené à terme.
	RecoveryFinished
)

// Recover reprend une bascule interrompue (journal présent) : validée, elle est finie en avant (ménage) ; non
// validée, elle est défaite (chaque unité retrouve son ancien contenu). Une erreur laisse le journal en place :
// la reprise suivante repart du même point. Sans journal, ne fait rien.
func Recover(ctx context.Context, root string) (Recovery, error) {
	return recoverWith(ctx, osOps, root)
}

func recoverWith(ctx context.Context, ops fileOps, root string) (Recovery, error) {
	raw, err := os.ReadFile(journalPath(root)) //nolint:gosec // dossier de travail du programme
	if errors.Is(err, fs.ErrNotExist) {
		return RecoveryNone, nil
	}
	if err != nil {
		return RecoveryNone, err
	}
	var j journal
	if err := json.Unmarshal(raw, &j); err != nil {
		return RecoveryNone, fmt.Errorf("resourcepack: journal de bascule illisible: %w", err)
	}
	if err := validateJournal(j); err != nil {
		return RecoveryNone, err
	}
	a := newActivation(ctx, ops, root, j.To)
	if j.Committed {
		return RecoveryFinished, a.finish(j.To)
	}
	if err := a.rollback(j.Units, j.New); err != nil {
		return RecoveryNone, fmt.Errorf("resourcepack: retour arrière incomplet, journal conservé: %w", err)
	}
	if err := ops.removeAll(a.old); err != nil {
		slog.WarnContext(ctx, "resources: dossier de l'ancien paquet non retiré après la reprise", "err", err)
	}
	return RecoveryUndone, ops.remove(journalPath(root))
}

// validateJournal refuse un journal dont une unité sort de la racine ou dont l'empreinte n'est pas une empreinte.
func validateJournal(j journal) error {
	if err := (Descriptor{Fingerprint: j.To, URL: "-", Signature: "-"}).Validate(); err != nil {
		return fmt.Errorf("resourcepack: empreinte du journal de bascule: %w", err)
	}
	for _, u := range j.Units {
		if !SafeRelPath(u) {
			return fmt.Errorf("resourcepack: unité non sûre dans le journal: %q", u)
		}
	}
	return nil
}

// writeJournal écrit le journal d'un bloc et durablement (avant le premier renommage, puis à la validation).
func writeJournal(ops fileOps, root string, j journal) error {
	if err := os.MkdirAll(WorkDir(root), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return ops.writeFile(journalPath(root), raw)
}
