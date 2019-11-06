package api

import (
	"fmt"
	"log"
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

// UserDetailer - used for mocking tests
type UserDetailer interface {
	GetUser(event *events.APIGatewayProxyRequest) *User
}

// UserDetails - Gets User information
type UserDetails struct {
	CognitoUserPoolID        string
	RolesAttributesAdminName string
	CognitoClient            awsiface.CognitoIdentityProviderAPI
}

// GetUser - Gets the username and role out of an event
func (u *UserDetails) GetUser(event *events.APIGatewayProxyRequest) *User {

	if event.RequestContext.Identity.CognitoIdentityPoolID == "" {
		// No cognito authentication means the user is considered an admin
		return &User{
			Role: AdminGroupName,
		}
	}

	congitoSubID := strings.Split(event.RequestContext.Identity.CognitoAuthenticationProvider, ":CognitoSignIn:")[1]

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
