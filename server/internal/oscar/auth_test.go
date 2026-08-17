package oscar

import (
	"database/sql"
	"testing"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/db"
	_ "modernc.org/sqlite"
)

func testAuthService(t *testing.T) accounts.Service {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Initialize(database); err != nil { t.Fatal(err) }
	service := accounts.Service{DB: database}
	if _, err := service.Register("testotter", "Test Otter", "", "test-password-12345"); err != nil { t.Fatal(err) }
	return service
}

func TestAuthenticateLogin(t *testing.T) {
	service := testAuthService(t)
	auth := Authenticator{Accounts: service, ReconnectURL: "127.0.0.1:5190"}

	request := SNAC{
		Family: SNACBUCP, Subtype: BUCPLoginRequest, Flags: 0, RequestID: 42,
		Payload: mustTLVs(t, []TLV{
			{Tag: TLVScreenName, Value: []byte("testotter")},
			{Tag: TLVPassword, Value: []byte("test-password-12345")},
		}),
	}
	response, err := auth.AuthenticateLogin(request)
	if err != nil { t.Fatal(err) }
	if response.Family != SNACBUCP || response.Subtype != BUCPLoginResponse || response.RequestID != 42 { t.Fatalf("unexpected response header: %+v", response) }

	tlvs, err := ParseTLVs(response.Payload)
	if err != nil { t.Fatal(err) }
	values := map[uint16]string{}
	for _, tlv := range tlvs { values[tlv.Tag] = string(tlv.Value) }
	if values[TLVScreenName] != "testotter" { t.Fatalf("screen name = %q", values[TLVScreenName]) }
	if values[TLVReconnectHere] != "127.0.0.1:5190" { t.Fatalf("reconnect URL = %q", values[TLVReconnectHere]) }
	if values[TLVAuthorizationCookie] == "" { t.Fatal("missing authorization cookie") }
}

func TestAuthenticateLoginRejectsBadPassword(t *testing.T) {
	service := testAuthService(t)
	auth := Authenticator{Accounts: service}
	request := SNAC{Family: SNACBUCP, Subtype: BUCPLoginRequest, RequestID: 7, Payload: mustTLVs(t, []TLV{
		{Tag: TLVScreenName, Value: []byte("testotter")},
		{Tag: TLVPassword, Value: []byte("wrong-password-12345")},
	})}
	if _, err := auth.AuthenticateLogin(request); err == nil { t.Fatal("expected bad password to fail") }
}

func mustTLVs(t *testing.T, tlvs []TLV) []byte {
	t.Helper()
	data, err := EncodeTLVs(tlvs)
	if err != nil { t.Fatal(err) }
	return data
}
