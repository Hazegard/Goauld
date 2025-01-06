package tui

import (
	"Goauld/client/api"
	"Goauld/client/types"
	"fmt"
	teatable "github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"sort"
	"strconv"
	"time"
)

const (
	action_delete = "ctrl+d"
	action_kill   = "ctrl+k"
)

var (
	textError   = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	textWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	textOk      = lipgloss.NewStyle().Foreground(lipgloss.Color("190"))
	textHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
)

type CmdResponse struct {
	Success bool
	Message string
	Action  string
}

type UpdateMessage struct {
	tick         bool
	agents       []types.Agent
	ErrorMessage string
}

var baseStyle = lipgloss.NewStyle().
	BorderForeground(lipgloss.Color("240"))

func NewTui(api *api.API) Model {

	ti := textinput.New()
	ti.Width = 100

	agents, err := api.GetAgents()
	if err != nil {
		fmt.Println(err)
		return Model{}
	}
	fmt.Printf("%+v\n", agents)
	m := Model{
		agentsTable: GenerateAgentTable().WithRows(AgentsToRow(agents)),
		agents:      agents,
		statusText:  ti,
		api:         api,
	}
	m.agentInfoTable = m.GenerateInfoTable(agents[0])

	return m
}

func (m Model) Run() error {
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}

type Model struct {
	api            *api.API
	agentsTable    table.Model
	agents         []types.Agent
	agentInfoTable teatable.Model
	statusText     textinput.Model
	confirmAction  string
}

func (m Model) Init() tea.Cmd { return m.doTick(m.agents) }

// Update follow bubble tea mechanism
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var agentsTableCmd tea.Cmd
	var agentInfoCmd tea.Cmd
	var statusTextCmd tea.Cmd
	var doUpdateStatus bool
	var text string

	var batch []tea.Cmd

	// If the selected agent is not empty, set it to be uased below
	var selectedAgent types.Agent
	selRow := m.agentsTable.HighlightedRow()
	data := selRow.Data["N"]
	if data != nil {
		id, err := strconv.Atoi(selRow.Data["N"].(string))
		if err == nil {
			selectedAgent = m.agents[id-1]
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		// ctrl+k: shortcut to kill the agent
		case action_kill:
			// if selected agent is not empty
			if m.confirmAction == "" {
				m.confirmAction = action_kill
				text = fmt.Sprintf("Confirm killing %s? (%s to confirm)", selectedAgent.Name, action_kill)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			} else if m.confirmAction == action_kill {
				if selectedAgent.Id != "" {
					m.confirmAction = action_kill
					if selectedAgent.Connected {
						text = fmt.Sprintf("Killing %s (%s)...", selectedAgent.Name, selectedAgent.Id)
						m.statusText.TextStyle = textError
						batch = append(batch, m.Kill(selectedAgent))
					} else {
						text = fmt.Sprintf("Already killed %s (%s)", selectedAgent.Name, selectedAgent.Id)
						m.statusText.TextStyle = textWarning
					}
					m.statusText.SetValue(text)
				}
			} else {
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true
		case action_delete:
			// if selected agent is not empty
			if m.confirmAction == "" {
				m.confirmAction = action_delete
				text = fmt.Sprintf("Confirm deleting %s? (%s to confirm)", selectedAgent.Name, action_delete)
				m.statusText.TextStyle = textError
				m.statusText.SetValue(text)
			} else if m.confirmAction == action_delete {
				if selectedAgent.Id != "" {
					m.confirmAction = action_delete
					if selectedAgent.Connected {
						text = fmt.Sprintf("Deleting %s (%s)...", selectedAgent.Name, selectedAgent.Id)
						m.statusText.TextStyle = textError
						batch = append(batch, m.Delete(selectedAgent))
					} else {
						text = fmt.Sprintf("Already deleted %s (%s)", selectedAgent.Name, selectedAgent.Id)
						m.statusText.TextStyle = textWarning
					}
					m.statusText.SetValue(text)
				}
			} else {
				m.confirmAction = ""
				m.statusText.SetValue("")
			}
			doUpdateStatus = true

		// ctrl+r: shortcut to update the agent list
		case "ctrl+r":
			return m, m.UpdateAgents(m.agents)
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
	// Handle update agent list
	case UpdateMessage:
		rows := AgentsToRow(msg.agents)
		m.agentsTable = m.agentsTable.WithRows(rows)

		m.statusText.SetValue(msg.ErrorMessage)
		m.statusText.TextStyle = textWarning
		doUpdateStatus = true
		// Return your Tick command again to loop.
		if msg.tick {
			batch = append(batch, m.doTick(m.agents))
		}
	}

	// Finalize the update mechanism

	if doUpdateStatus {
		m.statusText, statusTextCmd = m.statusText.Update(msg)
		batch = append(batch, statusTextCmd)
	}

	if selectedAgent.Id != "" {
		m.agentInfoTable = m.GenerateInfoTable(selectedAgent)
	}
	m.agentInfoTable, agentInfoCmd = m.agentInfoTable.Update(msg)
	m.agentsTable, agentsTableCmd = m.agentsTable.Update(msg)
	batch = append(batch, agentsTableCmd)
	batch = append(batch, agentInfoCmd)
	return m, tea.Sequence(batch...)
}

// UpdateAgents return a tea.Cmd used to update the agent list
// If the update fails, it returns the previous list to keep
// the table populated
func (m Model) UpdateAgents(prevAgents []types.Agent) tea.Cmd {
	return func() tea.Msg {
		return m.doUpdate(prevAgents)
	}
}

// doUpdate performs the update mechanism
func (m Model) doUpdate(prevAgents []types.Agent) UpdateMessage {
	agents, err := m.api.GetAgents()
	if err != nil {
		return UpdateMessage{
			agents:       prevAgents,
			tick:         true,
			ErrorMessage: err.Error(),
		}
	}
	return UpdateMessage{
		agents:       agents,
		tick:         false,
		ErrorMessage: "",
	}
}

func (m Model) Help() string {
	return textHelp.SetString("  [↑]:Up [↓]:Down  [←]:Previous  [→]:Next  [q]/[ctrl+c]:Quit  [ctrl+k]:Kill agent  [ctrl+d]:Delete agent  ").String()
}

// doTick handle the periodic update
func (m Model) doTick(prevAgents []types.Agent) tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		msg := m.doUpdate(prevAgents)
		msg.tick = true
		return msg
	})
}

