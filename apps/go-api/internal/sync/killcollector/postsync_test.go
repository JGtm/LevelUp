package killcollector

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
)

// TestOrdonnancer_InseresPuisPlusVieux : L ORDRE EST UNE COURSE CONTRE L EXPIRATION.
//
// Les films expirent cote serveur Halo et un film expire ne se retelecharge jamais. Une passe
// bornee doit donc sauver d abord les matchs frais (leur film est sur) puis les PLUS VIEUX du
// backlog — ceux a qui il reste le moins de temps. Un tri par recence perdrait exactement les
// films qu il fallait sauver.
func TestOrdonnancer_InseresPuisPlusVieux(t *testing.T) {
	// backlog rendu du plus vieux au plus recent par la requete.
	backlog := []string{"vieux1", "vieux2", "vieux3", "recent1", "recent2"}
	travail, restant := ordonnancer(backlog, []string{"recent2"}, 3)

	if len(travail) != 3 {
		t.Fatalf("travail = %v, attendu 3 entrees", travail)
	}
	if travail[0] != "recent2" {
		t.Errorf("travail[0] = %q, attendu l insere « recent2 »", travail[0])
	}
	if travail[1] != "vieux1" || travail[2] != "vieux2" {
		t.Errorf("suite = %v, attendus les plus vieux d abord (vieux1, vieux2)", travail[1:])
	}
	if restant != 2 {
		t.Errorf("restant = %d, attendu 2 — un backlog non rendu ne s observe pas", restant)
	}
}

// TestOrdonnancer_InsereDejaAJourIgnore : un match insere qui n est PAS au backlog est deja
// decode a la revision courante. Le reprendre ferait payer le reseau pour rien, et surtout
// prendrait la place d un match du backlog dans une passe bornee.
func TestOrdonnancer_InsereDejaAJourIgnore(t *testing.T) {
	travail, restant := ordonnancer([]string{"a", "b"}, []string{"deja-fait"}, 5)
	if len(travail) != 2 || travail[0] != "a" || travail[1] != "b" {
		t.Fatalf("travail = %v, attendu [a b] — « deja-fait » n est pas au backlog", travail)
	}
	if restant != 0 {
		t.Errorf("restant = %d, attendu 0", restant)
	}
}

// TestOrdonnancer_AucunDoublon : un match insere ET au backlog ne doit etre decode qu une fois
// — sinon la borne par cycle protegerait moins qu elle ne le dit.
func TestOrdonnancer_AucunDoublon(t *testing.T) {
	travail, _ := ordonnancer([]string{"a", "b", "c"}, []string{"b", "b"}, 10)
	if len(travail) != 3 {
		t.Fatalf("travail = %v, attendu 3 entrees distinctes", travail)
	}
	vus := map[string]int{}
	for _, id := range travail {
		vus[id]++
	}
	for id, n := range vus {
		if n != 1 {
			t.Errorf("%s apparait %d fois", id, n)
		}
	}
}

// TestOrdonnancer_BacklogVide : rien a faire, et rien a signaler.
func TestOrdonnancer_BacklogVide(t *testing.T) {
	travail, restant := ordonnancer(nil, []string{"x"}, 5)
	if len(travail) != 0 || restant != 0 {
		t.Errorf("travail = %v, restant = %d ; attendu vide et 0", travail, restant)
	}
}

// TestRunPostSync_InerteSansDependance : l etape est BEST-EFFORT, et son inertie ne doit
// jamais devenir une panne. Un client sans capacite film, un moteur sans segment de lecture,
// un hook absent : dans les trois cas elle ne fait rien et ne panique pas.
func TestRunPostSync_InerteSansDependance(t *testing.T) {
	ctx := context.Background()
	appele := false
	withRead := func(_ context.Context, _ string, fn func(*sql.DB)) { appele = true; fn(nil) }
	writer := persist.SharedWriterFn(func(context.Context) (*sql.DB, func(), error) { return nil, func() {}, nil })

	RunPostSync(ctx, nil, PostSyncDeps{Fetcher: &filmsEnMemoire{}, WithRead: withRead, AcquireWriter: writer}, nil)
	RunPostSync(ctx, NewPostSyncHook(t.TempDir(), 0), PostSyncDeps{WithRead: withRead, AcquireWriter: writer}, nil)
	RunPostSync(ctx, NewPostSyncHook(t.TempDir(), 0), PostSyncDeps{Fetcher: &filmsEnMemoire{}, AcquireWriter: writer}, nil)
	if appele {
		t.Error("un segment de lecture a ete ouvert alors qu une dependance manquait")
	}
}

// TestRunPostSync_CapabilityAbsente_EtapeVide : DEGRADATION GRACIEUSE, PAR LA CAPABILITY.
//
// ⚠ CE TEST TESTAIT AUTRE CHOSE QUE CE QU IL ANNONCAIT. Sa premiere version passait un
// `t.TempDir()` en repoRoot : `LoadFromConfigDir` echouait donc, et on sortait sur la branche
// « capabilities illisibles » — jamais sur la porte `caps.Has(CapFilmKillSource)`. Supprimer
// cette porte laissait le test vert. Il vise desormais un titre REEL du depot qui declare ses
// mappings SANS declarer `film.kill_source` (halo_5 : autre format de film, pas de decodeur),
// ce qui est exactement le cas a couvrir — et il l obtient sans jamais comparer un slug.
func TestRunPostSync_CapabilityAbsente_EtapeVide(t *testing.T) {
	lu := false
	ecrits := RunPostSync(context.Background(), NewPostSyncHook(racineDepot(t), 0), PostSyncDeps{
		Fetcher:       &filmsEnMemoire{},
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { lu = true; fn(nil) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return nil, func() {}, nil },
		TitleSlug:     "halo_5",
	}, []string{"m1"})
	if lu {
		t.Error("la base a ete lue alors que halo_5 ne declare pas film.kill_source")
	}
	if ecrits != 0 {
		t.Errorf("ecrits = %d, attendu 0", ecrits)
	}
}

