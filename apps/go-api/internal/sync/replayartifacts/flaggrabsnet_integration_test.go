//go:build integration

package replayartifacts

// flaggrabsnet_integration_test.go — LE CROCHET DES PRISES NETTES, EXECUTE POUR DE VRAI.
//
// MEME RAISON QUE bombstats_integration_test.go : le chemin complet — artefact range sur
// disque -> relecture JSON -> regle du jonglage -> persister INSERT-only -> vue
// `match_flag_grabs_net_latest` — traverse quatre paquets et deux serialisations.
//
// CE FICHIER EXISTE PARCE QUE LA REVUE DU 2026-09-13 A MUTE LE CODE ET N'A RIEN CASSE
// (constat C1) : neutraliser la porte de capability (`caps.Has(...) && false`) laissait toutes
// les suites vertes. Deux tests ci-dessous rougissent sous cette mutation — celui du titre
// sans capability, et celui du titre sans fenetre declaree.
//
// LES DEUX PORTES SONT JUGEES SUR LES VRAIS FICHIERS DU DEPOT : halo_infinite declare
// `film.flag_grabs_net` ET sa fenetre ; halo_5 ne declare ni l'une ni l'autre.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// artefactDrapeau forge un artefact de CTF minimal mais complet : deux drapeaux, un jonglage
// (deux portages du meme joueur a 800 ms d'ecart), une passe de main, et un roster de TROIS
// humains dont un qui n'a jamais touche le drapeau.
func artefactDrapeau(t *testing.T, dir, matchID string) string {
	t.Helper()
	x := func(s string) *string { return &s }
	doc := replay.ReplayDocument{
		SchemaVersion:   replay.SchemaVersion,
		MatchID:         matchID,
		FrameIntervalMS: 100,
		Roster: []replay.RosterEntry{
			{XUID: "111", Name: "Un"}, {XUID: "222", Name: "Deux"},
			{XUID: "333", Name: "Trois"},             // jamais de drapeau : ligne A ZERO attendue
			{XUID: "", Name: "Bot [bot]", Bot: true}, // AUCUNE ligne : un bot n'a pas de xuid
		},
		FlagCarries: []replay.FlagCarry{{Team: 0, Spans: []replay.FlagSpan{
			{State: replay.FlagStateCarried, T0: 50, T1: 100, XUID: x("111")},
			{State: replay.FlagStateDropped, T0: 100, T1: 108},
			// Reprise 800 ms plus tard par le MEME joueur : jonglage, replie.
			{State: replay.FlagStateCarried, T0: 108, T1: 140, XUID: x("111")},
			// Passe de main : elle COMPTE.
			{State: replay.FlagStateCarried, T0: 145, T1: 180, XUID: x("222")},
		}}},
		Coverage: &replay.Coverage{FlagCarries: &replay.FlagCarriesCoverage{
			FlagFilm: true, Openings: 5, Bursts: 3, Captures: 3, Steals: 4,
		}},
	}
	return ecrireArtefactDrapeau(t, dir, matchID, doc)
}

// artefactHorsCTF forge un artefact dont le verdict `flagFilm` est FAUX — le cas majoritaire
// du parc (63 artefacts sur 76 au releve du 2026-09-13).
func artefactHorsCTF(t *testing.T, dir, matchID string) string {
	t.Helper()
	return ecrireArtefactDrapeau(t, dir, matchID, replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: matchID, FrameIntervalMS: 100,
		Coverage: &replay.Coverage{FlagCarries: &replay.FlagCarriesCoverage{FlagFilm: false}},
	})
}

