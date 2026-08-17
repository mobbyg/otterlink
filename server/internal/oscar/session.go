package oscar

import (
	"encoding/binary"
	"errors"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

// userFromBOSSignon resolves the authorization cookie carried on the BOS
// sign-on frame to the native Otter Link account. BUCP authentication happens
// on the authorization connection; the BOS connection then presents that
// cookie and must be associated with the same user for presence and buddy
// events to work.
func (s *Server) userFromBOSSignon(frame Frame) (accounts.User, error) {
	if len(frame.Payload) < 4 {
		return accounts.User{}, errors.New("BOS sign-on payload too short")
	}
	if binary.BigEndian.Uint32(frame.Payload[:4]) != 1 {
		return accounts.User{}, errors.New("unsupported BOS sign-on version")
	}
	tlvs, err := ParseTLVs(frame.Payload[4:])
	if err != nil {
		return accounts.User{}, err
	}
	for _, tlv := range tlvs {
		if tlv.Tag == TLVAuthorizationCookie {
			return s.Authenticator.Accounts.FromToken(string(tlv.Value))
		}
	}
	return accounts.User{}, errors.New("BOS sign-on missing authorization cookie")
}
