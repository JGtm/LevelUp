package main

// cmd_replay_parc_vies.go — sous-commande `levelup replay-parc-vies` : LES VIES SANS IDENTITE
// DU PARC, film par film, sans rien publier.
//
// # CE QU'ELLE MESURE
//
// Pour chaque film nomme par un fichier de faits (`replay-facts-export`), elle cuit le rejeu
// dans un ENFANT BORNE (`internal/replaychild`, verrou solo en attente bornee pris sur le cache
// des films, sentinelle memoire, priorite basse), relit le document rendu et ecrit :
//
//	<sortie>/films.tsv           une ligne par film : match, build, issue, variante, pistes,
//	                             vies sans identite (pistes sans xuid ni bot), compteurs de
//	                             couverture du pont, et la liste « slot:debut-fin » des vies
//	                             sans identite ;
//	<sortie>/identites/<id>.tsv  l'identite de CHAQUE piste publiee (slot, bornes, equipe,
//	                             identite) : c'est ce que `--comparer` relit pour prouver
//	                             qu'un correctif ne fait perdre ni changer aucun nom ailleurs.
//
// Le journal des enfants (dont l'alarme `logUnnamedLives`) part sur la sortie standard, chaque
// film precede d'une ligne `=== replay-parc-vies <id>`.
//
// # CE QU'ELLE NE FAIT PAS
//
// Elle n'ouvre AUCUNE base (les faits viennent des fichiers) et n'ecrit RIEN sous la racine du
// depot : l'artefact passe par un fichier temporaire de l'enfant, les faits de film ne sont pas
// ranges (`SansEcritureDesFaits`). Un film deja present dans `films.tsv` est saute : la passe se
// reprend en la relancant.
//
// Usage :
//
//	levelup replay-parc-vies --faits <dossier> --cache <data/cache> --sortie <dossier> [--ids F] [--title slug]
//	levelup replay-parc-vies --comparer <sortie avant> <sortie apres>

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replaychild"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/finalise"
)

// parcViesOutil : le nom du detenteur du verrou solo, lu par qui attend son tour.
const parcViesOutil = "replay-parc-vies"

// parcViesAttenteVerrou : l'attente maximale du verrou solo par film (une passe n'a pas de
// cycle suivant, elle attend son tour comme `backfill-replay`).
const parcViesAttenteVerrou = 10 * time.Minute

// optionsParcVies : les drapeaux de la mesure.
type optionsParcVies struct {
	faits, cache, sortie, ids, titre string
	// garderDocuments : depose aussi le document cuit sous <sortie>/documents/<id>.json (enquete
	// sur pieces d'un sous-ensemble ; plusieurs Mo par film).
	garderDocuments bool
}

// pisteMesuree : ce que la mesure retient d'une piste publiee.
type pisteMesuree struct {
	Slot       uint32 `json:"slot"`
	Team       int    `json:"team"`
	XUID       string `json:"xuid"`
	Bot        string `json:"bot"`
	StartFrame int    `json:"startFrame"`
	EndFrame   int    `json:"endFrame"`
}

// identite rend l'identite d'une piste : le xuid, sinon `bot:<nom>`, sinon "".
func (p pisteMesuree) identite() string {
	switch {
	case p.XUID != "":
		return p.XUID
	case p.Bot != "":
		return "bot:" + p.Bot
	}
	return ""
}

// documentMesure : la seule part du document de rejeu que la mesure lit (les points des pistes
// ne sont pas desserialises).
type documentMesure struct {
	Tracks   []pisteMesuree `json:"tracks"`
	Identity *struct {
		BipedSlots []lienDeSlot `json:"bipedSlots"`
	} `json:"identity"`
	Coverage *struct {
		Bridge struct {
			NamedByPreviousLife   int `json:"namedByPreviousLife"`
			NamedByNextLife       int `json:"namedByNextLife"`
			NamedBySlotBridge     int `json:"namedBySlotBridge"`
			UnnamedLives          int `json:"unnamedLives"`
			UnnamedLivesContested int `json:"unnamedLivesContested"`
		} `json:"bridge"`
		Decoder *struct {
			Build string `json:"build"`
		} `json:"decoder"`
	} `json:"coverage"`
}

