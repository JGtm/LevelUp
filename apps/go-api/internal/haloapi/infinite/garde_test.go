package infinite

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/haloapi"
)

// doerGarde : un Doer qui porte une garde ([haloapi.Garde]) ; il compte les consultations de la
// garde et les émissions.
type doerGarde struct {
	doerCompteur
	verifs atomic.Int32
	refus  error
}

func (d *doerGarde) Verifie(context.Context) error {
	d.verifs.Add(1)
	return d.refus
}

// TestGarde_ConsulteeAvantLeLimiteur : une garde qui refuse arrête l'appel AVANT toute attente du
// limiteur et toute émission, pour les trois politiques et pour les blobs ; une garde qui laisse
// passer ne change rien.
func TestGarde_ConsulteeAvantLeLimiteur(t *testing.T) {
	cause := errors.New("sortie refusée")
	for _, r := range []Retry{RetryProvider(), RetryDiscovery(), RetryHaloClient()} {
		d := &doerGarde{refus: haloapi.NonRetryable(cause)}
		l := &limiteurCompteur{}
		c := New(d, sansDelai(r)).WithLimiter(l)
		_, err := c.GetMatchStats(context.Background(), authTest, matchTest)
		if !errors.Is(err, cause) || !errors.Is(err, haloapi.ErrDefinitif) {
			t.Errorf("essais=%d : erreur = %v, attendu la cause marquée définitive", r.Essais, err)
		}
		if l.n.Load() != 0 || d.n.Load() != 0 || d.verifs.Load() != 1 {
			t.Errorf("essais=%d : attentes=%d émissions=%d gardes=%d, attendu 0/0/1", r.Essais, l.n.Load(), d.n.Load(), d.verifs.Load())
		}
	}
	d := &doerGarde{refus: haloapi.NonRetryable(cause)}
	l := &limiteurCompteur{}
	_, _, err := New(d, sansDelai(RetryHaloClient())).WithLimiter(l).DownloadBlob(context.Background(), "http://x.invalid/b", 0)
	if !errors.Is(err, cause) || l.n.Load() != 0 || d.n.Load() != 0 || d.verifs.Load() != 1 {
		t.Errorf("blob : err=%v attentes=%d émissions=%d gardes=%d, attendu cause/0/0/1", err, l.n.Load(), d.n.Load(), d.verifs.Load())
	}
	passe := &doerGarde{doerCompteur: doerCompteur{err: errors.New("réseau")}}
	_, _ = New(passe, sansDelai(RetryProvider())).GetMatchStats(context.Background(), authTest, matchTest)
	if passe.n.Load() != 3 {
		t.Errorf("garde qui laisse passer : %d émissions, attendu 3", passe.n.Load())
	}
}
