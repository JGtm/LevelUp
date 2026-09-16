package filmdec

// object_deaths_calibrate.go — LA LARGEUR DU CHAMP BAS D IDENTIFIANT, BALAYEE ET PUBLIEE, SOUS
// GARDE-FOU.
//
// UNE SEULE GRANDEUR EST BALAYEE ICI, ET C EST DELIBERE.
//
//	`IDLowBits`          est une valeur de RUNTIME : elle n est ecrite nulle part dans le film et
//	                     vaut 11 sur un film, 13 sur un autre, 14 sur une capture live
//	                     (`frame_records.go`, en-tete de DefaultFrameConfig). Elle se balaye.
//	`PacketPreambleBits` est une propriete du FORMAT, PROUVEE, et elle ne se balaye PAS. Trois
//	                     temoins independants et concordants la fixent a 2 (`frame_records.go`,
//	                     en-tete de DefaultPacketPreambleBits) : le R(1) du desassemblage
//	                     (FUN_142987460 -> FUN_1406cf008), ce bit a 1 dans 100,00 % des 30 418
//	                     payloads de `000d5950`, et la FORME DU MASQUE (84,81 % de masques 1..7
//	                     a l amorce 2 contre 14,65 % a l amorce 0, niveau du hasard 10,67 %).
//	                     La balayer serait re-decider par heuristique ce que la grammaire etablit
//	                     — exactement ce que D13 interdit. Mesure qui l a impose (2026-09-16,
//	                     bobine `minibobine_000d5950`) : entre l amorce 2 et l amorce 1, a
//	                     `idLow=13`, la localisation est IDENTIQUE (142/147) et seul le rendement
//	                     en records propres departage, de 4 % (6 993 contre 6 730) — un
//	                     departage de 4 % sur le critere que l en-tete de ce fichier declare
//	                     precisement peu fiable.
//
// LE CRITERE EST LE TAUX DE LOCALISATION, ET RIEN D AUTRE. La signature du slot 123 (un delta de
// 35 bits EXACTEMENT, a composant unique) est un oracle fort du cadre. Le rendement en records
// « propres » ne l est pas : un cadre faux consomme des bits sans rien decoder de vrai et se
// referme quand meme. Mesure (lot V13, `0d76e8f1`) : `BestVariant` designe `idLow=10`, cadre sous
// lequel la marche complete localise 0 paquet a evenements sur 12 tout en rendant 492 records
// dits propres ; `idLow=13` en localise 97,4 %.
//
// LE GARDE-FOU, REPRIS DE `killsource/calibrate.go` : le candidat retenu doit DOMINER son
// dauphin d un facteur `calibDominationMin`. Sinon le profil est PLAT — le parametre reel n est
// pas dans l espace balaye, ou le film ne porte pas de quoi trancher — et alors ON NE DEVINE
// PAS : le cadre par defaut est conserve, le fait est PUBLIE (`ObjectDeathStats.CadreParDefaut`)
// et compte comme repli (`repli_cadre_de_marche_par_defaut_conserve`). Mesure qui l a impose
// (2026-09-16) : sur `minibobine_e5adf7b2` les DIX-HUIT cadres localisent 0 paquet sur 54, et le
// code retenait pourtant un cadre, en silence, au departage par records propres.

import "sort"

// calibPacketBudget / calibEventBudget bornent l echantillon de calibrage. Le budget est exprime
// EN PAQUETS A EVENEMENTS parce que le critere est leur taux de localisation ; la borne en
// paquets n existe que pour ne pas marcher un film entier quand il n en porte aucun.
const (
	calibPacketBudget = 3000
	calibEventBudget  = 150
)

// calibIDLowMin / calibIDLowMax bornent l espace balaye : les valeurs OBSERVEES (11, 13, 14)
// elargies d un cran de chaque cote.
const (
	calibIDLowMin = 10
	calibIDLowMax = 15
)

// calibDominationMin est le facteur par lequel le candidat retenu doit battre son dauphin sur le
// nombre de paquets a evenements LOCALISES. Deux, la valeur de `killsource/calibrate.go`
// (`flatRatio`), et la marge mesuree sur `minibobine_000d5950` est de 6,2 (142 contre 23) : le
// seuil separe un profil franc d un profil plat sans couper aucun cas reel connu.
const calibDominationMin = 2

// frameConfigScore est le rendement d un cadre candidat sur l echantillon de calibrage.
type frameConfigScore struct {
	cfg     FrameConfig
	located int
	events  int
	packets int
}

// calibrateFrameConfig balaye les largeurs candidates et rend le cadre retenu.
//
// `parDefaut` vrai = le profil est PLAT et le cadre rendu est `DefaultFrameConfig()` : la marche
// tourne, mais sur une largeur que rien n a confirmee — l appelant DOIT le publier.
func calibrateFrameConfig(
	reg *Registry, kfs []marchKeyframe, deltas []marchDelta,
) (cfg FrameConfig, parDefaut bool, meilleur, dauphin frameConfigScore) {
	defautLow := DefaultFrameConfig().IDLowBits
	scores := make([]frameConfigScore, 0, calibIDLowMax-calibIDLowMin+1)
	for low := calibIDLowMin; low <= calibIDLowMax; low++ {
		c := DefaultFrameConfig()
		c.IDLowBits = low
		scores = append(scores, trialFrameConfig(reg, kfs, deltas, c))
	}
	// TRI DETERMINISTE, et son second critere N EST PAS un departage de qualite : a egalite de
	// localisation, c est le cadre PAR DEFAUT qui passe devant, puis la plus petite largeur.
	// Departager deux ex aequo par leur rendement en records serait rouvrir la porte au critere
	// que le lot V13 a refute ; a egalite, on ne choisit pas, on garde ce qu on avait.
	sort.SliceStable(scores, func(i, j int) bool {
		switch {
		case scores[i].located != scores[j].located:
			return scores[i].located > scores[j].located
		case (scores[i].cfg.IDLowBits == defautLow) != (scores[j].cfg.IDLowBits == defautLow):
			return scores[i].cfg.IDLowBits == defautLow
		default:
			return scores[i].cfg.IDLowBits < scores[j].cfg.IDLowBits
		}
	})
	meilleur, dauphin = scores[0], scores[1]
	if meilleur.located < calibDominationMin*max(dauphin.located, 1) {
		return DefaultFrameConfig(), true, meilleur, dauphin
	}
	return meilleur.cfg, false, meilleur, dauphin
}

// trialFrameConfig marche l echantillon sous un cadre donne et rend son taux de localisation.
//
// MONDE NEUF A CHAQUE ESSAI : un candidat ne doit rien emporter du precedent.
func trialFrameConfig(
	reg *Registry, kfs []marchKeyframe, deltas []marchDelta, cfg FrameConfig,
) frameConfigScore {
	out := frameConfigScore{cfg: cfg}
	tl := newMarchTimeline(reg, kfs)
	for i := range deltas {
		if i >= calibPacketBudget || out.events >= calibEventBudget {
			break
		}
		d := deltas[i]
		w := tl.advanceTo(d.timestampUS)
		start, withEvents, ok := marchStartOf(d.payload, w, cfg)
		if withEvents {
			out.events++
		}
		if !ok {
			continue
		}
		if withEvents {
			out.located++
		}
		out.packets++
		// LA MARCHE TOURNE QUAND MEME : elle applique au monde les liaisons NEW/DEL du paquet,
		// et les paquets suivants en dependent. Ses records ne sont pas comptes — le critere ne
		// les regarde pas.
		marchRecordsOf(d.payload, w, cfg, start)
	}
	return out
}
