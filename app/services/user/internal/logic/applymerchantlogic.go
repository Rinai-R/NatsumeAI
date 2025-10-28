package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	userdal "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/user/internal/mq"
	"NatsumeAI/app/services/user/internal/svc"
	"NatsumeAI/app/services/user/user"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyMerchantLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewApplyMerchantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyMerchantLogic {
    return &ApplyMerchantLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *ApplyMerchantLogic) ApplyMerchant(in *user.ApplyMerchantRequest) (*user.ApplyMerchantResponse, error) {
    resp := &user.ApplyMerchantResponse{StatusCode: 0, StatusMsg: "OK"}

    // Basic validation
    if in.GetUserId() <= 0 || in.GetApplication() == nil {
        resp.StatusCode = -1
        resp.StatusMsg = "invalid request"
        return resp, nil
    }
    app := in.GetApplication()
    // normalize
    shop := strings.TrimSpace(app.GetShopName())
    contact := strings.TrimSpace(app.GetContactName())
    phone := strings.TrimSpace(app.GetContactPhone())
    addr := strings.TrimSpace(app.GetAddress())
    desc := strings.TrimSpace(app.GetDescription())
    if shop == "" || contact == "" || phone == "" || addr == "" {
        resp.StatusCode = -2
        resp.StatusMsg = "required fields missing"
        return resp, nil
    }

    // // Idempotency: if user already has an application, handle by current status
    // if row, err := l.svcCtx.MerchantsModel.FindOneByUserId(l.ctx, uint64(in.GetUserId())); err == nil && row != nil {
    //     switch strings.ToUpper(row.Status) {
    //     case "APPROVED":
    //         // 已通过，直接返回
    //         resp.ApplicationId = row.Id
    //         resp.ApplicationStatus = row.Status
    //         return resp, nil
    //     case "REJECTED":
    //         // 驳回后允许更新资料并重新提交为 PENDING
    //         row.ShopName = shop
    //         row.ContactName = contact
    //         row.ContactPhone = phone
    //         row.Address = addr
    //         row.Description = sql.NullString{String: desc, Valid: desc != ""}
    //         row.Status = "PENDING"
    //         row.RejectReason = ""
    //         // 清空审核时间
    //         row.ReviewedAt = sql.NullTime{Valid: false}
    //         if err := l.svcCtx.MerchantsModel.Update(l.ctx, row); err != nil {
    //             resp.StatusCode = -3
    //             resp.StatusMsg = "re-apply failed"
    //             return resp, nil
    //         }
    //         // 重新入队审核（幂等可重复投递）
    //         _ = mq.PublishMerchantReviewEvent(l.svcCtx, mq.MerchantReviewEvent{
    //             ApplicationID: row.Id,
    //             UserID:        in.GetUserId(),
    //             Application: mq.MerchantApplicationPayload{
    //                 ShopName:     shop,
    //                 ContactName:  contact,
    //                 ContactPhone: phone,
    //                 Address:      addr,
    //                 Description:  desc,
    //             },
    //         })
    //         resp.ApplicationId = row.Id
    //         resp.ApplicationStatus = "PENDING"
    //         return resp, nil
    //     default: // PENDING / ESCALATED 等待中，确保消息在队列中，再返回当前状态
    //         _ = mq.PublishMerchantReviewEvent(l.svcCtx, mq.MerchantReviewEvent{
    //             ApplicationID: row.Id,
    //             UserID:        in.GetUserId(),
    //             Application: mq.MerchantApplicationPayload{
    //                 ShopName:     shop,
    //                 ContactName:  contact,
    //                 ContactPhone: phone,
    //                 Address:      addr,
    //                 Description:  desc,
    //             },
    //         })
    //         resp.ApplicationId = row.Id
    //         resp.ApplicationStatus = row.Status
    //         return resp, nil
    //     }
    // }

    // Prefer DTM commit-and-submit when configured
    if l.svcCtx.Config.DtmConf.Server != "" && l.svcCtx.Config.DtmConf.BusiURL != "" {
        // Use DTM to generate a unique, random GID to avoid collisions
        gid := dtmcli.MustGenGid(l.svcCtx.Config.DtmConf.Server)
        // Keep body minimal; publish handler will fetch application_id by user_id
        body, _ := json.Marshal(mq.MerchantPublishPayload{
            UserID: in.GetUserId(),
            Application: mq.MerchantApplicationPayload{
                ShopName:     shop,
                ContactName:  contact,
                ContactPhone: phone,
                Address:      addr,
                Description:  desc,
            },
        })
        msg := dtmcli.NewMsg(l.svcCtx.Config.DtmConf.Server, gid).
            Add(l.svcCtx.Config.DtmConf.BusiURL+"/dtm/merchant/publish", body)
        qp := l.svcCtx.Config.DtmConf.BusiURL + "/dtm/merchant/query?user_id=" + strconv.FormatInt(in.GetUserId(), 10)
        rawdb, _ := l.svcCtx.MysqlConn.RawDB()
        if err := msg.DoAndSubmitDB(qp, rawdb, func(tx *sql.Tx) error {
            // outbox 插入
            q := "INSERT INTO merchants (user_id, shop_name, contact_name, contact_phone, address, description, status, reject_reason, reviewed_at) VALUES (?,?,?,?,?,?, 'PENDING','', NULL)"
            _, err := tx.ExecContext(l.ctx, q, uint64(in.GetUserId()), shop, contact, phone, addr, sql.NullString{String: desc, Valid: desc != ""})
            if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
                l.Logger.Error("幂等插入：", err)
                return err
            }
            return nil
        }); err != nil {
            resp.StatusCode = -3
            resp.StatusMsg = "apply failed"
            return resp, nil
        }
        // Return current status
        if row, err := l.svcCtx.MerchantsModel.FindOneByUserId(l.ctx, uint64(in.GetUserId())); err == nil && row != nil {
            resp.ApplicationId = row.Id
            resp.ApplicationStatus = row.Status
        } else {
            resp.ApplicationStatus = "PENDING"
        }
        return resp, nil
    }

    // Fallback: insert + direct publish (best-effort)
    rec := &userdal.Merchants{
        UserId:       uint64(in.GetUserId()),
        ShopName:     shop,
        ContactName:  contact,
        ContactPhone: phone,
        Address:      addr,
        Description:  sql.NullString{String: desc, Valid: desc != ""},
        Status:       "PENDING",
        RejectReason: "",
    }
    res, err := l.svcCtx.MerchantsModel.Insert(l.ctx, rec)
    if err != nil {
        if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
            if row, e2 := l.svcCtx.MerchantsModel.FindOneByUserId(l.ctx, uint64(in.GetUserId())); e2 == nil && row != nil {
                switch strings.ToUpper(row.Status) {
                case "APPROVED":
                    resp.ApplicationId = row.Id
                    resp.ApplicationStatus = row.Status
                    return resp, nil
                case "REJECTED":
                    // 驳回后重复提交，视为重新提交
                    row.ShopName = shop
                    row.ContactName = contact
                    row.ContactPhone = phone
                    row.Address = addr
                    row.Description = sql.NullString{String: desc, Valid: desc != ""}
                    row.Status = "PENDING"
                    row.RejectReason = ""
                    row.ReviewedAt = sql.NullTime{Valid: false}
                    if uerr := l.svcCtx.MerchantsModel.Update(l.ctx, row); uerr != nil {
                        resp.StatusCode = -3
                        resp.StatusMsg = "re-apply failed"
                        return resp, nil
                    }
                    _ = mq.PublishMerchantReviewEvent(l.svcCtx, mq.MerchantReviewEvent{
                        ApplicationID: row.Id,
                        UserID:        in.GetUserId(),
                        Application: mq.MerchantApplicationPayload{
                            ShopName:     shop,
                            ContactName:  contact,
                            ContactPhone: phone,
                            Address:      addr,
                            Description:  desc,
                        },
                    })
                    resp.ApplicationId = row.Id
                    resp.ApplicationStatus = "PENDING"
                    return resp, nil
                default: // PENDING/ESCALATED: 确保消息入队
                    _ = mq.PublishMerchantReviewEvent(l.svcCtx, mq.MerchantReviewEvent{
                        ApplicationID: row.Id,
                        UserID:        in.GetUserId(),
                        Application: mq.MerchantApplicationPayload{
                            ShopName:     shop,
                            ContactName:  contact,
                            ContactPhone: phone,
                            Address:      addr,
                            Description:  desc,
                        },
                    })
                    resp.ApplicationId = row.Id
                    resp.ApplicationStatus = row.Status
                    return resp, nil
                }
            }
        }
        resp.StatusCode = -3
        resp.StatusMsg = "apply failed"
        return resp, nil
    }
    id, _ := res.LastInsertId()
    if id <= 0 {
        if row, err := l.svcCtx.MerchantsModel.FindOneByUserId(l.ctx, uint64(in.GetUserId())); err == nil && row != nil {
            id = row.Id
        }
    }
    resp.ApplicationId = id
    resp.ApplicationStatus = "PENDING"
    _ = mq.PublishMerchantReviewEvent(l.svcCtx, mq.MerchantReviewEvent{ApplicationID: id, UserID: in.GetUserId(), Application: struct {
        ShopName string `json:"shop_name"`
        ContactName string `json:"contact_name"`
        ContactPhone string `json:"contact_phone"`
        Address string `json:"address"`
        Description string `json:"description"`
    }{ShopName: shop, ContactName: contact, ContactPhone: phone, Address: addr, Description: desc}})

    _ = time.Now() // keep import time
    return resp, nil
}
