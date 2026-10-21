package scheduler

// data_health_passes.go — LA JAUGE « PASSES PAR MATCH » du contrôle de santé des données.
//
// Pour chaque titre dont la base partagée existe et chaque table du registre de compaction, le
// rapport lignes brutes / lignes gardées (migration.MesurerPasses) est mesuré en LECTURE, une table
// par lecture courte de la base partagée (jamais d'écrivain, jamais de lecture tenue pendant tout
// le cycle : un swap du sync n'attend au plus qu'une table). Trois sorties :
//   - expvar `levelup.compaction.passes_par_match` (clé `<titre>/<table>`) et
//     `levelup.compaction.tables_au_dessus_du_seuil` ;
//   - un WARN par table au-delà de migration.SeuilPassesParMatch ;
//   - le fichier PathResolver.PassesGaugePath(), lu par le script hôte de nuit
//     (scripts/compact-passes-nuit.sh) qui arrête le serveur et compacte si une table dépasse.
//
// « Non mesuré ≠ sain » : une mesure partielle (une table en échec) ne publie RIEN — ni expvar,
// ni fichier. Le fichier précédent reste, son horodatage vieillit, et le script refuse d'agir sur
// une jauge périmée.

import (
	"context"
	"database/sql"
	"expvar"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/atomicfile"
	"levelup/go-api/internal/platform/duckdb"
)

var (
	passesParMatchGauge = expvar.NewMap("levelup.compaction.passes_par_match")
	tablesAuDessusGauge = expvar.NewInt("levelup.compaction.tables_au_dessus_du_seuil")
)

// formatHorodatageJauge : format de `mesure_utc` dans le fichier de jauge (lu par `date -d`).
const formatHorodatageJauge = time.RFC3339

// SharedReaderFunc ouvre une lecture de la base partagée d'un titre et rend sa fonction release
// (à appeler une fois la lecture finie). En production : le provider B-swap du titre.
type SharedReaderFunc func(ctx context.Context, titleSlug string) (*sql.DB, func(), error)

// PassesTitre : la jauge d'une table du registre, pour un titre.
type PassesTitre struct {
	TitleSlug string
	migration.JaugePasses
}

// WithSharedReader branche le lecteur de la base partagée de la jauge des passes. Sans lui, la
// jauge lit par duckdb.OpenReadForQuery (réutilise le handle tenu par le process s'il existe).
func (s *HealthScheduler) WithSharedReader(r SharedReaderFunc) *HealthScheduler {
	s.sharedReader = r
	return s
}

// WithClock remplace l'horloge qui date le fichier de jauge (tests).
func (s *HealthScheduler) WithClock(now func() time.Time) *HealthScheduler {
	if now != nil {
		s.now = now
	}
	return s
}

// auditTitlePasses mesure chaque table du registre pour un titre et range le résultat dans res.
// Une table absente de la base est sautée ; une mesure en échec compte comme sonde en échec ET
// comme table non mesurée (la publication est alors retenue).
func (s *HealthScheduler) auditTitlePasses(ctx context.Context, pr *titlePkg.PathResolver, slug string, res *DataHealthCheckResult) {
	for _, table := range migration.CompactableTables() {
		j, presente, err := s.mesurerUneTable(ctx, pr, slug, table)
		if err != nil {
			slog.WarnContext(ctx, "data_health: jauge des passes, mesure échouée",
				"titleSlug", slug, "table", table, "err", err)
			res.ProbeErrors++
			res.PassesUnmeasured++
			continue
		}
		if presente {
			res.Passes = append(res.Passes, PassesTitre{TitleSlug: slug, JaugePasses: j})
		}
	}
}

func (s *HealthScheduler) mesurerUneTable(ctx context.Context, pr *titlePkg.PathResolver, slug, table string) (migration.JaugePasses, bool, error) {
	db, release, err := s.lecteurShared(ctx, pr, slug)
	if err != nil {
		return migration.JaugePasses{Table: table}, false, fmt.Errorf("lecture de la base partagée: %w", err)
	}
	defer release()
	return migration.MesurerPasses(ctx, db, table)
}

