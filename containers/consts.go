package containers

var (
	RUNNING             string = "running"
	STOPED              string = "stoped"
	EXIT                string = "exit"
	DefaultInfoLocation string = "/var/run/miniker/%s/"
	ConfigName          string = "config.json"
	LogName             string = "container.log"
	ENV_EXEC_PID        string = "miniker_pid"
	ENV_EXEC_CMD        string = "miniker_cmd"
)
