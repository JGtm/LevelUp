package main

// cmd_backfill_killsource_etat_test.go — LE FICHIER D ETAT DIT LA VERITE, ET `--status` LA LIT
// (lot 5.24.3).
//
// Aucune base, aucun film : ce qui est teste ici est la COMPTABILITE de la passe et sa
// restitution. Le chainage reel (collecteur -> observateur) est couvert par le test
// d interruption de 5.24.4.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/sync/killcollector"
)

// suiviDeTest construit un suivi sur un fichier temporaire, avec trois films de couts differents.
func suiviDeTest(t *testing.T, ouvriers int) (*suiviDeLaPasse, string) {
	t.Helper()
	chemin := filepath.Join(t.TempDir(), "etat", "backfill_killsource_halo_infinite.json")
	candidats := []filmCandidat{
		{matchID: "petit", chunks: 5},
		{matchID: "moyen", chunks: 30},
		{matchID: "gros", chunks: 65},
	}
	bilan := bilanDeSelection{TotalRegistre: 1600, DejaAJour: 1500, SansFilmEnCache: 97}
	return nouveauSuivi(chemin, "halo_infinite", candidats, bilan,
		killsourceOptions{workers: ouvriers}), chemin
}

// lireEtat relit le fichier — comme `--status` le fait.
func lireEtat(t *testing.T, chemin string) etatDeLaPasse {
	t.Helper()
	raw, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("fichier d etat illisible: %v", err)
	}
	var e etatDeLaPasse
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("etat non deserialisable: %v", err)
	}
	return e
}

// TestEtat_EcritDesLeDemarrage — `--status` tape dans la seconde qui suit le lancement doit
// repondre « elle demarre », pas « aucun fichier ».
func TestEtat_EcritDesLeDemarrage(t *testing.T) {
	_, chemin := suiviDeTest(t, 3)
	e := lireEtat(t, chemin)
	if e.Phase != phaseFilms {
		t.Errorf("phase = %q, attendu %q", e.Phase, phaseFilms)
	}
	if e.Films.AFaire != 3 || e.Films.ChunksAFaire != 100 {
		t.Errorf("a_faire = %d / chunks_a_faire = %d, attendu 3 / 100", e.Films.AFaire, e.Films.ChunksAFaire)
	}
	if e.RepriseDe != 1500 || e.Films.TotalRegistre != 1600 {
		t.Errorf("repris_de = %d, total_registre = %d, attendu 1500 / 1600", e.RepriseDe, e.Films.TotalRegistre)
	}
	if e.RevisionMorts == "" || e.RevisionIsol == "" {
		t.Error("les deux revisions cibles doivent etre ecrites : c est ce qui dit POURQUOI la " +
			"passe redecode")
	}
	if e.PID != os.Getpid() {
		t.Errorf("pid = %d, attendu %d", e.PID, os.Getpid())
	}
}

// TestEtat_ChaqueFilmEstComptabilise — la comptabilite des cinq issues, et les chunks avec.
func TestEtat_ChaqueFilmEstComptabilise(t *testing.T) {
	s, chemin := suiviDeTest(t, 2)
	s.FilmDemarre("petit", time.Now())
	s.FilmDemarre("moyen", time.Now())

	enCours := lireEtat(t, chemin).Films.EnCours
	if len(enCours) != 2 || enCours[0].MatchID != "moyen" || enCours[1].MatchID != "petit" {
		t.Fatalf("en_cours = %+v, attendu les deux films dans l ordre stable moyen/petit", enCours)
	}

	s.FilmFini(killcollector.EvenementDeFilm{MatchID: "petit",
		Outcome: killcollector.OutcomeWritten, Morts: 12, Duree: time.Second})
	s.FilmFini(killcollector.EvenementDeFilm{MatchID: "moyen",
		Outcome: killcollector.OutcomeNoKillFeed, Duree: 3 * time.Second})

	e := lireEtat(t, chemin)
	f := e.Films
	if f.Traites != 2 || f.Ecrits != 1 || f.Morts != 12 || f.SansKillFeed != 1 {
		t.Errorf("traites/ecrits/morts/sans_kill_feed = %d/%d/%d/%d, attendu 2/1/12/1",
			f.Traites, f.Ecrits, f.Morts, f.SansKillFeed)
	}
	if f.ChunksFaits != 35 {
		t.Errorf("chunks_faits = %d, attendu 35 (5 + 30)", f.ChunksFaits)
	}
	if len(f.EnCours) != 0 {
		t.Errorf("en_cours = %+v, attendu vide : les deux films sont finis", f.EnCours)
	}
	if f.DernierFini == nil || f.DernierFini.MatchID != "moyen" ||
		f.DernierFini.Resultat != string(killcollector.OutcomeNoKillFeed) {
		t.Errorf("dernier_fini = %+v, attendu moyen/sans-killfeed", f.DernierFini)
	}
	// LE COUT PAR CHUNK EST MESURE, PAS SUPPOSE : 4 s de film pour 35 chunks.
	if got, want := f.CoutParChunkS, arrondi(4.0/35.0); got != want {
		t.Errorf("cout_par_chunk_s = %v, attendu %v (4 s / 35 chunks)", got, want)
	}
}

