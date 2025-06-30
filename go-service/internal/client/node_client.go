package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sync"
	"time"

	"github.com/shishir/go-service/models"
)

// one shared instance
var (
	nodeOnce   sync.Once
	nodeClient *NodeClient
)

// NodeClient maintains auth + HTTP client
type NodeClient struct {
	baseURL string
	email   string
	pass    string

	mu        sync.Mutex
	token     string        // bearer token, if backend returns one
	expiresAt time.Time     // naive expiry fallback (optional)
	client    *http.Client  // keeps cookies if backend uses them
}

// singleton accessor
func GetNodeClient() *NodeClient {
	nodeOnce.Do(func() {
		jar, _ := cookiejar.New(nil)
		nodeClient = &NodeClient{
			baseURL: os.Getenv("NODE_BASE_URL"),
			email:   os.Getenv("LOGIN_EMAIL"),
			pass:    os.Getenv("LOGIN_PASSWORD"),
			client:  &http.Client{Jar: jar, Timeout: 5 * time.Second},
		}
	})
	return nodeClient
}

/************ Public API ****************************************/

func GetStudentByID(id string) (*models.Student, error) {
	nc := GetNodeClient()

	// ensure we’re authenticated
	if err := nc.ensureAuth(); err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	// build request
	url := fmt.Sprintf("%s/students/%s", nc.baseURL, id)
	req, _ := http.NewRequest("GET", url, nil)

	// add bearer if we have one
	if nc.token != "" {
		req.Header.Set("Authorization", "Bearer "+nc.token)
	}

	resp, err := nc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// if session expired → re-login once and retry
	if resp.StatusCode == http.StatusUnauthorized {
		_ = nc.login() // ignore error; we’ll see on retry
		return GetStudentByID(id) // recursive one-time retry
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("backend %d: %s", resp.StatusCode, body)
	}

	var stu models.Student
	return &stu, json.NewDecoder(resp.Body).Decode(&stu)
}

/************ Internal helpers **********************************/

func (nc *NodeClient) ensureAuth() error {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// cheap check: if we have a bearer and it's <55 min old, assume OK
	if nc.token != "" && time.Until(nc.expiresAt) > 5*time.Minute {
		return nil
	}

	return nc.login()
}

func (nc *NodeClient) login() error {
	fmt.Println("email pass", nc.email, nc.pass)
	payload := map[string]string{"username": nc.email, "password": nc.pass}
	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", nc.baseURL+"/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := nc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("body", body)
		return fmt.Errorf("login failed (%d): %s", resp.StatusCode, body)
	}


	// OPTION A ─ bearer in JSON --------------------------------
	var body struct {
		Token string `json:"token"`
		Exp   int64  `json:"exp,omitempty"` // if backend returns expiry epoch
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil && body.Token != "" {
		nc.token = body.Token
		if body.Exp > 0 {
			nc.expiresAt = time.Unix(body.Exp, 0)
		} else {
			nc.expiresAt = time.Now().Add(60 * time.Minute) // default 1 h
		}
		return nil
	}

	// OPTION B ─ cookie-based session --------------------------
	// At this point, cookies are already stored in nc.client.Jar
	// We just set a fake far-future expiry to skip re-auth for a while.
	nc.token = ""                        // empty because we rely on cookie
	nc.expiresAt = time.Now().Add(60 * time.Minute)
	return nil
}
