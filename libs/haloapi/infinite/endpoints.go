package infinite

import "strings"

// Cle : la clé d'un hôte de service. Les huit premières valeurs sont celles de la section
// `[endpoints]` de `config/titles/{slug}/constants.toml` (mêmes chaînes) : l'appelant construit
// des [Endpoints] depuis son résolveur de titre clé pour clé.
type Cle string

// Clés des hôtes.
const (
	CleStats        Cle = "stats"
	CleGameCMS      Cle = "gamecms"
	CleEconomy      Cle = "economy"
	CleSkill        Cle = "skill"
	CleUGCFilm      Cle = "ugc_film"
	CleDiscoveryUGC Cle = "discovery_ugc"
	CleChallenges   Cle = "challenges"
	CleNameplate    Cle = "nameplate"
	// CleStatsSansPort : l'hôte des stats SANS `:443`, celui du service record, de la
	// confidentialité et de `FetchMatchStats` du fournisseur. Distinct de [CleStats] (avec `:443`,
	// historique et stats du client de synchronisation) : les deux formes sont « byte-distinctes »
	// dans l'en-tête Host et chaque appelant garde la sienne.
	CleStatsSansPort Cle = "stats_sans_port"
	// CleWaypointWeb : le site public (pages HTML du classement CSR mondial).
	CleWaypointWeb Cle = "waypoint_web"
)

// PrefixeJeuDefaut : le segment de jeu des chemins d'API de Halo Infinite.
const PrefixeJeuDefaut = "hi"

// hotesParDefaut : l'UNIQUE endroit des adresses par défaut des services Halo Infinite de la
// brique (une fonction, pas une variable : aucune valeur de paquet mutable).
func hotesParDefaut() map[Cle]string {
	return map[Cle]string{
		CleStats:         "https://halostats.svc.halowaypoint.com:443",
		CleGameCMS:       "https://gamecms-hacs.svc.halowaypoint.com",
		CleEconomy:       "https://economy.svc.halowaypoint.com",
		CleSkill:         "https://skill.svc.halowaypoint.com:443",
		CleUGCFilm:       "https://discovery-infiniteugc.svc.halowaypoint.com",
		CleDiscoveryUGC:  "https://discovery-infiniteugc.svc.halowaypoint.com",
		CleChallenges:    "https://halostats.svc.halowaypoint.com",
		CleNameplate:     "https://gamecms-hacs.svc.halowaypoint.com",
		CleStatsSansPort: "https://halostats.svc.halowaypoint.com",
		CleWaypointWeb:   "https://www.halowaypoint.com",
	}
}

// Endpoints : les hôtes des services et le segment de jeu des chemins. Valeur immuable : `Avec*`
// rend une copie.
type Endpoints struct {
	hotes   map[Cle]string
	prefixe string
}

// DefaultEndpoints rend les adresses de Halo Infinite.
func DefaultEndpoints() Endpoints {
	return Endpoints{hotes: hotesParDefaut(), prefixe: PrefixeJeuDefaut}
}

// Avec rend une copie où la clé `cle` vaut `hote` (barre finale retirée ; vide : l'adresse par
// défaut).
func (e Endpoints) Avec(cle Cle, hote string) Endpoints {
	out := Endpoints{hotes: make(map[Cle]string, len(e.hotes)+1), prefixe: e.prefixe}
	for k, v := range e.hotes {
		out.hotes[k] = v
	}
	out.hotes[cle] = strings.TrimRight(strings.TrimSpace(hote), "/")
	return out
}

// AvecPrefixeJeu rend une copie dont le segment de jeu vaut `p` (vide : [PrefixeJeuDefaut]).
func (e Endpoints) AvecPrefixeJeu(p string) Endpoints {
	out := e
	out.prefixe = strings.TrimSpace(p)
	return out
}

// Hote rend l'hôte de la clé ; une clé absente ou vide retombe sur l'adresse par défaut.
func (e Endpoints) Hote(cle Cle) string {
	if h := e.hotes[cle]; h != "" {
		return h
	}
	return hotesParDefaut()[cle]
}

// PrefixeJeu rend le segment de jeu des chemins (« hi » par défaut).
func (e Endpoints) PrefixeJeu() string {
	if e.prefixe == "" {
		return PrefixeJeuDefaut
	}
	return e.prefixe
}

// base rend « hôte/segment » pour la clé, forme commune des chemins d'API.
func (e Endpoints) base(cle Cle) string {
	return e.Hote(cle) + "/" + e.PrefixeJeu()
}
