package getconf

import (
	"strconv"
	"strings"
	"os"
	"os/exec"
)

const (
	//defaultGetConfCmd = "/usr/bin/getconf"
	defaultGetConfCmd = "getconf"
)

var getConfCmd string = defaultGetConfCmd

func init() {
	getConfCmd_var := os.Getenv("GETCONF_CMD")
	if getConfCmd_var != "" {
		getConfCmd = getConfCmd_var
	}
}

func GetConf(var_name string) ([]byte, error) {
	return exec.Command(getConfCmd, var_name).Output()
	// calling sysconf would be more efficient (but maybe less portable)
}

func GetConfAsUInt(var_name string) (res uint, err error) {
	var out []byte
	out, err = GetConf(var_name)
	if err != nil {
		return
	}
	var val uint64
	val, err = strconv.ParseUint(strings.Trim(string(out), " \t\n"), 10, 0)
	if err != nil {
		return
	}
	res = uint(val)
	return
}

func GetClkTck() (uint, error) {
	return GetConfAsUInt("CLK_TCK")
}

func GetNProcsConfigured() (uint, error) {
	return GetConfAsUInt("_NPROCESSORS_CONF")
}

func GetNProcsAvailable() (uint, error) {
	return GetConfAsUInt("_NPROCESSORS_ONLN")
}
