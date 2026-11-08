// Package identitefictive derive les identifiants Xbox FICTIFS des fixtures de film.
//
// Les films de test du decodeur portaient les xuids de joueurs reels. Ils sont remplaces, a
// LONGUEUR CONSTANTE (64 bits, 16 chiffres decimaux), par des identifiants fictifs tires par une
// fonction pure : meme xuid reel en entree, meme xuid fictif en sortie, sans etat ni table.
//
// # LA PLAGE
//
// Un xuid fictif reste dans `[PlageBas, PlageHaut[`. Le decodeur ne repere la table des joueurs
// d'un film (`grammar.balayerCandidats`) que sur un xuid de cette plage : hors d'elle la table n'est
// plus trouvee. La plage est aussi celle des xuids reels, si bien qu'un fictif a la meme forme qu'un
// vrai (16 chiffres, meme longueur binaire).
//
// # LA DERIVATION
//
// fictif = PlageBas + (premiers 8 octets de SHA-256(graine || reel sur 8 octets gros-boutiste), lus
// gros-boutiste, masques sur 48 bits). La graine est PUBLIQUE : la fonction ne cache pas l'identite
// d'origine a qui possede deja un xuid, elle ne fait qu'empecher de la LIRE dans le depot. La liste
// des xuids d'origine, dont la derivation a besoin pour s'inverser, vit hors du depot.
//
// Deux xuids d'origine peuvent en theorie rendre le meme fictif (48 bits) : l'outil
// `cmd/anonymise-gamertags` verifie l'injectivite sur l'ensemble qu'il traite et refuse sinon.
package identitefictive

import (
	"crypto/sha256"
	"encoding/binary"
)

const (
	// PlageBas est le plus petit xuid fictif (inclus).
	PlageBas = uint64(0x0009000000000000)
	// PlageHaut borne la plage (exclu).
	PlageHaut = uint64(0x000A000000000000)

	graine   = "levelup/identifiants-de-test-fictifs/v1"
	masque48 = uint64(1)<<48 - 1
)

// Xuid rend le xuid fictif de `reel`.
func Xuid(reel uint64) uint64 {
	var entree [8]byte
	binary.BigEndian.PutUint64(entree[:], reel)
	h := sha256.New()
	_, _ = h.Write([]byte(graine))
	_, _ = h.Write(entree[:])
	return PlageBas + binary.BigEndian.Uint64(h.Sum(nil)[:8])&masque48
}

// DansLaPlage dit si `x` est dans la plage des xuids (reels ou fictifs).
func DansLaPlage(x uint64) bool { return x >= PlageBas && x < PlageHaut }
