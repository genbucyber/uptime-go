package hooks

import "testing"

func TestExecuteCommand(t *testing.T){
	output, err := Execute("echo 'hello world'")
	if err != nil {
		t.Fatalf("Output Error: %v", err)
	}
	t.Logf("Output Succes: \n%s", output)
}

// func TestExecuteBash(t *testing.T){
// 	output, err := ExecuteBash("/tmp/bash-test.sh")
// 	if err != nil {
// 		t.Fatalf("Output Error: %v", err)
// 	}
// 	t.Logf("Output Succes: \n%s", output)
// }
