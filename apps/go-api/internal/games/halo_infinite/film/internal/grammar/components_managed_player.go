package grammar

// components_managed_player.go — LES COMPOSANTS DE L ARCHETYPE `managed-player` (ti=9), LOT 3.6.a.
//
// # CE QUE CE FICHIER FERME
//
// `ti=9` portait 8 composants sur 10, et sa fermeture d image-cle valait 0/1 717 sur les sept
// bobines du ratchet 0.A.3 — UN SEUL composant en cause, `i4`, nomme par le golden comme le
// bloquant des sept bobines. Dans une image-cle il n y a pas de masque de presence : tous les
// composants de l archetype sont la, dans l ordre du registre, donc un composant sans lecteur
// bloque tout ce qui le suit. `i4` et `i9` sont les deux derniers, et ils sont ici.
//
// # LES GRAMMAIRES SONT RELEVEES CHEZ L ECRIVAIN, PAS MESUREES
//
// Un composant porte est une fonction pure (profil, bits) : ce fichier ne lit aucune table du
// jeu, aucun etat de processus, et n a AUCUNE entree de profil — les grammaires portees ici sont
// des largeurs LITTERALES d instruction. Sources, sous Ghidra en lecture seule :
//
//	i4  `.ai/V7.5/film_re/NOTE_3_6_TI9_2026-09-16.md` §2 — ecrivain `FUN_142ed5bc8`
//	    (descripteur `0x143d08840`, ecrivain a `descripteur + 0x40` = `0x143d08880`).
//	i9  `.ai/V7.5/film_re/NOTE_3_6_TI9_TI35_RESTES_2026-09-17.md` §1 — ecrivain
//	    `FUN_141fcf160` (descripteur `0x143d08930`, `+0x40`), son sac texte `FUN_14080b034`
//	    et le corps d emplacement `FUN_1407f0ebc`. Grammaire VERIFIEE par un sceptique
//	    independant (20 points de controle, 19 concordants ; l unique discordance portait sur
//	    trois adresses de SLOTS du descripteur, aucune largeur — synthese
//	    `NOTE_3_6_SYNTHESE_2026-09-17.md` §10.2).

// Etiquettes de registre des composants de ti=9 portes ici.
const (
	compManagedPlayerForgeWeather = "managed-player-forge-weather-effect-overrides-component"
	compManagedPlayerInputPrompt  = "managed-player-custom-input-prompt-widget"
)

// consumeManagedPlayerForgeWeatherOverrides porte ti=9 `i4
// managed-player-forge-weather-effect-overrides-component` — ecrivain `FUN_142ed5bc8`.
//
// GRAMMAIRE : `R(32)` ; `R(32)`. Deux champs de 32 bits dans cet ordre, 64 bits
// INCONDITIONNELS : aucune porte, aucun compteur, aucune boucle, aucun appel virtuel. Relevee
// sur les deux `ADD dword ptr [<reg> + 0x2c], 0x20` du desassemblage (`142ed5c70` et
// `142ed5d20`, plus leurs jumeaux de chemin rapide `142ed5bf9` et `142ed5cb6`) et sur les deux
// seules ecritures de sortie (`142ed5ca1 MOV dword ptr [RDI + 0x3c]`, `142ed5d5d MOV dword ptr
// [RDI + 0x40]`).
//
// AUCUNE DEPENDANCE DE CONFIGURATION : rien de la largeur ne depend d une table du jeu, d une
// carte ni d un build. C est le meme patron que `i5
// managed-player-active-mission-name-component` (`FUN_142ed5ab0`), deja porte.
func consumeManagedPlayerForgeWeatherOverrides(br *Lecteur) {
	br.ReadBits(32)
	br.ReadBits(32)
}

// Sous-types d un emplacement du sac texte (`k = R(3)` de `FUN_1407f0ebc`). Les valeurs 4 a 7
// partagent un seul corps (`FUN_142c70be0`, `R(32)`), d ou l absence de constante pour elles.
const (
	sacTexteDrapeau     = 0 // 0 bit de charge : le jeu range le type 0 et s arrete
	sacTexteParticipant = 1 // reference de participant (`FUN_1407f2058`)
	sacTexteNombre      = 2 // `FUN_142c70cd0` : quantifie sur 24 bits, ou brut sur 32
	sacTexteStringID    = 3 // `R(32)` string-id
)

