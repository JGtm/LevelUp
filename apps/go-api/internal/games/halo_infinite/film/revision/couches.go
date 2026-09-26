package revision

// couches.go — LA LISTE DES COUCHES REVISEES : ou la fermeture des imports s ARRETE (lot J3.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25).
//
// POURQUOI ELLE VIT ICI, ET PAS DANS CHAQUE GATE. La fermeture d une couche doit reconnaitre
// TOUTES les autres couches revisees pour les faire entrer par leur valeur : chaque gate aurait
// donc recopie la meme liste — cinq copies, la ou CLAUDE.md regle 6 impose de centraliser des la
// troisieme. Le paquet porte les NOMS et les RACINES, jamais les VALEURS : celles-ci restent des
// constantes de leurs couches, que chaque gate fournit ([Module.CalculerCouche]) — ce paquet
// n importe aucune couche.
//
// LE SECOND ORACLE (`equivalence_test.go`) REDECLARE CETTE LISTE A PART, et c est voulu : une
// divergence entre les deux declarations est exactement ce qu aucun gate de couche ne peut voir.

// racineDesCouches : le dossier commun des couches revisees, relatif au module.
const racineDesCouches = "internal/games/halo_infinite/film/internal/"

// CouchesRevisees rend les couches dont la revision est une constante hachee, dans l ORDRE DU
// SENS UNIQUE (ADR 0034 D-1) — l ordre dans lequel une fermeture hache les valeurs amont qu elle
// rencontre.
func CouchesRevisees() []Couche {
	return []Couche{
		{Nom: "source", Racine: racineDesCouches + "source"},
		{Nom: "profile", Racine: racineDesCouches + "profile"},
		{Nom: "grammar", Racine: racineDesCouches + "grammar"},
		{Nom: "facts", Racine: racineDesCouches + "facts"},
	}
}

// EmpreinteDeCouche est ce que le gate d une couche appelle : le module est trouve depuis le
// dossier de la couche, la liste d arret est [CouchesRevisees], et `valeurs` porte les revisions
// des couches que sa fermeture rencontre — ni plus, ni moins ([Module.CalculerCouche]).
func EmpreinteDeCouche(dossier, nom string, exclure func(rel string) bool,
	valeurs map[string]string,
) (Resultat, error) {
	m, err := ModuleDe(dossier)
	if err != nil {
		return Resultat{}, err
	}
	res, _, err := m.CalculerCouche(nom, CouchesRevisees(), exclure, valeurs)
	return res, err
}
