package api

import (
	"context"
	"fmt"
	"github.com/Optum/dce/pkg/errors"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"log"
	"net/http"
	"strings"

	"github.com/Optum/dce/pkg/awsiface"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

type dceCtxKeyType string

// DceCtxKey - Context Key
const DceCtxKey dceCtxKeyType = "dce"

// UserGroupName - Has the string to define Users
const UserGroupName = "User"

// AdminGroupName - Has a string to define Admins
const AdminGroupName = "Admin"

// User - Has the username and their role
type User struct {
	Username string
	Role     string
}

// Authorize returns an error if the user is not authorized to act on the principalID
func (u *User) Authorize(principalID string) error {
	var err error
	if u.Role != AdminGroupName && principalID != u.Username {
		err = errors.NewUnathorizedError(fmt.Sprintf("User [%s] with role: [%s] attempted to act on a lease for [%s], but was not authorized",
			u.Username, u.Role, principalID))
	}
	return err
}

// UserDetailer - used for mocking tests
//go:generate mockery -name UserDetailer
type UserDetailer interface {
	GetUser(reqCtx *events.APIGatewayProxyRequestContext) *User
}

// UserDetails - Gets User information
type UserDetails struct {
	CognitoUserPoolID        string `env:"COGNITO_USER_POOL_ID" defaultEnv:"DefaultCognitoUserPoolId"`
	RolesAttributesAdminName string `env:"COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME" defaultEnv:"DefaultCognitoAdminName"`
	CognitoClient            awsiface.CognitoIdentityProviderAPI
}

// GetUser - Gets the username and role out of an http request object
// Assumes that the request is via a Lambda event.
// Uses cognito metadata from the request to determine the user info.
// If the request is not authenticated with cognito,
// returns a generic admin user: User{ Username: "", Role: "Admin" }
func (u *UserDetails) GetUser(reqCtx *events.APIGatewayProxyRequestContext) *User {
	if reqCtx.Identity.CognitoIdentityPoolID == "" {
		// No cognito authentication means the user is considered an admin
		return &User{
			Role: AdminGroupName,
		}
	}

	congitoSubID := strings.Split(reqCtx.Identity.CognitoAuthenticationProvider, ":CognitoSignIn:")[1]

	filter := fmt.Sprintf("sub = \"%s\"", congitoSubID)
	users, err := u.CognitoClient.ListUsers(&cognitoidentityprovider.ListUsersInput{
		Filter:     aws.String(filter),
		UserPoolId: aws.String(u.CognitoUserPoolID),
	})
	if err != nil {
		log.Printf("Error listing users from Cognito: %s", err)
		return &User{}
	}
	if len(users.Users) != 1 {
		log.Printf("Did not get the current user.  Found %d instead of 1.", len(users.Users))
		return &User{}
	}

	user := &User{
		Role:     UserGroupName,
		Username: *users.Users[0].Username,
	}

	for _, attribute := range users.Users[0].Attributes {
		if *attribute.Name == "custom:roles" {
			if u.isUserInAdminFromList(*attribute.Value) {
				user.Role = AdminGroupName
				return user
			}
		}
	}

	isAdmin, err := u.isUserInAdminGroup(user.Username)
	if err != nil {
		log.Printf("Got an error when quering groups for user: %s", err)
		return user
	}
	if isAdmin {
		user.Role = AdminGroupName
		return user
	}

	return user
}

func (u *UserDetails) isUserInAdminGroup(username string) (bool, error) {

	groups, err := u.CognitoClient.AdminListGroupsForUser(&cognitoidentityprovider.AdminListGroupsForUserInput{
		Username:   aws.String(username),
		UserPoolId: aws.String(u.CognitoUserPoolID),
	})
	if err != nil {
		log.Printf("Was not abile to query a users for its groups: %s", err)
		return false, fmt.Errorf("Was not abile to query a users for its groups: %s", err)
	}
	for _, group := range groups.Groups {
		if *group.GroupName == "Admins" {
			return true, nil
		}
	}
	return false, nil
}

func (u *UserDetails) isUserInAdminFromList(groups string) bool {

	for _, group := range strings.Split(groups, ",") {
		if strings.TrimSpace(group) == u.RolesAttributesAdminName {
			return true
		}
	}
	return false
}

type UserDetailsMiddleware struct {
	GorillaMuxAdapter *gorillamux.GorillaMuxAdapter
	UserDetailer UserDetailer
}

func (u *UserDetailsMiddleware) Middleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCtx, err := u.GorillaMuxAdapter.GetAPIGatewayContext(r)
		if err != nil {
			log.Printf("Failed to parse context object from request: %s", err)
			WriteAPIErrorResponse(w,
				errors.NewInternalServer("Internal server error", err),
			)
			return
		}

		user := u.UserDetailer.GetUser(&reqCtx)
		ctx := context.WithValue(r.Context(), User{}, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}