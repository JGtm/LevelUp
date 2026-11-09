package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// thumbRow : une vidéo visible du joueur et ce qu'on sait de sa miniature.
type thumbRow struct {
	id       int64
	fileHash string
	source   string // chemin stocké de l'original, ou du fichier d'une vidéo non convertie
	thumb    string // chemin stocké de la miniature liée, vide si aucune
	hasThumb bool
}

// thumbEnsurer porte l'état d'un passage : dossier des miniatures, ce que les
// fiches référencent déjà, ce que le dossier contient.
type thumbEnsurer struct {
	ctx       context.Context
	db        *sql.DB
	opts      MediaIndexOptions
	store     MediaPathStore
	thumbsDir string
	refs      map[string]bool     // pathKey de chaque miniature référencée
	onDisk    map[string][]string // nom sans extension et sans suffixe d'empreinte -> fichiers
	generated int
	errs      []string
}

// EnsureVideoThumbnails donne une miniature à chaque vidéo visible du joueur, à partir
// de son original (source_path), sinon de son fichier quand elle n'est pas convertie.
//   - miniature liée et présente : rien ;
//   - miniature liée mais absente, original présent : regénérée au même chemin ;
//   - aucune miniature liée : on lie celle du dossier thumbs/ qui porte le nom de
//     l'original (suffixe d'empreinte retiré) quand aucune autre fiche ne la
//     référence, sinon on en génère une, nommée {nom}.webp pour un original à la
//     racine quand ce nom est libre, {nom}_{12 premiers caractères de l'empreinte}.webp
//     sinon : deux clips de même nom ont chacun leur miniature.
//
// Une fiche dont l'original est absent est laissée telle quelle. Rend le nombre de
// miniatures générées ou liées, et les erreurs de génération.
func EnsureVideoThumbnails(ctx context.Context, db *sql.DB, opts MediaIndexOptions, store MediaPathStore) (int, []string) {
	e := &thumbEnsurer{
		ctx: ctx, db: db, opts: opts, store: store,
		thumbsDir: filepath.Join(opts.CapturesDir, "thumbs"),
	}
	rows, err := e.loadRows()
	if err != nil {
		slog.ErrorContext(ctx, "EnsureVideoThumbnails: lecture des fiches échouée",
			"player", opts.Gamertag, "err", err)
		return 0, []string{fmt.Sprintf("lecture des fiches: %v", err)}
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := e.loadRefs(); err != nil {
		slog.ErrorContext(ctx, "EnsureVideoThumbnails: lecture des miniatures liées échouée",
			"player", opts.Gamertag, "err", err)
		return 0, []string{fmt.Sprintf("lecture des miniatures liées: %v", err)}
	}
	e.listThumbsDir()
	for _, r := range rows {
		if ctx.Err() != nil {
			break
		}
		e.ensure(r)
	}
	return e.generated, e.errs
}

// loadRows lit les vidéos visibles du joueur dont un fichier peut donner la miniature.
func (e *thumbEnsurer) loadRows() ([]*thumbRow, error) {
	rs, err := e.db.QueryContext(e.ctx, `
		SELECT id, file_path, source_path, hls_path, thumbnail_path, file_hash
		FROM media_files
		WHERE player_slug = ? AND kind = 'video' AND `+platform_duckdb.MediaVisiblePredicate("")+`
		ORDER BY id`, e.opts.Gamertag)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []*thumbRow
	for rs.Next() {
		var id int64
		var fp, sp, hp, tp, fh sql.NullString
		if err := rs.Scan(&id, &fp, &sp, &hp, &tp, &fh); err != nil {
			return nil, err
		}
		r := &thumbRow{id: id, fileHash: fh.String, thumb: tp.String, hasThumb: tp.String != ""}
		r.source = sp.String
		if r.source == "" && hp.String == "" {
			r.source = fp.String
		}
		if r.source == "" {
			continue
		}
		out = append(out, r)
	}
	return out, rs.Err()
}

// loadRefs relève les miniatures déjà liées à une fiche, supprimées exceptées.
func (e *thumbEnsurer) loadRefs() error {
	rs, err := e.db.QueryContext(e.ctx, `
		SELECT thumbnail_path FROM media_files
		WHERE thumbnail_path IS NOT NULL AND `+platform_duckdb.MediaNotDeletedPredicate(""))
	if err != nil {
		return err
	}
	defer rs.Close()
	e.refs = map[string]bool{}
	for rs.Next() {
		var tp string
		if err := rs.Scan(&tp); err != nil {
			return err
		}
		e.refs[pathKey(e.store.ToAbs(tp))] = true
	}
	return rs.Err()
}

// listThumbsDir indexe les miniatures du dossier (webp, gif legacy) par nom d'original :
// leur nom sans extension, et ce nom privé du suffixe d'empreinte.
func (e *thumbEnsurer) listThumbsDir() {
	e.onDisk = map[string][]string{}
	entries, err := os.ReadDir(e.thumbsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.WarnContext(e.ctx, "EnsureVideoThumbnails: dossier des miniatures illisible",
				"dir", e.thumbsDir, "err", err)
		}
		return
	}
	for _, en := range entries {
		ext := strings.ToLower(filepath.Ext(en.Name()))
		if en.IsDir() || (ext != ".webp" && ext != ".gif") {
			continue
		}
		stem := strings.TrimSuffix(en.Name(), filepath.Ext(en.Name()))
		e.onDisk[stem] = append(e.onDisk[stem], en.Name())
		if stripped := thumbHashSuffixRe.ReplaceAllString(stem, ""); stripped != stem {
			e.onDisk[stripped] = append(e.onDisk[stripped], en.Name())
		}
	}
	for k := range e.onDisk {
		sort.Strings(e.onDisk[k])
	}
}

