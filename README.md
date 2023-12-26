# Structured logging in Golang echo

### Slog

The `slog` package has its origins in [this GitHub discussion opened by Jonathan Amsterdam](https://github.com/golang/go/discussions/54763) , which later led to the [proposal](https://github.com/golang/go/issues/56345)  describing the exact design of the package. Once finalized, it was released in [Go v1.21](https://tip.golang.org/doc/go1.21)  and now resides at `log/slog`.

### Echo
Echo is a High performance, extensible, minimalist Go web framework.

![LabStack](echo_framework_logo.png)

The Echo project is a powerful and versatile web framework for building scalable and high-performance web applications in the Go programming language. It follows the principles of simplicity, flexibility, and performance to provide developers with an efficient toolkit for building robust web applications.

## Key Features

- **Fast and Lightweight**: Echo is designed for speed and efficiency, ensuring minimal overhead and high performance for handling HTTP requests and responses.
- **Routing**: The framework offers a flexible and intuitive routing system that allows developers to define routes with parameters, query strings, and custom handlers.
- **Middleware Support**: Echo provides extensive middleware support, enabling developers to easily implement cross-cutting concerns such as logging, authentication, error handling, and more.
- **Context-based Request Handling**: With its context-based request handling, Echo offers easy access to request-specific data and parameters, simplifying the development of web applications.


###  Structured Logging
Structured logging means, logs have a defined structure (generally in a json format), this allows debugging through logs more intuitive. Also this enables us to write some ingestion on our logs to perform certain analytics.
Lets us understand by example.
legacy_logger.go
```go
package main  
  
import (  
    "errors"  
    "fmt"    
    "log"
    )  
  
func getFruitByIndex(index int, fruits ...string) (string, error) {  
    if len(fruits) < index || index < 0 {  
       return "", errors.New("not valid index")  
    }  
    return fruits[index], nil  
}  
  
func main() {  
    log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmsgprefix)  
    list := []string{"apple", "orange", "banana", "kivi"}  
    fruit, err := getFruitByIndex(1, list...)  
    if err != nil {  
       log.Printf("Error in fetching fruit. Error: %s", err.Error())  
    } else {  
       fmt.Printf("You will choosed fruit %s\n", fruit)  
    }  
  
    fruit, err = getFruitByIndex(5, list...)  
    if err != nil {  
       log.Printf("Error in fetching fruit. Error: %s", err.Error())  
    } else {  
       fmt.Printf("You will choosed fruit %s\n", fruit)  
    }  
  
}
```
Response for this code is:
```shell
You will choosed fruit orange
2023/12/25 17:44:12 slog_example.go:28: Error in fetching fruit. Error: not valid index
```

The `log.SetFlags()` method  in `log` package, is used to have fine control over the output of log.  We can customise the time format,  prefix and even file name of calling log statement.
The one problem with this logging is that suppose we have an error log  in our system, as we don't have any context of the logged data, or any meta data. We will have to search logs in a huge logs file to find the relevant log and trace it back to find root cause of error. Its like finding needle in a hail-stack.
The second problem will be that we don't have any structure of logs, we can use this information to our anomaly detection system.

#### Log formatting best practices
Their are many practices, but i found out these to be most necessary.
- Use structured JSON logging. Readability is important for logs when debugging or monitoring, JSON is human readable. Also JSON is widely used in APIs, it has many libraries, so it will be easy to process logs.
- Log should have source data related to the context. This include information like function name, file name, and line number of log.
- Include contextual meta fields. To make your log messages more informative and actionable, you need to add contextual fields to your log messages. These fields should help answer key questions about each logged event such as what happened, when, where, why, and the responsible entity. Contextual fields like route path, route method and user agent.
  With JSON schema, we can name the contextual field, this will make them more readable and structured.
- Add correlation id. A log statement represents the state in the current workflow, but a HTTP Api request can have many logs. A correlation id is an id, attached to the HTTP request at the start, this is propagated and used in logs. This will enable us to track the trace of various logs created.
- Implement selective logging. Overuse of anything is bad, JSON schema of logs is good but if we are not careful we will dump useless information in the logs. Some times we might dump sensitive information like User account details in logs, which we must not disclose it.

### Using Slog to have a structured logging system
Slog library has 3 main types.
- `Logger`: the "frontend" for logging with Slog. It provides level methods such as (`Info()` and `Error()`) for recording events of interest.
- `Record`: represents each self-contained log record object created by a `Logger`.
- `Handler`: the "backend" of the Slog package. It is an interface that, once implemented, determines the formatting and destination of each `Record`. Two handlers are included with the `slog` package by default: `TextHandler` and `JSONHandler`.

To have a structured logging system, we will have to use the `context.Context` present in `context` package. The `http.Request` has the `context.Context` field, this can be either client or server. We can use this `context.Context`, in all the functions, this will enable us to pass some information which we can use for logging.
_We should not add any sensitive information to context.Context._
_Avoid sending large data through context.Context._
We can use this `context.Context` and add the correlation id, and other meta information at the starting of the request. We will pass on this to each function and will use this for logging.
The `TextHandler's` and `JSONHandler's` `Handler` does not log information from `context.Context`.

To use `context.Context` we can `struct` embed `Handler` into a new type, and implement the interface.

```go
// ContextHandler is our base context handler, it will handle all requests
type ContextHandler struct {  
    slog.Handler  
}  

// Enabled determines if to log or not log, if it returns true then Handle will log func (ch ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {  
    return ch.Handler.Enabled(ctx,level)  
}
  
// Handle backend for api, this will be used to configure how the logs will be structured
func (ch ContextHandler) Handle(ctx context.Context, r slog.Record) error {  
    r.AddAttrs(ch.addRequestId(ctx)...)  
    return ch.Handler.Handle(ctx, r)  
}  
  
// WithAttrs overriding default implementation otherwise it will call the starting JSON Handler  
func (ch ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {  
    return ContextHandler{ch.Handler.WithAttrs(attrs)}  
}  
  
// WithGroup overriding default implementation otherwise it will call the starting JSON Handler  
func (ch ContextHandler) WithGroup(name string) slog.Handler {  
    return ContextHandler{ch.Handler.WithGroup(name)}  
}  
  
func (ch ContextHandler) addRequestId(ctx context.Context) []slog.Attr {  
    var as []slog.Attr  
    correlation := getDefaultValueFromContext(ctx, "correlation_id")  
    method := getDefaultValueFromContext(ctx, "request_method")  
    path := getDefaultValueFromContext(ctx, "request_path")  
    agent := getDefaultValueFromContext(ctx, "request_user_agent")  
  
    group := slog.Group("meta_information", slog.String("correlation_id", correlation),  
       slog.String("request_method", method),  
       slog.String("request_path", path),  
       slog.String("request_user_agent", agent))  
    as = append(as, group)  
    return as  
}  
  
// getDefaultValueFromContext get default value from contextfunc getDefaultValueFromContext(ctx context.Context, key string) string {  
    value := commonConstants.EmptyString  
    ctxValue := ctx.Value(key)  
    if ctxValue != nil {  
       value = ctxValue.(string)  
    }  
    return value  
}
```
`slog` has a method `NewJSONHandler` which takes 2 arguments, one is the `io.Writer` and another is `slog.HandlerOptions`. `slog.HandlerOptions` has fields `AddSource` which is for to include source information like file name, function name and line number. It has another field for level which is used to set the minimum level of log, any log below this will not be logged.
After creating the `JSONHandler` with `HandlerOptions` we can use struct embedding and create out own `ContextHandler` from `JSONHandler`. Now we can create a new logger and set the default `slog` logger to the out new logger.
```go
opts := slog.HandlerOptions{  
    AddSource: true,  
    Level:     slog.LevelInfo,  
}  
jsonHandler := slog.NewJSONHandler(os.Stdout, &opts)  
ctxHandler := model.ContextHandler{Handler: jsonHandler}  
logger := slog.New(ctxHandler)  
slog.SetDefault(logger) // setting default logger
```

To use slog for structure logging in echo we have various steps:
#### STEPS
#### 1. Add source data
When we defined our logger, we have included the source data in the logger.
```go 
opts := slog.HandlerOptions{  
    AddSource: true,  
    Level:     slog.LevelInfo,  
} 
```

#### 2.Add correlation id
We have created a middleware to add correlation id. This middle ware will be called for before each request.

Attaching middleware before starting route.
```
e.Use(middelware.CorrelationId) 
```
Middleware
```go
func CorrelationId(next echo.HandlerFunc) echo.HandlerFunc {  
    return func(c echo.Context) error {  
       requestId, err := uuid.GenerateUUID()  
       if err != nil {  
          slog.ErrorContext(c.Request().Context(), "Error in generating unique correlation id "+err.Error())  
          // generating a random string of 32  
          requestId = utils.GenerateRandomString(32)  
       }  
       ctx := context.WithValue(c.Request().Context(), "correlation_id", requestId)  
       request := c.Request().Clone(ctx)  
       c.SetRequest(request)  
       return next(c)  
    }  
}
```
#### 3. Add contextual meta data for request
We have created a middleware to add contextual data. This middle ware will be called for before each request.

Attaching middleware before starting route.
```
e.Use(middelware.AddMetaData) 
```
```go
// AddMetaData adding meta-information about the route. Method, Path, User Agent
func AddMetaData(next echo.HandlerFunc) echo.HandlerFunc {  
    return func(c echo.Context) error {  
       path := c.Request().RequestURI  
       method := c.Request().Method  
       userAgent := c.Request().UserAgent()  
       ctx := context.WithValue(c.Request().Context(), "request_path", path)  
       ctx = context.WithValue(ctx, "request_method", method)  
       ctx = context.WithValue(ctx, "request_user_agent", userAgent)  
       request := c.Request().Clone(ctx)  
       c.SetRequest(request)  
       return next(c)  
    }  
}
```
#### 4. Implementing selective fields logging
A `LogValuer` is any Go value that can convert itself into a Value for logging. This mechanism may be used to defer expensive operations until they are  needed, or to expand a single value into a sequence of components.
We can implement this interface on the types we need to log, this will give us the option to log only the field in which we are interested, and the fields we want to hide for security purpose.

```
type LogValuer interface {
	LogValue() Value
}
```
To implement `LogValuer` we need to implement the `LogValue` function. `LogValue` function returns a `Value`.
`Value` can represent any Go value, but unlike type any, it can represent most small values without an allocation.  The zero Value corresponds to nil.

Lets us have a type say Customer Type, this will have some filed like id, name, email id, and so on.
```go
// Customer type
type Customer struct {  
    UserId    string `json:"user_id"`  
    Name      string `json:"name"`  
    EmailId   string `json:"email_id"`  
    GSTNumber string `json:"gst_number"`  
}  
  
func (c Customer) LogValue() slog.Value {  
    var attributes []slog.Attr  
    attributes = append(attributes, slog.Attr{Key: "user_id", Value: slog.AnyValue(c.UserId)})  
    // it will return a json object, so the output will be json object  
    return slog.GroupValue(attributes...)  
}
```
Output:
```shell
{"time":"2023-12-25T20:49:19.67532+05:30","level":"INFO","source":{"function":"main.main.func1","file":"logging/main.go","line":32},"msg":"Logging customer data","customer":{"user_id":"WZhugDmTyiNxXSVKBZbboKbSj"},"meta_information":{"correlation_id":"cdd18a07-9e30-a6a8-dcf4-08b91c8f41ff","request_method":"GET","request_path":"/get_customer","request_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"}}

```
Bank type also implement `LogValuer`, but is use `slog.IntValue` which will result in a single field rather than a json object.
```go
type Bank struct {  
    BranchId     int        `json:"branch_id"`  
    BranchName   string     `json:"branch_name"`  
    BranchSecret string     `json:"branch-secret"`  
    Customers    []Customer `json:"customers"`  
}  
  
func (b Bank) LogValue() slog.Value {  
    // it will return a single value, so the output will be another field  
    return slog.IntValue(b.BranchId)  
}
```
Output:
```shell 
{"time":"2023-12-25T20:49:28.839859+05:30","level":"ERROR","source":{"function":"main.main.func2","file":"logging/main.go","line":38},"msg":"Logging customer data","bank":51,"meta_information":{"correlation_id":"3fd438ce-288d-c5f7-0634-0e994426fff6","request_method":"GET","request_path":"/get_bank","request_user_agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"}}

```

### Full code
main.go
```go
package main

import (
  "structured-logging-echo/logger"
  "context"
  "github.com/go-faker/faker/v4"
  "github.com/hashicorp/go-uuid"
  "github.com/labstack/echo"
  "log/slog"
  "math/rand"
  "os"
  "time")

func main() {  
    // setting Long date, Long time, Long Microseconds, and Long file path for log  
    opts := slog.HandlerOptions{  
       AddSource: true,  
       Level:     slog.LevelInfo,  
    }  
    jsonHandler := slog.NewJSONHandler(os.Stdout, &opts)  
    ctxHandler := logger.ContextHandler{Handler: jsonHandler}  
    logger := slog.New(ctxHandler)  
    slog.SetDefault(logger)  
  
    e := echo.New()  
    e.Use(CorrelationId)  
    e.Use(AddRouteMetaData)  
    e.GET("/get_customer", func(context echo.Context) error {  
       customer := Customer{}  
       faker.FakeData(&customer)  
       slog.InfoContext(context.Request().Context(), "Logging customer data", "customer", customer)  
       return nil  
    })  
    e.GET("/get_bank", func(context echo.Context) error {  
       bank := Bank{}  
       faker.FakeData(&bank)  
       slog.ErrorContext(context.Request().Context(), "Logging customer data", "bank", bank)  
       return nil  
    })  
  
    e.Logger.Fatal(e.Start(":8080"))  
}  
  
// Customer type 
type Customer struct {  
    UserId    string `json:"user_id"`  
    Name      string `json:"name"`  
    EmailId   string `json:"email_id"`  
    GSTNumber string `json:"gst_number"`  
}  
  
func (c Customer) LogValue() slog.Value {  
    var attributes []slog.Attr  
    attributes = append(attributes, slog.Attr{Key: "user_id", Value: slog.AnyValue(c.UserId)})  
    // it will return a json object, so the output will be json object  
    return slog.GroupValue(attributes...)  
}  
  
type Bank struct {  
    BranchId     int        `json:"branch_id"`  
    BranchName   string     `json:"branch_name"`  
    BranchSecret string     `json:"branch-secret"`  
    Customers    []Customer `json:"customers"`  
}  
  
func (b Bank) LogValue() slog.Value {  
    // it will return a single value, so the output will be another field  
    return slog.IntValue(b.BranchId)  
}  
  
// CorrelationId adding correlation id in contextfunc CorrelationId(next echo.HandlerFunc) echo.HandlerFunc {  
    return func(c echo.Context) error {  
       requestId, err := uuid.GenerateUUID()  
       if err != nil {  
          slog.ErrorContext(c.Request().Context(), "Error in generating unique correlation id "+err.Error())  
          // generating a random string of 32  
          requestId = randomString(32)  
       }  
       ctx := context.WithValue(c.Request().Context(), "correlation_id", requestId)  
       request := c.Request().Clone(ctx)  
       c.SetRequest(request)  
       return next(c)  
    }  
}  
  
// AddRouteMetaData adding meta-information about the route. Method, Path, User Agentfunc AddRouteMetaData(next echo.HandlerFunc) echo.HandlerFunc {  
    return func(c echo.Context) error {  
       path := c.Request().RequestURI  
       method := c.Request().Method  
       userAgent := c.Request().UserAgent()  
       ctx := context.WithValue(c.Request().Context(), "request_path", path)  
       ctx = context.WithValue(ctx, "request_method", method)  
       ctx = context.WithValue(ctx, "request_user_agent", userAgent)  
       request := c.Request().Clone(ctx)  
       c.SetRequest(request)  
       return next(c)  
    }  
}  
  
// Function to generate a random string of a given length  
func randomString(length int) string {  
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"  
    seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))  
  
    // Create a byte slice of the required length  
    randomBytes := make([]byte, length)  
    for i := range randomBytes {  
       randomBytes[i] = charset[seededRand.Intn(len(charset))]  
    }  
  
    return string(randomBytes)  
}
```
logger.go
```go
package logger  
  
import (  
    "context"  
    "log/slog")  
  
// ContextHandler is our base context handler, it will handle all requeststype ContextHandler struct {  
    slog.Handler  
}  
  
// Enabled determines if to log or not log, if it returns true then Handle will logfunc (ch ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {  
    return ch.Handler.Enabled(ctx, level)  
}  
  
// Handle backend for api, this will be used to configure how the logs will be structuredfunc (ch ContextHandler) Handle(ctx context.Context, r slog.Record) error {  
    r.AddAttrs(ch.addRequestId(ctx)...)  
    return ch.Handler.Handle(ctx, r)  
}  
  
// WithAttrs overriding default implementation otherwise it will call the starting JSON Handler  
func (ch ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {  
    return ContextHandler{ch.Handler.WithAttrs(attrs)}  
}  
  
// WithGroup overriding default implementation otherwise it will call the starting JSON Handler  
func (ch ContextHandler) WithGroup(name string) slog.Handler {  
    return ContextHandler{ch.Handler.WithGroup(name)}  
}  

  
func (ch ContextHandler) addRequestId(ctx context.Context) []slog.Attr {  
    var as []slog.Attr  
    correlation := getDefaultValueFromContext(ctx, "correlation_id")  
    method := getDefaultValueFromContext(ctx, "request_method")  
    path := getDefaultValueFromContext(ctx, "request_path")  
    agent := getDefaultValueFromContext(ctx, "request_user_agent")  
  
    group := slog.Group("meta_information", slog.String("correlation_id", correlation),  
       slog.String("request_method", method),  
       slog.String("request_path", path),  
       slog.String("request_user_agent", agent))  
    as = append(as, group)  
    return as  
}  
  
// getDefaultValueFromContext get default value from contextfunc getDefaultValueFromContext(ctx context.Context, key string) string {  
    value := ""  
    ctxValue := ctx.Value(key)  
    if ctxValue != nil {  
       value = ctxValue.(string)  
    }  
    return value  
}
```

