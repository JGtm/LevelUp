package filmdec

// film_context.go — CE QUE TOUS LES BALAYAGES D'UN MEME FILM SE PARTAGENT.
//
// # CE QUE CE FICHIER FERME
//
// Lot 2 de PLAN_CUISSON_PERF (2026-09-03). Le lot 1 avait supprime les ~36 relectures du film :
// les balayages recoivent un `*filmsource.Film` decompresse UNE fois. Restait le second etage du
// meme defaut — chaque balayage RECALCULAIT, sur ce film deja charge, les trois memes
// derivations :
//
//	la BANDE DE SLOTS BIPEDE   `bipedSlotBand` — une marche de l'image-cle de tete de CHAQUE
//	                            chunk. Huit balayages la relevaient (positions, ramassages
//	                            natifs, et les six canaux delta).
//	le DECOUPAGE D'i0          `DetectI0LayoutOf` — six chunks marches bit a bit, plus SA
//	                            PROPRE bande. Six balayages delta le redetectaient.
//	le REGISTRE chunk_00       `ParseRegistryChunk` — une douzaine de re-analyses par cuisson,
//	                            une par accesseur d'archetype.
//
// Ces trois valeurs ne dependent QUE du film. Les recalculer par balayage etait du temps pur.
// [FilmContext] les porte, construit UNE fois par `replay.BuildFromFilm`, et les balayages le
// recoivent a la place du film.
//
// # POURQUOI LA MEMOISATION EST PARESSEUSE, ET NON CALCULEE AU CONSTRUCTEUR
//
// Le lot 2 est un REFACTO PUR : les sorties doivent etre identiques a l'octet, et l'ORDRE des
// etapes observees ne doit pas bouger (garde `replay/observe_test.go`, harnais `cmd/replay-equiv`).
// Un constructeur qui calculerait tout d'avance deplacerait le premier calcul AVANT le premier
// balayage — donc avant l'installation des largeurs d'axe et avant le demarrage de l'horloge des
// etapes — et ferait travailler un film qui echoue des les positions. Paresseux, le premier
// calcul a lieu EXACTEMENT la ou il avait lieu avant (le premier balayage qui en a besoin) ;
// les suivants le lisent. C'est aussi ce qui garantit « jamais pire qu'avant » aux enveloppes D2,
// qui construisent leur propre contexte a chaque appel.
//
// # POURQUOI LE CONSTRUCTEUR NE REND PAS D'ERREUR
//
// Les trois derivations ECHOUENT sur des films legitimes — une bobine partielle n'a pas de
// `chunk_00`, un film trop court ne donne pas trois frontieres nettes dans i0 — et chaque
// balayage rend AUJOURD'HUI son propre message a son propre moment (`ErrNoRegistryChunk`,
// « decoupage i0 illisible », « aucun slot biped (ti=35) ... »). Refuser au constructeur
// changerait ces messages ET l'etape a laquelle la cuisson s'arrete : la fixture
// `replay/testdata/minifilm_000d5950` n'a ni registre ni slot bipede, et
// `replay/zero_disque_test.go` exige l'erreur EXACTE des positions. Le contexte MEMORISE donc
// l'echec au lieu de le lever, et chaque accesseur le rejoue a l'identique, autant de fois qu'on
// le lui demande.
//
// # CE QU'IL N'EST PAS
//
// Ni un cache global (aucun `var` de paquet — le ratchet `archlint/filmdec_package_vars_test.go`
// gele leur compte), ni un objet partageable entre goroutines : il n'est ni verrouille ni
// atomique, et il vit sous le meme `LockProcessDecode` que le decodage qu'il sert.

import "levelup/go-api/internal/analysis/filmsource"

// FilmContext porte les derivations d'un film qui ne dependent que de lui : les numeros de ses
// chunks de donnees, la bande de slots bipede, le decoupage d'i0 AUTO-DETECTE et le registre
// chunk_00. Il se construit par [NewFilmContext] et se passe aux balayages.
//
// LE DECOUPAGE EST L'AUTO-DETECTE, PAS CELUI DU CATALOGUE, et c'est la decision D3bis du plan :
// le lot 2 ne change aucune sortie, il ne fait que calculer une fois ce qui l'etait six. La
// bascule vers `opt.MapQuant.Layout()` — qui CHANGE la sortie sur les cartes a plus de deux
// regions (Live Fire) — est le lot 3, et lui seul.
//
// Les champs sont PRIVES : un layout ou un registre se lit avec l'erreur qui va avec (cf.
// l'en-tete), et une bande de slots exposee en clair serait modifiable par son lecteur.
type FilmContext struct {
	film *filmsource.Film

	chunks    []int
	chunksLus bool

	slots    map[uint32]bool
	slotsLus bool

	lay    I0Layout
	layErr error
	layLu  bool

	reg    *Registry
	regErr error
	regLu  bool
}