// ensure traite une vidéo.
func (e *thumbEnsurer) ensure(r *thumbRow) {
	srcAbs := e.store.ToAbs(r.source)
	if _, err := os.Stat(srcAbs); err != nil {
		return // original absent : rien à faire de plus
	}
	if r.hasThumb {
		thumbAbs := e.store.ToAbs(r.thumb)
		if _, err := os.Stat(thumbAbs); err == nil {
			return
		}
		e.generate(r, srcAbs, thumbAbs, "")
		return
	}
	name := stemOf(filepath.Base(srcAbs))
	for _, f := range e.onDisk[name] {
		abs := filepath.Join(e.thumbsDir, f)
		if e.refs[pathKey(abs)] {
			continue
		}
		e.link(r, abs)
		return
	}
	atRoot := pathKey(filepath.Dir(srcAbs)) == pathKey(e.opts.CapturesDir)
	target := filepath.Join(e.thumbsDir, name+".webp")
	if !atRoot || !e.free(target) {
		target = filepath.Join(e.thumbsDir, name+"_"+hashPrefix(r)+".webp")
	}
	if !e.free(target) {
		e.fail(srcAbs, fmt.Errorf("nom de miniature déjà pris: %s", filepath.Base(target)))
		return
	}
	e.generate(r, srcAbs, target, e.storedThumb(target))
}

// hashPrefix : 12 premiers caractères de l'empreinte du contenu, à défaut l'identifiant
// de la fiche.
func hashPrefix(r *thumbRow) string {
	if r.fileHash == "" {
		return fmt.Sprintf("id%d", r.id)
	}
	if len(r.fileHash) > 12 {
		return r.fileHash[:12]
	}
	return r.fileHash
}

// free : aucun fichier ni aucune fiche ne porte ce nom de miniature.
func (e *thumbEnsurer) free(abs string) bool {
	if e.refs[pathKey(abs)] {
		return false
	}
	_, err := os.Stat(abs)
	return os.IsNotExist(err)
}

// storedThumb rend le chemin de miniature tel que la base le stocke.
func (e *thumbEnsurer) storedThumb(abs string) string {
	if rel := e.store.ToRel(abs, e.opts.Gamertag); rel != "" {
		return rel
	}
	return abs
}

// link attache à la fiche une miniature déjà présente dans le dossier.
func (e *thumbEnsurer) link(r *thumbRow, abs string) {
	if err := e.setThumb(r.id, e.storedThumb(abs)); err != nil {
		e.fail(abs, err)
		return
	}
	e.refs[pathKey(abs)] = true
	e.generated++
}

// generate produit la miniature de srcAbs en thumbAbs. stored non vide : la fiche n'en
// avait pas, le chemin produit lui est lié ; vide : la fiche garde le chemin qu'elle a.
func (e *thumbEnsurer) generate(r *thumbRow, srcAbs, thumbAbs, stored string) {
	if err := os.MkdirAll(filepath.Dir(thumbAbs), 0o755); err != nil {
		e.fail(srcAbs, err)
		return
	}
	if err := generateAnimatedThumbnail(e.ctx, srcAbs, thumbAbs); err != nil {
		e.fail(srcAbs, err)
		return
	}
	if stored != "" {
		if err := e.setThumb(r.id, stored); err != nil {
			e.fail(srcAbs, err)
			return
		}
	}
	e.refs[pathKey(thumbAbs)] = true
	e.generated++
}

// setThumb lie une miniature à une fiche, désignée par son identifiant.
func (e *thumbEnsurer) setThumb(id int64, stored string) error {
	_, err := e.db.ExecContext(e.ctx, `UPDATE media_files SET thumbnail_path = ? WHERE id = ?`, stored, id)
	return err
}

// fail journalise un échec de miniature et le rend à l'appelant.
func (e *thumbEnsurer) fail(file string, err error) {
	slog.With("module", logModuleMedia).WarnContext(e.ctx, "EnsureVideoThumbnails: miniature échouée",
		"file", filepath.Base(file), "err", err)
	e.errs = append(e.errs, fmt.Sprintf("%s: %v", filepath.Base(file), err))
}
