package config

import "os"

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Neo4j    Neo4jConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Port string
}

type PostgresConfig struct {
	DSN string
}

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
}

type RedisConfig struct {
	Addr     string
	Password string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/contextdb?sslmode=disable"),
		},
		Neo4j: Neo4jConfig{
			URI:      getEnv("NEO4J_URI", "neo4j://localhost:7687"),
			Username: getEnv("NEO4J_USERNAME", "neo4j"),
			Password: getEnv("NEO4J_PASSWORD", "password"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
