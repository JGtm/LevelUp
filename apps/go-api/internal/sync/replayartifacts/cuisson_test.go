package replayartifacts

// cuisson_test.go — LES PROTECTIONS DU CYCLE DE CUISSON (lot 5 de PLAN_CUISSON_PERF).
//
// Cinq proprietes, et aucune ne decode le moindre film : le pont disque et la cuisson sont
// injectes.
//
//  1. l'artefact deja sur disque est lu UNE fois par match (item 5.3) ;
//  2. une cuisson qui ne rend jamais la main est COUPEE, comptee en echec, et le cycle continue
//     (item 5.5) — sans cette borne, un enfant bloque tenait la synchronisation entiere ;
//  3. le film du match SUIVANT se telecharge PENDANT la cuisson du courant, un seul d'avance,
//     dans l'ordre, et aucune goroutine ne survit au cycle (item 5.6) ;
//  4. un verrou de decodage tenu par un autre processus compte en ECHEC du cycle, jamais en
//     succes ni en blocage (item 5.7) ;
//  5. sous [PlancherCuisson], la cuisson ne part PAS : le match est reporte au cycle suivant
//     sans echec ni WARN — mais SANS cuisson cablee le pont disque continue, le plancher ne
//     protegeant qu'une cuisson (constat 6.3 de la revue de branche).

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/haloclient"
)

// poserArtefactCuisson ecrit un artefact au schema courant, avec ou sans compteurs de joueur.
func poserArtefactCuisson(t *testing.T, repoRoot, matchID string, joueurs int) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	doc := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion, MatchID: matchID,
		Tracks: []replay.Track{{Slot: 1, Team: -1}}}
	doc.ScoreTimeline = &replay.ScoreTimeline{}
	for i := range joueurs {
		doc.ScoreTimeline.Players = append(doc.ScoreTimeline.Players,
			replay.PlayerScore{XUID: strconv.Itoa(2533274819954312 + i)})
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

// TestEtatArtefact_UneSeuleLectureParArtefact — item 5.3.
//
// Les deux questions du garde de fraicheur (« a jour ? » puis « avec des compteurs ? »)
// ouvraient CHACUNE le fichier : deux lectures et deux deserialisations d'un document de
// plusieurs mega-octets, par match et par cycle, sur les deux chemins post-sync. Le compteur de
// lectures d'artefact est la mesure de cette promesse.
func TestEtatArtefact_UneSeuleLectureParArtefact(t *testing.T) {
	repoRoot := t.TempDir()
	const matchID = "64e8adfa-0000-0000-0000-000000000000"
	path := poserArtefactCuisson(t, repoRoot, matchID, 8)
	faits := port.MatchFacts{Players: []port.MatchPlayerFact{{XUID: "2533274819954312"}}}

	avant := observability.LoadCounter(replaybuild.CompteurLecturesArtefact)
	aJour, complet := etatArtefact(path, faits)
	lectures := observability.LoadCounter(replaybuild.CompteurLecturesArtefact) - avant

	if !aJour || !complet {
		t.Fatalf("etatArtefact = (%v, %v), attendu (true, true) — l'oracle du cas est faux", aJour, complet)
	}
	if lectures != 1 {
		t.Errorf("%d lecture(s) disque pour un artefact, attendu 1 : le garde de fraicheur "+
			"rouvre le meme document pour sa seconde question", lectures)
	}
}

// TestEtatArtefact_ArtefactPerime_NeLitQuUneFois — meme promesse sur le chemin ou l'artefact est
// perime : la seconde question ne se pose meme pas, elle ne doit surtout pas relire.
func TestEtatArtefact_ArtefactPerime_NeLitQuUneFois(t *testing.T) {
	repoRoot := t.TempDir()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, "vieux")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	avant := observability.LoadCounter(replaybuild.CompteurLecturesArtefact)
	if aJour, _ := etatArtefact(path, port.MatchFacts{}); aJour {
		t.Fatal("un artefact de schema 1 ne peut pas etre a jour")
	}
	if n := observability.LoadCounter(replaybuild.CompteurLecturesArtefact) - avant; n != 1 {
		t.Errorf("%d lecture(s), attendu 1", n)
	}
}

