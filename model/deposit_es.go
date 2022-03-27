package model

import (
	"errors"
	"finance/contrib/helper"
	"fmt"
	"github.com/olivere/elastic/v7"
)

//ES查询转账记录
func EsSearch(index, sortField string, page, pageSize int, fields []string,
	param map[string]interface{}, rangeParam map[string][]interface{}, aggField map[string]string) (int64, []*elastic.SearchHit, elastic.Aggregations, error) {

	boolQuery := elastic.NewBoolQuery()
	terms := make([]elastic.Query, 0)
	filters := make([]elastic.Query, 0)

	if len(rangeParam) > 0 {
		for k, v := range rangeParam {
			if v == nil {
				continue
			}

			if len(v) == 2 {

				if v[0] == nil && v[1] == nil {
					continue
				}
				if val, ok := v[0].(string); ok {
					switch val {
					case "gt":
						rg := elastic.NewRangeQuery(k).Gt(v[1])
						filters = append(filters, rg)
					case "gte":
						rg := elastic.NewRangeQuery(k).Gte(v[1])
						filters = append(filters, rg)
					case "lt":
						rg := elastic.NewRangeQuery(k).Lt(v[1])
						filters = append(filters, rg)
					case "lte":
						rg := elastic.NewRangeQuery(k).Lte(v[1])
						filters = append(filters, rg)
					}
					continue
				}

				rg := elastic.NewRangeQuery(k).Gte(v[0]).Lte(v[1])
				if v[0] == nil {
					rg.IncludeLower(false)
				}

				if v[1] == nil {
					rg.IncludeUpper(false)
				}

				filters = append(filters, rg)
			}
		}
	}

	if len(param) > 0 {
		for k, v := range param {
			if v == nil {
				continue
			}

			if vv, ok := v.([]interface{}); ok {
				filters = append(filters, elastic.NewTermsQuery(k, vv...))
				continue
			}

			terms = append(terms, elastic.NewTermQuery(k, v))
		}
	}

	boolQuery.Filter(filters...)
	boolQuery.Must(terms...)
	fsc := elastic.NewFetchSourceContext(true).Include(fields...)
	offset := (page - 1) * pageSize
	//打印es查询json
	esService := meta.ES.Search().FetchSourceContext(fsc).Query(boolQuery).From(offset).Size(pageSize).TrackTotalHits(true).Sort(sortField, false)

	// 聚合条件
	if len(aggField) > 0 {
		for k, v := range aggField {
			esService = esService.Aggregation(k, elastic.NewSumAggregation().Field(v))
		}
	}

	resOrder, err := esService.Index(index).Do(ctx)
	if err != nil {
		fmt.Println(err)
		return 0, nil, nil, pushLog(err, helper.ESErr)
	}

	if resOrder.Status != 0 || resOrder.Hits.TotalHits.Value <= int64(offset) {
		return resOrder.Hits.TotalHits.Value, nil, nil, nil
	}

	return resOrder.Hits.TotalHits.Value, resOrder.Hits.Hits, resOrder.Aggregations, nil
}

//ES查询存款记录
func DepositESQuery(index, sortField string, page, pageSize int,
	param map[string]interface{}, rangeParam map[string][]interface{}, aggField map[string]string) (FDepositData, error) {

	param["prefix"] = meta.Prefix
	data := FDepositData{Agg: map[string]string{}}
	total, esData, aggData, err := EsSearch(index, sortField, page, pageSize, depositFields, param, rangeParam, aggField)
	if err != nil {
		return data, pushLog(err, helper.ESErr)
	}

	for k, v := range aggField {
		amount, _ := aggData.Sum(k)
		if amount != nil {
			data.Agg[v] = fmt.Sprintf("%.4f", *amount.Value)
		}
	}

	data.T = total
	for _, v := range esData {

		deposit := Deposit{}
		deposit.ID = v.Id
		err = cjson.Unmarshal(v.Source, &deposit)
		if err != nil {
			return data, errors.New(helper.FormatErr)
		}
		data.D = append(data.D, deposit)
	}

	return data, nil
}
