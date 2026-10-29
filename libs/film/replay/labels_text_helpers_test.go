package replay

// Fixture minimale des libellés de test : deux codes de langue écrits à la main. La brique ne
// connaît aucune liste de langues (plan D2, item 2.9) ; ces deux codes sont des DONNÉES de test,
// ceux que portent les artefacts du parc.
const (
	codeTestEN = "en"
	codeTestFR = "fr"
)

// enFr fabrique le libellé d'un test dans les deux langues, comme le producteur
// (`replaylabels.TexteVersFilm`) les pose : chaque langue présente, vide comprise.
func enFr(en, fr string) Text {
	return Text{codeTestEN: en, codeTestFR: fr}
}

// enOf / frOf lisent la langue d'un libellé sans repli : une langue absente rend "".
func enOf(t Text) string { return t[codeTestEN] }
func frOf(t Text) string { return t[codeTestFR] }
