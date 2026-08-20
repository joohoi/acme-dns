package nameserver

import (
	"net"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/joohoi/acme-dns/pkg/acmedns"
	"github.com/joohoi/acme-dns/pkg/database"
)

// initAndStart runs InitAndStart with a deadline, so that a regression of the
// startup hang shows up as a failing test instead of a hanging test run.
func initAndStart(t *testing.T, config *acmedns.AcmeDnsConfig, db acmedns.AcmednsDB) ([]acmedns.AcmednsNS, error) {
	t.Helper()
	type result struct {
		servers []acmedns.AcmednsNS
		err     error
	}
	resChan := make(chan result, 1)
	go func() {
		servers, err := InitAndStart(config, db, testLogger(t), make(chan error, 1))
		resChan <- result{servers, err}
	}()
	select {
	case res := <-resChan:
		return res.servers, res.err
	case <-time.After(10 * time.Second):
		t.Fatal("InitAndStart did not return, the DNS server startup is stuck")
		return nil, nil
	}
}

func initConfig(t *testing.T, proto, listen string) (acmedns.AcmeDnsConfig, acmedns.AcmednsDB) {
	t.Helper()
	config, logger, _ := fakeConfigAndLogger()
	config.General.Domain = "auth.example.org"
	config.General.Listen = listen
	config.General.Proto = proto
	config.General.Nsname = "ns1.auth.example.org"
	config.General.Nsadmin = "admin.example.org"
	config.General.StaticRecords = records
	db, err := database.Init(&config, logger)
	if err != nil {
		t.Fatalf("Could not initialize database: %s", err)
	}
	return config, db
}

func testLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()
	_, logger, _ := fakeConfigAndLogger()
	return logger
}

func TestInitAndStartBoth(t *testing.T) {
	config, db := initConfig(t, "both", "127.0.0.1:15354")
	servers, err := initAndStart(t, &config, db)
	if err != nil {
		t.Errorf("Expected no error, but got: %s", err)
	}
	if len(servers) != 2 {
		t.Errorf("Expected 2 DNS servers for proto both, but got %d", len(servers))
	}
}

// TestInitAndStartUnusableProto makes sure that a protocol the DNS library can
// not listen on returns an error instead of waiting forever for a start
// notification that never arrives.
func TestInitAndStartUnusableProto(t *testing.T) {
	config, db := initConfig(t, "", "127.0.0.1:15355")
	if _, err := initAndStart(t, &config, db); err == nil {
		t.Errorf("Expected an error for an empty protocol, but got none")
	}
}

// TestInitAndStartPortInUse makes sure that a listener that can not bind its
// port reports the bind error instead of hanging the startup.
func TestInitAndStartPortInUse(t *testing.T) {
	addr := "127.0.0.1:15356"
	blocker, err := net.ListenPacket("udp", addr)
	if err != nil {
		t.Fatalf("Could not occupy the test port: %s", err)
	}
	defer blocker.Close()

	config, db := initConfig(t, "both", addr)
	if _, err := initAndStart(t, &config, db); err == nil {
		t.Errorf("Expected an error when the port is already in use, but got none")
	}
}
