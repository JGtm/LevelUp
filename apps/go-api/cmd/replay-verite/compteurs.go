package main

// compteurs.go — LE MODE `-compteurs` : LES COMPTEURS NOMMES DU TABLEAU DES SCORES CONTRE L'API.
//
// Il lit des rejeux DEJA CUITS (`scoreTimeline.players[]` : eliminations, morts, assistances, et
// `stats` : tirs a la tete, corps a corps, grenades, armes lourdes, serie maximale) et l'export de
// l'API (`match_participants`, une ligne par match et par joueur), et rend l'accord exact par
// compteur et par build du jeu, plus les defauts que l'API ne dit pas (un compteur nomme au-dessus des
// eliminations de la manche, une serie au-dessus des eliminations, la valeur 16 des grenades). Il ne
// decode aucun film, n'ouvre aucune base, et n'ecrit aucun identifiant : une sortie nomme des
// rejeux par les 8 premiers caracteres de leur match, jamais un joueur.
//
// L'API ne SERT QU'A MESURER : rien ici ne decide une valeur publiee.
//
// Avec `-apres-dossier <dossier>`, chaque rejeu cite en argument est lu AVANT (l'argument) et APRES (le
// meme nom dans le dossier), et la sortie compte, par compteur, les joueurs dont l'accord est gagne
// ou perdu.

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// seuilGrenadeSuspecte : au moins cette valeur finale de grenades, le compteur est compte a part (cf.
// `rendreDefauts`).
const seuilGrenadeSuspecte = 12

// valeurGrenadeSuspecte est la valeur que l'API ne dit jamais et que les films disent.
const valeurGrenadeSuspecte = 16

// nomsCompteurs : les compteurs mesures, dans l'ordre des colonnes.
var nomsCompteurs = []string{"eliminations", "morts", "assistances", "tirs a la tete", "corps a corps",
	"grenades", "armes lourdes", "serie"}

// Indices de [nomsCompteurs].
const (
	cEliminations = iota
	cMorts
	cAssistances
	cTete
	cCorpsACorps
	cGrenades
	cArmesLourdes
	cSerie
	nbCompteurs
)

// ligneAPI est la verite de l'API pour un joueur d'un match ; nil = colonne absente.
type ligneAPI [nbCompteurs]*int

// verite : (8 premiers caracteres du match + "|" + xuid) -> ligne de l'API.
type verite map[string]*ligneAPI

// serieDoc est une courbe publiee : le cumul et les manches.
type serieDoc struct {
	Total []struct {
		V int `json:"v"`
	} `json:"total"`
	Rounds []struct {
		Round  int `json:"round"`
		Points []struct {
			V int `json:"v"`
		} `json:"points"`
	} `json:"rounds"`
}

// finale rend la valeur finale d'une courbe : la derniere du cumul, sinon la somme des fins de manche.
func (s serieDoc) finale() (int, bool) {
	if n := len(s.Total); n > 0 {
		return s.Total[n-1].V, true
	}
	somme, lu := 0, false
	for _, r := range s.Rounds {
		if n := len(r.Points); n > 0 {
			somme += r.Points[n-1].V
			lu = true
		}
	}
	return somme, lu
}

// maximum rend le plus grand point de toutes les manches (la serie n'a pas de cumul).
func (s serieDoc) maximum() (int, bool) {
	m, lu := 0, false
	for _, r := range s.Rounds {
		for _, p := range r.Points {
			if !lu || p.V > m {
				m, lu = p.V, true
			}
		}
	}
	return m, lu
}

// finsDeManche rend la derniere valeur de chaque manche qui porte un point.
func (s serieDoc) finsDeManche() map[int]int {
	out := map[int]int{}
	for _, r := range s.Rounds {
		if n := len(r.Points); n > 0 {
			out[r.Round] = r.Points[n-1].V
		}
	}
	return out
}

// joueurDoc est un joueur du tableau des scores du document.
type joueurDoc struct {
	XUID    string              `json:"xuid"`
	Kills   serieDoc            `json:"kills"`
	Deaths  serieDoc            `json:"deaths"`
	Assists serieDoc            `json:"assists"`
	Stats   map[string]serieDoc `json:"stats"`
}

