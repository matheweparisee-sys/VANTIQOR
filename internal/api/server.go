// internal/api/server.go
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/agent"
	"github.com/clawtrade/clawtrade/internal/backtest"
	"github.com/clawtrade/clawtrade/internal/config"
	"github.com/clawtrade/clawtrade/internal/engine"
	"github.com/clawtrade/clawtrade/internal/mcp"
	"github.com/clawtrade/clawtrade/internal/memory"
	"github.com/clawtrade/clawtrade/internal/risk"
	"github.com/clawtrade/clawtrade/internal/security"
	"github.com/clawtrade/clawtrade/internal/subagent"
)

type Server struct {
	db       *sql.DB
	router   chi.Router
	cfg      *config.Config
	bus      *engine.EventBus
	hub      *Hub
	memory   *memory.Store
	audit    *security.AuditLog
	adapters map[string]adapter.TradingAdapter
	agentMgr *subagent.AgentManager
	ag       *agent.Agent
}

// SetAgentManager sets the sub-agent manager for agent status endpoints.
func (s *Server) SetAgentManager(mgr *subagent.AgentManager) {
	s.agentMgr = mgr
}

// SetAlertService connects the alert service to the agent's tools.
func (s *Server) SetAlertService(svc interface{}) {
	if as, ok := svc.(agent.AlertService); ok {
		s.ag.SetAlertService(as)
	}
}

func NewServer(cfg *config.Config, bus *engine.EventBus, mem *memory.Store, audit *security.AuditLog, adapters map[string]adapter.TradingAdapter, riskEngine *risk.Engine, databases ...*sql.DB) *Server {
	var db *sql.DB
	if len(databases) > 0 {
		db = databases[0]
	}
	s := &Server{
		db:       db,
		cfg:      cfg,
		bus:      bus,
		memory:   mem,
		audit:    audit,
		adapters: adapters,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.sameOriginWrites)
	r.Use(middleware.Timeout(120 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Admin-Key", "X-Support-Token"},
	}))

	ag := agent.New(cfg, adapters, riskEngine, mem, bus, db)

	// Connect external MCP servers to agent
	if len(cfg.MCP.Servers) > 0 {
		mcpManager := mcp.NewClientManager()
		for _, srv := range cfg.MCP.Servers {
			if !srv.Enabled {
				continue
			}
			client := mcp.NewClient(srv.Name, srv.Command, srv.Args, srv.Env)
			if err := mcpManager.Add(client); err != nil {
				fmt.Printf("MCP server %s: failed to connect: %v\n", srv.Name, err)
				continue
			}
			fmt.Printf("MCP server %s: connected\n", srv.Name)
		}
		if mcpManager.ServerCount() > 0 {
			ag.SetMCPBridge(agent.NewMCPBridge(mcpManager))
		}
	}

	if err := s.initAuth(); err != nil {
		panic(fmt.Errorf("initialize investor authentication: %w", err))
	}
	s.ag = ag

	// WebSocket hub for real-time data
	hub := NewHub()
	go hub.Run()
	hub.SubscribeToEvents(bus, []string{
		"market.*",
		"trade.*",
		"risk.*",
		"system.*",
		"price.*",     // live price updates
		"agent.*",     // sub-agent insights
		"portfolio.*", // portfolio changes
		"backtest.*",  // backtest progress & results
		"alert.*",     // alert triggers
	})
	s.hub = hub

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/system/health", s.handleHealth)
		r.Post("/auth/signup", s.handleSignup)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.requireInvestor(s.handleMe))
		r.Get("/my/deposit-applications", s.requireInvestor(s.handleMyDeposits))
		r.Get("/system/version", s.handleVersion)
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Post("/chat", s.requireInvestor(llm.HandleChat))

		// Customer deposit applications and support conversations.
		r.Post("/deposit-applications", s.requireInvestor(s.handleCreateDepositApplication))
		r.Post("/support/threads", s.handleCreateSupportThread)
		r.Get("/support/threads/{id}/messages", s.handleGetCustomerMessages)
		r.Post("/support/threads/{id}/messages", s.handlePostCustomerMessage)

		// Admin endpoints require ADMIN_API_KEY in the request header.
		r.Get("/admin/support/threads", s.requireAdmin(s.handleAdminListThreads))
		r.Get("/admin/support/threads/{id}/messages", s.requireAdmin(s.handleAdminGetMessages))
		r.Post("/admin/support/threads/{id}/messages", s.requireAdmin(s.handleAdminPostMessage))
		r.Patch("/admin/support/threads/{id}", s.requireAdmin(s.handleAdminUpdateThread))
		r.Get("/admin/deposit-applications", s.requireAdmin(s.handleAdminListDeposits))
		r.Patch("/admin/deposit-applications/{id}", s.requireAdmin(s.handleAdminUpdateDeposit))

		// Portfolio (aggregated across all exchanges)
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Get("/portfolio", s.requireInvestor(s.handleGetPortfolio))

		// Exchange data
		r.Get("/price", s.handleGetPrice)
		r.Get("/candles", s.handleGetCandles)
		r.Get("/orderbook", s.handleGetOrderBook)
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Get("/balances", s.requireInvestor(s.handleGetBalances))
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Get("/positions", s.requireInvestor(s.handleGetPositions))
		r.Get("/exchanges", s.handleListExchanges)

		// Backtesting
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Post("/backtest", s.requireInvestor(s.handleBacktest))

		// Sub-agent management
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Get("/agents", s.requireInvestor(s.handleListAgents))
		// Shared trading data is disabled for investor accounts until tenant isolation: r.Get("/agents/events", s.requireInvestor(s.handleAgentEvents))
	})

	// WebSocket endpoint
	// Disabled until per-investor WebSocket authorization and event filtering exist.
	// r.Get("/ws", hub.HandleWebSocket)

	// Serve the VANTIQOR web dashboard when the frontend has been built.
	// API and WebSocket routes above take precedence over this static fallback.
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "web/dist/index.html") })
	r.Handle("/*", http.FileServer(http.Dir("web/dist")))

	s.router = r
	return s
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) Start(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("API server listening on %s\n", addr)
	return srv.ListenAndServe()
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if s.agentMgr == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.agentMgr.Statuses())
}

