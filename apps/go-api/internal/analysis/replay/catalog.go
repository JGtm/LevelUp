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

// WeaponIconRef pointe l'icône EXTRAITE DU JEU d'une famille d'arme, telle que le titre
// la sert. Tinted dit si le visuel est un masque à teindre (cf. WeaponImageIsTinted).
type WeaponIconRef struct {
	URL    string
	Tinted bool
}

// LabelCatalog porte les tables de libellés et d'icônes d'un titre pour le rejeu 2D.
type LabelCatalog struct {
	// Weapons associe une FAMILLE d'arme (high-32 du weapon-id) à son libellé et à
	// l'effet de rendu de ses tirs. Une famille absente n'est pas nommée.
	Weapons map[uint32]WeaponLabel
	// Grenades nomme les rangs de type de grenade, DANS L'ORDRE DES RANGS. C'est la
	// seule table qui les nomme : un lancer porte son rang, pas son nom.
	Grenades []Label
	// Abilities porte les PALETTES de capacité d'armure du titre. Le rang transmis par le
	// film est un index dans un groupe de tags choisi à l'exécution : deux films peuvent
	// donner deux capacités différentes au même rang. C'est `classifyAbilityPalette`
	// (abilities.go) qui choisit laquelle s'applique, et qui refuse quand la signature du
	// film est ambiguë. Tables partielles par nature.
	Abilities []AbilityPalette
	// Icons pointe l'icône extraite d'une famille d'arme. Posée par la COUCHE TITRE
	// (elle seule connaît ses URLs d'assets), APRÈS NewLabelCatalog — d'où un champ et
	// pas un sixième paramètre. Une famille absente garde son libellé sans visuel :
	// le client affiche alors le texte, jamais l'icône d'une arme voisine.
	Icons map[uint32]WeaponIconRef
	// Keys est la table FAMILLE d'arme -> weapon_key, telle que le registre du titre la
	// donne. C'est la jointure que le service pose sur le document à la requête
	// (WeaponLabel.Key) : un tir du film porte un identifiant d'arme, les tables du
	// client (banque de sons) sont keyées par weapon_key.
	//
	// Weapons n'en garde que la PROJECTION nommée : une famille dont le weapon_key n'a
	// pas de nom n'y entre pas, alors qu'elle a bel et bien une clé — et un son.
	Keys map[uint32]string
	// Tints est la table weapon_key -> NATURE DE LA DECHARGE (kinetic, plasma_cool,
	// plasma_hot, forerunner, electric, needle, blast). Posée par la COUCHE TITRE après
	// NewLabelCatalog, comme Icons — un sixième paramètre ferait sauter le seuil du
	// dépôt, et cette table n'entre dans aucune jointure de construction.
	//
	// ELLE EST DISTINCTE DE Effects, et les deux ne se recouvrent pas : la FORME suit la
	// mécanique du projectile, la TEINTE sa nature énergétique. Deux armes `plasma`
	// n'ont pas la même couleur dans le jeu ; deux armes de même couleur n'ont pas la
	// même forme. Détail et raison dans replay_labels.toml.
	Tints map[string]string
	// Effects est la table weapon_key -> famille de rendu TELLE QUE LUE du titre
	// ([shot_effects]). Weapons n'en garde que la projection par famille d'arme FILM ;
	// or les kills du feed sont keyés par weapon_key — cette table est publiée telle
	// quelle (ReplayDocument.KillEffects) pour que le client puisse les joindre.
	Effects map[string]string
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
	abilities []AbilityPalette,
) LabelCatalog {
	weapons := make(map[uint32]WeaponLabel, len(familyToKey))
	for family, key := range familyToKey {
		name, ok := names[key]
		if !ok {
			continue
		}
		weapons[family] = WeaponLabel{En: name.En, Fr: name.Fr, Fx: effects[key]}
	}
	// La table brute est COPIÉE : le catalogue survit à son appelant, il ne doit pas
	// partager une map que celui-ci pourrait muter.
	var eff map[string]string
	if len(effects) > 0 {
		eff = make(map[string]string, len(effects))
		for k, v := range effects {
			eff[k] = v
		}
	}
	// La table des CLÉS est copiée pour la même raison que celle des effets : le
	// catalogue survit à son appelant et ne doit pas partager une map mutable.
	var keys map[uint32]string
	if len(familyToKey) > 0 {
		keys = make(map[uint32]string, len(familyToKey))
		for f, k := range familyToKey {
			keys[f] = k
		}
	}
	return LabelCatalog{
		Weapons:   weapons,
		Grenades:  grenades,
		Abilities: abilities,
		Keys:      keys,
		Effects:   eff,
	}
}
