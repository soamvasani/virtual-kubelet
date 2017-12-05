// +build !test_no_exec

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/pkg/integration/checker"
	"github.com/go-check/check"
)

// Regression test for #9414
func (s *DockerSuite) TestApiExecBasicCreateNoCmd(c *check.C) {
	printTestCaseName()
	defer printTestDuration(time.Now())

	name := "exec-test"
	dockerCmd(c, "run", "-d", "-t", "--name", name, "busybox", "top")

	status, body, err := sockRequest("POST", fmt.Sprintf("/containers/%s/exec", name), map[string]interface{}{"Cmd": nil})
	c.Assert(err, checker.IsNil)
	c.Assert(status, checker.Equals, http.StatusBadRequest)

	comment := check.Commentf("Expected message when creating exec command with no Cmd specified")
	c.Assert(string(body), checker.Contains, "No exec command specified", comment)
}

func (s *DockerSuite) TestApiExecBasicCreateNoValidContentType(c *check.C) {
	printTestCaseName()
	defer printTestDuration(time.Now())

	name := "exec-test"
	dockerCmd(c, "run", "-d", "-t", "--name", name, "busybox", "/bin/sh")

	jsonData := bytes.NewBuffer(nil)
	if err := json.NewEncoder(jsonData).Encode(map[string]interface{}{"Cmd": nil}); err != nil {
		c.Fatalf("Can not encode data to json %s", err)
	}

	res, body, err := sockRequestRaw("POST", fmt.Sprintf("/containers/%s/exec", name), jsonData, "text/plain")
	c.Assert(err, checker.IsNil)
	c.Assert(res.StatusCode, checker.Equals, http.StatusInternalServerError)

	b, err := readBody(body)
	c.Assert(err, checker.IsNil)

	comment := check.Commentf("Expected message when creating exec command with invalid Content-Type specified")
	c.Assert(string(b), checker.Equals, "The server encountered an internal error or misconfiguration...\n", comment)
}

//TODO: fix #86
/*func (s *DockerSuite) TestApiExecBasicApiStart(c *check.C) {
	printTestCaseName()
	defer printTestDuration(time.Now())

	testRequires(c, DaemonIsLinux) // Uses pause/unpause but bits may be salvagable to Windows to Windows CI
	dockerCmd(c, "run", "-d", "--name", "test", "busybox", "top")

	id := createExec(c, "test")
	startExec(c, id, http.StatusOK)

	id = createExec(c, "test")
	dockerCmd(c, "stop", "test")

	startExec(c, id, http.StatusNotFound)

	dockerCmd(c, "start", "test")
	startExec(c, id, http.StatusNotFound)
}

func (s *DockerSuite) TestApiExecBasicApiStartMultipleTimesError(c *check.C) {
	printTestCaseName()
	defer printTestDuration(time.Now())

	runSleepingContainer(c, "-d", "--name", "test")
	execID := createExec(c, "test")
	startExec(c, execID, http.StatusOK)

	timeout := time.After(60 * time.Second)
	var execJSON struct{ Running bool }
	for {
		select {
		case <-timeout:
			c.Fatal("timeout waiting for exec to start")
		default:
		}

		inspectExec(c, execID, &execJSON)
		if !execJSON.Running {
			break
		}
	}

	startExec(c, execID, http.StatusConflict)
}*/

func createExec(c *check.C, name string) string {
	_, b, err := sockRequest("POST", fmt.Sprintf("/containers/%s/exec", name), map[string]interface{}{"Cmd": []string{"true"}})
	c.Assert(err, checker.IsNil, check.Commentf(string(b)))

	createResp := struct {
		ID string `json:"Id"`
	}{}
	c.Assert(json.Unmarshal(b, &createResp), checker.IsNil, check.Commentf(string(b)))
	return createResp.ID
}

func startExec(c *check.C, id string, code int) {
	resp, body, err := sockRequestRaw("POST", fmt.Sprintf("/exec/%s/start", id), strings.NewReader(`{"Detach": true}`), "application/json")
	c.Assert(err, checker.IsNil)

	b, err := readBody(body)
	comment := check.Commentf("response body: %s", b)
	c.Assert(err, checker.IsNil, comment)
	c.Assert(resp.StatusCode, checker.Equals, code, comment)
}

func inspectExec(c *check.C, id string, out interface{}) {
	resp, body, err := sockRequestRaw("GET", fmt.Sprintf("/exec/%s/json", id), nil, "")
	c.Assert(err, checker.IsNil)
	defer body.Close()
	c.Assert(resp.StatusCode, checker.Equals, http.StatusOK)
	err = json.NewDecoder(body).Decode(out)
	c.Assert(err, checker.IsNil)
}