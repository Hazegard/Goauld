package tui

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"Goauld/client/api"
	"Goauld/client/types"
	"Goauld/common/log"

	teatable "github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

const (
	// actionDelete delete keybind.
	actionDelete = "ctrl+d"
	// actionKill kill keybind.
	actionKill = "ctrl+k"
	// actionReset reset keybind.
	actionReset = "ctrl+r"
	// actionVSCode vscode keybind.
	actionVSCode = "ctrl+e"
	// actionEnter enter keybind.
	actionEnter = "enter"
	// actionPlus plus keybind.
	actionPlus = "+"
)

var (
	textError   = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	textWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	textOk      = lipgloss.NewStyle().Foreground(lipgloss.Color("190"))
	textHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
)

type tickMessage time.Time

type cmdResponse struct {
	Success bool
	Message string
	Action  string
}

type updateMessage struct {
	agents       []types.Agent
	ErrorMessage string
}

type promptPassword struct{}

var baseStyle = lipgloss.NewStyle().
	BorderForeground(lipgloss.Color("240"))

// NewTui returns the Model holding the TUI.
func NewTui(apiClient *api.API, agentPwd map[string]string, auditMide bool) Model {
	ti := textinput.New()
	ti.Width = 100

	agents, err := apiClient.GetAgents()
	if err != nil {
		log.Error().Err(err).Str("Mode", "TUI").Msg("unable to fetch agents")
		os.Exit(1)

		return Model{}
	}
	m := Model{
		agentsTable:    GenerateAgentTable(false).WithRows(AgentsToRow(agents, false, auditMide)),
		agents:         agents,
		statusText:     ti,
		api:            apiClient,
		_agentPassword: agentPwd,
		auditMode:      auditMide,
	}
	if len(m.agents) == 0 {
		m.agentInfoTable = m.GenerateInfoTable(types.Agent{})

		return m
	}
	m.agentInfoTable = m.GenerateInfoTable(agents[0])

	return m
}

func MaskString(s string, hide bool) string {
	if !hide {
		return s
	}
	n := len(s)
	if n == 0 {
		return s
	}

	switch {
	case n < 4:
		// Show first 3 (or all if shorter)
		if n < 3 {
			return s
		}

		return s[:3]
	case n < 6:
		// Show first 2 and last 2
		return s[:2] + strings.Repeat("*", n-4) + s[n-2:]
	default:
		// Show first 3 and last 3
		return s[:3] + strings.Repeat("*", n-6) + s[n-3:]
	}
}

// Model holds the TUI model.
type Model struct {
	api             *api.API
	agentsTable     table.Model
	agents          []types.Agent
	agentInfoTable  teatable.Model
	statusText      textinput.Model
	confirmAction   string
	agent           string
	_agentPassword  map[string]string
	ti              textinput.Model
	askingPwd       bool
	password        string
	promptedAction  string
	extendedDetails bool
	execMode        string
	auditMode       bool
}

// MaskString replaces all characters in a string with '*'
// except the first and last few based on string length.
func (m *Model) MaskString(s string) string {
	return MaskString(s, m.auditMode)
}

// Run run the Model.
func (m *Model) Run() (string, string, error) {
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return "", "", err
	}

	return m.agent, m.execMode, nil
}

// Init initialize the TUI model.
func (m *Model) Init() tea.Cmd { return m.doTick() }

