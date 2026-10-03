package squademprise

// production.go — CE QUE CHAQUE CAMP A PRODUIT D'UNE RESSOURCE (cartes « Frags obtenus avec les
// ressources » et « Rendement face à l'adversaire »).
//
//	bonus            frags pendant l'effet, sur le temps d'effet (frags par minute d'effet) ;
//	armes spéciales  frags obtenus avec (feuille de match), sur les prises (frags par prise).
//
// UN RENDEMENT SE LIT SUR UN SEUL PÉRIMÈTRE. Les frags aux armes spéciales viennent de la feuille
// de match, qui couvre aussi les matchs sans film ; les prises, elles, n'existent que là où les
// niveaux de socle sont mesurés. Le rendement divise donc les frags de CES matchs-là
// (Exposure.Kills) par leurs prises ; Kills garde tous les matchs du périmètre. Pour les bonus,
// un match dont le film n'a pas d'échelle de temps n'apporte ni frags ni temps d'effet
// (timeScaled) : Kills = Exposure.Kills, un seul périmètre là aussi.

import "levelup/go-api/internal/domain"

// production rend les ressources produites ; une ressource sans mesure est absente.
func production(s *soiree) []domain.SquadEmpriseProduction {
	out := []domain.SquadEmpriseProduction{}
	if p, ok := productionBonus(s); ok {
		out = append(out, p)
	}
	if p, ok := productionArmes(s); ok {
		out = append(out, p)
	}
	if p, ok := productionVehicules(s); ok {
		out = append(out, p)
	}
	return out
}

func productionBonus(s *soiree) (domain.SquadEmpriseProduction, bool) {
	ms := s.effectMS
	kills := domain.SquadEmpriseCount{Us: s.effectKills[0], Them: s.effectKills[1]}
	if s.bonusMatches == 0 || (ms[0]+ms[1] == 0 && kills.Us+kills.Them == 0) {
		return domain.SquadEmpriseProduction{}, false
	}
	p := domain.SquadEmpriseProduction{
		Resource: domain.EmpriseResourcePowerup, Kills: kills,
		Exposure: &domain.SquadEmpriseExposure{
			Kind:  domain.EmpriseExposureEffectMS,
			Value: domain.SquadEmpriseCount{Us: int(ms[0]), Them: int(ms[1])},
			Kills: kills,
		},
		YieldUs:   parMinute(kills.Us, ms[0]),
		YieldThem: parMinute(kills.Them, ms[1]),
	}
	p.RelativeGap = ecartRelatif(p.YieldUs, p.YieldThem)
	return p, true
}

func productionArmes(s *soiree) (domain.SquadEmpriseProduction, bool) {
	if s.pwkMatches == 0 {
		return domain.SquadEmpriseProduction{}, false
	}
	p := domain.SquadEmpriseProduction{Resource: domain.EmpriseResourcePowerWeapon, Kills: s.pwk}
	if s.tiersMatches == 0 {
		return p, true
	}
	prises := s.obj.total(domain.EmpriseResourcePowerWeapon)
	p.Exposure = &domain.SquadEmpriseExposure{
		Kind: domain.EmpriseExposurePickups, Value: prises, Kills: s.pwkOnTiers,
	}
	p.YieldUs = parPrise(s.pwkOnTiers.Us, prises.Us)
	p.YieldThem = parPrise(s.pwkOnTiers.Them, prises.Them)
	p.RelativeGap = ecartRelatif(p.YieldUs, p.YieldThem)
	return p, true
}

// parMinute — frags par minute d'effet ; nil sans temps d'effet.
func parMinute(kills int, ms int64) *float64 {
	if ms <= 0 {
		return nil
	}
	v := float64(kills) / (float64(ms) / 60_000)
	return &v
}

// parPrise — frags par prise ; nil sans prise.
func parPrise(kills, prises int) *float64 {
	if prises <= 0 {
		return nil
	}
	v := float64(kills) / float64(prises)
	return &v
}

// ecartRelatif — notre rendement / le sien − 1 ; nil quand l'un manque ou que le sien est nul.
func ecartRelatif(us, them *float64) *float64 {
	if us == nil || them == nil || *them == 0 {
		return nil
	}
	v := *us / *them - 1
	return &v
}
