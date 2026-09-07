package domain

// squad_isolement.go — LE NUAGE « ISOLEMENT X COUVERTURE » DE LA PAGE ESCOUADE (plan
// tactique, phase 7, item 7.7 / lot « 7B »).
//
// Un point par (joueur, session) : la part de morts ISOLEES du joueur sur cette session
// (aucun coequipier VISIBLE a portee du radar, cf. analysis/coordination.Isolement) contre
// son taux d'ECHANGE sur la MEME session (la mesure de la section « echange » ci-dessus,
// restreinte a ce seul joueur et a ce seul groupe de matchs).
//
// POURQUOI LA SESSION ET PAS LE MATCH (maquette echange-escouade.html, carte « Pourquoi la
// vengeance ne vient pas ») : un point par mort donnerait un axe binaire, pas un nuage ; un
// point par match donnerait des taux calcules sur trois morts, qui ne valent que 0, 33, 50
// ou 100 % — un damier, pas une dispersion. La session est la plus petite maille ou un taux
// veut dire quelque chose.

// PlancherMortsSessionIsolement est le nombre minimal de morts EXAMINEES (cf.
// MortAExaminer) sous lequel une session n'entre PAS dans le nuage : 5 (maquette
// echange-escouade.html, "Une session sous 5 morts n'entre pas dans le nuage").
//
// CE PLANCHER PORTE SUR LA PUBLICATION DU POINT, PAS SUR LA CONFIANCE DU TAUX — c'est
// coordination.SeuilEchantillonFaible (30) qui pose l'EchantillonFaible de PartIsolee et de
// Couverture. Les deux seuils coexistent et ne se confondent pas : sous 5, il n'y a rien a
// montrer ; entre 5 et 30, le point existe mais reste marque comme peu sur.
const PlancherMortsSessionIsolement = 5

// SquadIsolementPoint est UN point du nuage : un joueur, une session.
type SquadIsolementPoint struct {
	XUID         string `json:"xuid"`
	Gamertag     string `json:"gamertag"`
	SessionLabel string `json:"session_label"`

	// MortsExaminees est le denominateur de PartIsolee : les morts du joueur sur cette
	// session ayant au moins un coequipier en mesure d'accompagner (« equipe a terre »
	// et matchs sans rayon exclus). C'est aussi la TAILLE du point dans le nuage.
	MortsExaminees int `json:"morts_examinees"`
	// MortsIsolees est le numerateur : parmi les morts examinees, celles sans coequipier
	// VISIBLE a portee du radar.
	MortsIsolees int `json:"morts_isolees"`

	// PartIsolee est le taux d'isolement de la session, sous sa forme canonique (jamais
	// un float nu) : Brut = MortsIsolees, N = MortsExaminees.
	PartIsolee Couverture `json:"part_isolee"`
	// Couverture est le taux d'ECHANGE du joueur sur la MEME session (morts vengees sur
	// morts vengeables) — la meme mesure que la section « echange », restreinte a ce seul
	// joueur et a cette seule session.
	Couverture Couverture `json:"couverture"`
}

// SquadNuageIsolement est la section « nuage isolement x couverture » du bloc Echange de la
// page Escouade.
//
// Absente (nil) quand le titre n'a pas de table de portee de radar cablee, ou quand aucun
// point ne franchit PlancherMortsSessionIsolement : une OMISSION, jamais un nuage vide
// affiche comme une mesure a zero.
type SquadNuageIsolement struct {
	Points []SquadIsolementPoint `json:"points"`

	// PlancherMortsSession republie PlancherMortsSessionIsolement : le client le NOMME
	// dans son pied de carte plutot que de le recopier en dur.
	PlancherMortsSession int `json:"plancher_morts_session"`
	// PlancherEchantillonFaible republie coordination.SeuilEchantillonFaible : le seuil
	// sous lequel PartIsolee.EchantillonFaible / Couverture.EchantillonFaible passe a
	// vrai.
	PlancherEchantillonFaible int `json:"plancher_echantillon_faible"`
}
