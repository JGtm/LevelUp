package canonical

import "testing"

// TestLinkSourcesSontFermees : la liste des provenances est FERMEE et exhaustive. Une valeur
// ajoutee a l'enum sans entrer dans AllLinkSources rendrait IsKnownLinkSource faux pour elle,
// et le producteur publierait une categorie que le gate corpus ne saurait pas compter.
func TestLinkSourcesSontFermees(t *testing.T) {
	all := AllLinkSources()
	if len(all) != 5 {
		t.Fatalf("AllLinkSources = %d valeurs, attendu 5", len(all))
	}
	for _, s := range all {
		if !IsKnownLinkSource(s) {
			t.Errorf("IsKnownLinkSource(%q) = false", s)
		}
	}
	if IsKnownLinkSource("geometrie") {
		t.Error("une provenance hors enum est acceptee — l'enum n'est plus fermee")
	}
}

// TestEntityKindsSontFermes : meme regle pour les types d'entite.
func TestEntityKindsSontFermes(t *testing.T) {
	all := AllEntityKinds()
	if len(all) != 9 {
		t.Fatalf("AllEntityKinds = %d valeurs, attendu 9", len(all))
	}
	vus := map[EntityKind]bool{}
	for _, k := range all {
		if vus[k] {
			t.Errorf("doublon dans AllEntityKinds : %q", k)
		}
		vus[k] = true
		if !IsKnownEntityKind(k) {
			t.Errorf("IsKnownEntityKind(%q) = false", k)
		}
	}
	if IsKnownEntityKind("grenade") {
		t.Error("un type d'entite hors enum est accepte")
	}
}

// TestLinkCountsCompteParProvenance : le recapitulatif compte ce qu'on lui donne, et une
// provenance inconnue n'entre nulle part — c'est ce qui rend l'invariant `Total == nb liens`
// capable de reveler un producteur fautif.
func TestLinkCountsCompteParProvenance(t *testing.T) {
	var c LinkCounts
	for _, s := range []LinkSource{LinkDirect, LinkDirect, LinkInferred, LinkUnresolved,
		LinkExternal, LinkCatalog} {
		c.Add(s)
	}
	if c.Direct != 2 || c.Inferred != 1 || c.Unresolved != 1 || c.External != 1 || c.Catalog != 1 {
		t.Fatalf("comptes inattendus : %+v", c)
	}
	if c.Total() != 6 {
		t.Fatalf("Total = %d, attendu 6", c.Total())
	}
	c.Add("inventee")
	if c.Total() != 6 {
		t.Fatalf("une provenance inconnue a ete comptee : Total = %d", c.Total())
	}
}
