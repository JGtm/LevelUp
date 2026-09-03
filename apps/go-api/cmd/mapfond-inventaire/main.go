// cmd/mapfond-inventaire — quelles cartes JOUEES n'ont pas de fond, et dans quelles playlists.
//
// POURQUOI CETTE COMMANDE EXISTE. Compter les cartes sans fond en regardant le repertoire
// `map_backgrounds/` donne un resultat FAUX, et l'erreur a deja ete commise deux fois
// (`.ai/V7.5/cartes/HANDOFF_FONDS_CARTE_2026-09-03.md`, section 7). Le fond se resout par TROIS
// chemins — la cle map_id, l'index par NOM des sidecars, puis l'heritage variante vers base — et
// seul le code qui sert le fond dit lequel s'applique. Cette commande appelle donc EXACTEMENT la
// meme resolution que `replayService.resolveBackgroundKey` : stat du sidecar sous map_id d'abord,
// `replay.MapBackgroundIndex.Lookup` sur les memes noms candidats ensuite, dans le meme ordre.
//
// CE QU'ELLE AJOUTE, ET C'EST LA QUESTION OUVERTE DU CHANTIER. Une carte sans fond ne merite une
// cuisson que si son MODE est supporte par le rejeu : ni le BTB ni le Firefight ne le sont, et y
// cuire un fond serait du travail perdu. Rien dans les catalogues ne le dit — l'inventaire UGC ne
// connait que `forge`/`native`, les vignettes de `static/maps` existent pour toutes les cartes.
// C'est `match_registry` qui porte la carte ET la playlist : la commande rend, par carte sans
// fond, ses playlists reelles, son drapeau Firefight et son effectif maximal (24 = BTB, 8 =
// arene). Le mode se lit, il ne se devine plus.
//
// SORTIE. Les DONNEES vont sur la sortie standard en TSV (une ligne par carte, playlists
// agregees dans une colonne) : c'est le produit de la commande, pas une trace. Le diagnostic —
// chemins ouverts, comptes, cartes ignorees — passe par `slog` sur la sortie d'erreur.
//
// OFFLINE PUR : deux bases ouvertes en LECTURE SEULE et un repertoire de sidecars versionnes.
// Rien n'ouvre le jeu, rien ne va sur le reseau, rien n'ecrit.
//
// Usage :
//
//	go run ./cmd/mapfond-inventaire                          # tout le registre, cartes sans fond
//	go run ./cmd/mapfond-inventaire --tous                   # y compris les cartes qui ont un fond
//	go run ./cmd/mapfond-inventaire --min-matchs 5           # au-dela de 5 matchs joues
//	go run ./cmd/mapfond-inventaire --shared X --metadata Y  # bases d'un autre worktree
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// voieResolution nomme le chemin par lequel le fond a ete trouve — la distinction qui manquait
// aux inventaires precedents, faits sur la seule cle map_id.
type voieResolution string

const (
	voieMapID  voieResolution = "map_id"
	voieIndex  voieResolution = "index-noms"
	voieAucune voieResolution = "AUCUN"
)

// verdict est la ligne d'inventaire d'une carte.
type verdict struct {
	Carte carteJouee
	Cle   string
	Voie  voieResolution
}

func main() {
	var (
		slug         = flag.String("title", title.DefaultSlug, "slug du titre")
		cheminShared = flag.String("shared", "", "chemin du registre partage ; vide = PathResolver")
		cheminMeta   = flag.String("metadata", "", "chemin de metadata.duckdb ; vide = PathResolver")
		dirFonds     = flag.String("fonds", "", "repertoire des fonds ; vide = PathResolver")
		tous         = flag.Bool("tous", false, "lister aussi les cartes qui ONT un fond")
		minMatchs    = flag.Int("min-matchs", 1, "n'afficher que les cartes au-dela de N matchs joues")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()

	chemins, err := resoutChemins(*slug, *cheminShared, *cheminMeta, *dirFonds)
	if err != nil {
		slog.ErrorContext(ctx, "inventaire des fonds : chemins non resolus", "err", err)
		os.Exit(1)
	}
	verdicts, err := inventorie(ctx, chemins)
	if err != nil {
		slog.ErrorContext(ctx, "inventaire des fonds : echec", "err", err)
		os.Exit(1)
	}
	imprime(ctx, verdicts, *tous, *minMatchs)
}

// cheminsInventaire regroupe les trois entrees, pour ne pas passer quatre chemins de fonction en
// fonction (seuil de 5 parametres, CLAUDE.md regle 5).
type cheminsInventaire struct {
	Shared   string
	Metadata string
	Fonds    string
}

// resoutChemins complete par le PathResolver ce que l'appelant n'a pas force.
func resoutChemins(slug, shared, metadata, fonds string) (cheminsInventaire, error) {
	c := cheminsInventaire{Shared: shared, Metadata: metadata, Fonds: fonds}
	if c.Shared != "" && c.Metadata != "" && c.Fonds != "" {
		return c, nil
	}
	racine, err := title.FindRepoRoot()
	if err != nil {
		return c, fmt.Errorf("racine du depot introuvable : %w", err)
	}
	res := title.NewPathResolver(racine)
	if c.Shared == "" {
		c.Shared = res.SharedDBPath(slug)
	}
	if c.Metadata == "" {
		c.Metadata = res.MetadataDBPath(slug)
	}
	if c.Fonds == "" {
		c.Fonds = res.MapBackgroundDir(slug)
	}
	return c, nil
}

// inventorie lit les deux bases, construit l'index des fonds et rend un verdict par carte jouee.
func inventorie(ctx context.Context, chemins cheminsInventaire) ([]verdict, error) {
	cartes, err := litCartes(ctx, chemins.Shared)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "registre lu", "cartes_jouees", len(cartes), "path", chemins.Shared)

	// Une metadata illisible ne stoppe pas l'inventaire : la production degrade de la meme
	// facon sur le seul libelle du registre. Mais elle est journalisee, jamais avalee.
	noms, err := litNomsAssets(ctx, chemins.Metadata)
	if err != nil {
		slog.WarnContext(ctx, "noms d'asset indisponibles — inventaire sur le seul libelle du registre",
			"err", err, "path", chemins.Metadata)
		noms = map[string]string{}
	} else {
		slog.InfoContext(ctx, "noms d'asset lus", "assets", len(noms), "path", chemins.Metadata)
	}

	idx, err := replay.MapBackgroundIndexFor(chemins.Fonds)
	if err != nil {
		return nil, fmt.Errorf("index des fonds : %w", err)
	}
	slog.InfoContext(ctx, "index des fonds construit",
		"fonds", idx.Cles(), "identites", idx.Identites(), "ambigues", len(idx.Ambigues()))
	for identite, cles := range idx.Ambigues() {
		slog.WarnContext(ctx, "identite ambigue — aucun fond ne sera servi sous ce nom",
			"identite", identite, "cles", strings.Join(cles, ", "))
	}

	out := make([]verdict, 0, len(cartes))
	for _, c := range cartes {
		c.NomAsset = noms[c.MapID]
		cle, voie := resoutFond(*c, chemins.Fonds, idx)
		out = append(out, verdict{Carte: *c, Cle: cle, Voie: voie})
	}
	trie(out)
	return out, nil
}