// Update follow bubble tea mechanism.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var agentsTableCmd tea.Cmd
	var agentInfoCmd tea.Cmd
	var statusTextCmd tea.Cmd
	var doUpdateStatus bool
	var text string

	var batch []tea.Cmd

	// If the selected agent is not empty, set it to be used below
	var selectedAgent types.Agent
	selRow := m.agentsTable.HighlightedRow()
	data := selRow.Data["N"]
	if data != nil {
		d, ok := data.(string)
		if ok {
			// We trim spaces as they might be added du to the padding (centering in the column).
			id, err := strconv.Atoi(strings.TrimSpace(d))
			if err == nil {
				// fmt.Println(len(m.agents))
				if len(m.agents) == 0 {
					selectedAgent = types.Agent{}
				} else {
					selectedAgent = m.agents[id-1]
				}
			}
		}
	}

	if m.askingPwd {
		var cmd tea.Cmd

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				m.promptedAction = ""
				m.ti.Placeholder = ""
				m.askingPwd = false

				return m, nil
			case actionEnter:
				m.password = m.ti.Value()
				m.askingPwd = false
				switch m.promptedAction {
				case actionKill:
					return m, m.Kill(selectedAgent, true, false, m.password)
				case actionReset:
					return m, m.Kill(selectedAgent, false, false, m.password)
				case actionDelete:
					return m, m.Kill(selectedAgent, true, true, m.password)
				}
			}
		default:
			m.ti, cmd = m.ti.Update(msg)

			return m, cmd
		}
		m.ti, cmd = m.ti.Update(msg)

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// ctrl+k: shortcut to kill the agent
		case actionKill:
			// if the selected agent is not empty
			switch m.confirmAction {
			case "":
				m.confirmAction = actionKill
				text = fmt.Sprintf("Confirm killing %s? (%s to confirm)", m.MaskString(selectedAgent.Name), actionKill)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			case actionKill:
				if selectedAgent.ID != "" {
					m.confirmAction = ""
					// if selectedAgent.Connected {
					text = fmt.Sprintf("Killing %s (%s)...", m.MaskString(selectedAgent.Name), m.MaskString(selectedAgent.ID))
					m.statusText.TextStyle = textError
					if selectedAgent.HasStaticPassword {
						pwd, ok := m._agentPassword[selectedAgent.Name]
						if ok {
							batch = append(batch, m.Kill(selectedAgent, true, false, pwd))
						} else {
							m.promptedAction = actionKill
							batch = append(batch, func() tea.Msg {
								return promptPassword{}
							})
						}
					} else {
						batch = append(batch, m.Kill(selectedAgent, true, false, ""))
					}
					m.statusText.SetValue(text)
				}
				m.confirmAction = ""
			default:
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case actionReset:
			// if the selected agent is not empty
			switch m.confirmAction {
			case "":
				m.confirmAction = actionReset
				text = fmt.Sprintf("Confirm reset %s? (%s to confirm)", m.MaskString(selectedAgent.Name), actionReset)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			case actionReset:
				if selectedAgent.ID != "" {
					m.confirmAction = actionReset
					// if selectedAgent.Connected {
					text = fmt.Sprintf("Resetting %s (%s)...", m.MaskString(selectedAgent.Name), selectedAgent.ID)
					m.statusText.TextStyle = textError
					if selectedAgent.HasStaticPassword {
						pwd, ok := m._agentPassword[selectedAgent.Name]
						if ok {
							batch = append(batch, m.Kill(selectedAgent, false, false, pwd))
						} else {
							m.promptedAction = actionReset
							batch = append(batch, func() tea.Msg {
								return promptPassword{}
							})
						}
					} else {
						batch = append(batch, m.Kill(selectedAgent, false, false, ""))
					}

					m.statusText.SetValue(text)
				}
				m.confirmAction = ""
			default:
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case actionDelete:
			// if the selected agent is not empty
			switch m.confirmAction {
			case "":
				m.confirmAction = actionDelete
				text = fmt.Sprintf("Confirm deleting %s? (%s to confirm)", m.MaskString(selectedAgent.Name), actionDelete)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			case actionDelete:
				if selectedAgent.ID != "" {
					m.confirmAction = actionDelete
					text = fmt.Sprintf("Deleting %s (%s)...", m.MaskString(selectedAgent.Name), m.MaskString(selectedAgent.ID))
					m.statusText.TextStyle = textError
					if selectedAgent.HasStaticPassword && selectedAgent.Connected {
						pwd, ok := m._agentPassword[selectedAgent.Name]
						if ok {
							batch = append(batch, m.Kill(selectedAgent, true, true, pwd))
						} else {
							m.promptedAction = actionDelete
							batch = append(batch, func() tea.Msg {
								return promptPassword{}
							})
						}
					} else {
						batch = append(batch, m.Kill(selectedAgent, true, true, ""))
					}

					m.statusText.SetValue(text)
				}
				m.confirmAction = ""
			default:
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case actionEnter:
			if m.askingPwd {
				m.password = m.ti.Value()
				m.askingPwd = false
				switch m.promptedAction {
				case actionKill:
					batch = append(batch, m.Kill(selectedAgent, true, false, m.password))
				case actionReset:
					batch = append(batch, m.Kill(selectedAgent, false, false, m.password))
				case actionDelete:
					batch = append(batch, m.Kill(selectedAgent, true, true, m.password))
				}
			} else {
				m.execMode = "ssh"
				m.agent = selectedAgent.Name

				return m, tea.Quit
			}

		case actionVSCode:
			// if the selected agent is not empty
			switch m.confirmAction {
			case "":
				m.confirmAction = actionVSCode
				text = fmt.Sprintf("Confirm launching remote VSCode on %s? (%s to confirm)", m.MaskString(selectedAgent.Name), actionVSCode)
				m.statusText.TextStyle = textWarning
				m.statusText.SetValue(text)
			case actionVSCode:
				if selectedAgent.ID != "" {
					m.confirmAction = actionVSCode
					text = fmt.Sprintf("Starting remote VSCode on %s (%s)...", m.MaskString(selectedAgent.Name), m.MaskString(selectedAgent.ID))
					m.statusText.TextStyle = textOk
					m.execMode = "vscode"
					m.agent = selectedAgent.Name
					m.statusText.SetValue(text)

					return m, tea.Quit
				}
				m.confirmAction = ""
			default:
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
		// r: shortcut to update the agent list
		case "r":
			batch = append(batch, m.doUpdate(m.agents))
			// return m, m.UpdateAgents(m.agents)
		case actionPlus:
			m.extendedDetails = !m.extendedDetails
			m.agentsTable = GenerateAgentTable(m.extendedDetails).WithRows(AgentsToRow(m.agents, m.extendedDetails, m.auditMode))
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			m.confirmAction = ""
			m.statusText.SetValue("")
			doUpdateStatus = true
		}
	// If the message is a response of an API call
	case cmdResponse:
		switch msg.Action {
		// Handle kill api call
		case "kill", "delete":
			if msg.Success {
				text = msg.Message
				m.statusText.TextStyle = textOk
			} else {
				text = msg.Message
				m.statusText.TextStyle = textWarning
			}
			m.statusText.SetValue(text)
		}
	// Handle updates agent list
	case updateMessage:
		rows := AgentsToRow(msg.agents, m.extendedDetails, m.auditMode)
		m.agents = msg.agents
		m.agentsTable = m.agentsTable.WithRows(rows)

		if msg.ErrorMessage != "" {
			m.statusText.SetValue(msg.ErrorMessage)
			m.statusText.TextStyle = textWarning
		}
	case tickMessage:
		batch = append(batch, m.doUpdate(m.agents), m.doTick())
	case promptPassword:
		ti := textinput.New()
		ti.Prompt = fmt.Sprintf("(%s) Password: ", m.MaskString(selectedAgent.Name))
		ti.Placeholder = ""
		ti.EchoMode = textinput.EchoPassword
		ti.PromptStyle = textWarning
		m.ti = ti
		m.askingPwd = true

		return m, m.ti.Focus()
	}

	// Finalize the update mechanism

	if doUpdateStatus {
		m.statusText, statusTextCmd = m.statusText.Update(msg)
		batch = append(batch, m.doUpdate(m.agents))
		batch = append(batch, statusTextCmd)
	}
	m.agentInfoTable, agentInfoCmd = m.agentInfoTable.Update(msg)
	if agentInfoCmd != nil {
		batch = append(batch, agentInfoCmd)
	}
	m.agentsTable, agentsTableCmd = m.agentsTable.Update(msg)
	if agentsTableCmd != nil {
		batch = append(batch, agentsTableCmd)
	}

	if selectedAgent.ID != "" {
		m.agentInfoTable = m.GenerateInfoTable(selectedAgent)
	}
	var seq tea.Cmd
	if batch != nil {
		seq = tea.Sequence(batch...)
	}

	return m, seq
}

// UpdateAgents return a tea.Cmd used to update the agent list
// If the update fails, it returns the previous list to keep
// the table populated.
func (m *Model) UpdateAgents(prevAgents []types.Agent) tea.Cmd {
	return func() tea.Msg {
		return m.doUpdate(prevAgents)
	}
}

// doUpdate performs the update mechanism.
func (m *Model) doUpdate(prevAgents []types.Agent) func() tea.Msg {
	return func() tea.Msg {
		agents, err := m.api.GetAgents()
		if err != nil {
			return updateMessage{
				agents:       prevAgents,
				ErrorMessage: err.Error(),
			}
		}

		return updateMessage{
			agents:       agents,
			ErrorMessage: "",
		}
	}
}

func (m *Model) help() string {
	return textHelp.SetString(
		"   [ctrl+r]:Reset agent    [ctrl+d]:Delete agent    [↑]:Up      [←]:Previous    [r]:Refresh view    [q]/[ctrl+c]:Quit" +
			"\n   [ctrl+k]:Kill agent     [Enter]: SSH agent       [↓]:Down    [→]:Next        [ctrl+e] VsCode     [+]Details").String()
}

// doTick handle the periodic update.
func (m *Model) doTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMessage(t)
	})
}

