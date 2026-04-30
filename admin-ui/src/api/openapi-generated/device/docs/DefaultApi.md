# DefaultApi

All URIs are relative to *http://localhost*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**bindDevice**](#binddevice) | **POST** /api/v1/device/bind | 绑定设备|
|[**getDeviceList**](#getdevicelist) | **GET** /api/v1/device/list | 获取设备列表|
|[**getPlaylist**](#getplaylist) | **GET** /api/v1/device/playlist | 获取播放列表|
|[**heartbeat**](#heartbeat) | **POST** /api/v1/device/heartbeat | 设备心跳|

# **bindDevice**
> BindDevice200Response bindDevice(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    BindDeviceRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: BindDeviceRequest; //

const { status, data } = await apiInstance.bindDevice(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **BindDeviceRequest**|  | |


### Return type

**BindDevice200Response**

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

# **getDeviceList**
> GetDeviceList200Response getDeviceList()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

const { status, data } = await apiInstance.getDeviceList();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**GetDeviceList200Response**

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

# **getPlaylist**
> GetPlaylist200Response getPlaylist()


### Example

```typescript
import {
    DefaultApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let deviceId: number; // (default to undefined)

const { status, data } = await apiInstance.getPlaylist(
    deviceId
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **deviceId** | [**number**] |  | defaults to undefined|


### Return type

**GetPlaylist200Response**

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

# **heartbeat**
> object heartbeat(body)


### Example

```typescript
import {
    DefaultApi,
    Configuration,
    HeartbeatRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new DefaultApi(configuration);

let body: HeartbeatRequest; //

const { status, data } = await apiInstance.heartbeat(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **HeartbeatRequest**|  | |


### Return type

**object**

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

