package replaybuild

// artifact_meta.go — L'EN-TETE D'UN ARTEFACT RANGE, LU SANS OUVRIR L'ARTEFACT.
//
// Un artefact de rejeu pese de 1 a 2 Mo (2 000 rejeux : 2 a 4 Go) et son JSON ne permet pas de lire
// seulement ses marques de version sans l'ouvrir en entier. Le decompte des rejeux en retard
// (`internal/replayregen`) a besoin de ces marques pour CHAQUE artefact : il lit donc un petit
// fichier voisin, `<short8>.meta`, ecrit par l'ecrivain d'artefact ([writeArtifactBytes]) a chaque
// ecriture reelle.
//
// CE QUE PORTE LE FICHIER : la version de schema, les revisions de couche, la taille de l'artefact
// qu'il decrit, et la VERSION DE L'APPLICATION qui l'a produit (absente = inconnue, jamais « trop
// recente »).
//
// QUAND IL FAIT FOI : il existe, il n'est pas plus ancien que l'artefact, et la taille qu'il declare
// est celle du fichier. Sinon (artefact d'avant ce fichier, ecriture interrompue entre les deux
// fichiers, artefact remplace par un outil qui ne l'ecrit pas) l'artefact est LU UNE FOIS et le
// fichier est (re)ecrit. Le decompte d'un parc ancien paie donc ses lectures une seule fois.

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/atomicfile"
	"levelup/go-api/internal/replayartifact"
)

// Compteurs de l'en-tete (expvar).
const (
	// CompteurOctetsLusEntetes : octets reellement lus pour juger des artefacts par [ArtifactHeader] — les
	// fichiers .meta, plus l'artefact entier a chaque lecture entiere.
	CompteurOctetsLusEntetes = "replay_artifact_header_bytes_read_total"
	// CompteurArtefactsLusEnEntier : artefacts lus en entier faute de fichier .meta exploitable.
	CompteurArtefactsLusEnEntier = "replay_artifact_header_full_reads_total"
	// CompteurMetaNonEcrit : fichiers .meta que l'ecriture a refuses.
	CompteurMetaNonEcrit = "replay_artifact_meta_write_failed_total"
)

// SuffixeMeta : l'extension du fichier d'en-tete, posee a la place de l'extension d'artefact (`.json.gz` comme `.json` : le
// nom `{short8}.meta` ne depend pas de la forme de l'artefact).
const SuffixeMeta = ".meta"

// MetaPath rend le chemin du fichier d'en-tete d'un artefact. Une seule source du nom : l'ecrivain, le
// lecteur et la purge passent par ici.
func MetaPath(artifactPath string) string {
	return title.ReplayArtifactStem(artifactPath) + SuffixeMeta
}

// artifactMeta : le contenu du fichier d'en-tete.
type artifactMeta struct {
	MatchID       string            `json:"matchId,omitempty"`
	SchemaVersion int               `json:"schemaVersion"`
	Layers        map[string]string `json:"layers,omitempty"`
	// Bytes : la TAILLE STOCKEE de l'artefact decrit (celle du fichier sur disque, compresse le cas echeant, cf.
	// artifact_gate.go) ; un artefact de taille differente n'est pas celui-la.
	Bytes int64 `json:"bytes"`
	// AppVersion : la version de l'application du processus qui a ecrit l'artefact ; vide = inconnue.
	AppVersion string `json:"appVersion,omitempty"`
}

// Header : ce que le decompte lit d'un artefact — ses marques et la version de l'application qui l'a produit.
type Header struct {
	Digest
	// AppVersion : la version de l'application qui a ecrit l'artefact ; vide = inconnue (artefact d'avant le
	// fichier d'en-tete, ou ecrit par un outil sans version).
	AppVersion string
}

// producteur : la version de l'application du processus courant, posee au demarrage du serveur.
var producteur atomic.Value // string

// SetAppVersion pose la version de l'application que le processus inscrit dans les fichiers d'en-tete qu'il
// ecrit. Appelee une fois au demarrage du serveur ; les outils qui ne l'appellent pas ecrivent « inconnue ».
func SetAppVersion(v string) { producteur.Store(strings.TrimSpace(v)) }

