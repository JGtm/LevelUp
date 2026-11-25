package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type decodeurFab struct{ echec map[string]bool }

func (d decodeurFab) Decoder(_ context.Context, wem, wav string) error {
	if d.echec[filepath.Base(wem)] {
		return errors.New("décodage refusé")
	}
	return os.WriteFile(wav, []byte("mono"), 0o644)
}

func stereoFab(_ context.Context, src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, append(b, []byte("+stéréo")...), 0o644)
}

func TestConvertirVoixDecodeEtMetEnStereo(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "wav")
	for _, n := range []string{"00000001.wem", "00000002.wem", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(src, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := convertirVoix(context.Background(), decodeurFab{}, stereoFab, src, dst, journalDeTest(t))
	if err != nil || n != 2 {
		t.Fatalf("convertis = %d, %v", n, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "00000001.wav")); string(b) != "mono+stéréo" {
		t.Errorf("wav = %q : décodé puis mis en stéréo", b)
	}
	if _, err := os.Stat(filepath.Join(dst, "notes.wav")); err == nil {
		t.Error("seuls les .wem se convertissent")
	}
}

func TestConvertirVoixNomeLesEchecsEtContinue(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	for _, n := range []string{"00000001.wem", "00000002.wem"} {
		if err := os.WriteFile(filepath.Join(src, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := convertirVoix(context.Background(), decodeurFab{echec: map[string]bool{"00000001.wem": true}}, stereoFab, src, dst, journalDeTest(t))
	if err == nil || n != 1 {
		t.Errorf("convertis = %d, %v : un média en échec ne doit pas arrêter les autres, mais l'étape doit échouer", n, err)
	}
	if _, err := os.Stat(filepath.Join(dst, "00000002.wav")); err != nil {
		t.Error("le deuxième média devait être converti malgré l'échec du premier")
	}
}
