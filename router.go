package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Client represents a device connected to the router.
type Client struct {
	Name   string `json:"name"`
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Policy string `json:"policy"`
	Deny   bool   `json:"deny"`
}

// KeeneticRouter communicates with the Keenetic router HTTP API.
type KeeneticRouter struct {
	BaseURL  string
	Username string
	Password string
	Name     string
	client   *http.Client
}

func NewKeeneticRouter(address, username, password, name string) *KeeneticRouter {
	if !strings.HasPrefix(address, "http") {
		address = strings.TrimSuffix(address, "/")
		address = "http://" + address
	}
	jar, _ := cookiejar.New(nil)
	return &KeeneticRouter{
		BaseURL:  address,
		Username: username,
		Password: password,
		Name:     name,
		client: &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		},
	}
}

func (r *KeeneticRouter) Login() error {
	resp, err := r.client.Get(r.BaseURL + "/auth")
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}
	if resp.StatusCode != 401 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	realm := resp.Header.Get("X-NDM-Realm")
	challenge := resp.Header.Get("X-NDM-Challenge")

	md5sum := md5.Sum([]byte(r.Username + ":" + realm + ":" + r.Password))
	sha256sum := sha256.Sum256([]byte(challenge + fmt.Sprintf("%x", md5sum)))

	body, _ := json.Marshal(map[string]string{
		"login":    r.Username,
		"password": fmt.Sprintf("%x", sha256sum),
	})
	authResp, err := r.client.Post(r.BaseURL+"/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("auth request error: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != 200 {
		return fmt.Errorf("authentication failed: status %d", authResp.StatusCode)
	}
	return nil
}

func (r *KeeneticRouter) get(endpoint string) ([]byte, error) {
	resp, err := r.client.Get(r.BaseURL + "/" + endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (r *KeeneticRouter) post(endpoint string, data interface{}) error {
	body, _ := json.Marshal(data)
	resp, err := r.client.Post(r.BaseURL+"/"+endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed: status %d", resp.StatusCode)
	}
	return nil
}

func (r *KeeneticRouter) GetNetworkIP() (string, error) {
	if err := r.Login(); err != nil {
		return "", err
	}
	data, err := r.get("rci/sc/interface/Bridge0/ip/address")
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	ip, _ := result["address"].(string)
	return ip, nil
}

func (r *KeeneticRouter) GetKeenDNSURLs() ([]string, error) {
	if err := r.Login(); err != nil {
		return nil, err
	}
	data, err := r.get("rci/ip/http/ssl/acme/list/certificate")
	if err != nil {
		return nil, err
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	var urls []string
	for _, item := range list {
		if d, ok := item["domain"].(string); ok {
			urls = append(urls, d)
		}
	}
	return urls, nil
}

func (r *KeeneticRouter) GetPolicies() (map[string]interface{}, error) {
	if err := r.Login(); err != nil {
		return nil, err
	}
	data, err := r.get("rci/show/rc/ip/policy")
	if err != nil {
		return nil, err
	}
	var policies map[string]interface{}
	if err := json.Unmarshal(data, &policies); err != nil {
		return nil, err
	}
	return policies, nil
}

func (r *KeeneticRouter) GetOnlineClients() ([]Client, error) {
	if err := r.Login(); err != nil {
		return nil, err
	}
	raw, err := r.get("rci/show/ip/hotspot/host")
	if err != nil {
		return nil, err
	}
	var rawClients []map[string]interface{}
	if err := json.Unmarshal(raw, &rawClients); err != nil {
		return nil, err
	}

	byMAC := make(map[string]*Client)
	for _, c := range rawClients {
		mac := strings.ToLower(fmt.Sprintf("%v", c["mac"]))
		cl := &Client{MAC: mac}
		if v, ok := c["name"].(string); ok {
			cl.Name = v
		}
		if v, ok := c["ip"].(string); ok {
			cl.IP = v
		}
		byMAC[mac] = cl
	}

	// Merge policy assignments
	if policyRaw, err := r.get("rci/show/rc/ip/hotspot/host"); err == nil {
		var policyList []map[string]interface{}
		if json.Unmarshal(policyRaw, &policyList) == nil {
			for _, p := range policyList {
				mac := strings.ToLower(fmt.Sprintf("%v", p["mac"]))
				cl, ok := byMAC[mac]
				if !ok {
					cl = &Client{MAC: mac, Name: "Unknown", IP: "N/A"}
					byMAC[mac] = cl
				}
				if v, ok := p["policy"].(string); ok {
					cl.Policy = v
				}
				if v, ok := p["deny"].(bool); ok {
					cl.Deny = v
				}
			}
		}
	}

	clients := make([]Client, 0, len(byMAC))
	for _, cl := range byMAC {
		clients = append(clients, *cl)
	}
	return clients, nil
}

func (r *KeeneticRouter) ApplyPolicy(mac, policy string) error {
	if err := r.Login(); err != nil {
		return err
	}
	var policyVal interface{} = policy
	if policy == "" {
		policyVal = false
	}
	return r.post("rci/ip/hotspot/host", map[string]interface{}{
		"mac":      mac,
		"policy":   policyVal,
		"permit":   true,
		"schedule": false,
	})
}

func (r *KeeneticRouter) SetClientBlock(mac string) error {
	if err := r.Login(); err != nil {
		return err
	}
	return r.post("rci/ip/hotspot/host", map[string]interface{}{
		"mac":      mac,
		"schedule": false,
		"deny":     true,
	})
}

// PolicyLabel returns a human-readable label for a policy value.
func PolicyLabel(policy string, policies map[string]interface{}, deny bool) string {
	if deny {
		return "Blocked"
	}
	if policy == "" {
		return "Default"
	}
	if info, ok := policies[policy]; ok {
		if m, ok := info.(map[string]interface{}); ok {
			if desc, ok := m["description"].(string); ok && desc != "" {
				return desc
			}
		}
	}
	return policy
}
