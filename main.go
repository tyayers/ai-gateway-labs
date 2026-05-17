package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2/google"
)

var (
	// Global cache for analytics data
	analyticsCache      []byte
	analyticsCacheMutex sync.RWMutex

	// SSE Broker
	clients      = make(map[chan []byte]bool)
	clientsMutex sync.Mutex
)

// Helper function to broadcast data to all connected SSE clients
func broadcast(data []byte) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		// Send data without blocking
		select {
		case client <- data:
		default:
			// If client channel is full, we could drop them, but let's just skip
		}
	}
}

// Helper function to handle JSON responses
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// withCORS middleware adds CORS headers for permissive access
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func getGCPClient(ctx context.Context) (*http.Client, error) {
	client, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}
	return client, nil
}

func doApigeeRequest(ctx context.Context, method, url string, body interface{}, target interface{}) error {
	client, err := getGCPClient(ctx)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("apigee API error (%d): %s", resp.StatusCode, string(respBytes))
	}

	if target != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

func fetchAnalyticsForEmail(ctx context.Context, projectId, env, escapedTimeRange, email string) (map[string]interface{}, error) {
	escapedFilter := strings.Replace(url.QueryEscape(fmt.Sprintf("(developer_email eq '%s')", email)), "+", "%20", -1)

	var appStats []interface{}
	var productStats []interface{}
	var modelStats []interface{}

	// 1. Stats by developer_app
	appStatsUrl := fmt.Sprintf("https://apigee.googleapis.com/v1/organizations/%s/environments/%s/stats/developer_app?select=sum(message_count),sum(dc_ai_prompt_token_count),sum(dc_ai_response_token_count),avg(dc_ai_time_first_token)&timeUnit=day&timeRange=%s&filter=%s",
		projectId, env, escapedTimeRange, escapedFilter)
	var appStatsResp map[string]interface{}
	if err := doApigeeRequest(ctx, "GET", appStatsUrl, nil, &appStatsResp); err == nil {
		delete(appStatsResp, "metaData")
		appStats = append(appStats, appStatsResp)
	}

	// 2. Stats by api_product
	prodStatsUrl := fmt.Sprintf("https://apigee.googleapis.com/v1/organizations/%s/environments/%s/stats/api_product?select=sum(message_count),sum(dc_ai_prompt_token_count),sum(dc_ai_response_token_count),avg(dc_ai_time_first_token)&timeUnit=day&timeRange=%s&filter=%s",
		projectId, env, escapedTimeRange, escapedFilter)
	var prodStatsResp map[string]interface{}
	if err := doApigeeRequest(ctx, "GET", prodStatsUrl, nil, &prodStatsResp); err == nil {
		delete(prodStatsResp, "metaData")
		productStats = append(productStats, prodStatsResp)
	}

	// 3. Stats by dc_ai_model
	modelStatsUrl := fmt.Sprintf("https://apigee.googleapis.com/v1/organizations/%s/environments/%s/stats/dc_ai_model?select=sum(dc_ai_prompt_token_count),sum(dc_ai_response_token_count),avg(dc_ai_time_first_token)&timeUnit=day&timeRange=%s&filter=%s",
		projectId, env, escapedTimeRange, escapedFilter)
	var modelStatsResp map[string]interface{}
	if err := doApigeeRequest(ctx, "GET", modelStatsUrl, nil, &modelStatsResp); err == nil {
		delete(modelStatsResp, "metaData")
		modelStats = append(modelStats, modelStatsResp)
	}

	return map[string]interface{}{
		"app":     appStats,
		"product": productStats,
		"model":   modelStats,
	}, nil
}

