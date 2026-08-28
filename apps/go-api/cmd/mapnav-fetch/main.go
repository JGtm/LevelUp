// cmd/mapnav-fetch — rapatrie les `navmesh.blob` des cartes Forge.
//
// POURQUOI CETTE COMMANDE EXISTE. Le maillage de navigation d'une carte Forge est ce qui rend
// son fond lisible (`.ai/V7.5/cartes/NAVMESH_FORGE_2026-08-27.md`) : il sert de reference
// d'altitude et il borne le terrain joue. Il n'est PAS dans l'installation du jeu — il est
// publie a cote de la variante, dans l'asset UGC. Sans lui, la cuisson d'une carte organique
// retombe sur une reference interpolee depuis une vingtaine d'ancres, et l'image reste illisible.
//
// LA RESOLUTION EST ANONYME, ET C'EST MESURE : la page publique de l'asset porte un bloc
// `__NEXT_DATA__` d'ou l'on tire `Files.Prefix`, et le stockage blob sert le fichier SANS aucun
// jeton. Deux requetes, zero authentification — a la difference de `cmd/mapobj-build`, qui passe
// par Discovery UGC (HTTP 401 en anonyme) parce qu'il lui faut les metadonnees.
//
// TROIS GARDE-FOUS, dans cet ordre d'importance :
//
//  1. MEMOIRE. Le blob va sur le disque en FLUX (`io.Copy` sur un `io.LimitReader`) : rien n'est
//     materialise en memoire. Un `lightprobes.blob` pese 9,1 Mo et un corpus complet se compte
//     en centaines — tout charger serait une bombe.
//  2. REPRISE. Un blob deja present est saute, et l'ecriture passe par un fichier temporaire
//     renomme a la fin : une campagne interrompue ne laisse jamais un blob tronque que la
//     reprise prendrait pour complet.
//  3. POLITESSE. Un delai entre deux requetes, un User-Agent qui dit qui nous sommes.
//
// UN 404 N'EST PAS UNE ERREUR : mesure sur 23 cartes, le navmesh n'existe qu'au-dela d'environ
// 1 000 objets — present sur 10/10 au-dessus, absent (`BlobNotFound`) sur 13/13 en dessous. La
// commande le compte et continue ; elle n'echoue que sur un vrai probleme.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
)

const (
	pageAsset       = "https://www.halowaypoint.com/halo-infinite/ugc/maps/"
	nomBlobNavmesh  = "navmesh.blob"
	delaiRequete    = 45 * time.Second
	tailleBlobMax   = 64 << 20 // un navmesh realiste pese 100 a 400 Ko ; la borne protege le disque
	taillePageMax   = 8 << 20
	agentPoli       = "LevelUp/1.0 (dashboard stats Halo, usage personnel)"
	delaiParDefaut  = 1200 // ms entre deux cartes
	codeSortieEchec = 1
)

// ErrPasDeNavmesh : l'asset ne publie pas de maillage. Cas NOMINAL sous ~1 000 objets.
var ErrPasDeNavmesh = errors.New("mapnav-fetch: cet asset ne publie pas de navmesh.blob")

// blocNextData isole le JSON de la page publique. La page est du HTML rendu cote serveur ;
// on ne lit QUE ce bloc, jamais le corps.
var blocNextData = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

func main() {
	var (
		mapIDs  listeChaines
		toutes  = flag.Bool("toutes", false, "toutes les cartes declarees dans himap.CartesForge")
		depuis  = flag.String("depuis-fichier", "", "fichier texte : un map_id par ligne (# = commentaire)")
		sortie  = flag.String("out-dir", "", "repertoire de sortie (defaut : "+himap.DepotNavmesh+")")
		rateMS  = flag.Int("rate-ms", delaiParDefaut, "delai entre deux cartes, en millisecondes")
		refaire = flag.Bool("refaire", false, "retelecharger meme si le blob est deja en depot")
		dryRun  = flag.Bool("dry-run", false, "resoudre les assets sans rien ecrire")
		verbeux = flag.Bool("v", false, "journal de niveau debug")
	)
	flag.Var(&mapIDs, "map-id", "identifiant d'asset de carte (repetable)")
	flag.Parse()

	niveau := slog.LevelInfo
	if *verbeux {
		niveau = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: niveau})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := executer(ctx, options{
		mapIDs:  mapIDs,
		toutes:  *toutes,
		depuis:  *depuis,
		sortie:  *sortie,
		rateMS:  *rateMS,
		refaire: *refaire,
		dryRun:  *dryRun,
	}); err != nil {
		slog.ErrorContext(ctx, "mapnav-fetch", "err", err)
		os.Exit(codeSortieEchec)
	}
}