// TestRunPostSync_CapabilitesIllisibles_EtapeVide : le cas VOISIN, distinct du precedent —
// `capabilities.toml` introuvable. Les deux doivent sortir avant toute lecture, mais par des
// chemins differents ; les confondre est ce qui avait rendu le test precedent aveugle.
func TestRunPostSync_CapabilitesIllisibles_EtapeVide(t *testing.T) {
	lu := false
	RunPostSync(context.Background(), NewPostSyncHook(t.TempDir(), 0), PostSyncDeps{
		Fetcher:       &filmsEnMemoire{},
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { lu = true; fn(nil) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return nil, func() {}, nil },
		TitleSlug:     "titre_inexistant",
	}, []string{"m1"})
	if lu {
		t.Error("la base a ete lue alors que les capabilities sont illisibles")
	}
}

// TestPostSyncHook_CapabilitesMemorisees : le succes ET l echec sont memorises. Sans cela un
// fichier absent produirait le meme WARN a chaque cycle de sync ; avec une memoisation posee
// trop tot, un echec transitoire figerait l etape pour la vie du process.
func TestPostSyncHook_CapabilitesMemorisees(t *testing.T) {
	h := NewPostSyncHook(racineDepot(t), 0)
	caps1, err1 := h.capabilities("halo_infinite")
	caps2, err2 := h.capabilities("halo_infinite")
	if err1 != nil || err2 != nil {
		t.Fatalf("capabilities halo_infinite : %v / %v", err1, err2)
	}
	if !caps1.Has(games.CapFilmKillSource) {
		t.Error("halo_infinite doit declarer film.kill_source — sinon l etape ne tourne nulle part")
	}
	// La memoisation rend le MEME resultat, y compris pour un autre slug : un hook est lie a
	// UN moteur, donc a UN titre (cf. NewSyncEngineForTitle).
	if caps3, _ := h.capabilities("halo_5"); !sameCaps(caps2, caps3) {
		t.Error("la memoisation doit rendre le resultat du premier titre resolu")
	}
}

func sameCaps(a, b games.CapabilityMap) bool {
	return a.Has(games.CapFilmKillSource) == b.Has(games.CapFilmKillSource)
}

// racineDepot : la racine du depot depuis ce paquet — c est la que vivent
// `config/titles/*/mappings/capabilities.toml`. Le test SAUTE si elle est introuvable
// plutot que d echouer : un checkout partiel n est pas une regression du lot.
func racineDepot(t *testing.T) string {
	t.Helper()
	// killcollector -> sync -> internal -> go-api -> apps -> racine : CINQ niveaux.
	racine := filepath.Join("..", "..", "..", "..", "..")
	// PAS DE t.Skip ICI. Un test qui saute est un test qui ne mord pas : la premiere version
	// de ce fichier sautait sur une mauvaise profondeur de chemin, et la mutation « retirer la
	// porte capability » passait donc au vert. Si la config manque, c est un echec.
	if _, err := os.Stat(filepath.Join(racine, "config", "titles", "halo_5", "mappings")); err != nil {
		t.Fatalf("config/titles introuvable depuis ce paquet (%s) : %v", racine, err)
	}
	return racine
}

// TestMatchsSansPasseAJour_NeSauteJamaisUnMatchCreditSeul : LE FILTRE QUI REND L ETAPE UTILE.
//
// Un match couvert par le producteur credit-seul porte pourtant la revision de decodeur
// courante. Sans `read_path <> credit-backfill`, la requete le croirait a jour — et comme
// TOUTE la base est couverte par le credit, l etape ne redecoderait plus jamais rien et
// l attribution des assistances ne repartirait pas.
func TestMatchsSansPasseAJour_NeSauteJamaisUnMatchCreditSeul(t *testing.T) {
	src := requeteBacklog
	for _, attendu := range []string{
		"match_kill_events_latest", // vue `_latest`, jamais la table brute (ADR 0026)
		"read_path <> ?",           // le credit-seul ne vaut pas une passe de film
		"decoder_rev = ?",
		"start_time_utc", // tri canonique (regle 8)
	} {
		if !strings.Contains(src, attendu) {
			t.Errorf("la requete du backlog ne contient plus %q — %s", attendu, pourquoi(attendu))
		}
	}
	if strings.Contains(src, "FROM match_kill_events\n") {
		t.Error("lecture de la TABLE brute : une passe perimee ferait sauter un match a redecoder (ADR 0026)")
	}
}

func pourquoi(fragment string) string {
	switch fragment {
	case "read_path <> ?":
		return "sans lui, la base entiere passe pour a jour (elle est couverte par le credit-seul)"
	case "match_kill_events_latest":
		return "une lecture brute servirait des passes perimees (ADR 0026)"
	case "start_time_utc":
		return "l ordre est une course contre l expiration des films"
	default:
		return "condition retiree sans justification"
	}
}
