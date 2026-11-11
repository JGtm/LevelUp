package replaybuild

// artifact_compress.go — CONVERTIR UN PARC D'ARTEFACTS BRUTS (`{short8}.json`) EN ARTEFACTS COMPRESSÉS
// (`{short8}.json.gz`), sur place. C'est la commande `levelup replay compress`.
//
// POUR CHAQUE ARTEFACT BRUT, DANS L'ORDRE, et l'ordre est la sûreté :
//
//  1. le JSON est lu (par la porte : un brut qui se décode mal est un ÉCHEC, il reste en place) ;
//  2. le `.json.gz` est posé par une création EXCLUSIVE ([replayartifact.WriteIfAbsent]) : un `.json.gz` déjà
//     présent n'est JAMAIS remplacé. Un serveur actif qui a posé un `.json.gz` plus récent entre la lecture et la
//     pose garde le sien ; le match est alors jugé comme un doublon (point 6) ;
//  3. le `.json.gz` posé est RELU et son contenu décompressé doit être, octet pour octet, le JSON d'origine —
//     sinon il est retiré (c'est le nôtre) et le `.json` reste ;
//  4. il reprend la DATE du `.json` : le rattrapage trie les artefacts sur leur date, une conversion ne les
//     rajeunit pas ;
//  5. l'en-tête `.meta` est réécrit APRÈS (il n'est donc pas plus ancien que l'artefact, et fait foi sans relire
//     l'artefact) et la marque de dérivation, si elle était à jour pour le brut, est reportée sur la nouvelle taille
//     stockée : convertir un parc ne fait rejouer aucune dérivation ;
//  6. alors seulement le `.json` est supprimé (son absence est tolérée : un serveur a pu le retirer).
//
// UN MATCH AUX DEUX FORMES (suite d'un arrêt entre 2 et 6, d'une suppression refusée, ou d'un serveur) se juge sur le
// contenu : `.gz` décodé IDENTIQUE au `.json` = le `.json` est supprimé (le doublon est résorbé, la relance
// termine ce qu'une interruption a laissé) ; contenus DIFFÉRENTS = rien n'est supprimé, le match est compté à part,
// lequel est le bon est un geste de l'opérateur.
//
// UN `.json.gz` INCOMPLET À CÔTÉ D'UN `.json` VALIDE (reste d'une pose par création exclusive interrompue, sur un dossier
// sans lien dur : cf. replayartifact.WriteIfAbsent) est retiré, puis le `.json` est converti. Cela ne vaut que si les
// octets du `.gz` ont été lus et ne se décodent pas, et si le `.json` se décode : jamais sur une erreur de lecture
// du disque, jamais quand le `.json` est lui aussi illisible.
//
// UNE SUPPRESSION QUI ÉCHOUE (fichier tenu ouvert par un lecteur sous Windows) n'est pas une conversion ratée : le
// `.json.gz` vérifié est en place, le match est compté « à relancer » et la relance résorbe le doublon.
//
// IDEMPOTENTE : un artefact déjà compressé n'est pas touché ; une seconde passe sans interruption ne fait rien.
//
// DEUX APPELANTS, nommés par `archlint/no_auto_replay_compress_test.go` : la commande `levelup replay compress`, et la file
// de régénération des rejeux du bureau, à la fin d'une passe lancée par l'utilisateur (`replayregen`). Elle se pilote par
// [CompressOptions] : une porte avant chaque artefact (pause, mode jeu, arrêt) et un compte rendu après chacun.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayartifact"
)