// GenerateInfoTable populate the info table to show the details of the currently selected agent.
func (m *Model) GenerateInfoTable(agent types.Agent) teatable.Model {
	ips := agent.IPs
	if m.auditMode {
		split := strings.Split(ips, ",")
		var _ips []string
		for _, ip := range split {
			ipRange := strings.Split(ip, "/")
			if len(ipRange) == 1 {
				_ips = append(_ips, m.MaskString(ipRange[0])+"/32")
			} else {
				_ips = append(_ips, m.MaskString(ipRange[0])+"/"+ipRange[1])
			}
		}
		ips = strings.Join(_ips, ",")
	}
	rows := []teatable.Row{
		{"ID", m.MaskString(agent.ID)},
		{"Username", m.MaskString(agent.Username)},
		{"Hostname", m.MaskString(agent.Hostname)},
		{"Path", m.MaskString(agent.Path)},
		{"Version", agent.Version.String()},
		{"OS", agent.Platform},
		{"Archi", agent.Architecture},
		{"Public Ip", m.MaskString(agent.RemoteAddr)},
		{"IPs", strings.ReplaceAll(ips, ",", ", ")},
	}
	height := len(rows) + 1
	if m.extendedDetails {
		agentRelay, ok := GetRelay(agent, m.agents)
		if !ok {
			rows = append(rows, teatable.Row{"Relay", "/"})
		} else {
			rows = append(rows, teatable.Row{"Relay", agentRelay.Name + " (" + agentRelay.ID + ")"})
		}
		height++
		var portRelay string
		if agent.GetRelayPort() == "/" {
			portRelay = "/"
		} else {
			portRelay = fmt.Sprintf("(%s)", agent.GetRelayPort())
		}
		details := []teatable.Row{
			{"Last Updated", absTime(agent.LastUpdated)},
			{"Last Ping", absTime(agent.LastPing)},
			{"SSH Mode", agent.SSHMode},
			{"SSHD Port", agent.GetSSHPort()},
			{"Socks Port", agent.GetSocksPort()},
			{"HTTP Port", agent.GetHTTPPort()},
			{"WG Port", agent.GetWGPort()},
			{"Relay Port", portRelay},
			{"HTTP MITP Port", agent.GetHTTPMITMPort()},
			{"Other Port", agent.GetOtherPort()},
		}
		rows = append(rows, details...)
		height += len(details)
	}

	// Compute the longest field that will be shown on the table
	length := 0
	lines := []string{
		agent.ID,
		agent.Version.String(),
		agent.Platform,
		agent.Architecture,
		agent.RemoteAddr,
		agent.Username,
		agent.Hostname,
		absTime(agent.LastUpdated),
		absTime(agent.LastPing),
		strings.ReplaceAll(agent.IPs, ",", ", "),
		agent.Path,
		agent.SSHMode,
		agent.GetSSHPort(),
		agent.GetSocksPort(),
		agent.GetHTTPPort(),
		agent.GetOtherPort(),
		agent.GetHTTPMITMPort(),
		agent.SSHPasswd,
	}
	for _, v := range lines {
		length = max(length, len(v))
	}
	s := teatable.DefaultStyles()
	s.Header = s.Header.
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)

	columns := []teatable.Column{
		{Title: "Name", Width: 15},
		{Title: m.MaskString(agent.Name), Width: length},
	}
	t := teatable.New(
		teatable.WithColumns(columns),
		teatable.WithRows(rows),
		teatable.WithFocused(false),
		teatable.WithHeight(height),
	)

	t.SetStyles(s)

	return t
}

