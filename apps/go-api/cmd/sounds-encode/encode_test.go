package main

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/media"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// fakeEncoder écrit un contenu fixe et note ses appels ; failOn nomme un src qui échoue.
type fakeEncoder struct {
	encoded, checked []string
	failOn           string
	checkErr         error
}

func (f *fakeEncoder) Encode(_ context.Context, src, dst string) error {
	if filepath.Base(src) == f.failOn {
		return errors.New("encodage refusé")
	}
	f.encoded = append(f.encoded, filepath.Base(src))
	return os.WriteFile(dst, []byte("OggS"), 0o644)
}

func (f *fakeEncoder) Check(_ context.Context, src, _ string) error {
	f.checked = append(f.checked, filepath.Base(src))
	return f.checkErr
}

func writeFile(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func TestRunEncodeMemeArborescenceEtIdempotent(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	writeFile(t, filepath.Join(root, "halo_infinite", "a.wav"), now)
	writeFile(t, filepath.Join(root, "halo_infinite", "sub", "b.WAV"), now)
	writeFile(t, filepath.Join(root, "halo_infinite", "note.txt"), now)

	enc := &fakeEncoder{}
	rep, err := run(context.Background(), options{Src: root, Dst: root}, enc, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Encoded != 2 || rep.Skipped != 0 {
		t.Fatalf("premier passage : %+v", rep)
	}
	for _, p := range []string{"halo_infinite/a.ogg", "halo_infinite/sub/b.ogg"} {
		if !exists(filepath.Join(root, filepath.FromSlash(p))) {
			t.Errorf("%s absent", p)
		}
	}
	if exists(filepath.Join(root, "halo_infinite", "note.ogg")) {
		t.Error("un fichier non WAV a été encodé")
	}

	enc2 := &fakeEncoder{}
	rep, err = run(context.Background(), options{Src: root, Dst: root}, enc2, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Encoded != 0 || rep.Skipped != 2 || len(enc2.encoded) != 0 {
		t.Fatalf("second passage non idempotent : %+v", rep)
	}
}

func TestRunReencodeUnOggPerime(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	writeFile(t, filepath.Join(root, "a.ogg"), now.Add(-time.Hour))
	writeFile(t, filepath.Join(root, "a.wav"), now)
	enc := &fakeEncoder{}
	rep, err := run(context.Background(), options{Src: root, Dst: root}, enc, quiet)
	if err != nil || rep.Encoded != 1 {
		t.Fatalf("rep=%+v err=%v", rep, err)
	}
}

func TestRunCheckCouvreAussiLesSonsAJour(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	writeFile(t, filepath.Join(root, "a.wav"), now.Add(-time.Hour))
	writeFile(t, filepath.Join(root, "a.ogg"), now)
	enc := &fakeEncoder{}
	rep, err := run(context.Background(), options{Src: root, Dst: root, Check: true}, enc, quiet)
	if err != nil || rep.Skipped != 1 || rep.Checked != 1 || len(enc.checked) != 1 {
		t.Fatalf("rep=%+v err=%v checked=%v", rep, err, enc.checked)
	}
}

func TestRunUnEchecNArretePasLesAutresEtRemonte(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	writeFile(t, filepath.Join(root, "a.wav"), now)
	writeFile(t, filepath.Join(root, "b.wav"), now)
	enc := &fakeEncoder{failOn: "a.wav"}
	rep, err := run(context.Background(), options{Src: root, Dst: root, RemoveSources: true}, enc, quiet)
	if err == nil || len(rep.Failures) != 1 || rep.Encoded != 1 {
		t.Fatalf("rep=%+v err=%v", rep, err)
	}
	if !exists(filepath.Join(root, "a.wav")) {
		t.Error("la source d'un son en échec a été retirée")
	}
	if exists(filepath.Join(root, "b.wav")) || !exists(filepath.Join(root, "b.ogg")) {
		t.Error("b : source non retirée ou ogg absent")
	}
}

func TestRunUnControleEnEchecGardeLaSource(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.wav"), time.Now())
	enc := &fakeEncoder{checkErr: errors.New("échantillons différents")}
	rep, err := run(context.Background(), options{Src: root, Dst: root, Check: true, RemoveSources: true}, enc, quiet)
	if err == nil || len(rep.Failures) != 1 || rep.Removed != 0 {
		t.Fatalf("rep=%+v err=%v", rep, err)
	}
	if !exists(filepath.Join(root, "a.wav")) {
		t.Error("la source a été retirée malgré l'échec du contrôle")
	}
}

func TestRunDossierSourceAbsent(t *testing.T) {
	_, err := run(context.Background(), options{Src: filepath.Join(t.TempDir(), "absent")}, &fakeEncoder{}, quiet)
	if err == nil {
		t.Fatal("un dossier source absent doit être une erreur")
	}
}

// writeSineWAV écrit un WAV PCM 16 bits de n échantillons par canal.
func writeSineWAV(t *testing.T, path string, channels, n int) {
	t.Helper()
	const rate = 48000
	body := make([]byte, n*channels*2)
	for i := 0; i < n; i++ {
		v := int16(8000 * math.Sin(2*math.Pi*440*float64(i)/rate))
		for c := 0; c < channels; c++ {
			binary.LittleEndian.PutUint16(body[(i*channels+c)*2:], uint16(v))
		}
	}
	h := make([]byte, 44)
	copy(h, "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+len(body)))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1)
	binary.LittleEndian.PutUint16(h[22:], uint16(channels))
	binary.LittleEndian.PutUint32(h[24:], rate)
	binary.LittleEndian.PutUint32(h[28:], uint32(rate*channels*2))
	binary.LittleEndian.PutUint16(h[32:], uint16(channels*2))
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(len(body)))
	if err := os.WriteFile(path, append(h, body...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Avec le vrai ffmpeg : stéréo et mono, nombre d'échantillons décodé identique, source retirée.
func TestRunAvecFFmpegReel(t *testing.T) {
	if _, err := media.ResolveFFmpeg(); err != nil {
		t.Skip("ffmpeg absent")
	}
	if _, err := media.ResolveFFprobe(); err != nil {
		t.Skip("ffprobe absent")
	}
	root := t.TempDir()
	writeSineWAV(t, filepath.Join(root, "stereo.wav"), 2, 48000+123)
	writeSineWAV(t, filepath.Join(root, "mono.wav"), 1, 7777)
	rep, err := run(context.Background(), options{Src: root, Dst: root, Check: true, RemoveSources: true}, mediaEncoder{}, quiet)
	if err != nil {
		t.Fatalf("rep=%+v err=%v", rep, err)
	}
	if rep.Encoded != 2 || rep.Checked != 2 || rep.Removed != 2 {
		t.Fatalf("bilan : %+v", rep)
	}
	for _, n := range []string{"stereo", "mono"} {
		if exists(filepath.Join(root, n+".wav")) || !exists(filepath.Join(root, n+".ogg")) {
			t.Errorf("%s : état des fichiers inattendu", n)
		}
		if exists(filepath.Join(root, n+".ogg.part")) {
			t.Errorf("%s : reste .part", n)
		}
	}
	got, err := media.DecodedSoundFrames(context.Background(), filepath.Join(root, "mono.ogg"))
	if err != nil || got.Frames != 7777 || got.Channels != 1 || got.SampleRate != 48000 {
		t.Fatalf("mono décodé : %+v err=%v", got, err)
	}
}
