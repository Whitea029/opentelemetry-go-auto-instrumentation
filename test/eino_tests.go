package test

import "testing"

const eino_dependency_name = "github.com/cloudwego/eino"
const eino_module_name = "eino"

func init() {
	TestCases = append(TestCases,
		NewGeneralTestCase("eino-0.3.36-invoke-agent-test", eino_module_name, "v0.3.36", "", "1.18", "", TestAgentInvokeEino),
		NewGeneralTestCase("eino-0.3.36-stream-agent-test", eino_module_name, "v0.3.36", "", "1.18", "", TestAgentStreamEino),
		NewMuzzleTestCase("eino-muzzle", eino_dependency_name, eino_module_name, "v0.3.36", "", "1.18", "", []string{"go", "build", "test_invoke_agent.go", "eino_common.go"}),
		NewLatestDepthTestCase("eino-latest-depth", eino_dependency_name, eino_module_name, "v0.3.36", "v0.3.37", "1.18", "", TestAgentInvokeEino),
	)
}

func TestAgentInvokeEino(t *testing.T, env ...string) {
	UseApp("eino/v0.3.36")
	RunGoBuild(t, "go", "build", "test_invoke_agent.go", "eino_common.go")
	RunApp(t, "test_invoke_agent", env...)
}

func TestAgentStreamEino(t *testing.T, env ...string) {
	UseApp("eino/v0.3.36")
	RunGoBuild(t, "go", "build", "test_stream_agent.go", "eino_common.go")
	RunApp(t, "test_stream_agent", env...)
}
