// cli.go — subcomandos administrativos do binário pinas.
//
// Uso (dentro do container):
//   pinas                                           # sobe o HTTP server (default)
//   pinas reset-admin                               # gera senha aleatória nova
//   pinas reset-admin --password=NovaSenha123       # define senha específica
//   pinas reset-admin --username=admin --password=X # se o admin tem nome diferente
//
// Exemplo prático (no host):
//   sudo docker compose exec pinas-api pinas reset-admin
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/pinas/pinas/internal/config"
	"github.com/pinas/pinas/internal/database"
	"github.com/pinas/pinas/internal/users"
	"github.com/pinas/pinas/pkg/argon2id"
)

// dispatchCLI processa os args. Devolve true se foi um subcomando
// (e o programa deve sair com o code retornado). False se main() deve continuar
// com o servidor HTTP normal.
func dispatchCLI() (handled bool, exitCode int) {
	if len(os.Args) < 2 {
		return false, 0
	}

	switch os.Args[1] {
	case "reset-admin":
		return true, cmdResetAdmin(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("pinas v0.2.0")
		return true, 0
	case "help", "-h", "--help":
		printHelp()
		return true, 0
	}
	return false, 0
}

func printHelp() {
	fmt.Print(`PiNAS — backend

Comandos:
  pinas                         Sobe o servidor HTTP (default)
  pinas reset-admin [flags]     Reseta a senha do admin
  pinas version                 Mostra a versão

Flags de reset-admin:
  --username=NAME    Nome do admin (default: lê de PINAS_ADMIN_USERNAME ou "admin")
  --password=SENHA   Define senha específica (default: gera aleatória)

Exemplos:
  sudo docker compose exec pinas-api pinas reset-admin
  sudo docker compose exec pinas-api pinas reset-admin --password=MinhaSenh@2026
  sudo docker compose exec pinas-api pinas version
`)
}

func cmdResetAdmin(args []string) int {
	fs := flag.NewFlagSet("reset-admin", flag.ContinueOnError)
	username := fs.String("username", "", "Nome do admin (default: PINAS_ADMIN_USERNAME ou 'admin')")
	password := fs.String("password", "", "Senha específica (default: gera aleatória de 24 caracteres)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao carregar config:", err)
		return 1
	}

	if *username == "" {
		if cfg.AdminUsername != "" {
			*username = cfg.AdminUsername
		} else {
			*username = "admin"
		}
	}

	// Gera senha se não foi passada.
	generated := false
	if *password == "" {
		p, err := generatePassword(24)
		if err != nil {
			fmt.Fprintln(os.Stderr, "erro ao gerar senha:", err)
			return 1
		}
		*password = p
		generated = true
	}

	if len(*password) < 8 {
		fmt.Fprintln(os.Stderr, "erro: senha mínima 8 caracteres")
		return 2
	}

	// Conecta no banco direto (sem subir nada mais).
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao abrir banco:", err)
		return 1
	}
	defer db.Close()

	repo := users.NewRepository(db)
	ctx := context.Background()

	user, err := repo.GetByUsername(ctx, *username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro: usuário '%s' não encontrado\n", *username)
		fmt.Fprintln(os.Stderr, "use --username=NOME se o admin tem nome diferente")
		return 1
	}

	if user.Role != "admin" {
		fmt.Fprintf(os.Stderr, "erro: usuário '%s' existe mas não é admin\n", *username)
		return 1
	}

	// Hash da nova senha.
	hash, err := argon2id.Hash(*password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro ao hashear senha:", err)
		return 1
	}

	// Atualiza no banco e re-habilita se estava desabilitado.
	if err := repo.UpdatePassword(ctx, user.ID, hash); err != nil {
		fmt.Fprintln(os.Stderr, "erro ao atualizar senha:", err)
		return 1
	}
	if user.Disabled {
		user.Disabled = false
		if err := repo.Update(ctx, user); err != nil {
			fmt.Fprintln(os.Stderr, "aviso: senha resetada mas falhou re-habilitar:", err)
		}
	}

	fmt.Println()
	fmt.Println("✓ Senha do admin resetada com sucesso")
	fmt.Printf("  usuário: %s\n", *username)
	if generated {
		fmt.Printf("  senha:   %s\n", *password)
		fmt.Println()
		fmt.Println("  ⚠ Anote essa senha agora — ela não será mostrada novamente.")
	} else {
		fmt.Println("  senha:   (a que você definiu)")
	}
	fmt.Println()
	return 0
}

// generatePassword gera senha aleatória URL-safe de length caracteres.
func generatePassword(length int) (string, error) {
	// base64-url usa ~4 chars pra cada 3 bytes.
	bytes := make([]byte, (length*3)/4+1)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(bytes)
	if len(s) > length {
		s = s[:length]
	}
	return s, nil
}
