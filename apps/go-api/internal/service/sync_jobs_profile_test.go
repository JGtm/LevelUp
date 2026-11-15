package service

import (
	"errors"
	"testing"

	"levelup/go-api/internal/port"
)

func TestFindPlayerProfile(t *testing.T) {
	f := newSyncJobsFixture(t)

	p, found, err := f.svc.findPlayerProfile("jgtm")
	if err != nil || !found || p.Gamertag != "JGtm" || p.XUID != "1001" {
		t.Fatalf("jgtm : %+v found=%v err=%v", p, found, err)
	}
	if got := f.profiles.filters[0]; len(got) != 0 {
		t.Errorf("sans titre, la recherche porte sur tous les titres, filtre reçu %v", got)
	}

	if _, found, err := f.svc.findPlayerProfile("inconnu"); err != nil || found {
		t.Errorf("joueur inconnu : found=%v err=%v", found, err)
	}

	// Restreint à un titre : un joueur suivi ailleurs seulement est absent.
	if _, found, _ := f.svc.findPlayerProfile("jgtm", "halo_5"); found {
		t.Error("jgtm n'est pas suivi sur halo_5")
	}
	if p, found, _ := f.svc.findPlayerProfile("h5only", "halo_5"); !found || p.TitleSlug != "halo_5" {
		t.Errorf("h5only sur halo_5 : %+v found=%v", p, found)
	}
}

// Un profil dont le gamertag est vide compte comme absent.
func TestFindPlayerProfile_SansGamertag(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.profiles.players[0].Gamertag = ""
	if _, found, err := f.svc.findPlayerProfile("jgtm"); err != nil || found {
		t.Errorf("profil sans gamertag : found=%v err=%v", found, err)
	}
}

// Une source illisible est un refus typé qui porte la cause.
func TestFindPlayerProfile_SourceIllisible(t *testing.T) {
	f := newSyncJobsFixture(t)
	boom := errors.New("fichier illisible")
	f.profiles.err = boom
	_, _, err := f.svc.findPlayerProfile("jgtm")
	je := requireSyncJobError(t, err, port.SyncErrProfilesLoad)
	if !errors.Is(je, boom) {
		t.Errorf("la cause doit rester accessible (errors.Is), got %v", je.Err)
	}
}
