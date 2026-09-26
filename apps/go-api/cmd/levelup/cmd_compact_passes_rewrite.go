package main

// cmd_compact_passes_rewrite.go — `compact-passes --rewrite-file` (DC.5, amendé en C.8) : rendre
// au disque la place que la compaction a libérée.
//
// DuckDB réutilise les blocs libérés mais ne rétrécit jamais un fichier : seule une recopie dans
// un fichier NEUF (migration.CopierBaseVers, COPY FROM DATABASE) le réduit. L'échange n'a lieu
// que si le fichier neuf, rouvert SEUL, porte exactement le même inventaire que l'ancien —
// catalogue (tables, vues, index, séquences avec leur prochaine valeur, macros, types), compte
// de chaque table, empreinte de chaque vue compactée.
//
// LES TROIS RÈGLES DE L'ÉCHANGE (revue adversariale L1 du 2026-09-26, C.8) :
//   - JAMAIS de fenêtre sans fichier au chemin de la base : l'ancien fichier n'est pas déplacé,
//     il est COPIÉ (sauvegarde `<nom>.avant-reecriture-<t>`), puis UN SEUL `rename(neuf, base)`
//     remplace la base en une opération (POSIX `rename` atomique ; sous Windows, `os.Rename`
//     remplace un fichier existant). Un arrêt à n'importe quel instant laisse au chemin de la base
//     soit l'ancienne base, soit la neuve, complètes ;
//   - la sauvegarde est une COPIE (io.Copy), jamais un rename : `--backup-dir` peut être sur un
//     autre volume. Le seul rename reste dans le dossier de la base (`<base>.reecriture` y est
//     écrit) ;
//   - juste avant l'échange, la base doit n'avoir PAS bougé depuis la copie (taille et date de
//     modification relevées à la fermeture du handle de copie) et n'être tenue par personne
//     (ouverture exclusive en écriture qui réussit, puis fermeture) : sous Linux, un processus
//     qui aurait ouvert la base entre-temps continuerait d'écrire dans l'inode remplacé, et ses
//     écritures disparaîtraient sans erreur. Sinon : refus, base intacte.
//
// Toute erreur après la création de `<base>.reecriture` le retire (et la sauvegarde devenue
// inutile), journalisée avant que l'erreur ne remonte. Un `<base>.reecriture` trouvé au départ
// est le reste d'une réécriture interrompue : la base, elle, est complète par construction ; le
// reste est retiré (WARN) au lieu de bloquer les réécritures suivantes.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// etapeReecriture : point d'observation entre les étapes de la réécriture (tests : arrêt simulé,
// source modifiée ou tenue). nil en production.
var etapeReecriture func(etape string) error

// Les étapes observables, dans l'ordre.
const (
	etapeApresCopie        = "apres-copie"
	etapeApresInventaire   = "apres-inventaire"
	etapeApresSauvegarde   = "apres-sauvegarde"
	etapeApresVerification = "apres-verification"
	etapeApresEchange      = "apres-echange"
)

func passerEtape(nom string) error {
	if etapeReecriture == nil {
		return nil
	}
	return etapeReecriture(nom)
}

// renommer : os.Rename, remplaçable par les tests (un rename entre deux volumes échoue).
var renommer = os.Rename

// etatFichier : ce qui dit qu'un fichier n'a pas bougé.
type etatFichier struct {
	taille int64
	modif  time.Time
}

func lireEtatFichier(path string) (etatFichier, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return etatFichier{}, err
	}
	return etatFichier{taille: fi.Size(), modif: fi.ModTime()}, nil
}

// reecriture : l'état d'une réécriture en cours, pour l'abandon.
type reecriture struct {
	path, neuf, sauvegarde string
}

// reecrireFichier réécrit `path` (appelé sous le bail d'écrivain, handle de compaction fermé).
func reecrireFichier(ctx context.Context, path, dossierSauvegarde string) error {
	r := reecriture{path: path, neuf: path + ".reecriture"}
	if err := retirerResteInterrompu(ctx, r.neuf); err != nil {
		return err
	}
	avant, etat, err := copierVersFichierNeuf(ctx, path, r.neuf)
	if err == nil {
		err = passerEtape(etapeApresCopie)
	}
	if err != nil {
		return r.abandonner(ctx, err)
	}
	apres, err := inventaireDuFichier(ctx, r.neuf)
	if err == nil {
		err = verifierInventaire(avant, apres)
	}
	if err == nil {
		err = passerEtape(etapeApresInventaire)
	}
	if err != nil {
		return r.abandonner(ctx, err)
	}
	if r.sauvegarde, err = copierFichier(path, dossierSauvegarde, "avant-reecriture"); err != nil {
		return r.abandonner(ctx, fmt.Errorf("réécriture : sauvegarde de %s: %w", path, err))
	}
	if err := r.echanger(ctx, etat); err != nil {
		return r.abandonner(ctx, err)
	}
	fmt.Printf("réécriture : inventaire identique (%d objets, %d tables, %d vues compactées) ; "+
		"ancien fichier sauvegardé : %s\n", len(apres.Objets), len(apres.Comptes), len(apres.Vues), r.sauvegarde)
	return passerEtape(etapeApresEchange)
}

func verifierInventaire(avant, apres migration.Inventaire) error {
	if ecarts := avant.Ecarts(apres); len(ecarts) > 0 {
		return fmt.Errorf("réécriture refusée, le fichier neuf diffère :\n%s", strings.Join(ecarts, "\n"))
	}
	return nil
}