// View return a string to print the model to the terminal.
func (m *Model) View() string {
	res := baseStyle.Render(m.statusText.View()) + "\n" + baseStyle.Render(m.agentInfoTable.View()) + "\n" + baseStyle.Render(m.agentsTable.View()) + "\n" + m.help() + "\n"
	if m.askingPwd {
		res += m.ti.View()
	}

	return res
}

// centerString adds left padding to center the string in the column given the column length.
func centerString(str string, length int) string {
	// If the string is already longer than or equal to the required length, return it as is.
	if len(str) >= length {
		return str
	}

	// Calculate how many spaces need to be added to the left and right.
	totalPadding := length - len(str)
	leftPadding := totalPadding / 2
	// rightPadding := totalPadding - leftPadding

	// Create the padded string.
	return strings.Repeat(" ", leftPadding) + str
}

// NewCenterColumn return a column with its title centered.
func NewCenterColumn(key string, title string, width int) table.Column {
	return table.NewColumn(key, centerString(title, width), width)
}

// GenerateAgentTable initialize the agent table.
func GenerateAgentTable(detail bool) table.Model {
	columns := []table.Column{
		NewCenterColumn("N", "N", 3),
		NewCenterColumn("Name", "Name", 30),
		NewCenterColumn("Last Updated", "Last Updated", 14),
		NewCenterColumn("Last Ping", "Last Ping", 13),
		NewCenterColumn("Mode", "Mode", 10),
		NewCenterColumn("SSHD Port", "SSHD Port", 13),
		NewCenterColumn("HTTP Port", "HTTP Port", 13),
		NewCenterColumn("Socks Port", "Socks Port", 14),
	}
	if detail {
		columns = append(columns, NewCenterColumn("WG Port", "WG Port", 13))
		columns = append(columns, NewCenterColumn("Relay Port", "Relay Port", 13))
		columns = append(columns, NewCenterColumn("HTTP MITM Port", "HTTP MITM Port", 16))
	}
	t := table.New(columns).
		Focused(true).
		WithBaseStyle(baseStyle).
		WithRowStyleFunc(func(input table.RowStyleFuncInput) lipgloss.Style {
			row := input.Row.Data
			s := input.Row.Style
			//nolint:forcetypeassert
			mode := row["Mode"].(string)
			mode = strings.TrimSpace(mode)
			if mode == "OFF" || mode == "" {
				s = s.Foreground(lipgloss.Color("245"))
			}
			if input.IsHighlighted {
				s = s.Background(lipgloss.Color("240"))
			}

			return s
		}).
		WithPageSize(20)

	return t
}