func runReplayParcVies(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("replay-parc-vies", flag.ContinueOnError)
	var o optionsParcVies
	fs.StringVar(&o.faits, "faits", "", "dossier des <id>.facts.json (replay-facts-export)")
	fs.StringVar(&o.cache, "cache", "", "racine du cache des films (porte film_chunks/ et le verrou solo)")
	fs.StringVar(&o.sortie, "sortie", "", "dossier de sortie (films.tsv, identites/)")
	fs.StringVar(&o.ids, "ids", "", "fichier listant les films a mesurer (defaut : tous les faits)")
	fs.StringVar(&o.titre, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.BoolVar(&o.garderDocuments, "garder-documents", false, "deposer aussi <sortie>/documents/<id>.json")
	comparer := fs.Bool("comparer", false, "compare deux sorties : --comparer <avant> <apres>")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *comparer {
		if fs.NArg() != 2 {
			return errors.New("usage : levelup replay-parc-vies --comparer <sortie avant> <sortie apres>")
		}
		return comparerParcVies(os.Stdout, fs.Arg(0), fs.Arg(1))
	}
	if o.faits == "" || o.cache == "" || o.sortie == "" {
		return errors.New("usage : levelup replay-parc-vies --faits <dossier> --cache <data/cache> --sortie <dossier> [--ids F] [--title slug]")
	}
	return mesurerParcVies(context.Background(), cfg.RepoRoot, o)
}

// mesurerParcVies cuit chaque film dans un enfant, un a la fois, et ecrit sa ligne.
func mesurerParcVies(ctx context.Context, repoRoot string, o optionsParcVies) error {
	ids, err := idsDuParc(o)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(o.sortie, "identites"), 0o750); err != nil {
		return fmt.Errorf("dossier de sortie : %w", err)
	}
	tsv := filepath.Join(o.sortie, "films.tsv")
	faits, err := idsDejaMesures(tsv)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(tsv, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // chemin de l'operateur
	if err != nil {
		return fmt.Errorf("films.tsv : %w", err)
	}
	defer func() { _ = out.Close() }()
	mesures := 0
	for _, id := range ids {
		if faits[id] {
			continue
		}
		fmt.Printf("=== %s %s\n", parcViesOutil, id)
		ligne, idents := mesurerUnFilm(ctx, repoRoot, o, id)
		if idents != nil {
			chemin := filepath.Join(o.sortie, "identites", id+".tsv")
			if err := os.WriteFile(chemin, []byte(*idents), 0o600); err != nil {
				return fmt.Errorf("identites de %s : %w", id, err)
			}
		}
		if _, err := out.WriteString(ligne + "\n"); err != nil {
			return fmt.Errorf("films.tsv : %w", err)
		}
		mesures++
	}
	fmt.Printf("%s : %d film(s) mesure(s), %d deja presents\n", parcViesOutil, mesures, len(faits))
	return nil
}

// idsDuParc rend la liste des films : celle du fichier --ids, sinon tous les faits du dossier.
func idsDuParc(o optionsParcVies) ([]string, error) {
	if o.ids != "" {
		raw, err := os.ReadFile(o.ids) //nolint:gosec // chemin de l'operateur
		if err != nil {
			return nil, fmt.Errorf("liste des films : %w", err)
		}
		var ids []string
		for _, l := range strings.Split(string(raw), "\n") {
			if l = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l), "/")); l != "" {
				ids = append(ids, l)
			}
		}
		return ids, nil
	}
	entrees, err := os.ReadDir(o.faits)
	if err != nil {
		return nil, fmt.Errorf("dossier des faits : %w", err)
	}
	var ids []string
	for _, e := range entrees {
		if id, ok := strings.CutSuffix(e.Name(), ".facts.json"); ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// idsDejaMesures relit les films deja presents dans films.tsv (reprise de la passe).
func idsDejaMesures(tsv string) (map[string]bool, error) {
	out := map[string]bool{}
	raw, err := os.ReadFile(tsv) //nolint:gosec // chemin de l'operateur
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("films.tsv : %w", err)
	}
	for _, l := range strings.Split(string(raw), "\n") {
		if id, _, ok := strings.Cut(l, "\t"); ok && id != "id" {
			out[id] = true
		}
	}
	return out, nil
}

// mesurerUnFilm cuit un film et rend sa ligne de films.tsv et, quand le document existe, le
// contenu de son fichier d'identites. Un echec n'arrete pas la passe : il s'ecrit dans la ligne.
func mesurerUnFilm(ctx context.Context, repoRoot string, o optionsParcVies, id string) (string, *string) {
	f, err := replaybuild.ReadFactsFile(filepath.Join(o.faits, id+".facts.json"))
	if err != nil {
		return ligneParcVies(id, "", "", "sans_faits", "", nil), nil
	}
	req := requeteParcVies(repoRoot, o, id, f)
	res, err := replaychild.SpawnWaiting(ctx, req, o.cache, parcViesOutil, parcViesAttenteVerrou)
	if err != nil {
		issue := "echec"
		if errors.Is(err, replaybuild.ErrMapNotInCatalog) || errors.Is(err, replaybuild.ErrUnknownFilmKey) ||
			errors.Is(err, finalise.ErrFilmNonFinalise) {
			issue = "ecarte"
		}
		slog.WarnContext(ctx, "replay-parc-vies : film non cuit", "film", id, "issue", issue, "err", err)
		return ligneParcVies(id, f.MatchID, "", issue, f.GameVariantName, nil), nil
	}
	var doc documentMesure
	if err := json.Unmarshal(res.Blob, &doc); err != nil {
		slog.ErrorContext(ctx, "replay-parc-vies : document illisible", "film", id, "err", err)
		return ligneParcVies(id, f.MatchID, "", "illisible", f.GameVariantName, nil), nil
	}
	build := ""
	if doc.Coverage != nil && doc.Coverage.Decoder != nil {
		build = doc.Coverage.Decoder.Build
	}
	if o.garderDocuments {
		deposerLeDocument(ctx, o.sortie, id, res.Blob)
	}
	idents := identitesDuDocument(doc)
	return ligneParcVies(id, f.MatchID, build, "ok", f.GameVariantName, &doc), &idents
}

