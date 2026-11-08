package resourcepack

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// Signature minisign (celle que produit `tauri signer sign`) : l'archive du paquet est signée par
// la clé de la mise à jour, la même que l'installeur. Seule la forme PRÉHACHÉE (algorithme « ED »,
// BLAKE2b-512 du fichier) est acceptée : c'est la forme par défaut de minisign, elle se vérifie
// en flux sans charger l'archive en mémoire. La forme historique « Ed » est refusée.

const (
	minisignAlgoPrehashed = "ED"
	minisignKeyLen        = 42 // algo (2) + identifiant de clé (8) + clé publique (32)
	minisignSigLen        = 74 // algo (2) + identifiant de clé (8) + signature (64)
	minisignGlobalLen     = 64
	trustedPrefix         = "trusted comment: "
)

// ErrBadSignature : la signature ne correspond pas au fichier ou à la clé.
var ErrBadSignature = errors.New("resourcepack: signature invalide")

// PublicKey est la clé publique de mise à jour.
type PublicKey struct {
	keyID [8]byte
	key   ed25519.PublicKey
}

// ParsePublicKey décode la clé telle que `tauri.conf.json` la porte : le contenu base64 du
// fichier `.pub` de minisign (une ligne de commentaire, puis la clé en base64).
func ParsePublicKey(confValue string) (PublicKey, error) {
	text, err := decodeStd(confValue)
	if err != nil {
		return PublicKey{}, fmt.Errorf("resourcepack: clé publique: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(text)), "\n")
	if len(lines) < 2 {
		return PublicKey{}, errors.New("resourcepack: clé publique: deux lignes attendues")
	}
	bin, err := decodeStd(strings.TrimSpace(lines[1]))
	if err != nil || len(bin) != minisignKeyLen {
		return PublicKey{}, errors.New("resourcepack: clé publique: encodage invalide")
	}
	if string(bin[:2]) != minisignAlgoPrehashed && string(bin[:2]) != "Ed" {
		return PublicKey{}, errors.New("resourcepack: clé publique: algorithme inconnu")
	}
	var pk PublicKey
	copy(pk.keyID[:], bin[2:10])
	pk.key = ed25519.PublicKey(append([]byte(nil), bin[10:]...))
	return pk, nil
}

// signature : une signature minisign décodée.
type signature struct {
	keyID     [8]byte
	sig       []byte
	trusted   string
	globalSig []byte
}

// parseSignature décode la signature telle que latest.json ou le fichier `.sig` de Tauri la
// porte : le contenu base64 du fichier de signature minisign (quatre lignes).
func parseSignature(confValue string) (signature, error) {
	text, err := decodeStd(confValue)
	if err != nil {
		return signature{}, fmt.Errorf("resourcepack: signature: %w", err)
	}
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(string(text)), "\r\n", "\n"), "\n")
	if len(lines) < 4 || !strings.HasPrefix(lines[2], trustedPrefix) {
		return signature{}, errors.New("resourcepack: signature: quatre lignes attendues")
	}
	bin1, err1 := decodeStd(strings.TrimSpace(lines[1]))
	bin2, err2 := decodeStd(strings.TrimSpace(lines[3]))
	if err1 != nil || err2 != nil || len(bin1) != minisignSigLen || len(bin2) != minisignGlobalLen {
		return signature{}, errors.New("resourcepack: signature: encodage invalide")
	}
	s := signature{
		sig:       bin1[10:],
		trusted:   strings.TrimPrefix(lines[2], trustedPrefix),
		globalSig: bin2,
	}
	copy(s.keyID[:], bin1[2:10])
	if string(bin1[:2]) != minisignAlgoPrehashed {
		return signature{}, errors.New("resourcepack: signature: seule la forme préhachée (ED) est acceptée")
	}
	return s, nil
}

// Verify vérifie que sigValue (contenu base64 d'un fichier de signature) signe le contenu de r
// avec cette clé : signature du fichier puis signature globale du commentaire de confiance.
func (pk PublicKey) Verify(r io.Reader, sigValue string) error {
	s, err := parseSignature(sigValue)
	if err != nil {
		return err
	}
	if s.keyID != pk.keyID {
		return fmt.Errorf("%w: identifiant de clé différent", ErrBadSignature)
	}
	h, err := blake2b.New512(nil)
	if err != nil {
		return err
	}
	if _, err := io.Copy(h, r); err != nil {
		return fmt.Errorf("resourcepack: lecture pour la signature: %w", err)
	}
	if !ed25519.Verify(pk.key, h.Sum(nil), s.sig) {
		return fmt.Errorf("%w: contenu", ErrBadSignature)
	}
	global := append(append([]byte(nil), s.sig...), []byte(s.trusted)...)
	if !ed25519.Verify(pk.key, global, s.globalSig) {
		return fmt.Errorf("%w: commentaire de confiance", ErrBadSignature)
	}
	return nil
}

// VerifyFile vérifie la signature du fichier path.
func (pk PublicKey) VerifyFile(path, sigValue string) error {
	fh, err := os.Open(path) //nolint:gosec // archive téléchargée par ce programme
	if err != nil {
		return err
	}
	defer fh.Close()
	return pk.Verify(fh, sigValue)
}

// decodeStd décode du base64 standard en tolérant les sauts de ligne en fin de texte.
func decodeStd(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
}
