package mq

type ProductCategoryRow struct {
	ID        int64  `json:"id,string"`
	ProductID int64  `json:"product_id,string"`
	Category  string `json:"category"`
}

type CanalProductCategoryMessage struct {
	Data      []ProductCategoryRow `json:"data"`
	Database  string               `json:"database"`
	Es        int64                `json:"es"`
	Gtid      string               `json:"gtid"`
	ID        int64                `json:"id"`
	IsDdl     bool                 `json:"isDdl"`
	MysqlType map[string]string    `json:"mysqlType"`
	Old       []map[string]any     `json:"old"`
	PkNames   []string             `json:"pkNames"`
	SQL       string               `json:"sql"`
	SQLType   map[string]int       `json:"sqlType"`
	Table     string               `json:"table"`
	Ts        int64                `json:"ts"`
	Type      string               `json:"type"`
}

type ProductRow struct {
	ID          int64  `json:"id,string"`
	MerchantID  int64  `json:"merchant_id,string"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Picture     string `json:"picture"`
	Price       int64  `json:"price,string"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CanalMessageProducts struct {
	Data      []ProductRow      `json:"data"`
	Database  string            `json:"database"`
	Es        int64             `json:"es"`
	Gtid      string            `json:"gtid"`
	ID        int64             `json:"id"`
	IsDdl     bool              `json:"isDdl"`
	MysqlType map[string]string `json:"mysqlType"`
	Old       []map[string]any  `json:"old"`
	PkNames   []string          `json:"pkNames"`
	SQL       string            `json:"sql"`
	SQLType   map[string]int    `json:"sqlType"`
	Table     string            `json:"table"`
	Ts        int64             `json:"ts"`
	Type      string            `json:"type"`
}
