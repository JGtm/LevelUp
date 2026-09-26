// Package sync — citations_weapons_source.go : la statistique `weapon_stat` du moteur de
// citations, lue dans la SOURCE DE DEGAT du film.
//
// # Ce que le moteur attend, et pourquoi ca ne change pas
//
// Les citations comptent des frags par NOM CANONIQUE d'arme (`name_en`) : « 1 000 frags au
// BR75 ». Ce contrat ne bouge pas — seule la voie qui l'alimente change. La voie historique
// partait d'un identifiant numerique (`weapon_kills`) ; celle-ci part d'un tag de source de
// degat, traduit en cle de registre puis en nom.
//
// # Pourquoi la traduction est en Go
//
// La table qui traduit un tag `jpt!` en entree de registre vit dans le binaire et evolue a
// chaque saison. La porter en SQL obligerait a la copier dans la base et a l'y maintenir
// synchronisee (decision D12 du plan du 2026-09-01). La requete ne manipule donc que des
// entiers.
package sync

import (
	"context"
	"database/sql"

	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/killcollector"
)

// citationWeaponSource porte de quoi compter les frags par arme depuis le film : le
// traducteur du titre, et la table « cle de registre -> nom canonique » qui rend le
// resultat comparable a celui de la voie historique.
//
// Zero-valeur (classifier nil) = titre sans decodeur de film : les appelants lisent alors
// `v_weapon_kills`, exactement comme avant la bascule.
type citationWeaponSource struct {
	classifier port.KillSourceClassifier
	keyNames   map[string]string // weapon_key -> name_en
}

// actif dit si cette source peut servir. Un traducteur sans table de noms ne sert a rien :
// il produirait des cles brutes la ou le moteur attend des noms canoniques.
func (s citationWeaponSource) actif() bool {
	return s.classifier != nil && len(s.keyNames) > 0
}

// loadWeaponKeyNames charge « cle de registre -> nom canonique EN » depuis la metadata.
//
// Le nom vient de `weapon_name_labels` (source unique keyee par cle, V72-06) et retombe sur
// le nom du registre. Best-effort : une metadata non seedee rend une table vide, la source
// est alors inactive et l'appelant garde sa voie historique.
func loadWeaponKeyNames(ctx context.Context, db *sql.DB, titleSlug string) (map[string]string, error) {
	const q = `
SELECT w.weapon_key,
       COALESCE(NULLIF(MIN(wnl.name_en), ''), NULLIF(MIN(w.name), ''), '') AS name_en
FROM weapons w
LEFT JOIN weapon_name_labels wnl
       ON wnl.title_slug = w.title_slug AND wnl.weapon_key = w.weapon_key
WHERE w.title_slug = ?
GROUP BY w.weapon_key`
	rows, err := db.QueryContext(ctx, q, titleSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key, name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, err
		}
		if name != "" {
			out[key] = name
		}
	}
	return out, rows.Err()
}

// loadWeaponKillsFromSource compte les frags par arme d'un joueur sur un match, depuis la
// source de degat. Meme forme de resultat que la voie historique : nom canonique -> frags.
func loadWeaponKillsFromSource(
	ctx context.Context,
	db *sql.DB,
	src citationWeaponSource,
	matchID, xuid string,
) (map[string]int, error) {
	const q = `
SELECT k.source_tag, COUNT(*)::INTEGER AS kills
FROM match_kill_events_latest k
WHERE k.match_id = ? AND k.feed_killer_xuid = ? AND k.source_tag IS NOT NULL
GROUP BY k.source_tag`
	rows, err := db.QueryContext(ctx, q, matchID, xuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var tag uint32
		var kills int
		if err := rows.Scan(&tag, &kills); err != nil {
			return nil, err
		}
		key, ok := src.classifier.KillSourceRegistryKey(tag)
		if !ok {
			continue // source hors registre : ne compte pour aucune arme (D7)
		}
		if name, known := src.keyNames[key]; known {
			result[name] += kills
		}
	}
	return result, rows.Err()
}

// CitationWeaponSource : de quoi armer la lecture des frags par arme depuis le film.
//
// Le TITRE y figure avec le traducteur, et pas separement : la table des noms canoniques
// est propre au titre, et un traducteur sans son titre ne permettrait pas de la charger.
// Les deux voyagent donc ensemble ou pas du tout.
type CitationWeaponSource struct {
	TitleSlug  string
	Classifier port.KillSourceClassifier
}

// citationWeaponSourceDuMoteur rend l'option de source de degat du titre courant, ou une
// option vide si le titre n'a pas de decodeur de film.
//
// Elle est resolue par le pont `killcollector`, qui porte deja l'import title-specific et
// la garde de capability — le paquet `sync` racine n'a donc rien a savoir du titre.
func (e *SyncEngine) citationWeaponSourceDuMoteur() []CitationWeaponSource {
	c := killcollector.ClassifierPourTitre(e.repoRoot, e.titleSlug)
	if c == nil {
		return nil
	}
	return []CitationWeaponSource{{TitleSlug: e.titleSlug, Classifier: c}}
}
