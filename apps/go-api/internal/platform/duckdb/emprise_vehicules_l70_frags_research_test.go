//go:build research

package duckdb

// emprise_vehicules_l70_frags_research_test.go — LOT L7.0 DU PLAN
// `.ai/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, VOLET BASE : les frags de CLASSE VEHICULE
// (decision D5), lus comme la « Repartition des frags » les lit.
//
// # UNE SEULE DEFINITION DE LA CLASSE
//
// Meme requete que `buildKillSourceWeaponQuery` (vue `match_kill_events_latest`, source
// mesuree, tueur du kill-feed resolu), meme traduction tag -> cle de registre
// (`halo_infinite.NewKillSourceRegistry`, le classificateur du lecteur), meme cle -> classe
// (`resolveWeaponKeyDimensions`, le resolveur du lecteur sur `metadata.weapons`). Une source
// sans cle de registre (les ecrasements `CollisionDamage` en tete) n'a pas de classe et sort du
// compte, comme dans la Repartition des frags. La seule difference : le lecteur agrege par
// (tueur, cle) ; ici chaque frag garde son INSTANT, parce que le seuil du plan le compare aux
// episodes d'occupation.
//
// Le camp du tueur vient de `match_participants.team_id`. Rien n'est ecrit en base : les deux
// bases sont des COPIES ouvertes en lecture seule ; la sortie JSON va dans le repertoire du lot.
//
// SANS SES VARIABLES, IL SE SAUTE (les bases ne sont pas versionnees) :
//
//	EMPRISE_L70_FILMS=<match_id,...> EMPRISE_L70_DIR=<scratch> EMPRISE_L70_SHARED=<copie shared> \
//	EMPRISE_L70_META=<copie metadata> \
//	  go test ./internal/platform/duckdb/ -run '^TestEmpriseL70Frags$' -v -count=1

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite"
)

// l70Frag : un frag de classe vehicule ou tourelle, avec son instant et les camps.
type l70Frag struct {
	TimeMS     int    `json:"timeMs"`
	KillerXUID string `json:"killerXuid"`
	KillerTeam *int   `json:"killerTeam"`
	Key        string `json:"key"`
	Class      string `json:"class"`
}

// l70Match : tous les frags MESURES d'un match, par classe, et ceux de classe d'engin.
type l70Match struct {
	MatchID        string         `json:"matchId"`
	Engins         []l70Frag      `json:"engins"`
	ParClasse      map[string]int `json:"parClasse"`
	SansCle        int            `json:"sansCle"`
	SansCleClasses map[string]int `json:"sansCleClasses"`
	SansClasse     int            `json:"sansClasse"`
	SansTeamTueur  int            `json:"sansTeamTueur"`
}

const l70RequeteFrags = `
SELECT k.time_ms, k.feed_killer_xuid, k.source_tag, p.team_id
FROM match_kill_events_latest k
LEFT JOIN match_participants p ON p.match_id = k.match_id AND p.xuid = k.feed_killer_xuid
WHERE k.match_id = ? AND k.source_tag IS NOT NULL AND k.feed_killer_xuid IS NOT NULL
ORDER BY k.time_ms`

func TestEmpriseL70Frags(t *testing.T) {
	dir, shared, meta := os.Getenv("EMPRISE_L70_DIR"), os.Getenv("EMPRISE_L70_SHARED"),
		os.Getenv("EMPRISE_L70_META")
	var films []string
	for _, f := range strings.Split(os.Getenv("EMPRISE_L70_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			films = append(films, f)
		}
	}
	if len(films) == 0 || dir == "" || shared == "" || meta == "" {
		t.Skip("EMPRISE_L70_FILMS, _DIR, _SHARED et _META requis : instrument saute")
	}
	ctx := context.Background()
	db, err := sql.Open("duckdb", shared+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("copie shared : %v", err)
	}
	defer func() { _ = db.Close() }()
	metaDB, err := OpenReadOnly(meta)
	if err != nil {
		t.Fatalf("copie metadata : %v", err)
	}
	classifier := halo_infinite.NewKillSourceRegistry()
	for _, id := range films {
		m := l70Lire(t, ctx, db, metaDB, classifier, id)
		blob, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("%s : serialisation : %v", id, err)
		}
		sortie := filepath.Join(dir, "frags")
		if err := os.MkdirAll(sortie, 0o750); err != nil {
			t.Fatalf("repertoire : %v", err)
		}
		court := title.FilmShortMatchID(id)
		if err := os.WriteFile(filepath.Join(sortie, court+".json"), blob, 0o600); err != nil {
			t.Fatalf("%s : ecriture : %v", court, err)
		}
		t.Logf("%s  frags d'engin=%d ; par classe %v ; sans cle %d %v, sans classe %d, tueur sans camp %d",
			court, len(m.Engins), m.ParClasse, m.SansCle, m.SansCleClasses, m.SansClasse, m.SansTeamTueur)
	}
}

// l70Lire lit les frags d'un match et les range par classe.
func l70Lire(t *testing.T, ctx context.Context, db *sql.DB, metaDB *DB,
	c halo_infinite.KillSourceRegistry, id string) l70Match {
	t.Helper()
	rows, err := db.QueryContext(ctx, l70RequeteFrags, id)
	if err != nil {
		t.Fatalf("%s : requete : %v", id, err)
	}
	defer func() { _ = rows.Close() }()
	type brut struct {
		timeMS int
		xuid   string
		tag    uint32
		team   *int
	}
	var lus []brut
	keys := map[string]bool{}
	for rows.Next() {
		var b brut
		var team sql.NullInt64
		if err := rows.Scan(&b.timeMS, &b.xuid, &b.tag, &team); err != nil {
			t.Fatalf("%s : lecture : %v", id, err)
		}
		if team.Valid {
			v := int(team.Int64)
			b.team = &v
		}
		lus = append(lus, b)
		if k, ok := c.KillSourceRegistryKey(b.tag); ok {
			keys[k] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("%s : lignes : %v", id, err)
	}
	cles := make([]string, 0, len(keys))
	for k := range keys {
		cles = append(cles, k)
	}
	meta := resolveWeaponKeyDimensions(ctx, metaDB, title.DefaultSlug, cles)
	m := l70Match{MatchID: id, ParClasse: map[string]int{}, SansCleClasses: map[string]int{}}
	for _, b := range lus {
		key, ok := c.KillSourceRegistryKey(b.tag)
		if !ok {
			m.SansCle++
			nom, _ := c.KillSourceClassName(b.tag)
			m.SansCleClasses[nom]++
			continue
		}
		r, ok := meta[key]
		if !ok {
			m.SansClasse++
			continue
		}
		m.ParClasse[r.class]++
		if r.class != "vehicle" && r.class != "turret" {
			continue
		}
		if b.team == nil {
			m.SansTeamTueur++
		}
		m.Engins = append(m.Engins, l70Frag{TimeMS: b.timeMS, KillerXUID: b.xuid, KillerTeam: b.team,
			Key: key, Class: r.class})
	}
	return m
}
