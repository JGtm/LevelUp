package service

// replay_medal_labels.go — LE LIBELLÉ ET LE VISUEL DE CHAQUE MÉDAILLE DU DOCUMENT, posés à la requête.
//
// CE QUE L'ARTEFACT PORTE ET CE QU'IL NE PORTE PAS. `medals[]` dit qui a gagné quoi et quand, avec le
// nom ANGLAIS de la médaille (`Medal.Name`) : une clé de référentiel, jamais un libellé d'interface.
// Le libellé par langue, la description, l'image et la rareté viennent du référentiel du titre
// (medal_definitions et medal_translations de la base de métadonnées), que l'artefact ne connaît pas
// et qui peut s'améliorer : même règle que les familles de châssis (cf. replay_vehicle_labels.go),
// donc une résolution à la requête, jamais figée dans un artefact déjà cuit.
//
// LES MÊMES LECTURES QUE LES AUTRES PAGES. Le pont nom anglais -> identité est
// [port.MedalNameLookup] ; libellé, description et rareté sont ceux de
// [port.MedalDefinitionsRepository] ; l'image est celle des tuiles de match
// (`static.MedalImage` : PNG, ou découpe de feuille de sprites quand le titre en sert).
//
// UNE TABLE PAR LANGUE, PAS UNE LANGUE PAR REQUÊTE : le document est gros et mis en cache côté client,
// la clé de requête ne porte pas la langue, donc la table porte TOUTES les langues de l'application et
// le client choisit (`pickText`). Un changement de langue ne recharge pas le document.
//
// TITLE-AGNOSTIC : l'adresse de l'image se compose pour le titre du service, sans comparaison de slug.
//
// UNE MÉDAILLE QUE LE RÉFÉRENTIEL NE CONNAÎT PAS N'A AUCUNE ENTRÉE, et chaque nom ainsi écarté est
// journalisé : le lecteur garde le nom du film, jamais un libellé ni le visuel d'une voisine.

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// WithReplayMedals donne au service les deux lectures du référentiel de médailles : le pont nom
// anglais -> identité, et les textes et la rareté par identité et par langue. L'une des deux nulle =
// aucune table, jamais une table à moitié résolue.
func WithReplayMedals(names port.MedalNameLookup, defs port.MedalDefinitionsRepository) ReplayOption {
	return func(s *replayService) { s.medalNames, s.medalDefs = names, defs }
}

// resolveMedalLabels pose `doc.MedalLabels` : une entrée par NOM de médaille du document que le
// référentiel connaît.
//
// SANS MÉDAILLE, AUCUNE TABLE (pas une table vide) ; sans entrée résolue, aucune table non plus. Une
// lecture du référentiel en échec est journalisée, puis le document part sans la table : le rejeu
// dégrade, il ne casse pas.
func (s *replayService) resolveMedalLabels(ctx context.Context, doc *replaydoc.ReplayDocument) {
	if doc == nil {
		return
	}
	names := medalNamesUsed(doc.Medals)
	if len(names) == 0 {
		return
	}
	if s.medalNames == nil || s.medalDefs == nil {
		slog.InfoContext(ctx, "rejeu 2D : aucune source de médailles — médailles servies sans libellé ni visuel",
			"noms", len(names), "titleSlug", s.titleSlug)
		return
	}
	metas, err := s.medalNames.LookupMedalMetaByName(ctx, names)
	if err != nil {
		slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError),
			"rejeu 2D : référentiel des médailles illisible — médailles servies sans libellé ni visuel",
			"noms", len(names), "titleSlug", s.titleSlug, "err", err)
		return
	}
	rows := s.medalRowsByLocale(ctx, medalIDsOf(metas))
	labels := make(map[string]replaydoc.MedalLabel, len(metas))
	var unknown []string
	for _, name := range names {
		meta, ok := metas[name]
		if !ok {
			unknown = append(unknown, name)
			continue
		}
		labels[name] = s.medalLabelOf(meta.MedalNameID, rows)
	}
	if len(unknown) > 0 {
		slog.WarnContext(ctx, "rejeu 2D : médailles absentes du référentiel — servies sans libellé ni visuel",
			"noms", unknown, "titleSlug", s.titleSlug)
	}
	if len(labels) > 0 {
		doc.MedalLabels = labels
	}
}

// medalNamesUsed relève les noms de médaille du document, uniques et TRIÉS (l'ordre d'une map ne se
// publie pas) ; une médaille sans nom (indice absent de la table du décodeur) n'a rien à résoudre.
func medalNamesUsed(medals []replaydoc.Medal) []string {
	seen := map[string]bool{}
	for _, m := range medals {
		if m.Name != "" {
			seen[m.Name] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// medalIDsOf rend les identités résolues, sans doublon (deux noms ne partagent pas une identité, mais
// la lecture ne doit pas dépendre de cette hypothèse).
func medalIDsOf(metas map[string]domain.MedalNameMeta) []int64 {
	seen := make(map[int64]bool, len(metas))
	out := make([]int64, 0, len(metas))
	for _, m := range metas {
		if !seen[m.MedalNameID] {
			seen[m.MedalNameID] = true
			out = append(out, m.MedalNameID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// medalRowsByLocale lit libellé, description et rareté de chaque identité, UNE lecture par langue de
// l'application. Une langue dont la lecture échoue est journalisée et manque à la table : le client
// replie alors sur la langue par défaut, comme pour tout nom de jeu.
func (s *replayService) medalRowsByLocale(ctx context.Context, ids []int64) map[i18n.Locale]map[int64]port.MedalDefinitionRow {
	out := map[i18n.Locale]map[int64]port.MedalDefinitionRow{}
	if len(ids) == 0 {
		return out
	}
	for _, l := range i18n.Locales() {
		rows, err := s.medalDefs.LookupByIDs(ctx, ids, string(l))
		if err != nil {
			slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelWarn),
				"rejeu 2D : textes de médailles illisibles — langue absente de la table",
				"locale", l, "titleSlug", s.titleSlug, "err", err)
			continue
		}
		out[l] = rows
	}
	return out
}

// medalLabelOf compose l'entrée d'UNE médaille : ses textes par langue, sa rareté, et son visuel.
//
// La rareté n'a qu'une valeur : elle se lit dans la première langue de [i18n.Locales] qui nomme la
// médaille et la porte (la langue par défaut d'abord), jamais dans l'ordre, non fixé, d'un parcours de
// table : deux requêtes sur la même base rendent la même rareté.
func (s *replayService) medalLabelOf(id int64, rows map[i18n.Locale]map[int64]port.MedalDefinitionRow) replaydoc.MedalLabel {
	lbl := replaydoc.MedalLabel{Text: i18n.Text{}}
	desc := i18n.Text{}
	for _, l := range i18n.Locales() {
		row, ok := rows[l][id]
		if !ok {
			continue
		}
		if row.Label != "" {
			lbl.Text[l] = row.Label
		}
		if row.Description != "" {
			desc[l] = row.Description
		}
		if lbl.Difficulty == "" {
			lbl.Difficulty = row.Difficulty
		}
	}
	if len(lbl.Text) == 0 {
		lbl.Text = nil
	}
	if len(desc) > 0 {
		lbl.Description = &replaydoc.MedalDescription{Text: desc}
	}
	png, sprite := static.MedalImage(s.titleSlug, id)
	if sprite != nil {
		lbl.Sprite = &replaydoc.MedalSprite{
			Sheet: sprite.SheetURL, Left: sprite.Left, Top: sprite.Top, Width: sprite.Width, Height: sprite.Height,
		}
		return lbl
	}
	lbl.Img = png
	return lbl
}
