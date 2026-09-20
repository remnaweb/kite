package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"panelvpn/internal/api"
	"panelvpn/internal/config"
	"panelvpn/internal/db"
	"panelvpn/internal/fw"
	"panelvpn/internal/setup"
	"panelvpn/internal/xray"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setting", "settings":
			if err := cmdSetting(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "init-inbound":
			if err := cmdInitInbound(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			return
		case "run", "serve":
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}
	runServer()
}

func printHelp() {
	fmt.Print(`Kite — панель Xray

  panel                    запуск панели
  panel setting            показать логин/хост
  panel setting [флаги]    задать данные админки
      -username NAME
      -password PASS
      -port PORT
      -host HOST           публичный IP или домен для vless-ссылок
  panel init-inbound       создать рабочий VLESS Reality
      -port 443
      -name main
      -remark VLESS-Reality
      -host HOST
`)
}

func cmdSetting(args []string) error {
	fs := flag.NewFlagSet("setting", flag.ContinueOnError)
	user := fs.String("username", "", "")
	pass := fs.String("password", "", "")
	port := fs.Int("port", 0, "")
	host := fs.String("host", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	conn, err := db.Open(cfg.DBPath())
	if err != nil {
		return err
	}

	changed := false
	if *user != "" || *pass != "" {
		u := *user
		p := *pass
		if u == "" {
			if a, err := setup.FirstAdmin(conn); err == nil {
				u = a.Username
			}
		}
		if p == "" {
			return fmt.Errorf("для смены логина укажи и -password")
		}
		if err := setup.SetAdmin(conn, u, p); err != nil {
			return err
		}
		changed = true
	}
	if *host != "" {
		if err := db.SetSetting(conn, "public_host", *host); err != nil {
			return err
		}
		changed = true
	}
	if *port > 0 {
		listen := fmt.Sprintf("0.0.0.0:%d", *port)
		if err := setup.WriteEnvFile(cfg.EnvFile, listen, cfg.DataDir, cfg.XrayBin); err != nil {
			log.Printf("env file: %v (продолжаю)", err)
		}
		_ = os.Setenv("PANEL_LISTEN", listen)
		fw.OpenTCP(*port)
		changed = true
		fmt.Printf("порт панели: %d (нужен restart сервиса)\n", *port)
	}

	admin, _ := setup.FirstAdmin(conn)
	uname := ""
	if admin != nil {
		uname = admin.Username
	}
	if !changed {
		fmt.Printf("username: %s\n", uname)
		fmt.Printf("listen:   %s\n", cfg.Listen)
		fmt.Printf("host:     %s\n", db.GetSetting(conn, "public_host", ""))
		fmt.Printf("data:     %s\n", cfg.DataDir)
		return nil
	}
	fmt.Printf("сохранено, username=%s host=%s\n", uname, db.GetSetting(conn, "public_host", *host))
	return nil
}

func cmdInitInbound(args []string) error {
	fs := flag.NewFlagSet("init-inbound", flag.ContinueOnError)
	port := fs.Int("port", 443, "")
	name := fs.String("name", "main", "")
	remark := fs.String("remark", "VLESS-Reality", "")
	host := fs.String("host", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	conn, err := db.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	if *host == "" {
		*host = db.GetSetting(conn, "public_host", "")
	}
	ib, c, link, err := setup.InitReality(conn, *remark, *port, *name, *host, true)
	if err != nil {
		return err
	}
	xm := xray.NewManager(cfg.XrayBin, cfg.XrayConfigPath(), cfg.XrayLogPath(), cfg.XrayAPIPort)
	if err := xray.WriteOnly(conn, xm); err != nil {
		return err
	}
	fmt.Printf("inbound #%d %s :%d\n", ib.ID, ib.Remark, ib.Port)
	fmt.Printf("client  #%d %s %s\n", c.ID, c.Name, c.UUID)
	fmt.Printf("sub     /sub/%s\n", c.SubID)
	fmt.Println(link)
	return nil
}

func runServer() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	conn, err := db.Open(cfg.DBPath())
	if err != nil {
		log.Fatal(err)
	}

	xm := xray.NewManager(cfg.XrayBin, cfg.XrayConfigPath(), cfg.XrayLogPath(), cfg.XrayAPIPort)
	srv := api.New(conn, xm)

	if err := srv.ApplyOnStart(); err != nil {
		log.Printf("xray start: %v", err)
	}

	stop := make(chan struct{})
	go srv.CollectLoop(stop)

	handler := srv.Router(cfg.WebDist)
	httpSrv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Kite listening on %s", cfg.Listen)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	close(stop)
	_ = xm.Stop()
	_ = httpSrv.Close()
}
