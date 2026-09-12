package filmdec

// components_game_engine.go — LES COMPOSANTS DE L'ENTITE MOTEUR DE JEU (ti=0), QUI PUBLIENT.
//
// CE QUE CE FICHIER CHANGE, ET CE QU'IL NE CHANGE PAS. Les cinq deserialiseurs ci-dessous
// consommaient EXACTEMENT les memes bits, en ligne dans le `switch` de `consumeByName`, et
// jetaient leurs valeurs. Ils sont ici, nommes, et ils PUBLIENT par un hook de paquet. Aucune
// largeur ne bouge, aucun ordre de lecture ne bouge : le garde-rail
// `TestHooksConsumeSameBitsWithoutHook` echoue si un seul bit se deplace.
//
// POURQUOI CETTE PLOMBERIE EST FAITE UNE FOIS, AVANT LES MESURES (D15 du plan d'exploitation) :
// les lots B (horloge et etats de partie), C (zones), D (equipement), F (sondes) et P (entite
// joueur) ont tous besoin de valeurs qui sortent de `traverse.go`. Chacun y touchant a son
// tour, ils se marcheraient dessus sur le fichier le plus dangereux du paquet. Apres ce lot,
// ils n'AJOUTENT que des fichiers.
//
// LA REGLE QUI GOUVERNE, reprise d'`equipment_state.go` : c'est le DESERIALISEUR qui publie,
// jamais un second lecteur pose a cote de lui. Deux lecteurs du meme champ divergent le jour
// ou l'un des deux est corrige.
//
// CE QUI RESTE SUR LA COUCHE DE CAPTURE, ET NE DOIT PAS ETRE DUPLIQUE ICI : ti=0 i5
// (`game-engine-round-timer-component`) et ti=5 i1 (`player-respawn-timer-component`) sont
// rendus par `consumeByNameCapturing` (`capture.go:20-25`) sous forme typee. Les publier une
// seconde fois par un hook creerait la troisieme copie que la regle du depot interdit.

import "fmt"

// Etiquettes de registre des composants de ti=0 publies ici. Elles servent DEUX fois chacune
// (le routage de `consumeByName` et `String()` ci-dessous) : les nommer evite qu'une
// correction de l'un oublie l'autre.
const (
	compGameEngineCurrentState        = "game-engine-current-state-component"
	compGameEngineCurrentRound        = "game-engine-current-round-component"
	compGameEngineSuddenDeath         = "game-engine-sudden-death-time-left-component"
	compGameEngineGracePeriod         = "game-engine-grace-period-time-left-component"
	compGameEngineRoundConditionFlags = "game-engine-round-condition-flags-component"
)

// GameEngineField designe l'un des cinq champs publies.
//
// POURQUOI UNE ENUMERATION ET NON L'INDEX DU REGISTRE. Le plan ecrit `comp int` ; le MODELE
// qu'il designe (`equipment_state.go`) resout l'index par NOM et ne le cable jamais, parce
// qu'un index de composant est un NUMERO DE BUILD (`equipment_state.go:157`) — et l'empreinte
// de registre livree au meme lot 0 le prouve : `06dfe6d9` ne porte pas le meme decoupage que
// `000d5950`. L'enumeration est donc notre identifiant STABLE, et `String()` rend l'etiquette
// de registre, seule facon honnete de nommer le champ.
type GameEngineField int

// Les cinq champs, dans l'ordre des index du registre de reference, et leur compte.
const (
	GameEngineState           GameEngineField = iota // i2 : R(3)
	GameEngineRound                                  // i4 : R(1) porte [si 0 -> R(5)]
	GameEngineSuddenDeath                            // i6 : R(16)+R(16)+R(5)
	GameEngineGracePeriod                            // i7 : R(16)+R(16)+R(5)
	GameEngineRoundConditions                        // i8 : R(10)
	GameEngineFieldCount      = 5
)

