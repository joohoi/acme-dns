package acmedns

import (
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"syscall"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/bcrypt"
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
		file        string
		errexpected bool
	}{
		{"text", "warn", "", false},
		{"json", "debug", "", false},
		{"text", "info", "", false},
		{"json", "error", "", false},
		{"text", "something", "", true},
		{"text", "info", "a path with\" in its name.txt", false},
	} {
		conf.Logconfig.Format = test.format
		conf.Logconfig.Level = test.level
		if test.file != "" {
			conf.Logconfig.File = test.file
			conf.Logconfig.Logtype = "file"

		}
		_, err := SetupLogging(conf)
		if test.errexpected && err == nil {
			t.Errorf("Expected error but did not get one for loglevel: %s", err)
		} else if !test.errexpected && err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		// clean up the file zap creates
		if test.file != "" {
			_ = os.Remove(test.file)
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
			t.Fatalf("Could not create temporary file: %s", err)
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(test.inFile); err != nil {
			t.Error("Could not write to temporary file")
		}

		if err := tmpfile.Close(); err != nil {
			t.Error("Could not close temporary file")
		}
		ret, _, _ := ReadConfig(tmpfile.Name(), "")
		if ret.General.Listen != test.output.General.Listen {
			t.Errorf("Test %d: Expected listen value %s, but got %s", i, test.output.General.Listen, ret.General.Listen)
		}
		if ret.API.Domain != test.output.API.Domain {
			t.Errorf("Test %d: Expected HTTP API domain %s, but got %s", i, test.output.API.Domain, ret.API.Domain)
		}
	}
}

func TestReadConfigFallback(t *testing.T) {
	var (
		path string
		err  error
	)

	testPath := "testdata/test_read_fallback_config.toml"

	path, err = getNonExistentPath()
	if err != nil {
		t.Errorf("failed getting non existant path: %s", err)
	}

	cfg, used, err := ReadConfig(path, testPath)
	if err != nil {
		t.Fatalf("failed to read a config file when we should have: %s", err)
	}

	if used != testPath {
		t.Fatalf("we read from the wrong file. got: %s, want: %s", used, testPath)
	}

	expected := AcmeDnsConfig{
		General: general{
			Listen:  "127.0.0.1:53",
			Proto:   "both",
			Domain:  "test.example.org",
			Nsname:  "test.example.org",
			Nsadmin: "test.example.org",
			Debug:   true,
			StaticRecords: []string{
				"test.example.org. A 127.0.0.1",
				"test.example.org. NS test.example.org.",
			},
		},
		Database: dbsettings{
			Engine:     "dinosaur",
			Connection: "roar",
		},
		API: httpapi{
			Domain:              "",
			IP:                  "0.0.0.0",
			DisableRegistration: false,
			AutocertPort:        "",
			Port:                "443",
			TLS:                 "none",
			TLSCertPrivkey:      "/etc/tls/example.org/privkey.pem",
			TLSCertFullchain:    "/etc/tls/example.org/fullchain.pem",
			ACMECacheDir:        "api-certs",
			NotificationEmail:   "",
			CorsOrigins:         []string{"*"},
			UseHeader:           true,
			HeaderName:          "X-is-gonna-give-it-to-ya",
		},
		Logconfig: logconfig{
			Level:   "info",
			Logtype: "stdout",
			File:    "./acme-dns.log",
			Format:  "json",
		},
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("Did not read the config correctly: got %+v, want: %+v", cfg, expected)
	}

}

func getNonExistentPath() (string, error) {
	path := fmt.Sprintf("/some/path/that/should/not/exist/on/any/filesystem/%10d.cfg", rand.Int())

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path, nil
	}

	return "", fmt.Errorf("attempted non existant file exists!?: %s", path)
}

// TestReadConfigFallbackError makes sure we error when we do not have a fallback config file
func TestReadConfigFallbackError(t *testing.T) {
	var (
		badPaths []string
		i        int
	)
	for len(badPaths) < 2 && i < 10 {
		i++

		if path, err := getNonExistentPath(); err == nil {
			badPaths = append(badPaths, path)
		}
	}

	if len(badPaths) != 2 {
		t.Fatalf("did not create exactly 2 bad paths")
	}

	_, _, err := ReadConfig(badPaths[0], badPaths[1])
	if err == nil {
		t.Errorf("Should have failed reading non existant file: %s", err)
	}
}

func TestFileCheckPermissionDenied(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "acmedns")
	if err != nil {
		t.Fatalf("Could not create temporary file: %s", err)
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
		t.Fatalf("Could not create temporary file: %s", err)
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
		{AcmeDnsConfig{
			General:  general{Domain: "example.com"},
			Database: dbsettings{Engine: "whatever", Connection: "whatever_too"},
			API:      httpapi{TLS: ApiTlsProviderNone},
		}, false},
		{AcmeDnsConfig{
			General:  general{Domain: "example.com"},
			Database: dbsettings{Engine: "", Connection: "whatever_too"},
			API:      httpapi{TLS: ApiTlsProviderNone},
		}, true},
		{AcmeDnsConfig{
			General:  general{Domain: "example.com"},
			Database: dbsettings{Engine: "whatever", Connection: ""},
			API:      httpapi{TLS: ApiTlsProviderNone},
		}, true},
		{AcmeDnsConfig{
			General:  general{Domain: "example.com"},
			Database: dbsettings{Engine: "whatever", Connection: "whatever_too"},
			API:      httpapi{TLS: "whatever"},
		}, true},
	} {
		_, err := prepareConfig(test.input)
		if test.shoulderror {
			if err == nil {
				t.Errorf("Test %d: Expected error with prepareConfig input data [%v]", i, test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Test %d: Expected no error with prepareConfig input data [%v], got %s", i, test.input, err)
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
