package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// vue_a_charges_execution.go — LES CHARGES DE LA VUE A GARDEES PAR UNE VALEUR D EXECUTION, ET CE QUI
// LA FIXE (lot VA de la campagne de grammaire, recherche R2 du 2026-10-04) : Script (15) et
// biped_throw_initiate (39). Memes conventions que `vue_a_charges.go`. Releve Ghidra
// `HaloInfinite.exe` HI_1_13_0, base 0x140000000, lecture seule
// (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/va_ghidra/`).
//
// # SCRIPT (15) : LE FILM PORTE LA VALEUR DE SON ENREGISTREUR
//
// Lecteur `FUN_14080bb4c` : si `FUN_1404f25f4()` (game_simulation == 2) : R(15) (`FUN_14080bd28`) ;
// puis R(13), n = R(10), puis `FUN_1406d676c(.., n)` = R(n) (R9D = la valeur de R(10) rangee en
// [+4], @14080bbf0). Ecrivain `FUN_142eec4d8` : si `FUN_1404f293c()` (game_simulation != 2) : W(15)
// (`FUN_141fd07f8`) ; W(13), W(10) n, `FUN_1406d60f4(.., n)`. `game_simulation` est le champ +4 des
// options de partie (`FUN_140be946c` le journalise sous ce nom ; enumeration `PTR_DAT_143cef4b0` :
// none, local, dist-client, dist-server). `chunk_00` serialise les options de l enregistreur en tete
// de son corps (`FUN_1407ec560`, [profile.FilmIdentity.SimulationDeLEnregistreur]). Le rejeu le
// confirme : `FUN_142e33478` pose, pour le mode reseau 5, game_simulation = 2 (dist-client) ET
// game_playback = 2 (replicated-film), le seul chemin qui pose ce playback ; le lecteur d un film
// rejoue lit donc le prefixe que l enregistreur a ecrit. La regle portee est celle de l ECRIVAIN,
// sur la valeur LUE dans le film ; sans options de partie lues, le Script ne se lit pas.
//
// # BIPED_THROW_INITIATE (39) : LA VALEUR SE LIT DANS LE LECTEUR LUI-MEME
//
// Lecteur `FUN_140c6a58c` et ecrivain `FUN_14104fc8c` : toute la charge est gardee par
// `*(DAT_144c1cfa8 + 4) == 2`, l etat du jeu principal (ecrit 2 par `FUN_140544ec8` au chargement
// reussi d une carte). Quand il ne vaut pas 2, le LECTEUR REND 0 : `FUN_14080a9d4` rend alors le code
// 3, et `FUN_14076a1c4` ARRETE la vue A. Un message 39 que le jeu accepte a donc ete lu avec sa
// charge.

// simulationDistClient est la valeur 2 de game_simulation (`PTR_DAT_143cef4b0[2]` = "dist-client").
const simulationDistClient = 2

// Les largeurs de `FUN_14080bb4c`.
const (
	largeurPrefixeScript  = 15 // FUN_14080bd28
	largeurTeteScript     = 13
	largeurLongueurScript = 10
)

// etatDuScript dit ce que le film declare du prefixe du message Script.
type etatDuScript uint8

// Les etats du prefixe du Script.
const (
	// scriptInconnu : options de partie non lues ; le Script ne se lit pas.
	scriptInconnu etatDuScript = iota
	// scriptSansPrefixe : enregistreur dist-client.
	scriptSansPrefixe
	// scriptAvecPrefixe : enregistreur d une autre simulation (dist-server, lu sur les films du parc).
	scriptAvecPrefixe
)

// scriptDuFilm applique la regle de l ecrivain a la simulation lue dans le film.
func scriptDuFilm(p profile.Profile) etatDuScript {
	if !p.IdentityRead() || !p.Identity().OptionsDePartieLues {
		return scriptInconnu
	}
	if p.Identity().SimulationDeLEnregistreur == simulationDistClient {
		return scriptSansPrefixe
	}
	return scriptAvecPrefixe
}

// chargeScript porte `FUN_14080bb4c` (`Script`).
func chargeScript(br *Lecteur) bool {
	switch br.vueA.script {
	case scriptInconnu:
		return false
	case scriptAvecPrefixe:
		br.Skip(largeurPrefixeScript)
	}
	br.Skip(largeurTeteScript)
	br.Skip(int(br.ReadBits(largeurLongueurScript)))
	return true
}

// chargeLancerInitie porte `FUN_140c6a58c` (`biped_throw_initiate`) : k = R(1) (`FUN_1424d9b10`) ;
// k = 0 : R(3) (`FUN_1424d9a30`) ; k = 1 : `FUN_14080d69c` puis R(4) (`FUN_142ed0674`) ; puis
// `FUN_1407f2058`.
func chargeLancerInitie(br *Lecteur) bool {
	if br.ReadBit() {
		consumeGateR(br, 32)
		br.Skip(4)
	} else {
		br.Skip(3)
	}
	consumeGate0R(br, 5)
	return true
}
