# DefaultApi

All URIs are relative to *http://localhost*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**applyAccountCancellation**](#applyaccountcancellation) | **POST** /api/v1/user/cancellation/apply | 申请注销账号（校验后进入冷静期，并吊销 refresh token）|
|[**bindContact**](#bindcontact) | **PUT** /api/v1/user/bind | 绑定邮箱/手机号（已登录，邮箱或手机号二选一 + 验证码；验证码校验成功后进行绑定）|
|[**changePassword**](#changepassword) | **PUT** /api/v1/user/password/change | 修改密码：验证旧密码后修改（新密码需输入两次且一致，且不能与旧密码一致）|
|[**getAccountCancellationConfirm**](#getaccountcancellationconfirm) | **GET** /api/v1/user/cancellation/confirm | 注销二次确认页说明文案|
|[**getAccountCancellationStatus**](#getaccountcancellationstatus) | **GET** /api/v1/user/cancellation/status | 查询注销状态：normal / cooling_off / cancelled|
|[**getRealNameAuditInfo**](#getrealnameauditinfo) | **GET** /api/v1/user/realname/audit | 获取最近一次实名提交的脱敏审核信息|
|[**getRealNameStatus**](#getrealnamestatus) | **GET** /api/v1/user/realname/status | 查询实名认证总状态|
|[**login**](#login) | **POST** /api/v1/user/login | 统一登录：account 或 email/mobile + password（密码登录）或 verify_code（验证码登录/静默注册，先发 verify/send scene&#x3D;login）；微信/Google 走 OAuth|
|[**oauthGoogleCallback**](#oauthgooglecallback) | **GET** /api/v1/user/oauth/google/callback | Google OAuth 直接登录（授权后回调，自动创建/绑定用户并返回 token，无需单独注册）|
|[**oauthWechatCallback**](#oauthwechatcallback) | **GET** /api/v1/user/oauth/wechat/callback | 微信 OAuth 直接登录（授权后回调，自动创建/绑定用户并返回 token，无需单独注册）|
|[**realnameRebind**](#realnamerebind) | **PUT** /api/v1/user/rebind/realname | 实名换绑：旧邮箱/手机不可收码时，凭姓名+身份证号+新号验证码（scene&#x3D;bind）换绑；须已通过个人身份证实名|
|[**rebindContact**](#rebindcontact) | **PUT** /api/v1/user/rebind | 换绑邮箱/手机号（已登录：旧账号验证码通过 + 新账号验证码通过后完成换绑）|
|[**refreshToken**](#refreshtoken) | **POST** /api/v1/user/token/refresh | 刷新 Token：使用 refresh_token 换取新的 access_token（无需重新登录）|
|[**register**](#register) | **POST** /api/v1/user/register | 用户注册（需先发验证码；按邮箱/手机判断是否已注册；密码加盐存储）|
|[**resetPassword**](#resetpassword) | **POST** /api/v1/user/password/reset | 忘记旧密码：邮箱/手机号发送验证码后，验证验证码重置密码（新密码需输入两次且一致，且不能与旧密码一致）|
|[**sendVerifyCode**](#sendverifycode) | **POST** /api/v1/user/verify/send | 发送验证码（邮箱或手机二选一，1分钟有效，1分钟最多3条，超过提示3分钟后重试）|
|[**submitRealName**](#submitrealname) | **POST** /api/v1/user/realname/submit | 提交实名认证（Mock/三方核验 + 可选人工审核）|
|[**updateUserInfo**](#updateuserinfo) | **PUT** /api/v1/user/info | 更新用户基本信息（传 user_id + nickname/avatar；Header 携带 token；仅可修改本人）|
|[**withdrawAccountCancellation**](#withdrawaccountcancellation) | **POST** /api/v1/user/cancellation/withdraw | 撤销注销申请（仅冷静期内）|

# **applyAccountCancellation**
> ApplyAccountCancellation200Response applyAccountCancellation(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    ApplyAccountCancellationRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: ApplyAccountCancellationRequest; //

const { status, data } = await apiInstance.applyAccountCancellation(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **ApplyAccountCancellationRequest**|  | |


### Return type

**ApplyAccountCancellation200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **bindContact**
> BindContact200Response bindContact(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    BindContactRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: BindContactRequest; //

const { status, data } = await apiInstance.bindContact(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **BindContactRequest**|  | |


### Return type

**BindContact200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **changePassword**
> BindContact200Response changePassword(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    ChangePasswordRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: ChangePasswordRequest; //

const { status, data } = await apiInstance.changePassword(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **ChangePasswordRequest**|  | |


### Return type

**BindContact200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAccountCancellationConfirm**
> GetAccountCancellationConfirm200Response getAccountCancellationConfirm()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getAccountCancellationConfirm();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetAccountCancellationConfirm200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getAccountCancellationStatus**
> GetAccountCancellationStatus200Response getAccountCancellationStatus()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getAccountCancellationStatus();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetAccountCancellationStatus200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getRealNameAuditInfo**
> GetRealNameAuditInfo200Response getRealNameAuditInfo()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getRealNameAuditInfo();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetRealNameAuditInfo200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **getRealNameStatus**
> GetRealNameStatus200Response getRealNameStatus()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getRealNameStatus();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetRealNameStatus200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **login**
> Login200Response login(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    LoginRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: LoginRequest; //

const { status, data } = await apiInstance.login(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **LoginRequest**|  | |


### Return type

**Login200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **oauthGoogleCallback**
> Login200Response oauthGoogleCallback()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let code: string; // (default to undefined)
let state: string; // (optional) (default to undefined)

const { status, data } = await apiInstance.oauthGoogleCallback(
    code,
    state
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **code** | [**string**] |  | defaults to undefined|
| **state** | [**string**] |  | (optional) defaults to undefined|


### Return type

**Login200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **oauthWechatCallback**
> Login200Response oauthWechatCallback()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let code: string; // (default to undefined)
let state: string; // (optional) (default to undefined)

const { status, data } = await apiInstance.oauthWechatCallback(
    code,
    state
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **code** | [**string**] |  | defaults to undefined|
| **state** | [**string**] |  | (optional) defaults to undefined|


### Return type

**Login200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **realnameRebind**
> BindContact200Response realnameRebind(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RealnameRebindRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: RealnameRebindRequest; //

const { status, data } = await apiInstance.realnameRebind(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **RealnameRebindRequest**|  | |


### Return type

**BindContact200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **rebindContact**
> BindContact200Response rebindContact(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RebindContactRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: RebindContactRequest; //

const { status, data } = await apiInstance.rebindContact(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **RebindContactRequest**|  | |


### Return type

**BindContact200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **refreshToken**
> RefreshToken200Response refreshToken(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RefreshTokenRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: RefreshTokenRequest; //

const { status, data } = await apiInstance.refreshToken(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **RefreshTokenRequest**|  | |


### Return type

**RefreshToken200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **register**
> Register200Response register(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    RegisterRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: RegisterRequest; //

const { status, data } = await apiInstance.register(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **RegisterRequest**|  | |


### Return type

**Register200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **resetPassword**
> BindContact200Response resetPassword(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    ResetPasswordRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: ResetPasswordRequest; //

const { status, data } = await apiInstance.resetPassword(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **ResetPasswordRequest**|  | |


### Return type

**BindContact200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **sendVerifyCode**
> SendVerifyCode200Response sendVerifyCode(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    SendVerifyCodeRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: SendVerifyCodeRequest; //

const { status, data } = await apiInstance.sendVerifyCode(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SendVerifyCodeRequest**|  | |


### Return type

**SendVerifyCode200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **submitRealName**
> SubmitRealName200Response submitRealName(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    SubmitRealNameRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: SubmitRealNameRequest; //

const { status, data } = await apiInstance.submitRealName(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SubmitRealNameRequest**|  | |


### Return type

**SubmitRealName200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **updateUserInfo**
> BindContact200Response updateUserInfo(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    UpdateUserInfoRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: UpdateUserInfoRequest; //

const { status, data } = await apiInstance.updateUserInfo(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **UpdateUserInfoRequest**|  | |


### Return type

**BindContact200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **withdrawAccountCancellation**
> WithdrawAccountCancellation200Response withdrawAccountCancellation()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.withdrawAccountCancellation();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**WithdrawAccountCancellation200Response**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** |  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