func (s *Server) handleBacktest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbol    string  `json:"symbol"`
		Timeframe string  `json:"timeframe"`
		Days      int     `json:"days"`
		Strategy  string  `json:"strategy"`
		Capital   float64 `json:"capital"`
		Exchange  string  `json:"exchange"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	// Defaults
	if req.Symbol == "" {
		req.Symbol = "BTC/USDT"
	}
	if req.Timeframe == "" {
		req.Timeframe = "1d"
	}
	if req.Days <= 0 {
		req.Days = 90
	}
	if req.Strategy == "" {
		req.Strategy = "momentum"
	}
	if req.Capital <= 0 {
		req.Capital = 10000
	}
	if req.Exchange == "" {
		req.Exchange = "binance"
	}

	adp, ok := s.adapters[req.Exchange]
	if !ok {
		http.Error(w, fmt.Sprintf(`{"error":"unknown exchange %q"}`, req.Exchange), http.StatusBadRequest)
		return
	}

	loader := backtest.NewDataLoader(nil, adp)
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -req.Days)
	candles, err := loader.LoadCandles(r.Context(), req.Symbol, req.Timeframe, from, now)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"load candles: %s"}`, err), http.StatusInternalServerError)
		return
	}

	strat := backtest.GetBuiltinStrategy(req.Strategy)
	if strat == nil {
		http.Error(w, fmt.Sprintf(`{"error":"unknown strategy %q"}`, req.Strategy), http.StatusBadRequest)
		return
	}

	eng := &backtest.Engine{Bus: s.bus}
	result, err := eng.Run(r.Context(), backtest.BacktestConfig{
		Symbol:    req.Symbol,
		Timeframe: req.Timeframe,
		From:      from,
		To:        now,
		Capital:   req.Capital,
	}, candles, strat)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"backtest run: %s"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleAgentEvents(w http.ResponseWriter, r *http.Request) {
	if s.agentMgr == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.agentMgr.Bus().RecentEvents(limit))
}
