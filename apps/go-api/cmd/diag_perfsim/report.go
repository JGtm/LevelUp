package main

// report.go — rendu Markdown du rapport de simulation (item B0.4).

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/halo_infinite"
)

// chainCompareRow confronte, sur le MÊME ensemble de matchs (celui de la chaîne
// du NOUVEAU régime), les notes des deux régimes.
type chainCompareRow struct {
	Chain      string
	N          int
	NScoredOld int
	NScoredNew int
	OldNotes   []float64
	NewNotes   map[float64][]float64
}

func chainCompare(pr *playerResult) []*chainCompareRow {
	byChain := map[string]*chainCompareRow{}
	ref := pr.NewByW[ospmReference]
	for id, sm := range ref.Matches {
		row := byChain[sm.Chain]
		if row == nil {
			row = &chainCompareRow{Chain: sm.Chain, NewNotes: map[float64][]float64{}}
			byChain[sm.Chain] = row
		}
		row.N++
		if old := pr.Actual.Matches[id]; old != nil && old.Note != nil {
			row.NScoredOld++
			row.OldNotes = append(row.OldNotes, *old.Note)
		}
		if sm.Note != nil {
			row.NScoredNew++
		}
		for _, w := range ospmVariants {
			if v := pr.NewByW[w].Matches[id]; v != nil && v.Note != nil {
				row.NewNotes[w] = append(row.NewNotes[w], *v.Note)
			}
		}
	}
	out := make([]*chainCompareRow, 0, len(byChain))
	for _, r := range byChain {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].N > out[j].N })
	return out
}

func writeReport(path string, results []*playerResult) error {
	var b strings.Builder
	all := collectWitnesses(results)

	secHeader(&b, results)
	secDedup(&b, results)
	secDistributions(&b, results)
	secPurge(&b, results)
	secWitnesses(&b, all)
	secGate(&b, results)
	secDecision(&b, results, all)
	secSubModes(&b, results)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func secHeader(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "# Rapport de simulation — note de performance, scission ranked + metrique objectif\n\n")
	fmt.Fprintf(b, "> Lot 0 du plan `.ai/PLAN_PERF_NOTE_OBJECTIFS.md`. Genere par `apps/go-api/cmd/diag_perfsim`\n")
	fmt.Fprintf(b, "> (outil jetable, DB ouvertes en `access_mode=read_only`, aucune ecriture).\n")
	fmt.Fprintf(b, "> Donnees reelles du checkout principal, 4 joueurs suivis. AUCUN code produit modifie.\n\n")

	fmt.Fprintf(b, "## Methode\n\n")
	fmt.Fprintf(b, "- **Univers** : SQL de `loadHistoryForPerf` (performance_helpers.go:154-172) a deux\n")
	fmt.Fprintf(b, "  differences pres — la clause `AND COALESCE(mp.outcome,0) != 4` est retiree et\n")
	fmt.Fprintf(b, "  `outcome` est projete, pour pouvoir classer les notes orphelines. Le filtre est\n")
	fmt.Fprintf(b, "  reapplique en Go : l'ensemble scorable est identique a la production.\n")
	fmt.Fprintf(b, "- **Exclusions** : `skill.LoadExcludedMatchIDs` repliquee verbatim.\n")
	fmt.Fprintf(b, "- **Moteur** : replique de `computeRelativePerformanceScore` — fenetre %d par chaine,\n", windowSize)
	fmt.Fprintf(b, "  seuil `MinMatchesPerChainForRelative`=10, percentiles ponderes, `dpm_deaths` inversee,\n")
	fmt.Fprintf(b, "  renormalisation par la somme des poids presents, arrondi 0.1, mode `force`.\n")
	fmt.Fprintf(b, "  `skill.RelativeWeights`, `skill.ComputeCombatYield`, `skillchain.ClassifyLUSRChain`,\n")
	fmt.Fprintf(b, "  `analysis.CombatEfficiency` et `analysis.NormalizeModeLabel` sont IMPORTES du code produit.\n")
	fmt.Fprintf(b, "- **`medal_exploit` est ABSENT de la simulation** : le chemin post-sync passe `nil` pour\n")
	fmt.Fprintf(b, "  `medalExploitByMatch` (performance.go:257), la simulation fait de meme. Son poids (0.06)\n")
	fmt.Fprintf(b, "  est donc redistribue ; l'effet est quantifie a la section Gate.\n")
	fmt.Fprintf(b, "- **Regimes compares** : ACTUEL (chaine `ranked` unique, `RelativeWeights` partout) vs\n")
	fmt.Fprintf(b, "  NOUVEAU (`ranked_slayer`/`ranked_objectif` + profil objectif avec ospm sur\n")
	fmt.Fprintf(b, "  `arena_objectif`/`ranked_objectif`), en 3 variantes de poids ospm : 0.08 / 0.12 / 0.16.\n\n")

	fmt.Fprintf(b, "## Corpus\n\n")
	fmt.Fprintf(b, "| Joueur | Univers | DNF (outcome=4) | Exclus | Scorables | Notes ACTUEL | Notes NOUVEAU | Matchs couverts PSA |\n")
	fmt.Fprintf(b, "|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, pr := range results {
		fmt.Fprintf(b, "| %s | %d | %d | %d | %d | %d | %d | %d |\n",
			pr.Player.Gamertag, len(pr.Universe), pr.DNFCount, pr.ExcludedCnt, len(pr.Scorable),
			countScored(pr.Actual), countScored(pr.NewByW[ospmReference]), pr.PSA.MatchesCovered)
	}
	fmt.Fprintf(b, "\n")
}

