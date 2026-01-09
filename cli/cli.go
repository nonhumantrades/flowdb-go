package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/AR1011/slog"
	"github.com/nonhumantrades/flowdb-go/client"
	"github.com/nonhumantrades/flowdb-go/proto"
	"github.com/nonhumantrades/flowdb-go/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultServerAddr = "localhost:7777"

type Opts struct {
	credentialsPath string
	serverAddr      string
}

func (o *Opts) FillDefaults() *Opts {
	if o == nil {
		o = &Opts{}
		o.FillDefaults()
		return o
	}

	if o.credentialsPath == "" {
		o.credentialsPath = getCredentialsPath()
	}

	if o.serverAddr == "" {
		o.serverAddr = defaultServerAddr
	}

	return o
}

type S3Profile struct {
	Name  string              `json:"name"`
	Creds types.S3Credentials `json:"creds"`
}

type storedState struct {
	S3Profiles []S3Profile `json:"s3_profiles,omitempty"`
	ServerAddr string      `json:"server_addr,omitempty"`
}

type Cli struct {
	opts   Opts
	parser *Parser
	reader *bufio.Reader
	client *client.Client

	ctx    context.Context
	cancel context.CancelFunc

	state storedState
}

func New(opts *Opts) *Cli {
	if opts == nil {
		opts = new(Opts)
	}

	opts = opts.FillDefaults()

	ctx, cancel := context.WithCancel(context.Background())

	c := &Cli{
		opts:   *opts,
		reader: bufio.NewReader(os.Stdin),
		parser: NewParser(),
		ctx:    ctx,
		cancel: cancel,
	}

	if err := c.init(); err != nil {
		slog.Warn("cli.init: error during init()", "error", err)
	}

	cl, err := client.Dial(ctx, client.Config{
		Address: c.opts.serverAddr,
	})
	if err != nil {
		slog.Warn("cli.init: error during client.New()", "error", err)
	} else {
		c.client = cl
	}

	c.registerCommands()
	return c
}

func (c *Cli) init() error {
	data, err := os.ReadFile(c.opts.credentialsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var st storedState
	if err := json.Unmarshal(data, &st); err == nil {
		c.state = st
		if c.state.ServerAddr != "" {
			c.opts.serverAddr = c.state.ServerAddr
		}
		return nil
	}

	return nil
}

func isEmptyS3(c types.S3Credentials) bool {
	return c.Bucket == "" && c.Url == "" && c.AccessKey == "" && c.SecretKey == "" && c.Region == ""
}

func (c *Cli) saveState() error {
	c.state.ServerAddr = c.opts.serverAddr

	data, err := json.MarshalIndent(&c.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.opts.credentialsPath, data, 0o600)
}

func (c *Cli) registerCommands() {
	// tables
	c.parser.Register("table", func() any { return &TableInfo{} })
	c.parser.Register("tables", func() any { return &ListTables{} })

	// s3 creds
	c.parser.RegisterMany(func() any { return &S3Help{} }, "s3", "s3 help", "s3 h")
	c.parser.Register("s3 list", func() any { return &S3List{} })
	c.parser.Register("s3 add", func() any { return &S3Add{} })
	c.parser.Register("s3 delete", func() any { return &S3Delete{} })

	// stats + queries
	c.parser.Register("stats", func() any { return &Stats{} })
	c.parser.Register("head", func() any { return &Head{} })
	c.parser.Register("query", func() any { return &Query{} })
	c.parser.Register("delete", func() any { return &Delete{} })

	// backup / restore
	c.parser.Register("backup", func() any { return &S3Backup{} })
	c.parser.Register("restore", func() any { return &S3Restore{} })

	// config
	c.parser.RegisterMany(func() any { return &ConfigHelp{} }, "config", "config help", "config h")
	c.parser.Register("config show", func() any { return &ShowConfig{} })
	c.parser.Register("config set", func() any { return &SetConfig{} })
	c.parser.Register("config reset", func() any { return &ResetConfig{} })
}

func (c *Cli) Loop() {
	defer func() {
		if c.client != nil {
			_ = c.client.Close()
		}
	}()

	for {
		line, err := c.readLine("flowdb> ")
		if err != nil {
			fmt.Println()
			return
		}

		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return
		}

		if line == "help" || line == "h" {
			c.handleHelp()
			continue
		}

		if line == "clear" || line == "cls" {
			fmt.Print("\033[H\033[2J")
			continue
		}

		pc, err := c.parser.Parse(line)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}

		switch cmd := pc.Value.(type) {
		case *TableInfo:
			c.handleTableInfo(cmd)
		case *ListTables:
			c.handleListTables(cmd)

		case *S3Help:
			c.handleS3Help(cmd)
		case *S3List:
			c.handleS3List(cmd)
		case *S3Add:
			c.handleS3Add(cmd)
		case *S3Delete:
			c.handleS3Delete(cmd)

		case *Stats:
			c.handleStats(cmd)
		case *Head:
			c.handleHead(cmd)
		case *Query:
			c.handleQuery(cmd)
		case *Delete:
			c.handleDelete(cmd)

		case *S3Backup:
			c.handleBackup(cmd)
		case *S3Restore:
			c.handleRestore(cmd)

		case *ConfigHelp:
			c.handleConfigHelp(cmd)
		case *ShowConfig:
			c.handleConfigShow(cmd)
		case *SetConfig:
			c.handleConfigSet(cmd)
		case *ResetConfig:
			c.handleConfigReset(cmd)
		default:
			fmt.Printf("no handler for type %T\n", cmd)
		}
	}
}

