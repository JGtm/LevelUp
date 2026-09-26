package main

// cmd_compact_passes_rewrite.go — `compact-passes --rewrite-file` (DC.5) : rendre au disque la
// place que la compaction a libérée.
//
// DuckDB réutilise les blocs libérés mais ne rétrécit jamais un fichier : seule une recopie dans
// un fichier NEUF (migration.CopierBaseVers, COPY FROM DATABASE) le réduit. L'échange n'a lieu
// que si le fichier neuf, rouvert SEUL, porte exactement le même inventaire que l'ancien —
// catalogue (tables, vues, index, séquences avec leur prochaine valeur, macros, types), compte
// de chaque table, empreinte de chaque vue compactée. Sinon le fichier neuf est supprimé et la
// base reste telle quelle. L'ancien fichier est GARDÉ, renommé `<nom>.avant-reecriture-<t>`.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// reecrireFichier réécrit `path` (appelé sous le bail d'écrivain, handle de compaction fermé).
func reecrireFichier(ctx context.Context, path, dossierSauvegarde string) error {
	neuf := path + ".reecriture"
	if _, err := os.Stat(neuf); err == nil {
		return fmt.Errorf("réécriture : %s existe déjà (reste d'une réécriture interrompue ?) — "+
			"le retirer à la main après vérification", neuf)
	}
	avant, err := copierVersFichierNeuf(ctx, path, neuf)
	if err != nil {
		return errors.Join(err, retirerFichierNeuf(neuf))
	}
	apres, err := inventaireDuFichier(ctx, neuf)
	if err != nil {
		return errors.Join(err, retirerFichierNeuf(neuf))
	}
	if ecarts := avant.Ecarts(apres); len(ecarts) > 0 {
		return errors.Join(fmt.Errorf("réécriture refusée, le fichier neuf diffère :\n%s",
			strings.Join(ecarts, "\n")), retirerFichierNeuf(neuf))
	}
	ancien, err := echangerFichiers(path, neuf, dossierSauvegarde)
	if err != nil {
		return err
	}
	fmt.Printf("réécriture : inventaire identique (%d objets, %d tables, %d vues compactées) ; "+
		"ancien fichier gardé : %s\n", len(apres.Objets), len(apres.Comptes), len(apres.Vues), ancien)
	return nil
}

// copierVersFichierNeuf : inventaire de la base SEULE, puis recopie dans `neuf`, handle fermé au
// retour (le fichier doit être libre pour l'échange).
func copierVersFichierNeuf(ctx context.Context, path, neuf string) (migration.Inventaire, error) {
	handle, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return migration.Inventaire{}, fmt.Errorf("réécriture : ouverture de %s: %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	avant, err := migration.LireInventaire(ctx, handle.SQLDb())
	if err != nil {
		return avant, fmt.Errorf("réécriture : inventaire de %s: %w", path, err)
	}
	if err := migration.CopierBaseVers(ctx, handle.SQLDb(), neuf); err != nil {
		return avant, err
	}
	return avant, nil
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

// echangerFichiers : l'ancien fichier part en sauvegarde, le neuf prend sa place. Un WAL restant
// d'un côté ou de l'autre arrête tout (il appartient à SON fichier) ; un WAL vide est retiré AVANT
// l'échange, pour qu'aucun fichier ne reprenne celui de l'autre.
func echangerFichiers(path, neuf, dossier string) (string, error) {
	for _, p := range []string{path + ".wal", neuf + ".wal"} {
		if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
			return "", fmt.Errorf("réécriture : %s non vide, échange refusé", p)
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("réécriture : WAL vide %s: %w", p, err)
		}
	}
	if dossier == "" {
		dossier = filepath.Dir(path)
	}
	nom := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	ancien := filepath.Join(dossier, fmt.Sprintf("%s.avant-reecriture-%s%s", nom,
		time.Now().UTC().Format("20060102T150405Z"), filepath.Ext(path)))
	if err := os.Rename(path, ancien); err != nil {
		return "", fmt.Errorf("réécriture : mise de côté de %s: %w", path, err)
	}
	if err := os.Rename(neuf, path); err != nil {
		// L'ancien revient à sa place : la base ne reste jamais sans fichier.
		if errRetour := os.Rename(ancien, path); errRetour != nil {
			return "", fmt.Errorf("réécriture : ÉCHANGE INTERROMPU, base à remettre à la main "+
				"depuis %s : %w", ancien, errors.Join(err, errRetour))
		}
		return "", fmt.Errorf("réécriture : mise en place de %s (ancien fichier remis): %w", neuf, err)
	}
	return ancien, nil
}

func retirerFichierNeuf(neuf string) error {
	for _, p := range []string{neuf, neuf + ".wal"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("réécriture : retrait de %s: %w", p, err)
		}
	}
	return nil
}
