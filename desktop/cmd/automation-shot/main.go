// Command automation-shot captures a running Aurora window through MyGo's
// automation socket. It is the client half of desktop/scripts/e2e-window.sh:
// the shell builds aurora-desktop with the e2e tag, waits for its socket and
// asks this program for a PNG, so a UI change can be checked against the real
// window backend instead of only the offscreen tester.
//
// The program talks the plugin's newline-delimited JSON protocol and uses the
// standard library only, so it builds without the desktop module's window
// dependencies.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type request struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *protocolError  `json:"error"`
}

type protocolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type windowInfo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
	// ContentBounds is the area the view occupies, in window coordinates. A
	// stage capture compares its target against this to prove the scroll moved
	// it into the picture rather than only into the tree.
	ContentBounds rect `json:"contentBounds"`
}

type rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type treeNode struct {
	ID string `json:"id"`
	// Key may be a string, a boolean or a number in the protocol; only string
	// keys name pages, so any other value is compared by its print form.
	Key         any      `json:"key"`
	Role        string   `json:"role"`
	Label       string   `json:"label"`
	Value       string   `json:"value"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	// Bounds are the element's box: the tree lists every element whether or not
	// the viewport shows it, so only the box says what the capture photographs.
	Bounds rect `json:"bounds"`
}

func (n treeNode) key() string {
	if s, ok := n.Key.(string); ok {
		return s
	}
	return ""
}

// pressable reports whether the node accepts a press, so a label shared by a
// heading and a control still presses the control.
func (n treeNode) pressable() bool {
	for _, action := range n.Actions {
		if action == "press" {
			return true
		}
	}
	return false
}

func main() {
	socket := flag.String("socket", "", "automation socket the app listens on")
	out := flag.String("out", "", "PNG file to write")
	expect := flag.String("expect", "", "texts that must appear in the window's accessibility tree, comma-separated")
	press := flag.String("press", "", "element keys or labels to press in order, comma-separated")
	scrollTo := flag.String("scroll-to", "", "texts whose element is scrolled into view before capturing, comma-separated")
	checkLabels := flag.Bool("check-labels", true, "fail when an interactive node has no accessible name")
	window := flag.Int("window", 0, "index of the window to capture")
	timeout := flag.Duration("timeout", 15*time.Second, "how long to wait for the window")
	flag.Parse()

	if *socket == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "automation-shot: -socket and -out are required")
		os.Exit(2)
	}
	if err := run(*socket, *out, *expect, *press, *scrollTo, *checkLabels, *window, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, "automation-shot:", err)
		os.Exit(1)
	}
}

func run(socket, out, expect, press, scrollTo string, checkLabels bool, index int, timeout time.Duration) error {
	conn, err := dial(socket, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := &protocol{
		conn:   conn,
		reader: bufio.NewReader(conn),
		enc:    json.NewEncoder(conn),
	}

	windows, err := client.waitForWindows(index, timeout)
	if err != nil {
		return err
	}
	win := windows[index]
	fmt.Printf("window %d %q (%s)\n", win.ID, win.Title, win.Kind)

	// The window exists before its view has built the sidebar this client may
	// press, so let the first frame settle before acting on it.
	settle := func() error {
		// Animations may still be running; wait for quiet, but a spinner asks
		// for frames forever, so a timeout still means "capture this frame".
		err := client.call("idle", map[string]any{"window": win.ID, "timeout": 2000}, nil)
		var protoErr *protocolError
		if err != nil && (!errors.As(err, &protoErr) || protoErr.Code != "timeout") {
			return err
		}
		return nil
	}
	if err := settle(); err != nil {
		return err
	}

	if press != "" {
		// Steps run in order: a page first, then a tab or stage inside it.
		for _, step := range strings.Split(press, ",") {
			step = strings.TrimSpace(step)
			if step == "" {
				continue
			}
			if err := client.press(win.ID, step); err != nil {
				return err
			}
			fmt.Printf("pressed %q\n", step)
			if err := settle(); err != nil {
				return err
			}
		}
		if latest, err := client.waitForWindows(index, timeout); err == nil {
			fmt.Printf("window title now %q\n", latest[index].Title)
		}
	}

	for _, want := range strings.Split(scrollTo, ",") {
		want = strings.TrimSpace(want)
		if want == "" {
			continue
		}
		if err := client.scrollPointIntoView(win, want); err != nil {
			return err
		}
		fmt.Printf("scrolled to %q\n", want)
		if err := settle(); err != nil {
			return err
		}
	}

	if checkLabels {
		if err := client.requireLabels(win.ID); err != nil {
			return err
		}
	}

	if expect != "" {
		var tree struct {
			Nodes []treeNode `json:"nodes"`
		}
		if err := client.call("tree", map[string]any{"window": win.ID}, &tree); err != nil {
			return err
		}
		for _, want := range strings.Split(expect, ",") {
			want = strings.TrimSpace(want)
			if want == "" {
				continue
			}
			found := false
			for _, node := range tree.Nodes {
				for _, text := range []string{node.Label, node.Value, node.Description} {
					if strings.Contains(text, want) {
						found = true
					}
				}
			}
			if !found {
				return fmt.Errorf("window %d does not show %q", win.ID, want)
			}
			fmt.Printf("tree shows %q\n", want)
		}
	}

	var shot struct {
		PNG    string `json:"png"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := client.call("capture", map[string]any{"window": win.ID}, &shot); err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(shot.PNG)
	if err != nil {
		return fmt.Errorf("decode capture: %w", err)
	}
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		return err
	}
	fmt.Printf("captured %dx%d -> %s\n", shot.Width, shot.Height, out)
	return nil
}

