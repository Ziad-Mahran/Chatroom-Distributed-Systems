package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"chatroom/common"
)

func main() {
	addr := flag.String("addr", "localhost:1234", "server address")
	name := flag.String("name", "", "username")
	flag.Parse()

	if *name == "" {
		fmt.Print("Username: ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		*name = strings.TrimSpace(line)
	}

	c, err := rpc.Dial("tcp", *addr)
	if err != nil {
		fmt.Println("cannot connect:", err)
		os.Exit(1)
	}
	if err := c.Call("Chat.Join", common.JoinArgs{Username: *name}, &common.Empty{}); err != nil {
		fmt.Println("join failed:", err)
		os.Exit(1)
	}
	fmt.Println("Joined as", *name, "— commands: /list, /quit")

	var once sync.Once
	quit := func(code int) {
		once.Do(func() {
			c.Call("Chat.Leave", common.LeaveArgs{Username: *name}, &common.Empty{})
			c.Close()
			fmt.Println("\nBye.")
			os.Exit(code)
		})
	}

	// Ctrl+C => clean leave
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; quit(0) }()

	// polling goroutine: prints incoming messages automatically
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			var r common.PollReply
			if err := c.Call("Chat.Poll", common.PollArgs{Username: *name}, &r); err != nil {
				fmt.Println("\nlost connection to server:", err)
				os.Exit(1)
			}
			for _, m := range r.Messages {
				fmt.Printf("\r%s\n> ", m)
			}
		}
	}()

	in := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for in.Scan() {
		text := strings.TrimSpace(in.Text())
		switch {
		case text == "":
		case text == "/quit":
			quit(0)
		case text == "/list":
			var r common.ListReply
			if err := c.Call("Chat.List", common.Empty{}, &r); err != nil {
				fmt.Println("error:", err)
			} else {
				fmt.Println("Online:", strings.Join(r.Users, ", "))
			}
		default:
			if err := c.Call("Chat.Send", common.SendArgs{Username: *name, Text: text}, &common.Empty{}); err != nil {
				fmt.Println("send failed:", err)
			}
		}
		fmt.Print("> ")
	}
	quit(0) // stdin closed (Ctrl+D)
}