func refreshAnalyticsData(projectId string) error {
	ctx := context.Background()

	// Compute timeRange (last 3 months)
	now := time.Now().UTC()
	threeMonthsAgo := now.AddDate(0, -3, 0)
	timeRange := fmt.Sprintf("%02d/%02d/%04d %02d:%02d~%02d/%02d/%04d %02d:%02d",
		threeMonthsAgo.Month(), threeMonthsAgo.Day(), threeMonthsAgo.Year(), threeMonthsAgo.Hour(), threeMonthsAgo.Minute(),
		now.Month(), now.Day(), now.Year(), now.Hour(), now.Minute())
	escapedTimeRange := strings.Replace(url.QueryEscape(timeRange), "+", "%20", -1)

	// Get all environments for this org
	envUrl := fmt.Sprintf("https://apigee.googleapis.com/v1/organizations/%s/environments", projectId)
	var envs []string
	if err := doApigeeRequest(ctx, "GET", envUrl, nil, &envs); err != nil {
		return fmt.Errorf("failed to get environments for org %s: %v", projectId, err)
	}

	// Determine the list of emails to query
	var emailsToQuery []string
	devsUrl := fmt.Sprintf("https://apigee.googleapis.com/v1/organizations/%s/developers", projectId)
	var devsResp struct {
		Developer []struct {
			Email string `json:"email"`
		} `json:"developer"`
	}
	if err := doApigeeRequest(ctx, "GET", devsUrl, nil, &devsResp); err != nil {
		return fmt.Errorf("failed to get developers for org %s: %v", projectId, err)
	}
	for _, d := range devsResp.Developer {
		emailsToQuery = append(emailsToQuery, d.Email)
	}

	result := make(map[string]interface{})

	for _, userEmail := range emailsToQuery {
		userStats := map[string][]interface{}{
			"app":     {},
			"product": {},
			"model":   {},
		}

		for _, env := range envs {
			stats, err := fetchAnalyticsForEmail(ctx, projectId, env, escapedTimeRange, userEmail)
			if err != nil {
				continue
			}

			if apps, ok := stats["app"].([]interface{}); ok {
				userStats["app"] = append(userStats["app"], apps...)
			}
			if prods, ok := stats["product"].([]interface{}); ok {
				userStats["product"] = append(userStats["product"], prods...)
			}
			if mods, ok := stats["model"].([]interface{}); ok {
				userStats["model"] = append(userStats["model"], mods...)
			}
		}
		result[userEmail] = userStats
	}

	newCache, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %v", err)
	}

	analyticsCacheMutex.Lock()
	changed := !bytes.Equal(analyticsCache, newCache)
	if changed {
		analyticsCache = newCache
	}
	analyticsCacheMutex.Unlock()

	if changed {
		log.Println("Data changed, broadcasting to clients...")
		broadcast(newCache)
	} else {
		log.Println("Data unchanged, skip broadcast.")
	}

	return nil
}

func userAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("projectId")
	if projectId == "" {
		projectId = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	// Always trigger a refresh when explicitly called
	go func() {
		log.Println("Manual cache refresh triggered")
		if err := refreshAnalyticsData(projectId); err != nil {
			log.Printf("Error refreshing analytics: %v", err)
		}
	}()

	// Immediately return whatever is in cache
	analyticsCacheMutex.RLock()
	cachedData := analyticsCache
	analyticsCacheMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if cachedData != nil {
		w.Write(cachedData)
	} else {
		w.Write([]byte(`{}`))
	}
}

func configHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"googleCloudProject": projectId,
	})
}

// SSE handler to push analytics to the client
func sseHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this specific client
	clientChan := make(chan []byte, 1)

	clientsMutex.Lock()
	clients[clientChan] = true
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(clients, clientChan)
		clientsMutex.Unlock()
		close(clientChan)
	}()

	// Push current cache immediately if available
	analyticsCacheMutex.RLock()
	if analyticsCache != nil {
		w.Write([]byte("data: "))
		w.Write(analyticsCache)
		w.Write([]byte("\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
	analyticsCacheMutex.RUnlock()

	// Wait for updates or connection close
	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case data := <-clientChan:
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func main() {

	mux := http.NewServeMux()

	landingFs := http.FileServer(http.Dir("public"))
	mux.Handle("/", landingFs)

	mux.HandleFunc("/api/projects/{projectId}/users/analytics/stream", withCORS(sseHandler))
	mux.HandleFunc("/api/projects/{projectId}/users/{email}/analytics", withCORS(userAnalyticsHandler))
	mux.HandleFunc("/api/projects/{projectId}/users/analytics", withCORS(userAnalyticsHandler))
	mux.HandleFunc("/api/config", withCORS(configHandler))

	// Initial cache load for the default project (if set)
	projectId := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectId != "" {
		log.Printf("Starting initial background cache refresh for project: %s", projectId)
		go func() {
			if err := refreshAnalyticsData(projectId); err != nil {
				log.Printf("Initial cache refresh failed: %v", err)
			}
		}()
	}

	log.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
