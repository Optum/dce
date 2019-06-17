package common

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat/go-jwx/jwk"
)

// JWTTokenService interface implements methods for JWT parse and key matching
type JWTTokenService interface {
	ParseJWT() error
	getKey(token *jwt.Token) (interface{}, error)
	getTID(token string) (string, error)
}

// JWT implements the JWTTokenService interface
type JWT struct {
	Req    *events.APIGatewayProxyRequest
	Claims *ClaimKey
}

// ClaimKey contains a parsed JWT claim from graph query
type ClaimKey struct {
	UserID   string
	TenantID string
	GroupID  string
}

// getKey pulls keys from provider for JWT parse/verify
func getKey(token *jwt.Token) (interface{}, error) {
	activeDirectoryEndpoint := "https://login.microsoftonline.com/"
	tenantID, err := getTID(token.Raw)
	if err != nil {
		log.Println("jwt value parse failed.")
		return nil, err
	}

	jwksURL := fmt.Sprintf("%s%s/discovery/v2.0/keys", activeDirectoryEndpoint, tenantID)

	set, err := jwk.FetchHTTP(jwksURL)
	if err != nil {
		return nil, err
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("Failed to find string 'kid' in JWT header")
	}
	if key := set.LookupKeyID(keyID); len(key) == 1 {
		return key[0].Materialize()
	}

	return nil, errors.New("Failed Key retrieval from provider")
}

// ParseJWT parse/verify, shorter, faster, and we pull the claims for user OID and Tenant ID
func (j *JWT) ParseJWT() error {
	splitToken := strings.Split(j.Req.Headers["Authorization"], " ")
	jwtToken := splitToken[len(splitToken)-1]

	token, err := jwt.Parse(jwtToken, getKey)
	if err != nil {
		log.Println("jwt claims parse failed.")
		return err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		j.Claims.UserID = claims["oid"].(string)
		j.Claims.TenantID = claims["tid"].(string)
	} else {
		log.Println("JWT Claims Verification failed.")
		return errors.New("jwt claims verification failed")
	}

	return nil
}

// getTID function is local only, parsing the jwt for the decoded Tenant ID for Azure
// This is required for the jwk url, so we Parse, but don't validate for this op
func getTID(token string) (string, error) {
	jwtSlc := strings.Split(token, ".")
	decoded, err := base64.RawStdEncoding.DecodeString(jwtSlc[1])
	if err != nil {
		log.Println("JWT TenantID parse failed to decode.")
		return "", err
	}
	var f interface{}
	err = json.Unmarshal(decoded, &f)
	if err != nil {
		log.Println("JWT TenantID parse failed to unmarshal.")
		return "", err
	}
	m := f.(map[string]interface{})
	for k, v := range m {
		if k == "tid" {
			return v.(string), nil
		}
	}
	return "", errors.New("jwt claims parse failed")
}
