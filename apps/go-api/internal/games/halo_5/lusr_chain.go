package halo_5

// lusr_chain.go — classification LUSR (chaîne TrueSkill) pour Halo 5.
//
// Halo 5 n'a PAS de pair_name (cf. mapping.go : PairMode nil) : la partition LUSR
// Infinite (pair_name → arena_slayer/arena_objectif/btb/chaos) ne s'applique pas.
// Premier jet (h5 expérimental) : une chaîne UNIQUE pour tous les matchs h5
// éligibles. Le filtrage Ranked→CSR / Firefight→PvE est fait en amont par le SQL
// du loader (is_ranked / is_firefight) ; Warzone/FFA (non 2-équipes) sont écartés
// par buildTwoTeamRosters. La v2 LUSR calcule sur des données basiques (k/d/a,
// outcome, time_played) que h5 fournit — pas besoin de MMR.
//
// Une partition PAR MODE h5 (dérivée de playlist_name) est un raffinement ultérieur :
// le seam classifier ne reçoit aujourd'hui que pair_name. Posé via
// sync.SetLUSRChainClassifierForTitle(halo5.TitleSlug, halo5.ClassifyLUSRChain).

// LUSRChainArena est l'unique chaîne LUSR Halo 5 (cf. ClassifyLUSRChain).
const LUSRChainArena = "h5_arena"

// ClassifyLUSRChain retourne la chaîne LUSR d'un match Halo 5. Le pair_name est
// ignoré (h5 n'en a pas) : tous les matchs éligibles tombent dans LUSRChainArena.
func ClassifyLUSRChain(_ string) string {
	return LUSRChainArena
}

// IsObjectiveSubMode répond à la question « ce match classé est-il de la famille
// objectif ? » pour Halo 5 : TOUJOURS false. Halo 5 n'a pas de pair_name (cf.
// mapping.go : PairMode nil), donc aucun sous-mode à lire ; ses matchs classés
// tombent tous dans la chaîne de performance ranked_slayer. Sans ce classifier
// dédié, le seam appliquerait la liste Infinite à un pair_name h5 vide — même
// résultat aujourd'hui, mais par accident plutôt que par décision.
//
// Une partition par mode h5 (dérivée de playlist_name, comme pour la chaîne LUSR)
// est le raffinement ultérieur : le seam ne reçoit aujourd'hui que pair_name.
// Posé via sync.SetObjectiveFamilyClassifierForTitle(halo5.TitleSlug, halo5.IsObjectiveSubMode).
func IsObjectiveSubMode(_ string) bool {
	return false
}
