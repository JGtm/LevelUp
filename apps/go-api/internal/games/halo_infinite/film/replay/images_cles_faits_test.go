package replay

// images_cles_faits_test.go — LA LECTURE DES IMAGES-CLES SURVIT A LA PUBLICATION REJOUEE DEPUIS LES
// FAITS (D1.3.2 du plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, ADR 0034 D-7).
//
// La grammaire lit l etat complet des bipedes aux images-cles et publie les records qu elle admet ;
// les fenetres de bits rendent les autres derriere elle, marques recuperes et comptes sous trois
// replis. Une republication depuis les faits ne rebalaie pas le film : elle doit publier les MEMES
// armes, le MEME inventaire et les MEMES comptes de replis que la cuisson qui a ecrit ces faits.
//
// LA BOBINE : `minifilm_11de8353` (registre et seize images-cles reelles du film), en contexte de
// cuisson (profil, carte, decoupage), lue par les PHASES DE PRODUCTION du balayage
// ([filmScan.lireLesImagesCles], [filmScan.balayerInventaire]) ; les autres entrees (positions,
// tirs, morts) sont celles du fixture du MEME film, pour que les fiches aient des trajectoires ou se
// poser. La capture des faits est celle de la cuisson ([versementDuBalayage], [faitsDuBalayage]).
//
// LA MARQUE PAR RECORD NE VOYAGE PAS, ET CE TEST LE TIENT : `grammar.RecordRecupere` vit dans le
// resultat de la grammaire (decision E-8 du plan, forme des faits inchangee) ; ce qui en est PUBLIE
// — les comptes des replis et les valeurs des records recuperes — est ce qui doit survivre, et
// survit. Mutation jouee le 2026-10-09, rouge puis retiree : les trois lignes de versement des
// replis retirees de `versement_des_replis.go`.

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// nomsDesReplisDesImagesCles : les trois replis des fenetres derriere la lecture.
var nomsDesReplisDesImagesCles = []fallback.Nom{fallback.NomFenetreArmesImageCle,
	fallback.NomFenetreInventaireImageCle, fallback.NomFenetreMarqueDePortage}

// TestLesImagesClesSurviventALaPublicationDepuisLesFaits : le document rejoue depuis les faits est,
// a l octet, celui de la cuisson ; il porte les comptes des trois replis notes par la grammaire et
// les valeurs des records recuperes.
func TestLesImagesClesSurviventALaPublicationDepuisLesFaits(t *testing.T) {
	b := goldenBuild{"11de8353", "Thunderhead", "HI_1_9_0"}
	entry, err := b.mapQuant()
	if err != nil {
		t.Fatalf("carte : %v", err)
	}
	relu, _ := chargerGoldenBuild(t, b)
	film, err := source.LoadDir(filepath.Join(goldenDir, "minifilm_"+b.Short8), nil)
	if err != nil {
		t.Fatalf("bobine : %v", err)
	}
	ctx := context.Background()
	opt := Options{MapQuant: &entry, Fallbacks: fallback.NouveauCompteur()}
	fc := grammar.NewFilmContextForMap(film, opt.MapQuant, nil)
	poserProfilPuisCarte(ctx, fc, b.Short8, opt)
	s := &filmScan{ctx: ctx, matchID: b.Short8, film: film, fc: fc, opt: opt, in: relu.FilmInputs}
	s.lireLesImagesCles()
	if s.errEtats != nil {
		t.Fatalf("images-cles de la bobine : %v", s.errEtats)
	}
	s.in.Loadouts, s.in.KeyframeWalk = s.etats.Loadouts, s.etats.Marche
	s.balayerInventaire()
	versementDuBalayage(opt.Fallbacks, fc)
	faits := faitsDuBalayage(b.Short8, fc, opt, s.in)
	if faits == nil {
		t.Fatal("aucun fait capture : decoupage illisible")
	}
	cuisson := s.assembler("halo_infinite", opt)

	blob, err := EncodeFilmFactsFile(faits)
	if err != nil {
		t.Fatalf("encodage des faits : %v", err)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture des faits : %v", err)
	}
	rejeu := BuildFromFacts(ctx, b.Short8, "halo_infinite", f, Options{MapQuant: &entry})

	verifierLesReplisDesImagesCles(t, s.etats.Admission, cuisson, rejeu)
	verifierLesRecuperesPublies(t, s.etats.Recuperes, premierPaquetUS(s.in.Positions), cuisson, rejeu)
	a, errA := json.Marshal(cuisson)
	r, errR := json.Marshal(rejeu)
	if errA != nil || errR != nil {
		t.Fatalf("serialisation : %v / %v", errA, errR)
	}
	if string(a) != string(r) {
		t.Error("le document rejoue depuis les faits differe de celui de la cuisson : les faits perdent une " +
			"lecture des images-cles ou un compte de leurs replis")
	}
}

