package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	handler := newTaskHandler(filepath.Join(t.TempDir(), "task.json"))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest("GET", "/health", nil))
	if res.Code != 200 || res.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected health response: %d %s", res.Code, res.Body)
	}
}

func TestTasksAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.json")
	handler := newTaskHandler(path)
	for _, tc := range []struct {
		method, body string
		want         int
	}{
		{"GET", "", 200},
		{"POST", `{"name":" First "}`, 201},
		{"POST", `{"name":"Second","status":"Done"}`, 201},
		{"POST", `{"name":""}`, 400},
		{"POST", `{"name":"Bad","status":"unknown"}`, 400},
		{"POST", `{"name":"Bad","id":7}`, 400},
		{"POST", `{`, 400},
		{"POST", `{"name":"Bad"} {}`, 400},
		{"DELETE", "", 405},
	} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(tc.method, "/tasks", strings.NewReader(tc.body)))
		if res.Code != tc.want {
			t.Fatalf("%s %s: got %d: %s", tc.method, tc.body, res.Code, res.Body)
		}
		if res.Header().Get("Content-Type") != "application/json" {
			t.Fatal("missing JSON content type")
		}
	}
	res := httptest.NewRecorder()
	newTaskHandler(path).ServeHTTP(res, httptest.NewRequest("GET", "/tasks", nil))
	var tasks []Task
	if err := json.Unmarshal(res.Body.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].Id != 1 || tasks[0].Name != "First" || tasks[0].Status != "Progress" || tasks[1].Id != 2 {
		t.Fatalf("unexpected persisted tasks: %+v", tasks)
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest("GET", "/tasks", nil))
	if res.Code != 500 {
		t.Fatalf("corrupt file: got %d", res.Code)
	}
}

func TestUpdateAndDeleteTask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "task.json")
	if err := saveTasks(path, []Task{{Id: 8, Name: "Buy Car", Status: "Progress"}, {Id: 12, Name: "Keep", Status: "Done"}}); err != nil {
		t.Fatal(err)
	}
	handler := newTaskHandler(path)
	for _, tc := range []struct {
		method, url, body string
		want              int
	}{
		{"PUT", "/tasks/8", `{"name":"Buy Car","status":"Done"}`, 200},
		{"PUT", "/tasks/8", `{"name":"Buy Car"}`, 400},
		{"PUT", "/tasks/8", `{"id":8,"name":"Buy Car","status":"Done"}`, 400},
		{"PUT", "/tasks/8", `{`, 400},
		{"PUT", "/tasks/8", `{"name":"Car","status":"Done"} {}`, 400},
		{"PUT", "/tasks/99", `{"name":"Car","status":"Done"}`, 404},
		{"DELETE", "/tasks/nope", "", 400},
		{"DELETE", "/tasks/0", "", 400},
		{"POST", "/tasks/8", "", 405},
		{"DELETE", "/tasks/8", "", 204},
		{"DELETE", "/tasks/8", "", 404},
	} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body)))
		if res.Code != tc.want {
			t.Fatalf("%s %s: got %d: %s", tc.method, tc.url, res.Code, res.Body)
		}
		if tc.want == 200 {
			var task Task
			if err := json.Unmarshal(res.Body.Bytes(), &task); err != nil || task.Id != 8 || task.Status != "Done" {
				t.Fatalf("unexpected update: %s", res.Body)
			}
		}
		if tc.want == 204 && res.Body.Len() != 0 {
			t.Fatal("delete response must have no body")
		}
	}
	res := httptest.NewRecorder()
	newTaskHandler(path).ServeHTTP(res, httptest.NewRequest("GET", "/tasks", nil))
	var tasks []Task
	if err := json.Unmarshal(res.Body.Bytes(), &tasks); err != nil || len(tasks) != 1 || tasks[0].Id != 12 {
		t.Fatalf("unexpected persisted tasks: %s", res.Body)
	}
}
