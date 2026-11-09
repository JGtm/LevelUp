package main

// livraison_opus_test.go — LA LIVRAISON ENCODE EN OPUS, AVEC LE VRAI ENCODEUR.
//
// TestLivrerOctetPourOctet prouve le mixage en PCM (encodeur identite). Celui-ci prouve que le
// fichier publie est bien un Ogg Opus qui se decode en EXACTEMENT les echantillons du golden
// PCM : la troncature a 1,2 s et le mixage survivent a l'encodage a l'echantillon pres (c'est
// ce qui protege les boucles). Il est saute sans ffmpeg/ffprobe.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/media"
)

func TestLivrerEncodeEnOpus(t *testing.T) {
	if _, err := media.ResolveFFmpeg(); err != nil {
		t.Skip("ffmpeg absent")
	}
	if _, err := media.ResolveFFprobe(); err != nil {
		t.Skip("ffprobe absent")
	}
	racine := t.TempDir()
	if err := livraisonEcrireJeuSynthetique(racine); err != nil {
		t.Fatalf("jeu synthetique: %v", err)
	}
	depot := filepath.Join(racine, "_depot")
	if err := livraisonEcrireDepotSynthetique(depot); err != nil {
		t.Fatalf("depot synthetique: %v", err)
	}
	if _, err := livraisonCapturerSortie(func() error {
		return livrer(filepath.Join(racine, "_donnees"), racine, depot)
	}); err != nil {
		t.Fatalf("livrer: %v", err)
	}

	cible := filepath.Join(depot, "static", "sounds", "halo_infinite")
	goldens, err := filepath.Glob(filepath.Join(livraisonGoldensDir, "hinf_*.wav"))
	if err != nil || len(goldens) == 0 {
		t.Fatalf("goldens introuvables (%v)", err)
	}
	ctx := context.Background()
	for _, g := range goldens {
		publie := filepath.Join(cible, strings.TrimSuffix(filepath.Base(g), ".wav")+".ogg")
		octets, err := os.ReadFile(publie)
		if err != nil {
			t.Errorf("%s absent: %v", publie, err)
			continue
		}
		if !bytes.HasPrefix(octets, []byte("OggS")) {
			t.Errorf("%s n'est pas un flux Ogg", publie)
			continue
		}
		livraisonVerifierEchantillons(ctx, t, g, publie)
	}
	// Aucun intermediaire PCM ne reste dans la cible, et le temoin hors perimetre est intact.
	entrees, err := os.ReadDir(cible)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entrees {
		if strings.HasSuffix(e.Name(), media.SoundSourceExt) || strings.HasPrefix(e.Name(), ".") {
			t.Errorf("fichier parasite dans la cible: %s", e.Name())
		}
	}
	temoin, err := os.ReadFile(filepath.Join(cible, "melee_kill.ogg"))
	if err != nil || string(temoin) != "OggS melee_kill.ogg" {
		t.Errorf("melee_kill.ogg altere (%q, %v)", temoin, err)
	}
}

// livraisonVerifierEchantillons verifie que l'Ogg se decode en autant d'echantillons que le
// golden PCM. Opus decode toujours a 48 kHz : une source a une autre cadence (le jeu synthetique
// en porte une a 44,1 kHz, que les vrais sons n'ont pas) est comptee a 48 kHz.
func livraisonVerifierEchantillons(ctx context.Context, t *testing.T, golden, ogg string) {
	t.Helper()
	veut, err := media.DecodedSoundFrames(ctx, golden)
	if err != nil {
		t.Errorf("golden %s: %v", golden, err)
		return
	}
	got, err := media.DecodedSoundFrames(ctx, ogg)
	if err != nil {
		t.Errorf("%s: %v", ogg, err)
		return
	}
	attendu := media.SoundFrames{SampleRate: 48000, Channels: veut.Channels, Frames: veut.Frames * 48000 / int64(veut.SampleRate)}
	if got != attendu {
		t.Errorf("%s decode en %+v, attendu %+v (golden %+v)", ogg, got, attendu, veut)
	}
}