// echanger : dernières vérifications de la source, puis le rename unique.
func (r reecriture) echanger(ctx context.Context, etat etatFichier) error {
	if err := passerEtape(etapeApresSauvegarde); err != nil {
		return err
	}
	if err := verifierSourceInchangee(r.path, etat); err != nil {
		return err
	}
	if err := verifierSourceLibre(ctx, r.path); err != nil {
		return err
	}
	if err := passerEtape(etapeApresVerification); err != nil {
		return err
	}
	// Un WAL VIDE resté à côté d'un fichier serait repris par l'autre après l'échange.
	for _, p := range []string{r.path + ".wal", r.neuf + ".wal"} {
		if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
			return fmt.Errorf("réécriture : %s non vide, échange refusé", p)
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("réécriture : WAL vide %s: %w", p, err)
		}
	}
	if err := renommer(r.neuf, r.path); err != nil {
		return fmt.Errorf("réécriture : remplacement de %s: %w", r.path, err)
	}
	return nil
}

// verifierSourceInchangee : la base est-elle celle que la copie a lue ?
func verifierSourceInchangee(path string, etat etatFichier) error {
	maintenant, err := lireEtatFichier(path)
	if err != nil {
		return fmt.Errorf("réécriture : état de %s: %w", path, err)
	}
	if maintenant.taille != etat.taille || !maintenant.modif.Equal(etat.modif) {
		return fmt.Errorf("réécriture refusée : %s a changé depuis la copie (taille %d -> %d, "+
			"modifiée %s -> %s) — un autre processus l'a ouverte ?", path, etat.taille,
			maintenant.taille, etat.modif.Format(time.RFC3339Nano), maintenant.modif.Format(time.RFC3339Nano))
	}
	return nil
}

// verifierSourceLibre : une ouverture EXCLUSIVE en écriture réussit (personne ne tient la base),
// puis fermeture.
func verifierSourceLibre(ctx context.Context, path string) error {
	handle, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("réécriture refusée : %s est tenue par un autre processus : %w", path, err)
	}
	if err := handle.Close(); err != nil {
		return fmt.Errorf("réécriture : fermeture de %s: %w", path, err)
	}
	slog.DebugContext(ctx, "réécriture : base libre avant l'échange", "path", path)
	return nil
}

// abandonner retire le fichier neuf et la sauvegarde devenue inutile, journalise, rend l'erreur.
func (r reecriture) abandonner(ctx context.Context, cause error) error {
	errRetrait := retirerFichiers(r.neuf, r.neuf+".wal")
	if r.sauvegarde != "" {
		errRetrait = errors.Join(errRetrait, retirerFichiers(r.sauvegarde))
	}
	slog.ErrorContext(ctx, "réécriture abandonnée, base laissée telle quelle", "path", r.path,
		"err", cause, "err_retrait", errRetrait)
	return errors.Join(cause, errRetrait)
}

// retirerResteInterrompu : un `<base>.reecriture` présent au départ vient d'une réécriture
// interrompue ; la base est complète par construction, le reste est retiré.
func retirerResteInterrompu(ctx context.Context, neuf string) error {
	if _, err := os.Stat(neuf); os.IsNotExist(err) {
		return nil
	}
	slog.WarnContext(ctx, "réécriture : reste d'une réécriture interrompue retiré", "fichier", neuf)
	return retirerFichiers(neuf, neuf+".wal")
}

// copierVersFichierNeuf : inventaire de la base SEULE, puis recopie dans `neuf`, handle fermé au
// retour ; rend aussi l'état du fichier à cet instant (référence de verifierSourceInchangee).
func copierVersFichierNeuf(ctx context.Context, path, neuf string) (migration.Inventaire, etatFichier, error) {
	avant, err := inventaireEtCopie(ctx, path, neuf)
	if err != nil {
		return avant, etatFichier{}, err
	}
	etat, err := lireEtatFichier(path)
	if err != nil {
		return avant, etat, fmt.Errorf("réécriture : état de %s: %w", path, err)
	}
	return avant, etat, nil
}

func inventaireEtCopie(ctx context.Context, path, neuf string) (migration.Inventaire, error) {
	handle, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return migration.Inventaire{}, fmt.Errorf("réécriture : ouverture de %s: %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	avant, err := migration.LireInventaire(ctx, handle.SQLDb())
	if err != nil {
		return avant, fmt.Errorf("réécriture : inventaire de %s: %w", path, err)
	}
	return avant, migration.CopierBaseVers(ctx, handle.SQLDb(), neuf)
}

// inventaireDuFichier relit le fichier neuf SEUL (aucun ATTACH : ses vues se lient à ses tables).
func inventaireDuFichier(ctx context.Context, neuf string) (migration.Inventaire, error) {
	handle, err := duckdb.OpenReadOnly(neuf)
	if err != nil {
		return migration.Inventaire{}, fmt.Errorf("réécriture : ouverture de %s: %w", neuf, err)
	}
	defer closeLogged(ctx, handle, neuf)
	inv, err := migration.LireInventaire(ctx, handle.SQLDb())
	if err != nil {
		return inv, fmt.Errorf("réécriture : inventaire de %s: %w", neuf, err)
	}
	return inv, nil
}

func retirerFichiers(chemins ...string) error {
	var errs error
	for _, p := range chemins {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			errs = errors.Join(errs, fmt.Errorf("réécriture : retrait de %s: %w", p, err))
		}
	}
	return errs
}
