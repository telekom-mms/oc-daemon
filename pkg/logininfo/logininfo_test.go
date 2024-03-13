package logininfo

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// getTestLoginInfo returns login info for testing.
func getTestLoginInfo() *LoginInfo {
	// COOKIE='3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...'
	// HOST='10.0.0.1'
	// CONNECT_URL='https://vpnserver.example.com'
	// FINGERPRINT='469bb424ec8835944d30bc77c77e8fc1d8e23a42'
	// RESOLVE='vpnserver.example.com:10.0.0.1'
	return &LoginInfo{
		Server:      "vpnserver.example.com",
		Cookie:      "3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...",
		Host:        "10.0.0.1",
		ConnectURL:  "https://vpnserver.example.com",
		Fingerprint: "469bb424ec8835944d30bc77c77e8fc1d8e23a42",
		Resolve:     "vpnserver.example.com:10.0.0.1",
	}
}

// TestLoginInfoCopy tests Copy of LoginInfo.
func TestLoginInfoCopy(t *testing.T) {
	// test nil
	if (*LoginInfo)(nil).Copy() != nil {
		t.Error("copy of nil should be nil")
	}

	// test valid config
	want := getTestLoginInfo()
	got := want.Copy()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestLoginInfoValid tests Valid of LoginInfo.
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

// TestLoginInfoParseLine tests ParseLine of LoginInfo.
func TestLoginInfoParseLine(t *testing.T) {
	want := getTestLoginInfo()
	got := &LoginInfo{}
	got.Server = "vpnserver.example.com"

	// parse lines (should match values returned by getTestLoginInfo)
	for _, line := range []string{
		"COOKIE='3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...'",
		"HOST='10.0.0.1'",
		"CONNECT_URL='https://vpnserver.example.com'",
		"FINGERPRINT='469bb424ec8835944d30bc77c77e8fc1d8e23a42'",
		"RESOLVE='vpnserver.example.com:10.0.0.1'",
		"And make sure other lines do not break anything",
	} {
		got.ParseLine(line)
	}

	// make sure result matches values returned by getTestLoginInfo
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestFromJSON tests JSON of LoginInfo and FromJSON.
func TestFromJSON(t *testing.T) {
	// create login info
	want := getTestLoginInfo()

	// convert to json
	b, err := want.JSON()
	if err != nil {
		t.Fatal(err)
	}

	// parse json
	got, err := FromJSON(b)
	if err != nil {
		t.Fatal(err)
	}

	// make sure both login infos match
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// test with marshal and parse error
	jsonMarshal = func(any) ([]byte, error) {
		return nil, errors.New("test error")
	}
	defer func() { jsonMarshal = json.Marshal }()

	b, err = want.JSON()
	if err == nil {
		t.Error("Marshal error should return error")
	}
	if _, err = FromJSON(b); err == nil {
		t.Error("parsing nil should return error")
	}
}
