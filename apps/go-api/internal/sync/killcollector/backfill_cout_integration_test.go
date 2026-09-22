//go:build integration

// Package killcollector — backfill_cout_integration_test.go : OU VA LE TEMPS D UN FILM
// (lot 5.24.1, 2026-09-22).
//
// # LA QUESTION, ET POURQUOI ELLE PRECEDE TOUT CORRECTIF
//
// La passe des films du 2026-09-21 a decode 1 598 films en 4 h 14 — ~9,5 s par film. Le brief
// du lot propose des OUVRIERS de decodage ; c est une reponse qui ne vaut que si le temps part
// REELLEMENT dans le decodage. S il partait dans le chargement des chunks, dans l ecriture ou
// dans une lecture de base repetee par match, le parallelisme ne ferait que deplacer la file.
// Cette mesure tranche, et elle tranche AVANT que le code ne change.
//
// # CE QU ELLE MESURE, ET SUR QUOI
//
// Les SEPT etapes d un match, chacune par UN appel a la fonction de production (le test vit
// dans le paquet : il n en recopie aucune) :
//
//	chargement     FilmChunksForMatch   les chunks du cache disque (jonction), octets bruts
//	chargement     FilmOf               decompression + decoupage (le pic memoire est ici)
//	decodage       decfilm.Decode       la passe chere
//	lecture        carteDuMatch         match_registry + catalogue de bornes
//	lecture        IdentitiesForMatch   le roster — la jointure `v_gamertag_lookup` PAR MATCH (D1 du 5.12)
//	ecriture       write                fusion credit + persist (morts)
//	ecriture       collectShots/Positions/Hits  les trois passes best-effort
//
// LA BASE EST UN BANC, PAS UN JOUET : `peuplerBancCredit` y met 180 000 lignes de
// `match_kill_events` et 50 000 couples, l ordre de grandeur d une campagne de production. Sans
// cela la jointure d identite couterait des microsecondes et la mesure mentirait par omission.
//
// LE PIC MEMOIRE est releve apres chaque film (`runtime.ReadMemStats`, HeapInuse et Sys) : c est
// lui qui decide combien d ouvriers tiennent sous le plafond, et l en-tete de la commande
// affirme « largement sous le gibioctet » — une affirmation de 2026-08-24 qu il faut re-verifier
// avant de la multiplier par N.
//
// ⚠ SANS `KILLSOURCE_FIXTURES`, IL SE SAUTE (les films ne sont pas versionnes) :
//
//	KILLSOURCE_FIXTURES=<racine>/data/cache/film_chunks \
//	  go test -count=1 -tags=integration -p 1 -timeout 3600s \
//	  -run PasseDesFilms_OuVaLeTemps -v ./internal/sync/killcollector/
package killcollector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/sync/haloclient"
)

// echantillonParDefaut : L ECHANTILLON DU LOT — des films PRODUCTIFS du bas du cout, deux films
// de cout median, et le PLUS GROS du corpus.
//
// IL EST ECRIT ICI ET PAS TIRE DU CACHE A CHAQUE FOIS, pour une raison : une mesure dont
// l echantillon change d une execution a l autre ne se compare pas a elle-meme. Les comptes de
// chunks sont ceux releves le 2026-09-22 sur les 1 612 films du cache (somme 48 791 chunks,
// moyenne 30,3). Un film absent de la machine est SAUTE, nomme — jamais une panne.
//
// ⚠ CE NE SONT PAS LES DIX MOINS CHERS, ET LA MESURE DIT POURQUOI. Le brief demandait « les 10
// films les moins chers » ; sur ce cache ils comptent 2 a 5 chunks et NEUF sur DIX ne produisent
// rien — cinq n ont aucune identite au fil des morts, trois n ont aucun chunk HIGHLIGHT
// (`ErrNoKillFeed`). Mesurer dessus, c est mesurer le cout de refuser un film. L echantillon
// retenu part donc de 5 chunks mais ne garde que ce qui ECRIT, ce qui est la seule population
// dont la passe de backfill paie le cout.
//
// `KILLSOURCE_ECHANTILLON` (liste separee par des virgules) remplace la liste, pour la machine
// qui n a pas ce cache-la.
var echantillonParDefaut = []string{
	// le bas du cout, films productifs (5 a 11 chunks)
	"4555ce28", "ee90570b", "c0a82e88", "e157a672", "1a37bcc8",
	"30d3c047", "cf040013", "114b0040", "aa056037", "bf5ced1b",
	// deux films au cout median (29 chunks)
	"e624c2a4", "e85d7bad",
	// le plus gros du corpus : 69 chunks, 92,2 Mio sur disque
	"1c4c63c2",
}

// echantillonDuLot rend les films a mesurer, et se saute proprement s il n y a pas de cache.
func echantillonDuLot(t *testing.T) []string {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("KILLSOURCE_ECHANTILLON")); v != "" {
		return strings.Split(v, ",")
	}
	if os.Getenv("KILLSOURCE_FIXTURES") == "" {
		t.Skip("KILLSOURCE_FIXTURES absent : les films ne sont pas versionnes. " +
			"KILLSOURCE_FIXTURES=<racine>/data/cache/film_chunks go test -tags=integration -p 1 ...")
	}
	return echantillonParDefaut
}

