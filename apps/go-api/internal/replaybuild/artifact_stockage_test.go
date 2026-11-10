package replaybuild

// artifact_stockage_test.go — LE STOCKAGE COMPRESSÉ VU DU PUITS : l'écriture range un `.json.gz`, l'en-tête
// `.meta` et la marque de dérivation portent la TAILLE STOCKÉE et gardent leurs noms. La porte elle-même
// (formes, parc mixte, listage) est éprouvée dans `internal/replayartifact`.

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replayartifact"
)

const (
	courtTest = "000d5950"
	matchTest = "000d5950-1234-4abc-9def-0123456789ab"
)

func cheminCanonique(dir string) string { return filepath.Join(dir, courtTest+title.ReplayArtifactExt) }
func cheminBrut(dir string) string {
	return filepath.Join(dir, courtTest+title.ReplayArtifactLegacyExt)
}

func gzipTest(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ecrireTest(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestPuits_RangeUnCompresseQuiSeRelitIdentique : le puits écrit sous le nom compressé, et la lecture rend,
// octet pour octet, le JSON déposé ; l'accusé annonce la taille stockée.
func TestPuits_RangeUnCompresseQuiSeRelitIdentique(t *testing.T) {
	dir := t.TempDir()
	blob := docJSON(t, matchTest, true)
	st, err := StoreArtifactAt(context.Background(), cheminCanonique(dir), title.DefaultSlug, matchTest, blob)
	if err != nil {
		t.Fatal(err)
	}
	if st.Path != cheminCanonique(dir) {
		t.Errorf("chemin rangé = %s", st.Path)
	}
	stocke, err := os.ReadFile(cheminCanonique(dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(stocke) < 2 || stocke[0] != 0x1f || stocke[1] != 0x8b || bytes.Equal(stocke, blob) {
		t.Fatal("le fichier rangé n'est pas du gzip")
	}
	lu, err := replayartifact.ReadBytes(cheminCanonique(dir))
	if err != nil || !bytes.Equal(lu, blob) {
		t.Fatalf("la lecture rend %d octets, le JSON déposé en compte %d (%v)", len(lu), len(blob), err)
	}
	if st.Bytes != len(stocke) {
		t.Errorf("l'accusé annonce %d octets, le fichier en compte %d : « bytes » est la taille stockée", st.Bytes, len(stocke))
	}
}

// TestPuits_RetireLeBrutAncien : réécrire un artefact qui existait en brut ne laisse pas deux fichiers.
func TestPuits_RetireLeBrutAncien(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminBrut(dir), []byte(`{"schemaVersion":1}`))
	if _, err := StoreArtifactAt(context.Background(), cheminCanonique(dir), title.DefaultSlug, matchTest,
		docJSON(t, matchTest, true)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cheminBrut(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("le .json ancien doit disparaître après l'écriture du .json.gz : %v", err)
	}
}

// TestPuits_EnteteBytesEgalTailleStockee : le fichier d'en-tête déclare la taille du fichier stocké, il garde le
// nom `{short8}.meta`, et il fait foi sans relire l'artefact.
func TestPuits_EnteteBytesEgalTailleStockee(t *testing.T) {
	dir := t.TempDir()
	if _, err := StoreArtifactAt(context.Background(), cheminCanonique(dir), title.DefaultSlug, matchTest,
		docJSON(t, matchTest, true)); err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(dir, courtTest+".meta")
	if MetaPath(cheminCanonique(dir)) != meta || MetaPath(cheminBrut(dir)) != meta {
		t.Errorf("MetaPath = %s / %s, attendu %s", MetaPath(cheminCanonique(dir)), MetaPath(cheminBrut(dir)), meta)
	}
	raw, err := os.ReadFile(meta)
	if err != nil {
		t.Fatal(err)
	}
	var m artifactMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(cheminCanonique(dir))
	if err != nil {
		t.Fatal(err)
	}
	if m.Bytes != st.Size() {
		t.Errorf("meta.bytes = %d, taille du fichier = %d", m.Bytes, st.Size())
	}
	avant := observability.LoadCounter(CompteurArtefactsLusEnEntier)
	h, ok := ArtifactHeader(context.Background(), cheminCanonique(dir))
	if !ok || h.SchemaVersion == 0 || h.Bytes != int(st.Size()) {
		t.Fatalf("ArtifactHeader = %+v, %v", h, ok)
	}
	if observability.LoadCounter(CompteurArtefactsLusEnEntier) != avant {
		t.Error("l'en-tête fait foi : l'artefact ne doit pas être relu")
	}
}

// TestPuits_EnteteAncienNeFaitPasFoi : un en-tête plus ancien que l'artefact stocké fait relire l'artefact.
func TestPuits_EnteteAncienNeFaitPasFoi(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	if _, err := StoreArtifactAt(ctx, cheminCanonique(dir), title.DefaultSlug, matchTest, docJSON(t, matchTest, true)); err != nil {
		t.Fatal(err)
	}
	vieux := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(dir, courtTest+".meta"), vieux, vieux); err != nil {
		t.Fatal(err)
	}
	avant := observability.LoadCounter(CompteurArtefactsLusEnEntier)
	if _, ok := ArtifactHeader(ctx, cheminCanonique(dir)); !ok {
		t.Fatal("en-tête illisible")
	}
	if observability.LoadCounter(CompteurArtefactsLusEnEntier) != avant+1 {
		t.Error("un en-tête plus ancien que l'artefact doit faire relire l'artefact")
	}
}

// TestPuits_DigestRendLaTailleStockee : le digest lu depuis le disque rend la taille du fichier, brut ou
// compressé.
func TestPuits_DigestRendLaTailleStockee(t *testing.T) {
	dir := t.TempDir()
	blob := docJSON(t, matchTest, true)
	ecrireTest(t, cheminBrut(dir), blob)
	d, ok := ArtifactDigest(cheminCanonique(dir))
	if !ok || d.Bytes != len(blob) {
		t.Fatalf("digest d'un brut : %+v, %v", d, ok)
	}
	ecrireTest(t, cheminCanonique(dir), gzipTest(t, blob))
	if err := os.Remove(cheminBrut(dir)); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(cheminCanonique(dir))
	d, ok = ArtifactDigest(cheminCanonique(dir))
	if !ok || d.Bytes != int(st.Size()) || d.MatchID != matchTest || d.SchemaVersion == 0 {
		t.Fatalf("digest d'un compressé : %+v, %v (taille stockée %d)", d, ok, st.Size())
	}
}

// TestPuits_DerivationsUpToDateSousCompression : la marque porte la taille stockée ; elle garde le nom
// `{short8}.derived.json` ; une réécriture de l'artefact l'invalide.
func TestPuits_DerivationsUpToDateSousCompression(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	if _, err := StoreArtifactAt(ctx, cheminCanonique(dir), title.DefaultSlug, matchTest, docJSON(t, matchTest, false)); err != nil {
		t.Fatal(err)
	}
	if got, want := DerivationsMarkPath(cheminCanonique(dir)), filepath.Join(dir, courtTest+".derived.json"); got != want {
		t.Fatalf("DerivationsMarkPath = %s, attendu %s", got, want)
	}
	if got, want := ArtefactDeLaMarque(DerivationsMarkPath(cheminCanonique(dir))), cheminCanonique(dir); got != want {
		t.Fatalf("ArtefactDeLaMarque = %s, attendu %s", got, want)
	}
	if DerivationsUpToDate(cheminCanonique(dir)) {
		t.Fatal("sans marque : pas dérivé")
	}
	c, err := replayartifact.Read(cheminCanonique(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteDerivationsMark(cheminCanonique(dir), 1, c.StoredBytes); err != nil {
		t.Fatal(err)
	}
	if !DerivationsUpToDate(cheminCanonique(dir)) {
		t.Fatal("marque à la taille stockée : à jour")
	}
	if _, err := StoreArtifactAt(ctx, cheminCanonique(dir), title.DefaultSlug, matchTest, docJSON(t, matchTest, true)); err != nil {
		t.Fatal(err)
	}
	if DerivationsUpToDate(cheminCanonique(dir)) {
		t.Fatal("l'artefact a été réécrit : les dérivés sont périmés")
	}
}
