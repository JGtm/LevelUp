package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// MissingPolicy règle ce que la passe de réconciliation fait d'un média dont
// aucun fichier n'existe plus sur le disque.
type MissingPolicy int

const (
	// MissingIgnore : ne marque jamais et ne déclenche pas le bandeau. Défaut des
	// chemins qui ne voient qu'une partie du dossier (envoi, CLI).
	MissingIgnore MissingPolicy = iota
	// MissingMark : racine fiable, la passe marque les médias introuvables sous le
	// garde-fou de disparition massive.
	MissingMark
	// MissingRootUnavailable : racine indisponible, rien n'est marqué ; l'appelant
	// en rend compte par le bandeau.
	MissingRootUnavailable
)

const (
	// mediaStableAfter : âge minimal d'un fichier pour qu'il soit haché, sous
	// SkipFresh (un fichier en cours d'écriture changerait d'empreinte).
	mediaStableAfter = time.Minute
	// bulkMinActive : nombre de lignes actives à partir duquel une disparition de
	// la moitié ou plus de ces lignes est tenue pour suspecte.
	bulkMinActive = 20
)

// ReconcileInput décrit une passe de réconciliation des médias d'un joueur avec
// les fichiers vus sur le disque.
type ReconcileInput struct {
	PlayerSlug string
	// Files : chemins absolus des médias vus sur le disque, déjà filtrés des
	// autres titres.
	Files  []string
	Store  MediaPathStore
	Policy MissingPolicy
	// SkipFresh : ne pas hacher les fichiers modifiés depuis moins de
	// mediaStableAfter.
	SkipFresh bool
	// AllowBulk : l'utilisateur a confirmé une disparition massive.
	AllowBulk bool
	// Now : instant de référence de la fraîcheur ; zéro = time.Now().
	Now time.Time
	// Hash : empreinte d'un fichier ; nil = HashFile. Injecté par les tests.
	Hash func(string) (string, error)
}

// ReconcileResult rend ce que la passe a vu et fait.
type ReconcileResult struct {
	// Hashes : empreinte des fichiers lus ou reconnus sans relecture, par pathKey.
	Hashes map[string]string
	// Fresh : fichiers laissés de côté parce que trop récents, par pathKey.
	Fresh map[string]bool

	Relinked, Converted, Backfilled, MarkedMissing, Restored, Reverted, SourcesCleared int
	// BulkPending : nombre de médias dont le marquage est suspendu par le
	// garde-fou ; 0 sinon.
	BulkPending int

	// writes : nombre d'écritures tentées dans la transaction (0 = passe sans effet).
	writes int
}

// pathKeyOn rend la clé de comparaison d'un chemin : nettoyé, et sans casse sous
// Windows dont le système de fichiers l'ignore.
func pathKeyOn(goos, p string) string {
	p = filepath.Clean(p)
	if goos == "windows" {
		return strings.ToLower(p)
	}
	return p
}

// pathKey : pathKeyOn pour le système courant.
func pathKey(p string) string { return pathKeyOn(runtime.GOOS, p) }

// diskFile : un média vu sur le disque, avec ce qu'un os.Stat en dit.
type diskFile struct {
	abs, key string
	size     int64
	mtimeNS  int64
	fresh    bool
	// hash vide : fichier frais, ou lecture en échec.
	hash string
}

// mediaRow : une ligne media_files du joueur et son état face au disque.
type mediaRow struct {
	id                                                        int64
	filePath, sourcePath, fileName, fileHash, hlsPath, status string
	sourceSize, sourceMtime                                   sql.NullInt64
	fileOK, srcOK, present, lost, relocated                   bool
}

// reconciler porte l'état d'une passe : disque, lignes, écritures prévues.
type reconciler struct {
	ctx     context.Context
	db      *sql.DB
	in      ReconcileInput
	now     time.Time
	hash    func(string) (string, error)
	files   map[string]*diskFile
	order   []string
	rows    []*mediaRow
	claimed map[string]bool
	byHash  map[string][]*diskFile
	byStem  map[string][]*diskFile
	ops     []reconcileOp
	res     ReconcileResult
}

// ReconcileMediaWithDisk aligne les lignes media_files d'un joueur sur ce que le
// disque contient : l'identité d'un média est l'empreinte de son contenu, le chemin
// est un attribut qui change. Elle relocalise une ligne dont le fichier a été
// renommé ou déplacé (même empreinte) ou converti de format (même dossier, même
// nom sans extension, candidat unique), retrouve le fichier d'un média introuvable,
// marque introuvable ce qui a disparu (sous la politique et le garde-fou), garde
// visible une vidéo convertie dont seul l'original a disparu, et renseigne
// l'original (chemin, taille, date) de chaque ligne.
//
// Aucune ligne supprimée n'est lue ni touchée. file_stem n'est jamais écrit. Les
// écritures tiennent dans une seule transaction, et une passe sans changement n'en
// ouvre aucune. Idempotente.
func ReconcileMediaWithDisk(ctx context.Context, db *sql.DB, in ReconcileInput) (ReconcileResult, error) {
	r := newReconciler(ctx, db, in)
	r.takeInventory()
	if err := r.loadRows(); err != nil {
		return r.res, err
	}
	if err := r.hashFiles(); err != nil {
		return r.res, err
	}
	r.assessRows()
	candidates := r.planRelocations()
	r.planComplements()
	r.planMarking(candidates)
	if err := r.apply(); err != nil {
		return r.res, err
	}
	return r.res, nil
}

