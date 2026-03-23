package diagnose

import tea "github.com/charmbracelet/bubbletea"

// TUIConfirm returns a ConfirmFunc that bridges action tool confirmation
// through Bubble Tea's message loop. When an action tool needs confirmation,
// it sends an actionConfirmMsg to the TUI and blocks until the operator
// responds.
func TUIConfirm(send func(tea.Msg)) ConfirmFunc {
	return func(action, service, reason string) bool {
		respond := make(chan bool, 1)
		send(actionConfirmMsg{
			action:  action,
			service: service,
			reason:  reason,
			respond: respond,
		})
		return <-respond
	}
}
