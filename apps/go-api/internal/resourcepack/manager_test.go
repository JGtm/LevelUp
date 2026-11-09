package resourcepack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// packServer sert une archive et sa signature. withRange : répond aux requêtes Range (206).
type packServer struct {
	srv       *httptest.Server
	zip       []byte
	sig       string
	withRange bool
	cutAfter  int // >0 : coupe la première réponse après ce nombre d'octets
	hits      atomic.Int32
}

func newPackServer(t *testing.T, zipPath, sig string, withRange bool) *packServer {
	t.Helper()
	raw, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	ps := &packServer{zip: raw, sig: sig, withRange: withRange}
	mux := http.NewServeMux()
	mux.HandleFunc("/pack.zip", ps.serveZip)
	mux.HandleFunc("/pack.zip.sig", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(ps.sig + "\n")) })
	ps.srv = httptest.NewServer(mux)
	t.Cleanup(ps.srv.Close)
	return ps
}

func (ps *packServer) serveZip(w http.ResponseWriter, r *http.Request) {
	n := ps.hits.Add(1)
	body, status := ps.zip, http.StatusOK
	if rng := r.Header.Get("Range"); rng != "" && ps.withRange {
		start, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(rng, "bytes="), "-"))
		body, status = ps.zip[start:], http.StatusPartialContent
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if ps.cutAfter > 0 && n == 1 {
		_, _ = w.Write(body[:ps.cutAfter])
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		panic(http.ErrAbortHandler) // coupe la connexion en plein corps
	}
	_, _ = w.Write(body)
}

func allowAny(*url.URL) bool { return true }

func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// newTestManager : une racine installée (ancien paquet) et un gestionnaire prêt à préparer le nouveau.
func newTestManager(t *testing.T, key testKey, gate func(context.Context, func()) error) (*Manager, string, *[]Status) {
	t.Helper()
	root := installedRoot(t)
	pk, err := ParsePublicKey(key.Public())
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var seen []Status
	m := NewManager(Config{
		Root: root, Expected: strings.Repeat("e", 64),
		ReleaseBase: "https://github.com/x/y/releases/download/updates", Key: &pk,
		Downloader: Downloader{Doer: http.DefaultClient, AllowURL: allowAny},
		Gate:       gate,
		OnStatus:   func(s Status) { mu.Lock(); seen = append(seen, s); mu.Unlock() },
	})
	return m, root, &seen
}

func newDescriptor(t *testing.T, key testKey, ps *packServer, pack Manifest, zipPath string) Descriptor {
	t.Helper()
	return Descriptor{
		Fingerprint: pack.Fingerprint, URL: ps.srv.URL + "/pack.zip", SHA256: sha(ps.zip),
		Size: int64(len(ps.zip)), Signature: key.Sign(t, zipPath),
	}
}

func TestPrepareTelechargeVerifieEtRangeSansToucherAuPaquetActif(t *testing.T) {
	key := newTestKey(t)
	m, root, seen := newTestManager(t, key, nil)
	before := readTree(t, root)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)

	if err := m.Prepare(t.Context(), newDescriptor(t, key, ps, pm, zipPath)); err != nil {
		t.Fatal(err)
	}
	st := m.Status()
	if st.Phase != PhaseReady || !st.RestartNeeded || st.Percent != 100 || st.Fingerprint != pm.Fingerprint {
		t.Fatalf("état final : %+v", st)
	}
	if !Staged(root, pm.Fingerprint) {
		t.Fatal("le paquet devrait être prêt")
	}
	// L'ancien paquet reste actif et intact tant que rien n'est activé.
	if got := readTree(t, root); len(got) != len(before) || got["static/old.png"] != "OLD" {
		t.Fatalf("le paquet actif a bougé pendant le téléchargement : %v", got)
	}
	var phases []Phase
	for _, s := range *seen {
		if len(phases) == 0 || phases[len(phases)-1] != s.Phase {
			phases = append(phases, s.Phase)
		}
	}
	if phases[len(phases)-1] != PhaseReady || !contains(phases, PhaseDownloading) || !contains(phases, PhaseVerifying) {
		t.Fatalf("étapes annoncées : %v", phases)
	}
	if _, err := os.Stat(PartPath(root, pm.Fingerprint)); err == nil {
		t.Fatal("l'archive partielle devrait être supprimée")
	}
}

