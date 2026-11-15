package domain

// spawn_model.go — LE FICHIER DES POIDS DU MODELE DE REAPPARITION.
//
// Il est ecrit hors serveur (`levelup spawn-model fit`), versionne avec le depot, et relu tel
// quel par le service. Un poids se nomme par sa VARIABLE (analysis/tactical.NomsVariables) :
// ajouter ou retirer une variable ne decale jamais les poids existants.

// SpawnModelRevision est la version du FORMAT du fichier de poids. Un fichier d'une autre
// revision n'est pas exploitable.
const SpawnModelRevision = 1

// FamilleToutesModes est la famille qui regroupe tous les modes mesures ; elle sert de repli
// quand un mode n'a pas ses propres poids (analysis/tactical.PoidsPourFamille). Les familles
// elles-memes sont celles de la table du titre (config/titles/{slug}/mappings/spawn_families.toml).
const FamilleToutesModes = "tous"

// PoidsReapparition est le contenu du fichier.
type PoidsReapparition struct {
	Revision int `json:"revision"`
	// Date : le jour de l'ajustement (AAAA-MM-JJ).
	Date string `json:"date"`
	// Definition dit ce que le fichier mesure : d'ou viennent les reapparitions, quels filtres les
	// retiennent, et ce que comptent les effectifs de chaque famille.
	Definition string `json:"definition"`
	// Familles : les poids par famille de modes, cle = nom de la famille.
	Familles map[string]PoidsFamille `json:"familles"`
}

// PoidsFamille : les poids d'une famille de modes, avec l'effectif et les scores qui les
// justifient (validation croisee par match).
type PoidsFamille struct {
	// Poids : variable -> poids. Une variable absente a un poids nul.
	Poids map[string]float64 `json:"poids"`
	// Reapparitions, Cartes, Matchs : l'effectif de l'ajustement, compte sur les memes
	// reapparitions retenues (apres filtres) : celles qui ont servi a ajuster et a evaluer.
	Reapparitions int `json:"reapparitions"`
	Cartes        int `json:"cartes"`
	Matchs        int `json:"matchs"`
	// Scores : la part de reapparitions retrouvees, en pourcentage.
	Scores ScoresReapparition `json:"scores"`
}

// ScoresReapparition : ce que vaut la prevision, en pourcentage, par echelle. « Hasard » est le
// tirage uniforme parmi les candidats ; « Frequence » ne regarde que la frequence du point.
type ScoresReapparition struct {
	Top1Hasard    float64 `json:"top1_hasard"`
	Top1Frequence float64 `json:"top1_frequence"`
	Top1          float64 `json:"top1"`
	Top3          float64 `json:"top3"`
	// Rayon1Hasard et Rayon3Hasard : la chance qu'un premier tirage, ou qu'au moins un des trois
	// tirages sans remise, tombe a moins de 8 m du point reel (esperance exacte par evenement).
	Rayon1Hasard float64 `json:"rayon8m_top1_hasard"`
	Rayon3Hasard float64 `json:"rayon8m_top3_hasard"`
	Rayon3       float64 `json:"rayon8m_top3"`
}
