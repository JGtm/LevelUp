package ops

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/domain"
)

// opKind : genre d'écriture que la passe de réconciliation prévoit.
type opKind int

const (
	opRelocate      opKind = iota // la ligne suit un fichier retrouvé ailleurs ou converti
	opAttachSource                // renseigne source_path, taille et date
	opRefreshSource               // taille, date et empreinte d'un original modifié
	opClearSource                 // l'original a disparu, la version convertie reste
	opRevert                      // retour en lecture directe sur l'original
	opRestore                     // un fichier est revenu : status repasse à NULL
	opMarkMissing                 // plus aucun fichier : status = missing
)

// reconcileOp : une écriture prévue, appliquée en transaction par apply.
type reconcileOp struct {
	kind opKind
	id   int64
	// path : source_path posé (opRelocate, opAttachSource) ou file_path cible d'un
	// retour en lecture directe (opRevert).
	path        string
	size, mtime int64
	name, ext   string
	hash        string
	hasHLS      bool
	converted   bool
	// oldFile : file_path avant l'écriture, d'où partent les likes.
	oldFile string
}

// stemOf rend un nom de fichier sans son extension.
func stemOf(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

func nullInt(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }

// indexFiles prépare les deux recherches de l'étape 5 : fichiers par empreinte, et
// par couple (dossier, nom sans extension). Seuls les fichiers hachés y figurent.
func (r *reconciler) indexFiles() (byHash, byStem map[string][]*diskFile) {
	byHash = map[string][]*diskFile{}
	byStem = map[string][]*diskFile{}
	for _, key := range r.order {
		f := r.files[key]
		if f.hash == "" {
			continue
		}
		byHash[f.hash] = append(byHash[f.hash], f)
		sk := stemKey(filepath.Dir(f.abs), stemOf(filepath.Base(f.abs)))
		byStem[sk] = append(byStem[sk], f)
	}
	return byHash, byStem
}

// stemKey : clé (dossier, nom sans extension) de la recherche de changement de format.
func stemKey(dir, stem string) string {
	return pathKey(dir) + "\x00" + stem
}

// statDiskFile rend taille et date d'un fichier hors inventaire.
func statDiskFile(abs string) (diskFile, bool) {
	st, err := os.Stat(abs)
	if err != nil {
		return diskFile{}, false
	}
	return diskFile{abs: abs, key: pathKey(abs), size: st.Size(), mtimeNS: st.ModTime().UnixNano()}, true
}

// firstUnclaimed rend le premier fichier que aucune ligne ne réclame.
func (r *reconciler) firstUnclaimed(files []*diskFile) *diskFile {
	for _, f := range files {
		if !r.claimed[f.key] {
			return f
		}
	}
	return nil
}

// planRelocations (étape 5) traite les lignes absentes et les lignes à l'original
// perdu, dans cet ordre : fichier non réclamé de même empreinte, changement de
// format, original perdu d'une version convertie qui reste, sinon candidate au
// marquage. Rend les candidates.
func (r *reconciler) planRelocations() []*mediaRow {
	r.byHash, r.byStem = r.indexFiles()
	var candidates []*mediaRow
	for _, row := range r.rows {
		if row.present && !row.lost {
			continue
		}
		switch {
		case r.relocateByHash(row):
		case r.relocateByFormat(row):
		case row.present:
			r.ops = append(r.ops, reconcileOp{kind: opClearSource, id: row.id})
			row.sourcePath, row.srcOK = "", false
			row.sourceSize, row.sourceMtime = sql.NullInt64{}, sql.NullInt64{}
		case row.status != domain.MediaStatusMissing:
			candidates = append(candidates, row)
		}
	}
	return candidates
}

// relocateByHash : la ligne suit un fichier non réclamé de même empreinte.
func (r *reconciler) relocateByHash(row *mediaRow) bool {
	if row.fileHash == "" {
		return false
	}
	f := r.firstUnclaimed(r.byHash[row.fileHash])
	if f == nil {
		return false
	}
	r.relocate(row, f, false)
	return true
}

// relocateByFormat : changement de format, c'est-à-dire un seul fichier non
// réclamé de même nom sans extension dans le dossier du chemin de référence.
// Plusieurs candidats : avertissement, aucune relocalisation.
func (r *reconciler) relocateByFormat(row *mediaRow) bool {
	ref := row.sourcePath
	if ref == "" {
		ref = row.filePath
	}
	stem := stemOf(row.fileName)
	if ref == "" || stem == "" {
		return false
	}
	var cands []*diskFile
	for _, f := range r.byStem[stemKey(filepath.Dir(r.in.Store.ToAbs(ref)), stem)] {
		if !r.claimed[f.key] {
			cands = append(cands, f)
		}
	}
	switch len(cands) {
	case 0:
		return false
	case 1:
		r.relocate(row, cands[0], true)
		return true
	}
	slog.WarnContext(r.ctx, "reconcile: plusieurs candidats de même nom, aucune relocalisation",
		"player", r.in.PlayerSlug, "id", row.id, "name", row.fileName, "candidates", len(cands))
	return false
}

// relocate prévoit l'écriture qui fait suivre à la ligne le fichier f, et met la
// ligne à jour en mémoire pour la suite de la passe.
func (r *reconciler) relocate(row *mediaRow, f *diskFile, converted bool) {
	stored := r.in.Store.ToRel(f.abs, r.in.PlayerSlug)
	if stored == "" {
		stored = f.abs
	}
	r.claimed[f.key] = true
	op := reconcileOp{
		kind: opRelocate, id: row.id, path: stored, size: f.size, mtime: f.mtimeNS,
		name: filepath.Base(f.abs), ext: strings.ToLower(filepath.Ext(f.abs)), hash: f.hash,
		hasHLS: row.hlsPath != "", converted: converted, oldFile: row.filePath,
	}
	r.ops = append(r.ops, op)
	row.sourcePath, row.srcOK = stored, true
	row.sourceSize, row.sourceMtime = nullInt(f.size), nullInt(f.mtimeNS)
	row.fileName, row.fileHash = op.name, f.hash
	row.present, row.lost, row.relocated = true, false, true
	if !op.hasHLS {
		row.filePath, row.fileOK = stored, true
	}
}

// planComplements (étape 6) complète les lignes présentes : original renseigné,
// original modifié, retour en lecture directe, fichier revenu.
func (r *reconciler) planComplements() {
	for _, row := range r.rows {
		if !row.present {
			continue
		}
		switch {
		case row.relocated:
		case row.sourcePath == "" && row.hlsPath == "":
			r.planBackfill(row)
		case row.sourcePath == "":
			r.planAttachSource(row)
		default:
			r.planRefresh(row)
		}
		r.planRevert(row)
		if !row.relocated && row.status == domain.MediaStatusMissing {
			r.ops = append(r.ops, reconcileOp{kind: opRestore, id: row.id})
		}
	}
}

// statStored rend taille et date d'un chemin stocké (inventaire, sinon os.Stat).
func (r *reconciler) statStored(stored string) (diskFile, bool) {
	abs := r.in.Store.ToAbs(stored)
	if f, ok := r.files[pathKey(abs)]; ok {
		return *f, true
	}
	return statDiskFile(abs)
}

// planBackfill : une ligne sans original dont le fichier est celui de l'app, qui
// devient son original. Un fichier encore frais attend le passage suivant.
func (r *reconciler) planBackfill(row *mediaRow) {
	f, ok := r.statStored(row.filePath)
	if !ok || f.fresh {
		return
	}
	r.ops = append(r.ops, reconcileOp{
		kind: opAttachSource, id: row.id, path: row.filePath, size: f.size, mtime: f.mtimeNS,
	})
	row.sourcePath = row.filePath
	row.sourceSize, row.sourceMtime = nullInt(f.size), nullInt(f.mtimeNS)
}

// planAttachSource : une vidéo convertie sans original dont l'empreinte se
// retrouve sur un fichier non réclamé. Aucun fichier : rien (cas du serveur).
func (r *reconciler) planAttachSource(row *mediaRow) {
	if row.fileHash == "" {
		return
	}
	f := r.firstUnclaimed(r.byHash[row.fileHash])
	if f == nil {
		return
	}
	stored := r.in.Store.ToRel(f.abs, r.in.PlayerSlug)
	if stored == "" {
		stored = f.abs
	}
	r.claimed[f.key] = true
	r.ops = append(r.ops, reconcileOp{
		kind: opAttachSource, id: row.id, path: stored, size: f.size, mtime: f.mtimeNS,
	})
	row.sourcePath, row.srcOK = stored, true
	row.sourceSize, row.sourceMtime = nullInt(f.size), nullInt(f.mtimeNS)
}

// planRefresh : l'original est au même chemin mais sa taille ou sa date ont
// changé ; l'empreinte lue à l'étape 3 le remplace. Un fichier frais ou sans
// empreinte attend.
func (r *reconciler) planRefresh(row *mediaRow) {
	f, ok := r.files[pathKey(r.in.Store.ToAbs(row.sourcePath))]
	if !ok || f.fresh || f.hash == "" {
		return
	}
	if row.sourceSize.Valid && row.sourceSize.Int64 == f.size &&
		row.sourceMtime.Valid && row.sourceMtime.Int64 == f.mtimeNS {
		return
	}
	r.ops = append(r.ops, reconcileOp{
		kind: opRefreshSource, id: row.id, size: f.size, mtime: f.mtimeNS, hash: f.hash,
	})
	row.sourceSize, row.sourceMtime, row.fileHash = nullInt(f.size), nullInt(f.mtimeNS), f.hash
}

// planRevert : une vidéo convertie dont la version de l'app n'existe plus mais
// dont l'original est là repasse en lecture directe ; le balayage la reconvertit.
func (r *reconciler) planRevert(row *mediaRow) {
	if row.hlsPath == "" || row.sourcePath == "" || row.fileOK {
		return
	}
	r.ops = append(r.ops, reconcileOp{
		kind: opRevert, id: row.id, path: row.sourcePath, oldFile: row.filePath,
	})
	row.filePath, row.hlsPath, row.fileOK = row.sourcePath, "", true
}

// planMarking (étape 7) décide du marquage des candidates. Hors MissingMark, rien
// n'est marqué et rien n'est suspendu. Sous MissingMark, le garde-fou suspend le
// marquage quand au moins bulkMinActive lignes sont actives et que la moitié ou
// plus deviendrait introuvable : une disparition de cette ampleur est plus
// probablement un dossier déplacé qu'un ménage.
func (r *reconciler) planMarking(candidates []*mediaRow) {
	if r.in.Policy != MissingMark || len(candidates) == 0 {
		return
	}
	actives := 0
	for _, row := range r.rows {
		if row.status != domain.MediaStatusMissing {
			actives++
		}
	}
	if !r.in.AllowBulk && actives >= bulkMinActive && len(candidates)*2 >= actives {
		r.res.BulkPending = len(candidates)
		slog.WarnContext(r.ctx, "reconcile: disparition massive, marquage suspendu",
			"player", r.in.PlayerSlug, "candidates", len(candidates), "actives", actives)
		return
	}
	for _, row := range candidates {
		r.ops = append(r.ops, reconcileOp{kind: opMarkMissing, id: row.id})
	}
}
