package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"taskmanager/config"
	"taskmanager/models"
	"taskmanager/routes"
	"taskmanager/utils"

	"github.com/gin-gonic/gin"
)

type testEnv struct {
	router       *gin.Engine
	dbCleanupSQL func(t *testing.T)

	admin models.User
	mgr   models.User
	mem   models.User
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	gin.SetMode(gin.TestMode)

	if os.Getenv("DB_NAME") == "" {
		_ = os.Setenv("DB_NAME", "testdbgo")
	}
	if os.Getenv("JWT_SECRET") == "" {
		_ = os.Setenv("JWT_SECRET", "test-secret")
	}

	db := config.ConnectDB()

	if err := db.Migrator().DropTable(&models.TaskAudit{}, &models.Task{}, &models.User{}); err != nil {
		t.Fatalf("failed to drop tables: %v", err)
	}
	if err := db.AutoMigrate(&models.Task{}, &models.TaskAudit{}, &models.User{}); err != nil {
		t.Fatalf("failed to migrate tables: %v", err)
	}

	router := routes.SetupRouter(db)

	admin := models.User{Name: "Admin", Email: "admin@example.com", Role: "admin"}
	mgr := models.User{Name: "Manager", Email: "manager@example.com", Role: "manager"}
	mem := models.User{Name: "Member", Email: "member@example.com", Role: "member"}

	for _, u := range []*models.User{&admin, &mgr, &mem} {
		h, err := utils.HashPassword("pass1234")
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		u.Password = h
		if err := db.Create(u).Error; err != nil {
			t.Fatalf("seed user %s: %v", u.Email, err)
		}
	}

	return &testEnv{
		router: router,
		dbCleanupSQL: func(t *testing.T) {
			t.Helper()
			_ = db.Migrator().DropTable(&models.TaskAudit{}, &models.Task{}, &models.User{})
		},
		admin: admin,
		mgr:   mgr,
		mem:   mem,
	}
}

