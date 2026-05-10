// Package samba — gerenciamento declarativo de compartilhamentos Samba.
//
// O backend NÃO modifica /etc/samba/smb.conf direto. Em vez disso, mantém
// um arquivo de "estado desejado" em /var/lib/pinas/samba/desired-state.json
// que descreve quais shares e usuários devem existir. Um script no host
// (samba-sync.sh) observa esse arquivo via inotify (systemd .path unit) e
// aplica as mudanças no Samba real.
//
// Vantagens:
//   - Backend não precisa de privilégio root nem de comandos especiais
//   - Mudanças são idempotentes e auditáveis
//   - Em caso de falha do sync, fica claro o "drift" entre desejado e atual
package samba

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// State é o estado desejado do Samba, serializado em JSON e lido pelo host.
type State struct {
	Version int     `json:"version" yaml:"version"`
	Shares  []Share `json:"shares"  yaml:"shares"`
	Users   []User  `json:"users"   yaml:"users"`
}

// Share representa um compartilhamento Samba.
type Share struct {
	// Nome do share (entre colchetes em smb.conf). Letras, números, _, -.
	Name string `json:"name" yaml:"name"`

	// Path absoluto no host (não é o path dentro do container PiNAS).
	// Ex: /home/gustavo/Documents, /srv/pinas/data/shared
	Path string `json:"path" yaml:"path"`

	// Comentário descritivo.
	Comment string `json:"comment,omitempty" yaml:"comment,omitempty"`

	// Se true, share é navegável (aparece na lista de redes do Windows).
	Browseable bool `json:"browseable" yaml:"browseable"`

	// Se true, share é somente leitura.
	ReadOnly bool `json:"read_only" yaml:"read_only"`

	// Se true, aceita acesso anônimo (sem usuário/senha). NÃO recomendado.
	GuestOk bool `json:"guest_ok" yaml:"guest_ok"`

	// Lista de usuários do sistema (Linux) com acesso. Usuários do PiNAS
	// (banco SQLite) e do Samba são DIFERENTES — veja docs.
	ValidUsers []string `json:"valid_users,omitempty" yaml:"valid_users,omitempty"`

	// Máscara de criação de arquivos/dirs (default 0664/0775).
	CreateMask    string `json:"create_mask,omitempty"    yaml:"create_mask,omitempty"`
	DirectoryMask string `json:"directory_mask,omitempty" yaml:"directory_mask,omitempty"`

	// User Linux que será dono dos arquivos criados. Se vazio, usa quem conectou.
	ForceUser string `json:"force_user,omitempty" yaml:"force_user,omitempty"`
}

// User representa um usuário Samba (no banco interno do smbpasswd).
// IMPORTANTE: o usuário precisa existir como Linux user antes de ser
// adicionado ao Samba. O sync script trata isso (cria como --system se faltar).
type User struct {
	// Username — também é o nome do usuário Linux.
	Username string `json:"username" yaml:"username"`

	// Hash da senha — MD4 (formato smbpasswd). Gerado pelo backend ao criar/alterar.
	// Nunca exposto em response da API.
	NTHash string `json:"nt_hash,omitempty" yaml:"nt_hash,omitempty"`

	// Se true, usuário está desativado.
	Disabled bool `json:"disabled" yaml:"disabled"`
}

var (
	ErrShareNotFound = errors.New("samba: share não encontrado")
	ErrShareExists   = errors.New("samba: share com esse nome já existe")
	ErrUserNotFound  = errors.New("samba: usuário não encontrado")
	ErrUserExists    = errors.New("samba: usuário com esse nome já existe")
	ErrInvalidName   = errors.New("samba: nome inválido (letras, números, _ ou -; 1-32 chars)")
	ErrInvalidPath   = errors.New("samba: path inválido (absoluto, sem .. ou caracteres especiais)")
	ErrReservedName  = errors.New("samba: nome reservado (global, homes, printers, etc)")
)

var (
	nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
	// Nomes reservados pelo Samba que não podem ser nomes de share.
	reservedNames = map[string]bool{
		"global": true, "homes": true, "printers": true,
		"print$": true, "ipc$": true,
	}
)

// ValidateShareName valida o nome de um share.
func ValidateShareName(name string) error {
	if !nameRegex.MatchString(name) {
		return ErrInvalidName
	}
	if reservedNames[strings.ToLower(name)] {
		return ErrReservedName
	}
	return nil
}

// ValidateUsername valida o nome de um usuário.
func ValidateUsername(name string) error {
	if !nameRegex.MatchString(name) {
		return ErrInvalidName
	}
	// Não permitir usuários de sistema sensíveis.
	bad := map[string]bool{"root": true, "daemon": true, "bin": true, "sys": true,
		"www-data": true, "nobody": true}
	if bad[strings.ToLower(name)] {
		return fmt.Errorf("samba: nome de usuário reservado: %s", name)
	}
	return nil
}

