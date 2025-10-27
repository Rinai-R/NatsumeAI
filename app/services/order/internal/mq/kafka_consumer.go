package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"NatsumeAI/app/common/consts/errno"
	orderdal "NatsumeAI/app/dal/order"
	couponsvcpb "NatsumeAI/app/services/coupon/coupon"
	invpb "NatsumeAI/app/services/inventory/inventory"
	"NatsumeAI/app/services/order/internal/svc"
	prodpb "NatsumeAI/app/services/product/product"

	"github.com/zeromicro/go-zero/core/logx"

	"strconv"

	"github.com/dtm-labs/client/dtmgrpc"
	_ "github.com/dtm-labs/dtmdriver-gozero"
	"github.com/hibiken/asynq"
	"github.com/segmentio/kafka-go"
)

// StartCheckoutConsumer starts a blocking Kafka consumer loop for checkout events.
// It performs pre-freeze and inserts preorder item, then schedules a delayed cancel task.
func StartCheckoutConsumer(ctx context.Context, sc *svc.ServiceContext) error {
    if len(sc.Config.KafkaConf.Broker) == 0 || sc.Config.KafkaConf.PreOrderTopic == "" || sc.Config.KafkaConf.Group == "" {
        return nil
    }
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:     sc.Config.KafkaConf.Broker,
        GroupID:     sc.Config.KafkaConf.Group,
        Topic:       sc.Config.KafkaConf.PreOrderTopic,
        MinBytes:    1,
        MaxBytes:    10 << 20,
        // Reduce fetch wait to improve end-to-end latency
        MaxWait:     50 * time.Millisecond,
        StartOffset: kafka.FirstOffset,
    })
    defer r.Close()

    for {
        m, err := r.FetchMessage(ctx)
        if err != nil {
            if ctx.Err() != nil {
                return nil 
            }
            continue
        }
        var evt CheckoutEvent
        if err := json.Unmarshal(m.Value, &evt); err == nil {
            _ = handleCheckout(ctx, sc, evt)
        }
        _ = r.CommitMessages(ctx, m)
    }
}