func doRequest(t *testing.T, r http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func bearerFor(t *testing.T, u models.User) string {
	t.Helper()
	tok, err := utils.GenerateJWT(u)
	if err != nil {
		t.Fatalf("generate jwt: %v", err)
	}
	return "Bearer " + tok
}

func TestAuth_RegisterAndLogin(t *testing.T) {
	env := setupTestEnv(t)
	defer env.dbCleanupSQL(t)

	regBody := map[string]any{
		"name":     "New User",
		"email":    "new@example.com",
		"password": "pass1234",
		"role":     "member",
	}

	w := doRequest(t, env.router, http.MethodPost, "/register", regBody, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}

	loginBody := map[string]any{"email": "new@example.com", "password": "pass1234"}
	w = doRequest(t, env.router, http.MethodPost, "/login", loginBody, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal login resp: %v", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatalf("expected token in response: %v", resp)
	}
}

func TestUsers_AdminOnly(t *testing.T) {
	env := setupTestEnv(t)
	defer env.dbCleanupSQL(t)

	adminAuth := map[string]string{"Authorization": bearerFor(t, env.admin)}
	mgrAuth := map[string]string{"Authorization": bearerFor(t, env.mgr)}

	w := doRequest(t, env.router, http.MethodGet, "/users", nil, adminAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /users as admin status=%d body=%s", w.Code, w.Body.String())
	}

	w = doRequest(t, env.router, http.MethodGet, "/users", nil, mgrAuth)
	if w.Code != http.StatusForbidden {
		t.Fatalf("GET /users as manager expected 403 got=%d body=%s", w.Code, w.Body.String())
	}

	upd := map[string]any{"role": "manager"}
	w = doRequest(t, env.router, http.MethodPut, "/users/"+itoa(env.mem.ID), upd, adminAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /users/:id as admin status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestTasks_CRUDAndDecisions(t *testing.T) {
	env := setupTestEnv(t)
	defer env.dbCleanupSQL(t)

	adminAuth := map[string]string{"Authorization": bearerFor(t, env.admin)}
	mgrAuth := map[string]string{"Authorization": bearerFor(t, env.mgr)}
	memAuth := map[string]string{"Authorization": bearerFor(t, env.mem)}

	create := map[string]any{
		"title":               "T1",
		"description":         "D1",
		"assigned_to_id":       env.mem.ID,
		"progress_percentage": 0,
	}
	// Members are not allowed to create tasks.
	w := doRequest(t, env.router, http.MethodPost, "/tasks", create, memAuth)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST /tasks as member expected 403 got=%d body=%s", w.Code, w.Body.String())
	}

	// Use admin for creation to avoid assignment permission complexity.
	w = doRequest(t, env.router, http.MethodPost, "/tasks", create, adminAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /tasks status=%d body=%s", w.Code, w.Body.String())
	}

	var created models.Task
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created task: %v", err)
	}

	w = doRequest(t, env.router, http.MethodGet, "/tasks", nil, memAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /tasks status=%d body=%s", w.Code, w.Body.String())
	}

	w = doRequest(t, env.router, http.MethodGet, "/tasks/"+itoa(created.ID), nil, memAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /tasks/:id status=%d body=%s", w.Code, w.Body.String())
	}

	statusInProgress := "in_progress"
	upd := map[string]any{"status": statusInProgress}
	w = doRequest(t, env.router, http.MethodPut, "/tasks/"+itoa(created.ID), upd, memAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /tasks/:id (to in_progress) status=%d body=%s", w.Code, w.Body.String())
	}

	pp := 100
	statusPending := "pending_approval"
	upd = map[string]any{"progress_percentage": pp, "status": statusPending}
	w = doRequest(t, env.router, http.MethodPut, "/tasks/"+itoa(created.ID), upd, memAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /tasks/:id (to pending_approval) status=%d body=%s", w.Code, w.Body.String())
	}

	// Approve as admin (manager cannot access tasks they didn't create/aren't assigned to).
	w = doRequest(t, env.router, http.MethodPost, "/tasks/"+itoa(created.ID)+"/approve", map[string]any{"comments": "ok"}, adminAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /tasks/:id/approve status=%d body=%s", w.Code, w.Body.String())
	}

	w = doRequest(t, env.router, http.MethodDelete, "/tasks/"+itoa(created.ID), nil, adminAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE /tasks/:id status=%d body=%s", w.Code, w.Body.String())
	}

	// Manager decision flow: create a task under manager's hierarchy and reject it.
	managerUpdate := map[string]any{"manager_id": env.mgr.ID}
	w = doRequest(t, env.router, http.MethodPut, "/users/"+itoa(env.mem.ID), managerUpdate, adminAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /users/:id set manager_id status=%d body=%s", w.Code, w.Body.String())
	}

	create2 := map[string]any{
		"title":               "T2",
		"description":         "D2",
		"assigned_to_id":       env.mem.ID,
		"progress_percentage": 0,
	}
	w = doRequest(t, env.router, http.MethodPost, "/tasks", create2, mgrAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /tasks (manager creates) status=%d body=%s", w.Code, w.Body.String())
	}
	var created2 models.Task
	if err := json.Unmarshal(w.Body.Bytes(), &created2); err != nil {
		t.Fatalf("unmarshal created2 task: %v", err)
	}

	w = doRequest(t, env.router, http.MethodPut, "/tasks/"+itoa(created2.ID), map[string]any{"status": "in_progress"}, memAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /tasks/:id (T2 to in_progress) status=%d body=%s", w.Code, w.Body.String())
	}
	w = doRequest(t, env.router, http.MethodPut, "/tasks/"+itoa(created2.ID), map[string]any{"progress_percentage": 100, "status": "pending_approval"}, memAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /tasks/:id (T2 to pending_approval) status=%d body=%s", w.Code, w.Body.String())
	}

	w = doRequest(t, env.router, http.MethodPost, "/tasks/"+itoa(created2.ID)+"/reject", map[string]any{"reason": "Missing tests", "comments": "Add tests"}, mgrAuth)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /tasks/:id/reject status=%d body=%s", w.Code, w.Body.String())
	}
}

func itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
