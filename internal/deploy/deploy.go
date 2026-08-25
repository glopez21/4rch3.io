package deploy

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/4rch3/portal/internal/config"
)

type Runner struct {
	targets map[string]config.DeployTarget
	timeout time.Duration
}

func New(cfg config.DeployConfig) *Runner {
	r := &Runner{
		targets: make(map[string]config.DeployTarget),
		timeout: cfg.Timeout,
	}
	for _, t := range cfg.Targets {
		r.targets[t.Service] = t
	}
	if r.timeout <= 0 {
		r.timeout = 5 * time.Minute
	}
	return r
}

func (r *Runner) HasTargets() bool {
	return len(r.targets) > 0
}

func (r *Runner) Allowed(service string) bool {
	_, ok := r.targets[service]
	return ok
}

func (r *Runner) Targets() []string {
	names := make([]string, 0, len(r.targets))
	for n := range r.targets {
		names = append(names, n)
	}
	return names
}

// Deploy runs `docker compose -f <dir>/<file>... pull` then `up -d`
// for a whitelisted service. Returns combined command output.
func (r *Runner) Deploy(ctx context.Context, service string) (string, error) {
	t, ok := r.targets[service]
	if !ok {
		return "", fmt.Errorf("service %q is not in the deploy whitelist", service)
	}

	if t.Dir == "" {
		return "", fmt.Errorf("service %q has no project dir configured", service)
	}

	args := []string{"compose"}
	for _, f := range t.Files {
		args = append(args, "-f", filepath.Join(t.Dir, f))
	}

	var out bytes.Buffer
	for _, op := range []string{"pull", "up"} {
		cmdArgs := append(append([]string{}, args...), op)
		if op == "up" {
			cmdArgs = append(cmdArgs, "-d")
		}
		cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = t.Dir
		if err := cmd.Run(); err != nil {
			return out.String(), fmt.Errorf("docker compose %s failed: %w", op, err)
		}
	}

	return out.String(), nil
}

func (r *Runner) ValidateConfig() error {
	for name, t := range r.targets {
		if t.Dir == "" {
			return fmt.Errorf("deploy target %q: missing dir", name)
		}
		if len(t.Files) == 0 {
			return fmt.Errorf("deploy target %q: no compose files", name)
		}
		for _, f := range t.Files {
			if !strings.HasSuffix(f, ".yml") && !strings.HasSuffix(f, ".yaml") {
				return fmt.Errorf("deploy target %q: compose file %q must be .yml/.yaml", name, f)
			}
		}
	}
	return nil
}