// documentCompteurs est la forme lue d'un rejeu par ce mode : le build, les joueurs du tableau.
type documentCompteurs struct {
	MatchID  string `json:"matchId"`
	Coverage struct {
		Decoder struct {
			Build string `json:"build"`
		} `json:"decoder"`
	} `json:"coverage"`
	ScoreTimeline *struct {
		Players []joueurDoc `json:"players"`
	} `json:"scoreTimeline"`
}

// valeurs rend la valeur lue de chaque compteur d'un joueur du tableau (nil = le film ne la porte pas).
func (j joueurDoc) valeurs() [nbCompteurs]*int {
	var v [nbCompteurs]*int
	pose := func(i int, x int, lu bool) {
		if lu {
			v[i] = &x
		}
	}
	x, lu := j.Kills.finale()
	pose(cEliminations, x, lu)
	x, lu = j.Deaths.finale()
	pose(cMorts, x, lu)
	x, lu = j.Assists.finale()
	pose(cAssistances, x, lu)
	for i, nom := range map[int]string{cTete: "headshots", cCorpsACorps: "meleeKills", cGrenades: "grenadeKills",
		cArmesLourdes: "powerWeaponKills"} {
		x, lu = j.Stats[nom].finale()
		pose(i, x, lu)
	}
	x, lu = j.Stats["maxSpree"].maximum()
	pose(cSerie, x, lu)
	return v
}

// defautsDe compte les defauts de ce joueur : les couples (manche) ou un compteur nomme depasse les
// eliminations, et si sa serie depasse ses eliminations.
func (j joueurDoc) defautsDe() (couples, violations int, serieAberrante bool) {
	nommes := []string{"headshots", "meleeKills", "grenadeKills", "powerWeaponKills"}
	fins := map[string]map[int]int{}
	for _, nom := range nommes {
		fins[nom] = j.Stats[nom].finsDeManche()
	}
	for manche, k := range j.Kills.finsDeManche() {
		couples++
		for _, nom := range nommes {
			if v, ok := fins[nom][manche]; ok && v > k {
				violations++
				break
			}
		}
	}
	sp, luSp := j.Stats["maxSpree"].maximum()
	k, luK := j.Kills.finale()
	return couples, violations, luSp && luK && sp > k
}

// courte rend les 8 premiers caracteres d'un identifiant de match.
func courte(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// lireTSV rend les lignes d'un TSV d'export par nom de colonne.
func lireTSV(chemin string, ligne func(col func(string) string) error) error {
	f, err := os.Open(chemin) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	if !sc.Scan() {
		return fmt.Errorf("%s : fichier vide", chemin)
	}
	index := map[string]int{}
	for i, h := range strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t") {
		index[h] = i
	}
	for sc.Scan() {
		cells := strings.Split(strings.TrimRight(sc.Text(), "\r"), "\t")
		err := ligne(func(nom string) string {
			if i, ok := index[nom]; ok && i < len(cells) {
				return cells[i]
			}
			return ""
		})
		if err != nil {
			return err
		}
	}
	return sc.Err()
}

// entier lit une cellule d'export ; vide ou illisible = absent.
func entier(s string) *int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

// chargerVerite lit l'export des participants et, en option, celui des series maximales.
// Deux matchs qui partageraient leurs 8 premiers caracteres rendent une erreur : la jointure s'y fie.
func chargerVerite(participants, series string) (verite, error) {
	v := verite{}
	longs := map[string]string{}
	err := lireTSV(participants, func(col func(string) string) error {
		m := col("match_id")
		if prec, ok := longs[courte(m)]; ok && prec != m {
			return fmt.Errorf("deux matchs partagent le prefixe %s", courte(m))
		}
		longs[courte(m)] = m
		v[courte(m)+"|"+col("xuid")] = &ligneAPI{cEliminations: entier(col("eliminations")), cMorts: entier(col("deaths")),
			cAssistances: entier(col("assists")), cTete: entier(col("headshot_kills")),
			cCorpsACorps: entier(col("melee_kills")), cGrenades: entier(col("grenade_kills")),
			cArmesLourdes: entier(col("power_weapon_kills"))}
		return nil
	})
	if err != nil || series == "" {
		return v, err
	}
	return v, lireTSV(series, func(col func(string) string) error {
		if l := v[courte(col("match_id"))+"|"+col("xuid")]; l != nil {
			l[cSerie] = entier(col("max_killing_spree"))
		}
		return nil
	})
}

