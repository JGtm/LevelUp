package main

// basecache.go — LE CACHE DES ARTEFACTS DE LA BASE : UNE CUISSON PAR (COMMIT DE BASE, TEMOIN).
//
// # POURQUOI
//
// Le mode `--reference=base` cuit chaque temoin DEUX fois : au HEAD et a la base. Quand la base
// est fixee par `--base=<commit>`, la cuisson de la base ne change pas d'un passage a l'autre —
// la refaire coute la moitie d'un gate de 13 a 25 min pour rien. Ce fichier range l'artefact de
// la base sous une CLE qui reprend tout ce qui l'influence, et le relit tant que la cle est
// identique.
//
// # LA CLE (cleBase) — CE QUI INFLUENCE LA CUISSON DE LA BASE
//
//	BaseSHA      le SHA COMPLET du commit (git rev-parse), jamais un nom de branche : c'est lui
//	             qui fixe le code compile ET les catalogues de reference copies du worktree base.
//	             Le worktree de base est cree DEPUIS ce SHA (base.go), donc le SHA de la cle est
//	             celui qui a reellement ete cuit, meme si une branche bouge en cours de route.
//	Temoin       l'id du temoin (nom court du film).
//	TitleSlug    le titre (catalogues, chemins).
//	MatchID      l'identite complete du match, telle que les faits la portent.
//	FaitsSHA256  l'empreinte des FAITS exportes (cartes candidates, joueurs, equipes...) : ils
//	             sont exportes depuis la base partagee par l'outil du HEAD, donc ils bougent avec
//	             les donnees et avec le HEAD, pas avec le SHA de base.
//	FilmSHA256   l'empreinte du MANIFESTE et des CHUNKS du film, lus dans la racine de travail
//	             de la base — l'entree binaire de la cuisson. Un film re-capture invalide.
//	GoVersion    `go env GOVERSION` dans le worktree base : la chaine d'outils qui compile
//	             replay-build. GOOS/GOARCH suivent (le binaire est compile pour l'hote,
//	             CGO_ENABLED=0 fixe).
//	MemGiB       le plafond memoire arme sur l'enfant : il ne change pas le contenu d'une cuisson
//	             reussie, mais une cuisson repetee sous un autre plafond n'est pas « la meme
//	             cuisson » ; choix prudent, le plafond est une constante du gate.
//	Version      la version du FORMAT du cache (versionCacheBase) : la monter invalide tout.
//
// PAS DANS LA CLE, ET POURQUOI : les champs du manifeste du corpus (famille, mode, carte,
// raison) — la cuisson ne lit que l'id du temoin et les faits ; `--reference`, `--strict`,
// `--allow-missing`, `--temoins` (ils changent l'affichage ou la selection, pas la cuisson) ; le
// HEAD (le cote HEAD n'est jamais cache).
//
// UN PARAMETRE NON CAPTURABLE = PAS DE CACHE : si le SHA complet, la version de Go ou une
// empreinte manque, `valider` refuse la cle, le temoin est cuit normalement et le refus est
// journalise — jamais un cache sur une cle incomplete.
//
// # LE STOCKAGE
//
// `<parc>/data/cache/replay_corpus_gate_base/<sha>/<temoin>-<empreinte>.artefact.json`, `.filmfacts.bin` et
// `.meta.json`. Les TROIS fichiers s'ecrivent par `atomicfile.WriteFileStrict` (temporaire du
// meme dossier + fsync + rename), l'artefact, les faits du film, PUIS le meta : le meta est le
// marqueur de validation. Un crash avant lui laisse des fichiers sans meta, ignores a la lecture.
//
// LES FAITS DU FILM (`<short8>.filmfacts.bin`, `PathResolver.FilmFactsPath`) sont ceux que la
// cuisson de la base depose sous sa racine de travail : le banc de verite les relit pour les kills
// individuels. Une entree est COMPLETE avec artefact ET faits, sinon elle est recuite ; une
// cuisson qui ne laisse pas de faits (puits d'artefact qui refuse) ne range rien, journalise.
// La cuisson de la base part SANS faits pour ce film (`cuissonDeLaBase`) et seuls des faits ecrits
// depuis son debut se rangent (`ecritsDepuis`) : une racine de travail reutilisee entre deux bases
// ne prete jamais a l une les faits de l autre (revue finale P1-c).
//
// # LA LECTURE
//
// Hit seulement si le meta se decode, porte EXACTEMENT la cle demandee, et que l'artefact a la
// taille et le SHA-256 du meta et se decode en document JSON. Tout autre cas — absent, meta
// illisible, cle differente, artefact tronque — est un MISS : l'illisible/incoherent est
// journalise en `slog.Warn` avec son contexte, puis la base est recuite et le cache reecrit.
// Une ecriture de cache qui echoue est journalisee et ne fait PAS echouer le temoin.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/atomicfile"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replaydiff"
)

