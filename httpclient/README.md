### Base Usage
1. 返回Response
```
httpclient.Get(url)
httpclient.PostForm(url, params)
httpclient.Post(url, bodyType, body)
httpclient.Do(request)
```

2. 返回String
```
httpclient.GetAsString(string)
httpclient.PostFormAsString(url, params)
httpclient.PostAsString(url, bodyType, body)
httpclient.DoAsString(request)
```

3. 返回Json
```
httpclient.GetAsJson(string, interface)
httpclient.PostFormAsJson(url, params, interface)
httpclient.PostAsJson(url, bodyType, body, interface)
httpclient.DoAsJson(request, interface)
```

### 自定义Client

默认httpclient请求超时1秒，通过自定义client,可以控制http各参数
```
cli := httpclient.NewClient(&http.Client{
        Timeout: time.Second*5,
    }
)
```

### 流式调用，支持request-id和httptrace

```
httpclient.NewRequest().
    Get("http://test.com").
    WithQuery("a", "b").
    WithHeader("a", "b").
    WithRequestID("xxxxx").
    WithTrace(&httptrace.ClientTrace{}).
    ExecuteAsString()

```

### 支持自动熔断
打开自动熔断之后，当一个接口失败满足一定条件之后，对该接口的请求将直接返回error，不再触发http请求，同时client会定时去检测该接口，一旦返回成功，接口将恢复正常请求:
- 以 host+path 用来标识同一个请求,不区分method
- 请求超时或者返回非200标识请求失败

```
cli := httpclient.NewClient(&http.Client{
        Timeout: time.Second*5,
    }
)
cli.CircuitBreaker(true) // false to close

```

当前熔断的默认参数设置如下：
- Timeout = 1000: 请求超时，默认1s
- MaxConcurrentRequests: 同一接口的默认并发数，默认10
- RequestVolumeThreshold: 触发熔断的最小请求量要求，默认20
- SleepWindow: 熔断状态下测试接口可用间隔，默认5s
- ErrorPercentThreshold: 当一个请求有多少比例错误的时候触发熔断，默认50%

修改熔断的默认配置参数（修改对所有httpclient全局有效）:

```
SetCircuitBreakerTimeout(int) //ms
SetCircuitBreakerMaxConcurrentRequests(int)
SetCircuitBreakerRequestVolumeThreshold(int)
SetCircuitBreakerSleepWindow(int) //ms
SetCircuitBreakerErrorPercentThreshold(int)
```

设置单个接口的熔断参数:

```
cli := httpclient.NewClient(&http.Client{
        Timeout: time.Second*5,
    }
)
cli.CircuitBreaker(true, CircuitBreakerConfig{
    Name:                   "api.pdtv.io/test",
    Timeout:                1,
    MaxConcurrentRequests:  1,
    SleepWindow:            1,
    RequestVolumeThreshold: 1,
    ErrorPercentThreshold:  1,
})

```
