package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"net/url"
	"strings"

	socket_io "Goauld/common/socket.io"

	"Goauld/client/types"
	common_types "Goauld/common/types"
	httpTypes "Goauld/common/types"
)

type API struct {
	client      *http.Client
	server      string
	accessToken string
	insecure    bool
}

// NewAPI return a new API
func NewAPI(server string, accessToken string, insecure bool) *API {
	api := &API{
		client:      &http.Client{},
		server:      server,
		accessToken: accessToken,
		insecure:    insecure,
	}
	if insecure {
		api.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	return api
}

// Delete generic method to perform DELETE request with the appropriate authentication header
func (api *API) delete(p string) (*http.Response, error) {
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

// get generic method to perform GET request with the appropriate authentication header
func (api *API) get(p string) (*http.Response, error) {
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
func (api *API) post(p string, body io.Reader) (*http.Response, error) {
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
	res, err := api.get("/manage/agent/")
	if err != nil {
		return nil, errors.New("Error while requesting agent list: " + err.Error())
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("Error while reading agent list: " + err.Error())
	}

	var agents []types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return nil, err
	}
	// for i := range agents {
	// 	agents[i].ParseFPR()
	// }
	return agents, nil
}

// GetAgentById fetch the agent associated to the id
func (api *API) GetAgentById(id string) (types.Agent, error) {
	id = url.PathEscape(id)
	res, err := api.get("/manage/agent/" + id)
	if err != nil {
		return types.Agent{}, errors.New("Error while requesting agent by id: " + err.Error())
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return types.Agent{}, errors.New("Error while reading agent by id: " + err.Error())
	}

	var agents types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return types.Agent{}, err
	}
	// for i := range agents {
	// 	agents[i].ParseFPR()
	// }
	return agents, nil
}

// GetAgentByName fetch the agent associated with the name
func (api *API) GetAgentByName(name string) (types.Agent, error) {
	name = url.PathEscape(name)
	res, err := api.get("/manage/agent/by_name/" + name)
	if err != nil {
		return types.Agent{}, fmt.Errorf("error while requesting agent by id: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return types.Agent{}, fmt.Errorf("Error while reading agent by id: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return types.Agent{}, fmt.Errorf("error while requesting agent by id: %s", strings.TrimSpace(string(body)))
	}

	var agents types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return types.Agent{}, fmt.Errorf("%v: %s", err, string(body))
	}
	// for i := range agents {
	// 	agents[i].ParseFPR()
	// }
	return agents, nil
}

// KillAgent kills the agent
func (api *API) KillAgent(id string, doExit bool) error {
	id = url.PathEscape(id)
	u := fmt.Sprintf("/manage/agent/%s/kill", id)
	body := socket_io.ExitData{Kill: doExit}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := api.post(u, bytes.NewBuffer(b))
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
	id = url.PathEscape(id)
	res, err := api.delete("/manage/agent/" + id)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%d", res.StatusCode)
	}
	return nil
}

func (api *API) DumpAll() (error, []common_types.State) {
	res, err := api.get("/admin/dump/")
	if err != nil {
		return err, nil
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err, nil
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error while dumping active agents : %s", strings.TrimSpace(string(body))), nil
	}

	var result []common_types.State
	err = yaml.Unmarshal(body, &result)
	if err != nil {
		return err, nil
	}

	return nil, result
}

func (api *API) UpdateLogLevel(level string) (error, map[string]interface{}) {
	res, err := api.post(fmt.Sprintf("/admin/loglevel/%s", url.PathEscape(level)), nil)
	if err != nil {
		return err, nil
	}
	defer res.Body.Close()
	result := map[string]interface{}{}
	body, err := io.ReadAll(res.Body)
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err, nil
	}
	return nil, result
}

func (api *API) GetConfig() (error, string) {
	res, err := api.get("/admin/config/")
	if err != nil {
		return err, ""
	}
	defer res.Body.Close()
	result := httpTypes.HttpResponse{}

	body, err := io.ReadAll(res.Body)
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err, ""
	}
	if !result.Success {
		return errors.New(result.Message), ""
	}
	cfg := result.Message
	return nil, cfg
}
