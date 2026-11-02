package syslocale

import (
	"errors"
	"testing"
	"time"
	_ "time/tzdata"
)

func TestIANAFromWindows(t *testing.T) {
	cas := map[string]string{
		"Romance Standard Time":          "Europe/Paris",
		"W. Europe Standard Time":        "Europe/Berlin",
		"GMT Standard Time":              "Europe/London",
		"Eastern Standard Time":          "America/New_York",
		"Pacific Standard Time":          "America/Los_Angeles",
		"Tokyo Standard Time":            "Asia/Tokyo",
		"Central Standard Time (Mexico)": "America/Mexico_City",
		"UTC":                            "Etc/UTC",
		"AUS Eastern Standard Time":      "Australia/Sydney",
	}
	for key, want := range cas {
		if got, ok := IANAFromWindows(key); !ok || got != want {
			t.Errorf("IANAFromWindows(%q) = %q, %v ; attendu %q", key, got, ok, want)
		}
	}
	for _, key := range []string{"", "Fuseau Inventé Standard Time", "romance standard time"} {
		if got, ok := IANAFromWindows(key); ok {
			t.Errorf("IANAFromWindows(%q) = %q ; une clé inconnue ne se traduit pas (la casse compte)", key, got)
		}
	}
}

// TestWindowsZones_EveryTargetLoads : chaque nom IANA de la table existe dans la base de fuseaux
// embarquée ; une faute de frappe ferait retomber silencieusement un utilisateur sur le repli.
func TestWindowsZones_EveryTargetLoads(t *testing.T) {
	for key, iana := range windowsZones {
		if _, err := time.LoadLocation(iana); err != nil {
			t.Errorf("%q -> %q : %v", key, iana, err)
		}
	}
}

func TestRawTimezone_PrefersAnExplicitIANAName(t *testing.T) {
	r := Raw{IANAZone: "Asia/Tokyo", WindowsZone: "Romance Standard Time"}
	if got, ok := r.Timezone(); !ok || got != "Asia/Tokyo" {
		t.Errorf("Timezone = %q, %v ; le nom IANA fourni gagne", got, ok)
	}
	if got, ok := (Raw{WindowsZone: "Romance Standard Time"}).Timezone(); !ok || got != "Europe/Paris" {
		t.Errorf("Timezone = %q, %v ; attendu Europe/Paris", got, ok)
	}
	if _, ok := (Raw{ReadErr: errors.New("registre illisible")}).Timezone(); ok {
		t.Error("rien de lu : aucun fuseau ne doit être rendu")
	}
}

// TestReadRaw_AnswersOnThisMachine : la lecture réelle ne panique pas ; les valeurs lues sont
// journalisées pour le témoin.
func TestReadRaw_AnswersOnThisMachine(t *testing.T) {
	r := ReadRaw()
	t.Logf("WindowsZone=%q IANAZone=%q UILanguages=%v ReadErr=%v", r.WindowsZone, r.IANAZone, r.UILanguages, r.ReadErr)
}
