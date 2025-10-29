package main

import (
    "flag"
    "fmt"
    "log"

    "github.com/casbin/casbin/v2"
    fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
    gormadapter "github.com/casbin/gorm-adapter/v3"
)

// A tiny helper to import Casbin policies from CSV into MySQL using gorm-adapter.
// Usage:
//   go run ./tools/casbinimport \
//     -dsn    "root:Natsume@tcp(mysql:3306)/Natsume?charset=utf8mb4&parseTime=True&loc=Local" \
//     -model  "manifest/casbin/model.conf" \
//     -policy "manifest/casbin/policy.csv"
func main() {
    dsn := flag.String("dsn", "", "MySQL DSN for casbin gorm-adapter")
    model := flag.String("model", "manifest/casbin/model.conf", "path to Casbin model.conf")
    policy := flag.String("policy", "manifest/casbin/policy.csv", "path to policy.csv to import")
    truncate := flag.Bool("truncate", false, "truncate existing casbin rules before import")
    flag.Parse()

    if *dsn == "" {
        log.Fatal("-dsn is required")
    }

    // Load policies from CSV into memory
    eFile, err := casbin.NewEnforcer(*model, fileadapter.NewAdapter(*policy))
    if err != nil { log.Fatalf("new file enforcer: %v", err) }
    if err := eFile.LoadPolicy(); err != nil { log.Fatalf("load file policy: %v", err) }

    // Prepare DB adapter
    a, err := gormadapter.NewAdapter("mysql", *dsn, true)
    if err != nil { log.Fatalf("new gorm adapter: %v", err) }

    // Persist into DB
    // Simple approach: optionally clear DB, then switch adapter and SavePolicy
    if *truncate {
        eTmp, err := casbin.NewEnforcer(*model, a)
        if err != nil { log.Fatalf("new db enforcer: %v", err) }
        eTmp.ClearPolicy()
        if err := eTmp.SavePolicy(); err != nil { log.Fatalf("flush db policy: %v", err) }
    }
    eFile.SetAdapter(a)
    if err := eFile.SavePolicy(); err != nil { log.Fatalf("save policy to db: %v", err) }

    // Verify by reloading from DB
    eDB, err := casbin.NewEnforcer(*model, a)
    if err != nil { log.Fatalf("new db enforcer: %v", err) }
    if err := eDB.LoadPolicy(); err != nil { log.Fatalf("load db policy: %v", err) }

    ps, _ := eDB.GetPolicy()
    gs, _ := eDB.GetGroupingPolicy()
    fmt.Printf("Imported %d p, %d g rules into DB.\n", len(ps), len(gs))
}
