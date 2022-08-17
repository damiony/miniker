package containers

var (
	RUNNING             string = "running"
	STOPED              string = "stoped"
	EXIT                string = "exit"
	DefaultInfoLocation string = "/var/run/miniker/info/%s/"
	ConfigName          string = "config.json"
	LogName             string = "container.log"
	ENV_EXEC_PID        string = "miniker_pid"
	ENV_EXEC_CMD        string = "miniker_cmd"
	ImageUrl            string = "%s/miniker/images/%s/"
	WriteLayer          string = "%s/miniker/write/%s/"
	MntUrl              string = "%s/miniker/mnt/%s/"
)