func (c *Cli) handleTableInfo(cmd *TableInfo) {
	if !c.requireClient() {
		return
	}

	if cmd.Name == "" {
		fmt.Println("usage: table name=<table_name>")
		return
	}

	table, err := c.client.GetTable(c.ctx, cmd.Name)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	if table == nil {
		fmt.Printf("table '%s' not found\n", cmd.Name)
		return
	}

	fmt.Printf("Table: %s\n", table.Name)
	fmt.Printf("  Rows:        %s\n", formatNumber(table.RowCount))
	fmt.Printf("  Size:        %s\n", formatBytes(table.DataBytes))
	fmt.Printf("  Min time:    %s\n", formatTimestamp(table.MinTimestamp))
	fmt.Printf("  Max time:    %s\n", formatTimestamp(table.MaxTimestamp))
	fmt.Printf("  Created:     %s\n", formatTimestamp(table.CreatedAt))
	fmt.Printf("  Last update: %s\n", formatTimestamp(table.LastUpdated))
}
func (c *Cli) handleListTables(cmd *ListTables) {
	if !c.requireClient() {
		return
	}

	tables, err := c.client.ListTables(c.ctx)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	if len(tables) == 0 {
		fmt.Println("no tables found")
		return
	}

	fmt.Printf("%-24s %12s %10s   %-19s   %-19s\n", "NAME", "ROWS", "SIZE", "MIN TIME", "MAX TIME")
	for _, t := range tables {
		fmt.Printf("%-24s %12s %10s   %-19s   %-19s\n",
			t.Name,
			formatNumber(t.RowCount),
			formatBytes(t.DataBytes),
			formatTimestampShort(t.MinTimestamp),
			formatTimestampShort(t.MaxTimestamp),
		)
	}
}

func (c *Cli) handleStats(cmd *Stats) {
	if !c.requireClient() {
		return
	}

	stats, err := c.client.GetStats(c.ctx)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Println("FlowDB Statistics")
	fmt.Println("─────────────────────────────────────")
	fmt.Printf("Uptime:              %s\n", formatDuration(time.Duration(stats.UptimeSeconds)*time.Second))
	fmt.Printf("Started:             %s\n", formatTimestamp(stats.StartedAt))
	fmt.Println()
	fmt.Println("Storage:")
	fmt.Printf("  On disk:           %s\n", formatBytes(stats.OnDiskBytes))
	fmt.Printf("  Bytes written:     %s\n", formatBytes(stats.BytesWritten))
	fmt.Printf("  Bytes deleted:     %s\n", formatBytes(stats.BytesDeleted))
	fmt.Println()
	fmt.Println("Queries:")
	fmt.Printf("  Datapoints queried: %s\n", formatNumber(stats.DatapointsQueried))
	fmt.Printf("  Total requests:     %s\n", formatNumber(stats.TotalRequests))
	fmt.Println()
	fmt.Println("Request breakdown:")
	fmt.Printf("  Insert:            %s\n", formatNumber(stats.InsertRequests))
	fmt.Printf("  Batch insert:      %s\n", formatNumber(stats.BatchInsertRequests))
	fmt.Printf("  Query:             %s\n", formatNumber(stats.QueryRequests))
	fmt.Printf("  Stream query:      %s\n", formatNumber(stats.StreamQueryRequests))
	fmt.Printf("  Delete:            %s\n", formatNumber(stats.DeleteRequests))
	fmt.Printf("  Create table:      %s\n", formatNumber(stats.CreateTableRequests))
	fmt.Printf("  Drop table:        %s\n", formatNumber(stats.DropTableRequests))
	fmt.Printf("  Get table:         %s\n", formatNumber(stats.GetTableRequests))
	fmt.Printf("  List tables:       %s\n", formatNumber(stats.ListTablesRequests))
	fmt.Printf("  Backup to S3:      %s\n", formatNumber(stats.BackupToS3Requests))
	fmt.Printf("  Restore from S3:   %s\n", formatNumber(stats.RestoreFromS3Requests))
}

