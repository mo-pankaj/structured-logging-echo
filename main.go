package main

import (
	"context"
	"github.com/go-faker/faker/v4"
	"github.com/hashicorp/go-uuid"
	"github.com/labstack/echo"
	"log/slog"
	"math/rand"
	"os"
	"structured-logging-echo/logger"
	"time"
)

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

// CorrelationId adding correlation id in context
func CorrelationId(next echo.HandlerFunc) echo.HandlerFunc {
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

// AddRouteMetaData adding meta-information about the route. Method, Path, User Agent
func AddRouteMetaData(next echo.HandlerFunc) echo.HandlerFunc {
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