// 消息队列处理 checkout，负责订单创建，库存扣减，如果失败，标记 fail 并回滚
func handleCheckout(c context.Context, s *svc.ServiceContext, e CheckoutEvent) error {
    preorderID := e.PreorderId

    logx.WithContext(c).Error("接收到消息了：", e)
    // logx.WithContext(c).Info("接收到消息了：", e)
    // 幂等
    if rows, err := s.PreItm.ListByPreorder(c, preorderID); err == nil && len(rows) > 0 {
        logx.WithContext(c).Error("重复请求")
        return nil
    }

    // 构建快照与价格：优先使用事件自带数据，缺失时再降级查询商品
    var (
        priceCents int64 = e.PriceCents
        snap       *CheckoutSnapshot = e.Snapshot
    )
    if priceCents <= 0 || snap == nil {
        if s.Product == nil {
            // 缺少商品服务且事件未带必要信息，回滚
            if resp, err := s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{
                PreorderId: preorderID,
                Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity},
            }); err != nil {
                logx.WithContext(c).Errorf("rollback return token failed: preorder=%d product=%d qty=%d err=%v", preorderID, e.ProductId, e.Quantity, err)
            } else if resp != nil && resp.StatusCode != errno.StatusOK {
                logx.WithContext(c).Infof("rollback return token status: preorder=%d code=%d msg=%s", preorderID, resp.StatusCode, resp.StatusMsg)
            }
            _ = s.Preorder.Delete(c, preorderID)
            return nil
        }
        if pr, err := s.Product.GetProduct(c, &prodpb.GetProductReq{ProductId: e.ProductId, UserId: e.UserId}); err == nil && pr != nil && pr.Product != nil {
            if priceCents <= 0 {
                priceCents = pr.Product.Price
            }
            if snap == nil {
                snap = &CheckoutSnapshot{Title: pr.Product.Name, CoverImage: pr.Product.Picture, Attributes: pr.Product.Description}
            }
        } else {
            // 商品查询失败或不存在，回滚
            if resp, err := s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); err != nil {
                logx.WithContext(c).Errorf("rollback return token failed: preorder=%d product=%d qty=%d err=%v", preorderID, e.ProductId, e.Quantity, err)
            } else if resp != nil && resp.StatusCode != errno.StatusOK {
                logx.WithContext(c).Infof("rollback return token status: preorder=%d code=%d msg=%s", preorderID, resp.StatusCode, resp.StatusMsg)
            }
            _ = s.Preorder.Delete(c, preorderID)
            return nil
        }
    }

    // 优惠券加锁迁移至消费者，成功后发生的错误才需要释放
    // 使用 DTM gRPC SAGA 编排：锁券(+回滚释放) + 预冻结库存(+回滚释放)
    if s.Config.DtmConf.Server != "" {
        gid := "saga-preorder-" + strconv.FormatInt(preorderID, 10)
        server := s.Config.DtmConf.GrpcServer
        if server == "" {
            // fallback to Server if user misconfigured; strip leading scheme if looks like http(s)://host:port/path
            server = s.Config.DtmConf.Server
        }
        saga := dtmgrpc.NewSagaGrpc(server, gid)
        // 等待分支结果，确保远端资源就绪后再进行本地写入与READY置位
        saga.Saga.TransBase.WaitResult = true

        // 构建 go-zero target（consul:// 或直连等），driver-gozero 将解析
        couponTarget, _ := s.Config.CouponRpc.BuildTarget()
        invTarget, _ := s.Config.InventoryRpc.BuildTarget()

        // Step1: LockCoupon -> ReleaseCoupon
        if e.CouponId > 0 && s.Coupon != nil {
            lockReq := &couponsvcpb.LockCouponReq{UserId: e.UserId, CouponId: e.CouponId, OrderId: preorderID}
            saga.Add(couponTarget+couponsvcpb.CouponService_LockCoupon_FullMethodName, couponTarget+couponsvcpb.CouponService_ReleaseCoupon_FullMethodName, lockReq)
        }

        // Step2: DecreasePreInventory -> ReturnPreInventory
        invReq := &invpb.InventoryReq{OrderId: preorderID, PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}
        saga.Add(invTarget+invpb.InventoryService_DecreasePreInventory_FullMethodName, invTarget+invpb.InventoryService_ReturnPreInventory_FullMethodName, invReq)

        if err := saga.Submit(); err != nil {
            // 提交失败，归还 token 并删除预订单（券/库存由 SAGA 补偿）
            if resp, rerr := s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); rerr != nil {
                logx.WithContext(c).Errorf("saga submit fail: return token failed: preorder=%d product=%d qty=%d err=%v", preorderID, e.ProductId, e.Quantity, rerr)
            } else if resp != nil && resp.StatusCode != errno.StatusOK {
                logx.WithContext(c).Infof("saga submit fail: return token status: preorder=%d code=%d msg=%s", preorderID, resp.StatusCode, resp.StatusMsg)
            }
            _ = s.Preorder.Delete(c, preorderID)
            return err
        }
    } else {
        // 无 DTM 配置，走原有直连逻辑（略）
        if e.CouponId > 0 && s.Coupon != nil {
            if lr, err := s.Coupon.LockCoupon(c, &couponsvcpb.LockCouponReq{UserId: e.UserId, CouponId: e.CouponId, OrderId: preorderID}); err != nil || lr == nil || lr.StatusCode != errno.StatusOK {
                if err != nil {
                    logx.WithContext(c).Errorf("coupon lock rpc failed: preorder=%d coupon=%d user=%d err=%v", preorderID, e.CouponId, e.UserId, err)
                } else {
                    logx.WithContext(c).Infof("coupon lock rejected: preorder=%d code=%d msg=%s", preorderID, lr.StatusCode, lr.StatusMsg)
                }
                if resp, rerr := s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); rerr != nil {
                    logx.WithContext(c).Errorf("rollback return token failed: preorder=%d product=%d qty=%d err=%v", preorderID, e.ProductId, e.Quantity, rerr)
                } else if resp != nil && resp.StatusCode != errno.StatusOK {
                    logx.WithContext(c).Infof("rollback return token status: preorder=%d code=%d msg=%s", preorderID, resp.StatusCode, resp.StatusMsg)
                }
                _ = s.Preorder.Delete(c, preorderID)
                return nil
            }
        }
        if rp, err := s.Inventory.DecreasePreInventory(c, &invpb.InventoryReq{OrderId: preorderID, PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); err != nil || (rp != nil && rp.StatusCode != errno.StatusOK) {
            if resp, _ := s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); resp != nil { /* log below */ }
            _ = s.Preorder.Delete(c, preorderID)
            if err != nil { return err }
            return nil
        }
    }

    // 快照
    var snapStr sql.NullString
    if b, err := json.Marshal(snap); err == nil {
        snapStr = sql.NullString{
            String: string(b),
            Valid:  true,
        }
    }
    
    // 这里不用事务的原因是这里还没有 ack mq 的消息，所以如果此时服务器挂了会重复投递然后保证至少一次，因此上面
    // 的事务需要做好幂等
    if res, err := s.PreItm.Insert(c, &orderdal.OrderPreorderItems{
        PreorderId: preorderID, 
        ProductId: e.ProductId, 
        Quantity: e.Quantity, 
        PriceCents: priceCents, 
        Snapshot: snapStr,
    }); err != nil {
        // 幂等
        if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
            return nil
        }
        // 否则清理：归还 token + 删除预订单（SAGA 已完成两阶段，无需手动释放券/库存）
        if resp, err := s.Inventory.ReturnToken(c, &invpb.ReturnTokenReq{PreorderId: preorderID, Item: &invpb.Item{ProductId: e.ProductId, Quantity: e.Quantity}}); err != nil {
            logx.WithContext(c).Errorf("rollback return token failed: preorder=%d product=%d qty=%d err=%v", preorderID, e.ProductId, e.Quantity, err)
        } else if resp != nil && resp.StatusCode != errno.StatusOK {
            logx.WithContext(c).Infof("rollback return token status: preorder=%d code=%d msg=%s", preorderID, resp.StatusCode, resp.StatusMsg)
        }
        _ = s.Preorder.Delete(c, preorderID)
        return err
    } else {
        _, _ = res.LastInsertId()
    }

    // 标记预订单 READY（仅当当前仍为 PENDING）
    if ok, err := s.Preorder.MarkReadyIfPending(c, preorderID); err != nil {
        logx.WithContext(c).Errorf("mark preorder ready failed: preorder=%d err=%v", preorderID, err)
    } else if ok {
        logx.WithContext(c).Infof("preorder marked READY: %d", preorderID)
    }

    // 发送延时任务取消（统一使用全局 TTL）
    delay := s.PreorderTTL
    if po, err := s.Preorder.FindOne(c, preorderID); err == nil {
        if d := time.Until(po.ExpireAt); d > 0 { delay = d } else { delay = time.Second * 1 }
    }
    payload, _ := json.Marshal(CancelTaskPayload{
        PreorderId: preorderID,
        UserId:     e.UserId,
    })
    task := asynq.NewTask(TaskCancelPreorder, payload)
    data, err := s.AsynqClient.Enqueue(task, asynq.ProcessIn(delay), asynq.Queue("default"))
    if err != nil {
        logx.WithContext(c).Error("asynq error: ", err, "info: ", data)
        return err
    }
    return nil
}

