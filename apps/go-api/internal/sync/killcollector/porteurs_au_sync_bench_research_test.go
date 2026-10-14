//go:build research

package killcollector

// porteurs_au_sync_bench_research_test.go — LE BANC DU COUT DE LA LECTURE DES PORTEURS AU SYNC
// (`.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, item 0.6, critere 4 et commande G-perf).
//
// # CE QU'IL CHRONOMETRE, ET RIEN D'AUTRE
//
// `replay.PortagesAuSync` sur l'entree que le collecteur lui donne en production, c'est-a-dire
// APRES la passe de positions : le contexte de film que l'etage du pont a ouvert sous la carte, le
// profil calibre par killsource, le registre d'identite. Le banc rebatit cette entree par les
// COUTURES de production du paquet, sans en recopier aucune — decodage (`decoderLeFilm`), etage du
// pont (`lireLePontDuCollecteur`), bots (`replayidentity.BotIdentities`), entree du registre
// (`entreeDuRegistre`), entree des porteurs (`depsDuPlacement.entreeDesPorteurs`, catalogues de
// `cataloguesDuPlacement`). Seul `PortagesAuSync` est sous le chronometre ; le contexte est NEUF a
// chaque tour (sa preparation, chronometre arrete), parce que les lectures des porteurs posent
// leur profil sur lui et que ses caches de lecture ne doivent pas servir un tour au suivant.
//
// # CE QUE LA BASE DONNERAIT, PRIS DANS LES FAITS VERSIONNES
//
// Le collecteur lit le roster, la feuille, les equipes, la variante et la carte dans la base
// partagee (`SharedRoster.participantsForMatch`). Le banc n'ouvre AUCUNE base : il les prend dans
// les faits d'equivalence versionnes du film (`replay/testdata/equivalence/<id>.facts.json`, la
// source de `replay-equiv`), par la projection de la cuisson pour le tableau des participants
// (`replaybuild.participantsDuTableau`). L'entree est donc la meme d'une execution a l'autre,
// entre la base et le lot : c'est ce que demande une comparaison de couts.
//
// # LE PIC
//
// L'empreinte du processus (`filmproc.Footprint`, la mesure de la sentinelle du depot et de
// `replay-equiv`), echantillonnee toutes les 10 ms pendant l'appel ; metrique `pic-Mio`, le plus
// haut des tours. Elle porte aussi ce que la passe de positions tient encore en memoire : c'est le
// niveau du collecteur au moment de l'appel.
//
// SANS LES VARIABLES, IL SE SAUTE (le cache de films n'est pas versionne) :
//
//	PORTEURS_BENCH_FILMS=<id court,...> PORTEURS_BENCH_CACHE=<racine data/cache> \
//	  go test -tags=research -run '^$' -bench '^BenchmarkPorteursAuSync$' -benchtime=1x -count=3 \
//	  ./internal/sync/killcollector/

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/replayidentity"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/testutil"
)

// dossierDesFaitsDEquivalence : les faits versionnes de `replay-equiv`, relatifs au paquet.
const dossierDesFaitsDEquivalence = "../../games/halo_infinite/film/replay/testdata/equivalence"

func BenchmarkPorteursAuSync(b *testing.B) {
	films, cache := strings.TrimSpace(os.Getenv("PORTEURS_BENCH_FILMS")), os.Getenv("PORTEURS_BENCH_CACHE")
	if films == "" || cache == "" {
		b.Skip("PORTEURS_BENCH_FILMS et PORTEURS_BENCH_CACHE requis : banc saute")
	}
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		b.Fatalf("racine du depot : %v", err)
	}
	ctx := context.Background()
	constructeur, err := replaybuild.NewBuilder(ctx, repoRoot, title.DefaultSlug)
	if err != nil {
		b.Fatalf("catalogue de bornes : %v", err)
	}
	libelles, objectifs := cataloguesDuPlacement(ctx, repoRoot, title.DefaultSlug)
	deps := depsDuPlacement{libelles: libelles, objectifs: objectifs}
	for _, court := range strings.Split(films, ",") {
		court = strings.TrimSpace(court)
		b.Run(court, func(b *testing.B) {
			banc := preparerLeBanc(b, constructeur, cache, court)
			var pic uint64
			var bilan replay.BilanPortages
			for b.Loop() {
				b.StopTimer()
				e := banc.entree(b, deps)
				ech := v0Echantillonner()
				b.StartTimer()
				_, bilan = replay.PortagesAuSync(ctx, e)
				b.StopTimer()
				pic = max(pic, ech.arreter())
				b.StartTimer()
			}
			b.ReportMetric(float64(pic)/(1<<20), "pic-Mio")
			b.ReportMetric(float64(bilan.Intervalles), "portages")
		})
	}
}