// deposerLeDocument ecrit le document cuit sous <sortie>/documents/ ; un echec se journalise et
// n'arrete pas la mesure (le document n'est qu'une piece d'enquete).
func deposerLeDocument(ctx context.Context, sortie, id string, blob []byte) {
	dir := filepath.Join(sortie, "documents")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		slog.ErrorContext(ctx, "replay-parc-vies : dossier des documents", "film", id, "err", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), blob, 0o600); err != nil {
		slog.ErrorContext(ctx, "replay-parc-vies : document non depose", "film", id, "err", err)
	}
}

// requeteParcVies assemble la requete de l'enfant : racine du depot pour les catalogues, film
// au cache de l'operateur, aucune ecriture de faits.
func requeteParcVies(repoRoot string, o optionsParcVies, id string, f replaybuild.FactsFile) replaychild.Request {
	return replaychild.Request{
		MatchID: f.MatchID, TitleSlug: o.titre, RepoRoot: repoRoot, MapNames: f.MapNames,
		FilmDir: filmcache.ChunkDir(o.cache, id), Facts: f.MatchFacts,
		SansEcritureDesFaits: true,
	}
}

// ligneParcVies formate la ligne de films.tsv d'un film. Colonnes : id, match, build (vide =
// build inconnu du profil), issue (ok, ecarte, echec, sans_faits, illisible), variante, pistes,
// pistes sans identite, residu publie par le pont, frontieres indecidables, nommees par la vie
// precedente/suivante/le pont, liste « slot:debut-fin » des pistes sans identite.
func ligneParcVies(id, matchID, build, issue, variante string, doc *documentMesure) string {
	champs := []string{id, matchID, build, issue, strings.ReplaceAll(variante, "\t", " ")}
	if doc == nil {
		return strings.Join(append(champs, "", "", "", "", "", ""), "\t")
	}
	var sansNom []string
	for _, t := range doc.Tracks {
		if t.identite() == "" {
			sansNom = append(sansNom, fmt.Sprintf("%d:%d-%d", t.Slot, t.StartFrame, t.EndFrame))
		}
	}
	prev, next, pont, residu, contestees := 0, 0, 0, 0, 0
	if doc.Coverage != nil {
		b := doc.Coverage.Bridge
		prev, next, pont = b.NamedByPreviousLife, b.NamedByNextLife, b.NamedBySlotBridge
		residu, contestees = b.UnnamedLives, b.UnnamedLivesContested
	}
	return strings.Join(append(champs, strconv.Itoa(len(doc.Tracks)), strconv.Itoa(len(sansNom)),
		strconv.Itoa(residu), strconv.Itoa(contestees),
		fmt.Sprintf("%d/%d/%d", prev, next, pont), strings.Join(sansNom, ";")), "\t")
}

// lienDeSlot : un lien du registre d'identite publie (section `identity.bipedSlots`).
type lienDeSlot struct {
	Slot uint32 `json:"slot"`
	Link struct {
		Source string `json:"source"`
		Method string `json:"method"`
		From   int    `json:"from"`
		To     int    `json:"to"`
	} `json:"link"`
}

// identitesDuDocument rend le fichier d'identites : une ligne par piste, dans l'ordre publie,
// avec la provenance du lien du registre qui couvre le milieu de la piste (« source/voie »,
// vide quand aucun lien ne la couvre).
func identitesDuDocument(doc documentMesure) string {
	var liens []lienDeSlot
	if doc.Identity != nil {
		liens = doc.Identity.BipedSlots
	}
	var b strings.Builder
	for _, t := range doc.Tracks {
		fmt.Fprintf(&b, "%d\t%d\t%d\t%d\t%s\t%s\n", t.Slot, t.StartFrame, t.EndFrame, t.Team,
			t.identite(), provenanceDeLaPiste(t, liens))
	}
	return b.String()
}

// provenanceDeLaPiste rend « source/voie » du lien du meme slot qui couvre le milieu de la piste.
func provenanceDeLaPiste(t pisteMesuree, liens []lienDeSlot) string {
	milieu := t.StartFrame + (t.EndFrame-t.StartFrame)/2
	for _, l := range liens {
		if l.Slot == t.Slot && l.Link.From <= milieu && milieu <= l.Link.To {
			return l.Link.Source + "/" + l.Link.Method
		}
	}
	return ""
}
