package realname

// 与 users.real_name_status 一致
const (
	UserRealNameNone       int16 = 0
	UserRealNameVerified   int16 = 1
	UserRealNameInProgress int16 = 2
	UserRealNameFailed     int16 = 3
)

// user_real_name_auth.auth_status
const (
	AuthPendingThirdParty int16 = 10
	AuthThirdPartyPass    int16 = 11
	AuthThirdPartyFail    int16 = 12
	AuthPendingManual     int16 = 20
	AuthManualPass        int16 = 21
	AuthManualReject      int16 = 22
	AuthCancelled         int16 = 30
)

// 证件类型 users.real_name_type / auth.cert_type
const (
	CertTypePersonal   int16 = 1
	CertTypeEnterprise int16 = 2
)
