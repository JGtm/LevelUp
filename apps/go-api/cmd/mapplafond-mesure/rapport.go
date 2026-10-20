package main

// rapport.go — LE TABLEAU, tel qu'il se colle dans un document de chantier.
//
// Un rapport de mesure doit pouvoir etre RELU sans le code : chaque colonne y est definie, et
// tout ce que l'instrument n'a pas pu mesurer y est nomme avec son motif. Une carte absente
// sans raison ecrite serait indiscernable d'un trou de l'instrument.

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

// ecritRapport ecrit le Markdown du tableau de mesure.
func ecritRapport(o options, lignes []ligne, freq *frequentation, ecartees []string, dirRejeux string) error {
	var sb strings.Builder
	enTete(&sb, o, freq, dirRejeux)
	tableauPrincipal(&sb, lignes)
	tableauVariante(&sb, lignes)
	tableauMatiere(&sb, lignes)
	annexes(&sb, lignes, freq, ecartees)
	return os.WriteFile(o.rapport, []byte(sb.String()), 0o644) //nolint:gosec // rapport de chantier
}

func enTete(sb *strings.Builder, o options, freq *frequentation, dirRejeux string) {
	fmt.Fprintf(sb, "# Mesure des plafonds — %s\n\n", time.Now().UTC().Format("2006-01-02"))
	fmt.Fprintf(sb, "Instrument `cmd/mapplafond-mesure` · marge %.1f m · corpus `%s` "+
		"(%d documents lus, %d rattaches a une carte)\n\n", o.marge, dirRejeux, freq.lus, freq.rattaches)
	fmt.Fprintln(sb, "Definitions des colonnes :")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "- `sol joue` : altitude du sol de jeu deduite des ancres d'objectifs, publiee par le")
	fmt.Fprintln(sb, "  fond de carte (`playLevelZ`). `plafond actuel` = `sol joue` + 28 m, la tranche de jeu")
	fmt.Fprintln(sb, "  universelle appliquee aujourd'hui par la cuisson (`himap.TrancheDeJeu`).")
	fmt.Fprintln(sb, "- `h med / p99 / max` : altitudes des positions de joueur du corpus, en metres monde")
	fmt.Fprintln(sb, "  (champ `tracks[].points[].z` des artefacts cuits).")
	fmt.Fprintln(sb, "- `seuil` : le plafond PROPOSE, `h max` + marge.")
	fmt.Fprintln(sb, "- `image changee` : part des pixels porteurs de matiere dont la surface AFFICHEE est")
	fmt.Fprintln(sb, "  au-dessus du seuil — ce que la coupe changerait a l'ecran.")
	fmt.Fprintln(sb, "- `volumes coupes` : instances de geometrie entierement au-dessus du seuil (supprimees)")
	fmt.Fprintln(sb, "  et a cheval sur lui (decapitees), sur le total des instances rendues.")
	fmt.Fprintln(sb, "- `zones nommees au-dessus` : zones NOMMEES du jeu (tag `levl`) dont le plancher est")
	fmt.Fprintln(sb, "  au-dessus du seuil. C'est le detecteur de FAUX POSITIFS independant du corpus : une")
	fmt.Fprintln(sb, "  zone nommee est un espace de jeu dessine par le designer, pas un toit.")
	fmt.Fprintln(sb, "- `perte si -1 film` : de combien `h max` descendrait si UN film manquait au corpus.")
	fmt.Fprintln(sb, "  Une perte de plusieurs metres dit que le corpus borne le CORPUS, pas la carte. Un")
	fmt.Fprintln(sb, "  tiret sous DEUX films : le controle n'est pas stable, il est impossible.")
	fmt.Fprintln(sb, "")
}

func tableauPrincipal(sb *strings.Builder, lignes []ligne) {
	fmt.Fprintln(sb, "## Coupe au maximum frequente + marge")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "| carte | films | positions | sol joue | h med | h p99 | h max | plafond actuel | seuil |"+
		" image changee | volumes coupes | zones nommees au-dessus | perte si -1 film |")
	fmt.Fprintln(sb, "|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|---:|")
	for i := range lignes {
		ecritLigne(sb, &lignes[i])
	}
	fmt.Fprintln(sb, "")
}

