package grammar

import "levelup/go-api/internal/games/halo_infinite/film/internal/profile"

// vue_a_charges_execution.go — LES CHARGES DE LA VUE A GARDEES PAR UNE VALEUR D EXECUTION, ET CE QUI
// LA FIXE (lot VA de la campagne de grammaire, recherche R2 du 2026-10-04 et etape V3) : Script (15),
// biped_throw_initiate (39), PlayerKilledEvent (85) et teleport_effects (116). Memes conventions que
// `vue_a_charges.go`. Releve Ghidra `HaloInfinite.exe` HI_1_13_0, base 0x140000000, lecture seule
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
// de son corps (`FUN_1407ec560`, [profile.FilmIdentity.SimulationDeLEnregistreur]). En rejeu,
// `FUN_142e33478` pose, pour le mode reseau 5, game_simulation = 2 (dist-client) ET game_playback = 2
// (replicated-film), le seul chemin qui pose ce playback : `FUN_1404f25f4` y est toujours vrai, et le
// lecteur du jeu lit TOUJOURS le R(15). Un film rejoue n est donc lisible par le jeu que si son
// enregistreur a ecrit le prefixe (simulation != 2). Pour un enregistreur a simulation 2, l ecrivain
// n ecrit pas ces 15 bits et le lecteur de rejeu les lirait quand meme : la regle portee
// ([scriptSansPrefixe]) est celle de l ECRIVAIN, et elle contredit alors le lecteur de rejeu. Aucun
// film du cache n exerce cette branche (simulation 3, dist-server, sur les 1 657 films a section
// d identification, sonde de la revue du lot VA). La valeur est LUE dans le film ; sans options de
// partie lues, le Script ne se lit pas.
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

// # PLAYERKILLEDEVENT (85) ET TELEPORT_EFFECTS (116) : CE QUE LA VARIANTE DE PARTIE DU FILM DECIDE
//
// Lecteur du 85 `FUN_14104bd08` : `FUN_1407f2058` x 2, R(32), R(1), `FUN_1407f2058`, R(32) ; puis la
// queue `FUN_1431eb378` (R(32), R(32), R(4)) si `FUN_14076d018() || FUN_14076cffc()` — la garde de
// l ecrivain `FUN_142f18fd0`, a l identique :
//
//	FUN_14076d018 = DAT_1451789b8 && FUN_1406aed00() && DAT_145121140 != 1 && variante[+0x238]
//	FUN_14076cffc = DAT_145178a48 && FUN_1406aed00() && variante[+0x240]
//
// `DAT_1451789b8` et `DAT_145178a48` sont les reglages nommes `kill_playback_enabled` et
// `play_of_the_game_enabled` (`FUN_140373a60`, `FUN_140373b40`), poses a l execution : le film ne les
// porte pas. `variante[+0x238]` et `[+0x240]` sont killcamEnabled et playOfTheGameEnabled, et
// `DAT_145121140` le type de l objet moteur, `FUN_14051a4b8(m_gameEngineType)` (`FUN_140a938b4`), qui
// vaut 1 si et seulement si m_gameEngineType vaut 1 : trois valeurs que le film porte
// ([profile.VarianteDePartie]). Quand `(killcam && moteur != 1) || playOfTheGame` est faux, la garde
// est fausse quels que soient les reglages : le 85 n a pas de queue. Sinon il ne se lit pas.
//
// Le troisieme terme, `FUN_1406aed00()`, n est PAS lu jusqu au bout. Il vaut vrai quand un etat de
// fil (`TLS + 0x238`, deux octets non tous nuls) est pose et que `FUN_1406aed60(options)` rend 2, sur
// les options de la partie courante (`DAT_1445c5838 * 0x1134F0 + DAT_145121d28`). `FUN_1406aed60` rend
// `options[0]` (game_mode, le premier R(3) du corps de `chunk_00`, que le film porte) quand l octet
// `options + 0xE2EE1` vaut 0, et 1 sinon. Le film porte game_mode (2 sur les 1 657 films du cache a
// section d identification, mesure par la sonde de la revue) ; l octet `+ 0xE2EE1` n est pas ecrit en
// clair par le lecteur du corps (`FUN_1407ee138`) et reste a localiser (dans la variante Bond, a
// lire). Un film dont game_mode != 2, ou dont cet octet vaut 1, rendrait la garde fausse quels que
// soient les reglages : son 85 se lirait par une regle lue. Tant que l octet n est pas lu, la regle
// ci-dessus n en tient pas compte : elle refuse plus qu il ne faut, jamais moins.
//
// Lecteur du 116 `FUN_142ef93e0` : R(1) ; si 1 : `FUN_140c5f938(.., mode 0)` ; R(1) ; `FUN_14080d69c`
// (rend son R(1)) ; si 1 : deux positions `FUN_1424e0e38` = `FUN_14076e494(.., 0x10, .., p6 = 0)`.
// `FUN_140c5f938` en mode 0 lit `FUN_140c5fa84` quand `DAT_145121140 != 1`, `FUN_142e29bac` sinon ;
// l ecrivain `FUN_142efa2a8` -> `FUN_141f86118` prend la meme branche. Le type de moteur du film la
// decide ; la branche `FUN_142e29bac` n est pas portee.

