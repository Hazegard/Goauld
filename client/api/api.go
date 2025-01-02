package api

import (
	"Goauld/client/config"
	"Goauld/common/types"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type API struct {
	client      *http.Client
	server      string
	accessToken string
}

// NewAPI return a new API
func NewAPI(cfg *config.ClientConfig) *API {
	return &API{
		client:      &http.Client{},
		server:      cfg.Server,
		accessToken: cfg.AccessToken,
	}
}

// Delete generic method to perform DELETE request with the appropriate authentication header
func (api *API) Delete(p string) (*http.Response, error) {
	u, err := url.JoinPath(api.server, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.accessToken)
	return api.client.Do(req)
}

// Get generic method to perform GET request with the appropriate authentication header
func (api *API) Get(p string) (*http.Response, error) {
	u, err := url.JoinPath(api.server, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.accessToken)
	return api.client.Do(req)
}

// Post generic method to perform POST request with the appropriate authentication header
func (api *API) Post(p string, body io.Reader) (*http.Response, error) {
	u, err := url.JoinPath(api.server, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.accessToken)
	return api.client.Do(req)
}

// GetAgents fetch a list of the agents
func (api *API) GetAgents() ([]types.Agent, error) {
	res, err := api.Get("/manage/agent/")
	if err != nil {
		return nil, errors.New("Error while requesting agent list")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("Error while requesting agent list")
	}

	var agents []types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return nil, err
	}
	return agents, nil
}

// KillAgent kills the agent
func (api *API) KillAgent(id string) error {
	res, err := api.Post("/manage/agent/kill/"+id, nil)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%d", res.StatusCode)
	}
	return nil
}

// DeleteAgent kills the agent
func (api *API) DeleteAgent(id string) error {
	res, err := api.Delete("/manage/agent/" + id)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%d", res.StatusCode)
	}
	return nil
}
