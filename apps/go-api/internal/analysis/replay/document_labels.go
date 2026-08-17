package replay

// document_labels.go — LES LIBELLES que l artefact embarque : le nom bilingue d une arme, d une
// grenade ou d une capacite, et ce qui se dessine avec.
//
// Extrait de `document.go` au correctif de revue du 2026-08-17, pour la meme raison que
// `document_ground_weapons.go` : le lot des socles avait pousse ce fichier de 631 a 673 lignes,
// au-dessus d un seuil deja gele par la baseline. AUCUNE ligne de ces deux types n a change —
// c est un deplacement, verifie par le golden d assemblage qui fige les libelles servis.

// Label est un libellé affichable dans les deux langues du produit.
//
// POURQUOI DEUX LANGUES DANS L'ARTEFACT, et pas une résolution au service : l'artefact
// est construit UNE FOIS, hors ligne, et servi tel quel — la locale, elle, change à
// chaque requête. Y figer une seule langue reviendrait à choisir la langue du lecteur au
// moment du décodage d'un film.
type Label struct {
	En string `json:"en"`
	Fr string `json:"fr"`
	// Img est l'URL de la vignette du HUD du jeu (grenades, capacités des fiches joueur).
	// Vide = pas de visuel : le client garde le libellé, jamais la vignette d'un voisin.
	// Tinted dit si le visuel est un masque à teindre (même contrat que WeaponLabel).
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}

// WeaponLabel est le libellé d'une arme, plus l'EFFET de rendu de ses tirs.
//
// L'effet vit à côté du nom parce qu'il se résout au même endroit et à partir de la même
// clé (le weapon_key du titre). Le publier ici est ce qui a permis de retirer du code web
// le catalogue des 22 noms d'armes Halo : le client dessine ce que le document dit, il
// n'a plus à savoir ce qu'est un Ravager.
type WeaponLabel struct {
	En string `json:"en"`
	Fr string `json:"fr"`
	// Fx est la famille de RENDU du tir (ballistic, plasma, light, shock, explosive,
	// melee, needles). Vide = arme non catégorisée : le client dessine le trait neutre,
	// jamais l'effet d'une arme voisine.
	Fx string `json:"fx,omitempty"`
	// Key est le weapon_key du titre (clé canonique du registre d'armes).
	//
	// POURQUOI IL EST PUBLIÉ : c'est le SEUL vocabulaire commun entre un tir du film
	// (qui porte un identifiant d'arme 64 bits) et les tables que le client tient par
	// weapon_key — la banque de sons du rejeu au premier chef. Sans lui, un tir ne peut
	// pas sonner l'arme qui l'a produit, et lui faire emprunter le son d'une voisine
	// serait un mensonge sonore. `killEffects` publie déjà ce même vocabulaire pour les
	// morts : la clé n'est pas un identifiant interne qui fuite, c'est la jointure.
	//
	// IL N'EST PAS ÉCRIT DANS L'ARTEFACT : il est rempli À LA REQUÊTE par le service
	// (replay_weapon_keys.go), comme `mapObjectives`. La raison est mesurée — figer la
	// clé au build laisserait muets les artefacts déjà cuits (23 en local, tous ceux de
	// la production) jusqu'à une re-cuisson complète, et une résolution qui peut
	// s'améliorer ne se stocke pas. Vide = le titre n'a pas de catalogue lisible, ou
	// l'arme n'est pas au registre : silence propre, jamais un son approchant.
	Key string `json:"key,omitempty"`
	// Tint est la NATURE DE LA DÉCHARGE (kinetic, plasma_cool, plasma_hot, forerunner,
	// electric, needle, blast) : ce qui sort du canon, jamais une couleur ni un camp.
	// C'est elle qui teinte l'éclair de bouche côté client — la COULEUR, elle, est un
	// token du thème, et c'est ce qui lui permet de valoir deux valeurs selon le thème.
	//
	// DISTINCTE DE Fx, et les deux ne se recouvrent pas : la forme suit la mécanique du
	// projectile, la teinte sa nature énergétique. Source : `[shot_tints]` du titre,
	// posée à la requête comme Key. Vide = arme non teintée (mêlée, hors table) : teinte
	// neutre du thème, jamais celle d'une voisine.
	Tint string `json:"tint,omitempty"`
	// Img est l'URL de l'icône EXTRAITE DU JEU (fiches joueur du rejeu). Vide = pas de
	// visuel : le client affiche le libellé, jamais l'icône d'une arme voisine. Tinted
	// dit si le visuel est un masque à teindre (même contrat que le kill feed).
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}