// fetcherFilms : un client film qui rend un chunk minuscule et NOTE l'ordre de ses appels.
type fetcherFilms struct {
	mu sync.Mutex
	// vus : les matchs demandes, dans l'ordre ou le pont disque les a demandes.
	vus []string
	// signale : ferme a chaque appel, pour que le test observe qu'un prechargement est parti.
	signale map[string]chan struct{}
}

func (f *fetcherFilms) GetFilmChunks(ctx context.Context, matchID string) ([]haloclient.FilmChunk, bool, error) {
	f.mu.Lock()
	f.vus = append(f.vus, matchID)
	sig, aSignal := f.signale[matchID]
	f.mu.Unlock()
	if aSignal {
		close(sig)
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	return []haloclient.FilmChunk{{Index: 0, ChunkType: 2, Data: []byte("x")}}, true, nil
}

func (f *fetcherFilms) ordre() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.vus...)
}

// TestBuildAll_CuissonBloquee_CoupeeEtComptee — item 5.5.
//
// Une cuisson qui ne rend jamais la main tenait le cycle de synchronisation INDEFINIMENT, et
// avec lui tout ce qui vient apres. La deadline la coupe : le film compte en echec, et le film
// SUIVANT est traite normalement — meme doctrine que le protocole de codes de sortie, la sante
// de la passe ne depend pas de la sante d'un film.
func TestBuildAll_CuissonBloquee_CoupeeEtComptee(t *testing.T) {
	var appels int
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug,
		CacheRoot: t.TempDir(), Fetcher: &fetcherFilms{},
		Budget: time.Minute, DeadlineParFilm: 80 * time.Millisecond,
		BuildOne: func(ctx context.Context, req BuildOneRequest) (BuildOneResult, error) {
			appels++
			if req.MatchID == "bloque" {
				<-ctx.Done() // l'enfant ne rend jamais la main : seule la deadline le coupe
				return BuildOneResult{}, ctx.Err()
			}
			return BuildOneResult{}, errors.New("echec ordinaire")
		},
	}
	debut := time.Now()
	b := buildAll(context.Background(), d, []buildWork{{matchID: "bloque"}, {matchID: "suivant"}})

	if appels != 2 {
		t.Fatalf("%d cuisson(s) tentee(s), attendu 2 : le cycle s'est arrete sur le film bloque", appels)
	}
	if b.echecs != 2 {
		t.Errorf("echecs = %d, attendu 2 (le film coupe compte en echec, jamais en succes)", b.echecs)
	}
	if b.construits != 0 {
		t.Errorf("construits = %d, attendu 0", b.construits)
	}
	if ecoule := time.Since(debut); ecoule > 5*time.Second {
		t.Errorf("le cycle a dure %v : la deadline n'a pas coupe l'enfant bloque", ecoule)
	}
}

// TestBuildAll_DeadlineSuitLeSoldeDeBudget — la borne d'un film est le MINIMUM du solde de
// budget et de la borne dure : rien ne sert de laisser courir un enfant un quart d'heure quand
// le cycle s'arrete dans trois minutes.
func TestBuildAll_DeadlineSuitLeSoldeDeBudget(t *testing.T) {
	if got := deadlineDuFilm(Deps{}, 3*time.Minute); got != 3*time.Minute {
		t.Errorf("solde de 3 min : borne = %v, attendu 3m (le solde commande)", got)
	}
	if got := deadlineDuFilm(Deps{}, time.Hour); got != DeadlineParFilm {
		t.Errorf("solde d'une heure : borne = %v, attendu %v (la borne dure commande)", got, DeadlineParFilm)
	}
	if got := deadlineDuFilm(Deps{DeadlineParFilm: time.Second}, time.Hour); got != time.Second {
		t.Errorf("borne injectee : %v, attendu 1s", got)
	}
}