// ValidateSharePath valida o path de um share.
func ValidateSharePath(p string) error {
	if !strings.HasPrefix(p, "/") {
		return ErrInvalidPath
	}
	if strings.Contains(p, "..") {
		return ErrInvalidPath
	}
	// Caracteres permitidos: letras, números, /, _, -, ., espaço
	if regexp.MustCompile(`[^a-zA-Z0-9/_\-. ]`).MatchString(p) {
		return ErrInvalidPath
	}
	return nil
}

// Service implementa as operações do módulo. Stateful em memória; cada
// mudança regrava o desired-state.json via Store.
type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetState() (*State, error) {
	return s.store.Load()
}

// ListShares devolve todos os shares.
func (s *Service) ListShares() ([]Share, error) {
	st, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return st.Shares, nil
}

// GetShare busca um share por nome.
func (s *Service) GetShare(name string) (*Share, error) {
	st, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	for i := range st.Shares {
		if strings.EqualFold(st.Shares[i].Name, name) {
			return &st.Shares[i], nil
		}
	}
	return nil, ErrShareNotFound
}

// CreateShare adiciona um novo share.
func (s *Service) CreateShare(sh Share) error {
	if err := ValidateShareName(sh.Name); err != nil {
		return err
	}
	if err := ValidateSharePath(sh.Path); err != nil {
		return err
	}
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for _, existing := range st.Shares {
		if strings.EqualFold(existing.Name, sh.Name) {
			return ErrShareExists
		}
	}
	// Defaults sãos.
	if sh.CreateMask == "" {
		sh.CreateMask = "0664"
	}
	if sh.DirectoryMask == "" {
		sh.DirectoryMask = "0775"
	}
	st.Shares = append(st.Shares, sh)
	return s.store.Save(st)
}

// UpdateShare substitui um share existente.
func (s *Service) UpdateShare(name string, sh Share) error {
	if err := ValidateShareName(sh.Name); err != nil {
		return err
	}
	if err := ValidateSharePath(sh.Path); err != nil {
		return err
	}
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range st.Shares {
		if strings.EqualFold(st.Shares[i].Name, name) {
			// Preserva o nome original se mudou apenas case.
			sh.Name = name
			st.Shares[i] = sh
			return s.store.Save(st)
		}
	}
	return ErrShareNotFound
}

// DeleteShare remove um share.
func (s *Service) DeleteShare(name string) error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for i, sh := range st.Shares {
		if strings.EqualFold(sh.Name, name) {
			st.Shares = append(st.Shares[:i], st.Shares[i+1:]...)
			return s.store.Save(st)
		}
	}
	return ErrShareNotFound
}

// ListUsers devolve todos os usuários (sem expor hashes).
func (s *Service) ListUsers() ([]User, error) {
	st, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	out := make([]User, len(st.Users))
	for i, u := range st.Users {
		out[i] = User{Username: u.Username, Disabled: u.Disabled}
	}
	return out, nil
}

// CreateUser adiciona um usuário Samba.
// O hash NT é gerado pelo sync script no host (que tem acesso ao smbpasswd).
// Aqui só guardamos a senha temporariamente em campo separado pra ser processada.
type CreateUserInput struct {
	Username string
	Password string
}

func (s *Service) CreateUser(in CreateUserInput) error {
	if err := ValidateUsername(in.Username); err != nil {
		return err
	}
	if len(in.Password) < 6 {
		return errors.New("samba: senha mínima 6 caracteres")
	}
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for _, u := range st.Users {
		if strings.EqualFold(u.Username, in.Username) {
			return ErrUserExists
		}
	}
	// O hash NT real será computado pelo script samba-sync.sh.
	// Aqui guardamos a senha em texto plano TEMPORARIAMENTE no estado pendente,
	// num campo SEPARADO que é apagado depois do sync. Pra não vazar pra UI,
	// nunca devolvemos esse campo em GET.
	st.Users = append(st.Users, User{
		Username: in.Username,
		NTHash:   "PENDING:" + in.Password, // marcador pro sync script
		Disabled: false,
	})
	return s.store.Save(st)
}

// SetUserPassword atualiza a senha de um usuário.
func (s *Service) SetUserPassword(username, password string) error {
	if len(password) < 6 {
		return errors.New("samba: senha mínima 6 caracteres")
	}
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range st.Users {
		if strings.EqualFold(st.Users[i].Username, username) {
			st.Users[i].NTHash = "PENDING:" + password
			return s.store.Save(st)
		}
	}
	return ErrUserNotFound
}

// SetUserDisabled habilita/desabilita um usuário.
func (s *Service) SetUserDisabled(username string, disabled bool) error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for i := range st.Users {
		if strings.EqualFold(st.Users[i].Username, username) {
			st.Users[i].Disabled = disabled
			return s.store.Save(st)
		}
	}
	return ErrUserNotFound
}

// DeleteUser remove um usuário.
func (s *Service) DeleteUser(username string) error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	for i, u := range st.Users {
		if strings.EqualFold(u.Username, username) {
			st.Users = append(st.Users[:i], st.Users[i+1:]...)
			return s.store.Save(st)
		}
	}
	return ErrUserNotFound
}
