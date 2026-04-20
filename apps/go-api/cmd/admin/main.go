// cmd/admin — CLI d'administration d'urgence pour LevelUp.
//
// Permet de créer un admin ou réinitialiser un mot de passe sans l'API web.
//
// Usage :
//
//	./admin create-admin --username admin --password secret123
//	./admin reset-password --username admin --password newpass123
//	./admin list-users
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/platform/userstore"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]

	// Parse flags after subcommand
	fs := flag.NewFlagSet(subcmd, flag.ExitOnError)
	username := fs.String("username", "", "Nom d'utilisateur")
	password := fs.String("password", "", "Mot de passe")
	dataDir := fs.String("data-dir", defaultDataDir(), "Répertoire data/auth")
	_ = fs.Parse(os.Args[2:])

	usersPath := filepath.Join(*dataDir, "users.json")
	store := userstore.NewStore(usersPath)

	switch subcmd {
	case "create-admin":
		createAdmin(store, *username, *password)
	case "reset-password":
		resetPassword(store, *username, *password)
	case "list-users":
		listUsers(store)
	default:
		fmt.Fprintf(os.Stderr, "Commande inconnue: %s\n", subcmd)
		printUsage()
		os.Exit(1)
	}
}

func createAdmin(store *userstore.Store, username, password string) {
	if username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "Erreur: --username et --password requis")
		os.Exit(1)
	}
	_, err := store.Create(username, password, "admin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur création admin: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Admin '%s' créé avec succès.\n", username)
}

func resetPassword(store *userstore.Store, username, password string) {
	if username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "Erreur: --username et --password requis")
		os.Exit(1)
	}
	if err := store.ResetPassword(username, password); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur réinitialisation MDP: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Mot de passe de '%s' réinitialisé.\n", username)
}

func listUsers(store *userstore.Store) {
	users, err := store.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur lecture users: %v\n", err)
		os.Exit(1)
	}
	if len(users) == 0 {
		fmt.Println("Aucun utilisateur enregistré.")
		return
	}
	fmt.Printf("%-20s %-8s %-20s %s\n", "USERNAME", "ROLE", "GAMERTAG", "CREATED")
	for _, u := range users {
		gt := "-"
		if u.Gamertag != "" {
			gt = u.Gamertag
		}
		fmt.Printf("%-20s %-8s %-20s %s\n", u.Username, u.Role, gt, u.CreatedAt)
	}
}

func defaultDataDir() string {
	if d := os.Getenv("LEVELUP_AUTH_DIR"); d != "" {
		return d
	}
	return filepath.Join("data", "auth")
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: admin <command> [options]

Commands:
  create-admin    Créer un utilisateur admin
  reset-password  Réinitialiser un mot de passe
  list-users      Lister tous les utilisateurs

Options:
  --username    Nom d'utilisateur
  --password    Mot de passe
  --data-dir    Répertoire data/auth (défaut: data/auth)`)
}
