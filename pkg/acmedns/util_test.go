package acmedns

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/bcrypt"
	"os"
	"syscall"
	"testing"
)

func fakeConfig() AcmeDnsConfig {
	conf := AcmeDnsConfig{}
	conf.Logconfig.Logtype = "stdout"
	return conf
}

func TestSetupLogging(t *testing.T) {
	conf := fakeConfig()
	for i, test := range []struct {
		format   string
		level    string
		expected zapcore.Level
	}{
		{"text", "warn", zap.WarnLevel},
		{"json", "debug", zap.DebugLevel},
		{"text", "info", zap.InfoLevel},
		{"json", "error", zap.ErrorLevel},
	} {
		conf.Logconfig.Format = test.format
		conf.Logconfig.Level = test.level
		logger, err := SetupLogging(conf)
		if err != nil {
			t.Errorf("Got unexpected error: %s", err)
		} else {
			if logger.Sugar().Level() != test.expected {
				t.Errorf("Test %d: Expected loglevel %s but got %s", i, test.expected, logger.Sugar().Level())
			}
		}
	}
}

func TestSetupLoggingError(t *testing.T) {
	conf := fakeConfig()
	for _, test := range []struct {
		format      string
		level       string
		errexpected bool
	}{
		{"text", "warn", false},
		{"json", "debug", false},
		{"text", "info", false},
		{"json", "error", false},
		{"text", "something", true},
	} {
		conf.Logconfig.Format = test.format
		conf.Logconfig.Level = test.level
		_, err := SetupLogging(conf)
		if test.errexpected && err == nil {
			t.Errorf("Expected error but did not get one for loglevel: %s", err)
		}
	}
}

func TestReadConfig(t *testing.T) {
	for i, test := range []struct {
		inFile []byte
		output AcmeDnsConfig
	}{
		{
			[]byte("[general]\nlisten = \":53\"\ndebug = true\n[api]\napi_domain = \"something.strange\""),
			AcmeDnsConfig{
				General: general{
					Listen: ":53",
					Debug:  true,
				},
				API: httpapi{
					Domain: "something.strange",
				},
			},
		},

		{
			[]byte("[\x00[[[[[[[[[de\nlisten =]"),
			AcmeDnsConfig{},
		},
	} {
		tmpfile, err := os.CreateTemp("", "acmedns")
		if err != nil {
			t.Error("Could not create temporary file")
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(test.inFile); err != nil {
			t.Error("Could not write to temporary file")
		}

		if err := tmpfile.Close(); err != nil {
			t.Error("Could not close temporary file")
		}
		ret, _, _ := ReadConfig(tmpfile.Name())
		if ret.General.Listen != test.output.General.Listen {
			t.Errorf("Test %d: Expected listen value %s, but got %s", i, test.output.General.Listen, ret.General.Listen)
		}
		if ret.API.Domain != test.output.API.Domain {
			t.Errorf("Test %d: Expected HTTP API domain %s, but got %s", i, test.output.API.Domain, ret.API.Domain)
		}
	}
}

func TestFileCheckPermissionDenied(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "acmedns")
	if err != nil {
		t.Error("Could not create temporary file")
	}
	defer os.Remove(tmpfile.Name())
	_ = syscall.Chmod(tmpfile.Name(), 0000)
	if FileIsAccessible(tmpfile.Name()) {
		t.Errorf("File should not be accessible")
	}
	_ = syscall.Chmod(tmpfile.Name(), 0644)
}

func TestFileCheckNotExists(t *testing.T) {
	if FileIsAccessible("/path/that/does/not/exist") {
		t.Errorf("File should not be accessible")
	}
}

func TestFileCheckOK(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "acmedns")
	if err != nil {
		t.Error("Could not create temporary file")
	}
	defer os.Remove(tmpfile.Name())
	if !FileIsAccessible(tmpfile.Name()) {
		t.Errorf("File should be accessible")
	}
}

func TestPrepareConfig(t *testing.T) {
	for i, test := range []struct {
		input       AcmeDnsConfig
		shoulderror bool
	}{
		{AcmeDnsConfig{Database: dbsettings{Engine: "whatever", Connection: "whatever_too"}}, false},
		{AcmeDnsConfig{Database: dbsettings{Engine: "", Connection: "whatever_too"}}, true},
		{AcmeDnsConfig{Database: dbsettings{Engine: "whatever", Connection: ""}}, true},
	} {
		_, err := prepareConfig(test.input)
		if test.shoulderror {
			if err == nil {
				t.Errorf("Test %d: Expected error with prepareConfig input data [%v]", i, test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Test %d: Expected no error with prepareConfig input data [%v]", i, test.input)
			}
		}
	}
}

func TestSanitizeString(t *testing.T) {
	for i, test := range []struct {
		input    string
		expected string
	}{
		{"abcd!abcd", "abcdabcd"},
		{"ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz0123456789", "ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz0123456789"},
		{"ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopq=@rstuvwxyz0123456789", "ABCDEFGHIJKLMNOPQRSTUVXYZabcdefghijklmnopqrstuvwxyz0123456789"},
	} {
		if SanitizeString(test.input) != test.expected {
			t.Errorf("Expected SanitizeString to return %s for test %d, but got %s instead", test.expected, i, SanitizeString(test.input))
		}
	}
}

func TestCorrectPassword(t *testing.T) {
	testPass, _ := bcrypt.GenerateFromPassword([]byte("nevergonnagiveyouup"), 10)
	for i, test := range []struct {
		input    string
		expected bool
	}{
		{"abcd", false},
		{"nevergonnagiveyouup", true},
		{"@rstuvwxyz0123456789", false},
	} {
		if test.expected && !CorrectPassword(test.input, string(testPass)) {
			t.Errorf("Expected CorrectPassword to return %t for test %d", test.expected, i)
		}
		if !test.expected && CorrectPassword(test.input, string(testPass)) {
			t.Errorf("Expected CorrectPassword to return %t for test %d", test.expected, i)
		}
	}
}