// dial waits for the socket: the app creates it after its first window exists,
// which is later than the shell can start the client.
func dial(socket string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var last error
	for {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			return conn, nil
		}
		last = err
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("connect %s: %w", socket, last)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

type protocol struct {
	conn   net.Conn
	reader *bufio.Reader
	enc    *json.Encoder
	nextID int
}

func (p *protocol) call(method string, params map[string]any, result any) error {
	p.nextID++
	if err := p.enc.Encode(request{ID: p.nextID, Method: method, Params: params}); err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	line, err := p.reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("%s: read answer: %w", method, err)
	}
	var resp response
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("%s: decode answer: %w", method, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("%s: %w", method, resp.Error)
	}
	if result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("%s: decode result: %w", method, err)
		}
	}
	return nil
}

// press acts on an element by its key first, because a sidebar entry's key is
// the page id the app navigates by and does not change with the language. A
// tab has no key, so a key miss falls back to a label match, preferring the
// node that accepts a press. Labels are tried exactly first and then by
// substring: a record row is named "Select btc", not "btc".
func (p *protocol) press(window int, name string) error {
	err := p.call("act", map[string]any{"window": window, "key": name, "action": "press"}, nil)
	var protoErr *protocolError
	if err == nil || !errors.As(err, &protoErr) || protoErr.Code != "no_element" {
		return err
	}
	var tree struct {
		Nodes []treeNode `json:"nodes"`
	}
	if err := p.call("tree", map[string]any{"window": window}, &tree); err != nil {
		return err
	}
	find := func(match func(string) bool, pressableOnly bool) string {
		target := ""
		for _, node := range tree.Nodes {
			if node.key() == name || !match(node.Label) {
				continue
			}
			if pressableOnly && !node.pressable() {
				continue
			}
			if target == "" || node.pressable() {
				target = node.ID
			}
			if node.pressable() {
				break
			}
		}
		return target
	}
	target := find(func(label string) bool { return strings.EqualFold(label, name) }, false)
	if target == "" {
		// The substring pass exists for "Select btc", and it must not fall back
		// to prose: "Health" once matched the first-run panel's step "Review
		// scheduler health..." and pressed a text node instead of failing.
		lower := strings.ToLower(name)
		target = find(func(label string) bool {
			return strings.Contains(strings.ToLower(label), lower)
		}, true)
	}
	if target == "" {
		return fmt.Errorf("no element named %q", name)
	}
	return p.call("act", map[string]any{"window": window, "id": target, "action": "press"}, nil)
}