func ecritLigne(sb *strings.Builder, l *ligne) {
	if l.erreur != "" {
		fmt.Fprintf(sb, "| `%s` | | | | | | | | | | | | ECHEC : %s |\n", l.c.module, l.erreur)
		return
	}
	n, hMed, hP99, hMax := 0, math.NaN(), math.NaN(), math.NaN()
	if l.freq != nil {
		n, hMed, hP99, hMax = l.freq.taille(), l.freq.centile(0.5), l.freq.centile(0.99), l.freq.maximum()
	}
	// Une carte non cuite (ou sans seuil, faute de corpus) n'a PAS de coupe : ses colonnes de
	// coupe portent un tiret. Un « 0,00 % » y serait une mesure, pas une absence.
	image, vol := "—", "—"
	if l.geom != nil && !math.IsNaN(l.seuil) {
		image, vol = part(l.coupe.partMatiere), volumes(l.coupe)
	}
	fmt.Fprintf(sb, "| `%s` | %d | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
		l.c.module, len(l.films), n,
		metres(niveauDe(l)), metres(hMed), metres(hP99), metres(hMax),
		metres(plafondDe(l)), metres(l.seuil),
		image, vol, zonesOuTiret(l), metres(l.perteMax))
}

func tableauVariante(sb *strings.Builder, lignes []ligne) {
	fmt.Fprintln(sb, "## Variante : coupe au 99e centile + marge")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "Ce n'est PAS une proposition : c'est le chiffrage de ce qu'on gagnerait a ignorer le")
	fmt.Fprintln(sb, "dernier pour cent des positions (grappin, canon, chute) et de ce qu'on y perdrait.")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "| carte | seuil p99 | image changee | volumes coupes | zones nommees au-dessus | positions au-dessus |")
	fmt.Fprintln(sb, "|---|---:|---:|---|---|---:|")
	for i := range lignes {
		l := &lignes[i]
		if l.erreur != "" || l.freq == nil {
			continue
		}
		fmt.Fprintf(sb, "| `%s` | %s | %s | %s | %s | %d |\n", l.c.module, metres(l.seuilP99),
			part(l.coupeP99.partMatiere), volumes(l.coupeP99), zones(l.zonesP99),
			l.freq.nombreAuDessus(l.seuilP99))
	}
	fmt.Fprintln(sb, "")
}

// tableauMatiere dit A QUELLE HAUTEUR AU-DESSUS DU SOL JOUE vit la matiere dessinee.
//
// C'EST LA TABLE QUI DECIDE SI UN PLAFOND PEUT MORDRE UN TOIT. Un toit n'est pas suspendu
// loin au-dessus de l'arene : il est juste au-dessus. Si la matiere d'une carte tient
// entierement dans les premiers metres au-dessus du sol joue, alors tout plafond pose a
// « hauteur frequentee + marge » passe AU-DESSUS d'elle et ne coupe rien — quelle que soit la
// qualite du corpus. Elle se lit SANS corpus, donc elle couvre aussi les cartes sans film.
func tableauMatiere(sb *strings.Builder, lignes []ligne) {
	fmt.Fprintln(sb, "## Ou vit la matiere dessinee (ecart au sol joue)")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "Centiles de l'altitude des pixels porteurs de matiere, en metres AU-DESSUS du sol")
	fmt.Fprintln(sb, "joue. `h max - sol` rappelle a quelle hauteur relative monte le corpus.")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "| carte | sol joue | matiere p50 | p90 | p99 | max | h max - sol | seuil - sol |")
	fmt.Fprintln(sb, "|---|---:|---:|---:|---:|---:|---:|---:|")
	for i := range lignes {
		l := &lignes[i]
		if l.geom == nil || l.geom.pixels.taille() == 0 {
			continue
		}
		sol := l.geom.niveauJeu
		hRel, seuilRel := math.NaN(), math.NaN()
		if l.freq != nil && l.freq.taille() > 0 {
			hRel, seuilRel = l.freq.maximum()-sol, l.seuil-sol
		}
		fmt.Fprintf(sb, "| `%s` | %s | %s | %s | %s | %s | %s | %s |\n", l.c.module, metres(sol),
			metres(l.geom.pixels.centile(0.5)-sol), metres(l.geom.pixels.centile(0.9)-sol),
			metres(l.geom.pixels.centile(0.99)-sol), metres(l.geom.pixels.maximum()-sol),
			metres(hRel), metres(seuilRel))
	}
	fmt.Fprintln(sb, "")
}

