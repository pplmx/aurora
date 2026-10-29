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
//
// Both outcomes move focus back into the page: cancel returns to the trigger,
// while confirm returns to confirmFocus because that control may disappear or
// become disabled while the destructive write runs.
//
// The dialog leads with a warning mark in the danger hue, so an irreversible
// question never looks like the save/confirm dialogs around it: the shape and
// the colour agree with the red action at the end. The question keeps the
// dialog's accessible name and the detail its description, exactly as before;
// only the presentation changed. The footer's hairline separates the decision
// from the explanation without adding a second panel.
func destructiveConfirm(c *ui.Context, open *bool, title, detail, confirmLabel string,
	cancelFocus, confirmFocus ui.Handle,
) destructiveResult {
	if !*open {
		return destructivePending
	}
	confirmed := false
	danger := ui.Hex(auroraRed)
	dialog := ui.Modal(c, open, func() {
		ui.Row(c).FillWidth().Gap(space3).AlignItems(ui.Start).Children(func() {
			iconTile(c, warningIcon, danger, iconTileMedium)
			ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(space2).Children(func() {
				heading(c, title, 17, 2)
				ui.Text(c, detail).FontSize(textSize(c, 12)).TextColor(c.Theme().TextMuted)
			})
		})
		ui.Row(c).FillWidth().Justify(ui.End).Gap(space2).Padding(space3, 0, 0, 0).
			BorderWidth(1, 0, 0, 0).BorderColor(c.Theme().Border).Children(func() {
			if secondaryButton(c, "Cancel").AutoFocus().Clicked() {
				*open = false
			}
			if dangerButton(c, confirmLabel).Clicked() {
				*open = false
				confirmed = true
			}
		})
	})
	// The panel's own background is the window tone, one step below the card
	// the dialog covers. A hairline and the danger tint along its top edge give
	// the question an edge of its own instead of leaving it as a floating block
	// of text over the dimmed page.
	dialog.MinWidth(400 * textScale(c)).
		BorderWidth(1).BorderColor(c.Theme().Border).
		Role(ui.RoleAlertDialog).Label(title).Description(detail)
	switch {
	case confirmed:
		restoreDestructiveFocus(c, confirmFocus)
		return destructiveConfirmed
	case !*open:
		restoreDestructiveFocus(c, cancelFocus)
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