// CompressReport : le compte rendu chiffré d'une conversion ([CompressDir]).
type CompressReport struct {
	// Converted : artefacts convertis (à convertir, en simulation).
	Converted int
	// AlreadyCompressed : artefacts déjà compressés, sans `.json` voisin, laissés tels quels.
	AlreadyCompressed int
	// Resolved : matchs aux deux formes dont le `.gz` décodé est identique au `.json` : le `.json` est supprimé
	// (à supprimer, en simulation).
	Resolved int
	// Different : matchs aux deux formes dont les contenus diffèrent : aucun fichier n'est supprimé.
	Different int
	// Replaced : artefacts dont le `.json.gz` posé par cette conversion a été remplacé par un autre écrivain (un
	// serveur qui a re-cuit le match) avant la vérification : rien n'est touché, ce n'est pas un échec.
	Replaced int
	// ToRetry : matchs dont le `.json.gz` est posé et vérifié mais dont le `.json` n'a pas pu être supprimé :
	// relancer la commande.
	ToRetry int
	// Failed : artefacts non convertis (illisibles, écriture ou vérification en échec) ; chacun est journalisé.
	Failed int
	// BytesBefore / BytesAfter : somme des tailles stockées des artefacts convertis, avant (bruts) et après
	// (compressés).
	BytesBefore, BytesAfter int64
}

// Points d'injection des tests ; la production n'a qu'une valeur de chacun.
var (
	// relireCompresse relit un `.json.gz` fraîchement posé pour le vérifier : le fichier précis, sans résolution de
	// jumeau.
	relireCompresse = func(path string) ([]byte, error) {
		stocke, err := replayartifact.ReadStored(path)
		if err != nil {
			return nil, err
		}
		// Un fichier vide, ou nommé `.json.gz` sans en-tête gzip (0 ou 1 octet d'une écriture interrompue), est illisible :
		// Decode, lui, le rendrait tel quel.
		if err := replayartifact.ValidateStored(path, stocke); err != nil {
			return nil, fmt.Errorf("%w : %w", errContenuIllisible, err)
		}
		doc, err := replayartifact.Decode(stocke)
		if err != nil {
			return nil, fmt.Errorf("%w : %w", errContenuIllisible, err)
		}
		return doc, nil
	}
	// retirerLeBrut supprime le `.json` d'un match dont le `.json.gz` est posé et vérifié.
	retirerLeBrut = os.Remove
	// apresLectureDuBrut est appelé entre la lecture du `.json` et la pose du `.json.gz` : la fenêtre de la course
	// avec un serveur actif.
	apresLectureDuBrut func(brut string)
	// apresLaPose est appelé entre la pose du `.json.gz` et sa vérification : la fenêtre où un serveur peut le remplacer.
	apresLaPose func(ecrit string)
	// avantLeJugement est appelé avant le jugement d'un match aux deux formes.
	avantLeJugement func(compressed string)
)

// errRemplace : le `.json.gz` posé par cette conversion n'est plus celui que nous avons écrit.
var errRemplace = errors.New("le .json.gz a été remplacé par un autre écrivain")

// errContenuIllisible : les octets d'un `.json.gz` ont été lus et ne se décodent pas (tronqué, corrompu), par
// opposition à une erreur de lecture du disque.
var errContenuIllisible = errors.New("contenu gzip illisible")

// issue : ce qu'est devenu un match.
type issue int

const (
	issueConverti issue = iota
	issueResorbe
	issueDifferent
	issueARelancer
	issueRemplace   // le .json.gz posé a été remplacé par un autre écrivain : rien n'est touché
	issueDejaRetire // le .json du doublon a été retiré par un autre acteur : déjà compressé, pas résorbé par nous
)

// Add ajoute le compte rendu d'un artefact (ou d'un dossier) à celui-ci.
func (r *CompressReport) Add(o CompressReport) {
	r.Converted += o.Converted
	r.AlreadyCompressed += o.AlreadyCompressed
	r.Resolved += o.Resolved
	r.Different += o.Different
	r.Replaced += o.Replaced
	r.ToRetry += o.ToRetry
	r.Failed += o.Failed
	r.BytesBefore += o.BytesBefore
	r.BytesAfter += o.BytesAfter
}

// CompressOptions : ce qu'un appelant pilote dans une conversion.
type CompressOptions struct {
	// DryRun : lit et compresse en mémoire pour chiffrer le gain, n'écrit ni ne supprime rien.
	DryRun bool
	// Gate est consultée avant chaque artefact qui demande du travail (jamais pour un artefact déjà compressé et
	// sans doublon) ; une erreur arrête le parc et remonte telle quelle. nil = jamais.
	Gate func(ctx context.Context) error
	// Progress reçoit, après chaque artefact, le compte rendu de cet artefact seul. nil = jamais.
	Progress func(CompressReport)
}

