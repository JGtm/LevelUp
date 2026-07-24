// Package api — post_sync_medal_first_earned.go : émission de la notification
// « médaille inédite » (V72-20) après une sync. Détection par diff d'ensemble :
// les medal_name_id présents dans le snapshot APRÈS mais absents de l'AVANT sont
// des médailles décrochées pour la première fois.
//
// Best-effort STRICT : toute erreur est loguée et n'interrompt JAMAIS le flux de
// sync — une notif ratée ne doit pas faire échouer une synchronisation.
//
// Garde cold-start (esprit de snapshotLooksCold) : si le snapshot AVANT ne porte
// AUCUNE médaille (première sync, ou lecture medals_earned indisponible), on sème
// silencieusement — sinon tout l'historique apparaîtrait « inédit » au premier
// cycle (burst). Anti-rafale : au-delà de medalFirstEarnedRecapThreshold médailles
// inédites dans un même cycle, une seule notification récapitulative est émise.
package wire

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

// medalFirstEarnedRecapThreshold : au-delà de ce nombre de médailles inédites dans
// un même cycle de sync, une notification récapitulative « N médailles inédites »
// remplace les notifs nominatives (anti-rafale, cohérent avec l'esprit du
// coalescing et de maxPlausibleCounterDelta). Un backfill peut en ramener beaucoup ;
// la garde cold-start couvre le tout-premier cas, ce seuil couvre le reste.
const medalFirstEarnedRecapThreshold = 3

// paramKeyMedalNameFR / paramKeyMedalNameEN : clés Params du nom localisé de la
// médaille. Le serveur résout les DEUX langues (patron title_fr/title_en de
// milestone_unlocked) ; le front choisit selon la locale. Jamais d'id brut affiché.
const (
	paramKeyMedalNameFR = "medal_name_fr"
	paramKeyMedalNameEN = "medal_name_en"
)

// medalsRouteSuffix : sous-chemin joueur de la page Médailles (cible des notifs
// « médaille inédite »). Route front RÉELLE (career/medals.tsx) → zéro hop.
const medalsRouteSuffix = "career/medals"

// medalNamePair porte le nom localisé (FR + EN) d'une médaille résolu côté serveur.
type medalNamePair struct {
	FR string
	EN string
}

// medalNameResolver résout les noms localisés d'un ensemble de medal_name_id.
// Interface locale pour rendre emitMedalFirstEarned testable sans metadata DB.
type medalNameResolver interface {
	MedalNames(ctx context.Context, ids []int64) map[int64]medalNamePair
}

// pdbMedalNamer résout les noms FR + EN via MedalDefinitionsRepo.LookupByIDs (deux
// lectures locale-aware, chaîne de fallback partagée medal_label_resolve.go).
type pdbMedalNamer struct {
	repo *duckdb.MedalDefinitionsRepo
}

// newMedalNamerForPDB construit un résolveur de noms scopé sur la metadata du titre.
// Retourne nil si pdb (ou sa metadata) est absent — l'émission est alors sautée
// proprement en amont.
func newMedalNamerForPDB(pdb *duckdb.PlayerDB) medalNameResolver {
	if pdb == nil || pdb.Metadata == nil {
		return nil
	}
	return &pdbMedalNamer{repo: duckdb.NewMedalDefinitionsRepo(pdb)}
}

// MedalNames résout FR + EN pour chaque id ; une lecture en échec est loguée et
// laisse un nom vide (emitSingleMedal saute alors la médaille — jamais d'id brut).
func (m *pdbMedalNamer) MedalNames(ctx context.Context, ids []int64) map[int64]medalNamePair {
	out := make(map[int64]medalNamePair, len(ids))
	fr, err := m.repo.LookupByIDs(ctx, ids, "fr")
	if err != nil {
		slog.WarnContext(ctx, "post_sync: medal names lookup FR", "err", err)
	}
	en, err := m.repo.LookupByIDs(ctx, ids, "en")
	if err != nil {
		slog.WarnContext(ctx, "post_sync: medal names lookup EN", "err", err)
	}
	for _, id := range ids {
		out[id] = medalNamePair{FR: fr[id].Label, EN: en[id].Label}
	}
	return out
}