func newReconciler(ctx context.Context, db *sql.DB, in ReconcileInput) *reconciler {
	r := &reconciler{
		ctx: ctx, db: db, in: in, now: in.Now, hash: in.Hash,
		files:   map[string]*diskFile{},
		claimed: map[string]bool{},
		res:     ReconcileResult{Hashes: map[string]string{}, Fresh: map[string]bool{}},
	}
	if r.now.IsZero() {
		r.now = time.Now()
	}
	if r.hash == nil {
		r.hash = HashFile
	}
	return r
}

// takeInventory (étape 1) relève taille et date de chaque fichier. Un fichier qui
// ne se laisse pas lire est ignoré, avec un avertissement.
func (r *reconciler) takeInventory() {
	for _, p := range r.in.Files {
		key := pathKey(p)
		if _, dup := r.files[key]; dup {
			continue
		}
		st, err := os.Stat(p)
		if err != nil {
			slog.WarnContext(r.ctx, "reconcile: fichier illisible, ignoré", "path", p, "err", err)
			continue
		}
		r.files[key] = &diskFile{
			abs: p, key: key, size: st.Size(), mtimeNS: st.ModTime().UnixNano(),
			fresh: r.in.SkipFresh && r.now.Sub(st.ModTime()) < mediaStableAfter,
		}
		r.order = append(r.order, key)
	}
	sort.Strings(r.order)
}

// loadRows (étape 2) lit les lignes du joueur qui ne sont pas supprimées, les
// introuvables comprises.
func (r *reconciler) loadRows() error {
	rows, err := r.db.QueryContext(r.ctx, `
		SELECT id, file_path, source_path, source_size, source_mtime_ns,
		       file_name, file_hash, hls_path, status
		FROM media_files
		WHERE player_slug = ? AND `+platform_duckdb.MediaNotDeletedPredicate("")+`
		ORDER BY id`, r.in.PlayerSlug)
	if err != nil {
		return fmt.Errorf("reconcile: lecture des lignes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m mediaRow
		var fp, sp, fn, fh, hp, st sql.NullString
		if err := rows.Scan(&m.id, &fp, &sp, &m.sourceSize, &m.sourceMtime, &fn, &fh, &hp, &st); err != nil {
			return fmt.Errorf("reconcile: lecture d'une ligne: %w", err)
		}
		m.filePath, m.sourcePath, m.fileName = fp.String, sp.String, fn.String
		m.fileHash, m.hlsPath, m.status = fh.String, hp.String, st.String
		r.rows = append(r.rows, &m)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reconcile: lecture des lignes: %w", err)
	}
	return nil
}

// hashFiles (étape 3) donne une empreinte à chaque fichier non frais : celle de la
// ligne qui le référence comme original quand taille et date n'ont pas bougé, sinon
// une lecture. Une lecture en échec laisse le fichier sans empreinte : il reste vu
// du disque mais ne peut cibler aucune relocalisation.
func (r *reconciler) hashFiles() error {
	bySource := make(map[string]*mediaRow)
	for _, row := range r.rows {
		if row.sourcePath != "" {
			bySource[pathKey(r.in.Store.ToAbs(row.sourcePath))] = row
		}
	}
	for _, key := range r.order {
		f := r.files[key]
		if f.fresh {
			r.res.Fresh[key] = true
			continue
		}
		if err := r.ctx.Err(); err != nil {
			return err
		}
		if row, ok := bySource[key]; ok && unchangedSince(row, f) {
			f.hash = row.fileHash
		} else {
			h, err := r.hash(f.abs)
			if err != nil {
				slog.WarnContext(r.ctx, "reconcile: empreinte impossible, fichier ignoré",
					"path", f.abs, "err", err)
				continue
			}
			f.hash = h
		}
		r.res.Hashes[key] = f.hash
	}
	return nil
}

// unchangedSince : la ligne a déjà vu ce fichier, avec cette taille, cette date et
// une empreinte.
func unchangedSince(row *mediaRow, f *diskFile) bool {
	return row.fileHash != "" &&
		row.sourceSize.Valid && row.sourceSize.Int64 == f.size &&
		row.sourceMtime.Valid && row.sourceMtime.Int64 == f.mtimeNS
}

// storedExists : le chemin stocké existe sur le disque (inventaire, sinon os.Stat).
func (r *reconciler) storedExists(stored string) bool {
	if stored == "" {
		return false
	}
	abs := r.in.Store.ToAbs(stored)
	if _, ok := r.files[pathKey(abs)]; ok {
		return true
	}
	_, err := os.Stat(abs)
	return err == nil
}

// assessRows (étape 4) décide de la présence de chaque ligne et réclame les
// fichiers qu'elle référence. Une vidéo convertie est présente si sa version
// convertie ou son original existe ; l'original est perdu quand source_path est
// renseigné et n'existe plus.
func (r *reconciler) assessRows() {
	for _, row := range r.rows {
		row.fileOK = r.storedExists(row.filePath)
		row.srcOK = row.sourcePath != "" && r.storedExists(row.sourcePath)
		row.lost = row.sourcePath != "" && !row.srcOK
		switch {
		case row.hlsPath != "":
			row.present = row.fileOK || row.srcOK
		case row.sourcePath != "":
			row.present = row.srcOK
		default:
			row.present = row.fileOK
		}
		if row.fileOK {
			r.claimed[pathKey(r.in.Store.ToAbs(row.filePath))] = true
		}
		if row.srcOK {
			r.claimed[pathKey(r.in.Store.ToAbs(row.sourcePath))] = true
		}
	}
}
