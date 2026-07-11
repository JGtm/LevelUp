package analysis

// kill_attribution.go — Structure résultat d'attribution d'un kill à une arme.
//
// Port de src/analysis/_kill_attribution.py.

// Valeurs possibles pour KillAttribution.AttributionPath.
const (
	AttributionPathFireEvent    = "fire_event"
	AttributionPathFormulaA     = "formula_a"
	AttributionPathDamageSource = "damage_source" // corrélation same-clock (source de dégât)
	AttributionPathNone         = "none"
)

// KillAttribution représente le résultat d'attribution d'un kill à une arme.
type KillAttribution struct {
	MatchID         string
	XUID            string
	TimeMS          int
	WeaponID        *uint64 // Film attribution (nil si non résolu)
	ReconciledAs    *uint64 // API override (nil si pas de réconciliation)
	DeltaMS         *int    // Écart fire event → kill (nil si pas de fire event)
	Confidence      string  // "high", "medium", "low", "none"
	AttributionPath string  // "fire_event", "formula_a", "none"
	SwapDetected    bool
	DelayedDamage   bool
	PlayerIndex     *int
	SourceChunkIdx  *int
}

// EffectiveWeaponID retourne COALESCE(reconciled_as, weapon_id).
func (ka *KillAttribution) EffectiveWeaponID() *uint64 {
	if ka.ReconciledAs != nil {
		return ka.ReconciledAs
	}
	return ka.WeaponID
}
