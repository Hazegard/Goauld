// Package api holds the client API
package api

import (
	"Goauld/common"
	"Goauld/common/log"
	"Goauld/common/net"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/goccy/go-yaml"
	"golang.org/x/crypto/bcrypt"

	socketio "Goauld/common/socket.io"

	"Goauld/client/types"
	commontypes "Goauld/common/types"
)

// API holds the CLI API information.
type API struct {
	client      *http.Client
	server      string
	accessToken string
	insecure    bool
	adminToken  string
}

// NewAPI return a new API.
func NewAPI(server string, accessToken string, insecure bool, adminToken string) *API {
	api := &API{
		client:      &http.Client{},
		server:      server,
		accessToken: accessToken,
		insecure:    insecure,
		adminToken:  adminToken,
	}
	if insecure {
		api.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		}
	}

	return api
}

// HandleError parses the output if it cannot be parsed as a JSON
// to display more useful messages to users.
func HandleError(b []byte) error {
	body := strings.TrimSpace(string(b))
	if body == net.Forbidden {
		return fmt.Errorf("%s: IP address not whitelisted", body)
	}
	if body == net.Unauthorized {
		return fmt.Errorf("%s: Invalid access token", body)
	}

	return errors.New(strings.TrimSpace(body))
}

// Delete generic method to perform DELETE request with the appropriate authentication header.
func (api *API) delete(p string) (*http.Response, error) {
	u, err := url.JoinPath(api.server, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodDelete, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.genToken())

	return api.client.Do(req)
}

func (api *API) genToken() string {
	if api.adminToken == "" {
		return api.accessToken
	}

	return fmt.Sprintf("%s:%s", api.accessToken, api.adminToken)
}

// get is generic method to perform GET request with the appropriate authentication header.
func (api *API) get(p string) (*http.Response, error) {
	u, err := url.JoinPath(api.server, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.genToken())

	return api.client.Do(req)
}

// Post generic method to perform POST request with the appropriate authentication header.
func (api *API) post(p string, body io.Reader) (*http.Response, error) {
	u, err := url.JoinPath(api.server, p)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.genToken())

	return api.client.Do(req)
}

// GetAgents fetch a list of the agents.
func (api *API) GetAgents() ([]types.Agent, error) {
	res, err := api.get("/manage/agent/")
	if err != nil {
		return nil, errors.New("Error while requesting agent list: " + err.Error())
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("Error while reading agent list: " + err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return nil, HandleError(body)
	}

	var agents []types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return nil, err
	}

	return agents, nil
}

// GetAgentByID fetch the agent associated with the id.
func (api *API) GetAgentByID(id string) (types.Agent, error) {
	id = url.PathEscape(id)
	res, err := api.get("/manage/agent/" + id)
	if err != nil {
		return types.Agent{}, errors.New("Error while requesting agent by id: " + err.Error())
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return types.Agent{}, errors.New("Error while reading agent by id: " + err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return types.Agent{}, HandleError(body)
	}

	var agents types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return types.Agent{}, err
	}

	return agents, nil
}

// GetAgentByName fetch the agent associated with the name.
func (api *API) GetAgentByName(name string) (types.Agent, error) {
	log.Trace().Str("name", name).Msg("GetAgentByName")
	name = url.PathEscape(name)
	res, err := api.get("/manage/agent/by_name/" + name)
	if err != nil {
		return types.Agent{}, fmt.Errorf("error while requesting agent by id: %w", err)
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return types.Agent{}, fmt.Errorf("error while reading agent by id: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return types.Agent{}, HandleError(body)
	}

	var agents types.Agent
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return types.Agent{}, fmt.Errorf("%w: %s", err, string(body))
	}
	// for i := range agents {
	// 	agents[i].ParseFPR()
	// }
	return agents, nil
}

