package grammar

// keyframe_etat_complet_bipede.go — L ETAT COMPLET DU BIPEDE AUX IMAGES-CLES, LU PAR LA GRAMMAIRE
// (plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, D1.1.2 ; ADR 0037 IR-6, option A de
// l utilisateur du 2026-10-04 : les fenetres de bits passent DERRIERE la lecture).
//
// Un canal de la phase des images-cles ([canalDeLEtatCompletBipede]) declare, par NOM, les
// composants du bipede (`ti=35`) dont il lit la valeur : l etat de mort (i11), l echelle (i12), les
// vitalites maximales (i13), les compteurs de grenades (i22), munitions, reserves et surchauffes des
// quatre emplacements (i30..i41), le jeu d armes (i42), l identite des quatre armes (i43..i46), le
// jeu de grenades (i47) et le jeu de capacites (i48). La phase traverse alors l etat complet de
// chaque record bipede ; le canal relit chaque occurrence a son etendue
// ([relireLOccurrence]) sous les crochets d une observation a lui. Les index viennent des noms du
// registre du film, jamais de constantes : un index de composant est un numero de build.
//
// CE QUE LA LECTURE REND N EST PUBLIE QUE SUR UN RECORD ADMIS ([admissionDeLEtatComplet], U-1
// amendee le 2026-10-09). Les autres records passent a la fenetre, sous un repli nomme et compte
// (D1.2).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// genreDeRole dit ce qu une occurrence du bipede porte pour le canal.
type genreDeRole uint8

// Les roles des composants que le canal lit.
const (
	roleAucun genreDeRole = iota
	roleMort
	roleEchelle
	roleVitalites
	roleGrenades
	roleChargeur
	roleReserve
	roleSurchauffe
	roleJeuDArmes
	roleArme
	roleJeuDeGrenades
	roleCapacite
)

// roleDOccurrence est le role d un index de composant du bipede et, pour les composants repetes
// par emplacement, son rang (0..3) dans l ordre des composants de l archetype.
type roleDOccurrence struct {
	genre genreDeRole
	rang  int
}

// nomDeRole : un nom de registre et le role de ses occurrences.
type nomDeRole struct {
	nom   string
	genre genreDeRole
}

// nomsDesRoles rend le nom de registre de chaque role ; un role a deux orthographes (i47, i48) les
// declare toutes les deux. Une fonction et non une variable de paquet : `archlint` gele l etat
// global de la grammaire.
func nomsDesRoles() []nomDeRole {
	return []nomDeRole{
		{deadStateComponentName, roleMort},
		{compObjectScale, roleEchelle},
		{compObjectMaximumVitalities, roleVitalites},
		{invDeltaGrenadeCountsName, roleGrenades},
		{invDeltaAmmoName, roleChargeur},
		{invDeltaRoundsName, roleReserve},
		{compWeaponStateOverheated, roleSurchauffe},
		{compBipedDesiredWeaponSet, roleJeuDArmes},
		{compWeaponStateTypeInfo, roleArme},
		{invDeltaGrenadeSetName, roleJeuDeGrenades},
		{invDeltaGrenadeSetAltName, roleJeuDeGrenades},
		{compBipedAbilitySet, roleCapacite},
		{compBipedAbilitySetAlt, roleCapacite},
	}
}

// Les noms de registre que seul ce canal cite.
const (
	compObjectScale           = "object-scale-component"
	compWeaponStateOverheated = "weapon-state-overheated"
	compBipedDesiredWeaponSet = "biped-desired-weapon-set"
)

// rolesDuBipede rend le role de chaque index de composant de l archetype bipede.
func rolesDuBipede(arch Archetype) map[int]roleDOccurrence {
	out := map[int]roleDOccurrence{}
	for _, n := range nomsDesRoles() {
		for rang, id := range arch.indicesOf(n.nom) {
			out[id] = roleDOccurrence{genre: n.genre, rang: rang}
		}
	}
	return out
}

// interetsDeLEtatComplet rend les interets du canal : chaque nom de [nomsDesRoles], dans la phase
// des images-cles, sur l archetype bipede.
func interetsDeLEtatComplet() []Interet {
	roles := nomsDesRoles()
	out := make([]Interet, 0, len(roles))
	for _, n := range roles {
		out = append(out, Interet{Phase: PhaseImagesCles, TI: keyframeBipedTI, Composant: n.nom})
	}
	return out
}

// emplacementsDArme est le nombre d emplacements que le bipede decrit (quatre entrees de la carte
// memoire, `0x7f0 + s*0x90`), la dimension de [lectureDEtatComplet].
const emplacementsDArme = 4

