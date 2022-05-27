package model

import (
	"finance/contrib/helper"
	"github.com/olivere/elastic/v7"
	"time"
)

// 发送站内信
func messageSend(msgID, title, subTitle, content, sendName, prefix string, isTop, isVip, ty int, names []string) error {

	data := Message{
		MsgID:    msgID,
		Title:    title,
		SubTitle: subTitle,
		Content:  content,
		IsTop:    isTop,
		IsVip:    isVip,
		IsRead:   0,
		Ty:       ty,
		SendName: sendName,
		SendAt:   time.Now().Unix(),
		Prefix:   prefix,
	}
	bulkRequest := meta.ES.Bulk().Index(meta.EsPrefix + "messages")
	for _, v := range names {
		data.Username = v
		doc := elastic.NewBulkIndexRequest().Id(helper.GenId()).Doc(data)
		bulkRequest = bulkRequest.Add(doc)
	}

	_, err := bulkRequest.Refresh("wait_for").Do(ctx)
	if err != nil {
		return err
	}

	return nil
}