// String rend l'etiquette de registre du champ.
func (f GameEngineField) String() string {
	switch f {
	case GameEngineState:
		return compGameEngineCurrentState
	case GameEngineRound:
		return compGameEngineCurrentRound
	case GameEngineSuddenDeath:
		return compGameEngineSuddenDeath
	case GameEngineGracePeriod:
		return compGameEngineGracePeriod
	case GameEngineRoundConditions:
		return compGameEngineRoundConditionFlags
	}
	return fmt.Sprintf("champ inconnu (%d)", int(f))
}

// gameEngineHook, si non nil, recoit CHAQUE lecture d'un des cinq composants.
//
// CONTRAT DE `values` ET DE `present`, commun aux quatre hooks de ce lot :
//
//	values   les champs lus par le deser, DANS L'ORDRE DU FLUX, bits de porte compris.
//	         Jamais de dequantification, jamais de mise a l'echelle : c'est le lot qui
//	         mesure qui decide du sens, pas la plomberie.
//	present  faux quand la porte de TETE du composant s'est fermee et qu'aucun champ n'a
//	         suivi. Une porte fermee n'est PAS une valeur nulle, et les confondre
//	         fabriquerait des transitions qui n'existent pas.
//
// PRE-REQUIS : l'appelant detient `LockProcessDecode` — le hook est un global de paquet.
var gameEngineHook func(f GameEngineField, values []uint64, present bool)

// SetGameEngineHook installe (ou retire, avec nil) la sonde des composants de ti=0.
func SetGameEngineHook(h func(f GameEngineField, values []uint64, present bool)) {
	gameEngineHook = h
}

func publishGameEngine(f GameEngineField, present bool, values ...uint64) {
	if gameEngineHook != nil {
		gameEngineHook(f, values, present)
	}
}

// consumeGameEngineCurrentState porte ti=0 i2 (FUN_14116d1d0) : R(3), sans porte.
func consumeGameEngineCurrentState(br *BitReader) {
	publishGameEngine(GameEngineState, true, br.ReadBits(3))
}

// consumeGameEngineCurrentRound porte ti=0 i4 (FUN_14116fc70) : R(1) porte de polarite
// INVERSEE — la manche n'est transmise que si le bit vaut 0.
func consumeGameEngineCurrentRound(br *BitReader) {
	if !br.ReadBit() {
		publishGameEngine(GameEngineRound, true, br.ReadBits(5))
		return
	}
	publishGameEngine(GameEngineRound, false)
}

// consumeGameEngineSuddenDeath porte ti=0 i6 (FUN_14116d3a4) : R(16)+R(16)+R(5), sans porte.
// C'est la source candidate de la prolongation (lot B, D5) — mesuree, jamais devinee.
// Les trois lectures sont des VARIABLES et non des arguments : l'ordre d'evaluation des
// arguments est bien garanti a gauche-droite par le langage, mais l'ordre des bits est ce que
// ce paquet a de plus fragile et il doit se LIRE, pas se deduire d'une regle du spec.
func consumeGameEngineSuddenDeath(br *BitReader) {
	a := br.ReadBits(16)
	b := br.ReadBits(16)
	c := br.ReadBits(5)
	publishGameEngine(GameEngineSuddenDeath, true, a, b, c)
}

// consumeGameEngineGracePeriod porte ti=0 i7 (FUN_141165d24) : R(16)+R(16)+R(5), meme forme
// que i6 et que le round-timer i5.
func consumeGameEngineGracePeriod(br *BitReader) {
	a := br.ReadBits(16)
	b := br.ReadBits(16)
	c := br.ReadBits(5)
	publishGameEngine(GameEngineGracePeriod, true, a, b, c)
}

// consumeGameEngineRoundConditionFlags porte ti=0 i8 (FUN_141132dc0) : R(10), sans porte.
func consumeGameEngineRoundConditionFlags(br *BitReader) {
	publishGameEngine(GameEngineRoundConditions, true, br.ReadBits(10))
}
