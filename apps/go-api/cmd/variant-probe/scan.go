// cmd/variant-probe — scan.go : mode HORS LIGNE de la sonde. Aucune requête, il
// ne lit que des fichiers déjà déposés par le mode réseau.
//
// Il applique les quatre sondes du seuil posé AVANT mesure
// (.ai/V7.5/replay2d/PLAN_SONDE_VARIANT_SOCLES.md, section 4) :
//
//	a) les 3 type_id de socle d'arme relevés dans les .mvar ;
//	b) les labels de mode connus, en clair ET par leur hash murmur3 ;
//	c) les hashs de label restés non résolus au lot socles-mvar ;
//	d) les chaînes de la famille spawner / pad / palette / weapon set / filtre.
//
// Chaque valeur est cherchée sous QUATRE encodages (4 octets LE, 4 octets BE,
// varint LEB128, varint zigzag) : les .mvar encodent leurs entiers en varint
// Bond, un scan en 4 octets fixes seul manquerait la cible et conclurait « absent »
// à tort.
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// padTypeIDs : les 3 type_id d'objet de socle, mesurés sur Catalyst, Cliffhanger
// et Smallhalla (PLAN_SOCLES_MVAR.md section 8).
var padTypeIDs = map[string]int32{
	"arme_de_pouvoir":  1597478195, // 0x5F379533
	"arme_de_rack":     1649659840, // 0x6253CFC0
	"socle_de_powerup": 1585893648, // 0x5E86D110
}

// knownLabels : NOMS des labels de mode résolus au lot socles-mvar. Le hash est
// RECALCULÉ par mapvar.LabelHash — on ne recopie pas la table hash→nom, on
// interroge la fonction de hachage de production.
var knownLabels = []string{
	"ctf_include", "ctf_exclude", "ctf_neutral_include", "ctf_multi_exclude",
	"flag_spawn", "flag_delivery",
	"stockpile_include", "stockpile_exclude", "stockpile_socket", "stockpile_navpoint",
	"oddball_spawn", "oddball_include",
	"strongholds_include", "strongholds_zone",
	"assault_include", "assault_exclude", "assault_bomb",
	"extraction_zone", "extraction_include",
	"infection_include", "infection_exclude",
	"elimination_include", "elimination_exclude",
	"skull_weapon",
	"firefight_include", "minigame_include", "forge_include",
}

// unresolvedLabelHashes : hashs de label vus dans les .mvar et jamais résolus
// (PLAN_SOCLES_MVAR.md sections 7 et 9). S'ils apparaissaient dans un .bin de
// mode, le mode nommerait les objets qu'il allume.
var unresolvedLabelHashes = []int32{-886053664, -831896525}

// familyPatterns : sonde (d). Motifs ASCII, insensibles à la casse.
var familyPatterns = []string{
	"spawner", "weapon_pad", "weaponpad", "pad_", "_pad",
	"palette", "objectfilter", "object_filter", "weaponset", "weapon_set",
	"_include", "_exclude", "powerup", "power_up", "overshield", "activecamo",
}

// runScan applique les quatre sondes à chaque fichier du dossier.
func runScan(ctx context.Context, dir string) error {
	files, err := collectFiles(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("aucun fichier sous %s", dir)
	}
	slog.InfoContext(ctx, "variant-probe/scan: début", "dir", dir, "fichiers", len(files))

	targets := buildTargets()
	for _, path := range files {
		blob, rerr := os.ReadFile(path) //nolint:gosec // chemin fourni explicitement à la sonde
		if rerr != nil {
			slog.ErrorContext(ctx, "variant-probe/scan: lecture", "path", path, "err", rerr)
			continue
		}
		scanOne(ctx, path, blob, targets)
		if strings.HasSuffix(strings.ToLower(path), ".bin") && len(blob) > 100000 {
			reportLabelHashes(ctx, path, blob)
		}
	}
	return nil
}

// namedValue associe un libellé de sonde à la valeur entière cherchée.
type namedValue struct {
	label string
	value int32
}

// buildTargets construit la liste des valeurs entières à chercher (sondes a/b/c).
func buildTargets() []namedValue {
	var out []namedValue
	for name, id := range padTypeIDs {
		out = append(out, namedValue{label: "type_id " + name, value: id})
	}
	for _, name := range knownLabels {
		out = append(out, namedValue{label: "label " + name, value: mapvar.LabelHash(name)})
	}
	for _, h := range unresolvedLabelHashes {
		out = append(out, namedValue{label: fmt.Sprintf("hash non resolu %d", h), value: h})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].label < out[j].label })
	return out
}

