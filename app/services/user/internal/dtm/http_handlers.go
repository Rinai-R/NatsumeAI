package dtmhandlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"NatsumeAI/app/services/user/internal/mq"
	"NatsumeAI/app/services/user/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// Register registers DTM HTTP handlers for user service.
// - /dtm/merchant/publish: publish merchant review event to Kafka
// - /dtm/merchant/query:   query-prepared, returns SUCCESS when row inserted
func Register(mux *http.ServeMux, sc *svc.ServiceContext) {
    mux.HandleFunc("/dtm/merchant/publish", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("mq请求来了")
        if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
        body, _ := io.ReadAll(r.Body)
        var payload mq.MerchantPublishPayload
        if err := json.Unmarshal(body, &payload); err != nil {
            // dtm may wrap in base64
            var s string
            if e2 := json.Unmarshal(body, &s); e2 == nil && s != "" {
                if raw, e3 := base64.StdEncoding.DecodeString(s); e3 == nil {
                    _ = json.Unmarshal(raw, &payload)
                }
            }
        }
        if payload.UserID <= 0 {
            w.WriteHeader(http.StatusBadRequest)
            _, _ = w.Write([]byte("INVALID"))
            return
        }
        // fetch inserted row to get application_id
        row, err := sc.MerchantsModel.FindOneByUserId(r.Context(), uint64(payload.UserID))
        if err != nil || row == nil || row.Id <= 0 {
            w.WriteHeader(http.StatusInternalServerError)
            _, _ = w.Write([]byte("FAILURE"))
            return
        }
        evt := mq.MerchantReviewEvent{
            ApplicationID: row.Id,
            UserID:        payload.UserID,
            Application:   payload.Application,
        }
        if err := mq.PublishMerchantReviewEvent(sc, evt); err != nil {
            logx.WithContext(r.Context()).Errorf("publish merchant event failed: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            _, _ = w.Write([]byte("FAILURE"))
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("SUCCESS"))
    })

    mux.HandleFunc("/dtm/merchant/query", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("请求来了")
        q := r.URL.Query()
        userIDStr := q.Get("user_id")
        if userIDStr == "" { userIDStr = q.Get("gid") }
        uid, _ := strconv.ParseInt(userIDStr, 10, 64)
        if uid <= 0 { w.WriteHeader(http.StatusOK); _, _ = w.Write([]byte("FAILURE")); return }
        if _, err := sc.MerchantsModel.FindOneByUserId(r.Context(), uint64(uid)); err == nil {
            w.WriteHeader(http.StatusOK)
            _, _ = w.Write([]byte("SUCCESS"))
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("FAILURE"))
    })
}
