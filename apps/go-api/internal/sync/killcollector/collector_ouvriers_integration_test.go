//go:build integration

// Package killcollector — collector_ouvriers_integration_test.go : LA PREUVE QUE N OUVRIERS
// ECRIVENT LES MEMES LIGNES QU UN SEUL (lot 5.24.2, 2026-09-22).
//
// # LE CONTRAT DU LOT, EN UN TEST
//
// Le lot 5.24 change COMMENT la passe de backfill tourne, pas CE QU ELLE PRODUIT. Cette
// affirmation ne vaut rien tant qu elle n est pas mesuree : `collector.go` a porte pendant des
// mois un avertissement (« paralleliser deux films contaminerait les deux », globaux de paquet du
// decodeur) qui etait VRAI a l epoque et qui est devenu FAUX sans que rien ne le dise — la
// cloture M3 d ADR 0034 a depense le profil, le dernier reglage global a disparu au lot E.2.
// Raisonner a nouveau produirait le meme genre de verite perissable. **On compare les lignes.**
//
// DEUX BASES, MEMES FILMS, MEMES GRAINES. La premiere passe la boucle en serie
// (`CollectMatches`, le chemin de reference), la seconde la passe a N ouvriers derriere la porte
// de la base. Les quatre tables du chantier — journal des morts, tirs par arme, vies, contexte de
// mort — plus les positions sont relues PAR LEURS VUES `_latest` (ADR 0026 : une lecture brute
// servirait des lignes perimees) et comparees LIGNE A LIGNE, colonnes `decode_pass` et
// `written_at` exclues (l une est un tirage aleatoire par passe, l autre l horloge).
//
// ⚠ SANS `KILLSOURCE_FIXTURES`, IL SE SAUTE.
package killcollector

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/sync/haloclient"
)

// ouvriersDuTest : le nombre d ouvriers oppose a la boucle en serie.
const ouvriersDuTest = 3

// vuesComparees : les vues `_latest` que la passe de film peuple, et dont l egalite EST le
// contrat du lot.
//
// LES VUES, JAMAIS LES TABLES (ADR 0026) : les tables sont append-only et portent les passes
// precedentes ; c est ce que la lecture SERT qui doit etre identique, pas ce qu elle accumule.
var vuesComparees = []string{
	"match_kill_events_latest",
	"match_weapon_shots_latest",
	"match_lives_latest",
	"match_death_context_latest",
	"kill_positions_latest",
}

// filmsDuTestDEgalite : des films du bas du cout QUI ECRIVENT (cf. D1 (5.24) : les moins chers
// du cache ne produisent rien neuf fois sur dix). Une dizaine de secondes en serie.
var filmsDuTestDEgalite = []string{
	"ee90570b", "c0a82e88", "e157a672", "1a37bcc8",
	"30d3c047", "cf040013", "114b0040", "aa056037", "bf5ced1b",
}

