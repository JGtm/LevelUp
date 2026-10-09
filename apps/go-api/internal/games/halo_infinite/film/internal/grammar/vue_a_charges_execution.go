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

// # PLAYERKILLEDEVENT (85) : LA LECTURE TIENT LES DEUX REGLAGES DE LA QUEUE A LEUR DEFAUT
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
// porte pas. L executable les enregistre a FAUX (`FUN_140ad2d08(.., 0)`) et son code ne fait que les
// lire. LA LECTURE LES TIENT A CE DEFAUT (decision de l utilisateur du 2026-10-07) : la garde est
// fausse quels que soient la variante de partie (`variante[+0x238]`, `[+0x240]` : killcamEnabled et
// playOfTheGameEnabled) et le troisieme terme (`FUN_1406aed00`), et le 85 se lit sans queue. C est une
// valeur presumee : un film enregistre sous l un des deux reglages leve porterait la queue, la lecture
// continuerait faux apres son 85, et la fin de la vue A ne deciderait du debut de la vue B que sous la
// regle de la classe du film ([debutParLaVueA]).
//
// # TELEPORT_EFFECTS (116) : CE QUE LA VARIANTE DE PARTIE DU FILM DECIDE
//
// Lecteur du 116 `FUN_142ef93e0` : R(1) ; si 1 : `FUN_140c5f938(.., mode 0)` ; R(1) ; `FUN_14080d69c`
// (rend son R(1)) ; si 1 : deux positions `FUN_1424e0e38` = `FUN_14076e494(.., 0x10, .., p6 = 0)`.
// `FUN_140c5f938` en mode 0 lit `FUN_140c5fa84` quand `DAT_145121140 != 1`, `FUN_142e29bac` sinon ;
// l ecrivain `FUN_142efa2a8` -> `FUN_141f86118` prend la meme branche. `DAT_145121140` est le type de
// l objet moteur, `FUN_14051a4b8(m_gameEngineType)` (`FUN_140a938b4`), qui vaut 1 si et seulement si
// m_gameEngineType vaut 1 : une valeur que le film porte ([profile.VarianteDePartie]). Le type de
// moteur du film decide la branche ; la branche `FUN_142e29bac` n est pas portee.

// typeDeMoteurUn est la valeur de m_gameEngineType que `FUN_14051a4b8` envoie sur le type de moteur 1
// (`DAT_145121140 == 1`) : 1 -> 1, 2 -> 3, 3 -> 2, autre -> 0.
const typeDeMoteurUn = 1

// varianteDeLaVueA est ce que la variante de partie du film decide pour la charge 116.
type varianteDeLaVueA struct {
	// lue : la variante est presente dans le film et lue ; faux : le 116 ne se lit pas.
	lue bool
	// moteurUn : `DAT_145121140 == 1`.
	moteurUn bool
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
	return varianteDeLaVueA{lue: true, moteurUn: v.TypeDeMoteur == typeDeMoteurUn}
}

// GenreJoueurTue est le genre du message `PlayerKilledEvent` de la vue A.
const GenreJoueurTue = 85

// chargeJoueurTue porte `FUN_14104bd08` (`PlayerKilledEvent`) sans sa queue (cf. plus haut) et range
// ses champs sur le lecteur ([Lecteur.killLu]), que la lecture de la vue A recueille.
func chargeJoueurTue(br *Lecteur) bool {
	k := &br.killLu
	k.Victime = int8(readOpt5Signed(br))         //nolint:gosec // FUN_1407f2058 : victime, R(5) ou -1
	k.Tueur = int8(readOpt5Signed(br))           //nolint:gosec // FUN_1407f2058 : tueur, R(5) ou -1
	k.PartDuTueur = uint32(br.ReadBits(32))      //nolint:gosec // [+8], R(32)
	k.Drapeau = uint8(br.ReadBits(1))            //nolint:gosec // [+0xc], R(1)
	k.Assistant = int8(readOpt5Signed(br))       //nolint:gosec // FUN_1407f2058 : assistant, R(5) ou -1
	k.PartDeLAssistant = uint32(br.ReadBits(32)) //nolint:gosec // [+0x14], R(32)
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
