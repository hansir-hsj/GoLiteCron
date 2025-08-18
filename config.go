package golitecron

type TaskConfig struct {
	ID            string `yaml:"id"`
	CronExpr      string `yaml:"cron_expr"`
	Timeout       int64  `yaml:"timeout"`
	Retry         int    `yaml:"retry"`
	Location      string `yaml:"location"`
	EnableSeconds bool   `yaml:"enable_seconds"`
	EnableYears   bool   `yaml:"enable_years"`
	FuncName      string `yaml:"func_name"`
}

type Config struct {
	Tasks []TaskConfig `yaml:"tasks"`
}
