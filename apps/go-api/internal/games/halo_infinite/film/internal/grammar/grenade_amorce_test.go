package grammar

// grenade_amorce_test.go — LES DEUX LARGEURS D AMORCE, PROUVEES SUR UNE FIXTURE (lot 3.3.1).
//
// CE QUE CE FICHIER TIENT, ET POURQUOI IL NE PEUT PAS ETRE UNE MINI-BOBINE. Les sept
// mini-bobines versionnees ne portent AUCUN paquet delta (decouverte D1 (3.3r), lue dans leur
// `PROVENANCE.txt`) — or le motif d amorce ne vit que la. Le banc est donc un flux CONSTRUIT,
// ecrit par la meme table de profil que la production lit, et il repond a la seule question
// qu un flux construit puisse trancher : la grammaire est-elle celle qu on a ecrite, et les deux
// largeurs sont-elles VRAIMENT distinctes.
//
// LE TEMOIN NEGATIF EST LE COEUR DU FICHIER. Avant le lot 3.3.1 la production comparait 24 bits
// sur TOUS les films : sur un build ancien, son vingt-quatrieme bit n est pas de l amorce, c est
// le bit de poids fort de l identifiant. Elle ne pouvait donc reconnaitre un lancer que si ce bit
// valait zero — un seul des quatre identifiants — et lisait alors `identifiant << 1`, jamais dans
// la liste blanche. D ou le ZERO ABSOLU mesure sur les cinq temoins anciens du corpus. Les cas
// `refusePar...` ci-dessous reproduisent ce zero, et rougiront si quelqu un remet une largeur
// unique.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// grammaireSous compose la grammaire de balayage d une amorce donnee, sur l archetype de
// projectile de reference.
func grammaireSous(a profile.AmorceGrenade) grenadeGrammaire {
	return grenadeGrammaire{amorce: a, ti: ProjectileTypeIndex, motif: a.MarqueurDe(ProjectileTypeIndex)}
}

// amorcePour rend l amorce d une cle, ou echoue : une cle de la table qui disparait doit faire
// rougir ce banc, pas le faire passer sur un profil de reference silencieux.
func amorcePour(t *testing.T, build string, majeure int) profile.AmorceGrenade {
	t.Helper()
	a, connue := profile.AmorceGrenadePour(build, majeure)
	if !connue {
		t.Fatalf("cle (build=%q, majeure=%d) absente de la table d amorce", build, majeure)
	}
	return a
}

// TestAmorceGrenadeLesDeuxLargeursSeLisent : chaque grammaire lit SON record, aux quatre rangs.
func TestAmorceGrenadeLesDeuxLargeursSeLisent(t *testing.T) {
	cas := []struct {
		nom     string
		build   string
		majeure int
		bits    int
		index   int
	}{
		{nom: "recent", build: "HI_1_13_0", bits: 24, index: 103},
		{nom: "ancien", build: "HI_1_10_0", bits: 23, index: 100},
		{nom: "majeure33", build: "HI_1_4_1", bits: 23, index: 99},
		{nom: "majeure31", majeure: 31, bits: 23, index: 99},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			a := amorcePour(t, c.build, c.majeure)
			if a.Bits != c.bits || a.IndexAuteurBit != c.index {
				t.Fatalf("amorce %d bits / index +%d, attendu %d / +%d — la table a bouge sans "+
					"que la mesure du lot 3.3.1 suive", a.Bits, a.IndexAuteurBit, c.bits, c.index)
			}
			g := grammaireSous(a)
			for rang, id := range GrenadeTypeIDsByRank {
				var cov grenadeCouverture
				got := scanGrenadeThrows(buildGrenadeRecordSous(a, 7, id, 19), g, &cov)
				if len(got) != 1 {
					t.Fatalf("rang %d (%08X) : %d lancer(s) lu(s), attendu 1", rang, id, len(got))
				}
				if got[0].FilmIndex != 19 {
					t.Errorf("rang %d : index de lanceur %d, attendu 19 — un film BTB indexe "+
						"jusqu a 23, le champ de cinq bits les porte", rang, got[0].FilmIndex)
				}
				if r, connu := got[0].Rank(); !connu || r != rang {
					t.Errorf("identifiant %08X : rang %d (connu=%v), attendu %d", id, r, connu, rang)
				}
				if got[0].BitPos != 8 {
					t.Errorf("motif rapporte au bit %d, attendu 8", got[0].BitPos)
				}
			}
		})
	}
}

