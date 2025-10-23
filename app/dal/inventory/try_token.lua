local cjson = require("cjson")

--[[
键（KEYS）布局：每个商品 3 个键，按顺序排列
  1: sku1 的 epoch 键（版本号）
  2: sku1 的 threshold 键（库存快照阈值）
  3: sku1 的 issued 键（已发放的令牌计数）
  4: sku2 的 epoch 键
  5: sku2 的 threshold 键
  6: sku2 的 issued 键
  ...

参数（ARGV）:
  1: 商品条目数 item_count（n）
  2: 预订单号 preorderId（用作幂等票据键的一部分）
  3: 准入票据 TTL（秒）
  4: 准入票据 JSON 字符串（包含 items 数组，已保证去重）
]]

local item_count = tonumber(ARGV[1])
if not item_count or item_count <= 0 then
    return { "INVALID_ITEM_COUNT" }
end

local preorder_id = tostring(ARGV[2] or "")
if preorder_id == "" then
    return { "INVALID_PREORDER" }
end

local ttl = tonumber(ARGV[3]) or 300
if ttl <= 0 then
    ttl = 300
end

local ticket_json = ARGV[4]
if not ticket_json or ticket_json == "" then
    return { "INVALID_TICKET" }
end

local ok_ticket, ticket = pcall(cjson.decode, ticket_json)
if not ok_ticket or type(ticket) ~= "table" or type(ticket.items) ~= "table" then
    return { "INVALID_TICKET_ITEMS" }
end

if #ticket.items ~= item_count then
    return { "ITEM_COUNT_MISMATCH" }
end

local ticket_key = "adm:" .. preorder_id

-- 拿锁保证幂等
local lock_res = redis.call("SET", ticket_key, "__LOCK__", "NX", "EX", ttl)
if not lock_res then
    return { "DUPLICATE", preorder_id }
end

local keys_index = 1

-- 校验
for i = 1, item_count do
    local item = ticket.items[i]
    if not item then
        redis.call("DEL", ticket_key)
        return { "INVALID_ITEM", i }
    end

    local sku = tostring(item.sku or "")
    if sku == "" then
        redis.call("DEL", ticket_key)
        return { "INVALID_SKU", i }
    end

    local expect_epoch = tostring(item.epoch or "")
    if expect_epoch == "" then
        redis.call("DEL", ticket_key)
        return { "INVALID_EPOCH", sku }
    end

    local need = tonumber(item.qty or item.quantity or 0)
    if not need or need <= 0 then
        redis.call("DEL", ticket_key)
        return { "INVALID_QUANTITY", sku }
    end

    local epoch_key = KEYS[keys_index]
    local threshold_key = KEYS[keys_index + 1]
    local issued_key = KEYS[keys_index + 2]
    keys_index = keys_index + 3

    local epoch = redis.call("GET", epoch_key)
    if not epoch then
        redis.call("DEL", ticket_key)
        return { "NO_EPOCH", sku }
    end

    if tostring(epoch) ~= expect_epoch then
        redis.call("DEL", ticket_key)
        return { "STALE", sku, epoch }
    end

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
    if available < 0 then
        available = 0
    end
    if available < need then
        redis.call("DEL", ticket_key)
        return { "NOT_ENOUGH", sku, available }
    end
end

-- 实际扣减
keys_index = 1
for i = 1, item_count do
    local item = ticket.items[i]
    local qty = tonumber(item.qty or item.quantity)
    local issued_key = KEYS[keys_index + 2]
    redis.call("INCRBY", issued_key, qty)
    keys_index = keys_index + 3
end

-- 持久化 token
local ok = redis.call("SET", ticket_key, ticket_json, "XX", "EX", ttl)
if not ok then
    -- 失败回滚数据
    keys_index = 1
    for i = 1, item_count do
        local item = ticket.items[i]
        local qty = tonumber(item.qty or item.quantity)
        local issued_key = KEYS[keys_index + 2]
        redis.call("DECRBY", issued_key, qty)
        keys_index = keys_index + 3
    end
    redis.call("DEL", ticket_key)
    return { "TICKET_STORE_FAILED", preorder_id }
end

return { "OK" }
