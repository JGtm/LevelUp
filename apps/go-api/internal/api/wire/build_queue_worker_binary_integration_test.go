//go:build integration && cgo

// Package api — build_queue_worker_binary_integration_test.go : LA PREUVE AVEC UN VRAI
// OUVRIER, SUR UN MINI-FILM VERSIONNÉ — ET ELLE TOURNE EN CI.
//
// Le test de transport voisin prouve le protocole avec un artefact fabriqué à la main.
// Celui-ci va au bout de la chaîne telle qu'elle tournera : le BINAIRE cmd/replay-worker
// est compilé et lancé, il prend un job par HTTP, télécharge les morceaux depuis des URL
// « pré-signées », DÉCODE UN VRAI FILM, pousse l'artefact, et le serveur le range — où le
// service de rejeu le lit.
//
// CE QUI A CHANGÉ, ET POURQUOI. Cette preuve se SAUTAIT en CI : elle lisait le film 000d5950
// (49 morceaux, 22 Mo) du cache utilisateur, gitignoré donc absent de CI ; et elle fabriquait
// le job À LA MAIN, contournant EnqueueReplayBuild. Désormais :
//
//  1. Le film est un FIXTURE VERSIONNÉ (testdata/film_e2e/c0a82e88) — le PLUS PETIT du corpus
//     à joueurs (Husky Raid:CTF, 8 morceaux, ~1,6 Mio compressé zlib, la forme même que sert
//     le CDN Azure). Résolu par le paquet, jamais par un cache utilisateur : plus de saut.
//  2. La mise en file passe par EnqueueReplayBuild RÉEL. Les deux SEULES frontières d'E/S sont
//     injectées par les seams nil-en-production : la résolution du manifeste Halo (remplacée
//     par un httptest servant les morceaux du fixture) et la lecture des faits (CI n'a pas de
//     shared DuckDB — les faits sont ceux, RÉELS, capturés du corpus et versionnés). Tout le
//     reste — assemblage du payload, enfilage, protocole ouvrier, décodage, rangement — est réel.
//
// CE QUI EST ISOLÉ : l'ouvrier reçoit un dépôt À LUI (copie des seules références versionnées :
// bornes, géométrie, structures, objectifs, libellés) et un dossier de travail temporaire. Rien
// n'est écrit dans le dépôt de test — surtout pas ses artefacts.
//
// COÛT : un décodage de film COMPLET mais MINIMAL (8 morceaux). D'où le tag integration ; le job
// CI go-coverage (CGO + integration) l'exécute.
//
// CORRECTIF DU 2026-09-05 — PLUS DE COPIE LOCALE CHEZ L'OUVRIER À RELIRE. Ce test comparait
// jusqu'ici l'artefact rangé par le serveur à une copie que l'ouvrier aurait écrite dans SON
// dépôt (`ouvrierRepo`) — une hypothèse d'architecture PÉRIMÉE : le lot 5 de PLAN_CUISSON_PERF
// (D8) a fait de l'ouvrier un pur producteur d'OCTETS envoyés en mémoire (`built.Blob` →
// `sendArtifact`), qui ne garde et n'écrit RIEN localement (cf.
// `TestOuvrier_NeComposeJamaisLEcritureDArtefact` et `TestBuildAndSend_NEcritAucunArtefactLocal`
// dans cmd/replay-worker) — seul le serveur range (`replaybuild.StoreArtifact`). La preuve
// d'identité octet à octet passe désormais par l'EMPREINTE sha256 que l'ouvrier calcule sur
// `built.Blob` avant l'envoi et qu'il porte dans le compte rendu de son job (`ResultJSON`,
// champ `sha256` — cf. job.go, buildAndSend) : cette valeur survit à son processus, contrairement
// à ses octets. Voir `assertArtefactLivreEtComplet`.
package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	syncpkg "levelup/go-api/internal/sync"
)

// fixtureShort : le film de la preuve — le plus petit du corpus qui rend un artefact À JOUEURS.
const fixtureShort = "c0a82e88"

// filmFixture : le mini-film versionné, chargé depuis testdata.
type filmFixture struct {
	MatchIDFull string
	ShortID     string
	MapNames    []string
	Facts       port.MatchFacts
	Chunks      []fixtureChunk
	chunksDir   string
}

type fixtureChunk struct {
	Index      int
	ChunkType  int
	StartMS    int
	DurationMS int
	File       string
}

