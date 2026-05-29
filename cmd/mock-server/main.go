package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/amoylab/unla/cmd/mock-server/backend"
	"github.com/amoylab/unla/pkg/version"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var (
	addr     string
	sseAddr  string
	nacosReg nacosRegistrationOptions
	logger   *zap.Logger
)

func init() {
	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVarP(&addr, "addr", "a", ":5236", "Address to listen on")
	rootCmd.PersistentFlags().StringVarP(&sseAddr, "sse-addr", "s", ":5237", "Address to listen on for SSE")
	rootCmd.PersistentFlags().BoolVar(&nacosReg.Enabled, "register-nacos", false, "Register the SSE MCP server instance to Nacos")
	rootCmd.PersistentFlags().StringVar(&nacosReg.NacosHost, "nacos-host", "127.0.0.1", "Nacos server host")
	rootCmd.PersistentFlags().Uint64Var(&nacosReg.NacosPort, "nacos-port", 8848, "Nacos server port")
	rootCmd.PersistentFlags().StringVar(&nacosReg.NacosScheme, "nacos-scheme", "http", "Nacos server scheme")
	rootCmd.PersistentFlags().StringVar(&nacosReg.NacosNamespace, "nacos-namespace", "", "Nacos namespace ID")
	rootCmd.PersistentFlags().StringVar(&nacosReg.NacosGroup, "nacos-group", "DEFAULT_GROUP", "Nacos service group")
	rootCmd.PersistentFlags().StringVar(&nacosReg.NacosCluster, "nacos-cluster", "DEFAULT", "Nacos cluster")
	rootCmd.PersistentFlags().StringVar(&nacosReg.ServiceName, "nacos-service-name", "mock-user-sse", "Nacos service name for this MCP server")
	rootCmd.PersistentFlags().StringVar(&nacosReg.RegisterIP, "mcp-register-host", "127.0.0.1", "MCP server IP registered to Nacos")
	rootCmd.PersistentFlags().Uint64Var(&nacosReg.RegisterPort, "mcp-register-port", 0, "MCP server port registered to Nacos; defaults to --sse-addr port")
	rootCmd.PersistentFlags().StringVar(&nacosReg.MCPScheme, "mcp-scheme", "http", "MCP server access scheme metadata")
	rootCmd.PersistentFlags().StringVar(&nacosReg.MCPProtocol, "mcp-protocol", "sse", "MCP server protocol metadata")
	rootCmd.PersistentFlags().StringVar(&nacosReg.MCPEndpoint, "mcp-endpoint", "/sse", "MCP server endpoint metadata")
	rootCmd.PersistentFlags().StringVar(&nacosReg.MCPHost, "mcp-host", "localhost", "Optional MCP access host metadata override")
}

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of mock-server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("mock-server version %s\n", version.Get())
		},
	}

	rootCmd = &cobra.Command{
		Use:   "mock-server",
		Short: "Mock Backend Server",
		Long:  `Mock Backend Server provide mock servers for testing`,
		Run: func(cmd *cobra.Command, args []string) {
			StartMockServer(addr)
		},
	}
)

func StartMockServer(addr string) {
	// Create a context that will be canceled on OS signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if nacosReg.Enabled {
		if err := registerSSEMCPServer(ctx, nacosReg, sseAddr); err != nil {
			logger.Fatal("Failed to register mock MCP server to Nacos", zap.Error(err))
		}
		defer deregisterSSEMCPServer(nacosReg, sseAddr)
	}

	// Create error channel to collect errors from all servers
	errChan := make(chan error, 3)

	// Start all servers with context
	go startHTTPServer(ctx, addr, errChan)
	go startStdioServer(ctx, errChan)
	go startSSEServer(ctx, addr, errChan)

	// Wait for either context cancellation or error
	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, stopping all servers...")
	case err := <-errChan:
		logger.Error("Server error occurred", zap.Error(err))
		stop() // Cancel context to trigger shutdown
	}

	// Wait for all servers to shutdown
	<-ctx.Done()
	logger.Info("All servers stopped")
}

func startHTTPServer(_ context.Context, addr string, errChan chan<- error) {
	httpServer := backend.NewHTTPServer()
	if err := httpServer.Start(addr); err != nil {
		errChan <- fmt.Errorf("HTTP server error: %w", err)
	}
}

func startStdioServer(_ context.Context, errChan chan<- error) {
	mcpServer := backend.NewMCPServer()

	logger.Info("Starting MCP server on stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		errChan <- fmt.Errorf("stdio server error: %w", err)
	}
}

func startSSEServer(_ context.Context, addr string, errChan chan<- error) {
	mcpServer := backend.NewMCPServer()

	sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost%s", sseAddr)))
	logger.Info("Starting SSE server", zap.String("addr", fmt.Sprintf("http://localhost%s/sse", sseAddr)))
	if err := sseServer.Start(sseAddr); err != nil {
		errChan <- fmt.Errorf("SSE server error: %w", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
