package migration

// title_steps.go — pont entre le registre global (legacy, peuplé par init()) et
// les migrations APPARTENANT à un titre (Phase 1.5.1 voie B, ADR 0025).
//
// Voie B : les steps d'un titre vivent dans internal/games/{slug}/migrations/ et
// sont fournis au runner via un provider, au lieu d'être enregistrés par init()
// dans le package migration. Pendant la transition, RunForDB combine les deux
// sources ; l'ordre d'exécution reste imposé par canonicalOrder (order.go).
//
// Provider non posé (nil) ⇒ comportement strictement identique au legacy
// (ForTarget global seul) — la bascule est donc additive et no-op tant
// qu'aucun step n'a été déplacé.

// titleStepsProvider fournit les migrations title-owned pour une target.
// Posé une fois au boot via SetTitleStepsProvider (par le câblage qui importe
// les packages de titres). nil = aucun step title-owned (legacy pur).
var titleStepsProvider func(TargetDB) []Migration

// SetTitleStepsProvider enregistre la source des migrations title-owned.
// À appeler au boot, avant tout RunForDB. Idempotent (dernier gagne).
func SetTitleStepsProvider(p func(TargetDB) []Migration) { titleStepsProvider = p }

// stepsForTarget assemble les migrations à exécuter pour une target : registre
// global (legacy) + steps title-owned, dédupliqués par Name (le title-owned
// remplace son homonyme legacy le temps de la migration fichier par fichier).
func stepsForTarget(target TargetDB) []Migration {
	base := ForTarget(target)
	if titleStepsProvider == nil {
		return base
	}
	return combineSteps(base, titleStepsProvider(target))
}

// combineSteps retourne base + extra, en retirant de base tout step dont le Name
// est (re)défini dans extra. L'ordre final est ré-imposé par sortByCanonicalOrder
// dans RunSteps, donc l'ordre de concaténation ici n'a pas d'importance.
func combineSteps(base, extra []Migration) []Migration {
	if len(extra) == 0 {
		return base
	}
	overridden := make(map[string]bool, len(extra))
	for _, m := range extra {
		overridden[m.Name] = true
	}
	out := make([]Migration, 0, len(base)+len(extra))
	for _, m := range base {
		if !overridden[m.Name] {
			out = append(out, m)
		}
	}
	return append(out, extra...)
}