// issue est le verdict d'un compteur d'un joueur contre l'API.
type issue int

const (
	issueSansAPI    issue = iota // l'API ne dit rien : non compte
	issueExact                   // lu et egal
	issueFaux                    // lu et different
	issueAbsentZero              // non lu, l'API dit 0 : accord large
	issueAbsentPlus              // non lu, l'API dit plus que 0
)

// juge rend l'issue d'une valeur du film contre celle de l'API.
func juge(film, api *int) issue {
	switch {
	case api == nil:
		return issueSansAPI
	case film == nil && *api == 0:
		return issueAbsentZero
	case film == nil:
		return issueAbsentPlus
	case *film == *api:
		return issueExact
	}
	return issueFaux
}

// large dit que l'issue compte dans l'accord « le film absent vaut 0 ».
func (i issue) large() bool { return i == issueExact || i == issueAbsentZero }

// casse compte les issues d'un (build, compteur).
type casse [issueAbsentPlus + 1]int

// mesure est le resultat d'une lecture d'un lot de rejeux.
type mesure struct {
	rejeux, joueurs, joints int
	parBuild                map[string]*[nbCompteurs]casse
	couples, violations     int
	rejeuxViolations        map[string]bool
	series                  int
	rejeuxSeries            map[string]bool
	grenadesHautes          int
	grenades16              int
	rejeuxGrenades16        map[string]bool
	defauts                 map[string]int
	natures                 map[string]string
}

func nouvelleMesure() *mesure {
	return &mesure{parBuild: map[string]*[nbCompteurs]casse{}, rejeuxViolations: map[string]bool{},
		rejeuxSeries: map[string]bool{}, rejeuxGrenades16: map[string]bool{}, defauts: map[string]int{}, natures: map[string]string{}}
}

// transitions compte, par (build, compteur), les joueurs dont l'accord large est gagne ou perdu entre
// deux lectures du meme rejeu, et ceux dont l'accord exact change de valeur lue.
type transitions struct {
	gagnes, perdus map[string]*[nbCompteurs]int
}

// buildDe rend le build d'un document, « (inconnu) » quand il n'en porte pas.
func buildDe(d *documentCompteurs) string {
	if b := d.Coverage.Decoder.Build; b != "" {
		return b
	}
	return "(inconnu)"
}

// ajouter lit un document dans la mesure et rend les issues de chaque joueur joint (xuid -> issues), pour
// la comparaison avant / apres.
func (m *mesure) ajouter(d *documentCompteurs, v verite) map[string][nbCompteurs]issue {
	m.rejeux++
	id := courte(d.MatchID)
	build := buildDe(d)
	issues := map[string][nbCompteurs]issue{}
	if d.ScoreTimeline == nil {
		return issues
	}
	for _, j := range d.ScoreTimeline.Players {
		m.joueurs++
		m.defauter(id, j)
		api := v[id+"|"+j.XUID]
		if api == nil {
			continue
		}
		m.joints++
		if m.parBuild[build] == nil {
			m.parBuild[build] = &[nbCompteurs]casse{}
		}
		lu := j.valeurs()
		var is [nbCompteurs]issue
		for c := range nbCompteurs {
			is[c] = juge(lu[c], api[c])
			m.parBuild[build][c][is[c]]++
		}
		issues[j.XUID] = is
	}
	return issues
}

// defauter compte les defauts d'un joueur (couples, serie, grenades a 16) dans la mesure.
func (m *mesure) defauter(id string, j joueurDoc) {
	couples, viol, serie := j.defautsDe()
	m.couples += couples
	m.violations += viol
	if viol > 0 {
		m.rejeuxViolations[id] = true
		m.defauts[id] += viol
		m.natures[id] += "c"
	}
	if serie {
		m.series++
		m.rejeuxSeries[id] = true
		m.defauts[id]++
		m.natures[id] += "s"
	}
	if g, ok := j.Stats["grenadeKills"].finale(); ok {
		if g >= seuilGrenadeSuspecte {
			m.grenadesHautes++
		}
		if g == valeurGrenadeSuspecte {
			m.grenades16++
			m.rejeuxGrenades16[id] = true
			m.defauts[id]++
			m.natures[id] += "g"
		}
	}
}
