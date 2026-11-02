// Package tokencipher : le port de chiffrement des fichiers de jetons `watcher_tokens/{xuid}.json`
// (ADR 0045 D-11, qui amende ADR 0023 : la source unique des jetons reste ce fichier, son contenu
// sensible peut être scellé).
//
// Deux implémentations : [Plain] (octets inchangés : instance hébergée, conteneur Linux, poste de
// développement) et DPAPI (Windows seulement, coffre du compte courant). Le choix revient au profil
// de déploiement, jamais au système d'exploitation seul : la capacité TokenVault de
// config.DeploymentCapabilities l'exprime, et [InstallForProfile] est appelée par le chargement de la
// configuration, nulle part ailleurs (archlint/token_cipher_single_decision_test.go).
//
// Le paquet est une feuille : il n'importe ni la configuration ni le magasin de jetons, qui
// l'importent tous deux.
package tokencipher

import (
	"errors"
	"log/slog"
	"sync/atomic"
)

// Noms des implémentations, tels qu'écrits dans le champ `cipher` d'un fichier de jetons.
const (
	// NamePlain : jetons en clair. Absent du fichier (un fichier sans champ `cipher` est `plain`), pour
	// que les fichiers d'une instance hébergée restent identiques octet pour octet.
	NamePlain = "plain"
	// NameDPAPI : champs sensibles scellés par le coffre Windows du compte courant.
	NameDPAPI = "dpapi"
)

// ErrUnavailable : l'implémentation demandée n'existe pas sur ce système (DPAPI hors Windows).
var ErrUnavailable = errors.New("tokencipher: coffre indisponible sur ce système")

// Cipher scelle et ouvre des octets. Un scellé n'est lisible que par l'implémentation qui l'a produit,
// dans le même contexte de compte : un fichier copié ailleurs échoue à Open, et c'est voulu.
type Cipher interface {
	// Name rend le nom écrit dans le champ `cipher` du fichier ([NamePlain], [NameDPAPI]).
	Name() string
	// Seal chiffre plain.
	Seal(plain []byte) ([]byte, error)
	// Open rend le clair d'un scellé produit par Seal.
	Open(sealed []byte) ([]byte, error)
}

// plainCipher : octets inchangés.
type plainCipher struct{}

func (plainCipher) Name() string                  { return NamePlain }
func (plainCipher) Seal(b []byte) ([]byte, error) { return append([]byte(nil), b...), nil }
func (plainCipher) Open(b []byte) ([]byte, error) { return append([]byte(nil), b...), nil }

// Plain rend l'implémentation sans chiffrement.
func Plain() Cipher { return plainCipher{} }

// NewDPAPI rend l'implémentation DPAPI (portée : utilisateur courant). Hors Windows : [ErrUnavailable].
func NewDPAPI() (Cipher, error) { return newDPAPI() }

// ForVault rend le chiffreur d'un profil : DPAPI si le profil offre le coffre, sinon [Plain].
func ForVault(vault bool) (Cipher, error) {
	if vault {
		return NewDPAPI()
	}
	return Plain(), nil
}

// processState : le chiffreur du processus et l'échec éventuel de sa construction.
type processState struct {
	cipher Cipher
	err    error
}

var process atomic.Pointer[processState]

// Process rend le chiffreur du processus : celui du dernier profil résolu, [Plain] tant qu'aucun ne
// l'a été. Les magasins de jetons le lisent à leur construction.
func Process() Cipher {
	if st := process.Load(); st != nil {
		return st.cipher
	}
	return plainCipher{}
}

// ProcessError rend l'échec de la dernière installation (le profil demande le coffre et le système ne
// l'offre pas), nil sinon. Le serveur refuse de démarrer sur cette erreur : le profil desktop est
// Windows seul (ADR 0045 D-2). Les autres consommateurs de la configuration (tests, outils) gardent
// le chiffreur [Plain].
func ProcessError() error {
	if st := process.Load(); st != nil {
		return st.err
	}
	return nil
}

// InstallForProfile pose le chiffreur du processus d'après la capacité TokenVault du profil résolu.
// Si le coffre est demandé et indisponible, l'erreur est journalisée, conservée pour [ProcessError] et
// le chiffreur reste [Plain] : cette fonction ne fait jamais échouer le chargement de la configuration.
func InstallForProfile(vault bool) {
	c, err := ForVault(vault)
	if err != nil {
		slog.Error("tokencipher: coffre demandé par le profil mais indisponible", "err", err)
		c = plainCipher{}
	}
	process.Store(&processState{cipher: c, err: err})
}

// InstallProcess pose un chiffreur explicite (tests).
func InstallProcess(c Cipher) { process.Store(&processState{cipher: c}) }