// TestEtat_UneErreurNEstPasUneIssue — un film en erreur compte comme erreur, PAS comme son
// outcome : la synthese de la passe et le fichier d etat doivent dire la meme chose.
func TestEtat_UneErreurNEstPasUneIssue(t *testing.T) {
	s, chemin := suiviDeTest(t, 1)
	s.FilmFini(killcollector.EvenementDeFilm{MatchID: "petit",
		Outcome: killcollector.OutcomeWritten, Morts: 9, Duree: time.Second,
		Err: os.ErrClosed})
	f := lireEtat(t, chemin).Films
	if f.Erreurs != 1 || f.Ecrits != 0 || f.Morts != 0 {
		t.Errorf("erreurs/ecrits/morts = %d/%d/%d, attendu 1/0/0 — un film en erreur n a rien ecrit",
			f.Erreurs, f.Ecrits, f.Morts)
	}
	if f.DernierFini == nil || f.DernierFini.Resultat != resultatFilmErreur {
		t.Errorf("dernier_fini = %+v, attendu un resultat 'erreur'", f.DernierFini)
	}
}

// TestEtat_ResteEstimeEnChunksEtParOuvrier — L ETA NE COMPTE PAS DES FILMS.
//
// C est LE point du calcul : les gros films sont a la fin, donc un reste compte en NOMBRE de
// films annoncerait une fin proche juste avant la queue la plus chere.
func TestEtat_ResteEstimeEnChunksEtParOuvrier(t *testing.T) {
	f := etatDesFilms{ChunksAFaire: 100, ChunksFaits: 20, CoutParChunkS: 0.5, Ouvriers: 4}
	if got := resteEstime(&f); got != 10 {
		t.Errorf("reste = %v s, attendu 10 (80 chunks x 0,5 s / 4 ouvriers)", got)
	}
	// Le meme etat compte en FILMS aurait dit tout autre chose : 2 films sur 3 faits.
	f.Ouvriers = 0 // un zero ne doit pas diviser
	if got := resteEstime(&f); got != 40 {
		t.Errorf("reste = %v s avec 0 ouvrier, attendu 40 (on compte 1)", got)
	}
	f.ChunksFaits = 100
	if got := resteEstime(&f); got != 0 {
		t.Errorf("reste = %v s alors qu il ne reste aucun chunk, attendu 0", got)
	}
	f.ChunksFaits, f.CoutParChunkS = 0, 0
	if got := resteEstime(&f); got != 0 {
		t.Errorf("reste = %v s sans cout mesure, attendu 0 — on n extrapole pas depuis rien", got)
	}
}

// TestEtat_CadenceDeLaLigneDeProgression — 25 films OU 60 s, le premier des deux, et le
// compteur REPART quelle que soit la cause.
func TestEtat_CadenceDeLaLigneDeProgression(t *testing.T) {
	s, _ := suiviDeTest(t, 1)
	jalons := 0
	for i := 0; i < pasDeProgressionFilms*3; i++ {
		s.depuisLaLigne++
		if s.jalonAtteint() {
			jalons++
		}
	}
	if jalons != 3 {
		t.Errorf("%d jalons pour %d films, attendu 3 (un tous les %d)",
			jalons, pasDeProgressionFilms*3, pasDeProgressionFilms)
	}
	// L HORLOGE SEULE, sur une passe lente : un seul film, mais la derniere ligne est vieille.
	s.derniereLigne = time.Now().Add(-2 * delaiDeProgressionMin)
	s.depuisLaLigne = 1
	if !s.jalonAtteint() {
		t.Errorf("aucun jalon apres %s sans ligne : une passe sur un gros film paraitrait bloquee", delaiDeProgressionMin)
	}
	if s.depuisLaLigne != 0 {
		t.Errorf("le compteur vaut %d apres un jalon d horloge : il doit repartir, sinon une "+
			"passe lente produit une rafale au 25e film", s.depuisLaLigne)
	}
}

// TestEtat_EcritureAtomique — `--status` peut lire pendant que la passe ecrit.
func TestEtat_EcritureAtomique(t *testing.T) {
	s, chemin := suiviDeTest(t, 1)
	s.FilmFini(killcollector.EvenementDeFilm{MatchID: "petit",
		Outcome: killcollector.OutcomeWritten, Duree: time.Second})
	if _, err := os.Stat(chemin + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("le fichier temporaire survit a l ecriture (%v) : le renommage n a pas eu lieu", err)
	}
}