type options struct {
	mapIDs  []string
	toutes  bool
	depuis  string
	sortie  string
	rateMS  int
	refaire bool
	dryRun  bool
}

type bilan struct {
	demandes, rapatries, deja, sansNavmesh, echecs int
}

func executer(ctx context.Context, o options) error {
	racine, err := title.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("racine du depot introuvable: %w", err)
	}
	dest := o.sortie
	if dest == "" {
		dest = filepath.Join(racine, himap.DepotNavmesh)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("repertoire de sortie: %w", err)
	}
	liste, err := cibles(o)
	if err != nil {
		return err
	}
	if len(liste) == 0 {
		return errors.New("aucune carte demandee : -map-id, -depuis-fichier ou -toutes")
	}

	c := &client{http: &http.Client{Timeout: delaiRequete}}
	b := bilan{demandes: len(liste)}
	for i, id := range liste {
		if err := ctx.Err(); err != nil {
			slog.WarnContext(ctx, "interrompu", "traitees", i, "sur", len(liste))
			break
		}
		chemin := filepath.Join(dest, id+".blob")
		if !o.refaire {
			if st, err := os.Stat(chemin); err == nil && st.Size() > 0 {
				b.deja++
				slog.DebugContext(ctx, "deja en depot", "map_id", id, "octets", st.Size())
				continue
			}
		}
		n, err := c.rapatrie(ctx, id, chemin, o.dryRun)
		switch {
		case errors.Is(err, ErrPasDeNavmesh):
			b.sansNavmesh++
			slog.InfoContext(ctx, "pas de navmesh publie pour cet asset", "map_id", id)
		case err != nil:
			b.echecs++
			slog.ErrorContext(ctx, "rapatriement", "err", err, "map_id", id)
		default:
			b.rapatries++
			slog.InfoContext(ctx, "navmesh rapatrie", "map_id", id, "octets", n,
				"avancement", fmt.Sprintf("%d/%d", i+1, len(liste)))
		}
		if i+1 < len(liste) {
			select {
			case <-ctx.Done():
			case <-time.After(time.Duration(o.rateMS) * time.Millisecond):
			}
		}
	}
	slog.InfoContext(ctx, "campagne terminee", "demandes", b.demandes, "rapatries", b.rapatries,
		"deja", b.deja, "sansNavmesh", b.sansNavmesh, "echecs", b.echecs)
	if b.echecs > 0 {
		return fmt.Errorf("%d carte(s) en echec", b.echecs)
	}
	return nil
}

// cibles assemble la liste des map_id demandes, sans doublon et dans un ordre stable.
func cibles(o options) ([]string, error) {
	vu := map[string]bool{}
	var out []string
	ajoute := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || vu[id] {
			return
		}
		vu[id] = true
		out = append(out, id)
	}
	for _, id := range o.mapIDs {
		ajoute(id)
	}
	if o.depuis != "" {
		brut, err := os.ReadFile(o.depuis)
		if err != nil {
			return nil, fmt.Errorf("liste de cartes: %w", err)
		}
		for _, l := range strings.Split(string(brut), "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				ajoute(strings.Fields(l)[0])
			}
		}
	}
	if o.toutes {
		for _, c := range himap.CartesForge {
			ajoute(c.MapID)
		}
	}
	return out, nil
}

type client struct{ http *http.Client }

// rapatrie resout l'asset puis ecrit son navmesh sur le disque. Rend le nombre d'octets ecrits.
func (c *client) rapatrie(ctx context.Context, mapID, chemin string, dryRun bool) (int64, error) {
	return c.rapatrieDepuis(ctx, pageAsset+mapID, chemin, dryRun)
}

