package main

// cmd_compact_passes_rewrite.go — `compact-passes --rewrite-file` (DC.5, amendé en C.8 puis C.10) :
// rendre au disque la place que la compaction a libérée.
//
// DuckDB réutilise les blocs libérés mais ne rétrécit jamais un fichier : seule une recopie dans
// un fichier NEUF (migration.CopierBaseVers, COPY FROM DATABASE) le réduit. L'échange n'a lieu
// que si le fichier neuf porte exactement le même inventaire que la base — catalogue (tables,
// vues, index, séquences avec leur prochaine valeur, macros, types), compte de chaque table,
// empreinte de chaque vue compactée.
//
// LA MÉTHODE (C.10, décision du superviseur après la seconde revue) : UNE connexion en écriture
// TIENT le verrou DuckDB de la base du début à la fin. Elle fait le CHECKPOINT, la copie vers le
// fichier neuf, puis la SAUVEGARDE par COPY FROM DATABASE (lire les octets d'un fichier que DuckDB
// tient est impossible sous Windows — mesuré : « utilisé par un autre processus »). L'inventaire
// du fichier neuf et celui de la sauvegarde se font sous ce verrou. Aucun autre processus ne peut
// donc ouvrir la base entre la copie et le remplacement : rien ne peut s'y écrire qui manquerait
// au fichier neuf.
//
// Le remplacement est un rename UNIQUE de `<base>.reecriture` sur la base — jamais de fenêtre
// sans fichier au chemin de la base — et dépend du système (cmd_compact_passes_remplacement_*.go) :
//   - POSIX : rename PENDANT que la connexion est ouverte (l'ancien inode reste verrouillé jusqu'à
//     la fermeture ; un processus qui ouvre le chemin après le rename ouvre le fichier neuf) ;
//   - Windows : un fichier tenu ne se remplace pas (mesuré : « Accès refusé ») — fermeture puis
//     rename IMMÉDIAT. Un tiers qui ouvre la base entre les deux la tient au moment du rename :
//     le rename échoue, refus, base d'origine intacte. Seule borne théorique : un tiers qui
//     ouvrirait, écrirait, ferait son CHECKPOINT et fermerait ENTIÈREMENT dans cet intervalle de
//     quelques microsecondes.
//
// Un WAL non vide à côté de la base AU LANCEMENT (transactions d'un processus tué avant son
// CHECKPOINT) fait refuser d'entrée : la commande n'ouvre pas une base qu'un autre a laissée à
// mi-chemin. Toute erreur avant le remplacement retire le fichier neuf et la sauvegarde partielle
// ou devenue inutile, journalisé. Un `<base>.reecriture` trouvé au départ est le reste d'une
// réécriture interrompue : la base est complète par construction, le reste est retiré (WARN).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// etapeReecriture : point d'observation entre les étapes de la réécriture (tests : arrêt simulé,
// tiers qui tente d'ouvrir la base). nil en production.
var etapeReecriture func(etape string) error

// Les étapes observables. `apres-fermeture` n'existe que sous Windows (entre la fermeture et le
// rename), `apres-remplacement` que sous POSIX (rename fait, connexion encore ouverte).
const (
	etapeApresCopie          = "apres-copie"
	etapeApresInventaire     = "apres-inventaire"
	etapeApresSauvegarde     = "apres-sauvegarde"
	etapeAvantRemplacement   = "avant-remplacement"
	etapeApresFermeture      = "apres-fermeture"
	etapeApresRemplacement   = "apres-remplacement"
	etapeApresEchange        = "apres-echange"
	suffixeFichierReecriture = ".reecriture"
)

func passerEtape(nom string) error {
	if etapeReecriture == nil {
		return nil
	}
	return etapeReecriture(nom)
}

// renommer : os.Rename, remplaçable par les tests (un rename entre deux volumes échoue).
var renommer = os.Rename

// reecriture : l'état d'une réécriture en cours.
type reecriture struct {
	path, neuf, sauvegarde string
	source                 *duckdb.DB // la connexion qui tient le verrou ; nil une fois fermée
}

// fermerSource ferme la connexion qui tient le verrou (une seule fois).
func (r *reecriture) fermerSource() error {
	if r.source == nil {
		return nil
	}
	h := r.source
	r.source = nil
	if err := h.Close(); err != nil {
		return fmt.Errorf("réécriture : fermeture de %s: %w", r.path, err)
	}
	return nil
}

// reecrireFichier réécrit `path` (appelé sous le bail d'écrivain, handle de compaction fermé).
func reecrireFichier(ctx context.Context, path, dossierSauvegarde string) error {
	r := &reecriture{path: path, neuf: path + suffixeFichierReecriture}
	if err := refuserWALNonVide(path); err != nil {
		return err
	}
	if err := retirerResteInterrompu(ctx, r.neuf); err != nil {
		return err
	}
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("refus : %s est tenue par un autre processus : %w", path, err)
	}
	r.source = h
	nObjets, err := r.preparerSousVerrou(ctx, dossierSauvegarde)
	if err != nil {
		return r.abandonner(ctx, err)
	}
	remplace, err := remplacerSousVerrou(r)
	if err != nil && !remplace {
		return r.abandonner(ctx, err)
	}
	if err != nil {
		// La base EST remplacée (fichier neuf complet) : rien à retirer, la sauvegarde reste.
		slog.ErrorContext(ctx, "réécriture : base remplacée, erreur après le remplacement",
			"path", path, "sauvegarde", r.sauvegarde, "err", err)
		return err
	}
	fmt.Printf("réécriture : inventaire identique (%d objets) ; ancien fichier sauvegardé : %s\n",
		nObjets, r.sauvegarde)
	return passerEtape(etapeApresEchange)
}