func (s *HealthScheduler) lecteurShared(ctx context.Context, pr *titlePkg.PathResolver, slug string) (*sql.DB, func(), error) {
	if s.sharedReader != nil {
		return s.sharedReader(ctx, slug)
	}
	return duckdb.OpenReadForQuery(pr.SharedDBPath(slug))
}

// publierJaugePasses : expvar, WARN par table au-delà du seuil, fichier de jauge — seulement
// si la mesure est complète.
func (s *HealthScheduler) publierJaugePasses(ctx context.Context, pr *titlePkg.PathResolver, res *DataHealthCheckResult) {
	if res.PassesUnmeasured > 0 {
		slog.WarnContext(ctx, "data_health: jauge des passes non publiée (mesure partielle)",
			"tables_non_mesurees", res.PassesUnmeasured)
		return
	}
	auDessus, rapportMax := 0, 1.0
	passesParMatchGauge.Init()
	for _, p := range res.Passes {
		v := new(expvar.Float)
		v.Set(p.Rapport())
		passesParMatchGauge.Set(p.TitleSlug+"/"+p.Table, v)
		rapportMax = max(rapportMax, p.Rapport())
		if p.AuDessusDuSeuil() {
			auDessus++
			slog.WarnContext(ctx, "data_health: passes supersédées au-delà du seuil, compaction de nuit attendue",
				"titleSlug", p.TitleSlug, "table", p.Table, "brutes", p.Brutes, "gardees", p.Gardees,
				"rapport", arrondi2(p.Rapport()), "seuil", migration.SeuilPassesParMatch)
		}
	}
	tablesAuDessusGauge.Set(int64(auDessus))
	res.PassesOverThreshold = auDessus
	slog.InfoContext(ctx, "data_health: jauge des passes publiée",
		"tables", len(res.Passes), "au_dessus_du_seuil", auDessus, "rapport_max", arrondi2(rapportMax))
	if err := ecrireJaugePasses(pr.PassesGaugePath(), formaterJaugePasses(s.now().UTC(), res.Passes)); err != nil {
		slog.ErrorContext(ctx, "data_health: écriture du fichier de jauge des passes", "path", pr.PassesGaugePath(), "err", err)
	}
}

func ecrireJaugePasses(path string, contenu []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicfile.WriteFileStrict(path, contenu, 0o644)
}

// formaterJaugePasses rend le fichier de jauge : en-tête `clé=valeur` puis une ligne `table=` par
// table mesurée. Le script hôte ne lit que `mesure_utc` et `tables_au_dessus_du_seuil` ; les
// lignes `table=` servent l'exploitant qui lit le fichier.
func formaterJaugePasses(mesure time.Time, passes []PassesTitre) []byte {
	var b strings.Builder
	auDessus := 0
	for _, p := range passes {
		if p.AuDessusDuSeuil() {
			auDessus++
		}
	}
	b.WriteString("# Jauge « passes par match » (lignes brutes / lignes gardées par la compaction),\n")
	b.WriteString("# écrite par le contrôle de santé des données ; lue par scripts/compact-passes-nuit.sh.\n")
	fmt.Fprintf(&b, "mesure_utc=%s\n", mesure.Format(formatHorodatageJauge))
	fmt.Fprintf(&b, "seuil=%.2f\n", migration.SeuilPassesParMatch)
	fmt.Fprintf(&b, "tables_au_dessus_du_seuil=%d\n", auDessus)
	for _, p := range passes {
		fmt.Fprintf(&b, "table=%s %s brutes=%d gardees=%d rapport=%.2f\n",
			p.TitleSlug, p.Table, p.Brutes, p.Gardees, p.Rapport())
	}
	return []byte(b.String())
}

func arrondi2(x float64) float64 {
	return float64(int64(x*100+0.5)) / 100
}
