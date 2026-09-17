// Package domain — compare_weapons.go : LE PROFIL D'ARMES DU FACE-À-FACE
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md).
//
// # DEUX FAMILLES DE TYPES, ET ELLES NE SE CONFONDENT PAS
//
// `CompareWeaponScope` est une ENTRÉE : l'ensemble de matchs sur lequel un côté est mesuré,
// lu par le repo et jamais sérialisé. `CompareWeaponProfile` et ses blocs sont la SORTIE : le
// contrat de réponse, additif à `CompareResponse`. Les premiers portent des match_id, les
// seconds n'en portent aucun — une page ne publie pas la liste des matchs qu'elle a lus.
package domain

// CompareWeaponScope est L'ENSEMBLE DE MATCHS d'un côté de la comparaison, et ce que le
// joueur y a fait.
//
// # POURQUOI LES MATCH_ID ET LES TOTAUX VOYAGENT ENSEMBLE
//
// Les deux lecteurs d'armes (frags par arme, frags mesurés) se bornent par `MatchIDs` — sans
// quoi ils balaieraient une table PARTAGÉE, tous joueurs confondus, ce que leur `Validate()`
// refuse. Les totaux, eux, sont les DÉNOMINATEURS de la publication : « N frags mesurés sur
// M » et la part de chaque classe de frags. Les tirer d'une seconde requête les ferait porter
// sur un scope potentiellement différent (une écriture concurrente entre les deux lectures
// suffit) ; ils sont donc agrégés SUR LES MÊMES LIGNES que les match_id.
//
// # UN SEUL SCOPE, LE MÊME POUR LES DEUX JOUEURS (D2, amendé au lot 3-bis le 2026-09-17)
//
// Tous les matchs du joueur PRÉSENTS DANS LA BASE PARTAGÉE, campagne exclue — la même doctrine
// que les métriques déjà publiées par la page. Un second scope « croisé » (l'intersection avec
// les matchs du joueur courant) était prévu pour un joueur non suivi ; il a été retiré parce
// qu'il lisait la même table avec la même exclusion et n'y ajoutait qu'un `EXISTS` : son
// résultat était un SOUS-ENSEMBLE du scope lifetime, donc « croisé non vide » impliquait
// « lifetime non vide » et la branche qui l'appelait ne pouvait jamais être prise.
//
// Ce qu'il fallait dire au lecteur — « ce joueur, on ne l'a vu que quelques fois » — est porté
// par `Matches`, publié pour les deux joueurs.
type CompareWeaponScope struct {
	// MatchIDs borne les deux lectures d'armes. Jamais vide quand le scope existe : un
	// scope sans match n'est pas construit (le repo rend (nil, nil)).
	MatchIDs []string
	// Matches est le nombre de matchs du scope — publié tel quel (« sur N matchs »).
	Matches int
	// Kills, Deaths sont les totaux du joueur sur le scope : dénominateurs de la part des
	// frags par classe et de la couverture de la portée.
	Kills, Deaths int
	// MeleeKills, GrenadeKills sont les compteurs NATIFS de l'API. Ils ne se déduisent pas
	// du registre d'armes : la mêlée et la grenade sont servies par ces compteurs, jamais
	// par une classification d'arme (cf. service/fragdist).
	MeleeKills, GrenadeKills int
}

// CompareWeaponProfile est le PROFIL D'ARMES d'une comparaison : les deux joueurs côte à côte,
// trois blocs chacun.
//
// AUCUN VAINQUEUR N'EST ÉLU (D3), et ce n'est pas un oubli : une distance plus longue n'est pas
// meilleure, une part de frags décrit un STYLE. La page compare des manières de jouer, pas des
// performances — élire un gagnant ferait dire à ces nombres ce qu'ils ne mesurent pas.
type CompareWeaponProfile struct {
	PlayerA CompareWeaponSide `json:"player_a"`
	PlayerB CompareWeaponSide `json:"player_b"`
}

