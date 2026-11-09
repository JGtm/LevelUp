package ops

// media_trash_test.go — OSMediaFileTrasher, entièrement sur des appels système injectés : AUCUN
// test n'envoie un fichier réel à la Corbeille. La doublure de l'opération de coquille retire le
// fichier elle-même (réussite) ou le laisse (refus).

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// coquilleDouble : une opération de coquille factice qui note ses appels.
type coquilleDouble struct {
	appels  []string
	code    int32
	aborted bool
	err     error
	retire  bool // retire le fichier quand elle réussit (une vraie Corbeille le fait)
}

func (c *coquilleDouble) delete(abs string) (int32, bool, error) {
	c.appels = append(c.appels, abs)
	if c.err == nil && c.code == 0 && !c.aborted && c.retire {
		_ = os.Remove(abs)
	}
	return c.code, c.aborted, c.err
}

func newTrasherDouble(base string, drive uint32, c *coquilleDouble) *OSMediaFileTrasher {
	return &OSMediaFileTrasher{
		capturesBase: func() string { return base },
		driveType:    func(string) uint32 { return drive },
		shellDelete:  c.delete,
	}
}

func fichierDeCapture(t *testing.T) (base, abs string) {
	t.Helper()
	base = t.TempDir()
	abs = filepath.Join(base, "JGtm", "clip.mkv")
	writeFileAt(t, abs)
	return base, abs
}

func existe(p string) bool { _, err := os.Lstat(p); return err == nil }

func TestOSMediaFileTrasher_LecteurFixeEnvoieALaCorbeille(t *testing.T) {
	base, abs := fichierDeCapture(t)
	c := &coquilleDouble{retire: true}
	ok, err := newTrasherDouble(base, 3, c).TrashMediaFile(context.Background(), "JGtm/clip.mkv")
	if err != nil || !ok {
		t.Fatalf("TrashMediaFile = %v, %v ; attendu true, nil", ok, err)
	}
	if len(c.appels) != 1 || c.appels[0] != abs {
		t.Fatalf("appels de la coquille = %v, attendu [%s]", c.appels, abs)
	}
	if existe(abs) {
		t.Error("le fichier est toujours dans le dossier")
	}
}

// Un lecteur qui n'est pas fixe (amovible 2, réseau 4, CD 5, disque RAM 6, inconnu 0, absent 1) n'a
// pas de Corbeille : rien n'est envoyé, l'appelant le sait (false, nil), le fichier est intact.
func TestOSMediaFileTrasher_LecteurSansCorbeille(t *testing.T) {
	for _, drive := range []uint32{0, 1, 2, 4, 5, 6} {
		base, abs := fichierDeCapture(t)
		c := &coquilleDouble{retire: true}
		ok, err := newTrasherDouble(base, drive, c).TrashMediaFile(context.Background(), "JGtm/clip.mkv")
		if err != nil || ok {
			t.Errorf("lecteur %d : %v, %v ; attendu false, nil", drive, ok, err)
		}
		if len(c.appels) != 0 {
			t.Errorf("lecteur %d : la coquille a été appelée (%v)", drive, c.appels)
		}
		if !existe(abs) {
			t.Errorf("lecteur %d : le fichier a disparu", drive)
		}
	}
}

func TestOSMediaFileTrasher_FichierAbsent(t *testing.T) {
	base := t.TempDir()
	c := &coquilleDouble{retire: true}
	ok, err := newTrasherDouble(base, 3, c).TrashMediaFile(context.Background(), "JGtm/absent.mkv")
	if err != nil || ok {
		t.Fatalf("fichier absent : %v, %v ; attendu false, nil", ok, err)
	}
	if len(c.appels) != 0 {
		t.Errorf("la coquille a été appelée pour un fichier absent : %v", c.appels)
	}
}

// Code d'erreur, opération annulée, échec de l'appel : une erreur, le fichier reste en place, et
// rien d'autre n'est tenté (une seule invocation de la coquille, aucune suppression de repli).
func TestOSMediaFileTrasher_EchecResteUnEchec(t *testing.T) {
	cas := map[string]*coquilleDouble{
		"code d'erreur":       {code: 0x7C, retire: true},
		"opération annulée":   {aborted: true, retire: true},
		"échec de l'appel":    {err: errors.New("shell32 introuvable"), retire: true},
		"annulée avec code":   {code: 0x4C7, aborted: true, retire: true},
		"réussite mensongère": {retire: false}, // code 0 mais le fichier est toujours là
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			base, abs := fichierDeCapture(t)
			ok, err := newTrasherDouble(base, 3, c).TrashMediaFile(context.Background(), "JGtm/clip.mkv")
			if err == nil || ok {
				t.Fatalf("%v, %v ; attendu false et une erreur", ok, err)
			}
			if len(c.appels) != 1 {
				t.Errorf("%d appels de la coquille, attendu 1", len(c.appels))
			}
			if !existe(abs) {
				t.Error("le fichier a disparu malgré l'échec")
			}
		})
	}
}

func TestOSMediaFileTrasher_DossierRefuse(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "JGtm", "dossier"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := &coquilleDouble{retire: true}
	ok, err := newTrasherDouble(base, 3, c).TrashMediaFile(context.Background(), "JGtm/dossier")
	if err == nil || ok || len(c.appels) != 0 {
		t.Fatalf("dossier : %v, %v, appels %v ; attendu une erreur sans appel", ok, err, c.appels)
	}
}

// Un chemin relatif sans dossier de captures n'est pas résolvable : rien n'est envoyé à l'aveugle.
func TestOSMediaFileTrasher_CheminNonResolvable(t *testing.T) {
	c := &coquilleDouble{retire: true}
	tr := newTrasherDouble("", 3, c)
	for _, stored := range []string{"JGtm/clip.mkv", "", "   "} {
		ok, err := tr.TrashMediaFile(context.Background(), stored)
		if err != nil || ok {
			t.Errorf("%q : %v, %v ; attendu false, nil", stored, ok, err)
		}
	}
	if len(c.appels) != 0 {
		t.Errorf("la coquille a été appelée : %v", c.appels)
	}
}

// La racine passée au type de lecteur est celle du volume du fichier, avec son séparateur final.
func TestOSMediaFileTrasher_RacineDuVolume(t *testing.T) {
	base, abs := fichierDeCapture(t)
	var racine string
	tr := &OSMediaFileTrasher{
		capturesBase: func() string { return base },
		driveType:    func(r string) uint32 { racine = r; return 3 },
		shellDelete:  (&coquilleDouble{retire: true}).delete,
	}
	if _, err := tr.TrashMediaFile(context.Background(), "JGtm/clip.mkv"); err != nil {
		t.Fatal(err)
	}
	if want := filepath.VolumeName(abs) + `\`; racine != want {
		t.Errorf("racine = %q, attendu %q", racine, want)
	}
	if !strings.HasSuffix(racine, `\`) {
		t.Errorf("racine %q sans séparateur final", racine)
	}
}
