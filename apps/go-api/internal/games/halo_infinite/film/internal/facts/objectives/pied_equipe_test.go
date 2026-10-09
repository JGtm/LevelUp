package objectives

// pied_equipe_test.go — L'ÉQUIPE QU'`Extract` PUBLIE EST CELLE QUE LE PIED ÉCRIT, ET LE ROSTER N'EN
// EST QUE LE CONTRÔLE (lot 1.7.3).
//
// La LECTURE du bloc — slot à l'octet 36, équipe à l'octet 37, instant aux octets 48 à 51 — est
// tenue par la grammaire, sur un bloc réel du pied de `53ce4390` dont la provenance est écrite à
// côté du binaire (`grammar/signaux/pied_de_film_test.go`, `grammar/signaux/testdata/`). Ce fichier tient ce que le
// lecteur d'objectifs EN FAIT, sur la même fixture : l'équipe publiée, et le contrôle par la
// feuille de match qui compte sans corriger.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/finalise"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/signaux"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Le bloc de référence, versionné par la grammaire, et ce que sa lecture rend.
const (
	piedFixture = "../../grammar/signaux/testdata/pied_bloc_53ce4390.bin"
	piedFilm    = "53ce4390"
	// piedChunk : le NUMÉRO du chunk de pied au manifeste du film (`chunk_40.bin`, type 3).
	piedChunk       = 40
	piedAttenduTime = 133033
	piedAttenduTeam = 1
	piedAttenduXUID = uint64(2535430195856593)
)

// TestExtractPrendLEquipeDuPied : le chemin qui CONSOMME les événements du pied voit le bloc par
// son point d'entrée public. `Extract` PREND L'ÉQUIPE DU PIED : le roster passé ici est VIDE, et
// `TeamID` vaut quand même celle du film.
func TestExtractPrendLEquipeDuPied(t *testing.T) {
	film := piedFilmEnMemoire(t)
	evs, ctl := Extract(piedFilm, "Strongholds:Arena", film, MapRoster{})
	if len(evs) != 1 {
		t.Fatalf("Extract rend %d événement(s), attendu 1", len(evs))
	}
	if evs[0].TimeMS == nil || *evs[0].TimeMS != piedAttenduTime {
		t.Fatalf("Extract : instant %v, attendu %d", evs[0].TimeMS, piedAttenduTime)
	}
	if evs[0].ObjectiveType != ObjectiveTypeZone || evs[0].Source != SourceTh10 {
		t.Fatalf("Extract : type=%q source=%q, attendu %q / %q",
			evs[0].ObjectiveType, evs[0].Source, ObjectiveTypeZone, SourceTh10)
	}
	if evs[0].TeamID == nil || *evs[0].TeamID != piedAttenduTeam {
		t.Fatalf("Extract : TeamID=%v, attendu %d — l'équipe vient du PIED (octet 37), pas du "+
			"roster, qui est vide ici", evs[0].TeamID, piedAttenduTeam)
	}
	// LE CONTRÔLE COMPTE, IL NE POSE RIEN : roster vide, donc un silence et rien d'autre.
	if ctl.Film != 1 || ctl.Silence != 1 || ctl.Accord != 0 || ctl.Contradiction != 0 {
		t.Fatalf("contrôle %+v : un roster VIDE doit rendre 1 lecture du film et 1 silence", ctl)
	}
}

