package grammar

// observateur_chaine.go — LES COMPTEURS DE L INFERENCE DE CHAINE de l observateur
// ([Observation]), sortis de `observateur.go` par DEPLACEMENT PUR au lot M4b (2026-09-24) : ce
// fichier-la atteignait le seuil de 500 lignes, et le lot y ajoute la porte de la vue de controle
// ([Observation.VueControleHook]). Aucune ligne de logique n a change.

// compterResyncValide compte une reprise par resynchronisation validee (diagnostic).
func (o *Observation) compterResyncValide() {
	if o != nil {
		o.ResyncValides++
	}
}

// compterReparation compte un record sauve par l inference de largeur de composant, et range les
// largeurs de bouchon gagnantes dans l histogramme du composant.
func (o *Observation) compterReparation(nom string, largeurs []int) {
	if o == nil {
		return
	}
	o.ChaineReparees++
	if o.CompWidths == nil {
		o.CompWidths = map[string]map[int]int{}
	}
	if o.CompWidths[nom] == nil {
		o.CompWidths[nom] = map[int]int{}
	}
	for _, w := range largeurs {
		o.CompWidths[nom][w]++
	}
}

// compterIssueDeChaine compte une resolution : immediate (le record suivant confirme) ou
// PROFONDE (la marche recursive a traverse une suite de transitoires).
func (o *Observation) compterIssueDeChaine(immediate bool) {
	switch {
	case o == nil:
	case immediate:
		o.ChaineImmediat++
	default:
		o.ChaineProfond++
	}
}

// compterEchecDeChaine compte un echec : budget epuise, ou aucun alignement confirme.
func (o *Observation) compterEchecDeChaine(budgetEpuise bool) {
	switch {
	case o == nil:
	case budgetEpuise:
		o.ChaineBudget++
	default:
		o.ChaineAucun++
	}
}

// compterAmbiguiteDeChaine compte un alignement AMBIGU — plusieurs candidats survivent, et on ne
// choisit pas.
func (o *Observation) compterAmbiguiteDeChaine() {
	if o != nil {
		o.ChaineAmbigu++
	}
}