// verifierLesReplisDesImagesCles : les trois comptes, non nuls, egaux aux records non admis donnes a
// chaque fenetre, sont publies a l identique par les deux chemins.
func verifierLesReplisDesImagesCles(t *testing.T, a grammar.ComptesDeLAdmission, cuisson, rejeu ReplayDocument) {
	t.Helper()
	attendus := map[fallback.Nom]int{fallback.NomFenetreArmesImageCle: a.FenetresArmes,
		fallback.NomFenetreInventaireImageCle: a.FenetresInventaire, fallback.NomFenetreMarqueDePortage: a.FenetresMarque}
	c, r := comptesPublies(cuisson), comptesPublies(rejeu)
	for _, nom := range nomsDesReplisDesImagesCles {
		if attendus[nom] == 0 {
			t.Fatalf("%s : aucun record donne a la fenetre sur la bobine, le test ne mord plus", nom)
		}
		if c[nom] != attendus[nom] || r[nom] != attendus[nom] {
			t.Errorf("%s : cuisson %d, rejeu depuis les faits %d, records non admis donnes %d", nom, c[nom],
				r[nom], attendus[nom])
		}
	}
}

// comptesPublies rend les declenchements publies dans `coverage.fallbacks`, par nom.
func comptesPublies(doc ReplayDocument) map[fallback.Nom]int {
	out := map[fallback.Nom]int{}
	if doc.Coverage == nil {
		return out
	}
	for _, d := range doc.Coverage.Fallbacks {
		out[fallback.Nom(d.Name)] += d.Hits
	}
	return out
}

// verifierLesRecuperesPublies : des records recuperes par la fenetre d inventaire sont publies, et le
// rejeu depuis les faits publie pour chacun la meme lecture que la cuisson.
func verifierLesRecuperesPublies(t *testing.T, recs []grammar.RecordRecupere, origine uint64, cuisson, rejeu ReplayDocument) {
	t.Helper()
	publies := map[[2]int]Inventory{}
	for _, inv := range cuisson.Inventory {
		publies[[2]int{inv.T, int(inv.Slot)}] = inv
	}
	rejoues := map[[2]int]string{}
	for _, inv := range rejeu.Inventory {
		j, _ := json.Marshal(inv)
		rejoues[[2]int{inv.T, int(inv.Slot)}] = string(j)
	}
	mordants := 0
	for _, k := range recs {
		if !k.Inventaire {
			continue
		}
		cle, ok := cleDePublication(cuisson, origine, k)
		if !ok {
			continue
		}
		inv, publie := publies[cle]
		if !publie {
			continue
		}
		mordants++
		j, _ := json.Marshal(inv)
		if rejoues[cle] != string(j) {
			t.Errorf("record recupere (image-cle %d, slot %d) : cuisson %s, rejeu %s", k.TimestampUS, k.Slot, j,
				rejoues[cle])
		}
	}
	if mordants == 0 {
		t.Fatal("aucun record recupere publie dans l inventaire de la bobine : le test ne mord plus")
	}
}

// cleDePublication rend la frame et le slot ou un record d image-cle est publie dans `doc`, dont
// l axe de frames part de `origine` (le premier paquet de position).
func cleDePublication(doc ReplayDocument, origine uint64, k grammar.RecordRecupere) ([2]int, bool) {
	if doc.FrameIntervalMS <= 0 || k.TimestampUS < origine {
		return [2]int{}, false
	}
	pas := uint64(doc.FrameIntervalMS) * 1000
	return [2]int{int((k.TimestampUS - origine) / pas), int(k.Slot)}, true
}