func secDedup(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "## B0.2 — Regle de dedup `personal_score_awards` retenue\n\n")
	fmt.Fprintf(b, "**Regle retenue** : par `(match_id, xuid)`, ne garder QUE les lignes de la generation\n")
	fmt.Fprintf(b, "MAXIMALE (`generation_id`), puis exclure les `is_tombstone`. C'est exactement la\n")
	fmt.Fprintf(b, "semantique de la vue `personal_score_awards_latest` (DENSE_RANK, ADR 0026,\n")
	fmt.Fprintf(b, "`steps_player_append_only_personal_score_awards.go:48-55`) — pas une regle inventee.\n")
	fmt.Fprintf(b, "La somme se fait sur `award_score` des lignes `award_category='objective'` retenues,\n")
	fmt.Fprintf(b, "divisee par les minutes jouees (meme repli 600 s que le reste du moteur).\n\n")
	fmt.Fprintf(b, "**La dedup n'est pas theorique** : des generations multiples existent en base.\n\n")

	fmt.Fprintf(b, "| Joueur | Lignes PSA | Lignes d'un autre xuid | Tombstones | Lignes de generation perimee | Matchs multi-generation | Somme objectif dedupliquee | Somme objectif SANS dedup |\n")
	fmt.Fprintf(b, "|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, pr := range results {
		s := pr.PSA
		fmt.Fprintf(b, "| %s | %d | %d | %d | %d | %d | %.0f | %.0f |\n",
			pr.Player.Gamertag, s.RowsTotal, s.RowsOtherXUID, s.RowsTombstone,
			s.RowsStaleGen, s.PairsMultiGen, s.ObjScoreLatest, s.ObjScoreNaiveAll)
	}
	fmt.Fprintf(b, "\n")
	fmt.Fprintf(b, "**Presence d'ospm** : ospm n'existe QUE si le match a une couverture PSA (au moins une\n")
	fmt.Fprintf(b, "ligne retenue). Un match couvert avec 0 point objectif vaut ospm = 0 (valeur legitime :\n")
	fmt.Fprintf(b, "le joueur n'a rien fait a l'objectif) ; un match SANS ligne PSA a une metrique ABSENTE\n")
	fmt.Fprintf(b, "(poids redistribue). Confondre les deux reviendrait a noter 0 une absence de donnee.\n\n")
	fmt.Fprintf(b, "**Piege majeur rencontre** (cf. Decouvertes du CR) : sur la DB `XxDaemonGamerxX`,\n")
	fmt.Fprintf(b, "l'index `idx_psa_match` est INCOHERENT — `WHERE match_id = '05fffb2a-...'` rend 2 lignes\n")
	fmt.Fprintf(b, "la ou le scan complet en rend 4. Toutes les lectures PSA de l'outil se font donc en scan\n")
	fmt.Fprintf(b, "complet SANS predicat, la selection etant faite en Go.\n\n")
}

