package filmdec

// inventory_delta_stats.go — LES DENOMINATEURS du balayage d'inventaire delta. Extrait
// d'inventory_delta.go (seuil de taille du depot, CLAUDE.md n°5), avec le meme motif que
// `KeyframeInventoryStats`, qui vit a cote d'`InventoryCoverage` pour la meme raison.
//
// POURQUOI CE TYPE PESE AUTANT. Un balayage bit a bit ne se juge pas sur sa liste de lectures :
// il se juge sur ce qu'il a RENCONTRE et sur ce qu'il a REFUSE. Chaque compteur ci-dessous
// existe parce qu'une question s'est posee pendant la mesure multi-films du 2026-08-25 et que
// seul ce compteur pouvait y repondre.

// InventoryDeltaStats compte ce que la marche a rencontré. Sans ces dénominateurs, une liste
// de lectures ne se juge pas — et c'est `Implausible` qui porte le test réfutable.
type InventoryDeltaStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI22 / WithI47 : records dont le masque ANNONCE le composant.
	WithI22, WithI47 int
	// I22Read / I47Read : lectures abouties (la marche a atteint le composant et le déser a
	// publié). I22Unread / I47Unread : la marche s'est arrêtée avant.
	I22Read, I22Unread int
	I47Read, I47Unread int
	// Implausible est le nombre de lectures d'i22 ABOUTIES mais rejetées : compteur != 4, ou
	// une valeur au-delà de invDeltaMaxPerType. C'est LE chiffre à surveiller.
	Implausible int
	// NoSelection est le nombre de lectures d'i47 qui ne désignent aucun type (sélection 0) —
	// une mesure, pas un échec.
	NoSelection int
	// MaskEmpty est le nombre de lectures d'i47 dont le MASQUE est vide : le porteur n'a plus
	// aucune grenade.
	//
	// CE CAS N'EST PAS UNE ANOMALIE, et l'avoir cru l'a été. i47 est le jeu DÉSIRÉ : sur
	// certains films le champ de sélection garde un rang résiduel alors que le masque est
	// retombé à zéro (03af54c3 : 75 lectures, masque 000000 et sélection non nulle ; 000d5950
	// : les 20 masques vides portent une sélection nulle). Compter ces lectures comme des
	// « sélections hors masque » faisait chuter la conformité du corpus à 93,4 % pour une
	// raison qui n'était pas une erreur de lecture. Elles sont publiées SANS sélection : ne
	// portant aucune grenade, le porteur n'en sélectionne aucune.
	MaskEmpty int
	// SelOutsideMask est le nombre de sélections non nulles qui n'appartiennent PAS à un
	// masque NON VIDE du même record — le vrai test réfutable du handoff. Ces lectures sont
	// publiées SANS sélection (Sel = InventoryDeltaNoSel).
	SelOutsideMask int
	// WithAmmo / WithRounds : records dont le masque annonce un `weapon-state-ammo` ou un
	// `weapon-state-rounds-inventory`, toutes occurrences confondues. AmmoRead / RoundsRead :
	// occurrences effectivement consommées par la marche. MagRead : celles dont la porte de
	// chargeur était ouverte (le film écrit une valeur).
	WithAmmo, WithRounds int
	AmmoRead, RoundsRead int
	MagRead              int
	// MagOutOfEnvelope / ResOutOfEnvelope : lectures rejetées parce qu'au-delà de l'enveloppe
	// mesurée (cf. invDeltaMagEnvelope). Ce sont ELLES qu'on surveille, pas une valeur isolée.
	MagOutOfEnvelope, ResOutOfEnvelope int
	// MagCorroborated / MagOutOfEnvelopeCorroborated : la même mesure, restreinte aux records
	// où i22 a rendu une lecture PLAUSIBLE. Sur ces records le curseur a passé un test
	// indépendant avant d'atteindre le chargeur ; comparer les deux taux dit si un dépassement
	// est une arme inhabituelle ou une marche qui a dérivé.
	MagCorroborated, MagOutOfEnvelopeCorroborated int
	// AmmoRefused dit que le canal MUNITIONS de ce film a ete refuse EN BLOC : sa distribution
	// de chargeurs est contaminee au-dela du seuil, donc aucune de ses valeurs n'est digne de
	// confiance, pas meme celles qui tombent sous l'enveloppe (cf. refuseAmmoIfContaminated).
	// Les GRENADES du meme film restent publiees : elles ont leurs propres tests.
	AmmoRefused bool
	// AccordChecked / Accord : LE CONTRÔLE CROISÉ LE PLUS FORT du balayage. Sur les records
	// qui portent i22 (lecture plausible) ET i47, le masque d'i47 doit être EXACTEMENT le
	// bitmap des compteurs non nuls d'i22 — c'est la règle que le canal des images-clés impose
	// déjà à sa propre lecture d'i47 (replay/inventory_grenade_selection.go). Les deux
	// composants sont désérialisés à des positions DIFFÉRENTES du même record, par des désers
	// différents : leur accord ne peut pas venir de la construction. Mesuré 100,0 % sur le
	// corpus (cf. inventory_delta_corpus_test.go).
	AccordChecked, Accord int
	// Emitted est le nombre de lectures publiées (au moins une des deux grandeurs lue).
	Emitted int
}
