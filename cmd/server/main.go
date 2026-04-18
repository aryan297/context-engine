package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/context-engine/internal/api"
	"github.com/context-engine/internal/api/handler"
	"github.com/context-engine/internal/config"
	"github.com/context-engine/internal/embedding"
	"github.com/context-engine/internal/parser"
	"github.com/context-engine/internal/service"
	neo4jstore "github.com/context-engine/internal/storage/neo4j"
	pgstore "github.com/context-engine/internal/storage/postgres"
	redisstore "github.com/context-engine/internal/storage/redis"
	"github.com/context-engine/pkg/cli"
)

func main() {
	cfg := config.Load()

	root := &cobra.Command{
		Use:   "context-engine",
		Short: "Context Engine — reduce LLM token usage via smart context retrieval",
	}

	// --- serve subcommand ---
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cfg)
		},
	}

	// --- ingest / query CLI subcommands ---
	serverURL, _ := root.PersistentFlags().GetString("server")
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%s", cfg.Server.Port)
	}

	ingestCmd := cli.NewIngestCmd(serverURL)
	queryCmd := cli.NewQueryCmd(serverURL)

	ingestCmd.Flags().String("project", "", "Project name (defaults to path)")
	queryCmd.Flags().String("project", "", "Project name to query against")

	root.AddCommand(serveCmd, ingestCmd, queryCmd)
	root.PersistentFlags().String("server", fmt.Sprintf("http://localhost:%s", cfg.Server.Port), "Context Engine server URL")

	if err := root.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runServer(cfg *config.Config) error {
	// Storage layer
	vectorStore, err := pgstore.NewVectorStore(cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}

	graphStore, err := neo4jstore.NewGraphStore(cfg.Neo4j.URI, cfg.Neo4j.Username, cfg.Neo4j.Password)
	if err != nil {
		return fmt.Errorf("neo4j: %w", err)
	}

	cache, err := redisstore.NewCache(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	// Core components
	emb := embedding.NewMockEmbedder()
	goParser := parser.NewGoParser()

	// Services
	ingestSvc := service.NewIngestService(goParser, graphStore, vectorStore, emb)
	querySvc := service.NewQueryService(graphStore, vectorStore, cache, emb)

	// HTTP layer
	ingestH := handler.NewIngestHandler(ingestSvc)
	queryH := handler.NewQueryHandler(querySvc)
	router := api.NewRouter(ingestH, queryH)

	addr := ":" + cfg.Server.Port
	fmt.Printf("context-engine listening on %s\n", addr)
	return router.Run(addr)
}