func annexes(sb *strings.Builder, lignes []ligne, freq *frequentation, ecartees []string) {
	fmt.Fprintln(sb, "## Cartes ecartees de la mesure")
	fmt.Fprintln(sb, "")
	if len(ecartees) == 0 {
		fmt.Fprintln(sb, "Aucune.")
	}
	for _, e := range ecartees {
		fmt.Fprintf(sb, "- %s\n", e)
	}
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "## Films non rattaches a une carte")
	fmt.Fprintln(sb, "")
	fmt.Fprintln(sb, "Un film ECARTE ne pese sur aucune hauteur : mieux vaut une carte sans corpus qu'une")
	fmt.Fprintln(sb, "carte dont la hauteur maximale vient d'un match joue ailleurs.")
	fmt.Fprintln(sb, "")
	aucun := true
	for i := range freq.films {
		if freq.films[i].module == "" {
			fmt.Fprintf(sb, "- `%s` : %s\n", freq.films[i].id, freq.films[i].ecart)
			aucun = false
		}
	}
	if aucun {
		fmt.Fprintln(sb, "Aucun.")
	}
	fmt.Fprintln(sb, "")
	degradations(sb, lignes)
}

func degradations(sb *strings.Builder, lignes []ligne) {
	fmt.Fprintln(sb, "## Degradations de cuisson")
	fmt.Fprintln(sb, "")
	aucune := true
	for i := range lignes {
		if lignes[i].geom == nil || len(lignes[i].geom.degradations) == 0 {
			continue
		}
		fmt.Fprintf(sb, "- `%s` : %s\n", lignes[i].c.module, strings.Join(lignes[i].geom.degradations, " · "))
		aucune = false
	}
	if aucune {
		fmt.Fprintln(sb, "Aucune.")
	}
	fmt.Fprintln(sb, "")
}

// niveauDe / plafondDe rendent le sol joue et le plafond actuel, NaN tant que la carte n'a
// pas ete cuite (ils viennent du bilan de cuisson, pas du sidecar : c'est la meme chaine).
func niveauDe(l *ligne) float64 {
	if l.geom == nil {
		return l.c.niveauJeu
	}
	return l.geom.niveauJeu
}

func plafondDe(l *ligne) float64 {
	if l.geom == nil {
		return math.NaN()
	}
	return l.geom.plafondActuel
}

// metres formate une altitude, ou un tiret quand la mesure n'existe pas — jamais un zero,
// qui se confondrait avec une mesure.
func metres(v float64) string {
	if math.IsNaN(v) {
		return "—"
	}
	return fmt.Sprintf("%.1f", v)
}

func part(v float64) string {
	if math.IsNaN(v) {
		return "—"
	}
	return fmt.Sprintf("%.2f %%", 100*v)
}

func volumes(c coupe) string {
	if c.total == 0 {
		return "—"
	}
	return fmt.Sprintf("%d supprimes (%.2f %%) · %d decapites (%.2f %%) / %d",
		c.supprimes, 100*c.partSupprimes(), c.decapites, 100*c.partDecapites(), c.total)
}

// zonesOuTiret rend la colonne des zones nommees : un tiret quand aucun seuil n'a pu etre
// pose (pas de corpus), le compte sinon — un « 0 » sans seuil serait une fausse assurance.
func zonesOuTiret(l *ligne) string {
	if math.IsNaN(l.seuil) {
		return "—"
	}
	return zones(l.zones)
}

// zones rend les zones nommees au-dessus du seuil, tronquees a quatre : la colonne doit rester
// lisible, et le compte total est ce qui decide.
func zones(noms []string) string {
	if len(noms) == 0 {
		return "0"
	}
	if len(noms) <= 4 {
		return fmt.Sprintf("%d : %s", len(noms), strings.Join(noms, ", "))
	}
	return fmt.Sprintf("%d : %s, ...", len(noms), strings.Join(noms[:4], ", "))
}