// TestExtractPrendLEquipeDuPiedEtCompteLeControle : le contrôle DISTINGUE l'accord de la
// contradiction, et la contradiction ne change PAS la valeur publiée.
//
// C'est la moitié que le test ci-dessus ne couvre pas : avec un roster vide, `accord` et
// `contradiction` restent structurellement à zéro et une implémentation qui les confondrait
// passerait. Ici la feuille de match DIT quelque chose — d'abord la même équipe, puis une autre.
func TestExtractPrendLEquipeDuPiedEtCompteLeControle(t *testing.T) {
	film := piedFilmEnMemoire(t)
	xuid := formatXUID(piedAttenduXUID)

	evs, ctl := Extract(piedFilm, "Strongholds:Arena", film, MapRoster{xuid: piedAttenduTeam})
	if ctl.Accord != 1 || ctl.Contradiction != 0 || ctl.Silence != 0 {
		t.Fatalf("contrôle %+v : la feuille dit la MÊME équipe, c'est un accord", ctl)
	}
	if evs[0].TeamID == nil || *evs[0].TeamID != piedAttenduTeam {
		t.Fatalf("Extract : TeamID=%v, attendu %d", evs[0].TeamID, piedAttenduTeam)
	}

	autre := piedAttenduTeam + 1
	evs, ctl = Extract(piedFilm, "Strongholds:Arena", film, MapRoster{xuid: autre})
	if ctl.Accord != 0 || ctl.Contradiction != 1 || ctl.Silence != 0 {
		t.Fatalf("contrôle %+v : la feuille dit une AUTRE équipe, c'est une contradiction", ctl)
	}
	if evs[0].TeamID == nil || *evs[0].TeamID != piedAttenduTeam {
		t.Fatalf("Extract : TeamID=%v alors que la feuille disait %d — une contradiction se "+
			"COMPTE, elle ne corrige rien : le film fait foi", evs[0].TeamID, autre)
	}
}

// TestCaptureScorerPrendLeDernierDuCluster : `captureScorer` est l'autre consommateur des
// événements du pied (chemin CTF). Il est PUR — une liste d'événements, un instant de burst — donc
// il se teste sans film : ce qui manquait pour couvrir le chemin CTF, ce n'est pas un fixture,
// c'est ce test.
//
// CE QUI RESTE A ZERO, ET POURQUOI : `extractCTF` lui-même. Il apparie des BURSTS DE CAPTURE —
// détectés dans les chunks de type 2, à six tiers distincts — avec les événements du pied. Une
// fixture de pied ne peut donc pas l atteindre : il faudrait un chunk de jeu, c est-à-dire un
// autre fixture, pour un chemin que ce lot n a fait que renommer. Dit, pas contourné.
func TestCaptureScorerPrendLeDernierDuCluster(t *testing.T) {
	evs := []signaux.FooterEvent{
		{TimeMS: 1000, Team: 0, XUID: 11},
		{TimeMS: 100000, Team: 1, XUID: 22}, // hors de la fenêtre de coïncidence
		{TimeMS: 1500, Team: 1, XUID: 33},   // le t MAX du cluster : c'est l'acteur
		{TimeMS: 1200, Team: 0, XUID: 44},
	}
	got, ok := captureScorer(evs, 1400)
	if !ok {
		t.Fatal("aucun événement coïncident alors que trois le sont")
	}
	if got.XUID != 33 || got.Team != 1 {
		t.Fatalf("acteur rendu : xuid=%d équipe=%d (attendu 33 / 1 — le t MAX du cluster)",
			got.XUID, got.Team)
	}
	// Et hors de toute fenêtre, il se tait au lieu de rendre le moins mauvais.
	if _, ok := captureScorer(evs, 500000); ok {
		t.Fatal("captureScorer a rendu un acteur pour un burst sans aucun événement coïncident")
	}
}

// piedFilmEnMemoire monte un film d'un seul chunk, de type 3, portant le bloc de référence.
func piedFilmEnMemoire(t *testing.T) *source.Film {
	t.Helper()
	bloc, err := os.ReadFile(piedFixture)
	if err != nil {
		t.Fatalf("fixture %s illisible : %v — elle est VERSIONNÉE par la grammaire, son absence "+
			"est une erreur", piedFixture, err)
	}
	dir := t.TempDir()
	nom := filepath.Join(dir, "chunk_40.bin")
	if err := os.WriteFile(nom, bloc, 0o600); err != nil {
		t.Fatalf("écriture de %s : %v", nom, err)
	}
	film, err := source.LoadDir(dir, []types.ChunkMeta{
		{Index: piedChunk, ChunkType: finalise.ChunkTypeTempsForts},
	})
	if err != nil {
		t.Fatalf("chargement du film de test : %v", err)
	}
	return film
}
