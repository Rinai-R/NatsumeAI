package config

import (
	"strings"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	rediswatcher "github.com/casbin/redis-watcher/v2"
	redis "github.com/redis/go-redis/v9"
)

// CasbinMiddlewareConf holds config for creating an Enforcer.
// Dns+Model are required. Optional: Watcher for distributed sync (e.g., Redis).
type CasbinMiddlewareConf struct {
    Dns     string
    Model   string
    Watcher WatcherConf
}

// WatcherConf configures the casbin watcher.
type WatcherConf struct {
    Type       string // "redis"
    Addr       string // e.g., "redis:6379"
    Channel    string // e.g., "casbin:policy"
    DB         int
    Password   string
    IgnoreSelf bool
}

func (c *CasbinMiddlewareConf) MustNewDistributedEnforcer() (enforcer *casbin.DistributedEnforcer) {
    adapter, err := gormadapter.NewAdapter("mysql", c.Dns, true)
    if err != nil {
        panic(err)
    }

    enforcer, err = casbin.NewDistributedEnforcer(c.Model, adapter)
    if err != nil {
        panic(err)
    }

    if strings.EqualFold(c.Watcher.Type, "redis") && c.Watcher.Addr != "" {
        w, werr := rediswatcher.NewWatcher(c.Watcher.Addr, rediswatcher.WatcherOptions{
            Options:    redis.Options{
				Addr: c.Watcher.Addr, 
				DB: c.Watcher.DB, 
				Password: c.Watcher.Password,
			},
            Channel:    c.Watcher.Channel,
            IgnoreSelf: c.Watcher.IgnoreSelf,
        })
        if werr == nil {
            _ = w.SetUpdateCallback(func(string) { _ = enforcer.LoadPolicy() })
            _ = enforcer.SetWatcher(w)
        }
    }
    return enforcer
}
