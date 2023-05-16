package logininfo

import (
	"log"
	"reflect"
	"testing"
)

// getTestLoginInfo returns login info for testing
func getTestLoginInfo() *LoginInfo {
	// COOKIE='3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...'
	// HOST='10.0.0.1'
	// CONNECT_URL='https://vpnserver.example.com'
	// FINGERPRINT='469bb424ec8835944d30bc77c77e8fc1d8e23a42'
	// RESOLVE='vpnserver.example.com:10.0.0.1'
	return &LoginInfo{
		Cookie:      "3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...",
		Host:        "10.0.0.1",
		ConnectURL:  "https://vpnserver.example.com",
		Fingerprint: "469bb424ec8835944d30bc77c77e8fc1d8e23a42",
		Resolve:     "vpnserver.example.com:10.0.0.1",
	}
}

// TestLoginInfoValid tests Valid of LoginInfo
func TestLoginInfoValid(t *testing.T) {
	// test invalid
	li := &LoginInfo{}
	want := false
	got := li.Valid()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}

	// test valid
	li = getTestLoginInfo()
	want = true
	got = li.Valid()
	if got != want {
		t.Errorf("got %t, want %t", got, want)
	}
}

// TestLoginInfoParseLine tests ParseLine of LoginInfo
func TestLoginInfoParseLine(t *testing.T) {
	want := getTestLoginInfo()
	got := &LoginInfo{}

	// parse lines (should match values returned by getTestLoginInfo)
	for _, line := range []string{
		"COOKIE='3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...'",
		"HOST='10.0.0.1'",
		"CONNECT_URL='https://vpnserver.example.com'",
		"FINGERPRINT='469bb424ec8835944d30bc77c77e8fc1d8e23a42'",
		"RESOLVE='vpnserver.example.com:10.0.0.1'",
	} {
		got.ParseLine(line)
	}

	// make sure result matches values returned by getTestLoginInfo
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoginInfoFromJSON tests JSON of LoginInfo and LoginInfoFromJSON
func TestLoginInfoFromJSON(t *testing.T) {
	// create login info
	want := getTestLoginInfo()

	// convert to json
	b, err := want.JSON()
	if err != nil {
		log.Fatal(err)
	}

	// parse json
	got, err := LoginInfoFromJSON(b)
	if err != nil {
		log.Fatal(err)
	}

	// make sure both login infos match
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
