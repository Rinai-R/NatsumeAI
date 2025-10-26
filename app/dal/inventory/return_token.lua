-- Note: Redis Lua provides global `cjson`; do not `require`.

-- 单商品版本：固定处理 1 个商品
-- KEYS:
--   1: epoch_key (inv:{sku}:epoch)
-- ARGV:
--   1: ticket_json（包含 items[1]）
--   2: preorder_id（可选，用于删除 adm:{preorder_id}）
--   3: expect_epoch_str（可选，字符串，避免 JSON 数字精度问题）

local ticket_json = ARGV[1]
if not ticket_json or ticket_json == "" then
    return { "INVALID_TICKET" }
end

local preorder_id = tostring(ARGV[2] or "")
local ticket_key = ""
if preorder_id ~= "" then
    ticket_key = "adm:" .. preorder_id
end

local ok_ticket, ticket = pcall(cjson.decode, ticket_json)
if not ok_ticket or type(ticket) ~= "table" or type(ticket.item) ~= "table" then
    return { "INVALID_TICKET" }
end

local epoch_key = KEYS[1]

local item = ticket.item
local sku = tostring(item.sku or "")
local qty = tonumber(item.qty or item.quantity or 0)
-- 优先使用参数中的精确 epoch 字符串
local expect_epoch = tostring(ARGV[3] or "")
if expect_epoch == "" then
    expect_epoch = tostring(item.epoch or "")
end
if sku == "" or not qty or qty <= 0 or expect_epoch == "" then
    return { "INVALID_ITEM" }
end

local current_epoch = redis.call("GET", epoch_key)
if not current_epoch or tostring(current_epoch) ~= expect_epoch then
    -- 跨 epoch，忽略回退，认为已过期
    if ticket_key ~= "" then redis.call("DEL", ticket_key) end
    return { "OK", 0, 1, "EPOCH_MISMATCH", tostring(current_epoch or ""), expect_epoch }
end

-- Build keys dynamically based on expected epoch
local threshold_key = "inv:{" .. sku .. "}:threshold:" .. expect_epoch
local issued_key = "inv:{" .. sku .. "}:issued:" .. expect_epoch

local threshold_raw = redis.call("GET", threshold_key)
if not threshold_raw then
    if ticket_key ~= "" then redis.call("DEL", ticket_key) end
    return { "OK", 0, 1, "NO_THRESHOLD" }
end

local threshold = tonumber(threshold_raw)
if not threshold then
    if ticket_key ~= "" then redis.call("DEL", ticket_key) end
    return { "OK", 0, 1, "INVALID_THRESHOLD" }
end

local issued = tonumber(redis.call("GET", issued_key) or "0")
local issued_after = issued - qty
if issued_after < 0 then issued_after = 0 end
if issued_after > threshold then issued_after = threshold end
redis.call("SET", issued_key, issued_after)

if ticket_key ~= "" then
    redis.call("DEL", ticket_key)
end

return { "OK", 1, 0, "ROLLED_BACK", tostring(issued), tostring(issued_after) }