// lectureDEtatComplet est ce que la grammaire lit de l etat complet d UN record bipede.
type lectureDEtatComplet struct {
	// grenadesLues : i22 relu ; compte, compteurs : son R(3) et ses compteurs R(8).
	grenadesLues bool
	compte       uint64
	compteurs    []uint64
	// chargeurLu, reserveLue, surchauffeLue : les composants de l emplacement k relus.
	chargeurLu, reserveLue, surchauffeLue [emplacementsDArme]bool
	aChargeur, aJauge                     [emplacementsDArme]bool
	chargeur, jauge, reserve, surchauffe  [emplacementsDArme]uint32
	drapeaux                              [emplacementsDArme]uint32
	// jeuLu, jeu : i42 relu et ses trois champs.
	jeuLu bool
	jeu   JeuDArmes
	// armeLue, armeHaute : l identite de l arme de l emplacement k relue, et sa moitie haute
	// ([noVariant] pour un emplacement vide).
	armeLue   [emplacementsDArme]bool
	armeHaute [emplacementsDArme]uint32
	// jeuDeGrenadesLu, masque, selection : i47 relu, son masque R(6) et sa selection R(3) en base 1.
	jeuDeGrenadesLu bool
	masque          uint32
	selection       int
	// capaciteLue, rang : i48 relu et le rang de palette ([AbilitySetNoRank] sans identite).
	capaciteLue bool
	rang        int
	// mortParDefaut, echelleAUn, vitalites : la configuration que la marque de portage recouvre.
	mortParDefaut bool
	echelleAUn    bool
	vitalites     int
	// debordements : occurrences dont la relecture ne tient pas l etendue de la marche.
	debordements int
}

// lireLEtatComplet relit les occurrences de role du record `r` du paquet `p`.
func lireLEtatComplet(p *lecture.Paquet, r *lecture.Record, arch Archetype, roles map[int]roleDOccurrence,
	ctx ContexteDeLecture) lectureDEtatComplet {
	l := lectureDEtatComplet{vitalites: -1, selection: -1, rang: AbilitySetNoRank}
	for _, co := range p.Comps[r.Comps[0]:r.Comps[1]] {
		role, ok := roles[int(co.Index)]
		if !ok || co.Etat != lecture.EtatInterprete {
			continue
		}
		if l.lireLaConfiguration(p.Payload, role, co) {
			continue
		}
		if !relireLOccurrence(p.Payload, ctx, arch, keyframeBipedTI, co, l.observation(role)) {
			l.debordements++
		}
	}
	return l
}

// lireLaConfiguration lit les trois occurrences de la configuration de la marque de portage
// (i11, i12, i13) dans leurs bits, et rend vrai quand l occurrence en etait une.
func (l *lectureDEtatComplet) lireLaConfiguration(pay []byte, role roleDOccurrence, co lecture.Composant) bool {
	switch role.genre {
	case roleMort:
		l.mortParDefaut = co.Bits == largeurDeLEtatDeMortParDefaut &&
			bitsDeLOccurrence(pay, co, largeurDeLEtatDeMortParDefaut) == etatDeMortParDefaut
	case roleEchelle:
		l.echelleAUn = co.Bits == 1 && bitsDeLOccurrence(pay, co, 1) == 1
	case roleVitalites:
		if co.Bits >= largeurDesDrapeauxDeVitalites {
			l.vitalites = int(bitsDeLOccurrence(pay, co, largeurDesDrapeauxDeVitalites)) //nolint:gosec // cinq bits
		}
	default:
		return false
	}
	return true
}

// observation rend les crochets qui recoivent la relecture d une occurrence de role `role`.
func (l *lectureDEtatComplet) observation(role roleDOccurrence) *Observation {
	k := role.rang
	obs := &Observation{}
	switch role.genre {
	case roleGrenades:
		obs.GrenadeCountsHook = func(n uint64, v []uint64) {
			l.grenadesLues, l.compte, l.compteurs = true, n, append([]uint64(nil), v...)
		}
	case roleChargeur:
		obs.WeaponAmmoHook = func(aC bool, c uint32, aJ bool, j uint32) {
			l.chargeurLu[k], l.aChargeur[k], l.chargeur[k], l.aJauge[k], l.jauge[k] = true, aC, c, aJ, j
		}
	case roleReserve:
		obs.WeaponRoundsHook = func(v uint32) { l.reserveLue[k], l.reserve[k] = true, v }
	case roleSurchauffe:
		obs.WeaponOverheatHook = func(q uint32, b1, b2 bool) {
			l.surchauffeLue[k], l.surchauffe[k], l.drapeaux[k] = true, q, uint32(bit2u(b1))|uint32(bit2u(b2))<<1
		}
	case roleJeuDArmes:
		obs.DesiredWeaponSetHook = func(j JeuDArmes) { l.jeuLu, l.jeu = true, j }
	case roleArme:
		obs.HeldWeaponHook = func(h, _ uint32) { l.armeLue[k], l.armeHaute[k] = true, h }
	case roleJeuDeGrenades:
		obs.GrenadeSetHook = func(m uint32, s int) { l.jeuDeGrenadesLu, l.masque, l.selection = true, m, s }
	case roleCapacite:
		obs.AbilitySetHook = func(_ uint64, rk, _ int) { l.capaciteLue, l.rang = true, rk }
	}
	return obs
}
