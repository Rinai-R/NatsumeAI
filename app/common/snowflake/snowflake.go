package snowflake

import (
    "hash/fnv"
    "os"
    "sync"

    bwsnowflake "github.com/bwmarrin/snowflake"
)

var (
    once sync.Once
    node *bwsnowflake.Node
)

// SetNodeID allows overriding the derived node ID (0-1023). Call once at bootstrap.
func SetNodeID(id int64) error {
    var err error
    once.Do(func() {}) // ensure we can set before init
    node, err = bwsnowflake.NewNode(id & 0x3FF)
    return err
}

func initNode() {
    if node != nil { return }
    // derive node from hostname hash (10 bits)
    host, _ := os.Hostname()
    h := fnv.New32a()
    _, _ = h.Write([]byte(host))
    id := int64(h.Sum32()) & 0x3FF
    n, err := bwsnowflake.NewNode(id)
    if err != nil {
        // fallback to node 1
        n, _ = bwsnowflake.NewNode(1)
    }
    node = n
}

// Next returns a new snowflake id using bwmarrin/snowflake.
func Next() int64 {
    once.Do(initNode)
    return node.Generate().Int64()
}

