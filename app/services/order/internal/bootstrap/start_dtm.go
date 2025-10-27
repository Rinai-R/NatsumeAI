package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	handlers "NatsumeAI/app/services/order/internal/dtm"
	"NatsumeAI/app/services/order/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

func StartDTMHTTP(sc *svc.ServiceContext) func() {
    cfg := sc.Config.DtmConf
    if cfg.Server == "" || cfg.BusiURL == "" {
        return nil
    }
    addr := cfg.BusiListen
    if addr == "" {
        if u, err := url.Parse(cfg.BusiURL); err == nil && u.Host != "" {
            addr = u.Host
        }
        if addr == "" {
            addr = ":13005"
            logx.Info("DtmConf.BusiListen not set; defaulting to ", addr)
        }
    }
    mux := http.NewServeMux()
    handlers.Register(mux, sc)

    srv := &http.Server{Addr: addr, Handler: mux}
    go func() {
        fmt.Println("开始监听：", srv)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            fmt.Println("开始监听：", srv)
            logx.Error("dtm http server error: ", err)
        }
    }()
    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        _ = srv.Shutdown(ctx)
    }
}
