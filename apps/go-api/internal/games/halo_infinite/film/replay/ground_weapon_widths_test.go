package replay

// ground_weapon_widths_test.go — LES LARGEURS DU BLOC MPP, ET LE SILENCE QU'ELLES CAUSAIENT.
//
// CE QUE CE FICHIER FERME (correctif de la revue du 2026-08-17). Le mot d'identite de 32 bits
// d'une arme au sol se lit derriere deux champs de largeur VARIABLE, mesures film par film par
// la calibration des poses `ti=37`. Cette calibration RESTAURE les largeurs en sortant : le
// balayage `ti=42` qui la suivait lisait donc l'identite aux largeurs par DEFAUT. Sur un film
// calibre autrement (8/3 sur les films BTB mesures), aucune creation n'aurait resolu d'arme —
// zero socle publie, et pas un mot au journal.
//
// DEUX VERROUS, PARCE QUE LE DEFAUT AVAIT DEUX MOITIES : les largeurs mesurees sont bien
// INSTALLEES puis RESTAUREES autour du balayage, et une identite qui ne resout RIEN se DIT.

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestGwInstallMPPWidthsInstalleEtRestaure : les largeurs mesurees valent pour le balayage, et
// pour lui seul — elles se posent sur le CONTEXTE du film (lot 2.3), et les rendre reste un
// contrat : la cuisson enchaine plusieurs archetypes sur le MEME contexte.
func TestGwInstallMPPWidthsInstalleEtRestaure(t *testing.T) {
	fc := grammar.NewFilmContext(nil)
	avant := fc.ProfilDeBalayage().MPP
	// Le decoupage d'un film BTB mesure (8/3), qui n'est PAS l'invariant du profil (9/5).
	btb := profile.MPPWidths{Lead: 8, Index: 3}
	if btb == avant {
		t.Fatalf("le decoupage temoin %s est deja l'invariant : le test ne verifie plus rien", btb)
	}
	restore := gwInstallMPPWidths(fc, btb)
	if got := fc.ProfilDeBalayage().MPP; got != btb {
		t.Fatalf("largeurs installees %s, attendu %s", got, btb)
	}
	restore()
	if got := fc.ProfilDeBalayage().MPP; got != avant {
		t.Fatalf("largeurs NON restaurees : %s, attendu %s", got, avant)
	}
}

// TestGwInstallMPPWidthsIgnoreUnDecoupageNonMesure : une calibration qui a REFUSE de trancher
// rend un decoupage nul. L'installer lirait zero bit de tete et zero bit d'index — pire que le
// defaut, qui a au moins ete mesure ailleurs.
func TestGwInstallMPPWidthsIgnoreUnDecoupageNonMesure(t *testing.T) {
	fc := grammar.NewFilmContext(nil)
	avant := fc.ProfilDeBalayage().MPP
	restore := gwInstallMPPWidths(fc, profile.MPPWidths{})
	if got := fc.ProfilDeBalayage().MPP; got != avant {
		t.Fatalf("un decoupage non mesure a ete installe : %s", got)
	}
	restore()
	if got := fc.ProfilDeBalayage().MPP; got != avant {
		t.Fatalf("largeurs abimees par une restauration inutile : %s", got)
	}
}

// TestCouvertureAvertitQuandAucuneIdentiteNeResout : une largeur FAUSSE ne rend pas des socles
// approximatifs, elle rend ZERO creation retenue pour des dizaines d'acceptees. C'est la
// signature de la panne, et elle doit s'entendre au journal.
//
// LA LARGEUR NE SE SIMULE PAS SANS FILM : ce qu'un decoupage faux PRODUIT, en revanche, se
// fabrique — des records acceptes dont le mot MPP ne resout aucune arme. C'est cet etat-la que
// le journal doit nommer, quelle qu'en soit la cause.
func TestCouvertureAvertitQuandAucuneIdentiteNeResout(t *testing.T) {
	fausse := WorldObjectScan{
		Scanned: true,
		Stats:   types.EquipmentCreationStats{Slots: 8, Anchors: 400, Accepted: 12},
		Creations: []types.EquipmentCreation{
			gwTestCreation(60, 0, 1_000_000, 0xDEADBEEF, 1, 1),
			gwTestCreation(61, 0, 2_000_000, 0xBADC0FFE, 2, 2),
		},
		Keyframes: grammar.WorldObjectKeyframes{TimesUS: []uint64{0, 20_000_000}},
	}
	_, _, cov, _ := buildWeaponPads(PadScans{Weapons: fausse}, nil, gwTestClock(), padCatalogs{})
	if cov.Kept != 0 || cov.Accepted == 0 {
		t.Fatalf("le cas temoin doit etre `0 retenue pour N acceptees` : %+v", cov)
	}
	if log := gwCaptureLog(t, func() { logGroundWeaponCoverage(cov) }); !strings.Contains(log, "largeurs MPP") {
		t.Fatalf("aucun avertissement sur une identite qui ne resout RIEN — le journal est"+
			" muet la ou un film entier sort sans socle. Journal obtenu :\n%s", log)
	}

	// LE CONTRE-CAS COMPTE AUTANT : une identite qui resout ne doit RIEN dire de special,
	// sans quoi l'avertissement se noierait dans le bruit et personne ne le lirait.
	bonne, pos := gwTestPadScan(t)
	_, _, ok, _ := buildWeaponPads(PadScans{Weapons: bonne}, pos, gwTestClock(), padCatalogs{})
	if ok.Kept == 0 {
		t.Fatalf("le contre-cas doit retenir des creations : %+v", ok)
	}
	if log := gwCaptureLog(t, func() { logGroundWeaponCoverage(ok) }); strings.Contains(log, "largeurs MPP") {
		t.Fatalf("avertissement emis alors que l'identite resout : %s", log)
	}
}

// gwCaptureLog capte le journal par defaut le temps d'un appel, et le rend.
func gwCaptureLog(t *testing.T, run func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)
	run()
	return buf.String()
}