// coutDUnFilm : la decomposition d un match, en nanosecondes de chaque etape.
type coutDUnFilm struct {
	film       string
	chunks     int
	octets     int64
	chargement time.Duration // FilmChunksForMatch
	assemblage time.Duration // FilmOf
	carte      time.Duration // carteDuMatch
	decodage   time.Duration // decfilm.Decode
	roster     time.Duration // IdentitiesForMatch
	ecriture   time.Duration // write (fusion + persist des morts)
	annexes    time.Duration // shots + positions + hits
	heapInuse  uint64
	sys        uint64
}

func (c coutDUnFilm) total() time.Duration {
	return c.chargement + c.assemblage + c.carte + c.decodage + c.roster + c.ecriture + c.annexes
}

// collecteurDeMesure construit un collecteur cable comme la commande de production : capture des
// positions active, catalogue de bornes REEL, roster REEL branche sur la base du banc.
func collecteurDeMesure(t *testing.T, db *sql.DB, chunks map[string][]haloclient.FilmChunk,
) *KillSourceCollector {
	t.Helper()
	cat := realMapQuantCatalog(t)
	return NewKillSourceCollector(
		&fakeFilmClient{chunks: chunks},
		NewSharedRoster(db),
		sharedWriter(db),
		games.CapabilityMap{
			games.CapFilmKillSource:    games.CapSupported,
			games.CapFilmWeaponShots:   games.CapSupported,
			games.CapFilmKillPositions: games.CapSupported,
		}, 0).
		WithPositionCapture(staticMapNames{names: allCatalogNames(cat)}, cat)
}

// inscrireFilmAuRegistre pose le match au registre et ses participants, tires DU FILM LUI-MEME
// (`replay.ScanDeaths`) : les xuids que la passe va resoudre sont alors les VRAIS, et la
// jointure `v_gamertag_lookup` du roster a de quoi travailler.
func inscrireFilmAuRegistre(t *testing.T, db *sql.DB, film string, decode *decfilm.Film) int {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, map_name) VALUES (?, ?)`,
		film, "streets"); err != nil {
		t.Fatalf("registre %s: %v", film, err)
	}
	deaths, err := replay.ScanDeaths(decode)
	if err != nil {
		return 0
	}
	vus := map[uint64]bool{}
	n := 0
	for _, d := range deaths {
		if d.XUID == 0 || vus[d.XUID] {
			continue
		}
		vus[d.XUID] = true
		x := strconv.FormatUint(d.XUID, 10)
		if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag, team_id)
			VALUES (?, ?, ?, ?)`, film, x, d.Gamertag, n%2); err != nil {
			t.Fatalf("participants %s: %v", film, err)
		}
		// La vue canonique d identite lit `killer_victim_pairs` : sans une ligne par joueur,
		// aucun nom ne remonte et la jointure du roster rendrait une table vide.
		if _, err := db.Exec(`INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
			VALUES (?, ?, ?, ?, ?)`, film, x, d.Gamertag, x, d.Gamertag); err != nil {
			t.Fatalf("couples %s: %v", film, err)
		}
		n++
	}
	return n
}

// TestPasseDesFilms_OuVaLeTemps — LA DECOMPOSITION, film par film.
//
// Elle n a AUCUNE assertion de seuil, et c est delibere : un budget sur le decodage serait un
// budget sur la machine (la mesure du 2026-09-21 a vu un `CollectMatch` credit passer de 7,6 a
// 75 ms selon la charge). Ce que ce test produit est un TABLEAU, colle au plan ; ce qui garde
// la vitesse est le test d egalite 1 contre N et le budget de lecture du roster, tous deux
// ailleurs.
func TestPasseDesFilms_OuVaLeTemps(t *testing.T) {
	films := echantillonDuLot(t)
	db := openSharedTestDB(t)
	peuplerBancCredit(t, db)
	ctx := context.Background()

	var couts []coutDUnFilm
	for _, film := range films {
		c := mesurerUnFilm(t, ctx, db, strings.TrimSpace(film))
		if c == nil {
			continue
		}
		couts = append(couts, *c)
	}
	if len(couts) == 0 {
		t.Skip("aucun film de l echantillon n est present sur cette machine")
	}
	collerTableauDesCouts(t, couts)
}

