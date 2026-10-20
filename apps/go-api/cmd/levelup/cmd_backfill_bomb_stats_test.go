package main

// cmd_backfill_bomb_stats_test.go — LA CLE DE REPRISE et LA GARDE DU BATCH VIDE de
// `backfill-bomb-stats` (revue adversariale de branche, constat I-3 : « 279 lignes sans un
// seul test »).
//
// Le patron est celui de cmd_backfill_usage_summary_test.go, et pour la meme raison : les deux
// seules decisions de cette commande — SAUTER un match deja projete, et REFUSER d ecrire une
// passe vide — se testent SANS BASE. `projeterCorpusBombe` ne touche a la connexion qu apres
// ces deux decisions ; en `--dry-run`, elle n y touche jamais.
//
// CE QUE CHAQUE CAS ATTRAPE, ecrit avant les cas :
//
//	la cle de reprise inversee   sans `--force`, un match deja en base doit compter comme
//	                             DEJA EN BASE — pas comme ecrit, pas comme projete. L inverser
//	                             re-ecrirait tout le corpus a chaque lancement (et, avec
//	                             `--force` inverse, ne re-ecrirait plus jamais rien) ;
//	la garde du batch vide       un artefact qui porte `bombStats` SANS aucune ligne joueur ni
//	                             aucun fait date ne doit RIEN faire ecrire : il compte en
//	                             « sans calque ». Sans cette garde, le persister recevrait une
//	                             passe vide et ecrirait un `written_at` qui ferait passer le
//	                             match pour projete a la reprise suivante ;
//	les quatre etats de lecture  artefact absent / illisible / sans calque / projetable, sur un
//	                             chemin construit — la seule I/O de la commande.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// bsEcrireArtefact pose un artefact de rejeu a l endroit EXACT ou la commande ira le chercher
// (`PathResolver.ReplayArtifactPath`), et rend le resolveur qui l y trouve. Construire le
// chemin a la main dans le test en ferait une seconde definition du rangement.
func bsEcrireArtefact(t *testing.T, matchID, corps string) *titlePkg.PathResolver {
	t.Helper()
	pr := titlePkg.NewPathResolver(t.TempDir())
	path := pr.ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creer le dossier des artefacts: %v", err)
	}
	if err := os.WriteFile(path, []byte(corps), 0o644); err != nil {
		t.Fatalf("ecrire artefact: %v", err)
	}
	return pr
}

// bsArtefactPlein : un artefact minimal qui porte UNE ligne joueur et UN fait date — de quoi
// faire une passe projetable.
const bsArtefactPlein = `{"schemaVersion":39,"bombStats":{"players":[` +
	`{"xuid":"2533274800000001","grabs":2,"detonations":1}],"coverage":{}},` +
	`"bombEvents":[{"type":"bomb_detonated","timeMs":1234,"xuid":"2533274800000001"}]}`

// bsArtefactCalqueVide : le calque EXISTE (le mode est bien de la famille bomb, l artefact est
// au bon schema) mais il ne nomme personne et ne date rien. C est le cas de la garde.
const bsArtefactCalqueVide = `{"schemaVersion":39,"bombStats":{"coverage":{"carryRead":true}}}`

// TestProjeterCorpusBombe_CleDeReprise — sans `--force`, un match deja en base est SAUTE et
// compte comme tel ; avec `--force`, il est re-projete.
func TestProjeterCorpusBombe_CleDeReprise(t *testing.T) {
	const matchID = "000d5950-1234-4abc-9def-0123456789ab"
	pr := bsEcrireArtefact(t, matchID, bsArtefactPlein)
	dejaEcrits := map[string]bool{matchID: true}

	cas := []struct {
		nom                    string
		force                  bool
		dejaEcrits             map[string]bool
		veutEcrits, veutDeja   int
		veutJoueurs, veutFaits int
	}{{
		nom:        "deja en base, sans --force : saute et compte comme deja en base",
		dejaEcrits: dejaEcrits, veutDeja: 1,
	}, {
		// `--force` fait tomber la cle de reprise ENTIEREMENT : elle n est meme plus lue
		// (`matchsDejaProjetes` n est pas appelee), d ou la map vide de ce cas.
		nom:   "--force : re-projete meme un match deja en base",
		force: true, dejaEcrits: map[string]bool{},
		veutEcrits: 1, veutJoueurs: 1, veutFaits: 1,
	}, {
		nom:        "jamais projete : projete",
		dejaEcrits: map[string]bool{}, veutEcrits: 1, veutJoueurs: 1, veutFaits: 1,
	}}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			// `--dry-run` : la passe ne touche JAMAIS la connexion, d ou le `nil` — un
			// deref serait un echec de test, pas un faux vert.
			o := bombStatsOptions{titleSlug: titlePkg.DefaultSlug, force: c.force, dryRun: true}
			b := projeterCorpusBombe(context.Background(), nil, pr, o,
				[]string{matchID}, c.dejaEcrits)
			// En --dry-run rien n est ECRIT : ce que le cas appelle « ecrits » est le compte
			// des matchs PROJETES, que le bilan publie par ses totaux.
			projetes := 0
			if b.totalJoueurs+b.totalFaits > 0 {
				projetes = 1
			}
			if projetes != c.veutEcrits {
				t.Errorf("matchs projetes = %d, attendu %d (bilan %+v)", projetes, c.veutEcrits, b)
			}
			if b.dejaEnBase != c.veutDeja {
				t.Errorf("dejaEnBase = %d, attendu %d — LA CLE DE REPRISE EST INVERSEE",
					b.dejaEnBase, c.veutDeja)
			}
			if b.totalJoueurs != c.veutJoueurs || b.totalFaits != c.veutFaits {
				t.Errorf("totaux = %d joueurs / %d faits, attendu %d / %d",
					b.totalJoueurs, b.totalFaits, c.veutJoueurs, c.veutFaits)
			}
			if b.echecs != 0 || b.sansArtefact != 0 {
				t.Errorf("bilan pollue : %+v", b)
			}
		})
	}
}