// NewAsynqMux 注册一个处理了延时任务的 handle
func NewAsynqMux(sc *svc.ServiceContext) *asynq.ServeMux {
    mux := asynq.NewServeMux()
    // Register handlers
    mux.HandleFunc(TaskCancelOrder, CancelOrderHandler(sc))
    mux.HandleFunc(TaskCancelPreorder, CancelPreorderHandler(sc))
    return mux
}

// CancelPreorderHandler returns an asynq handler that cancels a preorder if still pending.
func CancelPreorderHandler(sc *svc.ServiceContext) asynq.HandlerFunc {
    return func(ctx context.Context, t *asynq.Task) error {
        var p CancelTaskPayload
        if err := json.Unmarshal(t.Payload(), &p); err != nil {
            return err
        }
        return handleCancelPreorderTask(ctx, sc, p)
    }
}

// CancelOrderHandler returns an asynq handler that cancels an unpaid order.
func CancelOrderHandler(sc *svc.ServiceContext) asynq.HandlerFunc {
    return func(ctx context.Context, t *asynq.Task) error {
        var p CancelOrderTaskPayload
        if err := json.Unmarshal(t.Payload(), &p); err != nil {
            return err
        }
        return handleCancelOrderTask(ctx, sc, p)
    }
}

// handleCancelPreorderTask rolls back resources and cancels the preorder if still pending and no order exists.
func handleCancelPreorderTask(ctx context.Context, sc *svc.ServiceContext, p CancelTaskPayload) error {
    // 原子检查并更新，仅当仍为 PENDING 且还未生成订单时，标记为 CANCELLED
    cancelled, err := sc.Preorder.CancelIfPendingAndNoOrder(ctx, p.PreorderId)
    if err != nil {
        return err
    }
    if !cancelled {
        return nil
    }

    var itemProductId int64
    var itemQty int64
    if rows, err := sc.PreItm.ListByPreorder(ctx, p.PreorderId); err == nil && len(rows) > 0 {
        itemProductId, itemQty = rows[0].ProductId, rows[0].Quantity
    }
    // 解冻库存，同时会 returnToken，需要做好幂等
    if resp, err := sc.Inventory.ReturnPreInventory(ctx, &invpb.InventoryReq{
        OrderId:    p.PreorderId, 
        PreorderId: p.PreorderId, 
        Item: &invpb.Item{
            ProductId: itemProductId, 
            Quantity:  itemQty,
        },
    }); err != nil {
        logx.WithContext(ctx).Errorf("cancel task: return pre inventory failed: preorder=%d product=%d qty=%d err=%v", p.PreorderId, itemProductId, itemQty, err)
    } else if resp != nil && resp.StatusCode != errno.StatusOK {
        logx.WithContext(ctx).Infof("cancel task: return pre inventory status: preorder=%d code=%d msg=%s", p.PreorderId, resp.StatusCode, resp.StatusMsg)
    }
    // 释放优惠券
    if po, err := sc.Preorder.FindOne(ctx, p.PreorderId); err == nil && po.CouponId > 0 && sc.Coupon != nil {
        _, _ = sc.Coupon.ReleaseCoupon(ctx, &couponsvcpb.ReleaseCouponReq{
            UserId:  po.UserId, 
            CouponId: po.CouponId, 
            OrderId: p.PreorderId,
        })
    }
    return nil
}

