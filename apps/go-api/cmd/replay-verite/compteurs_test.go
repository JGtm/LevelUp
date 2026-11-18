package main

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const matchCompteurs = "aaaaaaaa-0000-0000-0000-000000000001"

// serieJSON rend une courbe publiee : un cumul et une manche portant les memes valeurs.
func serieJSON(valeurs ...int) string {
	var pas []string
	for _, v := range valeurs {
		pas = append(pas, `{"t":1,"v":`+strconv.Itoa(v)+`}`)
	}
	liste := strings.Join(pas, ",")
	return `{"total":[` + liste + `],"rounds":[{"round":0,"points":[` + liste + `]}]}`
}

// documentCompteursJSON rend un rejeu d'un joueur (xuid fictif « 111 ») : 12 eliminations, des corps a corps, des
// grenades et une serie donnes.
func documentCompteursJSON(build string, melee, grenades, serie int) string {
	return `{"matchId":"` + matchCompteurs + `","frameIntervalMs":100,"coverage":{"decoder":{"build":"` + build + `"}},
		"scoreTimeline":{"players":[{"xuid":"111","kills":` + serieJSON(12) + `,"deaths":` + serieJSON(3) + `,"assists":` + serieJSON(1) + `,
		"stats":{"meleeKills":` + serieJSON(melee) + `,"grenadeKills":` + serieJSON(grenades) + `,"maxSpree":` + serieJSON(serie) + `}}]}}`
}

const veriteCompteurs = "match_id\txuid\teliminations\tdeaths\tassists\theadshot_kills\tmelee_kills\tgrenade_kills\tpower_weapon_kills\n" +
	matchCompteurs + "\t111\t12\t3\t1\t0\t3\t0\t0\n"

const serieVerite = "match_id\txuid\tmax_killing_spree\n" + matchCompteurs + "\t111\t4\n"

func lancerCompteursTest(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var out, errw bytes.Buffer
	code := lancer(append([]string{"-compteurs"}, args...), &out, &errw)
	return out.String() + errw.String(), code
}

// TestCompteurs_AccordEtDefauts : l'accord est exact par compteur et par build, les defauts sont comptes, et aucun
// identifiant de joueur ne sort.
func TestCompteurs_AccordEtDefauts(t *testing.T) {
	dir := t.TempDir()
	ecrireArtefact(t, dir, "aaaaaaaa.json.gz", documentCompteursJSON("HI_1_13_0", 15, 16, 20))
	verite := ecrire(t, dir, "participants.tsv", veriteCompteurs)
	serie := ecrire(t, dir, "series.tsv", serieVerite)
	sortie, code := lancerCompteursTest(t, "-verite", verite, "-serie", serie, "-detail", dir)
	if code != codeOK {
		t.Fatalf("code %d : %s", code, sortie)
	}
	for _, attendu := range []string{
		"1 rejeux, 1 joueurs du tableau, 1 joints",
		"| HI_1_13_0 | 1/1 | 1/1 | 1/1 | 0/0 (+1) | 0/1 | 0/1 | 0/0 (+1) | 0/1 |",
		"compteur nomme > eliminations de la manche : 1 couples dans 1 rejeux",
		"serie (maxSpree) > eliminations du joueur : 1 joueurs dans 1 rejeux",
		"egales a 16 : 1 dans 1 rejeux",
		"aaaaaaaa:3csg",
	} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("%q absent de\n%s", attendu, sortie)
		}
	}
	if strings.Contains(sortie, "111") {
		t.Errorf("un identifiant de joueur est sorti :\n%s", sortie)
	}
}

// TestCompteurs_AvantApres : un compteur faux qui devient exact est un gain, et le tableau d'apres le montre.
func TestCompteurs_AvantApres(t *testing.T) {
	avant, apres := t.TempDir(), t.TempDir()
	ecrireArtefact(t, avant, "aaaaaaaa.json.gz", documentCompteursJSON("HI_1_13_0", 15, 16, 20))
	ecrireArtefact(t, apres, "aaaaaaaa.json.gz", documentCompteursJSON("HI_1_13_0", 3, 0, 4))
	verite := ecrire(t, avant, "participants.tsv", veriteCompteurs)
	serie := ecrire(t, avant, "series.tsv", serieVerite)
	sortie, code := lancerCompteursTest(t, "-verite", verite, "-serie", serie, "-apres-dossier", apres, filepath.Join(avant, "aaaaaaaa.json.gz"))
	if code != codeOK {
		t.Fatalf("code %d : %s", code, sortie)
	}
	for _, attendu := range []string{
		"ACCORD EXACT AVANT", "ACCORD EXACT APRES",
		"| HI_1_13_0 | 1/1 | 1/1 | 1/1 | 0/0 (+1) | 1/1 | 1/1 | 0/0 (+1) | 1/1 |",
		"| TOUS | +0 / -0 | +0 / -0 | +0 / -0 | +0 / -0 | +1 / -0 | +1 / -0 | +0 / -0 | +1 / -0 |",
	} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("%q absent de\n%s", attendu, sortie)
		}
	}
}

// TestCompteurs_Usage : sans export de verite, ou sans rejeu, le mode refuse.
func TestCompteurs_Usage(t *testing.T) {
	dir := t.TempDir()
	if _, code := lancerCompteursTest(t, dir); code != codeUsage {
		t.Errorf("sans -verite : code %d, attendu %d", code, codeUsage)
	}
	verite := ecrire(t, dir, "participants.tsv", veriteCompteurs)
	if _, code := lancerCompteursTest(t, "-verite", verite, dir); code != codeUsage {
		t.Errorf("dossier sans rejeu : code %d, attendu %d", code, codeUsage)
	}
}

// TestCompteurs_PrefixeAmbigu : deux matchs de l'export qui partagent leurs 8 premiers caracteres ne se joignent pas.
func TestCompteurs_PrefixeAmbigu(t *testing.T) {
	dir := t.TempDir()
	deux := veriteCompteurs + "aaaaaaaa-1111-0000-0000-000000000002\t222\t1\t0\t0\t0\t0\t0\t0\n"
	verite := ecrire(t, dir, "participants.tsv", deux)
	ecrireArtefact(t, dir, "aaaaaaaa.json.gz", documentCompteursJSON("HI_1_13_0", 3, 0, 4))
	if _, code := lancerCompteursTest(t, "-verite", verite, dir); code != codeUsage {
		t.Errorf("prefixe ambigu : code %d, attendu %d", code, codeUsage)
	}
}
