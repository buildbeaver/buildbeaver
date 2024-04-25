package credential

import (
	"crypto"
	"fmt"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/golang-jwt/jwt/v4"
)

const (
	DefaultJWTExpiryDuration = 24 * time.Hour
	DefaultJWTIssuer         = "BuildBeaver"
)

type IdentityTokenClaims struct {
	jwt.RegisteredClaims
}

type BuildTokenClaims struct {
	BuildID string `json:"build_id,omitempty"`
	jwt.RegisteredClaims
}

// CreateIdentityJWT creates a new JWT (JSON Web Token) credential that can be used to authenticate as
// the specified identity. The JWT will be signed using the supplied private key.
func CreateIdentityJWT(
	identityID models.IdentityID,
	issuer string,
	expiryDuration time.Duration,
	privateKey crypto.PrivateKey,
) (string, *IdentityTokenClaims, error) {
	// Create the claims.
	// The Subject is the identity being granted access - this JWT is about the identity and the fact that
	// the bearer can act as this identity. Further claims could be added to restrict what the user can
	// do as this identity.
	claims := &IdentityTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   identityID.String(), // the identity being granted access is the subject
		},
	}

	// Sign using Ed25519 private key, and get the complete encoded token as a string
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims) // EdDSA is an instance of SigningMethodEd25519
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", nil, err
	}

	return tokenString, claims, nil
}

// VerifyIdentityJWT verifies the signature on the supplied JWT (JSON Web Token) and returns the identity ID
// specified in the subject field. The identity ID is NOT checked against the database.
// The JWT signature will be verified using the supplied public key.
func VerifyIdentityJWT(tokenStr string, publicKey crypto.PublicKey) (models.IdentityID, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &IdentityTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm is as expected
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("error unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return models.IdentityID{}, fmt.Errorf("error parsing identity JWT: %w", err)
	}
	claims := token.Claims.(*IdentityTokenClaims)

	resourceID, err := models.ParseResourceID(claims.Subject)
	if err != nil {
		return models.IdentityID{}, fmt.Errorf("error parsing JWT subject as identity ID: %w", err)
	}
	if resourceID.Kind() != models.IdentityResourceKind {
		return models.IdentityID{}, fmt.Errorf("error JWT subject must be an identity resource ID, but found resource ID with kind '%s'", resourceID.Kind())
	}

	return models.IdentityIDFromResourceID(resourceID), nil
}
