package golitecron

type TaskConfig struct {
	ID            string `yaml:"id" json:"id"`
	CronExpr      string `yaml:"cron_expr" json:"cron_expr"`
	Timeout       int64  `yaml:"timeout" json:"timeout"`
	Retry         int    `yaml:"retry" json:"retry"`
	Location      string `yaml:"location" json:"location"`
	EnableSeconds bool   `yaml:"enable_seconds" json:"enable_seconds"`
	EnableYears   bool   `yaml:"enable_years" json:"enable_years"`
	FuncName      string `yaml:"func_name" json:"func_name"`
}

type Config struct {
	Tasks []TaskConfig `yaml:"tasks" json:"tasks"`
}