// scanOne applique les sondes à un fichier et journalise CE QUI EST TROUVÉ ainsi
// que le total cherché — un négatif n'a de valeur que si l'on sait ce qui a été
// cherché.
func scanOne(ctx context.Context, path string, blob []byte, targets []namedValue) {
	var hits []string
	for _, t := range targets {
		for enc, pattern := range encodings(t.value) {
			if bytes.Contains(blob, pattern) {
				hits = append(hits, fmt.Sprintf("%s [%s @%d]", t.label, enc, bytes.Index(blob, pattern)))
			}
		}
	}
	// Sonde (b) volet « en clair » et sonde (d) : motifs ASCII.
	lower := bytes.ToLower(blob)
	for _, name := range knownLabels {
		if bytes.Contains(lower, []byte(name)) {
			hits = append(hits, "label EN CLAIR "+name)
		}
	}
	var family []string
	for _, p := range familyPatterns {
		if bytes.Contains(lower, []byte(strings.ToLower(p))) {
			family = append(family, p)
		}
	}
	slog.InfoContext(ctx, "variant-probe/scan: fichier",
		"path", filepath.Base(path), "bytes", len(blob),
		"valeurs_cherchees", len(targets), "trouvailles", len(hits),
		"detail", strings.Join(hits, " | "),
		"famille_d", strings.Join(family, " | "))
}

// encodings retourne les quatre représentations binaires d'un int32.
func encodings(v int32) map[string][]byte {
	le := make([]byte, 4)
	binary.LittleEndian.PutUint32(le, uint32(v))
	be := make([]byte, 4)
	binary.BigEndian.PutUint32(be, uint32(v))
	return map[string][]byte{
		"4o_LE":         le,
		"4o_BE":         be,
		"varint":        leb128(uint64(uint32(v))),
		"varint_zigzag": leb128(uint64(uint32((v << 1) ^ (v >> 31)))),
	}
}

// leb128 encode un entier en varint non signé (encodage Bond des entiers).
func leb128(v uint64) []byte {
	var out []byte
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			out = append(out, b|0x80)
			continue
		}
		out = append(out, b)
		return out
	}
}

// collectFiles liste récursivement les fichiers d'un dossier, images exclues :
// elles sont volumineuses (41 Mo par variant) et ne portent aucune règle.
func collectFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".png", ".jpg", ".jpeg", ".webp":
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out, err
}

// --- Extraction de la LISTE de hashs de label d'un .bin de mode ---
//
// Motif mesuré dans MultiFlag.bin (offset 526034) : `0a 07 10 <varint zigzag> 00`,
// et sa variante `06 10 <varint zigzag> 00`. Le varint décodé à 526037 vaut
// exactement -2087265038, soit mapvar.LabelHash("ctf_include") : le motif est
// vérifié, pas supposé.
//
// Le filtre de taille (varint >= 4 octets) écarte les petits entiers de réglage
// (score à atteindre, durées) qui partagent la même forme d'encodage.

// hashOccurrence : un hash de label et son nombre d'apparitions.
type hashOccurrence struct {
	hash  int32
	count int
}

// extractLabelHashes liste les hashs de label d'un blob, du plus fréquent au moins
// fréquent.
func extractLabelHashes(blob []byte) []hashOccurrence {
	counts := map[int32]int{}
	for i := 2; i+1 < len(blob); i++ {
		if blob[i] != 0x10 || (blob[i-1] != 0x07 && blob[i-1] != 0x06) {
			continue
		}
		v, n, ok := readVarint(blob[i+1:])
		if !ok || n < 4 || i+1+n >= len(blob) || blob[i+1+n] != 0x00 {
			continue
		}
		counts[zigzagDecode(v)]++
	}
	out := make([]hashOccurrence, 0, len(counts))
	for h, c := range counts {
		out = append(out, hashOccurrence{hash: h, count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].hash < out[j].hash
	})
	return out
}

// readVarint lit un varint LEB128. Retourne (valeur, octets consommés, ok).
func readVarint(b []byte) (uint64, int, bool) {
	var v uint64
	for i := 0; i < len(b) && i < 10; i++ {
		v |= uint64(b[i]&0x7F) << (7 * i)
		if b[i]&0x80 == 0 {
			return v, i + 1, true
		}
	}
	return 0, 0, false
}

func zigzagDecode(v uint64) int32 {
	return int32(uint32(v>>1) ^ uint32(-int32(v&1)))
}

// reportLabelHashes journalise la liste de hashs d'un fichier, résolus et inconnus
// séparés — on ne devine pas un libellé (garde-fou objectives.go).
func reportLabelHashes(ctx context.Context, path string, blob []byte) {
	occ := extractLabelHashes(blob)
	var known, unknown []string
	for _, o := range occ {
		if name := mapvar.LabelName(o.hash); name != "" {
			known = append(known, fmt.Sprintf("%s x%d", name, o.count))
			continue
		}
		unknown = append(unknown, fmt.Sprintf("%d x%d", o.hash, o.count))
	}
	slog.InfoContext(ctx, "variant-probe/scan: hashs de label",
		"path", filepath.Base(path), "distincts", len(occ),
		"resolus", strings.Join(known, " | "),
		"inconnus_top", strings.Join(unknown[:minInt(len(unknown), 15)], " | "))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
