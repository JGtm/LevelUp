package main

// orchestrate.go — TRAITER UN TEMOIN : STAGER SES CHUNKS, LE CUIRE (UNE FOIS EN MODE PARC, DEUX
// FOIS EN MODE BASE), COMPARER.

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/replaybuild"
)

// errAbsentDuParc marque un temoin dont l'artefact de reference (mode parc) n'existe pas au
// parc local — un avertissement pour l'appelant, jamais une erreur de cuisson.
var errAbsentDuParc = errors.New("absent du parc")

// temoinContexte regroupe tout ce qu'un traitement de temoin a besoin de connaitre — une
// signature a plus de 5 parametres serait une violation directe (CLAUDE.md n°5).
type temoinContexte struct {
	ParcRoot     string
	WorkRoot     string // racine de cuisson HEAD
	WorkRootBase string // racine de cuisson BASE ; vide si Reference == "parc"
	BinHead      string // binaire replay-build compile au HEAD
	BinBase      string // binaire replay-build compile a la base ; vide si Reference == "parc"
	LockRoot     string
	TitleSlug    string
	FactsDir     string
	Reference    string // "base" (defaut) ou "parc"
}

// traiterTemoin cuit et compare UN temoin ; ne rend JAMAIS d'erreur — un temoin absent ou en
// echec produit une ligne qui le dit, pour que les autres temoins du manifeste soient traites
// quand meme (CLAUDE.md : un rapport partiel muet vaut moins qu'un rapport complet nomme).
func traiterTemoin(t Temoin, ctx temoinContexte) ligneRapport {
	base := ligneRapport{Temoin: t}

	if _, err := stageFilm(ctx.ParcRoot, ctx.WorkRoot, t.ID); err != nil {
		slog.Warn("replay-corpus-gate: temoin absent du parc local — ignore, pas un echec",
			"temoin", t.ID, "famille", t.Famille, "err", err)
		base.Absent, base.AbsentCause = true, err.Error()
		return base
	}
	if ctx.Reference == "base" {
		if _, err := stageFilm(ctx.ParcRoot, ctx.WorkRootBase, t.ID); err != nil {
			slog.Warn("replay-corpus-gate: temoin absent du parc local (racine base) — ignore",
				"temoin", t.ID, "famille", t.Famille, "err", err)
			base.Absent, base.AbsentCause = true, err.Error()
			return base
		}
	}

	factsPath := filepath.Join(ctx.FactsDir, t.ID+".facts.json")
	facts, err := replaybuild.ReadFactsFile(factsPath)
	if err != nil {
		base.Erreur = fmt.Errorf("faits du match : %w", err)
		return base
	}

	cuissonHead, err := bakeTemoin(ctx.BinHead, ctx.WorkRoot, ctx.LockRoot, ctx.TitleSlug, facts)
	if err != nil {
		base.Erreur = fmt.Errorf("cuisson HEAD : %w", err)
		return base
	}
	base.Duree = cuissonHead.Duree

	refPath, err := ctx.resoudreReference(facts)
	if err != nil {
		if errors.Is(err, errAbsentDuParc) {
			slog.Warn("replay-corpus-gate: aucun artefact de reference — temoin ignore",
				"temoin", t.ID, "err", err)
			base.Absent, base.AbsentCause = true, err.Error()
			return base
		}
		base.Erreur = fmt.Errorf("cuisson reference : %w", err)
		return base
	}

	rap, err := compareTemoin(refPath, cuissonHead.ArtifactPath)
	if err != nil {
		base.Erreur = fmt.Errorf("comparaison : %w", err)
		return base
	}
	base.SchemaReference, base.SchemaHEAD, base.Gains, base.Pertes, base.PertesDetail = bilanDepuisRapport(rap)
	return base
}

// resoudreReference rend le chemin de l'artefact de REFERENCE — celui deja cuit dans le parc
// (mode parc, lecture seule) ou une cuisson fraiche a la base (mode base).
func (ctx temoinContexte) resoudreReference(facts replaybuild.FactsFile) (string, error) {
	if ctx.Reference == "parc" {
		refPath := referenceArtifactPath(ctx.ParcRoot, ctx.TitleSlug, facts.MatchID)
		if _, err := os.Stat(refPath); err != nil {
			return "", fmt.Errorf("%w : %s (%v)", errAbsentDuParc, refPath, err)
		}
		return refPath, nil
	}
	cuissonBase, err := bakeTemoin(ctx.BinBase, ctx.WorkRootBase, ctx.LockRoot, ctx.TitleSlug, facts)
	if err != nil {
		return "", err
	}
	return cuissonBase.ArtifactPath, nil
}
