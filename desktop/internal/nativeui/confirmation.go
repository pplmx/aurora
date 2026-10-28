package nativeui

import "github.com/egoist/mygo/ui"

// destructiveConfirm shows an alert-style dialog whose initial focus is the
// safe choice. MyGo's AlertDialog focuses its last button, so using it for
// Burn or Delete made a stray Enter perform the irreversible write. The red
// action remains keyboard reachable after one Tab.
func destructiveConfirm(c *ui.Context, open *bool, title, detail, confirmLabel string) bool {
	if !*open {
		return false
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
	return confirmed
}
