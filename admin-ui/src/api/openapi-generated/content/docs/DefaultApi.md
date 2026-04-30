# DefaultApi

All URIs are relative to *http://localhost*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**getContentList**](#getcontentlist) | **GET** /api/v1/content/list | 获取内容列表|
|[**getProcessStatus**](#getprocessstatus) | **GET** /api/v1/content/status | 获取处理状态|
|[**getUploadUrl**](#getuploadurl) | **POST** /api/v1/content/upload | 获取上传地址|

# **getContentList**
> GetContentList200Response getContentList()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let page: number; // (default to undefined)
let pageSize: number; // (default to undefined)

const { status, data } = await apiInstance.getContentList(
    page,
    pageSize
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **page** | [**number**] |  | defaults to undefined|
| **pageSize** | [**number**] |  | defaults to undefined|


### Return type

**GetContentList200Response**

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

# **getProcessStatus**
> GetProcessStatus200Response getProcessStatus()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let rawContentId: number; // (default to undefined)

const { status, data } = await apiInstance.getProcessStatus(
    rawContentId
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **rawContentId** | [**number**] |  | defaults to undefined|


### Return type

**GetProcessStatus200Response**

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

# **getUploadUrl**
> GetUploadUrl200Response getUploadUrl(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    GetUploadUrlRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: GetUploadUrlRequest; //

const { status, data } = await apiInstance.getUploadUrl(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **GetUploadUrlRequest**|  | |


### Return type

**GetUploadUrl200Response**

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

