package disguise

import "fmt"

// --- docker theme ---

type dockerTheme struct{}

func (dockerTheme) Name() string { return "docker" }

var dockerSvcs = []string{"web", "api", "worker", "redis", "db"}

func (dockerTheme) LinePrefix(seed int) string {
	return fmt.Sprintf("moyu-%s-1| ", dockerSvcs[seed%len(dockerSvcs)])
}
func (dockerTheme) Header(width int, status string) string {
	return padBetween("docker compose up", "● running", width)
}
func (dockerTheme) Footer(width int, status string) string {
	return padBetween("[+] Running 5/5 · "+status, "? help", width)
}
func (dockerTheme) BossLine(seed int) string {
	return fitLine(dockerTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- npm theme ---

type npmTheme struct{}

func (npmTheme) Name() string { return "npm" }

var npmPrefixes = []string{"npm WARN deprecated ", "npm http fetch GET 200 ", "npm timing build:run ", "npm info run "}

func (npmTheme) LinePrefix(seed int) string { return npmPrefixes[seed%len(npmPrefixes)] }
func (npmTheme) Header(width int, status string) string {
	return padBetween("npm install", "⠹", width)
}
func (npmTheme) Footer(width int, status string) string {
	return padBetween("added 1287 packages in 14s · "+status, "? help", width)
}
func (npmTheme) BossLine(seed int) string {
	return fitLine(npmTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- pytest theme ---

type pytestTheme struct{}

func (pytestTheme) Name() string { return "pytest" }

var pytestMods = []string{"core", "api", "auth", "models", "utils", "cache"}

func (pytestTheme) LinePrefix(seed int) string {
	return fmt.Sprintf("tests/test_%s.py::test_%d ", pytestMods[seed%len(pytestMods)], seed%97)
}
func (pytestTheme) Header(width int, status string) string {
	return padBetween("pytest -v", "● running", width)
}
func (pytestTheme) Footer(width int, status string) string {
	return padBetween("== 142 passed in 3.21s == · "+status, "? help", width)
}
func (pytestTheme) BossLine(seed int) string {
	return fitLine(pytestTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}
