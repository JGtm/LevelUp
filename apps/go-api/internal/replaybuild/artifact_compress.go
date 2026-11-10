package replaybuild

// artifact_compress.go — CONVERTIR UN PARC D'ARTEFACTS BRUTS (`{short8}.json`) EN ARTEFACTS COMPRESSÉS
// (`{short8}.json.gz`), sur place. C'est la commande `levelup replay compress`.
//
// POUR CHAQUE ARTEFACT BRUT, DANS L'ORDRE, et l'ordre est la sûreté :
//
//  1. le JSON est lu (par la porte : un brut qui se décode mal est un ÉCHEC, il reste en place) ;
//  2. le `.json.gz` est écrit ATOMIQUEMENT (cf. [writeStored]) ;
//  3. il est RELU et son contenu décompressé doit être, octet pour octet, le JSON d'origine — sinon le
//     `.json.gz` est retiré et le `.json` reste ;
//  4. l'en-tête `.meta` est réécrit (la taille stockée a changé) et la marque de dérivation, si elle était à jour
//     pour le brut, est réécrite à la nouvelle taille stockée : convertir un parc ne fait rejouer aucune dérivation ;
//  5. alors seulement le `.json` est supprimé.
//
// IDEMPOTENTE : un artefact déjà compressé n'est pas touché ; une seconde passe ne convertit rien. Un match qui
// porte les DEUX formes est compté comme doublon et laissé tel quel (le compressé gagne à la lecture, cf.
// artifact_gate.go) : lequel des deux est le bon est un geste de l'opérateur.
//
// Un dossier qui sert un serveur en marche reste lisible pendant la conversion : le remplacement est atomique et
// la porte relit le compressé quand le brut vient de disparaître.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replayartifact"
)

// CompressReport : le compte rendu chiffré d'une conversion ([CompressDir]).
type CompressReport struct {
	// Converted : artefacts convertis (à convertir, en simulation).
	Converted int
	// AlreadyCompressed : artefacts déjà compressés, laissés tels quels.
	AlreadyCompressed int
	// Duplicates : matchs qui portent le `.json` ET le `.json.gz`, laissés tels quels.
	Duplicates int
	// Failed : artefacts non convertis (illisibles, écriture ou vérification en échec) ; chacun est journalisé.
	Failed int
	// BytesBefore / BytesAfter : somme des tailles stockées des artefacts convertis, avant (bruts) et après
	// (compressés).
	BytesBefore, BytesAfter int64
}

// relireCompresse relit un `.json.gz` fraîchement écrit pour le vérifier : le fichier précis, sans résolution de
// jumeau. Variable pour qu'un test fasse mentir la relecture ; la production n'a qu'une valeur.
var relireCompresse = func(path string) ([]byte, error) {
	stocke, err := replayartifact.ReadStored(path)
	if err != nil {
		return nil, err
	}
	return replayartifact.Decode(stocke)
}

// CompressDir convertit en place les artefacts bruts d'un dossier d'artefacts (cf. l'en-tête du fichier). `dryRun`
// lit et compresse en mémoire pour chiffrer le gain, et n'écrit ni ne supprime rien. Une erreur n'est rendue que
// pour un dossier illisible ou un contexte annulé ; les échecs par artefact sont dans le compte rendu.
func CompressDir(ctx context.Context, dir string, dryRun bool) (CompressReport, error) {
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
		if e.Name == e.Short+title.ReplayArtifactExt {
			if replayartifact.HasLegacySibling(path) {
				rep.Duplicates++
				slog.WarnContext(ctx, "replay compress: match présent sous deux formes — laissé tel quel", "compresse", path)
				continue
			}
			rep.AlreadyCompressed++
			continue
		}
		avant, apres, err := convertirUn(ctx, path, dryRun)
		if err != nil {
			rep.Failed++
			slog.ErrorContext(ctx, "replay compress: artefact non converti", "path", path, "err", err)
			continue
		}
		rep.Converted++
		rep.BytesBefore += avant
		rep.BytesAfter += apres
	}
	return rep, nil
}

// convertirUn convertit un artefact brut et rend sa taille avant et après.
func convertirUn(ctx context.Context, brut string, dryRun bool) (avant, apres int64, err error) {
	stocke, err := replayartifact.ReadStored(brut)
	if err != nil {
		return 0, 0, err
	}
	doc, err := replayartifact.Decode(stocke)
	if err != nil {
		return 0, 0, fmt.Errorf("lecture : %w", err)
	}
	if dryRun {
		compresse, err := replayartifact.Encode(doc)
		if err != nil {
			return 0, 0, err
		}
		return int64(len(stocke)), int64(len(compresse)), nil
	}
	ecrit, compresse, err := replayartifact.Write(ctx, brut, doc)
	if err != nil {
		return 0, 0, err
	}
	relu, err := relireCompresse(ecrit)
	if err == nil && !bytes.Equal(relu, doc) {
		err = fmt.Errorf("le contenu relu du .json.gz (%d octets) diffère du JSON d'origine (%d octets)", len(relu), len(doc))
	}
	if err != nil {
		if rerr := os.Remove(ecrit); rerr != nil { // ecrit seul : jamais le .json d origine
			slog.ErrorContext(ctx, "replay compress: .json.gz non vérifié non retiré", "path", ecrit, "err", rerr)
		}
		return 0, 0, fmt.Errorf("vérification : %w", err)
	}
	renouvelerVoisins(ctx, brut, ecrit, doc, int64(len(stocke)), len(compresse))
	if err := os.Remove(brut); err != nil {
		return 0, 0, fmt.Errorf("suppression du .json converti : %w", err)
	}
	return int64(len(stocke)), int64(len(compresse)), nil
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