const (
	// dossierCacheBase : le sous-dossier du cache du parc reserve a ce gate.
	dossierCacheBase = "replay_corpus_gate_base"
	// versionCacheBase : la version du format du cache — a monter si la forme de la cle, du
	// meta ou de l'artefact range change.
	versionCacheBase = 1
)

// cleBase est la cle de cache d'UN temoin a UNE base, cf. l'en-tete. Tous les champs sont
// comparables : `==` decide de la coherence entre le meta lu et la cle demandee.
type cleBase struct {
	Version     int    `json:"version"`
	BaseSHA     string `json:"baseSha"`
	Temoin      string `json:"temoin"`
	TitleSlug   string `json:"titleSlug"`
	MatchID     string `json:"matchId"`
	FaitsSHA256 string `json:"faitsSha256"`
	FilmSHA256  string `json:"filmSha256"`
	GoVersion   string `json:"goVersion"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	MemGiB      int    `json:"memGib"`
}

// shaComplet dit si `s` est un SHA Git complet (40 hex en SHA-1, 64 en SHA-256).
func shaComplet(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && s == strings.ToLower(s)
}

// valider refuse une cle incomplete : un cache ne se range ni ne se lit sur une cle dont un
// composant manque.
func (c cleBase) valider() error {
	var manques []string
	if !shaComplet(c.BaseSHA) {
		manques = append(manques, "SHA complet du commit de base")
	}
	for nom, v := range map[string]string{
		"temoin": c.Temoin, "titre": c.TitleSlug, "matchId": c.MatchID,
		"empreinte des faits": c.FaitsSHA256, "empreinte du film": c.FilmSHA256,
		"version de Go": c.GoVersion, "GOOS": c.GOOS, "GOARCH": c.GOARCH,
	} {
		if v == "" {
			manques = append(manques, nom)
		}
	}
	if len(manques) > 0 {
		return fmt.Errorf("cle de cache incomplete, manque : %s", strings.Join(manques, ", "))
	}
	return nil
}

// empreinte rend l'identifiant court de la cle — le SHA-256 de sa serialisation JSON (champs
// dans l'ordre de declaration : deterministe), tronque a 16 hex.
func (c cleBase) empreinte() string {
	blob, _ := json.Marshal(c) //nolint:errchkjson // struct de chaines et d'entiers : Marshal ne peut pas echouer
	somme := sha256.Sum256(blob)
	return hex.EncodeToString(somme[:])[:16]
}

// metaCacheBase est le marqueur de validation d'une entree : la cle complete, plus de quoi
// verifier les DEUX fichiers ranges a cote (artefact et faits du film).
type metaCacheBase struct {
	Cle            cleBase `json:"cle"`
	ArtefactSHA256 string  `json:"artefactSha256"`
	ArtefactOctets int64   `json:"artefactOctets"`
	FaitsSHA256    string  `json:"filmfactsSha256"`
	FaitsOctets    int64   `json:"filmfactsOctets"`
	RangeLe        string  `json:"rangeLe"`
}

// baseCache est le cache des artefacts de la base. `Racine` vide = cache desactive.
type baseCache struct {
	Racine string
}

// actif dit si le cache est configure.
func (bc baseCache) actif() bool { return bc.Racine != "" }

// entreeBase nomme les trois fichiers d'une entree de cache.
type entreeBase struct {
	Artefact, Faits, Meta string
}

// chemins rend les trois fichiers d'une entree.
func (bc baseCache) chemins(c cleBase) entreeBase {
	stem := filepath.Join(bc.Racine, c.BaseSHA, c.Temoin+"-"+c.empreinte())
	return entreeBase{
		Artefact: stem + ".artefact.json",
		Faits:    stem + title.ExtensionFilmFacts,
		Meta:     stem + ".meta.json",
	}
}

// presence dit, par stat, lesquels des deux fichiers de donnees existent — ce que le rapport
// affiche (une entree complete les a tous les deux).
func (e entreeBase) presence() (artefact, faits bool) {
	return fichierExiste(e.Artefact), fichierExiste(e.Faits)
}

func fichierExiste(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// chercher rend l'entree rangee pour `c`, ou (_, false). Un miss franc (aucune entree) est un
// Info ; une entree ILLISIBLE, INCOHERENTE ou INCOMPLETE (artefact sans faits, ou l'inverse)
// est un Warn avec sa cause.
func (bc baseCache) chercher(ctx context.Context, c cleBase) (entreeBase, bool) {
	e := bc.chemins(c)
	blob, err := os.ReadFile(e.Meta) //nolint:gosec // chemin construit par ce gate
	if errors.Is(err, fs.ErrNotExist) {
		slog.InfoContext(ctx, "replay-corpus-gate: base absente du cache — cuisson", "temoin", c.Temoin,
			"base", c.BaseSHA)
		return entreeBase{}, false
	}
	if err != nil {
		return entreeBase{}, rejeterEntree(ctx, c, e.Meta, "meta illisible", err)
	}
	var m metaCacheBase
	if err := json.Unmarshal(blob, &m); err != nil {
		return entreeBase{}, rejeterEntree(ctx, c, e.Meta, "meta corrompu", err)
	}
	if m.Cle != c {
		return entreeBase{}, rejeterEntree(ctx, c, e.Meta, "meta incoherent avec la cle demandee",
			fmt.Errorf("cle du meta %+v", m.Cle))
	}
	if err := verifierArtefactRange(e.Artefact, m); err != nil {
		return entreeBase{}, rejeterEntree(ctx, c, e.Artefact, "artefact range invalide", err)
	}
	if err := verifierSomme(e.Faits, m.FaitsSHA256, m.FaitsOctets); err != nil {
		return entreeBase{}, rejeterEntree(ctx, c, e.Faits, "faits du film ranges invalides ou absents", err)
	}
	slog.InfoContext(ctx, "replay-corpus-gate: base relue du cache", "temoin", c.Temoin,
		"base", c.BaseSHA, "artefact", e.Artefact, "faits", e.Faits)
	return e, true
}

// rejeterEntree journalise une entree de cache inutilisable et rend false (la base sera
// recuite). Jamais silencieux : CLAUDE.md regle 3.
func rejeterEntree(ctx context.Context, c cleBase, chemin, cause string, err error) bool {
	slog.WarnContext(ctx, "replay-corpus-gate: entree du cache de base ignoree — recuisson",
		"temoin", c.Temoin, "base", c.BaseSHA, "chemin", chemin, "cause", cause, "err", err)
	return false
}

// verifierSomme controle un fichier range contre la taille et le SHA-256 que le meta en dit.
func verifierSomme(chemin, sha string, octets int64) error {
	somme, n, err := sha256Fichier(chemin)
	if err != nil {
		return err
	}
	if n != octets {
		return fmt.Errorf("taille %d, le meta dit %d", n, octets)
	}
	if somme != sha {
		return fmt.Errorf("sha256 %s, le meta dit %s", somme, sha)
	}
	return nil
}

// verifierArtefactRange controle l'artefact contre son meta, et qu'il se decode en document
// JSON (ce que la comparaison lira).
func verifierArtefactRange(chemin string, m metaCacheBase) error {
	if err := verifierSomme(chemin, m.ArtefactSHA256, m.ArtefactOctets); err != nil {
		return err
	}
	if _, err := replaydiff.LireDocument(chemin); err != nil {
		return fmt.Errorf("artefact non decodable : %w", err)
	}
	return nil
}

// ranger copie l'artefact et les faits du film cuits dans le cache sous la cle `c` : artefact,
// faits, PUIS meta (le marqueur de validation), chacun atomique. Une entree n'existe qu'avec les
// DEUX fichiers : sans faits, l'appelant ne range rien (et le journalise).
func (bc baseCache) ranger(c cleBase, artefactSource, faitsSource string) error {
	if err := c.valider(); err != nil {
		return err
	}
	artefact, err := os.ReadFile(artefactSource) //nolint:gosec // artefact produit par ce gate
	if err != nil {
		return fmt.Errorf("lecture de l'artefact a ranger : %w", err)
	}
	faits, err := os.ReadFile(faitsSource) //nolint:gosec // faits produits par la cuisson de ce gate
	if err != nil {
		return fmt.Errorf("lecture des faits du film a ranger : %w", err)
	}
	e := bc.chemins(c)
	if err := os.MkdirAll(filepath.Dir(e.Artefact), 0o750); err != nil {
		return fmt.Errorf("dossier du cache : %w", err)
	}
	if err := atomicfile.WriteFileStrict(e.Artefact, artefact, 0o600); err != nil {
		return fmt.Errorf("ecriture de l'artefact : %w", err)
	}
	if err := atomicfile.WriteFileStrict(e.Faits, faits, 0o600); err != nil {
		return fmt.Errorf("ecriture des faits du film : %w", err)
	}
	sa, sf := sha256.Sum256(artefact), sha256.Sum256(faits)
	blob, err := json.MarshalIndent(metaCacheBase{
		Cle: c, ArtefactSHA256: hex.EncodeToString(sa[:]), ArtefactOctets: int64(len(artefact)),
		FaitsSHA256: hex.EncodeToString(sf[:]), FaitsOctets: int64(len(faits)),
		RangeLe: time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation du meta : %w", err)
	}
	if err := atomicfile.WriteFileStrict(e.Meta, append(blob, '\n'), 0o600); err != nil {
		return fmt.Errorf("ecriture du meta : %w", err)
	}
	return nil
}

// sha256Fichier rend le SHA-256 (hex) et la taille d'un fichier, lu en flux.
func sha256Fichier(chemin string) (string, int64, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin construit par ce gate
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// empreinteFaits rend le SHA-256 des faits, sous la serialisation meme que celle que la
// cuisson lit (ecrireFaitsTemp).
func empreinteFaits(facts replaybuild.FactsFile) (string, error) {
	blob, err := json.Marshal(facts)
	if err != nil {
		return "", fmt.Errorf("serialisation des faits : %w", err)
	}
	somme := sha256.Sum256(blob)
	return hex.EncodeToString(somme[:]), nil
}

// empreinteFilm rend le SHA-256 du manifeste et de tous les chunks du film `short`, tels que
// la cuisson les lit sous `workRoot` (nom relatif + taille + contenu, dans l'ordre lexical).
func empreinteFilm(workRoot, short string) (string, error) {
	cacheRoot := title.NewPathResolver(workRoot).CacheRootDir()
	h := sha256.New()
	if err := hacherFichier(h, "manifeste", filmcache.ManifestPath(cacheRoot, short)); err != nil {
		return "", err
	}
	chunks := filmcache.ChunkDir(cacheRoot, short)
	err := filepath.WalkDir(chunks, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(chunks, p)
		if err != nil {
			return err
		}
		return hacherFichier(h, filepath.ToSlash(rel), p)
	})
	if err != nil {
		return "", fmt.Errorf("chunks du film %s : %w", short, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// hacherFichier ajoute au hacheur `nom`, la taille et le contenu du fichier.
func hacherFichier(h io.Writer, nom, chemin string) error {
	f, err := os.Open(chemin) //nolint:gosec // chemin interne au gate
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(h, "%s\x00%d\x00", nom, info.Size())
	_, err = io.Copy(h, f)
	return err
}

// cleCacheBase construit la cle du temoin `t`. Une erreur = cle non capturable : l'appelant
// cuit sans cache.
func (tc temoinContexte) cleCacheBase(t Temoin, facts replaybuild.FactsFile) (cleBase, error) {
	faits, err := empreinteFaits(facts)
	if err != nil {
		return cleBase{}, err
	}
	film, err := empreinteFilm(tc.WorkRootBase, t.ID)
	if err != nil {
		return cleBase{}, err
	}
	c := cleBase{
		Version: versionCacheBase, BaseSHA: tc.BaseSHA, Temoin: t.ID, TitleSlug: tc.TitleSlug,
		MatchID: facts.MatchID, FaitsSHA256: faits, FilmSHA256: film,
		GoVersion: tc.BaseGoVersion, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, MemGiB: tc.MemGiB,
	}
	return c, c.valider()
}

// baseResolue est l'artefact de la base tel que resoudreAvecCache le rend.
type baseResolue struct {
	// Artefact : le chemin a comparer (celui du cache sur un hit, le frais sinon).
	Artefact string
	// DuCache : la base vient du cache, pas d'une cuisson de ce passage.
	DuCache bool
	// ArtefactEnCache / FaitsEnCache : presence des deux fichiers dans l'entree de cache a
	// l'issue de la resolution (faux tous deux si le cache est desactive ou si rien n'a pu etre
	// range).
	ArtefactEnCache, FaitsEnCache bool
}

// cuissonBaseFn cuit la base et rend les chemins de l'artefact et des faits du film.
type cuissonBaseFn func() (artefact, faits string, err error)

// resoudreAvecCache rend l'artefact de la base : relu du cache si l'entree y est COMPLETE et
// valide (et que `forcer` est faux), sinon cuit par `cuire` puis range. Un echec de rangement,
// ou une cuisson qui n'a pas laisse de faits, est journalise et n'est pas une erreur du temoin.
func resoudreAvecCache(ctx context.Context, bc baseCache, c cleBase, forcer bool,
	cuire cuissonBaseFn) (baseResolue, error) {
	if !forcer {
		if e, ok := bc.chercher(ctx, c); ok {
			a, f := e.presence()
			return baseResolue{Artefact: e.Artefact, DuCache: true, ArtefactEnCache: a, FaitsEnCache: f}, nil
		}
	}
	debut := time.Now()
	artefact, faits, err := cuire()
	if err != nil {
		return baseResolue{}, err
	}
	res := baseResolue{Artefact: artefact}
	if !ecritsDepuis(faits, debut) {
		slog.WarnContext(ctx, "replay-corpus-gate: la cuisson de la base n'a pas ecrit de faits du "+
			"film — rien n'est range au cache (une entree sans faits, ou avec ceux d'une autre "+
			"cuisson, serait fausse)", "temoin", c.Temoin, "base", c.BaseSHA, "faits", faits)
		return res, nil
	}
	if err := bc.ranger(c, artefact, faits); err != nil {
		slog.WarnContext(ctx, "replay-corpus-gate: rangement de la base au cache impossible — "+
			"le temoin continue avec la cuisson fraiche", "temoin", c.Temoin, "base", c.BaseSHA, "err", err)
		return res, nil
	}
	res.ArtefactEnCache, res.FaitsEnCache = bc.chemins(c).presence()
	return res, nil
}

// toleranceHorodatage : la marge accordee a la granularite de l'horodatage du systeme de fichiers
// (2 s sur FAT) quand on compare la date d'ecriture des faits au debut de la cuisson.
const toleranceHorodatage = 2 * time.Second

// ecritsDepuis dit si le fichier de faits `chemin` existe ET a ete ecrit depuis `debut` (revue
// finale P1-c) : un fichier de faits plus ancien que la cuisson n'est pas le sien — il vient d'une
// cuisson precedente de la meme racine de travail, peut-etre d'une autre base — et ne se range pas.
func ecritsDepuis(chemin string, debut time.Time) bool {
	info, err := os.Stat(chemin)
	if err != nil || info.IsDir() {
		return false
	}
	return !info.ModTime().Before(debut.Add(-toleranceHorodatage))
}