func (c *Cli) buildQueryRequest(table, prefix, from, to string, limit int, isHead bool) (*proto.QueryRequest, error) {
	if table == "" && prefix == "" {
		return nil, fmt.Errorf("either table= or prefix= is required")
	}

	var fromTime, toTime time.Time
	var err error

	if from == "" && to == "" {
		// Default to last hour
		toTime = time.Now().UTC()
		fromTime = toTime.Add(-time.Hour)
	} else {
		fromTime, err = parseTime(from)
		if err != nil {
			return nil, fmt.Errorf("invalid from time: %w", err)
		}
		toTime, err = parseTime(to)
		if err != nil {
			return nil, fmt.Errorf("invalid to time: %w", err)
		}
	}

	if limit <= 0 {
		limit = 100
	}

	req := &proto.QueryRequest{
		TableName: table,
		Prefix:    prefix,
		Head:      isHead,
		FilterOptions: &proto.FilterOptions{
			Limit: client.Int64(int64(limit)),
		},
	}

	if !fromTime.IsZero() {
		req.FilterOptions.From = timestamppb.New(fromTime)
	}
	if !toTime.IsZero() {
		req.FilterOptions.To = timestamppb.New(toTime)
	}

	return req, nil
}

func (c *Cli) handleHead(cmd *Head)   { fmt.Printf("head: %+v\n", *cmd) }
func (c *Cli) handleQuery(cmd *Query) { fmt.Printf("query: %+v\n", *cmd) }
func (c *Cli) handleDelete(cmd *Delete) {
	fmt.Printf("delete: %+v\n", *cmd)
}

func (c *Cli) handleBackup(cmd *S3Backup)   { fmt.Printf("backup: %+v\n", *cmd) }
func (c *Cli) handleRestore(cmd *S3Restore) { fmt.Printf("restore: %+v\n", *cmd) }

func (c *Cli) handleHelp() {
	fmt.Println("FlowDB CLI")
	fmt.Println()
	fmt.Println("General:")
	fmt.Println("  help, h              Show this help menu")
	fmt.Println("  clear, cls           Clear the screen")
	fmt.Println("  exit, quit           Exit the CLI")
	fmt.Println()
	fmt.Println("Tables:")
	fmt.Println("  table name=<table>   Show information about a specific table")
	fmt.Println("  tables                List all tables")
	fmt.Println()
	fmt.Println("S3 profiles:")
	fmt.Println("  s3, s3 help, s3 h     Show S3 help and usage")
	fmt.Println("  s3 list               List configured S3 profiles")
	fmt.Println("  s3 add [name=<name>]  Add or update an S3 profile (prompts for fields)")
	fmt.Println("  s3 delete [name=<n>]  Delete an S3 profile (interactive if omitted)")
	fmt.Println()
	fmt.Println("Queries:")
	fmt.Println("  head table=<t>|prefix=<p> from=<ts> to=<ts> [limit=<n>]")
	fmt.Println("                        Query earliest rows in a range")
	fmt.Println("  query table=<t>|prefix=<p> from=<ts> to=<ts> [limit=<n>]")
	fmt.Println("                        General query")
	fmt.Println("  delete table=<t>|prefix=<p> from=<ts> to=<ts> [limit=<n>]")
	fmt.Println("                        Delete rows in a range")
	fmt.Println()
	fmt.Println("Backup / Restore:")
	fmt.Println("  backup                Backup database (will use S3 profiles later)")
	fmt.Println("  restore               Restore database from backup")
	fmt.Println()
	fmt.Println("Stats:")
	fmt.Println("  stats                 Show database statistics (live mode later)")
	fmt.Println()
	fmt.Println("Config:")
	fmt.Println("  config, config help, config h")
	fmt.Println("                        Show config help and usage")
	fmt.Println("  config show           Show current server address")
	fmt.Println("  config set [addr=<a>] Set server address and recreate client")
	fmt.Println("  config reset          Reset server address to default")
	fmt.Println()
	fmt.Println("For more details on a group of commands:")
	fmt.Println("  s3 help               S3-specific help")
	fmt.Println("  config help           Config-specific help")
}
