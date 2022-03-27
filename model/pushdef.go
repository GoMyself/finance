package model

const (
	// 提款审核
	withdrawReviewFmt = `{
  "cn": {
    "title": "提款审核",
    "content": "会员 %s，申请提款 %s KVND，请尽快审核。",
    "url": "/risk/withdrawalReview"
  },
  "en": {
    "title": "Xét duyệt rút tiền",
    "content": "Thành viên %s, đăng ký rút tiền %s KVND， vui lòng nhanh chóng xét duyệt.",
    "url": "/risk/withdrawalReview"
  },
  "vn": {
     "title": "Xét duyệt rút tiền",
    "content": "Thành viên %s, đăng ký rút tiền %s KVND， vui lòng nhanh chóng xét duyệt.",
    "url": "/risk/withdrawalReview"
  }
}`
	// 补单审核
	manualReviewFmt = `{
  "cn": {
    "title": "财务补单审核",
    "content": "用户 %s，发起财务补单，%s 补单 %s KVND，请尽快审核。",
    "url": "/fin/DepositManagement?name=repOrderReview"
  },
  "en": {
    "title": "Xét duyệt tài vụ bù đơn",
    "content": "Người dùng %s, phát tài vụ bù đơn, %s bù đơn %s KVND, vui lòng nhanh chóng xét duyệt.",
    "url": "/fin/DepositManagement?name=repOrderReview"
  },
  "vn": {
     "title": "Xét duyệt tài vụ bù đơn",
    "content": "Người dùng %s, phát tài vụ bù đơn, %s bù đơn %s KVND, vui lòng nhanh chóng xét duyệt.",
    "url": "/fin/DepositManagement?name=repOrderReview"
  }
}`
	// 手动下分审核
	downgradeReviewFmt = `{
  "cn": {
    "title": "手动下分审核",
    "content": "用户 %s，发起手动下分，%s 下分 %s KVND，请尽快审核。",
    "url": "/fin/ManualUpAndDown?name=review_list"
  },
  "en": {
    "title": "Xét duyệt hạ điểm thủ công",
    "content": "Người dùng %s, Phát hạ điểm thủ công, %s hạ điểm %s KVND, vui lòng nhanh chóng xét duyệt.",
    "url": "/fin/ManualUpAndDown?name=review_list"
  },
  "vn": {
    "title": "Xét duyệt hạ điểm thủ công",
    "content": "Người dùng %s, Phát hạ điểm thủ công, %s hạ điểm %s KVND, vui lòng nhanh chóng xét duyệt.",
    "url": "/fin/ManualUpAndDown?name=review_list"
  }
}`
	// 代理提款审核
	agencyWithdrawReviewFmt = `{
  "cn": {
    "title": "代理提款审核",
    "content": "用户 %s，发起代理提款，%s 提款 %d KVD，请尽快审核。",
    "url": "/risk/withdrawalReview"
  },
  "en": {
    "title": "Xét  duyệt đại lý rút tiền",
    "content": "Người dùng %s, phát đại lý rút tiền, %s rút tiền %d KVD, vui lòng nhanh chóng xét duyệt.",
    "url": "/risk/withdrawalReview"
  },
  "vn": {
    "title": "Xét  duyệt đại lý rút tiền",
    "content": "Người dùng %s, phát đại lý rút tiền, %s rút tiền %d KVD, vui lòng nhanh chóng xét duyệt.",
    "url": "/risk/withdrawalReview"
  }
}`
)