// entreeSacTexte est UN emplacement du sac texte, tel que `FUN_1407f0ebc` le lit.
//
// LA VALEUR DU SOUS-TYPE 1 EST L INDEX BRUT, PAS CE QUE LE JEU EN FAIT : le binaire passe cet
// index a deux tables runtime (`FUN_14049746c`, `FUN_140e958c4`) qui ne recoivent PAS le flux et
// ne changent donc jamais le nombre de bits consommes. Publier l index brut est exactement ce que
// la note ti9ti35 §4 demande au port.
type entreeSacTexte struct {
	// SousType est le `k = R(3)` de tete.
	SousType uint64
	// Valeur est la charge lue, quand le sous-type en porte une.
	Valeur uint64
	// AValeur distingue « charge lue » de « charge absente » (sous-type 0, et sous-type 1
	// dont la porte est posee).
	AValeur bool
}

// sacTexte est ce que `FUN_14080b034(etat + 0x58, flux)` rend : un nom et ses emplacements types.
//
// `Present` est la porte `R(1)` de tete : a 0, le jeu ecrit le sentinelle `0xffffffff` et quatre
// octets de type nuls, et NE LIT RIEN DE PLUS.
type sacTexte struct {
	Present bool
	// Nom est le mot de 32 bits « text » (`FUN_14080dec4`). Le libelle est une chaine de
	// debogage passee a l appel, jamais un champ du flux.
	Nom uint64
	// Entrees porte les `n` emplacements lus. `n` se lit sur 3 bits (0..7) alors que la
	// structure du jeu n en a que QUATRE (`14080b066 LEA RCX,[RAX+0x20]`) : un ecrivain
	// conforme n emet pas `n > 4`, et le lecteur lit `n` corps quoi qu il arrive — c est ce
	// que fait le jeu, et le reproduire est la seule facon de rester synchrone.
	Entrees []entreeSacTexte
}

// consumeSacTexte porte `FUN_14080b034` — LE SAC TEXTE, COPIE UNIQUE DU DEPOT.
//
// GRAMMAIRE : `R(1)` texte ; si 1 : `R(32)` nom, `n = R(3)`, puis `n` fois un emplacement
// ([consumeEntreeSacTexte]). Preuves collees : `14080b056 CALL 0x1406cf008` (la porte, et le
// retour a 0 en `14080b05f..14080b077`), `14080b097 LEA RDX,[0x1436f4b68]` (les octets
// `74 65 78 74 00` = « text ») puis `14080b0a1 CALL 0x14080dec4` (le nom),
// `14080b0bd ADD dword ptr [RBX+0x2c],0x3` (le compteur), boucle `14080b0ef..14080b109`.
//
// POURQUOI UN HELPER DE PRODUCTION, ET PAS UNE TROISIEME COPIE (CLAUDE.md regle 6). Le meme sac
// sert l evenement `PlayerGameEventSmall` (type 82, `FUN_14080add8` -> `FUN_14080b034`), et deux
// instruments le decodaient chacun de leur cote — `playergameevent_0xe9_helpers_test.go` et
// `r7_charges_lot2_research_test.go`. Ils appellent desormais CETTE fonction. L un des deux
// marquait le sous-type 2 « quantifie a largeur runtime » et ne lisait AUCUN bit : sur pieces
// c est faux, `0x18` et `0x20` sont des litteraux (`142c70d02`, `142c70d27`).
func consumeSacTexte(br *Lecteur) sacTexte {
	var sac sacTexte
	if !br.ReadBit() { // FUN_1406cf008 : la porte « texte »
		return sac
	}
	sac.Present = true
	sac.Nom = br.ReadBits(32) // FUN_14080dec4(flux, "text", dst)
	n := br.ReadBits(3)
	sac.Entrees = make([]entreeSacTexte, 0, n)
	for i := uint64(0); i < n; i++ {
		sac.Entrees = append(sac.Entrees, consumeEntreeSacTexte(br))
	}
	return sac
}

