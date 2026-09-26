package grammar

// frame_closure_ratchet_test.go — LA FERMETURE DES TRAMES DELTA NE DESCEND JAMAIS (lot J4.0).
//
// # CE QUE CE RATCHET GARDE
//
// [FrameClosure] mesure, par vue et par archetype, combien de paquets delta et de records se
// FERMENT au bit pres (`frame_closure.go`). Le golden fige ces comptes sur les bobines du depot
// (cf. ci-dessous) ; le test rougit des qu un compte `fermes` DESCEND. Frere du ratchet d image-cle
// (`keyframe_closure_ratchet_test.go`) : meme comparateur, meme sens (une hausse est un gain a
// figer, une baisse une largeur qui a bouge), meme porte de regeneration NOMMEE qui ne rend
// jamais `ok`.
//
// # LE CONTEXTE MESURE
//
// Celui des instruments ([contexteDeBobine]) : le profil par defaut et les largeurs d axe LUES
// DANS LE FILM (`DetectI0LayoutOf`), que la cuisson prend au catalogue de la carte (accord 7 films
// sur 7 le 2026-08-15). La calibration de `killsource`, que la cuisson pose aussi, n y est pas :
// la carte mesure la grammaire, pas une calibration.
//
// # LES HUIT MINI-BOBINES NE PORTENT AUCUNE TRAME DELTA — CONSTAT DU LOT J4.0, 2026-09-26
//
// Le plan demandait ce golden « sur les huit mini-bobines » (`replay/testdata/minifilm_*`). Mesure :
// les sept bobines PAR BUILD ne portent que `chunk_00`, des paquets d IMAGE-CLE (type 2) et le pied
// du film (leur `PROVENANCE.txt` le dit : elles servent la fermeture d image-cle) — ZERO paquet
// delta ; et `minifilm_000d5950`, qui porte des paquets delta, n a pas de `chunk_00` : sans
// registre la marche de production refuse le film ([ScanMarcheDesTrames] rend l erreur du
// registre) et la carte aussi. Les huit lignes sont GARDEES — elles disent ce fait, et une bobine
// qui disparaitrait de la mesure rougirait — mais a elles seules elles ne mesurent rien, et
// aucune mutation ne pourrait les faire rougir.
//
// LE RATCHET MORD SUR LES DEUX BOBINES CONTIGUES DU DEPOT, ET C EST UN ECART DECLARE AU PLAN :
// `facts/killsource/testdata/minibobine_000d5950` (prefixe CONTIGU des chunks 00 a 05 du film,
// en-tete compris, octets bruts — sa provenance explique pourquoi une bobine de paquets choisis
// ne decode rien) et `minibobine_e5adf7b2` (version 40 : registre + premier chunk de donnees).
// Ce sont les seuls films versionnes qui portent a la fois un registre et des trames delta
// consecutives. Prefixe `ks_` dans le golden, pour ne pas confondre `ks_000d5950` avec la
// mini-bobine du rejeu du meme film.
//
// REGENERATION (jamais d edition a la main) :
//
//	go test ./internal/games/halo_infinite/film/internal/grammar/ -run FrameClosureRatchet -update-frame-closure

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// updateFrameClosure : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE (cf. la porte
// nommee du ratchet d image-cle, correctif R1 P1-1 : une porte doit nommer CE qu elle regenere).
var updateFrameClosure = flag.Bool("update-frame-closure", false,
	"reecrire testdata/frame_closure.golden (lot J4.0) — CE golden seulement")

// frameClosureGoldenPath : le golden, a cote de celui des images-cles.
const frameClosureGoldenPath = "testdata/frame_closure.golden"

// porteFrameClosure : le drapeau, tel qu il se tape.
const porteFrameClosure = "-update-frame-closure"

// bobineDeCarte est un film mesure : son nom dans le golden et son repertoire.
type bobineDeCarte struct {
	nom, dir string
}

