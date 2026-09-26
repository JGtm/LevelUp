package replaybuild

// filmfacts_refus_test.go — AUCUN FAIT N EST ECRIT QUAND L ARTEFACT EST REFUSE (lot J3.4 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, constat RA1-1).
//
// Le scenario de l audit : une cuisson sans faits de base (ouvrier, `replay-build` sans `--facts`,
// base indisponible au backfill) produit un artefact APPAUVRI, que le puits refuse — il
// retrograderait celui en place. Elle ecrivait pourtant ses faits, cuits sous des gardes pauvres,
// et la reparation suivante les rejouait. Les faits d une cuisson ne se rangent donc que si le
// puits RANGERAIT son artefact.

import (
	"context"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// faitsMinimaux : un fichier de faits que l encodeur accepte.
func faitsMinimaux() *replay.FilmFactsFile {
	return &replay.FilmFactsFile{Facts: replay.FilmFacts{Film: "abcd1234"}}
}

// TestCuisson_AucunFaitEcritQuandLArtefactEstRefuse : un artefact en place porte des compteurs de
// joueur, le candidat n en porte pas, au meme schema — le puits le refuserait, donc les faits de
// la cuisson ne s ecrivent pas. Sans artefact en place, ils s ecrivent.
func TestCuisson_AucunFaitEcritQuandLArtefactEstRefuse(t *testing.T) {
	const match = "abcd1234"
	b := &Builder{repoRoot: t.TempDir(), titleSlug: title.DefaultSlug}
	res := title.NewPathResolver(b.repoRoot)
	ctx := context.Background()
	pauvre := artefactAuSchema(replay.SchemaVersion, match, false)

	b.rangerLesFaits(ctx, match, pauvre, faitsMinimaux())
	if _, err := os.Stat(res.FilmFactsPath(b.titleSlug, match)); err != nil {
		t.Fatalf("premiere cuisson (aucun artefact en place) : faits non ecrits : %v", err)
	}
	if err := os.Remove(res.FilmFactsPath(b.titleSlug, match)); err != nil {
		t.Fatal(err)
	}

	riche := artefactAuSchema(replay.SchemaVersion, match, true)
	if _, err := StoreArtifact(b.repoRoot, b.titleSlug, match, riche); err != nil {
		t.Fatalf("depot de l artefact riche : %v", err)
	}
	b.rangerLesFaits(ctx, match, pauvre, faitsMinimaux())
	if _, err := os.Stat(res.FilmFactsPath(b.titleSlug, match)); !os.IsNotExist(err) {
		t.Fatalf("l artefact APPAUVRI est refuse par le puits, et ses faits sont ecrits quand "+
			"meme (err = %v) : la reparation suivante rejouerait des faits cuits sous des gardes "+
			"pauvres (RA1-1)", err)
	}
}

// TestCuisson_LesFaitsSeRangentApresLeVerdictDuPuits : LA FORME QUI TIENT LA REGLE. Les faits ne
// s ecrivent plus dans la production du document (`documentDeLaCuisson`), avant que l artefact
// existe : `BuildBytes` les range APRES la serialisation, par `rangerLesFaits`, qui consulte le
// puits.
func TestCuisson_LesFaitsSeRangentApresLeVerdictDuPuits(t *testing.T) {
	if corps := corpsDeFonction(t, "documentDeLaCuisson"); strings.Contains(corps, "ecrireLesFaits(") {
		t.Error("`documentDeLaCuisson` ecrit les faits AVANT que l artefact existe : un artefact " +
			"refuse par le puits laisserait ses faits sur le disque (RA1-1)")
	}
	corps := corpsDeFonction(t, "BuildBytes")
	iSer, iRang := strings.Index(corps, "serialiserDocument("), strings.Index(corps, "rangerLesFaits(")
	if iSer < 0 || iRang < 0 || iRang < iSer {
		t.Errorf("`BuildBytes` doit ranger les faits (`rangerLesFaits`) APRES `serialiserDocument` "+
			"(positions %d, %d)", iSer, iRang)
	}
}

// faitsFraisSousGardes : un fichier de faits FRAIS pour `entry` (revisions courantes, cle
// auto-detectee sur une entree sans bornes), cuit sous les gardes que `opts` commande, relu avec
// son en-tete.
func faitsFraisSousGardes(t *testing.T, entry decfilm.MapQuantEntry, opts replay.Options,
) (*replay.FilmFactsFile, replay.FilmFactsEntete) {
	t.Helper()
	rev := replay.RevisionsCourantesDesCouches()
	f := &replay.FilmFactsFile{
		Coverage: replay.DecoderCoverage{SourceRev: rev["source"], ProfileRev: rev["profile"],
			GrammarRev: rev["grammar"], KillsourceRev: rev["killsource"], ObjectivesRev: rev["objectives"]},
		Facts:  replay.FilmFacts{Film: "ab526724", MapModule: entry.Module, LayoutDetected: true},
		Gardes: replay.GardesDe(opts),
	}
	blob, err := replay.EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	entete, err := replay.DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatalf("en-tete : %v", err)
	}
	if err := entete.Frais(entry); err != nil {
		t.Fatalf("les faits du test ne sont pas frais — le cas ne mesure rien : %v", err)
	}
	return f, entete
}

// TestCuisson_DesFaitsSousDAutresGardesFontChargerLeFilm : RA1-1, cote bascule. Des faits FRAIS
// cuits sans catalogue de zones ne servent pas une cuisson qui en fournit un : le film est charge
// (ici le temoin partiel d `ab526724`, refuse par sa garde de finalisation — preuve que la branche
// du film a ete prise). Sous les memes gardes, les faits servent et aucun film n est ouvert.
func TestCuisson_DesFaitsSousDAutresGardesFontChargerLeFilm(t *testing.T) {
	entry := decfilm.MapQuantEntry{Module: "carte_du_test"}
	dir, _ := repertoireDuTemoin(t, temoinPartiel(t))
	sansZones := replay.Options{MapQuant: &entry}
	f, entete := faitsFraisSousGardes(t, entry, sansZones)
	src := entreesDeCuisson{faits: f, entete: entete}
	b := constructeurSansFaits(t)
	ctx := context.Background()

	avecZones := sansZones
	avecZones.Zone = replay.ZoneInput{Zones: make([]replay.Zone, 1)}
	_, err := b.documentDeLaCuisson(ctx, "ab526724", dir, avecZones, src)
	verifierRefusEcarte(t, err)

	cuit, err := b.documentDeLaCuisson(ctx, "ab526724", dir, sansZones, src)
	if err != nil || !cuit.depuisLesFaits {
		t.Fatalf("faits sous les MEMES gardes : err = %v, depuisLesFaits = %v — ils doivent "+
			"servir sans ouvrir le film", err, cuit.depuisLesFaits)
	}
}
