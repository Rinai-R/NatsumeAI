package dtmhandlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"NatsumeAI/app/services/order/internal/mq"
	"NatsumeAI/app/services/order/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// Register registers DTM HTTP handlers on the given mux.
func Register(mux *http.ServeMux, sc *svc.ServiceContext) {
    // Action: publish checkout event to Kafka
    mux.HandleFunc("/dtm/checkout/publish", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        body, _ := io.ReadAll(r.Body)
        var evt mq.CheckoutEvent
        if err := json.Unmarshal(body, &evt); err != nil {
            // dtm 直接给的 base64 编码的json
            var s string
            if e2 := json.Unmarshal(body, &s); e2 == nil && s != "" {
                if raw, e3 := base64.StdEncoding.DecodeString(s); e3 == nil {
                    if e4 := json.Unmarshal(raw, &evt); e4 == nil {
                        goto PUBLISH
                    }
                }
            }
            logx.WithContext(r.Context()).Errorf("dtm publish decode failed: %v body=%s", err, string(body))
            w.WriteHeader(http.StatusBadRequest)
            _, _ = w.Write([]byte("INVALID"))
            return
        }
    PUBLISH:
        if err := mq.PublishCheckoutEvent(sc, evt); err != nil {
            logx.WithContext(r.Context()).Errorf("publish checkout event failed: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            _, _ = w.Write([]byte("FAILURE"))
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("SUCCESS"))
    })

    // QueryPrepared: DTM probes whether local transaction (insert preorder) succeeded.
    mux.HandleFunc("/dtm/checkout/query", func(w http.ResponseWriter, r *http.Request) {
        fmt.Println("query事件来了")
        // dtm passes gid etc via query; we only need to return SUCCESS when preorder exists.
        q := r.URL.Query()
        preorderIDStr := q.Get("preorder_id")
        // optional: allow gid as preorder id as well
        if preorderIDStr == "" {
            preorderIDStr = q.Get("gid")
        }
        if preorderIDStr == "" {
            w.WriteHeader(http.StatusBadRequest)
            _, _ = w.Write([]byte("FAILURE"))
            return
        }
        // We don't parse to int here; existence check requires numeric id. Best-effort parse.
        // If parse fails, treat as failure.
        var preorderID int64
        if _, err := fmt.Sscan(preorderIDStr, &preorderID); err != nil || preorderID <= 0 {
            w.WriteHeader(http.StatusOK)
            _, _ = w.Write([]byte("FAILURE"))
            return
        }
        if _, err := sc.Preorder.FindOne(r.Context(), preorderID); err == nil {
            w.WriteHeader(http.StatusOK)
            _, _ = w.Write([]byte("SUCCESS"))
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("FAILURE"))
    })
}