// frameClosureBobines : les HUIT mini-bobines du rejeu — les sept par build du ratchet d image-cle
// et la bobine historique sans `chunk_00` — puis les DEUX bobines contigues de `killsource` (cf.
// l en-tete : ce sont elles qui portent des trames delta sous un registre).
func frameClosureBobines() []bobineDeCarte {
	var out []bobineDeCarte
	for _, court := range append(closureMiniFilms(), "000d5950") {
		out = append(out, bobineDeCarte{nom: court,
			dir: filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)})
	}
	for _, court := range []string{"000d5950", "e5adf7b2"} {
		out = append(out, bobineDeCarte{nom: "ks_" + court,
			dir: filepath.Join("..", "facts", "killsource", "testdata", "minibobine_"+court)})
	}
	return out
}

// TestFrameClosureRatchet : la fermeture des trames delta ne descend jamais.
func TestFrameClosureRatchet(t *testing.T) {
	got := mesurerCarteBobines(t)
	if *updateFrameClosure {
		if err := os.WriteFile(frameClosureGoldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture du golden : %v", err)
		}
		// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` : cf. `TestKeyframeClosureRatchet`.
		t.Fatalf("1 reference(s) reecrite(s) : %s (%d octets) ; relancer sans %s pour verifier",
			frameClosureGoldenPath, len(got), porteFrameClosure)
	}
	brut, err := os.ReadFile(frameClosureGoldenPath) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent (%s) : %v — regenerer avec %s", frameClosureGoldenPath, err, porteFrameClosure)
	}
	comparerFermeture(t, string(brut), got, porteFrameClosure)
}

// usagesProduitDeLaTable rend les composants a usage produit de `ecs_table.tsv` : la colonne
// `product_use` renseignee et differente de « aucun ». Les lignes d alias (`ti` = -1) n en portent
// pas.
func usagesProduitDeLaTable(t *testing.T) UsagesProduit {
	t.Helper()
	out := UsagesProduit{}
	for _, r := range loadECSTable(t) {
		if r.TI >= 0 && r.ProductUse != "" && !strings.HasPrefix(r.ProductUse, "aucun") {
			out[CleComposant(r.TI, r.Component)] = true
		}
	}
	if len(out) == 0 {
		t.Fatal("aucun composant a usage produit dans la table : la mesure des records utiles serait vide")
	}
	return out
}

// mesurerCarteBobines rend le rendu textuel de la carte sur les bobines, une a la fois.
func mesurerCarteBobines(t *testing.T) string {
	t.Helper()
	utiles := usagesProduitDeLaTable(t)
	var b strings.Builder
	b.WriteString(enteteCarteDeFermeture)
	for _, bo := range frameClosureBobines() {
		if _, err := os.Stat(bo.dir); err != nil {
			t.Fatalf("bobine absente (%s) : %v", bo.dir, err)
		}
		film, err := source.LoadDir(bo.dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", bo.dir, err)
		}
		if _, ok := FilmRegistryChunk(film); !ok {
			fmt.Fprintf(&b, "%s\tmarche\t0\t0\tsans registre (pas de chunk_00)\t\n", bo.nom)
			continue
		}
		rep, err := FrameClosure(contexteDeBobine(film), utiles)
		if err != nil {
			t.Fatalf("FrameClosure %s : %v", bo.nom, err)
		}
		ecrireCarte(&b, bo.nom, rep)
	}
	return b.String()
}

// ecrireCarte rend les lignes d UN film : paquets, trois vues, records utiles, archetypes.
func ecrireCarte(b *strings.Builder, film string, r FrameClosureReport) {
	fmt.Fprintf(b, "%s\tpaquets\t%d\t%d\t%s\tlistes_non_localisees=%d\n", film, r.PaquetsFermes,
		r.Paquets, r.BloquantPrincipal(), r.ListesNonLocalisees)
	for v, nom := range []string{"A", "B", "C"} {
		s := r.Vues[v]
		fmt.Fprintf(b, "%s\tvue=%s\t%d\t%d\t%s\tterminees=%d arrets=%s\n", film, nom, s.Fermes,
			s.Atteints, composantLePlusBloquant(s.Arrets), s.Terminees, detailDesArrets(s.Arrets))
	}
	fmt.Fprintf(b, "%s\tutiles\t%d\t%d\t\tentrees_controle_fermees=%d\n", film, r.Utiles.RecordsFermes,
		r.Utiles.Records, r.Utiles.EntreesDeControleFermees)
	tis := make([]int, 0, len(r.Archetypes))
	for ti := range r.Archetypes {
		tis = append(tis, ti)
	}
	sort.Ints(tis)
	for _, ti := range tis {
		a := r.Archetypes[ti]
		fmt.Fprintf(b, "%s\tti=%d\t%d\t%d\t%s\tneufs=%d/%d deltas=%d/%d utiles=%d/%d\n", film, ti,
			a.NeufsFermes+a.DeltasFermes, a.Neufs+a.Deltas, a.Blocking, a.NeufsFermes, a.Neufs,
			a.DeltasFermes, a.Deltas, a.UtilesFermes, a.Utiles)
	}
}

