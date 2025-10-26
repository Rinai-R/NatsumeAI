-- Note: Redis Lua provides global `cjson`; do not `require`.

-- KEYS:
--   1: epoch_key (inv:{sku}:epoch)
-- ARGV:
--   1: preorder_id (string)
--   2: ticket_ttl_seconds (number)
--   3: ticket_json (包含 item，epoch 字段可有可无)

local preorder_id = tostring(ARGV[1] or "")
if preorder_id == "" then
    return { "INVALID_PREORDER" }
end

local ttl = tonumber(ARGV[2]) or 0

local ticket_json = ARGV[3]
if not ticket_json or ticket_json == "" then
    return { "INVALID_TICKET" }
end

local ok_ticket, ticket = pcall(cjson.decode, ticket_json)
if not ok_ticket or type(ticket) ~= "table" or type(ticket.item) ~= "table" then
    return { "INVALID_TICKET_ITEMS" }
end

local ticket_key = "adm:" .. preorder_id

-- 幂等锁
local lock_res
if ttl > 0 then
    lock_res = redis.call("SET", ticket_key, "__LOCK__", "NX", "EX", ttl)
else
    lock_res = redis.call("SET", ticket_key, "__LOCK__", "NX")
end
if not lock_res then
    return { "DUPLICATE", preorder_id }
end

local item = ticket.item
if not item then
    redis.call("DEL", ticket_key)
    return { "INVALID_ITEM" }
end

local sku = tostring(item.sku or "")
if sku == "" then
    redis.call("DEL", ticket_key)
    return { "INVALID_SKU" }
end

-- 注意：不要依赖 ticket 内嵌的 epoch（JSON 数字精度可能丢失）
local need = tonumber(item.qty or item.quantity or 0)
if not need or need <= 0 then
    redis.call("DEL", ticket_key)
    return { "INVALID_QUANTITY", sku }
end

local epoch_key = KEYS[1]

local epoch = redis.call("GET", epoch_key)
if not epoch then
    redis.call("DEL", ticket_key)
    return { "NO_EPOCH", sku }
end

-- Compute keys after verifying epoch to avoid client-side key drift
local epoch_str = tostring(epoch)
local threshold_key = "inv:{" .. sku .. "}:threshold:" .. epoch_str
local issued_key = "inv:{" .. sku .. "}:issued:" .. epoch_str

local threshold_raw = redis.call("GET", threshold_key)
if not threshold_raw then
    redis.call("DEL", ticket_key)
    return { "NO_THRESHOLD", sku }
end
local threshold = tonumber(threshold_raw)
if not threshold then
    redis.call("DEL", ticket_key)
    return { "INVALID_THRESHOLD", sku }
end

local issued = tonumber(redis.call("GET", issued_key) or "0")
local available = threshold - issued
if available < 0 then available = 0 end
if available < need then
    redis.call("DEL", ticket_key)
    return { "NOT_ENOUGH", sku, available }
end

-- 扣减 issued
redis.call("INCRBY", issued_key, need)

-- 写入 ticket（替换锁值）
local ok
if ttl > 0 then
    ok = redis.call("SET", ticket_key, ticket_json, "XX", "EX", ttl)
else
    ok = redis.call("SET", ticket_key, ticket_json, "XX")
end
if not ok then
    -- 回滚
    redis.call("DECRBY", issued_key, need)
    redis.call("DEL", ticket_key)
    return { "TICKET_STORE_FAILED", preorder_id }
end

return { "OK" }
