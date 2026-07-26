//go:build unit

package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	entintercept "github.com/Wei-Shaw/sub2api/ent/intercept"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type checkoutInfoTestEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		Plans []json.RawMessage `json:"plans"`
	} `json:"data"`
}

func TestPaymentHandlerGetCheckoutInfoReturnsEmptyPlans(t *testing.T) {
	handler, _ := newCheckoutInfoTestHandler(t)

	recorder := runCheckoutInfoRequest(handler)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response checkoutInfoTestEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.NotNil(t, response.Data)
	require.NotNil(t, response.Data.Plans)
	require.Empty(t, response.Data.Plans)
}

func TestPaymentHandlerGetCheckoutInfoPropagatesPlanQueryError(t *testing.T) {
	handler, client := newCheckoutInfoTestHandler(t)
	queryErr := errors.New("subscription plan query failed")
	client.SubscriptionPlan.Intercept(entintercept.Func(func(context.Context, entintercept.Query) error {
		return queryErr
	}))

	recorder := runCheckoutInfoRequest(handler)

	requireCheckoutInfoInternalError(t, recorder)
}

func TestPaymentHandlerGetCheckoutInfoPropagatesGroupQueryError(t *testing.T) {
	handler, client := newCheckoutInfoTestHandler(t)
	_, err := client.SubscriptionPlan.Create().
		SetGroupID(42).
		SetName("Pro").
		SetPrice(9.99).
		SetForSale(true).
		Save(context.Background())
	require.NoError(t, err)

	queryErr := errors.New("plan group query failed")
	client.Group.Intercept(entintercept.Func(func(context.Context, entintercept.Query) error {
		return queryErr
	}))

	recorder := runCheckoutInfoRequest(handler)

	requireCheckoutInfoInternalError(t, recorder)
}

func newCheckoutInfoTestHandler(t *testing.T) (*PaymentHandler, *dbent.Client) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbName := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared",
		strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()),
	)
	db, err := sql.Open("sqlite", dbName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	configService := service.NewPaymentConfigService(
		client,
		repository.NewSettingRepository(client),
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	return NewPaymentHandler(nil, configService), client
}

func runCheckoutInfoRequest(handler *PaymentHandler) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/payment/checkout-info", nil)
	handler.GetCheckoutInfo(ctx)
	return recorder
}

func requireCheckoutInfoInternalError(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var response checkoutInfoTestEnvelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, "internal error", response.Message)
	require.Nil(t, response.Data)
}
