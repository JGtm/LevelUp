// Command updater-verify : verifie qu'un fichier .sig produit par le CLI Tauri correspond a la cle publique
// embarquee (plugins.updater.pubkey de tauri.conf.json). Un secret TAURI_SIGNING_PRIVATE_KEY different de la
// cle embarquee n'est signale que par un avertissement du CLI : la release lance cet outil avant de publier
// (ADR 0045 D-16).
// Usage : updater-verify -pubkey <base64 | fichier .pub | tauri.conf.json> -sig <fichier .sig> -file <installeur>
//
// Formats minisign : cle publique = alg[2] || key_id[8] || pk[32] ; signature = alg[2] || key_id[8] || sig[64].
// Tauri encode en base64 le fichier .pub entier dans tauri.conf.json, et le fichier .sig entier de meme.
// alg "ED" : signature de BLAKE2b-512 du fichier ; "Ed" : du fichier brut. Le commentaire de confiance et sa
// signature globale ne sont pas verifies.
package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/blake2b"
)

const (
	commentPrefix = "untrusted comment:"
	algRaw        = "Ed"
	algPrehashed  = "ED"
	keyIDLen      = 8
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("updater-verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pubkey := fs.String("pubkey", "", "cle publique : base64, fichier .pub ou tauri.conf.json")
	sigPath := fs.String("sig", "", "fichier .sig de l'installeur")
	filePath := fs.String("file", "", "installeur signe")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *pubkey == "" || *sigPath == "" || *filePath == "" {
		fmt.Fprintln(stderr, "updater-verify : -pubkey, -sig et -file sont obligatoires")
		return 2
	}
	pubText, err := loadPubkey(*pubkey)
	if err != nil {
		fmt.Fprintf(stderr, "updater-verify : cle publique illisible : %v\n", err)
		return 1
	}
	sigRaw, err := os.ReadFile(*sigPath)
	if err != nil {
		fmt.Fprintf(stderr, "updater-verify : signature illisible : %v\n", err)
		return 1
	}
	content, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(stderr, "updater-verify : fichier illisible : %v\n", err)
		return 1
	}
	if err := verify(pubText, string(sigRaw), content); err != nil {
		fmt.Fprintf(stderr, "updater-verify : ECHEC : %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "updater-verify : la signature de %s correspond a la cle publique embarquee\n", *filePath)
	return 0
}

// loadPubkey rend le texte du fichier .pub depuis le premier argument : un tauri.conf.json, un fichier
// .pub, ou la valeur base64 elle-meme.
func loadPubkey(arg string) (string, error) {
	text := arg
	if raw, err := os.ReadFile(arg); err == nil {
		text = strings.TrimSpace(string(raw))
		if strings.HasPrefix(text, "{") {
			var conf struct {
				Plugins struct {
					Updater struct {
						Pubkey string `json:"pubkey"`
					} `json:"updater"`
				} `json:"plugins"`
			}
			if err := json.Unmarshal(raw, &conf); err != nil || conf.Plugins.Updater.Pubkey == "" {
				return "", errors.New("plugins.updater.pubkey absent du fichier JSON")
			}
			text = conf.Plugins.Updater.Pubkey
		}
	}
	return unwrap(text)
}

// unwrap rend le texte minisign : tel quel s'il commence par le commentaire, sinon decode de base64.
func unwrap(text string) (string, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, commentPrefix) {
		return text, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil || !strings.HasPrefix(string(decoded), commentPrefix) {
		return "", errors.New("ni un texte minisign ni son encodage base64")
	}
	return string(decoded), nil
}

// payloadLine rend la premiere ligne decodee en base64 qui suit le commentaire d'un texte minisign.
func payloadLine(text string, want int) (alg string, keyID, body []byte, err error) {
	_, rest, _ := strings.Cut(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	line, _, _ := strings.Cut(rest, "\n")
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(line))
	if err != nil || len(raw) != 2+keyIDLen+want {
		return "", nil, nil, fmt.Errorf("ligne de donnees minisign invalide (attendu %d octets)", 2+keyIDLen+want)
	}
	return string(raw[:2]), raw[2 : 2+keyIDLen], raw[2+keyIDLen:], nil
}

// verify controle l'identifiant de cle ET la signature ed25519 de content.
func verify(pubText, sigRaw string, content []byte) error {
	_, pubKeyID, pk, err := payloadLine(pubText, ed25519.PublicKeySize)
	if err != nil {
		return fmt.Errorf("cle publique : %w", err)
	}
	sigText, err := unwrap(sigRaw)
	if err != nil {
		return fmt.Errorf("signature : %w", err)
	}
	alg, sigKeyID, sig, err := payloadLine(sigText, ed25519.SignatureSize)
	if err != nil {
		return fmt.Errorf("signature : %w", err)
	}
	if !bytes.Equal(pubKeyID, sigKeyID) {
		return fmt.Errorf("la signature est de la cle %X, la cle publique embarquee est %X : le secret TAURI_SIGNING_PRIVATE_KEY n'est pas celui de la cle publique (rotation non faite ? voir l'ADR 0045 D-16)", sigKeyID, pubKeyID)
	}
	message := content
	switch alg {
	case algRaw:
	case algPrehashed:
		sum := blake2b.Sum512(content)
		message = sum[:]
	default:
		return fmt.Errorf("algorithme de signature %q inconnu", alg)
	}
	if !ed25519.Verify(pk, message, sig) {
		return errors.New("signature invalide pour ce fichier (fichier modifie, ou signature d'une autre cle de meme identifiant)")
	}
	return nil
}
