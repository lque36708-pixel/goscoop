package powershell

import (
	"fmt"
	"os"
	"os/exec"
)

type Vars struct {
	Dir          string
	PersistDir   string
	App          string
	Version      string
	Bucket       string
	BucketsDir   string
	Architecture string
	Global       bool
	CleanVersion string
	Fname        string
}

func RunScript(script string, vars Vars) error {
	psScript := buildScript(script, vars)

	f, err := os.CreateTemp("", "ss_ps_*.ps1")
	if err != nil {
		return fmt.Errorf("create temp script: %w", err)
	}
	if _, err := f.WriteString(psScript); err != nil {
		f.Close()
		os.Remove(f.Name())
		return fmt.Errorf("write temp script: %w", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	cmd := exec.Command("powershell", "-NoProfile", "-File", f.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("powershell: %w", err)
	}
	return nil
}

func RunScripts(scripts []string, vars Vars) error {
	if len(scripts) == 0 {
		return nil
	}
	joined := ""
	for i, s := range scripts {
		if i > 0 {
			joined += "\n"
		}
		joined += s
	}
	return RunScript(joined, vars)
}

func buildScript(script string, vars Vars) string {
	globalStr := "$false"
	if vars.Global {
		globalStr = "$true"
	}

	preamble := fmt.Sprintf(`$dir = %s
$persist_dir = %s
$version = "%s"
$app = "%s"
$bucket = "%s"
$bucketsdir = %s
$architecture = "%s"
$global = %s
$cleanVersion = "%s"
$fname = "%s"
`, quotePath(vars.Dir), quotePath(vars.PersistDir), vars.Version, vars.App,
		vars.Bucket, quotePath(vars.BucketsDir), vars.Architecture, globalStr,
		vars.CleanVersion, vars.Fname)

	return preamble + script
}

func quotePath(path string) string {
	return "'" + escApos(path) + "'"
}

func escApos(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '\'')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}
