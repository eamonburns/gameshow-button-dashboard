package tui

import (
	"fmt"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/eamonburns/gameshow-button-dashboard/internal/webhook"
)

type model struct {
	buttonId  uint
	webhookCh <-chan webhook.Data

	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

func initialModel(webhookCh <-chan webhook.Data) model {
	return model{
		webhookCh: webhookCh,

		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return m.waitForWebhook()
}

type tickMsg time.Time

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case webhook.Data:
		m.buttonId = msg.ButtonId
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t) // Send message to start waiting again
		})

	case tickMsg:
		return m, m.waitForWebhook()

	// Is it a key press?
	case tea.KeyPressMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the space bar toggle the selected state
		// for the item that the cursor is pointing at.
		case "enter", "space":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() tea.View {
	// The header
	s := fmt.Sprintf("(buttonId: %d) ", m.buttonId)
	s += "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

// Create a Cmd that will send a Msg of type webhook.Data after the webserver
// receives a webhook request
func (m model) waitForWebhook() tea.Cmd {
	return func() tea.Msg {
		log.Printf("waiting for webhook...")
		data := <-m.webhookCh
		log.Printf("got webhook data")
		return data
	}
}
