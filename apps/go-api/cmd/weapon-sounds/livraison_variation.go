package main

// livraison_variation.go — la fourchette RANGED a servir pour une arme livree, et l'ordre de
// traitement des dossiers candidats. Port de livraison.py:variationDe/estVariante et de la
// cle de tri du script principal.

import (
	"sort"
	"strconv"
	"strings"
)

// livraisonParseHex32 lit un identifiant hexadecimal 32 bits (forme "%08x" de Python).
func livraisonParseHex32(hex string) (uint32, error) {
	v, err := strconv.ParseUint(hex, 16, 32)
	return uint32(v), err
}

// livraisonFiltreVariation ne garde que les fourchettes NON DEGENEREES (bas et haut pas tous
// deux nuls) — port de l'intersection Python `v.get(k) and (v[k]["bas"] or v[k]["haut"])`.
// Rend nil si aucune des deux composantes ne passe le filtre (dict vide -> None en Python).
func livraisonFiltreVariation(v *livraisonVariation) *livraisonVariationOut {
	if v == nil {
		return nil
	}
	out := &livraisonVariationOut{}
	if v.VolumeDB != nil && (v.VolumeDB.basF() != 0 || v.VolumeDB.hautF() != 0) {
		out.VolumeDB = v.VolumeDB
	}
	if v.PitchCents != nil && (v.PitchCents.basF() != 0 || v.PitchCents.hautF() != 0) {
		out.PitchCents = v.PitchCents
	}
	if out.VolumeDB == nil && out.PitchCents == nil {
		return nil
	}
	return out
}

// livraisonVariationDe rend la fourchette RANGED a servir pour ce dossier : d'abord la
// couche dominante (plus fort gain de chemin) de l'evenement rendu si `evHex` est non vide,
// sinon la variation globale de lot2 — port de livraison.py:variationDe.
func livraisonVariationDe(dossier, evHex string, lot1 map[string]livraisonLot1Arme, lot2 map[string]livraisonLot2Arme) *livraisonVariationOut {
	if evHex != "" {
		if out := livraisonVariationDeEvenement(dossier, evHex, lot1); out != nil {
			return out
		}
	}
	if a2, ok := lot2[dossier]; ok {
		return livraisonFiltreVariation(a2.Variation)
	}
	return nil
}

// livraisonVariationDeEvenement cherche l'evenement `evHex` dans lot1 et rend la variation de
// sa couche au plus fort gain_db_couche (ties : la premiere rencontree gagne, comme
// `max()` en Python).
func livraisonVariationDeEvenement(dossier, evHex string, lot1 map[string]livraisonLot1Arme) *livraisonVariationOut {
	a1, ok := lot1[dossier]
	if !ok {
		return nil
	}
	idEvent, err := livraisonParseHex32(evHex)
	if err != nil {
		return nil
	}
	var ev *livraisonLot1Event
	for i := range a1.Evenements {
		if a1.Evenements[i].IDEvent == idEvent {
			ev = &a1.Evenements[i]
			break
		}
	}
	if ev == nil {
		return nil
	}
	var meilleure *livraisonVariation
	var meilleurGain float64
	for _, c := range ev.Couches {
		if c.Variation == nil {
			continue
		}
		if meilleure == nil || c.Variation.GainDBCouche > meilleurGain {
			meilleure, meilleurGain = c.Variation, c.Variation.GainDBCouche
		}
	}
	return livraisonFiltreVariation(meilleure)
}

// livraisonEstVariante dit si le nom FR du dossier porte une parenthese ("(infectee)",
// "(legendaire)"...) — port de livraison.py:estVariante, qui sert a traiter les variantes
// APRES l'arme de base dans l'ordre de livraison.
func livraisonEstVariante(dossier string, manifeste map[string]livraisonManifesteArme) bool {
	a, ok := manifeste[dossier]
	if !ok || a.NomFr == nil {
		return false
	}
	return strings.Contains(*a.NomFr, "(")
}

// livraisonCandidats rend les dossiers eligibles a la livraison : une cle canonique non vide,
// et au moins un vote retenu OU un role confirme — port du filtre `candidats` de
// livraison.py. Ordre d'entree = ordre d'apparition dans manifeste.json (non signifiant, la
// cle de tri qui suit est un ordre total).
func livraisonCandidats(d *livraisonDonnees) []string {
	var out []string
	for _, dossier := range d.OrdreManifeste {
		a := d.Manifeste[dossier]
		if a.Cle == nil || *a.Cle == "" {
			continue
		}
		_, role := livraisonRoles[dossier]
		if !role && len(livraisonVotesDe(dossier, d.Votes)) == 0 {
			continue
		}
		out = append(out, dossier)
	}
	return out
}

// livraisonTriKey rend la cle de tri (role confirme d'abord, puis nom de base avant variante,
// puis alphabetique) — port de `key=lambda d: (d not in ROLES, estVariante(d), d)`. Le
// troisieme element (le nom) est TOUJOURS unique : aucune egalite ne survit, donc l'ordre
// rendu est deterministe quel que soit l'ordre d'entree.
func livraisonTriKey(dossier string, manifeste map[string]livraisonManifesteArme) (int, int, string) {
	notInRoles := 1
	if _, ok := livraisonRoles[dossier]; ok {
		notInRoles = 0
	}
	variante := 0
	if livraisonEstVariante(dossier, manifeste) {
		variante = 1
	}
	return notInRoles, variante, dossier
}

// livraisonOrdre trie les candidats selon livraisonTriKey.
func livraisonOrdre(candidats []string, manifeste map[string]livraisonManifesteArme) []string {
	ordre := append([]string(nil), candidats...)
	moinsQue := func(i, j int) bool {
		a1, a2, a3 := livraisonTriKey(ordre[i], manifeste)
		b1, b2, b3 := livraisonTriKey(ordre[j], manifeste)
		if a1 != b1 {
			return a1 < b1
		}
		if a2 != b2 {
			return a2 < b2
		}
		return a3 < b3
	}
	sort.Slice(ordre, moinsQue)
	return ordre
}
