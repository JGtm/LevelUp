package replayverite

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/replaybuild"
)

// reel_test.go — LE BANC SUR UN ARTEFACT REEL.
//
// `testdata/bcb6d393.json.gz` : l'artefact du temoin `bcb6d393` (CTF:Arena, Cliffhanger, 15
// joueurs, un slot statborg nomme par le triplet, camps rattaches par `a0`), cuit a `b452391f7`
// (J6-ter, schema 76), ELAGUE aux seules cles que le banc lit (relu par LireDocument puis
// reserialise : environ 2 Mo -> quelques centaines de Ko compresses). Ses faits :
// `testdata/bcb6d393.facts.json`, copie de `film/replay/testdata/equivalence/`.
//
// Regeneration de l'artefact elague, depuis un artefact complet :
//
//	LEVELUP_UPDATE_REPLAYVERITE_ARTEFACT=<chemin>/bcb6d393.json go test ./internal/replayverite/ -run TestReel
//
// Regeneration du bulletin fige (apres un changement VOULU du banc) :
//
//	LEVELUP_UPDATE_REPLAYVERITE_GOLDEN=1 go test ./internal/replayverite/ -run TestReel

const (
	artefactReel = "testdata/bcb6d393.json.gz"
	faitsReels   = "testdata/bcb6d393.facts.json"
	bulletinReel = "testdata/bcb6d393.bulletin.golden"
)

func lireArtefactReel(t *testing.T) *Document {
	t.Helper()
	if src := os.Getenv("LEVELUP_UPDATE_REPLAYVERITE_ARTEFACT"); src != "" {
		elaguer(t, src)
	}
	f, err := os.Open(artefactReel)
	if err != nil {
		t.Fatalf("artefact reel : %v", err)
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("artefact reel : %v", err)
	}
	blob, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("artefact reel : %v", err)
	}
	d, err := LireDocument(blob)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// elaguer relit un artefact complet par la forme du banc et ecrit la version elaguee compressee.
func elaguer(t *testing.T, src string) {
	t.Helper()
	blob, err := os.ReadFile(src) //nolint:gosec // chemin fourni par l'operateur qui regenere
	if err != nil {
		t.Fatal(err)
	}
	d, err := LireDocument(blob)
	if err != nil {
		t.Fatal(err)
	}
	reduit, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(reduit); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artefactReel, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func lireFaitsReels(t *testing.T) replaybuild.FactsFile {
	t.Helper()
	f, err := replaybuild.ReadFactsFile(faitsReels)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestReel_BulletinFige : le bulletin de l'artefact reel est fige. Un changement de valeur est soit
// un changement VOULU du banc (regenerer), soit une regression du banc.
func TestReel_BulletinFige(t *testing.T) {
	d := lireArtefactReel(t)
	faits := lireFaitsReels(t)
	got := decrireBulletin(Noter(d, faits.MatchFacts))
	if os.Getenv("LEVELUP_UPDATE_REPLAYVERITE_GOLDEN") == "1" {
		if err := os.WriteFile(bulletinReel, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(bulletinReel)
	if err != nil {
		t.Fatalf("bulletin fige : %v", err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("bulletin de bcb6d393 change :\n--- obtenu ---\n%s\n--- fige ---\n%s", got, want)
	}
}

// TestReel_ComparaisonAvecSoiEstOk, puis chaque abimage d'un artefact reel rend le verdict attendu.
func TestReel_AbimagesDUnArtefactReel(t *testing.T) {
	faits := lireFaitsReels(t).MatchFacts
	ref := Noter(lireArtefactReel(t), faits)
	if c := Comparer(ref, ref, nil); c.Statut != StatutOK {
		t.Fatalf("un artefact contre lui-meme : %s %+v", c.Statut, c.Bloquants())
	}
	cas := []struct {
		nom    string
		abimer func(d *Document)
		veut   Statut
		mesure string
	}{
		{"vie dupliquee", func(d *Document) { d.Tracks = append(d.Tracks, d.Tracks[0]) }, StatutFaux, ViolDeuxCorps},
		// Retirer une vie cree AUSSI des actions hors vie (ses tirs) : FAUX, et le manque de O-V1 est la.
		{"vie retiree", func(d *Document) { d.Tracks = d.Tracks[1:] }, StatutFaux, ScoreMortsVies},
		{"courbe de kills vide", viderLesKillsDUnJoueurNote, StatutManque, ScoreKills},
		{"paquet fermé perdu", func(d *Document) { d.Coverage.ContinuousFire.Closed-- }, StatutManque, PreuveFermeture},
	}
	for _, k := range cas {
		d := lireArtefactReel(t)
		k.abimer(d)
		c := Comparer(ref, Noter(d, faits), nil)
		if c.Statut != k.veut {
			t.Errorf("%s : statut %s, veut %s", k.nom, c.Statut, k.veut)
			continue
		}
		trouve := false
		for _, x := range c.Bloquants() {
			trouve = trouve || strings.HasPrefix(x.Mesure, k.mesure)
		}
		if !trouve {
			t.Errorf("%s : aucun bloquant %q dans %+v", k.nom, k.mesure, c.Bloquants())
		}
	}
}

// viderLesKillsDUnJoueurNote vide la courbe de kills du premier joueur qui en a et que le banc note
// (un joueur nomme par le triplet est exclu : le vider ne changerait rien).
func viderLesKillsDUnJoueurNote(d *Document) {
	circ := xuidsCirculaires(d)
	for i, p := range d.ScoreTimeline.Players {
		if p.Kills.Finale() > 0 && !circ[p.XUID] {
			d.ScoreTimeline.Players[i].Kills = Serie{}
			return
		}
	}
}

// decrireBulletin rend un bulletin en texte stable, instances comprises.
func decrireBulletin(b Bulletin) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "match %s\n", b.MatchID)
	for _, id := range clesTriees(b.Scores) {
		s := b.Scores[id]
		fmt.Fprintf(&sb, "%s : %s\n", id, formatScore(s))
		for _, k := range clesTriees(s.Ecarts) {
			fmt.Fprintf(&sb, "  %s : publie %d / officiel %d\n", k, s.Ecarts[k].Pub, s.Ecarts[k].Off)
		}
	}
	for _, id := range clesTriees(b.Preuves) {
		fmt.Fprintf(&sb, "%s : %d\n", id, b.Preuves[id].Valeur)
	}
	for _, id := range clesTriees(b.Violations) {
		v := b.Violations[id]
		fmt.Fprintf(&sb, "%s : %s\n", id, formatViolation(v))
		for _, x := range v.Instances {
			fmt.Fprintf(&sb, "  %s\n", x)
		}
	}
	for _, n := range clesTriees(b.Replis) {
		fmt.Fprintf(&sb, "R-1 %s : %d\n", n, b.Replis[n])
	}
	return sb.String()
}