// TestBuildAll_SoldeSousLePlancher_ReporteSansEchec — constat 6.3 de la revue de branche.
//
// LE CAS QUE LA REVUE A TROUVE : un solde de budget POSITIF (la garde d'entree de boucle passe
// donc) mais minuscule. La borne du film valait alors ce solde, `cuireUnMatch` recevait un
// contexte deja expire, l'enfant mourait a la naissance, le film comptait en ECHEC et un WARN
// « artefact rejeu non construit » accusait le decodage d'une panne qui n'existe pas. Sous
// [PlancherCuisson], la cuisson NE PART PLUS : le budget s'applique entre deux matchs, et un
// report est nominal (Info), pas un incident.
func TestBuildAll_SoldeSousLePlancher_ReporteSansEchec(t *testing.T) {
	f := &fetcherFilms{}
	avant := runtime.NumGoroutine()
	var appels int
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: f, Budget: PlancherCuisson / 2,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			appels++
			return BuildOneResult{}, errors.New("aucune cuisson ne doit partir sous le plancher")
		},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})

	if appels != 0 {
		t.Errorf("%d cuisson(s) lancee(s) sous le plancher, attendu 0", appels)
	}
	if b.echecs != 0 {
		t.Errorf("echecs = %d, attendu 0 : un report n'est pas un echec", b.echecs)
	}
	if b.construits != 0 {
		t.Errorf("construits = %d, attendu 0", b.construits)
	}
	if !b.budgetEpuise {
		t.Error("le bilan ne signale pas le budget epuise : le cycle suivant ne saurait pas qu'il reste du travail")
	}
	// LE FILM EST QUAND MEME PERSISTE, et c'est le point : il EXPIRE cote serveur Halo,
	// l'artefact non. Le report ne coute donc pas le film.
	if b.filmsSauves != 1 {
		t.Errorf("films persistes = %d, attendu 1 (le pont disque a fait son travail avant le report)",
			b.filmsSauves)
	}
	verifierAucuneGoroutineSurvivante(t, avant)
}

// TestBuildAll_SoldeSousLePlancher_SansCuissonCablee_PersisteQuandMeme — le REVERS du plancher.
//
// Sans `BuildOne`, la boucle n'est plus qu'un pont disque : il n'y a aucune cuisson a proteger
// d'une deadline derisoire, et un arret anticipe ne ferait que perdre des films — qui EXPIRENT
// cote serveur Halo, la ou un artefact se refait. Le plancher ne doit donc pas s'appliquer.
func TestBuildAll_SoldeSousLePlancher_SansCuissonCablee_PersisteQuandMeme(t *testing.T) {
	f := &fetcherFilms{}
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: f, Budget: PlancherCuisson / 2, // sous le plancher, mais positif
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})

	if b.filmsSauves != 2 {
		t.Errorf("films persistes = %d, attendu 2 : sans cuisson cablee, le pont disque continue",
			b.filmsSauves)
	}
	if b.budgetEpuise {
		t.Error("budget signale epuise : le plancher ne s'applique pas quand rien ne cuit")
	}
	if b.echecs != 0 || b.construits != 0 {
		t.Errorf("bilan = %+v, attendu 0 echec et 0 construit", b)
	}
}