// CompressDir convertit en place les artefacts bruts d'un dossier d'artefacts (cf. l'en-tête du fichier). `dryRun`
// lit et compresse en mémoire pour chiffrer le gain, et n'écrit ni ne supprime rien. Une erreur n'est rendue que
// pour un dossier illisible ou un contexte annulé ; les échecs par artefact sont dans le compte rendu.
func CompressDir(ctx context.Context, dir string, dryRun bool) (CompressReport, error) {
	return CompressDirWith(ctx, dir, CompressOptions{DryRun: dryRun})
}

// CompressDirWith est [CompressDir] pour un appelant qui interrompt et suit la conversion (cf. [CompressOptions]).
// Les artefacts se traitent un à un, jamais en parallèle.
func CompressDirWith(ctx context.Context, dir string, opts CompressOptions) (CompressReport, error) {
	entries, err := replayartifact.List(dir)
	if err != nil {
		return CompressReport{}, fmt.Errorf("dossier des artefacts %s : %w", dir, err)
	}
	var rep CompressReport
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		path := filepath.Join(dir, e.Name)
		travail := e.Name != e.Short+title.ReplayArtifactExt || replayartifact.HasLegacySibling(path)
		if travail && opts.Gate != nil {
			if err := opts.Gate(ctx); err != nil {
				return rep, err
			}
		}
		un := compressEntry(ctx, path, e, travail, opts.DryRun)
		rep.Add(un)
		if opts.Progress != nil {
			opts.Progress(un)
		}
	}
	return rep, nil
}

// compressEntry traite UN artefact du dossier et rend son compte rendu. `travail` : l'artefact est brut ou aux deux
// formes.
func compressEntry(ctx context.Context, path string, e replayartifact.Entry, travail, dryRun bool) CompressReport {
	if !travail {
		return CompressReport{AlreadyCompressed: 1}
	}
	var (
		res          issue
		avant, apres int64
		err          error
	)
	if e.Name != e.Short+title.ReplayArtifactExt {
		res, avant, apres, err = convertirUn(ctx, path, dryRun)
	} else {
		res, avant, apres, err = traiterUnDoublon(ctx, path, dryRun)
	}
	if err != nil {
		slog.ErrorContext(ctx, "replay compress: artefact non converti", "path", path, "err", err)
		return CompressReport{Failed: 1}
	}
	switch res {
	case issueConverti:
		return CompressReport{Converted: 1, BytesBefore: avant, BytesAfter: apres}
	case issueResorbe:
		return CompressReport{Resolved: 1}
	case issueDifferent:
		return CompressReport{Different: 1}
	case issueRemplace:
		slog.InfoContext(ctx, "replay compress: .json.gz remplacé par un autre écrivain après notre pose — rien n'est touché",
			"path", path)
		return CompressReport{Replaced: 1}
	case issueDejaRetire:
		return CompressReport{AlreadyCompressed: 1}
	default:
		return CompressReport{ToRetry: 1}
	}
}

// traiterUnDoublon juge un match aux deux formes ; si son `.json.gz` est incomplet (reste d'une écriture interrompue :
// les octets lus ne se décodent pas) alors que le `.json` est valide, le `.gz` est retiré et le `.json` converti.
// Une erreur de LECTURE du disque ne retire jamais rien : le `.gz` peut être le plus récent.
func traiterUnDoublon(ctx context.Context, compressed string, dryRun bool) (res issue, avant, apres int64, err error) {
	res, err = jugerLeDoublon(ctx, compressed, dryRun)
	if err == nil || !errors.Is(err, errContenuIllisible) {
		return res, 0, 0, err
	}
	_, legacy := title.ReplayArtifactCandidates(compressed)
	doc, lerr := readDecoded(legacy)
	if lerr == nil && !json.Valid(doc) {
		lerr = errors.New("JSON mal formé ou tronqué")
	}
	if lerr != nil {
		return 0, 0, 0, fmt.Errorf("%w ; le .json voisin est illisible aussi (%w) : rien n'est supprimé", err, lerr)
	}
	slog.WarnContext(ctx, "replay compress: .json.gz incomplet à côté d'un .json valide — repris", "compresse", compressed)
	if !dryRun {
		if rerr := os.Remove(compressed); rerr != nil {
			return 0, 0, 0, fmt.Errorf(".json.gz incomplet non retiré : %w", rerr)
		}
	}
	return convertirUn(ctx, legacy, dryRun)
}

