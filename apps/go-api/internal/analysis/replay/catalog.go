package replay

// catalog.go — LE CATALOGUE DE LIBELLÉS DU TITRE, injecté par l'appelant.
//
// POURQUOI IL EST INJECTÉ ET NON CODÉ ICI. `internal/analysis/` est la couche des
// algorithmes purs : elle sait décoder un film, pas ce qu'est un Ravager. Les noms
// d'armes, de grenades et de capacités sont des CATALOGUES DE TITRE — ils vivent dans
// `config/titles/{slug}/mappings/` et sont chargés par
// `internal/games/halo_infinite/replaylabels`. Jusqu'au 2026-08-02 ils étaient codés
// ici, dont deux EN FRANÇAIS, ce qui interdisait l'anglais autant qu'un second titre.
//
// UN CATALOGUE VIDE EST UN CAS NORMAL, PAS UNE ERREUR : le document sort alors sans
// table de libellés, et le client affiche les identifiants bruts. C'est exactement la
// règle du chantier — mieux vaut un identifiant qu'un mot faux, parce qu'un mot faux se
// lit comme une certitude.

// LabelCatalog porte les trois tables de libellés d'un titre pour le rejeu 2D.
type LabelCatalog struct {
	// Weapons associe une FAMILLE d'arme (high-32 du weapon-id) à son libellé et à
	// l'effet de rendu de ses tirs. Une famille absente n'est pas nommée.
	Weapons map[uint32]WeaponLabel
	// Grenades nomme les rangs de type de grenade, DANS L'ORDRE DES RANGS. C'est la
	// seule table qui les nomme : un lancer porte son rang, pas son nom.
	Grenades []Label
	// Abilities nomme les index de capacité d'armure. Table partielle par nature.
	Abilities map[int]Label
}

// Empty dit si le catalogue ne nomme rien. Utile aux appelants qui veulent journaliser
// un rejeu volontairement anonyme plutôt que de le confondre avec un défaut de build.
func (c LabelCatalog) Empty() bool {
	return len(c.Weapons) == 0 && len(c.Grenades) == 0 && len(c.Abilities) == 0
}

// NewLabelCatalog assemble le catalogue à partir des tables DÉJÀ LUES du titre. C'est LA
// jointure `famille d'arme -> weapon_key -> {nom, effet}`, et elle n'existe qu'ici.
//
// POURQUOI LA JOINTURE EST DANS `replay` ET LA LECTURE DES FICHIERS AILLEURS : le
// chargement des TOML appartient au titre (games/halo_infinite/replaylabels), mais si la
// jointure y vivait aussi, aucun test de ce paquet ne pourrait la rejouer sans créer un
// cycle d'import — et le golden d'assemblage figerait alors des libellés de fixture au
// lieu des vrais. Une jointure recopiée dans un test est une jointure qui dérive.
//
// UNE FAMILLE SANS weapon_key, OU DONT LE weapon_key N'A PAS DE NOM, N'ENTRE PAS : elle
// gardera son hexadécimal à l'écran. Règle du chantier — un nom approchant se lit comme
// une certitude.
func NewLabelCatalog(
	familyToKey map[uint32]string,
	names map[string]Label,
	effects map[string]string,
	grenades []Label,
	abilities map[int]Label,
) LabelCatalog {
	weapons := make(map[uint32]WeaponLabel, len(familyToKey))
	for family, key := range familyToKey {
		name, ok := names[key]
		if !ok {
			continue
		}
		weapons[family] = WeaponLabel{En: name.En, Fr: name.Fr, Fx: effects[key]}
	}
	return LabelCatalog{Weapons: weapons, Grenades: grenades, Abilities: abilities}
}
