// Package playerdirectory — anomalies.go : lecture des incohérences entre
// registres (ADR 0035 D2).
//
// computeAnomalies est une FONCTION PURE de la ligne d'identité : elle ne lit ni
// fichier, ni disque, ni horloge. C'est ce qui permet de la tester cas par cas
// (`anomalies_test.go`) et de garantir que deux lectures de la même ligne
// donnent les mêmes badges.
//
// Le résultat porte des CODES machine et un Detail de contexte (slug de titre,
// nom de dossier) : le libellé affiché est construit côté web, en FR et en EN.
package playerdirectory

import "levelup/go-api/internal/domain"

// computeAnomalies rend les anomalies d'une identité, dans un ordre stable :
// les `warning` (incohérences) d'abord, puis les `info` (états normaux qu'un
// administrateur doit pouvoir reconnaître au lieu de les prendre pour des
// pannes : un profil d'ami n'a pas de compte, un profil servi par le pool
// d'auth n'a pas de credentials propres).
func computeAnomalies(rec domain.IdentityRecord) []domain.IdentityAnomaly {
	out := make([]domain.IdentityAnomaly, 0, 2)
	hasProfile := len(rec.Profiles) > 0

	// Un compte qui peut se connecter alors que rien ne le suit : l'état exact
	// du compte du 2026-07-23, celui que cet annuaire existe pour montrer.
	if rec.Account != nil && !hasProfile {
		out = append(out, warning(domain.AnomalyAccountWithoutProfile, rec.Account.Username))
	}
	// Des credentials que plus personne ne réclame.
	if rec.Token != nil && rec.Account == nil && !hasProfile {
		out = append(out, warning(domain.AnomalyTokenOrphan, rec.XUID))
	}
	for _, dir := range rec.OrphanDirs {
		out = append(out, warning(domain.AnomalyPlayerDirOrphan, dir.TitleSlug+"/"+dir.Name))
	}
	for _, slug := range rec.Watched {
		if !hasProfileForTitle(rec, slug) {
			out = append(out, warning(domain.AnomalyWatchedWithoutProfile, slug))
		}
	}

	if hasProfile && rec.Account == nil {
		out = append(out, info(domain.AnomalyProfileWithoutAccount, ""))
	}
	if hasProfile && rec.Token == nil {
		out = append(out, info(domain.AnomalyProfileWithoutToken, ""))
	}
	return out
}

// hasProfileForTitle : un profil existe-t-il pour CE titre ? La question se pose
// par titre et jamais globalement — un profil Halo 5 ne justifie pas un suivi
// live Halo Infinite (même piège que la normalisation des portes, ADR 0035 D3).
func hasProfileForTitle(rec domain.IdentityRecord, titleSlug string) bool {
	for _, p := range rec.Profiles {
		if p.TitleSlug == titleSlug {
			return true
		}
	}
	return false
}

func warning(code, detail string) domain.IdentityAnomaly {
	return domain.IdentityAnomaly{Code: code, Severity: domain.AnomalySeverityWarning, Detail: detail}
}

func info(code, detail string) domain.IdentityAnomaly {
	return domain.IdentityAnomaly{Code: code, Severity: domain.AnomalySeverityInfo, Detail: detail}
}
