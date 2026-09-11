package title

import (
	"path/filepath"
	"testing"
)

// TestMapGeometryDirEstParCarte — LE REPERTOIRE DES PROPS PORTE LE MODULE, ET C'EST UN
// CORRECTIF.
//
// `MapGeometryDir` ne prenait que le titre et rendait UN repertoire : le CSV qui s'y trouvait
// etait servi comme decor de CHAQUE match, quelle que soit la carte. Mesure du 2026-09-11 sur
// les 76 artefacts du parc : 382 props IDENTIQUES partout, cartes confondues.
//
// LA CLE EST LE MODULE — la meme que `map_quant_bounds.json`, `MapStructurePath` et
// `MapBackgroundPath`. Le lien nom affiche -> module reste declare a un seul endroit.
func TestMapGeometryDirEstParCarte(t *testing.T) {
	pr := NewPathResolver("/repo")
	racine := filepath.Join("/repo", "data", "titles", "halo_infinite", "reference", "map_geometry")

	if got, want := pr.MapGeometryDir("halo_infinite", "ridgeline"),
		filepath.Join(racine, "ridgeline"); got != want {
		t.Errorf("MapGeometryDir(halo_infinite, ridgeline) = %q, attendu %q", got, want)
	}
	// DEUX CARTES NE PARTAGENT PLUS LEUR DECOR : c'est tout l'objet du lot.
	if a, b := pr.MapGeometryDir("halo_infinite", "ridgeline"),
		pr.MapGeometryDir("halo_infinite", "sgh_interlock"); a == b {
		t.Errorf("deux cartes rendent le meme repertoire de props : %q", a)
	}
	// MODULE VIDE = LE REPERTOIRE DU TITRE, celui du CATALOGUE DES TYPES. Il vaut pour toutes
	// les cartes ; le recopier sous chacune en ferait diverger les copies.
	if got, want := pr.MapGeometryDir("halo_infinite", ""), racine; got != want {
		t.Errorf("MapGeometryDir(halo_infinite, \"\") = %q, attendu %q", got, want)
	}
	// L'ISOLATION PAR TITRE tient, comme pour tous les chemins (ADR 0008).
	if a, b := pr.MapGeometryDir("halo_infinite", "ridgeline"),
		pr.MapGeometryDir("halo_5", "ridgeline"); a == b {
		t.Errorf("deux titres rendent le meme repertoire de props : %q", a)
	}
}
