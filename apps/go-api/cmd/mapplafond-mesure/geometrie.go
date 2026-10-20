package main

// geometrie.go — CE QUE LA CARTE DESSINE, ET A QUELLE ALTITUDE.
//
// Deux lectures de la meme carte, parce qu'elles ne repondent pas a la meme question :
//
//	LE PIXEL   la cuisson de production (himap.CuitCarteNative) rend le z-buffer. L'altitude
//	           d'un pixel est celle de la surface AFFICHEE : c'est elle que l'utilisateur voit,
//	           et donc elle qui dit ce qu'un plafond CHANGERAIT a l'image.
//	LE VOLUME  les instances du bloc `instanced geometry instances` du sbsp portent leur boite
//	           MONDE (AABBMin/AABBMax, champ @0x7C — cf. rendu.go:143, qui s'en sert comme
//	           bornes monde). Une instance entierement au-dessus du seuil est un volume que la
//	           coupe SUPPRIME ; une instance a cheval est un volume qu'elle DECAPITE.
//
// La coupe etudiee n'est pas une invention : `Rendu.Plafond` existe deja et la cuisson
// l'applique par la TRANCHE DE JEU, `[zJeu-12 ; zJeu+28]` (rendu.go:44). Le point 10 de
// l'encadre Notion revient a remplacer ce +28 universel par une valeur DEDUITE des hauteurs
// frequentees. L'instrument mesure ce que ce remplacement couperait, il ne le fait pas.

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"levelup/go-api/internal/himap"
)

// mesureGeom est ce qu'une carte rend a l'instrument.
type mesureGeom struct {
	module string
	// niveauJeu / plafondActuel : le sol joue deduit des ancres, et le plafond que la cuisson
	// applique aujourd'hui (`himap.TrancheDeJeu`).
	niveauJeu     float64
	plafondActuel float64
	// pixels porte la distribution d'altitude de la MATIERE DESSINEE, un compte par pixel.
	pixels *histogramme
	// instances : les instances retenues par la cuisson (hors supprimees en edition et hors
	// projecteurs d'ombre — meme tri que `himap.PeupleRendu`).
	instances []boiteInstance
	// degradations reprend celles de la cuisson : une carte cuite sans frontiere ou sans eau
	// n'est pas une carte fausse, mais le tableau doit le dire.
	degradations []string
}

// boiteInstance est la boite monde d'une instance, reduite a ce que la coupe interroge.
type boiteInstance struct {
	zBas  float64
	zHaut float64
}

// mesure une carte : la cuit, puis relit ses instances. Un seul module ouvert a la fois — le
// corpus entier dans un processus est une bombe RAM vecue.
//
// Le rendu est RENDU A L'APPELANT (et non garde) : c'est lui qui decide d'en tirer une planche,
// et il le relache aussitot apres — une carte cuite pese quelques millions de pixels.
func mesureGeometrie(ctx context.Context, racineJeu string, c carte) (mesureGeom, *himap.Rendu, error) {
	m := mesureGeom{module: c.module, pixels: nouvelHistogramme()}
	rendu, bilan, err := himap.CuitCarteNative(ctx, himap.OptionsCuisson{
		RacineDeploy: racineJeu, CheminModule: c.chemin, Ancres: c.ancres,
	})
	if err != nil {
		return m, nil, fmt.Errorf("cuisson de %s : %w", c.module, err)
	}
	m.niveauJeu = bilan.NiveauDeJeu
	_, m.plafondActuel = himap.TrancheDeJeu(bilan.NiveauDeJeu)
	m.degradations = bilan.Degradations
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if z, ok := rendu.Altitude(i, j); ok {
				m.pixels.ajoute(z)
			}
		}
	}
	if err := m.litInstances(c); err != nil {
		return m, nil, err
	}
	slog.InfoContext(ctx, "carte mesuree", "module", c.module,
		"pixels", m.pixels.taille(), "instances", len(m.instances),
		"niveauJeu", fmt.Sprintf("%.1f", m.niveauJeu),
		"plafondActuel", fmt.Sprintf("%.1f", m.plafondActuel))
	return m, rendu, nil
}

// litInstances relit les boites monde du bsp retenu par les ancres — le MEME bsp que la
// cuisson (`himap.ChoisitBSP`), sans quoi on compterait les volumes d'un horizon lointain.
func (m *mesureGeom) litInstances(c carte) error {
	bsps, err := himap.ReadModuleInstances(c.chemin)
	if err != nil {
		return fmt.Errorf("instances de %s : %w", c.module, err)
	}
	bsp := himap.ChoisitBSP(bsps, c.ancres)
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		m.instances = append(m.instances, boiteInstance{zBas: in.AABBMin[2], zHaut: in.AABBMax[2]})
	}
	return nil
}

// coupe est ce qu'un plafond donne ferait a une carte.
type coupe struct {
	seuil float64
	// partMatiere : part des pixels porteurs de matiere dont la surface AFFICHEE est au-dessus
	// du seuil — la part de l'image qui changerait.
	partMatiere float64
	// supprimes / decapites : instances entierement au-dessus du seuil, et instances a cheval.
	supprimes int
	decapites int
	// total : instances retenues, denominateur des deux precedents.
	total int
}

// evalueCoupe mesure ce qu'un plafond couperait sur une carte deja mesuree.
func (m *mesureGeom) evalueCoupe(seuil float64) coupe {
	c := coupe{seuil: seuil, partMatiere: m.pixels.partAuDessus(seuil), total: len(m.instances)}
	for _, b := range m.instances {
		switch {
		case b.zBas > seuil:
			c.supprimes++
		case b.zHaut > seuil:
			c.decapites++
		}
	}
	return c
}

func (c coupe) partSupprimes() float64 {
	if c.total == 0 {
		return math.NaN()
	}
	return float64(c.supprimes) / float64(c.total)
}

func (c coupe) partDecapites() float64 {
	if c.total == 0 {
		return math.NaN()
	}
	return float64(c.decapites) / float64(c.total)
}

// zonesAuDessus rend les ZONES NOMMEES de la carte dont la tranche habitee commence au-dessus
// du seuil.
//
// C'EST LE DETECTEUR DE FAUX POSITIFS INDEPENDANT DU CORPUS, et c'est lui qui compte. Une zone
// nommee est un espace de jeu DESSINE PAR LE DESIGNER (tag `levl`) : elle existe que le corpus
// local l'ait visitee ou non. Une zone dont le plancher est au-dessus du plafond propose est
// un etage praticable que la coupe supprimerait — pas un toit.
func (c *carte) zonesAuDessus(seuil float64) []string {
	var out []string
	for i := range c.prismes {
		if c.prismes[i].zBas > seuil {
			out = append(out, c.prismes[i].nom)
		}
	}
	return out
}
