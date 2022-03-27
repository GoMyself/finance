package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"finance/contrib/validator"
	"time"

	g "github.com/doug-martin/goqu/v9"
)

type CreditLevel struct {
	ID        string   `db:"id" json:"id"`
	Level     int64    `db:"level" json:"level"`
	CateID    string   `db:"cate_id" json:"cate_id"`
	CateName  string   `db:"-" json:"cate_name"`
	ChannelID string   `db:"channel_id" json:"channel_id"`
	PaymentID string   `db:"payment_id" json:"payment_id"`
	Fmin      int64    `db:"fmin" json:"fmin"`
	Fmax      int64    `db:"fmax" json:"fmax"`
	State     string   `db:"state" json:"state"`
	Members   []string `db:"-" json:"members"`
	CreatedAt int64    `db:"created_at" json:"created_at"`
}

type MemberCreditLevel struct {
	ID            int64  `db:"id" json:"id"`
	CreditLevelID string `db:"credit_level_id" json:"credit_level_id"`
	Uid           int64  `db:"uid" json:"uid"`
	Username      string `db:"username" json:"username"`
	CreatedAt     int64  `db:"created_at" json:"created_at"`
}

// CreditLevelData 财务管理-会员信用等级-列表 response structure
type CreditLevelData struct {
	D []CreditLevel `json:"d"`
	T int64         `json:"t"`
	S uint16        `json:"s"`
}

// MemberCreditLevelData 财务管理-会员信用等级-会员列表 response structure
type MemberCreditLevelData struct {
	D []MemberCreditLevel `json:"d"`
	T int64               `json:"t"`
	S uint16              `json:"s"`
}