// TestAmorceGrenadeUneLargeurNeLitPasLAutre : LE TEMOIN NEGATIF, c est-a-dire le zero historique.
func TestAmorceGrenadeUneLargeurNeLitPasLAutre(t *testing.T) {
	recente := amorcePour(t, "HI_1_13_0", 0)
	ancienne := amorcePour(t, "HI_1_10_0", 0)
	maj31 := amorcePour(t, "", 31)
	croix := []struct {
		nom          string
		ecrite, lue  profile.AmorceGrenade
		pourquoiZero string
	}{
		{nom: "ancien_lu_par_recent", ecrite: ancienne, lue: recente,
			pourquoiZero: "la production d avant le lot 3.3.1 comparait 24 bits partout : c est " +
				"exactement ce zero que les cinq temoins anciens du corpus publiaient"},
		{nom: "recent_lu_par_ancien", ecrite: recente, lue: ancienne,
			pourquoiZero: "la symetrie doit tenir aussi : une grammaire ancienne appliquee a un " +
				"film recent ne doit rien inventer"},
		{nom: "majeure31_lue_par_ancien", ecrite: maj31, lue: ancienne,
			pourquoiZero: "les deux amorces font 23 bits et n ont PAS la meme valeur (0x20400 " +
				"contre 0x20600) : la largeur seule ne suffit pas a decrire la grammaire"},
		{nom: "ancien_lu_par_majeure31", ecrite: ancienne, lue: maj31,
			pourquoiZero: "meme raison, dans l autre sens"},
	}
	for _, c := range croix {
		t.Run(c.nom, func(t *testing.T) {
			g := grammaireSous(c.lue)
			for _, id := range GrenadeTypeIDsByRank {
				var cov grenadeCouverture
				got := scanGrenadeThrows(buildGrenadeRecordSous(c.ecrite, 7, id, 3), g, &cov)
				if len(got) != 0 {
					t.Errorf("identifiant %08X : %d lancer(s) lu(s) sous la mauvaise grammaire, "+
						"attendu 0 — %s", id, len(got), c.pourquoiZero)
				}
			}
		})
	}
}

// TestAmorceGrenadeEcarteLesNaissancesDeManagedPlayer : LE SIXIEME BIT D INDEX EST LU.
//
// Le motif ne porte que CINQ bits du typeIndex : `ti=41` (projectile) et `ti=9`
// (`managed-player`) produisent le MEME (decouverte D2 (3.3r)). Sans la lecture du sixieme bit,
// une naissance de `managed-player` suivie par hasard d un identifiant de la liste blanche
// passerait pour un lancer. Le banc construit exactement cette naissance-la.
func TestAmorceGrenadeEcarteLesNaissancesDeManagedPlayer(t *testing.T) {
	a := amorcePour(t, "HI_1_13_0", 0)
	g := grammaireSous(a)
	w := &bitw{}
	w.pad(7)
	w.put(0, 1) // sixieme bit du typeIndex a ZERO : ti = 9, pas 41
	w.put(a.MarqueurDe(ProjectileTypeIndex), a.Bits)
	w.put(uint64(GrenadeFragmentation), 32)
	w.pad(a.IndexAuteurBit - a.Bits - 32)
	w.put(3, profile.AmorceGrenadeIndexBits)
	w.pad(32)
	var cov grenadeCouverture
	got := scanGrenadeThrows(w.buf, g, &cov)
	if len(got) != 0 {
		t.Fatalf("%d lancer(s) publie(s) sur une naissance de managed-player : le sixieme bit "+
			"d index n est plus lu, et le balayage ramasse deux archetypes", len(got))
	}
	if cov.naissancesAutresArchetypes != 1 {
		t.Errorf("naissancesAutresArchetypes=%d, attendu 1 — un rejet non compte est un rejet "+
			"invisible", cov.naissancesAutresArchetypes)
	}
	if cov.motifs != 1 {
		t.Errorf("motifs=%d, attendu 1", cov.motifs)
	}
}

