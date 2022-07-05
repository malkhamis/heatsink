package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMain(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	origOsExit := osExit
	defer func() { osExit = origOsExit }()

	origNewLogger := newLogger
	defer func() { newLogger = origNewLogger }()

	newLogger = func() *zap.Logger { return zap.NewNop() }
	os.Args = nil
	osExit = func(actualExitCode int) {
		if expected := 64; actualExitCode != expected {
			t.Fatalf(
				"expected exit code to indicate invalid argument count\nwant: %d\n got:%v",
				expected, actualExitCode,
			)
		}
	}

	main()
}

func Test_execute(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	tmpFileConfig, cleanup := temporaryFile(t)
	defer cleanup()
	tmpFileFan, cleanup := temporaryFile(t)
	defer cleanup()
	tmpFileSensor, cleanup := temporaryFile(t)
	defer cleanup()

	validConfig := fmt.Sprintf(`
    {
      "heatsinks": [

        {
          "name":"heatsink/1",
          "min_temp": 35,
          "max_temp": 65,
          "temp_check_period": "1s",
          "sensor_path_globs": [%q],
          "fan": {
            "name": "fan/1",
            "path_glob": %q,
            "pwm_period": "50ms",
            "min_speed_value": "0",
            "max_speed_value": "255"
          }
        }
      ]
    }`,
		tmpFileSensor.Name(), tmpFileFan.Name(),
	)

	if _, err := tmpFileConfig.WriteString(validConfig); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"program-name", tmpFileConfig.Name()}
	actual := execute()
	if expected := 1; actual != expected {
		t.Fatalf("actual exit code doesn't match expected\nwant: %d\n got: %d", expected, actual)
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(
				string(logLine),
				`"msg":"thermal control returned an error"`,
			) {
				return // test passed
			}
		default:
		}
	}
}

func Test_execute_noFileArg(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	os.Args = []string{"program-name"}
	actual := execute()
	if expected := 64; actual != expected {
		t.Fatalf("actual exit code doesn't match expected\nwant: %d\n got: %d", expected, actual)
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(string(logLine), "no filepath given for json config") {
				return // test passed
			}
		default:
		}
	}
}

func Test_execute_fileNotExist(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	os.Args = []string{"program-name", "/this/file/does/not/exist"}
	actual := execute()
	if expected := 66; actual != expected {
		t.Fatalf("actual exit code doesn't match expected\nwant: %d\n got: %d", expected, actual)
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(
				string(logLine),
				`"msg":"opening the given file",`+
					`"error":"open /this/file/does/not/exist: no such file or directory"`,
			) {
				return // test passed
			}
		default:
		}
	}
}

func Test_execute_badJsonFile(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()
	if _, err := tmpFile.WriteString("{ bad json data }"); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"program-name", tmpFile.Name()}
	actual := execute()
	if expected := 78; actual != expected {
		t.Fatalf("actual exit code doesn't match expected\nwant: %d\n got: %d", expected, actual)
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(
				string(logLine),
				`"msg":"creating heatsink config","error":"error decoding json config: `+
					`invalid character 'b' looking for beginning of object key string"`,
			) {
				return // test passed
			}
		default:
		}
	}
}

func Test_execute_badHeatsinkConfig(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()
	if _, err := tmpFile.WriteString(`{"heatsinks":[]}`); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"program-name", tmpFile.Name()}
	actual := execute()
	if expected := 78; actual != expected {
		t.Fatalf("actual exit code doesn't match expected\nwant: %d\n got: %d", expected, actual)
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(
				string(logLine),
				`"msg":"creating heatsink config","error":"no heatsink config in given json data"`,
			) {
				return // test passed
			}
		default:
		}
	}
}

func Test_execute_invalidHeatsinkConfig(t *testing.T) {

	restoreProcArgs := backupProcArgs(t)
	defer restoreProcArgs()

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	tmpFile, cleanup := temporaryFile(t)
	defer cleanup()

	invalidConfig := `{
    "heatsinks":[
      {
        "min_temp": 10,
        "max_temp": 20
      }
    ]
  }`
	if _, err := tmpFile.WriteString(invalidConfig); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"program-name", tmpFile.Name()}
	actual := execute()
	if expected := 78; actual != expected {
		t.Fatalf("actual exit code doesn't match expected\nwant: %d\n got: %d", expected, actual)
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(
				string(logLine),
				`"msg":"instantiating heatsinks","error":"heatsink '': failed to create all sensors`,
			) {
				return // test passed
			}
		default:
		}
	}
}

func Test_getLoggerAndPrintErrIfAny(t *testing.T) {

	stdoutLines, streamErr, restoreStdout := stdoutStream(t)
	defer restoreStdout()

	orig := log.Writer()
	defer log.SetOutput(orig)
	log.SetOutput(os.Stdout)

	actual := getLoggerAndPrintErrIfAny(nil, errors.New("simulated error"))
	if actual == nil {
		t.Fatal("expected a non-nil logger")
	}

	for deadline := time.After(1 * time.Second); ; {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for the expected log entry")
		case err := <-streamErr:
			t.Fatalf("reading stdout stream: %v", err)
		case logLine := <-stdoutLines:
			if strings.Contains(string(logLine), "error creating logger: simulated error") {
				return // test passed
			}
		default:
		}
	}
}
