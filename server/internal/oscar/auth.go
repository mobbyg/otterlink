package oscar

import (
	"errors"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

const TLVPassword uint16 = 0x0002

type Authenticator struct {
	Accounts accounts.Service
	ReconnectURL string
}

// AuthenticateLogin translates the OSCAR BUCP login request into an Otter Link
// account authentication and returns an OSCAR BUCP login response SNAC.
// This is the bridge boundary: OSCAR credentials are validated by the native
// Otter Link account service, while the resulting session token is returned as
// the OSCAR authorization cookie.
func (a Authenticator) AuthenticateLogin(snac SNAC) (SNAC, error) {
	if snac.Family != SNACBUCP || snac.Subtype != BUCPLoginRequest {
		return SNAC{}, errors.New("not an OSCAR BUCP login request")
	}
	tlvs, err := ParseTLVs(snac.Payload)
	if err != nil {
		return SNAC{}, err
	}
	username, password := "", ""
	for _, tlv := range tlvs {
		switch tlv.Tag {
		case TLVScreenName:
			username = strings.TrimSpace(string(tlv.Value))
		case TLVPassword:
			password = string(tlv.Value)
		}
	}
	if username == "" || password == "" {
		return SNAC{}, errors.New("OSCAR login requires screen name and password")
	}

	user, token, err := a.Accounts.Authenticate(username, password)
	if err != nil {
		return SNAC{}, errors.New("invalid username or password")
	}

	payload, err := EncodeTLVs([]TLV{
		{Tag: TLVReconnectHere, Value: []byte(a.ReconnectURL)},
		{Tag: TLVAuthorizationCookie, Value: []byte(token)},
		{Tag: TLVScreenName, Value: []byte(user.Username)},
	})
	if err != nil {
		return SNAC{}, err
	}
	return SNAC{
		Family: SNACBUCP,
		Subtype: BUCPLoginResponse,
		Flags: snac.Flags,
		RequestID: snac.RequestID,
		Payload: payload,
	}, nil
}
