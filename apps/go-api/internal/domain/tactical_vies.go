package domain

// tactical_vies.go — LE CHOIX DES APPARITIONS QUI FORMENT LES GRAPPES DE REAPPARITION.
//
// Deux vocabulaires, et la valeur par defaut ne change rien a ce qui existait :
//
//   - `depart` : seule la PREMIERE vie du joueur compte, une par match. La grappe dit « d'ou
//     je pars au debut du match », et le filtre de grappe restreint l'univers aux matchs
//     partis de cet amas.
//   - `toutes` : TOUTES les apparitions du joueur comptent — premiere vie, reapparitions
//     apres une mort, debuts de manche. La grappe dit « ou je nais, quelle que soit la vie ».
//     Sous un filtre de grappe, la lecture « routes » ne retient que les routes des vies qui
//     partent de cet amas.
//
// Une piste du film qui n'est ni une premiere vie, ni une reapparition, ni un debut de manche
// (une vie coupee en deux par un trou de replication) n'est PAS une apparition : elle n'entre
// dans aucune grappe, sous aucun des deux vocabulaires.

import (
	"errors"
	"fmt"
)

// Les deux valeurs du parametre `vies`.
const (
	TacticalViesDepart = "depart"
	TacticalViesToutes = "toutes"
)

// ErrTacticalViesInconnue : le parametre `vies` n'est ni `depart` ni `toutes`. Une valeur
// presente mais hors vocabulaire est REFUSEE (400), jamais remplacee par le defaut : servir
// `depart` a qui a demande autre chose repondrait a une autre question sans le dire.
var ErrTacticalViesInconnue = errors.New("tactique: vies inconnues (depart | toutes)")

// NormaliserVies rend la valeur canonique du parametre `vies` : vide vaut `depart`, une valeur
// hors vocabulaire est refusee.
func NormaliserVies(vies string) (string, error) {
	switch vies {
	case "", TacticalViesDepart:
		return TacticalViesDepart, nil
	case TacticalViesToutes:
		return TacticalViesToutes, nil
	default:
		return "", fmt.Errorf("%w (%q)", ErrTacticalViesInconnue, vies)
	}
}
