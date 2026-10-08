//go:build research

package replay

// emprise_v1_porteurs_research_test.go — LOT V1.4 DU PLAN `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md` :
// L'ENTREE DE PRODUCTION `PortagesAuSync` REND-ELLE LES PORTAGES DU DOCUMENT DE REJEU ?
//
// # LE PROTOCOLE EST CELUI DU V0, L'INSTRUMENT EST REMPLACE PAR LA PRODUCTION
//
// Meme base (la passe du collecteur rejouee par `v0ChargerBase`, registre CONTROLE contre celui
// de la vraie passe), meme reference (le document de rejeu construit en memoire par la cuisson,
// volet `replaybuild` du V0, pose dans `ref/`), memes socles et libelles que la cuisson (pris dans
// la reference, comme le V0 : le collecteur les recevra du meme catalogue en V2). Ce qui change :
// la voie (b) n'est plus assemblee par l'instrument, c'est `PortagesAuSync` qui lit et assemble.
//
// CRITERE DU PLAN : intervalles IDENTIQUES a ceux du document de rejeu, sur les 9 films a porteur
// du V0. Chaque joueur est compare par l'union de ses portages en ms du MATCH (reference convertie
// avec son propre calage) : egalite EXACTE, et en second le taux a +-100 ms du V0.
//
// SANS LES VARIABLES DU V0, IL SE SAUTE (films, reference et passe du collecteur ne sont pas
// versionnes) :
//
//	EMPRISE_V0_FILMS=<9 match_id> EMPRISE_V0_DIR=<scratch du V0> EMPRISE_V0_CACHE=<racine data/cache> \
//	  go test ./internal/games/halo_infinite/film/replay/ -run '^TestEmpriseV1Porteurs$' -v -count=1

import (
	"context"
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/testutil"
)

func TestEmpriseV1Porteurs(t *testing.T) {
	films, dir, cache, _ := v0Films(t)
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := decfilm.LoadMapQuantCatalog(title.NewPathResolver(repoRoot).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	for _, id := range films {
		court := title.FilmShortMatchID(id)
		var e v0Entree
		if !v0Lire(t, dir, "ref", court, &e.ref) || !v0Lire(t, dir, "col", court, &e.col) ||
			!v0Lire(t, dir, "k1", court, &e.k1) {
			t.Fatalf("%s : ref/col/k1 absent — jouer les volets cuisson et collecteur du V0", court)
		}
		e.id, e.cache, e.cat, e.tours = id, cache, cat, 1
		v1ComparerUnFilm(t, e)
	}
}

// v1ComparerUnFilm rejoue la base, appelle l'entree de production et compare a la reference.
func v1ComparerUnFilm(t *testing.T, e v0Entree) {
	t.Helper()
	court := title.FilmShortMatchID(e.id)
	fam := v0FamilleDuMode(e.ref.Variante)
	b, _ := v0ChargerBase(t, e)
	if ecart := v0RegistreConforme(b); ecart != "" {
		t.Fatalf("%s : registre rejoue different de celui du collecteur : %s", court, ecart)
	}
	prof := b.res.ProfilCalibre
	debut := time.Now()
	portages, bilan := PortagesAuSync(context.Background(), EntreePorteursAuSync{
		MatchID: e.id, Film: b.film, Contexte: b.fc, Carte: b.entry, Variante: e.ref.Variante,
		ProfilDeBalayage: &prof, Lignes: v1Lignes(e.k1), Socles: e.ref.Socles, Libelles: e.ref.Libelles,
		Identite: IdentityInput{
			Positions: b.positions, BipedCreations: b.creations, Deaths: b.deaths,
			PlayerIndices: b.idx, RosterXUIDs: b.roster, Bots: e.col.Bots,
			Participants: e.col.Participants, MatchID: e.id,
		},
	})
	duree := time.Since(debut)
	var r v0Resultat
	ref := v0IntervallesRef(e, &r)
	cand := map[string]map[string][]v0Iv{fam: {}}
	for x, l := range portages {
		for _, iv := range l {
			cand[fam][strconv.FormatUint(x, 10)] = append(cand[fam][strconv.FormatUint(x, 10)], v0Iv{iv.DebutMS, iv.FinMS})
		}
	}
	joueurs, identiques := v1Identiques(ref[fam], cand[fam])
	f := v0ComparerFamilles(ref, cand)[fam]
	t.Logf("%s  %-26s famille=%-7s lectures=%+v intervalles=%d ouverts=%d | joueurs identiques %d/%d | "+
		"reference %d ms, identique a +-100 ms %d ms, en trop %d ms (%.2f %%) | %.0f ms", court,
		e.ref.Variante, fam, bilan.Lectures, bilan.Intervalles, bilan.DrapeauOuverts, identiques, joueurs,
		f.RefMS, f.OkMS, f.ExtraMS, 100*f.Taux(), float64(duree.Microseconds())/1000)
	if identiques != joueurs || f.Taux() < 1 {
		t.Errorf("%s : portages differents du document de rejeu (%d/%d joueurs identiques, %.2f %%)",
			court, identiques, joueurs, 100*f.Taux())
	}
}

// v1Lignes : la feuille du match, comme la cuisson la passe au pont par manche.
func v1Lignes(k1 v0K1) []types.PlayerLine {
	out := make([]types.PlayerLine, 0, len(k1.Faits.Players))
	for _, p := range k1.Faits.Players {
		out = append(out, types.PlayerLine{XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists})
	}
	return out
}

// v1Identiques compte les joueurs (reference OU candidat) dont l'union des portages est EXACTEMENT
// la meme des deux cotes.
func v1Identiques(ref, cand map[string][]v0Iv) (joueurs, identiques int) {
	vus := map[string]bool{}
	for x := range ref {
		vus[x] = true
	}
	for x := range cand {
		vus[x] = true
	}
	for x := range vus {
		joueurs++
		a, c := v0Union(ref[x]), v0Union(cand[x])
		if len(a) != len(c) {
			continue
		}
		egaux := true
		for i := range a {
			if a[i] != c[i] {
				egaux = false
				break
			}
		}
		if egaux {
			identiques++
		}
	}
	return joueurs, identiques
}
