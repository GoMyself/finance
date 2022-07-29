package model

import (
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"time"
)

// 发送站内信
func messageSend(msgID, title, content, sendName, prefix string, isTop, isVip, ty int, names []string) error {

	record := g.Record{
		"message_id": msgID,
		"title":      title,
		"content":    content,
		"send_name":  sendName,
		"prefix":     prefix,
		"is_top":     isTop,
		"is_vip":     isVip,
		"is_read":    0,
		"is_delete":  0,
		"send_at":    time.Now().Unix(),
		"ty":         ty,
	}
	var records []g.Record
	for _, v := range names {
		ts := time.Now()
		record["ts"] = ts.UnixMicro()
		record["username"] = v
		records = append(records, record)
	}

	query, _, _ := dialect.Insert("messages").Rows(records).ToSQL()
	fmt.Println(query)
	_, err := meta.MerchantTD.Exec(query)
	if err != nil {
		fmt.Println("insert messages = ", err.Error(), records)
		return pushLog(err, helper.DBErr)
	}

	return nil
}
