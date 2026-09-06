// Package domain — synthesis_weapon_range.go : LA PORTÉE ET LE DÉNIVELÉ DES ENGAGEMENTS,
// contrat de la section Synthèse (plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md, lot 4).
//
// # CE QUE LA SECTION RÉPOND
//
// « Où je frague, où je meurs, et d'en haut ou d'en bas » — la même mesure lue des DEUX
// côtés, arme par arme. Côté frags l'arme est la mienne ; côté morts c'est celle DU TUEUR,
// jamais celle que je tenais.
//
// # C'EST UN USAGE, JAMAIS UNE PORTÉE THÉORIQUE (D8)
//
// Ces nombres disent à quelle distance CE joueur a effectivement tué et s'est fait tuer avec
// cette arme. Aucun libellé produit ne doit dire « portée de l'arme » : le jeu ne publie nulle
// part de portée d'ingénierie, et ces mesures ne permettent pas de l'inférer.
//
// # DEUX COUVERTURES, TROIS DÉNOMINATEURS, ET AUCUN N'EST INTERCHANGEABLE
//
// Un frag n'est MESURÉ que si la position des deux joueurs est connue à l'instant du coup
// fatal ; l'entame en exige une seconde, un temps-pour-tuer plus tôt, et sa couverture est
// PARTIELLE PAR CONSTRUCTION tant que le backfill n'a pas tourné (D5). D'où
// `MeasuredKills` / `TotalKills` (la section publie « 1 214 frags mesurés sur 1 602 ») et un
// bloc `Opening` NIL — jamais un zéro — quand aucune entame n'est lue.
package domain

// SynthesisWeaponRange est le bloc « Portée par arme » de la Synthèse.
//
// Nil (champ omis) quand le titre ne produit pas de positions par kill, quand le scope est
// vide, ou quand la lecture échoue : la page n'affiche alors PAS la section, elle n'affiche
// pas une section vide.
type SynthesisWeaponRange struct {
	// Weapons : une ligne par arme, triée par médiane des frags croissante — le graphe se
	// lit comme un continuum du contact à la longue portée (D6). Une arme mesurée d'un seul
	// côté n'a qu'un côté renseigné.
	//
	// PEUT ÊTRE VIDE, ET LA SECTION RESTE PUBLIÉE (décision du 2026-09-06, constat F3 de la
	// revue adversariale du lot 4). Un joueur dont TOUTES les armes passent sous le seuil de
	// publication a bel et bien des mesures : les deux médianes globales, les deux
	// couvertures et les listes nommées ci-dessous portent l'information, seul le graphe par
	// arme n'a rien à tracer. Faire disparaître la section dans ce cas se lirait « aucune
	// mesure », ce qui est faux. Toujours sérialisé (`[]`, jamais `null`) : le front itère
	// sans garde.
	Weapons []WeaponRangeRow `json:"weapons"`

	// MedianKillsM / MedianDeathsM : les deux médianes GLOBALES, en mètres — le diagnostic
	// en deux nombres, au-dessus du graphe. Calculées sur TOUS les frags mesurés, jamais
	// comme une moyenne des médianes par arme (qui pèserait autant un couteau qu'un fusil).
	MedianKillsM  float64 `json:"median_kills_m"`
	MedianDeathsM float64 `json:"median_deaths_m"`

	// MeasuredKills / TotalKills, MeasuredDeaths / TotalDeaths : la couverture. Les totaux
	// viennent du scope canonique (les frags et morts du joueur sur la période), les
	// mesurés de la table de positions. L'écart est la part du corpus non décodée : il se
	// publie, il ne se cache pas.
	MeasuredKills  int `json:"measured_kills"`
	TotalKills     int `json:"total_kills"`
	MeasuredDeaths int `json:"measured_deaths"`
	TotalDeaths    int `json:"total_deaths"`

	// BelowThresholdKills / BelowThresholdDeaths : les armes ÉCARTÉES faute d'assez de
	// mesures (`analysis.WeaponRangeMinMeasured`), NOMMÉES et comptées, par côté (D9). Un
	// seuil qui cache en silence ferait croire que l'arme n'a jamais servi.
	BelowThresholdKills  []WeaponBelowThreshold `json:"below_threshold_kills,omitempty"`
	BelowThresholdDeaths []WeaponBelowThreshold `json:"below_threshold_deaths,omitempty"`

	// Opening : le proxy de distance d'entame (D5). NIL tant qu'aucune entame n'est
	// mesurée sur le scope — JAMAIS un bloc à zéro, qui se lirait « le joueur engage au
	// contact » alors que la vérité est « on ne sait pas ».
	Opening *SynthesisOpening `json:"opening,omitempty"`
}