func contains(phases []Phase, p Phase) bool {
	for _, x := range phases {
		if x == p {
			return true
		}
	}
	return false
}

func TestPrepareRefuseUneSignatureInvalideEtNeRangeRien(t *testing.T) {
	key, intruder := newTestKey(t), newTestKey(t)
	m, root, _ := newTestManager(t, key, nil)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	desc := newDescriptor(t, intruder, ps, pm, zipPath) // signée par une autre clé

	err := m.Prepare(t.Context(), desc)
	if err == nil || m.Status().Code != CodeSignature || m.Status().Phase != PhaseFailed {
		t.Fatalf("signature étrangère : %v %+v", err, m.Status())
	}
	if Staged(root, pm.Fingerprint) {
		t.Fatal("un paquet mal signé ne doit jamais être rangé")
	}
	if _, statErr := os.Stat(PartPath(root, pm.Fingerprint)); statErr == nil {
		t.Fatal("l'archive mal signée doit être supprimée")
	}
}

func TestPrepareRefuseUneSommeFausse(t *testing.T) {
	key := newTestKey(t)
	m, root, _ := newTestManager(t, key, nil)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	desc := newDescriptor(t, key, ps, pm, zipPath)
	desc.SHA256 = strings.Repeat("0", 64)

	if err := m.Prepare(t.Context(), desc); !errors.Is(err, ErrChecksum) || m.Status().Code != CodeChecksum {
		t.Fatalf("somme fausse : %v %+v", err, m.Status())
	}
	if Staged(root, pm.Fingerprint) {
		t.Fatal("rien ne doit être rangé")
	}
}

func TestPrepareReprendUnTelechargementCoupe(t *testing.T) {
	key := newTestKey(t)
	m, root, _ := newTestManager(t, key, nil)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	ps.cutAfter = len(ps.zip) / 3
	desc := newDescriptor(t, key, ps, pm, zipPath)

	if err := m.Prepare(t.Context(), desc); err == nil {
		t.Fatal("la coupure devrait faire échouer le premier essai")
	}
	part := fileSize(PartPath(root, pm.Fingerprint))
	if part <= 0 || part >= int64(len(ps.zip)) {
		t.Fatalf("archive partielle attendue, taille %d", part)
	}
	if err := m.Prepare(t.Context(), desc); err != nil {
		t.Fatalf("la reprise devrait aboutir : %v", err)
	}
	if !Staged(root, pm.Fingerprint) {
		t.Fatal("paquet prêt attendu après reprise")
	}
	if ps.hits.Load() != 2 {
		t.Fatalf("deux requêtes attendues, %d reçues", ps.hits.Load())
	}
}

func TestPrepareRepartDeZeroQuandLeServeurIgnoreRange(t *testing.T) {
	key := newTestKey(t)
	m, root, _ := newTestManager(t, key, nil)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", false)
	ps.cutAfter = 100
	desc := newDescriptor(t, key, ps, pm, zipPath)
	_ = m.Prepare(t.Context(), desc)
	if err := m.Prepare(t.Context(), desc); err != nil {
		t.Fatalf("serveur sans Range : %v", err)
	}
	if !Staged(root, pm.Fingerprint) {
		t.Fatal("paquet prêt attendu")
	}
}