func ecrireArtefactDrapeau(t *testing.T, dir, matchID string, doc replay.ReplayDocument) string {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	path := filepath.Join(dir, matchID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return path
}

// TestPersisterPrisesNettes_UnMatchCTFEcritSesLignes : le chemin nominal, jonglage replie et
// ZERO MESURE compris.
func TestPersisterPrisesNettes_UnMatchCTFEcritSesLignes(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	ctx := context.Background()

	persisterPrisesNettes(ctx, d, &bilanDerivations{}, lus(d,
		ArtefactRange{MatchID: "m-ctf", Path: artefactDrapeau(t, dir, "m-ctf")},
	))

	if acquis != 1 || relaches != 1 {
		t.Fatalf("writer acquis %d fois, relache %d fois — attendu 1 et 1 (burst unique)", acquis, relaches)
	}
	// LE JONGLAGE EST REPLIE : deux ramassages, une prise nette.
	var brut, net, openings, fenetre int
	err := db.QueryRowContext(ctx, `
		SELECT flag_grabs_raw, flag_grabs_net, openings, juggle_window_ms
		FROM match_flag_grabs_net_latest WHERE match_id = 'm-ctf' AND xuid = '111'`).
		Scan(&brut, &net, &openings, &fenetre)
	if err != nil {
		t.Fatalf("lecture match_flag_grabs_net_latest: %v", err)
	}
	if brut != 2 || net != 1 {
		t.Errorf("joueur 111 = (%d brut, %d net), attendu (2, 1) — le jonglage n'est pas replie", brut, net)
	}
	// LES DEUX FAITS DE MATCH VOYAGENT AVEC LA MESURE.
	if openings != 5 {
		t.Errorf("openings = %d, attendu 5 (le compte de l'oracle du document)", openings)
	}
	if fenetre != 1500 {
		t.Errorf("fenetre = %d ms, attendu 1500 (regulation.toml du titre)", fenetre)
	}
	// LA PASSE DE MAIN COMPTE.
	if err := db.QueryRowContext(ctx, `
		SELECT flag_grabs_raw, flag_grabs_net FROM match_flag_grabs_net_latest
		WHERE match_id = 'm-ctf' AND xuid = '222'`).Scan(&brut, &net); err != nil {
		t.Fatalf("lecture joueur 222: %v", err)
	}
	if brut != 1 || net != 1 {
		t.Errorf("joueur 222 = (%d, %d), attendu (1, 1) — une passe de main est une vraie prise", brut, net)
	}
	// UN ZERO MESURE S'ECRIT 0 : sans sa ligne, ce joueur serait indistinguable d'un joueur
	// d'un match sans film.
	if err := db.QueryRowContext(ctx, `
		SELECT flag_grabs_raw, flag_grabs_net FROM match_flag_grabs_net_latest
		WHERE match_id = 'm-ctf' AND xuid = '333'`).Scan(&brut, &net); err != nil {
		t.Fatalf("lecture joueur 333 (roster, zero prise): %v", err)
	}
	if brut != 0 || net != 0 {
		t.Errorf("joueur 333 = (%d, %d), attendu (0, 0)", brut, net)
	}
	// LE BOT N'A PAS DE LIGNE : il n'a pas de xuid, et son absence n'est pas un zero.
	var lignes int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_flag_grabs_net_latest WHERE match_id = 'm-ctf'`).Scan(&lignes); err != nil {
		t.Fatalf("count: %v", err)
	}
	if lignes != 3 {
		t.Errorf("%d lignes, attendu 3 (trois humains du roster, aucun bot)", lignes)
	}
}

// TestPersisterPrisesNettes_ModeHorsCTFNEcritRien : un artefact dont le verdict `flagFilm` est
// faux n'est PAS un echec — c'est le cas majoritaire. Aucun writer, aucune ligne.
func TestPersisterPrisesNettes_ModeHorsCTFNEcritRien(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)

	persisterPrisesNettes(context.Background(), d, &bilanDerivations{}, lus(d,
		ArtefactRange{MatchID: "m-slayer", Path: artefactHorsCTF(t, dir, "m-slayer")},
	))

	if acquis != 0 {
		t.Fatalf("writer acquis %d fois pour un lot hors CTF, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("un mode hors CTF a ecrit %d ligne(s), attendu 0", n)
	}
}

// TestPersisterPrisesNettes_CapabiliteSeuleFermee : LA PORTE 1, ISOLEE.
//
// Le titre declare sa FENETRE mais PAS la capability. C'est le seul montage qui juge la porte
// de capability SEULE : sur halo_5, les deux portes sont fermees, et un test qui s'y appuie
// resterait vert si la capability etait neutralisee (constate en rejouant la mutation M6 de la
// revue — le test halo_5 passait pour la mauvaise raison).
func TestPersisterPrisesNettes_CapabiliteSeuleFermee(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	d.RepoRoot = racineSansCapabilite(t, d.RepoRoot)

	persisterPrisesNettes(context.Background(), d, &bilanDerivations{}, lus(d,
		ArtefactRange{MatchID: "m-sans-cap", Path: artefactDrapeau(t, dir, "m-sans-cap")},
	))

	if acquis != 0 {
		t.Fatalf("writer acquis %d fois sans la capability, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("un titre sans la capability a ecrit %d ligne(s), attendu 0", n)
	}
}

// TestPersisterPrisesNettes_TitreSansCapability : halo_5, LE CAS REEL — il ne declare NI la
// capability NI la fenetre, et le lot entier est ecarte avant toute projection et tout writer
// (title-agnostic, jamais un slug). Les deux portes sont fermees : c'est le test d'INTEGRATION
// du titre, pas celui d'une porte isolee (cf. les deux tests dedies ci-dessus et ci-dessous).
func TestPersisterPrisesNettes_TitreSansCapability(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_5", &acquis, &relaches)

	persisterPrisesNettes(context.Background(), d, &bilanDerivations{}, lus(d,
		ArtefactRange{MatchID: "m-h5", Path: artefactDrapeau(t, dir, "m-h5")},
	))

	if acquis != 0 {
		t.Fatalf("writer acquis %d fois pour un titre sans capability, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("un titre sans capability a ecrit %d ligne(s), attendu 0", n)
	}
}

// TestPersisterPrisesNettes_TitreSansFenetreDeclaree : LA SECONDE PORTE. Un titre peut
// declarer la capability et ne PAS declarer sa regle de jonglage — il ne publie alors aucune
// prise nette. Le cas est joue sur une racine de depot forgee : capabilities du titre reel,
// regulation.toml SANS la cle.
func TestPersisterPrisesNettes_TitreSansFenetreDeclaree(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	d.RepoRoot = racineSansFenetre(t, d.RepoRoot)

	persisterPrisesNettes(context.Background(), d, &bilanDerivations{}, lus(d,
		ArtefactRange{MatchID: "m-sans-regle", Path: artefactDrapeau(t, dir, "m-sans-regle")},
	))

	if acquis != 0 {
		t.Fatalf("writer acquis %d fois sans fenetre declaree, attendu 0", acquis)
	}
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM match_flag_grabs_net`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("un titre sans fenetre a ecrit %d ligne(s), attendu 0", n)
	}
}