// emitMedalFirstEarned détecte les médailles décrochées pour la première fois
// (after \ before) et émet les notifications correspondantes. Cf. en-tête de
// fichier pour la garde cold-start et l'anti-rafale.
func emitMedalFirstEarned(
	ctx context.Context,
	emitter notifications.Emitter,
	namer medalNameResolver,
	titleSlug, slug string,
	before, after *PlayerSnapshot,
) {
	if emitter == nil || namer == nil || before == nil || after == nil {
		return
	}
	// Garde cold-start : aucun repère médaille avant → premier passage / lecture
	// indisponible. On sème silencieusement (les médailles serviront de baseline au
	// prochain cycle) plutôt que de notifier tout l'historique.
	if len(before.EarnedMedalIDs) == 0 {
		slog.DebugContext(ctx, "post_sync: médailles amorcées sans notification (cold-start)", "slug", slug)
		return
	}
	newIDs := newlyEarnedMedalIDs(before, after)
	if len(newIDs) == 0 {
		return
	}
	if len(newIDs) > medalFirstEarnedRecapThreshold {
		emitMedalRecap(ctx, emitter, titleSlug, slug, len(newIDs))
		return
	}
	names := namer.MedalNames(ctx, newIDs)
	for _, id := range newIDs {
		emitSingleMedal(ctx, emitter, titleSlug, slug, id, names[id])
	}
}

// newlyEarnedMedalIDs = after \ before, trié (émission déterministe / tests stables).
func newlyEarnedMedalIDs(before, after *PlayerSnapshot) []int64 {
	var newIDs []int64
	for id := range after.EarnedMedalIDs {
		if _, ok := before.EarnedMedalIDs[id]; !ok {
			newIDs = append(newIDs, id)
		}
	}
	sort.Slice(newIDs, func(i, j int) bool { return newIDs[i] < newIDs[j] })
	return newIDs
}

// emitMedalRecap émet la notification récapitulative « N médailles inédites ».
func emitMedalRecap(ctx context.Context, emitter notifications.Emitter, titleSlug, slug string, count int) {
	err := emitter.Emit(ctx, notifications.EmitInput{
		Category:    notifications.CategoryMedalFirstEarned,
		Severity:    notifications.SeveritySuccess,
		TitleKey:    "notif.medal_first_earned.recap.title",
		BodyKey:     "notif.medal_first_earned.recap.body",
		Params:      map[string]any{paramKeyCount: count},
		TargetRoute: notifications.PlayerTargetRoute(titleSlug, slug, medalsRouteSuffix),
		Source:      postSyncSource,
	})
	if err != nil {
		slog.WarnContext(ctx, "post_sync: medal_first_earned recap emit", "slug", slug, "count", count, "err", err)
	}
}

// emitSingleMedal émet une notification « médaille inédite » nominative. Si le nom
// localisé est introuvable dans les DEUX langues, on saute (jamais d'id brut). Un
// nom présent dans une seule langue est réutilisé pour l'autre (dégradation propre).
func emitSingleMedal(
	ctx context.Context,
	emitter notifications.Emitter,
	titleSlug, slug string,
	id int64,
	name medalNamePair,
) {
	if name.FR == "" && name.EN == "" {
		slog.WarnContext(ctx, "post_sync: medal_first_earned nom introuvable — émission sautée",
			"slug", slug, "medal_id", id)
		return
	}
	fr, en := name.FR, name.EN
	if fr == "" {
		fr = en
	}
	if en == "" {
		en = fr
	}
	err := emitter.Emit(ctx, notifications.EmitInput{
		Category: notifications.CategoryMedalFirstEarned,
		Severity: notifications.SeveritySuccess,
		TitleKey: "notif.medal_first_earned.title",
		BodyKey:  "notif.medal_first_earned.body",
		Params: map[string]any{
			paramKeyMedalNameFR: fr,
			paramKeyMedalNameEN: en,
		},
		TargetRoute: notifications.PlayerTargetRoute(titleSlug, slug, medalsRouteSuffix),
		Source:      postSyncSource,
	})
	if err != nil {
		slog.WarnContext(ctx, "post_sync: medal_first_earned emit", "slug", slug, "medal_id", id, "err", err)
	}
}