// TestOuvrierReel_ConstruitEtLivre : la chaîne complète, avec le vrai binaire, sur le fixture.
func TestOuvrierReel_ConstruitEtLivre(t *testing.T) {
	fx := chargerFixture(t)

	// ── Le CDN : les morceaux du fixture, déjà zlib (servis tels quels, sans auth) ───────
	cdn := serveurDeMorceaux(t, fx)

	// ── Le web : les vraies routes, un dépôt vierge, une vraie base ──────────────────────
	srv, reg, serveurRepo := transportStack(t)

	// ── Les deux SEULES frontières injectées : manifeste Halo (→ CDN) et faits (→ fixture) ─
	reg.replayFilmResolver = &stubFilmResolver{found: true, refs: fx.chunkRefs(cdn.URL)}
	reg.replayJobFactsFn = fx.factsFn()

	// ── LA MISE EN FILE RÉELLE : EnqueueReplayBuild assemble le payload (faits + URLs) et
	//    enfile. Le job n'est PLUS fabriqué à la main — c'est le contournement corrigé. ─────
	job, created, err := reg.EnqueueReplayBuild(context.Background(), titlePkg.DefaultSlug, fixtureShort)
	if err != nil || !created {
		t.Fatalf("EnqueueReplayBuild (mise en file réelle): err=%v created=%v", err, created)
	}

	// ── L'ouvrier : son propre dépôt (références versionnées du dépôt CI), son dossier ───
	ouvrierRepo := depotOuvrier(t, repoRootDepuisTest(t))
	travail := t.TempDir()
	binaire := compilerOuvrier(t)

	debut := time.Now()
	lancerOuvrier(t, binaire, srv.URL+"/internal", ouvrierRepo, travail)
	t.Logf("DÉCODAGE + LIVRAISON en %s (fixture %s, 8 morceaux)", time.Since(debut).Round(time.Millisecond), fixtureShort)

	// ── Le job est `succeeded` (donc le compte rendu a trouvé le fichier) — et c'est CE
	//    compte rendu qui porte l'empreinte utilisée juste après. ────────────────────────
	vue, err := reg.monitoringStore.BuildQueueReport(context.Background(), 10)
	if err != nil {
		t.Fatalf("BuildQueueReport: %v", err)
	}
	if vue.Counts.Succeeded != 1 {
		t.Fatalf("file après passage de l'ouvrier : %+v, attendu 1 fait (job %s)", vue.Counts, job.JobID)
	}

	// ── L'artefact rangé = celui construit (empreinte sha256 déclarée par l'ouvrier dans son
	//    compte rendu), lisible par le service, et NON APPAUVRI ─────────────────────────────
	assertArtefactLivreEtComplet(t, serveurRepo, vue.Jobs, job.JobID, fx)

	// ── L'ouvrier n'a rien gardé : ses morceaux sont effacés ─────────────────────────────
	if _, err := os.Stat(filepath.Join(travail, "film_chunks", fixtureShort)); !os.IsNotExist(err) {
		t.Errorf("l'ouvrier a conservé ses morceaux de film (%v) — il ne doit rien garder", err)
	}
}