// GetAgents calls the API and returns the slice of the agents.
func (m *Model) GetAgents() []table.Row {
	agents, err := m.api.GetAgents()
	if err != nil {
		panic(err)
	}

	return AgentsToRow(agents, m.extendedDetails, m.auditMode)
}

// AgentsToRow converts a slice of agents to a slice of rows to be used in the table component.
func AgentsToRow(agents []types.Agent, details bool, hide bool) []table.Row {
	var rows []table.Row
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].LastUpdated.After(agents[j].LastUpdated)
	})
	for i, agent := range agents {
		data := table.RowData{
			"ID":           agent.ID,
			"N":            centerString(strconv.Itoa(i+1), 3),
			"Name":         centerString(MaskString(agent.Name, hide), 30),
			"Last Updated": centerString(relTIme(agent.LastUpdated), 14),
			"Last Ping":    centerString(relTIme(agent.LastPing), 13),
			"Mode":         centerString(agent.SSHMode, 10),
			"SSHD Port":    centerString(agent.GetSSHPort(), 13),
			"Socks Port":   centerString(agent.GetSocksPort(), 13),
			"HTTP Port":    centerString(agent.GetHTTPPort(), 14),
			"Other Port":   agent.GetOtherPort(),
		}
		if details {
			data["WG Port"] = centerString(agent.GetWGPort(), 13)
			relayPort := agent.GetRelayPort()
			if relayPort == "/" {
				data["Relay Port"] = centerString(agent.GetRelayPort(), 13)
			} else {
				data["Relay Port"] = centerString(fmt.Sprintf("(%s)", agent.GetRelayPort()), 13)
			}
			data["HTTP MITM Port"] = centerString(agent.GetHTTPMITMPort(), 15)
		}
		row := table.NewRow(data)

		rows = append(rows, row)
	}

	return rows
}