// TestBuildAll_PrechargeLeFilmSuivantPendantLaCuisson — item 5.6.
//
// LE CAS PROUVE LA SIMULTANEITE, pas seulement l'ordre : la cuisson du premier match ne rend la
// main qu'APRES avoir vu passer la demande de film du second. Un pont disque sequentiel
// echouerait donc ici sur son delai de garde au lieu de passer par hasard.
func TestBuildAll_PrechargeLeFilmSuivantPendantLaCuisson(t *testing.T) {
	demandeM2 := make(chan struct{})
	f := &fetcherFilms{signale: map[string]chan struct{}{"m2": demandeM2}}
	avant := runtime.NumGoroutine()

	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: f, Budget: time.Minute,
		BuildOne: func(ctx context.Context, req BuildOneRequest) (BuildOneResult, error) {
			if req.MatchID == "m1" {
				select {
				case <-demandeM2: // le prechargement de m2 est parti PENDANT la cuisson de m1
				case <-time.After(5 * time.Second):
					t.Error("le film suivant n'a pas ete precharge pendant la cuisson du courant")
				}
			}
			return BuildOneResult{}, errors.New("pas de cuisson reelle dans ce cas")
		},
	}
	b := buildAll(context.Background(), d,
		[]buildWork{{matchID: "m1"}, {matchID: "m2"}, {matchID: "m3"}})

	// L'ORDRE DES ECRITURES EST CELUI DU LOT : precharger ne rebat pas les cartes.
	ordre := f.ordre()
	if len(ordre) != 3 || ordre[0] != "m1" || ordre[1] != "m2" || ordre[2] != "m3" {
		t.Errorf("ordre des films telecharges = %v, attendu [m1 m2 m3]", ordre)
	}
	// CHAQUE FILM EST TELECHARGE UNE FOIS : le prechargement est CONSOMME, jamais refait.
	if b.filmsSauves != 3 {
		t.Errorf("films persistes = %d, attendu 3 (un prechargement consomme n'est pas re-telecharge)",
			b.filmsSauves)
	}
	verifierAucuneGoroutineSurvivante(t, avant)
}

// TestBuildAll_BudgetEpuise_NePrechargePlus — le prechargement ne deborde pas le budget : passe
// la borne du cycle, la bande passante depensee servirait un film que personne ne cuira.
func TestBuildAll_BudgetEpuise_NePrechargePlus(t *testing.T) {
	f := &fetcherFilms{}
	avant := runtime.NumGoroutine()
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: f, Budget: -1, // deja epuise : la garde s'applique avant le premier match
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			return BuildOneResult{}, errors.New("pas de cuisson reelle dans ce cas")
		},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})
	if !b.budgetEpuise {
		t.Fatal("le bilan ne signale pas le budget epuise")
	}
	if ordre := f.ordre(); len(ordre) != 0 {
		t.Errorf("films telecharges = %v, attendu aucun : le budget etait deja epuise", ordre)
	}
	verifierAucuneGoroutineSurvivante(t, avant)
}

// TestBuildAll_VerrouTenuAilleurs_CompteEnEchec — item 5.7, vu du cycle.
//
// Le refus du verrou de decodage remonte comme n'importe quel echec de cuisson : le film compte
// en ECHEC, le cycle CONTINUE, et rien n'est ecrit. C'est ce qui permet au post-sync de refuser
// tout de suite (le match revient au cycle suivant) plutot que d'attendre son tour derriere un
// decodage qui ne lui appartient pas.
func TestBuildAll_VerrouTenuAilleurs_CompteEnEchec(t *testing.T) {
	d := Deps{
		RepoRoot: t.TempDir(), TitleSlug: titlePkg.DefaultSlug, CacheRoot: t.TempDir(),
		Fetcher: &fetcherFilms{}, Budget: time.Minute,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			return BuildOneResult{}, filmproc.ErrDecodeBusy
		},
	}
	b := buildAll(context.Background(), d, []buildWork{{matchID: "m1"}, {matchID: "m2"}})
	if b.echecs != 2 || b.construits != 0 {
		t.Errorf("bilan = %+v, attendu 2 echecs et 0 construit — un verrou tenu n'est ni un "+
			"succes ni un arret de cycle", b)
	}
}

// verifierAucuneGoroutineSurvivante : le prechargement ne doit rien laisser courir apres le
// cycle. Une goroutine du cycle N qui ecrirait pendant le cycle N+1 rendrait le pic memoire du
// serveur imprevisible — c'est precisement ce que la profondeur 1 borne.
func verifierAucuneGoroutineSurvivante(t *testing.T, avant int) {
	t.Helper()
	for range 100 {
		if runtime.NumGoroutine() <= avant {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("goroutines : %d avant le cycle, %d apres — un prechargement a survecu au cycle",
		avant, runtime.NumGoroutine())
}