// rapatrieDepuis est la meme chose depuis une URL de page explicite. La couture existe pour
// que les temoins puissent servir une page et un blob sans sortir de la machine.
func (c *client) rapatrieDepuis(ctx context.Context, urlAsset, chemin string, dryRun bool) (int64, error) {
	prefixe, fichiers, err := c.resoutDepuis(ctx, urlAsset)
	if err != nil {
		return 0, err
	}
	if !contient(fichiers, nomBlobNavmesh) {
		return 0, ErrPasDeNavmesh
	}
	if dryRun {
		slog.InfoContext(ctx, "resolu (dry-run)", "asset", urlAsset, "prefixe", prefixe)
		return 0, nil
	}
	return c.telecharge(ctx, strings.TrimSuffix(prefixe, "/")+"/"+nomBlobNavmesh, chemin)
}

// filesAsset est la seule partie du document `__NEXT_DATA__` que l'on lit.
type filesAsset struct {
	Props struct {
		PageProps struct {
			Asset struct {
				Files struct {
					Prefix            string   `json:"Prefix"`
					FileRelativePaths []string `json:"FileRelativePaths"`
				} `json:"Files"`
			} `json:"asset"`
		} `json:"pageProps"`
	} `json:"props"`
}

// resout lit la page publique de l'asset et en tire le prefixe de stockage et la liste des
// fichiers publies. Aucun jeton : la page est publique.
func (c *client) resoutDepuis(ctx context.Context, urlAsset string) (string, []string, error) {
	corps, err := c.lit(ctx, urlAsset, taillePageMax)
	if err != nil {
		return "", nil, fmt.Errorf("page de l'asset: %w", err)
	}
	m := blocNextData.FindSubmatch(corps)
	if m == nil {
		return "", nil, errors.New("bloc __NEXT_DATA__ absent de la page")
	}
	var doc filesAsset
	if err := json.Unmarshal(m[1], &doc); err != nil {
		return "", nil, fmt.Errorf("__NEXT_DATA__ illisible: %w", err)
	}
	f := doc.Props.PageProps.Asset.Files
	if f.Prefix == "" || len(f.FileRelativePaths) == 0 {
		return "", nil, errors.New("l'asset ne declare aucun fichier")
	}
	return f.Prefix, f.FileRelativePaths, nil
}

// lit rend le corps d'une reponse, borne. Reserve aux documents COURTS (la page HTML) — les
// blobs, eux, ne passent jamais par la memoire (voir telecharge).
func (c *client) lit(ctx context.Context, url string, max int64) ([]byte, error) {
	rep, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer rep.Body.Close()
	if rep.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", rep.StatusCode)
	}
	return io.ReadAll(io.LimitReader(rep.Body, max))
}

func (c *client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agentPoli)
	return c.http.Do(req)
}

// telecharge ecrit le blob EN FLUX, par un fichier temporaire renomme a la fin.
func (c *client) telecharge(ctx context.Context, url, chemin string) (int64, error) {
	rep, err := c.get(ctx, url)
	if err != nil {
		return 0, fmt.Errorf("stockage blob: %w", err)
	}
	defer rep.Body.Close()
	if rep.StatusCode == http.StatusNotFound {
		return 0, ErrPasDeNavmesh
	}
	if rep.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("stockage blob: HTTP %d", rep.StatusCode)
	}
	tmp, err := os.CreateTemp(filepath.Dir(chemin), ".navmesh-*.part")
	if err != nil {
		return 0, err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name()) // sans effet si le renommage a eu lieu
	}()
	n, err := io.Copy(tmp, io.LimitReader(rep.Body, tailleBlobMax))
	if err != nil {
		return 0, fmt.Errorf("ecriture: %w", err)
	}
	if n == 0 {
		return 0, errors.New("blob vide")
	}
	if n == tailleBlobMax {
		return 0, fmt.Errorf("blob au-dela du plafond de %d octets", int64(tailleBlobMax))
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp.Name(), chemin); err != nil {
		return 0, fmt.Errorf("renommage: %w", err)
	}
	return n, nil
}

func contient(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// listeChaines rend un drapeau repetable.
type listeChaines []string

func (l *listeChaines) String() string     { return strings.Join(*l, ",") }
func (l *listeChaines) Set(v string) error { *l = append(*l, v); return nil }
