package main

// names.go — LE NOM INTERNE DE CHAQUE ICÔNE, craqué et non deviné.
//
// LE PROBLÈME. Le bloc `UI display info` porte un champ `name`, mais ce n'est pas une chaîne :
// c'est un StringID, un murmur3. Les tags de release embarquent bien une table de chaînes —
// sauf qu'elle ne contient QUE des couples (index, hash) : les textes sont strippés. Le nom
// n'est donc pas lisible, il est haché.
//
// LA RECETTE, DÉJÀ VALIDÉE AILLEURS DANS LE DÉPÔT. `ETAT_DE_L_ART_FORGE_PALETTE_ZONES`
// §Q1.0-septies a craqué les noms d'objets Forge de la même façon : moissonner les chaînes
// imprimables du binaire du jeu, les hacher, et voir lesquelles retombent sur les StringID
// cherchés. La fonction de hachage n'est pas redevinée ici : c'est `mapvar.LabelHash`
// (murmur3 x86_32, seed 0), établie par correspondance directe et couverte par des tests.
//
// CE QUE ÇA REND, ET CE QUE ÇA NE REND PAS. Un nom INTERNE (`assault_rifle`, `skull`, `flag`,
// `mutilator`), pas un libellé produit — ce sont deux choses différentes et le libellé FR/EN
// reste celui de `weapon_names.toml`. Le nom interne sert à IDENTIFIER une icône, ce qui est
// précisément la question ouverte pour tout ce qui n'est pas au registre.
//
// LES CONFLITS SONT PUBLIÉS, PAS ARBITRÉS. Plusieurs `weap` peuvent revendiquer le même index
// avec des noms différents (des variantes de campagne portent des index périmés). Quand c'est
// le cas, tous les noms sont rendus : trancher au hasard donnerait une étiquette fausse avec
// l'apparence d'une certitude.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// gameBinary rend le binaire du jeu, c'est-à-dire LE PLUS GROS des candidats : le fichier à
// la racine de l'installation est un lanceur de 3,9 Mo, le vrai binaire vit sous `game/` et
// pèse 80 Mo. Prendre le premier trouvé donne 0 nom craqué — mesuré.
func gameBinary() string {
	root := moduleRoot()
	if root == "" {
		return ""
	}
	base := filepath.Dir(root)
	var best string
	var bestSize int64
	for _, c := range []string{
		filepath.Join(base, "game", "HaloInfinite.exe"),
		filepath.Join(base, "HaloInfinite.exe"),
	} {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Size() > bestSize {
			best, bestSize = c, fi.Size()
		}
	}
	return best
}

// harvestStrings moissonne les suites ASCII imprimables d'au moins minLen caractères.
func harvestStrings(path string, minLen int) (map[string]bool, error) {
	f, err := os.Open(path) //nolint:gosec // chemin dérivé de l'installation du jeu
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := make(map[string]bool, 1<<19)
	r := bufio.NewReaderSize(f, 1<<20)
	cur := make([]byte, 0, 256)
	flush := func() {
		if len(cur) >= minLen {
			out[string(cur)] = true
		}
		cur = cur[:0]
	}
	buf := make([]byte, 1<<16)
	for {
		n, rerr := r.Read(buf)
		for i := 0; i < n; i++ {
			if b := buf[i]; b >= 0x20 && b < 0x7f {
				cur = append(cur, b)
				continue
			}
			flush()
		}
		if rerr != nil {
			break
		}
	}
	flush()
	return out, nil
}

// nameVariants : formes dérivées d'une chaîne moissonnée. Le jeu hache souvent un fragment
// (dernier segment d'un chemin, extension retirée, minuscules) plutôt que la chaîne entière.
func nameVariants(s string) []string {
	v := []string{s, strings.ToLower(s)}
	if i := strings.LastIndexAny(s, "/\\"); i >= 0 && i+1 < len(s) {
		v = append(v, s[i+1:], strings.ToLower(s[i+1:]))
	}
	if i := strings.LastIndex(s, "."); i > 0 {
		v = append(v, s[:i], strings.ToLower(s[:i]))
	}
	return v
}

