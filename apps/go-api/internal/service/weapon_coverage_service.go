// Package service — weapon_coverage_service.go : la couverture de résolution d'arme (page d'administration) —
// quelle part des frags se résout à une arme nommée.
//
// DÉCISION A2.7 (2026-09-01) : la page SURVIT à la bascule, elle change de source. Sa question — « quelle part
// des frags se résout à une arme nommée ? » — reste la bonne au moment où l'on change de mesure ; c'est même la
// seule surface qui la pose en continu. Sur un titre à décodeur de film, l'unité comptée n'est plus un
// identifiant d'arme mais un TAG de source de dégât, et « non résolu » veut dire « aucune clé de registre ».
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

const weaponCoverageTopUnresolved = 15

// WeaponCoverageDeps assemble ce que le service ne sait pas construire lui-même.
type WeaponCoverageDeps struct {
	// Open ouvre les lectures du titre ; release les rend. Une erreur est rendue telle quelle.
	Open func(ctx context.Context, titleSlug string) (repo port.WeaponCoverageRepo, release func(), err error)
	// Classifier rend le classificateur de source de dégât du titre ; nil quand le titre n'a pas de
	// décodeur de film (la couverture se compte alors par identifiant d'arme).
	Classifier func(titleSlug string) port.KillSourceClassifier
}

// WeaponCoverageService calcule la couverture de résolution d'arme d'un titre.
type WeaponCoverageService struct {
	deps WeaponCoverageDeps
}

// NewWeaponCoverageService construit le service.
func NewWeaponCoverageService(deps WeaponCoverageDeps) *WeaponCoverageService {
	return &WeaponCoverageService{deps: deps}
}

// WeaponCoverage calcule la couverture de résolution d'arme d'un titre.
func (s *WeaponCoverageService) WeaponCoverage(ctx context.Context, titleSlug string) (domain.AdminWeaponCoverage, error) {
	resp := domain.AdminWeaponCoverage{
		TitleSlug:     titleSlug,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		TopUnresolved: []domain.AdminWeaponCoverageItem{},
	}
	repo, release, err := s.deps.Open(ctx, titleSlug)
	if err != nil {
		return resp, err
	}
	defer release()

	if classifier := s.deps.Classifier(titleSlug); classifier != nil {
		return weaponCoverageFromSource(ctx, repo, classifier, resp)
	}
	return weaponCoverageFromIDs(ctx, repo, titleSlug, resp)
}

// weaponCoverageFromIDs compte la couverture par identifiant d'arme : les identifiants distincts (hors
// sentinelles 0/1/2) et leurs frags, puis leur classification par le registre et les libellés.
func weaponCoverageFromIDs(ctx context.Context, repo port.WeaponCoverageRepo, titleSlug string, resp domain.AdminWeaponCoverage) (domain.AdminWeaponCoverage, error) {
	counts, err := repo.KillsByWeaponID(ctx)
	if err != nil {
		return resp, err
	}
	killsByID := make(map[int64]int, len(counts))
	ids := make([]int64, 0, 64)
	for _, c := range counts {
		killsByID[c.WeaponID] = c.Kills
		ids = append(ids, c.WeaponID)
	}
	resp.DistinctWeapons = len(ids)
	if len(ids) == 0 {
		return resp, nil
	}

	// Classification via metadata (registre + weapon_labels). Sans metadata : tout est non résolu.
	inReg, inLabel := map[int64]bool{}, map[int64]bool{}
	if repo.HasMetadata() {
		inReg, inLabel = repo.ClassifyWeaponIDs(ctx, titleSlug, ids)
	}
	for _, id := range ids {
		switch {
		case inReg[id]:
			resp.ResolvedRegistry++
		case inLabel[id]:
			resp.ResolvedLabel++
		default:
			resp.Unresolved++
			resp.TopUnresolved = append(resp.TopUnresolved,
				domain.AdminWeaponCoverageItem{WeaponID: strconv.FormatInt(id, 10), Kills: killsByID[id]})
		}
	}
	sort.Slice(resp.TopUnresolved, func(i, j int) bool { return resp.TopUnresolved[i].Kills > resp.TopUnresolved[j].Kills })
	if len(resp.TopUnresolved) > weaponCoverageTopUnresolved {
		resp.TopUnresolved = resp.TopUnresolved[:weaponCoverageTopUnresolved]
	}
	if resp.DistinctWeapons > 0 {
		d := float64(resp.DistinctWeapons)
		resp.CoveragePercent = float64(resp.ResolvedRegistry+resp.ResolvedLabel) / d * 100
		resp.RegistryPercent = float64(resp.ResolvedRegistry) / d * 100
	}
	return resp, nil
}

