package resourcepack

import (
	"errors"
	"io/fs"
)

// State : le verdict du contrôle de démarrage.
type State string

const (
	// StateUnchecked : le moteur n'embarque aucune empreinte (build de développement ou de test) :
	// aucun contrôle n'a lieu.
	StateUnchecked State = "unchecked"
	// StateMatch : le paquet installé est celui que le moteur attend.
	StateMatch State = "match"
	// StateMismatch : un paquet est installé mais ce n'est pas celui du moteur.
	StateMismatch State = "mismatch"
	// StateAbsent : aucun manifeste sous la racine de ressources.
	StateAbsent State = "absent"
	// StateInvalid : le manifeste existe mais est illisible ou incohérent.
	StateInvalid State = "invalid"
)

// Verdict : ce que le contrôle a trouvé.
type Verdict struct {
	State State
	// Expected : l'empreinte embarquée par le moteur.
	Expected string
	// Active : l'empreinte du paquet installé ("" si absent ou invalide).
	Active string
	// Err : la cause pour StateInvalid.
	Err error
}

// Check compare le paquet installé sous root à l'empreinte expected du moteur. Il ne lit que le
// manifeste : la vérification fichier par fichier se fait au téléchargement, jamais à chaque
// démarrage (150 Mo). expected vide : aucun contrôle.
func Check(root, expected string) Verdict {
	v := Verdict{Expected: expected}
	if expected == "" {
		v.State = StateUnchecked
		return v
	}
	m, err := ReadManifest(root)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		v.State = StateAbsent
	case err != nil:
		v.State, v.Err = StateInvalid, err
	case m.Fingerprint != expected:
		v.State, v.Active = StateMismatch, m.Fingerprint
	default:
		v.State, v.Active = StateMatch, m.Fingerprint
	}
	return v
}