// resolveIconNames rend, par index d'atlas, les noms internes craqués — triés et dédoublonnés.
//
// `canon` est l'ensemble des `weap` que le registre utilise. Il sert de FILTRE PRIORITAIRE et
// ce n'est pas cosmétique : des tags de campagne portent des index d'atlas PÉRIMÉS, et leur
// nom — juste pour eux — atterrissait sur l'icône d'une autre arme. Mesuré : l'index 7 se
// faisait appeler « shotgun » par un tag legacy alors que le registre y lit la Hydra, et
// l'index 5 recevait « needler » en plus de « sniper_rifle ». Quand un tag canonique nomme un
// index, lui seul compte ; sinon on prend ce qu'on a, conflits publiés.
//
// Retourne une map vide (et pas une erreur) si le binaire du jeu est introuvable : le nom est
// un bonus, son absence ne doit pas faire échouer l'extraction des images.
func resolveIconNames(ix *tagIndex, canon map[uint32]bool, owned map[int]bool) (map[int][]string, error) {
	fields, err := loadWeapPlugin()
	if err != nil {
		return nil, err
	}
	offSprite, offIndex, offName, ok := uiBlockOffsets(fields)
	if !ok || offName < 0 {
		return nil, fmt.Errorf("champs `name` / `sprite index` absents du plugin")
	}
	exe := gameBinary()
	if exe == "" {
		return map[int][]string{}, nil
	}
	strs, err := harvestStrings(exe, 3)
	if err != nil {
		return map[int][]string{}, nil //nolint:nilerr // le nom est un bonus, pas un requis
	}
	dict := make(map[uint32]string, len(strs)*2)
	// Le vocabulaire cure comble ce que la moisson ne rend pas : le binaire ne contient pas
	// « fusion_coil » ni « sandwich » comme jeton isole.
	for _, w := range curatedVocabulary {
		if h := uint32(mapvar.LabelHash(w)); dict[h] == "" {
			dict[h] = w
		}
	}
	for s := range strs {
		for _, v := range nameVariants(s) {
			h := uint32(mapvar.LabelHash(v))
			if _, dup := dict[h]; !dup {
				dict[h] = v
			}
		}
	}

	byIndex := map[int]map[string]bool{}
	canonNamed := map[int]bool{} // index nommé par un tag canonique
	seen := map[uint32]bool{}
	for _, r := range ix.byGroup["weap"] {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		data, err := ix.extract(r)
		if err != nil {
			continue
		}
		wt, err := openWeapTag(data)
		if err != nil {
			continue
		}
		abs, _, _, found := findUIBlock(wt, data, offSprite)
		if !found || abs+offName+4 > len(data) {
			continue
		}
		sid := readU32LE(data, abs+offName)
		name, hit := dict[sid]
		if hit && !plausibleIdent(name) {
			hit = false // collision : un nom du jeu s ecrit [a-z0-9_]+
		}
		if !hit {
			continue
		}
		idx := int(readU32LE(data, abs+offIndex))
		isCanon := canon[r.ID]
		if !isCanon {
			// PROVENANCE, et elle change tout. Un nom issu d'un `weap` NON canonique n'est
			// fiable que si son index l'est aussi — or les tags de campagne portent des index
			// PÉRIMÉS. Cas avéré : l'index 31 se faisait appeler `shade_turret` alors que
			// l'image est une caisse. Le nom est conservé mais MARQUÉ, jamais servi comme un
			// fait ; c'est la page de nommage qui le donne à vérifier.
			name = "?" + name
		}
		// Un index que le registre REVENDIQUE n accepte que le nom d un tag canonique, meme
		// si aucun canonique n a craque : sinon un tag legacy baptisait l index 7 « shotgun »
		// la ou le registre y lit la Hydra.
		if owned[idx] && !isCanon {
			continue
		}
		if canonNamed[idx] && !isCanon {
			continue // un tag canonique a déjà nommé cet index : les autres se taisent
		}
		if isCanon && !canonNamed[idx] {
			canonNamed[idx] = true
			delete(byIndex, idx) // on écarte ce que les tags non canoniques avaient déposé
		}
		if byIndex[idx] == nil {
			byIndex[idx] = map[string]bool{}
		}
		byIndex[idx][name] = true
	}

	out := make(map[int][]string, len(byIndex))
	for idx, set := range byIndex {
		names := make([]string, 0, len(set))
		for n := range set {
			names = append(names, n)
		}
		sort.Strings(names)
		out[idx] = names
	}
	return out, nil
}

func readU32LE(b []byte, o int) uint32 {
	if o < 0 || o+4 > len(b) {
		return 0
	}
	return uint32(b[o]) | uint32(b[o+1])<<8 | uint32(b[o+2])<<16 | uint32(b[o+3])<<24
}