func CreditLevelList(level, chanName string, page, pageSize uint16) (CreditLevelData, error) {

	data := CreditLevelData{}

	ex := g.Ex{
		"prefix": meta.Prefix,
	}
	if level != "0" {
		ex["level"] = level
	}

	if chanName != "" {
		var cateID string
		query, _, _ := dialect.From("f_category").Select("id").Where(g.Ex{"name": chanName, "prefix": meta.Prefix}).ToSQL()
		err := meta.MerchantDB.Get(&cateID, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if cateID == "" {
			return data, nil
		}

		ex["cate_id"] = cateID
	}

	if page == 1 {
		query, _, _ := dialect.From("f_credit_level").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_credit_level").
		Select(colCreditLevel...).Where(ex).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	// 拼装查询cate_name的cids和members的ids
	var cids []string
	var ids []string

	for _, v := range data.D {
		cids = append(cids, v.CateID)
		ids = append(ids, v.ID)
	}

	// 查询members
	m, _ := creditMembersByCid(ids)

	// 查询cate name
	cm, _ := cateByIDS(cids)

	for k := range data.D {
		if name, ok := cm[data.D[k].CateID]; ok {
			data.D[k].CateName = name
		}
		if v, ok := m[data.D[k].ID]; ok {
			data.D[k].Members = v
		} else {
			data.D[k].Members = make([]string, 0)
		}
	}

	data.S = pageSize
	return data, nil
}

func CreditLevelInsert(param map[string]string) error {

	// check cate id and channel id
	channel, err := ChanByCateAndChan(param["cate_id"], param["channel_id"])
	if err != nil {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	if len(channel.ID) == 0 {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	// check fmin scope
	fmin, ok := validator.CheckFloatScope(param["fmin"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.TunnelMinLimitErr)
	}
	// check fmax scope
	fmax, ok := validator.CheckFloatScope(param["fmax"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.TunnelMaxLimitErr)
	}

	if !fmin.LessThanOrEqual(fmax) {
		return errors.New(helper.TunnelLimitParamErr)
	}

	var id string
	// 一個level對同一個payment id只能有一條記錄
	ex := g.Ex{
		"level":      param["level"],
		"payment_id": channel.ID,
		"prefix":     meta.Prefix,
	}
	query, _, _ := dialect.From("f_credit_level").Select("id").Where(ex).Limit(1).ToSQL()
	err = meta.MerchantDB.Get(&id, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if len(id) != 0 {
		return errors.New(helper.RecordExistErr)
	}

	record := g.Record{
		"id":         param["id"],
		"level":      param["level"],
		"cate_id":    param["cate_id"],
		"channel_id": param["channel_id"],
		"payment_id": channel.ID,
		"fmin":       param["fmin"],
		"fmax":       param["fmax"],
		"state":      "0",
		"created_at": param["created_at"],
		"prefix":     meta.Prefix,
	}
	query, _, _ = dialect.Insert("f_credit_level").Rows(record).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return nil
}

func CreditLevelUpdate(paymentID string, param map[string]string) error {

	// check cate id and channel id
	channel, err := ChanByID(paymentID)
	if err != nil {
		return err
	}

	if len(channel.ID) == 0 {
		return errors.New(helper.CateIDAndChannelIDErr)
	}

	// check fmin scope
	fmin, ok := validator.CheckFloatScope(param["fmin"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.TunnelMinLimitErr)
	}

	// check fmax scope
	fmax, ok := validator.CheckFloatScope(param["fmax"], channel.Fmin, channel.Fmax)
	if !ok {
		return errors.New(helper.TunnelMaxLimitErr)
	}

	if !fmin.LessThanOrEqual(fmax) {
		return errors.New(helper.TunnelLimitParamErr)
	}

	record := g.Record{
		"fmin": param["fmin"],
		"fmax": param["fmax"],
	}
	query, _, _ := dialect.Update("f_credit_level").Set(record).Where(g.Ex{"id": param["id"]}).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return err
}

func CreditLevelUpdateState(id, state, paymentID string) error {

	// 上级通道关闭的时候不能开启
	if state == "1" {
		cate, err := ChanByID(paymentID)
		if err != nil {
			return err
		}

		if len(cate.ID) == 0 {
			return errors.New(helper.CateNotExist)
		}

		if cate.State == "0" {
			return errors.New(helper.ParentChannelClosed)
		}
	}

	query, _, _ := dialect.Update("f_credit_level").Set(g.Record{"state": state}).Where(g.Ex{"id": id}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	// todo cache

	return nil
}

func MemberCreditLevelList(clID string, page, pageSize uint16, users []string) (MemberCreditLevelData, error) {

	data := MemberCreditLevelData{}
	ex := g.Ex{
		"credit_level_id": clID,
		"prefix":          meta.Prefix,
	}

	if len(users) > 0 {
		ex["username"] = users
	}

	if page == 1 {
		query, _, _ := dialect.From("f_member_credit_level").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil && err != sql.ErrNoRows {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_member_credit_level").
		Select(colMemberCreditLevel...).Where(ex).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	data.S = pageSize
	return data, nil
}

func MemberCreditLevelInsert(clID string, t time.Time, users []string) error {

	m, err := membersUidByUsername(users)
	if err != nil {
		return err
	}

	if len(m) != len(users) {
		return errors.New(helper.UsernameErr)
	}

	// 过滤掉已经存在的users
	users, err = creditMembersInsertFilter(clID, users)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return nil
	}

	var record []g.Record
	for _, v := range users {
		record = append(record, g.Record{
			"uid":             m[v],
			"username":        v,
			"created_at":      t.Unix(),
			"credit_level_id": clID,
			"prefix":          meta.Prefix,
		})
	}

	query, _, _ := dialect.Insert("f_member_credit_level").Rows(record).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return err
}

func MemberCreditLevelDelete(ids []string) error {

	query, _, _ := dialect.Delete("f_member_credit_level").Where(g.Ex{"id": ids, "prefix": meta.Prefix}).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	return err
}

func CreditLevelByID(id string) (CreditLevel, error) {

	creditLevel := CreditLevel{}
	query, _, _ := dialect.From("f_credit_level").Select(colCreditLevel...).Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&creditLevel, query)
	if err != nil && err != sql.ErrNoRows {
		return creditLevel, pushLog(err, helper.DBErr)
	}

	return creditLevel, nil
}

func creditMembersByCid(cid []string) (map[string][]string, error) {

	var mcl []MemberCreditLevel
	ex := g.Ex{"credit_level_id": cid, "prefix": meta.Prefix}
	query, _, _ := dialect.From("f_member_credit_level").Select(colMemberCreditLevel...).Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&mcl, query)
	if err != nil && err != sql.ErrNoRows {
		return nil, pushLog(err, helper.DBErr)
	}

	res := map[string][]string{}
	for _, v := range mcl {
		if _, ok := res[v.CreditLevelID]; !ok {
			res[v.CreditLevelID] = make([]string, 0)
		}
		res[v.CreditLevelID] = append(res[v.CreditLevelID], v.Username)
	}

	return res, nil
}

// 过滤掉已经存在的会员
func creditMembersInsertFilter(cid string, users []string) ([]string, error) {

	var mcl []string
	ex := g.Ex{"credit_level_id": cid, "username": users, "prefix": meta.Prefix}
	query, _, _ := dialect.From("f_member_credit_level").Select("username").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&mcl, query)
	if err != nil && err != sql.ErrNoRows {
		return nil, pushLog(err, helper.DBErr)
	}

	var res []string
	m := make(map[string]bool, len(mcl))
	for _, v := range mcl {
		m[v] = true
	}

	for _, v := range users {
		if _, ok := m[v]; !ok {
			res = append(res, v)
		}
	}

	return res, nil
}

func CreditMemberLevelByID(id string) (MemberCreditLevel, error) {

	creditLevel := MemberCreditLevel{}
	query, _, _ := dialect.From("f_member_credit_level").Select(colMemberCreditLevel...).Where(g.Ex{"id": id}).ToSQL()
	err := meta.MerchantDB.Get(&creditLevel, query)
	if err != nil && err != sql.ErrNoRows {
		return creditLevel, pushLog(err, helper.DBErr)
	}

	return creditLevel, nil
}
