package model

import (
	"bitbucket.org/nwf2013/schema"
	"github.com/valyala/fastjson"
)

type rpcResult struct {
	Err string `json:"err"`
	Res string `json:"res"`
}

/*
https://github.com/francoispqt/gojay
*/

func RpcGetDecode(col string, isHide bool, ids []string) ([]rpcResult, error) {

	var res []schema.Dec_t
	for _, v := range ids {
		recs := schema.Dec_t{
			Field: col,
			Hide:  isHide,
			ID:    v,
		}
		res = append(res, recs)
	}

	record, err := rpcGet(res)
	return record, err
}

func rpcGet(data []schema.Dec_t) ([]rpcResult, error) {

	var p fastjson.Parser
	var results []rpcResult

	res, err := meta.Grpc.Call("Decrypt", data)
	if err != nil {
		return results, err
	}

	vv, err := p.ParseBytes(res.([]byte))
	if err != nil {
		return results, err
	}

	value, err := vv.Array()
	if err != nil {
		return results, err
	}

	for _, val := range value {
		r := rpcResult{
			Err: string(val.GetStringBytes("err")),
			Res: string(val.GetStringBytes("res")),
		}
		results = append(results, r)
	}

	return results, nil
}
