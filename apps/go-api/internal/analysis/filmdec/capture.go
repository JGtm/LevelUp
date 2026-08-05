package filmdec

// Couche de CAPTURE : elle rend les valeurs des composants que le décodeur ne faisait
// jusqu'ici que traverser.
//
// POURQUOI UNE COUCHE SÉPARÉE plutôt qu'un 4e retour sur consumeByName : ce dispatch a plus
// de deux cents `return variant, nil, true`. Y ajouter une valeur de retour serait une
// édition mécanique massive, à fort risque, pour quatre composants. La couche ci-dessous
// intercepte AVANT le dispatch les seuls noms qui portent une valeur exploitable, et
// délègue tout le reste inchangé.
//
// L'ÉTIQUETTE DE COMPOSANT APPARAÎT DONC DANS DEUX `switch`. Ce n'est pas une duplication
// de GRAMMAIRE : les deux branches appellent la MÊME fonction `decodeXxx` (vitality.go), et
// TestCaptureConsumesSameBitsAsDispatch (capture_test.go) échoue si l'une des deux venait à
// consommer un nombre de bits différent de l'autre. C'est le garde-rail exigé par la règle
// « ≤ 2 copies, et une factorisation sans garde-rail re-diverge ».

// captureNames liste les composants dont consumeByNameCapturing rend la valeur. Sert au
// garde-rail : le test balaie cette liste, il ne peut donc pas oublier un nom ajouté ici.
var captureNames = []string{
	compObjectBodyVitality,
	"object-shield-vitality-component",
	"player-respawn-timer-component",
	"game-engine-round-timer-component",
}

// consumeByNameCapturing consomme un composant comme consumeByName, et rend en plus sa
// valeur décodée (payload) pour les composants de captureNames. payload est nil partout
// ailleurs — le champ CompResult.Payload est donc absent par défaut, sans coût.
func consumeByNameCapturing(br *BitReader, name string, typeIndex, level uint32) (
	variant uint32, dead *DeadState, payload any, ported bool) {
	switch name {
	case compObjectBodyVitality: // i4
		return noVariant, nil, decodeObjectBodyVitality(br), true
	case "object-shield-vitality-component": // i5
		return noVariant, nil, decodeObjectShieldVitality(br), true
	case "player-respawn-timer-component": // ti=5 i1
		return noVariant, nil, decodePlayerRespawnTimer(br), true
	case "game-engine-round-timer-component": // ti=0 i5
		return noVariant, nil, decodeGameEngineRoundTimer(br), true
	}
	variant, dead, ported = consumeByName(br, name, typeIndex, level)
	return variant, dead, nil, ported
}

// BodyOf rend la vitalité i4 capturée par un composant, si c'en est une.
func (c CompResult) BodyOf() (BodyVitality, bool) {
	v, ok := c.Payload.(BodyVitality)
	return v, ok
}

// ShieldOf rend la vitalité i5 capturée par un composant, si c'en est une.
func (c CompResult) ShieldOf() (ShieldVitality, bool) {
	v, ok := c.Payload.(ShieldVitality)
	return v, ok
}

// RespawnOf rend le compteur de réapparition capturé, si c'en est un.
func (c CompResult) RespawnOf() (RespawnTimer, bool) {
	v, ok := c.Payload.(RespawnTimer)
	return v, ok
}

// RoundTimerOf rend l'horloge de manche capturée, si c'en est une.
func (c CompResult) RoundTimerOf() (RoundTimer, bool) {
	v, ok := c.Payload.(RoundTimer)
	return v, ok
}
