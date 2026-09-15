package filmdec

// object_deaths_calibrate.go — LE CADRE DE LA BOUCLE DE RECORDS, balayé et publié, jamais
// supposé.
//
// TROIS GRANDEURS DE RUNTIME. `IDLowBits` (largeur du champ bas de l'identifiant de record),
// `PacketPreambleBits` (amorce du paquet) et la présence de champs supplémentaires ne sont PAS
// écrites dans le film : elles dépendent de la configuration du serveur qui l'a produit.
// Mesurées : idLow vaut 11 sur `000d5950`, 13 sur d'autres, 14 sur un film de capture live.
//
// LE CRITÈRE DE SÉLECTION EST LE TAUX DE LOCALISATION, pas le rendement en records. La mesure
// qui l'impose (lot V13, 2026-09-05) : sur `0d76e8f1`, `BestVariant` — qui compte les bits
// consommés sur UN paquet dans un monde neuf — désigne idLow=10, cadre sous lequel la marche
// complète localise 0 paquet à événements sur 12 et rend pourtant 492 records dits « propres » ;
// idLow=13 en localise 97,4 %. Un cadre faux consomme des bits sans rien décoder de vrai ; seule
// la signature du slot 123 (un delta de 35 bits exactement) le démasque.
//
// LE COÛT EST BORNÉ par l'échantillon : au plus `calibPacketBudget` paquets et
// `calibEventBudget` paquets à événements par cadre candidat, sur 18 candidats.

// calibPacketBudget / calibEventBudget bornent l'échantillon de calibrage. Le budget est
// exprimé EN PAQUETS À ÉVÉNEMENTS parce que le critère de sélection est leur taux de
// localisation ; la borne en paquets n'est là que pour ne pas marcher un film entier quand un
// film n'en porte aucun.
const (
	calibPacketBudget = 3000
	calibEventBudget  = 150
)

// calibIDLowMin / calibIDLowMax et calibPreambleMax bornent le balayage des cadres candidats.
// Les bornes viennent des valeurs OBSERVÉES (11, 13, 14) élargies d'un cran de chaque côté ;
// l'amorce, elle, est une propriété du format et ne prend que trois valeurs plausibles.
const (
	calibIDLowMin    = 10
	calibIDLowMax    = 15
	calibPreambleMax = 2
)

// frameConfigScore est le rendement d'un cadre candidat sur l'échantillon de calibrage.
type frameConfigScore struct {
	cfg      FrameConfig
	located  int
	events   int
	packets  int
	records  int
	cleanRec int
}

// better dit si `s` l'emporte sur `other` : d'abord le nombre de paquets à événements
// LOCALISÉS, puis, à égalité, le nombre de records entièrement portés.
func (s frameConfigScore) better(other frameConfigScore) bool {
	if s.located != other.located {
		return s.located > other.located
	}
	return s.cleanRec > other.cleanRec
}

// calibrateFrameConfig balaye les cadres candidats et rend le meilleur, avec son score.
//
// MONDE NEUF À CHAQUE ESSAI : un candidat ne doit rien emporter du précédent.
func calibrateFrameConfig(
	reg *Registry, kfs []marchKeyframe, deltas []marchDelta,
) (FrameConfig, frameConfigScore) {
	best := frameConfigScore{cfg: DefaultFrameConfig(), located: -1, cleanRec: -1}
	for pre := 0; pre <= calibPreambleMax; pre++ {
		for low := calibIDLowMin; low <= calibIDLowMax; low++ {
			cfg := DefaultFrameConfig()
			cfg.IDLowBits, cfg.PacketPreambleBits = low, pre
			s := trialFrameConfig(reg, kfs, deltas, cfg)
			if s.better(best) {
				best = s
			}
		}
	}
	return best.cfg, best
}

// trialFrameConfig marche l'échantillon sous un cadre donné et rend son rendement.
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
		for _, r := range marchRecordsOf(d.payload, w, cfg, start) {
			out.records++
			if r.DesyncAt == -1 {
				out.cleanRec++
			}
		}
	}
	return out
}