// WeaponRangeRow est UNE arme, ses deux côtés côte à côte.
//
// Les deux côtés sont des POINTEURS et pas des structures nues : nil dit « aucune mesure de
// ce côté » et une valeur zéro dirait « mesuré, à zéro mètre ». La différence est exactement
// ce que l'infobulle du graphe doit rendre (« aucune mesure » vs « 0 m »).
type WeaponRangeRow struct {
	// WeaponKey est la clé canonique du registre ("hinf_br75"). TOUJOURS renseignée : elle
	// est le repli d'affichage quand le libellé n'a pas pu être résolu.
	WeaponKey string `json:"weapon_key"`
	// Label / LabelEN : le nom d'affichage FR-first et EN-first, résolus depuis la metadata
	// du titre (jamais un libellé en dur côté Go). Vides si la metadata ne connaît pas la
	// clé — le front retombe alors sur WeaponKey.
	Label   string `json:"label,omitempty"`
	LabelEN string `json:"label_en,omitempty"`

	// Kills : « où je frague avec cette arme ». Nil si aucun frag mesuré au-dessus du seuil.
	Kills *WeaponRangeSide `json:"kills,omitempty"`
	// Deaths : « où je meurs sous cette arme ». Nil si aucune mort mesurée au-dessus du seuil.
	Deaths *WeaponRangeSide `json:"deaths,omitempty"`
}

// WeaponRangeSide est UN côté d'une ligne : sa portée et son dénivelé.
type WeaponRangeSide struct {
	// Measured est l'effectif du côté — le dénominateur de tout le reste.
	Measured int `json:"measured"`

	// P10, Median, P90 : la portée mesurée, en mètres. p10 -> p90 et NON min -> max (D6) :
	// sur des centaines de frags, le minimum et le maximum décrivent deux accidents (un tir
	// de mêlée, un tir chanceux à travers la carte).
	P10    float64 `json:"p10"`
	Median float64 `json:"median"`
	P90    float64 `json:"p90"`

	// AbovePct / LevelPct / BelowPct : la ventilation du dénivelé, EN POURCENTAGE 0..100
	// (convention `*Pct` du dépôt), VUE DU CÔTÉ DEMANDÉ. Côté morts, « d'en haut » veut
	// donc dire que J'ÉTAIS au-dessus du tueur — pas l'inverse. Leur somme vaut 100 aux
	// arrondis près. Seuil des classes : un mètre (`analysis.WeaponRangeLevelBandM`), une
	// marche ou un rebord, pas un décalage de capsule entre deux joueurs debout.
	AbovePct float64 `json:"above_pct"`
	LevelPct float64 `json:"level_pct"`
	BelowPct float64 `json:"below_pct"`
}

// WeaponBelowThreshold nomme une arme écartée par le seuil de publication et dit combien de
// mesures elle portait. « Hydra (6) » se lit ; « 3 armes sous le seuil » ne se lit pas.
type WeaponBelowThreshold struct {
	WeaponKey string `json:"weapon_key"`
	Label     string `json:"label,omitempty"`
	LabelEN   string `json:"label_en,omitempty"`
	Measured  int    `json:"measured"`
}

// SynthesisOpening : la distance d'ENTAME et son écart au coup fatal (D5).
//
// L'entame vraie exigerait le premier dégât de l'échange, que le film n'émet pas. Le proxy est
// la distance un temps-pour-tuer (1,5 s) avant le coup fatal, validé le 2026-09-06 : écart
// médian 1,24 m sur 87 cas, contre un gate de 2 m écrit avant la mesure.
type SynthesisOpening struct {
	// MedianM est la médiane des distances d'entame des FRAGS, en mètres.
	MedianM float64 `json:"median_m"`
	// MeasuredKills est le nombre de frags dont l'entame est mesurée — la couverture du
	// proxy, à publier telle quelle (« 618 frags mesurés »).
	MeasuredKills int `json:"measured_kills"`

	// Delta : l'écart entame -> coup fatal, NIL quand aucun frag ne porte les DEUX mesures.
	//
	// POURQUOI UN SOUS-OBJET OPTIONNEL PLUTÔT QUE TROIS CHAMPS PLATS (2026-09-06, constat
	// F9 de la revue adversariale du lot 4). Les deux couvertures sont indépendantes :
	// `kill_positions` et `kill_openings` s'écrivent sous deux leases distincts, et un
	// scope où les entames sont lues mais aucune position de coup fatal ne l'est est
	// atteignable. À plat, ce cas publiait `delta_median_m: 0` et `closing_share_pct: 0` —
	// deux champs requis, donc toujours présents — qui se lisent « l'engagement ne se ferme
	// jamais » alors que la vérité est « on ne sait pas ». C'est exactement la faute que D5
	// interdit au niveau du bloc entier ; elle est ici interdite au niveau du delta.
	Delta *SynthesisOpeningDelta `json:"delta,omitempty"`
}

// SynthesisOpeningDelta : de l'entame au coup fatal, sur les frags qui portent LES DEUX
// mesures. Publié seulement quand il y en a au moins un.
type SynthesisOpeningDelta struct {
	// MedianM est la médiane, PAR FRAG APPARIÉ, de `distance au coup fatal − distance à
	// l'entame`. NÉGATIF = l'engagement se ferme. Calculée frag par frag et JAMAIS entre
	// deux médianes : les deux couvertures ne décrivent pas la même population
	// d'engagements (cf. analysis.WeaponOpeningDelta).
	MedianM float64 `json:"median_m"`
	// ClosingSharePct est la part des frags appariés dont la distance se ferme, en
	// pourcentage 0..100.
	ClosingSharePct float64 `json:"closing_share_pct"`
	// N est le nombre de frags APPARIÉS — le dénominateur des deux nombres ci-dessus, plus
	// petit que `SynthesisOpening.MeasuredKills` dès qu'une mesure manque d'un côté.
	N int `json:"n"`
}