// weaponCoverageFromSource calcule la couverture pour un titre dont l'arme vient de la SOURCE DE DÉGÂT du
// film : l'unité comptée est le tag `jpt!`, et « résolu » veut dire « le classificateur du titre lui donne
// une clé de registre ».
//
// `ResolvedLabel` reste à zéro, et ce n'est pas un oubli : sur cette voie il n'existe pas de résolution « par
// libellé seul » — un tag donne une clé de registre, ou rien.
func weaponCoverageFromSource(
	ctx context.Context,
	repo port.WeaponCoverageRepo,
	classifier port.KillSourceClassifier,
	resp domain.AdminWeaponCoverage,
) (domain.AdminWeaponCoverage, error) {
	counts, err := repo.KillsBySourceTag(ctx)
	if err != nil {
		return resp, err
	}
	cles := map[string]bool{}
	for _, c := range counts {
		resp.DistinctWeapons++
		key, ok := classifier.KillSourceRegistryKey(c.SourceTag)
		if !ok {
			resp.Unresolved++
			resp.TopUnresolved = append(resp.TopUnresolved, domain.AdminWeaponCoverageItem{
				WeaponID: fmt.Sprintf("0x%08x", c.SourceTag), Kills: c.Kills,
			})
			continue
		}
		resp.ResolvedRegistry++
		cles[key] = true
	}
	// Une clé que le REGISTRE ne connaît pas est aussi peu résolue qu'un tag sans clé : le classificateur a
	// beau nommer l'objet, rien ne l'affichera. On le signale.
	if repo.HasMetadata() {
		signalKeysOutsideRegistry(ctx, repo, resp.TitleSlug, cles)
	}
	sort.Slice(resp.TopUnresolved, func(i, j int) bool {
		return resp.TopUnresolved[i].Kills > resp.TopUnresolved[j].Kills
	})
	if len(resp.TopUnresolved) > weaponCoverageTopUnresolved {
		resp.TopUnresolved = resp.TopUnresolved[:weaponCoverageTopUnresolved]
	}
	if resp.DistinctWeapons > 0 {
		d := float64(resp.DistinctWeapons)
		resp.CoveragePercent = float64(resp.ResolvedRegistry) / d * 100
		resp.RegistryPercent = resp.CoveragePercent
	}
	return resp, nil
}

// signalKeysOutsideRegistry journalise les clés rendues par le classificateur que le registre d'armes ne
// porte pas. Best-effort : la page reste servie, mais l'anomalie ne passe pas en silence (règle « jamais
// d'erreur avalée »).
func signalKeysOutsideRegistry(ctx context.Context, repo port.WeaponCoverageRepo, slug string, cles map[string]bool) {
	if len(cles) == 0 {
		return
	}
	connues, err := repo.RegistryWeaponKeys(ctx, slug)
	if err != nil {
		slog.WarnContext(ctx, "couverture d'arme: registre illisible", "title", slug, "err", err)
		return
	}
	manquantes := make([]string, 0)
	for k := range cles {
		if !connues[k] {
			manquantes = append(manquantes, k)
		}
	}
	if len(manquantes) > 0 {
		sort.Strings(manquantes)
		slog.WarnContext(ctx, "couverture d'arme: cles nommees par le film mais absentes du registre",
			"title", slug, "cles", strings.Join(manquantes, ", "))
	}
}
