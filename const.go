package main

const (
	time_layout        = "2006-01-02 15:04:05"
	Index_FIXED_DIALOG = "dialog:%s:%s"
	Index_BotUser      = "uid:%s:%s:bot:%s"
	Index_Profile      = "uid:%s:%s:prof:%s"
	TwUser             = "TU"
	LineUser           = "LU"

	Index_App    = "app:%s"
	Index_TrapLs = "app:%s:trap"

	keyUid          = "uid:%s:%s"
	Index_StatusIds = "uid:%s:%s:ids"
	keyUidFollowers = "uid:%s:%s:followers"
	keyUidFriends   = "uid:%s:%s:frineds"

	Index_TmpId = "tmp:id:%s:%s"

	zkeyTimeStamp       = "timestamp:%s"
	ZKey_TotalTalkCount = "rank:talk_cnt:%s:%s"
	ZKey_Fav            = "rank:fav:%s:%s"
	zkey_Exp            = "rank:exp:%s:%s"
	ZKey_TotalExp       = "rank:total_exp:%s:%s"
	ZKey_Intimacy       = "rank:intimacy:%s:%s"
	ZKey_MaxIntimacy    = "rank:max_intimacy:%s:%s"

	apps_cron_315umi   = "apps:cron:315umi"
	apps_cron_kusoripu = "apps:cron:kusoripu"
	apps_cron_generate = "apps:cron:generate"
	apps_cron_334      = "apps:cron:334"

	apps_t_wait      = "apps:T:wait"
	apps_t_streaming = "apps:T:streaming"
	apps_l_wait      = "apps:L:wait"
	apps_l_streaming = "apps:L:streaming"

	apps_mimicking = "apps:mimicking"
	apps_trapping  = "apps:trap"
	apps_replygen  = "apps:replygen"
	apps_fav       = "apps:fav"

	usersAgrvNG       = "users:agrvNG"
	users_kusoripu_ok = "users:kusoripuOK"
	SET_MsgINFO       = "users:msginfo"
	SET_TLReact       = "users:tl_react"

	SET_TwitterAuth = "auth:twtr"

	default_char    = "海未"
	DEFAULT_STATUS  = "dialog"
	EXP_TALK        = 10.0
	URL_TwLogin     = "/login/twitter/auth"
	URL_CallBack    = "/login/twitter/auth/callback"
	URL_AuthSuccess = "/login/twitter/auth/registered"
)
