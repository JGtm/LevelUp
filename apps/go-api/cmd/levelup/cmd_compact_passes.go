package main

// cmd_compact_passes.go — sous-commande `levelup compact-passes` : L'ENTRETIEN qui retire les
// passes de décodage supersédées de la base partagée des matchs (plan perf du 2026-09-26, étape C).
//
// POURQUOI UNE COMMANDE ET PAS UN AUTOMATISME (DC.6). Chaque redécodage d'un film ajoute une passe
// entière aux tables du film (INSERT-only, ADR 0019 / 0026) ; la vue `_latest` n'en sert qu'une,
// mais toute lecture la paie en entier. La compaction reconstruit ces tables à leurs seules lignes
// servies (migration.CompactSupersededPasses : jamais de DELETE, swap transactionnel vérifié).
// Elle exige l'ÉCRIVAIN EXCLUSIF — le serveur arrêté — : ni au boot, ni après un sync. Elle se joue
// après une campagne de redécodage (recuisson, `backfill-killsource`).
//
// Nom : `compact-passes`, verbe + objet comme `rebuild-pme-art` (autre entretien serveur arrêté) —
// l'objet est la PASSE, pas la table : seules les passes supersédées partent.
//
// Usage (SERVEUR ARRÊTÉ) :
//
//	levelup compact-passes --dry-run                 # lecture seule : brutes / _latest / gain par table
//	levelup compact-passes                           # sauvegarde, puis compaction, tous les titres
//	levelup compact-passes --title halo_5            # un seul titre
//	levelup compact-passes --rewrite-file            # + réécriture du fichier (rend la place au disque)
//	levelup compact-passes --backup-dir D:\sauvegardes
//
// La sauvegarde préalable n'est pas optionnelle : copie OCTET POUR OCTET du fichier (après
// CHECKPOINT, fichier fermé), à côté de la base par défaut. L'outillage de
// sauvegarde existant (`backup`, pkg/duckdbbackup) exporte en Parquet table par table : il
// perd séquences, vues et index, il ne restaure donc pas cette base à l'identique.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

// compactPassesOptions : les réglages de la commande.
type compactPassesOptions struct {
	title       string
	dryRun      bool
	rewriteFile bool
	backupDir   string
}

// compactLeaseTimeout : attente du bail d'écrivain du fichier (ADR 0013). Variable pour les tests.
var compactLeaseTimeout = dblease.SharedLeaseTimeout

func runCompactPasses(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("compact-passes", flag.ExitOnError)
	o := compactPassesOptions{}
	fs.StringVar(&o.title, "title", "", "slug du titre (défaut : chaque titre du registre dont la base partagée existe)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "lecture seule : mesure par table, aucune écriture, aucune sauvegarde")
	fs.BoolVar(&o.rewriteFile, "rewrite-file", false, "après compaction, réécrire le fichier pour rendre la place au disque")
	fs.StringVar(&o.backupDir, "backup-dir", "", "dossier des sauvegardes (défaut : celui de la base)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.dryRun && o.rewriteFile {
		return fmt.Errorf("--rewrite-file écrit : incompatible avec --dry-run")
	}
	reg := titlePkg.NewRegistryFromConfig(cfg.RepoRoot, slog.Default())
	pr := titlePkg.NewPathResolver(cfg.RepoRoot, reg)
	slugs, err := titresACompacter(reg, pr, o.title)
	if err != nil {
		return err
	}
	ctx := context.Background()
	for _, slug := range slugs {
		if err := compacterTitre(ctx, slug, pr.SharedDBPath(slug), o); err != nil {
			return fmt.Errorf("compact-passes %s: %w", slug, err)
		}
	}
	return nil
}

// titresACompacter : le titre demandé (qui doit exister au registre), ou chaque titre du
// registre dont la base partagée existe sur disque.
func titresACompacter(reg *titlePkg.Registry, pr *titlePkg.PathResolver, demande string) ([]string, error) {
	if demande != "" {
		if !reg.Exists(demande) {
			return nil, fmt.Errorf("titre inconnu du registre : %q", demande)
		}
		return []string{demande}, nil
	}
	var out []string
	for _, d := range reg.All() {
		if _, err := os.Stat(pr.SharedDBPath(d.Slug)); err == nil {
			out = append(out, d.Slug)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("aucune base partagée trouvée sous %s", pr.TitlesRootDir())
	}
	return out, nil
}

// compacterTitre : la base partagée d'un titre, en dry-run ou pour de bon.
func compacterTitre(ctx context.Context, slug, path string, o compactPassesOptions) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("base partagée introuvable (%s): %w", path, err)
	}
	fmt.Printf("== %s : %s (%s)\n", slug, path, tailleLisible(tailleDe(path)))
	if o.dryRun {
		return mesurerSansEcrire(ctx, path)
	}
	release, err := dblease.AcquireLease(path, compactLeaseTimeout)
	if err != nil {
		return fmt.Errorf("refus : bail d'écrivain indisponible : %w", err)
	}
	defer release()

	avant := tailleDe(path)
	if err := compacterSousVerrou(ctx, slug, path, o); err != nil {
		return err
	}
	if o.rewriteFile {
		if err := reecrireFichier(ctx, path, o.backupDir); err != nil {
			return err
		}
	}
	fmt.Printf("fichier : %s -> %s\n", tailleLisible(avant), tailleLisible(tailleDe(path)))
	return nil
}