// baseDeLEgalite prepare une base de test peuplee des memes films, et rend le collecteur cable
// comme la commande de production. `porte` non nil garde la base derriere son jeton.
func baseDeLEgalite(t *testing.T, films map[string][]haloclient.FilmChunk, porte *PorteDeLaBase,
	annuaire bool,
) (*sql.DB, *KillSourceCollector) {
	t.Helper()
	db := openSharedTestDB(t)
	cat := realMapQuantCatalog(t)
	for film, chunks := range films {
		decode, err := FilmOf(chunks)
		if err != nil {
			t.Fatalf("assemblage %s: %v", film, err)
		}
		inscrireFilmAuRegistre(t, db, film, decode)
	}
	partage := NewSharedRoster(db)
	if annuaire {
		partage = partage.AvecAnnuaireDePasse()
	}
	var roster KillSourceRoster = partage
	// LES NOMS DE CARTE SONT TRIES, ET CE N EST PAS COSMETIQUE. `allCatalogNames` parcourt une
	// MAP : l ordre des candidats change a chaque appel, et `entreeDeCatalogueParNom` retient le
	// PREMIER qui resout (`repli_carte_premier_nom_resolu`). Deux passes tireraient donc deux
	// entrees de catalogue differentes, donc deux dequantifications, donc deux jeux de positions
	// — un ecart imputable au DOUBLE DE TEST, pas aux ouvriers. En production le nom vient de
	// `match_registry` et il n y a rien a trier.
	noms := allCatalogNames(cat)
	sort.Strings(noms)
	var cartes = staticMapNames{names: noms}
	col := NewKillSourceCollector(
		&fakeFilmClient{chunks: films},
		porte.GarderLeRoster(roster),
		porte.GarderLeWriter(sharedWriter(db)),
		games.CapabilityMap{
			games.CapFilmKillSource:    games.CapSupported,
			games.CapFilmWeaponShots:   games.CapSupported,
			games.CapFilmKillPositions: games.CapSupported,
		}, 0).
		WithPositionCapture(porte.GarderLesCartes(cartes), cat)
	return db, col
}

