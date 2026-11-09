package replaybuild

// app_version.go — la comparaison de versions de l'application (major.minor.patch), pour ne pas regenerer un
// artefact produit par une version plus recente que le binaire qui le regarde (retour a une version anterieure).

import (
	"strconv"
	"strings"
)

// version : major.minor.patch et son eventuelle pre-version (`8.1.0-rc1`).
type version struct {
	nums [3]int
	pre  string
}

// parseVersion lit « 8.1.0 », « v8.1.0 », « 8.1 », « 8.1.0-rc1 », « 8.1.0+build ». Faux pour tout le reste
// (« dev », vide) : une version qu'on ne sait pas lire ne se compare pas.
func parseVersion(s string) (version, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var v version
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], s[i+1:]
	}
	if s == "" {
		return version{}, false
	}
	parts := strings.Split(s, ".")
	if len(parts) > len(v.nums) {
		return version{}, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return version{}, false
		}
		v.nums[i] = n
	}
	return v, true
}

// compare rend <0, 0, >0. Une pre-version est plus ancienne que la version finale de meme numero.
func (v version) compare(o version) int {
	for i := range v.nums {
		if v.nums[i] != o.nums[i] {
			return v.nums[i] - o.nums[i]
		}
	}
	switch {
	case v.pre == o.pre:
		return 0
	case v.pre == "":
		return 1
	case o.pre == "":
		return -1
	}
	return strings.Compare(v.pre, o.pre)
}

// versionPlusRecente : a est lisible, b est lisible, et a est strictement plus recente que b.
func versionPlusRecente(a, b string) bool {
	va, okA := parseVersion(a)
	vb, okB := parseVersion(b)
	return okA && okB && va.compare(vb) > 0
}
