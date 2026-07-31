package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type P struct {
	T          int `json:"t"`
	X, Y, Z, H float32
	Sh         *float32 `json:"sh"`
	Hp         *float32 `json:"hp"`
}
type T struct {
	Slot   uint32 `json:"slot"`
	Points []P    `json:"points"`
}
type D struct {
	SchemaVersion int `json:"schemaVersion"`
	FrameCount    int `json:"frameCount"`
	Tracks        []T `json:"tracks"`
	Shots         []struct {
		T    int    `json:"t"`
		Slot uint32 `json:"slot"`
	} `json:"shots"`
	Structure []any `json:"structure"`
	Geometry  []any `json:"geometry"`
}

func load(p string) D {
	b, e := os.ReadFile(p)
	if e != nil {
		panic(e)
	}
	var d D
	if e := json.Unmarshal(b, &d); e != nil {
		panic(e)
	}
	return d
}
func main() {
	for _, p := range os.Args[1:] {
		d := load(p)
		n, sh, hp := 0, 0, 0
		for _, t := range d.Tracks {
			for _, q := range t.Points {
				n++
				if q.Sh != nil {
					sh++
				}
				if q.Hp != nil {
					hp++
				}
			}
		}
		fmt.Printf("%s : schema=%d tracks=%d points=%d shots=%d struct=%d geom=%d frames=%d | sh=%d hp=%d\n",
			p[len(p)-24:], d.SchemaVersion, len(d.Tracks), n, len(d.Shots), len(d.Structure), len(d.Geometry), d.FrameCount, sh, hp)
	}
}