// TestAmorceGrenadeTableCompleteEtMesuree : les NEUF clefs du depot ont leur ligne, et les
// valeurs sont celles que le lot 3.3.1 a mesurees.
//
// LES NEUF CLEFS SONT CELLES DE LA TABLE DES EMPREINTES DE REGISTRE (lot 3.2.1) : sept builds et
// deux versions majeures sans section d identification. Une clef qui perdrait sa ligne tomberait
// sur le profil de REFERENCE — donc sur la grammaire a 24 bits — et re-publierait le zero que ce
// lot vient de fermer.
func TestAmorceGrenadeTableCompleteEtMesuree(t *testing.T) {
	attendu := []struct {
		build          string
		majeure        int
		bits           int
		etat           uint32
		indexAuteurBit int
	}{
		{build: "HI_1_13_0", bits: 24, etat: 0x40C00, indexAuteurBit: 103},
		{build: "HI_1_12_0", bits: 24, etat: 0x40C00, indexAuteurBit: 103},
		{build: "HI_1_11_0", bits: 23, etat: 0x20600, indexAuteurBit: 100},
		{build: "HI_1_10_0", bits: 23, etat: 0x20600, indexAuteurBit: 100},
		{build: "HI_1_9_0", bits: 23, etat: 0x20600, indexAuteurBit: 100},
		{build: "HI_1_8_0", bits: 23, etat: 0x20600, indexAuteurBit: 100},
		{build: "HI_1_4_1", bits: 23, etat: 0x20600, indexAuteurBit: 99},
		{majeure: 33, bits: 23, etat: 0x20600, indexAuteurBit: 99},
		{majeure: 31, bits: 23, etat: 0x20400, indexAuteurBit: 99},
	}
	for _, c := range attendu {
		a := amorcePour(t, c.build, c.majeure)
		if a.Bits != c.bits || a.EtatParDefaut != c.etat || a.IndexAuteurBit != c.indexAuteurBit {
			t.Errorf("(build=%q, majeure=%d) : %d bits / 0x%05X / +%d, attendu %d / 0x%05X / +%d",
				c.build, c.majeure, a.Bits, a.EtatParDefaut, a.IndexAuteurBit,
				c.bits, c.etat, c.indexAuteurBit)
		}
	}
	// LE MOTIF DE PRODUCTION HISTORIQUE EST CE QUE LA REGLE COMPOSE, et c est l invariant sur
	// lequel toute la lecture « le motif est une donnee de build » repose. S il tombe, la
	// derivation est fausse et rien de ce qui precede ne vaut.
	if m := amorcePour(t, "HI_1_13_0", 0).MarqueurDe(ProjectileTypeIndex); m != 0x4C0C00 {
		t.Errorf("motif derive 0x%06X, attendu 0x4C0C00 — la derivation ((ti&31)<<19)|0x40C00 "+
			"ne rend plus la constante historique", m)
	}
	if m := amorcePour(t, "HI_1_10_0", 0).MarqueurDe(ProjectileTypeIndex); m != 0x260600 {
		t.Errorf("motif ancien derive 0x%06X, attendu 0x260600", m)
	}
	if m := amorcePour(t, "", 31).MarqueurDe(ProjectileTypeIndex); m != 0x260400 {
		t.Errorf("motif de la majeure 31 derive 0x%06X, attendu 0x260400", m)
	}
}

// TestAmorceGrenadeCleInconnueTombeSurLaReference : une clef absente ne met AUCUN film de cote.
func TestAmorceGrenadeCleInconnueTombeSurLaReference(t *testing.T) {
	if _, connue := profile.AmorceGrenadePour("HI_9_9_9", 0); connue {
		t.Fatal("un build inexistant est declare connu : la table repond a cote")
	}
	ref := profile.AmorceGrenadeDeReference()
	if ref.Connue {
		t.Error("le profil de reference se declare CONNU : le repli deviendrait invisible")
	}
	if ref.Bits != 24 || ref.IndexAuteurBit != 103 {
		t.Errorf("profil de reference %d bits / +%d, attendu 24 / +103", ref.Bits, ref.IndexAuteurBit)
	}
}
