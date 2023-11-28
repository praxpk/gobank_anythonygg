package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccountByID(int) (*Account, error)
}

type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
	DBName   string `yaml:"dbName,omitempty"`
}

type PostgresStore struct {
	db *sql.DB
}

func getPostgresInfo() (*Config, error) {
	// get config details from yaml file
	f, err := os.ReadFile("config.yml")

	if err != nil {
		return &Config{}, fmt.Errorf("unable to open config yaml file for postgres server connection: %s", err)
	}

	var cfg Config
	err = yaml.Unmarshal(f, &cfg)
	if err != nil {
		return &Config{}, fmt.Errorf("unable to decode config yaml file for postgres server connection: %s", err)
	}

	return &cfg, nil
}

func NewPostgresStore() (*PostgresStore, error) {
	// get db server config details
	postgresConfig, err := getPostgresInfo()
	if err != nil {
		return nil, fmt.Errorf("error parsing config yaml file:%v", err)
	}

	// connect to db server
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		postgresConfig.Host,
		postgresConfig.Port,
		postgresConfig.User,
		postgresConfig.Password,
		postgresConfig.DBName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating postgres db: %v\n", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging postgres db: %v\n", err)
	}
	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) CreateAccount(*Account) error {
	return nil
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	return &Account{}, nil
}
func (s *PostgresStore) UpdateAccount(*Account) error {
	return nil
}
func (s *PostgresStore) DeleteAccount(id int) error {
	return nil
}

func (s *PostgresStore) CreateAccountTable() error {
	return nil
}
