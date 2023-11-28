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
	GetAccounts() ([]*Account, error)
}

type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbName"`
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

func (s *PostgresStore) CreateAccount(acc *Account) error {
	query := "INSERT INTO account (first_name, last_name, number, balance, created_at) VALUES ($1, $2, $3, $4, $5)"
	result, err := s.db.Query(query,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.Balance,
		acc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("could not create account for %s %s: %v", acc.FirstName, acc.LastName, err)
	}
	fmt.Printf("account creation => %v\n", result)
	return nil
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	var acc Account
	query := "SELECT * FROM account WHERE id=$1"

	err := s.db.QueryRow(query, id).Scan(
		&acc.ID,
		&acc.FirstName,
		&acc.LastName,
		&acc.Number,
		&acc.Balance,
		&acc.CreatedAt,
	)
	if err != nil {
		// TODO if record not found send different error to the generic one below
		return nil, fmt.Errorf("could not get account with id %d: %v", id, err)
	}
	return &acc, nil
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

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	query := "SELECT * FROM account"
	rows, err := s.db.Query(query)
	if err != nil {
		return []*Account{}, fmt.Errorf("could not get accounts from db: %v", err)
	}
	var accounts []*Account
	for rows.Next() {
		acc := new(Account)
		err := rows.Scan(
			&acc.ID,
			&acc.FirstName,
			&acc.LastName,
			&acc.Number,
			&acc.Balance,
			&acc.CreatedAt)
		if err != nil {
			return []*Account{}, fmt.Errorf("could not parse response from db: %v", err)
		}
		accounts = append(accounts, acc)
	}
	return accounts, nil
}

func (s *PostgresStore) Init() error {
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		number serial,
		balance numeric,
		created_at timestamp

	)`
	_, err := s.db.Exec(query)
	return err
}