// assertArtefactLivreEtComplet vérifie que l'artefact rangé par le web est À L'OCTET celui
// construit par l'ouvrier, lisible par le service de rejeu, et NON APPAUVRI.
//
// L'OUVRIER NE GARDE NI N'ÉCRIT AUCUNE COPIE LOCALE (PLAN_CUISSON_PERF §3 D8 — il envoie des
// OCTETS en mémoire, `built.Blob`, jamais un fichier ; garde-rail
// `TestOuvrier_NeComposeJamaisLEcritureDArtefact`, cmd/replay-worker) : il n'existe donc PLUS de
// double locale à relire pour prouver l'identité octet à octet. La preuve passe par l'EMPREINTE
// sha256 que l'ouvrier calcule lui-même sur ses octets, AVANT l'envoi, et qu'il porte dans le
// compte rendu de son job (`ResultJSON`, champ `sha256` — cf. job.go, buildAndSend) : cette
// valeur survit à son processus quand ses octets ne survivent pas. `jobs` est la liste rendue
// par `BuildQueueReport` (lue APRÈS le passage de l'ouvrier) ; `jobID` désigne celui qu'il vient
// de traiter.
//
// C'EST LE CRITÈRE DE SUCCÈS DU CHANTIER. Un ouvrier qui décode SANS les faits rend un artefact
// APPAUVRI (mesuré : 0 joueur de courbe de score, camps `unresolved`) qui porte pourtant le bon
// numéro de schéma. La présence de compteurs de joueur est LA ligne qui distingue « livré » de
// « livré vide » — exactement l'appauvrissement que le transport des faits (via EnqueueReplayBuild)
// doit supprimer. Ici, AVEC faits : 5 joueurs de courbe de score et 12 actions d'objectif publiées
// (8 `kills`, 4 `assists`), contre 0 sans.
//
// CE QUE « 92 » ÉTAIT, ET POURQUOI CETTE LIGNE A MENTI HUIT JOURS. Elle a dit jusqu'au 2026-09-06
// « 92 actions d'objectif nommées (famille flag) ». Le 92 est bien réel, mais c'est le compteur de
// JOURNAL `nommees` — les émissions d'emplacement de statistique que `objectiveevents.NamedEvents`
// reconnaît, AVANT le pont d'identité. Le document, lui, n'en portait alors AUCUNE (le calque
// `objectives` était vide en production à cette date). Confondre le compteur amont avec le contenu
// publié a fait passer pour une régression, le 2026-09-05, un chiffre qui n'avait jamais été celui
// du document. Mesure sur pièces (2026-09-06, films complet ET fixture, schémas 38 et 39
// confondus) : 92 nommées, 12 identifiées, 12 publiées.
//
// LES VALEURS SONT VÉRIFIÉES ICI, contre l'oracle INDÉPENDANT du fixture (la feuille de match de
// l'API, `facts`) quand il existe — cf. `assertCalquesDObjectif`.
func assertArtefactLivreEtComplet(t *testing.T, serveurRepo string, jobs []domain.BuildQueueJob,
	jobID string, fx filmFixture,
) {
	t.Helper()
	var resultJSON string
	trouve := false
	for _, j := range jobs {
		if j.JobID == jobID {
			resultJSON, trouve = j.ResultJSON, true
			break
		}
	}
	if !trouve {
		t.Fatalf("job %s absent du rapport de file — impossible de lire son compte rendu", jobID)
	}
	var rendu struct {
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal([]byte(resultJSON), &rendu); err != nil {
		t.Fatalf("compte rendu du job illisible (%q): %v", resultJSON, err)
	}
	if rendu.SHA256 == "" {
		t.Fatalf("compte rendu du job sans empreinte sha256 (%q)", resultJSON)
	}

	// Le chemin canonique est TOUJOURS la forme courte (ReplayArtifactPath normalise).
	recu := filepath.Join(serveurRepo, "data", "cache", "replays", titlePkg.DefaultSlug, fixtureShort+".json")
	blobRecu, err := os.ReadFile(recu)
	if err != nil {
		t.Fatalf("aucun artefact rangé côté serveur (%s): %v", recu, err)
	}
	empreinteRecue := sha256.Sum256(blobRecu)
	if hex.EncodeToString(empreinteRecue[:]) != rendu.SHA256 {
		t.Fatalf("empreinte de l'artefact rangé (%d octets) ≠ empreinte déclarée par l'ouvrier dans son compte rendu (%s)",
			len(blobRecu), rendu.SHA256)
	}

	doc, err := service.NewReplayService(titlePkg.DefaultSlug, serveurRepo, nil).
		GetReplay(context.Background(), fixtureShort)
	if err != nil {
		t.Fatalf("le service de rejeu ne lit pas l'artefact livré: %v", err)
	}
	if doc.SchemaVersion != replay.SchemaVersion || len(doc.Tracks) == 0 {
		t.Fatalf("document servi : schéma %d (veut %d), %d trajectoires", doc.SchemaVersion, replay.SchemaVersion, len(doc.Tracks))
	}
	t.Logf("artefact livré : %d octets, %d trajectoires, %d frames", len(blobRecu), len(doc.Tracks), doc.FrameCount)

	if doc.ScoreTimeline == nil || len(doc.ScoreTimeline.Players) == 0 {
		t.Fatal("artefact livré SANS compteurs de joueur — les faits du job n'ont pas été utilisés " +
			"(appauvrissement que le transport des faits doit supprimer)")
	}
	if doc.Coverage == nil || doc.Coverage.Score == nil {
		t.Fatal("artefact livré sans couverture de score : impossible de dire ce que vaut la courbe")
	}
	// L'IDENTITÉ DES CAMPS EST INFORMATIVE, PAS UN CRITÈRE. Sur ce film Husky Raid:CTF elle reste
	// `unresolved` (le décodeur ne rattache qu'UN camp aux slots d'entité de ce mode, et le signale
	// proprement) — ce n'est PAS l'appauvrissement (qui est l'ABSENCE de joueurs), c'est une
	// propriété de ce film. La ligne dure ci-dessus (joueurs présents) porte la preuve.
	t.Logf("artefact COMPLET : %d joueurs de courbe de score, identité des camps = %q (informatif)",
		len(doc.ScoreTimeline.Players), doc.Coverage.Score.TeamIdentity)

	assertCalquesDObjectif(t, doc, fx)
}

// assertCalquesDObjectif confronte les calques d'objectif à l'ORACLE INDÉPENDANT du fixture — la
// feuille de match de l'API, que le décodeur ne lit jamais.
//
// CE QU'IL ASSERTE, ET POURQUOI CHAQUE LIGNE EST JUSTIFIABLE :
//
//	les CAPTURES        `coverage.flagCarries.captures` compte les captures des DEUX camps ; son
//	                    oracle est donc la SOMME des scores d'équipe, pas le score du gagnant.
//	                    L'égalité avec `TeamScores[0]` seul ne tenait sur ce fixture que parce que
//	                    le perdant a marqué 0 — invariant arithmétiquement faux, corrigé le
//	                    2026-09-06 (revue CTF-R1, constat 4).
//	les COMPTEURS       chaque joueur que le pont d'identité nomme publie ses actions `kills` et
//	                    `assists` ; elles ne peuvent pas dépasser ce que la feuille de match lui
//	                    donne. C'est vrai QUELS QUE SOIENT les joueurs pontés, donc c'est un
//	                    invariant et non une épingle.
//	les VIES DE L'OBJET `objectLives` = 4 : épingle de caractérisation, mesurée identique sur le
//	                    film complet du cache aux schémas 20, 38 et 39. Elle n'est PAS la
//	                    « signature du registre ECS » que la première version de ce commentaire
//	                    annonçait : la revue CTF-R1 a montré qu'elle vaut 4 même avec le fixture
//	                    mal généré (la chaîne E2E pelait ses deux couches). Le seul garde-rail qui
//	                    attrape ce défaut-là est `film_fixture_integrite_cgo_test.go`.
//
// CE QUI N'EST DÉLIBÉRÉMENT PAS FIGÉ ICI, ET LE RESTERA TANT QUE CE NE SERA PAS INSTRUIT. Le
// COMPTE d'actions par famille (12 : 8 `kills` + 4 `assists`) et la LISTE des joueurs pontés ne
// sont PAS des attentes de ce test. L'artefact du parc pour ce même match (schéma 20) porte
// 17 actions, dont un `flag_captures` et un `flag_steals` qui ont disparu depuis, et un ensemble
// de joueurs pontés DIFFÉRENT. Figer l'état d'aujourd'hui sanctifierait une dérive non expliquée
// (revue CTF-R1, constat 3) — un test ne sanctuarise pas ce qu'on n'a pas compris.
func assertCalquesDObjectif(t *testing.T, doc replay.ReplayDocument, fx filmFixture) {
	t.Helper()
	fc := doc.Coverage.FlagCarries
	if fc == nil {
		t.Fatal("artefact sans couverture du drapeau : le film est un CTF, le calque doit se prononcer")
	}
	if !fc.FlagFilm {
		t.Fatalf("film NON reconnu comme CTF (bursts=%d captures=%d steals=%d)", fc.Bursts, fc.Captures, fc.Steals)
	}
	if fx.Facts.TeamScores == nil {
		t.Fatal("fixture sans scores d'équipe : plus d'oracle pour les captures")
	}
	// ORACLE INDÉPENDANT : les captures des deux camps réunies valent la somme des scores.
	attendu := fx.Facts.TeamScores[0] + fx.Facts.TeamScores[1]
	if fc.Captures != attendu || fc.Bursts != attendu {
		t.Errorf("captures reconstruites = %d (bursts %d), la feuille de match en donne %d "+
			"(%d + %d) — le décodage du film et l'API doivent tomber d'accord",
			fc.Captures, fc.Bursts, attendu, fx.Facts.TeamScores[0], fx.Facts.TeamScores[1])
	}
	// ÉPINGLE DE CARACTÉRISATION (pas un oracle) : les vies libres de l'objet drapeau.
	if fc.ObjectLives != 4 {
		t.Errorf("vies libres de l'objet drapeau = %d, attendu 4 (mesure du 2026-09-06 sur le film "+
			"complet du cache, schémas 20, 38 et 39 confondus)", fc.ObjectLives)
	}
	// ORACLE INDÉPENDANT : aucun joueur ne peut publier plus d'actions que la feuille ne lui en
	// donne, et toute action publiée appartient à un joueur du match.
	feuille := map[string]port.MatchPlayerFact{}
	for _, p := range fx.Facts.Players {
		feuille[p.XUID] = p
	}
	parJoueur := map[string]map[string]int{}
	for _, a := range doc.Objectives {
		if _, connu := feuille[a.XUID]; !connu {
			t.Errorf("action d'objectif `%s` attribuée au xuid %s, absent de la feuille de match", a.Stat, a.XUID)
			continue
		}
		if parJoueur[a.XUID] == nil {
			parJoueur[a.XUID] = map[string]int{}
		}
		parJoueur[a.XUID][a.Stat]++
	}
	for xuid, actions := range parJoueur {
		p := feuille[xuid]
		if actions["kills"] > p.Kills {
			t.Errorf("joueur %s : %d actions `kills` publiées pour %d frags à la feuille de match",
				xuid, actions["kills"], p.Kills)
		}
		if actions["assists"] > p.Assists {
			t.Errorf("joueur %s : %d actions `assists` publiées pour %d assistances à la feuille de match",
				xuid, actions["assists"], p.Assists)
		}
	}
	if len(parJoueur) == 0 {
		t.Error("aucune action d'objectif attribuée : le pont d'identité n'a nommé personne")
	}
	t.Logf("calque objectifs : %d actions, %d joueurs pontés (compte NON figé — cf. l'en-tête)",
		len(doc.Objectives), len(parJoueur))
}

// chargerFixture lit le mini-film versionné (fixture.json + chunks) résolu PAR LE PAQUET
// (runtime.Caller), jamais par un cache utilisateur — c'est ce qui le rend exécutable en CI.
func chargerFixture(t *testing.T) filmFixture {
	t.Helper()
	dir := filepath.Join(fixtureDir(t), fixtureShort)
	raw, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
	if err != nil {
		t.Fatalf("fixture.json introuvable (%s): %v — le fixture VERSIONNÉ doit être présent en CI", dir, err)
	}
	var f struct {
		MatchIDFull string   `json:"matchIdFull"`
		ShortID     string   `json:"shortId"`
		MapNames    []string `json:"mapNames"`
		Facts       struct {
			GameVariantName string `json:"gameVariantName"`
			TeamScores      [2]int `json:"teamScores"`
			MapID           string `json:"mapId"`
			Players         []struct {
				XUID    string `json:"xuid"`
				Kills   int    `json:"kills"`
				Deaths  int    `json:"deaths"`
				Assists int    `json:"assists"`
				TeamID  int    `json:"teamId"`
			} `json:"players"`
		} `json:"facts"`
		Chunks []struct {
			Index      int    `json:"index"`
			ChunkType  int    `json:"chunkType"`
			StartMS    int    `json:"startMs"`
			DurationMS int    `json:"durationMs"`
			File       string `json:"file"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("fixture.json illisible: %v", err)
	}
	scores := f.Facts.TeamScores
	facts := port.MatchFacts{
		GameVariantName: f.Facts.GameVariantName,
		TeamScores:      &scores,
		MapID:           f.Facts.MapID,
	}
	for _, p := range f.Facts.Players {
		facts.Players = append(facts.Players, port.MatchPlayerFact{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists, TeamID: p.TeamID,
		})
	}
	fx := filmFixture{
		MatchIDFull: f.MatchIDFull, ShortID: f.ShortID, MapNames: f.MapNames,
		Facts: facts, chunksDir: filepath.Join(dir, "chunks"),
	}
	for _, c := range f.Chunks {
		fx.Chunks = append(fx.Chunks, fixtureChunk{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS, File: c.File,
		})
	}
	if len(fx.Chunks) == 0 || len(fx.Facts.Players) == 0 {
		t.Fatalf("fixture dégénéré : %d morceaux, %d joueurs", len(fx.Chunks), len(fx.Facts.Players))
	}
	return fx
}

// chunkRefs fabrique les références de manifeste que la frontière Halo rendrait : chaque
// morceau pointé par une URL pré-signée vers le CDN httptest.
func (fx filmFixture) chunkRefs(cdnURL string) []syncpkg.FilmChunkRef {
	out := make([]syncpkg.FilmChunkRef, 0, len(fx.Chunks))
	for _, c := range fx.Chunks {
		out = append(out, syncpkg.FilmChunkRef{
			Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS, DurationMS: c.DurationMS,
			URL: cdnURL + "/" + c.File,
		})
	}
	return out
}

// factsFn rend le seam de lecture des faits : identité complète + noms de carte + faits, tels
// que EnqueueReplayBuild les lirait en base — mais servis par le fixture (CI n'a pas de shared).
func (fx filmFixture) factsFn() func(context.Context, string, string) (string, []string, port.MatchFacts, error) {
	return func(context.Context, string, string) (string, []string, port.MatchFacts, error) {
		return fx.MatchIDFull, fx.MapNames, fx.Facts, nil
	}
}

// serveurDeMorceaux imite le CDN Azure : il sert les morceaux du fixture, DÉJÀ compressés en
// zlib, sans authentification — c'est exactement ce que l'ouvrier attend d'une URL pré-signée,
// et ça vérifie au passage sa décompression.
func serveurDeMorceaux(t *testing.T, fx filmFixture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nom := filepath.Base(r.URL.Path)
		blob, err := os.ReadFile(filepath.Join(fx.chunksDir, nom))
		if err != nil {
			http.Error(w, "morceau absent", http.StatusNotFound)
			return
		}
		_, _ = w.Write(blob) // déjà zlib : servi tel quel
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fixtureDir rend le dossier testdata/film_e2e, résolu relativement au paquet.
func fixtureDir(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("chemin du test introuvable")
	}
	return filepath.Join(filepath.Dir(ici), "testdata", "film_e2e")
}

// repoRootDepuisTest rend la racine du dépôt (5 niveaux au-dessus du paquet), d'où l'ouvrier
// copie ses références VERSIONNÉES — jamais un cache utilisateur.
func repoRootDepuisTest(t *testing.T) string {
	t.Helper()
	_, ici, _, _ := runtime.Caller(0)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(ici), "..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("racine du dépôt introuvable: %v", err)
	}
	return root
}

// depotOuvrier fabrique un dépôt À LUI : uniquement les références versionnées que replaybuild
// charge (bornes, géométrie, structures, objectifs de carte, libellés). Copie et non lien :
// l'ouvrier ne doit pas pouvoir toucher au dépôt de test.
func depotOuvrier(t *testing.T, depot string) string {
	t.Helper()
	dst := t.TempDir()
	for _, rel := range []string{
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_quant_bounds.json"),
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_objectives.json"),
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_geometry"),
		filepath.Join("data", "titles", titlePkg.DefaultSlug, "reference", "map_structure"),
		filepath.Join("config", "titles", titlePkg.DefaultSlug, "mappings"),
	} {
		if err := copierArborescence(filepath.Join(depot, rel), filepath.Join(dst, rel)); err != nil {
			t.Fatalf("référence %s indisponible pour l'ouvrier (doit être versionnée): %v", rel, err)
		}
	}
	return dst
}

// copierArborescence copie un fichier ou un répertoire (récursif).
func copierArborescence(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		blob, rerr := os.ReadFile(src)
		if rerr != nil {
			return rerr
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, blob, 0o644)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if err := copierArborescence(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// compilerOuvrier construit le binaire de l'ouvrier. Compilé AVANT tout décodage (jamais
// pendant : deux travaux lourds en parallèle sur la même machine, c'est la leçon des gels).
func compilerOuvrier(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "replay-worker")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/replay-worker")
	cmd.Dir = racineGoAPI(t)
	if sortie, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation de l'ouvrier: %v\n%s", err, sortie)
	}
	return out
}

// racineGoAPI rend apps/go-api (le module Go).
func racineGoAPI(t *testing.T) string {
	t.Helper()
	_, ici, _, _ := runtime.Caller(0)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(ici), "..", "..", ".."))
	if err != nil {
		t.Fatalf("racine du module: %v", err)
	}
	return root
}

// lancerOuvrier exécute le binaire en mode --once : il prend UN job, le traite, et sort.
func lancerOuvrier(t *testing.T, binaire, url, repo, travail string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaire,
		"--once",
		"--url", url,
		"--token", transportToken,
		"--id", "ouvrier-preuve",
		"--repo", repo,
		"--work", travail)
	sortie, err := cmd.CombinedOutput()
	t.Logf("journal de l'ouvrier :\n%s", sortie)
	if err != nil {
		t.Fatalf("l'ouvrier s'est arrêté sur erreur: %v", err)
	}
}
