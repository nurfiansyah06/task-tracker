package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s %s", r.Method, r.URL.RequestURI(), rec.status, time.Since(start), r.RemoteAddr)
	})
}

func newTaskHandler(path string) http.Handler {
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, POST")
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		mu.Lock()
		defer mu.Unlock()

		tasks := []Task{}
		data, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			writeJSON(w, 500, map[string]string{"error": "cannot read tasks"})
			return
		}
		if err == nil {
			if err := json.Unmarshal(data, &tasks); err != nil {
				writeJSON(w, 500, map[string]string{"error": "cannot read tasks"})
				return
			}
		}
		if tasks == nil {
			tasks = []Task{}
		}
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, tasks)
			return
		}
		var input struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON body"})
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "expected one JSON object"})
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		if input.Status == "" {
			input.Status = "Progress"
		}
		if input.Name == "" || (input.Status != "Progress" && input.Status != "Done") {
			writeJSON(w, 400, map[string]string{"error": "name is required; status must be Progress or Done"})
			return
		}
		task := Task{Id: 1, Name: input.Name, Status: input.Status}
		for _, existing := range tasks {
			if existing.Id >= task.Id {
				task.Id = existing.Id + 1
			}
		}
		tasks = append(tasks, task)
		if err := saveTasks(path, tasks); err != nil {
			writeJSON(w, 500, map[string]string{"error": "cannot save task"})
			return
		}
		writeJSON(w, http.StatusCreated, task)
	})
	mux.HandleFunc("/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodDelete {
			w.Header().Set("Allow", "PUT, DELETE")
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id < 1 {
			writeJSON(w, 400, map[string]string{"error": "id must be a positive integer"})
			return
		}
		mu.Lock()
		defer mu.Unlock()
		data, err := os.ReadFile(path)
		tasks := []Task{}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			writeJSON(w, 500, map[string]string{"error": "cannot read tasks"})
			return
		}
		if err == nil {
			if err := json.Unmarshal(data, &tasks); err != nil {
				writeJSON(w, 500, map[string]string{"error": "cannot read tasks"})
				return
			}
		}
		index := -1
		for i, task := range tasks {
			if task.Id == id {
				index = i
				break
			}
		}
		if index == -1 {
			writeJSON(w, 404, map[string]string{"error": "task not found"})
			return
		}
		if r.Method == http.MethodDelete {
			tasks = append(tasks[:index], tasks[index+1:]...)
		} else {
			var input struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&input); err != nil {
				writeJSON(w, 400, map[string]string{"error": "invalid JSON body; send name and status only"})
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				writeJSON(w, 400, map[string]string{"error": "expected one JSON object"})
				return
			}
			input.Name = strings.TrimSpace(input.Name)
			if input.Name == "" || (input.Status != "Progress" && input.Status != "Done") {
				writeJSON(w, 400, map[string]string{"error": "name is required; status must be Progress or Done"})
				return
			}
			tasks[index] = Task{Id: id, Name: input.Name, Status: input.Status}
		}
		if err := saveTasks(path, tasks); err != nil {
			writeJSON(w, 500, map[string]string{"error": "cannot save tasks"})
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, tasks[index])
	})
	return mux
}

func saveTasks(path string, tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", " ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".tasks-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
