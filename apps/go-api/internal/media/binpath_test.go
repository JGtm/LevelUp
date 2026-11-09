package media

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fauxBinaire écrit un fichier exécutable factice nommé name dans dir et rend son chemin.
func fauxBinaire(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name+binarySuffix())
	if err := os.WriteFile(p, []byte("faux"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestLocateBinary_OrdreDeRecherche : LEVELUP_FFMPEG_DIR, puis dossier de l'exécutable, puis PATH.
func TestLocateBinary_OrdreDeRecherche(t *testing.T) {
	envDir := t.TempDir()
	exeDirPath := t.TempDir()
	pathDir := t.TempDir()
	pathBin := fauxBinaire(t, pathDir, FFmpegBinary)
	exeBin := fauxBinaire(t, exeDirPath, FFmpegBinary)
	envBin := fauxBinaire(t, envDir, FFmpegBinary)

	exeFn := func() (string, error) { return exeDirPath, nil }
	lookPath := func(name string) (string, error) {
		if name == FFmpegBinary {
			return pathBin, nil
		}
		return "", errors.New("absent")
	}

	cases := []struct {
		name   string
		envSet bool
		want   string
	}{
		{"variable posée et binaire présent : elle gagne", true, envBin},
		{"variable vide : dossier de l'exécutable avant PATH", false, exeBin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envSet {
				t.Setenv(EnvFFmpegDir, envDir)
			} else {
				t.Setenv(EnvFFmpegDir, "")
			}
			got, err := locateBinary(FFmpegBinary, exeFn, lookPath)
			if err != nil || got != tc.want {
				t.Fatalf("locateBinary = %q, %v ; attendu %q", got, err, tc.want)
			}
		})
	}

	t.Run("variable posée sans binaire : repli sur le dossier de l'exécutable", func(t *testing.T) {
		t.Setenv(EnvFFmpegDir, t.TempDir())
		got, err := locateBinary(FFmpegBinary, exeFn, lookPath)
		if err != nil || got != exeBin {
			t.Fatalf("locateBinary = %q, %v ; attendu %q", got, err, exeBin)
		}
	})

	t.Run("aucun emplacement avec le binaire : PATH", func(t *testing.T) {
		t.Setenv(EnvFFmpegDir, t.TempDir())
		emptyExe := func() (string, error) { return t.TempDir(), nil }
		got, err := locateBinary(FFmpegBinary, emptyExe, lookPath)
		if err != nil || got != pathBin {
			t.Fatalf("locateBinary = %q, %v ; attendu %q", got, err, pathBin)
		}
	})

	t.Run("introuvable partout : ErrBinaryNotFound", func(t *testing.T) {
		t.Setenv(EnvFFmpegDir, "")
		none := func(string) (string, error) { return "", errors.New("absent") }
		_, err := locateBinary(FFprobeBinary, exeFn, none)
		if !errors.Is(err, ErrBinaryNotFound) {
			t.Fatalf("erreur = %v ; attendu ErrBinaryNotFound", err)
		}
	})
}