func secDistributions(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "## B0.1 / B0.3 — Distributions par joueur x chaine (chaines du NOUVEAU regime)\n\n")
	fmt.Fprintf(b, "Les deux regimes sont evalues sur le MEME ensemble de matchs (celui de la chaine du\n")
	fmt.Fprintf(b, "nouveau regime), ce qui rend les medianes directement comparables. `n scorees` differe\n")
	fmt.Fprintf(b, "entre regimes uniquement par l'effet du seuil de 10 matchs sur une chaine scindee.\n\n")

	for _, pr := range results {
		fmt.Fprintf(b, "### %s\n\n", pr.Player.Gamertag)
		fmt.Fprintf(b, "| Chaine | n matchs | n scorees ACTUEL | n scorees NOUVEAU | med ACTUEL | p10/p90 ACTUEL | med 0.08 | med 0.12 | med 0.16 | p10/p90 (0.12) |\n")
		fmt.Fprintf(b, "|---|---:|---:|---:|---:|---|---:|---:|---:|---|\n")
		for _, row := range chainCompare(pr) {
			ref := row.NewNotes[ospmReference]
			fmt.Fprintf(b, "| `%s` | %d | %d | %d | %s | %s | %s | %s | %s | %s |\n",
				row.Chain, row.N, row.NScoredOld, row.NScoredNew,
				f1(quantile(row.OldNotes, 0.5)), span(row.OldNotes),
				f1(quantile(row.NewNotes[0.08], 0.5)), f1(quantile(ref, 0.5)),
				f1(quantile(row.NewNotes[0.16], 0.5)), span(ref))
		}
		fmt.Fprintf(b, "\n")
	}
}

func secPurge(b *strings.Builder, results []*playerResult) {
	fmt.Fprintf(b, "## D-D — Purge des notes orphelines\n\n")
	fmt.Fprintf(b, "Une note ne doit exister QUE pour un match qualifie : non-DNF, non-exclu, et au-dela du\n")
	fmt.Fprintf(b, "10e match de sa chaine (nouveau regime). Comptes des notes STOCKEES qui disparaissent,\n")
	fmt.Fprintf(b, "par cause (priorite DNF > exclu > sous-seuil > hors univers).\n\n")
	fmt.Fprintf(b, "| Joueur | Notes stockees | Conservees | Purgees | dont DNF | dont exclus | dont sous-seuil | dont hors univers |\n")
	fmt.Fprintf(b, "|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, pr := range results {
		p := pr.Purge
		fmt.Fprintf(b, "| %s | %d | %d | **%d** | %d | %d | %d | %d |\n",
			pr.Player.Gamertag, p.StoredScored, p.Kept, p.Total(),
			p.DNF, p.Excluded, p.BelowThreshold, p.OutOfUniverse)
	}
	fmt.Fprintf(b, "\n")

	fmt.Fprintf(b, "Repartition des notes purgees selon la chaine STOCKEE (celle qui disparait) :\n\n")
	fmt.Fprintf(b, "| Joueur | Chaine stockee | Notes purgees |\n|---|---|---:|\n")
	for _, pr := range results {
		keys := make([]string, 0, len(pr.Purge.ByChainStored))
		for k := range pr.Purge.ByChainStored {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return pr.Purge.ByChainStored[keys[i]] > pr.Purge.ByChainStored[keys[j]]
		})
		for _, k := range keys {
			fmt.Fprintf(b, "| %s | `%s` | %d |\n", pr.Player.Gamertag, k, pr.Purge.ByChainStored[k])
		}
	}
	fmt.Fprintf(b, "\n")
}

