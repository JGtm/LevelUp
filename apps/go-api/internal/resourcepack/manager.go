package resourcepack

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Phase : l'étape où en est la préparation d'un paquet.
type Phase string

const (
	// PhaseIdle : rien en cours, rien demandé.
	PhaseIdle Phase = "idle"
	// PhaseCurrent : le paquet voulu est déjà le paquet installé.
	PhaseCurrent Phase = "current"
	// PhaseWaiting : le téléchargement attend la fin d'une partie (mode jeu).
	PhaseWaiting Phase = "waiting_game"
	// PhaseDownloading : l'archive arrive.
	PhaseDownloading Phase = "downloading"
	// PhaseVerifying : signature, somme et manifeste sont contrôlés.
	PhaseVerifying Phase = "verifying"
	// PhaseReady : le paquet est rangé, prêt à être activé au prochain démarrage.
	PhaseReady Phase = "ready"
	// PhaseFailed : la préparation a échoué (voir Code).
	PhaseFailed Phase = "failed"
)

// Codes d'erreur fixes (le texte vit côté interface, dans les deux langues).
const (
	CodeDownload  = "resources_download_failed"
	CodeSignature = "resources_signature_invalid"
	CodeChecksum  = "resources_checksum_mismatch"
	CodeUnpack    = "resources_unpack_failed"
	CodeNoKey     = "resources_no_key"
	CodeRefused   = "resources_refused"
)

// Status : l'état du paquet de ressources, tel que l'interface et la coquille le lisent.
type Status struct {
	Phase Phase `json:"phase"`
	// Fingerprint : le paquet visé par la préparation en cours ou la dernière.
	Fingerprint string `json:"fingerprint,omitempty"`
	// Expected : l'empreinte que le moteur attend.
	Expected string `json:"expected,omitempty"`
	// Active : l'empreinte du paquet installé ("" si aucun).
	Active string `json:"active,omitempty"`
	// Percent : avancement du téléchargement, 0 à 100.
	Percent int `json:"percent"`
	// Code : code d'erreur de PhaseFailed.
	Code string `json:"code,omitempty"`
	// RestartNeeded : un paquet prêt n'est pas encore le paquet actif.
	RestartNeeded bool `json:"restart_needed"`
}

// Config assemble un Manager.
type Config struct {
	// Root : la racine de ressources.
	Root string
	// Expected : l'empreinte embarquée par le moteur.
	Expected string
	// Version : la version de l'application (compose l'adresse du paquet propre).
	Version string
	// ReleaseBase : racine des publications, `https://github.com/<dépôt>/releases/download`.
	ReleaseBase string
	// Key : la clé publique qui signe archives et installeurs ; vide = aucun téléchargement.
	Key *PublicKey
	// Downloader : le client de téléchargement.
	Downloader Downloader
	// Gate attend que le travail lourd soit permis (fin de partie) ; nil : jamais d'attente.
	// Elle est appelée AVANT le téléchargement, jamais pendant.
	Gate func(ctx context.Context, onWait func()) error
	// OnStatus reçoit chaque changement d'état (événement vers la coquille) ; nil : aucun.
	OnStatus func(Status)
}

// Manager prépare le paquet de ressources : télécharge, vérifie, range. Il ne bascule jamais lui-même :
// l'activation se fait au démarrage suivant ([ActivateAtBoot]).
type Manager struct {
	cfg Config

	mu      sync.Mutex
	status  Status
	running bool
	// ctx et goFn : voir [Manager.Bind].
	ctx  context.Context
	goFn func(fn func())
}

// NewManager crée un gestionnaire et lit l'état installé.
func NewManager(cfg Config) *Manager {
	m := &Manager{cfg: cfg}
	m.status = Status{Phase: PhaseIdle, Expected: cfg.Expected, Active: ActiveFingerprint(cfg.Root)}
	return m
}

// Bind relie les préparations lancées par [Manager.Start] au cycle de vie du serveur : ctx les arrête avec lui,
// run les lance en tâche de fond suivie par son arrêt. Sans Bind, Start utilise Background et une goroutine nue.
func (m *Manager) Bind(ctx context.Context, run func(fn func())) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx, m.goFn = ctx, run
}