// handleCancelOrderTask cancels an unpaid order and rolls back related resources.
func handleCancelOrderTask(ctx context.Context, sc *svc.ServiceContext, p CancelOrderTaskPayload) error {
    ord, err := sc.Orders.FindOne(ctx, p.OrderId)
    if err != nil {
        return nil
    }
    if ord.Status != "PENDING_PAYMENT" {
        return nil
    }
    // get item from preorder items
    var itemProductId int64
    var itemQty int64
    if rows, err := sc.PreItm.ListByPreorder(ctx, ord.PreorderId); err == nil && len(rows) > 0 {
        itemProductId, itemQty = rows[0].ProductId, rows[0].Quantity
    }
    // unfreeze inventory
    if resp, err := sc.Inventory.ReturnPreInventory(ctx, &invpb.InventoryReq{
        OrderId:    ord.PreorderId,
        PreorderId: ord.PreorderId,
        Item: &invpb.Item{
            ProductId: itemProductId,
            Quantity:  itemQty,
        },
    }); err != nil {
        logx.WithContext(ctx).Errorf("order cancel: return pre inventory failed: order=%d preorder=%d err=%v", ord.OrderId, ord.PreorderId, err)
    } else if resp != nil && resp.StatusCode != errno.StatusOK {
        logx.WithContext(ctx).Infof("order cancel: return pre inventory status: order=%d preorder=%d code=%d msg=%s", ord.OrderId, ord.PreorderId, resp.StatusCode, resp.StatusMsg)
    }
    // release coupon
    if ord.CouponId > 0 && sc.Coupon != nil {
        _, _ = sc.Coupon.ReleaseCoupon(ctx, &couponsvcpb.ReleaseCouponReq{
            UserId:  ord.UserId,
            CouponId: ord.CouponId,
            OrderId: p.OrderId,
        })
    }
    // token 将在 ReturnPreInventory 内部归还，无需显式 ReturnToken
    // update order status
    ord.Status = "CANCELLED"
    if err := sc.Orders.Update(ctx, ord); err != nil {
        logx.WithContext(ctx).Errorf("order cancel: update status failed: order=%d err=%v", ord.OrderId, err)
    }
    return nil
}
