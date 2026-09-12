package filmdec

// emp_timer.go — LE MINUTEUR EMP DU BIPÈDE (i51), publié le 2026-08-24 pour l'item 0.5 du
// plan .ai/V7.5/replay2d/PLAN_CAPACITES_ACTIVES.md.
//
// POURQUOI CE COMPOSANT-LÀ. L'item 0.4 cherche un canal DÉDIÉ à l'usage du répulseur et du
// propulseur dans `i56` (l'énergie de capacité) ; `i51` est le candidat SECONDAIRE, désigné
// par le plan parce qu'il n'avait jamais été interrogé. Sa sémantique documentée
// (`testdata/ecs_table.tsv`, ti=35 i51) est « combien de temps le joueur reste neutralisé »
// — un effet SUBI, pas une action. La mesure dira si ce champ bouge sur les vies porteuses
// d'une capacité cible ; elle ne suppose pas qu'il le fasse.
//
// CE QUI CHANGE DANS LE FLUX : RIEN. Le déserialiseur lisait déjà ses 8 bits pour rester
// aligné et les jetait — même défaut qu'i48 (corrigé le 2026-08-14), i56 (le 2026-08-15) et
// les quatre champs de ti=37. La règle qui gouverne est celle d'`ability_state_hooks.go` :
// c'est le DÉSERIALISEUR qui publie, jamais un second lecteur posé à côté de lui. Aucune
// largeur ne change.

// EmpTimerQuantMax est la borne haute du quantum R(8) : le champ est un minuteur quantifié
// sur 0..10 s (déser FUN_142f02830). La conversion en secondes n'est PAS faite ici — les
// bornes de déquantification ne sont pas établies pour ce composant, et publier un quantum
// brut vaut mieux que publier une seconde inventée.
const EmpTimerQuantMax = 255

// empTimerHook, si non nil, reçoit le quantum R(8) de CHAQUE lecture d'i51 par le déser de
// production. Global de paquet : un seul décodage filmdec par process (cf. decode_gate.go).
var empTimerHook func(quant uint32)

// SetEmpTimerHook installe (ou retire, avec nil) la sonde d'i51.
func SetEmpTimerHook(h func(quant uint32)) { empTimerHook = h }

// publishEmpTimer est appelé par le déserialiseur d'i51 (traverse.go) juste après la lecture
// de ses 8 bits. Le parcours de bits est inchangé.
func publishEmpTimer(quant uint64) {
	if empTimerHook != nil {
		empTimerHook(uint32(quant))
	}
}
