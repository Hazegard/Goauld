package tui

import (
	"fmt"
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
	action_delete = "ctrl+d"
	action_kill   = "ctrl+k"
	action_reset  = "ctrl+r"
	action_ssh    = "enter"
)

var (
	textError   = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	textWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	textOk      = lipgloss.NewStyle().Foreground(lipgloss.Color("190"))
	textHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
)

type TickMessage time.Time

type CmdResponse struct {
	Success bool
	Message string
	Action  string
}

type UpdateMessage struct {
	agents       []types.Agent
	ErrorMessage string
}

var baseStyle = lipgloss.NewStyle().
	BorderForeground(lipgloss.Color("240"))

// NewTui returns the Model holding the TUI
func NewTui(api *api.API) Model {
	ti := textinput.New()
	ti.Width = 100

	agents, err := api.GetAgents()
	if err != nil {
		log.Error().Err(err).Str("Mode", "TUI").Msg("unable to fetch agents")
		os.Exit(1)
		return Model{}
	}
	m := Model{
		agentsTable: GenerateAgentTable().WithRows(AgentsToRow(agents)),
		agents:      agents,
		statusText:  ti,
		api:         api,
	}
	if len(m.agents) == 0 {
		m.agentInfoTable = m.GenerateInfoTable(types.Agent{})
		return m
	}
	m.agentInfoTable = m.GenerateInfoTable(agents[0])

	return m
}

func (m *Model) Run() (error, string) {
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err, ""
	}
	return nil, m.agent
}

type Model struct {
	api            *api.API
	agentsTable    table.Model
	agents         []types.Agent
	agentInfoTable teatable.Model
	statusText     textinput.Model
	confirmAction  string
	agent          string
}

func (m *Model) Init() tea.Cmd { return m.doTick() }

