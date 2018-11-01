// +build unit

package main

import (
	"io/ioutil"
	"os/exec"
	"testing"
)

func TestProcessReq(t *testing.T) {
	t.Log("test process req")
	app := App{}

	b, err := ioutil.ReadFile("./req_file_test.txt")
	if err != nil {
		t.Fatalf("err reading from file\n%v", err)
	}

	if err := app.processReq(string(b)); err != nil {
		t.Fatalf("err processing req\n%v", err)
	}
}

func TestShellScripts(t *testing.T) {
	cmd := exec.Command("/bin/sh", "./set_state.sh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("err running script\n%v\nnoutput:\n%s", err, string(out))
	}

	t.Log(string(out))
	// if err := cmd.Start(); err != nil {
	// 	t.Fatalf("err executing set-state\n%v", err)
	// }

	//if err := cmd.Wait(); err != nil {
	//	t.Fatalf("err waiting on test\n%v", err)
	//}
}