// mesurerUnFilm chronometre les sept etapes d un match. Rend nil si le film n est pas la.
func mesurerUnFilm(t *testing.T, ctx context.Context, db *sql.DB, film string) *coutDUnFilm {
	t.Helper()
	if !filmPresent(film) {
		t.Logf("film %s absent du cache — saute", film)
		return nil
	}
	chunksDisque := chargerFilmDeFixture(t, film)
	if len(chunksDisque) == 0 {
		return nil
	}
	col := collecteurDeMesure(t, db, map[string][]haloclient.FilmChunk{film: chunksDisque})

	c := coutDUnFilm{film: film, chunks: len(chunksDisque)}
	for _, ch := range chunksDisque {
		c.octets += int64(len(ch.Data))
	}

	debut := time.Now()
	chunks, trouve, err := FilmChunksForMatch(ctx, col.client, film)
	c.chargement = time.Since(debut)
	if err != nil || !trouve {
		t.Logf("film %s : chunks indisponibles (%v) — saute", film, err)
		return nil
	}

	debut = time.Now()
	decode, err := FilmOf(chunks)
	c.assemblage = time.Since(debut)
	if err != nil {
		t.Logf("film %s : assemblage echoue (%v) — saute", film, err)
		return nil
	}
	if inscrireFilmAuRegistre(t, db, film, decode) == 0 {
		t.Logf("film %s : aucune identite au fil des morts — saute", film)
		return nil
	}

	debut = time.Now()
	opts := decfilm.DefaultOptions()
	opts.Carte = col.carteDuMatch(ctx, film)
	c.carte = time.Since(debut)

	debut = time.Now()
	res, err := decfilm.Decode(ctx, film, decode, &opts)
	c.decodage = time.Since(debut)
	if err != nil {
		t.Logf("film %s : decodage echoue (%v) — saute", film, err)
		return nil
	}

	debut = time.Now()
	ids, err := col.roster.IdentitiesForMatch(ctx, film)
	c.roster = time.Since(debut)
	if err != nil {
		t.Fatalf("roster %s: %v", film, err)
	}

	batch := BuildKillSourceBatch(film, res, ids)
	debut = time.Now()
	fusionnees, err := col.write(ctx, batch)
	c.ecriture = time.Since(debut)
	if err != nil {
		t.Fatalf("ecriture %s: %v", film, err)
	}

	debut = time.Now()
	col.collectShots(ctx, film, chunks, ids)
	col.collectPositions(ctx, film, decode, res, ids, batch.Deaths, fusionnees)
	col.collectHits(ctx, film, chunks, ids)
	c.annexes = time.Since(debut)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.heapInuse, c.sys = m.HeapInuse, m.Sys
	return &c
}

// filmPresent : le repertoire du film existe-t-il sous la racine de fixtures ?
func filmPresent(film string) bool {
	root := os.Getenv("KILLSOURCE_FIXTURES")
	if root == "" {
		return false
	}
	info, err := os.Stat(root + string(os.PathSeparator) + film)
	return err == nil && info.IsDir()
}

// collerTableauDesCouts colle LE TABLEAU du lot : une ligne par film, puis les parts.
func collerTableauDesCouts(t *testing.T, couts []coutDUnFilm) {
	t.Helper()
	sort.Slice(couts, func(i, j int) bool { return couts[i].chunks < couts[j].chunks })

	var b strings.Builder
	b.WriteString("\nfilm     chunks   Mio | charg. assembl.  carte  DECODE  roster  ecrit. annexes |  TOTAL | s/chunk | heap Mio  sys Mio\n")
	var somme coutDUnFilm
	for _, c := range couts {
		fmt.Fprintf(&b, "%-8s %6d %5.1f | %6s %8s %6s %7s %7s %7s %7s | %6s | %7.2f | %8.0f %8.0f\n",
			c.film, c.chunks, float64(c.octets)/(1<<20),
			ms(c.chargement), ms(c.assemblage), ms(c.carte), ms(c.decodage),
			ms(c.roster), ms(c.ecriture), ms(c.annexes), ms(c.total()),
			c.total().Seconds()/float64(max(c.chunks, 1)),
			float64(c.heapInuse)/(1<<20), float64(c.sys)/(1<<20))
		somme.chargement += c.chargement
		somme.assemblage += c.assemblage
		somme.carte += c.carte
		somme.decodage += c.decodage
		somme.roster += c.roster
		somme.ecriture += c.ecriture
		somme.annexes += c.annexes
		somme.chunks += c.chunks
		if c.heapInuse > somme.heapInuse {
			somme.heapInuse = c.heapInuse
		}
		if c.sys > somme.sys {
			somme.sys = c.sys
		}
	}
	tot := somme.total()
	fmt.Fprintf(&b, "\n%d films, %d chunks, total %s — pic HeapInuse %.0f Mio, pic Sys %.0f Mio\n",
		len(couts), somme.chunks, tot.Round(time.Millisecond),
		float64(somme.heapInuse)/(1<<20), float64(somme.sys)/(1<<20))
	part := func(nom string, d time.Duration) {
		fmt.Fprintf(&b, "  %-12s %8s  %5.1f %%\n", nom, d.Round(time.Millisecond),
			100*float64(d)/float64(max64(int64(tot), 1)))
	}
	part("chargement", somme.chargement)
	part("assemblage", somme.assemblage)
	part("carte", somme.carte)
	part("DECODAGE", somme.decodage)
	part("roster", somme.roster)
	part("ecriture", somme.ecriture)
	part("annexes", somme.annexes)
	t.Log(b.String())
}

func ms(d time.Duration) string { return d.Round(time.Millisecond).String() }

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
