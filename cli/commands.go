package cli

// tables
type TableInfo struct {
	Name string `cli:"name"`
}

type ListTables struct{}

// s3 creds
type S3Help struct{}
type S3List struct{}

type S3Add struct {
	Name string `cli:"name"`
}

type S3Delete struct {
	Name string `cli:"name"`
}

// queries
type Head struct {
	Table  string `cli:"table"`
	Prefix string `cli:"prefix"`
	From   string `cli:"from"`
	To     string `cli:"to"`
	Limit  int    `cli:"limit"`
}

type Query struct {
	Table  string `cli:"table"`
	Prefix string `cli:"prefix"`
	From   string `cli:"from"`
	To     string `cli:"to"`
	Limit  int    `cli:"limit"`
}

type Delete struct {
	Table  string `cli:"table"`
	Prefix string `cli:"prefix"`
	From   string `cli:"from"`
	To     string `cli:"to"`
	Limit  int    `cli:"limit"`
}

// backup + restore
type S3Backup struct{}
type S3Restore struct{}

// db stats
type Stats struct{}

// config
type ConfigHelp struct{}
type ShowConfig struct{}
type ResetConfig struct{}
type SetConfig struct {
	Addr string `cli:"addr"`
}
