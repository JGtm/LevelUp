package domain

// csr_tier.go — dérivation (tier, sous-palier) depuis un CSR numérique.
//
// Halo Waypoint ne publie que le CSR numérique sur ses leaderboards ; le palier
// se déduit selon la spec Halo Infinite. Source unique réutilisée par le scraper
// (écriture du snapshot) et le repo (lecture / badge de rang).

// Bandes CSR officielles Halo Infinite : 5 paliers de 300 (6 sous-paliers de 50),
// Onyx = 1500+ (numérique, sans sous-palier).
const (
	csrBandWidth    = 300
	csrSubBandWidth = 50
	csrOnyxFloor    = 1500
)

var csrTierNames = []string{"Bronze", "Silver", "Gold", "Platinum", "Diamond"}

// DeriveCSRTier convertit un CSR numérique en (tier, sous-palier 1..6).
// Onyx (>=1500) : tier "Onyx", subTier 0 (affichage numérique).
func DeriveCSRTier(csr int) (string, int) {
	if csr >= csrOnyxFloor {
		return "Onyx", 0
	}
	if csr < 0 {
		csr = 0
	}
	bandIdx := csr / csrBandWidth // 0..4
	if bandIdx >= len(csrTierNames) {
		bandIdx = len(csrTierNames) - 1
	}
	sub := (csr%csrBandWidth)/csrSubBandWidth + 1 // 1..6
	return csrTierNames[bandIdx], sub
}
