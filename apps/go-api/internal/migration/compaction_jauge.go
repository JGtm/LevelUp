package migration

// compaction_jauge.go — LA JAUGE « PASSES PAR MATCH » : ce que la compaction retirerait, mesuré
// en LECTURE SEULE, table par table.
//
// Le rapport d'une table est `brutes / gardées`, où « gardées » est la cardinalité de la règle du
// registre (requeteGardees) — exactement ce que la compaction laisserait. Une table compactée vaut
// donc 1, sans exception. Le dénominateur n'est PAS le compte de la vue `_latest` : une vue qui
// filtre en plus des lignes de la passe retenue (cf. `compactable.keep`) donnerait un rapport > 1
// à une table déjà compacte, et la compaction de nuit se déclencherait chaque nuit pour rien.
//
// Lecteurs : le contrôle de santé des données (scheduler/data_health_passes.go) la publie, et le
// script hôte de nuit (scripts/compact-passes-nuit.sh) décide sur elle s'il arrête le serveur.

import (
	"context"
	"database/sql"
	"fmt"
)

// SeuilPassesParMatch : rapport brutes / gardées au-delà duquel une table mérite d'être compactée.
//
// Le coût d'une lecture par une vue `_latest` suit les lignes BRUTES qu'elle parcourt (mesure
// consignée dans l'ADR 0026, section « Automatic compaction »). À 1,5, une lecture paie au plus
// 50 % de lignes mortes avant que la compaction de nuit ne ramène la table à 1 ; plus bas, le
// serveur s'arrêterait pour un gain invisible ; plus haut, une demi-recuisson ralentirait toutes
// les pages jusqu'à la nuit suivante.
const SeuilPassesParMatch = 1.5

// JaugePasses : la mesure d'une table du registre.
type JaugePasses struct {
	Table   string
	Brutes  int64 // lignes de la table
	Gardees int64 // lignes que la compaction garderait (la passe retenue)
}

// Rapport rend brutes / gardées ; une table vide vaut 1 (rien à retirer).
func (j JaugePasses) Rapport() float64 {
	if j.Gardees <= 0 || j.Brutes <= 0 {
		return 1
	}
	return float64(j.Brutes) / float64(j.Gardees)
}

// AuDessusDuSeuil dit si la table dépasse SeuilPassesParMatch.
func (j JaugePasses) AuDessusDuSeuil() bool {
	return j.Rapport() > SeuilPassesParMatch
}

// MesurerPasses mesure UNE table du registre, sans aucune écriture. `presente` est faux quand la
// table n'existe pas dans cette base (titre qui ne la porte pas) ; une table inconnue du registre
// est une erreur.
func MesurerPasses(ctx context.Context, db *sql.DB, table string) (j JaugePasses, presente bool, err error) {
	c, ok := compactableParNom(table)
	if !ok {
		return j, false, fmt.Errorf("jauge des passes: %s n'est pas une table du registre de compaction", table)
	}
	j.Table = table
	hasTable, err := tableExists(db, table)
	if err != nil {
		return j, false, fmt.Errorf("jauge des passes %s: lecture du catalogue: %w", table, err)
	}
	if !hasTable {
		return j, false, nil
	}
	if err := compter(ctx, db, "SELECT COUNT(*) FROM "+table, &j.Brutes); err != nil {
		return j, true, fmt.Errorf("jauge des passes %s: compte brut: %w", table, err)
	}
	if err := compter(ctx, db, requeteGardees(c), &j.Gardees); err != nil {
		return j, true, fmt.Errorf("jauge des passes %s: compte des lignes gardées: %w", table, err)
	}
	return j, true, nil
}

func compactableParNom(table string) (compactable, bool) {
	for _, c := range tablesCompactables {
		if c.Table == table {
			return c, true
		}
	}
	return compactable{}, false
}