// bancDuFilm : ce qui ne change pas d'un tour a l'autre — le film, la carte, le decodage
// killsource et les identites du match.
type bancDuFilm struct {
	matchID string
	film    *decfilm.Film
	carte   decfilm.MapQuantEntry
	res     *decfilm.Result
	ids     MatchIdentities
}

// preparerLeBanc charge le film et ses faits, resout la carte comme la cuisson
// (`Builder.ResolveMapEntry`) et decode killsource comme le collecteur (`decoderSousLaCarte`).
func preparerLeBanc(b *testing.B, constructeur *replaybuild.Builder, cache, court string) bancDuFilm {
	b.Helper()
	faits, err := replaybuild.ReadFactsFile(filepath.Join(dossierDesFaitsDEquivalence, court+".facts.json"))
	if err != nil {
		b.Fatalf("%s : %v", court, err)
	}
	film, ok, err := filmcache.LoadFilm(cache, court)
	if err != nil || !ok {
		b.Fatalf("%s : film absent ou illisible du cache %s : %v", court, cache, err)
	}
	carte, err := constructeur.ResolveMapEntry(faits.MapNames)
	if err != nil {
		b.Fatalf("%s : %v", court, err)
	}
	opts := decfilm.DefaultOptions()
	opts.Carte = &carte
	res, err := decoderLeFilm(context.Background(), faits.MatchID, film, &opts)
	if err != nil {
		b.Fatalf("%s : decodage killsource : %v", court, err)
	}
	return bancDuFilm{matchID: faits.MatchID, film: film, carte: carte, res: res, ids: identitesDesFaits(faits.MatchFacts)}
}

// entree rejoue l'etage du pont sur un contexte NEUF et rend l'entree des porteurs du collecteur.
// Un pont non publiable saute le film : le collecteur n'appelle pas les porteurs dans ce cas
// (`portagesDuMatch`).
func (f bancDuFilm) entree(b *testing.B, deps depsDuPlacement) replay.EntreePorteursAuSync {
	b.Helper()
	ctx := context.Background()
	lectures, _, err := lireLePontDuCollecteur(ctx, f.film, f.carte, f.ids, f.matchID)
	if err != nil {
		b.Fatalf("%s : etage du pont : %v", f.matchID, err)
	}
	entree := entreeDuRegistre(lectures, f.ids, replayidentity.BotIdentities(f.res), f.matchID)
	reg := replay.BuildIdentityRegistry(ctx, entree)
	if !reg.PontPubliable() {
		b.Skipf("%s : pont non publiable, le collecteur ne lit pas les porteurs", f.matchID)
	}
	mat := materiauDIsolement{registre: reg, positions: lectures.positions, film: f.film,
		contexte: lectures.contexte, carte: f.carte, profil: profilCalibre(f.res), identite: entree}
	return deps.entreeDesPorteurs(ctx, f.matchID, mat, f.ids)
}

// identitesDesFaits rend, depuis les faits versionnes, ce que `SharedRoster.participantsForMatch`
// lit en base : roster, feuille, equipes (un camp inconnu, -1, n'entre pas), tableau des
// participants (projection de la cuisson, `replaybuild.participantsDuTableau`), variante et carte.
// Lignes dans l'ordre des xuids, comme la requete de production.
func identitesDesFaits(f domain.MatchFacts) MatchIdentities {
	joueurs := append([]domain.MatchPlayerFact(nil), f.Players...)
	sort.Slice(joueurs, func(i, j int) bool { return joueurs[i].XUID < joueurs[j].XUID })
	ids := MatchIdentities{Equipes: map[string]int{}, Variante: f.GameVariantName, CarteID: f.MapID}
	for _, p := range joueurs {
		if p.XUID == "" {
			continue
		}
		ids.XUIDs = append(ids.XUIDs, p.XUID)
		ids.Feuille = append(ids.Feuille, decfilm.PlayerLine{XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists})
		if p.TeamID >= 0 {
			ids.Equipes[p.XUID] = p.TeamID
		}
		ids.Participants = append(ids.Participants, replay.Participant{
			ID: p.XUID, JoinedInProgress: p.JoinedInProgress, JoinMatchMS: p.JoinMatchMS,
		})
	}
	return ids
}