// racineSansFenetre : les mappings REELS du titre, moins la section `[flag_grabs_net]` du
// regulation.toml. Les capabilities restent intactes — c'est bien la SECONDE porte que le test
// juge, pas la premiere.
func racineSansFenetre(t *testing.T, reel string) string {
	t.Helper()
	return racineForgee(t, reel, func(nom string, raw []byte) []byte {
		if nom != "regulation.toml" {
			return raw
		}
		// La section visee est la DERNIERE du fichier : la coupe ne retire rien d'autre.
		return tronquerAvant(raw, "[flag_grabs_net]")
	})
}

// racineSansCapabilite : les mappings REELS du titre, moins la LIGNE `film.flag_grabs_net` du
// capabilities.toml. La fenetre reste declaree — c'est bien la PREMIERE porte que le test
// juge, et elle seule.
func racineSansCapabilite(t *testing.T, reel string) string {
	t.Helper()
	return racineForgee(t, reel, func(nom string, raw []byte) []byte {
		if nom != "capabilities.toml" {
			return raw
		}
		var gardees []string
		for _, ligne := range strings.Split(string(raw), "\n") {
			if strings.Contains(ligne, "\"film.flag_grabs_net\"") {
				continue
			}
			gardees = append(gardees, ligne)
		}
		return []byte(strings.Join(gardees, "\n"))
	})
}

// racineForgee copie les deux mappings du titre dans une racine temporaire, en passant chacun
// par `transformer`. Une racine forgee plutot qu'un autre titre : c'est le SEUL moyen de juger
// une porte SEULE, les titres reels fermant les deux a la fois.
func racineForgee(t *testing.T, reel string, transformer func(nom string, raw []byte) []byte) string {
	t.Helper()
	racine := t.TempDir()
	dst := filepath.Join(racine, "config", "titles", "halo_infinite", "mappings")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := filepath.Join(reel, "config", "titles", "halo_infinite", "mappings")
	// TOUS les mappings, pas les deux qui nous interessent : `LoadCapabilityMap` charge le
	// jeu COMPLET du titre et ECHOUE si `fields.toml` manque. Une racine partielle rendait le
	// test vert par le chemin « capabilities illisibles » — donc vert MEME avec la porte
	// neutralisee, ce qui est exactement le defaut que ce fichier existe pour attraper
	// (constate en rejouant la mutation M6).
	entrees, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("lecture des mappings du titre: %v", err)
	}
	for _, e := range entrees {
		if e.IsDir() {
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(src, e.Name()))
		if rerr != nil {
			t.Fatalf("lecture %s: %v", e.Name(), rerr)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), transformer(e.Name(), raw), 0o644); err != nil {
			t.Fatalf("ecriture %s: %v", e.Name(), err)
		}
	}
	return racine
}

// tronquerAvant coupe le contenu juste avant la premiere occurrence du marqueur.
func tronquerAvant(raw []byte, marqueur string) []byte {
	if i := strings.Index(string(raw), marqueur); i >= 0 {
		return raw[:i]
	}
	return raw
}
