package main

// livraison_ntpath_test.go — joliBaseSansExt EST un port de ntpath, ou n'en est pas un.
//
// La cle de dossier d'une arme sort de `os.path.splitext(os.path.basename(pck))[0]` execute
// SOUS WINDOWS (module `ntpath`) : c'est le seul OS sur lequel `_outils/livraison.py` a
// tourne, et les chemins de lot1.json/lot2.json sont toujours des chemins Windows, quelle que
// soit la plateforme qui execute ce binaire. Une cle fausse PERD L'ARME : elle n'est plus
// retrouvee dans lot1/lot2 (sa fourchette RANGED disparait du catalogue) ou elle est livree
// sous une autre cle.
//
// La premiere version promettait `ntpath` en commentaire sans porter ni le prefixe de lecteur
// ni la regle du point de tete (constat C8 de la revue R1). La table ci-dessous est la SORTIE
// REELLE de CPython 3.12 sur chacune de ces entrees, relevee une fois par une sonde hors
// depot — pas une reconstitution de memoire.

import "testing"

func TestJoliBaseSansExt_FideleANtpath(t *testing.T) {
	cas := []struct{ entree, veut string }{
		// LES DEUX FORMES QUE LA PREMIERE VERSION RATAIT.
		{`C:sb_010_wea_un_relatifdrive.pck`, "sb_010_wea_un_relatifdrive"}, // relatif au lecteur
		{`C:sb_010_wea_un_relatifdrive`, "sb_010_wea_un_relatifdrive"},
		{`c:sb_010_wea_cv_needler.pck`, "sb_010_wea_cv_needler"},
		{`C:x`, "x"},
		{".pck", ".pck"},   // point de tete : pas une extension
		{"..pck", "..pck"}, // deux points de tete non plus
		{".a.pck", ".a"},   // le second point, lui, en ouvre une
		{`C:\dossier\.cache.pck`, ".cache"},

		// LES FORMES REELLES DU CHANTIER, deja correctes avant.
		{`C:\Steam\SFX\sb_010_wea_un_assaultrifle.pck`, "sb_010_wea_un_assaultrifle"},
		{"C:/Steam/SFX/sb_010_wea_un_assaultrifle.pck", "sb_010_wea_un_assaultrifle"},
		{`C:\Steam/SFX\sb_010_tur_bt_gatlingmortar.pck`, "sb_010_tur_bt_gatlingmortar"},
		{`C:\dossier.avec.points\sb_010_wea_cv_needler.pck`, "sb_010_wea_cv_needler"},
		{"relatif/sous/dossier/sb_010_wea_bt_mangler.pck", "sb_010_wea_bt_mangler"},
		{"sb_010_whizby_pl_generic.pck", "sb_010_whizby_pl_generic"},
		{"x.pck", "x"},
		{"x.", "x"},
		{"x", "x"},
		{"", ""},

		// RACINES ET CHEMINS SANS NOM DE FICHIER.
		{`D:`, ""},
		{`D:\`, ""},
		{`C:\Steam\SFX\`, ""},
		{`\x`, "x"},

		// UNC, PERIPHERIQUE, PREFIXE ETENDU : le « lecteur » va jusqu'au deuxieme separateur
		// qui suit, et ce qui reste ensuite seulement est un nom de fichier.
		{`\\serveur\partage\sb_010_wea_un_unc.pck`, "sb_010_wea_un_unc"},
		{`\\serveur\partage`, ""},
		{`\\serveur`, ""},
		{`\\`, ""},
		{`\\\x`, ""},
		{"//serveur/partage/sb_010_wea_un_unc2.pck", "sb_010_wea_un_unc2"},
		{`\\?\UNC\s\p\x.pck`, "x"},
		// Le prefixe etendu est reconnu SANS LA CASSE (`normp[:8].upper()` chez ntpath) :
		// sans cela, le nom de partage passait pour un nom de fichier.
		{`\\?\unc\s\p`, ""},
		{`\\?\unc\serveur\partage.pck`, ""},
		{`\\.\device\x.pck`, "x"},
	}
	for _, c := range cas {
		if got := joliBaseSansExt(c.entree); got != c.veut {
			t.Errorf("joliBaseSansExt(%q) = %q, veut %q (sortie de ntpath sous CPython 3.12)",
				c.entree, got, c.veut)
		}
	}
}

// TestJoliDossier_FormesLimites : la consequence produit du constat C8 — l'arme est
// retrouvee, ou elle est perdue.
func TestJoliDossier_FormesLimites(t *testing.T) {
	cas := map[string]string{
		`C:sb_010_wea_un_relatifdrive.pck`: "UNSC_relatifdrive",
		".pck":                             ".pck", // pas de match : le prefixe sb_010_ est retire tel quel
	}
	for pck, veut := range cas {
		if got := joliDossier(pck); got != veut {
			t.Errorf("joliDossier(%q) = %q, veut %q", pck, got, veut)
		}
	}
}
