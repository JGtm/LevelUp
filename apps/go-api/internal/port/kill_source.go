// Package port — kill_source.go : la traduction d'une SOURCE DE DEGAT du film.
//
// Ce fichier ne porte que des CONTRATS, jamais de donnees. La table qui traduit un tag
// `jpt!` en objet est propre au titre (Halo Infinite : `film/killicon` adossee a
// `film/damagetag`) ; l'injecter au cablage garde `platform/duckdb` title-agnostic —
// meme motif que la ModeTaxonomy injectee dans MatchViewRepo.
//
// Il remplace l'ancien `kill_source_class.go`, supprime avec le second chemin de
// chargement (decision D11 du plan du 2026-09-01 : les kills par arme et les kills hors
// arsenal ne sont plus deux voies a departager, mais une seule lecture).
package port

import "context"

// KillSourceClassifier traduit une SOURCE DE DEGAT du film en cle du registre d'armes.
//
// Second retour faux = cette source ne designe aucune entree de registre. Le kill n'est
// alors PAS remonte : il reste dans « Non attribue ». C'est la decision D7 du plan — on
// ne devine pas, on ne proratise pas.
type KillSourceClassifier interface {
	// KillSourceRegistryKey rend la weapon_key du registre pour une source de degat.
	KillSourceRegistryKey(sourceTag uint32) (string, bool)
}

// KillSourceDescriber NOMME la classe d'une source de degat, pour la JOURNALISATION
// seule.
//
// POURQUOI C'EST UNE INTERFACE A PART, ET OPTIONNELLE. Un kill que le rejeu 2D sait
// nommer mais que le graphe classe « Non attribue » ne doit pas disparaitre en silence
// (decision D13). Pour le dire, le lecteur doit pouvoir citer la CLASSE de la source
// ecartee — une information que seul le titre possede. Mais aucune decision de
// comportement n'en depend : un titre qui ne l'implemente pas voit simplement ses
// compteurs regroupes sous une classe inconnue. La rendre obligatoire ferait payer a tous
// les titres le prix d'une ligne de journal.
type KillSourceDescriber interface {
	// KillSourceClassName rend le nom de la classe de la source (« ARME », « VEHICULE »,
	// « MELEE »…). Second retour faux = tag inconnu de la table du titre.
	KillSourceClassName(sourceTag uint32) (string, bool)
}

// KillSourceCategorizer range une source de degat dans une CATEGORIE canonique
// (domain.KillSourceCategory*) — objet explosif du decor, chute et environnement.
//
// POURQUOI UNE TROISIEME INTERFACE, ET OPTIONNELLE. La cle de registre (Classifier) ne
// suffit pas a compter un bidon : la plupart des objets explosifs du decor n'ont AUCUNE
// cle (un seul tag par type de bobine en porte une), et leurs frags retombaient en
// « Non attribue ». La CATEGORIE, elle, est certaine pour chacun. Seul le titre sait la
// lire dans ses propres classes de source ; un titre sans film n'en fournit pas, et le
// lecteur rend alors games.ErrCapabilityNotSupported (degradation propre).
//
// Consommateur : « Outils de destruction » de l'Escouade (decision D8 du plan du
// 2026-09-26). La « Repartition des frags » (fragdist) ne la lit pas.
type KillSourceCategorizer interface {
	// KillSourceCategory rend la categorie canonique de la source. Second retour faux =
	// la source n'appartient a aucune categorie (arme, melee, grenade, vehicule, inconnue).
	KillSourceCategory(sourceTag uint32) (string, bool)
}

// KillSourceCategoryRow : frags d'un joueur dont la source appartient a une categorie.
// WeaponKey est la cle de registre de la source quand elle en a une ("" sinon) : elle
// dit quelles lignes par arme du meme joueur ces frags recouvrent, pour ne pas les
// compter deux fois.
type KillSourceCategoryRow struct {
	XUID      string
	Category  string
	WeaponKey string
	Kills     int
}

// KillSourceCategoryRepository est l'interface OPTIONNELLE du lecteur adosse a la source
// de degat : les frags par (joueur, categorie de source), sur les memes filtres que
// WeaponKillsRepository. Decouverte par assertion ; le lecteur historique (titre sans
// film) ne l'implemente pas.
type KillSourceCategoryRepository interface {
	LoadKillSourceCategoryKills(
		ctx context.Context,
		slug string,
		filters WeaponKillFilters,
	) ([]KillSourceCategoryRow, error)
}