// CompareWeaponSide est le profil d'UN joueur : son scope, sa répartition par classe, sa
// portée par rôle et ses trois armes les plus meurtrières.
//
// CHAQUE BLOC EST INDÉPENDAMMENT ABSENT (D9). Un titre sans positions par kill n'a pas de
// portée mais a des classes et un top 3 ; un joueur dont aucune arme n'est résolue n'a pas de
// top 3 mais a ses compteurs natifs. Faire tomber le profil entier au premier bloc manquant
// cacherait ce qui est pourtant mesuré.
type CompareWeaponSide struct {
	// Matches est la taille du scope (D2) — le nombre de matchs de ce joueur PRÉSENTS DANS
	// LA BASE PARTAGÉE, campagne exclue.
	//
	// IL EST PUBLIÉ POUR LES DEUX JOUEURS, TOUJOURS, et c'est lui qui porte le « sur N
	// matchs » du front. Un drapeau `is_sample` a existé ici jusqu'au lot 3-bis
	// (2026-09-17) pour distinguer une carrière d'un échantillon croisé ; il a été retiré
	// avec le scope croisé, qui était un sous-ensemble du scope lifetime et n'était donc
	// jamais atteint. Ce qu'il prétendait dire, ce nombre le dit déjà : un joueur peu vu
	// affiche un petit N, et le lecteur en tire la même conclusion sans qu'on la lui
	// qualifie.
	Matches int `json:"matches"`
	// TotalKills est le DÉNOMINATEUR des parts de FragClasses — les frags du joueur sur le
	// scope, tels que les compte la base, jamais la somme des classes (qui, elle, peut
	// laisser un résidu « non attribué » et c'est précisément ce que la part doit montrer).
	TotalKills int `json:"total_kills"`
	// FragClasses : la part des frags par classe. TOUJOURS sérialisée (`[]`, jamais null) :
	// le front itère sans garde.
	FragClasses []CompareFragClass `json:"frag_classes"`
	// Range : la portée par RÔLE (D1/D8). `WeaponRangeRow.weapon_key` porte une clé de
	// RÔLE (`precision`, `automatic`, ...) et non une clé d'arme ; `label`/`label_en`
	// restent VIDES, le front résolvant le libellé depuis son manifeste. `Opening` est
	// toujours nil : l'entame est une lecture de la Synthèse, hors du périmètre de la page.
	// Nil si le titre ne produit pas de positions par kill, ou si aucun frag n'est mesuré.
	Range *SynthesisWeaponRange `json:"range,omitempty"`
	// TopWeapons : les trois armes les plus meurtrières, 3 au plus. TOUJOURS sérialisée
	// (`[]`, jamais null).
	TopWeapons []CompareTopWeapon `json:"top_weapons"`
}

// CompareFragClass est UNE classe d'arme et la part des frags du joueur qu'elle porte.
type CompareFragClass struct {
	// Class est une clé `domain.FragClass*` (le front résout `frags.class.<clé>`), jamais
	// un libellé : aucun nom FR/EN ne s'écrit côté Go (D10).
	Class string `json:"class"`
	Kills int    `json:"kills"`
	// SharePct est la part en POURCENTAGE 0..100 (convention `*Pct` du dépôt). La classe
	// résiduelle « non attribué » est CONSERVÉE : sans elle les parts ne sommeraient pas à
	// 100 et le lecteur attribuerait la différence à une erreur d'arrondi.
	SharePct float64 `json:"share_pct"`
}

// CompareTopWeapon est UNE arme du top 3 : son nom, ses frags, son icône.
//
// LE NOM VIENT DE LA CHAÎNE weapon_names.toml DU TITRE (porté jusqu'ici par
// `port.WeaponKillRow.Label/LabelEN`), jamais d'un libellé écrit en Go : ces noms sont propres
// à chaque titre, et les recopier côté serveur dupliquerait le TOML du titre.
type CompareTopWeapon struct {
	Label   string `json:"label"`
	LabelEN string `json:"label_en,omitempty"`
	Kills   int    `json:"kills"`
	// Class / Role : les dimensions registre de l'arme, pour que le front puisse la
	// colorer comme le reste de la page. Vides si le registre ne les connaît pas.
	Class string `json:"class,omitempty"`
	Role  string `json:"role,omitempty"`
	// ImageURL est l'URL de l'icône, vide si le titre n'en a pas (le front affiche alors
	// le nom seul). ImageTinted dit que l'icône est un MASQUE à teinter et non un dessin
	// fini — le rendu diffère, et se tromper donne une silhouette noire.
	ImageURL    string `json:"image_url,omitempty"`
	ImageTinted bool   `json:"image_tinted,omitempty"`
}