// typeDeMoteurUn est la valeur de m_gameEngineType que `FUN_14051a4b8` envoie sur le type de moteur 1
// (`DAT_145121140 == 1`) : 1 -> 1, 2 -> 3, 3 -> 2, autre -> 0.
const typeDeMoteurUn = 1

// varianteDeLaVueA est ce que la variante de partie du film decide pour les charges 85 et 116.
type varianteDeLaVueA struct {
	// lue : la variante est presente dans le film et lue ; faux : ni le 85 ni le 116 ne se lisent.
	lue bool
	// moteurUn : `DAT_145121140 == 1`.
	moteurUn bool
	// queueDuKillPossible : la garde de la queue du 85 peut etre vraie selon des reglages que le
	// film ne porte pas.
	queueDuKillPossible bool
}

// varianteDuFilm derive de l identite d un profil ce que sa variante de partie decide.
func varianteDuFilm(p profile.Profile) varianteDeLaVueA {
	if !p.IdentityRead() {
		return varianteDeLaVueA{}
	}
	v := p.Identity().Variante
	if !v.Lue || !v.Presente {
		return varianteDeLaVueA{}
	}
	moteurUn := v.TypeDeMoteur == typeDeMoteurUn
	return varianteDeLaVueA{lue: true, moteurUn: moteurUn,
		queueDuKillPossible: (v.KillcamEnabled && !moteurUn) || v.PlayOfTheGameEnabled}
}

// chargeJoueurTue porte `FUN_14104bd08` (`PlayerKilledEvent`) quand le film decide que sa queue est
// absente.
func chargeJoueurTue(br *Lecteur) bool {
	if v := br.vueA.variante; !v.lue || v.queueDuKillPossible {
		return false
	}
	consumeGate0R(br, 5) // FUN_1407f2058 : victime
	consumeGate0R(br, 5) // FUN_1407f2058 : tueur
	br.Skip(32 + 1)      // [+8], [+0xc]
	consumeGate0R(br, 5) // FUN_1407f2058 : assistant
	br.Skip(32)          // [+0x14]
	return true
}

// chargeEffetsDeTeleportation porte `FUN_142ef93e0` (`teleport_effects`).
func chargeEffetsDeTeleportation(br *Lecteur) bool {
	if br.ReadBit() {
		if v := br.vueA.variante; !v.lue || v.moteurUn {
			return false
		}
		consumeObjectForwardAndUp(br) // FUN_140c5f938(mode 0) -> FUN_140c5fa84
	}
	br.Skip(1)
	if !br.ReadBit() { // FUN_14080d69c
		return true
	}
	br.Skip(32)                                                           // FUN_14080d6f0
	if _, ok := lireE494Sur(br, niveauPosition, br.vueA.positions); !ok { // FUN_1424e0e38 [+0x20]
		return false
	}
	_, ok := lireE494Sur(br, niveauPosition, br.vueA.positions) // FUN_1424e0e38 [+0x2c]
	return ok
}
