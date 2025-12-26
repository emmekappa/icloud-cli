package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Account struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	IsDefault bool   `json:"is_default"`
	Alias     string `json:"alias"`
}

func (a *Account) GetAlias() string {
	if a.Alias != "" {
		return a.Alias
	}
	if idx := strings.Index(a.Email, "@"); idx > 0 {
		return a.Email[:idx]
	}
	return a.Email
}

func (a *Account) Matches(identifier string) bool {
	return a.Name == identifier ||
		a.Email == identifier ||
		a.Alias == identifier ||
		a.GetAlias() == identifier
}

type Config struct {
	Accounts []Account `json:"accounts"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "icloud-cli", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Accounts: []Account{}}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) AddAccount(account Account) error {
	for _, a := range c.Accounts {
		if a.Email == account.Email {
			return fmt.Errorf("account with email %s already exists", account.Email)
		}
	}

	if account.Alias == "" {
		account.Alias = account.GetAlias()
	}

	if len(c.Accounts) == 0 {
		account.IsDefault = true
	}

	c.Accounts = append(c.Accounts, account)
	return c.Save()
}

func (c *Config) GetDefaultAccount() (*Account, error) {
	for _, a := range c.Accounts {
		if a.IsDefault {
			return &a, nil
		}
	}
	return nil, errors.New("no default account set")
}

func (c *Config) GetAccount(identifier string) (*Account, error) {
	for _, a := range c.Accounts {
		if a.Matches(identifier) {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("account %s not found", identifier)
}

func (c *Config) GetAccountOrDefault(identifier string) (*Account, error) {
	if identifier != "" {
		return c.GetAccount(identifier)
	}
	return c.GetDefaultAccount()
}

func (c *Config) SetDefault(identifier string) error {
	found := false
	for i := range c.Accounts {
		if c.Accounts[i].Matches(identifier) {
			c.Accounts[i].IsDefault = true
			found = true
		} else {
			c.Accounts[i].IsDefault = false
		}
	}

	if !found {
		return fmt.Errorf("account %s not found", identifier)
	}

	return c.Save()
}

func (c *Config) RemoveAccount(identifier string) error {
	for i, a := range c.Accounts {
		if a.Matches(identifier) {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			if a.IsDefault && len(c.Accounts) > 0 {
				c.Accounts[0].IsDefault = true
			}
			return c.Save()
		}
	}
	return fmt.Errorf("account %s not found", identifier)
}