// preparerSousVerrou : CHECKPOINT, copie vers le fichier neuf, sauvegarde, et les deux inventaires
// comparés à celui de la base — tout sous le verrou de r.source. Rend le nombre d'objets.
func (r *reecriture) preparerSousVerrou(ctx context.Context, dossierSauvegarde string) (int, error) {
	db := r.source.SQLDb()
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return 0, fmt.Errorf("réécriture : checkpoint: %w", err)
	}
	avant, err := migration.LireInventaire(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("réécriture : inventaire de %s: %w", r.path, err)
	}
	if err := migration.CopierBaseVers(ctx, db, r.neuf); err != nil {
		return 0, err
	}
	if err := passerEtape(etapeApresCopie); err != nil {
		return 0, err
	}
	if err := verifierCopie(ctx, avant, r.neuf); err != nil {
		return 0, err
	}
	if err := passerEtape(etapeApresInventaire); err != nil {
		return 0, err
	}
	// Le chemin est retenu AVANT la copie : abandonner retire aussi une sauvegarde partielle.
	if r.sauvegarde, err = cheminDeSauvegarde(r.path, dossierSauvegarde, "avant-reecriture"); err != nil {
		return 0, fmt.Errorf("réécriture : dossier de sauvegarde: %w", err)
	}
	if err := migration.CopierBaseVers(ctx, db, r.sauvegarde); err != nil {
		return 0, fmt.Errorf("réécriture : sauvegarde: %w", err)
	}
	if err := verifierCopie(ctx, avant, r.sauvegarde); err != nil {
		return 0, fmt.Errorf("réécriture : sauvegarde: %w", err)
	}
	if err := passerEtape(etapeApresSauvegarde); err != nil {
		return 0, err
	}
	return len(avant.Objets), passerEtape(etapeAvantRemplacement)
}

// verifierCopie relit une copie SEULE (aucun ATTACH : ses vues se lient à ses tables) et la
// compare à l'inventaire de la base.
func verifierCopie(ctx context.Context, avant migration.Inventaire, chemin string) error {
	handle, err := duckdb.OpenReadOnly(chemin)
	if err != nil {
		return fmt.Errorf("réécriture : ouverture de %s: %w", chemin, err)
	}
	apres, err := migration.LireInventaire(ctx, handle.SQLDb())
	closeLogged(ctx, handle, chemin)
	if err != nil {
		return fmt.Errorf("réécriture : inventaire de %s: %w", chemin, err)
	}
	if ecarts := avant.Ecarts(apres); len(ecarts) > 0 {
		return fmt.Errorf("réécriture refusée, %s diffère de la base :\n%s", chemin, strings.Join(ecarts, "\n"))
	}
	return nil
}

// verifierWALAvantRemplacement : aucun WAL ne doit accompagner l'échange — celui de la base serait
// repris par le fichier neuf, celui du fichier neuf n'a pas lieu d'être après son DETACH.
func verifierWALAvantRemplacement(r *reecriture) error {
	for _, p := range []string{r.path + ".wal", r.neuf + ".wal"} {
		if fi, err := os.Stat(p); err == nil && fi.Size() > 0 {
			return fmt.Errorf("réécriture : %s non vide, échange refusé", p)
		}
	}
	return retirerFichiers(r.neuf + ".wal") // vide ou absent : le fichier neuf n'est ouvert par personne
}

// refuserWALNonVide : une base dont le WAL porte des transactions d'un processus tué avant son
// CHECKPOINT n'est pas réécrite ; l'exploitant l'ouvre d'abord (serveur ou CLI) pour le rejouer.
func refuserWALNonVide(path string) error {
	if fi, err := os.Stat(path + ".wal"); err == nil && fi.Size() > 0 {
		return fmt.Errorf("réécriture refusée : %s.wal non vide (%d octets) — transactions d'un autre "+
			"processus non intégrées ; ouvrir la base une fois (serveur ou CLI) pour les rejouer, puis "+
			"relancer", path, fi.Size())
	}
	return nil
}

// abandonner ferme la source si elle est encore ouverte, retire le fichier neuf et la sauvegarde
// (partielle ou devenue inutile), journalise, rend l'erreur. La base n'a pas été touchée.
func (r *reecriture) abandonner(ctx context.Context, cause error) error {
	errs := r.fermerSource()
	errs = errors.Join(errs, retirerFichiers(r.neuf, r.neuf+".wal"))
	if r.sauvegarde != "" {
		errs = errors.Join(errs, retirerFichiers(r.sauvegarde, r.sauvegarde+".wal"))
	}
	slog.ErrorContext(ctx, "réécriture abandonnée, base laissée telle quelle", "path", r.path,
		"err", cause, "err_retrait", errs)
	return errors.Join(cause, errs)
}

// retirerResteInterrompu : un `<base>.reecriture` présent au départ vient d'une réécriture
// interrompue ; la base est complète par construction, le reste est retiré.
func retirerResteInterrompu(ctx context.Context, neuf string) error {
	if _, err := os.Stat(neuf); os.IsNotExist(err) {
		return nil
	}
	slog.WarnContext(ctx, "réécriture : reste d'une réécriture interrompue retiré", "fichier", neuf)
	return retirerFichiers(neuf, neuf+".wal")
}

func retirerFichiers(chemins ...string) error {
	var errs error
	for _, p := range chemins {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			errs = errors.Join(errs, fmt.Errorf("réécriture : retrait de %s: %w", p, err))
		}
	}
	return errs
}
