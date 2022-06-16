package model

import (
	"fmt"

	"github.com/lucacasonato/mqtt"
)

func Publish(name string, data []byte) error {

	/*
		headers := map[string]string{}
		uri := fmt.Sprintf("http://10.170.0.3:4757/pub?id=%s", name)

		b := helper.Addslashes(string(data))
		_, statusCode, err := helper.HttpDoTimeout([]byte(b), "POST", uri, headers, 5*time.Second)
		if err != nil {
			fmt.Println("err = ", err.Error())
			fmt.Println("statusCode = ", statusCode)
		}
	*/
	err := meta.MerchantMqtt.Publish(ctx, name, data, mqtt.AtLeastOnce)
	if err != nil {
		fmt.Printf("merchantNats.Publish %s = %s\n", name, err.Error())

	}

	return err
}
