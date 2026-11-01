package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func requireMediaTools(t *testing.T) {
	t.Helper()
	if _, err := ResolveFFmpeg(); err != nil {
		t.Skip("ffmpeg absent")
	}
	if _, err := ResolveFFprobe(); err != nil {
		t.Skip("ffprobe absent")
	}
}

// writeTestWAV écrit un WAV PCM 16 bits stéréo 48 kHz de n échantillons par canal.
func writeTestWAV(t *testing.T, path string, n int) {
	t.Helper()
	body := make([]byte, n*4)
	for i := 0; i < n; i++ {
		binary.LittleEndian.PutUint16(body[i*4:], uint16(int16((i*37)%6000-3000)))
		binary.LittleEndian.PutUint16(body[i*4+2:], uint16(int16((i*53)%6000-3000)))
	}
	h := make([]byte, 44)
	copy(h, "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+len(body)))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1)
	binary.LittleEndian.PutUint16(h[22:], 2)
	binary.LittleEndian.PutUint32(h[24:], 48000)
	binary.LittleEndian.PutUint32(h[28:], 48000*4)
	binary.LittleEndian.PutUint16(h[32:], 4)
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(len(body)))
	if err := os.WriteFile(path, append(h, body...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSoundEncodeArgsRecette(t *testing.T) {
	args := soundEncodeArgs("in.wav", "out.part")
	for _, want := range []string{"libopus", "96k", "ogg", "+bitexact", "-map_metadata"} {
		if !slices.Contains(args, want) {
			t.Errorf("argument %q absent de %v", want, args)
		}
	}
	if args[len(args)-1] != "out.part" {
		t.Errorf("la sortie doit être le dernier argument : %v", args)
	}
}

func TestEncodeSoundReproductibleEtExact(t *testing.T) {
	requireMediaTools(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "a.wav")
	writeTestWAV(t, src, 48000+321)
	ctx := context.Background()

	first, second := filepath.Join(dir, "x", "one.ogg"), filepath.Join(dir, "two.ogg")
	if err := EncodeSound(ctx, src, first); err != nil {
		t.Fatal(err)
	}
	if err := EncodeSound(ctx, src, second); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if !bytes.Equal(a, b) || !bytes.HasPrefix(a, []byte("OggS")) {
		t.Fatal("deux encodages de la même source doivent rendre le même Ogg")
	}
	if _, err := os.Stat(first + ".part"); err == nil {
		t.Error("reste .part après un encodage réussi")
	}
	if err := CheckSameFrames(ctx, src, first); err != nil {
		t.Fatalf("échantillons décodés différents : %v", err)
	}

	other := filepath.Join(dir, "b.wav")
	writeTestWAV(t, other, 48000+322)
	if err := CheckSameFrames(ctx, other, first); err == nil {
		t.Error("un nombre d'échantillons différent doit être refusé")
	}
}

func TestEncodeSoundSourceIllisibleNeLaissePasDeFichier(t *testing.T) {
	requireMediaTools(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "bad.wav")
	if err := os.WriteFile(src, []byte("pas un wav"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "bad.ogg")
	if err := EncodeSound(context.Background(), src, dst); err == nil {
		t.Fatal("une source illisible doit échouer")
	}
	for _, p := range []string{dst, dst + ".part"} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s laissé derrière l'échec", p)
		}
	}
}
