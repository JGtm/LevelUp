package main

// libelles.go — LA TRADUCTION VIT ICI, PAS DANS LE PAQUET.
//
// Le paquet `killsource` ne porte AUCUN libelle FR/EN : il publie des identifiants stables
// (`ARME`, `SilentMelee`, `marche`...) et laisse la couche d affichage decider. C est la regle du
// depot, et c est aussi ce qui permettra a l interface web de traduire autrement que cette CLI.
//
// REGLE D HONNETETE DE CE FICHIER : on ne traduit que ce que le chantier a ETABLI. Une categorie
// dont la semantique n a pas ete confirmee garde son identifiant moteur — inventer un mot francais
// plausible sur un champ mal compris, c est fabriquer de l information.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// natureFR : la nature de la source, en clair.
func natureFR(c damagetag.Class) string {
	switch c {
	case damagetag.ClassArme:
		return "arme"
	case damagetag.ClassMelee:
		return "melee"
	case damagetag.ClassGrenade:
		return "grenade"
	case damagetag.ClassVehicule:
		return "vehicule"
	case damagetag.ClassObjet:
		return "objet explosif"
	case damagetag.ClassGlobal:
		return "environnement"
	case damagetag.ClassInconnu:
		return "non remontee"
	default:
		return string(c)
	}
}

// statutFR : ce que le consommateur a le droit de faire de l etiquette. Une ligne VALIDE ne dit
// rien : le silence est le cas normal, et seul l inhabituel merite un mot.
func statutFR(s damagetag.Status) string {
	switch s {
	case damagetag.StatusValide:
		return ""
	case damagetag.StatusReserve:
		return "nom sous reserve"
	case damagetag.StatusAmbigu:
		return "nom ambigu, non publie"
	case damagetag.StatusInconnu:
		return "aucune source remontee"
	default:
		return string(s)
	}
}

// categorieFR : le modificateur de rapport de degat.
//
// QUATRE SONT ETABLIS PAR LE CHANTIER et se traduisent ; les autres gardent leur identifiant
// moteur entre parentheses, parce qu aucune verite terrain ne les a confirmes. Rappel mesure :
// la categorie ne discrimine PAS grenade et melee ordinaire — les deux sortent `None`, c est le
// TAG qui les porte (RE_LOG 7ter.37).
func categorieFR(c decfilm.Category) string {
	switch c {
	case decfilm.CategoryNone:
		return ""
	case decfilm.CategoryHeadshot:
		return "tir a la tete"
	case decfilm.CategoryHeadshotMultiplier:
		return "tir a la tete (multiplicateur)"
	case decfilm.CategorySilentMelee:
		return "assassinat"
	case decfilm.CategoryAttachedDamage:
		return "projectile fixe sur la cible"
	case decfilm.CategoryCollisionDamage:
		return "collision"
	default:
		return fmt.Sprintf("(%s)", c.Name())
	}
}

// voieFR : par quelle voie la ligne a ete lue. A PONDERER, pas a interpreter : les deux lisent le
// MEME champ, au MEME bit quand elles repondent toutes les deux (346/346, desaccord 0).
func voieFR(p decfilm.Path) string {
	switch p {
	case decfilm.PathWalk:
		return "sequentielle"
	case decfilm.PathScan:
		return "balayage"
	default:
		return string(p)
	}
}

// origineFR : comment la ligne a ete appariee au kill-feed.
func origineFR(o decfilm.Origin) string {
	switch o {
	case decfilm.OriginCredit:
		return "les deux verites concordent"
	case decfilm.OriginSelfSource:
		return "source appartenant a la victime"
	case decfilm.OriginBot:
		return "mort de bot (absente du kill-feed)"
	case decfilm.OriginBotKiller:
		return "mort infligee par un bot (kill absent du kill-feed)"
	default:
		return string(o)
	}
}

// mmss : un instant du film, en minutes:secondes depuis son debut.
func mmss(ms int) string { return fmt.Sprintf("%02d:%02d", ms/60000, (ms/1000)%60) }

// pct : un pourcentage lisible, ou "-" quand le denominateur est vide. Un taux sans denominateur
// n existe pas : on ne l affiche pas.
func pct(num, den int) string {
	if den <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f %%", 100*float64(num)/float64(den))
}