// ActiveFingerprint rend l'empreinte du paquet installé sous root, "" s'il n'y en a pas de lisible.
func ActiveFingerprint(root string) string {
	mf, err := ReadManifest(root)
	if err != nil {
		return ""
	}
	return mf.Fingerprint
}

// Status rend l'état courant.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// set remplace l'état et le publie.
func (m *Manager) set(mut func(*Status)) {
	m.mu.Lock()
	mut(&m.status)
	st := m.status
	m.mu.Unlock()
	if m.cfg.OnStatus != nil {
		m.cfg.OnStatus(st)
	}
}

// OwnDescriptor rend le descripteur du paquet de CE moteur : archive et signature publiées avec sa
// version. Le moteur ne connaît ainsi jamais qu'un paquet, le sien.
func (m *Manager) OwnDescriptor() Descriptor {
	base := fmt.Sprintf("%s/v%s/levelup-resources-%s.zip", m.cfg.ReleaseBase, m.cfg.Version, m.cfg.Expected)
	return Descriptor{Fingerprint: m.cfg.Expected, URL: base, SignatureURL: base + ".sig"}
}

// ErrBusy : une préparation est déjà en cours.
var ErrBusy = errors.New("resourcepack: une préparation est déjà en cours")

// Start lance Prepare en tâche de fond (contexte et suivi de la configuration). Rend ErrBusy, sans rien lancer,
// quand une préparation tourne déjà.
func (m *Manager) Start(desc Descriptor) error {
	if m.Status().Phase.busy() {
		return ErrBusy
	}
	m.mu.Lock()
	ctx, goFn := m.ctx, m.goFn
	m.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	run := func() {
		if err := m.Prepare(ctx, desc); err != nil && !errors.Is(err, ErrBusy) {
			slog.WarnContext(ctx, "resources: préparation en tâche de fond échouée", "err", err)
		}
	}
	if goFn != nil {
		goFn(run)
	} else {
		go run()
	}
	return nil
}

// busy dit si la phase est celle d'une préparation en cours.
func (p Phase) busy() bool {
	return p == PhaseWaiting || p == PhaseDownloading || p == PhaseVerifying
}

// Prepare télécharge, vérifie et range le paquet du descripteur, en attendant la fin d'une partie. Une seule
// préparation à la fois. Rend nil quand le paquet est prêt (ou déjà installé) ; sinon l'erreur, dont le code est
// dans Status().Code.
func (m *Manager) Prepare(ctx context.Context, desc Descriptor) error {
	return m.run(ctx, desc, true)
}

// PrepareNow est Prepare sans attendre la fin d'une partie : le démarrage qui ne peut pas se faire sans le paquet.
func (m *Manager) PrepareNow(ctx context.Context, desc Descriptor) error {
	return m.run(ctx, desc, false)
}

func (m *Manager) run(ctx context.Context, desc Descriptor, gated bool) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return ErrBusy
	}
	m.running = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()
	m.set(func(s *Status) {
		*s = Status{Phase: PhaseIdle, Fingerprint: desc.Fingerprint, Expected: m.cfg.Expected, Active: ActiveFingerprint(m.cfg.Root)}
	})
	err := m.prepare(ctx, desc, gated)
	if err != nil {
		slog.ErrorContext(ctx, "resources: préparation du paquet échouée", "err", err,
			"fingerprint", desc.Fingerprint, "code", m.Status().Code)
	}
	return err
}

func (m *Manager) fail(code string, err error) error {
	m.set(func(s *Status) { s.Phase, s.Code, s.RestartNeeded = PhaseFailed, code, false })
	return fmt.Errorf("%s: %w", code, err)
}

