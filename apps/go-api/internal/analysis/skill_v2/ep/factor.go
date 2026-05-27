package ep

// Factor est l'interface que tous les facteurs du graph implémentent.
//
// La sémantique d'un facteur en EP : il reçoit l'état courant des variables
// auxquelles il est connecté, calcule des nouveaux messages sortants (un par
// variable connectée), et les pousse aux variables. La variable absorbe le
// message via UpdateMessage qui renvoie la "delta" — somme des deltas → critère
// de convergence du runner.
//
// La méthode UpdateMessages doit être idempotente sur un graph convergé :
// appeler deux fois ne change plus rien (delta < eps). Un graph divergent
// peut être détecté en surveillant la delta cumulative sur les N dernières
// passes.
type Factor interface {
	// UpdateMessages calcule et pousse les nouveaux messages aux variables
	// connectées. Retourne la delta totale (= somme des AbsoluteDifference
	// sur les variables touchées). Utilisé par le runner pour détecter la
	// convergence.
	UpdateMessages() float64

	// Name retourne un identifiant lisible pour debug/logs.
	Name() string
}

// Runner orchestre les message-passes EP jusqu'à convergence.
//
// Stratégie : round-robin simple — à chaque pass, on appelle UpdateMessages
// sur tous les facteurs dans l'ordre fourni. Convergence détectée quand le
// max-delta d'une pass < tolérance. Échec si maxIters atteint sans converger.
//
// Cette stratégie est suffisante pour les factor graphs TS / TS2 classiques
// (acycliques ou faiblement cycliques). Pour un graph très cyclique, on
// pourrait ajouter du damping (averaging des nouveaux messages avec les
// anciens) — pas nécessaire pour l'instant.
type Runner struct {
	Factors   []Factor
	Tolerance float64 // delta max accepté pour considérer convergé. Typiquement 1e-4.
	MaxIters  int     // borne sur le nombre de passes (anti-divergence). Typiquement 50.
}

// NewRunner construit un runner avec des défauts raisonnables.
func NewRunner(factors []Factor) *Runner {
	return &Runner{
		Factors:   factors,
		Tolerance: 1e-4,
		MaxIters:  50,
	}
}

// Run exécute des passes jusqu'à convergence ou MaxIters. Retourne (iterations
// effectuées, max-delta de la dernière pass, true si convergé).
func (r *Runner) Run() (iterations int, lastMaxDelta float64, converged bool) {
	for i := 0; i < r.MaxIters; i++ {
		maxDelta := 0.0
		for _, f := range r.Factors {
			d := f.UpdateMessages()
			if d > maxDelta {
				maxDelta = d
			}
		}
		iterations = i + 1
		lastMaxDelta = maxDelta
		if maxDelta < r.Tolerance {
			converged = true
			return
		}
	}
	return
}