func TestPrepareAttendLaFinDeLaPartieAvantLePremierOctet(t *testing.T) {
	key := newTestKey(t)
	released := make(chan struct{})
	gate := func(ctx context.Context, onWait func()) error {
		onWait()
		select {
		case <-released:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	m, _, _ := newTestManager(t, key, gate)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	desc := newDescriptor(t, key, ps, pm, zipPath)

	done := make(chan error, 1)
	go func() { done <- m.Prepare(t.Context(), desc) }()
	for m.Status().Phase != PhaseWaiting {
		select {
		case err := <-done:
			t.Fatalf("la préparation a fini sans attendre : %v", err)
		default:
		}
	}
	if ps.hits.Load() != 0 {
		t.Fatal("aucun octet ne doit être demandé pendant une partie")
	}
	close(released)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if m.Status().Phase != PhaseReady {
		t.Fatalf("état : %+v", m.Status())
	}
}

func TestPrepareNeRetelechargeNiLeCourantNiLePret(t *testing.T) {
	key := newTestKey(t)
	m, root, _ := newTestManager(t, key, nil)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	desc := newDescriptor(t, key, ps, pm, zipPath)
	if err := m.Prepare(t.Context(), desc); err != nil {
		t.Fatal(err)
	}
	if err := m.Prepare(t.Context(), desc); err != nil || ps.hits.Load() != 1 {
		t.Fatalf("paquet déjà prêt : %v, %d requêtes", err, ps.hits.Load())
	}
	if err := Activate(t.Context(), root, pm.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if err := m.Prepare(t.Context(), desc); err != nil || ps.hits.Load() != 1 || m.Status().Phase != PhaseCurrent {
		t.Fatalf("paquet déjà actif : %v, %d requêtes, %+v", err, ps.hits.Load(), m.Status())
	}
}

func TestPrepareRefuseSansCleEtUneAdresseHorsListe(t *testing.T) {
	m := NewManager(Config{Root: t.TempDir(), Expected: strings.Repeat("e", 64)})
	if err := m.Prepare(t.Context(), Descriptor{}); err == nil || m.Status().Code != CodeNoKey {
		t.Fatalf("sans clé : %v %+v", err, m.Status())
	}
	key := newTestKey(t)
	pk, _ := ParsePublicKey(key.Public())
	m = NewManager(Config{Root: t.TempDir(), Key: &pk, Downloader: Downloader{Doer: http.DefaultClient}})
	d := Descriptor{Fingerprint: strings.Repeat("a", 64), URL: "https://evil.example/pack.zip", Signature: "x"}
	if err := m.Prepare(t.Context(), d); err == nil || m.Status().Code != CodeDownload {
		t.Fatalf("hôte hors liste : %v %+v", err, m.Status())
	}
	d.URL = "http://github.com/pack.zip"
	if err := m.Prepare(t.Context(), d); err == nil {
		t.Fatal("http clair accepté")
	}
}

func TestOwnDescriptorEtSignatureParAdresse(t *testing.T) {
	key := newTestKey(t)
	m, root, _ := newTestManager(t, key, nil)
	own := m.OwnDescriptor()
	wantURL := "https://github.com/x/y/releases/download/updates/levelup-resources-" + strings.Repeat("e", 64) + ".zip"
	if own.URL != wantURL || own.SignatureURL != wantURL+".sig" || own.Validate() != nil {
		t.Fatalf("descripteur propre : %+v (%v)", own, own.Validate())
	}
	// Le paquet propre se prépare avec une signature lue à l'adresse `.sig`, sans somme fournie.
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, key.Sign(t, zipPath), true)
	desc := Descriptor{Fingerprint: pm.Fingerprint, URL: ps.srv.URL + "/pack.zip", SignatureURL: ps.srv.URL + "/pack.zip.sig"}
	if err := m.Prepare(t.Context(), desc); err != nil {
		t.Fatal(err)
	}
	if !Staged(root, pm.Fingerprint) {
		t.Fatal("paquet prêt attendu")
	}
}

func TestStartLanceEnTacheDeFondSuivieEtRefuseUneSecondePreparation(t *testing.T) {
	key := newTestKey(t)
	released := make(chan struct{})
	gate := func(ctx context.Context, onWait func()) error {
		onWait()
		select {
		case <-released:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	m, root, _ := newTestManager(t, key, gate)
	var launched sync.WaitGroup
	m.Bind(t.Context(), func(fn func()) {
		launched.Add(1)
		go func() { defer launched.Done(); fn() }()
	})
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	desc := newDescriptor(t, key, ps, pm, zipPath)

	if err := m.Start(desc); err != nil {
		t.Fatal(err)
	}
	for m.Status().Phase != PhaseWaiting {
	}
	if err := m.Start(desc); !errors.Is(err, ErrBusy) {
		t.Fatalf("seconde préparation : %v", err)
	}
	close(released)
	launched.Wait()
	if !Staged(root, pm.Fingerprint) || m.Status().Phase != PhaseReady {
		t.Fatalf("paquet prêt attendu : %+v", m.Status())
	}
}

func TestPrepareNowNAttendPasLaFinDeLaPartie(t *testing.T) {
	key := newTestKey(t)
	gate := func(context.Context, func()) error { return errors.New("la porte ne doit pas être consultée") }
	m, root, _ := newTestManager(t, key, gate)
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	if err := m.PrepareNow(t.Context(), newDescriptor(t, key, ps, pm, zipPath)); err != nil {
		t.Fatal(err)
	}
	if !Staged(root, pm.Fingerprint) {
		t.Fatal("paquet prêt attendu")
	}
}

// TestRetryPoseLaPhaseAvantDeRendre : la réponse du nouvel essai porte déjà l'étape en cours (jamais l'échec qu'on
// réessaie), et un second essai tout de suite est refusé : l'interface ne reste pas sur l'échec sans progression.
func TestRetryPoseLaPhaseAvantDeRendre(t *testing.T) {
	released := make(chan struct{})
	gate := func(ctx context.Context, onWait func()) error {
		onWait()
		select {
		case <-released:
			return errors.New("fin du test")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	m, _, _ := newTestManager(t, newTestKey(t), gate)
	var launched sync.WaitGroup
	m.Bind(t.Context(), func(fn func()) {
		launched.Add(1)
		go func() { defer launched.Done(); fn() }()
	})
	m.set(func(s *Status) { s.Phase, s.Code = PhaseFailed, CodeDownload })

	st, err := m.Retry()
	if err != nil {
		t.Fatal(err)
	}
	if !st.Phase.busy() || st.Code != "" || st.Fingerprint != m.cfg.Expected {
		t.Fatalf("la réponse de Retry doit porter la préparation lancée, pas l'échec : %+v", st)
	}
	if _, err := m.Retry(); !errors.Is(err, ErrBusy) {
		t.Fatalf("second essai : %v", err)
	}
	close(released)
	launched.Wait()
}

// TestStartPourUnAutrePaquetRemplaceLaPreparationEnCours : une demande pour le paquet B pendant la préparation du
// paquet A aboutit toujours à un événement terminal pour B : A est annulée (événement d'échec pour A), B est
// lancée à sa suite.
func TestStartPourUnAutrePaquetRemplaceLaPreparationEnCours(t *testing.T) {
	key := newTestKey(t)
	var calls atomic.Int32
	gate := func(ctx context.Context, onWait func()) error {
		if calls.Add(1) > 1 {
			return nil
		}
		onWait()
		<-ctx.Done() // la préparation de A attend la fin d'une partie, jusqu'à ce qu'on l'annule
		return ctx.Err()
	}
	m, root, seen := newTestManager(t, key, gate)
	var launched sync.WaitGroup
	m.Bind(t.Context(), func(fn func()) {
		launched.Add(1)
		go func() { defer launched.Done(); fn() }()
	})
	a := Descriptor{Fingerprint: strings.Repeat("a", 64), URL: "https://github.com/x/a.zip", Signature: "x"}
	pack, pm := buildPack(t, newPackFiles())
	zipPath := zipPack(t, pack, nil)
	ps := newPackServer(t, zipPath, "", true)
	b := newDescriptor(t, key, ps, pm, zipPath)

	if err := m.Start(a); err != nil {
		t.Fatal(err)
	}
	for m.Status().Phase != PhaseWaiting {
	}
	if err := m.Start(b); err != nil {
		t.Fatalf("une demande pour un autre paquet doit être acceptée : %v", err)
	}
	launched.Wait()

	if !Staged(root, pm.Fingerprint) {
		t.Fatalf("le paquet B devait être préparé : %+v", m.Status())
	}
	terminal := map[string]Phase{}
	for _, s := range *seen {
		if s.Phase == PhaseReady || s.Phase == PhaseFailed || s.Phase == PhaseCurrent {
			terminal[s.Fingerprint] = s.Phase
		}
	}
	if terminal[a.Fingerprint] != PhaseFailed || terminal[pm.Fingerprint] != PhaseReady {
		t.Fatalf("événements terminaux par paquet : %v", terminal)
	}
}