// TestEtat_PhasesEtCredit — une commande, un etat : la passe credit ecrit dans le meme fichier.
func TestEtat_PhasesEtCredit(t *testing.T) {
	s, chemin := suiviDeTest(t, 1)
	s.PhaseCredit(9144)
	s.ProgressionCredit(500, 90*time.Second)
	e := lireEtat(t, chemin)
	if e.Phase != phaseCredit || e.Credit.AExaminer != 9144 || e.Credit.Examines != 500 {
		t.Errorf("phase/credit = %q/%+v, attendu %q + 500/9144", e.Phase, e.Credit, phaseCredit)
	}
	s.Terminee("SIGINT")
	e = lireEtat(t, chemin)
	if e.Phase != phaseTerminee || e.Interrompue != "SIGINT" {
		t.Errorf("phase/interrompue = %q/%q, attendu %q/SIGINT", e.Phase, e.Interrompue, phaseTerminee)
	}
}

// TestStatus_RenduLisible — ce que l utilisateur voit dans son second terminal.
func TestStatus_RenduLisible(t *testing.T) {
	s, chemin := suiviDeTest(t, 3)
	s.FilmDemarre("gros", time.Now().Add(-12*time.Second))
	s.FilmFini(killcollector.EvenementDeFilm{MatchID: "petit",
		Outcome: killcollector.OutcomeWritten, Morts: 42, Duree: 2 * time.Second})
	e := lireEtat(t, chemin)

	sortie := rendreEtat(e, time.Now())
	// LA SORTIE EST JOURNALISEE : c est l exemple que la doc cite, et le citer depuis le test
	// evite qu il vieillisse sans qu on le voie.
	t.Logf("exemple de sortie de --status :\n%s", sortie)
	for _, attendu := range []string{
		"backfill-killsource [halo_infinite]", "phase films",
		"1 / 3 traites", "42 morts", "1500 deja a jour", "3 ouvrier(s)",
		"dernier fini petit", "EN COURS     gros",
	} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("la sortie ne contient pas %q :\n%s", attendu, sortie)
		}
	}
}

// TestStatus_EtatPerimeLeDit — un chiffre vieux d une heure ne doit pas se lire comme un
// chiffre courant.
func TestStatus_EtatPerimeLeDit(t *testing.T) {
	e := etatDeLaPasse{Commande: "backfill-killsource", Phase: phaseFilms,
		MiseAJourA: time.Now().Add(-time.Hour)}
	if !strings.Contains(rendreEtat(e, time.Now()), "AUCUNE MISE A JOUR DEPUIS") {
		t.Error("un etat vieux d une heure ne s annonce pas comme perime")
	}
	// Une passe TERMINEE, elle, a le droit d etre vieille.
	e.Phase = phaseTerminee
	if strings.Contains(rendreEtat(e, time.Now()), "AUCUNE MISE A JOUR DEPUIS") {
		t.Error("une passe terminee est signalee comme perimee alors qu elle est finie")
	}
}

// TestStatus_FichierAbsentNEstPasUneErreur — `--status` avant toute passe doit EXPLIQUER, pas
// echouer : un code de sortie non nul ferait croire a une panne.
func TestStatus_FichierAbsentNEstPasUneErreur(t *testing.T) {
	if err := afficherEtatDeLaPasse(filepath.Join(t.TempDir(), "absent.json")); err != nil {
		t.Errorf("fichier absent rendu comme une erreur : %v", err)
	}
}

// TestStatus_FichierTronqueEstUneErreur — un JSON coupe n est PAS un etat vide : le dire.
func TestStatus_FichierTronqueEstUneErreur(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "tronque.json")
	if err := os.WriteFile(chemin, []byte(`{"commande":"backfill-kill`), 0o644); err != nil {
		t.Fatalf("ecriture: %v", err)
	}
	if err := afficherEtatDeLaPasse(chemin); err == nil {
		t.Error("un fichier tronque passe pour un etat valide")
	}
}

// TestBilanInitial_PorteLeTempsEtLesChunks — ce que `--dry-run` doit dire.
func TestBilanInitial_PorteLeTempsEtLesChunks(t *testing.T) {
	bilan := bilanInitial([]filmCandidat{{matchID: "a", chunks: 10}, {matchID: "b", chunks: 65}},
		1600, 1500, 3)
	for _, attendu := range []string{"1600 matchs au registre", "1500 deja a jour",
		"2 films a decoder", "75 chunks", "1 film(s) au-dela de 50 chunks", "3 ouvrier(s)"} {
		if !strings.Contains(bilan, attendu) {
			t.Errorf("le bilan ne contient pas %q :\n%s", attendu, bilan)
		}
	}
}