// detailDesArrets rend les arrets d une vue, tries par nom : `cause:n;cause:n`.
func detailDesArrets(arrets map[string]int) string {
	causes := make([]string, 0, len(arrets))
	for c := range arrets {
		causes = append(causes, c)
	}
	sort.Strings(causes)
	parts := make([]string, 0, len(causes))
	for _, c := range causes {
		parts = append(parts, fmt.Sprintf("%s:%d", c, arrets[c]))
	}
	return strings.Join(parts, ";")
}

// enteteCarteDeFermeture : l en-tete du golden, HISTORIQUE DES REGENERATIONS compris. Il vit ICI,
// dans le generateur, et pas dans le golden : un historique que la porte d ecriture ne connait pas
// est efface par la regeneration suivante (lecon du lot 3.6.a, ratchet d image-cle).
const enteteCarteDeFermeture = "" +
	"# CARTE DE FERMETURE DES TRAMES DELTA — golden du lot J4.0 (plan de suite de l audit du decodeur).\n" +
	"# Colonnes : film, mesure, fermes, total, bloquant le plus frequent, detail.\n" +
	"#   paquets : paquets delta fermes au bit pres / marches ; bloquant = premiere cause d arret la plus frequente.\n" +
	"#   vue=A|B|C : paquets ou la vue se termine ET le paquet ferme / paquets ou la vue est atteinte.\n" +
	"#   utiles : records de la vue B dont le masque annonce un composant a usage produit\n" +
	"#     (colonne product_use de ecs_table.tsv), fermes / lus.\n" +
	"#   ti=N : records NEW + DELTA de l archetype, fermes / lus ; ti=-1 = archetype non resolu.\n" +
	"# Un record ne ferme que si son paquet ferme : la fin du paquet (vue C, reste de 0 a 7 bits nuls)\n" +
	"# est le seul oracle independant d une trame lue en sequence.\n" +
	"# Le ratchet rougit sur une BAISSE de `fermes`. Regeneration :\n" +
	"#   go test ./internal/games/halo_infinite/film/internal/grammar/ -run FrameClosureRatchet -update-frame-closure\n" +
	"#\n" +
	"# HISTORIQUE DES REGENERATIONS — une ligne par lot, avec CE QUI MONTE ET POURQUOI.\n" +
	"#\n" +
	"#   2026-09-26 lot J4.0 : creation, DIX bobines. Les huit mini-bobines du rejeu que le plan\n" +
	"#     designait ne portent AUCUNE trame delta sous un registre (sept par build : images-cles\n" +
	"#     seules, 0 paquet ; 000d5950 : pas de chunk_00, non marchee) ; les deux bobines contigues\n" +
	"#     de killsource (ks_) sont les seules du depot qui en portent — ecart declare au plan.\n" +
	"#     Premier apercu, contexte d instrument :\n" +
	"#       ks_000d5950 paquets 1 823/5 957 (30,6 %), records utiles 5 889/30 428 (19,4 %),\n" +
	"#         vue C fermee 1 823/3 485 atteintes ; premier bloquant\n" +
	"#         `ti=43 i35 device-animation-layer-state-component` (2 365 paquets arretes en vue B).\n" +
	"#       ks_e5adf7b2 paquets 371/584 (63,5 %), records utiles 0/1 ; premier bloquant\n" +
	"#         `vue C : terminateur hors cadre` (160 paquets).\n" +
	"#     Mutation jouee : `Block64` d object-shield-vitality lu sur 17 bits au lieu de 16 ->\n" +
	"#     sept lignes de ks_000d5950 en BAISSE et une disparue : le ratchet rougit.\n"
