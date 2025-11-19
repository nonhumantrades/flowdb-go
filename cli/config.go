package cli

import (
	"fmt"
	"strings"

	"github.com/nonhumantrades/flowdb-go/client"
)

func (c *Cli) handleConfigHelp(_ *ConfigHelp) {
	fmt.Println("Config commands:")
	fmt.Println("  config show")
	fmt.Println("      Show the current FlowDB client configuration (server address).")
	fmt.Println()
	fmt.Println("  config set [addr=<host:port>]")
	fmt.Println("      Set the server address and recreate the client.")
	fmt.Println("      If addr is omitted, you will be prompted with the current value as default.")
	fmt.Println()
	fmt.Println("  config reset")
	fmt.Println("      Reset the server address back to the built-in default:")
	fmt.Printf("      %s\n", defaultServerAddr)
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  config show")
	fmt.Println("  config set addr=127.0.0.1:7777")
	fmt.Println("  config reset")
}

func (c *Cli) handleConfigShow(_ *ShowConfig) {
	fmt.Println("Current config:")
	fmt.Printf("  server address: %s\n", c.opts.serverAddr)
}

func (c *Cli) handleConfigSet(cmd *SetConfig) {
	newAddr := strings.TrimSpace(cmd.Addr)

	var err error
	if newAddr == "" {
		newAddr, err = c.promptWithDefault("Server address", c.opts.serverAddr)
		if err != nil {
			fmt.Printf("aborted: %v\n", err)
			return
		}
		newAddr = strings.TrimSpace(newAddr)
	}

	if newAddr == "" {
		fmt.Println("server address cannot be empty")
		return
	}

	if newAddr == c.opts.serverAddr {
		fmt.Println("server address unchanged")
		return
	}

	newClient, err := client.Dial(c.ctx, client.Config{
		Address: newAddr,
	})
	if err != nil {
		fmt.Printf("failed to create client with new address: %v\n", err)
		return
	}

	old := c.client
	c.client = newClient
	c.opts.serverAddr = newAddr

	if old != nil {
		_ = old.Close()
	}

	if err := c.saveState(); err != nil {
		fmt.Printf("warning: config changed but failed to save: %v\n", err)
	}

	fmt.Printf("server address updated to %s\n", newAddr)
}

func (c *Cli) handleConfigReset(_ *ResetConfig) {
	if c.opts.serverAddr == defaultServerAddr {
		fmt.Println("server address already at default:", defaultServerAddr)
		return
	}

	newClient, err := client.Dial(c.ctx, client.Config{
		Address: defaultServerAddr,
	})
	if err != nil {
		fmt.Printf("failed to reset client config: %v\n", err)
		return
	}

	old := c.client
	c.client = newClient
	c.opts.serverAddr = defaultServerAddr

	if old != nil {
		_ = old.Close()
	}

	if err := c.saveState(); err != nil {
		fmt.Printf("warning: config reset but failed to save: %v\n", err)
	}

	fmt.Printf("server address reset to default (%s)\n", defaultServerAddr)
}
