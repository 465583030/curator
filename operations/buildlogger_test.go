package operations

import (
	"os"
	"os/exec"
	"testing"

	"github.com/mongodb/grip"
	"github.com/stretchr/testify/assert"
)

func TestBuildLoggerRunCommand(t *testing.T) {
	var err error
	var cmd *exec.Cmd

	assert := assert.New(t)
	logger := grip.NewJournaler("buildlogger.test")

	// error and non-error cases should both work as expected
	err = runCommand(logger, exec.Command("ls"))
	grip.Info(err)
	assert.NoError(err)

	err = runCommand(logger, exec.Command("dfkjdexit", "0"))
	grip.Info(err)
	assert.Error(err)

	// want to make sure that we exercise the path with too-small buffers.
	err = runCommand(logger, exec.Command("ls"))
	grip.Info(err)
	assert.NoError(err)

	err = runCommand(logger, exec.Command("dfkjdexit", "0"))
	grip.Info(err)
	assert.Error(err)

	// runCommand should error if the output streams are pre set.
	cmd = &exec.Cmd{}
	cmd.Stderr = os.Stderr
	err = runCommand(logger, cmd)
	grip.Info(err)
	assert.Error(err)

	cmd = &exec.Cmd{}
	cmd.Stdout = os.Stdout
	err = runCommand(logger, cmd)
	grip.Info(err)
	assert.Error(err)
}