// resoutFond REJOUE `replayService.resolveBackgroundKey`, dans le meme ordre : le sidecar sous la
// cle map_id d'abord (cartes Forge encore publiees sous leur asset du jour), l'index par nom
// ensuite — lequel essaie l'identite exacte puis, a defaut, l'identite de base d'une variante.
//
// Toute divergence avec ce fichier rendrait l'inventaire faux sans qu'aucun test ne le voie :
// c'est pour cela qu'on appelle le MEME index plutot que de recopier sa regle.
func resoutFond(c carteJouee, dirFonds string, idx *replay.MapBackgroundIndex) (string, voieResolution) {
	if c.MapID != "" {
		if _, err := os.Stat(filepath.Join(dirFonds, c.MapID+".json")); err == nil {
			return c.MapID, voieMapID
		}
	}
	for _, nom := range c.Candidats() {
		if cle, ok := idx.Lookup(nom); ok {
			return cle, voieIndex
		}
	}
	return "", voieAucune
}

// trie met les cartes sans fond en tete, puis les plus jouees d'abord — l'ordre de traitement du
// chantier est celui des matchs decroissants (`RECETTE_TRAITEMENT_CARTE.md`, etape 1).
func trie(v []verdict) {
	sort.Slice(v, func(i, j int) bool {
		si, sj := v[i].Voie == voieAucune, v[j].Voie == voieAucune
		if si != sj {
			return si
		}
		if v[i].Carte.Matchs != v[j].Carte.Matchs {
			return v[i].Carte.Matchs > v[j].Carte.Matchs
		}
		return v[i].Carte.Libelle() < v[j].Carte.Libelle()
	})
}

// enteteTSV nomme les colonnes de la sortie.
const enteteTSV = "carte\tmap_id\tfond\tvoie\tmatchs\tfirefight\tjoueurs_max\tdernier\tplaylists"

// imprime rend le TSV sur la sortie standard et le bilan sur la sortie d'erreur.
func imprime(ctx context.Context, verdicts []verdict, tous bool, minMatchs int) {
	fmt.Println(enteteTSV)
	sansFond, retenues := 0, 0
	for _, v := range verdicts {
		if v.Voie == voieAucune {
			sansFond++
		}
		if !tous && v.Voie != voieAucune {
			continue
		}
		if v.Carte.Matchs < minMatchs {
			continue
		}
		retenues++
		fmt.Println(ligneTSV(v))
	}
	slog.InfoContext(ctx, "inventaire termine",
		"cartes_jouees", len(verdicts), "sans_fond", sansFond, "lignes_imprimees", retenues)
}

// ligneTSV met un verdict en une ligne. Les playlists tiennent dans une colonne, la plus jouee
// d'abord, sous la forme `nom (n)` — une colonne par playlist ferait une table a trous.
func ligneTSV(v verdict) string {
	dernier := ""
	if !v.Carte.Dernier.IsZero() {
		dernier = v.Carte.Dernier.UTC().Format("2006-01-02")
	}
	playlists := make([]string, 0, len(v.Carte.Playlists))
	for _, nom := range v.Carte.PlaylistsTriees() {
		playlists = append(playlists, fmt.Sprintf("%s (%d)", nom, v.Carte.Playlists[nom]))
	}
	return strings.Join([]string{
		v.Carte.Libelle(),
		v.Carte.MapID,
		v.Cle,
		string(v.Voie),
		fmt.Sprint(v.Carte.Matchs),
		fmt.Sprint(v.Carte.Firefight),
		fmt.Sprint(v.Carte.JoueursMax),
		dernier,
		strings.Join(playlists, " · "),
	}, "\t")
}