// readDecoded lit et décode le fichier EXACT `path`.
func readDecoded(path string) ([]byte, error) {
	stocke, err := replayartifact.ReadStored(path)
	if err != nil {
		return nil, err
	}
	return replayartifact.Decode(stocke)
}

// convertirUn convertit un artefact brut et rend son issue et sa taille avant et après.
func convertirUn(ctx context.Context, brut string, dryRun bool) (res issue, avant, apres int64, err error) {
	stocke, err := replayartifact.ReadStored(brut)
	if err != nil {
		return 0, 0, 0, err
	}
	st, err := os.Stat(brut)
	if err != nil {
		return 0, 0, 0, err
	}
	doc, err := replayartifact.Decode(stocke)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("lecture : %w", err)
	}
	if apresLectureDuBrut != nil {
		apresLectureDuBrut(brut)
	}
	if dryRun {
		compresse, err := replayartifact.Encode(doc)
		if err != nil {
			return 0, 0, 0, err
		}
		return issueConverti, int64(len(stocke)), int64(len(compresse)), nil
	}
	ecrit, compresse, cree, err := replayartifact.WriteIfAbsent(ctx, brut, doc)
	if err != nil {
		return 0, 0, 0, err
	}
	if !cree {
		// Un `.json.gz` est apparu depuis le listage : on ne le touche pas, on juge le doublon.
		res, err = jugerLeDoublon(ctx, ecrit, false)
		return res, 0, 0, err
	}
	if err := verifierLaPose(ctx, ecrit, doc, compresse, st.ModTime()); err != nil {
		if errors.Is(err, errRemplace) {
			return issueRemplace, 0, 0, nil
		}
		return 0, 0, 0, err
	}
	renouvelerVoisins(ctx, brut, ecrit, doc, int64(len(stocke)), len(compresse))
	if retirerLeBrutVerifie(ctx, brut) == issueARelancer {
		return issueARelancer, 0, 0, nil
	}
	return issueConverti, int64(len(stocke)), int64(len(compresse)), nil
}

// verifierLaPose relit le `.json.gz` que cette conversion vient de poser. Il doit d'abord être CELUI QUE NOUS AVONS
// ÉCRIT (octets identiques à `pose`) : un serveur qui re-cuit le match peut l'avoir remplacé, et un fichier qui n'est pas
// le nôtre n'est jamais supprimé ([errRemplace]). Puis son contenu décompressé doit être, octet pour octet, le JSON
// d'origine (sinon il est retiré : c'est le nôtre), puis il reprend la date du `.json`.
func verifierLaPose(ctx context.Context, ecrit string, doc, pose []byte, date time.Time) error {
	if apresLaPose != nil {
		apresLaPose(ecrit)
	}
	actuel, err := replayartifact.ReadStored(ecrit)
	if err != nil {
		return fmt.Errorf("vérification : %w", err)
	}
	if !bytes.Equal(actuel, pose) {
		return errRemplace
	}
	relu, err := relireCompresse(ecrit)
	if err == nil && !bytes.Equal(relu, doc) {
		err = fmt.Errorf("le contenu relu du .json.gz (%d octets) diffère du JSON d'origine (%d octets)", len(relu), len(doc))
	}
	if err != nil {
		if rerr := os.Remove(ecrit); rerr != nil { // ecrit seul : c'est le .json.gz que cette conversion vient de poser
			slog.ErrorContext(ctx, "replay compress: .json.gz non vérifié non retiré", "path", ecrit, "err", rerr)
		}
		return fmt.Errorf("vérification : %w", err)
	}
	if err := os.Chtimes(ecrit, date, date); err != nil {
		slog.WarnContext(ctx, "replay compress: date du .json non reportée sur le .json.gz", "path", ecrit, "err", err)
	}
	return nil
}