// TestOuvriers_MemesLignesQuUnSeulOuvrier — LE GATE DU LOT.
func TestOuvriers_MemesLignesQuUnSeulOuvrier(t *testing.T) {
	films := chargerLesFilmsDuTest(t, filmsDuTestDEgalite)
	if len(films) < 2 {
		t.Skipf("il faut au moins deux films du cache pour opposer une serie a des ouvriers (%d trouve(s))", len(films))
	}
	ids := make([]string, 0, len(films))
	for id := range films {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ctx := context.Background()

	dbSerie, colSerie := baseDeLEgalite(t, films, nil, false)
	debut := time.Now()
	sumSerie := colSerie.CollectMatches(ctx, ids)
	tempsSerie := time.Since(debut)

	porte := NouvellePorteDeLaBase()
	dbOuvriers, colOuvriers := baseDeLEgalite(t, films, porte, false)
	debut = time.Now()
	sumOuvriers := colOuvriers.CollectMatchesOuvriers(ctx, ids, ouvriersDuTest)
	tempsOuvriers := time.Since(debut)

	t.Logf("%d films — 1 ouvrier : %s | %d ouvriers : %s | gain x%.2f",
		len(ids), tempsSerie.Round(time.Millisecond), ouvriersDuTest,
		tempsOuvriers.Round(time.Millisecond),
		float64(tempsSerie)/float64(max64(int64(tempsOuvriers), 1)))

	comparerSyntheses(t, sumSerie, sumOuvriers)
	for _, vue := range vuesComparees {
		comparerVue(t, dbSerie, dbOuvriers, vue)
	}
}

// TestAnnuaireDePasse_MemesLignesQueLaJointure — LE GATE DE L ANNUAIRE DE PASSE (5.24.2).
//
// Deux passes en SERIE, memes films, memes graines : l une resout les identites par la jointure
// `v_gamertag_lookup` PAR MATCH (le chemin historique), l autre par l annuaire charge UNE FOIS.
// Les cinq vues doivent servir les memes lignes — noms compris, puisque c est precisement la
// quantite que l annuaire change de source.
//
// CE QUE CE TEST PEUT LAISSER PASSER, ET C EST ECRIT DANS `AvecAnnuaireDePasse` : l ecart
// theorique connu (un `xuid:NNN` ecrit par la passe et relu par un match suivant) ne se produit
// que si un film ne nomme pas un joueur absent du roster du match. Les films du cache ne le
// declenchent pas ; le jour ou un corpus le declencherait, c est CE test qui le dirait, avec la
// ligne exacte.
func TestAnnuaireDePasse_MemesLignesQueLaJointure(t *testing.T) {
	films := chargerLesFilmsDuTest(t, filmsDuTestDEgalite)
	if len(films) == 0 {
		t.Skip("aucun film du cache")
	}
	ids := make([]string, 0, len(films))
	for id := range films {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ctx := context.Background()

	dbJointure, colJointure := baseDeLEgalite(t, films, nil, false)
	debut := time.Now()
	sumJointure := colJointure.CollectMatches(ctx, ids)
	tempsJointure := time.Since(debut)

	dbAnnuaire, colAnnuaire := baseDeLEgalite(t, films, nil, true)
	debut = time.Now()
	sumAnnuaire := colAnnuaire.CollectMatches(ctx, ids)
	tempsAnnuaire := time.Since(debut)

	t.Logf("%d films — jointure par match : %s | annuaire de passe : %s",
		len(ids), tempsJointure.Round(time.Millisecond), tempsAnnuaire.Round(time.Millisecond))
	comparerSyntheses(t, sumJointure, sumAnnuaire)
	for _, vue := range vuesComparees {
		comparerVue(t, dbJointure, dbAnnuaire, vue)
	}
}

// chargerLesFilmsDuTest lit les films presents, en sautant proprement les absents.
func chargerLesFilmsDuTest(t *testing.T, ids []string) map[string][]haloclient.FilmChunk {
	t.Helper()
	out := map[string][]haloclient.FilmChunk{}
	for _, id := range ids {
		if !filmPresent(id) {
			continue
		}
		if chunks := chargerFilmDeFixture(t, id); len(chunks) > 0 {
			out[id] = chunks
		}
	}
	return out
}

// comparerSyntheses : les compteurs de la passe, qui doivent etre les memes a l issue pres de
// l ordre (une passe a N ouvriers finit dans le desordre, elle ne finit pas autre chose).
func comparerSyntheses(t *testing.T, serie, ouvriers KillSourceSummary) {
	t.Helper()
	serie.ElapsedTime, ouvriers.ElapsedTime = 0, 0
	if serie != ouvriers {
		t.Errorf("syntheses differentes :\n  serie    %+v\n  ouvriers %+v", serie, ouvriers)
	}
	if serie.Written == 0 {
		t.Fatal("aucun film ecrit par la passe en serie : le test ne compare rien — " +
			"les films du cache ne produisent plus de journal des morts")
	}
}

// comparerVue relit UNE vue `_latest` des deux bases et compare ligne a ligne.
func comparerVue(t *testing.T, serie, ouvriers *sql.DB, vue string) {
	t.Helper()
	a := lignesDeLaVue(t, serie, vue)
	b := lignesDeLaVue(t, ouvriers, vue)
	if len(a) != len(b) {
		t.Errorf("%s : %d lignes en serie, %d a %d ouvriers", vue, len(a), ouvriersDuTest, len(b))
		return
	}
	if len(a) == 0 {
		t.Logf("%s : aucune ligne des deux cotes (rien a comparer)", vue)
		return
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("%s : ligne %d differente\n  serie    %s\n  ouvriers %s", vue, i, a[i], b[i])
			return
		}
	}
	t.Logf("%s : %d lignes IDENTIQUES", vue, len(a))
}

// colonnesSansIdentiteTechnique : les colonnes de la vue moins les TROIS qui ne portent aucun
// fait — et l omission de la premiere a fait echouer la premiere version de ce test.
//
//	id            la PK NON NATURELLE des tables append-only (ADR 0026), tiree d une sequence.
//	              Elle suit l ORDRE D INSERTION : une passe a N ouvriers finit les films dans le
//	              desordre, donc elle numerote autrement. Comparer `id` reviendrait a exiger que
//	              les ouvriers arrivent dans l ordre de depart, ce que ce lot ne promet pas.
//	decode_pass   un tirage aleatoire par passe (`newDecodePassID`).
//	written_at    l horloge.
//
// La liste est LUE AU SCHEMA (`DESCRIBE`), jamais recopiee : une colonne ajoutee a une des cinq
// tables entre donc d office dans la comparaison, au lieu d en sortir en silence.
func colonnesSansIdentiteTechnique(t *testing.T, db *sql.DB, vue string) []string {
	t.Helper()
	rows, err := db.Query("DESCRIBE " + vue)
	if err != nil {
		t.Fatalf("DESCRIBE %s: %v", vue, err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("DESCRIBE %s (colonnes): %v", vue, err)
	}
	horsFait := map[string]bool{"id": true, "decode_pass": true, "written_at": true}
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("DESCRIBE %s (scan): %v", vue, err)
		}
		nom := fmt.Sprintf("%v", vals[0])
		if !horsFait[nom] {
			out = append(out, `"`+nom+`"`)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("DESCRIBE %s (lignes): %v", vue, err)
	}
	if len(out) == 0 {
		t.Fatalf("%s : aucune colonne porteuse de fait — le schema a change", vue)
	}
	return out
}

// lignesDeLaVue rend les lignes de la vue, identite technique exclue, triees.
func lignesDeLaVue(t *testing.T, db *sql.DB, vue string) []string {
	t.Helper()
	colonnes := strings.Join(colonnesSansIdentiteTechnique(t, db, vue), ", ")
	rows, err := db.Query(fmt.Sprintf(`SELECT %s FROM %s ORDER BY ALL`, colonnes, vue))
	if err != nil {
		t.Fatalf("lecture de %s: %v", vue, err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("colonnes de %s: %v", vue, err)
	}
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan de %s: %v", vue, err)
		}
		var b strings.Builder
		for i, v := range vals {
			if i > 0 {
				b.WriteByte('|')
			}
			fmt.Fprintf(&b, "%v", v)
		}
		out = append(out, b.String())
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("lignes de %s: %v", vue, err)
	}
	return out
}