// mesurerSansEcrire : le dry-run — ouverture en LECTURE SEULE (refusée si un autre processus
// tient la base en écriture), aucune migration, aucune écriture.
func mesurerSansEcrire(ctx context.Context, path string) error {
	handle, err := duckdb.OpenReadOnly(path)
	if err != nil {
		return fmt.Errorf("refus : %s est tenue par un autre processus (serveur arrêté ?) : %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	rs, err := migration.CompactSupersededPasses(ctx, handle.SQLDb(), true)
	imprimerRapportCompaction(rs)
	return err
}

// compacterSousVerrou : préparation (migrations, CHECKPOINT) et sauvegarde, puis compaction sur un
// handle rouvert. Le handle est fermé au retour.
//
// LA SAUVEGARDE SE FAIT FICHIER FERMÉ : sous Windows, un fichier que DuckDB tient en écriture ne
// s'ouvre pas en lecture (mesuré, « utilisé par un autre processus »). L'ouverture en écriture qui
// précède prouve que personne d'autre ne tenait la base ; la réouverture qui suit échoue si un
// autre processus l'a prise entre-temps — rien n'est alors compacté.
func compacterSousVerrou(ctx context.Context, slug, path string, o compactPassesOptions) error {
	if err := preparerPourSauvegarde(ctx, slug, path); err != nil {
		return err
	}
	sauvegarde, err := copierFichier(ctx, path, o.backupDir, "avant-compaction")
	if err != nil {
		return fmt.Errorf("sauvegarde préalable impossible, rien n'est compacté : %w", err)
	}
	fmt.Printf("sauvegarde : %s\n", sauvegarde)
	handle, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("refus : %s a été prise par un autre processus après la sauvegarde, "+
			"rien n'est compacté : %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	rs, err := migration.CompactSupersededPasses(ctx, handle.SQLDb(), false)
	imprimerRapportCompaction(rs)
	return err
}

// preparerPourSauvegarde : ouverture en écriture (le verrou du fichier refuse un autre
// processus), migrations du titre, CHECKPOINT (le WAL rejoint le fichier), fermeture.
func preparerPourSauvegarde(ctx context.Context, slug, path string) error {
	handle, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("refus : %s est tenue par un autre processus (serveur arrêté ?) : %w", path, err)
	}
	defer closeLogged(ctx, handle, path)
	db := handle.SQLDb()
	// Un orphelin se refuse AVANT les migrations et AVANT la sauvegarde (migration.refuserOrphelin).
	if err := migration.VerifierAucunOrphelin(ctx, db); err != nil {
		return err
	}
	// Les tables et vues du registre doivent être à jour AVANT d'être lues : cette commande tourne
	// serveur arrêté, rien n'a joué les migrations pour elle.
	if err := migration.RunForTitleDB(db, slug, migration.TargetShared); err != nil {
		return fmt.Errorf("migrations shared: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("checkpoint avant sauvegarde: %w", err)
	}
	return nil
}

func closeLogged(ctx context.Context, handle *duckdb.DB, path string) {
	if err := handle.Close(); err != nil {
		slog.ErrorContext(ctx, "compact-passes: fermeture de la base", "path", path, "err", err)
	}
}

// cheminDeSauvegarde rend `<dossier>/<nom>.<etiquette>-<horodatage UTC><ext>` (dossier de la base
// si `dossier` est vide), dossier créé au besoin.
func cheminDeSauvegarde(path, dossier, etiquette string) (string, error) {
	if dossier == "" {
		dossier = filepath.Dir(path)
	}
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		return "", err
	}
	nom := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return filepath.Join(dossier, fmt.Sprintf("%s.%s-%s%s", nom, etiquette,
		time.Now().UTC().Format("20060102T150405Z"), filepath.Ext(path))), nil
}

// copierFichier copie `path` octet pour octet sous cheminDeSauvegarde et rend le chemin écrit.
// Refus si la base porte un WAL non vide (le CHECKPOINT qui précède doit l'avoir vidé). Une copie
// qui échoue en cours de route (disque plein, Sync) est RETIRÉE : un fichier tronqué ne reste
// jamais sous un nom de sauvegarde valide.
func copierFichier(ctx context.Context, path, dossier, etiquette string) (string, error) {
	if fi, err := os.Stat(path + ".wal"); err == nil && fi.Size() > 0 {
		return "", fmt.Errorf("%s.wal non vide (%d octets) : la copie ne serait pas la base entière", path, fi.Size())
	}
	cible, err := cheminDeSauvegarde(path, dossier, etiquette)
	if err != nil {
		return "", err
	}
	src, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer src.Close() //nolint:errcheck // lecture seule
	dst, err := os.OpenFile(cible, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	_, errCopie := io.Copy(dst, src)
	if errCopie == nil {
		errCopie = dst.Sync()
	}
	if err := errors.Join(errCopie, dst.Close()); err != nil {
		errRetrait := retirerFichiers(cible)
		slog.ErrorContext(ctx, "sauvegarde en échec, copie partielle retirée", "cible", cible,
			"err", err, "err_retrait", errRetrait)
		return "", errors.Join(fmt.Errorf("copie vers %s: %w", cible, err), errRetrait)
	}
	return cible, nil
}

func tailleDe(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func tailleLisible(n int64) string {
	return fmt.Sprintf("%.1f Mio", float64(n)/(1<<20))
}

// imprimerRapportCompaction : une ligne par table — la sortie que l'exploitant lit.
func imprimerRapportCompaction(rs []migration.CompactionTable) {
	fmt.Printf("%-28s %-14s %10s %10s %10s %10s %7s %9s\n",
		"table", "statut", "brutes", "_latest", "gardees", "apres", "gain", "duree")
	var brutes, apres int64
	for _, r := range rs {
		gain := "-"
		if r.Raw > 0 {
			gain = fmt.Sprintf("%.0f %%", 100*float64(r.Raw-r.After)/float64(r.Raw))
		}
		fmt.Printf("%-28s %-14s %10d %10d %10d %10d %7s %9s\n", r.Table, r.Status, r.Raw,
			r.Latest, r.Kept, r.After, gain, r.Duration.Round(time.Millisecond))
		brutes += r.Raw
		apres += r.After
	}
	fmt.Printf("total : %d lignes brutes -> %d\n", brutes, apres)
}