// NewFilmContext ouvre le contexte d'un film DEJA CHARGE. Il ne lit rien : chaque derivation est
// calculee au premier accesseur qui la demande, puis memorisee (cf. l'en-tete du fichier).
//
// `film` nil est ACCEPTE et n'est pas une erreur : la cuisson passe un film nil quand les chunks
// sont illisibles (`replaybuild.chargerFilm`), et chaque balayage rend alors son
// [ErrNoFilmChunk] a sa place — exactement comme un repertoire vide avant le lot 1.
func NewFilmContext(film *filmsource.Film) *FilmContext {
	return &FilmContext{film: film}
}

// Film rend le film sous-jacent, pour les balayages qui lisent des chunks sans rien deriver.
func (c *FilmContext) Film() *filmsource.Film {
	if c == nil {
		return nil
	}
	return c.film
}

// ChunkNumbers rend les numeros des chunks de DONNEES du film ([FilmChunkNumbers]), releves une
// fois. La tranche est celle du contexte : ses lecteurs la parcourent, ils ne la modifient pas.
func (c *FilmContext) ChunkNumbers() []int {
	if c == nil {
		return nil
	}
	if !c.chunksLus {
		c.chunks, c.chunksLus = FilmChunkNumbers(c.film), true
	}
	return c.chunks
}

// ChunkAt rend les octets decompresses du chunk de NUMERO `num` et ses paquets ([FilmChunkAt]).
// Rien n'est memorise : la conversion des en-tetes est deja le prix plancher (cf. film_chunks.go).
func (c *FilmContext) ChunkAt(num int) ([]byte, []FilmPacket, bool) {
	if c == nil {
		return nil, nil, false
	}
	return FilmChunkAt(c.film, num)
}

// BipedSlots rend la bande de slots bipede du film ([bipedSlotBand] sur TOUS les chunks de
// donnees), relevee une fois.
//
// Bande VIDE quand le film n'a pas de chunk de donnees : les balayages testent `len(...) == 0`
// avant d'appeler, et rendent [ErrNoFilmChunk] — la garde ci-dessous ne fait que leur epargner
// l'indexation d'une tranche vide dans `bipedSlotBand`.
//
// CE N'EST PAS LA BANDE DE `DetectI0LayoutOf`, qui releve la SIENNE sur les six premiers chunks
// seulement : deux valeurs differentes, deux calculs, et c'est pourquoi la detection garde le
// sien (cf. i0_layout.go).
func (c *FilmContext) BipedSlots() map[uint32]bool {
	if c == nil {
		return nil
	}
	if !c.slotsLus {
		if nums := c.ChunkNumbers(); len(nums) > 0 {
			c.slots = bipedSlotBand(c.film, nums)
		}
		c.slotsLus = true
	}
	return c.slots
}

// I0Layout rend le decoupage d'i0 AUTO-DETECTE dans le film ([DetectI0LayoutOf]), detecte une
// fois. L'erreur rendue est celle de la detection, BRUTE : c'est l'appelant qui l'habille
// (« decoupage i0 illisible : %w »), comme il le faisait de l'appel direct.
func (c *FilmContext) I0Layout() (I0Layout, error) {
	if c == nil {
		return I0Layout{}, ErrNoFilmChunk
	}
	if !c.layLu {
		c.lay, _, c.layErr = DetectI0LayoutOf(c.film)
		c.layLu = true
	}
	return c.lay, c.layErr
}

// Registry rend le registre ECS du film (le chunk NUMERO 0), analyse une fois.
//
// [ErrNoRegistryChunk] quand le film ne porte pas son registre — bobine partielle, fixture : les
// lecteurs d'archetype le disent plutot que de rendre un registre vide, qui se lirait comme
// « archetype absent du build ».
//
// C'EST LE SEUL LECTEUR DE REGISTRE DU PAQUET : les six accesseurs d'archetype
// (`bipedArchetype`, `EquipmentArchetype`, `groundWeaponArchetype`, `managedPropertyArchetype`,
// `filmArchetype`, `objectiveArchetype`) en derivent tous, chacun ne gardant que son message
// d'archetype manquant. Un garde-rail ferme la porte a un second site d'analyse
// (`archlint/no_recomputed_film_context_test.go`).
func (c *FilmContext) Registry() (*Registry, error) {
	if c == nil {
		return nil, ErrNoRegistryChunk
	}
	if !c.regLu {
		c.regLu = true
		raw, ok := FilmRegistryChunk(c.film)
		if !ok {
			c.regErr = ErrNoRegistryChunk
		} else {
			c.reg, c.regErr = ParseRegistryChunk(raw)
		}
	}
	return c.reg, c.regErr
}

// archetype rend l'archetype `ti` du registre memorise. `ok` faux = le registre a ete lu mais ne
// porte pas cet archetype ; c'est a l'accesseur nomme de dire lequel manque, avec SON message —
// les six libelles historiques sont conserves mot pour mot.
func (c *FilmContext) archetype(ti int) (Archetype, *Registry, bool, error) {
	reg, err := c.Registry()
	if err != nil {
		return Archetype{}, nil, false, err
	}
	arch, ok := reg.Archetype(ti)
	return arch, reg, ok, nil
}