// missedObjectiveLabels — sous-modes objectif que la liste de classify.go:78-80
// ne reconnait PAS aujourd'hui (constat, pas correctif : le lot 0 ne touche a
// aucun fichier produit). `arena` vient de pair_names inverses type
// `Strongholds:Arena on Behemoth`.
var missedObjectiveLabels = map[string]bool{
	"neutral bomb": true, "one bomb": true, "neutral bomb squad": true,
	"vip": true, "ctf 3 captures": true, "arena": true,
}

// missClassified compte les matchs d'un mode objectif evident tombes en famille slayer.
func missClassified(rows []*subModeRow) int {
	n := 0
	for _, r := range rows {
		if missedObjectiveLabels[r.Label] && !isObjectiveChain(r.Chain) {
			n += r.N
		}
	}
	return n
}

// subModeRow agrege un sous-mode normalise et la famille qui lui est attribuee.
type subModeRow struct {
	Label  string
	Chain  string
	N      int
	Sample string
}

// secSubModes expose la classification effective des sous-modes des categories
// ou la liste objectif est consultee (Assassin et Ranked). Sert de piece de
// verification au lot 1 : toute ligne « objectif evident classee slayer » est une
// lacune de la liste de classify.go:78-80.
func secSubModes(b *strings.Builder, results []*playerResult) {
	type key struct{ label, chain string }
	agg := map[key]*subModeRow{}
	for _, pr := range results {
		for i := range pr.Universe {
			m := &pr.Universe[i]
			cat := halo_infinite.InferModeCategoryFromPairName(m.PairName)
			if cat != halo_infinite.ModeCategoryAssassin && cat != halo_infinite.ModeCategoryRanked {
				continue
			}
			k := key{strings.ToLower(analysis.NormalizeModeLabel(m.PairName)), chainSplit(m)}
			r := agg[k]
			if r == nil {
				r = &subModeRow{Label: k.label, Chain: k.chain, Sample: m.PairName}
				agg[k] = r
			}
			r.N++
		}
	}
	rows := make([]*subModeRow, 0, len(agg))
	for _, r := range agg {
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].N > rows[j].N })

	fmt.Fprintf(b, "## Annexe — classification effective des sous-modes (categories Assassin + Ranked)\n\n")
	fmt.Fprintf(b, "Sous-modes normalises par `analysis.NormalizeModeLabel`, avec la famille que leur\n")
	fmt.Fprintf(b, "attribue le NOUVEAU regime. Corpus des 4 joueurs, univers complet.\n\n")
	fmt.Fprintf(b, "**Lacunes de la liste objectif reperees ici** (PRE-EXISTANTES : elles affectent deja\n")
	fmt.Fprintf(b, "`lusrChainForAssassin` et donc les chaines LUSR d'aujourd'hui — NON corrigees par ce\n")
	fmt.Fprintf(b, "lot 0, a statuer au lot 1) : %d matchs d'un mode objectif evident tombent en famille\n", missClassified(rows))
	fmt.Fprintf(b, "slayer — Assaut (`neutral bomb`, `one bomb`, `neutral bomb squad`), `vip`,\n")
	fmt.Fprintf(b, "`ctf 3 captures`, et surtout `arena` (14 matchs dont le pair_name est INVERSE :\n")
	fmt.Fprintf(b, "`Strongholds:Arena on Behemoth` — le mode est a GAUCHE du deux-points, la\n")
	fmt.Fprintf(b, "normalisation retient donc « Arena » comme sous-mode).\n\n")
	fmt.Fprintf(b, "| Sous-mode normalise | Famille attribuee | n matchs | Exemple de pair_name |\n")
	fmt.Fprintf(b, "|---|---|---:|---|\n")
	for _, r := range rows {
		fmt.Fprintf(b, "| `%s` | `%s` | %d | %s |\n", r.Label, r.Chain, r.N, safePair(r.Sample))
	}
	fmt.Fprintf(b, "\n")
}

// ── Formatage ───────────────────────────────────────────────────────────────

func f1(v float64) string {
	if math.IsNaN(v) {
		return "-"
	}
	return fmt.Sprintf("%.1f", v)
}

func span(values []float64) string {
	if len(values) == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f / %.1f", quantile(values, 0.10), quantile(values, 0.90))
}
