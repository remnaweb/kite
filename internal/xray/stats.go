package xray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Stat struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type statsQuery struct {
	Stat []statValue `json:"stat"`
}

type statValue struct {
	Name  string      `json:"name"`
	Value json.Number `json:"value"`
}

func (m *Manager) QueryStats(reset bool) ([]Stat, error) {
	if !m.BinaryExists() || !m.Running() {
		return nil, nil
	}
	args := []string{"api", "statsquery", "--server", fmt.Sprintf("127.0.0.1:%d", m.APIPort)}
	if reset {
		args = append(args, "-reset")
	}
	cmd := exec.Command(m.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("statsquery: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		return nil, nil
	}
	var parsed statsQuery
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode stats: %w", err)
	}
	out := make([]Stat, 0, len(parsed.Stat))
	for _, s := range parsed.Stat {
		v, _ := s.Value.Int64()
		out = append(out, Stat{Name: s.Name, Value: v})
	}
	return out, nil
}

func UserTrafficDeltas(stats []Stat) map[string][2]int64 {
	// email -> [up, down]
	res := map[string][2]int64{}
	for _, s := range stats {
		// user>>>email>>>traffic>>>uplink
		parts := strings.Split(s.Name, ">>>")
		if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
			continue
		}
		email := parts[1]
		cur := res[email]
		switch parts[3] {
		case "uplink":
			cur[0] += s.Value
		case "downlink":
			cur[1] += s.Value
		}
		res[email] = cur
	}
	return res
}

func (m *Manager) Version() string {
	if !m.BinaryExists() {
		return ""
	}
	cmd := exec.Command(m.Bin, "version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := strings.SplitN(string(out), "\n", 2)[0]
	return strings.TrimSpace(line)
}