// TestOuvriers_UnSeulOuvrierEstLaBoucleEnSerie — le ratchet du chemin de reference.
//
// `CollectMatchesOuvriers(..., 1)` ne doit pas etre « une passe a un ouvrier qui ressemble a la
// boucle » : ce DOIT etre la boucle. Sans cette garantie, le temoin du test d egalite pourrait
// deriver de lui-meme.
func TestOuvriers_UnSeulOuvrierEstLaBoucleEnSerie(t *testing.T) {
	db := openSharedTestDB(t)
	_ = db
	col := NewKillSourceCollector(&fakeFilmClient{}, fakeRoster{}, sharedWriter(db),
		games.CapabilityMap{}, 0)
	for _, n := range []int{0, 1, -3} {
		sum := col.CollectMatchesOuvriers(context.Background(), []string{"a", "b"}, n)
		if sum.Total != 2 || sum.NotSupport != 2 {
			t.Errorf("ouvriers=%d : synthese %+v, attendu la boucle en serie (2 capability-absente)", n, sum)
		}
	}
}

// TestPorteDeLaBase_UnSeulJetonALaFois — la porte tient ce qu elle promet.
func TestPorteDeLaBase_UnSeulJetonALaFois(t *testing.T) {
	p := NouvellePorteDeLaBase()
	rendre, err := p.prendre(context.Background())
	if err != nil {
		t.Fatalf("premiere prise: %v", err)
	}
	ctx, annuler := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer annuler()
	if _, err := p.prendre(ctx); err == nil {
		t.Fatal("la porte a rendu DEUX jetons : l invariant d un seul ecrivain (ADR 0013) est perdu")
	}
	rendre()
	rendre() // idempotente : un second appel ne doit PAS fabriquer un jeton de plus
	// SANS DELAI POUR CELLE-CI, ET C EST UNE CORRECTION DE REVUE (2026-09-22) : le jeton est
	// DISPONIBLE, donc `prendre` ne doit pas attendre. Avec un `WithTimeout(50 ms)`, les deux
	// branches du `select` pouvaient etre pretes en meme temps si le goroutine de test etait
	// desordonnance, et Go en choisit une au hasard — un rouge intermittent sur le seul test de
	// la porte qui tourne en CI. Les deux prises qui DOIVENT echouer, elles, gardent leur delai.
	rendre2, err := p.prendre(context.Background())
	if err != nil {
		t.Fatalf("le jeton n a pas ete rendu: %v", err)
	}
	ctx3, annuler3 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer annuler3()
	if _, err := p.prendre(ctx3); err == nil {
		t.Fatal("un rendu double a fabrique un second jeton")
	}
	rendre2()
}