// Kill performs a call to the API to kill the selected agent.
func (m *Model) Kill(agent types.Agent, doExit bool, doDelete bool, password string) tea.Cmd {
	killing := "resetting"
	reset := "Reset"
	if doExit {
		killing = "killing"
		reset = "Killed"
	}

	return func() tea.Msg {
		err := m.api.KillAgent(agent.ID, doExit, doDelete, password)
		if err != nil {
			log.Error().Err(err).Str("agent", agent.ID).Msg("failed to kill agent")

			return cmdResponse{
				Success: false,
				Message: fmt.Sprintf("Error %s %s (%s): %s", killing, m.MaskString(agent.Name), agent.ID, err),
				Action:  "kill",
			}
		}

		return cmdResponse{
			Success: false,
			Message: fmt.Sprintf("%s %s (%s)", reset, m.MaskString(agent.Name), agent.ID),
			Action:  "kill",
		}
	}
}

// Delete performs a call to the API to delete the selected agent.
func (m *Model) Delete(agent types.Agent) tea.Cmd {
	return func() tea.Msg {
		err := m.api.DeleteAgent(agent.ID)
		if err != nil {
			return cmdResponse{
				Success: false,
				Message: fmt.Sprintf("Error deleting %s (%s): %s", m.MaskString(agent.Name), agent.ID, err),
				Action:  "delete",
			}
		}

		return cmdResponse{
			Success: false,
			Message: fmt.Sprintf("Deleted %s (%s)", agent.Name, agent.ID),
			Action:  "delete",
		}
	}
}

func absTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// relTIme converts a date to XXX times ago.
func relTIme(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	switch {
	case duration.Seconds() < 60:
		return fmt.Sprintf("%.0f sec ago", duration.Seconds())
	case duration.Minutes() < 60:
		return fmt.Sprintf("%.0f min ago", duration.Minutes())
	case duration.Hours() < 24:
		return fmt.Sprintf("%.0f hours ago", duration.Hours())
	case duration.Hours() < 24*30:
		return fmt.Sprintf("%.0f days ago", duration.Hours()/24)
	case duration.Hours() < 24*365:
		return fmt.Sprintf("%.0f months ago", duration.Hours()/(24*30))
	default:
		return fmt.Sprintf("%.0f years ago", duration.Hours()/(24*365))
	}
}

func GetRelay(agent types.Agent, agents []types.Agent) (types.Agent, bool) {
	split := strings.Split(agent.Relay, ":")
	if len(split) != 2 {
		return types.Agent{}, false
	}
	ipStr := split[0]
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return types.Agent{}, false
	}
	port := split[1]
	for _, ag := range agents {
		if ag.GetRelayPort() == port {
			if ip.IsLoopback() {
				return ag, true
			}
			if strings.Contains(ipStr, ag.IPs) {
				return ag, true
			}
		}
	}

	return types.Agent{}, false
}
