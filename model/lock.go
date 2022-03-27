package model

import (
	"database/sql"
	"errors"
	"finance/contrib/helper"
	"fmt"
	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

type MemberLock struct {
	ID          string `db:"id" json:"id"`
	UID         string `db:"uid" json:"uid"`
	Username    string `db:"username" json:"username"`
	State       string `db:"state" json:"state"`
	CreatedAt   int64  `db:"created_at" json:"created_at"`
	Comment     string `db:"comment" json:"comment"`
	UpdatedAt   int64  `db:"updated_at" json:"updated_at"`
	UpdatedUID  string `db:"updated_uid" json:"updated_uid"`
	UpdatedName string `db:"updated_name" json:"updated_name"`
	CreatedUID  string `db:"created_uid" json:"created_uid"`
	CreatedName string `db:"created_name" json:"created_name"`
	Level       string `db:"-" json:"level"`
}

type MemberLockData struct {
	D []MemberLock `json:"d"`
	T int64        `json:"t"`
	S uint16       `json:"s"`
}

func LockList(username, lockName, start, end string, page, pageSize uint16) (MemberLockData, error) {

	data := MemberLockData{}

	ex := g.Ex{"state": "1", "prefix": meta.Prefix}

	if username != "" {
		ex["username"] = username
	}

	if lockName != "" {
		ex["created_name"] = lockName
	}

	if start != "" && end != "" {

		startAt, err := helper.TimeToLoc(start, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		endAt, err := helper.TimeToLoc(end, loc)
		if err != nil {
			return data, errors.New(helper.DateTimeErr)
		}

		if startAt >= endAt {
			return data, errors.New(helper.QueryTimeRangeErr)
		}

		ex["created_at"] = g.Op{"between": exp.NewRangeVal(startAt, endAt)}
	}

	if page == 1 {
		query, _, _ := dialect.From("f_member_lock").Select(g.COUNT(1)).Where(ex).ToSQL()
		err := meta.MerchantDB.Get(&data.T, query)
		if err != nil {
			return data, pushLog(err, helper.DBErr)
		}

		if data.T == 0 {
			return data, nil
		}
	}

	offset := (page - 1) * pageSize
	query, _, _ := dialect.From("f_member_lock").
		Select(colMemberLock...).Where(ex).Offset(uint(offset)).Limit(uint(pageSize)).ToSQL()
	err := meta.MerchantDB.Select(&data.D, query)
	if err != nil {
		return data, pushLog(err, helper.DBErr)
	}

	// 获取会员等级
	var uids []string
	for _, v := range data.D {
		uids = append(uids, v.UID)
	}

	levels, _ := MemberLevelByUID(uids)
	for k := range data.D {
		level, ok := levels[data.D[k].UID]
		if ok {
			data.D[k].Level = fmt.Sprintf("VIP%d", level-1)
		}
	}

	data.S = pageSize
	return data, nil
}

func LockInsert(param map[string]string) error {

	err := lockMemberCheck(param["username"])
	if err != nil {
		return err
	}

	record := g.Record{
		"id":           param["id"],
		"uid":          param["uid"],
		"username":     param["username"],
		"state":        param["state"],
		"comment":      param["comment"],
		"created_at":   param["created_at"],
		"created_uid":  param["created_uid"],
		"created_name": param["created_name"],
		"updated_at":   "0",
		"updated_uid":  "0",
		"updated_name": "",
		"prefix":       meta.Prefix,
	}
	query, _, _ := dialect.Insert("f_member_lock").Rows(record).ToSQL()
	_, err = meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	key := fmt.Sprintf("DL:%s", param["uid"])
	err = meta.MerchantRedis.Set(ctx, key, "1", 0).Err()
	if err != nil {
		_ = pushLog(err, helper.RedisErr)
	}

	return nil
}

func LockUpdateState(uid string, param map[string]string) error {

	record := g.Record{
		"state":        param["state"],
		"updated_uid":  param["updated_uid"],
		"updated_name": param["updated_name"],
		"updated_at":   param["updated_at"],
	}
	ex := g.Ex{
		"id":    param["id"],
		"state": "1",
	}
	query, _, _ := dialect.Update("f_member_lock").Set(record).Where(ex).Limit(1).ToSQL()
	_, err := meta.MerchantDB.Exec(query)
	if err != nil {
		return pushLog(err, helper.DBErr)
	}

	key := fmt.Sprintf("DL:%s", uid)
	err = meta.MerchantRedis.Unlink(ctx, key).Err()
	if err != nil {
		_ = pushLog(err, helper.RedisErr)
	}

	return nil
}

func lockMemberCheck(username string) error {

	var id string
	ex := g.Ex{
		"username": username,
		"state":    "1",
		"prefix":   meta.Prefix,
	}
	query, _, _ := dialect.From("f_member_lock").Select("id").Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&id, query)
	if err != nil && err != sql.ErrNoRows {
		return pushLog(err, helper.DBErr)
	}

	if id == "" {
		return nil
	}

	return errors.New(helper.MemberLockAlready)
}

// 根据给定会员的uid查询会员是否被lock了
func lockMapByUids(uids []string) (map[string]bool, error) {

	var lms []string
	ex := g.Ex{
		"uid":   uids,
		"state": "1",
	}
	query, _, _ := dialect.From("f_member_lock").Select("uid").Where(ex).ToSQL()
	err := meta.MerchantDB.Select(&lms, query)
	if err != nil {
		return nil, pushLog(err, helper.DBErr)
	}

	lm := make(map[string]bool, len(uids))
	// 初始化, 所有的会员都是未被lock的状态
	for _, v := range uids {
		lm[v] = false
	}

	// 修改锁定的会员的lock状态
	for _, v := range lms {
		lm[v] = true
	}

	return lm, nil
}

func LockById(id string) (MemberLock, error) {

	data := MemberLock{}
	ex := g.Ex{"id": id}
	query, _, _ := dialect.From("f_member_lock").Select(colMemberLock...).Where(ex).Limit(1).ToSQL()
	err := meta.MerchantDB.Get(&data, query)
	if err != nil && err != sql.ErrNoRows {
		return data, pushLog(err, helper.DBErr)
	}

	return data, nil
}