// consumeEntreeSacTexte porte `FUN_1407f0ebc` — le corps d etiquette d UN emplacement.
//
// GRAMMAIRE : `k = R(3)` (`1407f0edd MOV EDI,0x3`), puis
//
//	k = 0     0 bit           `1407f0f09 JZ` vers le bloc froid : le jeu range le type 0.
//	k = 1     `R(1)` ; si 0 : `R(5)`  — `FUN_1407f2058` (`1407f0f1c CALL`). Les deux appels
//	          qui suivent (`FUN_14049746c`, `FUN_140e958c4`) ne recoivent pas le flux : 0 bit.
//	k = 2     `R(1)` ; si 1 : `R(24)` ; si 0 : `R(32)` — `FUN_142c70cd0` (`142c70cdc` la porte,
//	          `142c70d0a` le `FUN_1406d84b4` a `n = 0x18` pose en `142c70d02`, `142c70d33` le
//	          `FUN_1406d676c` a `R9D = 0x20` pose en `142c70d27`).
//	k = 3     `R(32)` — `FUN_14080dec4(flux, "string_id", ...)`.
//	k = 4..7  `R(32)` — `FUN_142c70be0` (`142c70c05 ADD dword ptr [R10+0x2c],0x20`).
//
// Largeur d un corps, `k` compris : 3 ; 4 ou 9 ; 28 ou 36 ; 35 ; 35.
func consumeEntreeSacTexte(br *Lecteur) entreeSacTexte {
	e := entreeSacTexte{SousType: br.ReadBits(3)}
	switch e.SousType {
	case sacTexteDrapeau:
		return e
	case sacTexteParticipant:
		if idx := readOpt5Signed(br); idx >= 0 { // FUN_1407f2058
			e.Valeur, e.AValeur = uint64(idx), true //nolint:gosec // idx >= 0 dans cette branche
		}
		return e
	case sacTexteNombre:
		if br.ReadBit() { // FUN_142c70cd0 : porte de quantification
			e.Valeur, e.AValeur = br.ReadBits(24), true // FUN_1406d84b4, n = 0x18
			return e
		}
		e.Valeur, e.AValeur = br.ReadBits(32), true // FUN_1406d676c, n = 0x20
		return e
	case sacTexteStringID:
		e.Valeur, e.AValeur = br.ReadBits(32), true // FUN_14080dec4 "string_id"
		return e
	default: // 4..7
		e.Valeur, e.AValeur = br.ReadBits(32), true // FUN_142c70be0
		return e
	}
}

// consumeManagedPlayerInputPrompt porte ti=9 `i9 managed-player-custom-input-prompt-widget` —
// ecrivain `FUN_141fcf160`.
//
// GRAMMAIRE : `R(1)` present (`141fcf17d INC dword ptr [RBX+0x2c]` sur le chemin rapide,
// `141fcf1b1 CALL FUN_1406d6c7c(flux, 1)` sur le lent) ; a 0, le lecteur du jeu rend `AL = 1`
// sans lire un bit de plus (`141fcf19c`, `141fcf22c`). A 1 : `R(2)` mode
// (`141fcf1dc ADD dword ptr [RBX+0x2c],0x2`) puis le sac texte
// (`141fcf20f LEA RCX,[RDI+0x58] ; 141fcf213 CALL 0x14080b034`).
//
// LA BOUCLE A ETIQUETTE EST DECIDABLE HORS LIGNE, ET C EST CE QUE CE LOT FERME. Le depot rendait
// `ported = false` des que le compteur `n` depassait 0 — une desynchronisation PROPRE, mais sur
// un composant dont la grammaire est entierement litterale. Aucune entree de profil, aucune
// dependance de configuration : les seules fonctions qui touchent de la RAM externe ne recoivent
// pas le flux.
func consumeManagedPlayerInputPrompt(br *Lecteur) {
	if !br.ReadBit() { // present
		return
	}
	br.ReadBits(2) // mode
	consumeSacTexte(br)
}
