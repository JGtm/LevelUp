package main

// model.go — les formes lues : artefacts de rejeu (JSON) et oracle API (TSV).
//
// Aucune base n'est ouverte : l'instrument ne lit que des fichiers plats.

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// --- Artefact de rejeu -------------------------------------------------------

type artefact struct {
	SchemaVersion   int              `json:"schemaVersion"`
	MatchID         string           `json:"matchId"`
	FrameCount      int              `json:"frameCount"`
	FrameIntervalMS int              `json:"frameIntervalMs"`
	DurationMS      int              `json:"durationMs"`
	Tracks          []track          `json:"tracks"`
	Roster          []rosterEntry    `json:"roster"`
	Objectives      []objectiveRow   `json:"objectives"`
	FlagCarries     []flagCarry      `json:"flagCarries"`
	SkullCarries    []skullCarry     `json:"skullCarries"`
	BombCarries     []skullCarry     `json:"bombCarries"`
	ZoneStates      []zoneState      `json:"zoneStates"`
	Vehicles        []vehiculeArt    `json:"vehicles"`
	ScoreTimeline   scoreTimelineArt `json:"scoreTimeline"`
	Coverage        json.RawMessage  `json:"coverage"`
}

type track struct {
	Slot     int     `json:"slot"`
	Team     int     `json:"team"`
	XUID     string  `json:"xuid"`
	Points   []point `json:"points"`
	EndFrame int     `json:"endFrame"`
}

type point struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type rosterEntry struct {
	XUID string `json:"xuid"`
	Name string `json:"name"`
}

type objectiveRow struct {
	T      int    `json:"t"`
	XUID   string `json:"xuid"`
	Stat   string `json:"stat"`
	TimeMS int    `json:"timeMs"`
}

type flagCarry struct {
	Team  int        `json:"team"`
	Spans []flagSpan `json:"spans"`
}

type flagSpan struct {
	State string `json:"state"`
	T0    int    `json:"t0"`
	T1    int    `json:"t1"`
	XUID  string `json:"xuid"`
}

// skullCarry est la forme du calque crane (et de la bombe) : un intervalle par portage.
type skullCarry struct {
	XUID string `json:"xuid"`
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
}

type zoneState struct {
	ZoneRef    int         `json:"zoneRef"`
	LetterRank int         `json:"letterRank"`
	Spans      []zoneSpan  `json:"spans"`
	Gauge      []gaugePt   `json:"gauge"`
	Periods    []hillPer   `json:"periods"`
	Extra      interface{} `json:"-"`
}

type zoneSpan struct {
	T0     int  `json:"t0"`
	T1     int  `json:"t1"`
	Owner  int  `json:"owner"`
	Active bool `json:"active"`
}

type gaugePt struct {
	T int     `json:"t"`
	V float64 `json:"v"`
}

type hillPer struct {
	T0    int `json:"t0"`
	T1    int `json:"t1"`
	Owner int `json:"owner"`
}

// couverture : la partie de `coverage` que l'audit interroge.
type couverture struct {
	Objectives struct {
		Available   int `json:"available"`
		Attached    int `json:"attached"`
		NoSlot      int `json:"noSlot"`
		Ambiguous   int `json:"ambiguous"`
		OutOfWindow int `json:"outOfWindow"`
		Unpublished int `json:"unpublished"`
	} `json:"objectives"`
	FlagCarries *struct {
		FlagFilm    bool `json:"flagFilm"`
		Bursts      int  `json:"bursts"`
		Captures    int  `json:"captures"`
		Steals      int  `json:"steals"`
		Openings    int  `json:"openings"`
		Carries     int  `json:"carries"`
		Closed      int  `json:"closed"`
		Open        int  `json:"open"`
		NoBridge    int  `json:"noBridge"`
		NoTrack     int  `json:"noTrack"`
		OutOfWindow int  `json:"outOfWindow"`
		Unresolved  int  `json:"unresolved"`
		NeutralFlag bool `json:"neutralFlag"`
	} `json:"flagCarries"`
	SkullCarries *struct {
		Carries       int `json:"carries"`
		NoBridge      int `json:"noBridge"`
		CarrierAbsent int `json:"carrierAbsent"`
	} `json:"skullCarries"`
	Zones *struct {
		Method       string `json:"method"`
		Roles        string `json:"roles"`
		Catalog      int    `json:"catalog"`
		Slots        int    `json:"slots"`
		Paired       int    `json:"paired"`
		Unpaired     int    `json:"unpaired"`
		Captures     int    `json:"captures"`
		Attributed   int    `json:"attributed"`
		NoPosition   int    `json:"noPosition"`
		Outside      int    `json:"outside"`
		Spans        int    `json:"spans"`
		HillPeriods  int    `json:"hillPeriods"`
		Letters      int    `json:"letters"`
		UnknownOwner int    `json:"unknownOwner"`
	} `json:"zones"`
	Score struct {
		TeamIdentity  string `json:"teamIdentity"`
		Rounds        int    `json:"rounds"`
		ModeSupported bool   `json:"modeSupported"`
		Points        int    `json:"points"`
	} `json:"score"`
	Bridge struct {
		Slots        int `json:"slots"`
		LivesNamed   int `json:"livesNamed"`
		LivesTotal   int `json:"livesTotal"`
		UnnamedLives int `json:"unnamedLives"`
	} `json:"bridge"`
}

func lireArtefact(path string) (*artefact, *couverture, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var a artefact
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, nil, err
	}
	var c couverture
	if len(a.Coverage) > 0 {
		if err := json.Unmarshal(a.Coverage, &c); err != nil {
			return nil, nil, err
		}
	}
	return &a, &c, nil
}

// --- Oracle API (TSV `diag_q`) ----------------------------------------------

// tsv est une table lue avec son en-tete ; la derniere ligne « (N rows) » est ecartee.
type tsv struct {
	head map[string]int
	rows [][]string
}

func lireTSV(path string) (*tsv, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	t := &tsv{head: map[string]int{}}
	first := true
	for sc.Scan() {
		ligne := strings.TrimRight(sc.Text(), "\r")
		ligne = strings.TrimPrefix(ligne, "\ufeff")
		if ligne == "" {
			continue
		}
		if strings.HasPrefix(ligne, "(") && strings.HasSuffix(ligne, "rows)") {
			continue // pied de `diag_q`
		}
		champs := strings.Split(ligne, "\t")
		if first {
			for i, n := range champs {
				t.head[n] = i
			}
			first = false
			continue
		}
		t.rows = append(t.rows, champs)
	}
	return t, sc.Err()
}

func (t *tsv) s(r []string, col string) string {
	i, ok := t.head[col]
	if !ok || i >= len(r) {
		return ""
	}
	v := r[i]
	if v == "NULL" {
		return ""
	}
	return v
}

func (t *tsv) f(r []string, col string) (float64, bool) {
	v := t.s(r, col)
	if v == "" {
		return 0, false
	}
	x, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return x, true
}

func (t *tsv) i(r []string, col string) (int, bool) {
	x, ok := t.f(r, col)
	return int(x), ok
}
