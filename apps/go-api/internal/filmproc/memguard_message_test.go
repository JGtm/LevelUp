package filmproc

// memguard_message_test.go — LA LIGNE D'ARMEMENT EST UNE SORTIE OBSERVABLE.
//
// Deux `main` portaient chacun leur copie de la sentinelle et journalisaient LEUR texte
// d'armement ; la centralisation du lot v2 G.1 les a remplaces par un texte unique, ce qui a
// casse tout filtre de journal cale sur les anciens libelles (constat C5 de la revue R1). Le
// texte est redevenu le choix de l'appelant : ce test prouve que l'option est honoree, que le
// defaut sert les autres binaires, et que les trois champs d'avant (`souple_gib`,
// `dur_octets`, plus `outil`) sont bien ceux qui partent.

import (
	"context"
	"log/slog"
	"runtime/debug"
	"testing"
)

// capteurSlog retient les enregistrements emis, pour les relire dans le test.
type capteurSlog struct{ enregs []slog.Record }

func (c *capteurSlog) Enabled(context.Context, slog.Level) bool { return true }
func (c *capteurSlog) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *capteurSlog) WithGroup(string) slog.Handler            { return c }
func (c *capteurSlog) Handle(_ context.Context, r slog.Record) error {
	c.enregs = append(c.enregs, r)
	return nil
}

// armerSousCapteur arme une sentinelle et rend l'enregistrement journalise.
//
// LE PLAFOND SOUPLE DU PROCESSUS DE TEST EST RESTAURE : `Arm` appelle debug.SetMemoryLimit,
// qui est un etat GLOBAL du runtime — le laisser pose ferait travailler le GC des tests
// suivants de ce paquet sous une contrainte qu'ils n'ont pas demandee.
func armerSousCapteur(t *testing.T, opts ...ArmOption) slog.Record {
	t.Helper()
	ancienPlafond := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(ancienPlafond) })
	ancienLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(ancienLogger) })
	c := &capteurSlog{}
	slog.SetDefault(slog.New(c))

	Arm("outil-de-test", 64, nil, opts...).Disarm()
	if len(c.enregs) != 1 {
		t.Fatalf("%d ligne(s) journalisee(s) a l'armement, attendu 1", len(c.enregs))
	}
	return c.enregs[0]
}

func TestArmJournaliseLeMessageDeLAppelant(t *testing.T) {
	r := armerSousCapteur(t, WithArmMessage("plafond memoire arme"))
	if r.Message != "plafond memoire arme" {
		t.Errorf("message = %q, attendu %q (le texte d'avant la centralisation, cf. "+
			"a21fd77f4:cmd/levelup/backfill_memlimit.go)", r.Message, "plafond memoire arme")
	}
	champs := map[string]string{}
	r.Attrs(func(a slog.Attr) bool {
		champs[a.Key] = a.Value.String()
		return true
	})
	if champs["souple_gib"] != "64" {
		t.Errorf("champ souple_gib = %q, attendu \"64\"", champs["souple_gib"])
	}
	if got, veut := champs["dur_octets"], HardLimitFor(64); got != uint64String(veut) {
		t.Errorf("champ dur_octets = %q, attendu %d — HardLimitFor doit rendre le plafond "+
			"que Arm pose reellement", got, veut)
	}
	if champs["outil"] != "outil-de-test" {
		t.Errorf("champ outil = %q, attendu \"outil-de-test\"", champs["outil"])
	}
}

func TestArmMessageParDefautSansOption(t *testing.T) {
	if r := armerSousCapteur(t); r.Message != messageArmementParDefaut {
		t.Errorf("message par defaut = %q, attendu %q", r.Message, messageArmementParDefaut)
	}
}

// TestHardLimitForDesarme : l'echappatoire de l'operateur ne pose aucun plafond dur, et
// l'appelant qui journalise ce plafond doit lire zero plutot qu'une marge sur zero.
func TestHardLimitForDesarme(t *testing.T) {
	for _, giB := range []int{0, -1} {
		if got := HardLimitFor(giB); got != 0 {
			t.Errorf("HardLimitFor(%d) = %d, attendu 0", giB, got)
		}
	}
	if got, veut := HardLimitFor(4), uint64(4*octetsParGiB+octetsParGiB); got != veut {
		t.Errorf("HardLimitFor(4) = %d, attendu %d (souple + 25 %%)", got, veut)
	}
}

// uint64String : la forme que slog donne a un uint64 dans Value.String().
func uint64String(v uint64) string {
	return slog.Uint64Value(v).String()
}
