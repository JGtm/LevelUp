package grammar

// vue_a_charges_sacs.go — LES CHARGES A SAC DE PROPRIETES NOMMEES DE LA VUE A (lot LN de la
// campagne de grammaire) : `PlayerGameEventSmall` (genre 82).
//
// Le sac principal est `FUN_14080b1b8` : `n = R(3)`, puis `n` fois un nom `R(32)` et une valeur
// etiquetee (`FUN_14080ef08` : etiquette `R(3)`, puis `FUN_14080eff0`). Le sous-sac est
// `FUN_14080b034`, porte une seule fois dans le depot ([consumeSacTexte]).

// largeurEtiquetteDePropriete est le `R(3)` de `FUN_14080ef08` et le compte `R(3)` de
// `FUN_14080b1b8`.
const largeurEtiquetteDePropriete = 3

// longueurChaineDePropriete est le quatrieme argument `0x10` de `FUN_1407cbc24` a l etiquette 5.
const longueurChaineDePropriete = 16

// chargeEvenementJoueurCourt porte `FUN_14080add8` (`PlayerGameEventSmall`) : `FUN_14080b30c`
// (initialisation, aucun bit), `FUN_14080ae70` (R(32), R(8), le sac, le sous-sac), puis
// `FUN_14080ae28` (32 x R(1)).
func chargeEvenementJoueurCourt(br *Lecteur) bool {
	if !lireEnTeteDEvenementDeJeu(br, largeurCompteSacCourt) {
		return false
	}
	br.Skip(32)
	return true
}

// lireSacDeProprietes porte `FUN_14080b1b8` (compte sur `largeurCompte` = 3 bits) et `FUN_142efb634`
// (4 bits, dont l element est `FUN_140f8d4d0` : le meme nom puis `FUN_14080ef08`).
func lireSacDeProprietes(br *Lecteur, largeurCompte uint) bool {
	n := br.ReadBits(largeurCompte)
	for range n {
		br.Skip(32) // FUN_14080dec4 « property-name »
		if !lireValeurDePropriete(br, br.ReadBits(largeurEtiquetteDePropriete)) {
			return false
		}
	}
	return true
}

// lireValeurDePropriete porte `FUN_14080eff0` : la valeur d une propriete selon son etiquette.
func lireValeurDePropriete(br *Lecteur, etiquette uint64) bool {
	switch etiquette {
	case 0: // rien n est lu
	case 1, 2, 3, 6: // FUN_140f2734c, FUN_14080dec4 « string-id-value », FUN_14080f0c4, FUN_140f04e28
		br.Skip(32)
	case 4:
		br.Skip(1) // FUN_1406cf008
	case 5:
		return lireChaine(br, longueurChaineDePropriete)
	default: // FUN_140f04f18 : FUN_14076f91c ? R(96) : FUN_14076e524(..., 0x10)
		lireE494(br, niveauPosition)
	}
	return true
}

// lireChaine porte `FUN_1407cbc24(flux, 0, dst, n)` : des octets R(8) jusqu au premier nul, `n` au
// plus. `n` octets sans nul levent l erreur du flux (`flux + 0x24 = 1`) : la lecture est alors
// refusee.
func lireChaine(br *Lecteur, n int) bool {
	for range n {
		if br.ReadBits(8) == 0 {
			return true
		}
	}
	return false
}

// Les largeurs du compte du sac principal : `FUN_14080b1b8` (evenements courts) et
// `FUN_142efb634` (evenements longs).
const (
	largeurCompteSacCourt uint = 3
	largeurCompteSacLong  uint = 4
)

// lireEnTeteDEvenementDeJeu porte `FUN_14080ae70` (sac court) et `FUN_142efb480` (sac long) :
// R(32), R(8), le sac principal, puis le sous-sac `FUN_14080b034`.
func lireEnTeteDEvenementDeJeu(br *Lecteur, largeurCompte uint) bool {
	br.Skip(32 + 8)
	if !lireSacDeProprietes(br, largeurCompte) {
		return false
	}
	consumeSacTexte(br)
	return true
}

// largeurQueueDEquipe est le R(9) de `FUN_140f58324`, la queue des evenements d equipe.
const largeurQueueDEquipe = 9

// chargeEvenementJoueur porte `FUN_142f164f4` (`PlayerGameEvent`) : `FUN_142efe2c4`
// (initialisation, aucun bit), `FUN_142efb480`, puis `FUN_14080ae28` (32 x R(1)).
func chargeEvenementJoueur(br *Lecteur) bool {
	if !lireEnTeteDEvenementDeJeu(br, largeurCompteSacLong) {
		return false
	}
	br.Skip(32)
	return true
}

// chargeEvenementDEquipe porte `FUN_142f1686c` (`TeamGameEvent`) : `FUN_142efb480` puis
// `FUN_140f58324`.
func chargeEvenementDEquipe(br *Lecteur) bool {
	if !lireEnTeteDEvenementDeJeu(br, largeurCompteSacLong) {
		return false
	}
	br.Skip(largeurQueueDEquipe)
	return true
}

// chargeEvenementDEquipeCourt porte `FUN_142f16818` (`TeamGameEventSmall`) : `FUN_14080ae70` puis
// `FUN_140f58324`.
func chargeEvenementDEquipeCourt(br *Lecteur) bool {
	if !lireEnTeteDEvenementDeJeu(br, largeurCompteSacCourt) {
		return false
	}
	br.Skip(largeurQueueDEquipe)
	return true
}

// largeurTexteDeDebogage et longueurChaineDeDebogage sont les immediats de `FUN_142eebf1c`
// (`ShowDebugText`) : `FUN_1406d676c(..., 0x280)` et `FUN_1407cbc24(..., 0x60)`.
const (
	largeurTexteDeDebogage   = 0x280
	longueurChaineDeDebogage = 0x60
)

// chargeTexteDeDebogage porte `FUN_142eebf1c` : R(32), R(0x280), puis une chaine de 0x60 octets
// au plus ([lireChaine]).
func chargeTexteDeDebogage(br *Lecteur) bool {
	br.Skip(32 + largeurTexteDeDebogage)
	return lireChaine(br, longueurChaineDeDebogage)
}

// chargeDEvenementDeJeu rend le lecteur de charge d un evenement de jeu porte ici ; nil sinon.
func chargeDEvenementDeJeu(genre int) func(*Lecteur) bool {
	switch genre {
	case 16:
		return chargeTexteDeDebogage
	case 81:
		return chargeEvenementJoueur
	case 82:
		return chargeEvenementJoueurCourt
	case 83:
		return chargeEvenementDEquipe
	case 84:
		return chargeEvenementDEquipeCourt
	}
	return nil
}
