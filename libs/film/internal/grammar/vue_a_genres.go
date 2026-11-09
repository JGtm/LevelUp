package grammar

// vue_a_genres.go — LES 123 GENRES DE MESSAGE DE LA VUE A, TELS QUE LE JEU LES ENREGISTRE (lot LN
// de la campagne de grammaire).
//
// # LA TABLE EST STATIQUE, ET ELLE SE LIT DANS L EXECUTABLE
//
// `FUN_14054d014` pose `DAT_144e61d88 = base + 0x8aed0` puis appelle `FUN_140e453b4`, qui ecrit
// les 123 descripteurs a `DAT_144e61d88 + 0x210 + genre * 8` par des affectations d ADRESSES
// CONSTANTES, et rend le compte `0x7b` (rang `+0x208`). Le lecteur d un message (`FUN_14080a9d4`)
// prend son descripteur a la meme adresse : `*(obj[0x18] + 0x210 + genre * 8)`. Chaque descripteur
// est un objet de 8 octets dont le seul champ est sa vtable :
//
//	vtable + 0x08  nom de debogage (`FUN_14080ada0` le rend)
//	vtable + 0x10  taille de la structure du message : 0 -> AUCUNE charge lue (`FUN_14080a9d4`)
//	vtable + 0x58  domaine de la reference d indice i (`FUN_1406d3140`, categorie `param_3`)
//	vtable + 0x60  ecrivain de la charge (`FUN_1424d80bc`, pre-serialisation du film)
//	vtable + 0x68  lecteur de la charge, appele avec `param_5 = 1`
//
// La table complete (nom, adresse du descripteur, adresse du lecteur) est celle du test
// (`vue_a_genres_test.go`), qui la confronte a [tableDesGenresVueA]. Les charges portees sont
// rendues par [chargeDuGenre] ; un genre sans charge portee arrete la lecture de la vue A
// ([lireLaVueA]).

// tableDesGenresVueA porte, genre par genre dans l ordre de `FUN_140e453b4`, une entree de
// [largeurEntreeDeGenre] caracteres : les domaines des trois references (rendus de `vtable + 0x58`
// pour i = 0, 1, 2 : cascades `85 D2 74 ..` et leurs blocs froids `83 EA 01 ..`, lues octet par
// octet), puis `v` quand `vtable + 0x10` rend 0 (`33 C0 C3`, lecteur `FUN_1408d8220` qui rend 1 sans
// lire) et `.` sinon.
const tableDesGenresVueA = "" +
	"117.187.187.087v087v587.587.187.237.287." + // 0-9
	"887.287.887.487.227.887.887.667.687.087." + // 10-19
	"117.487.117.087v027v027v027v007.087.687." + // 20-29
	"487.187.187.347v887.187.187.187.287.287." + // 30-39
	"207.387.287.207.207.287.487.487.487.327v" + // 40-49
	"447.287.207.117.117v687.517.207v187.087v" + // 50-59
	"087.087.687.287.687.387.687.687.687.087." + // 60-69
	"287.287.287.287.287.187.187.687.487.407." + // 70-79
	"887.087.087.087.087.887.187.687.687.687." + // 80-89
	"687.687.687v207.687.687.687.687.117.887." + // 90-99
	"187.687.287.007v007.007.007.087.687.007." + // 100-109
	"187.687.687.687.687.107.107.207.187.207." + // 110-119
	"887.887.887." // 120-122

// largeurEntreeDeGenre : trois domaines et la marque de vacuite.
const largeurEntreeDeGenre = 4

// marqueGenreVide est la marque d un genre dont la taille de structure est nulle.
const marqueGenreVide = 'v'

// descripteurDuGenre rend les domaines des trois references d un genre (0 <= genre < GenresVueA) et
// dit si le genre est vide.
func descripteurDuGenre(genre int) (domaines [3]int, vide bool) {
	e := tableDesGenresVueA[genre*largeurEntreeDeGenre : (genre+1)*largeurEntreeDeGenre]
	for i := range domaines {
		domaines[i] = int(e[i] - '0')
	}
	return domaines, e[3] == marqueGenreVide
}