// TestPorteDeLaBase_NilEstUnPassePlat — les trois chemins qui ne parallelisent pas ne changent
// pas de comportement.
func TestPorteDeLaBase_NilEstUnPassePlat(t *testing.T) {
	var p *PorteDeLaBase
	fn := sharedWriter(nil)
	if got := p.GarderLeWriter(fn); got == nil {
		t.Error("porte nil : le lease doit passer tel quel")
	}
	var r KillSourceRoster = fakeRoster{}
	if got := p.GarderLeRoster(r); got == nil {
		t.Error("porte nil : le roster doit passer tel quel")
	}
	if got := p.GarderLesCartes(staticMapNames{}); got == nil {
		t.Error("porte nil : la resolution de carte doit passer tel quel")
	}
	rendre, err := p.prendre(context.Background())
	if err != nil || rendre == nil {
		t.Fatalf("porte nil : prendre() doit rendre un no-op sans erreur, err=%v rendre_nil=%t",
			err, rendre == nil)
	}
	rendre()
}

var _ = decfilm.Rev // la revision du decodeur ne bouge pas dans ce lot : le test la cite pour que
// tout changement de revision fasse relire ce fichier.

// TestArretDoux_LaBoucleEnSerieLHonoreAussi — le constat de revue du 2026-09-22 : l arret doux
// de la boucle EN SERIE (`roster.go`, le chemin de `--workers 1`) n etait couvert par AUCUN
// test, ni local ni CI. Le gate de reprise, lui, tourne a deux ouvriers.
//
// Sans ce chemin, `backfill-killsource --workers 1` devient ININTERRUPTIBLE : le contexte de
// travail est `context.WithoutCancel`, donc il n existe aucun autre point d arret, et il faut un
// SECOND signal — qui tue le film en vol, c est-a-dire exactement ce que l arret doux existe
// pour eviter.
func TestArretDoux_LaBoucleEnSerieLHonoreAussi(t *testing.T) {
	db := openSharedTestDB(t)
	client := &fakeFilmClient{}
	col := NewKillSourceCollector(client, fakeRoster{}, sharedWriter(db), capsAvecFilm(), 0)

	// Temoin : sans arret, les trois matchs sont examines (le client ne rend aucun film, donc
	// trois `film-absent` — ce qui compte est que la boucle les ait PRIS).
	sum := col.CollectMatches(context.Background(), []string{"a", "b", "c"})
	if sum.NoFilm != 3 {
		t.Fatalf("temoin : %d films examines, attendu 3 (%+v)", sum.NoFilm, sum)
	}
	appelsSansArret := client.calls

	arret, annuler := context.WithCancel(context.Background())
	annuler()
	sum = col.AvecArretDoux(arret).CollectMatches(context.Background(), []string{"a", "b", "c"})
	if sum.NoFilm != 0 {
		t.Errorf("%d films examines malgre l arret demande, attendu 0 : la boucle en serie "+
			"n honore pas l arret doux et `--workers 1` serait ininterruptible", sum.NoFilm)
	}
	if client.calls != appelsSansArret {
		t.Errorf("la source de films a ete interrogee %d fois de plus apres l arret",
			client.calls-appelsSansArret)
	}
	if sum.Total != 3 {
		t.Errorf("total = %d, attendu 3 : la synthese doit annoncer ce qu on lui a demande, "+
			"pas ce qu elle a fait", sum.Total)
	}
}
