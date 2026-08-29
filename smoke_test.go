package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go_shope/dao"
	"go_shope/model"
	"go_shope/router"
	"go_shope/service"

	"golang.org/x/crypto/bcrypt"
)

type smokeClient struct {
	t      *testing.T
	base   string
	token  string
	client *http.Client
}

func (c smokeClient) request(method, path string, body any, wantStatus int, target any) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s: got status %d, want %d, body=%s", method, path, resp.StatusCode, wantStatus, data)
	}
	if target != nil && len(data) > 0 {
		if err := json.Unmarshal(data, target); err != nil {
			c.t.Fatalf("decode %s %s response: %v, body=%s", method, path, err, data)
		}
	}
}

func (c smokeClient) login(username, password string) string {
	c.t.Helper()
	var response struct {
		Token string `json:"token"`
	}
	c.request(http.MethodPost, "/api/auth/login", map[string]any{"username": username, "password": password}, http.StatusOK, &response)
	if response.Token == "" {
		c.t.Fatal("login returned an empty token")
	}
	return response.Token
}

// TestBaselineHTTPFlow is opt-in because it writes short-lived records to the
// configured MySQL database. It cleans those records before returning.
func TestBaselineHTTPFlow(t *testing.T) {
	if os.Getenv("SHOPE_SMOKE") != "1" {
		t.Skip("set SHOPE_SMOKE=1 and MYSQL_DSN to run the MySQL-backed smoke test")
	}
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Fatal("MYSQL_DSN is required when SHOPE_SMOKE=1")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "baseline-smoke-test-secret"
	}

	repo, err := dao.New(dsn)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UnixNano()
	prefix := fmt.Sprintf("smoke_%d", stamp)
	buyerName := prefix + "_buyer"
	adminName := prefix + "_admin"
	password := "smoke-password"
	var productID, activityID, normalOrderID, seckillOrderID uint64
	defer func() {
		if normalOrderID != 0 || seckillOrderID != 0 {
			repo.DB.Where("id IN ?", []uint64{normalOrderID, seckillOrderID}).Delete(&model.Order{})
		}
		if activityID != 0 {
			repo.DB.Delete(&model.SeckillActivity{}, activityID)
		}
		if productID != 0 {
			repo.DB.Delete(&model.Product{}, productID)
		}
		repo.DB.Where("username IN ?", []string{buyerName, adminName}).Delete(&model.User{})
	}()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	admin := &model.User{Username: adminName, PasswordHash: string(hash), Role: "ADMIN"}
	if err := repo.CreateUser(admin); err != nil {
		t.Fatal(err)
	}

	engine := router.New(service.NewUserService(repo), service.NewProductService(repo), service.NewActivityService(repo), service.NewOrderService(repo), secret)
	server := httptest.NewServer(engine)
	defer server.Close()
	public := smokeClient{t: t, base: server.URL, client: server.Client()}
	for _, path := range []string{"/health", "/", "/admin"} {
		public.request(http.MethodGet, path, nil, http.StatusOK, nil)
	}
	public.request(http.MethodPost, "/api/auth/register", map[string]any{"username": buyerName, "password": password}, http.StatusCreated, nil)

	adminClient := public
	adminClient.token = adminClient.login(adminName, password)
	buyerClient := public
	buyerClient.token = buyerClient.login(buyerName, password)

	var product model.Product
	adminClient.request(http.MethodPost, "/api/admin/products", map[string]any{
		"name": prefix + " product", "description": "baseline smoke product", "price": 1299, "stock": 10, "status": "ON_SALE",
	}, http.StatusCreated, &product)
	productID = product.ID
	if productID == 0 {
		t.Fatal("created product has no ID")
	}

	start := time.Now().Add(-time.Minute).UTC()
	end := time.Now().Add(time.Hour).UTC()
	var activity model.SeckillActivity
	adminClient.request(http.MethodPost, "/api/admin/seckill/activities", map[string]any{
		"product_id": productID, "seckill_price": 999, "total_stock": 3, "start_time": start, "end_time": end, "status": "ACTIVE",
	}, http.StatusCreated, &activity)
	activityID = activity.ID

	normalRequestID := prefix + "_normal"
	var normalOrder, normalRetry model.Order
	buyerClient.request(http.MethodPost, fmt.Sprintf("/api/products/%d/orders", productID), map[string]any{"quantity": 2, "request_id": normalRequestID}, http.StatusCreated, &normalOrder)
	normalOrderID = normalOrder.ID
	buyerClient.request(http.MethodPost, fmt.Sprintf("/api/products/%d/orders", productID), map[string]any{"quantity": 2, "request_id": normalRequestID}, http.StatusCreated, &normalRetry)
	if normalRetry.ID != normalOrder.ID {
		t.Fatalf("normal-order retry created another order: %d != %d", normalRetry.ID, normalOrder.ID)
	}
	buyerClient.request(http.MethodPost, fmt.Sprintf("/api/orders/%d/cancel", normalOrder.ID), nil, http.StatusOK, nil)

	seckillRequestID := prefix + "_seckill"
	var seckillOrder, seckillRetry model.Order
	buyerClient.request(http.MethodPost, fmt.Sprintf("/api/seckill/activities/%d/orders", activityID), map[string]any{"request_id": seckillRequestID}, http.StatusCreated, &seckillOrder)
	seckillOrderID = seckillOrder.ID
	buyerClient.request(http.MethodPost, fmt.Sprintf("/api/seckill/activities/%d/orders", activityID), map[string]any{"request_id": seckillRequestID}, http.StatusCreated, &seckillRetry)
	if seckillRetry.ID != seckillOrder.ID {
		t.Fatalf("seckill-order retry created another order: %d != %d", seckillRetry.ID, seckillOrder.ID)
	}
	buyerClient.request(http.MethodPost, fmt.Sprintf("/api/orders/%d/pay", seckillOrder.ID), nil, http.StatusOK, nil)

	var updatedActivity model.SeckillActivity
	adminClient.request(http.MethodPut, fmt.Sprintf("/api/admin/seckill/activities/%d", activityID), map[string]any{
		"product_id": productID, "seckill_price": 899, "total_stock": 4, "start_time": start, "end_time": end, "status": "ACTIVE",
	}, http.StatusOK, &updatedActivity)
	if updatedActivity.AvailableStock != 3 {
		t.Fatalf("activity stock mismatch: got %d available after one sale and total 4", updatedActivity.AvailableStock)
	}

	var storedProduct model.Product
	if err := repo.DB.First(&storedProduct, productID).Error; err != nil {
		t.Fatal(err)
	}
	if storedProduct.Stock != 9 {
		t.Fatalf("product stock mismatch: got %d, want 9", storedProduct.Stock)
	}
	var adminOrders []model.Order
	adminClient.request(http.MethodGet, "/api/admin/orders", nil, http.StatusOK, &adminOrders)
	found := 0
	for _, order := range adminOrders {
		if strings.HasPrefix(order.RequestID, prefix) {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("admin order list contains %d smoke orders, want 2", found)
	}
}