// GenerateInfoTable populate the info table to show the details of the currently selected agent
func (m Model) GenerateInfoTable(agent types.Agent) teatable.Model {
	rows := []teatable.Row{
		{"Id", agent.Id},
		{"OS", agent.Platform},
		{"Archi", agent.Architecture},
		{"Username", agent.Username},
		{"Hostname", agent.Hostname},
		{"IPs", agent.IPs},
		{"Path", agent.Path},
		{"Ssh Mode", agent.SshMode},
		{"SSHD Port", agent.GetSSHPort()},
		{"Socks Ports", agent.GetSocksPort()},
		{"Other Ports", agent.GetOtherPort()},
	}

	// Compute the longest field that will be shown on the table
	length := 0
	lines := []string{
		agent.Id,
		agent.Platform,
		agent.Architecture,
		agent.Username,
		agent.Hostname,
		agent.IPs,
		agent.Path,
		agent.SshMode,
		agent.GetSSHPort(),
		agent.GetSocksPort(),
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
		{Title: "Name", Width: 10},
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

func (m Model) View() string {
	return baseStyle.Render(m.statusText.View()) + "\n" + baseStyle.Render(m.agentInfoTable.View()) + "\n" + baseStyle.Render(m.agentsTable.View()) + "\n" + m.Help() + "\n"
}

// GenerateAgentTable initialize the agent table
func GenerateAgentTable() table.Model {
	columns := []table.Column{
		// {Title: "ID", Width: 32},
		table.NewColumn("N", "N", 3),
		table.NewColumn("Name", "Name", 30),
		table.NewColumn("Last updates", "Last updates", 15),
		table.NewColumn("Mode", "Mode", 10),
		table.NewColumn("SSHD Port", "SSHD Port", 15),
		table.NewColumn("Socks Ports", "Socks Ports", 15),
	}
	t := table.New(columns).
		Focused(true).
		WithBaseStyle(baseStyle).
		WithRowStyleFunc(func(input table.RowStyleFuncInput) lipgloss.Style {
			row := input.Row.Data
			s := input.Row.Style
			if row["Mode"] == "OFF" || row["Mode"] == "" {
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

// GetAgents call the API and returns the slice of the agents
func (m Model) GetAgents() []table.Row {
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
				"N":            strconv.Itoa(i + 1),
				"Name":         agent.Name,
				"Last updates": timeAgo(agent.LastUpdated),
				"Mode":         agent.SshMode,
				"SSHD Port":    " " + agent.GetSSHPort(),
				"Socks Ports":  " " + agent.GetSocksPort(),
				"Other Ports":  " " + agent.GetOtherPort(),
			})

		rows = append(rows, row)
	}

	return rows
}

// Kill performs a call to the API to kill the selected agent
func (m Model) Kill(agent types.Agent) tea.Cmd {
	return func() tea.Msg {
		err := m.api.KillAgent(agent.Id)
		if err != nil {
			return CmdResponse{
				Success: false,
				Message: fmt.Sprintf("Error killing %s (%s): %s", agent.Name, agent.Id, err),
				Action:  "kill",
			}
		}
		return CmdResponse{
			Success: false,
			Message: fmt.Sprintf("Killed %s (%s)", agent.Name, agent.Id),
			Action:  "kill",
		}
	}
}

// Delete performs a call to the API to delete the selected agent
func (m Model) Delete(agent types.Agent) tea.Cmd {
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