// Update follow bubble tea mechanism
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
		if ok == true {
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		// ctrl+k: shortcut to kill the agent
		case action_kill:
			// if the selected agent is not empty
			if m.confirmAction == "" {
				m.confirmAction = action_kill
				text = fmt.Sprintf("Confirm killing %s? (%s to confirm)", selectedAgent.Name, action_kill)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			} else if m.confirmAction == action_kill {
				if selectedAgent.Id != "" {
					m.confirmAction = ""
					// if selectedAgent.Connected {
					text = fmt.Sprintf("Killing %s (%s)...", selectedAgent.Name, selectedAgent.Id)
					m.statusText.TextStyle = textError
					batch = append(batch, m.Kill(selectedAgent, true))
					// } else {
					// 	text = fmt.Sprintf("Already killed %s (%s)", selectedAgent.Name, selectedAgent.Id)
					// 	m.statusText.TextStyle = textWarning
					// }
					m.statusText.SetValue(text)
				}
				m.confirmAction = ""
			} else {
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case action_reset:
			// if the selected agent is not empty
			if m.confirmAction == "" {
				m.confirmAction = action_reset
				text = fmt.Sprintf("Confirm reset %s? (%s to confirm)", selectedAgent.Name, action_reset)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			} else if m.confirmAction == action_reset {
				if selectedAgent.Id != "" {
					m.confirmAction = ""
					// if selectedAgent.Connected {
					text = fmt.Sprintf("Resetting %s (%s)...", selectedAgent.Name, selectedAgent.Id)
					m.statusText.TextStyle = textError
					batch = append(batch, m.Kill(selectedAgent, false))
					m.statusText.SetValue(text)
				}
				m.confirmAction = ""
			} else {
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case action_delete:
			// if the selected agent is not empty
			if m.confirmAction == "" {
				m.confirmAction = action_delete
				text = fmt.Sprintf("Confirm deleting %s? (%s to confirm)", selectedAgent.Name, action_delete)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			} else if m.confirmAction == action_delete {
				if selectedAgent.Id != "" {
					m.confirmAction = action_delete
					text = fmt.Sprintf("Deleting %s (%s)...", selectedAgent.Name, selectedAgent.Id)
					m.statusText.TextStyle = textError
					batch = append(batch, m.Delete(selectedAgent))
					m.statusText.SetValue(text)
				}
				m.confirmAction = ""
			} else {
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case action_ssh:
			m.agent = selectedAgent.Name
			return m, tea.Quit

		// r: shortcut to update the agent list
		case "r":
			batch = append(batch, m.doUpdate(m.agents))
			// return m, m.UpdateAgents(m.agents)
		default:
			m.confirmAction = ""
			m.statusText.SetValue("")
			doUpdateStatus = true
		}
	// If the message is a response of an API call
	case CmdResponse:
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
	case UpdateMessage:
		rows := AgentsToRow(msg.agents)
		m.agents = msg.agents
		m.agentsTable = m.agentsTable.WithRows(rows)

		if msg.ErrorMessage != "" {
			m.statusText.SetValue(msg.ErrorMessage)
			m.statusText.TextStyle = textWarning
			doUpdateStatus = true
		}
	case TickMessage:
		batch = append(batch, m.doUpdate(m.agents), m.doTick())
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

	if selectedAgent.Id != "" {
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
// the table populated
func (m *Model) UpdateAgents(prevAgents []types.Agent) tea.Cmd {
	return func() tea.Msg {
		return m.doUpdate(prevAgents)
	}
}

// doUpdate performs the update mechanism
func (m *Model) doUpdate(prevAgents []types.Agent) func() tea.Msg {
	return func() tea.Msg {
		agents, err := m.api.GetAgents()
		if err != nil {
			return UpdateMessage{
				agents:       prevAgents,
				ErrorMessage: err.Error(),
			}
		}
		return UpdateMessage{
			agents:       agents,
			ErrorMessage: "",
		}
	}
}

func (m *Model) Help() string {
	return textHelp.SetString(
		"    [ctrl+r]:Reset agent      [ctrl+d]:Delete agent      [↑]:Up        [←]:Previous      [r]:Refresh view" +
			"\n    [ctrl+k]:Kill agent       [Enter]: SSH agent         [↓]:Down      [→]:Next          [q]/[ctrl+c]:Quit").String()

}

// doTick handle the periodic update
func (m *Model) doTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TickMessage(t)
	})
}

// GenerateInfoTable populate the info table to show the details of the currently selected agent
func (m *Model) GenerateInfoTable(agent types.Agent) teatable.Model {

	rows := []teatable.Row{
		{"Id", agent.Id},
		{"OS", agent.Platform},
		{"Archi", agent.Architecture},
		{"Username", agent.Username},
		{"Hostname", agent.Hostname},
		{"Last Updated", timeAgo(agent.LastUpdated)},
		{"Last Ping", timeAgo(agent.LastPing)},
		{"IPs", agent.IPs},
		{"Path", agent.Path},
		{"Ssh Mode", agent.SshMode},
		{"SSHD Port", agent.GetSSHPort()},
		{"Socks Port", agent.GetSocksPort()},
		{"HTTP Port", agent.GetHttpPort()},
		{"Other Port", agent.GetOtherPort()},
	}

	// Compute the longest field that will be shown on the table
	length := 0
	lines := []string{
		agent.Id,
		agent.Platform,
		agent.Architecture,
		agent.Username,
		agent.Hostname,
		timeAgo(agent.LastUpdated),
		timeAgo(agent.LastPing),
		agent.IPs,
		agent.Path,
		agent.SshMode,
		agent.GetSSHPort(),
		agent.GetSocksPort(),
		agent.GetHttpPort(),
		agent.GetOtherPort(),
		agent.SshPasswd,
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
		{Title: "Name", Width: 12},
		{Title: agent.Name, Width: length},
	}
	t := teatable.New(
		teatable.WithColumns(columns),
		teatable.WithRows(rows),
		teatable.WithFocused(false),
		teatable.WithHeight(len(lines)),
	)

	t.SetStyles(s)
	return t
}

func (m *Model) View() string {
	return baseStyle.Render(m.statusText.View()) + "\n" + baseStyle.Render(m.agentInfoTable.View()) + "\n" + baseStyle.Render(m.agentsTable.View()) + "\n" + m.Help() + "\n"
}

// centerString adds left padding to center the string in the column given the column length
func centerString(str string, length int) string {
	// If the string is already longer than or equal to the required length, return it as is.
	if len(str) >= length {
		return str
	}

	// Calculate how many spaces need to be added to the left and right.
	totalPadding := length - len(str)
	leftPadding := totalPadding / 2
	//rightPadding := totalPadding - leftPadding

	// Create the padded string.
	return strings.Repeat(" ", leftPadding) + str
}

// NewCenterColumn return a column with its title centered
func NewCenterColumn(key string, title string, width int) table.Column {
	return table.NewColumn(key, centerString(title, width), width)
}

// GenerateAgentTable initialize the agent table
func GenerateAgentTable() table.Model {
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
	t := table.New(columns).
		Focused(true).
		WithBaseStyle(baseStyle).
		WithRowStyleFunc(func(input table.RowStyleFuncInput) lipgloss.Style {
			row := input.Row.Data
			s := input.Row.Style
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

// GetAgents calls the API and returns the slice of the agents
func (m *Model) GetAgents() []table.Row {
	agents, err := m.api.GetAgents()
	if err != nil {
		panic(err)
	}
	return AgentsToRow(agents)
}

// AgentsToRow converts a slice of agents to a slice of rows to be used in the table component
func AgentsToRow(agents []types.Agent) []table.Row {
	var rows []table.Row
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].LastUpdated.After(agents[j].LastUpdated)
	})
	for i, agent := range agents {
		row := table.NewRow(
			table.RowData{
				"Id":           agent.Id,
				"N":            centerString(strconv.Itoa(i+1), 3),
				"Name":         centerString(agent.Name, 30),
				"Last Updated": centerString(timeAgo(agent.LastUpdated), 14),
				"Last Ping":    centerString(timeAgo(agent.LastPing), 13),
				"Mode":         centerString(agent.SshMode, 10),
				"SSHD Port":    centerString(agent.GetSSHPort(), 13),
				"Socks Port":   centerString(agent.GetSocksPort(), 13),
				"HTTP Port":    centerString(agent.GetHttpPort(), 14),
				"Other Port":   agent.GetOtherPort(),
			})

		rows = append(rows, row)
	}

	return rows
}

// Kill performs a call to the API to kill the selected agent
func (m *Model) Kill(agent types.Agent, doExit bool) tea.Cmd {
	killing := "resetting"
	reset := "Reset"
	if doExit {
		killing = "killing"
		reset = "Killed"
	}
	return func() tea.Msg {
		err := m.api.KillAgent(agent.Id, doExit)
		if err != nil {
			return CmdResponse{
				Success: false,
				Message: fmt.Sprintf("Error %s %s (%s): %s", killing, agent.Name, agent.Id, err),
				Action:  "kill",
			}
		}
		return CmdResponse{
			Success: false,
			Message: fmt.Sprintf("%s %s (%s)", reset, agent.Name, agent.Id),
			Action:  "kill",
		}
	}
}

// Delete performs a call to the API to delete the selected agent
func (m *Model) Delete(agent types.Agent) tea.Cmd {
	return func() tea.Msg {
		err := m.api.DeleteAgent(agent.Id)
		if err != nil {
			return CmdResponse{
				Success: false,
				Message: fmt.Sprintf("Error deleting %s (%s): %s", agent.Name, agent.Id, err),
				Action:  "delete",
			}
		}
		return CmdResponse{
			Success: false,
			Message: fmt.Sprintf("Deleted %s (%s)", agent.Name, agent.Id),
			Action:  "delete",
		}
	}
}

// timeAgo converts a date to XXX times ago
func timeAgo(t time.Time) string {
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