// KillAgent kills the agent.
func (api *API) KillAgent(id string, doExit bool, doDelete bool, password string) error {
	id = url.PathEscape(id)
	u := fmt.Sprintf("/manage/agent/%s/kill", id)

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	body := socketio.ExitRequest{
		Kill:           doExit,
		Delete:         doDelete,
		HashedPassword: hashedPwd,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	res, err := api.post(u, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		//nolint:errcheck
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("error reading response (%s)", res.Status)
		}

		return HandleError(body)
	}

	return nil
}

// DeleteAgent kills the agent.
func (api *API) DeleteAgent(id string) error {
	id = url.PathEscape(id)
	res, err := api.delete("/manage/agent/" + id)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		//nolint:errcheck
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("error reading response (%s)", res.Status)
		}

		return HandleError(body)
	}

	return nil
}

// DumpAll return the information related to running agents connected to the server.
func (api *API) DumpAll() ([]commontypes.State, error) {
	res, err := api.get("/admin/dump/")
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, HandleError(body)
	}

	var result []commontypes.State
	err = yaml.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateLogLevel updates the server log level.
func (api *API) UpdateLogLevel(level string) (map[string]any, error) {
	res, err := api.post("/admin/loglevel/"+url.PathEscape(level), nil)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, HandleError(body)
	}
	result := map[string]any{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Version return.
func (api *API) Version() (common.JVersion, error) {
	return api.version("manage")
}

// version fetches the server version to check whether the client or the agent are using the same version.
func (api *API) version(route string) (common.JVersion, error) {
	res, err := api.get(fmt.Sprintf("/%s/version/", route))
	if err != nil {
		return common.JVersion{}, err
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return common.JVersion{}, err
	}
	if res.StatusCode != http.StatusOK {
		return common.JVersion{}, HandleError(body)
	}

	result := common.JVersion{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return common.JVersion{}, err
	}

	return result, nil
}

// GetConfig fetches the server side configuration.
func (api *API) GetConfig() (string, error) {
	res, err := api.get("/admin/config/")
	if err != nil {
		return "", err
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", HandleError(body)
	}
	result := commontypes.HTTPResponse{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	if !result.Success {
		return "", errors.New(result.Message)
	}
	cfg := result.Message

	return cfg, nil
}

// DumpState fetch the whole state of the server (agent state, connected or not, server configuration, etc.)
func (api *API) DumpState() (commontypes.Status, error) {
	res, err := api.get("/admin/state/")
	if err != nil {
		return commontypes.Status{}, err
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return commontypes.Status{}, err
	}
	if res.StatusCode != http.StatusOK {
		return commontypes.Status{}, HandleError(body)
	}

	var result commontypes.Status
	err = yaml.Unmarshal(body, &result)
	if err != nil {
		return commontypes.Status{}, err
	}

	return result, nil
}

// GetClipboard deprecated.
func (api *API) GetClipboard(id string, password string) (string, error) {
	u := fmt.Sprintf("/manage/agent/%s/getClipboard", id)
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("error generating hash from password")
	}
	req := socketio.ClipboardMessage{
		HashPassword: string(hashedPwd),
	}
	jsBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	res, err := api.post(u, bytes.NewBuffer(jsBody))
	if err != nil {
		return "", fmt.Errorf("error sending request (%s)", err.Error())
	}
	//nolint:errcheck
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response (%s)", res.Status)
	}
	if res.StatusCode != http.StatusOK {
		return "", HandleError(body)
	}
	var result socketio.ClipboardMessage
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling response (%s)", res.Status)
	}

	return result.Content, result.ErrorMsg
}

// SetClipboard deprecated.
func (api *API) SetClipboard(id string, password string, clipboard string) error {
	u := fmt.Sprintf("/manage/agent/%s/setClipboard", id)
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error generating password (%s)", clipboard)
	}
	req := socketio.ClipboardMessage{
		HashPassword: string(hashedPwd),
		Content:      clipboard,
	}
	jsBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling request body (%s)", clipboard)
	}
	res, err := api.post(u, bytes.NewBuffer(jsBody))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response (%s)", res.Status)
	}
	if res.StatusCode != http.StatusOK {
		return HandleError(body)
	}

	return nil
}
