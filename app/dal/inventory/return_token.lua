local cjson = require("cjson")

--[[
脚本用途：
  当某个预订单（preorder）因超时、取消、支付失败等原因，
  需要归还已发放的库存令牌（issued）时执行。

key 布局（每个商品 3 个 key）：
  1: sku1 的 epoch 键（版本号，用于校验当前库存批次是否一致）
  2: sku1 的 threshold 键（库存阈值快照）
  3: sku1 的 issued 键（已发出的令牌数量）
  4: sku2 的 epoch 键
  5: sku2 的 threshold 键
  6: sku2 的 issued 键
  ...

参数（ARGV）:
  1: 商品条目数 n
  2: 准入票据 JSON（包含 items 数组，每个 item 包含 sku、qty、epoch）
  3: preorderId（可选，用于删除 adm:<preorderId> 票据键）
]]

local item_count = tonumber(ARGV[1])
if not item_count or item_count <= 0 then
    return { "INVALID_ITEM_COUNT" }
end

local ticket_json = ARGV[2]
if not ticket_json or ticket_json == "" then
    return { "INVALID_TICKET" }
end

local preorder_id = tostring(ARGV[3] or "")
local ticket_key = ""
if preorder_id ~= "" then
    ticket_key = "adm:" .. preorder_id
end

local ok_ticket, ticket = pcall(cjson.decode, ticket_json)
if not ok_ticket or type(ticket) ~= "table" then
    return { "INVALID_TICKET" }
end

local items = ticket.items
if type(items) ~= "table" then
    items = ticket
end

if type(items) ~= "table" or #items ~= item_count then
    return { "ITEM_COUNT_MISMATCH" }
end

local restored = 0
local skipped = 0
local keys_index = 1

for i = 1, item_count do
    local epoch_key = KEYS[keys_index]
    local threshold_key = KEYS[keys_index + 1]
    local issued_key = KEYS[keys_index + 2]
    keys_index = keys_index + 3

    local item = items[i]
    if not item then
        skipped = skipped + 1
    else
        local sku = tostring(item.sku or "")
        local qty = tonumber(item.qty or item.quantity or 0)
        local expect_epoch = tostring(item.epoch or "")
        if sku == "" or not qty or qty <= 0 or expect_epoch == "" then
            skipped = skipped + 1
        else
            -- 这里是防止旧 epoch 的数据被误归还
            local current_epoch = redis.call("GET", epoch_key)
            if current_epoch and tostring(current_epoch) == expect_epoch then
                local threshold_raw = redis.call("GET", threshold_key)
                if not threshold_raw then
                    skipped = skipped + 1
                else
                    local threshold = tonumber(threshold_raw)
                    local issued = tonumber(redis.call("GET", issued_key) or "0")
                    if not threshold then
                        skipped = skipped + 1
                    else
                        local issued_after = issued - qty
                        if issued_after < 0 then
                            issued_after = 0
                        end
                        if issued_after > threshold then
                            issued_after = threshold
                        end
                        redis.call("SET", issued_key, issued_after)
                        restored = restored + 1
                    end
                end
            else
                skipped = skipped + 1
            end
        end
    end
end

if ticket_key ~= "" then
    redis.call("DEL", ticket_key)
end

return { "OK", restored, skipped }