// TestProjeterCorpusBombe_BatchVide — un calque present mais VIDE ne fait rien ecrire, et il se
// compte sous sa cause propre. C est la garde qui empeche un `written_at` sans contenu.
func TestProjeterCorpusBombe_BatchVide(t *testing.T) {
	const matchID = "9f57c612-1234-4abc-9def-0123456789ab"
	pr := bsEcrireArtefact(t, matchID, bsArtefactCalqueVide)
	o := bombStatsOptions{titleSlug: titlePkg.DefaultSlug, dryRun: true}
	b := projeterCorpusBombe(context.Background(), nil, pr, o,
		[]string{matchID}, map[string]bool{})
	if b.sansCalque != 1 {
		t.Errorf("sansCalque = %d, attendu 1", b.sansCalque)
	}
	if b.ecrits != 0 || b.totalJoueurs != 0 || b.totalFaits != 0 {
		t.Errorf("une passe VIDE a ete comptee comme projetee : %+v — le persister recevrait "+
			"un batch sans contenu, et le match passerait pour projete a la reprise suivante", b)
	}
}

// TestProjeterCorpusBombe_Limit — `--limit` borne les matchs PROJETES, pas les matchs examines :
// un match saute ne consomme pas le quota.
func TestProjeterCorpusBombe_Limit(t *testing.T) {
	const a, bID = "000d5950-1234-4abc-9def-0123456789ab", "35b75a31-1234-4abc-9def-0123456789ab"
	pr := bsEcrireArtefact(t, a, bsArtefactPlein)
	pathB := pr.ReplayArtifactPath(titlePkg.DefaultSlug, bID)
	if err := os.WriteFile(pathB, []byte(bsArtefactPlein), 0o644); err != nil {
		t.Fatalf("ecrire artefact: %v", err)
	}
	o := bombStatsOptions{titleSlug: titlePkg.DefaultSlug, dryRun: true, limit: 1}
	bilan := projeterCorpusBombe(context.Background(), nil, pr, o,
		[]string{a, bID}, map[string]bool{})
	if bilan.totalJoueurs != 1 || bilan.totalFaits != 1 {
		t.Errorf("--limit 1 : totaux = %d joueurs / %d faits, attendu 1 / 1 (bilan %+v)",
			bilan.totalJoueurs, bilan.totalFaits, bilan)
	}
}

// TestLireUnArtefactBombe — les quatre etats de la seule I/O de la commande, sur un chemin et un
// matchID construits. Aucune base.
func TestLireUnArtefactBombe(t *testing.T) {
	dir := t.TempDir()
	ecrire := func(nom, corps string) string {
		path := filepath.Join(dir, nom)
		if err := os.WriteFile(path, []byte(corps), 0o644); err != nil {
			t.Fatalf("ecrire %s: %v", nom, err)
		}
		return path
	}
	cas := []struct {
		nom  string
		path string
		want etatBombeMatch
	}{
		{"artefact absent : releve de backfill-replay", filepath.Join(dir, "jamais-ecrit.json"),
			bombeSansArtefact},
		{"artefact illisible : degrade CE match", ecrire("corrompu.json", "{pas du json"),
			bombeEchec},
		{"artefact sans calque (schema anterieur, ou hors famille bomb)",
			ecrire("sans.json", `{"schemaVersion":38}`), bombeSansCalque},
		{"calque present mais vide", ecrire("vide.json", bsArtefactCalqueVide), bombeSansCalque},
		{"artefact projetable", ecrire("plein.json", bsArtefactPlein), bombeAProjeter},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			batch, etat := lireUnArtefactBombe(c.path, "m1")
			if etat != c.want {
				t.Fatalf("etat = %d, attendu %d", etat, c.want)
			}
			if c.want != bombeAProjeter {
				if len(batch.Players) != 0 || len(batch.Events) != 0 {
					t.Fatalf("un match saute ne doit rendre aucune passe : %+v", batch)
				}
				return
			}
			if batch.MatchID != "m1" {
				t.Errorf("matchID = %q, attendu %q", batch.MatchID, "m1")
			}
			if len(batch.Players) != 1 || len(batch.Events) != 1 {
				t.Fatalf("passe = %d joueur(s) / %d fait(s), attendu 1 / 1", len(batch.Players),
					len(batch.Events))
			}
			// LA PROVENANCE EST RESOLUE A LA LECTURE, jamais laissee vide : le persister
			// refuserait la ligne (bomb_stats_persister.go).
			ev := batch.Events[0]
			if ev.Source == "" || ev.Confidence == "" {
				t.Errorf("fait date sans provenance : %+v", ev)
			}
			// Les colonnes non mesurees restent des POINTEURS NULS jusqu en base — « absent
			// n est pas zero » traverse la conversion sans etre aplati.
			pl := batch.Players[0]
			if pl.Grabs == nil || *pl.Grabs != 2 {
				t.Errorf("grabs = %v, attendu 2", pl.Grabs)
			}
			if pl.Arms != nil {
				t.Errorf("arms = %d alors que l artefact ne le porte pas — ABSENT n est pas ZERO",
					*pl.Arms)
			}
		})
	}
}
