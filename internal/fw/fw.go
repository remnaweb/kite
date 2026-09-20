package fw

import (
	"os/exec"
	"strconv"
)

func OpenTCP(port int) {
	if port <= 0 || port > 65535 {
		return
	}
	p := strconv.Itoa(port)
	if _, err := exec.LookPath("ufw"); err == nil {
		_ = exec.Command("ufw", "allow", p+"/tcp", "comment", "panelvpn").Run()
		return
	}
	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		_ = exec.Command("firewall-cmd", "--permanent", "--add-port="+p+"/tcp").Run()
		_ = exec.Command("firewall-cmd", "--reload").Run()
	}
}