// retirerLeBrutVerifie supprime le `.json` d'un match dont le `.json.gz` est vérifié. Absent = déjà retiré (tolérée).
// Un échec n'est pas une conversion ratée : le match est « à relancer ».
func retirerLeBrutVerifie(ctx context.Context, brut string) issue {
	err := retirerLeBrut(brut)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return issueConverti
	}
	slog.WarnContext(ctx, "replay compress: .json non supprimé — le .json.gz est posé et vérifié, relancer la commande",
		"path", brut, "err", err)
	return issueARelancer
}

// jugerLeDoublon juge un match qui porte `.json.gz` (compressedPath) ET `.json` : identique après décodage, le
// `.json` est supprimé (à supprimer, en simulation) ; différent, rien n'est supprimé.
func jugerLeDoublon(ctx context.Context, compressedPath string, dryRun bool) (issue, error) {
	if avantLeJugement != nil {
		avantLeJugement(compressedPath)
	}
	_, legacy := title.ReplayArtifactCandidates(compressedPath)
	brutStocke, err := replayartifact.ReadStored(legacy)
	if errors.Is(err, fs.ErrNotExist) {
		return issueDejaRetire, nil // retiré entre-temps par un autre acteur : plus de doublon, rien résorbé par nous
	}
	if err != nil {
		return 0, fmt.Errorf("lecture du .json du doublon : %w", err)
	}
	brutDoc, err := replayartifact.Decode(brutStocke)
	if err != nil {
		return 0, fmt.Errorf("lecture du .json du doublon : %w", err)
	}
	gz, err := relireCompresse(compressedPath)
	if err != nil {
		return 0, fmt.Errorf("lecture du .json.gz du doublon : %w", err)
	}
	if !bytes.Equal(gz, brutDoc) {
		slog.WarnContext(ctx, "replay compress: match aux deux formes de contenus DIFFÉRENTS — rien n'est supprimé, "+
			"choisir à la main", "compresse", compressedPath, "brut", legacy)
		return issueDifferent, nil
	}
	if dryRun {
		return issueResorbe, nil
	}
	if stGz, serr := replayartifact.ReadStored(compressedPath); serr == nil {
		d, _ := digestFromBytes(gz)
		d.Bytes = len(stGz)
		ecrireEntete(ctx, compressedPath, d, appVersionDeLEntete(legacy))
	}
	if retirerLeBrutVerifie(ctx, legacy) == issueARelancer {
		return issueARelancer, nil
	}
	return issueResorbe, nil
}

// renouvelerVoisins réécrit l'en-tête `.meta` du compressé et reporte la marque de dérivation du brut sur la taille
// stockée du compressé quand elle était à jour. Best-effort journalisé : un voisin non renouvelé coûte une relecture
// ou une dérivation rejouée, jamais un artefact faux.
func renouvelerVoisins(ctx context.Context, brut, ecrit string, doc []byte, tailleBrut int64, tailleCompresse int) {
	d, _ := digestFromBytes(doc)
	d.Bytes = tailleCompresse
	ecrireEntete(ctx, ecrit, d, appVersionDeLEntete(brut))
	m, ok := ReadDerivationsMark(brut)
	if !ok || m.Rev != DerivationsRev || int64(m.ArtifactBytes) != tailleBrut {
		return
	}
	m.ArtifactBytes = tailleCompresse
	if err := writeDerivationsMark(ecrit, m); err != nil {
		slog.WarnContext(ctx, "replay compress: marque de dérivation non reportée — les dérivés seront rejoués",
			"path", DerivationsMarkPath(ecrit), "err", err)
	}
}

// appVersionDeLEntete rend la version de l'application inscrite dans l'en-tête d'un artefact ("" si l'en-tête manque
// ou ne se lit pas : la version du producteur est alors inconnue, comme avant le fichier d'en-tête).
func appVersionDeLEntete(artifactPath string) string {
	raw, err := os.ReadFile(MetaPath(artifactPath))
	if err != nil {
		return ""
	}
	var m artifactMeta
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return m.AppVersion
}