// AppVersion rend la version posee par [SetAppVersion] ("" si aucune).
func AppVersion() string {
	v, _ := producteur.Load().(string)
	return v
}

// ArtifactHeader rend les marques d'un artefact rangé en lisant le plus petit fichier possible : le fichier
// d'en-tete quand il fait foi, l'artefact entier une fois sinon (et le fichier d'en-tete est alors ecrit).
// ok=false si l'artefact est absent ou illisible, comme [ArtifactDigest].
func ArtifactHeader(ctx context.Context, path string) (Header, bool) {
	f, err := replayartifact.Locate(path)
	if err != nil {
		return Header{}, false
	}
	st := f.Info
	if h, ok := lireEntete(f.Path, st); ok {
		return h, true
	}
	observability.IncCounter(CompteurArtefactsLusEnEntier)
	observability.AddInt(CompteurOctetsLusEntetes, st.Size())
	d, ok := ArtifactDigest(f.Path)
	if !ok {
		return Header{}, false
	}
	ecrireEntete(ctx, f.Path, d, "")
	return Header{Digest: d}, true
}

// lireEntete lit le fichier d'en-tete s'il fait foi pour l'artefact decrit par st.
func lireEntete(path string, st os.FileInfo) (Header, bool) {
	mp := MetaPath(path)
	mst, err := os.Stat(mp)
	if err != nil || mst.ModTime().Before(st.ModTime()) {
		return Header{}, false // absent, ou plus ancien que l'artefact : il decrit un autre document
	}
	raw, err := os.ReadFile(mp)
	if err != nil {
		return Header{}, false
	}
	observability.AddInt(CompteurOctetsLusEntetes, int64(len(raw)))
	var m artifactMeta
	if json.Unmarshal(raw, &m) != nil || m.Bytes != st.Size() || m.SchemaVersion == 0 {
		return Header{}, false
	}
	return Header{
		Digest:     Digest{MatchID: m.MatchID, SchemaVersion: m.SchemaVersion, Layers: m.Layers, Bytes: int(m.Bytes)},
		AppVersion: m.AppVersion,
	}, true
}

// metaDejaAvertie : un seul WARN par processus quand le fichier d'en-tete ne s'ecrit pas (dossier en lecture
// seule) ; le compteur dit combien.
var metaDejaAvertie atomic.Bool

// ecrireEntete ecrit le fichier d'en-tete d'un artefact qui vient d'etre ecrit ou lu. BEST-EFFORT SIGNALE : un
// echec n'empeche pas l'artefact d'etre servi (le decompte relira l artefact entier), mais il se compte et
// se journalise.
func ecrireEntete(ctx context.Context, artifactPath string, d Digest, appVersion string) {
	blob, err := json.Marshal(artifactMeta{
		MatchID: d.MatchID, SchemaVersion: d.SchemaVersion, Layers: d.Layers,
		Bytes: int64(d.Bytes), AppVersion: appVersion,
	})
	if err == nil {
		err = atomicfile.WriteFile(ctx, MetaPath(artifactPath), blob, 0o644)
	}
	if err == nil {
		return
	}
	observability.IncCounter(CompteurMetaNonEcrit)
	if metaDejaAvertie.CompareAndSwap(false, true) {
		slog.WarnContext(ctx, "replaybuild: fichier d'en-tete d'artefact non ecrit — le decompte relira l'artefact entier",
			"path", MetaPath(artifactPath), "err", err)
	}
}

// ProduitParUneVersionPlusRecente dit si l'artefact a ete produit par une version de l'application STRICTEMENT
// plus recente que `binaire` (comparaison semver sur major.minor.patch). Une version absente ou illisible de
// l'un des deux cotes rend faux : on ne declare jamais « plus recent » ce qu'on ne sait pas comparer.
func (h Header) ProduitParUneVersionPlusRecente(binaire string) bool {
	return versionPlusRecente(h.AppVersion, binaire)
}

// RemoveMeta retire le fichier d'en-tete d'un artefact purge ; l'absence n'est pas une erreur.
func RemoveMeta(artifactPath string) error {
	if err := os.Remove(MetaPath(artifactPath)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
