package resourcepack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Descriptor décrit un paquet à télécharger : ce que latest.json publie sous « resources » et ce
// que le serveur reconstruit pour SON propre paquet à partir de la version de l'application.
type Descriptor struct {
	// Fingerprint : l'empreinte du paquet (64 caractères hexadécimaux).
	Fingerprint string `json:"fingerprint"`
	// URL : l'archive .zip (https, hôte autorisé).
	URL string `json:"url"`
	// SHA256 : somme de l'archive, en hexadécimal ; vide = non fournie (la signature seule garantit).
	SHA256 string `json:"sha256,omitempty"`
	// Size : taille attendue de l'archive, 0 si inconnue.
	Size int64 `json:"size,omitempty"`
	// Signature : contenu base64 du fichier de signature minisign de l'archive.
	Signature string `json:"signature,omitempty"`
	// SignatureURL : à défaut de Signature, l'adresse du fichier `.sig`.
	SignatureURL string `json:"signature_url,omitempty"`
}

// maxSignatureBytes borne le fichier `.sig` téléchargé.
const maxSignatureBytes = 8 << 10

// allowedHosts : les seuls hôtes d'où une archive de paquet peut venir (publication GitHub et les
// adresses vers lesquelles elle redirige).
var allowedHosts = map[string]bool{
	"github.com":                            true,
	"objects.githubusercontent.com":         true,
	"release-assets.githubusercontent.com":  true,
	"github-releases.githubusercontent.com": true,
}

// DefaultAllowURL admet une adresse https dont l'hôte est un hôte de publication connu.
func DefaultAllowURL(u *url.URL) bool {
	return u.Scheme == "https" && allowedHosts[strings.ToLower(u.Hostname())]
}

// Validate vérifie la forme d'un descripteur (pas l'hôte : voir Downloader.AllowURL).
func (d Descriptor) Validate() error {
	if len(d.Fingerprint) != sha256.Size*2 {
		return errors.New("resourcepack: empreinte du descripteur mal formée")
	}
	if _, err := hex.DecodeString(d.Fingerprint); err != nil {
		return errors.New("resourcepack: empreinte du descripteur mal formée")
	}
	if d.URL == "" {
		return errors.New("resourcepack: adresse de l'archive absente")
	}
	if d.Signature == "" && d.SignatureURL == "" {
		return errors.New("resourcepack: aucune signature fournie")
	}
	if d.SHA256 != "" {
		if _, err := hex.DecodeString(d.SHA256); err != nil || len(d.SHA256) != sha256.Size*2 {
			return errors.New("resourcepack: somme du descripteur mal formée")
		}
	}
	return nil
}

// Doer émet une requête HTTP (netguard.Doer en production, un client de test sinon).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Downloader télécharge archives et signatures.
type Downloader struct {
	Doer Doer
	// AllowURL décide si une adresse peut être appelée ; nil : [DefaultAllowURL].
	AllowURL func(*url.URL) bool
	// Progress est appelé au fil de la lecture (octets reçus, taille totale connue ou 0).
	Progress func(done, total int64)
}

func (d Downloader) allow(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("resourcepack: adresse invalide: %w", err)
	}
	allow := d.AllowURL
	if allow == nil {
		allow = DefaultAllowURL
	}
	if !allow(u) {
		return nil, fmt.Errorf("resourcepack: adresse refusée: %s", u.Redacted())
	}
	return u, nil
}

// Signature rend la signature du descripteur : celle qu'il porte, sinon celle du fichier `.sig`.
func (d Downloader) Signature(ctx context.Context, desc Descriptor) (string, error) {
	if desc.Signature != "" {
		return desc.Signature, nil
	}
	u, err := d.allow(desc.SignatureURL)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := d.Doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("resourcepack: téléchargement de la signature: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resourcepack: signature: réponse %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxSignatureBytes))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// Fetch télécharge l'archive du descripteur vers part, en REPRENANT un fichier partiel laissé par
// une coupure (requête Range ; un serveur qui répond 200 fait repartir de zéro). À la fin la
// taille et la somme déclarées sont vérifiées ; une archive corrompue est supprimée.
func (d Downloader) Fetch(ctx context.Context, desc Descriptor, part string) error {
	u, err := d.allow(desc.URL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(part), 0o755); err != nil {
		return err
	}
	have := fileSize(part)
	if desc.Size > 0 && have > desc.Size {
		_ = os.Remove(part)
		have = 0
	}
	if have == 0 || have != desc.Size {
		if err := d.transfer(ctx, u, part, have, desc.Size); err != nil {
			return err
		}
	}
	return verifyDownloaded(part, desc)
}

// transfer reçoit le corps dans part à partir de l'octet have.
func (d Downloader) transfer(ctx context.Context, u *url.URL, part string, have, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
	}
	resp, err := d.Doer.Do(req)
	if err != nil {
		return fmt.Errorf("resourcepack: téléchargement: %w", err)
	}
	defer resp.Body.Close()
	flags := os.O_WRONLY | os.O_CREATE
	switch resp.StatusCode {
	case http.StatusPartialContent:
		flags |= os.O_APPEND
	case http.StatusRequestedRangeNotSatisfiable:
		return nil // le fichier partiel est déjà complet : la signature tranche
	case http.StatusOK:
		flags |= os.O_TRUNC
		have = 0
	default:
		return fmt.Errorf("resourcepack: téléchargement: réponse %d", resp.StatusCode)
	}
	out, err := os.OpenFile(part, flags, 0o644) //nolint:gosec // dossier de travail du programme
	if err != nil {
		return err
	}
	total := size
	if total == 0 && resp.ContentLength > 0 {
		total = have + resp.ContentLength
	}
	copyErr := d.copyProgress(out, resp.Body, have, total)
	return errors.Join(copyErr, out.Close())
}

// copyProgress copie src vers dst en annonçant l'avancement.
func (d Downloader) copyProgress(dst io.Writer, src io.Reader, done, total int64) error {
	buf := make([]byte, 256<<10)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if d.Progress != nil {
				d.Progress(done, total)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("resourcepack: lecture interrompue: %w", err)
		}
	}
}

// verifyDownloaded contrôle taille et somme de l'archive reçue ; une archive fausse est retirée.
func verifyDownloaded(part string, desc Descriptor) error {
	if desc.Size > 0 && fileSize(part) != desc.Size {
		return fmt.Errorf("resourcepack: archive incomplète (%d octets sur %d)", fileSize(part), desc.Size)
	}
	if desc.SHA256 == "" {
		return nil
	}
	fh, err := os.Open(part) //nolint:gosec // dossier de travail du programme
	if err != nil {
		return err
	}
	defer fh.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != strings.ToLower(desc.SHA256) {
		_ = os.Remove(part)
		return ErrChecksum
	}
	return nil
}

// ErrChecksum : la somme SHA-256 de l'archive ne correspond pas au descripteur.
var ErrChecksum = errors.New("resourcepack: somme SHA-256 de l'archive différente du descripteur")

func fileSize(p string) int64 {
	st, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return st.Size()
}
