package main

import (
	"encoding/json"
	"os"

	"github.com/flyqie/godbus/v5"
	"github.com/flyqie/godbus/v5/introspect"
)

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	node, err := introspect.Call(conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus"))
	if err != nil {
		panic(err)
	}
	data, _ := json.MarshalIndent(node, "", "    ")
	os.Stdout.Write(data)
}
