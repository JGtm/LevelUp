package mappings

// loader_replay_labels_flagzone.go — LA ZONE DE RETOUR DU DRAPEAU : la regle du mode, telle que
// le titre la declare.
//
// # POURQUOI CETTE TABLE EXISTE, ET POURQUOI ELLE N'EST PAS EN GO
//
// Le rejeu doit dessiner, autour d'un drapeau tombe, la zone dans laquelle ses coequipiers le
// renvoient — et la jauge qui se vide. Ce sont des grandeurs du JEU : un rayon, une minuterie.
// Les ecrire en Go ferait entrer une constante d'Halo Infinite dans un paquet title-agnostic, ce
// que le depot interdit. Elles vivent donc dans le manifeste du titre, avec leur provenance.
//
// # DEUX PROVENANCES, ET ELLES NE SE VALENT PAS
//
//	RAYON        releve dans le JEU : pool de constantes du script `parcel_deliver_object.lua`
//	             (tag `hsc*`), table CONFIG, champ `innerAreaMonitorRadius`. Le meme pool rend
//	             `flagCarrierMovespeedScalar = 0,715`, la penalite de vitesse du porteur — un
//	             temoin verifiable qui atteste le decodage.
//	MINUTERIE    MESUREE sur les films : le defaut de la bibliotheque (`flagResetSeconds = 15`)
//	             est ecrase par instance (`FlagInitArgs.returnTimer`), et c'est la valeur du MATCH
//	             qui compte pour un rejeu. Voir `.ai/V7.5/PLAN_CTF_ZONE_RETOUR_2026-08-30.md`.
//
// # ABSENTE = RIEN N'EST DESSINE
//
// Un titre qui ne declare pas la section n'a pas de zone de retour : le rejeu publie ses drapeaux
// comme avant, sans cercle ni jauge. C'est la degradation gracieuse habituelle — jamais une
// valeur par defaut inventee, qui dessinerait un cercle faux sur un titre qui n'en a pas.

import "fmt"

// FlagReturnZone porte la regle de retour du drapeau d'un titre. Toutes ses grandeurs sont
// positives ; une section presente mais incomplete est une ERREUR de configuration, pas une
// zone a moitie declaree.
type FlagReturnZone struct {
	// RadiusM est le rayon de la zone de RETOUR, dans les COORDONNEES DU REJEU (les memes que les
	// socles du catalogue de cartes et que les positions du film).
	RadiusM float64
	// ContestRadiusM est le rayon de la zone de CONTESTATION : un ENNEMI du camp proprietaire qui
	// s'y tient empeche le retour (`GetAnyEnemyTeamInOuterArea`, etats `Contested` /
	// `ContestedRefilling`).
	ContestRadiusM float64
	// ResetSeconds est la duree qu'un drapeau au sol met a rentrer TOUT SEUL, personne dans la
	// zone.
	ResetSeconds float64
	// SoloSeconds est la duree qu'il met quand UN defenseur s'y tient. Le jeu accelere ensuite en
	// serie HARMONIQUE avec le nombre de presents (`CalculateReturnRateHarmonic`) : deux
	// defenseurs valent 1 + 1/2, trois 1 + 1/2 + 1/3 — rendement decroissant, jamais lineaire.
	SoloSeconds float64
}

// Declared dit si le titre declare une zone de retour exploitable.
func (z FlagReturnZone) Declared() bool {
	return z.RadiusM > 0 && z.ContestRadiusM > 0 && z.ResetSeconds > 0 && z.SoloSeconds > 0
}

// flagReturnZoneTOML — projection brute de la section [flag_return_zone].
type flagReturnZoneTOML struct {
	RadiusM        float64 `toml:"radius_m"`
	ContestRadiusM float64 `toml:"contest_radius_m"`
	ResetSeconds   float64 `toml:"reset_seconds"`
	SoloSeconds    float64 `toml:"solo_seconds"`
}

// parseFlagReturnZone valide la section. Absente : zone nulle, et le rejeu se tait.
func parseFlagReturnZone(path string, in *flagReturnZoneTOML) (FlagReturnZone, error) {
	if in == nil {
		return FlagReturnZone{}, nil
	}
	z := FlagReturnZone{RadiusM: in.RadiusM, ContestRadiusM: in.ContestRadiusM,
		ResetSeconds: in.ResetSeconds, SoloSeconds: in.SoloSeconds}
	for _, c := range []struct {
		nom string
		v   float64
	}{{"radius_m", z.RadiusM}, {"contest_radius_m", z.ContestRadiusM},
		{"reset_seconds", z.ResetSeconds}, {"solo_seconds", z.SoloSeconds}} {
		if c.v <= 0 {
			return FlagReturnZone{}, fmt.Errorf("%s: [flag_return_zone].%s doit être > 0 (reçu %g)",
				path, c.nom, c.v)
		}
	}
	if z.SoloSeconds >= z.ResetSeconds {
		return FlagReturnZone{}, fmt.Errorf("%s: [flag_return_zone].solo_seconds (%g) doit être "+
			"inférieur à reset_seconds (%g) — s'y tenir ACCÉLÈRE le retour", path,
			z.SoloSeconds, z.ResetSeconds)
	}
	return z, nil
}
