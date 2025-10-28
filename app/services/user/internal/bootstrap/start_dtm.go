package bootstrap

import (
	"context"
	"net/http"
	"net/url"
	"time"

	handlers "NatsumeAI/app/services/user/internal/dtm"
	"NatsumeAI/app/services/user/internal/svc"
)

// StartDTMHTTP starts a lightweight HTTP server for DTM callbacks if configured.
func StartDTMHTTP(sc *svc.ServiceContext) func() {
    cfg := sc.Config.DtmConf
    if cfg.Server == "" || cfg.BusiURL == "" { return nil }
    addr := cfg.BusiListen
    if addr == "" {
        if u, err := url.Parse(cfg.BusiURL); err == nil && u.Host != "" { addr = u.Host }
        if addr == "" { addr = ":13001" }
    }
    mux := http.NewServeMux()
    handlers.Register(mux, sc)
    srv := &http.Server{Addr: addr, Handler: mux}
    go func() { _ = srv.ListenAndServe() }()
    return func() { ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel(); _ = srv.Shutdown(ctx) }
}

