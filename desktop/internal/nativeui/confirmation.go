package nativeui

import "github.com/egoist/mygo/ui"

type destructiveResult uint8

const (
	destructivePending destructiveResult = iota
	destructiveCancelled
	destructiveConfirmed
)

// destructiveConfirm shows an alert-style dialog whose initial focus is the
// safe choice. MyGo's AlertDialog focuses its last button, so using it for
// Burn or Delete made a stray Enter perform the irreversible write. The red
// action remains keyboard reachable after one Tab.
func destructiveConfirm(c *ui.Context, open *bool, title, detail, confirmLabel string) destructiveResult {
	if !*open {
		return destructivePending
	}
	confirmed := false
	dialog := ui.Modal(c, open, func() {
		heading(c, title, 17, 2)
		ui.Text(c, detail).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
			if secondaryButton(c, "Cancel").AutoFocus().Clicked() {
				*open = false
			}
			if dangerButton(c, confirmLabel).Clicked() {
				*open = false
				confirmed = true
			}
		})
	})
	dialog.MinWidth(360 * textScale(c)).
		Role(ui.RoleAlertDialog).Label(title).Description(detail)
	switch {
	case confirmed:
		return destructiveConfirmed
	case !*open:
		return destructiveCancelled
	default:
		return destructivePending
	}
}

// restoreDestructiveFocus returns the keyboard to the action that opened the
// dialog. A hidden trigger may not be built in this frame; requesting focus
// from an unbound handle would leave a pending request that fires unexpectedly
// later, so only focus a control present in the current pass.
func restoreDestructiveFocus(c *ui.Context, trigger ui.Handle) {
	if c.Resolve(trigger).Valid() {
		trigger.Focus(c)
	}
}
