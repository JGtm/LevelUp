package domain

import "testing"

func TestDeriveCSRTier(t *testing.T) {
	cases := []struct {
		csr     int
		tier    string
		subTier int
	}{
		{0, "Bronze", 1},
		{299, "Bronze", 6},
		{300, "Silver", 1},
		{600, "Gold", 1},
		{900, "Platinum", 1},
		{1200, "Diamond", 1},
		{1499, "Diamond", 6},
		{1500, "Onyx", 0},
		{2180, "Onyx", 0},
	}
	for _, c := range cases {
		tier, sub := DeriveCSRTier(c.csr)
		if tier != c.tier || sub != c.subTier {
			t.Errorf("DeriveCSRTier(%d) = (%q, %d), attendu (%q, %d)",
				c.csr, tier, sub, c.tier, c.subTier)
		}
	}
}
