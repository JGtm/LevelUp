package skill

// Révisions des formules de notation. Toute modification d'une formule ou d'un paramètre de
// calcul impose de changer la constante correspondante : la valeur composée par ScoringRevision
// est comparée à celle stockée par joueur, et une différence relance le calcul complet de
// l'historique au prochain démarrage. Le garde-rail TestScoringParametersMatchRevision (paquet sync) échoue
// quand un paramètre change sans que sa révision change.
const (
	// PerformanceScoreRev est la révision de la note de performance relative (poids par
	// chaîne, fenêtre glissante, seuils de matchs, durée de repli).
	PerformanceScoreRev = "perf-2026-11-08"
	// LUSRRev est la révision du LUSR (paramètres TrueSkill, poids du score composite,
	// pénalités d'abandon, couplage entre modes, décalage d'escouade, grille des paliers).
	LUSRRev = "lusr-2026-11-08"

	// SyncMetaScoringRevision est la clé de `sync_meta` qui porte, par joueur, la dernière
	// valeur de ScoringRevision dont le recalcul complet a réussi.
	SyncMetaScoringRevision = "scoring_revision"
)

// ScoringRevision compose les révisions en une seule valeur stockée par joueur.
func ScoringRevision() string { return "perf=" + PerformanceScoreRev + ";lusr=" + LUSRRev }
