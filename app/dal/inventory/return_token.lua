-- 单商品版本：固定处理 1 个商品
-- KEYS:
--   1: epoch_key (inv:{sku}:epoch)
-- ARGV:
--   1: ticket_json（包含 item，可作为后备解析）
--   2: preorder_id（用于定位 adm:{preorder_id}）
--   3: expect_epoch_str（可选，字符串，避免 JSON 数字精度问题）

local ticket_json = ARGV[1]
local preorder_id = tostring(ARGV[2] or "")
local epoch_key = KEYS[1]

local ticket_key = ""
if preorder_id ~= "" then
    ticket_key = "adm:" .. preorder_id
end

-- 为了避免重复回退，只有当票据存在时才执行回退；否则视为幂等成功
local stored = nil
if ticket_key ~= "" then
    stored = redis.call("GET", ticket_key)
end
if not stored or stored == false then
    return { "OK", 0, 1, "NO_TICKET" }
end

-- 解析票据，以存储的为准；若异常，尝试使用入参作为后备
local ok_ticket, ticket = pcall(cjson.decode, stored)
if not ok_ticket or type(ticket) ~= "table" or type(ticket.item) ~= "table" then
    ok_ticket, ticket = pcall(cjson.decode, ticket_json)
    if not ok_ticket or type(ticket) ~= "table" or type(ticket.item) ~= "table" then
        return { "INVALID_TICKET" }
    end
end

local item = ticket.item
local sku = tostring(item.sku or "")
local qty = tonumber(item.qty or item.quantity or 0)
local expect_epoch = tostring(ARGV[3] or "")
if expect_epoch == "" then
    expect_epoch = tostring(item.epoch or "")
end
if sku == "" or not qty or qty <= 0 or expect_epoch == "" then
    return { "INVALID_ITEM" }
end

local current_epoch = redis.call("GET", epoch_key)
if not current_epoch or tostring(current_epoch) ~= expect_epoch then
    if ticket_key ~= "" then redis.call("DEL", ticket_key) end
    return { "OK", 0, 1, "EPOCH_MISMATCH", tostring(current_epoch or ""), expect_epoch }
end

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
local delta = qty
if delta > issued then delta = issued end
local issued_after = issued - delta
if issued_after < 0 then issued_after = 0 end
if issued_after > threshold then issued_after = threshold end
redis.call("SET", issued_key, issued_after)

if ticket_key ~= "" then
    redis.call("DEL", ticket_key)
end

return { "OK", 1, 0, "ROLLED_BACK", tostring(issued), tostring(issued_after) }