func (p *protocol) waitForWindows(index int, timeout time.Duration) ([]windowInfo, error) {
	deadline := time.Now().Add(timeout)
	for {
		var list struct {
			Windows []windowInfo `json:"windows"`
		}
		if err := p.call("windows", nil, &list); err != nil {
			return nil, err
		}
		if index < len(list.Windows) {
			return list.Windows, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no window %d after %s", index, timeout)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Error lets callers print the plugin's code beside its message.
func (e *protocolError) Error() string { return e.Code + ": " + e.Message }

// scrollPointIntoView brings the element showing text into the content area by
// wheeling over the page. The accessibility action scrolls only as far as the
// view's own containers allow in that frame; a stage body below a tall card was
// reported as scrolled while nothing moved, so three different NFT stages
// photographed the same top-of-page image and every capture still passed.
//
// The wheel is what a person does, and the loop is bounded: at most
// maxWheelSteps notches, each followed by a frame, stopping as soon as the box
// is inside the view. Requiring the box afterwards is what turns "nothing
// happened" into a failure.
func (p *protocol) scrollPointIntoView(win windowInfo, text string) error {
	const maxWheelSteps = 40
	const notch = 240.0 // DIPs per step, a few lines of text

	area := win.ContentBounds
	if area.Width <= 0 || area.Height <= 0 {
		return fmt.Errorf("window %d reports no content area to scroll %q into", win.ID, text)
	}
	// Wheel over the middle of the content, so the pointer is inside the
	// scroll container rather than over the sidebar.
	midX := area.X + area.Width/2
	midY := area.Y + area.Height/2

	for step := 0; step < maxWheelSteps; step++ {
		node, ok, err := p.findBox(win.ID, text)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no element with a box shows %q", text)
		}
		if onScreen(node.Bounds, area) {
			return nil
		}
		dy := notch * wheelDirection(node.Bounds, area)
		if err := p.call("input", map[string]any{
			"window": win.ID,
			"events": []map[string]any{{"type": "wheel", "x": midX, "y": midY, "dx": 0, "dy": dy}},
		}, nil); err != nil {
			return err
		}
		if err := p.waitFrame(win.ID); err != nil {
			return err
		}
	}
	return fmt.Errorf("could not scroll %q into the content area after %d steps", text, maxWheelSteps)
}

// findBox returns the first node showing text that has a box.
func (p *protocol) findBox(window int, text string) (treeNode, bool, error) {
	var tree struct {
		Nodes []treeNode `json:"nodes"`
	}
	if err := p.call("tree", map[string]any{"window": window}, &tree); err != nil {
		return treeNode{}, false, err
	}
	lower := strings.ToLower(text)
	for _, node := range tree.Nodes {
		if !strings.Contains(strings.ToLower(node.Label), lower) &&
			!strings.Contains(strings.ToLower(node.Value), lower) {
			continue
		}
		if node.Bounds.Width <= 0 || node.Bounds.Height <= 0 {
			continue
		}
		return node, true, nil
	}
	return treeNode{}, false, nil
}

// waitFrame lets the view draw the frame the input asked for.
func (p *protocol) waitFrame(window int) error {
	err := p.call("idle", map[string]any{"window": window, "timeout": 2000}, nil)
	var protoErr *protocolError
	if err != nil && (!errors.As(err, &protoErr) || protoErr.Code != "timeout") {
		return err
	}
	return nil
}

// wheelDirection is which way to scroll to reach box from the current offset:
// 1 down, -1 up. A box that starts above the content area is reached by
// scrolling up, which matters when an earlier step went too far.
func wheelDirection(box, area rect) float64 {
	if box.Y+box.Height <= area.Y {
		return -1
	}
	return 1
}

// onScreen reports whether box is inside area.
func onScreen(box, area rect) bool {
	return box.X+box.Width > area.X && box.X < area.X+area.Width &&
		box.Y+box.Height > area.Y && box.Y < area.Y+area.Height
}

// interactiveRoles are the nodes a screen reader announces as controls. A
// nameless one is a failure no screenshot can show.
var interactiveRoles = map[string]bool{
	"button": true, "textField": true, "tab": true, "treeItem": true,
	"checkBox": true, "radio": true, "comboBox": true, "popUpButton": true,
	"menuButton": true, "stepper": true,
}

// unlabeledInteractive names the interactive nodes without a label or
// description; either counts as an accessible name.
func unlabeledInteractive(nodes []treeNode) []string {
	missing := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if !interactiveRoles[node.Role] {
			continue
		}
		if strings.TrimSpace(node.Label) != "" || strings.TrimSpace(node.Description) != "" {
			continue
		}
		missing = append(missing, fmt.Sprintf("%s %q", node.Role, node.Value))
	}
	return missing
}

// requireLabels fails the capture when the window shows a control a screen
// reader would announce without a name.
func (p *protocol) requireLabels(window int) error {
	var tree struct {
		Nodes []treeNode `json:"nodes"`
	}
	if err := p.call("tree", map[string]any{"window": window}, &tree); err != nil {
		return err
	}
	missing := unlabeledInteractive(tree.Nodes)
	if len(missing) == 0 {
		return nil
	}
	shown := missing
	if len(shown) > 5 {
		shown = append(append([]string{}, shown[:5]...), fmt.Sprintf("and %d more", len(missing)-5))
	}
	return fmt.Errorf("controls without an accessible name: %s", strings.Join(shown, ", "))
}