func (m *Manager) prepare(ctx context.Context, desc Descriptor, gated bool) error {
	if m.cfg.Key == nil {
		return m.fail(CodeNoKey, errors.New("aucune clé publique de mise à jour"))
	}
	if err := desc.Validate(); err != nil {
		return m.fail(CodeRefused, err)
	}
	if ActiveFingerprint(m.cfg.Root) == desc.Fingerprint {
		m.set(func(s *Status) { s.Phase, s.Percent = PhaseCurrent, 100 })
		return nil
	}
	if Staged(m.cfg.Root, desc.Fingerprint) {
		m.set(func(s *Status) { s.Phase, s.Percent, s.RestartNeeded = PhaseReady, 100, true })
		return nil
	}
	if gated {
		if err := m.waitForGameToEnd(ctx); err != nil {
			return m.fail(CodeDownload, err)
		}
	}
	part, err := m.download(ctx, desc)
	if err != nil {
		return err
	}
	return m.verifyAndStage(ctx, desc, part)
}

// waitForGameToEnd attend la porte du mode jeu, une seule fois, avant le premier octet.
func (m *Manager) waitForGameToEnd(ctx context.Context) error {
	if m.cfg.Gate == nil {
		return nil
	}
	return m.cfg.Gate(ctx, func() { m.set(func(s *Status) { s.Phase = PhaseWaiting }) })
}

// download reçoit l'archive ; rend son chemin.
func (m *Manager) download(ctx context.Context, desc Descriptor) (string, error) {
	part := PartPath(m.cfg.Root, desc.Fingerprint)
	m.set(func(s *Status) { s.Phase, s.Percent = PhaseDownloading, 0 })
	var last time.Time
	dl := m.cfg.Downloader
	dl.Progress = func(done, total int64) {
		if total <= 0 || time.Since(last) < 500*time.Millisecond {
			return
		}
		last = time.Now()
		pct := int(done * 100 / total)
		m.set(func(s *Status) { s.Percent = min(pct, 99) })
	}
	if err := dl.Fetch(ctx, desc, part); err != nil {
		code := CodeDownload
		if errors.Is(err, ErrChecksum) {
			code = CodeChecksum
		}
		return "", m.fail(code, err)
	}
	return part, nil
}

// verifyAndStage contrôle la signature, extrait, compare l'empreinte et marque le paquet prêt.
func (m *Manager) verifyAndStage(ctx context.Context, desc Descriptor, part string) error {
	m.set(func(s *Status) { s.Phase, s.Percent = PhaseVerifying, 99 })
	sig, err := m.cfg.Downloader.Signature(ctx, desc)
	if err != nil {
		return m.fail(CodeDownload, err)
	}
	if err := m.cfg.Key.VerifyFile(part, sig); err != nil {
		_ = os.Remove(part)
		return m.fail(CodeSignature, err)
	}
	if _, err := StageFromZip(m.cfg.Root, part, desc.Fingerprint); err != nil {
		_ = os.Remove(part)
		return m.fail(CodeUnpack, err)
	}
	_ = os.Remove(part)
	if err := Purge(m.cfg.Root, desc.Fingerprint); err != nil {
		slog.WarnContext(ctx, "resources: nettoyage du dossier de travail incomplet", "err", err)
	}
	m.set(func(s *Status) {
		s.Phase, s.Percent, s.Code = PhaseReady, 100, ""
		s.RestartNeeded = s.Active != desc.Fingerprint
	})
	return nil
}

// processManager : le gestionnaire du processus, posé au démarrage desktop et lu par la route d'état.
var processManager atomic.Pointer[Manager]

// SetDefault installe le gestionnaire du processus (nil le retire : tests).
func SetDefault(m *Manager) { processManager.Store(m) }

// Default rend le gestionnaire du processus, nil hors du profil qui télécharge.
func Default() *Manager { return processManager.Load() }

// Retry relance, en tâche de fond, le téléchargement du paquet de CE moteur (bouton « Réessayer » de la page des
// réglages) et rend l'état. ErrBusy quand une préparation tourne déjà.
func (m *Manager) Retry() (Status, error) {
	if err := m.Start(m.OwnDescriptor()); err != nil {
		return m.Status(), err
	}
	return m.Status(), nil
}
