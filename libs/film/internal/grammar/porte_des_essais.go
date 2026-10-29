package grammar

// porte_des_essais.go — LA PORTE DES ESSAIS : UNE LECTURE SPECULATIVE NE PUBLIE RIEN.
//
// Les chemins speculatifs de la marche lisent des bits qu ils abandonneront peut-etre : le
// LOCALISATEUR de liste essaie des positions (`localisateur.go` : `marchLocateStrict`,
// `marchLocateFallback`, [LocaliserBoucleDeRecords]), l INFERENCE essaie des archetypes et des
// largeurs ([inferUnboundArchetype], [inferChainArchetype], [repairUnportedComponent],
// [validatedResync]), la RECOLTE rejoue des records. Chaque essai traverse l entite pour de vrai,
// donc les deserialiseurs publient ; une lecture speculative n est pas une lecture, et sans la
// porte les essais abandonnes deposeraient des valeurs a des positions que la traversee retenue
// ne lit jamais.
//
// POURQUOI UNE PORTE ET PAS UN FILTRE AVAL. Un filtre par calque devrait redecouvrir, apres coup,
// quel alignement la marche a retenu — c est-a-dire refaire le travail de la marche, et diverger
// d elle des qu elle change. Ici c est la marche elle-meme qui dit « cette lecture est un essai ».
//
// LES COMPTEURS RESTENT PARTAGES : c est le meme observateur, seuls les crochets sont eteints.
//
// CE QUI N EST PAS ETEINT, ET C EST ASSUME : `MobilityActionHook`, la porte historique d `i54`,
// sans consommateur de production ; et les crochets de TRAME (`VueControleHook`), qu aucun essai
// de record n atteint — l essai de fermeture du localisateur pose sa propre observation
// ([lectureDEssai]).

// crochetsDeCanal sont les crochets qu un canal de la marche des trames pose et qu un
// deserialiseur de composant publie pendant la lecture d un record : la porte des etats de
// mouvement et les onze crochets des lecteurs de composants bipedes (charges, impulsions,
// rangs, camouflage, grappin, arme portee, inventaire, equipement). Garde-rail :
// `porte_des_essais_guard_test.go` (chaque crochet eteint et restaure ; tout crochet qu un canal
// pose y est inscrit ; le localisateur declare la porte).
type crochetsDeCanal struct {
	etats         func(EtatMouvementComposant, uint32, []uint64)
	energie       func(uint32, [AbilityEnergyCharges]int)
	camouflage    func(CamoState)
	nonPredite    func(AbilityNonPredictedState)
	spartan       func(uint64, uint64, uint64, bool)
	rang          func(uint64, int, int)
	choixGrenades func(uint32, int)
	comptes       func(uint64, []uint64)
	armePortee    func(uint32, uint32)
	munitions     func(bool, uint32, bool, uint32)
	cartouches    func(uint32)
	equipement    func(UnitEquipmentRead)
}

// lesCrochetsDeCanal rend les crochets de canal que l observation porte.
func (o *Observation) lesCrochetsDeCanal() crochetsDeCanal {
	return crochetsDeCanal{
		etats: o.EtatMouvementHook, energie: o.AbilityEnergyHook, camouflage: o.CamoStateHook,
		nonPredite: o.AbilityNonPredictedHook, spartan: o.SpartanAbilityHook, rang: o.AbilitySetHook,
		choixGrenades: o.GrenadeSetHook, comptes: o.GrenadeCountsHook, armePortee: o.HeldWeaponHook,
		munitions: o.WeaponAmmoHook, cartouches: o.WeaponRoundsHook, equipement: o.UnitEquipmentHook,
	}
}

// poserLesCrochetsDeCanal installe `c` sur l observation.
func (o *Observation) poserLesCrochetsDeCanal(c crochetsDeCanal) {
	o.EtatMouvementHook, o.AbilityEnergyHook, o.CamoStateHook = c.etats, c.energie, c.camouflage
	o.AbilityNonPredictedHook, o.SpartanAbilityHook, o.AbilitySetHook = c.nonPredite, c.spartan, c.rang
	o.GrenadeSetHook, o.GrenadeCountsHook, o.HeldWeaponHook = c.choixGrenades, c.comptes, c.armePortee
	o.WeaponAmmoHook, o.WeaponRoundsHook, o.UnitEquipmentHook = c.munitions, c.cartouches, c.equipement
}

// neutraliserLesCrochetsDeCanal eteint les crochets de canal ([crochetsDeCanal]) et rend leur
// restauration. C EST LA PORTE UNIQUE DES ESSAIS : tous les chemins speculatifs de la marche
// passent par elle, directement (le localisateur) ou par les deux neutralisations des captures.
func (o *Observation) neutraliserLesCrochetsDeCanal() func() {
	if o == nil {
		return func() {}
	}
	c := o.lesCrochetsDeCanal()
	o.poserLesCrochetsDeCanal(crochetsDeCanal{})
	return func() { o.poserLesCrochetsDeCanal(c) }
}

// neutraliserCaptures eteint le crochet de position, celui de la reference d unite et les
// crochets de canal, et rend leur restauration : la porte des chemins d INFERENCE.
func (o *Observation) neutraliserCaptures() func() {
	if o == nil {
		return func() {}
	}
	pos, ref := o.PosCaptureHook, o.UnitRefHook
	o.PosCaptureHook, o.UnitRefHook = nil, nil
	restaureCanaux := o.neutraliserLesCrochetsDeCanal()
	return func() {
		o.PosCaptureHook, o.UnitRefHook = pos, ref
		restaureCanaux()
	}
}

// neutraliserCapturePosition eteint le crochet de position et les crochets de canal, et rend leur
// restauration. Elle laisse vivre la reference d unite, que les marches profondes comptent — c est
// la SEULE difference avec [Observation.neutraliserCaptures].
func (o *Observation) neutraliserCapturePosition() func() {
	if o == nil {
		return func() {}
	}
	pos := o.PosCaptureHook
	o.PosCaptureHook = nil
	restaureCanaux := o.neutraliserLesCrochetsDeCanal()
	return func() {
		o.PosCaptureHook = pos
		restaureCanaux()
	}
}
