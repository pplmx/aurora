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
	expect := flag.String("expect", "", "text that must appear in the window's accessibility tree")
	press := flag.String("press", "", "element key to press before capturing, such as a sidebar page id")
	window := flag.Int("window", 0, "index of the window to capture")
	timeout := flag.Duration("timeout", 15*time.Second, "how long to wait for the window")
	flag.Parse()

	if *socket == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "automation-shot: -socket and -out are required")
		os.Exit(2)
	}
	if err := run(*socket, *out, *expect, *press, *window, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, "automation-shot:", err)
		os.Exit(1)
	}
}

func run(socket, out, expect, press string, index int, timeout time.Duration) error {
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
		if err := client.press(win.ID, press); err != nil {
			return err
		}
		fmt.Printf("pressed %q\n", press)
		if err := settle(); err != nil {
			return err
		}
		if latest, err := client.waitForWindows(index, timeout); err == nil {
			fmt.Printf("window title now %q\n", latest[index].Title)
		}
	}

	if expect != "" {
		var tree struct {
			Nodes []treeNode `json:"nodes"`
		}
		if err := client.call("tree", map[string]any{"window": win.ID}, &tree); err != nil {
			return err
		}
		found := false
		for _, node := range tree.Nodes {
			for _, text := range []string{node.Label, node.Value, node.Description} {
				if strings.Contains(text, expect) {
					found = true
				}
			}
		}
		if !found {
			return fmt.Errorf("window %d does not show %q", win.ID, expect)
		}
		fmt.Printf("tree shows %q\n", expect)
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
	find := func(match func(string) bool) string {
		target := ""
		for _, node := range tree.Nodes {
			if node.key() == name || !match(node.Label) {
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
	target := find(func(label string) bool { return strings.EqualFold(label, name) })
	if target == "" {
		lower := strings.ToLower(name)
		target = find(func(label string) bool {
			return strings.Contains(strings.ToLower(label), lower)
		})
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
